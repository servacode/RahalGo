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
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// handleOrdersExport يُخرج طلباتِ مدّةٍ بأنصبتها — CSV بترميزٍ يقرؤه Excel.
//
// ══════════════════════════════════════════════════════════════════════
// **والملفُّ هو الشاشة** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (سجلُّ الطلبات، البند ٢)
// ══════════════════════════════════════════════════════════════════════
//
//   - **بشرط السجلّ نفسِه** (`orders.ListWhere`) — الحالُ والبحثُ والمتجرُ
//     والسائقُ والنوعُ ومدى التاريخ. **فالرقمُ في الشاشة هو الرقمُ في الملف.**
//   - **والطلبُ الخاصُّ فيه** — كان المتجرُ يُضمّ ضمّاً صلباً **فيسقط كلُّ طلبٍ
//     بلا متجر** (١٧ من ١٧٩ على التجهيز) ومالُه معه.
//   - **واليومُ يومُ دمشق** في الشرط وفي عمود التاريخ — كان بتوقيت القاعدة.
//   - **والقيمُ بالعربيّة** — الحالُ والدفعُ والنوع.
//   - **وعمودُ الهاتف لمن يملك `users.contact.read` وحدَه.**
//   - **ولا معادلةَ في خليّة** — نصٌّ يبدأ بـ`=` أو `+` أو `-` أو `@` يُسبَق
//     بفاصلةٍ عليا (حقنُ CSV)، **والأرقامُ والهواتفُ تبقى أرقاماً.**
func (s *Server) handleOrdersExport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	f := orders.ListFilter{
		MerchantID: q.Get("merchant_id"),
		DriverID:   q.Get("driver_id"),
		Query:      q.Get("query"),
		ClosedOnly: q.Get("closed") == "1",
		From:       from,
		To:         to,
		Kind:       q.Get("kind"),
		AnySource:  true,
	}
	where, args, err := orders.ListWhere(f, orders.StuckLimits{})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if st := q.Get("status"); st != "" {
		args = append(args, st)
		where += ` AND o.status = $` + strconv.Itoa(len(args))
	}
	withPhone := s.hasCapability(r, authz.UsersContactRead)

	rows, err := s.pg.Query(r.Context(), `
		SELECT o.number,
		       to_char(o.created_at AT TIME ZONE 'Asia/Damascus', 'YYYY-MM-DD HH24:MI'),
		       o.status, o.payment_method, o.kind,
		       COALESCE(cu.full_name, o.recipient_name, ''), COALESCE(cu.phone::text, o.recipient_phone, ''),
		       COALESCE(mr.name, ''),
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
		LEFT JOIN users cu ON cu.id = o.customer_id
		-- **والمتجرُ يُضمّ يساراً** — الطلبُ الخاصُّ لا متجرَ له (٢٠٢٦-١٠-٠٤).
		LEFT JOIN merchants mr ON mr.id = o.merchant_id
		LEFT JOIN users dr ON dr.id = o.driver_id`+where+`
		ORDER BY o.number`, args...)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	name := "orders.csv"
	if from != "" || to != "" {
		name = "orders-" + from + "_" + to + ".csv"
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	// **BOM** — بدونه يقرأ Excel العربيةَ رموزاً.
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	head := []string{"رقم الطلب", "التاريخ (دمشق)", "الحالة", "الدفع", "النوع", "الزبون"}
	if withPhone {
		head = append(head, "هاتف الزبون")
	}
	head = append(head, "المتجر", "السائق",
		"الأصناف", "التوصيل", "الخصم", "الإجمالي",
		"من المحفظة", "نقداً", "مستحق المتجر", "أجر السائق",
		"عمولة المندوب", "نصيب المنصة", "سبب الإنهاء")
	_ = cw.Write(head)

	count := 0
	for rows.Next() {
		var number int64
		var created, status, pay, kind, customer, phone, merchant, driver, reason string
		var sub, fee, disc, total, wallet, cash, mEarn, dEarn, comm, plat int64
		if err := rows.Scan(&number, &created, &status, &pay, &kind, &customer, &phone,
			&merchant, &driver, &sub, &fee, &disc, &total, &wallet, &cash,
			&mEarn, &dEarn, &comm, &plat, &reason); err != nil {
			s.respondErr(w, err)
			return
		}
		n := func(v int64) string { return strconv.FormatInt(v, 10) }
		if merchant == "" && kind == "custom" {
			merchant = exportNoStore
		}
		line := []string{n(number), created, orders.StatusAr(status), exportPay(pay), exportKind(kind),
			csvSafe(customer)}
		if withPhone {
			line = append(line, csvSafe(phone))
		}
		line = append(line, csvSafe(merchant), csvSafe(driver),
			n(sub), n(fee), n(disc), n(total),
			n(wallet), n(cash), n(mEarn), n(dEarn), n(comm), n(plat), csvSafe(reason))
		_ = cw.Write(line)
		count++
	}
	cw.Flush()
	if err := rows.Err(); err != nil {
		s.logger.Error("orders export", "error", err)
	}
	s.audit(r, "finance.orders_exported", "order", "", map[string]any{
		"from": from, "to": to, "status": q.Get("status"), "query": q.Get("query"),
		"merchant_id": f.MerchantID, "driver_id": f.DriverID, "kind": f.Kind,
		"closed": f.ClosedOnly, "phone": withPhone, "rows": count,
	})
}

