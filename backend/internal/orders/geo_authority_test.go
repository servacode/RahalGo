package orders

// Batch 3a — سلطةُ الجغرافيا الإداريّة عند الإنشاء (اختباراتٌ سلبيّةٌ حقيقيّةٌ على القاعدة).
//
// **الحكمُ للأبِ لا للابن**: منطقةٌ نشطةٌ تحت مدينةٍ/محافظةٍ مُطفأةٍ لا تفتح الباب.
// **ولا يُطفَأ صفُّ المنطقة** — الحكمُ لحظيٌّ من حالة الأب، فيُختبَر أنّ `active`
// الابنِ لم يتبدّل.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

var geoNonce int64

func nonce() int64 { return atomic.AddInt64(&geoNonce, 1) }

// --- fixtures ---------------------------------------------------------------

func mkGov(t *testing.T, active bool) string {
	t.Helper()
	pool := testdb.Pool(t)
	var id string
	name := fmt.Sprintf("محافظةُ اختبار %d", nonce())
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO governorates (name, active) VALUES ($1,$2) RETURNING id::text`, name, active).Scan(&id); err != nil {
		t.Fatalf("mkGov: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM governorates WHERE id=$1`, id) })
	return id
}

func mkDistrict(t *testing.T, govID string) string {
	t.Helper()
	pool := testdb.Pool(t)
	var id string
	name := fmt.Sprintf("منطقةُ اختبار %d", nonce())
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO districts (governorate_id, name, active) VALUES ($1,$2,true) RETURNING id::text`, govID, name).Scan(&id); err != nil {
		t.Fatalf("mkDistrict: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM districts WHERE id=$1`, id) })
	return id
}

func mkCity(t *testing.T, districtID string, lat, lng float64, active bool) string {
	t.Helper()
	pool := testdb.Pool(t)
	var id string
	name := fmt.Sprintf("مدينةُ اختبار %d", nonce())
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO cities (name, center, radius_m, active, district_id)
		VALUES ($1, ST_SetSRID(ST_MakePoint($3,$2),4326)::geography, 25000, $4, $5) RETURNING id::text`,
		name, lat, lng, active, districtID).Scan(&id); err != nil {
		t.Fatalf("mkCity: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM cities WHERE id=$1`, id) })
	return id
}

func mkZone(t *testing.T, cityID string, lat, lng float64, active, hoursEnforced bool) string {
	t.Helper()
	pool := testdb.Pool(t)
	var id string
	name := fmt.Sprintf("زونُ اختبار %d", nonce())
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO delivery_zones (name, center, radius_m, active, city_id, shape, hours_enforced)
		VALUES ($1, ST_SetSRID(ST_MakePoint($3,$2),4326)::geography, 3000, $4, $5, 'radius', $6) RETURNING id::text`,
		name, lat, lng, active, cityID, hoursEnforced).Scan(&id); err != nil {
		t.Fatalf("mkZone: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM delivery_zones WHERE id=$1`, id) })
	return id
}

func newGeoService(t *testing.T) *Service {
	t.Helper()
	pool := testdb.Pool(t)
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewService(pool, nil, nil, nil, nil, quiet)
}

// --- tests ------------------------------------------------------------------

// نقطةٌ معزولةٌ بعيدةٌ عن مُدُن البذور (0151) — ومركزُ مدينةِ الاختبار عليها تماماً
// فتفوز دائماً بأقربيّةِ classifyPlace، فلا تتداخل بياناتُ البذور مع الحكم.
const rqLat, rqLng = 33.30, 42.55

