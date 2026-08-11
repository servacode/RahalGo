package server

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// ══════════════════════════════════════════════════════════════════════
// **الموقعُ بالجملة — والأثرُ يُقرأ بزمن الجهاز**
// ══════════════════════════════════════════════════════════════════════
//
// **ما يُحرَس هنا ثلاثةٌ، وسقوطُ أيّها يُفسد الإسنادَ لا الرسمَ:**
//
// **الأوّل** أنّ كلَّ نقطةٍ تحتفظ بزمنها — **وبلاه تُسجَّل عشرون نقطةً في
// ثانيةٍ واحدة**، فيبدو من وقف عشرين دقيقةً كأنّه قطع المدينةَ في لحظة،
// **ويقرأ `ApproachTo` أنّه اقترب وابتعد في اللحظة نفسِها.**
//
// **والثاني** أنّ إعادةَ الدفعة لا تُضاعف — **دفعةٌ وصلت وانقطع ردُّها
// يعيدها التطبيقُ لأنّه لا يعلم**، والتكرارُ لا يظهر في شيء: النقاطُ
// صحيحةٌ ومكرّرة، **فيبدو السائقُ أكثفَ حركةً ممّا كان.**
//
// **والثالث** ألّا ترتدّ دفعةٌ قديمةٌ بالموضع الحاليّ إلى الوراء.

func batchFixture(t *testing.T) (*Server, string) {
	t.Helper()
	pool := testdb.Pool(t)
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1}))
	return &Server{pg: pool, logger: quiet}, testdb.NewUser(t, pool, "driver")
}

func TestLocationBatch_KeepsDeviceTime(t *testing.T) {
	srv, uid := batchFixture(t)
	ctx := context.Background()
	base := time.Now().Add(-20 * time.Minute).Truncate(time.Second)

	pts := []trackPoint{
		{Lat: 35.9500, Lng: 39.0100, At: base},
		{Lat: 35.9520, Lng: 39.0130, At: base.Add(5 * time.Minute)},
		{Lat: 35.9550, Lng: 39.0180, At: base.Add(10 * time.Minute)},
	}
	if err := srv.saveTrackBatch(ctx, uid, pts); err != nil {
		t.Fatalf("حفظُ الدفعة: %v", err)
	}

	rows, err := srv.pg.Query(ctx,
		`SELECT recorded_at FROM driver_track WHERE driver_id = $1 ORDER BY recorded_at`, uid)
	if err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	defer rows.Close()
	var got []time.Time
	for rows.Next() {
		var at time.Time
		if rows.Scan(&at) == nil {
			got = append(got, at)
		}
	}
	if len(got) != 3 {
		t.Fatalf("حُفظت %d نقاطٍ من ٣", len(got))
	}
	// **والفروقُ محفوظةٌ كما أرسلها الجهاز** — لا مضغوطةً في ثانية.
	if d := got[2].Sub(got[0]); d < 9*time.Minute || d > 11*time.Minute {
		t.Fatalf("المدّةُ بين أوّل نقطةٍ وآخرِها %v لا ~١٠ دقائق — "+
			"**فتُقرأ الدفعةُ كأنّها لحظةٌ واحدة، ويُخطئ الإسنادُ في الاتّجاه**", d)
	}
}

func TestLocationBatch_ReplayDoesNotDuplicate(t *testing.T) {
	srv, uid := batchFixture(t)
	ctx := context.Background()
	base := time.Now().Add(-10 * time.Minute).Truncate(time.Second)
	pts := []trackPoint{
		{Lat: 35.95, Lng: 39.01, At: base},
		{Lat: 35.96, Lng: 39.02, At: base.Add(time.Minute)},
	}

	for i := range 3 { // **ثلاثُ محاولاتٍ كما يفعل تطبيقٌ لم يصله ردّ**
		if err := srv.saveTrackBatch(ctx, uid, pts); err != nil {
			t.Fatalf("المحاولةُ %d: %v", i+1, err)
		}
	}

	var n int
	if err := srv.pg.QueryRow(ctx,
		`SELECT count(*) FROM driver_track WHERE driver_id = $1`, uid).Scan(&n); err != nil {
		t.Fatalf("عدّ: %v", err)
	}
	if n != 2 {
		t.Fatalf("صار الأثرُ %d نقاطٍ من ٢ — **وإعادةُ الدفعة تُضاعفه بلا أن يظهر**: "+
			"النقاطُ صحيحةٌ ومكرّرة، فيبدو السائقُ أكثفَ حركةً ممّا كان", n)
	}
}

