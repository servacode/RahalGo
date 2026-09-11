package server

import (
	"context"
	"net/http"
	"runtime"
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
// مبلغاً.** **صحّةُ تشغيلٍ لا تجسّسٌ على قاعدة.** **ومن أضاف «أبطأُ
// استعلام» أضاف معه جدولاً وشرطاً وربّما رقمَ هاتفٍ في مُعامل.**
//
// # وحدودُ ما يُقاس تُقال
//
// **والعدّادُ يُصفَّر بإقلاع المحرّك** — **ومعه `uptime` فيُشتقّ
// المعدّل.** **ولا يُدَّعى تاريخٌ لا يملكه.**

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
type opsDBActivity struct {
	// Reachable **أجابت؟** — والباقي لا معنى له إن لم تُجب.
	Reachable bool `json:"reachable"`
	// PingMs **زمنُ ردِّها.**
	PingMs int64 `json:"ping_ms"`
	// Backends **جلساتُ هذه القاعدة كلُّها** (لا اتّصالاتُنا وحدَنا):
	// **جارٌ يشاركنا الخادمَ يظهر هنا**، وهذا مقصود.
	Backends int `json:"backends"`
	// Active **تنفّذ الآن**، وIdleInTransaction **فتحت معاملةً ونامت**
	// — **وهي أخطرُ الأرقام**: تُمسك أقفالاً ولا تتقدّم.
	Active            int `json:"active"`
	IdleInTransaction int `json:"idle_in_transaction"`
	// LongestTxSeconds **عمرُ أقدم معاملةٍ مفتوحة** — `XG-34` يُبحث
	// عنه هنا. **وصفرٌ يعني لا معاملةَ مفتوحة.**
	LongestTxSeconds int64 `json:"longest_tx_seconds"`
	// WaitingLocks **كم جلسةً تنتظر قفلاً** الآن.
	WaitingLocks int `json:"waiting_locks"`
}

// opsRedis **حالُ الذاكرة** — بلا مفتاحٍ ولا قيمة.
type opsRedis struct {
	Reachable bool  `json:"reachable"`
	PingMs    int64 `json:"ping_ms"`
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
	// SourceCommit/BuildID **هويّةُ ما يعمل** — **وهي في
	// `/public/identity` كذلك**، وتُعاد هنا **لأنّ قارئَ التشخيص لا
	// يُطالَب بجمع ردّين ليعرف أيَّ بناءٍ يقرأ.**
	SourceCommit string `json:"source_commit"`
	BuildID      string `json:"build_id"`
}

// opsWS **البثّ** — أعدادٌ لا هويّات.
type opsWS struct {
	// Subscriptions **اشتراكاتٌ حيّةٌ في الموزّع** — **وليست عددَ
	// المقابس**: مقبسُ المعلَّق يشترك مرّتين.
	Subscriptions int `json:"subscriptions"`
	// Auth **حصيلةُ المصافحات بسببها** منذ الإقلاع.
	Auth map[string]int64 `json:"auth"`
}

type opsHealth struct {
	Status  string           `json:"status"`
	Pool    opsPoolStat      `json:"pg_pool"`
	DB      opsDBActivity    `json:"pg"`
	Redis   opsRedis         `json:"redis"`
	Runtime opsRuntime       `json:"runtime"`
	WS      opsWS            `json:"ws"`
	Push    map[string]int64 `json:"push"`
	// Clients **نداءاتٌ بحسب «نوعُ التطبيق:نسخته»** — تجميعاً.
	Clients        map[string]int64 `json:"clients"`
	ClientsDropped int64            `json:"clients_dropped"`
}

// handleOpsHealth **يجمع اللقطةَ ويردّها.**
//
// **ولا يُسقط الباب عطبُ جزءٍ منه**: **قاعدةٌ لا تُجيب هي الخبرُ
// نفسُه** — **ومن ردّ ٥٠٣ عند أوّل عطبٍ حجب بقيّةَ الصورة عمّن جاء
// يشخّص.** **فيُقال «غيرُ سليم» في الحقل، ويُردّ ٢٠٠ بالجسد كاملاً.**
func (s *Server) handleOpsHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := timeoutCtx(r, 5*time.Second)
	defer cancel()

	out := opsHealth{Status: "ok"}

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

	// ── القاعدة ────────────────────────────────────────────────
	t0 := time.Now()
	if err := s.pg.Ping(ctx); err == nil {
		out.DB.Reachable = true
		out.DB.PingMs = time.Since(t0).Milliseconds()
		s.readDBActivity(ctx, &out.DB)
	} else {
		out.Status = "degraded"
	}

	// ── الذاكرة ────────────────────────────────────────────────
	t0 = time.Now()
	if err := s.rdb.Ping(ctx).Err(); err == nil {
		out.Redis.Reachable = true
		out.Redis.PingMs = time.Since(t0).Milliseconds()
	} else {
		out.Status = "degraded"
	}
	rs := s.rdb.PoolStats()
	out.Redis.PoolHits, out.Redis.PoolMisses = rs.Hits, rs.Misses
	out.Redis.PoolTimeouts = rs.Timeouts
	out.Redis.TotalConns, out.Redis.IdleConns = rs.TotalConns, rs.IdleConns

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

	// ── البثّ والدفع ───────────────────────────────────────────
	out.WS = opsWS{Subscriptions: s.hub.Count(), Auth: snap.WSAuth}
	out.Push = snap.Push
	out.Clients, out.ClientsDropped = snap.Clients, snap.ClientsDropped

	httpx.JSON(w, http.StatusOK, out)
}

// readDBActivity **يسأل `pg_stat_activity` تجميعاً.**
//
// **ولا `query` ولا `application_name` ولا اسمُ مستخدم** — **وعمودُ
// `query` يحمل نصَّ الجملة ومُعاملاتِها المُضمَّنة**، **وفيه يمرّ رقمُ
// هاتفٍ ومبلغ.** **فلا يُقرأ أصلاً**: ما لا يُقرأ لا يُسرَّب.
//
// **وعطبُ هذا الاستعلام لا يُسقط الباب** — **قد تُمنَع القراءةُ من
// `pg_stat_activity` بصلاحيّةٍ أضيق**، **وحينها تبقى بقيّةُ الصورة.**
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
	if err := s.pg.QueryRow(ctx, q).Scan(
		&out.Backends, &out.Active, &out.IdleInTransaction,
		&out.LongestTxSeconds, &out.WaitingLocks,
	); err != nil {
		s.logger.Warn("الرصد: تعذّرت قراءةُ نشاط القاعدة", "error", err)
	}
}
