package server

// تقريرُ «أرصدة من بضاعة» — **قراءةٌ فقط** (قرارُ المالك ٢٠٢٦-١١-١٠، القرار ٥، خطّةُ المال B1).
//
// القاعدةُ الجديدة: **البضاعةُ لا تدخل حساباً** — المنصّةُ تلمس الأجرةَ والعمولةَ وحدَهما. **والأرصدةُ
// القديمةُ تبقى كما هي** (لا تُمحى ولا تُحوَّل)، فيلزم المالكَ أن يرى مَن يحمل اليومَ مالاً أصلُه بضاعة:
//
//  1. store_wallets    — محافظُ المتاجر «محفظة»: ما دخلها مستحقّاً عن بضاعة (`merchant_settlements`
//     بطريقة wallet، صافيَ المعكوس) ورصيدُ المحفظة اليوم.
//  2. store_debts      — ديونُ متاجرَ أصلُها بضاعة: استرجاعٌ بعد دفع · بضاعةٌ رُدّت · تعديلٌ بعد الاستلام ·
//     عمولةُ طلب واتساب.
//  3. driver_debts     — ديونُ سائقين من خسارة بضاعة (`failed_order_driver`).
//  4. cash_holding     — مستحقّاتُ متاجرَ نقديّةٌ لم تُدفع بعد (محفظةُ الاحتباس) — **بشرط تبويب النقد نفسِه.**
//  5. customer_refunds — ما دخل محافظَ الزبائن من ثمن بضاعةٍ باسترجاع طلبٍ مُسلَّم (المسترجَعُ فوق الأجرة).
//  6. driver_advances  — ما دفعه سائقٌ لمتجرٍ عن طلبٍ استلمه ولم ينتهِ بعد (سلفةٌ قائمة).
//
// **ولا زرَّ يحرّك مالاً** — الصفحةُ للعرض، والقرارُ في كلّ رصيدٍ للمالك.

import (
	"context"
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// goodsBalanceRow طرفٌ يحمل مالاً أصلُه بضاعة.
type goodsBalanceRow struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Phone  string  `json:"phone"`
	Amount int64   `json:"amount"`
	Count  int64   `json:"count"`
	Orders []int64 `json:"orders"`
	// Balance **رصيدُ المحفظة اليوم** — للمتاجر والزبائن وحدَهم (وإلّا null).
	Balance *int64 `json:"balance"`
}

type goodsBalanceSection struct {
	Rows  []goodsBalanceRow `json:"rows"`
	Total int64             `json:"total"`
}

// goodsStoreDebtCauses **أسبابُ دين المتجر التي أصلُها بضاعة** (`obligations.go`).
const goodsStoreDebtCauses = `('refund_merchant_earning', 'returned_goods', 'order_edit', 'whatsapp_order_commission')`

// goodsOpenOrderSQL **طلبٌ لم ينتهِ** — ما لم يُسلَّم ولم يُغلق.
const goodsOpenOrderSQL = `o.status NOT IN ('delivered', 'rejected', 'cancelled', 'failed', 'refunded')`

