package server

// ══════════════════════════════════════════════════════════════════════
// **عدّادُ أخطاء الخادم** (مراقبةُ المنصّة — الدفعةُ الأولى ٢٠٢٦-١٠-٠٩)
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **خطأُ ٥٠٠ يُكتب سطراً في السجلّ ولا يعدّه أحد** — فمسارٌ ينهار مئةَ مرّةٍ
// في الساعة لا يُعرف إلّا إن فتح أحدٌ السجلَّ وقرأه.
//
// # وما صار
//
//   - **وسيطٌ يعدّ كلَّ ردٍّ ٥xx** بنمط مساره ورمزه — في الذاكرة، بقفلٍ قصير.
//   - **والراصدُ يكتب ما تجمّع في الجدول دفعةً كلَّ دقيقة** (`server_errors`).
//   - **وحلقةٌ بآخر الأوقات** تقول كم خطأً في آخر عشر دقائق — للإنذار بالقفزة.
//   - **وبابٌ** `GET /admin/monitoring/errors` يعرضها مجمَّعةً.
//
// # والكتابةُ رخيصةٌ لا تنتظر
//
// **لا قاعدةَ في مسار الردّ** — قفلٌ وزيادةُ عدّاد. **فخادمٌ يتعثّر لا يُثقَل
// بكتابةٍ مع كلّ خطأ**، وإن وقعت القاعدةُ بقي العدُّ في الذاكرة حتّى تعود.
//
// **ولا رابطَ خامّاً** — نمطُ المسار (`/orders/{id}`) لا الرابط: لا معرّفَ
// ولا هاتفَ ولا نصَّ استعلام.

import (
	"bytes"
	"context"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// errRingSize **كم وقتاً تحفظ الحلقة** — يكفي لقفزةٍ أكبرَ من أيّ حدٍّ معقول.
const errRingSize = 2048

// errPendingCap **سقفُ المفاتيح المعلّقة في الذاكرة** — والقاعدةُ واقعةٌ طويلاً.
// **وما زاد يُعدّ في الحلقة ولا يُحفظ** — فلا تنمو الذاكرةُ بلا حدّ.
const errPendingCap = 5000

type errKey struct {
	Hour  time.Time
	Route string
	Code  int
}

type errAgg struct {
	Count       int64
	First, Last time.Time
}

// errorTracker **عدّادُ الأخطاء في الذاكرة** — وصفرُه صالحٌ للعمل.
type errorTracker struct {
	mu      sync.Mutex
	pending map[errKey]*errAgg
	ring    [errRingSize]time.Time
	next    int
	total   int64
}

// record **يعدّ خطأً واحداً** — قفلٌ وزيادة، ولا شيءَ غيرَهما.
func (t *errorTracker) record(route string, code int, now time.Time) {
	if route == "" {
		route = "unmatched"
	}
	k := errKey{Hour: now.UTC().Truncate(time.Hour), Route: route, Code: code}
	t.mu.Lock()
	defer t.mu.Unlock()
	// **وردودُ «مشغولة» والقاعدةُ واقفةٌ لا تدخل حلقةَ القفزة** — للقاعدة
	// تنبيهُها، **وإلّا وصل «أخطاءٌ كثيرة» بعد عودتها مباشرةً** عن انقطاعٍ انتهى.
	if !dbDownFlag.Load() {
		t.ring[t.next] = now
		t.next = (t.next + 1) % errRingSize
	}
	t.total++
	if t.pending == nil {
		t.pending = map[errKey]*errAgg{}
	}
	if a, ok := t.pending[k]; ok {
		a.Count++
		a.Last = now
		return
	}
	if len(t.pending) >= errPendingCap {
		return
	}
	t.pending[k] = &errAgg{Count: 1, First: now, Last: now}
}

// countSince **كم خطأً منذ لحظة** — من الحلقة.
func (t *errorTracker) countSince(since time.Time) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := 0
	for _, at := range t.ring {
		if !at.IsZero() && !at.Before(since) {
			n++
		}
	}
	return n
}

