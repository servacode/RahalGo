package qa

// ══════════════════════════════════════════════════════════════════════
// **إثباتُ تأكيدٍ يُصنَع للفحص الذي يقيس عقداً آخر** — `ADG-3`
// ══════════════════════════════════════════════════════════════════════
//
// # المسألة
//
// **صار كلُّ فعلٍ شديدٍ يلزمه تأكيدٌ بكلمةِ صاحبه** — **وواحدٌ
// وخمسون فحصاً تقيس عقوداً أخرى تمرّ بأفعالٍ شديدة**: قيدُ محفظة ·
// منحُ دور · حظرُ حساب · قرارُ سحب.
//
// **ولو أُضيف نموذجُ تأكيدٍ في كلٍّ منها لَصار الفحصُ يقيس `ADG-3`
// بدل ما بُني ليقيسه** — **ولَتعلّم من يكتبه أن يتجاهل الحدَّ.**
//
// # فيُصنَع الإثباتُ عند الباب
//
// **والمِسنَدُ يمرّر إثباتاً صالحاً لكلّ نداءٍ يحتاجه** — **إلّا حين
// يُمرَّر واحدٌ صراحةً**: فحوصُ `ADG-3` تُرسل ما تريد قياسَه.
//
// # ولا يُضعِف هذا الحدَّ
//
// **الحدُّ في الخادم يُقاس بفحوصه هو** — نداءٌ بلا إثباتٍ يُردّ،
// وبإثباتٍ لغيره يُردّ، وبإثباتٍ مستهلَكٍ يُردّ. **وهذه يسرُ فحصٍ
// لا ثغرةُ منتج**: **لا تعمل إلّا بامتلاك القاعدة.**

import (
	"encoding/json"

	"github.com/golang-jwt/jwt/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/identity"
)

// mintStepUp يكتب إثباتاً صالحاً لهذا النداء ويردّ معرّفَه.
//
// **وفارغٌ يعني «لا يلزم»** — أو أنّ التوكن ليس لحسابٍ بجلسة.
func (h *Harness) mintStepUp(method, path, token string, body []byte) string {
	act, ok := authz.LookupSensitive(method, authz.AdminPattern(path))
	if !ok {
		return ""
	}
	userID, sid := claimsOf(token)
	if userID == "" || sid == "" {
		return ""
	}
	scope := identity.StepUpScope(method, path, authz.Material(act, body))
	var id string
	// **والحقبةُ تُؤخَذ من صفّ الحساب كما يفعل المنتج** — فلا يمرّ
	// إثباتٌ بحقبةٍ لا تطابق.
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO step_up_grants
		    (user_id, session_id, action, target_type, target_id,
		     scope_hash, epoch, expires_at)
		SELECT u.id, $2::uuid, $3, $4, $5, $6,
		       u.sessions_revoked_at, now() + interval '5 minutes'
		  FROM users u WHERE u.id = $1::uuid
		RETURNING id::text`,
		userID, sid, act.Action, act.TargetType, authz.Target(act, path), scope).
		Scan(&id); err != nil {
		return ""
	}
	return id
}

// claimsOf يقرأ صاحبَ التوكن وعائلةَ جلسته — **بلا تحقّقِ توقيع**:
// **المِسنَدُ هو من أصدره.**
func claimsOf(token string) (userID, sid string) {
	if token == "" {
		return "", ""
	}
	var claims jwt.MapClaims
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	if _, _, err := parser.ParseUnverified(token, &claims); err != nil {
		return "", ""
	}
	sub, _ := claims["sub"].(string)
	s, _ := claims["sid"].(string)
	return sub, s
}

// bodyBytes يحوّل جسمَ النداء إلى بايتاتٍ كما يُرسَل.
func bodyBytes(body any) []byte {
	if body == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(body)
	if err != nil {
		return []byte("{}")
	}
	return b
}
