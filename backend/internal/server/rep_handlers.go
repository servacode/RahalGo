package server

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// لوحة المندوب: كوده للدعوة، متاجره، عمولاته — قراءة فقط (الترويج يتم ميدانياً).

// handleRepMe ملخص المندوب: كود الدعوة والإحصاءات المالية.
func (s *Server) handleRepMe(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	var out struct {
		InviteCode       *string `json:"invite_code"`
		FullName         string  `json:"full_name"`
		Merchants        int     `json:"merchants"`
		DeliveredOrders  int     `json:"delivered_orders"`
		TotalCommissions int64   `json:"total_commissions"`
		Balance          int64   `json:"balance"`
		// الشهر الجاري — الهدف يُقاس عليه لا على المجموع التراكمي
		MonthMerchants   int   `json:"month_merchants"`
		MonthDelivered   int   `json:"month_delivered"`
		MonthCommissions int64 `json:"month_commissions"`
		MonthlyTarget    int   `json:"monthly_target"`
		PendingLeads     int   `json:"pending_leads"`
		// أدوات الدعوة مقفلة حتى يوثّق المندوب قناة تواصله
		WhatsAppVerified bool `json:"whatsapp_verified"`
	}
	err := s.pg.QueryRow(r.Context(), `
		SELECT u.invite_code, u.full_name,
		       (SELECT count(*) FROM merchants m WHERE m.sales_rep_user_id = u.id),
		       (SELECT count(*) FROM orders o JOIN merchants m ON m.id = o.merchant_id
		        WHERE m.sales_rep_user_id = u.id AND o.status = 'delivered'),
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t
		                 WHERE t.user_id = u.id AND t.kind = 'commission'), 0),
		       COALESCE((SELECT w.balance FROM wallets w WHERE w.user_id = u.id), 0),
		       -- الشهر الجاري بتوقيت سوريا (لا UTC: نهاية الشهر تهمّ المندوب)
		       (SELECT count(*) FROM merchants m
		        WHERE m.sales_rep_user_id = u.id
		          AND date_trunc('month', m.created_at AT TIME ZONE 'Asia/Damascus')
		            = date_trunc('month', now() AT TIME ZONE 'Asia/Damascus')),
		       (SELECT count(*) FROM orders o JOIN merchants m ON m.id = o.merchant_id
		        WHERE m.sales_rep_user_id = u.id AND o.status = 'delivered'
		          AND date_trunc('month', o.delivered_at AT TIME ZONE 'Asia/Damascus')
		            = date_trunc('month', now() AT TIME ZONE 'Asia/Damascus')),
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t
		                 WHERE t.user_id = u.id AND t.kind = 'commission'
		                   AND date_trunc('month', t.created_at AT TIME ZONE 'Asia/Damascus')
		                     = date_trunc('month', now() AT TIME ZONE 'Asia/Damascus')), 0),
		       COALESCE((SELECT (value#>>'{}')::int FROM app_settings
		                 WHERE key = 'sales.monthly_target'), 5),
		       (SELECT count(*) FROM merchant_leads l
		        WHERE l.sales_rep_user_id = u.id AND l.status = 'new'),
		       u.whatsapp_verified_at IS NOT NULL
		FROM users u WHERE u.id = $1`, uid).
		Scan(&out.InviteCode, &out.FullName, &out.Merchants, &out.DeliveredOrders,
			&out.TotalCommissions, &out.Balance,
			&out.MonthMerchants, &out.MonthDelivered, &out.MonthCommissions,
			&out.MonthlyTarget, &out.PendingLeads, &out.WhatsAppVerified)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// القفل في الخادم لا في الواجهة: إخفاء الكود من الشاشة وحدها ليس قفلاً —
	// من يفتح أدوات المتصفح يقرأه من الاستجابة. فلا يُرسل أصلاً قبل التوثيق.
	if !out.WhatsAppVerified {
		out.InviteCode = nil
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleRepMerchants متاجر المندوب مع نشاط كل متجر.
func (s *Server) handleRepMerchants(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT m.id, m.name, c.icon, lm.thumb_path, m.status, m.created_at::date::text,
		       (SELECT count(*) FROM orders o WHERE o.merchant_id = m.id AND o.status = 'delivered')
		FROM merchants m
		JOIN categories c ON c.id = m.category_id
		LEFT JOIN media lm ON lm.id = m.logo_media_id
		WHERE m.sales_rep_user_id = $1
		ORDER BY m.created_at DESC`, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type repMerchant struct {
		ID           string  `json:"id"`
		Name         string  `json:"name"`
		CategoryIcon string  `json:"category_icon"`
		LogoThumbURL *string `json:"logo_thumb_url"`
		Status       string  `json:"status"`
		JoinedAt     string  `json:"joined_at"`
		Delivered    int     `json:"delivered_orders"`
	}
	out := []repMerchant{}
	for rows.Next() {
		var m repMerchant
		if err := rows.Scan(&m.ID, &m.Name, &m.CategoryIcon, &m.LogoThumbURL,
			&m.Status, &m.JoinedAt, &m.Delivered); err != nil {
			s.respondErr(w, err)
			return
		}
		m.LogoThumbURL = media.URLForPtr(m.LogoThumbURL)
		out = append(out, m)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleRepWallet كشف محفظة المندوب (العمولات والتسويات).
func (s *Server) handleRepWallet(w http.ResponseWriter, r *http.Request) {
	// بلا مدى: لمحة اللوحة (آخر 50). بمدى: كشف حساب كامل قابل للطباعة.
	st, err := s.wallet.Statement(r.Context(), userIDFrom(r), statementRange(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}
