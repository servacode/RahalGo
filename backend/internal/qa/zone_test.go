package qa

// **الطبقةُ العاشرة — التغطيةُ الجغرافيّة وبابُها المفتوح.**
//
// المعرّفات: `ZONE-*` · الوسم: `@api @critical @release`
//
// # ولماذا وُجدت هذه الحزمة
//
// **كشفتها القاعدةُ النظيفة** (`QA HYGIENE`, ٢٠٢٦-٠٨-٢٠): يومَ صارت
// الاختباراتُ تبدأ من الصفر **سقط أربعةٌ وعشرون اختباراً دفعةً واحدة**،
// وكلُّها تُنشئ طلباً. **والعلّةُ أنّ «لا منطقةَ» كانت تُكتب نصّاً
// فارغاً في عمودِ `uuid`** — فيردّ بوستغرس `22P02` ويتحوّل إلى ٤٠٤.
//
// **ولم يكن اختبارٌ واحدٌ يفحص إنشاءَ طلبٍ بلا دوائرَ مرسومة.**
//
// # وحالتان لا تُخلطان أبداً
//
//	لا دوائرَ في النظام أصلاً       ←  **بابٌ مفتوح** — الطلبُ يمرّ بلا منطقة
//	دوائرُ موجودةٌ والدبّوسُ خارجَها  ←  **`out_of_zone`** — يُردّ بحقّ
//
// (أمرُ المالك ٢٠٢٦-٠٨-٢٠: «لا تجعل إصلاح ZONE-000 يحوّل النظامَ إلى
//  توصيلٍ مفتوحٍ عندما توجد مناطقُ تغطيةٍ فعليّة».)
//
// **وهو شرطُ التوسّع إلى سوريا**: مدينةٌ تُفعَّل ولم تُرسم دوائرُها بعد
// **تعمل**، ثمّ تُرسم **فيصير التحقّقُ الجغرافيُّ نافذاً.**

import (
	"context"
	"testing"
)

// noZones **يُطفئ كلَّ الدوائر لهذا الاختبار وحدَه** — ثمّ يعيدها.
//
// **ولا تُحذف**: العُدّةُ ترسم دائرةً مرّةً في العمليّة (`seedZone`)،
// **وحذفُها يُسقط ما بعدها.** والإطفاءُ يُرجَع بعينه.
func (h *Harness) noZones(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	rows, err := h.Pool.Query(ctx, `SELECT id::text FROM delivery_zones WHERE active`)
	if err != nil {
		t.Fatalf("ZONE: تعذّرت قراءةُ الدوائر: %v", err)
	}
	var saved []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatalf("ZONE: تعذّرت قراءةُ دائرة: %v", err)
		}
		saved = append(saved, id)
	}
	rows.Close()
	if _, err := h.Pool.Exec(ctx, `UPDATE delivery_zones SET active = false`); err != nil {
		t.Fatalf("ZONE: تعذّر إطفاءُ الدوائر: %v", err)
	}
	// **والإرجاعُ بسياقٍ حيٍّ لا بسياق الاختبار** — سياقُ الاختبار
	// يُلغى **قبل** أن يعمل التنظيف، فتبقى الدوائرُ مطفأةً لمن بعده.
	// (وقعت هذه بعينها في `CITY-001`.)
	t.Cleanup(func() {
		if len(saved) > 0 {
			_, _ = h.Pool.Exec(context.Background(),
				`UPDATE delivery_zones SET active = true WHERE id::text = ANY($1)`, saved)
		}
	})
}

// zoneCount **كم دائرةً فعّالة** — يُقرأ في التوكيد لا يُفترض.
func (h *Harness) zoneCount(t *testing.T) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM delivery_zones WHERE active`).Scan(&n); err != nil {
		t.Fatalf("ZONE: تعذّر عدُّ الدوائر: %v", err)
	}
	return n
}

// zoneOf **منطقةُ الطلب كما في القاعدة** — وفارغةٌ تعني `NULL` حقّاً.
func (h *Harness) zoneOf(t *testing.T, orderID string) *string {
	t.Helper()
	var z *string
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT zone_id::text FROM orders WHERE id = $1::uuid`, orderID).Scan(&z); err != nil {
		t.Fatalf("ZONE: تعذّرت قراءةُ منطقة الطلب: %v", err)
	}
	return z
}

