package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

// TestAuditActionsHaveArabicLabels **كلُّ فعلٍ يُسجَّل له اسمٌ يُقرأ.**
//
// (شكوى المالك ٢٠٢٦-٠٨-٠٧: «نصوصٌ إنكليزيّةٌ بكلّ اللوحات».)
//
// # ورابعُ مواضع العائلة
//
// **سبقتها**: حالاتُ الطلب («استرجاع طلب (cancelled)») · أنواعُ صندوق
// السائق (`order_collection`) · أنواعُ الدفتر (`platform_profit`).
//
// **والقاسمُ واحد**: اسمٌ يُكتب في Go واسمٌ يُسمّى في المعجم، **ولا شيءَ
// يجمع بينهما.** والشيفرةُ مكتوبةٌ لتسقط على الرمز الخام عند الفشل — **فلا
// خطأ ولا صمت، إنّما إنكليزيّةٌ على شاشة.**
//
// **وأثبت التكرارُ أنّ الإصلاحَ وحدَه لا يكفي**: أصلحتُ `menu.section_update`
// فظهر `finance.orders_exported` في التشغيل التالي. **وما يتكرّر يُحرَس.**
//
// # وتُستخرج الأفعالُ من الشيفرة لا من قائمةٍ تُصان
//
// **قائمةٌ تُكتب بيدٍ تشيخ يومَ يُضاف فعلٌ ولا يُذكر فيها** — وهو بعينه
// العطبُ الذي نحرسه.
//
// # وعَمِيَ الحارسُ عن نصف النداءات
//
// (كشفه المالك ٢٠٢٦-٠٨-٠٨ بلقطةٍ من سجلّ نشاط حساب — «شوف سجلّ النشاط شلون مو
// مفهوم شي» وفيها `menu.item_create` خامّاً.)
//
// **وحرفان كانا سببَ العمى**: `audit(` بحرفٍ صغيرٍ لا غير — **و`repo.Audit(`
// بحرفٍ كبيرٍ هو ما تكتب به حزمةُ الهويّة كلُّها.** فأربعةُ أفعالٍ حقيقيّةٍ
// (`admin.logout_all` · `auth.password_reset` · `auth.phone_change` ·
// `user.self_delete`) مرّت بلا اسمٍ عربيٍّ والحارسُ يقول «سليم».
//
// **وقائمةُ الجذور المكتوبةُ بيدٍ هي القائمةُ التي حذّر منها التعليقُ نفسُه**
// — خمسةُ مجلّداتٍ تُسمّى، **ومن أنشأ حزمةً سادسةً لا يعلم أنّ عليه ذكرَها.**
// فيُمشى على `internal/` كلِّها.
func TestAuditActionsHaveArabicLabels(t *testing.T) {
	actions := map[string]bool{}
	// **بحرفٍ كبيرٍ وصغير** — `s.audit(` و`repo.Audit(` كلاهما يسجّل.
	//
	// # وعَمِيَ ثانيةً عن `auditTx(` و`auditDetail(` (فحصُ ٢٠٢٦-١٠-٠٤)
	//
	// **`\b[Aa]udit\(` لا يرى `auditTx(`** — فتسعةُ أفعالٍ حسّاسةٍ (منحُ
	// صلاحيّةٍ لدور وسحبُها · إنشاءُ دور · نقلُ متجرٍ لمندوب · دفعُ مستحقٍّ
	// نقديّ …) طُبعت بالرمز الخام والحارسُ يقول «سليم». **فصار الاسمُ أيَّ
	// دالّةٍ فيها `audit`**، **ومعه مسحُ كلِّ نصٍّ بشكل فعلٍ** في الشيفرة
	// (`auditLiteralActions`) — ما ليس مفتاحَ إعدادٍ ولا قدرةً ولا مستثنىً
	// بسببٍ مكتوب.
	call := regexp.MustCompile(`\b\w*[Aa]udit\w*\([^)]*?"([a-z_]+\.[a-z_]+)"`)
	_ = filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, m := range call.FindAllStringSubmatch(string(src), -1) {
			actions[m[1]] = true
		}
		return nil
	})
	for _, a := range auditLiteralActions(t) {
		actions[a] = true
	}
	// **والمعاجمُ التي تسمّي أفعالاً بأسمائها** — لا تُستخرج بنصٍّ بل تُقرأ.
	for _, a := range authz.SensitiveActions() {
		actions[a.Action] = true
	}
	for a := range criticalAuditActions {
		actions[a] = true
	}
	for a := range identity.CriticalActions {
		actions[a] = true
	}
	for _, a := range auditSensitiveExtra {
		actions[a] = true
	}
	for _, a := range auditSessionActions {
		actions[a] = true
	}
	actions[identity.OwnerBootstrapAction] = true
	if len(actions) < 40 {
		t.Fatalf("لم أجد إلّا %d فعلاً — تبدّل شكلُ النداء والحارسُ صار أعمى", len(actions))
	}

	raw, err := os.ReadFile("../../../web/packages/i18n/src/locales/ar.json")
	if err != nil {
		t.Skipf("المعجمُ غيرُ متاح: %v", err)
	}
	var dict map[string]any
	if err := json.Unmarshal(raw, &dict); err != nil {
		t.Fatalf("المعجمُ غيرُ صالح: %v", err)
	}
	admin, _ := dict["admin"].(map[string]any)
	audit, _ := admin["audit"].(map[string]any)
	node, _ := audit["actions"].(map[string]any)
	if node == nil {
		t.Fatal("لا admin.audit.actions في المعجم")
	}

	for a := range actions {
		v, ok := node[a]
		if !ok {
			t.Errorf("الفعل %q بلا اسمٍ في admin.audit.actions — يُعرض خامّاً في سجلّ التدقيق", a)
			continue
		}
		s, _ := v.(string)
		if strings.ContainsAny(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			t.Errorf("اسمُ الفعل %q هو %q وفيه حرفٌ لاتينيّ", a, s)
		}
	}

	// **ولكلّ فعلٍ تبويب** (المشكلةُ الخامسة: أحداثُ السائق بلا تبويب).
	covered := map[string]bool{}
	for _, g := range auditGroups {
		for _, p := range g.Prefixes {
			covered[p] = true
		}
	}
	for a := range actions {
		if p := strings.SplitN(a, ".", 2)[0]; !covered[p] {
			t.Errorf("الفعل %q بادئتُه %q بلا تبويبٍ في auditGroups — لا يُرى إلّا في «الكل»", a, p)
		}
	}
	// **ولكلّ تبويبٍ اسمٌ عربيّ.**
	groups, _ := audit["groups"].(map[string]any)
	for _, g := range auditGroups {
		if v, _ := groups[g.Key].(string); v == "" {
			t.Errorf("التبويب %q بلا اسمٍ في admin.audit.groups", g.Key)
		}
	}

	// **ولا خريطةَ ثانيةٍ لنفس الأفعال.**
	//
	// **كانت `admin.users.auditActions` بأربعةَ عشرَ اسماً** تقرؤها شاشةُ
	// «سجلّ النشاط» في ملفّ الحساب، **وهذه المحروسةُ بأربعةٍ وسبعين** يقرؤها
	// سجلُّ التدقيق. **وثمانيةٌ فقط تتطابق** — فالشاشةُ الأولى تطبع الخامَّ
	// والحارسُ يشهد للثانية أنّها تامّة.
	//
	// **ونصفُ الأسماء في الميتة لا يكتبها المحرّك أصلاً** (`auth.register`
	// والمحرّكُ يكتب `user.register`) — **اسمٌ لفعلٍ لا يقع.**
	//
	// فما يُحرَس واحد، **ومن نسخ خريطةً ثانيةً أسقط البناء** لا انتظر شكوى.
	var stray func(path string, node map[string]any)
	stray = func(path string, node map[string]any) {
		hits := 0
		for k := range node {
			if actions[k] {
				hits++
			}
		}
		// **و`admin.stepUp.actions` ليست أسماءَ سجلّ** — جملةُ نافذة التأكيد
		// («منحُ قدرةٍ لدور» قبل الفعل)، ويحرسها `check-stepup-client`.
		if hits >= 3 && path != "admin.audit.actions" && path != "admin.stepUp.actions" {
			t.Errorf("خريطةُ أفعالٍ ثانيةٌ في %s (%d فعلاً) — تشيخ صامتةً فتُطبع المفاتيحُ خامّةً؛ والمحروسةُ admin.audit.actions", path, hits)
			return
		}
		for k, v := range node {
			if child, ok := v.(map[string]any); ok {
				if path == "" {
					stray(k, child)
				} else {
					stray(path+"."+k, child)
				}
			}
		}
	}
	stray("", dict)
}

