package server

// ══════════════════════════════════════════════════════════════════════
// **قسمُ «العروض والخصومات»** — قراراتُ المالك ٢٠٢٦-١٠-٠٤
// ══════════════════════════════════════════════════════════════════════
//
//   - ملخّصٌ أعلى الصفحة: أكوادٌ سارية · خصوماتٌ سارية · كلفةُ هذا الشهر على
//     المنصّة · كلفتُه على المتاجر (`/promos/summary`).
//   - موافقاتُ الماليّة على ما فوق حدّ المحتوى (`/promo-approvals`) — على
//     عقد الموافقات الموحّد، ويقرؤها لوحُ الموافقات في الخزينة.
//   - عددُ من يصله إشعارُ العرض قبل «انشر» (`/offers/audience`).
//   - «ادعُ صديقاً»: المبالغُ والشرطُ وجدولُ مين دعا مين (`/referrals`).

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/approval"
	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/offers"
)

// monthStartDamascus **أوّلُ الشهر بتوقيت دمشق** — كالحوافز.
const monthStartDamascus = `(date_trunc('month', now() AT TIME ZONE 'Asia/Damascus') AT TIME ZONE 'Asia/Damascus')`

// promoSummary **ملخّصُ أعلى الصفحة.**
type promoSummary struct {
	ActiveCodes      int64 `json:"active_codes"`
	ActiveDiscounts  int64 `json:"active_discounts"`
	PendingApprovals int64 `json:"pending_approvals"`
	// PlatformCost = Codes + FreeDelivery + PlatformItems + Referrals — هذا الشهر.
	PlatformCost  int64 `json:"platform_cost"`
	Codes         int64 `json:"codes"`
	FreeDelivery  int64 `json:"free_delivery"`
	PlatformItems int64 `json:"platform_items"`
	Referrals     int64 `json:"referrals"`
	// StoresCost **خصوماتٌ تحمّلتها المتاجر** هذا الشهر.
	StoresCost int64 `json:"stores_cost"`
}

