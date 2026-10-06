package orders

import (
	"context"
	"testing"
)

// TestAvailability_LaunchClosedCarriesNotice **قبل الافتتاح يُقال إعلانُ المالك** (٢٠٢٦-١٠-٠٦) — لا
// «هذا لم يُفتح بعد» تحت «موقعك الحاليّ». والتطبيقُ يرسم `message` إن جاء.
func TestAvailability_LaunchClosedCarriesNotice(t *testing.T) {
	var s Service
	av, err := s.AvailabilityAt(context.Background(), nil, Gates{LaunchOpen: false, LaunchMessage: "قريباً الافتتاح"}, nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if av.Reason != ReasonLaunchClosed || av.Message != "قريباً الافتتاح" {
		t.Fatalf("إعلانُ الإطلاق لم يصل: %+v", av)
	}
}
