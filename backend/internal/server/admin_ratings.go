package server

// **التقييماتُ مجموعةً — من يشكو منه الناس؟**
//
// # المسألة
//
// التقييماتُ تُقرأ في ملفّ كلّ إنسانٍ على حدة. **فسؤالُ «أيُّ سائقٍ يشكو منه
// الناس؟» لا جوابَ له إلّا بفتح عشرين ملفّاً** — فلا يُفتح، **فلا يُعرف.**
//
// **ونجمتان لسائقٍ في ملفّه رقمٌ يخصّه، ونجمتان بين خمسةٍ متوسّطُهم أربعٌ
// مسألةٌ تخصّ المنصة.** والفرقُ بينهما هو الفرقُ بين بيانٍ ومعلومة.
//
// # ولماذا المتوسّطُ وحدَه لا يكفي
//
// سائقٌ أوصل طلبين وأخذ نجمةً في أحدهما متوسّطُه ثلاث، **وسائقٌ أوصل مئتين
// ومتوسّطُه ثلاثٌ حالةٌ أخرى تماماً.** فيُعرض **العددُ مع المتوسّط**، ويُرتَّب
// بالأسوأ **بشرط عددٍ أدنى** — فلا يتصدّر القائمةَ من قُيّم مرّةً واحدة.
//
// # والتعليقاتُ الأخيرة معها
//
// **ورقمٌ بلا كلامٍ لا يُصلح شيئاً**: من رأى «٢٫٤» لا يعرف أهو تأخّرٌ أم سوءُ
// خلق. **والكلمةُ تقول ما لا يقوله الرقم.**

import (
	"net/http"
	"strconv"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

type ratedParty struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Phone   string  `json:"phone"`
	Count   int     `json:"count"`
	Average float64 `json:"average"`
	Low     int     `json:"low"` // كم مرّةً نجمتان أو أقلّ
}

type ratingComment struct {
	OrderNumber int64     `json:"order_number"`
	Customer    string    `json:"customer"`
	Driver      string    `json:"driver"`
	Platform    int       `json:"platform_stars"`
	DriverStars *int      `json:"driver_stars"`
	Comment     string    `json:"comment"`
	CreatedAt   time.Time `json:"created_at"`
}

// handleAdminRatings **صورةُ التقييمات كلِّها**: متوسّطُ المنصة، والسائقون
// مرتَّبين بالأسوأ، وآخرُ ما كُتب.
func (s *Server) handleAdminRatings(w http.ResponseWriter, r *http.Request) {
	// **الحدُّ الأدنى للعدّ** — كي لا يتصدّر من قُيّم مرّةً واحدة.
	minCount := 1
	if v, err := strconv.Atoi(r.URL.Query().Get("min")); err == nil && v > 0 {
		minCount = v
	}

	// ١ · متوسّطُ المنصة وعددُ التقييمات — **السؤالُ الأوّل قبل التفصيل.**
	var total int
	var avg *float64
	var lowCount int
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*), avg(platform_stars)::float8,
		       count(*) FILTER (WHERE platform_stars <= 2)
		FROM order_ratings`).Scan(&total, &avg, &lowCount); err != nil {
		s.respondErr(w, err)
		return
	}

	// ٢ · السائقون — **بالأسوأ أوّلاً**، ومن لم يبلغ الحدَّ لا يُعرض.
	drivers := []ratedParty{}
	rows, err := s.pg.Query(r.Context(), `
		SELECT u.id::text, u.full_name, u.phone::text,
		       count(*), avg(rt.driver_stars)::float8,
		       count(*) FILTER (WHERE rt.driver_stars <= 2)
		FROM order_ratings rt
		JOIN orders o ON o.id = rt.order_id
		JOIN users u  ON u.id = o.driver_id
		WHERE rt.driver_stars IS NOT NULL
		GROUP BY u.id, u.full_name, u.phone
		HAVING count(*) >= $1
		ORDER BY avg(rt.driver_stars), count(*) DESC
		LIMIT 100`, minCount)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for rows.Next() {
		var d ratedParty
		if err := rows.Scan(&d.ID, &d.Name, &d.Phone, &d.Count, &d.Average, &d.Low); err != nil {
			rows.Close()
			s.respondErr(w, err)
			return
		}
		drivers = append(drivers, d)
	}
	rows.Close()

	// ٣ · آخرُ ما كُتب — **ورقمٌ بلا كلامٍ لا يُصلح شيئاً.**
	//
	// **وما فيه تعليقٌ أو نجمتان فأقلّ** — والباقي رضاً صامتاً لا يحتاج قراءة.
	comments := []ratingComment{}
	crows, err := s.pg.Query(r.Context(), `
		SELECT o.number, cu.full_name, COALESCE(dr.full_name, ''),
		       rt.platform_stars, rt.driver_stars, rt.comment, rt.created_at
		FROM order_ratings rt
		JOIN orders o  ON o.id = rt.order_id
		JOIN users cu  ON cu.id = rt.customer_id
		LEFT JOIN users dr ON dr.id = o.driver_id
		WHERE rt.comment <> '' OR rt.platform_stars <= 2
		   OR (rt.driver_stars IS NOT NULL AND rt.driver_stars <= 2)
		ORDER BY rt.created_at DESC
		LIMIT 50`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer crows.Close()
	for crows.Next() {
		var c ratingComment
		if err := crows.Scan(&c.OrderNumber, &c.Customer, &c.Driver,
			&c.Platform, &c.DriverStars, &c.Comment, &c.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		comments = append(comments, c)
	}

	// **و`avg` فارغةٌ حين لا تقييمَ** — ولا تُعرض صفراً: **صفرٌ يُقرأ «سيّئ»
	// و«لا شيءَ بعد» شيءٌ آخر.**
	var average float64
	if avg != nil {
		average = *avg
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"total": total, "average": average, "has_average": avg != nil,
		"low": lowCount, "drivers": drivers, "comments": comments,
	})
}
