package orders

import (
	"context"
	"slices"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

// TestRolesUnderMode الوضعُ يمنع فعلاً لا في الشاشة وحدها.
//
// **هذا هو الاختبارُ الذي لولاه لبقي الوضعُ زينةً**: كانت بوابةُ المتجر تُخفي
// أزرارَ القبول في وضع «المنصة تدير»، والمحرّكُ لا يعرف الوضعَ أصلاً — فمن
// استدعى الواجهةَ البرمجية مباشرةً قبِل طلبَه رغم أن السياسة تمنعه.
func TestRolesUnderMode(t *testing.T) {
	const (
		platformManages = false // المنصة تدير
		merchantManages = true  // المتجر يدير
	)
	cases := []struct {
		name        string
		selfManage  bool
		from, to    string
		roles       []string
		driverHolds bool
		allowed     bool
	}{
		// ── المتجر يدير: المنصةُ عينٌ لا يد ──────────────────────────────
		{"المتجر يقبل طلبه", merchantManages, StPending, StAccepted,
			[]string{"merchant"}, false, true},
		{"العملياتُ لا تقبل نيابةً عنه", merchantManages, StPending, StAccepted,
			[]string{"ops"}, false, false},
		{"ولا ترفض نيابةً عنه", merchantManages, StPending, StRejected,
			[]string{"ops"}, false, false},
		// وما عدا القبول والرفض تبقى العمليات عاملةً — التوصيلُ شأنُها.
		{"العملياتُ تطلب سائقاً", merchantManages, StPreparing, StDispatching,
			[]string{"ops"}, false, true},

		// ── المنصة تدير: المتجرُ يشاهد ──────────────────────────────────
		{"العملياتُ تقبل", platformManages, StPending, StAccepted,
			[]string{"ops"}, false, true},
		{"المتجرُ لا يقبل", platformManages, StPending, StAccepted,
			[]string{"merchant"}, false, false},
		{"ولا يرفض", platformManages, StPending, StRejected,
			[]string{"merchant"}, false, false},
		{"ولا يبدأ تحضيراً", platformManages, StAccepted, StPreparing,
			[]string{"merchant"}, false, false},
		{"ولا يلغي", platformManages, StAccepted, StCancelled,
			[]string{"merchant"}, false, false},

		// ── حدُّ الإلغاء: قبل السائق وبعده ───────────────────────────────
		{"والسائقُ يُفشله من عند الباب", platformManages, StAtPickup, StFailed,
			[]string{"driver"}, true, true},

		// ── «المنصةُ عينٌ لا يد» — قرارُ المالك ٢٠٢٦-٠٨-٠٢ ──────────────
		//
		// **وأوّلُها وقع أمامنا حيّاً**: ضغطت العملياتُ «بدء التحضير» بعد
		// ثانيةٍ من القبول والمتجرُ لم يُبلَّغ بعد.
		{"العملياتُ لا تُعلن تحضيراً", platformManages, StAccepted, StPreparing,
			[]string{"ops"}, false, false},
		{"ولا تقول وصلَ المتجر", platformManages, StAssigned, StAtPickup,
			[]string{"ops"}, true, false},
		{"ولا تقول استلم", platformManages, StAtPickup, StPickedUp,
			[]string{"ops"}, true, false},
		{"ولا تقول سلّم", platformManages, StAtDropoff, StDelivered,
			[]string{"ops"}, true, false},
		{"ولا تُفشل نيابةً عنه", platformManages, StAtDropoff, StFailed,
			[]string{"ops"}, true, false},
		// **والسائقُ يملكها كلَّها** — فالنزعُ من العمليات لا يعطّل الطريق.
		{"والسائقُ يملكها", platformManages, StAtPickup, StPickedUp,
			[]string{"driver"}, true, true},

		// ── الإلغاء: قبل التحويل وبعده ──────────────────────────────────
		//
		// **ما قبل التحويل بيدها**: طلبٌ لم يعلم به مطبخٌ ولا تحرّك له سائق.
		{"تُلغي ما لم يُحوَّل", platformManages, StAccepted, StCancelled,
			[]string{"ops"}, false, true},
		// **وما بعده ليس لها** — ولو لم يمسكه سائقٌ بعد.
		{"ولا تُلغي بعد التحويل", platformManages, StDispatching, StCancelled,
			[]string{"ops"}, false, false},
		{"ولا بعد الإسناد", platformManages, StAssigned, StCancelled,
			[]string{"ops"}, true, false},

		// ── الأدمن فوق الوضعين ─────────────────────────────────────────
		{"الأدمن يقبل ولو كان المتجر يدير", merchantManages, StPending, StAccepted,
			[]string{"admin"}, false, true},
		{"والأدمن يلغي ولو أمسكه سائق", platformManages, StAtPickup, StCancelled,
			[]string{"admin"}, true, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eff := rolesUnderMode(c.selfManage, c.from, c.to, c.roles, c.driverHolds)
			got := canTransition(KindStandard, c.from, c.to, eff)
			if got != c.allowed {
				t.Errorf("النتيجة %v والمتوقّع %v (الأدوار بعد التنقية: %v)",
					got, c.allowed, eff)
			}
		})
	}
}

