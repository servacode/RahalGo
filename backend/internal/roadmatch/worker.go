package roadmatch

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
)

// ══════════════════════════════════════════════════════════════════════
// **دورة المطابق** (C2) — كل دقيقة، **ونداء تتبّع واحد بالمرّة**
// ══════════════════════════════════════════════════════════════════════
//
//  1. ولا شي إذا الإعداد مطفي، أو القاطع مفتوح، أو جولة تعلّم الطرق شغّالة.
//  2. **الحيّ أوّلاً**: كل سائق إله أثر بعد مؤشّره (`matched_until`) أو إجته نقاط متأخرة (`dirty_from`).
//     القطع الخالصة بتنطابق وبتتخزّن مسكّرة؛ والذيل المفتوح بينطابق مؤقّتاً كل ٥ دقايق أو لما يخلص طلب.
//  3. **الرجوع** (بس إذا الحيّ ما عمل ولا نداء): من `backfill_before` لورا، الأحدث أوّلاً، ≤ ٢٠ نداء.
//     ومعه: القطع بقواعد أقدم (`params`) أو `no_match` أقدم من بناء البلاطات بتنعاد مرّة.
//  4. **الحكم قبل الكتابة**: عاصفة 4xx (٥ ورا بعض، أو ≥ ٥٠٪ من ≥ ١٠ نداءات) ⇒ عطل بالمحرّك، ما
//     بيتخزّن شي والقاطع بيفتح ٥ دقايق. و٣ أعطال ورا بعض (5xx، انقطاع، مهلة) ⇒ نفس الشي.
//
// **والقطع قبل `frozen_before` ما بتنمسّ** (عقد كنّاس العدّاد — `Freeze`).

// pgTick **أصغر فرق وقت بتحفظه القاعدة** (ميكروثانية): حدّ «لحد وشامل» بيتكتب `t + pgTick`. و`t + 1ns`
// بينقصّ لـ`t` وهو رايح للقاعدة ⇒ بتضيع النقطة المشتركة بين الرجوع والحيّ (وقعت بالاختبار).
const pgTick = time.Microsecond

// Options عتبات الدورة.
type Options struct {
	CallsPerTick   int
	BackfillCalls  int
	TailEvery      time.Duration
	LiveLag        time.Duration // آخر نقاط ما بتنقرأ (ممكن لسّا عم توصل)
	FirstSight     time.Duration // أول مرّة منشوف السائق: الحيّ بيبلّش من هالقدّ لورا
	BackfillWindow time.Duration
	BreakerFails   int
	BreakerPause   time.Duration
	Storm4xxRun    int
	StormMinCalls  int
	TilesEvery     time.Duration
	// Drivers لو مش فاضية: هدول السائقين بس (للاختبارات).
	Drivers []string
}

// DefaultOptions القيم المعتمدة.
func DefaultOptions() Options {
	return Options{
		CallsPerTick: 60, BackfillCalls: 20, TailEvery: 5 * time.Minute, LiveLag: 30 * time.Second,
		FirstSight: 30 * time.Minute, BackfillWindow: 3 * time.Hour, BreakerFails: 3, BreakerPause: 5 * time.Minute,
		Storm4xxRun: 5, StormMinCalls: 10, TilesEvery: time.Hour,
	}
}

// Status حالة المطابق — لسطر بصفحة صحّة التشغيل. **مو جزء من الحكم** (ما بيرفع الشريط الأحمر).
type Status struct {
	Enabled bool `json:"enabled"`
	// State: off · ok · paused · waiting (جولة تعلّم الطرق) · no_engine (بلا Valhalla)
	State        string     `json:"state"`
	PausedUntil  *time.Time `json:"paused_until,omitempty"`
	LastError    string     `json:"last_error,omitempty"`
	LastErrorAt  *time.Time `json:"last_error_at,omitempty"`
	LastTick     *time.Time `json:"last_tick,omitempty"`
	CallsHour    int        `json:"calls_hour"`
	NoMatchHour  int        `json:"no_match_hour"`
	NoMatchShare float64    `json:"no_match_share"`
	BacklogDays  float64    `json:"backlog_days"`
}

// TickResult شو صار بالدورة.
type TickResult struct {
	Calls    int
	NoMatch  int
	Rows     int
	Storm    bool
	Breaker  bool
	Backfill bool
}

// Worker المطابق. واحد بالخادم.
type Worker struct {
	Pool    *pgxpool.Pool
	Matcher Matcher
	// Tiles وقت بناء البلاطات (اختياري) — `no_match` أقدم منه بينعاد مرّة.
	Tiles func(ctx context.Context) time.Time
	// Busy جولة تعلّم الطرق شغّالة؟ (`maplearn.RoadsBusy`) ⇒ المطابق بيستنّى.
	Busy   func() bool
	Logger *slog.Logger
	Opt    Options
	Cfg    drivermonitor.Config

	mu          sync.Mutex
	st          Status
	pausedUntil time.Time
	consecFails int
	calls       []callMark
	tilesAt     time.Time
	tilesRead   time.Time
}

