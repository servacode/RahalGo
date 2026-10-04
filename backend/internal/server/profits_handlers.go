package server

/*
الأرباح والخسائر — تبويب «الأرباح والخسائر» في قسم الخزينة.

قرارات المالك ٢٠٢٦-١٠-٠٤ (تُقرأ في docs/TRUTH.md):

 1. الأرباح حسب تاريخ القيد في دفتر الخزينة، بيوم دمشق، والشاشة تقول ذلك.
    والطلب يُحسب ربحه كاملاً في يوم آخر قيد تسوية له (يوم تسليمه أو فشله أو
    إلغائه) — فقيد الاستلام وقيد التسليم لا ينقسمان على يومين.
 2. خسارة الطلبات الفاشلة والملغاة سطر لحاله (ومعها المستردّ بعد التسليم).
 3. سحب الأدمن من الخزينة والإيداع اليدويّ ليسا ربحاً — يظهران في كشف
    الخزينة فقط، وهنا رقماً «خارج الربح».
 4. مال انقبض لطلب لم يُسلَّم بعد «ربح معلّق» بسطر لحاله، ولا يدخل الصافي.
 5. مكافآت الدعوات ومكافآت الأهداف سطران.

والكشف يتجمّع بالضبط:

	هامش + عمولة المتجر + حصّة التوصيل − خصومات (± فروق تسوية)
	− نصيب المندوب − خسائر الفاشلة والملغاة − تعويضات (+ مستردّ)
	− مكافآت الدعوات − مكافآت الأهداف − مصروفات التشغيل + عقوبات = الصافي

والصافي يُحسب مستقلّاً: كلّ قيد في الخزينة ليس «خارج الربح» ولا «معلّقاً».
فإن ظهر في الخزينة نوع قيد جديد لا يعرفه هذا الملفّ دخل الصافي ولم يدخل أيّ
سطر، فيسقط سطر التحقّق «مجموع البنود = الصافي» ويقول الفرق.

ومصروفات التشغيل على يوم الصرف (spent_at) لا يوم التسجيل — كصفحة المصروفات،
فإيجار أيلول المسجَّل في تشرين يُحسب في أيلول، وإلغاؤه يُطرح من أيلول.
*/

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// profitRange شرط المدى بيوم دمشق على عمود وقت — $1 و$2 نصّان YYYY-MM-DD،
// والفارغ بلا حدّ.
func profitRange(col string) string {
	return `($1 = '' OR ` + col + ` >= (NULLIF($1, '')::date::timestamp AT TIME ZONE 'Asia/Damascus'))
	   AND ($2 = '' OR ` + col + ` < ((NULLIF($2, '')::date + 1)::timestamp AT TIME ZONE 'Asia/Damascus'))`
}

// refUUID يحوّل مرجع القيد (نصّ) إلى uuid إن كان شكله uuid، وإلّا NULL.
func refUUID(col string) string {
	return `(CASE WHEN ` + col + ` ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
	         THEN ` + col + `::uuid END)`
}

// profitClosedStatuses حالات انتهى فيها الطلب — يُحسب ربحه أو خسارته.
// وما سواها طلب جارٍ: ماله «ربح معلّق».
const profitClosedStatuses = `'delivered','failed','cancelled','rejected','refunded'`

// profitOutsideKinds أنواع قيد تقع على محفظة الخزينة وليست ربحاً ولا خسارة:
// الخزينة محفظة الأدمن، فشحنه وسحبه وطلباته الشخصيّة ومالُ الاحتباس تمرّ
// فيها ولا تُحسب ربحاً (القرار ٣).
const profitOutsideKinds = `'topup','payout','adjustment','order_payment','refund','compensation',
	'commission','driver_earning','merchant_earning','merchant_cash_accrued','merchant_cash_paid','treasury_withdrawal'`

