package qa

// ══════════════════════════════════════════════════════════════════════
// **أماكن بلا اسم وأسامي** — قرار المالك ٢٠٢٦-١١-٠٣ («اي موافق»)
// ══════════════════════════════════════════════════════════════════════
//
// كاشفان ببيانات اصطناعية: وقفاتُ السائقين المتكرّرة التي لا معلمَ عندها، وأسماءُ المعالم في عناوين الزبائن؛
// والموافقةُ تُدخل المعلمَ في `map_landmarks` بمصدر `drivers`؛ والأبواب بالصلاحية.

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/maplearn"
)

type mpCand struct {
	ID        int64    `json:"id"`
	Source    string   `json:"source"`
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Lat       float64  `json:"lat"`
	Lng       float64  `json:"lng"`
	Visits    int      `json:"visits"`
	Drivers   int      `json:"drivers"`
	Customers int      `json:"customers"`
	Days      int      `json:"days"`
	Evidence  int      `json:"evidence"`
	Samples   []string `json:"samples"`
}

func mpList(t *testing.T, h *Harness, tok, query string) []mpCand {
	t.Helper()
	res := h.GET("/api/v1/admin/ops-map/place-candidates"+query, tok)
	if res.Code != 200 {
		t.Fatalf("قائمة الأماكن: %s", res)
	}
	var env struct {
		Data struct {
			Candidates []mpCand `json:"candidates"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatalf("فك القائمة: %v (%s)", err, res)
	}
	return env.Data.Candidates
}

func mpNear(cs []mpCand, p dmSpot) *mpCand {
	for i := range cs {
		if abs(cs[i].Lat-p.lat) < 0.0006 && abs(cs[i].Lng-p.lng) < 0.0006 {
			return &cs[i]
		}
	}
	return nil
}

func mpByName(cs []mpCand, name string) *mpCand {
	for i := range cs {
		if cs[i].Name == name {
			return &cs[i]
		}
	}
	return nil
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// mpStand سائقٌ يقف عند نقطة أربع دقائق في يومٍ قبل اليوم بـ`ago` يوماً.
func mpStand(t *testing.T, h *Harness, driver string, p dmSpot, ago, slot int) {
	t.Helper()
	_, base := dmDay()
	tr := &dmTrail{t: t, h: h, driver: driver, cur: base.AddDate(0, 0, -ago).Add(time.Duration(slot) * time.Hour), pos: p}
	tr.put(tr.cur, p, false)
	tr.stand(4 * time.Minute)
}

func TestPlaces_StopsDetector_ApproveInsertsLandmark(t *testing.T) {
	h := New(t)
	f := h.Factory()
	ctx := ctxBG()
	if _, err := h.Pool.Exec(ctx, `DELETE FROM map_place_candidates`); err != nil {
		t.Fatal(err)
	}
	adm := h.NewUser("admin").Token

	d1, d2, d3 := f.Driver(), f.Driver(), f.Driver()
	var (
		s1 = dmSpot{35.4000, 38.4000} // ثلاثة سائقين في يوم → مرشَّح
		s2 = dmSpot{35.4100, 38.4000} // سائقان في يوم → لا
		s3 = dmSpot{35.4200, 38.4000} // ثلاثة سائقين لكن بجواره متجر → لا
		s4 = dmSpot{35.4300, 38.4000} // سائق واحد في ثلاثة أيام → مرشَّح
	)
	for _, d := range []string{d1.ID, d2.ID, d3.ID} {
		mpStand(t, h, d, s1, 1, 0)
		mpStand(t, h, d, s3, 1, 1)
	}
	mpStand(t, h, d1.ID, s2, 1, 2)
	mpStand(t, h, d2.ID, s2, 1, 2)
	for _, ago := range []int{1, 2, 3} {
		mpStand(t, h, d1.ID, s4, ago, 3)
	}
	store := f.Merchant()
	dmPlaceStore(t, h, store.ID, dmSpot{s3.lat + 0.0003, s3.lng}) // ~٣٣ م
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_place_candidates`) })

	res, err := maplearn.RunPlaces(ctx, h.Pool, maplearn.DefaultPlaceParams())
	if err != nil {
		t.Fatalf("الجولة: %v", err)
	}
	if res.StopPlaces < 2 {
		t.Fatalf("المتوقَّع مكانان على الأقل، والناتج %+v", res)
	}
	cs := mpList(t, h, adm, "?source=stops")
	c1, c4 := mpNear(cs, s1), mpNear(cs, s4)
	if c1 == nil || c1.Visits != 3 || c1.Drivers != 3 || c1.Source != "stops" {
		t.Fatalf("مكان ١: %+v", c1)
	}
	if c4 == nil || c4.Days != 3 || c4.Drivers != 1 {
		t.Fatalf("مكان ٤: %+v", c4)
	}
	if mpNear(cs, s2) != nil {
		t.Error("سائقان في يومٍ واحد لا يكفيان")
	}
	if mpNear(cs, s3) != nil {
		t.Error("بجوار متجرٍ معروف: لا يُقترح")
	}

	// موافقةٌ بلا اسمٍ مرفوضة؛ ومعها اسمٌ ونوعٌ تُدخل المعلم.
	path := func(id int64) string {
		return "/api/v1/admin/ops-map/place-candidates/" + strings.TrimSpace(string(mlJSON(id))) + "/decide"
	}
	if r := h.POST(path(c1.ID), adm, map[string]any{"approve": true, "name": " "}); r.Code != 422 {
		t.Errorf("موافقة بلا اسم = %d والمتوقَّع 422", r.Code)
	}
	if r := h.POST(path(c1.ID), adm, map[string]any{"approve": true, "name": "مكان التجربة", "kind": "نوع-فاسد"}); r.Code != 422 {
		t.Errorf("نوع فاسد = %d والمتوقَّع 422", r.Code)
	}
	if r := h.POST(path(c1.ID), adm, map[string]any{"approve": true, "name": "صيدلية التجربة", "kind": "pharmacy"}); r.Code != 200 {
		t.Fatalf("موافقة: %s", r)
	}
	if r := h.POST(path(c1.ID), adm, map[string]any{"approve": false}); r.Code != 409 {
		t.Errorf("قرار مكرَّر = %d والمتوقَّع 409", r.Code)
	}
	var kind, src string
	var dist float64
	if err := h.Pool.QueryRow(ctx, `
		SELECT kind, source, ST_Distance(point, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography)
		  FROM map_landmarks WHERE name = 'صيدلية التجربة'`, s1.lat, s1.lng).Scan(&kind, &src, &dist); err != nil {
		t.Fatalf("المعلم لم يدخل الجدول: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_landmarks WHERE name = 'صيدلية التجربة'`)
	})
	if kind != "amenity=pharmacy" || src != "drivers" || dist > 15 {
		t.Errorf("المعلم: kind=%s source=%s dist=%.1f", kind, src, dist)
	}

	// ورفضٌ لا يُعاد اقتراحُه.
	if r := h.POST(path(c4.ID), adm, map[string]any{"approve": false}); r.Code != 200 {
		t.Fatalf("رفض: %s", r)
	}
	if _, err := maplearn.RunPlaces(ctx, h.Pool, maplearn.DefaultPlaceParams()); err != nil {
		t.Fatal(err)
	}
	cs = mpList(t, h, adm, "?source=stops")
	if mpNear(cs, s1) != nil || mpNear(cs, s4) != nil {
		t.Errorf("المقرَّر عليه أُعيد اقتراحُه: %+v", cs)
	}
	if rej := mpList(t, h, adm, "?status=rejected"); mpNear(rej, s4) == nil {
		t.Error("المرفوض لا يظهر في تبويبه")
	}
}