type callMark struct {
	at      time.Time
	noMatch bool
}

// NewWorker مطابق بالقيم المعتمدة.
func NewWorker(pool *pgxpool.Pool, m Matcher, logger *slog.Logger) *Worker {
	return &Worker{Pool: pool, Matcher: m, Logger: logger, Opt: DefaultOptions(), Cfg: drivermonitor.DefaultConfig()}
}

// Status لقطة الحالة.
func (w *Worker) Status(now time.Time) Status {
	w.mu.Lock()
	defer w.mu.Unlock()
	s := w.st
	w.trimCalls(now)
	s.CallsHour, s.NoMatchHour = len(w.calls), 0
	for _, c := range w.calls {
		if c.noMatch {
			s.NoMatchHour++
		}
	}
	if s.CallsHour > 0 {
		s.NoMatchShare = float64(s.NoMatchHour) / float64(s.CallsHour)
	}
	if now.Before(w.pausedUntil) {
		t := w.pausedUntil
		s.PausedUntil = &t
		if s.Enabled {
			s.State = "paused"
		}
	}
	return s
}

func (w *Worker) trimCalls(now time.Time) {
	i := 0
	for i < len(w.calls) && now.Sub(w.calls[i].at) > time.Hour {
		i++
	}
	w.calls = w.calls[i:]
}

func (w *Worker) logger() *slog.Logger {
	if w.Logger != nil {
		return w.Logger
	}
	return slog.Default()
}

// errBreaker القاطع فتح بنص الدورة.
var errBreaker = errors.New("roadmatch: القاطع فتح")

// tally عدّادات الدورة.
type tally struct {
	calls, n4xx, consec4xx, budget int
	storm                          bool
}

func (w *Worker) opt() Options {
	o := w.Opt
	d := DefaultOptions()
	if o.CallsPerTick <= 0 {
		o.CallsPerTick = d.CallsPerTick
	}
	if o.BackfillCalls <= 0 {
		o.BackfillCalls = d.BackfillCalls
	}
	if o.TailEvery <= 0 {
		o.TailEvery = d.TailEvery
	}
	if o.LiveLag <= 0 {
		o.LiveLag = d.LiveLag
	}
	if o.FirstSight <= 0 {
		o.FirstSight = d.FirstSight
	}
	if o.BackfillWindow <= 0 {
		o.BackfillWindow = d.BackfillWindow
	}
	if o.BreakerFails <= 0 {
		o.BreakerFails = d.BreakerFails
	}
	if o.BreakerPause <= 0 {
		o.BreakerPause = d.BreakerPause
	}
	if o.Storm4xxRun <= 0 {
		o.Storm4xxRun = d.Storm4xxRun
	}
	if o.StormMinCalls <= 0 {
		o.StormMinCalls = d.StormMinCalls
	}
	if o.TilesEvery <= 0 {
		o.TilesEvery = d.TilesEvery
	}
	return o
}

// Tick **دورة واحدة**. `enabled` = `drivers.road_match_enabled`.
func (w *Worker) Tick(ctx context.Context, now time.Time, enabled bool) (TickResult, error) {
	var res TickResult
	o := w.opt()
	w.mu.Lock()
	w.st.Enabled = enabled
	t := now
	w.st.LastTick = &t
	switch {
	case !enabled:
		w.st.State = "off"
	case w.Matcher == nil:
		w.st.State = "no_engine"
	case now.Before(w.pausedUntil):
		w.st.State = "paused"
	case w.Busy != nil && w.Busy():
		w.st.State = "waiting"
	default:
		w.st.State = "ok"
	}
	state := w.st.State
	w.mu.Unlock()
	if state != "ok" {
		return res, nil
	}
	if w.Cfg.GapAfter == 0 {
		w.Cfg = drivermonitor.DefaultConfig()
	}

	if err := w.firstSight(ctx, now, o); err != nil {
		return res, err
	}

	// ── ١ · الحيّ ──
	tl := &tally{budget: o.CallsPerTick}
	plans, err := w.livePlans(ctx, now, o)
	if err != nil {
		return res, err
	}
	var works []*work
	var runErr error
	for _, p := range plans {
		if tl.calls >= tl.budget || tl.storm {
			break
		}
		wk, err := w.runLive(ctx, p, now, o, tl)
		if wk != nil {
			works = append(works, wk)
		}
		if err != nil {
			runErr = err
			break
		}
	}
	res.Calls, res.NoMatch = tl.calls, tl.n4xx
	if w.judge(tl, now, o, &res) {
		return res, nil
	}
	if runErr != nil && !errors.Is(runErr, ErrDown) && !errors.Is(runErr, errBreaker) {
		return res, runErr
	}
	// اللي انطابق قبل العطل سليم ومرتّب ⇒ بينكتب، والمؤشّر بيوقف عند آخر قطعة انطابقت.
	for _, wk := range works {
		n, err := w.writeLive(ctx, wk, now)
		if err != nil {
			return res, err
		}
		res.Rows += n
	}
	res.Breaker = errors.Is(runErr, errBreaker)
	if tl.calls > 0 || runErr != nil {
		return res, nil
	}

	// ── ٢ · الرجوع (الحيّ ما عمل ولا نداء) ──
	res.Backfill = true
	bt := &tally{budget: o.BackfillCalls}
	plan, err := w.backfill(ctx, now, o, bt)
	res.Calls += bt.calls
	res.NoMatch += bt.n4xx
	if w.judge(bt, now, o, &res) {
		return res, nil
	}
	if err != nil && !errors.Is(err, ErrDown) && !errors.Is(err, errBreaker) {
		return res, err
	}
	n, werr := w.writeBackfill(ctx, plan)
	res.Rows += n
	res.Breaker = res.Breaker || errors.Is(err, errBreaker)
	return res, werr
}

