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
	"time"

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
		// **ورقمٌ عشريٌّ للدولار** (٢٠٢٦-١٠-٢٠) — والعملةُ عملةُ الطلب لا ما يقوله السائق.
		Amount *float64 `json:"amount"`
		// Currency **عملةُ ثمن طلب الواتساب يختارها السائق** (قرارُ المالك ٢٠٢٦-١١-٠٩) — `SYP` أو
		// `USD`. **وفي «إحضار» تُهمَل**: العملةُ عملةُ المتجر. وغيابُها عملةُ الطلب كما كانت.
		Currency string `json:"currency"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Amount == nil {
		s.respondErr(w, errValidation)
		return
	}
	if req.Currency != "" && req.Currency != orders.CurrencySYP && req.Currency != orders.CurrencyUSD {
		s.respondErr(w, errValidation)
		return
	}
	var currency, origin string
	_ = s.pg.QueryRow(r.Context(), `SELECT amount_currency, origin FROM orders WHERE id::text = $1`, orderID).
		Scan(&currency, &origin)
	if origin != orders.OriginWhatsApp {
		req.Currency = ""
	}
	if req.Currency != "" {
		currency = req.Currency
	}
	units, ok := orders.AmountUnits(*req.Amount, currency)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	o, err := s.orders.ProposeFetchAmountCurrency(r.Context(), orderID, userIDFrom(r), units, req.Currency)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **وطلبُ الواتساب: الزبونُ يُخبَر بالمبالغ** (قرارُ المالك ٢٠٢٦-١١-٠٦) — إعلامٌ بلا موافقة،
	// **ولا يُنتظر**: رسالةٌ لم تصل لا توقف السائق.
	if o.Origin == orders.OriginWhatsApp {
		s.sendWhatsAppGoodsNotice(context.WithoutCancel(r.Context()), orderID)
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

// sendWhatsAppGoodsNotice **يرسل رسالةَ المبالغ لزبون طلب الواتساب** — بانتظاره كرسالة الموقع.
func (s *Server) sendWhatsAppGoodsNotice(ctx context.Context, orderID string) {
	phone, text, ok, err := s.orders.WhatsAppGoodsNotice(ctx, orderID)
	if err != nil {
		s.logger.Warn("طلب واتساب: قراءةُ رسالة المبالغ سقطت", "order", orderID, "error", err)
		return
	}
	if !ok {
		return
	}
	sendCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if !s.sendText(sendCtx, phone, text) {
		s.logger.Warn("طلب واتساب: رسالةُ المبالغ لم تصل للزبون", "order", orderID)
	}
}
