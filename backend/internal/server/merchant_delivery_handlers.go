package server

// ══════════════════════════════════════════════════════════════════════
// **«لدي توصيلة» — أبوابُ صاحب المتجر** (الخطوة ١٨، ٢٠٢٦-١٠-٠١)
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٩-٢٩ — والعقدُ في `docs/DELIVERY-MONEY-CONTRACT.md`.)
//
// **كان المنطقُ مبنيّاً في `orders` ولا بابَ إليه** — فالميزةُ غائبةٌ عن كلّ
// تطبيق. **وأربعةُ أبواب**: عرضُ السعر قبل الطلب · الإنشاء · القائمة ·
// الإلغاء قبل الاستلام.
//
// **وكلُّها بتحقّق الملكيّة نفسِه** (`ownsMerchant`) — **ومن طلب سائقاً لمتجرٍ
// غيرِ متجره خصم من محفظةِ غيره.**

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/textguard"
)

// handleMerchantDeliveryQuote **أجرةُ النقطة ومقدرةُ المتجر** — قبل الإرسال.
func (s *Server) handleMerchantDeliveryQuote(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	// **والمنصّةُ في دوامها وغيرُ موقوفة** — بوّابةُ طلب الزبون نفسُها
	// (قرارُ المالك ٢٠٢٦-١٠-٠١: «نحمي الخطوة إذا كانت المنصّة خارج أوقات العمل»).
	if !s.requireOrdering(w, r) {
		return
	}
	// **والنقطةُ اختياريّة** — بلاها فالأجرةُ من موقع المتجر.
	var lat, lng float64
	has := r.URL.Query().Get("lat") != "" || r.URL.Query().Get("lng") != ""
	if has {
		var err1, err2 error
		lat, err1 = strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
		lng, err2 = strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
		if err1 != nil || err2 != nil {
			s.respondErr(w, errValidation)
			return
		}
	}
	q, err := s.orders.QuoteMerchantDelivery(r.Context(), merchantID, lat, lng, has)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, q)
}