func TestLocationBatch_StaleDoesNotRewindPosition(t *testing.T) {
	srv, uid := batchFixture(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// نبضةٌ حديثةٌ أوّلاً.
	if err := srv.saveTrackBatch(ctx, uid, []trackPoint{
		{Lat: 35.99, Lng: 39.09, At: now.Add(-time.Minute)},
	}); err != nil {
		t.Fatalf("النبضةُ الحديثة: %v", err)
	}
	// ثمّ دفعةٌ قديمةٌ تأخّرت في الطريق.
	if err := srv.saveTrackBatch(ctx, uid, []trackPoint{
		{Lat: 35.90, Lng: 39.00, At: now.Add(-30 * time.Minute)},
	}); err != nil {
		t.Fatalf("الدفعةُ القديمة: %v", err)
	}

	var lat float64
	if err := srv.pg.QueryRow(ctx,
		`SELECT ST_Y(last_location::geometry) FROM users WHERE id = $1`, uid).Scan(&lat); err != nil {
		t.Fatalf("قراءةُ الموضع: %v", err)
	}
	if lat < 35.98 {
		t.Fatalf("الموضعُ ارتدّ إلى %v — **فدفعةٌ تأخّرت في الطريق تعيد السائقَ "+
			"إلى حيث كان قبل نصف ساعة**، ويُسنَد على موضعٍ غادره", lat)
	}
	// **والأثرُ يحفظ الاثنتين** — الارتدادُ ممنوعٌ في الموضع الحاليّ وحدَه،
	// **والتاريخُ يُكتب كما وقع.**
	var n int
	_ = srv.pg.QueryRow(ctx, `SELECT count(*) FROM driver_track WHERE driver_id = $1`, uid).Scan(&n)
	if n != 2 {
		t.Fatalf("الأثرُ %d نقطةً من ٢ — **والنقطةُ القديمةُ تاريخٌ يُكتب لا يُرفَض**", n)
	}
}

// TestLocationBatch_RejectsNonsense **وما لا يُعتدّ به يُرفض قبل القاعدة.**
func TestLocationBatch_RejectsNonsense(t *testing.T) {
	now := time.Now()
	bad := map[string]trackPoint{
		"صفرٌ صفر — جهازٌ لم يجد إشارة، ونقطةٌ في الأطلسيّ تجعل كلَّ مسافةٍ آلافَ الكيلومترات": {At: now},
		"خطُّ عرضٍ خارجَ المدى": {Lat: 91, Lng: 39, At: now},
		"خطُّ طولٍ خارجَ المدى": {Lat: 35, Lng: 181, At: now},
		"بلا زمن": {Lat: 35.9, Lng: 39.0},
		"في الغد — ليست تفاوتَ ساعةِ جهاز": {Lat: 35.9, Lng: 39.0, At: now.Add(24 * time.Hour)},
		"أقدمُ ممّا يُقلَّم أصلاً":         {Lat: 35.9, Lng: 39.0, At: now.Add(-3 * time.Hour)},
	}
	for name, p := range bad {
		if p.valid(now) {
			t.Fatalf("قُبلت نقطةٌ لا يُعتدّ بها: %s", name)
		}
	}
	// **وتفاوتُ ثوانٍ يُقبل** — ساعاتُ الهواتف تسبق وتتأخّر، **ورفضُ نقطةٍ
	// سبقت بثانيتين يفقد أثراً صحيحاً.**
	ok := trackPoint{Lat: 35.9, Lng: 39.0, At: now.Add(30 * time.Second)}
	if !ok.valid(now) {
		t.Fatal("رُفضت نقطةٌ سبقت بنصف دقيقة — **وساعةُ الهاتف تسبق دقائق**")
	}
}
