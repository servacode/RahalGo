package qa

// ══════════════════════════════════════════════════════════════════════
// **أوقاتُ مناطق التوصيل — من باب الشبكة** (`ZH`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// **وآلةُ الجدول مقيسةٌ بالحساب** في `internal/platform` — الحدُّ وعبورُ
// منتصف الليل وحدُّ الأسبوع والتقاطعُ مع المنصّة. **وهذه تقيس ما لا
// يُقاس هناك**:
//
//	**أنّ الجغرافيا تسبق الوقتَ ولا يحجبها** ·
//	**وأنّ المنطقةَ التي سُعِّرت هي التي يُسأل عن وقتها** ·
//	**وأنّ المردودَ لا يُخلّف أثراً** ·
//	**وأنّ الطلبَ القائمَ يمضي** ·
//	**وأنّ منطقةً مُطفأةً لا تصير صالحةً بجدول.**

import (
	"net/http"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/platform"
)

// ═════════════════ تجهيزٌ ═════════════════

// zone منطقةُ توصيلٍ دائريّةٌ حول نقطةٍ بعينها.
type zoneFx struct {
	ID       string
	Lat, Lng float64
}

// newZone **ينشئ منطقةً فعّالةً حول نقطةٍ** — نصفُ قطرها كيلومتران.
func newZone(t *testing.T, h *Harness, name string, lat, lng float64) zoneFx {
	t.Helper()
	var id string
	err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO delivery_zones (name, center, radius_m, shape, delivery_fee, min_order, active)
		VALUES ($1, ST_SetSRID(ST_MakePoint($3,$2),4326)::geography, 2000, 'radius', 500, 0, true)
		RETURNING id::text`, name, lat, lng).Scan(&id)
	if err != nil {
		t.Fatalf("إنشاءُ منطقة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM delivery_zones WHERE id = $1::uuid`, id)
	})
	return zoneFx{ID: id, Lat: lat, Lng: lng}
}

// zoneHours **يكتب جدولَ منطقةٍ ويضبط سريانَه.**
func zoneHours(t *testing.T, h *Harness, id string, enforced bool, ws ...platform.Window) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(),
		`DELETE FROM delivery_zone_hours WHERE zone_id = $1::uuid`, id); err != nil {
		t.Fatalf("مسحُ جدول المنطقة: %v", err)
	}
	for _, w := range ws {
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO delivery_zone_hours (zone_id, day_of_week, starts_at, ends_at)
			VALUES ($1::uuid, $2, make_time($3, $4, 0), make_time($5, $6, 0))`,
			id, w.Day, int(w.Start)/60, int(w.Start)%60,
			int(w.End)/60, int(w.End)%60); err != nil {
			t.Fatalf("كتابةُ فترةِ منطقة: %v", err)
		}
	}
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE delivery_zones SET hours_enforced = $2 WHERE id = $1::uuid`,
		id, enforced); err != nil {
		t.Fatalf("ضبطُ سريان المنطقة: %v", err)
	}

	// ══════════════════════════════════════════════════════════════════
	// **وما سرى يُرفَع** — **وإلّا أسقط الفحصُ جيرانَه** (٢٠٢٦-٠٩-١٤)
	// ══════════════════════════════════════════════════════════════════
	//
	// **ومنطقةٌ سارٍ جدولُها ومغلقةٌ الآن تبقى بعد الفحص** — **فيأتي
	// جارٌ ينشئ طلباً فيُردّ بـ`zone_closed_now`** وهو لا يعرف لماذا.
	//
	// **وقِيس**: سقطت `ZONE-002` و`ZONE-003` و`ZONE-006` في جولةٍ
	// كاملةٍ بـ«zone_closed_now» — **وهي خضراءُ منفردةً.**
	//
	// **ولا يكفي أن تُمحى المنطقة** — **فمنطقةٌ يُعيدها `otherZonesOff`
	// إلى الحياة تعود بسريانها معها.**
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(),
			`UPDATE delivery_zones SET hours_enforced = false WHERE id = $1::uuid`, id)
		_, _ = h.Pool.Exec(ctxBG(),
			`DELETE FROM delivery_zone_hours WHERE zone_id = $1::uuid`, id)
	})
}

