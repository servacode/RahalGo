package qa

// حاقنُ الفشل — **`P-6`.**
//
// # لماذا محفّزٌ في القاعدة لا خطّافٌ في الشيفرة
//
// **السؤالُ الذي تجيب عنه هذه المرحلة**: إذا نجحت الخطواتُ ١..N وسقطت
// N+1، **ما الحقيقةُ الباقيةُ في القاعدة والمال والحال؟**
//
// **وجوابُه يحتاج أن تسقط خطوةٌ بعينها** — لا خطوةٌ عشوائيّة، ولا «أطفئ
// القاعدةَ وانظر» (البند ٢٦).
//
// **ولا بابَ خلفيٌّ في المنتج** (البند ١): لا نقطةَ تصحيحٍ ولا رايةَ فشلٍ
// ولا متغيّرَ بيئةٍ يُشغَّل في الإنتاج سهواً، **ولا سطرَ شيفرةٍ تشغيليّةٍ
// يتبدّل.**
//
// **فالسقوطُ يُصنَع حيث لا يراه المنتج**: **محفّزٌ في قاعدة الاختبار
// يرفع استثناءً حين تُكتب صفّاً بعينه.** **والمنتجُ يرى خطأَ قاعدةٍ
// عاديّاً** — وهو بالضبط ما سيراه يومَ تقع العلّةُ حقّاً.
//
// # وخصائصُ كلّ نقطة (البند ٢)
//
//	NAMED         اسمٌ يُقرأ في التقرير
//	DETERMINISTIC تصيب ما سُمّي لا غيره
//	COUNTED       مرّةً واحدةً أو عدداً محدوداً
//	SCOPED        بصفٍّ بعينه — فلا تُصيب سيناريو غيرَها
//	CLEANED       تُنزَع بعد الاختبار

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// failpointDDL بنيةُ الحاقن — **تُنشأ مرّةً في العمليّة.**
//
// **والمحفّزُ عامٌّ**: يقرأ اسمَ الجدول والعمليّةَ من سياقه
// (`TG_TABLE_NAME` · `TG_OP`)، **فلا دالّةَ لكلّ جدول.**
const failpointDDL = `
CREATE TABLE IF NOT EXISTS qa_failpoints (
	id         bigserial PRIMARY KEY,
	name       text NOT NULL,
	table_name text NOT NULL,
	op         text NOT NULL,
	match_col  text,
	match_val  text,
	remaining  int  NOT NULL DEFAULT 1,
	seq_name   text NOT NULL
);

CREATE OR REPLACE FUNCTION qa_failpoint_fire() RETURNS trigger AS $$
DECLARE
	fp qa_failpoints%ROWTYPE;
BEGIN
	FOR fp IN
		SELECT * FROM qa_failpoints
		WHERE table_name = TG_TABLE_NAME AND op = TG_OP
		ORDER BY id
	LOOP
		IF fp.match_col IS NULL
		   OR (to_jsonb(NEW) ->> fp.match_col) = fp.match_val THEN
			-- ══════════════════════════════════════════════════════
			-- **والعدُّ بمتتالِيةٍ لا بجدول**
			-- ══════════════════════════════════════════════════════
			--
			-- **جرّبتُ تحديثَ عدّادٍ في جدولٍ
			-- فكان يعود صفراً دائماً** — **وRAISE يُرجع المعاملةَ
			-- كلَّها ومعها تحديثي.** فالنقطةُ تُصيب ولا تُحصى.
			--
			-- **وnextval لا يخضع للإرجاع** — وهو ما يجعلها الأداةَ
			-- الوحيدةَ التي تعدّ ما أسقطته.
			IF nextval(fp.seq_name) <= fp.remaining THEN
				RAISE EXCEPTION 'QA_FAILPOINT %', fp.name
					USING ERRCODE = 'raise_exception';
			END IF;
		END IF;
	END LOOP;
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;
`

var failpointOnce sync.Once
var failpointErr error

// Failpoint نقطةُ فشلٍ مسلَّحة.
type Failpoint struct {
	h     *Harness
	id    int64
	seq   string
	times int
	Name  string
	Table string
	Op    string
}

// seqCounter يمنح كلَّ نقطةٍ متتالِيةً خاصّةً بها.
var seqCounter atomic.Int64

// Arm يُسلّح نقطةَ فشلٍ على جدولٍ وعمليّة.
//
//	times   كم مرّةً تُصيب ثمّ تصمت — **واحدةٌ عادةً.**
//	col/val تُصيب الصفَّ الذي يحمل هذه القيمةَ في هذا العمود — **وفارغان
//	        يعنيان أيَّ صفّ.** (وهو ما يمنع نقطةً من إصابة سيناريو غيرِها.)
func (h *Harness) Arm(name, table, op string, times int, col, val string) *Failpoint {
	h.T.Helper()
	failpointOnce.Do(func() {
		_, failpointErr = h.Pool.Exec(ctxBG(), failpointDDL)
	})
	if failpointErr != nil {
		h.T.Fatalf("qa: تعذّر تجهيزُ الحاقن: %v", failpointErr)
	}
	op = strings.ToUpper(op)
	// **والمحفّزُ يُركَّب على الجدول عند التسليح ويُنزَع عند النزع** —
	// **فجدولٌ لا نقطةَ عليه لا يمرّ بمحفّزٍ أصلاً.**
	trg := triggerName(table, op)
	if _, err := h.Pool.Exec(ctxBG(), fmt.Sprintf(`
		DROP TRIGGER IF EXISTS %s ON %s;
		CREATE TRIGGER %s BEFORE %s ON %s
		FOR EACH ROW EXECUTE FUNCTION qa_failpoint_fire();`,
		trg, table, trg, op, table)); err != nil {
		h.T.Fatalf("qa: تعذّر تركيبُ محفّز %s: %v", name, err)
	}

	var id int64
	var mc, mv any
	if col != "" {
		mc, mv = col, val
	}
	seq := fmt.Sprintf("qa_fp_seq_%d", seqCounter.Add(1))
	if _, err := h.Pool.Exec(ctxBG(),
		fmt.Sprintf(`DROP SEQUENCE IF EXISTS %s; CREATE SEQUENCE %s`, seq, seq)); err != nil {
		h.T.Fatalf("qa: متتالِيةُ %s: %v", name, err)
	}
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO qa_failpoints (name, table_name, op, match_col, match_val, remaining, seq_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		name, table, op, mc, mv, times, seq).Scan(&id); err != nil {
		h.T.Fatalf("qa: تعذّر تسليحُ %s: %v", name, err)
	}
	fp := &Failpoint{h: h, id: id, seq: seq, times: times, Name: name, Table: table, Op: op}
	h.T.Cleanup(fp.Disarm)
	return fp
}

