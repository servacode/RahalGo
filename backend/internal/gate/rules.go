package gate

import (
	"fmt"
	"sort"
	"strings"
)

// ══════════════════════════════════════════════════════════════════════
// **سياسةُ المنع — مكتوبةٌ مرّةً واحدةً ومقروءة**
// ══════════════════════════════════════════════════════════════════════
//
// **البند ٤ حرفيّاً**: **المانعُ يمنع، والحرجُ يمنع ما لم يوجد استثناءُ
// مالكٍ صريح.** **والعالي فما دون تحذيرٌ لا منع.**
//
// **ولا تُخترَع استثناءات** — **ولا تُخترَع تصعيدات.** جرّبتُ تصعيدَ كلِّ
// ما يحمله تدفّقٌ مانع، **فبلغ ستّةً وعشرين بنداً بينها متوسّطات** —
// **وقائمةُ موانعَ فيها متوسّطاتٌ لا تدلُّ على شيء.**
func blockingBySeverity(s Severity) bool { return s == Blocker || s == Critical }

// catOfDomain تصنيفُ سجلٍّ من عمود «المجال» في السجلّ المجمَّد.
//
// **والمجالُ مقروءٌ لا مُعلَن** — أُضيف إلى `TEST_TRUTH` في `P-10`.
func catOfDomain(domain string) Category {
	d := domain
	has := func(ss ...string) bool {
		for _, s := range ss {
			if strings.Contains(d, s) {
				return true
			}
		}
		return false
	}
	switch {
	case has("تزامن"):
		return CatConcurrency
	case has("خصوصيّة", "خصوصية", "تقليلُ بيانات"):
		return CatPrivacy
	case has("تدقيق"):
		return CatAudit
	case has("الملاحة"):
		return CatNavigation
	case has("بطّاريّة", "بطارية"):
		return CatBackground
	case has("مال"):
		return CatFinancial
	case has("أمن", "جلسة", "صلاحيّات", "صلاحيات", "الهويّة", "الهوية"):
		return CatSecurity
	case has("لحظيّ", "إشعار"):
		return CatNotify
	case has("أداء"):
		return CatPerf
	case has("توفّر", "عمليّات", "تشغيليّ", "رصد"):
		return CatOps
	case has("الطلب", "الطلبات", "حالة", "نزاهة"):
		return CatOrder
	}
	return CatCoverage
}

// catOfGap تصنيفُ فجوةٍ من عنوانها — **والعناوينُ وصفيّةٌ لا رمزيّة.**
func catOfGap(g TruthGap) Category {
	t := g.Title
	has := func(ss ...string) bool {
		for _, s := range ss {
			if strings.Contains(t, s) {
				return true
			}
		}
		return false
	}
	switch {
	case has("عمولة", "رصيد", "استرداد", "مكافأة", "احتساب", "طبقات", "التفعيل", "نسبة"):
		return CatFinancial
	case has("التدقيق"):
		return CatAudit
	case has("media", "سرد"):
		return CatMedia
	case has("إشعار", "الإشعار", "توجيه"):
		return CatNotify
	case has("ازدواج", "تعليق", "تدخّل", "موقوف", "نقل"):
		return CatOrder
	}
	return CatCoverage
}

// BuildRules يبني قواعدَ البوّابة كلَّها من الدليل.
//
// **ولا قاعدةَ بلا مصدرِ عقدٍ ولا بلا دليلٍ مطلوب.**
func BuildRules(e *Evidence) []Rule {
	var rules []Rule
	rules = append(rules, defectRules(e)...)
	rules = append(rules, gapRules(e)...)
	rules = append(rules, riskRules(e)...)
	rules = append(rules, financialRules(e)...)
	rules = append(rules, failureRules(e)...)
	rules = append(rules, freshnessRules(e)...)
	rules = append(rules, androidRules(e)...)
	rules = append(rules, externalRules()...)
	rules = append(rules, opsRules(e)...)
	rules = append(rules, coverageRules(e)...)
	rules = append(rules, traceabilityRules(e)...)
	sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })
	return rules
}

// ── العيوب (البند ٦) ─────────────────────────────────────────────

