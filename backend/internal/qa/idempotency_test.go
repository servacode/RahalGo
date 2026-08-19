package qa

// **الطبقةُ الرابعة — منعُ التكرار: أخطرُ ما في المال.**
//
// المعرّفات: `IDEM-*` · الوسم: `@api @critical @release @regression`
//
// # لماذا لها حزمةٌ وحدَها
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٩: «هذه أولويّةٌ خاصّةٌ بسبب المشاكل
//
//	المكتشفة… أريد Tests دائمةً لجميع العمليّات الحسّاسة».)
//
// **والخطرُ ليس الضغطتين المتتاليتين** — تلك يمنعها `busy` في الشاشة.
// **الخطرُ أن يُنشأ الطلبُ في الخادم ثمّ ينقطع قبل أن يصل الردّ**: يرى
// «تعذّر» فيضغط ثانيةً — **فطلبان وسائقان وخصمان.**
//
// # وما كُشف قبل هذه الحزمة
//
// **BUG-002**: `/orders` ملفوفةٌ و`/orders/custom` لا — بابان لفعلٍ
// واحدٍ أحدُهما محروس. **وBUG-001**: الخادمُ يحرس والتطبيقُ لا يرسل
// المفتاحَ أصلا. **وكلاهما أُصلح ٢٠٢٦-٠٨-١٩ قبل وجود هذه الحزمة** —
// فتبقى حارساً لا كاشفا.

import (
	"net/http"
	"testing"
)

// TestIDEM_001_SameKeyOneOrder **مفتاحٌ واحدٌ = طلبٌ واحد.**
func TestIDEM_001_SameKeyOneOrder(t *testing.T) {
	h := New(t)
	u := h.Customer()
	item := h.NewItem(1500)
	key := uniq("idem")

	before := h.CountOrders(u.ID)
	a := h.POSTKey("/api/v1/orders", u.Token, key, orderBody(item, 1))
	if a.Code >= 400 {
		t.Fatalf("IDEM-001 تعذّر النداءُ الأوّل: %s", a)
	}
	b := h.POSTKey("/api/v1/orders", u.Token, key, orderBody(item, 1))
	if b.Code >= 400 {
		t.Fatalf("IDEM-001 الإعادةُ بالمفتاح نفسِه رُدَّت: %s", b)
	}

	if got := h.CountOrders(u.ID) - before; got != 1 {
		t.Errorf("IDEM-001 نداءان بمفتاحٍ واحد أنشآ %d طلبا — يُنتظر 1", got)
	}
	if a.JSON()["id"] != b.JSON()["id"] {
		t.Errorf("IDEM-001 الردّان لطلبين: %v ≠ %v", a.JSON()["id"], b.JSON()["id"])
	}
}

// TestIDEM_002_LostResponse **الردُّ ضاع والزبونُ أعاد.**
//
// **وهي الحالةُ التي بُني لها المفتاح** — تُحاكى بنداءين متطابقين
// بالمفتاح نفسِه، **وهو ما يفعله عميلٌ يحفظ المفتاحَ للمحاولة لا
// للضغطة.**
func TestIDEM_002_LostResponse(t *testing.T) {
	h := New(t)
	u := h.Customer()
	item := h.NewItem(2000)
	key := uniq("idem")
	body := orderBody(item, 3)

	before := h.CountOrders(u.ID)
	_ = h.POSTKey("/api/v1/orders", u.Token, key, body) // **الردُّ يُهمَل — كأنّه ضاع**
	again := h.POSTKey("/api/v1/orders", u.Token, key, body)

	if again.Code >= 400 {
		t.Errorf("IDEM-002 إعادةٌ بعد ضياع الردّ رُدَّت — **الزبونُ لا يستطيع الطلبَ أبدا**: %s", again)
	}
	if got := h.CountOrders(u.ID) - before; got != 1 {
		t.Errorf("IDEM-002 أنشأت %d طلبا — يُنتظر 1", got)
	}
}

