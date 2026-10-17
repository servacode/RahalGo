package server

// **حرّاسُ عين الأمان** (مراقبةُ المنصّة — الدفعةُ الثانية ٢٠٢٦-١٠-١٧):
// العدُّ ونافذةُ القفزة وحدُّها، و«المختلفُ وحدَه» في رفض التخويل، ومنعُ تكرار
// التنبيه الفوريّ، ونصوصٌ بلا رقمٍ كامل، وعدٌّ يوميٌّ يبقى بعد إعادة التشغيل.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/platform"
)

func TestSecurityTracker_SpikeWindowAndThreshold(t *testing.T) {
	s := &Server{}
	now := time.Now()
	// **ومحاولاتٌ قديمةٌ خارجَ النافذة لا تُعدّ في القفزة.**
	for range 30 {
		s.sec.record(secLoginFailed, now.Add(-11*time.Minute))
	}
	for range 19 {
		s.sec.record(secLoginFailed, now)
	}
	var b alertBook
	var sent []string
	send := func(m string) bool { sent = append(sent, m); return true }
	b.step(s.securityChecks(context.Background(), now), now, send)
	if len(sent) != 0 {
		t.Fatalf("١٩ محاولةً تحت حدّ ٢٠ أنذرت — %v", sent)
	}
	s.sec.record(secLoginFailed, now)
	b.step(s.securityChecks(context.Background(), now), now, send)
	if len(sent) != 1 || !strings.HasPrefix(sent[0], "🚨 محاولات دخول غلط كتير: ٢٠ بعشر دقايق") {
		t.Fatalf("٢٠ محاولةً لم تُنذر بالنصّ المنتظر — %v", sent)
	}
	b.step(s.securityChecks(context.Background(), now.Add(time.Minute)), now.Add(time.Minute), send)
	if len(sent) != 1 {
		t.Fatalf("القفزةُ تكرّرت في الدقيقة التالية — %v", sent)
	}
	// **وبعد عشر دقائقَ تخرج من النافذة ← «رجع طبيعي».**
	later := now.Add(spikeWindow + time.Minute)
	b.step(s.securityChecks(context.Background(), later), later, send)
	if len(sent) != 2 || !strings.Contains(sent[1], "رجع طبيعي") {
		t.Fatalf("عودةُ السلامة لم تُقل — %v", sent)
	}
	// **والعدُّ اليوميُّ يحوي الكلّ** — القديمَ والجديد.
	if got := s.sec.take()[secDayKey{Day: now.In(platform.Location()).Format("2006-01-02"), Kind: secLoginFailed}]; got < 20 {
		t.Fatalf("العدُّ اليوميّ = %d", got)
	}
}

func TestSecurityTracker_ForbiddenSpikeAt30(t *testing.T) {
	s := &Server{}
	now := time.Now()
	for i := range 29 {
		s.sec.recordDistinct(secForbidden, uint64(i+1), now)
	}
	// **والبصمةُ نفسُها لا تُعدّ مرّتين.**
	for range 50 {
		s.sec.recordDistinct(secForbidden, 1, now)
	}
	var b alertBook
	var sent []string
	send := func(m string) bool { sent = append(sent, m); return true }
	b.step(s.securityChecks(context.Background(), now), now, send)
	if len(sent) != 0 {
		t.Fatalf("٢٩ مختلفةً تحت حدّ ٣٠ أنذرت — %v", sent)
	}
	s.sec.recordDistinct(secForbidden, 999, now)
	b.step(s.securityChecks(context.Background(), now), now, send)
	if len(sent) != 1 || !strings.HasPrefix(sent[0], "🚨 محاولات فتح بيانات الغير كتير: ٣٠ بعشر دقايق") {
		t.Fatalf("٣٠ مختلفةً لم تُنذر بالنصّ المنتظر — %v", sent)
	}
}

