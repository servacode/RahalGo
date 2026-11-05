// Package drivermonitor **مراقبة حركة السائقين لمنع الاحتيال** (قرار المالك ٢٠٢٦-١١-٠٣).
//
// «بدنا نعرف كل سائق كم كيلو مشي باليوم، بطلب وبلا طلب، وهل بيتحرك بلا طلب،
// مثلا بيوصّل طلب من ورا ضهري. هاد حقي لأحمي حالي.»
//
// ثلاثة أجزاء:
//
//   - `trail.go`   الكتابة: جدول `driver_trail` (تاريخ، مخفَّف، يُقلَّم حسب الإعداد)
//   - `analyze.go` التحليل: دالة صافية على نقاط ونوافذ طلبات ومتاجر
//   - `load.go`    القراءة: يجمع المدخلات من القاعدة ليوم دمشقيّ
//
// **وهي إشارة لا دليل**: السائق قد يقف عند المطعم ليأكل. الحكم للمالك بعد ما يسأله.
// **و`driver_track` (ساعتان، للتوزيع) ما بيتغيّر** — هاد جدول تاني.
package drivermonitor

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/servacode/rahalgo/backend/internal/routing"
)

const (
	// KeepEvery أقل فاصل زمني بين نقطتين محفوظتين.
	KeepEvery = 20 * time.Second
	// KeepMoveM أو مسافة تحرّك تكفي لحفظ النقطة قبل الفاصل.
	KeepMoveM = 25.0

	// DefaultRetentionDays مدة الحفظ إن لم يُقرأ الإعداد.
	DefaultRetentionDays = 30
)

// Point نقطة بوقت التقاطها.
type Point struct {
	At        time.Time
	Lat, Lng  float64
	AccuracyM *float64
	SpeedMps  *float64
	Mocked    bool
	// BearingDeg اتجاه الجهاز — **لنقطة العرض وحدها** (`display.go`)؛ الأثر ما بيخزّنه.
	BearingDeg *float64
}

func (p Point) pos() routing.Point { return routing.Point{Lat: p.Lat, Lng: p.Lng} }

// Dist المسافة بالمتر بين نقطتين.
func Dist(a, b Point) float64 { return routing.MetersBetween(a.pos(), b.pos()) }

// Keep **هل تُحفظ هذه النقطة؟** — دالة صافية.
//
// تُحفظ إن: أول نقطة، أو مزيَّفة (دائما)، أو مرّ ≥ ٢٠ ثانية على آخر محفوظة (وهذا يشمل
// أي انقطاع)، أو تحرّكت ≥ ٢٥ متر. **والواقف على نبضة الدقيقتين تُحفظ كلها** فيبقى الوقوف مقيسا.
func Keep(prev *Point, p Point) bool {
	if prev == nil || p.Mocked {
		return true
	}
	if p.At.Sub(prev.At) >= KeepEvery {
		return true
	}
	return Dist(*prev, p) >= KeepMoveM
}

// Filter يمرّر نقاطا مرتّبة بالزمن على `Keep` بدءا من `prev`.
func Filter(prev *Point, pts []Point) []Point {
	out := make([]Point, 0, len(pts))
	for _, p := range pts {
		if !Keep(prev, p) {
			continue
		}
		q := p
		prev = &q
		out = append(out, p)
	}
	return out
}

// DB ما تحتاجه الكتابة والقراءة من القاعدة (مجمّع اتصالات أو معاملة).
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Record **يكتب نقاط سائق** (واحدة أو دفعة) بعد التخفيف. **يرجّع كم كُتب.**
//
// النقاط تُرتَّب هنا بالزمن، والمقارنة تبدأ من آخر نقطة محفوظة قبل أولاها.
func Record(ctx context.Context, db DB, driverID string, pts []Point) (int, error) {
	if len(pts) == 0 {
		return 0, nil
	}
	sorted := append([]Point(nil), pts...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].At.Before(sorted[j].At) })

	var prev *Point
	var p0 Point
	err := db.QueryRow(ctx, `
		SELECT recorded_at, lat, lng FROM driver_trail
		 WHERE driver_id = $1::uuid AND recorded_at <= $2
		 ORDER BY recorded_at DESC LIMIT 1`, driverID, sorted[0].At).Scan(&p0.At, &p0.Lat, &p0.Lng)
	switch err {
	case nil:
		prev = &p0
	case pgx.ErrNoRows:
	default:
		return 0, err
	}
	keep := Filter(prev, sorted)
	if len(keep) == 0 {
		return 0, nil
	}
	b := &pgx.Batch{}
	for _, p := range keep {
		b.Queue(`
			INSERT INTO driver_trail (driver_id, recorded_at, lat, lng, accuracy_m, speed_mps, mocked)
			VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (driver_id, recorded_at) DO NOTHING`,
			driverID, p.At, p.Lat, p.Lng, p.AccuracyM, p.SpeedMps, p.Mocked)
	}
	br, ok := db.(interface {
		SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	})
	if !ok {
		for _, p := range keep {
			if _, err := db.Exec(ctx, `
				INSERT INTO driver_trail (driver_id, recorded_at, lat, lng, accuracy_m, speed_mps, mocked)
				VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
				ON CONFLICT (driver_id, recorded_at) DO NOTHING`,
				driverID, p.At, p.Lat, p.Lng, p.AccuracyM, p.SpeedMps, p.Mocked); err != nil {
				return 0, err
			}
		}
		return len(keep), nil
	}
	res := br.SendBatch(ctx, b)
	for range keep {
		if _, err := res.Exec(); err != nil {
			_ = res.Close()
			return 0, err
		}
	}
	if err := res.Close(); err != nil {
		return 0, err
	}
	return len(keep), nil
}

// Prune **يحذف ما جاوز مدة الحفظ** — لكل السائقين بحذف واحد على فهرس الزمن.
func Prune(ctx context.Context, db DB, retentionDays int) (int64, error) {
	if retentionDays < 1 {
		retentionDays = DefaultRetentionDays
	}
	tag, err := db.Exec(ctx, `
		DELETE FROM driver_trail
		 WHERE recorded_at < now() - make_interval(days => $1::int)`, retentionDays)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

var pruneGate struct {
	sync.Mutex
	last time.Time
}

// PruneDue **أحان التقليم؟** — مرة في الساعة للمنصّة كلها، لا مع كل نبضة.
func PruneDue(now time.Time) bool {
	pruneGate.Lock()
	defer pruneGate.Unlock()
	if now.Sub(pruneGate.last) < time.Hour {
		return false
	}
	pruneGate.last = now
	return true
}
