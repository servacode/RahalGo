// Package catalog الفئات الديناميكية والمتاجر.
package catalog

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/media"
)

var (
	ErrCategoryInvalid = httpx.NewError(http.StatusBadRequest, "invalid_category", "errors.invalid_category")
	ErrNameRequired    = httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
)

type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	SortOrder int    `json:"sort_order"`
	Active    bool   `json:"active"`
}

type Merchant struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	CategoryID      string    `json:"category_id"`
	CategoryName    string    `json:"category_name"`
	CategoryIcon    string    `json:"category_icon"`
	Phone           string    `json:"phone"`
	AddressText     string    `json:"address_text"`
	OwnerUserID     *string   `json:"owner_user_id"`
	OwnerPhone      *string   `json:"owner_phone"`
	SalesRepPhone   *string   `json:"sales_rep_phone"`
	SalesRepCode    *string   `json:"sales_rep_code"`
	Lat             *float64  `json:"lat"`
	Lng             *float64  `json:"lng"`
	LogoURL         *string   `json:"logo_url"`
	LogoThumbURL    *string   `json:"logo_thumb_url"`
	Status          string    `json:"status"`
	CommissionPct   int       `json:"commission_percent"`
	EmergencyClosed bool      `json:"emergency_closed"`
	CreatedAt       time.Time `json:"created_at"`
}

type MerchantPage struct {
	Merchants []Merchant `json:"merchants"`
	Total     int        `json:"total"`
	Page      int        `json:"page"`
	PerPage   int        `json:"per_page"`
}

type Service struct {
	db       *pgxpool.Pool
	identity *identity.Service
}

func NewService(db *pgxpool.Pool, identitySvc *identity.Service) *Service {
	return &Service{db: db, identity: identitySvc}
}

// ---------- الفئات ----------

