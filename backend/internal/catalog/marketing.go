package catalog

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// أكواد الخصم والبانرات — تُدار من اللوحة بالكامل.

var ErrCodeTaken = httpx.NewError(http.StatusConflict, "code_taken", "errors.code_taken")

type PromoCode struct {
	ID             string     `json:"id"`
	Code           string     `json:"code"`
	Kind           string     `json:"kind"`
	Value          int64      `json:"value"`
	MinOrder       int64      `json:"min_order"`
	FirstOrderOnly bool       `json:"first_order_only"`
	OncePerUser    bool       `json:"once_per_user"`
	MaxUses        *int       `json:"max_uses"`
	UsedCount      int        `json:"used_count"`
	ExpiresAt      *time.Time `json:"expires_at"`
	Active         bool       `json:"active"`
	CreatedAt      time.Time  `json:"created_at"`
}

const promoCols = `id, code, kind, value, min_order, first_order_only, once_per_user,
	max_uses, used_count, expires_at, active, created_at`

func scanPromo(row pgx.Row) (*PromoCode, error) {
	var p PromoCode
	err := row.Scan(&p.ID, &p.Code, &p.Kind, &p.Value, &p.MinOrder, &p.FirstOrderOnly,
		&p.OncePerUser, &p.MaxUses, &p.UsedCount, &p.ExpiresAt, &p.Active, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) ListPromos(ctx context.Context) ([]PromoCode, error) {
	rows, err := s.db.Query(ctx, `SELECT `+promoCols+` FROM promo_codes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PromoCode{}
	for rows.Next() {
		p, err := scanPromo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

type PromoInput struct {
	Code           *string    `json:"code"`
	Kind           *string    `json:"kind"`
	Value          *int64     `json:"value"`
	MinOrder       *int64     `json:"min_order"`
	FirstOrderOnly *bool      `json:"first_order_only"`
	OncePerUser    *bool      `json:"once_per_user"`
	MaxUses        *int       `json:"max_uses"`
	ExpiresAt      *time.Time `json:"expires_at"`
	Active         *bool      `json:"active"`
}

func (s *Service) CreatePromo(ctx context.Context, actorID string, in PromoInput, ip string) (*PromoCode, error) {
	if in.Code == nil || *in.Code == "" || in.Kind == nil {
		return nil, ErrNameRequired
	}
	switch *in.Kind {
	case "percent", "fixed", "free_delivery":
	default:
		return nil, ErrNameRequired
	}
	p, err := scanPromo(s.db.QueryRow(ctx, `
		INSERT INTO promo_codes (code, kind, value, min_order, first_order_only, once_per_user, max_uses, expires_at)
		VALUES ($1, $2, COALESCE($3,0), COALESCE($4,0), COALESCE($5,false), COALESCE($6,true), $7, $8)
		RETURNING `+promoCols,
		*in.Code, *in.Kind, in.Value, in.MinOrder, in.FirstOrderOnly, in.OncePerUser, in.MaxUses, in.ExpiresAt))
	if isUniqueViolationCatalog(err) {
		return nil, ErrCodeTaken
	}
	if err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "admin.promo_create", "promo", p.ID, ip)
	return p, nil
}

func (s *Service) UpdatePromo(ctx context.Context, actorID, id string, in PromoInput, ip string) (*PromoCode, error) {
	p, err := scanPromo(s.db.QueryRow(ctx, `
		UPDATE promo_codes SET
			value            = COALESCE($2, value),
			min_order        = COALESCE($3, min_order),
			first_order_only = COALESCE($4, first_order_only),
			once_per_user    = COALESCE($5, once_per_user),
			max_uses         = COALESCE($6, max_uses),
			expires_at       = COALESCE($7, expires_at),
			active           = COALESCE($8, active)
		WHERE id = $1
		RETURNING `+promoCols,
		id, in.Value, in.MinOrder, in.FirstOrderOnly, in.OncePerUser, in.MaxUses, in.ExpiresAt, in.Active))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "admin.promo_update", "promo", id, ip)
	return p, nil
}

// ---------- البانرات ----------

type Banner struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	ImageURL      *string `json:"image_url"`
	ImageThumbURL *string `json:"image_thumb_url"`
	Target        string  `json:"target"`
	SortOrder     int     `json:"sort_order"`
	Active        bool    `json:"active"`
}

const bannerSelect = `
	SELECT b.id, b.title, bm.path, bm.thumb_path, b.target, b.sort_order, b.active
	FROM banners b
	LEFT JOIN media bm ON bm.id = b.image_media_id`

func scanBanner(row pgx.Row) (*Banner, error) {
	var b Banner
	if err := row.Scan(&b.ID, &b.Title, &b.ImageURL, &b.ImageThumbURL,
		&b.Target, &b.SortOrder, &b.Active); err != nil {
		return nil, err
	}
	b.ImageURL = media.URLForPtr(b.ImageURL)
	b.ImageThumbURL = media.URLForPtr(b.ImageThumbURL)
	return &b, nil
}

func (s *Service) ListBanners(ctx context.Context) ([]Banner, error) {
	rows, err := s.db.Query(ctx, bannerSelect+` ORDER BY b.sort_order, b.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Banner{}
	for rows.Next() {
		b, err := scanBanner(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

type BannerInput struct {
	Title *string `json:"title"`
	// معرف وسائط الصورة: غير مُرسل = بلا تغيير، "" = إزالة الصورة
	ImageMediaID *string `json:"image_media_id"`
	Target       *string `json:"target"`
	SortOrder    *int    `json:"sort_order"`
	Active       *bool   `json:"active"`
}

func (s *Service) CreateBanner(ctx context.Context, actorID string, in BannerInput, ip string) (*Banner, error) {
	// **ولا عنوانَ يُشترط.**
	//
	// (قرارُ المالك 2026-08-09: «لا يوجد داعٍ لعنوان البانر ولا للزرّ أيضاً».)
	//
	// **واللافتةُ صورةٌ تُعرض** — تصميمٌ فيه كلامُه ودعوتُه. **وشرطٌ على حقلٍ
	// لا يُعرض يمنع الحفظَ بلا أن يفهم أحدٌ لماذا.**
	//
	// **والعمودُ يبقى**: صفوفٌ قديمةٌ فيها عناوين، **وحذفُ عمودٍ فيه بياناتٌ
	// قرارٌ آخر.** ويُقرأ في `alt` لمن لا يرى.
	if in.Title == nil {
		empty := ""
		in.Title = &empty
	}
	var id string
	err := s.db.QueryRow(ctx, `
		INSERT INTO banners (title, image_media_id, target, sort_order)
		VALUES ($1, NULLIF(COALESCE($2,''), '')::uuid, COALESCE($3,''),
		        COALESCE($4, (SELECT COALESCE(max(sort_order)+1,1) FROM banners)))
		RETURNING id`,
		*in.Title, in.ImageMediaID, in.Target, in.SortOrder).Scan(&id)
	if err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "admin.banner_create", "banner", id, ip)
	return s.bannerByID(ctx, id)
}

func (s *Service) UpdateBanner(ctx context.Context, actorID, id string, in BannerInput, ip string) (*Banner, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE banners SET
			title      = COALESCE($2, title),
			image_media_id = CASE WHEN $3::text IS NULL THEN image_media_id
			                      ELSE NULLIF($3, '')::uuid END,
			target     = COALESCE($4, target),
			sort_order = COALESCE($5, sort_order),
			active     = COALESCE($6, active)
		WHERE id = $1`,
		id, in.Title, in.ImageMediaID, in.Target, in.SortOrder, in.Active)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, httpx.ErrNotFound
	}
	s.audit(ctx, actorID, "admin.banner_update", "banner", id, ip)
	return s.bannerByID(ctx, id)
}

func (s *Service) bannerByID(ctx context.Context, id string) (*Banner, error) {
	b, err := scanBanner(s.db.QueryRow(ctx, bannerSelect+` WHERE b.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	return b, err
}

func (s *Service) DeleteBanner(ctx context.Context, actorID, id, ip string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM banners WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	s.audit(ctx, actorID, "admin.banner_delete", "banner", id, ip)
	return nil
}

func isUniqueViolationCatalog(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
