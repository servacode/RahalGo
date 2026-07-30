package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
)

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	res, err := s.identity.AdminListUsers(r.Context(), q.Get("query"), q.Get("role"), page, perPage)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	req, err := decode[identity.CreateUserInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	user, err := s.identity.AdminCreateUser(r.Context(), userIDFrom(r), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, user)
}

func (s *Server) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	req, err := decode[identity.UpdateUserInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	user, err := s.identity.AdminUpdateUser(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (s *Server) handleAdminGrantRole(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Role string `json:"role"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.AdminGrantRole(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), req.Role, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"granted": true})
}

func (s *Server) handleAdminRevokeRole(w http.ResponseWriter, r *http.Request) {
	if err := s.identity.AdminRevokeRole(r.Context(), userIDFrom(r),
		chi.URLParam(r, "id"), chi.URLParam(r, "role"), clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"revoked": true})
}

// handleAdminUserRoleCounts أعداد الحسابات لكل دور — تغذي الكروت الذكية
// الفلترة أعلى شاشة الحسابات.
func (s *Server) handleAdminUserRoleCounts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(),
		`SELECT role_code, count(*) FROM user_roles WHERE role_code <> 'admin' GROUP BY role_code`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var role string
		var n int
		if err := rows.Scan(&role, &n); err != nil {
			s.respondErr(w, err)
			return
		}
		counts[role] = n
	}
	var total int
	if err := s.pg.QueryRow(r.Context(), `SELECT count(*) FROM users u WHERE NOT EXISTS (SELECT 1 FROM user_roles r WHERE r.user_id = u.id AND r.role_code = 'admin')`).Scan(&total); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"total": total, "roles": counts})
}

// handleAdminGetUser صفحة تفاصيل الحساب: الملف + المحفظة + مؤشرات حسب أدواره.
func (s *Server) handleAdminGetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var out struct {
		ID          string   `json:"id"`
		Phone       string   `json:"phone"`
		FullName    string   `json:"full_name"`
		Status      string   `json:"status"`
		InviteCode  *string  `json:"invite_code"`
		Roles       []string `json:"roles"`
		CreatedAt   string   `json:"created_at"`
		Balance     int64    `json:"balance"`
		OrdersCount int      `json:"orders_count"` // كزبون
		OrdersSpent int64    `json:"orders_spent"` // إنفاقه المُسلَّم
		Merchants   []string `json:"merchants"`    // متاجر يملكها
		RepStores   int      `json:"rep_stores"`   // متاجر جلبها كمندوب
		Commissions int64    `json:"commissions"`  // عمولاته كمندوب
		DriverCash  int64    `json:"driver_cash"`  // نقد بحوزته كسائق
		Deliveries  int      `json:"deliveries"`   // توصيلاته المُسلَّمة
	}
	err := s.pg.QueryRow(r.Context(), `
		SELECT u.id, u.phone, u.full_name, u.status, u.invite_code, u.created_at::text,
		       COALESCE((SELECT array_agg(role_code ORDER BY role_code) FROM user_roles WHERE user_id = u.id), '{}'),
		       COALESCE((SELECT balance FROM wallets WHERE user_id = u.id), 0),
		       (SELECT count(*) FROM orders o WHERE o.customer_id = u.id),
		       COALESCE((SELECT sum(o.total) FROM orders o WHERE o.customer_id = u.id AND o.status = 'delivered'), 0),
		       COALESCE((SELECT array_agg(m.name ORDER BY m.created_at) FROM merchants m WHERE m.owner_user_id = u.id), '{}'),
		       (SELECT count(*) FROM merchants m WHERE m.sales_rep_user_id = u.id),
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t WHERE t.user_id = u.id AND t.kind = 'commission'), 0),
		       COALESCE((SELECT held FROM driver_cash_boxes WHERE driver_id = u.id), 0),
		       (SELECT count(*) FROM orders o WHERE o.driver_id = u.id AND o.status = 'delivered')
		FROM users u WHERE u.id = $1`, id).
		Scan(&out.ID, &out.Phone, &out.FullName, &out.Status, &out.InviteCode, &out.CreatedAt,
			&out.Roles, &out.Balance, &out.OrdersCount, &out.OrdersSpent, &out.Merchants,
			&out.RepStores, &out.Commissions, &out.DriverCash, &out.Deliveries)
	if err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleAdminResetPassword تعيين كلمة مرور جديدة لحساب — أدمن حصراً، وتُسجل تدقيقاً.
func (s *Server) handleAdminResetPassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	req, err := decode[struct {
		Password string `json:"password"`
	}](r)
	if err != nil || len(req.Password) < 8 {
		s.respondErr(w, httpx.NewError(http.StatusBadRequest, "weak_password", "errors.weak_password"))
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	tag, err := s.pg.Exec(r.Context(),
		`UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, id, hash)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	actor := userIDFrom(r)
	_, _ = s.pg.Exec(r.Context(), `
		INSERT INTO audit_log (actor_user_id, action, entity, entity_id, ip)
		VALUES ($1, 'admin.password_reset', 'user', $2, $3)`, actor, id, clientIP(r))
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}
