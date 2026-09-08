package identity

// ══════════════════════════════════════════════════════════════════════
// **إثباتُ التأكيد — إصدارٌ واستهلاك** — `ADG-3` · `AQ-1`
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا هنا
//
// **التحقّقُ من الكلمة يعيش في الهويّة** — **ولا تُكتب دلالةُ كلمةٍ
// ثانيةٌ في الخادم.** والحقبةُ (`sessions_revoked_at`) تعيش هنا أيضاً.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/servacode/rahalgo/backend/internal/auth"
)

// StepUpTTL **خمسُ دقائقَ حدّاً** — بعقد المالك.
//
// **وفعلٌ يُؤكَّد يُنفَّذ في ثانيته** — والمهلةُ سقفٌ لا وعد.
const StepUpTTL = 5 * time.Minute

var (
	// ErrStepUpPassword كلمةٌ خاطئة — **ولا إثبات.**
	ErrStepUpPassword = errors.New("step-up: كلمةٌ خاطئة")
	// ErrStepUpNoPassword حسابٌ بلا كلمة — **لا يُؤكَّد بها.**
	ErrStepUpNoPassword = errors.New("step-up: لا كلمةَ لهذا الحساب")
	// ErrStepUpUnusable **لا إثباتَ صالحٌ لهذا النداء بعينه.**
	//
	// **ولا يُفصّل السبب**: منتهٍ أم مستهلَكٌ أم لغير هذا الفعل —
	// **وتفصيلُ الرفض يعلّم المهاجمَ أين يقترب.**
	ErrStepUpUnusable = errors.New("step-up: لا إثباتَ صالحٌ لهذا الفعل")
)

// StepUpScope بصمةُ ما نوى الفاعلُ فعلَه.
//
// **والطريقةُ والمسارُ الكاملُ داخلَها** — **فحدُّ الهدف يقع من نفسِه**:
// مسارُ حسابٍ غيرُ مسارِ حسابٍ آخر. **والحقولُ الجوهريّةُ معها** —
// **فمن أكّد مبلغاً لا يكون قد أكّد غيرَه.**
func StepUpScope(method, path, material string) string {
	sum := sha256.Sum256([]byte(method + "\n" + path + "\n" + material))
	return hex.EncodeToString(sum[:])
}

// StepUpGrant ما يُردّ لصاحبه بعد أن أثبت أنّه هو.
type StepUpGrant struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	ExpiresAt time.Time `json:"expires_at"`
}

