package server

/*
**مصروفاتُ التشغيل — ما ينفقه المكتبُ لا ما يخسره العمل.**

(قرارُ المالك ٢٠٢٦-٠٨-١٦: «يوجد مكتبٌ للشركة وموظّفون وعمّال… يجب أن تُوثَّق
 بشكلٍ صحيح» · «نعم يخرج من خزينة المنصّة لأنّه مصروفٌ تابعٌ للمنصّة» · «قائمةٌ
 ويمكنني الإضافة والحذف والتعديل» · «أسجّل كلَّ شيءٍ بيدي».)

# والمالُ يخرج من الخزينة فعلاً

**قاعدةُ المالك ٢٠٢٦-٠٨-٠٤**: «لا يُدفع لأحدٍ إلّا وخرج من الخزينة، ولا يدخل
مالٌ إلّا ودخلها».

**فكلُّ مصروفٍ قيدان**: صفٌّ يقول **على أيّ بابٍ صُرف**، وقيدٌ في الخزينة يقول
**كم نقص رصيدُها**. **وصفٌّ بلا قيدٍ يجعل الشاشةَ تقول ما لا تقوله الخزينة.**

# ولا يُخلط بالخسائر

**`operating_expense` لا `platform_expense`** — والثاني تقرؤه شاشةُ الخسائر
كلَّه. **وإيجارُ المكتب ليس خسارة**، ولو خُلطا لَتضخّم تقريرُ الخسائر بالإيجار
**فيبدو أداءُ المنصّة أسوأَ ممّا هو**، ولا يُعرف كم كلّف الفشلُ فعلاً.

# قراراتُ المالك ٢٠٢٦-١٠-٠٤

	١ · فوق السقف (`finance.expense_approval_threshold`) يصير المصروفُ اقتراحاً في
	    `expense_requests` يوافق عليه شخصٌ آخر (`approval.Check`)، وتحته يُقيَّد مباشرة.
	٢ · الشهرُ بتاريخ الصرف في صفحة المصروفات وفي الأرباح (`opexBySpendDate`).
	٤ · صورةُ الإيصال اختياريّة، وإلزاميّةٌ فوق السقف.
	٥ · الإلغاءُ بسببٍ إلزاميٍّ ومن شخصٍ غيرِ من سجّل، والملغى يبقى ظاهراً مشطوباً.
*/

import (
	"context"
	"encoding/csv"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/approval"
	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

var (
	errExpenseFutureDate = httpx.NewError(http.StatusBadRequest,
		"expense_future_date", "errors.expense_future_date")
	errExpenseReceiptRequired = httpx.NewError(http.StatusBadRequest,
		"expense_receipt_required", "errors.expense_receipt_required")
	errExpenseVoidReason = httpx.NewError(http.StatusBadRequest,
		"expense_void_reason_required", "errors.expense_void_reason_required")
	errExpenseVoidSelf = httpx.NewError(http.StatusForbidden,
		"expense_void_self", "errors.expense_void_self")
	errExpenseCategoryOff = httpx.NewError(http.StatusBadRequest,
		"expense_category_inactive", "errors.expense_category_inactive")
	errExpenseBadRange = httpx.NewError(http.StatusBadRequest,
		"expense_bad_range", "errors.expense_bad_range")
)

// expenseApprovalThreshold **سقفُ الموافقة الثانية** — من الإعدادات.
func (s *Server) expenseApprovalThreshold(ctx context.Context) int64 {
	return s.settings.GetNum(ctx, "finance.expense_approval_threshold", 500000)
}

// damascusLoc منطقةُ المنصّة — وبلا قاعدةِ مناطقٍ لا يسقط الطلب.
func damascusLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Damascus")
	if err != nil {
		return time.UTC
	}
	return loc
}

// expenseToday **يومُ دمشق** — وكان يومَ الخادم (UTC)، فمن سجّل بين منتصف
// الليل والثالثة فجراً رأى تاريخَ أمس.
func expenseToday() string {
	return time.Now().In(damascusLoc()).Format("2006-01-02")
}

// treasuryExpenseNote **نصُّ قيد الخزينة**: «مصروف تشغيل — <الباب> — <ملاحظة>».
//
// **وكان الملاحظةَ وحدَها** — فإن فرغت لم يُعرف من الكشف على ماذا صُرف المال.
func treasuryExpenseNote(prefix, category, note string) string {
	out := prefix + " — " + category
	if n := strings.TrimSpace(note); n != "" {
		out += " — " + n
	}
	return clip(out, 300)
}

