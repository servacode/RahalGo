package qa

// **الإنذارُ يصل صاحبَه** — `CAF-19` · `CUST-SUP-012`.
//
// **لوحةُ الإدارة تَعِد أنّ الإنذارَ «يصل صاحبَ الحساب بنصّه ويبقى في سجلّه»**
// — **وكان يُخزَّن ولا يُبلَّغ.** فالمحرّكُ الآن يُنشئ إشعارَ حساب (`account`)
// بنصٍّ عربيٍّ عند كلّ إنذار، **فيظهر في صندوق الزبون** (الجرس يطلب كلَّ
// الأنواع)، **و`‎/my/warnings` يعرض سجلَّه معزولاً عن غيره.**

import (
	"strings"
	"testing"
)

func TestCAF19_WarningReachesCustomerAndIsIsolated(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	victim := h.Customer()
	bystander := h.Customer()

	got := h.POST("/api/v1/admin/users/"+victim.ID+"/warnings", admin.Token, map[string]any{
		"reason": "repeated_cancel",
		"note":   "إلغاءٌ متكرّر للطلبات",
	})
	if got.Code >= 400 {
		t.Fatalf("CAF-19 تعذّر إصدارُ الإنذار: %s", got)
	}

	// ── ١ · إشعارُ حسابٍ للزبون، بنصٍّ عربيٍّ لا رمزٍ خام ──────────────
	var n int
	var body string
	if err := h.Pool.QueryRow(t.Context(),
		`SELECT count(*), COALESCE(max(body), '') FROM notifications
		  WHERE user_id = $1::uuid AND kind = 'account'`, victim.ID).Scan(&n, &body); err != nil {
		t.Fatalf("CAF-19 تعذّرت قراءةُ الإشعارات: %v", err)
	}
	if n < 1 {
		t.Fatalf("CAF-19 **لم يصل إشعارٌ للزبون** — عدد=%d (الإنذارُ صامت)", n)
	}
	if body == "" || strings.Contains(body, "repeated_cancel") {
		t.Errorf("CAF-19 نصُّ الإشعار خامٌّ أو فارغ: %q", body)
	}
	if !strings.Contains(body, "إلغاء") {
		t.Errorf("CAF-19 نصُّ الإشعار لا يحمل السببَ/الملاحظة: %q", body)
	}

	// ── ٢ · ‎/my/warnings يعرض إنذارَه ──────────────────────────────
	mine := h.GET("/api/v1/my/warnings", victim.Token)
	if mine.Code >= 400 {
		t.Fatalf("CAF-19 ‎/my/warnings رُدّ: %s", mine)
	}
	ws, _ := mine.JSON()["warnings"].([]any)
	if len(ws) != 1 {
		t.Fatalf("CAF-19 سجلُّ إنذاراتِ الزبون عددُه %d — يُنتظر 1", len(ws))
	}

	// ── ٣ · العزل: زبونٌ آخر لا يرى إنذارَ غيره، ولا إشعارَ له ────────
	other := h.GET("/api/v1/my/warnings", bystander.Token)
	if ows, _ := other.JSON()["warnings"].([]any); len(ows) != 0 {
		t.Errorf("CAF-19 **تسريب**: زبونٌ آخر يرى %d إنذاراً", len(ows))
	}
	var bn int
	if err := h.Pool.QueryRow(t.Context(),
		`SELECT count(*) FROM notifications WHERE user_id = $1::uuid AND kind = 'account'`,
		bystander.ID).Scan(&bn); err != nil {
		t.Fatalf("CAF-19: %v", err)
	}
	if bn != 0 {
		t.Errorf("CAF-19 **تسريبُ إشعار**: زبونٌ آخر لديه %d إشعارَ حساب", bn)
	}
}
