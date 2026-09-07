package qa

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// ══════════════════════════════════════════════════════════════════════
// **تسعةُ أدوارٍ وظيفيّةٍ ومصفوفتُها** — `ADG-2` · `AQ-1`
// ══════════════════════════════════════════════════════════════════════
//
// **ولا خانةَ مظنونة**: كلُّ سطرٍ هنا نداءٌ حقيقيٌّ على مسارٍ قائم.
//
// **والعقدُ لكلّ دور** مدوَّنٌ في `ADMIN_FINAL_PRODUCT_DECISIONS.md`
// تحت `AQ-1` بقرار المالك.

// roleUser حسابٌ بدورٍ كانونيٍّ واحدٍ وجلسةٌ دائمة.
func roleUser(t *testing.T, hh *Harness, role string) (u *User, tok string) {
	t.Helper()
	return capUser(t, hh, role)
}

// probe نداءٌ واحدٌ ونتيجتُه: أمُنع أم لا.
type probe struct {
	Name   string
	Method string
	Path   string
	Body   map[string]any
}

// denied **أمُنع هذا النداء؟** — و`403` وحدَها منعٌ.
//
// **و`404`/`409`/`400` نجاحُ تخويلٍ وردُّ منطقٍ** — **ولا تُخلَط.**
func denied(t *testing.T, hh *Harness, tok string, p probe) (bool, int) {
	t.Helper()
	var res Res
	switch p.Method {
	case "GET":
		res = hh.GET(p.Path, tok)
	case "PUT":
		res = hh.Call("PUT", p.Path, tok, p.Body, nil)
	case "PATCH":
		res = hh.PATCH(p.Path, tok, p.Body)
	case "DELETE":
		res = hh.Call("DELETE", p.Path, tok, nil, nil)
	default:
		res = hh.POST(p.Path, tok, p.Body)
	}
	return res.Code == http.StatusForbidden, res.Code
}

// checkRole يفحص دوراً بمجموعتَي مسموحٍ وممنوع.
func checkRole(t *testing.T, hh *Harness, role string, allow, deny []probe) {
	t.Helper()
	_, tok := roleUser(t, hh, role)
	for _, p := range allow {
		if d, code := denied(t, hh, tok, p); d {
			t.Errorf("**`%s` مُنع من %s** (%d) — **وهو من عمله.**",
				role, p.Name, code)
		}
	}
	for _, p := range deny {
		d, code := denied(t, hh, tok, p)
		if !d {
			t.Errorf("**`%s` بلغ %s** (%d) — **وأقلُّ صلاحيّةٍ تمنعه.**",
				role, p.Name, code)
		}
	}
	t.Logf("✓ %-22s مسموحٌ=%d · ممنوعٌ=%d", role, len(allow), len(deny))
}

