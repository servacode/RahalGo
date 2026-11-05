package server

import (
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
	mu sync.Mutex
	st clockState
}

var deviceClocks sync.Map // driverID → *clockEntry

// deviceClockFix **كم يُضاف لأوقات الجهاز في هذا النداء** — صفرٌ بلا ترويسة (النسخ القديمة).
//
// **وفي الذاكرة لكلّ سائق** (عمليّةٌ واحدة، كنقطة العرض): بعد إعادة التشغيل يُحكم من أوّل نداء.
// ويكتب سطراً في السجلّ حين يتبدّل تصحيحُ سائق — **ليُعرف كم هاتفاً ساعتُه خاطئة.**
func (s *Server) deviceClockFix(r *http.Request, driverID string, now time.Time) time.Duration {
	v := strings.TrimSpace(r.Header.Get(deviceTimeHeader))
	if v == "" {
		return 0
	}
	dev, err := time.Parse(time.RFC3339Nano, v)
	if err != nil || dev.IsZero() {
		return 0
	}
	e, _ := deviceClocks.LoadOrStore(driverID, &clockEntry{})
	entry := e.(*clockEntry)
	entry.mu.Lock()
	prev := entry.st
	entry.st = nextClockFix(prev, now.Sub(dev))
	fix := entry.st.fix
	entry.mu.Unlock()
	if fix != prev.fix && s.logger != nil {
		s.logger.Info("الموضع: ساعةُ الهاتف تُصحَّح", "driver", driverID,
			"fix_sec", int(fix.Seconds()), "previous_fix_sec", int(prev.fix.Seconds()))
	}
	return fix
}
