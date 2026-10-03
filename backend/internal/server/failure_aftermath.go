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
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

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

// handleCompensateDriver تعويضُ سائقٍ عن طلبٍ فشل — بمبلغٍ يقدّره إنسان.
func (s *Server) handleCompensateDriver(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	req, err := decode[struct {
		Amount int64  `json:"amount"`
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **السببُ إلزاميّ**: مالٌ يخرج من المنصة بتقدير موظّف، وبلا كلمةٍ لا
	// يُراجَع ولا يُقاس من يُكثره.
	note := strings.TrimSpace(req.Note)
	if req.Amount <= 0 || note == "" {
		s.respondErr(w, errValidation)
		return
	}

	/* ══════════════════════════════════════════════════════════════════
	   **وتعويضٌ وقع لا يقع مرّتين**
	   ══════════════════════════════════════════════════════════════════

	   (كشفه فحصُ التغطية ٢٠٢٦-٠٨-٠٨، وأُصلح بقرار المالك: «نعم ابدأ بالخمسة».)

	   كانت الحالةُ تُقرأ خارجَ المعاملة وبلا قفل، **ولا شيءَ يُعلَّم بعدها**:
	   الطلبُ يبقى `failed` بعد التعويض كما كان قبله.

	   **فمن نادى مرّتين دُفع التعويضُ مرّتين وخُصمت الخزينةُ مرّتين.**
	   قِيس: نداءان متتاليان ← ٦٠٠٠ بدل ٣٠٠٠، وكلاهما ٢٠٠.

	   **وليس سباقاً**: ضغطةٌ مكرّرةٌ أو شبكةٌ أعادت الإرسال تكفي. **والسباقُ
	   يزيده سوءاً فقط.**

	   **والعادةُ موجودةٌ في المشروع**: `settleMerchant` تفحص وجودَ قيدٍ
	   بالمرجع نفسِه قبل أن تقيّد. **وهذا المسلكُ كان خارجَها.**

	   **والقفلُ على صفّ الطلب هو ما يجعل الفحصَ صادقاً**: فحصٌ بلا قفلٍ يمرّ
	   عليه اثنان معاً — وهو الدرسُ نفسُه من دفعة السحب.
	   ══════════════════════════════════════════════════════════════════ */
	actor := userIDFrom(r)
	tx, err := s.pg.Begin(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var status string
	var driverID *string
	if err := tx.QueryRow(r.Context(),
		`SELECT status, driver_id FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&status, &driverID); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	// **وطلبُ تعويضٍ معلَّقٌ يسبق كلَّ شرط** (٢٠٢٦-١٠-٠٢): تعذّرُ المتجر لا يُفشل
	// الطلب — يعود إلى المكتب حيّاً **والسائقُ حُرّر منه** — فلا `failed` ولا
	// `driver_id`. **والسائقُ المستحقُّ مكتوبٌ في الطلب المعلَّق نفسِه.**
	pending, err := s.orders.PendingCompensationTx(r.Context(), tx, orderID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if pending != nil {
		driverID = &pending.DriverID
	} else {
		if status != "failed" {
			s.respondErr(w, errNotFailed)
			return
		}
		if driverID == nil {
			s.respondErr(w, errNoDriverOnOrder)
			return
		}
	}

	var already bool
	if err := tx.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM wallet_transactions
		              WHERE ref = $1 AND kind = 'compensation' AND user_id = $2)`,
		orderID, *driverID).Scan(&already); err != nil {
		s.respondErr(w, err)
		return
	}
	if already {
		s.respondErr(w, errAlreadyCompensated)
		return
	}

	// **بمرجع الطلب** — فيُقرأ لاحقاً في كشف السائق وفي تفصيل الطلب معاً.
	if _, err := s.wallet.ApplyTx(r.Context(), tx, *driverID, req.Amount,
		"compensation", orderID, note, &actor); err != nil {
		s.respondErr(w, err)
		return
	}
	// **ويخرج من الخزينة في القيد نفسه.**
	//
	// تعويضٌ يُقيَّد للسائق وحده يجعل المنصةَ تظهر رابحةً وهي تدفع — **والربحُ
	// الذي لا يعرف مصاريفه ليس ربحاً.** ومعاً في معاملةٍ واحدة: أحدُهما بلا
	// الآخر دفترٌ لا يتوازن.
	if err := s.orders.DebitTreasury(r.Context(), tx, req.Amount,
		orderID, "تعويضُ سائقٍ عن طلبٍ فشل", actor); err != nil {
		s.respondErr(w, err)
		return
	}
	// **والطلبُ المعلَّقُ يُغلق بالموافقة في المعاملة نفسِها** — فلا يبقى في
	// قائمة الانتظار مالٌ قُيّد. **والمطالبةُ على المتجر تُفتح بما دُفع فعلاً**
	// (قرارُ ٢٠٢٦-٠٨-٠٣ باقٍ: المنصةُ تعوّض وتفتح نزاعاً مع المتجر).
	if pending != nil {
		if err := s.orders.DecideCompensationTx(r.Context(), tx, pending.ID,
			orders.CompensationApproved, req.Amount, actor, note); err != nil {
			s.respondErr(w, err)
			return
		}
		if pending.Fault == orders.FaultMerchant {
			if err := s.orders.OpenMerchantClaimTx(r.Context(), tx, orderID, req.Amount); err != nil {
				s.respondErr(w, err)
				return
			}
		}
	}
	// **والأثرُ في المعاملة نفسِها — لا بعد التثبيت** (`AQ-4`/`PF-06`): تعويضٌ
	// خرج والخزينةُ خُصمت، **فسقوطُ سطر التدقيق بعد التثبيت يترك مالاً تحرّك
	// بلا من ولا متى، وإعادةُ النداء تُردّ `already_compensated` فلا يُستدرَك.**
	if err := s.auditTx(r.Context(), tx, r, "finance.driver_compensation", "order", orderID, map[string]any{
		"driver_id": *driverID, "amount": req.Amount, "note": note,
		"request_id": requestID(pending),
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("order", "ops")
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"compensated": req.Amount})
}

// requestID معرّفُ الطلب المعلَّق في الأثر — وفراغٌ للتعويض اليدويّ الصرف.
func requestID(p *orders.CompensationRequest) string {
	if p == nil {
		return ""
	}
	return p.ID
}

// handleRejectCompensation **رفضُ طلب تعويضٍ معلَّق** — بسببٍ إلزاميّ.
//
// **والرفضُ قرارٌ يُكتب لا صمتٌ يُترك**: طلبٌ لا يُقضى فيه يبقى في القائمة أبداً،
// **ومن سأل السائقُ عنه بعد شهرٍ لم يجد من يقول لماذا لم يُعوَّض.**
func (s *Server) handleRejectCompensation(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	req, err := decode[struct {
		Note string `json:"note"`
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
	ctx := r.Context()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	pending, err := s.orders.PendingCompensationTx(ctx, tx, orderID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if pending == nil {
		s.respondErr(w, errCompensationNotPending)
		return
	}
	if err := s.orders.DecideCompensationTx(ctx, tx, pending.ID,
		orders.CompensationRejected, 0, userIDFrom(r), clip(note, 300)); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.auditTx(ctx, tx, r, "finance.driver_compensation_rejected", "order", orderID, map[string]any{
		"driver_id": pending.DriverID, "request_id": pending.ID, "note": note,
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"rejected": true})
}

// handlePendingCompensations **ما ينتظر قراراً من طلبات التعويض** — بالأقدم أوّلاً.
//
// **وكلُّ صفٍّ يحمل ما يلزم القرار**: الطلبُ وحالُه، والسائقُ، والذنبُ والسبب،
// **والمبلغُ المقترَح** — فيُوافَق عليه كما هو أو يُعدَّل أو يُرفض.
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

	switch err := s.orders.SettleGoods(r.Context(), orderID, req.To, userIDFrom(r), req.Compensation); {
	case err == nil:
	case errors.Is(err, orders.ErrGoodsBadCompensation):
		s.respondErr(w, errValidation)
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
		"to": req.To, "compensation": req.Compensation,
	})
	s.touch("order", "ops")
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"settled_to": req.To})
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
