package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/referrals"
)

// ══════════════════════════════════════════════════════════════════════
// **شاهدُ الإحالة الحيّ — رصيدٌ حقيقيٌّ في محفظةٍ حقيقيّة** (Batch 4)
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٩-٢٥: «اضبط مكافآتِ الإحالة الأربعَ وما بعدَها بقيمٍ
//  اختباريّةٍ متمايزة، احفظها واستعدها، وادعُ خمسةَ مدعوّين ثابتين عبر مسار
//  Attach الحقيقيّ، وأكمل التأهيلَ عبر المسار الحقيقيّ، ثمّ **تحقّق من رصيد
//  محفظة الداعي فعليّاً**: الأولى→L1 … الخامسة→reward_rest. لا تزوّر الرصيدَ
//  بحقن SQL».)
//
// # لماذا هذه القدرةُ آمنة — **مسارٌ حقيقيٌّ على هويّاتٍ ثابتة**
//
// **لا توكن، ولا صلاحيّةَ أدمن، ولا معرّفَ مستخدمٍ من الطلب.** الهويّاتُ
// ثابتة (داعٍ واحدٌ وخمسةُ مدعوّين بأرقامٍ محجوزةٍ لـQA)، والقيمُ ثابتة،
// **وتسقط مغلقةً في الإنتاج** (المنادي `handleQAStagingSeed` يحرس
// `qaStagingEnabled`، وحارسٌ ثانٍ هنا).
//
// **والصرفُ يمرّ بـ`referrals.Attach` ثمّ `referrals.SettleOnSignup`
// الحقيقيّين** — نفسُ ما يقع عند تسجيلِ زبونٍ جاء برمز دعوة. **والرصيدُ
// يُقاس بـ`wallet.Balance` قبلَ الصرفِ وبعدَه** — لا حقنَ صفٍّ، لا تزويرَ
// رقم. **والحسبةُ تعود إلى الصفر**: يُعكس ما صُرف بقيدٍ مزدوجٍ متوازنٍ
// (الداعي ↔ الخزينة) فيبقى الدفترُ متوازناً ولا ينمو الرصيدُ عبر التكرار.

const (
	// qaRefInviterPhone **الداعي الثابتُ الوحيد** — رقمٌ محجوزٌ لـQA.
	qaRefInviterPhone = "+963900556000"
)

// qaRefInviteePhones **المدعوّون الخمسةُ الثابتون** — الترتيبُ يعطي الرتب
// ١..٥، فالأوّلُ رتبةٌ أولى (L1) والخامسُ رتبةٌ خامسة (reward_rest).
var qaRefInviteePhones = []string{
	"+963900556001",
	"+963900556002",
	"+963900556003",
	"+963900556004",
	"+963900556005",
}

// qaRefRewards **القيمُ الاختباريّةُ المتمايزة** — متمايزةٌ كلُّها كي يُقرأ كلُّ
// رتبةٍ على حدة: لو تساوت رتبتان لَما عُرف أيُّهما صُرف.
var qaRefRewards = map[string]int64{
	"referral.reward_1":    1000,
	"referral.reward_2":    2000,
	"referral.reward_3":    3000,
	"referral.reward_4":    4000,
	"referral.reward_rest": 500,
}

// qaRefExpected مكافأةُ كلّ رتبةٍ كما ستُصرف — ١..٥ (والخامسةُ = reward_rest).
var qaRefExpected = []int64{1000, 2000, 3000, 4000, 500}