// defectRules قاعدةٌ لكلّ عيبٍ مجمَّد.
//
// # والمبدأُ الذي لا يُخالَف
//
//	DEFECT EXISTS + TEST EXPECTED FAILS = STILL OPEN
//
// **والعيبُ لا يُغلَق لأنّ له اختباراً** — **الاختبارُ يثبته لا يصلحه.**
func defectRules(e *Evidence) []Rule {
	var out []Rule
	for _, d := range e.Truth.Defects {
		sev := Severity(d.Severity)
		if sev == "" {
			sev = Unknown
		}
		st, why := ExpectedFail, ""
		switch {
		case len(d.Tests) == 0:
			st = NotRun
			why = "**لا اختبارَ انحدارٍ يحرسه** — فلا دليلَ على حاله اليومَ"
		case d.Fixed != "":
			// ══════════════════════════════════════════════════════
			// **وعيبٌ أُصلح وله حرّاسٌ لا يبقى مانعاً**
			// ══════════════════════════════════════════════════════
			//
			// **وكانت البوّابةُ لا تقرأ دليلَ الإصلاح للعيوب** —
			// **تقرؤه للفجوات وحدَها** (`gapRules`). **فعيبٌ أُغلق
			// بدليلٍ وحرّاسٍ يبقى `EXPECTED_FAIL` إلى الأبد.**
			//
			// **والشرطان معاً**: **دليلٌ وحارس.** **ومن كتب دليلاً
			// وحذف الحارسَ أغلقه بالكلام.**
			st = Pass
			why = "**أُصلح** — " + d.Fixed
		default:
			res, ev := e.ResultOf(d.ID)
			why = fmt.Sprintf("**قائمٌ ومُثبَت** — %d اختباراً يسقط عمداً", len(d.Tests))
			if ev != "" {
				why = fmt.Sprintf("**%s** — %s", res, ev)
			}
		}
		cat := catOfDomain(d.Domain)
		out = append(out, Rule{
			ID:               "GATE-" + d.ID,
			Category:         cat,
			Requirement:      "إغلاقُ العيب: " + d.Title,
			SourceContract:   "DEFECT REGISTER · `FINAL_STATIC_CLOSEOUT.md` — المجال: " + d.Domain,
			Severity:         sev,
			RequiredEvidence: "إصلاحٌ في المنتج ثمّ اختبارُ انحدارٍ ينجح بعد أن كان يسقط",
			Status:           st,
			Blocking:         blockingBySeverity(sev) && st != Pass,
			Reason:           why,
			Registers:        []string{d.ID},
			Tests:            d.Tests,
			WaiverPolicy:     waiverOf(cat, sev),
		})
	}
	return out
}

// ── الفجوات (البندان ٥ و٨) ───────────────────────────────────────

