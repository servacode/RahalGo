package qa

import (
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **مصفوفةُ الصلاحيّات كما هي — قياسٌ لا نصّ** — `RBAC-01` · `AQ-1`
// ══════════════════════════════════════════════════════════════════════
//
// **مستندُ القرار كُتب قبل أشهر** — **والمصدرُ يتحرّك.** **فتُقاس كلُّ
// خانةٍ بنداءٍ حقيقيّ** لا تُقرأ من جدولٍ في وثيقة.

// roleToken حسابٌ بدورٍ واحدٍ وجلسةٌ دائمةٌ له.
func roleToken(t *testing.T, hh *Harness, role string) (u *User, tok string) {
	t.Helper()
	u, tok, _ = staffToken(t, hh, role)
	return u, tok
}

// cell خانةٌ في المصفوفة: دورٌ وقدرةٌ ونداء.
type rbacCell struct {
	Role   string
	Cap    string
	Method string
	Path   string
	Body   map[string]any
	// Deny **أيُمنَع هذا؟** — مثبَّتٌ بما قيس، فلا يتوسّع دورٌ صامتاً.
	Deny bool
}

// TestRBAC_Matrix_Measured **المصفوفةُ الحاليّةُ مقيسةً بنداءٍ حقيقيّ.**
//
// # وما تُثبته
//
// **الخانةُ التي تسمّيها البوّابةُ بعينها**: «موظّفُ تشغيلٍ يُمنَع من
// تبديل إعدادٍ ماليٍّ حسّاسٍ **من الخادم**» — **ممنوعةٌ اليوم.**
//
// **ومستندُ `AQ-1` يقول خلافَ ذلك** («ولا `RequireRoles("admin")` على
// `PUT /admin/settings/{key}`») — **وكُتب قبل أشهر، والمسارُ انتقل
// إلى مجموعة الأدمن منذئذٍ.** **فالمصدرُ يسبق النصّ.**
//
// # وحارسٌ ضدّ التوسيع
//
// **كلُّ منعٍ مقيسٍ هنا يُثبَّت** — **فمن وسّع دوراً غداً سقط بناؤه**،
// ولا يُكتشَف الأمرُ بمراجعةٍ يدويّة.
func TestRBAC_Matrix_Measured(t *testing.T) {
	hh := New(t)
	victim := hh.NewUser("customer")

	cells := []rbacCell{
		// ── الإعداداتُ الحسّاسةُ ماليّاً ───────────────────────────
		{"ops", "settings.sensitive", "PUT",
			"/api/v1/admin/settings/merchants.commission_percent",
			map[string]any{"value": 11}, true},
		{"finance", "settings.sensitive", "PUT",
			"/api/v1/admin/settings/merchants.commission_percent",
			map[string]any{"value": 11}, true},
		{"admin", "settings.sensitive", "PUT",
			"/api/v1/admin/settings/merchants.commission_percent",
			map[string]any{"value": 11}, false},

		// ── إدارةُ الأدوار ────────────────────────────────────────
		{"ops", "roles.manage", "POST",
			"/api/v1/admin/users/" + victim.ID + "/roles",
			map[string]any{"role": "finance", "reason": "RBAC-01"}, true},
		{"finance", "roles.manage", "POST",
			"/api/v1/admin/users/" + victim.ID + "/roles",
			map[string]any{"role": "admin", "reason": "RBAC-01"}, true},

		// ── المالُ: قيدُ محفظة ─────────────────────────────────────
		{"ops", "finance.manage", "POST",
			"/api/v1/admin/users/" + victim.ID + "/wallet",
			map[string]any{"amount": 1000, "kind": "topup", "note": "RBAC-01"}, true},
		{"finance", "finance.manage", "POST",
			"/api/v1/admin/users/" + victim.ID + "/wallet",
			map[string]any{"amount": 1000, "kind": "topup", "note": "RBAC-01"}, false},

		// ── تبديلُ حالِ حساب ──────────────────────────────────────
		{"ops", "users.manage", "PATCH",
			"/api/v1/admin/users/" + victim.ID,
			map[string]any{"status": "suspended", "status_reason": "RBAC-01"}, true},
		{"finance", "users.manage", "PATCH",
			"/api/v1/admin/users/" + victim.ID,
			map[string]any{"status": "suspended", "status_reason": "RBAC-01"}, true},

		// ── قراءةٌ إداريّةٌ عامّة ──────────────────────────────────
		{"ops", "users.read", "GET", "/api/v1/admin/users?limit=1", nil, false},
		{"finance", "users.read", "GET", "/api/v1/admin/users?limit=1", nil, false},

		// ── ودورٌ غيرُ إداريٍّ أصلاً ───────────────────────────────
		{"driver", "users.read", "GET", "/api/v1/admin/users?limit=1", nil, true},
		{"sales", "settings.sensitive", "PUT",
			"/api/v1/admin/settings/merchants.commission_percent",
			map[string]any{"value": 11}, true},
	}

	t.Logf("%-8s %-20s %-6s %s", "الدور", "القدرة", "الردّ", "الحكم")
	for _, c := range cells {
		_, tok := roleToken(t, hh, c.Role)
		var res Res
		switch c.Method {
		case "GET":
			res = hh.GET(c.Path, tok)
		case "PUT":
			res = hh.Call("PUT", c.Path, tok, c.Body, nil)
		case "PATCH":
			res = hh.PATCH(c.Path, tok, c.Body)
		default:
			res = hh.POST(c.Path, tok, c.Body)
		}
		verdict := "مسموحٌ الآن"
		if res.Code == http.StatusForbidden || res.Code == http.StatusUnauthorized {
			verdict = "ممنوعٌ الآن"
		} else if res.Code >= 400 {
			verdict = "رُدَّ لسببٍ آخر"
		}
		t.Logf("%-8s %-20s %-6d %s", c.Role, c.Cap, res.Code, verdict)

		if c.Deny && verdict != "ممنوعٌ الآن" {
			t.Errorf("**`%s` يبلغ `%s`** (%d) — **وأقلُّ صلاحيّةٍ تُفرَض "+
				"في الخادم لا في الواجهة.** (`RBAC-01` · `AQ-1`)",
				c.Role, c.Cap, res.Code)
		}
		if !c.Deny && verdict == "ممنوعٌ الآن" {
			t.Errorf("**`%s` فقد `%s`** (%d) — **وتضييقٌ غيرُ مقصودٍ "+
				"يُعطّل عملاً مشروعاً.**", c.Role, c.Cap, res.Code)
		}
	}
}
