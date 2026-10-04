package server

// ══════════════════════════════════════════════════════════════════════
// **قسمُ الخزينة الموحّد** — قراراتُ المالك ٢٠٢٦-١٠-٠٤
// ══════════════════════════════════════════════════════════════════════
//
//	نظرةٌ عامّة       رصيدُ الخزينة · الصندوق · النقدُ في الشارع · ما ينتظر قراراً
//	كشفُ الحساب      كلُّ قيدٍ في الخزينة برصيده الجاري، وتصديرٌ CSV بيوم دمشق
//	سحبُ الأدمن       «حتّى لو دفع من المحفظة رح يكون واضح إنّ الأدمن سحب من رصيد
//	                  الخزينة» — نوعُ قيدٍ خاصّ (`treasury_withdrawal`) باسم صاحبه
//	صحّةُ الدفتر      فحوصُ `fininv` نفسُها التي يشغّلها `moneycheck`، أحمرُ وأخضر
//
// **والخزينةُ تبقى محفظةَ الأدمن** (`wallets.is_treasury`) — لا حسابٌ ثانٍ.

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/officecash"
)

var (
	errTreasuryMissing = httpx.NewError(http.StatusConflict,
		"treasury_missing", "errors.treasury_missing")
	errTreasuryOverBalance = httpx.NewError(http.StatusConflict,
		"treasury_over_balance", "errors.treasury_over_balance")
)

// treasuryWithdrawalNote **ما يُكتب في كشف الخزينة** — كما قاله المالك.
const treasuryWithdrawalNote = "سحبُ الأدمن من رصيد الخزينة"

// ── النظرةُ العامّة ───────────────────────────────────────────────────

func (s *Server) handleTreasuryOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	type overview struct {
		TreasuryBalance  int64          `json:"treasury_balance"`
		HasTreasury      bool           `json:"has_treasury"`
		Cashbox          cashPosition   `json:"cashbox"`
		TodayIn          int64          `json:"today_in"`
		TodayOut         int64          `json:"today_out"`
		CashInStreet     int64          `json:"cash_in_street"`
		MerchantCashDue  int64          `json:"merchant_cash_due"`
		PendingPayouts   int64          `json:"pending_payouts"`
		WithdrawalsMonth int64          `json:"withdrawals_month"`
		OpenShortfalls   int64          `json:"open_shortfalls"`
		PendingCount     int            `json:"pending_count"`
		PendingTotal     int64          `json:"pending_total"`
		LastClose        *cashCloseRow  `json:"last_close"`
		Approvals        []approvalItem `json:"approvals"`
	}
	var o overview
	var bal *int64
	if err := s.pg.QueryRow(ctx, `
		SELECT (SELECT balance FROM wallets WHERE is_treasury LIMIT 1),
		       COALESCE((SELECT sum(amount) FILTER (WHERE direction = 'in')  FROM office_cash_entries
		                  WHERE created_at >= (date_trunc('day', now() AT TIME ZONE 'Asia/Damascus')
		                                       AT TIME ZONE 'Asia/Damascus')), 0)::bigint,
		       COALESCE((SELECT sum(amount) FILTER (WHERE direction = 'out') FROM office_cash_entries
		                  WHERE created_at >= (date_trunc('day', now() AT TIME ZONE 'Asia/Damascus')
		                                       AT TIME ZONE 'Asia/Damascus')), 0)::bigint,
		       COALESCE((SELECT sum(held) FROM driver_cash_boxes), 0)::bigint,
		       COALESCE((SELECT balance FROM wallets WHERE is_cash_holding LIMIT 1), 0)::bigint,
		       COALESCE((SELECT sum(amount) FROM payout_requests WHERE status IN ('pending', 'processing')), 0)::bigint,
		       COALESCE((SELECT sum(amount) FROM treasury_withdrawals
		                  WHERE created_at >= (date_trunc('month', now() AT TIME ZONE 'Asia/Damascus')
		                                       AT TIME ZONE 'Asia/Damascus')), 0)::bigint,
		       COALESCE((SELECT sum(amount) FROM office_cash_shortfalls WHERE status = 'pending'), 0)::bigint`).
		Scan(&bal, &o.TodayIn, &o.TodayOut, &o.CashInStreet, &o.MerchantCashDue,
			&o.PendingPayouts, &o.WithdrawalsMonth, &o.OpenShortfalls); err != nil {
		s.respondErr(w, err)
		return
	}
	if bal != nil {
		o.HasTreasury, o.TreasuryBalance = true, *bal
	}
	pos, err := cashboxPosition(ctx, s.pg)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	o.Cashbox = pos
	last, err := s.closesWhere(ctx, s.pg, `true`, 1)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if len(last) > 0 {
		o.LastClose = &last[0]
	}
	items, err := s.pendingApprovals(ctx, s.pg, r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	o.PendingCount = len(items)
	for _, it := range items {
		o.PendingTotal += it.Amount
	}
	if len(items) > 5 {
		items = items[:5]
	}
	o.Approvals = items
	httpx.JSON(w, http.StatusOK, o)
}

