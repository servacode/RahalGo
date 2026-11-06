package orders

// ══════════════════════════════════════════════════════════════════════
// **متجرٌ رفض قسمَه أو شاله المكتب — والطلبُ يكمل بغيره** (§6 · قرارُ المالك ٢٠٢٦-١١-٠٦)
// ══════════════════════════════════════════════════════════════════════
//
// **بنودُ المتجر تخرج من الطلب وتُحفظ لقطتُها في قسمه** (`removed_items`) — فيبقى في
// سجلّه ما رفضه، **ويُعاد حسابُ الطلب بالدالّة نفسِها التي يحسب بها «صنفٌ غيرُ متوفّر»**
// (`recomputeAfterRemovalTx`): رسمُ المحطّة الزائدة يسقط، **وأجرُ السائق الأساسيُّ وحدَه
// ولو وصل لعنده** (١ب)، والمحفظةُ يُردّ لها الفرق.
//
//	قبل قبول الطلب  ⇒ القسمُ «rejected»   ·   بعده ⇒ «removed»
//	آخرُ قسمٍ حيّ    ⇒ لا تُحذف بنودُه: الطلبُ كلُّه يُرفض (قبل القبول) أو يُلغى (بعده)
//
// # من يشيل
//
//	المتجرُ  قسمَه ما دام الطلبُ «بانتظار/مقبول/قيد التحضير» — وبعدها المكتب (`ErrUsePart`)
//	المكتبُ  أيَّ قسمٍ لم يُستلم — **وآخرُ قسمٍ بعد التحويل إلغاءٌ للطلب: للأدمن وحدَه** (الآلة)
//
// **وقسمٌ استُلم لا يُمسّ** (`ErrPartCollected`): مستحقُّ متجره قُيّد.
//
// # والذنب (٦أ)
//
// **رفضُ المتجر قسمَه ذنبُه** — يُكتب عليه إنذارٌ ويُعدّ مخالفةً كرفض طلبٍ عاديّ.
// **وشيلُ المكتب يختار ذنبَه**: على المتجر (إنذارٌ على هذا المتجر وحدَه) أو على المنصّة.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/itemmatch"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

var (
	// ErrPartCollected **السائقُ استلم قسمَ هذا المتجر** — لا يُشال ولا يُحوَّل ولا تتبدّل بنودُه.
	// يترجمه البابُ (`part_collected`).
	ErrPartCollected = errors.New("قسمُ المتجر مستلَم")
	// ErrTransferTargetInOrder **المتجرُ الجديدُ في الطلب أصلاً** (`transfer_target_in_order`).
	ErrTransferTargetInOrder = errors.New("المتجرُ في الطلب أصلاً")
	// ErrPartFault **ذنبُ الشيل لا يُعرف** — المتجرُ أو المنصّة وحدَهما.
	ErrPartFault = errors.New("ذنبُ الشيل مجهول")
)

// PartTransferUnmatched **أصنافُ القسم بلا مقابلٍ عند المتجر الجديد** — تُسمّى.
type PartTransferUnmatched struct{ Items []string }

func (e *PartTransferUnmatched) Error() string {
	return "أصنافٌ بلا مقابل: " + strings.Join(e.Items, "، ")
}

// partRemoveFrom **حالاتُ الطلب التي يُشال فيها قسم** — كلُّها قبل الاستلام.
var partRemoveFrom = itemUnavailableFrom

// merchantRemoveFrom **ما يشيل فيه المتجرُ قسمَه بنفسه** — نافذةُ إلغائه للطلب العاديّ.
var merchantRemoveFrom = map[string]bool{StPending: true, StAccepted: true, StPreparing: true}

// partRemoval **قسمٌ خرج** — يقرؤه ما بعد التثبيت.
type partRemoval struct {
	merchantID, name, owner string
	status, reason, fault   string
	items                   []string
	// continued **والطلبُ يكمل بغيره** — بنودُه خرجت وأُعيد الحساب.
	continued bool
	// byStore **المتجرُ شاله بنفسه** — والمكتبُ يُخبَر.
	byStore bool
	// transfer **حُوّل لمتجرٍ آخر** — لا خبرَ «ما توفّر» للزبون.
	transfer bool
	money    removalMoney
}