// otherZonesOff **يُطفئ كلَّ منطقةٍ سواها — ثمّ يُعيدها.**
//
// **فالقاعدةُ مشتركةٌ بين الفحوص**، **ومنطقةٌ خلّفها فحصٌ سابقٌ قد تفوز
// بالقرب فيُقاس جدولُ غيرِ التي نقصد.**
//
// ══════════════════════════════════════════════════════════════════════
// **وما أُطفئ يُعاد — وإلّا أسقط الفحصُ جيرانَه** (٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// **وأوّلُ صياغةٍ أطفأت ولم تُعِدْ** — **فسقطت `ZONE-003` و`ZONE-004`
// و`ZONE-006` في الجولة الكاملة بـ«لا دائرةَ فعّالة»**، **وهي خضراءُ
// منفردةً.** **وسقوطٌ في جارٍ يُطارَد في غير موضعه.**
//
// **والإعادةُ في `Cleanup` تقع ولو سقط الفحص** — **وخاتمةٌ تُكتب في
// آخر الدالّة لا تُنفَّذ إن سقط قبلها.**
func otherZonesOff(t *testing.T, h *Harness, keep string) {
	t.Helper()
	rows, err := h.Pool.Query(ctxBG(),
		`SELECT id::text FROM delivery_zones WHERE active AND id <> $1::uuid`, keep)
	if err != nil {
		t.Fatalf("قراءةُ المناطق الفعّالة: %v", err)
	}
	was := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatalf("قراءةُ منطقة: %v", err)
		}
		was = append(was, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("قراءةُ المناطق الفعّالة: %v", err)
	}

	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE delivery_zones SET active = false WHERE id = ANY($1::uuid[])`, was); err != nil {
		t.Fatalf("إطفاءُ المناطق: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(),
			`UPDATE delivery_zones SET active = true WHERE id = ANY($1::uuid[])`, was)
	})
}

// zoneBody طلبٌ عاديٌّ إلى نقطةٍ بعينها.
func zoneBody(it *Item, lat, lng float64) map[string]any {
	return map[string]any{
		"items":          []map[string]any{{"menu_item_id": it.ID, "qty": 1}},
		"address_text":   "الرقة — شارع الاختبار",
		"lat":            lat,
		"lng":            lng,
		"payment_method": "cash",
	}
}

// zoneCustomBody طلبٌ مخصَّصٌ إلى نقطةٍ بعينها.
func zoneCustomBody(lat, lng float64) map[string]any {
	return map[string]any{
		"request":        "كيلو بندورة",
		"address_text":   "الرقة — شارع الاختبار",
		"lat":            lat,
		"lng":            lng,
		"payment_method": "cash",
	}
}

// isZoneClosed **أهذا ردُّ «المنطقةُ خارجَ وقتها»؟**
func isZoneClosed(r Res) bool {
	return r.Code == http.StatusServiceUnavailable && r.Err() == "zone_closed_now"
}

// zhOpen فترةٌ تحوي هذه اللحظة · zhShut فترةٌ بعد ساعتين.
func zhOpen() platform.Window { return around(-time.Hour, time.Hour) }
func zhShut() platform.Window { return around(2*time.Hour, 3*time.Hour) }

// ═════════════════ ZH-01 — الافتراضُ لا يُغلق شيئاً ═════════════════

// TestZH01_DisabledScheduleKeepsExistingBehaviour **منطقةٌ بلا سريانٍ
// تعمل كما كانت** — **وهو حالُ كلّ منطقةٍ قائمةٍ بعد الهجرة.**
func TestZH01_DisabledScheduleKeepsExistingBehaviour(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ ZH-01", 35.9506, 39.0094)
	otherZonesOff(t, hh, z.ID)
	// **وجدولٌ مكتوبٌ ومغلقٌ الآن — والرايةُ مطفأة.**
	zoneHours(t, hh, z.ID, false, zhShut())

	u := hh.Customer()
	it := hh.NewItem(900)
	r := hh.POST("/api/v1/orders", u.Token, zoneBody(it, z.Lat, z.Lng))
	if r.Code >= 400 {
		t.Fatalf("**منطقةٌ بلا سريانٍ ردّت طلباً**: %d / %s — "+
			"**فنشرُ الهجرة يُغلق مناطقَ قائمة**", r.Code, r.Err())
	}
}

// ═════════════════ ZH-18 · ZH-19 — المنعُ حين يسري ═════════════════

// TestZH18_ZoneClosedBlocksNormalOrder **وخارجَ وقتِ المنطقة يُردّ
// الطلبُ برمزٍ يخصّه.**
func TestZH18_ZoneClosedBlocksNormalOrder(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ ZH-18", 35.9506, 39.0094)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, true, zhShut())

	u := hh.Customer()
	it := hh.NewItem(900)
	r := hh.POST("/api/v1/orders", u.Token, zoneBody(it, z.Lat, z.Lng))
	if !isZoneClosed(r) {
		t.Fatalf("**طلبٌ مرّ والمنطقةُ خارجَ وقتها**: %d / %s", r.Code, r.Err())
	}
	// **وموعدُ العودة يصل العميل.**
	if e, _ := r.JSON()["error"].(map[string]any); e != nil {
		d, _ := e["details"].(map[string]any)
		if d == nil || d["next_available_at"] == nil {
			t.Fatalf("**أُغلقت بلا موعدِ عودةٍ وهو معلوم**: %v", d)
		}
	}
}

// TestZH19_ZoneClosedBlocksCustomOrder **والمخصَّصُ كذلك.**
func TestZH19_ZoneClosedBlocksCustomOrder(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ ZH-19", 35.9506, 39.0094)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, true, zhShut())

	u := hh.Customer()
	r := hh.POST("/api/v1/orders/custom", u.Token, zoneCustomBody(z.Lat, z.Lng))
	if !isZoneClosed(r) {
		t.Fatalf("**طلبٌ مخصَّصٌ مرّ والمنطقةُ خارجَ وقتها**: %d / %s", r.Code, r.Err())
	}
}

// **وداخلَ وقتها يمضي** — **وحارسٌ يمنع دائماً ليس حارساً.**
func TestZH_OpenZoneAdmitsOrder(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ ZH-فتح", 35.9506, 39.0094)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, true, zhOpen())

	u := hh.Customer()
	it := hh.NewItem(900)
	if r := hh.POST("/api/v1/orders", u.Token, zoneBody(it, z.Lat, z.Lng)); r.Code >= 400 {
		t.Fatalf("**طلبٌ رُدّ داخلَ وقت المنطقة**: %d / %s", r.Code, r.Err())
	}
}

// ═════════════════ ZH-15 · ZH-16 · ZH-17 — الجغرافيا تسبق ═════════════════

// TestZH15_OutOfZoneStaysOutOfZone **ونقطةٌ خارجَ الأشكال تبقى
// `out_of_zone`** — **ولا يحجبها وقتُ منطقةٍ ليست فيها.**
func TestZH15_OutOfZoneStaysOutOfZone(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ ZH-15", 35.9506, 39.0094)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, true, zhShut())

	u := hh.Customer()
	it := hh.NewItem(900)
	// **دمشقُ — بعيدةٌ عن كلّ منطقةٍ فعّالة.**
	r := hh.POST("/api/v1/orders", u.Token, zoneBody(it, 33.5138, 36.2765))
	if r.Err() != "out_of_zone" {
		t.Fatalf("**نقطةٌ خارجَ التغطية رُدّت بغير `out_of_zone`**: %d / %s",
			r.Code, r.Err())
	}
	// ══════════════════════════════════════════════════════════════════
	// **والحالُ عقدٌ كالرمز — ٤٠٠ لا سواها** (٢٠٢٦-٠٩-١٤)
	// ══════════════════════════════════════════════════════════════════
	//
	// **و`out_of_zone` حكمٌ على العنوان المُرسَل**: **رفضٌ مفهومٌ لا
	// عطبُ خادمٍ ولا «ليس الآن».** **وعميلٌ يفرّق بالحال قبل أن يقرأ
	// الرمزَ يبني تصرّفَه عليها** — **فتبديلُها كسرُ عقدٍ منشور.**
	//
	// **ويحرسها `XG-46` كذلك** — **وهذا يضعها بجانب أختها الزمنيّة
	// فيُقرأ الفرقُ في موضعٍ واحد.**
	if r.Code != http.StatusBadRequest {
		t.Fatalf("**حالُ `out_of_zone` تبدّلت**: %d — **والعقدُ ٤٠٠**", r.Code)
	}

	// **وأختُها الزمنيّةُ تفترق رمزاً وحالاً** — ٥٠٣: **«ليس الآن» لا
	// «طلبُك خطأ».**
	zoneHours(t, hh, z.ID, true, zhShut())
	inside := hh.POST("/api/v1/orders", u.Token, zoneBody(it, z.Lat, z.Lng))
	if inside.Err() != "zone_closed_now" {
		t.Fatalf("**عنوانٌ داخلَ منطقةٍ خارجَ وقتها رُدّ بغير `zone_closed_now`**: %d / %s",
			inside.Code, inside.Err())
	}
	if inside.Code != http.StatusServiceUnavailable {
		t.Fatalf("**حالُ `zone_closed_now` ليست ٥٠٣**: %d", inside.Code)
	}
	if inside.Code == r.Code {
		t.Fatal("**الحالان تساوتا** — **والعميلُ يفرّق بالحال قبل الرمز**")
	}
}

// TestZH17_InactiveZoneNeverBecomesUsable **ومنطقةٌ مُطفأةٌ لا تصير
// صالحةً بجدولٍ مفتوح.**
//
// **والجدولُ لا يُصلح تغطيةً** — **وصفٌّ مُطفأٌ ليس تغطية.**
func TestZH17_InactiveZoneNeverBecomesUsable(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ ZH-17", 35.9506, 39.0094)
	otherZonesOff(t, hh, z.ID)
	// **جدولٌ مفتوحٌ الآن — ثمّ تُطفأ المنطقة.**
	zoneHours(t, hh, z.ID, true, zhOpen())
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE delivery_zones SET active = false WHERE id = $1::uuid`, z.ID); err != nil {
		t.Fatalf("إطفاءُ المنطقة: %v", err)
	}

	u := hh.Customer()
	it := hh.NewItem(900)
	r := hh.POST("/api/v1/orders", u.Token, zoneBody(it, z.Lat, z.Lng))
	if r.Code < 400 {
		t.Fatalf("**منطقةٌ مُطفأةٌ قبلت طلباً لأنّ جدولَها مفتوح**: %d", r.Code)
	}
	if r.Err() == "zone_closed_now" {
		t.Fatalf("**رُدّت بوقتِ منطقةٍ مُطفأةٍ** — **والجغرافيا تسبق**: %s", r.Err())
	}
	// **ولا تغطيةَ صالحةً ⇒ `coverage_unavailable`** — عقدُ الإخفاق المغلق.
	if r.Err() != "coverage_unavailable" && r.Err() != "out_of_zone" {
		t.Fatalf("**رمزٌ غيرُ متوقَّع**: %d / %s", r.Code, r.Err())
	}
}

