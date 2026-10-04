package server

// ما بعد فشل الطلب — من يحمل الخسارة.
//
// # المسألة
//
// `failed` كانت نهايةً محاسبيةً ناقصة: يُردّ للزبون ما دفع، **ثم لا شيء**.
// والطعامُ مطبوخٌ بيد السائق، والسائقُ قاد، **ولا قيدَ يقول من يحمل الثمن.**
//
// وقاعدةُ المالك «المنصةُ والزبون لا يخسران» تحسم طرفين من أربعة — **ويبقى
// المتجرُ والسائق.** فهذان قرارُه (٢٠٢٦-٠٨-٠١):
//
// # ١ · السائق — **بموافقة إنسانٍ لا لحظةَ الضغطة** (قرارُ المالك ٢٠٢٦-١٠-٠٢)
//
// **كان هذا الرأسُ يقول «يدويٌّ دائماً» والمحرّكُ يقيّد تلقائيّاً** — نصفَ أجر
// السائق في معاملة الفشل نفسِها (`compensateDriverOnFail`). **وثيقتان تتناقضان
// صامتتين.** وقِيس على التجهيز: ٥٬٠٠٠ تُدفع فوراً في كلّ ضغطة، **حتّى لـ«تأخّرتُ
// أنا».** فسائقٌ يكسب بضغطة.
//
// **فصارت القاعدةُ واحدةً في الموضعين**: المحرّكُ يكتب **طلبَ تعويضٍ معلَّقاً**
// (`orders/compensation_requests.go`) — حين يكون الذنبُ على الزبون أو المتجر
// وحدَهما، ومعه **مبلغٌ مقترَح** بالمعادلة القديمة — **ويُنبَّه المكتب.**
// **وهذا البابُ هو الموافقة**: يقيّد المبلغَ ويُغلق الطلبَ المعلَّق في المعاملة
// نفسِها، **ويفتح المطالبةَ على المتجر** إن كان الذنبُ ذنبَه. ورفضُه بابٌ
// بجانبه (`handleRejectCompensation`). **ولا قيدَ تعويضٍ تلقائيٌّ في أيّ مكان.**
//
// **وما لا طلبَ معلَّقاً له يبقى كما كان**: تعويضٌ يدويٌّ عن طلبٍ فشل بتقدير
// إنسان. والنقطةُ هنا لا في صفحة المحفظة: **زرٌّ في مكانٍ آخر زرٌّ لا يُضغط.**
//
// وتُقيَّد بمرجع الطلب — **فتعويضٌ بلا مرجع مالٌ خرج بلا سبب يُقرأ.**
//
// # ٢ · البضاعة — إلى المكتب ثم أحدُ أمرين
//
//   - **المتجرُ يستردّها**: لا قيد. أخذ بضاعته وانتهى.
//   - **المتجرُ يرفض**: **المنصةُ تتحمّل كاملاً** وتدفع له.
//
// **وكم تدفع؟** ما كان سيقبضه لو نجح الطلب — البضاعةُ ناقصَ عمولة المنصة.
// **فيُجبَر تماماً ولا يُعطى أكثر من بيعةٍ ناجحة**، وعمولةُ المنصة تبقى صفراً
// **فلا تربح من فشل** بل تتحمّل التوصيل كلَّه.

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

var (
	errNotFailed        = httpx.NewError(http.StatusConflict, "order_not_failed", "errors.order_not_failed")
	errGoodsSettled     = httpx.NewError(http.StatusConflict, "goods_already_settled", "errors.goods_already_settled")
	errNoDriverOnOrder  = httpx.NewError(http.StatusConflict, "order_has_no_driver", "errors.order_has_no_driver")
	errGoodsFlowChanged = httpx.NewError(http.StatusGone, "goods_flow_changed", "errors.goods_flow_changed")

	// **وتعويضٌ وقع لا يقع مرّتين** — انظر الشرحَ عند `handleCompensateDriver`.
	errAlreadyCompensated = httpx.NewError(http.StatusConflict,
		"driver_already_compensated", "errors.driver_already_compensated")

	// **ولا رفضَ لما لا ينتظر** — طلبٌ قُضي فيه أو لم يُطلَب أصلاً.
	errCompensationNotPending = httpx.NewError(http.StatusConflict,
		"compensation_not_pending", "errors.compensation_not_pending")
)

