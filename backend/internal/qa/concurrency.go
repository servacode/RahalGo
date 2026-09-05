package qa

// مِسنَدُ التزامن — **`P-5`.**
//
// # لماذا لا يكفي خيطان
//
// **«أطلقتُ goroutineين معاً» ليس دليلاً على تداخل.** جدولةُ Go لا تَعِد
// بشيء، **وخيطٌ قد ينتهي قبل أن يبدأ الآخر** — فيُسمّى الاختبارُ سباقاً
// وهو تتابعٌ سريع. **واختبارُ سباقٍ ينجح بالصدفة أسوأُ من لا اختبار**:
// يعطي طمأنينةً لا يسندها شيء.
//
// **فالتداخلُ هنا يُصنَع لا يُرجى**: حاجزٌ يمنع الجميعَ حتّى يصل الجميع،
// **ثمّ يُفتح مرّةً واحدة.** **ومِسبارٌ يقيس كم كان في النافذة معاً** —
// فيُقال «اثنان» بالعدّ لا بالظنّ.
//
// **ولا `sleep` مِقوداً** (البند ٢٧): النومُ مهلةٌ وحدَها، **والتحكّمُ
// بأدوات المزامنة.**

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ══════════════════════════════════════════════════════════════════════
// **الحاجز**
// ══════════════════════════════════════════════════════════════════════

// Barrier حاجزٌ يحبس `n` ممثّلين حتّى يصلوا جميعاً ثمّ يُطلقهم معاً.
//
// **ولا يُطلَق واحدٌ قبل وصول الأخير** — **وهذا هو الفرقُ بين سباقٍ
// مصنوعٍ وتتابعٍ سريع.**
type Barrier struct {
	n       int
	mu      sync.Mutex
	arrived int
	release chan struct{}
	// firstAt متى وصل أوّلُهم · lastAt متى وصل آخرُهم — دليلٌ يُطبع.
	firstAt, lastAt time.Time
}

// NewBarrier حاجزٌ لـ`n`.
func NewBarrier(n int) *Barrier {
	return &Barrier{n: n, release: make(chan struct{})}
}

// Arrive يُعلن الوصولَ ثمّ ينتظر الإطلاق — **ويردّ ترتيبَ وصوله.**
func (b *Barrier) Arrive(ctx context.Context) (order int, err error) {
	b.mu.Lock()
	b.arrived++
	order = b.arrived
	now := time.Now()
	if order == 1 {
		b.firstAt = now
	}
	if order == b.n {
		b.lastAt = now
		close(b.release) // **الإطلاقُ مرّةً واحدةً للجميع**
	}
	b.mu.Unlock()

	select {
	case <-b.release:
		return order, nil
	case <-ctx.Done():
		return order, fmt.Errorf("الحاجز: انتظارٌ انقطع بعد وصول %d من %d", b.Arrived(), b.n)
	}
}

// Arrived كم وصل.
func (b *Barrier) Arrived() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.arrived
}

// Spread الفرقُ بين أوّلِ واصلٍ وآخرِه — **زمنُ الحبس.**
func (b *Barrier) Spread() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.firstAt.IsZero() || b.lastAt.IsZero() {
		return 0
	}
	return b.lastAt.Sub(b.firstAt)
}

// ══════════════════════════════════════════════════════════════════════
// **المِسبار — كم كان في النافذة معاً**
// ══════════════════════════════════════════════════════════════════════

// Probe يعدّ من دخل النافذةَ ولم يخرج — **ويحفظ أقصى ما بلغه.**
//
// **وهو الدليلُ لا الظنّ**: `Max() == 2` تعني أنّ اثنين كانا في النافذة
// **في لحظةٍ واحدة**، **و`Max() == 1` تعني أنّ ما جرى تتابعٌ لا تداخل.**
type Probe struct {
	cur, max atomic.Int64
}

