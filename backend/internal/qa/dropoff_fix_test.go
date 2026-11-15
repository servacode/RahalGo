package qa

import (
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **«هل هذا عنوانك الصحيح؟» تلقائيّاً عند التسليم** (قرارُ المالك ٢٠٢٦-١١-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// البابُ الحقيقيّ: موقعُ الزبون المباشر الدقيقُ الحديث، وإلّا آخرُ نقطةٍ للسائق.
// بين ١٠٠ م و٣ كم من الدبّوس، والدبّوسُ من عنوانٍ محفوظ ⇒ اقتراحٌ وإشعارٌ واحد.

type dropoffFx struct {
	h        *Harness
	cust     *User
	drv      *User
	oid      string
	lat, lng float64
}

// newDropoffFx **طلبٌ عاديّ بسائقٍ مُسنَد** — و`saved` يحفظ للزبون عنواناً على دبّوسه.
func newDropoffFx(t *testing.T, saved bool) dropoffFx {
	t.Helper()
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	var lat, lng float64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT ST_Y(dropoff::geometry), ST_X(dropoff::geometry)
		FROM orders WHERE id = $1::uuid`, oid).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if saved {
		if res := h.POST("/api/v1/my/addresses", cust.Token, map[string]any{
			"area_building": "الرقة — بيت الاختبار", "kind": "home", "lat": lat, "lng": lng,
		}); res.Code >= 300 {
			t.Fatalf("العنوان: %s", res)
		}
	}
	return dropoffFx{h: h, cust: cust, drv: h.driverOf(oid), oid: oid, lat: lat, lng: lng}
}

func (f dropoffFx) driverAt(t *testing.T, dLat float64, accuracy float64) {
	t.Helper()
	if _, err := f.h.Pool.Exec(ctxBG(), `INSERT INTO driver_track (driver_id, at, recorded_at, accuracy_m)
		VALUES ($1::uuid, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, now(), $4)`,
		f.drv.ID, f.lng, f.lat+dLat, accuracy); err != nil {
		t.Fatal(err)
	}
}

func (f dropoffFx) customerLiveAt(t *testing.T, dLat float64, accuracy float64) {
	t.Helper()
	if _, err := f.h.Pool.Exec(ctxBG(), `INSERT INTO order_live_locations (order_id, customer_id, point, accuracy_m, at)
		VALUES ($1::uuid, $2::uuid, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, $5, now())`,
		f.oid, f.cust.ID, f.lng, f.lat+dLat, accuracy); err != nil {
		t.Fatal(err)
	}
}

// suggested **نقطةُ الاقتراح كما يراها الزبون** — وصفرٌ لا اقتراح.
func (f dropoffFx) suggested(t *testing.T) float64 {
	t.Helper()
	mine := f.h.GET("/api/v1/my/orders/"+f.oid, f.cust.Token)
	if mine.Code != 200 {
		t.Fatalf("طلبُ الزبون: %s", mine)
	}
	v, _ := mine.JSON()["dropoff_suggest_lat"].(float64)
	return v
}

func (f dropoffFx) asked(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
		WHERE user_id = $1::uuid AND entity_id = $2 AND title = 'هل هذا عنوانك الصحيح؟'`,
		f.cust.ID, f.oid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func nearLat(got, want float64) bool { return got > want-0.00005 && got < want+0.00005 }

// TestDropoffFix_CustomerLiveWins **موقعُ الزبون المباشر يسبق السائق** — ولو كان للسائق نقطةٌ بعيدةٌ أخرى.
// كان الزنادُ يمحو الموقعَ المباشر مع التسليم فلا يُقرأ أبداً، فتغلب نقطةُ السائق.
func TestDropoffFix_CustomerLiveWins(t *testing.T) {
	f := newDropoffFx(t, true)
	f.customerLiveAt(t, 0.0045, 8) // ~٥٠٠ م
	f.driverAt(t, 0.0027, 10)      // ~٣٠٠ م
	deliverOrder(t, f.h, f.oid, f.drv)
	if got := f.suggested(t); !nearLat(got, f.lat+0.0045) {
		t.Fatalf("الاقتراحُ ليس موقعَ الزبون المباشر: %.5f (المتوقّع %.5f)", got, f.lat+0.0045)
	}
	if n := f.asked(t); n != 1 {
		t.Fatalf("الزبونُ سُئل %d مرّة، والمطلوب مرّةٌ واحدة", n)
	}
}

// TestDropoffFix_ImpreciseLiveFallsBackToDriver **موقعٌ مباشرٌ تقريبيّ لا يُعتمد** — فنقطةُ السائق.
func TestDropoffFix_ImpreciseLiveFallsBackToDriver(t *testing.T) {
	f := newDropoffFx(t, true)
	f.customerLiveAt(t, 0.0045, 400)
	f.driverAt(t, 0.0027, 10)
	deliverOrder(t, f.h, f.oid, f.drv)
	if got := f.suggested(t); !nearLat(got, f.lat+0.0027) {
		t.Fatalf("الاقتراحُ ليس نقطةَ السائق: %.5f", got)
	}
}

// TestDropoffFix_FarDriverBeyondOldCap **السائقُ سلّم على ~١ كم** — أبعدُ من حدّ الـ١٥٠ م القديم، فيُسأل.
func TestDropoffFix_FarDriverBeyondOldCap(t *testing.T) {
	f := newDropoffFx(t, true)
	f.driverAt(t, 0.009, 10)
	deliverOrder(t, f.h, f.oid, f.drv)
	if got := f.suggested(t); !nearLat(got, f.lat+0.009) {
		t.Fatalf("الزبونُ لم يُسأل: %.5f", got)
	}
	if n := f.asked(t); n != 1 {
		t.Fatalf("الزبونُ سُئل %d مرّة", n)
	}
	// **وجوابُه يُحترم** — «لا، عنواني صحيح» يمحو الاقتراح، وبابُ السائق بعده لا يعيد السؤال.
	if res := f.h.DEL("/api/v1/orders/"+f.oid+"/accept-dropoff-fix", f.cust.Token); res.Code != 200 {
		t.Fatalf("الرفض: %s", res)
	}
	if got := f.suggested(t); got != 0 {
		t.Fatalf("الاقتراحُ باقٍ بعد الرفض: %.5f", got)
	}
	res := f.h.POST("/api/v1/driver/orders/"+f.oid+"/entrance", f.drv.Token, map[string]any{"lat": f.lat + 0.002, "lng": f.lng})
	if res.Code != 200 || res.JSON()["suggested"] != false {
		t.Fatalf("السؤالُ أُعيد بعد الرفض: %s", res)
	}
	if n := f.asked(t); n != 1 {
		t.Fatalf("الزبونُ سُئل %d مرّة بعد الرفض", n)
	}
}

// TestDropoffFix_None **لا سؤال**: قريبٌ من الدبّوس، أو أبعدُ من ٣ كم، أو بلا عنوانٍ محفوظ، أو نقطةٌ قديمة.
func TestDropoffFix_None(t *testing.T) {
	cases := []struct {
		name  string
		saved bool
		dLat  float64
		stale bool
	}{
		{"قريب ٦٠ م", true, 0.00055, false},
		{"أبعد من ٣ كم", true, 0.03, false},
		{"بلا عنوان محفوظ", false, 0.0027, false},
		{"نقطة السائق قديمة", true, 0.0027, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newDropoffFx(t, c.saved)
			f.driverAt(t, c.dLat, 10)
			if c.stale {
				if _, err := f.h.Pool.Exec(ctxBG(), `UPDATE driver_track SET created_at = now() - interval '10 minutes',
					recorded_at = now() - interval '10 minutes' WHERE driver_id = $1::uuid`, f.drv.ID); err != nil {
					t.Fatal(err)
				}
			}
			deliverOrder(t, f.h, f.oid, f.drv)
			if got := f.suggested(t); got != 0 {
				t.Fatalf("سُئل الزبونُ ولا ينبغي: %.5f", got)
			}
			if n := f.asked(t); n != 0 {
				t.Fatalf("إشعارٌ لا ينبغي: %d", n)
			}
		})
	}
}

// TestEntrance_BeyondCapStillSuggests **بابٌ على ٢٢٠ م** — لا يُحفظ مدخلاً (أبعدُ من ١٥٠ م)،
// **ويُسأل الزبونُ مع ذلك** (٢٠٢٦-١١-١٥: كان يُردّ `entrance_too_far` فلا يُسأل أحد)؛
// **والضغطةُ الثانية لا تكرّر السؤال**، والموافقةُ تنقل العنوان.
func TestEntrance_BeyondCapStillSuggests(t *testing.T) {
	h := New(t)
	const lat, lng = 35.9447, 39.0031
	cust, drv, id := mcDeliveredOrder(t, h, lat, lng)
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET status = 'delivered' WHERE id = $1::uuid`, id); err != nil {
		t.Fatal(err)
	}
	door := map[string]any{"lat": lat + 0.002, "lng": lng}
	res := h.POST("/api/v1/driver/orders/"+id+"/entrance", drv.Token, door)
	if res.Code != 200 || res.JSON()["saved"] != false || res.JSON()["suggested"] != true {
		t.Fatalf("بابٌ على ٢٢٠ م: %s", res)
	}
	again := h.POST("/api/v1/driver/orders/"+id+"/entrance", drv.Token, door)
	if again.Code != 200 || again.JSON()["suggested"] != false {
		t.Fatalf("الضغطةُ الثانية كرّرت السؤال: %s", again)
	}
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
		WHERE user_id = $1::uuid AND entity_id = $2 AND title = 'هل هذا عنوانك الصحيح؟'`, cust.ID, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("الزبونُ سُئل %d مرّة", n)
	}
	if ok := h.POST("/api/v1/orders/"+id+"/accept-dropoff-fix", cust.Token, map[string]any{}); ok.Code != 200 {
		t.Fatalf("الموافقة: %s", ok)
	}
	var moved float64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT ST_Distance(location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography)
		FROM user_addresses WHERE user_id = $1::uuid`, cust.ID, lng, lat+0.002).Scan(&moved); err != nil {
		t.Fatal(err)
	}
	if moved > 1 {
		t.Fatalf("نقطةُ العنوان لم تنتقل (بُعد %.0f م)", moved)
	}
}