// treasuryEntriesCTE قيود الخزينة مصنّفة: لكلّ قيد بابه (bucket) ويومه (at).
// ويليه شرط المدى على at في الاستعلام نفسه (inr).
var treasuryEntriesCTE = `
	WITH tr AS (SELECT user_id FROM wallets WHERE is_treasury LIMIT 1),
	raw AS (
		SELECT t.id, t.kind, t.amount, t.ref, t.created_at, ` + refUUID("t.ref") + ` AS ref_id
		FROM wallet_transactions t JOIN tr ON tr.user_id = t.user_id
	),
	j AS (
		SELECT r.*, o.id AS order_id, o.status AS ostatus, wr.kind AS wr_kind, x.spent_at
		FROM raw r
		LEFT JOIN orders o ON o.id = r.ref_id
		LEFT JOIN wallet_requests wr ON wr.id = r.ref_id
		LEFT JOIN expenses x ON x.id = r.ref_id AND r.kind = 'operating_expense'
	),
	c AS (
		SELECT j.kind, j.amount, j.order_id, j.ostatus,
			CASE
				WHEN j.kind = 'platform_profit' AND j.order_id IS NOT NULL AND j.ostatus = 'delivered' THEN 'order'
				WHEN j.kind = 'platform_profit' AND j.order_id IS NOT NULL AND j.ostatus = 'failed' THEN 'lost_failed'
				WHEN j.kind = 'platform_profit' AND j.order_id IS NOT NULL AND j.ostatus IN ('cancelled','rejected') THEN 'lost_cancelled'
				WHEN j.kind = 'platform_profit' AND j.order_id IS NOT NULL AND j.ostatus = 'refunded' THEN 'lost_refunded'
				WHEN j.kind = 'platform_profit' AND j.order_id IS NOT NULL THEN 'pending'
				-- سدادُ دَينٍ من شحن المحفظة (قسمُ الديون ٢٠٢٦-١٠-٠٤): قيدُ الخزينة مرجعُه طلبُ
				-- الشحن، وهو مالٌ مستردٌّ كسداد الدَّين نقداً بالمكتب — لا حركةٌ يدويّة.
				WHEN j.kind = 'platform_profit' AND j.wr_kind = 'topup' THEN 'recovered'
				WHEN j.kind IN ('platform_profit','platform_expense') AND j.wr_kind IN ('adjustment','topup') THEN 'outside'
				WHEN j.kind = 'platform_profit' THEN 'recovered'
				WHEN j.kind = 'platform_expense' THEN 'compensation'
				WHEN j.kind = 'reward' AND j.ref <> '' THEN 'referral'
				WHEN j.kind = 'reward' THEN 'target'
				WHEN j.kind = 'operating_expense' THEN 'opex'
				WHEN j.kind = 'penalty' THEN 'penalty'
				WHEN j.kind IN (` + profitOutsideKinds + `) THEN 'outside'
				ELSE 'unclassified'
			END AS bucket,
			CASE
				WHEN j.kind = 'platform_profit' AND j.order_id IS NOT NULL AND j.ostatus IN (` + profitClosedStatuses + `)
					THEN max(j.created_at) FILTER (WHERE j.kind = 'platform_profit') OVER (PARTITION BY j.order_id)
				WHEN j.kind = 'operating_expense' AND j.spent_at IS NOT NULL
					THEN j.spent_at::timestamp AT TIME ZONE 'Asia/Damascus'
				ELSE j.created_at
			END AS at
		FROM j
	),
	inr AS (SELECT * FROM c WHERE ` + profitRange("c.at") + `)`

// profitRangeOK يقرأ المدى من الرابط ويردّ الخطأ إن كان تاريخاً غير صالح أو
// «من» بعد «إلى».
func profitRangeOK(from, to string) error {
	var fd, td time.Time
	var err error
	if from != "" {
		if fd, err = time.Parse("2006-01-02", from); err != nil {
			return errValidation
		}
	}
	if to != "" {
		if td, err = time.Parse("2006-01-02", to); err != nil {
			return errValidation
		}
	}
	if from != "" && to != "" && fd.After(td) {
		return errReportRangeInverted
	}
	return nil
}

// handleProfits الأرباح والخسائر بتبويباتها الخمسة.
func (s *Server) handleProfits(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	if err := profitRangeOK(from, to); err != nil {
		s.respondErr(w, err)
		return
	}
	switch q.Get("tab") {
	case "", "platform":
		s.profitsPlatform(w, r, from, to)
	case "customers", "reps", "drivers":
		s.profitsParties(w, r, from, to, q.Get("tab"))
	case "merchants":
		s.profitsMerchants(w, r, from, to)
	default:
		s.respondErr(w, httpx.ErrNotFound)
	}
}

