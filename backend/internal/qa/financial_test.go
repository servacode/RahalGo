package qa

// اختباراتُ الثوابت الماليّة — `P-4`.
//
// **وكلُّها على PostgreSQL/PostGIS حقيقيّة** (البند ٢٢): المالُ لا يُثبَت
// بمحاكاة. **والمُفسِّراتُ وحدَها تُختبَر بلا قاعدة** — وهي في
// `internal/fininv`.
//
// **ولا تُصلَح شيفرةُ منتجٍ هنا** (البند ٢٦): ما كشفه اختبارٌ يُعلَن
// `EXPECTED_FAIL` باسم فجوته، **ولا يُخفى ولا يُصلَح.**

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/fininv"
)

// ══════════════════════════════════════════════════════════════════════
// **٢٧ · أمانُ القاعدة — يُعاد إثباتُه قبل أيّ كتابةٍ حسّاسة**
// ══════════════════════════════════════════════════════════════════════

// TestFIN_DatabaseSafetyBeforeWrites **لا كتابةَ قبل الهويّة.**
//
// **ولا يُعاد بناءُ `P-3V`** — إثباتُ الأمان يكفي: **هويّةٌ مقروءةٌ من
// القاعدة نفسِها، وحارسُ إنتاجٍ يرفض ما ليس اختباراً.**
func TestFIN_DatabaseSafetyBeforeWrites(t *testing.T) {
	if !Enabled() {
		t.Skip("TEST_DATABASE_URL غير مضبوط")
	}
	raw := os.Getenv("TEST_DATABASE_URL")
	if err := checkTestDatabase(raw); err != nil {
		t.Fatalf("حارسُ الإنتاج رفض قاعدةَ الاختبار: %v", err)
	}
	pool, err := pgxpool.New(context.Background(), raw)
	if err != nil {
		t.Fatalf("اتّصال: %v", err)
	}
	defer pool.Close()

	var db, ver string
	if err := pool.QueryRow(context.Background(),
		`SELECT current_database(), version()`).Scan(&db, &ver); err != nil {
		t.Fatalf("هويّة: %v", err)
	}
	if !strings.HasSuffix(db, "_test") {
		t.Fatalf("قاعدةٌ لا تنتهي بـ_test: %s", db)
	}
	t.Logf("DATABASE = %s", db)
	t.Logf("POSTGRES = %s", strings.SplitN(ver, " on ", 2)[0])
	t.Logf("URL      = %s", redact(raw))
	t.Logf("PRODUCTION DATABASE GUARD = PASS")

	// **والحارسُ يُثبَت بالرفض لا بالقبول** — قبولُ ما هو سليمٌ لا يقول
	// شيئاً عمّا يفعله بما ليس كذلك.
	for _, bad := range []string{
		"postgres://u:p@localhost:5434/rahalgo?sslmode=disable",
		"postgres://u:p@195.201.141.130:5432/rahalgo_test",
	} {
		if err := checkTestDatabase(bad); err == nil {
			t.Errorf("الحارسُ قبل ما كان يجب أن يرفض: %s", bad)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٧ · حارسُ أنواع القيد — من القاعدة الحيّة**
// ══════════════════════════════════════════════════════════════════════

// TestFIN_LedgerKindAllowlistGuard **البند ٧.**
//
//	NEW LEDGER KIND WITHOUT FINANCIAL CONTRACT = FAIL
func TestFIN_LedgerKindAllowlistGuard(t *testing.T) {
	h := New(t)
	schema, err := fininv.SchemaKinds(ctxBG(), h.Pool)
	if err != nil {
		t.Fatalf("قراءةُ القيد: %v", err)
	}
	t.Logf("LEDGER KINDS DISCOVERED = %d — %s", len(schema), strings.Join(schema, ", "))

	missing, stale := fininv.KindDrift(schema)
	if len(missing) > 0 {
		t.Errorf("NEW LEDGER KIND WITHOUT FINANCIAL CONTRACT = FAIL — %v\n"+
			"يُملأ في internal/fininv/kinds.go: من يكتبه · دلالتُه الماليّة · "+
			"ثوابتُه · أيشترط مرجعاً", missing)
	}
	if len(stale) > 0 {
		t.Errorf("عقودٌ لأنواعٍ لم تعد في القاعدة: %v", stale)
	}
	if len(missing) == 0 && len(stale) == 0 {
		t.Logf("LEDGER KINDS WITH CONTRACT = %d/%d", len(schema), len(schema))
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · الدالّةُ الواحدة — فحصُ الحقيقة الماليّة لحالةِ اختبار**
// ══════════════════════════════════════════════════════════════════════

// assertFinancialTruth يشغّل كلَّ ما يُثبَت — **ويطبع الخرقَ بصفوفه.**
func assertFinancialTruth(t *testing.T, h *Harness, ids ...string) []fininv.Violation {
	t.Helper()
	return assertNewViolations(t, h, financialBaseline(t, h, ids...), ids...)
}

// financialBaseline يصوّر ما هو مخروقٌ **قبل** السيناريو.
//
// **وعلّتُه في `fininv/baseline.go`**: قاعدةُ الاختبار مشتركةٌ ومقبرةُ
// سيناريوهات — **بعضُها يُفسد عمداً، وبعضُها ينظّف مستخدماً فيجرّ حذفُه
// قيودَه بالتتالي** فيبقى طلبٌ مسلَّمٌ بلا مستحقِّ متجر.
// **وسقوطُ اختبارٍ بذنبِ غيره أسوأُ من لا اختبار.**
func financialBaseline(t *testing.T, h *Harness, ids ...string) fininv.Baseline {
	t.Helper()
	b, err := fininv.Capture(ctxBG(), h.Pool, ids...)
	if err != nil {
		t.Fatalf("تصويرُ الخروق القائمة تعذّر: %v", err)
	}
	return b
}

// assertNewViolations ما استجدّ بعد الصورة — **وهو وحدَه ذنبُ السيناريو.**
func assertNewViolations(t *testing.T, h *Harness, base fininv.Baseline, ids ...string) []fininv.Violation {
	t.Helper()
	vs, err := fininv.Run(ctxBG(), h.Pool, ids...)
	if err != nil {
		t.Fatalf("محرّكُ الثوابت تعذّر: %v", err)
	}
	fresh := base.New(vs)
	for _, v := range fresh {
		t.Errorf("FINANCIAL INVARIANT VIOLATION\n%s\n  %s", v, strings.ReplaceAll(v.Check.Why, "**", ""))
	}
	return fresh
}

// TestFIN_EngineRunsOnRealDatabase **كلُّ فحصٍ يُثبَت يُنفَّذ فعلاً** —
// **ولا فحصَ يمرّ لأنّ استعلامَه لا يُصرَّف.**
func TestFIN_EngineRunsOnRealDatabase(t *testing.T) {
	h := New(t)
	run := fininv.Select()
	for _, c := range run {
		if _, err := fininv.Run(ctxBG(), h.Pool, c.ID); err != nil {
			t.Errorf("%s استعلامُه لا يُصرَّف: %v", c.ID, err)
		}
	}
	t.Logf("FINANCIAL CHECKS EXECUTED = %d/%d", len(run), len(run))
}

// ══════════════════════════════════════════════════════════════════════
// **٦ · حفظُ اقتصاد الطلب — على طلبٍ مرّ بدورة حياته كاملةً**
// ══════════════════════════════════════════════════════════════════════

// deliverOrder يقود طلباً إلى التسليم بانتقالات المجال الحقيقيّة.
//
// **ولا كتابةَ حالةٍ رأساً** — **وحالٌ كُتبت بـ`UPDATE` لم تمرّ بتسويةٍ،
// فاختبارُ المال عليها يختبر لا شيء.**
func deliverOrder(t *testing.T, h *Harness, oid string, drv *User) {
	t.Helper()
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to})
		if got.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s رُدّ: %s", to, got)
		}
	}
	// **والتسليمُ يشترط شاهداً** — صورةً أو تخطّياً بسبب (delivery_proof.go:40).
	if skip := h.POST("/api/v1/driver/orders/"+oid+"/proof/skip", drv.Token,
		map[string]any{"reason": "P-4 — إثباتُ ثابتٍ ماليّ"}); skip.Code >= 400 {
		t.Fatalf("تخطّي الإثبات رُدّ: %s", skip)
	}
	if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"}); got.Code >= 400 {
		t.Fatalf("التسليمُ رُدّ: %s", got)
	}
}

