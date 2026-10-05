package opsmap

import (
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **حالُ السائق في لون — قرارُ المالك ٢٠٢٦-١٠-٠٥**
// ══════════════════════════════════════════════════════════════════════

// TestDriverTone_Matrix **أخضرُ متاح · برتقاليٌّ معه طلب · رماديٌّ خامل.**
func TestDriverTone_Matrix(t *testing.T) {
	cases := []struct {
		name      string
		onShift   bool
		active    int
		freshness string
		want      string
	}{
		{"على الدوام وموضعُه نابض", true, 0, FreshnessLive, DriverAvailable},
		{"على الدوام وموضعُه صالح", true, 0, FreshnessFresh, DriverAvailable},
		{"على الدوام وموضعُه شاخ", true, 0, FreshnessStale, DriverIdle},
		{"على الدوام بلا موضع", true, 0, FreshnessNone, DriverIdle},
		{"خارجَ الدوام وموضعُه نابض", false, 0, FreshnessLive, DriverIdle},
		// **ومعه طلبٌ يسبق كلَّ شيء** — ولو شاخ موضعُه أو انصرف.
		{"معه طلبٌ وموضعُه نابض", true, 1, FreshnessLive, DriverBusy},
		{"معه طلبٌ وموضعُه شاخ", true, 2, FreshnessStale, DriverBusy},
		{"معه طلبٌ وهو خارجَ الدوام", false, 1, FreshnessNone, DriverBusy},
	}
	for _, c := range cases {
		if got := DriverTone(c.onShift, c.active, c.freshness); got != c.want {
			t.Errorf("%s: %s والمتوقَّع %s", c.name, got, c.want)
		}
	}
}

// TestCountDrivers_UsesTone **العدّادُ يعدّ بما يلوّن** — ولا حكمَ ثانٍ.
func TestCountDrivers_UsesTone(t *testing.T) {
	list := []Driver{
		{Tone: DriverAvailable}, {Tone: DriverAvailable},
		{Tone: DriverBusy}, {Tone: DriverIdle}, {Tone: ""},
	}
	got := CountDrivers(list)
	want := DriverCounts{Total: 5, Available: 2, Busy: 1, Idle: 2}
	if got != want {
		t.Fatalf("العدّ %+v والمتوقَّع %+v", got, want)
	}
}

// TestFirstStuck_OldestStuckWins **نقرةُ «عالق» تذهب إلى أقدم عالق** — وغيرُ
// العالق لا يُختار مهما قدُم.
func TestFirstStuck_OldestStuckWins(t *testing.T) {
	now := time.Now()
	reason := "no_driver"
	list := []Order{
		{ID: "new-stuck", CreatedAt: now.Add(-5 * time.Minute), StuckReason: &reason},
		{ID: "old-fine", CreatedAt: now.Add(-90 * time.Minute)},
		{ID: "old-stuck", CreatedAt: now.Add(-40 * time.Minute), StuckReason: &reason},
	}
	if got := FirstStuck(list); got == nil || got.ID != "old-stuck" {
		t.Fatalf("الأوّلُ %+v والمتوقَّع old-stuck", got)
	}
	if got := FirstStuck(list[1:2]); got != nil {
		t.Fatalf("لا عالقَ والدالّةُ أرجعت %+v", got)
	}
}

// TestCustomerCells_PrivacyConstants **الخليّةُ حيٌّ لا بيت، والحدُّ ثلاثة.**
//
// **ومن صغّر الخليّةَ أو أنزل الحدَّ اقترب من البيت** — فيُسقطه هذا قبل أن
// يُنشر.
func TestCustomerCells_PrivacyConstants(t *testing.T) {
	if CustomerMinCount < 3 {
		t.Fatalf("الحدُّ الأدنى %d — خليّةٌ ببيتٍ أو اثنين تدلّ على أصحابها", CustomerMinCount)
	}
	// ‎٠٫٠٠٣ درجة ≈ ٣٣٠ م — وما دونها شارعٌ لا حيّ.
	if CustomerCellDeg < 0.003 {
		t.Fatalf("ضلعُ الخليّة %v درجة — أصغرُ من حيّ", CustomerCellDeg)
	}
}
