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
*/

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// handleExpenseCategories **أبوابُ المصروف — تُدار ولا تُكتب في الشيفرة.**
//
// **وقائمةٌ في الشيفرة تعني نشرةً جديدةً لإضافة باب** — ومن احتاج باباً اليومَ
// كتبه في «أخرى»، **فيضيع التبويبُ الذي بُنيت الشاشةُ لأجله.**
func (s *Server) handleExpenseCategories(w http.ResponseWriter, r *http.Request) {
	// **والمُطفأةُ تُقرأ في الإدارة** — لتُعاد، **ولتُقرأ أسماءُ ما مضى.**
	rows, err := s.pg.Query(r.Context(), `
		SELECT c.id::text, c.name, c.active, c.sort_order,
		       (SELECT count(*) FROM expenses e
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
		// Used **كم صُرف على هذا الباب** — **ومن أراد حذفَ بابٍ يعرف
		// أنّه يمحو تبويبَ خمسين قيدا.**
		Used int `json:"used"`
	}
	out := []cat{}
	for rows.Next() {
		var c cat
		if err := rows.Scan(&c.ID, &c.Name, &c.Active, &c.Sort, &c.Used); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, c)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"categories": out})
}

// handleSaveExpenseCategory **يُضاف بابٌ أو يُعدَّل اسمُه أو يُطفأ.**
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
		if err := s.pg.QueryRow(r.Context(), `
			INSERT INTO expense_categories (name, sort_order)
			VALUES ($1, COALESCE($2, 0)) RETURNING id::text`,
			name, req.Sort).Scan(&id); err != nil {
			// **واسمٌ مكرَّرٌ يُقال بنصّه** — **ورسالةٌ عامّةٌ تجعل من كرّر
			// يعيد المحاولةَ بلا فائدة.**
			if isUniqueViolation(err) {
				s.respondErr(w, httpx.NewError(http.StatusConflict,
					"duplicate_name", "errors.duplicate_name"))
				return
			}
			s.respondErr(w, err)
			return
		}
		s.audit(r, "finance.expense_category", "expense_category", id,
			map[string]any{"name": name})
		httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
		return
	}
	// **ولا يُحذف بابٌ صُرف عليه** — يُطفأ فلا يُختار جديداً، **وحذفُه يمحو
	// تبويبَ ما مضى** فيُقرأ تاريخُ الإنفاق ناقصاً.
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE expense_categories
		SET name = COALESCE(NULLIF($2, ''), name),
		    active = COALESCE($3, active),
		    sort_order = COALESCE($4, sort_order)
		WHERE id = $1`, *req.ID, name, req.Active, req.Sort)
	if err != nil {
		if isUniqueViolation(err) {
			s.respondErr(w, httpx.NewError(http.StatusConflict,
				"duplicate_name", "errors.duplicate_name"))
			return
		}
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	s.audit(r, "finance.expense_category", "expense_category", *req.ID,
		map[string]any{"name": name})
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

// handleListExpenses **ما أُنفق — بمداه وبمجموعه وتوزيعِه على الأبواب.**
func (s *Server) handleListExpenses(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	pg := pagingOf(r, 25)

	// **والشرطُ واحدٌ للعدّ وللمجموع وللقائمة** — ثلاثةُ نصوصٍ تفترق يوماً
	// **فيقول العنوانُ مليوناً ويقول المجموعُ غيرَه.**
	//
	// **والملغى يُقصى من الثلاثة** — **وقيدٌ أُلغيَ ويُعدّ في المجموع مالٌ
	// يُحسب مرّتين.**
	//
	// **ومن سجّل القيدَ يُوصل هنا لا في القائمة وحدَها** — **وأوّلُ كتابةٍ
	// تركته خارجَ الجسم المشترك فردَّ الخادمُ `missing FROM-clause entry
	// for table "u"` عند كلّ فتحةٍ للقسم.** (كشفه المالكُ على شاشته
	// ٢٠٢٦-٠٨-١٦.)
	//
	// **ووصلةٌ يساريّةٌ إلى واحدٍ لا تضاعف صفّاً** — فالعدُّ والمجموعُ لا
	// يتبدّلان بها.
	const scope = `
		FROM expenses e
		JOIN expense_categories c ON c.id = e.category_id
		LEFT JOIN users u ON u.id = e.created_by
		WHERE e.voided_at IS NULL
		  AND ($1 = '' OR e.spent_at >= $1::date)
		  AND ($2 = '' OR e.spent_at <= $2::date)`

	var count int
	var total int64
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*), COALESCE(sum(e.amount), 0)`+scope, from, to).
		Scan(&count, &total); err != nil {
		s.respondErr(w, err)
		return
	}

	// **والتوزيعُ على الأبواب هو السؤال** — «كم على الرواتب وكم على الإيجار»،
	// **ومجموعٌ واحدٌ لا يقول أين ذهب المال.**
	byCat := []map[string]any{}
	if cRows, err := s.pg.Query(r.Context(),
		`SELECT c.name, count(*), COALESCE(sum(e.amount), 0)`+scope+`
		 GROUP BY c.name ORDER BY sum(e.amount) DESC`, from, to); err == nil {
		for cRows.Next() {
			var name string
			var n int
			var sum int64
			if cRows.Scan(&name, &n, &sum) == nil {
				byCat = append(byCat, map[string]any{
					"name": name, "count": n, "total": sum})
			}
		}
		cRows.Close()
	}

	rows, err := s.pg.Query(r.Context(), `
		SELECT e.id::text, c.name, e.amount, e.note, e.spent_at,
		       COALESCE(NULLIF(u.full_name, ''), u.phone::text, '')`+scope+`
		 ORDER BY e.spent_at DESC, e.created_at DESC
		 LIMIT $3 OFFSET $4`, from, to, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type row struct {
		ID       string    `json:"id"`
		Category string    `json:"category"`
		Amount   int64     `json:"amount"`
		Note     string    `json:"note"`
		SpentAt  time.Time `json:"spent_at"`
		By       string    `json:"by"`
	}
	out := []row{}
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.ID, &x.Category, &x.Amount, &x.Note, &x.SpentAt, &x.By); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	res := paged("expenses", out, count, pg)
	// **والمبلغُ اسمُه غيرُ اسم العدد** — **و`total` تحمل معنيين تجعل
	// الترقيمَ يقرأ المبلغَ عددَ صفوف**: مليونٌ ونصف يصير أربعين ألفَ
	// صفحة. (وقع في أوّل كتابةٍ لهذا المعالِج وأُمسك قبل النشر.)
	res["total_amount"] = total
	res["by_category"] = byCat
	httpx.JSON(w, http.StatusOK, res)
}

// handleCreateExpense **يُسجَّل المصروفُ ويخرج من الخزينة معاً.**
//
// **وصفٌّ بلا قيدٍ يجعل الشاشةَ تقول ما لا تقوله الخزينة** — ورصيدٌ يخالف
// دفترَه لا يُصدَّق أيُّهما.
func (s *Server) handleCreateExpense(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		CategoryID string `json:"category_id"`
		Amount     int64  `json:"amount"`
		Note       string `json:"note"`
		SpentAt    string `json:"spent_at"`
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
	if spent == "" {
		spent = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", spent); err != nil {
		s.respondErr(w, errValidation)
		return
	}
	// **ولا يُصرف على بابٍ مُطفأ** — **وحكمٌ في الشاشة وحدَها وعدٌ بحكم.**
	var active bool
	if err := s.pg.QueryRow(r.Context(),
		`SELECT active FROM expense_categories WHERE id = $1`, req.CategoryID).
		Scan(&active); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if !active {
		s.respondErr(w, errValidation)
		return
	}

	actor := userIDFrom(r)
	var id string
	if err := s.pg.QueryRow(r.Context(), `
		INSERT INTO expenses (category_id, amount, note, spent_at, created_by)
		VALUES ($1, $2, $3, $4::date, $5) RETURNING id::text`,
		req.CategoryID, req.Amount, clip(strings.TrimSpace(req.Note), 300),
		spent, actor).Scan(&id); err != nil {
		s.respondErr(w, err)
		return
	}

	// **والخزينةُ تنقص** — والمرجعُ معرّفُ المصروف، **فمن قرأ قيداً في
	// الدفتر عرف على أيّ بابٍ صُرف.**
	if _, err := s.wallet.Apply(r.Context(), s.orders.TreasuryID(r.Context()),
		-req.Amount, "operating_expense", id, clip(strings.TrimSpace(req.Note), 300),
		&actor); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "finance.expense_added", "expense", id,
		map[string]any{"amount": req.Amount, "spent_at": spent})
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

// handleVoidExpense **الخطأُ يُلغى ولا يُمحى.**
//
// **وقاعدةُ الدفتر عندنا**: لا يُعدَّل بل يُصحَّح بقيدٍ مضادّ. **فحذفُ الصفّ
// يترك قيدَه في الخزينة بلا صاحب** — مالٌ خرج ولا يُعرف لماذا.
func (s *Server) handleVoidExpense(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	actor := userIDFrom(r)
	var amount int64
	// **ولا يُلغى مرّتين** — **وإلّا رُدّ المالُ ضِعفَه إلى الخزينة.**
	if err := s.pg.QueryRow(r.Context(), `
		UPDATE expenses SET voided_at = now(), voided_by = $2
		WHERE id = $1 AND voided_at IS NULL
		RETURNING amount`, id, actor).Scan(&amount); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if _, err := s.wallet.Apply(r.Context(), s.orders.TreasuryID(r.Context()),
		amount, "operating_expense", id, "إلغاءُ مصروف", &actor); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "finance.expense_voided", "expense", id,
		map[string]any{"amount": amount})
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"voided": true})
}

var _ = strconv.Itoa