// treasury يضمن خزينةً — **وثوابتُ الحفظ مشروطةٌ بها** (`FI-02.c`).
func treasury(t *testing.T, h *Harness) string {
	t.Helper()
	return h.Factory().Treasury()
}

// TestFIN_OrderEconomicConservation **`FI-06.a` — قانونُ الحفظ الأكبر.**
func TestFIN_OrderEconomicConservation(t *testing.T) {
	h := New(t)
	treasury(t, h)
	base := financialBaseline(t, h)

	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 2))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	var net, cash, cashDue int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT sum(amount) FROM wallet_transactions WHERE ref = $1), 0),
		       COALESCE((SELECT sum(amount) FROM driver_cash_entries
		                 WHERE ref = $1 AND kind = 'order_collection'), 0),
		       (SELECT cash_due FROM orders WHERE id = $1::uuid)`, oid).
		Scan(&net, &cash, &cashDue); err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	t.Logf("صافي الدفتر = %d · نقدُ الصندوق = %d · نقدُ الطلب = %d", net, cash, cashDue)
	if net != cash {
		t.Errorf("FI-06.a خُرق: صافي الدفتر %d ≠ نقدُ الصندوق %d", net, cash)
	}
	if cash != cashDue {
		t.Errorf("FI-10.b خُرق: المحصَّل %d ≠ المستحقّ %d", cash, cashDue)
	}
	assertNewViolations(t, h, base, "FI-06", "FI-10", "FI-04", "FI-03")
}

// TestFIN_ConservationSurvivesRefund **الاستردادُ لا يكسر الحفظ.**
func TestFIN_ConservationSurvivesRefund(t *testing.T) {
	h := New(t)
	treasury(t, h)
	base := financialBaseline(t, h)

	cust := h.Customer()
	item := h.NewItem(1500)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	admin := h.NewUser("admin")
	got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "P-4 — إثباتُ ثابتٍ ماليّ"})
	if got.Code >= 400 {
		t.Fatalf("الاستردادُ رُدّ: %s", got)
	}

	var net, cash int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT sum(amount) FROM wallet_transactions WHERE ref = $1), 0),
		       COALESCE((SELECT sum(amount) FROM driver_cash_entries
		                 WHERE ref = $1 AND kind = 'order_collection'), 0)`, oid).
		Scan(&net, &cash); err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	t.Logf("بعد الاسترداد: صافي الدفتر = %d · نقدُ الصندوق = %d", net, cash)
	if net != cash {
		t.Errorf("FI-06.a خُرق بعد الاسترداد: %d ≠ %d", net, cash)
	}
	assertNewViolations(t, h, base, "FI-06", "FI-05", "FI-02")
}

