package catalog

// ══════════════════════════════════════════════════════════════════════
// **أدواتٌ توفّر وقتَ المتجر** (طلبُ المالك ٢٠٢٦-١٠-٢٧)
// ══════════════════════════════════════════════════════════════════════
//
//   DuplicateItem    قميصٌ بخمسة ألوان يُعمل مرّةً ويُنسخ.
//   BulkPrice        «زيد كلّ أسعاري ١٠٪» أو قسماً واحداً منها.

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// copySuffix ما يُلحق باسم النسخة — يعرف صاحبُ المتجر أيَّهما الجديد.
const copySuffix = " (نسخة)"

// DuplicateItem **ينسخ صنفاً بصورته وقسمه وخياراته** — والنسخةُ تمرّ بما
// يمرّ به أيُّ صنفٍ جديد (المراجعة، والمشروبات تُنسخ إليها من القائمة).
// ويُرجع معرّفَ النسخة ليفتحها التطبيقُ فيعدّلها.
func (s *Service) DuplicateItem(ctx context.Context, actorID, itemID string, maxName int, ip string) (string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var merchantID, name, desc string
	var price int64
	var image, section *string
	var usdCents *int64
	err = tx.QueryRow(ctx, `
		SELECT merchant_id::text, name, description, merchant_price, image_media_id::text,
		       platform_section_id::text, price_usd_cents
		  FROM menu_items WHERE id = $1::uuid`, itemID).
		Scan(&merchantID, &name, &desc, &price, &image, &section, &usdCents)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", httpx.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if section == nil || *section == "" {
		return "", ErrSectionRequired
	}

	// **والاسمُ يُقصّ ليتّسع للّاحقة** — لا يُرفض نسخُ صنفٍ اسمُه طويل.
	if maxName <= 0 {
		maxName = MaxItemName
	}
	base := []rune(name)
	room := maxName - len([]rune(copySuffix))
	if room < 1 {
		room = 1
	}
	if len(base) > room {
		base = base[:room]
	}
	newName := strings.TrimSpace(string(base)) + copySuffix

	in := MenuItemInput{
		Name:              &newName,
		Description:       &desc,
		Price:             &price,
		ImageMediaID:      image,
		PlatformSectionID: section,
	}
	if usdCents != nil {
		cur := CurrencyUSD
		d := float64(*usdCents) / 100
		in.PriceCurrency, in.PriceUSD = &cur, &d
	}

	// **والخياراتُ بلا المشروبات المنسوخة** — تلك تعود من قائمة المتجر وحدَها.
	groups := []ModifierGroupInput{}
	rows, err := tx.Query(ctx, `
		SELECT g.id::text, g.name, g.min_select, g.max_select
		  FROM modifier_groups g
		 WHERE g.item_id = $1::uuid AND NOT g.store_drinks
		 ORDER BY g.sort_order`, itemID)
	if err != nil {
		return "", err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		var g ModifierGroupInput
		if err := rows.Scan(&id, &g.Name, &g.MinSelect, &g.MaxSelect); err != nil {
			rows.Close()
			return "", err
		}
		ids = append(ids, id)
		groups = append(groups, g)
	}
	rows.Close()
	for i, gid := range ids {
		orows, err := tx.Query(ctx, `
			SELECT name, price_delta, available, price_delta_usd_cents
			  FROM modifier_options WHERE group_id = $1::uuid ORDER BY sort_order`, gid)
		if err != nil {
			return "", err
		}
		for orows.Next() {
			var o ModifierOptionInput
			var av bool
			var oc *int64
			if err := orows.Scan(&o.Name, &o.PriceDelta, &av, &oc); err != nil {
				orows.Close()
				return "", err
			}
			o.Available = &av
			if oc != nil {
				d := float64(*oc) / 100
				o.PriceDeltaUSD = &d
			}
			groups[i].Options = append(groups[i].Options, o)
		}
		orows.Close()
	}
	in.Modifiers = &groups

	id, err := s.CreateItemIn(ctx, tx, merchantID, in)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	s.audit(ctx, actorID, "menu.item_duplicate", "menu_item", id, ip)
	return id, nil
}

// BulkPriceInput **تغييرُ الأسعار دفعةً واحدة.**
type BulkPriceInput struct {
	// Percent **نسبةُ التغيير** — ١٠ تعني زيادةً ١٠٪ و-١٠ نقصاً.
	Percent float64 `json:"percent"`
	// PlatformSectionID **قسمٌ واحد** — وفراغُه كلُّ الأصناف.
	PlatformSectionID *string `json:"platform_section_id"`
	// Confirm **بلاه معاينةٌ لا تكتب** — يرى العددَ والأمثلةَ قبل أن يقرّر.
	Confirm bool `json:"confirm"`
}

