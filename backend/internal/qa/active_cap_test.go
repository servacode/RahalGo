// سقفُ الطلبات النشطة — **`D24`.**
//
// # العقدُ القائم
//
// **`drivers.max_active_orders`** — عددٌ من ١ إلى ٢٠، افتراضُه ١،
// **عامٌّ لا لكلّ سائق.**
//
// **والنشطُ ما بيده ولم يُغلَق**: `driver_id = <السائق> AND closed_at
// IS NULL`. **ويُغلَق بالتسليم وبكلّ حالٍ نهائيّ** — وبالعودة إلى
// الطابور يُنزَع السائقُ نفسُه.
//
// # وما يُقاس
//
// **بابان يضعان طلباً في يد سائق**: **انتزاعُ السائق من الطابور**
// و**إسنادُ المكتب**. **ولا يُقاس أحدُهما ويُترَك أخوه** — **ولا
// يُقاسان منفردَين وحدَهما**: **بابان مختلفان يتسابقان على آخر
// مكانٍ عند السائق نفسِه.**
package qa

import (
	"fmt"
	"sync"
	"testing"
)

// capFixture سائقٌ وسقفٌ ومكتب.
type capFixture struct {
	Driver *User
	Admin  *User
	Cust   *User
	Limit  int64
}

func newCapFixture(t *testing.T, h *Harness, limit int64) capFixture {
	t.Helper()
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.max_active_orders", fmt.Sprint(limit))
	// **وسقفُ المفتوح للزبون يُلغى هنا** — `D4` · دورةُ ٦١.
	//
	// **وهذه الفحوصُ تقيس سقفَ السائق لا سقفَ الزبون** — **تفتح
	// عشراتِ الطلبات لزبونٍ واحدٍ لتُشبعه.** **وكان سقفُ الزبون
	// معطَّلاً بإطفاء واتساب فمرّت**، **فلمّا صار حارساً قائماً
	// بنفسه ردّها.** **وصفرٌ يُلغي الحدَّ بعقد الإعداد** — **ولا
	// يُضعَّف الحارسُ لأجل مِسنَد.**
	h.Setting("orders.max_open_per_customer", "0")
	h.Setting("drivers.cash_limit", "900000000")
	h.Setting("drivers.assignment_mode", `"queue"`)
	return capFixture{
		Driver: f.Driver(OnShift()),
		Admin:  h.NewUser("admin"),
		Cust:   h.Customer(),
		Limit:  limit,
	}
}

