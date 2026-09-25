package qa

import (
	"fmt"
	"strings"
	"testing"
)

// TestBatch4_MyTicketsFiledByMeOnly — **«شكاوى قدّمتها» = ما قدّمه هو وحدَه**
// (Batch 4، C1). بلاغُ سائقٍ ضدَّ الزبون يحمل customer_id = صاحبَ الطلب، فكان
// يتسلّل إلى قائمة شكاواه ويكشف أنّ بلاغاً رُفع عليه (تسريب). بعد الترشيح
// بـ created_by + opened_by_customer لا يظهر إلّا ما فتحه هو.
func TestBatch4_MyTicketsFiledByMeOnly(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	drv := h.NewUser("driver")
	ctx := ctxBG()

	var mineNum, againstNum int64
	if err := h.Pool.QueryRow(ctx, `
		INSERT INTO tickets (customer_id, subject, status, created_by, opened_by_customer, reason)
		VALUES ($1::uuid, 'شكواي على الطلب', 'open', $1::uuid, true, 'other')
		RETURNING number`, cust.ID).Scan(&mineNum); err != nil {
		t.Fatalf("إدراجُ شكوى الزبون: %v", err)
	}
	if err := h.Pool.QueryRow(ctx, `
		INSERT INTO tickets (customer_id, subject, status, created_by, opened_by_customer, reason, against_user_id)
		VALUES ($1::uuid, 'بلاغُ سائق', 'open', $2::uuid, false, 'customer_conduct', $1::uuid)
		RETURNING number`, cust.ID, drv.ID).Scan(&againstNum); err != nil {
		t.Fatalf("إدراجُ بلاغِ السائق: %v", err)
	}

	res := h.GET("/api/v1/my/tickets", cust.Token)
	if res.Code != 200 {
		t.Fatalf("my/tickets = %d: %s", res.Code, trimBody(res))
	}
	body := string(res.Body)
	if !strings.Contains(body, fmt.Sprintf(`"number":%d`, mineNum)) {
		t.Errorf("**شكوى الزبون التي قدّمها غائبةٌ عن قائمته**: %s", body)
	}
	if strings.Contains(body, fmt.Sprintf(`"number":%d`, againstNum)) {
		t.Errorf("**بلاغُ سائقٍ ضدَّ الزبون ظهر في «شكاواي» — تسريب**")
	}
	if strings.Contains(body, "customer_conduct") {
		t.Errorf("**سببُ بلاغٍ ضدَّ الزبون ظهر في قائمته — تسريب**")
	}
}

// TestBatch4_SignupRetryDoesNotLockLogin — **إعادةُ محاولةِ تسجيلٍ ضاع ردُّها
// تقفل عدّادَ التسجيل لا عدّادَ الدخول** (Batch 4، SG2): يبقى الدخولُ بكلمةِ
// المرور (مسارُ الاستعادة) مفتوحاً بعد إعاداتٍ كثيرة. **والتكاملُ يحفظ
// حمايةَ الاستيلاء** (SU01–10): التسجيلُ نفسُه يبقى مرفوضاً على حسابٍ قائم.
func TestBatch4_SignupRetryDoesNotLockLogin(t *testing.T) {
	h := New(t)
	suPolicy(h, false) // بلا رمزٍ للتسجيل — كالإنتاج
	phone := uniqPhone()
	suDropPhone(t, h, phone)
	ip := suIP(t, h)

	// أوّلُ تأكيدٍ ينشئ الحساب بكلمةٍ معروفة (كلمةُ suConfirm الثابتة).
	first := suConfirm(h, ip, phone, "")
	if first.Code != 200 {
		t.Fatalf("**أوّلُ تسجيلٍ فشل** — %d %s", first.Code, trimBody(first))
	}
	// إعاداتٌ (كأنّ ردَّها ضاع على العميل) — تقفل عدّادَ التسجيل لا الدخول.
	last := 0
	for i := 0; i < 8; i++ {
		last = suConfirm(h, ip, phone, "").Code
	}
	if last < 400 {
		t.Fatalf("**إعادةُ تسجيلٍ على حسابٍ قائمٍ قُبلت** — انهارت حمايةُ الاستيلاء: %d", last)
	}

	// والدخولُ بالكلمةِ نفسِها — مسارُ الاستعادة — يبقى مفتوحاً غيرَ مقفول.
	login := h.Call("POST", "/api/v1/auth/login", "",
		map[string]any{"phone": phone, "password": "Attacker-Pass-999"},
		map[string]string{"X-Real-IP": ip})
	if login.Code != 200 {
		t.Fatalf("**قُفل الدخولُ بإعاداتِ التسجيل — الاستعادةُ مسدودة**: %d %s",
			login.Code, trimBody(login))
	}
	if suToken(login) == "" {
		t.Errorf("**دخولٌ ناجحٌ بلا جلسة**")
	}
}
