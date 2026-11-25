package server

// ══════════════════════════════════════════════════════════════════════
// **«جوائز رحّال غو»** — أبوابُ الزبون واللوحة (قرارا المالك ٢٠٢٦-١١-٢٣ و٢٠٢٦-١١-٢٥: مسابقةُ نقاط)
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

// handlePrizesMe `GET /prizes/me` — نقاطي وترتيبي وشروطي وموعدُ الحسم ورصيدي. **ولا شيءَ عن غيري.**
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

// handlePrizeCheckIn `POST /prizes/checkin` — «سجّل حضوري»، مرّةً باليوم. يردّ الحالَ بعده.
func (s *Server) handlePrizeCheckIn(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	if _, err := s.prizes.CheckIn(r.Context(), uid); err != nil {
		s.respondErr(w, err)
		return
	}
	s.handlePrizesMe(w, r)
}

// handlePrizeFollow `POST /prizes/follow` {network: facebook|instagram} — بالأمانة، مرّةً في العمر.
func (s *Server) handlePrizeFollow(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Network string `json:"network"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.prizes.Follow(r.Context(), userIDFrom(r), req.Network); err != nil {
		s.respondErr(w, err)
		return
	}
	s.handlePrizesMe(w, r)
}

// handleAdminPrizeLeaderboard `GET /admin/prizes/leaderboard?month=YYYY-MM` — ترتيبُ الشهر كاملاً
// بالأسماء والأرقام (افتراضاً المسابقةُ الجارية).
func (s *Server) handleAdminPrizeLeaderboard(w http.ResponseWriter, r *http.Request) {
	_, launch := s.prizes.LaunchOn(r.Context())
	month := prizes.CurrentMonth(time.Now(), launch)
	if raw := r.URL.Query().Get("month"); raw != "" {
		m, ok := prizes.ParseMonth(raw)
		if !ok {
			s.respondErr(w, errValidation)
			return
		}
		month = m
	}
	list, err := s.prizes.AdminLeaderboard(r.Context(), month)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"month":       prizes.MonthKey(month),
		"cutoff_at":   prizes.CutoffAt(month).UTC(),
		"announce_at": prizes.AnnounceAt(month).UTC(),
		"min_points":  s.prizes.MinPoints(r.Context()),
		"winners":     s.prizes.WinnersCount(r.Context()),
		"standings":   list,
	})
}

// handleAdminPrizeDraws `GET /admin/prizes/draws` — النتائجُ السابقة بكلفتها.
func (s *Server) handleAdminPrizeDraws(w http.ResponseWriter, r *http.Request) {
	draws, err := s.prizes.AdminDraws(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"draws": draws})
}

// RunPrizes **نبضةُ الجوائز** — انتهاءٌ ثمّ إعلانٌ ثمّ تذكير، ولا شيءَ يتكرّر.
func (s *Server) RunPrizes(ctx context.Context, interval time.Duration) {
	s.prizes.Sweep(ctx)
	s.prizes.Run(ctx, interval)
}

// SweepPrizes **جولةٌ واحدة** — للفحص.
func (s *Server) SweepPrizes(ctx context.Context) { s.prizes.Sweep(ctx) }

// Prizes **خدمةُ الجوائز** — للفحص.
func (s *Server) Prizes() *prizes.Service { return s.prizes }

// handleAdminPrizeStore `GET /admin/prizes/store` — متجرُ «رحّال غو» للجوائز وعددُ أصنافه (٢٠٢٦-١١-٢٤).
func (s *Server) handleAdminPrizeStore(w http.ResponseWriter, r *http.Request) {
	st, err := prizes.StoreForAdmin(r.Context(), s.pg)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"store": st})
}

// handleSetPrizeStore `PUT /admin/prizes/store` {merchant_id} — يختار متجرَ الجوائز، والفارغُ يرجّعه
// متجراً عاديّاً مُطفأً. **فيختفي من السوق فوراً وتطلع أصنافُه بصفحة الجوائز وحدَها.**
func (s *Server) handleSetPrizeStore(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		MerchantID string `json:"merchant_id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := prizes.SetStore(r.Context(), s.pg, req.MerchantID); err != nil {
		s.respondErr(w, err)
		return
	}
	st, err := prizes.StoreForAdmin(r.Context(), s.pg)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"store": st})
}
