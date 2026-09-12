package qa

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// ══════════════════════════════════════════════════════════════════════
// **حدُّ العمليّات والماليّة — متقاطعاً** (`OFM`)
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا اختبارٌ للحدّ لا للدور
//
// **ومصفوفةُ `ADG-2` تقيس ما يملكه كلُّ دورٍ من عمله.** **وهذه تقيس
// ما لا يملكه من عمل غيره** — **وهما ليسا الشيءَ نفسَه**: **دورٌ
// يُمنَح قدرةً زائدةً غداً يبقى ناجحاً في الأولى ويسقط في هذه.**
//
// # وعقدُ المالك (٢٠٢٦-٠٩-١٣)
//
//	العمليّاتُ   تنفيذُ الطلب واستمرارُ الخدمة
//	الماليّةُ    حقيقةُ المال والتسوية
//
// **وأثرُ الفعل التشغيليِّ الماليُّ يقع تلقائيّاً في معاملةِ المجال
// الموثوقة** — **ولا يُعيد قسمٌ كتابةَ ما كتبه قسمٌ آخر بيده.**

// opsFinanceProbes **نداءاتُ الحدّ** — ولا تُخلَط بمجسّات `ADG-2`.
//
// **وكلُّها على صفٍّ حقيقيٍّ قائم**: **مسارٌ بمعرّفٍ مخترَعٍ يردّ ٤٠٤
// فيُقرأ منعاً وهو غياب.**
func opsFinanceProbes(t *testing.T, hh *Harness) map[string]probe {
	t.Helper()
	v := hh.NewUser("customer")
	m := hh.Factory().Merchant()
	oid, dr := activeOrderFor(t, hh)
	return map[string]probe{
		// ── أفعالٌ ماليّةٌ لا تخصّ العمليّات ──────────────────────
		"تسويةُ نقدِ سائق": {"تسويةِ نقدِ سائق", "POST",
			"/api/v1/admin/drivers/" + dr.ID + "/settle",
			map[string]any{"amount": 100, "note": "OFM"}},
		"تعويضُ سائق": {"تعويضِ سائقٍ عن طلب", "POST",
			"/api/v1/admin/orders/" + oid + "/compensate-driver",
			map[string]any{"amount": 100, "reason": "OFM"}},
		"مصروفٌ جديد": {"قيدِ مصروف", "POST", "/api/v1/admin/expenses",
			map[string]any{"amount": 100, "note": "OFM"}},
		"قراءةُ نقدِ السائقين": {"قراءةِ نقدِ السائقين المعلَّق", "GET",
			"/api/v1/admin/cash/outstanding", nil},
		"تفصيلُ مالِ الطلب": {"تفصيلِ مالِ الطلب", "GET",
			"/api/v1/admin/orders/" + oid + "/breakdown", nil},
		"مرشَّحو الخزينة": {"مرشَّحي الخزينة", "GET",
			"/api/v1/admin/treasury-candidates", nil},
		"تسويةُ نزاع": {"تسويةِ نزاع", "POST",
			"/api/v1/admin/disputes/00000000-0000-0000-0000-000000000001/settle",
			map[string]any{"amount": 100}},

		// ── أفعالٌ تشغيليّةٌ لا تخصّ الماليّة ─────────────────────
		"إسنادُ سائق": {"إسنادِ سائقٍ لطلب", "POST",
			"/api/v1/admin/orders/" + oid + "/assign",
			map[string]any{"driver_id": dr.ID}},
		"تحويلُ طلب": {"تحويلِ طلبٍ إلى متجرٍ آخر", "POST",
			"/api/v1/admin/orders/" + oid + "/transfer",
			map[string]any{"merchant_id": m.ID}},
		"إعادةُ حسابِ طلب": {"إعادةِ حسابِ طلب", "POST",
			"/api/v1/admin/orders/" + oid + "/recompute", map[string]any{}},

		// ── والقراءتان المقسومتان (٢٠٢٦-٠٩-١٣) ───────────────────
		"قراءةُ المتاجر": {"قراءةِ سجلّ المتاجر", "GET",
			"/api/v1/admin/merchants", nil},
		"قراءةُ الإعدادات": {"قراءةِ لوح الإعدادات", "GET",
			"/api/v1/admin/settings", nil},
		"إنشاءُ متجر": {"إنشاءِ متجر", "POST", "/api/v1/admin/merchants",
			map[string]any{"name": "OFM", "phone": "+963900000777"}},
		"نشاطُ المندوبين": {"نشاطِ المندوبين", "GET",
			"/api/v1/admin/ops-map/reps", nil},

		// ── وأبوابٌ لا يملكها القسمان ────────────────────────────
		"منحُ دور": {"منحِ دور", "POST",
			"/api/v1/admin/users/" + v.ID + "/roles",
			map[string]any{"role": "finance", "reason": "OFM"}},
		"صحّةُ المنصّة": {"صحّةِ المنصّة", "GET",
			"/api/v1/admin/ops/health", nil},
	}
}

