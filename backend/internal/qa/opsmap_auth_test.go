package qa

import (
	"encoding/json"
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **تصريحُ خريطة العمليات — البندان ٣٢ و٤٧**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا تُفتح الصفحةُ بتوكن إدارةٍ وحدَه.** **والحارسُ في الخادم لا في
// الواجهة** — **ومن حَرَس بإخفاء زرٍّ لم يحرس شيئاً.**

func TestOpsMap_RequiresAdminPanelRole(t *testing.T) {
	h := New(t)

	// **الزبونُ والسائقُ خارجَ الباب** — قبل الصلاحيّات المسمّاة.
	for _, who := range []struct {
		name  string
		token string
	}{
		{"زبون", h.NewUser("customer").Token},
		{"سائق", h.NewUser("driver").Token},
	} {
		res := h.GET("/api/v1/admin/ops-map/meta", who.token)
		if res.Code != http.StatusForbidden && res.Code != http.StatusUnauthorized {
			t.Errorf("%s فتح الخريطة — الردّ %d", who.name, res.Code)
		}
	}

	// **وبلا توكنٍ أصلاً.**
	if res := h.GET("/api/v1/admin/ops-map/meta", ""); res.Code != http.StatusUnauthorized {
		t.Errorf("بلا توكنٍ والردّ %d", res.Code)
	}
}

func TestOpsMap_MetaGrantsNamedPermissions(t *testing.T) {
	h := New(t)
	res := h.GET("/api/v1/admin/ops-map/meta", h.NewUser("admin").Token)
	if res.Code != http.StatusOK {
		t.Fatalf("الأدمنُ لم يفتح الخريطة — %d · %s", res.Code, string(res.Body))
	}
	// **والردُّ ملفوفٌ في `data`** — كما كلُّ ردود المنصّة.
	var env struct {
		Data struct {
			Permissions     []string `json:"permissions"`
			LocationPingSec int64    `json:"location_ping_sec"`
			AssignableSec   int      `json:"assignable_sec"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	out := env.Data
	// **والأدمنُ يملك كلَّ شيء** — ولا صلاحيّةَ تُنسى في الترجمة.
	want := []string{
		"VIEW_OPERATIONS_MAP", "VIEW_DRIVER_LOCATIONS", "VIEW_MERCHANT_LOCATIONS",
		"VIEW_ACTIVE_ORDERS", "MANAGE_COVERAGE", "MANAGE_BRANCHES",
		"VIEW_REP_ACTIVITY", "VIEW_DEMAND_ANALYTICS", "VIEW_MAP_FINANCIALS",
	}
	have := map[string]bool{}
	for _, p := range out.Permissions {
		have[p] = true
	}
	for _, w := range want {
		if !have[w] {
			t.Errorf("الأدمنُ لا يملك %s", w)
		}
	}
	// **والنبضةُ تُرسَل** — الواجهةُ تشرح بها معنى «حيّ».
	if out.LocationPingSec <= 0 || out.AssignableSec <= 0 {
		t.Errorf("عتباتٌ فارغة: %+v", out)
	}
}

func TestOpsMap_DriversLayerReturnsShape(t *testing.T) {
	h := New(t)
	res := h.GET("/api/v1/admin/ops-map/drivers", h.NewUser("admin").Token)
	if res.Code != http.StatusOK {
		t.Fatalf("طبقةُ السائقين — %d · %s", res.Code, string(res.Body))
	}
	var env struct {
		Data struct {
			Drivers []struct {
				ID        string `json:"id"`
				Freshness string `json:"freshness"`
			} `json:"drivers"`
			Count int `json:"count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	out := env.Data
	if out.Count != len(out.Drivers) {
		t.Errorf("عدٌّ %d وصفوفٌ %d", out.Count, len(out.Drivers))
	}
	// **وكلُّ صفٍّ يحمل حكمَ طزاجةٍ** — ولا صفَّ بلا حكم.
	for _, d := range out.Drivers {
		switch d.Freshness {
		case "LIVE", "FRESH", "STALE", "NO_LOCATION":
		default:
			t.Errorf("%s بحكمِ طزاجةٍ مجهول %q", d.ID, d.Freshness)
		}
	}
}

func TestOpsMap_BadBBoxIsRejected(t *testing.T) {
	h := New(t)
	tok := h.NewUser("admin").Token
	for _, bad := range []string{"1,2,3", "a,b,c,d", "40,35,39,36", "-200,35,39,36"} {
		res := h.GET("/api/v1/admin/ops-map/drivers?bbox="+bad, tok)
		if res.Code != http.StatusBadRequest {
			t.Errorf("مستطيلٌ %q قُبل — الردّ %d", bad, res.Code)
		}
	}
	// **والمعقولُ يمرّ.**
	if res := h.GET("/api/v1/admin/ops-map/drivers?bbox=38.9,35.9,39.1,36.0",
		tok); res.Code != http.StatusOK {
		t.Errorf("مستطيلُ الرقّة رُفض — %d", res.Code)
	}
}
