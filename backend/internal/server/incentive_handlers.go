package server

// نقاطُ الأهداف والمكافآت — **من يُعطي ومن يُعطى.**
//
// # ولماذا دورٌ في المسار لا في الجسد
//
// «سائقون» و«مندوبون» شاشتان مختلفتان عند الإدارة، **والمقياسُ يختلف بالدور**:
// السائقُ بما وصّل والمندوبُ بما فتح من متاجر.
//
// ══════════════════════════════════════════════════════════════════════
// **قراراتُ المالك ٢٠٢٦-١٠-٠٤ (قسمُ الأهداف)**
// ══════════════════════════════════════════════════════════════════════
//
//	١ · اليدويُّ طلبٌ يوافق عليه شخصٌ ثانٍ — `incentive_requests` + `approval.Check`
//	٣ و٤ · تنبيهٌ والماليّةُ تقرّر — `POST /incentive-alerts/{id}/decide`
//	٥ · اختيارُ شهرٍ وتصدير — `?month=2026-09` و`/incentives/{role}/export`
//	٦ · اليدويُّ «تقدير» دائماً — `for_target` في الجسد يُتجاهَل

import (
	"context"
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/incentives"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

func incentiveRole(r *http.Request) (string, bool) {
	role := chi.URLParam(r, "role")
	return role, role == "driver" || role == "sales"
}

// handleIncentiveStandings حالُ كلّ من في دورٍ لشهرٍ — مع الكروت والمراحل والتنبيهات.
func (s *Server) handleIncentiveStandings(w http.ResponseWriter, r *http.Request) {
	role, ok := incentiveRole(r)
	if !ok {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	month, err := incentives.NormalizeMonth(r.URL.Query().Get("month"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	ctx := r.Context()
	rows, err := s.incentives.StandingsFor(ctx, role, month)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	sum, err := s.incentives.SummaryOf(ctx, role, month, rows)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	alerts, err := s.incentives.Alerts(ctx, role, marketTestMarker)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"month":     month,
		"standings": rows,
		"summary":   sum,
		"levels":    s.incentives.LevelsOf(ctx, role),
		"alerts":    alerts,
		"cap":       s.walletManualMax(ctx),
	})
}

// handleIncentiveExport **تصديرُ الشهر** — ملفُّ CSV بالأعمدة التي في الصفحة.
func (s *Server) handleIncentiveExport(w http.ResponseWriter, r *http.Request) {
	role, ok := incentiveRole(r)
	if !ok {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	month, err := incentives.NormalizeMonth(r.URL.Query().Get("month"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.incentives.StandingsFor(r.Context(), role, month)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		`attachment; filename="incentives-`+role+`-`+month+`.csv"`)
	// **وعلامةُ الترميز أوّلاً** — وإلّا فتح إكسلُ العربيّةَ رموزاً.
	_, _ = w.Write([]byte("\xEF\xBB\xBF"))
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"الاسم", "الهاتف", "الإنجاز", "المرحلة", "من", "مكافأة الهدف",
		"مكافأة يدوية", "عقوبات"})
	for _, x := range rows {
		_ = cw.Write([]string{x.Name, x.Phone, strconv.Itoa(x.Done), strconv.Itoa(x.Level),
			strconv.Itoa(x.Levels), strconv.FormatInt(x.AutoPaid, 10),
			strconv.FormatInt(x.ManualPaid, 10), strconv.FormatInt(x.Penalized, 10)})
	}
	cw.Flush()
	s.audit(r, "finance.incentive_export", "incentives", role, map[string]any{"month": month})
}

// handleIncentiveList كشفُ مكافآتِ شخصٍ وعقوباته — للّوحة الجانبيّة.
func (s *Server) handleIncentiveList(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rows, err := s.incentives.List(r.Context(), chi.URLParam(r, "id"), limit)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"entries": rows})
}