func gapRules(e *Evidence) []Rule {
	var out []Rule
	for _, g := range e.Truth.Gaps {
		sev := Severity(g.Severity)
		st := NotImplemented
		why := "**عقدٌ ناقصٌ لم يُبنَ بعد**"
		if len(g.Tests) > 0 {
			st = ExpectedFail
			why = fmt.Sprintf("**مُثبَتٌ بـ%d اختباراً يسقط عمداً**", len(g.Tests))
		}
		if res, ev := e.ResultOf(g.ID); ev != "" {
			why = fmt.Sprintf("**%s** — %s", res, ev)
		}
		// ══════════════════════════════════════════════════════════
		// **وفجوةٌ أُصلحت بدليلٍ وحارسٍ تُغلَق**
		// ══════════════════════════════════════════════════════════
		//
		// **والشرطان معاً لا أحدُهما**: **دليلٌ مكتوبٌ في الحقيقة،
		// وحارسُ انحدارٍ دائمٌ مربوطٌ بها.** **ومن أغلقها بدليلٍ بلا
		// حارسٍ أغلقها بالكلام** — **ومن أغلقها بحارسٍ بلا دليلٍ لم
		// يقل لأحدٍ لماذا.**
		if g.Fixed != "" && len(g.Tests) > 0 {
			st = Pass
			why = "**أُصلحت** — " + g.Fixed
		}
		// **وفجوةٌ جذرُها جذرُ أخرى تُقرأ ولا تُعَدّ** — **سببٌ واحدٌ
		// لا مانعان مصطنعان** (البند ٧).
		super := ""
		if g.PartOf != "" {
			super = "GATE-" + g.PartOf
			why = fmt.Sprintf("**نطاقٌ من `%s` — جذرُهما واحد** · %s",
				g.PartOf, why)
		}
		cat := catOfGap(g)
		r := Rule{
			SupersededBy:     super,
			ID:               "GATE-" + g.ID,
			Category:         cat,
			Requirement:      "سدُّ الفجوة: " + g.Title,
			SourceContract:   "CONTRACT GAP REGISTER · شدّةُ `TQ-1`",
			Severity:         sev,
			RequiredEvidence: "بناءُ العقد ثمّ اختبارٌ يثبته",
			Status:           st,
			Blocking:         blockingBySeverity(sev) && st != Pass,
			Reason:           why,
			Registers:        []string{g.ID},
			Tests:            g.Tests,
			WaiverPolicy:     waiverOf(cat, sev),
		}
		// ── الفجوةُ النائمة (البند ٥) ──────────────────────────
		//
		// **ولا تُعدُّ آمنةً لأنّ مفتاحَها مطفأ** — **الإدارةُ تشعله بعد
		// الإطلاق بضغطة، ولا قفلَ إداريٌّ يمنعها.**
		if g.WokenBy != "" {
			r.Latent, r.WokenBy = true, g.WokenBy
			r.Requirement += fmt.Sprintf(" (نائمةٌ يوقظها `%s`)", g.WokenBy)
			r.Reason += fmt.Sprintf(" · **ويوقظها `%s` بضغطةٍ من الإدارة بعد الإطلاق** — "+
				"**ولا قفلَ إداريٌّ قائم**، فبقاؤها نائمةً ليس أماناً بل صدفةَ ضبط", g.WokenBy)
			if blockingBySeverity(sev) {
				// **البند ٥ خيارُ (ب)**: لا قفل ⇒ الإطلاقُ يُمنع.
				r.Blocking = true
			}
		}
		out = append(out, r)
	}
	return out
}

// ── المخاطر (البند ٧) ────────────────────────────────────────────

func riskRules(e *Evidence) []Rule {
	var out []Rule
	for _, k := range e.Truth.Risks {
		sev := e.FlowSeverityOf(k.ID)
		res, ev := e.ResultOf(k.ID)
		st, why := NotRun, "**لم يُحسَم** — لا اختبارَ ولا دليلٌ يثبته أو ينفيه"
		verdict := "UNVALIDATED"
		switch {
		case res == "RISK_CONFIRMED":
			st, verdict = ExpectedFail, "CONFIRMED"
			why = "**تأكّد** — " + ev
		// ══════════════════════════════════════════════════════════
		// **وعقدُ حدثٍ صامدٌ مقيسٌ حسمٌ** — دورةُ ٢٣
		// ══════════════════════════════════════════════════════════
		//
		// **`HELD` تعني «قِيس وصمد»** — كـ`PASS` في مصفوفةٍ أخرى.
		// **وكانت تُقرأ «لم يُحسَم» فيبقى الخطرُ مانعاً إلى الأبد
		// رغم دليلٍ يخصُّه.**
		//
		// **وشرطُ سلامتها أن يكون الدليلُ منسوباً إلى سجلٍّ يقيسه**
		// — وهو ما صُحّح في `EV-11`.
		//
		// **والأسوأُ يغلب دائماً**: صفٌّ واحدٌ يقول `RISK_CONFIRMED`
		// يسبق كلَّ صامد.
		case res == "PASS" || res == "PROVEN" || res == "HELD":
			st, verdict = Pass, "DISPROVEN"
			why = "**انتفى بالقياس** — " + ev
		case ev != "":
			st = ExpectedFail
			why = fmt.Sprintf("**%s** — %s", res, ev)
		case len(k.Tests) > 0:
			st = ExpectedFail
			why = fmt.Sprintf("**له %d اختباراً ولم يُحسَم إلى نفيٍ بعد**", len(k.Tests))
		}
		cat := catOfDomain(k.Domain)
		r := Rule{
			ID:               "GATE-" + k.ID,
			Category:         cat,
			Requirement:      "حسمُ الخطر: " + k.Title,
			SourceContract:   "RISK REGISTER · المجال: " + k.Domain + " · الحكم: " + verdict,
			Severity:         sev,
			RequiredEvidence: "اختبارٌ يثبته فيصير عيباً، أو ينفيه فيُغلَق",
			Status:           st,
			Blocking:         blockingBySeverity(sev) && st != Pass,
			Reason:           why,
			Registers:        []string{k.ID},
			Tests:            k.Tests,
			WaiverPolicy:     waiverOf(cat, sev),
		}
		// **والخطرُ الذي صار عيباً لا يُحتسب مانعاً ثانياً** (البند ٧).
		if d, ok := Supersede[k.ID]; ok {
			r.SupersededBy = d
			r.SourceContract += fmt.Sprintf(" · **تأكّد فصار `%s`** — سببٌ واحدٌ لا مانعان", d)
		}
		out = append(out, r)
	}
	return out
}

