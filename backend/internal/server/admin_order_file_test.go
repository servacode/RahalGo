package server

import (
	"testing"
	"time"
)

// TestWithReadyStep **«المتجر جهّز الطلب» في موضعه من المسار** (ملفُّ الطلب ٢٠٢٦-١٠-٣٠) — بلا قاعدة.
func TestWithReadyStep(t *testing.T) {
	t0 := time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC)
	steps := []TimelineStep{
		{Status: "accepted", At: t0},
		{Status: "preparing", At: t0.Add(time.Minute)},
		{Status: "picked_up", At: t0.Add(20 * time.Minute)},
	}
	if got := withReadyStep(steps, nil, "x"); len(got) != 3 {
		t.Fatalf("بلا وقت تجهيز تغيّر المسار: %v", got)
	}
	ready := t0.Add(15 * time.Minute)
	got := withReadyStep(steps, &ready, "مطعم")
	if len(got) != 4 || got[2].Status != "ready" || got[2].ActorRole != "merchant" ||
		got[2].Actor == nil || *got[2].Actor != "مطعم" || !got[2].At.Equal(ready) {
		t.Fatalf("خطوةُ التجهيز: %+v", got)
	}
	late := t0.Add(time.Hour)
	if got := withReadyStep(steps, &late, ""); got[3].Status != "ready" || got[3].Actor != nil {
		t.Fatalf("تجهيزٌ بعد آخر خطوة: %+v", got)
	}
}

// TestLateDelivery **«متأخّر» بمهلة عمر الطلب** — توأمُ `orders.LateDeliverySQL`.
func TestLateDelivery(t *testing.T) {
	t0 := time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC)
	on := t0.Add(59 * time.Minute)
	off := t0.Add(61 * time.Minute)
	if lateDelivery(t0, nil, 60) || lateDelivery(t0, &on, 60) || !lateDelivery(t0, &off, 60) {
		t.Fatal("حكمُ التأخير خاطئ")
	}
}
