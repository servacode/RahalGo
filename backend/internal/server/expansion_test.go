package server

// «طلباتُ التوسّع» — قرارُ المالك ٢٠٢٦-١٠-٠٤.
//
//   - الأرقامُ والجدولُ محافظة ← مدينة ← منطقة، **والملغاةُ لا تُحسب.**
//   - «بلّغ المنتظرين الآن» يدويٌّ: يُبلّغ منتظري المنطقة وحدَهم، مرّةً واحدة،
//     ويختم صفوفَهم.
//   - التذكيرُ يعدّ من صار مغطّىً ولم يُبلَّغ، وينقص بعد الضغط.
//   - ولا إبلاغَ آليّاً عند حفظ منطقةٍ أو إطلاق مدينة.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func expServer(t *testing.T) *Server {
	t.Helper()
	srv := overviewServer(t)
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv.notify = notifications.New(srv.pg, srv.hub, quiet)
	return srv
}

// expPlace مكانٌ معزولٌ لكلّ اختبار — محافظةٌ ومدينةٌ ومنطقةٌ نشطة.
type expFixture struct {
	gov, city, zone string
	lat, lng        float64
}

func expPlaceFixture(t *testing.T) expFixture {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	n := gnN()
	f := expFixture{lat: 33.10 + float64(n%40)*0.11, lng: 38.30}
	f.gov = gnGov(t, true)
	dist := gnDistrict(t, f.gov)
	if err := pool.QueryRow(ctx, `
		INSERT INTO cities (name, center, radius_m, active, district_id)
		VALUES ($1, ST_SetSRID(ST_MakePoint($3,$2),4326)::geography, 25000, true, $4) RETURNING id::text`,
		fmt.Sprintf("مدينةُ توسّعٍ %d", n), f.lat, f.lng, dist).Scan(&f.city); err != nil {
		t.Fatalf("city: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `DELETE FROM cities WHERE id=$1`, f.city) })
	if err := pool.QueryRow(ctx, `
		INSERT INTO delivery_zones (name, center, radius_m, active, city_id, shape)
		VALUES ($1, ST_SetSRID(ST_MakePoint($3,$2),4326)::geography, 2000, true, $4, 'radius') RETURNING id::text`,
		fmt.Sprintf("منطقةُ توسّعٍ %d", n), f.lat, f.lng, f.city).Scan(&f.zone); err != nil {
		t.Fatalf("zone: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `DELETE FROM delivery_zones WHERE id=$1`, f.zone) })
	return f
}

// expReq يُدرج طلباً بنقطةٍ وحالٍ وعدد.
func expReq(t *testing.T, kind, target string, lat, lng float64, active bool, requests int, cityID string) (string, string) {
	t.Helper()
	pool := testdb.Pool(t)
	uid := testdb.NewUser(t, pool, "customer")
	var city *string
	if cityID != "" {
		city = &cityID
	}
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO coverage_requests (user_id, at, kind, target_key, active, status, source, requests, city_id, address_text)
		VALUES ($1::uuid, ST_SetSRID(ST_MakePoint($3,$2),4326)::geography, $4, $5, $6, 'new', 'customer_app', $7, $8, 'حيّ التجربة')
		RETURNING id::text`, uid, lat, lng, kind, target, active, requests, city).Scan(&id); err != nil {
		t.Fatalf("expReq: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM coverage_requests WHERE id=$1`, id) })
	return id, uid
}

func readExpansion(t *testing.T, s *Server) expansionSummary {
	t.Helper()
	var sum expansionSummary
	if err := json.Unmarshal(dataOf(t, callAs(s.handleExpansion, "GET",
		"/admin/ops-map/expansion", "", "", nil)), &sum); err != nil {
		t.Fatalf("الملخّصُ لا يُقرأ: %v", err)
	}
	return sum
}

func rowOf(sum expansionSummary, key string) *expansionRow {
	for i := range sum.Rows {
		if sum.Rows[i].Key == key {
			return &sum.Rows[i]
		}
	}
	return nil
}

func pressNotify(t *testing.T, s *Server, key string) int {
	t.Helper()
	req := httptest.NewRequest("POST", "/admin/ops-map/expansion/notify",
		strings.NewReader(fmt.Sprintf(`{"key":%q}`, key)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handleExpansionNotify(w, req)
	var out struct {
		Notified int `json:"notified"`
	}
	if err := json.Unmarshal(dataOf(t, w), &out); err != nil {
		t.Fatalf("ردُّ الإبلاغ: %v", err)
	}
	return out.Notified
}

// ── ١ · التجميعُ والملغاةُ لا تُحسب ─────────────────────────────────────

func TestExpansion_GroupsAndExcludesCancelled(t *testing.T) {
	s := expServer(t)
	f := expPlaceFixture(t)
	cell := fmt.Sprintf("cell:exp:%d", gnN())
	expReq(t, "coverage_request", cell+"a", f.lat, f.lng, true, 3, f.city)
	expReq(t, "coverage_request", cell+"b", f.lat+0.001, f.lng, true, 1, f.city)
	expReq(t, "coverage_request", cell+"c", f.lat, f.lng+0.001, false, 5, f.city) // ملغى
	expReq(t, "service_interest", "city:"+f.city, f.lat, f.lng, true, 1, f.city)
	expReq(t, "service_interest", "city:"+f.city, f.lat, f.lng, false, 1, f.city) // ملغى

	sum := readExpansion(t, s)
	z := rowOf(sum, "zone:"+f.zone)
	if z == nil {
		t.Fatalf("لا صفَّ للمنطقة: %+v", sum.Rows)
	}
	if z.People != 2 || z.Requests != 4 {
		t.Fatalf("الملغى حُسب أو التجميعُ خطأ: أشخاص %d طلبات %d", z.People, z.Requests)
	}
	if z.CityID != f.city || z.GovernorateID != f.gov || z.ZoneID != f.zone {
		t.Fatalf("الهرمُ خطأ: %+v", z)
	}
	if z.Status != expReady || z.Notifiable != 2 {
		t.Fatalf("منطقةٌ مغطّاةٌ بمنتظرَين: حال %s يُبلَّغ %d", z.Status, z.Notifiable)
	}
	c := rowOf(sum, "city:"+f.city)
	if c == nil || c.People != 1 || c.Kind != "service_interest" {
		t.Fatalf("صفُّ «أخبرني» للمدينة: %+v", c)
	}
	if z.FirstAt.IsZero() || z.LastAt.Before(z.FirstAt) {
		t.Fatalf("أوّلُ وآخرُ طلب: %v %v", z.FirstAt, z.LastAt)
	}
}

// ── ٢ · الإبلاغُ اليدويّ ───────────────────────────────────────────────

func TestExpansion_ManualNotifyOnceForZoneOnly(t *testing.T) {
	s := expServer(t)
	f := expPlaceFixture(t)
	cell := fmt.Sprintf("cell:exp:%d", gnN())
	in1, u1 := expReq(t, "coverage_request", cell+"a", f.lat, f.lng, true, 1, f.city)
	in2, u2 := expReq(t, "coverage_request", cell+"b", f.lat+0.002, f.lng, true, 1, f.city)
	gone, _ := expReq(t, "coverage_request", cell+"c", f.lat, f.lng, false, 1, f.city)
	out, _ := expReq(t, "coverage_request", cell+"d", f.lat+0.06, f.lng, true, 1, f.city) // خارجَ المنطقة
	si, _ := expReq(t, "service_interest", "city:"+f.city, f.lat, f.lng, true, 1, f.city)

	if got := pressNotify(t, s, "zone:"+f.zone); got != 2 {
		t.Fatalf("يُبلَّغ منتظرا المنطقة وحدَهما: %d", got)
	}
	for _, id := range []string{in1, in2} {
		if !gnNotified(t, id) {
			t.Fatalf("لم يُختَم %s", id)
		}
	}
	for _, id := range []string{gone, out, si} {
		if gnNotified(t, id) {
			t.Fatalf("خُتم من ليس منتظراً في المنطقة: %s", id)
		}
	}
	pool := testdb.Pool(t)
	for _, u := range []string{u1, u2} {
		var n int
		pool.QueryRow(context.Background(), `
			SELECT count(*) FROM notifications WHERE user_id=$1::uuid AND entity='area_coverage'`, u).Scan(&n)
		if n != 1 {
			t.Fatalf("إشعارٌ واحدٌ لكلّ منتظر: %d", n)
		}
	}
	// **ومرّةً واحدة** — الضغطةُ الثانية لا تُبلّغ أحداً.
	if got := pressNotify(t, s, "zone:"+f.zone); got != 0 {
		t.Fatalf("الضغطةُ الثانيةُ أبلغت %d", got)
	}
	var n int
	pool.QueryRow(context.Background(), `
		SELECT count(*) FROM notifications WHERE user_id=$1::uuid AND entity='area_coverage'`, u1).Scan(&n)
	if n != 1 {
		t.Fatalf("أُبلغ مرّتين: %d", n)
	}
	// **والمدينةُ بزرّها** — «أخبرني» يُبلَّغ حين تُطلَق.
	if got := pressNotify(t, s, "city:"+f.city); got != 1 || !gnNotified(t, si) {
		t.Fatalf("إبلاغُ مشترِكي المدينة: %d", got)
	}
}

// ── ٣ · التذكير ────────────────────────────────────────────────────────

func TestExpansion_ReminderCountsCoveredNotNotified(t *testing.T) {
	s := expServer(t)
	read := func() (int, int) {
		var m struct {
			Pending int `json:"pending_notify"`
			Places  int `json:"places"`
		}
		if err := json.Unmarshal(dataOf(t, callAs(s.handleExpansionReminder, "GET",
			"/admin/ops-map/expansion/reminder", "", "", nil)), &m); err != nil {
			t.Fatal(err)
		}
		return m.Pending, m.Places
	}
	before, beforePlaces := read()
	f := expPlaceFixture(t)
	cell := fmt.Sprintf("cell:exp:%d", gnN())
	expReq(t, "coverage_request", cell+"a", f.lat, f.lng, true, 1, f.city)
	expReq(t, "coverage_request", cell+"b", f.lat, f.lng+0.002, true, 1, f.city)
	expReq(t, "coverage_request", cell+"c", f.lat, f.lng, false, 1, f.city) // ملغى

	mid, midPlaces := read()
	if mid-before != 2 || midPlaces-beforePlaces != 1 {
		t.Fatalf("التذكيرُ يعدّ المغطَّين غيرَ المُبلَّغين: %d→%d، أماكن %d→%d",
			before, mid, beforePlaces, midPlaces)
	}
	pressNotify(t, s, "zone:"+f.zone)
	after, _ := read()
	if after != before {
		t.Fatalf("بعد الإبلاغ يعود التذكير: %d بدل %d", after, before)
	}
}

// ── ٤ · رقمُ «ينتظرون ولم يُبلَّغوا» هو رقمُ الرئيسة ──────────────────────

func TestExpansion_WaitingEqualsHomeFigure(t *testing.T) {
	s := expServer(t)
	f := expPlaceFixture(t)
	cell := fmt.Sprintf("cell:exp:%d", gnN())
	expReq(t, "coverage_request", cell+"a", f.lat, f.lng, true, 1, f.city)
	expReq(t, "coverage_request", cell+"b", f.lat+0.06, f.lng, true, 1, f.city)
	expReq(t, "coverage_request", cell+"c", f.lat, f.lng, false, 1, f.city)

	sum := readExpansion(t, s)
	home := must(t, "expansion_waiting", readOverview(t, s).Awaiting.ExpansionWaiting)
	if int64(sum.WaitingNotNotified) != home {
		t.Fatalf("الصفحةُ %d والرئيسةُ %d", sum.WaitingNotNotified, home)
	}
}

// ── ٥ · لا إبلاغَ آليّاً ─────────────────────────────────────────────────

func TestExpansion_NoAutomaticNotifyOnSave(t *testing.T) {
	for _, file := range []string{"admin_zones_handlers.go", "city_handlers.go",
		"divisions_handlers.go", "opsmap_handlers.go"} {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, call := range []string{"s.notifyAreaCoverage(", "s.notifyCityLaunch(",
			"s.notifyGovernorateLaunch(", "s.notifyZoneWaiting("} {
			if strings.Contains(string(raw), call) {
				t.Errorf("%s يُبلّغ آليّاً (%s) — والإبلاغُ بزرٍّ يدويّ", file, call)
			}
		}
	}

}