func (s *Service) ListCategories(ctx context.Context, activeOnly bool) ([]Category, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, icon, sort_order, active FROM categories
		WHERE NOT $1 OR active
		ORDER BY sort_order, name`, activeOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Icon, &c.SortOrder, &c.Active); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type CategoryInput struct {
	Name      *string `json:"name"`
	Icon      *string `json:"icon"`
	SortOrder *int    `json:"sort_order"`
	Active    *bool   `json:"active"`
}

func (s *Service) CreateCategory(ctx context.Context, actorID string, in CategoryInput, ip string) (*Category, error) {
	if in.Name == nil || *in.Name == "" {
		return nil, ErrNameRequired
	}
	var c Category
	err := s.db.QueryRow(ctx, `
		INSERT INTO categories (name, icon, sort_order)
		VALUES ($1, COALESCE($2, ''), COALESCE($3, 0))
		RETURNING id, name, icon, sort_order, active`,
		*in.Name, in.Icon, in.SortOrder).
		Scan(&c.ID, &c.Name, &c.Icon, &c.SortOrder, &c.Active)
	if err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "admin.category_create", "category", c.ID, ip)
	return &c, nil
}

func (s *Service) UpdateCategory(ctx context.Context, actorID, id string, in CategoryInput, ip string) (*Category, error) {
	var c Category
	err := s.db.QueryRow(ctx, `
		UPDATE categories SET
			name       = COALESCE($2, name),
			icon       = COALESCE($3, icon),
			sort_order = COALESCE($4, sort_order),
			active     = COALESCE($5, active)
		WHERE id = $1
		RETURNING id, name, icon, sort_order, active`,
		id, in.Name, in.Icon, in.SortOrder, in.Active).
		Scan(&c.ID, &c.Name, &c.Icon, &c.SortOrder, &c.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "admin.category_update", "category", id, ip)
	return &c, nil
}

// ---------- المتاجر ----------

const merchantSelect = `
	SELECT m.id, m.name, m.description, m.category_id, c.name, c.icon,
	       m.phone, m.address_text, m.owner_user_id, u.phone, sr.phone, sr.invite_code,
	       ST_Y(m.location::geometry), ST_X(m.location::geometry),
	       lm.path, lm.thumb_path,
	       m.status, m.commission_percent, m.emergency_closed, m.created_at
	FROM merchants m
	JOIN categories c ON c.id = m.category_id
	LEFT JOIN users u ON u.id = m.owner_user_id
	LEFT JOIN users sr ON sr.id = m.sales_rep_user_id
	LEFT JOIN media lm ON lm.id = m.logo_media_id`

func scanMerchant(row pgx.Row) (*Merchant, error) {
	var m Merchant
	err := row.Scan(&m.ID, &m.Name, &m.Description, &m.CategoryID, &m.CategoryName, &m.CategoryIcon,
		&m.Phone, &m.AddressText, &m.OwnerUserID, &m.OwnerPhone, &m.SalesRepPhone, &m.SalesRepCode,
		&m.Lat, &m.Lng, &m.LogoURL, &m.LogoThumbURL,
		&m.Status, &m.CommissionPct, &m.EmergencyClosed, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	m.LogoURL = media.URLForPtr(m.LogoURL)
	m.LogoThumbURL = media.URLForPtr(m.LogoThumbURL)
	return &m, nil
}

func (s *Service) ListMerchants(ctx context.Context, query, categoryID, status string, page, perPage int) (*MerchantPage, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	where := ` WHERE ($1 = '' OR m.name ILIKE '%'||$1||'%' OR m.phone ILIKE '%'||$1||'%')
	           AND ($2 = '' OR m.category_id::text = $2)
	           AND ($3 = '' OR m.status = $3)`

	var total int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM merchants m`+where,
		query, categoryID, status).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, merchantSelect+where+`
		ORDER BY m.created_at DESC LIMIT $4 OFFSET $5`,
		query, categoryID, status, perPage, (page-1)*perPage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	merchants := []Merchant{}
	for rows.Next() {
		m, err := scanMerchant(rows)
		if err != nil {
			return nil, err
		}
		merchants = append(merchants, *m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &MerchantPage{Merchants: merchants, Total: total, Page: page, PerPage: perPage}, nil
}

type MerchantInput struct {
	Name            *string  `json:"name"`
	Description     *string  `json:"description"`
	CategoryID      *string  `json:"category_id"`
	Phone           *string  `json:"phone"`
	AddressText     *string  `json:"address_text"`
	Status          *string  `json:"status"`
	CommissionPct   *int     `json:"commission_percent"`
	EmergencyClosed *bool    `json:"emergency_closed"`
	OwnerPhone      *string  `json:"owner_phone"`    // يربط/ينشئ حساب صاحب المتجر بدور merchant
	SalesRepCode    *string  `json:"sales_rep_code"` // كود دعوة المندوب — يُنسب له المتجر
	Lat             *float64 `json:"lat"`            // دبوس الموقع على الخريطة
	Lng             *float64 `json:"lng"`
	// معرف وسائط الشعار: غير مُرسل = بلا تغيير، "" = إزالة الشعار
	LogoMediaID *string `json:"logo_media_id"`
}

func (s *Service) CreateMerchant(ctx context.Context, actorID string, in MerchantInput, ip string) (*Merchant, error) {
	if in.Name == nil || *in.Name == "" || in.CategoryID == nil {
		return nil, ErrNameRequired
	}
	ownerID, err := s.resolveOwner(ctx, actorID, in.OwnerPhone, ip)
	if err != nil {
		return nil, err
	}
	repID, err := s.resolveRep(ctx, in.SalesRepCode)
	if err != nil {
		return nil, err
	}

	var id string
	err = s.db.QueryRow(ctx, `
		INSERT INTO merchants (name, description, category_id, phone, address_text, owner_user_id, sales_rep_user_id, location, logo_media_id, commission_percent, default_prep_minutes)
		VALUES ($1, COALESCE($2,''), $3, COALESCE($4,''), COALESCE($5,''), $6, $7,
		        CASE WHEN $8::float8 IS NOT NULL AND $9::float8 IS NOT NULL
		             THEN ST_SetSRID(ST_MakePoint($9::float8, $8::float8), 4326)::geography END,
		        NULLIF(COALESCE($10, ''), '')::uuid,
		        COALESCE($11, (SELECT (value#>>'{}')::int FROM app_settings
		                       WHERE key = 'merchants.default_commission_percent'), 10),
		        -- وقت التحضير الافتراضي من اللوحة لا من افتراض العمود: كان ٢٠
		        -- مكتوباً في الترحيل 0035، يرثه كل متجرٍ جديد ولا يملك المالك
		        -- تغييره لمن يأتي بعده.
		        COALESCE((SELECT (value#>>'{}')::int FROM app_settings
		                  WHERE key = 'merchants.default_prep_minutes'), 20))
		RETURNING id`,
		*in.Name, in.Description, *in.CategoryID, in.Phone, in.AddressText, ownerID, repID, in.Lat, in.Lng, in.LogoMediaID, in.CommissionPct).Scan(&id)
	if isFKViolation(err) {
		return nil, ErrCategoryInvalid
	}
	if err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "admin.merchant_create", "merchant", id, ip)
	return s.merchantByID(ctx, id)
}