// ── المال (البند ١٣) ─────────────────────────────────────────────

// financialRules قاعدةٌ لكلّ ثابتٍ ماليٍّ غيرِ مُثبَت.
//
// **والثوابتُ الثمانيةُ والثلاثون التي تُثبَت اليومَ تمرّ** — **وما لم
// يُبنَ منها لا يمرّ بالسكوت.**
func financialRules(e *Evidence) []Rule {
	var out []Rule
	ids := make([]string, 0, len(e.FinChecks))
	for id := range e.FinChecks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	notImpl := 0
	for _, id := range ids {
		if e.FinChecks[id] != "PROVABLE_NOW" {
			notImpl++
		}
	}
	out = append(out, Rule{
		ID:       "GATE-FIN-01",
		Category: CatFinancial,
		Requirement: "ثوابتُ المال كلُّها مُثبَتةٌ على المرشَّح — مصالحةُ المحفظة · " +
			"نقدُ السائق · حفظُ اقتصاد الطلب · اللقطة · نزاهةُ الاسترداد · " +
			"عكسُ عمولة المندوب · طبقاتُ الدفع · الخزينة",
		SourceContract:   "`P-4` · محرّكُ الثوابت الماليّة — ٤٦ ثابتاً في ١٢ عائلة",
		Severity:         Blocker,
		RequiredEvidence: "تشغيلُ `internal/fininv` على قاعدةِ المرشَّح بلا خرقٍ واحد",
		Status:           NotRun,
		Blocking:         true,
		Reason: fmt.Sprintf("**%d ثابتاً من %d لم يُبنَ أو أُجِّل** — "+
			"**و`38` ثابتاً تُثبَت اليوم لا تُغني عمّا لم يُبنَ**، "+
			"**ولم يُسجَّل تشغيلٌ على مرشَّحٍ ثابت**", notImpl, len(ids)),
		WaiverPolicy: WaiverForbidden,
	})
	return out
}

// ── تعافي الفشل (البند ٢٨) ───────────────────────────────────────

// failureRules **أيُّ عمليّةٍ جزئيّةٍ تترك مالاً خطأً أو طلباً عالقاً.**
//
// **والدليلُ من `P-6`** — تسعةُ تدفّقاتِ فشلٍ جزئيٍّ مقيسة.
func failureRules(e *Evidence) []Rule {
	bad, invisible := 0, 0
	var worst []string
	for _, r := range e.Rows {
		if r.Matrix != "FAILURE" {
			continue
		}
		switch r.Result {
		case "DEFECT_REPRODUCED", "RISK_CONFIRMED", "EXPECTED_FAIL", "UNPROVEN", "PARTIAL":
			bad++
			worst = append(worst, fmt.Sprintf("%s:%s", r.ID, r.Result))
		}
	}
	st, why := Pass, "**كلُّ فشلٍ جزئيٍّ يتعافى أو يُرى**"
	if bad > 0 {
		st = ExpectedFail
		why = fmt.Sprintf("**%d تدفّقَ فشلٍ جزئيٍّ يترك أثراً غيرَ متعافٍ**: %v — "+
			"**ومنها ما لا يراه زبونٌ ولا إدارة**", bad, worst)
	}
	_ = invisible
	return []Rule{{
		ID: "GATE-FAIL-01", Category: CatFailure,
		Requirement:      "لا عمليّةٌ جزئيّةٌ تترك مالاً خطأً ولا طلباً عالقاً ولا متجراً مكرَّراً ولا عمليّةً خفيّة",
		SourceContract:   "`P-6` · `FAILURE_INJECTION_MATRIX` — تسعةُ تدفّقاتٍ جزئيّة",
		Severity:         Blocker,
		RequiredEvidence: "كلُّ تدفّقِ فشلٍ جزئيٍّ إمّا يتعافى ذاتيّاً وإمّا يظهر للإدارة",
		Status:           st, Blocking: true, Reason: why,
		WaiverPolicy: WaiverForbidden,
	}}
}