// handleCompensateDriver **الموافقةُ على تعويضِ سائقٍ معلَّق — بالبابِ القديم.**
//
// ══════════════════════════════════════════════════════════════════════
// **ولا تعويضَ مباشرٌ بعد اليوم** (قرارُ المالك ٢٠٢٦-١٠-٠٤: طريقُ موافقةٍ واحد)
// ══════════════════════════════════════════════════════════════════════
//
// كان هذا البابُ يدفع مباشرةً إن لم يجد طلباً معلَّقاً — **موظّفٌ واحدٌ يقترح
// ويوافق.** والآن يوافق على المعلَّق وحدَه، **ومن أراد تعويضاً جديداً اقترحه**
// (`POST /compensations`) فوافق غيرُه.
//
// **وإن كان للطلب أكثرُ من معلَّقٍ لزم `request_id`** — كان يأخذ الأقدمَ فيدفع
// لسائقٍ غيرِ المعروض. **والصفحةُ تستعمل `POST /compensations/{id}/approve`.**
//
// **وتعويضٌ وقع لا يقع مرّتين** (فحصُ التغطية ٢٠٢٦-٠٨-٠٨): القفلُ على صفّ الطلب
// المعلَّق، وفحصُ القيد السابق بالمرجع نفسِه، في المعاملة نفسِها.
func (s *Server) handleCompensateDriver(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	req, err := decode[struct {
		Amount    int64  `json:"amount"`
		Note      string `json:"note"`
		RequestID string `json:"request_id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	note := strings.TrimSpace(req.Note)
	if req.Amount <= 0 || note == "" {
		s.respondErr(w, errValidation)
		return
	}
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		var c *orders.CompensationRequest
		var err error
		if req.RequestID != "" {
			if !isUUID(req.RequestID) {
				return errValidation
			}
			c, err = s.orders.CompensationByIDTx(ctx, q, req.RequestID)
			if err == nil && c != nil && c.OrderID != orderID {
				c = nil
			}
		} else {
			c, err = s.orders.PendingCompensationTx(ctx, q, orderID)
		}
		if err != nil {
			return compensationErr(err)
		}
		if c == nil {
			// **وما عُوِّض يُقال إنّه عُوِّض** — لا «لا معلَّق» يُحيّر.
			var already bool
			if err := q.QueryRow(ctx, `
				SELECT EXISTS(SELECT 1 FROM wallet_transactions t
				              JOIN orders o ON o.id::text = t.ref AND o.driver_id = t.user_id
				              WHERE t.ref = $1::text AND t.kind = 'compensation')
				    OR EXISTS(SELECT 1 FROM driver_compensation_requests
				              WHERE order_id = $1::uuid AND kind = 'driver' AND status = 'approved')`,
				orderID).Scan(&already); err != nil {
				return err
			}
			if already {
				return errAlreadyCompensated
			}
			return errCompensationNotPending
		}
		return s.approveCompensationTx(ctx, q, r, c, req.Amount, note)
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("order", "ops")
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"compensated": req.Amount})
}

// handleRejectCompensation **رفضُ طلب تعويضٍ معلَّق بالبابِ القديم** — بسببٍ إلزاميّ.
//
// **والرفضُ قرارٌ يُكتب لا صمتٌ يُترك.** وللطلب أكثرُ من معلَّقٍ ⇒ `request_id`.
func (s *Server) handleRejectCompensation(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	req, err := decode[struct {
		Note      string `json:"note"`
		RequestID string `json:"request_id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	note := strings.TrimSpace(req.Note)
	if note == "" {
		s.respondErr(w, errValidation)
		return
	}
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		var c *orders.CompensationRequest
		var err error
		if req.RequestID != "" && isUUID(req.RequestID) {
			c, err = s.orders.CompensationByIDTx(ctx, q, req.RequestID)
			if err == nil && c != nil && c.OrderID != orderID {
				c = nil
			}
		} else {
			c, err = s.orders.PendingCompensationTx(ctx, q, orderID)
		}
		if err != nil {
			return compensationErr(err)
		}
		if c == nil {
			return errCompensationNotPending
		}
		return s.rejectCompensationTx(ctx, q, r, c, note)
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"rejected": true})
}