// opexBySpendDate **مصروفاتُ التشغيل في مدًى بتاريخ الصرف** — قرارُ المالك
// ٢٠٢٦-١٠-٠٤ (المصروفات، البند ٢): «الشهرُ بتاريخ الصرف في المكانين».
//
// **وإيجارُ أيلول يُسجَّل في تشرين فيُعدّ في أيلول**، والملغى يخرج من شهره
// الأصليّ. **وتقرؤه صفحةُ الأرباح** فتطابق صفحةَ المصروفات حرفاً.
// `from`/`to` بصيغة `YYYY-MM-DD` شاملان، وفارغٌ بلا حدّ.
func opexBySpendDate(ctx context.Context, q dbtx.Querier, from, to string) (int64, error) {
	var sum int64
	err := q.QueryRow(ctx, `
		SELECT COALESCE(sum(amount), 0) FROM expenses
		WHERE voided_at IS NULL
		  AND ($1 = '' OR spent_at >= NULLIF($1, '')::date)
		  AND ($2 = '' OR spent_at <= NULLIF($2, '')::date)`, from, to).Scan(&sum)
	return sum, err
}

// handleExpenseCategories **أبوابُ المصروف — تُدار ولا تُكتب في الشيفرة.**
func (s *Server) handleExpenseCategories(w http.ResponseWriter, r *http.Request) {
	// **والمُطفأةُ تُقرأ في الإدارة** — لتُعاد، **ولتُقرأ أسماءُ ما مضى.**
	// **ومع كلّ بابٍ مبلغُه** — والعددُ وحدَه لا يقول أين ذهب المال.
	rows, err := s.pg.Query(r.Context(), `
		SELECT c.id::text, c.name, c.active, c.sort_order,
		       (SELECT count(*) FROM expenses e
		        WHERE e.category_id = c.id AND e.voided_at IS NULL),
		       (SELECT COALESCE(sum(e.amount), 0) FROM expenses e
		        WHERE e.category_id = c.id AND e.voided_at IS NULL)
		FROM expense_categories c
		ORDER BY c.sort_order, c.name`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type cat struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Active bool   `json:"active"`
		Sort   int    `json:"sort_order"`
		Used   int    `json:"used"`
		Spent  int64  `json:"spent"`
	}
	out := []cat{}
	for rows.Next() {
		var c cat
		if err := rows.Scan(&c.ID, &c.Name, &c.Active, &c.Sort, &c.Used, &c.Spent); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"categories": out})
}