// TestOFM1_OperationsHasNoFinancialHand **لا يدَ ماليّةً للعمليّات.**
//
// **وشرطُ المالك**: **أثرُ الفعل التشغيليِّ الماليُّ يقع تلقائيّاً** —
// **ولا تُعيد العمليّاتُ كتابتَه بيدها.**
func TestOFM1_OperationsHasNoFinancialHand(t *testing.T) {
	hh := New(t)
	p := opsFinanceProbes(t, hh)

	checkRole(t, hh, "operations",
		// **ومن عملها**: سياقُ المتجر والمفتاحُ التشغيليّ.
		[]probe{p["قراءةُ المتاجر"], p["قراءةُ الإعدادات"],
			p["إسنادُ سائق"], p["إعادةُ حسابِ طلب"]},
		// **وليس من عملها**: كلُّ يدٍ على المال، **ولا إنشاءُ متجرٍ
		// ولا نشاطُ مندوبين** — نُزعت `merchants.manage` ٢٠٢٦-٠٩-١٣.
		[]probe{
			p["تسويةُ نقدِ سائق"], p["تعويضُ سائق"], p["مصروفٌ جديد"],
			p["قراءةُ نقدِ السائقين"], p["تفصيلُ مالِ الطلب"],
			p["مرشَّحو الخزينة"], p["تسويةُ نزاع"],
			p["إنشاءُ متجر"], p["نشاطُ المندوبين"],
			p["منحُ دور"], p["صحّةُ المنصّة"],
		})
}

// TestOFM2_FinanceHasNoOperationalHand **لا يدَ تشغيليّةً للماليّة.**
func TestOFM2_FinanceHasNoOperationalHand(t *testing.T) {
	hh := New(t)
	p := opsFinanceProbes(t, hh)

	checkRole(t, hh, "finance",
		// **ومن عملها**: المالُ كلُّه، وسياقُ الطلب الماليّ.
		[]probe{
			p["تسويةُ نقدِ سائق"], p["مصروفٌ جديد"],
			p["قراءةُ نقدِ السائقين"], p["تفصيلُ مالِ الطلب"],
			p["تعويضُ سائق"], p["قراءةُ الإعدادات"],
		},
		// **وليس من عملها**: يدٌ على تشغيل الطلب، **ولا سجلُّ
		// متاجرَ ولا مندوبون ولا أدوار.**
		[]probe{
			p["إسنادُ سائق"], p["تحويلُ طلب"], p["إعادةُ حسابِ طلب"],
			p["إنشاءُ متجر"], p["نشاطُ المندوبين"],
			p["منحُ دور"], p["صحّةُ المنصّة"],
		})
}

// TestOFM3_ReadSplitDoesNotWiden **والقسمةُ لم توسّع أحداً.**
//
// **وقدرةٌ تُقسَم قد تُقرأ توسعةً**: **من نال `merchants.read` قد
// ينال الكتابةَ معها لو أخطأ الجدولُ الأخصَّ.** **فيُقاس أنّ من نال
// القراءةَ وحدَها لم ينل غيرَها.**
func TestOFM3_ReadSplitDoesNotWiden(t *testing.T) {
	hh := New(t)
	p := opsFinanceProbes(t, hh)

	// **ودورٌ يُصنَع لهذا القياس بقدرتين لا غير** — و`capRole` تنظّف بعدَه.
	capRole(t, hh, "ofm_reader", authz.MerchantsRead, authz.SettingsRead)

	checkRole(t, hh, "ofm_reader",
		[]probe{p["قراءةُ المتاجر"], p["قراءةُ الإعدادات"]},
		[]probe{
			p["إنشاءُ متجر"], p["نشاطُ المندوبين"],
			p["إسنادُ سائق"], p["مصروفٌ جديد"], p["منحُ دور"],
			p["صحّةُ المنصّة"],
		})
}

// TestOFM4_CapabilityNotRoleName **والقدرةُ هي المرجع لا الاسم.**
//
// **ودورٌ مخصَّصٌ يُنشَأ اليومَ بقدرةِ المال يبلغ أبوابَ المال** —
// **بلا سطرٍ يُكتب له في مصفوفة.** (عقدُ المالك:
// `NO-CODE FOR OPERATIONS`.)
func TestOFM4_CapabilityNotRoleName(t *testing.T) {
	hh := New(t)
	p := opsFinanceProbes(t, hh)

	capRole(t, hh, "ofm_custom_money", authz.FinanceRead, authz.FinanceManage)

	checkRole(t, hh, "ofm_custom_money",
		[]probe{p["قراءةُ نقدِ السائقين"], p["مصروفٌ جديد"],
			p["تسويةُ نقدِ سائق"], p["تفصيلُ مالِ الطلب"]},
		[]probe{p["إسنادُ سائق"], p["إنشاءُ متجر"], p["منحُ دور"],
			p["صحّةُ المنصّة"]})
}