// TestRolesUnderMode_KeepsOtherRoles التنقيةُ تُسقط الدورَ الممنوع وحده.
//
// **ولا يُحرَم أحدٌ من حقٍّ يملكه بسببٍ يخصّ حقّاً آخر**: من يحمل دورين
// يمرّ بالدور الذي يخوّله. ولو رُفض الفاعلُ كلُّه بدل تنقية أدواره لسقط هذا.
func TestRolesUnderMode_KeepsOtherRoles(t *testing.T) {
	// صاحبُ متجرٍ يعمل في العمليات أيضاً: دورُ المتجر ممنوعٌ هنا، ودورُ
	// العمليات هو ما يخوّله.
	eff := rolesUnderMode(false, StPending, StAccepted,
		[]string{"merchant", "ops"}, false)
	if slices.Contains(eff, "merchant") {
		t.Errorf("دورُ المتجر بقي: %v", eff)
	}
	if !canTransition(KindStandard, StPending, StAccepted, eff) {
		t.Errorf("دورُ العمليات سقط معه: %v", eff)
	}
}

// TestAdminIsAboveAuthorityNotAboveKnowledge المالكُ فوق السلطة لا فوق العِلم.
//
// # ما يمسكه
//
// كان `if isAdmin { return roles }` يتخطّى الحُرّاسَ كلَّها — **فسقط أهمُّها
// عمّن أحدث المشكلة**: شكوى المالك الحيّة قالت «ضُغطت **بيد الأدمن**».
//
// **وثلاثةُ أزرارٍ بقيت ظاهرةً له وحدَه**، وكشفها بعينه في تجربة ٢٠٢٦-٠٨-٠٢
// بعد أن قرأ التوثيقَ وظنّ أنّ شيئاً لم يتغيّر. **وكان محقّاً: من مقعده لم
// يتغيّر شيء.**
func TestAdminIsAboveAuthorityNotAboveKnowledge(t *testing.T) {
	admin := []string{"admin", "ops"}

	// ── ما لا يملكه أحدٌ: عِلمٌ لا سلطة ──────────────────────────────────
	if canTransition(KindStandard, StAccepted, StPreparing,
		rolesUnderMode(false, StAccepted, StPreparing, admin, false)) {
		t.Error("المالكُ يُعلن بدءَ تحضيرٍ في مطبخٍ لا يراه")
	}
	for _, to := range []string{StAtPickup, StPickedUp, StOnTheWay, StAtDropoff, StDelivered} {
		eff := rolesUnderMode(false, StAssigned, to, admin, true)
		if canTransition(KindStandard, StAssigned, to, eff) {
			t.Errorf("المالكُ يُعلن %q بلا توقيع — **ومراحلُ الطريق مقروءةٌ لا ملموسة**", to)
		}
	}

	// ── ولا يفتحها توقيعٌ ولا سلطة ───────────────────────────────────────
	//
	// **والتوقيعُ يُصدّق السجلَّ ولا يُصدّق الواقع.** ومن يجلس خلف مكتبه لا
	// يعرف أين المتجرُ ولا تحرّك من مكانه — **فكيف يقول إنّ السائقَ استلم؟**
	// (قرارُ المالك ٢٠٢٦-٠٨-٠٤.)
	//
	// **ومخرجُ الطلب العالق التحريرُ لا الإعلان** — يُفحص أدناه.
	for _, to := range []string{StAtPickup, StPickedUp, StOnTheWay, StAtDropoff, StDelivered, StFailed} {
		if canTransition(KindStandard, StAssigned, to, rolesUnderMode(false, StAssigned, to, admin, true)) {
			t.Errorf("المنصةُ أعلنت %q — ومراحلُ الطريق لمن يسير فيها", to)
		}
	}

	// ── وما هو سلطةٌ يبقى له ────────────────────────────────────────────
	if !canTransition(KindStandard, StDispatching, StCancelled,
		rolesUnderMode(false, StDispatching, StCancelled, admin, false)) {
		t.Error("المالكُ فقد الإلغاءَ بعد التحويل — **وهو سلطتُه لا عِلمُه**")
	}
	if !canTransition(KindStandard, StPending, StAccepted,
		rolesUnderMode(true, StPending, StAccepted, admin, false)) {
		t.Error("المالكُ فقد القبولَ نيابةً عن متجرٍ لا يستجيب")
	}

	// ── والعملياتُ تبقى ممنوعةً حيث كانت ────────────────────────────────
	ops := []string{"ops"}
	if canTransition(KindStandard, StDispatching, StCancelled,
		rolesUnderMode(false, StDispatching, StCancelled, ops, false)) {
		t.Error("العملياتُ تُلغي بعد التحويل")
	}
}

