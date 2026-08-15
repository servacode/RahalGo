package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// كشف مالي لأي شخص حسب دوره: ما استحقه (له)، وما عليه، والطلبات المرتجعة وأسبابها.
// يبني على دفتر القيود القائم (wallet_transactions) ولقطات الطلبات — لا يخترع أرصدة.

type finRate struct {
	Label   string `json:"label"`
	Percent int    `json:"percent"`
}

type finEntry struct {
	// Ref **رقمُ الطلب — لا معرّفُه.**
	//
	// (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٥.)
	//
	// **كان يُرسَل `o.id::text`** — والشاشةُ تبني منه
	// `‎/dashboard/orders?q=<معرّف>`، **وبحثُ الطلبات يطابق الرقمَ أو
	// الهاتفَ لا المعرّف.** فكلُّ سطرٍ في كشف سائقٍ أو متجرٍ يُضغط
	// **فيردّ «لا نتائج»** — ولا خطأ ولا إنذار: شاشةٌ تعمل وتُجيب بالفراغ.
	//
	// **وفارغٌ يعني «لا طلبَ له»** — كسطر «نقدٌ بحوزته»، فلا يُرسم رابطا.
	Ref    string    `json:"ref"`
	Label  string    `json:"label"` // اسم المتجر / وصف الحركة
	Amount int64     `json:"amount"`
	Status string    `json:"status"` // للطلبات المرتجعة فقط
	Reason string    `json:"reason"` // سبب الإرجاع/الإلغاء
	Date   time.Time `json:"date"`
}

type finBucket struct {
	// Total **مجموعُ المال — على كلّ الصفوف لا على المعروض.**
	Total int64 `json:"total"`
	// Count **عددُ الصفوف كلِّها.**
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «لا تنسَ إضافة الباجينيشن».)
	//
	// **والقائمةُ هنا عيّنةٌ لا جردٌ**: المجموعُ يُحسب على الكلّ باستعلامٍ
	// مستقلّ، **والأسطرُ أحدثُ خمسين.** فيُقال العددُ صراحةً — **وسقفٌ صامتٌ
	// يُقرأ «هذا كلُّ ما عليه»**، فيُصالَح المتجرُ على نصف دينه.
	Count int        `json:"count"`
	Items []finEntry `json:"items"`
}

