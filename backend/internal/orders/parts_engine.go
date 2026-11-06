package orders

// ══════════════════════════════════════════════════════════════════════
// **فعلُ القسم — والآلةُ واحدة** (طلبٌ بعدّة محطّات، قرارُ المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// **الطلبُ يبقى «بانتظار» حتّى يجيب كلُّ متجرٍ عن قسمه.** **وآخرُ جوابٍ ينقل
// الطلبَ كلَّه في المعاملة نفسِها** — `pending → accepted` إن قُبل قسمٌ على الأقلّ،
// أو `pending → rejected` إن رُفضت كلُّها — **بآلة الحال الواحدة** (`applyTransitionTx`).
// **فكلُّ مُنزِلٍ إلى السائقين ينتظر المتاجرَ كلَّها بلا تبديل**: لا إنزالَ إلّا من
// `accepted/preparing`.
//
// # فعلُ القسم في معاملةٍ واحدة — `XG-46` · `D6`
//
//	Begin → lockOrderTx → فحصٌ تحت القفل → كتابةُ القسم وحدثِه → الانتقالُ التابع → Commit
//
// **ولا `s.Transition` داخلَها**: وصلةٌ ثانيةٌ على صفٍّ مقفولٍ جمود. **ولا انتقالَ
// تابعٌ في معاملةٍ ثانية**: كان كانسُ القبول يقبل الطلبَ بين رفض الأخير وانتقاله
// فيصير «مقبولاً» بلا قسمٍ مقبول.
//
// # والبابُ القديمُ يمرّ من هنا
//
// **قبولُ المكتب أو رفضُه أو كنسُه للطلب كلِّه** يجيب عن كلّ قسمٍ معلَّقٍ في
// المعاملة نفسِها (`partGatesTx`، G1). **والمتجرُ لا يقبل الطلبَ كلَّه** — بابُه
// يُوجَّه إلى قسمه (G1/G6، `ErrUsePart`). **ولا إنزالَ وقسمٌ معلَّق** (G2).
//
// # ورفضُ قسمٍ والطلبُ ماضٍ بغيره
//
// **بنودُه تخرج ويُعاد حسابُ المبلغ** (`removePartTx`، `parts_remove.go`) — **ولا يُقبل طلبٌ
// يدفع فيه الزبونُ ثمنَ ما لن يصله.** وآخرُ قسمٍ يُرفض يرفض الطلبَ كلَّه.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

var (
	// ErrUsePart **الطلبُ من عدّة متاجر — وكلُّ متجرٍ يجيب عن قسمه** (G1/G6).
	ErrUsePart = httpx.NewError(http.StatusConflict, "use_part", "errors.use_part")
	// ErrPartsPending **لا إنزالَ وفي الطلب قسمٌ لم يُجَب** (G2).
	ErrPartsPending = httpx.NewError(http.StatusConflict, "parts_pending", "errors.parts_pending")
	// ErrPartNotReady **قسمٌ لا يُعلَن جاهزاً في حاله** — يترجمه بابُ المتجر (`not_readyable`).
	ErrPartNotReady = errors.New("القسمُ لا يُعلَن جاهزاً في حاله")
)

// حالاتُ القسم (`order_parts.status`).
const (
	PartPending  = "pending"
	PartAccepted = "accepted"
	PartRejected = "rejected"
	PartRemoved  = "removed"
	PartPickedUp = "picked_up"
)

// partsAnsweredNote **حدثُ الطلب حين يجيب آخرُ متجر** — والفاعلُ النظام.
const partsAnsweredNote = "جاوبت كل المتاجر"

// allPartsRejectedReason **سببُ رفض الطلب حين لا يُقبل قسمٌ منه** — يصل الزبونَ.
const allPartsRejectedReason = "ما توفّر شي من الطلب"