// removePartReq **ما طُلب من الشيل.**
type removePartReq struct {
	store, actorID string
	roles          []string
	reason, fault  string
	// transfer **الشيلُ جزءُ تحويل القسم** — بنودُه لا تُحذف بل تنتقل (`TransferPart`).
	transfer bool
}

// partCollectedTx **أاستُلم قسمُ هذا المتجر؟** — صفُّ تسويته قائمٌ أو قسمُه «picked_up».
func partCollectedTx(ctx context.Context, q dbtx.Querier, orderID, store string) (bool, error) {
	var got bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM merchant_settlements WHERE order_id = $1 AND merchant_id = $2::uuid)
		    OR EXISTS (SELECT 1 FROM order_parts WHERE order_id = $1 AND merchant_id = $2::uuid
		                AND status = 'picked_up')`, orderID, store).Scan(&got)
	return got, err
}

// removePartTx **يشيل قسمَ متجرٍ من الطلب** — داخل فعل القسم (`partAction`)، **والانتقالُ
// التابعُ في المعاملة نفسِها** (§2.4).
//
// **ويُنادى من ثلاثة أبواب**: رفضُ المتجر قسمَه (`AnswerParts`)، وشيلُ المكتب (`RemovePart`)،
// وتحويلُ القسم (`TransferPart`). **وفي الدفعة التالية** خيارُ الطوارئ «شيل المحطّات الباقية».
func (s *Service) removePartTx(ctx context.Context, tx pgx.Tx, lo *lockedOrder, req removePartReq,
	out *partOutcome) error {
	req.store = strings.ToLower(strings.TrimSpace(req.store))
	if terminal(lo.from) {
		return ErrBadTransition
	}
	if !partRemoveFrom[lo.from] {
		return ErrPartCollected
	}
	byStore := slices.Contains(req.roles, "merchant") && !slices.Contains(req.roles, "ops") &&
		!slices.Contains(req.roles, "admin")
	fault := req.fault
	switch {
	case byStore:
		// **المتجرُ يشيل قسمَه في نافذة إلغائه وحدَها** — وبعدها المكتب.
		if !merchantRemoveFrom[lo.from] {
			return ErrUsePart
		}
		to := StCancelled
		if lo.from == StPending {
			to = StRejected
		}
		eff := rolesUnderMode(s.MerchantsSelfManage(ctx), lo.from, to, req.roles, lo.driverID != nil)
		if !canTransition(lo.kind, lo.from, to, eff) {
			return ErrBadTransition
		}
		fault = FaultMerchant
	case fault != FaultMerchant && fault != FaultPlatform:
		return ErrPartFault
	}
	owner := ""
	if byStore {
		owner = req.actorID
	}
	got, err := lockPartsTx(ctx, tx, lo.id, []string{req.store}, owner)
	if err != nil {
		return err
	}
	switch got[req.store] {
	case PartPending, PartAccepted:
	case PartRejected, PartRemoved:
		// **شيلٌ أُعيد** — لا يُكتب شيء.
		out.noop = true
		return nil
	default:
		return ErrPartCollected
	}
	if collected, err := partCollectedTx(ctx, tx, lo.id, req.store); err != nil {
		return err
	} else if collected {
		return ErrPartCollected
	}

	// ── ١ · لقطةُ ما يخرج — قبل أن يُحذف ─────────────────────────────────
	var primary string
	if err := tx.QueryRow(ctx, `SELECT COALESCE(merchant_id::text, '') FROM orders WHERE id = $1`,
		lo.id).Scan(&primary); err != nil {
		return err
	}
	var snap []byte
	var removedTotal int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(json_agg(json_build_object(
		           'id', oi.id, 'menu_item_id', oi.menu_item_id, 'name', oi.name, 'qty', oi.qty,
		           'unit_price', oi.unit_price, 'merchant_price', oi.merchant_price,
		           'note', oi.note, 'options', COALESCE(oi.options, '[]'::jsonb), 'drink_for', oi.drink_for)
		         ORDER BY oi.drink_for NULLS FIRST, oi.name, oi.id), '[]'::json)::jsonb,
		       COALESCE(sum(oi.unit_price * oi.qty), 0)
		FROM order_items oi
		WHERE oi.order_id = $1 AND COALESCE(oi.merchant_id::text, $3) = lower($2)`,
		lo.id, req.store, primary).Scan(&snap, &removedTotal); err != nil {
		return err
	}
	var items []struct {
		Name     string  `json:"name"`
		DrinkFor *string `json:"drink_for"`
	}
	_ = json.Unmarshal(snap, &items)
	r := partRemoval{
		merchantID: req.store, reason: req.reason, fault: fault,
		byStore: byStore, transfer: req.transfer,
	}
	for _, it := range items {
		if it.DrinkFor == nil {
			r.items = append(r.items, it.Name)
		}
	}
	r.status = PartRemoved
	if lo.from == StPending {
		r.status = PartRejected
	}

	// ── ٢ · أيبقى في الطلب قسمٌ حيٌّ غيرُه؟ ────────────────────────────────
	var others int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM order_parts
		WHERE order_id = $1 AND merchant_id <> $2::uuid AND status IN ('pending', 'accepted', 'picked_up')`,
		lo.id, req.store).Scan(&others); err != nil {
		return err
	}
	r.continued = others > 0 || req.transfer
	if others > 0 && !req.transfer {
		// **بنودُه تخرج والطلبُ يُعاد حسابُه** — ومشروباتُ الصنف معه (`ON DELETE CASCADE`).
		if _, err := tx.Exec(ctx, `
			DELETE FROM order_items WHERE order_id = $1 AND COALESCE(merchant_id::text, $3) = lower($2)`,
			lo.id, req.store, primary); err != nil {
			return err
		}
		if r.money, err = s.recomputeAfterRemovalTx(ctx, tx, lo.id, req.store, req.actorID,
			"قسم من الطلب ما توفّر"); err != nil {
			return err
		}
		// **والآلةُ تقرأ مالَ الطلب الجديد** إن تبعه انتقال.
		if fresh, err := lockOrderTx(ctx, tx, lo.id); err == nil {
			fresh.from = lo.from
			*lo = *fresh
		} else {
			return err
		}
	}

	// ── ٣ · القسمُ وحدثُه ───────────────────────────────────────────────
	role := partRole(req.actorID, authorRole(req.roles))
	if err := tx.QueryRow(ctx, `
		UPDATE order_parts p
		   SET status = $3, reason = $4, removed_fault = $5,
		       removed_items = $6::jsonb, removed_total = $7,
		       answered_at = CASE WHEN p.status = 'pending' THEN now() ELSE p.answered_at END,
		       answered_by = CASE WHEN p.status = 'pending' THEN $8::uuid ELSE p.answered_by END,
		       answered_role = CASE WHEN p.status = 'pending' THEN $9 ELSE p.answered_role END,
		       updated_at = now()
		  FROM merchants m
		 WHERE m.id = p.merchant_id AND p.order_id = $1 AND p.merchant_id = $2::uuid
		RETURNING m.name, COALESCE(m.owner_user_id::text, '')`,
		lo.id, req.store, r.status, req.reason, fault, snap, removedTotal, actorArg(req.actorID), role).
		Scan(&r.name, &r.owner); err != nil {
		return err
	}
	note := fmt.Sprintf("انشال قسم %s — %s", r.name, req.reason)
	switch {
	case byStore && r.status == PartRejected:
		note = fmt.Sprintf("رفض %s قسمه — %s", r.name, req.reason)
	case byStore:
		note = fmt.Sprintf("شال %s قسمه — %s", r.name, req.reason)
	case r.status == PartRejected:
		note = fmt.Sprintf("رُفض قسم %s — %s", r.name, req.reason)
	}
	if err := partEventTx(ctx, tx, lo.id, lo.from, req.actorID, note, lo.driverID, req.store); err != nil {
		return err
	}
	out.removed = append(out.removed, r)

	// ── ٤ · الانتقالُ التابع (§2.4) ────────────────────────────────────
	if r.continued {
		if lo.from == StPending && out.follow == nil {
			out.follow, err = s.partsFollowUpTx(ctx, tx, lo)
		}
		// **والطلبُ عند المحطّة وكلُّ ما بقي استلمه سائقُه** ⇒ `picked_up` في المعاملة نفسِها
		// (§2.4) — شال المكتبُ آخرَ محطّةٍ لم تُستلم، **فالبضاعةُ كلُّها في يده.**
		if err == nil && out.follow == nil && !req.transfer {
			out.follow, err = s.stopsDoneFollowTx(ctx, tx, lo, req.actorID)
		}
		return err
	}
	if lo.from == StPending {
		// **رُفض آخرُ قسم** — الطلبُ كلُّه مرفوض («ما توفّر شي من الطلب»).
		out.follow, err = s.partsFollowUpTx(ctx, tx, lo)
		return err
	}
	// **وآخرُ قسمٍ بعد القبول إلغاءٌ للطلب كلِّه — بأدوار الفاعل**، **وبعد التحويل للأدمن وحدَه**
	// (D10) في الوضعين — **ورفضُه يُرجع المعاملةَ كلَّها** فلا يُكتب القسم.
	if afterHandoff[lo.from] && !slices.Contains(req.roles, "admin") {
		return ErrBadTransition
	}
	out.follow, err = s.applyTransitionTx(ctx, tx, lo, transitionReq{
		actorID: req.actorID, actorRoles: req.roles, to: StCancelled, note: req.reason, fromParts: true,
	})
	return err
}

// authorRole **الدورُ الذي يُكتب على القسم** — الأدمنُ قبل المكتب، والمتجرُ بلاهما.
func authorRole(roles []string) string {
	switch {
	case slices.Contains(roles, "admin"):
		return "admin"
	case slices.Contains(roles, "ops"):
		return "ops"
	case slices.Contains(roles, "merchant"):
		return "merchant"
	}
	return "ops"
}

// RemovePart **يشيل قسمَ متجرٍ من طلبٍ بعدّة محطّات** — `fault` «merchant» أو «platform».
//
// **والمتجرُ ينادي بدوره** (`roles = ["merchant"]`) فيكون الذنبُ ذنبَه. **والردُّ الطلبُ
// بعد الشيل** — والانتقالُ التابعُ إن وقع (رفضٌ أو إلغاءٌ للطلب كلِّه) قد مرّ بآلته.
func (s *Service) RemovePart(ctx context.Context, actorID string, roles []string, orderID, storeID,
	reason, fault string, hook func(context.Context, dbtx.Querier) error) (*Order, error) {
	var h func(context.Context, dbtx.Querier, partRemoval) error
	if hook != nil {
		h = func(ctx context.Context, q dbtx.Querier, _ partRemoval) error { return hook(ctx, q) }
	}
	return s.removePart(ctx, actorID, roles, orderID, storeID, reason, fault, h)
}

// removePart **الشيلُ بأثره في المعاملة نفسِها** — `hook` يقرأ ما خرج (التدقيق).
func (s *Service) removePart(ctx context.Context, actorID string, roles []string, orderID, storeID,
	reason, fault string, hook func(context.Context, dbtx.Querier, partRemoval) error) (*Order, error) {
	reason = strings.TrimSpace(reason)
	updated, out, err := s.partAction(ctx, orderID, func(ctx context.Context, tx pgx.Tx, unit *Service,
		lo *lockedOrder, out *partOutcome) error {
		if err := unit.removePartTx(ctx, tx, lo, removePartReq{
			store: storeID, actorID: actorID, roles: roles, reason: reason, fault: fault,
		}, out); err != nil {
			return err
		}
		if hook != nil && len(out.removed) > 0 {
			return hook(ctx, tx, out.removed[len(out.removed)-1])
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.afterPartRemovals(ctx, orderID, out)
	return updated, nil
}

// afterPartRemovals **ما بعد التثبيت** — الإخطارُ والإنذارُ والإنزال.
func (s *Service) afterPartRemovals(ctx context.Context, orderID string, out *partOutcome) {
	if out == nil || out.noop {
		return
	}
	s.notifyPartRemovals(ctx, orderID, out)
	for _, r := range out.removed {
		if r.fault == FaultMerchant {
			s.warnMerchantPart(ctx, orderID, r.merchantID, r.byStore)
		}
	}
	if len(out.removed) > 0 {
		s.publishWalletsOf(ctx, orderID)
	}
	s.maybeDispatchParts(ctx, orderID)
}

// notifyPartRemovals **من يُخبَر بقسمٍ خرج:**
//
//   - **الزبونُ** حين يكمل الطلبُ بغيره — «قسم من طلبك ما توفّر» بأصنافه ومجموعه الجديد،
//     **بلا اسم متجر** (`P-1`). **وآخرُ قسمٍ يقوله الطلبُ نفسُه** («اعتذر المتجر عن طلبك»).
//   - **السائقُ** إن كان الطلبُ معه — «انشالت محطة من الطلب»: لا يروح لعندها.
//   - **المتجرُ** حين شاله المكتب · **والمكتبُ** حين شاله المتجرُ بنفسه (C13).
func (s *Service) notifyPartRemovals(ctx context.Context, orderID string, out *partOutcome) {
	if s.notify == nil {
		return
	}
	var number int64
	var customerID string
	var driverID *string
	if err := s.db.QueryRow(ctx,
		`SELECT number, COALESCE(customer_id::text, ''), driver_id::text FROM orders WHERE id = $1`, orderID).
		Scan(&number, &customerID, &driverID); err != nil {
		return
	}
	ref := fmt.Sprintf("#%d", number)
	for _, r := range out.removed {
		if r.continued && !r.transfer && customerID != "" {
			body := fmt.Sprintf("%s — وباقي طلبك مكمّل · صار طلبك %s ل.س", ref, groupDigits(r.money.newTotal))
			if len(r.items) > 0 {
				body = fmt.Sprintf("%s — ما توفّر: %s · وباقي طلبك مكمّل · صار طلبك %s ل.س",
					ref, strings.Join(r.items, "، "), groupDigits(r.money.newTotal))
			}
			s.notify.Notify(ctx, notifications.Input{
				UserID: customerID, Kind: notifications.KindOrder,
				Title: t.partUnavailable, Body: body,
				Entity: "order", EntityID: orderID, Href: "/portal/orders",
				Apps: []string{notifications.AppCustomer},
			})
		}
		if r.continued && driverID != nil && *driverID != "" {
			s.notify.Notify(ctx, notifications.Input{
				UserID: *driverID, Kind: notifications.KindOrder,
				Title: t.partRemovedDriver, Body: fmt.Sprintf("%s — %s — لا تروح لعندها", ref, r.name),
				Entity: "order", EntityID: orderID, Href: "/portal",
				Apps: []string{notifications.AppDriver},
			})
		}
		switch {
		case r.byStore:
			body := fmt.Sprintf("%s — %s رفض قسمه", ref, r.name)
			if r.status == PartRemoved {
				body = fmt.Sprintf("%s — %s شال قسمه", ref, r.name)
			}
			if r.reason != "" {
				body += " · " + r.reason
			}
			s.notify.NotifyRoles(ctx, notifications.OpsDesk, notifications.Input{
				Kind: notifications.KindOrder, Title: t.partRejectedOps, Body: body,
				Entity: "order", EntityID: orderID, Href: "/dashboard/orders",
			})
		case !r.byStore && r.owner != "":
			body := ref
			if r.reason != "" {
				body += " — " + r.reason
			}
			s.notify.Notify(ctx, notifications.Input{
				UserID: r.owner, Kind: notifications.KindOrder,
				Title: t.partRemovedMerchant, Body: body,
				Entity: "order", EntityID: orderID, Href: "/portal",
				Apps: []string{notifications.AppMerchant},
			})
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **تحويلُ قسمٍ لمتجرٍ آخر** (§7.3) — والأقسامُ الأخرى لا تُمسّ
// ══════════════════════════════════════════════════════════════════════
//
// **القسمُ القديمُ «removed»** («حُوّل لمتجر آخر»، ويُخبَر متجرُه، ولا يُمحى شيء) **وقسمٌ
// جديدٌ للمتجر الجديد** مقبولٌ بيد المكتب ولم يُبلَّغ، ترتيبُه آخرُ المحطّات. **وبنودُه تنتقل
// إليه**: مقابلُها عنده بالمطابقة نفسِها التي تقترحها شاشةُ التحويل (`itemmatch`)، **وسعرُ
// الزبون كما رآه**، **وطريقةُ التسوية من المتجر الجديد** (PD-3). **وحالُ الطلب لا تُرجَع**
// (C16)، **ومهلةُ الإسناد الصامتة تُمحى** (D13).

// TransferPartResult **ما انتقل.**
type TransferPartResult struct {
	Items int `json:"items"`
}

// TransferPart **ينقل قسمَ متجرٍ في طلبٍ بعدّة محطّات إلى متجرٍ آخر** — بيد المكتب.
func (s *Service) TransferPart(ctx context.Context, actorID string, roles []string, orderID, fromStore,
	toStore, reason string, hook func(context.Context, dbtx.Querier) error) (*Order, TransferPartResult, error) {
	var res TransferPartResult
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "حُوّل لمتجر آخر"
	}
	updated, out, err := s.partAction(ctx, orderID, func(ctx context.Context, tx pgx.Tx, unit *Service,
		lo *lockedOrder, out *partOutcome) error {
		if terminal(lo.from) || lo.from == StPending {
			// **قبل القبول الأفعالُ ثلاثة**: قبولٌ أو رفضٌ أو إلغاء — كتحويل الطلب كلِّه.
			return ErrBadTransition
		}
		if lo.kind != KindStandard {
			return ErrBadTransition
		}
		var inOrder bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM order_parts
			WHERE order_id = $1 AND merchant_id = $2::uuid)`, lo.id, toStore).Scan(&inOrder); err != nil {
			return err
		}
		if inOrder {
			return ErrTransferTargetInOrder
		}
		var method string
		if err := tx.QueryRow(ctx,
			`SELECT settlement_method FROM merchants WHERE id = $1::uuid AND status = 'active' FOR SHARE`,
			toStore).Scan(&method); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrBadTransition
			}
			return err
		}
		if method == "" {
			method = "cash"
		}
		// **القسمُ القديمُ يُشال أوّلاً** — فحوصُه (مستلَم؟ حيّ؟) هي فحوصُ الشيل.
		if err := unit.removePartTx(ctx, tx, lo, removePartReq{
			store: fromStore, actorID: actorID, roles: roles, reason: reason, fault: FaultPlatform,
			transfer: true,
		}, out); err != nil {
			return err
		}
		if out.noop {
			return ErrBadTransition
		}
		// ── المقابلُ عند المتجر الجديد ─────────────────────────────────
		lines, err := partLinesTx(ctx, tx, lo.id, fromStore)
		if err != nil {
			return err
		}
		menu, err := menuCandidatesTx(ctx, tx, toStore)
		if err != nil {
			return err
		}
		var missing []string
		for i := range lines {
			if lines[i].drinkFor {
				continue
			}
			if j, _ := itemmatch.Best(lines[i].name, lines[i].merchantPrice, menu); j >= 0 {
				lines[i].menuID, lines[i].price = menu[j].ID, menu[j].Price
			} else {
				missing = append(missing, lines[i].name)
			}
		}
		if len(missing) > 0 {
			return &PartTransferUnmatched{Items: missing}
		}
		for _, l := range lines {
			if l.drinkFor {
				// **والمشروباتُ تتبع صنفَها** — سعرُها كما رآه الزبون.
				_, err = tx.Exec(ctx, `UPDATE order_items SET merchant_id = $2::uuid,
					merchant_settlement_method = $3 WHERE id = $1::uuid`, l.id, toStore, method)
			} else {
				_, err = tx.Exec(ctx, `UPDATE order_items SET menu_item_id = $2::uuid, merchant_id = $3::uuid,
					merchant_price = $4, merchant_settlement_method = $5 WHERE id = $1::uuid`,
					l.id, l.menuID, toStore, l.price, method)
			}
			if err != nil {
				return err
			}
			res.Items++
		}
		// ── القسمُ الجديد: مقبولٌ بيد المكتب، لم يُبلَّغ، آخرُ المحطّات ──
		role := partRole(actorID, authorRole(roles))
		var name string
		if err := tx.QueryRow(ctx, `
			INSERT INTO order_parts (order_id, merchant_id, seq, status, answered_at, answered_by,
			                         answered_role, prep_minutes)
			SELECT $1, m.id, (SELECT COALESCE(max(seq), 0) + 1 FROM order_parts WHERE order_id = $1),
			       'accepted', now(), $3::uuid, $4, LEAST(GREATEST(COALESCE(m.default_prep_minutes, 0), 0), 600)
			FROM merchants m WHERE m.id = $2::uuid
			RETURNING (SELECT name FROM merchants WHERE id = $2::uuid)`,
			lo.id, toStore, actorArg(actorID), role).Scan(&name); err != nil {
			return err
		}
		// **ومتجرُ الطلب إن كان القديمَ صار الجديد** — ومهلةُ الإسناد الصامتة تُمحى.
		if _, err := tx.Exec(ctx, `
			UPDATE orders SET merchant_id = CASE WHEN merchant_id = $2::uuid THEN $3::uuid ELSE merchant_id END,
			       offer_expires_at = NULL, updated_at = now()
			WHERE id = $1`, lo.id, fromStore, toStore); err != nil {
			return err
		}
		if err := partEventTx(ctx, tx, lo.id, lo.from, actorID,
			fmt.Sprintf("%s — إلى %s", StoreTransferNote, name), lo.driverID, toStore); err != nil {
			return err
		}
		out.transferTo = partRef{merchantID: toStore, name: name}
		if hook != nil {
			return hook(ctx, tx)
		}
		return nil
	})
	if err != nil {
		return nil, res, err
	}
	s.afterPartRemovals(ctx, orderID, out)
	if out.transferTo.merchantID != "" {
		s.notifyPartTransferred(ctx, orderID, out.transferTo)
	}
	return updated, res, nil
}

