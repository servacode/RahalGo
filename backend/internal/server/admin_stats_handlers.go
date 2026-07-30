package server

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// إحصاءات لوحة القيادة — أرقام حية من قاعدة البيانات.
func (s *Server) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	var st struct {
		Customers       int `json:"customers"`
		Drivers         int `json:"drivers"`
		SalesReps       int `json:"sales_reps"`
		MerchantsActive int `json:"merchants_active"`
		MerchantsTotal  int `json:"merchants_total"`
		MenuItems       int `json:"menu_items"`
		ZonesActive     int `json:"zones_active"`
		PromosActive    int `json:"promos_active"`
	}
	err := s.pg.QueryRow(r.Context(), `
		SELECT
			(SELECT count(*) FROM user_roles WHERE role_code = 'customer'),
			(SELECT count(*) FROM user_roles WHERE role_code = 'driver'),
			(SELECT count(*) FROM user_roles WHERE role_code = 'sales'),
			(SELECT count(*) FROM merchants WHERE status = 'active'),
			(SELECT count(*) FROM merchants),
			(SELECT count(*) FROM menu_items),
			(SELECT count(*) FROM delivery_zones WHERE active),
			(SELECT count(*) FROM promo_codes WHERE active)`).
		Scan(&st.Customers, &st.Drivers, &st.SalesReps, &st.MerchantsActive,
			&st.MerchantsTotal, &st.MenuItems, &st.ZonesActive, &st.PromosActive)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}
