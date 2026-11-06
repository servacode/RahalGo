package orders

// ══════════════════════════════════════════════════════════════════════
// **محطّاتُ السائق — وصولٌ واستلامٌ لكلّ متجر** (طلبٌ بعدّة محطّات، الدفعة P4)
// ══════════════════════════════════════════════════════════════════════
//
// **السائقُ يقول «وصلت» و«استلمت» لكلّ محطّةٍ باسمها** (`stop` = معرّفُ المتجر):
//
//	{to: at_pickup, stop}  ⇒ `StopArrive`  — أوّلُ وصولٍ ينقل الطلبَ `assigned → at_pickup`
//	{to: picked_up, stop}  ⇒ `StopPickup`  — يُسوّى متجرُ المحطّة وحدَه (قرارُ المالك ٣أ)،
//	                                         وآخرُ محطّةٍ تنقل الطلبَ `at_pickup → picked_up`
//
// **وبلا `stop` ⇒ ٤٢٦** (G3): التطبيقُ القديمُ كان يخمّن المحطّة، **وإعادةُ «استلمت» بعد ضياع
// ردّها تستلم المحطّةَ التالية والسائقُ عند الأولى.** **والإعادةُ بالمحطّة نفسِها لا تكتب شيئاً.**
//
// # البضاعةُ في اليد (G5)
//
// **من استلم محطّةً لا يترك الباقي** — لا تحرير ولا إلغاء ولا إفشال (`ErrStopsInHand`)،
// **والمكتبُ وحدَه يقرّر**: يرسل سائقاً آخرَ يأخذ البضاعةَ منه («محطّةُ تسليم»، ٥أ) أو يشيل
// المحطّاتِ الباقية فيمضي الطلبُ بما استُلم.
//
// # والفعلُ قسمٌ في معاملةٍ واحدة
//
// **كأفعال الأقسام كلِّها** (`partAction`): قفلٌ ففعلٌ فانتقالٌ تابعٌ بآلة الحال نفسِها — `XG-46`.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	// ErrStopsInHand **السائقُ يحمل بضاعةَ محطّة — المكتبُ يقرّر** (G5).
	ErrStopsInHand = httpx.NewError(http.StatusConflict, "stops_in_hand", "errors.stops_in_hand")
	// ErrStopNotArrived **«استلمت» قبل «وصلت» لهذه المحطّة.**
	ErrStopNotArrived = httpx.NewError(http.StatusConflict, "stop_not_arrived", "errors.stop_not_arrived")
	// ErrStopNotLive **المحطّةُ انشالت من الطلب.**
	ErrStopNotLive = httpx.NewError(http.StatusConflict, "stop_not_live", "errors.stop_not_live")
	// ErrStopRequired **خطوةُ محطّةٍ بلا محطّة** — تطبيقٌ قديم (G3): ٤٢٦ كبوّابة التحديث.
	ErrStopRequired = httpx.NewError(http.StatusUpgradeRequired, "update_required", "errors.update_required")
)

// goodsInHandTx **أبيد سائق الطلب بضاعة؟** — بعد الاستلام، **أو استلم هو محطّةً من طلبٍ بعدّة محطّات.**
func goodsInHandTx(ctx context.Context, q dbtx.Querier, orderID string) (bool, error) {
	var in bool
	err := q.QueryRow(ctx, `
		SELECT o.status IN ('picked_up', 'on_the_way', 'at_dropoff')
		    OR (o.stops > 1 AND o.driver_id IS NOT NULL AND EXISTS (
		        SELECT 1 FROM order_parts p WHERE p.order_id = o.id
		          AND p.status = 'picked_up' AND p.picked_up_by = o.driver_id))
		FROM orders o WHERE o.id = $1`, orderID).Scan(&in)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, httpx.ErrNotFound
	}
	return in, err
}

// GoodsInHand **أبيد السائق بضاعةُ هذا الطلب؟** (§5.3) — يقرؤه الطارئُ والتركُ وغرفةُ الطوارئ
// بدل الحال وحدَها. **والخطأُ «لا»** — والآلةُ تحرس بنفسها (G5).
func (s *Service) GoodsInHand(ctx context.Context, orderID string) bool {
	in, err := goodsInHandTx(ctx, s.db, orderID)
	return err == nil && in
}

