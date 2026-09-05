package qa

// اختبارُ الحاقنِ لنفسِه — **`P-6` البند ٤.**
//
// **وحاقنٌ لم يُثبَت أنّه يصيب ما سُمّي وحدَه لا يُوثَق بنتيجةٍ منه.**

import (
	"context"
	"strings"
	"testing"
)

// TestFAIL_SelfTest_HitsOnlyTheNamedStep **ثلاثُ خطواتٍ · والثالثةُ تسقط.**
//
//	الخطوة ١ تنجح
//	الخطوة ٢ تنجح
//	الخطوة ٣ تسقط بنقطةٍ مسمّاة
//
// **ويُثبَت أربعةُ أشياء**: أصابت المقصودةَ · ولم تمسّ غيرَها ·
// **وما نجح بقي** · **والنقطةُ نُزعت بعدها.**
func TestFAIL_SelfTest_HitsOnlyTheNamedStep(t *testing.T) {
	h := New(t)
	ctx := ctxBG()

	// **جدولُ فكسچرٍ للاختبار وحدَه** — لا جدولَ منتجٍ يُمسّ في اختبار
	// الحاقن نفسِه.
	if _, err := h.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS qa_selftest_steps (
			id bigserial PRIMARY KEY, scenario text NOT NULL, step text NOT NULL)`); err != nil {
		t.Fatalf("جدولُ الفكسچر: %v", err)
	}
	scen := uniq("s")
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM qa_selftest_steps WHERE scenario = $1`, scen)
	})

	// **مسلَّحةٌ على الخطوة الثالثة بعينها** — بالعمود والقيمة.
	fp := h.Arm("SELFTEST/step-3", "qa_selftest_steps", "INSERT", 1, "step", "three")

	write := func(step string) error {
		_, err := h.Pool.Exec(ctx,
			`INSERT INTO qa_selftest_steps (scenario, step) VALUES ($1, $2)`, scen, step)
		return err
	}

	if err := write("one"); err != nil {
		t.Fatalf("الخطوة ١ كان يجب أن تنجح: %v", err)
	}
	if err := write("two"); err != nil {
		t.Fatalf("الخطوة ٢ كان يجب أن تنجح: %v", err)
	}
	err := write("three")
	if err == nil {
		t.Fatal("الخطوة ٣ نجحت — **والحاقنُ لم يُصِب**")
	}
	if !strings.Contains(err.Error(), "QA_FAILPOINT") {
		t.Errorf("سقطت بخطأٍ آخرَ لا بالنقطة: %v", err)
	}
	t.Logf("الخطوة ٣ سقطت كما سُمّي: %s", fpLine(err.Error()))
	fp.MustFire(t)

	// **وما نجح بقي** — الفشلُ لم يجرّ ما قبله.
	var kept []string
	rows, err := h.Pool.Query(ctx,
		`SELECT step FROM qa_selftest_steps WHERE scenario = $1 ORDER BY id`, scen)
	if err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	for rows.Next() {
		var s string
		_ = rows.Scan(&s)
		kept = append(kept, s)
	}
	rows.Close()
	t.Logf("الباقي في القاعدة: %v", kept)
	if len(kept) != 2 || kept[0] != "one" || kept[1] != "two" {
		t.Errorf("الخطوتان الناجحتان لم تبقيا كما هما: %v", kept)
	}

	// **ومرّةً واحدةً** — الرصاصةُ نفدت.
	if fp.Remaining() != 0 {
		t.Errorf("النقطةُ ما تزال مسلَّحةً بعد إصابتها: %d", fp.Remaining())
	}
	if err := write("three"); err != nil {
		t.Errorf("الإعادةُ بعد نفاد النقطة سقطت: %v", err)
	} else {
		t.Logf("ONE-SHOT = PROVEN — الإعادةُ نجحت بعد نفاد النقطة")
	}

	fp.Disarm()
	if err := write("three"); err != nil {
		t.Errorf("بعد النزع ما تزال تسقط: %v", err)
	}
	t.Logf("FAILURE INJECTION SELF-TEST = PROVEN")
}

// TestFAIL_SelfTest_ScopeDoesNotLeak **البند ٣ — نقطةُ سيناريو لا تُصيب غيرَه.**
func TestFAIL_SelfTest_ScopeDoesNotLeak(t *testing.T) {
	h := New(t)
	ctx := ctxBG()
	if _, err := h.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS qa_selftest_steps (
			id bigserial PRIMARY KEY, scenario text NOT NULL, step text NOT NULL)`); err != nil {
		t.Fatalf("جدولُ الفكسچر: %v", err)
	}
	a, b := uniq("A"), uniq("B")
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM qa_selftest_steps WHERE scenario IN ($1, $2)`, a, b)
	})

	fp := h.Arm("SELFTEST/scenario-A", "qa_selftest_steps", "INSERT", 1, "scenario", a)

	// **السيناريو ب يمرّ سالماً والنقطةُ مسلَّحةٌ لأ.**
	if _, err := h.Pool.Exec(ctx,
		`INSERT INTO qa_selftest_steps (scenario, step) VALUES ($1, 'x')`, b); err != nil {
		t.Fatalf("SCENARIO ISOLATION خُرق: نقطةُ أ أصابت ب: %v", err)
	}
	t.Logf("السيناريو ب مرّ سالماً — والنقطةُ مسلَّحةٌ لأ")
	if fp.Fired() != 0 {
		t.Errorf("النقطةُ أصابت %d مرّةً وهي لأ وحدَه", fp.Fired())
	}

	// **ثمّ أ يسقط.**
	if _, err := h.Pool.Exec(ctx,
		`INSERT INTO qa_selftest_steps (scenario, step) VALUES ($1, 'x')`, a); err == nil {
		t.Fatal("السيناريو أ نجح — والنقطةُ مسلَّحةٌ له")
	}
	fp.MustFire(t)
	t.Logf("SCENARIO A FAILPOINT DOES NOT AFFECT SCENARIO B = PROVEN")
}

// TestFAIL_SelfTest_CleanupLeavesNothing **البند ٢٨.**
func TestFAIL_SelfTest_CleanupLeavesNothing(t *testing.T) {
	h := New(t)
	func() {
		fp := h.ArmAny("SELFTEST/cleanup", "qa_selftest_steps", "INSERT")
		var trgs int
		_ = h.Pool.QueryRow(ctxBG(),
			`SELECT count(*) FROM pg_trigger WHERE tgname = $1`,
			triggerName("qa_selftest_steps", "INSERT")).Scan(&trgs)
		if trgs != 1 {
			t.Errorf("المحفّزُ لم يُركَّب عند التسليح: %d", trgs)
		}
		fp.Disarm()
	}()
	FailpointsClean(t, h)
}

// fpLine أوّلُ سطرٍ من رسالة الخطأ — **فالتشخيصُ يُقرأ ولا يُغرِق.**
func fpLine(s string) string {
	if i := strings.IndexByte(s, 10); i > 0 {
		return s[:i]
	}
	return s
}
