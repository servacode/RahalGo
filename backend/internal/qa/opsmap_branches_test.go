package qa

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **سلامةُ الهرم — البند ٤٩**
// ══════════════════════════════════════════════════════════════════════
//
//	محافظة ← مدينة ← فرعُ المدينة الرئيسيّ ← مناطقُ تشغيليّة
//	                       └── فروعٌ فرعيّةٌ اختياريّة

type mapBranch struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Status   string   `json:"status"`
	CityID   string   `json:"city_id"`
	ParentID *string  `json:"parent_id"`
	Parent   *string  `json:"parent"`
	Lat      *float64 `json:"lat"`
	Areas    int      `json:"areas"`
}

type mapArea struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Active   bool    `json:"active"`
	CityID   string  `json:"city_id"`
	BranchID *string `json:"branch_id"`
	ZoneID   *string `json:"zone_id"`
}

// qaCity مدينةٌ للاختبار — **وتُمحى بعده.**
func qaCity(t *testing.T, h *Harness, name string) string {
	t.Helper()
	var id string
	err := h.Pool.QueryRow(context.Background(), `
		INSERT INTO cities (name, center, radius_m, active)
		VALUES ($1, ST_SetSRID(ST_MakePoint(39.0094, 35.9506),4326)::geography, 25000, true)
		RETURNING id::text`, name).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = h.Pool.Exec(ctx, `DELETE FROM operational_areas WHERE city_id = $1::uuid`, id)
		_, _ = h.Pool.Exec(ctx, `DELETE FROM branches WHERE city_id = $1::uuid AND type = 'sub'`, id)
		_, _ = h.Pool.Exec(ctx, `DELETE FROM branches WHERE city_id = $1::uuid`, id)
		_, _ = h.Pool.Exec(ctx, `DELETE FROM cities WHERE id = $1::uuid`, id)
	})
	return id
}

func postID(t *testing.T, h *Harness, path, tok string, body any) (string, int, string) {
	t.Helper()
	res := h.POST(path, tok, body)
	var env struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(res.Body, &env)
	return env.Data.ID, res.Code, string(res.Body)
}