// stopCollectedAnyTx **أاستُلمت محطّةٌ من الطلب — بيد أيّ سائق؟**
func stopCollectedAnyTx(ctx context.Context, q dbtx.Querier, orderID string) (bool, error) {
	var got bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM order_parts
		WHERE order_id = $1 AND status = 'picked_up')`, orderID).Scan(&got)
	return got, err
}

// stopGatesTx **بوّاباتُ المحطّات في الآلة** — للطلب بعدّة محطّات وحدَه:
//
//	G3  السائقُ إلى at_pickup/picked_up بلا محطّة (لا من فعل قسم)  ⇒ ٤٢٦ `update_required`
//	G5  البضاعةُ بيد السائق والطلبُ عند المحطّات ⇒ لا طابورَ ولا إلغاءَ ولا إفشال (`stops_in_hand`)
//	    **إلّا «كمّل بسائق تاني» من غرفة الطوارئ** (`handover`). **وبضاعةُ سائقٍ سابقٍ لم تُسلَّم
//	    بعدُ للجديد** تمنع الإلغاءَ والإفشال أيضاً: سلفتُه خرجت ولا يُغلق طلبٌ لم يُقل أين بضاعتُه.
//	G8  «المتجرُ مغلق» عند المحطّة وفي الطلب قسمٌ حيٌّ غيرُه ⇒ `use_part` — المكتبُ يشيل القسمَ أو يحوّله
func (s *Service) stopGatesTx(ctx context.Context, tx pgx.Tx, lo *lockedOrder, req transitionReq, to string) error {
	from := lo.from
	if !req.fromParts && (to == StAtPickup || to == StPickedUp) && slices.Contains(req.actorRoles, "driver") &&
		!slices.Contains(req.actorRoles, "ops") && !slices.Contains(req.actorRoles, "admin") {
		return ErrStopRequired
	}
	if (from == StDispatching || from == StAssigned || from == StAtPickup) &&
		(to == StDispatching || to == StFailed || to == StCancelled) {
		inHand, err := goodsInHandTx(ctx, tx, lo.id)
		if err != nil {
			return err
		}
		if inHand && !(req.handover && to == StDispatching) {
			return ErrStopsInHand
		}
		if to != StDispatching {
			if collected, err := stopCollectedAnyTx(ctx, tx, lo.id); err != nil {
				return err
			} else if collected {
				return ErrStopsInHand
			}
		}
	}
	if from == StAtPickup && to == StFailed {
		var live int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM order_parts
			WHERE order_id = $1 AND status IN ('pending', 'accepted', 'picked_up')`, lo.id).Scan(&live); err != nil {
			return err
		}
		if live > 1 {
			return ErrUsePart
		}
	}
	return nil
}

// stopsAfterTransitionTx **ما يتبع الانتقالَ في المحطّات**: **كلُّ عودةٍ إلى الطابور تمحو الوصول**
// (`arrived_at` للسائق الحاليّ) — فيصل السائقُ التالي من جديد، ولو إلى محطّة تسليم.
func stopsAfterTransitionTx(ctx context.Context, q dbtx.Querier, orderID, to string) error {
	if to != StDispatching {
		return nil
	}
	_, err := q.Exec(ctx, `UPDATE order_parts SET arrived_at = NULL, updated_at = now()
		WHERE order_id = $1 AND arrived_at IS NOT NULL`, orderID)
	return err
}

// stopRow **محطّةٌ مقفولة.**
type stopRow struct {
	status, name, pickedBy string
	arrived                *time.Time
}

func lockStopTx(ctx context.Context, q dbtx.Querier, orderID, store string) (stopRow, error) {
	var r stopRow
	err := q.QueryRow(ctx, `
		SELECT p.status, m.name, COALESCE(p.picked_up_by::text, ''), p.arrived_at
		FROM order_parts p JOIN merchants m ON m.id = p.merchant_id
		WHERE p.order_id = $1 AND p.merchant_id = $2::uuid
		FOR UPDATE OF p`, orderID, store).Scan(&r.status, &r.name, &r.pickedBy, &r.arrived)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, httpx.ErrNotFound
	}
	return r, err
}

