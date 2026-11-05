package qa

// ══════════════════════════════════════════════════════════════════════
// **معالمُ السائقين: «موجود من قبل» والعدُّ اليوميّ وتصحيحُ الموقع** (قرارُ المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// «لازم كل سائق باليوم يرسل اقل شي 5 معالم، واذا ارسلو شي قديم طالعه».
//
// **والنقاطُ هنا بعيدةٌ عن معالم الرقّة المبذورة** (الهجرة 0467: عرض ٣٥٫٦–٣٦٫٨) **وعن
// متاجر العُدّة** (٣٥٫٩٥، ٣٩٫٠١) — فلا يطابق معلمُ اختبارٍ معلماً حقيقيّاً. **ولكلّ اختبارٍ
// نقطتُه** على بُعد كيلومتر، **ولاسمه لاحقةٌ فريدة.**

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

// lmPost **يرسل معلماً كما يرسله تطبيقُ السائق.**
func lmPost(h *Harness, token, name, kind string, lat, lng float64) Res {
	h.T.Helper()
	return h.POST("/api/v1/driver/landmarks", token, map[string]any{"name": name, "kind": kind, "lat": lat, "lng": lng})
}

// lmDetails **تفصيلُ الخطأ كما تفكّه نسخةُ الميدان 0.7.0** — `Map<String,String>`؛ **ورقمٌ
// فيه يُسقط الفكَّ كلَّه** فيسقط الاختبار.
func lmDetails(t *testing.T, res Res) map[string]string {
	t.Helper()
	var env struct {
		Error struct {
			Code    string            `json:"code"`
			Details map[string]string `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatalf("**تفصيلُ الخطأ لا يُفكّ نصوصاً** (تسقط به 0.7.0): %v — %s", err, res)
	}
	return env.Error.Details
}

// lmMapLandmark **معلمٌ على الخريطة** — كما تكتبه الهجرةُ أو الموافقة.
func lmMapLandmark(t *testing.T, h *Harness, name, kind string, lat, lng float64) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO map_landmarks (name, kind, point, source)
		VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, 'osm')`, name, kind, lng, lat); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_landmarks WHERE name = $1`, name) })
}

// lmLastStatus **حالةُ آخر ما أرسله السائق.**
func lmLastStatus(t *testing.T, h *Harness, driverID string) string {
	t.Helper()
	var st string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT status FROM landmark_suggestions
		WHERE suggested_by = $1::uuid ORDER BY id DESC LIMIT 1`, driverID).Scan(&st); err != nil {
		t.Fatal(err)
	}
	return st
}

// lmToday **عدُّ اليوم في ردّ النجاح.**
func lmToday(t *testing.T, res Res) int {
	t.Helper()
	v, ok := res.JSON()["today"].(float64)
	if !ok {
		t.Fatalf("الردُّ بلا «today»: %s", res)
	}
	return int(v)
}

// TestLandmark_DuplicateOfMapLandmark **معلمٌ على الخريطة يُقال للسائق ولا يُعدّ** — «جامع النور»
// على بُعد ٤٠ م من «مسجد النور».
func TestLandmark_DuplicateOfMapLandmark(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	sfx := uniq("")
	const lat, lng = 34.6010, 40.2010
	onMap := "مسجد النور " + sfx
	lmMapLandmark(t, h, onMap, "amenity=place_of_worship", lat, lng)

	res := lmPost(h, drv.Token, "جامع النور "+sfx, "mosque", lat+0.00036, lng)
	if res.Code != 409 || res.Err() != "landmark_exists" {
		t.Fatalf("**المعلمُ الموجودُ على الخريطة انحسب جديداً**: %s", res)
	}
	d := lmDetails(t, res)
	if !strings.Contains(d["notice"], "موجود من قبل") || !strings.Contains(d["notice"], onMap) {
		t.Fatalf("النصُّ لا يقول ما هو الموجود: %q", d["notice"])
	}
	if d["today"] != "0" || d["quota"] != "5" || d["source"] != "map" || d["existing_name"] != onMap {
		t.Fatalf("التفصيل: %v", d)
	}
	if m, _ := strconv.Atoi(d["distance_m"]); m < 30 || m > 50 {
		t.Fatalf("المسافة %q والمنتظَرُ نحو ٤٠ م", d["distance_m"])
	}
	if st := lmLastStatus(t, h, drv.ID); st != "duplicate" {
		t.Fatalf("صفُّ المكرَّر حالتُه %q والمنتظَر duplicate", st)
	}
}

