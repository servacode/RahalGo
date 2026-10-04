package server

// ══════════════════════════════════════════════════════════════════════
// **الموافقاتُ الموحّدة** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (الخزينة)
// ══════════════════════════════════════════════════════════════════════
//
// **كلُّ صرفٍ يقترحه شخصٌ ويوافق غيرُه** — وكلُّ قسمٍ يحفظ اقتراحاتِه في جدوله
// (عقدُ لوحة التنسيق: `id · status · amount · note · proposed_by · decided_by ·
// created_at · decided_at`). **وهذه الصفحةُ تجمعها في مكانٍ واحد**: تقرأ كلَّ جدولٍ
// مسجَّلٍ هنا موجودٍ في القاعدة، **وتنادي بابَ القسم نفسَه** للموافقة والرفض —
// فالحكمُ (المقترحُ غيرُ الموافق، والقيدُ بطرفين) يبقى في القسم لا يُكتب ثانيةً.
//
// **ولإضافة قسم**: سطرٌ واحدٌ في `approvalSources`. والجدولُ الغائبُ أو الناقصُ
// عموداً يُتخطّى بلا خطأ (`to_regclass` + `information_schema`).

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/approval"
	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// approvalSource **جدولُ اقتراحاتِ قسمٍ واحد.**
type approvalSource struct {
	// Key معرّفٌ ثابت — يُقرأ في الشاشة.
	Key string
	// Table اسمُ الجدول.
	Table string
	// Section مفتاحُ اسم القسم في المعجم: `admin.treasury.sections.<Section>`.
	Section string
	// Capability **القدرةُ التي تملك الموافقة** — بها يُحسب «أيستطيع هذا الموظّف».
	Capability authz.Capability
	// ApprovePath · RejectPath **بابُ القسم نفسُه** (`{id}` يُستبدل). وفارغُهما يعني
	// أنّ القرارَ يُتّخذ في صفحة القسم (`Href`) لا هنا.
	ApprovePath, RejectPath string
	// Href صفحةُ القسم.
	Href string
	// AmountCol · NoteCol · ProposerCol · DueCol — أعمدةٌ تخالف العقد إن وُجدت.
	AmountCol, NoteCol, ProposerCol, DueCol string
}