// normStop **معرّفُ المحطّة كما يُقارَن** — ومعرّفٌ ليس بصيغته «لا محطّة» لا خطأَ قاعدة.
func normStop(stop string) (string, error) {
	stop = strings.ToLower(strings.TrimSpace(stop))
	if stop == "" {
		return "", ErrStopRequired
	}
	if len(stop) != 36 || strings.Count(stop, "-") != 4 {
		return "", httpx.ErrNotFound
	}
	return stop, nil
}

func holds(lo *lockedOrder, driverID string) bool {
	return lo.driverID != nil && strings.EqualFold(*lo.driverID, driverID)
}

// StopArrive **وصل السائقُ إلى محطّة** — `arrived_at` لها، **وأوّلُ وصولٍ ينقل الطلبَ إلى «عند
// المتجر».** **ووصولٌ ثانٍ إلى المحطّة نفسِها لا يكتب شيئاً.** وأمرُ الإدارة عند الباب يُمحى:
// كلُّ وقفةٍ وقفةٌ جديدة.
func (s *Service) StopArrive(ctx context.Context, driverID, orderID, stop string) (*Order, error) {
	stop, err := normStop(stop)
	if err != nil {
		return nil, err
	}
	updated, _, err := s.partAction(ctx, orderID, func(ctx context.Context, tx pgx.Tx, unit *Service,
		lo *lockedOrder, out *partOutcome) error {
		if !holds(lo, driverID) {
			return ErrBadTransition
		}
		p, err := lockStopTx(ctx, tx, lo.id, stop)
		if err != nil {
			return err
		}
		if p.status == PartRemoved || p.status == PartRejected {
			return ErrStopNotLive
		}
		if p.arrived != nil || (p.status == PartPickedUp && strings.EqualFold(p.pickedBy, driverID)) {
			out.noop = true
			return nil
		}
		if lo.from != StAssigned && lo.from != StAtPickup {
			return ErrBadTransition
		}
		if _, err := tx.Exec(ctx, `UPDATE order_parts SET arrived_at = now(), updated_at = now()
			WHERE order_id = $1 AND merchant_id = $2::uuid`, lo.id, stop); err != nil {
			return err
		}
		note := fmt.Sprintf("وصل السائق إلى %s", p.name)
		if p.status == PartPickedUp {
			note = fmt.Sprintf("وصل السائق لاستلام بضاعة %s من السائق السابق", p.name)
		}
		if err := partEventTx(ctx, tx, lo.id, lo.from, driverID, note, lo.driverID, stop); err != nil {
			return err
		}
		if lo.from == StAssigned {
			out.follow, err = unit.applyTransitionTx(ctx, tx, lo, transitionReq{
				actorID: driverID, actorRoles: []string{"driver"}, to: StAtPickup, fromParts: true,
			})
			return err
		}
		// **وكلُّ محطّةٍ وقفةٌ جديدة** — «سلّم الآن» عند الأولى لا تُقرأ عند الثانية.
		_, err = tx.Exec(ctx, `UPDATE orders SET door_instruction = '', door_instruction_note = '',
			door_instruction_at = NULL, updated_at = now() WHERE id = $1`, lo.id)
		return err
	})
	return updated, err
}