// partStoresOf **متاجرُ أقسام الطلب بترتيبها.**
func partStoresOf(ctx context.Context, q dbtx.Querier, orderID string) ([]string, error) {
	rows, err := q.Query(ctx,
		`SELECT merchant_id::text FROM order_parts WHERE order_id = $1 ORDER BY seq`, orderID)
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

// StoreOrdersSQL **طلباتُ متجرٍ: متجرُها الأوّلُ أو له فيها قسم** — على الاسم المستعار `o`.
//
// **وبمصفوفةٍ تُحسب مرّةً لا بـ`EXISTS` لكلّ صفّ** — فيبقى الفهرسان (`merchant_id`
// والمفتاح) يعملان معاً، **ولا يُمسح جدولُ الطلبات كلُّه** لسجلّ متجرٍ واحد.
func StoreOrdersSQL(p string) string {
	return `(o.merchant_id = ` + p + `::uuid OR o.id = ANY (ARRAY(
	         SELECT op.order_id FROM order_parts op WHERE op.merchant_id = ` + p + `::uuid)))`
}

// partClosedSQL **قسمُ المتجر انتهى له** — رُفض أو أُزيل والطلبُ قد يمضي بغيره.
// يُكتب على الاسم المستعار `o` و`p` معاملُ المتجر.
func partClosedSQL(p string) string {
	return `EXISTS (SELECT 1 FROM order_parts pc WHERE pc.order_id = o.id
	        AND pc.merchant_id = ` + p + `::uuid AND pc.status IN ('rejected', 'removed'))`
}

// StoreStatusSQL **حالُ الطلب كما يراه متجرٌ بعينه** — حالُ قسمه حين يفترق عن الطلب.
//
//	قسمٌ رُفض ⇒ rejected · قسمٌ أُزيل ⇒ cancelled · قبل قسمَه والطلبُ ينتظر غيرَه ⇒ accepted
//	(**وpreparing في وضع «المتاجر تدير»** — كما يعرضه الإسقاطُ `projectOrder`، فيجد الفلترُ ما يُعرض)
//
// **وطلبُ المتجر الواحد بلا قسمٍ فحالُه حالُه.** على الاسم المستعار `o`.
func StoreStatusSQL(p string, selfManage bool) string {
	acceptedAs := StAccepted
	if selfManage {
		acceptedAs = StPreparing
	}
	return `COALESCE((SELECT CASE WHEN sp.status = 'rejected' THEN 'rejected'
	                              WHEN sp.status = 'removed' THEN 'cancelled'
	                              WHEN o.status = 'pending' AND sp.status = 'accepted' THEN '` + acceptedAs + `' END
	                  FROM order_parts sp WHERE sp.order_id = o.id AND sp.merchant_id = ` + p + `::uuid),
	                 o.status)`
}

// partCounts **أقسامُ الطلب بأحوالها.**
type partCounts struct {
	pending int
	// live مقبولٌ أو مستلَم.
	live int
}

func partCountsTx(ctx context.Context, q dbtx.Querier, orderID string) (partCounts, error) {
	var c partCounts
	err := q.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE p.status = 'pending'),
		       count(*) FILTER (WHERE p.status IN ('accepted', 'picked_up'))
		FROM order_parts p WHERE p.order_id = $1`, orderID).Scan(&c.pending, &c.live)
	return c, err
}

// partRole **بأيّ صفةٍ أُجيب القسم** — والفاعلُ الفارغُ هو النظام (الكانس).
func partRole(actorID, role string) string {
	if actorID == "" {
		return "system"
	}
	switch role {
	case "merchant", "ops", "admin":
		return role
	}
	return "ops"
}

// partRef **قسمٌ كتبه فعل** — متجرُه واسمُه.
type partRef struct {
	merchantID string
	name       string
}

// actorArg **الفاعلُ الفارغُ `NULL`** — لا معرّفٌ مختلَق (انظر `PF-08` في الآلة).
func actorArg(actorID string) any {
	if actorID == "" {
		return nil
	}
	return actorID
}

// partEventTx **حدثُ قسمٍ في سجلّ الطلب** — `from = to` = حالُ الطلب، **و`merchant_id`
// يقول إنّه حدثُ قسمٍ** فيتخطّاه من يقرأ «دخولَ حال» (باب الانتظار وسجلُّ الزبون).
func partEventTx(ctx context.Context, q dbtx.Querier, orderID, status, actorID, note string,
	driverID *string, merchantID string) error {
	_, err := q.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note, driver_id, merchant_id)
		VALUES ($1, $2, $2, $3, $4, $5, $6)`, orderID, status, actorArg(actorID), note, driverID, merchantID)
	return err
}

