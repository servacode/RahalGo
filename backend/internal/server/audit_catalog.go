package server

// ══════════════════════════════════════════════════════════════════════
// **تصنيفُ أفعال السجلّ — مكتوبٌ مرّةً هنا**
// ══════════════════════════════════════════════════════════════════════
//
// (قراراتُ المالك ٢٠٢٦-١٠-٠٤ على فحص «سجل الأحداث».)
//
// يقرؤه ثلاثة: **صفحةُ السجلّ** (تبويباتُها)، **والتصدير** (المرشّحاتُ
// نفسُها)، **وعاملُ الحفظ** (ما يُحذف بعد مدّة). **وثلاثُ قوائمَ تفترق
// يوماً فيقول التبويبُ شيئاً ويحذف العاملُ شيئاً آخر.**

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

// auditSessionActions **أفعالُ الدخول والجلسة** — تُحذف بعد مدّة الحفظ.
//
// (قرارُ المالك الثاني: «المالُ والصلاحيّاتُ للأبد، والدخولُ تسعون يوماً».)
//
// **ونسختُها في القاعدة** (`audit_session_actions()` في الهجرة ٠٢٢٠) —
// **والحارسُ هناك يقرأ تلك**، وفحصٌ يُسقط البناءَ إن افترقتا.
//
// **وليس منها** تغييرُ كلمة السرّ أو الرقم أو رمز الدخول، ولا التسجيل:
// تلك تغييراتٌ على الحساب تبقى.
var auditSessionActions = []string{
	"auth.otp_login", "auth.password_login", "auth.pin_login",
	"auth.pin_verified", "auth.refresh", "auth.logout",
	"auth.password_failed", "auth.sso", "auth.whatsapp_verified",
}

// auditGroup تبويبٌ في صفحة السجلّ.
type auditGroup struct {
	Key string
	// Prefixes بادئاتُ الأفعال التي يجمعها — **وفارغٌ لتبويبٍ له شرطُه**.
	Prefixes []string
}

// auditGroups **كلُّ بادئةٍ تكتبها الشيفرة لها تبويب** — ويحرسه فحص.
//
// (المشكلةُ الخامسة: أحداثُ السائق ١١١ سطراً على التجهيز ولا تبويبَ لها.)
var auditGroups = []auditGroup{
	{Key: "sensitive"},
	{Key: "all"},
	{Key: "login", Prefixes: []string{"auth"}},
	{Key: "finance", Prefixes: []string{"finance"}},
	{Key: "ops", Prefixes: []string{"ops"}},
	{Key: "admin", Prefixes: []string{"admin"}},
	{Key: "order", Prefixes: []string{"order"}},
	{Key: "driver", Prefixes: []string{"driver"}},
	{Key: "merchant", Prefixes: []string{"merchant"}},
	{Key: "customer", Prefixes: []string{"customer"}},
	{Key: "user", Prefixes: []string{"user"}},
	{Key: "menu", Prefixes: []string{"menu"}},
	{Key: "catalog", Prefixes: []string{"catalog", "market"}},
	{Key: "platform", Prefixes: []string{"platform"}},
	{Key: "geo", Prefixes: []string{"coverage", "coverage_request", "operational_area", "branch", "expansion"}},
}

// auditGroupByKey التبويبُ باسمه — **والاسمُ القديمُ `auth` يُقرأ «الدخول».**
func auditGroupByKey(key string) (auditGroup, bool) {
	if key == "auth" {
		key = "login"
	}
	for _, g := range auditGroups {
		if g.Key == key {
			return g, true
		}
	}
	return auditGroup{}, false
}

// auditSensitiveExtra **أفعالٌ حسّاسةٌ ليست في معاجم التأكيد ولا الصنف `A`.**
//
// (قرارُ المالك الثالث: «الأفعال الحساسة» = المالُ والصلاحيّاتُ والإعداداتُ
// الحسّاسةُ والإيقافُ والطوارئ.)
var auditSensitiveExtra = []string{
	// ── الصلاحيّاتُ والحسابات ────────────────────────────────────
	"admin.owner_bootstrap", "admin.logout_all", "admin.password_reset",
	"admin.whatsapp_pair", "admin.whatsapp_unpair", "user.self_delete",
	"auth.phone_change", "auth.password_reset",
	// ── الإيقافُ والإغلاق ─────────────────────────────────────────
	"ops.merchant_suspend", "admin.platform_closure",
	"merchant.emergency_close", "merchant.emergency_reopen",
	// ── الطوارئ ───────────────────────────────────────────────────
	"driver.emergency", "ops.emergency_ack", "ops.emergency_resolved",
	"ops.store_emergency_ack",
	"ops.emergency_driver_ok", "ops.emergency_outcome", "ops.emergency_money", "ops.emergency_note",
	// ── مالٌ أو أثرُه ─────────────────────────────────────────────
	"admin.merchant_rep_transfer", "merchant.settlement_update",
	"admin.launch_preset",
	// ── والتصديرُ نفسُه (القرارُ السادس) ──────────────────────────
	"admin.audit_exported",
}

