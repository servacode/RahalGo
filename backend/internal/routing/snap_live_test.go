package routing

import (
	"context"
	"errors"
	"os"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **الحمايةُ على محرّكٍ حيّ — المرحلة ٨أ، البند ١٢**
// ══════════════════════════════════════════════════════════════════════
//
// **والخادمُ المزيّفُ يُثبت أنّنا نرسل الحدّ.** وهذا يُثبت أنّ
// **المحرّكَ يحترمه** — وهما شيئان.
//
// يُشغَّل بـ`OSRM_URL`، ويُتخطّى بلاها.

// TestLiveFarCoordinatesRejected **الحالاتُ التي كشفها قياسُ المرحلة ٨.**
//
// **قبل هذه الدفعة**: باريس تلتقط على طريقٍ سوريٍّ بعد **٣١١١ كم**
// **ويردّ المحرّكُ مساراً كاملاً بلا خطأ** — والقاهرة ٥٢٣ كم،
// وبغداد ٣٣٢ كم، وأنقرة ٤٧٣ كم، والبحرُ المتوسّط ١٧٠ كم.
func TestLiveFarCoordinatesRejected(t *testing.T) {
	base := os.Getenv("OSRM_URL")
	if base == "" {
		t.Skip("لا محرّك")
	}
	raqqa := Point{35.9500, 39.0100}
	far := []struct {
		name string
		p    Point
		// snappedKm **ما قِيس قبل الحماية.**
		snappedKm int
	}{
		{"باريس", Point{48.8566, 2.3522}, 3111},
		{"القاهرة", Point{30.0444, 31.2357}, 523},
		{"أنقرة", Point{39.9334, 32.8597}, 473},
		{"بغداد", Point{33.3152, 44.3661}, 332},
		{"البحرُ المتوسّط", Point{35.0000, 34.0000}, 170},
		{"الباديةُ السوريّة", Point{33.5000, 39.5000}, 23},
	}

	// **بلا حدٍّ: يمرّ كلُّها** — وهذا هو العيب.
	open := NewWithSnap(base, SnapPolicy{})
	guarded := New(base)

	for _, tc := range far {
		t.Run(tc.name, func(t *testing.T) {
			// ── ١ · العيبُ يُرى أوّلاً ─────────────────────────────
			if r, err := open.Route(context.Background(), tc.p, raqqa); err == nil && r != nil {
				t.Logf("بلا حدّ: مسارٌ %.0f كم من إحداثيّةٍ تبعد ~%d كم — **العيب**",
					r.DistanceM/1000, tc.snappedKm)
			} else {
				t.Logf("بلا حدّ: %v", err)
			}

			// ── ٢ · ثمّ يُغلق ──────────────────────────────────────
			_, err := guarded.Route(context.Background(), tc.p, raqqa)
			if !errors.Is(err, ErrNoSegment) {
				t.Fatalf("لم تُرفض: %v", err)
			}

			// ── ٣ · ووجهةً أيضاً — البند ٨ ─────────────────────────
			_, err = guarded.Route(context.Background(), raqqa, tc.p)
			if !errors.Is(err, ErrNoSegment) {
				t.Fatalf("لم تُرفض وجهةً: %v", err)
			}
		})
	}
}

// TestLiveValidPointsStillRoute **ولا يُرفض ما يجب أن يمرّ.**
//
// **وهذا نصفُ المسألة الآخر**: حدٌّ يرفض باريسَ ويرفض معها متجراً في
// الرقّة **لا يصلح.**
func TestLiveValidPointsStillRoute(t *testing.T) {
	base := os.Getenv("OSRM_URL")
	if base == "" {
		t.Skip("لا محرّك")
	}
	c := New(base)
	cases := []struct {
		name     string
		from, to Point
	}{
		// **مواضعُ حضريّةٌ حقيقيّة** — من رفيدة المرحلة ٨.
		{"الرقّة حضريّ", Point{35.9500, 39.0100}, Point{35.9600, 39.0200}},
		{"دمشق حضريّ", Point{33.5138, 36.2765}, Point{33.5220, 36.2900}},
		{"حلب حضريّ", Point{36.2021, 37.1343}, Point{36.2100, 37.1500}},
		// **وريفيّةٌ وبين المدن** — حيث الشبكةُ أرقّ.
		{"الرقّة→الطبقة", Point{35.9500, 39.0100}, Point{35.8400, 38.5450}},
		{"الرقّة→دمشق", Point{35.9500, 39.0100}, Point{33.5138, 36.2765}},
		{"ديرالزور→دمشق", Point{35.3350, 40.1400}, Point{33.5138, 36.2765}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := c.Route(context.Background(), tc.from, tc.to)
			if err != nil {
				t.Fatalf("رُفض مسارٌ صحيح: %v", err)
			}
			if r.DistanceM <= 0 || len(r.Geometry) < 2 {
				t.Fatalf("مسارٌ فاسد: %.0fم · %d نقطة", r.DistanceM, len(r.Geometry))
			}
		})
	}
}

