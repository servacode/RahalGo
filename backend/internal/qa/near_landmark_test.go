package qa

import (
	"strings"
	"testing"
)

// TestNearLandmark_DriverSeesItCustomerDoesNot **«قرب المعلم» للسائق وحدَه**
// (قرارُ المالك ٢٠٢٦-١٠-٢١): يُحسب أقربُ معلمٍ عند الإنشاء، فيراه السائقُ ملصقاً
// بالعنوان في الحقل الذي تعرضه نسختُه الحاليّة — **وللزبون عنوانُه كما كتبه.**
func TestNearLandmark_DriverSeesItCustomerDoesNot(t *testing.T) {
	h := New(t)
	name := uniq("جامع الاختبار ")
	// **نقطةٌ لا يستعملها فحصٌ آخر، ومعلمٌ على عشرة أمتارٍ منها** — فمتاجرُ الفحوص
	// الأخرى (كلُّها على 35.9506/39.0094) لا تسبقه.
	const lat, lng = 35.9431, 39.0012
	if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO map_landmarks (name, kind, point, source)
		VALUES ($1, 'amenity=place_of_worship', ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, 'qa')`,
		name, lng, lat+0.00009); err != nil {
		t.Fatalf("معلمُ الاختبار: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_landmarks WHERE name = $1`, name)
	})

	cust := h.Customer()
	created := h.POST("/api/v1/orders/custom", cust.Token, map[string]any{
		"request": "كيلو لحمٍ من ملحمة الحيّ", "address_text": "الرقة — شارع الاختبار",
		"lat": lat, "lng": lng,
	})
	if created.Code != 201 {
		t.Fatalf("إنشاءُ الطلب: %s", created)
	}
	id, _ := created.JSON()["id"].(string)
	var near string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(near_landmark, '') FROM orders WHERE id = $1::uuid`,
		id).Scan(&near); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(near, "قرب "+name) || !strings.Contains(near, "(10 م)") {
		t.Fatalf("near_landmark = %q — والمتوقَّع «قرب %s (10 م)»", near, name)
	}

	drv := h.NewUser("driver")
	assignDriver(t, h, id, drv.ID)
	res := h.GET("/api/v1/driver/orders", drv.Token)
	if res.Code != 200 {
		t.Fatalf("طلباتُ السائق: %s", res)
	}
	if !strings.Contains(string(res.Body), "الرقة — شارع الاختبار · ") || !strings.Contains(string(res.Body), name) {
		t.Fatalf("السائقُ لا يرى المعلمَ ملصقاً بالعنوان: %s", res)
	}

	mine := h.GET("/api/v1/my/orders/"+id, cust.Token)
	if mine.Code != 200 {
		t.Fatalf("طلبُ الزبون: %s", mine)
	}
	if strings.Contains(string(mine.Body), name) {
		t.Fatalf("**الزبونُ يرى المعلمَ في عنوانه** — والعنوانُ له كما كتبه: %s", mine)
	}
	if addr, _ := mine.JSON()["address_text"].(string); addr != "الرقة — شارع الاختبار" {
		t.Fatalf("عنوانُ الزبون تبدّل: %q", addr)
	}
}
