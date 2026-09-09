// سقفُ النقد — **`D7`**.
//
// # ما يقيسه هذا الملفّ
//
// **السقفُ سقفُ تعرّضٍ لا سقفُ ما في الجيب.** **و`driver_cash_boxes.held`
// لا يرتفع إلّا عند التحصيل** (`CollectTx` لحظةَ التسليم) — **فطلبٌ
// أُسنِد ولم يُسلَّم بعدُ لا يُرى في الرقم.**
//
// **فيُقاس ثلاثةُ أشياء**:
//
//	١ طلبٌ واحدٌ نقدُه يفوق السقفَ وحدَه
//	٢ وطلباتٌ صغيرةٌ يجمعها سائقٌ واحدٌ فتفوقه
//	٣ وطلبان متزامنان يقرآن الرقمَ نفسَه
package qa

import (
	"fmt"
	"sync"
	"testing"
)

// cashFixture سائقٌ وسقفٌ وزبون.
type cashFixture struct {
	Driver *User
	Admin  *User
	Cust   *User
	Limit  int64
}

func newCashFixture(t *testing.T, h *Harness, limit int64) cashFixture {
	t.Helper()
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.cash_limit", fmt.Sprint(limit))
	h.Setting("drivers.max_active_orders", "50")
	h.Setting("drivers.assignment_mode", `"queue"`)
	return cashFixture{
		Driver: f.Driver(OnShift()),
		Admin:  h.NewUser("admin"),
		Cust:   h.Customer(),
		Limit:  limit,
	}
}

// cashOrder **طلبٌ نقديٌّ جاهزٌ للإسناد** — ويردّ معرّفَه ونقدَه.
func cashOrder(t *testing.T, h *Harness, cust *User, unit int64, qty int) (string, int64) {
	t.Helper()
	item := h.NewItem(unit)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, qty))
	if made.Code >= 400 {
		t.Fatalf("تجهيزُ طلبٍ نقديّ: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET status = 'dispatching' WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("تهيئةُ الحال: %v", err)
	}
	var due int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT cash_due FROM orders WHERE id = $1::uuid`, oid).Scan(&due); err != nil {
		t.Fatalf("قراءةُ النقد: %v", err)
	}
	if due <= 0 {
		t.Fatalf("**طلبٌ بلا نقد** — ولا يُقاس سقفُ النقد على صفر")
	}
	return oid, due
}

// exposureOf **ما على السائق فعلاً** — المحصَّلُ والمُسنَدُ لم يُحصَّل.
func exposureOf(t *testing.T, h *Harness, driverID string) (held, inflight int64) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT held FROM driver_cash_boxes WHERE driver_id = $1::uuid), 0),
		       COALESCE((SELECT sum(cash_due) FROM orders
		                  WHERE driver_id = $1::uuid AND closed_at IS NULL), 0)`,
		driverID).Scan(&held, &inflight); err != nil {
		t.Fatalf("قراءةُ التعرّض: %v", err)
	}
	return held, inflight
}

// ══════════════════════════════════════════════════════════════════════
// **`D7-1` · طلبٌ واحدٌ يفوق السقفَ وحدَه**
// ══════════════════════════════════════════════════════════════════════

