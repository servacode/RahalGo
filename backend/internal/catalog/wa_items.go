package catalog

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// ══════════════════════════════════════════════════════════════════════
// **أصناف الواتساب** (قرار المالك 2026-11-09) — ما يحتاجه بوت «صورة + اسم وسعر» من الكتالوج
// ══════════════════════════════════════════════════════════════════════
//
// البوت لا يملك مسار إدراجٍ ثانياً: يستعمل `CreateItemIn` نفسه. وهنا فقط:
//   - توحيد الاسم (للمكرّر) — نفس `normName` اللي يعتمدها الأمين.
//   - قسم الصنف من اسمه — نفس مصنّف الأمين (`SuggestSection`)، وإلا قسم الانتظار.

// NormName **الاسم موحَّداً** (همزات، تاء مربوطة، فراغات) — لكشف المكرّر.
func NormName(s string) string { return normName(s) }

// SectionByName **قسم السوق لاسم صنف** بنفس قاعدة الأمين: مقترحٌ أكيد من الاسم ⇒ `name`،
// وإلا قسم الانتظار ⇒ `pending` (غير فعّال، ويرتّبه المكتب). يردّ معرّف القسم واسمه ومصدره.
func SectionByName(ctx context.Context, q dbtx.Querier, itemName string) (id, name, source string, err error) {
	r := newSectionResolver(q)
	id, err = r.suggest(ctx, itemName)
	if err != nil {
		return "", "", "", err
	}
	if id != "" {
		return id, r.names[id], SourceName, nil
	}
	id, err = ensurePendingSection(ctx, q)
	if err != nil {
		return "", "", "", err
	}
	return id, PendingSectionName, SourcePending, nil
}
