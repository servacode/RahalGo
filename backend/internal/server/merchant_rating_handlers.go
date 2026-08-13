package server

// تقييمُ السائق للمتجر — **والطرفُ الذي لا يُقيَّم لا يُقاس.**
//
// # ولماذا السائقُ هو من يقيّمه
//
// الزبونُ يرى الطعامَ ولا يرى المطبخ. **والسائقُ يقف عند بابه كلَّ يوم**:
// يعرف من يجهّز في خمس دقائق ومن يُبقيه نصفَ ساعة، **ومن يعتذر ومن يصرخ.**
//
// # ونجمتان لا واحدة
//
// **«المتجر» و«التعامل» شيئان يفترقان**: مطعمٌ سريعٌ فظّ، وآخرُ بطيءٌ مهذّب.
// **ونجمةٌ واحدةٌ تجمعهما تُخفي أيَّهما المشكلة** — فلا يُعرف أنُكلّم المطبخَ
// أم صاحبَ المحلّ.

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	errNotPickedUp = httpx.NewError(http.StatusConflict,
		"never_picked_up", "errors.never_picked_up")
	errAlreadyRated = httpx.NewError(http.StatusConflict,
		"already_rated", "errors.already_rated")
)

// errNoMerchant **لا متجرَ في هذا الطلب** — الخاصُّ يشتريه السائقُ بنفسه.
var errNoMerchant = httpx.NewError(http.StatusConflict, "no_merchant", "errors.no_merchant")

// handleDriverRateMerchant يسجّل تقييمَ السائق لمتجر طلبٍ نفّذه.
func (s *Server) handleDriverRateMerchant(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	req, err := decode[struct {
		SpeedStars   int    `json:"speed_stars"`
		ConductStars int    `json:"conduct_stars"`
		Comment      string `json:"comment"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.SpeedStars < 1 || req.SpeedStars > 5 ||
		req.ConductStars < 1 || req.ConductStars > 5 {
		s.respondErr(w, errValidation)
		return
	}

	// **ولا يقيّم إلّا من وقف عند بابه فعلاً.**
	//
	// **والشرطُ في الخادم لا في الشاشة**: شاشةٌ تُخفي الزرَّ لا تمنع من ينادي
	// الواجهةَ مباشرةً — **وهي عائلةُ «قاعدةٌ تُطبَّق في العرض» التي تكرّرت.**
	// **وفارغُه طلبٌ خاصّ** — وكان يُقرأ في نصٍّ غيرِ قابلٍ للفراغ
	// **فيسقط الفحصُ ويُردّ «غير موجود»**: رسالةٌ تقول للسائق إنّ طلبَه
	// ذهب، **وهو في يده.**
	var merchantID string
	var reached bool
	if err := s.pg.QueryRow(r.Context(), `
		SELECT COALESCE(o.merchant_id::text, ''),
		       (o.picked_up_at IS NOT NULL
		        OR o.status = 'failed' AND o.fault = 'merchant')
		FROM orders o WHERE o.id = $1 AND o.driver_id = $2`,
		orderID, userIDFrom(r)).Scan(&merchantID, &reached); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if merchantID == "" {
		// **ولا متجرَ في الطلب الخاصّ** — فلا يُقيَّم.
		s.respondErr(w, errNoMerchant)
		return
	}
	if !reached {
		s.respondErr(w, errNotPickedUp)
		return
	}

	tag, err := s.pg.Exec(r.Context(), `
		INSERT INTO merchant_ratings
		    (order_id, driver_id, merchant_id, speed_stars, conduct_stars, comment)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (order_id) DO NOTHING`,
		orderID, userIDFrom(r), merchantID,
		req.SpeedStars, req.ConductStars, clip(req.Comment, 500))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والقاعدةُ تمنع التكرار لا فحصٌ قبله** — ضغطتان متزامنتان تمرّان كلتاهما
	// لو فُحص ثمّ أُدرج.
	if tag.RowsAffected() == 0 {
		s.respondErr(w, errAlreadyRated)
		return
	}

	// **والمتجرُ يرى ما قيل فيه** — ومن قُيّم ولا يعلم لا يُصلح شيئاً.
	s.touch("rating", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"rated": true})
}