// probes مجموعةُ نداءاتٍ تمثيليّةٍ لكلّ حدٍّ في المصفوفة.
func probes(t *testing.T, hh *Harness) map[string]probe {
	t.Helper()
	v := hh.NewUser("customer")
	m := hh.Factory().Merchant()
	oid, dr := activeOrderFor(t, hh)
	return map[string]probe{
		"قراءةُ الحسابات": {"قراءةِ الحسابات", "GET", "/api/v1/admin/users?limit=1", nil},
		"قراءةُ الطلبات":  {"قراءةِ الطلبات", "GET", "/api/v1/admin/orders?limit=1", nil},
		"تدخّلٌ في طلب":   {"تدخّلٍ في طلب", "POST", "/api/v1/admin/orders/" + oid + "/transition", map[string]any{"to": "cancelled", "note": "ADG-2"}},
		"قيدُ محفظة":      {"قيدِ محفظة", "POST", "/api/v1/admin/users/" + v.ID + "/wallet", map[string]any{"amount": 100, "kind": "topup", "note": "ADG-2"}},
		"قراءةٌ ماليّة":   {"قراءةٍ ماليّة", "GET", "/api/v1/admin/expenses?limit=1", nil},
		"منحُ دور":        {"منحِ دور", "POST", "/api/v1/admin/users/" + v.ID + "/roles", map[string]any{"role": "finance", "reason": "ADG-2"}},
		"تبديلُ حال":      {"تبديلِ حالِ حساب", "PATCH", "/api/v1/admin/users/" + v.ID, map[string]any{"status": "suspended", "status_reason": "ADG-2"}},
		"تعليقُ متجر":     {"تعليقِ متجر", "POST", "/api/v1/admin/merchants/" + m.ID + "/suspend", map[string]any{"suspended": true, "note": "ADG-2"}},
		"مراجعةُ القوائم": {"مراجعةِ القوائم", "GET", "/api/v1/admin/menu/pending", nil},
		"إدارةُ المتاجر":  {"إدارةِ المتاجر", "PATCH", "/api/v1/admin/merchants/" + m.ID, map[string]any{"name": "ADG-2"}},
		// **وسجلُّ السائقين قراءةٌ، وإنهاءُ الوردية تشغيل** — ولا يُقاسان
		// بمجسٍّ واحد. (مصالحةُ دورةِ ٢٦.)
		"قراءةُ السائقين": {"قراءةِ سجلّ السائقين", "GET", "/api/v1/admin/drivers?limit=1", nil},
		"إنهاءُ وردية":    {"إنهاءِ ورديّةِ سائق", "POST", "/api/v1/admin/drivers/" + dr.ID + "/end-shift", map[string]any{}},
		// ══════════════════════════════════════════════════════════
		// **و«يقرأ» ليست صنفاً واحداً** — مجسٌّ لكلّ حدٍّ مقيس
		// ══════════════════════════════════════════════════════════
		//
		// **التصديرُ إخراجُ القاعدة** · **والمحادثةُ كلامُ الناس** ·
		// **ودفترُ العناوين بيوتُ المرء كلُّها.**
		"تصديرُ الحسابات": {"تصديرِ دليل الحسابات", "GET", "/api/v1/admin/users/export", nil},
		"محادثةُ طلب":     {"محادثةِ الطلب", "GET", "/api/v1/admin/orders/" + oid + "/chat", nil},
		"عناوينُ المرء":   {"دفترِ عناوين المرء", "GET", "/api/v1/admin/users/" + v.ID + "/addresses", nil},
		// **وتصديرُ الطلبات فيه هاتفُ الزبون** — فيُقاس بذاته.
		"تصديرُ الطلبات": {"تصديرِ الطلبات", "GET", "/api/v1/admin/orders/export?from=2026-01-01&to=2026-01-02", nil},
		"الدعم":          {"الدعمِ والتذاكر", "GET", "/api/v1/admin/tickets?limit=1", nil},
		"محتوى":          {"المحتوى", "GET", "/api/v1/admin/banners", nil},
		"إعدادٌ عامّ":    {"إعدادٍ عامّ", "PUT", "/api/v1/admin/settings/orders.auto_transfer", map[string]any{"value": false}},
		"إعدادٌ ماليّ":   {"إعدادٍ ماليّ", "PUT", "/api/v1/admin/settings/merchants.commission_percent", map[string]any{"value": 11}},
		"إعدادٌ أمنيّ":   {"إعدادٍ أمنيّ", "PUT", "/api/v1/admin/settings/security.session_days", map[string]any{"value": 20}},
		"تحليلات":        {"التحليلات", "GET", "/api/v1/admin/stats", nil},
		"سجلُّ التدقيق":  {"سجلِّ التدقيق", "GET", "/api/v1/admin/audit?limit=1", nil},
		"قرارُ سحب":      {"قرارِ سحب", "GET", "/api/v1/admin/payouts?limit=1", nil},
	}
}