// auditActionPrefixes بادئاتُ أفعال السجلّ — **ما يُمسح به نصُّ الشيفرة.**
var auditActionPrefixes = regexp.MustCompile(
	`"((?:admin|auth|finance|ops|menu|merchant|driver|customer|order|catalog|platform|user|branch|coverage|coverage_request|operational_area)\.[a-z_]+)"`)

// auditLiteralNotActions **نصوصٌ بشكل فعلٍ وليست فعلاً** — ولكلٍّ سببُه.
var auditLiteralNotActions = map[string]string{
	"auth.signup_verify": "مفتاحُ إعدادٍ تقرؤه الهويّةُ باسمه (boolSetting)",
}

// auditLiteralActions **كلُّ نصٍّ بشكل فعلٍ في الشيفرة** — ما ليس مفتاحَ
// إعدادٍ ولا قدرةً ولا مستثنىً.
//
// **ويُسقط البناءَ من كتب فعلاً جديداً بأيّ صيغةٍ** — متغيّرٍ أو ثابتٍ أو
// معاملٍ لدالّةٍ لا تحمل اسمَ `audit` — **ولم يسمّه بالعربيّة.**
func auditLiteralActions(t *testing.T) []string {
	t.Helper()
	skip := map[string]bool{}
	for _, d := range settings.Catalog {
		skip[d.Key] = true
	}
	for _, c := range authz.All() {
		skip[string(c)] = true
	}
	for k := range auditLiteralNotActions {
		skip[k] = true
	}
	var out []string
	for _, root := range []string{"..", filepath.Join("..", "..", "cmd", "api")} {
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			// **أدواتُ الاختبار والتجهيز ليست شيفرةَ المنصّة** — أعلامُها
			// بشكل مفاتيح الإعداد.
			if strings.Contains(filepath.ToSlash(path), "/testtruth/") || strings.Contains(filepath.ToSlash(path), "/qa/") {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			for _, m := range auditActionPrefixes.FindAllStringSubmatch(string(src), -1) {
				if !skip[m[1]] {
					out = append(out, m[1])
				}
			}
			return nil
		})
	}
	return out
}
