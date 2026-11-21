package drivermonitor

// **برّا الدوام بخانةٍ لحالها** (قرارُ المالك ٢٠٢٦-١١-٢٢: «حتى اذا مسكر دوامو اقدر اوصل لموقعه») —
// التطبيقُ صار يبعت ودوامُه مسكّر، فقيادتُه وقتها ما بتنحسب «بلا طلب» ولا بتطلّع إنذارَه.

import (
	"testing"
	"time"
)

func TestOFF01_OffShiftDrivingHasItsOwnBucket(t *testing.T) {
	// ٣ كم بالدوام بلا طلب، وبعد ما سكّر ٤ كم.
	start := origin()
	onPts := drive(start, 3, 8, 10*time.Second)
	closed := last(onPts).At.Add(30 * time.Second)
	offFrom := last(onPts)
	offFrom.At = closed.Add(30 * time.Second)
	offPts := drive(offFrom, 4, 8, 10*time.Second)
	pts := append(append([]Point{start}, onPts...), offFrom)
	pts = append(pts, offPts...)
	shifts := []ShiftSpan{{OpenedAt: start.At.Add(-time.Minute), ClosedAt: &closed}}

	res := Analyze(Input{Points: pts, Cfg: DefaultConfig(), Shifts: shifts, OffShiftFrom: start.At.Add(-time.Hour)})
	s := res.Summary
	if s.DriveKmOffShift < 3.5 || s.DriveKmOffShift > 4.5 {
		t.Fatalf("برّا الدوام لازم ~٤ كم، طلع %.2f", s.DriveKmOffShift)
	}
	if s.DriveKmNoOrder < 2.5 || s.DriveKmNoOrder > 3.5 {
		t.Fatalf("بلا طلب (بالدوام) لازم ~٣ كم، طلع %.2f", s.DriveKmNoOrder)
	}
	if res.Alert(3.6) {
		t.Fatal("قيادة برّا الدوام ما لازم تطلّع إنذار «بلا طلب»")
	}
	if d := s.DriveKmTotal - (s.DriveKmNoOrder + s.DriveKmOffShift + s.DriveKmWithOrder + s.DriveKmToPickup); d > 1e-9 || d < -1e-9 {
		t.Fatalf("المجموع لازم يشمل برّا الدوام: %+v", s)
	}
	seen := false
	for _, p := range res.Path {
		if p.Class == ClassOffShift {
			seen = true
		}
	}
	if !seen {
		t.Fatal("المسار لازم فيه جزء «برّا الدوام»")
	}

	// **وبلا `OffShiftFrom` (أيام قبل الميزة) متل قبل بالضبط** — الكل «بلا طلب».
	old := Analyze(Input{Points: pts, Cfg: DefaultConfig(), Shifts: shifts}).Summary
	if old.DriveKmOffShift != 0 || old.DriveKmNoOrder < 6.5 {
		t.Fatalf("بلا تاريخ بداية لازم ما يتغيّر شي: %+v", old)
	}
}
