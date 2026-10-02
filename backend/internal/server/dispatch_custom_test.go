package server

// الطلبُ الخاصُّ في مسار التوزيع — **لا متجرَ له، ووصلةٌ داخليّةٌ تُسقطه بلا
// خطأ ولا سطرٍ في سجلّ.**
//
// # لماذا لزمت هذه الاختبارات
//
// **قِيس حيّاً على التجهيز ٢٠٢٦-٠٩-٢٩**: طلبٌ خاصٌّ (#1224) بقي في
// `dispatching` ثلاثاً وعشرين دقيقةً **ولم يُعرَض على سائقٍ واحد** — و`OfferNext`
// تقرأ `ErrNoRows` فتُفرّغ العرضَ وتعود، **ولأنّه `ErrNoRows` لا يُسجَّل شيء.**
//
// **والسببُ أنّ `orderDispatchInfo` تصل `merchants` وصلةً داخليّة**، والخاصُّ
// `merchant_id` فيه NULL. **وواحدٌ وعشرون ومئةُ طلبٍ خاصٍّ في القاعدة كلُّها
// كذلك.**
//
// **وهي العائلةُ التي يسمّيها التعليقُ في `admin_users_handlers.go`** ويقول إنّ
// المشيَ الحيَّ أمسكها خمسَ مرّاتٍ من قبل، **«وكلُّ مرّةٍ تُصلَح واحدةً ويبقى
// الباقي» — لأنّها تُكتب في كلّ استعلامٍ بيده.** فما لا اختبارَ له يعود.
//
// **ولا اختبارَ في `dispatch_proximity_test.go` يذكر `custom` ولا مرّةً** —
// فمرّت الميزةُ كلُّها وهي تُسقط منتجاً بأكمله.

