package server

// إرجاعُ البضاعة إلى المتجر — بعد تعذّر التسليم.
//
// # المسألة
//
// المتجرُ قبض ثمنَ بضاعته **لحظةَ خروجها من يده**. فإن تعذّر التسليمُ صار
// الطعامُ ملكَ المنصة — تحمّلت ثمنَه.
//
// **ومنهم من يستردّ ومنهم من يرفض** — سياسةُ متجرٍ لا قاعدةُ منصة
// (`merchants.accepts_returns`).
//
//   - **يستردّ**: يعيده السائقُ **ويُخصم مستحقُّه تلقائياً**. فلا هو خسر ولا
//     ربح — أخذ بضاعتَه وردّ ثمنَها.
//   - **يرفض**: يعود السائقُ بالطعام إلى المكتب، **والمنصةُ تتحمّل كاملاً**.
//
// **ولو تُرك له المالُ والبضاعةُ معاً لربح من الفشل أكثرَ ممّا يربح من
// النجاح** — وهو حافزٌ مقلوب.
//
// # ولماذا لا ذنبَ للطعام بمن أخطأ
//
// يُعرض الإرجاعُ **مهما كان الذنب**: على الزبون أو على السائق. **والفرقُ في
// التعويض لا في مصير الطعام** — ومتجرٌ يستردّ بضاعتَه يستردّها في الحالين.

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	errNoReturns       = httpx.NewError(http.StatusConflict, "merchant_no_returns", "errors.merchant_no_returns")
	errAlreadyReturned = httpx.NewError(http.StatusConflict, "already_returned", "errors.already_returned")
	errNotReturnable   = httpx.NewError(http.StatusConflict, "order_not_returnable", "errors.order_not_returnable")
)

// handleDriverReturn يعيد بضاعةَ طلبٍ تعذّر تسليمُه إلى متجرها.
func (s *Server) handleDriverReturn(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	if !s.driverOwnsOrder(r, orderID) {
		s.respondErr(w, errNotYourOrder)
		return
	}

	tx, err := s.pg.Begin(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var status string
	var returned *string
	var accepts bool
	var ownerID *string
	// **القفلُ داخل المعاملة**: ضغطتان متزامنتان تخصمان من المتجر مرّتين.
	if err := tx.QueryRow(r.Context(), `
		SELECT o.status, o.returned_at::text, m.accepts_returns, m.owner_user_id
		FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1 FOR UPDATE OF o`, orderID).
		Scan(&status, &returned, &accepts, &ownerID); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if status != "failed" {
		s.respondErr(w, errNotReturnable)
		return
	}
	if returned != nil {
		s.respondErr(w, errAlreadyReturned)
		return
	}
	if !accepts {
		s.respondErr(w, errNoReturns)
		return
	}

	// **ما قُيّد له فعلاً يُعكس** — لا ما تقول المعادلةُ إنه قُيّد.
	//
	// عمولتُه قد تكون تغيّرت منذ الاستلام، **وحسبتُها من جديدٍ تعكس مبلغاً
	// غيرَ الذي دُفع** — فيبقى فرقٌ في محفظته بلا سبب.
	var paid int64
	if err := tx.QueryRow(r.Context(), `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'merchant_earning'`, orderID).Scan(&paid); err != nil {
		s.respondErr(w, err)
		return
	}

	actor := userIDFrom(r)
	if paid > 0 && ownerID != nil {
		if _, err := s.wallet.ApplyTx(r.Context(), tx, *ownerID, -paid,
			"adjustment", orderID, "طلب مسترد — أُعيدت البضاعة", &actor); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	if _, err := tx.Exec(r.Context(),
		`UPDATE orders SET returned_at = now() WHERE id = $1`, orderID); err != nil {
		s.respondErr(w, err)
		return
	}
	// **والخزينةُ تُعيد الحساب** — فيعود إليها ما دفعته، وتبقى خسارتُها
	// تعويضَ السائق وحدَه.
	if err := s.orders.CreditTreasuryTx(r.Context(), tx, orderID, actor); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.respondErr(w, err)
		return
	}

	s.audit(r, "driver.order_returned", "order", orderID, map[string]any{"amount": paid})
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"returned": true, "reversed": paid})
}

// handleFailReasons أسبابُ التعذّر المعرَّفة لحالةٍ بعينها.
//
// **والرموزُ تأتي من الخادم لا تُكتب في التطبيق**: قائمةٌ تُكرَّر في مكانين
// **تفترق حين يُضاف سببٌ في أحدهما** — فيرسل السائقُ رمزاً لا يعرفه الخادم،
// ويُردّ عليه بلا أن يفهم لماذا.
func (s *Server) handleFailReasons(w http.ResponseWriter, r *http.Request) {
	type reason struct {
		Code  string `json:"code"`
		Fault string `json:"fault"`
	}
	out := []reason{}
	for _, x := range s.orders.FailReasonsAt(r.URL.Query().Get("at")) {
		out = append(out, reason{Code: x.Code, Fault: x.Fault})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"reasons": out})
}
