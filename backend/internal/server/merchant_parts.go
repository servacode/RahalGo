package server

// ══════════════════════════════════════════════════════════════════════
// **بابُ المتجر في الطلب بعدّة محطّات — كلُّ متجرٍ يجيب عن قسمه** (§3.3)
// ══════════════════════════════════════════════════════════════════════
//
// **تطبيقُ المتجر المنشورُ (0.11 و0.12) وموقعُه يرسلان ما اعتادا** — `accepted` ·
// `rejected` · `preparing` · `cancelled` على الطلب كلِّه. **والخادمُ يوجّهها إلى
// أقسام صاحبها في الطلب** — **فلا تحديثَ لتطبيقٍ ولا يقرّر متجرٌ عن غيره** (G-04):
//
//	accepted   ⇒ قبولُ قسمه بمدّة تحضيره
//	rejected   ⇒ رفضُ قسمه — والسببُ إلزاميّ
//	preparing  ⇒ لا شيء: قبولُ القسم هو بدءُ التحضير
//	cancelled  ⇒ رفضُ قسمه أو شيلُه بذنبه حتّى «قيد التحضير»؛ وبعدها **الشيلُ للمكتب** (`use_part`)
//
// **ومالكٌ له فرعان في الطلب يجيب عنهما معاً.** **والردُّ حمولتُه هو** — قسمُه وماله.

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// merchantPartTransition **يوجّه فعلَ المتجر إلى أقسامه** ثمّ يردّ حمولتَه.
func (s *Server) merchantPartTransition(w http.ResponseWriter, r *http.Request, orderID string,
	stores []string, to, note string, prep *int) {
	if len(stores) == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx, actor := r.Context(), userIDFrom(r)
	roles := []string{"merchant"}
	var err error
	switch to {
	case orders.StAccepted:
		_, err = s.orders.AnswerParts(ctx, actor, roles, orderID, stores, true, prep, "")
	case orders.StRejected:
		_, err = s.orders.AnswerParts(ctx, actor, roles, orderID, stores, false, nil, note)
	case orders.StPreparing:
		// **قبولُ القسم هو بدءُ تحضيره** — وزرُّ «ابدأ التحضير» القديمُ يمرّ بلا أثر.
	case orders.StCancelled:
		// **«لا» على قسمٍ لم يُجَب رفضٌ له** — **وعلى قسمٍ قبله شيلٌ له بذنبه** (`RemovePart`) ما دام
		// الطلبُ «بانتظار/مقبول/قيد التحضير»؛ **وبعدها الشيلُ للمكتب** (`use_part`) — نافذةُ إلغاء
		// المتجر لطلبه العاديّ نفسُها.
		_, err = s.orders.AnswerParts(ctx, actor, roles, orderID, stores, false, nil, note)
		if errors.Is(err, orders.ErrBadTransition) {
			for _, id := range stores {
				if _, err = s.orders.RemovePart(ctx, actor, roles, orderID, id, note,
					orders.FaultMerchant, nil); err != nil {
					break
				}
			}
		}
		if errors.Is(err, orders.ErrBadTransition) {
			err = orders.ErrUsePart
		}
	default:
		err = orders.ErrBadTransition
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	o, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	view, err := s.merchantOrderView(r, o, true)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, view)
}

// handleMerchantPartTransition **بابُ القسم الصريح** — `POST /merchant/orders/{id}/parts/{storeID}/transition`.
//
// **للتطبيقات القادمة**: مالكٌ له فرعان في الطلب يجيب عن واحدٍ بعينه.
func (s *Server) handleMerchantPartTransition(w http.ResponseWriter, r *http.Request) {
	orderID, storeID := chi.URLParam(r, "id"), chi.URLParam(r, "storeID")
	req, err := decode[struct {
		To          string `json:"to"`
		Note        string `json:"note"`
		PrepMinutes *int   `json:"prep_minutes"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if (req.To == orders.StCancelled || req.To == orders.StRejected) && strings.TrimSpace(req.Note) == "" {
		s.respondErr(w, errReasonRequired)
		return
	}
	mine, err := s.orders.OwnerPartStores(r.Context(), orderID, userIDFrom(r))
	if err != nil || !isUUID(orderID) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	for _, id := range mine {
		if strings.EqualFold(id, storeID) {
			s.merchantPartTransition(w, r, orderID, []string{id}, req.To, req.Note, req.PrepMinutes)
			return
		}
	}
	s.respondErr(w, httpx.ErrNotFound)
}
