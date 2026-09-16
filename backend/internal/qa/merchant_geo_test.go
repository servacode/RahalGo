package qa

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **جغرافيا المتجر — ومتجرٌ بلا مدينةٍ لا يراه أحد** (`GEO`، ٢٠٢٦-٠٩-١٦)
// ══════════════════════════════════════════════════════════════════════
//
// # العطبُ المقيس على جهازٍ حقيقيّ
//
// **ووقف الجهازُ داخلَ الرقّة — ألفَ مترٍ من مركزها — ولم يرَ صنفاً
// واحداً.** **وفي القاعدة ستّةٌ وثلاثون صنفاً.**
//
// **والسلسلة**: **التطبيقُ يرسل موضعَه ⇒ يشتغل مرشّحُ المدينة ⇒ شرطُه
// `m.city_id = (أقربُ مدينةٍ تحوي النقطة)` ⇒ ومدينةُ المتجر `NULL`**
// — **و`NULL = x` ليست صحيحةً في SQL ولا تكون** — **فيسقط كلُّ متجر.**
//
// **والردُّ `200` بقائمةٍ فارغة** — **فلا خطأَ يُرى ولا سطرَ في سجلّ.**
//
// # ولماذا عاد وقد أُصلح
//
// **أُصلح الإنشاءُ في ٢٠٢٦-٠٨-٢٢** بعد شكوى المالك «الأصنافُ لم تظهر
// بالتطبيق أبداً» — **وأُصلح موضعُ الإنشاء وحدَه.**
//
// **وبقيت ثلاثةُ مواضعَ تكتب جغرافيا المتجر ولا تشتقّ المدينة**:
// **التحديثُ وبذرتان.** **فنسخةٌ تُصلَح وثلاثٌ تبقى.**
//
// **وهذا الحارسُ يحرس المواضعَ كلَّها لا الموضعَ الذي انكسر.**

// geoSource **يقرأ ملفَّ مصدرٍ من جذر المستودع** — بنهاياتٍ موحَّدة.
func geoSource(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ %s: %v", rel, err)
	}
	// **وgit على ويندوز يكتب CRLF** — وحارسٌ يطابق LF يحمرّ باطلاً.
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

// TestGEO1_EveryMerchantWritePathDerivesCity **كلُّ من يكتب متجراً
// يشتقّ مدينتَه** — **ولا يُترَك العمودُ فارغاً.**
func TestGEO1_EveryMerchantWritePathDerivesCity(t *testing.T) {
	// **والمواضعُ بأعيانها** — **ومن أضاف مساراً رابعاً يضيفه هنا،
	// وإلّا بقي حارساً لثلاثة.**
	paths := []string{
		"backend/internal/catalog/catalog.go",
		"backend/cmd/seed/main.go",
		"backend/cmd/seed/store.go",
		"backend/internal/qa/harness.go",
	}
	for _, p := range paths {
		src := geoSource(t, p)
		// **وكلُّ `INSERT INTO merchants` يذكر العمود.**
		for i, part := range strings.Split(src, "INSERT INTO merchants")[1:] {
			head := part
			if j := strings.Index(head, "RETURNING"); j > 0 {
				head = head[:j]
			}
			if !strings.Contains(head, "city_id") {
				t.Errorf("**%s: إدراجُ متجرٍ رقم %d بلا `city_id`** — "+
					"**ومتجرٌ بلا مدينةٍ لا يظهر لزبونٍ يرسل موقعَه.**", p, i+1)
			}
		}
		// **ولا اشتقاقَ مكتوبٌ باليد** — **التعبيرُ واحدٌ أو لا يكون.**
		if strings.Contains(src, "SELECT c.id FROM cities c") &&
			!strings.Contains(src, "CityOfPointSQL") {
			t.Errorf("**%s: اشتقاقُ مدينةٍ مكتوبٌ باليد** — "+
				"**ونسختان بمعنيين تضعان المتجرَ في مدينةٍ والزبونَ في أخرى.**", p)
		}
	}
}

// TestGEO2_LocationUpdateRecomputesCity **ومن نقل متجرَه نُقلت مدينتُه.**
//
// **وكان الموقعُ يُبدَّل والمدينةُ تبقى** — **فمتجرٌ انتقل يظلّ منسوباً
// إلى مدينته الأولى**: **يراه من لا يصله، ولا يراه من يجاوره.**
func TestGEO2_LocationUpdateRecomputesCity(t *testing.T) {
	src := geoSource(t, "backend/internal/catalog/catalog.go")
	i := strings.Index(src, "UPDATE merchants SET")
	if i < 0 {
		t.Fatal("**ذهبت جملةُ تحديث المتجر**")
	}
	j := strings.Index(src[i:], "WHERE")
	if j < 0 {
		t.Fatal("**لم أجد نهايةَ جملة التحديث**")
	}
	stmt := src[i : i+j]
	if !strings.Contains(stmt, "location") {
		t.Fatal("**جملةُ التحديث لا تمسّ الموقعَ أصلاً** — تبدّل العقدُ")
	}
	if !strings.Contains(stmt, "city_id") {
		t.Error("**تحديثُ المتجر يبدّل الموقعَ ولا يُعيد حسابَ المدينة** — " +
			"**فتشيخ النسبةُ ويختفي المتجرُ عن جيرانه.**")
	}
	if !strings.Contains(stmt, "CityOfPointSQL") {
		t.Error("**التحديثُ لا يستعمل التعبيرَ المركزيّ** — " +
			"**واشتقاقان بمعنيين أسوأُ من واحدٍ ناقص.**")
	}
}

// TestGEO3_CityExpressionMatchesBrowseFilter **واشتقاقُ الكتابة هو
// شرطُ القراءة** — **وإلّا كُتب متجرٌ في مدينةٍ ويُبحَث عنه في أخرى.**
func TestGEO3_CityExpressionMatchesBrowseFilter(t *testing.T) {
	write := geoSource(t, "backend/internal/catalog/citysql.go")
	read := geoSource(t, "backend/internal/server/city_filter.go")
	// **والعناصرُ الثلاثةُ التي تُعرّف «مدينةُ هذه النقطة».**
	for _, k := range []string{"c.active", "ST_DWithin", "ST_Distance", "LIMIT 1"} {
		if !strings.Contains(write, k) {
			t.Errorf("**تعبيرُ الكتابة لا يذكر %s**", k)
		}
		if !strings.Contains(read, k) {
			t.Errorf("**شرطُ القراءة لا يذكر %s**", k)
		}
	}
	// **ونصفُ القطر هو الحدّ في الاثنين** — لا حدٌّ ثانٍ يُخترَع.
	if !strings.Contains(write, "c.radius_m") || !strings.Contains(read, "c.radius_m") {
		t.Error("**حدُّ المدينة ليس `radius_m` في الطرفين** — " +
			"**فيُكتب المتجرُ بحدٍّ ويُقرأ بحدٍّ آخر.**")
	}
}