// exportNoStore **المتجرُ في الطلب الخاصّ** — لا متجرَ له، فيُقال ذلك لا خليّةٌ فارغة.
const exportNoStore = "طلب خاص — بلا متجر"

// exportPay طريقةُ الدفع بالعربيّة.
func exportPay(p string) string {
	switch p {
	case "cash":
		return "نقداً"
	case "wallet":
		return "محفظة"
	case "mixed":
		return "محفظة ونقد"
	}
	return p
}

// exportKind نوعُ الطلب بالعربيّة.
func exportKind(k string) string {
	switch k {
	case "standard":
		return "طلب متجر"
	case "custom":
		return "طلب خاص"
	case "merchant_delivery":
		return "توصيلة متجر"
	}
	return k
}

// csvSafe **لا معادلةَ في خليّة** — حقنُ CSV (`=HYPERLINK(...)` في اسم زبون).
//
// **والرقمُ والهاتفُ يبقيان كما هما**: `+963…` و`-500` أرقامٌ لا معادلات،
// **وفاصلةٌ عليا قبل الهاتف تُفسده لمن ينسخه.**
func csvSafe(v string) string {
	if v == "" {
		return v
	}
	switch v[0] {
	case '=', '+', '-', '@', '\t', '\r':
		if numericCell(v) {
			return v
		}
		return "'" + v
	}
	return v
}

// numericCell أهي أرقامٌ بعلامةٍ اختياريّةٍ ومسافات — هاتفٌ أو مبلغ.
func numericCell(v string) bool {
	body := strings.TrimLeft(v, "+-")
	if body == "" || len(v)-len(body) > 1 {
		return false
	}
	for _, r := range body {
		if (r < '0' || r > '9') && r != ' ' {
			return false
		}
	}
	return true
}

