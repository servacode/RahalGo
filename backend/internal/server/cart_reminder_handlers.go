package server

// ══════════════════════════════════════════════════════════════════════
// **«سلّتك ناطرتك»** — أبوابُ الزبون والمكنسة (قرارُ المالك ٢٠٢٦-١١-٢٥)
// ══════════════════════════════════════════════════════════════════════
//
// المنطقُ كلُّه في `internal/cartreminder`. **السلّةُ عند العميل** (التطبيقُ و/shop)، فيُبلغ حالَها هنا
// عند كلّ تغيير ويُفرغها عند الإفراغ والطلب. **والتذكيرُ نفسُه مفتاحُه `cart_reminder.enabled`
// (مطفأٌ افتراضاً)** — والبلاغُ يُخزَّن دائماً ليعمل التذكيرُ فور التشغيل.

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/cartreminder"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// handleCartState `PUT /my/cart-state` {merchant_id, items_count, signature} — حالُ السلّة الآن.
func (s *Server) handleCartState(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		MerchantID string `json:"merchant_id"`
		ItemsCount int    `json:"items_count"`
		Signature  string `json:"signature"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.cartReminder.Report(r.Context(), userIDFrom(r), req.MerchantID, req.ItemsCount, req.Signature); err != nil {
		if errors.Is(err, cartreminder.ErrInvalid) {
			s.respondErr(w, errValidation)
			return
		}
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleCartStateClear `DELETE /my/cart-state` — السلّةُ فُرّغت أو صارت طلباً.
func (s *Server) handleCartStateClear(w http.ResponseWriter, r *http.Request) {
	if err := s.cartReminder.Clear(r.Context(), userIDFrom(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// RunCartReminder **نبضةُ تذكير السلّة** — دقيقةٌ تكفي (التأخيرُ ساعة، والختمُ يمنع التكرار).
func (s *Server) RunCartReminder(ctx context.Context, interval time.Duration) {
	s.cartReminder.Run(ctx, interval)
}

// SweepCartReminder **جولةٌ واحدة** — للفحص.
func (s *Server) SweepCartReminder(ctx context.Context) int { return s.cartReminder.Sweep(ctx) }

// CartReminder **الخدمة** — للفحص.
func (s *Server) CartReminder() *cartreminder.Service { return s.cartReminder }
