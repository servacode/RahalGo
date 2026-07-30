// Package orders محرك الطلبات: الإنشاء بتسعير خادمي، آلة الحالات، والتسويات المالية.
package orders

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// Publisher واجهة البث الحي — ينشر المحرك تحديثات الطلبات عبرها.
type Publisher interface {
	Publish(topic string, event any)
}

type noopPublisher struct{}

func (noopPublisher) Publish(string, any) {}

type Service struct {
	db       *pgxpool.Pool
	identity *identity.Service
	wallet   *wallet.Service
	cashbox  *cashbox.Service
	pub      Publisher
	logger   *slog.Logger
}

func NewService(db *pgxpool.Pool, identitySvc *identity.Service, walletSvc *wallet.Service,
	cashboxSvc *cashbox.Service, pub Publisher, logger *slog.Logger) *Service {
	if pub == nil {
		pub = noopPublisher{}
	}
	return &Service{db: db, identity: identitySvc, wallet: walletSvc, cashbox: cashboxSvc, pub: pub, logger: logger}
}

// publishOrder يبث ملخص الطلب لغرفة العمليات.
func (s *Service) publishOrder(o *Order) {
	if o == nil {
		return
	}
	s.pub.Publish("ops", map[string]any{"type": "order", "order": o})
}

// Create ينشئ طلباً كاملاً: تحقق المتجر، تسعير خادمي للأصناف والخيارات،
// منطقة التسليم ورسمها، كود الخصم، ثم الدفع (نقدي/محفظة/مختلط) — كله ذرّياً.
func (s *Service) Create(ctx context.Context, actorID string, actorRoles []string, in CreateInput, ip string) (*Order, error) {
	if len(in.Items) == 0 || in.AddressText == "" || in.MerchantID == "" {
		return nil, ErrBadItems
	}
	switch in.PaymentMethod {
	case "", "cash":
		in.PaymentMethod = "cash"
	case "wallet", "mixed":
	default:
		return nil, ErrBadItems
	}

	// الزبون: معرف مباشر أو رقم هاتف (طلب هاتفي — يُنشأ الحساب إن لزم)
	customerID := in.CustomerID
	if customerID == "" {
		if in.CustomerPhone == "" {
			return nil, ErrBadItems
		}
		u, err := s.identity.EnsureUserWithRole(ctx, actorID, in.CustomerPhone, "customer", ip)
		if err != nil {
			return nil, err
		}
		customerID = u.ID
	}

	// المتجر فعّال وغير مغلق طارئاً
	var merchantActive bool
	var emergencyClosed bool
	err := s.db.QueryRow(ctx,
		`SELECT status = 'active', emergency_closed FROM merchants WHERE id = $1`,
		in.MerchantID).Scan(&merchantActive, &emergencyClosed)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !merchantActive || emergencyClosed {
		return nil, ErrMerchantClosed
	}

	// التسعير الخادمي للأصناف والخيارات (لقطة ثابتة)
	items, subtotal, err := s.priceItems(ctx, in.MerchantID, in.Items)
	if err != nil {
		return nil, err
	}

	// منطقة التسليم من الدبوس
	var zoneID, zoneName string
	var deliveryFee, minOrder int64
	err = s.db.QueryRow(ctx, `
		SELECT id, name, delivery_fee, min_order FROM delivery_zones
		WHERE active AND ST_DWithin(center, ST_SetSRID(ST_MakePoint($2,$1),4326)::geography, radius_m)
		ORDER BY ST_Distance(center, ST_SetSRID(ST_MakePoint($2,$1),4326)::geography)
		LIMIT 1`, in.Lat, in.Lng).Scan(&zoneID, &zoneName, &deliveryFee, &minOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOutOfZone
	}
	if err != nil {
		return nil, err
	}
	if subtotal < minOrder {
		return nil, ErrBelowMinOrder
	}

	// كود الخصم
	var promoID *string
	var discount int64
	promoCode := strings.TrimSpace(strings.ToUpper(in.PromoCode))
	if promoCode != "" {
		promoID, discount, err = s.validatePromo(ctx, promoCode, customerID, subtotal, &deliveryFee)
		if err != nil {
			return nil, err
		}
	}

	total := subtotal - discount + deliveryFee
	if total < 0 {
		total = 0
	}

	// توزيع الدفع
	var walletPaid int64
	switch in.PaymentMethod {
	case "wallet":
		walletPaid = total
	case "mixed":
		balance, err := s.wallet.Balance(ctx, customerID)
		if err != nil {
			return nil, err
		}
		walletPaid = min64(balance, total)
	}
	cashDue := total - walletPaid

	// الإنشاء الذرّي
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var orderID string
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, address_text, dropoff, zone_id,
			payment_method, subtotal, delivery_fee, discount, total, wallet_paid, cash_due,
			promo_code, notes, created_by)
		VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($5,$4),4326)::geography, $6,
			$7, $8, $9, $10, $11, $12, $13, NULLIF($14,''), $15, $16)
		RETURNING id`,
		customerID, in.MerchantID, in.AddressText, in.Lat, in.Lng, zoneID,
		in.PaymentMethod, subtotal, deliveryFee, discount, total, walletPaid, cashDue,
		promoCode, in.Notes, actorID).Scan(&orderID)
	if err != nil {
		return nil, err
	}

	for _, it := range items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO order_items (order_id, menu_item_id, name, unit_price, qty, note, options)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			orderID, it.MenuItemID, it.Name, it.UnitPrice, it.Qty, it.Note,
			marshalOptions(it.Options)); err != nil {
			return nil, err
		}
	}

	if promoID != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO promo_redemptions (promo_id, order_id, user_id) VALUES ($1, $2, $3)`,
			*promoID, orderID, customerID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE promo_codes SET used_count = used_count + 1 WHERE id = $1`, *promoID); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
		VALUES ($1, '', 'pending', $2, '')`, orderID, actorID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// خصم المحفظة بعد نجاح الإنشاء (حركة مدققة بمرجع الطلب)
	if walletPaid > 0 {
		if _, err := s.wallet.Apply(ctx, customerID, -walletPaid, "order_payment",
			orderID, fmt.Sprintf("دفع طلب #%s", orderID[:8]), &actorID); err != nil {
			// فشل الخصم (رصيد تغير لحظياً) — نلغي الطلب فوراً بدل تركه معلقاً
			_, _ = s.db.Exec(ctx, `
				UPDATE orders SET status='cancelled', cancel_reason='wallet_charge_failed',
					closed_at=now(), updated_at=now() WHERE id=$1`, orderID)
			return nil, err
		}
	}

	created, err := s.GetByID(ctx, orderID)
	if err == nil {
		s.publishOrder(created)
	}
	return created, err
}

