package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
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
	call := regexp.MustCompile(`\b[Aa]udit\([^)]*?"([a-z_]+\.[a-z_]+)"`)
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
		if hits >= 3 && path != "admin.audit.actions" {
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
