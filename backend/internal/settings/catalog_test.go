package settings

// اختبارات كتالوج الإعدادات.
//
// وهي أرخص اختبارات في المشروع وأعلاها مردوداً: لا قاعدة ولا شبكة، ومع ذلك
// تحرس الشاشة التي تحكم المال كلَّه. فقبلها كان `drivers.share_percent = 200`
// يُقبل بلا سؤال، فتدفع المنصة ضعف رسم التوصيل لكل سائق في كل طلب — صامتةً.

import "testing"

// TestValidate_RejectsUnknownKey المفتاح المجهول يُرفض ولا يُنشأ.
//
// كان `INSERT ... ON CONFLICT` يعني أن خطأً مطبعياً يُولّد مفتاحاً جديداً لا
// يقرؤه أحد: يمضي النظام بالافتراضي والمالك يظنّ أنه غيّر. **صمتٌ أسوأ من خطأ.**
func TestValidate_RejectsUnknownKey(t *testing.T) {
	if _, err := Validate("drivers.share_percen", 70.0); err == nil {
		t.Fatal("قُبل مفتاح مجهول (خطأ مطبعيّ في اسم مفتاح قائم)")
	}
	// والمفتاح الصحيح يمرّ — كي لا يمرّ الاختبار لأن كل شيء مرفوض
	if _, err := Validate("drivers.share_percent", 70.0); err != nil {
		t.Fatalf("رُفض مفتاح صحيح: %v", err)
	}
}

// TestValidate_RejectsOutOfRange القيمة خارج المدى تُرفض.
func TestValidate_RejectsOutOfRange(t *testing.T) {
	cases := []struct {
		key string
		val float64
		why string
	}{
		{"drivers.share_percent", 200, "نسبةٌ فوق المئة تجعل المنصة تدفع أكثر ممّا قبضت"},
		{"drivers.share_percent", -10, "أجرٌ سالب يخصم من محفظة السائق عند كل تسليم"},
		{"drivers.share_fixed", -5000, "مبلغٌ مقطوع سالب"},
		{"merchants.default_commission_percent", 150, "عمولةٌ تفوق قيمة البضاعة"},
		{"sales.commission_percent", -1, "عمولةٌ سالبة"},
		{"orders.accept_timeout_min", 0, "مهلةٌ صفر تُنذر كل طلبٍ لحظة ميلاده"},
		{"drivers.max_active_orders", 0, "صفرٌ يمنع كل سائق من أخذ أيّ طلب"},
		{"security.password_min_length", 3, "كلمة مرور من ثلاثة أحرف"},
	}
	for _, c := range cases {
		if _, err := Validate(c.key, c.val); err == nil {
			t.Errorf("قُبل %s = %v — %s", c.key, c.val, c.why)
		}
	}
}

// TestValidate_RejectsWrongType النوع الخاطئ يُرفض.
//
// وهذا أخطرها أثراً: `(value#>>'{}')::float8` على نصٍّ يرمي خطأً **داخل معاملة
// التسليم**. فحرفٌ في مربّع نصٍّ كان يكسر خطّ التوصيل كلَّه — لا شاشةً واحدة.
func TestValidate_RejectsWrongType(t *testing.T) {
	if _, err := Validate("drivers.cash_limit", "خمسمئة ألف"); err == nil {
		t.Fatal("قُبل نصٌّ في مفتاح رقميّ — يكسر معاملة التسليم عند أول طلب")
	}
	if _, err := Validate("drivers.share_percent", 70.5); err == nil {
		t.Fatal("قُبل كسرٌ في مفتاح يُقرأ عدداً صحيحاً")
	}
	if _, err := Validate("merchants.menu_requires_approval", 1.0); err == nil {
		t.Fatal("قُبل رقمٌ في مفتاح منطقيّ")
	}
	if _, err := Validate("platform.invite_code", 12345.0); err == nil {
		t.Fatal("قُبل رقمٌ في مفتاح نصّي")
	}
}

// TestValidate_RejectsBadChoice الخيار خارج القائمة يُرفض.
//
// `drivers.share_mode = "banana"` كان يمرّ إلى `switch` الذي `default` فيه
// «نسبة» — فيُحتسب الأجر بطريقةٍ لم يخترها أحد.
func TestValidate_RejectsBadChoice(t *testing.T) {
	if _, err := Validate("drivers.share_mode", "banana"); err == nil {
		t.Fatal("قُبل خيارٌ ليس في القائمة")
	}
	for _, ok := range []string{"percent", "fixed"} {
		if _, err := Validate("drivers.share_mode", ok); err != nil {
			t.Errorf("رُفض خيارٌ صحيح %q: %v", ok, err)
		}
	}
}

