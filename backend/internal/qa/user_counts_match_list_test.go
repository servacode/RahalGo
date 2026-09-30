package qa

// ══════════════════════════════════════════════════════════════════════
//  **البطاقةُ تعدّ ما تعرضه القائمةُ تحتها**  `BOOK-06`
// ══════════════════════════════════════════════════════════════════════
//
// **رآه المالكُ بعينه على التجهيز ٢٠٢٦-٠٩-٣٠**، بعد أن صار في المنصّة خمسةُ
// موظّفين:
//
//	بطاقةُ «موظّفو المنصّة»  تقول  ١
//	وضغطُها يُخرج            ٥
//	وبطاقةُ «كلّ الحسابات»   تقول  ٥   والجدولُ تحتها ٦ صفوف
//
// **ورقمان يتناقضان في شاشةٍ واحدةٍ أسوأُ من رقمٍ غائب**: من قرأ البطاقة ولم
// يضغطها استنتج أن لا موظّفَ في منصّته.
//
// # ولماذا وقع مرّتين قبل هذه
//
// العطبُ نفسُه أُصلح في `identity.ListUsers` (`BOOK-01` و`BOOK-05`) **ولم
// يُصلَح في `handleAdminUserRoleCounts`** — **مُسنَدان لنفس السؤال في موضعين.**
//
// # فهذا يقيس الاتّفاق لا الرقم
//
// **ولا يثبّت عدداً** — قاعدةُ الاختبار مشتركة، والعددُ المطلقُ يشيخ بأوّل
// صفٍّ يضيفه غيرُه. **بل يقيس أن يقول البابان الشيءَ نفسَه**: وهو بعينه ما
// انكسر، **ويبقى صحيحاً مهما نمت القاعدة.**

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// stats يقرأ بطاقاتِ الشاشة — `GET /admin/users/stats`.
func stats(t *testing.T, h *Harness, tok string) (total int, roles map[string]int) {
	t.Helper()
	res := h.GET("/api/v1/admin/users/stats", tok)
	if res.Code != http.StatusOK {
		t.Fatalf("بطاقاتُ الحسابات: %s", res)
	}
	var body struct {
		Data struct {
			Total int            `json:"total"`
			Roles map[string]int `json:"roles"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &body); err != nil {
		t.Fatalf("قراءةُ البطاقات: %v — %s", err, res.Body)
	}
	return body.Data.Total, body.Data.Roles
}

// listTotal يقرأ ما تقوله القائمةُ نفسُها لمُرشِّحٍ بعينه.
func listTotal(t *testing.T, h *Harness, tok, query string) (int, int) {
	t.Helper()
	res := h.GET("/api/v1/admin/users?"+query+"&per_page=100", tok)
	if res.Code != http.StatusOK {
		t.Fatalf("قائمةُ الحسابات (%s): %s", query, res)
	}
	var body struct {
		Data struct {
			Total int `json:"total"`
			Users []struct {
				Phone string   `json:"phone"`
				Roles []string `json:"roles"`
			} `json:"users"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &body); err != nil {
		t.Fatalf("قراءةُ القائمة: %v — %s", err, res.Body)
	}
	return body.Data.Total, len(body.Data.Users)
}

// ── أ · «موظّفو المنصّة»: البطاقةُ = القائمة ──────────────────────────
//
// **وهذا هو السطرُ الذي رآه المالك**: بطاقةٌ تقول ١ وقائمةٌ تُخرج ٥.
func TestBOOK06_StaffCardMatchesStaffList(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")

	// **ويُنشأ من كلّ صنفٍ واحدٌ** — فالمجموعةُ غيرُ فارغةٍ بيقين، **ولا
	// يخضرّ الادّعاءُ على لا شيء.**
	for _, role := range []string{"operations", "finance", "customer_support", "platform_monitor"} {
		h.NewUser(role)
	}
	h.NewUser("customer")
	h.NewUser("driver")

	_, roles := stats(t, h, admin.Token)
	card := roles["staff"]
	total, rows := listTotal(t, h, admin.Token, "role=staff")

	if card == 0 || total == 0 {
		t.Fatalf("مجموعةٌ فارغةٌ فالادّعاءُ ميّت: بطاقة=%d قائمة=%d", card, total)
	}
	if card != total {
		t.Fatalf("البطاقةُ تقول %d والقائمةُ تقول %d — رقمان في شاشةٍ واحدة", card, total)
	}
	if total != rows {
		t.Fatalf("قائمةُ الطاقم: مجموعٌ %d وصفوفٌ %d", total, rows)
	}
}

// ── ب · والطاقمُ يشمل مديرَ المنصّة والمراقبَ معاً ────────────────────
//
// **وهو نصُّ قرار المالك ٢٠٢٦-٠٩-٣٠**: «ويجب أن يكون موظّف العمليّات وموظّف
// المالية وخدمة العملاء **ومدير المنصّة** ضمن موظّفي المنصّة، **ومراقب
// المنصّة** أيضاً».
//
// **والبطاقةُ كانت تُسقط الأربعةَ من الخمسة.**
func TestBOOK06_StaffCardCountsEveryStaffRole(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")

	_, before := stats(t, h, admin.Token)
	base := before["staff"]

	// **خمسةُ أدوارٍ من أصنافٍ ثلاثة** — `ClassStaff` و`ClassElevated`.
	// **وواحدٌ من كلّ صنفٍ يكفي**: العطبُ كان يُسقط الصنفَ لا الفرد.
	added := []string{"operations", "finance", "customer_support", "platform_monitor", "admin"}
	for _, role := range added {
		h.NewUser(role)
	}

	_, after := stats(t, h, admin.Token)
	got := after["staff"] - base
	if got != len(added) {
		t.Fatalf("أُنشئ %d موظّفاً والبطاقةُ زادت %d — دورٌ لا تعدّه", len(added), got)
	}
}

// ── ج · «كلّ الحسابات»: البطاقةُ = مجموعُ القائمة ─────────────────────
//
// **ولا يُخفى الأدمنُ ولا الحسابُ النظاميّ** — قرارُ المالك ٢٠٢٦-٠٩-٣٠،
// ودرسُ `BOOK-01`: **ما لا يُعَدّ ولا يُعرَض شبح.**
func TestBOOK06_TotalCardMatchesUnfilteredList(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	h.NewUser("customer")
	h.NewUser("operations")

	card, _ := stats(t, h, admin.Token)
	total, _ := listTotal(t, h, admin.Token, "")

	if card == 0 || total == 0 {
		t.Fatalf("مجموعةٌ فارغةٌ فالادّعاءُ ميّت: بطاقة=%d قائمة=%d", card, total)
	}
	if card != total {
		t.Fatalf("بطاقةُ «كلّ الحسابات» تقول %d والقائمةُ تقول %d", card, total)
	}
}

// ── د · وكلُّ بطاقةِ دورٍ تطابق مُرشِّحَها ─────────────────────────────
//
// **والأدمنُ منها** — فعدُّ الأدوار كان يحجبه صراحةً (`<> 'admin'`).
func TestBOOK06_EveryRoleCardMatchesItsFilter(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	for _, role := range []string{"customer", "driver", "merchant", "sales", "operations", "admin"} {
		h.NewUser(role)
	}

	_, roles := stats(t, h, admin.Token)
	for _, role := range []string{"customer", "driver", "sales", "operations", "admin"} {
		card := roles[role]
		total, _ := listTotal(t, h, admin.Token, "role="+role)
		if card == 0 || total == 0 {
			t.Errorf("%s: مجموعةٌ فارغةٌ فالادّعاءُ ميّت (بطاقة=%d قائمة=%d)", role, card, total)
			continue
		}
		if card != total {
			t.Errorf("%s: البطاقةُ %d والقائمةُ %d", role, card, total)
		}
	}
}

// ── هـ · والحالةُ السالبة: `ops` المُحالُ لا يُنشِئ طاقماً ─────────────
//
// **وهذا يُثبت أنّ «د» ليس تطابقاً عارضاً**: لو بقي المُسنَدُ
// `IN ('ops','finance')` **لكان الطاقمُ يساوي عددَ الماليّة وحدَه** —
// فيُقاس صراحةً أنّ الطاقمَ أكبرُ منها.
func TestBOOK06_StaffIsNotJustFinance(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	h.NewUser("finance")
	h.NewUser("operations")
	h.NewUser("customer_support")

	_, roles := stats(t, h, admin.Token)
	staff, finance := roles["staff"], roles["finance"]
	if staff <= finance {
		t.Fatalf("الطاقمُ %d والماليّةُ %d — فالمُسنَدُ يعدّ الماليّةَ وحدَها", staff, finance)
	}
	// **ولا يُعَدُّ الزبونُ طاقماً** — فالنفيُ لا يبتلع الجميع.
	h.NewUser("customer")
	_, after := stats(t, h, admin.Token)
	if after["staff"] != staff {
		t.Fatalf("زبونٌ واحدٌ زاد الطاقمَ من %d إلى %d", staff, after["staff"])
	}
}

var _ = fmt.Sprintf
