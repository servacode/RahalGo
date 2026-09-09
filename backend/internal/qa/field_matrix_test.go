package qa

// ══════════════════════════════════════════════════════════════════════
// **أقلُّ صلاحيّةٍ في الحقول لا في الأبواب** — `XG-42`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان يقع
//
// **قسمةُ القدرات تحرس البابَ لا الحمولة.** **فمن ملك `users.read`
// لأنّه يجد حساباً لقيدٍ ماليّ نال هاتفَه أيضاً** — **ولا يتّصل بأحد.**
//
// **وإخفاءُ الحقل في الشاشة ليس حمايةً**: **الردُّ الخام هو الحقيقة.**
//
// # والعقدُ المقاس
//
//	رقمُ الاتّصال      ⇒  `users.contact.read`
//	وما عداه من هويّة  ⇒  `users.read` كما كان
//
// **ولا اسمَ دورٍ في الشرط** — **قدرةٌ تُحسب من الحقيقة الموثوقة**،
// **فدورٌ جديدٌ يصنعه الإداريُّ يأخذ حكمَه من قدراته لا من اسمه.**
//
// # وهذا الفحصُ يقرأ الردَّ الخام
//
// **ويمشي في العمق**: `order.customer_phone` كما `user.phone` —
// **وحقلٌ يُحذف من السطح ويبقى في عشٍّ داخليٍّ ليس محذوفاً.**

import (
	"encoding/json"
	"strings"
	"testing"
)

// contactKeys **حقولُ الاتّصال المحميّة** — تُقرأ في الفحص وفي المنتَج
// من موضعٍ واحدٍ معنىً: **من بدّل المعجمَ هناك رأى هذا يسقط.**
var contactKeys = []string{"phone", "customer_phone", "driver_phone"}

// realValuesIn يمشي في الجسم كلِّه ويعدّ القيمَ الحقيقيّةَ للمفاتيح.
//
// **ويُقرأ المعنى لا النصّ**: **فكٌّ للترميز ثمّ مشيٌ في الأعشاش** —
// **و`grep` على نصّ الردّ يخطئ التعشيش ويصطاد أسماءَ حقولٍ لا قيمَها.**
func realValuesIn(t *testing.T, body []byte, keys []string) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return out
	}
	want := map[string]bool{}
	for _, k := range keys {
		want[k] = true
	}
	var walk func(any)
	walk = func(n any) {
		switch v := n.(type) {
		case map[string]any:
			for k, val := range v {
				if want[k] {
					if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
						out[k] = append(out[k], s)
					}
				}
				walk(val)
			}
		case []any:
			for _, e := range v {
				walk(e)
			}
		}
	}
	walk(root)
	return out
}

// matrixReader قارئٌ بدورٍ وقدراتِه المتوقَّعة.
type matrixReader struct {
	Role    string
	Contact bool // **أيملك `users.contact.read`؟**
}

