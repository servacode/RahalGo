package server

// **قبولُ طلبات الانضمام تلقائياً** — قرارُ المالك ٢٠٢٦-١٠-٠٨.
//
//  1. مطفأً ⇒ الطلبُ يبقى «جديداً» ولا متجر.
//  2. مشغَّلاً ⇒ متجرٌ وصاحبُه بكلمةٍ مؤقّتة والطلبُ محوَّل، والفاعلُ حسابُ النظام.
//  3. سقوطُ التحويل ⇒ نداءُ المندوب ناجحٌ والطلبُ «جديد» ينتظر المكتب.

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func (f *leadFx) setAutoApprove(t *testing.T, on bool) {
	t.Helper()
	ctx := context.Background()
	if err := f.srv.settings.Set(ctx, "leads.auto_approve", on, nil); err != nil {
		t.Fatalf("ضبطُ القبول التلقائيّ: %v", err)
	}
	t.Cleanup(func() {
		_ = f.srv.settings.Set(context.Background(), "leads.auto_approve", false, nil)
	})
}

// repLead يرفع طلباً من بابِ المندوب ويُعيد معرّفَه.
func (f *leadFx) repLead(t *testing.T, name, phone string, withPoint bool) string {
	t.Helper()
	ctx := context.Background()
	_, _ = f.pool.Exec(ctx, `UPDATE users SET whatsapp_verified_at = now() WHERE id = $1`, f.rep)
	body := `{"store_name":"` + name + `","owner_name":"صاحبُه","phone":"` + phone +
		`","category_id":"` + f.category + `","district_id":"` + liveDistrict(t, f.driverFixture) + `"`
	if withPoint {
		body += `,"lat":35.9528,"lng":39.0079`
	}
	body += `}`
	w := postRepLead(t, f.driverFixture, f.rep, body)
	if w.Code != http.StatusCreated {
		t.Fatalf("رفعُ الطلب ردّ %d: %s", w.Code, w.Body.String())
	}
	var env struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil || env.Data.ID == "" {
		t.Fatalf("لا معرّفَ في الجواب: %s", w.Body.String())
	}
	id := env.Data.ID
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `UPDATE merchant_leads SET merchant_id = NULL WHERE id = $1`, id)
		_, _ = f.pool.Exec(c, `DELETE FROM merchants WHERE lead_id = $1`, id)
		_, _ = f.pool.Exec(c, `DELETE FROM merchant_leads WHERE id = $1`, id)
	})
	return id
}

func (f *leadFx) leadState(t *testing.T, id string) (status string, merchant *string) {
	t.Helper()
	if err := f.pool.QueryRow(context.Background(),
		`SELECT status, merchant_id::text FROM merchant_leads WHERE id = $1`, id).
		Scan(&status, &merchant); err != nil {
		t.Fatalf("قراءةُ الطلب: %v", err)
	}
	return
}

func TestLeadAutoApprove_OffStaysPending(t *testing.T) {
	f := newLeadFx(t)
	f.setAutoApprove(t, false)
	id := f.repLead(t, "متجرُ القبول اليدويّ", f.freshPhone(t), true)
	status, merchant := f.leadState(t, id)
	if status != "new" || merchant != nil {
		t.Fatalf("مطفأً: الحالةُ %q والمتجر %v — والمنتظَرُ «new» بلا متجر", status, merchant)
	}
}

func TestLeadAutoApprove_OnConvertsWithSystemActor(t *testing.T) {
	f := newLeadFx(t)
	f.setAutoApprove(t, true)
	ctx := context.Background()
	phone := f.freshPhone(t)
	id := f.repLead(t, "متجرُ القبول التلقائيّ", phone, true)

	status, merchant := f.leadState(t, id)
	if status != "converted" || merchant == nil {
		t.Fatalf("مشغَّلاً: الحالةُ %q والمتجر %v — والمنتظَرُ محوَّلاً بمتجر", status, merchant)
	}
	var owner string
	var must, hasExp bool
	if err := f.pool.QueryRow(ctx, `
		SELECT u.id::text, u.must_change_password, u.temp_password_expires_at IS NOT NULL
		  FROM merchants mm JOIN users u ON u.id = mm.owner_user_id
		 WHERE mm.id = $1`, *merchant).Scan(&owner, &must, &hasExp); err != nil {
		t.Fatalf("صاحبُ المتجر: %v", err)
	}
	if !must || !hasExp {
		t.Errorf("صاحبُ المتجر بلا كلمةٍ مؤقّتة: يُجبَر=%v مهلة=%v", must, hasExp)
	}
	got := waitAudit(t, f.driverFixture, "lead", id, "ops.lead_auto_approved")
	sys := f.srv.systemActorID(ctx)
	if sys == "" || got["_actor"] != sys || got["auto"] != true {
		t.Fatalf("سطرُ القبول التلقائيّ %v — والمنتظَرُ فاعلُ النظام %q وعلامةُ auto", got, sys)
	}
	waitAudit(t, f.driverFixture, "user", owner, "admin.welcome_message")
}

func TestLeadAutoApprove_ConversionFailureKeepsLead(t *testing.T) {
	f := newLeadFx(t)
	f.setAutoApprove(t, true)
	// **بلا نقطةٍ على الخريطة** — `CreateMerchantTx` يرفضه، فيسقط التحويل.
	id := f.repLead(t, "متجرٌ بلا نقطة", f.freshPhone(t), false)
	status, merchant := f.leadState(t, id)
	if status != "new" || merchant != nil {
		t.Fatalf("تحويلٌ ساقط: الحالةُ %q والمتجر %v — والمنتظَرُ «new» ينتظر المكتب", status, merchant)
	}
}
