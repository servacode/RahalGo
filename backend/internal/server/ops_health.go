package server

import (
	"context"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/servacode/rahalgo/backend/internal/envguard"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/obs"
)

// ══════════════════════════════════════════════════════════════════════
// **صحّةُ المنصّة الداخليّة — بابٌ محميٌّ يُقرأ ولا يُكتب** (دورة ٧٠أ)
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا لا يُوسَّع `/healthz`
//
// **العامُّ يُسأل من كلّ أحدٍ بلا اسم** — **ومسبحُ اتّصالاتٍ ومعاملاتٌ
// عالقةٌ وعددُ مقابسَ متّصلةٍ خريطةُ حملٍ تُقرأ من خارج.** **ومن عرف
// متى يكون المسبحُ ممتلئاً عرف متى يدفع.**
//
// **و`/healthz` بابُ نجاةٍ مفتوحٌ في بوّابة التحديث** (`min_version.go`)
// — **فما يُضاف إليه يصير عامّاً فعلاً لا نظراً.**
//
// # وتجميعٌ لا تجسّس
//
// **ولا نصَّ `SQL` ولا مُعاملاتِه ولا معرّفَ إنسانٍ ولا هاتفاً ولا
// مبلغاً.** **صحّةُ تشغيلٍ لا تجسّسٌ على قاعدة.**
//
// # والحكمُ لم يعد «القاعدةُ أجابت؟» وحدَه
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤ — «مراقبة التشغيل»، البندان ١ و٣.)
//
// **كان «سليم» ما دامت القاعدةُ والذاكرةُ تُجيبان** — وعلى التجهيز ٦٨ انتظاراً
// على مسبحٍ فارغ والحبّةُ خضراء. **وصار لكلّ جزءٍ كلمةٌ** (سليم · بطيء · واقف)
// **وسببٌ قصير**، والحكمُ من نافذةِ الدقائق العشر الأخيرة لا من عدّادٍ منذ
// الإقلاع (`opsTracker`).

// opsPoolStat **مسبحُ اتّصالات القاعدة** — أرقامُ `pgxpool` كما هي.
type opsPoolStat struct {
	// Max **السقفُ المُهيَّأ**، وAcquired **ما هو مُعارٌ الآن**.
	Max      int32 `json:"max"`
	Acquired int32 `json:"acquired"`
	Idle     int32 `json:"idle"`
	Total    int32 `json:"total"`
	// Constructing **قيدُ الفتح** — وارتفاعُه المستمرُّ ضغطٌ.
	Constructing int32 `json:"constructing"`
	// EmptyAcquireCount **كم مرّةً انتظر طالبٌ مسبحاً فارغاً** منذ
	// الإقلاع — **وهي أوّلُ علامةِ اختناق.**
	EmptyAcquireCount int64 `json:"empty_acquire_count"`
	// CanceledAcquireCount **وكم طالبٍ يئس** قبل أن ينال اتّصالاً.
	CanceledAcquireCount int64 `json:"canceled_acquire_count"`
	// AcquireDurationMsTotal **مجموعُ انتظار الطالبين** بالميلي ثانية.
	AcquireDurationMsTotal int64 `json:"acquire_duration_ms_total"`
}

// opsDBActivity **ما تقوله القاعدةُ عن نفسها** — تجميعاً بلا نصوص.
//
// **والمجهولُ `null` لا صفر** (المشكلة ٦): صلاحيّةٌ أضيقُ على
// `pg_stat_activity` كانت تُرسم «معلّقةٌ داخل معاملة: ٠» وهي غيرُ معروفة.
type opsDBActivity struct {
	// Reachable **أجابت؟** — والباقي لا معنى له إن لم تُجب.
	Reachable bool `json:"reachable"`
	// PingMs **زمنُ ردِّها.**
	PingMs *int64 `json:"ping_ms"`
	// Backends **جلساتُ هذه القاعدة كلُّها** (لا اتّصالاتُنا وحدَنا).
	Backends *int `json:"backends"`
	// Active **تنفّذ الآن**، وIdleInTransaction **فتحت معاملةً ونامت**.
	Active            *int `json:"active"`
	IdleInTransaction *int `json:"idle_in_transaction"`
	// LongestTxSeconds **عمرُ أقدم معاملةٍ مفتوحة** — وصفرٌ لا معاملة.
	LongestTxSeconds *int64 `json:"longest_tx_seconds"`
	// WaitingLocks **كم جلسةً تنتظر قفلاً** الآن.
	WaitingLocks *int `json:"waiting_locks"`
}