// ═════════════════ ZH-23 · ZH-24 · ZH-25 — الأسبقيّة ═════════════════

// TestZH23_ZH24_ZH25_Precedence **ولا يحجب وقتُ المنطقة سبباً أسبقَ منه.**
func TestZH23_ZH24_ZH25_Precedence(t *testing.T) {
	hh := New(t)
	z := newZone(t, hh, "منطقةُ الأسبقيّة", 35.9506, 39.0094)
	otherZonesOff(t, hh, z.ID)

	u := hh.Customer()
	it := hh.NewItem(900)
	body := zoneBody(it, z.Lat, z.Lng)

	// ZH-24 · **وضعُ الإطلاق يغلب الكلَّ.**
	ordersOpen(t, hh)
	zoneHours(t, hh, z.ID, true, zhShut())
	hh.Setting("launch.customer_orders", "false")
	if got := hh.POST("/api/v1/orders", u.Token, body); got.Err() != "launch_closed" {
		t.Fatalf("**وقتُ المنطقة حجب وضعَ الإطلاق**: %s", got.Err())
	}

	// ZH-25 · **ثمّ الإيقافُ المؤقّت.**
	ordersOpen(t, hh)
	zoneHours(t, hh, z.ID, true, zhShut())
	closure(t, hh, true, "صيانة", nil)
	if got := hh.POST("/api/v1/orders", u.Token, body); got.Err() != "temporarily_unavailable" {
		t.Fatalf("**وقتُ المنطقة حجب الإيقافَ المؤقّت**: %s", got.Err())
	}

	// ZH-23 · **ثمّ دوامُ المنصّة.**
	ordersOpen(t, hh)
	zoneHours(t, hh, z.ID, true, zhShut())
	closedNow(t, hh)
	if got := hh.POST("/api/v1/orders", u.Token, body); got.Err() != "platform_closed_now" {
		t.Fatalf("**وقتُ المنطقة حجب دوامَ المنصّة**: %s", got.Err())
	}

	// **وحين تُفتح الثلاثُ فوقَه يظهر هو.**
	ordersOpen(t, hh)
	zoneHours(t, hh, z.ID, true, zhShut())
	if got := hh.POST("/api/v1/orders", u.Token, body); !isZoneClosed(got) {
		t.Fatalf("**وقتُ المنطقة لم يظهر وقد خلا له الطريق**: %d / %s",
			got.Code, got.Err())
	}
}

