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
	ErrNotDelivered = httpx.NewError(http.StatusConflict, "not_delivered", "errors.not_delivered")
	ErrAlreadyRated = httpx.NewError(http.StatusConflict, "already_rated", "errors.already_rated")
	ErrBadStars     = httpx.NewError(http.StatusBadRequest, "invalid_stars", "errors.validation")
	ErrNotYourOrder = httpx.NewError(http.StatusForbidden, "forbidden", "errors.forbidden")
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
	err := s.db.QueryRow(ctx,
		`SELECT status, customer_id, driver_id FROM orders WHERE id = $1`, orderID).
		Scan(&status, &customerID, &driverID)
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