// TestCountErrors_AuthzRefusalsCountedDistinct **٤٠٣ تخويلٍ و٤٠٤ معرّفٍ بتوكن يُعدّان**،
// وقرارُ العمل والمجهولُ وبلا توكنٍ لا، **والتكرارُ على الرابط نفسِه مرّة.**
func TestCountErrors_AuthzRefusalsCountedDistinct(t *testing.T) {
	s := &Server{}
	r := chi.NewRouter()
	r.Use(s.countErrors)
	r.Get("/orders/{id}", func(w http.ResponseWriter, _ *http.Request) { httpx.Error(w, httpx.ErrNotFound) })
	r.Get("/admin/x", func(w http.ResponseWriter, _ *http.Request) { httpx.Error(w, errForbidden) })
	r.Get("/driver/{id}", func(w http.ResponseWriter, _ *http.Request) { httpx.Error(w, errNotYourOrder) })
	r.Get("/store", func(w http.ResponseWriter, _ *http.Request) { httpx.Error(w, errStoreSuspended) })
	r.Get("/list", func(w http.ResponseWriter, _ *http.Request) { httpx.Error(w, httpx.ErrNotFound) })
	call := func(p, tok string) {
		req := httptest.NewRequest("GET", p, nil)
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		r.ServeHTTP(httptest.NewRecorder(), req)
	}
	call("/orders/a", "tok1") // ✓
	call("/orders/b", "tok1") // ✓
	call("/orders/b", "tok1") // تكرار
	call("/orders/b", "tok1") // تكرار
	call("/orders/b", "tok2") // ✓ حسابٌ آخر
	call("/orders/c", "")     // بلا توكن
	call("/admin/x", "tok1")  // ✓
	call("/driver/z", "tok1") // ✓
	call("/store", "tok1")    // قرارُ عمل
	call("/list", "tok1")     // ٤٠٤ بلا معرّف
	call("/missing", "tok1")  // لا مسار
	if n := s.sec.countSince(secForbidden, time.Now().Add(-time.Minute)); n != 5 {
		t.Fatalf("عُدّ %d — والمنتظَرُ ٥", n)
	}
	if n := s.errs.countSince(time.Now().Add(-time.Minute)); n != 0 {
		t.Fatalf("رفضُ التخويل دخل عدّادَ أخطاء الخادم — %d", n)
	}
}

func TestSecurityAlerts_OncePerSubjectPerHour(t *testing.T) {
	var tr securityTracker
	t0 := time.Now()
	var sent []string
	ok := false
	send := func(m string) bool {
		if ok {
			sent = append(sent, m)
		}
		return ok
	}
	tr.enqueue("code:o1", "A", t0)
	tr.enqueue("code:o1", "A", t0) // في الطابور
	tr.drain(t0, send)             // فشل ← يبقى
	ok = true
	tr.drain(t0.Add(time.Minute), send)
	if len(sent) != 1 || sent[0] != "A" {
		t.Fatalf("ما فشل لم يُعَد أو تكرّر — %v", sent)
	}
	tr.enqueue("code:o1", "A", t0.Add(30*time.Minute))
	tr.enqueue("code:o2", "B", t0.Add(30*time.Minute))
	tr.drain(t0.Add(30*time.Minute), send)
	if len(sent) != 2 || sent[1] != "B" {
		t.Fatalf("الصاحبُ نفسُه في الساعة أُعيد، أو الآخرُ لم يُرسَل — %v", sent)
	}
	tr.enqueue("code:o1", "A", t0.Add(62*time.Minute))
	tr.drain(t0.Add(62*time.Minute), send)
	if len(sent) != 3 || sent[2] != "A" {
		t.Fatalf("بعد ساعةٍ يُنبَّه من جديد — %v", sent)
	}
}