// BulkPriceResult ما يُردّ في المعاينة والتطبيق معاً.
type BulkPriceResult struct {
	Count   int          `json:"count"`
	Samples []RateSample `json:"samples"`
	Applied bool         `json:"applied"`
}

var ErrBulkPercent = httpx.NewError(http.StatusBadRequest, "bulk_percent_invalid", "errors.bulk_percent_invalid")

// newPriceSQL **السعرُ الجديد لصفٍّ من `menu_items i`** — والليرةُ تُقرَّب
// لأقرب ليرة ولا تنزل تحت الواحد، **والدولارُ يُغيَّر بسنته ثمّ يُحسب بالليرة
// بسعر الصرف** كما يُحسب في كلّ مكان (`usd.go`).
const newPriceSQL = `
	CASE WHEN i.price_usd_cents IS NULL
	     THEN GREATEST(1, round(i.merchant_price * (100 + $2::numeric) / 100))::bigint
	     ELSE (round(round(i.price_usd_cents * (100 + $2::numeric) / 100) * COALESCE(m.exchange_rate, 0) / 100 / 5) * 5)::bigint
	END`

// BulkPrice **يعاين أو يطبّق** نسبةً على أصناف المتجر أو قسمٍ منها.
func (s *Service) BulkPrice(ctx context.Context, actorID, merchantID string, in BulkPriceInput, ip string) (*BulkPriceResult, error) {
	if in.Percent == 0 || in.Percent < -50 || in.Percent > 100 || math.IsNaN(in.Percent) {
		return nil, ErrBulkPercent
	}
	section := ""
	if in.PlatformSectionID != nil {
		section = strings.TrimSpace(*in.PlatformSectionID)
	}
	// **والصنفُ المجّانيُّ يبقى مجّانيّاً** (قرارُ المالك ٢٠٢٦-١٠-٢٨: السعرُ صفرٌ
	// مسموح) — كان `GREATEST(1, …)` يقلب صفرَه ليرةً. فلا يُمسّ ولا يُعدّ.
	where := `i.merchant_id = $1::uuid AND ($3::text = '' OR i.platform_section_id::text = $3)
	          AND ((i.price_usd_cents IS NULL AND i.merchant_price > 0)
	               OR (i.price_usd_cents > 0 AND COALESCE(m.exchange_rate, 0) > 0))`

	res := &BulkPriceResult{Samples: []RateSample{}}
	rows, err := s.db.Query(ctx, fmt.Sprintf(`
		SELECT i.name, i.merchant_price, %s, count(*) OVER ()
		  FROM menu_items i JOIN merchants m ON m.id = i.merchant_id
		 WHERE %s
		 ORDER BY i.sort_order, i.created_at`, newPriceSQL, where), merchantID, in.Percent, section)
	if err != nil {
		return nil, err
	}
	maxPrice := int64(0)
	if s.settings != nil {
		maxPrice = s.settings.GetInt(ctx, "merchants.max_item_price")
	}
	for rows.Next() {
		var sm RateSample
		if err := rows.Scan(&sm.Name, &sm.OldPrice, &sm.NewPrice, &res.Count); err != nil {
			rows.Close()
			return nil, err
		}
		if maxPrice > 0 && sm.NewPrice > maxPrice {
			rows.Close()
			return nil, &httpx.AppError{Status: http.StatusBadRequest, Code: "item_price_too_high",
				MessageKey: "errors.item_price_too_high", Details: map[string]any{"max": maxPrice}}
		}
		if len(res.Samples) < 3 {
			res.Samples = append(res.Samples, sm)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if !in.Confirm || res.Count == 0 {
		return res, nil
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// **والدولاريُّ تتبدّل سنتُه أوّلاً** — فيبقى سعرُ الصرف القادمُ يحسب منها.
	if _, err := tx.Exec(ctx, fmt.Sprintf(`
		UPDATE menu_items i
		   SET merchant_price = %[1]s,
		       price          = %[1]s,
		       price_usd_cents = CASE WHEN i.price_usd_cents IS NULL THEN NULL
		                              ELSE round(i.price_usd_cents * (100 + $2::numeric) / 100)::bigint END,
		       updated_at = now()
		  FROM merchants m
		 WHERE m.id = i.merchant_id AND %[2]s`, newPriceSQL, where), merchantID, in.Percent, section); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	res.Applied = true
	s.auditDetail(ctx, actorID, "menu.bulk_price", "merchant", merchantID, ip,
		map[string]any{"percent": in.Percent, "section": section, "count": res.Count})
	return res, nil
}
