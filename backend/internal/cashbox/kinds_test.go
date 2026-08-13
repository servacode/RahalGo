package cashbox

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestCashKindsHaveArabicLabels **كلُّ نوعٍ يكتبه المحرّكُ له اسمٌ في المعجم.**
//
// (شهده المالك ٢٠٢٦-٠٨-٠٧ في شاشة صندوق السائق: `order_collection` و
// `settlement` معروضتان خامّتين على سائق.)
//
// # ولماذا وقع
//
// **المحرّكُ يكتب `order_collection` و`settlement`، والمعجمُ يسمّيهما
// `collect` و`settle`.** واسمان لم يتطابقا قطّ: البحثُ يفشل دائماً،
// **والشيفرةُ مكتوبةٌ لتسقط على الرمز الخام عند الفشل** — فلا خطأ ولا صمت،
// **إنّما إنكليزيّةٌ على شاشة سائقٍ في الرقّة.**
//
// **ولا يمسكه مترجمٌ ولا حارسُ أصناف**: النوعان نصّان في طرفين لا يعرف
// أحدُهما الآخر.
//
// # فيُقرأ الطرفان ويُقارَنان
//
// **تُستخرج الأنواعُ من `cashbox.go` نفسِه** — لا من قائمةٍ تُصان بيدٍ
// فتشيخ — **ويُفتَّش عنها في `ar.json`.**
func TestCashKindsHaveArabicLabels(t *testing.T) {
	src, err := os.ReadFile("cashbox.go")
	if err != nil {
		t.Fatalf("تعذّر قراءة cashbox.go: %v", err)
	}
	// s.apply(ctx, driverID, amount, "<النوع>", ref, note, actorID)
	re := regexp.MustCompile(`apply\([^)]*?"([a-z_]+)"`)
	found := map[string]bool{}
	for _, mm := range re.FindAllStringSubmatch(string(src), -1) {
		found[mm[1]] = true
	}
	if len(found) == 0 {
		t.Fatal("لم أجد نوعاً واحداً في cashbox.go — تبدّل شكلُ النداء والحارسُ صار أعمى")
	}

	raw, err := os.ReadFile("../../../web/packages/i18n/src/locales/ar.json")
	if err != nil {
		t.Skipf("المعجمُ غيرُ متاح: %v", err)
	}
	var dict map[string]any
	if err := json.Unmarshal(raw, &dict); err != nil {
		t.Fatalf("المعجمُ غيرُ صالح: %v", err)
	}
	labels := dig(dict, "driver", "cashbox", "kinds")
	if labels == nil {
		t.Fatal("لا driver.cashbox.kinds في المعجم")
	}

	for kind := range found {
		s, ok := labels[kind]
		if !ok {
			t.Errorf("النوع %q بلا اسمٍ في driver.cashbox.kinds — يُعرض خامّاً على السائق", kind)
			continue
		}
		// **ولا حرفَ لاتينيٍّ في الاسم** — «settlement تسليم» يمرّ الوجودَ
		// ويبقى غيرَ مفهوم.
		if strings.ContainsAny(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			t.Errorf("اسمُ النوع %q هو %q وفيه حرفٌ لاتينيّ", kind, s)
		}
	}

	// **والزائدُ يُقال كذلك** — اسمٌ لنوعٍ لا يكتبه أحدٌ يُصان بلا فائدة،
	// **ويوهم أنّ الحالةَ مغطّاة.**
	for name := range labels {
		if !found[name] {
			t.Errorf("المعجمُ يسمّي %q ولا يكتبه المحرّك — اسمٌ ميّت", name)
		}
	}
}

func dig(m map[string]any, path ...string) map[string]string {
	cur := m
	for i, k := range path {
		v, ok := cur[k]
		if !ok {
			return nil
		}
		next, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		if i == len(path)-1 {
			out := map[string]string{}
			for kk, vv := range next {
				if s, ok := vv.(string); ok {
					out[kk] = s
				}
			}
			return out
		}
		cur = next
	}
	return nil
}

// TestCashKindsHaveAppLabels **والتطبيقُ يسمّيها كما يسمّيها الموقع.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٣: «يجب أن توحّد الويبَ بنفس الطريقة المتّبعة
//
//	بالتطبيق… نفس النموذج والتسميات والشكل والأفعال والأسماء».)
//
// **وشاشةُ «صندوقي» بُنيت في التطبيق** فصار للنوع موضعان يُسمّى فيهما.
// **والحارسُ الذي يحرس واحداً منهما يترك الآخر يشيخ** — فيُعرض
// `order_collection` خامّاً على سائقٍ فتح التطبيق، **وهو عينُ ما وقع في
// الموقع سنةَ ٢٠٢٦-٠٨-٠٧.**
//
// **والمفتاحُ في أندرويد `cash_k_<النوع>`** — اصطلاحٌ يُقرأ من الطرفين.
func TestCashKindsHaveAppLabels(t *testing.T) {
	src, err := os.ReadFile("cashbox.go")
	if err != nil {
		t.Fatalf("تعذّر قراءة cashbox.go: %v", err)
	}
	re := regexp.MustCompile(`apply\([^)]*?"([a-z_]+)"`)
	found := map[string]bool{}
	for _, mm := range re.FindAllStringSubmatch(string(src), -1) {
		found[mm[1]] = true
	}
	if len(found) == 0 {
		t.Fatal("لم أجد نوعاً واحداً في cashbox.go — تبدّل شكلُ النداء والحارسُ صار أعمى")
	}

	raw, err := os.ReadFile("../../../mobile/app-driver/src/main/res/values/strings.xml")
	if err != nil {
		t.Skipf("نصوصُ التطبيق غيرُ متاحة: %v", err)
	}
	text := string(raw)
	for kind := range found {
		key := "cash_k_" + kind
		if !strings.Contains(text, `name="`+key+`"`) {
			t.Errorf("النوع %q بلا نصٍّ في التطبيق (%s) — يُعرض خامّاً على السائق", kind, key)
		}
	}

	// **والزائدُ يُقال كذلك** — مفتاحٌ لنوعٍ لا يكتبه المحرّك اسمٌ ميّت.
	keyRe := regexp.MustCompile(`name="cash_k_([a-z_]+)"`)
	for _, mm := range keyRe.FindAllStringSubmatch(text, -1) {
		if !found[mm[1]] {
			t.Errorf("التطبيقُ يسمّي %q ولا يكتبه المحرّك — اسمٌ ميّت", mm[1])
		}
	}
}
