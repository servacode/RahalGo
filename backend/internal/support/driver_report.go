package support

// بلاغُ السائق — **بابٌ عنده كما للزبون بابُه.**
//
// # المسألة
//
// الشكوى كانت من طرفٍ واحد: الزبونُ يشكو والسائقُ يُشتكى عليه. **والسائقُ يرى
// ما لا يراه أحد**: متجرٌ يُبقيه واقفاً نصفَ ساعة، وزبونٌ يعطي عنواناً وهمياً،
// وآخرُ يسبّه عند الباب.
//
// **وما لا بابَ له لا يُقال** — فيُقال في مجموعةِ واتساب أو لا يُقال، **ولا
// يدخل رقماً في أيّ تقرير.** ثمّ يُسأل: لماذا يترك السائقون؟
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «ممكن بدّو يقدّم بلاغاً بحقّ متجرٍ أو زبون».)
//
// # وضدَّ من
//
// **كلُّ سببٍ يحمل خصمَه معه** — كما يحمل سببُ التعذّر ذنبَه (`failreasons.go`).
// **وبلاغٌ لا يُعرف على من هو بلاغٌ لا يُتابَع**: تقرأه العملياتُ فلا تعرف
// أتتّصل بالمتجر أم بالزبون.
//
// # ولا يُبلَّغ عن طلبٍ يجري
//
// من طلبُه في يده لا يشكو منه — **يُنهيه**. ولمن تعثّر في الطريق زرُّ الطارئ
// وزرُّ التعذّر، **وهما فعلٌ لا شكوى.**

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// خصومُ البلاغ.
const (
	AgainstMerchant = "merchant"
	AgainstCustomer = "customer"
)

// DriverReportReason سببٌ معرَّف، وعلى من هو.
type DriverReportReason struct {
	Code string `json:"code"`
	// Against المتجرُ أم الزبون — **تقرؤه العملياتُ فتعرف بمن تتّصل.**
	Against string `json:"against"`
}

// DriverReportReasons ما يملك السائقُ اختيارَه.
//
// **والرموزُ من الخادم لا تُكتب في التطبيق** — قائمةٌ في مكانين تفترق حين
// يُضاف سببٌ في أحدهما، فيُرسل التطبيقُ رمزاً لا يعرفه الخادم.
var DriverReportReasons = []DriverReportReason{
	// ── على المتجر ──────────────────────────────────────────────────────
	{Code: "merchant_slow", Against: AgainstMerchant},
	{Code: "merchant_refused", Against: AgainstMerchant},
	{Code: "merchant_wrong_goods", Against: AgainstMerchant},
	{Code: "merchant_conduct", Against: AgainstMerchant},

	// ── على الزبون ──────────────────────────────────────────────────────
	{Code: "customer_absent", Against: AgainstCustomer},
	{Code: "customer_address", Against: AgainstCustomer},
	{Code: "customer_refused", Against: AgainstCustomer},
	{Code: "customer_conduct", Against: AgainstCustomer},

	{Code: "other", Against: ""},
}

// validDriverReason أرمزٌ يعرفه الخادم؟
func validDriverReason(code string) bool {
	for _, r := range DriverReportReasons {
		if r.Code == code {
			return true
		}
	}
	return false
}

// ReasonsForKind **ما يصلح لطلبٍ من هذا النوع.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٣: «أصلحها» — بعد أن قيس أنّ أسبابَ المتجر
//
//	تُعرض في طلبٍ بلا متجر.)
//
// **والطلبُ الخاصُّ بلا متجر** — و«المتجر أوقفني طويلاً» فيه سؤالٌ عمّا
// لا وجودَ له. **ومن اختاره فُتح بلاغٌ بلا مشتكًى عليه**: تذكرةٌ تذهب
// إلى العمليات ولا أحدَ فيها.
//
// **ودالّةٌ واحدةٌ تقرّر** — تقرؤها نقطةُ العرض ونقطةُ الفتح معاً:
// **قائمةٌ تُصفّى في العرض وحدَه لا تمنع من ينادي الواجهةَ مباشرةً**،
// وهي عائلةُ «قاعدةٌ تُطبَّق في الشاشة» التي تكرّرت في هذا المشروع.
func ReasonsForKind(kind string) []DriverReportReason {
	if kind != "custom" {
		return DriverReportReasons
	}
	out := make([]DriverReportReason, 0, len(DriverReportReasons))
	for _, r := range DriverReportReasons {
		if r.Against == AgainstMerchant {
			continue
		}
		out = append(out, r)
	}
	return out
}

// AllowedForKind **أيصلح هذا السببُ لطلبٍ من هذا النوع؟**
func AllowedForKind(kind, code string) bool {
	for _, r := range ReasonsForKind(kind) {
		if r.Code == code {
			return true
		}
	}
	return false
}