// ══════════════════════════════════════════════════════════════════════
// **١٩ · اختبارُ الحارسِ لنفسِه — إفسادٌ مقصودٌ ثمّ تنظيف**
// ══════════════════════════════════════════════════════════════════════

// TestFIN_CorruptionDetectionSelfTest **البند ١٩** — رصيدٌ يخالف دفترَه.
//
// **وهذا اختبارُ الحارسِ لا عيبُ منتج**: إن لم يمسك الفسادَ المصنوعَ بيدٍ،
// **فلا يمسك الواقعَ يومَ يقع.**
func TestFIN_CorruptionDetectionSelfTest(t *testing.T) {
	h := New(t)
	f := h.Factory()
	u := f.NewUserWith("customer")
	f.Credit(u.ID, 50_000, "topup")

	// **ولا يُشترط أن تكون القاعدةُ نظيفةً** — هي مشتركةٌ ومخلَّفاتُ غيرِنا
	// فيها. **والمقيسُ هو الفرق**: ما لم يكن مخروقاً وصار.
	base := financialBaseline(t, h, "FI-02.a")

	f.UnsafeCorruptBalance(u.ID, 999_999)

	all, err := fininv.Run(ctxBG(), h.Pool, "FI-02.a")
	if err != nil {
		t.Fatalf("محرّك: %v", err)
	}
	vs := base.New(all)
	if len(vs) == 0 {
		t.Fatal("FINANCIAL INVARIANT ENGINE لم يمسك رصيداً يخالف دفترَه — الحارسُ أعمى")
	}
	t.Logf("CORRUPTION DETECTION SELF-TEST = PROVEN — %s", vs[0])

	// **والتنظيفُ صريح** — فلا يُترَك فسادٌ يُسقط اختباراً آخرَ فيُظنّ عيباً.
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE wallets SET balance = (SELECT COALESCE(sum(amount),0)
		 FROM wallet_transactions WHERE user_id = $1) WHERE user_id = $1`, u.ID); err != nil {
		t.Fatalf("تنظيف: %v", err)
	}
	if after, _ := fininv.Run(ctxBG(), h.Pool, "FI-02.a"); len(base.New(after)) > 0 {
		t.Errorf("بقي فسادٌ بعد التنظيف: %v", base.New(after)[0])
	}
}

// TestFIN_MissingReferenceSelfTest **البند ٢٠** — قيدٌ يشترط مرجعاً بلا مرجع.
func TestFIN_MissingReferenceSelfTest(t *testing.T) {
	h := New(t)
	f := h.Factory()
	u := f.NewUserWith("customer")

	base := financialBaseline(t, h, "FI-01.d")

	// **صفٌّ اختباريٌّ محضٌ يُكتب هنا لا في شيفرة الإنتاج** (البند ٢٠):
	// `commission` يشترط مرجعَ طلبٍ، وهذا بلا مرجع.
	var id int64
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO wallet_transactions (user_id, amount, kind, ref, note)
		VALUES ($1::uuid, 1, 'commission', '', 'P-4 self-test')
		RETURNING id`, u.ID).Scan(&id); err != nil {
		t.Fatalf("الإفسادُ المقصود: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM wallet_transactions WHERE id = $1`, id)
	})

	all, err := fininv.Run(ctxBG(), h.Pool, "FI-01.d")
	if err != nil {
		t.Fatalf("محرّك: %v", err)
	}
	vs := base.New(all)
	if len(vs) == 0 {
		t.Fatal("لم يُمسَك قيدٌ بلا مرجعٍ وهو يشترطه")
	}
	t.Logf("MISSING REFERENCE SELF-TEST = PROVEN — %s", vs[0])
}