// priceItems يجلب الأسعار الحقيقية من القائمة ويتحقق من الخيارات وقيود المجموعات.
func (s *Service) priceItems(ctx context.Context, merchantID string, inputs []ItemInput) ([]OrderItem, int64, error) {
	items := make([]OrderItem, 0, len(inputs))
	var subtotal int64

	for _, in := range inputs {
		if in.Qty < 1 || in.Qty > 50 {
			return nil, 0, ErrBadItems
		}
		var it OrderItem
		var available bool
		err := s.db.QueryRow(ctx, `
			SELECT id, name, price, available FROM menu_items
			WHERE id = $1 AND merchant_id = $2`, in.MenuItemID, merchantID).
			Scan(&it.MenuItemID, &it.Name, &it.UnitPrice, &available)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, ErrBadItems
		}
		if err != nil {
			return nil, 0, err
		}
		if !available {
			return nil, 0, ErrItemUnavailable
		}
		it.Qty = in.Qty
		it.Note = in.Note
		it.Options = []OptionSnapshot{}

		// الخيارات: يجب أن تتبع مجموعات هذا الصنف وتحترم أدنى/أقصى اختيار
		type groupRule struct{ min, max, chosen int }
		rules := map[string]*groupRule{}
		gRows, err := s.db.Query(ctx,
			`SELECT id, min_select, max_select FROM modifier_groups WHERE item_id = $1`, in.MenuItemID)
		if err != nil {
			return nil, 0, err
		}
		for gRows.Next() {
			var id string
			var mn, mx int
			if err := gRows.Scan(&id, &mn, &mx); err != nil {
				gRows.Close()
				return nil, 0, err
			}
			rules[id] = &groupRule{min: mn, max: mx}
		}
		gRows.Close()

		for _, optID := range in.OptionIDs {
			var groupID, groupName, optName string
			var delta int64
			var optAvailable bool
			err := s.db.QueryRow(ctx, `
				SELECT g.id, g.name, o.name, o.price_delta, o.available
				FROM modifier_options o
				JOIN modifier_groups g ON g.id = o.group_id
				WHERE o.id = $1 AND g.item_id = $2`, optID, in.MenuItemID).
				Scan(&groupID, &groupName, &optName, &delta, &optAvailable)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, 0, ErrBadItems
			}
			if err != nil {
				return nil, 0, err
			}
			if !optAvailable {
				return nil, 0, ErrItemUnavailable
			}
			rules[groupID].chosen++
			it.UnitPrice += delta
			it.Options = append(it.Options, OptionSnapshot{Group: groupName, Name: optName, PriceDelta: delta})
		}
		for _, r := range rules {
			if r.chosen < r.min || r.chosen > r.max {
				return nil, 0, ErrBadItems
			}
		}

		subtotal += it.UnitPrice * int64(it.Qty)
		items = append(items, it)
	}
	return items, subtotal, nil
}