// judge **عاصفة 4xx** (أو مهل) ⇒ عطل بالمحرّك مو بالقطع: القاطع بيفتح وما بيتخزّن شي من الدورة.
func (w *Worker) judge(tl *tally, now time.Time, o Options, res *TickResult) bool {
	storm := tl.storm || tl.consec4xx >= o.Storm4xxRun || (tl.calls >= o.StormMinCalls && tl.n4xx*2 >= tl.calls)
	if storm {
		res.Storm = true
		w.open(now, o, "عاصفة 4xx من محرّك المطابقة — ما انخزّن شي")
		return true
	}
	return false
}

func (w *Worker) open(now time.Time, o Options, why string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pausedUntil = now.Add(o.BreakerPause)
	w.st.LastError = why
	t := now
	w.st.LastErrorAt = &t
	w.consecFails = 0
	w.logger().Warn("مطابقة الطريق: القاطع فتح", "until", w.pausedUntil, "error", why)
}

func (w *Worker) mark(now time.Time, noMatch bool) {
	w.mu.Lock()
	w.calls = append(w.calls, callMark{at: now, noMatch: noMatch})
	w.trimCalls(now)
	w.mu.Unlock()
}

// matchChunk **قطعة ⇒ أسطر**. المهلة بتقسم القطعة لنصّين (لحد ٣٠ نقطة مبعوتة)، والقطعة الصغيرة
// اللي خلصت مهلتها مرّتين بتتخزّن `no_match('timeout')` — نفس النداء البطيء ما بينبعت مرّة تالتة.
func (w *Worker) matchChunk(ctx context.Context, c Chunk, now time.Time, o Options, tl *tally) ([]Row, error) {
	if c.SentCount() < 2 {
		return []Row{FlatRow(c, "ok", "")}, nil
	}
	return w.matchPiece(ctx, c, 0, now, o, tl)
}

func (w *Worker) matchPiece(ctx context.Context, c Chunk, tries int, now time.Time, o Options, tl *tally) ([]Row, error) {
	t0 := time.Now()
	a, err := w.Matcher.Match(ctx, sentPoints(c))
	took := int(time.Since(t0).Milliseconds())
	tl.calls++
	var nm *NoMatchError
	switch {
	case err == nil:
		w.mark(now, false)
		tl.consec4xx = 0
		w.mu.Lock()
		w.consecFails = 0
		w.mu.Unlock()
		rows := Rows(c, a)
		for i := range rows {
			rows[i].TookMs = took
		}
		return rows, nil
	case errors.As(err, &nm):
		w.mark(now, true)
		tl.n4xx++
		tl.consec4xx++
		if tl.consec4xx >= o.Storm4xxRun {
			tl.storm = true
		}
		r := FlatRow(c, "no_match", nm.Code)
		r.TookMs = took
		return []Row{r}, nil
	case errors.Is(err, ErrTimeout):
		// المهلة بتنعدّ متل 4xx بحكم العاصفة: محرّك ما بيردّ أبداً ما لازم يصير `no_match` دايم.
		w.mark(now, true)
		tl.n4xx++
		tl.consec4xx++
		if tl.consec4xx >= o.Storm4xxRun {
			tl.storm = true
			return nil, nil
		}
		if c.SentCount() > MinSplitSent {
			h1, h2 := splitHalves(c)
			r1, err := w.matchPiece(ctx, h1, 0, now, o, tl)
			if err != nil {
				return nil, err
			}
			r2, err := w.matchPiece(ctx, h2, 0, now, o, tl)
			if err != nil {
				return nil, err
			}
			return append(r1, r2...), nil
		}
		if tries == 0 {
			return w.matchPiece(ctx, c, 1, now, o, tl)
		}
		r := FlatRow(c, "no_match", "timeout")
		r.TookMs = took
		return []Row{r}, nil
	case ctx.Err() != nil:
		return nil, ctx.Err()
	default:
		w.mark(now, false)
		if w.fail(now, o, err) {
			return nil, errBreaker
		}
		return nil, err
	}
}

