package orders

// ══════════════════════════════════════════════════════════════════════
// **طلبُ الجوائز** — من متجر «رحّال غو» برصيد الجوائز (قرارُ المالك ٢٠٢٦-١١-٢٣)
// ══════════════════════════════════════════════════════════════════════
//
// **يُعرف من أصنافه لا من حقلٍ يُرسَل**: صنفٌ من متجرٍ حالتُه `prize` ⇒ طلبُ جوائز.
// والقواعد:
//
//	لا خلطَ           ←  أصنافُ متجر الجوائز وحدَها (`prize_mixed`)
//	رصيدٌ لازم        ←  رصيدُ الجوائز فوق الصفر (`prize_no_balance`)
//	الرصيدُ أوّلاً     ←  `prize_discount = min(الرصيد، البضاعة)` — والباقي نقداً أو محفظة
//	التوصيلُ على المنصّة ←  يُعفى الزبونُ كلَّه والكابتنُ بأجره كاملاً (`promo_delivery_waived`)
//	لا كود            ←  لا يجتمع كودُ خصمٍ مع رصيد الجوائز (`prize_no_promo`)
//
// **والرصيدُ خصمٌ تتحمّله المنصّة** — داخلَ `discount` ككود الخصم، فحسابُ الخزينة الفرقيُّ
// يقرؤه بلا قيدٍ جديد. **والصرفُ داخل معاملة الإنشاء بقفل صفّ الرصيد** — طلبان متزامنان
// لا يصرفان رصيداً واحداً مرّتين.

import (
	"context"
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	// ErrPrizeMixed **أصنافُ الجوائز لا تُخلط بغيرها.**
	ErrPrizeMixed = httpx.NewError(http.StatusConflict, "prize_mixed", "errors.prize_mixed")
	// ErrPrizeNoBalance **لا رصيدَ جوائز.**
	ErrPrizeNoBalance = httpx.NewError(http.StatusConflict, "prize_no_balance", "errors.prize_no_balance")
	// ErrPrizeNoPromo **ولا كودَ خصمٍ مع رصيد الجوائز.**
	ErrPrizeNoPromo = httpx.NewError(http.StatusConflict, "prize_no_promo", "errors.prize_no_promo")
	// ErrPrizeOrderLocked **طلبُ الجوائز لا يُحوَّل لمتجرٍ آخر.**
	ErrPrizeOrderLocked = httpx.NewError(http.StatusConflict, "prize_order_locked", "errors.prize_order_locked")
)

// PrizeOpenNowSQL **`OpenNowSQL` لمتجر الجوائز** — دوامُه وإغلاقُه الطارئ كأيّ متجر، والحالةُ `prize`.
var PrizeOpenNowSQL = strings.Replace(OpenNowSQL, "m.status = 'active'", "m.status = 'prize'", 1)

// OrderableNowSQL **أيُطلب منه الآن** — متجرٌ نشطٌ أو متجرُ الجوائز، بدوامه.
var OrderableNowSQL = "(" + OpenNowSQL + " OR " + PrizeOpenNowSQL + ")"

// PrizeLedger **رصيدُ الجوائز كما يحتاجه الطلب** — `prizes.Service`.
type PrizeLedger interface {
	LockedBalanceTx(ctx context.Context, q dbtx.Querier, userID string) (int64, error)
	SpendTx(ctx context.Context, q dbtx.Querier, userID, orderID string, amount int64) error
	RefundOrderTx(ctx context.Context, q dbtx.Querier, orderID string, keep int64) (int64, error)
}

// SetPrizes يربط رصيدَ الجوائز — مرّةً عند الإقلاع.
func (s *Service) SetPrizes(p PrizeLedger) { s.prizes = p }

// isPrizeCart **أفي المصادر متجرُ الجوائز؟** — ويُرفض خلطُه بغيره.
func isPrizeCart(ctx context.Context, q dbtx.Querier, ids []string) (bool, error) {
	if len(ids) == 0 {
		return false, nil
	}
	var prize int
	if err := q.QueryRow(ctx, `
		SELECT count(*) FROM merchants WHERE id = ANY($1::uuid[]) AND status = 'prize'`,
		ids).Scan(&prize); err != nil {
		return false, err
	}
	if prize == 0 {
		return false, nil
	}
	if len(ids) > 1 {
		return true, ErrPrizeMixed
	}
	return true, nil
}

// IsPrizeCart **سلّةٌ من متجر الجوائز؟** — للتسعيرة قبل الطلب.
func (s *Service) IsPrizeCart(ctx context.Context, items []ItemInput) (bool, error) {
	src, err := s.SourcesOf(ctx, s.db, items)
	if err != nil {
		return false, err
	}
	return isPrizeCart(ctx, s.db, src.IDs)
}

// prizeRefit **حصّةُ الجوائز بعد إعادة حساب البضاعة** — `min(الأصل، البضاعة الجديدة)`،
// ويُردّ الفرقُ إلى رصيد الجوائز ويُكتب العمودُ قبل الخصم (قيدُ `prize_discount <= discount`).
// يردّ الحصّةَ الباقية — وصفراً لطلبٍ بلا جوائز.
func (s *Service) prizeRefit(ctx context.Context, q dbtx.Querier, orderID string, subtotal int64) (int64, error) {
	var cur int64
	if err := q.QueryRow(ctx, `SELECT prize_discount FROM orders WHERE id = $1`, orderID).Scan(&cur); err != nil {
		return 0, err
	}
	if cur == 0 {
		return 0, nil
	}
	keep := cur
	if subtotal < keep {
		keep = subtotal
	}
	if keep < 0 {
		keep = 0
	}
	if keep == cur {
		return cur, nil
	}
	if s.prizes != nil {
		if _, err := s.prizes.RefundOrderTx(ctx, q, orderID, keep); err != nil {
			return 0, err
		}
	}
	if _, err := q.Exec(ctx, `UPDATE orders SET prize_discount = $2 WHERE id = $1`, orderID, keep); err != nil {
		return 0, err
	}
	return keep, nil
}

// prizeOrderOf **أطلبُ جوائز؟**
func prizeOrderOf(ctx context.Context, q dbtx.Querier, orderID string) (bool, error) {
	var p bool
	err := q.QueryRow(ctx, `SELECT prize_order FROM orders WHERE id = $1`, orderID).Scan(&p)
	return p, err
}