// handleMerchantCreateDelivery **يطلب سائقاً لغرضٍ جاهز** — ومرّةً واحدة.
//
// **والإنشاءُ وعلامةُ منع التكرار في معاملةٍ واحدة** (`WithIdempotentTx`):
// توصيلةٌ أُنشئت وضاع ردُّها فأُعيدت **لا تُنشأ ثانيةً ولا يُخصم أجرُها مرّتين.**
func (s *Server) handleMerchantCreateDelivery(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	// **والمنصّةُ في دوامها وغيرُ موقوفة** — بوّابةُ طلب الزبون نفسُها
	// (قرارُ المالك ٢٠٢٦-١٠-٠١: «نحمي الخطوة إذا كانت المنصّة خارج أوقات العمل»).
	if !s.requireOrdering(w, r) {
		return
	}
	req, err := decode[struct {
		RecipientName  string `json:"recipient_name"`
		RecipientPhone string `json:"recipient_phone"`
		AddressText    string `json:"address_text"`
		// **والنقطةُ اختياريّة** — فراغُها «لا عنوانَ على الخريطة».
		Lat        *float64 `json:"lat"`
		Lng        *float64 `json:"lng"`
		ParcelNote string   `json:"parcel_note"`
		DriverNote string   `json:"driver_note"`
		FeePayer   string   `json:"fee_payer"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **ونصوصُ المستلِم تمرّ بالحارس** — انظر `text_limits.go`.
	if _, err := s.guardText(r.Context(),
		tf("recipient_name", &req.RecipientName, maxPersonName, textguard.Name),
		tf("address_text", &req.AddressText, maxAddressText, textguard.Address),
		tf("parcel_note", &req.ParcelNote, maxOrderNotes, textguard.Notes),
		tf("driver_note", &req.DriverNote, maxOrderNotes, textguard.Notes),
	); err != nil {
		s.respondErr(w, err)
		return
	}
	// **ورقمُ المستلِم يُطبَّع كرقم أيّ حساب** — يتّصل به السائقُ من الشارع،
	// **ورقمٌ ناقصٌ يُكتشف عند الباب لا عند الطلب.**
	phone, ok := identity.NormalizePhone(req.RecipientPhone)
	if !ok {
		s.respondErr(w, identity.ErrInvalidPhone)
		return
	}
	in := orders.MerchantDeliveryInput{
		RecipientName: req.RecipientName, RecipientPhone: phone,
		AddressText: req.AddressText,
		ParcelNote:  req.ParcelNote, DriverNote: req.DriverNote, FeePayer: req.FeePayer,
		HasPoint: req.Lat != nil && req.Lng != nil,
	}
	if in.HasPoint {
		in.Lat, in.Lng = *req.Lat, *req.Lng
	}
	actor := userIDFrom(r)
	s.WithIdempotentTx(w, r, func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error) {
		id, err := s.orders.CreateMerchantDeliveryIn(ctx, q, merchantID, actor, in)
		if err != nil {
			return IdempotentBody{}, err
		}
		return IdempotentBody{
			Status:  http.StatusCreated,
			Payload: map[string]any{"id": id},
			// **والعرضُ على السائقين بعد التثبيت لا داخلَه** — عرضٌ خرج ثمّ
			// ارتدّت المعاملةُ يوقظ سائقاً لتوصيلةٍ لا وجودَ لها.
			AfterCommit: func() {
				if _, err := s.orders.AfterMerchantDelivery(context.WithoutCancel(r.Context()), id); err != nil {
					s.logger.Error("التوصيلة: بعد الإنشاء", "order", id, "error", err)
				}
				s.touch("orders", "ops")
			},
		}, nil
	})
}

// handleMerchantDeliveries **توصيلاتُ المتجر** — جاريةً ومنتهية، الأحدثُ أوّلاً.
func (s *Server) handleMerchantDeliveries(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	// **وسجلُّ التوصيلات كلِّها صفحةً صفحة** — (نصُّ المالك: «سجلّ التوصيلات
	// بحيث نستطيع الوصولَ إلى كلّ عمليّات التوصيل التي قمنا بها».)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	list, err := s.orders.ListMerchantDeliveries(r.Context(), merchantID, 50, page)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **ومن بوّابة الحقول لا الطلبَ خاماً** — وكان يُرسَل كاملاً فيه هاتفُ
	// السائق ومعرّفاتٌ لا تخصّ المتجر. (أمسكه فحصُ الخطوة ١٨ قبل النشر.)
	views, err := orderViews(orders.AudienceMerchantDelivery, list)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deliveries": views})
}

// handleMerchantCancelDelivery **يُلغي المتجرُ توصيلتَه ما دام الغرضُ عنده.**
//
// **والآلةُ تحكم** (`merchantDeliveryEdges`): في الطابور أو أُسندت أو وصل
// السائقُ إليه — **وبعد الاستلام يُردّ بلفظ الآلة.** **والمالُ يعود لمن دفعه**
// (`settleMerchantDelivery`).
func (s *Server) handleMerchantCancelDelivery(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	var merchantID, kind string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT COALESCE(merchant_id::text, ''), kind FROM orders WHERE id = $1`, orderID).
		Scan(&merchantID, &kind); err != nil || kind != orders.KindMerchantDelivery {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	o, err := s.orders.Transition(r.Context(), userIDFrom(r), []string{"merchant"},
		orderID, orders.StCancelled, "ألغاها المتجر")
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
	httpx.JSON(w, http.StatusOK, view)
}

// handleMerchantDelivery **توصيلةٌ واحدة** — لشاشة مراقبتها (الخطوة ١٨).
//
// (نصُّ المالك: «شاشة مراقبة الطلب بسجلّ الطلبات ليعرف المتجرُ حالةَ
// توصيلته».) **وتُقرأ كلَّ ثوانٍ** ما دامت الشاشةُ مفتوحة.
func (s *Server) handleMerchantDelivery(w http.ResponseWriter, r *http.Request) {
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
	o, err := s.orders.GetByID(r.Context(), orderID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	view, err := orderView(orders.AudienceMerchantDelivery, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, view)
}

// ══════════════════════════════════════════════════════════════════════
// **سقفُ دينِ التوصيلة — للإدارة وحدَها** (العقد §٣: «سقفٌ إداريٌّ افتراضُه صفر»)
// ══════════════════════════════════════════════════════════════════════
//
// **كان العقدُ يقول «الإدارةُ وحدَها ترفعه» ولا بابَ يرفعه** — فكلُّ متجرٍ
// سقفُه صفرٌ أبداً، **ومن لا رصيدَ له لا يطلب توصيلةً يدفعها هو.**
//
// **وصلاحيّةٌ ماليّةٌ لا إدارةُ متجر** (`SettingsFinancialManage`) — كطريقة
// التسوية جارتِه، **وبخطوة تحقّقٍ** (`sensitive.go`): رقمٌ يُدين به المتجرُ المنصّة.

// handleAdminMerchantDeliveryCredit **السقفُ والقائم** — يُقرآن معاً.
func (s *Server) handleAdminMerchantDeliveryCredit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	limit, owed, err := s.orders.MerchantDeliveryCredit(r.Context(), id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"limit": limit, "owed": owed})
}

// handleAdminSetMerchantDeliveryCredit **يضبط السقف** — ولا يمسّ القائم.
//
// **وخفضُه دون القائم لا يُسقط ديناً** — يمنع الجديدَ وحدَه حتّى يُسدَّد.
func (s *Server) handleAdminSetMerchantDeliveryCredit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Limit *int64 `json:"limit"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Limit == nil || *req.Limit < 0 {
		s.respondErr(w, errValidation)
		return
	}
	var before int64
	if err := s.pg.QueryRow(r.Context(), `
		UPDATE merchants m SET delivery_credit_limit = $2
		  FROM (SELECT delivery_credit_limit AS old FROM merchants WHERE id = $1) o
		 WHERE m.id = $1 RETURNING o.old`, id, *req.Limit).Scan(&before); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	s.audit(r, "admin.merchant_delivery_credit", "merchant", id,
		map[string]any{"from": before, "to": *req.Limit})
	httpx.JSON(w, http.StatusOK, map[string]any{"limit": *req.Limit})
}
