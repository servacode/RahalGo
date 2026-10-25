package server

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// ══════════════════════════════════════════════════════════════════════
// **البحثُ العامُّ في اللوحة** — `GET /admin/search?q=` (تصميمُ اللوحة ٢٠٢٦-١٠-٢٥)
// ══════════════════════════════════════════════════════════════════════
//
// **خانةٌ واحدةٌ أعلى كلّ صفحة**: رقمُ طلبٍ أو هاتفٌ أو اسم — **فيصل الموظّفُ إلى
// الطلب أو الحساب أو المتجر من أيّ مكان** بدل أن يتذكّر أيَّ شاشةٍ تحمله.
//
// **وكلُّ قسمٍ بقدرته** — الطلباتُ لمن يقرأ الطلبات، والحساباتُ لمن يقرأ الحسابات،
// والمتاجرُ لمن يقرأ سجلّها. **ومن لا يملك قسماً لا يُرسَل له مفتاحُه صفراً.**
// والتطبيعُ هو تطبيعُ قائمة الطلبات نفسُه (`orders.NormalizeSearch`) — فما يجده
// هنا يجده هناك.

type searchOrderHit struct {
	ID           string `json:"id"`
	Number       int64  `json:"number"`
	Status       string `json:"status"`
	Kind         string `json:"kind"`
	MerchantName string `json:"merchant_name"`
	CustomerName string `json:"customer_name"`
	Open         bool   `json:"open"`
	CreatedAt    string `json:"created_at"`
}

type searchUserHit struct {
	ID       string   `json:"id"`
	FullName string   `json:"full_name"`
	Phone    string   `json:"phone"`
	Roles    []string `json:"roles"`
	Status   string   `json:"status"`
}

type searchMerchantHit struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Number int64  `json:"number"`
	Status string `json:"status"`
}

const searchLimit = 6

func (s *Server) handleAdminSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	t := orders.NormalizeSearch(r.URL.Query().Get("q"))
	out := map[string]any{}
	// **وحرفٌ واحدٌ لا يُمسح به جدول** — كبحث الخريطة.
	if t == (orders.SearchTerms{}) || (t.Name != "" && len([]rune(t.Name)) < 2) {
		httpx.JSON(w, http.StatusOK, out)
		return
	}

	if hasCap(r, authz.OrdersRead) {
		hits := []searchOrderHit{}
		rows, err := s.pg.Query(ctx, `
			SELECT o.id::text, o.number, o.status, o.kind,
			       COALESCE(mr.name, ''), COALESCE(cu.full_name, o.recipient_name, ''),
			       o.closed_at IS NULL, to_char(o.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
			FROM orders o
			LEFT JOIN users cu ON cu.id = o.customer_id
			LEFT JOIN merchants mr ON mr.id = o.merchant_id
			WHERE ($1::bigint > 0 AND o.number = $1)
			   OR ($2 <> '' AND COALESCE(cu.phone::text, o.recipient_phone, '') LIKE '%' || $2 || '%' ESCAPE '\')
			   OR ($3 <> '' AND COALESCE(cu.full_name, o.recipient_name, '') ILIKE '%' || $3 || '%' ESCAPE '\')
			ORDER BY (o.number = $1) DESC, (o.closed_at IS NULL) DESC, o.created_at DESC
			LIMIT $4`, t.Number, orders.EscapeLike(t.Phone), t.Name, searchLimit)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		for rows.Next() {
			var h searchOrderHit
			if err := rows.Scan(&h.ID, &h.Number, &h.Status, &h.Kind, &h.MerchantName,
				&h.CustomerName, &h.Open, &h.CreatedAt); err != nil {
				rows.Close()
				s.respondErr(w, err)
				return
			}
			hits = append(hits, h)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			s.respondErr(w, err)
			return
		}
		out["orders"] = hits
	}

	if hasCap(r, authz.UsersRead) {
		hits := []searchUserHit{}
		rows, err := s.pg.Query(ctx, `
			SELECT u.id::text, u.full_name, u.phone::text, u.status,
			       COALESCE(array_agg(ur.role_code ORDER BY ur.role_code)
			                FILTER (WHERE ur.role_code IS NOT NULL), '{}')
			FROM users u
			LEFT JOIN user_roles ur ON ur.user_id = u.id
			WHERE u.deleted_at IS NULL
			  AND (($1 <> '' AND u.phone::text LIKE '%' || $1 || '%' ESCAPE '\')
			    OR ($2 <> '' AND u.full_name ILIKE '%' || $2 || '%' ESCAPE '\'))
			GROUP BY u.id
			ORDER BY u.full_name
			LIMIT $3`, orders.EscapeLike(t.Phone), t.Name, searchLimit)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		for rows.Next() {
			var h searchUserHit
			if err := rows.Scan(&h.ID, &h.FullName, &h.Phone, &h.Status, &h.Roles); err != nil {
				rows.Close()
				s.respondErr(w, err)
				return
			}
			hits = append(hits, h)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			s.respondErr(w, err)
			return
		}
		out["users"] = hits
	}

	if hasCap(r, authz.MerchantsRead) {
		hits := []searchMerchantHit{}
		rows, err := s.pg.Query(ctx, `
			SELECT m.id::text, m.name, m.number, m.status
			FROM merchants m
			WHERE ($1 <> '' AND m.name ILIKE '%' || $1 || '%' ESCAPE '\')
			   OR ($2 <> '' AND m.phone::text LIKE '%' || $2 || '%' ESCAPE '\')
			   OR ($3::bigint > 0 AND m.number = $3)
			ORDER BY m.name
			LIMIT $4`, t.Name, orders.EscapeLike(t.Phone), t.Number, searchLimit)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		for rows.Next() {
			var h searchMerchantHit
			if err := rows.Scan(&h.ID, &h.Name, &h.Number, &h.Status); err != nil {
				rows.Close()
				s.respondErr(w, err)
				return
			}
			hits = append(hits, h)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			s.respondErr(w, err)
			return
		}
		out["merchants"] = hits
	}

	httpx.JSON(w, http.StatusOK, out)
}
