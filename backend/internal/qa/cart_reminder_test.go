package qa

// ══════════════════════════════════════════════════════════════════════
// **«سلّتك ناطرتك» — تذكيرُ السلّة المتروكة** (قرارُ المالك ٢٠٢٦-١١-٢٥)
// ══════════════════════════════════════════════════════════════════════
//
// **والساعةُ تُثبَّت** (`SetClock`) — فالتأخيرُ ساعةٌ ولا ينتظر اختبارٌ ساعة. **والعدُّ يُقرأ من صندوق
// الزبون نفسِه** (عنوانُ الإشعار وكيانُه وتطبيقُه) لا من قيمة رجوع.

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

const crTitle = "سلّتك ناطرتك 🛒"

// crDay **ظهرُ يومٍ بدمشق** — خارجَ ساعة الهدوء (٢٢←٨).
func crDay(h, m int) time.Time { return lmAt(2031, 5, 12, h, m) }

type crFx struct {
	h    *Harness
	user *User
	item *Item
	now  time.Time
}

func crSetup(t *testing.T) *crFx {
	t.Helper()
	h := New(t)
	h.Setting("cart_reminder.enabled", "true")
	h.Setting("cart_reminder.after_minutes", "60")
	h.Setting("notify.quiet_from", "22")
	h.Setting("notify.quiet_to", "8")
	fx := &crFx{h: h, user: h.NewUser("customer"), item: h.NewItem(1000), now: crDay(12, 0)}
	h.API.CartReminder().SetClock(func() time.Time { return fx.now })
	t.Cleanup(func() { h.API.CartReminder().SetClock(nil) })
	return fx
}

// put **العميلُ يُبلغ حالَ السلّة** — ومن ثمّ يُرجَع تغييرُها إلى `at` في القاعدة.
func (fx *crFx) put(t *testing.T, signature string, at time.Time) {
	t.Helper()
	res := fx.h.Call("PUT", "/api/v1/my/cart-state", fx.user.Token, map[string]any{
		"merchant_id": fx.item.MerchantID, "items_count": 2, "signature": signature,
	}, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("بلاغُ السلّة: %d %s", res.Code, res.Body)
	}
	fx.at(t, at)
}

// at **يؤرّخ آخرَ تغييرٍ** — ولا يُمسّ ختمُ التذكير.
func (fx *crFx) at(t *testing.T, at time.Time) {
	t.Helper()
	if _, err := fx.h.Pool.Exec(ctxBG(),
		`UPDATE cart_reminders SET changed_at = $2 WHERE user_id = $1::uuid`, fx.user.ID, at); err != nil {
		t.Fatal(err)
	}
}

// inbox **تذكيراتُ السلّة في صندوق الزبون.**
func (fx *crFx) inbox(t *testing.T) (n int, body, entity, entityID, apps string) {
	t.Helper()
	rows, err := fx.h.Pool.Query(ctxBG(), `SELECT body, entity, entity_id, push_apps::text, kind
		FROM notifications WHERE user_id = $1::uuid AND title = $2`, fx.user.ID, crTitle)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var kind string
		if err := rows.Scan(&body, &entity, &entityID, &apps, &kind); err != nil {
			t.Fatal(err)
		}
		if kind != "promo" {
			t.Fatalf("نوعُ الإشعار %q لا promo", kind)
		}
		n++
	}
	return
}

func (fx *crFx) sweep() int { return fx.h.API.SweepCartReminder(ctxBG()) }

