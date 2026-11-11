package qa

// ══════════════════════════════════════════════════════════════════════
// **«أضف فرع»** (قرارُ المالك ٢٠٢٦-١١-١٠) — الطلبُ من صاحب المتجر ومن مندوبه، وموافقةُ المكتب
// بعمولةٍ تُنشئ متجراً بصاحبه نفسِه، ورفضٌ بسبب. وكلُّ ذلك بإشعارٍ لصاحب الطلب.
// ══════════════════════════════════════════════════════════════════════

import (
	"testing"
)

func branchBody(name string) map[string]any {
	return map[string]any{"name": name, "lat": 35.9528, "lng": 39.0079, "phone": "0944000111"}
}

// branchNotifCount **عددُ إشعاراتِ مستخدمٍ بعنوانٍ بعينه.**
func branchNotifCount(t *testing.T, h *Harness, userID, title string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM notifications WHERE user_id = $1::uuid AND title = $2`, userID, title).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestBranchRequest_MerchantCreateAndForbidden(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	other := f.Merchant()
	name := uniq("فرع")

	res := h.POST("/api/v1/merchant/stores/"+m.ID+"/branch-requests", m.Owner.Token, branchBody(name))
	if res.Code != 201 {
		t.Fatalf("صاحبُ المتجر لا يطلب فرعاً: %s", res)
	}
	if st, _ := res.JSON()["status"].(string); st != "pending" {
		t.Fatalf("الحالةُ الأولى بانتظار الموافقة: %s", res)
	}

	// **القائمةُ تعرضه بحالته.**
	list := h.GET("/api/v1/merchant/branch-requests", m.Owner.Token)
	if list.Code != 200 || len(list.JSON()["requests"].([]any)) != 1 {
		t.Fatalf("قائمةُ طلباته: %s", list)
	}

	// **ونقرتان بالاسم نفسِه لا تصنعان طلبين.**
	if dup := h.POST("/api/v1/merchant/stores/"+m.ID+"/branch-requests", m.Owner.Token, branchBody(name)); dup.Code != 409 {
		t.Fatalf("طلبٌ مكرّر: %s", dup)
	}
	// **بلا موقعٍ أو اسم ⇒ رفض.**
	if r := h.POST("/api/v1/merchant/stores/"+m.ID+"/branch-requests", m.Owner.Token,
		map[string]any{"name": uniq("ب")}); r.Code != 400 {
		t.Fatalf("بلا موقع: %s", r)
	}
	if r := h.POST("/api/v1/merchant/stores/"+m.ID+"/branch-requests", m.Owner.Token,
		map[string]any{"lat": 35.9, "lng": 39.0}); r.Code != 400 {
		t.Fatalf("بلا اسم: %s", r)
	}
	// **ومتجرٌ ليس له ⇒ ممنوع.**
	if r := h.POST("/api/v1/merchant/stores/"+other.ID+"/branch-requests", m.Owner.Token, branchBody(uniq("غ"))); r.Code != 403 {
		t.Fatalf("متجرُ غيره: %s", r)
	}
	// **وزبونٌ لا يطلب.**
	c := h.Customer()
	if r := h.POST("/api/v1/merchant/stores/"+m.ID+"/branch-requests", c.Token, branchBody(uniq("ز"))); r.Code != 403 {
		t.Fatalf("زبون: %s", r)
	}
	// **ولا يقرأ طلباتِ غيره.**
	if l := h.GET("/api/v1/merchant/branch-requests", other.Owner.Token); l.Code != 200 ||
		len(l.JSON()["requests"].([]any)) != 0 {
		t.Fatalf("رأى طلباتِ غيره: %s", l)
	}
}

func TestBranchRequest_RepOnlyForOwnClient(t *testing.T) {
	h := New(t)
	f := h.Factory()
	rep, rep2 := f.RepAccount(), f.RepAccount()
	m := f.Merchant(OwnedByRep(rep.ID))

	res := h.POST("/api/v1/rep/merchants/"+m.ID+"/branch-requests", rep.Token, branchBody(uniq("فرعُ مندوب")))
	if res.Code != 201 {
		t.Fatalf("المندوبُ لا يطلب لعميله: %s", res)
	}
	if role, _ := res.JSON()["requested_by_role"].(string); role != "sales" {
		t.Fatalf("يُسجَّل أنّ المندوبَ هو الطالب: %s", res)
	}
	if l := h.GET("/api/v1/rep/merchants/"+m.ID+"/branch-requests", rep.Token); l.Code != 200 ||
		len(l.JSON()["requests"].([]any)) != 1 {
		t.Fatalf("قائمةُ المندوب: %s", l)
	}
	// **مندوبٌ آخر ⇒ ممنوعٌ كتابةً وقراءة.**
	if r := h.POST("/api/v1/rep/merchants/"+m.ID+"/branch-requests", rep2.Token, branchBody(uniq("س"))); r.Code != 403 {
		t.Fatalf("مندوبٌ غريب: %s", r)
	}
	if r := h.GET("/api/v1/rep/merchants/"+m.ID+"/branch-requests", rep2.Token); r.Code != 403 {
		t.Fatalf("قراءةُ مندوبٍ غريب: %s", r)
	}
	// **وصاحبُ المتجر لا يدخل بابَ المندوب.**
	if r := h.POST("/api/v1/rep/merchants/"+m.ID+"/branch-requests", m.Owner.Token, branchBody(uniq("م"))); r.Code != 403 {
		t.Fatalf("تاجرٌ بباب المندوب: %s", r)
	}
}

func TestBranchRequest_ApproveCreatesLinkedStoreWithCommission(t *testing.T) {
	h := New(t)
	f := h.Factory()
	admin := h.NewUser("admin")
	rep := f.RepAccount()
	m := f.Merchant(OwnedByRep(rep.ID), Commission(6))
	name := uniq("فرعٌ موافَق")

	cr := h.POST("/api/v1/rep/merchants/"+m.ID+"/branch-requests", rep.Token, branchBody(name))
	if cr.Code != 201 {
		t.Fatal(cr)
	}
	id := cr.JSON()["id"].(string)

	// **طابورُ المكتب يعرضه ومعه النسبةُ المقترحة = نسبةُ الأمّ.**
	q := h.GET("/api/v1/admin/branch-requests", admin.Token)
	if q.Code != 200 {
		t.Fatalf("الطابور: %s", q)
	}
	found := false
	for _, it := range q.JSON()["requests"].([]any) {
		row := it.(map[string]any)
		if row["id"] == id {
			found = true
			if d, _ := row["default_commission_percent"].(float64); d != 6 {
				t.Fatalf("النسبةُ المقترحةُ نسبةُ الأمّ (6): %v", row["default_commission_percent"])
			}
		}
	}
	if !found {
		t.Fatalf("الطلبُ غائبٌ عن الطابور: %s", q)
	}
	if c := h.GET("/api/v1/admin/branch-requests/count", admin.Token); c.Code != 200 {
		t.Fatalf("العدّاد: %s", c)
	}

	// **غيرُ المكتب لا يوافق.**
	if r := h.POST("/api/v1/admin/branch-requests/"+id+"/approve", m.Owner.Token, map[string]any{}); r.Code != 403 {
		t.Fatalf("تاجرٌ يوافق: %s", r)
	}

	ap := h.POST("/api/v1/admin/branch-requests/"+id+"/approve", admin.Token, map[string]any{"commission_percent": 7.5})
	if ap.Code != 200 {
		t.Fatalf("الموافقة: %s", ap)
	}
	branchID, _ := ap.JSON()["branch_merchant_id"].(string)
	if branchID == "" {
		t.Fatalf("لا معرّفَ للفرع: %s", ap)
	}
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM merchants WHERE id = $1::uuid`, branchID) })

	var (
		owner, parentOwner, cat, parentCat string
		pct                                float64
		rep2                               *string
		cityLat                            float64
	)
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT b.owner_user_id::text, b.category_id::text, b.commission_percent::float8, b.sales_rep_user_id::text,
		       ST_Y(b.location::geometry), p.owner_user_id::text, p.category_id::text
		  FROM merchants b, merchants p WHERE b.id = $1::uuid AND p.id = $2::uuid`, branchID, m.ID).
		Scan(&owner, &cat, &pct, &rep2, &cityLat, &parentOwner, &parentCat); err != nil {
		t.Fatal(err)
	}
	if owner != parentOwner {
		t.Fatalf("**الفرعُ لحساب الصاحب نفسِه**: %s مقابل %s", owner, parentOwner)
	}
	if cat != parentCat {
		t.Fatal("التصنيفُ يُنسخ من الأمّ")
	}
	if pct != 7.5 {
		t.Fatalf("العمولةُ التي حدّدها المكتب: %v", pct)
	}
	if rep2 == nil || *rep2 != rep.ID {
		t.Fatal("المندوبُ يُنسخ من الأمّ")
	}
	if cityLat < 35.95 || cityLat > 35.96 {
		t.Fatalf("موقعُ الفرع من الطلب: %v", cityLat)
	}

	// **والفرعُ يظهر في متاجر الصاحب نفسِه (مبدّل المتاجر).**
	st := h.GET("/api/v1/merchant/stores", m.Owner.Token)
	if st.Code != 200 || len(st.JSON()["stores"].([]any)) != 2 {
		t.Fatalf("متاجرُ الصاحب بعد الموافقة: %s", st)
	}
	// **وإشعارٌ للمندوب الطالب.**
	if branchNotifCount(t, h, rep.ID, "تمت الموافقة على فرعك") != 1 {
		t.Fatal("لا إشعارَ بالموافقة لصاحب الطلب")
	}
	// **والقرارُ لا يتكرّر.**
	if again := h.POST("/api/v1/admin/branch-requests/"+id+"/approve", admin.Token, map[string]any{}); again.Code != 409 {
		t.Fatalf("موافقةٌ ثانية: %s", again)
	}
	if again := h.POST("/api/v1/admin/branch-requests/"+id+"/reject", admin.Token, map[string]any{}); again.Code != 409 {
		t.Fatalf("رفضٌ بعد الموافقة: %s", again)
	}
	// **وحالتُه عند الصاحب «تمت الموافقة».**
	if l := h.GET("/api/v1/rep/merchants/"+m.ID+"/branch-requests", rep.Token); l.Code != 200 {
		t.Fatal(l)
	} else if s := l.JSON()["requests"].([]any)[0].(map[string]any)["status"]; s != "approved" {
		t.Fatalf("الحالةُ بعد الموافقة: %v", s)
	}
}

func TestBranchRequest_ApproveDefaultsToParentCommission(t *testing.T) {
	h := New(t)
	f := h.Factory()
	admin := h.NewUser("admin")
	m := f.Merchant(Commission(4))
	cr := h.POST("/api/v1/merchant/stores/"+m.ID+"/branch-requests", m.Owner.Token, branchBody(uniq("فرع")))
	if cr.Code != 201 {
		t.Fatal(cr)
	}
	id := cr.JSON()["id"].(string)
	ap := h.POST("/api/v1/admin/branch-requests/"+id+"/approve", admin.Token, map[string]any{})
	if ap.Code != 200 {
		t.Fatalf("الموافقة: %s", ap)
	}
	branchID := ap.JSON()["branch_merchant_id"].(string)
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM merchants WHERE id = $1::uuid`, branchID) })
	var pct float64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT commission_percent::float8 FROM merchants WHERE id = $1::uuid`, branchID).Scan(&pct); err != nil || pct != 4 {
		t.Fatalf("الافتراضُ نسبةُ الأمّ: %v %v", pct, err)
	}
	if branchNotifCount(t, h, m.Owner.ID, "تمت الموافقة على فرعك") != 1 {
		t.Fatal("لا إشعارَ لصاحب المتجر")
	}
	// **عمولةٌ خارجَ الحدّ تُردّ ولا يُنشأ شيء.**
	cr2 := h.POST("/api/v1/merchant/stores/"+m.ID+"/branch-requests", m.Owner.Token, branchBody(uniq("فرع٢")))
	id2 := cr2.JSON()["id"].(string)
	if bad := h.POST("/api/v1/admin/branch-requests/"+id2+"/approve", admin.Token, map[string]any{"commission_percent": 150}); bad.Code == 200 {
		t.Fatalf("عمولةُ ١٥٠٪ قُبلت: %s", bad)
	}
	var st string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT status FROM branch_requests WHERE id = $1::uuid`, id2).Scan(&st)
	if st != "pending" {
		t.Fatalf("بقي الطلبُ بانتظار القرار بعد فشل الموافقة: %s", st)
	}
}

