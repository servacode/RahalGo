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

// TestBatch4_DriverReportPrivacyAgainstCustomer — **بلاغُ سائقٍ ضدَّ الزبونِ
// عبرَ المسارِ الحقيقيّ** (Batch 4، C1/C2/C4). يُنشئ طلباً حقيقيّاً، يُسنِد
// سائقاً ويُغلقه داخلَ المهلة (تجهيز)، ثمّ يرفع السائقُ بلاغاً عبرَ نقطةِ
// النهاية الحقيقيّة `POST /driver/orders/{id}/report`. ويتحقّق أنّ:
//   - ردَّ البلاغ أدنى (id/number/status) بلا هاتفِ الزبونِ أو اسمِه أو
//     معرّفِ من أُبلِغ عنه أو سببِه (C4)،
//   - البلاغَ لا يظهر في «شكاوى قدّمتُها» للزبون (C1)،
//   - يظهر في كشفِ «ضدّي» ملثوماً بلا هويّةِ المُبلِّغ ولا سببِه (C2).
func TestBatch4_DriverReportPrivacyAgainstCustomer(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	drv := h.NewUser("driver")
	item := h.NewItem(5000)
	ctx := ctxBG()

	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	if oid == "" {
		t.Fatalf("**تعذّر إنشاءُ الطلب**: %d %s", made.Code, trimBody(made))
	}
	// **تجهيزٌ لا مسارٌ مشهود**: يُسنَد السائقُ ويُغلق الطلبُ داخلَ المهلة.
	if _, err := h.Pool.Exec(ctx, `
		UPDATE orders SET status = 'delivered', closed_at = now(), driver_id = $2::uuid
		WHERE id = $1::uuid`, oid, drv.ID); err != nil {
		t.Fatalf("تجهيزُ الطلب: %v", err)
	}

	// ── البلاغُ عبرَ المسارِ الحقيقيّ — والردُّ أدنى (C4) ──────────────────
	drvTok := h.TokenFor(drv.ID, "driver")
	rep := h.POST("/api/v1/driver/orders/"+oid+"/report", drvTok,
		map[string]any{"reason": "customer_conduct", "note": "شاهدٌ آليّ"})
	if rep.Code != 200 {
		t.Fatalf("**بلاغُ السائق فشل**: %d %s", rep.Code, trimBody(rep))
	}
	rb := rep.JSON()
	if rb["id"] == nil || rb["number"] == nil || rb["status"] == nil {
		t.Errorf("**ردُّ البلاغ ناقصٌ للحقولِ الآمنة (id/number/status)**: %s", trimBody(rep))
	}
	body := string(rep.Body)
	for _, leak := range []string{cust.Phone, cust.ID, drv.Phone, "against_user", "customer_conduct", "author"} {
		if leak != "" && strings.Contains(body, leak) {
			t.Errorf("**تسريبٌ في ردّ البلاغ (%q)**: %s", leak, body)
		}
	}
	tnum := int64(rb["number"].(float64))

	// ── (C1) لا يظهر في «شكاوى قدّمتُها» للزبون ───────────────────────────
	mine := h.GET("/api/v1/my/tickets", cust.Token)
	if strings.Contains(string(mine.Body), fmt.Sprintf(`"number":%d`, tnum)) {
		t.Errorf("**بلاغُ السائقِ ضدَّ الزبونِ ظهر في «شكاواي» — تسريب**: %s", trimBody(mine))
	}

	// ── (C2) يظهر في «ضدّي» ملثوماً بلا هويّةِ المُبلِّغِ ولا سببِه ─────────
	me := h.GET("/api/v1/me/reputation", cust.Token)
	if me.Code != 200 {
		t.Fatalf("me/reputation = %d: %s", me.Code, trimBody(me))
	}
	mebody := string(me.Body)
	if !strings.Contains(mebody, fmt.Sprintf(`"number":%d`, tnum)) {
		t.Errorf("**بلاغٌ ضدَّ الزبونِ غائبٌ عن كشفِ «ضدّي»**: %s", mebody)
	}
	for _, leak := range []string{drv.ID, drv.Phone, "customer_conduct"} {
		if leak != "" && strings.Contains(mebody, leak) {
			t.Errorf("**كشفُ «ضدّي» يكشف هويّةَ المُبلِّغِ أو سببَه (%q)**: %s", leak, mebody)
		}
	}
}

// TestBatch4_MerchantReportResponseMinimal — **ردُّ بلاغِ المتجرِ أدنى** (Batch
// 4، C4). المتجرُ يُبلّغ عن سائقٍ عبرَ المسارِ الحقيقيّ، والردُّ إقرارُ تسجيلٍ
// (id/number/status) بلا هاتفِ زبونٍ أو اسمِه أو سببٍ أو جهةٍ — صفُّ المكتبِ
// الكاملُ لبابِ الأدمن وحدَه.
func TestBatch4_MerchantReportResponseMinimal(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	drv := h.NewUser("driver")
	item := h.NewItem(5000)
	ctx := ctxBG()

	var ownerID string
	if err := h.Pool.QueryRow(ctx,
		`SELECT owner_user_id::text FROM merchants WHERE id = $1::uuid`, item.MerchantID).Scan(&ownerID); err != nil {
		t.Fatalf("مالكُ المتجر: %v", err)
	}

	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	if oid == "" {
		t.Fatalf("**تعذّر إنشاءُ الطلب**: %d %s", made.Code, trimBody(made))
	}
	if _, err := h.Pool.Exec(ctx, `
		UPDATE orders SET status = 'delivered', closed_at = now(), driver_id = $2::uuid
		WHERE id = $1::uuid`, oid, drv.ID); err != nil {
		t.Fatalf("تجهيزُ الطلب: %v", err)
	}

	mtok := h.TokenFor(ownerID, "merchant")
	rep := h.POST("/api/v1/merchant/orders/"+oid+"/report", mtok,
		map[string]any{"reason": "driver_late_pickup", "note": "شاهدٌ آليّ"})
	// **بلاغُ المتجرِ يُقرّ بـ201** (إنشاء) بخلافِ بلاغِ السائقِ (200).
	if rep.Code != 201 {
		t.Fatalf("**بلاغُ المتجر فشل**: %d %s", rep.Code, trimBody(rep))
	}
	rb := rep.JSON()
	if rb["id"] == nil || rb["number"] == nil || rb["status"] == nil {
		t.Errorf("**ردُّ بلاغِ المتجر ناقصٌ للحقولِ الآمنة**: %s", trimBody(rep))
	}
	body := string(rep.Body)
	for _, leak := range []string{cust.Phone, cust.ID, drv.Phone, "against_user", "driver_late_pickup", "author"} {
		if leak != "" && strings.Contains(body, leak) {
			t.Errorf("**تسريبٌ في ردّ بلاغِ المتجر (%q)**: %s", leak, body)
		}
	}
}
