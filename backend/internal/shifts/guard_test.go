package shifts

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// **حارسُ الباب الواحد** — لا `UPDATE users SET … on_shift =` خارج هالحزمة.
//
// كانت تسعةُ مواضع تكتب العمود مباشرة (ثمانيةٌ في `server/` وواحدٌ في `qa/factory.go`)
// **ولا سجلّ**: دوامٌ ينسكّر بالترك أو الطارئ أو العمليات أو إيقاف الحساب ولا يُعرف.
// ومسارٌ ثامنٌ جديد يكتب العمود بيده يرجّع العمى نفسه — **فيسقط البناء عنده.**

var (
	updUsers = regexp.MustCompile(`(?is)UPDATE\s+users\s+SET\s`)
	onShiftW = regexp.MustCompile(`(?i)\bon_shift\s*=`)
)

// directWrites مواضعُ كتابة `on_shift` في ملفّات Go غير الاختبار تحت `root`.
func directWrites(t *testing.T, root string, skipShifts bool) []string {
	t.Helper()
	var out []string
	self, _ := filepath.Abs(".")
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if abs, _ := filepath.Abs(path); skipShifts && filepath.Dir(abs) == self {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		src := string(raw)
		for _, loc := range updUsers.FindAllStringIndex(src, -1) {
			rest := src[loc[1]:]
			end := len(rest)
			for _, stop := range []string{"WHERE", "where", "`", "RETURNING"} {
				if i := strings.Index(rest, stop); i >= 0 && i < end {
					end = i
				}
			}
			if onShiftW.MatchString(rest[:end]) {
				out = append(out, filepath.ToSlash(path)+":"+strconv.Itoa(strings.Count(src[:loc[0]], "\n")+1))
			}
		}
		return nil
	})
	return out
}

// TestShifts_NoDirectOnShiftWrites **كل فتحٍ وسكّرٍ للدوام يمرّ من `shifts`** — فينكتب صفّه وسببه.
func TestShifts_NoDirectOnShiftWrites(t *testing.T) {
	// **والحارسُ ليس أعمى**: يرى كتاباتِ هالحزمة نفسها (فتحٌ وسكّرٌ وإرجاع).
	if self := directWrites(t, ".", false); len(self) < 3 {
		t.Fatalf("الحارسُ أعمى: وجد %d كتابةً في shifts نفسها والمتوقَّع ٣ — %v", len(self), self)
	}
	var bad []string
	for _, root := range []string{"..", filepath.Join("..", "..", "cmd")} {
		bad = append(bad, directWrites(t, root, true)...)
	}
	for _, b := range bad {
		t.Errorf("كتابةٌ مباشرة لـon_shift خارج internal/shifts: %s — استعمل shifts.Open/Close", b)
	}
}
