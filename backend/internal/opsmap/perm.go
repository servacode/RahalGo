// Package opsmap خريطةُ العمليات — **ما يقرؤه المكتبُ عن الأرض.**
//
// # ولماذا حزمةٌ لا حزمةُ خادم
//
// **الاستعلاماتُ الجغرافيّةُ تُقاس وتُختبر بلا خادم** — **ومن كتبها في
// معالجٍ لم يستطع أن يختبر مضلَّعاً إلّا بنداءٍ كامل.**
//
// # ولا تُكتب هنا حقيقةٌ ثانية
//
// **الموضعُ يُقرأ من `users.last_location`** كما هو، **ولا يُكتب** —
// (البند ٤٥: **لا يُمسّ كاتبُ الموضع**). **والخريطةُ تقرأ ولا تصلح.**
package opsmap

import "sort"

// Perm صلاحيّةٌ مسمّاةٌ في الخريطة (البند ٣٢).
//
// # ولماذا أسماءٌ والأدوارُ خشنةٌ اليوم
//
// **`AQ-1` فجوةٌ قائمة**: الأدوارُ ثلاثةٌ خشنةٌ في الخادم (`admin` ·
// `ops` · `finance`) **ولا صلاحيّاتٍ دقيقة.** **ولا يُبنى نظامُ صلاحيّاتٍ
// كاملٌ هنا** — خارجَ النطاق.
//
// **لكنّ الأسماءَ تُكتب من اليوم** — **فحين يُبنى `AQ-1` تُربَط هذه
// بالأدوار الجديدة في موضعٍ واحد**، ولا يُفتَّش عن `roles` مبعثرةٍ في
// عشرين معالجاً.
type Perm string

const (
	// PermViewMap **دخولُ الخريطة أصلاً** — ولا تُفتح بتوكن إدارةٍ وحدَه.
	PermViewMap Perm = "VIEW_OPERATIONS_MAP"

	PermViewDrivers   Perm = "VIEW_DRIVER_LOCATIONS"
	PermViewMerchants Perm = "VIEW_MERCHANT_LOCATIONS"
	PermViewOrders    Perm = "VIEW_ACTIVE_ORDERS"

	PermManageCoverage Perm = "MANAGE_COVERAGE"
	PermManageBranches Perm = "MANAGE_BRANCHES"

	PermViewRepActivity Perm = "VIEW_REP_ACTIVITY"
	PermViewDemand      Perm = "VIEW_DEMAND_ANALYTICS"

	// PermViewMoney **الأرقامُ الماليّةُ في البطاقات** — صندوقُ السائق
	// وأرباحُ المندوب. **وليست جزءاً من رؤية الخريطة.**
	PermViewMoney Perm = "VIEW_MAP_FINANCIALS"
)

// All الصلاحيّاتُ كلُّها بترتيبٍ ثابت.
func All() []Perm {
	return []Perm{
		PermViewMap, PermViewDrivers, PermViewMerchants, PermViewOrders,
		PermManageCoverage, PermManageBranches,
		PermViewRepActivity, PermViewDemand, PermViewMoney,
	}
}

// rolePerms ربطُ الأدوار الخشنة بالصلاحيّات المسمّاة.
//
// # ولماذا تُمنَع الماليّةُ من مواضع السائقين
//
// **موضعُ إنسانٍ ليس رقماً ماليّاً** — **ومن لا يوزّع الطلبات لا يحتاج
// أن يعرف أين يقف السائقُ الآن.** (البند ٣٣: الخريطةُ ليست إعفاءً من
// الخصوصيّة.)
//
// # ولماذا لا تُدير العملياتُ الفروعَ والتغطية
//
// **رسمُ التغطية قرارُ عملٍ لا تشغيلٌ يوميّ** — ومن بدّل مضلَّعاً بدّل
// من تصله المنصّةُ أصلاً.
var rolePerms = map[string][]Perm{
	"admin": All(),
	"ops": {
		PermViewMap, PermViewDrivers, PermViewMerchants, PermViewOrders,
		PermViewRepActivity, PermViewDemand,
	},
	"finance": {
		PermViewMap, PermViewMerchants, PermViewOrders,
		PermViewDemand, PermViewMoney,
	},
}

// Allows **أيملك صاحبُ هذه الأدوار هذه الصلاحيّة؟**
func Allows(roles []string, p Perm) bool {
	for _, r := range roles {
		for _, have := range rolePerms[r] {
			if have == p {
				return true
			}
		}
	}
	return false
}

// Granted ما يملكه صاحبُ هذه الأدوار — **تُرسَل إلى الواجهة.**
//
// **فلا ترسم الواجهةُ طبقةً لا يملكها صاحبُها** — **ورؤيةُ طبقةٍ تردّ
// `403` أسوأُ من غيابها** (وهي قاعدةُ سايدبار اللوحة نفسُها).
func Granted(roles []string) []Perm {
	seen := map[Perm]bool{}
	for _, r := range roles {
		for _, p := range rolePerms[r] {
			seen[p] = true
		}
	}
	out := make([]Perm, 0, len(seen))
	for _, p := range All() {
		if seen[p] {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