// TestPlatformWatchesAfterPickup **بعد الاستلام مراقبةٌ لا تحكّم.**
//
// # القاعدة
//
// **بمجرّد أن تصير البضاعةُ بيد السائق تفقد المنصةُ كلَّ زرّ.** (قرارُ المالك
// ٢٠٢٦-٠٨-٠٤.)
//
// # ولماذا لا يفتحه التوقيع
//
// التوقيعُ يُصدّق السجلَّ («أعلنتها المنصة») **ولا يُصدّق الواقع**: من يقول
// «تمّ التسليم» وهو في المكتب يقول ما لا يعلمه. **والمالُ يتحرّك على قوله** —
// يُقيَّد للمتجر ولنا وللمندوب، ويُحمَّل السائقُ نقداً لم يقبضه.
//
// **ووقع أمام المالك** (٢٠٢٦-٠٨-٠٤): زرُّ «تدخّل يدويّ: تمّ التسليم» معروضٌ
// على طلبٍ حالتُه «السائق وصل».
func TestPlatformWatchesAfterPickup(t *testing.T) {
	admin := []string{"admin", "ops"}

	// ── بعد الاستلام: لا شيءَ بيدها ──────────────────────────────────────
	after := []struct{ from, to string }{
		{StPickedUp, StOnTheWay},
		{StOnTheWay, StAtDropoff},
		{StAtDropoff, StDelivered},
		{StAtDropoff, StFailed},
	}
	for _, c := range after {
		if canTransition(KindStandard, c.from, c.to, rolesUnderMode(false, c.from, c.to, admin, true)) {
			t.Errorf("المنصةُ تحكّمت في %q←%q بعد أن صارت البضاعةُ بيد السائق", c.from, c.to)
		}
	}

	// ── والتحريرُ يبقى مفتوحاً — فليس إعلانَ واقعة ────────────────────
	//
	// **سائقٌ اختفى بطلبٍ في يده** كان يترك الطلبَ إلى الأبد: لا يُلغى ولا
	// يُفشل ولا يُسنَد لغيره. **والحارسُ الأوّلُ سدّ هذا البابَ معه** —
	// وأمسكه `TestPostPickupHasExit`.
	//
	//	«تمّ التسليم»  ←  إعلانُ واقعةٍ لا يعلمها من في المكتب — يُمنع
	//	«حرّر الطلب»   ←  التخلّي عن إسنادٍ لم يُثمر — يبقى
	for _, from := range []string{StPickedUp, StOnTheWay, StAtDropoff} {
		if !canTransition(KindStandard, from, StDispatching,
			rolesUnderMode(false, from, StDispatching, []string{"ops"}, true)) {
			t.Errorf("العملياتُ فقدت تحريرَ طلبٍ عالقٍ من %q — والزبونُ ينتظر من لن يأتي", from)
		}
	}

	// ── وقبل الاستلام كذلك — لا استثناءَ لمرحلةِ طريق ────────────────────
	//
	// **كان التوقيعُ يفتح `assigned→at_pickup` و`at_pickup→picked_up`** بحجّة
	// هاتفٍ نفدت بطاريتُه. **وقولُ «وصل المتجر» من مكتبٍ ادّعاءُ ما لا يُعلَم
	// كقولِ «تمّ التسليم»** — والمخرجُ التحريرُ لا الإعلان.
	for _, c := range []struct{ from, to string }{
		{StAssigned, StAtPickup},
		{StAtPickup, StPickedUp},
	} {
		if canTransition(KindStandard, c.from, c.to, rolesUnderMode(false, c.from, c.to, admin, true)) {
			t.Errorf("المنصةُ أعلنت %q←%q — وهي مرحلةُ طريق", c.from, c.to)
		}
	}
}