// fail عطل (5xx، انقطاع، مهلة). بيرجّع true إذا فتح القاطع.
func (w *Worker) fail(now time.Time, o Options, err error) bool {
	w.mu.Lock()
	w.consecFails++
	n := w.consecFails
	w.mu.Unlock()
	if n >= o.BreakerFails {
		w.open(now, o, err.Error())
		return true
	}
	return false
}

// splitHalves قطعة لنصّين بالنقاط المبعوتة، بنقطة مشتركة.
func splitHalves(c Chunk) (Chunk, Chunk) {
	half := c.SentCount() / 2
	idx, k := 0, 0
	for i, s := range c.Sent {
		if s {
			if k == half {
				idx = i
				break
			}
			k++
		}
	}
	h1 := Chunk{Points: append([]drivermonitor.Point(nil), c.Points[:idx+1]...),
		Sent: append([]bool(nil), c.Sent[:idx+1]...), Closed: true}
	h2 := Chunk{Points: append([]drivermonitor.Point(nil), c.Points[idx:]...),
		Sent: append([]bool(nil), c.Sent[idx:]...), Closed: c.Closed}
	h2.Sent[0] = true
	return h1, h2
}

// ───────────────────────── الحيّ ─────────────────────────

type progress struct {
	driver         string
	matchedUntil   time.Time
	backfillBefore time.Time
	frozenBefore   *time.Time
	dirtyFrom      *time.Time
	dirtyRev       int
	tailAt         *time.Time
}

type work struct {
	p progress
	// deleteAfter الأسطر اللي بتخلص بعده بتنمحى (نقاط متأخرة).
	deleteAfter *time.Time
	rows        []Row
	newUntil    time.Time
	tail        bool // انطابق الذيل ⇒ tail_at = الآن
	dropOpen    bool
}

// firstSight أول مرّة منشوف السائق: المؤشّرين عند آخر نقطة قبل ٣٠ دقيقة (أو أول نقطة إله).
func (w *Worker) firstSight(ctx context.Context, now time.Time, o Options) error {
	_, err := w.Pool.Exec(ctx, `
		INSERT INTO driver_road_progress (driver_id, matched_until, backfill_before)
		SELECT d.id, c.t, c.t
		  FROM (SELECT DISTINCT ur.user_id AS id FROM user_roles ur
		         WHERE ur.role_code = 'driver' AND (cardinality($2::text[]) = 0 OR ur.user_id::text = ANY($2))) d
		 CROSS JOIN LATERAL (SELECT COALESCE(
		        (SELECT max(recorded_at) FROM driver_trail WHERE driver_id = d.id AND recorded_at <= $1),
		        (SELECT min(recorded_at) FROM driver_trail WHERE driver_id = d.id)) AS t) c
		 WHERE c.t IS NOT NULL
		   AND NOT EXISTS (SELECT 1 FROM driver_road_progress p WHERE p.driver_id = d.id AND p.matched_until IS NOT NULL)
		ON CONFLICT (driver_id) DO UPDATE
		   SET matched_until = GREATEST(EXCLUDED.matched_until, COALESCE(driver_road_progress.frozen_before, EXCLUDED.matched_until)),
		       backfill_before = GREATEST(EXCLUDED.backfill_before, COALESCE(driver_road_progress.frozen_before, EXCLUDED.backfill_before))
		 WHERE driver_road_progress.matched_until IS NULL`, now.Add(-o.FirstSight), o.driversArg())
	return err
}

func (o Options) driversArg() []string {
	if o.Drivers == nil {
		return []string{}
	}
	return o.Drivers
}

