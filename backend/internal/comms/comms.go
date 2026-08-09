// Package comms **التواصلُ بين طرفَي الطلب — بلا رقمٍ يعرفه أحدُهما.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «لازم الاثنان لا يقدران يوصلان لبعض إلّا عن طريق
//
//	المنصّة فقط».)
//
// # القاعدةُ الواحدة
//
// **لا يُطلب التواصلُ بمعرّف مستخدم، إنّما بمعرّف طلب.** والخادمُ يستخرج
// الطرفَ الآخر. **فلا يستطيع أحدٌ أن يبلغ من يشاء** بأن يبدّل رقماً في نداء.
//
// **وهي أهمُّ سطرٍ في هذه الحزمة** — وما يُخطئ فيه أكثرُ من بنى نظاماً كهذا:
// `POST /call/user/52` بابٌ مفتوحٌ على كلّ مستخدمٍ في المنصة.
//
// # والصلاحيةُ تُشتقّ ولا تُخزَّن
//
// **صفٌّ يَنسخ حالةً ينحرف عنها**: يُلغى الطلبُ ولا يُحدَّث الصفّ، **فتبقى
// الصلاحيةُ مفتوحةً على طلبٍ ميّت** — ولا يظهر في أيّ خطأ. ويحتاج كنّاساً
// دوريّاً، **وكنّاسٌ يتوقّف يترك أبواباً مفتوحة.**
//
// **والاشتقاقُ لا ينحرف**: الطرفان وحالةُ الطلب ووقتُ تسليمه كلُّها في
// `orders`. **والسؤالُ يُسأل لحظةَ الاستعمال فيُجاب بالحقيقة.**
//
// # ولمَ لا يخرس الزبونُ حتّى يتكلّم السائق
//
// **اقترحت الوثيقةُ أن يُمنع الزبونُ حتّى يتّصل السائقُ أوّلاً** — حمايةً
// للسائق.
//
// **وأكثرُ الحاجات شيوعاً عكسُها**: «أنا نازل لا تصعد» · «غيّرتُ العنوان» ·
// «اتركه عند البوّاب». **وفي تلك القاعدة هو أخرسُ حتّى يقرّر السائقُ أن
// يتكلّم** — وقد لا يتكلّم أبداً: يمشي على الخريطة ويقرع الباب.
//
// **وثغرتُها أنّ من لا تصله الرنّةُ لا ينال حقَّ الردّ**: هاتفٌ مغلقٌ أو بلا
// إنترنت **لا يبلغ `ringing`** — وهو بعينه من سيحتاج أن يعاود حين يفتح هاتفه.
//
// **فيُفتح للطرفين من الإسناد، ويُحَدّ المعدَّل بدلَ المنع.** تُنال الحمايةُ
// بلا خرس.
//
// # وقناتان على صلاحيةٍ واحدة
//
// **الرسائلُ اليوم والصوتُ مع التطبيق** — وكلاهما يسأل `Permit` نفسَها. ولو
// كان لكلٍّ منهما بابُه لَاختلفا يوماً: **تُغلق قناةٌ عند الإلغاء وتبقى
// الأخرى.**
package comms

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	// ErrNotParty **لستَ طرفاً في هذا الطلب.**
	//
	// **ويردّ ٤٠٤ لا ٤٠٣**: «ممنوع» تؤكّد أنّ الطلبَ موجود، **فتصير أداةَ
	// استكشاف** لمن يجرّب المعرّفات. ومن ليس طرفاً لا يعنيه وجودُه.
	ErrNotParty = httpx.NewError(http.StatusNotFound, "not_found", "errors.not_found")

	// ErrChannelClosed **القناةُ أُغلقت** — طلبٌ انتهى أو أُلغي.
	ErrChannelClosed = httpx.NewError(http.StatusConflict,
		"comms_closed", "errors.comms_closed")

	// ErrNoDriverYet **لا سائقَ بعد** — فلا طرفَ ثانيَ للحديث.
	ErrNoDriverYet = httpx.NewError(http.StatusConflict,
		"comms_no_driver", "errors.comms_no_driver")

	// ErrEmptyBody **رسالةٌ بلا نصّ** — خطأُ تحقّقٍ لا قناةٌ مغلقة.
	ErrEmptyBody = httpx.NewError(http.StatusBadRequest,
		"validation", "errors.validation")

	// ErrTooFast **حدُّ المعدّل** — بديلُ المنع.
	ErrTooFast = httpx.NewError(http.StatusTooManyRequests,
		"rate_limited", "errors.rate_limited")
)

// Role طرفُ الطلب — **اثنان لا ثالثَ لهما في هذه القناة.**
//
// **والإدارةُ ليست طرفاً**: هي ترى ولا تشارك. **وقناةٌ ثلاثيّةٌ تجعل الطرفين
// يكتبان لجمهور** لا لبعضهما.
type Role string

const (
	RoleCustomer Role = "customer"
	RoleDriver   Role = "driver"
)

// **ولا مهلةَ بعد الانتهاء — تُغلق لحظتَه.**
//
// (تصحيحُ المالك ٢٠٢٦-٠٨-٠٩: «والمحادثات تُغلق بعد تسليم الطلب بشكلٍ إلزاميّ،
//
//	لا تبقى أيُّ دردشةٍ مفتوحة — لا بطلبٍ خاصٍّ ولا عاديّ. مجرّد تسليم الطلب
//	أو انتهاء الطلب تُغلق الدردشة».)
//
// # وكانت ربعَ ساعة
//
// **بُنيت على وثيقة المالك الأولى** — «من اكتشف نقصاً يكتشفه وهو يفتح الكيس».
// **وقرّر المالكُ غيرَها**: القناةُ للطلب القائم، **وما بعد التسليم بابُه
// الشكوى** — تُقرأ وتُحقَّق ويُعوَّض، لا محادثةٌ بين طرفين بلا شاهد.
//
// **والقراءةُ تبقى مفتوحة**: الحديثُ يُقرأ بعد الإغلاق ولا يُكتب — **حديثٌ
// يختفي بانتهاء الطلب يمحو ما يُحتجّ به.**

