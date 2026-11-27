package server

// قائمة بوت الواتساب (قرارُ المالك ٢٠٢٦-١١-٢٧) — الدوالُّ الخالصة: توحيدُ ما يكتبه الناس، ونعم/لا،
// ورسالةُ خارج الدوام بيوم الفتح وساعته بتوقيت دمشق.

import (
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/platform"
)

func TestWABOTU01_NormAndYesNo(t *testing.T) {
	for in, want := range map[string]string{"١": "1", "1️⃣": "1", " ٦ ": "6", "قائمة.": "قائمة", "إي": "اي", "أيوه": "ايوه", "YES": "yes"} {
		if got := waBotNorm(in); got != want {
			t.Fatalf("waBotNorm(%q) = %q، المنتظَر %q", in, got, want)
		}
	}
	for _, y := range []string{"نعم", "إي", "اي", "أيوه", "ايه", "yes"} {
		if waBotYesNo(waBotNorm(y), true) != 1 {
			t.Fatalf("%q ليست نعم", y)
		}
	}
	for _, n := range []string{"لا", "لأ", "no", "كلا"} {
		if waBotYesNo(waBotNorm(n), true) != 0 {
			t.Fatalf("%q ليست لا", n)
		}
	}
	if waBotYesNo("ربما", true) != -1 || waBotYesNo("نعم", false) != -1 {
		t.Fatal("غيرُ المفهوم صار جواباً")
	}
	if n, ok := waBotOrderNumber(waBotNorm("#١٢٣")); !ok || n != 123 {
		t.Fatalf("رقمُ الطلب: %d %v", n, ok)
	}
	if _, ok := waBotOrderNumber("abc"); ok {
		t.Fatal("نصٌّ صار رقمَ طلب")
	}
}

func TestWABOTU02_ClosedMessage(t *testing.T) {
	loc := platform.Location()
	now := time.Date(2026, 11, 27, 23, 0, 0, 0, loc) // الجمعة
	next := time.Date(2026, 11, 28, 9, 0, 0, 0, loc)
	if got := waBotClosedMessage(now, &next); !strings.Contains(got, "انتهى دوامنا لهذا اليوم، ونسعد بخدمتك غداً من الساعة 9:00 صباحاً.") {
		t.Fatalf("غداً: %q", got)
	}
	early := time.Date(2026, 11, 28, 6, 0, 0, 0, loc)
	if got := waBotClosedMessage(early, &next); !strings.Contains(got, "اليوم من الساعة 9:00 صباحاً") {
		t.Fatalf("اليوم: %q", got)
	}
	later := time.Date(2026, 11, 30, 16, 30, 0, 0, loc)
	if got := waBotClosedMessage(now, &later); !strings.Contains(got, "يوم الاثنين من الساعة 4:30 مساءً") {
		t.Fatalf("يومٌ آخر: %q", got)
	}
	if got := waBotClosedMessage(now, nil); !strings.Contains(got, "في أقرب وقت") {
		t.Fatalf("بلا موعد: %q", got)
	}
}