// ══════════════════════════════════════════════════════════════════════
//
//	١ · المنصّة — كشف الأرباح والخسائر
//
// ══════════════════════════════════════════════════════════════════════

// profitLine سطر في الكشف — مبلغه بأثره على الربح (السالب ينقص الربح).
type profitLine struct {
	Key    string `json:"key"`
	Amount int64  `json:"amount"`
}

func (s *Server) profitsPlatform(w http.ResponseWriter, r *http.Request, from, to string) {
	p, err := s.platformProfit(r.Context(), from, to)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	sum := p.LinesSum()
	httpx.JSON(w, http.StatusOK, map[string]any{
		// الأساس مكتوب في الرد — والشاشة تقوله (القرار ١).
		"basis":  "ledger_entry_damascus",
		"orders": p.Orders, "sales": p.Sales,
		"income":               p.Income(),
		"lines":                p.Lines(),
		"lines_sum":            sum,
		"net":                  p.Net,
		"check_ok":             sum == p.Net,
		"check_diff":           p.Net - sum,
		"store_item_discounts": p.StoreItemDiscounts,
		"pending":              p.Pending,
		"outside":              p.Outside,
		"losses":               p.Losses(),
		"expenses_paid":        p.Opex,
	})
}

// platformProfit أرباح المنصّة في مدى — حساب واحد تقرؤه صفحة الأرباح
// ورئيسيّة الإدارة (/admin/overview) والتقارير (/admin/reports) و/admin/stats.
//
// ⚠️ تغيّر معنى Net في ٢٠٢٦-١٠-٠٤ (قرارات الأرباح): كان مجموع حركة الخزينة
// كلّها، وصار صافي الكشف — بلا سحب الأدمن والإيداع اليدويّ وبلا الربح
// المعلّق. والمنادون الثلاثة يقرؤون Net نفسه فيبقون متطابقين.
//
// والمدى بيوم دمشق — YYYY-MM-DD شامل طرفيه، والفارغ بلا حدّ.
func (s *Server) platformProfit(ctx context.Context, from, to string) (platformProfitSum, error) {
	var p platformProfitSum

	// ── قيود الخزينة بأبوابها ──
	if err := s.pg.QueryRow(ctx, treasuryEntriesCTE+`
		SELECT
			COALESCE(sum(amount) FILTER (WHERE bucket = 'order'), 0),
			COALESCE(sum(amount) FILTER (WHERE bucket = 'lost_failed'), 0),
			COALESCE(sum(amount) FILTER (WHERE bucket = 'lost_cancelled'), 0),
			COALESCE(sum(amount) FILTER (WHERE bucket = 'lost_refunded'), 0),
			COALESCE(sum(amount) FILTER (WHERE bucket = 'compensation'), 0),
			COALESCE(sum(amount) FILTER (WHERE bucket = 'recovered'), 0),
			COALESCE(sum(amount) FILTER (WHERE bucket = 'referral'), 0),
			COALESCE(sum(amount) FILTER (WHERE bucket = 'target'), 0),
			COALESCE(sum(amount) FILTER (WHERE bucket = 'opex'), 0),
			COALESCE(sum(amount) FILTER (WHERE bucket = 'penalty'), 0),
			COALESCE(sum(amount) FILTER (WHERE bucket = 'pending'), 0),
			COALESCE(sum(amount) FILTER (WHERE bucket = 'outside'), 0),
			-- الصافي مستقلّ عن الأبواب: كلّ ما ليس خارج الربح ولا معلّقاً.
			COALESCE(sum(amount) FILTER (WHERE bucket NOT IN ('outside','pending')), 0)
		FROM inr`, from, to).Scan(
		&p.orderLedger, &p.LostFailed, &p.LostCancelled, &p.LostRefunded,
		&p.Compensations, &p.Recovered, &p.Referrals, &p.Targets, &p.Opex,
		&p.Penalties, &p.Pending, &p.Outside, &p.Net); err != nil {
		return p, err
	}

	// ── الطلبات المسلّمة التي وقع ربحها في المدى: من أين جاء ──
	//
	// الهامش والعمولة والتوصيل والخصم من أعمدة الطلب، ونصيب المندوب من
	// دفتره. وما لا تشرحه الأعمدة يظهر سطر «فروق تسوية» — فالكشف يتجمّع
	// بالضبط إلى ما قيّده الدفتر.
	if err := s.pg.QueryRow(ctx, treasuryEntriesCTE+`,
		od AS (SELECT DISTINCT order_id FROM inr WHERE bucket = 'order'),
		rep AS (
			SELECT `+refUUID("w.ref")+` AS order_id, sum(w.amount) AS amt
			FROM wallet_transactions w
			WHERE w.kind = 'commission'
			GROUP BY 1
		)
		SELECT count(*),
		       COALESCE(sum(o.total), 0),
		       COALESCE(sum(`+orders.OrderMarginSQL("o.id")+`), 0),
		       COALESCE(sum(o.platform_commission), 0),
		       COALESCE(sum(o.delivery_fee + o.promo_delivery_waived - o.driver_fee), 0),
		       COALESCE(sum(o.discount), 0),
		       COALESCE(sum(rep.amt), 0),
		       COALESCE(sum(o.promo_delivery_waived), 0),
		       COALESCE(sum(it.platform_cut), 0),
		       COALESCE(sum(it.store_cut), 0)
		FROM od
		JOIN orders o ON o.id = od.order_id
		LEFT JOIN rep ON rep.order_id = o.id
		-- **وخصمُ الصنف من البند** (قسمُ العروض ٢٠٢٦-١٠-٠٤): ما تحمّلته المنصّةُ نقص
		-- هامشَها فيُعاد إليه ويُكتب سطراً لحاله، وما تحمّله المتجرُ لا يمسّ ربحَها.
		LEFT JOIN LATERAL (
			SELECT COALESCE(sum(oi.offer_cut * oi.qty) FILTER (WHERE oi.offer_borne_by = 'platform'), 0) AS platform_cut,
			       COALESCE(sum(oi.offer_cut * oi.qty) FILTER (WHERE oi.offer_borne_by = 'merchant'), 0) AS store_cut
			  FROM order_items oi WHERE oi.order_id = o.id) it ON true`, from, to).Scan(
		&p.Orders, &p.Sales, &p.Margin, &p.Commission, &p.DeliveryShare,
		&p.Discount, &p.RepShare, &p.FreeDelivery, &p.ItemDiscounts, &p.StoreItemDiscounts); err != nil {
		return p, err
	}
	// الهامشُ قبل خصم الصنف — والخصمُ سطرُه.
	p.Margin += p.ItemDiscounts
	return p, nil
}