// كلُّ استعلامٍ يردّ: المعرّف · الاسم · الهاتف · المبلغ · العدد · أرقامُ الطلبات (أوّلُ عشرين) · الرصيد.
var goodsBalanceQueries = []struct{ key, sql string }{
	{"store_wallets", `
		SELECT m.id::text, COALESCE(m.name, ''), COALESCE(NULLIF(m.phone, ''), u.phone, ''),
		       sum(ms.amount - ms.reversed_amount)::bigint, count(*),
		       (array_agg(o.number ORDER BY ms.created_at DESC))[1:20],
		       COALESCE(w.balance, 0)::bigint
		FROM merchant_settlements ms
		JOIN merchants m ON m.id = ms.merchant_id
		JOIN orders o ON o.id = ms.order_id
		LEFT JOIN users u ON u.id = m.owner_user_id
		LEFT JOIN wallets w ON w.user_id = m.owner_user_id
		WHERE ms.method = 'wallet' AND ms.amount > ms.reversed_amount
		GROUP BY m.id, m.name, m.phone, u.phone, w.balance
		ORDER BY 4 DESC`},
	{"store_debts", `
		SELECT m.id::text, COALESCE(m.name, ''), COALESCE(NULLIF(m.phone, ''), u.phone, ''),
		       sum(fo.amount - fo.settled)::bigint, count(*),
		       array_remove((array_agg(o.number ORDER BY fo.created_at DESC))[1:20], NULL),
		       NULL::bigint
		FROM financial_obligations fo
		JOIN merchants m ON m.id = fo.party_id
		LEFT JOIN orders o ON o.id = fo.order_id
		LEFT JOIN users u ON u.id = m.owner_user_id
		WHERE fo.party_kind = 'merchant' AND fo.closed_at IS NULL
		  AND fo.cause IN ` + goodsStoreDebtCauses + `
		GROUP BY m.id, m.name, m.phone, u.phone
		ORDER BY 4 DESC`},
	{"driver_debts", `
		SELECT u.id::text, COALESCE(u.full_name, ''), u.phone,
		       sum(fo.amount - fo.settled)::bigint, count(*),
		       array_remove((array_agg(o.number ORDER BY fo.created_at DESC))[1:20], NULL),
		       NULL::bigint
		FROM financial_obligations fo
		JOIN users u ON u.id = fo.party_id
		LEFT JOIN orders o ON o.id = fo.order_id
		WHERE fo.party_kind = 'driver' AND fo.closed_at IS NULL AND fo.cause = 'failed_order_driver'
		GROUP BY u.id, u.full_name, u.phone
		ORDER BY 4 DESC`},
	{"cash_holding", `
		SELECT m.id::text, COALESCE(m.name, ''), COALESCE(NULLIF(m.phone, ''), u.phone, ''),
		       sum(ms.amount - ms.reversed_amount)::bigint, count(*),
		       (array_agg(o.number ORDER BY ms.created_at))[1:20],
		       NULL::bigint
		FROM merchant_settlements ms
		JOIN merchants m ON m.id = ms.merchant_id
		JOIN orders o ON o.id = ms.order_id
		LEFT JOIN users u ON u.id = m.owner_user_id
		WHERE ` + merchantCashDueWhere + `
		GROUP BY m.id, m.name, m.phone, u.phone
		ORDER BY 4 DESC`},
	{"customer_refunds", `
		WITH r AS (
			SELECT o.customer_id, o.number, o.updated_at,
			       GREATEST((SELECT COALESCE(sum(t.amount), 0) FROM wallet_transactions t
			                  WHERE t.kind = 'refund' AND t.ref = o.id::text AND t.user_id = o.customer_id)
			                - (o.delivery_fee - o.promo_delivery_waived), 0) AS goods
			FROM orders o
			WHERE o.status = 'refunded' AND o.customer_id IS NOT NULL
			  AND EXISTS (SELECT 1 FROM order_events e
			               WHERE e.order_id = o.id AND e.to_status = 'delivered'))
		SELECT u.id::text, COALESCE(u.full_name, ''), u.phone,
		       sum(r.goods)::bigint, count(*),
		       (array_agg(r.number ORDER BY r.updated_at DESC))[1:20],
		       COALESCE(w.balance, 0)::bigint
		FROM r
		JOIN users u ON u.id = r.customer_id
		LEFT JOIN wallets w ON w.user_id = u.id
		WHERE r.goods > 0
		GROUP BY u.id, u.full_name, u.phone, w.balance
		ORDER BY 4 DESC`},
	{"driver_advances", `
		SELECT u.id::text, COALESCE(u.full_name, ''), u.phone,
		       sum(ms.amount - ms.reversed_amount)::bigint, count(*),
		       (array_agg(o.number ORDER BY ms.created_at DESC))[1:20],
		       NULL::bigint
		FROM merchant_settlements ms
		JOIN orders o ON o.id = ms.order_id
		JOIN users u ON u.id = ms.paid_by
		WHERE ms.paid_via = 'driver' AND ms.state = 'cash_paid' AND NOT ms.off_ledger
		  AND ms.amount > ms.reversed_amount AND ` + goodsOpenOrderSQL + `
		GROUP BY u.id, u.full_name, u.phone
		ORDER BY 4 DESC`},
}

// handleGoodsBalances **مَن يحمل مالاً أصلُه بضاعة — قراءةٌ فقط** (`finance.read`).
func (s *Server) handleGoodsBalances(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{"generated_at": time.Now().UTC()}
	for _, q := range goodsBalanceQueries {
		sec, err := s.goodsBalanceSection(r.Context(), q.sql)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		out[q.key] = sec
	}
	stuck, err := s.stuckDriverMoney(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	out["stuck_driver_money"] = stuck
	httpx.JSON(w, http.StatusOK, out)
}

// stuckRow **«مصاري الكابتن عالقة»** — طلبٌ من النموذج ٢ رفض متجرُه يرجّع ثمنَ بضاعته (القرار ٤).
type stuckRow struct {
	OrderID  string  `json:"order_id"`
	Number   int64   `json:"number"`
	Driver   string  `json:"driver"`
	Phone    string  `json:"phone"`
	Amount   int64   `json:"amount"`
	Decision *string `json:"decision"`
}

// stuckDriverMoney **ما بانتظار قرار المكتب أوّلاً، ثمّ ما قُرّر «دين على الزبون»** (سجلٌّ مفتوح).
func (s *Server) stuckDriverMoney(ctx context.Context) ([]stuckRow, error) {
	rows, err := s.pg.Query(ctx, `
		SELECT o.id::text, o.number, COALESCE(u.full_name, ''), COALESCE(u.phone, ''),
		       o.stuck_goods_amount, o.stuck_goods_decision
		FROM orders o
		LEFT JOIN users u ON u.id = o.driver_id
		WHERE o.stuck_goods_amount > 0
		  AND (o.stuck_goods_decision IS NULL OR o.stuck_goods_decision = 'customer')
		ORDER BY o.stuck_goods_decision NULLS FIRST, o.updated_at DESC
		LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []stuckRow{}
	for rows.Next() {
		var x stuckRow
		if err := rows.Scan(&x.OrderID, &x.Number, &x.Driver, &x.Phone, &x.Amount, &x.Decision); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Server) goodsBalanceSection(ctx context.Context, sql string) (*goodsBalanceSection, error) {
	rows, err := s.pg.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sec := &goodsBalanceSection{Rows: []goodsBalanceRow{}}
	for rows.Next() {
		var g goodsBalanceRow
		if err := rows.Scan(&g.ID, &g.Name, &g.Phone, &g.Amount, &g.Count, &g.Orders, &g.Balance); err != nil {
			return nil, err
		}
		if g.Orders == nil {
			g.Orders = []int64{}
		}
		sec.Total += g.Amount
		sec.Rows = append(sec.Rows, g)
	}
	return sec, rows.Err()
}