// opsRedis **حالُ الذاكرة** — بلا مفتاحٍ ولا قيمة.
type opsRedis struct {
	Reachable bool   `json:"reachable"`
	PingMs    *int64 `json:"ping_ms"`
	// PoolHits/Misses/Timeouts **من عدّاد المكتبة نفسِها.**
	PoolHits     uint32 `json:"pool_hits"`
	PoolMisses   uint32 `json:"pool_misses"`
	PoolTimeouts uint32 `json:"pool_timeouts"`
	TotalConns   uint32 `json:"total_conns"`
	IdleConns    uint32 `json:"idle_conns"`
}

// opsRuntime **حالُ العمليّة** — بلا أثرِ نداءٍ ولا مسارِ ملفّ.
type opsRuntime struct {
	UptimeSeconds int64  `json:"uptime_seconds"`
	Goroutines    int    `json:"goroutines"`
	HeapMB        uint64 `json:"heap_mb"`
	SysMB         uint64 `json:"sys_mb"`
	GCCount       uint32 `json:"gc_count"`
	SourceCommit  string `json:"source_commit"`
	BuildID       string `json:"build_id"`
}

// opsWS **البثّ** — أعدادٌ لا هويّات.
type opsWS struct {
	// Subscriptions **اشتراكاتٌ حيّةٌ في الموزّع** — وليست عددَ المقابس.
	Subscriptions int `json:"subscriptions"`
	// Auth **حصيلةُ المصافحات بسببها** منذ الإقلاع.
	Auth map[string]int64 `json:"auth"`
}

// opsPart **حالُ جزءٍ بكلمة** — `ok` سليم · `slow` بطيء · `down` واقف ·
// `unknown` غيرُ معروف. **والسببُ مفتاحٌ من معجمٍ مغلق** لا نصُّ خطأ.
type opsPart struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

// opsVerdict **الحكمُ** — `ok` · `degraded` · `down` (القاعدةُ أو الذاكرةُ واقفة).
type opsVerdict struct {
	State string             `json:"state"`
	Parts map[string]opsPart `json:"parts"`
}

// أجزاءُ الحكم بترتيب العرض — الخادمُ · القاعدةُ · الذاكرةُ · الإشعاراتُ · الاتّصالُ اللحظيّ.
const (
	partAPI      = "api"
	partDB       = "db"
	partCache    = "cache"
	partPush     = "push"
	partRealtime = "realtime"
)

var opsPartsOrder = []string{partAPI, partDB, partCache, partPush, partRealtime}

// كلماتُ الحال.
const (
	stOK      = "ok"
	stSlow    = "slow"
	stDown    = "down"
	stUnknown = "unknown"
)

// أسبابٌ — معجمٌ مغلقٌ تترجمه الشاشة.
const (
	whyNoAnswer   = "no_answer"
	whySlowAnswer = "slow_answer"
	whyPoolWaits  = "pool_waits"
	whyLongTx     = "long_tx"
	whyLockWaits  = "lock_waits"
	whyPushFail   = "push_failing"
	whyPushSome   = "push_some_failed"
	whyWSBackend  = "ws_backend"
	whyBusy       = "too_busy"
	whyNotWired   = "not_wired"
)

// عتباتُ الحكم — **أرقامٌ مسمّاةٌ هنا لا مبعثرةٌ في الشروط.**
const (
	// opsWindow **نافذةُ الحكم** — ما وقع في الدقائق العشر الأخيرة.
	opsWindow          = 10 * time.Minute
	dbSlowMs           = 500
	cacheSlowMs        = 200
	longTxSlowSec      = 300
	lockWaitsSlow      = 5
	poolEmptySlow      = 20
	pushFailMinAttempt = 3
	goroutinesBusy     = 20000
)