// approvalSources **السجلّ** — سطرٌ لكلّ قسم.
var approvalSources = []approvalSource{
	{Key: "wallet_requests", Table: "wallet_requests", Section: "walletRequests",
		Capability:  authz.FinanceManage,
		ApprovePath: "/api/v1/admin/wallet-requests/{id}/approve",
		RejectPath:  "/api/v1/admin/wallet-requests/{id}/reject",
		Href:        "/dashboard/treasury?tab=approvals"},
	{Key: "cash_closes", Table: "office_cash_closes", Section: "cashCloses",
		Capability:  authz.FinanceManage,
		ApprovePath: "/api/v1/admin/cashbox/closes/{id}/approve",
		RejectPath:  "/api/v1/admin/cashbox/closes/{id}/reject",
		Href:        "/dashboard/treasury?tab=cashbox"},
	{Key: "cash_shortfalls", Table: "office_cash_shortfalls", Section: "cashShortfalls",
		Capability:  authz.TreasuryManage,
		ApprovePath: "/api/v1/admin/cashbox/shortfalls/{id}/approve",
		RejectPath:  "/api/v1/admin/cashbox/shortfalls/{id}/reject",
		Href:        "/dashboard/treasury?tab=cashbox", DueCol: "eligible_at"},
	// **التعويضاتُ كلُّها** (سائق · بضاعة متجر · شكوى — طابورٌ واحد، هجرة 0360) —
	// يقترحها النظامُ أو موظّف، **ويقرّر المبلغَ الموظّفُ في صفحتها** (الموافقةُ تحمل
	// المبلغ)، فهي رابطٌ لا زرّ. والمبلغُ المقترحُ `suggested_amount` ما دام معلّقاً،
	// و`proposed_by` الفارغُ هو النظام.
	{Key: "driver_compensations", Table: "driver_compensation_requests", Section: "compensations",
		Capability: authz.FinanceManage, Href: "/dashboard/compensations",
		AmountCol: "suggested_amount"},
	// **مصروفاتُ التشغيل فوق السقف** — موافقةٌ ثانيةٌ بكلمة السرّ.
	{Key: "expenses", Table: "expense_requests", Section: "expenses",
		Capability:  authz.FinanceManage,
		ApprovePath: "/api/v1/admin/expense-requests/{id}/approve",
		RejectPath:  "/api/v1/admin/expense-requests/{id}/reject",
		Href:        "/dashboard/expenses"},
	// **الديون: دفعٌ نقداً بالمكتب أو شطب** — والشطبُ يحتاج فوقها
	// `finance.writeoff.approve` (يردّه البابُ نفسُه `403 writeoff_owner_only`).
	{Key: "obligations", Table: "obligation_requests", Section: "obligations",
		Capability:  authz.FinanceManage,
		ApprovePath: "/api/v1/admin/obligation-requests/{id}/approve",
		RejectPath:  "/api/v1/admin/obligation-requests/{id}/reject",
		Href:        "/dashboard/obligations"},
	// **المكافآتُ والعقوباتُ اليدويّة** — اقتراحٌ يوافق عليه موظّفٌ آخر (هجرة 0380).
	{Key: "incentives", Table: "incentive_requests", Section: "incentives",
		Capability:  authz.FinanceManage,
		ApprovePath: "/api/v1/admin/incentive-requests/{id}/approve",
		RejectPath:  "/api/v1/admin/incentive-requests/{id}/reject",
		Href:        "/dashboard/incentives"},
	// **حسمُ النزاعات: خصمٌ أو إسقاط** — اقتراحٌ من الماليّة وموافقةُ غيرِ المقترِح (هجرة 0350).
	{Key: "disputes", Table: "dispute_resolutions", Section: "disputes",
		Capability:  authz.FinanceManage,
		ApprovePath: "/api/v1/admin/dispute-resolutions/{id}/approve",
		RejectPath:  "/api/v1/admin/dispute-resolutions/{id}/reject",
		Href:        "/dashboard/losses"},
}

var approvalIdent = regexp.MustCompile(`^[a-z_]+$`)

type approvalItem struct {
	Key          string     `json:"key"`
	Section      string     `json:"section"`
	ID           string     `json:"id"`
	Amount       int64      `json:"amount"`
	Note         string     `json:"note"`
	ProposedBy   string     `json:"proposed_by"`
	ProposerName string     `json:"proposer_name"`
	CreatedAt    time.Time  `json:"created_at"`
	DueAt        *time.Time `json:"due_at"`
	ApprovePath  string     `json:"approve_path"`
	RejectPath   string     `json:"reject_path"`
	Href         string     `json:"href"`
	// CanApprove **أيستطيع هذا الموظّفُ أن يوافق؟** — قدرةٌ + ليس صاحبَ الاقتراح.
	CanApprove bool `json:"can_approve"`
	// SelfApproval **سيوافق على اقتراحه** — المالكُ وحدَه حين لا يوجد غيرُه، ويُعلَّم.
	SelfApproval bool `json:"self_approval"`
}

