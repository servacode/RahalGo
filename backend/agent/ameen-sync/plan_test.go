package main

import (
	"testing"
	"time"
)

// TestPlanPausedSkipsEverything **موقوفٌ مؤقّتاً ⇒ لا قراءةَ ولا رفع** ولو طُلبت «مزامنة الآن» والجداول.
func TestPlanPausedSkipsEverything(t *testing.T) {
	now := time.Now()
	rc := &Remote{Paused: true, SyncNow: true, WantSchema: true, Interval: 1}
	if schema, sync := plan(rc, time.Time{}, now); schema || sync {
		t.Fatalf("paused: schema=%v sync=%v, want both false", schema, sync)
	}
}

// TestPlanRunning **شغّال ⇒ يزامن عند حلول المدّة أو «الآن»، ويبعث الجداول إن طُلبت.**
func TestPlanRunning(t *testing.T) {
	now := time.Now()
	if schema, sync := plan(&Remote{WantSchema: true, Interval: 5}, time.Time{}, now); !schema || !sync {
		t.Fatalf("due: schema=%v sync=%v", schema, sync)
	}
	if _, sync := plan(&Remote{Interval: 5}, now.Add(-time.Minute), now); sync {
		t.Fatal("synced a minute ago with interval 5: must wait")
	}
	if _, sync := plan(&Remote{Interval: 5, SyncNow: true}, now.Add(-time.Minute), now); !sync {
		t.Fatal("sync_now must sync immediately")
	}
}