type opsHealth struct {
	// Status **`ok` أو `degraded`** — عقدُ الدورة ٧٠أ يقرؤه، **وكلُّ ما ليس
	// سليماً `degraded`.** والتفصيلُ في `verdict`.
	Status  string           `json:"status"`
	Verdict opsVerdict       `json:"verdict"`
	Pool    opsPoolStat      `json:"pg_pool"`
	DB      opsDBActivity    `json:"pg"`
	Redis   opsRedis         `json:"redis"`
	Runtime opsRuntime       `json:"runtime"`
	WS      opsWS            `json:"ws"`
	Push    map[string]int64 `json:"push"`
	// Clients **عددُ النداءات من كلّ «تطبيق:نسخة» منذ آخر إقلاع** — لا
	// اتّصالاتٌ حيّة (المشكلة ٧).
	Clients map[string]int64 `json:"clients"`
	// ClientsDropped **نسخٌ لم تُعدّ لأنّ القائمةَ امتلأت** (سقفُها ٦٤) — لا
	// «نداءاتٌ مرفوضة» (المشكلة ٧).
	ClientsDropped int64 `json:"clients_dropped"`
	// WindowMinutes **نافذةُ الحكم بالدقائق** — والعدّاداتُ نفسُها منذ الإقلاع.
	WindowMinutes int `json:"window_minutes"`
}

// ══════════════════════════════════════════════════════════════════════
// **نافذةُ الحكم — ما تغيّر في الدقائق الأخيرة لا منذ الإقلاع**
// ══════════════════════════════════════════════════════════════════════

// opsCounters **عدّاداتٌ تراكميّةٌ يُحكَم على فرقها.**
type opsCounters struct {
	pushAttempted, pushFailed int64
	wsBackend                 int64
	poolEmpty, poolCanceled   int64
	cacheTimeouts             int64
}

func (a opsCounters) minus(b opsCounters) opsCounters {
	d := func(x, y int64) int64 {
		if x < y {
			return 0 // صُفّر العدّاد — لا فرقَ سالب
		}
		return x - y
	}
	return opsCounters{
		pushAttempted: d(a.pushAttempted, b.pushAttempted),
		pushFailed:    d(a.pushFailed, b.pushFailed),
		wsBackend:     d(a.wsBackend, b.wsBackend),
		poolEmpty:     d(a.poolEmpty, b.poolEmpty),
		poolCanceled:  d(a.poolCanceled, b.poolCanceled),
		cacheTimeouts: d(a.cacheTimeouts, b.cacheTimeouts),
	}
}

type opsSample struct {
	at time.Time
	c  opsCounters
}

// opsTracker **ذاكرةُ المراقبة** — عيّناتُ النافذة وحالُ الانقطاع وآخرُ حكم.
//
// **في الذاكرة لا في القاعدة**: القاعدةُ هي ما قد يقع، **ومن كتب حالَ
// انقطاعها فيها لم يكتب شيئاً.** وإعادةُ التشغيل تبدأ العدَّ من جديد.
type opsTracker struct {
	mu      sync.Mutex
	samples []opsSample
	// آخرُ حكمٍ ووقتُه — يقرؤه الشريطُ بلا كلفة.
	last   *opsVerdict
	lastAt time.Time
	// الانقطاع: متى بدأ، وأأُشعر المالكُ به.
	badSince time.Time
	notified bool
}

// observe **يحفظ العيّنة ويُرجع ما تغيّر في النافذة.**
//
// **والأساسُ أقدمُ عيّنةٍ في النافذة** — ولا عيّنةَ بعد ⇒ منذ الإقلاع.
func (t *opsTracker) observe(now time.Time, c opsCounters) opsCounters {
	t.mu.Lock()
	defer t.mu.Unlock()
	cut := now.Add(-opsWindow)
	kept := t.samples[:0]
	for _, s := range t.samples {
		if !s.at.Before(cut) {
			kept = append(kept, s)
		}
	}
	t.samples = kept
	base := opsCounters{}
	if len(t.samples) > 0 {
		base = t.samples[0].c
	}
	t.samples = append(t.samples, opsSample{at: now, c: c})
	return c.minus(base)
}

