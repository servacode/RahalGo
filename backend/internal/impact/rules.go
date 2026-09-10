package impact

import (
	"path"
	"strings"
)

// Classify يصنّف ملفّاً — **بالمسار ثمّ بالمحتوى حيث يلزم** (البند ٣).
//
// **ولا يُترَك ملفٌّ بلا تصنيف**: ما لم يُعرَف يصير `UNKNOWN`، **و`UNKNOWN`
// يُوسّع ولا يُهمَل** (البند ١٠).
func Classify(p string) File {
	p = strings.ReplaceAll(p, "\\", "/")
	f := File{Path: p}

	switch {
	case strings.HasSuffix(p, "_test.go") || strings.Contains(p, "/src/test/") ||
		strings.Contains(p, "/src/androidTest/"):
		// **واختبارُ البنية يُميَّز عن اختبار الميزة** (البندان ٢٣ و٢٤).
		if isTestInfraPath(p) {
			f.Category, f.Why = CatTestInfra, "اختبارٌ في حزمةِ بنيةٍ تحتيّة"
		} else {
			f.Category, f.Why = CatTest, "ملفُّ اختبار"
		}
	case strings.HasPrefix(p, "backend/internal/migrate/migrations/"):
		f.Category, f.Why = CatMigration, "هجرةُ قاعدةِ بيانات"
	case strings.HasPrefix(p, "backend/internal/settings/"):
		f.Category, f.Why = CatSettings, "معجمُ الإعدادات"
	case isTestInfraPath(p):
		f.Category, f.Why = CatTestInfra, "حزمةُ بنيةٍ تحتيّةٍ للاختبار"
	case strings.HasPrefix(p, "backend/"):
		f.Category, f.Why = CatBackend, "شيفرةُ المحرّك"
	case strings.HasPrefix(p, "web/"):
		f.Category, f.Why = CatAdminWeb, "لوحةُ الإدارة"
	case strings.HasPrefix(p, "mobile/app-customer/"):
		f.Category, f.Why = CatAndCust, "تطبيقُ الزبون"
	case strings.HasPrefix(p, "mobile/app-driver/"):
		f.Category, f.Why = CatAndDriver, "تطبيقُ السائق"
	case strings.HasPrefix(p, "mobile/app-merchant/"):
		f.Category, f.Why = CatAndMerch, "تطبيقُ المتجر"
	case strings.HasPrefix(p, "mobile/app-rep/"):
		f.Category, f.Why = CatAndRep, "تطبيقُ المندوب"
	case strings.HasPrefix(p, "mobile/"):
		f.Category, f.Why = CatAndShared, "وحدةٌ مشتركةٌ في أندرويد"
	case strings.HasPrefix(p, "deploy/") || strings.HasPrefix(p, "scripts/") ||
		path.Base(p) == "docker-compose.yml" || strings.HasSuffix(p, "Dockerfile") ||
		strings.Contains(p, "nginx"):
		f.Category, f.Why = CatDeploy, "نشرٌ أو تشغيل"
	// ── ملفّاتُ المستودع نفسِه ────────────────────────────────
	//
	// **ولا تمسّ سلوكاً** — `.gitignore` و`.gitattributes` و`LICENSE`.
	//
	// **وكانت تُصنَّف مجهولةً فتُسقط الحكمَ إلى التوسيع الآمن كلَّه**
	// (قِيس في `P-0`: تبديلُ سطرٍ في `.gitignore` طلب الحزمةَ كاملةً
	// بثقةٍ منخفضة). **والتوسيعُ الآمنُ صحيحٌ للمجهول، وهذا معلوم.**
	case path.Base(p) == ".gitignore" || path.Base(p) == ".gitattributes" ||
		path.Base(p) == "LICENSE" || path.Base(p) == ".editorconfig":
		f.Category, f.Why = CatDocs, "ملفُّ مستودعٍ لا يمسّ سلوكاً"
	// ── عقدُ الـAPI ───────────────────────────────────────────
	//
	// **مولَّدٌ من الموجّه بـ`cmd/apidoc`** — **فتبدّلُه أثرُ تبدّلٍ لا
	// سببُه**، **وحارسُه `TestContractIsCurrent` يكشف شيخوخته.**
	//
	// **وكان مجهولاً فيُسقط الحكمَ إلى التوسيع الآمن** (قِيس في `P-0`).
	case p == "api/contract.json":
		f.Category, f.Why = CatTruthDocs, "**عقدُ الـAPI مولَّدٌ** — حارسُه يكشف شيخوخته"
	case isTruthDoc(p):
		f.Category, f.Why = CatTruthDocs, "**حقيقةُ منتجٍ لا وثيقةٌ عاديّة**"
	case strings.HasSuffix(p, ".md") || strings.HasPrefix(p, "docs/"):
		f.Category, f.Why = CatDocs, "توثيقٌ عاديّ"
	default:
		f.Category, f.Why = CatUnknown, "**لا قاعدةَ تصنّفه** — يُوسَّع ولا يُهمَل"
	}
	return f
}