// finLimit **أحدثُ ما يُعرض من كلّ سلّة.**
//
// **والمجاميعُ لا تتعلّق به** — تُحسب باستعلامٍ مستقلٍّ على كلّ الصفوف.
const finLimit = 50

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
		// **وعددُ المرتجعات كلِّها** — والمعروضُ أحدثُ خمسين.
		ReturnsCount int `json:"returns_count"`
		// **وسقفُ العرض يُرسَل** — فتقول الشاشةُ «أحدثُ ٥٠ من ٢١٣» بلا رقمٍ
		// مكتوبٍ فيها **يفترق عن رقم الخادم يوماً.**
		Limit int `json:"limit"`
	}{Limit: finLimit, Roles: roles, Rates: []finRate{}, OwedTo: finBucket{Items: []finEntry{}}, OwedBy: finBucket{Items: []finEntry{}}, Returns: []finEntry{}}

	// ---- النِسَب المطبّقة حسب الدور ----
	// **والنسبةُ المعروضة هي النافذةُ لا رقمٌ يُقرأ من عمود.**
	//
	// كانت تُقرأ بـSQL هنا وبـSQL في التسوية — **ورقمان لمعنًى واحدٍ يفترقان**،
	// فيرى المندوبُ نسبةً ويُقيَّد له بغيرها.
	if has("sales") {
		rep := pricing.RepCommission(ctx, s.settings)
		out.Rates = append(out.Rates, finRate{
			Label: "نسبة عمولة المندوب من عمولة المنصة", Percent: int(rep.Value)})
	}
	if has("merchant") {
		rows, err := s.pg.Query(ctx,
			`SELECT name, commission_percent FROM merchants WHERE owner_user_id = $1 ORDER BY name`, id)
		if err == nil {
			for rows.Next() {
				var name string
				var override *int64
				if rows.Scan(&name, &override) == nil {
					rate := pricing.MerchantCommission(ctx, s.settings, override)
					out.Rates = append(out.Rates, finRate{
						Label: "عمولة المنصة على " + name, Percent: int(rate.Value)})
				}
			}
			rows.Close()
		}
	}

	// ---- مستحق له: عمولات المندوب المقيّدة في محفظته ----
	if has("sales") {
		var sum int64
		_ = s.pg.QueryRow(ctx,
			`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions WHERE user_id = $1 AND kind = 'commission'`,
			id).Scan(&sum)
		out.OwedTo.Total += sum
		var n int
		_ = s.pg.QueryRow(ctx,
			`SELECT count(*) FROM wallet_transactions WHERE user_id = $1 AND kind = 'commission'`,
			id).Scan(&n)
		out.OwedTo.Count += n
		// **ومرجعُ الحركة معرّفُ طلبٍ** — يُترجَم إلى رقمه، **وهو ما يفتحه
		// البحثُ في شاشة الطلبات.** وفارغٌ لحركةٍ لا طلبَ لها.
		rows, err := s.pg.Query(ctx, `
			SELECT COALESCE(o.number::text, ''),
			       COALESCE(NULLIF(t.note, ''), 'عمولة'), t.amount, t.created_at
			FROM wallet_transactions t
			LEFT JOIN orders o ON t.ref <> '' AND o.id::text = t.ref
			WHERE t.user_id = $1 AND t.kind = 'commission'
			ORDER BY t.created_at DESC LIMIT `+strconv.Itoa(finLimit), id)
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

	// ---- مستحق له: أجور توصيل السائق على الطلبات التي سلّمها ----
	if has("driver") {
		var fees int64
		_ = s.pg.QueryRow(ctx, `
			SELECT COALESCE(sum(delivery_fee), 0) FROM orders
			WHERE driver_id = $1 AND status = 'delivered'`, id).Scan(&fees)
		out.OwedTo.Total += fees
		var n int
		_ = s.pg.QueryRow(ctx, `
			SELECT count(*) FROM orders
			WHERE driver_id = $1 AND status = 'delivered' AND delivery_fee > 0`, id).Scan(&n)
		out.OwedTo.Count += n
		// **والمتجرُ يُضمّ يساراً** — **والطلبُ الخاصُّ لا متجرَ له**، وضمٌّ
		// صلبٌ يُسقطه من كشف السائق بلا خطأ.
		rows, err := s.pg.Query(ctx, `
			SELECT o.number::text, COALESCE(m.name, ''), o.delivery_fee, o.created_at
			FROM orders o LEFT JOIN merchants m ON m.id = o.merchant_id
			WHERE o.driver_id = $1 AND o.status = 'delivered' AND o.delivery_fee > 0
			ORDER BY o.created_at DESC LIMIT `+strconv.Itoa(finLimit), id)
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
		_ = s.pg.QueryRow(ctx, `
			SELECT count(*) FROM orders o JOIN merchants m ON m.id = o.merchant_id
			WHERE m.owner_user_id = $1 AND o.status = 'delivered'
			  AND o.platform_commission > 0`, id).Scan(&out.OwedBy.Count)
		rows, err := s.pg.Query(ctx, `
			SELECT o.number::text, m.name, o.platform_commission, o.created_at
			FROM orders o JOIN merchants m ON m.id = o.merchant_id
			WHERE m.owner_user_id = $1 AND o.status = 'delivered' AND o.platform_commission > 0
			ORDER BY o.created_at DESC LIMIT `+strconv.Itoa(finLimit), id)
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
	// **والمتجرُ يُضمّ يساراً هنا أيضاً** — **وطلبٌ خاصٌّ أُلغي كان يختفي من
	// كشف صاحبه**، وهو أوّلُ ما يُسأل عنه عند شكوى.
	_ = s.pg.QueryRow(ctx, `
		SELECT count(*) FROM orders o
		LEFT JOIN merchants m ON m.id = o.merchant_id
		WHERE (m.owner_user_id = $1 OR o.driver_id = $1 OR o.customer_id = $1)
		  AND o.status IN ('cancelled', 'rejected', 'failed', 'refunded')`,
		id).Scan(&out.ReturnsCount)
	rows, err := s.pg.Query(ctx, `
		SELECT o.number::text, COALESCE(m.name, ''), o.subtotal, o.status, o.cancel_reason, o.created_at
		FROM orders o LEFT JOIN merchants m ON m.id = o.merchant_id
		WHERE (m.owner_user_id = $1 OR o.driver_id = $1 OR o.customer_id = $1)
		  AND o.status IN ('cancelled', 'rejected', 'failed', 'refunded')
		ORDER BY o.created_at DESC LIMIT `+strconv.Itoa(finLimit), id)
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
