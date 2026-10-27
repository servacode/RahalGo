package server

// **طلباتُ «بحاجة معلومات» تُعدّ في لوحة المندوب** (٢٠٢٦-١٠-٢٨)
//
// كانت `pending_leads` تعدّ `new` وحدَها — فطلبٌ ردّته الإدارةُ تسأل عن معلومةٍ
// لا يظهر في اللوحة، **ولا يعلم المندوبُ أنّ عليه شيئاً.** فصار له عدّادُه:
// `needs_info_leads`، **ولا يدخل في `pending_leads`.**

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestRepMe_CountsNeedsInfoLeadsApart(t *testing.T) {
	f := newLeadFx(t)

	f.newLead(t, "متجرُ الانتظار", f.freshPhone(t), f.rep)
	asked := f.newLead(t, "متجرُ السؤال", f.freshPhone(t), f.rep)
	if code, out := f.setStatus(t, asked, `{"status":"needs_info","note":"الموقع على الخريطة غير دقيق"}`); code != http.StatusOK {
		t.Fatalf("needs_info: %d %v", code, out)
	}

	w := callWithID(f.driverFixture, f.srv.handleRepMe, "", f.rep, []string{"sales"}, "{}")
	if w.Code != http.StatusOK {
		t.Fatalf("rep/me: %d %s", w.Code, w.Body.String())
	}
	var env struct {
		Data struct {
			PendingLeads   *int `json:"pending_leads"`
			NeedsInfoLeads *int `json:"needs_info_leads"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v — %s", err, w.Body.String())
	}
	if env.Data.NeedsInfoLeads == nil {
		t.Fatalf("needs_info_leads missing from response: %s", w.Body.String())
	}
	if got := *env.Data.NeedsInfoLeads; got != 1 {
		t.Fatalf("needs_info_leads = %d, want 1", got)
	}
	if env.Data.PendingLeads == nil || *env.Data.PendingLeads != 1 {
		t.Fatalf("pending_leads should count only new: %s", w.Body.String())
	}
}