func TestXG42_ContactFieldsFollowCapabilityNotRoute(t *testing.T) {
	h := New(t)
	// **حسابٌ يُقرأ**: زبونٌ له هاتفٌ وطلبٌ قائم — **فالردودُ ليست فارغة.**
	f := h.Factory()
	cust := f.NewUserWith("customer")
	treasury(t, h)
	// **وطلبٌ قائمٌ فتحمل مساراتُ الطلبات هاتفَ زبونه** — **وإلّا
	// قِيس الفراغُ وظُنّ حمايةً.**
	item := h.NewItem(1000)
	if made := h.POSTKey("/api/v1/orders", cust.Token, uniq("xg42"),
		orderBody(item, 1)); made.Code >= 400 {
		t.Fatalf("طلبٌ للقياس: %s", made)
	}

	readers := []matrixReader{
		{Role: "owner_super_admin", Contact: true},
		{Role: "ops", Contact: true},
		{Role: "operations", Contact: true},
		{Role: "customer_support", Contact: true},
		{Role: "trust_safety", Contact: true},
		{Role: "driver_verification", Contact: true},
		// **والماليّةُ تجد الحسابَ ولا تتّصل به** — قيدٌ لا مكالمة.
		{Role: "finance", Contact: false},
		{Role: "analytics", Contact: false},
	}

	routes := []string{
		"/api/v1/admin/users",
		"/api/v1/admin/users/" + cust.ID,
		"/api/v1/admin/customers",
		"/api/v1/admin/salesreps",
		"/api/v1/admin/drivers",
		"/api/v1/admin/orders",
	}

	for _, rd := range readers {
		u := h.NewUser(rd.Role)
		for _, route := range routes {
			got := h.GET(route, u.Token)
			// **ومن مُنع عند الباب لا يُقاس حقلُه** — **`ADG-2` بابُه
			// وهذا حِمله**، ولا يُخلَط الحكمان.
			if got.Code == 403 || got.Code == 404 {
				continue
			}
			if got.Code != 200 {
				t.Errorf("%s → %s: %d — **ردٌّ غيرُ متوقَّع**", rd.Role, route, got.Code)
				continue
			}
			found := realValuesIn(t, got.Body, contactKeys)
			total := 0
			for _, v := range found {
				total += len(v)
			}
			switch {
			case rd.Contact && total == 0:
				// **وما يجب أن يحمل رقماً يُدان إن لم يحمله** —
				// **وحمايةٌ تُفقِر المسموحَ عطبٌ لا إصلاح.**
				if mustCarryContact[route] {
					t.Errorf("**%s مسموحٌ ولم ينل رقماً** — %s — "+
						"**وحجبٌ عمّن يحتاجه يكسر عملاً مشروعاً.** (`XG-42`)",
						rd.Role, route)
				} else {
					t.Logf("  %s → %s: مسموحٌ ولا رقمَ في الردّ (جدولٌ فارغ)", rd.Role, route)
				}
			case rd.Contact:
				t.Logf("  %s → %s: مسموحٌ · أرقامٌ=%d ✓", rd.Role, route, total)
			case total > 0:
				t.Errorf("**%s نال رقمَ اتّصالٍ ولا يملك `users.contact.read`** — "+
					"%s · %d قيمةً (%v) — **والبابُ ليس الحمولة.** (`XG-42`)",
					rd.Role, route, total, contactKeysOf(found))
			default:
				t.Logf("  %s → %s: ممنوعٌ ولا رقم ✓", rd.Role, route)
			}
		}
	}
}

// mustCarryContact **مساراتٌ يُشترط أن يبلغ الرقمُ من يستحقّه فيها** —
// **فالفحصُ يقيس المنعَ والمنحَ معاً.**
var mustCarryContact = map[string]bool{
	"/api/v1/admin/users":     true,
	"/api/v1/admin/customers": true,
	"/api/v1/admin/orders":    true,
}

