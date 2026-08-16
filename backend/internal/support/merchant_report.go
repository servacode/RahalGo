package support

/*
**والمتجرُ يشتكي أيضاً — على السائق.**

(قرارُ المالك ٢٠٢٦-٠٨-١٦: «الشكاوى هي القادمةُ للمنصّة من تاجرٍ من مندوبٍ من
 زبونٍ من سائق أو عليهم» — وكشف الفحصُ أنّ التاجرَ لا بابَ له.)

# ما كان

**ثلاثةُ أبوابٍ تكتب في جدول التذاكر**: الزبونُ والسائقُ والمكتبُ نيابةً. **ولا
بابَ لصاحب المتجر** — فمن أخّره سائقٌ أو رفض أخذَ طلبٍ جاهزٍ أو أساء **لا يملك
أن يقول.**

**وطرفٌ يُشتكى عليه ولا يشتكي طرفٌ ناقص**: المتجرُ يُنذَر ويُحظَر بعدّاد
مخالفاتٍ، **ولا يُسمع منه.**

# ولماذا على السائق وحدَه

**والزبونُ ليس طرفَ المتجر في هذه المنصّة**: لا يعرف اسمَه ولا يختاره، **والمنصّةُ
هي التي تسلّم.** فمن أساء إليه فالسائقُ، ومن تأخّر فالسائق.

**والشكوى على المنصّة نفسِها بابُها الدعمُ لا هذا** — وهي حوارٌ لا بلاغ.

# والمهلةُ هي مهلةُ الجميع

**رقمٌ واحدٌ لمعنًى واحد**: الذاكرةُ تُنسى والدليلُ يذهب للأطراف الثلاثة سواء.
*/

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// MerchantReportReasons **ما يملك المتجرُ الإبلاغَ عنه.**
//
// **وكلُّها على السائق** — فلا حاجةَ لعمود جهةٍ كما في بلاغ السائق: **جهةٌ
// واحدةٌ لا تُسأل.**
//
// (سمّاها المالكُ ٢٠٢٦-٠٨-١٦.)
var MerchantReportReasons = []string{
	// **الطلبُ جاهزٌ والسائقُ لم يأتِ** — وهو أكثرُ ما يشكوه مطعم.
	"driver_late_pickup",
	// **رفض أخذَ الطلب** بلا سبب.
	"driver_refused",
	"driver_conduct",
	// **و«سببٌ آخر» يبقى بلا جهةٍ مُشتقّة** — لا تُخمَّن على أحد.
	"other",
}

// ErrNotMerchantOrder **طلبٌ ليس من متاجره.**
var ErrNotMerchantOrder = httpx.NewError(404, "not_found", "errors.not_found")

func validMerchantReason(code string) bool {
	for _, c := range MerchantReportReasons {
		if c == code {
			return true
		}
	}
	return false
}

// MerchantReport **يفتح بلاغَ متجرٍ على سائقِ طلبٍ من طلباته.**
//
// **وصاحبُ التذكرة زبونُ الطلب لا المتجر** — عمودُ `customer_id` يقول «تذكرةُ
// أيّ طلبٍ هذه» لا «من كتبها»، **ومن كتبها في `created_by`.** (وهو العرفُ
// نفسُه في بلاغ السائق.)
func (s *Service) MerchantReport(ctx context.Context, ownerID, orderID, reason, note string) (*Ticket, error) {
	if !validMerchantReason(reason) {
		return nil, ErrBadReason
	}

	var customerID string
	var driverID *string
	var closedAt *time.Time
	// **والطلبُ من متاجره هو** — والشرطُ على المالك لا على المتجر:
	// **من ملك متجرين لا يُبلّغ عن طلبِ ثالثٍ ليس له.**
	err := s.db.QueryRow(ctx, `
		SELECT o.customer_id::text, o.driver_id::text, o.closed_at
		FROM orders o
		JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1 AND m.owner_user_id = $2`, orderID, ownerID).
		Scan(&customerID, &driverID, &closedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotMerchantOrder
	}
	if err != nil {
		return nil, err
	}
	// **ولا بلاغَ على سائقٍ لم يُسنَد** — **وشكوى بلا مشكوٍّ عليه تقف عند
	// المكتب بلا طرفٍ يُسأل.**
	if driverID == nil {
		return nil, ErrBadReason
	}
	// **ولا يُبلَّغ عن طلبٍ لم ينتهِ** — يُغلق أوّلاً ثمّ يُحكى عنه، **وإلّا
	// رُدّ على ما لم يقع بعد.**
	if closedAt == nil {
		return nil, ErrOrderNotClosed
	}
	if over, err := s.pastComplaintWindow(ctx, *closedAt); err != nil || over {
		if over {
			return nil, ErrComplaintWindow
		}
		return nil, err
	}

	// **ولا رقمَ في العنوان** — الطلبُ خانةٌ قائمةٌ بذاتها.
	var ticketID string
	err = s.db.QueryRow(ctx, `
		INSERT INTO tickets (customer_id, order_id, subject, reason, created_by,
		                     opened_by_customer, against_user_id)
		VALUES ($1, $2, 'بلاغُ متجر', $3, $4, false, $5) RETURNING id`,
		customerID, orderID, reason, ownerID, driverID).Scan(&ticketID)
	// **والفهرسُ الفريدُ يمنع التكرار** — لا فحصٌ قبله يترك ثغرةً بينهما.
	if err != nil && strings.Contains(err.Error(), "tickets_one_open_per_order") {
		return nil, ErrComplaintOpen
	}
	if err != nil {
		return nil, err
	}
	if note = strings.TrimSpace(note); note != "" {
		if _, err := s.db.Exec(ctx, `
			INSERT INTO ticket_replies (ticket_id, author_id, body) VALUES ($1, $2, $3)`,
			ticketID, ownerID, note); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, ticketID)
}
