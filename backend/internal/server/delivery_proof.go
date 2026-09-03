package server

// **إثباتُ التسليم** — صورةٌ وإحداثياتٌ ووقت.
//
// # المسألة
//
// قاعدتُنا: **«الزبونُ يُصدَّق أوّلَ مرّة»** — والمنصةُ تتحمّل. **وقاعدةٌ بلا
// دليلٍ تكلفةٌ بلا سقف**: من عرف أنّ كلمتَه تكفي قالها مرّةً بعد مرّة، **ولا
// يبقى للسائق الصادق ما يدفع به عن نفسه.**
//
// **والمشكلةُ ليست في تصديق الزبون، هي في أنّ لا شيءَ يُقاس عليه.**
//
// # والمعيارُ العالميّ ثلاثةٌ لا واحد
//
//	الصورة       ←  موضعُ التسليم لا الطردُ وحدَه
//	الإحداثيات   ←  تُلتقط آلياً فتثبت أنّه كان هناك
//	الوقت        ←  آليٌّ كذلك — لا يُكتب بيد
//
// **وأهمُّها الإحداثيات**: صورةُ بابٍ قد تكون لأيّ باب، **وصورةٌ بإحداثياتٍ
// على بُعد أمتارٍ من عنوان الزبون بيّنة.**
//
// # ولا يقف التسليمُ على كاميرا
//
// هاتفٌ لا يعمل، أو إذنٌ مرفوض، أو ليلٌ لا يُرى فيه شيء — **وسائقٌ لا يستطيع
// إنهاء طلبٍ سلّمه فعلاً يقف في الشارع.** فيُتاح التخطّي **بكلمةٍ تُكتب
// وتُقرأ يومَ النزاع**: من تخطّى عشراً يُقرأ ذلك في صفّه، ومن تخطّى مرّةً لا
// يُلام.

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var errProofRequired = httpx.NewError(http.StatusConflict,
	"delivery_proof_required", "errors.delivery_proof_required")

// handleDeliveryProof يحفظ صورةَ التسليم وموضعَها.
//
// **تُرفع قبل «سُلّم» لا بعده**: بعد الإغلاق يصير الطلبُ تاريخاً، **وصورةٌ
// تُضاف إلى تاريخٍ مغلقٍ تُقرأ إضافةً متأخّرة** — وهي أضعفُ ما يُحتجّ به.
func (s *Server) handleDeliveryProof(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	if !s.driverOwnsOrder(r, orderID) {
		s.respondErr(w, errNotYourOrder)
		return
	}

	// **الصورةُ تُرفع كسائر الوسائط** — بالفحص والحدّ نفسِهما.
	file, _, err := r.FormFile("file")
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	defer func() { _ = file.Close() }()

	md, err := s.media.Save(r.Context(), userIDFrom(r), "delivery_proof", file)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// **الإحداثياتُ تأتي مع الصورة لا بعدها.**
	//
	// موضعٌ يُرسَل في نداءٍ ثانٍ **قد يُرسَل من مكانٍ آخر** — والسائقُ يتحرّك.
	lat, errLat := strconv.ParseFloat(r.FormValue("lat"), 64)
	lng, errLng := strconv.ParseFloat(r.FormValue("lng"), 64)
	hasPoint := errLat == nil && errLng == nil

	// ══════════════════════════════════════════════════════════════════
	// **وموقعٌ مزيَّفٌ لا يُقبل إثباتاً**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قِيس 2026-09-02: لا فحصَ للتزييف في المنصّة كلِّها.)
	//
	// **وإثباتُ التسليم يقول «على بعد خمسةِ أمتارٍ من العنوان»**
	// محسوبةً من النقطة التي يرسلها الجهازُ نفسُه. **فمن زيّف موضعَه
	// كتب الإثباتَ بيده** — والحارسُ الذي بُني للحماية يشهد له.
	//
	// **فتُرفض النقطةُ ولا تُرفض الصورة**: الصورةُ وقعت وقد تكون
	// صادقة، **والموضعُ وحدَه كذب.** فيُحفظ الإثباتُ بلا موضعٍ
	// موسوماً بالتزييف، **ويقرؤه المكتبُ فيعلم.**
	mocked := r.FormValue("mocked") == "true"
	if mocked {
		hasPoint = false
	}

	q := `UPDATE orders SET pod_media_id = $2, pod_taken_at = now(), pod_skip_reason = '',
	                       pod_mocked = ` + boolLit(mocked)
	args := []any{orderID, md.ID}
	if hasPoint {
		q += `, pod_at = ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography`
		args = append(args, lng, lat)
	}
	q += ` WHERE id = $1`
	if _, err := s.pg.Exec(r.Context(), q, args...); err != nil {
		s.respondErr(w, err)
		return
	}

	s.audit(r, "driver.delivery_proof", "order", orderID, map[string]any{
		"media": md.ID, "located": hasPoint, "mocked": mocked,
	})
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{
		"media_id": md.ID, "located": hasPoint,
	})
}

// handleSkipDeliveryProof يُسجّل تعذّرَ الصورة بسببه.
//
// **والسببُ إلزاميّ**: تخطٍّ بلا كلمةٍ لا يُقرأ يومَ النزاع، **ولا يُفرَّق بين
// من عطبت كاميرتُه ومن لم يشأ.**
func (s *Server) handleSkipDeliveryProof(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	if !s.driverOwnsOrder(r, orderID) {
		s.respondErr(w, errNotYourOrder)
		return
	}
	req, err := decode[struct {
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		s.respondErr(w, errReasonRequired)
		return
	}
	if _, err := s.pg.Exec(r.Context(),
		`UPDATE orders SET pod_skip_reason = $2 WHERE id = $1`,
		orderID, clip(reason, 200)); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "driver.delivery_proof_skipped", "order", orderID,
		map[string]any{"reason": reason})
	httpx.JSON(w, http.StatusOK, map[string]any{"skipped": true})
}

// requireProofBeforeDelivery يمنع «سُلّم» بلا إثباتٍ ولا سببٍ لتخطّيه.
//
// **والحارسُ في الخادم لا في الشاشة**: زرٌّ يُخفى يُلتفّ عليه، **وقاعدةٌ
// تُفرض في المحرّك لا.**
func (s *Server) requireProofBeforeDelivery(r *http.Request, orderID string) error {
	if !s.settings.GetBool(r.Context(), "drivers.require_delivery_photo") {
		return nil
	}
	var mediaID *string
	var skip string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT pod_media_id::text, pod_skip_reason FROM orders WHERE id = $1`,
		orderID).Scan(&mediaID, &skip); err != nil {
		return err
	}
	if mediaID == nil && strings.TrimSpace(skip) == "" {
		return errProofRequired
	}
	return nil
}

// boolLit **حرفٌ منطقيٌّ في نصّ الاستعلام** — لا وسيطٌ مرقَّم.
//
// **والوسائطُ هنا مرقّمةٌ بترتيبٍ يتبدّل** (`$3` و`$4` تُضافان مع
// الموضع وحدَه)، **ومن أقحم وسيطاً في المنتصف بدّل ما بعده.** والقيمةُ
// منطقيّةٌ من مقارنةٍ عندنا لا من مدخلٍ خارجيّ، **فلا حقنَ فيها.**
func boolLit(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
