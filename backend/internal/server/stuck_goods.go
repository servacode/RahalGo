package server

// ══════════════════════════════════════════════════════════════════════
// **«مصاري الكابتن عالقة»** — النموذج ٢، القرار ٤ (خطّةُ المال ٢٠٢٦-١١-١٠، الدفعة B5)
// ══════════════════════════════════════════════════════════════════════
//
// طلبٌ من النموذج ٢ فشل بعد الاستلام، والبضاعةُ رجعت للمكتب لأنّ المتجرَ رفض يرجّع ثمنها:
// الكابتنُ دفع من جيبه ولم يسترجع. **والمكتبُ يقرّر كلَّ مرّة** (جوابُ المالك):
//
//	platform  المنصّةُ تعوّض الكابتن (من الخزينة إلى محفظته)
//	customer  دينٌ على الزبون — سجلٌّ لا يحرّك مالاً الآن
//
// **بقدرة `finance.manage`، ومرّةً واحدة** — والمحرّكُ يحرس التكرار (`DecideStuckGoodsTx`).

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

func (s *Server) handleStuckGoodsDecision(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	req, err := decode[struct {
		// Decision **platform | customer**.
		Decision string `json:"decision"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	var amount int64
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		res, err := s.orders.DecideStuckGoodsTx(ctx, q, orderID, req.Decision, userIDFrom(r))
		if err != nil {
			return err
		}
		amount = res.Amount
		return s.auditTx(ctx, q, r, "finance.stuck_goods_decided", "order", orderID, map[string]any{
			"decision": req.Decision, "amount": res.Amount, "number": res.Number,
		})
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("order", "ops")
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"decision": req.Decision, "amount": amount})
}
