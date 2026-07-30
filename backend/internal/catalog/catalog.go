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
	Status          string    `json:"status"`
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
	       m.phone, m.address_text, m.owner_user_id, u.phone, m.status, m.emergency_closed, m.created_at
	FROM merchants m
	JOIN categories c ON c.id = m.category_id
	LEFT JOIN users u ON u.id = m.owner_user_id`

func scanMerchant(row pgx.Row) (*Merchant, error) {
	var m Merchant
	err := row.Scan(&m.ID, &m.Name, &m.Description, &m.CategoryID, &m.CategoryName, &m.CategoryIcon,
		&m.Phone, &m.AddressText, &m.OwnerUserID, &m.OwnerPhone, &m.Status, &m.EmergencyClosed, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
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
	Name            *string `json:"name"`
	Description     *string `json:"description"`
	CategoryID      *string `json:"category_id"`
	Phone           *string `json:"phone"`
	AddressText     *string `json:"address_text"`
	Status          *string `json:"status"`
	EmergencyClosed *bool   `json:"emergency_closed"`
	OwnerPhone      *string `json:"owner_phone"` // يربط/ينشئ حساب صاحب المتجر بدور merchant
}

func (s *Service) CreateMerchant(ctx context.Context, actorID string, in MerchantInput, ip string) (*Merchant, error) {
	if in.Name == nil || *in.Name == "" || in.CategoryID == nil {
		return nil, ErrNameRequired
	}
	ownerID, err := s.resolveOwner(ctx, actorID, in.OwnerPhone, ip)
	if err != nil {
		return nil, err
	}

	var id string
	err = s.db.QueryRow(ctx, `
		INSERT INTO merchants (name, description, category_id, phone, address_text, owner_user_id)
		VALUES ($1, COALESCE($2,''), $3, COALESCE($4,''), COALESCE($5,''), $6)
		RETURNING id`,
		*in.Name, in.Description, *in.CategoryID, in.Phone, in.AddressText, ownerID).Scan(&id)
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

	tag, err := s.db.Exec(ctx, `
		UPDATE merchants SET
			name          = COALESCE($2, name),
			description   = COALESCE($3, description),
			category_id   = COALESCE($4, category_id),
			phone         = COALESCE($5, phone),
			address_text  = COALESCE($6, address_text),
			status        = COALESCE($7, status),
			emergency_closed = COALESCE($8, emergency_closed),
			owner_user_id = COALESCE($9, owner_user_id),
			updated_at    = now()
		WHERE id = $1`,
		id, in.Name, in.Description, in.CategoryID, in.Phone, in.AddressText, in.Status, in.EmergencyClosed, ownerID)
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