// answerPartsTx **يكتب جوابَ أقسامٍ معلَّقة** — `stores` الفارغُ يعني «كلُّ معلَّق»
// (قبولُ المكتب أو كنسُه). **ويردّ ما كُتب** ليُكتب حدثُه ويُخطَر صاحبُه.
func answerPartsTx(ctx context.Context, q dbtx.Querier, lo *lockedOrder, stores []string,
	actorID, role string, accept bool, prep *int, reason string, fromStatuses []string) ([]partRef, error) {
	var storeArg any
	if len(stores) > 0 {
		storeArg = stores
	}
	var sql string
	args := []any{lo.id, storeArg, actorArg(actorID), role, fromStatuses}
	if accept {
		// **ومدّةُ التحضير ما قاله المتجر — وإلّا افتراضُه** (كما في طلب المتجر الواحد).
		sql = `
			UPDATE order_parts p
			   SET status = 'accepted', answered_at = now(), answered_by = $3, answered_role = $4,
			       prep_minutes = LEAST(GREATEST(COALESCE($6::int, m.default_prep_minutes, 0), 0), 600),
			       updated_at = now()
			  FROM merchants m
			 WHERE m.id = p.merchant_id AND p.order_id = $1 AND p.status = ANY($5::text[])
			   AND ($2::uuid[] IS NULL OR p.merchant_id = ANY($2::uuid[]))
			RETURNING p.merchant_id::text, m.name, p.seq`
		args = append(args, prep)
	} else {
		sql = `
			UPDATE order_parts p
			   SET status = 'rejected', answered_at = now(), answered_by = $3, answered_role = $4,
			       reason = $6, updated_at = now()
			  FROM merchants m
			 WHERE m.id = p.merchant_id AND p.order_id = $1 AND p.status = ANY($5::text[])
			   AND ($2::uuid[] IS NULL OR p.merchant_id = ANY($2::uuid[]))
			RETURNING p.merchant_id::text, m.name, p.seq`
		args = append(args, reason)
	}
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	type row struct {
		partRef
		seq int
	}
	var got []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.merchantID, &r.name, &r.seq); err != nil {
			rows.Close()
			return nil, err
		}
		got = append(got, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// **والحدثُ بترتيب المحطّات** — يُقرأ في السجلّ كما تُزار.
	for i := 1; i < len(got); i++ {
		for j := i; j > 0 && got[j].seq < got[j-1].seq; j-- {
			got[j], got[j-1] = got[j-1], got[j]
		}
	}
	out := make([]partRef, 0, len(got))
	for _, r := range got {
		note := fmt.Sprintf("قبل %s قسمه", r.name)
		if !accept {
			note = fmt.Sprintf("رفض %s قسمه — %s", r.name, reason)
		}
		if err := partEventTx(ctx, q, lo.id, lo.from, actorID, note, lo.driverID, r.merchantID); err != nil {
			return nil, err
		}
		out = append(out, r.partRef)
	}
	return out, nil
}

// partGate **ما يقرّره بابُ الأقسام لانتقالٍ على الطلب كلِّه.**
type partGate struct {
	to, note string
}

// partGatesTx **بوّاباتُ الطلب بعدّة محطّات لانتقالٍ لم يأتِ من فعل قسم** — G1 · G2 · G6.
//
//	G1  pending → accepted|rejected بيد المكتب أو الكانس ⇒ يُجاب كلُّ قسمٍ معلَّق في المعاملة نفسِها؛
//	    **والوجهةُ تُشتقّ من الأقسام تحت القفل**: لا قسمَ مقبولاً ⇒ rejected، لا «مقبولٌ بلا قسم» (D6).
//	    وبيد المتجر ⇒ مرفوض (`ErrUsePart`) — بابُه يُوجَّه إلى قسمه.
//	G2  → dispatching وقسمٌ معلَّق ⇒ مرفوض (`ErrPartsPending`) — حزامٌ فوق الآلة.
//	G6  إلغاءُ المتجر للطلب كلِّه ⇒ مرفوض (`ErrUsePart`).
func (s *Service) partGatesTx(ctx context.Context, tx pgx.Tx, lo *lockedOrder,
	actorID, to, note, endedBy string) (partGate, error) {
	g := partGate{to: to, note: note}
	switch {
	case lo.from == StPending && (to == StAccepted || to == StRejected):
		if endedBy == "merchant" {
			return g, ErrUsePart
		}
		c, err := partCountsTx(ctx, tx, lo.id)
		if err != nil {
			return g, err
		}
		role := partRole(actorID, endedBy)
		if to == StAccepted {
			switch {
			case c.pending+c.live == 0:
				// **رُفضت الأقسامُ كلُّها قبل أن يصل القبول** — فالطلبُ مرفوض.
				g.to, g.note = StRejected, allPartsRejectedReason
			default:
				if _, err := answerPartsTx(ctx, tx, lo, nil, actorID, role, true, nil, "",
					[]string{PartPending}); err != nil {
					return g, err
				}
				return g, nil
			}
		}
		reason := strings.TrimSpace(g.note)
		if reason == "" {
			reason = allPartsRejectedReason
		}
		if _, err := answerPartsTx(ctx, tx, lo, nil, actorID, role, false, nil, reason,
			[]string{PartPending, PartAccepted}); err != nil {
			return g, err
		}
	case to == StCancelled && endedBy == "merchant":
		return g, ErrUsePart
	case to == StDispatching && (lo.from == StAccepted || lo.from == StPreparing):
		c, err := partCountsTx(ctx, tx, lo.id)
		if err != nil {
			return g, err
		}
		if c.pending > 0 {
			return g, ErrPartsPending
		}
	}
	return g, nil
}

