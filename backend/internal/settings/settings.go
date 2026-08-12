// Package settings مخزن الإعدادات الديناميكية المركزية — تُدار من لوحة الأدمن
// وتُقرأ لحظياً؛ لا تتطلب أي نشر جديد (GROUND-RULES §1.3).
package settings

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store { return &Store{db: db} }

// errNoStore يُعاد حين لا مخزن — **ويُبتلع في القرّاء الآمنين** فيعيدون
// افتراضاتهم. **ولا يُصدَّر**: ليس خطأً يُعالَج بل حالٌ تُقرأ.
var errNoStore = errors.New("settings: لا مخزن")

// Get يقرأ قيمةً خاماً ويفكّها في `out`.
//
// # ومخزنٌ فارغٌ يُقرأ كغيابِ قيمة لا كانهيار
//
// **`*Store` فارغٌ حالٌ مشروعةٌ في هذا المشروع**: خدماتٌ تُبنى قبل حقن المخزن،
// واختباراتٌ تمرّره `nil` عمداً لتفحص السلوك الافتراضيّ. **وكلُّ نداءٍ يتحقّق
// بنفسه شرطٌ يُنسى مرّةً فينهار الخادم.**
//
// **وقد وقع فعلاً**: `pricing.RuleFrom` تأخذ واجهةً، **ومؤشّرٌ فارغٌ داخل واجهة
// ليس واجهةً فارغة** — فمرّ فحصُ `st == nil` ثمّ انهار عند `s.db`. **وشرطٌ
// يبدو صحيحاً وهو لا يُمسك شيئاً أخطرُ من غياب الشرط.**
//
// **فالحارسُ هنا يحرس كلَّ النداءات** — لا نداءً واحداً.
func (s *Store) Get(ctx context.Context, key string, out any) error {
	if s == nil || s.db == nil {
		return errNoStore
	}
	var raw []byte
	err := s.db.QueryRow(ctx, `SELECT value FROM app_settings WHERE key = $1`, key).Scan(&raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

// GetString يقرأ قيمةً نصيةً ويعيد افتراضَ الفهرس عند غيابها — لا يفشل أبداً.
//
// # ولا يأخذ احتياطياً من المنادي
//
// كان توقيعُها `GetString(ctx, key, fallback)`، **فكتب كلُّ منادٍ افتراضَه
// بيده**: «percent» في التسعير، و«queue» في التوزيع، و«manual» في الحظر —
// **والفهرسُ يحمل الافتراضَ نفسَه في موضعٍ آخر.**
//
// **ورقمان لمعنًى واحدٍ يفترقان**: يُغيَّر افتراضُ الفهرس فتبقى الشيفرةُ على
// القديم، **أو يُنادى المفتاحُ من موضعين باحتياطيّين مختلفين** فيعمل النظامُ
// بقيمتين لمفتاحٍ واحدٍ بحسب من سأل. **ولا يظهر ذلك في أيّ خطأ.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «الأرقامُ تصدر من مكانٍ مركزيٍّ واحدٍ وليس من
// أماكنَ متفرّقة».)
//
// **ومفتاحٌ ليس في الفهرس يعيد فراغاً** — وهو ما يجب: المفتاحُ المجهول لا
// افتراضَ له، **واختراعُ قيمةٍ له يُخفي خطأً مطبعيّاً.**
func (s *Store) GetString(ctx context.Context, key string) string {
	var v string
	if err := s.Get(ctx, key, &v); err == nil && v != "" {
		return v
	}
	if def, ok := Lookup(key); ok {
		if str, ok := def.Default.(string); ok {
			return str
		}
	}
	return ""
}

// GetInt يقرأ عدداً ويعيد افتراضي الكتالوج عند غيابه أو فساده — لا يفشل أبداً.
//
// كان كل موضعٍ يحتاج رقماً يكتب استعلامه بنفسه مع `COALESCE(..., 500000)`،
// فصار الافتراضي مكتوباً في خمسة أماكن. وتغييرُ واحدٍ منها يترك الأربعة
// تعمل بالرقم القديم — وهو أسوأ من غياب الافتراضي كلّه.
func (s *Store) GetInt(ctx context.Context, key string) int64 {
	def, known := Lookup(key)
	fallback := int64(0)
	if known {
		if n, ok := toNumber(def.Default); ok {
			fallback = int64(n)
		}
	}
	var v float64
	if err := s.Get(ctx, key, &v); err != nil {
		return fallback
	}
	return int64(v)
}

// GetNum **جسرٌ إلى الحزم الدنيا يقبل العدديَّ والمنطقيَّ معاً.**
//
// (قرارُ المالك 2026-08-09: خيارُ إجبار تبديل الكلمة — **وهو منطقيّ**، تقرؤه
//
//	حزمةُ الهوية عبر جسرٍ عدديّ.)
//
// # لماذا لا `GetInt`
//
// **`GetInt` تفكّ `float64` وحدَه** — و`true` في العمود ليست رقماً، **ففكُّها
// يفشل فتردّ الاحتياطيَّ أبداً مهما ضبطه المالك.** زرٌّ يُضغط ولا يفعل شيئاً،
// **ولا خطأ ولا سجلّ.**
//
// # ولماذا هنا لا في `main`
//
// **كانت هذه الترجمةُ في `cmd/api/main.go`** — دالّةٌ مجهولةٌ داخلَ دالّةِ
// إقلاع، **لا يبلغها اختبار.** وهي بالضبط الموضعُ الذي يقع فيه الخطأ:
// نوعٌ يُفكّ إلى نوعٍ آخر.
//
// **والاحتياطيُّ من المنادي لا من الفهرس**: الحزمُ الدنيا تحمل ثوابتَها
// وتعمل بها بلا مخزن.
func (s *Store) GetNum(ctx context.Context, key string, fallback int64) int64 {
	var v any
	if err := s.Get(ctx, key, &v); err != nil {
		return fallback
	}
	return coerceNum(v, fallback)
}

// coerceNum **الترجمةُ وحدَها — مفصولةً عن القاعدة لتُختبر.**
//
// **وأوّلُ اختبارٍ لهذا كتب نسخةً من هذا الجسم في ملفّ الاختبار** — فحُذفت
// حالةُ `bool` من الأصل **ومرّ الاختبارُ ناجحاً**: كان يفحص نسختَه لا الشيفرة.
//
// **ومحاكاةٌ تُختبَر بدل أصلها حارسٌ يحرس نفسَه** — ولا مشروعَ فيه اختبارٌ
// يفتح قاعدةً، **فتُفصل الترجمةُ لتُنادى كما هي.**
func coerceNum(v any, fallback int64) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case bool:
		if t {
			return 1
		}
		return 0
	}
	return fallback
}