// platformProfitSum أرقام الكشف. كلّ ما في الدفتر بأثره على الربح (سالب =
// خرج من الربح)، والهامش والعمولة والخصم ونصيب المندوب مبالغ موجبة.
type platformProfitSum struct {
	Orders int
	Sales  int64 // قيمة الطلبات المسلّمة مع التوصيل (total)

	Margin        int64
	Commission    int64
	DeliveryShare int64 // delivery_fee − driver_fee
	Discount      int64
	RepShare      int64 // ما قُيّد للمندوبين عن هذه الطلبات
	// **كلفةُ العروض سطورٌ لحالها** (قسمُ العروض ٢٠٢٦-١٠-٠٤): التوصيلُ المجانيّ
	// (`orders.promo_delivery_waived`) · وخصمُ الصنف على حساب المنصّة
	// (`order_items.offer_cut × qty`، `offer_borne_by = 'platform'`). والهامشُ
	// والتوصيلُ قبلهما، فالمجموعُ هو نفسُه. وخصمُ المتجر للعرض لا يدخل الكشف.
	FreeDelivery       int64
	ItemDiscounts      int64
	StoreItemDiscounts int64

	orderLedger int64 // ما قيّدته الخزينة عن الطلبات المسلّمة (بعد نصيب المندوب)

	LostFailed    int64
	LostCancelled int64
	LostRefunded  int64
	Compensations int64
	Recovered     int64
	Referrals     int64
	Targets       int64
	Opex          int64
	Penalties     int64

	Pending int64 // ربح معلّق — خارج الصافي
	Outside int64 // سحب وإيداع وحركات الأدمن الشخصيّة — خارج الصافي
	Net     int64
}

