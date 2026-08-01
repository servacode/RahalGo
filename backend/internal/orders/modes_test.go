package orders

import (
	"slices"
	"testing"
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
		{"العملياتُ تلغي ما لم يمسكه سائق", platformManages, StDispatching, StCancelled,
			[]string{"ops"}, false, true},
		{"ولا تلغي ما في يد سائق", platformManages, StAtPickup, StCancelled,
			[]string{"ops"}, true, false},
		{"والسائقُ يُفشله من عند الباب", platformManages, StAtPickup, StFailed,
			[]string{"driver"}, true, true},

		// ── الأدمن فوق الوضعين ─────────────────────────────────────────
		{"الأدمن يقبل ولو كان المتجر يدير", merchantManages, StPending, StAccepted,
			[]string{"admin"}, false, true},
		{"والأدمن يلغي ولو أمسكه سائق", platformManages, StAtPickup, StCancelled,
			[]string{"admin"}, true, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eff := rolesUnderMode(c.selfManage, c.from, c.to, c.roles, c.driverHolds)
			got := canTransition(c.from, c.to, eff)
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
	if !canTransition(StPending, StAccepted, eff) {
		t.Errorf("دورُ العمليات سقط معه: %v", eff)
	}
}
