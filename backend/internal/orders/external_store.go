package orders

// ══════════════════════════════════════════════════════════════════════
// **«من متجر غير تابع لنا»** — قرارُ المالك ٢٠٢٦-١٠-٢٩
// ══════════════════════════════════════════════════════════════════════
//
// طلبٌ عاديٌّ نقديٌّ تعذّر من متجره، والزبونُ يريده من متجرٍ ليس في المنصّة. **فيقلبه
// المكتبُ طلبَ مشتريات** — النوعُ يُقلب في مكانه (`kind='custom'`، `custom_mode='purchase'`)
// **ومن بعدها يمشي في مسار الطلب الخاصّ القائم كلِّه**:
//
//	الأجرة       تثبّتها المنصّة (`admin_defined` بلا إذنٍ للسائق) — أجرةُ الطلب الأصليّة كاملةً
//	البضاعة      يوثّق السائقُ ثمنَها بعد الشراء (`AgreeCustomStep` خطوة `goods`)
//	الزبون       يؤكّد المجموع، ثمّ يمضي الطلبُ إلى الاستلام والتسليم
//	التسوية      `settleCustomCash` — لا مستحقَّ متجرٍ ولا عمولة
//
// **ولا منطقَ ماليَّ جديد**: كلُّ مسارات المال تتفرّع على النوع أصلاً. **ولذلك لا يُقلب
// طلبٌ تحرّك فيه مال** — محفظةٌ أو تسويةُ متجرٍ أو صندوقُ سائق — فيبقى للقلب معنًى واحد:
// طلبٌ لم يُدفع فيه شيءٌ يُعاد تعريفُه.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// **أخطاءُ القلب علاماتٌ لا ردود** — الردُّ في بابه (`server/order_external_store.go`): البابُ
// للمكتب وحدَه، **فرموزُه رموزُ لوحةٍ لا تبلغ هاتفاً** ولا يُطلب لها نصٌّ في أندرويد.
var (
	// ErrExtStoreNotStandard **القلبُ للطلب العاديّ وحدَه.**
	ErrExtStoreNotStandard = errors.New("ext_store_not_standard")
	// ErrExtStoreBadStatus **بعد القبول وقبل الاستلام وحدَهما** — قبله يُقبل أو يُرفض،
	// وبعده البضاعةُ في يد السائق.
	ErrExtStoreBadStatus = errors.New("ext_store_bad_status")
	// ErrExtStoreWallet **النقدُ وحدَه** — طلبٌ دُفع من المحفظة مالُه بمسارٍ آخر.
	ErrExtStoreWallet = errors.New("ext_store_wallet")
	// ErrExtStoreMoneyMoved **مالٌ تحرّك على الطلب** — لا يُقلب ما قُيّد.
	ErrExtStoreMoneyMoved = errors.New("ext_store_money_moved")
)

// extStoreStatuses **الحالاتُ التي يُقلب فيها** — هي حالاتُ «التحويل لمتجرٍ آخر» نفسُها.
var extStoreStatuses = map[string]bool{
	StAccepted: true, StPreparing: true, StDispatching: true, StAssigned: true, StAtPickup: true,
}

// ExtStoreChange **ما وقع في القلب** — للتدقيق.
type ExtStoreChange struct {
	FromMerchantID   string `json:"from_merchant_id"`
	FromMerchantName string `json:"from_merchant_name"`
	// ItemStores **كلُّ صنفٍ ومتجرُه قبل القلب** — `order_item_id → merchant_id`.
	ItemStores map[string]string `json:"item_stores"`
	FromStatus string            `json:"from_status"`
	ToStatus   string            `json:"to_status"`
	Reason     string            `json:"reason"`
	// ReleasedPromo **الكودُ الذي عاد للزبون** — فراغٌ إن لم يكن.
	ReleasedPromo string `json:"released_promo"`
	// FeeSnapshot **الأجرةُ التي ثبّتتها المنصّة** — أجرةُ الطلب الأصليّة كاملةً.
	FeeSnapshot int64 `json:"fee_snapshot"`
	// Stores **كلُّ متجرٍ كان يرى الطلب** — متجرُه ومتاجرُ أصنافه، بلا تكرار.
	Stores []string `json:"-"`
}

