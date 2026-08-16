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
	errPayoutBelowMin   = httpx.NewError(http.StatusConflict, "payout_below_min", "errors.payout_below_min")
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
	// حدٌّ أدنى: طلبٌ بليرةٍ واحدة يمرّ بدورة الموافقة كاملةً ويشغل المالية،
	// وقيمة القرار أكبر من قيمة المبلغ. **ويُستثنى من يسحب رصيده كلَّه** —
	// من بقي له ألفٌ لا يُحبس عنه لأن الألف دون الحدّ.
	minAmount := s.settings.GetInt(r.Context(), "payouts.min_amount")
	if req.Amount < minAmount && req.Amount != balance {
		s.respondErr(w, errPayoutBelowMin)
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
	s.touchUser(uid, "wallet")
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

// handleAdminPayouts كل طلبات السحب (ترشيح بالحالة) — للأدمن والمالية.
func (s *Server) handleAdminPayouts(w http.ResponseWriter, r *http.Request) {
	// ══════════════════════════════════════════════════════════════════
	// **ومئتان صامتةٌ في المال**
	// ══════════════════════════════════════════════════════════════════
	//
	// (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)
	//
	// **كان `LIMIT 200` بلا عدٍّ ولا ترقيمٍ ولا كلمة** — والردُّ مصفوفةٌ
	// مجرّدة، **لا حقلَ فيه يقول «هناك أكثر».**
	//
	// **والمعلَّقُ يتصدّر فيخفّ الأثر** — لكنّه يبقى في التاريخ: من رشّح
	// «مدفوع» ليراجع ما صُرف **يرى آخرَ مئتين ويظنّها كلَّ ما دُفع**.
	// **وهذا مالٌ خرج، ومراجعتُه ناقصةً أسوأُ من عدمها.**
	status := r.URL.Query().Get("status")
	pg := pagingOf(r, 25)

	// **والعدُّ بشرط القائمة نفسِه** — نصّان يفترقان يوماً.
	var count int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*) FROM payout_requests p WHERE ($1 = '' OR p.status = $1)`,
		status).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}

	// ══════════════════════════════════════════════════════════════════
	// **ومجموعُ ما يُنتظر صرفُه**
	// ══════════════════════════════════════════════════════════════════
	//
	// **وكلُّ شاشةِ مالٍ في المنصّة تقول مجموعَها**: الخزينةُ رصيدَها،
	// والخسائرُ مجموعَها، والنزاعاتُ «كم لنا عند الناس»، وأموالٌ لم
	// تُستلم ما في الشارع. **وهذه وحدَها لا تقول كم عليها أن تدفع.**
	//
	// **ومالٌ لا يُرى مجموعاً لا يُخطَّط له.**
	//
	// **وللمعلَّق وحدَه ولا يتبع الترشيح**: سؤالُه «كم عليّ الآن؟» —
	// **ومجموعٌ يتبع مُرشِّحاً يقول صفراً لمن يقرأ المرفوضَ**، وهو لا
	// يخصّه.
	var pending int64
	if err := s.pg.QueryRow(r.Context(),
		`SELECT COALESCE(sum(amount), 0) FROM payout_requests WHERE status = 'pending'`).
		Scan(&pending); err != nil {
		s.respondErr(w, err)
		return
	}

	rows, err := s.pg.Query(r.Context(), payoutSelect+`
		WHERE ($1 = '' OR p.status = $1)
		ORDER BY (p.status = 'pending') DESC, p.created_at DESC
		LIMIT $2 OFFSET $3`, status, pg.PerPage, pg.Offset)
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
	res := paged("payouts", out, count, pg)
	res["pending_total"] = pending
	httpx.JSON(w, http.StatusOK, res)
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

	/* ══════════════════════════════════════════════════════════════════
	   **القرارُ خطوةٌ واحدةٌ — لا ثلاث**
	   ══════════════════════════════════════════════════════════════════

	   (كشفه فحصُ المشروع ٢٠٢٦-٠٨-٠٧، وأُصلح بقرار المالك: «نبدأ إذاً».)

	   كان ثلاثَ خطواتٍ على البِركة مباشرةً: قراءةُ الحالة، ثمّ خصمٌ في
	   معاملتِه، ثمّ تحديثُ الحالة في ثالثة.

	   **فموافقتان متزامنتان تقرآن «معلّق» كلتاهما وتمرّان الفحصَ كلتاهما**
	   — فيُخصَم الرصيدُ مرّتين ويُصرف السحبُ مرّتين. **وقيدُ `balance >= 0`
	   لا يوقفه إن كان الرصيدُ كافياً.**

	   قِيس بثمانية نداءاتٍ متزامنة: **صُرف مرّتين وخُصم ضعفُ المبلغ، في
	   سبعِ تشغيلاتٍ من عشر.**

	   **ولا يظهر في أيّ سجلّ**: كلا النداءين يردّ ٢٠٠، وكلا القيدين صالح.

	   **والقفلُ هو الحلّ لا قيدُ التفرّد**: `FOR UPDATE` يجعل الثانيةَ
	   تنتظر، **فتقرأ الحالةَ بعد أن كُتبت** فتراها `paid` وترفض. ومعاملةٌ
	   واحدةٌ تلفّ الثلاثة، **فإن سقط التحديثُ رجع الخصمُ معه** ولا يبقى مالٌ
	   خرج وطلبٌ معلّق.
	   ══════════════════════════════════════════════════════════════════ */
	tx, err := s.pg.Begin(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var userID string
	var amount int64
	var status string
	if err := tx.QueryRow(r.Context(),
		`SELECT user_id, amount, status FROM payout_requests WHERE id = $1 FOR UPDATE`, id).
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
		if _, err := s.wallet.ApplyTx(r.Context(), tx, userID, -amount, "payout",
			id, clip(req.Decision, 300), &actor); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	if _, err := tx.Exec(r.Context(), `
		UPDATE payout_requests SET status = $2, decision = $3, decided_by = $4, decided_at = now()
		WHERE id = $1`, id, req.Status, clip(req.Decision, 300), actor); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.respondErr(w, err)
		return
	}

	s.audit(r, "finance.payout_decide", "payout", id, map[string]any{
		"status": req.Status, "amount": amount, "user_id": userID,
		"decision": req.Decision,
	})

	title := notifTitles.payoutPaid
	if req.Status == "rejected" {
		title = notifTitles.payoutRejected
	}
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: userID, Kind: notifications.KindWallet, Title: title,
		Body: req.Decision, Entity: "payout", EntityID: id, Href: "/portal/wallet",
	})
	s.touch("wallet", "ops")
	// **وصاحبُ الطلب يرى قرارَه ورصيدَه فوراً** — لا حين يُحدّث الصفحة.
	s.touchUser(userID, "wallet")
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
