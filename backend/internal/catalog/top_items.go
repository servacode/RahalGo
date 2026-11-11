package catalog

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/media"
)

// TopItem one of the store's best sellers: delivered quantity over the window.
type TopItem struct {
	// ItemID is empty when the item was deleted after it sold.
	ItemID        string  `json:"item_id"`
	Name          string  `json:"name"`
	Qty           int     `json:"qty"`
	ImageThumbURL *string `json:"image_thumb_url"`
}

// TopSellingItems the store's best sellers over the last `days` days (owner 2026-11-11:
// «الأكتر مبيعاً هالأسبوع»): items by DELIVERED quantity only, highest first, at most `limit`.
// A non-delivered order (cancelled, failed, still open) never counts.
func (s *Service) TopSellingItems(ctx context.Context, merchantID string, days, limit int) ([]TopItem, error) {
	if days <= 0 {
		days = 7
	}
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	rows, err := s.db.Query(ctx, `
		SELECT COALESCE(oi.menu_item_id::text, ''), (array_agg(oi.name ORDER BY o.created_at DESC))[1],
		       sum(oi.qty)::int AS qty, max(m.thumb_path)
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		LEFT JOIN menu_items mi ON mi.id = oi.menu_item_id
		LEFT JOIN media m ON m.id = mi.image_media_id
		WHERE oi.merchant_id = $1
		  AND o.status = 'delivered'
		  AND COALESCE(o.delivered_at, o.created_at) >= now() - make_interval(days => $2)
		GROUP BY COALESCE(oi.menu_item_id::text, oi.name)
                 , COALESCE(oi.menu_item_id::text, '')
		ORDER BY qty DESC, 2
		LIMIT $3`, merchantID, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TopItem{}
	for rows.Next() {
		var t TopItem
		if err := rows.Scan(&t.ItemID, &t.Name, &t.Qty, &t.ImageThumbURL); err != nil {
			return nil, err
		}
		t.ImageThumbURL = media.URLForPtr(t.ImageThumbURL)
		out = append(out, t)
	}
	return out, rows.Err()
}
