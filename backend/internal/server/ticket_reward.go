package server

// ══════════════════════════════════════════════════════════════════════
// **مكافأةُ بلاغ «السائق طلب رقمي»: توصيلةٌ مجّانيّة** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٦)
// ══════════════════════════════════════════════════════════════════════
//
// **الزبونُ الذي بلّغ عن سائقٍ يحاول سحبَه برّا المنصّة يستحقّ شكراً يُحسّ** — وتمنحه
// الإدارةُ بعد أن تتأكّد من البلاغ (محادثةُ الطلب أمامها).
//
// # ولماذا كودُ خصمٍ لا رصيدٌ في المحفظة
//
// **كودُ «توصيل مجّاني» آليّةٌ قائمةٌ بحسابها كاملاً** (`catalog/promos.go`): الإعفاءُ
// يُقيَّد `promo_delivery_waived` **والسائقُ يأخذ أجرتَه كاملة** والخزينةُ تتحمّل الفرق.
// **فلا سطرَ مالٍ جديد** — والرصيدُ في المحفظة حركةُ مالٍ تمرّ بالماليّة.
//
// **وكودٌ بسقف استعمالٍ واحدٍ لا يحتاج موافقةَ الماليّة** (`needsApproval`).
//
// # ومنعُ التكرار بلا هجرة
//
// **الكودُ يبدأ بـ`RG<رقم التذكرة>-`** — فإن وُجد كودٌ بهذه البداية فالمكافأةُ مُنحت،
// ويُردّ الكودُ نفسُه. **ولا جدولَ ربطٍ جديد.**

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/support"
)

var (
	// errRewardNotEligible **ليست بلاغَ «طلب رقمي» فتحه الزبون** — لا مكافأةَ عليها.
	errRewardNotEligible = httpx.NewError(http.StatusConflict,
		"reward_not_eligible", "errors.reward_not_eligible")
	// errRewardAlreadyGranted **مُنحت قبل** — ويُردّ الكودُ الموجود في `details.code`.
	errRewardAlreadyGranted = httpx.NewError(http.StatusConflict,
		"reward_already_granted", "errors.reward_already_granted")
)

// rewardValidDays **صلاحيةُ كود المكافأة** — ثلاثون يوماً.
const rewardValidDays = 30

// rewardCodeAlphabet **حروفُ الذيل العشوائيّ** — بلا ما يلتبس قراءةً (O/0 · I/1).
const rewardCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func rewardCodePrefix(ticketNumber int64) string {
	return "RG" + strconv.FormatInt(ticketNumber, 10) + "-"
}

func randomRewardTail(n int) (string, error) {
	out := make([]byte, n)
	max := big.NewInt(int64(len(rewardCodeAlphabet)))
	for i := range out {
		k, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = rewardCodeAlphabet[k.Int64()]
	}
	return string(out), nil
}

// rewardReplyText **ردُّ المكتب على البلاغ** — بالكود وتاريخ انتهائه.
func rewardReplyText(code, until string) string {
	return "شكراً لأنك بلّغتنا 🌷 إلك توصيلة مجانية: اكتب الكود " + code +
		" بالسلّة بطلبك الجاي. صالح لحسابك بس ولمرة وحدة لحد " + until + "."
}