func mpAddress(t *testing.T, h *Harness, text string, p dmSpot) string {
	t.Helper()
	cust := h.Customer()
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO user_addresses (user_id, label, address_text, area_building, location)
		VALUES ($1::uuid, 'البيت', $2, $2, ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography)`,
		cust.ID, text, p.lat, p.lng); err != nil {
		t.Fatalf("عنوان: %v", err)
	}
	return cust.ID
}

func mpOrderNote(t *testing.T, h *Harness, merchant, notes string, p dmSpot) {
	t.Helper()
	cust := h.Customer()
	var oid string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO orders (customer_id, merchant_id, status, address_text, notes, dropoff,
		                    payment_method, subtotal, delivery_fee, total, delivered_at, closed_at)
		VALUES ($1::uuid, $2::uuid, 'delivered', 'الرقة', $3,
		        ST_SetSRID(ST_MakePoint($5, $4), 4326)::geography, 'cash', 10, 5, 15, now(), now())
		RETURNING id::text`, cust.ID, merchant, notes, p.lat, p.lng).Scan(&oid); err != nil {
		t.Fatalf("طلب: %v", err)
	}
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM orders WHERE id = $1::uuid`, oid) })
}

func TestPlaces_AddressNamesDetector(t *testing.T) {
	h := New(t)
	f := h.Factory()
	ctx := ctxBG()
	if _, err := h.Pool.Exec(ctx, `DELETE FROM map_place_candidates`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_place_candidates`) })
	adm := h.NewUser("admin").Token
	m := f.Merchant()

	var (
		n1 = dmSpot{35.3000, 38.3000} // صيدلية الشفاء — عنوانان بصيغتين وزبونان
		n2 = dmSpot{35.3100, 38.3000} // فرن الأمل — من ملاحظات طلبين
		n3 = dmSpot{35.3200, 38.3000} // زبون واحد فقط
		n4 = dmSpot{35.3300, 38.3000} // اسمٌ موجودٌ في المعالم
	)
	mpAddress(t, h, "جانب صيدلية الشفاء، الطابق الثاني، جوال 0933123456", n1)
	mpAddress(t, h, "بجانب صيدليّه الشفاء", dmSpot{n1.lat + 0.0004, n1.lng})
	mpOrderNote(t, h, m.ID, "مقابل فرن الأمل", n2)
	mpOrderNote(t, h, m.ID, "قرب فرن الأمل لو سمحت", dmSpot{n2.lat, n2.lng + 0.0005})
	mpAddress(t, h, "عند مخبز الوحيد", n3)
	mpAddress(t, h, "قرب صيدلية النور", n4)
	mpAddress(t, h, "جانب صيدلية النور", n4)
	if _, err := h.Pool.Exec(ctx, `
		INSERT INTO map_landmarks (name, kind, point, source)
		VALUES ('صيدلية النور', 'amenity=pharmacy', ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, 'osm')`,
		n4.lat+0.0005, n4.lng); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_landmarks WHERE name = 'صيدلية النور'`)
	})

	if _, err := maplearn.RunPlaces(ctx, h.Pool, maplearn.DefaultPlaceParams()); err != nil {
		t.Fatalf("الجولة: %v", err)
	}
	cs := mpList(t, h, adm, "?source=addresses")
	ph := mpByName(cs, "صيدلية الشفاء")
	if ph == nil || ph.Customers != 2 || ph.Kind != "pharmacy" || mpNear([]mpCand{*ph}, n1) == nil {
		t.Fatalf("الصيدلية: %+v", ph)
	}
	for _, s := range ph.Samples {
		if strings.Contains(s, "0933123456") {
			t.Errorf("هاتفٌ في الدليل: %q", s)
		}
	}
	if len(ph.Samples) == 0 {
		t.Error("لا نصوص دليل")
	}
	fr := mpByName(cs, "فرن الأمل")
	if fr == nil || fr.Customers != 2 || fr.Kind != "shop" {
		t.Fatalf("الفرن: %+v", fr)
	}
	if mpByName(cs, "مخبز الوحيد") != nil {
		t.Error("زبون واحد لا يكفي")
	}
	if mpByName(cs, "صيدلية النور") != nil {
		t.Error("الاسم الموجود في المعالم لا يُقترح")
	}

	// موافقةٌ باسمٍ معدَّل → معلمٌ بمصدر drivers.
	r := h.POST("/api/v1/admin/ops-map/place-candidates/"+strings.TrimSpace(string(mlJSON(ph.ID)))+"/decide", adm,
		map[string]any{"approve": true, "name": "صيدلية الشفاء الجديدة", "kind": "pharmacy"})
	if r.Code != 200 {
		t.Fatalf("موافقة: %s", r)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_landmarks WHERE name = 'صيدلية الشفاء الجديدة'`)
	})
	var n int
	if err := h.Pool.QueryRow(ctx, `SELECT count(*)::int FROM map_landmarks WHERE name = 'صيدلية الشفاء الجديدة' AND source = 'drivers'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("المعلم المعدَّل: n=%d err=%v", n, err)
	}
	// ولا يُعاد اقتراح الاسم بعد الموافقة.
	if _, err := maplearn.RunPlaces(ctx, h.Pool, maplearn.DefaultPlaceParams()); err != nil {
		t.Fatal(err)
	}
	if mpByName(mpList(t, h, adm, "?source=addresses"), "صيدلية الشفاء") != nil {
		t.Error("أُعيد اقتراح اسمٍ وُوفق عليه")
	}
}

func TestPlaces_Permissions(t *testing.T) {
	h := New(t)
	for _, role := range []string{"customer", "driver", "finance"} {
		tok := h.NewUser(role).Token
		if r := h.GET("/api/v1/admin/ops-map/place-candidates", tok); r.Code != 403 {
			t.Errorf("%s: القائمة = %d والمتوقَّع 403", role, r.Code)
		}
		if r := h.POST("/api/v1/admin/ops-map/place-candidates/run", tok, nil); r.Code != 403 {
			t.Errorf("%s: التشغيل = %d والمتوقَّع 403", role, r.Code)
		}
	}
	ops := h.NewUser("operations").Token
	if r := h.POST("/api/v1/admin/ops-map/place-candidates/1/decide", ops, map[string]any{"approve": true, "name": "س س"}); r.Code != 403 {
		t.Errorf("العمليات تقرّر = %d والمتوقَّع 403", r.Code)
	}
	if r := h.POST("/api/v1/admin/ops-map/place-candidates/run", ops, nil); r.Code != 403 {
		t.Errorf("العمليات تشغّل = %d والمتوقَّع 403", r.Code)
	}
	adm := h.NewUser("admin").Token
	if r := h.POST("/api/v1/admin/ops-map/place-candidates/999999999/decide", adm, map[string]any{"approve": false}); r.Code != 404 {
		t.Errorf("مجهول = %d والمتوقَّع 404", r.Code)
	}
}
