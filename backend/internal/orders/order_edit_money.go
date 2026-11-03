package orders

// ══════════════════════════════════════════════════════════════════════
// **تصحيحُ حقّ المتجر بعد الاستلام** — قرارُ المالك ٢٠٢٦-١١-٠٣ (المرحلة ٢)
// ══════════════════════════════════════════════════════════════════════
//
// عند الاستلام قُيّد حقُّ كلِّ متجرٍ في صفّ تسويته (`settleMerchant`). **فبعد تعديل البنود
// يُعاد حسابُه بالمعادلة نفسِها** (`merchantShares` — سعرُ الشراء ناقصَ العمولة بالنسبة
// المحفوظة في الطلب) **ويُقيَّد الفرقُ وحدَه قيداً جديداً** — ولا يُمسّ قيدٌ قديم:
//
//	محفظة  زيادة  ←  +merchant_earning في محفظة المتجر
//	       نقص    ←  −merchant_earning بما يحتمله رصيدُه، **والباقي دينٌ عليه** (`order_edit`)
//	نقداً  زيادة  ←  +merchant_cash_accrued ثمّ −merchant_cash_paid: **السائقُ الحاليُّ دفع الفرق**
//	       نقص    ←  +merchant_cash_paid ثمّ −merchant_cash_accrued: **المتجرُ ردّ الفرقَ للسائق**
//
// **والأنواعُ نفسُها التي تقرؤها الخزينةُ والكشوفُ والاسترجاع** — فلا يغيب التصحيحُ عن
// حسبةٍ تجمع بالنوع. **ويُعرف أنّه تصحيحٌ لا تكرار** بسطره في
// `merchant_settlement_adjustments` (وبه تميّزه فحوصُ الدفتر)، **و`amount` الأصليُّ لا يُمسّ**:
// الحقُّ الحاليّ = `amount + adjusted_amount`.
//
// # ما لا يُصحَّح آليّاً
//
// متجرٌ «نقداً» لم يُدفع له عند الاستلام (`cash_due`) أو اقتطع دينُه حقَّه كلَّه
// (`cash_reversed`)، أو مُحيت عمولتُه — **يُردّ التعديلُ برسالته** ولا يُخمَّن قيد.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

var (
	// ErrOrderEditSettlement **حقُّ متجرٍ في حالٍ لا يُصحَّح آليّاً.**
	ErrOrderEditSettlement = httpx.NewError(http.StatusConflict,
		"order_edit_settlement_state", "errors.order_edit_settlement_state")
	// ErrOrderEditNoDriver **فرقٌ نقديٌّ لمتجرٍ ولا سائقَ على الطلب يدفعه أو يقبضه.**
	ErrOrderEditNoDriver = httpx.NewError(http.StatusConflict,
		"order_edit_needs_driver", "errors.order_edit_needs_driver")
)