func (t *opsTracker) remember(v opsVerdict, at time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.last, t.lastAt = &v, at
}

func (t *opsTracker) recent(maxAge time.Duration, now time.Time) (opsVerdict, time.Time, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.last == nil || now.Sub(t.lastAt) > maxAge {
		return opsVerdict{}, time.Time{}, false
	}
	return *t.last, t.badSince, true
}

// ══════════════════════════════════════════════════════════════════════
// **الحكم — دالّةٌ محضةٌ تُفحَص بلا قاعدة**
// ══════════════════════════════════════════════════════════════════════

// opsJudgeInput **ما يُحكَم به** — المقيسُ الآن وفرقُ النافذة.
type opsJudgeInput struct {
	DBUp         bool
	DBPingMs     int64
	LongestTxSec *int64
	WaitingLocks *int
	PoolMax      int32
	PoolAcquired int32
	CacheWired   bool
	CacheUp      bool
	CachePingMs  int64
	RealtimeOn   bool
	Goroutines   int
	// Delta **ما تغيّر في النافذة.**
	Delta opsCounters
}

// judgeOps **الحكمُ بكلمةٍ لكلّ جزء** (المشكلة ٤).
//
//	القاعدة     لا تُجيب ⇒ واقف · ردٌّ بطيء أو معاملةٌ طويلة أو أقفالٌ منتظرة
//	            أو انتظارٌ على مسبحٍ فارغ ⇒ بطيء
//	الذاكرة     لا تُجيب ⇒ واقف · ردٌّ بطيء أو مهلُ مسبح ⇒ بطيء
//	الإشعارات   نصفُ المحاولات فشلت (٣ فأكثر) ⇒ واقف · فشلٌ أقلّ ⇒ بطيء
//	الاتّصال    تعذّر سؤالُ الجلسة عند المصافحة ⇒ بطيء
//	الخادم      يُجيب ⇒ سليم · خيوطٌ فوق الحدّ ⇒ بطيء
func judgeOps(in opsJudgeInput) opsVerdict {
	p := map[string]opsPart{}

	switch {
	case in.Goroutines > goroutinesBusy:
		p[partAPI] = opsPart{State: stSlow, Reason: whyBusy}
	default:
		p[partAPI] = opsPart{State: stOK}
	}

	switch {
	case !in.DBUp:
		p[partDB] = opsPart{State: stDown, Reason: whyNoAnswer}
	case in.DBPingMs >= dbSlowMs:
		p[partDB] = opsPart{State: stSlow, Reason: whySlowAnswer}
	case in.Delta.poolCanceled > 0 || in.Delta.poolEmpty >= poolEmptySlow ||
		(in.PoolMax > 0 && in.PoolAcquired >= in.PoolMax):
		p[partDB] = opsPart{State: stSlow, Reason: whyPoolWaits}
	case in.LongestTxSec != nil && *in.LongestTxSec >= longTxSlowSec:
		p[partDB] = opsPart{State: stSlow, Reason: whyLongTx}
	case in.WaitingLocks != nil && *in.WaitingLocks >= lockWaitsSlow:
		p[partDB] = opsPart{State: stSlow, Reason: whyLockWaits}
	default:
		p[partDB] = opsPart{State: stOK}
	}

	switch {
	case !in.CacheWired:
		p[partCache] = opsPart{State: stUnknown, Reason: whyNotWired}
	case !in.CacheUp:
		p[partCache] = opsPart{State: stDown, Reason: whyNoAnswer}
	case in.CachePingMs >= cacheSlowMs:
		p[partCache] = opsPart{State: stSlow, Reason: whySlowAnswer}
	case in.Delta.cacheTimeouts > 0:
		p[partCache] = opsPart{State: stSlow, Reason: whyPoolWaits}
	default:
		p[partCache] = opsPart{State: stOK}
	}

	switch {
	case in.Delta.pushAttempted >= pushFailMinAttempt && in.Delta.pushFailed*2 >= in.Delta.pushAttempted:
		p[partPush] = opsPart{State: stDown, Reason: whyPushFail}
	case in.Delta.pushFailed > 0:
		p[partPush] = opsPart{State: stSlow, Reason: whyPushSome}
	default:
		p[partPush] = opsPart{State: stOK}
	}

	switch {
	case !in.RealtimeOn:
		p[partRealtime] = opsPart{State: stUnknown, Reason: whyNotWired}
	case in.Delta.wsBackend > 0:
		p[partRealtime] = opsPart{State: stSlow, Reason: whyWSBackend}
	default:
		p[partRealtime] = opsPart{State: stOK}
	}

	v := opsVerdict{State: stOK, Parts: p}
	for _, k := range opsPartsOrder {
		switch p[k].State {
		case stDown:
			if k == partDB || k == partCache {
				v.State = stDown
			} else if v.State == stOK {
				v.State = "degraded"
			}
		case stSlow:
			if v.State == stOK {
				v.State = "degraded"
			}
		}
	}
	return v
}