// livePlans السائقين اللي إلهم شغل حيّ، الأقدم مؤشّراً أوّلاً.
func (w *Worker) livePlans(ctx context.Context, now time.Time, o Options) ([]progress, error) {
	rows, err := w.Pool.Query(ctx, `
		SELECT p.driver_id::text, p.matched_until, p.backfill_before, p.frozen_before, p.dirty_from, p.dirty_rev, p.tail_at
		  FROM driver_road_progress p
		 WHERE p.matched_until IS NOT NULL
		   AND (cardinality($2::text[]) = 0 OR p.driver_id::text = ANY($2))
		   AND (p.dirty_from IS NOT NULL OR EXISTS (
		        SELECT 1 FROM driver_trail t
		         WHERE t.driver_id = p.driver_id AND t.recorded_at > p.matched_until AND t.recorded_at <= $1))
		 ORDER BY COALESCE(LEAST(p.dirty_from, p.matched_until), p.matched_until)`, now.Add(-o.LiveLag), o.driversArg())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []progress
	for rows.Next() {
		var p progress
		if err := rows.Scan(&p.driver, &p.matchedUntil, &p.backfillBefore, &p.frozenBefore, &p.dirtyFrom,
			&p.dirtyRev, &p.tailAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// runLive شغل سائق حيّ: القطع الخالصة بالترتيب، وبعدها الذيل إذا حان.
func (w *Worker) runLive(ctx context.Context, p progress, now time.Time, o Options, tl *tally) (*work, error) {
	wk := &work{p: p, newUntil: p.matchedUntil}
	start := p.matchedUntil
	if p.dirtyFrom != nil {
		d := *p.dirtyFrom
		if d.Before(p.backfillBefore) {
			d = p.backfillBefore
		}
		if p.frozenBefore != nil && d.Before(*p.frozenBefore) {
			d = *p.frozenBefore
		}
		var last *time.Time
		if err := w.Pool.QueryRow(ctx, `
			SELECT max(to_at) FROM driver_road_chunks
			 WHERE driver_id = $1::uuid AND closed AND to_at < $2 AND to_at >= $3`,
			p.driver, d, p.backfillBefore).Scan(&last); err != nil {
			return nil, err
		}
		start = p.backfillBefore
		if last != nil {
			start = *last
		}
		if start.After(p.matchedUntil) {
			start = p.matchedUntil
		}
		s := start
		wk.deleteAfter = &s
		wk.newUntil = start
		wk.dropOpen = true
	}
	until := now.Add(-o.LiveLag)
	pts, err := drivermonitor.LoadPoints(ctx, w.Pool, p.driver, start, until.Add(pgTick))
	if err != nil {
		return nil, err
	}
	chunks := Chunks(pts, w.Cfg, until)
	var tail *Chunk
	for i := range chunks {
		c := chunks[i]
		if !c.Closed {
			tail = &chunks[i]
			break
		}
		if tl.calls >= tl.budget || tl.storm {
			return wk, nil // الباقي بالدورة الجاية
		}
		if len(c.Points) >= 2 {
			rows, err := w.matchChunk(ctx, c, now, o, tl)
			if err != nil {
				return wk, err
			}
			wk.rows = append(wk.rows, rows...)
			wk.dropOpen = true
		}
		wk.newUntil = c.To()
	}
	if tail == nil || len(tail.Points) < 2 || tl.calls >= tl.budget || tl.storm {
		return wk, nil
	}
	due := wk.dropOpen || p.tailAt == nil || now.Sub(*p.tailAt) >= o.TailEvery
	if !due {
		var ended bool
		if err := w.Pool.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM orders WHERE driver_id = $1::uuid
			                AND COALESCE(delivered_at, returned_at, closed_at) > $2)`, p.driver, *p.tailAt).Scan(&ended); err != nil {
			return nil, err
		}
		due = ended
	}
	if !due {
		return wk, nil
	}
	rows, err := w.matchChunk(ctx, *tail, now, o, tl)
	if err != nil {
		return wk, err
	}
	wk.rows = append(wk.rows, rows...)
	wk.tail = true
	wk.dropOpen = true
	return wk, nil
}

// writeLive يكتب شغل سائق بمعاملة وحدة. **ونقاط متأخرة إجت بنص الدورة ما بتنمحى** (`dirty_rev`).
func (w *Worker) writeLive(ctx context.Context, wk *work, now time.Time) (int, error) {
	if len(wk.rows) == 0 && wk.newUntil.Equal(wk.p.matchedUntil) && wk.deleteAfter == nil {
		return 0, nil
	}
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var cur time.Time
	if err := tx.QueryRow(ctx, `SELECT matched_until FROM driver_road_progress WHERE driver_id = $1::uuid FOR UPDATE`,
		wk.p.driver).Scan(&cur); err != nil {
		return 0, err
	}
	if !cur.Equal(wk.p.matchedUntil) {
		return 0, nil // انكتب من مكان تاني — الدورة الجاية بتعيد
	}
	if wk.deleteAfter != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM driver_road_chunks WHERE driver_id = $1::uuid AND to_at > $2`,
			wk.p.driver, *wk.deleteAfter); err != nil {
			return 0, err
		}
	}
	if wk.dropOpen {
		if _, err := tx.Exec(ctx, `DELETE FROM driver_road_chunks WHERE driver_id = $1::uuid AND NOT closed`,
			wk.p.driver); err != nil {
			return 0, err
		}
	}
	if err := insertRows(ctx, tx, wk.p.driver, wk.rows); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE driver_road_progress
		   SET matched_until = $2,
		       tail_at = CASE WHEN $3 THEN $4::timestamptz WHEN $5 THEN NULL ELSE tail_at END,
		       dirty_from = CASE WHEN dirty_rev = $6 THEN NULL ELSE dirty_from END
		 WHERE driver_id = $1::uuid`,
		wk.p.driver, wk.newUntil, wk.tail, now, wk.dropOpen && !wk.tail, wk.p.dirtyRev); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(wk.rows), nil
}

func insertRows(ctx context.Context, tx pgx.Tx, driver string, rows []Row) error {
	for _, r := range rows {
		pos := make([]float32, len(r.PosM))
		for i, v := range r.PosM {
			pos[i] = float32(v)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO driver_road_chunks (driver_id, from_at, to_at, closed, status, error_code, point_at, pos_m,
			                                line6, road_m, straight_m, params, took_ms)
			VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
			driver, r.FromAt, r.ToAt, r.Closed, r.Status, r.ErrorCode, r.PointAt, pos, r.Line6,
			float32(r.RoadM), float32(r.StraightM), ParamsVersion, r.TookMs); err != nil {
			return err
		}
	}
	return nil
}

// ───────────────────────── الرجوع ─────────────────────────

// bfWrite كتابة رجوع واحدة: نافذة سائق (`bb` → `newBB`) أو إعادة سطر (`rematchID`).
type bfWrite struct {
	driver    string
	bb, newBB time.Time
	rematchID int64
	rows      []Row
}

type bfCand struct {
	driver string
	bb     time.Time
	frozen *time.Time
}

// backfill **من المؤشّر لورا** — الأحدث أوّلاً، وبعده القطع اللي لازم تنعاد. **بيخطّط بس**: الكتابة
// بعد حكم العاصفة (`writeBackfill`).
func (w *Worker) backfill(ctx context.Context, now time.Time, o Options, tl *tally) ([]bfWrite, error) {
	rows, err := w.Pool.Query(ctx, `
		SELECT p.driver_id::text, p.backfill_before, p.frozen_before
		  FROM driver_road_progress p
		 WHERE p.backfill_before IS NOT NULL
		   AND (cardinality($1::text[]) = 0 OR p.driver_id::text = ANY($1))
		   AND EXISTS (SELECT 1 FROM driver_trail t
		                WHERE t.driver_id = p.driver_id AND t.recorded_at < p.backfill_before
		                  AND t.recorded_at >= COALESCE(p.frozen_before, '-infinity'::timestamptz))`, o.driversArg())
	if err != nil {
		return nil, err
	}
	var cands []bfCand
	for rows.Next() {
		var c bfCand
		if err := rows.Scan(&c.driver, &c.bb, &c.frozen); err != nil {
			rows.Close()
			return nil, err
		}
		cands = append(cands, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var plan []bfWrite
	for step := 0; step < 50 && len(cands) > 0 && tl.calls < tl.budget && !tl.storm; step++ {
		sort.Slice(cands, func(i, j int) bool { return cands[i].bb.After(cands[j].bb) })
		c := &cands[0]
		wr, done, err := w.backfillStep(ctx, c.driver, c.bb, c.frozen, now, o, tl)
		if err != nil {
			return plan, err
		}
		if tl.storm {
			return plan, nil
		}
		plan = append(plan, wr)
		c.bb = wr.newBB
		if done {
			cands = cands[1:]
		}
	}
	if tl.calls >= tl.budget || tl.storm {
		return plan, nil
	}
	more, err := w.rematch(ctx, now, o, tl)
	return append(plan, more...), err
}

// backfillStep نافذة وحدة لسائق: [bb − W, bb]. أول قطعة ممكن تكون مقصوصة ببداية النافذة ⇒ بتتأجّل للنافذة
// الجاية (إلا إذا قبلها صمت). `done` = ما ضل شي قبلها.
func (w *Worker) backfillStep(ctx context.Context, driver string, bb time.Time, frozen *time.Time, now time.Time,
	o Options, tl *tally) (bfWrite, bool, error) {
	wr := bfWrite{driver: driver, bb: bb, newBB: bb}
	win := o.BackfillWindow
	var chunks []Chunk
	var pts []drivermonitor.Point
	natural := false
	var prevAt *time.Time
	for attempt := 0; attempt < 4; attempt++ {
		ws := bb.Add(-win)
		if frozen != nil && ws.Before(*frozen) {
			ws = *frozen
		}
		var err error
		pts, err = drivermonitor.LoadPoints(ctx, w.Pool, driver, ws, bb.Add(pgTick))
		if err != nil {
			return wr, false, err
		}
		prevAt = nil
		if err := w.Pool.QueryRow(ctx, `
			SELECT max(recorded_at) FROM driver_trail
			 WHERE driver_id = $1::uuid AND recorded_at < $2 AND recorded_at >= COALESCE($3::timestamptz, '-infinity'::timestamptz)`,
			driver, ws, frozen).Scan(&prevAt); err != nil {
			return wr, false, err
		}
		chunks = Chunks(pts, w.Cfg, bb.Add(365*24*time.Hour))
		natural = prevAt == nil || (len(pts) > 0 && pts[0].At.Sub(*prevAt) > w.Cfg.GapAfter)
		if natural || len(chunks) >= 2 {
			break
		}
		win *= 2
	}
	keep := chunks
	switch {
	case len(pts) == 0:
		if prevAt != nil {
			wr.newBB = *prevAt
		}
	case natural || len(chunks) < 2:
		wr.newBB = pts[0].At
	default:
		keep = chunks[1:]
		wr.newBB = chunks[1].From()
	}
	for _, c := range keep {
		c.Closed = true
		if len(c.Points) < 2 {
			continue
		}
		r, err := w.matchChunk(ctx, c, now, o, tl)
		if err != nil {
			return wr, false, err
		}
		wr.rows = append(wr.rows, r...)
	}
	done := prevAt == nil && (len(pts) == 0 || !wr.newBB.After(pts[0].At))
	if !done && wr.newBB.Equal(bb) {
		wr.newBB = bb.Add(-pgTick) // ما في تقدّم ⇒ خطوة لورا
	}
	return wr, done, nil
}

// rematch **قطع لازم تنعاد مرّة**: بقواعد أقدم من `ParamsVersion`، أو `no_match` أقدم من بناء البلاطات.
// بتنعاد بمدى السطر نفسه، والقديم بينمحى بنفس المعاملة.
func (w *Worker) rematch(ctx context.Context, now time.Time, o Options, tl *tally) ([]bfWrite, error) {
	tiles := w.tiles(ctx, now, o)
	var tilesArg *time.Time
	if !tiles.IsZero() {
		tilesArg = &tiles
	}
	rows, err := w.Pool.Query(ctx, `
		SELECT c.id, c.driver_id::text, c.from_at, c.to_at
		  FROM driver_road_chunks c
		 WHERE c.closed
		   AND (cardinality($3::text[]) = 0 OR c.driver_id::text = ANY($3))
		   AND (c.params < $1 OR (c.status = 'no_match' AND $2::timestamptz IS NOT NULL AND c.matched_at < $2))
		   AND c.from_at >= COALESCE((SELECT p.frozen_before FROM driver_road_progress p WHERE p.driver_id = c.driver_id),
		                             '-infinity'::timestamptz)
		 ORDER BY c.to_at DESC LIMIT $4`, ParamsVersion, tilesArg, o.driversArg(), o.BackfillCalls)
	if err != nil {
		return nil, err
	}
	type span struct{ from, to time.Time }
	var list []bfWrite
	var spans []span
	for rows.Next() {
		var wr bfWrite
		var s span
		if err := rows.Scan(&wr.rematchID, &wr.driver, &s.from, &s.to); err != nil {
			rows.Close()
			return nil, err
		}
		list = append(list, wr)
		spans = append(spans, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var plan []bfWrite
	for i := range list {
		if tl.calls >= tl.budget || tl.storm {
			break
		}
		wr, s := list[i], spans[i]
		pts, err := drivermonitor.LoadPoints(ctx, w.Pool, wr.driver, s.from, s.to.Add(pgTick))
		if err != nil {
			return plan, err
		}
		for _, c := range Chunks(pts, w.Cfg, s.to.Add(365*24*time.Hour)) {
			c.Closed = true
			if len(c.Points) < 2 {
				continue
			}
			r, err := w.matchChunk(ctx, c, now, o, tl)
			if err != nil {
				return plan, err
			}
			wr.rows = append(wr.rows, r...)
		}
		if tl.storm {
			break
		}
		plan = append(plan, wr)
	}
	return plan, nil
}

// writeBackfill يكتب خطّة الرجوع — كل بند بمعاملة.
func (w *Worker) writeBackfill(ctx context.Context, plan []bfWrite) (int, error) {
	stored := 0
	for _, wr := range plan {
		n, err := w.writeOne(ctx, wr)
		if err != nil {
			return stored, err
		}
		stored += n
	}
	return stored, nil
}

func (w *Worker) writeOne(ctx context.Context, wr bfWrite) (int, error) {
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if wr.rematchID != 0 {
		tag, err := tx.Exec(ctx, `DELETE FROM driver_road_chunks WHERE id = $1`, wr.rematchID)
		if err != nil || tag.RowsAffected() != 1 {
			return 0, err
		}
	} else {
		tag, err := tx.Exec(ctx, `
			UPDATE driver_road_progress SET backfill_before = $3 WHERE driver_id = $1::uuid AND backfill_before = $2`,
			wr.driver, wr.bb, wr.newBB)
		if err != nil || tag.RowsAffected() != 1 {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM driver_road_chunks WHERE driver_id = $1::uuid AND from_at >= $2 AND to_at <= $3`,
			wr.driver, wr.newBB, wr.bb); err != nil {
			return 0, err
		}
	}
	if err := insertRows(ctx, tx, wr.driver, wr.rows); err != nil {
		return 0, err
	}
	return len(wr.rows), tx.Commit(ctx)
}

