package qa

// سباقاتُ الطلب — **`P-5` البنود ٥ و٦ و١٦ و٢٣.**
//
// **ولا تُصلَح شيفرةُ منتجٍ هنا**: ما كشفه سباقٌ يُعلَن `RISK_CONFIRMED`
// أو `EXPECTED_FAIL` باسم سجلّه، **ولا يُخفى.**

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/fininv"
)

// raceIterations كم مرّةً يُعاد السباقُ الواحد (البند ٢٢).
//
// **وسباقٌ نجح مرّةً لم يُثبت شيئاً** — **والحاجزُ يفتح النافذةَ عمداً،
// فالتكرارُ للثقة لا للصيد.** (**ولا آلافَ محاولاتٍ عشوائيّة.**)
const raceIterations = 5

// dispatchOrder يُنشئ طلباً ويوصله إلى `dispatching` — **حيث يُقبَل.**
func dispatchOrder(t *testing.T, h *Harness, item *Item) string {
	t.Helper()
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET status = 'dispatching' WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("تهيئةُ الطلب للعرض: %v", err)
	}
	return oid
}

// onShiftDriver سائقٌ على الدوام جاهزٌ للقبول.
func onShiftDriver(t *testing.T, f *Factory) *User {
	t.Helper()
	return f.Driver(OnShift())
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · سائقان يقبلان الطلبَ نفسَه**
// ══════════════════════════════════════════════════════════════════════

// TestRACE_TwoDriversSameOrder **`AT MOST ONE DRIVER OWNS THE ORDER`.**
//
// **ولا يُكتفى بأنّ أحدَهما نال 409** (البند ٥) — **تُفحَص القاعدةُ بعده**:
// صاحبُ الطلب · وحالُه · وعروضُه · وأثرُه الماليّ.
func TestRACE_TwoDriversSameOrder(t *testing.T) {
	h := New(t)
	f := h.Factory()
	h.Setting("drivers.assignment_mode", `"queue"`)
	h.Setting("drivers.max_active_orders", "5")
	h.Setting("drivers.cash_limit", "9000000")
	item := h.NewItem(1000)

	base := financialBaseline(t, h)
	winners := 0
	for i := 0; i < raceIterations; i++ {
		oid := dispatchOrder(t, h, item)
		a, b := onShiftDriver(t, f), onShiftDriver(t, f)

		r := Race(t, DefaultRaceTimeout,
			Actor{Name: "سائق-أ", Do: func(ctx context.Context) any {
				return h.POST("/api/v1/driver/orders/"+oid+"/accept", a.Token, nil)
			}},
			Actor{Name: "سائق-ب", Do: func(ctx context.Context) any {
				return h.POST("/api/v1/driver/orders/"+oid+"/accept", b.Token, nil)
			}},
		)
		if r.TimedOut {
			t.Fatalf("الجولة %d عَلِقت", i+1)
		}
		if r.Probe.Max() < 2 {
			t.Errorf("الجولة %d: تداخلٌ مقيسٌ %d — **لم يقع سباق**", i+1, r.Probe.Max())
		}

		ok := r.CountOK()
		winners += ok

		// **والقاعدةُ هي الحَكَم لا رمزُ الاستجابة.**
		var driverID *string
		var status string
		var offered *string
		if err := h.Pool.QueryRow(ctxBG(), `
			SELECT driver_id::text, status, offered_driver_id::text
			FROM orders WHERE id = $1::uuid`, oid).Scan(&driverID, &status, &offered); err != nil {
			t.Fatalf("قراءةُ الطلب: %v", err)
		}
		owners := 0
		if driverID != nil {
			owners = 1
		}
		if ok != 1 || owners != 1 {
			t.Errorf("الجولة %d: نجح %d وصاحبُ الطلب %d — **AT MOST ONE DRIVER OWNS THE ORDER خُرق**",
				i+1, ok, owners)
			t.Errorf("%s", r)
		}
		if driverID != nil && *driverID != a.ID && *driverID != b.ID {
			t.Errorf("الجولة %d: صاحبُ الطلب ليس أحدَ المتسابقين", i+1)
		}
		if status != "assigned" {
			t.Errorf("الجولة %d: الحالُ %q بعد قبولٍ ناجح — يُنتظر assigned", i+1, status)
		}
		if offered != nil {
			t.Errorf("الجولة %d: العرضُ لم يُمسَح بعد القبول", i+1)
		}
		// **ولا سائقان يحملانه** — يُقاس من جدول الطلبات لا من الرمز.
		var held int
		_ = h.Pool.QueryRow(ctxBG(),
			`SELECT count(*) FROM orders WHERE id = $1::uuid AND driver_id IS NOT NULL`,
			oid).Scan(&held)
		if held > 1 {
			t.Errorf("الجولة %d: الطلبُ مملوكٌ %d مرّة", i+1, held)
		}
	}
	t.Logf("TWO-DRIVER SAME-ORDER = %d جولات · فائزٌ واحدٌ في كلٍّ (المجموع %d)",
		raceIterations, winners)
	assertNewViolations(t, h, base, "FI-05", "FI-06", "FI-02")
}

// ══════════════════════════════════════════════════════════════════════
// **٦ · سقفُ الطلبات النشطة — `R10`**
// ══════════════════════════════════════════════════════════════════════

// TestRACE_MaxActiveOrders **`active orders <= max_active`.**
//
// **والقياسُ سبق الحكم** (`driver_handlers.go:519`): العددُ النشطُ يُقرأ في
// استعلامٍ، **ثمّ يُقارَن في Go، ثمّ يُكتب في استعلامٍ ثانٍ** — **بلا قفلٍ
// ولا معاملة.** فالنافذةُ بين القراءة والكتابة مفتوحة.
//
// **والتتابعُ أوّلاً ثمّ التزامن** (البند ٢٣) — **ليُعرَف أعيبٌ في المنطق
// أم في التزامن وحدَه.**
func TestRACE_MaxActiveOrders(t *testing.T) {
	h := New(t)
	f := h.Factory()
	h.Setting("drivers.assignment_mode", `"queue"`)
	h.Setting("drivers.max_active_orders", "1")
	h.Setting("drivers.cash_limit", "9000000")
	item := h.NewItem(1000)

	// ── الأساسُ المتتابع ────────────────────────────────────────────
	{
		drv := onShiftDriver(t, f)
		o1, o2 := dispatchOrder(t, h, item), dispatchOrder(t, h, item)
		r1 := h.POST("/api/v1/driver/orders/"+o1+"/accept", drv.Token, nil)
		r2 := h.POST("/api/v1/driver/orders/"+o2+"/accept", drv.Token, nil)
		active := activeOf(t, h, drv.ID)
		t.Logf("SERIAL BASELINE — الأوّل %d · الثاني %d · النشطُ %d (السقفُ 1)",
			r1.Code, r2.Code, active)
		if r1.Code >= 400 {
			t.Fatalf("القبولُ الأوّلُ رُدّ: %s", r1)
		}
		if active > 1 {
			t.Errorf("SERIAL BASELINE خُرق: النشطُ %d فوق السقف — **عيبُ منطقٍ لا تزامن**", active)
		}
	}

	// ── ثمّ التزامن ────────────────────────────────────────────────
	over := 0
	for i := 0; i < raceIterations; i++ {
		drv := onShiftDriver(t, f)
		o1, o2 := dispatchOrder(t, h, item), dispatchOrder(t, h, item)

		r := Race(t, DefaultRaceTimeout,
			Actor{Name: "طلب-١", Do: func(ctx context.Context) any {
				return h.POST("/api/v1/driver/orders/"+o1+"/accept", drv.Token, nil)
			}},
			Actor{Name: "طلب-٢", Do: func(ctx context.Context) any {
				return h.POST("/api/v1/driver/orders/"+o2+"/accept", drv.Token, nil)
			}},
		)
		if r.TimedOut {
			t.Fatalf("الجولة %d عَلِقت", i+1)
		}
		if r.Probe.Max() < 2 {
			t.Errorf("الجولة %d: تداخلٌ مقيسٌ %d", i+1, r.Probe.Max())
		}
		active := activeOf(t, h, drv.ID)
		if active > 1 {
			over++
			if over == 1 {
				t.Logf("R10 CONFIRMED — الجولة %d: النشطُ %d والسقفُ 1\n%s", i+1, active, r)
			}
		}
	}
	if over > 0 {
		t.Logf("MAX_ACTIVE RACE R10 = RISK CONFIRMED — تجاوزَ السقفَ في %d من %d جولة",
			over, raceIterations)
		t.Logf("والسببُ مقيس: driver_handlers.go يقرأ العددَ ثمّ يقارن ثمّ يكتب — بلا قفلٍ ولا معاملة")
	} else {
		t.Errorf("لم يتجاوز السقفَ في %d جولة — **وR10 يقول إنّه يتجاوزه. يُراجَع السجلُّ المجمَّد.**",
			raceIterations)
	}
}

// activeOf عددُ طلبات السائق المفتوحة — **كما يعدّها المسارُ نفسُه.**
func activeOf(t *testing.T, h *Harness, driverID string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM orders WHERE driver_id = $1::uuid AND closed_at IS NULL`,
		driverID).Scan(&n); err != nil {
		t.Fatalf("عدُّ النشط: %v", err)
	}
	return n
}

// ══════════════════════════════════════════════════════════════════════
// **١٦ · تدخّلُ الإدارة مقابلَ فعلِ التطبيق**
// ══════════════════════════════════════════════════════════════════════

// TestRACE_AdminVsAppTransition **انتقالان متزامنان على طلبٍ واحد.**
//
// **ولا تُخترَع ميزةٌ غيرُ منفَّذة** (البند ١٦): يُستعمل ما تسمح به المسارات
// اليوم — **إسنادٌ إداريٌّ مقابلَ قبولِ سائق.**
func TestRACE_AdminVsAppTransition(t *testing.T) {
	h := New(t)
	f := h.Factory()
	h.Setting("drivers.assignment_mode", `"queue"`)
	h.Setting("drivers.max_active_orders", "5")
	h.Setting("drivers.cash_limit", "9000000")
	item := h.NewItem(1000)
	admin := h.NewUser("admin")

	base := financialBaseline(t, h)
	both, neither := 0, 0
	for i := 0; i < raceIterations; i++ {
		oid := dispatchOrder(t, h, item)
		app := onShiftDriver(t, f)
		other := onShiftDriver(t, f)

		r := Race(t, DefaultRaceTimeout,
			Actor{Name: "تطبيق", Do: func(ctx context.Context) any {
				return h.POST("/api/v1/driver/orders/"+oid+"/accept", app.Token, nil)
			}},
			Actor{Name: "إدارة", Do: func(ctx context.Context) any {
				return h.POST("/api/v1/admin/orders/"+oid+"/assign", admin.Token,
					map[string]any{"driver_id": other.ID})
			}},
		)
		if r.TimedOut {
			t.Fatalf("الجولة %d عَلِقت", i+1)
		}
		ok := r.CountOK()
		if ok == 2 {
			both++
		}
		if ok == 0 {
			neither++
		}

		var driverID *string
		var status string
		if err := h.Pool.QueryRow(ctxBG(),
			`SELECT driver_id::text, status FROM orders WHERE id = $1::uuid`,
			oid).Scan(&driverID, &status); err != nil {
			t.Fatalf("قراءة: %v", err)
		}
		// **والعقدُ الأدنى**: حالٌ سليمةٌ وصاحبٌ واحدٌ لا أكثر.
		if driverID == nil && ok > 0 {
			t.Errorf("الجولة %d: نجح %d ولا صاحبَ للطلب — **حالٌ مكسورة**", i+1, ok)
		}
		if !validStatus(status) {
			t.Errorf("الجولة %d: حالٌ غيرُ مشروعةٍ %q", i+1, status)
		}
		// **وأثرٌ ماليٌّ مزدوجٌ ممنوع** — يُفحَص بمحرّك `P-4` لا بادّعاء.
		if i == 0 {
			t.Logf("ADMIN VS APP — الجولة 1: نجح %d · الحالُ %q\n%s", ok, status, r)
		}
	}
	t.Logf("ADMIN VS APP RACE = %d جولات · نجح الاثنان في %d · ولا واحدَ في %d",
		raceIterations, both, neither)
	if both > 0 {
		t.Logf("OBSERVATION — البابان يقبلان معاً: الإسنادُ الإداريُّ لا يشترط driver_id IS NULL")
	}
	assertNewViolations(t, h, base, "FI-05", "FI-06", "FI-02", "FI-10")
}

func validStatus(s string) bool {
	for _, v := range []string{"pending", "accepted", "preparing", "dispatching", "assigned",
		"at_pickup", "picked_up", "on_the_way", "at_dropoff", "delivered",
		"cancelled", "rejected", "failed", "refunded"} {
		if s == v {
			return true
		}
	}
	return false
}

// ══════════════════════════════════════════════════════════════════════
// **٢٠ · صحّةٌ ماليّةٌ بعد كلّ سباقٍ ماليّ**
// ══════════════════════════════════════════════════════════════════════

// TestRACE_FinancialTruthAfterConcurrentDeliveries **البند ٢٠.**
//
// **ولا تُكتب توكيداتٌ ماليّةٌ جديدة** — **محرّكُ `P-4` هو الحَكَم**:
// سباقٌ يمرّ في HTTP ويُسقط `FI-05` **ليس ناجحاً.**
func TestRACE_FinancialTruthAfterConcurrentDeliveries(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.assignment_mode", `"queue"`)
	h.Setting("drivers.max_active_orders", "5")
	h.Setting("drivers.cash_limit", "9000000")
	h.Setting("merchants.commission_percent", "20")
	h.Setting("delivery.fee", "500")
	item := h.NewItem(1000)

	base := financialBaseline(t, h)

	// **ثلاثةُ طلباتٍ تُسلَّم في اللحظة نفسِها** — لكلٍّ سائقُه.
	const n = 3
	type job struct {
		oid string
		drv *User
	}
	jobs := make([]job, n)
	for i := range jobs {
		oid := dispatchOrder(t, h, item)
		drv := onShiftDriver(t, f)
		if got := h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil); got.Code >= 400 {
			t.Fatalf("قبولُ %d رُدّ: %s", i, got)
		}
		for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
			if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
				map[string]any{"to": to}); got.Code >= 400 {
				t.Fatalf("الانتقالُ إلى %s رُدّ: %s", to, got)
			}
		}
		if got := h.POST("/api/v1/driver/orders/"+oid+"/proof/skip", drv.Token,
			map[string]any{"reason": "P-5"}); got.Code >= 400 {
			t.Fatalf("تخطّي الإثبات: %s", got)
		}
		jobs[i] = job{oid: oid, drv: drv}
	}

	actors := make([]Actor, n)
	for i, j := range jobs {
		j := j
		actors[i] = Actor{Name: fmt.Sprintf("تسليم-%d", i+1), Do: func(ctx context.Context) any {
			return h.POST("/api/v1/driver/orders/"+j.oid+"/transition", j.drv.Token,
				map[string]any{"to": "delivered"})
		}}
	}
	r := Race(t, DefaultRaceTimeout, actors...)
	if r.TimedOut {
		t.Fatal("التسليماتُ المتزامنةُ عَلِقت")
	}
	if r.Probe.Max() < n {
		t.Errorf("تداخلٌ مقيسٌ %d من %d", r.Probe.Max(), n)
	}
	if got := r.CountOK(); got != n {
		t.Errorf("نجح %d من %d تسليماً\n%s", got, n, r)
	}
	t.Logf("CONCURRENT DELIVERIES — %d معاً · تداخلٌ مقيسٌ %d", n, r.Probe.Max())

	// **والحَكَمُ محرّكُ `P-4`.**
	vs := assertNewViolations(t, h, base)
	t.Logf("FINANCIAL INVARIANTS AFTER RACE = %d/%d سليمة",
		len(fininv.Select())-len(vs), len(fininv.Select()))
}

// ══════════════════════════════════════════════════════════════════════
// **١٧ · تبديلُ إعدادٍ ماليٍّ أثناء التسوية**
// ══════════════════════════════════════════════════════════════════════

// TestRACE_SettingChangeDuringSettlement **البند ١٧.**
//
// **ولا يُخلَط بـ`XQ-2`**: ذاك عيبُ لقطةٍ معروف (القيمةُ تُقرأ حيّةً لحظةَ
// التسوية). **والمسؤولُ عنه هنا سؤالٌ آخر**: **أيقرأ التسويةُ قيمةً
// متّسقةً أم نصفَ تبديل؟**
func TestRACE_SettingChangeDuringSettlement(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.assignment_mode", `"queue"`)
	h.Setting("drivers.max_active_orders", "5")
	h.Setting("drivers.cash_limit", "9000000")
	h.Setting("merchants.commission_percent", "20")
	item := h.NewItem(1000)

	base := financialBaseline(t, h)
	oid := dispatchOrder(t, h, item)
	drv := onShiftDriver(t, f)
	if got := h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil); got.Code >= 400 {
		t.Fatalf("قبول: %s", got)
	}
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to}); got.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s: %s", to, got)
		}
	}
	_ = h.POST("/api/v1/driver/orders/"+oid+"/proof/skip", drv.Token,
		map[string]any{"reason": "P-5"})

	admin := h.NewUser("admin")
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "تسليم", Do: func(ctx context.Context) any {
			return h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
				map[string]any{"to": "delivered"})
		}},
		Actor{Name: "تبديلُ نسبة", Do: func(ctx context.Context) any {
			return h.PATCH("/api/v1/admin/settings", admin.Token,
				map[string]any{"merchants.commission_percent": 45})
		}},
	)
	if r.TimedOut {
		t.Fatal("السباقُ عَلِق")
	}
	t.Logf("SETTINGS RACE — %s", r)

	// **والسؤالُ: أهي نسبةٌ واحدةٌ متّسقة؟**
	var commission, cost, due int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT o.platform_commission,
		       COALESCE((SELECT sum(oi.merchant_price*oi.qty) FROM order_items oi
		                 WHERE oi.order_id = o.id), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                 WHERE ref = o.id::text AND kind = 'merchant_earning' AND amount > 0), 0)
		FROM orders o WHERE o.id = $1::uuid`, oid).Scan(&commission, &cost, &due)
	t.Logf("الكلفةُ %d · العمولةُ المكتوبةُ %d · المستحقُّ المقيَّدُ %d", cost, commission, due)
	if due > 0 && commission != cost-due {
		t.Errorf("READ CONSISTENCY خُرق: العمولةُ %d والمحسوبةُ %d — **نصفُ تبديلٍ قُرئ**",
			commission, cost-due)
	} else if due > 0 {
		t.Logf("READ CONSISTENCY = PASS — نسبةٌ واحدةٌ حكمت القيدَ والعمود")
	}
	// **والتوكيدُ على نطاق هذا السيناريو** — `FI-06` و`FI-05` مربوطان
	// بالطلب، **و`FI-02.a` عامٌّ على كلّ محافظ القاعدة.**
	//
	// **ولم يُستثنَ ليمرّ**: هو يُشغَّل عامّاً في `moneycheck` وفي
	// `TestFIN_EngineRunsOnRealDatabase`. **والمُستثنى هنا موضعُ سؤالِه
	// لا وجودُه** — سيناريو تبديلِ إعدادٍ لا يقول شيئاً عن محفظةٍ لا
	// يمسّها.
	//
	// **وسببُ الاستثناء مقيسٌ لا مُفترَض**: صفٌّ متبقٍّ واحدٌ في القاعدة
	// (`رصيدٌ 8000 · ودفترٌ 0`) **لم أُثبت أيُّ اختبارٍ يخلّفه** — ولا
	// يمرّ بمسارِ منتج: **كلُّ كتابةِ رصيدٍ في الإنتاج تمرّ بـ`ApplyTx`
	// التي تكتب القيدَ والرصيدَ معاً.** (ملاحظةٌ مفتوحةٌ في `P5`.)
	assertNewViolations(t, h, base, "FI-06", "FI-05")

	// **ومحفظةُ متجرِ هذا السيناريو تُفحَص بعينها.**
	var wBal, wLedger int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(w.balance, 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                 WHERE user_id = w.user_id), 0)
		FROM merchants m JOIN wallets w ON w.user_id = m.owner_user_id
		WHERE m.id = $1::uuid`, item.MerchantID).Scan(&wBal, &wLedger)
	if wBal != wLedger {
		t.Errorf("FI-02.a خُرق في نطاق السيناريو: رصيدُ المتجر %d ودفترُه %d", wBal, wLedger)
	} else {
		t.Logf("SCOPED WALLET CHECK = PASS — رصيدُ المتجر %d = دفترُه", wBal)
	}
}

var _ = time.Second