// GetBool مثلها للمفاتيح المنطقية.
func (s *Store) GetBool(ctx context.Context, key string) bool {
	var v bool
	if err := s.Get(ctx, key, &v); err != nil {
		if def, ok := Lookup(key); ok {
			b, _ := def.Default.(bool)
			return b
		}
		return false
	}
	return v
}

// Set يكتب قيمة بعد التحقق من الكتالوج.
//
// **والمفتاح المجهول يُرفض** ولا يُنشأ: كان `ON CONFLICT` يعني أن خطأً مطبعياً
// في اسم المفتاح يُولّد مفتاحاً جديداً لا يقرؤه أحد، ويمضي النظام بالافتراضي
// بينما يظنّ المالك أنه غيّر. صمتٌ أسوأ من خطأ.
// Set يكتب قيمةً بعد التحقّق من الفهرس.
//
// **والمفتاحُ المجهولُ يُرفض** ولا يُنشأ: كان `ON CONFLICT` يعني أنّ خطأً
// مطبعيّاً في اسم المفتاح يُولّد مفتاحاً جديداً لا يقرؤه أحد، **ويمضي النظامُ
// بالافتراضيّ بينما يظنّ المالكُ أنّه غيّر.** صمتٌ أسوأ من خطأ.
//
// **ولا حدَّ خاصٌّ هنا بعد اليوم**: كان للهامش سقفٌ يتوقّف على نمطه («٣٠٠٠»
// تعني ثلاثةَ آلاف ليرةٍ في الثابت وواحداً وثلاثين ضعفاً في النسبة) —
// **وقد ذهب النمطُ فصار الهامشُ ليرةً لا غير**، ويحرسه مدى الفهرس وحدَه.
func (s *Store) Set(ctx context.Context, key string, value any, updatedBy *string) error {
	return s.set(ctx, key, value, updatedBy)
}

func (s *Store) set(ctx context.Context, key string, value any, updatedBy *string) error {
	clean, err := Validate(key, value)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `
		INSERT INTO app_settings (key, value, updated_by) VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = now(), updated_by = EXCLUDED.updated_by`,
		key, raw, updatedBy)
	return err
}

// SetInternal يكتب بلا تحقق — للبذر والترحيلات لا للمستخدمين.
func (s *Store) SetInternal(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
		key, raw)
	return err
}

var ErrNotFound = errors.New("settings: not found")

func (s *Store) GetRaw(ctx context.Context, key string) (json.RawMessage, error) {
	var raw []byte
	err := s.db.QueryRow(ctx, `SELECT value FROM app_settings WHERE key = $1`, key).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return raw, err
}

// ══════════════════════════════════════════════════════════════════════
// **RequireWhatsApp — أيُطلب التوثيقُ من هذا الدور**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٣: مفتاحٌ واحدٌ يوقف الشرطَ عن الجميع.)
//
// **والمفتاحُ العامُّ يعلو مفتاحَ الدور**: مطفأً لا يُسأل أحد **مهما كان
// مفتاحُ دوره** — وهو معنى «كبسة زرّ».
//
// **ومن قرأ مفتاحَ الدور وحدَه** يبقي دوراً محجوباً بعد أن أُطفئ الشرطُ
// عن الجميع، **ولا يعرف صاحبُه لماذا.**
//
// **وموضعٌ واحدٌ يقرؤهما** — والقاعدةُ المكتوبةُ في ثلاثة مواضعَ تفترق.
func (s *Store) RequireWhatsApp(ctx context.Context, roleKey string) bool {
	if s == nil || !s.GetBool(ctx, "auth.require_whatsapp") {
		return false
	}
	return s.GetBool(ctx, roleKey)
}