func TestBranchRequest_Reject(t *testing.T) {
	h := New(t)
	f := h.Factory()
	admin := h.NewUser("admin")
	m := f.Merchant()
	cr := h.POST("/api/v1/merchant/stores/"+m.ID+"/branch-requests", m.Owner.Token, branchBody(uniq("فرعٌ مرفوض")))
	if cr.Code != 201 {
		t.Fatal(cr)
	}
	id := cr.JSON()["id"].(string)

	if r := h.POST("/api/v1/admin/branch-requests/"+id+"/reject", m.Owner.Token, map[string]any{"reason": "x"}); r.Code != 403 {
		t.Fatalf("تاجرٌ يرفض: %s", r)
	}
	rj := h.POST("/api/v1/admin/branch-requests/"+id+"/reject", admin.Token, map[string]any{"reason": "الموقعُ غيرُ واضح"})
	if rj.Code != 200 {
		t.Fatalf("الرفض: %s", rj)
	}
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM merchants WHERE owner_user_id = $1::uuid`, m.Owner.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("الرفضُ لا يُنشئ متجراً: %d", n)
	}
	var body string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT body FROM notifications WHERE user_id = $1::uuid AND title = 'رُفض طلب الفرع'`, m.Owner.ID).Scan(&body); err != nil {
		t.Fatalf("لا إشعارَ بالرفض: %v", err)
	}
	if want := "الموقعُ غيرُ واضح"; len(body) < len(want) {
		t.Fatalf("السببُ في الإشعار: %q", body)
	}
	l := h.GET("/api/v1/merchant/branch-requests", m.Owner.Token)
	row := l.JSON()["requests"].([]any)[0].(map[string]any)
	if row["status"] != "rejected" || row["reject_reason"] != "الموقعُ غيرُ واضح" {
		t.Fatalf("حالةُ المرفوض عند الصاحب: %v", row)
	}
	if again := h.POST("/api/v1/admin/branch-requests/"+id+"/reject", admin.Token, map[string]any{}); again.Code != 409 {
		t.Fatalf("رفضٌ ثانٍ: %s", again)
	}
	// **وبعد الرفض يجوز طلبٌ بالاسم نفسِه من جديد** — الفهرسُ الفريدُ للمفتوح وحدَه.
	if r := h.POST("/api/v1/merchant/stores/"+m.ID+"/branch-requests", m.Owner.Token, branchBody(row["name"].(string))); r.Code != 201 {
		t.Fatalf("إعادةُ الطلب بعد الرفض: %s", r)
	}
}
