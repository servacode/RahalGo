package opsmap

import (
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **الصلاحيّات — البند ٣٢**
// ══════════════════════════════════════════════════════════════════════

func TestPerm_NoRoleNoMap(t *testing.T) {
	// **ولا تُفتح بتوكن إدارةٍ وحدَه** — من لا دورَ له لا يرى شيئاً.
	if Allows(nil, PermViewMap) {
		t.Fatal("بلا دورٍ والخريطةُ مفتوحة")
	}
	for _, p := range All() {
		if Allows([]string{"customer"}, p) || Allows([]string{"driver"}, p) {
			t.Fatalf("زبونٌ أو سائقٌ يملك %s", p)
		}
	}
}

func TestPerm_RoleMatrix(t *testing.T) {
	cases := []struct {
		role string
		perm Perm
		want bool
	}{
		{"admin", PermManageCoverage, true},
		{"admin", PermViewMoney, true},
		// **والعملياتُ توزّع فتحتاج المواضع** — ولا تُدير التغطيةَ ولا الفروع.
		{"ops", PermViewDrivers, true},
		{"ops", PermManageCoverage, false},
		{"ops", PermManageBranches, false},
		{"ops", PermViewMoney, false},
		// **والماليّةُ تقرأ المالَ ولا تتبع مواضعَ الناس** (البند ٣٣).
		{"finance", PermViewMoney, true},
		{"finance", PermViewDrivers, false},
	}
	for _, c := range cases {
		if got := Allows([]string{c.role}, c.perm); got != c.want {
			t.Errorf("%s · %s = %v وأُريد %v", c.role, c.perm, got, c.want)
		}
	}
}

func TestPerm_GrantedIsStableAndDeduped(t *testing.T) {
	a := Granted([]string{"ops", "finance", "ops"})
	b := Granted([]string{"finance", "ops"})
	if len(a) != len(b) {
		t.Fatalf("ترتيبٌ غيرُ ثابت: %v ≠ %v", a, b)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("ترتيبٌ غيرُ ثابت: %v ≠ %v", a, b)
		}
	}
	seen := map[Perm]int{}
	for _, p := range a {
		seen[p]++
		if seen[p] > 1 {
			t.Fatalf("تكرارٌ في %s", p)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **الطزاجة — البند ٥**
// ══════════════════════════════════════════════════════════════════════

func TestFreshness_DerivedNotInvented(t *testing.T) {
	const ping = 60
	cases := []struct {
		name string
		age  time.Duration
		has  bool
		want string
	}{
		{"بلا موضعٍ قطّ", 0, false, FreshnessNone},
		{"نبضةٌ الآن", 5 * time.Second, true, FreshnessLive},
		{"ثلاثُ نبضاتٍ بالضبط", 180 * time.Second, true, FreshnessLive},
		{"بعد ثلاثِ نبضات", 181 * time.Second, true, FreshnessFresh},
		{"حدُّ المحرّك بالضبط", AssignableWindow, true, FreshnessFresh},
		{"بعد حدّ المحرّك", AssignableWindow + time.Second, true, FreshnessStale},
	}
	for _, c := range cases {
		if got := Freshness(c.age, c.has, ping); got != c.want {
			t.Errorf("%s: %s وأُريد %s", c.name, got, c.want)
		}
	}
}

func TestFreshness_FollowsSettingNotConstant(t *testing.T) {
	// **ومن غيّر النبضةَ من اللوحة تحرّك حدُّ «حيّ» معه** — وهذا هو
	// «التشغيل بلا شيفرة».
	// **عمرٌ واحدٌ وثلاثةُ أحكامٍ بثلاثِ نبضات** — والحكمُ يتبع اللوحة.
	age := 100 * time.Second
	if got := Freshness(age, true, 20); got != FreshnessFresh {
		t.Fatalf("بنبضةِ ٢٠ ثانيةً (حدُّ حيٍّ ٦٠): %s", got)
	}
	if got := Freshness(age, true, 60); got != FreshnessLive {
		t.Fatalf("بنبضةِ ٦٠ ثانيةً (حدُّ حيٍّ ١٨٠): %s", got)
	}
	if got := Freshness(age, true, 120); got != FreshnessLive {
		t.Fatalf("بنبضةِ ١٢٠ ثانيةً: %s", got)
	}
}

func TestFreshness_ZeroPingFallsBackNotPanics(t *testing.T) {
	// **وإعدادٌ فارغٌ لا يجعل كلَّ سائقٍ حيّاً** — يُرتدّ إلى الافتراض.
	if got := Freshness(10*time.Second, true, 0); got != FreshnessLive {
		t.Fatalf("%s", got)
	}
	if got := Freshness(10*time.Minute, true, 0); got != FreshnessFresh {
		t.Fatalf("%s", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **مستطيلُ المشهد — البندان ٣١ و٤٣**
// ══════════════════════════════════════════════════════════════════════

func TestBBox_RejectsNonsense(t *testing.T) {
	bad := []BBox{
		{MinLng: 40, MinLat: 35, MaxLng: 39, MaxLat: 36}, // مقلوبٌ طولاً
		{MinLng: 39, MinLat: 36, MaxLng: 40, MaxLat: 35}, // مقلوبٌ عرضاً
		{MinLng: -200, MinLat: 35, MaxLng: 39, MaxLat: 36},
		{MinLng: 39, MinLat: -95, MaxLng: 40, MaxLat: 36},
		{},
	}
	for i, b := range bad {
		if b.Valid() {
			t.Errorf("مستطيلٌ %d غيرُ معقولٍ وقُبل: %+v", i, b)
		}
	}
	good := BBox{MinLng: 38.9, MinLat: 35.9, MaxLng: 39.1, MaxLat: 36.0}
	if !good.Valid() {
		t.Fatal("مستطيلُ الرقّة رُفض")
	}
}

func TestBBox_SQLUsesParametersNotLiterals(t *testing.T) {
	b := BBox{MinLng: 38.9, MinLat: 35.9, MaxLng: 39.1, MaxLat: 36.0}
	sql := b.SQL("u.last_location", 3)
	// **ولا رقمَ يُلصَق في النصّ** — والحقنُ يبدأ من هنا.
	for _, bad := range []string{"38.9", "35.9", "39.1", "36.0"} {
		if contains(sql, bad) {
			t.Fatalf("رقمٌ ملصوقٌ في النصّ: %s", sql)
		}
	}
	for _, want := range []string{"$3", "$4", "$5", "$6", "ST_MakeEnvelope"} {
		if !contains(sql, want) {
			t.Fatalf("ينقص %s في %s", want, sql)
		}
	}
	if len(b.Args()) != 4 {
		t.Fatalf("معاملاتٌ %d", len(b.Args()))
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