// ══════════════════════════════════════════════════════════════════════
// **ZONE-001 — لا دوائرَ، والطلبُ يمرّ**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذا هو العطبُ بعينه**: كان يردّ ٤٠٤ «غير موجود» — **ورسالةٌ تقول
// غيرَ ما وقع أسوأُ من رسالةٍ لا تقول شيئاً**، لأنّ من يقرؤها يبحث في
// المكان الخطأ.
func TestZONE_001_NoZonesOrderPasses(t *testing.T) {
	h := New(t)
	h.noZones(t)
	if n := h.zoneCount(t); n != 0 {
		t.Fatalf("ZONE-001 المقدّمةُ لم تتحقّق — %d دائرةً فعّالة", n)
	}

	cust := h.Customer()
	item := h.NewItem(2000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code != 201 {
		t.Fatalf("ZONE-001 **بلا دوائرَ لم يُنشأ الطلب**: %s", made)
	}

	oid, _ := made.JSON()["id"].(string)
	if z := h.zoneOf(t, oid); z != nil {
		t.Errorf("ZONE-001 المنطقةُ %q — والمنتظَرُ فراغٌ حقيقيّ", *z)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ZONE-002 — والدورةُ كلُّها لا تنكسر بعده**
// ══════════════════════════════════════════════════════════════════════
//
// **وطلبٌ يُنشأ ثمّ ينهار عند أوّل انتقالٍ ليس طلباً** — فالفحصُ يمشي
// به إلى التسليم، **ويقرأه من نقطة الزبون** حيث يُقرأ `zone_name`.
func TestZONE_002_NoZoneSurvivesLifecycle(t *testing.T) {
	h := New(t)
	h.noZones(t)

	cust := h.Customer()
	item := h.NewItem(1500)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code != 201 {
		t.Fatalf("ZONE-002 تعذّر الإنشاء: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)

	drv := h.driverOf(oid)
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to})
		if got.Code >= 400 {
			t.Fatalf("ZONE-002 **الانتقالُ إلى %s انكسر بلا منطقة**: %s", to, got)
		}
	}
	// **والتسليمُ يحتاج إثباتاً** — قاعدةُ عملٍ لا علاقةَ لها بالمنطقة
	// (انظر `LIFE-001`)، **فتُخطّى بسببٍ مكتوبٍ كما تفعل دورةُ الحياة.**
	if skip := h.POST("/api/v1/driver/orders/"+oid+"/proof/skip", drv.Token,
		map[string]any{"reason": "اختبارٌ آليّ"}); skip.Code >= 400 {
		t.Fatalf("ZONE-002 تعذّر تخطّي الإثبات: %s", skip)
	}
	if done := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"}); done.Code >= 400 {
		t.Fatalf("ZONE-002 **التسليمُ انكسر بلا منطقة**: %s", done)
	}
	if now := h.statusOf(oid); now != "delivered" {
		t.Fatalf("ZONE-002 الحالُ %q — والدورةُ لم تكتمل", now)
	}

	// **والقراءةُ من نقطة الزبون** — `LEFT JOIN` على منطقةٍ فارغة.
	seen := h.GET("/api/v1/my/orders/"+oid, cust.Token)
	if seen.Code != 200 {
		t.Fatalf("ZONE-002 **الطلبُ لا يُقرأ بلا منطقة**: %s", seen)
	}
	if z := seen.JSON()["zone_id"]; z != nil {
		t.Errorf("ZONE-002 `zone_id` = %v — والمنتظَرُ null", z)
	}
	if z := seen.JSON()["zone_name"]; z != nil {
		t.Errorf("ZONE-002 `zone_name` = %v — والمنتظَرُ null", z)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ZONE-003 — ودائرةٌ تغطّيه: كما كان**
// ══════════════════════════════════════════════════════════════════════
func TestZONE_003_InsideZoneUnchanged(t *testing.T) {
	h := New(t)
	if n := h.zoneCount(t); n == 0 {
		t.Fatalf("ZONE-003 المقدّمةُ لم تتحقّق — لا دائرةَ فعّالة")
	}

	cust := h.Customer()
	item := h.NewItem(2000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code != 201 {
		t.Fatalf("ZONE-003 تعذّر الإنشاء داخلَ الدائرة: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	z := h.zoneOf(t, oid)
	if z == nil {
		t.Fatalf("ZONE-003 **الطلبُ بلا منطقةٍ والدبّوسُ داخلَ دائرة**")
	}
	if made.JSON()["zone_name"] == nil {
		t.Errorf("ZONE-003 اسمُ المنطقة فارغٌ في الردّ — والمنطقةُ %q", *z)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ZONE-004 — دوائرُ موجودةٌ والدبّوسُ خارجَها: يُردّ**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذا الشقُّ هو ما لا يجوز أن يفتحه إصلاحُ `ZONE-000`.**
//
// **والدبّوسُ في دمشق** — أربعُ مئةِ كيلومترٍ عن الرقّة، **وخارجَ أيّ
// دائرةِ عُدّة.**
func TestZONE_004_OutsideAllZonesRejected(t *testing.T) {
	h := New(t)
	if n := h.zoneCount(t); n == 0 {
		t.Fatalf("ZONE-004 المقدّمةُ لم تتحقّق — لا دائرةَ فعّالة")
	}

	cust := h.Customer()
	item := h.NewItem(2000)
	body := orderBody(item, 1)
	body["lat"] = 33.5138 // دمشق
	body["lng"] = 36.2765

	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), body)
	if made.Code != 400 {
		t.Fatalf("ZONE-004 **دبّوسٌ خارجَ التغطية لم يُردّ**: %s", made)
	}
	code := made.Err()
	if code != "out_of_zone" {
		t.Errorf("ZONE-004 الرمزُ %q — والمنتظَرُ out_of_zone: %s", code, made)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ZONE-005 — ولا نصَّ فارغاً يبلغ بوستغرس**
// ══════════════════════════════════════════════════════════════════════
//
// (أمرُ المالك: «لا أريد `22P02` أن يتحوّل إلى ٤٠٤».)
//
// **والفحصُ على الأثر لا على النيّة**: `22P02` يصل الزبونَ ٤٠٤ عبر
// `respondErr` — **فردٌّ ٤٠٤ من هذا الباب هو التوقيعُ الوحيدُ الذي
// يُرى من الخارج.** ولا صفَّ بنصٍّ فارغٍ يبقى في القاعدة.
func TestZONE_005_NoEmptyUUIDReachesPostgres(t *testing.T) {
	h := New(t)
	h.noZones(t)

	cust := h.Customer()
	item := h.NewItem(1200)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code == 404 {
		t.Fatalf("ZONE-005 **عاد ٤٠٤ — و`22P02` يتسلّل من جديد**: %s", made)
	}
	if made.Code != 201 {
		t.Fatalf("ZONE-005 الردُّ %d — والمنتظَرُ ٢٠١: %s", made.Code, made)
	}

	// **ولا معرّفَ نصّيٌّ فارغٌ في الجدول** — والعمودُ `uuid` لا يحتمله
	// أصلاً، **فوجودُ صفٍّ كهذا يعني أنّ النوعَ تبدّل تحتنا.**
	var bad int
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM orders WHERE zone_id::text = ''`).Scan(&bad); err != nil {
		t.Fatalf("ZONE-005 تعذّر العدّ: %v", err)
	}
	if bad != 0 {
		t.Errorf("ZONE-005 %d طلباً بمعرّفِ منطقةٍ نصُّه فارغ", bad)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ZONE-006 — والتوافقُ الخلفيّ: طلبٌ بمنطقةٍ حقيقيّةٍ كما كان**
// ══════════════════════════════════════════════════════════════════════
//
// **ويُقرأ الاثنان جنباً إلى جنب**: واحدٌ بمنطقةٍ وآخرُ بلا منطقة،
// **فالفارقُ يُرى في الردّ لا في الظنّ.**
func TestZONE_006_BackwardCompatible(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(2000)

	// **أوّلاً بمنطقةٍ حقيقيّة.**
	withZone := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if withZone.Code != 201 {
		t.Fatalf("ZONE-006 تعذّر إنشاءُ الطلب بمنطقة: %s", withZone)
	}
	oldID, _ := withZone.JSON()["id"].(string)
	if h.zoneOf(t, oldID) == nil {
		t.Fatalf("ZONE-006 **الطلبُ الأوّلُ بلا منطقةٍ والدوائرُ فعّالة**")
	}

	// **ثمّ تُطفأ الدوائرُ ويُنشأ ثانٍ** — والأوّلُ يبقى كما كُتب.
	h.noZones(t)
	openOne := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if openOne.Code != 201 {
		t.Fatalf("ZONE-006 تعذّر الإنشاءُ بعد الإطفاء: %s", openOne)
	}

	if z := h.zoneOf(t, oldID); z == nil {
		t.Errorf("ZONE-006 **الطلبُ القديمُ فقد منطقتَه**")
	}
	seen := h.GET("/api/v1/my/orders/"+oldID, cust.Token)
	if seen.Code != 200 {
		t.Fatalf("ZONE-006 الطلبُ القديمُ لا يُقرأ: %s", seen)
	}
	if seen.JSON()["zone_name"] == nil {
		t.Errorf("ZONE-006 **اسمُ منطقة الطلب القديم اختفى** — والتوافقُ انكسر")
	}
}