// take **يأخذ المعلَّقَ ويُفرغه** — ويُعاد إن فشلت الكتابة (`putBack`).
func (t *errorTracker) take() map[errKey]*errAgg {
	t.mu.Lock()
	defer t.mu.Unlock()
	p := t.pending
	t.pending = nil
	return p
}

// putBack **يُرجع ما لم يُكتب** — فيُضاف إلى ما تجمّع بعده.
func (t *errorTracker) putBack(p map[errKey]*errAgg) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pending == nil {
		t.pending = map[errKey]*errAgg{}
	}
	for k, a := range p {
		if cur, ok := t.pending[k]; ok {
			cur.Count += a.Count
			if a.First.Before(cur.First) {
				cur.First = a.First
			}
			if a.Last.After(cur.Last) {
				cur.Last = a.Last
			}
			continue
		}
		t.pending[k] = a
	}
}

// countErrors **الوسيطُ** — يُلبس الردَّ غلافاً يحفظ رمزَه، ويعدّ ٥xx بعده.
//
// **ويُركَّب قبل `Recoverer`** — فالذعرُ الذي يصير ٥٠٠ يُعدّ أيضاً.
// **والغلافُ غلافُ chi نفسُه** — يحفظ `Hijacker` و`Flusher`، فلا ينكسر البثّ.
func (s *Server) countErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		var head headCapture
		ww.Tee(&head)
		next.ServeHTTP(ww, r)
		code := ww.Status()
		if code < 500 {
			return
		}
		if deliberateRefusal(head.buf) {
			return
		}
		route := ""
		if rc := chi.RouteContext(r.Context()); rc != nil {
			route = rc.RoutePattern()
		}
		s.errs.record(route, code, time.Now())
	})
}

// flushErrors **يكتب ما تجمّع في الجدول** — دفعةً واحدة.
func (s *Server) flushErrors(ctx context.Context) error {
	p := s.errs.take()
	if len(p) == 0 || s.pg == nil {
		if len(p) > 0 {
			s.errs.putBack(p)
		}
		return nil
	}
	for k, a := range p {
		_, err := s.pg.Exec(ctx, `
			INSERT INTO server_errors (hour, route, code, count, first_seen, last_seen)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (hour, route, code) DO UPDATE SET
				count = server_errors.count + EXCLUDED.count,
				first_seen = LEAST(server_errors.first_seen, EXCLUDED.first_seen),
				last_seen = GREATEST(server_errors.last_seen, EXCLUDED.last_seen)`,
			k.Hour, k.Route, k.Code, a.Count, a.First, a.Last)
		if err != nil {
			// **وما لم يُكتب يعود إلى الذاكرة** — يُكتب في الدقيقة التالية.
			rest := map[errKey]*errAgg{}
			for k2, a2 := range p {
				rest[k2] = a2
			}
			s.errs.putBack(rest)
			return err
		}
		delete(p, k)
	}
	return nil
}

