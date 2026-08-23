package qa

// **الطبقةُ الحاديةَ عشرةَ — إعادةُ حساب المسار من موضع الجهاز.**
//
// المعرّفات: `RRT-*` · الوسم: `@api @critical @release`
//
// # ولماذا يُرسَل الموضعُ أصلاً
//
// **خدمةُ الورديّة ترسل كلَّ خمس ثوانٍ وبعد عشرين متراً**
// (`MIN_SEND_GAP_MS` و`MIN_MOVE_M`)، **وحلقةُ الملاحة تقرأ كلَّ
// ثانيةٍ بلا حدّ مسافة.** فسائقٌ بخمسين كم/س **موضعُه عند الخادم
// متأخّرٌ تسعةً وستّين متراً** في أحسن أحواله.
//
// # وما لا يُرسَل
//
// **لا وجهةَ ولا مسافةَ ولا زمنَ ولا حالَ طلب** — الجهازُ يرسل
// واقعةً والخادمُ يقرّر. **وهذه الحزمةُ تُثبت ذلك بالنداء لا
// بالقراءة.**

import (
	"strings"
	"testing"
)

// routePath **مسارُ النقطة** — بموضعٍ أو بلاه.
func routePath(orderID, query string) string {
	p := "/api/v1/driver/orders/" + orderID + "/route"
	if query != "" {
		p += "?" + query
	}
	return p
}

