package server

// ══════════════════════════════════════════════════════════════════════
// **الكابتنُ عند الزبون بالأمتار** (قرارُ المالك ٢٠٢٦-١٠-١٦)
// ══════════════════════════════════════════════════════════════════════
//
// **حكمان يُقاسان على القاعدة الحقيقيّة**:
//
//   - **الطريقُ يتحوّل إلى الزبون بشرطين** — أبعدُ من خمسين متراً عن الدبّوس **ودقيقٌ**
//     (دقّتُه دون ثلاثين). **والتقريبيُّ يُرى ولا يحوّل.**
//   - **«الكابتن قريب منك» مرّةً واحدة** — حين يدخل ثلاثمئة مترٍ من الوجهة بعد
//     الاستلام، **لا قبله ولا ثانيةً.**

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// على خطّ الطول نفسِه شمالَ الدبّوس وجنوبَه (الدبّوسُ 35.9528 · 39.0079) — **الدرجةُ نحو ١١١كم.**
const (
	// **مئةُ مترٍ ودقّةُ اثني عشر** — فوق حدّ الخمسين ودقيقة.
	preciseAt100 = `{"lat": 35.9537, "lng": 39.0079, "accuracy_m": 12}`
	// **ثلاثمئةٍ ونيّفٌ ودقّةُ خمسةٍ وأربعين** — بعيدةٌ وتقريبيّة.
	impreciseAt330 = `{"lat": 35.9558, "lng": 39.0079, "accuracy_m": 45}`
	// **ثلاثمئةٍ ونيّفٌ بلا دقّة** — جهازٌ لم يقل كم يخطئ.
	unknownAt330 = `{"lat": 35.9558, "lng": 39.0079}`
)

// TestLiveLocation_ImprecisePointNeverSwitches **التقريبيُّ يُرى ولا يحوّل الطريق** — ولو بعُد.
func TestLiveLocation_ImprecisePointNeverSwitches(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	for _, body := range []string{impreciseAt330, unknownAt330} {
		orderID, owner := f.liveOrder(t, "on_the_way", &d)
		if w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, orderID, owner, "customer", body); w.Code != http.StatusOK {
			t.Fatalf("الكتابة: %d %s", w.Code, w.Body.String())
		}
		p, ok := liveOf(f.driverOrdersOf(t, d), orderID)
		if !ok {
			t.Fatalf("%s: التقريبيُّ لم يصل السائق — يُرى ولو لم يحوّل", body)
		}
		if gap, _ := p["pin_gap_m"].(float64); gap < 300 {
			t.Fatalf("بُعدُ الدبّوس %v — المتوقّع فوق ٣٠٠", gap)
		}
		if pr, has := p["precise"]; !has || pr != false {
			t.Errorf("%s: precise = %v (%v) لا false", body, pr, has)
		}
		if s := f.srv.liveRouteTarget(context.Background(), orderID, d); s != nil {
			t.Errorf("%s: نقطةٌ تقريبيّةٌ حوّلت الطريق", body)
		}
	}
}

// TestLiveLocation_PreciseFarPointSwitchesAt50 **الدقيقُ فوق الخمسين يحوّل** — بالافتراض الجديد.
func TestLiveLocation_PreciseFarPointSwitchesAt50(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	orderID, owner := f.liveOrder(t, "on_the_way", &d)
	w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, orderID, owner, "customer", preciseAt100)
	if w.Code != http.StatusOK {
		t.Fatalf("الكتابة: %d %s", w.Code, w.Body.String())
	}
	// **والحدُّ يعود إلى هاتف الزبون** — به يرشّح قراءاتِه.
	var ack struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &ack)
	if ack.Data["max_accuracy_m"] != float64(30) {
		t.Errorf("ردُّ الكتابة بلا حدّ الدقّة ٣٠: %v", ack.Data)
	}
	p, ok := liveOf(f.driverOrdersOf(t, d), orderID)
	if !ok {
		t.Fatal("النقطةُ لم تصل")
	}
	if p["switch_m"] != float64(50) || p["precise"] != true {
		t.Errorf("switch_m=%v precise=%v — المتوقّع ٥٠ وtrue", p["switch_m"], p["precise"])
	}
	if s := f.srv.liveRouteTarget(context.Background(), orderID, d); s == nil {
		t.Error("دقيقٌ على مئة مترٍ من الدبّوس ولم يتحوّل الطريق")
	}
}

