package orders

// ══════════════════════════════════════════════════════════════════════
// **أقسامُ الطلب في حمولة كلّ طرف** — المكتبُ والزبونُ والمتجر (عقدُ التطبيقات §2–§4)
// ══════════════════════════════════════════════════════════════════════
//
//	المكتب   parts         متجرُ كلّ قسمٍ وهاتفُه وعنوانُه · حالُه ومن أجابه · أوقاتُه · بنودُه
//	                       (ولقسمٍ رُفض أو انشال: لقطةُ ما خرج) · ماله وطريقةُ تسويته وما يدفعه السائق
//	الزبون   sections      «القسم ١ / ٢» بحاله وبنوده — **بلا اسم متجرٍ ولا معرّفه** (`P-1`)؛
//	                       وما خرج «unavailable». **وقسمٌ حُوّل لمتجرٍ آخر لا يُرى**: بضاعتُه آتية.
//	المتجر   shared_stops  عددُ متاجر الطلب — بلا أسمائها (`ProjectForStores`)
//
// **ولطلب المتجر الواحد لا شيء** (`omitempty`) — **حمولتُه حرفاً كما كانت.**

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// OpsPartItem **بندُ قسمٍ كما يراه المكتب.**
type OpsPartItem struct {
	Name          string `json:"name"`
	Qty           int    `json:"qty"`
	UnitPrice     int64  `json:"unit_price"`
	MerchantPrice int64  `json:"merchant_price"`
	Options       string `json:"options"`
	Note          string `json:"note"`
}

// OpsPart **قسمُ متجرٍ في حمولة المكتب** — العقدُ §2.
type OpsPart struct {
	MerchantID       string        `json:"merchant_id"`
	Seq              int           `json:"seq"`
	Name             string        `json:"name"`
	Phone            string        `json:"phone"`
	Address          string        `json:"address"`
	Lat              *float64      `json:"lat"`
	Lng              *float64      `json:"lng"`
	Status           string        `json:"status"`
	AnsweredRole     string        `json:"answered_role"`
	AnsweredAt       *time.Time    `json:"answered_at"`
	Reason           string        `json:"reason"`
	RemovedFault     string        `json:"removed_fault"`
	PrepMinutes      *int          `json:"prep_minutes"`
	ReadyAt          *time.Time    `json:"ready_at"`
	SentAt           *time.Time    `json:"sent_at"`
	SentChannel      string        `json:"sent_channel"`
	ArrivedAt        *time.Time    `json:"arrived_at"`
	PickedUpAt       *time.Time    `json:"picked_up_at"`
	PickedUpByName   string        `json:"picked_up_by_name"`
	Collected        bool          `json:"collected"`
	Items            []OpsPartItem `json:"items"`
	Subtotal         int64         `json:"subtotal"`
	Cost             int64         `json:"cost"`
	SettlementMethod string        `json:"settlement_method"`
	PayAmount        int64         `json:"pay_amount"`
	RepName          string        `json:"rep_name"`
	RepCommission    int64         `json:"rep_commission"`
}

// CustomerSectionItem **بندُ قسمٍ للزبون** — اسمُه وعددُه.
type CustomerSectionItem struct {
	Name string `json:"name"`
	Qty  int    `json:"qty"`
}

// CustomerSection **قسمٌ من طلب الزبون** — العقدُ §3: بلا اسم متجرٍ ولا معرّفه.
type CustomerSection struct {
	Seq    int                   `json:"seq"`
	Status string                `json:"status"`
	Items  []CustomerSectionItem `json:"items"`
}

// snapItem **بندٌ في لقطة ما خرج** (`order_parts.removed_items`).
type snapItem struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Qty           int              `json:"qty"`
	UnitPrice     int64            `json:"unit_price"`
	MerchantPrice int64            `json:"merchant_price"`
	Note          string           `json:"note"`
	Options       []OptionSnapshot `json:"options"`
	DrinkFor      *string          `json:"drink_for"`
}

