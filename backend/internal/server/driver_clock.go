package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **تصحيحُ ساعة الهاتف** (A6.4، نسخة السائق 0.9.0 — ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// **وقتُ النقطة من ساعة الجهاز** (`recorded_at` ووقتُ كلّ نقطةٍ في الدفعة). **وهاتفٌ ساعتُه
// متأخّرةٌ ساعةً يُكتب موضعُه قديماً**: يشيخ عند التوزيع فلا يُعرض عليه طلب، ولا يُبثّ للخريطة
// (أقدمُ من دقيقتين)، **وما تأخّر ساعتين لا يُكتب أبداً** (`batchMaxAge`).
//
// **فالنسخةُ الجديدة ترسل ساعتَها لحظةَ الإرسال** في ترويسة `X-Device-Time`، والفرقُ بينها وبين
// ساعة الخادم يُضاف لكلّ وقتٍ في النداء. **والنسخُ القديمة لا ترسلها** فلا يتغيّر لها شيء.
//
// **وزمنُ الوصول داخلٌ في الفرق** (ثوانٍ، ومهلةُ التطبيق عشرون): **فما دون الدقيقة لا يُصحَّح**
// — ساعةٌ صحيحةٌ على شبكةٍ بطيئة. **وما صُحّح يثبت** ما دام الفرقُ الجديد ضمن نصف دقيقةٍ منه:
// تصحيحٌ يتأرجح مع زمن الوصول يقلب ترتيبَ نقطتين بينهما أربعُ ثوانٍ.

// deviceTimeHeader **ساعةُ الجهاز لحظةَ الإرسال** — `RFC3339`.
const deviceTimeHeader = "X-Device-Time"

const (
	// clockFixFrom **أقلُّ فرقٍ يُصحَّح** — دونه زمنُ وصولٍ لا ساعةٌ خاطئة.
	clockFixFrom = time.Minute
	// clockFixHold **ما دام الفرقُ الجديد ضمن هذا من التصحيح الحاليّ يبقى التصحيحُ كما هو.**
	clockFixHold = 30 * time.Second
)

// clockState **التصحيحُ المعمولُ به لسائق.**
type clockState struct {
	fix   time.Duration
	known bool
}

// nextClockFix **التصحيحُ بعد ترويسةٍ جديدة** — دالةٌ صافية.
//
// `observed` = ساعةُ الخادم − ساعةُ الجهاز (فيه زمنُ الوصول).
func nextClockFix(prev clockState, observed time.Duration) clockState {
	if prev.known && absDuration(observed-prev.fix) <= clockFixHold {
		return prev
	}
	if absDuration(observed) < clockFixFrom {
		return clockState{known: true}
	}
	return clockState{fix: observed, known: true}
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

type clockEntry struct {
	mu     sync.Mutex
	loaded bool
	st     clockState
	// **آخرُ تبدّل** (مراجعةُ ٢٠٢٦-١١-٠٦): التصحيحُ قبله، ووقتُ الخادم لما تبدّل — لنقاط الطابور.
	prev      time.Duration
	changedAt time.Time
}

var deviceClocks sync.Map // driverID → *clockEntry

// clockView **التصحيحُ لنداءٍ واحد** — الحاليّ، وآخرُ تبدّلٍ إن كان خلال عمر الطابور.
type clockView struct {
	fix       time.Duration
	changed   bool
	prev      time.Duration
	changedAt time.Time
}

// correct **وقتُ نقطةٍ من الطابور بعد التصحيح.**
//
// # لماذا لا يكفي تصحيحُ لحظة الإرسال (مراجعةُ ٢٠٢٦-١١-٠٦)
//
// **ساعةُ الهاتف تنضبط غالباً لحظةَ عودة الشبكة** — أي قبل إفراغ الطابور مباشرة. فنقاطُ ساعةِ
// الانقطاع انلقطت بالساعة القديمة، **وتصحيحُ لحظة الإرسال كان يزيحها ساعات** فتُرمى أو تنحطّ
// بمكانٍ غلط. فبعد تبدّلٍ نقيس الوقتين: **بالتصحيح القديم** (يصحّ لما قبل التبدّل) **وبالجديد**
// (لما بعده) — والقديمُ يُؤخذ إذا وحدَه منطقيّ: لا يقع بعد التبدّل، والجديدُ يقع قبله.
// وإن صحّ الاثنان فالجديد (كما كان).
func (c clockView) correct(raw time.Time) time.Time {
	tNew := raw.Add(c.fix)
	if !c.changed || c.prev == c.fix {
		return tNew
	}
	tOld := raw.Add(c.prev)
	oldOK := !tOld.After(c.changedAt.Add(clockFixHold))
	newOK := !tNew.Before(c.changedAt.Add(-clockFixHold))
	if oldOK && !newOK {
		return tOld
	}
	return tNew
}

// clockKey **مفتاحُ التصحيح بـRedis** — فيثبت بعد إعادة تشغيل الخادم: **وتصحيحٌ يُحسب من جديدٍ
// يختلف بزمن الوصول**، فدفعةٌ أُعيد إرسالها تُكتب بأوقاتٍ أخرى ولا يمنعها `ON CONFLICT` فتتكرّر
// نقاطُها وتُعدّ مسافتُها مرّتين.
func clockKey(driverID string) string { return "devclock:" + driverID }

// clockSaved **ما يُحفظ بـRedis.**
type clockSaved struct {
	Fix       int64     `json:"fix"`
	Known     bool      `json:"known"`
	Prev      int64     `json:"prev"`
	ChangedAt time.Time `json:"changed_at"`
}

// deviceClockFix **كم يُضاف لأوقات الجهاز في هذا النداء** — صفرٌ بلا ترويسة (النسخ القديمة).
func (s *Server) deviceClockFix(r *http.Request, driverID string, now time.Time) time.Duration {
	return s.deviceClock(r, driverID, now).fix
}

// deviceClock **التصحيحُ لهذا النداء** — صفرٌ بلا ترويسة (النسخ القديمة).
//
// **وفي الذاكرة لكلّ سائق، ونسخةٌ بـRedis** تُقرأ أوّلَ مرّةٍ بعد إعادة التشغيل (انظر `clockKey`).
// ويكتب سطراً في السجلّ حين يتبدّل تصحيحُ سائق — **ليُعرف كم هاتفاً ساعتُه خاطئة.**
func (s *Server) deviceClock(r *http.Request, driverID string, now time.Time) clockView {
	v := strings.TrimSpace(r.Header.Get(deviceTimeHeader))
	if v == "" {
		return clockView{}
	}
	dev, err := time.Parse(time.RFC3339Nano, v)
	if err != nil || dev.IsZero() {
		return clockView{}
	}
	e, _ := deviceClocks.LoadOrStore(driverID, &clockEntry{})
	entry := e.(*clockEntry)
	entry.mu.Lock()
	if !entry.loaded {
		entry.loaded = true
		s.loadClock(r.Context(), driverID, entry)
	}
	prev := entry.st
	entry.st = nextClockFix(prev, now.Sub(dev))
	fix := entry.st.fix
	changed := entry.st != prev
	if prev.known && fix != prev.fix {
		entry.prev, entry.changedAt = prev.fix, now
	}
	view := clockView{fix: fix}
	if !entry.changedAt.IsZero() && now.Sub(entry.changedAt) <= batchMaxAge {
		view.changed, view.prev, view.changedAt = true, entry.prev, entry.changedAt
	}
	saved := clockSaved{Fix: int64(entry.st.fix), Known: entry.st.known, Prev: int64(entry.prev), ChangedAt: entry.changedAt}
	entry.mu.Unlock()
	if changed {
		s.saveClock(r.Context(), driverID, saved)
	}
	if fix != prev.fix && s.logger != nil {
		s.logger.Info("الموضع: ساعةُ الهاتف تُصحَّح", "driver", driverID,
			"fix_sec", int(fix.Seconds()), "previous_fix_sec", int(prev.fix.Seconds()))
	}
	return view
}

// loadClock **يقرأ التصحيحَ المحفوظ** (والقفلُ ممسوك) — وبلا Redis أو بلا قيمةٍ يبقى فارغاً.
func (s *Server) loadClock(ctx context.Context, driverID string, e *clockEntry) {
	if s.rdb == nil {
		return
	}
	raw, err := s.rdb.Get(ctx, clockKey(driverID)).Bytes()
	if err != nil {
		return
	}
	var c clockSaved
	if json.Unmarshal(raw, &c) != nil {
		return
	}
	e.st = clockState{fix: time.Duration(c.Fix), known: c.Known}
	e.prev, e.changedAt = time.Duration(c.Prev), c.ChangedAt
}

// saveClock **يحفظ التصحيح** ليوم — وتعثّرُه لا يُسقط الموضع.
func (s *Server) saveClock(ctx context.Context, driverID string, c clockSaved) {
	if s.rdb == nil {
		return
	}
	b, _ := json.Marshal(c)
	if err := s.rdb.Set(ctx, clockKey(driverID), b, 24*time.Hour).Err(); err != nil && s.logger != nil {
		s.logger.Warn("الموضع: حفظُ تصحيح الساعة", "driver", driverID, "err", err)
	}
}
