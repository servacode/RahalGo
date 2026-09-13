package qa

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **انحدارُ خدمةِ التغطية — البندان ١٥ و٤٨**
// ══════════════════════════════════════════════════════════════════════
//
// # وهذه أخطرُ ما في `MAP-3`
//
// **`ZoneAt` تقرّر من تصله المنصّةُ أصلاً** — **وخطأٌ فيها يردّ زبائنَ
// حقيقيّين بـ«خارج نطاق التوصيل».**
//
// **والعقدُ لم يتبدّل**: `Delivery Address controls serviceability`.
// **وما تبدّل كيف تُقاس المنطقة** — **وكلُّ منطقةٍ بشكلها هي.**

// zoneScrub يمسح مناطقَ الاختبار — **والجدولُ مشتركٌ بين كلّ الاختبارات**،
// **ومنطقةٌ تبقى تغيّر جوابَ غيرها.**
func zoneScrub(t *testing.T, h *Harness, ids ...string) {
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = h.Pool.Exec(context.Background(),
				`DELETE FROM delivery_zones WHERE id = $1::uuid`, id)
		}
	})
}

// isolateZones يُخفي كلَّ منطقةٍ قائمةٍ ثمّ يعيدها.
//
// **ولا يُحذف شيء** — تُطفأ ثمّ تُشعَل، **فبياناتُ من سبقنا سالمة.**
func isolateZones(t *testing.T, h *Harness) {
	t.Helper()
	ctx := context.Background()
	rows, err := h.Pool.Query(ctx, `SELECT id::text FROM delivery_zones WHERE active`)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if _, err := h.Pool.Exec(ctx,
			`UPDATE delivery_zones SET active = false WHERE id = $1::uuid`, id); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = h.Pool.Exec(context.Background(),
				`UPDATE delivery_zones SET active = true WHERE id = $1::uuid`, id)
		}
	})
}

// makeRadiusZone منطقةٌ دائريّةٌ بالنموذج القديم — **حرفاً كما كانت.**
func makeRadiusZone(t *testing.T, h *Harness, lat, lng float64, radius int) string {
	t.Helper()
	var id string
	err := h.Pool.QueryRow(context.Background(), `
		INSERT INTO delivery_zones (name, center, radius_m, delivery_fee, min_order, active)
		VALUES ('QA دائرة', ST_SetSRID(ST_MakePoint($2,$1),4326)::geography, $3, 0, 0, true)
		RETURNING id::text`, lat, lng, radius).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	zoneScrub(t, h, id)
	return id
}

// makePolygonZone مربَّعٌ حول نقطةٍ بنصفِ ضلعٍ بالدرجات.
func makePolygonZone(t *testing.T, h *Harness, lat, lng, half float64) string {
	t.Helper()
	ring := [][2]float64{
		{lng - half, lat - half},
		{lng + half, lat - half},
		{lng + half, lat + half},
		{lng - half, lat + half},
	}
	body := map[string]any{"name": "QA مضلَّع", "ring": ring,
		"delivery_fee": 0, "min_order": 0}
	res := h.POST("/api/v1/admin/ops-map/coverage", h.NewUser("admin").Token, body)
	if res.Code != http.StatusOK {
		t.Fatalf("إنشاءُ مضلَّع: %d · %s", res.Code, string(res.Body))
	}
	var env struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	if env.Data.ID == "" {
		t.Fatalf("مضلَّعٌ بلا معرّف: %s", string(res.Body))
	}
	zoneScrub(t, h, env.Data.ID)
	return env.Data.ID
}