// ═════════════════ ZH-20 · ZH-21 — المردودُ بلا أثر ═════════════════

// TestZH20_RejectedOrderLeavesNothing **ولا صفَّ ولا قيدَ ولا إشعار —
// ولا يُحرَق مفتاحُ التفرّد.**
func TestZH20_RejectedOrderLeavesNothing(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ ZH-20", 35.9506, 39.0094)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, true, zhShut())

	u := hh.Customer()
	it := hh.NewItem(900)
	before := hh.CountOrders(u.ID)
	ledger := phCount(t, hh, `SELECT count(*) FROM wallet_transactions`)
	notes := phCount(t, hh, `SELECT count(*) FROM notifications`)

	key := uniq("zh20")
	if r := hh.POSTKey("/api/v1/orders", u.Token, key, zoneBody(it, z.Lat, z.Lng)); !isZoneClosed(r) {
		t.Fatalf("**لم يُردّ الطلب**: %d / %s", r.Code, r.Err())
	}

	if got := hh.CountOrders(u.ID); got != before {
		t.Fatalf("**صفُّ طلبٍ وُلد من نداءٍ مردود**: %d ← %d", before, got)
	}
	if got := phCount(t, hh, `SELECT count(*) FROM wallet_transactions`); got != ledger {
		t.Fatalf("**قيدٌ ماليٌّ وُلد من نداءٍ مردود**: %d ← %d", ledger, got)
	}
	if got := phCount(t, hh, `SELECT count(*) FROM notifications`); got != notes {
		t.Fatalf("**إشعارٌ وُلد من نداءٍ مردود**: %d ← %d", notes, got)
	}

	// ZH-21 · **والمفتاحُ يمضي حين يعود الوقت.**
	zoneHours(t, hh, z.ID, true, zhOpen())
	if again := hh.POSTKey("/api/v1/orders", u.Token, key, zoneBody(it, z.Lat, z.Lng)); again.Code >= 400 {
		t.Fatalf("**المفتاحُ أُحرق على منطقةٍ خارجَ وقتها**: %d / %s",
			again.Code, again.Err())
	}
}

