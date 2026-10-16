package server

// ══════════════════════════════════════════════════════════════════════
// **الدفترُ والبطء — الراصدُ يرى المالَ والقاعدة** (مراقبةُ المنصّة، الدفعةُ الثالثة ٢٠٢٦-١٠-١٧)
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **خرقُ الدفتر يُرى حين يفتح أحدٌ «صحة الدفتر»** — قيدٌ بلا طلب، رصيدٌ
// لا يطابق حركاتِه: يبقى أيّاماً ولا يعلم به أحد. **والقاعدةُ البطيئة لا
// تُرى أصلاً** — `dbDownFlag` يقول «واقفة» أو «شغّالة»، وما بينهما (ثانيتان
// لكلّ استعلام) يحسّه الزبونُ ثقلاً ولا يصل المالكَ منه شيء.
//
// # وما صار
//
// **١ · الدفترُ كلَّ ساعة** — فحوصُ «صحة الدفتر» بعينها (`ledgerHealth`: مصدرٌ
// واحدٌ للشاشة وللراصد)، **وكلُّ فحصٍ يسقط تنبيهٌ باسمه ورقمه وعددِ صفوفه**:
//
//	⚠️ فحص مالي فشل: FI-06.a — <اسمُ الفحص> (٣ سطر)
//
// **ولا اسمَ زبونٍ ولا رقمَه** — الصفوفُ نفسُها تبقى في اللوحة، وزرُّ «انسخ
// للمطوّر» هناك يحملها لمن يصلح.
//
// **والفحصُ الذي لم يُشغَّل** (خطأُ استعلامٍ أو مهلة) **لم يُقَس** — فلا تنبيهَ
// ولا «رجع طبيعي» كاذب. وهي قاعدةُ الدفعة الأولى نفسُها.
//
// **ولماذا كلَّ ساعةٍ لا كلَّ دقيقة**: خمسةٌ وسبعون استعلاماً على الدفتر كلِّه
// ثقلٌ لا يُحمَّل للقاعدة ستّين مرّةً في الساعة، **ومالٌ انكسر لا يُصلَح في
// دقيقة.** وبين الساعتين تبقى النتيجةُ الأخيرةُ محفوظةً فلا يتبدّل حكمُها.
//
// **٢ · البطءُ كلَّ دقيقة** — `SELECT 1` عبر المجمّع نفسِه (فانتظارُ اتّصالٍ
// فارغٍ يُحسب بطئاً، وهو بطءٌ يحسّه الزبون). **وثلاثُ دقائقَ متتالية** قبل
// التنبيه — فاستعلامٌ ثقيلٌ واحدٌ لا يوقظ المالك.
//
// **والقاعدةُ الواقفة ليست «بطيئة»** — لها تنبيهُها (`db`)، فلا يُقال الاثنان معاً.

import (
	"context"
	"errors"
	"time"
)

const (
	// moneyEvery **الدفترُ يُفحص مرّةً في الساعة.**
	moneyEvery = time.Hour
	// moneyBudget **مهلةُ فحوص الدفتر** — وما لم يتمّ فيها «لم يُقَس».
	moneyBudget = 30 * time.Second
	// dbSlowTicks **كم دقيقةً متتاليةً قبل تنبيه البطء.**
	dbSlowTicks = 3
	// dbPingBudget **مهلةُ `SELECT 1`** — وما تجاوزها بطيءٌ لا واقف.
	dbPingBudget = 10 * time.Second
)

// moneyDue **أحان فحصُ الدفتر؟** — أوّلَ نبضةٍ بعد الإقلاع، ثمّ كلَّ ساعة.
func moneyDue(now, last time.Time) bool {
	return last.IsZero() || now.Sub(last) >= moneyEvery
}

// moneyMonChecks **نتيجةُ «صحة الدفتر» فحوصاً للراصد** — فحصٌ لكلّ ثابت.
//
// **ومفتاحُ كلٍّ منها معرّفُه** (`money:FI-06.a`) — فسقوطُ فحصٍ ثانٍ تنبيهٌ ثانٍ،
// وعودةُ أحدهما لا تُسكت الآخر.
func moneyMonChecks(res []ledgerCheck) []monCheck {
	out := make([]monCheck, 0, len(res))
	for _, c := range res {
		out = append(out, monCheck{
			Key:   "money:" + c.ID,
			Known: c.Error == "",
			Bad:   c.Count > 0,
			Text: "⚠️ فحص مالي فشل: " + c.ID + " — " + c.Name + " (" + arNum(int64(c.Count)) + " سطر)\n" +
				"افتح اللوحة ← الخزينة ← صحة الدفتر، واضغط «انسخ للمطوّر» وابعته للمطوّر.",
			OKText: "✅ رجع طبيعي: الفحص المالي " + c.ID + " رجع سليم.",
		})
	}
	return out
}

