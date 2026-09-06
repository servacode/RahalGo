package qa

// البنودُ ١٣ و١٤ و١٥ و٢٣ — **التوقيتُ والمصروفُ والترتيبُ والحدود.**

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
)

// ══════════════════════════════════════════════════════════════════════
// **١٣ · حالُ الطلب مقابلَ المال**
// ══════════════════════════════════════════════════════════════════════

// TestFIN_MoneyTimingByStatus **البند ١٣ — أثرٌ في مرحلةٍ خاطئةٍ خطأ.**
//
// **والعقدُ مقيسٌ لا مفترَض** (`transitions.go:686` و`:702`):
//
//	قبل الاستلام ← لا مالَ إلّا خصمُ المحفظة عند الإنشاء
//	الاستلام     ← مستحقُّ المتجرِ يُقيَّد هنا لا عند التسليم
//	التسليم      ← النقدُ والأجرُ وعمولةُ المندوب
func TestFIN_MoneyTimingByStatus(t *testing.T) {
	h := New(t)
	treasury(t, h)
	base := financialBaseline(t, h)
	f := h.Factory()
	h.Setting("merchants.commission_percent", "20")
	h.Setting("sales.commission_percent", "10")
	h.Setting("sales.activation_orders", "0")
	h.Setting("pricing.margin_fixed", "1000")
	h.Setting("delivery.fee", "500")

	rep := f.RepAccount()
	m := f.Merchant(OwnedByRep(rep.ID))
	item := h.NewItemFor(m, 4000)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)

	kindsAt := func(stage string) map[string]int64 {
		got := map[string]int64{}
		rows, err := h.Pool.Query(ctxBG(),
			`SELECT kind, sum(amount) FROM wallet_transactions WHERE ref = $1 GROUP BY kind`, oid)
		if err != nil {
			t.Fatalf("قراءة: %v", err)
		}
		defer rows.Close()
		for rows.Next() {
			var k string
			var v int64
			_ = rows.Scan(&k, &v)
			got[k] = v
		}
		t.Logf("%-12s ⇒ %v", stage, got)
		return got
	}

	at := kindsAt("assigned")
	for _, forbidden := range []string{"merchant_earning", "driver_earning", "commission"} {
		if _, ok := at[forbidden]; ok {
			t.Errorf("MONEY TIMING خُرق: %s قُيّد قبل الاستلام", forbidden)
		}
	}

	for _, to := range []string{"at_pickup", "picked_up"} {
		if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to}); got.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s رُدّ: %s", to, got)
		}
	}
	picked := kindsAt("picked_up")
	if _, ok := picked["merchant_earning"]; !ok {
		t.Errorf("MONEY TIMING خُرق: المتجرُ لم يُقبَض له عند الاستلام (العقدُ §٣-١)")
	}
	if _, ok := picked["commission"]; ok {
		t.Errorf("MONEY TIMING خُرق: عمولةُ المندوبِ قبل التسليم — قبل الالتزام الماليّ")
	}
	if _, ok := picked["driver_earning"]; ok {
		t.Errorf("MONEY TIMING خُرق: أجرُ السائقِ قبل التسليم")
	}

	for _, to := range []string{"on_the_way", "at_dropoff"} {
		if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to}); got.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s رُدّ: %s", to, got)
		}
	}
	if skip := h.POST("/api/v1/driver/orders/"+oid+"/proof/skip", drv.Token,
		map[string]any{"reason": "P-4"}); skip.Code >= 400 {
		t.Fatalf("تخطّي الإثبات: %s", skip)
	}
	if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"}); got.Code >= 400 {
		t.Fatalf("التسليم: %s", got)
	}
	done := kindsAt("delivered")
	if done["driver_earning"] != 500 {
		t.Errorf("أجرُ السائقِ عند التسليم %d — يُنتظر 500", done["driver_earning"])
	}
	if done["commission"] == 0 {
		t.Errorf("عمولةُ المندوبِ لم تُقيَّد عند التسليم")
	}
	assertNewViolations(t, h, base, "FI-06", "FI-10", "FI-05", "FI-04", "FI-03")
}

// ══════════════════════════════════════════════════════════════════════
// **١٥ · المصروفُ والخزينة — `D5`**
// ══════════════════════════════════════════════════════════════════════

