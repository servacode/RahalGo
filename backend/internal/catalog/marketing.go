package catalog

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// أكواد الخصم في `promos.go`، والبانرات هنا — تُدار من اللوحة بالكامل.

// ---------- البانرات ----------

type Banner struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	ImageURL      *string `json:"image_url"`
	ImageThumbURL *string `json:"image_thumb_url"`
	Target        string  `json:"target"`
	SortOrder     int     `json:"sort_order"`
	Active        bool    `json:"active"`
	// Placement **أيُّ صفحةٍ تعرضها** — "shop" أو "home".
	//
	// (تصحيحُ المالك ٢٠٢٦-٠٨-١٧: «بانرات صفحة التسوّق مختلفة برأيي عن
	//  الرئيسيّة».)
	Placement string `json:"placement"`
	// Blur **لمحةٌ فوريّةٌ تُرسل مع الورقة** — (طلبُ المالك ٢٠٢٦-٠٨-١٧):
	// **تُرى في أوّل رسمةٍ فلا يُرى إطارٌ فارغٌ ينتظر.**
	Blur string `json:"blur"`
	// Sizes **هل للصورة نسخٌ أصغر؟** — فتُكتب `srcset` ولا تُخترع مسارات.
	Sizes bool `json:"sizes"`
}

const bannerSelect = `
	SELECT b.id, b.title, bm.path, bm.thumb_path, b.target, b.sort_order, b.active,
	       b.placement, COALESCE(bm.blur, ''), COALESCE(bm.sizes, false)
	FROM banners b
	LEFT JOIN media bm ON bm.id = b.image_media_id`

func scanBanner(row pgx.Row) (*Banner, error) {
	var b Banner
	if err := row.Scan(&b.ID, &b.Title, &b.ImageURL, &b.ImageThumbURL,
		&b.Target, &b.SortOrder, &b.Active, &b.Placement, &b.Blur, &b.Sizes); err != nil {
		return nil, err
	}
	b.ImageURL = media.URLForPtr(b.ImageURL)
	b.ImageThumbURL = media.URLForPtr(b.ImageThumbURL)
	return &b, nil
}

// ListBanners **لافتاتُ موضعٍ بعينه** — وفارغُ `placement` يعني الكلَّ.
//
// **ولا دالّتان**: الشاشةُ تطلب موضعاً والفحصُ يطلب الكلَّ، **ودالّتان
// بجسمٍ واحدٍ تفترقان يومَ يُضاف عمود.**
func (s *Service) ListBanners(ctx context.Context, placement string) ([]Banner, error) {
	rows, err := s.db.Query(ctx, bannerSelect+`
		WHERE ($1 = '' OR b.placement = $1)
		ORDER BY b.sort_order, b.created_at`, placement)
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
	// **وموضعُها يُختار عند الإنشاء** — وغيرُ المُرسل يبقى كما هو.
	Placement *string `json:"placement"`
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
	// **والموضعُ يُقيَّد هنا لا في القاعدة وحدَها** — قيدُ القاعدة يردّ
	// بخمسمئة، **ورسالةٌ تقول «موضعٌ لا أعرفه» تُقرأ.**
	place := "shop"
	if in.Placement != nil && *in.Placement == "home" {
		place = "home"
	}
	var id string
	err := s.db.QueryRow(ctx, `
		INSERT INTO banners (title, image_media_id, target, sort_order, placement)
		VALUES ($1, NULLIF(COALESCE($2,''), '')::uuid, COALESCE($3,''),
		        COALESCE($4, (SELECT COALESCE(max(sort_order)+1,1)
		                      FROM banners WHERE placement = $5)), $5)
		RETURNING id`,
		*in.Title, in.ImageMediaID, in.Target, in.SortOrder, place).Scan(&id)
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