// TestLiveGpsOffsetStillRoutes **إزاحاتُ GPS ١٠·٢٥·٥٠·٧٥م — البند ١٢.**
//
// **موضعُ السائق ليس دقيقاً أبداً.** فلو رفض الحدُّ إزاحةً واقعيّةً
// **لانقطعت الملاحةُ في يد سائقٍ يقود.**
//
// **والإزاحةُ ثابتةٌ لا عشوائيّة** — فمن أعاد التشغيل رأى ما رأيت.
func TestLiveGpsOffsetStillRoutes(t *testing.T) {
	base := os.Getenv("OSRM_URL")
	if base == "" {
		t.Skip("لا محرّك")
	}
	c := New(base)
	dst := Point{35.9600, 39.0250}
	origins := []Point{
		{35.9500, 39.0100}, // الرقّة
		{33.5138, 36.2765}, // دمشق
		{36.2021, 37.1343}, // حلب
	}
	// **درجةُ عرضٍ ≈ ١١١٣٢٠م** — فالإزاحةُ تُحسب منها.
	const degPerM = 1.0 / 111320.0

	for _, o := range origins {
		for _, off := range []float64{10, 25, 50, 75} {
			// **شمالاً ثمّ شرقاً** — اتّجاهان لا واحد.
			for _, d := range []Point{
				{o.Lat + off*degPerM, o.Lng},
				{o.Lat, o.Lng + off*degPerM*1.22},
			} {
				to := dst
				if o.Lat > 35.0 && o.Lng < 38.0 {
					to = Point{o.Lat + 0.02, o.Lng + 0.02}
				}
				_, err := c.Route(context.Background(), d, to)
				if errors.Is(err, ErrNoSegment) {
					t.Errorf("إزاحةُ %.0fم رُفضت عند %.4f,%.4f", off, o.Lat, o.Lng)
				}
			}
		}
	}
}

// TestLiveRadiusIsHonoured **الحدُّ يعمل فعلاً — لا يُرسَل ويُهمَل.**
//
// **يُقاس بحدٍّ ضيّقٍ متعمَّد**: نقطةٌ تبعد عن أقرب طريقٍ ٤٥٫٥م
// (قِيست) — **تمرّ بحدّ ١٥٠ وتُرفض بحدّ ٢٥.**
func TestLiveRadiusIsHonoured(t *testing.T) {
	base := os.Getenv("OSRM_URL")
	if base == "" {
		t.Skip("لا محرّك")
	}
	p := Point{35.9500, 39.0100}
	to := Point{35.9600, 39.0200}

	if _, err := New(base).Route(context.Background(), p, to); err != nil {
		t.Fatalf("الحدُّ الافتراضيُّ رفض نقطةً على ٤٥٫٥م: %v", err)
	}
	tight := NewWithSnap(base, SnapPolicy{OriginM: 25, DestinationM: 250})
	if _, err := tight.Route(context.Background(), p, to); !errors.Is(err, ErrNoSegment) {
		t.Fatalf("حدُّ ٢٥م لم يرفض نقطةً على ٤٥٫٥م — فالحدُّ يُهمَل: %v", err)
	}
}
