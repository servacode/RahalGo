package server

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	res, err := s.identity.AdminListUsers(r.Context(), q.Get("query"), q.Get("role"), q.Get("online") == "true", q.Get("status"), page, perPage)
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
	// صاحب الحساب يعرف بتغيّر حالته فوراً بدل أن يكتشفه عند أول رفض
	if req.Status != nil {
		title := notifTitles.accountActivated
		if *req.Status != "active" {
			title = notifTitles.accountSuspended
		}
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: user.ID, Kind: notifications.KindAccount,
			Title: title, Body: derefOr(req.StatusReason, ""),
			Entity: "user", EntityID: user.ID,
		})
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (s *Server) handleAdminGrantRole(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Role   string `json:"role"`
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.AdminGrantRole(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), req.Role, req.Reason, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"granted": true})
}

func (s *Server) handleAdminRevokeRole(w http.ResponseWriter, r *http.Request) {
	if err := s.identity.AdminRevokeRole(r.Context(), userIDFrom(r),
		chi.URLParam(r, "id"), chi.URLParam(r, "role"), r.URL.Query().Get("reason"), clientIP(r)); err != nil {
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
	var total, staff, online int
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*),
		       count(*) FILTER (WHERE EXISTS (SELECT 1 FROM user_roles sr
		           WHERE sr.user_id = u.id AND sr.role_code IN ('ops','finance'))),
		       count(*) FILTER (WHERE u.last_seen_at > now() - interval '2 minutes')
		FROM users u
		WHERE NOT EXISTS (SELECT 1 FROM user_roles r WHERE r.user_id = u.id AND r.role_code = 'admin')`).
		Scan(&total, &staff, &online); err != nil {
		s.respondErr(w, err)
		return
	}
	counts["staff"] = staff
	counts["online"] = online
	httpx.JSON(w, http.StatusOK, map[string]any{"total": total, "roles": counts})
}

// handleAdminGetUser صفحة تفاصيل الحساب: الملف + المحفظة + مؤشرات حسب أدواره.
func (s *Server) handleAdminGetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	var out struct {
		ID           string   `json:"id"`
		Phone        string   `json:"phone"`
		FullName     string   `json:"full_name"`
		Status       string   `json:"status"`
		InviteCode   *string  `json:"invite_code"`
		AvatarThumb  *string  `json:"avatar_thumb_url"`
		StatusReason string   `json:"status_reason"`
		AdminNotes   string   `json:"admin_notes"`
		Sessions     int      `json:"active_sessions"`
		Roles        []string `json:"roles"`
		CreatedAt    string   `json:"created_at"`
		Balance      int64    `json:"balance"`
		OrdersCount  int      `json:"orders_count"` // كزبون
		OrdersSpent  int64    `json:"orders_spent"` // إنفاقه المُسلَّم
		Merchants    []string `json:"merchants"`    // متاجر يملكها
		RepStores    int      `json:"rep_stores"`   // متاجر جلبها كمندوب
		Commissions  int64    `json:"commissions"`  // عمولاته كمندوب
		DriverCash   int64    `json:"driver_cash"`  // نقد بحوزته كسائق
		Deliveries   int      `json:"deliveries"`   // توصيلاته المُسلَّمة
	}
	err := s.pg.QueryRow(r.Context(), `
		SELECT u.id, u.phone, u.full_name, u.status, u.invite_code, u.status_reason, u.admin_notes,
		       (SELECT count(*) FROM refresh_tokens rt WHERE rt.user_id = u.id AND rt.revoked_at IS NULL AND rt.expires_at > now()),
		       (SELECT m.thumb_path FROM media m WHERE m.id = u.avatar_media_id), u.created_at::text,
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
		Scan(&out.ID, &out.Phone, &out.FullName, &out.Status, &out.InviteCode, &out.StatusReason, &out.AdminNotes, &out.Sessions, &out.AvatarThumb, &out.CreatedAt,
			&out.Roles, &out.Balance, &out.OrdersCount, &out.OrdersSpent, &out.Merchants,
			&out.RepStores, &out.Commissions, &out.DriverCash, &out.Deliveries)
	if err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	out.AvatarThumb = media.URLForPtr(out.AvatarThumb)
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

// handleAdminUserActivity سجل نشاط الحساب: ما فعله وما فُعل به (من سجل التدقيق).
func (s *Server) handleAdminUserActivity(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := s.pg.Query(r.Context(), `
		SELECT a.action, a.entity, COALESCE(a.entity_id, ''), COALESCE(a.ip, ''),
		       COALESCE(a.details::text, ''), a.created_at,
		       NULLIF(COALESCE(au.full_name, au.phone::text), ''),
		       (a.actor_user_id IS NOT DISTINCT FROM u.id) AS by_self
		FROM audit_log a
		CROSS JOIN (SELECT id FROM users WHERE id = $1) u
		LEFT JOIN users au ON au.id = a.actor_user_id
		WHERE a.actor_user_id = u.id OR (a.entity = 'user' AND a.entity_id = u.id::text)
		ORDER BY a.id DESC LIMIT 100`, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type entry struct {
		Action    string    `json:"action"`
		Entity    string    `json:"entity"`
		EntityID  string    `json:"entity_id"`
		IP        string    `json:"ip"`
		Details   string    `json:"details"`
		ByName    *string   `json:"by_name"`
		BySelf    bool      `json:"by_self"`
		CreatedAt time.Time `json:"created_at"`
	}
	out := []entry{}
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.Action, &e.Entity, &e.EntityID, &e.IP, &e.Details,
			&e.CreatedAt, &e.ByName, &e.BySelf); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, e)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleAdminLogoutAll إنهاء كل جلسات الحساب فوراً.
func (s *Server) handleAdminLogoutAll(w http.ResponseWriter, r *http.Request) {
	n, err := s.identity.AdminLogoutAll(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"revoked_sessions": n})
}

