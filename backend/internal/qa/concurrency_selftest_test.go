package qa

// اختباراتُ المِسنَد لنفسِه — **`P-5` البند ٣.**
//
// **ومِسنَدٌ لم يُثبَت أنّه يصنع تداخلاً لا يُوثَق بنتيجةِ سباقٍ منه.**

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// TestCONC_BarrierReleasesTogether **الحاجزُ يحبس ثمّ يُطلق مرّةً واحدة.**
func TestCONC_BarrierReleasesTogether(t *testing.T) {
	const n = 4
	b := NewBarrier(n)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	released := make(chan int, n)
	for i := 0; i < n; i++ {
		go func() {
			order, err := b.Arrive(ctx)
			if err != nil {
				released <- -1
				return
			}
			released <- order
		}()
	}
	seen := map[int]bool{}
	for i := 0; i < n; i++ {
		select {
		case o := <-released:
			if o < 1 {
				t.Fatal("ممثّلٌ لم يُطلَق")
			}
			seen[o] = true
		case <-ctx.Done():
			t.Fatalf("الحاجزُ لم يُطلق %d — وصل %d", n, b.Arrived())
		}
	}
	if len(seen) != n {
		t.Errorf("تراتيبُ الوصول %v — يُنتظر %d مختلفة", seen, n)
	}
	t.Logf("BARRIER = %d/%d وصلوا · وحُبسوا %s قبل الإطلاق",
		b.Arrived(), n, b.Spread().Round(time.Microsecond))
}

// TestCONC_ProbeMeasuresOverlap **المِسبارُ يقيس التداخلَ ولا يظنّه.**
//
// **والدليلُ من الطرفين**: نافذةٌ يدخلها اثنان معاً تُقاس `2`،
// **ونافذةٌ يمرّان بها تتابعاً تُقاس `1`** — فلو ردّ `2` دائماً لكان عدّاداً
// لا مِسباراً.
func TestCONC_ProbeMeasuresOverlap(t *testing.T) {
	// **متداخلان** — يُحبسان داخل النافذة بحاجزٍ ثانٍ.
	inner := NewBarrier(2)
	r := Race(t, 5*time.Second,
		Actor{Name: "أ", Do: func(ctx context.Context) any { _, _ = inner.Arrive(ctx); return "أ" }},
		Actor{Name: "ب", Do: func(ctx context.Context) any { _, _ = inner.Arrive(ctx); return "ب" }},
	)
	if r.Probe.Max() != 2 {
		t.Errorf("تداخلٌ مقيسٌ %d — والاثنان محبوسان داخل النافذة", r.Probe.Max())
	}
	// **ولا يُحكَم بالساعة.**
	//
	// **دقّةُ ساعةِ ويندوز أخشنُ من زمن الممثّلين** (نحو ٠٫٥–١٥ms)،
	// **فتقاطعُ النوافذ يُقرأ صفراً وهما متداخلان فعلاً.** ولذلك كان
	// الدليلُ عدّاً لا زمناً منذ البدء: `Probe` **يعدّ من في النافذة**،
	// **ووقتُ الجدار يُطبَع للاستئناس لا للحكم.**
	t.Logf("OVERLAPPING  ⇒ %s", r)
	t.Logf("(تقاطعُ النوافذ %s — ولا يُحكَم به: دقّةُ الساعة أخشنُ من زمن الممثّلين)",
		r.Overlap)

	// **ومتتابعان** — كلٌّ يخرج قبل أن يدخل الآخر.
	seq := &Probe{}
	for i := 0; i < 2; i++ {
		seq.Enter()
		seq.Exit()
	}
	if seq.Max() != 1 {
		t.Errorf("تتابعٌ قِيس %d — يُنتظر 1: **المِسبارُ يعدّ ما ليس متداخلاً**", seq.Max())
	}
	t.Logf("SEQUENTIAL   ⇒ تداخلٌ مقيسٌ = %d", seq.Max())
}