// SettleDiff ما قيّده الدفتر عن الطلبات المسلّمة ولا تشرحه أعمدتها.
func (p platformProfitSum) SettleDiff() int64 {
	return p.orderLedger + p.RepShare - (p.Margin + p.Commission + p.DeliveryShare - p.Discount -
		p.FreeDelivery - p.ItemDiscounts)
}

// Income دخل الطلبات المسلّمة قبل نصيب المندوب.
func (p platformProfitSum) Income() int64 { return p.orderLedger + p.RepShare }

// Lost خسارة الطلبات الفاشلة والملغاة والمستردّة (بأثرها: سالبة عادةً).
func (p platformProfitSum) Lost() int64 { return p.LostFailed + p.LostCancelled + p.LostRefunded }

// Losses الخسائر والتعويضات رقماً موجباً — لرئيسيّة الإدارة.
func (p platformProfitSum) Losses() int64 { return -(p.Lost() + p.Compensations) }

// Lines سطور الكشف بترتيبها، بأثرها على الربح.
func (p platformProfitSum) Lines() []profitLine {
	return []profitLine{
		{"margin", p.Margin},
		{"commission", p.Commission},
		{"delivery_share", p.DeliveryShare},
		{"discount", -p.Discount},
		{"free_delivery", -p.FreeDelivery},
		{"item_discounts", -p.ItemDiscounts},
		{"settle_diff", p.SettleDiff()},
		{"rep_share", -p.RepShare},
		{"lost_failed", p.LostFailed},
		{"lost_cancelled", p.LostCancelled},
		{"lost_refunded", p.LostRefunded},
		{"compensations", p.Compensations},
		{"recovered", p.Recovered},
		{"referrals", p.Referrals},
		{"targets", p.Targets},
		{"opex", p.Opex},
		{"penalties", p.Penalties},
	}
}

// LinesSum مجموع السطور — يساوي الصافي ما لم يظهر نوع قيد لا يعرفه الكشف.
func (p platformProfitSum) LinesSum() int64 {
	var n int64
	for _, l := range p.Lines() {
		n += l.Amount
	}
	return n
}

// ══════════════════════════════════════════════════════════════════════
//
//	٢ و٣ و٤ · الزبائن والمندوبون والسائقون
//
// ══════════════════════════════════════════════════════════════════════
//
// المصدر الدفتر، وكلّ عمود نوع قيد واحد — العمولة لحالها والمكافأة لحالها.
// والمدى على يوم القيد بدمشق. ومن كلّ أعمدته صفر لا يُعرض.

// partyCol عمود في تبويب شخص: مفتاحه (يترجمه معجم الشاشة) وتعبيره في SQL.
type partyCol struct {
	Key string
	SQL string
}

// partyTab أعمدة كلّ تبويب ودوره.
func partyTab(tab string) (role string, cols []partyCol, notRoles string) {
	sumKind := func(cond string) string {
		return `COALESCE(sum(t.amount) FILTER (WHERE ` + cond + `), 0)`
	}
	penalty := partyCol{"penalty", `COALESCE(sum(-t.amount) FILTER (WHERE t.kind = 'penalty'), 0)`}
	switch tab {
	case "customers":
		// المندوب والسائق لهما تبويبهما — فلا يظهران هنا لأنّ معهما دور زبون.
		return "customer", []partyCol{
			{"referral", sumKind(`t.kind = 'reward' AND t.ref <> ''`)},
			{"bonus", sumKind(`t.kind = 'reward' AND t.ref = ''`)},
			{"compensation", sumKind(`t.kind = 'compensation'`)},
			penalty,
		}, `'sales','driver'`
	case "reps":
		return "sales", []partyCol{
			{"commission", sumKind(`t.kind = 'commission'`)},
			{"bonus", sumKind(`t.kind = 'reward'`)},
			penalty,
		}, ``
	default:
		return "driver", []partyCol{
			{"delivery", sumKind(`t.kind = 'driver_earning'`)},
			{"compensation", sumKind(`t.kind = 'compensation'`)},
			{"bonus", sumKind(`t.kind = 'reward'`)},
			penalty,
		}, ``
	}
}

