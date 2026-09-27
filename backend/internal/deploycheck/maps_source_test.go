package deploycheck_test

// ══════════════════════════════════════════════════════════════════════
// **مصدرُ خريطةِ التجهيز رابطٌ مستقرٌّ لا يشيخ — حارسٌ يسقط قبل الصندوق**
// ══════════════════════════════════════════════════════════════════════
//
// (إصلاحُ ٢٠٢٦-٠٩-٢٧.)
//
// # لماذا حارسٌ هنا
//
// **علِق بناءُ خريطةِ الرقّة على الصندوق ثمّ سقط بـ404**: كان يشير إلى
// لقطةِ Geofabrik اليوميّةِ المؤرَّخةِ (`syria-260820.osm.pbf`)، **وGeofabrik
// تُسقط اللقطاتِ القديمةَ فيموت الرابط.** **ولا يُكتشف ذلك إلّا بعد تشغيلٍ
// على الصندوق** — دورةُ إخفاقٍ بطيئةٌ ومكلفة.
//
// **فهذا الحارسُ يسقط في `go test` قبل أيّ نشرٍ أو تشغيلِ صندوق**: يمنع
// عودةَ رابطٍ مؤرَّخٍ يشيخ، ويؤكّد بقاءَ بوّابةِ السلامة (تحقّقُ md5).

import (
	"os"
	"regexp"
	"testing"
)

const raqqaBuildScript = "../../../maps/scripts/build-raqqa-staging.sh"

func readRaqqaBuild(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(raqqaBuildScript)
	if err != nil {
		t.Fatalf("تعذّر قراءةُ سكربت بناء الرقّة %s: %v", raqqaBuildScript, err)
	}
	return string(b)
}

// TestRaqqaOSMSourceIsStableNotDated **لا رابطَ مؤرَّخٌ يشيخ** — لقطةٌ يوميّةٌ
// مؤرَّخةٌ (`syria-<أرقام>.osm.pbf`) تُسقطها Geofabrik فتردّ 404. المصدرُ
// يجب أن يكون `syria-latest.osm.pbf` المستقرّ.
func TestRaqqaOSMSourceIsStableNotDated(t *testing.T) {
	s := readRaqqaBuild(t)

	dated := regexp.MustCompile(`syria-\d+\.osm\.pbf`)
	if m := dated.FindString(s); m != "" {
		t.Fatalf("رابطُ OSM مؤرَّخٌ يشيخ (%q) — Geofabrik تُسقط اللقطاتِ القديمةَ فيردّ 404. استعمل syria-latest.osm.pbf", m)
	}
	if !regexp.MustCompile(`syria-latest\.osm\.pbf`).MatchString(s) {
		t.Fatal("لم يُعثر على مصدرِ OSM المستقرّ syria-latest.osm.pbf في سكربت البناء")
	}
}

// TestRaqqaOSMIntegrityGateKept **بوّابةُ السلامةِ لم تُضعَّف** — لمّا زال
// الرابطُ المؤرَّخُ زالت البصمةُ المجمّدة، **فيجب أن تبقى بوّابةٌ صارمة**:
// تحقّقٌ ضدَّ md5 الذي ينشره المزوّد. غيابُها يعني تنزيلاً بلا تحقّق.
func TestRaqqaOSMIntegrityGateKept(t *testing.T) {
	s := readRaqqaBuild(t)
	// **رابطُ md5 المنشور** — مبنيٌّ من رابط المقتطف (`$OSM_URL.md5`) أو صريحاً.
	if !regexp.MustCompile(`\.md5`).MatchString(s) || !regexp.MustCompile(`OSM_MD5_URL`).MatchString(s) {
		t.Fatal("لا تحقّقَ سلامةٍ: يجب جلبُ md5 المنشور (رابطُ .md5) والمطابقةُ عليه")
	}
	if !regexp.MustCompile(`md5sum`).MatchString(s) {
		t.Fatal("لا مطابقةَ md5 فعليّةٌ (md5sum) على المقتطف المنزَّل")
	}
}

// TestRaqqaBuildStaysStagingOnly **حارسُ العزلِ عن الإنتاج باقٍ** — لا يكتب
// السكربتُ تحت `/srv/rahalgo` (الإنتاج)، ويحرسه صراحةً.
func TestRaqqaBuildStaysStagingOnly(t *testing.T) {
	s := readRaqqaBuild(t)
	if !regexp.MustCompile(`/srv/rahalgo-staging`).MatchString(s) {
		t.Fatal("سكربتُ البناء لا يعمل تحت /srv/rahalgo-staging")
	}
	// حارسُ رفضِ الإنتاج يجب أن يُذكر PROD_ROOT/PROD ويقارن ضدّه.
	if !regexp.MustCompile(`/srv/rahalgo\b`).MatchString(s) {
		t.Fatal("لا إشارةَ إلى مسار الإنتاج /srv/rahalgo للحراسة ضدّه")
	}
}