// TestCartReminder_AfterAnHourOncePerCart **بعد ساعةٍ مرّةً لكلّ سلّة** — لا قبلها، ولا تكرار، ولا بإعادة
// البلاغ بالمحتوى نفسِه؛ **وتغييرُ المحتوى بعد التذكير يفتح تذكيراً جديداً.**
func TestCartReminder_AfterAnHourOncePerCart(t *testing.T) {
	fx := crSetup(t)
	fx.put(t, "a", crDay(11, 0)) // آخرُ تغيير ١١:٠٠، والآن ١٢:٠٠ = ساعةٌ تماماً

	fx.now = crDay(11, 59)
	if n := fx.sweep(); n != 0 {
		t.Fatalf("**ذُكِّر قبل ساعة**: %d", n)
	}
	fx.now = crDay(12, 0)
	if n := fx.sweep(); n != 1 {
		t.Fatalf("**لم يُذكَّر بعد ساعة**: %d", n)
	}
	n, body, entity, entityID, apps := fx.inbox(t)
	if n != 1 {
		t.Fatalf("عددُ التذكيرات %d", n)
	}
	var store string
	if err := fx.h.Pool.QueryRow(ctxBG(), `SELECT name FROM merchants WHERE id = $1::uuid`, fx.item.MerchantID).Scan(&store); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, store) || !strings.Contains(body, "كمّل طلبك") {
		t.Fatalf("نصُّ التذكير لا يذكر المتجرَ ولا «كمّل طلبك»: %q", body)
	}
	if entity != "cart" || entityID != fx.item.MerchantID || apps != "{customer}" {
		t.Fatalf("وجهةُ التذكير: entity=%q id=%q apps=%s", entity, entityID, apps)
	}

	// **مرّةً لكلّ سلّة**: جولاتٌ لاحقةٌ، وبلاغٌ بالمحتوى نفسِه (إعادةُ فتح التطبيق)، لا تكرّر.
	fx.now = crDay(15, 0)
	if n := fx.sweep(); n != 0 {
		t.Fatalf("تكرّر التذكير: %d", n)
	}
	fx.put(t, "a", crDay(13, 0)) // بالمحتوى نفسِه ⇒ الصفُّ لا يُمسّ — فالتاريخُ يبقى ما أرّخناه
	fx.at(t, crDay(11, 0))
	if n := fx.sweep(); n != 0 {
		t.Fatalf("بلاغٌ بالمحتوى نفسِه أعاد التذكير: %d", n)
	}

	// **وتغييرُ المحتوى بعد التذكير يفتح تذكيراً جديداً** (بعد ساعةٍ من التغيير).
	fx.put(t, "b", crDay(13, 0))
	fx.now = crDay(13, 30)
	if n := fx.sweep(); n != 0 {
		t.Fatalf("ذُكِّر قبل ساعةٍ من التغيير الثاني: %d", n)
	}
	fx.now = crDay(14, 0)
	if n := fx.sweep(); n != 1 {
		t.Fatalf("**لم يُذكَّر بعد تغيير السلّة**: %d", n)
	}
	if n, _, _, _, _ := fx.inbox(t); n != 2 {
		t.Fatalf("عددُ التذكيرات بعد التغيير %d", n)
	}
}