func (s *Server) promoSummary(ctx context.Context) (promoSummary, error) {
	var p promoSummary
	err := s.pg.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM promo_codes p WHERE `+catalog.PromoLiveSQL+`),
		  (SELECT count(*) FROM offers o WHERE `+offers.LiveCond+`),
		  (SELECT count(*) FROM promo_approvals WHERE status = 'pending'),
		  (SELECT COALESCE(sum(discount), 0) FROM orders
		    WHERE status = 'delivered' AND delivered_at >= `+monthStartDamascus+`),
		  (SELECT COALESCE(sum(promo_delivery_waived), 0) FROM orders
		    WHERE status = 'delivered' AND delivered_at >= `+monthStartDamascus+`),
		  (SELECT COALESCE(sum(oi.offer_cut * oi.qty) FILTER (WHERE oi.offer_borne_by = 'platform'), 0)
		     FROM order_items oi JOIN orders o ON o.id = oi.order_id
		    WHERE o.status = 'delivered' AND o.delivered_at >= `+monthStartDamascus+`),
		  (SELECT COALESCE(sum(oi.offer_cut * oi.qty) FILTER (WHERE oi.offer_borne_by = 'merchant'), 0)
		     FROM order_items oi JOIN orders o ON o.id = oi.order_id
		    WHERE o.status = 'delivered' AND o.delivered_at >= `+monthStartDamascus+`),
		  (SELECT COALESCE(sum(reward_amount), 0) FROM referrals
		    WHERE rewarded_at >= `+monthStartDamascus+`)`).
		Scan(&p.ActiveCodes, &p.ActiveDiscounts, &p.PendingApprovals, &p.Codes,
			&p.FreeDelivery, &p.PlatformItems, &p.StoresCost, &p.Referrals)
	p.PlatformCost = p.Codes + p.FreeDelivery + p.PlatformItems + p.Referrals
	return p, err
}

// handlePromoSummary `GET /admin/promos/summary`.
func (s *Server) handlePromoSummary(w http.ResponseWriter, r *http.Request) {
	p, err := s.promoSummary(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

// ── موافقاتُ الماليّة ─────────────────────────────────────────────────

type promoApprovalRow struct {
	ID             string     `json:"id"`
	Status         string     `json:"status"`
	Amount         int64      `json:"amount"`
	Note           string     `json:"note"`
	TargetKind     string     `json:"target_kind"`
	TargetID       string     `json:"target_id"`
	ProposedBy     string     `json:"proposed_by"`
	ProposedByName string     `json:"proposed_by_name"`
	DecidedBy      *string    `json:"decided_by"`
	DecidedByName  string     `json:"decided_by_name"`
	DecisionNote   string     `json:"decision_note"`
	SelfApproved   bool       `json:"self_approved"`
	CreatedAt      time.Time  `json:"created_at"`
	DecidedAt      *time.Time `json:"decided_at"`
}

const promoApprovalCols = `
	SELECT pa.id::text, pa.status, pa.amount, pa.note, pa.target_kind, pa.target_id::text,
	       pa.proposed_by::text, COALESCE(pu.full_name, ''),
	       pa.decided_by::text, COALESCE(du.full_name, ''), pa.decision_note, pa.self_approved,
	       pa.created_at, pa.decided_at
	FROM promo_approvals pa
	LEFT JOIN users pu ON pu.id = pa.proposed_by
	LEFT JOIN users du ON du.id = pa.decided_by`

func scanPromoApproval(row pgx.Row) (promoApprovalRow, error) {
	var a promoApprovalRow
	err := row.Scan(&a.ID, &a.Status, &a.Amount, &a.Note, &a.TargetKind, &a.TargetID,
		&a.ProposedBy, &a.ProposedByName, &a.DecidedBy, &a.DecidedByName, &a.DecisionNote,
		&a.SelfApproved, &a.CreatedAt, &a.DecidedAt)
	return a, err
}

// handleListPromoApprovals `GET /admin/promo-approvals?status=pending|approved|rejected|all`.
func (s *Server) handleListPromoApprovals(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	if status == "all" {
		status = ""
	}
	rows, err := s.pg.Query(r.Context(), promoApprovalCols+`
		WHERE ($1 = '' OR pa.status = $1)
		ORDER BY pa.created_at DESC LIMIT 200`, status)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []promoApprovalRow{}
	for rows.Next() {
		a, err := scanPromoApproval(rows)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"approvals": out})
}

// handleDecidePromoApproval `POST /admin/promo-approvals/{id}/approve|reject`.
//
// **والمقترحُ غيرُ الموافق** (`approval.Check`) — من عمل الكودَ لا يوافق عليه.
func (s *Server) handleDecidePromoApproval(approve bool) http.HandlerFunc {
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
		var out promoApprovalRow
		if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
			a, err := scanPromoApproval(q.QueryRow(ctx, promoApprovalCols+`
				WHERE pa.id = $1 FOR UPDATE OF pa`, id))
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.ErrNotFound
			}
			if err != nil {
				return err
			}
			if a.Status != "pending" {
				return errRequestDecided
			}
			var verdict approval.Verdict
			if approve {
				if verdict, err = approval.Check(ctx, q, approval.Request{
					ProposedBy: a.ProposedBy, Actor: actor, Capability: authz.FinanceManage,
				}); err != nil {
					return err
				}
			}
			switch a.TargetKind {
			case "promo":
				state := offers.ApprovalRejected
				if approve {
					state = offers.ApprovalOK
				}
				if _, err := q.Exec(ctx, `
					UPDATE promo_codes SET approval_state = $2, active = $3 WHERE id = $1::uuid`,
					a.TargetID, state, approve); err != nil {
					return err
				}
			case "offer":
				if err := s.offers.Decide(ctx, q, a.TargetID, approve); err != nil {
					return err
				}
			}
			status := "rejected"
			if approve {
				status = "approved"
			}
			if _, err := q.Exec(ctx, `
				UPDATE promo_approvals
				   SET status = $2, decided_by = $3::uuid, decided_at = now(),
				       decision_note = $4, self_approved = $5
				 WHERE id = $1::uuid`, id, status, actor,
				clip(strings.TrimSpace(req.Note), 500), verdict.SelfApproved); err != nil {
				return err
			}
			a.Status = status
			a.SelfApproved = verdict.SelfApproved
			out = a
			details := map[string]any{"approval_id": id, "target_kind": a.TargetKind,
				"amount": a.Amount, "proposed_by": a.ProposedBy, "note": strings.TrimSpace(req.Note)}
			for k, v := range verdict.AuditFields() {
				details[k] = v
			}
			action := "finance.promo_rejected"
			if approve {
				action = "finance.promo_approved"
			}
			return s.auditTx(ctx, q, r, action, a.TargetKind, a.TargetID, details)
		}); err != nil {
			s.respondErr(w, err)
			return
		}
		s.touch("offer", "ops")
		httpx.JSON(w, http.StatusOK, map[string]any{"status": out.Status,
			"self_approved": out.SelfApproved})
	}
}

// ── جمهورُ إشعار العرض ────────────────────────────────────────────────

// handleOfferAudience `GET /admin/offers/audience?menu_item_id=` — عددُ من
// يصله الإشعار (زبائنُ منطقة المتجر).
func (s *Server) handleOfferAudience(w http.ResponseWriter, r *http.Request) {
	item := r.URL.Query().Get("menu_item_id")
	if !isUUID(item) {
		s.respondErr(w, errValidation)
		return
	}
	n, err := s.offers.AudienceCount(r.Context(), item)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"count": n})
}

// notifyOfferArea **يبلّغ زبائنَ منطقة المتجر** — ويردّ عددَهم.
func (s *Server) notifyOfferArea(ctx context.Context, o *offers.Offer) int {
	if o == nil || o.MenuItemID == nil || !o.Live {
		return 0
	}
	ids, err := s.offers.Audience(ctx, *o.MenuItemID)
	if err != nil || len(ids) == 0 {
		return 0
	}
	s.notify.NotifyMany(ctx, ids, notifications.Input{
		Kind:     "offer",
		Title:    o.Title,
		Body:     offerBody(o),
		Entity:   "offer",
		EntityID: o.ID,
		Href:     offerHref,
	})
	return len(ids)
}

// ── «ادعُ صديقاً» ─────────────────────────────────────────────────────

type referralRow struct {
	InviterName  string     `json:"inviter_name"`
	InviteeName  string     `json:"invitee_name"`
	Rank         int        `json:"rank"`
	CreatedAt    time.Time  `json:"created_at"`
	RewardedAt   *time.Time `json:"rewarded_at"`
	RewardAmount int64      `json:"reward_amount"`
}

// handleAdminReferrals `GET /admin/referrals?page=` — المبالغُ والشرطُ (من
// الإعدادات، تُعدَّل هناك) وجدولُ الدعوات.
func (s *Server) handleAdminReferrals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	const per = 25
	rows, err := s.pg.Query(ctx, `
		SELECT COALESCE(iu.full_name, ''), COALESCE(eu.full_name, ''), r.rank,
		       r.created_at, r.rewarded_at, r.reward_amount
		FROM referrals r
		LEFT JOIN users iu ON iu.id = r.inviter_id
		LEFT JOIN users eu ON eu.id = r.invitee_id
		ORDER BY r.created_at DESC
		LIMIT $1 OFFSET $2`, per, (page-1)*per)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	list := []referralRow{}
	for rows.Next() {
		var x referralRow
		if err := rows.Scan(&x.InviterName, &x.InviteeName, &x.Rank, &x.CreatedAt,
			&x.RewardedAt, &x.RewardAmount); err != nil {
			s.respondErr(w, err)
			return
		}
		list = append(list, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	var total, rewarded, paid int64
	if err := s.pg.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE rewarded_at IS NOT NULL),
		       COALESCE(sum(reward_amount), 0)
		FROM referrals`).Scan(&total, &rewarded, &paid); err != nil {
		s.respondErr(w, err)
		return
	}
	on := s.settings.GetString(ctx, "referral.reward_on")
	if on == "" {
		on = "signup"
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"settings": map[string]any{
			"reward_on":   on,
			"reward_1":    s.settings.GetInt(ctx, "referral.reward_1"),
			"reward_2":    s.settings.GetInt(ctx, "referral.reward_2"),
			"reward_3":    s.settings.GetInt(ctx, "referral.reward_3"),
			"reward_4":    s.settings.GetInt(ctx, "referral.reward_4"),
			"reward_rest": s.settings.GetInt(ctx, "referral.reward_rest"),
		},
		"total": total, "rewarded": rewarded, "paid": paid,
		"page": page, "per_page": per,
		"referrals": list,
	})
}
