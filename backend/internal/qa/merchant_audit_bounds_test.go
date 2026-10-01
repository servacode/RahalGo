package qa

// **ثغراتُ الفحص الشامل لتطبيق المتجر** — ٢٠٢٦-١٠-٠١، قِيست على التجهيز.
//
//   - CAP01 صنفٌ بتسعة تريليونات قُبل ⇒ **أعلى سعرٍ ١٠٠٬٠٠٠** (قرارُ المالك).
//   - CAP02 اسمٌ بخمسة آلاف حرفٍ ووصفٌ بعشرين ألفاً خُزّنا ⇒ ١٢٠ و٥٠٠.
//   - CASH01 ثلاثةُ طلباتٍ أُسنِدت تلقائيّاً إلى سائقٍ في ثلاث ثوانٍ فبلغ
//     نقدُه ٥٦١٬٤٠٠ والسقفُ ٥٠٠٬٠٠٠ — **الإسنادُ التلقائيُّ لم يكن يمرّ بحارس القبول.**

import (
	"net/http"
	"strings"
	"testing"
)

type capStore struct {
	m    *Merchant
	tok  string
	plat string
}

func newCapStore(t *testing.T, h *Harness) capStore {
	t.Helper()
	f := h.Factory()
	rep := h.NewUser("sales")
	m := f.Merchant(OwnedByRep(rep.ID))
	seed := h.NewItemFor(m, 1500)
	return capStore{m: m, tok: h.TokenFor(m.Owner.ID, "merchant"), plat: seed.SectionID}
}

func (c capStore) add(h *Harness, body map[string]any) Res {
	body["platform_section_id"] = c.plat
	return h.POSTKey("/api/v1/merchant/stores/"+c.m.ID+"/menu/items", c.tok, uniq("cap"), body)
}

func TestCAP01_ItemPriceCeiling(t *testing.T) {
	h := New(t)
	h.Setting("merchants.max_item_price", "100000")
	cs := newCapStore(t, h)
	over := cs.add(h, map[string]any{"name": "غالٍ", "price": 100001})
	if over.Code != http.StatusBadRequest || over.Err() != "item_price_too_high" {
		t.Fatalf("سعرٌ فوق الحدّ: %d / %s — والمنتظَرُ 400 / item_price_too_high", over.Code, over.Err())
	}
	at := cs.add(h, map[string]any{"name": "عند الحدّ", "price": 100000})
	if at.Code != http.StatusCreated {
		t.Fatalf("السعرُ عند الحدّ تماماً يُقبل: %d / %s", at.Code, at.Err())
	}
	id, _ := at.JSON()["id"].(string)
	// **والتعديلُ بابٌ ثانٍ** — حارسٌ على واحدٍ من بابين ليس حارساً.
	up := h.Call("PATCH", "/api/v1/merchant/menu/items/"+id, cs.tok, map[string]any{"price": 9_000_000_000_000}, nil)
	if up.Code != http.StatusBadRequest || up.Err() != "item_price_too_high" {
		t.Fatalf("تعديلُ السعر فوق الحدّ: %d / %s", up.Code, up.Err())
	}
}

func TestCAP02_ItemTextLimits(t *testing.T) {
	h := New(t)
	cs := newCapStore(t, h)
	for _, c := range []struct {
		name string
		body map[string]any
		want int
	}{
		{"اسم ١٢١ حرفاً", map[string]any{"name": strings.Repeat("ش", 121), "price": 500}, 400},
		{"وصف ٥٠١ حرف", map[string]any{"name": "صنف", "description": strings.Repeat("و", 501), "price": 500}, 400},
		{"اسم ١٢٠ ووصف ٥٠٠", map[string]any{"name": strings.Repeat("ش", 120), "description": strings.Repeat("و", 500), "price": 500}, 201},
	} {
		r := cs.add(h, c.body)
		if r.Code != c.want {
			t.Fatalf("%s: %d / %s — والمنتظَرُ %d", c.name, r.Code, r.Err(), c.want)
		}
		if c.want == 400 && r.Err() != "item_text_too_long" {
			t.Fatalf("%s: الرمزُ %s — والمنتظَرُ item_text_too_long", c.name, r.Err())
		}
	}
}

// TestCASH01_AutoAssignObeysCashCeiling **الإسنادُ التلقائيُّ لا يتجاوز سقفَ النقد.**
//
// سقفٌ ٥٠٠٬٠٠٠ · طلبٌ بـ٣٠٠٬٠٠٠ يُسنَد · **ثمّ طلبٌ بـ٢٥٠٬٠٠٠ لا يُسنَد إليه**
// — والمحصَّلُ صفر: **المُسنَدُ الذي لم يُسلَّم هو التعرّض.**
func TestCASH01_AutoAssignObeysCashCeiling(t *testing.T) {
	h := New(t)
	fx := newCashFixture(t, h, 500_000)
	h.Setting("drivers.assignment_mode", `"rotation"`)
	h.Setting("drivers.direct_assign", "true")
	h.Setting("drivers.proximity_enabled", "false")
	h.Setting("drivers.same_route_extra", "0")
	// **وسائقٌ واحدٌ بالدوام** — سائقو الفحوص السابقة في القاعدة نفسِها يأخذون
	// الدورَ فلا يُقاس سقفُ هذا. (قاعدةُ الفحص لا التجهيز.)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE users SET on_shift = false WHERE on_shift AND id <> $1::uuid`, fx.Driver.ID); err != nil {
		t.Fatal(err)
	}

	first, due1 := cashOrder(t, h, fx.Cust, 300_000, 1)
	if err := h.Orders.OfferNext(ctxBG(), first, nil); err != nil {
		t.Fatal(err)
	}
	var drv1 *string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT driver_id::text FROM orders WHERE id = $1`, first).Scan(&drv1); err != nil {
		t.Fatal(err)
	}
	if drv1 == nil || *drv1 != fx.Driver.ID {
		var st string
		var off *string
		_ = h.Pool.QueryRow(ctxBG(), `SELECT status, offered_driver_id::text FROM orders WHERE id = $1`, first).Scan(&st, &off)
		t.Logf("الحال %s · المعروضُ عليه %v · السائق %s", st, off, fx.Driver.ID)
		t.Fatalf("الأوّلُ (نقدُه %d) لم يُسنَد إلى السائق الوحيد", due1)
	}

	second, due2 := cashOrder(t, h, fx.Cust, 250_000, 1)
	if err := h.Orders.OfferNext(ctxBG(), second, nil); err != nil {
		t.Fatal(err)
	}
	var drv2 *string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT driver_id::text FROM orders WHERE id = $1`, second).Scan(&drv2); err != nil {
		t.Fatal(err)
	}
	held, inflight := exposureOf(t, h, fx.Driver.ID)
	if drv2 != nil {
		t.Fatalf("**أُسنِد الثاني (نقدُه %d) والتعرّضُ صار %d والسقفُ %d**", due2, held+inflight, fx.Limit)
	}
}
