package auth

import "strings"

// ══════════════════════════════════════════════════════════════════════
// **الكلمةُ كما كتبها صاحبُها لا كما ضبطها الكيبورد** (بلاغُ المالك ٢٠٢٦-١٠-١٩)
// ══════════════════════════════════════════════════════════════════════
//
// **«عم يعطيني كلمة السر خطأ وأنا متأكّد إنها صح»** — والكلمةُ لم تتبدّل منذ
// أُنشئ الحساب. **الكيبوردُ العربيّ يكتب ١٢٣ لا 123**، ويزيد فراغاً بعد الكلمة،
// **والمقارنةُ حرفاً بحرف** فتُرفض كلمةٌ صحيحة. فتُجرَّب صيغُها: كما هي، ثمّ
// بلا فراغٍ على الطرفين، ثمّ أرقامُها لاتينيّةً، ثمّ عربيّة.

var (
	toLatinDigits = strings.NewReplacer(
		"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4",
		"٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
		"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4",
		"۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
	)
	toArabicDigits = strings.NewReplacer(
		"0", "٠", "1", "١", "2", "٢", "3", "٣", "4", "٤",
		"5", "٥", "6", "٦", "7", "٧", "8", "٨", "9", "٩",
	)
)

// PasswordVariants **صيغُ الكلمة المقبولة** — الأولى كما كُتبت، وبلا تكرار.
func PasswordVariants(password string) []string {
	out := []string{password}
	add := func(v string) {
		if v == "" {
			return
		}
		for _, o := range out {
			if o == v {
				return
			}
		}
		out = append(out, v)
	}
	trimmed := strings.TrimSpace(password)
	add(trimmed)
	add(toLatinDigits.Replace(trimmed))
	add(toArabicDigits.Replace(toLatinDigits.Replace(trimmed)))
	return out
}

// VerifyPasswordTyped **كـ`VerifyPassword` ويقبل صيغَ الكيبورد** — أرقاماً عربيّةً
// أو لاتينيّة وفراغاً زائداً.
func VerifyPasswordTyped(password, encoded string) (bool, error) {
	for _, v := range PasswordVariants(password) {
		ok, err := VerifyPassword(v, encoded)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}