// tiles وقت البلاطات، مقروء مرّة بالساعة.
func (w *Worker) tiles(ctx context.Context, now time.Time, o Options) time.Time {
	if w.Tiles == nil {
		return time.Time{}
	}
	w.mu.Lock()
	fresh := !w.tilesRead.IsZero() && now.Sub(w.tilesRead) < o.TilesEvery
	t := w.tilesAt
	w.mu.Unlock()
	if fresh {
		return t
	}
	t = w.Tiles(ctx)
	w.mu.Lock()
	w.tilesAt, w.tilesRead = t, now
	w.mu.Unlock()
	return t
}

// ───────────────────────── لغير المطابق ─────────────────────────

// DB ما بتحتاجه الدوال العامّة (مجمّع أو معاملة).
type DB = drivermonitor.DB

// MarkDirty **نقاط متأخرة** (دفعة الطابور) أقدم من مؤشّر السائق ⇒ القطع من عندها بتنعاد. تحديث واحد.
// أقدم من `frozen_before` ما بتنمسّ.
func MarkDirty(ctx context.Context, db DB, driverID string, oldest time.Time) error {
	_, err := db.Exec(ctx, `
		UPDATE driver_road_progress
		   SET dirty_from = LEAST(COALESCE(dirty_from, $2), $2), dirty_rev = dirty_rev + 1
		 WHERE driver_id = $1::uuid AND matched_until IS NOT NULL AND $2 < matched_until
		   AND $2 >= COALESCE(frozen_before, '-infinity'::timestamptz)`, driverID, oldest)
	return err
}

