package orders

// ══════════════════════════════════════════════════════════════════════
// **عمولةُ المندوب في طلبٍ بعدّة محطّات — كلُّ مندوبٍ من ربح متجره** (قرارُ المالك ٢٠٢٦-١١-٠٦، ٢أ)
// ══════════════════════════════════════════════════════════════════════
//
// «كلّ مندوب ياخذ عمولته من ربح المنصّة من متجره هو». **وكان مندوبُ المتجر الأوّل يأخذ
// على ربح الطلب كلّه** — ومندوبُ المتجر الثاني لا شيء.
//
// لكلّ متجرٍ حيٍّ `s` في الطلب (بنودُه باقية):
//
//	pc_s    عمولةُ المنصّة منه — من حصّته كما تُسوّى (`merchantShares`، ونسبتُه المجمّدة)
//	m_s     هامشُ بنوده — `(سعرُ البيع − سعرُ الشراء) × الكمّيّة`
//	promo_s **نصيبُه من الكوبون بقيمة بضاعته** — والباقي من القسمة لأكبر متجر
//	base_s  `RepCommissionBase(الوضع, pc_s, m_s, promo_s)` — **والصفرُ أرضٌ لكلّ متجر**
//	R_s     نسبةُ المندوب من لقطة الطلب × base_s
//
// **ثمّ لكلّ مندوب**: سطرُ عمولةٍ واحدٌ بمجموع متاجره · اقتطاعُ دَينه · أو حجزُها إن كان
// موقوفاً · **و`order_parts.rep_id/rep_commission` تفصيلٌ يحرسه `FI-09.b`.**
//
// **والأرضُ لكلّ متجرٍ قد تدفع أكثرَ قليلاً من أرضٍ على الطلب كلّه** (كوبونٌ أكبرُ من ربح
// متجرٍ واحد) **ولا تدفع أقلَّ أبداً** — وهو ما وافق عليه المالك.
//
// **وطلبُ المتجر الواحد لا يمرّ من هنا** — `settleRep` على مساره اليوم حرفاً.

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/pricing"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// repPaid **عمولةٌ قُيّدت لمندوب** — تُبلَّغ بعد الإيداع (`notifyCommission`).
type repPaid struct {
	repID  string
	amount int64
	// store اسمُ متجره في طلبٍ بعدّة محطّات — وفارغُه متجرُ الطلب.
	store string
}

// stopsAbove1 **أطلبٌ بعدّة محطّات؟** (`orders.stops > 1`).
func stopsAbove1(ctx context.Context, q dbtx.Querier, orderID string) (bool, error) {
	var n int
	if err := q.QueryRow(ctx, `SELECT stops FROM orders WHERE id = $1`, orderID).Scan(&n); err != nil {
		return false, err
	}
	return n > 1, nil
}

// repStoreShare **نصيبُ مندوبِ متجرٍ واحدٍ في الطلب** (R_s) — ومندوبُه كما هو اليوم وكما قُيّد.
type repStoreShare struct {
	merchantID, name string
	// repID مندوبُ المتجر اليوم (`merchants.sales_rep_user_id`) · partRep من قُيّد له عند التسليم.
	repID, partRep *string
	repIsBuyer     bool
	amount         int64
}

// repPromoCostSQL **كلفةُ الكود على المنصّة** — المعادلةُ نفسُها في `settleRep` و`repShareToReverse`
// (خصمُه وإعفاءُ توصيله، **وعرضُ التوصيل المجّانيّ لا يُطرح** — قرارُ المالك ٢٠٢٦-١٠-٢٦).
const repPromoCostSQL = `
	SELECT o.discount + o.promo_delivery_waived
	         - COALESCE((SELECT c.waived FROM free_delivery_claims c WHERE c.order_id = o.id), 0)
	FROM orders o WHERE o.id = $1`

