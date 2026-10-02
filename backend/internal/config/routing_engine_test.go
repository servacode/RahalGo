package config

// **اختيارُ محرّك المسارات** — طلبُ المالك ٢٠٢٦-١٠-٠٢: الموتور يدخل كلّ
// الطرق مو سيارة. والافتراضُ OSRM، ومجهولُ الاسم يُرفض لا يُبدَّل صامتاً.

import "testing"

func TestRoutingEngineDefaultsToOSRM(t *testing.T) {
	t.Setenv("ROUTING_ENGINE", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.RoutingEngine != "osrm" {
		t.Fatalf("الافتراض %q", cfg.RoutingEngine)
	}
}

func TestRoutingEngineValhallaIsRead(t *testing.T) {
	t.Setenv("ROUTING_ENGINE", " Valhalla ")
	t.Setenv("VALHALLA_URL", "http://valhalla:8002")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.RoutingEngine != "valhalla" || cfg.ValhallaURL != "http://valhalla:8002" {
		t.Fatalf("قُرئ %q %q", cfg.RoutingEngine, cfg.ValhallaURL)
	}
}

func TestRoutingEngineUnknownIsRefused(t *testing.T) {
	t.Setenv("ROUTING_ENGINE", "graphhopper")
	if _, err := Load(); err == nil {
		t.Fatal("محرّكٌ مجهولٌ قُبل")
	}
}
