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
// # ١ · السائق — تعويضٌ يدويّ دائماً
//
// **لا تلقائياً.** أجرُ التوصيل مقابل تسليمٍ تمّ، وما وقع رحلةٌ لا تسليم.
// **وتقديرُ الرحلة يختلف**: مشوارٌ إلى الحيّ المجاور ليس كمشوارٍ عبر المدينة،
// **ورقمٌ آليٌّ واحد يظلم أحدهما**. فيُقدّرها إنسانٌ ويوقّع عليها.
//
// والنقطةُ هنا لا في صفحة المحفظة: **زرٌّ في مكانٍ آخر زرٌّ لا يُضغط.** ومن
// أراد تعويضَ سائقٍ عن طلبٍ بعينه يجده عند الطلب لا في بحثٍ عن اسمه.
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
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	errNotFailed       = httpx.NewError(http.StatusConflict, "order_not_failed", "errors.order_not_failed")
	errGoodsSettled    = httpx.NewError(http.StatusConflict, "goods_already_settled", "errors.goods_already_settled")
	errNoDriverOnOrder = httpx.NewError(http.StatusConflict, "order_has_no_driver", "errors.order_has_no_driver")
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

	var status string
	var driverID *string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT status, driver_id FROM orders WHERE id = $1`, orderID).
		Scan(&status, &driverID); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if status != "failed" {
		s.respondErr(w, errNotFailed)
		return
	}
	if driverID == nil {
		s.respondErr(w, errNoDriverOnOrder)
		return
	}

	actor := userIDFrom(r)
	// **بمرجع الطلب** — فيُقرأ لاحقاً في كشف السائق وفي تقارير الطلب معاً.
	if _, err := s.wallet.Apply(r.Context(), *driverID, req.Amount,
		"compensation", orderID, note, &actor); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "finance.driver_compensation", "order", orderID, map[string]any{
		"driver_id": *driverID, "amount": req.Amount, "note": note,
	})
	s.touch("order", "ops")
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"compensated": req.Amount})
}

// handleSettleGoods يحسم مصير بضاعة طلبٍ فشل.
func (s *Server) handleSettleGoods(w http.ResponseWriter, r *http.Request) {
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
	var settled *string
	var subtotal int64
	var merchantPct int
	var ownerID *string
	// **القفل داخل المعاملة**: ضغطتان متزامنتان تدفعان للمتجر مرّتين لولاه.
	if err := tx.QueryRow(r.Context(), `
		SELECT o.status, o.goods_settled_to, o.subtotal, m.commission_percent, m.owner_user_id
		FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1 FOR UPDATE OF o`, orderID).
		Scan(&status, &settled, &subtotal, &merchantPct, &ownerID); err != nil {
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

	var paid int64
	if req.To == "platform" && ownerID != nil {
		// **ما كان سيقبضه لو نجح الطلب** — بالمعادلة نفسها التي في التسوية،
		// فلا يفترق تعويضُ الفشل عن أجر النجاح بحسبةٍ ثانية تنحرف يوماً.
		if paid = subtotal - subtotal*int64(merchantPct)/100; paid > 0 {
			actor := userIDFrom(r)
			if _, err := s.wallet.ApplyTx(r.Context(), tx, *ownerID, paid,
				"compensation", orderID,
				"تعويضُ بضاعةٍ لم تُسترَدّ — طلبٌ فشل", &actor); err != nil {
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