// ConvertToExternalStore **يقلب طلباً عاديّاً نقديّاً طلبَ مشترياتٍ من متجرٍ غير تابع.**
//
// **والتدقيقُ في المعاملة نفسِها** (`hook`) — قلبٌ بلا أثرٍ لا يقع، وأثرٌ بلا قلبٍ لا يُكتب.
func (s *Service) ConvertToExternalStore(ctx context.Context, actorID, orderID, reason string,
	hook func(context.Context, dbtx.Querier, ExtStoreChange) error) (*Order, ExtStoreChange, error) {
	var ch ExtStoreChange
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, ch, httpx.ErrNotFound
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, ch, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	ch.Reason = reason

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, ch, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var kind, status, pay, merchantID, merchantName, promo string
	var walletPaid, fee, waived int64
	var driverID *string
	err = tx.QueryRow(ctx, `
		SELECT o.kind, o.status, o.payment_method, o.wallet_paid,
		       COALESCE(o.merchant_id::text, ''), COALESCE(m.name, ''),
		       COALESCE(o.promo_code::text, ''), o.delivery_fee, o.promo_delivery_waived,
		       o.driver_id::text
		FROM orders o LEFT JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1 FOR UPDATE OF o`, orderID).
		Scan(&kind, &status, &pay, &walletPaid, &merchantID, &merchantName, &promo,
			&fee, &waived, &driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ch, httpx.ErrNotFound
	}
	if err != nil {
		return nil, ch, err
	}
	if kind != KindStandard {
		return nil, ch, ErrExtStoreNotStandard
	}
	if !extStoreStatuses[status] {
		return nil, ch, ErrExtStoreBadStatus
	}
	if walletPaid > 0 || pay != "cash" {
		return nil, ch, ErrExtStoreWallet
	}
	// **ولا قيدَ على الطلب في أيّ دفتر** — محفظةٌ أو تسويةُ متجرٍ أو صندوقُ سائقٍ أو درجُ المكتب.
	var moved bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM wallet_transactions WHERE ref = $1::text)
		    OR EXISTS (SELECT 1 FROM merchant_settlements WHERE order_id = $1::uuid)
		    OR EXISTS (SELECT 1 FROM driver_cash_entries WHERE ref = $1::text)
		    OR EXISTS (SELECT 1 FROM office_cash_entries WHERE ref = $1::text)`,
		orderID).Scan(&moved); err != nil {
		return nil, ch, err
	}
	if moved {
		return nil, ch, ErrExtStoreMoneyMoved
	}
	ch.FromMerchantID, ch.FromMerchantName, ch.FromStatus = merchantID, merchantName, status

	// ── نصُّ الطلب من أصنافه — السائقُ يقرأ في الخاصّ هذا النصَّ وحدَه ──────
	request, itemStores, err := extStoreRequest(ctx, tx, orderID)
	if err != nil {
		return nil, ch, err
	}
	ch.ItemStores = itemStores
	seen := map[string]bool{}
	for _, id := range append([]string{merchantID}, mapValues(itemStores)...) {
		if id != "" && !seen[id] {
			seen[id] = true
			ch.Stores = append(ch.Stores, id)
		}
	}

	// ── الكودُ يعود للزبون — كما في الإلغاء حرفاً، ولا شيءَ في المحفظة ─────
	if promo != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE promo_codes SET used_count = greatest(used_count - 1, 0) WHERE code = $1`,
			promo); err != nil {
			return nil, ch, err
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM promo_redemptions WHERE order_id = $1`, orderID); err != nil {
			return nil, ch, err
		}
		ch.ReleasedPromo = promo
	}

	if _, err := tx.Exec(ctx, `UPDATE order_items SET merchant_id = NULL WHERE order_id = $1`,
		orderID); err != nil {
		return nil, ch, err
	}

	// ── الحال: قبل الطابور ⇒ الطابور، وعند المتجر ⇒ في الطريق ─────────────
	// **والخاصُّ بلا «مقبول» ولا «تحضير» ولا «عند المتجر»** (`customTransitions`).
	to := status
	switch status {
	case StAccepted, StPreparing:
		to = StDispatching
	case StAtPickup:
		to = StAssigned
	}
	ch.ToStatus = to
	// **والأجرةُ أجرةُ الطلب كاملةً** — ما دفعه الزبونُ وما أُعفي منه؛ والإعفاءُ بعرض
	// التوصيل المجانيّ يعود عند تثبيتها (`freeDeliveryForQuote`)، وبالكود ذهب مع الكود.
	ch.FeeSnapshot = fee + waived

	if _, err := tx.Exec(ctx, `
		UPDATE orders SET
		       kind = 'custom', custom_mode = 'purchase', custom_request = $2,
		       ext_store_merchant_id = merchant_id, ext_store_at = now(),
		       ext_store_by = $3::uuid, ext_store_reason = $4,
		       merchant_id = NULL,
		       partner_merchant_id = NULL, partner_commission_percent = NULL, partner_linked_at = NULL,
		       subtotal = 0, delivery_fee = 0, discount = 0, total = 0, cash_due = 0,
		       driver_fee = 0, promo_delivery_waived = 0, platform_commission = 0,
		       commission_waived = 0, commission_waived_at = NULL, commission_waived_by = NULL,
		       commission_waived_reason = NULL, sources_fee = 0,
		       promo_code = NULL,
		       custom_goods_amount = NULL, custom_fee = NULL, custom_agreed_at = NULL,
		       custom_paid_at = NULL, custom_store_paid = NULL, custom_goods_pending = false,
		       custom_reserved_amount = 0,
		       quote_version = 0, quote_confirmed_at = NULL, quote_confirmed_total = NULL,
		       quote_confirmed_version = NULL, quote_rejected_at = NULL,
		       custom_fee_source = 'admin_defined', custom_fee_snapshot = $5,
		       custom_driver_may_change_fee = false,
		       door_instruction = '', door_instruction_note = '', door_instruction_at = NULL,
		       sent_to_merchant_at = NULL, prep_minutes = NULL, ready_at = NULL,
		       pickup_override = NULL, pickup_override_note = '',
		       status = $6,
		       dispatched_at = CASE WHEN $6 = 'dispatching' AND status <> 'dispatching'
		                            THEN now() ELSE dispatched_at END,
		       updated_at = now()
		 WHERE id = $1`,
		orderID, request, actorID, reason, ch.FeeSnapshot, to); err != nil {
		return nil, ch, err
	}

	if to != status {
		if _, err := tx.Exec(ctx, `
			INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
			VALUES ($1, $2, $3, $4, '')`, orderID, status, to, actorID); err != nil {
			return nil, ch, err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
		VALUES ($1, $2, $2, $3, $4)`,
		orderID, to, actorID, "من متجرٍ غير تابع لنا — "+reason); err != nil {
		return nil, ch, err
	}

	if hook != nil {
		if err := hook(ctx, tx, ch); err != nil {
			return nil, ch, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, ch, err
	}

	// ── ما بعد التثبيت: البثّ، والطابور أو تثبيتُ الأجرة، والإشعارات ──────────
	o, err := s.GetByID(ctx, orderID)
	if err != nil {
		return nil, ch, err
	}
	s.publishOrder(o)
	// **وشاشاتُ المتاجر تُسقطه** — إشارةٌ بلا حمولة، وكلُّ متجرٍ يعيد قراءةَ قائمته.
	for _, id := range ch.Stores {
		s.pub.Publish("merchant:"+id, map[string]any{"type": "order"})
	}
	switch {
	case to == StDispatching && status != StDispatching:
		if err := s.OfferNext(ctx, orderID, nil); err != nil {
			s.logger.Error("من متجرٍ غير تابع: تعذّر أوّلُ عرض", "order", orderID, "error", err)
		}
	case driverID != nil && *driverID != "":
		s.autoAgreeFixedFee(ctx, orderID, o.CustomerID, *driverID)
		if fresh, err := s.GetByID(ctx, orderID); err == nil {
			o = fresh
			s.publishOrder(o)
		}
	}
	s.notifyExtStore(ctx, o, driverID, ch.Stores)
	return o, ch, nil
}