// ── طزاجةُ الدليل وهويّةُ المرشَّح (البندان ٣٣ و٣٤) ───────────────

// freshnessRules **ولا بوّابةَ نهائيّةٌ على شجرةِ عملٍ متغيّرة.**
func freshnessRules(e *Evidence) []Rule {
	st, why := NotRun, "**يُحكَم عند التشغيل** — تُقارَن الشجرةُ بالالتزام"
	return []Rule{{
		ID: "GATE-FRESH-01", Category: CatFreshness,
		Requirement:      "المرشَّحُ ثابتٌ ودليلُه من بنائه هو",
		SourceContract:   "`P-10` البندان ٣٣ و٣٤",
		Severity:         Blocker,
		RequiredEvidence: "التزامٌ نظيفٌ · بصماتُ بناءٍ مسجَّلة · دليلٌ مأخوذٌ من المرشَّح نفسِه",
		Status:           st, Blocking: true, Reason: why,
		WaiverPolicy: WaiverOwnerOnly,
	}}
}

// ── أندرويد (البند ١٨) ───────────────────────────────────────────

func androidRules(e *Evidence) []Rule {
	notRun := e.AndroidCounts["NotRunN"]
	cases := e.AndroidCounts["Cases"]
	avail := e.AndroidCounts["DevicesAvailable"]
	return []Rule{{
		ID:               "GATE-AND-02",
		Category:         CatAndroid,
		Requirement:      "تنفيذُ مصفوفة أندرويد كاملةً على أجهزةٍ حقيقيّة",
		SourceContract:   "`P-8` · `ANDROID_TEST_MATRIX` · `TQ-2`",
		Severity:         Blocker,
		RequiredEvidence: "سجلُّ تنفيذٍ لكلّ بندٍ على الجهاز المرجعيّ",
		Status:           NotRun,
		Blocking:         true,
		Reason: fmt.Sprintf("**%d بنداً من %d لم يُشغَّل · وأجهزةٌ متاحةٌ = %d** — "+
			"**والمِسنَدُ مكتملٌ والقبولُ لم يبدأ**", notRun, cases, avail),
		WaiverPolicy: WaiverOwnerOnly,
	}}
}

// ── ما لا يُثبَت محلّيّاً ────────────────────────────────────────

func externalRules() []Rule {
	var out []Rule
	for _, r := range ExternalReqs {
		out = append(out, Rule{
			ID: r.ID, Category: r.Category, Requirement: r.Requirement,
			SourceContract: r.Source, Severity: r.Severity,
			RequiredEvidence: r.Evidence, Status: r.Status,
			Blocking: r.Blocking, Reason: r.Why, WaiverPolicy: r.Waiver,
		})
	}
	return out
}

// ── التشغيل (البندان ٣٢ و٢٣) ─────────────────────────────────────

