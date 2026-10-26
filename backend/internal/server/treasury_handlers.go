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

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/envguard"
	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/officecash"
	"github.com/servacode/rahalgo/backend/internal/orders"
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
		TreasuryBalance  int64        `json:"treasury_balance"`
		HasTreasury      bool         `json:"has_treasury"`
		Cashbox          cashPosition `json:"cashbox"`
		TodayIn          int64        `json:"today_in"`
		TodayOut         int64        `json:"today_out"`
		CashInStreet     int64        `json:"cash_in_street"`
		MerchantCashDue  int64        `json:"merchant_cash_due"`
		PendingPayouts   int64        `json:"pending_payouts"`
		WithdrawalsMonth int64        `json:"withdrawals_month"`
		OpenShortfalls   int64        `json:"open_shortfalls"`
		PendingCount     int          `json:"pending_count"`
		PendingTotal     int64        `json:"pending_total"`
		PendingUndecided int          `json:"pending_undecided"`
		// ProfitCards **صافي الربح اليومَ وهذا الأسبوعَ وهذا الشهرَ ومن أوّل يوم** (قرارُ المالك
		// ٢٠٢٦-١٠-١٥) — **«صافي الربح» في صفحة الأرباح بعينه** (`platformProfit(..).Net`) للمدى
		// نفسِه بيوم دمشق، ومعه المدى فتفتح البطاقةُ الصفحةَ عليه.
		ProfitCards []profitCard   `json:"profit_cards"`
		LastClose   *cashCloseRow  `json:"last_close"`
		Approvals   []approvalItem `json:"approvals"`
	}
	var o overview
	var bal *int64
	if err := s.pg.QueryRow(ctx, `
		SELECT (SELECT balance FROM wallets WHERE is_treasury LIMIT 1),
		       COALESCE((SELECT sum(amount) FILTER (WHERE direction = 'in')  FROM office_cash_entries
		                  WHERE created_at >= account_day_start(account_day(now()))), 0)::bigint,
		       COALESCE((SELECT sum(amount) FILTER (WHERE direction = 'out') FROM office_cash_entries
		                  WHERE created_at >= account_day_start(account_day(now()))), 0)::bigint,
		       COALESCE((SELECT sum(held) FROM driver_cash_boxes), 0)::bigint,
		       -- **«مستحقّ للمتاجر نقداً» رقمُ تبويب المتاجر نفسُه** (قرارُ المالك ٢٠٢٦-١٠-١٥) —
		       -- من صفوف التسوية لا من رصيد محفظة الاحتباس.
		       `+merchantCashDueTotalSQL+`::bigint,
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
	o.PendingTotal, o.PendingUndecided = approvalsTotal(items)
	cards, err := s.profitCards(ctx, time.Now())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	o.ProfitCards = cards
	if len(items) > 5 {
		items = items[:5]
	}
	o.Approvals = items
	httpx.JSON(w, http.StatusOK, o)
}

// profitCard **بطاقةُ صافي الربح لمدًى** — `period` يومٌ أو أسبوعٌ أو شهرٌ أو `all`.
type profitCard struct {
	Period string `json:"period"`
	From   string `json:"from"`
	To     string `json:"to"`
	Net    int64  `json:"net"`
}

// profitCards **الصافي للمُدد الأربع** — بحساب صفحة الأرباح نفسِه، لا بمعادلةٍ ثانية.
//
// **والأسبوعُ آخرُ سبعة أيّامٍ باليوم** — كما تعرّفه صفحةُ الأرباح نفسُها (زرُّ «هذا الأسبوع»
// في `profits/page.tsx`)، فتطابق البطاقةُ ما يراه المالكُ هناك (`damascusPeriod`).
func (s *Server) profitCards(ctx context.Context, now time.Time) ([]profitCard, error) {
	out := make([]profitCard, 0, 4)
	for _, period := range []string{"day", "week", "month", "all"} {
		from, to, _ := damascusPeriod(period, now, s.settings.GetInt(ctx, "platform.day_start_hour"))
		p, err := s.platformProfit(ctx, from, to)
		if err != nil {
			return nil, err
		}
		out = append(out, profitCard{Period: period, From: from, To: to, Net: p.Net})
	}
	return out, nil
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
	// Personal **«حركة شخصية»** (قرارُ المالك ٢٠٢٦-١٠-١٥) — الخزينةُ محفظةُ الأدمن، فدفعُ طلباته
	// وشحنُه ومستردُّه تمرّ فيها **وليست ربحاً**: الأنواعُ «خارج الربح» نفسُها في صفحة الأرباح
	// (`profitOutsideKinds`) — **إلّا سحبَ الأدمن**: له اسمُه وسطرُه.
	Personal bool `json:"personal"`
	// Grouped **«ربح طلب #N»** — قيودُ `platform_profit` لطلبٍ واحدٍ سطراً واحداً (`Entries` عددُها)،
	// **وتفصيلُه من بابه** (`/treasury/statement/orders/{id}`). عرضٌ وحدَه: الدفترُ لا يتغيّر.
	Grouped bool   `json:"grouped"`
	Entries int    `json:"entries"`
	OrderID string `json:"order_id"`
}

// treasuryPersonalSQL **أحركةٌ شخصيّة؟** — على عمود النوع `t.kind`.
const treasuryPersonalSQL = `(t.kind IN (` + profitOutsideKinds + `) AND t.kind <> 'treasury_withdrawal')`

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
	       o.number::bigint, COALESCE(NULLIF(b.full_name, ''), b.phone::text, ''),
	       ` + treasuryPersonalSQL + `
	  FROM t
	  LEFT JOIN orders o ON o.id::text = t.ref
	  LEFT JOIN users b ON b.id = t.created_by
	 WHERE (NULLIF($1, '') IS NULL OR t.created_at >= account_day_start(NULLIF($1, '')::date))
	   AND (NULLIF($2, '') IS NULL OR t.created_at < account_day_start(NULLIF($2, '')::date + 1))
	   AND (NULLIF($3, '') IS NULL OR t.kind = $3)`

// treasuryGroupedSQL **الكشفُ كما يُعرض**: قيودُ ربح الطلب الواحد سطراً واحداً «ربح طلب #N».
//
// **والرصيدُ الجاري للسطر المجمَّع رصيدُ الخزينة بعد آخر قيدٍ منه** — والمبلغُ مجموعُ قيوده
// في المدى. والباقي سطرٌ لكلّ قيد كما كان. **والدفترُ لا يُمسّ** — تجميعٌ عند القراءة.
const treasuryGroupedSQL = `
	WITH t AS (
		SELECT t.id, t.created_at, t.kind, t.amount, COALESCE(t.note, '') AS note, t.ref, t.created_by,
		       sum(t.amount) OVER (ORDER BY t.created_at, t.id) AS balance_after
		  FROM wallet_transactions t
		  JOIN wallets w ON w.user_id = t.user_id AND w.is_treasury
	), f AS (
		SELECT t.*, o.id AS order_id, o.number AS order_number, ` + treasuryPersonalSQL + ` AS personal,
		       CASE WHEN t.kind = 'platform_profit' AND o.id IS NOT NULL THEN 'o:' || o.id::text
		            ELSE 'l:' || t.id::text END AS grp
		  FROM t
		  LEFT JOIN orders o ON o.id::text = t.ref
		 WHERE (NULLIF($1, '') IS NULL OR t.created_at >= account_day_start(NULLIF($1, '')::date))
		   AND (NULLIF($2, '') IS NULL OR t.created_at < account_day_start(NULLIF($2, '')::date + 1))
		   AND (NULLIF($3, '') IS NULL OR t.kind = $3)
	), g AS (
		SELECT grp, max(id) AS id, max(created_at) AS created_at, min(kind) AS kind,
		       sum(amount)::bigint AS amount,
		       (array_agg(balance_after ORDER BY created_at DESC, id DESC))[1] AS balance_after,
		       (array_agg(note ORDER BY created_at DESC, id DESC))[1] AS note,
		       min(ref) AS ref, max(order_number) AS order_number,
		       COALESCE(max(order_id::text), '') AS order_id,
		       (array_agg(created_by ORDER BY created_at DESC, id DESC))[1] AS created_by,
		       bool_or(personal) AS personal,
		       count(*)::int AS n
		  FROM f GROUP BY grp
	)
	SELECT g.id, g.created_at, g.kind, g.amount, g.balance_after::bigint, g.note, g.ref,
	       g.order_number::bigint, COALESCE(NULLIF(b.full_name, ''), b.phone::text, ''),
	       g.personal, g.grp LIKE 'o:%', g.n, g.order_id
	  FROM g
	  LEFT JOIN users b ON b.id = g.created_by`

func scanTreasuryLine(rows pgx.Rows) (treasuryLine, error) {
	var l treasuryLine
	err := rows.Scan(&l.ID, &l.CreatedAt, &l.Kind, &l.Amount, &l.Balance, &l.Note, &l.Ref,
		&l.OrderNo, &l.ByName, &l.Personal)
	l.KindAr = ledgerKindAr(l.Kind)
	l.Entries = 1
	return l, err
}

func scanTreasuryGroup(rows pgx.Rows) (treasuryLine, error) {
	var l treasuryLine
	err := rows.Scan(&l.ID, &l.CreatedAt, &l.Kind, &l.Amount, &l.Balance, &l.Note, &l.Ref,
		&l.OrderNo, &l.ByName, &l.Personal, &l.Grouped, &l.Entries, &l.OrderID)
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
		         AND t.created_at < account_day_start(NULLIF($1, '')::date)), 0)::bigint,
		       COALESCE(sum(t.amount) FILTER (WHERE NULLIF($2, '') IS NULL
		         OR t.created_at < account_day_start(NULLIF($2, '')::date + 1)), 0)::bigint
		  FROM wallet_transactions t JOIN wallets w ON w.user_id = t.user_id AND w.is_treasury`,
		from, to).Scan(&opening, &closing); err != nil {
		s.respondErr(w, err)
		return
	}
	// **وما دخل وما خرج من القيود نفسِها** — لا من السطور المجمَّعة (ربحُ طلبٍ صافيه لا جانباه).
	// **والداخلُ والخارجُ من الأسطر المعروضة نفسِها** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «صحّح الصفحة بحيث تكون
	// مفهومة») — صافي كلّ طلبٍ سطرٌ واحد، فداخلٌ = مجموعُ الموجب وخارجٌ = مجموعُ السالب. كانت تُجمع
	// خطواتُ القيد الداخليّة (−٧٦ عند الاستلام ثمّ الباقي) فلا تطابق ما يُقرأ تحتها، والصافي هو هو.
	if err := s.pg.QueryRow(ctx, `
		SELECT COALESCE(sum(amount) FILTER (WHERE amount > 0), 0)::bigint,
		       COALESCE(-sum(amount) FILTER (WHERE amount < 0), 0)::bigint
		  FROM (`+treasuryGroupedSQL+`) s`,
		from, to, kind).Scan(&in, &out); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.pg.QueryRow(ctx, `
		SELECT count(*)::int FROM (`+treasuryGroupedSQL+`) s`, from, to, kind).Scan(&total); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(ctx, treasuryGroupedSQL+`
		ORDER BY g.created_at DESC, g.id DESC
		LIMIT $4 OFFSET $5`, from, to, kind, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	lines := []treasuryLine{}
	for rows.Next() {
		l, err := scanTreasuryGroup(rows)
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
	// **والملفُّ قيدٌ قيدٌ كما في الدفتر** — والتجميعُ عرضٌ في الشاشة. وعمودُ «حركة شخصية» يقول
	// ما ليس ربحاً من حركات الأدمن (قرارُ المالك ٢٠٢٦-١٠-١٥).
	_ = cw.Write([]string{"التاريخ (دمشق)", "النوع", "المبلغ", "الرصيد بعده", "الملاحظة",
		"رقم الطلب", "بيد", "حركة شخصية"})
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
		personal := ""
		if l.Personal {
			personal = "نعم"
		}
		_ = cw.Write([]string{
			l.CreatedAt.In(dam).Format("2006-01-02 15:04"), l.KindAr,
			strconv.FormatInt(l.Amount, 10), strconv.FormatInt(l.Balance, 10),
			csvSafe(l.Note), orderNo, csvSafe(l.ByName), personal,
		})
		count++
	}
	cw.Flush()
	s.audit(r, "finance.treasury_statement_exported", "wallet", "", map[string]any{
		"from": from, "to": to, "kind": kind, "rows": count,
	})
}

// ── تفصيلُ «ربح طلب #N» ──────────────────────────────────────────────

// orderProfitPart سطرٌ في تفصيل ربح الطلب — بأثره على الربح (السالبُ ينقصه).
type orderProfitPart struct {
	Key    string `json:"key"`
	Amount int64  `json:"amount"`
}

type orderProfitEntry struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Amount    int64     `json:"amount"`
	Note      string    `json:"note"`
}

// orderProfitBreakdown **من أين جاء ربحُ طلبٍ في الخزينة** — بالقاعدة نفسِها التي تبني كشفَ
// الأرباح (`platformProfit`): الهامشُ والعمولةُ وأجرةُ التوصيل من أعمدة الطلب، وأجرةُ السائق
// والخصمُ والعروضُ تنقصه، ونصيبُ المندوب من دفتره. **وما لا تشرحه الأعمدةُ سطرُ «فرق»**
// (`diff`) — فالسطورُ تُجمع دائماً إلى ما قيّدته الخزينة في المدى، ولا يُخفى شيء.
//
// **والطلبُ الذي لم يُسلَّم** (فشل، أُلغي، رُدّ) لا تشرحه أعمدةُ البيع — فسطرٌ واحدٌ بحاله
// (`not_delivered`) بمبلغ قيوده.
func (s *Server) orderProfitBreakdown(ctx context.Context, orderID, from, to string) (map[string]any, error) {
	var number int64
	var status string
	var margin, commission, deliveryFee, driverFee, discount, free, itemCut, rep int64
	err := s.pg.QueryRow(ctx, `
		SELECT o.number::bigint, o.status,
		       `+orders.OrderMarginSQL("o.id")+` + COALESCE(it.platform_cut, 0),
		       o.platform_commission,
		       o.delivery_fee + o.promo_delivery_waived, o.driver_fee,
		       o.discount, o.promo_delivery_waived, COALESCE(it.platform_cut, 0),
		       COALESCE((SELECT sum(w.amount) FROM wallet_transactions w
		                  WHERE w.kind = 'commission' AND w.ref = o.id::text), 0)::bigint
		  FROM orders o
		  LEFT JOIN LATERAL (
		      SELECT sum(oi.offer_cut * oi.qty) FILTER (WHERE oi.offer_borne_by = 'platform') AS platform_cut
		        FROM order_items oi WHERE oi.order_id = o.id) it ON true
		 WHERE o.id = $1`, orderID).
		Scan(&number, &status, &margin, &commission, &deliveryFee, &driverFee, &discount, &free, &itemCut, &rep)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.pg.Query(ctx, `
		SELECT t.id, t.created_at, t.amount, COALESCE(t.note, '')
		  FROM wallet_transactions t
		  JOIN wallets w ON w.user_id = t.user_id AND w.is_treasury
		 WHERE t.kind = 'platform_profit' AND t.ref = $1
		   AND (NULLIF($2, '') IS NULL OR t.created_at >= account_day_start(NULLIF($2, '')::date))
		   AND (NULLIF($3, '') IS NULL OR t.created_at < account_day_start(NULLIF($3, '')::date + 1))
		 ORDER BY t.created_at, t.id`, orderID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []orderProfitEntry{}
	var total int64
	for rows.Next() {
		var e orderProfitEntry
		if err := rows.Scan(&e.ID, &e.CreatedAt, &e.Amount, &e.Note); err != nil {
			return nil, err
		}
		total += e.Amount
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	parts := []orderProfitPart{}
	if status == "delivered" {
		for _, p := range []orderProfitPart{
			{"margin", margin},
			{"commission", commission},
			{"delivery_fee", deliveryFee},
			{"driver_fee", -driverFee},
			{"discount", -discount},
			{"free_delivery", -free},
			{"item_discounts", -itemCut},
			{"rep_share", -rep},
		} {
			if p.Amount != 0 {
				parts = append(parts, p)
			}
		}
	} else {
		parts = append(parts, orderProfitPart{"not_delivered", total})
	}
	var sum int64
	for _, p := range parts {
		sum += p.Amount
	}
	if d := total - sum; d != 0 {
		parts = append(parts, orderProfitPart{"diff", d})
	}
	return map[string]any{
		"order_id": orderID, "order_number": number, "status": status,
		"total": total, "lines": parts, "entries": entries,
	}, nil
}

// handleTreasuryOrderBreakdown **تفصيلُ سطر «ربح طلب #N»** —
// `GET /admin/treasury/statement/orders/{id}?from=&to=` بمدى الكشف نفسِه، فمجموعُ التفصيل هو
// مبلغُ السطر.
func (s *Server) handleTreasuryOrderBreakdown(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	from, to, _, err := treasuryRange(r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	out, err := s.orderProfitBreakdown(r.Context(), id, from, to)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
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
	out, bad := s.ledgerHealth(r.Context())
	commit, _ := envguard.BuildInfo()
	httpx.JSON(w, http.StatusOK, map[string]any{
		"checks": out, "failing": bad, "total": len(out), "checked_at": time.Now(),
		"source_commit": commit,
	})
}

// ledgerHealth **الفحوصُ نفسُها بلا HTTP** — تقرؤها الشاشةُ والراصدُ كلَّ ساعة
// (مراقبةُ المنصّة — الدفعةُ الثالثة ٢٠٢٦-١٠-١٧). **مصدرٌ واحدٌ لا نسختان**:
// ما تراه في «صحة الدفتر» هو بعينه ما يُنبَّه به على الواتساب.
func (s *Server) ledgerHealth(ctx context.Context) ([]ledgerCheck, int) {
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
	return out, bad
}