// TestValidate_NormalizesInt JSON لا يفرّق بين ٧٠ و٧٠٫٠ — واللوحة تفرّق.
func TestValidate_NormalizesInt(t *testing.T) {
	v, err := Validate("drivers.share_percent", float64(70))
	if err != nil {
		t.Fatalf("رُفضت قيمة صحيحة: %v", err)
	}
	if _, ok := v.(int64); !ok {
		t.Fatalf("لم تُطبَّع إلى عدد صحيح: %T", v)
	}
}

// TestValidate_TextLengthInRunes الطول بالمحارف لا بالبايتات.
//
// الحرف العربي بايتان، فقياسُه بالبايت يقصّ نصّاً عربياً عند نصف ما يقصّ عنده
// نصّاً لاتينياً. وقالب رسالة التحقّق عربيٌّ كلُّه.
func TestValidate_TextLengthInRunes(t *testing.T) {
	// ٢٠٠ حرف عربي = ٤٠٠ بايت، وحدّ القالب ٥٠٠ محرفاً
	long := ""
	for i := 0; i < 200; i++ {
		long += "ا"
	}
	if _, err := Validate("whatsapp.otp_template", long); err != nil {
		t.Fatalf("رُفض نصٌّ عربيّ دون الحدّ (قياسٌ بالبايت لا بالمحرف؟): %v", err)
	}
	tooLong := ""
	for i := 0; i < 501; i++ {
		tooLong += "ا"
	}
	if _, err := Validate("whatsapp.otp_template", tooLong); err == nil {
		t.Fatal("قُبل نصٌّ يتجاوز الحدّ")
	}
}

// TestCatalog_DefaultsAreValid كلُّ افتراضٍ في الكتالوج يجب أن يجتاز تحقّقه.
//
// افتراضٌ خارج مداه فخٌّ صامت: النظام يعمل به، وأوّل محاولةٍ لحفظه من اللوحة
// تُرفض — فيبدو أن اللوحة معطوبة والعطب في التعريف.
func TestCatalog_DefaultsAreValid(t *testing.T) {
	for _, d := range Catalog {
		if _, err := Validate(d.Key, d.Default); err != nil {
			t.Errorf("افتراض %s لا يجتاز تحقّقه: %v", d.Key, err)
		}
	}
}

// TestCatalog_NoDuplicateKeys مفتاحٌ مكرّر يجعل الفهرس يبتلع أحد التعريفين
// صامتاً — فيُحرَس المفتاح بمدىً غير الذي يظنّه من كتبه.
func TestCatalog_NoDuplicateKeys(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range Catalog {
		if seen[d.Key] {
			t.Errorf("مفتاح مكرّر في الكتالوج: %s", d.Key)
		}
		seen[d.Key] = true
	}
	if len(seen) != len(Catalog) {
		t.Fatalf("الفهرس %d والكتالوج %d", len(seen), len(Catalog))
	}
}

// TestGroups_CoversCatalog **كلُّ مجموعةٍ لها مفاتيحُ لها موضعٌ في الترتيب.**
//
// الترتيبُ صار قائمةً مستقلّةً عن `Catalog` كي يظهر قسمٌ فارغ. **وقائمتان
// تصفان الشيءَ نفسَه تفترقان**: تُضاف مجموعةٌ لمفتاحٍ جديدٍ ولا تُضاف هنا،
// **فتختفي مفاتيحُها من اللوحة كلَّها** — ولا شيءَ يقول إنّها اختفت.
func TestGroups_CoversCatalog(t *testing.T) {
	listed := map[Group]bool{}
	for _, g := range Groups {
		if listed[g] {
			t.Fatalf("مجموعةٌ مكرّرةٌ في الترتيب: %s", g)
		}
		listed[g] = true
	}
	for _, d := range Catalog {
		if !listed[d.Group] {
			t.Fatalf("المفتاح %s في مجموعة %s وهي ليست في Groups — لن تظهر في اللوحة",
				d.Key, d.Group)
		}
	}
}
