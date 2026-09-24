package server

// Batch 3d — إشعارُ الإطلاق/التغطية: السلطةُ الكاملةُ للهرم + مرّةً واحدة.
//
// **الاختباراتُ السلبيّةُ المطلوبة**:
//   - منطقةٌ نشطةٌ + مدينةٌ مُطفأةٌ  ⇒ لا إشعارَ تغطية.
//   - مدينةٌ نشطةٌ + محافظةٌ مُطفأةٌ ⇒ لا إشعارَ إطلاقِ مدينة.
// **والختمُ `notified_at` هو دليلُ الإرسال** (الختمُ قفلٌ سابقٌ للإرسال) —
// فيُختبَر عبره أنّ الإرسالَ وقع/لم يقع، وأنّه مرّةٌ واحدةٌ لا تتكرّر.

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

var gnNonce int64

func gnN() int64 { return atomic.AddInt64(&gnNonce, 1) }

const gnLat, gnLng = 33.31, 42.56 // نقطةٌ معزولة

func geoNotifyServer(t *testing.T) *Server {
	t.Helper()
	pool := testdb.Pool(t)
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	hub := realtime.NewHub(quiet)
	return &Server{
		pg:     pool,
		logger: quiet,
		notify: notifications.New(pool, hub, quiet),
		orders: orders.NewService(pool, nil, nil, nil, nil, quiet),
	}
}

func gnGov(t *testing.T, active bool) string {
	pool := testdb.Pool(t)
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO governorates (name, active) VALUES ($1,$2) RETURNING id::text`,
		fmt.Sprintf("محافظةُ 3d %d", gnN()), active).Scan(&id); err != nil {
		t.Fatalf("gnGov: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM governorates WHERE id=$1`, id) })
	return id
}

func gnDistrict(t *testing.T, govID string) string {
	pool := testdb.Pool(t)
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO districts (governorate_id, name, active) VALUES ($1,$2,true) RETURNING id::text`,
		govID, fmt.Sprintf("منطقةُ 3d %d", gnN())).Scan(&id); err != nil {
		t.Fatalf("gnDistrict: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM districts WHERE id=$1`, id) })
	return id
}

func gnCity(t *testing.T, distID string, active bool) string {
	pool := testdb.Pool(t)
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO cities (name, center, radius_m, active, district_id)
		VALUES ($1, ST_SetSRID(ST_MakePoint($3,$2),4326)::geography, 25000, $4, $5) RETURNING id::text`,
		fmt.Sprintf("مدينةُ 3d %d", gnN()), gnLat, gnLng, active, distID).Scan(&id); err != nil {
		t.Fatalf("gnCity: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM cities WHERE id=$1`, id) })
	return id
}

func gnZone(t *testing.T, cityID string, active bool) string {
	pool := testdb.Pool(t)
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO delivery_zones (name, center, radius_m, active, city_id, shape)
		VALUES ($1, ST_SetSRID(ST_MakePoint($3,$2),4326)::geography, 3000, $4, $5, 'radius') RETURNING id::text`,
		fmt.Sprintf("زونُ 3d %d", gnN()), gnLat, gnLng, active, cityID).Scan(&id); err != nil {
		t.Fatalf("gnZone: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM delivery_zones WHERE id=$1`, id) })
	return id
}

// gnSub inserts a demand row; returns its id. kind = service_interest | coverage_request.
func gnSub(t *testing.T, kind, targetKey string) (reqID, userID string) {
	pool := testdb.Pool(t)
	userID = testdb.NewUser(t, pool, "customer")
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO coverage_requests (user_id, at, kind, target_key, active, status, source)
		VALUES ($1::uuid, ST_SetSRID(ST_MakePoint($3,$2),4326)::geography, $4, $5, true, 'new', 'customer_app')
		RETURNING id::text`, userID, gnLat, gnLng, kind, targetKey).Scan(&reqID); err != nil {
		t.Fatalf("gnSub: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM coverage_requests WHERE id=$1`, reqID) })
	return reqID, userID
}

func gnNotified(t *testing.T, reqID string) bool {
	var notified bool
	testdb.Pool(t).QueryRow(context.Background(),
		`SELECT notified_at IS NOT NULL FROM coverage_requests WHERE id=$1`, reqID).Scan(&notified)
	return notified
}