// isTestInfraPath حزمُ البنية التحتيّة للاختبار (البند ٢٤).
func isTestInfraPath(p string) bool {
	for _, d := range []string{
		"backend/internal/qa/", "backend/internal/testtruth/", "backend/internal/testdb/",
		"backend/internal/fininv/", "backend/internal/racemap/", "backend/internal/failmap/",
		"backend/internal/eventmap/", "backend/internal/androidmap/", "backend/internal/impact/",
		"backend/cmd/testtruth/", "backend/cmd/fininvdoc/", "backend/cmd/moneycheck/",
		"backend/cmd/testimpact/", "mobile/testkit/",
	} {
		if strings.HasPrefix(p, d) {
			return true
		}
	}
	return false
}

// isTruthDoc **حقيقةُ المنتج** — تبديلُها يستوجب إعادةَ توليدٍ وحرّاسَ انحراف
// (البند ٢٥).
func isTruthDoc(p string) bool {
	for _, d := range []string{
		"docs/testing/FINAL_STATIC_CLOSEOUT.md",
		"docs/testing/CROSS_SYSTEM_INTERACTION_CLOSURE.md",
		"docs/testing/system/",
		"docs/TRUTH.md", "docs/GROUND-RULES.md",
	} {
		if strings.HasPrefix(p, d) {
			return true
		}
	}
	return false
}

// domainRule قاعدةُ نطاقٍ في المحرّك — **مسارٌ يُطلق أثراً مسمّىً.**
type domainRule struct {
	// Match بادئةٌ أو جزءٌ من المسار.
	Match []string
	// Name اسمُ القاعدة في التقرير.
	Name string
	// Flows التدفّقاتُ المتأثّرة.
	Flows []string
	// Apps التطبيقاتُ المتأثّرة.
	Apps []string
	// Modes الأوضاعُ اللازمة.
	Modes []Mode
	// Packages حزمُ الاختبار اللازمة.
	Packages []string
	// Why مسارُ الأثر مقروءاً.
	Why string
	// Device أيحتاج جهازاً حقيقيّاً.
	Device string
	// Staging أيحتاج بيئةَ تكامل.
	Staging string
}

