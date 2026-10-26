package server

// **أموالٌ لم تُستلم** — ما في أيدي السائقين ولم يبلغ المكتبَ بعد.
//
// # المسألة
//
// النقدُ الذي يقبضه السائقُ **مالُ المنصة يحمله**، لا مالُه. وكان لا يُرى
// مجموعاً في مكان: **من أراد أن يعرف كم في الشارع فتح كشفَ كلِّ سائقٍ على
// حدة** — فلا يفعل، فلا يعرف.
//
// # ولماذا مدّةُ الحمل بجانب المبلغ
//
// **المبلغُ وحدَه لا يقول شيئاً**: خمسون ألفاً قُبضت قبل ساعة عملٌ يجري،
// وخمسون ألفاً منذ أسبوعٍ مسألةٌ أخرى. **والقِدَمُ هو الإشارة لا المقدار.**
//
// # قراراتُ المالك ٢٠٢٦-١٠-٠٤ (تبويبُ «النقد والصندوق» في الخزينة)
//
//   - «منذ» = أقدمُ مالٍ باقٍ بيده، والأقدمُ يُسدَّد أوّلاً — فالتسليمُ الجزئيُّ
//     لا يُصفّر القِدَم (`driver_cash_oldest_unpaid`، الهجرة 0330).
//   - السقفُ يُعرض كما يمنع: ما بالجيب + نقدُ طلباتٍ بيده لم تُغلق، مقابلَ سقفه
//     الخاصّ إن ضُبط وإلّا العامّ — صيغةُ `cashbox.Exposure` نفسُها.
//   - «متأخّر بالتسليم» بعد `drivers.cash_overdue_days` — تنبيهٌ في الرئيسيّة.
//   - مستحقّاتُ المتاجر النقديّة كلُّها في مكانٍ واحد، أيّاً كانت طريقتُها اليوم.

