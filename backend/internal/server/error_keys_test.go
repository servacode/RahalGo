package server

// **كلُّ خطأٍ يرسله المحرّك له رسالةٌ عربيّة.**
//
// # ولماذا يُحرَس
//
// **الواجهةُ تترجم `message_key`** — ومفتاحٌ لا تجده تسقط إلى «حدث خطأٌ غير
// متوقّع». **فيُقال للمستخدم إنّ في المنصّة عطباً وليس فيها عطب**: القاعدةُ
// رفضت بحقٍّ ولها سببٌ مفهوم، **لكنّه لم يصل.**
//
// **وقِيس ٢٠٢٦-٠٨-١١**: سبعةُ مفاتيحَ بلا رسائل — منها «ساعات العمل غير
// صالحة» و«القسم يحتوي أصنافاً». **وكلُّها كانت تُقرأ عطباً في الخادم.**
//
// **والحارسُ يفحص المصدرين معاً**: ما يرسله Go وما يعرفه المعجم، **فلا
// يُضاف خطأٌ جديدٌ بلا رسالة.**

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// keyRe **مفاتيحُ الخطأ كما تُكتب في الشيفرة** — `httpx.NewError(..., "errors.x")`.
var keyRe = regexp.MustCompile(`"(errors\.[a-z_0-9]+)"`)

func TestEveryErrorKeyHasArabicMessage(t *testing.T) {
	root := filepath.Join("..", "..")
	keys := map[string][]string{} // مفتاح → الملفّات التي ترسله

	err := filepath.Walk(filepath.Join(root, "internal"), func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, m := range keyRe.FindAllStringSubmatch(string(b), -1) {
			keys[m[1]] = append(keys[m[1]], filepath.Base(p))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("تعذّر مسحُ الشيفرة: %v", err)
	}
	if len(keys) == 0 {
		t.Fatal("لم يُعثر على مفتاحٍ واحد — **الحارسُ يقرأ فراغاً فيمرّ دائماً**")
	}

	raw, err := os.ReadFile(filepath.Join(root, "..", "web", "packages", "i18n", "src", "locales", "ar.json"))
	if err != nil {
		t.Skipf("المعجمُ غيرُ متاح: %v", err)
	}
	var dict map[string]any
	if err := json.Unmarshal(raw, &dict); err != nil {
		t.Fatalf("المعجمُ لا يُقرأ: %v", err)
	}

	missing := []string{}
	for key, files := range keys {
		node := any(dict)
		ok := true
		for _, part := range strings.Split(key, ".") {
			m, isMap := node.(map[string]any)
			if !isMap {
				ok = false
				break
			}
			node, ok = m[part]
			if !ok {
				break
			}
		}
		if s, isStr := node.(string); !ok || !isStr || s == "" {
			missing = append(missing, key+"  ("+strings.Join(files, ", ")+")")
		}
	}
	if len(missing) > 0 {
		t.Fatalf("مفاتيحُ خطأٍ بلا رسالةٍ عربيّة — **تُقرأ «حدث خطأٌ غير متوقّع»**:\n  %s",
			strings.Join(missing, "\n  "))
	}
}