// handleTicketRewardFreeDelivery `POST /admin/tickets/{id}/reward-free-delivery`.
//
// **بقدرتين**: الدعمُ على المسار (`support.manage`) والعروضُ هنا (`offers.manage`) —
// **فمن لا يملك صنعَ كودٍ لا يصنعه من بابٍ جانبيّ.**
func (s *Server) handleTicketRewardFreeDelivery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !s.hasCapability(r, authz.OffersManage) {
		s.respondErr(w, errForbiddenCap)
		return
	}
	t, err := s.support.Get(ctx, chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if t.Reason != support.ReasonOffPlatform || !t.OpenedByCustomer {
		s.respondErr(w, errRewardNotEligible)
		return
	}
	prefix := rewardCodePrefix(t.Number)
	// ══════════════════════════════════════════════════════════════════
	// **وقفلٌ على البلاغ، والفحصُ والإنشاءُ تحته في وصلةٍ واحدة** (مراجعةُ الفرع)
	// ══════════════════════════════════════════════════════════════════
	//
	// **كان فحصاً ثمّ إنشاءً بلا قفل** — فضغطتان متتاليتان أو موظّفان معاً يجدان «لا كود»
	// كلاهما **فيخرج كودان للبلاغ نفسِه.** **فقفلُ معاملةٍ باسم البلاغ**: الثاني ينتظر حتّى
	// يُثبَّت الأوّل، **ثمّ يجد كودَه فيردّ ٤٠٩ به.** ويُفكّ بإنهاء المعاملة أيّاً كان المخرج.
	//
	// **والإنشاءُ في المعاملة نفسِها** (`CreatePromoTx`) لا بوصلةٍ ثانية — **من أمسك وصلةً
	// وطلب أخرى والمنتظرون يمسكون الباقي جمد المسبح** (`XG-46`).
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('ticket_reward:' || $1))`, t.ID); err != nil {
		s.respondErr(w, err)
		return
	}
	var existing string
	err = tx.QueryRow(ctx,
		`SELECT code FROM promo_codes WHERE code LIKE $1 || '%' ORDER BY created_at LIMIT 1`,
		prefix).Scan(&existing)
	if err == nil {
		httpx.Error(w, &httpx.AppError{
			Status: errRewardAlreadyGranted.Status, Code: errRewardAlreadyGranted.Code,
			MessageKey: errRewardAlreadyGranted.MessageKey,
			Details:    map[string]any{"code": existing},
		})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, err)
		return
	}

	until := time.Now().In(damascusLoc()).AddDate(0, 0, rewardValidDays).Format("2006-01-02")
	kind, maxUses, once, active := "free_delivery", 1, true, true
	var promo *catalog.PromoCode
	var audited func()
	for attempt := 0; attempt < 3; attempt++ {
		tail, err := randomRewardTail(5)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		code := prefix + tail
		// **ونقطةُ حفظٍ لكلّ محاولة** — تعارضُ الكود يُفسد المعاملةَ كلَّها بلاها.
		sp, err := tx.Begin(ctx)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		promo, audited, err = s.catalog.CreatePromoTx(ctx, sp, userIDFrom(r), catalog.PromoInput{
			Code: &code, Kind: &kind, MaxUses: &maxUses, OncePerUser: &once,
			ExpiresOn: &until, Active: &active,
		}, clientIP(r))
		if errors.Is(err, catalog.ErrCodeTaken) {
			_ = sp.Rollback(ctx)
			continue
		}
		// **والكودُ لحساب المبلِّغ وحدَه** (قرارُ المالك ٢٠٢٦-١١-٠٦) — `validatePromo` يردّه
		// لغيره بـ`promo_not_yours`. وفي المعاملة نفسِها: لا كودَ يُثبَّت بلا ربط.
		if err == nil {
			_, err = sp.Exec(ctx, `UPDATE promo_codes SET bound_user_id = $2::uuid WHERE id = $1::uuid`,
				promo.ID, t.ComplainantID)
		}
		if err == nil {
			err = sp.Commit(ctx)
		}
		if err != nil {
			s.respondErr(w, err)
			return
		}
		break
	}
	if promo == nil {
		s.respondErr(w, catalog.ErrCodeTaken)
		return
	}
	// **والكودُ يُثبَّت فيُفكّ القفل** — لا ينتظر الثاني ردَّ التذكرة والإشعار.
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	audited()

	// **والردُّ على البلاغ نفسِه** — يقرؤه الزبونُ في شكواه. **وتذكرةٌ حُلّت لا يُكتب
	// فيها** (`ticket_resolved`): فيصله الكودُ بالإشعار وحدَه.
	text := rewardReplyText(promo.Code, until)
	if _, err := s.support.Reply(ctx, userIDFrom(r), t.ID, text); err != nil &&
		!errors.Is(err, support.ErrTicketClosed) {
		s.logger.Warn("مكافأةُ البلاغ: تعذّر الردُّ على التذكرة", "ticket", t.ID, "error", err)
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: t.ComplainantID, Kind: notifications.KindTicket,
		Title: notifTitles.ticketReply, Body: text,
		Entity: "ticket", EntityID: t.ID, Href: "/portal/orders",
	})
	s.audit(r, "ops.ticket_reward_free_delivery", "ticket", t.ID, map[string]any{
		"promo_id": promo.ID, "code": promo.Code, "expires_on": until,
	})
	httpx.JSON(w, http.StatusCreated, map[string]any{
		"code": promo.Code, "promo_id": promo.ID, "expires_on": until,
	})
}
