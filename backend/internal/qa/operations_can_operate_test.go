package qa

// ══════════════════════════════════════════════════════════════════════
//  **موظّفُ العمليّات يُدير المنصّة**  `BOOK-03`
// ══════════════════════════════════════════════════════════════════════
//
// **قِيس ٢٠٢٦-٠٩-٣٠ في تجربة المنصّة**:
//
//	statuses.go:76   opsRoles = []string{"ops"}
//	roleclass.go:91  "ops" → ClassLegacy ⇒ GrantNever
//	roleclass.go:70  "operations" → ClassStaff   ← الدورُ الحيّ
//	و"operations" لا يظهر في internal/orders إطلاقاً
//
// **فموظّفُ عمليّاتٍ يُنشأ من اللوحة لا يستطيع نقلَ حالةِ طلبٍ واحدة**:
// **البابُ يفتح له بالقدرة** (`orders.intervene`) **والآلةُ ترفضه بالاسم.**
//
// **ووضعُ المنصّة كلُّه يقوم على أنّ العمليّاتَ تقبل وتوزّع** — فالموظّفُ
// الذي يُدير المنصّة كان لا يستطيع أن يُديرها. **ولم يظهر العطبُ في
// التجربة** لأنّ `canTransition` تستثني `admin` صراحةً: **المالكُ يعمل
// والموظّفُ لا.**
//
// # ولماذا لا يُرقَّع باسمٍ ثانٍ
//
// **عقدُ `ADG-2`**: «كلُّ بابٍ بقدرته لا باسم دور». فإضافةُ `"operations"`
// إلى الخريطة تُثبّت اسماً ثانياً **وتُعيد العطبَ لثالثٍ غداً.**
// **فالفاعلُ `ops` صار يُشتقّ من `orders.intervene`** — أيَّ دورٍ حمله.
//
// # وهذا يقيس البابَ لا المنطق
//
// **حرّاسُ المنطق تقيس الخريطة، وهذا يقيس ما يقع لموظّفٍ حقيقيّ**: حسابٌ
// بدور `operations`، وطلبٌ حقيقيّ، **ونداءٌ عبر الشبكة.**

import (
	"net/http"
	"testing"
)

// ── أ · موظّفُ `operations` ينقل حالةَ طلب ────────────────────────────
func TestBOOK03_OperationsStaffCanTransitionOrder(t *testing.T) {
	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	treasury(t, h)

	// **ودورُ العمليّات الحيُّ `operations` لا `ops`** — وهو ما تُنشئه اللوحة.
	// **و`h.NewUser` لا `f.NewUserWith`**: الثاني يُصدر توكناً **بلا معرّف
	// جلسة** (`IssueAccess(id, roles, "")`)، **والوسيطُ ينصّ أنّ ما لا جلسةَ
	// له لا قدراتِ له فالافتراضُ منع** — فكلُّ بابٍ يردّ ٤٠٣ ولو مَلَك الدورُ
	// القدرة. (وقعتُ فيه ٢٠٢٦-٠٩-٣٠ فكِدتُ أُسمّي عطبَ مِسنَدٍ عطبَ منتج.)
	staff := h.NewUser("operations")
	item := h.NewItem(1000)
	cust := h.NewUser("customer")

	// **تشخيصٌ أوّلاً**: أتعمل قدراتُ هذا الحساب على بابٍ آخرَ يطلب قدرةً يملكها؟
	probe := h.GET("/api/v1/admin/orders?per_page=1", staff.Token)
	t.Logf("تشخيص: GET /admin/orders بدور operations ⇒ %d", probe.Code)

	created := h.POSTKey("/api/v1/orders", cust.Token, uniq("book03"), orderBody(item, 1))
	if created.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ الطلب: %s", created)
	}
	oid, _ := created.JSON()["id"].(string)
	if oid == "" {
		t.Fatalf("لا معرّفَ للطلب — %s", created)
	}

	// **والقبولُ فعلُ العمليّات في وضع المنصّة** — وهو أوّلُ ما يفعله.
	res := h.POST("/api/v1/admin/orders/"+oid+"/transition", staff.Token,
		map[string]any{"to": "accepted", "note": "BOOK-03 شاهد"})
	if res.Code >= 400 {
		t.Fatalf("**موظّفُ `operations` لا يستطيع قبولَ طلب** — رُدّ %d · %s\n"+
			"**ووضعُ المنصّة يقوم على أنّ العمليّاتَ تقبل وتوزّع** (BOOK-03)",
			res.Code, res.String())
	}

	// **والحالُ في القاعدة تبدّلت** — فالردُّ ليس تجميلاً.
	var status string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT status FROM orders WHERE id = $1::uuid`, oid).Scan(&status); err != nil {
		t.Fatalf("قراءةُ الحال: %v", err)
	}
	if status != "accepted" {
		t.Fatalf("رُدّ نجاحاً والحالُ %q — **نجاحٌ لا يكتب**", status)
	}
}

// ── ب · ومن لا يملك القدرةَ لا يصير فاعلَ عمليّات ─────────────────────
//
// **وحارسٌ يفتح لكلّ موظّفٍ حارسٌ يُنزَع**: خدمةُ العملاء لا تملك
// `orders.intervene`، **فلا تنقل حالةَ طلب.**
func TestBOOK03_SupportStaffStillCannotTransition(t *testing.T) {
	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	treasury(t, h)

	support := h.NewUser("customer_support")
	item := h.NewItem(1000)
	cust := h.NewUser("customer")

	created := h.POSTKey("/api/v1/orders", cust.Token, uniq("book03b"), orderBody(item, 1))
	if created.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ الطلب: %s", created)
	}
	oid, _ := created.JSON()["id"].(string)

	res := h.POST("/api/v1/admin/orders/"+oid+"/transition", support.Token,
		map[string]any{"to": "accepted", "note": "BOOK-03 سالب"})
	if res.Code < 400 {
		t.Fatalf("**خدمةُ العملاء نقلت حالةَ طلب** — رُدّ %d · %s\n"+
			"ولا تملك `orders.intervene` (BOOK-03)", res.Code, res.String())
	}

	var status string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT status FROM orders WHERE id = $1::uuid`, oid).Scan(&status); err != nil {
		t.Fatalf("قراءةُ الحال: %v", err)
	}
	if status == "accepted" {
		t.Fatal("**رُدّ منعاً والحالُ تبدّلت** — منعٌ لا يمنع")
	}
}