// IssueStepUp يُصدر إثباتاً لفعلٍ واحدٍ بعد التحقّق من كلمة صاحبه.
//
// # وحدّان لسباق الاعتماد
//
// **الأوّلُ عند الكتابة**: **بصمةُ الكلمة شرطٌ في `WHERE`** — فتبديلٌ
// ثُبِّت بين البصم والكتابة يمنع الإصدارَ أصلاً.
//
// **والثاني عند الاستعمال**: **الحقبةُ تُقارَن** — **فإثباتٌ نجا من
// الأوّل يموت في الثاني.** **والمقارنةُ لحظةَ الاستعمال هي الحكم** —
// وهي القاعدةُ نفسُها التي حرست `R15`.
func (s *Service) IssueStepUp(ctx context.Context, userID, sessionID,
	action, targetType, targetID, scope, password string) (*StepUpGrant, error) {

	// ══════════════════════════════════════════════════════════════
	// **ولا بصمُ كلمةٍ داخلَ معاملةٍ مفتوحة**
	// ══════════════════════════════════════════════════════════════
	//
	// **`argon2` عملٌ ثقيلٌ بالقصد** — **ومعاملةٌ تبقى مفتوحةً حولَه
	// تمسك اتّصالاً من المسبح طوالَ ذلك.**
	//
	// **وقيس أثرُه**: **نداءٌ آخرُ لم يجد اتّصالاً ففشل فحصُ جلسته
	// فردّ `503`** — **لا لأنّ الكلمةَ خاطئة.** (دورةُ ٢٨.)
	//
	// **فالبصمُ خارجَ القاعدة، والحسمُ بشرطٍ في الكتابة.**
	var hash *string
	if err := s.repo.db.QueryRow(ctx,
		`SELECT password_hash FROM users WHERE id = $1::uuid`, userID).
		Scan(&hash); err != nil {
		return nil, err
	}
	if hash == nil || *hash == "" {
		return nil, ErrStepUpNoPassword
	}
	ok, err := auth.VerifyPassword(password, *hash)
	if err != nil || !ok {
		return nil, ErrStepUpPassword
	}

	// ══════════════════════════════════════════════════════════════
	// **والكتابةُ مشروطةٌ ببقاء الاعتماد كما قُرئ**
	// ══════════════════════════════════════════════════════════════
	//
	// **فإن بُدّلت الكلمةُ بين البصم والكتابة لم يُكتب إثباتٌ أصلاً**
	// — **وهو حدٌّ أضيقُ من قفلٍ مشترَك، وبلا معاملةٍ تطول.**
	//
	// **والبصمةُ تُقارَن ولا تُخزَّن** — ولا تدخل صفَّ الإثبات.
	//
	// **والحقبةُ تُؤخَذ من الصفّ نفسِه في الكتابة نفسِها** — فلا
	// تفترق عمّا قُرئ.
	var g StepUpGrant
	if err := s.repo.db.QueryRow(ctx, `
		INSERT INTO step_up_grants
		    (user_id, session_id, action, target_type, target_id,
		     scope_hash, epoch, expires_at)
		SELECT u.id, $2::uuid, $3, $4, $5, $6,
		       u.sessions_revoked_at, now() + $8::interval
		  FROM users u
		 WHERE u.id = $1::uuid AND u.password_hash = $7
		RETURNING id::text, action, expires_at`,
		userID, sessionID, action, targetType, targetID, scope, *hash,
		StepUpTTL.String()).
		Scan(&g.ID, &g.Action, &g.ExpiresAt); err != nil {
		// **وصفرُ صفوفٍ يعني أنّ الاعتمادَ تبدّل** — ولا يُفرَّق في الردّ.
		return nil, ErrStepUpPassword
	}
	return &g, nil
}

// ClaimStepUp يستهلك إثباتاً لهذا النداء بعينه — **مرّةً واحدة.**
//
// # ونقطةُ الحسم
//
// **تحديثٌ شرطيٌّ واحدٌ يُثبَّت قبل أن يبدأ عملُ الأعمال** —
// **فنداءان متزامنان لا يظفر بالإثبات إلّا أحدُهما**، ولا يقع تبديلُ
// أعمالٍ بلا استهلاكٍ سبقه.
//
// # وسقوطُ العمل بعده لا يُعيده
//
// **بعقد المالك: الاستهلاكُ يبقى** — **وذاك السقوطُ الآمن.** **ولو
// أُعيد لَصار الإثباتُ قابلاً للاستعمال مرّتين بافتعال فشل.**
//
// # وثلاثةُ شروطٍ تُقاس هنا
//
// **الفاعلُ وجلستُه** · **وبصمةُ الفعل** · **وحقبةُ الاعتماد** —
// **والقدرةُ والجلسةُ تُقاسان في مواضعهما** (`ADG-2` و`R16`)،
// **ولا يُبنى فحصٌ ثانٍ لهما.**
func (s *Service) ClaimStepUp(ctx context.Context, userID, sessionID, scope string) error {
	var id string
	err := s.repo.db.QueryRow(ctx, `
		UPDATE step_up_grants g
		   SET consumed_at = now()
		 WHERE g.id = (
		         SELECT c.id FROM step_up_grants c
		          WHERE c.user_id    = $1::uuid
		            AND c.session_id = $2::uuid
		            AND c.scope_hash = $3
		            AND c.consumed_at IS NULL
		            AND c.expires_at > now()
		            AND c.epoch IS NOT DISTINCT FROM
		                (SELECT u.sessions_revoked_at FROM users u WHERE u.id = $1::uuid)
		          ORDER BY c.created_at
		          FOR UPDATE SKIP LOCKED
		          LIMIT 1)
		RETURNING g.id::text`, userID, sessionID, scope).Scan(&id)
	if err != nil {
		return ErrStepUpUnusable
	}
	return nil
}