// TestFIN_DuplicateEffectSelfTest **البند ٢١** — أثرٌ ماليٌّ مكرَّر.
//
// **ولا يُسمّى المحرّكُ «قيداً مزدوجاً» بلا دليل** (البند ٢١): النموذجُ
// **رصيدٌ وقيودُه** (`wallet.go:251` تُحدّث الرصيدَ ثمّ تكتب القيد)،
// **لا حسابان لكلّ حركة.** والتكرارُ يُعرَّف بالمرجع والنوع — وهو حتميّ.
func TestFIN_DuplicateEffectSelfTest(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(800)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	base := financialBaseline(t, h, "FI-05.c")

	var id int64
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO wallet_transactions (user_id, amount, kind, ref, note)
		SELECT user_id, amount, kind, ref, 'P-4 duplicate self-test'
		FROM wallet_transactions WHERE ref = $1 AND kind = 'merchant_earning' AND amount > 0 LIMIT 1
		RETURNING id`, oid).Scan(&id); err != nil {
		t.Fatalf("الإفسادُ المقصود: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM wallet_transactions WHERE id = $1`, id)
	})

	all, err := fininv.Run(ctxBG(), h.Pool, "FI-05.c")
	if err != nil {
		t.Fatalf("محرّك: %v", err)
	}
	vs := base.New(all)
	if len(vs) == 0 {
		t.Fatal("أثرٌ ماليٌّ كُرِّر ولم يُمسَك")
	}
	t.Logf("DUPLICATE EFFECT SELF-TEST = PROVEN — %s", vs[0])
}

// ══════════════════════════════════════════════════════════════════════
// **١٢ · تكرارُ الأفعال الماليّة — بالتسلسل**
// ══════════════════════════════════════════════════════════════════════

// TestFIN_RepeatedRefundIsIdempotent **استردادٌ يُطلَب مرّتين.**
func TestFIN_RepeatedRefundIsIdempotent(t *testing.T) {
	h := New(t)
	treasury(t, h)
	base := financialBaseline(t, h)
	cust := h.Customer()
	item := h.NewItem(1200)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	admin := h.NewUser("admin")
	first := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "P-4 — إثباتُ ثابتٍ ماليّ"})
	if first.Code >= 400 {
		t.Fatalf("الاستردادُ الأوّل رُدّ: %s", first)
	}
	second := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "P-4 — إثباتُ ثابتٍ ماليّ"})
	t.Logf("الاستردادُ الثاني: %s", second.String()[:min(60, len(second.String()))])

	var refunds, sum int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*), COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'refund'`, oid).Scan(&refunds, &sum); err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	t.Logf("قيودُ الاسترداد = %d · مجموعُها = %d", refunds, sum)
	if refunds > 1 {
		t.Errorf("FI-05.g خُرق: استُرِدَّ %d مرّةً", refunds)
	}
	assertNewViolations(t, h, base, "FI-05", "FI-06", "FI-02")
}

// TestFIN_RepeatedPayoutDecision **قرارُ سحبٍ يُتَّخذ مرّتين.**
func TestFIN_RepeatedPayoutDecision(t *testing.T) {
	h := New(t)
	treasury(t, h)
	base := financialBaseline(t, h)
	f := h.Factory()
	drv := f.Driver()
	f.Credit(drv.ID, 100_000, "topup")

	made := h.POST("/api/v1/me/payouts", drv.Token, map[string]any{"amount": 100_000})
	if made.Code >= 400 {
		t.Fatalf("طلبُ السحب رُدّ: %s", made)
	}
	pid, _ := made.JSON()["id"].(string)
	if pid == "" {
		var got string
		if err := h.Pool.QueryRow(ctxBG(),
			`SELECT id::text FROM payout_requests WHERE user_id = $1::uuid`, drv.ID).Scan(&got); err != nil {
			t.Fatalf("لم أجد طلبَ السحب: %v", err)
		}
		pid = got
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(), `DELETE FROM payout_requests WHERE id = $1::uuid`, pid)
	})

	admin := h.NewUser("admin")
	body := map[string]any{"status": "paid", "decision": "P-4"}
	first := h.POST("/api/v1/admin/payouts/"+pid+"/decide", admin.Token, body)
	second := h.POST("/api/v1/admin/payouts/"+pid+"/decide", admin.Token, body)
	t.Logf("القرارُ الأوّل %d · الثاني %d", first.Code, second.Code)
	if first.Code >= 400 {
		t.Fatalf("القرارُ الأوّل رُدّ: %s", first)
	}
	if second.Code < 400 {
		t.Errorf("قرارٌ ثانٍ قُبل على طلبٍ مدفوع — %s", second)
	}

	var n, sum int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*), COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'payout'`, pid).Scan(&n, &sum); err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	t.Logf("قيودُ السحب = %d · مجموعُها = %d", n, sum)
	if n != 1 {
		t.Errorf("FI-05.d خُرق: خُصم %d مرّةً", n)
	}
	if sum != -100_000 {
		t.Errorf("FI-11.c خُرق: خُصم %d ويُنتظر -100000", sum)
	}
	assertNewViolations(t, h, base, "FI-11", "FI-05.d", "FI-04.c", "FI-02")
}