// auditSensitiveActions **قائمةُ «الأفعال الحساسة» بالاسم** — والمالُ كلُّه
// (`finance.*`) يُضاف بالبادئة، **والإعدادُ بمفتاحه** (`auditSensitiveSettingKeys`).
//
// **مشتقّةٌ من المعاجم القائمة لا مكتوبةٌ ثانية**: أفعالُ التأكيد
// (`authz.SensitiveActions`) · الصنفُ `A` هنا وفي الهويّة · وما فوق.
func auditSensitiveActions() []string {
	set := map[string]bool{}
	for _, a := range authz.SensitiveActions() {
		set[a.Action] = true
	}
	for a := range criticalAuditActions {
		set[a] = true
	}
	for a := range identity.CriticalActions {
		set[a] = true
	}
	for _, a := range auditSensitiveExtra {
		set[a] = true
	}
	// **والإعدادُ ليس حسّاساً باسمه** — نصُّ صفحةٍ ليس عمولة.
	delete(set, "admin.setting_update")
	out := make([]string, 0, len(set))
	for a := range set {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// auditSensitiveSettingKeys **مفاتيحُ الإعداد الحسّاسة** — بتصنيف دورةِ ٢١
// نفسِه (`criticalSettingKey`)، فلا يُخترَع تصنيفٌ ثانٍ.
func auditSensitiveSettingKeys() []string {
	var out []string
	for _, d := range settings.Catalog {
		if criticalSettingKey(d.Key) {
			out = append(out, d.Key)
		}
	}
	return out
}

// ══════════════════════════════════════════════════════════════════════
// **المبالغُ تُخفى عمّن ليس طرفاً في المال** — القرارُ الأوّل
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك: «الكلّ يشوف السجلّ، بس المبالغ تنخفى عن غير المالية
//  والأدمن».)
//
// **وتُخفى في الخادم لا في الشاشة** — شاشةٌ تُخفي ما وصلها قد وصلها؛ ومن
// فتح أدوات المتصفّح قرأه.

// canSeeAuditMoney **أيرى هذا الموظّفُ المبالغ؟** — من ملك قراءةَ المال،
// أو كان أدمنَ المنصّة أو مالكَها.
func (s *Server) canSeeAuditMoney(r *http.Request) bool {
	if s.hasCapability(r, authz.FinanceRead) {
		return true
	}
	roles, _ := r.Context().Value(ctxRoles).([]string)
	return hasRole(roles, "admin") || hasRole(roles, "owner_super_admin")
}

// auditMoneyKey **أهذا الحقلُ مبلغ؟** — بالاسم: الحقولُ المكتوبةُ في
// تفاصيل السجلّ اليوم (`amount` · `compensation` · `old_total` …) وما يشبهها
// غداً.
func auditMoneyKey(k string) bool {
	switch k {
	case "amount", "compensation", "balance", "total", "fee", "price", "cost",
		"limit", "delta", "percent", "commission", "debt", "credit", "salary":
		return true
	}
	for _, part := range []string{"amount", "total", "fee", "balance", "price",
		"cost", "goods", "compensation", "held", "credit", "debt", "commission"} {
		if strings.Contains(k, part) {
			return true
		}
	}
	return false
}

// redactAuditMoney يحذف المبالغَ من التفاصيل — **ويقول هل حذف.**
//
// **والإعدادُ الماليُّ يُخفى قبله وبعده** — عمولةُ المنصّة رقمٌ ماليّ وإن
// لم يُسمَّ `amount`.
func redactAuditMoney(action, entityID string, raw json.RawMessage) (json.RawMessage, bool) {
	if len(raw) == 0 {
		return raw, false
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw, false
	}
	hit := false
	if action == "admin.setting_update" && criticalSettingKey(entityID) &&
		!strings.HasPrefix(entityID, "security.") && !updateGateSetting(entityID) {
		if m, ok := v.(map[string]any); ok {
			for _, k := range []string{"before", "after", "value"} {
				if _, ok := m[k]; ok {
					delete(m, k)
					hit = true
				}
			}
		}
	}
	var walk func(x any)
	walk = func(x any) {
		switch t := x.(type) {
		case map[string]any:
			for k, val := range t {
				if auditMoneyKey(k) {
					switch val.(type) {
					case float64, string:
						delete(t, k)
						hit = true
						continue
					}
				}
				walk(val)
			}
		case []any:
			for _, e := range t {
				walk(e)
			}
		}
	}
	walk(v)
	if !hit {
		return raw, false
	}
	out, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`), true
	}
	return out, true
}
