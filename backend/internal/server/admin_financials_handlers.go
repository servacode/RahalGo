package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// كشف مالي لأي شخص حسب دوره: ما استحقه (له)، وما عليه، والطلبات المرتجعة وأسبابها.
// يبني على دفتر القيود القائم (wallet_transactions) ولقطات الطلبات — لا يخترع أرصدة.

type finRate struct {
	Label   string `json:"label"`
	Percent int    `json:"percent"`
}

type finEntry struct {
	Ref    string    `json:"ref"`   // رقم الطلب أو المرجع
	Label  string    `json:"label"` // اسم المتجر / وصف الحركة
	Amount int64     `json:"amount"`
	Status string    `json:"status"` // للطلبات المرتجعة فقط
	Reason string    `json:"reason"` // سبب الإرجاع/الإلغاء
	Date   time.Time `json:"date"`
}

type finBucket struct {
	Total int64      `json:"total"`
	Items []finEntry `json:"items"`
}

func (s *Server) handleAdminUserFinancials(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()

	var roles []string
	if err := s.pg.QueryRow(ctx,
		`SELECT COALESCE(array_agg(role_code), '{}') FROM user_roles WHERE user_id = $1`, id).
		Scan(&roles); err != nil {
		s.respondErr(w, err)
		return
	}
	has := func(role string) bool {
		for _, x := range roles {
			if x == role {
				return true
			}
		}
		return false
	}

	out := struct {
		Roles   []string   `json:"roles"`
		Rates   []finRate  `json:"rates"`
		OwedTo  finBucket  `json:"owed_to"`
		OwedBy  finBucket  `json:"owed_by"`
		Returns []finEntry `json:"returns"`
	}{Roles: roles, Rates: []finRate{}, OwedTo: finBucket{Items: []finEntry{}}, OwedBy: finBucket{Items: []finEntry{}}, Returns: []finEntry{}}

	// ---- النِسَب المطبّقة حسب الدور ----
	if has("sales") {
		var pct int
		_ = s.pg.QueryRow(ctx, `
			SELECT COALESCE((SELECT (value#>>'{}')::int FROM app_settings
			                 WHERE key = 'sales.commission_percent'), 10)`).Scan(&pct)
		out.Rates = append(out.Rates, finRate{Label: "نسبة عمولة المندوب من عمولة المنصة", Percent: pct})
	}
	if has("merchant") {
		rows, err := s.pg.Query(ctx,
			`SELECT name, commission_percent FROM merchants WHERE owner_user_id = $1 ORDER BY name`, id)
		if err == nil {
			for rows.Next() {
				var name string
				var pct int
				if rows.Scan(&name, &pct) == nil {
					out.Rates = append(out.Rates, finRate{Label: "عمولة المنصة على " + name, Percent: pct})
				}
			}
			rows.Close()
		}
	}

	// ---- مستحق له: عمولات المندوب المقيّدة في محفظته ----
	if has("sales") {
		_ = s.pg.QueryRow(ctx,
			`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions WHERE user_id = $1 AND kind = 'commission'`,
			id).Scan(&out.OwedTo.Total)
		rows, err := s.pg.Query(ctx, `
			SELECT t.ref, COALESCE(NULLIF(t.note, ''), 'عمولة'), t.amount, t.created_at
			FROM wallet_transactions t
			WHERE t.user_id = $1 AND t.kind = 'commission'
			ORDER BY t.created_at DESC LIMIT 50`, id)
		if err == nil {
			for rows.Next() {
				var e finEntry
				if rows.Scan(&e.Ref, &e.Label, &e.Amount, &e.Date) == nil {
					out.OwedTo.Items = append(out.OwedTo.Items, e)
				}
			}
			rows.Close()
		}
	}

	// ---- مستحق عليه ----
	if has("merchant") {
		// عمولة المنصة على طلبات متاجره المُسلَّمة (يدين بها للمنصة)
		_ = s.pg.QueryRow(ctx, `
			SELECT COALESCE(sum(o.platform_commission), 0)
			FROM orders o JOIN merchants m ON m.id = o.merchant_id
			WHERE m.owner_user_id = $1 AND o.status = 'delivered'`, id).Scan(&out.OwedBy.Total)
		rows, err := s.pg.Query(ctx, `
			SELECT o.id::text, m.name, o.platform_commission, o.created_at
			FROM orders o JOIN merchants m ON m.id = o.merchant_id
			WHERE m.owner_user_id = $1 AND o.status = 'delivered' AND o.platform_commission > 0
			ORDER BY o.created_at DESC LIMIT 50`, id)
		if err == nil {
			for rows.Next() {
				var e finEntry
				if rows.Scan(&e.Ref, &e.Label, &e.Amount, &e.Date) == nil {
					out.OwedBy.Items = append(out.OwedBy.Items, e)
				}
			}
			rows.Close()
		}
	}
	if has("driver") {
		// النقد بحوزته يدين به للمنصة
		var held int64
		_ = s.pg.QueryRow(ctx,
			`SELECT COALESCE((SELECT held FROM driver_cash_boxes WHERE driver_id = $1), 0)`, id).Scan(&held)
		out.OwedBy.Total += held
		if held > 0 {
			out.OwedBy.Items = append(out.OwedBy.Items, finEntry{Label: "نقد بحوزته", Amount: held})
		}
	}

	// ---- الطلبات المرتجعة/الملغاة وأسبابها (تخص المستخدم كمتجر/سائق/زبون) ----
	rows, err := s.pg.Query(ctx, `
		SELECT o.id::text, m.name, o.subtotal, o.status, o.cancel_reason, o.created_at
		FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE (m.owner_user_id = $1 OR o.driver_id = $1 OR o.customer_id = $1)
		  AND o.status IN ('cancelled', 'rejected', 'failed', 'refunded')
		ORDER BY o.created_at DESC LIMIT 50`, id)
	if err == nil {
		for rows.Next() {
			var e finEntry
			if rows.Scan(&e.Ref, &e.Label, &e.Amount, &e.Status, &e.Reason, &e.Date) == nil {
				out.Returns = append(out.Returns, e)
			}
		}
		rows.Close()
	}

	httpx.JSON(w, http.StatusOK, out)
}