// ══════════════════════════════════════════════════════════════════════
// **١٦ · سقفُ النقد — العقدُ المعتمد**
// ══════════════════════════════════════════════════════════════════════

// TestFIN_CashExposureContract **البند ١٦.**
//
// العقدُ: **المحتجَزُ يمنع طلباً نقديّاً جديداً، ولا يمنع طلباً مدفوعاً من
// المحفظة.** — **ويُقاس ما تحسبه الشيفرةُ فعلاً** (`transitions.go:1176`).
func TestFIN_CashExposureContract(t *testing.T) {
	h := New(t)
	f := h.Factory()
	h.Setting("drivers.cash_limit", "100000")

	over := f.Driver(OnShift(), CashHeld(150_000))
	held, err := f.HeldOf(over.ID)
	if err != nil {
		t.Fatalf("قراءةُ المحتجَز: %v", err)
	}
	t.Logf("المحتجَزُ = %d · السقفُ = 100000", held)

	// **١ — طلبٌ نقديٌّ يُمنَع.**
	cust := h.Customer()
	item := h.NewItem(2000)
	cash := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	cid, _ := cash.JSON()["id"].(string)
	admin := h.NewUser("admin")
	got := h.POST("/api/v1/admin/orders/"+cid+"/assign", admin.Token,
		map[string]any{"driver_id": over.ID})
	t.Logf("إسنادُ طلبٍ نقديٍّ لسائقٍ فوق السقف: %d", got.Code)
	if got.Code < 400 {
		t.Errorf("CASH EXPOSURE CONTRACT خُرق: طلبٌ نقديٌّ أُسنِد لسائقٍ محتجَزُه %d فوق السقف", held)
	}

	// **٢ — وطلبٌ من المحفظةِ لا يُمنَع بالمحتجَز وحدَه.**
	rich := h.Customer()
	f.Credit(rich.ID, 500_000, "topup")
	body := orderBody(item, 1)
	body["payment_method"] = "wallet"
	paid := h.POSTKey("/api/v1/orders", rich.Token, uniq("k"), body)
	if paid.Code >= 400 {
		t.Skipf("تعذّر إنشاءُ طلبٍ مدفوعٍ من المحفظة: %s", paid)
	}
	wid, _ := paid.JSON()["id"].(string)
	var due int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT cash_due FROM orders WHERE id = $1::uuid`, wid).Scan(&due)
	if due != 0 {
		t.Skipf("الطلبُ لم يُدفَع كاملاً من المحفظة (نقدُه %d) — والحالُ غيرُ التي نختبر", due)
	}
	assign := h.POST("/api/v1/admin/orders/"+wid+"/assign", admin.Token,
		map[string]any{"driver_id": over.ID})
	t.Logf("إسنادُ طلبٍ من المحفظةِ للسائق نفسِه: %d", assign.Code)
	if assign.Code >= 400 && strings.Contains(assign.Err(), "cash") {
		t.Errorf("CASH EXPOSURE CONTRACT خُرق: طلبٌ بلا نقدٍ مُنع بسبب المحتجَز — %s", assign)
	}

	// **٣ — وما لا يقيسه العقدُ**: المكشوفُ بعد الإسناد.
	//
	// **الشرطُ `held >= limit` لا يجمع نقدَ الطلب الداخل** — فسائقٌ محتجَزُه
	// صفرٌ يقبل طلباً نقدُه ضعفُ السقف. **وهو `D7` بعينه، ولا يُصلَح هنا.**
	fresh := f.Driver(OnShift())
	big := h.NewItem(90_000)
	huge := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(big, 3))
	hid, _ := huge.JSON()["id"].(string)
	var hugeDue int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT cash_due FROM orders WHERE id = $1::uuid`, hid).Scan(&hugeDue)
	res := h.POST("/api/v1/admin/orders/"+hid+"/assign", admin.Token,
		map[string]any{"driver_id": fresh.ID})
	t.Logf("D7 — طلبٌ نقدُه %d لسائقٍ محتجَزُه 0 والسقفُ 100000: %d", hugeDue, res.Code)
	if hugeDue > 100_000 && res.Code < 400 {
		t.Logf("EXPECTED_FAIL (D7) — السقفُ يقيس المحصَّل لا المكشوف: قُبل طلبٌ نقدُه %d", hugeDue)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٩ و١١ · مصفوفةُ المندوب — العمولةُ والاسترداد**
// ══════════════════════════════════════════════════════════════════════

// TestFIN_RepCommissionReversal **`FI-09` — `EXPECTED_FAIL` باسم `XG-10`.**
func TestFIN_RepCommissionReversal(t *testing.T) {
	h := New(t)
	treasury(t, h)
	f := h.Factory()
	h.Setting("sales.commission_percent", "10")
	h.Setting("sales.activation_orders", "0")
	h.Setting("merchants.commission_percent", "20")
	// **والعمولةُ تُحسَب من الهامش** (settleRep:963) — فبلا هامشٍ لا عمولة.
	h.Setting("pricing.margin_fixed", "1500")

	rep := f.RepAccount()
	item := h.NewItemFor(f.Merchant(OwnedByRep(rep.ID)), 5000)
	cust := h.Customer()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 2))
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	var granted int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'commission'`, oid).Scan(&granted)
	t.Logf("عمولةُ المندوبِ المقيَّدة = %d", granted)
	if granted == 0 {
		t.Skip("لم تُقيَّد عمولةٌ — والسيناريو يشترطها (شرطُ التفعيل أو الهامش)")
	}

	admin := h.NewUser("admin")
	got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "P-4 — إثباتُ ثابتٍ ماليّ"})
	if got.Code >= 400 {
		t.Fatalf("الاستردادُ رُدّ: %s", got)
	}

	var after int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'commission'`, oid).Scan(&after)
	t.Logf("وبعد الاسترداد = %d", after)
	// **وقد أُصلحت `XG-10` في دورةِ إصلاحٍ ٢** (٢٠٢٦-٠٩-٠٦):
	// **العكسُ يقع، وما عجز عنه الرصيدُ يصير التزاماً.**
	//
	// **فانقلب هذا الرصدُ**: كان يسجّل غيابَ العكس، **وصار يحرس
	// وقوعَه.**
	if after != 0 {
		t.Errorf("XG-10 — **العمولةُ بقيت %d بعد الاسترداد**: العكسُ لم يقع. "+
			"والعقدُ (`RQ-5` · ٢٠٢٦-٠٩-٠٥): ORDER REVENUE REVERSED → "+
			"RELATED REP COMMISSION REVERSED", after)
	} else {
		t.Logf("XG-10 CLOSED — **العكسُ وقع والصافي صفر.**")
	}
}

