package server

import (
	"context"
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// كشف سمعة المستخدم (موظف): تقييمه واتجاهه، والشكاوى بحقّه، والتقييمات والتعليقات
// التي تلقّاها — كي يرى كلٌّ ما قُدّم بحقّه بشفافية. حسب دوره (متجر/سائق/مندوب).

type repReview struct {
	OrderNumber  int64  `json:"order_number"`
	MerchantName string `json:"merchant_name"`
	// CustomerName من قيّم — **وهو من فتح البابَ له**.
	//
	// كان التقييمُ يصل بلا اسم: نجومٌ ورقمُ طلب. **ومن نال ثلاثاً لا يعرف
	// أيَّ بابٍ كان** فلا يتعلّم منها شيئاً — والتقييمُ الذي لا يُنسَب إلى
	// واقعةٍ يُقرأ حكماً عامّاً على شخصه.
	//
	// **والشكوى تبقى بلا اسم** (`ReputationComplaints`): تلك خصومةٌ تُحقَّق،
	// وكشفُ صاحبها يفتح باباً لمن يريد أن يردّ عليه. **وهذا ثناءٌ أو ملاحظةٌ
	// على خدمةٍ وقف فيها أمامه.** (قرارُ المالك ٢٠٢٦-٠٨-٠٥.)
	CustomerName string    `json:"customer_name"`
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
	//  - السائق: طلبات سلّمها، وعمود driver_stars
	//  - المتجر: طلبات متاجره — **بلا نجوم**، انظر أدناه
	//
	// المندوب **ليس منهما عمداً**: عمله جلب العملاء وقبض العمولة، ولا أحد
	// يقيّمه. كان يُعرض له تقييم متاجره باسم "تقييمي" — رقم لا يقيس عمله ولا
	// يملك تغييره. من لا سمعة له يُعاد له كشف فارغ لا كشف غيره.
	// ownerCond: شرط ربط الطلب بالمستخدم؛ starCol: عمود النجوم المعني.
	//
	// **والمتجرُ بلا نجومٍ عمداً.**
	//
	// كان يُعرض له متوسّطُ `merchant_stars` — **وهي صارت نجمةَ المنصة**: الزبونُ
	// لا يرى اسمَ متجرٍ ولا يختاره، فما حكَم عليه هو خدمتُنا كلُّها. **ومتجرٌ
	// يُحاسَب على تأخيرٍ سببُه سائقُنا يُظلم**، وآخرُ يُمدح على سرعةٍ سببُها
	// قربُ العنوان.
	//
	// **ولا يُترك له الرقمُ القديمَ ولو تغيّر معناه**: رقمٌ يراه في شاشته يصدّقه
	// ويقيس عليه، **ومعنًى انقلب في الخلفية لا يبلغه.**
	//
	// **وما يبقى له أصدق**: الشكاوى على طلباته — واقعةٌ بواقعة، لا متوسّطٌ
	// يخلط ما يملكه بما لا يملكه.
	// **والزبونُ آخرُهم لأنّ الأدوارَ تجتمع.**
	//
	// **وكلُّهم زبائنُ أيضاً**: للسائق حسابٌ يطلب به، وللمتجر كذلك. **فلو
	// سُئل عن الزبون أوّلاً لَرأى السائقُ بلاغاتِ طلباته لا تقييمَ عمله.**
	//
	// # ولا نجومَ للزبون
	//
	// **لا أحدَ يقيّمه**: في `order_ratings` نجمتان — للمنصة وللسائق، **ولا
	// ثالثةَ له.** فبطاقةُ «٠٫٠ من ٥» في شاشته **تُقرأ حكماً عليه** وهي لا
	// تقيس شيئاً — و`Rated` تبقى كاذبةً فلا تُعرض.
	//
	// **وما يُكتب عنه بلاغاتٌ لا تقييم** — وهي وحدَها ما يستحقّ أن يراه.
	var ownerJoin, ownerCond, starCol string
	// complaintCond ما يُستثنى من كشفه — **افتراضُه لا شيء.**
	complaintCond := ``
	switch {
	case has("merchant"):
		ownerJoin = `JOIN merchants mm ON mm.id = o.merchant_id`
		ownerCond = `mm.owner_user_id = $1`
	case has("driver"):
		ownerJoin = ``
		ownerCond = `o.driver_id = $1`
		starCol = `rt.driver_stars`
	default:
		ownerCond = `o.customer_id = $1`
		// **وشكواه هو ليست سمعتَه.**
		//
		// **وله صفحتُها** (`‎/complaints`) — ولو ظهرت هنا أيضاً **لَقرأ ما
		// اشتكى منه شكوى عليه**، وعدَّ نفسَه متّهماً بما هو صاحبُ الحقّ فيه.
		complaintCond = ` AND NOT t.opened_by_customer`
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
		// Rated هل لهذا الدور نجومٌ أصلاً — **فبطاقةٌ فارغةٌ تُقرأ صفراً.**
		//
		// «٠٫٠ من ٥» في شاشةِ من لا يُقيَّم أسوأُ من غياب البطاقة: **يقرؤها
		// حكماً عليه** فيسأل عمّا فعل، ولم يفعل شيئاً.
		Rated bool `json:"rated"`
	}{Complaints: []repComplaint{}, Reviews: []repReview{}, Rated: starCol != ""}

	// التقييم: المتوسط والعدد، ومتوسط آخر 30 يوماً لاستنتاج الاتجاه.
	if starCol != "" {
		s.fillRating(ctx, uid, starCol, ownerJoin, ownerCond, &out.Rating, &out.Reviews)
	}
	out.Rating.Trend = trendOf(out.Rating.Count, out.Rating.Avg, out.Rating.RecentAvg)

	// ── الشكاوى بحقّه — **بحقّه هو لا بحقّ الطلب** ─────────────────────
	//
	// (شكوى المالك ٢٠٢٦-٠٨-٠٩: «مين ضد مين وكلّ شخص ياخذ حقّه».)
	//
	// **كان الشرطُ «تذكرةٌ على طلبٍ قاده» أو «على طلبٍ من متجره»** — فشكوى
	// على السائق يقرؤها صاحبُ المتجر وكأنّها عليه، **والعكس.** ويعدّ نفسَه
	// متّهماً بما لا شأنَ له به، ولا يجد ما يصحّحه.
	//
	// **وصار الشرطُ المشتكى عليه نفسَه** — يُشتقّ من سبب الشكوى عند فتحها.
	//
	// **والقديمةُ بلا مشتكًى عليه تبقى للجميع**: صفوفٌ سبقت الهجرة، **وإخفاؤها
	// عن الطرفين يمحو تاريخاً** بدل أن يوزّعه. (`against_user_id IS NULL`
	// تعني «على المنصّة» أو «قبل أن يُعرف».)
	crows, err := s.pg.Query(ctx, `
		SELECT t.number, o.number, t.subject, t.status, t.created_at
		FROM tickets t
		JOIN orders o ON o.id = t.order_id
		`+ownerJoin+`
		WHERE `+ownerCond+complaintCond+`
		  AND (t.against_user_id IS NULL OR t.against_user_id = $1)
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

// trendOf اتّجاهُ التقييم من متوسّطه العامّ ومتوسّط شهره الأخير.
func trendOf(count int, avg, recent float64) string {
	switch {
	case count == 0 || recent == 0:
		return "flat"
	case recent > avg+0.1:
		return "up"
	case recent < avg-0.1:
		return "down"
	}
	return "flat"
}

// rating الشكلُ المشترك — يُمرَّر بالمرجع كي تُملأ حقولُه.
type ratingOut = struct {
	Avg       float64 `json:"avg"`
	Count     int     `json:"count"`
	RecentAvg float64 `json:"recent_avg"`
	Trend     string  `json:"trend"`
}

// fillRating نجومُ من يُقيَّم — ولا تُنادى لمن لا يُقيَّم.
func (s *Server) fillRating(ctx context.Context, uid, starCol, ownerJoin, ownerCond string,
	out *ratingOut, reviews *[]repReview) {
	_ = s.pg.QueryRow(ctx, `
		SELECT COALESCE(avg(`+starCol+`), 0), count(`+starCol+`),
		       COALESCE(avg(`+starCol+`) FILTER (WHERE rt.created_at > now() - interval '30 days'), 0)
		FROM order_ratings rt JOIN orders o ON o.id = rt.order_id `+ownerJoin+`
		WHERE `+ownerCond+` AND `+starCol+` IS NOT NULL`, uid).
		Scan(&out.Avg, &out.Count, &out.RecentAvg)

	// التقييمات والتعليقات المتلقّاة (نُظهر ذوات التعليق أولاً).
	rows, err := s.pg.Query(ctx, `
		SELECT o.number, m.name, COALESCE(NULLIF(cu.full_name, ''), cu.phone::text),
		       `+starCol+`, COALESCE(rt.comment, ''), rt.created_at
		FROM order_ratings rt
		JOIN orders o ON o.id = rt.order_id
		JOIN merchants m ON m.id = o.merchant_id
		JOIN users cu ON cu.id = rt.customer_id
		`+ownerJoin+`
		WHERE `+ownerCond+` AND `+starCol+` IS NOT NULL
		ORDER BY (rt.comment <> '') DESC, rt.created_at DESC LIMIT 50`, uid)
	if err == nil {
		for rows.Next() {
			var rv repReview
			if rows.Scan(&rv.OrderNumber, &rv.MerchantName, &rv.CustomerName,
				&rv.Stars, &rv.Comment, &rv.CreatedAt) == nil {
				*reviews = append(*reviews, rv)
			}
		}
		rows.Close()
	}
}
