package server

import (
	"context"
	"net/http"
	"sync"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/referrals"
)

// ══════════════════════════════════════════════════════════════════════
// **شاهدُ الإحالة من واجهة الزبون — تسليحٌ وتنظيف** (Batch 5، Customer Final)
// ══════════════════════════════════════════════════════════════════════
//
// (قرار المالك ٢٠٢٦-٠٩-٢٥: شاهدُ الإحالة يجب أن يمرّ بواجهة الزبون الحقيقيّة
//  لا بالخادم وحدَه: رمزُ الداعي من التطبيق (QA1)، وخمسةُ حساباتٍ جديدةٍ تدخل
//  عبر مسار الدعوة الحقيقيّ — تسجيلٌ بالرمز ثمّ توثيقُ واتساب يصرف المكافأةَ —
//  ورصيدُ QA1 يتحرّك في التطبيق. reward_on=signup.)
//
// # ما تفعله هذه القدرة (وما لا تفعله)
//
// **لا تُسجّل ولا توثّق ولا تصرف** — ذلك كلُّه يقوده السائقُ عبر نقاط النهاية
// الحقيقيّة (`/auth/signup/*`، `/auth/whatsapp/verify-confirm`) بأرقام QA.
// **بل تُهيّئ وتنظّف فقط**: تضبط الرتبَ الخمسَ بقيمٍ متمايزةٍ معلومة، وتُخلي
// أرقامَ المدعوّين (٩٩٠–٩٩٤) لتسجيلٍ نظيف؛ ثمّ بعد الشاهد **تعكس رصيدَ QA1
// بقيدٍ مزدوجٍ متوازن** وتُجهِّل الحساباتِ الخمسةَ عبر مسار الحذف الحقيقيّ.
//
// **آمنةٌ كأخواتها**: لا توكن، تسقط مغلقةً في الإنتاج، الداعي QA1 الثابت
// والمدعوّون أرقامٌ محجوزةٌ لـQA، **والعكسُ متوازنٌ لا يمسّ صافي الدفتر.**

// qaRefUIRewards الرتبُ الخمسُ بقيمٍ متمايزةٍ — كي يُقرأ كلُّ رتبةٍ على حدة.
var qaRefUIRewards = map[string]int64{
	"referral.reward_1":    1000,
	"referral.reward_2":    2000,
	"referral.reward_3":    3000,
	"referral.reward_4":    4000,
	"referral.reward_rest": 500,
}

// qaRefUITotal مجموعُ الرتبِ الخمس — رصيدُ QA1 يجب أن يزيدَ به تماماً.
const qaRefUITotal int64 = 1000 + 2000 + 3000 + 4000 + 500 // 10500

// qaRefUISaved الإعداداتُ السابقةُ لتُستعاد — لا تبقى قيمةٌ اختباريّةٌ نافذة.
var qaRefUISaved struct {
	mu           sync.Mutex
	armed        bool
	prevInt      map[string]int64
	prevRewardOn string
}

// qaRefWitnessPhones **خمسةُ أرقامٍ محجوزةٌ نظيفةٌ لشاهد الإحالة من الواجهة** —
// **لم تُستعمل قطّ** (قرار المالك ٢٠٢٦-٠٩-٢٥): لا تُعاد أرقامُ شاهدِ التسجيل
// (٩٩٠–٩٩٤) لأنّ لها التزاماتٍ تاريخيّةً من الدفعة الرابعة تمنع تجهيلَها.
// **مضافةٌ إلى `qaOTPPhones`** (signup + whatsapp)، والترتيبُ يعطي الرتبَ ١..٥.
var qaRefWitnessPhones = []string{
	"+963900555980",
	"+963900555981",
	"+963900555982",
	"+963900555983",
	"+963900555984",
}

// qaRefInviteePhonesUI المدعوّون الخمسةُ عبر المسار الحقيقيّ.
func qaRefInviteePhonesUI() []string { return qaRefWitnessPhones }