// Freeze **عقد كنّاس العدّاد**: ما قبل `before` صار نهائي ⇒ المطابق ما بيخزّن ولا بيعيد ولا بيرجع قبله.
func Freeze(ctx context.Context, db DB, driverID string, before time.Time) error {
	_, err := db.Exec(ctx, `
		INSERT INTO driver_road_progress (driver_id, frozen_before) VALUES ($1::uuid, $2)
		ON CONFLICT (driver_id) DO UPDATE
		   SET frozen_before = GREATEST(COALESCE(driver_road_progress.frozen_before, EXCLUDED.frozen_before), EXCLUDED.frozen_before)`,
		driverID, before)
	return err
}

// Prune **بيمحي القطع الأقدم من مدّة حفظ الأثر** — نفس ساعة تقليم الأثر.
func Prune(ctx context.Context, db DB, retentionDays int) (int64, error) {
	if retentionDays < 1 {
		retentionDays = drivermonitor.DefaultRetentionDays
	}
	tag, err := db.Exec(ctx, `
		DELETE FROM driver_road_chunks WHERE to_at < now() - make_interval(days => $1::int)`, retentionDays)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// Backlog **كم يوم لسّا للرجوع** — أكبرها بين السائقين.
func Backlog(ctx context.Context, db DB) (float64, error) {
	var secs *float64
	err := db.QueryRow(ctx, `
		SELECT max(EXTRACT(EPOCH FROM p.backfill_before - f.first))::float8
		  FROM driver_road_progress p
		 CROSS JOIN LATERAL (SELECT min(t.recorded_at) AS first FROM driver_trail t
		                      WHERE t.driver_id = p.driver_id
		                        AND t.recorded_at >= COALESCE(p.frozen_before, '-infinity'::timestamptz)) f
		 WHERE p.backfill_before IS NOT NULL AND f.first IS NOT NULL AND p.backfill_before > f.first`).Scan(&secs)
	if err != nil || secs == nil {
		return 0, err
	}
	return math.Round(*secs/86400*10) / 10, nil
}
