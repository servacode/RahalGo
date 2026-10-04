package server

// ══════════════════════════════════════════════════════════════════════
// **قسمُ الحسابات — أفعالُ الملفّ** (قراراتُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
//	تغييرُ الرقم            بخطوةِ تحقّق · يُخرج الجلسات · يُبلَّغ القديم · وفوق حدٍّ ينتظر ثانياً
//	بعد تبديل الحال         مندوبٌ عاد فعّالاً تُصرف عمولتُه المحجوزة · ومحظورٌ نهائيّاً
//	                        تنتقل متاجرُه إلى المنصّة وتسقط محجوزاتُه
//	الملاحظاتُ الداخليّة     سجلٌّ لا يُمحى بكاتبه وتاريخه
//	منعُ النقد              يُرى بسببه وانتهائه · ويرفعه مديرُ المنصّة بسببٍ مكتوب
//	السائق                  «السائقُ بخير» بعد حادث · المركبة · سقفُ نقدٍ خاصّ

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

var (
	errPhonePending = httpx.NewError(http.StatusConflict,
		"phone_change_pending", "errors.phone_change_pending")
	errSecondPerson = httpx.NewError(http.StatusForbidden,
		"second_person_required", "errors.second_person_required")
	// errAccidentCheck **لا دوامَ بعد حادثٍ قبل «السائقُ بخير».**
	errAccidentCheck = httpx.NewError(http.StatusConflict,
		"accident_check_required", "errors.accident_check_required")
)

// ── تغييرُ الرقم ─────────────────────────────────────────────────────

// changePhone **يبدّل الرقمَ أو يضعه في انتظار شخصٍ ثانٍ** — بحسب رصيد الحساب.
func (s *Server) changePhone(r *http.Request, userID, rawPhone string) (map[string]any, error) {
	ctx := r.Context()
	phone, ok := identity.NormalizePhone(rawPhone)
	if !ok {
		return nil, identity.ErrInvalidPhone
	}
	var old string
	var balance int64
	if err := s.pg.QueryRow(ctx, `
		SELECT u.phone, COALESCE(w.balance, 0)
		FROM users u LEFT JOIN wallets w ON w.user_id = u.id WHERE u.id = $1`, userID).
		Scan(&old, &balance); err != nil {
		return nil, httpx.ErrNotFound
	}
	if old == phone {
		return map[string]any{"pending": false, "changed": false}, nil
	}
	threshold := s.settings.GetNum(ctx, "security.phone_change_approval_balance", 100000)
	if threshold > 0 && balance > threshold {
		var id string
		err := s.pg.QueryRow(ctx, `
			INSERT INTO phone_change_requests (user_id, old_phone, new_phone, balance, proposed_by)
			VALUES ($1, $2, $3, $4, $5) RETURNING id::text`,
			userID, old, phone, balance, userIDFrom(r)).Scan(&id)
		if isUniqueViolation(err) {
			return nil, errPhonePending
		}
		if err != nil {
			return nil, err
		}
		s.audit(r, "admin.phone_change_requested", "user", userID, map[string]any{
			"request_id": id, "old_phone": old, "new_phone": phone, "balance": balance})
		s.notify.NotifyOps(ctx, notifications.Input{
			Kind: notifications.KindAccount, Title: notifTitles.phoneChangeRequest,
			Entity: "user", EntityID: userID, Href: "/dashboard/users/" + userID,
		})
		return map[string]any{"pending": true, "request_id": id}, nil
	}
	return s.applyPhoneChange(ctx, userIDFrom(r), userID, phone, clientIP(r), "")
}

// applyPhoneChange يبدّل ويُبلغ الرقمَ القديم.
func (s *Server) applyPhoneChange(ctx context.Context, actor, userID, phone, ip, approvedBy string) (map[string]any, error) {
	old, _, err := s.identity.AdminChangePhone(ctx, actor, userID, phone, ip, approvedBy)
	if err != nil {
		return nil, err
	}
	// **والرقمُ القديمُ يُبلَّغ** — إن لم يطلب صاحبُه ذلك عرف من أين يسأل.
	sent := s.sendText(ctx, old, "رحّال غو — نُقل حسابك إلى رقم آخر بطلب من الإدارة. "+
		"إن لم تطلب ذلك فاتصل بالدعم فوراً.")
	s.touchUser(userID, "account")
	s.touch("account", "ops")
	return map[string]any{"pending": false, "changed": true, "old_notified": sent}, nil
}