// partsPrepTx **مدّةُ تحضير الطلب = أبطأُ أقسامه** — من لحظة القبول (`accepted_at`).
//
// `ceil(max(answered_at + prep) − now)` ولا أقلّ من صفر — **فـ`accepted_at + prep_minutes`
// موعدُ أبطأ متجر**، وهو ما يحسبه السائقُ وما يقيس به المكتبُ التأخّر.
func partsPrepTx(ctx context.Context, q dbtx.Querier, orderID string) error {
	_, err := q.Exec(ctx, `
		UPDATE orders o SET prep_minutes = sub.prep
		FROM (SELECT GREATEST(0, CEIL(EXTRACT(EPOCH FROM
		              max(p.answered_at + make_interval(mins => COALESCE(p.prep_minutes, 0))) - now()) / 60))::int
		              AS prep
		        FROM order_parts p
		       WHERE p.order_id = $1 AND p.status IN ('accepted', 'picked_up')) sub
		WHERE o.id = $1 AND sub.prep IS NOT NULL`, orderID)
	return err
}

// syncPartFieldsTx **ما يُشتقّ من الأقسام للطلب كلِّه** (§2.4):
//
//	ready_at             حين يجهز كلُّ قسمٍ مقبول — وإلّا فارغ
//	sent_to_merchant_at  حين يُقبل كلُّ قسمٍ حيٍّ ويُبلَّغ — وإلّا فارغ
//
// **والطلبُ من متجرٍ واحدٍ لا يُمسّ** (`stops > 1`).
func syncPartFieldsTx(ctx context.Context, q dbtx.Querier, orderID string) error {
	_, err := q.Exec(ctx, `
		UPDATE orders o SET
		  ready_at = CASE
		    WHEN EXISTS (SELECT 1 FROM order_parts p WHERE p.order_id = o.id
		                  AND p.status IN ('accepted', 'picked_up'))
		     AND NOT EXISTS (SELECT 1 FROM order_parts p WHERE p.order_id = o.id
		                  AND (p.status = 'pending' OR (p.status = 'accepted' AND p.ready_at IS NULL)))
		    THEN COALESCE(o.ready_at, (SELECT max(p.ready_at) FROM order_parts p
		                               WHERE p.order_id = o.id AND p.status IN ('accepted', 'picked_up')), now())
		    END,
		  sent_to_merchant_at = CASE
		    WHEN EXISTS (SELECT 1 FROM order_parts p WHERE p.order_id = o.id
		                  AND p.status IN ('accepted', 'picked_up'))
		     AND NOT EXISTS (SELECT 1 FROM order_parts p WHERE p.order_id = o.id
		                  AND (p.status = 'pending' OR (p.status = 'accepted' AND p.sent_at IS NULL)))
		    THEN COALESCE(o.sent_to_merchant_at, (SELECT max(p.sent_at) FROM order_parts p
		                               WHERE p.order_id = o.id AND p.status IN ('accepted', 'picked_up')), now())
		    END
		WHERE o.id = $1 AND o.stops > 1`, orderID)
	return err
}

// partCancelWithinWindowTx **أما زال الزبونُ في مهلة الإلغاء؟** (G7) — والمهلةُ من أوّل قسمٍ قُبل.
// **ولا قسمَ مقبولاً ⇒ يُلغي متى شاء** — كطلبٍ لم يقبله متجر.
func partCancelWithinWindowTx(ctx context.Context, q dbtx.Querier, orderID string, windowSec int64) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `
		SELECT COALESCE(min(answered_at) > now() - make_interval(secs => $2), true)
		FROM order_parts WHERE order_id = $1 AND status IN ('accepted', 'picked_up')`,
		orderID, windowSec).Scan(&ok)
	return ok, err
}

// firstPartAcceptedAt **متى قُبل أوّلُ قسم** — وnil: لم يُقبل شيء.
func (s *Service) firstPartAcceptedAt(ctx context.Context, orderID string) *time.Time {
	var at *time.Time
	_ = s.db.QueryRow(ctx, `
		SELECT min(answered_at) FROM order_parts
		WHERE order_id = $1 AND status IN ('accepted', 'picked_up')`, orderID).Scan(&at)
	return at
}

