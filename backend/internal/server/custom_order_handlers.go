package server

/*
**الطلبُ الخاصّ — بابان: يُطلب، ويُوثَّق ما اتُّفق عليه.**

(قرارُ المالك ٢٠٢٦-٠٨-٠٩.)
*/

import (
	"context"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// handleCreateCustomOrder **الزبونُ يصف ما يريد.**
//
// **ولا أصنافَ ولا سعر** — ولا يُسأل عن متجر: هو يطلب ما ليس في المنصّة.
func (s *Server) handleCreateCustomOrder(w http.ResponseWriter, r *http.Request) {
	// **والطلبُ المخصَّصُ بابٌ مستقلّ** — يُفتح ويُغلق دون العاديّ.
	//
	// **وقبل قراءةِ الجسم** — فلا يُستهلك مفتاحُ تفرّدٍ لبابٍ مغلق.
	if !s.requireLaunch(w, r, launchCustomerCustomOrders) {
		return
	}
	// **والمخصَّصُ يخضع لدوام المنصّة كالعاديّ** — **ومكتبٌ مغلقٌ لا
	// يحضّر طلباً موصوفاً كما لا يحضّر طلباً من متجر.**
	if !s.requireOrdering(w, r) {
		return
	}
	req, err := decode[struct {
		Request     string  `json:"request"`
		AddressText string  `json:"address_text"`
		Lat         float64 `json:"lat"`
		Lng         float64 `json:"lng"`
		// Payment **نقدٌ أو محفظة** — والخصمُ عند التسليم لا عند الطلب.
		Payment string `json:"payment_method"`
		// Notes **ملاحظاتٌ للسائق** — `CAF-07`/`CUST-CUSTOM-019`: كانت
		// تُرسَل ولا تُفكّ فتُهمَل صامتةً، **فيشتري السائقُ بلا تعليماتِ
		// صاحبها.** فتُفكّ الآن وتُخزَّن كالعاديّ (عمودُ `orders.notes`).
		Notes string `json:"notes"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **العملُ وعلامةُ تثبيتِ منع التكرار في معاملةٍ واحدة** — `XG-33`.
	s.WithIdempotentTx(w, r, func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error) {
		o, after, err := s.orders.CreateCustomTx(ctx, q, userIDFrom(r), req.Request,
			req.AddressText, req.Payment, req.Notes, req.Lat, req.Lng)
		if err != nil {
			return IdempotentBody{}, err
		}
		// **وردُّ الخاصّ يُشكَّل كالعاديّ** — **ولا بابَ يُعفى.**
		view, err := orderView(orders.AudienceCustomer, o)
		if err != nil {
			return IdempotentBody{}, err
		}
		return IdempotentBody{Status: http.StatusCreated, Payload: view, AfterCommit: func() {
			// **والبثُّ بعد التثبيت** — بابٌ واحدٌ يبلغ الأطرافَ كلَّها:
			// **العملياتِ التي تنتظر موافقتَها، وصاحبَ الطلب** —
			// **وكان يبلغ المكتبَ وحدَه** (`D22`).
			//
			// **وطلبٌ ينتظر من لا يعلم أنّه ينتظره لا يُخدَم** —
			// **وصاحبُه ينظر إلى شاشةٍ لا تتحرّك.**
			after()
		}}, nil
	})
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
	httpx.JSON(w, http.StatusOK, map[string]any{"agreed": true})
}
