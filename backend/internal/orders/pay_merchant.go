package orders

// ══════════════════════════════════════════════════════════════════════
// **كم يدفع السائقُ لكلّ متجرٍ عند الاستلام** — قرارُ المالك ٢٠٢٦-١٠-١٥
// ══════════════════════════════════════════════════════════════════════
//
// السائقُ يدفع للمتجر «نقداً» صافيه عند الاستلام (`payCashByDriver`). **فيلزمه أن
// يعرف الرقمَ قبل أن يصل** — والرقمُ من حسبة التسوية نفسِها (`merchantShares`)
// لا من حسبةٍ ثانيةٍ في الشاشة:
//
//	قبل الاستلام  ←  سعرُ الشراء − العمولة − دَينُ المتجر القائم (تقديرٌ بدَين اليوم)
//	بعد الاستلام  ←  ما دفعه فعلاً (القائمُ في صفّ التسوية)
//
// **والمتجرُ «محفظة» لا يُدفع له شيء** — مستحقُّه في محفظته. **والتوصيلةُ خارجَ هذا**،
// **والطلبُ الخاصُّ داخلَه إن رُبط بمتجرٍ تابع** (قرارُ المالك ٢٠٢٦-١٠-١٥): الثمنُ ناقصَ العمولة.

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/obligations"
)

// StorePay **ما يدفعه السائقُ لمتجرٍ واحدٍ عند الاستلام.**
type StorePay struct {
	MerchantID   string `json:"merchant_id"`
	MerchantName string `json:"merchant_name"`
	Amount       int64  `json:"amount"`
}

// PayMerchants **ما يدفعه السائقُ لكلّ متجرٍ في كلّ طلب** — والطلبُ بلا دفعٍ غائبٌ.
func (s *Service) PayMerchants(ctx context.Context, q dbtx.Querier, orderIDs []string) (map[string][]StorePay, error) {
	out := map[string][]StorePay{}
	for _, id := range orderIDs {
		list, err := s.payMerchantsFor(ctx, q, id)
		if err != nil {
			return nil, err
		}
		if len(list) > 0 {
			out[id] = list
		}
	}
	return out, nil
}

func (s *Service) payMerchantsFor(ctx context.Context, q dbtx.Querier, orderID string) ([]StorePay, error) {
	var kind string
	if err := q.QueryRow(ctx, `SELECT kind FROM orders WHERE id = $1`, orderID).Scan(&kind); err != nil {
		return nil, err
	}
	if kind == KindCustom {
		return s.payPartnerFor(ctx, q, orderID)
	}
	if kind != "standard" {
		return nil, nil
	}
	shares, _, err := s.merchantShares(ctx, q, orderID)
	if err != nil {
		return nil, err
	}
	var out []StorePay
	for _, sh := range shares {
		if sh.ownerID == nil || sh.due <= 0 {
			continue
		}
		p := StorePay{MerchantID: sh.merchantID}
		if err := q.QueryRow(ctx, `SELECT name FROM merchants WHERE id = $1`, sh.merchantID).
			Scan(&p.MerchantName); err != nil {
			return nil, err
		}
		// **صفُّ التسوية إن نشأ هو الحقيقة** — طريقتُه ومبلغُه وحالُه.
		var method, state string
		var amount, reversed int64
		err := q.QueryRow(ctx, `
			SELECT method, state, amount, reversed_amount FROM merchant_settlements
			 WHERE order_id = $1 AND merchant_id = $2`, orderID, sh.merchantID).
			Scan(&method, &state, &amount, &reversed)
		switch {
		case err == nil:
			if method != "cash" || state == "cash_reversed" {
				continue
			}
			p.Amount = amount - reversed
		case errors.Is(err, pgx.ErrNoRows):
			if sh.method != "cash" {
				continue
			}
			// **تقديرٌ بدَين اليوم** — والاقتطاعُ الفعليُّ لحظةَ الاستلام.
			debt, err := obligations.Balance(ctx, q, obligations.PartyMerchant, sh.merchantID)
			if err != nil {
				return nil, err
			}
			if debt < 0 {
				debt = 0
			}
			if debt > sh.due {
				debt = sh.due
			}
			p.Amount = sh.due - debt
		default:
			return nil, err
		}
		if p.Amount > 0 {
			out = append(out, p)
		}
	}
	return out, nil
}

// payPartnerFor **ما يدفعه السائقُ للمتجر التابع في المشتريات** (قرارُ المالك ٢٠٢٦-١٠-١٥) —
// الثمنُ ناقصَ العمولة (`customStoreCut`)، **بعد توثيق الثمن وحدَه**: قبله لا يُعرف رقم.
// **والمشترياتُ بلا متجرٍ تابعٍ غائبةٌ هنا** — يرى السائقُ «ثمنَ البضاعة» كما كان.
func (s *Service) payPartnerFor(ctx context.Context, q dbtx.Querier, orderID string) ([]StorePay, error) {
	var p StorePay
	var documented bool
	err := q.QueryRow(ctx, `
		SELECT o.partner_merchant_id::text, m.name,
		       o.custom_goods_amount IS NOT NULL AND NOT o.custom_goods_pending
		FROM orders o JOIN merchants m ON m.id = o.partner_merchant_id
		WHERE o.id = $1 AND o.custom_mode = 'purchase'`, orderID).
		Scan(&p.MerchantID, &p.MerchantName, &documented)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !documented {
		return nil, nil
	}
	if p.Amount, _, err = customStoreCut(ctx, q, orderID); err != nil {
		return nil, err
	}
	if p.Amount <= 0 {
		return nil, nil
	}
	return []StorePay{p}, nil
}

// fillPayMerchant **يملأ ما يدفعه السائقُ في الطلب** — للعاديّ، وللمشتريات من متجرٍ تابع.
func (s *Service) fillPayMerchant(ctx context.Context, q dbtx.Querier, o *Order) error {
	if o.Kind != "standard" && o.Kind != KindCustom {
		return nil
	}
	list, err := s.payMerchantsFor(ctx, q, o.ID)
	if err != nil {
		return err
	}
	o.PayMerchants = list
	o.PayMerchant = 0
	for _, p := range list {
		o.PayMerchant += p.Amount
	}
	return nil
}