// partsFollowUpTx **الانتقالُ الذي يتبع جوابَ قسم** (§2.4) — في المعاملة نفسِها.
//
//	لا معلَّق · قسمٌ مقبولٌ فأكثر ⇒ pending → accepted («جاوبت كل المتاجر»)
//	لا معلَّق · لا مقبول          ⇒ pending → rejected («ما توفّر شي من الطلب»)
//
// **والفاعلُ النظام** — **ودورُه دورُ المكتب** فتجيزه الخارطةُ في الوضعين.
func (s *Service) partsFollowUpTx(ctx context.Context, tx pgx.Tx, lo *lockedOrder) (*appliedTransition, error) {
	if lo.from != StPending {
		return nil, nil
	}
	c, err := partCountsTx(ctx, tx, lo.id)
	if err != nil {
		return nil, err
	}
	if c.pending > 0 {
		return nil, nil
	}
	to, note := StAccepted, partsAnsweredNote
	if c.live == 0 {
		to, note = StRejected, allPartsRejectedReason
	}
	return s.applyTransitionTx(ctx, tx, lo, transitionReq{
		actorRoles: []string{"ops"}, to: to, note: note, fromParts: true,
	})
}

// partOutcome **ما وقع في فعل القسم** — يقرؤه ما بعد التثبيت.
type partOutcome struct {
	follow   *appliedTransition
	accepted []partRef
	rejected []partRef
	reason   string
	// noop **لا شيءَ كُتب** — جوابٌ أُعيد على ما هو عليه.
	noop bool
	// removed **أقسامٌ خرجت** (`removePartTx`) · transferTo **قسمٌ جديدٌ بالتحويل.**
	removed    []partRemoval
	transferTo partRef
	// settled **ما قُيّد عند محطّة** (`StopPickup`) — يُبلَّغ أصحابُه بعد التثبيت.
	settled settled
}