// Enter دخولُ النافذة.
func (p *Probe) Enter() {
	c := p.cur.Add(1)
	for {
		m := p.max.Load()
		if c <= m || p.max.CompareAndSwap(m, c) {
			return
		}
	}
}

// Exit خروجٌ منها.
func (p *Probe) Exit() { p.cur.Add(-1) }

// Max أقصى تداخلٍ قِيس.
func (p *Probe) Max() int { return int(p.max.Load()) }

// ══════════════════════════════════════════════════════════════════════
// **الممثّلون والنتائج**
// ══════════════════════════════════════════════════════════════════════

// Actor ممثّلٌ في سباق.
type Actor struct {
	Name string
	// Do ما يفعله — **يُنادى بعد فتح الحاجز لا قبله.**
	Do func(ctx context.Context) any
}

// Outcome ما خرج به ممثّلٌ واحد.
type Outcome struct {
	Name       string
	Value      any
	Panic      any
	Start, End time.Time
}

// Dur كم استغرق.
func (o Outcome) Dur() time.Duration { return o.End.Sub(o.Start) }

// RaceResult حصيلةُ سباق.
type RaceResult struct {
	Outcomes []Outcome
	Barrier  *Barrier
	Probe    *Probe
	// Overlap تقاطعُ نوافذ الممثّلين — **صفرٌ يعني تتابعاً.**
	Overlap time.Duration
	// TimedOut أعلق السباق.
	TimedOut bool
}

// Values قِيمُ الممثّلين بترتيب تعريفهم.
func (r *RaceResult) Values() []any {
	out := make([]any, len(r.Outcomes))
	for i, o := range r.Outcomes {
		out[i] = o.Value
	}
	return out
}

// Codes رموزُ الاستجابة حين تكون القيمُ `Res`.
func (r *RaceResult) Codes() []int {
	var out []int
	for _, o := range r.Outcomes {
		if res, ok := o.Value.(Res); ok {
			out = append(out, res.Code)
		}
	}
	return out
}

// CountCode كم ممثّلاً ردَّ هذا الرمز.
func (r *RaceResult) CountCode(code int) int {
	n := 0
	for _, c := range r.Codes() {
		if c == code {
			n++
		}
	}
	return n
}

// CountOK كم ممثّلاً نجح (أقلّ من 400).
func (r *RaceResult) CountOK() int {
	n := 0
	for _, c := range r.Codes() {
		if c < 400 {
			n++
		}
	}
	return n
}

func (r *RaceResult) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "تداخلٌ مقيسٌ = %d · تقاطعُ النوافذ = %s · حبسُ الحاجز = %s",
		r.Probe.Max(), r.Overlap.Round(time.Microsecond), r.Barrier.Spread().Round(time.Microsecond))
	for _, o := range r.Outcomes {
		fmt.Fprintf(&b, "\n  %-10s %v (%s)", o.Name, brief(o.Value), o.Dur().Round(time.Microsecond))
		if o.Panic != nil {
			fmt.Fprintf(&b, " ← ذُعرٌ: %v", o.Panic)
		}
	}
	return b.String()
}

func brief(v any) string {
	if res, ok := v.(Res); ok {
		s := res.String()
		if len(s) > 70 {
			s = s[:70] + "…"
		}
		return s
	}
	return fmt.Sprint(v)
}

// ══════════════════════════════════════════════════════════════════════
// **السباق**
// ══════════════════════════════════════════════════════════════════════

// DefaultRaceTimeout مهلةُ السباق — **ولا اختبارَ يعلّق إلى الأبد.**
const DefaultRaceTimeout = 30 * time.Second

