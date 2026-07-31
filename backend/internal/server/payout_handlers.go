package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// طلبات سحب الرصيد — تُغلق دورة المال: يطلب صاحب الرصيد، وتصرف المالية بقيد
// في الدفتر نفسه. الخصم لحظة الصرف لا لحظة الطلب (لا نجمّد مال أحد بطلب).

var (
	errPayoutPending    = httpx.NewError(http.StatusConflict, "payout_pending", "errors.payout_pending")
	errPayoutOver       = httpx.NewError(http.StatusConflict, "insufficient_balance", "errors.insufficient_balance")
	errPayoutNotAllowed = httpx.NewError(http.StatusForbidden, "payout_not_allowed", "errors.payout_not_allowed")
	errPayoutClosed     = httpx.NewError(http.StatusConflict, "payout_closed", "errors.payout_closed")
)

type payout struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	UserName  string     `json:"user_name"`
	UserPhone string     `json:"user_phone"`
	Amount    int64      `json:"amount"`
	Status    string     `json:"status"`
	Note      string     `json:"note"`
	Decision  string     `json:"decision"`
	Balance   int64      `json:"balance"` // رصيد الطالب الآن — تحتاجه المالية للقرار
	CreatedAt time.Time  `json:"created_at"`
	DecidedAt *time.Time `json:"decided_at"`
}

const payoutSelect = `
	SELECT p.id, p.user_id, COALESCE(NULLIF(u.full_name,''), u.phone::text), u.phone,
	       p.amount, p.status, p.note, p.decision,
	       COALESCE((SELECT balance FROM wallets w WHERE w.user_id = p.user_id), 0),
	       p.created_at, p.decided_at
	FROM payout_requests p JOIN users u ON u.id = p.user_id`

func scanPayouts(rows interface {
	Next() bool
	Scan(...any) error
}) ([]payout, error) {
	out := []payout{}
	for rows.Next() {
		var p payout
		if err := rows.Scan(&p.ID, &p.UserID, &p.UserName, &p.UserPhone, &p.Amount,
			&p.Status, &p.Note, &p.Decision, &p.Balance, &p.CreatedAt, &p.DecidedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// handleMyPayouts طلبات السحب الخاصة بصاحب الحساب.
func (s *Server) handleMyPayouts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(),
		payoutSelect+` WHERE p.user_id = $1 ORDER BY p.created_at DESC LIMIT 50`, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out, err := scanPayouts(rows)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

// payoutRoles من يحقّ له سحب رصيده نقداً من المنصة.
//
// **الفرق ليس في المال بل في مصدره**: المندوب والسائق والمتجر يكسبون رصيدهم من
// المنصة (عمولة، أجر، ثمن بضاعة) فالسحب هو قبضُ ما استحقّوه. أمّا رصيد الزبون
// فمصدره شحنٌ سلّمه نقداً أو استرجاعُ طلب — وهو **رصيد إنفاق لا رصيد دخل**،
// وتحويله إلى نقدٍ يجعل المحفظة قناة صرافة لا وسيلة دفع.
//
// وكان الفحص غائباً كلياً: الواجهة تُخفي الزرّ عن الزبون، والإخفاء ليس قفلاً.
var payoutRoles = []string{"sales", "driver", "merchant"}

func mayRequestPayout(roles []string) bool {
	for _, r := range roles {
		for _, allowed := range payoutRoles {
			if r == allowed {
				return true
			}
		}
	}
	return false
}

// handleCreatePayout طلب سحب جديد — بحدود الرصيد الحالي وبطلب معلّق واحد.
func (s *Server) handleCreatePayout(w http.ResponseWriter, r *http.Request) {
	roles, _ := r.Context().Value(ctxRoles).([]string)
	if !mayRequestPayout(roles) {
		s.respondErr(w, errPayoutNotAllowed)
		return
	}
	req, err := decode[struct {
		Amount int64  `json:"amount"`
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	uid := userIDFrom(r)
	balance, err := s.wallet.Balance(r.Context(), uid)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Amount <= 0 || req.Amount > balance {
		s.respondErr(w, errPayoutOver)
		return
	}

	var id string
	err = s.pg.QueryRow(r.Context(), `
		INSERT INTO payout_requests (user_id, amount, note) VALUES ($1, $2, $3)
		RETURNING id`, uid, req.Amount, clip(req.Note, 300)).Scan(&id)
	if isUniqueViolation(err) {
		s.respondErr(w, errPayoutPending) // الفهرس الفريد يمنع طلبين معلّقين
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// المالية تعرف فوراً — الطلب بلا متابع يبقى معلّقاً بلا نهاية
	s.notify.NotifyRoles(r.Context(), []string{"admin", "finance"}, notifications.Input{
		Kind: notifications.KindWallet, Title: notifTitles.payoutRequested,
		Body: s.userLabel(r.Context(), uid), Entity: "payout", EntityID: id,
		Href: "/dashboard/payouts",
	})
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

// handleAdminPayouts كل طلبات السحب (ترشيح بالحالة) — للأدمن والمالية.
func (s *Server) handleAdminPayouts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), payoutSelect+`
		WHERE ($1 = '' OR p.status = $1)
		ORDER BY (p.status = 'pending') DESC, p.created_at DESC LIMIT 200`,
		r.URL.Query().Get("status"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out, err := scanPayouts(rows)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleDecidePayout صرف الطلب أو رفضه — أدمن/مالية حصراً.
// الصرف يقيّد `payout` في دفتر المحفظة: المال يخرج بأثر، لا بتعديل رصيد.
func (s *Server) handleDecidePayout(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Status   string `json:"status"` // paid | rejected
		Decision string `json:"decision"`
	}](r)
	if err != nil || (req.Status != "paid" && req.Status != "rejected") {
		s.respondErr(w, errValidation)
		return
	}
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}

	var userID string
	var amount int64
	var status string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT user_id, amount, status FROM payout_requests WHERE id = $1`, id).
		Scan(&userID, &amount, &status); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if status != "pending" {
		s.respondErr(w, errPayoutClosed)
		return
	}

	actor := userIDFrom(r)
	if req.Status == "paid" {
		// الخصم أولاً: إن لم يكفِ الرصيد يُرفض القرار ولا يُقفل الطلب
		if _, err := s.wallet.Apply(r.Context(), userID, -amount, "payout",
			id, clip(req.Decision, 300), &actor); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	if _, err := s.pg.Exec(r.Context(), `
		UPDATE payout_requests SET status = $2, decision = $3, decided_by = $4, decided_at = now()
		WHERE id = $1`, id, req.Status, clip(req.Decision, 300), actor); err != nil {
		s.respondErr(w, err)
		return
	}

	title := notifTitles.payoutPaid
	if req.Status == "rejected" {
		title = notifTitles.payoutRejected
	}
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: userID, Kind: notifications.KindWallet, Title: title,
		Body: req.Decision, Entity: "payout", EntityID: id, Href: "/portal/wallet",
	})
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

// userLabel اسم المستخدم أو هاتفه — لنصوص الإشعارات.
func (s *Server) userLabel(ctx context.Context, id string) string {
	var label string
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(NULLIF(full_name,''), phone::text) FROM users WHERE id = $1`, id).Scan(&label)
	return label
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