// activeOf **ما بيده ولم يُغلَق** — تعريفُ المنتَج نفسُه.
func activeCount(t *testing.T, h *Harness, driverID string) int64 {
	t.Helper()
	var n int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM orders WHERE driver_id = $1::uuid AND closed_at IS NULL`,
		driverID).Scan(&n); err != nil {
		t.Fatalf("عدُّ النشط: %v", err)
	}
	return n
}

// queuedOrder طلبٌ في الطابور — **ولا عرضَ عليه لأحد.**
//
// **والعرضُ يُلغي السقف** بقرار المالك (٢٠٢٦-٠٨-١٣): **ما عُرض عليه
// بعينه يُقبَل ولو بلغ سقفَه.** **فلا يُقاس السقفُ على معروض.**
func queuedOrder(t *testing.T, h *Harness, cust *User, cost int64) string {
	t.Helper()
	item := h.NewItem(cost)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("d24"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("تجهيزُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET status = 'dispatching',
		    offered_driver_id = NULL, offer_expires_at = NULL
		WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	return oid
}

// ══════════════════════════════════════════════════════════════════════
// **١ · إسنادُ المكتب** — أيحترم السقفَ أصلاً؟
// ══════════════════════════════════════════════════════════════════════

func TestD24_ManualAssignmentObeysCap(t *testing.T) {
	h := New(t)
	fx := newCapFixture(t, h, 1)

	codes := []int{}
	for i := 0; i < 3; i++ {
		oid := queuedOrder(t, h, fx.Cust, 1000)
		res := h.POST("/api/v1/admin/orders/"+oid+"/assign", fx.Admin.Token,
			map[string]any{"driver_id": fx.Driver.ID})
		codes = append(codes, res.Code)
	}
	active := activeCount(t, h, fx.Driver.ID)
	t.Logf("D24-إسناد — السقفُ %d · ثلاثُ محاولات ⇒ %v · النشطُ %d",
		fx.Limit, codes, active)

	if active > fx.Limit {
		t.Errorf("**النشطُ %d فوق السقف %d من بابِ الإسناد** — "+
			"**والمكتبُ يضع في يده ما شاء.** (`D24`)", active, fx.Limit)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · وانتزاعُ السائق متزامناً**
// ══════════════════════════════════════════════════════════════════════

func TestD24_ConcurrentAcceptCannotExceedCap(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.max_active_orders", "1")
	h.Setting("orders.max_open_per_customer", "0") // D4 · دورةُ ٦١
	h.Setting("drivers.cash_limit", "900000000")
	h.Setting("drivers.assignment_mode", `"queue"`)
	cust := h.Customer()

	const rounds = 100
	over := 0
	for r := 0; r < rounds; r++ {
		drv := f.Driver(OnShift())
		var ids []string
		for i := 0; i < 2; i++ {
			ids = append(ids, queuedOrder(t, h, cust, 1000))
		}

		start := make(chan struct{})
		var wg sync.WaitGroup
		codes := make([]int, 2)
		for i := range ids {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				codes[i] = h.POST("/api/v1/driver/orders/"+ids[i]+"/accept",
					drv.Token, nil).Code
			}(i)
		}
		close(start)
		wg.Wait()

		if a := activeCount(t, h, drv.ID); a > 1 {
			over++
			if over <= 3 {
				t.Errorf("**الجولةُ %d — النشطُ %d والسقفُ 1 · ردودٌ %v** — "+
					"**قراءةٌ ثمّ مقارنةٌ ثمّ كتابةٌ بلا قفل.** (`D24`)",
					r+1, a, codes)
			}
		}
	}
	t.Logf("D24 CONCURRENT ACCEPT ×%d — تجاوزٌ %d", rounds, over)
	if over > 0 {
		t.Errorf("**%d جولةً من %d تجاوزت السقف.** (`D24`)", over, rounds)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · وبابان مختلفان يتسابقان على المكان نفسِه**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يكفي أن يحرس كلُّ بابٍ نفسَه** — **إن لم يشتركا في سلطةِ
// سعةٍ واحدةٍ مرّا معاً.**

func TestD24_MixedPathRaceCannotExceedCap(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.max_active_orders", "1")
	h.Setting("orders.max_open_per_customer", "0") // D4 · دورةُ ٦١
	h.Setting("drivers.cash_limit", "900000000")
	h.Setting("drivers.assignment_mode", `"queue"`)
	cust := h.Customer()
	admin := h.NewUser("admin")

	const rounds = 100
	over := 0
	for r := 0; r < rounds; r++ {
		drv := f.Driver(OnShift())
		byAccept := queuedOrder(t, h, cust, 1000)
		byAdmin := queuedOrder(t, h, cust, 1000)

		start := make(chan struct{})
		var wg sync.WaitGroup
		codes := make([]int, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			codes[0] = h.POST("/api/v1/driver/orders/"+byAccept+"/accept",
				drv.Token, nil).Code
		}()
		go func() {
			defer wg.Done()
			<-start
			codes[1] = h.POST("/api/v1/admin/orders/"+byAdmin+"/assign",
				admin.Token, map[string]any{"driver_id": drv.ID}).Code
		}()
		close(start)
		wg.Wait()

		if a := activeCount(t, h, drv.ID); a > 1 {
			over++
			if over <= 3 {
				t.Errorf("**الجولةُ %d — النشطُ %d والسقفُ 1 · "+
					"[انتزاعٌ %d · إسنادٌ %d]** — **بابان لا يشتركان في "+
					"سلطةِ سعة.** (`D24`)", r+1, a, codes[0], codes[1])
			}
		}
	}
	t.Logf("D24 MIXED PATH ×%d — تجاوزٌ %d", rounds, over)
	if over > 0 {
		t.Errorf("**%d جولةً من %d تجاوزت السقف بين بابين.** (`D24`)", over, rounds)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٤ · ومصفوفةُ الحدّ** — وما دونه وما فوقه
// ══════════════════════════════════════════════════════════════════════

func TestD24_CapacityMatrix(t *testing.T) {
	h := New(t)
	fx := newCapFixture(t, h, 3)

	assign := func() int {
		oid := queuedOrder(t, h, fx.Cust, 1000)
		return h.POST("/api/v1/admin/orders/"+oid+"/assign", fx.Admin.Token,
			map[string]any{"driver_id": fx.Driver.ID}).Code
	}

	for want, label := range map[int]string{0: "نشطٌ 0", 1: "نشطٌ 1", 2: "نشطٌ 2 — الحدُّ"} {
		got := assign()
		t.Logf("  %s ⇒ %d · النشطُ الآن %d", label, got, activeCount(t, h, fx.Driver.ID))
		if got >= 400 {
			t.Errorf("**رُدَّ إسنادٌ والسقفُ %d والنشطُ %d** — "+
				"**والعقدُ يجيز حتّى السقف.**", fx.Limit, want)
		}
	}

	// **والرابعُ يُردّ** — النشطُ ثلاثةٌ والسقفُ ثلاثة.
	got := assign()
	t.Logf("  نشطٌ 3 — فوق الحدّ ⇒ %d · النشطُ %d", got, activeCount(t, h, fx.Driver.ID))
	if got < 400 {
		t.Errorf("**قُبل رابعٌ والسقفُ 3** — %d", got)
	}
	if a := activeCount(t, h, fx.Driver.ID); a > fx.Limit {
		t.Errorf("**النشطُ %d فوق السقف %d.**", a, fx.Limit)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · والمكانُ يعود** — ولا مكانَ وهميٌّ يبقى
// ══════════════════════════════════════════════════════════════════════

func TestD24_CapacityIsReleasedOnClose(t *testing.T) {
	h := New(t)
	fx := newCapFixture(t, h, 1)

	first := queuedOrder(t, h, fx.Cust, 1000)
	if got := h.POST("/api/v1/admin/orders/"+first+"/assign", fx.Admin.Token,
		map[string]any{"driver_id": fx.Driver.ID}); got.Code >= 400 {
		t.Fatalf("الإسنادُ الأوّل: %s", got)
	}

	second := queuedOrder(t, h, fx.Cust, 1000)
	blocked := h.POST("/api/v1/admin/orders/"+second+"/assign", fx.Admin.Token,
		map[string]any{"driver_id": fx.Driver.ID})
	t.Logf("والثاني قبل الإغلاق ⇒ %d · النشطُ %d",
		blocked.Code, activeCount(t, h, fx.Driver.ID))
	if blocked.Code < 400 {
		t.Fatalf("**مرّ الثاني والسعةُ ممتلئة** — ولا يُقاس إفراجٌ بلا حبس")
	}

	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET status = 'delivered', delivered_at = now(), closed_at = now()
		WHERE id = $1::uuid`, first); err != nil {
		t.Fatalf("الإغلاق: %v", err)
	}
	t.Logf("بعد الإغلاق — النشطُ %d", activeCount(t, h, fx.Driver.ID))

	again := h.POST("/api/v1/admin/orders/"+second+"/assign", fx.Admin.Token,
		map[string]any{"driver_id": fx.Driver.ID})
	t.Logf("D24 — والثاني بعد الإغلاق ⇒ %d", again.Code)
	if again.Code >= 400 {
		t.Errorf("**لم يعُد المكانُ بعد إغلاق الطلب** — "+
			"**وحارسٌ يمنع ولا يُفرِج يشلّ السائق.** %s", again)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٦ · والسائقان لا ينتظر أحدُهما الآخر**
// ══════════════════════════════════════════════════════════════════════
//
// **والقفلُ على السائق لا على النظام** — **وحارسٌ يُسلسِل الإسنادَ
// كلَّه يشلّ المنصّةَ في ساعة الذروة.**

func TestD24_DifferentDriversAreNotSerialized(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.max_active_orders", "1")
	h.Setting("orders.max_open_per_customer", "0") // D4 · دورةُ ٦١
	h.Setting("drivers.cash_limit", "900000000")
	h.Setting("drivers.assignment_mode", `"queue"`)
	cust := h.Customer()
	admin := h.NewUser("admin")

	const pairs = 50
	ok := 0
	for r := 0; r < pairs; r++ {
		a := f.Driver(OnShift())
		b := f.Driver(OnShift())
		oa := queuedOrder(t, h, cust, 1000)
		ob := queuedOrder(t, h, cust, 1000)

		start := make(chan struct{})
		var wg sync.WaitGroup
		codes := make([]int, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			codes[0] = h.POST("/api/v1/admin/orders/"+oa+"/assign", admin.Token,
				map[string]any{"driver_id": a.ID}).Code
		}()
		go func() {
			defer wg.Done()
			<-start
			codes[1] = h.POST("/api/v1/admin/orders/"+ob+"/assign", admin.Token,
				map[string]any{"driver_id": b.ID}).Code
		}()
		close(start)
		wg.Wait()

		if codes[0] < 400 && codes[1] < 400 {
			ok++
		}
	}
	t.Logf("D24 MULTI-DRIVER ×%d — نجح الاثنان في %d", pairs, ok)
	if ok < pairs {
		t.Errorf("**سائقان لهما سعةٌ ولم يمرّ الاثنان إلّا في %d من %d** — "+
			"**والقفلُ على السائق لا على النظام.**", ok, pairs)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٧ · وحارسان يجتمعان** — `D7` و`D24`
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يُكتب إسنادٌ نجح فيه حارسٌ وسقط فيه أخوه** — **والمصفوفةُ
// ثلاثيّة.**

func TestD24_ComposesWithCashCeiling(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.assignment_mode", `"queue"`)
	admin := h.NewUser("admin")
	cust := h.Customer()

	assign := func(oid, driverID string) (int, string) {
		res := h.POST("/api/v1/admin/orders/"+oid+"/assign", admin.Token,
			map[string]any{"driver_id": driverID})
		return res.Code, res.Err()
	}

	// ── ١ ── السعةُ تتّسع والنقدُ لا ──────────────────────────────
	h.Setting("drivers.max_active_orders", "5")
	h.Setting("orders.max_open_per_customer", "0") // D4 · دورةُ ٦١
	h.Setting("drivers.cash_limit", "100000")
	drvA := f.Driver(OnShift())
	bigCash := queuedOrder(t, h, cust, 90_000)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET cash_due = 270000, total = 270000 WHERE id = $1::uuid`,
		bigCash); err != nil {
		t.Fatalf("تهيئةُ النقد: %v", err)
	}
	code, msg := assign(bigCash, drvA.ID)
	t.Logf("  سعةٌ نعم · نقدٌ لا ⇒ %d %s · النشطُ %d",
		code, msg, activeCount(t, h, drvA.ID))
	if code < 400 || msg != "cash_limit_exceeded" {
		t.Errorf("**مرّ إسنادٌ يتجاوز سقفَ النقد والسعةُ تتّسع** — %d %s", code, msg)
	}
	if a := activeCount(t, h, drvA.ID); a != 0 {
		t.Errorf("**كُتب إسنادٌ رغم سقوط حارس النقد** — النشطُ %d", a)
	}

	// ── ٢ ── والنقدُ يتّسع والسعةُ لا ─────────────────────────────
	h.Setting("drivers.max_active_orders", "1")
	h.Setting("orders.max_open_per_customer", "0") // D4 · دورةُ ٦١
	h.Setting("drivers.cash_limit", "900000000")
	drvB := f.Driver(OnShift())
	first := queuedOrder(t, h, cust, 1000)
	if c, _ := assign(first, drvB.ID); c >= 400 {
		t.Fatalf("الإسنادُ الأوّل: %d", c)
	}
	second := queuedOrder(t, h, cust, 1000)
	code, msg = assign(second, drvB.ID)
	t.Logf("  نقدٌ نعم · سعةٌ لا ⇒ %d %s · النشطُ %d",
		code, msg, activeCount(t, h, drvB.ID))
	if code < 400 || msg != "too_many_active_orders" {
		t.Errorf("**مرّ إسنادٌ يتجاوز سقفَ الطلبات والنقدُ يتّسع** — %d %s", code, msg)
	}
	if a := activeCount(t, h, drvB.ID); a != 1 {
		t.Errorf("**النشطُ %d والسقفُ 1.**", a)
	}

	// ── ٣ ── وكلاهما يتّسع ────────────────────────────────────────
	h.Setting("drivers.max_active_orders", "5")
	h.Setting("orders.max_open_per_customer", "0") // D4 · دورةُ ٦١
	drvC := f.Driver(OnShift())
	fine := queuedOrder(t, h, cust, 1000)
	code, msg = assign(fine, drvC.ID)
	t.Logf("  كلاهما نعم ⇒ %d %s · النشطُ %d", code, msg, activeCount(t, h, drvC.ID))
	if code >= 400 {
		t.Errorf("**رُدَّ إسنادٌ والحارسان يتّسعان** — %d %s", code, msg)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٨ · وخفضُ السقف لا يُلغي ما بيده**
// ══════════════════════════════════════════════════════════════════════
//
// **ومن أنقص السقفَ لا يُلغي طلباتٍ قائمة** — **يمنع الجديدَ حتّى
// ينزل العددُ.** **وإلغاءُ ما وقع بتبديلِ إعدادٍ ظلمٌ لثلاثة أطراف.**

func TestD24_LoweringLimitBlocksNewOnly(t *testing.T) {
	h := New(t)
	fx := newCapFixture(t, h, 5)

	for i := 0; i < 3; i++ {
		oid := queuedOrder(t, h, fx.Cust, 1000)
		if got := h.POST("/api/v1/admin/orders/"+oid+"/assign", fx.Admin.Token,
			map[string]any{"driver_id": fx.Driver.ID}); got.Code >= 400 {
			t.Fatalf("الإسنادُ %d: %s", i+1, got)
		}
	}
	before := activeCount(t, h, fx.Driver.ID)

	// **ثمّ يُنقَص السقفُ تحت العدد.**
	h.Setting("drivers.max_active_orders", "1")
	h.Setting("orders.max_open_per_customer", "0") // D4 · دورةُ ٦١
	after := activeCount(t, h, fx.Driver.ID)
	t.Logf("D24 — نشطٌ قبل الخفض %d · بعده %d (السقفُ صار 1)", before, after)
	if after != before {
		t.Errorf("**تبدّل عددُ النشط بخفض الإعداد** — %d ← %d", before, after)
	}

	oid := queuedOrder(t, h, fx.Cust, 1000)
	got := h.POST("/api/v1/admin/orders/"+oid+"/assign", fx.Admin.Token,
		map[string]any{"driver_id": fx.Driver.ID})
	t.Logf("  والجديدُ بعد الخفض ⇒ %d %s", got.Code, got.Err())
	if got.Code < 400 {
		t.Errorf("**قُبل جديدٌ والعددُ %d فوق السقف 1.**", after)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٩ · وإعادةُ الإسناد** — سعةُ الوجهة تُفحَص
// ══════════════════════════════════════════════════════════════════════

func TestD24_ReassignmentChecksTargetCapacity(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.max_active_orders", "1")
	h.Setting("orders.max_open_per_customer", "0") // D4 · دورةُ ٦١
	h.Setting("drivers.cash_limit", "900000000")
	h.Setting("drivers.assignment_mode", `"queue"`)
	admin := h.NewUser("admin")
	cust := h.Customer()

	a := f.Driver(OnShift())
	b := f.Driver(OnShift())

	// **وB عند سقفه.**
	busy := queuedOrder(t, h, cust, 1000)
	if got := h.POST("/api/v1/admin/orders/"+busy+"/assign", admin.Token,
		map[string]any{"driver_id": b.ID}); got.Code >= 400 {
		t.Fatalf("إشغالُ B: %s", got)
	}

	// **وطلبٌ بيد A.**
	moving := queuedOrder(t, h, cust, 1000)
	if got := h.POST("/api/v1/admin/orders/"+moving+"/assign", admin.Token,
		map[string]any{"driver_id": a.ID}); got.Code >= 400 {
		t.Fatalf("إسنادُ A: %s", got)
	}

	// **ثمّ يُنقَل إلى B الممتلئ.**
	got := h.POST("/api/v1/admin/orders/"+moving+"/assign", admin.Token,
		map[string]any{"driver_id": b.ID})
	t.Logf("D24 — نقلٌ إلى سائقٍ عند سقفه ⇒ %d %s · A=%d · B=%d",
		got.Code, got.Err(), activeCount(t, h, a.ID), activeCount(t, h, b.ID))

	if got.Code < 400 {
		t.Errorf("**نُقل طلبٌ إلى سائقٍ عند سقفه** — %d", got.Code)
	}
	// **ولا يُترَك الطلبُ بلا صاحب** — **ورفضٌ يُيتّم طلباً أسوأُ من
	// قبولٍ يتجاوز.**
	var owner *string
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT driver_id::text FROM orders WHERE id = $1::uuid`, moving).Scan(&owner)
	if owner == nil || *owner != a.ID {
		t.Errorf("**سقط النقلُ وضاع صاحبُ الطلب** — %v (والمتوقَّع A=%s)", owner, a.ID)
	}
	if bb := activeCount(t, h, b.ID); bb > 1 {
		t.Errorf("**B صار %d والسقفُ 1.**", bb)
	}
}