type phoneRequestRow struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	OldPhone   string    `json:"old_phone"`
	NewPhone   string    `json:"new_phone"`
	Balance    int64     `json:"balance"`
	Status     string    `json:"status"`
	ProposedBy string    `json:"proposed_by"`
	Proposer   string    `json:"proposer_name"`
	CreatedAt  time.Time `json:"created_at"`
}

// handleListPhoneRequests طلباتُ تغيير الرقم المعلّقة — كلُّها أو لحسابٍ بعينه.
func (s *Server) handleListPhoneRequests(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user_id")
	if user != "" && !isUUID(user) {
		s.respondErr(w, errValidation)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT pr.id::text, pr.user_id::text, pr.old_phone, pr.new_phone, pr.balance, pr.status,
		       pr.proposed_by::text, COALESCE(NULLIF(p.full_name, ''), p.phone, ''), pr.created_at
		FROM phone_change_requests pr JOIN users p ON p.id = pr.proposed_by
		WHERE pr.status = 'pending' AND ($1 = '' OR pr.user_id::text = $1)
		ORDER BY pr.created_at DESC LIMIT 100`, user)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []phoneRequestRow{}
	for rows.Next() {
		var p phoneRequestRow
		if err := rows.Scan(&p.ID, &p.UserID, &p.OldPhone, &p.NewPhone, &p.Balance, &p.Status,
			&p.ProposedBy, &p.Proposer, &p.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, p)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"requests": out})
}

// handleDecidePhoneRequest **موافقةُ شخصٍ ثانٍ** — لا يوافق صاحبُ الطلب على طلبه.
func (s *Server) handleDecidePhoneRequest(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if !isUUID(id) {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		actor := userIDFrom(r)
		var p phoneRequestRow
		err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
			if err := q.QueryRow(ctx, `
				SELECT id::text, user_id::text, old_phone, new_phone, balance, status, proposed_by::text
				FROM phone_change_requests WHERE id = $1 FOR UPDATE`, id).
				Scan(&p.ID, &p.UserID, &p.OldPhone, &p.NewPhone, &p.Balance, &p.Status, &p.ProposedBy); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return httpx.ErrNotFound
				}
				return err
			}
			if p.Status != "pending" {
				return errRequestDecided
			}
			if approve && p.ProposedBy == actor {
				return errSecondPerson
			}
			status := "rejected"
			if approve {
				status = "approved"
			}
			_, err := q.Exec(ctx, `UPDATE phone_change_requests
				SET status = $2, decided_by = $3, decided_at = now() WHERE id = $1`, id, status, actor)
			return err
		})
		if err != nil {
			s.respondErr(w, err)
			return
		}
		if !approve {
			s.audit(r, "admin.phone_change_rejected", "user", p.UserID,
				map[string]any{"request_id": id, "new_phone": p.NewPhone})
			httpx.JSON(w, http.StatusOK, map[string]any{"status": "rejected"})
			return
		}
		res, err := s.applyPhoneChange(r.Context(), actor, p.UserID, p.NewPhone, clientIP(r), actor)
		if err != nil {
			// **والطلبُ يعود معلَّقاً إن سقط التبديل** — الرقمُ أُخذ مثلاً.
			_, _ = s.pg.Exec(r.Context(), `UPDATE phone_change_requests
				SET status = 'pending', decided_by = NULL, decided_at = NULL WHERE id = $1`, id)
			s.respondErr(w, err)
			return
		}
		res["status"] = "approved"
		httpx.JSON(w, http.StatusOK, res)
	}
}

// ── بعد تبديل الحال ──────────────────────────────────────────────────

// afterStatusChange **أثرُ الحال على المال والمتاجر** — بعد التثبيت.
//
//   - **فعّالٌ**: تُصرف عمولاتُه المحجوزة (قرار ١٩).
//   - **محظورٌ نهائيّاً**: تنتقل متاجرُه إلى المنصّة وتسقط محجوزاتُه (قرار ٢١).
func (s *Server) afterStatusChange(r *http.Request, userID, status string) {
	ctx := r.Context()
	actor := userIDFrom(r)
	switch status {
	case "active":
		var n int
		var total int64
		err := s.inTx(ctx, func(ctx context.Context, q dbtx.Querier) error {
			var err error
			n, total, err = s.orders.ReleaseHeldCommissionsTx(ctx, q, userID, actor)
			if err != nil || n == 0 {
				return err
			}
			return s.auditTx(ctx, q, r, "finance.rep_commission_released", "user", userID,
				map[string]any{"orders": n, "amount": total})
		})
		if err != nil {
			s.logger.Error("تعذّر صرفُ العمولات المحجوزة", "user", userID, "error", err)
			return
		}
		if n > 0 {
			s.notify.Notify(ctx, notifications.Input{
				UserID: userID, Kind: notifications.KindWallet,
				Title: notifTitles.commissionEarned, Entity: "wallet", Href: "/portal/wallet",
			})
			s.touchUser(userID, "wallet")
		}
	case "blocked":
		err := s.inTx(ctx, func(ctx context.Context, q dbtx.Querier) error {
			rows, err := q.Query(ctx, `
				UPDATE merchants SET sales_rep_user_id = NULL, updated_at = now()
				 WHERE sales_rep_user_id = $1 RETURNING id::text`, userID)
			if err != nil {
				return err
			}
			var moved []string
			for rows.Next() {
				var id string
				if rows.Scan(&id) == nil {
					moved = append(moved, id)
				}
			}
			rows.Close()
			for _, m := range moved {
				if err := s.auditTx(ctx, q, r, "admin.merchant_rep_transfer", "merchant", m,
					map[string]any{"from_rep": userID, "to_rep": nil,
						"reason": "حُظر المندوب نهائياً — انتقل المتجر إلى المنصة"}); err != nil {
					return err
				}
			}
			forfeited, err := s.orders.ForfeitHeldCommissionsTx(ctx, q, userID, actor)
			if err != nil {
				return err
			}
			if len(moved) > 0 || forfeited > 0 {
				return s.auditTx(ctx, q, r, "admin.rep_banned_cleanup", "user", userID,
					map[string]any{"stores_moved": len(moved), "held_forfeited": forfeited})
			}
			return nil
		})
		if err != nil {
			s.logger.Error("تعذّر نقلُ متاجر المندوب المحظور", "user", userID, "error", err)
		}
		s.touch("merchant", "ops")
	}
}

// ── الملاحظاتُ الداخليّة ─────────────────────────────────────────────

type userNote struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	Author    *string   `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

// handleUserNotes سجلُّ الملاحظات — الأحدثُ أوّلاً.
func (s *Server) handleUserNotes(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT n.id, n.body, NULLIF(COALESCE(NULLIF(a.full_name, ''), a.phone, ''), ''), n.created_at
		FROM user_notes n LEFT JOIN users a ON a.id = n.author_id
		WHERE n.user_id = $1 ORDER BY n.id DESC LIMIT 200`, chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []userNote{}
	for rows.Next() {
		var n userNote
		if err := rows.Scan(&n.ID, &n.Body, &n.Author, &n.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, n)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"notes": out})
}

// handleAddUserNote يضيف سطراً — **ولا تعديلَ ولا حذف.**
func (s *Server) handleAddUserNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	req, err := decode[struct {
		Body string `json:"body"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	body := strings.TrimSpace(req.Body)
	if body == "" || !isUUID(id) {
		s.respondErr(w, errValidation)
		return
	}
	var n userNote
	err = s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		if err := q.QueryRow(ctx, `
			INSERT INTO user_notes (user_id, author_id, body)
			SELECT $1, $2, $3 WHERE EXISTS (SELECT 1 FROM users WHERE id = $1)
			RETURNING id, body, created_at`, id, userIDFrom(r), clip(body, 2000)).
			Scan(&n.ID, &n.Body, &n.CreatedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.ErrNotFound
			}
			return err
		}
		return s.auditTx(ctx, q, r, "admin.user_note_added", "user", id,
			map[string]any{"note_id": n.ID})
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, n)
}

