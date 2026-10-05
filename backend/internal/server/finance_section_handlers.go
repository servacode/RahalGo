package server

// ══════════════════════════════════════════════════════════════════════
// **بابان للماليّة داخل أقسامها** (قرارُ المالك ٢٠٢٦-١٠-٠٥، أجوبةُ الأسئلة ١ و٢)
// ══════════════════════════════════════════════════════════════════════
//
// **الماليّةُ لا ترى الحساباتِ ولا الطلبات** (هجرة `0422`) — وعملان من عملها كانا
// هناك وحدَهما:
//
//	«شحن محفظة»      اقتراحُ حركةٍ يدويّة كان من ملفّ الحساب. **فبحثٌ ضيّقٌ بقدرة
//	                 `finance.manage`** يردّ ما يلزم الاقتراحَ وحدَه — المعرّف والاسم
//	                 والهاتف والأدوار والرصيد — **ولا ملفَّ ولا عنوانَ ولا نشاط.** والاقتراحُ
//	                 نفسُه من بابه القائم (`POST /users/{id}/wallet`): سقفُه
//	                 `finance.manual_wallet_max`، **ولا يوافق عليه مقترحُه** (`approval.Check`).
//
//	«تعويضُ بضاعةٍ راجعة»  كان زرّاً في لوح الطلبات. **فقائمةٌ في «الخسائر والنزاعات»**
//	                 بالطلبات التي حسمت العمليّاتُ بضاعتَها «إلى المتجر» ولم تُعوَّض،
//	                 **والتعويضُ من بابه القائم** (`POST /orders/{id}/goods/compensation`)
//	                 فيمرّ بصفحة «التعويضات» كغيره.

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// walletLookupRow **ما يلزم اقتراحَ حركةٍ لا أكثر** — يحرسه `TestFinSections_WalletTopupFromTreasury`.
//
// **والهاتفُ يُحذف عند حدّ الخروج** لمن لا يملك `users.contact.read` (`XG-42`،
// `response_shape.go`) — **فالماليّةُ تجد الحسابَ برقمه ولا تقرأ الرقم.**
type walletLookupRow struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Phone   string   `json:"phone"`
	Roles   []string `json:"roles"`
	Balance int64    `json:"balance"`
}

// handleWalletLookup **بحثُ «شحن محفظة»** — `GET /admin/treasury/wallet-lookup?q=`.
//
// **بالهاتف أو الاسم، وحرفان على الأقلّ، وعشرون سطراً أقصى** — بحثٌ لا دليل:
// **من أراد تصفّحَ الحسابات فبابُه الحسابات** (`users.read`)، وليس هذا.
func (s *Server) handleWalletLookup(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	out := []walletLookupRow{}
	if len([]rune(q)) < 2 {
		httpx.JSON(w, http.StatusOK, map[string]any{"accounts": out})
		return
	}
	// **والرقمُ يُطابَق بأرقامه** — «0933…» و«+963933…» سواء.
	digits := strings.Map(func(c rune) rune {
		if c >= '0' && c <= '9' {
			return c
		}
		return -1
	}, q)
	if strings.HasPrefix(digits, "0") {
		digits = digits[1:]
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT u.id::text, COALESCE(u.full_name, ''), COALESCE(u.phone, ''),
		       COALESCE((SELECT array_agg(role_code ORDER BY role_code)
		                   FROM user_roles WHERE user_id = u.id), '{}'),
		       COALESCE((SELECT balance FROM wallets WHERE user_id = u.id), 0)
		FROM users u
		WHERE u.deleted_at IS NULL
		  AND NOT EXISTS (SELECT 1 FROM wallets tw WHERE tw.user_id = u.id AND tw.is_treasury)
		  AND (u.full_name ILIKE '%' || $1 || '%'
		       OR (length($2) >= 3 AND u.phone LIKE '%' || $2 || '%'))
		ORDER BY u.full_name NULLS LAST, u.phone
		LIMIT 20`, q, digits)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var a walletLookupRow
		if err := rows.Scan(&a.ID, &a.Name, &a.Phone, &a.Roles, &a.Balance); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"accounts": out})
}

// goodsCompRow **طلبٌ حُسمت بضاعتُه «إلى المتجر»** — وحالُ تعويضه.
type goodsCompRow struct {
	OrderID      string     `json:"order_id"`
	OrderNumber  int64      `json:"order_number"`
	MerchantName string     `json:"merchant_name"`
	GoodsCost    int64      `json:"goods_cost"`
	ClosedAt     *time.Time `json:"closed_at"`
	// RequestStatus **طلبُ التعويض إن كُتب**: `pending` · `rejected` · وفارغٌ = لم يُكتب.
	RequestStatus string `json:"request_status"`
	RequestAmount int64  `json:"request_amount"`
}

// handleGoodsCompensations **بضاعةٌ رجعت إلى متجرها ولم تُعوَّض** —
// `GET /admin/losses/goods-compensations`.
//
// **ولا يظهر ما عُوِّض** (قيدُ `compensation` على الطلب لصاحب المتجر) **ولا ما
// وُوفق عليه** — والمعلَّقُ والمرفوضُ يظهران بحالهما فلا يُكتب طلبٌ ثانٍ عمىً.
func (s *Server) handleGoodsCompensations(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 && n < limit {
		limit = n
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT o.id::text, o.number, COALESCE(m.name, ''),
		       COALESCE((SELECT sum(oi.merchant_price * oi.qty) FROM order_items oi
		                  WHERE oi.order_id = o.id), 0),
		       o.closed_at,
		       COALESCE(req.status, ''), COALESCE(req.suggested_amount, 0)
		FROM orders o
		JOIN merchants m ON m.id = o.merchant_id
		LEFT JOIN LATERAL (
		    SELECT r.status, r.suggested_amount FROM driver_compensation_requests r
		     WHERE r.order_id = o.id AND r.kind = 'merchant_goods'
		     ORDER BY r.created_at DESC LIMIT 1) req ON true
		WHERE o.goods_settled_to = 'merchant'
		  AND COALESCE(req.status, '') <> 'approved'
		  AND NOT EXISTS (
		      SELECT 1 FROM wallet_transactions wt
		       WHERE wt.ref = o.id::text AND wt.kind = 'compensation'
		         AND wt.user_id = m.owner_user_id)
		ORDER BY o.closed_at DESC NULLS LAST, o.number DESC
		LIMIT $1`, limit)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []goodsCompRow{}
	for rows.Next() {
		var g goodsCompRow
		if err := rows.Scan(&g.OrderID, &g.OrderNumber, &g.MerchantName, &g.GoodsCost,
			&g.ClosedAt, &g.RequestStatus, &g.RequestAmount); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"orders": out})
}
