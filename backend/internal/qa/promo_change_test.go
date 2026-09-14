package qa

// ══════════════════════════════════════════════════════════════════════
// **الخصمُ يسقط — ويُقال باسمه** (`PR`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// **ووعدٌ بخصمٍ سقط أسوأُ من لا خصم** — **ومن قرأ مجموعاً ثمّ دُفع
// غيرُه ظنّ أنّه خُدع.**
//
// # ولا انتظارَ لساعةِ حائط
//
// **والصلاحيّةُ تُقاس بساعة الخادم** (`time.Now()` في `validatePromo`)
// — **وأدواتُها في القاعدة**: `active` و`expires_at` و`used_count`.
// **فيُطفأ الكودُ أو يُؤرَّخ في الماضي، ويُقاس في الحال** — **ولا فحصٌ
// ينام دقيقةً ليرى انتهاءً.**
//
// # وساعةُ الجهاز لا سلطانَ لها
//
// **ولا يُرسَل وقتٌ من الهاتف أصلاً** — **والحكمُ يقع في الخادم على
// صفٍّ مقفولٍ بـ`FOR UPDATE`.** **فتقديمُ ساعةِ الهاتف لا يُحيي كوداً
// ميّتاً ولا يقتل حيّاً.**

import (
	"net/http"
	"testing"
	"time"
)

// promoFx كودُ خصمٍ للفحص — يُمحى بعده.
type promoFx struct {
	ID   string
	Code string
}

// newPromo كودٌ ثابتُ القيمة، فعّالٌ بلا انتهاء.
func newPromo(t *testing.T, h *Harness, code string, value int64) promoFx {
	t.Helper()
	var id string
	err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO promo_codes (code, kind, value, once_per_user, active)
		VALUES ($1, 'fixed', $2, false, true)
		RETURNING id::text`, code, value).Scan(&id)
	if err != nil {
		t.Fatalf("إنشاءُ كود: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM promo_codes WHERE id = $1::uuid`, id)
	})
	return promoFx{ID: id, Code: code}
}

// preview يعاين الكودَ — ويحمل ما كان معروضاً إن طُلب.
func preview(t *testing.T, h *Harness, tok, code string,
	subtotal, fee int64, expected *int64) Res {
	t.Helper()
	body := map[string]any{"code": code, "subtotal": subtotal, "delivery_fee": fee}
	if expected != nil {
		body["expected_discount"] = *expected
	}
	return h.POST("/api/v1/promo/preview", tok, body)
}

// ═════════════════ PR-01 · PR-02 · PR-04 ═════════════════

// TestPR01_PR02_PR04_PromoLossIsNamedAndTotalsAreCurrent **والكودُ
// يسقط فيُقال، والمجموعُ يصير الحقيقيّ.**
func TestPR01_PR02_PR04_PromoLossIsNamedAndTotalsAreCurrent(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ PR-01")
	u := hh.Customer()
	p := newPromo(t, hh, "PRTEST01", 3000)

	// **PR-01 · الكودُ صالحٌ فيُخصَم.**
	first := preview(t, hh, u.Token, p.Code, 20000, 2000, nil)
	if first.Code != http.StatusOK {
		t.Fatalf("**المعاينةُ رُدّت**: %d / %s", first.Code, first.Err())
	}
	j := first.JSON()
	if j["valid"] != true {
		t.Fatalf("**كودٌ صالحٌ قيل إنّه باطل**: %v", j)
	}
	was := int64(j["discount"].(float64))
	if was != 3000 {
		t.Fatalf("**خصمٌ غيرُ متوقَّع**: %v", was)
	}

	// **ويُطفأ الكودُ** — **بساعة الخادم لا بانتظار.**
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE promo_codes SET active = false WHERE id = $1::uuid`, p.ID); err != nil {
		t.Fatalf("إطفاءُ الكود: %v", err)
	}

	// **PR-02 · فيُقال التبدّلُ باسمه.**
	after := preview(t, hh, u.Token, p.Code, 20000, 2000, &was)
	if after.Code != http.StatusOK {
		t.Fatalf("**المعاينةُ بعد الإطفاء رُدّت**: %d / %s", after.Code, after.Err())
	}
	ch := hasChange(changesOf(t, after), "promo_or_discount_changed")
	if ch == nil {
		t.Fatalf("**سقوطُ الخصم لم يُقَل** — **فيُقرأ مجموعٌ لا يقع**: %v", after.JSON())
	}
	if int64(ch["old_value"].(float64)) != 3000 || int64(ch["new_value"].(float64)) != 0 {
		t.Fatalf("**الرقمان لم يُقالا معاً**: %v", ch)
	}

	// **PR-04 · والمجموعُ الحاليُّ بلا خصمٍ ميّت.**
	if after.JSON()["valid"] != false {
		t.Fatalf("**كودٌ مُطفأٌ قيل إنّه صالح**: %v", after.JSON())
	}
	if int64(after.JSON()["discount"].(float64)) != 0 {
		t.Fatalf("**خصمٌ باقٍ بعد الإطفاء**: %v", after.JSON()["discount"])
	}
}

// ═════════════════ PR-02ب — والانتهاءُ بالتاريخ كالإطفاء ═════════════

// TestPR02b_ExpiryByServerTimeIsNamed **والتأريخُ في الماضي حكمُه
// حكمُ الإطفاء** — **وبساعة الخادم.**
func TestPR02b_ExpiryByServerTimeIsNamed(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ PR-02ب")
	u := hh.Customer()
	p := newPromo(t, hh, "PRTEST02", 1500)

	was := int64(1500)
	if v := preview(t, hh, u.Token, p.Code, 20000, 2000, nil).JSON()["discount"]; int64(v.(float64)) != was {
		t.Fatalf("**خصمٌ غيرُ متوقَّع**: %v", v)
	}

	// **ويُؤرَّخ في الماضي** — **ولا يُنتظَر.**
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE promo_codes SET expires_at = $2 WHERE id = $1::uuid`,
		p.ID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("تأريخُ الكود: %v", err)
	}

	got := preview(t, hh, u.Token, p.Code, 20000, 2000, &was)
	if hasChange(changesOf(t, got), "promo_or_discount_changed") == nil {
		t.Fatalf("**انتهاءٌ بالتاريخ لم يُقَل**: %v", got.JSON())
	}
}