// domainRules القواعدُ المشتقّةُ من مواضعَ مقيسةٍ في `P-1…P-8`.
//
// **ولا قاعدةَ بلا مصدرٍ مقيس** — كلُّ سطرٍ هنا يشير إلى ما أُثبت.
var domainRules = []domainRule{
	// ── المال ─────────────────────────────────────────────────────
	{
		Name: "FINANCIAL",
		Match: []string{"backend/internal/wallet/", "backend/internal/cashbox/",
			"backend/internal/orders/transitions.go", "backend/internal/orders/treasury.go",
			"backend/internal/orders/goods.go", "backend/internal/pricing/",
			"backend/internal/incentives/", "backend/internal/referrals/",
			"backend/internal/server/payout_handlers.go",
			"backend/internal/server/expenses_handlers.go",
			"backend/internal/server/admin_wallet_handlers.go"},
		Flows:    []string{"F-14", "F-15", "F-23", "F-24", "F-25", "F-26", "F-27"},
		Apps:     []string{"customer", "merchant", "driver", "rep", "admin"},
		Modes:    []Mode{ModeFinancial, ModeConcurrency, ModeFailure},
		Packages: []string{"internal/fininv", "internal/qa"},
		Why:      "مسٌّ لمسارٍ ماليّ ⇒ ثوابتُ `P-4` وسباقاتُها وحقنُ فشلها (البند ١٤)",
	},
	// ── الخصوصيّة ──────────────────────────────────────────────────
	{
		// ══════════════════════════════════════════════════════════════
		// **منعُ التكرار يعبر المسارات المحميّة كلَّها**
		// ══════════════════════════════════════════════════════════════
		//
		// **وكان غيرَ مذكورٍ في أيّ قاعدة** — فوقع في العامّ، **فقالت
		// `P-9` في دورةِ ٩ تدفّقين اثنين** بينما المنسّقُ يمسّ ستّةَ
		// مساراتٍ في ستّةِ تدفّقاتٍ مختلفة. (صُحّح في مصالحة دورةِ ١٠.)
		//
		// **والمطابقةُ من التسجيل نفسِه**:
		//
		//	POST /orders                     ⇒ F-01 · F-03
		//	POST /orders/custom              ⇒ F-02
		//	POST /admin/users/{id}/wallet    ⇒ F-25
		//	POST /admin/users/{id}/incentive ⇒ F-23 · F-25
		//	POST /admin/payouts/{id}/decide  ⇒ F-24
		//	POST /admin/drivers/{id}/settle  ⇒ F-27
		//
		// **و`F-03` («ردُّ إنشاءٍ ضائع») هو تدفّقُ منع التكرار بعينه** —
		// **وكان غائباً عن أثرِ الدورة التي أصلحته.**
		// **والوسائطُ تعبر كلَّ سطحٍ يعرض صورة** — **وقاعدةُ الأثر
		// كانت تقول تدفّقين** لأنّ `media/` لم يكن مذكوراً، **وهي
		// العلّةُ الوصفيّةُ نفسُها التي صُحّحت لمنع التكرار.**
		Name:     "MEDIA",
		Match:    []string{"backend/internal/media/"},
		Flows:    []string{"F-01", "F-13", "F-21", "F-30", "F-32"},
		Apps:     []string{"customer", "merchant", "driver", "rep", "admin"},
		Modes:    []Mode{ModeSecurity, ModeFailure},
		Packages: []string{"internal/catalog", "internal/qa", "internal/server"},
		Why:      "الوسائطُ تعبر كلَّ سطحٍ يعرض صورةً ⇒ إذنُ الجلب وعقدُ المسارات",
	},
	{
		// **وطبقةُ التدقيق مشتركةٌ بين كلّ فعلٍ حسّاس** — `PF-06`.
		//
		// **وكانت `audit.go` غيرَ مذكورةٍ في أيّ قاعدة** فقالت `P-9`
		// **تدفّقاتٍ صفراً** لتغييرٍ يمسّ ستّةَ أفعالٍ ماليّة.
		// **وهي العلّةُ الوصفيّةُ نفسُها** التي صُحّحت لمنع التكرار
		// وللوسائط — **وثالثةُ مرّةٍ تعني نمطاً لا سهواً.**
		//
		// **والتدفّقاتُ من معجم التدفّقات لا من أسماءٍ مخمَّنة**:
		//
		//	قيدُ المحفظة        ⇒ F-25
		//	الحافزُ            ⇒ F-23 · F-25
		//	قرارُ السحب         ⇒ F-24
		//	تسويةُ نقد السائق   ⇒ F-27
		//	المصروفُ وإلغاؤه    ⇒ F-26
		//	والاستردادُ والتدخّلُ ⇒ F-15 (وهو في `XG-35` بعدُ)
		Name: "AUDIT",
		Match: []string{"backend/internal/server/audit.go",
			"backend/internal/server/audit_tx.go"},
		Flows:    []string{"F-15", "F-23", "F-24", "F-25", "F-26", "F-27"},
		Apps:     []string{"admin", "driver", "merchant", "rep"},
		Modes:    []Mode{ModeSecurity, ModeFinancial, ModeFailure},
		Packages: []string{"internal/fininv", "internal/qa", "internal/server"},
		Why:      "طبقةُ تدقيقٍ مشتركةٌ لأفعالٍ ماليّةٍ ⇒ ذرّيّةُ الأثر وثوابتُ المال",
	},
	{
		Name: "IDEMPOTENCY",
		Match: []string{"backend/internal/server/idempotency.go",
			"backend/internal/server/idempotency_tx.go"},
		Flows: []string{"F-01", "F-02", "F-03", "F-23", "F-24", "F-25", "F-27"},
		Apps:  []string{"customer", "merchant", "driver", "rep", "admin"},
		Modes: []Mode{ModeFinancial, ModeConcurrency, ModeFailure},
		Packages: []string{"internal/fininv", "internal/orders", "internal/qa",
			"internal/server"},
		Why: "منعُ التكرار يحرس ستّةَ مساراتٍ ماليّةٍ ⇒ ثوابتُ المال " +
			"وسباقاتُها وحقنُ فشلها",
	},
	{
		Name: "PRIVACY",
		Match: []string{"backend/internal/server/customer_privacy.go",
			"backend/internal/server/merchant_privacy.go",
			"backend/internal/orders/models.go", "backend/internal/orders/service.go",
			"backend/internal/orders/custom.go"},
		Flows:    []string{"F-01", "F-02", "F-13", "F-14"},
		Apps:     []string{"customer", "merchant", "driver"},
		Modes:    []Mode{ModeSecurity, ModeRealtime},
		Packages: []string{"internal/qa"},
		Why:      "مسٌّ لحقول الطلب أو للتنقية ⇒ عقدُ `P-1` وقنواتُ `P-7` (البند ١٥)",
	},
	// ── الهويّةُ والجلسة ───────────────────────────────────────────
	{
		Name: "AUTH_SESSION",
		// **وبابُ المصافحة منها** — `backend/internal/server/ws`.
		//
		// **وكان غائباً**: **تبديلُ `ws.go` وحدَه يردّ لا تدفّقاً ولا
		// عيباً ولا وضعاً** — **فتخويلُ البثّ لا يراه أثرُ التغيير
		// أصلاً.** (قيس في دورةِ ٥٤.)
		Match: []string{"backend/internal/auth/", "backend/internal/identity/",
			"backend/internal/server/middleware", "backend/internal/server/auth_handlers.go",
			"backend/internal/server/ws", "backend/internal/realtime/"},
		// **وتدفّقا التعليق منها** — `F-28` و`F-29`.
		//
		// **وعليهما `D14`**: **مصافحةُ البثّ لا تسأل عن حال الحساب.**
		// **فمن بدّل توثيقاً أو تخويلَ بثٍّ مسّ إنفاذَ التعليق** —
		// **وكان لا يُنبَّه إليه.**
		Flows:    []string{"F-28", "F-29", "F-30", "F-34"},
		Apps:     []string{"customer", "merchant", "driver", "rep", "admin"},
		Modes:    []Mode{ModeSecurity, ModeRealtime},
		Packages: []string{"internal/qa", "internal/identity"},
		Why:      "مسٌّ للهويّة أو الجلسة أو تخويل البثّ ⇒ أمنٌ وبثٌّ (البند ١٦)",
		Staging:  "**`R16`** — فشلُ Redis المفتوح يحتاج طقمَ خدماتٍ حقيقيّاً",
	},
	// ── الأحداثُ والإشعار ──────────────────────────────────────────
	{
		Name: "EVENTS",
		Match: []string{"backend/internal/notifications/", "backend/internal/push/",
			"backend/internal/orders/watchdog.go"},
		Flows:    []string{"F-07", "F-09", "F-14", "F-34"},
		Apps:     []string{"customer", "merchant", "driver", "rep"},
		Modes:    []Mode{ModeRealtime, ModeFailure},
		Packages: []string{"internal/qa", "internal/orders"},
		Why:      "مسٌّ لباثٍّ أو مُشعِرٍ أو حمولة ⇒ عقودُ `P-7` (البند ١٩)",
		Device:   "الروابطُ العميقةُ وصوتُ الإشعار يحتاجان جهازاً",
	},
	// ── دورةُ حياة الطلب ───────────────────────────────────────────
	{
		Name: "ORDER_LIFECYCLE",
		Match: []string{"backend/internal/orders/", "backend/internal/server/driver_handlers.go",
			"backend/internal/server/admin_orders_handlers.go"},
		Flows: []string{"F-01", "F-04", "F-07", "F-08", "F-12", "F-13", "F-14",
			"F-15", "F-16", "F-17", "F-19"},
		Apps:     []string{"customer", "merchant", "driver", "admin"},
		Modes:    []Mode{ModeConcurrency, ModeFailure},
		Packages: []string{"internal/qa", "internal/orders"},
		Why:      "دورةُ حياة الطلب تمسّ أربعةَ تطبيقاتٍ ولو تبدّل ملفٌّ واحد (البند ١٨)",
	},
	// ── التحويلُ إلى المتجر ────────────────────────────────────────
	//
	// **و`auto_transfer.go` كان يُقرأ «ملفَّ خادمٍ عاديّاً»** —
	// `FLOWS = []` و`RISK CLASS = LOW`. **وهو منفّذُ `F-20` بعينه**،
	// **ويحمل `R21` و`R24`.** (مصالحةُ دورةِ ٢٧.)
	//
	// **وقناةُ الإبلاغ خارجيّةٌ تفشل** — فوضعُ الفشل إلزاميّ.
	{
		Name: "ORDER_TRANSFER",
		Match: []string{"backend/internal/server/auto_transfer.go",
			"backend/internal/server/merchant_dispatch.go",
			"backend/internal/notify/whatsapp"},
		Flows:    []string{"F-01", "F-14", "F-20"},
		Apps:     []string{"customer", "merchant", "admin"},
		Modes:    []Mode{ModeFailure, ModeRealtime},
		Packages: []string{"internal/qa", "internal/orders"},
		Why:      "التحويلُ إلى المتجر — `F-20` و`R21` و`R24`: قناةٌ خارجيّةٌ تفشل بعد قبولٍ وقع",
	},
	// ── المندوبُ والمرشَّح ─────────────────────────────────────────
	{
		Name:     "REP_LEADS",
		Match:    []string{"backend/internal/server/leads_handlers.go"},
		Flows:    []string{"F-21", "F-22", "F-23"},
		Apps:     []string{"rep", "merchant", "admin"},
		Modes:    []Mode{ModeFailure, ModeConcurrency, ModeFinancial},
		Packages: []string{"internal/qa"},
		Why:      "تحويلُ المرشَّح — `D2` و`D25` و`XG-18`",
	},
	// ── أندرويد · السائق ───────────────────────────────────────────
	{
		Name: "ANDROID_DRIVER_LOCATION",
		Match: []string{"mobile/app-driver/src/main/kotlin/com/rahalgo/driver/location/",
			"mobile/driver-navigation/", "mobile/map/"},
		Flows:    []string{"F-11", "F-13"},
		Apps:     []string{"driver"},
		Modes:    []Mode{ModeAndroid},
		Packages: []string{"internal/androidmap"},
		Why:      "طبقةُ الموقعِ أو الملاحة ⇒ `R17` و`R19` و`R20` و`D16` و`D18`",
		Device: "**`REAL DEVICE VALIDATION REQUIRED`** — الخلفيّةُ والغفوةُ وموتُ " +
			"العمليّة لا تُثبَت محلّيّاً (البند ٤٤)",
	},
	// ── الويب ──────────────────────────────────────────────────────
	{
		Name:  "ADMIN_WEB",
		Match: []string{"web/"},
		Flows: []string{"F-19", "F-25", "F-26", "F-33"},
		Apps:  []string{"admin"},
		Modes: []Mode{ModeWeb},
		Why:   "لوحةُ الإدارة — حرّاسُ الويب و`typecheck` و`eslint`",
	},
	// ── النشر ──────────────────────────────────────────────────────
	{
		Name:    "DEPLOY",
		Match:   []string{"deploy/", "docker-compose.yml", "Dockerfile", "nginx"},
		Modes:   []Mode{ModeFull},
		Why:     "تبديلُ نشرٍ أو تشغيل — لا يُثبَت محلّيّاً (البند ٢٦)",
		Staging: "**`STAGING VALIDATION REQUIRED`** — الصحّةُ والجاهزيّةُ والهجرات",
	},
}