// plainParts **الكلماتُ بلا أسباب** — لمن يرى سيرَ الطلبات ولا يرى التفاصيلَ
// التقنيّة (قرارُ المالك ٢٠٢٦-١٠-٠٤ — «مراقبة التشغيل»، البند ٢).
func (v opsVerdict) plainParts() map[string]string {
	out := make(map[string]string, len(v.Parts))
	for k, p := range v.Parts {
		out[k] = p.State
	}
	return out
}

// ══════════════════════════════════════════════════════════════════════
// **الجمع**
// ══════════════════════════════════════════════════════════════════════

// collectOps **يجمع اللقطةَ ويحكم عليها.**
//
// **ولا يُسقطه عطبُ جزءٍ منه**: قاعدةٌ لا تُجيب هي الخبرُ نفسُه.
// **والذاكرةُ والبثُّ قد يغيبان في الفحص** — فيُقالان «غير معروف» لا «سليم».
func (s *Server) collectOps(ctx context.Context) opsHealth {
	now := time.Now()
	out := opsHealth{WindowMinutes: int(opsWindow / time.Minute)}
	in := opsJudgeInput{}

	// ── مسبحُ الاتّصالات ────────────────────────────────────────
	st := s.pg.Stat()
	out.Pool = opsPoolStat{
		Max:                    st.MaxConns(),
		Acquired:               st.AcquiredConns(),
		Idle:                   st.IdleConns(),
		Total:                  st.TotalConns(),
		Constructing:           st.ConstructingConns(),
		EmptyAcquireCount:      st.EmptyAcquireCount(),
		CanceledAcquireCount:   st.CanceledAcquireCount(),
		AcquireDurationMsTotal: st.AcquireDuration().Milliseconds(),
	}
	in.PoolMax, in.PoolAcquired = out.Pool.Max, out.Pool.Acquired

	// ── القاعدة ────────────────────────────────────────────────
	pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	t0 := time.Now()
	if err := s.pg.Ping(pctx); err == nil {
		ms := time.Since(t0).Milliseconds()
		out.DB.Reachable, out.DB.PingMs = true, &ms
		in.DBUp, in.DBPingMs = true, ms
		s.readDBActivity(pctx, &out.DB)
	}
	cancel()
	in.LongestTxSec, in.WaitingLocks = out.DB.LongestTxSeconds, out.DB.WaitingLocks

	// ── الذاكرة ────────────────────────────────────────────────
	var cacheTimeouts int64
	if s.rdb != nil {
		in.CacheWired = true
		rctx, rcancel := context.WithTimeout(ctx, 3*time.Second)
		t0 = time.Now()
		if err := s.rdb.Ping(rctx).Err(); err == nil {
			ms := time.Since(t0).Milliseconds()
			out.Redis.Reachable, out.Redis.PingMs = true, &ms
			in.CacheUp, in.CachePingMs = true, ms
		}
		rcancel()
		rs := s.rdb.PoolStats()
		out.Redis.PoolHits, out.Redis.PoolMisses = rs.Hits, rs.Misses
		out.Redis.PoolTimeouts = rs.Timeouts
		out.Redis.TotalConns, out.Redis.IdleConns = rs.TotalConns, rs.IdleConns
		cacheTimeouts = int64(rs.Timeouts)
	}

	// ── العمليّة ───────────────────────────────────────────────
	snap := obs.Take()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	commit, build := envguard.BuildInfo()
	out.Runtime = opsRuntime{
		UptimeSeconds: snap.UptimeSeconds,
		Goroutines:    runtime.NumGoroutine(),
		HeapMB:        ms.HeapAlloc / (1 << 20),
		SysMB:         ms.Sys / (1 << 20),
		GCCount:       ms.NumGC,
		SourceCommit:  commit,
		BuildID:       build,
	}
	in.Goroutines = out.Runtime.Goroutines

	// ── البثّ والدفع ───────────────────────────────────────────
	if s.hub != nil {
		in.RealtimeOn = true
		out.WS.Subscriptions = s.hub.Count()
	}
	out.WS.Auth = snap.WSAuth
	out.Push = snap.Push
	out.Clients, out.ClientsDropped = snap.Clients, snap.ClientsDropped

	in.Delta = s.opsT.observe(now, opsCounters{
		pushAttempted: snap.Push[string(obs.PushAttempted)],
		pushFailed:    snap.Push[string(obs.PushFailed)],
		wsBackend:     snap.WSAuth[string(obs.WSBackendUnavailable)],
		poolEmpty:     out.Pool.EmptyAcquireCount,
		poolCanceled:  out.Pool.CanceledAcquireCount,
		cacheTimeouts: cacheTimeouts,
	})
	out.Verdict = judgeOps(in)
	out.Status = "ok"
	if out.Verdict.State != stOK {
		out.Status = "degraded"
	}
	s.opsT.remember(out.Verdict, now)
	return out
}

