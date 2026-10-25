package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **كلُّ تقييمٍ بسطره** — `GET /admin/ratings/list` (المالك ٢٠٢٦-١٠-٢٥: «لازم أعرف كل زبون
// قيّم شو للسائق والمتجر»)
// ══════════════════════════════════════════════════════════════════════
//
// كانت شاشةُ التقييمات معدّلاتٍ بالسائق وبالمتجر، وتعليقاتِ المنخفض وحدَها. **وهذه القائمةُ
// كلُّ طلبٍ قُيّم**: الزبونُ ونجومُه للطلب وللسائق، **والمتجرُ وتقييمُ السائق له** (السرعة
// والتعامل — السائقُ يقيّم المتجر لا الزبون). الأحدثُ أوّلاً، و`low=1` المنخفضُ وحدَه.

type ratingRow struct {
	OrderID       string    `json:"order_id"`
	OrderNumber   int64     `json:"order_number"`
	CreatedAt     time.Time `json:"created_at"`
	CustomerID    string    `json:"customer_id"`
	CustomerName  string    `json:"customer_name"`
	PlatformStars *int      `json:"platform_stars"`
	DriverID      string    `json:"driver_id"`
	DriverName    string    `json:"driver_name"`
	DriverStars   *int      `json:"driver_stars"`
	MerchantID    string    `json:"merchant_id"`
	MerchantName  string    `json:"merchant_name"`
	StoreSpeed    *int      `json:"store_speed"`
	StoreConduct  *int      `json:"store_conduct"`
}

func (s *Server) handleAdminRatingsList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	const perPage = 30
	low := q.Get("low") == "1"
	ctx := r.Context()

	// **الطلبُ المُقيَّمُ من أيٍّ من الجهتين** — الزبونُ أو السائق.
	const base = `
		FROM orders o
		LEFT JOIN order_ratings rt ON rt.order_id = o.id
		LEFT JOIN merchant_ratings mr ON mr.order_id = o.id
		LEFT JOIN users cu ON cu.id = o.customer_id
		LEFT JOIN users dr ON dr.id = o.driver_id
		LEFT JOIN merchants m ON m.id = COALESCE(mr.merchant_id, o.merchant_id)
		WHERE (rt.order_id IS NOT NULL OR mr.order_id IS NOT NULL)
		  AND (NOT $1 OR rt.platform_stars <= 2 OR rt.driver_stars <= 2
		       OR mr.speed_stars <= 2 OR mr.conduct_stars <= 2)`

	var total int64
	if err := s.pg.QueryRow(ctx, `SELECT count(*) `+base, low).Scan(&total); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(ctx, `
		SELECT o.id::text, o.number, GREATEST(rt.created_at, mr.created_at),
		       COALESCE(o.customer_id::text, ''), COALESCE(cu.full_name, o.recipient_name, ''),
		       rt.platform_stars,
		       COALESCE(o.driver_id::text, ''), COALESCE(dr.full_name, ''), rt.driver_stars,
		       COALESCE(m.id::text, ''), COALESCE(m.name, ''), mr.speed_stars, mr.conduct_stars
		`+base+`
		ORDER BY GREATEST(rt.created_at, mr.created_at) DESC
		LIMIT $2 OFFSET $3`, low, perPage, (page-1)*perPage)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []ratingRow{}
	for rows.Next() {
		var x ratingRow
		if err := rows.Scan(&x.OrderID, &x.OrderNumber, &x.CreatedAt, &x.CustomerID, &x.CustomerName,
			&x.PlatformStars, &x.DriverID, &x.DriverName, &x.DriverStars,
			&x.MerchantID, &x.MerchantName, &x.StoreSpeed, &x.StoreConduct); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"ratings": out, "total": total, "page": page, "per_page": perPage,
	})
}