import (
	"context"
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

type cashHolder struct {
	DriverID string `json:"driver_id"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	// Held ما قبضه ولم يسلّمه (`driver_cash_boxes.held`).
	Held int64 `json:"held"`
	// OpenCash نقدُ طلباتٍ بيده لم تُغلق بعد — يُحسب في السقف كما يُحسب في المنع.
	OpenCash int64 `json:"open_cash"`
	// Exposure = Held + OpenCash — **ما يقارنه حارسُ الإسناد بالسقف.**
	Exposure int64 `json:"exposure"`
	// Limit سقفُه الفعليّ: الخاصُّ إن ضُبط وإلّا العامّ.
	Limit int64 `json:"limit"`
	// OverLimit **فوق سقفه** — التعرّضُ أكبرُ من السقف (قرارُ المالك ٢٠٢٦-١٠-١٥).
	//
	// **والحدُّ حدُّ الحارس نفسُه**: `GuardTx` يقبل حتّى السقف (`≤`) ويردّ ما فوقه،
	// **فمن كان عند السقف تماماً ليس «فوقه»** — كان يُعدّ فوقه (`>=`) فتقول الصفحةُ
	// غيرَ ما يقوله الحارس.
	OverLimit bool `json:"over_limit"`
	// AtLimit **عند السقف تماماً** — لا يُعدّ فوقه، **ولا يأخذ نقديّاً جديداً**
	// (أيُّ واردٍ يتجاوز به السقف). يُعرض «عند السقف».
	AtLimit bool `json:"at_limit"`
	// OldestAt أقدمُ مالٍ باقٍ بيده — والأقدمُ يُسدَّد أوّلاً.
	OldestAt *time.Time `json:"oldest_at"`
	// Overdue مضى على أقدم مالٍ باقٍ عددُ أيّام التنبيه أو أكثر.
	Overdue       bool       `json:"overdue"`
	LastSettledAt *time.Time `json:"last_settled_at"`
	OnShift       bool       `json:"on_shift"`
	// Wallet **حقُّ السائق بمحفظته** (أجورُه) — ليُقرأ الصافي بجانب ما معه للمكتب (قرارُ المالك
	// ٢٠٢٦-١٠-٢٦: «صلّح الأرقام لتكون واضحة»). المتاحُ بعد المحجوز.
	Wallet int64 `json:"wallet"`
}

// cashOverview كشفُ النقد كلُّه بمجاميعه — مصدرٌ واحدٌ للصفحة وللرئيسيّة.
type cashOverview struct {
	Holders      []cashHolder
	Total        int64
	OverCount    int64
	OverdueCount int64
	OverdueDays  int64
}

// handleCashOutstanding من يحمل نقداً ولم يورّده — الأقدمُ أوّلاً.
func (s *Server) handleCashOutstanding(w http.ResponseWriter, r *http.Request) {
	ov, err := s.cashOverviewOf(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"holders": ov.Holders, "total": ov.Total, "limit": s.cashbox.Limit(r.Context()),
		"over_count": ov.OverCount, "overdue_count": ov.OverdueCount,
		"overdue_days": ov.OverdueDays,
	})
}

// cashHolders **من يحمل نقداً ومجموعُه** — لصفحة النقد ولرئيسيّة المدير معاً.
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤: «النقد مع السائقين» في الرئيسيّة **هو رقمُ صفحة
// النقد نفسُه** — لا جمعٌ ثانٍ من جدولٍ آخر.)
func (s *Server) cashHolders(ctx context.Context) ([]cashHolder, int64, error) {
	ov, err := s.cashOverviewOf(ctx)
	if err != nil {
		return nil, 0, err
	}
	return ov.Holders, ov.Total, nil
}

func (s *Server) cashOverviewOf(ctx context.Context) (cashOverview, error) {
	ov := cashOverview{Holders: []cashHolder{}, OverdueDays: s.settings.GetInt(ctx, "drivers.cash_overdue_days")}
	if ov.OverdueDays < 1 {
		ov.OverdueDays = 1
	}
	rows, err := s.pg.Query(ctx, `
		SELECT u.id, COALESCE(u.full_name, ''), u.phone, COALESCE(b.held, 0),
		       COALESCE((SELECT sum(o.cash_due) FROM orders o
		                  WHERE o.driver_id = u.id AND o.closed_at IS NULL), 0) AS open_cash,
		       COALESCE(u.cash_limit_override, $1::bigint) AS lim,
		       x.oldest_at,
		       COALESCE(x.oldest_at <= now() - make_interval(days => $2::int), false),
		       (SELECT max(e.created_at) FROM driver_cash_entries e
		         WHERE e.driver_id = u.id AND e.kind = 'settlement'),
		       COALESCE(u.on_shift, false),
		       COALESCE((SELECT wl.balance - wl.reserved FROM wallets wl WHERE wl.user_id = u.id), 0)
		FROM users u
		LEFT JOIN driver_cash_boxes b ON b.driver_id = u.id
		CROSS JOIN LATERAL (SELECT driver_cash_oldest_unpaid(u.id) AS oldest_at) x
		-- **ومن جيبُه فارغٌ ونقدُ طلباته المفتوحة يبلغ سقفَه يُعرض أيضاً** (قرارُ المالك
		-- ٢٠٢٦-١٠-١٥): كان الشرطُ «بالجيب شيء» وحدَه، **فسائقٌ ممنوعٌ بنقدٍ في الطريق
		-- لا يظهر في الصفحة ولا في «فوق السقف»** — والحارسُ يمنعه.
		WHERE u.id IN (SELECT driver_id FROM driver_cash_boxes WHERE held > 0
		               UNION
		               SELECT driver_id FROM orders
		                WHERE driver_id IS NOT NULL AND closed_at IS NULL AND cash_due > 0)
		-- **والقِدَمُ أوّلاً — وهو الإشارة لا المقدار** (فحصُ المالك ٢٠٢٦-٠٨-١٦)،
		-- **والمقدارُ يفصل بين المتساويين**، وبلا تاريخٍ يهبط.
		ORDER BY x.oldest_at ASC NULLS LAST, COALESCE(b.held, 0) DESC, open_cash DESC`,
		s.cashbox.Limit(ctx), ov.OverdueDays)
	if err != nil {
		return ov, err
	}
	defer rows.Close()
	for rows.Next() {
		var x cashHolder
		if err := rows.Scan(&x.DriverID, &x.Name, &x.Phone, &x.Held, &x.OpenCash,
			&x.Limit, &x.OldestAt, &x.Overdue, &x.LastSettledAt, &x.OnShift, &x.Wallet); err != nil {
			return ov, err
		}
		x.Exposure = x.Held + x.OpenCash
		// **فوق السقف = أكبرُ منه** — حدُّ الحارس نفسُه (`current+incoming > limit`).
		x.OverLimit = x.Exposure > x.Limit
		x.AtLimit = x.Exposure == x.Limit && x.Exposure > 0
		ov.Total += x.Held
		if x.OverLimit {
			ov.OverCount++
		}
		if x.Overdue {
			ov.OverdueCount++
		}
		ov.Holders = append(ov.Holders, x)
	}
	return ov, rows.Err()
}

// handleCashOutstandingExport **كشفُ النقد ملفّاً** — بالترتيب المعروض نفسِه.
func (s *Server) handleCashOutstandingExport(w http.ResponseWriter, r *http.Request) {
	ov, err := s.cashOverviewOf(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="driver-cash.csv"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	_ = cw.Write(cashExportHead)
	n := func(v int64) string { return strconv.FormatInt(v, 10) }
	t := func(v *time.Time) string {
		if v == nil {
			return ""
		}
		return v.In(damascusLoc()).Format("2006-01-02 15:04")
	}
	for _, h := range ov.Holders {
		shift := cashExportOffShift
		if h.OnShift {
			shift = cashExportOnShift
		}
		_ = cw.Write([]string{csvSafe(h.Name), csvSafe(h.Phone), n(h.Held), n(h.OpenCash),
			n(h.Exposure), n(h.Limit), t(h.OldestAt), t(h.LastSettledAt), shift})
	}
	cw.Flush()
	s.audit(r, "finance.cash_exported", "cashbox", "", map[string]any{"rows": len(ov.Holders)})
}

// merchantCashDue متجرٌ له مستحقٌّ نقديٌّ لم يُدفع — أيّاً كانت طريقتُه اليوم.
type merchantCashDue struct {
	MerchantID  string     `json:"merchant_id"`
	Name        string     `json:"name"`
	Method      string     `json:"settlement_method"`
	Outstanding int64      `json:"outstanding"`
	Count       int64      `json:"count"`
	OldestAt    *time.Time `json:"oldest_at"`
}

// merchantCashDueWhere **شرطُ «مستحقٌّ للمتاجر نقداً» — مصدرٌ واحد** (قرارُ المالك ٢٠٢٦-١٠-١٥).
//
// **كان رقمان لمعنًى واحد**: بطاقةُ الخزينة تقرأ رصيدَ محفظة الاحتباس، **وتبويبُ
// المتاجر يجمع صفوفَ التسوية** — **ويفترقان عند أوّل قيدٍ بلا صفٍّ أو صفٍّ بلا قيد**
// (وذاك ما يمسكه `FI-14.a`)، **فيرى المالكُ رقمين ولا يعرف أيَّهما يصدّق.**
//
// **فالمصدرُ الصفوف** — منها يُعرف لمن ومنذ متى، **ورصيدُ الاحتباس يُحرَس بها لا
// يحلّ محلَّها.** (`ms` اسمُ الجدول في الاستعلامين.)
const merchantCashDueWhere = `ms.method = 'cash' AND ms.state = 'cash_due'
		  AND ms.amount > ms.reversed_amount`

// merchantCashDueTotalSQL **مجموعُه** — يقرؤه نظرُ الخزينة، **بشرط التبويب نفسِه.**
const merchantCashDueTotalSQL = `(SELECT COALESCE(sum(ms.amount - ms.reversed_amount), 0)
		  FROM merchant_settlements ms WHERE ` + merchantCashDueWhere + `)`

// handleMerchantCashDues **مستحقّاتُ المتاجر النقديّة كلُّها في مكانٍ واحد.**
//
// كان الكشفُ داخل ملفّ كلّ متجر، **ولا يظهر إلّا إن كانت طريقتُه اليوم «نقد»** —
// فمتجرٌ له مئتا ألفٍ ثمّ حُوّل إلى المحفظة اختفى مستحقُّه. والمستحقُّ يُقرأ
// من صفوفه (`method = 'cash'`) لا من طريقة المتجر الحاليّة.
func (s *Server) handleMerchantCashDues(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT m.id::text, COALESCE(m.name, ''), m.settlement_method,
		       sum(ms.amount - ms.reversed_amount), count(*), min(ms.created_at)
		FROM merchant_settlements ms
		JOIN merchants m ON m.id = ms.merchant_id
		WHERE `+merchantCashDueWhere+`
		GROUP BY m.id, m.name, m.settlement_method
		ORDER BY min(ms.created_at) ASC`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []merchantCashDue{}
	var total int64
	for rows.Next() {
		var d merchantCashDue
		if err := rows.Scan(&d.MerchantID, &d.Name, &d.Method, &d.Outstanding,
			&d.Count, &d.OldestAt); err != nil {
			s.respondErr(w, err)
			return
		}
		total += d.Outstanding
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"merchants": out, "total": total})
}