// TestFIN_RefundNotConditionedOnRepBalance **البند ١٧ — عقدٌ حرِج.**
//
//	CUSTOMER REFUND ENTITLEMENT IS NOT CONDITIONED ON REP CURRENT WALLET BALANCE
func TestFIN_RefundNotConditionedOnRepBalance(t *testing.T) {
	h := New(t)
	treasury(t, h)
	f := h.Factory()
	h.Setting("sales.commission_percent", "10")
	h.Setting("sales.activation_orders", "0")
	h.Setting("merchants.commission_percent", "20")
	// **والعمولةُ تُحسَب من الهامش** (settleRep:963) — فبلا هامشٍ لا عمولة.
	h.Setting("pricing.margin_fixed", "1500")

	rep := f.RepAccount()
	item := h.NewItemFor(f.Merchant(OwnedByRep(rep.ID)), 5000)
	cust := h.Customer()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 2))
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	// **المندوبُ يسحب كلَّ ما عنده** — فرصيدُه صفرٌ حين يُطلَب الاسترداد.
	var bal int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1::uuid`, rep.ID).Scan(&bal)
	if bal > 0 {
		f.Credit(rep.ID, -bal, "payout")
	}
	var now int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1::uuid`, rep.ID).Scan(&now)
	t.Logf("رصيدُ المندوبِ قبل الاسترداد = %d (كان %d)", now, bal)

	admin := h.NewUser("admin")
	got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "P-4 — إثباتُ ثابتٍ ماليّ"})
	t.Logf("الاسترداد: %d", got.Code)
	if got.Code >= 400 {
		t.Errorf("REFUND DEPENDS ON DOWNSTREAM BALANCE — حقُّ الزبونِ سقط "+
			"لأنّ رصيدَ المندوبِ صفر: %s (XG-11)", got)
		return
	}

	var refunded int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'refund'`, oid).Scan(&refunded)
	t.Logf("المستردُّ للزبون = %d", refunded)
	t.Logf("PASS — **ونجاحٌ بالعَرَض لا بالالتزام**: لا عكسَ لقيد المندوب أصلاً " +
		"(XG-10)، **فلم يُسأل رصيدُه.** ويومَ يُنفَّذ العكسُ يصير XG-11 حيّاً.")
}

// TestFIN_MerchantWithdrewThenRefund **الوجهُ الحيُّ لآليّةِ `XG-11`.**
//
// **والمتجرُ يُعكَس فعلاً** (`reverseCommissions:1085`) — **فإن كان قد سحب،
// أسقط `CHECK (balance >= 0)` المعاملةَ كلَّها وضاع حقُّ الزبون.**
func TestFIN_MerchantWithdrewThenRefund(t *testing.T) {
	h := New(t)
	treasury(t, h)
	f := h.Factory()
	m := f.Merchant()
	item := h.NewItemFor(m, 4000)
	cust := h.Customer()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	var owner string
	var bal int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT m.owner_user_id::text, COALESCE(w.balance, 0)
		FROM merchants m LEFT JOIN wallets w ON w.user_id = m.owner_user_id
		WHERE m.id = $1::uuid`, m.ID).Scan(&owner, &bal); err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	t.Logf("رصيدُ صاحبِ المتجر بعد التسليم = %d", bal)
	if bal <= 0 {
		t.Skip("لم يُقيَّد مستحقٌّ — والسيناريو يشترطه")
	}
	f.Credit(owner, -bal, "payout")

	admin := h.NewUser("admin")
	got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "P-4 — إثباتُ ثابتٍ ماليّ"})
	t.Logf("الاستردادُ بعد سحبِ المتجرِ مستحقَّه: %d", got.Code)

	var refunded int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'refund'`, oid).Scan(&refunded)
	if got.Code >= 400 || refunded == 0 {
		t.Logf("EXPECTED_FAIL (XG-11) — REFUND CONDITIONED ON DOWNSTREAM BALANCE: "+
			"سقط الاستردادُ (%d) لأنّ صاحبَ المتجرِ سحب مستحقَّه. المستردُّ = %d",
			got.Code, refunded)
	} else {
		t.Logf("الاستردادُ نجح والمستردُّ = %d — **والقيدُ سمح بالسالب أو عُوّض**", refunded)
	}
	// **ولا يُفحَص `FI-02.b` هنا**: الحالُ المقصودةُ قد تترك رصيداً سالباً
	// **وهو الكشفُ نفسُه، لا خللٌ في المِسنَد.**
}

// ══════════════════════════════════════════════════════════════════════
// **٨ و١٠ · مصدرُ العمولة واللقطةُ الاقتصاديّة**
// ══════════════════════════════════════════════════════════════════════

// TestFIN_CommissionSourceMatrix **البند ٨ — أربعُ حالات.**
func TestFIN_CommissionSourceMatrix(t *testing.T) {
	// **خزينةٌ للقاعدة قبل أن تبدأ الحالات** — واحدةٌ تكفي الأربع.
	treasury(t, New(t))

	// **والهامشُ يُضبَط بالإعداد لا بسعرِ الصنف** — قِيس (`service.go:608`):
	// `unit_price = SalePrice(merchant_price, ...)` **يُشتقّ من
	// `pricing.margin_fixed`**، **وسعرُ الصنف في القائمة لا يدخل الحساب.**
	// (وهذا اكتُشف هنا: مصفوفةٌ بُنيت على سعرِ الصنف أعطت هامشاً صفراً في
	// الحالات الأربع.)
	cases := []struct {
		name                      string
		merchantPct, repP, margin int
		cost                      int64
	}{
		{"A — عمولةٌ فقط بلا هامش", 20, 10, 0, 5000},
		{"B — هامشٌ فقط بلا عمولة", 0, 10, 1000, 5000},
		{"C — الاثنتان معاً", 20, 10, 1000, 5000},
		{"D — لا هذه ولا تلك", 0, 10, 0, 5000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := New(t)
			treasury(t, h)
			base := financialBaseline(t, h)
			f := h.Factory()
			h.Setting("merchants.commission_percent", itoa(c.merchantPct))
			h.Setting("sales.commission_percent", itoa(c.repP))
			h.Setting("sales.activation_orders", "0")
			h.Setting("pricing.margin_fixed", itoa(c.margin))

			rep := f.RepAccount()
			m := f.Merchant(OwnedByRep(rep.ID))
			item := h.NewItemFor(m, c.cost)
			made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
			if made.Code >= 400 {
				t.Fatalf("إنشاء: %s", made)
			}
			oid, _ := made.JSON()["id"].(string)
			drv := h.driverOf(oid)
			deliverOrder(t, h, oid, drv)

			var platform, repCom, margin int64
			_ = h.Pool.QueryRow(ctxBG(), `
				SELECT o.platform_commission,
				       COALESCE((SELECT sum(amount) FROM wallet_transactions
				                 WHERE ref = o.id::text AND kind = 'commission'), 0),
				       COALESCE((SELECT sum((oi.unit_price - oi.merchant_price) * oi.qty)
				                 FROM order_items oi WHERE oi.order_id = o.id), 0)
				FROM orders o WHERE o.id = $1::uuid`, oid).Scan(&platform, &repCom, &margin)
			t.Logf("عمولةُ المنصّة = %d · الهامش = %d · عمولةُ المندوب = %d",
				platform, margin, repCom)

			// ══════════════════════════════════════════════════════
			// **والصيغةُ صارت بالوضع المعتمد** — `XG-14` · `RQ-6`
			// ══════════════════════════════════════════════════════
			//
			// **وكانت هنا بوّابةٌ مثبَّتة**: «يخرج إن كانت عمولةُ
			// المنصّة صفراً — فالهامشُ وحدَه لا يُعطي المندوبَ
			// شيئاً». **وذاك ما وثّقه هذا الفحصُ عيباً ومرّ ما دام
			// قائماً.**
			//
			// **والافتراضُ `pricing_margin`** — فالقاعدةُ الهامشُ
			// وحدَه، **بلا بوّابةِ عمولة.**
			want := margin * int64(c.repP) / 100
			if repCom != want {
				t.Errorf("عمولةُ المندوب %d — والوضعُ الافتراضيُّ يعطي %d", repCom, want)
			}
			assertNewViolations(t, h, base, "FI-06", "FI-05", "FI-04")
		})
	}
}

// TestFIN_SnapshotVsLiveEconomics **البند ١٠ — ما هو ملقوطٌ وما يُقرأ حيّاً.**
func TestFIN_SnapshotVsLiveEconomics(t *testing.T) {
	h := New(t)
	treasury(t, h)
	f := h.Factory()
	h.Setting("merchants.commission_percent", "10")
	h.Setting("sales.commission_percent", "10")
	h.Setting("sales.activation_orders", "0")
	h.Setting("pricing.margin_fixed", "1000")
	h.Setting("delivery.fee", "700")

	rep := f.RepAccount()
	m := f.Merchant(OwnedByRep(rep.ID))
	item := h.NewItemFor(m, 5000)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)

	// **ما لُقط لحظةَ الإنشاء** — يُقرأ من القاعدة قبل التبديل وبعده.
	var unit, mp, fee int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT oi.unit_price, oi.merchant_price, o.delivery_fee
		FROM orders o JOIN order_items oi ON oi.order_id = o.id
		WHERE o.id = $1::uuid LIMIT 1`, oid).Scan(&unit, &mp, &fee); err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	t.Logf("ALREADY SNAPSHOTTED — unit_price=%d · merchant_price=%d · delivery_fee=%d", unit, mp, fee)

	// **ثمّ تتبدّل الإعداداتُ قبل التسليم.**
	h.Setting("merchants.commission_percent", "40")
	h.Setting("sales.commission_percent", "50")
	h.Setting("delivery.fee", "99999")

	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	var unit2, mp2, fee2, platform, repCom, driverPaid int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT oi.unit_price, oi.merchant_price, o.delivery_fee, o.platform_commission,
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                 WHERE ref = o.id::text AND kind = 'commission'), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                 WHERE ref = o.id::text AND kind = 'driver_earning'), 0)
		FROM orders o JOIN order_items oi ON oi.order_id = o.id
		WHERE o.id = $1::uuid LIMIT 1`, oid).Scan(&unit2, &mp2, &fee2, &platform, &repCom, &driverPaid)

	if unit != unit2 || mp != mp2 || fee != fee2 {
		t.Errorf("قيمةٌ ملقوطةٌ تبدّلت: %d/%d/%d ⇒ %d/%d/%d", unit, mp, fee, unit2, mp2, fee2)
	}
	if driverPaid != fee {
		t.Errorf("أجرُ السائقِ %d وأجرةُ الطلبِ الملقوطةُ %d — قُرئ الإعدادُ حيّاً", driverPaid, fee)
	}
	t.Logf("SNAPSHOT HELD — أجرُ السائقِ %d = الأجرةُ الملقوطةُ %d (والإعدادُ صار 99999)",
		driverPaid, fee)

	// ══════════════════════════════════════════════════════════════
	// **والعمولةُ صارت ملقوطةً** — `XQ-2` · دورةُ إصلاحٍ ٣٢
	// ══════════════════════════════════════════════════════════════
	//
	// **كان هذا الفحصُ يوثّق القراءةَ الحيّةَ عيباً** ويكتب
	// `EXPECTED_FAIL` ما دامت قائمة، **ويطلب مراجعتَه يومَ تُلتقَط.**
	// **وقد التُقطت** — فصار يقيس العقدَ لا العيب.
	margin := (unit - mp) * 1
	live := mp * 40 / 100
	atCreate := mp * 10 / 100
	t.Logf("SNAPSHOT — عمولةُ المنصّةِ المقيَّدةُ = %d · بنسبةِ الإنشاء(10%%) = %d · بالنسبةِ الحيّة(40%%) = %d",
		platform, atCreate, live)
	if platform != atCreate {
		t.Errorf("**عمولةُ المنصّة %d — ولقطةُ الإنشاء توجب %d** · "+
			"**وإعدادٌ لاحقٌ لا يبدّل اقتصادَ طلبٍ قائم.** (`XQ-2`)", platform, atCreate)
	}
	t.Logf("SNAPSHOT — عمولةُ المندوبِ = %d (الهامشُ %d · والنسبةُ الملقوطةُ لا الحيّة)",
		repCom, margin)
}

func itoa(n int) string { return fmt.Sprint(n) }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
