// تشخيصُ سباق دفعات الموقع — **دورةُ ٥٢، ولا شيفرةَ منتَجٍ فيها.**
//
// # ما وقع
//
// **`TestRACE_LocationQueueIsClientSide` سقط في جولتين من أربعٍ** من
// الحزمة الكاملة (دورةُ ٥١): **أربعُ دفعاتِ موقعٍ متزامنةٍ عَلِقت
// ثلاثين ثانية.**
//
// **ثمّ لم يسقط**: أساسٌ ٠ من ٣ · وشجرةٌ نهائيّةٌ ٠ من ٣ · ومنفرداً
// ×١٠ يمرّ.
//
// # وما يقيسه هذا الملفّ
//
// **تكرارٌ محكومٌ يفرّق ثلاثةً**:
//
//	السائقُ نفسُه       ←  تزاحمٌ على صفٍّ واحد؟
//	وسائقون مختلفون    ←  مَوردٌ مشترَك؟
//	وشاهدٌ لا يمرّ بالمسار ←  أهو المنتَجُ أم المَسبَحُ والقاعدة؟
//
// **ولا يُقال «جمود» إلّا بدليل**: **الرقمُ وحدَه لا يميّز انتظارَ
// قفلٍ من جفاف مَسبَحٍ من بطء بيئة.**
package qa

import (
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"
)

// locBattery نتيجةُ بطّاريّةٍ واحدة.
type locBattery struct {
	Runs, Timeouts, Fails int
	Max                   time.Duration
	Lat                   []time.Duration
}

func (b *locBattery) add(d time.Duration, code int, timeout bool) {
	b.Runs++
	b.Lat = append(b.Lat, d)
	if d > b.Max {
		b.Max = d
	}
	if timeout {
		b.Timeouts++
	}
	if code >= 500 {
		b.Fails++
	}
}

func (b *locBattery) report(t *testing.T, name string) {
	t.Helper()
	sort.Slice(b.Lat, func(i, j int) bool { return b.Lat[i] < b.Lat[j] })
	pick := func(p float64) time.Duration {
		if len(b.Lat) == 0 {
			return 0
		}
		i := int(float64(len(b.Lat)-1) * p)
		return b.Lat[i].Round(time.Millisecond)
	}
	t.Logf("  %-30s نداءاتٌ=%-4d مهلةٌ=%-3d عطبٌ=%-3d · "+
		"p50=%v p95=%v p99=%v أقصى=%v",
		name, b.Runs, b.Timeouts, b.Fails,
		pick(0.50), pick(0.95), pick(0.99), b.Max.Round(time.Millisecond))
	if b.Timeouts > 0 || b.Fails > 0 {
		t.Errorf("**%s: %d مهلةً و%d عطباً من %d** — "+
			"**وهذا تكرارٌ حاضرٌ لا شاهدٌ تاريخيّ.**",
			name, b.Timeouts, b.Fails, b.Runs)
	}
}

// pushLocation دفعةُ موقعٍ واحدةٌ بالمسار الحقيقيّ.
func pushLocation(h *Harness, drv *User, i int) (time.Duration, int) {
	start := time.Now()
	res := h.POST("/api/v1/driver/location", drv.Token, map[string]any{
		"lat": 35.9506 + float64(i%50)/10000, "lng": 39.0094,
	})
	return time.Since(start), res.Code
}

// concurrentPush **يبدأ الجميعُ معاً ببوّابةٍ واحدة** — وإلّا لم يتزاحموا.
func concurrentPush(h *Harness, drivers []*User, round int, budget time.Duration) []struct {
	D    time.Duration
	Code int
	TO   bool
} {
	out := make([]struct {
		D    time.Duration
		Code int
		TO   bool
	}, len(drivers))

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range drivers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			done := make(chan struct{})
			var d time.Duration
			var code int
			go func() {
				d, code = pushLocation(h, drivers[i], round)
				close(done)
			}()
			select {
			case <-done:
				out[i].D, out[i].Code = d, code
			case <-time.After(budget):
				out[i].TO = true
				out[i].D = budget
			}
		}(i)
	}
	close(start)
	wg.Wait()
	return out
}

// ══════════════════════════════════════════════════════════════════════
// **أ · دفعةٌ واحدة ×100**
// ══════════════════════════════════════════════════════════════════════

func TestDIAG_LocationSingle(t *testing.T) {
	if testing.Short() {
		t.Skip("تشخيصٌ — لا يُشغَّل في الوضع القصير")
	}
	h := New(t)
	drv := h.Factory().Driver(OnShift())

	var b locBattery
	for i := 0; i < 100; i++ {
		d, code := pushLocation(h, drv, i)
		b.add(d, code, d > 10*time.Second)
	}
	b.report(t, "دفعةٌ واحدة ×100")
}

// ══════════════════════════════════════════════════════════════════════
// **ب و د · أربعٌ متزامنةٌ للسائق نفسِه ×200**
// ══════════════════════════════════════════════════════════════════════
//
// **وهي صورةُ ما سقط**: تطبيقُ السائق يُفرِغ طابورَه بعد انقطاعِ شبكة.