import (
	"context"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

// customOrderAt يُنشئ طلباً خاصّاً بشكله الحقيقيّ — **بلا متجرٍ وبأصفارِ مال**
// (كما في `CreateCustomTx`: `kind='custom'` و`merchant_id` غائب).
func (f *driverFixture) customOrderAt(t *testing.T, dLat, dLng float64, status string) string {
	t.Helper()
	customer := newCustomer(t, f)
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO orders (kind, customer_id, address_text, dropoff, custom_request,
			status, payment_method, subtotal, delivery_fee, total, cash_due,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
		VALUES ('custom', $1, 'عنوانُ طلبٍ خاصّ',
		        ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography,
		        'قنينتا ماءٍ من سوق الأمين', $4, 'cash', 0, 0, 0, 0,
		        `+qaSnapSQL()+`)
		RETURNING id`, customer, dLat, dLng, status).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ طلبٍ خاصّ: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id)
	})
	return id
}

// ── أ · الخاصُّ يُعرض على سائقٍ مؤهَّل ────────────────────────────────────────
//
// **وهو الفحصُ الأدنى**: لا قُربَ ولا عدلَ ولا حداثة — **هل يُعرض أصلاً.**
func TestCustomOrder_IsOfferedToEligibleDriver(t *testing.T) {
	f := newDriverFixture(t, 1)
	armProximity(t, f)
	d := f.drivers[0]
	f.onShift(t, d, true)
	f.standAt(t, d, nearLat, nearLng)

	ord := f.customOrderAt(t, pdLat, pdLng, "dispatching")
	f.offer(t, ord, nil)

	want(t, f.offeredDriver(t, ord), d, "الطلبُ الخاصُّ يُعرض على المؤهَّل")
}

// ── ب · والحديثُ يسبق الشائخَ في الخاصِّ كما في العاديّ ──────────────────────
//
// **وهذا يمنع إصلاحاً نصفيّاً**: من بدّل الوصلةَ وحدَها (`LEFT JOIN`) أعاد
// الطلبَ إلى الكتلة الأولى — **`legacyRotationCandidate` بلا شرطِ حداثة** —
// فيُسلَّم لمن لم يُرسل موضعَه من أيّامٍ لأنّ ترتيبَها `last_assigned_at NULLS
// FIRST`. **وقرارُ المالك ٢٠٢٦-٠٩-٢٨: «لا تهبط إلى شائخ».**
//
// **والشائخُ هنا أقربُ وأقدمُ دوراً** — فلو فُضِّل لكان القربُ والعدلُ عذرَه،
// **ولا عذرَ لموضعٍ عمرُه ساعة.**
func TestCustomOrder_FreshPreferredOverStale(t *testing.T) {
	f := newDriverFixture(t, 2)
	armProximity(t, f)
	fresh, stale := f.drivers[0], f.drivers[1]
	f.onShift(t, fresh, true)
	f.onShift(t, stale, true)
	f.standAt(t, fresh, midLat, midLng)   // أبعدُ — ~٨٦٠م
	f.standAt(t, stale, nearLat, nearLng) // أقربُ — ~٤٠م
	// **والموضعُ يُعتَّق بعد الوقوف** — `standAt` تكتب `now()`.
	f.locationAgo(t, stale, 3600)      // أشيخُ من `location_fresh_sec` (٦٠٠)
	f.lastAssignedAgo(t, stale, 99999) // **والعدلُ يفضّله لولا الحداثة**
	f.lastAssignedAgo(t, fresh, 10)

	ord := f.customOrderAt(t, pdLat, pdLng, "dispatching")
	f.offer(t, ord, nil)

	want(t, f.offeredDriver(t, ord), fresh, "الحديثُ يسبق الشائخَ في الخاصّ")
}

// ── ب٢ · والقربُ يُقاس من باب الزبون في الخاصّ (٢٠٢٦-١٠-٠٢) ──────────────────
//
// **كان الخاصُّ بلا نقطةٍ تُقاس** — فيُعرض بالعدل وحدَه على سائقٍ بعيدٍ أقدمَ
// دوراً، **وآخرُ واقفٌ بجانب الزبون.**
func TestCustomOrder_NearestToCustomerIsOffered(t *testing.T) {
	f := newDriverFixture(t, 2)
	armProximity(t, f)
	byCustomer, far := f.drivers[0], f.drivers[1]
	f.onShift(t, byCustomer, true)
	f.onShift(t, far, true)
	f.standAt(t, byCustomer, pdLat+0.0003, pdLng+0.0003) // ~٤٠م من الزبون
	f.standAt(t, far, pmLat, pmLng)                      // ~١٫٤كم منه
	// **والعدلُ يفضّل البعيدَ لولا القرب.**
	f.lastAssignedAgo(t, far, 99999)
	f.lastAssignedAgo(t, byCustomer, 10)

	ord := f.customOrderAt(t, pdLat, pdLng, "dispatching")
	f.offer(t, ord, nil)

	want(t, f.offeredDriver(t, ord), byCustomer, "الأقربُ إلى الزبون في الخاصّ")
}

// ── ب٣ · وإلغاءُ الزبون والسائقُ في السوق يُقال للسائق (٢٠٢٦-١٠-٠٢) ─────────
//
// **الزبونُ يملك الإلغاءَ حتّى «اشتريتُ»** (قرارُ المالك ٢٠٢٦-٠٨-١٠) — **والسائقُ
// قد يكون في السوق.** فكان يُلغى عليه صامتاً.
func TestCustomOrder_CustomerCancelTellsTheDriver(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	f.srv.orders.SetSettings(settings.NewStore(f.pool))
	f.srv.orders.SetNotifier(notifications.New(f.pool, f.srv.hub, f.srv.logger))
	ord := f.customOrderAt(t, pdLat, pdLng, "dispatching")
	var customer string
	if err := f.pool.QueryRow(context.Background(), `
		UPDATE orders SET status = 'assigned', driver_id = $2, accepted_at = now()
		WHERE id = $1 RETURNING customer_id::text`, ord, d).Scan(&customer); err != nil {
		t.Fatal(err)
	}
	if _, err := f.srv.orders.Transition(context.Background(), customer, []string{"customer"}, ord, "cancelled", ""); err != nil {
		t.Fatalf("تعذّر الإلغاء: %v", err)
	}
	var body string
	if err := f.pool.QueryRow(context.Background(), `
		SELECT body FROM notifications WHERE user_id = $1 AND entity_id = $2
		ORDER BY created_at DESC LIMIT 1`, d, ord).Scan(&body); err != nil {
		t.Fatalf("لم يُخبَر السائقُ بالإلغاء: %v", err)
	}
	if !strings.Contains(body, "ألغى الزبون الطلب") {
		t.Errorf("النصّ %q", body)
	}
}

// ── ج · والراصدُ يرى الخاصَّ العالق ──────────────────────────────────────────
//
// **قِيس حيّاً**: `driver_timeout_min` عشرُ دقائق، والطلبُ الخاصُّ مضى عليه
// ثلاثٌ وعشرون — **واستعلامُ الراصد بوصلته لا يردّه**، وبلا الوصلة يردّه.
// **فالعالقُ الخاصُّ لا يظهر لغرفة العمليّات أبداً.**
func TestWatchdog_SeesStuckCustomOrder(t *testing.T) {
	f := newDriverFixture(t, 1)
	// **والمخزنُ يُحقَن صراحةً** — بلا حقنٍ تُقرأ الحدودُ من الفهرس لا من
	// القاعدة، **فيصير الاختبارُ يقيس افتراضاً لا إعداداً.**
	f.srv.orders.SetSettings(settings.NewStore(f.pool))
	f.setSetting(t, "orders.driver_timeout_min", 10)

	ord := f.customOrderAt(t, pdLat, pdLng, "dispatching")
	f.dispatchedAgo(t, ord, 30*60) // ثلاثون دقيقةً — ضعفُ الحدِّ وأكثر

	alerts, err := f.srv.orders.Alerts(context.Background())
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ التنبيهات: %v", err)
	}
	for _, a := range alerts {
		if a.OrderID == ord {
			if a.Reason != "no_driver" {
				t.Fatalf("السببُ %q والمتوقّع no_driver", a.Reason)
			}
			return
		}
	}
	t.Fatalf("الراصدُ لا يرى الطلبَ الخاصَّ العالق — %d تنبيهاً ولا واحدٌ له", len(alerts))
}

// ── د · وإنشاءُ الخاصِّ يُشعِر مكتبَ المنصّة ───────────────────────────────────
//
// **قِيس حيّاً بالمقارنة**: الطلبُ العاديُّ #1223 ولّد **١٨ إشعاراً**
// والخاصُّ #1224 ولّد **صفراً** — لأنّ `CreateCustomTx` لا تنادي
// `notifyCreated` أصلاً، **تنادي البثَّ وحدَه والبثُّ يصل الشاشاتِ المفتوحةَ
// فقط.** (وهي العلّةُ المكتوبةُ في `watchdog.go`: «فمن أغلق اللوحة ليلاً لم
// يصله شيء».)
func TestCustomOrder_NotifiesOpsOnCreate(t *testing.T) {
	f := newDriverFixture(t, 1)
	f.srv.orders.SetSettings(settings.NewStore(f.pool)) // **ولا لقطةَ اقتصادٍ بلا مخزن**
	// **والناقلُ حقيقيٌّ لا فارغ** — `NotifyMany` تبثّ قبل أن تكتب،
	// **ومؤشّرٌ فارغٌ فيها يذعر** (وهو عطبُ عُدّةٍ لا عطبُ منتج: الخادمُ
	// يحقن الناقلَ دائماً).
	f.srv.orders.SetNotifier(notifications.New(f.pool, f.srv.hub, f.srv.logger))

	// **ومن يستحقّ الإشعارَ يجب أن يوجد** — `NotifyRoles` تكتب لحامليه.
	ops := newOpsUser(t, f)
	customer := newCustomer(t, f)

	before := countUserNotifs(t, f, ops)
	o, err := f.srv.orders.CreateCustom(context.Background(), customer,
		"قنينتا ماءٍ من سوق الأمين", "عنوانُ طلبٍ خاصّ", "cash", "", pdLat, pdLng)
	if err != nil {
		t.Fatalf("تعذّر إنشاءُ الطلب الخاصّ: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, o.ID)
	})

	if after := countUserNotifs(t, f, ops); after == before {
		t.Fatalf("الطلبُ الخاصُّ لم يُشعِر المكتبَ — الإشعاراتُ %d قبلَه و%d بعدَه",
			before, after)
	}
}

func newOpsUser(t *testing.T, f *driverFixture) string {
	t.Helper()
	ctx := context.Background()
	var id string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO users (phone, full_name)
		VALUES ('+9639' || lpad((nextval('test_phone_seq') % 100000000)::text, 8, '0'),
		        'موظّفُ مكتب')
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ موظّف: %v", err)
	}
	if _, err := f.pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_code) VALUES ($1, 'ops')`, id); err != nil {
		t.Fatalf("تعذّر منحُ دور ops: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

func countUserNotifs(t *testing.T, f *driverFixture, userID string) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM notifications WHERE user_id = $1`, userID).Scan(&n); err != nil {
		t.Fatalf("تعذّر عدُّ الإشعارات: %v", err)
	}
	return n
}