func TestRequirePlaceLaunched_ParentAuthority(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	s := newGeoService(t)

	gov := mkGov(t, true)
	dist := mkDistrict(t, gov)
	city := mkCity(t, dist, rqLat, rqLng, true)
	zone := mkZone(t, city, rqLat, rqLng, true, false)

	// all launched -> nil
	if err := s.requirePlaceLaunched(ctx, pool, rqLat, rqLng); err != nil {
		t.Fatalf("all-active must pass: %v", err)
	}
	// disable CITY (zone stays active) -> city_not_supported
	if _, err := pool.Exec(ctx, `UPDATE cities SET active=false WHERE id=$1`, city); err != nil {
		t.Fatal(err)
	}
	if err := s.requirePlaceLaunched(ctx, pool, rqLat, rqLng); !errors.Is(err, ErrCityNotSupported) {
		t.Fatalf("disabled city must reject with ErrCityNotSupported, got %v", err)
	}
	// zone.active must be UNCHANGED (no destructive cascade)
	var zoneActive bool
	pool.QueryRow(ctx, `SELECT active FROM delivery_zones WHERE id=$1`, zone).Scan(&zoneActive)
	if !zoneActive {
		t.Fatal("child zone.active must be preserved (no destructive cascade)")
	}
	// re-enable city -> passes again (zero zone restoration needed)
	pool.Exec(ctx, `UPDATE cities SET active=true WHERE id=$1`, city)
	if err := s.requirePlaceLaunched(ctx, pool, rqLat, rqLng); err != nil {
		t.Fatalf("re-enabled city must pass: %v", err)
	}
	// disable GOVERNORATE (city+zone active) -> province_not_supported
	pool.Exec(ctx, `UPDATE governorates SET active=false WHERE id=$1`, gov)
	if err := s.requirePlaceLaunched(ctx, pool, rqLat, rqLng); !errors.Is(err, ErrProvinceNotSupported) {
		t.Fatalf("disabled governorate must reject with ErrProvinceNotSupported, got %v", err)
	}
	pool.QueryRow(ctx, `SELECT active FROM delivery_zones WHERE id=$1`, zone).Scan(&zoneActive)
	if !zoneActive {
		t.Fatal("child zone.active must be preserved under disabled governorate")
	}
}

func TestRequirePlaceLaunched_AreaUnknown(t *testing.T) {
	ctx := context.Background()
	s := newGeoService(t)
	// a point with no city containing it -> area_not_supported (Atlantic)
	if err := s.requirePlaceLaunched(ctx, testdb.Pool(t), 10.0, -30.0); !errors.Is(err, ErrAreaNotSupported) {
		t.Fatalf("unknown area must reject with ErrAreaNotSupported, got %v", err)
	}
}

func TestCoverableAt_EffectiveAuthority_IgnoresHours(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	s := newGeoService(t)
	gov := mkGov(t, true)
	dist := mkDistrict(t, gov)
	city := mkCity(t, dist, rqLat, rqLng, true)
	// zone with hours ENFORCED (and no open windows => clock-closed) but geometry covers.
	mkZone(t, city, rqLat, rqLng, true, true)

	// clock-closed zone is still "coverable" (geographic activation, not the clock)
	ok, err := s.CoverableAt(ctx, pool, rqLat, rqLng)
	if err != nil || !ok {
		t.Fatalf("all-active zone (even hours-closed) must be coverable, got ok=%v err=%v", ok, err)
	}
	// disable city -> NOT coverable (active zone + inactive city)
	pool.Exec(ctx, `UPDATE cities SET active=false WHERE id=$1`, city)
	ok, err = s.CoverableAt(ctx, pool, rqLat, rqLng)
	if err != nil || ok {
		t.Fatalf("active zone under inactive city must NOT be coverable, got ok=%v err=%v", ok, err)
	}
	// disable gov instead -> NOT coverable
	pool.Exec(ctx, `UPDATE cities SET active=true WHERE id=$1`, city)
	pool.Exec(ctx, `UPDATE governorates SET active=false WHERE id=$1`, gov)
	ok, _ = s.CoverableAt(ctx, pool, rqLat, rqLng)
	if ok {
		t.Fatal("active zone under inactive governorate must NOT be coverable")
	}
}

func TestCityEffectivelyLaunched(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	s := newGeoService(t)
	gov := mkGov(t, true)
	dist := mkDistrict(t, gov)
	city := mkCity(t, dist, rqLat, rqLng, true)

	ok, err := s.CityEffectivelyLaunched(ctx, pool, city)
	if err != nil || !ok {
		t.Fatalf("city+gov active must be effectively launched: ok=%v err=%v", ok, err)
	}
	pool.Exec(ctx, `UPDATE governorates SET active=false WHERE id=$1`, gov)
	ok, _ = s.CityEffectivelyLaunched(ctx, pool, city)
	if ok {
		t.Fatal("city active but governorate inactive must NOT be effectively launched")
	}
	pool.Exec(ctx, `UPDATE governorates SET active=true WHERE id=$1`, gov)
	pool.Exec(ctx, `UPDATE cities SET active=false WHERE id=$1`, city)
	ok, _ = s.CityEffectivelyLaunched(ctx, pool, city)
	if ok {
		t.Fatal("inactive city must NOT be effectively launched")
	}
}
