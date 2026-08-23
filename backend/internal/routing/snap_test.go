package routing

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **حدُّ الالتقاط — TD-SNAP-RADIUS، المرحلة ٨أ**
// ══════════════════════════════════════════════════════════════════════
//
// **العيبُ المقيس** (٢٠٢٦-٠٨-٢١، OSRM v26.8.0 على `syria.osm.pbf`
// تسلسل ٤٦٣٥): إحداثيّةُ باريس تلتقط على طريقٍ سوريٍّ بعد **٣١١١ كم**
// **ويردّ المحرّكُ مساراً بلا خطأ.**

// TestSnapPolicyDefaultsAreCalibrated **الأرقامُ مُعايَرةٌ لا مُختارة.**
//
// **ولو بدّلها أحدٌ يوماً سقط هذا** — فيقرأ لماذا كانت كذلك.
func TestSnapPolicyDefaultsAreCalibrated(t *testing.T) {
	p := DefaultSnapPolicy()

	// **الأصلُ ١٥٠م** — قِيس: حضريّ ٩٨٫٥٪ · ريفيّ ١٠٠٪ عبرَ إزاحاتِ
	// GPS ٠·١٠·٢٥·٥٠·٧٥م على ١٨٩ أصلاً.
	if p.OriginM != 150 {
		t.Fatalf("حدُّ الأصل تبدّل: %.0f (كان ١٥٠ مُعايَراً)", p.OriginM)
	}
	// **والوجهةُ ٢٥٠م** — قِيس على ٩٠٠ متجرٍ حقيقيٍّ من الـPBF:
	// حضريّ ١٠٠٪ · خارجَ المدن ٩٦٫٠٪.
	if p.DestinationM != 250 {
		t.Fatalf("حدُّ الوجهة تبدّل: %.0f (كان ٢٥٠ مُعايَراً)", p.DestinationM)
	}
	// **والوجهةُ أوسعُ عمداً**: رفضُها يعني «لا يمكن التوصيل» لزبونٍ
	// حقيقيّ. **وذلك أسوأُ من مسارٍ يبدأ من زاوية الشارع.**
	if p.DestinationM <= p.OriginM {
		t.Fatalf("الوجهةُ يجب أن تكون أوسعَ من الأصل: %.0f ≤ %.0f",
			p.DestinationM, p.OriginM)
	}
	// **وأقربُ نقطةٍ خاطئةٍ قِيست على ٢٣٠٣٦م** — البادية السوريّة.
	// فالحدّان دونها بعشراتِ الأضعاف.
	const nearestBadM = 23036.0
	if p.DestinationM*10 > nearestBadM {
		t.Fatalf("الهامشُ إلى أقربِ نقطةٍ خاطئةٍ ضاق: %.0f × ١٠ > %.0f",
			p.DestinationM, nearestBadM)
	}
}

// TestSnapRadiusesSent **يُرسَل في الرابط — وإلّا فلا حماية.**
func TestSnapRadiusesSent(t *testing.T) {
	for _, tc := range []struct {
		name string
		snap SnapPolicy
		want string
	}{
		{"الافتراضيّ", DefaultSnapPolicy(), "150;250"},
		{"مُعطًى", SnapPolicy{OriginM: 40, DestinationM: 80}, "40;80"},
		{"بلا حدٍّ للأصل", SnapPolicy{OriginM: 0, DestinationM: 80}, "unlimited;80"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			srv := fakeOSRM(t, func(q url.Values) any {
				got = q.Get("radiuses")
				return okBody()
			})
			defer srv.Close()

			c := NewWithSnap(srv.URL, tc.snap)
			if _, err := c.Route(context.Background(), Point{35.95, 39.01}, Point{35.96, 39.02}); err != nil {
				t.Fatalf("مسار: %v", err)
			}
			if got != tc.want {
				t.Fatalf("radiuses = %q — والمنتظر %q", got, tc.want)
			}
		})
	}
}