func opsRules(e *Evidence) []Rule {
	var out []Rule

	// **`OPS1` — الصحّةُ والجهوزيّة.**
	var missing []string
	covered := map[string]bool{}
	for _, d := range e.HealthCovers {
		covered[d] = true
	}
	for _, d := range e.CriticalDeps {
		if !covered[d] {
			missing = append(missing, d)
		}
	}
	st, why := Pass, "**يغطّي التبعيّاتِ الحرجةَ كلَّها**"
	if len(missing) > 0 || !e.ReadinessEndpoint {
		st = Fail
		why = fmt.Sprintf("**`/healthz` يفحص %v ولا يفحص %v** · بابُ جهوزيّةٍ منفصلٌ = %v — "+
			"**فالموجّهُ يرسل الحملَ إلى عُقدةٍ لم تجهز**",
			e.HealthCovers, missing, e.ReadinessEndpoint)
	}
	out = append(out, Rule{
		ID: "GATE-OPS-01", Category: CatHealth,
		Requirement:      "الصحّةُ والجهوزيّةُ تغطّيان التبعيّاتِ الحرجة",
		SourceContract:   "`OPS1` · دَينُ التشغيل",
		Severity:         High,
		RequiredEvidence: "بابُ صحّةٍ يفحص كلَّ تبعيّةٍ حرجة، وبابُ جهوزيّةٍ يمنع استقبالَ الحمل قبل الاستعداد",
		Status:           st, Blocking: false, Reason: why,
		WaiverPolicy: WaiverOwnerOnly,
	})

	// **جهوزيّةُ الإعدادات (البند ٢٣)** — الافتراضاتُ غيرُ الآمنة.
	unsafe := 0
	var moneyNoTest []string
	for _, s := range e.Truth.Settings {
		if s.Money && len(s.Tests) == 0 {
			unsafe++
			if len(moneyNoTest) < 8 {
				moneyNoTest = append(moneyNoTest, s.Key)
			}
		}
	}
	out = append(out, Rule{
		ID: "GATE-SET-01", Category: CatSettings,
		Requirement:      "لا مفتاحَ ماليٌّ يُغيِّر السلوكَ بلا اختبارٍ يحرسه",
		SourceContract:   "`TEST_TRUTH` · الإعداداتُ المُغيِّرةُ للسلوك",
		Severity:         High,
		RequiredEvidence: "اختبارٌ لكلّ مفتاحٍ ماليٍّ يثبت أثرَه على الحساب",
		Status: func() State {
			if unsafe == 0 {
				return Pass
			}
			return NotRun
		}(),
		Blocking: false,
		Reason: fmt.Sprintf("**%d مفتاحاً ماليّاً بلا اختبار** — منها %v",
			unsafe, moneyNoTest),
		WaiverPolicy: WaiverOwnerOnly,
	})
	return out
}

// ── التغطيةُ والاختباراتُ المعروفة (البنود ١٠ · ١١ · ١٢ · ٤٨) ────

func coverageRules(e *Evidence) []Rule {
	var out []Rule

	orphan, mapped, infra := 0, 0, 0
	for _, t := range e.Truth.Tests {
		switch {
		case t.Orphan:
			orphan++
		case t.Purpose != "" && strings.Contains(t.Purpose, "GENERATOR"):
			infra++
		default:
			mapped++
		}
	}

	// **البند ٤٨**: اليتيمُ لا يُسقط الإطلاق — **العقدُ بلا دليلٍ يُسقطه.**
	uncovered := 0
	var worst []string
	for _, f := range e.Truth.Flows {
		if len(f.Tests) == 0 {
			uncovered++
			worst = append(worst, f.ID)
		}
	}
	st := Pass
	why := fmt.Sprintf("**كلُّ تدفّقٍ له دليل** — %d مربوطاً · %d يتيماً · %d بنيةً تحتيّة",
		mapped, orphan, infra)
	if uncovered > 0 {
		st = Fail
		why = fmt.Sprintf("**%d تدفّقاً بلا اختبارٍ واحد**: %v", uncovered, worst)
	}
	out = append(out, Rule{
		ID: "GATE-COV-01", Category: CatCoverage,
		Requirement:      "كلُّ سلوكٍ مطلوبٍ له دليلٌ مربوط",
		SourceContract:   "`P-2` · `TEST_TRUTH`",
		Severity:         High,
		RequiredEvidence: "اختبارٌ مربوطٌ لكلّ تدفّقٍ عابرٍ للأنظمة",
		Status:           st, Blocking: false, Reason: why,
		WaiverPolicy: WaiverOwnerOnly,
	})

	// **الاختباراتُ المعروفةُ الحال (البندان ١١ و١٢).**
	for _, kt := range KnownTests {
		st, blocking := Pass, false
		why := "**" + kt.Verdict + "** — " + kt.Why + " · " + kt.Fix
		if kt.Unguarded {
			st = Fail
			why += " · **والعقدُ الذي كان يحرسه صار بلا حارس**: " + kt.Contract
		}
		out = append(out, Rule{
			ID: "GATE-TST-" + shortName(kt.Name), Category: CatTestInfra,
			Requirement:      "الاختبارُ المعروفُ الحال لا يدخل الحزمةَ الإلزاميّةَ وهو كاذب: " + kt.Name,
			SourceContract:   "`P-10` البندان ١١ و١٢ · مصالحةٌ مقيسة",
			Severity:         Medium,
			RequiredEvidence: "إمّا تصحيحُ الاختبار وإمّا إخراجُه من الحزمة الإلزاميّة مع توثيق السبب",
			Status:           st, Blocking: blocking, Reason: why,
			WaiverPolicy: WaiverOwnerOnly,
		})
	}
	return out
}

