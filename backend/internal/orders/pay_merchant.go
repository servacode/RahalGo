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
// **والمتجرُ «محفظة» لا يُدفع له شيء** — مستحقُّه في محفظته. **والطلبُ الخاصُّ
// والتوصيلةُ خارجَ هذا** — لا مستحقَّ متجرٍ فيهما.

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

// fillPayMerchant **يملأ ما يدفعه السائقُ في الطلب** — للطلب العاديّ وحدَه.
func (s *Service) fillPayMerchant(ctx context.Context, q dbtx.Querier, o *Order) error {
	if o.Kind != "standard" {
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