func contactKeysOf(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// ══════════════════════════════════════════════════════════════════════
// **اتّحادُ الأدوار · ودورٌ يصنعه الإداريّ · ونزعٌ يسري في الحال**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذه الثلاثةُ هي ما يجعل الحكمَ بالقدرة لا بالاسم قابلاً للحياة**:
// **من جمع دورين نال اتّحادَهما**، **ودورٌ جديدٌ يأخذ حكمَه من قدراته**،
// **ونزعُ القدرة يسري بلا خروجٍ ولا دخول** (`R15`).
func TestXG42_CapabilityUnionCustomRoleAndRevoke(t *testing.T) {
	h := New(t)
	f := h.Factory()
	_ = f.NewUserWith("customer")

	phones := func(tok string) int {
		t.Helper()
		got := h.GET("/api/v1/admin/users", tok)
		if got.Code != 200 {
			t.Fatalf("قراءةُ الدليل: %d — %s", got.Code, got.Body)
		}
		n := 0
		for _, v := range realValuesIn(t, got.Body, contactKeys) {
			n += len(v)
		}
		return n
	}

	// ── ١ ── **اتّحادُ دورين** ────────────────────────────────────
	//
	// **ماليّةٌ لا ترى الرقم، ودعمٌ يراه** — **ومن جمعهما رآه.**
	fin := h.NewUser("finance")
	if n := phones(fin.Token); n != 0 {
		t.Errorf("**الماليّةُ وحدَها نالت %d رقماً**", n)
	}
	if _, err := h.Pool.Exec(ctxBG(),
		`INSERT INTO user_roles (user_id, role_code) VALUES ($1::uuid, 'customer_support')
		 ON CONFLICT DO NOTHING`, fin.ID); err != nil {
		t.Fatalf("ضمُّ دورٍ ثانٍ: %v", err)
	}
	union := phones(fin.Token)
	t.Logf("UNION: ماليّةٌ+دعمٌ ⇒ أرقامٌ=%d", union)
	if union == 0 {
		t.Error("**جمع دورين أحدهما يملك القدرةَ ولم ينل الرقم** — " +
			"**والاتّحادُ عقدُ القدرات.**")
	}

	// ── ٢ ── **دورٌ يصنعه الإداريُّ بقدراته** ─────────────────────
	//
	// **ولا اسمَ له في الشيفرة** — **فلو كان الحكمُ بالاسم لسقط هنا.**
	const custom = "qa_custom_reader"
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO roles (code, name_key) VALUES ($1, 'roles.qa_custom_reader')
		ON CONFLICT DO NOTHING`, custom); err != nil {
		t.Skipf("تعذّر إنشاءُ دورٍ مخصَّص: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM role_capabilities WHERE role_code = $1`, custom)
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM user_roles WHERE role_code = $1`, custom)
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM roles WHERE code = $1`, custom)
	})
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO role_capabilities (role_code, capability_code)
		VALUES ($1, 'users.read') ON CONFLICT DO NOTHING`, custom); err != nil {
		t.Fatalf("منحُ قدرة: %v", err)
	}
	reader := h.NewUser(custom)
	if n := phones(reader.Token); n != 0 {
		t.Errorf("**دورٌ مخصَّصٌ بلا `users.contact.read` نال %d رقماً**", n)
	}

	// ── ٣ ── **ومنحٌ يسري في الحال** ──────────────────────────────
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO role_capabilities (role_code, capability_code)
		VALUES ($1, 'users.contact.read') ON CONFLICT DO NOTHING`, custom); err != nil {
		t.Fatalf("منحُ الاتّصال: %v", err)
	}
	granted := phones(reader.Token)
	t.Logf("CUSTOM ROLE: بعد المنح ⇒ أرقامٌ=%d (الجلسةُ نفسُها)", granted)
	if granted == 0 {
		t.Error("**مُنحت القدرةُ ولم يصل الرقم** — **والحكمُ بالقدرة لا بالاسم**")
	}

	// ── ٤ ── **ونزعٌ يسري في الحال أيضاً** — `R15` ────────────────
	//
	// **بلا خروجٍ ولا رمزٍ جديد**: الحقيقةُ الموثوقةُ تُقرأ في كلّ طلب.
	if _, err := h.Pool.Exec(ctxBG(), `
		DELETE FROM role_capabilities
		 WHERE role_code = $1 AND capability_code = 'users.contact.read'`, custom); err != nil {
		t.Fatalf("نزعُ القدرة: %v", err)
	}
	after := phones(reader.Token)
	t.Logf("REVOKE: بعد النزع ⇒ أرقامٌ=%d (الجلسةُ نفسُها)", after)
	if after != 0 {
		t.Errorf("**نُزعت القدرةُ وما زال الرقمُ يصل** (%d) — "+
			"**وسلطةُ الحقول في القاعدة لا في رمزٍ محفوظ.** (`R15`)", after)
	}
}
