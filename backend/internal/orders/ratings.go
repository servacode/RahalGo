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
	// PlatformStars تقييمُ الخدمة — **للمنصة لا للمتجر**.
	//
	// الزبونُ لا يرى اسمَ المتجر ولا يختاره: يطلب من «رحّال غو» ونحن نختار
	// من أين نشتري. **فنجمةٌ تُنسب إلى متجرٍ لم يعرفه نجمةٌ بلا معنى** — وهو
	// يحكم على طعامٍ ووقتٍ ومعاملة، **وثلاثتُها من عندنا.**
	PlatformStars int       `json:"platform_stars"`
	DriverStars   *int      `json:"driver_stars"`
	Comment       string    `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
}

// RateOrder تقييم مزدوج لطلب مُسلَّم — مرة واحدة، من زبون الطلب نفسه (أو الأدمن).
func (s *Service) RateOrder(ctx context.Context, actorID string, actorRoles []string,
	orderID string, platformStars int, driverStars *int, comment string) error {
	if platformStars < 1 || platformStars > 5 ||
		(driverStars != nil && (*driverStars < 1 || *driverStars > 5)) {
		return ErrBadStars
	}

	var status, customerID string
	var driverID *string
	var repIsBuyer bool
	err := s.db.QueryRow(ctx, `
		SELECT o.status, o.customer_id, o.driver_id,
		       COALESCE(m.sales_rep_user_id = o.customer_id, false)
		-- **ويُضمّ المتجرُ يساراً** — (الطلبُ الخاصّ ٢٠٢٦-٠٨-٠٩): لا متجرَ له.
		--
		-- **وضمٌّ صلبٌ يجعل تقييمَه يردّ «الطلب غير موجود»** — وهو مكتوبٌ في
		-- القاعدة وسُلّم للتوّ. (شهده المالك.)
		FROM orders o LEFT JOIN merchants m ON m.id = o.merchant_id
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
		INSERT INTO order_ratings (order_id, customer_id, platform_stars, driver_stars, comment)
		VALUES ($1, $2, $3, $4, $5)`, orderID, customerID, platformStars, driverStars, comment)
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
		SELECT platform_stars, driver_stars, comment, created_at
		FROM order_ratings WHERE order_id = $1`, orderID).
		Scan(&r.PlatformStars, &r.DriverStars, &r.Comment, &r.CreatedAt)
	if err != nil {
		return nil
	}
	return &r
}
