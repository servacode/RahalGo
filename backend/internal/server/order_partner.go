package server

// **حوّله لمتجرٍ تابعٍ إلنا** — قرارُ المالك ٢٠٢٦-١٠-١٥.
//
// **البابُ رقيقٌ والمنطقُ في المحرّك** (`orders.LinkPartner`): الشروطُ والنسبةُ الملقوطةُ
// والكتابة — **والتدقيقُ في معاملته نفسِها.** والإرسالُ للمتجر على واتساب بالبابين
// القائمين (`/message` و`/whatsapp`) — **فلا قناةٌ ثانيةٌ تفترق عن الأولى.**

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// handleLinkPartner `POST /admin/orders/{id}/partner` — `{"merchant_id": "…"}`.
func (s *Server) handleLinkPartner(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		MerchantID string `json:"merchant_id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.MerchantID == "" {
		s.respondErr(w, errValidation)
		return
	}
	s.changePartner(w, r, req.MerchantID, "ops.order_partner_linked")
}

// handleUnlinkPartner `DELETE /admin/orders/{id}/partner` — «إلغاء الربط».
func (s *Server) handleUnlinkPartner(w http.ResponseWriter, r *http.Request) {
	s.changePartner(w, r, "", "ops.order_partner_unlinked")
}

func (s *Server) changePartner(w http.ResponseWriter, r *http.Request, merchantID, action string) {
	orderID := chi.URLParam(r, "id")
	o, _, err := s.orders.LinkPartner(r.Context(), userIDFrom(r), orderID, merchantID,
		func(ctx context.Context, q dbtx.Querier, ch orders.PartnerChange) error {
			return s.auditTx(ctx, q, r, action, "order", orderID, map[string]any{
				"merchant_id": ch.MerchantID, "merchant_name": ch.MerchantName,
				"commission_percent":   ch.CommissionPercent,
				"previous_merchant_id": ch.PreviousMerchantID,
			})
		})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("order", "ops")
	view, err := orderView(orders.AudienceOps, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"order": view})
}
