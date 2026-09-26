package server

import (
	"context"
	"errors"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"net/http"
	"slices"
	"strconv"
	"strings"
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
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	UserPhone string `json:"user_phone"`
	Amount    int64  `json:"amount"`
	Status    string `json:"status"`
	Note      string `json:"note"`
	Decision  string `json:"decision"`
	// **والطبقاتُ الثلاثُ تُعرَض ولا يُسمّى المجموعُ متاحاً** —
	// `XG-12`: **الماليّةُ تقرّر على ما يجوز صرفُه لا على ما تراه.**
	Balance   int64      `json:"balance"`   // المُقيَّد
	Reserved  int64      `json:"reserved"`  // المحجوزُ لطلباتٍ جارية
	Available int64      `json:"available"` // = المُقيَّد − المحجوز
	CreatedAt time.Time  `json:"created_at"`
	DecidedAt *time.Time `json:"decided_at"`
}

const payoutSelect = `
	SELECT p.id, p.user_id, COALESCE(NULLIF(u.full_name,''), u.phone::text), u.phone,
	       p.amount, p.status, p.note, p.decision,
	       COALESCE((SELECT w.balance  FROM wallets w WHERE w.user_id = p.user_id), 0),
	       COALESCE((SELECT w.reserved FROM wallets w WHERE w.user_id = p.user_id), 0),
	       COALESCE((SELECT w.balance - w.reserved FROM wallets w WHERE w.user_id = p.user_id), 0),
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
			&p.Status, &p.Note, &p.Decision, &p.Balance, &p.Reserved, &p.Available,
			&p.CreatedAt, &p.DecidedAt); err != nil {
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
	// ══════════════════════════════════════════════════════════════
	// **والأهليّةُ بالمتاح لا بالرصيد** — `XG-12` · `AQ-3`
	// ══════════════════════════════════════════════════════════════
	//
	// **والرصيدُ يشمل ما حُجز لطلبٍ آخرَ جارٍ** — **ومن سُئل عن
	// رصيده أُجيب بمالٍ ليس له أن ينفقه.**
	balance, err := s.wallet.Available(r.Context(), uid)
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

	// ══════════════════════════════════════════════════════════════
	// **والطلبُ وحجزُه فعلٌ واحد** — `XG-12`
	// ══════════════════════════════════════════════════════════════
	//
	// **وحالان محرَّمتان**: **طلبٌ ثُبِّت بلا حجز** فيُنفَق مالُه ثمّ
	// يرتدّ قرارُه · **وحجزٌ ثُبِّت بلا طلب** فيُجمَّد مالٌ لا سبب له.
	//
	// **والفحصُ فوق القفل لا بدلَ منه**: **المتاحُ يُقرأ ثانيةً داخل
	// المعاملة** — **فطلبان متزامنان لا يحجزان ضعفَ المتاح**،
	// **والقيدُ في الجدول حارسٌ ثالث.**
	// ══════════════════════════════════════════════════════════════════
	// **والطلبُ وحجزُه وعلامةُ منعِ التكرار فعلٌ واحد** — `XG-33`
	// ══════════════════════════════════════════════════════════════════
	//
	// **ملفوفٌ بـ`s.idempotent` (server.go) فهو مسارٌ محميّ** — والحارسُ
	// البنيويّ (`TestIDEM_AllProtectedPathsUseCoordinator`) يفرض أن يمرّ عبر
	// `WithIdempotentTx`: **فطلبُ السحب وحجزُه وعلامةُ التثبيت في معاملةٍ
	// واحدة**، **فلا نقرةٌ مزدوجةٌ تُنشئ طلبَين ولا علامةٌ تُكتب خارجَ الحجز.**
	// **وضمانةُ الطلبِ المعلَّقِ الواحد باقيةٌ**: الفهرسُ الفريدُ يردّ الثانيَ
	// `errPayoutPending` (نقرةٌ مزدوجةٌ بمفتاحٍ واحدٍ ⇒ استرجاعُ الأوّل).
	s.WithIdempotentTx(w, r, func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error) {
		// **وقفلُ صفّ المحفظة يُرتِّب المتزاحمَين** — والثاني يقرأ
		// بعد أن كُتب الأوّل.
		if _, e := q.Exec(ctx, `
			INSERT INTO wallets (user_id) VALUES ($1)
			ON CONFLICT (user_id) DO NOTHING`, uid); e != nil {
			return IdempotentBody{}, e
		}
		var bal, res int64
		if e := q.QueryRow(ctx,
			`SELECT balance, reserved FROM wallets WHERE user_id = $1 FOR UPDATE`,
			uid).Scan(&bal, &res); e != nil {
			return IdempotentBody{}, e
		}
		if req.Amount > bal-res {
			return IdempotentBody{}, errPayoutOver
		}
		var id string
		if e := q.QueryRow(ctx, `
			INSERT INTO payout_requests (user_id, amount, note) VALUES ($1, $2, $3)
			RETURNING id`, uid, req.Amount, clip(req.Note, 300)).Scan(&id); e != nil {
			if isUniqueViolation(e) {
				return IdempotentBody{}, errPayoutPending // الفهرس الفريد يمنع طلبين معلّقين
			}
			return IdempotentBody{}, e
		}
		if e := s.wallet.ReserveTx(ctx, q, uid, req.Amount); e != nil {
			return IdempotentBody{}, e
		}
		return IdempotentBody{
			Status:  http.StatusCreated,
			Payload: map[string]any{"id": id},
			AfterCommit: func() {
				// المالية تعرف فوراً — الطلب بلا متابع يبقى معلّقاً بلا نهاية
				s.notify.NotifyRoles(r.Context(), []string{"admin", "finance"}, notifications.Input{
					Kind: notifications.KindWallet, Title: notifTitles.payoutRequested,
					Body: s.userLabel(r.Context(), uid), Entity: "payout", EntityID: id,
					Href: "/dashboard/payouts",
				})
				s.touch("wallet", "ops")
				s.touchUser(uid, "wallet")
			},
		}, nil
	})
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
		Status   string `json:"status"`
		Decision string `json:"decision"`
	}](r)
	// **والحالاتُ ستٌّ بعقد `AQ-3`** — ولا يُقبَل ما ليس منها.
	if err != nil || !slices.Contains(
		[]string{"processing", "paid", "rejected", "failed", "reversed"}, req.Status) {
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
	// **العملُ وعلامةُ تثبيتِ منع التكرار في معاملةٍ واحدة** — `XG-33`.
	//
	// **وكان هذا المسارُ يملك معاملتَه** — **فصار يشاركها المنسّق**،
	// ولا معاملتان في فعلٍ واحد.
	s.WithIdempotentTx(w, r, func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error) {
		var userID string
		var amount int64
		var status string
		if err := q.QueryRow(ctx,
			`SELECT user_id, amount, status FROM payout_requests WHERE id = $1 FOR UPDATE`, id).
			Scan(&userID, &amount, &status); err != nil {
			return IdempotentBody{}, httpx.ErrNotFound
		}
		// ══════════════════════════════════════════════════════════
		// **وآلةُ الحالات تُحرَس بمصدرها لا بوجهتها وحدَها**
		// ══════════════════════════════════════════════════════════
		//
		//	pending    → processing · paid · rejected · failed
		//	processing → paid · failed
		//	paid       → reversed
		//
		// **و`paid` لا تُعاد** — الحارسُ هنا هو ما يمنع خصماً ثانياً
		// لقرارٍ يُكرَّر، **فوق منع التكرار.**
		allowed := map[string][]string{
			"pending":    {"processing", "paid", "rejected", "failed"},
			"processing": {"paid", "failed"},
			"paid":       {"reversed"},
		}
		if !slices.Contains(allowed[status], req.Status) {
			return IdempotentBody{}, errPayoutClosed
		}

		actor := userIDFrom(r)
		// ══════════════════════════════════════════════════════════
		// **وكلُّ حالٍ تفعل بالحجز ما يوجبه معناها** — `XG-12`
		// ══════════════════════════════════════════════════════════
		//
		//	paid       يُفكّ الحجزُ ويُخصَم — **فعلٌ واحد**
		//	rejected   يُفكّ بلا خصم — لم يقع صرف
		//	failed     **فشلٌ مُثبَتٌ ولا صرفَ وقع** — يُفكّ بلا خصم
		//	processing **بدأ الصرفُ ولم يثبت** — **الحجزُ كما هو**
		//	reversed   دُفع ثمّ ارتدّ — قيدٌ مقابلٌ يُعيد المال
		//
		// **و`failed` ليست لمجهول النتيجة** — **ومجهولُها يبقى
		// `processing` حتّى تُثبت المصالحةُ حقيقتَه**، **ولا يُخلَق
		// مالٌ من شكّ.**
		switch req.Status {
		case "paid":
			// **ويُفكّ ثمّ يُخصَم** — فالقيد `reserved <= balance`
			// يرفض العكس.
			if _, err := s.wallet.SettleReservedTx(ctx, q, userID, amount, "payout",
				id, clip(req.Decision, 300), &actor); err != nil {
				return IdempotentBody{}, err
			}
		case "rejected", "failed":
			if err := s.wallet.ReleaseTx(ctx, q, userID, amount); err != nil {
				return IdempotentBody{}, err
			}
		case "processing":
			// **ولا يُخصَم لأنّ الصرفَ بدأ** — البدءُ ليس نجاحاً.
		case "reversed":
			// ══════════════════════════════════════════════════════
			// **والارتدادُ قيدٌ مقابلٌ لا محوٌ لتاريخ**
			// ══════════════════════════════════════════════════════
			//
			// **الخصمُ الأوّلُ وقع ويبقى في الدفتر** — **ودفترٌ
			// يُمحى منه سطرٌ لا يُراجَع.** **فيُقيَّد ردٌّ يُعيد
			// المالَ إلى صاحبه.**
			//
			// **ولا حجزَ يُفكّ**: فُكّ يومَ `paid`.
			if _, err := s.wallet.ApplyTx(ctx, q, userID, amount, "refund",
				id, clip(req.Decision, 300), &actor); err != nil {
				return IdempotentBody{}, err
			}
		}
		if _, err := q.Exec(ctx, `
			UPDATE payout_requests SET status = $2, decision = $3, decided_by = $4, decided_at = now()
			WHERE id = $1`, id, req.Status, clip(req.Decision, 300), actor); err != nil {
			return IdempotentBody{}, err
		}

		// **والأثرُ يُقيَّد في المعاملة نفسِها** — `PF-06`:
		// **فعلٌ حسّاسٌ نجح بلا أثرٍ لا يُراجَع ولا يُنازَع فيه.**
		// **وسقوطُ القيد يُسقط الفعلَ كلَّه** — وذلك هو المقصود.
		if err := s.auditTx(ctx, q, r, "finance.payout_decide", "payout", id,
			map[string]any{
				"status": req.Status, "amount": amount, "user_id": userID,
				"decision": req.Decision,
			}); err != nil {
			return IdempotentBody{}, err
		}
		return IdempotentBody{
			Status:  http.StatusOK,
			Payload: map[string]any{"updated": true},
			AfterCommit: func() {

				// ══════════════════════════════════════════════════════════
				// **والإشعارُ يقول أثرَه في الرصيد صراحةً** — لا القرارَ وحدَه
				// ══════════════════════════════════════════════════════════
				//
				// **كان الجسدُ ملاحظةَ الماليّة فقط** — **فيقرأ المندوبُ «صُرف
				// طلب السحب» ويرى رصيدَه نقص ولا يعرف لماذا.** (قرارُ المالك:
				// «لازم يكون واضحاً أنّه خُصم من الرصيد ليعرف الشخصُ صحّ».)
				//
				// **فيُقال المبلغُ واتّجاهُه**: `paid` خُصم من الرصيد، و`rejected`
				// أُعيد إليه (فُكّ الحجز). وملاحظةُ الماليّة تُلحَق إن وُجدت.
				title := notifTitles.payoutPaid
				body := "خُصم " + fmtMoneyAr(amount) + " من رصيدك"
				switch req.Status {
				case "rejected":
					title = notifTitles.payoutRejected
					body = "أُعيد " + fmtMoneyAr(amount) + " إلى رصيدك"
				case "failed", "reversed":
					// **رُدّ المالُ إلى الرصيد** — فُشل الصرفُ أو ارتدّ.
					body = "أُعيد " + fmtMoneyAr(amount) + " إلى رصيدك"
				case "processing":
					body = "طلبُ سحبك قيد الصرف"
				}
				if note := strings.TrimSpace(req.Decision); note != "" {
					body += " · " + note
				}
				s.notify.Notify(r.Context(), notifications.Input{
					UserID: userID, Kind: notifications.KindWallet, Title: title,
					Body: body, Entity: "payout", EntityID: id, Href: "/portal/wallet",
					// **إلى تطبيق المستحقِّ وحدَه** (OBS-R7): الدفعةُ لعاملٍ
					// (مندوبٍ/سائقٍ/متجر)، فيُوجَّه القرارُ إلى تطبيقه لا إلى
					// تطبيق الزبون على جهازه نفسِه. يُحسب من دور المستحقّ.
					Apps: s.payeeWorkerApps(r.Context(), userID),
				})
				s.touch("wallet", "ops")
				// **وصاحبُ الطلب يرى قرارَه ورصيدَه فوراً** — لا حين يُحدّث الصفحة.
				s.touchUser(userID, "wallet")
			},
		}, nil
	})
}

// payeeWorkerApps **تطبيقاتُ العامل المستحقِّ الدفعة** — يُحسب من أدواره
// فيُوجَّه إشعارُ قرار الدفعة إلى تطبيقه (المندوب/السائق/المتجر) لا إلى تطبيق
// الزبون على جهازه نفسِه (OBS-R7). **والدفعةُ لعاملٍ دائماً** (البابُ محصورٌ
// في `payoutRoles`)، فلا يعود فارغاً في الممارسة؛ ولو عاد فارغاً احتياطاً
// وُجِّه الإشعارُ كما كان (بلا قيد) فلا يضيع.
func (s *Server) payeeWorkerApps(ctx context.Context, userID string) []string {
	rows, err := s.pg.Query(ctx,
		`SELECT role_code FROM user_roles WHERE user_id = $1::uuid`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var role string
		if rows.Scan(&role) != nil {
			continue
		}
		switch role {
		case "sales":
			out = append(out, notifications.AppRep)
		case "driver":
			out = append(out, notifications.AppDriver)
		case "merchant":
			out = append(out, notifications.AppMerchant)
		}
	}
	return out
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

// fmtMoneyAr **مبلغٌ بفواصل الآلاف ووحدةِ العملة** — «100000» ⟵ «100,000 ل.س».
//
// **يُقرأ في إشعارٍ لا يُنقَر** — فالفواصلُ تجعل المبلغَ الكبيرَ مقروءاً بلمحة،
// **ومن رأى «100000» بلا فاصلةٍ في إشعارٍ عابرٍ يخطئ في قدره.**
func fmtMoneyAr(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var b []byte
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			b = append(b, ',')
		}
		b = append(b, s[i])
	}
	out := string(b)
	if neg {
		out = "-" + out
	}
	return out + " ل.س"
}