// TestSnapRadiusesOnEveryPath **مفردٌ وبدائلُ وإعادةُ حسابٍ سواء.**
//
// **أمرُ المالك (البندان ٩ و١٠)**: «لا يكون single protected
// لكن alternatives unlimited».
//
// **وإعادةُ الحساب تمرّ بالمنفذ نفسِه** — `RouteSet`/`Route` لا ثالثَ
// لهما، فحمايةٌ فيهما حمايةٌ للكلّ.
func TestSnapRadiusesOnEveryPath(t *testing.T) {
	var seen []string
	srv := fakeOSRM(t, func(q url.Values) any {
		seen = append(seen, q.Get("radiuses"))
		return okBody()
	})
	defer srv.Close()

	c := New(srv.URL)
	ctx := context.Background()
	if _, err := c.Route(ctx, Point{35.95, 39.01}, Point{35.96, 39.02}); err != nil {
		t.Fatalf("مفرد: %v", err)
	}
	if _, err := c.RouteSet(ctx, Point{35.95, 39.01}, Point{35.96, 39.02}); err != nil {
		t.Fatalf("بدائل: %v", err)
	}
	if len(seen) < 2 {
		t.Fatalf("طلباتٌ أقلُّ من اثنين: %d", len(seen))
	}
	for i, r := range seen {
		if r != "150;250" {
			t.Fatalf("الطلبُ %d بلا حدٍّ صحيح: %q", i, r)
		}
	}
}

// TestNoSegmentIsTypedAndNotRetried **البند ٧ — ولا توسيعَ عند الفشل.**
//
// **وهذا أخطرُ ما في الدفعة**: من ردّ عليه `NoSegment` ثمّ أعاد
// السؤالَ بلا حدٍّ **أعاد العيبَ الصامتَ بعينه** — مسارٌ واثقٌ من
// إحداثيّةٍ لا تعني شيئاً.
func TestNoSegmentIsTypedAndNotRetried(t *testing.T) {
	var calls int
	var radiuses []string
	srv := fakeOSRM(t, func(q url.Values) any {
		calls++
		radiuses = append(radiuses, q.Get("radiuses"))
		return map[string]any{"code": "NoSegment",
			"message": "Could not find a matching segment for coordinate"}
	})
	defer srv.Close()

	c := New(srv.URL)
	// **باريس** — الحالةُ التي كشفها القياس.
	_, err := c.Route(context.Background(), Point{48.8566, 2.3522}, Point{35.95, 39.01})
	if !errors.Is(err, ErrNoSegment) {
		t.Fatalf("الخطأُ ليس ErrNoSegment: %v", err)
	}
	if calls != 1 {
		t.Fatalf("أُعيد الطلبُ %d مرّة — والمنتظر واحدة", calls)
	}
	for _, r := range radiuses {
		if r == "" || strings.Contains(r, "unlimited") {
			t.Fatalf("أُعيد الطلبُ بلا حدّ: %q — وهذا يُعيد العيبَ الصامت", r)
		}
	}
}

// TestNoRouteIsTyped **طرفان على الشبكة ولا سبيلَ بينهما.**
func TestNoRouteIsTyped(t *testing.T) {
	srv := fakeOSRM(t, func(url.Values) any {
		return map[string]any{"code": "NoRoute"}
	})
	defer srv.Close()
	_, err := New(srv.URL).Route(context.Background(), Point{35.95, 39.01}, Point{35.96, 39.02})
	if !errors.Is(err, ErrNoRoute) {
		t.Fatalf("الخطأُ ليس ErrNoRoute: %v", err)
	}
	// **ويُميَّز عن `NoSegment`** — فعلاجُهما مختلف.
	if errors.Is(err, ErrNoSegment) {
		t.Fatal("خُلط NoRoute بـNoSegment")
	}
}

// TestOverviewRetryKeepsRadius **شبكةُ أمانِ المرحلة ٧ لا تخترق الحدّ.**
//
// **العيبُ المحتمل**: `fetch` تعيد الطلبَ بـ`overview=full` إن جاءت
// الهندسةُ فارغة. **فلو أسقطت الحدَّ في الإعادة** لصار مسارٌ يُرفض
// أوّلاً ثمّ يُقبل ثانياً.
func TestOverviewRetryKeepsRadius(t *testing.T) {
	var seen []string
	var overviews []string
	first := true
	srv := fakeOSRM(t, func(q url.Values) any {
		seen = append(seen, q.Get("radiuses"))
		overviews = append(overviews, q.Get("overview"))
		if first {
			first = false
			// **ردٌّ بلا هندسة** — يُطلق الإعادة.
			return map[string]any{"code": "Ok", "routes": []any{
				map[string]any{"distance": 100.0, "duration": 10.0}}}
		}
		return okBody()
	})
	defer srv.Close()

	if _, err := New(srv.URL).Route(context.Background(),
		Point{35.95, 39.01}, Point{35.96, 39.02}); err != nil {
		t.Fatalf("مسار: %v", err)
	}
	if len(seen) != 2 {
		t.Fatalf("طلبات=%d — والمنتظر اثنان (الأصلُ ثمّ الإعادة)", len(seen))
	}
	if overviews[0] != "false" || overviews[1] != "full" {
		t.Fatalf("الإعادةُ ليست بـoverview=full: %v", overviews)
	}
	if seen[1] != "150;250" {
		t.Fatalf("الإعادةُ أسقطت الحدّ: %q", seen[1])
	}
}