// notifyPartTransferred **السائقُ يُخبَر بالمتجر الجديد** — والمتجرُ الجديدُ يُبلَّغ بتذكرته.
func (s *Service) notifyPartTransferred(ctx context.Context, orderID string, to partRef) {
	if s.notify == nil {
		return
	}
	var number int64
	var driverID *string
	if err := s.db.QueryRow(ctx, `SELECT number, driver_id::text FROM orders WHERE id = $1`, orderID).
		Scan(&number, &driverID); err != nil || driverID == nil || *driverID == "" {
		return
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: *driverID, Kind: notifications.KindOrder,
		Title: t.partTransferredDriver, Body: fmt.Sprintf("#%d — محطة جديدة: %s", number, to.name),
		Entity: "order", EntityID: orderID, Href: "/portal",
		Apps: []string{notifications.AppDriver},
	})
	s.pub.Publish("driver:"+*driverID, map[string]any{"type": "order"})
}

// partLine **بندُ قسمٍ ينتقل.**
type partLine struct {
	id, name      string
	merchantPrice int64
	drinkFor      bool
	menuID        string
	price         int64
}

func partLinesTx(ctx context.Context, q dbtx.Querier, orderID, store string) ([]partLine, error) {
	rows, err := q.Query(ctx, `
		SELECT oi.id::text, oi.name, oi.merchant_price, oi.drink_for IS NOT NULL
		FROM order_items oi
		WHERE oi.order_id = $1 AND oi.merchant_id = $2::uuid
		ORDER BY oi.name, oi.id FOR UPDATE`, orderID, store)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []partLine
	for rows.Next() {
		var l partLine
		if err := rows.Scan(&l.id, &l.name, &l.merchantPrice, &l.drinkFor); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// menuCandidatesTx **أصنافُ المتجر الجديد المتاحةُ المُقرّة** — كما تقرؤها شاشةُ التحويل.
func menuCandidatesTx(ctx context.Context, q dbtx.Querier, store string) ([]itemmatch.Candidate, error) {
	rows, err := q.Query(ctx, `
		SELECT id::text, name, merchant_price FROM menu_items
		WHERE merchant_id = $1::uuid AND available AND approved
		ORDER BY name, id`, store)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []itemmatch.Candidate
	for rows.Next() {
		var c itemmatch.Candidate
		if err := rows.Scan(&c.ID, &c.Name, &c.Price); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
