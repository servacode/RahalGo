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
// # وقد رُبطت بالقدرات
//
// **كُتبت الأسماءُ يومَ كانت الأدوارُ ثلاثةً خشنةً** (`admin` · `ops` ·
// `finance`)، **على أن تُربَط بـ`AQ-1` حين يُبنى.** **وقد بُني**
// (`ADG-1`/`ADG-2`) — **فالربطُ هنا بالقدرة لا بالاسم.**
//
// **ولولا ذلك لَما فتح أحدٌ الخريطةَ بدورٍ كانونيّ**: `operations`
// ليس `ops`، **فكان يُردّ عند الباب وإن ملك كلَّ قدرةٍ تلزمه.**
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

// capPerms **ربطُ صلاحيّات الخريطة بقدرات `AQ-1`.**
//
// # ولماذا موضعٌ واحد
//
// **الخريطةُ طبقاتٌ أدقُّ من القدرات** — موضعُ سائقٍ غيرُ موضعِ متجرٍ
// غيرُ رقمِ صندوقه. **ولا تُخترَع قدرةٌ لكلّ طبقة**؛ **تُشتقُّ الطبقةُ
// من القدرة التي تحرس مسارَها في الجدول المركزيّ**، فلا تفترق حراستان.
//
// # ولماذا تُمنَع الماليّةُ من مواضع السائقين
//
// **موضعُ إنسانٍ ليس رقماً ماليّاً** — **ومن لا يوزّع الطلبات لا يحتاج
// أن يعرف أين يقف السائقُ الآن.** (البند ٣٣: الخريطةُ ليست إعفاءً من
// الخصوصيّة.) **والماليّةُ لا تملك `drivers.read`** فلا تراها.
//
// # ولماذا لا تُدير العملياتُ الفروعَ والتغطية
//
// **رسمُ التغطية قرارُ عملٍ لا تشغيلٌ يوميّ** — ومن بدّل مضلَّعاً بدّل
// من تصله المنصّةُ أصلاً. **وهي بقدرة الإعدادات العامّة.**
var capPerms = map[string][]Perm{
	"orders.read":             {PermViewMap, PermViewOrders, PermViewMerchants},
	"drivers.read":            {PermViewDrivers},
	"merchants.manage":        {PermViewRepActivity},
	"analytics.read":          {PermViewDemand},
	"finance.read":            {PermViewMoney},
	"settings.general.manage": {PermManageCoverage, PermManageBranches},
}

// Allows **أيملك صاحبُ هذه القدرات هذه الصلاحيّة؟**
func Allows(caps []string, p Perm) bool {
	for _, r := range caps {
		for _, have := range capPerms[r] {
			if have == p {
				return true
			}
		}
	}
	return false
}

// Granted ما يملكه صاحبُ هذه القدرات — **تُرسَل إلى الواجهة.**
//
// **فلا ترسم الواجهةُ طبقةً لا يملكها صاحبُها** — **ورؤيةُ طبقةٍ تردّ
// `403` أسوأُ من غيابها** (وهي قاعدةُ سايدبار اللوحة نفسُها).
func Granted(caps []string) []Perm {
	seen := map[Perm]bool{}
	for _, r := range caps {
		for _, p := range capPerms[r] {
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
