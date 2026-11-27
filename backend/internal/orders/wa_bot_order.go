package orders

// ══════════════════════════════════════════════════════════════════════
// **طلبُ واتساب من البوت — بانتظار موافقة الإدارة** (قرارُ المالك ٢٠٢٦-١١-٢٧)
// ══════════════════════════════════════════════════════════════════════
//
// «طلب الواتساب لما يجهز بينزل طلب بانتظار موافقة الإدارة». الزبونُ يكتب طلبَه للبوت ويبعث موقعه
// (`server/wa_bot_menu.go`) ⇒ **طلبُ واتساب** (`origin='whatsapp'`) بعلامة `wa_bot` وحالُه `pending`.
//
//   - **لا أجرةَ عليه بعد** — الموظّفُ يكتبها عند الموافقة (`ApproveWABotOrder`).
//   - **لا يبلغ الكباتن قبل الموافقة**: القبولُ التلقائيُّ والنزولُ بلا واتساب يتخطّيانه، **والآلةُ
//     تردّ قبولَه ما دامت أجرتُه صفراً** (`ErrWAFeeRequired`) — فزرُّ القبول العامّ لا يلتفّ عليه.
//   - **والرفضُ** من الطابور نفسِه (`pending → rejected` للمكتب).

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// WABotNoStore **اسمُ المتجر حين كتب الزبونُ «لا يوجد»** — والقيدُ يشترط اسماً للاستلام.
const WABotNoStore = "غير محدّد"

// waBotBirthNote **سطرُ الميلاد في سجلّ الطلب.**
const waBotBirthNote = "طلب واتساب من البوت — بانتظار موافقة الإدارة"

var (
	// ErrWABotNotPending **الطلبُ ليس طلبَ بوتٍ معلّقاً** — وُوفق عليه أو رُفض أو أُلغي.
	ErrWABotNotPending = errors.New("wa_bot_not_pending")
)

// WABotOrderInput **ما جمعه البوت.**
type WABotOrderInput struct {
	// Phone **رقمُ المرسِل مُطبَّعاً** — رقمُ الزبون على الطلب.
	Phone string
	// Details **نصُّ الطلب والتعديلات كاملاً.**
	Details string
	// StoreName **اسمُ المتجر إن كتبه** — وفارغٌ «لا يوجد».
	StoreName string
	Lat, Lng  float64
}

// WABotCreated **رقمُ الطلب ومعرّفُه.**
type WABotCreated struct {
	ID     string
	Number int64
}

// CreateWABotOrder **يكتب طلبَ البوت معلّقاً** — بنقطة الزبون، وبلا أجرة.
//
// **والمنطقةُ منطقةُ نقطته إن كانت في التغطية**، وإلّا بلا منطقة — والموظّفُ يقرّر (يرفضه أو يوافق).
// **ولا يسقط الطلبُ لنقطةٍ خارجَ التغطية**: الزبونُ كتب طلبَه، والإدارةُ أعلم.
func (s *Service) CreateWABotOrder(ctx context.Context, in WABotOrderInput) (*WABotCreated, error) {
	in.Phone = strings.TrimSpace(in.Phone)
	in.Details = strings.TrimSpace(in.Details)
	in.StoreName = strings.TrimSpace(in.StoreName)
	if in.Phone == "" || in.Details == "" || !ValidPoint(in.Lat, in.Lng) {
		return nil, ErrBadPoint
	}
	if r := []rune(in.Details); len(r) > MaxWhatsAppDetails {
		in.Details = string(r[:MaxWhatsAppDetails])
	}
	if r := []rune(in.StoreName); len(r) > 120 {
		in.StoreName = string(r[:120])
	}
	if in.StoreName == "" {
		in.StoreName = WABotNoStore
	}
	out := &WABotCreated{}
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		var zoneID *string
		if z, err := s.RequireServiceable(ctx, tx, in.Lat, in.Lng); err == nil && z.ID != "" {
			zoneID = &z.ID
		}
		snap, err := s.snapshotNow(ctx, tx)
		if err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO orders (kind, origin, wa_bot, pickup_name, address_text, dropoff, zone_id,
			                    recipient_name, recipient_phone, parcel_note, fee_payer,
			                    status, payment_method, subtotal, delivery_fee, driver_fee, total, cash_due,
			                    snap_merchant_commission_percent, snap_rep_commission_percent,
			                    snap_commission_source, snap_activation_orders,
			                    dropoff_known, delivery_mode, money_model)
			VALUES ('merchant_delivery', 'whatsapp', true, $1, '',
			        ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, $4,
			        $5, $6, $7, 'recipient',
			        'pending', 'cash', 0, 0, 0, 0, 0,
			        $8, $9, $10, $11,
			        true, 'send', $12)
			RETURNING id::text, number`,
			in.StoreName, in.Lat, in.Lng, zoneID,
			WhatsAppRecipientName, in.Phone, in.Details,
			snap.MerchantCommission, snap.RepCommissionPercent, snap.CommissionSource, snap.ActivationOrders,
			s.newOrderMoneyModel(ctx, tx)).Scan(&out.ID, &out.Number); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
			VALUES ($1, '', 'pending', NULL, $2)`, out.ID, waBotBirthNote)
		return err
	})
	if err != nil {
		return nil, err
	}
	if o, err := s.GetByID(ctx, out.ID); err == nil {
		s.publishOrder(o)
	}
	return out, nil
}

