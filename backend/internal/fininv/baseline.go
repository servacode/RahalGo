package fininv

import (
	"context"
	"fmt"
)

// Baseline صورةُ الخروق القائمةِ قبل السيناريو.
//
// # لماذا تلزم
//
// **ثوابتُ المال عامّةٌ بطبعها**: «لا محفظةَ سالبةٌ في القاعدة» لا
// «في هذا السيناريو». **وهذا هو الصواب على قاعدة تشغيل** — وهو ما يشغّله
// `moneycheck`.
//
// **لكنّ قاعدةَ الاختبار مشتركة**: عشراتُ السيناريوهات تكتب فيها، وبعضُها
// **يُفسد عمداً**، وبعضُها ينظّف مستخدماً **فيجرّ حذفُه قيودَه بالتتالي**
// فيبقى طلبٌ مسلَّمٌ بلا مستحقِّ متجر. **فيسقط سيناريو بذنبِ غيره.**
//
// **والحلُّ ليس إضعافَ الاستعلام** — استعلامٌ يُضيَّق ليمرّ لا يحرس شيئاً.
// **بل قياسُ الفرق**: يُصوَّر ما هو مخروقٌ قبل، فما ظهر بعدُ **هو ما
// أحدثه هذا السيناريو وحدَه.**
type Baseline map[string]map[string]bool

// Capture يصوّر الخروق القائمة.
func Capture(ctx context.Context, q Querier, ids ...string) (Baseline, error) {
	b := Baseline{}
	vs, err := Run(ctx, q, ids...)
	if err != nil {
		return nil, err
	}
	for _, v := range vs {
		rows := map[string]bool{}
		for _, r := range v.Rows {
			rows[fingerprint(r)] = true
		}
		b[v.Check.ID] = rows
	}
	return b, nil
}

// New ما استجدّ من الخروق بعد الصورة — **وهو وحدَه ذنبُ السيناريو.**
func (b Baseline) New(vs []Violation) []Violation {
	var out []Violation
	for _, v := range vs {
		known := b[v.Check.ID]
		fresh := Violation{Check: v.Check, Cols: v.Cols}
		for _, r := range v.Rows {
			if !known[fingerprint(r)] {
				fresh.Rows = append(fresh.Rows, r)
			}
		}
		if len(fresh.Rows) > 0 {
			out = append(out, fresh)
		}
	}
	return out
}

func fingerprint(row []any) string { return fmt.Sprintf("%v", row) }
