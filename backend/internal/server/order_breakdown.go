package server

// تفصيلُ مال الطلب — من دفع، ومن قبض، وكم بقي للمنصة.
//
// # ولماذا يُقرأ من الدفتر لا يُحسب بجانبه
//
// كان يمكن أن تُحسب الأنصبةُ هنا بالمعادلات: عمولةُ المتجر من نسبته، وأجرُ
// السائق من الإعدادات، ونصيبُ المندوب من عمولة المنصة. **وتلك حسبةٌ ثانيةٌ
// بجانب الحسبة التي دفعت فعلاً — وحسبتان تفترقان يوماً.**
//
// فتُغيَّر نسبةُ السائق اليوم، فتظهر طلباتُ الأمس بأجرٍ لم يقبضه. أو يُرفض
// قيدٌ لنقص رصيد، **فتقول الشاشةُ إنه قبض وهو لم يقبض.**
//
// **فالمصدرُ واحد: `wallet_transactions` بمرجع الطلب.** ما تراه هنا هو ما وقع
// في الدفتر حرفياً — **وشاشةٌ تقرأ الدفتر لا تكذب عليه.**
//
// # وأربعةُ أطرافٍ لا ثلاثة
//
// الزبونُ يدفع، والمتجرُ والسائقُ والمندوبُ يقبضون، **والمنصةُ تأخذ ما بقي.**
// وكان الطرفُ الرابع غائباً عن كل شاشة: تُعرض العمولةُ ولا يُعرض ما خرج منها.

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

type breakdownLine struct {
	// Party merchant | driver | sales | platform | customer
	Party  string `json:"party"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Amount int64  `json:"amount"`
	Note   string `json:"note"`
}

// handleOrderBreakdown تفصيلُ توزيع مبلغ الطلب.
func (s *Server) handleOrderBreakdown(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	if !isUUID(orderID) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}

	var total, walletPaid, cashDue, deliveryFee, subtotal, discount int64
	var status string
	if err := s.pg.QueryRow(r.Context(), `
		SELECT total, wallet_paid, cash_due, delivery_fee, subtotal, discount, status
		FROM orders WHERE id = $1`, orderID).
		Scan(&total, &walletPaid, &cashDue, &deliveryFee, &subtotal, &discount, &status); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}

	// **كلُّ قيدٍ حمل مرجعَ هذا الطلب** — بلا استثناءٍ ولا انتقاء.
	//
	// وبالاختيار لا بالمسح هنا عمداً، عكسَ ردّ الخصوصية: هناك كان يُخشى تسريبُ
	// حقلٍ جديد، **وهنا يُخشى إخفاءُ قيدٍ جديد** — ونوعٌ لا يُعرض يعني مالاً
	// تحرّك ولا يظهر في التفصيل، وهو أسوأ من عدم التفصيل أصلاً.
	rows, err := s.pg.Query(r.Context(), `
		SELECT t.kind, t.amount, t.note,
		       COALESCE(NULLIF(u.full_name, ''), u.phone::text),
		       COALESCE((SELECT string_agg(ur.role_code, ',' ORDER BY ur.role_code)
		                 FROM user_roles ur WHERE ur.user_id = t.user_id), '')
		FROM wallet_transactions t
		JOIN users u ON u.id = t.user_id
		WHERE t.ref = $1
		ORDER BY t.created_at, t.id`, orderID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	lines := []breakdownLine{}
	var toParties, platform int64
	for rows.Next() {
		var l breakdownLine
		var roles string
		if err := rows.Scan(&l.Kind, &l.Amount, &l.Note, &l.Name, &roles); err != nil {
			s.respondErr(w, err)
			return
		}
		// **الطرفُ من نوع القيد لا من دور المستخدم.**
		//
		// الأدوارُ تتغيّر: مندوبٌ يصير موظّفَ عمليات فتُقرأ عمولاتُه القديمة
		// على أنها مصاريفُ إدارة. **ونوعُ القيد لا يتغيّر أبداً.**
		switch l.Kind {
		case "merchant_earning":
			l.Party = "merchant"
			toParties += l.Amount
		case "driver_earning":
			l.Party = "driver"
			toParties += l.Amount
		case "commission":
			l.Party = "sales"
			toParties += l.Amount
		case "platform_profit":
			l.Party = "platform"
			platform += l.Amount
		case "order_payment", "refund":
			l.Party = "customer"
		default:
			// تعويضٌ أو تسوية — يُعرض باسمه ولا يُبتلع.
			l.Party = "other"
		}
		lines = append(lines, l)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"total":        total,
		"subtotal":     subtotal,
		"delivery_fee": deliveryFee,
		"discount":     discount,
		"wallet_paid":  walletPaid,
		"cash_due":     cashDue,
		"status":       status,
		// **مجاميعُ مقروءةٌ من الأسطر نفسها** — لا محسوبةٌ من معادلةٍ ثانية.
		"to_parties": toParties,
		"platform":   platform,
		"lines":      lines,
	})
}