// ApproveWABotOrder **الموظّفُ يكتب الأجرةَ ويوافق** ⇒ مقبولٌ ثمّ الطابور (فالكباتن).
//
// **الأجرةُ بحدّي أجرة الطلب الخاصّ وحصّةِ «طلب واتساب»** كما في نموذج الواتساب، **واسمُ المتجر
// يُصحَّح إن كُتب.** والتدقيقُ (`audit`) في معاملة الأجرة نفسِها.
func (s *Service) ApproveWABotOrder(ctx context.Context, actorID, orderID string, fee int64, storeName string,
	audit func(ctx context.Context, q pgx.Tx) error) (*Order, error) {
	storeName = strings.TrimSpace(storeName)
	if r := []rune(storeName); len(r) > 120 {
		storeName = string(r[:120])
	}
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		var bot bool
		var status string
		err := tx.QueryRow(ctx, `SELECT wa_bot, status FROM orders WHERE id = $1 FOR UPDATE`, orderID).
			Scan(&bot, &status)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && (!bot || status != StPending)) {
			return ErrWABotNotPending
		}
		if err != nil {
			return err
		}
		if fee <= 0 {
			return ErrWAFeeRequired
		}
		lo, hi := s.whatsAppFeeBounds(ctx, tx)
		if fee < lo || (hi > 0 && fee > hi) {
			return &WAFeeRangeError{Min: lo, Max: hi}
		}
		pct, fixed := s.platformShare(ctx, tx, ShareWhatsApp)
		if fixed > 0 && fixed >= fee {
			return ErrWAFeeBelowShare
		}
		driverFee := DriverFeeWithShare(fee, pct, fixed)
		if _, err := tx.Exec(ctx, `
			UPDATE orders SET delivery_fee = $2, total = $2, cash_due = $2, driver_fee = $3,
			       snap_platform_delivery_percent = $4, snap_platform_delivery_fixed = $5,
			       pickup_name = COALESCE(NULLIF($6, ''), pickup_name)
			 WHERE id = $1`, orderID, fee, driverFee, pct, fixed, storeName); err != nil {
			return err
		}
		if audit != nil {
			return audit(ctx, tx)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	o, err := s.Transition(ctx, actorID, []string{"ops"}, orderID, StAccepted, waBotApproveNote)
	if err != nil {
		return nil, err
	}
	// **والقبولُ يُنزله بنفسه إن اشتعل الإنزالُ التلقائيّ** — وإلّا يُنزله هنا: الموافقةُ موافقةٌ على الإرسال.
	if o.Status == StAccepted {
		if d, err := s.Transition(ctx, actorID, []string{"ops"}, orderID, StDispatching, waBotApproveNote); err == nil {
			o = d
		} else {
			s.logger.Warn("طلب البوت: الإنزالُ بعد الموافقة تعثّر", "order", orderID, "error", err)
		}
	}
	return o, nil
}

// waBotApproveNote **سطرُ الموافقة في السجلّ.**
const waBotApproveNote = "موافقة الإدارة على طلب البوت"

// RejectWABotOrder **الموظّفُ يرفض طلبَ البوت المعلّق** — بسببٍ إلزاميّ يُكتب في السجلّ.
func (s *Service) RejectWABotOrder(ctx context.Context, actorID, orderID, reason string) (*Order, error) {
	var bot bool
	var status string
	err := s.db.QueryRow(ctx, `SELECT wa_bot, status FROM orders WHERE id = $1`, orderID).Scan(&bot, &status)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (!bot || status != StPending)) {
		return nil, ErrWABotNotPending
	}
	if err != nil {
		return nil, err
	}
	return s.Transition(ctx, actorID, []string{"ops"}, orderID, StRejected, strings.TrimSpace(reason))
}

// WABotTrack **حالُ الطلب برقمه — لصاحبه وحدَه** (قرارُ المالك ٢٠٢٦-١١-٢٧).
//
// **صاحبُه**: رقمُ المستلِم على الطلب، أو رقمُ حساب الزبون. **وغيرُه «لم نجد»** — لا يُقال إنّ الرقمَ
// موجودٌ لغيره، فلا تُكشف طلباتُ الناس بالتخمين.
func (s *Service) WABotTrack(ctx context.Context, phone string, number int64) (status string, found bool, err error) {
	err = s.db.QueryRow(ctx, `
		SELECT o.status FROM orders o
		  LEFT JOIN users u ON u.id = o.customer_id
		 WHERE o.number = $1 AND (o.recipient_phone = $2 OR u.phone = $2)
		 LIMIT 1`, number, phone).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return status, true, nil
}
