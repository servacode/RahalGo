package server

// ══════════════════════════════════════════════════════════════════════
// **تقسيمُ سوريا مبذورٌ كاملاً — ولا يُنقَص صامتاً**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-٣٠: «قائمةُ المحافظات يجب أن تضمّ كلَّ محافظات
//  سوريا، والمناطقُ تضمّ مناطقَ كلِّ محافظةٍ سوريّة، **وبهذا لا نعود
//  مرّةً أخرى لتعديل الكود**».)
//
// # ولماذا يُحرَس البذر
//
// **البذرُ يقع مرّةً في هجرةٍ لا يقرؤها أحدٌ بعدها** — ومن حذف سطراً
// منها أو أخطأ في اسم محافظةٍ **لا يسقط بناءٌ ولا اختبار**، **ويظهر
// النقصُ يومَ يفتح مندوبٌ النموذجَ في محافظةٍ لا يجدها.**
//
// **وأربعَ عشرةَ محافظةً رقمٌ لا يتغيّر** — فيُثبَّت هنا.
//
// # ولا يُحرَس عددُ المناطق بالضبط
//
// **المالكُ يملك أن يضيف ويحذف من اللوحة** — وهو الغرضُ كلُّه. **فحارسٌ
// يشترط ثلاثاً وستّين يسقط أوّلَ يومٍ يستعمل فيه صاحبُ المنصّة لوحتَه.**
//
// **إنّما يُحرَس ألّا تبقى محافظةٌ بلا منطقةٍ واحدة** — تلك ليست خياراً
// إداريّاً، **بل بذرٌ نُسي**: يختارها المندوبُ فيجد القائمةَ الثانيةَ
// فارغةً ولا يُكمل.

import (
	"context"
	"testing"
)

func TestGeoSeed_AllSyrianGovernorates(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()

	// **والأسماءُ تُفحص لا العددُ وحدَه** — أربعَ عشرةَ صفّاً بأسماءَ
	// خاطئةٍ تمرّ من عدّادٍ ولا تمرّ من قائمة.
	want := []string{
		"دمشق", "ريف دمشق", "حلب", "حمص", "حماة", "اللاذقية", "طرطوس",
		"إدلب", "الحسكة", "دير الزور", "الرقة", "درعا", "السويداء", "القنيطرة",
	}
	for _, name := range want {
		var ok bool
		if err := f.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM governorates WHERE name = $1)`, name).Scan(&ok); err != nil {
			t.Fatalf("تعذّرت القراءة: %v", err)
		}
		if !ok {
			t.Errorf("محافظةُ %q غيرُ مبذورة — **والمندوبُ لا يجدها في النموذج**", name)
		}
	}

	var n int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM governorates`).Scan(&n); err != nil {
		t.Fatalf("تعذّر العدّ: %v", err)
	}
	if n < len(want) {
		t.Errorf("المحافظاتُ %d والمطلوبُ %d على الأقلّ", n, len(want))
	}
}

// TestGeoSeed_NoGovernorateWithoutDistricts **ولا محافظةَ بلا منطقة.**
//
// **ومحافظةٌ فارغةٌ تُختار ثمّ تقف** — القائمةُ الثانيةُ لا شيءَ فيها،
// **فلا يُكمل من فتح النموذج** ولا يعرف أنّ العطبَ في البيانات لا فيه.
func TestGeoSeed_NoGovernorateWithoutDistricts(t *testing.T) {
	f := newDriverFixture(t, 0)
	rows, err := f.pool.Query(context.Background(), `
		SELECT g.name FROM governorates g
		WHERE NOT EXISTS (SELECT 1 FROM districts d WHERE d.governorate_id = g.id)`)
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("تعذّر المسح: %v", err)
		}
		t.Errorf("محافظةُ %q بلا مناطق — **تُختار ثمّ يقف من اختارها**", name)
	}
}

// TestGeoSeed_DistrictsBelongToLiveGovernorate **ولا منطقةَ يتيمة.**
//
// **والقاعدةُ تمنعها بمفتاحٍ أجنبيّ** — وهذا يفحص أنّ المفتاحَ قائمٌ
// فعلاً: **قيدٌ يُنسى في هجرةٍ لاحقةٍ لا يُسقط بناءً.**
func TestGeoSeed_DistrictsBelongToLiveGovernorate(t *testing.T) {
	f := newDriverFixture(t, 0)
	var orphans int
	if err := f.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM districts d
		WHERE NOT EXISTS (SELECT 1 FROM governorates g WHERE g.id = d.governorate_id)`).
		Scan(&orphans); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if orphans != 0 {
		t.Errorf("مناطقُ بلا محافظة: %d", orphans)
	}
}
