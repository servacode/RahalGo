package support

// شكوى الزبون على طلبه — **بابٌ عنده لا رقمُ هاتفٍ يتّصل به.**
//
// # المسألة
//
// التذاكرُ كانت تُفتح من لوحة الإدارة وحدَها: يتّصل الزبونُ بالمكتب، فيفتح
// موظّفٌ تذكرةً بالنيابة عنه. **فمن لم يجد من يردّ عليه لم يجد بابَ شكوى
// أصلاً** — وشكواه تضيع، ونحن لا نعلم أنّها وقعت.
//
// **والصمتُ يُقرأ رضاً وهو ليس رضاً**: عشرُ شكاوى لم تُقل تبدو في تقاريرنا
// عشرَ طلباتٍ ناجحة.
//
// # ولماذا سببٌ مصنَّفٌ لا نصٌّ حرّ
//
// **للعلّةِ نفسِها التي صُنّف بها سببُ تعذّر التسليم** (`failreasons.go`):
// نصٌّ حرٌّ لا يُعدّ ولا يُقاس. لا يُعرف كم مرّةً قيل «لم يصلني طلبي» هذا
// الشهر، ولا أيُّ سائقٍ يتكرّر معه ذلك — **والتكرارُ هو الخبر، لا الحادثةُ
// الواحدة.**
//
// والنصُّ الحرُّ يبقى بجانب السبب: **القائمةُ تُصنّف والنصُّ يشرح.**

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	ErrComplaintWindow = httpx.NewError(http.StatusConflict, "complaint_window_passed", "errors.complaint_window_passed")
	ErrComplaintOpen   = httpx.NewError(http.StatusConflict, "complaint_already_open", "errors.complaint_already_open")
	ErrBadReason       = httpx.NewError(http.StatusBadRequest, "bad_complaint_reason", "errors.bad_complaint_reason")
	ErrOrderNotClosed  = httpx.NewError(http.StatusConflict, "order_not_closed", "errors.order_not_closed")
)

// ComplaintReason سببٌ معرَّف، وعلى أيّ الطلبات يُعرض.
type ComplaintReason struct {
	Code string `json:"code"`
	// Delivered هل يُعرض على الطلبات المسلَّمة فقط.
	//
	// **«لم يصلني طلبي» على طلبٍ لم يُسلَّم لغو**: حالتُه تقول ذلك أصلاً،
	// **وسؤالُ المستخدم عمّا نعرفه يجعله يشكّ فيما نعرف.**
	Delivered bool `json:"delivered_only"`
}

// ComplaintReasons ما يملك الزبونُ اختيارَه.
//
// **والرموزُ من الخادم لا تُكتب في التطبيق**: قائمةٌ تُكرَّر في مكانين تفترق
// حين يُضاف سببٌ في أحدهما.
var ComplaintReasons = []ComplaintReason{
	// **أخطرُها وأوّلُها.**
	//
	// طلبٌ حالتُه «مُسلَّم» ولم يصل يعني أحدَ أمرين: **سائقٌ ضغط الزرّ ولم
	// يسلّم، أو تسليمٌ إلى غير صاحبه.** وكلاهما مالٌ خرج من الزبون بلا مقابل،
	// **ولا يُكتشف إلّا إن قاله هو** — لا قيدَ في دفترنا يشي به.
	{Code: "not_received", Delivered: true},
	{Code: "missing_items", Delivered: true},
	{Code: "wrong_items", Delivered: true},
	{Code: "quality", Delivered: true},
	{Code: "late", Delivered: true},
	// **يُعرض على غير المسلَّم أيضاً**: من أُلغي طلبُه بلا سببٍ يفهمه يشكو،
	// ومن تعذّر تسليمُه وهو في بيته يشكو.
	{Code: "driver_conduct"},
	{Code: "money", Delivered: false},
	{Code: "other"},
}

// ReasonsFor ما يُعرض على طلبٍ بحالته.
func ReasonsFor(status string) []ComplaintReason {
	out := []ComplaintReason{}
	for _, r := range ComplaintReasons {
		if r.Delivered && status != "delivered" {
			continue
		}
		out = append(out, r)
	}
	return out
}

