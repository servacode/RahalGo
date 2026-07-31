package orders

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	ErrNotDelivered  = httpx.NewError(http.StatusConflict, "not_delivered", "errors.not_delivered")
	ErrAlreadyRated  = httpx.NewError(http.StatusConflict, "already_rated", "errors.already_rated")
	ErrBadStars      = httpx.NewError(http.StatusBadRequest, "invalid_stars", "errors.validation")
	ErrNotYourOrder  = httpx.NewError(http.StatusForbidden, "forbidden", "errors.forbidden")
	ErrRateOwnClient = httpx.NewError(http.StatusForbidden, "rate_own_client", "errors.rate_own_client")
)

type Rating struct {
	MerchantStars int       `json:"merchant_stars"`
	DriverStars   *int      `json:"driver_stars"`
	Comment       string    `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
}

// RateOrder تقييم مزدوج لطلب مُسلَّم — مرة واحدة، من زبون الطلب نفسه (أو الأدمن).
func (s *Service) RateOrder(ctx context.Context, actorID string, actorRoles []string,
	orderID string, merchantStars int, driverStars *int, comment string) error {
	if merchantStars < 1 || merchantStars > 5 ||
		(driverStars != nil && (*driverStars < 1 || *driverStars > 5)) {
		return ErrBadStars
	}

	var status, customerID string
	var driverID *string
	var repIsBuyer bool
	err := s.db.QueryRow(ctx, `
		SELECT o.status, o.customer_id, o.driver_id,
		       m.sales_rep_user_id = o.customer_id
		FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1`, orderID).
		Scan(&status, &customerID, &driverID, &repIsBuyer)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != StDelivered {
		return ErrNotDelivered
	}
	if customerID != actorID && !slices.Contains(actorRoles, "admin") {
		return ErrNotYourOrder
	}
	// المندوب لا يقيّم متجراً هو مندوبه.
	//
	// التقييم شهادةُ زبونٍ مستقلّ، وشهادةُ من ينتفع بنجاح المتجر ليست شهادة.
	// وبلا هذا المنع يستطيع رفع تقييم عملائه بطلباتٍ يشتريها بنفسه — وهو نظير
	// قبضِه عمولةً على شرائه، وقد مُنع للسبب نفسه: **لا مكافأة على ما ليس ترويجاً**.
	// والأدمن مستثنى: تدخّله موثّق في سجل التدقيق ومسؤوليته عليه.
	if repIsBuyer && !slices.Contains(actorRoles, "admin") {
		return ErrRateOwnClient
	}
	if driverID == nil {
		driverStars = nil
	}

	_, err = s.db.Exec(ctx, `
		INSERT INTO order_ratings (order_id, customer_id, merchant_stars, driver_stars, comment)
		VALUES ($1, $2, $3, $4, $5)`, orderID, customerID, merchantStars, driverStars, comment)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrAlreadyRated
	}
	return err
}

// ratingFor يجلب تقييم الطلب إن وُجد (لتفاصيل الطلب).
func (s *Service) ratingFor(ctx context.Context, orderID string) *Rating {
	var r Rating
	err := s.db.QueryRow(ctx, `
		SELECT merchant_stars, driver_stars, comment, created_at
		FROM order_ratings WHERE order_id = $1`, orderID).
		Scan(&r.MerchantStars, &r.DriverStars, &r.Comment, &r.CreatedAt)
	if err != nil {
		return nil
	}
	return &r
}
