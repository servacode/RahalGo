package server

// **زوّارُ الموقع والتحميلات** — ما يُحرَس:
//
//   - الزيارةُ مرّتين من الشخص نفسِه في اليوم نفسِه = واحدة.
//   - تنزيلٌ مقطّعٌ بطلبَي مدىً = تحميلٌ واحد.
//   - شكلُ ردّ الإحصاء وأرقامُه، والسلسلةُ أيّامٌ متّصلةٌ بأصفارها.
//   - من لا يملك `platform.overview` يُردّ ٤٠٣.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

func hitCount(t *testing.T, s *Server, kind, visitor string) int {
	t.Helper()
	var n int
	if err := s.pg.QueryRow(context.Background(),
		`SELECT count(*) FROM site_hits WHERE kind = $1 AND visitor = $2`, kind, visitor).Scan(&n); err != nil {
		t.Fatalf("تعذّر العدّ: %v", err)
	}
	return n
}

func uniqueUA(t *testing.T) string {
	return fmt.Sprintf("site-stats-test/%s/%d", t.Name(), time.Now().UnixNano())
}

func TestPublicVisitCountsOncePerDay(t *testing.T) {
	s := overviewServer(t)
	ua := uniqueUA(t)
	var visitor string
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/public/visit", nil)
		req.RemoteAddr = "203.0.113.7:5555"
		req.Header.Set("User-Agent", ua)
		w := httptest.NewRecorder()
		s.handlePublicVisit(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("ردّ %d لا ٢٠٤", w.Code)
		}
		visitor = visitorOf(req)
	}
	if len(visitor) != 32 {
		t.Fatalf("البصمةُ %d خانة لا ٣٢", len(visitor))
	}
	if n := hitCount(t, s, hitVisit, visitor); n != 1 {
		t.Fatalf("زيارتان من الشخص نفسِه = %d صفّ، والمطلوب ١", n)
	}
	t.Cleanup(func() {
		_, _ = s.pg.Exec(context.Background(), `DELETE FROM site_hits WHERE visitor = $1`, visitor)
	})
}

func TestDownloadRangeRequestsCountOnce(t *testing.T) {
	f := newAppFixture(t)
	if code := f.upload("rahalgo-1.0.apk", apk(4096)).Code; code != http.StatusCreated {
		t.Fatalf("رُفض الرفعُ برمز %d", code)
	}
	ua := uniqueUA(t)
	var visitor string
	for _, rng := range []string{"bytes=0-999", "bytes=1000-"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/public/app/driver", nil)
		req.RemoteAddr = "198.51.100.9:4444"
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Range", rng)
		rc := chi.NewRouteContext()
		rc.URLParams.Add("key", "driver")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
		w := httptest.NewRecorder()
		f.srv.handleDownloadAppByKey(w, req)
		if w.Code != http.StatusPartialContent {
			t.Fatalf("طلبُ المدى %q ردّ %d", rng, w.Code)
		}
		visitor = visitorOf(req)
	}
	t.Cleanup(func() {
		_, _ = f.srv.pg.Exec(context.Background(), `DELETE FROM site_hits WHERE visitor = $1`, visitor)
	})
	// **والتسجيلُ في الخلفيّة** — يُنتظر قليلاً.
	deadline := time.Now().Add(5 * time.Second)
	n := 0
	for time.Now().Before(deadline) {
		if n = hitCount(t, f.srv, "download:driver", visitor); n >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)
	if n = hitCount(t, f.srv, "download:driver", visitor); n != 1 {
		t.Fatalf("طلبا مدىً لتنزيلٍ واحد = %d صفّ، والمطلوب ١", n)
	}
}

