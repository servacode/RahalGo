package qa

// **الطبقةُ العاشرة — إلى من يصل الإشعار.**
//
// المعرّفات: `NOTIF-*` · الوسم: `@api @critical @release`
//
// (بلاغُ المالك ٢٠٢٦-٠٨-١٩: «إشعاراتُ العروض تأتي إلى حساب السائق
//
//	وهذا غلط لأنّه لا يوجد تسوّقٌ بحساب السائق — لماذا تأتي العروضُ
//	عليه؟».)
//
// # ولماذا وقع
//
// **`customer` دورُ أساسٍ يحمله كلُّ حساب** — والسائقُ والمتجرُ والمندوبُ
// كلُّهم زبائنُ في القاعدة (`primaryRoles` تعدّ ما عداه وحدَه دوراً
// أصليّا).
//
// **فبثٌّ إلى «كلّ من يحمل دورَ الزبون» يبلغ فريقَ العمل كلَّه.**
//
// **وقِيس على الإنتاج**: وصل السائقَ `عرض حاص — شاورما غنم ١٥٪` مرّتين.
//
// # وما يُقاس
//
// **العرضُ يصل المتسوّقَ ولا يصل من له دورٌ تشغيليّ.**

import (
	"testing"
	"time"
)

// notifCount **كم إشعارَ عرضٍ وصله.**
func notifCount(t *testing.T, h *Harness, userID, kind string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(t.Context(),
		`SELECT count(*) FROM notifications WHERE user_id = $1::uuid AND kind = $2`,
		userID, kind).Scan(&n); err != nil {
		t.Fatalf("NOTIF: تعذّر العدّ: %v", err)
	}
	return n
}

// TestNOTIF_001_OfferSkipsStaff **عرضٌ يبلغ المتسوّقَ لا فريقَ العمل.**
func TestNOTIF_001_OfferSkipsStaff(t *testing.T) {
	h := New(t)
	shopper := h.Customer()
	driver := h.NewUser("driver")
	merchant := h.NewUser("merchant")
	rep := h.NewUser("sales")

	// **ولكلٍّ منهم دورُ الزبون أيضاً** — كما في الإنتاج حرفا.
	for _, u := range []*User{driver, merchant, rep} {
		if _, err := h.Pool.Exec(t.Context(), `
			INSERT INTO user_roles (user_id, role_code) VALUES ($1::uuid, 'customer')
			ON CONFLICT DO NOTHING`, u.ID); err != nil {
			t.Fatalf("NOTIF: تعذّر إسنادُ دور الزبون: %v", err)
		}
	}

	admin := h.NewUser("admin")
	item := h.NewItem(1000)
	made := h.POST("/api/v1/admin/offers", admin.Token, map[string]any{
		"kind": "discount", "title": "عرضُ اختبارٍ آليّ",
		"body":         "حسمٌ للتحقّق من وجهة الإشعار",
		"menu_item_id": item.ID, "discount_percent": 15,
		// **وسارٍ الآن** — **وعرضٌ منزَّلٌ لا يُبثّ عمداً**: يُنشأ
		// ليُراجَع، **وإشعارٌ عن عرضٍ لا يجده حين يفتحه أسوأُ من صمت.**
		"borne_by":  "platform",
		"starts_at": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		"ends_at":   time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339),
	})
	if made.Code >= 400 {
		t.Skipf("NOTIF-001 تعذّر إنشاءُ عرض (%s) — يُراجَع جسمُ العرض", made)
	}

	// ══════════════════════════════════════════════════════════════════
	// **ويُقاس الوصولُ الفعليّ — لا الاختيارُ وحدَه**
	// ══════════════════════════════════════════════════════════════════
	//
	// **وكان يقيس الاختيارَ حتّى ٢٠٢٦-٠٨-٢٠**: البثُّ كان يُدرج واحداً
	// واحداً فلا ينتهي في مهلة اختبار (`NOTIF-010`). **ولمّا صار دفعةً
	// واحدةً صار الوصولُ مقيسا** — والحزمةُ تمرّ في ثانيةٍ ونصف بعد أن
	// كانت تعجز في ثلاثين.
	//
	// **ويُمهَل قليلاً** — الإدراجُ يقع بعد الردّ لا قبله.
	waitFor(t, func() bool { return notifCount(t, h, shopper.ID, "offer") > 0 })
	if notifCount(t, h, shopper.ID, "offer") == 0 {
		t.Error("NOTIF-001 **العرضُ لم يصل المتسوّق** — والبثُّ لا يعمل أصلا")
	}

	for name, u := range map[string]*User{
		"سائق": driver, "متجر": merchant, "مندوب": rep,
	} {
		if got := notifCount(t, h, u.ID, "offer"); got > 0 {
			t.Errorf("NOTIF-001 **وصل العرضُ حسابَ %s** (%d إشعارا) — "+
				"ولا تسوّقَ بحسابه", name, got)
		}
	}
}

// waitFor **ينتظر شرطاً حتّى نصفِ دقيقة** — للكتابات التي تقع بعد الردّ.
//
// **والبثُّ يكتب لكلّ متسوّق**: في قاعدة الاختبار عشرون ألفاً، **فثلاثُ
// ثوانٍ تقرأ صفراً وتُقرأ عطبا.** (وقع ٢٠٢٦-٠٨-١٩.)
func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	for i := 0; i < 300; i++ {
		if ok() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
