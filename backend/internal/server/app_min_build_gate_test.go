package server

// **بابا البناء في الخادم يساويان رقمَ البناء في التطبيقين** (مراجعةُ الفرع ٢٠٢٦-١١-٠٥).
//
// `customerOffPlatformMinBuild` يُخفي سببَ «السائق طلب رقمي» عن تطبيق زبونٍ دونه، و
// `driverTermsMinBuild` لا يقفل الدوامَ قبل الموافقة إلّا لتطبيق سائقٍ بلغه. **فإن بُني
// التطبيقُ برقمٍ دونهما** (للتجهيز أو للإصدار) **غاب السببُ عن التطبيق الذي فيه اسمُه، ولم
// يُقفل دوامُ سائقٍ في يده شاشةُ الشروط** — بلا خطأٍ ولا سطرٍ في سجلّ. **فالحارسُ يقرأ
// `versionCode` من ملفّ البناء نفسِه.**

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

var versionCodeRe = regexp.MustCompile(`(?m)^\s*versionCode\s*=\s*(\d+)`)

func appVersionCode(t *testing.T, app string) int {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "mobile", app, "build.gradle.kts"))
	if err != nil {
		t.Fatalf("ملفُّ بناء %s: %v", app, err)
	}
	m := versionCodeRe.FindSubmatch(raw)
	if m == nil {
		t.Fatalf("لا versionCode في %s", app)
	}
	n, _ := strconv.Atoi(string(m[1]))
	return n
}

func TestAppMinBuildGatesMatchVersionCode(t *testing.T) {
	if got := appVersionCode(t, "app-customer"); got < customerOffPlatformMinBuild {
		t.Errorf("**تطبيقُ الزبون %d وبابُ «السائق طلب رقمي» %d** — السببُ لا يظهر في التطبيق الذي فيه اسمُه",
			got, customerOffPlatformMinBuild)
	}
	if got := appVersionCode(t, "app-driver"); got < driverTermsMinBuild {
		t.Errorf("**تطبيقُ السائق %d وبابُ الشروط %d** — شاشةُ الموافقة في يده والدوامُ لا يُقفل",
			got, driverTermsMinBuild)
	}
}
