package server

// **حرّاسُ الدفعة الثالثة من راصد المنصّة** (٢٠٢٦-١٠-١٧): الدفترُ كلَّ ساعة
// بفحصٍ لكلّ ثابت، وبطءُ القاعدة بثلاث دقائقَ متتالية، وسطرُ الصباح —
// **وخرقٌ حقيقيٌّ في قاعدة الاختبار يُنبَّه به ثمّ يُقال «رجع طبيعي».**

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/platform"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestMoneyMonChecks_AlertOnceErroredUnknownThenRecovered
// **فحصٌ يسقط يُنبَّه به مرّة · والفحصُ الذي لم يُشغَّل لا يُحكم عليه · والعودةُ تُقال.**
func TestMoneyMonChecks_AlertOnceErroredUnknownThenRecovered(t *testing.T) {
	var b alertBook
	var sent []string
	send := func(s string) bool { sent = append(sent, s); return true }
	t0 := time.Date(2026, 10, 17, 3, 0, 0, 0, time.UTC)

	failing := []ledgerCheck{
		{ID: "FI-06.a", Name: "مستحقُّ المتجر يطابق", Count: 3},
		{ID: "FI-01.a", Name: "لا قيدَ بصفر", OK: true},
	}
	b.step(moneyMonChecks(failing), t0, send)
	if len(sent) != 1 {
		t.Fatalf("فحصٌ واحدٌ ساقطٌ يُنبَّه به مرّة — %v", sent)
	}
	want := "⚠️ فحص مالي فشل: FI-06.a — مستحقُّ المتجر يطابق (٣ سطر)"
	if !strings.HasPrefix(sent[0], want) {
		t.Fatalf("نصُّ التنبيه:\n%s\nوالمنتظَرُ أن يبدأ بـ:\n%s", sent[0], want)
	}
	// **والنتيجةُ نفسُها بعد دقيقة لا تُكرَّر.**
	b.step(moneyMonChecks(failing), t0.Add(time.Minute), send)
	if len(sent) != 1 {
		t.Fatalf("التنبيهُ تكرّر قبل الساعة — %v", sent)
	}
	// **والفحصُ الذي تعذّر تشغيلُه لم يُقَس** — فلا «رجع طبيعي» كاذب.
	errored := []ledgerCheck{{ID: "FI-06.a", Name: "مستحقُّ المتجر يطابق", Error: "timeout"}}
	b.step(moneyMonChecks(errored), t0.Add(2*time.Minute), send)
	if len(sent) != 1 {
		t.Fatalf("فحصٌ لم يُشغَّل حُكم عليه — %v", sent)
	}
	passing := []ledgerCheck{{ID: "FI-06.a", Name: "مستحقُّ المتجر يطابق", OK: true}}
	b.step(moneyMonChecks(passing), t0.Add(61*time.Minute), send)
	if len(sent) != 2 || sent[1] != "✅ رجع طبيعي: الفحص المالي FI-06.a رجع سليم." {
		t.Fatalf("عودةُ الفحص لم تُقَل — %v", sent)
	}
	// **وفحصان ساقطان تنبيهان** — وعودةُ أحدهما لا تُسكت الآخر.
	var b2 alertBook
	sent = nil
	two := []ledgerCheck{{ID: "FI-02.a", Count: 1}, {ID: "FI-03.a", Count: 2}}
	b2.step(moneyMonChecks(two), t0, send)
	if len(sent) != 2 {
		t.Fatalf("فحصان ساقطان — %d تنبيه", len(sent))
	}
	b2.step(moneyMonChecks([]ledgerCheck{{ID: "FI-02.a", OK: true}, {ID: "FI-03.a", Count: 2}}), t0.Add(time.Minute), send)
	if len(sent) != 3 || !strings.Contains(sent[2], "FI-02.a") {
		t.Fatalf("عودةُ واحدٍ من اثنين — %v", sent)
	}
}

// TestMoneyDue_FirstTickThenHourly **أوّلَ نبضةٍ بعد الإقلاع، ثمّ كلَّ ساعةٍ لا كلَّ دقيقة.**
func TestMoneyDue_FirstTickThenHourly(t *testing.T) {
	t0 := time.Date(2026, 10, 17, 3, 0, 0, 0, time.UTC)
	if !moneyDue(t0, time.Time{}) {
		t.Fatal("أوّلُ نبضةٍ لا تفحص الدفتر")
	}
	if moneyDue(t0.Add(59*time.Minute), t0) {
		t.Fatal("فُحص الدفترُ قبل الساعة")
	}
	if !moneyDue(t0.Add(time.Hour), t0) {
		t.Fatal("مرّت ساعةٌ ولم يُفحص الدفتر")
	}
}