// errorGroup **خطأٌ مجمَّعٌ بمساره ورمزه** — كما يعرضه البابُ والتقرير.
type errorGroup struct {
	Route     string    `json:"route"`
	Code      int       `json:"code"`
	Count     int64     `json:"count"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

type errorsSummary struct {
	Since      time.Time    `json:"since"`
	Total      int64        `json:"total"`
	LastHour   int64        `json:"last_hour"`
	Last10Min  int          `json:"last_10_min"`
	Groups     []errorGroup `json:"groups"`
	FlushError bool         `json:"flush_error"`
}

// errorsSince **الأخطاءُ مجمَّعةً منذ لحظة** — من الجدول بعد كتابة المعلَّق.
func (s *Server) errorsSince(ctx context.Context, since, now time.Time) (errorsSummary, error) {
	out := errorsSummary{Since: since, Groups: []errorGroup{}}
	if err := s.flushErrors(ctx); err != nil {
		out.FlushError = true
	}
	out.Last10Min = s.errs.countSince(now.Add(-10 * time.Minute))
	rows, err := s.pg.Query(ctx, `
		SELECT route, code, sum(count), min(first_seen), max(last_seen)
		FROM server_errors WHERE last_seen >= $1
		GROUP BY route, code`, since)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var g errorGroup
		if err := rows.Scan(&g.Route, &g.Code, &g.Count, &g.FirstSeen, &g.LastSeen); err != nil {
			return out, err
		}
		out.Total += g.Count
		out.Groups = append(out.Groups, g)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	// **الأكثرُ أوّلاً** — وما تساوى فالأحدث.
	sort.Slice(out.Groups, func(i, j int) bool {
		if out.Groups[i].Count != out.Groups[j].Count {
			return out.Groups[i].Count > out.Groups[j].Count
		}
		return out.Groups[i].LastSeen.After(out.Groups[j].LastSeen)
	})
	if len(out.Groups) > 50 {
		out.Groups = out.Groups[:50]
	}
	if err := s.pg.QueryRow(ctx, `
		SELECT COALESCE(sum(count), 0) FROM server_errors WHERE last_seen >= $1`,
		now.Add(-time.Hour)).Scan(&out.LastHour); err != nil {
		return out, err
	}
	return out, nil
}

// handleMonitoringErrors **أخطاءُ الخادم آخرَ يوم** — `GET /admin/monitoring/errors`
// بقدرة `observability.read`. و`?hours=` بين ١ و١٦٨ (افتراضُه ٢٤).
func (s *Server) handleMonitoringErrors(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := timeoutCtx(r, 8*time.Second)
	defer cancel()
	// **وما خرج عن المدى يُقصّ إليه** — لا خطأَ على رقمٍ في رابط.
	hours := 24
	if n, err := strconv.Atoi(r.URL.Query().Get("hours")); err == nil {
		hours = min(max(n, 1), 168)
	}
	now := time.Now()
	out, err := s.errorsSince(ctx, now.Add(-time.Duration(hours)*time.Hour), now)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

// ══════════════════════════════════════════════════════════════════════
//
//	**الرفضُ المقصود ليس عطلاً** (تنبيهٌ كاذبٌ ٢٠٢٦-١٠-١٠)
//
// ══════════════════════════════════════════════════════════════════════
//
// **وصل المالكَ «٢١ خطأ بآخر عشر دقايق»** — وكلُّها `delivery-quote (503)`:
// متجرٌ على شاشة «لدي توصيلة» يسأل عن الأجرة والاستقبالُ مغلق. **والخادمُ
// أجاب كما ينبغي** — «لا يمكن الآن» — **فعدَّه العدّادُ عطلاً لأنّ رمزَه ٥٠٣.**
//
// **فما يردّه الخادمُ عمداً لا يُعدّ**: إغلاقُ المنصّة أو المنطقة، وقبلَ
// الافتتاح، ولا تغطية، ولا سائقَ في الدوام. **والعطبُ الحقيقيُّ باقٍ
// يُعدّ**: `service_busy` (القاعدة)، و`otp_send_failed` (الواتساب)، والذعر.
var deliberateRefusalCodes = []string{
	"platform_closed_now", "temporarily_unavailable", "launch_closed",
	"zone_closed_now", "coverage_unavailable", "no_drivers_on_shift",
}

func deliberateRefusal(head []byte) bool {
	if len(head) == 0 {
		return false
	}
	for _, c := range deliberateRefusalCodes {
		if bytes.Contains(head, []byte(`"code":"`+c+`"`)) {
			return true
		}
	}
	return false
}

// headCapture **يحفظ أوّلَ ٥١٢ بايتاً من الردّ ولا يزيد** — رمزُ الخطأ في
// رأس الجسم، **والبثُّ والملفّاتُ لا تُنسخ.**
type headCapture struct{ buf []byte }

func (h *headCapture) Write(p []byte) (int, error) {
	if room := 512 - len(h.buf); room > 0 {
		if len(p) < room {
			room = len(p)
		}
		h.buf = append(h.buf, p[:room]...)
	}
	return len(p), nil
}
