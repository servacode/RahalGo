package androidmap

import "testing"

// TestNoMissingDeviceCountsAsPass **البند ٤٤** — جهازٌ غيرُ متوفّرٍ ليس نجاحاً.
func TestNoMissingDeviceCountsAsPass(t *testing.T) {
	for _, d := range Devices {
		if d.Status != Available && d.Run == Executed {
			t.Errorf("%s: غيرُ متوفّرٍ ومسجَّلٌ EXECUTED", d.ID)
		}
		if d.Why == "" || d.Evidence == "" {
			t.Errorf("%s: بلا سببٍ أو بلا دليل", d.ID)
		}
	}
	primaries := 0
	for _, d := range Devices {
		if d.Primary {
			primaries++
		}
	}
	if primaries != 1 {
		t.Errorf("أجهزةُ قبولٍ مرجعيّةٌ %d — يُنتظر واحد", primaries)
	}
	c := Snapshot()["counts"].(Counts)
	t.Logf("DEVICES = %d · AVAILABLE = %d · EXECUTED = %d",
		c.Devices, c.DevicesAvailable, c.DevicesExecuted)
}

// TestDeviceCasesAreNotClaimedRun **ولا يُدّعى تشغيلٌ لم يقع.**
func TestDeviceCasesAreNotClaimedRun(t *testing.T) {
	avail := map[string]bool{}
	for _, d := range Devices {
		avail[d.Class] = d.Status == Available
	}
	anyDevice := false
	for _, d := range Devices {
		if d.Status == Available {
			anyDevice = true
		}
	}
	for _, x := range Cases {
		if x.Evidence == "" {
			t.Errorf("%s: بلا دليل", x.ID)
		}
		if (x.Kind == OnDevice || x.Kind == Drive) && x.Result != NotRun && !anyDevice {
			t.Errorf("%s: نتيجةٌ %s ولا جهازَ متوفّر — **ادّعاءُ تشغيلٍ لم يقع**",
				x.ID, x.Result)
		}
		if x.Kind == Structural && x.Test == "" {
			t.Errorf("%s: بنيويٌّ بلا اختبارٍ يحرسه", x.ID)
		}
		if x.Result == NotRun && x.Needs == "" {
			t.Errorf("%s: لم يُشغَّل ولا يقول ما يلزم", x.ID)
		}
	}
	c := Snapshot()["counts"].(Counts)
	t.Logf("CASES = %d · STRUCTURAL = %d · DEVICE = %d · DRIVE = %d",
		c.Cases, c.Structural, c.DeviceK, c.DriveK)
	t.Logf("AUTOMATED = %d · SEMI = %d · MANUAL = %d",
		c.AutomatedN, c.SemiAutoN, c.ManualN)
	t.Logf("EXPECTED_FAIL = %d · RISK_CONFIRMED = %d · NOT_RUN = %d",
		c.ExpectedFailN, c.RiskConfirmedN, c.NotRunN)
	t.Logf("APPS COVERED = %d/4", c.Apps)
}

// TestUniqueIDs معرّفاتٌ لا تتكرّر.
func TestUniqueIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range Devices {
		if seen[d.ID] {
			t.Errorf("جهازٌ مكرَّر: %s", d.ID)
		}
		seen[d.ID] = true
	}
	seen = map[string]bool{}
	for _, x := range Cases {
		if seen[x.ID] {
			t.Errorf("حالةٌ مكرَّرة: %s", x.ID)
		}
		seen[x.ID] = true
	}
}
