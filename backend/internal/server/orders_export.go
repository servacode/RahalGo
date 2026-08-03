package server

// **تصديرُ الطلبات — لمن يحاسب خارج الشاشة.**
//
// # المسألة
//
// كان التصديرُ للحسابات وحدَها. **ومحاسبٌ يريد كشفاً شهرياً لا يجد ما يأخذه**،
// فينسخ من الشاشة صفحةً صفحة أو يطلب من مبرمجٍ استعلاماً. **وكشفٌ يُبنى بالنسخ
// يُخطئ**، وكشفٌ يُطلب من مبرمجٍ لا يُطلب إلّا مرّةً في السنة.
//
// # وسطرٌ لكلّ طلبٍ بأنصبته
//
// **لا مجاميعُ** — التقاريرُ تُخرج المجاميع بالفعل. **وما ينقص هو الأصل الذي
// تُبنى عليه**: من دفع كم، ولمن ذهب، وماذا بقي. فمن شكّ في مجموعٍ عاد إلى
// السطور، **ومن لا يملك السطورَ يُصدّق أو يشكّ بلا سبيل.**
//
// # والأنصبةُ من الدفتر لا من المعادلة
//
// لو حُسبت هنا لَخالفت ما قُيّد فعلاً حين تتغيّر نسبةٌ — **وكشفٌ يخالف الدفتر
// أسوأُ من غياب الكشف.**

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"
)

// handleOrdersExport يُخرج طلباتِ مدّةٍ بأنصبتها — CSV بترميزٍ يقرؤه Excel.
func (s *Server) handleOrdersExport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	if from == "" || to == "" {
		s.respondErr(w, errValidation)
		return
	}
	if _, err := time.Parse("2006-01-02", from); err != nil {
		s.respondErr(w, errValidation)
		return
	}
	if _, err := time.Parse("2006-01-02", to); err != nil {
		s.respondErr(w, errValidation)
		return
	}

	rows, err := s.pg.Query(r.Context(), `
		SELECT o.number, o.created_at, o.status, o.payment_method,
		       cu.full_name, cu.phone::text, mm.name,
		       COALESCE(dr.full_name, ''),
		       o.subtotal, o.delivery_fee, o.discount, o.total,
		       o.wallet_paid, o.cash_due,
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t
		                 WHERE t.ref = o.id::text AND t.kind = 'merchant_earning'), 0),
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t
		                 WHERE t.ref = o.id::text AND t.kind = 'driver_earning'), 0),
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t
		                 WHERE t.ref = o.id::text AND t.kind = 'commission'), 0),
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t
		                 WHERE t.ref = o.id::text AND t.kind IN ('platform_profit','platform_expense')), 0),
		       COALESCE(o.cancel_reason, '')
		FROM orders o
		JOIN users cu ON cu.id = o.customer_id
		JOIN merchants mm ON mm.id = o.merchant_id
		LEFT JOIN users dr ON dr.id = o.driver_id
		WHERE o.created_at >= $1::date AND o.created_at < ($2::date + 1)
		ORDER BY o.number`, from, to)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="orders.csv"`)
	// **BOM** — بدونه يقرأ Excel العربيةَ رموزاً.
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"رقم الطلب", "التاريخ", "الحالة", "الدفع",
		"الزبون", "هاتف الزبون", "المتجر", "السائق",
		"الأصناف", "التوصيل", "الخصم", "الإجمالي",
		"من المحفظة", "نقداً", "مستحق المتجر", "أجر السائق",
		"عمولة المندوب", "نصيب المنصة", "سبب الإنهاء",
	})

	for rows.Next() {
		var number int64
		var created time.Time
		var status, pay, customer, phone, merchant, driver, reason string
		var sub, fee, disc, total, wallet, cash, mEarn, dEarn, comm, plat int64
		if err := rows.Scan(&number, &created, &status, &pay, &customer, &phone,
			&merchant, &driver, &sub, &fee, &disc, &total, &wallet, &cash,
			&mEarn, &dEarn, &comm, &plat, &reason); err != nil {
			s.respondErr(w, err)
			return
		}
		n := func(v int64) string { return strconv.FormatInt(v, 10) }
		_ = cw.Write([]string{
			n(number), created.Format("2006-01-02 15:04"), status, pay,
			customer, phone, merchant, driver,
			n(sub), n(fee), n(disc), n(total),
			n(wallet), n(cash), n(mEarn), n(dEarn), n(comm), n(plat), reason,
		})
	}
	cw.Flush()
	if err := rows.Err(); err != nil {
		s.logger.Error("orders export", "error", err)
	}
	s.audit(r, "finance.orders_exported", "order", "", map[string]any{"from": from, "to": to})
}

// handleLedgerExport يُخرج قيودَ الدفتر في مدّةٍ — **الأصلُ الذي تُبنى عليه
// الكشوف.**
//
// **وقيدٌ لكلّ سطر**: من، وكم، ولماذا، وبأيّ مرجع. **ولا يُجمَع هنا** — من
// أراد مجموعاً جمعه في جدوله، **ومن أراد أن يتحقّق وجد السطر.**
func (s *Server) handleLedgerExport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	if from == "" || to == "" {
		s.respondErr(w, errValidation)
		return
	}
	if _, err := time.Parse("2006-01-02", from); err != nil {
		s.respondErr(w, errValidation)
		return
	}
	if _, err := time.Parse("2006-01-02", to); err != nil {
		s.respondErr(w, errValidation)
		return
	}

	rows, err := s.pg.Query(r.Context(), `
		SELECT t.created_at, u.full_name, u.phone::text, t.kind, t.amount,
		       COALESCE(t.note, ''), COALESCE(o.number::text, ''),
		       COALESCE(b.full_name, '')
		FROM wallet_transactions t
		JOIN users u ON u.id = t.user_id
		LEFT JOIN orders o ON o.id::text = t.ref
		LEFT JOIN users b ON b.id = t.created_by
		WHERE t.created_at >= $1::date AND t.created_at < ($2::date + 1)
		ORDER BY t.created_at`, from, to)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="ledger.csv"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"التاريخ", "الاسم", "الهاتف", "النوع", "المبلغ", "الملاحظة", "رقم الطلب", "بيد",
	})
	for rows.Next() {
		var created time.Time
		var name, phone, kind, note, orderNo, by string
		var amount int64
		if err := rows.Scan(&created, &name, &phone, &kind, &amount, &note, &orderNo, &by); err != nil {
			s.respondErr(w, err)
			return
		}
		_ = cw.Write([]string{
			created.Format("2006-01-02 15:04"), name, phone, kind,
			strconv.FormatInt(amount, 10), note, orderNo, by,
		})
	}
	cw.Flush()
	if err := rows.Err(); err != nil {
		s.logger.Error("ledger export", "error", err)
	}
	s.audit(r, "finance.ledger_exported", "wallet", "", map[string]any{"from": from, "to": to})
}