// TestLandmark_DuplicateOfOwnPending **ما أرسله هو قبلاً** — الثانيةُ «بعتّ هالمعلم قبل».
func TestLandmark_DuplicateOfOwnPending(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	sfx := uniq("")
	const lat, lng = 34.6110, 40.2110
	name := "صيدلية الأمان " + sfx
	first := lmPost(h, drv.Token, name, "pharmacy", lat, lng)
	if first.Code != 200 || first.JSON()["counted"] != true || lmToday(t, first) != 1 {
		t.Fatalf("الأوّلُ: %s", first)
	}
	again := lmPost(h, drv.Token, name, "pharmacy", lat+0.00009, lng)
	if again.Code != 409 || again.Err() != "landmark_exists" {
		t.Fatalf("**الثاني نفسُه انحسب مرّتين**: %s", again)
	}
	d := lmDetails(t, again)
	if d["source"] != "self" || !strings.Contains(d["notice"], "بعتّ هالمعلم قبل") || d["today"] != "1" {
		t.Fatalf("التفصيل: %v", d)
	}
	var pending, dup int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FILTER (WHERE status = 'pending'),
		count(*) FILTER (WHERE status = 'duplicate') FROM landmark_suggestions WHERE suggested_by = $1::uuid`,
		drv.ID).Scan(&pending, &dup)
	if pending != 1 || dup != 1 {
		t.Fatalf("بانتظار %d ومكرّر %d — والمنتظَر واحدٌ وواحد", pending, dup)
	}
}

// TestLandmark_DuplicateOfOtherDriverPending **ما أرسله سائقٌ قبله ولم يُقرَّر بعد.**
func TestLandmark_DuplicateOfOtherDriverPending(t *testing.T) {
	h := New(t)
	a, b := h.NewUser("driver"), h.NewUser("driver")
	sfx := uniq("")
	const lat, lng = 34.6210, 40.2210
	if res := lmPost(h, a.Token, "مدرسة الرشيد "+sfx, "school", lat, lng); res.Code != 200 {
		t.Fatalf("الأوّل: %s", res)
	}
	res := lmPost(h, b.Token, "المدرسة الرشيد "+sfx, "school", lat+0.00018, lng)
	if res.Code != 409 {
		t.Fatalf("**ما أرسله سائقٌ قبله انحسب للثاني**: %s", res)
	}
	d := lmDetails(t, res)
	if d["source"] != "suggestion" || !strings.Contains(d["notice"], "بانتظار الموافقة") || d["today"] != "0" {
		t.Fatalf("التفصيل: %v", d)
	}
}

// TestLandmark_DuplicateOfOurStore **متجرُنا معلمٌ على كلّ خريطة** — «ماركت الأمل» هو «سوبرماركت الأمل».
func TestLandmark_DuplicateOfOurStore(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	sfx := uniq("")
	const lat, lng = 34.6310, 40.2310
	m := h.Factory().Merchant()
	store := "سوبرماركت الأمل " + sfx
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET name = $2,
		location = ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography WHERE id = $1::uuid`, m.ID, store, lng, lat); err != nil {
		t.Fatal(err)
	}
	res := lmPost(h, drv.Token, "ماركت الأمل "+sfx, "shop", lat, lng+0.0003)
	if res.Code != 409 {
		t.Fatalf("**متجرُنا انحسب معلماً جديداً**: %s", res)
	}
	if d := lmDetails(t, res); d["source"] != "store" || d["existing_name"] != store {
		t.Fatalf("التفصيل: %v", d)
	}
}