// qaReferralWitness يُجري شاهدَ الإحالة كاملاً ويردّ تقريراً بنّياً بالنتائج.
//
// **حتميٌّ وقابلٌ لإعادة التشغيل**: ينظّف حالةَ الإحالةِ السابقةَ للهويّات
// الثابتة (صفوفُ `referrals` وحجوزُ `phone_claims`) قبلَ البدء، ويعكس
// المصروفَ وينظّف بعدَه، ويستعيد الإعداداتِ في كلّ الأحوال.
func (s *Server) qaReferralWitness(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()
	actor := s.qaCoverageActor(ctx) // معرّفُ موظّفٍ ثابتٍ لكتابة الأثر — لا جلسة
	ip := clientIP(r)

	// ── ١) الهويّاتُ الثابتة عبر المسار الحقيقيّ ─────────────────────────
	inviter, err := s.identity.EnsureUserWithRole(ctx, actor, qaRefInviterPhone, identity.RoleCustomer, "QA Referral Inviter", "", ip)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	inviteeIDs := make([]string, len(qaRefInviteePhones))
	inviteePhones := make([]string, len(qaRefInviteePhones))
	for i, p := range qaRefInviteePhones {
		u, e := s.identity.EnsureUserWithRole(ctx, actor, p, identity.RoleCustomer, "QA Referral Invitee", "", ip)
		if e != nil {
			s.respondErr(w, e)
			return
		}
		inviteeIDs[i] = u.ID
		norm, _ := identity.NormalizePhone(p)
		inviteePhones[i] = norm
	}

	// ── ٢) حفظُ الإعداداتِ الحاليّةِ ثمّ ضبطُ القيمِ الاختباريّة ───────────
	prevInt := map[string]int64{}
	for k, v := range qaRefRewards {
		prevInt[k] = s.settings.GetInt(ctx, k)
		if err := s.settings.Set(ctx, k, v, nil); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	prevRewardOn := s.settings.GetString(ctx, "referral.reward_on")
	if err := s.settings.Set(ctx, "referral.reward_on", referrals.OnSignup, nil); err != nil {
		s.respondErr(w, err)
		return
	}
	// **الاستعادةُ في كلّ الأحوال** — لا يبقى إعدادٌ اختباريٌّ نافذاً.
	defer func() {
		rc := context.Background()
		for k, v := range prevInt {
			_ = s.settings.Set(rc, k, v, nil)
		}
		if prevRewardOn != "" {
			_ = s.settings.Set(rc, "referral.reward_on", prevRewardOn, nil)
		}
	}()

	// ── ٣) تنظيفُ الحالةِ السابقةِ — إعادةُ التشغيلِ نظيفة ───────────────
	if err := s.qaReferralCleanState(ctx, inviter.ID, inviteeIDs, inviteePhones); err != nil {
		s.respondErr(w, err)
		return
	}

	// ── ٤) رمزُ الداعي عبر المسار الحقيقيّ ───────────────────────────────
	code, err := s.referrals.MyCode(ctx, inviter.ID)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// ── ٥) حمايةُ «لا يدعو أحدٌ نفسَه» ───────────────────────────────────
	selfErr := s.referrals.Attach(ctx, inviter.ID, code)
	selfBlocked := errors.Is(selfErr, referrals.ErrSelfInvite)

	// ── ٦) الرتبُ الخمسُ — نسبٌ حقيقيٌّ ثمّ صرفٌ حقيقيٌّ، وقياسُ الرصيد ─────
	type tierResult struct {
		Rank     int    `json:"rank"`
		Invitee  string `json:"invitee_id"`
		Delta    int64  `json:"delta"`
		Expected int64  `json:"expected"`
		OK       bool   `json:"ok"`
	}
	tiers := make([]tierResult, 0, len(inviteeIDs))
	var creditedTotal int64
	allTiersOK := true
	for i, invitee := range inviteeIDs {
		if err := s.referrals.Attach(ctx, invitee, code); err != nil {
			s.respondErr(w, err)
			return
		}
		before, err := s.wallet.Balance(ctx, inviter.ID)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		// **الصرفُ عند إكمال التسجيل** — نفسُ نداءِ `ConfirmSignup` الحقيقيّ.
		s.referrals.SettleOnSignup(ctx, invitee, actor)
		after, err := s.wallet.Balance(ctx, inviter.ID)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		delta := after - before
		creditedTotal += delta
		ok := delta == qaRefExpected[i]
		if !ok {
			allTiersOK = false
		}
		tiers = append(tiers, tierResult{Rank: i + 1, Invitee: invitee, Delta: delta, Expected: qaRefExpected[i], OK: ok})
	}

	// ── ٧) «تُصرف مرّةً واحدةً» — صرفٌ ثانٍ لنفسِ المدعوّ لا يزيد شيئاً ─────
	payOnceBefore, err := s.wallet.Balance(ctx, inviter.ID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.referrals.SettleOnSignup(ctx, inviteeIDs[0], actor)
	payOnceAfter, err := s.wallet.Balance(ctx, inviter.ID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	payOnceOK := payOnceAfter == payOnceBefore

	// ── ٨) «مرّةً واحدةً للرقم» — إعادةُ نسبِ نفسِ المدعوّ لا تُنشئ صفّاً ──
	countBefore, err := s.qaReferralCount(ctx, inviter.ID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.referrals.Attach(ctx, inviteeIDs[0], code); err != nil {
		s.respondErr(w, err)
		return
	}
	countAfter, err := s.qaReferralCount(ctx, inviter.ID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	reattachBlockedOK := countAfter == countBefore

	// ── ٩) التنظيف — عكسُ المصروفِ (قيدٌ مزدوجٌ متوازن) ثمّ حذفُ الصفوف ────
	reversed, err := s.qaReferralReverseWallet(ctx, inviter.ID, creditedTotal, actor)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.qaReferralCleanState(ctx, inviter.ID, inviteeIDs, inviteePhones); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touchUser(inviter.ID, "wallet")

	allOK := selfBlocked && allTiersOK && payOnceOK && reattachBlockedOK && reversed == creditedTotal
	s.logger.Warn("QA referral witness (staging-only)",
		"inviter", inviter.ID, "credited_total", creditedTotal, "reversed", reversed, "all_ok", allOK)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"kind":                "referral_witness",
		"reward_settings":     qaRefRewards,
		"self_invite_blocked": selfBlocked,
		"tiers":               tiers,
		"pay_once_ok":         payOnceOK,
		"reattach_blocked_ok": reattachBlockedOK,
		"credited_total":      creditedTotal,
		"wallet_reversed":     reversed,
		"all_ok":              allOK,
	})
}

// qaReferralCount عددُ من نُسبوا لهذا الداعي — لشهودِ «لا صفَّ مكرّر».
func (s *Server) qaReferralCount(ctx context.Context, inviterID string) (int, error) {
	var n int
	err := s.pg.QueryRow(ctx, `SELECT count(*) FROM referrals WHERE inviter_id = $1`, inviterID).Scan(&n)
	return n, err
}

// qaReferralCleanState يحذف صفوفَ الإحالةِ وحجوزَ الأرقامِ للهويّاتِ الثابتةِ
// وحدَها — **فلا يمسّ إحالةَ أحدٍ آخر مهما كان.**
func (s *Server) qaReferralCleanState(ctx context.Context, inviterID string, inviteeIDs, inviteePhones []string) error {
	if _, err := s.pg.Exec(ctx,
		`DELETE FROM referrals WHERE inviter_id = $1 OR invitee_id = ANY($2)`,
		inviterID, inviteeIDs); err != nil {
		return err
	}
	// **حجزُ الرقمِ يُحسب بنفسِ بصمةِ `claimPhoneOnce`** — الملحُ من
	// `app_secrets` والهضمُ في القاعدة، فلا يُعاد بناؤه في الذاكرة.
	_, err := s.pg.Exec(ctx, `
		DELETE FROM phone_claims
		WHERE kind = 'referral'
		  AND phone_hash IN (
		    SELECT encode(sha256(((SELECT value FROM app_secrets WHERE key = 'phone_pepper') || u.phone)::bytea), 'hex')
		    FROM users u WHERE u.phone = ANY($1)
		  )`, inviteePhones)
	return err
}

// ══════════════════════════════════════════════════════════════════════
// **شاهدُ «بلاغٌ ضدَّ الزبون» الحيّ — عبرَ خدمةِ البلاغِ الحقيقيّة** (Batch 4)
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٩-٢٥: «سائقُ QA ثابتٌ يرفع بلاغاً حقيقيّاً ضدَّ زبونِ
//  QA الثابتِ عبرَ مسارِ البلاغِ الحقيقيّ؛ ثمّ يُتحقَّق أنّ `‎/my/tickets` لا
//  يحويه و`‎/me/reputation` يحويه ملثوماً بلا هويّةِ المُبلِّغ».)
//
// # لماذا هذه القدرةُ آمنة
//
// **لا توكن، ولا صلاحيّةَ أدمن، ولا معرّفَ طلبٍ من الطلب.** السائقُ ثابتٌ
// (رقمٌ محجوزٌ لـQA)، والزبونُ زبونُ QA الثابت، **والبلاغُ يمرّ بـ
// `support.DriverReport` الحقيقيّة** (نفسُ ما يفعله بابُ السائق). **وتسقط
// مغلقةً في الإنتاج** (حارسٌ في المنادي وحارسٌ هنا).
//
// **والتجهيزُ الوحيدُ محصورٌ في طلبٍ واحدٍ لزبونِ QA**: يُسنَد إليه السائقُ
// الثابتُ ويُضبط إغلاقُه داخلَ المهلة، **ثمّ يُستعادان فوراً** بعد فتح
// البلاغ — عمودان عكوسان على طلبٍ واحد، لا كنسٌ عامٌّ ولا حقنُ تذكرةٍ خام.

// qaReportDriverPhone **السائقُ الثابتُ الوحيد** — رقمٌ محجوزٌ لـQA، لا بلاغَ
// له إلّا هذا الشاهد.
const qaReportDriverPhone = "+963900556010"

// qaReportAgainstCustomer يرفع بلاغَ سائقٍ حقيقيّاً ضدَّ زبونِ QA الثابتِ عبرَ
// `support.DriverReport`، ويترك التذكرةَ قائمةً لشاهدِ الجهاز — تُحذف
// بـ `report_cleanup`.
func (s *Server) qaReportAgainstCustomer(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()
	actor := s.qaCoverageActor(ctx)
	ip := clientIP(r)

	// **السائقُ الثابت** عبرَ المسارِ الحقيقيّ.
	drv, err := s.identity.EnsureUserWithRole(ctx, actor, qaReportDriverPhone, "driver", "QA Report Driver", "", ip)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **زبونُ QA الثابت.**
	custPhone, ok := identity.NormalizePhone(qaStagingPhone)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	var custID string
	if err := s.pg.QueryRow(ctx, `SELECT id::text FROM users WHERE phone = $1`, custPhone).Scan(&custID); err != nil {
		s.respondErr(w, err)
		return
	}

	// **تنظيفُ بلاغاتِ الشاهدِ السابقة** — السائقُ الثابتُ لا تذكرةَ له إلّا
	// شاهد، فحذفُها يُعيد التشغيلَ نظيفاً (الردودُ تُحذف بالـcascade).
	if _, err := s.pg.Exec(ctx, `DELETE FROM tickets WHERE created_by = $1::uuid`, drv.ID); err != nil {
		s.respondErr(w, err)
		return
	}

	// **طلبٌ مغلقٌ لزبونِ QA بلا تذكرةٍ قائمة** — يُختار تجهيزاً.
	var orderID string
	var orderNumber int64
	var origDriver *string
	var origClosed *time.Time
	err = s.pg.QueryRow(ctx, `
		SELECT o.id::text, o.number, o.driver_id::text, o.closed_at
		FROM orders o
		WHERE o.customer_id = $1::uuid AND o.closed_at IS NOT NULL
		  AND NOT EXISTS (SELECT 1 FROM tickets t WHERE t.order_id = o.id AND t.status <> 'resolved')
		ORDER BY o.closed_at DESC LIMIT 1`, custID).
		Scan(&orderID, &orderNumber, &origDriver, &origClosed)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_no_closed_order", "errors.conflict"))
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// **تجهيزٌ لا مسارٌ مشهود** — يُسنَد السائقُ الثابتُ ويُضبط الإغلاقُ داخلَ
	// المهلة، ثمّ يُستعادان فورَ فتحِ البلاغ. البلاغُ نفسُه يمرّ بالخدمةِ الحقيقيّة.
	if _, err := s.pg.Exec(ctx,
		`UPDATE orders SET driver_id = $2::uuid, closed_at = now() WHERE id = $1::uuid`,
		orderID, drv.ID); err != nil {
		s.respondErr(w, err)
		return
	}
	restore := func() {
		rc := context.Background()
		_, _ = s.pg.Exec(rc, `UPDATE orders SET driver_id = $2::uuid, closed_at = $3 WHERE id = $1::uuid`,
			orderID, origDriver, origClosed)
	}

	t, err := s.support.DriverReport(ctx, drv.ID, orderID, "customer_conduct", "بلاغُ شاهدِ QA (Batch 4)")
	restore()
	if err != nil {
		s.respondErr(w, err)
		return
	}

	s.touch("ticket", "ops")
	s.logger.Warn("QA report-against-customer witness (staging-only)",
		"driver", drv.ID, "customer", custID, "order", orderID, "ticket", t.ID)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"kind":             "report_against_customer",
		"ticket_id":        t.ID,
		"ticket_number":    t.Number,
		"ticket_status":    t.Status,
		"order_id":         orderID,
		"order_number":     orderNumber,
		"against_customer": custID,
		"reporter_driver":  drv.ID,
	})
}

// qaReportCleanup يحذف بلاغاتِ السائقِ الثابتِ — تنظيفٌ نهائيٌّ بعد شاهدِ الجهاز.
func (s *Server) qaReportCleanup(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()
	drvPhone, ok := identity.NormalizePhone(qaReportDriverPhone)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	var drvID string
	err := s.pg.QueryRow(ctx, `SELECT id::text FROM users WHERE phone = $1`, drvPhone).Scan(&drvID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusOK, map[string]any{"deleted": 0, "note": "no_driver"})
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	tag, err := s.pg.Exec(ctx, `DELETE FROM tickets WHERE created_by = $1::uuid`, drvID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA report witness cleaned (staging-only)", "driver", drvID, "deleted", tag.RowsAffected())
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": tag.RowsAffected(), "kind": "report_cleanup"})
}

// qaReferralReverseWallet يعكس ما صُرف للداعي بقيدٍ مزدوجٍ متوازنٍ
// (الداعي ↔ الخزينة) في معاملةٍ واحدة — **فيبقى الدفترُ متوازناً ولا ينمو
// الرصيدُ عبر التكرار.** صفرٌ ⇒ لا شيءَ يُعكس.
func (s *Server) qaReferralReverseWallet(ctx context.Context, inviterID string, amount int64, actor string) (int64, error) {
	if amount <= 0 {
		return 0, nil
	}
	var tid string
	if err := s.pg.QueryRow(ctx, `SELECT user_id::text FROM wallets WHERE is_treasury LIMIT 1`).Scan(&tid); err != nil {
		return 0, err
	}
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, _, err := s.wallet.ApplyTxID(ctx, tx, inviterID, -amount, "adjustment",
		"qa-referral-witness-reverse", "عكسُ مكافآت شاهد الإحالة (تنظيف QA)", &actor); err != nil {
		return 0, err
	}
	if _, _, err := s.wallet.ApplyTxID(ctx, tx, tid, amount, "adjustment",
		"qa-referral-witness-reverse", "عكسُ مكافآت الإحالة — استعادةُ الخزينة (تنظيف QA)", &actor); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return amount, nil
}