// ══════════════════════════════════════════════════════════════════════
// **R1…R21 · مصفوفةُ الأدوار التسعة**
// ══════════════════════════════════════════════════════════════════════
func TestADG2_RoleMatrix(t *testing.T) {
	hh := New(t)
	p := probes(t, hh)

	// ── R1 · المالك: كلُّ شيء ────────────────────────────────────
	checkRole(t, hh, "owner_super_admin", []probe{
		p["قراءةُ الحسابات"], p["قراءةُ الطلبات"], p["قيدُ محفظة"],
		p["منحُ دور"], p["تعليقُ متجر"], p["إعدادٌ ماليّ"],
		p["إعدادٌ أمنيّ"], p["سجلُّ التدقيق"], p["الدعم"],
	}, nil)

	// ── R2+R3+R4 · العمليّات ─────────────────────────────────────
	checkRole(t, hh, "operations", []probe{
		p["قراءةُ الطلبات"], p["تدخّلٌ في طلب"],
		p["قراءةُ الحسابات"], p["قراءةُ السائقين"], p["إنهاءُ وردية"],
	}, []probe{
		p["قيدُ محفظة"], p["منحُ دور"], p["إعدادٌ ماليّ"],
		p["إعدادٌ أمنيّ"], p["تعليقُ متجر"], p["تبديلُ حال"],
		// **ومن يوزّع طلباً قائماً لا يقرأ كلامَ الناس ولا دفترَ
		// بيوتهم ولا يسحب القاعدةَ ملفّاً.**
		p["تصديرُ الطلبات"], p["تصديرُ الحسابات"],
		p["محادثةُ طلب"], p["عناوينُ المرء"],
	})

	// ── R5+R6+R7 · دعمُ الزبائن ──────────────────────────────────
	checkRole(t, hh, "customer_support", []probe{
		p["الدعم"], p["قراءةُ الطلبات"], p["قراءةُ الحسابات"],
		// **والشكوى تُقرأ من كلامها، والتسليمُ من عنوانه.**
		p["محادثةُ طلب"], p["عناوينُ المرء"],
	}, []probe{
		p["قيدُ محفظة"], p["منحُ دور"], p["إعدادٌ ماليّ"],
		p["إعدادٌ أمنيّ"], p["تعليقُ متجر"], p["تبديلُ حال"],
		p["تصديرُ الحسابات"], p["تصديرُ الطلبات"], p["قراءةٌ ماليّة"],
	})

	// ── R8+R9 · توثيقُ السائقين ──────────────────────────────────
	checkRole(t, hh, "driver_verification", []probe{
		p["قراءةُ السائقين"],
	}, []probe{
		p["قيدُ محفظة"], p["منحُ دور"], p["تعليقُ متجر"],
		p["إعدادٌ ماليّ"], p["مراجعةُ القوائم"],
		// **ومن وُظّف للتوثيق لا يُخرج سائقاً من عمله.**
		p["إنهاءُ وردية"],
		// **و`GET /drivers` تُخرج الاسمَ والهاتفَ والحال** — **فلا
		// يحتاج دليلَ الحسابات، ولو ناله لَورث تصديرَه وعناوينَه.**
		p["قراءةُ الحسابات"], p["تصديرُ الحسابات"],
		p["عناوينُ المرء"], p["محادثةُ طلب"],
	})

	// ── R10+R11 · توثيقُ المتاجر ─────────────────────────────────
	checkRole(t, hh, "merchant_verification", []probe{
		p["مراجعةُ القوائم"],
	}, []probe{
		p["قيدُ محفظة"], p["منحُ دور"], p["إدارةُ المتاجر"],
		p["قراءةُ السائقين"], p["تعليقُ متجر"],
		// **ولا مسارَ مراجعةٍ يقرأ طلباً ولا دليلَ حسابات.**
		p["قراءةُ الطلبات"], p["قراءةُ الحسابات"],
	})

	// ── R12+R13+R14+R15 · الماليّة ───────────────────────────────
	checkRole(t, hh, "finance", []probe{
		p["قيدُ محفظة"], p["قراءةٌ ماليّة"], p["قرارُ سحب"],
		p["إعدادٌ ماليّ"], p["قراءةُ الطلبات"],
		// **وكشفُ المحاسبة عملُها المكتوب.**
		p["تصديرُ الطلبات"],
	}, []probe{
		p["منحُ دور"], p["إعدادٌ أمنيّ"], p["تعليقُ متجر"],
		p["تبديلُ حال"], p["تدخّلٌ في طلب"],
		// **ولا دليلَ حساباتٍ تُصدّره.**
		p["تصديرُ الحسابات"],
	})

	// ── R16+R17 · الثقةُ والسلامة ────────────────────────────────
	checkRole(t, hh, "trust_safety", []probe{
		p["تعليقُ متجر"], p["تبديلُ حال"], p["قراءةُ الحسابات"],
		p["سجلُّ التدقيق"],
		// **والتحقيقُ يقرأ الكلامَ والأثر.**
		p["محادثةُ طلب"], p["عناوينُ المرء"],
	}, []probe{
		p["قيدُ محفظة"], p["منحُ دور"], p["إعدادٌ ماليّ"],
		p["إعدادٌ أمنيّ"], p["محتوى"],
		// **ودورٌ أمنيٌّ لا يُمنَح سحبَ القاعدة لأنّه أمنيّ.**
		p["تصديرُ الحسابات"], p["تصديرُ الطلبات"],
	})

	// ── R18+R19 · التسويقُ والمحتوى ──────────────────────────────
	checkRole(t, hh, "marketing_content", []probe{
		p["محتوى"],
	}, []probe{
		p["إعدادٌ ماليّ"], p["إعدادٌ أمنيّ"], p["قيدُ محفظة"],
		p["منحُ دور"], p["تبديلُ حال"], p["تعليقُ متجر"],
		// **والإعدادُ العامُّ يحكم المناطقَ ومضلَّعاتِ التغطية.**
		p["إعدادٌ عامّ"],
	})

	// ── R20+R21 · التحليلُ — قراءةٌ محضة ─────────────────────────
	checkRole(t, hh, "analytics", []probe{
		p["تحليلات"],
	}, []probe{
		p["قيدُ محفظة"], p["منحُ دور"], p["تبديلُ حال"],
		p["تعليقُ متجر"], p["تدخّلٌ في طلب"], p["إدارةُ المتاجر"],
		p["محتوى"], p["إعدادٌ عامّ"], p["إعدادٌ ماليّ"], p["إعدادٌ أمنيّ"],
		// ══════════════════════════════════════════════════════════
		// **و«يقرأ فقط» ليست «أقلَّ صلاحيّة»**
		// ══════════════════════════════════════════════════════════
		//
		// **قراءةُ الطلبات تفتح محادثاتِها** · **وقراءةُ الحسابات
		// تفتح دليلَ الناس وعناوينَهم** · **والقراءةُ الماليّةُ تفتح
		// محافظَ الأفراد** · **والتصديرُ يُخرج أرقامَ الهواتف.**
		//
		// **ولا واحدةَ منها تقرير.**
		p["قراءةُ الطلبات"], p["قراءةُ الحسابات"],
		p["قراءةٌ ماليّة"], p["تصديرُ الطلبات"],
		p["تصديرُ الحسابات"], p["محادثةُ طلب"], p["عناوينُ المرء"],
	})
}

