package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// TestDriverRoute_FinishedOrderHasNoRoute **طلبٌ انتهى لا طريقَ له عند السائق** (قرارُ المالك
// ٢٠٢٦-١١-٠٥، البند ٢ — مراجعةُ الفرع). كان البابُ يسأل «أهو سائقُه؟» وحدَها، **وذلك يبقى صحيحاً
// بعد التسليم** — فيردّ خطَّ الشوارع إلى باب الزبون لكلّ رقمٍ في سجلّه، **فيجمع أبوابَ زبائنه.**
// **ومشوارُ الإرجاع المفتوح يبقى مرسوماً** — البضاعةُ معه.
func TestDriverRoute_FinishedOrderHasNoRoute(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	f.srv.route = &fakeRoutes{}
	// **والمكتبُ قبل أيّ نداء** — الإعداداتُ تُقرأ مرّةً وتُحفظ.
	f.armOffice(t, "35.9600,39.0100", "مكتب رحّال — شارع الاختبار")
	ctx := context.Background()
	route := func(id string) map[string]any {
		t.Helper()
		w := f.call(f.srv.handleDriverOrderRoute, http.MethodGet, "/driver/orders/"+id+"/route", id, d,
			[]string{"driver"}, "")
		if w.Code != http.StatusOK {
			t.Fatalf("المسارُ ردّ %d: %s", w.Code, w.Body.String())
		}
		var body struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.Data
	}

	id := f.deliveryOrderAt(t, "on_the_way", d, true, "recipient")
	if r := route(id); r["available"] != true {
		t.Fatalf("قبل التسليم: available=%v — **والطريقُ إلى الباب يُرسم**", r["available"])
	}
	for _, st := range []string{"delivered", "cancelled", "failed"} {
		if _, err := f.pool.Exec(ctx, `
			UPDATE orders SET status = $2, closed_at = now(), return_to = NULL, goods_handed_at = NULL
			WHERE id = $1`, id, st); err != nil {
			t.Fatal(err)
		}
		r := route(id)
		if r["available"] != false {
			t.Errorf("%s: available=%v — **طلبٌ انتهى يردّ طريقاً إلى باب الزبون**", st, r["available"])
		}
		if _, ok := r["points"]; ok {
			t.Errorf("%s: في الجواب نقاطُ المسار — **إحداثيّاتُ باب زبونٍ انتهى طلبُه**", st)
		}
	}

	// **ومشوارُ إرجاعٍ لم تُسلَّم بضاعتُه يبقى طريقاً** — إلى المكتب لا إلى الزبون، **ومن
	// موضعه** (لا متجرَ يُبدأ منه).
	if _, err := f.pool.Exec(ctx, `
		UPDATE users SET last_location = ST_SetSRID(ST_MakePoint(39.01, 35.95), 4326)::geography,
		       last_location_at = now() WHERE id = $1`, d); err != nil {
		t.Fatal(err)
	}
	ret := f.atDropoffOrder(t, d)
	if w := f.endAtDoor(t, ret, "customer", "customer_refused"); w.Code != http.StatusOK {
		t.Fatalf("العودةُ رُدّت: %d — %s", w.Code, w.Body.String())
	}
	if r := route(ret); r["available"] != true {
		t.Errorf("مشوارُ الإرجاع: available=%v — **والبضاعةُ معه ولا طريقَ إلى المكتب**", r["available"])
	}
}
