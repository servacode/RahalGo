package server

import (
	"context"
	"fmt"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// errDriverHasOpenOrders **لا يُغلق دوامُ من بيده طلبٌ حيّ** — الطلبُ في صندوقه
// والزبونُ ينتظره، وإغلاقُ دوامه لا يُعيده إلى الطابور.
var errDriverHasOpenOrders = httpx.NewError(http.StatusConflict,
	"driver_has_open_orders", "errors.driver_has_open_orders")

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
		        AND o.delivered_at::date = now()::date),
		       -- ══════════════════════════════════════════════════════
		       -- **وموضعٌ شاخ والورديّةُ مفتوحة**
		       -- ══════════════════════════════════════════════════════
		       --
		       -- (قِيس 2026-09-02: المحرّكُ يعرف آخرَ وقتِ موضعٍ
		       --  ويحجب الشائخَ عن الطلبات، **واللوحةُ لا تراه**.)
		       --
		       -- **ونظامُ الهاتف يقتل خدمةَ الموقع** — سامسونغ تُنيم
		       -- التطبيقات وشاومي أشرس. **فيقفل السائقُ الشاشةَ فيقف
		       -- موضعُه، وورديّتُه مفتوحة.**
		       --
		       -- **فيرى المكتبُ سائقاً على الدوام ويظنّه يعمل** —
		       -- ويسأله بالهاتف، والجوابُ في القاعدة.
		       u.last_location_at
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
		// LastLocationAt **آخرُ موضعٍ وصل** — وفارغٌ يعني «لم يصل قطّ».
		//
		// **وتُقرأ مع `OnShift`**: ورديّةٌ مفتوحةٌ وموضعٌ شاخ **علامةُ
		// تطبيقٍ قتله النظام**، لا سائقٍ متوقّف.
		LastLocationAt *time.Time `json:"last_location_at"`
	}
	out := []driver{}
	for rows.Next() {
		var d driver
		if err := rows.Scan(&d.ID, &d.Phone, &d.FullName, &d.Status,
			&d.OnShift, &d.ShiftStartedAt,
			&d.CashHeld, &d.OpenOrders, &d.DeliveredToday,
			&d.LastLocationAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, d)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"drivers":    out,
		"cash_limit": s.cashbox.Limit(r.Context()),
		// **وحدُّ الشيخوخة من المحرّك لا من اللوحة** — رقمٌ واحدٌ
		// يقرّره موضعٌ واحد، **ومن كتبه في الاثنين افترقا يوماً.**
		"stale_location_minutes": staleLocationMinutes,
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
	// **العملُ وعلامةُ تثبيتِ منع التكرار في معاملةٍ واحدة** — `XG-33`.
	s.WithIdempotentTx(w, r, func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error) {
		held, err := s.cashbox.SettleTx(ctx, q, driverID, req.Amount, req.Note, userIDFrom(r))
		if err != nil {
			return IdempotentBody{}, err
		}
		return IdempotentBody{
			Status:  http.StatusOK,
			Payload: map[string]any{"held": held},
			AfterCommit: func() {
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
					// **تسويةُ نقدِ السائق تخصّ تطبيقَه** — وهو يحمل تطبيقَ الزبون أيضاً.
					Apps: []string{notifications.AppDriver},
				})
				s.touch("wallet", "ops")
				s.touchUser(driverID, "wallet")
			},
		}, nil
	})
}

// handleAdminEndShift **تُغلق الإدارةُ دوامَ سائقٍ نسي أن يُغلقه.**
//
// # المسألة
//
// علَمُ الدوام بيد السائق وحدَه. **ومن ذهب إلى بيته ونسي أن يُطفئه يبقى في
// الدور**: يُعرض عليه كلُّ طلبٍ خمساً وأربعين ثانيةً ثمّ يمضي إلى غيره — **فكلُّ
// طلبٍ يتأخّر بمقدار غيابه**، وقد يمرّ على ثلاثةٍ غائبين فيضيع دقيقتان قبل أن
// يصل إلى من يعمل.
//
// **والعملياتُ تراه «على الدوام» ولا تملك إنزاله** — فتتّصل به، فإن لم يردّ
// انتظرت.
//
// # ولا تُفتَح بيد الإدارة
//
// **إغلاقٌ فقط لا تشغيل.** فتحُ الدوام إقرارٌ من إنسانٍ بأنّه جاهزٌ الآن
// **وشهادةٌ على نفسه** — ومن فُتح له دوامُه وهو نائمٌ تُسند إليه طلباتٌ لا
// يعرف بها. **والمنصةُ تُعلن ما تعلم**: تعلم أنّه لا يستجيب، ولا تعلم أنّه
// جاهز.
//
// # ويُخطَر بما وقع
//
// **ومن أُغلق دوامُه بلا علمه يظنّ أنّ النظام أعطبه** — فيشتكي، أو يظنّ أنّ
// لا طلباتِ اليوم.
func (s *Server) handleAdminEndShift(w http.ResponseWriter, r *http.Request) {
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

	// **ولا يُغلق دوامُ من بيده طلبٌ حيّ.**
	//
	// الطلبُ في صندوقه والزبونُ ينتظره، **وإغلاقُ دوامه لا يُعيد الطلبَ إلى
	// الطابور** — يتركه معلّقاً بيد من صار في النظام «غير عامل». ومن أراد أن
	// يُخرجه من طلبه فله «إسنادٌ يدويّ» في الطلب نفسِه.
	var open int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*) FROM orders WHERE driver_id = $1 AND closed_at IS NULL`,
		id).Scan(&open); err != nil {
		s.respondErr(w, err)
		return
	}
	if open > 0 {
		s.respondErr(w, errDriverHasOpenOrders)
		return
	}

	tag, err := s.pg.Exec(r.Context(), `
		UPDATE users SET on_shift = false, shift_started_at = NULL, updated_at = now()
		WHERE id = $1 AND on_shift`, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}

	if s.notify != nil {
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: id, Kind: notifications.KindAccount,
			Title: "أُغلق دوامُك من المنصة",
			Body:  req.Note,
			Href:  "/portal",
		})
	}
	s.audit(r, "ops.driver_shift_ended", "user", id, map[string]any{"note": req.Note})
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"on_shift": false})
}