// handleSaveExpenseCategory **يُضاف بابٌ أو يُعدَّل اسمُه أو ترتيبُه أو يُطفأ.**
func (s *Server) handleSaveExpenseCategory(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		ID     *string `json:"id"`
		Name   string  `json:"name"`
		Active *bool   `json:"active"`
		Sort   *int    `json:"sort_order"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if req.ID == nil && name == "" {
		s.respondErr(w, errValidation)
		return
	}
	if req.ID == nil {
		var id string
		var sort int
		if err := s.pg.QueryRow(r.Context(), `
			INSERT INTO expense_categories (name, sort_order)
			VALUES ($1, COALESCE($2, (SELECT COALESCE(max(sort_order), 0) + 1
			                          FROM expense_categories WHERE sort_order < 99)))
			RETURNING id::text, sort_order`,
			clip(name, 60), req.Sort).Scan(&id, &sort); err != nil {
			if isUniqueViolation(err) {
				s.respondErr(w, httpx.NewError(http.StatusConflict,
					"duplicate_name", "errors.duplicate_name"))
				return
			}
			s.respondErr(w, err)
			return
		}
		s.audit(r, "finance.expense_category", "expense_category", id,
			map[string]any{"op": "create", "name": name, "active": true, "sort_order": sort})
		httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
		return
	}
	if !isUUID(*req.ID) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	// **ولا يُحذف بابٌ صُرف عليه** — يُطفأ فلا يُختار جديداً، **وحذفُه يمحو
	// تبويبَ ما مضى** فيُقرأ تاريخُ الإنفاق ناقصاً.
	//
	// **والسجلُّ يحفظ ما كان وما صار** — وكان يحفظ الاسمَ المرسَل وحدَه، فإطفاءُ
	// بابٍ يُكتب باسمٍ فارغٍ ولا يُعرف أنّه أُطفئ.
	var before, after struct {
		Name   string
		Active bool
		Sort   int
	}
	err = s.pg.QueryRow(r.Context(), `
		WITH old AS (SELECT name, active, sort_order FROM expense_categories WHERE id = $1)
		UPDATE expense_categories c
		SET name = COALESCE(NULLIF($2, ''), c.name),
		    active = COALESCE($3, c.active),
		    sort_order = COALESCE($4, c.sort_order)
		FROM old
		WHERE c.id = $1
		RETURNING old.name, old.active, old.sort_order, c.name, c.active, c.sort_order`,
		*req.ID, clip(name, 60), req.Active, req.Sort).
		Scan(&before.Name, &before.Active, &before.Sort, &after.Name, &after.Active, &after.Sort)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if err != nil {
		if isUniqueViolation(err) {
			s.respondErr(w, httpx.NewError(http.StatusConflict,
				"duplicate_name", "errors.duplicate_name"))
			return
		}
		s.respondErr(w, err)
		return
	}
	s.audit(r, "finance.expense_category", "expense_category", *req.ID,
		map[string]any{"op": "update",
			"name": after.Name, "active": after.Active, "sort_order": after.Sort,
			"old_name": before.Name, "old_active": before.Active, "old_sort_order": before.Sort})
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

// expenseRow **سطرٌ في جدول المصروفات** — مصروفٌ مقيَّد أو ملغى أو اقتراحٌ ينتظر أو مرفوض.
type expenseRow struct {
	ID       string    `json:"id"`
	Source   string    `json:"source"` // expense | request
	Status   string    `json:"status"` // posted | voided | pending | rejected
	Category string    `json:"category"`
	Amount   int64     `json:"amount"`
	Note     string    `json:"note"`
	SpentAt  time.Time `json:"spent_at"`
	By       string    `json:"by"`
	ByID     string    `json:"by_id"`
	// Decider من ألغى أو وافق أو رفض — بحسب الحالة.
	Decider      string     `json:"decider"`
	DecidedAt    *time.Time `json:"decided_at"`
	Reason       string     `json:"reason"`
	SelfApproved bool       `json:"self_approved"`
	ReceiptURL   *string    `json:"receipt_url"`
	CreatedAt    time.Time  `json:"created_at"`
}

// expenseRowsCTE **المصروفاتُ والاقتراحاتُ في جدولٍ واحد** — والاقتراحُ الموافَقُ
// عليه صار مصروفاً فلا يُعرض مرّتين.
const expenseRowsCTE = `
	WITH x AS (
		SELECT e.id, 'expense'::text AS src,
		       CASE WHEN e.voided_at IS NULL THEN 'posted' ELSE 'voided' END AS status,
		       e.category_id, e.amount, e.note, e.spent_at, e.created_by AS by_id,
		       CASE WHEN e.voided_at IS NULL THEN e.approved_by ELSE e.voided_by END AS decider_id,
		       e.voided_at AS decided_at, e.void_reason AS reason,
		       (CASE WHEN e.voided_at IS NULL THEN e.self_approved ELSE e.void_self_approved END) AS self_ok,
		       e.receipt_media_id, e.created_at
		FROM expenses e
		UNION ALL
		SELECT q.id, 'request', q.status, q.category_id, q.amount, q.note, q.spent_at,
		       q.proposed_by, q.decided_by, q.decided_at, q.decision_note, q.self_approved,
		       q.receipt_media_id, q.created_at
		FROM expense_requests q WHERE q.status <> 'approved'
	)`

// expenseScope **الشرطُ واحدٌ للعدّ وللقائمة وللتصدير** — والحالةُ `$4` فارغةٌ للكلّ.
const expenseScope = `
	FROM x
	JOIN expense_categories c ON c.id = x.category_id
	LEFT JOIN users u ON u.id = x.by_id
	LEFT JOIN users d ON d.id = x.decider_id
	LEFT JOIN media md ON md.id = x.receipt_media_id
	WHERE ($1 = '' OR x.spent_at >= NULLIF($1, '')::date)
	  AND ($2 = '' OR x.spent_at <= NULLIF($2, '')::date)
	  AND ($3 = '' OR x.category_id::text = $3)
	  AND ($4 = '' OR x.status = $4)
	  AND ($5 = '' OR x.note ILIKE '%' || $5 || '%' OR c.name ILIKE '%' || $5 || '%'
	       OR COALESCE(u.full_name, '') ILIKE '%' || $5 || '%')`

const expenseCols = `
	SELECT x.id::text, x.src, x.status, c.name, x.amount, x.note, x.spent_at,
	       COALESCE(NULLIF(u.full_name, ''), u.phone::text, ''), COALESCE(x.by_id::text, ''),
	       COALESCE(NULLIF(d.full_name, ''), d.phone::text, ''), x.decided_at, x.reason,
	       x.self_ok, COALESCE(md.path, ''), x.created_at`

func scanExpenseRow(row pgx.Row) (expenseRow, error) {
	var x expenseRow
	var receipt string
	err := row.Scan(&x.ID, &x.Source, &x.Status, &x.Category, &x.Amount, &x.Note, &x.SpentAt,
		&x.By, &x.ByID, &x.Decider, &x.DecidedAt, &x.Reason, &x.SelfApproved, &receipt, &x.CreatedAt)
	if receipt != "" {
		u := media.SignedURL(receipt)
		x.ReceiptURL = &u
	}
	return x, err
}

// expenseFilters **مرشّحاتُ القائمة** — المدى بتاريخ الصرف والبابُ والحالةُ والبحث.
type expenseFilters struct {
	From, To, Category, Status, Q string
}

func readExpenseFilters(r *http.Request) (expenseFilters, error) {
	q := r.URL.Query()
	f := expenseFilters{
		From: q.Get("from"), To: q.Get("to"), Category: q.Get("category_id"),
		Status: q.Get("status"), Q: strings.TrimSpace(q.Get("q")),
	}
	if f.Status == "all" {
		f.Status = ""
	}
	for _, d := range []string{f.From, f.To} {
		if d == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", d); err != nil {
			return f, errValidation
		}
	}
	// **ومدًى مقلوبٌ يُقال** — وكان يعطي قائمةً فارغةً تُقرأ «لا مصروفات».
	if f.From != "" && f.To != "" && f.From > f.To {
		return f, errExpenseBadRange
	}
	if f.Category != "" && !isUUID(f.Category) {
		return f, errValidation
	}
	switch f.Status {
	case "", "posted", "voided", "pending", "rejected":
	default:
		return f, errValidation
	}
	f.Q = clip(f.Q, 80)
	return f, nil
}

func (f expenseFilters) args() []any {
	return []any{f.From, f.To, f.Category, f.Status, f.Q}
}

// monthBounds **أوّلُ الشهر وآخرُه** — للشهر الذي فيه اليوم `day`.
func monthBounds(day time.Time) (string, string) {
	first := time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1)
	return first.Format("2006-01-02"), last.Format("2006-01-02")
}

// handleListExpenses **ما أُنفق — بمداه ومجموعه وتوزيعه على الأبواب وبطاقاتِ شهره.**
func (s *Server) handleListExpenses(w http.ResponseWriter, r *http.Request) {
	f, err := readExpenseFilters(r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	pg := pagingOf(r, 25)
	ctx := r.Context()

	var count int
	if err := s.pg.QueryRow(ctx, expenseRowsCTE+` SELECT count(*)`+expenseScope, f.args()...).
		Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}

	// **والمجموعُ للمقيَّد وحدَه** — **والملغى يبقى ظاهراً ولا يُعدّ**، والمنتظرُ
	// يُقال وحدَه: مالٌ لم يخرج بعد. والحالةُ المختارةُ لا تمسّ المجموع.
	sumArgs := []any{f.From, f.To, f.Category, "", f.Q}
	var total, pendingAmount int64
	var pendingCount int
	if err := s.pg.QueryRow(ctx, expenseRowsCTE+`
		SELECT COALESCE(sum(x.amount) FILTER (WHERE x.status = 'posted'), 0),
		       COALESCE(sum(x.amount) FILTER (WHERE x.status = 'pending'), 0),
		       count(*) FILTER (WHERE x.status = 'pending')`+expenseScope, sumArgs...).
		Scan(&total, &pendingAmount, &pendingCount); err != nil {
		s.respondErr(w, err)
		return
	}

	// **والتوزيعُ على الأبواب هو السؤال** — «كم على الرواتب وكم على الإيجار».
	// **وخطؤه يُردّ ولا يُبلَع** — وكان يُبلَع فيظهر المجموعُ وحدَه بلا تفصيلٍ ولا رسالة.
	byCat := []map[string]any{}
	cRows, err := s.pg.Query(ctx, expenseRowsCTE+`
		SELECT c.id::text, c.name, count(*), COALESCE(sum(x.amount), 0)`+expenseScope+`
		  AND x.status = 'posted'
		GROUP BY c.id, c.name ORDER BY sum(x.amount) DESC`, sumArgs...)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for cRows.Next() {
		var id, name string
		var n int
		var sum int64
		if err := cRows.Scan(&id, &name, &n, &sum); err != nil {
			cRows.Close()
			s.respondErr(w, err)
			return
		}
		byCat = append(byCat, map[string]any{"id": id, "name": name, "count": n, "total": sum})
	}
	cRows.Close()
	if err := cRows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}

	args := append(f.args(), pg.PerPage, pg.Offset)
	rows, err := s.pg.Query(ctx, expenseRowsCTE+expenseCols+expenseScope+`
		ORDER BY x.spent_at DESC, x.created_at DESC
		LIMIT $6 OFFSET $7`, args...)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []expenseRow{}
	for rows.Next() {
		x, err := scanExpenseRow(rows)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}

	// ── بطاقاتُ الشهر: مجموعُه · الفرقُ عن الذي قبله · أكبرُ باب ─────────
	//
	// **والشهرُ شهرُ نهاية المدى** (أو شهرُ اليوم بيوم دمشق)، وبتاريخ الصرف.
	ref := time.Now().In(damascusLoc())
	if t, err := time.Parse("2006-01-02", f.To); err == nil {
		ref = t
	} else if t, err := time.Parse("2006-01-02", f.From); err == nil {
		ref = t
	}
	mFrom, mTo := monthBounds(ref)
	pFrom, pTo := monthBounds(time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0))
	monthTotal, err := opexBySpendDate(ctx, s.pg, mFrom, mTo)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	prevTotal, err := opexBySpendDate(ctx, s.pg, pFrom, pTo)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	var topName string
	var topTotal int64
	if err := s.pg.QueryRow(ctx, `
		SELECT c.name, sum(e.amount) FROM expenses e
		JOIN expense_categories c ON c.id = e.category_id
		WHERE e.voided_at IS NULL AND e.spent_at BETWEEN $1::date AND $2::date
		GROUP BY c.name ORDER BY sum(e.amount) DESC, c.name LIMIT 1`, mFrom, mTo).
		Scan(&topName, &topTotal); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, err)
		return
	}
	var balance int64
	if t := s.orders.TreasuryID(ctx); t != "" {
		if b, err := s.wallet.Balance(ctx, t); err == nil {
			balance = b
		}
	}

	res := paged("expenses", out, count, pg)
	// **والمبلغُ اسمُه غيرُ اسم العدد** — **و`total` تحمل معنيين تجعل
	// الترقيمَ يقرأ المبلغَ عددَ صفوف.**
	res["total_amount"] = total
	res["pending_amount"] = pendingAmount
	res["pending_count"] = pendingCount
	res["by_category"] = byCat
	res["summary"] = map[string]any{
		"month_from": mFrom, "month_to": mTo,
		"month_total": monthTotal, "prev_month_total": prevTotal,
		"top_category": topName, "top_category_total": topTotal,
	}
	res["threshold"] = s.expenseApprovalThreshold(ctx)
	res["treasury_balance"] = balance
	res["today"] = expenseToday()
	httpx.JSON(w, http.StatusOK, res)
}

// handleExportExpenses **تصديرُ الجدول بمرشّحاته نفسِها** — CSV يُفتح في Excel.
func (s *Server) handleExportExpenses(w http.ResponseWriter, r *http.Request) {
	f, err := readExpenseFilters(r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(r.Context(), expenseRowsCTE+expenseCols+expenseScope+`
		ORDER BY x.spent_at DESC, x.created_at DESC LIMIT 20000`, f.args()...)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	list := []expenseRow{}
	for rows.Next() {
		x, err := scanExpenseRow(rows)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		list = append(list, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	statusAr := map[string]string{
		"posted": "مصروف", "voided": "ملغى", "pending": "بانتظار الموافقة", "rejected": "مرفوض",
	}
	name := "expenses-" + f.From + "_" + f.To + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"تاريخ الصرف", "الباب", "المبلغ (ل.س)", "الملاحظة", "سجّله", "الحالة", "السبب"})
	for _, x := range list {
		_ = cw.Write([]string{
			x.SpentAt.Format("2006-01-02"), csvSafe(x.Category), strconv.FormatInt(x.Amount, 10),
			csvSafe(x.Note), csvSafe(x.By), statusAr[x.Status], csvSafe(x.Reason),
		})
	}
	cw.Flush()
}

// handleUploadExpenseReceipt **صورةُ الإيصال** — صنفٌ محميٌّ يُقرأ برابطٍ موقَّع.
//
// **ونقطةٌ لأهل الماليّة** — رفعُ الوسائط العامُّ للمحتوى (`content.manage`).
func (s *Server) handleUploadExpenseReceipt(w http.ResponseWriter, r *http.Request) {
	lim := s.media.MaxBytes(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, lim+64<<10)
	if err := r.ParseMultipartForm(lim); err != nil {
		s.respondErr(w, media.ErrTooLarge)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	defer file.Close()
	m, err := s.media.Save(r.Context(), userIDFrom(r), "expense_receipt", file)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والمعاينةُ برابطٍ موقَّع** — العاري يُحجب لأنّ الصنفَ محميّ.
	m.URL = media.SignedURL(strings.TrimPrefix(m.URL, "/media/"))
	m.ThumbURL = media.SignedURL(strings.TrimPrefix(m.ThumbURL, "/media/"))
	httpx.JSON(w, http.StatusCreated, m)
}

// handleCreateExpense **يُسجَّل المصروفُ ويخرج من الخزينة معاً — أو يصير اقتراحاً.**
//
// **تحت السقف أو عنده**: صفٌّ وقيدٌ في معاملةٍ واحدة (`201`، `status=posted`).
// **وفوقه**: اقتراحٌ في `expense_requests` بإيصالٍ إلزاميّ، لا يمسّ الخزينة حتّى
// يوافق عليه شخصٌ آخر (`201`، `status=pending`).
func (s *Server) handleCreateExpense(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		CategoryID     string `json:"category_id"`
		Amount         int64  `json:"amount"`
		Note           string `json:"note"`
		SpentAt        string `json:"spent_at"`
		ReceiptMediaID string `json:"receipt_media_id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if !isUUID(req.CategoryID) || req.Amount <= 0 {
		s.respondErr(w, errValidation)
		return
	}
	spent := strings.TrimSpace(req.SpentAt)
	today := expenseToday()
	if spent == "" {
		spent = today
	}
	if _, err := time.Parse("2006-01-02", spent); err != nil {
		s.respondErr(w, errValidation)
		return
	}
	// **ولا صرفَ في الغد** — تاريخٌ في المستقبل يُدخل المالَ شهراً لم يأتِ.
	if spent > today {
		s.respondErr(w, errExpenseFutureDate)
		return
	}
	var receipt *string
	if id := strings.TrimSpace(req.ReceiptMediaID); id != "" {
		if !isUUID(id) {
			s.respondErr(w, errValidation)
			return
		}
		var kind string
		if err := s.pg.QueryRow(r.Context(), `SELECT kind FROM media WHERE id = $1`, id).
			Scan(&kind); err != nil || kind != "expense_receipt" {
			s.respondErr(w, errValidation)
			return
		}
		receipt = &id
	}
	// **ولا يُصرف على بابٍ مُطفأ** — **وحكمٌ في الشاشة وحدَها وعدٌ بحكم.**
	var active bool
	var catName string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT active, name FROM expense_categories WHERE id = $1`, req.CategoryID).
		Scan(&active, &catName); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if !active {
		s.respondErr(w, errExpenseCategoryOff)
		return
	}
	note := clip(strings.TrimSpace(req.Note), 300)
	threshold := s.expenseApprovalThreshold(r.Context())
	needsApproval := req.Amount > threshold
	if needsApproval && receipt == nil {
		s.respondErr(w, errExpenseReceiptRequired)
		return
	}

	actor := userIDFrom(r)

	// ══════════════════════════════════════════════════════════════
	// **المصروفُ وخصمُ الخزينة كتابةٌ واحدة** — `PF-02` · `D5`
	// ══════════════════════════════════════════════════════════════
	//
	// **و`ApplyTx` تكتب في المُمرَّرة**، فتقع الكتابتان أو لا تقع واحدة.
	tx, err := s.pg.Begin(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	if needsApproval {
		// **فوق السقف: اقتراحٌ لا قيد** — يوافق عليه شخصٌ آخر (قرارُ المالك ١).
		var id string
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO expense_requests (category_id, amount, note, spent_at, receipt_media_id, proposed_by)
			VALUES ($1, $2, $3, $4::date, $5, $6) RETURNING id::text`,
			req.CategoryID, req.Amount, note, spent, receipt, actor).Scan(&id); err != nil {
			s.respondErr(w, err)
			return
		}
		if err := s.auditTx(r.Context(), tx, r, "finance.expense_request", "expense_request", id,
			map[string]any{"amount": req.Amount, "spent_at": spent, "category": catName,
				"note": note, "receipt": receipt != nil, "threshold": threshold}); err != nil {
			s.respondErr(w, err)
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			s.respondErr(w, err)
			return
		}
		s.touch("wallet", "ops")
		httpx.JSON(w, http.StatusCreated, map[string]any{"id": id, "status": "pending"})
		return
	}

	var id string
	if err := tx.QueryRow(r.Context(), `
		INSERT INTO expenses (category_id, amount, note, spent_at, created_by, receipt_media_id)
		VALUES ($1, $2, $3, $4::date, $5, $6) RETURNING id::text`,
		req.CategoryID, req.Amount, note, spent, actor, receipt).Scan(&id); err != nil {
		s.respondErr(w, err)
		return
	}

	// **والخزينةُ تنقص** — والمرجعُ معرّفُ المصروف، **والنصُّ يقول الباب.**
	if _, err := s.wallet.ApplyTx(r.Context(), tx, s.orders.TreasuryID(r.Context()),
		-req.Amount, "operating_expense", id, treasuryExpenseNote("مصروف تشغيل", catName, note),
		&actor); err != nil {
		s.respondErr(w, err)
		return
	}
	// **والأثرُ في المعاملة نفسِها** — `PF-06`.
	if err := s.auditTx(r.Context(), tx, r, "finance.expense_added", "expense", id,
		map[string]any{"amount": req.Amount, "spent_at": spent, "category": catName,
			"note": note, "receipt": receipt != nil}); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id, "status": "posted"})
}

