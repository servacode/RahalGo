package qa

// **الطبقةُ التاسعة — إرجاعُ البضاعة وردُّ المال.**
//
// المعرّفات: `RET-*` · الوسم: `@api @critical @release`
//
// # وهذا آخرُ مسارٍ يمسّ المال
//
// **الطلبُ فشل والبضاعةُ في يد السائق** — فإمّا تُعاد إلى المتجر وتُردّ
// أرباحُه، **وإمّا تبقى عنده ويُحاسَب عليها.**
//
// **وإرجاعٌ يقع مرّتين يخصم من المتجر مرّتين** — وهو مالٌ لا يُستدرَك
// إلّا بشكوى.
//
// **ودفترُ المال لا يُمسّ إلّا بقرارٍ صريح** (CLAUDE.md) — **وهذه قاعدةُ
// اختبارٍ لا الإنتاج**، و`testdb` يرفض غيرَها.

import (
	"testing"
)

// failedOrder **طلبٌ بلغ `failed` بسائقٍ معلوم** — الحالُ الوحيدةُ
// التي يُقبل فيها الإرجاع.
func (h *Harness) failedOrder(t *testing.T) (string, *User, *Item) {
	t.Helper()
	cust := h.Customer()
	item := h.NewItem(2000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("RET: تعذّر التجهيز: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	// **وتُزرع الحالُ مباشرةً** — بلوغُها بالطريق الشرعيّ يقيس الطريقَ
	// لا الإرجاع، **وله حزمتُه** (`FAIL-*`).
	if _, err := h.Pool.Exec(t.Context(),
		`UPDATE orders SET status = 'failed' WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("RET: تعذّرت زراعةُ الحال: %v", err)
	}
	return oid, drv, item
}

func (h *Harness) setAcceptsReturns(t *testing.T, merchantID string, on bool) {
	t.Helper()
	if _, err := h.Pool.Exec(t.Context(),
		`UPDATE merchants SET accepts_returns = $2 WHERE id = $1::uuid`, merchantID, on); err != nil {
		t.Fatalf("RET: تعذّر ضبطُ قبولِ الإرجاع: %v", err)
	}
}

// TestRET_001_OnlyFailedIsReturnable **ولا يُرجَع إلّا الفاشل.**
//
// **وطلبٌ سُلّم يُرجَع يخصم من المتجر مالاً قبضه بحقّ.**
func TestRET_001_OnlyFailedIsReturnable(t *testing.T) {
	h := New(t)
	oid, drv, item := h.failedOrder(t)
	h.setAcceptsReturns(t, item.MerchantID, true)

	for _, st := range []string{"delivered", "on_the_way", "pending", "cancelled"} {
		if _, err := h.Pool.Exec(t.Context(),
			`UPDATE orders SET status = $2 WHERE id = $1::uuid`, oid, st); err != nil {
			t.Fatalf("تعذّرت زراعةُ %q: %v", st, err)
		}
		got := h.POST("/api/v1/driver/orders/"+oid+"/return", drv.Token, map[string]any{})
		if got.Code < 400 {
			t.Errorf("RET-001 أُرجع طلبٌ حالُه %q: %s", st, got)
		}
	}
}

// TestRET_002_NeedsMerchantConsent **ومتجرٌ لا يقبل الإرجاع لا يُرجَع
// إليه.**
//
// **وبضاعةٌ تُعاد إلى متجرٍ لا يقبلها تبقى في يد السائق** — ثمّ يُحاسَب
// عليها وقد أُعيدت في الدفتر.
func TestRET_002_NeedsMerchantConsent(t *testing.T) {
	h := New(t)
	oid, drv, item := h.failedOrder(t)
	h.setAcceptsReturns(t, item.MerchantID, false)

	got := h.POST("/api/v1/driver/orders/"+oid+"/return", drv.Token, map[string]any{})
	if got.Code < 400 {
		t.Errorf("RET-002 أُرجع إلى متجرٍ لا يقبل الإرجاع: %s", got)
	}
	if got.Err() != "" && got.Err() != "no_returns" {
		t.Logf("RET-002 رُدّ برمزٍ آخر: %s", got.Err())
	}
}

// TestRET_003_HappyPath **والإرجاعُ يقع حين تجتمع الشروط.**
func TestRET_003_HappyPath(t *testing.T) {
	h := New(t)
	oid, drv, item := h.failedOrder(t)
	h.setAcceptsReturns(t, item.MerchantID, true)

	got := h.POST("/api/v1/driver/orders/"+oid+"/return", drv.Token, map[string]any{})
	if got.Code >= 400 {
		t.Fatalf("RET-003 الإرجاعُ رُدّ رغمَ اجتماع الشروط: %s", got)
	}
	if ok, _ := got.JSON()["returned"].(bool); !ok {
		t.Errorf("RET-003 الردُّ لا يقول إنّه أُرجع: %s", got)
	}
	// **ويُوسَم في القاعدة** — **وردٌّ ٢٠٠ بلا وسمٍ يجعله يُرجَع ثانية.**
	var returned *string
	if err := h.Pool.QueryRow(t.Context(),
		`SELECT returned_at::text FROM orders WHERE id = $1::uuid`, oid).Scan(&returned); err != nil {
		t.Fatalf("RET-003 تعذّرت القراءة: %v", err)
	}
	if returned == nil {
		t.Error("RET-003 **لم يُوسَم `returned_at`** — فيُرجَع مرّةً أخرى")
	}
}

// TestRET_010_NoDoubleReturn **ولا يُرجَع مرّتين.**
//
// **وخصمان من متجرٍ واحدٍ لطلبٍ واحد** — مالٌ لا يُستدرَك إلّا بشكوى.
func TestRET_010_NoDoubleReturn(t *testing.T) {
	h := New(t)
	oid, drv, item := h.failedOrder(t)
	h.setAcceptsReturns(t, item.MerchantID, true)

	first := h.POST("/api/v1/driver/orders/"+oid+"/return", drv.Token, map[string]any{})
	if first.Code >= 400 {
		t.Fatalf("RET-010 الإرجاعُ الأوّلُ رُدّ: %s", first)
	}
	second := h.POST("/api/v1/driver/orders/"+oid+"/return", drv.Token, map[string]any{})
	if second.Code < 400 {
		t.Errorf("RET-010 **أُرجع مرّتين** — وخُصم من المتجر مرّتين: %s", second)
	}
}

// TestRET_020_ForeignDriverCannotReturn **وسائقٌ غيرُ سائقِه لا يُرجعه.**
//
// **ومن أرجع طلبَ غيرِه خصم من متجرٍ لا علاقةَ له به.**
func TestRET_020_ForeignDriverCannotReturn(t *testing.T) {
	h := New(t)
	oid, _, item := h.failedOrder(t)
	h.setAcceptsReturns(t, item.MerchantID, true)
	intruder := h.NewUser("driver")

	got := h.POST("/api/v1/driver/orders/"+oid+"/return", intruder.Token, map[string]any{})
	if got.Code < 400 {
		t.Errorf("RET-020 أرجع سائقٌ أجنبيٌّ طلبَ غيرِه: %s", got)
	}
	var returned *string
	_ = h.Pool.QueryRow(t.Context(),
		`SELECT returned_at::text FROM orders WHERE id = $1::uuid`, oid).Scan(&returned)
	if returned != nil {
		t.Error("RET-020 **وُسم الطلبُ مُرجَعاً بيدِ سائقٍ أجنبيّ**")
	}
}

// TestRET_030_ReversesMerchantEarning **وأرباحُ المتجر تُردّ.**
//
// **وهذا موضعُ المال بعينه**: أُرجعت البضاعةُ فيجب أن يُخصم ما دُفع
// للمتجر. **وإرجاعٌ لا يمسّ الدفترَ يُرجع البضاعةَ ويترك المال.**
func TestRET_030_ReversesMerchantEarning(t *testing.T) {
	h := New(t)
	oid, drv, item := h.failedOrder(t)
	h.setAcceptsReturns(t, item.MerchantID, true)

	// **يُزرع ربحُ متجرٍ في الدفتر** — كما يقيّده إغلاقُ الطلب.
	var owner string
	if err := h.Pool.QueryRow(t.Context(),
		`SELECT owner_user_id::text FROM merchants WHERE id = $1::uuid`,
		item.MerchantID).Scan(&owner); err != nil {
		t.Fatalf("RET-030 تعذّرت قراءةُ صاحب المتجر: %v", err)
	}
	seed(t, h, owner, 10_000)
	if _, err := h.Pool.Exec(t.Context(), `
		INSERT INTO wallet_transactions (user_id, amount, kind, ref, note)
		VALUES ($1::uuid, $2, 'merchant_earning', $3::uuid, 'اختبارُ إرجاع')`,
		owner, 2000, oid); err != nil {
		t.Skipf("RET-030 تعذّر تجهيزُ قيدِ الربح (%v) — يُراجَع مخطّطُ الدفتر", err)
	}

	before := walletOf(t, h, owner)
	got := h.POST("/api/v1/driver/orders/"+oid+"/return", drv.Token, map[string]any{})
	if got.Code >= 400 {
		t.Fatalf("RET-030 الإرجاعُ رُدّ: %s", got)
	}
	after := walletOf(t, h, owner)

	if rev, _ := got.JSON()["reversed"].(float64); rev <= 0 {
		t.Errorf("RET-030 **لم يُردّ شيءٌ من المال**: reversed=%v", got.JSON()["reversed"])
	}
	if after >= before {
		t.Errorf("RET-030 **رصيدُ المتجر لم ينقص**: %d ← %d", before, after)
	}
}

func walletOf(t *testing.T, h *Harness, userID string) int64 {
	t.Helper()
	var bal int64
	if err := h.Pool.QueryRow(t.Context(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1::uuid`, userID).
		Scan(&bal); err != nil {
		return 0
	}
	return bal
}