// qaReferralUIArm يضبط الرتبَ ويُخلي أرقامَ المدعوّين لتسجيلٍ نظيف.
func (s *Server) qaReferralUIArm(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()
	ip := clientIP(r)
	actor := s.qaCoverageActor(ctx)

	qa1, err := s.qaUserIDByPhone(ctx, qaStagingPhone)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	phones, ids := s.qaResolveInvitees(ctx)
	// **تنظيفُ الحالةِ أوّلاً** (الأرقامُ ما زالت أصليّةً فتُمحى حجوزُها)، ثمّ التجهيل.
	if err := s.qaReferralCleanState(ctx, qa1, ids, phones); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.qaClearInviteePhoneClaims(ctx, phones); err != nil {
		s.respondErr(w, err)
		return
	}
	purged := s.qaAnonymizeInvitees(ctx, ip, actor)

	qaRefUISaved.mu.Lock()
	if !qaRefUISaved.armed {
		qaRefUISaved.prevInt = map[string]int64{}
		for k := range qaRefUIRewards {
			qaRefUISaved.prevInt[k] = s.settings.GetInt(ctx, k)
		}
		qaRefUISaved.prevRewardOn = s.settings.GetString(ctx, "referral.reward_on")
	}
	qaRefUISaved.armed = true
	qaRefUISaved.mu.Unlock()

	for k, v := range qaRefUIRewards {
		if err := s.settings.Set(ctx, k, v, nil); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	if err := s.settings.Set(ctx, "referral.reward_on", referrals.OnSignup, nil); err != nil {
		s.respondErr(w, err)
		return
	}

	code, err := s.referrals.MyCode(ctx, qa1)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	bal, _ := s.wallet.Balance(ctx, qa1)
	s.logger.Warn("QA referral UI witness armed (staging-only)", "inviter", qa1, "purged", purged)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"kind": "referral_ui_arm", "inviter_code": code, "inviter_balance": bal,
		"rewards": qaRefUIRewards, "expected_total": qaRefUITotal,
		"invitee_phones": phones, "purged": purged,
	})
}

// qaReferralUICleanup يعكس رصيدَ QA1 (متوازن) ويُجهِّل المدعوّين ويستعيد الرتب.
func (s *Server) qaReferralUICleanup(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()
	ip := clientIP(r)
	actor := s.qaCoverageActor(ctx)

	qa1, err := s.qaUserIDByPhone(ctx, qaStagingPhone)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	phones, ids := s.qaResolveInvitees(ctx)
	// **مقدارُ ما صُرف لـQA1 من هؤلاء المدعوّين** — يُحسَب قبل حذف الصفوف.
	var credited int64
	if len(ids) > 0 {
		_ = s.pg.QueryRow(ctx, `
			SELECT COALESCE(sum(reward_amount), 0) FROM referrals
			WHERE inviter_id = $1 AND invitee_id = ANY($2) AND rewarded_at IS NOT NULL`,
			qa1, ids).Scan(&credited)
	}
	reversed, err := s.qaReferralReverseWallet(ctx, qa1, credited, actor)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.qaReferralCleanState(ctx, qa1, ids, phones); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.qaClearInviteePhoneClaims(ctx, phones); err != nil {
		s.respondErr(w, err)
		return
	}
	purged := s.qaAnonymizeInvitees(ctx, ip, actor)

	// استعادةُ الرتبِ والوضع.
	qaRefUISaved.mu.Lock()
	if qaRefUISaved.armed {
		for k, v := range qaRefUISaved.prevInt {
			_ = s.settings.Set(ctx, k, v, nil)
		}
		if qaRefUISaved.prevRewardOn != "" {
			_ = s.settings.Set(ctx, "referral.reward_on", qaRefUISaved.prevRewardOn, nil)
		}
		qaRefUISaved.armed = false
	}
	qaRefUISaved.mu.Unlock()

	s.touchUser(qa1, "wallet")
	s.logger.Warn("QA referral UI witness cleaned (staging-only)", "inviter", qa1, "reversed", reversed, "purged", purged)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"kind": "referral_ui_cleanup", "reversed": reversed, "purged": purged,
	})
}