// StopPickup **استلم السائقُ من محطّة** — يُسوّى متجرُها وحدَه في اللحظة (قرارُ المالك ٣أ):
// مستحقُّ المحفظة، أو المستحقُّ النقديُّ يدفعه السائقُ من سلفته (`payCashByDriver`)، **ونسبةُ
// العمولة تُجمَّد في صفّها** (PD-1). **ومحطّةُ تسليمٍ** (بضاعةٌ استلمها سائقٌ سابق) **تنقل
// الحاملَ وحدَه بلا مال** — مالُها قُيّد عند الأوّل.
//
// **وحين يستلم السائقُ آخرَ محطّةٍ حيّة** ينتقل الطلبُ إلى `picked_up` في المعاملة نفسِها —
// كودُ التسليم وزمنُ الطريق إلى الباب والتسويةُ الأخيرةُ تتحقّق من صفوف المحطّات.
func (s *Service) StopPickup(ctx context.Context, driverID, orderID, stop string) (*Order, error) {
	stop, err := normStop(stop)
	if err != nil {
		return nil, err
	}
	updated, out, err := s.partAction(ctx, orderID, func(ctx context.Context, tx pgx.Tx, unit *Service,
		lo *lockedOrder, out *partOutcome) error {
		if !holds(lo, driverID) {
			return ErrBadTransition
		}
		p, err := lockStopTx(ctx, tx, lo.id, stop)
		if err != nil {
			return err
		}
		if p.status == PartRemoved || p.status == PartRejected {
			return ErrStopNotLive
		}
		if p.status == PartPickedUp && strings.EqualFold(p.pickedBy, driverID) {
			out.noop = true
			return nil
		}
		switch {
		case lo.from == StAssigned:
			return ErrStopNotArrived
		case lo.from != StAtPickup:
			return ErrBadTransition
		case p.arrived == nil:
			return ErrStopNotArrived
		}
		if p.status == PartPickedUp {
			// **محطّةُ تسليم** — الحاملُ يتبدّل وحدَه، **والمالُ قُيّد عند الأوّل.**
			if _, err := tx.Exec(ctx, `UPDATE order_parts SET picked_up_by = $3::uuid, updated_at = now()
				WHERE order_id = $1 AND merchant_id = $2::uuid`, lo.id, stop, driverID); err != nil {
				return err
			}
			if err := partEventTx(ctx, tx, lo.id, lo.from, driverID,
				fmt.Sprintf("استلم السائق بضاعة %s من السائق السابق", p.name), lo.driverID, stop); err != nil {
				return err
			}
		} else {
			if p.status != PartAccepted {
				return ErrBadTransition
			}
			if err := unit.settleMerchantFor(ctx, tx, lo.id, driverID, lo.driverID, stop, &out.settled); err != nil {
				return err
			}
			if err := unit.creditTreasury(ctx, tx, lo.id, driverID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE order_parts
				SET status = 'picked_up', picked_up_at = now(), picked_up_by = $3::uuid, updated_at = now()
				WHERE order_id = $1 AND merchant_id = $2::uuid`, lo.id, stop, driverID); err != nil {
				return err
			}
			if err := partEventTx(ctx, tx, lo.id, lo.from, driverID,
				fmt.Sprintf("استلم السائق قسم %s", p.name), lo.driverID, stop); err != nil {
				return err
			}
		}
		out.follow, err = unit.stopsDoneFollowTx(ctx, tx, lo, driverID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if out != nil && !out.noop {
		s.notifyCredits(ctx, orderID, out.settled.credits)
		s.publishWalletsOf(ctx, orderID)
	}
	return updated, nil
}

// stopsDoneFollowTx **كلُّ محطّةٍ حيّةٍ استلمها السائقُ الحاليّ ⇒ `at_pickup → picked_up`** (§2.4) —
// بعد استلام آخر محطّة، **أو بعد أن يشيل المكتبُ آخرَ محطّةٍ لم تُستلم.**
//
// **وبدور السائق** — الانتقالُ انتقالُه ولو شال المكتبُ المحطّة: البضاعةُ كلُّها في يده.
func (s *Service) stopsDoneFollowTx(ctx context.Context, tx pgx.Tx, lo *lockedOrder, actorID string) (*appliedTransition, error) {
	if lo.from != StAtPickup || lo.driverID == nil {
		return nil, nil
	}
	var done bool
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE status IN ('pending', 'accepted', 'picked_up')) > 0
		   AND count(*) FILTER (WHERE status IN ('pending', 'accepted')
		                           OR (status = 'picked_up' AND picked_up_by IS DISTINCT FROM $2::uuid)) = 0
		FROM order_parts WHERE order_id = $1`, lo.id, *lo.driverID).Scan(&done); err != nil {
		return nil, err
	}
	if !done {
		return nil, nil
	}
	return s.applyTransitionTx(ctx, tx, lo, transitionReq{
		actorID: actorID, actorRoles: []string{"driver"}, to: StPickedUp, fromParts: true,
	})
}

// DropRemainingStops **«شيل المحطّات الباقية»** (٥أ، غرفةُ الطوارئ): **كلُّ محطّةٍ لم تُستلم تُشال
// بذنب المنصّة** (`removePartTx` — أجرُ السائق الأساسيُّ وحدَه، ١ب) **فيمضي الطلبُ بما في يد سائقه**
// إلى `picked_up`، وتسري عليه مخارجُ ما بعد الاستلام كما هي.
func (s *Service) DropRemainingStops(ctx context.Context, actorID string, roles []string, orderID,
	reason string) (*Order, error) {
	reason = strings.TrimSpace(reason)
	updated, out, err := s.partAction(ctx, orderID, func(ctx context.Context, tx pgx.Tx, unit *Service,
		lo *lockedOrder, out *partOutcome) error {
		if lo.from != StAssigned && lo.from != StAtPickup {
			return ErrBadTransition
		}
		inHand, err := goodsInHandTx(ctx, tx, lo.id)
		if err != nil {
			return err
		}
		if !inHand {
			return ErrBadTransition
		}
		rows, err := tx.Query(ctx, `SELECT merchant_id::text FROM order_parts
			WHERE order_id = $1 AND status IN ('pending', 'accepted') ORDER BY seq`, lo.id)
		if err != nil {
			return err
		}
		var todo []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			todo = append(todo, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(todo) == 0 {
			out.noop = true
			return nil
		}
		for _, id := range todo {
			if err := unit.removePartTx(ctx, tx, lo, removePartReq{
				store: id, actorID: actorID, roles: roles, reason: reason, fault: FaultPlatform,
			}, out); err != nil {
				return err
			}
		}
		if out.follow == nil {
			out.follow, err = unit.stopsDoneFollowTx(ctx, tx, lo, actorID)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	s.afterPartRemovals(ctx, orderID, out)
	return updated, nil
}

// RedispatchWithGoods **«كمّل بسائق تاني» والبضاعةُ مع الأوّل** (٥أ، غرفةُ الطوارئ) — الاستثناءُ
// الوحيدُ من G5: **الطلبُ إلى الطابور والمحطّاتُ المستلَمةُ تبقى بحاملها الأوّل**، فهي للجديد
// «محطّاتُ تسليم» عند نقطة الطارئ (`pickup_override`). **ولا مال**: الأوّلُ دفع للمتجر عند
// محطّته، وعند التسليم تُردّ سلفتُه كلُّها (`collectAndReturnAdvance`).
func (s *Service) RedispatchWithGoods(ctx context.Context, actorID string, roles []string, orderID,
	note string) (*Order, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	lo, err := lockOrderTx(ctx, tx, orderID)
	if err != nil {
		return nil, err
	}
	if lo.stops <= 1 || (lo.from != StAssigned && lo.from != StAtPickup) {
		return nil, ErrBadTransition
	}
	unit := s.on(tx)
	ap, err := unit.applyTransitionTx(ctx, tx, lo, transitionReq{
		actorID: actorID, actorRoles: roles, to: StDispatching, note: note, handover: true,
	})
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.afterTransition(ctx, ap)
}

// MultiStoreGoods **أفي الطلب بضاعةٌ من متجرين فأكثر؟** — **فلا يعود إلى متجر** (قرارُ المالك ٤أ):
// البضاعةُ كلُّها للمكتب، والمتاجرُ تبقى على مستحقّها، والمنصّةُ تتحمّل. **ويسري على الطلب القديم
// من متجرين أيضاً** (PD-6/PD-7). وطلبٌ بقي فيه متجرٌ واحدٌ بعد الشيل على قاعدته اليوم.
func MultiStoreGoods(ctx context.Context, q dbtx.Querier, orderID string) (bool, error) {
	var n int
	err := q.QueryRow(ctx, `
		SELECT count(DISTINCT COALESCE(oi.merchant_id, o.merchant_id))
		FROM order_items oi JOIN orders o ON o.id = oi.order_id
		WHERE oi.order_id = $1`, orderID).Scan(&n)
	return n > 1, err
}

// ── حمولةُ السائق ────────────────────────────────────────────────────

// DriverStop **محطّةٌ في حمولة السائق** — عقدُ التطبيقات §1.
type DriverStop struct {
	MerchantID  string     `json:"merchant_id"`
	Seq         int        `json:"seq"`
	Name        string     `json:"name"`
	Phone       string     `json:"phone"`
	Address     string     `json:"address"`
	Lat         *float64   `json:"lat"`
	Lng         *float64   `json:"lng"`
	Status      string     `json:"status"`
	Ready       bool       `json:"ready"`
	ReadyAt     *time.Time `json:"ready_at"`
	PrepMinutes *int       `json:"prep_minutes"`
	ArrivedAt   *time.Time `json:"arrived_at"`
	PickedUpAt  *time.Time `json:"picked_up_at"`
	// Mine **استلمتُها أنا** · Handover **استلمها سائقٌ سابق — يأخذها منه عند نقطة الطارئ.**
	Mine     bool  `json:"mine"`
	Handover bool  `json:"handover"`
	Items    []any `json:"items"`
	// PayAmount **ما يدفعه للمتجر عند الاستلام** — صفرٌ لمتجر المحفظة ولمحطّة التسليم.
	PayAmount int64  `json:"pay_amount"`
	PaysBy    string `json:"pays_by"`
	// ItemIDs **بنودُ المحطّة** — يملأ منها البابُ أصنافَها بشكل `driverItem`.
	ItemIDs []string `json:"-"`
	// ToM **كم بين السائق والمحطّة بالمتر** — وسالبٌ: لا يُعرف.
	ToM float64 `json:"-"`
}

// DriverStops **محطّاتُ كلّ طلبٍ بعدّة محطّات كما يراها هذا السائق** — والتالية (`next`): أوّلُ
// محطّةٍ حيّةٍ لم يستلمها هو، بترتيب المحطّات. **وطلبُ المتجر الواحد غائب.**
func (s *Service) DriverStops(ctx context.Context, q dbtx.Querier, orderIDs []string, driverID string,
	staleMin int) (map[string][]DriverStop, map[string]string, error) {
	out := map[string][]DriverStop{}
	next := map[string]string{}
	if len(orderIDs) == 0 {
		return out, next, nil
	}
	rows, err := q.Query(ctx, `
		SELECT p.order_id::text, p.merchant_id::text, p.seq, m.name,
		       COALESCE(NULLIF(mo.phone::text, ''), ''), COALESCE(m.address_text, ''),
		       ST_Y(m.location::geometry), ST_X(m.location::geometry),
		       ST_Y(o.pickup_override::geometry), ST_X(o.pickup_override::geometry),
		       COALESCE(o.pickup_override_note, ''),
		       p.status, p.ready_at, p.prep_minutes, p.arrived_at, p.picked_up_at,
		       COALESCE(p.picked_up_by::text, ''),
		       COALESCE(ms.method,
		                (SELECT max(oi.merchant_settlement_method) FROM order_items oi
		                  WHERE oi.order_id = p.order_id AND oi.merchant_id = p.merchant_id),
		                NULLIF(m.settlement_method, ''), 'cash'),
		       COALESCE(ST_Distance(
		           (SELECT du.last_location FROM users du WHERE du.id = $2::uuid
		              AND du.last_location_at > now() - make_interval(mins => $3::int)),
		           CASE WHEN p.status = 'picked_up' AND p.picked_up_by IS DISTINCT FROM $2::uuid
		                THEN COALESCE(o.pickup_override, m.location) ELSE m.location END), -1)
		FROM order_parts p
		JOIN orders o ON o.id = p.order_id
		JOIN merchants m ON m.id = p.merchant_id
		LEFT JOIN users mo ON mo.id = m.owner_user_id
		LEFT JOIN merchant_settlements ms ON ms.order_id = p.order_id AND ms.merchant_id = p.merchant_id
		WHERE p.order_id = ANY($1::uuid[]) AND o.stops > 1 AND p.status <> 'pending'
		ORDER BY p.order_id, p.seq`, orderIDs, driverID, staleMin)
	if err != nil {
		return nil, nil, err
	}
	var ids []string
	seen := map[string]bool{}
	for rows.Next() {
		var oid, by, overNote string
		var d DriverStop
		var oLat, oLng *float64
		if err := rows.Scan(&oid, &d.MerchantID, &d.Seq, &d.Name, &d.Phone, &d.Address, &d.Lat, &d.Lng,
			&oLat, &oLng, &overNote, &d.Status, &d.ReadyAt, &d.PrepMinutes, &d.ArrivedAt, &d.PickedUpAt,
			&by, &d.PaysBy, &d.ToM); err != nil {
			rows.Close()
			return nil, nil, err
		}
		switch d.Status {
		case PartRejected:
			d.Status = PartRemoved
		case PartPickedUp:
			d.Mine = strings.EqualFold(by, driverID)
			d.Handover = !d.Mine
		}
		if d.Handover {
			// **نقطةُ التسليم موضعُ الطارئ** — وكلمتُه إن لم يُلتقط موضع.
			if oLat != nil && oLng != nil {
				d.Lat, d.Lng = oLat, oLng
			}
			if overNote != "" {
				d.Address = overNote
			}
		}
		d.Ready = d.ReadyAt != nil
		d.Items = []any{}
		d.ItemIDs = []string{}
		out[oid] = append(out[oid], d)
		if !seen[oid] {
			seen[oid] = true
			ids = append(ids, oid)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(ids) == 0 {
		return out, next, nil
	}
	irows, err := q.Query(ctx, `
		SELECT oi.order_id::text, COALESCE(oi.merchant_id, o.merchant_id)::text, oi.id::text
		FROM order_items oi JOIN orders o ON o.id = oi.order_id
		WHERE oi.order_id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, nil, err
	}
	lines := map[string][]string{}
	for irows.Next() {
		var oid, mid, id string
		if err := irows.Scan(&oid, &mid, &id); err != nil {
			irows.Close()
			return nil, nil, err
		}
		k := oid + "|" + strings.ToLower(mid)
		lines[k] = append(lines[k], id)
	}
	irows.Close()
	if err := irows.Err(); err != nil {
		return nil, nil, err
	}
	pays, err := s.on(q).PayMerchants(ctx, q, ids)
	if err != nil {
		return nil, nil, err
	}
	for _, oid := range ids {
		list := out[oid]
		for i := range list {
			d := &list[i]
			if d.Status != PartRemoved {
				d.ItemIDs = append(d.ItemIDs, lines[oid+"|"+strings.ToLower(d.MerchantID)]...)
			}
			if d.PaysBy == "cash" && !d.Handover && d.Status != PartRemoved {
				for _, sp := range pays[oid] {
					if strings.EqualFold(sp.MerchantID, d.MerchantID) {
						d.PayAmount = sp.Amount
					}
				}
			}
			if _, ok := next[oid]; !ok && (d.Status == PartAccepted || d.Handover) {
				next[oid] = d.MerchantID
			}
		}
	}
	return out, next, nil
}

// StopPoint **إلى أين يمشي السائقُ قبل الاستلام** — المحطّةُ المسمّاة (`stop`) أو التالية.
// **و`ok` كاذبٌ لطلب المتجر الواحد** فيبقى الطريقُ كما كان.
func (s *Service) StopPoint(ctx context.Context, orderID, driverID, stop string) (lat, lng *float64, ok bool, err error) {
	stops, next, err := s.DriverStops(ctx, s.db, []string{orderID}, driverID, 0)
	if err != nil {
		return nil, nil, false, err
	}
	list := stops[orderID]
	if len(list) == 0 {
		return nil, nil, false, nil
	}
	want := strings.ToLower(strings.TrimSpace(stop))
	if want == "" {
		want = strings.ToLower(next[orderID])
	}
	for _, d := range list {
		if strings.EqualFold(d.MerchantID, want) {
			return d.Lat, d.Lng, true, nil
		}
	}
	if stop != "" {
		return nil, nil, true, httpx.ErrNotFound
	}
	return nil, nil, true, nil
}

// editCollectedStoresTx **متاجرُ الطلب بعدّة محطّات التي استُلمت** — صفُّ تسويتها قائمٌ أو قسمُها
// «picked_up». **و`perStore` صحيحٌ ما دام الطلبُ كلُّه لم يُستلم** (`stops > 1`)؛ وبعده كما كان.
func editCollectedStoresTx(ctx context.Context, q dbtx.Querier, orderID, status string) (bool, map[string]bool, error) {
	out := map[string]bool{}
	var stops int
	if err := q.QueryRow(ctx, `SELECT stops FROM orders WHERE id = $1`, orderID).Scan(&stops); err != nil {
		return false, out, err
	}
	if stops <= 1 || pastPickup[status] {
		return false, out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT merchant_id::text FROM merchant_settlements WHERE order_id = $1
		UNION SELECT merchant_id::text FROM order_parts WHERE order_id = $1 AND status = 'picked_up'`, orderID)
	if err != nil {
		return false, out, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return false, out, err
		}
		out[strings.ToLower(id)] = true
	}
	return true, out, rows.Err()
}

// markEmptyPartsTx **قسمٌ خرجت بنودُه كلُّها بتعديل الطلب يصير «انشال»** — بذنب المنصّة، بحدثه.
// **قبل قبول الطلب «رُفض».** ولقطةُ ما خرج في سجلّ التعديل نفسِه.
func markEmptyPartsTx(ctx context.Context, q dbtx.Querier, orderID, actorID, reason, status string,
	driverID *string) error {
	rows, err := q.Query(ctx, `
		UPDATE order_parts p
		   SET status = CASE WHEN p.status = 'pending' THEN 'rejected' ELSE 'removed' END,
		       reason = $2, removed_fault = 'platform', updated_at = now()
		  FROM merchants m
		 WHERE m.id = p.merchant_id AND p.order_id = $1 AND p.status IN ('pending', 'accepted')
		   AND NOT EXISTS (SELECT 1 FROM order_items oi WHERE oi.order_id = p.order_id
		                    AND oi.merchant_id = p.merchant_id)
		RETURNING p.merchant_id::text, m.name`, orderID, strings.TrimSpace(reason))
	if err != nil {
		return err
	}
	var refs []partRef
	for rows.Next() {
		var r partRef
		if err := rows.Scan(&r.merchantID, &r.name); err != nil {
			rows.Close()
			return err
		}
		refs = append(refs, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, r := range refs {
		if err := partEventTx(ctx, q, orderID, status, actorID,
			fmt.Sprintf("انشال قسم %s بتعديل الطلب — %s", r.name, reason), driverID, r.merchantID); err != nil {
			return err
		}
	}
	return nil
}

// compensationStoreTx **على أيّ متجرٍ تُكتب مطالبةُ تعويض السائق بذنب متجر** (٦أ): طلبُ المتجر
// الواحد ⇒ متجرُه (فارغ)؛ **وطلبٌ بعدّة محطّات ⇒ ما اختاره المكتبُ من متاجره** — وبلا اختيارٍ لا
// موافقة (`compensation_store_required`). **ويُكتب الاختيارُ في طلب التعويض** فيُقرأ بعدها.
func compensationStoreTx(ctx context.Context, q dbtx.Querier, c *CompensationRequest) (string, error) {
	var stops int
	if err := q.QueryRow(ctx, `SELECT stops FROM orders WHERE id = $1`, c.OrderID).Scan(&stops); err != nil {
		return "", err
	}
	if stops <= 1 {
		return "", nil
	}
	if c.MerchantID == nil || strings.TrimSpace(*c.MerchantID) == "" {
		return "", ErrCompensationStoreRequired
	}
	store := strings.ToLower(strings.TrimSpace(*c.MerchantID))
	if _, err := normStop(store); err != nil {
		return "", ErrCompensationStoreRequired
	}
	var inOrder bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM order_parts
		WHERE order_id = $1 AND merchant_id = $2::uuid)`, c.OrderID, store).Scan(&inOrder); err != nil {
		return "", err
	}
	if !inOrder {
		return "", ErrCompensationStoreRequired
	}
	if _, err := q.Exec(ctx, `UPDATE driver_compensation_requests SET merchant_id = $2::uuid WHERE id = $1`,
		c.ID, store); err != nil {
		return "", err
	}
	return store, nil
}