// ══════════════════════════════════════════════════════════════════════
// **R24 · اتّحادُ دورين مشروعَين**
// ══════════════════════════════════════════════════════════════════════
func TestADG2_R24_MultiRoleUnion(t *testing.T) {
	hh := New(t)
	p := probes(t, hh)
	_, tok := capUser(t, hh, "finance", "analytics")

	for _, want := range []probe{p["قيدُ محفظة"], p["تحليلات"], p["قراءةٌ ماليّة"]} {
		if d, code := denied(t, hh, tok, want); d {
			t.Errorf("**الاتّحادُ فقد %s** (%d)", want.Name, code)
		}
	}
	// **ولا يجمع الاتّحادُ ما لا يملكه أحدُهما.**
	for _, no := range []probe{p["منحُ دور"], p["إعدادٌ أمنيّ"], p["تعليقُ متجر"]} {
		if d, _ := denied(t, hh, tok, no); !d {
			t.Errorf("**الاتّحادُ نال %s ولا يملكه دورٌ منهما**", no.Name)
		}
	}
	t.Log("✓ R24 اتّحادُ `finance`+`analytics` — ولا زيادة")
}

// ══════════════════════════════════════════════════════════════════════
// **R28 · والأدوارُ القديمةُ تحتفظ بوصولها المُثبَت**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا تُحذَف ولا تُرقّى** (قرارُ المالك) — **حتّى يقع ترحيلُ
// الحسابات قبل الإنتاج.**
func TestADG2_R28_LegacyRolesKeepDocumentedAccess(t *testing.T) {
	hh := New(t)
	p := probes(t, hh)

	checkRole(t, hh, "admin", []probe{
		p["قراءةُ الحسابات"], p["قيدُ محفظة"], p["منحُ دور"],
		p["إعدادٌ ماليّ"], p["إعدادٌ أمنيّ"], p["سجلُّ التدقيق"],
	}, nil)

	checkRole(t, hh, "ops", []probe{
		p["قراءةُ الطلبات"], p["تدخّلٌ في طلب"], p["إدارةُ المتاجر"],
	}, []probe{p["قيدُ محفظة"], p["منحُ دور"], p["إعدادٌ ماليّ"]})

	// **و`sales` ليس دورَ سطحٍ إداريّ** — لا صفَّ له في القدرات.
	checkRole(t, hh, "sales", nil, []probe{
		p["قراءةُ الحسابات"], p["قيدُ محفظة"], p["محتوى"],
	})
}