// handlePendingCompensations **ما ينتظر قراراً** — بالأقدم أوّلاً (يبقى للرئيسيّة).
func (s *Server) handlePendingCompensations(w http.ResponseWriter, r *http.Request) {
	pg := pagingOf(r, 20)
	list, total, err := s.orders.PendingCompensations(r.Context(), pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, paged("compensations", list, total, pg))
}

// handleSettleGoods **مهجورة** — كانت تعوّض المتجرَ عن بضاعةٍ لم يستردّها.
//
// # لماذا بطلت
//
// وُضعت حين كان المتجرُ **يقبض عند التسليم**: يفشل الطلبُ فلا يقبض شيئاً،
// فتُسأل العملياتُ «أنعوّضه؟».
//
// **وبعد أن صار يقبض عند خروج البضاعة، صار استعمالُها دفعاً ثانياً**: مالٌ
// قبضه عند الاستلام يُدفع له مرّةً أخرى باسم التعويض. **ونقطةٌ تعمل بمنطقٍ
// انقضى أخطرُ من نقطةٍ لا تعمل** — هذه تُردّ وتلك تدفع.
//
// **والسؤالُ انقلب**: لم يعد «أنعوّضه؟» بل **«أنستردّ منه؟»** — وذاك
// handleReturnToMerchant.
//
// وتُبقى مردودةً لا محذوفة: **واجهةٌ تختفي فجأةً تُسقط شاشةً لم تُحدَّث بعد**،
// وردٌّ صريحٌ يقول ما جرى.
func (s *Server) handleSettleGoods(w http.ResponseWriter, r *http.Request) {
	s.respondErr(w, errGoodsFlowChanged)
}