// ═════════════════ PR-03 — ولا يُنشأ طلبٌ بخصمٍ ميّت ═════════════════

// TestPR03_StaleDiscountCannotSubmit **والمنعُ عند الإنشاء لا في
// الشاشة.**
//
// **وعميلٌ معدَّلٌ يحمل الخصمَ القديمَ يُردّ** — **والشاشةُ حاجزُ
// راحةٍ لا حاجزُ أمان.**
func TestPR03_StaleDiscountCannotSubmit(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ PR-03")
	u := hh.Customer()
	it := hh.NewItem(5000)
	p := newPromo(t, hh, "PRTEST03", 1000)

	// **ويُطفأ بعد أن رآه.**
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE promo_codes SET active = false WHERE id = $1::uuid`, p.ID); err != nil {
		t.Fatalf("إطفاءُ الكود: %v", err)
	}

	r := hh.POST("/api/v1/orders", u.Token, map[string]any{
		"items":          []map[string]any{{"menu_item_id": it.ID, "qty": 1}},
		"address_text":   "الرقة — شارع الاختبار",
		"lat":            z.Lat,
		"lng":            z.Lng,
		"payment_method": "cash",
		"promo_code":     p.Code,
	})
	if r.Code < 400 {
		t.Fatalf("**طلبٌ أُنشئ بخصمٍ ميّت**: %d — **فيُقيَّد مجموعٌ لا يقع**", r.Code)
	}
}

// ═════════════════ PR-05 — والحالُ الجديدةُ تمرّ ═════════════════

// TestPR05_ReviewedCurrentStateProceeds **ومن راجع الحالَ الجديدةَ
// أتمّ طلبَه.**
//
// **ولا يُحبَس أحدٌ لأنّ كوداً سقط** — **يُقال له ويمضي بلا خصم.**
func TestPR05_ReviewedCurrentStateProceeds(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ PR-05")
	u := hh.Customer()
	it := hh.NewItem(5000)

	r := hh.POST("/api/v1/orders", u.Token, map[string]any{
		"items":          []map[string]any{{"menu_item_id": it.ID, "qty": 1}},
		"address_text":   "الرقة — شارع الاختبار",
		"lat":            z.Lat,
		"lng":            z.Lng,
		"payment_method": "cash",
	})
	if r.Code >= 400 {
		t.Fatalf("**الحالُ الجديدةُ رُدّت**: %d / %s", r.Code, r.Err())
	}
}

// ═════════════════ PR-06 — والمطابقُ يصمت ═════════════════

// TestPR06_UnchangedDiscountIsSilent **ولا يُزعَج بمراجعةٍ لا داعيَ
// لها** (`CA-12`).
//
// **وإنذارٌ بلا تبدّلٍ يُعلَّم عليه** — **فيُقرأ كلُّ إنذارٍ بعده
// ضجيجا.**
func TestPR06_UnchangedDiscountIsSilent(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ PR-06")
	u := hh.Customer()
	p := newPromo(t, hh, "PRTEST06", 2500)

	was := int64(2500)
	got := preview(t, hh, u.Token, p.Code, 20000, 2000, &was)
	if hasChange(changesOf(t, got), "promo_or_discount_changed") != nil {
		t.Fatalf("**خصمٌ لم يتبدّل قيل إنّه تبدّل**: %v", got.JSON())
	}
}