func (s *Service) UpdateMerchant(ctx context.Context, actorID, id string, in MerchantInput, ip string) (*Merchant, error) {
	if in.Status != nil && *in.Status != "active" && *in.Status != "inactive" {
		return nil, ErrNameRequired
	}
	ownerID, err := s.resolveOwner(ctx, actorID, in.OwnerPhone, ip)
	if err != nil {
		return nil, err
	}
	repID, err := s.resolveRep(ctx, in.SalesRepCode)
	if err != nil {
		return nil, err
	}

	tag, err := s.db.Exec(ctx, `
		UPDATE merchants SET
			name          = COALESCE($2, name),
			description   = COALESCE($3, description),
			category_id   = COALESCE($4, category_id),
			phone         = COALESCE($5, phone),
			address_text  = COALESCE($6, address_text),
			status        = COALESCE($7, status),
			commission_percent = COALESCE($13, commission_percent),
			emergency_closed = COALESCE($8, emergency_closed),
			owner_user_id = COALESCE($9, owner_user_id),
			sales_rep_user_id = COALESCE($10, sales_rep_user_id),
			location      = COALESCE(
				CASE WHEN $11::float8 IS NOT NULL AND $12::float8 IS NOT NULL
				     THEN ST_SetSRID(ST_MakePoint($12::float8, $11::float8), 4326)::geography END,
				location),
			logo_media_id = CASE WHEN $14::text IS NULL THEN logo_media_id
			                     ELSE NULLIF($14, '')::uuid END,
			updated_at    = now()
		WHERE id = $1`,
		id, in.Name, in.Description, in.CategoryID, in.Phone, in.AddressText, in.Status, in.EmergencyClosed, ownerID, repID, in.Lat, in.Lng, in.CommissionPct, in.LogoMediaID)
	if isFKViolation(err) {
		return nil, ErrCategoryInvalid
	}
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, httpx.ErrNotFound
	}
	s.audit(ctx, actorID, "admin.merchant_update", "merchant", id, ip)
	return s.merchantByID(ctx, id)
}

func (s *Service) merchantByID(ctx context.Context, id string) (*Merchant, error) {
	m, err := scanMerchant(s.db.QueryRow(ctx, merchantSelect+` WHERE m.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	return m, err
}

// resolveRep ينسب المتجر للمندوب صاحب كود الدعوة.
func (s *Service) resolveRep(ctx context.Context, repCode *string) (*string, error) {
	if repCode == nil || *repCode == "" {
		return nil, nil
	}
	user, err := s.identity.SalesRepByInviteCode(ctx, *repCode)
	if err != nil {
		return nil, err
	}
	return &user.ID, nil
}

// resolveOwner يجد/ينشئ حساب صاحب المتجر بدور merchant من رقم هاتفه.
func (s *Service) resolveOwner(ctx context.Context, actorID string, ownerPhone *string, ip string) (*string, error) {
	if ownerPhone == nil || *ownerPhone == "" {
		return nil, nil
	}
	user, err := s.identity.EnsureUserWithRole(ctx, actorID, *ownerPhone, "merchant", ip)
	if err != nil {
		return nil, err
	}
	return &user.ID, nil
}

func (s *Service) audit(ctx context.Context, actorID, action, entity, entityID, ip string) {
	_, _ = s.db.Exec(ctx, `
		INSERT INTO audit_log (actor_user_id, action, entity, entity_id, ip)
		VALUES ($1, $2, $3, $4, $5)`, actorID, action, entity, entityID, ip)
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "22P02")
}