func TestD7_OversizedCashOrderIsRefused(t *testing.T) {
	h := New(t)
	fx := newCashFixture(t, h, 100_000)
	oid, due := cashOrder(t, h, fx.Cust, 90_000, 3)

	held, inflight := exposureOf(t, h, fx.Driver.ID)
	got := h.POST("/api/v1/admin/orders/"+oid+"/assign", fx.Admin.Token,
		map[string]any{"driver_id": fx.Driver.ID})
	t.Logf("D7-1 — سقفٌ=%d · محصَّلٌ=%d · مُسنَدٌ=%d · نقدُ الطلب=%d ⇒ %d %s",
		fx.Limit, held, inflight, due, got.Code, got.Err())

	if got.Code < 400 {
		t.Errorf("**قُبل طلبٌ نقدُه %d لسائقٍ تعرّضُه %d والسقفُ %d** — "+
			"**والسقفُ سقفُ تعرّضٍ لا سقفُ ما في الجيب.** (`D7`)",
			due, held+inflight, fx.Limit)
	}

	// **ورفضٌ لا يترك أثراً** — **والفحصُ والكتابةُ في معاملةٍ واحدة**:
	// **فلا سائقَ أُسنِد ولا تعرّضَ وهميٌّ يبقى يمنع ما بعده.**
	var assigned *string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT driver_id::text FROM orders WHERE id = $1::uuid`, oid).
		Scan(&assigned); err != nil {
		t.Fatalf("قراءةُ الإسناد: %v", err)
	}
	afterHeld, afterInflight := exposureOf(t, h, fx.Driver.ID)
	t.Logf("بعد الرفض — سائقُ الطلب=%v · تعرّضُ السائق=%d",
		assigned, afterHeld+afterInflight)
	if assigned != nil {
		t.Errorf("**رُدَّ الإسنادُ وبقي السائقُ مكتوباً** — %v", *assigned)
	}
	if afterHeld+afterInflight != 0 {
		t.Errorf("**تعرّضٌ وهميٌّ %d بعد رفض** — **يمنع ما بعده بلا سبب.**",
			afterHeld+afterInflight)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D7-2` و`D7-3` · والصغارُ يجتمعن**
// ══════════════════════════════════════════════════════════════════════
//
// **و`held` لا يرتفع إلّا بالتحصيل** — **فثلاثةُ طلباتٍ أُسنِدت ولم
// تُسلَّم تُقرأ صفراً**، ويقبل السائقُ رابعاً وخامساً.

func TestD7_CumulativeExposureIsCounted(t *testing.T) {
	h := New(t)
	fx := newCashFixture(t, h, 100_000)

	accepted, refused := 0, 0
	for i := 0; i < 3; i++ {
		oid, due := cashOrder(t, h, fx.Cust, 40_000, 1)
		got := h.POST("/api/v1/admin/orders/"+oid+"/assign", fx.Admin.Token,
			map[string]any{"driver_id": fx.Driver.ID})
		held, inflight := exposureOf(t, h, fx.Driver.ID)
		t.Logf("  الطلبُ %d — نقدُه %d ⇒ %d · وبعده: محصَّلٌ=%d مُسنَدٌ=%d",
			i+1, due, got.Code, held, inflight)
		if got.Code < 400 {
			accepted++
		} else {
			refused++
		}
	}

	held, inflight := exposureOf(t, h, fx.Driver.ID)
	total := held + inflight
	t.Logf("D7-2/3 — قُبل %d · رُدَّ %d · التعرّضُ النهائيُّ %d والسقفُ %d",
		accepted, refused, total, fx.Limit)

	if total > fx.Limit {
		t.Errorf("**تعرّضُ السائق %d فوق السقف %d** — "+
			"**والمُسنَدُ الذي لم يُحصَّل تعرّضٌ قائم.** (`D7`)", total, fx.Limit)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D7-5` · وغيرُ النقديّ لا يُمنَع**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يُصلَح سقفٌ بمنع ما لا نقدَ فيه** — **وحارسٌ يمنع الجائزَ
// يُطفأ في أوّل شكوى.**

func TestD7_WalletOrderIsNotBlocked(t *testing.T) {
	h := New(t)
	f := h.Factory()
	fx := newCashFixture(t, h, 100_000)

	// **ويُملأ جيبُه حتّى يبلغ السقفَ** — ثمّ يُسنَد طلبٌ بلا نقد.
	rich := h.Customer()
	f.Credit(rich.ID, 500_000, "topup")
	item := h.NewItem(20_000)
	body := orderBody(item, 1)
	body["payment_method"] = "wallet"
	paid := h.POSTKey("/api/v1/orders", rich.Token, uniq("k"), body)
	if paid.Code >= 400 {
		t.Fatalf("طلبُ المحفظة: %s", paid)
	}
	oid, _ := paid.JSON()["id"].(string)

	var due int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT cash_due FROM orders WHERE id = $1::uuid`, oid).Scan(&due)
	if due != 0 {
		t.Skipf("الطلبُ لم يُدفَع كاملاً من المحفظة (نقدُه %d)", due)
	}
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET status = 'dispatching' WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}

	// **والسائقُ عند سقفه** — يُملأ المحصَّلُ مباشرةً.
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO driver_cash_boxes (driver_id, held) VALUES ($1::uuid, $2)
		ON CONFLICT (driver_id) DO UPDATE SET held = EXCLUDED.held`,
		fx.Driver.ID, fx.Limit); err != nil {
		t.Fatalf("ملءُ الجيب: %v", err)
	}

	got := h.POST("/api/v1/admin/orders/"+oid+"/assign", fx.Admin.Token,
		map[string]any{"driver_id": fx.Driver.ID})
	t.Logf("D7-5 — طلبٌ بلا نقدٍ لسائقٍ محصَّلُه %d والسقفُ %d ⇒ %d %s",
		fx.Limit, fx.Limit, got.Code, got.Err())
	if got.Code >= 400 {
		t.Errorf("**مُنع طلبٌ لا نقدَ فيه بسقف النقد** — %s", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **والتزامنُ يقرأ الرقمَ نفسَه**
// ══════════════════════════════════════════════════════════════════════
//
// **وفحصٌ ثمّ كتابةٌ بلا قفلٍ يمرّان معاً** — **واثنان كلٌّ منهما
// جائزٌ وحدَه يفوقان السقفَ مجتمعين.**

func TestD7_ConcurrentAssignmentsCannotOversubscribe(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	const limit = 100_000
	h.Setting("drivers.cash_limit", fmt.Sprint(limit))
	h.Setting("drivers.max_active_orders", "50")
	h.Setting("drivers.assignment_mode", `"queue"`)
	admin := h.NewUser("admin")
	cust := h.Customer()

	// **وجولةٌ واحدةٌ لا تُثبت قفلاً**: **نداءان قد لا يتزاحمان**،
	// **فيمرّ الفحصُ ولا قفلَ أصلاً.** **قيس ذلك**: نُزع القفلُ
	// فمرّت الجولةُ الواحدة. **فالتزاحمُ يُطلَب مئةَ مرّة.**
	const rounds = 100
	over, accepted := 0, 0
	for r := 0; r < rounds; r++ {
		drv := f.Driver(OnShift())
		var ids []string
		for i := 0; i < 2; i++ {
			oid, _ := cashOrder(t, h, cust, 60_000, 1)
			ids = append(ids, oid)
		}

		// **والبدءُ معاً** — بوّابةٌ تُفتح للاثنين في لحظة.
		start := make(chan struct{})
		var wg sync.WaitGroup
		codes := make([]int, 2)
		for i := range ids {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				res := h.POST("/api/v1/admin/orders/"+ids[i]+"/assign", admin.Token,
					map[string]any{"driver_id": drv.ID})
				codes[i] = res.Code
			}(i)
		}
		close(start)
		wg.Wait()

		for _, c := range codes {
			if c < 400 {
				accepted++
			}
		}
		held, inflight := exposureOf(t, h, drv.ID)
		if held+inflight > limit {
			over++
			if over <= 3 {
				t.Errorf("**الجولةُ %d — تعرّضٌ %d فوق السقف %d · ردودٌ %v** — "+
					"**قراءةٌ ثمّ كتابةٌ بلا قفلٍ تمرّان معاً.** (`D7`)",
					r+1, held+inflight, limit, codes)
			}
		}
	}
	t.Logf("D7 CONCURRENCY ×%d — تجاوزٌ %d · إسناداتٌ ناجحة %d", rounds, over, accepted)
	if over > 0 {
		t.Errorf("**%d جولةً من %d تجاوزت السقف.** (`D7`)", over, rounds)
	}
	if accepted == 0 {
		t.Errorf("**لم يُقبل إسنادٌ واحد** — **وحارسٌ يمنع الكلَّ لا يُثبت شيئاً.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D7-9` · وكلُّ بابٍ يضع نقداً في يد سائقٍ يخضع للحكم نفسِه**
// ══════════════════════════════════════════════════════════════════════
//
// **وبابان يضعانه**: إسنادُ المكتب، وانتزاعُ السائق من الطابور.
// **ولا يُصلَح أحدُهما ويُترَك أخوه** — **فمن مُنع من بابٍ دخل من
// الثاني.**

func TestD7_DriverAcceptObeysSameCeiling(t *testing.T) {
	h := New(t)
	fx := newCashFixture(t, h, 100_000)

	// **أوّلاً**: طلبٌ نقدُه يفوق السقفَ وحدَه.
	big, due := cashOrder(t, h, fx.Cust, 90_000, 3)
	got := h.POST("/api/v1/driver/orders/"+big+"/accept", fx.Driver.Token, nil)
	t.Logf("D7-9 — انتزاعُ طلبٍ نقدُه %d والسقفُ %d ⇒ %d %s",
		due, fx.Limit, got.Code, got.Err())
	if got.Code < 400 {
		t.Errorf("**انتُزع طلبٌ نقدُه %d والسقفُ %d.** (`D7`)", due, fx.Limit)
	}

	// **وثانياً**: صغارٌ يجتمعن — **وهو ما كان `held` يُخفيه.**
	accepted := 0
	for i := 0; i < 3; i++ {
		oid, unit := cashOrder(t, h, fx.Cust, 40_000, 1)
		res := h.POST("/api/v1/driver/orders/"+oid+"/accept", fx.Driver.Token, nil)
		t.Logf("  انتزاعُ %d — نقدُه %d ⇒ %d", i+1, unit, res.Code)
		if res.Code < 400 {
			accepted++
		}
	}
	held, inflight := exposureOf(t, h, fx.Driver.ID)
	t.Logf("D7-9 — انتُزع %d · التعرّضُ %d والسقفُ %d",
		accepted, held+inflight, fx.Limit)
	if held+inflight > fx.Limit {
		t.Errorf("**تعرّضٌ %d فوق السقف %d من بابِ الانتزاع.** (`D7`)",
			held+inflight, fx.Limit)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D7-7` · والسعةُ تعود** — **ولا يبقى التعرّضُ إلى الأبد**
// ══════════════════════════════════════════════════════════════════════
//
// **وحارسٌ يمنع ولا يُفرِج يشلّ السائقَ بعد ثلاثة طلبات.**

func TestD7_ExposureIsReleasedWhenOrderCloses(t *testing.T) {
	h := New(t)
	fx := newCashFixture(t, h, 100_000)

	oid, due := cashOrder(t, h, fx.Cust, 90_000, 1)
	if got := h.POST("/api/v1/admin/orders/"+oid+"/assign", fx.Admin.Token,
		map[string]any{"driver_id": fx.Driver.ID}); got.Code >= 400 {
		t.Fatalf("الإسنادُ الأوّل: %s", got)
	}
	held, inflight := exposureOf(t, h, fx.Driver.ID)
	t.Logf("بعد الإسناد — محصَّلٌ=%d مُسنَدٌ=%d (نقدُ الطلب %d)", held, inflight, due)

	// **وثانٍ يُمنع** — السعةُ ممتلئة.
	second, _ := cashOrder(t, h, fx.Cust, 90_000, 1)
	blocked := h.POST("/api/v1/admin/orders/"+second+"/assign", fx.Admin.Token,
		map[string]any{"driver_id": fx.Driver.ID})
	t.Logf("والثاني قبل الإغلاق ⇒ %d %s", blocked.Code, blocked.Err())
	if blocked.Code < 400 {
		t.Fatalf("**مرّ الثاني والسعةُ ممتلئة** — ولا يُقاس إفراجٌ بلا حبس")
	}

	// **ثمّ يُغلَق الأوّل** — إلغاءً: **لا نقدَ قُبض فلا تعرّضَ يبقى.**
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET status = 'cancelled', closed_at = now(), driver_id = NULL
		WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("الإغلاق: %v", err)
	}
	held, inflight = exposureOf(t, h, fx.Driver.ID)
	t.Logf("بعد الإغلاق — محصَّلٌ=%d مُسنَدٌ=%d", held, inflight)

	again := h.POST("/api/v1/admin/orders/"+second+"/assign", fx.Admin.Token,
		map[string]any{"driver_id": fx.Driver.ID})
	t.Logf("D7-7 — والثاني بعد الإغلاق ⇒ %d %s", again.Code, again.Err())
	if again.Code >= 400 {
		t.Errorf("**لم تعُد السعةُ بعد إغلاق الطلب** — "+
			"**وحارسٌ يمنع ولا يُفرِج يشلّ السائق.** (`D7`) %s", again)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D7-4` · وعند السقف تماماً — يُقبَل**
// ══════════════════════════════════════════════════════════════════════
//
// **وهو العقدُ القائم** (`held + cash_due <= limit` في بابِ القبول
// ومرشَّح الطابور) — **ولا يُبدَّل عقدٌ في إصلاح عيب.**

func TestD7_AtLimitExactlyIsAllowed(t *testing.T) {
	h := New(t)
	fx := newCashFixture(t, h, 100_000)

	oid, due := cashOrder(t, h, fx.Cust, 50_000, 1)
	// **والسقفُ يُضبَط على نقدِ الطلب بعينه** — فيقع الحدُّ تماماً.
	h.Setting("drivers.cash_limit", fmt.Sprint(due))

	got := h.POST("/api/v1/admin/orders/"+oid+"/assign", fx.Admin.Token,
		map[string]any{"driver_id": fx.Driver.ID})
	t.Logf("D7-4 — نقدُ الطلب %d والسقفُ %d ⇒ %d %s", due, due, got.Code, got.Err())
	if got.Code >= 400 {
		t.Errorf("**رُدَّ عند السقف تماماً** — **والعقدُ القائمُ يُجيزه** "+
			"(`<= limit`): %s", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **والتكرارُ يكشف ما لا تكشفه مرّة**
// ══════════════════════════════════════════════════════════════════════

func TestD7_CeilingUnderRepetition(t *testing.T) {
	if testing.Short() {
		t.Skip("تكرارٌ — لا يُشغَّل في الوضع القصير")
	}
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	const limit = 100_000
	h.Setting("drivers.cash_limit", fmt.Sprint(limit))
	h.Setting("drivers.max_active_orders", "50")
	h.Setting("drivers.assignment_mode", `"queue"`)
	admin := h.NewUser("admin")
	cust := h.Customer()

	assign := func(oid, driverID string) int {
		return h.POST("/api/v1/admin/orders/"+oid+"/assign", admin.Token,
			map[string]any{"driver_id": driverID}).Code
	}

	// ── الكبيرُ وحدَه ×50 ─────────────────────────────────────────
	leaked := 0
	for i := 0; i < 50; i++ {
		drv := f.Driver(OnShift())
		oid, _ := cashOrder(t, h, cust, 90_000, 3)
		if assign(oid, drv.ID) < 400 {
			leaked++
		}
	}
	t.Logf("  الكبيرُ ×50 — مرّ %d", leaked)

	// ── والحدُّ تماماً ×50 ────────────────────────────────────────
	refusedAtLimit := 0
	for i := 0; i < 50; i++ {
		drv := f.Driver(OnShift())
		oid, due := cashOrder(t, h, cust, 50_000, 1)
		h.Setting("drivers.cash_limit", fmt.Sprint(due))
		if assign(oid, drv.ID) >= 400 {
			refusedAtLimit++
		}
		h.Setting("drivers.cash_limit", fmt.Sprint(limit))
	}
	t.Logf("  الحدُّ تماماً ×50 — رُدَّ %d", refusedAtLimit)

	// ── والتراكمُ ×50 ────────────────────────────────────────────
	overCum := 0
	for i := 0; i < 50; i++ {
		drv := f.Driver(OnShift())
		for k := 0; k < 3; k++ {
			oid, _ := cashOrder(t, h, cust, 40_000, 1)
			assign(oid, drv.ID)
		}
		held, inflight := exposureOf(t, h, drv.ID)
		if held+inflight > limit {
			overCum++
		}
	}
	t.Logf("  التراكمُ ×50 — تجاوزٌ %d", overCum)

	// ── وغيرُ النقديّ ×30 ────────────────────────────────────────
	//
	// **ويُقاس بسائقٍ عند سقفه** — **وطلبٌ بلا نقدٍ لا يزيده.**
	falseBlocks := 0
	rich := h.Customer()
	f.Credit(rich.ID, 3_000_000, "topup")
	for i := 0; i < 30; i++ {
		drv := f.Driver(OnShift())
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO driver_cash_boxes (driver_id, held) VALUES ($1::uuid, $2)
			ON CONFLICT (driver_id) DO UPDATE SET held = EXCLUDED.held`,
			drv.ID, int64(limit)); err != nil {
			t.Fatalf("ملءُ الجيب: %v", err)
		}
		item := h.NewItem(20_000)
		body := orderBody(item, 1)
		body["payment_method"] = "wallet"
		paid := h.POSTKey("/api/v1/orders", rich.Token, uniq("k"), body)
		if paid.Code >= 400 {
			t.Fatalf("طلبُ المحفظة %d: %s", i, paid)
		}
		oid, _ := paid.JSON()["id"].(string)
		var due int64
		_ = h.Pool.QueryRow(ctxBG(),
			`SELECT cash_due FROM orders WHERE id = $1::uuid`, oid).Scan(&due)
		if due != 0 {
			continue
		}
		if _, err := h.Pool.Exec(ctxBG(),
			`UPDATE orders SET status = 'dispatching' WHERE id = $1::uuid`, oid); err != nil {
			t.Fatalf("تهيئة: %v", err)
		}
		if assign(oid, drv.ID) >= 400 {
			falseBlocks++
		}
	}
	t.Logf("  غيرُ النقديّ ×30 — مُنع %d", falseBlocks)

	// ── والإفراجُ ×30 ────────────────────────────────────────────
	stuck := 0
	for i := 0; i < 30; i++ {
		drv := f.Driver(OnShift())
		first, _ := cashOrder(t, h, cust, 90_000, 1)
		assign(first, drv.ID)
		if _, err := h.Pool.Exec(ctxBG(), `
			UPDATE orders SET status='cancelled', closed_at=now(), driver_id=NULL
			WHERE id = $1::uuid`, first); err != nil {
			t.Fatalf("الإغلاق: %v", err)
		}
		second, _ := cashOrder(t, h, cust, 90_000, 1)
		if assign(second, drv.ID) >= 400 {
			stuck++
		}
	}
	t.Logf("  الإفراجُ ×30 — لم تعُد السعةُ %d", stuck)

	if leaked > 0 || refusedAtLimit > 0 || overCum > 0 || falseBlocks > 0 || stuck > 0 {
		t.Errorf("**التكرار: كبيرٌ مرّ %d · رُدَّ عند الحدّ %d · تراكمٌ متجاوزٌ %d · "+
			"نقديٌّ كاذبٌ %d · سعةٌ لم تعُد %d.** (`D7`)",
			leaked, refusedAtLimit, overCum, falseBlocks, stuck)
	}
}