// TestMerchantsSelfManage_ReadsTheMode **الوضعُ يُقرأ من مفتاحٍ واحد.**
//
// كان المفتاحُ منطقيّاً يُقرأ في أربعة مواضع، **فلمّا صار وضعين لزم تعديلُ
// أربعةٍ** — ومن نسي واحداً جعل شاشةً تعمل بوضعٍ ومحرّكاً بآخر.
//
// **وبلا مخزنٍ وضعُ المنصة**: المكتبُ يقبل ويحوّل، **فلا يبقى طلبٌ ينتظر
// متجراً لا يعلم أنّ عليه أن ينظر.**
func TestMerchantsSelfManage_ReadsTheMode(t *testing.T) {
	var s Service
	if s.MerchantsSelfManage(context.Background()) {
		t.Error("بلا مخزنٍ قُرئ وضعُ المتاجر — والافتراضُ وضعُ المنصة")
	}
}

// TestOrdersMode_BothValuesAreInTheCatalog **ولفظا الوضعين هما خياراه.**
//
// **وقيمةٌ في الشيفرة ليست في الفهرس تُقرأ ولا تُكتب**: يعمل المحرّكُ بها
// **ولا يستطيع أحدٌ أن يعيده إليها من الشاشة.**
func TestOrdersMode_BothValuesAreInTheCatalog(t *testing.T) {
	def, ok := settings.Lookup("platform.orders_mode")
	if !ok {
		t.Fatal("مفتاحُ وضع الطلبات ليس في الفهرس")
	}
	for _, want := range []string{ModePlatform, ModeMerchants} {
		if !slices.Contains(def.Options, want) {
			t.Errorf("الوضع %q ليس من خيارات المفتاح %v", want, def.Options)
		}
	}
	if def.Default != ModePlatform {
		t.Errorf("الافتراضُ %v والمنتظَر %q — وهو أقلُّ الوضعين مفاجأة", def.Default, ModePlatform)
	}
}