// TestSecurityAlertText_NoFullPhone **آخرُ ثلاثة أرقامٍ وحدَها.**
func TestSecurityAlertText_NoFullPhone(t *testing.T) {
	phone := "+963912345456"
	tail := identity.PhoneTail(phone)
	if tail != "456" {
		t.Fatalf("الذيل = %q", tail)
	}
	m := lockAlertText(tail)
	if !strings.HasPrefix(m, "🔐 حساب انقفل من كتر محاولات كلمة السر: رقم ينتهي بـ …456") {
		t.Fatalf("النصّ:\n%s", m)
	}
	for _, leak := range []string{phone, "963912345456", "912345456", "2345456", "5456"} {
		if strings.Contains(m, leak) {
			t.Fatalf("التنبيهُ يحمل من الرقم أكثرَ من ثلاثة أرقام (%q):\n%s", leak, m)
		}
	}
	if c := codeLockAlertText(1234); !strings.HasPrefix(c, "🔢 كود التسليم انقفل على الطلب #1234\n") {
		t.Fatalf("نصُّ الكود:\n%s", c)
	}
	if g := mockedAlertText("سامر الأحمد", 77); !strings.HasPrefix(g, "📍 سائق حاول يزيّف موقعه: سامر الأحمد — الطلب #77\n") {
		t.Fatalf("نصُّ التزييف:\n%s", g)
	}
}

func TestSecurityMorningLine_Text(t *testing.T) {
	if got := securityMorningLine(map[string]int64{}); got != "الأمان مبارح: ولا محاولة مشبوهة ✅" {
		t.Fatalf("الفارغ: %q", got)
	}
	got := securityMorningLine(map[string]int64{secLoginFailed: 12, secLoginLocked: 2, secCodeLocked: 1, secGPSMocked: 3, secOTPLimited: 9})
	if got != "الأمان مبارح: ١٢ محاولة دخول غلط · ٢ حساب انقفل · ١ كود تسليم انقفل · ٣ موقع مزيّف" {
		t.Fatalf("السطر: %q", got)
	}
	if got := securityMorningLine(map[string]int64{secForbidden: 4}); !strings.HasSuffix(got, "· ٤ محاولة فتح بيانات الغير") {
		t.Fatalf("فتحُ بيانات الغير وحدَه لا يُقال «ولا شي»: %q", got)
	}
}

// TestSecurityDaily_SurvivesRestart **العدُّ يُكتب باليوم ويقرؤه خادمٌ جديد** —
// فتقريرُ الصباح يرى مبارح ولو أُعيد التشغيلُ ليلاً.
func TestSecurityDaily_SurvivesRestart(t *testing.T) {
	srv := overviewServer(t)
	ctx := context.Background()
	loc := platform.Location()
	day := time.Date(2031, 3, 3, 15, 0, 0, 0, loc)
	t.Cleanup(func() {
		_, _ = srv.pg.Exec(context.Background(), `DELETE FROM security_events_daily WHERE day = '2031-03-03'`)
	})
	_, _ = srv.pg.Exec(ctx, `DELETE FROM security_events_daily WHERE day = '2031-03-03'`)
	for range 7 {
		srv.sec.record(secLoginFailed, day)
	}
	srv.sec.record(secLoginLocked, day)
	srv.sec.record(secGPSMocked, day)
	if err := srv.flushSecurity(ctx); err != nil {
		t.Fatalf("الكتابة: %v", err)
	}
	srv.sec.record(secCodeLocked, day) // في الذاكرة بعد
	if err := srv.flushSecurity(ctx); err != nil {
		t.Fatalf("الكتابة ٢: %v", err)
	}
	fresh := overviewServer(t) // **خادمٌ أُعيد تشغيلُه** — ذاكرتُه فارغة.
	line, ok := fresh.securityMorning(ctx, time.Date(2031, 3, 4, 9, 0, 0, 0, loc))
	want := "الأمان مبارح: ٧ محاولة دخول غلط · ١ حساب انقفل · ١ كود تسليم انقفل · ١ موقع مزيّف"
	if !ok || line != want {
		t.Fatalf("سطرُ الصباح = %q", line)
	}
}