// Race يُطلق الممثّلين من حاجزٍ واحدٍ ويجمع نتائجَهم.
//
// **ويسقط بـ`DEADLOCK / TIMEOUT` لا يعلّق** (البند ٢١) — **ومعه
// تشخيصٌ**: من وصل الحاجزَ ومن لم يصل، وكومةُ الخيوط.
func Race(t *testing.T, timeout time.Duration, actors ...Actor) *RaceResult {
	t.Helper()
	if timeout <= 0 {
		timeout = DefaultRaceTimeout
	}
	n := len(actors)
	res := &RaceResult{
		Outcomes: make([]Outcome, n),
		Barrier:  NewBarrier(n),
		Probe:    &Probe{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(n)
	for i, a := range actors {
		go func(i int, a Actor) {
			defer wg.Done()
			defer func() {
				if p := recover(); p != nil {
					res.Outcomes[i].Panic = p
					res.Outcomes[i].End = time.Now()
					res.Probe.Exit()
				}
			}()
			res.Outcomes[i].Name = a.Name
			if _, err := res.Barrier.Arrive(ctx); err != nil {
				res.Outcomes[i].Panic = err
				return
			}
			// **ومن هنا فصاعداً هم في النافذة معاً.**
			res.Outcomes[i].Start = time.Now()
			res.Probe.Enter()
			v := a.Do(ctx)
			res.Probe.Exit()
			res.Outcomes[i].End = time.Now()
			res.Outcomes[i].Value = v
		}(i, a)
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-ctx.Done():
		res.TimedOut = true
		t.Errorf("DEADLOCK / TIMEOUT بعد %s\n"+
			"وصل الحاجزَ %d من %d\n%s",
			timeout, res.Barrier.Arrived(), n, goroutineDump())
		return res
	}

	res.Overlap = intersect(res.Outcomes)
	return res
}

// intersect تقاطعُ نوافذ الممثّلين جميعاً — **آخرُ بدايةٍ حتّى أوّلِ نهاية.**
func intersect(os []Outcome) time.Duration {
	var lastStart, firstEnd time.Time
	for i, o := range os {
		if o.Start.IsZero() || o.End.IsZero() {
			return 0
		}
		if i == 0 || o.Start.After(lastStart) {
			lastStart = o.Start
		}
		if i == 0 || o.End.Before(firstEnd) {
			firstEnd = o.End
		}
	}
	if d := firstEnd.Sub(lastStart); d > 0 {
		return d
	}
	return 0
}

func goroutineDump() string {
	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, true)
	s := string(buf[:n])
	if len(s) > 4000 {
		s = s[:4000] + "\n… (قُصّت)"
	}
	return s
}

// ══════════════════════════════════════════════════════════════════════
// **دليلُ التداخل من القاعدة نفسِها**
// ══════════════════════════════════════════════════════════════════════

// WaitBlocked ينتظر حتّى تصير جلسةٌ محجوبةً على قفلٍ في القاعدة.
//
// **وهذا أقوى دليلٍ على تداخلٍ حقيقيّ**: لا يُقاس بالساعة بل **تقوله
// القاعدةُ عن نفسها** — جلسةٌ تنتظر قفلاً تحمله جلسةٌ أخرى **لا تكون
// إلّا والاثنتان حيّتان معاً.**
//
// **وليس نوماً مِقوداً** (البند ٢٧): هذا انتظارُ شرطٍ يُقرأ من القاعدة،
// **والمهلةُ حدٌّ لا آلةُ تحكّم.**
func WaitBlocked(ctx context.Context, pool *pgxpool.Pool, want int, timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)
	for {
		var n int
		err := pool.QueryRow(ctx, `
			SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database()
			  AND wait_event_type = 'Lock'
			  AND state = 'active'`).Scan(&n)
		if err != nil {
			return 0, err
		}
		if n >= want {
			return n, nil
		}
		if time.Now().After(deadline) {
			return n, fmt.Errorf("لم تُحجَب %d جلسةٍ خلال %s — الحاجبُ %d", want, timeout, n)
		}
		select {
		case <-ctx.Done():
			return n, ctx.Err()
		case <-time.After(2 * time.Millisecond): // **نبضُ استطلاعٍ لا تحكّم**
		}
	}
}
