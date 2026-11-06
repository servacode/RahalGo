package catalog

import "context"

// NoPhotoItem is one item of a store that has no photo yet (fast photo mode, owner decision 2026-11-06).
type NoPhotoItem struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	SectionName      string   `json:"section_name"`
	MerchantPrice    int64    `json:"merchant_price"`
	PriceCurrency    string   `json:"price_currency"`
	MerchantPriceUSD *float64 `json:"merchant_price_usd"`
	Available        bool     `json:"available"`
}

// NoPhotoPage is a keyset page of photo-less items plus the total still without a photo.
type NoPhotoPage struct {
	Items     []NoPhotoItem `json:"items"`
	Total     int           `json:"total"`
	NextAfter string        `json:"next_after"`
}

// ListNoPhotoItems lists the store's items that have no photo, ordered by market section name,
// then item name, then id (stable). `after` is the id of the last item already shown: the page
// starts strictly after it, so the list stays correct while items leave it as photos are saved.
func (s *Service) ListNoPhotoItems(ctx context.Context, merchantID, after string, limit int) (*NoPhotoPage, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	const from = `
		FROM menu_items i
		LEFT JOIN platform_sections ps ON ps.id = i.platform_section_id
		WHERE i.merchant_id = $1 AND i.image_media_id IS NULL`
	out := &NoPhotoPage{Items: []NoPhotoItem{}}
	if err := s.db.QueryRow(ctx, `SELECT count(*) `+from, merchantID).Scan(&out.Total); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT i.id, i.name, COALESCE(ps.name, ''), i.merchant_price, i.price_usd_cents, i.available `+from+`
		   AND ($2 = '' OR (COALESCE(ps.name, ''), i.name, i.id::text) > (
		        SELECT COALESCE(ps2.name, ''), i2.name, i2.id::text
		          FROM menu_items i2 LEFT JOIN platform_sections ps2 ON ps2.id = i2.platform_section_id
		         WHERE i2.id::text = $2 AND i2.merchant_id = $1))
		 ORDER BY COALESCE(ps.name, ''), i.name, i.id::text
		 LIMIT $3`, merchantID, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var it NoPhotoItem
		var usdCents *int64
		if err := rows.Scan(&it.ID, &it.Name, &it.SectionName, &it.MerchantPrice, &usdCents, &it.Available); err != nil {
			return nil, err
		}
		it.PriceCurrency = CurrencySYP
		if usdCents != nil {
			it.PriceCurrency = CurrencyUSD
			d := float64(*usdCents) / 100
			it.MerchantPriceUSD = &d
		}
		out.Items = append(out.Items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if n := len(out.Items); n > 0 && n == limit {
		out.NextAfter = out.Items[n-1].ID
	}
	return out, nil
}