// TestLandmark_RejectedIsNotExisting **المرفوضُ ليس موجوداً** — يُرسَل ثانيةً فيُعدّ، **والرفضُ
// يُنقص العدّ.**
func TestLandmark_RejectedIsNotExisting(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	admin := h.NewUser("admin")
	sfx := uniq("")
	const lat, lng = 34.6410, 40.2410
	name := "فرن السلام " + sfx
	first := lmPost(h, drv.Token, name, "shop", lat, lng)
	if first.Code != 200 || lmToday(t, first) != 1 {
		t.Fatalf("الأوّل: %s", first)
	}
	id, _ := first.JSON()["id"].(float64)
	if d := h.POST("/api/v1/admin/ops-map/landmark-suggestions/"+itoa(int(id))+"/decide", admin.Token,
		map[string]any{"approve": false}); d.Code != 200 {
		t.Fatalf("الرفض: %s", d)
	}
	again := lmPost(h, drv.Token, name, "shop", lat, lng)
	if again.Code != 200 || again.JSON()["counted"] != true {
		t.Fatalf("**المرفوضُ حُسب موجوداً**: %s", again)
	}
	// **والرفضُ أنقص العدّ** — الجديدُ وحدَه محسوب.
	if n := lmToday(t, again); n != 1 {
		t.Fatalf("العدُّ بعد الرفض %d والمنتظَر ١", n)
	}
}

// TestLandmark_FarOrDifferentCounts **الجديدُ يُعدّ**: الاسمُ نفسُه على ٣٠٠ م، واسمٌ آخر في
// النقطة نفسِها، و«صيدلية الشفاء» جنب «مشفى الشفاء».
func TestLandmark_FarOrDifferentCounts(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	sfx := uniq("")
	const lat, lng = 34.6510, 40.2510
	lmMapLandmark(t, h, "مشفى الشفاء "+sfx, "amenity=hospital", lat+0.01, lng)
	steps := []struct {
		name, kind string
		lat        float64
	}{
		{"جامع الفاروق " + sfx, "mosque", lat},
		{"جامع الفاروق " + sfx, "mosque", lat + 0.0027}, // ~٣٠٠ م
		{"جامع عمر " + sfx, "mosque", lat},
		{"صيدلية الشفاء " + sfx, "pharmacy", lat + 0.01},
	}
	for i, s := range steps {
		res := lmPost(h, drv.Token, s.name, s.kind, s.lat, lng)
		if res.Code != 200 || res.JSON()["counted"] != true {
			t.Fatalf("الخطوة %d «%s»: %s", i+1, s.name, res)
		}
		if n := lmToday(t, res); n != i+1 {
			t.Fatalf("الخطوة %d: العدّ %d والمنتظَر %d", i+1, n, i+1)
		}
		if q, _ := res.JSON()["quota"].(float64); q != 5 {
			t.Fatalf("المطلوبُ %v والمنتظَر ٥", res.JSON()["quota"])
		}
		if notice, _ := res.JSON()["notice"].(string); !strings.Contains(notice, "معالمك اليوم: "+itoa(i+1)+" من 5") {
			t.Fatalf("نصُّ النجاح: %q", notice)
		}
	}
}

// TestLandmark_DetailsAreAllStrings **تفصيلُ الـ٤٠٩ نصوصٌ كلُّه** — نسخةُ الميدان تفكّه
// `Map<String,String>`. **و`bad_point` للمعلم يقول ما يفعل.**
func TestLandmark_DetailsAreAllStrings(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	sfx := uniq("")
	const lat, lng = 34.6610, 40.2610
	lmMapLandmark(t, h, "دوار النعيم "+sfx, "junction=roundabout", lat, lng)
	res := lmPost(h, drv.Token, "دوار النعيم "+sfx, "other", lat, lng)
	if res.Code != 409 {
		t.Fatalf("«غير ذلك» بكلمة «دوار» لم يطابق الدوّار: %s", res)
	}
	d := lmDetails(t, res)
	for _, k := range []string{"notice", "today", "quota", "existing_name", "distance_m", "source"} {
		if strings.TrimSpace(d[k]) == "" {
			t.Fatalf("التفصيلُ بلا %q: %v", k, d)
		}
	}
	bad := lmPost(h, drv.Token, "جامع "+sfx, "mosque", 0, 0)
	if bad.Code != 422 || bad.Err() != "bad_point" {
		t.Fatalf("نقطةٌ صفر: %s", bad)
	}
	if n := lmDetails(t, bad)["notice"]; !strings.Contains(n, "موقعك") {
		t.Fatalf("نصُّ «bad_point» للمعلم: %q", n)
	}
}

