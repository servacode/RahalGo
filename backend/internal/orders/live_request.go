package orders

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// ══════════════════════════════════════════════════════════════════════
// **«اطلب موقع الزبون المباشر»** (قرارُ المالك ٢٠٢٦-١٠-٢١، هجرة ٠٤٦٩)
// ══════════════════════════════════════════════════════════════════════
//
// **السائقُ يطلبه فيُسأل الزبونُ ثانيةً** — وإن رفض أوّلَ مرّةٍ بالخطأ. **ويصله
// إشعار**، ويُبثّ الطلبُ فيرى تطبيقُه `live_requested_at` الجديد فيُعيد السؤال.
//
// **ثلاثُ مرّاتٍ للطلب ودقيقتان بينها** — في شرطٍ واحدٍ ذرّيٍّ مع الكتابة، فلا
// ضغطتان متزامنتان تتجاوزانه.

const (
	LiveRequestMax = 3
	LiveRequestGap = 2 * time.Minute
)

var (
	ErrLiveRequestLimit = httpx.NewError(http.StatusTooManyRequests,
		"live_request_limit", "errors.rate_limited")
	ErrLiveRequestTooSoon = httpx.NewError(http.StatusTooManyRequests,
		"live_request_too_soon", "errors.rate_limited")
	ErrLiveRequestStage = httpx.NewError(http.StatusConflict,
		"live_request_stage", "errors.conflict")
	errLiveNotYours = httpx.NewError(http.StatusForbidden,
		"not_your_order", "errors.not_your_order")
)

// liveRequestStatuses **ما دام السائقُ في طريقه** — من الإسناد حتّى الباب.
var liveRequestStatuses = []string{"assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff"}

// RequestLiveLocation **السائقُ يطلب موقعَ الزبون المباشر.**
func (s *Service) RequestLiveLocation(ctx context.Context, orderID, driverID string) (*Order, error) {
	gapSec := int(LiveRequestGap.Seconds())
	tag, err := s.db.Exec(ctx, `
		UPDATE orders
		   SET live_requested_at = now(), live_request_count = live_request_count + 1, updated_at = now()
		 WHERE id = $1 AND driver_id = $2::uuid AND status = ANY($3)
		   AND live_request_count < $4
		   AND (live_requested_at IS NULL OR live_requested_at < now() - make_interval(secs => $5))`,
		orderID, driverID, liveRequestStatuses, LiveRequestMax, gapSec)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, s.liveRequestRefusal(ctx, orderID, driverID, gapSec)
	}
	o, err := s.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	s.publishOrder(o)
	if s.notify != nil && o.CustomerID != "" {
		s.notify.Notify(ctx, notifications.Input{
			UserID: o.CustomerID, Kind: notifications.KindOrder,
			Title:  t.liveLocationRequested,
			Body:   fmt.Sprintf("#%d", o.Number),
			Entity: "order", EntityID: o.ID, Href: "/portal/orders",
			Apps: []string{notifications.AppCustomer},
		})
	}
	return o, nil
}

// liveRequestRefusal **لماذا رُدّ** — بسببه لا رفضاً عامّاً.
func (s *Service) liveRequestRefusal(ctx context.Context, orderID, driverID string, gapSec int) error {
	var owner *string
	var status string
	var count int
	var waitSec int
	err := s.db.QueryRow(ctx, `
		SELECT driver_id::text, status, live_request_count,
		       GREATEST(0, CEIL(EXTRACT(EPOCH FROM (live_requested_at + make_interval(secs => $2) - now()))))::int
		  FROM orders WHERE id = $1`, orderID, gapSec).Scan(&owner, &status, &count, &waitSec)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	switch {
	case owner == nil || *owner != driverID:
		return errLiveNotYours
	case count >= LiveRequestMax:
		return ErrLiveRequestLimit
	case waitSec > 0:
		e := *ErrLiveRequestTooSoon
		e.Details = map[string]any{"retry_after_sec": waitSec}
		return &e
	default:
		return ErrLiveRequestStage
	}
}