// correctSettlementsTx **يُعيد حسابَ حقّ كلِّ متجرٍ ويقيّد الفرق** — في معاملة التعديل.
func (s *Service) correctSettlementsTx(ctx context.Context, tx pgx.Tx, orderID, actorID string,
	driverID *string, reason string) ([]EditStoreDelta, error) {
	shares, totalCommission, err := s.merchantShares(ctx, tx, orderID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE orders SET platform_commission = $2 WHERE id = $1`, orderID, totalCommission); err != nil {
		return nil, err
	}
	newDue := map[string]int64{}
	for _, sh := range shares {
		if sh.due > 0 {
			newDue[sh.merchantID] = sh.due
		}
	}

	type row struct {
		id, merchantID, ownerID, method, state, name string
		amount, adjusted, reversed, waived           int64
	}
	rows, err := tx.Query(ctx, `
		SELECT ms.id::text, ms.merchant_id::text, ms.owner_user_id::text, ms.method, ms.state,
		       m.name, ms.amount, ms.adjusted_amount, ms.reversed_amount, ms.commission_waived
		FROM merchant_settlements ms
		JOIN merchants m ON m.id = ms.merchant_id
		WHERE ms.order_id = $1
		ORDER BY ms.created_at, ms.id
		FOR UPDATE OF ms`, orderID)
	if err != nil {
		return nil, err
	}
	var list []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.merchantID, &r.ownerID, &r.method, &r.state, &r.name,
			&r.amount, &r.adjusted, &r.reversed, &r.waived); err != nil {
			rows.Close()
			return nil, err
		}
		list = append(list, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []EditStoreDelta
	note := fmt.Sprintf("تصحيح حق المتجر بتعديل الطلب: %s", reason)
	for _, r := range list {
		cur := r.amount + r.adjusted
		delta := newDue[r.merchantID] - cur
		if delta == 0 {
			continue
		}
		if r.waived > 0 {
			return nil, ErrOrderEditSettlement
		}
		var adjID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO merchant_settlement_adjustments (settlement_id, order_id, delta, reason, actor_id)
			VALUES ($1, $2, $3, $4, NULLIF($5, '')::uuid) RETURNING id::text`,
			r.id, orderID, delta, reason, actorID).Scan(&adjID); err != nil {
			return nil, err
		}
		d := EditStoreDelta{MerchantID: r.merchantID, Name: r.name, Method: r.method,
			OldDue: cur, NewDue: cur + delta}

		switch r.method {
		case "wallet":
			if r.state != "wallet_credited" {
				return nil, ErrOrderEditSettlement
			}
			amount := delta
			if delta < 0 {
				// **ما يحتمله رصيدُه يُستردّ، والباقي دينٌ عليه** — ولا رصيدَ سالب.
				avail, err := s.wallet.AvailableTx(ctx, tx, r.ownerID)
				if err != nil {
					return nil, err
				}
				if avail < 0 {
					avail = 0
				}
				take := min64(-delta, avail)
				amount = -take
				d.Debt = -delta - take
			}
			if amount != 0 {
				_, txID, err := s.wallet.ApplyTxID(ctx, tx, r.ownerID, amount, "merchant_earning",
					orderID, note, &actorID)
				if err != nil {
					return nil, err
				}
				if _, err := tx.Exec(ctx,
					`UPDATE merchant_settlement_adjustments SET earning_tx_id = $2 WHERE id = $1`,
					adjID, txID); err != nil {
					return nil, err
				}
			}
			if d.Debt > 0 {
				if _, err := obligations.CreateRef(ctx, tx, obligations.PartyMerchant, r.merchantID,
					d.Debt, obligations.CauseOrderEdit, orderID, adjID, &actorID); err != nil {
					return nil, err
				}
				if _, err := tx.Exec(ctx,
					`UPDATE merchant_settlement_adjustments SET debt = $2 WHERE id = $1`,
					adjID, d.Debt); err != nil {
					return nil, err
				}
			}
		case "cash":
			// **النقدُ يُدفع بيد السائق عند الاستلام** (قرارُ المالك ٢٠٢٦-١٠-١٥) — فالفرقُ كذلك.
			if r.state != "cash_paid" {
				return nil, ErrOrderEditSettlement
			}
			if driverID == nil || *driverID == "" {
				return nil, ErrOrderEditNoDriver
			}
			// **ولا يردّ المتجرُ أكثرَ ممّا قبض.**
			if delta < 0 && -delta > cur-r.reversed {
				return nil, ErrOrderEditSettlement
			}
			hid, err := s.cashHoldingOn(ctx, tx)
			if err != nil {
				return nil, err
			}
			// **الترتيبُ يحفظ الاحتباسَ فوق الصفر**: ما يزيده أوّلاً ثمّ ما ينقصه.
			var accruedTx, paidTx int64
			if delta > 0 {
				if _, accruedTx, err = s.wallet.ApplyTxID(ctx, tx, hid, delta, "merchant_cash_accrued",
					orderID, note, &actorID); err != nil {
					return nil, err
				}
				if _, paidTx, err = s.wallet.ApplyTxID(ctx, tx, hid, -delta, "merchant_cash_paid",
					orderID, "فرقُ حقّ المتجر دفعه السائقُ بتعديل الطلب", driverID); err != nil {
					return nil, err
				}
			} else {
				if _, paidTx, err = s.wallet.ApplyTxID(ctx, tx, hid, -delta, "merchant_cash_paid",
					orderID, "فرقُ حقّ المتجر ردّه المتجرُ للسائق بتعديل الطلب", driverID); err != nil {
					return nil, err
				}
				if _, accruedTx, err = s.wallet.ApplyTxID(ctx, tx, hid, delta, "merchant_cash_accrued",
					orderID, note, &actorID); err != nil {
					return nil, err
				}
			}
			if _, err := tx.Exec(ctx, `
				UPDATE merchant_settlement_adjustments
				   SET accrued_tx_id = $2, paid_tx_id = $3, paid_by = $4, paid_via = $5
				 WHERE id = $1`, adjID, accruedTx, paidTx, *driverID, paidViaDriver); err != nil {
				return nil, err
			}
			d.DriverPays = delta
		default:
			return nil, ErrOrderEditSettlement
		}
		if _, err := tx.Exec(ctx, `
			UPDATE merchant_settlements SET adjusted_amount = adjusted_amount + $2 WHERE id = $1`,
			r.id, delta); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

// editDebtsByOwner **دَينُ تعديل الطلب لكلّ صاحبِ محفظة** — ما لم يكفِه رصيدُ متجرٍ «محفظة»
// حين نقص حقُّه. **يُطرح ممّا يُسترَدّ من الدفتر** (الاسترجاعُ والبضاعةُ المردودة): قائمٌ
// عليه ديناً، فلا يُؤخذ مرّتين.
func editDebtsByOwner(ctx context.Context, q wallet.Querier, orderID string) (map[string]int64, error) {
	rows, err := q.Query(ctx, `
		SELECT ms.owner_user_id::text, sum(a.debt)::bigint
		FROM merchant_settlement_adjustments a
		JOIN merchant_settlements ms ON ms.id = a.settlement_id
		WHERE a.order_id = $1 AND a.debt > 0
		GROUP BY 1`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var owner string
		var d int64
		if err := rows.Scan(&owner, &d); err != nil {
			return nil, err
		}
		out[owner] = d
	}
	return out, rows.Err()
}