// TestMoneySummary_MorningLine **سطرُ الصباح: سليمٌ بعدده · أو فاشلٌ بعدده.**
func TestMoneySummary_MorningLine(t *testing.T) {
	green := make([]ledgerCheck, 75)
	for i := range green {
		green[i].OK = true
	}
	if got := moneySummary(green); got != "الحسابات المالية: ٧٥ فحص سليم" {
		t.Fatalf("السليم: %q", got)
	}
	green[3] = ledgerCheck{Count: 4}
	green[9] = ledgerCheck{Count: 1}
	green[10] = ledgerCheck{Error: "x"}
	if got := moneySummary(green); got != "⚠️ الحسابات المالية: ٢ فحص فاشل" {
		t.Fatalf("الفاشل: %q", got)
	}
	if got := moneySummary([]ledgerCheck{{OK: true}, {Error: "x"}}); !strings.Contains(got, "ما قدرت شغّله") {
		t.Fatalf("ما لم يُشغَّل لا يُقال «سليم»: %q", got)
	}
}

// TestSlowWatch_ThreeTicksThenRecovered
// **دقيقتان بطيئتان لا تنبيه · الثالثةُ تنبيه · ودقيقةٌ سريعةٌ «رجع طبيعي».**
func TestSlowWatch_ThreeTicksThenRecovered(t *testing.T) {
	var w slowWatch
	var b alertBook
	var sent []string
	send := func(s string) bool { sent = append(sent, s); return true }
	t0 := time.Date(2026, 10, 17, 3, 0, 0, 0, time.UTC)
	tick := func(i int, ms int64, ok bool, limit int64) {
		b.step([]monCheck{w.observe(ms, ok, limit)}, t0.Add(time.Duration(i)*time.Minute), send)
	}
	tick(0, 1500, true, 1000)
	tick(1, 1800, true, 1000)
	if len(sent) != 0 {
		t.Fatalf("دقيقتان بطيئتان لا تكفيان — %v", sent)
	}
	// **ودقيقةٌ سريعةٌ بينهما تبدأ العدَّ من جديد.**
	tick(2, 20, true, 1000)
	tick(3, 1500, true, 1000)
	tick(4, 1500, true, 1000)
	if len(sent) != 0 {
		t.Fatalf("العدُّ لم يُصفَّر بدقيقةٍ سريعة — %v", sent)
	}
	tick(5, 2345, true, 1000)
	if len(sent) != 1 || !strings.HasPrefix(sent[0], "🐢 قاعدة البيانات بطيئة: ٢٬٣٤٥ ميلي ثانية") {
		t.Fatalf("ثلاثُ دقائقَ متتاليةٌ لم تُنذَر — %v", sent)
	}
	// **والقاعدةُ الواقفةُ لا يُحكم على بطئها** — لا عودةَ كاذبة.
	tick(6, 0, false, 1000)
	if len(sent) != 1 {
		t.Fatalf("ما لم يُقَس حُكم عليه — %v", sent)
	}
	tick(7, 30, true, 1000)
	if len(sent) != 2 || sent[1] != "✅ رجع طبيعي: قاعدة البيانات رجعت سريعة." {
		t.Fatalf("العودةُ لم تُقَل — %v", sent)
	}
	// **وصفرُ الإعداد يُطفئه.**
	for i := 8; i < 12; i++ {
		tick(i, 99999, true, 0)
	}
	if len(sent) != 2 {
		t.Fatalf("الإعدادُ صفرٌ وأُنذر — %v", sent)
	}
}

// TestDBSlowCheck_DownIsNotSlow **القاعدةُ الواقفةُ ليست «بطيئة»** — لها تنبيهُها.
func TestDBSlowCheck_DownIsNotSlow(t *testing.T) {
	s := &Server{}
	s.mon.slow.streak = 5
	c := s.dbSlowCheck(context.Background(), true)
	if c.Known || c.Bad || s.mon.slow.streak != 0 {
		t.Fatalf("القاعدةُ واقفةٌ وحُكم على بطئها — %+v streak=%d", c, s.mon.slow.streak)
	}
}

