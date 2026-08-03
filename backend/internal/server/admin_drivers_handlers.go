package server

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// قائمة السائقين مع صندوق كل منهم وطلباته الجارية.
func (s *Server) handleListDrivers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT u.id, u.phone, u.full_name, u.status,
		       -- الدوام: بُني علَمُه للسائق ولم تكن اللوحة تراه، فبقيت العمليات
		       -- تسأل «من يعمل الآن؟» بالهاتف وهي تملك الجواب في قاعدتها.
		       u.on_shift, u.shift_started_at,
		       COALESCE(cb.held, 0),
		       (SELECT count(*) FROM orders o WHERE o.driver_id = u.id AND o.closed_at IS NULL),
		       (SELECT count(*) FROM orders o WHERE o.driver_id = u.id AND o.status = 'delivered'
		        AND o.delivered_at::date = now()::date)
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'driver'
		LEFT JOIN driver_cash_boxes cb ON cb.driver_id = u.id
		ORDER BY u.created_at`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type driver struct {
		ID             string     `json:"id"`
		Phone          string     `json:"phone"`
		FullName       string     `json:"full_name"`
		Status         string     `json:"status"`
		OnShift        bool       `json:"on_shift"`
		ShiftStartedAt *time.Time `json:"shift_started_at"`
		CashHeld       int64      `json:"cash_held"`
		OpenOrders     int        `json:"open_orders"`
		DeliveredToday int        `json:"delivered_today"`
	}
	out := []driver{}
	for rows.Next() {
		var d driver
		if err := rows.Scan(&d.ID, &d.Phone, &d.FullName, &d.Status,
			&d.OnShift, &d.ShiftStartedAt,
			&d.CashHeld, &d.OpenOrders, &d.DeliveredToday); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, d)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"drivers":    out,
		"cash_limit": s.cashbox.Limit(r.Context()),
	})
}

func (s *Server) handleDriverCashStatement(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	st, err := s.cashbox.StatementFor(r.Context(), chi.URLParam(r, "id"), limit)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}

// تسليم الصندوق (كلياً أو جزئياً) — أدمن/مالية فقط.
func (s *Server) handleDriverSettle(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Amount int64  `json:"amount"`
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	driverID := chi.URLParam(r, "id")
	held, err := s.cashbox.Settle(r.Context(), driverID, req.Amount, req.Note, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// نقدٌ يُسلَّم في مكتب: لا إيصال إلكتروني له إلا هذا القيد. ومن أنكر
	// التسليم أو أنكر الاستلام، فالسطر هنا هو ما يُرجَع إليه.
	s.audit(r, "finance.driver_settle", "user", driverID, map[string]any{
		"amount": req.Amount, "note": req.Note, "held_after": held,
	})

	// نقود تنتقل من يد إلى يد: صاحبها يعرف، وشاشة الصناديق تتحدّث لحظياً.
	//
	// **والإشعارُ يقول المبلغَ والباقي.**
	//
	// كان نصُّه `req.Note` وحدَه — **والملاحظةُ اختيارية**، فإن تُركت فارغةً
	// وصل السائقَ «سُلّم صندوقك» بلا رقم. **وخبرُ مالٍ لا يقول كم مالٌ خبرٌ
	// يجب أن يُتحقّق منه في مكانٍ آخر** — فلا يُغني عن السؤال الذي وُضع
	// ليمنعه، **ويترك بابَ الخلاف مفتوحاً: «سلّمتُ خمسين» «بل أربعين».**
	body := fmt.Sprintf("%d — والباقي بذمّتك %d", req.Amount, held)
	if req.Note != "" {
		body += " · " + req.Note
	}
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: driverID, Kind: notifications.KindWallet,
		Title: notifTitles.cashSettled, Body: body,
		Entity: "cashbox", Href: "/",
	})
	s.touch("wallet", "ops")
	s.touchUser(driverID, "wallet")
	httpx.JSON(w, http.StatusOK, map[string]any{"held": held})
}
