package server

import (
	"testing"
	"time"
)

// TestDriveFailures_BackoffThenRetry **دوامٌ تعثّر قياسُه يُنحّى عشرَ دقائق ثمّ يعود** — ونجاحُه يمحوه.
// كان المتعثّرُ يبقى بلا `computed_at` فيعود رأسَ كلّ جولة، وخمسون منه توقف الكانس كلَّه.
func TestDriveFailures_BackoffThenRetry(t *testing.T) {
	var f driveFailures
	now := time.Now()
	if got := f.skip(now); len(got) != 0 {
		t.Fatalf("فارغٌ في البدء: %v", got)
	}
	f.failed("a", now)
	f.failed("b", now)
	if got := f.skip(now.Add(time.Minute)); len(got) != 2 {
		t.Fatalf("المتعثّران منحّيان: %v", got)
	}
	f.ok("a")
	if got := f.skip(now.Add(time.Minute)); len(got) != 1 || got[0] != "b" {
		t.Fatalf("النجاحُ يمحو: %v", got)
	}
	if got := f.skip(now.Add(driveFailBackoff + time.Second)); len(got) != 0 {
		t.Fatalf("بعد المهلة يعود: %v", got)
	}
}