// partAction **هيكلُ كلّ فعلِ قسم** — قفلٌ ففعلٌ فانتقالٌ تابعٌ في معاملةٍ واحدة،
// ثمّ ما بعد التثبيت.
func (s *Service) partAction(ctx context.Context, orderID string,
	act func(ctx context.Context, tx pgx.Tx, unit *Service, lo *lockedOrder, out *partOutcome) error) (*Order, *partOutcome, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	lo, err := lockOrderTx(ctx, tx, orderID)
	if err != nil {
		return nil, nil, err
	}
	if lo.stops <= 1 {
		return nil, nil, ErrBadTransition
	}
	unit := s.on(tx)
	out := &partOutcome{}
	if err := act(ctx, tx, unit, lo, out); err != nil {
		return nil, nil, err
	}
	if !out.noop {
		if err := syncPartFieldsTx(ctx, tx, orderID); err != nil {
			return nil, nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	if out.follow != nil {
		updated, err := s.afterTransition(ctx, out.follow)
		return updated, out, err
	}
	updated, err := s.GetByID(ctx, orderID)
	if err != nil {
		return nil, out, err
	}
	if !out.noop {
		s.publishOrder(updated)
	}
	return updated, out, nil
}

// lockPartsTx **أقسامُ متاجرَ بعينها في الطلب — مقفولةً** — ومتجرٌ لا قسمَ له فيه ⇒ «غير موجود».
//
// **و`owner` غيرُ الفارغ يشترط أن يملكها كلَّها** — حزامٌ فوق باب المتجر.
func lockPartsTx(ctx context.Context, q dbtx.Querier, orderID string, stores []string, owner string) (map[string]string, error) {
	rows, err := q.Query(ctx, `
		SELECT p.merchant_id::text, p.status, COALESCE(m.owner_user_id::text, '')
		FROM order_parts p JOIN merchants m ON m.id = p.merchant_id
		WHERE p.order_id = $1 AND p.merchant_id = ANY($2::uuid[])
		FOR UPDATE OF p`, orderID, stores)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	got := map[string]string{}
	for rows.Next() {
		var id, st, own string
		if err := rows.Scan(&id, &st, &own); err != nil {
			return nil, err
		}
		if owner != "" && own != owner {
			return nil, httpx.ErrNotFound
		}
		got[id] = st
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(got) == 0 || len(got) != len(uniqStrings(stores)) {
		return nil, httpx.ErrNotFound
	}
	return got, nil
}

// answerRoles **أدوارُ من يرفض كما يقرؤها الشيل** — المتجرُ متجرٌ، والكانسُ والمكتبُ مكتب.
func answerRoles(actorRoles []string, role string) []string {
	if role == "merchant" {
		return []string{"merchant"}
	}
	return actorRoles
}

// sortBySeq **المتاجرُ بترتيب محطّاتها** — يُكتب حدثُها كما تُزار.
func sortBySeq(ctx context.Context, q dbtx.Querier, orderID string, stores []string) {
	order, err := partStoresOf(ctx, q, orderID)
	if err != nil {
		return
	}
	pos := map[string]int{}
	for i, id := range order {
		pos[strings.ToLower(id)] = i
	}
	sort.SliceStable(stores, func(i, j int) bool {
		return pos[strings.ToLower(stores[i])] < pos[strings.ToLower(stores[j])]
	})
}

func uniqStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		k := strings.ToLower(v)
		if !seen[k] {
			seen[k] = true
			out = append(out, v)
		}
	}
	return out
}

// AnswerParts **متجرٌ (أو المكتبُ) يجيب عن قسمه** — قبولٌ بمدّة تحضير أو رفضٌ بسبب.
//
// **ومن يجيب يتبع وضعَ الطلبات** (`modes.go`): «المنصّة تدير» ⇒ المكتبُ وحدَه؛
// «المتاجر تدير» ⇒ المتجرُ عن قسمه أو المكتب. **ومالكٌ له فرعان في الطلب يجيب عنهما
// معاً** (`stores`). **وإعادةُ الجواب نفسِه لا تكتب شيئاً.**
func (s *Service) AnswerParts(ctx context.Context, actorID string, actorRoles []string, orderID string,
	stores []string, accept bool, prep *int, reason string) (*Order, error) {
	if len(stores) == 0 {
		return nil, httpx.ErrNotFound
	}
	to, want := StRejected, PartRejected
	if accept {
		to, want = StAccepted, PartAccepted
	}
	reason = strings.TrimSpace(reason)
	updated, out, err := s.partAction(ctx, orderID, func(ctx context.Context, tx pgx.Tx, unit *Service,
		lo *lockedOrder, out *partOutcome) error {
		eff := rolesUnderMode(unit.MerchantsSelfManage(ctx), StPending, to, actorRoles, lo.driverID != nil)
		if !canTransition(lo.kind, StPending, to, eff) {
			return ErrBadTransition
		}
		role := authorizingRole(lo.kind, StPending, to, eff)
		owner := ""
		if role == "merchant" {
			owner = actorID
		}
		got, err := lockPartsTx(ctx, tx, lo.id, stores, owner)
		if err != nil {
			return err
		}
		todo := []string{}
		for id, st := range got {
			switch st {
			case want:
			case PartPending:
				todo = append(todo, id)
			default:
				return ErrBadTransition
			}
		}
		if len(todo) == 0 {
			out.noop = true
			return nil
		}
		// **ولا جوابَ بعد أن مضى الطلب** — الجوابُ لطلبٍ «بانتظار» وحدَه.
		if lo.from != StPending {
			return ErrBadTransition
		}
		if !accept {
			// **رفضُ القسم يُخرج بنودَه ويُعاد الحساب** (`removePartTx`) — وآخرُ قسمٍ يرفض الطلب.
			// **وذنبُه على المتجر إن رفض بنفسه** (٦أ)، وعلى المنصّة إن رفضه المكتب.
			fault := FaultPlatform
			if role == "merchant" {
				fault = FaultMerchant
			}
			out.reason = reason
			sortBySeq(ctx, tx, lo.id, todo)
			for _, id := range todo {
				if err := unit.removePartTx(ctx, tx, lo, removePartReq{
					store: id, actorID: actorID, roles: answerRoles(actorRoles, role), reason: reason, fault: fault,
				}, out); err != nil {
					return err
				}
			}
			// **والمكتبُ يُخبَر برفض المتجر من باب الشيل** (`notifyPartRemovals`) — خبرٌ واحد.
			return nil
		}
		refs, err := answerPartsTx(ctx, tx, lo, todo, actorID, partRole(actorID, role), accept, prep, reason,
			[]string{PartPending})
		if err != nil {
			return err
		}
		out.accepted = refs
		out.follow, err = unit.partsFollowUpTx(ctx, tx, lo)
		return err
	})
	if err != nil {
		return nil, err
	}
	if !out.noop {
		s.notifyPartAnswers(ctx, orderID, out)
		if len(out.removed) > 0 {
			s.afterPartRemovals(ctx, orderID, out)
		} else {
			s.maybeDispatchParts(ctx, orderID)
		}
	}
	return updated, nil
}

// partReadyable **حالاتُ الطلب التي يُعلَن فيها قسمٌ جاهزاً** — من قبوله حتّى وقوف
// السائق عند الباب، **والطلبُ قد يكون «بانتظار» غيرِه.**
var partReadyable = map[string]bool{
	StPending: true, StAccepted: true, StPreparing: true, StDispatching: true,
	StAssigned: true, StAtPickup: true,
}

// PartReady **المتجرُ جهّز قسمَه** — وقتٌ لا حال. **والطلبُ كلُّه جاهزٌ حين تجهز أقسامُه
// المقبولةُ كلُّها** (`syncPartFieldsTx`). **وإعلانٌ ثانٍ لا يبدّل الوقتَ الأوّل.**
func (s *Service) PartReady(ctx context.Context, actorID, orderID string, stores []string, owner string) (*Order, error) {
	updated, _, err := s.partAction(ctx, orderID, func(ctx context.Context, tx pgx.Tx, unit *Service,
		lo *lockedOrder, out *partOutcome) error {
		if !partReadyable[lo.from] {
			return ErrPartNotReady
		}
		got, err := lockPartsTx(ctx, tx, lo.id, stores, owner)
		if err != nil {
			return err
		}
		for _, st := range got {
			if st != PartAccepted {
				return ErrPartNotReady
			}
		}
		rows, err := tx.Query(ctx, `
			UPDATE order_parts p SET ready_at = now(), updated_at = now()
			  FROM merchants m
			 WHERE m.id = p.merchant_id AND p.order_id = $1 AND p.merchant_id = ANY($2::uuid[])
			   AND p.status = 'accepted' AND p.ready_at IS NULL
			RETURNING p.merchant_id::text, m.name`, lo.id, stores)
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
		if len(refs) == 0 {
			out.noop = true
			return nil
		}
		for _, r := range refs {
			if err := partEventTx(ctx, tx, lo.id, lo.from, actorID, fmt.Sprintf("جهّز %s قسمه", r.name),
				lo.driverID, r.merchantID); err != nil {
				return err
			}
		}
		return nil
	})
	return updated, err
}

// PartSentChannels **قناتا إبلاغ المتجر بقسمه** — واتساب أو المكتبُ بنفسه.
var PartSentChannels = map[string]string{"whatsapp": "واتساب", "manual": "يدويّاً"}

// MarkPartSent **أُبلغ المتجرُ بقسمه** — `sent_at` القسم، **و`sent_to_merchant_at` للطلب حين
// يُبلَّغ كلُّ قسمٍ حيّ**، ثمّ يُنزَل إن تمّ (`maybeDispatchParts`). **وإعادةُ الإرسال تكتب وقتَها.**
func (s *Service) MarkPartSent(ctx context.Context, actorID, orderID, merchantID, channel string) (*Order, error) {
	label, ok := PartSentChannels[channel]
	if !ok {
		return nil, ErrBadTransition
	}
	updated, _, err := s.partAction(ctx, orderID, func(ctx context.Context, tx pgx.Tx, unit *Service,
		lo *lockedOrder, out *partOutcome) error {
		// **ولا تذكرةَ بعد الاستلام** — البضاعةُ مع السائق، والإرسالُ حدثٌ يُقرأ.
		if terminal(lo.from) || AfterPickup(lo.from) {
			return ErrBadTransition
		}
		got, err := lockPartsTx(ctx, tx, lo.id, []string{merchantID}, "")
		if err != nil {
			return err
		}
		for _, st := range got {
			if st != PartAccepted {
				return ErrBadTransition
			}
		}
		var name string
		if err := tx.QueryRow(ctx, `
			UPDATE order_parts p SET sent_at = now(), sent_channel = $3, updated_at = now()
			  FROM merchants m
			 WHERE m.id = p.merchant_id AND p.order_id = $1 AND p.merchant_id = $2::uuid
			RETURNING m.name`, lo.id, merchantID, channel).Scan(&name); err != nil {
			return err
		}
		return partEventTx(ctx, tx, lo.id, lo.from, actorID,
			fmt.Sprintf("أُبلغ %s بقسمه (%s)", name, label), lo.driverID, merchantID)
	})
	if err != nil {
		return nil, err
	}
	s.maybeDispatchParts(ctx, orderID)
	return updated, nil
}

// maybeDispatchParts **يُنزَل الطلبُ حين يُبلَّغ آخرُ أقسامه** (D5) — كإنزال بابِ الإرسال
// لطلب المتجر الواحد (`handleSendOrderToMerchant`): مقبولٌ أو قيد التحضير، غيرُ خاصّ،
// **و`sent_to_merchant_at` قائمٌ = كلُّ قسمٍ حيٍّ قُبل وأُبلغ.**
//
// **بعد التثبيت لا داخلَه** — الإنزالُ انتقالٌ ثانٍ، وتعثّرُه لا يُبطل إرسالاً وقع.
func (s *Service) maybeDispatchParts(ctx context.Context, orderID string) {
	var st, kind string
	var stops int
	var sent *time.Time
	if err := s.db.QueryRow(ctx,
		`SELECT status, kind, stops, sent_to_merchant_at FROM orders WHERE id = $1`, orderID).
		Scan(&st, &kind, &stops, &sent); err != nil {
		return
	}
	if stops <= 1 || kind == KindCustom || sent == nil || (st != StAccepted && st != StPreparing) {
		return
	}
	if err := s.AutoDispatch(ctx, "", orderID); err != nil && s.logger != nil {
		s.logger.Warn("الأقسام: الإنزالُ بعد إبلاغ آخر متجرٍ تعثّر — ينتظر المكتب",
			"order", orderID, "error", err)
	}
}

// notifyPartAnswers **من يُخبَر بجواب قسم.**
//
//   - **الزبونُ حين يُقبل قسمٌ والطلبُ ما زال ينتظر غيرَه** — «قسم من طلبك انقبل»، يرنّ ولا
//     يُحفَظ؛ **وآخرُ قبولٍ يقول «قبل المتجر طلبك» من آلة الحال** — لا خبران لشيءٍ واحد.
//   - **المكتبُ حين يرفض متجرٌ قسمَه** (C13) — كإنهاء المتجر للطلب اليوم.
//
// **ولا اسمَ متجرٍ في إشعار الزبون** (`P-1`).
func (s *Service) notifyPartAnswers(ctx context.Context, orderID string, out *partOutcome) {
	if s.notify == nil || out == nil {
		return
	}
	var number int64
	var customerID string
	if err := s.db.QueryRow(ctx,
		`SELECT number, COALESCE(customer_id::text, '') FROM orders WHERE id = $1`, orderID).
		Scan(&number, &customerID); err != nil {
		return
	}
	ref := fmt.Sprintf("#%d", number)
	finalized := out.follow != nil && out.follow.to == StAccepted
	if len(out.accepted) > 0 && !finalized && customerID != "" {
		s.notify.Notify(ctx, notifications.Input{
			UserID: customerID, Kind: notifications.KindOrder,
			Title: t.partAccepted, Body: ref,
			Entity: "order", EntityID: orderID, Href: "/portal/orders",
			Apps:      []string{notifications.AppCustomer},
			Transient: true,
		})
	}
	for _, r := range out.rejected {
		body := fmt.Sprintf("%s — %s رفض قسمه", ref, r.name)
		if out.reason != "" {
			body += " · " + out.reason
		}
		s.notify.NotifyRoles(ctx, notifications.OpsDesk, notifications.Input{
			Kind: notifications.KindOrder, Title: t.partRejectedOps, Body: body,
			Entity: "order", EntityID: orderID, Href: "/dashboard/orders",
		})
	}
}

// notifyPartsCreated **«طلب جديد قادم إليك» لكلّ متجرٍ في الطلب — بقسمه هو** (G-01).
//
// **ويردّ أسماءَ المتاجر بترتيبها** لإشعار المكتب.
func (s *Service) notifyPartsCreated(ctx context.Context, orderID, ref string) string {
	rows, err := s.db.Query(ctx, `
		SELECT COALESCE(m.owner_user_id::text, ''), m.name,
		       (SELECT count(*) FROM order_items oi
		         WHERE oi.order_id = p.order_id AND oi.merchant_id = p.merchant_id),
		       (SELECT COALESCE(sum(oi.unit_price * oi.qty), 0) FROM order_items oi
		         WHERE oi.order_id = p.order_id AND oi.merchant_id = p.merchant_id)
		FROM order_parts p JOIN merchants m ON m.id = p.merchant_id
		WHERE p.order_id = $1 ORDER BY p.seq`, orderID)
	if err != nil {
		return ""
	}
	type part struct {
		owner, name string
		count       int
		amount      int64
	}
	var parts []part
	for rows.Next() {
		var p part
		if rows.Scan(&p.owner, &p.name, &p.count, &p.amount) == nil {
			parts = append(parts, p)
		}
	}
	rows.Close()
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		names = append(names, p.name)
		if p.owner == "" {
			continue
		}
		body := ref
		if p.count > 0 {
			body = fmt.Sprintf("%s — %s · %s ل.س", ref, itemsCountAr(p.count), groupDigits(p.amount))
		}
		s.notify.Notify(ctx, notifications.Input{
			UserID: p.owner, Kind: notifications.KindOrder,
			Title: t.newOrderMerchant, Body: body,
			Entity: "order", EntityID: orderID, Href: "/portal",
			Apps: []string{notifications.AppMerchant},
		})
	}
	return strings.Join(names, " + ")
}

// livePartOwners **أصحابُ الأقسام الباقية في الطلب** — مقبولةً أو مستلَمة.
func (s *Service) livePartOwners(ctx context.Context, orderID string) []string {
	rows, err := s.db.Query(ctx, `
		SELECT m.owner_user_id::text
		FROM order_parts p JOIN merchants m ON m.id = p.merchant_id
		WHERE p.order_id = $1 AND p.status IN ('accepted', 'picked_up') AND m.owner_user_id IS NOT NULL
		ORDER BY p.seq`, orderID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			out = append(out, id)
		}
	}
	return out
}