// ═════════════════ ZH-22 — الطلبُ القائمُ يمضي ═════════════════

// TestZH22_ExistingOrderSurvivesZoneClosing **طلبٌ صحّ قبل إغلاق
// المنطقة يبقى صحيحاً بعده.**
func TestZH22_ExistingOrderSurvivesZoneClosing(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ ZH-22", 35.9506, 39.0094)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, true, zhOpen())

	fx := populatedOrder(t, hh)
	was := phStatus(t, hh, fx.OrderID)

	// **ثمّ يُغلق وقتُ المنطقة — من باب اللوحة كما يفعل المالك**،
	// **لا بكتابةٍ في القاعدة**: **فالمسارُ الحقيقيُّ هو الذي يُقاس.**
	_, tok := capUser(t, hh, "admin")
	shut := zhShut()
	set := hh.Call("PUT", "/api/v1/admin/zones/"+z.ID+"/hours", tok, map[string]any{
		"enforced": true,
		"windows": []any{map[string]any{
			"day_of_week": shut.Day,
			"start":       shut.Start.String(),
			"end":         shut.End.String(),
		}},
	}, nil)
	if set.Code != http.StatusOK {
		t.Fatalf("ضبطُ جدول المنطقة من اللوحة: %d / %s", set.Code, set.Err())
	}

	if got := phStatus(t, hh, fx.OrderID); got != was {
		t.Fatalf("**حالُ طلبٍ قائمٍ تبدّلت بإغلاق المنطقة**: %q ← %q", was, got)
	}
	if r := hh.GET("/api/v1/my/orders/"+fx.OrderID, fx.Cust.Token); r.Code != http.StatusOK {
		t.Fatalf("**الزبونُ لا يرى طلبَه بعد إغلاق المنطقة**: %d / %s", r.Code, r.Err())
	}
	if r := hh.GET("/api/v1/merchant/orders/"+fx.OrderID, fx.MerchToken); r.Code != http.StatusOK {
		t.Fatalf("**المتجرُ حُجب عن طلبٍ قائم**: %d / %s", r.Code, r.Err())
	}
}

