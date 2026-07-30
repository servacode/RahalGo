package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// قسم الزبائن: الحساب + إحصاءات طلباته ومحفظته.
func (s *Server) handleListCustomers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	search := q.Get("query")

	where := `WHERE EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = u.id AND ur.role_code = 'customer')
	          AND ($1 = '' OR u.phone ILIKE '%'||$1||'%' OR u.full_name ILIKE '%'||$1||'%')`

	var total int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*) FROM users u `+where, search).Scan(&total); err != nil {
		s.respondErr(w, err)
		return
	}

	rows, err := s.pg.Query(r.Context(), `
		SELECT u.id, u.phone, u.full_name, u.status, u.created_at,
		       COALESCE(wl.balance, 0),
		       (SELECT count(*) FROM orders o WHERE o.customer_id = u.id),
		       (SELECT count(*) FROM orders o WHERE o.customer_id = u.id AND o.status = 'delivered'),
		       COALESCE((SELECT sum(o.total) FROM orders o WHERE o.customer_id = u.id AND o.status = 'delivered'), 0),
		       (SELECT max(o.created_at) FROM orders o WHERE o.customer_id = u.id)
		FROM users u
		LEFT JOIN wallets wl ON wl.user_id = u.id
		`+where+`
		ORDER BY u.created_at DESC LIMIT $2 OFFSET $3`, search, perPage, (page-1)*perPage)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type customer struct {
		ID          string     `json:"id"`
		Phone       string     `json:"phone"`
		FullName    string     `json:"full_name"`
		Status      string     `json:"status"`
		CreatedAt   time.Time  `json:"created_at"`
		Balance     int64      `json:"balance"`
		OrdersCount int        `json:"orders_count"`
		Delivered   int        `json:"delivered_count"`
		TotalSpent  int64      `json:"total_spent"`
		LastOrderAt *time.Time `json:"last_order_at"`
	}
	out := []customer{}
	for rows.Next() {
		var c customer
		if err := rows.Scan(&c.ID, &c.Phone, &c.FullName, &c.Status, &c.CreatedAt,
			&c.Balance, &c.OrdersCount, &c.Delivered, &c.TotalSpent, &c.LastOrderAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, c)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"customers": out, "total": total, "page": page, "per_page": perPage,
	})
}

// قسم المندوبين: الكود، متاجره، عمولاته المتراكمة، رصيده.
func (s *Server) handleListSalesReps(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT u.id, u.phone, u.full_name, u.status, u.invite_code,
		       COALESCE(wl.balance, 0),
		       (SELECT count(*) FROM merchants mr WHERE mr.sales_rep_user_id = u.id),
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t
		                 WHERE t.user_id = u.id AND t.kind = 'commission'), 0)
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'sales'
		LEFT JOIN wallets wl ON wl.user_id = u.id
		ORDER BY u.created_at`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type rep struct {
		ID               string  `json:"id"`
		Phone            string  `json:"phone"`
		FullName         string  `json:"full_name"`
		Status           string  `json:"status"`
		InviteCode       *string `json:"invite_code"`
		Balance          int64   `json:"balance"`
		MerchantsCount   int     `json:"merchants_count"`
		TotalCommissions int64   `json:"total_commissions"`
	}
	out := []rep{}
	for rows.Next() {
		var p rep
		if err := rows.Scan(&p.ID, &p.Phone, &p.FullName, &p.Status, &p.InviteCode,
			&p.Balance, &p.MerchantsCount, &p.TotalCommissions); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, p)
	}
	httpx.JSON(w, http.StatusOK, out)
}
