package orders

// ══════════════════════════════════════════════════════════════════════
// **كلُّ متجرٍ يرى قسمَه وحدَه** — طلبٌ بعدّة محطّات (G-02 · G-03 · C6)
// ══════════════════════════════════════════════════════════════════════
//
// **الطلبُ واحدٌ وفيه بضاعةُ متجرين** — **ومتجرٌ لا يرى أصنافَ غيره ولا أسعارَها
// ولا مالَ الطلب كلِّه.** **فحمولةُ المتجر تُسقَط على قسمه** قبل ملء ماله وقبل
// مرشَّح الجمهور (`orderview.go`) — **والترتيبُ نفسُه في بوّابته: يُملأ ثمّ يُنقّى.**
//
//	الأصنافُ وعددُها ومعاينتُها        أصنافُه (ولقسمٍ رُفض: ما رفضه إن حُفظ)
//	المتجرُ واسمُه وشعارُه            قسمُه
//	التحضيرُ والجهوزُ والقبولُ والإرسال  أوقاتُ قسمه
//	وقتُ الوصول إلى المتجر           إن كان قسمُه المحطّةَ التالية — وإلّا فارغ
//	الحال                            حالُ قسمه حين يفترق عن الطلب (جدولُ §3.2)
//
// **وطلبُ المتجر الواحد لا يُمسّ** (`Stops = 1`) — **حمولتُه حرفاً كما كانت.**

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/media"
)

// partView **قسمٌ كما يُسقَط عليه الطلب.**
type partView struct {
	merchantID   string
	seq          int
	status       string
	answeredAt   *time.Time
	prep         *int
	readyAt      *time.Time
	sentAt       *time.Time
	arrivedAt    *time.Time
	pickedUpAt   *time.Time
	reason       string
	removedItems []byte
	updatedAt    time.Time
	name         string
	thumb        *string
}

