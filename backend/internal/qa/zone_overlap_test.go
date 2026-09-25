package qa

// ══════════════════════════════════════════════════════════════════════
// **مناطقُ متداخلةٌ — المفتوحةُ تُنقذ** (`ZOV`، ٢٠٢٦-٠٩-٢٥)
// ══════════════════════════════════════════════════════════════════════
//
// (قرار المالك: نقطةٌ تغطّيها أكثرُ من منطقة ⇒ إن كانت واحدةٌ مفتوحةً
//  فالتوصيلُ متاح؛ منطقةٌ مغلقةٌ لا تحجب ما تفتحه أخرى. واختيارُ النافذة
//  حاسمٌ: `sort_order` ثمّ الأقربَ مركزاً ثمّ `id`.)
//
// **وهذه تقيس ما لم يكن مقيساً**: كانت `ZoneAt` تختار الأقربَ مركزاً
// هندسيّاً ثمّ تفحص وقتَها وحدَها — **فمنطقةٌ مغلقةٌ أقربُ مركزاً تحجب
// طلباً تغطّيه منطقةٌ مفتوحة.** والحزمةُ القائمةُ تتجنّب التداخلَ عمداً
// (`otherZonesOff`)، فلم يُقَس هذا الحال قطّ.

import (
	"testing"
)

// keepActiveZones **يُطفئ كلَّ منطقةٍ فعّالةٍ سوى المذكورةِ — ثمّ يُعيدها.**
//
// **ونظيرُ `otherZonesOff` لكنّه يُبقي أكثرَ من واحدة** — فالتداخلُ يلزمه
// منطقتان حيّتان على النقطة نفسِها.
func keepActiveZones(t *testing.T, h *Harness, keep ...string) {
	t.Helper()
	inSet := map[string]bool{}
	for _, id := range keep {
		inSet[id] = true
	}
	rows, err := h.Pool.Query(ctxBG(), `SELECT id::text FROM delivery_zones WHERE active`)
	if err != nil {
		t.Fatalf("قراءةُ المناطق الفعّالة: %v", err)
	}
	var off []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatalf("قراءةُ منطقة: %v", err)
		}
		if !inSet[id] {
			off = append(off, id)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("قراءةُ المناطق الفعّالة: %v", err)
	}
	if len(off) == 0 {
		return
	}
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE delivery_zones SET active = false WHERE id = ANY($1::uuid[])`, off); err != nil {
		t.Fatalf("إطفاءُ المناطق: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(),
			`UPDATE delivery_zones SET active = true WHERE id = ANY($1::uuid[])`, off)
	})
}

// zoneSort **يضبط أولويّةَ منطقةٍ** (`sort_order`) — أصغرُها يفوز عند التداخل.
func zoneSort(t *testing.T, h *Harness, id string, order int) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE delivery_zones SET sort_order = $2 WHERE id = $1::uuid`, id, order); err != nil {
		t.Fatalf("ضبطُ sort_order: %v", err)
	}
}