// TestFIN_ExpenseTreasuryInvariant **البند ١٥.**
//
//	PROVABLE NOW                الحالُ المكسورةُ تُمسَك (FI-04.d)
//	DEFERRED FAILURE-INJECTION  وقوعُها بإسقاط الخطوة الثانية (FI-12.c → P-6)
func TestFIN_ExpenseTreasuryInvariant(t *testing.T) {
	h := New(t)
	treasury(t, h)
	base := financialBaseline(t, h)
	admin := h.NewUser("admin")

	var catID string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO expense_categories (name, sort_order, active)
		VALUES ($1, 1, true) RETURNING id::text`, uniq("بندُ مصروف QA ")).Scan(&catID); err != nil {
		t.Fatalf("بندُ المصروف: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM expense_categories WHERE id = $1::uuid`, catID)
	})

	made := h.POST("/api/v1/admin/expenses", admin.Token,
		map[string]any{"category_id": catID, "amount": 25_000, "note": "P-4"})
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ المصروف رُدّ: %s", made)
	}
	eid, _ := made.JSON()["id"].(string)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = h.Pool.Exec(ctx, `DELETE FROM wallet_transactions WHERE ref = $1`, eid)
		_, _ = h.Pool.Exec(ctx, `DELETE FROM expenses WHERE id = $1::uuid`, eid)
		// **والرصيدُ يُصالَح بعد حذفِ قيوده** — **وإلّا بقيت الخزينةُ تحمل
		// خصمَ مصروفٍ لا قيدَ له، فأسقطت `FI-02.a` في اختبارٍ لاحقٍ بلا
		// ذنبٍ له.** (وقع فعلاً: −23300 في الخزينة عبرت ثلاثةَ اختبارات.)
		_, _ = h.Pool.Exec(ctx, `
			UPDATE wallets SET balance = COALESCE(
				(SELECT sum(amount) FROM wallet_transactions WHERE user_id = wallets.user_id), 0)
			WHERE is_treasury`)
	})

	var posted int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'operating_expense'`, eid).Scan(&posted)
	t.Logf("المصروفُ 25000 · وقيدُ الخزينة = %d", posted)
	if posted != -25_000 {
		t.Errorf("FI-04.d خُرق: المصروفُ 25000 وقيدُه %d", posted)
	}
	assertNewViolations(t, h, base, "FI-01.f", "FI-04.d", "FI-04.e", "FI-12.a", "FI-12.b")

	// **والحالُ المكسورةُ تُصنَع بيدٍ** — إسقاطُ الخطوة الثانية عمداً حقنُ
	// عطلٍ وهو `P-6`. **والمقصودُ هنا: أيمسك الحارسُ الحالَ إن وقعت؟**
	if _, err := h.Pool.Exec(ctxBG(),
		`DELETE FROM wallet_transactions WHERE ref = $1 AND kind = 'operating_expense'`, eid); err != nil {
		t.Fatalf("الإفسادُ المقصود: %v", err)
	}
	vs, err := fininv.Run(ctxBG(), h.Pool, "FI-04.d")
	if err != nil {
		t.Fatalf("محرّك: %v", err)
	}
	if len(vs) == 0 {
		t.Error("D5 — مصروفٌ بلا قيدِ خزينةٍ لم يُمسَك: الحارسُ أعمى عن الحال التي يخلقها العيب")
	} else {
		t.Logf("D5 STATE DETECTION = PROVEN — %s", vs[0])
		t.Logf("D5 ATOMICITY PROOF = DEFERRED TO P-6 (FI-12.c)")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **١٤ · `D2` / `XQ-3` — الوجهُ الماليّ**
// ══════════════════════════════════════════════════════════════════════

// TestFIN_TargetRewardPrecedesCommit **البند ١٤ — الترتيبُ يُثبَت الآن.**
//
// **ولا يُصلَح `D2` هنا.** المطلوبُ إثباتُ الترتيب نفسِه: **المكافأةُ
// تُصرَف قبل أن يُثبَّت التحويلُ الذي استحقّها.** والحالُ الجزئيّةُ
// (مكافأةٌ بلا تحويل) تحتاج إسقاطَ التثبيت عمداً: `FI-03.d` → `P-6`.
func TestFIN_TargetRewardPrecedesCommit(t *testing.T) {
	src, err := os.ReadFile("../server/leads_handlers.go")
	if err != nil {
		t.Fatalf("قراءةُ المصدر: %v", err)
	}
	body := string(src)
	i := strings.Index(body, "s.grantSalesTargetIfAny(ctx, mrch.ID)")
	j := strings.Index(body, "UPDATE merchant_leads SET status = 'converted'")
	if i < 0 || j < 0 {
		t.Skip("المسارُ تبدّل — يُعاد قياسُ D2 قبل الحكم")
	}
	line := func(k int) int { return strings.Count(body[:k], "\n") + 1 }
	t.Logf("grantSalesTargetIfAny : السطر %d", line(i))
	t.Logf("تثبيتُ التحويل        : السطر %d", line(j))
	if i < j {
		t.Logf("EXPECTED_FAIL (D2 · XQ-3) — REWARD BEFORE CONVERSION COMMIT: "+
			"المكافأةُ تُصرَف في السطر %d والتحويلُ يُثبَّت في %d، وهي تُودَع "+
			"في معاملتها الخاصّة فوراً (target.go:245). "+
			"FULL PARTIAL-STATE PROOF = DEFERRED TO P-6 (FI-03.d)", line(i), line(j))
	} else {
		t.Errorf("الترتيبُ صار سليماً — وD2 يقول غيرَ ذلك. يُراجَع السجلُّ المجمَّد.")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٢٣ · حدودُ المعاملات — تُقاس من الاستعمال لا من الأسماء**
// ══════════════════════════════════════════════════════════════════════

// TestFIN_TransactionBoundaries **البند ٢٣.**
//
// **ويُعاد القياسُ في كلّ تشغيل** — **فسجلٌّ يُكتب مرّةً ويشيخ أسوأُ من
// لا سجلّ**، ومن فتح معاملةً في `handleCreateExpense` غداً رأى هذا يسقط.
func TestFIN_TransactionBoundaries(t *testing.T) {
	counts := map[string]int{}
	for _, b := range fininv.TxBoundaries {
		src, err := os.ReadFile("../../" + b.File)
		if err != nil {
			t.Errorf("%s: %v", b.Op, err)
			continue
		}
		body := string(src)
		i := strings.Index(body, b.Sig)
		if i < 0 {
			t.Errorf("%s: لم أجد %q في %s — يُعاد القياس", b.Op, b.Sig, b.File)
			continue
		}
		block := body[i:]
		if k := strings.Index(block, "\n}\n"); k > 0 {
			block = block[:k]
		}
		got := "NON_ATOMIC"
		switch {
		case strings.Contains(block, ".Begin("):
			got = "ATOMIC"
		case strings.Contains(block, "q wallet.Querier"),
			strings.Contains(block, "q Querier"),
			strings.Contains(block, "q dbtx.Querier"),
			strings.Contains(block, "tx dbtx.Querier"),
			// **ومنسّقُ منع التكرار يملك المعاملةَ ويمرّرها** —
			// **فمن ناداه ورث معاملتَه** (`XG-33`، دورةُ إصلاحٍ ٩).
			strings.Contains(block, "WithIdempotentTx"),
			strings.Contains(block, "tx pgx.Tx"):
			got = "INHERITS_TX"
		}
		if got != b.Class {
			t.Errorf("%s: مقيسٌ %s ومسجَّلٌ %s — يُصحَّح السجلُّ لا القياس", b.Op, got, b.Class)
			continue
		}
		counts[b.Class]++
		t.Logf("%-26s %-12s %s", b.Op, b.Class, b.File)
	}
	t.Logf("ATOMIC = %d · PARTIALLY_ATOMIC (INHERITS_TX) = %d · NON_ATOMIC = %d",
		counts["ATOMIC"], counts["INHERITS_TX"], counts["NON_ATOMIC"])
}

// TestFIN_LedgerKindGuardFailsOnNewKind **البند ٧ — يُثبَت بالتجربة.**
//
// **وحارسٌ لم يُرَ ساقطاً ليس حارساً.** فيُضاف نوعٌ إلى قيد القاعدة
// **داخل معاملةٍ تُرجَع** — فتُقرأ منها، ويُثبَت أنّ الحارسَ يمسكه،
// **ولا تُمسّ القاعدةُ خارجَ الاختبار.**
func TestFIN_LedgerKindGuardFailsOnNewKind(t *testing.T) {
	h := New(t)
	tx, err := h.Pool.Begin(ctxBG())
	if err != nil {
		t.Fatalf("معاملة: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	if _, err := tx.Exec(ctxBG(), `
		ALTER TABLE wallet_transactions DROP CONSTRAINT wallet_transactions_kind_check;
		ALTER TABLE wallet_transactions ADD CONSTRAINT wallet_transactions_kind_check
		  CHECK (kind = ANY (ARRAY['topup','order_payment','refund','compensation',
		    'commission','merchant_earning','driver_earning','payout','adjustment',
		    'platform_profit','platform_expense','operating_expense','reward','penalty',
		    'cashback']));`); err != nil {
		t.Fatalf("إضافةُ النوع تجريبيّاً: %v", err)
	}

	schema, err := fininv.SchemaKinds(ctxBG(), tx)
	if err != nil {
		t.Fatalf("قراءةُ القيد: %v", err)
	}
	if len(schema) != 15 {
		t.Fatalf("قُرئ %d نوعاً — يُنتظر 15", len(schema))
	}
	missing, stale := fininv.KindDrift(schema)
	if len(missing) != 1 || missing[0] != "cashback" {
		t.Fatalf("الحارسُ لم يمسك النوعَ الجديد: %v", missing)
	}
	if len(stale) != 0 {
		t.Errorf("شائخٌ لا محلَّ له: %v", stale)
	}
	t.Logf("LEDGER KIND DRIFT GUARD = PROVEN — أُضيف %q إلى القيد فسقط الحارس", missing[0])
	t.Logf("NEW LEDGER KIND WITHOUT FINANCIAL CONTRACT = FAIL")

	if err := tx.Rollback(ctxBG()); err != nil {
		t.Fatalf("إرجاعُ المعاملة: %v", err)
	}
	after, err := fininv.SchemaKinds(ctxBG(), h.Pool)
	if err != nil {
		t.Fatalf("قراءةٌ بعد الإرجاع: %v", err)
	}
	if len(after) != 14 {
		t.Errorf("القاعدةُ لم تعد كما كانت: %d نوعاً", len(after))
	}
	if m, s := fininv.KindDrift(after); len(m) != 0 || len(s) != 0 {
		t.Errorf("بقي انحرافٌ بعد الإرجاع: %v %v", m, s)
	}
}