// ── كشفُ الحساب برصيدٍ جارٍ ──────────────────────────────────────────

type treasuryLine struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Kind      string    `json:"kind"`
	KindAr    string    `json:"kind_ar"`
	Amount    int64     `json:"amount"`
	Balance   int64     `json:"balance"` // الرصيدُ بعد هذا القيد
	Note      string    `json:"note"`
	Ref       string    `json:"ref"`
	OrderNo   *int64    `json:"order_number"`
	ByName    string    `json:"by_name"`
}

// treasuryStatementSQL **كلُّ قيود الخزينة برصيدها الجاري** — يُحسب على الدفتر
// كلِّه ثمّ يُرشَّح، فرصيدُ السطر هو رصيدُ الخزينة لحظتَها لا مجموعُ الصفحة.
const treasuryStatementSQL = `
	WITH t AS (
		SELECT t.id, t.created_at, t.kind, t.amount, COALESCE(t.note, '') AS note, t.ref, t.created_by,
		       sum(t.amount) OVER (ORDER BY t.created_at, t.id) AS balance_after
		  FROM wallet_transactions t
		  JOIN wallets w ON w.user_id = t.user_id AND w.is_treasury
	)
	SELECT t.id, t.created_at, t.kind, t.amount, t.balance_after::bigint, t.note, t.ref,
	       o.number::bigint, COALESCE(NULLIF(b.full_name, ''), b.phone::text, '')
	  FROM t
	  LEFT JOIN orders o ON o.id::text = t.ref
	  LEFT JOIN users b ON b.id = t.created_by
	 WHERE (NULLIF($1, '') IS NULL OR t.created_at >= (NULLIF($1, '')::date::timestamp AT TIME ZONE 'Asia/Damascus'))
	   AND (NULLIF($2, '') IS NULL OR t.created_at < ((NULLIF($2, '')::date + 1)::timestamp AT TIME ZONE 'Asia/Damascus'))
	   AND (NULLIF($3, '') IS NULL OR t.kind = $3)`

func scanTreasuryLine(rows pgx.Rows) (treasuryLine, error) {
	var l treasuryLine
	err := rows.Scan(&l.ID, &l.CreatedAt, &l.Kind, &l.Amount, &l.Balance, &l.Note, &l.Ref,
		&l.OrderNo, &l.ByName)
	l.KindAr = ledgerKindAr(l.Kind)
	return l, err
}

func treasuryRange(r *http.Request) (from, to, kind string, err error) {
	q := r.URL.Query()
	from, to, kind = q.Get("from"), q.Get("to"), q.Get("kind")
	var fd, td time.Time
	if from != "" {
		if fd, err = time.Parse("2006-01-02", from); err != nil {
			return "", "", "", errValidation
		}
	}
	if to != "" {
		if td, err = time.Parse("2006-01-02", to); err != nil {
			return "", "", "", errValidation
		}
	}
	if from != "" && to != "" && fd.After(td) {
		return "", "", "", errReportRangeInverted
	}
	if kind != "" && !approvalIdent.MatchString(kind) {
		return "", "", "", errValidation
	}
	return from, to, kind, nil
}

