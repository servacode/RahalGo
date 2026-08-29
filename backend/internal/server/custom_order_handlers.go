package server

/*
**الطلبُ الخاصّ — بابان: يُطلب، ويُوثَّق ما اتُّفق عليه.**

(قرارُ المالك ٢٠٢٦-٠٨-٠٩.)
*/

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// handleCreateCustomOrder **الزبونُ يصف ما يريد.**
//
// **ولا أصنافَ ولا سعر** — ولا يُسأل عن متجر: هو يطلب ما ليس في المنصّة.
func (s *Server) handleCreateCustomOrder(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Request     string  `json:"request"`
		AddressText string  `json:"address_text"`
		Lat         float64 `json:"lat"`
		Lng         float64 `json:"lng"`
		// Payment **نقدٌ أو محفظة** — والخصمُ عند التسليم لا عند الطلب.
		Payment string `json:"payment_method"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	o, err := s.orders.CreateCustom(r.Context(), userIDFrom(r), req.Request,
		req.AddressText, req.Payment, req.Lat, req.Lng)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والعملياتُ تُخبَر فوراً** — الطلبُ الخاصُّ ينتظر موافقتَها، **وطلبٌ
	// ينتظر من لا يعلم أنّه ينتظره لا يُخدَم.**
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusCreated, o)
}

// handleAgreeCustom **السائقُ يوثّق ما اتّفق عليه مع الزبون.**
//
// **بعد المحادثة لا قبلها**: يقول له الثمنَ فيوافق، ويقول له الأجرةَ فيوافق،
// **ثمّ يُوثَّق.** (قرارُ المالك.)
func (s *Server) handleAgreeCustom(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Goods int64 `json:"goods_amount"`
		Fee   int64 `json:"fee"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	orderID := chi.URLParam(r, "id")
	if err := s.orders.AgreeCustom(r.Context(), orderID, userIDFrom(r),
		req.Goods, req.Fee); err != nil {
		s.respondErr(w, err)
		return
	}

	// **ويصل الزبونَ ما وُثّق باسمه.**
	//
	// **واتّفاقٌ لا يراه صاحبُه ليس اتّفاقاً**: قيل له في المحادثة، **ويبقى
	// مكتوباً حيث يراه** — فمن نسي رجع إليه، ومن خولف احتجّ به.
	var customerID string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT customer_id::text FROM orders WHERE id = $1`, orderID).
		Scan(&customerID); err == nil {
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: customerID, Kind: notifications.KindOrder,
			Title: notifTitles.customAgreed, Body: "",
			Entity: "order", EntityID: orderID, Href: "/portal/orders",
		})
	}
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"agreed": true})
}
