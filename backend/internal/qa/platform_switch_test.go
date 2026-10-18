package qa

// **زرُّ الدوام** (طلبُ المالك ٢٠٢٦-١٠-١٨: «زر واحد مثل المتجر والسائق بالعمل
// وخارج العمل») — يسبق الجدولَ في الاتّجاهين، **و«حسب الجدول» يعيده كما كان.**

import (
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/platform"
)

// **مسكّرين بالزرّ ⇒ «خارج أوقات الدوام» ولو كان الجدولُ مفتوحاً.**
func TestPlatformSwitch_ClosedBlocksInsideSchedule(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	openNow(t, hh)
	hh.Setting(platform.SwitchKey, `"closed"`)

	u := hh.Customer()
	it := hh.NewItem(900)
	if r := hh.POST("/api/v1/orders", u.Token, orderBody(it, 1)); !isClosedNow(r) {
		t.Fatalf("**طلبٌ مرّ والدوامُ مسكّرٌ بالزرّ**: %d / %s", r.Code, r.Err())
	}
	if r := hh.POST("/api/v1/orders/custom", u.Token, phCustomBody()); !isClosedNow(r) {
		t.Fatalf("**طلبٌ مخصَّصٌ مرّ والدوامُ مسكّرٌ بالزرّ**: %d / %s", r.Code, r.Err())
	}
}

// **فاتحين بالزرّ ⇒ يمضي الطلبُ ولو كان خارجَ الجدول.**
func TestPlatformSwitch_OpenAdmitsOutsideSchedule(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	closedNow(t, hh)
	hh.Setting(platform.SwitchKey, `"open"`)

	u := hh.Customer()
	it := hh.NewItem(900)
	if r := hh.POST("/api/v1/orders", u.Token, orderBody(it, 1)); r.Code >= 400 {
		t.Fatalf("**طلبٌ رُدّ والدوامُ مفتوحٌ بالزرّ**: %d / %s", r.Code, r.Err())
	}
}

// **«حسب الجدول» يعيد الحكمَ إلى الجدول** — والجدولُ لم يُمحَ.
func TestPlatformSwitch_ScheduleFollowsHours(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	setHours(t, hh, around(2*time.Hour, 3*time.Hour))
	hh.Setting(platform.EnforcedKey, "true")
	hh.Setting(platform.SwitchKey, `"schedule"`)

	u := hh.Customer()
	it := hh.NewItem(900)
	if r := hh.POST("/api/v1/orders", u.Token, orderBody(it, 1)); !isClosedNow(r) {
		t.Fatalf("**الجدولُ لم يحكم بعد ترك الزرّ**: %d / %s", r.Code, r.Err())
	}
}