// modeSet يجمع الأوضاعَ بلا تكرار.
func modeSet(dst *[]Mode, ms ...Mode) {
	seen := map[Mode]bool{}
	for _, m := range *dst {
		seen[m] = true
	}
	for _, m := range ms {
		if !seen[m] {
			seen[m] = true
			*dst = append(*dst, m)
		}
	}
}

// RulesSnapshot صورةٌ آليّةٌ لقواعد الاختيار (البند ٤٩).
//
// **ولا تُترَك قواعدُ الاختيار في Markdown وحدَه** — `P-10` يقرؤها.
func RulesSnapshot() map[string]any {
	type ruleOut struct {
		Name     string   `json:"name"`
		Match    []string `json:"match"`
		Flows    []string `json:"flows,omitempty"`
		Apps     []string `json:"apps,omitempty"`
		Modes    []Mode   `json:"modes,omitempty"`
		Packages []string `json:"packages,omitempty"`
		Why      string   `json:"why"`
		Device   string   `json:"device_required,omitempty"`
		Staging  string   `json:"staging_required,omitempty"`
	}
	var out []ruleOut
	for _, r := range domainRules {
		out = append(out, ruleOut{Name: r.Name, Match: r.Match, Flows: r.Flows,
			Apps: r.Apps, Modes: r.Modes, Packages: r.Packages, Why: r.Why,
			Device: r.Device, Staging: r.Staging})
	}
	cats := []Category{CatBackend, CatAdminWeb, CatAndCust, CatAndDriver,
		CatAndMerch, CatAndRep, CatAndShared, CatMigration, CatSettings,
		CatTestInfra, CatTest, CatDeploy, CatDocs, CatTruthDocs, CatUnknown}
	return map[string]any{
		"domain_rules": out,
		"categories":   cats,
		"counts": map[string]int{
			"domain_rules": len(out),
			"categories":   len(cats),
			"input_modes":  3,
		},
		"policy": map[string]string{
			"unknown":             "UNKNOWN IMPACT → SAFE FULL FALLBACK",
			"low_confidence":      "LOW CONFIDENCE → SAFE FULL FALLBACK",
			"no_false_confidence": "**«لا أثر» تُقال بدليلٍ أو لا تُقال**",
		},
	}
}