// TestMonitorTick_LedgerViolationAlertsThenResolved
// **خرقٌ حقيقيٌّ في الدفتر** — قيدُ سحبٍ يشير إلى طلبِ سحبٍ لا وجودَ له (FI-01.e):
// يُنبَّه به في أوّل نبضة، **ولا يُكرَّر قبل ساعة**، وبعد إصلاحه وحلول الساعة
// يُقال «رجع طبيعي». **ولا رقمَ هاتفٍ في أيّ رسالة.**
func TestMonitorTick_LedgerViolationAlertsThenResolved(t *testing.T) {
	srv := overviewServer(t)
	out := &captureSender{}
	srv.SetAlertSender(out)
	ctx := context.Background()
	uid := testdb.NewUser(t, srv.pg, "customer")
	var phone string
	if err := srv.pg.QueryRow(ctx, `SELECT phone FROM users WHERE id = $1`, uid).Scan(&phone); err != nil {
		t.Fatalf("الهاتف: %v", err)
	}
	ref := "qa-monitor-missing-payout-" + time.Now().Format("150405.000000")
	var txID int64
	if err := srv.pg.QueryRow(ctx, `
		INSERT INTO wallet_transactions (user_id, amount, kind, ref, note)
		VALUES ($1, -500, 'payout', $2, 'qa monitoring batch 3') RETURNING id`, uid, ref).Scan(&txID); err != nil {
		t.Fatalf("الخرق: %v", err)
	}
	t.Cleanup(func() {
		_, _ = srv.pg.Exec(context.Background(), `DELETE FROM wallet_transactions WHERE id = $1`, txID)
		_ = srv.settings.Set(context.Background(), "monitoring.money_check", true, nil)
	})
	// **في الثالثة فجراً بدمشق** — فلا يتدخّل التقريرُ الصباحيّ.
	t0 := time.Date(2026, 10, 17, 3, 0, 0, 0, platform.Location())
	all := []string{}
	fi := func() []string {
		var r []string
		for _, m := range out.take() {
			all = append(all, m)
			if strings.Contains(m, "FI-01.e") {
				r = append(r, m)
			}
		}
		return r
	}

	srv.monitorTick(ctx, t0)
	m := fi()
	if len(m) != 1 || !strings.Contains(m[0], "⚠️ فحص مالي فشل: FI-01.e") || !strings.Contains(m[0], "(١ سطر)") {
		t.Fatalf("خرقُ الدفتر لم يُنبَّه به — %v", m)
	}
	t.Logf("التنبيه:\n%s", m[0])

	// **أُصلح الخرق — لكنّ الدفترَ لا يُعاد فحصُه قبل الساعة**، فلا شيء.
	if _, err := srv.pg.Exec(ctx, `DELETE FROM wallet_transactions WHERE id = $1`, txID); err != nil {
		t.Fatalf("الإصلاح: %v", err)
	}
	srv.monitorTick(ctx, t0.Add(30*time.Minute))
	if m := fi(); len(m) != 0 {
		t.Fatalf("فحصُ الدفتر أُعيد قبل الساعة — %v", m)
	}
	srv.monitorTick(ctx, t0.Add(61*time.Minute))
	m = fi()
	if len(m) != 1 || m[0] != "✅ رجع طبيعي: الفحص المالي FI-01.e رجع سليم." {
		t.Fatalf("إصلاحُ الخرق لم يُقَل — %v", m)
	}
	for _, msg := range all {
		if strings.Contains(msg, phone) || strings.Contains(msg, uid) {
			t.Fatalf("رسالةٌ فيها هاتفٌ أو معرّف:\n%s", msg)
		}
	}

	// **والإعدادُ يُطفئه**: خرقٌ جديدٌ والفحصُ مُطفأ — لا شيء.
	if err := srv.settings.Set(ctx, "monitoring.money_check", false, nil); err != nil {
		t.Fatalf("الإعداد: %v", err)
	}
	if err := srv.pg.QueryRow(ctx, `
		INSERT INTO wallet_transactions (user_id, amount, kind, ref, note)
		VALUES ($1, -500, 'payout', $2, 'qa monitoring batch 3') RETURNING id`, uid, ref).Scan(&txID); err != nil {
		t.Fatalf("الخرق الثاني: %v", err)
	}
	srv.monitorTick(ctx, t0.Add(3*time.Hour))
	if m := fi(); len(m) != 0 {
		t.Fatalf("الفحصُ مُطفأٌ وأُنذر — %v", m)
	}
}

// TestMonitorTick_MorningReportHasMoneyLine **التقريرُ الصباحيّ فيه سطرُ الحسابات المالية.**
func TestMonitorTick_MorningReportHasMoneyLine(t *testing.T) {
	srv := overviewServer(t)
	out := &captureSender{}
	srv.SetAlertSender(out)
	srv.monitorTick(context.Background(), time.Date(2026, 10, 17, 9, 0, 0, 0, platform.Location()))
	var report string
	for _, m := range out.take() {
		if strings.Contains(m, "صباح الخير") {
			report = m
		}
	}
	if !strings.Contains(report, "الحسابات المالية: ") {
		t.Fatalf("التقريرُ بلا سطر الحسابات المالية:\n%s", report)
	}
}