// ── منعُ النقد ───────────────────────────────────────────────────────

// handleUserCashBan حالُ منع النقد لزبون.
func (s *Server) handleUserCashBan(w http.ResponseWriter, r *http.Request) {
	info, err := s.orders.CashBan(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, info)
}

// handleLiftCashBan **يرفع منعَ النقد بسببٍ مكتوب** — لمديرِ المنصّة وحدَه.
func (s *Server) handleLiftCashBan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	req, err := decode[struct {
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		s.respondErr(w, errReasonRequired)
		return
	}
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	err = s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		tag, err := q.Exec(ctx, `UPDATE users SET cash_ban_lifted_at = now() WHERE id = $1`, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return httpx.ErrNotFound
		}
		if _, err := q.Exec(ctx, `INSERT INTO cash_ban_lifts (user_id, lifted_by, reason)
			VALUES ($1, $2, $3)`, id, userIDFrom(r), clip(reason, 500)); err != nil {
			return err
		}
		return s.auditTx(ctx, q, r, "admin.cash_ban_lifted", "user", id,
			map[string]any{"reason": reason})
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"lifted": true})
}

// ── السائق ───────────────────────────────────────────────────────────

// handleDriverOK **«السائقُ بخير»** — يرفع قفلَ ما بعد الحادث باسم من أكّد.
func (s *Server) handleDriverOK(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	req, _ := decode[struct {
		Note string `json:"note"`
	}](r)
	note := ""
	if req != nil {
		note = strings.TrimSpace(req.Note)
	}
	err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		tag, err := q.Exec(ctx, `
			UPDATE users SET accident_cleared_at = now(), accident_cleared_by = $2
			 WHERE id = $1 AND accident_lock_at IS NOT NULL
			   AND (accident_cleared_at IS NULL OR accident_cleared_at < accident_lock_at)`,
			id, userIDFrom(r))
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errRequestDecided
		}
		return s.auditTx(ctx, q, r, "ops.driver_cleared_after_accident", "user", id,
			map[string]any{"note": note})
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: id, Kind: notifications.KindAccount, Title: notifTitles.driverCleared,
		Entity: "user", EntityID: id,
	})
	s.touchUser(id, "account")
	s.touch("driver", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"cleared": true})
}