// partyRow سطر شخص — قيمه بترتيب columns في الردّ.
type partyRow struct {
	UserID string  `json:"user_id"`
	Name   string  `json:"name"`
	Phone  string  `json:"phone"`
	Values []int64 `json:"values"`
}

func (s *Server) profitsParties(w http.ResponseWriter, r *http.Request, from, to, tab string) {
	pg := pagingOf(r, 25)
	role, cols, notRoles := partyTab(tab)

	keys := make([]string, 0, len(cols)+1)
	sel := make([]string, 0, len(cols))
	nonZero := make([]string, 0, len(cols)+1)
	for i, c := range cols {
		keys = append(keys, c.Key)
		sel = append(sel, c.SQL+` AS c`+itoa(i))
		nonZero = append(nonZero, `COALESCE(e.c`+itoa(i)+`, 0) <> 0`)
	}
	// والزبون معه «أنفق لدينا» — قيمة طلباته المسلّمة بيوم تسليمها.
	spentSel := `0::bigint`
	if tab == "customers" {
		keys = append(keys, "spent")
		spentSel = `COALESCE(sp.paid, 0)`
		nonZero = append(nonZero, `COALESCE(sp.paid, 0) <> 0`)
	}

	ctes := `
		WITH e AS (
			SELECT t.user_id, ` + strings.Join(sel, ", ") + `
			FROM wallet_transactions t
			WHERE ` + profitRange("t.created_at") + `
			GROUP BY t.user_id
		), sp AS (
			SELECT o.customer_id AS uid, COALESCE(sum(o.total), 0) AS paid
			FROM orders o
			WHERE o.status = 'delivered' AND ` + profitRange("o.delivered_at") + `
			GROUP BY o.customer_id
		)`
	exclude := ``
	if notRoles != "" {
		exclude = ` AND NOT EXISTS (SELECT 1 FROM user_roles x
			WHERE x.user_id = u.id AND x.role_code IN (` + notRoles + `))`
	}
	body := `
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = '` + role + `'
		LEFT JOIN e ON e.user_id = u.id
		LEFT JOIN sp ON sp.uid = u.id
		WHERE u.deleted_at IS NULL` + exclude + `
		  AND (` + strings.Join(nonZero, " OR ") + `)`

	var count int
	if err := s.pg.QueryRow(r.Context(), ctes+` SELECT count(*)`+body, from, to).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}
	vals := make([]string, 0, len(cols)+1)
	for i := range cols {
		vals = append(vals, `COALESCE(e.c`+itoa(i)+`, 0)`)
	}
	vals = append(vals, spentSel)
	rows, err := s.pg.Query(r.Context(), ctes+`
		SELECT u.id::text, COALESCE(u.full_name, ''), u.phone::text,
		       ARRAY[`+strings.Join(vals, ", ")+`]::bigint[]`+body+`
		ORDER BY COALESCE(e.c0, 0) DESC, `+spentSel+` DESC, u.id
		LIMIT $3 OFFSET $4`, from, to, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []partyRow{}
	for rows.Next() {
		var x partyRow
		if err := rows.Scan(&x.UserID, &x.Name, &x.Phone, &x.Values); err != nil {
			s.respondErr(w, err)
			return
		}
		x.Values = x.Values[:len(keys)]
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	res := paged("rows", out, count, pg)
	res["columns"] = keys
	httpx.JSON(w, http.StatusOK, res)
}

// ══════════════════════════════════════════════════════════════════════
//
//	٥ · المتاجر
//
// ══════════════════════════════════════════════════════════════════════
//
// والسطر للمتجر لا لصاحبه. وما كسبه المتجر يُقرأ من الدفتر (مستحقّه في
// المحفظة أو الاحتباس النقديّ) لا يُحسب من الطلبات — فهو يطابق محفظته، ومعه
// ما قُبض عن طلبات فشلت بعد الاستلام. وهامشنا وعمولتنا ومبيعات أصنافه من
// الطلبات المسلّمة بيوم تسليمها.
func (s *Server) profitsMerchants(w http.ResponseWriter, r *http.Request, from, to string) {
	pg := pagingOf(r, 25)
	ctes := `
		WITH led AS (
			SELECT t.user_id, t.kind, t.amount, o.id AS order_id, o.merchant_id AS order_merchant
			FROM wallet_transactions t
			JOIN orders o ON o.id = ` + refUUID("t.ref") + `
			WHERE t.kind IN ('merchant_earning','merchant_cash_accrued','compensation')
			  AND ` + profitRange("t.created_at") + `
		), led_m AS (
			SELECT COALESCE(
			         (SELECT m.id FROM merchants m WHERE m.id = led.order_merchant AND m.owner_user_id = led.user_id),
			         (SELECT oi.merchant_id FROM order_items oi JOIN merchants m ON m.id = oi.merchant_id
			          WHERE oi.order_id = led.order_id AND m.owner_user_id = led.user_id
			          ORDER BY oi.merchant_id LIMIT 1)) AS merchant_id,
			       led.kind, led.amount
			FROM led
		), earned AS (
			SELECT merchant_id,
			       COALESCE(sum(amount) FILTER (WHERE kind IN ('merchant_earning','merchant_cash_accrued')), 0) AS got,
			       COALESCE(sum(amount) FILTER (WHERE kind = 'compensation'), 0) AS comp
			FROM led_m WHERE merchant_id IS NOT NULL
			GROUP BY merchant_id
		), os AS (
			SELECT o.id AS order_id, o.merchant_id AS order_merchant, o.platform_commission,
			       COALESCE(oi.merchant_id, o.merchant_id) AS merchant_id,
			       sum(oi.unit_price * oi.qty) AS gross,
			       sum((oi.unit_price - oi.merchant_price) * oi.qty) AS margin,
			       sum(oi.merchant_price * oi.qty) AS cost
			FROM orders o JOIN order_items oi ON oi.order_id = o.id
			WHERE o.status = 'delivered' AND ` + profitRange("o.delivered_at") + `
			GROUP BY o.id, o.merchant_id, o.platform_commission, COALESCE(oi.merchant_id, o.merchant_id)
		), took AS (
			SELECT os.merchant_id,
			       sum(os.gross) AS gross, sum(os.margin) AS margin,
			       sum(CASE WHEN ms.amount IS NOT NULL THEN os.cost - ms.amount
			                WHEN os.order_merchant = os.merchant_id THEN os.platform_commission
			                ELSE 0 END) AS commission
			FROM os
			LEFT JOIN merchant_settlements ms ON ms.order_id = os.order_id AND ms.merchant_id = os.merchant_id
			GROUP BY os.merchant_id
		)`
	body := `
		FROM merchants m
		LEFT JOIN earned e ON e.merchant_id = m.id
		LEFT JOIN took k ON k.merchant_id = m.id
		LEFT JOIN users u ON u.id = m.owner_user_id
		WHERE COALESCE(k.gross, 0) <> 0 OR COALESCE(k.margin, 0) <> 0 OR COALESCE(k.commission, 0) <> 0
		   OR COALESCE(e.got, 0) <> 0 OR COALESCE(e.comp, 0) <> 0`

	var count int
	if err := s.pg.QueryRow(r.Context(), ctes+` SELECT count(*)`+body, from, to).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(r.Context(), ctes+`
		SELECT m.id::text, m.name, COALESCE(u.phone::text, ''),
		       ARRAY[COALESCE(k.margin, 0), COALESCE(k.commission, 0), COALESCE(k.gross, 0),
		             COALESCE(e.got, 0), COALESCE(e.comp, 0)]::bigint[]`+body+`
		ORDER BY COALESCE(k.gross, 0) DESC, COALESCE(e.got, 0) DESC, m.name
		LIMIT $3 OFFSET $4`, from, to, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []partyRow{}
	for rows.Next() {
		var x partyRow
		if err := rows.Scan(&x.UserID, &x.Name, &x.Phone, &x.Values); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	res := paged("rows", out, count, pg)
	res["columns"] = []string{"our_margin", "our_commission", "gross", "store_earned", "compensation"}
	httpx.JSON(w, http.StatusOK, res)
}