func col(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// pendingApprovals **كلُّ ما ينتظر قراراً** — من كلّ جدولٍ مسجَّلٍ موجود.
func (s *Server) pendingApprovals(ctx context.Context, q dbtx.Querier, r *http.Request) ([]approvalItem, error) {
	me := ""
	if r != nil {
		me = userIDFrom(r)
	}
	out := []approvalItem{}
	ownerAlone := map[authz.Capability]bool{}
	for _, src := range approvalSources {
		amountC := col(src.AmountCol, "amount")
		noteC := col(src.NoteCol, "note")
		propC := col(src.ProposerCol, "proposed_by")
		need := []string{"id", "status", "created_at", amountC, noteC}
		if propC != "-" {
			need = append(need, propC)
		}
		if src.DueCol != "" {
			need = append(need, src.DueCol)
		}
		ok, err := tableHasColumns(ctx, q, src.Table, need)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		propExpr, joinUser := "''", ""
		if propC != "-" {
			propExpr = "COALESCE(x." + propC + "::text, '')"
			joinUser = " LEFT JOIN users p ON p.id::text = x." + propC + "::text"
		}
		nameExpr := "''"
		if joinUser != "" {
			nameExpr = "COALESCE(NULLIF(p.full_name, ''), p.phone::text, '')"
		}
		dueExpr := "NULL::timestamptz"
		if src.DueCol != "" {
			dueExpr = "x." + src.DueCol
		}
		rows, err := q.Query(ctx, `
			SELECT x.id::text, COALESCE(x.`+amountC+`, 0)::bigint, COALESCE(x.`+noteC+`::text, ''),
			       `+propExpr+`, `+nameExpr+`, x.created_at, `+dueExpr+`
			  FROM `+src.Table+` x`+joinUser+`
			 WHERE x.status = 'pending'
			 ORDER BY x.created_at
			 LIMIT 200`)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			it := approvalItem{Key: src.Key, Section: src.Section, Href: src.Href}
			if err := rows.Scan(&it.ID, &it.Amount, &it.Note, &it.ProposedBy, &it.ProposerName,
				&it.CreatedAt, &it.DueAt); err != nil {
				rows.Close()
				return nil, err
			}
			if src.ApprovePath != "" {
				it.ApprovePath = strings.ReplaceAll(src.ApprovePath, "{id}", it.ID)
				it.RejectPath = strings.ReplaceAll(src.RejectPath, "{id}", it.ID)
			}
			out = append(out, it)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		// **أيستطيع؟** — يُحسب بعد إغلاق القراءة (لا نداءَ داخل نداءٍ مفتوح).
		for i := range out {
			it := &out[i]
			if it.Key != src.Key || it.ApprovePath == "" || r == nil {
				continue
			}
			if !s.hasCapability(r, src.Capability) {
				continue
			}
			if it.ProposedBy == "" || it.ProposedBy != me {
				it.CanApprove = true
				continue
			}
			alone, seen := ownerAlone[src.Capability]
			if !seen {
				v, err := approval.Check(ctx, q, approval.Request{
					ProposedBy: me, Actor: me, Capability: src.Capability})
				if err != nil && !errors.Is(err, approval.ErrSelfApprove) {
					return nil, err
				}
				alone = err == nil && v.SelfApproved
				ownerAlone[src.Capability] = alone
			}
			it.CanApprove = alone
			it.SelfApproval = alone
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

// tableHasColumns **أفي القاعدة جدولٌ بهذه الأعمدة؟** — وغيابُه ليس خطأ.
func tableHasColumns(ctx context.Context, q dbtx.Querier, table string, cols []string) (bool, error) {
	if !approvalIdent.MatchString(table) {
		return false, nil
	}
	for _, c := range cols {
		if !approvalIdent.MatchString(c) {
			return false, nil
		}
	}
	var n int
	if err := q.QueryRow(ctx, `
		SELECT count(DISTINCT column_name)::int FROM information_schema.columns
		 WHERE table_schema = current_schema() AND table_name = $1
		   AND column_name = ANY ($2::text[])
		   AND to_regclass($1) IS NOT NULL`, table, cols).Scan(&n); err != nil {
		return false, err
	}
	uniq := map[string]bool{}
	for _, c := range cols {
		uniq[c] = true
	}
	return n == len(uniq), nil
}

// handleApprovals **صفحةُ الموافقات الموحّدة.**
func (s *Server) handleApprovals(w http.ResponseWriter, r *http.Request) {
	items, err := s.pendingApprovals(r.Context(), s.pg, r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	var total int64
	for _, it := range items {
		total += it.Amount
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items), "total": total})
}