// orderZoneName **اسمُ المنطقةِ التي سُجِّلت على الطلب** — لنرى أيَّ منطقةٍ فازت.
func orderZoneName(t *testing.T, h *Harness, orderID string) string {
	t.Helper()
	var name string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT z.name FROM orders o JOIN delivery_zones z ON z.id = o.zone_id
		 WHERE o.id = $1::uuid`, orderID).Scan(&name); err != nil {
		t.Fatalf("قراءةُ منطقةِ الطلب: %v", err)
	}
	return name
}

// نقطةُ الاختبار في الرقّة (مدينةٌ مُطلَقة)، ومركزٌ ثانٍ ~١١١م شمالَها،
// فتقع النقطةُ داخلَ الدائرتين (نصفُ قطر كلٍّ ٢كم).
const (
	zovLat  = 35.9506
	zovLng  = 39.0094
	zovLat2 = 35.9516 // +0.001° ≈ 111م
)

// TestZOV1_OpenZoneRescuesClosedNearer **منطقةٌ مفتوحةٌ تُنقذ نقطةً
// تحجبها منطقةٌ مغلقةٌ أقربُ مركزاً.**
//
// **وهو العطبُ بعينه**: المغلقةُ مركزُها النقطةُ (الأقرب)، والمفتوحةُ
// أبعدُ قليلاً وتغطّيها. **قبل الإصلاح: `zone_closed_now`. بعده: يُقبَل
// بمنطقةِ المفتوحة.**
func TestZOV1_OpenZoneRescuesClosedNearer(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)

	closedNear := newZone(t, hh, "ZOV1-مغلقةٌ-أقرب", zovLat, zovLng) // مركزُها النقطة
	openFar := newZone(t, hh, "ZOV1-مفتوحةٌ-أبعد", zovLat2, zovLng)  // ~١١١م شمالاً، تغطّي النقطة
	keepActiveZones(t, hh, closedNear.ID, openFar.ID)
	zoneHours(t, hh, closedNear.ID, true, zhShut()) // مغلقةٌ الآن، سارٍ جدولُها
	// openFar تبقى غيرَ سارية (hours_enforced=false) ⇒ مفتوحةٌ دائماً.

	u := hh.Customer()
	it := hh.NewItem(900)
	r := hh.POST("/api/v1/orders", u.Token, zoneBody(it, zovLat, zovLng))
	if r.Code >= 400 {
		t.Fatalf("**طُلبٌ رُدّ رغمَ وجودِ منطقةٍ مفتوحةٍ تغطّي النقطة**: %d / %s",
			r.Code, r.Err())
	}
	oid, _ := r.JSON()["id"].(string)
	if oid == "" {
		t.Fatalf("**لا معرّفَ طلبٍ في الردّ**: %s", r.String())
	}
	if got := orderZoneName(t, hh, oid); got != "ZOV1-مفتوحةٌ-أبعد" {
		t.Fatalf("**سُعِّر الطلبُ بمنطقةٍ غيرِ المفتوحة**: %q", got)
	}
}

// TestZOV2_AllCoveringClosedIsZoneClosedNow **وإن كانت كلُّ المناطق
// المُغطِّية مغلقةً ⇒ `zone_closed_now`** — لا يُنقِذ أحدٌ لأنّ لا مفتوحَ.
func TestZOV2_AllCoveringClosedIsZoneClosedNow(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)

	a := newZone(t, hh, "ZOV2-أ", zovLat, zovLng)
	b := newZone(t, hh, "ZOV2-ب", zovLat2, zovLng)
	keepActiveZones(t, hh, a.ID, b.ID)
	zoneHours(t, hh, a.ID, true, zhShut())
	zoneHours(t, hh, b.ID, true, zhShut())

	u := hh.Customer()
	it := hh.NewItem(900)
	r := hh.POST("/api/v1/orders", u.Token, zoneBody(it, zovLat, zovLng))
	if !isZoneClosed(r) {
		t.Fatalf("**كلُّ المناطق مغلقةٌ ولم يُردّ بـ`zone_closed_now`**: %d / %s",
			r.Code, r.Err())
	}
}

// TestZOV3_DeterministicBySortOrder **وعند تعدّدِ المفتوحةِ يفوز أصغرُ
// `sort_order`** — لا الأقربُ مركزاً. **حاسمٌ ولو اختلفت الرسومُ/الجداول.**
func TestZOV3_DeterministicBySortOrder(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)

	near := newZone(t, hh, "ZOV3-أقربُ-أولويّةٌ-أدنى", zovLat, zovLng) // مركزُها النقطة (الأقرب)
	far := newZone(t, hh, "ZOV3-أبعدُ-أولويّةٌ-أعلى", zovLat2, zovLng) // أبعدُ لكنْ أولى
	keepActiveZones(t, hh, near.ID, far.ID)
	// كلتاهما مفتوحةٌ (غيرُ سارية). الأبعدُ أولويّتُها أعلى (sort_order أصغر).
	zoneSort(t, hh, near.ID, 10)
	zoneSort(t, hh, far.ID, 5)

	u := hh.Customer()
	it := hh.NewItem(900)
	r := hh.POST("/api/v1/orders", u.Token, zoneBody(it, zovLat, zovLng))
	if r.Code >= 400 {
		t.Fatalf("**طُلبٌ رُدّ رغمَ منطقتين مفتوحتين**: %d / %s", r.Code, r.Err())
	}
	oid, _ := r.JSON()["id"].(string)
	// **الأبعدُ (sort_order=5) تفوز على الأقرب (10)** — الأولويّةُ تسبق القرب.
	if got := orderZoneName(t, hh, oid); got != "ZOV3-أبعدُ-أولويّةٌ-أعلى" {
		t.Fatalf("**فازت المنطقةُ الخطأ — `sort_order` لم يُقدَّم على القرب**: %q", got)
	}
}