// validatePromo يتحقق من كل قواعد الكود ويعيد الخصم (وقد يصفّر رسم التوصيل).
func (s *Service) validatePromo(ctx context.Context, code, customerID string, subtotal int64, deliveryFee *int64) (*string, int64, error) {
	var id, kind string
	var value, minOrder int64
	var firstOnly, oncePerUser, active bool
	var maxUses *int
	var usedCount int
	var expiresAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT id, kind, value, min_order, first_order_only, once_per_user,
		       max_uses, used_count, expires_at, active
		FROM promo_codes WHERE code = $1`, code).
		Scan(&id, &kind, &value, &minOrder, &firstOnly, &oncePerUser,
			&maxUses, &usedCount, &expiresAt, &active)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, ErrInvalidPromo
	}
	if err != nil {
		return nil, 0, err
	}

	switch {
	case !active,
		expiresAt != nil && time.Now().After(*expiresAt),
		maxUses != nil && usedCount >= *maxUses,
		subtotal < minOrder:
		return nil, 0, ErrInvalidPromo
	}

	if oncePerUser {
		var used bool
		if err := s.db.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM promo_redemptions WHERE promo_id = $1 AND user_id = $2)`,
			id, customerID).Scan(&used); err != nil {
			return nil, 0, err
		}
		if used {
			return nil, 0, ErrInvalidPromo
		}
	}
	if firstOnly {
		var hasOrders bool
		if err := s.db.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM orders WHERE customer_id = $1
				AND status NOT IN ('cancelled','rejected','failed'))`,
			customerID).Scan(&hasOrders); err != nil {
			return nil, 0, err
		}
		if hasOrders {
			return nil, 0, ErrInvalidPromo
		}
	}

	var discount int64
	switch kind {
	case "percent":
		discount = subtotal * value / 100
	case "fixed":
		discount = min64(value, subtotal)
	case "free_delivery":
		*deliveryFee = 0
	}
	return &id, discount, nil
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