// handleListExpenseRequests **الاقتراحاتُ** — المنتظرةُ افتراضاً؛ لصفحة الموافقات الموحّدة.
func (s *Server) handleListExpenseRequests(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	if status == "all" {
		status = ""
	}
	switch status {
	case "", "pending", "approved", "rejected":
	default:
		s.respondErr(w, errValidation)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT q.id::text, q.status, c.name, q.amount, q.note, q.spent_at,
		       q.proposed_by::text, COALESCE(NULLIF(p.full_name, ''), p.phone::text, ''),
		       q.decided_by::text, q.decided_at, q.decision_note, q.self_approved,
		       COALESCE(md.path, ''), q.created_at
		FROM expense_requests q
		JOIN expense_categories c ON c.id = q.category_id
		LEFT JOIN users p ON p.id = q.proposed_by
		LEFT JOIN media md ON md.id = q.receipt_media_id
		WHERE ($1 = '' OR q.status = $1)
		ORDER BY q.created_at DESC LIMIT 200`, status)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type reqRow struct {
		ID           string     `json:"id"`
		Status       string     `json:"status"`
		Category     string     `json:"category"`
		Amount       int64      `json:"amount"`
		Note         string     `json:"note"`
		SpentAt      time.Time  `json:"spent_at"`
		ProposedBy   string     `json:"proposed_by"`
		ProposerName string     `json:"proposer_name"`
		DecidedBy    *string    `json:"decided_by"`
		DecidedAt    *time.Time `json:"decided_at"`
		DecisionNote string     `json:"decision_note"`
		SelfApproved bool       `json:"self_approved"`
		ReceiptURL   *string    `json:"receipt_url"`
		CreatedAt    time.Time  `json:"created_at"`
	}
	out := []reqRow{}
	for rows.Next() {
		var x reqRow
		var receipt string
		if err := rows.Scan(&x.ID, &x.Status, &x.Category, &x.Amount, &x.Note, &x.SpentAt,
			&x.ProposedBy, &x.ProposerName, &x.DecidedBy, &x.DecidedAt, &x.DecisionNote,
			&x.SelfApproved, &receipt, &x.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		if receipt != "" {
			u := media.SignedURL(receipt)
			x.ReceiptURL = &u
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"requests": out,
		"threshold": s.expenseApprovalThreshold(r.Context())})
}

// handleApproveExpenseRequest **يوافق شخصٌ آخر فيُقيَّد المصروفُ ويخرج المال.**
//
// **وصاحبُ الاقتراح لا يوافق عليه** — إلّا المالكُ الأعلى إن لم يكن غيرُه،
// **ويُعلَّم ذلك في الصفّ والسجلّ** (`approval.Check`).
func (s *Server) handleApproveExpenseRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Note string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	actor := userIDFrom(r)
	ctx := r.Context()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status, categoryID, catName, note, proposedBy string
	var amount int64
	var spent time.Time
	var receipt *string
	err = tx.QueryRow(ctx, `
		SELECT q.status, q.category_id::text, c.name, q.amount, q.note, q.spent_at,
		       q.proposed_by::text, q.receipt_media_id::text
		FROM expense_requests q JOIN expense_categories c ON c.id = q.category_id
		WHERE q.id = $1 FOR UPDATE OF q`, id).
		Scan(&status, &categoryID, &catName, &amount, &note, &spent, &proposedBy, &receipt)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if status != "pending" {
		s.respondErr(w, errRequestDecided)
		return
	}
	verdict, err := approval.Check(ctx, tx, approval.Request{
		ProposedBy: proposedBy, Actor: actor, Capability: authz.FinanceManage,
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	var expenseID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO expenses (category_id, amount, note, spent_at, created_by, receipt_media_id,
		                      approved_by, self_approved)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id::text`,
		categoryID, amount, note, spent, proposedBy, receipt, actor, verdict.SelfApproved).
		Scan(&expenseID); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.wallet.ApplyTx(ctx, tx, s.orders.TreasuryID(ctx),
		-amount, "operating_expense", expenseID, treasuryExpenseNote("مصروف تشغيل", catName, note),
		&actor); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := tx.Exec(ctx, `
		UPDATE expense_requests
		   SET status = 'approved', decided_by = $2, decided_at = now(),
		       decision_note = $3, self_approved = $4, expense_id = $5
		 WHERE id = $1`, id, actor, clip(strings.TrimSpace(req.Note), 300),
		verdict.SelfApproved, expenseID); err != nil {
		s.respondErr(w, err)
		return
	}
	details := map[string]any{"request_id": id, "expense_id": expenseID, "amount": amount,
		"category": catName, "spent_at": spent.Format("2006-01-02"), "proposed_by": proposedBy,
		"note": strings.TrimSpace(req.Note)}
	for k, v := range verdict.AuditFields() {
		details[k] = v
	}
	if err := s.auditTx(ctx, tx, r, "finance.expense_request_approved", "expense", expenseID,
		details); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"status": "approved",
		"expense_id": expenseID, "self_approved": verdict.SelfApproved})
}