// TestBackendContract **`Client` يحقّق العقدَ — ولا اسمَ محرّكٍ فيه.**
func TestBackendContract(t *testing.T) {
	var b Backend = New("http://x")
	if !b.Enabled() {
		t.Fatal("Enabled")
	}
	if New("").Enabled() {
		t.Fatal("فارغٌ يجب أن يكون معطّلاً")
	}
	// **ولا كلمةَ محرّكٍ في أسماء العقد** — البند ١.
	for _, name := range []string{"Route", "RouteSet", "Enabled"} {
		if strings.Contains(strings.ToLower(name), "osrm") ||
			strings.Contains(strings.ToLower(name), "valhalla") {
			t.Fatalf("اسمُ محرّكٍ في العقد: %s", name)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **خادمٌ مزيَّف — الشبكةُ وحدَها هي المزيَّفة**
// ══════════════════════════════════════════════════════════════════════

// rawQueries **صورةُ السلك كما وصلت** — قبل فكّ الترميز.
var rawQueries []string

func fakeOSRM(t *testing.T, reply func(url.Values) any) *httptest.Server {
	t.Helper()
	rawQueries = nil
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQueries = append(rawQueries, r.URL.RawQuery)
		q := r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(reply(q))
	}))
}

// TestRadiusSeparatorIsEncodedOnTheWire **والفاصلةُ مُرمَّزةٌ في السلك.**
//
// **العيبُ الذي وقعتُ فيه أنا** (٢٠٢٦-٠٨-٢١): فاصلةٌ منقوطةٌ
// خامةٌ في الاستعلام **تجعل `net/url` تُسقط كلّ المعاملات
// صامتةً** — فظننتُ أنّ الشيفرةَ لا ترسل الحدّ أصلاً.
//
// **والمحرّكُ يقبل الاثنتين** — قُستا حيّاً. **لكنّ المُرمَّزةَ
// تمرّ من أيّ محلّلٍ بيننا وبينه.**
func TestRadiusSeparatorIsEncodedOnTheWire(t *testing.T) {
	srv := fakeOSRM(t, func(url.Values) any { return okBody() })
	defer srv.Close()
	if _, err := New(srv.URL).Route(context.Background(),
		Point{35.95, 39.01}, Point{35.96, 39.02}); err != nil {
		t.Fatalf("مسار: %v", err)
	}
	if len(rawQueries) == 0 {
		t.Fatal("لا طلب")
	}
	raw := rawQueries[0]
	if !strings.Contains(raw, "radiuses=150%3B250") {
		t.Fatalf("الفاصلةُ غيرُ مُرمَّزة: %s", raw)
	}
	// **ولا فاصلةَ خامةً في الاستعلام** — وإلّا سقطت المعاملات.
	if strings.Contains(raw, ";") {
		t.Fatalf("فاصلةٌ خامةٌ في الاستعلام: %s", raw)
	}
}

func okBody() map[string]any {
	return map[string]any{
		"code": "Ok",
		"routes": []any{map[string]any{
			"distance": 1907.0, "duration": 148.9,
			"weight": 148.9, "weight_name": "routability",
			"geometry": map[string]any{"coordinates": [][]float64{
				{39.010, 35.950}, {39.015, 35.955}, {39.020, 35.960}}},
			"legs": []any{map[string]any{"steps": []any{
				map[string]any{
					"distance": 1907.0, "duration": 148.9, "name": "طريق",
					"maneuver": map[string]any{"type": "depart",
						"location": []float64{39.010, 35.950}},
					"geometry": map[string]any{"coordinates": [][]float64{
						{39.010, 35.950}, {39.015, 35.955}, {39.020, 35.960}}},
				},
				map[string]any{
					"distance": 0.0, "duration": 0.0, "name": "طريق",
					"maneuver": map[string]any{"type": "arrive",
						"location": []float64{39.020, 35.960}},
					"geometry": map[string]any{"coordinates": [][]float64{
						{39.020, 35.960}}},
				},
			}}},
		}},
	}
}

var _ = strconv.Itoa
