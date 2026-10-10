package server

// **حرّاسُ راصد المنصّة** (مراقبةُ المنصّة — الدفعةُ الأولى ٢٠٢٦-١٠-٠٩):
// منعُ التكرار وعودةُ السلامة، وموعدُ التقرير الصباحيّ بساعةٍ تُمرَّر،
// ووسيطُ عدّ ٥xx، والكتابةُ في الجدول، ونبضةٌ كاملةٌ بمُرسِلٍ يلتقط الرسائل.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/platform"
)

// captureSender **يلتقط ما كان سيُرسَل على الواتساب.**
type captureSender struct {
	mu   sync.Mutex
	msgs []string
	fail bool
}

func (c *captureSender) SendToSelf(_ context.Context, text string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return errors.New("not ready")
	}
	c.msgs = append(c.msgs, text)
	return nil
}

func (c *captureSender) take() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	m := c.msgs
	c.msgs = nil
	return m
}

// fakeBot **بوتٌ يُقرَّر أمتّصلٌ هو.**
type fakeBot struct{ ready bool }

func (f *fakeBot) SendText(context.Context, string, string) error { return nil }
func (f *fakeBot) Ready() bool                                    { return f.ready }

func TestAlertBook_DedupRepeatAndResolve(t *testing.T) {
	var b alertBook
	var sent []string
	ok := true
	send := func(s string) bool {
		if ok {
			sent = append(sent, s)
		}
		return ok
	}
	bad := []monCheck{{Key: "db", Known: true, Bad: true, Text: "واقفة", OKText: "رجعت"}}
	good := []monCheck{{Key: "db", Known: true, Bad: false, Text: "واقفة", OKText: "رجعت"}}
	unknown := []monCheck{{Key: "db", Known: false}}
	t0 := time.Date(2026, 10, 9, 1, 0, 0, 0, time.UTC)

	b.step(good, t0, send)
	if len(sent) != 0 {
		t.Fatalf("سليمٌ من البداية لا يُرسل شيئاً — %v", sent)
	}
	b.step(bad, t0, send)
	b.step(bad, t0.Add(time.Minute), send)
	b.step(bad, t0.Add(59*time.Minute), send)
	if len(sent) != 1 || sent[0] != "واقفة" {
		t.Fatalf("التنبيهُ مرّةً واحدةً خلال الساعة — %v", sent)
	}
	// **وما لم يُقَس لا يُقال عنه «رجع طبيعي».**
	b.step(unknown, t0.Add(61*time.Minute), send)
	if len(sent) != 1 {
		t.Fatalf("فحصٌ لم يُقَس أرسل شيئاً — %v", sent)
	}
	b.step(bad, t0.Add(61*time.Minute), send)
	if len(sent) != 2 || !strings.Contains(sent[1], "لسّا") {
		t.Fatalf("بعد ساعةٍ يُذكَّر مرّة — %v", sent)
	}
	b.step(good, t0.Add(62*time.Minute), send)
	if len(sent) != 3 || sent[2] != "رجعت" {
		t.Fatalf("عودةُ السلامة تُقال — %v", sent)
	}
	b.step(good, t0.Add(63*time.Minute), send)
	if len(sent) != 3 {
		t.Fatalf("«رجع طبيعي» مرّةً لا أكثر — %v", sent)
	}
	// **وإن عادت المشكلةُ أُنذر من جديد فوراً.**
	b.step(bad, t0.Add(64*time.Minute), send)
	if len(sent) != 4 || sent[3] != "واقفة" {
		t.Fatalf("مشكلةٌ عادت لا تُنذَر — %v", sent)
	}
	// **وما لم يصل يُعاد في النبضة التالية** لا يُعلَّم مُرسَلاً.
	var b2 alertBook
	ok = false
	b2.step(bad, t0, send)
	ok = true
	b2.step(bad, t0.Add(time.Minute), send)
	if len(sent) != 5 || sent[4] != "واقفة" {
		t.Fatalf("تنبيهٌ فشل إرسالُه لم يُعَد — %v", sent)
	}
}