// handleRejectExpenseRequest **يُرفض الاقتراح** — لا مالَ يتحرّك، ويبقى ظاهراً مرفوضاً.
func (s *Server) handleRejectExpenseRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Note string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	actor := userIDFrom(r)
	note := clip(strings.TrimSpace(req.Note), 300)
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		var amount int64
		var status string
		err := q.QueryRow(ctx, `SELECT status, amount FROM expense_requests WHERE id = $1 FOR UPDATE`, id).
			Scan(&status, &amount)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if status != "pending" {
			return errRequestDecided
		}
		if _, err := q.Exec(ctx, `
			UPDATE expense_requests SET status = 'rejected', decided_by = $2, decided_at = now(),
			       decision_note = $3 WHERE id = $1`, id, actor, note); err != nil {
			return err
		}
		return s.auditTx(ctx, q, r, "finance.expense_request_rejected", "expense_request", id,
			map[string]any{"amount": amount, "note": note})
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"status": "rejected"})
}

// handleVoidExpense **الخطأُ يُلغى ولا يُمحى — بسببٍ ومن شخصٍ آخر.**
//
// **وقاعدةُ الدفتر عندنا**: لا يُعدَّل بل يُصحَّح بقيدٍ مضادّ. **فحذفُ الصفّ
// يترك قيدَه في الخزينة بلا صاحب** — مالٌ خرج ولا يُعرف لماذا.
func (s *Server) handleVoidExpense(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	reason := clip(strings.TrimSpace(req.Reason), 300)
	if reason == "" {
		s.respondErr(w, errExpenseVoidReason)
		return
	}
	actor := userIDFrom(r)

	// **والإلغاءُ كالإنشاء** — `PF-02`: **وسمُ الإلغاءِ وردُّ المال
	// كتابةٌ واحدة.**
	tx, err := s.pg.Begin(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var amount int64
	var createdBy, catName string
	// **ولا يُلغى مرّتين** — **وإلّا رُدّ المالُ ضِعفَه إلى الخزينة.**
	if err := tx.QueryRow(r.Context(), `
		SELECT e.amount, COALESCE(e.created_by::text, ''), c.name
		FROM expenses e JOIN expense_categories c ON c.id = e.category_id
		WHERE e.id = $1 AND e.voided_at IS NULL FOR UPDATE OF e`, id).
		Scan(&amount, &createdBy, &catName); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	// **ومن سجّل لا يلغي** (قرارُ المالك ٥) — إلّا المالكُ وحدَه وبلا بديل، مُعلَّماً.
	verdict, err := approval.Check(r.Context(), tx, approval.Request{
		ProposedBy: createdBy, Actor: actor, Capability: authz.FinanceManage,
	})
	if errors.Is(err, approval.ErrSelfApprove) {
		s.respondErr(w, errExpenseVoidSelf)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := tx.Exec(r.Context(), `
		UPDATE expenses SET voided_at = now(), voided_by = $2, void_reason = $3,
		       void_self_approved = $4
		WHERE id = $1`, id, actor, reason, verdict.SelfApproved); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.wallet.ApplyTx(r.Context(), tx, s.orders.TreasuryID(r.Context()),
		amount, "operating_expense", id, treasuryExpenseNote("إلغاء مصروف تشغيل", catName, reason),
		&actor); err != nil {
		s.respondErr(w, err)
		return
	}
	details := map[string]any{"amount": amount, "category": catName, "reason": reason,
		"recorded_by": createdBy}
	for k, v := range verdict.AuditFields() {
		details[k] = v
	}
	// **والأثرُ في المعاملة نفسِها** — `PF-06`.
	if err := s.auditTx(r.Context(), tx, r, "finance.expense_voided", "expense", id,
		details); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"voided": true})
}
