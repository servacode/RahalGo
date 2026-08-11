package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// ══════════════════════════════════════════════════════════════════════
// **فعلٌ واحدٌ مهما أُعيد إرساله**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١١، البند الرابع.)
//
// **الزبونُ يضغط «اطلب» فينقطع الردُّ في الطريق.** التطبيقُ لا يعلم أوصل
// الطلبُ أم لا **فيعيد الإرسال** — وينشأ طلبان وخصمان.
//
// # ما يُحرَس
//
// **الأوّل** أنّ الفعلَ ينفَّذ مرّةً — وهو بيتُ القصيد.
// **والثاني** أنّ الردَّ المحفوظَ يعود كما هو — **فالتطبيقُ ينتظر رقمَ
// الطلب ليعرضه**، وردٌّ فارغٌ يترك صاحبَه لا يعرف أنجح أم لا.
// **والثالث** أنّ الفاشلَ لا يُحفَظ — **ومن فشل طلبُه لانقطاعٍ لحظةً يجب
// أن يستطيع الإعادة.**
// **والرابع** أنّ مفتاحَ زيدٍ لا يمنع فعلَ عمرو.

func idemFixture(t *testing.T) (*Server, string) {
	t.Helper()
	pool := testdb.Pool(t)
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1}))
	return &Server{pg: pool, logger: quiet}, testdb.NewUser(t, pool, "customer")
}

// call ينادي معالِجاً ملفوفاً بمفتاحٍ ويعيد الرمزَ والنصّ.
func call(t *testing.T, srv *Server, uid, key string, h http.HandlerFunc) (int, string) {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader("{}"))
	if key != "" {
		r.Header.Set(idempotencyHeader, key)
	}
	r = r.WithContext(context.WithValue(r.Context(), ctxUserID, uid))
	w := httptest.NewRecorder()
	srv.idempotent(h)(w, r)
	return w.Code, w.Body.String()
}

func TestIdempotency_RunsOnce(t *testing.T) {
	srv, uid := idemFixture(t)
	var runs int
	h := func(w http.ResponseWriter, r *http.Request) {
		runs++
		httpx.JSON(w, http.StatusOK, map[string]any{"order_number": 1234})
	}

	code1, body1 := call(t, srv, uid, "key-a", h)
	code2, body2 := call(t, srv, uid, "key-a", h)

	if runs != 1 {
		t.Fatalf("نُفّذ الفعلُ %d مرّاتٍ من ١ — **فضغطةٌ على شبكةٍ سيّئة تُنشئ "+
			"طلبين وخصمين**", runs)
	}
	if code1 != code2 {
		t.Fatalf("الرمزُ تبدّل بين المحاولتين: %d ثمّ %d", code1, code2)
	}
	// **والردُّ نفسُه** — التطبيقُ ينتظر رقمَ الطلب ليعرضه.
	if body1 != body2 {
		t.Fatalf("الردُّ تبدّل — **فصاحبُه لا يعرف أنجح طلبُه أم لا**\nالأوّل: %s\nالثاني: %s",
			body1, body2)
	}
	if !strings.Contains(body2, "1234") {
		t.Fatalf("الردُّ المُعاد بلا بيانات: %s", body2)
	}
}

// TestIdempotency_DifferentKeysBothRun **ومفتاحان مختلفان فعلان مختلفان.**
//
// **والزبونُ قد يطلب مرّتين بالمحتوى نفسِه في دقيقة — وهذا حقُّه.**
func TestIdempotency_DifferentKeysBothRun(t *testing.T) {
	srv, uid := idemFixture(t)
	var runs int
	h := func(w http.ResponseWriter, r *http.Request) {
		runs++
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
	}

	call(t, srv, uid, "key-1", h)
	call(t, srv, uid, "key-2", h)
	if runs != 2 {
		t.Fatalf("نُفّذ %d من ٢ — **فمن أراد طلبين مقصودين حُرم الثاني**", runs)
	}
}

