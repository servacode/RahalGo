package server

// ══════════════════════════════════════════════════════════════════════
// **«جوائز رحّال غو»** — أبوابُ الزبون واللوحة (قرارُ المالك ٢٠٢٦-١١-٢٣)
// ══════════════════════════════════════════════════════════════════════
//
// المنطقُ كلُّه في `internal/prizes` — وهنا قراءةٌ وردّ. **والطلبُ من متجر الجوائز يمرّ
// بباب الطلب العاديّ نفسِه**، والخادمُ يعرفه من أصنافه (`orders/prize_order.go`).

import (
	"context"
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/prizes"
)

// handlePrizesMe `GET /prizes/me` — دعواتي هذا الشهر وفرصي وموعدُ السحب ورصيدي.
func (s *Server) handlePrizesMe(w http.ResponseWriter, r *http.Request) {
	me, err := s.prizes.MeFor(r.Context(), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, me)
}

// handlePrizeItems `GET /prizes/items` — أصنافُ متجر الجوائز مرتّبةً بالسعر.
func (s *Server) handlePrizeItems(w http.ResponseWriter, r *http.Request) {
	store, items, err := s.prizes.Items(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"store_id": store, "items": items})
}

// handlePrizeWinners `GET /prizes/winners` — رابحو آخر سحب، بالاسم الأوّل وحرفٍ ولا رقم.
func (s *Server) handlePrizeWinners(w http.ResponseWriter, r *http.Request) {
	month, winners, err := s.prizes.LastWinners(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"month": month, "winners": winners})
}

// handleAdminPrizeEntries `GET /admin/prizes/entries?month=YYYY-MM` — مشتركو الشهر (افتراضاً الحاليّ).
func (s *Server) handleAdminPrizeEntries(w http.ResponseWriter, r *http.Request) {
	month := prizes.MonthStart(time.Now())
	if raw := r.URL.Query().Get("month"); raw != "" {
		m, ok := prizes.ParseMonth(raw)
		if !ok {
			s.respondErr(w, errValidation)
			return
		}
		month = m
	}
	list, err := s.prizes.AdminEntries(r.Context(), month)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"month":             prizes.MonthKey(month),
		"invites_per_entry": s.prizes.PerEntry(r.Context()),
		"entrants":          list,
	})
}

// handleAdminPrizeDraws `GET /admin/prizes/draws` — السحوباتُ السابقة بكلفتها.
func (s *Server) handleAdminPrizeDraws(w http.ResponseWriter, r *http.Request) {
	draws, err := s.prizes.AdminDraws(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"draws": draws})
}

// RunPrizes **نبضةُ الجوائز** — انتهاءٌ ثمّ سحبٌ ثمّ تذكير، ولا شيءَ يتكرّر.
func (s *Server) RunPrizes(ctx context.Context, interval time.Duration) {
	s.prizes.Sweep(ctx)
	s.prizes.Run(ctx, interval)
}

// SweepPrizes **جولةٌ واحدة** — للفحص.
func (s *Server) SweepPrizes(ctx context.Context) { s.prizes.Sweep(ctx) }

// Prizes **خدمةُ الجوائز** — للفحص.
func (s *Server) Prizes() *prizes.Service { return s.prizes }
