package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// ══════════════════════════════════════════════════════════════════════
// **التعليقُ يمنع نشاطاً جديداً ولا يشلّ طلباً قائماً** — `XG-22`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **`RequireAuth` يردّ `403` على كلّ نداءٍ مصادَقٍ فورَ التعليق.**
// **فسائقٌ عُلِّق وهو يحمل طلباً لا يستطيع تسليمَه ولا رؤيتَه** —
// **والطلبُ يبقى معلَّقاً بمن لا يقدر**، والزبونُ ينتظر.
//
// # والحالان في المنتَج ليسا واحداً
//
// **`AdminUpdateUser` يقبل ثلاثاً**: `active` · `suspended` · `blocked`.
// **والعاديُّ `suspended`، والاستثنائيُّ `blocked`** — **ولم يُخترَع
// بابٌ ثالث**، وإنّما فُرّق بين قائمَين.
//
// **فالحظرُ يقف عند كلّ شيءٍ ولا استثناءَ فيه** — احتيالٌ أو خطرٌ أو
// حسابٌ مسروق. **والتعليقُ يمنع الجديدَ ويترك القائمَ يبلغ نهايتَه.**
//
// # والاستثناءُ ضيّقٌ ثلاثيُّ الشرط
//
//	الفاعلُ  +  طلبٌ بعينه في مسار النداء  +  فعلٌ من قائمةٍ مغلقة
//
// **ولا يُقال «معلَّقٌ ومعه طلبٌ فليعمل ما شاء»**: **من فتح البابَ
// بشرطٍ واحدٍ فتحه كلَّه.**
//
// **والقائمةُ مغلقةٌ لا نمطٌ عامّ** — **ومسارٌ جديدٌ لا يدخلها بالسهو.**

// continuationRoutes **أفعالُ الاستمرار المسموحة لمعلَّقٍ** — بالدور.
//
// **ولكلّ مدخلٍ مسارُ الطلب في العنوان** — **فيُقرأ منه ويُتحقَّق أنّ
// الفاعلَ طرفٌ فيه.**
//
// **وليس فيها إنشاءٌ ولا قبولٌ جديد ولا وردية** — **فتلك أنشطةٌ
// جديدةٌ يمنعها التعليق.**
var continuationRoutes = []struct {
	Method string
	Prefix string // بادئةٌ قبل معرّف الطلب
	Suffix string // ما بعده — وفارغٌ يعني نهايةَ العنوان
	Role   string
}{
	// **السائق** — يُتمّ رحلتَه أو يعلن تعذّرها.
	{"POST", "/api/v1/driver/orders/", "/transition", "driver"},
	{"POST", "/api/v1/driver/orders/", "/proof", "driver"},
	{"POST", "/api/v1/driver/orders/", "/proof/skip", "driver"},
	{"GET", "/api/v1/driver/orders/", "", "driver"},

	// **المتجر** — يقبل أو يرفض ما بين يديه.
	{"POST", "/api/v1/merchant/orders/", "/transition", "merchant"},
	{"GET", "/api/v1/merchant/orders/", "", "merchant"},

	// **الزبون** — يرى طلبَه القائمَ ويستطيع إلغاءه.
	//
	// **ودفع مالَه** — **فحجبُ رؤيته عنه عقوبةٌ على مالٍ لا على فعل.**
	{"GET", "/api/v1/orders/", "", "customer"},
	{"POST", "/api/v1/orders/", "/cancel", "customer"},
}

// suspendedMayContinue أيسمح لهذا النداءِ من معلَّق؟
//
// **ويُرجع `false` عند أدنى شكّ** — **ومن شكّ فمنع أخطأ في الأمان،
// ومن شكّ فأذِن أخطأ في المال.**
func (s *Server) suspendedMayContinue(ctx context.Context, r *http.Request,
	userID string, roles []string) bool {
	for _, c := range continuationRoutes {
		if r.Method != c.Method || !strings.HasPrefix(r.URL.Path, c.Prefix) {
			continue
		}
		if !hasRole(roles, c.Role) {
			continue
		}
		rest := strings.TrimPrefix(r.URL.Path, c.Prefix)
		id, ok := strings.CutSuffix(rest, c.Suffix)
		if !ok || id == "" || strings.Contains(id, "/") || !isUUID(id) {
			continue
		}
		return s.isLiveParticipant(ctx, id, userID, c.Role)
	}
	return false
}

// isLiveParticipant أهو طرفٌ في طلبٍ **ما زال حيّاً**؟
//
// **والحالُ النهائيّةُ تُنهي الإذنَ** — **فمن سلّم طلبَه عاد معلَّقاً
// كسائر المعلَّقين.**
//
// **ويُسأل الدفترُ عن الطرف بدوره**: **سائقُ الطلب هو `driver_id`،
// وصاحبُ المتجر هو `owner_user_id` لمتجره، والزبونُ `customer_id`** —
// **ولا يكفي أن يكون له طلبٌ ما، بل هذا الطلبُ بعينه.**
func (s *Server) isLiveParticipant(ctx context.Context, orderID, userID, role string) bool {
	var col string
	switch role {
	case "driver":
		col = `o.driver_id = $2::uuid`
	case "customer":
		col = `o.customer_id = $2::uuid`
	case "merchant":
		col = `EXISTS (SELECT 1 FROM merchants m
		                WHERE m.id = o.merchant_id AND m.owner_user_id = $2::uuid)`
	default:
		return false
	}
	var live bool
	if err := s.pg.QueryRow(ctx, `
		SELECT NOT (o.status = ANY($3::text[]))
		  FROM orders o
		 WHERE o.id = $1::uuid AND `+col,
		orderID, userID, orders.TerminalStatuses()).Scan(&live); err != nil {
		return false
	}
	return live
}

func hasRole(roles []string, want string) bool {
	for _, r := range roles {
		if r == want {
			return true
		}
	}
	return false
}
