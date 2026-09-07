package server

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/dbtx"

	"github.com/servacode/rahalgo/backend/internal/fininv"
)

// ══════════════════════════════════════════════════════════════════════
// **أيُّ إعدادٍ «حسّاس»؟** — `XG-20` · `AQ-4`
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا لا تُعَدّ كلُّها
//
// **المعجمُ فيه أكثرُ من مئة مفتاح** — **ونصُّ صورةٍ في صفحةٍ ليس
// فعلاً أمنيّاً.** **ومن جعل كلَّ إعدادٍ معاملةً أمنيّةً أسقط تبديلَ
// عنوانٍ لأنّ سطرَ تدقيقٍ تعثّر.**
//
// # وأيُّها حسّاس — بنصّ `AQ-4`
//
// **«العمولات · التسعيرَ والهوامش»** — **وهذه مقيسةٌ سلفاً**:
// `fininv.FinancialSettings` معجمٌ **مقروءٌ من مواضع القراءة نفسِها**
// في `pricing` و`cashbox` و`server`، **وحارسٌ يطابقه بمعجم الإعدادات
// فيسقط إن اختفى مفتاح.**
//
// **فلا يُخترَع تصنيفٌ ثانٍ** — **وتصنيفان يفترقان يومَ يُضاف مفتاح.**
//
// # ويُضاف الأمنُ إليها
//
// **`security.*` تحكم مهلةَ الجلسة وطولَ الكلمة وسقفَ المحاولات** —
// **ومن بدّلها بدّل حدودَ الدخول إلى المنصّة كلِّها.**
//
// # وما بقي يبقى أفضلَ جهد
//
// **لافتةٌ أو تصنيفٌ أو نصُّ صفحةٍ سقط سطرُه لا يُساوي إسقاطَ العملية.**

// criticalSettingKey **أهذا المفتاحُ من الصنف `A`؟**
func criticalSettingKey(key string) bool {
	if strings.HasPrefix(key, "security.") {
		return true
	}
	for _, k := range fininv.FinancialSettings {
		if k == key {
			return true
		}
	}
	return false
}

// redactSettingValue **قيمةٌ سرّيّةٌ لا تُكتب في سجلّ يُقرأ.**
//
// **ولا مفتاحَ سرٍّ في المعجم اليوم** — **والحارسُ موضوعٌ سلفاً**:
// **من أضاف مفتاحاً باسمٍ يقول سرّاً كُتبت قيمتُه محجوبةً**، ولا
// يُنتظَر أن يتذكّر أحد.
func redactSettingValue(key string, raw []byte) any {
	lower := strings.ToLower(key)
	for _, mark := range []string{"secret", "password", "token", "key", "api_key", "webhook"} {
		if strings.Contains(lower, mark) {
			return "«محجوبٌ عمداً»"
		}
	}
	return jsonRaw(raw)
}

// jsonRaw يمرّر القيمةَ كما هي إلى `details`.
func jsonRaw(raw []byte) any {
	if len(raw) == 0 {
		return json.RawMessage("null")
	}
	return json.RawMessage(raw)
}

// inTx **معاملةٌ للفعل وأثرِه** — `XG-20`.
//
// **ولا نداءَ خارجيٌّ داخلَها**: **الدفعُ والبثُّ والواتساب بعد
// التثبيت** — **قفلٌ ينتظر شبكةً قفلٌ ينتظر الأبد.**
func (s *Server) inTx(ctx context.Context, do func(context.Context, dbtx.Querier) error) error {
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := do(ctx, tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
