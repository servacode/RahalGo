package qa

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **`XG-34` — نطاقُ القفل وتسرّبُ المعاملات**
// ══════════════════════════════════════════════════════════════════════
//
// **رُصد تعثّرٌ متقطّعٌ تحت حمل الحزمة الكاملة** — ٦٢ و٣٢ و٣٠ ثانية،
// **ومهلةُ الطلب ثلاثون.** **والظنُّ كان أنّ منسّقَ منع التكرار يُطيل
// إمساكَ المعاملات فيتنازع.**
//
// **وهذان الحارسان يقطعان الظنّ**: **مفتاحان مختلفان لا يتسلسلان**،
// **ولا معاملةَ تبقى مفتوحةً بعد الفحص.**

// ══════════════════════════════════════════════════════════════════════
// **١ · مفتاحان مختلفان لا يتسلسل أحدُهما خلف الآخر**
// ══════════════════════════════════════════════════════════════════════
//
// **والقفلُ على صفٍّ واحدٍ بمفتاح `(user_id, endpoint, key)`** — **فمن
// اختلف مفتاحُه اختلف صفُّه.** **ولو تسلسلا لكان القفلُ على الجدول لا
// على الصفّ، وذلك عطبٌ يخنق المنصّةَ تحت حمل.**
func TestXG34_DifferentKeysDoNotSerialize(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)

	before := idemOrdersOf(t, h, cust.ID)
	body := orderBody(item, 1)
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "مفتاحٌ-أ", Do: func(ctx context.Context) any {
			return h.POSTKey("/api/v1/orders", cust.Token, "xg34-a", body)
		}},
		Actor{Name: "مفتاحٌ-ب", Do: func(ctx context.Context) any {
			return h.POSTKey("/api/v1/orders", cust.Token, "xg34-b", body)
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	after := idemOrdersOf(t, h, cust.ID)
	t.Logf("مفتاحان مختلفان: طلباتٌ %d ← %d · نجح %d من 2 — %s",
		before, after, r.CountOK(), r)

	// **والتسلسلُ يظهر في العدّ لا في الزمن**: **نداءان كلاهما نجح
	// وأنتج طلباً يعني أنّهما لم يحبس أحدُهما الآخرَ إلى الفشل.**
	if after != before+2 {
		t.Errorf("**مفتاحان مختلفان ولم يُنتجا طلبين**: %d ← %d — "+
			"**فالقفلُ أوسعُ من صفِّه.**", before, after)
	}
	if r.CountOK() != 2 {
		t.Errorf("نجح %d من 2 — **ومفتاحان مختلفان لا يتنازعان**", r.CountOK())
	}
	// **وتداخلٌ مقيسٌ يُثبت أنّهما تزاحما فعلاً** — ولولاه لكان الفحصُ
	// نداءين متتاليين لا سباقاً.
	if r.Probe.Max() < 2 {
		t.Errorf("تداخلٌ مقيسٌ %d — **والسيناريو يشترط تزامناً**", r.Probe.Max())
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · لا معاملةَ تُهجَر ولا جلسةَ تُحجَب طويلاً**
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان يقيسه هذا الفاحصُ — ولماذا كان يكذب
//
// **كان يعدّ `state = 'idle in transaction'` في القاعدة كلِّها لحظةً
// واحدة**، **بلا عمرٍ ولا مالكٍ ولا إعادةِ قياس** — **فيتّهم معاملةً
// حيّةً عمرُها ثلاثةُ أجزاءٍ من الألف.**
//
// **وقيس في دورة ٣٥** (`dc5998b3`): **سقط أربعَ مرّاتٍ من ثمانٍ وهو
// يُشغَّل وحدَه**، والمتَّهمُ في اللقطة:
//
//	pid=44651 · xact_age=0.003 · wait=Client/ClientRead · blockedBy=0
//	q = UPDATE notifications SET push_pending = false WHERE id = $1
//
// **وتلك جملةُ `fanOutOne` الثانيةُ في معاملتها** — **جولةٌ خلفيّةٌ
// أيقظها الطلبُ نفسُه** (`Kick`)، **تعمل بعد أن يعود الردّ.** ولقطةٌ
// أخرى كان استعلامُها `begin` حرفاً.
//
// **و«معلَّقٌ في معاملة» ليس عطباً**: **هي حالُ كلِّ معاملةٍ بين
// جملتين** — الخادمُ ينتظر جملةَ عميله (`Client/ClientRead`).
// **وقيس أثناء شوطٍ ناجحٍ بلا سقطةٍ واحدة: عشرون ظهوراً، أعمارُها
// بين ٥٤ و٣٠٧ أجزاءٍ من الألف.**
//
// # فما العطبُ إذاً
//
// **المهجورة**: معاملةٌ يمضي عليها الدهرُ ولا صاحبَ لها — **تمسك
// اتّصالاً وقفلاً فيتعثّر فحصٌ بعيدٌ بلا سببٍ ظاهر.**
//
// **فيُشترط ثلاثةٌ**: **عمرٌ يتجاوز الحدّ** · **وبقاءٌ بين لقطتين
// بينهما ثلاثُ ثوانٍ** — فالعابرةُ تختفي — · **وليست لقطةَ الفاحص
// نفسِه.**
//
// **والمحجوبُ يُقاس بمدّته لا بوجوده**: **قيس أطولَ انتظارِ قفلٍ في
// عشرة أشواطٍ كاملةٍ للعائلة: خمسون جزءاً من الألف** — **وذاك تزاحمٌ
// صحيحٌ لا عطب.**
//
// # وفاحصٌ لا يُثبت أنّه يرى العطبَ لا يُصدَّق
//
// **ففيه شاهدٌ سالب**: **تُهجَر معاملةٌ عمداً فيجب أن تُرى**، **ثمّ
// تُفكّ فيجب أن يصفو.** **وحارسٌ لم يُرَ ساقطاً لا يُقال إنّه يحرس.**

// abandonedAfter **حدُّ الهجر** — **وأطولُ ما قيس لمعاملةٍ صحيحةٍ
// جزءٌ من الثلاثة أعشار.** **وخمسٌ فوقه بمراتب، والمهجورةُ بلا حدّ.**
const abandonedAfter = 5 * time.Second

// suspect صفُّ اتّهامٍ من `pg_stat_activity`.
type suspect struct {
	PID       int
	State     string
	XactAge   float64
	QueryAge  float64
	WaitType  string
	WaitEvent string
	BlockedBy int
	Query     string
}

func (s suspect) String() string {
	return fmt.Sprintf("pid=%d state=%q xact=%.2fs query=%.2fs wait=%s/%s blockedBy=%d q=%s",
		s.PID, s.State, s.XactAge, s.QueryAge, s.WaitType, s.WaitEvent, s.BlockedBy, s.Query)
}

// longLived الجلساتُ التي تجاوزت الحدَّ الآن — **بلا حكمٍ بعد.**
//
// **ولا تُحسَب جلسةُ الفاحص** ولا **قفلُ الجلسة المقصود** في `testdb`
// (**وهو `idle` لا `idle in transaction`، ولا معاملةَ له**).
func longLived(t *testing.T, h *Harness, min time.Duration) map[int]suspect {
	t.Helper()
	out := map[int]suspect{}
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT pid, state,
		       COALESCE(extract(epoch from (now() - xact_start)), 0),
		       COALESCE(extract(epoch from (now() - query_start)), 0),
		       COALESCE(wait_event_type, '-'), COALESCE(wait_event, '-'),
		       cardinality(pg_blocking_pids(pid)),
		       left(regexp_replace(query, '\s+', ' ', 'g'), 120)
		  FROM pg_stat_activity
		 WHERE datname = current_database()
		   AND pid <> pg_backend_pid()
		   AND ((state = 'idle in transaction' AND now() - xact_start > $1::interval)
		        OR (cardinality(pg_blocking_pids(pid)) > 0
		            AND now() - query_start > $1::interval))`, min.String())
	if err != nil {
		t.Skipf("تعذّرت قراءةُ حال الخادم: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s suspect
		if err := rows.Scan(&s.PID, &s.State, &s.XactAge, &s.QueryAge,
			&s.WaitType, &s.WaitEvent, &s.BlockedBy, &s.Query); err != nil {
			t.Fatalf("مسحُ حال الخادم: %v", err)
		}
		out[s.PID] = s
	}
	return out
}

// abandoned ما بقي بين لقطتين — **فالعابرُ يسقط من الحساب.**
func abandoned(t *testing.T, h *Harness) []suspect {
	t.Helper()
	first := longLived(t, h, abandonedAfter)
	if len(first) == 0 {
		return nil
	}
	time.Sleep(3 * time.Second)
	var out []suspect
	for pid, s := range longLived(t, h, abandonedAfter) {
		if _, was := first[pid]; was {
			out = append(out, s)
		}
	}
	return out
}

func TestXG34_NoTransactionIsLeftOpen(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)

	for i := 0; i < 5; i++ {
		if got := h.POSTKey("/api/v1/orders", cust.Token,
			uniq("xg34-leak"), orderBody(item, 1)); got.Code >= 400 {
			t.Fatalf("الطلبُ %d: %s", i+1, got)
		}
	}

	// ── ١ ── **الشاهدُ السالب: يُهجَر عمداً فيجب أن يُرى** ────────
	//
	// **وبلا هذا لا يُعرَف أنّ الفاحصَ يرى شيئاً أصلاً.**
	conn, err := h.Pool.Acquire(ctxBG())
	if err != nil {
		t.Fatalf("اتّصالٌ للهجر: %v", err)
	}
	tx, err := conn.Begin(ctxBG())
	if err != nil {
		conn.Release()
		t.Fatalf("معاملةُ الهجر: %v", err)
	}
	var leakPID int
	if err := tx.QueryRow(ctxBG(), `SELECT pg_backend_pid()`).Scan(&leakPID); err != nil {
		t.Fatalf("معرّفُ المهجورة: %v", err)
	}
	time.Sleep(abandonedAfter + time.Second)

	seen := abandoned(t, h)
	found := false
	for _, s := range seen {
		if s.PID == leakPID {
			found = true
			t.Logf("NEGATIVE CONTROL: رُئيت المهجورةُ — %s", s)
		}
	}
	if !found {
		t.Errorf("**هُجرت معاملةٌ (pid=%d) ولم يرَها الفاحص** — "+
			"**وحارسٌ لا يرى العطبَ المصنوعَ بيدٍ لا يراه يومَ يقع.**", leakPID)
	}

	_ = tx.Rollback(ctxBG())
	conn.Release()

	// ── ٢ ── **ثمّ يصفو** ─────────────────────────────────────────
	rest := abandoned(t, h)
	if len(rest) > 0 {
		var lines []string
		for _, s := range rest {
			lines = append(lines, "  "+s.String())
		}
		t.Errorf("**معاملةٌ مهجورةٌ أو جلسةٌ محجوبةٌ فوق %s** — "+
			"**تمسك اتّصالاً وقفلاً فيتعثّر فحصٌ بعيدٌ بلا سببٍ ظاهر** "+
			"(`XG-34`):\n%s", abandonedAfter, strings.Join(lines, "\n"))
	}

	st := h.Pool.Stat()
	t.Logf("بعد خمسةِ نداءاتٍ محميّة: مهجورةٌ=%d · المَسبَح: كلٌّ=%d محجوزٌ=%d خاملٌ=%d",
		len(rest), st.TotalConns(), st.AcquiredConns(), st.IdleConns())
}
