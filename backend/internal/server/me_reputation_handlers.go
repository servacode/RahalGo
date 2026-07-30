package server

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// كشف سمعة المستخدم (موظف): تقييمه واتجاهه، والشكاوى بحقّه، والتقييمات والتعليقات
// التي تلقّاها — كي يرى كلٌّ ما قُدّم بحقّه بشفافية. حسب دوره (متجر/سائق/مندوب).

type repReview struct {
	OrderNumber  int64     `json:"order_number"`
	MerchantName string    `json:"merchant_name"`
	Stars        int       `json:"stars"`
	Comment      string    `json:"comment"`
	CreatedAt    time.Time `json:"created_at"`
}

type repComplaint struct {
	Number      int64     `json:"number"`
	OrderNumber *int64    `json:"order_number"`
	Subject     string    `json:"subject"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Server) handleMeReputation(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	ctx := r.Context()

	var roles []string
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(array_agg(role_code), '{}') FROM user_roles WHERE user_id = $1`, uid).Scan(&roles)
	has := func(role string) bool {
		for _, x := range roles {
			if x == role {
				return true
			}
		}
		return false
	}

	// نحدّد نطاق الطلبات ذات الصلة حسب الدور الأساسي:
	//  - المتجر: طلبات متاجره، وعمود النجوم merchant_stars
	//  - السائق: طلبات سلّمها، وعمود driver_stars
	//  - المندوب: طلبات متاجره (جودة محفظته)، merchant_stars
	// ownerCond: شرط ربط الطلب بالمستخدم؛ starCol: عمود النجوم المعني.
	var ownerJoin, ownerCond, starCol string
	switch {
	case has("merchant"):
		ownerJoin = `JOIN merchants mm ON mm.id = o.merchant_id`
		ownerCond = `mm.owner_user_id = $1`
		starCol = `rt.merchant_stars`
	case has("driver"):
		ownerJoin = ``
		ownerCond = `o.driver_id = $1`
		starCol = `rt.driver_stars`
	case has("sales"):
		ownerJoin = `JOIN merchants mm ON mm.id = o.merchant_id`
		ownerCond = `mm.sales_rep_user_id = $1`
		starCol = `rt.merchant_stars`
	default:
		httpx.JSON(w, http.StatusOK, map[string]any{
			"rating":     map[string]any{"avg": 0, "count": 0, "trend": "flat"},
			"complaints": []repComplaint{}, "reviews": []repReview{}})
		return
	}

	out := struct {
		Rating struct {
			Avg       float64 `json:"avg"`
			Count     int     `json:"count"`
			RecentAvg float64 `json:"recent_avg"`
			Trend     string  `json:"trend"` // up | down | flat
		} `json:"rating"`
		Complaints []repComplaint `json:"complaints"`
		Reviews    []repReview    `json:"reviews"`
	}{Complaints: []repComplaint{}, Reviews: []repReview{}}

	// التقييم: المتوسط والعدد، ومتوسط آخر 30 يوماً لاستنتاج الاتجاه.
	_ = s.pg.QueryRow(ctx, `
		SELECT COALESCE(avg(`+starCol+`), 0), count(`+starCol+`),
		       COALESCE(avg(`+starCol+`) FILTER (WHERE rt.created_at > now() - interval '30 days'), 0)
		FROM order_ratings rt JOIN orders o ON o.id = rt.order_id `+ownerJoin+`
		WHERE `+ownerCond+` AND `+starCol+` IS NOT NULL`, uid).
		Scan(&out.Rating.Avg, &out.Rating.Count, &out.Rating.RecentAvg)
	switch {
	case out.Rating.Count == 0 || out.Rating.RecentAvg == 0:
		out.Rating.Trend = "flat"
	case out.Rating.RecentAvg > out.Rating.Avg+0.1:
		out.Rating.Trend = "up"
	case out.Rating.RecentAvg < out.Rating.Avg-0.1:
		out.Rating.Trend = "down"
	default:
		out.Rating.Trend = "flat"
	}

	// التقييمات والتعليقات المتلقّاة (نُظهر ذوات التعليق أولاً).
	rows, err := s.pg.Query(ctx, `
		SELECT o.number, m.name, `+starCol+`, COALESCE(rt.comment, ''), rt.created_at
		FROM order_ratings rt
		JOIN orders o ON o.id = rt.order_id
		JOIN merchants m ON m.id = o.merchant_id
		`+ownerJoin+`
		WHERE `+ownerCond+` AND `+starCol+` IS NOT NULL
		ORDER BY (rt.comment <> '') DESC, rt.created_at DESC LIMIT 50`, uid)
	if err == nil {
		for rows.Next() {
			var rv repReview
			if rows.Scan(&rv.OrderNumber, &rv.MerchantName, &rv.Stars, &rv.Comment, &rv.CreatedAt) == nil {
				out.Reviews = append(out.Reviews, rv)
			}
		}
		rows.Close()
	}

	// الشكاوى/البلاغات بحقّه (تذاكر على طلباته).
	crows, err := s.pg.Query(ctx, `
		SELECT t.number, o.number, t.subject, t.status, t.created_at
		FROM tickets t
		JOIN orders o ON o.id = t.order_id
		`+ownerJoin+`
		WHERE `+ownerCond+`
		ORDER BY t.created_at DESC LIMIT 50`, uid)
	if err == nil {
		for crows.Next() {
			var c repComplaint
			if crows.Scan(&c.Number, &c.OrderNumber, &c.Subject, &c.Status, &c.CreatedAt) == nil {
				out.Complaints = append(out.Complaints, c)
			}
		}
		crows.Close()
	}

	httpx.JSON(w, http.StatusOK, out)
}
