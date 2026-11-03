package server

// **نبضُ المتجر** — قراءةٌ فقط لرأس ملفّ المتجر (٣٦٠): أمفتوحٌ الآن؟ وكم طلباً اليوم؟ وكم صنفاً غير متوفر؟
//
//   - **«مفتوح الآن» بـ`OpenNowSQL` نفسِه** الذي يحكم إنشاءَ الطلب — فلا يفترق العرضُ عن القبول.
//   - **اليومُ يومُ المنصّة** (`platform.day_start_hour`) كما في الرئيسيّة.
//   - **الرفضُ أرقامٌ لا نسبة**: عدد المرفوض من المُرسَل خلال ٣٠ يوماً — والنسبةُ يحسبها من يقرأ.
//   - لا كتابةَ ولا مالَ هنا.

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

type merchantPulse struct {
	OpenNow          bool       `json:"open_now"`
	NextOpen         *time.Time `json:"next_open"`
	PrepMinutes      int        `json:"default_prep_minutes"`
	OrdersToday      int64      `json:"orders_today"`
	DeliveredToday   int64      `json:"delivered_today"`
	OrdersOpen       int64      `json:"orders_open"`
	Sent30d          int64      `json:"sent_30d"`
	Rejected30d      int64      `json:"rejected_30d"`
	ItemsTotal       int64      `json:"items_total"`
	ItemsUnavailable int64      `json:"items_unavailable"`
	DayStart         time.Time  `json:"day_start"`
}

// handleAdminMerchantPulse GET /admin/merchants/{id}/pulse
func (s *Server) handleAdminMerchantPulse(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()
	today, _, _ := overviewDays(time.Now(), s.settings.GetInt(ctx, "platform.day_start_hour"))
	var p merchantPulse
	p.DayStart = today
	err := s.pg.QueryRow(ctx, `
		SELECT `+orders.OpenNowSQL+`, `+orders.NextOpenSQL+`, m.default_prep_minutes,
		  (SELECT count(*) FROM orders o WHERE o.merchant_id = m.id AND o.created_at >= $2),
		  (SELECT count(*) FROM orders o WHERE o.merchant_id = m.id AND o.status = 'delivered' AND o.delivered_at >= $2),
		  (SELECT count(*) FROM orders o WHERE o.merchant_id = m.id AND o.closed_at IS NULL),
		  (SELECT count(*) FROM orders o WHERE o.merchant_id = m.id AND o.sent_to_merchant_at >= now() - interval '30 days'),
		  (SELECT count(*) FROM orders o WHERE o.merchant_id = m.id AND o.status = 'rejected'
		      AND o.sent_to_merchant_at >= now() - interval '30 days'),
		  (SELECT count(*) FROM menu_items i WHERE i.merchant_id = m.id),
		  (SELECT count(*) FROM menu_items i WHERE i.merchant_id = m.id AND NOT i.available)
		FROM merchants m WHERE m.id = $1`, id, today).Scan(
		&p.OpenNow, &p.NextOpen, &p.PrepMinutes, &p.OrdersToday, &p.DeliveredToday, &p.OrdersOpen,
		&p.Sent30d, &p.Rejected30d, &p.ItemsTotal, &p.ItemsUnavailable)
	if err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}