// qaUserIDByPhone معرّفُ مستخدمٍ برقمه (المعياريّ)، أو خطأٌ إن لم يوجد.
func (s *Server) qaUserIDByPhone(ctx context.Context, phone string) (string, error) {
	p, _ := identity.NormalizePhone(phone)
	var id string
	err := s.pg.QueryRow(ctx, `SELECT id::text FROM users WHERE phone = $1`, p).Scan(&id)
	return id, err
}

// qaResolveInvitees أرقامُ المدعوّين المعياريّة، ومعرّفاتُ الحيّة منهم (غيرِ المُجهَّلة).
func (s *Server) qaResolveInvitees(ctx context.Context) (phones []string, ids []string) {
	for _, p := range qaRefInviteePhonesUI() {
		if n, ok := identity.NormalizePhone(p); ok {
			phones = append(phones, n)
		}
	}
	rows, err := s.pg.Query(ctx,
		`SELECT id::text FROM users WHERE phone = ANY($1) AND status <> 'deleted'`, phones)
	if err != nil {
		return phones, nil
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return phones, ids
}

// qaClearInviteePhoneClaims **يمحو حجوزَ أرقام المدعوّين بالبصمةِ المحسوبةِ من
// الرقمِ الخام** — لا عبر جدول `users`: فحسابٌ جُهِّل صار رقمُه `deleted-…`،
// وحجزُه القديمُ يبقى مربوطاً ببصمةِ الرقمِ الأصليّ. **فلو لم يُمحَ لَفشلَ
// `claimPhoneOnce` في تسجيلٍ لاحقٍ على الرقم نفسِه.**
func (s *Server) qaClearInviteePhoneClaims(ctx context.Context, phones []string) error {
	if len(phones) == 0 {
		return nil
	}
	_, err := s.pg.Exec(ctx, `
		DELETE FROM phone_claims
		WHERE kind IN ('referral', 'signup_bonus')
		  AND phone_hash = ANY (
		    SELECT encode(sha256(((SELECT value FROM app_secrets WHERE key = 'phone_pepper') || p)::bytea), 'hex')
		    FROM unnest($1::text[]) AS p
		  )`, phones)
	return err
}

// qaAnonymizeInvitees يُجهِّل حساباتِ المدعوّين الحيّةَ عبر مسار الحذف الحقيقيّ
// (رمزٌ حقيقيٌّ ثمّ تأكيد) — فيتحرّر الرقمُ لتسجيلٍ جديد. يُرجع عددَ ما جُهِّل.
//
// **وقبل الحذفِ تُفرَّغ محفظةُ المدعوّ** (هديّةُ التسجيل) **بقيدٍ متوازنٍ إلى
// الخزينة** — فالحذفُ الحقيقيُّ يرفض محفظةً غيرَ فارغة (`wallet_not_empty`).
// **ولا يمسّ إلّا أرقامَ المدعوّين الخمسةَ الجديدة** (٩٨٠–٩٨٤)، لا حسابَ آخر.
func (s *Server) qaAnonymizeInvitees(ctx context.Context, ip, actor string) int {
	n := 0
	for _, p := range qaRefInviteePhonesUI() {
		norm, ok := identity.NormalizePhone(p)
		if !ok {
			continue
		}
		var uid string
		if err := s.pg.QueryRow(ctx, `SELECT id::text FROM users WHERE phone = $1`, norm).Scan(&uid); err != nil {
			continue // لا حساب — لا شيء
		}
		// **إفراغُ الرصيدِ بقيدٍ متوازن** — يُخصَم من المدعوّ ويُردّ إلى الخزينة،
		// فيبقى صافي الدفتر صفراً، وتصير المحفظةُ فارغةً فتُقبَل الأنسنة.
		if bal, err := s.wallet.Balance(ctx, uid); err == nil && bal > 0 {
			_, _ = s.qaReferralReverseWallet(ctx, uid, bal, actor)
		}
		code, err := s.identity.QAIssueDeleteCode(ctx, norm)
		if err != nil {
			continue
		}
		if err := s.identity.ConfirmAccountDeletion(ctx, uid, code, ip); err == nil {
			n++
		}
	}
	return n
}