// ═════════════════ ZH-14 · ZH-34 — العنوانُ والمنطقةُ الحاكمة ═════════════════

// TestZH14_ZH34_AddressPicksItsOwnZone **وتبديلُ العنوان يبدّل الجواب —
// وكلُّ منطقةٍ بجدولها.**
//
// **وهو `ZH-34` أيضاً**: **المنطقةُ التي سُعِّرت هي التي يُسأل عن
// وقتها** — **ولو سُئل «أيُّ منطقةٍ تحوي النقطة» لَجاز أن يقع على
// غيرها.**
func TestZH14_ZH34_AddressPicksItsOwnZone(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	// **منطقتان متباعدتان** — نهاريّةٌ مفتوحةٌ الآن وليليّةٌ مغلقة.
	open := newZone(t, hh, "منطقةٌ مفتوحة", 35.9506, 39.0094)
	shut := newZone(t, hh, "منطقةٌ مغلقة", 36.2000, 37.1500)
	otherZonesOff(t, hh, open.ID)
	// **والثانيةُ تُعاد إلى الحياة بعد أن أطفأتها الأولى.**
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE delivery_zones SET active = true WHERE id = $1::uuid`, shut.ID); err != nil {
		t.Fatalf("إعادةُ المنطقة الثانية: %v", err)
	}
	zoneHours(t, hh, open.ID, true, zhOpen())
	zoneHours(t, hh, shut.ID, true, zhShut())

	u := hh.Customer()
	it := hh.NewItem(900)

	// **العنوانُ الأوّلُ يمضي.**
	if r := hh.POST("/api/v1/orders", u.Token, zoneBody(it, open.Lat, open.Lng)); r.Code >= 400 {
		t.Fatalf("**العنوانُ في المنطقة المفتوحة رُدّ**: %d / %s", r.Code, r.Err())
	}
	// **وتبديلُ العنوان وحدَه يقلب الجواب.**
	if r := hh.POST("/api/v1/orders", u.Token, zoneBody(it, shut.Lat, shut.Lng)); !isZoneClosed(r) {
		t.Fatalf("**العنوانُ في المنطقة المغلقة مرّ**: %d / %s", r.Code, r.Err())
	}
	// **والعودةُ إلى الأوّل تُرجع القبول.**
	if r := hh.POST("/api/v1/orders", u.Token, zoneBody(it, open.Lat, open.Lng)); r.Code >= 400 {
		t.Fatalf("**العودةُ إلى المنطقة المفتوحة رُدّت**: %d / %s", r.Code, r.Err())
	}
}

// ═════════════════ ZH-18ب — التسعيرةُ تُخبِر قبل الضغط ═════════════════

// TestZH_QuoteSaysZoneClosed **والسلّةُ تعرف قبل الإتمام.**
//
// **وحقلٌ يُضاف لا عقدٌ يُكسَر** — **والحقولُ القائمةُ تبقى** (`ZH-31`).
func TestZH_QuoteSaysZoneClosed(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ التسعيرة", 35.9506, 39.0094)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, true, zhShut())

	u := hh.Customer()
	it := hh.NewItem(900)
	r := hh.POST("/api/v1/public/quote", u.Token, map[string]any{
		"items": []map[string]any{{"menu_item_id": it.ID, "qty": 1}},
		"lat":   z.Lat, "lng": z.Lng,
	})
	if r.Code != http.StatusOK {
		t.Fatalf("**التسعيرةُ سقطت والعنوانُ مُغطّى**: %d / %s — "+
			"**وسلّةٌ تنهار لأنّ الوقتَ انتهى سلّةٌ لا تُستعمل**", r.Code, r.Err())
	}
	j := r.JSON()
	if j["zone_closed"] != true {
		t.Fatalf("**التسعيرةُ لا تقول إنّ المنطقةَ خارجَ وقتها**: %v", j["zone_closed"])
	}
	if j["serviceable_reason"] != "zone_closed_now" {
		t.Fatalf("**سببٌ غيرُ الذي يردّ به الإنشاء**: %v", j["serviceable_reason"])
	}
	if j["out_of_zone"] == true {
		t.Fatal("**قيل «خارجَ التغطية» لعنوانٍ داخلَها** — وهو كذبٌ")
	}
	if _, ok := j["next_available_at"].(string); !ok {
		t.Fatalf("**موعدُ العودة غائبٌ وهو معلوم**: %v", j["next_available_at"])
	}
	// **والعقدُ القديمُ لم يُكسَر** (`ZH-31`).
	for _, k := range []string{"subtotal", "delivery_fee", "total", "serviceable", "out_of_zone"} {
		if _, ok := j[k]; !ok {
			t.Errorf("**حقلٌ قائمٌ اختفى من التسعيرة**: %s", k)
		}
	}
}

// ═════════════════ ZH-30 — التصفّحُ يبقى ═════════════════

// TestZH30_BrowsingStaysOpen **وإغلاقُ منطقةٍ ليس إطفاءَ محرّك.**
func TestZH30_BrowsingStaysOpen(t *testing.T) {
	hh := New(t)
	hh.Setting("launch.customer_browse", "true")
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ ZH-30", 35.9506, 39.0094)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, true, zhShut())

	u := hh.Customer()
	for _, p := range []string{"/api/v1/public/home", "/api/v1/public/platform"} {
		if r := hh.GET(p, ""); r.Code != http.StatusOK {
			t.Errorf("**بابُ تصفّحٍ أُغلق مع المنطقة**: %s ⇒ %d", p, r.Code)
		}
	}
	if r := hh.GET("/api/v1/my/orders", u.Token); r.Code != http.StatusOK {
		t.Errorf("**الزبونُ حُجب عن طلباته**: %d", r.Code)
	}
}

// ═════════════════ ZH-32 · ZH-33 — اللوحةُ ودورةُ الحياة ═════════════════

// TestZH32_AdminValidation **والتحقّقُ في المحرّك لا في المتصفّح.**
func TestZH32_AdminValidation(t *testing.T) {
	hh := New(t)
	_, tok := capUser(t, hh, "admin")
	z := newZone(t, hh, "منطقةُ ZH-32", 35.9506, 39.0094)
	path := "/api/v1/admin/zones/" + z.ID + "/hours"

	bad := []map[string]any{
		{"day_of_week": 0, "start": "09:00", "end": "14:00"},
		{"day_of_week": 0, "start": "13:00", "end": "18:00"}, // تداخل
	}
	if r := hh.Call("PUT", path, tok, map[string]any{"enforced": true, "windows": bad}, nil); r.Code < 400 {
		t.Fatalf("**جدولٌ متداخلٌ قُبل**: %d", r.Code)
	}
	for _, one := range []map[string]any{
		{"day_of_week": 9, "start": "09:00", "end": "17:00"},
		{"day_of_week": 0, "start": "9", "end": "17:00"},
		{"day_of_week": 0, "start": "09:00", "end": "09:00"},
	} {
		body := map[string]any{"enforced": true, "windows": []any{one}}
		if r := hh.Call("PUT", path, tok, body, nil); r.Code < 400 {
			t.Errorf("**قُبل ما لا يُقبَل**: %v ⇒ %d", one, r.Code)
		}
	}
	// **ومنطقةٌ لا وجودَ لها تُردّ.**
	ghost := "/api/v1/admin/zones/00000000-0000-0000-0000-000000000009/hours"
	if r := hh.Call("PUT", ghost, tok, map[string]any{"enforced": true}, nil); r.Code < 400 {
		t.Fatalf("**جدولٌ كُتب لمنطقةٍ لا وجودَ لها**: %d", r.Code)
	}
}

// **والجدولُ يُكتب ويُقرأ كما كُتب — ولو أُطفئ سريانُه.**
func TestZH_AdminRoundTrip(t *testing.T) {
	hh := New(t)
	_, tok := capUser(t, hh, "admin")
	z := newZone(t, hh, "منطقةُ الدورة", 35.9506, 39.0094)
	path := "/api/v1/admin/zones/" + z.ID + "/hours"

	body := map[string]any{"enforced": false, "windows": []any{
		map[string]any{"day_of_week": 0, "start": "08:00", "end": "18:00"},
		map[string]any{"day_of_week": 0, "start": "20:00", "end": "02:00"},
	}}
	if r := hh.Call("PUT", path, tok, body, nil); r.Code != http.StatusOK {
		t.Fatalf("**جدولٌ صالحٌ رُدّ**: %d / %s", r.Code, r.Err())
	}
	got := hh.GET(path, tok).JSON()
	ws, _ := got["windows"].([]any)
	if len(ws) != 2 {
		t.Fatalf("**الجدولُ لم يُحفظ كما كُتب**: %v", got["windows"])
	}
	if got["enforced"] != false {
		t.Fatalf("**رايةُ السريان لم تُحفظ**: %v", got["enforced"])
	}
	first, _ := ws[0].(map[string]any)
	if first["start"] != "08:00" || first["end"] != "18:00" {
		t.Fatalf("**فترةٌ عادت مبدَّلة**: %v", first)
	}
}

// TestZH33_DeleteZoneDropsItsSchedule **ومحوُ المنطقة يمحو جدولَها.**
//
// **وجدولٌ يتيمٌ يبقى في القاعدة بلا منطقةٍ يصفها** — **ولا أحدَ يعرف
// لماذا هو هناك.**
func TestZH33_DeleteZoneDropsItsSchedule(t *testing.T) {
	hh := New(t)
	z := newZone(t, hh, "منطقةُ ZH-33", 35.9506, 39.0094)
	zoneHours(t, hh, z.ID, true, zhOpen())

	before := phCount(t, hh, `SELECT count(*) FROM delivery_zone_hours`)
	if before == 0 {
		t.Fatal("مقدّمةٌ مكسورة: لا جدولَ كُتب")
	}
	if _, err := hh.Pool.Exec(ctxBG(),
		`DELETE FROM delivery_zones WHERE id = $1::uuid`, z.ID); err != nil {
		t.Fatalf("**محوُ المنطقة سقط** — **وجدولُها يمنعه**: %v", err)
	}
	var left int
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM delivery_zone_hours WHERE zone_id = $1::uuid`, z.ID).
		Scan(&left); err != nil {
		t.Fatalf("عدٌّ: %v", err)
	}
	if left != 0 {
		t.Fatalf("**جدولٌ يتيمٌ بقي بعد محو منطقته**: %d صفّاً", left)
	}
}
