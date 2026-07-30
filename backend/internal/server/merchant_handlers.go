package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// بوابة المتجر — كل نقطة تتحقق أن الكيان مملوك لصاحب الحساب الفاعل:
// المتجر عبر owner_user_id، والطلب/الصنف عبر متجرهما.

// ownsMerchant يتحقق أن المتجر مملوك للمستخدم.
func (s *Server) ownsMerchant(r *http.Request, merchantID string) bool {
	var owns bool
	err := s.pg.QueryRow(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM merchants WHERE id = $1 AND owner_user_id = $2)`,
		merchantID, userIDFrom(r)).Scan(&owns)
	return err == nil && owns
}

type merchantStore struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	CategoryIcon    string  `json:"category_icon"`
	LogoThumbURL    *string `json:"logo_thumb_url"`
	Status          string  `json:"status"`
	EmergencyClosed bool    `json:"emergency_closed"`
}

// handleMerchantStores متاجر صاحب الحساب.
func (s *Server) handleMerchantStores(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT m.id, m.name, c.icon, lm.thumb_path, m.status, m.emergency_closed
		FROM merchants m
		JOIN categories c ON c.id = m.category_id
		LEFT JOIN media lm ON lm.id = m.logo_media_id
		WHERE m.owner_user_id = $1 ORDER BY m.created_at`, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []merchantStore{}
	for rows.Next() {
		var m merchantStore
		if err := rows.Scan(&m.ID, &m.Name, &m.CategoryIcon, &m.LogoThumbURL,
			&m.Status, &m.EmergencyClosed); err != nil {
			s.respondErr(w, err)
			return
		}
		m.LogoThumbURL = media.URLForPtr(m.LogoThumbURL)
		out = append(out, m)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleMerchantOrders طلبات متجر مملوك.
func (s *Server) handleMerchantOrders(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	res, err := s.orders.List(r.Context(), orders.ListFilter{
		MerchantID: merchantID,
		Status:     q.Get("status"),
		OpenOnly:   q.Get("open_only") == "true",
		Page:       page,
		PerPage:    perPage,
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

// merchantOwnsOrder يتحقق أن الطلب يتبع متجراً مملوكاً للمستخدم.
func (s *Server) merchantOwnsOrder(r *http.Request, orderID string) bool {
	var owns bool
	err := s.pg.QueryRow(r.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM orders o JOIN merchants m ON m.id = o.merchant_id
			WHERE o.id = $1 AND m.owner_user_id = $2)`,
		orderID, userIDFrom(r)).Scan(&owns)
	return err == nil && owns
}

func (s *Server) handleMerchantGetOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !s.merchantOwnsOrder(r, id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	o, err := s.orders.GetByID(r.Context(), id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}

// handleMerchantTransition قبول/رفض/بدء تحضير — آلة الحالات تضبط المسموح لدور المتجر.
func (s *Server) handleMerchantTransition(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !s.merchantOwnsOrder(r, id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		To   string `json:"to"`
		Note string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	o, err := s.orders.Transition(r.Context(), userIDFrom(r), []string{"merchant"}, id, req.To, req.Note)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}

func (s *Server) handleMerchantMenu(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	menu, err := s.catalog.GetMenu(r.Context(), merchantID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, menu)
}

// handleMerchantItemAvailability تشغيل التوفر اليومي — صلاحية المتجر الوحيدة على القائمة
// (إدارة القائمة السيادية للمنصة — قرار 15).
func (s *Server) handleMerchantItemAvailability(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")
	req, err := decode[struct {
		Available *bool `json:"available"`
	}](r)
	if err != nil || req.Available == nil {
		s.respondErr(w, errValidation)
		return
	}
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE menu_items i SET available = $3, updated_at = now()
		FROM merchants m
		WHERE i.id = $1 AND m.id = i.merchant_id AND m.owner_user_id = $2`,
		itemID, userIDFrom(r), *req.Available)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"available": *req.Available})
}

// handleMerchantEmergency إغلاق/فتح طارئ للمتجر من صاحبه.
func (s *Server) handleMerchantEmergency(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	req, err := decode[struct {
		Closed *bool `json:"closed"`
	}](r)
	if err != nil || req.Closed == nil {
		s.respondErr(w, errValidation)
		return
	}
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE merchants SET emergency_closed = $3, updated_at = now()
		WHERE id = $1 AND owner_user_id = $2`,
		merchantID, userIDFrom(r), *req.Closed)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, errForbidden)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"emergency_closed": *req.Closed})
}

// handleMerchantReports ملخص تشغيلي ومالي لمتجر مملوك بمدى زمني.
func (s *Server) handleMerchantReports(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	to := time.Now()
	from := to.AddDate(0, 0, -6)
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			from = t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			to = t
		}
	}
	toEnd := to.AddDate(0, 0, 1)

	var summary struct {
		Orders     int   `json:"orders"`
		Delivered  int   `json:"delivered"`
		Cancelled  int   `json:"cancelled"`
		Sales      int64 `json:"sales"`
		Commission int64 `json:"platform_commission"`
	}
	err := s.pg.QueryRow(r.Context(), `
		SELECT count(*),
		       count(*) FILTER (WHERE status = 'delivered'),
		       count(*) FILTER (WHERE status IN ('cancelled','rejected','failed')),
		       COALESCE(sum(subtotal) FILTER (WHERE status = 'delivered'), 0),
		       COALESCE(sum(platform_commission) FILTER (WHERE status = 'delivered'), 0)
		FROM orders
		WHERE merchant_id = $1 AND created_at >= $2 AND created_at < $3`,
		merchantID, from, toEnd).
		Scan(&summary.Orders, &summary.Delivered, &summary.Cancelled,
			&summary.Sales, &summary.Commission)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	type day struct {
		Date      string `json:"date"`
		Orders    int    `json:"orders"`
		Delivered int    `json:"delivered"`
		Sales     int64  `json:"sales"`
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT d::date::text,
		       COALESCE(o.orders, 0), COALESCE(o.delivered, 0), COALESCE(o.sales, 0)
		FROM generate_series($2::date, $3::date, '1 day') d
		LEFT JOIN (
			SELECT created_at::date AS day, count(*) AS orders,
			       count(*) FILTER (WHERE status = 'delivered') AS delivered,
			       COALESCE(sum(subtotal) FILTER (WHERE status = 'delivered'), 0) AS sales
			FROM orders WHERE merchant_id = $1 AND created_at >= $2 AND created_at < $4
			GROUP BY 1
		) o ON o.day = d::date`,
		merchantID, from.Format("2006-01-02"), to.Format("2006-01-02"), toEnd)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	days := []day{}
	for rows.Next() {
		var d day
		if err := rows.Scan(&d.Date, &d.Orders, &d.Delivered, &d.Sales); err != nil {
			s.respondErr(w, err)
			return
		}
		days = append(days, d)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"summary": summary, "days": days})
}
