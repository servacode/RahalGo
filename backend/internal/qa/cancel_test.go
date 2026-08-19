package qa

// **الطبقةُ السادسة — الإلغاءُ والمحفظةُ والحسم.**
//
// المعرّفات: `CANC-*` · `WALL-*` · `PROMO-*`
// الوسم: `@api @critical @release`
//
// (أولويّةُ المالك ٢٠٢٦-٠٨-١٩، البنود ١٠ و١١: «Cancellation · Wallet /
//
//	payment إن وُجدت».)
//
// # ولماذا الإلغاءُ أخطرُ من الطلب
//
// **الطلبُ يُنشئ ديناً والإلغاءُ يفكّه** — **وفكٌّ ناقصٌ يترك المالَ
// معلّقا**: خُصم من محفظته ثمّ أُلغي فلم يُردّ.
//
// **ونافذةُ التدارك مقصودة**: الزبونُ يُلغي ما لم يُقبَل بعد، **ولا
// يُلغي طلباً في يد سائقٍ يقف على بابه.**

import (
	"fmt"
	"testing"
)

// TestCANC_001_CustomerCancelsPending **يُلغي ما لم يُقبَل.**
func TestCANC_001_CustomerCancelsPending(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("CANC-001 تعذّر التجهيز: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)

	got := h.POST("/api/v1/orders/"+oid+"/cancel", cust.Token,
		map[string]any{"reason": "غيّرتُ رأيي"})
	if got.Code >= 400 {
		t.Fatalf("CANC-001 الإلغاءُ قبل القبول رُدّ: %s", got)
	}
	if st := h.statusOf(oid); st != "cancelled" {
		t.Errorf("CANC-001 الحالُ %q بعد الإلغاء — يُنتظر cancelled", st)
	}
}

// TestCANC_002_DoubleCancel **وإلغاءان لا يفعلان مرّتين.**
//
// (أولويّةُ المالك، البند ١٥: «Double tap Cancel».)
//
// **وإلغاءٌ ثانٍ يردّ ٢٠٠ يجعل الشاشةَ تُظهر نجاحاً مرّتين** — أو أسوأ:
// **يُعيد المالَ مرّتين** إن كان الردُّ يمرّ بالدفتر.
func TestCANC_002_DoubleCancel(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)

	first := h.POST("/api/v1/orders/"+oid+"/cancel", cust.Token,
		map[string]any{"reason": "الأولى"})
	if first.Code >= 400 {
		t.Fatalf("CANC-002 الإلغاءُ الأوّلُ رُدّ: %s", first)
	}
	second := h.POST("/api/v1/orders/"+oid+"/cancel", cust.Token,
		map[string]any{"reason": "الثانية"})
	if second.Code < 400 {
		t.Errorf("CANC-002 إلغاءٌ ثانٍ قُبل — **الشاشةُ تقول نجحَ مرّتين**: %s", second)
	}
	if st := h.statusOf(oid); st != "cancelled" {
		t.Errorf("CANC-002 الحالُ %q بعد إلغاءين", st)
	}
}

// TestCANC_003_ForeignCancel **وطلبُ غيرِه لا يُلغيه.**
//
// **وهذا IDOR في أخطر موضع**: من عرف معرّفَ طلبٍ ألغاه على صاحبه
// **وهو ينتظر طعامَه.**
func TestCANC_003_ForeignCancel(t *testing.T) {
	h := New(t)
	victim, attacker := h.Customer(), h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", victim.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)

	got := h.POST("/api/v1/orders/"+oid+"/cancel", attacker.Token,
		map[string]any{"reason": "تخريب"})
	if got.Code < 400 {
		t.Errorf("CANC-003 حسابٌ أجنبيٌّ ألغى طلبَ غيرِه: %s", got)
	}
	if st := h.statusOf(oid); st == "cancelled" {
		t.Error("CANC-003 **أُلغي الطلبُ بيدِ غريب**")
	}
}

// TestCANC_010_CancelAfterDelivered **والمُسلَّمُ لا يُلغى.**
//
// **وإلغاءُ ما سُلّم يفكّ ديناً قُبض** — والسائقُ أخذ المالَ نقدا.
func TestCANC_010_CancelAfterDelivered(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	if _, err := h.Pool.Exec(t.Context(),
		`UPDATE orders SET status = 'delivered' WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("تعذّرت زراعةُ الحال: %v", err)
	}
	got := h.POST("/api/v1/orders/"+oid+"/cancel", cust.Token,
		map[string]any{"reason": "بعد التسليم"})
	if got.Code < 400 {
		t.Errorf("CANC-010 أُلغي طلبٌ سُلّم — **مالٌ قُبض ودَينٌ فُكّ**: %s", got)
	}
}

// TestWALL_001_BalanceIsOwn **ورصيدُه رصيدُه.**
func TestWALL_001_BalanceIsOwn(t *testing.T) {
	h := New(t)
	a, b := h.Customer(), h.Customer()
	seed(t, h, a.ID, 7500)

	ra := h.GET("/api/v1/my/wallet", a.Token)
	rb := h.GET("/api/v1/my/wallet", b.Token)
	if ra.Code != 200 || rb.Code != 200 {
		t.Fatalf("WALL-001 تعذّرت القراءة: %s · %s", ra, rb)
	}
	if fmt.Sprint(ra.JSON()["balance"]) != "7500" {
		t.Errorf("WALL-001 رصيدٌ خاطئ: %v — يُنتظر 7500", ra.JSON()["balance"])
	}
	if fmt.Sprint(rb.JSON()["balance"]) != "0" {
		t.Errorf("WALL-001 حسابٌ جديدٌ برصيدٍ غيرِ صفر: %v", rb.JSON()["balance"])
	}
}

// TestWALL_010_CannotPayBeyondBalance **ولا يُدفع ما لا يملك.**
//
// **ورصيدٌ يهبط تحت الصفر دَينٌ لم يوافق عليه أحد.**
func TestWALL_010_CannotPayBeyondBalance(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	seed(t, h, cust.ID, 10)
	item := h.NewItem(50_000)

	body := orderBody(item, 5)
	body["payment_method"] = "wallet"
	got := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), body)
	if got.Code < 400 {
		var bal int64
		_ = h.Pool.QueryRow(t.Context(),
			`SELECT balance FROM wallets WHERE user_id = $1::uuid`, cust.ID).Scan(&bal)
		if bal < 0 {
			t.Errorf("WALL-010 **الرصيدُ هبط تحت الصفر**: %d", bal)
		}
	}
}

// TestPROMO_001_InvalidCodeRejected **وكودٌ لا وجودَ له يُردّ.**
//
// **وكودٌ خاطئٌ يُقبل بحسمِ صفرٍ يُقرأ نجاحا** — ثمّ يسأل صاحبُه أين
// حسمُه.
func TestPROMO_001_InvalidCodeRejected(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	got := h.POST("/api/v1/promo/preview", cust.Token, map[string]any{
		"code":  "لا-وجود-له-" + uniq(""),
		"items": []map[string]any{{"menu_item_id": item.ID, "qty": 1}},
	})
	if got.Code < 400 {
		if d, _ := got.JSON()["discount"].(float64); d > 0 {
			t.Errorf("PROMO-001 كودٌ وهميٌّ أعطى حسماً %v", d)
		}
	}
}