// --- city-launch -----------------------------------------------------------

func TestNotifyCityLaunch_EffectiveAndIdempotent(t *testing.T) {
	s := geoNotifyServer(t)
	ctx := context.Background()
	gov := gnGov(t, true)
	dist := gnDistrict(t, gov)
	city := gnCity(t, dist, true)
	req, _ := gnSub(t, "service_interest", "city:"+city)

	// effectively launched (city+gov active) -> notified once
	s.notifyCityLaunch(ctx, city)
	if !gnNotified(t, req) {
		t.Fatal("effectively-launched city must notify its waitlist")
	}
	// idempotent: second call does not re-open (still notified, no error/duplicate churn)
	var before string
	s.pg.QueryRow(ctx, `SELECT notified_at::text FROM coverage_requests WHERE id=$1`, req).Scan(&before)
	s.notifyCityLaunch(ctx, city)
	var after string
	s.pg.QueryRow(ctx, `SELECT notified_at::text FROM coverage_requests WHERE id=$1`, req).Scan(&after)
	if before != after {
		t.Fatalf("idempotent: notified_at must not change on repeat (%s -> %s)", before, after)
	}
}

// REQUIRED NEGATIVE: active city + inactive governorate -> NO city-launch notification.
func TestNotifyCityLaunch_InactiveGovernorate_NoNotify(t *testing.T) {
	s := geoNotifyServer(t)
	ctx := context.Background()
	gov := gnGov(t, false) // governorate OFF
	dist := gnDistrict(t, gov)
	city := gnCity(t, dist, true) // city ON
	req, _ := gnSub(t, "service_interest", "city:"+city)

	s.notifyCityLaunch(ctx, city)
	if gnNotified(t, req) {
		t.Fatal("active city under INACTIVE governorate must NOT notify (not effectively launched)")
	}
}

func TestNotifyGovernorateLaunch_NotifiesChildCities(t *testing.T) {
	s := geoNotifyServer(t)
	ctx := context.Background()
	gov := gnGov(t, true)
	dist := gnDistrict(t, gov)
	city := gnCity(t, dist, true)
	req, _ := gnSub(t, "service_interest", "city:"+city)

	s.notifyGovernorateLaunch(ctx, gov)
	if !gnNotified(t, req) {
		t.Fatal("governorate launch must notify active child-city waitlists")
	}
}

// --- area-coverage ---------------------------------------------------------

func TestNotifyAreaCoverage_EffectiveAndIdempotent(t *testing.T) {
	s := geoNotifyServer(t)
	ctx := context.Background()
	gov := gnGov(t, true)
	dist := gnDistrict(t, gov)
	city := gnCity(t, dist, true)
	gnZone(t, city, true) // active zone covering the point
	req, _ := gnSub(t, "coverage_request", fmt.Sprintf("cell:3d:%d", gnN()))

	s.notifyAreaCoverage(ctx)
	if !gnNotified(t, req) {
		t.Fatal("point under active gov+city+zone must be notified as covered")
	}
	// idempotent
	var before, after string
	s.pg.QueryRow(ctx, `SELECT notified_at::text FROM coverage_requests WHERE id=$1`, req).Scan(&before)
	s.notifyAreaCoverage(ctx)
	s.pg.QueryRow(ctx, `SELECT notified_at::text FROM coverage_requests WHERE id=$1`, req).Scan(&after)
	if before != after {
		t.Fatalf("idempotent: coverage notified_at must not change on repeat")
	}
}

// REQUIRED NEGATIVE: active zone + inactive city -> NO coverage notification.
func TestNotifyAreaCoverage_InactiveCity_NoNotify(t *testing.T) {
	s := geoNotifyServer(t)
	ctx := context.Background()
	gov := gnGov(t, true)
	dist := gnDistrict(t, gov)
	city := gnCity(t, dist, false) // city OFF
	gnZone(t, city, true)          // zone ON
	req, _ := gnSub(t, "coverage_request", fmt.Sprintf("cell:3d:%d", gnN()))

	s.notifyAreaCoverage(ctx)
	if gnNotified(t, req) {
		t.Fatal("active zone under INACTIVE city must NOT trigger coverage notification")
	}
}