func shortName(n string) string {
	n = strings.TrimPrefix(n, "Test")
	if i := strings.Index(n, "_"); i > 0 {
		n = n[:i]
	}
	if len(n) > 18 {
		n = n[:18]
	}
	return strings.ToUpper(n)
}

// ── دَينُ التتبّع (البند ٩) ───────────────────────────────────────

// traceabilityRules **ما لا يبلغه رسمُ الأثر ولا يحرسه اختبار.**
//
// # ولماذا لا يكفي السقوطُ الآمن
//
// **`SAFE FALLBACK` يحمي اختيارَ الاختبارات، ولا يحوّل تتبّعاً مفقوداً
// إلى تغطيةٍ مُثبَتة** (البند ٩ حرفيّاً). **فالحزمةُ كلُّها قد تُشغَّل
// ولا اختبارَ فيها يمسُّ `D3` أصلاً.**
func traceabilityRules(e *Evidence) []Rule {
	var unmappedBlocking, unmappedAll []string

	check := func(id string, sev Severity, tests []string) {
		if len(tests) > 0 || len(e.EvidenceFor(id)) > 0 {
			return
		}
		unmappedAll = append(unmappedAll, id)
		if blockingBySeverity(sev) {
			unmappedBlocking = append(unmappedBlocking, id)
		}
	}
	for _, d := range e.Truth.Defects {
		check(d.ID, Severity(d.Severity), d.Tests)
	}
	for _, g := range e.Truth.Gaps {
		check(g.ID, Severity(g.Severity), g.Tests)
	}
	for _, k := range e.Truth.Risks {
		check(k.ID, e.FlowSeverityOf(k.ID), k.Tests)
	}
	sort.Strings(unmappedAll)
	sort.Strings(unmappedBlocking)

	st, blocking := Pass, false
	why := "**كلُّ سجلٍّ مانعٍ له دليلٌ مربوط**"
	if len(unmappedBlocking) > 0 {
		st, blocking = Fail, true
		why = fmt.Sprintf("**%d سجلّاً مانعاً أو حرجاً بلا اختبارٍ ولا دليلِ مصفوفة**: %v — "+
			"**والسقوطُ الآمنُ لا يجعل هذا تغطيةً**",
			len(unmappedBlocking), unmappedBlocking)
	}
	rules := []Rule{{
		ID: "GATE-TRC-01", Category: CatTrace,
		Requirement:      "كلُّ سجلٍّ مانعٍ للإطلاق مربوطٌ بدليلٍ يخصُّه",
		SourceContract:   "`P-9` · `P-10` البند ٩",
		Severity:         Blocker,
		RequiredEvidence: "اختبارٌ أو صفُّ مصفوفةٍ يشير إلى السجلّ بعينه",
		Status:           st, Blocking: blocking, Reason: why,
		Registers:    unmappedBlocking,
		WaiverPolicy: WaiverForbidden,
	}}

	if len(unmappedAll) > 0 {
		rules = append(rules, Rule{
			ID: "GATE-TRC-02", Category: CatTrace,
			Requirement:      "تتبّعٌ كاملٌ — لا سجلَّ بلا دليل",
			SourceContract:   "`P-10` البند ٩ · `FULL TRACEABILITY`",
			Severity:         High,
			RequiredEvidence: "ربطُ كلّ سجلٍّ باختبارٍ أو صفِّ مصفوفة",
			Status:           Fail, Blocking: false,
			Reason: fmt.Sprintf("**%d سجلّاً بلا دليلٍ يخصُّه**: %v — "+
				"**فـ`FULL TRACEABILITY` لا يمرّ**", len(unmappedAll), unmappedAll),
			Registers:    unmappedAll,
			WaiverPolicy: WaiverOwnerOnly,
		})
	}
	return rules
}

// waiverOf سياسةُ التنازل — **والمالُ والخصوصيّةُ والأمنُ في شدّةٍ عاليةٍ
// لا يُتنازَل عنها** (البند ٣٧).
func waiverOf(c Category, s Severity) WaiverPolicy {
	switch c {
	case CatFinancial, CatPrivacy, CatSecurity, CatMedia, CatAudit:
		if blockingBySeverity(s) {
			return WaiverForbidden
		}
	}
	return WaiverOwnerOnly
}