func (s *Server) handleTreasuryStatement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	from, to, kind, err := treasuryRange(r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	pg := pagingOf(r, 50)
	// **أوّلُ المدّة وآخرُها وما دخل وما خرج فيها** — من الدفتر نفسِه.
	var opening, closing, in, out int64
	var total int
	if err := s.pg.QueryRow(ctx, `
		SELECT COALESCE(sum(t.amount) FILTER (WHERE NULLIF($1, '') IS NOT NULL
		         AND t.created_at < (NULLIF($1, '')::date::timestamp AT TIME ZONE 'Asia/Damascus')), 0)::bigint,
		       COALESCE(sum(t.amount) FILTER (WHERE NULLIF($2, '') IS NULL
		         OR t.created_at < ((NULLIF($2, '')::date + 1)::timestamp AT TIME ZONE 'Asia/Damascus')), 0)::bigint
		  FROM wallet_transactions t JOIN wallets w ON w.user_id = t.user_id AND w.is_treasury`,
		from, to).Scan(&opening, &closing); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.pg.QueryRow(ctx, `
		SELECT count(*)::int,
		       COALESCE(sum(amount) FILTER (WHERE amount > 0), 0)::bigint,
		       COALESCE(-sum(amount) FILTER (WHERE amount < 0), 0)::bigint
		  FROM (`+treasuryStatementSQL+`) s(id, created_at, kind, amount, balance, note, ref, num, by_name)`,
		from, to, kind).Scan(&total, &in, &out); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(ctx, treasuryStatementSQL+`
		ORDER BY t.created_at DESC, t.id DESC
		LIMIT $4 OFFSET $5`, from, to, kind, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	lines := []treasuryLine{}
	for rows.Next() {
		l, err := scanTreasuryLine(rows)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		lines = append(lines, l)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	res := paged("lines", lines, total, pg)
	res["opening"], res["closing"], res["in"], res["out"] = opening, closing, in, out
	httpx.JSON(w, http.StatusOK, res)
}

// handleTreasuryStatementExport **الكشفُ ملفّاً** — بيوم دمشق، بالعربيّة، ولا معادلةَ
// في خليّة (`csvSafe`).
func (s *Server) handleTreasuryStatementExport(w http.ResponseWriter, r *http.Request) {
	from, to, kind, err := treasuryRange(r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(r.Context(), treasuryStatementSQL+`
		ORDER BY t.created_at, t.id`, from, to, kind)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	name := "treasury"
	if from != "" || to != "" {
		name += "-" + from + "_" + to
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.csv"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"التاريخ (دمشق)", "النوع", "المبلغ", "الرصيد بعده", "الملاحظة",
		"رقم الطلب", "بيد"})
	dam := damascusLoc()
	count := 0
	for rows.Next() {
		l, err := scanTreasuryLine(rows)
		if err != nil {
			s.logger.Error("treasury export", "error", err)
			break
		}
		orderNo := ""
		if l.OrderNo != nil {
			orderNo = strconv.FormatInt(*l.OrderNo, 10)
		}
		_ = cw.Write([]string{
			l.CreatedAt.In(dam).Format("2006-01-02 15:04"), l.KindAr,
			strconv.FormatInt(l.Amount, 10), strconv.FormatInt(l.Balance, 10),
			csvSafe(l.Note), orderNo, csvSafe(l.ByName),
		})
		count++
	}
	cw.Flush()
	s.audit(r, "finance.treasury_statement_exported", "wallet", "", map[string]any{
		"from": from, "to": to, "kind": kind, "rows": count,
	})
}

func damascusLoc() *time.Location {
	if loc, err := time.LoadLocation("Asia/Damascus"); err == nil {
		return loc
	}
	return time.FixedZone("Asia/Damascus", 3*60*60)
}

// ── سحبُ الأدمن من رصيد الخزينة ──────────────────────────────────────

type treasuryWithdrawalRow struct {
	ID          string    `json:"id"`
	Amount      int64     `json:"amount"`
	Note        string    `json:"note"`
	FromCashbox bool      `json:"from_cashbox"`
	ByName      string    `json:"by_name"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Server) handleTreasuryWithdrawals(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT tw.id::text, tw.amount, tw.note, tw.from_cashbox,
		       COALESCE(NULLIF(u.full_name, ''), u.phone::text, ''), tw.created_at
		  FROM treasury_withdrawals tw JOIN users u ON u.id = tw.withdrawn_by
		 ORDER BY tw.created_at DESC LIMIT 100`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []treasuryWithdrawalRow{}
	for rows.Next() {
		var x treasuryWithdrawalRow
		if err := rows.Scan(&x.ID, &x.Amount, &x.Note, &x.FromCashbox, &x.ByName, &x.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"withdrawals": out})
}

// handleTreasuryWithdraw **مديرُ المنصّة يسحب من رصيد الخزينة.**
//
// **قيدٌ في الخزينة بنوعه** (`treasury_withdrawal`) باسم من سحب — فلا يُقرأ
// خسارةً ولا مصروفاً ولا ربحاً. **ونقدٌ أُخذ من الدرج يخرج من الصندوق أيضاً**
// في المعاملة نفسِها. **ولا يُسحب أكثرُ من الرصيد**: مالٌ ليس في الخزينة لا يُسحب.
func (s *Server) handleTreasuryWithdraw(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Amount      int64  `json:"amount"`
		Note        string `json:"note"`
		FromCashbox *bool  `json:"from_cashbox"`
	}](r)
	if err != nil || req.Amount <= 0 {
		s.respondErr(w, errValidation)
		return
	}
	note := clip(strings.TrimSpace(req.Note), 500)
	if note == "" {
		s.respondErr(w, errNoteRequired)
		return
	}
	fromCashbox := req.FromCashbox == nil || *req.FromCashbox
	actor := userIDFrom(r)
	s.WithIdempotentTx(w, r, func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error) {
		var tid string
		var bal int64
		if err := q.QueryRow(ctx, `
			SELECT user_id::text, balance FROM wallets WHERE is_treasury LIMIT 1 FOR UPDATE`).
			Scan(&tid, &bal); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return IdempotentBody{}, errTreasuryMissing
			}
			return IdempotentBody{}, err
		}
		if req.Amount > bal {
			return IdempotentBody{}, errTreasuryOverBalance
		}
		var id string
		if err := q.QueryRow(ctx, `
			INSERT INTO treasury_withdrawals (amount, note, withdrawn_by, from_cashbox)
			VALUES ($1, $2, $3, $4) RETURNING id::text`,
			req.Amount, note, actor, fromCashbox).Scan(&id); err != nil {
			return IdempotentBody{}, err
		}
		if _, err := s.wallet.ApplyTx(ctx, q, tid, -req.Amount, "treasury_withdrawal", id,
			treasuryWithdrawalNote+" — "+note, &actor); err != nil {
			return IdempotentBody{}, err
		}
		if fromCashbox {
			if err := officecash.Record(ctx, q, officecash.Entry{
				Direction: officecash.Out, Amount: req.Amount,
				Source: officecash.SourceTreasuryWithdrawal, Ref: id, UserID: actor,
				Actor: actor, Note: note,
			}); err != nil {
				return IdempotentBody{}, err
			}
		}
		if err := s.auditTx(ctx, q, r, "finance.treasury_withdrawal", "treasury_withdrawal", id,
			map[string]any{"amount": req.Amount, "note": note, "from_cashbox": fromCashbox}); err != nil {
			return IdempotentBody{}, err
		}
		return IdempotentBody{
			Status:      http.StatusCreated,
			Payload:     map[string]any{"id": id, "balance": bal - req.Amount},
			AfterCommit: func() { s.touch("wallet", "ops") },
		}, nil
	})
}