// ArmAny نقطةٌ تصيب أوّلَ صفٍّ يُكتب في الجدول — **بلا تضييق.**
func (h *Harness) ArmAny(name, table, op string) *Failpoint {
	return h.Arm(name, table, op, 1, "", "")
}

// Disarm ينزع النقطةَ ومحفّزَها — **ولا يبقى أثرٌ لاختبارٍ انتهى** (البند ٢٨).
func (f *Failpoint) Disarm() {
	ctx := context.Background()
	_, _ = f.h.Pool.Exec(ctx, `DELETE FROM qa_failpoints WHERE id = $1`, f.id)
	var others int
	_ = f.h.Pool.QueryRow(ctx, `
		SELECT count(*) FROM qa_failpoints WHERE table_name = $1 AND op = $2`,
		f.Table, f.Op).Scan(&others)
	if others == 0 {
		_, _ = f.h.Pool.Exec(ctx,
			fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON %s`, triggerName(f.Table, f.Op), f.Table))
	}
	_, _ = f.h.Pool.Exec(ctx, fmt.Sprintf(`DROP SEQUENCE IF EXISTS %s`, f.seq))
}

// hits كم نداءً بلغ النقطةَ — **من المتتالِية، فلا يُمحى بإرجاع.**
func (f *Failpoint) hits() int {
	var n int
	if err := f.h.Pool.QueryRow(ctxBG(),
		fmt.Sprintf(`SELECT CASE WHEN is_called THEN last_value ELSE 0 END FROM %s`, f.seq)).
		Scan(&n); err != nil {
		return 0
	}
	return n
}

// Fired كم مرّةً أسقطت فعلاً.
func (f *Failpoint) Fired() int {
	if n := f.hits(); n < f.times {
		return n
	}
	return f.times
}

// Remaining كم بقي لها.
func (f *Failpoint) Remaining() int { return f.times - f.Fired() }

// MustFire يشترط أنّها أصابت — **وسقوطٌ لم يقع يعني اختباراً لم يختبر شيئاً.**
func (f *Failpoint) MustFire(t *testing.T) {
	t.Helper()
	if n := f.Fired(); n == 0 {
		t.Fatalf("FAILPOINT %s لم تُصِب — **والسيناريو لم يُختبَر**", f.Name)
	} else {
		t.Logf("FAILPOINT %s أصابت %d مرّة (%s %s)", f.Name, n, f.Op, f.Table)
	}
}

// MustNotFire يشترط أنّها لم تُصِب — **لإثبات أنّ نقطةً لا تُصيب غيرَ مقصدها.**
func (f *Failpoint) MustNotFire(t *testing.T) {
	t.Helper()
	if n := f.Fired(); n != 0 {
		t.Errorf("FAILPOINT %s أصابت %d مرّة — **وكان يجب ألّا تُصيب**", f.Name, n)
	}
}

func triggerName(table, op string) string {
	return "qa_fp_" + strings.ToLower(table) + "_" + strings.ToLower(op)
}

// FailpointsClean يتحقّق أنّ القاعدةَ خلَت من نقاطٍ ومحفّزات (البند ٢٨).
func FailpointsClean(t *testing.T, h *Harness) {
	t.Helper()
	var pts, trgs int
	var seqs int
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT (SELECT count(*) FROM qa_failpoints),
		       (SELECT count(*) FROM pg_trigger WHERE tgname LIKE 'qa\_fp\_%'),
		       (SELECT count(*) FROM pg_class WHERE relkind = 'S' AND relname LIKE 'qa\_fp\_seq\_%')`).
		Scan(&pts, &trgs, &seqs)
	if pts != 0 || trgs != 0 || seqs != 0 {
		t.Errorf("FAILPOINT CLEANUP خُرق: بقيت %d نقطةً و%d محفّزاً و%d متتالِية", pts, trgs, seqs)
	} else {
		t.Logf("FAILPOINT CLEANUP = PASS — لا نقطةَ ولا محفّزَ ولا متتالِيةَ باقية")
	}
}

// WaitFire ينتظر أن تُصيب النقطةُ — **لأنّ بعضَ الأعمال لا تُنتظَر.**
//
// **والتدقيقُ يُكتب في خيطٍ منفصل** (`server/audit.go:53`)، **فقراءةُ
// العدّاد فورَ عودة الردّ تقرأ صفراً وتقول «لم يُنادَ» وهو نُودي.**
//
// **وليس نوماً مِقوداً**: انتظارُ شرطٍ يُقرأ من القاعدة، **والمهلةُ حدٌّ
// لا آلةُ تحكّم.**
func (f *Failpoint) WaitFire(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if f.Fired() > 0 {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(2 * time.Millisecond)
	}
}