func listBranches(t *testing.T, h *Harness, tok, query string) []mapBranch {
	t.Helper()
	res := h.GET("/api/v1/admin/ops-map/branches"+query, tok)
	if res.Code != http.StatusOK {
		t.Fatalf("الفروع: %d · %s", res.Code, string(res.Body))
	}
	var env struct {
		Data struct {
			Branches []mapBranch `json:"branches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	return env.Data.Branches
}

func TestBranch_CityGetsOnePrimaryAndManySubs(t *testing.T) {
	h := New(t)
	tok := h.NewUser("admin").Token
	city := qaCity(t, h, "QA مدينة "+uniq("c"))

	// **فرعُ المدينة الرئيسيّ.**
	primary, code, body := postID(t, h, "/api/v1/admin/ops-map/branches", tok,
		map[string]any{"name": "الفرعُ الرئيسيّ", "type": "primary",
			"city_id": city, "lat": 35.95, "lng": 39.01})
	if code != http.StatusOK || primary == "" {
		t.Fatalf("إنشاءُ فرعٍ رئيسيّ: %d · %s", code, body)
	}

	// **وثانٍ رئيسيٌّ في المدينة نفسِها يُردّ** (البند ٢٠).
	_, code, _ = postID(t, h, "/api/v1/admin/ops-map/branches", tok,
		map[string]any{"name": "رئيسيٌّ ثانٍ", "type": "primary", "city_id": city})
	if code < 400 {
		t.Error("فرعٌ رئيسيٌّ ثانٍ في المدينة نفسِها قُبل")
	}

	// **والفروعُ الفرعيّةُ تتعدّد.**
	for _, n := range []string{"فرعُ الشمال", "فرعُ الجنوب"} {
		id, code, body := postID(t, h, "/api/v1/admin/ops-map/branches", tok,
			map[string]any{"name": n, "type": "sub", "city_id": city, "parent_id": primary})
		if code != http.StatusOK || id == "" {
			t.Fatalf("فرعٌ فرعيٌّ %s: %d · %s", n, code, body)
		}
	}

	list := listBranches(t, h, tok, "?city_id="+city)
	if len(list) != 3 {
		t.Fatalf("فروعُ المدينة %d ولا ثلاثة", len(list))
	}
	subs := 0
	for _, b := range list {
		if b.Type == "sub" {
			subs++
			if b.ParentID == nil || *b.ParentID != primary {
				t.Errorf("فرعٌ فرعيٌّ %q أبوه %v", b.Name, b.ParentID)
			}
			if b.Parent == nil {
				t.Errorf("فرعٌ فرعيٌّ %q بلا اسمِ أب", b.Name)
			}
		}
	}
	if subs != 2 {
		t.Errorf("فروعٌ فرعيّةٌ %d ولا اثنان", subs)
	}
}

func TestBranch_HierarchyRulesEnforced(t *testing.T) {
	h := New(t)
	tok := h.NewUser("admin").Token
	city := qaCity(t, h, "QA هرم "+uniq("c"))
	primary, _, _ := postID(t, h, "/api/v1/admin/ops-map/branches", tok,
		map[string]any{"name": "رئيسيّ", "type": "primary", "city_id": city})

	cases := []struct {
		name string
		body map[string]any
	}{
		{"فرعيٌّ بلا أب", map[string]any{
			"name": "يتيم", "type": "sub", "city_id": city}},
		{"رئيسيٌّ بأب", map[string]any{
			"name": "رئيسيٌّ بأب", "type": "primary", "city_id": city, "parent_id": primary}},
		{"نوعٌ مجهول", map[string]any{
			"name": "غريب", "type": "regional", "city_id": city}},
		{"بلا اسم", map[string]any{
			"name": "   ", "type": "primary", "city_id": city}},
		{"بلا مدينة", map[string]any{"name": "بلا مدينة", "type": "primary"}},
		{"موقعٌ خارجَ الأرض", map[string]any{
			"name": "بعيد", "type": "sub", "city_id": city, "parent_id": primary,
			"lat": 999.0, "lng": 39.0}},
	}
	for _, c := range cases {
		_, code, body := postID(t, h, "/api/v1/admin/ops-map/branches", tok, c.body)
		if code != http.StatusBadRequest {
			t.Errorf("%s: قُبل — %d · %s", c.name, code, body)
		}
	}
}

// TestBranch_DistrictNeverBecomesBranch **البند ١٩ · `Geography ≠ Branch`.**
func TestBranch_DistrictNeverBecomesBranch(t *testing.T) {
	h := New(t)
	tok := h.NewUser("admin").Token
	city := qaCity(t, h, "QA جغرافيا "+uniq("c"))

	var districts int
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM districts`).Scan(&districts); err != nil {
		t.Fatal(err)
	}
	if districts == 0 {
		t.Skip("لا مناطقَ إداريّةً مبذورةً في هذه القاعدة")
	}

	// **وإنشاءُ مدينةٍ لا يفتح فرعاً** — ولا مسارَ يحوّل تقسيمَ الدولة
	// إلى فروع.
	if got := listBranches(t, h, tok, "?city_id="+city); len(got) != 0 {
		t.Fatalf("مدينةٌ جديدةٌ وُلد لها %d فرعاً تلقائيّاً", len(got))
	}
	var branches int
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM branches`).Scan(&branches); err != nil {
		t.Fatal(err)
	}
	if branches >= districts {
		t.Errorf("فروعٌ %d ومناطقُ إداريّةٌ %d — **وكلُّ منطقةٍ صارت فرعاً؟**",
			branches, districts)
	}
}

func TestArea_RelationsAndNotADistrict(t *testing.T) {
	h := New(t)
	tok := h.NewUser("admin").Token
	city := qaCity(t, h, "QA مناطق "+uniq("c"))
	primary, _, _ := postID(t, h, "/api/v1/admin/ops-map/branches", tok,
		map[string]any{"name": "رئيسيّ", "type": "primary", "city_id": city})

	// **منطقةٌ تشغيليّةٌ مسنَدةٌ إلى فرع.**
	areaID, code, body := postID(t, h, "/api/v1/admin/ops-map/areas", tok,
		map[string]any{"name": "حيُّ المشلب", "city_id": city, "branch_id": primary})
	if code != http.StatusOK || areaID == "" {
		t.Fatalf("منطقةٌ تشغيليّة: %d · %s", code, body)
	}
	// **ومنطقةٌ بلا فرعٍ مقبولة** — تُسنَد لاحقاً.
	if _, code, body := postID(t, h, "/api/v1/admin/ops-map/areas", tok,
		map[string]any{"name": "حيٌّ بلا فرع", "city_id": city}); code != http.StatusOK {
		t.Fatalf("منطقةٌ بلا فرع: %d · %s", code, body)
	}
	// **والاسمُ لا يتكرّر في المدينة.**
	if _, code, _ := postID(t, h, "/api/v1/admin/ops-map/areas", tok,
		map[string]any{"name": "حيُّ المشلب", "city_id": city}); code < 400 {
		t.Error("اسمُ منطقةٍ تكرّر في المدينة نفسِها")
	}

	res := h.GET("/api/v1/admin/ops-map/areas?city_id="+city, tok)
	if res.Code != http.StatusOK {
		t.Fatalf("قائمةُ المناطق: %d", res.Code)
	}
	var env struct {
		Data struct {
			Areas []mapArea `json:"areas"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	if len(env.Data.Areas) != 2 {
		t.Fatalf("مناطقُ %d ولا اثنتان", len(env.Data.Areas))
	}
	// **والفرعُ يعرف كم منطقةً تحته.**
	for _, b := range listBranches(t, h, tok, "?city_id="+city) {
		if b.ID == primary && b.Areas != 1 {
			t.Errorf("الفرعُ يقول %d منطقةً وله واحدة", b.Areas)
		}
	}
}

func TestBranch_ManagePermissionRequired(t *testing.T) {
	h := New(t)
	city := qaCity(t, h, "QA صلاحيّة "+uniq("c"))
	body := map[string]any{"name": "فرع", "type": "primary", "city_id": city}

	// **والعملياتُ لا تفتح فرعاً** — قرارُ عملٍ لا تشغيلٌ يوميّ.
	if _, code, _ := postID(t, h, "/api/v1/admin/ops-map/branches",
		h.NewUser("ops").Token, body); code != http.StatusForbidden {
		t.Errorf("العملياتُ فتحت فرعاً — %d", code)
	}
	// **وتقرأ الفروع.**
	if res := h.GET("/api/v1/admin/ops-map/branches",
		h.NewUser("ops").Token); res.Code != http.StatusOK {
		t.Errorf("العملياتُ حُجبت عن قراءة الفروع — %d", res.Code)
	}
}