// handleGoods يحسم مصيرَ بضاعةِ طلبٍ فشل — **زرّان في شاشة العمليات.**
//
// # ولماذا هنا لا عند السائق
//
// كان زرُّ «أعدتُ البضاعة» في شاشة السائق، **وإقرارُ من يحملها ليس تسليماً**:
// يقع الأمرُ بين يدي من يستلمها في المكتب، وهو من يعرف أاستردّها المتجرُ أم
// رفض. (قرارُ المالك ٢٠٢٦-٠٨-٠٤.)
//
// # والزرّان معاً لمن يستردّ
//
// متجرٌ يستردّ نظاماً **قد يكون مغلقاً يومَها أو يرفض هذه بعينها** — فلو تبع
// الزرُّ بندَ الاسترداد حرفياً لَبقي الطلبُ معلّقاً بلا مخرج. **ومن لا يستردّ
// يرى «إلى المكتب» وحدَه** — لا يُعرض عليه ما لا يقع.
//
// **والحسابُ في المحرّك لا هنا** (`orders/goods.go`): هذه تقرأ الطلبَ وتنادي.
func (s *Server) handleGoods(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	req, err := decode[struct {
		To string `json:"to"` // merchant | platform
		// Compensation **تعويضُ الإدارة للمتجر** عن بضاعةٍ رُدّت إليه — وصفرُه أو غيابُه
		// لا تعويض (قرارُ المالك ٢٠٢٦-١٠-٠٣). **يُدفع من الخزينة مرّةً واحدة.**
		Compensation int64 `json:"compensation"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.To != orders.GoodsToMerchant && req.To != orders.GoodsToOffice {
		s.respondErr(w, errValidation)
		return
	}
	// ══════════════════════════════════════════════════════════════════
	// **والتعويضُ ليس من هذا الباب** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ١٢)
	// ══════════════════════════════════════════════════════════════════
	//
	// **العمليّاتُ تقرّر أين البضاعة، والماليّةُ تكتب المبلغ** — بابُه
	// `POST /orders/{id}/goods/compensation` بقدرة `finance.manage` وتأكيدِ كلمة السرّ.
	if req.Compensation != 0 {
		s.respondErr(w, errGoodsCompFinance)
		return
	}

	switch err := s.orders.SettleGoods(r.Context(), orderID, req.To, userIDFrom(r), 0); {
	case err == nil:
	case errors.Is(err, orders.ErrGoodsBadCompensation):
		s.respondErr(w, errValidation)
		return
	case errors.Is(err, orders.ErrGoodsNotHanded):
		s.respondErr(w, errGoodsNotHanded)
		return
	case errors.Is(err, orders.ErrGoodsWrongPlace):
		s.respondErr(w, errGoodsWrongPlace)
		return
	case errors.Is(err, orders.ErrGoodsNotFailed):
		s.respondErr(w, errNotFailed)
		return
	case errors.Is(err, orders.ErrGoodsAlreadySettled):
		s.respondErr(w, errGoodsSettled)
		return
	case errors.Is(err, orders.ErrGoodsLedgerMismatch):
		// **ولا يُسترجع بالتقدير**: مالُ الناس لا يُقاس بالتقريب، **ورفضٌ
		// صريحٌ يُقرأ خيرٌ من قيدٍ يمرّ وهو خطأ.**
		s.respondErr(w, httpx.NewError(http.StatusConflict,
			"goods_ledger_mismatch", "errors.goods_ledger_mismatch"))
		return
	default:
		s.respondErr(w, err)
		return
	}

	s.audit(r, "finance.goods_settled", "order", orderID, map[string]any{
		"to": req.To,
	})
	s.touch("order", "ops")
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"settled_to": req.To})
}

var (
	errGoodsCompFinance = httpx.NewError(http.StatusForbidden,
		"goods_compensation_finance", "errors.goods_compensation_finance")
	errGoodsNotHanded = httpx.NewError(http.StatusConflict,
		"goods_not_handed", "errors.goods_not_handed")
	errGoodsWrongPlace = httpx.NewError(http.StatusConflict,
		"goods_wrong_place", "errors.goods_wrong_place")
	errGoodsCompCap = httpx.NewError(http.StatusBadRequest,
		"goods_compensation_cap", "errors.goods_compensation_cap")
	errGoodsCompensated = httpx.NewError(http.StatusConflict,
		"goods_already_compensated", "errors.goods_already_compensated")
)

// handleGoodsCompensation **تعويضُ الماليّة للمتجر عن بضاعةٍ رُدّت إليه.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤، البندان ١٢ و١٣.) **بقدرة `finance.manage`، وتأكيدِ
// كلمة السرّ** (`authz/sensitive.go` — كأخيه تعويضِ السائق)، **وسقفُه سعرُ شراء
// البضاعة الراجعة** — «حدا بيكتب ٥٠٠٠٠٠ بدل ٥٠٠٠٠ وبينقيّد». والحسابُ في المحرّك
// (`orders.CompensateGoods`).
func (s *Server) handleGoodsCompensation(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	req, err := decode[struct {
		Amount int64 `json:"amount"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// ══════════════════════════════════════════════════════════════════
	// **والمبلغُ اقتراحٌ يمرّ بصفحة «التعويضات»** (قرارُ المالك ٢٠٢٦-١٠-٠٤:
	// طريقُ موافقةٍ واحدٌ لكلّ تعويض) — كان يُدفع بضغطة كاتبه. **والسقفُ يُفحص
	// هنا وعند الموافقة معاً**، ولا يوافق عليه كاتبُه.
	// ══════════════════════════════════════════════════════════════════
	var id string
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		if err := orders.GoodsCompensableTx(ctx, q, orderID, req.Amount); err != nil {
			return compensationErr(err)
		}
		var owner *string
		var fault string
		if err := q.QueryRow(ctx, `
			SELECT m.owner_user_id::text, COALESCE(o.fault, '')
			FROM orders o JOIN merchants m ON m.id = o.merchant_id WHERE o.id = $1`,
			orderID).Scan(&owner, &fault); err != nil {
			return err
		}
		if owner == nil {
			return errValidation
		}
		var e error
		id, e = orders.ProposeCompensationTx(ctx, q, orders.CompensationProposal{
			Kind: orders.CompKindMerchantGoods, OrderID: orderID, BeneficiaryID: *owner,
			Fault: fault, Amount: req.Amount, ProposedBy: userIDFrom(r),
		})
		if e != nil {
			return compensationErr(e)
		}
		return s.auditTx(ctx, q, r, "finance.compensation_proposed", "order", orderID, map[string]any{
			"request_id": id, "kind": orders.CompKindMerchantGoods, "amount": req.Amount,
		})
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusAccepted, map[string]any{"proposed": id})
}

// handleSettleGoodsLegacy الجسدُ القديم — يبقى للقراءة لا للنداء.
//
//nolint:unused
func (s *Server) handleSettleGoodsLegacy(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	req, err := decode[struct {
		To string `json:"to"` // merchant | platform
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.To != "merchant" && req.To != "platform" {
		s.respondErr(w, errValidation)
		return
	}

	tx, err := s.pg.Begin(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var status string
	var snapPct *int64
	var settled *string
	var subtotal int64
	// **تجاوزُ المتجر** — وفراغُه «اتبع العامّ».
	var merchantPct *int64
	var ownerID *string
	// **القفل داخل المعاملة**: ضغطتان متزامنتان تدفعان للمتجر مرّتين لولاه.
	if err := tx.QueryRow(r.Context(), `
		SELECT o.status, o.goods_settled_to, o.subtotal, m.commission_percent,
		       o.snap_merchant_commission_percent, m.owner_user_id
		FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1 FOR UPDATE OF o`, orderID).
		Scan(&status, &settled, &subtotal, &merchantPct, &snapPct, &ownerID); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if status != "failed" {
		s.respondErr(w, errNotFailed)
		return
	}
	if settled != nil {
		s.respondErr(w, errGoodsSettled)
		return
	}

	// ══════════════════════════════════════════════════════════════
	// **وبالنسبة الملقوطة لا نسبةِ اليوم** — `XQ-2` · `XG-25`
	// ══════════════════════════════════════════════════════════════
	//
	// **تعويضُ الفشل يُحسب بمعادلة التسوية نفسِها** — **فلو قُرئت
	// نسبةُ اليوم لَافترق التعويضُ عن الأجر الذي كان سيُقبَض.**
	if merchantPct == nil && snapPct != nil {
		merchantPct = snapPct
	}
	var paid int64
	if req.To == "platform" && ownerID != nil {
		// **ما كان سيقبضه لو نجح الطلب** — بالمعادلة نفسها التي في التسوية،
		// فلا يفترق تعويضُ الفشل عن أجر النجاح بحسبةٍ ثانية تنحرف يوماً.
		if paid = subtotal - pricing.MerchantCommission(r.Context(), s.settings, merchantPct).Of(subtotal); paid > 0 {
			actor := userIDFrom(r)
			if _, err := s.wallet.ApplyTx(r.Context(), tx, *ownerID, paid,
				"compensation", orderID,
				"تعويضُ بضاعةٍ لم تُسترَدّ — طلبٌ فشل", &actor); err != nil {
				s.respondErr(w, err)
				return
			}
			// **«المنصةُ تتحمّل كاملاً» تُكتب في دفترها لا في نيّتها.**
			if err := s.orders.DebitTreasury(r.Context(), tx, paid,
				orderID, "بضاعةٌ لم يستردّها المتجر — تحمّلتها المنصة", actor); err != nil {
				s.respondErr(w, err)
				return
			}
		}
	}

	if _, err := tx.Exec(r.Context(),
		`UPDATE orders SET goods_settled_to = $2 WHERE id = $1`, orderID, req.To); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.respondErr(w, err)
		return
	}

	s.audit(r, "ops.goods_settled", "order", orderID, map[string]any{
		"to": req.To, "paid": paid,
	})
	s.touch("order", "ops")
	if paid > 0 {
		s.touch("wallet", "ops")
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"to": req.To, "paid": paid})
}