// DriverReport يفتح بلاغَ سائقٍ على طلبٍ من طلباته.
//
// **وصاحبُ التذكرة زبونُ الطلب لا السائق** — عمودُ `customer_id` يقول «تذكرةُ
// أيّ طلبٍ هذه» لا «من كتبها»، **ومن كتبها في `created_by`.** ولو وُضع السائقُ
// فيه لَظهر البلاغُ في «شكاواي» عنده وفي ملفّ زبونٍ لا يخصّه.
func (s *Service) DriverReport(ctx context.Context, driverID, orderID, reason, note string) (*Ticket, error) {
	if !validDriverReason(reason) {
		return nil, ErrBadReason
	}

	var number int64
	var customerID, kind string
	var closedAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT o.number, o.customer_id::text, o.kind, o.closed_at
		FROM orders o WHERE o.id = $1 AND o.driver_id = $2`, orderID, driverID).
		Scan(&number, &customerID, &kind, &closedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if closedAt == nil {
		return nil, ErrOrderNotClosed
	}
	// **ولا سببَ متجرٍ في طلبٍ بلا متجر** — والمنعُ هنا لا في الشاشة:
	// **من نادى الواجهةَ مباشرةً لا يوقفه إخفاءُ خيار.**
	if !AllowedForKind(kind, reason) {
		return nil, ErrBadReason
	}

	// **والمهلةُ هي مهلةُ الزبون نفسُها** — لا رقمٌ ثانٍ لمعنًى واحد: الذاكرةُ
	// تُنسى والدليلُ يذهب للطرفين سواء.
	if over, err := s.pastComplaintWindow(ctx, *closedAt); err != nil || over {
		if over {
			return nil, ErrComplaintWindow
		}
		return nil, err
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
	subject := "بلاغُ سائق"
	var ticketID string
	// ══════════════════════════════════════════════════════════════════
	// **وعلى من هو — يُشتقّ من سببه**
	// ══════════════════════════════════════════════════════════════════
	//
	// (شكوى المالك ٢٠٢٦-٠٨-١٣: «أنا كسائقٍ قمتُ بالإبلاغ على زبونٍ
	//  ومتجر، ولكنّ البلاغَ تمّ فهمُه على أنّه ضدّي».)
	//
	// **وكان العمودُ يُترك فارغاً** — وفارغُه في `me/reputation` يعني
	// «تُعرض للجميع»، **فيقرأ السائقُ بلاغَه هو في «الشكاوى عليك».**
	//
	// **ومن رفع بلاغاً فوجده شكوى عليه لا يرفع ثانياً** — وهو أسوأُ
	// ما يقع بباب شكوى: **يُسكِت من فتحه ليتكلّم.**
	//
	// **والجهةُ معروفةٌ في الرمز نفسِه** (`DriverReportReasons`): أربعةٌ
	// على المتجر وأربعةٌ على الزبون. **وكانت مكتوبةً ولا تُقرأ.**
	//
	// **و«سببٌ آخر» يبقى بلا جهة** — لا تُخمَّن على أحد.
	against := s.againstDriverReport(ctx, orderID, reason)
	err = s.db.QueryRow(ctx, `
		INSERT INTO tickets (customer_id, order_id, subject, reason, created_by,
		                     opened_by_customer, against_user_id)
		VALUES ($1, $2, $3, $4, $5, false, $6) RETURNING id`,
		customerID, orderID, subject, reason, driverID, against).Scan(&ticketID)
	// **والفهرسُ الفريدُ هو من يمنع التكرار** — لا فحصٌ قبله يترك ثغرةً بينهما.
	if err != nil && strings.Contains(err.Error(), "tickets_one_open_per_order") {
		return nil, ErrComplaintOpen
	}
	if err != nil {
		return nil, err
	}
	if note = strings.TrimSpace(note); note != "" {
		if _, err := s.db.Exec(ctx, `
			INSERT INTO ticket_replies (ticket_id, author_id, body) VALUES ($1, $2, $3)`,
			ticketID, driverID, note); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, ticketID)
}

// pastComplaintWindow أمضت المهلةُ على هذا الإغلاق؟
//
// **مصدرٌ واحدٌ للمهلة** — كانت مكتوبةً في `Complaint` بيدها، **ونسخةٌ ثانيةٌ
// هنا تفترق يوماً**: تُوسَّع للزبون وتبقى للسائق على القديم.
func (s *Service) pastComplaintWindow(ctx context.Context, closedAt time.Time) (bool, error) {
	hours := int64(24)
	if s.settings != nil {
		if v := s.settings.GetInt(ctx, "support.complaint_window_hours"); v > 0 {
			hours = v
		}
	}
	return time.Since(closedAt) > time.Duration(hours)*time.Hour, nil
}

// againstDriverReport **على من هذا البلاغ** — من رمز سببه.
//
// **ونظيرُ `againstFor` للزبون** — وهما دالّتان لأنّ القائمتين مختلفتان:
// **أسبابُ الزبون على السائق والمتجر، وأسبابُ السائق على المتجر
// والزبون.**
func (s *Service) againstDriverReport(ctx context.Context, orderID, reason string) *string {
	var target string
	for _, r := range DriverReportReasons {
		if r.Code == reason {
			target = r.Against
			break
		}
	}
	if target == "" {
		return nil
	}
	q := `SELECT o.customer_id::text FROM orders o WHERE o.id = $1`
	if target == AgainstMerchant {
		q = `SELECT m.owner_user_id::text FROM orders o
		     JOIN merchants m ON m.id = o.merchant_id WHERE o.id = $1`
	}
	var id *string
	if err := s.db.QueryRow(ctx, q, orderID).Scan(&id); err != nil {
		return nil
	}
	return id
}