func optionsLabel(opts []OptionSnapshot) string {
	names := make([]string, 0, len(opts))
	for _, o := range opts {
		if o.Name != "" {
			names = append(names, o.Name)
		}
	}
	return strings.Join(names, "، ")
}

// fillPartPayloads **يملأ أقسامَ كلّ طلبٍ بعدّة محطّات** — للمكتب والزبون؛ وما سواه لا يُمسّ.
func (s *Service) fillPartPayloads(ctx context.Context, q dbtx.Querier, list []*Order) error {
	var ids []string
	byID := map[string]*Order{}
	for _, o := range list {
		if o != nil && o.Stops > 1 {
			ids = append(ids, o.ID)
			byID[o.ID] = o
		}
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := q.Query(ctx, `
		SELECT p.order_id::text, p.merchant_id::text, p.seq, m.name,
		       COALESCE(NULLIF(mo.phone::text, ''), ''), COALESCE(m.address_text, ''),
		       ST_Y(m.location::geometry), ST_X(m.location::geometry),
		       p.status, p.answered_role, p.answered_at, p.reason, p.removed_fault,
		       p.prep_minutes, p.ready_at, p.sent_at, p.sent_channel, p.arrived_at, p.picked_up_at,
		       COALESCE(pu.full_name, ''),
		       p.status = 'picked_up' OR ms.order_id IS NOT NULL,
		       p.removed_items, p.removed_total,
		       COALESCE(ms.method,
		                (SELECT max(oi.merchant_settlement_method) FROM order_items oi
		                  WHERE oi.order_id = p.order_id AND oi.merchant_id = p.merchant_id),
		                NULLIF(m.settlement_method, ''), 'cash'),
		       COALESCE(ru.full_name, ''), p.rep_commission,
		       -- **قسمٌ حُوّل لمتجرٍ آخر**: بنودُ لقطته ما زالت في الطلب — انتقلت ولم تُحذف.
		       p.status = 'removed' AND EXISTS (
		           SELECT 1 FROM jsonb_array_elements(p.removed_items) x
		           JOIN order_items oi ON oi.id::text = x->>'id' AND oi.order_id = p.order_id),
		       -- **أبيد سائق الطلب بضاعةُ هذه المحطّة؟** (goods_in_hand).
		       COALESCE(p.status = 'picked_up'
		           AND p.picked_up_by = (SELECT o2.driver_id FROM orders o2 WHERE o2.id = p.order_id), false)
		FROM order_parts p
		JOIN merchants m ON m.id = p.merchant_id
		LEFT JOIN users mo ON mo.id = m.owner_user_id
		LEFT JOIN users pu ON pu.id = p.picked_up_by
		LEFT JOIN users ru ON ru.id = p.rep_id
		LEFT JOIN merchant_settlements ms ON ms.order_id = p.order_id AND ms.merchant_id = p.merchant_id
		WHERE p.order_id = ANY($1::uuid[])
		ORDER BY p.order_id, p.seq`, ids)
	if err != nil {
		return err
	}
	type row struct {
		orderID     string
		part        OpsPart
		snap        []byte
		snapTotal   int64
		transferred bool
		inHand      bool
	}
	var got []row
	for rows.Next() {
		var r row
		p := &r.part
		if err := rows.Scan(&r.orderID, &p.MerchantID, &p.Seq, &p.Name, &p.Phone, &p.Address, &p.Lat, &p.Lng,
			&p.Status, &p.AnsweredRole, &p.AnsweredAt, &p.Reason, &p.RemovedFault,
			&p.PrepMinutes, &p.ReadyAt, &p.SentAt, &p.SentChannel, &p.ArrivedAt, &p.PickedUpAt,
			&p.PickedUpByName, &p.Collected, &r.snap, &r.snapTotal, &p.SettlementMethod,
			&p.RepName, &p.RepCommission, &r.transferred, &r.inHand); err != nil {
			rows.Close()
			return err
		}
		got = append(got, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// **بنودُ كلّ متجرٍ الحيّة** — بسعري البيع والشراء.
	irows, err := q.Query(ctx, `
		SELECT oi.order_id::text, oi.merchant_id::text, oi.name, oi.qty, oi.unit_price, oi.merchant_price,
		       oi.note, oi.options
		FROM order_items oi
		WHERE oi.order_id = ANY($1::uuid[]) AND oi.merchant_id IS NOT NULL
		ORDER BY oi.drink_for NULLS FIRST, oi.name, oi.id`, ids)
	if err != nil {
		return err
	}
	live := map[string][]OpsPartItem{}
	for irows.Next() {
		var oid, mid string
		var it OpsPartItem
		var opts []byte
		if err := irows.Scan(&oid, &mid, &it.Name, &it.Qty, &it.UnitPrice, &it.MerchantPrice, &it.Note, &opts); err != nil {
			irows.Close()
			return err
		}
		var os []OptionSnapshot
		_ = json.Unmarshal(opts, &os)
		it.Options = optionsLabel(os)
		k := oid + "|" + strings.ToLower(mid)
		live[k] = append(live[k], it)
	}
	irows.Close()
	if err := irows.Err(); err != nil {
		return err
	}

	// **وما يدفعه السائقُ لكلّ متجرٍ نقديّ** — الحسبةُ التي تعرضها شاشتُه (`PayMerchants`).
	// **ومن منفّذ النداء لا من المَسبَح** (`XG-46`) — قد يُقرأ الطلبُ داخلَ معاملة إنشائه.
	pays, err := s.on(q).PayMerchants(ctx, q, ids)
	if err != nil {
		return err
	}

	for _, r := range got {
		o := byID[r.orderID]
		p := r.part
		p.Items = []OpsPartItem{}
		closed := p.Status == PartRejected || p.Status == PartRemoved
		if closed {
			var snap []snapItem
			_ = json.Unmarshal(r.snap, &snap)
			for _, it := range snap {
				p.Items = append(p.Items, OpsPartItem{Name: it.Name, Qty: it.Qty, UnitPrice: it.UnitPrice,
					MerchantPrice: it.MerchantPrice, Options: optionsLabel(it.Options), Note: it.Note})
				p.Cost += it.MerchantPrice * int64(it.Qty)
			}
			p.Subtotal = r.snapTotal
		} else {
			p.Items = append(p.Items, live[r.orderID+"|"+strings.ToLower(p.MerchantID)]...)
			for _, it := range p.Items {
				p.Subtotal += it.UnitPrice * int64(it.Qty)
				p.Cost += it.MerchantPrice * int64(it.Qty)
			}
			for _, sp := range pays[r.orderID] {
				if strings.EqualFold(sp.MerchantID, p.MerchantID) {
					p.PayAmount = sp.Amount
				}
			}
		}
		o.Parts = append(o.Parts, p)
		if o.GoodsInHand == nil {
			v := AfterPickup(o.Status)
			o.GoodsInHand = &v
		}
		if r.inHand {
			*o.GoodsInHand = true
		}

		// **والزبونُ يرى أقسامَه بلا متجر** — وما حُوّل لا يُرى: بضاعتُه في قسمٍ آخر.
		if r.transferred {
			continue
		}
		sec := CustomerSection{Status: p.Status, Items: []CustomerSectionItem{}}
		switch p.Status {
		case PartRejected, PartRemoved:
			sec.Status = "unavailable"
		case PartPending, PartAccepted, PartPickedUp:
		default:
			continue
		}
		for _, it := range p.Items {
			sec.Items = append(sec.Items, CustomerSectionItem{Name: it.Name, Qty: it.Qty})
		}
		o.Sections = append(o.Sections, sec)
	}
	// **والأقسامُ تُرقَّم للزبون ١ ثمّ ٢** — بترتيب المحطّات، لا بأرقامٍ يتركها تحويلٌ لا يعرفه.
	for _, o := range byID {
		for i := range o.Sections {
			o.Sections[i].Seq = i + 1
		}
	}
	return nil
}
