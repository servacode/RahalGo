package qa

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **صحّةُ التحليلات — البند ٥٠**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يكفي أن يردّ البابُ `200`** — **الأعدادُ تُقارَن ببياناتٍ
// معلومةٍ زُرعت بيدنا.**

type cell struct {
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
	Count int     `json:"count"`
}

type demandOut struct {
	Orders    []cell  `json:"orders"`
	Requests  []cell  `json:"requests"`
	Merchants []cell  `json:"merchants"`
	Drivers   []cell  `json:"drivers"`
	Unserved  []cell  `json:"unserved"`
	CellDeg   float64 `json:"cell_deg"`
}

func demandOf(t *testing.T, h *Harness, tok, query string) demandOut {
	t.Helper()
	res := h.GET("/api/v1/admin/ops-map/demand"+query, tok)
	if res.Code != http.StatusOK {
		t.Fatalf("التحليلات: %d · %s", res.Code, string(res.Body))
	}
	var env struct {
		Data demandOut `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	return env.Data
}

// cellAt عدُّ الخليّة التي تقع فيها هذه النقطة.
func cellAt(cells []cell, lat, lng, deg float64) int {
	for _, c := range cells {
		if c.Lat-deg/2 <= lat && lat < c.Lat+deg/2 &&
			c.Lng-deg/2 <= lng && lng < c.Lng+deg/2 {
			return c.Count
		}
	}
	return 0
}

// seedRequests يزرع طلباتِ تغطيةٍ في نقطةٍ واحدة.
func seedRequests(t *testing.T, h *Harness, lat, lng float64, n int) {
	t.Helper()
	ctx := context.Background()
	var ids []string
	for i := 0; i < n; i++ {
		var id string
		// **والخليّةُ والهويّةُ تُكتبان كما يكتبهما المنتَج** —
		// **وزرعٌ ينقص عمّا تكتبه الشيفرةُ يقيس جدولاً لا يقع.**
		// (`0153` · `0154`: لا صفَّ بلا هويّةٍ تُقرأ.)
		err := h.Pool.QueryRow(ctx, `
			WITH c AS (
			    SELECT floor($1 / 0.01) * 0.01 + 0.005 AS cy,
			           floor($2 / 0.01) * 0.01 + 0.005 AS cx
			)
			INSERT INTO coverage_requests
			  (at, address_text, source, cell_y, cell_x, target_key)
			SELECT ST_SetSRID(ST_MakePoint($2,$1),4326)::geography, 'QA', 'test',
			       c.cy, c.cx, 'cell:' || c.cy::text || ',' || c.cx::text
			  FROM c
			RETURNING id::text`, lat, lng).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = h.Pool.Exec(context.Background(),
				`DELETE FROM coverage_requests WHERE id = $1::uuid`, id)
		}
	})
}

// TestDemand_RequestCellCountsMatchSeed **البند ٥٠.**
func TestDemand_RequestCellCountsMatchSeed(t *testing.T) {
	h := New(t)
	tok := h.NewUser("admin").Token
	// **نقطةٌ بعيدةٌ عن كلّ ما زرعه غيرُنا** — فالخليّةُ تخصُّنا وحدَنا.
	const lat, lng = 33.123456, 41.987654

	before := demandOf(t, h, tok, "")
	base := cellAt(before.Requests, lat, lng, before.CellDeg)

	seedRequests(t, h, lat, lng, 4)

	after := demandOf(t, h, tok, "")
	got := cellAt(after.Requests, lat, lng, after.CellDeg)
	if got != base+4 {
		t.Fatalf("خليّةُ الطلبات %d وأُريد %d — **العدُّ لا يطابق ما زُرع**",
			got, base+4)
	}
	// **وضلعُ الخليّة يُرسَل** — والواجهةُ تشرح به معنى الرقم.
	if after.CellDeg <= 0 {
		t.Error("ضلعُ الخليّة صفرٌ — ولا يُفسَّر عددٌ بلا مساحة")
	}
}

// TestDemand_LayersAreSeparateNotMerged **البند ١٨.**
func TestDemand_LayersAreSeparateNotMerged(t *testing.T) {
	h := New(t)
	tok := h.NewUser("admin").Token
	const lat, lng = 33.223456, 41.887654

	seedRequests(t, h, lat, lng, 3)
	d := demandOf(t, h, tok, "")

	if cellAt(d.Requests, lat, lng, d.CellDeg) != 3 {
		t.Fatalf("طلباتُ التغطيةِ %d ولا ثلاثة",
			cellAt(d.Requests, lat, lng, d.CellDeg))
	}
	// **وطلبُ تغطيةٍ ليس طلباً نُفِّذ** — **والخلطُ يمحو المعنيين.**
	if n := cellAt(d.Orders, lat, lng, d.CellDeg); n != 0 {
		t.Errorf("طلبُ تغطيةٍ عُدَّ في كثافة الطلبات — %d", n)
	}
	if n := cellAt(d.Merchants, lat, lng, d.CellDeg); n != 0 {
		t.Errorf("طلبُ تغطيةٍ عُدَّ في كثافة المتاجر — %d", n)
	}
}

// TestDemand_UnservedShrinksWhenCoverageDrawn **أدقُّ رقمٍ في الصفحة.**
func TestDemand_UnservedShrinksWhenCoverageDrawn(t *testing.T) {
	h := New(t)
	tok := h.NewUser("admin").Token
	const lat, lng = 33.323456, 41.787654

	seedRequests(t, h, lat, lng, 2)
	if n := cellAt(demandOf(t, h, tok, "").Unserved, lat, lng, 0.01); n != 2 {
		t.Fatalf("غيرُ المغطّى %d ولا اثنان", n)
	}

	// **ومن رسم منطقةً هنا نقص الرقمُ** — بلا أن يلمس أحدٌ حالَ الطلبات.
	makePolygonZone(t, h, lat, lng, 0.02)
	if n := cellAt(demandOf(t, h, tok, "").Unserved, lat, lng, 0.01); n != 0 {
		t.Errorf("رُسمت التغطيةُ والرقمُ ما زال %d", n)
	}
}

// TestDemand_TimeRangeFilters **البند ٢٩.**
func TestDemand_TimeRangeFilters(t *testing.T) {
	h := New(t)
	tok := h.NewUser("admin").Token
	const lat, lng = 33.423456, 41.687654
	seedRequests(t, h, lat, lng, 2)

	// **واليومَ يشملها** — زُرعت الآن.
	if n := cellAt(demandOf(t, h, tok, "?range=today").Requests, lat, lng, 0.01); n != 2 {
		t.Errorf("مدى اليوم: %d ولا اثنان", n)
	}
	// **ومدىً ماضٍ لا يشملها.**
	past := "?from=2020-01-01T00:00:00Z&to=2020-02-01T00:00:00Z"
	if n := cellAt(demandOf(t, h, tok, past).Requests, lat, lng, 0.01); n != 0 {
		t.Errorf("مدىً في ٢٠٢٠ ردَّ %d طلباً زُرع اليوم", n)
	}
}

// TestOpportunity_ScoreIsExplainedNotPredicted **البند ٢٨.**
func TestOpportunity_ScoreIsExplainedNotPredicted(t *testing.T) {
	h := New(t)
	tok := h.NewUser("admin").Token
	const lat, lng = 33.523456, 41.587654
	seedRequests(t, h, lat, lng, 5)

	res := h.GET("/api/v1/admin/ops-map/opportunities", tok)
	if res.Code != http.StatusOK {
		t.Fatalf("الفرص: %d · %s", res.Code, string(res.Body))
	}
	var env struct {
		Data struct {
			Opportunities []struct {
				Lat      float64  `json:"lat"`
				Lng      float64  `json:"lng"`
				Requests int      `json:"requests"`
				Score    int      `json:"score"`
				Reasons  []string `json:"reasons"`
			} `json:"opportunities"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, o := range env.Data.Opportunities {
		if o.Lat-0.005 <= lat && lat < o.Lat+0.005 &&
			o.Lng-0.005 <= lng && lng < o.Lng+0.005 {
			found = true
			if o.Requests != 5 {
				t.Errorf("طلباتُ الخليّة %d ولا خمسة", o.Requests)
			}
			// **والدرجةُ محسوبةٌ بالمعادلة المكتوبة**: ٥×٣ + ٥×٢ = ٢٥.
			if o.Score != 25 {
				t.Errorf("الدرجةُ %d والمعادلةُ تعطي 25", o.Score)
			}
			// **ولا درجةَ بلا سببٍ مسمّى.**
			if len(o.Reasons) == 0 {
				t.Error("فرصةٌ بلا سبب")
			}
		}
	}
	if !found {
		t.Fatal("الخليّةُ المزروعةُ ليست في الفرص")
	}
}

// TestRep_ActivityFromExistingDataOnly **البندان ٢٤ و٢٥.**
func TestRep_ActivityFromExistingDataOnly(t *testing.T) {
	h := New(t)
	tok := h.NewUser("admin").Token
	res := h.GET("/api/v1/admin/ops-map/reps", tok)
	if res.Code != http.StatusOK {
		t.Fatalf("المندوبون: %d · %s", res.Code, string(res.Body))
	}
	var env struct {
		Data struct {
			Reps []struct {
				ID        string   `json:"id"`
				Merchants int      `json:"merchants"`
				Earnings  *int64   `json:"earnings"`
				Lat       *float64 `json:"lat"`
			} `json:"reps"`
			Points []struct {
				RepID string `json:"rep_id"`
			} `json:"points"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	// **والأدمنُ يرى المستحقّات.**
	for _, r := range env.Data.Reps {
		if r.Earnings == nil {
			t.Errorf("الأدمنُ لا يرى مستحقّاتِ %s", r.ID)
		}
	}
	// **والعملياتُ تراقب النشاطَ ولا ترى المال** (البند ٢٦).
	res2 := h.GET("/api/v1/admin/ops-map/reps", h.NewUser("ops").Token)
	if res2.Code != http.StatusOK {
		t.Fatalf("العملياتُ حُجبت عن نشاط المندوبين: %d", res2.Code)
	}
	var env2 struct {
		Data struct {
			Reps []struct {
				Earnings *int64 `json:"earnings"`
			} `json:"reps"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res2.Body, &env2); err != nil {
		t.Fatal(err)
	}
	for _, r := range env2.Data.Reps {
		if r.Earnings != nil {
			t.Fatalf("العملياتُ ترى مستحقَّ مندوبٍ %d", *r.Earnings)
		}
	}
	// **والماليّةُ لا تراقب نشاطَ المندوبين أصلاً.**
	if res := h.GET("/api/v1/admin/ops-map/reps",
		h.NewUser("finance").Token); res.Code != http.StatusForbidden {
		t.Errorf("الماليّةُ فتحت نشاطَ المندوبين — %d", res.Code)
	}
}

// TestRep_MapNeverGatesConversion **البند ٢٤ · لا مناطقَ إلزاميّة.**
//
// **والخريطةُ تُلاحِظ ولا تحكم** — **ولا مسارَ إنشاءٍ يستدعي شيئاً من
// حزمة الخريطة.**
func TestRep_MapNeverGatesConversion(t *testing.T) {
	h := New(t)
	// **ومندوبٌ بلا متجرٍ واحدٍ يظلّ يستطيع أن يسجّل** — ولا موضعَ
	// يمنعه. **والدليلُ أنّ بابَ المرشَّحين لا يمرّ بالخريطة**:
	// يُنشأ مرشَّحٌ بعيدٌ عن كلّ منطقةٍ ويُقبَل.
	rep := h.NewUser("sales")
	res := h.POST("/api/v1/rep/leads", rep.Token, map[string]any{
		"store_name": "متجرٌ بعيدٌ جدّاً",
		"owner_name": "صاحبُه",
		"phone":      uniqPhone(),
		"area":       "خارجَ كلّ تغطية",
		"lat":        33.9,
		"lng":        41.9,
	})
	// **والمقبولُ أو المرفوضُ لسببٍ غيرِ جغرافيّ كلاهما يثبت العقد** —
	// **والمرفوضُ بـ«خارج المنطقة» وحدَه يكسره.**
	if contains(string(res.Body), "out_of_zone") ||
		contains(string(res.Body), "territory") {
		t.Fatalf("تسجيلُ مرشَّحٍ رُدَّ لسببٍ جغرافيّ: %s", string(res.Body))
	}
}

// TestSearch_ScopedToPermissions **البند ٣٦.**
func TestSearch_ScopedToPermissions(t *testing.T) {
	h := New(t)
	item := h.NewItem(5000)
	var mname string
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT name FROM merchants WHERE id = $1::uuid`, item.MerchantID).Scan(&mname); err != nil {
		t.Fatal(err)
	}

	find := func(tok, q string) []struct {
		Kind  string `json:"kind"`
		ID    string `json:"id"`
		Label string `json:"label"`
	} {
		res := h.GET("/api/v1/admin/ops-map/search?q="+q, tok)
		if res.Code != http.StatusOK {
			t.Fatalf("البحث: %d · %s", res.Code, string(res.Body))
		}
		var env struct {
			Data struct {
				Hits []struct {
					Kind  string `json:"kind"`
					ID    string `json:"id"`
					Label string `json:"label"`
				} `json:"hits"`
			} `json:"data"`
		}
		if err := json.Unmarshal(res.Body, &env); err != nil {
			t.Fatal(err)
		}
		return env.Data.Hits
	}

	// **والأدمنُ يجد المتجر** — **ويُبحَث بما يخصُّه**: «QA» يحملها
	// مئةُ متجرٍ في قاعدةٍ مشتركة، **وحدُّ العشرة يقصّها.**
	uniqPart := mname[len(mname)-6:]
	hit := false
	for _, x := range find(h.NewUser("admin").Token, uniqPart) {
		if x.ID == item.MerchantID && x.Kind == "merchant" {
			hit = true
		}
	}
	if !hit {
		t.Errorf("الأدمنُ لم يجد متجرَ %q بالبحث عن %q", mname, uniqPart)
	}

	// **والماليّةُ لا تبحث في السائقين** — لا تملك رؤيةَ مواضعهم.
	drv := h.NewUser("driver")
	for _, x := range find(h.NewUser("finance").Token, "QA") {
		if x.Kind == "driver" {
			t.Fatalf("الماليّةُ وجدت سائقاً %q — وهي لا تملك رؤيتَهم", x.Label)
		}
	}
	_ = drv

	// **وحرفٌ واحدٌ لا يبحث** — ولا يُمسح الجدولُ كلُّه لحرف.
	//
	// **والحرفُ العربيُّ بايتان**، **فعدُّ البايتات يمرّره** — وهو ما
	// وقع فعلاً (٢٠٢٦-٠٩-٠٦: ردَّ ثمانيَ نتائج). **والعدُّ بالحروف.**
	for _, one := range []string{"ا", "Q"} {
		if got := find(h.NewUser("admin").Token, one); len(got) != 0 {
			t.Errorf("حرفٌ واحدٌ %q ردَّ %d نتيجة", one, len(got))
		}
	}
}