// servable **أتُخدَم هذه النقطة؟** — بالباب العامّ الذي يستعمله الزبون.
//
// **ولا يُسأل `ZoneAt` مباشرةً** — **العقدُ ما يراه الزبونُ لا ما تراه
// دالّة.**
func servable(t *testing.T, h *Harness, lat, lng float64) bool {
	t.Helper()
	res := h.GET(zoneQuery(lat, lng), "")
	switch res.Code {
	case http.StatusOK:
		return true
	case http.StatusBadRequest:
		// **و`out_of_zone` هو جوابُ «لا نصلك»** — ولا يُخلَط بخطأ صيغة.
		if contains(string(res.Body), "out_of_zone") {
			return false
		}
	}
	t.Fatalf("بابُ المنطقة ردَّ %d · %s", res.Code, string(res.Body))
	return false
}

func zoneQuery(lat, lng float64) string {
	return "/api/v1/public/zone?lat=" + ftoa(lat) + "&lng=" + ftoa(lng)
}

func ftoa(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

// ══════════════════════════════════════════════════════════════════════
// **الحالاتُ السبع — البند ٤٨**
// ══════════════════════════════════════════════════════════════════════

func TestCoverage_LegacyRadiusUnchanged(t *testing.T) {
	h := New(t)
	isolateZones(t, h)
	// **دائرةٌ نصفُ قطرها كيلومتران حول مركز الرقّة.**
	makeRadiusZone(t, h, 35.9506, 39.0094, 2000)

	if !servable(t, h, 35.9506, 39.0094) {
		t.Error("مركزُ الدائرة خارجَ التغطية — النموذجُ القديم انكسر")
	}
	// **ونقطةٌ على بعد ~٥٥ كم** (الطبقة) — خارجَ الدائرة.
	if servable(t, h, 35.8300, 38.5500) {
		t.Error("نقطةٌ بعيدةٌ صارت مغطّاةً — الدائرةُ لا تحدّ")
	}
}

func TestCoverage_PolygonInsideOutsideBoundary(t *testing.T) {
	h := New(t)
	isolateZones(t, h)
	// **مربَّعٌ نصفُ ضلعه ٠٫٠١ درجة ≈ ١٫١ كم.**
	makePolygonZone(t, h, 35.9506, 39.0094, 0.01)

	if !servable(t, h, 35.9506, 39.0094) {
		t.Error("داخلَ المضلَّع ولا يُخدَم")
	}
	if servable(t, h, 35.9506, 39.0500) {
		t.Error("خارجَ المضلَّع ويُخدَم")
	}
	// **والحدُّ نفسُه** — `ST_Covers` تشمل الحافّة.
	if !servable(t, h, 35.9506, 39.0194) {
		t.Error("نقطةٌ على الحافّة رُفضت — و`ST_Covers` تشملها")
	}
}

func TestCoverage_DisabledZoneDoesNotServe(t *testing.T) {
	h := New(t)
	isolateZones(t, h)
	id := makePolygonZone(t, h, 35.9506, 39.0094, 0.01)

	if !servable(t, h, 35.9506, 39.0094) {
		t.Fatal("المضلَّعُ لا يخدم وهو مفعَّل")
	}
	res := h.POST("/api/v1/admin/ops-map/coverage/"+id+"/active",
		h.NewUser("admin").Token, map[string]any{"active": false})
	if res.Code != http.StatusOK {
		t.Fatalf("إيقافُ المنطقة: %d · %s", res.Code, string(res.Body))
	}
	// **ومنطقةٌ موقوفةٌ لا تخدم** — **وجدولٌ صار فارغاً من المفعَّل
	// يفتح البابَ بقرارٍ قديمٍ مكتوب**، فتُبقى دائرةٌ بعيدةٌ مفعَّلة
	// ليبقى الحدُّ عاملاً.
	makeRadiusZone(t, h, 34.0000, 38.0000, 1000)
	if servable(t, h, 35.9506, 39.0094) {
		t.Error("منطقةٌ موقوفةٌ ما زالت تخدم")
	}
}

func TestCoverage_OverlappingZonesPickNearestCentre(t *testing.T) {
	h := New(t)
	isolateZones(t, h)
	// **مضلَّعٌ ودائرةٌ يتداخلان على النقطة نفسِها.**
	makePolygonZone(t, h, 35.9506, 39.0094, 0.02)
	makeRadiusZone(t, h, 35.9506, 39.0094, 3000)

	// **والمهمُّ أنّها تُخدَم** — والترتيبُ بالمسافة إلى المركز كما كان،
	// **ولا يقع خطأُ «صفّان فأكثر».**
	if !servable(t, h, 35.9510, 39.0100) {
		t.Error("نقطةٌ داخلَ منطقتين متداخلتين لم تُخدَم")
	}
}

func TestCoverage_MultipleZonesEachMeasuredByItsOwnShape(t *testing.T) {
	h := New(t)
	isolateZones(t, h)
	// **دائرةٌ في الرقّة ومضلَّعٌ بعيدٌ عنها** — **وكلٌّ يُقاس بشكله.**
	makeRadiusZone(t, h, 35.9506, 39.0094, 1500)
	makePolygonZone(t, h, 36.5000, 40.0000, 0.02)

	if !servable(t, h, 35.9506, 39.0094) {
		t.Error("داخلَ الدائرة ولا يُخدَم")
	}
	if !servable(t, h, 36.5000, 40.0000) {
		t.Error("داخلَ المضلَّع البعيد ولا يُخدَم")
	}
	// **وبينهما لا شيء.**
	if servable(t, h, 36.2000, 39.5000) {
		t.Error("نقطةٌ بين المنطقتين صارت مغطّاة")
	}
}

// TestCoverage_EmptyTableIsClosed **ونُقض القرارُ** (٢٠٢٦-٠٩-١٣).
//
// **وكان يشترط أن يبقى البابُ مفتوحاً بجدولٍ فارغ** (قرارُ ٢٠٢٦-٠٨-١٨)
// — **ونقضه المالكُ لإطلاقٍ عامّ**: **إعدادُ تغطيةٍ غائبٌ أو مُطفأٌ لا
// يفتح العالمَ لقبولِ الطلبات.**
func TestCoverage_EmptyTableIsClosed(t *testing.T) {
	h := New(t)
	isolateZones(t, h)
	// **ولا منطقةَ مفعَّلةٌ الآن** — **فالإخفاقُ يُغلق.**
	if servable(t, h, 35.9506, 39.0094) {
		t.Error("**بلا تغطيةٍ صالحةٍ بقي التوصيلُ مفتوحاً** — " +
			"**ولا تغطيةَ صالحةً ليس «كلُّ مكانٍ مُغطّى».**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **صحّةُ الشكل — البند ١٣**
// ══════════════════════════════════════════════════════════════════════

func TestCoverage_GeometryValidationRejectsBadShapes(t *testing.T) {
	h := New(t)
	tok := h.NewUser("admin").Token
	cases := []struct {
		name string
		ring [][2]float64
	}{
		{"نقطتان", [][2]float64{{39.0, 35.9}, {39.1, 35.9}}},
		{"فارغ", nil},
		{"خارجَ الأرض", [][2]float64{{999, 35.9}, {39.1, 35.9}, {39.1, 36.0}}},
		// **وفراشةٌ تتقاطع مع نفسها** — لا يكشفها فحصٌ نصّيّ،
		// **و`ST_IsValid` تكشفها.**
		{"تتقاطع مع نفسها", [][2]float64{
			{39.00, 35.90}, {39.02, 35.92}, {39.02, 35.90}, {39.00, 35.92},
		}},
	}
	for _, c := range cases {
		res := h.POST("/api/v1/admin/ops-map/coverage", tok, map[string]any{
			"name": "QA سيّئ", "ring": c.ring,
		})
		if res.Code != http.StatusBadRequest {
			t.Errorf("%s: قُبل — الردّ %d · %s", c.name, res.Code, string(res.Body))
		}
	}
	// **والاسمُ الفارغُ يُردّ أيضاً.**
	res := h.POST("/api/v1/admin/ops-map/coverage", tok, map[string]any{
		"name": "  ", "ring": [][2]float64{{39.0, 35.9}, {39.1, 35.9}, {39.1, 36.0}},
	})
	if res.Code != http.StatusBadRequest {
		t.Errorf("اسمٌ فارغٌ قُبل — %d", res.Code)
	}
}

func TestCoverage_LegacyScreenNeverSeesPolygons(t *testing.T) {
	h := New(t)
	id := makePolygonZone(t, h, 35.9506, 39.0094, 0.01)

	// **شاشةُ الدوائر لا تعرف رسمَ مضلَّع** — **ولو رأته لَعرضته
	// دائرةً كاذبةً ومحاه من سحب مقبضَها.**
	res := h.GET("/api/v1/admin/zones", h.NewUser("admin").Token)
	if res.Code != http.StatusOK {
		t.Fatalf("قائمةُ المناطق: %d · %s", res.Code, string(res.Body))
	}
	if contains(string(res.Body), id) {
		t.Error("مضلَّعٌ ظهر في شاشة الدوائر")
	}
}

func TestCoverage_ManagePermissionRequired(t *testing.T) {
	h := New(t)
	body := map[string]any{"name": "QA", "ring": [][2]float64{
		{39.0, 35.9}, {39.1, 35.9}, {39.1, 36.0}}}
	// **والعملياتُ لا تُدير التغطية** — رسمُ مضلَّعٍ يبدّل من تصله المنصّة.
	if res := h.POST("/api/v1/admin/ops-map/coverage",
		h.NewUser("ops").Token, body); res.Code != http.StatusForbidden {
		t.Errorf("العملياتُ رسمت منطقةً — %d", res.Code)
	}
	// **وتقرؤها.**
	if res := h.GET("/api/v1/admin/ops-map/coverage",
		h.NewUser("ops").Token); res.Code != http.StatusOK {
		t.Errorf("العملياتُ حُجبت عن قراءة التغطية — %d", res.Code)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ══════════════════════════════════════════════════════════════════════
// **دورةُ حياة طلب التغطية — البندان ١٦ و٣٥**
// ══════════════════════════════════════════════════════════════════════

type covRequest struct {
	ID      string  `json:"id"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Address string  `json:"address"`
	Status  string  `json:"status"`
	Source  string  `json:"source"`
	City    *string `json:"city"`
	UserID  *string `json:"user_id"`
	Note    string  `json:"note"`
}

func listRequests(t *testing.T, h *Harness, tok, query string) []covRequest {
	t.Helper()
	res := h.GET("/api/v1/admin/ops-map/coverage-requests"+query, tok)
	if res.Code != http.StatusOK {
		t.Fatalf("طلباتُ التغطية: %d · %s", res.Code, string(res.Body))
	}
	var env struct {
		Data struct {
			Requests []covRequest `json:"requests"`
			States   []string     `json:"states"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	if len(env.Data.States) != 5 {
		t.Errorf("حالاتٌ %d ولا خمس: %v", len(env.Data.States), env.Data.States)
	}
	return env.Data.Requests
}

func TestCoverageRequest_AnonymousMayAskAndLifecycleRuns(t *testing.T) {
	h := New(t)
	// **ومن لا حسابَ له يضغط الزرَّ** — وهو أصدقُ إشارةٍ عندنا.
	res := h.POST("/api/v1/public/coverage-request", "", map[string]any{
		"lat": 35.9506, "lng": 39.0094, "address": "حيُّ المشلب",
	})
	if res.Code != http.StatusCreated {
		t.Fatalf("طلبٌ بلا حساب: %d · %s", res.Code, string(res.Body))
	}
	var env struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	id := env.Data.ID
	if id == "" {
		t.Fatalf("طلبٌ بلا معرّف: %s", string(res.Body))
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM coverage_requests WHERE id = $1::uuid`, id)
	})

	tok := h.NewUser("admin").Token
	var found *covRequest
	for _, x := range listRequests(t, h, tok, "?status=new") {
		if x.ID == id {
			c := x
			found = &c
		}
	}
	if found == nil {
		t.Fatal("الطلبُ لا يظهر في الجديد")
	}
	if found.Status != "new" {
		t.Errorf("حالٌ ابتدائيّةٌ %q ولا `new`", found.Status)
	}
	if found.UserID != nil {
		t.Errorf("طلبٌ بلا حسابٍ ونُسب إلى %q", *found.UserID)
	}
	// **والمدينةُ تُستنتج من الدائرة** — ولا تُسأل ممّن يضغط.
	if found.City == nil {
		t.Error("لم تُستنتج مدينةُ الطلب — ونقطةُ الرقّة داخلَ دائرتها")
	}

	// ── تبديلُ الحال وملاحظةٌ داخليّة ────────────────────────
	up := h.PATCH("/api/v1/admin/ops-map/coverage-requests/"+id, tok,
		map[string]any{"status": "planned", "note": "بعد رمضان"})
	if up.Code != http.StatusOK {
		t.Fatalf("تبديلُ الحال: %d · %s", up.Code, string(up.Body))
	}
	after := listRequests(t, h, tok, "?status=planned")
	hit := false
	for _, x := range after {
		if x.ID == id {
			hit = true
			if x.Note != "بعد رمضان" {
				t.Errorf("الملاحظةُ %q", x.Note)
			}
		}
	}
	if !hit {
		t.Error("الطلبُ لم ينتقل إلى `planned`")
	}
	// **وقرارٌ بلا صاحبٍ لا يُراجَع** — فيُسجَّل من بدّله ومتى.
	var by *string
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT decided_by::text FROM coverage_requests WHERE id = $1::uuid`,
		id).Scan(&by); err != nil {
		t.Fatal(err)
	}
	if by == nil {
		t.Error("تبديلُ الحال بلا صاحب")
	}
}

func TestCoverageRequest_UnknownStatusRejected(t *testing.T) {
	h := New(t)
	res := h.POST("/api/v1/public/coverage-request", "",
		map[string]any{"lat": 35.9506, "lng": 39.0094})
	if res.Code != http.StatusCreated {
		t.Fatalf("%d · %s", res.Code, string(res.Body))
	}
	var env struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(res.Body, &env)
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM coverage_requests WHERE id = $1::uuid`, env.Data.ID)
	})

	tok := h.NewUser("admin").Token
	// **وحالٌ لا يعرفها القيدُ تُردّ قبل القاعدة** — بجوابٍ مقروء
	// لا بـ`SQLSTATE`.
	up := h.PATCH("/api/v1/admin/ops-map/coverage-requests/"+env.Data.ID, tok,
		map[string]any{"status": "مؤجّل"})
	if up.Code != http.StatusBadRequest {
		t.Errorf("حالٌ مجهولةٌ قُبلت — %d", up.Code)
	}
	// **والترشيحُ بحالٍ مجهولةٍ يُردّ أيضاً.**
	if res := h.GET("/api/v1/admin/ops-map/coverage-requests?status=xx", tok); res.Code != http.StatusBadRequest {
		t.Errorf("ترشيحٌ بحالٍ مجهولةٍ — %d", res.Code)
	}
}

func TestCoverageRequest_BadPointRejected(t *testing.T) {
	h := New(t)
	for _, bad := range []map[string]any{
		{"lat": 999.0, "lng": 39.0},
		{"lat": 35.9, "lng": -200.0},
	} {
		if res := h.POST("/api/v1/public/coverage-request", "", bad); res.Code != http.StatusBadRequest {
			t.Errorf("نقطةٌ خارجَ الأرض قُبلت — %d", res.Code)
		}
	}
}