// handleLedgerExport يُخرج قيودَ الدفتر في مدّةٍ — **الأصلُ الذي تُبنى عليه
// الكشوف.**
//
// **وقيدٌ لكلّ سطر**: من، وكم، ولماذا، وبأيّ مرجع. **ولا يُجمَع هنا** — من
// أراد مجموعاً جمعه في جدوله، **ومن أراد أن يتحقّق وجد السطر.**
//
// ══════════════════════════════════════════════════════════════════════
// **وكالتصدير الآخر** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (التقارير، القرار ١)
// ══════════════════════════════════════════════════════════════════════
//
//   - **عمودُ الهاتف لمن يملك `users.contact.read` وحدَه** — الماليّةُ
//     تملك التصديرَ ومحرومةٌ من الأرقام عمداً، **وكان الملفُّ يُخرجها كاملة.**
//   - **ونوعُ القيد بالعربيّة** (ledgerKindAr) — كان merchant_earning.
//   - **والمدى والوقتُ بيوم دمشق** — كانا بتوقيت القاعدة.
//   - **ولا معادلةَ في خليّة** (csvSafe).
func (s *Server) handleLedgerExport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	if from == "" || to == "" {
		s.respondErr(w, errValidation)
		return
	}
	fd, err := time.Parse("2006-01-02", from)
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	td, err := time.Parse("2006-01-02", to)
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	if fd.After(td) {
		s.respondErr(w, errReportRangeInverted)
		return
	}
	withPhone := s.hasCapability(r, authz.UsersContactRead)

	rows, err := s.pg.Query(r.Context(), `
		SELECT to_char(t.created_at AT TIME ZONE 'Asia/Damascus', 'YYYY-MM-DD HH24:MI'),
		       COALESCE(u.full_name, ''), u.phone::text, t.kind, t.amount,
		       COALESCE(t.note, ''), COALESCE(o.number::text, ''),
		       COALESCE(b.full_name, '')
		FROM wallet_transactions t
		JOIN users u ON u.id = t.user_id
		LEFT JOIN orders o ON o.id::text = t.ref
		LEFT JOIN users b ON b.id = t.created_by
		WHERE t.created_at >= ($1::date::timestamp AT TIME ZONE 'Asia/Damascus')
		  AND t.created_at < (($2::date + 1)::timestamp AT TIME ZONE 'Asia/Damascus')
		ORDER BY t.created_at`, from, to)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="ledger-`+from+"_"+to+`.csv"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	head := []string{"التاريخ (دمشق)", "الاسم"}
	if withPhone {
		head = append(head, "الهاتف")
	}
	head = append(head, "النوع", "المبلغ", "الملاحظة", "رقم الطلب", "بيد")
	_ = cw.Write(head)
	count := 0
	for rows.Next() {
		var created, name, phone, kind, note, orderNo, by string
		var amount int64
		if err := rows.Scan(&created, &name, &phone, &kind, &amount, &note, &orderNo, &by); err != nil {
			s.respondErr(w, err)
			return
		}
		line := []string{created, csvSafe(name)}
		if withPhone {
			line = append(line, csvSafe(phone))
		}
		line = append(line, ledgerKindAr(kind), strconv.FormatInt(amount, 10),
			csvSafe(note), orderNo, csvSafe(by))
		_ = cw.Write(line)
		count++
	}
	cw.Flush()
	if err := rows.Err(); err != nil {
		s.logger.Error("ledger export", "error", err)
	}
	s.audit(r, "finance.ledger_exported", "wallet", "", map[string]any{
		"from": from, "to": to, "phone": withPhone, "rows": count,
	})
}

// ledgerKinds **أسماءُ أنواع القيد بالعربيّة** — كما في معجم الشاشة
// (shared.txKinds)، **ويحرسها اختبارٌ يطابق الاثنين ويطابق قيدَ القاعدة.**
var ledgerKinds = map[string]string{
	"topup":                 "شحن رصيد",
	"order_payment":         "دفع طلب",
	"refund":                "استرجاع",
	"compensation":          "تعويض",
	"commission":            "عمولة",
	"platform_profit":       "ربح المنصة",
	"platform_expense":      "نفقة المنصة",
	"payout":                "سحب رصيد",
	"adjustment":            "تسوية إدارية",
	"operating_expense":     "مصروف تشغيل",
	"merchant_earning":      "مستحق مبيعات",
	"driver_earning":        "أجر توصيل",
	"reward":                "مكافأة",
	"penalty":               "عقوبة",
	"merchant_cash_accrued": "مستحق نقدي للمتجر",
	"merchant_cash_paid":    "دفع نقدي للمتجر",
	"treasury_withdrawal":   "سحب الأدمن من رصيد الخزينة",
}

// ledgerKindAr نوعُ القيد بالعربيّة — والمجهولُ يبقى كما هو لا فراغاً.
func ledgerKindAr(k string) string {
	if v, ok := ledgerKinds[k]; ok {
		return v
	}
	return k
}
