package server

// نقاطُ الأهداف والمكافآت — **بابان: من يُعطي ومن يُعطى.**
//
// # ولماذا دورٌ في المسار لا في الجسد
//
// «سائقون» و«مندوبون» شاشتان مختلفتان عند الإدارة، **والمقياسُ يختلف بالدور**:
// السائقُ بما وصّل والمندوبُ بما بيع من متاجرَ جلبها. **ومسارٌ يقرأ الدورَ من
// الجسد يجعل الشاشةَ تختار ما تُظهر** — وشاشةٌ تختار تُخطئ يوماً.

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/incentives"
)

// handleIncentiveStandings حالُ كلّ من في دورٍ هذا الشهر.
func (s *Server) handleIncentiveStandings(w http.ResponseWriter, r *http.Request) {
	role := chi.URLParam(r, "role")
	if role != "driver" && role != "sales" {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	rows, err := s.incentives.Standings(r.Context(), role)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"standings": rows})
}

// handleIncentiveList كشفُ مكافآتِ شخصٍ وعقوباته — للإدارة.
func (s *Server) handleIncentiveList(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rows, err := s.incentives.List(r.Context(), chi.URLParam(r, "id"), limit)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"entries": rows})
}

// handleIncentiveGrant يرسل مكافأةً أو يوقّع عقوبة.
//
// **وللمالية والإدارة وحدَهما** — كتعويض السائق: **مالٌ يخرج بتقدير إنسان،
// وموظّفُ العمليات ليس طرفاً في المال.**
func (s *Server) handleIncentiveGrant(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Kind      string `json:"kind"`
		Amount    int64  `json:"amount"`
		Reason    string `json:"reason"`
		ForTarget bool   `json:"for_target"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	e, err := s.incentives.Grant(r.Context(), userIDFrom(r), chi.URLParam(r, "id"),
		req.Kind, req.Amount, req.Reason, req.ForTarget)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "finance.incentive", "user", chi.URLParam(r, "id"), map[string]any{
		"kind": req.Kind, "amount": req.Amount, "reason": req.Reason,
	})
	// **ومن نال يعلم** — مكافأةٌ لا يراها صاحبُها مكافأةٌ لم تُصرف في نظره،
	// **وعقوبةٌ لا يعلم بها لا تُصلح شيئاً.**
	s.touchUser(chi.URLParam(r, "id"), "wallet")
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, e)
}

// handleMyIncentives هدفي وما نلتُ — للسائق والمندوب.
//
// **والدورُ من الحساب لا من الطلب**: من يسأل عن نفسه لا يُسأل عن دوره.
func (s *Server) handleMyIncentives(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	role := "driver"
	for _, x := range rolesFrom(r) {
		if x == "sales" {
			role = "sales"
		}
	}
	st, err := s.incentives.MyStanding(r.Context(), uid, role)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	list, err := s.incentives.List(r.Context(), uid, 50)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"standing": st,
		"entries":  list,
		"kinds":    []string{incentives.KindReward, incentives.KindPenalty},
	})
}