// repStoreShares **R_s لكلّ متجرٍ حيٍّ في الطلب** — بترتيب المحطّات.
//
// `zeroCommission` **العمولةُ صفرٌ لكلّ متجر** — ما يبقى للمندوب بعد محو العمولة
// (`commission_waiver.go`). **ويُحسب المبلغُ لكلّ متجرٍ ولو بلا مندوب** — والمستدعي يتخطّى.
func (s *Service) repStoreShares(ctx context.Context, q wallet.Querier, orderID string,
	snap EconomicsSnapshot, zeroCommission bool) ([]repStoreShare, error) {
	var promoCost int64
	if err := q.QueryRow(ctx, repPromoCostSQL, orderID).Scan(&promoCost); err != nil {
		return nil, err
	}
	pcOf := map[string]int64{}
	if !zeroCommission {
		ms, _, err := s.merchantShares(ctx, q, orderID)
		if err != nil {
			return nil, err
		}
		for _, sh := range ms {
			pcOf[sh.merchantID] += sh.commission
		}
	}

	rows, err := q.Query(ctx, `
		SELECT m.id::text, m.name, m.sales_rep_user_id::text,
		       COALESCE(m.sales_rep_user_id = o.customer_id, false),
		       p.rep_id::text,
		       sum(oi.unit_price * oi.qty)::bigint,
		       sum((oi.unit_price - oi.merchant_price) * oi.qty)::bigint,
		       COALESCE(p.seq, 32767)
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN merchants m ON m.id = COALESCE(oi.merchant_id, o.merchant_id)
		LEFT JOIN order_parts p ON p.order_id = o.id AND p.merchant_id = m.id
		WHERE oi.order_id = $1
		GROUP BY m.id, m.name, m.sales_rep_user_id, o.customer_id, p.rep_id, p.seq
		ORDER BY 8, 1`, orderID)
	if err != nil {
		return nil, err
	}
	type line struct {
		sh           repStoreShare
		sale, margin int64
	}
	var lines []line
	var saleTotal int64
	largest := -1
	for rows.Next() {
		var l line
		var seq int
		if err := rows.Scan(&l.sh.merchantID, &l.sh.name, &l.sh.repID, &l.sh.repIsBuyer, &l.sh.partRep,
			&l.sale, &l.margin, &seq); err != nil {
			rows.Close()
			return nil, err
		}
		lines = append(lines, l)
		saleTotal += l.sale
		if largest < 0 || l.sale > lines[largest].sale {
			largest = len(lines) - 1
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// **الكوبونُ بقيمة البضاعة** — والباقي من القسمة لأكبر متجر (وأوّلُهما إن تساويا).
	promo := make([]int64, len(lines))
	if promoCost > 0 && len(lines) > 0 {
		var given int64
		for i, l := range lines {
			if saleTotal > 0 {
				promo[i] = promoCost * l.sale / saleTotal
			}
			given += promo[i]
		}
		promo[largest] += promoCost - given
	}

	src := pricing.CommissionSource(snap.CommissionSource)
	pct := pricing.Amount{Mode: "percent", Value: snap.RepCommissionPercent}
	out := make([]repStoreShare, 0, len(lines))
	for i, l := range lines {
		base := pricing.RepCommissionBase(src, pcOf[l.sh.merchantID], l.margin, promo[i])
		l.sh.amount = pct.Of(base)
		out = append(out, l.sh)
	}
	return out, nil
}

// settleRepParts **عمولةُ كلِّ مندوبٍ من متاجره في الطلب** — عند التسليم، بقواعد `settleRep`:
// لا مندوبَ ⇒ لا مستحِقّ · المندوبُ لا يأخذ على شرائه · عتبةُ التفعيل · الموقوفُ يُحجز.
func (s *Service) settleRepParts(ctx context.Context, q wallet.Querier, orderID, actorID string, out *settled) error {
	snap, err := s.snapshotOf(ctx, q, orderID)
	if err != nil {
		return err
	}
	activated, err := s.merchantActivated(ctx, q, orderID, snap.ActivationOrders)
	if err != nil || !activated {
		return err
	}
	shares, err := s.repStoreShares(ctx, q, orderID, snap, false)
	if err != nil {
		return err
	}

	// **لكلّ مندوبٍ مجموعُ متاجره** — بترتيب أوّل محطّةٍ له.
	var reps []*repPaid
	byRep := map[string]*repPaid{}
	for _, sh := range shares {
		var rep *string
		amount := sh.amount
		if sh.repID != nil && !sh.repIsBuyer && amount > 0 {
			rep = sh.repID
		} else {
			amount = 0
		}
		// **التفصيلُ لكلّ متجر** — للشاشة ولقسمة نصيب المندوب عند الاسترجاع (`FI-09.b`).
		if _, err := q.Exec(ctx, `
			UPDATE order_parts SET rep_id = $3::uuid, rep_commission = $4, updated_at = now()
			 WHERE order_id = $1 AND merchant_id = $2`, orderID, sh.merchantID, rep, amount); err != nil {
			return err
		}
		if rep == nil {
			continue
		}
		r := byRep[*rep]
		if r == nil {
			r = &repPaid{repID: *rep, store: sh.name}
			byRep[*rep] = r
			reps = append(reps, r)
		}
		r.amount += amount
	}

	for _, r := range reps {
		// **والموقوفُ تُحجَز عمولتُه وحدَه** — صفٌّ لكلّ (طلب، مندوب)، الهجرة 0518.
		if held, err := s.holdRepCommissionIfInactive(ctx, q, r.repID, orderID, r.amount); err != nil {
			return err
		} else if held {
			continue
		}
		if _, err := s.wallet.ApplyTx(ctx, q, r.repID, r.amount, "commission",
			orderID, "عمولة عن طلب تم تسليمه", &actorID); err != nil {
			return err
		}
		if err := s.offsetRepDebt(ctx, q, r.repID, r.amount, orderID, actorID); err != nil {
			return err
		}
		out.reps = append(out.reps, *r)
	}
	return nil
}

// repReverse **ما يُعكَس من نصيب مندوبٍ عند محو العمولة.**
type repReverse struct {
	repID  string
	amount int64
}

// repSharesToReverse **لكلّ مندوبٍ قبض عن الطلب: كم يُعكَس من نصيبه** — واحدٌ لطلب المتجر
// الواحد (`repShareToReverse` كما كان)، **ولكلّ مندوبٍ من متاجره هو** في طلبٍ بعدّة محطّات.
func (s *Service) repSharesToReverse(ctx context.Context, q wallet.Querier,
	orderID string, commission int64) ([]repReverse, error) {
	multi, err := stopsAbove1(ctx, q, orderID)
	if err != nil {
		return nil, err
	}
	if !multi {
		repID, rev, err := s.repShareToReverse(ctx, q, orderID, commission)
		if err != nil || repID == "" {
			return nil, err
		}
		return []repReverse{{repID: repID, amount: rev}}, nil
	}

	// **ما بقي لكلّ مندوبٍ بعد المحو** — من متاجره التي قُيّد له عنها (`order_parts.rep_id`)،
	// بالقاعدة نفسِها والعمولةُ صفر. **وبلا لقطةٍ: صفر** (يُعكَس كلُّه).
	remain := map[string]int64{}
	var order []string
	snap, err := s.snapshotOf(ctx, q, orderID)
	var shares []repStoreShare
	switch {
	case errors.Is(err, ErrNoSnapshot):
		if shares, err = s.repStoreSharesNoAmount(ctx, q, orderID); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if shares, err = s.repStoreShares(ctx, q, orderID, snap, true); err != nil {
			return nil, err
		}
	}
	for _, sh := range shares {
		if sh.partRep == nil {
			continue
		}
		if _, ok := remain[*sh.partRep]; !ok {
			order = append(order, *sh.partRep)
		}
		remain[*sh.partRep] += sh.amount
	}

	var out []repReverse
	for _, rep := range order {
		// **المحجوزُ القائمُ لهذا المندوب** — يُنقَص في مكانه، ولا قيدَ في محفظته.
		var heldID string
		var heldAmount int64
		err := q.QueryRow(ctx, `
			SELECT id::text, amount FROM rep_held_commissions
			 WHERE order_id = $1 AND rep_id = $2 AND status = 'held' FOR UPDATE`, orderID, rep).
			Scan(&heldID, &heldAmount)
		if err == nil {
			if cut := heldAmount - remain[rep]; cut > 0 {
				if heldAmount-cut <= 0 {
					_, err = q.Exec(ctx, `
						UPDATE rep_held_commissions SET status = 'forfeited', decided_at = now() WHERE id = $1`, heldID)
				} else {
					_, err = q.Exec(ctx, `UPDATE rep_held_commissions SET amount = amount - $2 WHERE id = $1`, heldID, cut)
				}
				if err != nil {
					return nil, err
				}
			}
			out = append(out, repReverse{repID: rep})
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		var paid int64
		if err := q.QueryRow(ctx, `
			SELECT COALESCE(sum(amount), 0)::bigint FROM wallet_transactions
			 WHERE ref = $1 AND kind = 'commission' AND user_id = $2 AND amount > 0`,
			orderID, rep).Scan(&paid); err != nil {
			return nil, err
		}
		out = append(out, repReverse{repID: rep, amount: max(paid-remain[rep], 0)})
	}
	return out, nil
}

// repStoreSharesNoAmount **متاجرُ الطلب ومن قُيّد له عنها — بلا مبلغ** (طلبٌ بلا لقطة).
func (s *Service) repStoreSharesNoAmount(ctx context.Context, q wallet.Querier, orderID string) ([]repStoreShare, error) {
	rows, err := q.Query(ctx, `
		SELECT merchant_id::text, rep_id::text FROM order_parts
		 WHERE order_id = $1 AND rep_id IS NOT NULL ORDER BY seq`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []repStoreShare
	for rows.Next() {
		var sh repStoreShare
		if err := rows.Scan(&sh.merchantID, &sh.partRep); err != nil {
			return nil, err
		}
		out = append(out, sh)
	}
	return out, rows.Err()
}

// chargeRefundRepSharePartsTx **نصيبُ المندوبين على متاجرهم** — استرجاعٌ بعد التسليم والمكتبُ
// اختار المتجر، في طلبٍ بعدّة محطّات (الخطّة §7.2، قرارُ المالك ٢٠٢٦-١١-٠٦).
//
// **من الدفتر لا من النسبة**: لكلّ مندوبٍ ما بقي له فعلاً (`kept_r` = مجموعُ سطور `commission`
// باسمه وبمرجع الطلب، **بعد اقتطاع دَينه**، وصفرٌ لمحجوزٍ لم يُصرف)، **ويُقسَم على متاجره بنسبة
// `R_s` إلى `R_r`** — والباقي من القسمة لأكبرها. **وما بقي بلا تفصيلٍ** (طلبٌ سُوّي قبل هذا)
// **على متجر الطلب كما كان.**
func chargeRefundRepSharePartsTx(ctx context.Context, q dbtx.Querier, orderID string, actor *string) (int64, error) {
	kept := map[string]int64{}
	var total int64
	rows, err := q.Query(ctx, `
		SELECT user_id::text, sum(amount)::bigint FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'commission' GROUP BY user_id`, orderID)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var u string
		var k int64
		if err := rows.Scan(&u, &k); err != nil {
			rows.Close()
			return 0, err
		}
		kept[u] = k
		total += k
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if total <= 0 {
		return 0, nil
	}

	type part struct {
		merchantID, repID string
		r                 int64
	}
	var parts []part
	var repOrder []string
	repTotal := map[string]int64{}
	prows, err := q.Query(ctx, `
		SELECT merchant_id::text, rep_id::text, rep_commission FROM order_parts
		 WHERE order_id = $1 AND rep_id IS NOT NULL AND rep_commission > 0
		 ORDER BY rep_commission DESC, seq`, orderID)
	if err != nil {
		return 0, err
	}
	for prows.Next() {
		var p part
		if err := prows.Scan(&p.merchantID, &p.repID, &p.r); err != nil {
			prows.Close()
			return 0, err
		}
		parts = append(parts, p)
		if _, ok := repTotal[p.repID]; !ok {
			repOrder = append(repOrder, p.repID)
		}
		repTotal[p.repID] += p.r
	}
	prows.Close()
	if err := prows.Err(); err != nil {
		return 0, err
	}

	// **لكلّ مندوبٍ حصّةُ كلِّ متجرٍ من متاجره** — وأوّلُها (الأكبرُ) يأخذ باقيَ القسمة.
	charge := map[string]int64{}
	var storeOrder []string
	var given int64
	for _, rep := range repOrder {
		rr, k := repTotal[rep], kept[rep]
		if k <= 0 {
			continue
		}
		var sum int64
		first := ""
		for _, p := range parts {
			if p.repID != rep {
				continue
			}
			a := k * p.r / rr
			if first == "" {
				first = p.merchantID
			}
			if _, ok := charge[p.merchantID]; !ok {
				storeOrder = append(storeOrder, p.merchantID)
			}
			charge[p.merchantID] += a
			sum += a
		}
		charge[first] += k - sum
		given += k
	}

	// **وما لا تفصيلَ له على متجر الطلب** — بالمرجع القديم (مرجعُ الطلب وحدَه).
	if rest := total - given; rest > 0 {
		var primary *string
		if err := q.QueryRow(ctx, `
			SELECT COALESCE(merchant_id, partner_merchant_id)::text FROM orders WHERE id = $1`, orderID).
			Scan(&primary); err != nil {
			return 0, err
		}
		if primary == nil || *primary == "" {
			return 0, ErrRepShareNoParty
		}
		if _, err := obligations.CreateRef(ctx, q, obligations.PartyMerchant, *primary, rest,
			obligations.CauseRefundRepShare, orderID, orderID, actor); err != nil {
			return 0, err
		}
	}
	for _, mid := range storeOrder {
		a := charge[mid]
		if a <= 0 {
			continue
		}
		// **مرجعٌ لكلّ (طلب، متجر)** — فريدُ (السبب، المرجع) يمنع دينين من مصدرٍ واحد.
		if _, err := obligations.CreateRef(ctx, q, obligations.PartyMerchant, mid, a,
			obligations.CauseRefundRepShare, orderID, fmt.Sprintf("%s:%s", orderID, mid), actor); err != nil {
			return 0, err
		}
	}
	return total, nil
}
