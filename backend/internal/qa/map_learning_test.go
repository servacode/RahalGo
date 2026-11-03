package qa

// ══════════════════════════════════════════════════════════════════════
// **الخريطة من السائقين** — قرار المالك ٢٠٢٦-١١-٠٣
// ══════════════════════════════════════════════════════════════════════
//
// أبواب الصفحة: القائمة والموافقة والرفض والتصدير، والصلاحية، وأنّ المصدَّر هو الموافَق عليه وحده.

import (
	"encoding/json"
	"strings"
	"testing"
)

func mlInsertCandidate(t *testing.T, h *Harness, lng0 float64) int64 {
	t.Helper()
	var id int64
	err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO map_street_candidates (line, length_m, points, days, first_day, last_day)
		VALUES (ST_GeogFromText($1), 120, 40, 3, CURRENT_DATE - 3, CURRENT_DATE - 1) RETURNING id`,
		"LINESTRING("+mlF(lng0)+" 35.9, "+mlF(lng0+0.001)+" 35.9)").Scan(&id)
	if err != nil {
		t.Fatalf("إدخال مرشَّح: %v", err)
	}
	return id
}

func mlF(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

func TestMapLearning_ApproveRejectExport(t *testing.T) {
	h := New(t)
	if _, err := h.Pool.Exec(ctxBG(), `DELETE FROM map_street_candidates`); err != nil {
		t.Fatal(err)
	}
	adm := h.NewUser("admin").Token
	a := mlInsertCandidate(t, h, 39.001)
	b := mlInsertCandidate(t, h, 39.011)
	c := mlInsertCandidate(t, h, 39.021)

	res := h.GET("/api/v1/admin/ops-map/street-candidates", adm)
	if res.Code != 200 || !strings.Contains(string(res.Body), `"count":3`) {
		t.Fatalf("القائمة: %s", res)
	}

	id := func(n int64) string { return strings.TrimSpace(string(mlJSON(n))) }
	if r := h.POST("/api/v1/admin/ops-map/street-candidates/"+id(a)+"/decide", adm,
		map[string]any{"approve": true, "name": "حارة التجربة"}); r.Code != 200 {
		t.Fatalf("موافقة: %s", r)
	}
	if r := h.POST("/api/v1/admin/ops-map/street-candidates/"+id(b)+"/decide", adm,
		map[string]any{"approve": false}); r.Code != 200 {
		t.Fatalf("رفض: %s", r)
	}
	// قرارٌ ثانٍ على نفس المرشَّح مرفوض.
	if r := h.POST("/api/v1/admin/ops-map/street-candidates/"+id(a)+"/decide", adm,
		map[string]any{"approve": false}); r.Code != 409 {
		t.Errorf("قرار مكرَّر = %d والمتوقَّع 409", r.Code)
	}
	if r := h.POST("/api/v1/admin/ops-map/street-candidates/999999999/decide", adm,
		map[string]any{"approve": true}); r.Code != 404 {
		t.Errorf("مجهول = %d والمتوقَّع 404", r.Code)
	}

	// المصدَّر هو الموافَق عليه وحده، باسمه.
	g := h.GET("/api/v1/admin/ops-map/street-candidates/export?format=geojson", adm)
	if g.Code != 200 || !strings.Contains(string(g.Body), "حارة التجربة") {
		t.Fatalf("تصدير geojson: %s", g)
	}
	var fc struct {
		Features []json.RawMessage `json:"features"`
	}
	if err := json.Unmarshal(g.Body, &fc); err != nil || len(fc.Features) != 1 {
		t.Fatalf("المتوقَّع خطٌّ واحد موافَق عليه، والناتج %d (%v)", len(fc.Features), err)
	}
	o := h.GET("/api/v1/admin/ops-map/street-candidates/export?format=osm", adm)
	if o.Code != 200 || !strings.Contains(string(o.Body), "<osm") {
		t.Errorf("تصدير osm: %s", o)
	}
	if r := h.GET("/api/v1/admin/ops-map/street-candidates/export?format=pdf", adm); r.Code != 422 {
		t.Errorf("صيغة فاسدة = %d والمتوقَّع 422", r.Code)
	}
	_ = c

	sum := h.GET("/api/v1/admin/ops-map/street-candidates/summary", adm)
	if sum.Code != 200 || sum.JSON()["pending"] != float64(1) || sum.JSON()["approved"] != float64(1) || sum.JSON()["rejected"] != float64(1) {
		t.Errorf("الملخّص: %s", sum)
	}
	if r := h.GET("/api/v1/admin/ops-map/street-candidates/heat?days=30", adm); r.Code != 200 {
		t.Errorf("الحرارة: %s", r)
	}
}

func TestMapLearning_RequiresCapability(t *testing.T) {
	h := New(t)
	for _, role := range []string{"customer", "driver", "finance"} {
		tok := h.NewUser(role).Token
		if r := h.GET("/api/v1/admin/ops-map/street-candidates", tok); r.Code != 403 {
			t.Errorf("%s: القائمة = %d والمتوقَّع 403", role, r.Code)
		}
		if r := h.GET("/api/v1/admin/ops-map/street-candidates/export?format=osm", tok); r.Code != 403 {
			t.Errorf("%s: التصدير = %d والمتوقَّع 403", role, r.Code)
		}
	}
	// موظّف العمليات يقرأ ولا يقرّر.
	ops := h.NewUser("operations").Token
	if r := h.POST("/api/v1/admin/ops-map/street-candidates/1/decide", ops, map[string]any{"approve": true}); r.Code != 403 {
		t.Errorf("العمليات تقرّر = %d والمتوقَّع 403", r.Code)
	}
}

func mlJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