// driverAccidentLocked **أمقفولٌ بعد حادثٍ لم يؤكَّد بعدُ أنّه بخير؟**
func (s *Server) driverAccidentLocked(ctx context.Context, driverID string) bool {
	var locked bool
	_ = s.pg.QueryRow(ctx, `
		SELECT accident_lock_at IS NOT NULL
		   AND (accident_cleared_at IS NULL OR accident_cleared_at < accident_lock_at)
		FROM users WHERE id = $1`, driverID).Scan(&locked)
	return locked
}

// lockDriverAfterAccident يقفل دوامَ السائق بعد حادث — حتّى تؤكّد العمليّات أنّه بخير.
func (s *Server) lockDriverAfterAccident(ctx context.Context, driverID string) {
	if _, err := s.pg.Exec(ctx, `
		UPDATE users SET accident_lock_at = now(), on_shift = false WHERE id = $1`, driverID); err != nil {
		s.logger.Error("تعذّر قفلُ السائق بعد الحادث", "driver", driverID, "error", err)
	}
}

// handleDriverVehicle معلوماتُ المركبة — اختياريّةٌ كلُّها.
func (s *Server) handleDriverVehicle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	req, err := decode[struct {
		Type  string `json:"vehicle_type"`
		Plate string `json:"vehicle_plate"`
		Color string `json:"vehicle_color"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	err = s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		tag, err := q.Exec(ctx, `UPDATE users SET vehicle_type = $2, vehicle_plate = $3,
			vehicle_color = $4, updated_at = now() WHERE id = $1`, id,
			clip(strings.TrimSpace(req.Type), 40), clip(strings.TrimSpace(req.Plate), 20),
			clip(strings.TrimSpace(req.Color), 30))
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return httpx.ErrNotFound
		}
		return s.auditTx(ctx, q, r, "admin.driver_vehicle", "user", id, map[string]any{
			"vehicle_type": req.Type, "vehicle_plate": req.Plate, "vehicle_color": req.Color})
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"saved": true})
}

// handleDriverCashLimit **سقفُ نقدٍ خاصٌّ بسائق** — وفارغُه يعيده إلى السقف العامّ.
func (s *Server) handleDriverCashLimit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	req, err := decode[struct {
		Limit *int64 `json:"limit"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Limit != nil && *req.Limit < 0 {
		s.respondErr(w, errValidation)
		return
	}
	general := s.cashbox.Limit(r.Context())
	err = s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		tag, err := q.Exec(ctx, `UPDATE users SET cash_limit_override = $2, updated_at = now()
			WHERE id = $1`, id, req.Limit)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return httpx.ErrNotFound
		}
		return s.auditTx(ctx, q, r, "admin.driver_cash_limit", "user", id,
			map[string]any{"limit": req.Limit, "general": general})
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("driver", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"limit": req.Limit})
}