// TestIdempotency_KeyIsPerUser **ومفتاحُ زيدٍ لا يمنع فعلَ عمرو.**
//
// **ولولا ربطُه بصاحبه** لَاستطاع أحدٌ أن يُعطّل فعلَ غيره بمفتاحٍ يخمّنه،
// **ولَتلقّى ردّاً ليس له.**
func TestIdempotency_KeyIsPerUser(t *testing.T) {
	srv, first := idemFixture(t)
	second := testdb.NewUser(t, srv.pg, "customer")
	var runs int
	h := func(w http.ResponseWriter, r *http.Request) {
		runs++
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
	}

	call(t, srv, first, "same-key", h)
	call(t, srv, second, "same-key", h)
	if runs != 2 {
		t.Fatalf("نُفّذ %d من ٢ — **فمفتاحُ حسابٍ يقفل فعلَ حسابٍ آخر**", runs)
	}
}

// TestIdempotency_FailureIsRetryable **والفاشلُ لا يُحفَظ.**
//
// **وردٌّ بخطأٍ محفوظٌ يعني أنّ صاحبَه يتلقّى الخطأَ نفسَه إلى الأبد** —
// فمن فشل طلبُه لانقطاع قاعدةٍ لحظةً لا يستطيع أن يعيد أبداً.
func TestIdempotency_FailureIsRetryable(t *testing.T) {
	srv, uid := idemFixture(t)
	var runs int
	h := func(w http.ResponseWriter, r *http.Request) {
		runs++
		if runs == 1 {
			httpx.Error(w, &httpx.AppError{Status: 500, Code: "internal", MessageKey: "errors.internal"})
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
	}

	if code, _ := call(t, srv, uid, "key-f", h); code != 500 {
		t.Fatalf("المحاولةُ الأولى ردّت %d لا ٥٠٠", code)
	}
	code, _ := call(t, srv, uid, "key-f", h)
	if code != http.StatusOK {
		t.Fatalf("الإعادةُ بعد فشلٍ ردّت %d — **فمن فشل طلبُه لحظةً لا يستطيع أن يعيد أبداً**", code)
	}
	if runs != 2 {
		t.Fatalf("نُفّذ %d من ٢", runs)
	}
}

// TestIdempotency_NoHeaderPassesThrough **وبلا ترويسةٍ يمرّ كما كان.**
//
// **والويبُ لا يرسلها** — ونقطةٌ ترفض من لا يرسل مفتاحاً تكسر كلَّ شاشةٍ
// تعمل اليوم.
func TestIdempotency_NoHeaderPassesThrough(t *testing.T) {
	srv, uid := idemFixture(t)
	var runs int
	h := func(w http.ResponseWriter, r *http.Request) {
		runs++
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
	}
	call(t, srv, uid, "", h)
	call(t, srv, uid, "", h)
	if runs != 2 {
		t.Fatalf("نُفّذ %d من ٢ بلا ترويسة — **فالويبُ الذي لا يرسلها ينكسر**", runs)
	}
}

// TestIdempotency_ConcurrentSecondGets409 **ونداءان معاً: الثاني ينتظر.**
//
// **و409 لا 500**: الحالةُ سليمةٌ ومؤقّتة، **ومن رأى خمسمئة ظنّ المنصّةَ
// معطوبةً وأعاد بمفتاحٍ جديد** — وذاك بالضبط ما نمنعه.
func TestIdempotency_ConcurrentSecondGets409(t *testing.T) {
	srv, uid := idemFixture(t)
	release := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once
	h := func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(started) })
		<-release
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
	}

	go func() { call(t, srv, uid, "key-c", h) }()
	<-started

	code, _ := call(t, srv, uid, "key-c", func(w http.ResponseWriter, r *http.Request) {
		t.Error("نُفّذ الفعلُ مرّةً ثانيةً وهو ما زال يُنفَّذ")
	})
	close(release)

	if code != http.StatusConflict {
		t.Fatalf("النداءُ المتزامن ردّ %d لا ٤٠٩ — **ومن رأى خمسمئة ظنّ المنصّةَ "+
			"معطوبةً وأعاد بمفتاحٍ جديد**", code)
	}
}