// TestLandmark_StalePointFixedFromTrack **نقطةُ التطبيق القديمة تُصحَّح من نبضة السائق** — آخرُ
// تسليمٍ بين طلبين على بُعد ٢ كم. **والنبضةُ القديمةُ أو المزيَّفةُ لا تصحّح شيئاً.**
func TestLandmark_StalePointFixedFromTrack(t *testing.T) {
	h := New(t)
	sfx := uniq("")
	const lat, lng = 34.6710, 40.2710
	ping := func(driverID, ago string, mocked bool) {
		t.Helper()
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO driver_track (driver_id, at, recorded_at, accuracy_m, mocked)
			VALUES ($1::uuid, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, now() - $4::interval, 10, $5)`,
			driverID, lng, lat, ago, mocked); err != nil {
			t.Fatal(err)
		}
	}
	type row struct {
		toPing, toSent float64
		fromPing       bool
		gap            *int
	}
	read := func(id float64) row {
		t.Helper()
		var r row
		if err := h.Pool.QueryRow(ctxBG(), `
			SELECT ST_Distance(point, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography),
			       COALESCE(ST_Distance(sent_point, ST_SetSRID(ST_MakePoint($2, $4), 4326)::geography), -1),
			       point_from_ping, ping_gap_m
			  FROM landmark_suggestions WHERE id = $1`, int64(id), lng, lat, lat+0.018).Scan(
			&r.toPing, &r.toSent, &r.fromPing, &r.gap); err != nil {
			t.Fatal(err)
		}
		return r
	}

	// ١ · نبضةٌ عمرُها ٣٠ ثانية ونقطةُ التطبيق على ٢ كم ⇒ يُحفظ على النبضة.
	fresh := h.NewUser("driver")
	ping(fresh.ID, "30 seconds", false)
	res := lmPost(h, fresh.Token, "جامع الحديقة "+sfx, "mosque", lat+0.018, lng)
	if res.Code != 200 || res.JSON()["point_from_ping"] != true {
		t.Fatalf("الإرسال: %s", res)
	}
	id, _ := res.JSON()["id"].(float64)
	if r := read(id); r.toPing > 1 || !r.fromPing || r.toSent < 0 || r.toSent > 1 || r.gap == nil || *r.gap < 1900 || *r.gap > 2100 {
		t.Fatalf("**حُفظ المعلمُ على نقطة التطبيق القديمة**: %+v", r)
	}

	// ٢ · نبضةٌ عمرُها خمسُ دقائق ⇒ كما جاء، «بلا تحقّق».
	stale := h.NewUser("driver")
	ping(stale.ID, "5 minutes", false)
	res = lmPost(h, stale.Token, "جامع الحديقة الثاني "+sfx, "mosque", lat+0.018, lng)
	id, _ = res.JSON()["id"].(float64)
	if r := read(id); res.Code != 200 || r.fromPing || r.toPing < 1900 || r.gap != nil {
		t.Fatalf("نبضةٌ قديمةٌ صحّحت الموقع: %s %+v", res, r)
	}

	// ٣ · نبضةٌ مزيَّفة ⇒ لا تُصحّح.
	fake := h.NewUser("driver")
	ping(fake.ID, "10 seconds", true)
	res = lmPost(h, fake.Token, "جامع الحديقة الثالث "+sfx, "mosque", lat+0.018, lng)
	id, _ = res.JSON()["id"].(float64)
	if r := read(id); res.Code != 200 || r.fromPing {
		t.Fatalf("نبضةٌ مزيَّفةٌ صحّحت الموقع: %+v", r)
	}

	// ٤ · نبضةٌ حديثةٌ قريبة (٥٠ م) ⇒ تبقى نقطةُ التطبيق، **والبُعدُ يُحفظ تحقّقاً.**
	near := h.NewUser("driver")
	ping(near.ID, "20 seconds", false)
	res = lmPost(h, near.Token, "جامع الحديقة الرابع "+sfx, "mosque", lat+0.00045, lng)
	id, _ = res.JSON()["id"].(float64)
	if r := read(id); res.Code != 200 || r.fromPing || r.gap == nil || *r.gap < 40 || *r.gap > 60 {
		t.Fatalf("النبضةُ القريبة: %s %+v", res, r)
	}
}