func validReason(code, status string) bool {
	for _, r := range ReasonsFor(status) {
		if r.Code == code {
			return true
		}
	}
	return false
}

// Complaint شكوى الزبون على طلبه — تُفتح تذكرةً كسائر التذاكر.
//
// **تذكرةٌ لا نوعٌ ثانٍ من السجلّات**: للدعم مكانٌ واحدٌ ينظر فيه، **ومكتبٌ
// ينظر في مكانين ينسى أحدَهما.**
func (s *Service) Complaint(ctx context.Context, customerID, orderID, reason, note string) (*Ticket, error) {
	var status, subject string
	var number int64
	var closedAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT o.status, o.number, o.closed_at
		FROM orders o WHERE o.id = $1 AND o.customer_id = $2`, orderID, customerID).
		Scan(&status, &number, &closedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// **لا شكوى على طلبٍ يجري.**
	//
	// من طلبُه في الطريق لا يشكو منه — **يتابعه**. وبابُ الشكوى قبل النهاية
	// يجعل الدعمَ يردّ على ما لم يقع بعد.
	if closedAt == nil {
		return nil, ErrOrderNotClosed
	}
	if !validReason(reason, status) {
		return nil, ErrBadReason
	}

	// **مهلةٌ لأن الذاكرةَ تُنسى والدليلَ يذهب.**
	//
	// شكوى بعد شهرٍ لا تُحقَّق: السائقُ لا يذكر، والبضاعةُ ذهبت، **ولا يبقى
	// إلّا كلمةٌ ضدّ كلمة** — فتُقبل بلا بيّنة أو تُردّ بلا بيّنة، وكلاهما
	// ظلمٌ لأحدهما. والمهلةُ في الإعدادات: يملك المالكُ توسيعَها.
	if over, err := s.pastComplaintWindow(ctx, *closedAt); err != nil {
		return nil, err
	} else if over {
		return nil, ErrComplaintWindow
	}

	// **ولا رقمَ في العنوان — الطلبُ خانةٌ قائمةٌ بذاتها.**
	//
	// (شهده المالك ٢٠٢٦-٠٨-٠٧: «مكرّرة بلا فائدة».)
	//
	// **كان العنوانُ يحمل «على الطلب ‎#1002» والصفُّ يحمل خانةَ «الطلب
	// ‎#1002» بجانبه** — الرقمُ نفسُه مرّتين في سطرٍ واحد، **ونصفُ عرض
	// الشاشة يقول ما يقوله ربعُها.**
	//
	// **والعنوانُ نصٌّ محفوظٌ في قاعدة البيانات** — فلا يتبع تبديلاً في
	// الشاشة. **فيُكتب ما لا يوجد في خانةٍ أخرى، ولا يُكرَّر ما يوجد.**
	subject = "شكوى على طلب"
	var ticketID string
	err = s.db.QueryRow(ctx, `
		INSERT INTO tickets (customer_id, order_id, subject, reason, created_by, opened_by_customer)
		VALUES ($1, $2, $3, $4, $1, true) RETURNING id`,
		customerID, orderID, subject, reason).Scan(&ticketID)
	// **الفهرسُ الفريدُ هو من يمنع التكرار لا فحصٌ قبله.**
	//
	// فحصٌ ثمّ إدراجٌ يترك بينهما ثغرةً: ضغطتان متزامنتان تمرّان كلتاهما.
	// **والقاعدةُ تحكم حيث لا يحكم الترتيب.**
	if err != nil && strings.Contains(err.Error(), "tickets_one_open_per_order") {
		return nil, ErrComplaintOpen
	}
	if err != nil {
		return nil, err
	}
	if note = strings.TrimSpace(note); note != "" {
		if _, err := s.db.Exec(ctx, `
			INSERT INTO ticket_replies (ticket_id, author_id, body) VALUES ($1, $2, $3)`,
			ticketID, customerID, note); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, ticketID)
}
