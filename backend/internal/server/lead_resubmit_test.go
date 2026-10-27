package server

// ══════════════════════════════════════════════════════════════════════
// **«بحاجة معلومات» يصلحها المندوب ويعيدها — قرارُ المالك ٢٠٢٦-١٠-٢٧ (البند ١٧)**
// ══════════════════════════════════════════════════════════════════════
//
// كانت الإدارةُ تُرجع الطلبَ بملاحظة، **والمندوبُ لا يقدر أن يصلحه** — فيبقى
// معلّقاً ورقمُه محجوزٌ به. والآن `PUT /rep/leads/{id}`:
//
//   - **صاحبُ الطلب وحدَه**، وفي `needs_info` وحدَها.
//   - **يعود «جديداً»** بالحقول الجديدة، **وملاحظةُ الإدارة تبقى**.
//   - **وفي السجلّ** `ops.lead_resubmitted`.

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestRepResubmitLead_NeedsInfoBackToNew(t *testing.T) {
	f := newLeadFx(t)
	ctx := context.Background()
	dist := liveDistrict(t, f.driverFixture)

	id := f.newLead(t, "بقالةُ الفحص", f.freshPhone(t), f.rep)
	body := func(name string) string {
		return `{"store_name":"` + name + `","owner_name":"صاحبُه","phone":"` + f.freshPhone(t) +
			`","area":"مقابل الجامع","district_id":"` + dist + `","category_id":"` + f.category +
			`","lat":35.95,"lng":39.01}`
	}
	resubmit := func(user, b string) int {
		return callWithID(f.driverFixture, f.srv.handleRepResubmitLead, id, user, []string{"sales"}, b).Code
	}

	// **وهو «جديد» — لا يُعدَّل:** ينتظر المكتبَ كما هو.
	if code := resubmit(f.rep, body("قبل الطلب")); code != http.StatusConflict {
		t.Fatalf("طلبٌ جديدٌ عُدِّل من المندوب (%d) — **والتعديلُ لـ«بحاجة معلومات» وحدَها**", code)
	}

	const note = "الموقع على الخريطة غير دقيق"
	if code, out := f.setStatus(t, id, `{"status":"needs_info","note":"`+note+`"}`); code != http.StatusOK {
		t.Fatalf("«بحاجة معلومات» ردّت %d: %v", code, out)
	}

	// **ومندوبٌ آخرُ لا يراه أصلاً** — «غير موجود» لا «ممنوع».
	other := testdb.NewUser(t, f.pool, "sales")
	if code := resubmit(other, body("سرقة")); code != http.StatusNotFound {
		t.Errorf("مندوبٌ غريبٌ عدّل طلبَ غيره أو عرف بوجوده (%d)", code)
	}

	// **وبلا منطقةٍ يُردّ** — كما في الإنشاء.
	bad := `{"store_name":"س","owner_name":"ص","phone":"` + f.freshPhone(t) +
		`","category_id":"` + f.category + `"}`
	if code := resubmit(f.rep, bad); code < 400 {
		t.Errorf("إعادةُ إرسالٍ بلا منطقةٍ قُبلت (%d)", code)
	}

	// **وصاحبُه يصلحه فيعود «جديداً»** بالحقول الجديدة والملاحظةُ باقية.
	if code := resubmit(f.rep, body("بقالةُ الفحص المصحَّحة")); code != http.StatusOK {
		t.Fatalf("صاحبُ الطلب لم يقدر أن يعيده (%d)", code)
	}
	var status, name, decision string
	var lat *float64
	if err := f.pool.QueryRow(ctx,
		`SELECT status, store_name, decision_note, lat FROM merchant_leads WHERE id = $1`, id).
		Scan(&status, &name, &decision, &lat); err != nil {
		t.Fatal(err)
	}
	if status != "new" || name != "بقالةُ الفحص المصحَّحة" {
		t.Errorf("بعد الإعادة: الحالةُ %q والاسم %q — **والمطلوب «new» بالاسم الجديد**", status, name)
	}
	if decision != note {
		t.Errorf("**ملاحظةُ الإدارة مُحيت** بعد الإعادة: %q", decision)
	}
	if lat == nil || *lat != 35.95 {
		t.Errorf("النقطةُ لم تتبدّل: %v", lat)
	}
	meta := waitAudit(t, f.driverFixture, "lead", id, "ops.lead_resubmitted")
	if meta["note"] != note {
		t.Errorf("قيدُ الإعادة بلا ملاحظة الإدارة: %v", meta)
	}

	// **وضغطةٌ ثانيةٌ تُردّ** — عاد «جديداً» فلا يُعدَّل.
	if code := resubmit(f.rep, body("مرّة ثانية")); code != http.StatusConflict {
		t.Errorf("إعادةٌ ثانيةٌ مرّت (%d)", code)
	}

	// **وقائمتُه تحمل معرّفاتِ اختياراته** ليملأ نموذجُ التعديل ما اختاره.
	w := callWithID(f.driverFixture, f.srv.handleRepLeads, "", f.rep, []string{"sales"}, "")
	var mine struct {
		Data []struct {
			ID            string  `json:"id"`
			CategoryID    *string `json:"category_id"`
			DistrictID    *string `json:"district_id"`
			GovernorateID *string `json:"governorate_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &mine)
	found := false
	for _, l := range mine.Data {
		if l.ID != id {
			continue
		}
		found = true
		if l.CategoryID == nil || *l.CategoryID != f.category ||
			l.DistrictID == nil || *l.DistrictID != dist || l.GovernorateID == nil {
			t.Errorf("قائمةُ المندوب بلا معرّفات الاختيار: %+v", l)
		}
	}
	if !found {
		t.Error("الطلبُ غاب عن قائمة صاحبه")
	}
}
