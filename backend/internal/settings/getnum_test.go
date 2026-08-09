package settings

// **الجسرُ العدديُّ يجب أن يقبل `true`.**
//
// (كُتب مع خيار إجبار تبديل الكلمة ٢٠٢٦-٠٨-٠٩ — **وهو منطقيٌّ تقرؤه حزمةُ
//
//	الهوية عبر جسرٍ عدديّ.**)
//
// # المرض الذي يمنعه
//
// **`GetInt` تفكّ `float64` وحدَه** — و`true` في العمود ليست رقماً، **ففكُّها
// يفشل فتردّ الاحتياطيَّ أبداً.** والمالكُ يضغط الزرَّ ويُحفَظ الصفُّ ولا
// يتبدّل شيء: **زرٌّ يُضغط ولا يفعل، بلا خطأٍ ولا سجلّ.**
//
// # ويُنادى الأصلُ لا نسخةٌ منه
//
// **أوّلُ صياغةٍ لهذا الاختبار نسخت جسمَ الدالّة إلى الملفّ** — فحُذفت حالةُ
// `bool` من الأصل **ومرّ الاختبارُ ناجحاً.** كان يفحص نسختَه.
//
// **فمُيّزت الترجمةُ في `coerceNum`** وتُنادى هنا كما هي: **لا قاعدةَ تُفتح،
// ولا نسخةَ تفترق.**

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCoerceNumAcceptsBoolAndNumber(t *testing.T) {
	cases := []struct {
		raw  string
		want int64
	}{
		{"true", 1},
		{"false", 0},
		{"30", 30},
		{"5.0", 5},
		{`"نص"`, 99}, // نصٌّ ليس رقماً ولا منطقيّاً — الاحتياطيّ
		{"null", 99}, // فارغٌ — الاحتياطيّ
	}
	for _, c := range cases {
		var v any
		if err := json.Unmarshal([]byte(c.raw), &v); err != nil {
			t.Fatalf("%s ليس JSON صالحاً: %v", c.raw, err)
		}
		if got := coerceNum(v, 99); got != c.want {
			t.Errorf("%s → %d، والمنتظَر %d", c.raw, got, c.want)
		}
	}
}

// TestGetNumFallsBackWithoutStore **مخزنٌ فارغٌ يردّ الاحتياطيَّ لا صفراً.**
//
// **وصفرٌ مكانَ الاحتياطيّ هو العطبُ نفسُه بوجهٍ آخر**: خدمةٌ تُبنى قبل حقن
// المخزن تعمل بصفرٍ بدل ثابتِها.
func TestGetNumFallsBackWithoutStore(t *testing.T) {
	var s *Store
	if got := s.GetNum(context.Background(), "أي.مفتاح", 42); got != 42 {
		t.Errorf("مخزنٌ فارغٌ ردّ %d، والاحتياطيُّ ٤٢", got)
	}
}