// TestIDEM_003_FreshKeyNewOrder **ومفتاحٌ جديدٌ يعني طلباً جديداً.**
//
// **وهذا نصفُ الحارس المنسيّ**: خادمٌ يردّ الطلبَ الأوّلَ دائماً ينجح
// في الاختبارين قبله — **ويمنع الزبونَ من طلبٍ ثانٍ إلى الأبد.**
func TestIDEM_003_FreshKeyNewOrder(t *testing.T) {
	h := New(t)
	u := h.Customer()
	item := h.NewItem(1200)

	before := h.CountOrders(u.ID)
	a := h.POSTKey("/api/v1/orders", u.Token, uniq("idem"), orderBody(item, 1))
	b := h.POSTKey("/api/v1/orders", u.Token, uniq("idem"), orderBody(item, 1))
	if a.Code >= 400 || b.Code >= 400 {
		t.Fatalf("IDEM-003 نداءٌ رُدّ: %s · %s", a, b)
	}
	if got := h.CountOrders(u.ID) - before; got != 2 {
		t.Errorf("IDEM-003 مفتاحان مختلفان أنشآ %d طلبا — يُنتظر 2", got)
	}
	if a.JSON()["id"] == b.JSON()["id"] {
		t.Errorf("IDEM-003 مفتاحان مختلفان ردّا الطلبَ نفسَه: %v", a.JSON()["id"])
	}
}

// TestIDEM_004_KeyIsPerUser **ومفتاحُ زيدٍ لا يمنع طلبَ عمرو.**
//
// **ومفتاحٌ عامٌّ يجعل زبوناً يمنع زبوناً** — بل قد يردّ عليه طلبَ
// غيرِه، وهي تسريبُ بياناتٍ لا منعَ تكرار.
func TestIDEM_004_KeyIsPerUser(t *testing.T) {
	h := New(t)
	a, b := h.Customer(), h.Customer()
	item := h.NewItem(1000)
	key := uniq("shared")

	ra := h.POSTKey("/api/v1/orders", a.Token, key, orderBody(item, 1))
	rb := h.POSTKey("/api/v1/orders", b.Token, key, orderBody(item, 1))
	if ra.Code >= 400 || rb.Code >= 400 {
		t.Fatalf("IDEM-004 نداءٌ رُدّ: %s · %s", ra, rb)
	}
	if ra.JSON()["id"] == rb.JSON()["id"] {
		t.Errorf("IDEM-004 زبونان بمفتاحٍ واحدٍ حصلا على الطلب نفسِه — **تسريب**: %v",
			ra.JSON()["id"])
	}
	if h.CountOrders(b.ID) == 0 {
		t.Error("IDEM-004 مفتاحُ زبونٍ منع زبوناً آخرَ من الطلب")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **و`IDEM-005` نُقلت إلى `CUST-003`** — ٢٠٢٦-٠٨-٢٠
// ══════════════════════════════════════════════════════════════════════
//
// **كانت تتخطّى نفسَها منذ كُتبت**: أرسلت `description` والمحرّكُ يقرأ
// `request`، **فيُردّ النداءُ فتُقرأ «تعذّر التجهيز» لا «الاسمُ خطأ».**
//
// **واختبارٌ يتخطّى دائماً أسوأُ من غيابه** — يُعدّ في التغطية ولا يقيس
// شيئا.
//
// **وحين صحّ الحقلُ صارت تكراراً لـ`CUST-003`** — فأُزيلت، **وحزمةُ
// الطلب الخاصّ أولى بها** (`custom_test.go`).

// TestIDEM_006_MissingKeyStillWorks **وبلا مفتاحٍ يمرّ النداء.**
//
// **والشرطُ مقصود**: بابٌ يرفض بلا مفتاحٍ يكسر كلَّ عميلٍ قديم —
// **إنّما يُوثَّق أنّ العبءَ على المنادي**، وهو نصفُ BUG-001.
func TestIDEM_006_MissingKeyStillWorks(t *testing.T) {
	h := New(t)
	u := h.Customer()
	item := h.NewItem(900)
	got := h.POST("/api/v1/orders", u.Token, orderBody(item, 1))
	if got.Code == http.StatusBadRequest {
		t.Errorf("IDEM-006 نداءٌ بلا مفتاحٍ رُدّ — **يكسر العملاءَ القدامى**: %s", got)
	}
}