// Permission **مَن يستطيع أن يبلغ مَن، ولماذا لا.**
type Permission struct {
	// OrderID الطلبُ الذي تُنسب إليه القناة.
	OrderID string
	// Me دورُ السائل في هذا الطلب.
	Me Role
	// SelfID معرّفُ السائل — **يُثبَّت هنا بعد التحقّق منه.**
	//
	// **ولا يُعاد تمريرُه من المنادي بعد ذلك**: من مرّره مرّةً في النداء
	// **قد يمرّر غيرَه في النداء الثاني** — والصلاحيةُ تُبنى مرّةً وتُقرأ
	// مراراً، فتحمل ما تحقّقت منه.
	SelfID string
	// PeerID الطرفُ الآخر — **يُستخرج ولا يُرسَل من العميل.**
	PeerID string
	// PeerName اسمُه — **ولا رقمَ معه أبداً.**
	PeerName string
	// Open أمفتوحةٌ القناةُ الآن.
	Open bool
	// ClosesAt متى تُغلق — للعرض، `nil` إن لم تُحدَّد بعد.
	ClosesAt *time.Time
}

// Service قارئُ الصلاحية وحاملُ الرسائل.
type Service struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *Service { return &Service{db: db} }

// openStatuses **الحالاتُ التي يجوز فيها التواصل.**
//
// **وهي حالاتُ رحّال غو لا حالاتُ الوثيقة**: كُتب فيها `DRIVER_ASSIGNED` و
// `DELIVERING`، **ولا وجودَ لهما هنا.** واسمٌ يُنقل حرفيّاً من ورقةٍ إلى شيفرة
// يصير شرطاً لا يتحقّق أبداً.
//
// **و`delivered` داخلةٌ بمهلتها** — تُفحص بالوقت لا بالحالة.
var openStatuses = map[string]bool{
	"assigned": true, "at_pickup": true, "picked_up": true,
	"on_the_way": true, "at_dropoff": true,
}

// Permit **يقرأ الصلاحيةَ لهذا الطلب ولهذا السائل.**
//
// **ولا يأخذ دوراً من المنادي**: يُستخرج من الطلب نفسِه. **فمن ادّعى دوراً
// لا يملكه لا يُصدَّق** — والادّعاءُ هو أوّلُ ما يُجرَّب.
func (s *Service) Permit(ctx context.Context, orderID, userID string) (*Permission, error) {
	var (
		customerID               string
		driverID                 *string
		status                   string
		deliveredAt, closedAt    *time.Time
		customerName, driverName string
	)
	err := s.db.QueryRow(ctx, `
		SELECT o.customer_id::text, o.driver_id::text, o.status,
		       o.delivered_at, o.closed_at,
		       COALESCE(cu.full_name, ''), COALESCE(dr.full_name, '')
		FROM orders o
		JOIN users cu ON cu.id = o.customer_id
		LEFT JOIN users dr ON dr.id = o.driver_id
		WHERE o.id = $1`, orderID).
		Scan(&customerID, &driverID, &status, &deliveredAt, &closedAt,
			&customerName, &driverName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotParty
	}
	if err != nil {
		return nil, err
	}

	p := &Permission{OrderID: orderID, SelfID: userID}
	switch {
	case userID == customerID:
		p.Me = RoleCustomer
		if driverID == nil {
			// **ولا سائقَ بعد** — القناةُ قائمةٌ ولا طرفَ لها.
			return p, ErrNoDriverYet
		}
		p.PeerID, p.PeerName = *driverID, driverName
	case driverID != nil && userID == *driverID:
		p.Me = RoleDriver
		p.PeerID, p.PeerName = customerID, customerName
	default:
		// **ولا الإدارةُ طرف** — ترى السجلَّ من بابها لا من هذا.
		return nil, ErrNotParty
	}

	p.Open, p.ClosesAt = channelOpen(status, deliveredAt, closedAt)
	return p, nil
}

// channelOpen **الحكمُ وحدَه — مفصولاً عن القاعدة ليُختبر.**
//
// **ودالّةٌ تُنادي القاعدةَ لا يبلغها اختبار** — والحكمُ هو ما يُخطئ فيه:
// طلبٌ مُلغًى تبقى قناتُه، أو مُسلَّمٌ تُغلق لحظتَه.
func channelOpen(status string, deliveredAt, closedAt *time.Time) (bool, *time.Time) {
	if openStatuses[status] {
		return true, nil
	}
	// **وكلُّ ما عدا مراحلِ الطريق مغلق** — مُسلَّمٌ ومُلغًى ومرفوضٌ ومتعذّرٌ
	// ومسترجَع.
	//
	// **ولا فرقَ بين نهايةٍ ونهاية**: انتهى الطلبُ فانتهت قناتُه.
	//
	// **وما بعد التسليم بابُه الشكوى** — تُقرأ وتُحقَّق ويُعوَّض، **ومحادثةٌ
	// بلا شاهدٍ تصير كلمتَه ضدّ كلمته.**
	_, _ = deliveredAt, closedAt
	return false, nil
}
