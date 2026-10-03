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
	{"GET", "/api/v1/driver/orders/", "", "driver"},
	// ══════════════════════════════════════════════════════════════════
	// **وما يلزم لإتمامه فعلاً** (٢٠٢٦-١٠-٠٢)
	// ══════════════════════════════════════════════════════════════════
	//
	// **قِيس**: الانتقالُ والإثباتُ وحدَهما مأذونان — **فالمعلَّقُ لا يرى
	// طريقَه ولا يكلّم زبونَه ولا يملك «لدي مشكلة» ولا الطارئ ولا إعادةَ
	// الطلب.** فيُحمَل على إكمالِ ما لا يستطيع أن يراه.
	{"GET", "/api/v1/driver/orders/", "/route", "driver"},
	{"POST", "/api/v1/driver/orders/", "/road-correlation", "driver"},
	{"POST", "/api/v1/driver/orders/", "/emergency", "driver"},
	{"POST", "/api/v1/driver/orders/", "/report", "driver"},
	{"POST", "/api/v1/driver/orders/", "/release", "driver"},
	// **وحديثُ طلبه** — الطريقُ الوحيدُ إلى زبونه (لا رقمَ معه).
	{"GET", "/api/v1/orders/", "/messages", "driver"},
	{"POST", "/api/v1/orders/", "/messages", "driver"},

	// **المتجر** — يقبل أو يرفض ما بين يديه.
	{"POST", "/api/v1/merchant/orders/", "/transition", "merchant"},
	{"GET", "/api/v1/merchant/orders/", "", "merchant"},

	// **الزبون** — يرى طلبَه القائمَ ويستطيع إلغاءه.
	//
	// **ودفع مالَه** — **فحجبُ رؤيته عنه عقوبةٌ على مالٍ لا على فعل.**
	//
	// **والرؤيةُ عبر المسار الحقيقيّ** (`GET /api/v1/my/orders/{id}` — `handleMyOrder`)
	// **لا `GET /api/v1/orders/{id}` الذي لا وجودَ له** (كان `CAF-04`: استثناءٌ يشير
	// إلى مسارٍ غير مسجَّل، فيبقى الطلبُ غيرَ مرئيٍّ لصاحبه الموقوف). **والإلغاءُ
	// مسارُه `POST /api/v1/orders/{id}/cancel` فيبقى كما هو.** وكلاهما محروسٌ
	// بـ`isLiveParticipant` (صاحبُه + حيّ).
	{"GET", "/api/v1/my/orders/", "", "customer"},
	{"POST", "/api/v1/orders/", "/cancel", "customer"},
}

// holdingRoutes **أبوابٌ بلا معرّفِ طلب — تُفتح لسائقٍ معلَّقٍ ما دام يحمل
// طلباً حيّاً** (٢٠٢٦-١٠-٠٢).
//
// **وكلُّها قراءةٌ أو موضع**: قائمةُ طلباته (والتطبيقُ لا يرى رحلتَه بلاها)،
// وحالُه (ليقول له «حسابك موقوف»)، وموضعُه (العملياتُ والزبونُ يتبعانه)،
// وأسبابُ «لدي مشكلة»، وقائمةُ أحاديثه. **ولا ورديّةَ ولا قبولَ ولا طابور** — الطابورُ
// عملٌ جديد، **وجلسةُ الموقوف لا تصير إذناً عامّاً** (`XG39 S4`). والتطبيقُ يقرأ ردَّه
// قائمةً فارغةً فلا تسقط قراءتُه كلُّها (`OrdersLoader`).
//
// **وتُغلق بانتهاء آخر طلب** — فيعود معلَّقاً كسائر المعلَّقين.
var holdingRoutes = []struct{ Method, Path string }{
	{"GET", "/api/v1/driver/orders"},
	{"GET", "/api/v1/driver/me"},
	{"POST", "/api/v1/driver/location"},
	{"POST", "/api/v1/driver/location/batch"},
	{"GET", "/api/v1/driver/fail-reasons"},
	{"GET", "/api/v1/driver/orders/report-reasons"},
	{"GET", "/api/v1/my/chats"},
}

// holdsLiveOrder **أيحمل هذا السائقُ طلباً لم يُغلق؟**
func (s *Server) holdsLiveOrder(ctx context.Context, driverID string) bool {
	var holds bool
	// **ومشوارُ إرجاعٍ لم يُسلَّم حملٌ كذلك** (قرارُ المالك ٢٠٢٦-١٠-٠٣: «تسليمُ البضاعة لمصلحة
	// المنصّة») — الطلبُ أُغلق والبضاعةُ ما زالت معه.
	if err := s.pg.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM orders WHERE driver_id = $1::uuid
		               AND (closed_at IS NULL OR (return_to IS NOT NULL AND goods_handed_at IS NULL)))`,
		driverID).Scan(&holds); err != nil {
		return false
	}
	return holds
}

// suspendedMayContinue أيسمح لهذا النداءِ من معلَّق؟
//
// **ويُرجع `false` عند أدنى شكّ** — **ومن شكّ فمنع أخطأ في الأمان،
// ومن شكّ فأذِن أخطأ في المال.**
func (s *Server) suspendedMayContinue(ctx context.Context, r *http.Request,
	userID string, roles []string) bool {
	if hasRole(roles, "driver") {
		for _, h := range holdingRoutes {
			if r.Method == h.Method && r.URL.Path == h.Path {
				return s.holdsLiveOrder(ctx, userID)
			}
		}
	}
	// **والموقوفُ يُرجع البضاعةَ ويرى طريقَه إليها** (قرارُ المالك ٢٠٢٦-١٠-٠٣) — كان «سلّمت البضاعة»
	// مقفولاً عليه لأنّ الطلبَ مغلق، **فتبقى البضاعةُ معه ولا بابَ يُسلّمها منه.**
	if hasRole(roles, "driver") && r.Method != "" {
		for _, suf := range []string{"/goods-handed", "/route"} {
			rest, ok := strings.CutPrefix(r.URL.Path, "/api/v1/driver/orders/")
			if !ok {
				break
			}
			if id, ok := strings.CutSuffix(rest, suf); ok && isUUID(id) && !strings.Contains(id, "/") {
				if s.pendingReturn(ctx, id, userID) {
					return true
				}
			}
		}
	}
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

// pendingReturn **أعليه مشوارُ إرجاعٍ لهذا الطلب لم يُسلَّم بعد؟**
func (s *Server) pendingReturn(ctx context.Context, orderID, driverID string) bool {
	var pending bool
	if err := s.pg.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM orders WHERE id = $1::uuid AND driver_id = $2::uuid
		               AND return_to IS NOT NULL AND goods_handed_at IS NULL)`,
		orderID, driverID).Scan(&pending); err != nil {
		return false
	}
	return pending
}
