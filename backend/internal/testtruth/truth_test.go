// حارسُ حقيقةِ الاختبار — **الانحرافُ سقوطٌ لا تقرير.**
//
// (البند ١٢ من طلب المالك: `TRUTH DRIFT = BUILD/QA FAILURE`.)
//
// **وهو النمطُ العاملُ في المشروع لـ`TRUTH.md`** — يُعاد التوليدُ ويُقارَن،
// **فما شاخ أسقط البناء.**
package testtruth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
)

// roots جذرا المشروع من موضع هذه الحزمة.
func roots(t *testing.T) (backend, docs string) {
	t.Helper()
	wd, err := os.Getwd() // internal/testtruth
	if err != nil {
		t.Fatal(err)
	}
	backend = filepath.Dir(filepath.Dir(wd))
	docs = filepath.Join(filepath.Dir(backend), "docs")
	if _, err := os.Stat(filepath.Join(backend, "go.mod")); err != nil {
		t.Fatalf("لم أجد جذرَ المحرّك عند %s", backend)
	}
	return backend, docs
}

func build(t *testing.T) *Truth {
	t.Helper()
	backend, docs := roots(t)
	tr, err := Build(backend, docs)
	if err != nil {
		t.Fatalf("تعذّر بناءُ الحقيقة: %v", err)
	}
	return tr
}

// ══════════════════════════════════════════════════════════════════════
// **١ · الانحراف — أهمُّ حارسٍ في المرحلة**
// ══════════════════════════════════════════════════════════════════════

func TestTruthIsCurrent(t *testing.T) {
	tr := build(t)
	_, docs := roots(t)
	out := filepath.Join(docs, "testing", "system")

	for _, c := range []struct {
		name string
		want []byte
	}{
		{"TEST_TRUTH.json", mustJSON(t, tr)},
		{"TEST_TRUTH.md", []byte(tr.Report())},
	} {
		got, err := os.ReadFile(filepath.Join(out, c.name))
		if err != nil {
			t.Fatalf("%s غيرُ موجود — شغّلْ `go run ./cmd/testtruth`", c.name)
		}
		if norm(string(got)) != norm(string(c.want)) {
			t.Errorf("%s شاخ — أعِد التوليدَ بـ`go run ./cmd/testtruth`", c.name)
		}
	}
}

