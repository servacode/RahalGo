package server

// ══════════════════════════════════════════════════════════════════════
// **ملفُّ الطلب في اللوحة** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
// ══════════════════════════════════════════════════════════════════════
//
// «لمّا افتح السجلّ لازم افهم لكلّ طلب: شو صار، كيف مشي، وحركة المال.»
//
// **ما لا يحمله `orders.Order`** ويحتاجه الملفّ — يُلحق بردّ `GET /admin/orders/{id}` وحدَه،
// **فلا يصل تطبيقَ الزبون ولا المتجر ولا البثّ** (عقدُ الخصوصيّة لا يتغيّر):
//
//   - **خصمُ العرض على كلّ صنفٍ ومن تحمّله** — `order_items.offer_cut/offer_borne_by`.
//   - **محوُ العمولة: كم، ومن، ومتى، ولماذا.**
//   - **الشكاوى والمطالبات والطوارئ** المربوطةُ بالطلب، بحالها.
//   - **مجاميعُ الدفتر للطلب** — قراءةٌ لا كتابة: ما قُيِّد للمتجر والسائق والمندوب والمنصّة.
//   - **مهلةُ «متأخّر»** وحكمُها — الشرطُ نفسُه الذي يفلتر به السجلّ (`orders.LateDeliverySQL`).
//
// **وكلُّ قسمٍ يفشل وحدَه** — قراءةٌ تعذّرت تترك قسمها فارغاً ولا تُسقط الملفّ.

import (
	"context"
	"time"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

type fileItemOffer struct {
	ItemID string `json:"item_id"`
	// Cut **خصمُ العرض على الوحدة** — يُضرب بعدد البند.
	Cut     int64  `json:"offer_cut"`
	BorneBy string `json:"offer_borne_by"`
}

type fileWaiver struct {
	Amount int64      `json:"amount"`
	ByName string     `json:"by_name"`
	At     *time.Time `json:"at"`
	Reason string     `json:"reason"`
}

type fileCase struct {
	// Kind ticket · dispute · emergency
	Kind      string    `json:"kind"`
	ID        string    `json:"id"`
	Number    *int64    `json:"number,omitempty"`
	Status    string    `json:"status"`
	Title     string    `json:"title"`
	Amount    int64     `json:"amount,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// fileLedger **مجاميعُ الدفتر للطلب بنوع القيد** — عرضٌ لا حساب.
type fileLedger struct {
	Merchant int64 `json:"merchant"`
	Driver   int64 `json:"driver"`
	Rep      int64 `json:"rep"`
	Platform int64 `json:"platform"`
	Lines    int   `json:"lines"`
}

type orderFile struct {
	ItemOffers   []fileItemOffer `json:"item_offers"`
	Waiver       *fileWaiver     `json:"commission_waiver"`
	Cases        []fileCase      `json:"cases"`
	Ledger       fileLedger      `json:"ledger"`
	LateAfterMin int             `json:"late_after_min"`
	Late         bool            `json:"late"`
}

// orderFileOf **يبني ملحقَ ملفّ الطلب** — قراءاتٌ لا كتابة.
func (s *Server) orderFileOf(ctx context.Context, o *orders.Order) orderFile {
	f := orderFile{ItemOffers: []fileItemOffer{}, Cases: []fileCase{}}

	if rows, err := s.pg.Query(ctx, `
		SELECT id::text, offer_cut, COALESCE(offer_borne_by, '')
		FROM order_items WHERE order_id = $1 AND offer_cut > 0 ORDER BY id`, o.ID); err == nil {
		for rows.Next() {
			var it fileItemOffer
			if rows.Scan(&it.ItemID, &it.Cut, &it.BorneBy) == nil {
				f.ItemOffers = append(f.ItemOffers, it)
			}
		}
		rows.Close()
	}

	var w fileWaiver
	if err := s.pg.QueryRow(ctx, `
		SELECT o.commission_waived, o.commission_waived_at,
		       COALESCE(NULLIF(u.full_name, ''), u.phone::text, ''), COALESCE(o.commission_waived_reason, '')
		FROM orders o LEFT JOIN users u ON u.id = o.commission_waived_by
		WHERE o.id = $1`, o.ID).Scan(&w.Amount, &w.At, &w.ByName, &w.Reason); err == nil && w.Amount > 0 {
		f.Waiver = &w
	}

	if rows, err := s.pg.Query(ctx, `
		SELECT 'ticket', id::text, number, status, COALESCE(NULLIF(subject, ''), COALESCE(reason, '')), compensation, created_at
		  FROM tickets WHERE order_id = $1
		UNION ALL
		SELECT 'dispute', id::text, NULL::bigint, status, reason, amount, created_at
		  FROM disputes WHERE order_id = $1
		UNION ALL
		SELECT 'emergency', id::text, NULL::bigint, status, note, 0, created_at
		  FROM driver_emergencies WHERE order_id = $1
		ORDER BY 7`, o.ID); err == nil {
		for rows.Next() {
			var c fileCase
			if rows.Scan(&c.Kind, &c.ID, &c.Number, &c.Status, &c.Title, &c.Amount, &c.CreatedAt) == nil {
				f.Cases = append(f.Cases, c)
			}
		}
		rows.Close()
	}

	// **مجاميعُ الدفتر بنوع القيد** — كما يقرؤها تفصيلُ المال (`order_breakdown.go`): الطرفُ من نوع
	// القيد لا من دور صاحبه. **قراءةٌ وحدَها — الدفترُ لا يُمسّ.**
	if rows, err := s.pg.Query(ctx, `
		SELECT kind, COALESCE(sum(amount), 0), count(*)
		FROM wallet_transactions WHERE ref = $1 GROUP BY kind`, o.ID); err == nil {
		for rows.Next() {
			var kind string
			var sum int64
			var n int
			if rows.Scan(&kind, &sum, &n) != nil {
				continue
			}
			f.Ledger.Lines += n
			switch kind {
			case "merchant_earning":
				f.Ledger.Merchant += sum
			case "driver_earning":
				f.Ledger.Driver += sum
			case "commission":
				f.Ledger.Rep += sum
			case "platform_profit":
				f.Ledger.Platform += sum
			}
		}
		rows.Close()
	}

	f.LateAfterMin = s.orders.StuckLimitsOf(ctx).DeliveryMin
	f.Late = lateDelivery(o.CreatedAt, o.DeliveredAt, f.LateAfterMin)
	return f
}

// lateDelivery **سُلّم بعد مهلة عمر الطلب** — توأمُ `orders.LateDeliverySQL` لطلبٍ واحد.
func lateDelivery(created time.Time, delivered *time.Time, limitMin int) bool {
	if delivered == nil {
		return false
	}
	if limitMin < 0 {
		limitMin = 0
	}
	return delivered.Sub(created) > time.Duration(limitMin)*time.Minute
}