func TestDIAG_LocationSameDriverConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("تشخيصٌ — لا يُشغَّل في الوضع القصير")
	}
	h := New(t)
	drv := h.Factory().Driver(OnShift())
	four := []*User{drv, drv, drv, drv}

	var b locBattery
	for r := 0; r < 200; r++ {
		for _, o := range concurrentPush(h, four, r, 10*time.Second) {
			b.add(o.D, o.Code, o.TO)
		}
		if b.Timeouts > 0 {
			t.Logf("  **أوّلُ مهلةٍ في الجولة %d** — %s", r+1, poolLine(h))
			break
		}
	}
	b.report(t, "السائقُ نفسُه ×4 ×200")
	t.Logf("  %s", poolLine(h))
}

// ══════════════════════════════════════════════════════════════════════
// **ج · وثمانٍ متزامنة ×100**
// ══════════════════════════════════════════════════════════════════════

func TestDIAG_LocationEightConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("تشخيصٌ — لا يُشغَّل في الوضع القصير")
	}
	h := New(t)
	drv := h.Factory().Driver(OnShift())
	eight := make([]*User, 8)
	for i := range eight {
		eight[i] = drv
	}

	var b locBattery
	for r := 0; r < 100; r++ {
		for _, o := range concurrentPush(h, eight, r, 10*time.Second) {
			b.add(o.D, o.Code, o.TO)
		}
		if b.Timeouts > 0 {
			t.Logf("  **أوّلُ مهلةٍ في الجولة %d** — %s", r+1, poolLine(h))
			break
		}
	}
	b.report(t, "السائقُ نفسُه ×8 ×100")
	t.Logf("  %s", poolLine(h))
}

// ══════════════════════════════════════════════════════════════════════
// **هـ · وسائقون مختلفون ×200**
// ══════════════════════════════════════════════════════════════════════
//
// **وهو ما يفرّق تزاحمَ صفٍّ واحدٍ من مَوردٍ مشترَك.**

func TestDIAG_LocationDifferentDriversConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("تشخيصٌ — لا يُشغَّل في الوضع القصير")
	}
	h := New(t)
	f := h.Factory()
	four := []*User{f.Driver(OnShift()), f.Driver(OnShift()),
		f.Driver(OnShift()), f.Driver(OnShift())}

	var b locBattery
	for r := 0; r < 200; r++ {
		for _, o := range concurrentPush(h, four, r, 10*time.Second) {
			b.add(o.D, o.Code, o.TO)
		}
		if b.Timeouts > 0 {
			t.Logf("  **أوّلُ مهلةٍ في الجولة %d** — %s", r+1, poolLine(h))
			break
		}
	}
	b.report(t, "سائقون مختلفون ×4 ×200")
	t.Logf("  %s", poolLine(h))
}

// ══════════════════════════════════════════════════════════════════════
// **و · شاهدٌ لا يمرّ بمسار المنتَج**
// ══════════════════════════════════════════════════════════════════════
//
// **وبلا هذا لا يُفرَّق عيبُ المنتَج من بطء القاعدة أو المَسبَح** —
// **والنسبتان تُقارَنان لا تُفسَّر إحداهما وحدَها.**

func TestDIAG_LocationControlNonProductPath(t *testing.T) {
	if testing.Short() {
		t.Skip("تشخيصٌ — لا يُشغَّل في الوضع القصير")
	}
	h := New(t)
	drv := h.Factory().Driver(OnShift())

	var b locBattery
	for r := 0; r < 200; r++ {
		start := make(chan struct{})
		var wg sync.WaitGroup
		res := make([]time.Duration, 4)
		errs := make([]error, 4)
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				t0 := time.Now()
				// **ثلاثُ كتاباتٍ بحجمٍ مقارِبٍ على الجدول نفسِه** —
				// **بلا مسار HTTP ولا وسائطَ ولا معالِج.**
				_, e1 := h.Pool.Exec(ctxBG(),
					`UPDATE users SET last_location_at = now() WHERE id = $1::uuid`, drv.ID)
				_, e2 := h.Pool.Exec(ctxBG(), `
					INSERT INTO driver_track (driver_id, at, recorded_at)
					VALUES ($1::uuid, ST_SetSRID(ST_MakePoint(39.0,35.9),4326)::geography, now())
					ON CONFLICT (driver_id, recorded_at) DO NOTHING`, drv.ID)
				_, e3 := h.Pool.Exec(ctxBG(), `
					DELETE FROM driver_track
					WHERE driver_id = $1::uuid AND recorded_at < now() - interval '2 hours'`,
					drv.ID)
				res[i] = time.Since(t0)
				for _, e := range []error{e1, e2, e3} {
					if e != nil {
						errs[i] = e
					}
				}
			}(i)
		}
		close(start)
		wg.Wait()
		for i, d := range res {
			code := 200
			if errs[i] != nil {
				code = 500
			}
			b.add(d, code, d > 10*time.Second)
		}
		if b.Timeouts > 0 || b.Fails > 0 {
			t.Logf("  **أوّلُ تعثّرٍ في الجولة %d** — %v · %s", r+1, errs, poolLine(h))
			break
		}
	}
	b.report(t, "شاهدٌ خارجَ المسار ×4 ×200")
	t.Logf("  %s", poolLine(h))
}

// poolLine حالُ المَسبَح في سطر.
func poolLine(h *Harness) string {
	st := h.Pool.Stat()
	return fmt.Sprintf("المَسبَح: محجوزٌ=%d خاملٌ=%d سقفٌ=%d انتظارٌ=%s",
		st.AcquiredConns(), st.IdleConns(), st.MaxConns(),
		st.AcquireDuration().Round(time.Millisecond))
}
