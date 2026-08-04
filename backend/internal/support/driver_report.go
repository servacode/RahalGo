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
	"strconv"
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
	var customerID string
	var closedAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT o.number, o.customer_id::text, o.closed_at
		FROM orders o WHERE o.id = $1 AND o.driver_id = $2`, orderID, driverID).
		Scan(&number, &customerID, &closedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if closedAt == nil {
		return nil, ErrOrderNotClosed
	}

	// **والمهلةُ هي مهلةُ الزبون نفسُها** — لا رقمٌ ثانٍ لمعنًى واحد: الذاكرةُ
	// تُنسى والدليلُ يذهب للطرفين سواء.
	if over, err := s.pastComplaintWindow(ctx, *closedAt); err != nil || over {
		if over {
			return nil, ErrComplaintWindow
		}
		return nil, err
	}

	subject := "بلاغُ سائقٍ على الطلب #" + strconv.FormatInt(number, 10)
	var ticketID string
	err = s.db.QueryRow(ctx, `
		INSERT INTO tickets (customer_id, order_id, subject, reason, created_by, opened_by_customer)
		VALUES ($1, $2, $3, $4, $5, false) RETURNING id`,
		customerID, orderID, subject, reason, driverID).Scan(&ticketID)
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