// extStoreRequest **نصُّ الطلب من أصنافه** — سطرٌ لكلّ صنف: «٢ × اسم — خيارات — ملاحظة».
// ويردّ متجرَ كلّ صنفٍ قبل القلب.
func extStoreRequest(ctx context.Context, q dbtx.Querier, orderID string) (string, map[string]string, error) {
	rows, err := q.Query(ctx, `
		SELECT oi.id::text, COALESCE(oi.merchant_id::text, ''), oi.qty, oi.name, oi.note,
		       COALESCE((SELECT string_agg(x->>'name', '، ' ORDER BY ord)
		                 FROM jsonb_array_elements(COALESCE(oi.options, '[]'::jsonb))
		                      WITH ORDINALITY AS t(x, ord)
		                 WHERE COALESCE(x->>'name', '') <> ''), '')
		FROM order_items oi WHERE oi.order_id = $1 ORDER BY oi.name, oi.id`, orderID)
	if err != nil {
		return "", nil, err
	}
	defer rows.Close()
	stores := map[string]string{}
	var lines []string
	for rows.Next() {
		var id, store, name, note, opts string
		var qty int
		if err := rows.Scan(&id, &store, &qty, &name, &note, &opts); err != nil {
			return "", nil, err
		}
		stores[id] = store
		line := fmt.Sprintf("%d × %s", qty, strings.TrimSpace(name))
		if opts = strings.TrimSpace(opts); opts != "" {
			line += " — " + opts
		}
		if note = strings.TrimSpace(note); note != "" {
			line += " — " + note
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return "", nil, err
	}
	return strings.Join(lines, "\n"), stores, nil
}

func mapValues(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// notifyExtStore **من يخصّه القلب يُخبَر.**
//
//   - الزبون: سيُشترى طلبُه من متجرٍ آخر، وتصله قيمةُ المشتريات ليوافق.
//   - السائق إن كان: صار الطلبُ مشتريات — يدفع للمتجر نقداً ويوثّق الثمن.
//   - المتجرُ في وضع «المتاجر تدير» وحدَه — **فهو من استلم الطلبَ في تطبيقه.** وفي
//     وضع المنصّة لم يستلمه أصلاً، فلا يُخبَر بما لم يعلم به.
func (s *Service) notifyExtStore(ctx context.Context, o *Order, driverID *string, stores []string) {
	if s.notify == nil || o == nil {
		return
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: o.CustomerID, Kind: notifications.KindOrder,
		Title:  t.extStoreCustomer,
		Body:   fmt.Sprintf("#%d — ستصلك قيمةُ المشتريات لتوافق عليها", o.Number),
		Entity: "order", EntityID: o.ID, Href: "/portal/orders",
		Apps: []string{notifications.AppCustomer},
	})
	if driverID != nil && *driverID != "" {
		s.notify.Notify(ctx, notifications.Input{
			UserID: *driverID, Kind: notifications.KindOrder,
			Title:  t.extStoreDriver,
			Body:   fmt.Sprintf("#%d — ادفع للمتجر نقداً ووثّق ثمن البضاعة", o.Number),
			Entity: "order", EntityID: o.ID, Href: "/portal",
			Apps: []string{notifications.AppDriver},
		})
	}
	if !s.MerchantsSelfManage(ctx) || len(stores) == 0 {
		return
	}
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT owner_user_id::text FROM merchants
		 WHERE id = ANY($1::uuid[]) AND owner_user_id IS NOT NULL`, stores)
	if err != nil {
		s.logger.Error("من متجرٍ غير تابع: تعذّرت قراءةُ أصحاب المتاجر", "order", o.ID, "error", err)
		return
	}
	var owners []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			owners = append(owners, id)
		}
	}
	rows.Close()
	for _, owner := range owners {
		s.notify.Notify(ctx, notifications.Input{
			UserID: owner, Kind: notifications.KindOrder,
			Title:  t.extStoreMerchant,
			Body:   fmt.Sprintf("الطلب #%d لم يعد من متجرك", o.Number),
			Entity: "order", EntityID: o.ID, Href: "/portal",
			Apps: []string{notifications.AppMerchant},
		})
	}
}