// moneyMonitor **فحوصُ الدفتر للنبضة** — تُشغَّل إن حانت، وإلّا فالأخيرةُ المحفوظة.
//
// **ولا شيءَ إن أُطفئ الإعداد أو وقفت القاعدة** — لا يُقاس، فلا يُحكم.
func (s *Server) moneyMonitor(ctx context.Context, now time.Time) []monCheck {
	if s.pg == nil || dbDownFlag.Load() || !s.monSetting(ctx, "monitoring.money_check") {
		return nil
	}
	if moneyDue(now, s.mon.moneyAt) {
		mctx, cancel := context.WithTimeout(ctx, moneyBudget)
		s.mon.money, _ = s.ledgerHealth(mctx)
		cancel()
		s.mon.moneyAt = now
	}
	return moneyMonChecks(s.mon.money)
}

// moneySummary **سطرُ الدفتر في التقرير الصباحيّ.**
func moneySummary(res []ledgerCheck) string {
	failing, errored := 0, 0
	for _, c := range res {
		switch {
		case c.Error != "":
			errored++
		case c.Count > 0:
			failing++
		}
	}
	switch {
	case failing > 0:
		return "⚠️ الحسابات المالية: " + arNum(int64(failing)) + " فحص فاشل"
	case errored > 0:
		return "⚠️ الحسابات المالية: " + arNum(int64(errored)) + " فحص ما قدرت شغّله"
	default:
		return "الحسابات المالية: " + arNum(int64(len(res))) + " فحص سليم"
	}
}

// moneyMorningLine **سطرُ الدفتر للصباح** — من آخر فحصٍ إن كان حديثاً، وإلّا يُشغَّل مرّة.
func (s *Server) moneyMorningLine(ctx context.Context, now time.Time) (string, bool) {
	if s.pg == nil || dbDownFlag.Load() || !s.monSetting(ctx, "monitoring.money_check") {
		return "", false
	}
	if s.mon.moneyAt.IsZero() || now.Sub(s.mon.moneyAt) > 2*moneyEvery {
		mctx, cancel := context.WithTimeout(ctx, moneyBudget)
		s.mon.money, _ = s.ledgerHealth(mctx)
		cancel()
		s.mon.moneyAt = now
	}
	if len(s.mon.money) == 0 {
		return "", false
	}
	return moneySummary(s.mon.money), true
}

// slowWatch **عدّادُ الدقائق البطيئة المتتالية** — وصفرُه صالح.
type slowWatch struct{ streak int }

// observe **قياسُ دقيقةٍ واحدة** ← حكمُ البطء.
//
// `ok` كاذبٌ **إن لم يُقَس** (القاعدةُ واقفة، أو خطأٌ غيرُ المهلة) — فيُصفَّر
// العدّادُ ولا يُحكم. **و`limit` صفرٌ يُطفئه.**
func (w *slowWatch) observe(ms int64, ok bool, limit int64) monCheck {
	c := monCheck{Key: "db_slow",
		OKText: "✅ رجع طبيعي: قاعدة البيانات رجعت سريعة."}
	if !ok || limit <= 0 {
		w.streak = 0
		return c
	}
	c.Known = true
	if ms > limit {
		w.streak++
	} else {
		w.streak = 0
	}
	c.Bad = w.streak >= dbSlowTicks
	c.Text = "🐢 قاعدة البيانات بطيئة: " + arNum(ms) + " ميلي ثانية\n" +
		"صرلها " + arNum(int64(w.streak)) + " دقايق أبطأ من " + arNum(limit) +
		" ميلي ثانية. المنصة رح تكون تقيلة ع الكل."
	return c
}

// dbSlowCheck **فحصُ البطء للنبضة.**
func (s *Server) dbSlowCheck(ctx context.Context, dbDown bool) monCheck {
	limit := s.monInt(ctx, "monitoring.db_slow_ms", 1000)
	if s.pg == nil || dbDown || limit <= 0 {
		return s.mon.slow.observe(0, false, limit)
	}
	ms, ok := s.pingDB(ctx)
	return s.mon.slow.observe(ms, ok, limit)
}

// pingDB **كم أخذ `SELECT 1` بالميلي ثانية** — والمهلةُ بطءٌ لا عطل.
func (s *Server) pingDB(ctx context.Context) (int64, bool) {
	pctx, cancel := context.WithTimeout(ctx, dbPingBudget)
	defer cancel()
	start := time.Now()
	var one int
	err := s.pg.QueryRow(pctx, `SELECT 1`).Scan(&one)
	ms := time.Since(start).Milliseconds()
	switch {
	case err == nil:
		return ms, true
	case errors.Is(pctx.Err(), context.DeadlineExceeded) && ctx.Err() == nil:
		return ms, true
	default:
		return 0, false
	}
}