func mustJSON(t *testing.T, tr *Truth) []byte {
	t.Helper()
	b, err := tr.JSON()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// norm **نهاياتُ الأسطر ليست انحرافاً** — ويندوز يبدّلها عند السحب.
func norm(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }

// ══════════════════════════════════════════════════════════════════════
// **٢ · لا عيبَ يجهله النظام**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ٥: **ممنوعٌ عيبٌ مسجَّلٌ لا يعرف النظامُ بوجوده.**)

func TestEveryDefectIsKnown(t *testing.T) {
	tr := build(t)
	if len(tr.Defects) == 0 {
		t.Fatal("لم يُقرأ عيبٌ واحدٌ من السجلّ المجمَّد")
	}
	seen := map[string]bool{}
	for _, d := range tr.Defects {
		if seen[d.ID] {
			t.Errorf("عيبٌ مكرَّرٌ في السجلّ: %s", d.ID)
		}
		seen[d.ID] = true
		if d.Title == "" {
			t.Errorf("%s بلا عنوان — تبدّلت صيغةُ السجلّ", d.ID)
		}
	}
	// **والمتّصلُ يكشف ثغرةً** — عيبٌ حُذف أو رقمٌ قُفز.
	for i := 1; i <= len(tr.Defects); i++ {
		id := "D" + itoa(i)
		if !seen[id] {
			t.Errorf("ثغرةٌ في ترقيم العيوب: %s مفقود", id)
		}
	}
	t.Logf("DEFECTS MAPPED = %d/%d", len(tr.Defects), len(tr.Defects))
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · لكلّ خطرٍ استراتيجيّةُ تحقّق**
// ══════════════════════════════════════════════════════════════════════

func TestEveryRiskHasStrategy(t *testing.T) {
	tr := build(t)
	if len(tr.Risks) == 0 {
		t.Fatal("لم يُقرأ خطرٌ واحدٌ من السجلّ")
	}
	for _, r := range tr.Risks {
		if r.Strategy == "" {
			t.Errorf("%s بلا استراتيجيّةِ تحقّق — أضِفها إلى RiskStrategy", r.ID)
		}
	}
	t.Logf("RISKS MAPPED = %d/%d", len(tr.Risks), len(tr.Risks))
}

// ══════════════════════════════════════════════════════════════════════
// **٤ · لا مرجعَ شائخ**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ١٤: **ممنوعٌ مرجعٌ وهميّ.**)

func TestNoStaleReferences(t *testing.T) {
	tr := build(t)
	for _, s := range tr.Stale {
		t.Errorf("مرجعٌ شائخ: %s", s)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · التدفّقاتُ الخمسةُ والثلاثون كلُّها معلَنة**
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا صارت خمسةً وثلاثين
//
// **`F-35` — خريطةُ العمليات**، بُنيت بطلب المالك ٢٠٢٦-٠٩-٠٦.
//
// **و`F-36` — حذفُ الحساب بطلب صاحبه**، **سُجّل في دورةِ ٥٧**:
// **مسارٌ قائمٌ في المنتَج لم يُسجَّل قطّ** — **ولا اختبارَ يمرّ
// به** (`XG-49`). **والتسجيلُ يُظهر نقصَ الدليل، وتركُه يُخفيه.**
//
// **والحارسُ لا يُضعَّف ليمرّ**: **بقي عدداً مقطوعاً وتُرقّم فجوتُه** —
// **ومن جعله `>= 34` سمح لتدفّقٍ أن يسقط صامتاً.** **وميزةٌ جديدةٌ
// تُعلَن بتبديل رقمٍ في سطرٍ يُقرأ، لا بحارسٍ يتساهل.**
const declaredFlows = 36

func TestAllFlowsDeclared(t *testing.T) {
	tr := build(t)
	if len(tr.Flows) != declaredFlows {
		t.Fatalf("التدفّقاتُ %d لا %d — تبدّل الكتالوج",
			len(tr.Flows), declaredFlows)
	}
	seen := map[string]bool{}
	for _, f := range tr.Flows {
		if seen[f.ID] {
			t.Errorf("تدفّقٌ مكرَّر: %s", f.ID)
		}
		seen[f.ID] = true
		if f.Title == "" || f.Severity == "" {
			t.Errorf("%s ناقصُ العنوان أو الشدّة", f.ID)
		}
		if len(f.NeedLevels) == 0 {
			t.Errorf("%s بلا طبقةٍ لازمة — قاعدةُ الاشتقاق مكسورة", f.ID)
		}
	}
	for i := 1; i <= declaredFlows; i++ {
		id := "F-" + pad2(i)
		if !seen[id] {
			t.Errorf("تدفّقٌ مفقود: %s", id)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٦ · المستخرِجاتُ تُطابق مصادرَها**
// ══════════════════════════════════════════════════════════════════════
//
// **ومستخرِجٌ صامتٌ يردّ صفراً أخطرُ من مستخرِجٍ يسقط** — **ورقمٌ خاطئٌ
// يُصدَّق.** (وقع ثلاثَ مرّاتٍ في بناء هذه المرحلة: **٣ انتقالاً من ٥٥ ·
// ونوعٌ واحدٌ من ثلاثةَ عشر · و٩٢ إعداداً من ١١٨.**)

func TestExtractorsAgreeWithSource(t *testing.T) {
	tr := build(t)
	d := tr.Derived

	checks := []struct {
		name string
		got  int
		min  int
	}{
		{"أبوابٌ في الموجّه", d.Routes, 200},
		{"انتقالاتُ الطلب", d.Transitions, 50},
		{"أنواعُ قيدِ المحفظة", len(d.LedgerKinds), 9},
		{"مواضعُ الإشعار", d.NotifySites, 40},
		{"غرفُ البثّ", len(d.PublishRooms), 4},
		{"تعريفاتُ الإعدادات", d.SettingDefs, 110},
		{"حقولُ الطلب", d.OrderFields, 70},
		{"ملفّاتُ الاختبار", d.TestFiles, 150},
		{"دوالُّ الاختبار", d.TestFuncs, 400},
	}
	for _, c := range checks {
		if c.got < c.min {
			t.Errorf("%s = %d — والحدُّ الأدنى المعقول %d. **المستخرِجُ مكسورٌ أو المصدرُ تبدّل.**",
				c.name, c.got, c.min)
		}
	}

	// **والإعداداتُ المُغيِّرةُ للسلوك ١٠٣** — كما في `CONFIG_IMPACT_MAP.md`.
	//
	// **وصارت ستّاً وتسعين في دورةِ ٣١** — `sales.commission_source`
	// (`XG-13`): **مفتاحٌ يبدّل قاعدةَ حسابِ مالٍ يُدفَع.**
	//
	// **وسبعةً ومئةً في دفعةِ ما قبل الإطلاق** (٢٠٢٦-٠٩-١٣): **أبوابُ
	// وضعِ الإطلاق السبعة** — تسجيلٌ وتصفّحٌ وطلبٌ عاديٌّ ومخصَّصٌ
	// ودوامُ سائقٍ وطلباتُ متجرٍ وضمُّ متاجر. **وكلُّها تبدّل مساراً:
	// يُردّ النداءُ أو يمرّ.**
	//
	// **و`launch.notice` ليست فيها** — **نصٌّ يُقرأ لا مسارٌ يتبدّل**،
	// ونوعُه `longtext` فيُصنَّف عرضاً كسائر النصوص الطويلة.
	// **وأحدَ عشرَ ومئةً بمركز التنزيل** (`DLC`، ٢٠٢٦-٠٩-١٣): **ثمانيةٌ
	// أُضيفت** — **رابطُ متجرٍ ونسخةٌ لكلٍّ من الأربعة.**
	//
	// **والرابطُ يبدّل مساراً**: **زرٌّ يظهر ويقود إلى خارج الموقع.**
	// **والنسخةُ تبدّل اسمَ الملفّ المُنزَّل** — **فليست نصَّ صفحةٍ.**
	//
	// **وملفّاتُ الآثار الأربعةُ عرضٌ بقاعدة `behaviour`** (`file`) —
	// **وهي التي تفتح الزرَّ فعلاً.** **فالقاعدةُ تقرأ النوعَ لا
	// الأثرَ**، **ولا تُبدَّل قاعدةٌ عامّةٌ لأجل أربعةِ مفاتيح** —
	// **ويُسجَّل الحدُّ هنا كي يُقرأ.**
	// **وثلاثةٌ أُضيفت بمركز الإشعارات** (الدفعة الثامنة، ٢٠٢٦-٠٩-١٥):
	// **ساعتا الهدوء وسقفُ اليوم** — **وكلُّها تبدّل سلوكاً**: **من
	// رفع الهدوءَ أيقظ الناسَ ليلاً، ومن رفع السقفَ أغرقهم.**
	// **وثلاثةٌ أُضيفت بعقد الطلب المخصَّص** (Batch 2a): **مصدرُ أجرة
	// المخصَّص وقيمتُها وإذنُ السائق بتغييرها** — **وكلُّها تبدّل ما يُحسَب.**
	// **وواحدةٌ أُضيفت في Batch 4**: `referral.reward_4` — مكافأةُ الدعوةِ
	// الرابعةِ رتبةً قائمةً بذاتها (تبدّل ما يُصرَف).
	// **وواحدةٌ أُضيفت في إغلاق الزبون** (COD): `customers.cod_limit` — سقفُ
	// النقد غيرِ المسدَّدِ بذمّة الزبون **يبدّل ما يُقبَل**: من رفعه سمح
	// بنقدٍ أكثرَ لا يُقبض، ومن خفضه ردَّ طلباتٍ نقديّة.
	// **وتسعةٌ أُضيفت بالتوزيع بالقرب** (`a0a6b42a`، ٢٠٢٦-٠٩-٢٨) —
	// **ونزلت بلا تحديثِ هذا العدد، فبقي البناءُ أحمرَ منذاك** (مقيسٌ على
	// `d55efc73` نظيفاً: ١٢٩ لا ١٢٠، ٢٠٢٦-٠٩-٢٩). **وكلُّها تبدّل من يأخذ
	// الطلبَ لا شكلَ شاشة:**
	//
	//	proximity_enabled        ← محرّكٌ كاملٌ يُشعَل ويُطفأ
	//	location_fresh_sec       ← بعدها لا يُعرض على السائق أصلاً
	//	dispatch_radius_initial_m · _step_m · _max_m ← حدُّ من يُعرض عليه ومتى يتوسّع
	//	proximity_bucket_m       ← شريحةُ التعادل: من يسبق من عند تقاربِ المسافة
	//	same_route_radius_m · same_route_spread_m ← ضمُّ طلبٍ ثانٍ إلى راكبِ الطريق
	//	zone_gate_enabled        ← منعُ من هو خارجَ المنطقة
	//
	// **ولا تُبدَّل هذه الأرقامُ لتمريرِ بناء** — **يُقرأ الفرقُ ويُسمّى**،
	// وإلّا صار الحارسُ عدّاداً يُرضى لا حقيقةً تُحرَس.
	// **وواحدٌ أُضيف بنصيب المنصّة من التوصيل** (قرارا المالك ٢٠٢٦-٠٩-٢٩):
	// `delivery.merchant_delivery_platform_percent` — **يبدّل ما يُقسَم بين
	// السائق والمنصّة في «لدي توصيلة» وفي الطلب الخاصّ معاً.** **ويُلقَط
	// على الطلب لحظةَ الاتّفاق**، فتغييرُه لا يمسّ ما مضى.
	// **وواحدٌ أُضيف بحدّ سعر الصنف** (قرارُ المالك ٢٠٢٦-١٠-٠١: «أعلى سعر ١٠٠ ألف»):
	// `merchants.max_item_price` — **يردّ صنفاً أغلى منه في الإنشاء والتعديل.**
	// **وواحدٌ أُضيف بانتظار الباب** (قرارُ المالك ٢٠٢٦-١٠-٠٢): `drivers.door_wait_sec`
	// — **ثمّ حُذف** (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢): السائقُ لا يُنهي الطلبَ عند
	// الباب، **والإدارةُ تُنهي** — فلا ضغطةَ تُؤخَّر.
	// **وواحدٌ حُذف بدعم المتجر عند ردّ البضاعة** (قرارُ المالك ٢٠٢٦-١٠-٠٣):
	// `merchants.return_support_percent` — **صار التعويضُ مبلغاً تكتبه الإدارةُ عند الحسم.**
	// **وثلاثةٌ أُضيفت بمنع احتيال السائق** (قرارُ المالك ٢٠٢٦-١٠-٠٣ مساءً): حدّا أجرة الطلب
	// الخاصّ `delivery.custom_fee_min/max` **وحدُّ صورة التسليم** `drivers.proof_max_m`.
	// **وواحدٌ أُضيف بمهلة الردّ على الشكوى** (قرارُ المالك ٢٠٢٦-١٠-٠٤):
	// `support.late_reply_hours` — **شكوى بلا ردٍّ بعدها تُعدّ متأخّرةً في الرئيسيّة.**
	// **وواحدٌ حُذف بمراجعة الأصناف** (قرارُ المالك ٢٠٢٦-١٠-٠٤: «ما في داعي للموافقة
	// على الصنف أساساً»): `merchants.menu_requires_approval` — هجرة ٠٢٠٠.
	// **وثلاثةٌ أُضيفت بلوحة الطلبات** (قرارُ المالك ٢٠٢٦-١٠-٠٤): مهلةُ الإسناد اليدويّ
	// `orders.manual_assign_after_min` (كانت تُقرأ ولا فهرسَ لها) **والقبولُ حين يفرغ المكتب**
	// `orders.unattended_auto_accept_min` و`orders.staff_presence_min`.
	// **وواحدٌ أُضيف بمدّة حفظ سطور الدخول** (قرارُ المالك ٢٠٢٦-١٠-٠٤ على سجلّ الأحداث):
	// `security.audit_session_retention_days` — **سطورُ الدخول والجلسة تُحذف بعدها، وما سواها للأبد.**
	// **واثنان أُضيفا بمراقبة التشغيل** (قرارُ المالك ٢٠٢٦-١٠-٠٤): تكرارُ تذكير العالق
	// `ops.stuck_reminder_min` **وإشعارُ تعطّل الخادم** `ops.outage_notify_min`.
	// **وأربعةٌ أُضيفت بقسم الحسابات** (قرارُ المالك ٢٠٢٦-١٠-٠٤): مهلةُ الكلمة المؤقّتة
	// `security.temp_password_hours` · حدُّ الرصيد لموافقةٍ ثانيةٍ على تغيير الرقم
	// `security.phone_change_approval_balance` · سقفُ الحركة اليدويّة `finance.manual_wallet_max`
	// · وحدُّ تنبيه الإنذارات `safety.warnings_alert_count`.
	// **وواحدٌ أُضيف بعودة التذكير بعد «أنا عليه»** (قرارُ المالك 2026-10-04):
	// `ops.stuck_ack_snooze_min` — **«أنا عليه» تُسكت التذكيرَ ساعةً لا للأبد.**
	// **وواحدٌ أُضيف بغرفة الطوارئ** (قرارُ المالك ٢٠٢٦-١٠-٠٤): `ops.emergency_unacked_red_min`
	// — **طارئٌ بلا مستلِمٍ عشرَ دقائق يحمرّ.**
	// **وواحدٌ أُضيف بمصروفات التشغيل** (قرارُ المالك ٢٠٢٦-١٠-٠٤): سقفُ الموافقة الثانية
	// `finance.expense_approval_threshold`.
	// **واثنان أُضيفا بقسم النقد** (قرارُ المالك ٢٠٢٦-١٠-٠٤): `drivers.cash_overdue_days`
	// و`drivers.cash_overdue_stop`.
	// **وأربعةٌ أُضيفت بقسم التعويضات** (قرارُ المالك ٢٠٢٦-١٠-٠٤): سقوفُ الأنواع الثلاثة
	// `compensations.cap_driver|cap_merchant_goods|cap_complaint` و`compensations.overdue_hours`.
	// **وواحدٌ أُضيف بقسم الديون** (قرارُ المالك ٢٠٢٦-١٠-٠٤): `finance.obligation_alert_days`.
	// **ونقصت أربعةً بقسم الإعدادات** (قراراتُ المالك ٢٠٢٦-١٠-٠٤): حُذف مصدرُ عمولة
	// المندوب `sales.commission_source` وعتبةُ التفعيل `sales.activation_orders` ورمزُ
	// دعوة المنصّة `platform.invite_code` والمفتاحان القديمان للتطبيق
	// `platform.app_url`/`platform.app_file`، **وصارت حصّةُ المنصّة مفتاحاً واحداً
	// لكلّ الأنواع** `delivery.platform_percent` بدل القديم، **وأُضيف قالبُ الترحيب**
	// `accounts.welcome_template` (نصٌّ طويلٌ — عرضٌ لا سلوك).
	// **وواحدٌ أُضيف بترحيب الزبون** (قرارُ المالك ٢٠٢٦-١٠-٠٥): مفتاحُ الواتساب
	// `customers.welcome_whatsapp` — **والقالبُ نصٌّ طويلٌ لا يُعدّ.**
	// **وأربعةٌ أُضيفت بإشعارات الوجبات** (قرارُ المالك ٢٠٢٦-١٠-٠٥): `meals.enabled`
	// ومواعيدُ `meals.breakfast_at|lunch_at|dinner_at` — **والنصوصُ طويلةٌ لا تُعدّ.**
	// **وواحدٌ أُضيف بإظهار السلايدر** (قرارُ المالك ٢٠٢٦-١٠-٠٥): `home.banner_enabled`
	// — **ورسائلُ التطبيق `app_text.*` نصوصٌ لا تُعدّ.**
	if d.BehaviourSettings != 155 {
		t.Errorf("إعداداتُ السلوك = %d لا 155 — راجِعْ قاعدةَ `behaviour` أو المعجم",
			d.BehaviourSettings)
	}
	// **والموجَّهُ من الإشعارات ١١ من ٤٩** — حقيقةٌ مقيسةٌ في إغلاق المنظومة.
	if d.NotifyTargeted > d.NotifySites {
		t.Errorf("الموجَّهُ %d أكبرُ من المجموع %d — القياسُ مكسور",
			d.NotifyTargeted, d.NotifySites)
	}
	t.Logf("derived: أبوابٌ %d · انتقالاتٌ %d · أنواعٌ %d · إشعاراتٌ %d/%d · غرفٌ %d",
		d.Routes, d.Transitions, len(d.LedgerKinds), d.NotifyTargeted, d.NotifySites, len(d.PublishRooms))
	t.Logf("        إعداداتٌ %d (سلوكٌ %d) · حقولُ طلبٍ %d · اختباراتٌ %d في %d ملفّاً",
		d.SettingDefs, d.BehaviourSettings, d.OrderFields, d.TestFuncs, d.TestFiles)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func pad2(n int) string {
	s := itoa(n)
	if len(s) < 2 {
		return "0" + s
	}
	return s
}

// TestFinancialSettingsExist **البند ٩ من `P-4`** — مفاتيحُ الإعدادات
// الماليّةِ موجودةٌ في المعجم الفعليّ.
//
// **ولا يُنسَخ مفتاحٌ من ذاكرة.** `fininv.FinancialSettings` تُسمّي ستّةً
// وعشرين مفتاحاً تدخل حساباً ماليّاً، **وهذا يطابقها بما يستخرجه المولّدُ
// من `settings/catalog.go`** — **فمفتاحٌ يُعاد تسميتُه غداً يُسقط البناءَ
// بدل أن يصمت الحسابُ.**
func TestFinancialSettingsExist(t *testing.T) {
	backend, _ := roots(t)
	defs, err := Root(backend).SettingDefs()
	if err != nil {
		t.Fatalf("معجمُ الإعدادات: %v", err)
	}
	have := map[string]bool{}
	for _, d := range defs {
		have[d.Key] = true
	}
	missing := 0
	for _, key := range fininv.FinancialSettings {
		if !have[key] {
			t.Errorf("مفتاحٌ ماليٌّ لا وجودَ له في المعجم: %q", key)
			missing++
		}
	}
	if missing == 0 {
		t.Logf("FINANCIAL SETTINGS MAPPED = %d/%d", len(fininv.FinancialSettings), len(defs))
	}
}