func TestSiteStatsShapeAndAuthz(t *testing.T) {
	s := overviewServer(t)
	ctx := context.Background()
	now := time.Now()
	today := siteDay(now)
	tag := fmt.Sprintf("sst%d", now.UnixNano())

	read := func() siteStats {
		t.Helper()
		w := callAs(s.handleSiteStats, http.MethodGet, "/admin/site-stats?days=7", "", "",
			[]string{string(authz.PlatformOverview)})
		var out siteStats
		if err := json.Unmarshal(dataOf(t, w), &out); err != nil {
			t.Fatalf("ردٌّ لا يُقرأ: %v", err)
		}
		return out
	}
	before := read()

	// زائران اليوم، وأحدُهما زار أمس أيضاً، وتحميلان للزبون وواحدٌ للمندوب.
	yesterday := now.AddDate(0, 0, -1)
	hits := []struct {
		kind, who string
		at        time.Time
	}{
		{hitVisit, tag + "a", now}, {hitVisit, tag + "a", now}, {hitVisit, tag + "b", now},
		{hitVisit, tag + "a", yesterday},
		{"download:customer", tag + "a", now}, {"download:customer", tag + "b", now},
		{"download:rep", tag + "a", now},
	}
	for _, h := range hits {
		if err := s.recordHit(ctx, h.kind, h.who, h.at); err != nil {
			t.Fatalf("تعذّر التسجيل: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = s.pg.Exec(ctx, `DELETE FROM site_hits WHERE visitor LIKE $1`, tag+"%")
	})

	after := read()
	if d := after.Today.Visits - before.Today.Visits; d != 2 {
		t.Fatalf("زوّارُ اليوم زادوا %d والمطلوب ٢", d)
	}
	if d := after.Totals.VisitsUniqueEver - before.Totals.VisitsUniqueEver; d != 2 {
		t.Fatalf("كلُّ الزوّار زادوا %d والمطلوب ٢", d)
	}
	if d := after.Totals.VisitsDaysSum - before.Totals.VisitsDaysSum; d != 3 {
		t.Fatalf("مجموعُ الأيّام زاد %d والمطلوب ٣", d)
	}
	if d := after.Today.Downloads.Customer - before.Today.Downloads.Customer; d != 2 {
		t.Fatalf("تحميلاتُ الزبون اليوم زادت %d والمطلوب ٢", d)
	}
	if d := after.Totals.Downloads.Rep - before.Totals.Downloads.Rep; d != 1 {
		t.Fatalf("تحميلاتُ المندوب زادت %d والمطلوب ١", d)
	}
	if len(after.Series) != 7 {
		t.Fatalf("السلسلةُ %d يوماً والمطلوب ٧", len(after.Series))
	}
	last := after.Series[len(after.Series)-1]
	if last.Day != today {
		t.Fatalf("آخرُ يومٍ %q واليوم %q", last.Day, today)
	}
	if last.Visits != after.Today.Visits {
		t.Fatalf("زوّارُ اليوم في السلسلة %d وفي البطاقة %d", last.Visits, after.Today.Visits)
	}
	if last.NewAccounts != after.Today.NewAccounts {
		t.Fatalf("حساباتُ اليوم في السلسلة %d وفي البطاقة %d", last.NewAccounts, after.Today.NewAccounts)
	}
	for i := 1; i < len(after.Series); i++ {
		a, _ := time.Parse("2006-01-02", after.Series[i-1].Day)
		b, _ := time.Parse("2006-01-02", after.Series[i].Day)
		if b.Sub(a) != 24*time.Hour {
			t.Fatalf("السلسلةُ غيرُ متّصلة: %s ثمّ %s", after.Series[i-1].Day, after.Series[i].Day)
		}
	}

	// **وحسابٌ جديدٌ اليوم يُعدّ** — وحسابُ النظام لا.
	var uid string
	phone := fmt.Sprintf("+9639%08d", now.UnixNano()%100000000)
	if err := s.pg.QueryRow(ctx, `INSERT INTO users (phone, full_name, status) VALUES ($1, 'site stats', 'active') RETURNING id`,
		phone).Scan(&uid); err != nil {
		t.Fatalf("تعذّر إنشاءُ الحساب: %v", err)
	}
	t.Cleanup(func() { _, _ = s.pg.Exec(ctx, `DELETE FROM users WHERE id = $1`, uid) })
	again := read()
	if d := again.Today.NewAccounts - after.Today.NewAccounts; d != 1 {
		t.Fatalf("حساباتُ اليوم زادت %d والمطلوب ١", d)
	}

	// **ومن لا يملك القدرةَ يُردّ.**
	w := callAs(s.handleSiteStats, http.MethodGet, "/admin/site-stats", "", "",
		[]string{string(authz.AnalyticsRead)})
	if w.Code != http.StatusForbidden {
		t.Fatalf("بلا `platform.overview` ردّ %d لا ٤٠٣", w.Code)
	}
}