// handleAdminUserFeedback الشكاوى والتقييمات المرتبطة بالحساب:
// تذاكره كزبون، تقييماته الممنوحة، والواردة عليه (كسائق أو صاحب متاجر).
func (s *Server) handleAdminUserFeedback(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	out := struct {
		Tickets []map[string]any `json:"tickets"`
		Given   []map[string]any `json:"ratings_given"`
		Recv    []map[string]any `json:"ratings_received"`
		AvgRecv *float64         `json:"avg_received"`
	}{Tickets: []map[string]any{}, Given: []map[string]any{}, Recv: []map[string]any{}}

	rows, err := s.pg.Query(r.Context(), `
		SELECT t.number, t.subject, t.status, t.compensation, t.created_at
		FROM tickets t WHERE t.customer_id = $1 ORDER BY t.number DESC LIMIT 20`, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for rows.Next() {
		var num, comp int64
		var subject, status string
		var at time.Time
		if err := rows.Scan(&num, &subject, &status, &comp, &at); err == nil {
			out.Tickets = append(out.Tickets, map[string]any{
				"number": num, "subject": subject, "status": status,
				"compensation": comp, "created_at": at})
		}
	}
	rows.Close()

	rows, err = s.pg.Query(r.Context(), `
		SELECT o.number, m.name, rt.merchant_stars, rt.driver_stars, rt.comment, rt.created_at
		FROM order_ratings rt
		JOIN orders o ON o.id = rt.order_id
		JOIN merchants m ON m.id = o.merchant_id
		WHERE rt.customer_id = $1 ORDER BY rt.created_at DESC LIMIT 20`, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for rows.Next() {
		var num int64
		var mName, comment string
		var ms int
		var ds *int
		var at time.Time
		if err := rows.Scan(&num, &mName, &ms, &ds, &comment, &at); err == nil {
			out.Given = append(out.Given, map[string]any{
				"order_number": num, "merchant_name": mName,
				"merchant_stars": ms, "driver_stars": ds, "comment": comment, "created_at": at})
		}
	}
	rows.Close()

	// الواردة: كسائق (نجوم السائق على طلباته) + كصاحب متاجر (نجوم متاجره)
	rows, err = s.pg.Query(r.Context(), `
		SELECT o.number, m.name, rt.driver_stars, rt.comment, rt.created_at, 'driver'
		FROM order_ratings rt
		JOIN orders o ON o.id = rt.order_id
		JOIN merchants m ON m.id = o.merchant_id
		WHERE o.driver_id = $1 AND rt.driver_stars IS NOT NULL
		UNION ALL
		SELECT o.number, m.name, rt.merchant_stars, rt.comment, rt.created_at, 'merchant'
		FROM order_ratings rt
		JOIN orders o ON o.id = rt.order_id
		JOIN merchants m ON m.id = o.merchant_id
		WHERE m.owner_user_id = $1
		ORDER BY 5 DESC LIMIT 20`, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	sum, n := 0, 0
	for rows.Next() {
		var num int64
		var mName, comment, as string
		var stars int
		var at time.Time
		if err := rows.Scan(&num, &mName, &stars, &comment, &at, &as); err == nil {
			out.Recv = append(out.Recv, map[string]any{
				"order_number": num, "merchant_name": mName, "stars": stars,
				"comment": comment, "created_at": at, "as": as})
			sum += stars
			n++
		}
	}
	rows.Close()
	if n > 0 {
		avg := float64(sum) / float64(n)
		out.AvgRecv = &avg
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleAdminUsersExport تصدير الحسابات المفلترة إلى CSV (بلا ترقيم — كلها).
func (s *Server) handleAdminUsersExport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	res, err := s.identity.AdminListUsers(r.Context(), q.Get("query"), q.Get("role"),
		q.Get("online") == "true", q.Get("status"), 1, 10000)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="accounts.csv"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF}) // BOM لعرض العربية في Excel
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"الاسم", "الهاتف", "الأدوار", "الحالة", "كود الدعوة", "آخر ظهور", "تاريخ التسجيل"})
	for _, u := range res.Users {
		lastSeen := ""
		if u.LastSeenAt != nil {
			lastSeen = u.LastSeenAt.Format("2006-01-02 15:04")
		}
		invite := ""
		if u.InviteCode != nil {
			invite = *u.InviteCode
		}
		_ = cw.Write([]string{
			u.FullName, u.Phone, strings.Join(u.Roles, "+"), u.Status, invite,
			lastSeen, u.CreatedAt.Format("2006-01-02"),
		})
	}
	cw.Flush()
}

// derefOr يقرأ مؤشراً نصياً بأمان.
func derefOr(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
}
