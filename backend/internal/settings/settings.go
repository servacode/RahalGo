// Package settings مخزن الإعدادات الديناميكية المركزية — تُدار من لوحة الأدمن
// وتُقرأ لحظياً؛ لا تتطلب أي نشر جديد (GROUND-RULES §1.3).
package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
// marginCeiling حدُّ الهامش بحسب نمطه.
//
// **النسبةُ تُحدُّ بمئتين**: ضعفٌ ونصفُ الثمن سقفٌ لا يُتجاوَز في تجارةِ
// توصيل، **وما فوقه خطأُ كتابةٍ لا قرارُ تسعير.** والثابتُ يبقى بحدّ الفهرس:
// صينيةٌ بستّين ألفاً قد يُضاف عليها عشرة، **والرقمُ الكبير فيه معقول.**
const marginPercentMax = 200

func (s *Store) Set(ctx context.Context, key string, value any, updatedBy *string) error {
	// **الحدُّ الذي لا يعرفه الفهرس** — لأنه يتوقّف على قيمة مفتاحٍ آخر.
	if key == "pricing.margin_value" {
		if n, ok := toNumber(value); ok &&
			s.GetString(ctx, "pricing.margin_mode") == "percent" &&
			n > marginPercentMax {
			return ErrInvalidValue{Key: key,
				Reason: fmt.Sprintf("النسبةُ لا تتجاوز %d%%", marginPercentMax)}
		}
	}
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