// ── صحّةُ الدفتر ──────────────────────────────────────────────────────

type ledgerCheck struct {
	ID     string     `json:"id"`
	Family string     `json:"family"`
	Name   string     `json:"name"`
	Why    string     `json:"why"`
	OK     bool       `json:"ok"`
	Count  int        `json:"count"`
	Error  string     `json:"error,omitempty"`
	Cols   []string   `json:"cols,omitempty"`
	Sample [][]string `json:"sample,omitempty"`
}

// handleTreasuryHealth **فحوصُ `moneycheck` نفسُها** — كلٌّ أحمرُ أو أخضر، والخرقُ
// بصفوفه الأولى. **ولا فحصَ ثانٍ يُكتب للشاشة** — الحزمةُ نفسُها (`fininv`).
func (s *Server) handleTreasuryHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	out := []ledgerCheck{}
	bad := 0
	// **وأنواعُ القيد أوّلاً** — نوعٌ بلا عقدٍ يجعل ما بعده مقيساً بمسطرةٍ ناقصة.
	kc := ledgerCheck{ID: "KINDS", Family: "FI-01", Name: "أنواعُ القيد في القاعدة = العقود",
		Why: "نوعٌ في القاعدة بلا عقدٍ أو عقدٌ بلا نوع"}
	if schema, err := fininv.SchemaKinds(ctx, s.pg); err != nil {
		kc.Error = err.Error()
	} else {
		missing, stale := fininv.KindDrift(schema)
		kc.Count = len(missing) + len(stale)
		for _, k := range missing {
			kc.Sample = append(kc.Sample, []string{"بلا عقد", k})
		}
		for _, k := range stale {
			kc.Sample = append(kc.Sample, []string{"عقدٌ بلا نوع", k})
		}
	}
	kc.OK = kc.Error == "" && kc.Count == 0
	if !kc.OK {
		bad++
	}
	out = append(out, kc)
	for _, c := range fininv.Select() {
		if !c.Ops {
			continue
		}
		lc := ledgerCheck{ID: c.ID, Family: string(c.Family), Name: c.Name, Why: c.Why}
		vs, err := fininv.Run(ctx, s.pg, c.ID)
		switch {
		case err != nil:
			lc.Error = err.Error()
		case len(vs) > 0:
			lc.Count = len(vs[0].Rows)
			lc.Cols = vs[0].Cols
			for i, row := range vs[0].Rows {
				if i >= 3 {
					break
				}
				cells := make([]string, len(row))
				for j, v := range row {
					cells[j] = fmt.Sprint(v)
				}
				lc.Sample = append(lc.Sample, cells)
			}
		}
		lc.OK = lc.Error == "" && lc.Count == 0
		if !lc.OK {
			bad++
		}
		out = append(out, lc)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"checks": out, "failing": bad, "total": len(out), "checked_at": time.Now(),
	})
}