// ══════════════════════════════════════════════════════════════════════
// **M1…M12 · إدارةُ الأدوار والقدرات من اللوحة**
// ══════════════════════════════════════════════════════════════════════
func TestADG2_Management(t *testing.T) {
	hh := New(t)
	_, owner := capUser(t, hh, "owner_super_admin")
	_, weak := capUser(t, hh, "operations")
	victim := hh.NewUser("customer")
	capRole(t, hh, "qa_managed")

	// ── M1 · قائمةُ الأدوار ──────────────────────────────────────
	list := hh.GET("/api/v1/admin/roles", owner)
	t.Logf("M1: قائمةُ الأدوار ⇒ %d", list.Code)
	if list.Code != http.StatusOK {
		t.Fatalf("**قائمةُ الأدوار لم تُقرأ** (%s)", list)
	}

	// ── M2 · تفصيلُ دور ─────────────────────────────────────────
	det := hh.GET("/api/v1/admin/roles/qa_managed", owner)
	t.Logf("M2: تفصيلُ دور ⇒ %d", det.Code)
	if det.Code != http.StatusOK {
		t.Errorf("**تفصيلُ الدور لم يُقرأ** (%d)", det.Code)
	}

	// ── والمعجمُ يُقرأ ولا يُحرَّر ───────────────────────────────
	cat := hh.GET("/api/v1/admin/capabilities", owner)
	t.Logf("M2: المعجمُ ⇒ %d", cat.Code)
	if cat.Code != http.StatusOK {
		t.Errorf("**المعجمُ لم يُقرأ** (%d)", cat.Code)
	}

	// ── M2+M9 · منحُ قدرةٍ — ومكرَّرٌ آمن ────────────────────────
	g1 := hh.POST("/api/v1/admin/roles/qa_managed/capabilities", owner,
		map[string]any{"capability": string(authz.FinanceManage)})
	g2 := hh.POST("/api/v1/admin/roles/qa_managed/capabilities", owner,
		map[string]any{"capability": string(authz.FinanceManage)})
	t.Logf("M2/M9: منحٌ ⇒ %d · مكرَّرٌ ⇒ %d", g1.Code, g2.Code)
	if g1.Code != http.StatusOK || g2.Code != http.StatusOK {
		t.Errorf("**المنحُ أو تكرارُه أخفق**: %d · %d", g1.Code, g2.Code)
	}

	// ── M8 · ومجهولةٌ تُرفَض ────────────────────────────────────
	bad := hh.POST("/api/v1/admin/roles/qa_managed/capabilities", owner,
		map[string]any{"capability": "لا.وجودَ.لها"})
	t.Logf("M8: قدرةٌ مجهولة ⇒ %d", bad.Code)
	if bad.Code < 400 {
		t.Errorf("**قُبلت قدرةٌ مجهولة** (%d)", bad.Code)
	}

	// ── M2 نافذٌ فوراً ──────────────────────────────────────────
	_, holder := capUser(t, hh, "qa_managed")
	money := probe{"قيدِ محفظة", "POST",
		"/api/v1/admin/users/" + victim.ID + "/wallet",
		map[string]any{"amount": 100, "kind": "topup", "note": "ADG-2"}}
	if d, code := denied(t, hh, holder, money); d {
		t.Errorf("**القدرةُ المُنحت لم تسرِ** (%d)", code)
	}

	// ── M3+M10 · النزعُ نافذٌ فوراً ──────────────────────────────
	rev := hh.Call("DELETE",
		"/api/v1/admin/roles/qa_managed/capabilities/"+string(authz.FinanceManage),
		owner, nil, nil)
	d, code := denied(t, hh, holder, money)
	t.Logf("M3/M10: النزعُ ⇒ %d · ثمّ الفعلُ ⇒ %d", rev.Code, code)
	if rev.Code != http.StatusOK {
		t.Errorf("**النزعُ أخفق** (%d)", rev.Code)
	}
	if !d {
		t.Errorf("**قدرةٌ نُزعت ما زالت تعمل** (%d)", code)
	}

	// ── M4+M5 · منحُ دورٍ لحسابٍ ونزعُه ──────────────────────────
	gr := hh.POST("/api/v1/admin/users/"+victim.ID+"/roles", owner,
		map[string]any{"role": "analytics", "reason": "ADG-2"})
	rr := hh.Call("DELETE", "/api/v1/admin/users/"+victim.ID+"/roles/analytics",
		owner, nil, nil)
	t.Logf("M4/M5: منحُ دورٍ ⇒ %d · نزعُه ⇒ %d", gr.Code, rr.Code)
	if gr.Code >= 400 || rr.Code >= 400 {
		t.Errorf("**إدارةُ أدوار الحساب أخفقت**: %d · %d", gr.Code, rr.Code)
	}

	// ── M6+M7+M12 · ولا يُدير غيرُ المخوَّل ولا يُصعّد نفسَه ─────
	m6 := hh.POST("/api/v1/admin/roles/qa_managed/capabilities", weak,
		map[string]any{"capability": string(authz.RolesManage)})
	m7 := hh.POST("/api/v1/admin/users/"+victim.ID+"/roles", weak,
		map[string]any{"role": "owner_super_admin", "reason": "ADG-2"})
	var leaked bool
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT EXISTS (SELECT 1 FROM role_capabilities
		                WHERE role_code='qa_managed' AND capability_code=$1)`,
		string(authz.RolesManage)).Scan(&leaked); err != nil {
		t.Fatalf("قراءةُ القدرات: %v", err)
	}
	t.Logf("M6: غيرُ مخوَّلٍ يمنح قدرة ⇒ %d · M7: يُصعّد نفسَه ⇒ %d · تسرّبٌ=%v",
		m6.Code, m7.Code, leaked)
	if m6.Code != http.StatusForbidden || m7.Code != http.StatusForbidden {
		t.Errorf("**غيرُ المخوَّل أدار الأدوار**: %d · %d", m6.Code, m7.Code)
	}
	if leaked {
		t.Error("**M12: مُنع الفعلُ ووقع تبديلُه**")
	}

	// ── M11 · وتبديلُ سياسةِ الأمن يُقيَّد ───────────────────────
	n := auditCount(t, hh, "admin.role_capability_grant", "qa_managed")
	t.Logf("M11: آثارُ منحِ القدرات = %d", n)
	if n == 0 {
		t.Error("**تبديلُ سياسةِ تخويلٍ بلا أثر** — **و`AQ-4` مغلقة.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **C3…C6 · تزامنُ تبديلاتِ التخويل**
// ══════════════════════════════════════════════════════════════════════
func TestADG2_ConcurrentPolicyChanges(t *testing.T) {
	hh := New(t)
	victim := hh.NewUser("customer")
	capRole(t, hh, "qa_c", authz.FinanceManage)
	u, tok := capUser(t, hh, "qa_c")
	money := probe{"قيدِ محفظة", "POST",
		"/api/v1/admin/users/" + victim.ID + "/wallet",
		map[string]any{"amount": 100, "kind": "topup", "note": "ADG-2"}}

	// ── C4 · نزعُ قدرةٍ يتزامن مع فعل ───────────────────────────
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "فعلٌ مخوَّل", Do: func(context.Context) any {
			d, c := denied(t, hh, tok, money)
			return []any{d, c}
		}},
		Actor{Name: "نزعُ القدرة", Do: func(context.Context) any {
			_, err := hh.Pool.Exec(ctxBG(),
				`DELETE FROM role_capabilities WHERE role_code='qa_c'`)
			return err
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	d, code := denied(t, hh, tok, money)
	t.Logf("C4: بعد ثبوت النزع ⇒ %d — %s", code, r)
	if !d {
		t.Errorf("**فعلٌ مرّ بعد ثبوت نزع القدرة** (%d)", code)
	}

	// ── C3 · ومنحٌ يتزامن ⇒ يسري بعد الثبوت ─────────────────────
	if _, err := hh.Pool.Exec(ctxBG(), `
		INSERT INTO role_capabilities (role_code, capability_code)
		VALUES ('qa_c', $1) ON CONFLICT DO NOTHING`,
		string(authz.FinanceManage)); err != nil {
		t.Fatalf("المنح: %v", err)
	}
	d2, code2 := denied(t, hh, tok, money)
	t.Logf("C3: بعد ثبوت المنح ⇒ %d", code2)
	if d2 {
		t.Errorf("**منحٌ ثبت ولم يسرِ** (%d)", code2)
	}

	// ── C6 · وتبديلُ دورِ الحساب مع تبديل قدرات الدور ───────────
	if _, err := hh.Pool.Exec(ctxBG(),
		`DELETE FROM user_roles WHERE user_id=$1::uuid AND role_code='qa_c'`,
		u.ID); err != nil {
		t.Fatalf("نزعُ الدور: %v", err)
	}
	d3, code3 := denied(t, hh, tok, money)
	t.Logf("C6: نُزع الدورُ والقدرةُ قائمة ⇒ %d", code3)
	if !d3 {
		t.Errorf("**نُزع الدورُ والقدرةُ ما زالت تعمل** (%d)", code3)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **حارسُ التغطية: لا مسارَ إداريٍّ بلا سياسةٍ أو استثناءٍ مكتوب**
// ══════════════════════════════════════════════════════════════════════
//
// **ومشّاءٌ على الموجِّه لا بحثٌ في نصّ** — **فمسارٌ يُضاف غداً بلا
// تصنيفٍ يُسقط البناء.**
// ══════════════════════════════════════════════════════════════════════
// **وقدرةٌ لا مسارَ لها** — `ADG-2` · جولةُ دورةِ ٢٦ الثانية
// ══════════════════════════════════════════════════════════════════════
//
// **حارسان يمشيان بين الموجِّه والجدول** — **ولا ثالثَ يمشي بين
// الجدول والمعجم.**
//
// **وقدرةٌ في المعجم لا يحرسها مسارٌ تُعرَض في اللوحة فتُمنَح لدور**،
// **ويظنّ مانحُها أنّه أعطى شيئاً.** **وهي لا تفتح باباً** — **لكنّها
// تُقرأ سياسةً وهي اسم.**
func TestADG2_EveryCapabilityHasARoute(t *testing.T) {
	used := map[authz.Capability]bool{}
	for _, r := range authz.Rules() {
		used[r.Need] = true
	}
	// **وإعداداتُ الأثر تُحسَم في المعالِج لا في الجدول** — المسارُ
	// `PUT /settings/{key}` مستثنىً عمداً، **وقدرتُه تتبع المفتاح.**
	for _, c := range []authz.Capability{
		authz.SettingsGeneralManage,
		authz.SettingsFinancialManage,
		authz.SettingsSecurityManage,
	} {
		used[c] = true
	}
	var orphans []string
	for _, c := range authz.All() {
		if !used[c] {
			orphans = append(orphans, string(c))
		}
	}
	t.Logf("المعجمُ=%d · بلا مسارٍ=%d", authz.Count(), len(orphans))
	for _, c := range orphans {
		t.Errorf("**قدرةٌ لا يحرسها مسار**: `%s` — **تُقرأ سياسةً وهي "+
			"اسم.** (`ADG-2`)", c)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **وسطرُ سياسةٍ لا مسارَ له** — `ADG-2` · مصالحةُ دورةِ ٢٦
// ══════════════════════════════════════════════════════════════════════
//
// # المسألة
//
// **الحارسُ الأوّلُ يمشي من المسار إلى السياسة** — فيمسك مساراً بلا
// حراسة. **ولا يمسك العكس**: **سطراً في الجدول لا مسارَ له.**
//
// **وخمسةٌ منها كانت** (`/demand` · `/meta` · `/opportunities` ·
// `/reps` · `/search`) — **كُتبت من ذاكرةِ مسارٍ لا من الموجِّه.**
//
// # ولماذا يُهمّ وهي لا تمنح شيئاً
//
// **الجدولُ يُقرأ عقداً** — **ويُراجَع بعينٍ بشريّةٍ حين يُسأل «من يبلغ
// ماذا».** **وسطرٌ يصف مساراً لا وجودَ له يُضلّل المراجع**، **وأسوأُ
// منه أن يُنشأ المسارُ يوماً بغير القدرة المكتوبة فيُظنّ محروساً.**
func TestADG2_EveryPolicyRuleHasARoute(t *testing.T) {
	hh := New(t)
	live := map[string][]string{}
	err := chi.Walk(hh.API.RouterForWalk(),
		func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
			const prefix = "/api/v1/admin"
			if len(route) > len(prefix) && route[:len(prefix)] == prefix {
				live[method] = append(live[method], route[len(prefix):])
			}
			return nil
		})
	if err != nil {
		t.Fatalf("المشّاء: %v", err)
	}
	same := func(policy, actual string) bool {
		ps := strings.Split(strings.Trim(policy, "/"), "/")
		as := strings.Split(strings.Trim(actual, "/"), "/")
		if len(ps) != len(as) {
			return false
		}
		for i, seg := range ps {
			if strings.HasPrefix(seg, "{") {
				continue
			}
			if seg != as[i] {
				return false
			}
		}
		return true
	}
	var orphans []string
	for _, rule := range authz.Rules() {
		hit := false
		for method, routes := range live {
			if rule.Method != "" && rule.Method != method {
				continue
			}
			for _, rt := range routes {
				if same(rule.Pattern, rt) {
					hit = true
				}
			}
		}
		if !hit {
			m := rule.Method
			if m == "" {
				m = "*"
			}
			orphans = append(orphans, m+" "+rule.Pattern)
		}
	}
	t.Logf("سطورُ الجدول=%d · بلا مسارٍ=%d", authz.PolicyCount(), len(orphans))
	for _, o := range orphans {
		t.Errorf("**سطرُ سياسةٍ لا مسارَ له**: %s — **يُقرأ عقداً وهو وهم.** (`ADG-2`)", o)
	}
}

func TestADG2_EveryAdminRouteClassified(t *testing.T) {
	hh := New(t)
	var checked, exempt int
	var missing []string

	err := chi.Walk(hh.API.RouterForWalk(),
		func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
			const prefix = "/api/v1/admin"
			if len(route) < len(prefix) || route[:len(prefix)] != prefix {
				return nil
			}
			pattern := route[len(prefix):]
			if pattern == "" {
				return nil
			}
			if _, ok := authz.IsExempt(pattern); ok {
				exempt++
				return nil
			}
			if _, ok := authz.LookupAdmin(method, pattern); !ok {
				missing = append(missing, method+" "+pattern)
				return nil
			}
			checked++
			return nil
		})
	if err != nil {
		t.Fatalf("المشّاء: %v", err)
	}
	t.Logf("مساراتٌ مصنَّفةٌ=%d · مستثناةٌ=%d · بلا سياسةٍ=%d · جدولٌ=%d سطراً",
		checked, exempt, len(missing), authz.PolicyCount())
	for _, m := range missing {
		t.Errorf("**مسارٌ إداريٌّ بلا سياسةٍ ولا استثناء**: %s — "+
			"**ولا يُقرأ الصمتُ إذناً.** (`ADG-2`)", m)
	}
}
