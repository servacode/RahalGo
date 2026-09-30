package qa

// ══════════════════════════════════════════════════════════════════════
//  **حسابُ المالك لا يمسّه أحد — على البابِ الحقيقيّ**  `BOOK-02`
// ══════════════════════════════════════════════════════════════════════
//
// **وحرّاسُ الحزمة تقيس المنطق، وهذا يقيس البابَ**: قدرةٌ مُمنوحة، وخطوةُ
// تحقّقٍ مُستوفاة، **ثمّ يُردّ لأنّ الهدفَ حسابُ المالك** — لا لأنّ شيئاً
// ناقصاً في الطريق.
//
// **وقِيس قبل الحارس** (٢٠٢٦-٠٩-٣٠): `users.status.manage` يملكها
// `trust_safety`، **والحارسُ الوحيدُ كان «لا تفعلها بنفسك»** — فالنداءُ
// كان ينجح.
//
// **وقرارُ المالك**: «صاحبُ المنصّة لا أحدَ يستطيع تعديلَ أيِّ إجراءٍ يخصّه».

import (
	"net/http"
	"testing"
)

// suspendBody جسمُ الحظر — **والسببُ إلزاميٌّ للإيقاف.**
func suspendBody() map[string]any {
	return map[string]any{"status": "suspended", "status_reason": "BOOK-02 شاهد"}
}

func userPath(id string) string { return "/api/v1/admin/users/" + id }

// attempt **نداءٌ مباشرٌ، وإن طُلب الإثباتُ يُطلَب ويُعاد** — فالاختبارُ
// صامدٌ لتصنيفِ الحساسيّة: **المقيسُ الحارسُ لا طريقُ التأكيد.**
func attempt(t *testing.T, h *Harness, tok, method, path string, body any) Res {
	t.Helper()
	res := h.Call(method, path, tok, body, nil)
	if res.Code == http.StatusForbidden && res.Err() == "step_up_required" {
		grant, code := askStepUp(t, h, tok, method, path, body, stepPass)
		if code != http.StatusOK || grant == "" {
			t.Fatalf("**طُلب الإثباتُ وتعذّر الحصولُ عليه** — رمز=%d: "+
				"فالاختبارُ لا يقيس الحارسَ بل الطريق", code)
		}
		res = withGrant(h, tok, method, path, body, grant)
	}
	return res
}

// ── أ · موظّفُ طاقمٍ بقدرةٍ وإثباتٍ يُردّ عن حساب المالك ──────────────
func TestBOOK02_StaffWithGrantCannotSuspendOwner(t *testing.T) {
	h := New(t)
	f := h.Factory()

	staff := stepUser(t, h, "trust_safety")
	owner := f.NewUserWith("owner_super_admin")

	res := attempt(t, h, staff.Token, "PATCH", userPath(owner.ID), suspendBody())
	if res.Code < 400 {
		t.Fatalf("**موظّفُ `trust_safety` حظر حسابَ المالك** — رُدّ %d · %s\n"+
			"وقرارُ المالك أنّ حسابَه لا يمسّه أحد (BOOK-02)", res.Code, res.String())
	}
	if res.Code != http.StatusForbidden {
		t.Errorf("رُدّ %d والمنتظَر ٤٠٣ — %s", res.Code, res.String())
	}

	// **والحالُ في القاعدة لم تتبدّل** — فالردُّ ليس تجميلاً.
	var status string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT status FROM users WHERE id = $1::uuid`, owner.ID).Scan(&status); err != nil {
		t.Fatalf("قراءةُ الحال: %v", err)
	}
	if status != "active" {
		t.Fatalf("**حالُ حسابِ المالك صارت %q** — فالردُّ لم يمنع الكتابة", status)
	}
}

// ── ب · ولا تُعاد كلمةُ مرورِ المالك من لوحة ──────────────────────────
func TestBOOK02_StaffCannotResetOwnerPassword(t *testing.T) {
	h := New(t)
	f := h.Factory()

	staff := stepUser(t, h, "trust_safety")
	owner := f.NewUserWith("owner_super_admin")
	path := userPath(owner.ID) + "/password"
	body := map[string]any{"password": "Book02-Reset-2026!"}

	res := attempt(t, h, staff.Token, "POST", path, body)
	if res.Code < 400 {
		t.Fatalf("**موظّفٌ أعاد تعيينَ كلمةِ مرورِ المالك** — رُدّ %d · %s\n"+
			"**وهذا استيلاءٌ على المنصّة** (BOOK-02)", res.Code, res.String())
	}
}

// ── ج · والحسابُ العاديُّ يُدار كما كان ───────────────────────────────
//
// **وحارسٌ يمنع كلَّ شيءٍ حارسٌ يُنزَع** — فمكتبُ الثقة والسلامة يعمل.
func TestBOOK02_StaffStillManagesOrdinaryAccounts(t *testing.T) {
	h := New(t)

	staff := stepUser(t, h, "trust_safety")
	victim := h.Customer()

	res := attempt(t, h, staff.Token, "PATCH", userPath(victim.ID), suspendBody())
	if res.Code >= 400 {
		t.Fatalf("**مُنع موظّفُ الطاقم من حظرِ زبون** — رُدّ %d · %s\n"+
			"والحارسُ للمحميِّ والمرتفعِ وحدَهما", res.Code, res.String())
	}
	var status string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT status FROM users WHERE id = $1::uuid`, victim.ID).Scan(&status); err != nil {
		t.Fatalf("قراءةُ الحال: %v", err)
	}
	if status != "suspended" {
		t.Fatalf("رُدّ نجاحاً والحالُ %q — **نجاحٌ لا يكتب** ", status)
	}
}