// handleOpsHealth **التفاصيلُ للمهندس** — `observability.read` وحدَها
// (قرارُ المالك ٢٠٢٦-١٠-٠٤ — «مراقبة التشغيل»، البند ٢).
//
// **ويُردّ ٢٠٠ بالجسد كاملاً حتّى والقاعدةُ واقفة** — من ردّ ٥٠٣ عند أوّل
// عطبٍ حجب بقيّةَ الصورة عمّن جاء يشخّص.
func (s *Server) handleOpsHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := timeoutCtx(r, 5*time.Second)
	defer cancel()
	httpx.JSON(w, http.StatusOK, s.collectOps(ctx))
}

// readDBActivity **يسأل `pg_stat_activity` تجميعاً.**
//
// **ولا `query` ولا `application_name` ولا اسمُ مستخدم** — عمودُ `query` يحمل
// نصَّ الجملة ومُعاملاتِها. **فلا يُقرأ أصلاً**: ما لا يُقرأ لا يُسرَّب.
//
// **وعطبُ هذا الاستعلام لا يُسقط الباب — ويترك الحقولَ `null`** (المشكلة ٦):
// كان يتركها أصفاراً تُقرأ «لا معاملةَ معلّقة» وهي غيرُ معروفة.
func (s *Server) readDBActivity(ctx context.Context, out *opsDBActivity) {
	const q = `
		SELECT count(*),
		       count(*) FILTER (WHERE state = 'active'),
		       count(*) FILTER (WHERE state = 'idle in transaction'),
		       COALESCE(EXTRACT(EPOCH FROM (now() - min(xact_start)
		                 FILTER (WHERE xact_start IS NOT NULL)))::bigint, 0),
		       count(*) FILTER (WHERE wait_event_type = 'Lock')
		  FROM pg_stat_activity
		 WHERE datname = current_database()`
	var backends, active, idleTx, locks int
	var longest int64
	if err := s.pg.QueryRow(ctx, q).Scan(&backends, &active, &idleTx, &longest, &locks); err != nil {
		s.logger.Warn("الرصد: تعذّرت قراءةُ نشاط القاعدة", "error", err)
		return
	}
	out.Backends, out.Active, out.IdleInTransaction = &backends, &active, &idleTx
	out.LongestTxSeconds, out.WaitingLocks = &longest, &locks
}