// ProjectForStores **يُسقط الطلبَ بعدّة محطّات على أقسام هذه المتاجر** — وما سواه كما هو.
//
// `stores` متاجرُ القارئ: متجرٌ واحدٌ في سجلّه، **ومتاجرُ مالكه في الطلب الواحد**
// (فرعان في طلبٍ واحدٍ يُجاب عنهما معاً).
func (s *Service) ProjectForStores(ctx context.Context, list []Order, stores []string) error {
	var ids []string
	for i := range list {
		if list[i].Stops > 1 {
			ids = append(ids, list[i].ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	mine := map[string]bool{}
	for _, id := range stores {
		mine[strings.ToLower(id)] = true
	}

	rows, err := s.db.Query(ctx, `
		SELECT p.order_id::text, p.merchant_id::text, p.seq, p.status, p.answered_at, p.prep_minutes,
		       p.ready_at, p.sent_at, p.arrived_at, p.picked_up_at, p.reason, p.removed_items,
		       p.updated_at, m.name, lm.thumb_path
		FROM order_parts p
		JOIN merchants m ON m.id = p.merchant_id
		LEFT JOIN media lm ON lm.id = m.logo_media_id
		WHERE p.order_id = ANY($1::uuid[])
		ORDER BY p.order_id, p.seq`, ids)
	if err != nil {
		return err
	}
	parts := map[string][]partView{}
	for rows.Next() {
		var oid string
		var p partView
		if err := rows.Scan(&oid, &p.merchantID, &p.seq, &p.status, &p.answeredAt, &p.prep,
			&p.readyAt, &p.sentAt, &p.arrivedAt, &p.pickedUpAt, &p.reason, &p.removedItems,
			&p.updatedAt, &p.name, &p.thumb); err != nil {
			rows.Close()
			return err
		}
		parts[oid] = append(parts[oid], p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// **ومصدرُ كلّ بندٍ داخليٌّ لا يصل أحداً** (`OrderItem.MerchantID` بلا وسم) — فيُقرأ هنا.
	irows, err := s.db.Query(ctx, `
		SELECT oi.id::text, oi.merchant_id::text FROM order_items oi
		WHERE oi.order_id = ANY($1::uuid[]) AND oi.merchant_id IS NOT NULL`, ids)
	if err != nil {
		return err
	}
	itemStore := map[string]string{}
	for irows.Next() {
		var id, mid string
		if err := irows.Scan(&id, &mid); err != nil {
			irows.Close()
			return err
		}
		itemStore[id] = strings.ToLower(mid)
	}
	irows.Close()
	if err := irows.Err(); err != nil {
		return err
	}

	selfManage := s.MerchantsSelfManage(ctx)
	for i := range list {
		if list[i].Stops > 1 {
			projectOrder(&list[i], parts[list[i].ID], itemStore, mine, selfManage)
		}
	}
	return nil
}

// projectOrder **طلبٌ واحدٌ على أقسام قارئه.**
func projectOrder(o *Order, parts []partView, itemStore map[string]string, mine map[string]bool, selfManage bool) {
	var own []partView
	anyArrived := false
	firstLive := ""
	for _, p := range parts {
		if mine[strings.ToLower(p.merchantID)] {
			own = append(own, p)
		}
		if p.arrivedAt != nil && p.status == PartAccepted {
			anyArrived = true
		}
		if firstLive == "" && (p.status == PartPending || p.status == PartAccepted) {
			firstLive = p.merchantID
		}
	}

	items := make([]OrderItem, 0, len(o.Items))
	for _, it := range o.Items {
		if mine[itemStore[it.ID]] {
			items = append(items, it)
		}
	}
	if len(own) == 0 {
		setItems(o, items)
		return
	}
	p0 := own[0]

	// **وقسمٌ رُفض أو أُزيل يبقى في سجلّه بما رفضه** (C6) — لقطتُه إن حُفظت.
	closed := true
	for _, p := range own {
		if p.status != PartRejected && p.status != PartRemoved {
			closed = false
		}
	}
	if closed {
		var snap []OrderItem
		for _, p := range own {
			var got []OrderItem
			if len(p.removedItems) > 0 && json.Unmarshal(p.removedItems, &got) == nil {
				snap = append(snap, got...)
			}
		}
		if len(snap) > 0 {
			items = snap
		}
	}
	setItems(o, items)

	o.MerchantID = p0.merchantID
	o.MerchantName = p0.name
	o.MerchantLogoThumb = media.URLForPtr(p0.thumb)
	o.PrepMinutes = p0.prep
	o.ReadyAt = p0.readyAt
	o.SentToMerchantAt = p0.sentAt
	o.AcceptedAt = nil
	if p0.status == PartAccepted || p0.status == PartPickedUp {
		o.AcceptedAt = p0.answeredAt
	}
	if p0.pickedUpAt != nil {
		o.PickedUpAt = p0.pickedUpAt
	}
	// **وزمنُ وصول السائق لمن هو محطّتُه التالية** — لا لغيره.
	if !strings.EqualFold(p0.merchantID, firstLive) {
		o.ToStoreETASec = nil
	}

	switch p0.status {
	case PartRejected:
		o.Status, o.CancelReason, o.ClosedAt = StRejected, p0.reason, p0.answeredAt
		o.PickedUpAt, o.ReadyAt = nil, nil
	case PartRemoved:
		at := p0.updatedAt
		o.Status, o.CancelReason, o.ClosedAt = StCancelled, p0.reason, &at
		o.PickedUpAt = nil
	default:
		switch {
		case terminal(o.Status):
			// **الطلبُ انتهى — فحالُه حالُ كلّ قسمٍ فيه.**
		case o.Status == StPending && p0.status == PartAccepted:
			// **قبل قسمَه والطلبُ ينتظر غيرَه** — يراه مقبولاً (أو قيد التحضير في وضع المتاجر).
			o.Status = StAccepted
			if selfManage {
				o.Status = StPreparing
			}
		case o.Status == StAtPickup && p0.status == PartPickedUp:
			o.Status = StPickedUp
		case o.Status == StAtPickup && anyArrived && p0.arrivedAt == nil:
			// **السائقُ عند متجرٍ آخر** — في طريقه إليه.
			o.Status = StAssigned
		}
	}
	o.SetStage()
}

// setItems **أصنافُ القسم وعددُها ومعاينتُها** — كما يحسبها استعلامُ القائمة.
func setItems(o *Order, items []OrderItem) {
	o.Items = items
	n := 0
	names := make([]string, 0, len(items))
	for _, it := range items {
		n += it.Qty
		names = append(names, it.Name)
	}
	sort.Strings(names)
	if len(names) > 3 {
		names = names[:3]
	}
	o.ItemsCount = n
	o.ItemsPreview = strings.Join(names, "، ")
}

// OwnerPartStores **متاجرُ هذا المالك في أقسام الطلب** — بترتيب المحطّات.
func (s *Service) OwnerPartStores(ctx context.Context, orderID, ownerID string) ([]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT p.merchant_id::text
		FROM order_parts p JOIN merchants m ON m.id = p.merchant_id
		WHERE p.order_id = $1 AND m.owner_user_id = $2
		ORDER BY p.seq`, orderID, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// StopsOf **عددُ متاجر الطلب** — ١ لطلبٍ من متجرٍ واحد، وصفرٌ لطلبٍ لا وجودَ له.
func (s *Service) StopsOf(ctx context.Context, orderID string) int {
	var n int
	if err := s.db.QueryRow(ctx, `SELECT stops FROM orders WHERE id = $1`, orderID).Scan(&n); err != nil {
		return 0
	}
	return n
}

// PartStores **متاجرُ أقسام الطلب بترتيب المحطّات** — وفارغٌ لطلب المتجر الواحد.
func (s *Service) PartStores(ctx context.Context, orderID string) []string {
	ids, err := partStoresOf(ctx, s.db, orderID)
	if err != nil {
		return nil
	}
	return ids
}

// NextUnsentPart **قسمُ أيّ متجرٍ تخصّه التذكرة** — المطلوبُ إن كان له قسمٌ في الطلب،
// **وإلّا أوّلُ قسمٍ مقبولٍ لم يُبلَّغ**، ثمّ أوّلُ قسمٍ حيّ. **وفارغٌ: لا قسمَ بهذا المتجر.**
//
// **وبابُ اللوحة القديمُ يرسل بلا متجر** — فيذهب إلى أوّل من ينتظر تذكرتَه.
func (s *Service) NextUnsentPart(ctx context.Context, orderID, want string) string {
	var id string
	if err := s.db.QueryRow(ctx, `
		SELECT merchant_id::text FROM order_parts
		WHERE order_id = $1 AND ($2 = '' OR merchant_id::text = lower($2))
		ORDER BY (status = 'accepted' AND sent_at IS NULL) DESC,
		         (status IN ('pending', 'accepted')) DESC, seq
		LIMIT 1`, orderID, want).Scan(&id); err != nil {
		return ""
	}
	return id
}
