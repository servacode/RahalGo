package server

// ══════════════════════════════════════════════════════════════════════
// **«إحضار»: المبلغُ الحقيقيّ وموافقةُ المتجر، وكودُ الاستلام للمتجر** (قرارُ المالك ٢٠٢٦-١٠-١٦)
// ══════════════════════════════════════════════════════════════════════
//
//	السائق  POST /driver/orders/{id}/fetch-amount            {amount}
//	المتجر  POST /merchant/deliveries/{id}/fetch-amount/approve {expected_version}
//	المتجر  POST /merchant/deliveries/{id}/fetch-amount/reject  {expected_version}
//
// **والمنطقُ في المحرّك** (`orders/fetch_amount.go`) — وهنا الملكيّةُ والجسمُ وحدَهما.

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// handleDriverFetchAmount **السائقُ يكتب ما دفعه بمكان الاستلام.**
func (s *Server) handleDriverFetchAmount(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	if !s.driverOwnsOrder(r, orderID) {
		s.respondErr(w, errNotYourOrder)
		return
	}
	req, err := decode[struct {
		Amount *int64 `json:"amount"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Amount == nil {
		s.respondErr(w, errValidation)
		return
	}
	o, err := s.orders.ProposeFetchAmount(r.Context(), orderID, userIDFrom(r), *req.Amount)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("orders", "ops")
	httpx.JSON(w, http.StatusOK, orders.ViewFor(orders.AudienceDriver, o))
}

// handleMerchantApproveFetchAmount **المتجرُ يوافق على المبلغ.**
func (s *Server) handleMerchantApproveFetchAmount(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		ExpectedVersion *int64 `json:"expected_version"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.merchantDecideFetchAmount(w, r, true, req.ExpectedVersion)
}

// handleMerchantRejectFetchAmount **المتجرُ يرفض المبلغ** — والسائقُ يكتب غيره.
func (s *Server) handleMerchantRejectFetchAmount(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		ExpectedVersion *int64 `json:"expected_version"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.merchantDecideFetchAmount(w, r, false, req.ExpectedVersion)
}

// merchantDecideFetchAmount **والنسخةُ إلزاميّة** — موافقةٌ بلا نسخةٍ تُقفل ما لم يره صاحبُه.
func (s *Server) merchantDecideFetchAmount(w http.ResponseWriter, r *http.Request, approve bool, version *int64) {
	orderID := chi.URLParam(r, "id")
	var merchantID, kind string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT COALESCE(merchant_id::text, ''), kind FROM orders WHERE id = $1`, orderID).
		Scan(&merchantID, &kind); err != nil || kind != orders.KindMerchantDelivery {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if version == nil {
		s.respondErr(w, errValidation)
		return
	}
	o, err := s.orders.DecideFetchAmount(r.Context(), orderID, merchantID, userIDFrom(r),
		approve, *version)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("orders", "ops")
	view, err := orderView(orders.AudienceMerchantDelivery, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.attachFetchCodes(r.Context(), []map[string]any{view}); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, view)
}

// attachFetchCodes **كودُ الاستلام للمتجر في «إحضار»** (قرارُ المالك ٢٠٢٦-١٠-١٦) — يُلحَق
// بحمولة توصيلاته هو وحدَها، **فهو المستلِمُ فيها**؛ ولا يُلحَق في «توصيل» أبداً (الكودُ
// هناك للمستلِم). **ولا يُرسَل للسائق في أيّ حال** — يطلبه منه عند الوصول.
func (s *Server) attachFetchCodes(ctx context.Context, views []map[string]any) error {
	ids := make([]string, 0, len(views))
	for _, v := range views {
		if v["delivery_mode"] != orders.DeliveryModeFetch {
			continue
		}
		if id, ok := v["id"].(string); ok {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := s.pg.Query(ctx, `
		SELECT id::text, delivery_code FROM orders
		 WHERE id = ANY($1::uuid[]) AND kind = 'merchant_delivery' AND delivery_mode = 'fetch'
		   AND delivery_code IS NOT NULL
		   AND status IN ('picked_up', 'on_the_way', 'at_dropoff')`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	codes := map[string]string{}
	for rows.Next() {
		var id, code string
		if err := rows.Scan(&id, &code); err != nil {
			return err
		}
		codes[id] = code
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, v := range views {
		if id, ok := v["id"].(string); ok {
			if c, ok := codes[id]; ok {
				v["delivery_code"] = c
			}
		}
	}
	return nil
}