// assignedOrder **طلبٌ بسائقٍ معلوم** — والحالُ تُزرع كما في `RET`.
func (h *Harness) assignedOrder(t *testing.T) (string, *User) {
	t.Helper()
	cust := h.Customer()
	item := h.NewItem(2000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("RRT: تعذّر التجهيز: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	return oid, h.driverOf(oid)
}

// ══════════════════════════════════════════════════════════════════════
// **١ · التخويل**
// ══════════════════════════════════════════════════════════════════════

// TestRRT_001_OnlyOwnDriverGetsRoute **ولا يُعطى المسارَ إلّا صاحبُه.**
//
// **ومسارُ طلبٍ يكشف موضعَ الزبون** — من عرف رقمَ الطلب عرف بابَ من
// يسكن فيه.
func TestRRT_001_OnlyOwnDriverGetsRoute(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)

	mine := h.GET(routePath(oid, ""), drv.Token)
	if mine.Code != 200 {
		t.Fatalf("RRT-001 صاحبُ الطلب لم يُعطَ مسارَه: %s", mine)
	}

	other := h.NewUser("driver")
	foreign := h.GET(routePath(oid, ""), other.Token)
	if foreign.Code < 400 {
		t.Errorf("RRT-001 **سائقٌ آخرُ قرأ مسارَ طلبٍ ليس له**: %s", foreign)
	}
}

// TestRRT_002_UnknownOrder **وطلبٌ لا وجودَ له لا يُفتح.**
func TestRRT_002_UnknownOrder(t *testing.T) {
	h := New(t)
	_, drv := h.assignedOrder(t)
	got := h.GET(routePath("00000000-0000-0000-0000-000000000000", ""), drv.Token)
	if got.Code < 400 {
		t.Errorf("RRT-002 طلبٌ غيرُ موجودٍ ردَّ مساراً: %s", got)
	}
}

// TestRRT_003_UnassignedOrder **وطلبٌ بلا سائقٍ لا يُعطى لأحد.**
func TestRRT_003_UnassignedOrder(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(2000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("RRT-003 تعذّر التجهيز: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)

	drv := h.NewUser("driver")
	got := h.GET(routePath(oid, "lat=35.95&lng=39.01"), drv.Token)
	if got.Code < 400 {
		t.Errorf("RRT-003 **طلبٌ غيرُ مسنَدٍ ردَّ مساراً**: %s", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · التوافقُ الخلفيّ والتحقّق**
// ══════════════════════════════════════════════════════════════════════

// TestRRT_010_NoCoordinatesKeepsOldBehaviour **ونسخةٌ قديمةٌ تعمل كما كانت.**
func TestRRT_010_NoCoordinatesKeepsOldBehaviour(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)
	got := h.GET(routePath(oid, ""), drv.Token)
	if got.Code != 200 {
		t.Fatalf("RRT-010 **نداءٌ بلا موضعٍ كُسر**: %s", got)
	}
	// **والحقولُ القديمةُ باقيةٌ كلُّها** — أيّاً كان حالُ المحرّك.
	body := got.JSON()
	if _, ok := body["available"]; !ok {
		t.Errorf("RRT-010 الحقلُ `available` اختفى: %s", got)
	}
}

// TestRRT_011_ValidCoordinatesAccepted **وموضعٌ صالحٌ يُقبل.**
func TestRRT_011_ValidCoordinatesAccepted(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)
	got := h.GET(routePath(oid, "lat=35.9506&lng=39.0094"), drv.Token)
	if got.Code != 200 {
		t.Fatalf("RRT-011 موضعٌ صالحٌ رُدّ: %s", got)
	}
}

// TestRRT_012_PartialPairRejected **ونصفُ زوجٍ ليس موضعا.**
//
// **وقبولُه بصمتٍ يعني مساراً من مكانٍ لم يقله أحد.**
func TestRRT_012_PartialPairRejected(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)
	for _, q := range []string{"lat=35.9506", "lng=39.0094"} {
		got := h.GET(routePath(oid, q), drv.Token)
		if got.Code != 400 {
			t.Errorf("RRT-012 نصفُ زوجٍ (%s) لم يُردّ: %s", q, got)
		}
	}
}

// TestRRT_013_MalformedCoordinatesRejected **والقيمُ الفاسدةُ تُردّ.**
//
// **و`NaN` و`Inf` تمرّان من `ParseFloat`** — تُكتبان نصّاً فتُقبلان،
// **ثمّ تصيران `null` في JSON أو تُسقطان حساباً في المحرّك.**
func TestRRT_013_MalformedCoordinatesRejected(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)
	bad := []string{
		"lat=NaN&lng=39.0094",
		"lat=35.95&lng=Inf",
		"lat=+Inf&lng=39.0094",
		"lat=abc&lng=39.0094",
		"lat=95&lng=39.0094",
		"lat=35.95&lng=200",
		"lat=-95&lng=39.0094",
		"lat=&lng=",
	}
	for _, q := range bad {
		got := h.GET(routePath(oid, q), drv.Token)
		if got.Code != 400 {
			t.Errorf("RRT-013 **قيمةٌ فاسدةٌ قُبلت** (%s): %s", q, got)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · ولا يحدّد الجهازُ وجهةً**
// ══════════════════════════════════════════════════════════════════════

// TestRRT_020_ClientCannotSetDestination **والوجهةُ من الطلب لا من الرابط.**
//
// **والفحصُ على النقطة لا على الشيفرة**: تُرسَل كلُّ الأسماء التي قد
// يظنّها أحدٌ بابا — **فإن غيّرت شيئاً ظهر في الردّ.**
func TestRRT_020_ClientCannotSetDestination(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)

	plain := h.GET(routePath(oid, "lat=35.9506&lng=39.0094"), drv.Token)
	if plain.Code != 200 {
		t.Fatalf("RRT-020 تعذّر النداءُ الأساس: %s", plain)
	}

	injected := h.GET(routePath(oid,
		"lat=35.9506&lng=39.0094"+
			"&to_lat=33.5138&to_lng=36.2765"+
			"&dest_lat=33.5138&dest_lng=36.2765"+
			"&pickup_lat=33.5&pickup_lng=36.2"+
			"&dropoff_lat=33.5&dropoff_lng=36.2"+
			"&status=delivered&picked=true"+
			"&distance_m=1&duration_s=1"), drv.Token)
	if injected.Code != 200 {
		t.Fatalf("RRT-020 النداءُ بالحقن رُدّ: %s", injected)
	}

	// **والجوابان سواء** — فما أُرسل لم يُقرأ.
	a, b := plain.JSON(), injected.JSON()
	for _, k := range []string{"available", "distance_m", "duration_s"} {
		if !sameJSON(a[k], b[k]) {
			t.Errorf("RRT-020 **الحقلُ %q تبدّل بحقنِ الرابط**: %v ← %v", k, a[k], b[k])
		}
	}
}

// TestRRT_021_ServerChoosesTargetByStatus **والطورُ يقرّر الوجهةَ في الخادم.**
//
// **ويُقرأ من الشيفرة لا يُخمَّن**: الحدُّ الفاصلُ `assigned`
// و`at_pickup` — وما بعدهما إلى الزبون.
func TestRRT_021_ServerChoosesTargetByStatus(t *testing.T) {
	h := New(t)
	src := readSource(t, "../server/driver_route.go")
	if !strings.Contains(src, `picked := status != "assigned" && status != "at_pickup"`) {
		t.Error("RRT-021 **قاعدةُ اختيار الوجهة تبدّلت** — تُراجَع مع الاختبار")
	}
	// **ولا اسمَ حقلٍ من الرابط يمسّ الوجهة.**
	if strings.Contains(src, `q.Get("to_lat")`) || strings.Contains(src, `q.Get("dest`) {
		t.Error("RRT-021 **الخادمُ صار يقرأ وجهةً من الرابط**")
	}
	oid, drv := h.assignedOrder(t)
	if got := h.GET(routePath(oid, "lat=35.9506&lng=39.0094"), drv.Token); got.Code != 200 {
		t.Fatalf("RRT-021 النداءُ رُدّ: %s", got)
	}
}

// sameJSON **مقارنةٌ متساهلةٌ لقيمتين من JSON** — والفراغُ يساوي الفراغ.
func sameJSON(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a == b
}