// handleIncentiveGrant **يقترح** مكافأةً أو عقوبة — ولا يمسّ المال.
//
// **كان يقيّد فوراً بيد شخصٍ واحد** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ١). والآن
// الردُّ ٢٠١ بطلبٍ معلَّق، والقيدُ عند موافقة غيره. **وسقفُ الحركة اليدويّة
// نفسُه** (`finance.manual_wallet_max`).
func (s *Server) handleIncentiveGrant(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Kind   string `json:"kind"`
		Amount int64  `json:"amount"`
		Reason string `json:"reason"`
		// ForTarget **يُتجاهَل** — اليدويُّ «تقدير» دائماً (البند ٦).
		ForTarget bool `json:"for_target"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	target := chi.URLParam(r, "id")
	if !isUUID(target) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if req.Amount > s.walletManualMax(r.Context()) {
		s.respondErr(w, errWalletOverCap)
		return
	}
	actor := userIDFrom(r)
	s.WithIdempotentTx(w, r, func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error) {
		id, err := s.incentives.Propose(ctx, q, incentives.Proposal{
			Actor: actor, UserID: target, Kind: req.Kind, Amount: req.Amount, Note: req.Reason,
		})
		if err != nil {
			return IdempotentBody{}, err
		}
		if err := s.auditTx(ctx, q, r, "finance.incentive_request", "user", target,
			map[string]any{"request_id": id, "kind": req.Kind, "amount": req.Amount,
				"reason": strings.TrimSpace(req.Reason)}); err != nil {
			return IdempotentBody{}, err
		}
		return IdempotentBody{
			Status:      http.StatusCreated,
			Payload:     map[string]any{"request_id": id, "status": "pending"},
			AfterCommit: func() { s.touch("incentive", "ops") },
		}, nil
	})
}

// handleListIncentiveRequests الطلبات — المعلَّقةُ افتراضاً، أو طلباتُ حسابٍ بعينه.
func (s *Server) handleListIncentiveRequests(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := q.Get("status")
	if status == "" {
		status = "pending"
	}
	user := q.Get("user_id")
	if user != "" && !isUUID(user) {
		s.respondErr(w, errValidation)
		return
	}
	rows, err := s.incentives.ListRequests(r.Context(), status, user)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"requests": rows})
}

// handleDecideIncentiveRequest يوافق على الطلب أو يرفضه.
//
// **والموافقةُ من غير صاحب الاقتراح** (`approval.Check`)، وتقيّد بطرفين،
// **وتُشعر صاحبَ المكافأة أو العقوبة بسببها** — كانت اليدويّةُ تحدّث المحفظةَ
// بلا رسالة، فيرى رقماً تغيّر ولا يعرف لماذا.
func (s *Server) handleDecideIncentiveRequest(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if !isUUID(id) {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		req, err := decode[struct {
			Note string `json:"note"`
		}](r)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		actor := userIDFrom(r)
		var out incentives.Request
		if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
			row, verdict, err := s.incentives.DecideRequest(ctx, q, id, actor, approve, req.Note)
			if err != nil {
				return err
			}
			out = row
			action := "finance.incentive_request_rejected"
			if approve {
				action = "finance.incentive"
			}
			details := map[string]any{"request_id": id, "kind": row.Kind, "amount": row.Amount,
				"reason": row.Note, "proposed_by": row.ProposedBy, "source": row.Source,
				"note": strings.TrimSpace(req.Note)}
			for k, v := range verdict.AuditFields() {
				details[k] = v
			}
			return s.auditTx(ctx, q, r, action, "user", row.UserID, details)
		}); err != nil {
			s.respondErr(w, err)
			return
		}
		if approve {
			s.notifyIncentive(r.Context(), out)
			s.touchUser(out.UserID, "wallet")
		}
		s.touch("incentive", "ops")
		httpx.JSON(w, http.StatusOK, map[string]any{"status": out.Status,
			"self_approved": out.SelfApproved})
	}
}

// notifyIncentive **صاحبُ المكافأة أو العقوبة يعلم بها وبسببها.**
func (s *Server) notifyIncentive(ctx context.Context, x incentives.Request) {
	title := notifTitles.incentiveReward
	if x.Kind == incentives.KindPenalty {
		title = notifTitles.incentivePenalty
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: x.UserID, Kind: notifications.KindWallet,
		Title:  title,
		Body:   strconv.FormatInt(x.Amount, 10) + " " + currencyWord + " — " + x.Note,
		Entity: "wallet", Href: "/portal/wallet",
		Apps: []string{notifications.AppDriver, notifications.AppRep},
	})
}

// handleDecideIncentiveAlert **قرارُ الماليّة على تنبيه** (البندان ٣ و٤):
// `keep` تبقى المكافأة · `clawback` طلبُ عقوبةٍ بمبلغها يوافق عليه شخصٌ ثانٍ.
func (s *Server) handleDecideIncentiveAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Decision string `json:"decision"`
		Note     string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	actor := userIDFrom(r)
	var reqID string
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		rid, err := s.incentives.DecideAlert(ctx, q, id, actor, req.Decision, req.Note)
		if err != nil {
			return err
		}
		reqID = rid
		return s.auditTx(ctx, q, r, "finance.incentive_alert", "incentive", id,
			map[string]any{"decision": req.Decision, "note": strings.TrimSpace(req.Note),
				"request_id": rid})
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("incentive", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"decided": true, "request_id": reqID})
}

// handleRetryIncentiveFailure **يعيد مكافأةَ هدفٍ تعثّرت** — لشهرها.
func (s *Server) handleRetryIncentiveFailure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	p, err := s.incentives.RetryFailure(r.Context(), id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "finance.incentive_retry", "user", p.UserID,
		map[string]any{"failure_id": id, "paid": p.Amount})
	s.notifyTargetRetried(r.Context(), p)
	s.touch("incentive", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"paid": p.Amount})
}

func (s *Server) notifyTargetRetried(ctx context.Context, p incentives.Paid) {
	if p.Amount <= 0 {
		return
	}
	app := notifications.AppDriver
	if p.Role == "sales" {
		app = notifications.AppRep
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: p.UserID, Kind: notifications.KindWallet,
		Title:  notifTitles.targetReached,
		Body:   strconv.FormatInt(p.Amount, 10) + " " + currencyWord,
		Entity: "wallet", Href: "/portal/wallet",
		Apps: []string{app},
	})
	s.touchUser(p.UserID, "wallet")
}

// RunIncentiveRetries **يعيد ما تعثّر من مكافآت الهدف دوريّاً.**
func (s *Server) RunIncentiveRetries(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, p := range s.incentives.RetryFailures(ctx) {
				s.notifyTargetRetried(ctx, p)
			}
		}
	}
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
	// **ومكافأةُ الهدف تُقال قبل أن يُبلَغ** (شكوى المالك ٢٠٢٦-٠٨-٠٩)،
	// **والمراحلُ تُرسَل كلُّها** (قرارُ المالك ٢٠٢٦-٠٨-٣١). **و`target_reward`
	// تبقى** — مكافأةُ الأولى لشاشاتٍ قديمةٍ لم تُحدَّث.
	key := "drivers.target_reward"
	if role == "sales" {
		key = "sales.target_reward"
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"standing":      st,
		"entries":       list,
		"target_reward": s.settings.GetInt(r.Context(), key),
		"levels":        s.incentives.LevelsOf(r.Context(), role),
		"kinds":         []string{incentives.KindReward, incentives.KindPenalty},
	})
}