// TestCartReminder_CancelledByEmptyingOrOrdering **إفراغُ السلّة أو الطلبُ يلغيه** — الإفراغُ بـ`DELETE`،
// **والطلبُ ولو من عميلٍ لم يُبلغ** (المكنسةُ تقرأ الطلبَ نفسَه).
func TestCartReminder_CancelledByEmptyingOrOrdering(t *testing.T) {
	fx := crSetup(t)
	fx.put(t, "a", crDay(10, 0))
	if res := fx.h.DEL("/api/v1/my/cart-state", fx.user.Token); res.Code != http.StatusOK {
		t.Fatalf("إفراغُ السلّة: %d %s", res.Code, res.Body)
	}
	if n := fx.sweep(); n != 0 {
		t.Fatalf("**ذُكِّر بعد إفراغ السلّة**: %d", n)
	}
	// **وعددٌ صفرٌ في البلاغ إفراغٌ كذلك.**
	fx.put(t, "a", crDay(10, 0))
	res := fx.h.Call("PUT", "/api/v1/my/cart-state", fx.user.Token, map[string]any{
		"merchant_id": fx.item.MerchantID, "items_count": 0, "signature": "",
	}, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("بلاغٌ فارغ: %d %s", res.Code, res.Body)
	}
	if n := fx.sweep(); n != 0 {
		t.Fatalf("**ذُكِّر بعد بلاغٍ فارغ**: %d", n)
	}

	// **والطلبُ بعد آخر تغيير يمنعه** — والصفُّ باقٍ لم يُحذف.
	fx.put(t, "c", crDay(10, 0))
	made := fx.h.POSTKey("/api/v1/orders", fx.user.Token, uniq("k"), orderBody(fx.item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	if _, err := fx.h.Pool.Exec(ctxBG(),
		`UPDATE cart_reminders SET changed_at = now() - interval '2 hours' WHERE user_id = $1::uuid`, fx.user.ID); err != nil {
		t.Fatal(err)
	}
	fx.now = time.Now().Add(time.Minute)
	fx.h.Setting("notify.quiet_from", "0")
	fx.h.Setting("notify.quiet_to", "0")
	if n := fx.sweep(); n != 0 {
		t.Fatalf("**ذُكِّر وقد طلب**: %d", n)
	}
	if n, _, _, _, _ := fx.inbox(t); n != 0 {
		t.Fatalf("وصل إشعارٌ لمن طلب: %d", n)
	}
}

// TestCartReminder_QuietHoursDeferNotDrop **ساعةُ الهدوء تؤجِّل ولا تُسقط** — والمفتاحُ المطفأ يُسكتها كلَّها.
func TestCartReminder_QuietHoursDeferNotDrop(t *testing.T) {
	fx := crSetup(t)
	fx.put(t, "a", crDay(21, 0))

	fx.now = crDay(23, 0) // مرّت ساعتان لكنّنا في الهدوء
	if n := fx.sweep(); n != 0 {
		t.Fatalf("**ذُكِّر في ساعة الهدوء**: %d", n)
	}
	fx.now = crDay(23, 0).Add(9 * time.Hour) // ٨:٠٠ صباحاً — انتهى الهدوء
	if n := fx.sweep(); n != 1 {
		t.Fatalf("**سقط التذكيرُ بدل أن يُؤجَّل**: %d", n)
	}

	// **والمفتاحُ مطفأٌ ⇒ لا شيء** (وهو الافتراضُ حتّى يشغّله المالك).
	fx.put(t, "z", crDay(9, 0))
	fx.h.Setting("cart_reminder.enabled", "false")
	fx.now = crDay(12, 0)
	if n := fx.sweep(); n != 0 {
		t.Fatalf("**ذُكِّر والمفتاحُ مطفأ**: %d", n)
	}
}

// TestCartReminder_StaleCartNotRevived **سلّةُ أمسِ بعد يومٍ لا تُحيا** — انقطاعُ المكنسة أو تفعيلُ المفتاح
// متأخّراً لا يُرسل تذكيراً عن سلّةٍ قديمة.
func TestCartReminder_StaleCartNotRevived(t *testing.T) {
	fx := crSetup(t)
	fx.put(t, "a", crDay(12, 0).Add(-30*time.Hour))
	if n := fx.sweep(); n != 0 {
		t.Fatalf("**ذُكِّر بسلّةٍ عمرُها ٣٠ ساعة**: %d", n)
	}
}

// TestCartReminder_BadReportsRefused **بلاغٌ فاسدٌ يُردّ** — متجرٌ غيرُ معروف، وبلا توكن.
func TestCartReminder_BadReportsRefused(t *testing.T) {
	fx := crSetup(t)
	res := fx.h.Call("PUT", "/api/v1/my/cart-state", fx.user.Token, map[string]any{
		"merchant_id": "not-a-uuid", "items_count": 1, "signature": "a",
	}, nil)
	if res.Code != http.StatusBadRequest && res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("معرّفُ متجرٍ فاسدٌ قُبل: %d %s", res.Code, res.Body)
	}
	res = fx.h.Call("PUT", "/api/v1/my/cart-state", "", map[string]any{
		"merchant_id": fx.item.MerchantID, "items_count": 1, "signature": "a",
	}, nil)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("**بلاغٌ بلا توكن قُبل**: %d", res.Code)
	}
	var n int
	if err := fx.h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM cart_reminders WHERE user_id = $1::uuid`, fx.user.ID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("خُزِّن بلاغٌ مرفوض: n=%d err=%v", n, err)
	}
}