func TestMorningDue_DamascusHourOncePerDay(t *testing.T) {
	loc := platform.Location()
	at := func(h, m int) time.Time { return time.Date(2026, 10, 9, h, m, 0, 0, loc) }
	if _, due := morningDue(at(8, 59), 9, ""); due {
		t.Fatal("قبل الساعة لا يُرسَل")
	}
	day, due := morningDue(at(9, 0), 9, "")
	if !due || day != "2026-10-09" {
		t.Fatalf("في الساعة يُرسَل — %v %q", due, day)
	}
	if _, due := morningDue(at(9, 30), 9, day); due {
		t.Fatal("أُرسل مرّتين في اليوم نفسِه")
	}
	if _, due := morningDue(at(10, 0), 9, ""); due {
		t.Fatal("بعد الساعة لا يُرسَل — خادمٌ أُعيد ظهراً لا يقول «صباح الخير»")
	}
	// **وتوقيتُ دمشق لا غرينتش**: السادسةُ بغرينتش هي التاسعةُ بدمشق.
	if _, due := morningDue(time.Date(2026, 10, 9, 6, 5, 0, 0, time.UTC), 9, ""); !due {
		t.Fatal("الساعةُ تُقرأ بتوقيت دمشق")
	}
	next, due := morningDue(at(9, 0).AddDate(0, 0, 1), 9, day)
	if !due || next != "2026-10-10" {
		t.Fatal("اليومُ التالي يُرسَل من جديد")
	}
}