// nearHub **يُركّب الإشعارات على بثّ العُدّة** — ويردّ عدّاداً لما يصل الزبون.
func (f *driverFixture) nearHub(t *testing.T, customer string) func() int {
	t.Helper()
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	f.srv.notify = notifications.New(f.pool, f.srv.hub, quiet)
	ch, cancel := f.srv.hub.Subscribe([]string{"user:" + customer})
	t.Cleanup(cancel)
	seen := 0
	return func() int {
		for {
			select {
			case raw := <-ch:
				var ev struct {
					Notification notifications.Notification `json:"notification"`
				}
				_ = json.Unmarshal(raw, &ev)
				if ev.Notification.Title == nearNoticeTitle {
					seen++
				}
			default:
				return seen
			}
		}
	}
}

func (f *driverFixture) pingAt(t *testing.T, driver, lat string) {
	t.Helper()
	w := f.call(f.srv.handleDriverLocation, http.MethodPost, "/driver/location", "", driver,
		[]string{"driver"}, `{"lat":`+lat+`,"lng":39.0079,"accuracy_m":5}`)
	if w.Code != http.StatusOK {
		t.Fatalf("النبضة: %d %s", w.Code, w.Body.String())
	}
}

// TestNearNotice_OncePerOrderAfterPickup **«الكابتن قريب منك» مرّةً — بعد الاستلام وحدَه.**
func TestNearNotice_OncePerOrderAfterPickup(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	orderID, owner := f.liveOrder(t, "assigned", &d)
	seen := f.nearHub(t, owner)

	// **قبل الاستلام لا تنبيه** — ولو مرّ على مئة مترٍ من البيت.
	f.pingAt(t, d, "35.9537")
	if n := seen(); n != 0 {
		t.Fatalf("نُبّه الزبونُ قبل الاستلام: %d", n)
	}
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET status = 'picked_up' WHERE id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	// **وبعيدٌ عن الحدّ لا تنبيه** — كيلومترٌ شمالاً.
	f.pingAt(t, d, "35.9618")
	if n := seen(); n != 0 {
		t.Fatalf("نُبّه والكابتنُ على كيلومتر: %d", n)
	}
	// **ودخل ثلاثمئة متر** — ٢٥٠م: تنبيهٌ واحد.
	f.pingAt(t, d, "35.95505")
	if n := seen(); n != 1 {
		t.Fatalf("دخل الحدَّ والتنبيهات %d لا ١", n)
	}
	// **ولا ثانيةً** — اقترب أكثر، وتبدّلت الحالُ إلى «بالطريق».
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET status = 'on_the_way' WHERE id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	f.pingAt(t, d, "35.9533")
	f.pingAt(t, d, "35.9529")
	if n := seen(); n != 1 {
		t.Fatalf("تكرّر التنبيه: %d لا ١", n)
	}
}

// TestNearNotice_TargetIsThePreciseLivePoint **الوجهةُ وجهةُ الطريق** — الزبونُ الدقيقُ لا الدبّوس.
func TestNearNotice_TargetIsThePreciseLivePoint(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	orderID, owner := f.liveOrder(t, "on_the_way", &d)
	seen := f.nearHub(t, owner)
	// **الزبونُ دقيقٌ على ٨٠٠م جنوبَ الدبّوس** — فالطريقُ إليه.
	if w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, orderID, owner, "customer",
		`{"lat": 35.9456, "lng": 39.0079, "accuracy_m": 6}`); w.Code != http.StatusOK {
		t.Fatalf("الكتابة: %d", w.Code)
	}
	// **والكابتنُ على ٢٥٠م شمالَ الدبّوس** — قريبٌ من الدبّوس الخطأ، بعيدٌ عن الزبون.
	f.pingAt(t, d, "35.95505")
	if n := seen(); n != 0 {
		t.Fatalf("نُبّه والكابتنُ قريبٌ من الدبّوس لا من الزبون: %d", n)
	}
	// **ووصل ٢٠٠م من الزبون** — تنبيه.
	f.pingAt(t, d, "35.9474")
	if n := seen(); n != 1 {
		t.Fatalf("دخل حدَّ الزبون والتنبيهات %d لا ١", n)
	}
}