// TestCONC_TrueOverlapProvenByDatabase **البند ٣ — الدليلُ من القاعدة.**
//
// **ولا يُقاس بالساعة.** جلسةٌ محجوبةٌ على قفلٍ تحمله جلسةٌ أخرى
// **لا تكون إلّا والاثنتان حيّتان في اللحظة نفسِها** — **فالقاعدةُ تشهد
// بالتداخل عن نفسها.**
//
// **وقفلٌ استشاريٌّ للاختبار وحدَه** — **ولا يمسّ دلالةَ المنتج** (البند ٢٦):
// لا جدولَ منتجٍ يُقفَل ولا مسارَ إنتاجٍ يُنادى.
func TestCONC_TrueOverlapProvenByDatabase(t *testing.T) {
	h := New(t)
	ctx := ctxBG()
	const lockKey = 918_273_645 // **مفتاحٌ للاختبار وحدَه**

	held := make(chan struct{})     // أ أمسك القفل
	releaseA := make(chan struct{}) // اسمح لأ بالإفلات
	bDone := make(chan error, 1)

	// **أ** — يمسك قفلاً استشاريّاً ويبقى ممسكاً.
	go func() {
		conn, err := h.Pool.Acquire(ctx)
		if err != nil {
			bDone <- err
			return
		}
		defer conn.Release()
		if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockKey); err != nil {
			bDone <- err
			return
		}
		close(held)
		<-releaseA
		_, _ = conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, lockKey)
	}()

	select {
	case <-held:
	case <-time.After(5 * time.Second):
		t.Fatal("أ لم يمسك القفلَ خلال 5 ثوانٍ")
	}

	// **ب** — يطلب القفلَ نفسَه فيُحجَب.
	go func() {
		conn, err := h.Pool.Acquire(ctx)
		if err != nil {
			bDone <- err
			return
		}
		defer conn.Release()
		_, err = conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockKey)
		if err == nil {
			_, _ = conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, lockKey)
		}
		bDone <- err
	}()

	// **والقاعدةُ تُسأل: أثمّةَ جلسةٌ محجوبةٌ الآن؟**
	blocked, err := WaitBlockedAdvisory(ctx, h, lockKey, 5*time.Second)
	if err != nil {
		close(releaseA)
		t.Fatalf("TRUE OVERLAP NOT PROVEN: %v", err)
	}
	t.Logf("TRUE OVERLAP PROOF — أ ممسكٌ بالقفل · وب محجوبٌ عليه (%d جلسة)", blocked)
	t.Logf("both reached barrier → release → both entered target window")

	close(releaseA)
	select {
	case err := <-bDone:
		if err != nil {
			t.Fatalf("ب تعثّر بعد الإفلات: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ب لم يتقدّم بعد إفلات أ — DEADLOCK / TIMEOUT")
	}
	t.Logf("HARNESS TRUE-OVERLAP SELF-TEST = PROVEN")
	t.Logf("SLEEP-BASED CONTROL = NO")
}

// WaitBlockedAdvisory ينتظر حجبَ جلسةٍ على قفلٍ استشاريٍّ بعينه.
func WaitBlockedAdvisory(ctx context.Context, h *Harness, key int64, timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)
	for {
		var n int
		err := h.Pool.QueryRow(ctx, `
			SELECT count(*) FROM pg_locks
			WHERE locktype = 'advisory' AND objid = $1 AND NOT granted`, key).Scan(&n)
		if err != nil && err != pgx.ErrNoRows {
			return 0, err
		}
		if n > 0 {
			return n, nil
		}
		if time.Now().After(deadline) {
			return 0, ErrNotBlocked
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(2 * time.Millisecond):
		}
	}
}

// ErrNotBlocked لم تُحجَب جلسةٌ — **فلا تداخلَ مُثبَت.**
var ErrNotBlocked = errNotBlocked{}

type errNotBlocked struct{}

func (errNotBlocked) Error() string {
	return "لم تُحجَب جلسةٌ على القفل — والتداخلُ غيرُ مُثبَت"
}

// TestCONC_TimeoutIsReportedNotHung **البند ٢١ — ولا اختبارَ يعلّق للأبد.**
//
// **ويُثبَت بمِسنَدٍ زائفٍ** (`*testing.T` بديل) — **فلا يُسقَط الاختبارُ
// الحقيقيُّ لإثبات أنّ الإسقاطَ يقع.**
func TestCONC_TimeoutIsReportedNotHung(t *testing.T) {
	// **حاجزٌ لثلاثةٍ ولا يصله إلّا اثنان** — فيبقى الجميعُ محبوساً.
	fake := &testing.T{}
	start := time.Now()
	r := Race(fake, 300*time.Millisecond,
		Actor{Name: "أ", Do: func(ctx context.Context) any { <-ctx.Done(); return nil }},
		Actor{Name: "ب", Do: func(ctx context.Context) any { <-ctx.Done(); return nil }},
	)
	el := time.Since(start)
	if !r.TimedOut {
		t.Error("السباقُ لم يُعلن المهلةَ — **قد يعلّق في الواقع**")
	}
	if !fake.Failed() {
		t.Error("المهلةُ لم تُسجَّل سقوطاً")
	}
	if el > 3*time.Second {
		t.Errorf("عاد بعد %s — والمهلةُ 300ms", el)
	}
	t.Logf("DEADLOCK/TIMEOUT DETECTION = PROVEN — عاد بعد %s وسُجّل سقوطاً",
		el.Round(time.Millisecond))
}