func TestCountErrors_CountsOnly5xxByRoutePattern(t *testing.T) {
	s := &Server{}
	r := chi.NewRouter()
	r.Use(s.countErrors)
	r.Get("/orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		if chi.URLParam(r, "id") == "bad" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/busy", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	r.Get("/boom", func(http.ResponseWriter, *http.Request) { panic("x") })
	call := func(p string) {
		defer func() { _ = recover() }()
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", p, nil))
	}
	call("/orders/ok")
	call("/orders/bad")
	call("/orders/bad")
	call("/busy")
	call("/missing") // ٤٠٤ لا يُعدّ
	if n := s.errs.countSince(time.Now().Add(-time.Minute)); n != 3 {
		t.Fatalf("عُدّ %d خطأً — والمنتظَرُ ٣ (اثنان ٥٠٠ وواحد ٥٠٣)", n)
	}
	p := s.errs.take()
	got := map[string]int64{}
	for k, a := range p {
		got[k.Route+"|"+http.StatusText(k.Code)] = a.Count
		if strings.Contains(k.Route, "bad") {
			t.Fatalf("الرابطُ الخامُّ حُفظ بدل النمط — %q", k.Route)
		}
	}
	if got["/orders/{id}|Internal Server Error"] != 2 || got["/busy|Service Unavailable"] != 1 {
		t.Fatalf("التجميعُ بالنمط والرمز خطأ — %v", got)
	}
}

func TestErrorTracker_FlushThenSummary(t *testing.T) {
	srv := overviewServer(t)
	ctx := context.Background()
	route := "/qa/monitoring/" + time.Now().Format("150405.000000")
	t.Cleanup(func() {
		_, _ = srv.pg.Exec(context.Background(), `DELETE FROM server_errors WHERE route = $1`, route)
	})
	now := time.Now()
	for range 4 {
		srv.errs.record(route, 500, now)
	}
	if err := srv.flushErrors(ctx); err != nil {
		t.Fatalf("الكتابة: %v", err)
	}
	srv.errs.record(route, 500, now)
	sum, err := srv.errorsSince(ctx, now.Add(-time.Hour), now)
	if err != nil {
		t.Fatalf("القراءة: %v", err)
	}
	var found *errorGroup
	for i := range sum.Groups {
		if sum.Groups[i].Route == route {
			found = &sum.Groups[i]
		}
	}
	if found == nil || found.Count != 5 || found.Code != 500 {
		t.Fatalf("المجمَّعُ لم يجمع ما في الجدول وما في الذاكرة — %+v", found)
	}
}

func TestMonitorTick_BotAlertOnceThenResolved(t *testing.T) {
	srv := overviewServer(t)
	out := &captureSender{}
	bot := &fakeBot{ready: false}
	srv.SetAlertSender(out)
	srv.SetMerchantNotifier(bot)
	ctx := context.Background()
	// **في الثالثة فجراً بدمشق** — فلا يتدخّل التقريرُ الصباحيّ.
	t0 := time.Date(2026, 10, 9, 3, 0, 0, 0, platform.Location())
	botMsgs := func() []string {
		var r []string
		for _, m := range out.take() {
			if strings.Contains(m, "بوت الواتساب") {
				r = append(r, m)
			}
		}
		return r
	}

	srv.monitorTick(ctx, t0)
	if m := botMsgs(); len(m) != 0 {
		t.Fatalf("انفصالٌ أقلُّ من خمس دقائق لا يُنذَر — %v", m)
	}
	srv.monitorTick(ctx, t0.Add(6*time.Minute))
	m := botMsgs()
	if len(m) != 1 || !strings.Contains(m[0], "مفصول") {
		t.Fatalf("بعد خمس دقائق يُنذَر مرّة — %v", m)
	}
	t.Logf("التنبيه:\n%s", m[0])
	srv.monitorTick(ctx, t0.Add(7*time.Minute))
	if m := botMsgs(); len(m) != 0 {
		t.Fatalf("التنبيهُ تكرّر في الساعة نفسِها — %v", m)
	}
	bot.ready = true
	srv.monitorTick(ctx, t0.Add(8*time.Minute))
	m = botMsgs()
	if len(m) != 1 || !strings.Contains(m[0], "رجع طبيعي") {
		t.Fatalf("عودةُ البوت لم تُقَل — %v", m)
	}

	// **والإعدادُ يُطفئه**: لا تنبيهَ والراصدُ مُطفأ.
	if err := srv.settings.Set(ctx, "monitoring.enabled", false, nil); err != nil {
		t.Fatalf("الإعداد: %v", err)
	}
	t.Cleanup(func() { _ = srv.settings.Set(context.Background(), "monitoring.enabled", true, nil) })
	bot.ready = false
	srv.monitorTick(ctx, t0.Add(9*time.Minute))
	srv.monitorTick(ctx, t0.Add(20*time.Minute))
	if m := out.take(); len(m) != 0 {
		t.Fatalf("الراصدُ مُطفأٌ وأرسل — %v", m)
	}
}

func TestMonitorTick_MorningReportOnce(t *testing.T) {
	srv := overviewServer(t)
	out := &captureSender{}
	srv.SetAlertSender(out)
	ctx := context.Background()
	at := time.Date(2026, 10, 9, 9, 0, 0, 0, platform.Location())
	morning := func() []string {
		var r []string
		for _, m := range out.take() {
			if strings.Contains(m, "صباح الخير") {
				r = append(r, m)
			}
		}
		return r
	}
	srv.monitorTick(ctx, at)
	m := morning()
	if len(m) != 1 {
		t.Fatalf("التقريرُ الصباحيّ لم يُرسَل مرّة — %d", len(m))
	}
	t.Logf("التقرير:\n%s", m[0])
	for _, want := range []string{"مبارح", "أخطاء الخادم", "وضع رحّال غو"} {
		if !strings.Contains(m[0], want) {
			t.Errorf("التقريرُ بلا «%s»:\n%s", want, m[0])
		}
	}
	srv.monitorTick(ctx, at.Add(10*time.Minute))
	if m := morning(); len(m) != 0 {
		t.Fatalf("التقريرُ تكرّر في اليوم نفسِه — %d", len(m))
	}
}

// TestCountErrors_DeliberateRefusalNotCounted — **الرفضُ المقصود ليس عطلاً**
// (تنبيهٌ كاذبٌ ٢٠٢٦-١٠-١٠): إغلاقُ الاستقبال ولا سائقَ في الدوام ٥٠٣ لا
// يُعدّان، **و`service_busy` يبقى يُعدّ.**
func TestCountErrors_DeliberateRefusalNotCounted(t *testing.T) {
	s := &Server{}
	r := chi.NewRouter()
	r.Use(s.countErrors)
	r.Get("/quote", func(w http.ResponseWriter, _ *http.Request) {
		httpx.Error(w, ErrPlatformClosedNow)
	})
	r.Get("/nodrivers", func(w http.ResponseWriter, _ *http.Request) {
		httpx.Error(w, orders.ErrNoDriversOnShift)
	})
	r.Get("/busy", func(w http.ResponseWriter, _ *http.Request) {
		httpx.Error(w, errServiceBusy)
	})
	for _, p := range []string{"/quote", "/quote", "/nodrivers", "/busy"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest("GET", p, nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s ردَّ %d", p, rec.Code)
		}
	}
	if n := s.errs.countSince(time.Now().Add(-time.Minute)); n != 1 {
		t.Fatalf("عُدّ %d — والمنتظَرُ ١ (`service_busy` وحدَه)", n)
	}
}
