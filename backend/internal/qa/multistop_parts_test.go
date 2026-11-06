package qa

// ══════════════════════════════════════════════════════════════════════
// **طلبٌ واحدٌ بعدّة محطّات — كلُّ متجرٍ يجيب عن قسمه** (الدفعة P2)
// ══════════════════════════════════════════════════════════════════════
//
// **الطلبُ «بانتظار» حتّى يجيب كلُّ متجر** — وآخرُ جوابٍ ينقله في المعاملة نفسِها.
// **وكلُّ متجرٍ يرى قسمَه وحدَه** · **ويجهز قسمُه وحدَه** · **ولكلّ متجرٍ تذكرتُه** ·
// **والإنزالُ بعد آخر المتاجر.**

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// msShop **متجران ومالكاهما وزبون** — والطلبُ أ ثمّ ب في السلّة (أ هو الأوّل).
type msShop struct {
	h *Harness
	// f **مصنعٌ واحدٌ للاختبار** — مصنعٌ ثانٍ يعيد الأرقامَ نفسَها فيصطدم الهاتف.
	f          *Factory
	mA, mB     *Merchant
	a, b       *Item
	tokA, tokB string
	cust       *User
	admin      *User
}

func newMSShop(t *testing.T, mode string) *msShop {
	t.Helper()
	h := New(t)
	treasury(t, h)
	h.OpenMultiStop()
	h.Setting("orders.max_sources", "2")
	h.Setting("orders.max_open_per_customer", "10")
	h.Setting("platform.orders_mode", `"`+mode+`"`)
	h.Setting("orders.auto_transfer", "false")
	h.Setting("orders.auto_accept_min", "0")
	h.Setting("orders.auto_dispatch", "true")
	h.Setting("orders.notify_merchant", "true")
	// **والطابورُ لا الإسنادُ المباشر** — سائقٌ على ورديّةٍ من اختبارٍ آخر لا يأخذ الطلبَ وحدَه.
	h.Setting("drivers.assignment_mode", `"queue"`)
	h.Setting("drivers.max_active_orders", "5")
	h.Setting("drivers.cash_limit", "9000000")
	f := h.Factory()
	mA := f.Merchant(Commission(10))
	mB := f.Merchant(Commission(20))
	return &msShop{
		h: h, f: f, mA: mA, mB: mB,
		a: h.NewItemPriced(mA, 1000, 1200), b: h.NewItemPriced(mB, 500, 600),
		tokA: h.TokenFor(mA.Owner.ID, "merchant"), tokB: h.TokenFor(mB.Owner.ID, "merchant"),
		cust: h.Customer(), admin: h.NewUser("admin"),
	}
}

// order **طلبٌ من المتجرين** — أ أوّلاً.
func (s *msShop) order(t *testing.T) string {
	t.Helper()
	r := s.h.POSTKey("/api/v1/orders", s.cust.Token, uniq("ms2"), twoStoreBody(s.a, s.b))
	if r.Code >= 400 {
		t.Fatalf("إنشاءُ طلبٍ من متجرين: %s", r)
	}
	id, _ := r.JSON()["id"].(string)
	if id == "" {
		t.Fatalf("لا معرّفَ في الردّ: %s", r)
	}
	return id
}

// answer **البابُ القديم** — ما يرسله تطبيقُ المتجر 0.11/0.12 وموقعُه.
func (s *msShop) answer(tok, oid string, body map[string]any) Res {
	return s.h.POST("/api/v1/merchant/orders/"+oid+"/transition", tok, body)
}

// mustAnswer **جوابٌ يُنتظر قبولُه.**
func (s *msShop) mustAnswer(t *testing.T, label, tok, oid string, body map[string]any) map[string]any {
	t.Helper()
	r := s.answer(tok, oid, body)
	if r.Code != http.StatusOK {
		t.Fatalf("%s: %s", label, r)
	}
	return r.JSON()
}

// partOf **قسمُ متجرٍ في الطلب كما في القاعدة.**
type msPart struct {
	Status, Role, Reason string
	Prep                 *int
	Ready, Sent          bool
}

func partOf(t *testing.T, h *Harness, oid, mid string) msPart {
	t.Helper()
	var p msPart
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT status, answered_role, reason, prep_minutes, ready_at IS NOT NULL, sent_at IS NOT NULL
		FROM order_parts WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, mid).
		Scan(&p.Status, &p.Role, &p.Reason, &p.Prep, &p.Ready, &p.Sent); err != nil {
		t.Fatalf("قراءةُ القسم: %v", err)
	}
	return p
}

// orderFlags **جهوزُ الطلب وإرسالُه للمتجر.**
func orderFlags(t *testing.T, h *Harness, oid string) (ready, sent bool) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT ready_at IS NOT NULL, sent_to_merchant_at IS NOT NULL FROM orders WHERE id = $1::uuid`,
		oid).Scan(&ready, &sent); err != nil {
		t.Fatalf("قراءةُ الطلب: %v", err)
	}
	return ready, sent
}

// listed **أفي القائمة هذا الطلب؟** — ويردّ حمولتَه.
func listed(r Res, oid string) map[string]any {
	list, _ := r.JSON()["orders"].([]any)
	for _, v := range list {
		if m, ok := v.(map[string]any); ok && m["id"] == oid {
			return m
		}
	}
	return nil
}

// waitStatusIs **ينتظر حالاً — والأفعالُ التلقائيّةُ في خيطٍ مستقلّ.**
func waitStatusIs(t *testing.T, h *Harness, oid, want string, within time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(within)
	for {
		got := h.statusOf(oid)
		if got == want || time.Now().After(deadline) {
			return got
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TestMultiStop_SecondStoreListsAndOpens **المتجرُ الثاني يرى الطلبَ ويفتحه** — `T2` (G-02).
//
// **كان المتجرُ الثاني أعمى**: لا يجده في قائمته ولا يفتحه (٤٠٤) — **والطلبُ ينتظر طبخَه.**
func TestMultiStop_SecondStoreListsAndOpens(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid := s.order(t)

	for _, c := range []struct {
		label string
		m     *Merchant
		tok   string
	}{{"المتجرُ الأوّل", s.mA, s.tokA}, {"المتجرُ الثاني", s.mB, s.tokB}} {
		r := s.h.GET("/api/v1/merchant/stores/"+c.m.ID+"/orders?with_active=true", c.tok)
		if r.Code != http.StatusOK {
			t.Fatalf("%s: قائمتُه رُدّت: %s", c.label, r)
		}
		v := listed(r, oid)
		if v == nil {
			t.Errorf("**%s لا يجد الطلبَ في قائمته**", c.label)
		} else if v["status"] != "pending" {
			t.Errorf("%s: حالُه في القائمة %v — يُنتظر pending (يرنّ)", c.label, v["status"])
		}
		if g := s.h.GET("/api/v1/merchant/orders/"+oid, c.tok); g.Code != http.StatusOK {
			t.Errorf("**%s لا يفتح الطلب**: %s", c.label, g)
		}
	}
	// **ومتجرٌ ليس في الطلب لا يراه.**
	other := s.f.Merchant()
	if g := s.h.GET("/api/v1/merchant/orders/"+oid, s.h.TokenFor(other.Owner.ID, "merchant")); g.Code != http.StatusNotFound {
		t.Errorf("**متجرٌ غريبٌ فتح الطلب**: %s", g)
	}
	// **ومعلَّقٌ لا يدخل سجلَّه بعد** — والسجلُّ للمنتهي.
	if r := s.h.GET("/api/v1/merchant/stores/"+s.mB.ID+"/orders?closed_only=true", s.tokB); listed(r, oid) != nil {
		t.Errorf("**الطلبُ المعلَّق في سجلّ المتجر الثاني**: %s", r)
	}
}

// TestMultiStop_EachStoreSeesOnlyOwnItemsAndMoney **كلُّ متجرٍ يرى قسمَه ومالَه وحدَه** — `T3` (G-03 · C10).
//
// **كان المتجرُ الأوّلُ يرى أصنافَ الثاني وأسعارَها ومالَ الطلب كلِّه** — في `REST` وفي البثّ.
// **وصار البثُّ لغرفة المتجر إشارةً بلا حمولة**، والحمولةُ من بابه مُسقَطةً على قسمه.
func TestMultiStop_EachStoreSeesOnlyOwnItemsAndMoney(t *testing.T) {
	s := newMSShop(t, "merchants")
	capA := s.h.Listen("merchant:" + s.mA.ID)
	capB := s.h.Listen("merchant:" + s.mB.ID)
	oid := s.order(t)

	check := func(label string, v map[string]any, own, other *Item, ownM, otherM *Merchant, sub, net int64) {
		t.Helper()
		if v == nil {
			t.Fatalf("%s: لا حمولة", label)
		}
		raw, _ := json.Marshal(v)
		txt := string(raw)
		for _, leak := range []string{other.Name, otherM.Name, otherM.ID, other.ID} {
			if strings.Contains(txt, leak) {
				t.Errorf("**%s: تسرّب ما ليس له** (%q):\n%s", label, leak, txt)
			}
		}
		items, _ := v["items"].([]any)
		if len(items) != 1 {
			t.Fatalf("%s: %d صنفاً — يُنتظر صنفُه وحدَه:\n%s", label, len(items), txt)
		}
		if it, _ := items[0].(map[string]any); it["name"] != own.Name {
			t.Errorf("%s: الصنف %v — يُنتظر %s", label, it["name"], own.Name)
		}
		if v["merchant_id"] != ownM.ID || v["merchant_name"] != ownM.Name {
			t.Errorf("%s: المتجر %v %v — يُنتظر متجرُه", label, v["merchant_id"], v["merchant_name"])
		}
		if num(v["items_count"]) != 1 || int64(num(v["subtotal"])) != sub || int64(num(v["merchant_net"])) != net {
			t.Errorf("%s: العدد %v المجموع %v الصافي %v — يُنتظر 1 و%d و%d",
				label, v["items_count"], v["subtotal"], v["merchant_net"], sub, net)
		}
	}
	detail := func(tok string) map[string]any {
		r := s.h.GET("/api/v1/merchant/orders/"+oid, tok)
		if r.Code != http.StatusOK {
			t.Fatalf("فتحُ الطلب: %s", r)
		}
		return r.JSON()
	}
	check("تفصيلُ المتجر الأوّل", detail(s.tokA), s.a, s.b, s.mA, s.mB, 1000, 900)
	check("تفصيلُ المتجر الثاني", detail(s.tokB), s.b, s.a, s.mB, s.mA, 500, 400)
	check("قائمةُ المتجر الثاني", listed(s.h.GET("/api/v1/merchant/stores/"+s.mB.ID+"/orders?with_active=true",
		s.tokB), oid), s.b, s.a, s.mB, s.mA, 500, 400)

	// **وردُّ الجواب حمولتُه هو** — لا الطلبُ كلُّه.
	check("ردُّ قبول المتجر الثاني", s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid,
		map[string]any{"to": "accepted"}), s.b, s.a, s.mB, s.mA, 500, 400)

	for _, c := range []struct {
		label string
		cap   *Capture
	}{{"غرفةُ المتجر الأوّل", capA}, {"غرفةُ المتجر الثاني", capB}} {
		evs := c.cap.Drain(1500 * time.Millisecond)
		if len(evs) == 0 {
			t.Errorf("**%s لم يصلها شيء** — والمتجرُ ينتظر إشارةً ليعيد القراءة", c.label)
		}
		for _, ev := range evs {
			if _, body := ev["order"]; body {
				raw, _ := json.Marshal(ev)
				t.Errorf("**%s وصلها جسمُ الطلب** — وفيه مالُ المتجرين:\n%s", c.label, raw)
			}
		}
	}
}

// TestMultiStop_OldMerchantCallsRouteToPart **البابُ القديمُ يجيب عن قسم صاحبه** — `T4` (G-04 · G-05).
//
// **كان قبولُ المتجر الأوّل يقبل الطلبَ كلَّه** — وطبخُ الثاني لم يُسأل عنه، **والثاني ٤٠٤.**
func TestMultiStop_OldMerchantCallsRouteToPart(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid := s.order(t)

	v := s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted", "prep_minutes": 12})
	if v["status"] != "preparing" {
		t.Errorf("حالُ الطلب للمتجر الأوّل %v — يُنتظر preparing (قبل قسمَه، وضعُ المتاجر)", v["status"])
	}
	if got := s.h.statusOf(oid); got != "pending" {
		t.Fatalf("**قبولُ المتجر الأوّل نقل الطلبَ كلَّه** إلى %q — والثاني لم يجب", got)
	}
	if p := partOf(t, s.h, oid, s.mA.ID); p.Status != "accepted" || p.Role != "merchant" || p.Prep == nil || *p.Prep != 12 {
		t.Errorf("قسمُ الأوّل %+v — يُنتظر accepted بيد المتجر ومدّة ١٢", p)
	}
	if p := partOf(t, s.h, oid, s.mB.ID); p.Status != "pending" {
		t.Errorf("قسمُ الثاني %+v — يُنتظر pending", p)
	}

	// «ابدأ التحضير» القديم بلا أثر · والرفضُ بلا سبب يُردّ.
	if r := s.answer(s.tokB, oid, map[string]any{"to": "preparing"}); r.Code != http.StatusOK || r.JSON()["status"] != "pending" {
		t.Errorf("«ابدأ التحضير» على قسمٍ معلَّق: %s — يُنتظر ٢٠٠ بلا أثر", r)
	}
	if r := s.answer(s.tokB, oid, map[string]any{"to": "rejected"}); r.Code != http.StatusBadRequest || r.Err() != "reason_required" {
		t.Errorf("رفضٌ بلا سبب: %s — يُنتظر ٤٠٠ reason_required", r)
	}

	// آخرُ الأجوبة ينقل الطلب — **في وضع المتاجر إلى التحضير فالطابور.**
	s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})
	if got := s.h.statusOf(oid); got != "dispatching" {
		t.Errorf("بعد آخر الأجوبة: %q — يُنتظر dispatching", got)
	}
	// **وإعادةُ الجواب نفسِه لا تكتب شيئاً.**
	if r := s.answer(s.tokB, oid, map[string]any{"to": "accepted"}); r.Code != http.StatusOK {
		t.Errorf("إعادةُ القبول: %s — يُنتظر ٢٠٠ بلا أثر", r)
	}
	var partEvents, answered int
	if err := s.h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FILTER (WHERE merchant_id IS NOT NULL),
		       count(*) FILTER (WHERE merchant_id IS NULL AND to_status = 'accepted' AND actor_id IS NULL)
		FROM order_events WHERE order_id = $1::uuid`, oid).Scan(&partEvents, &answered); err != nil {
		t.Fatal(err)
	}
	if partEvents != 2 || answered != 1 {
		t.Errorf("أحداثُ الأقسام %d وقبولُ النظام %d — يُنتظر ٢ و١", partEvents, answered)
	}
}

// TestMultiStop_DispatchWaitsForAllParts **لا سائقَ قبل أن يجيب كلُّ متجر** — `T5`.
//
// **كان قبولُ الأوّل (أو أوّلُ تذكرة) يُنزل الطلبَ** — فيقف السائقُ عند الثاني ولم يُطبخ شيء.
func TestMultiStop_DispatchWaitsForAllParts(t *testing.T) {
	t.Run("merchants", func(t *testing.T) {
		s := newMSShop(t, "merchants")
		oid := s.order(t)
		s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
		if got := s.h.statusOf(oid); got != "pending" {
			t.Fatalf("**نزل بعد قبول متجرٍ واحد**: %q", got)
		}
		s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})
		if got := s.h.statusOf(oid); got != "dispatching" {
			t.Errorf("بعد قبول المتجرين: %q — يُنتظر dispatching", got)
		}
	})
	t.Run("platform", func(t *testing.T) {
		s := newMSShop(t, "platform")
		oid := s.order(t)
		if r := s.h.POST("/api/v1/admin/orders/"+oid+"/transition", s.admin.Token,
			map[string]any{"to": "accepted"}); r.Code >= 400 {
			t.Fatalf("قبولُ المكتب: %s", r)
		}
		for _, m := range []*Merchant{s.mA, s.mB} {
			if p := partOf(t, s.h, oid, m.ID); p.Status != "accepted" || p.Role != "admin" {
				t.Errorf("**قبولُ المكتب لم يجب عن كلّ قسم**: %+v", p)
			}
		}
		send := func(mid string) {
			t.Helper()
			if r := s.h.POST("/api/v1/admin/orders/"+oid+"/whatsapp", s.admin.Token,
				map[string]any{"channel": "manual", "merchant_id": mid}); r.Code >= 400 {
				t.Fatalf("الإرسال: %s", r)
			}
		}
		send(s.mA.ID)
		if got := s.h.statusOf(oid); got != "accepted" {
			t.Fatalf("**نزل بعد تذكرة المتجر الأوّل**: %q — والثاني لم يعلم", got)
		}
		if _, sent := orderFlags(t, s.h, oid); sent {
			t.Errorf("**الطلبُ «أُبلغ» ومتجرٌ لم يُبلَّغ**")
		}
		send(s.mB.ID)
		if got := s.h.statusOf(oid); got != "dispatching" {
			t.Errorf("بعد التذكرتين: %q — يُنتظر dispatching", got)
		}
		if _, sent := orderFlags(t, s.h, oid); !sent {
			t.Errorf("أُبلغ المتجران والطلبُ لم يُوسَم")
		}
	})
}

// TestMultiStop_SweepsAnswerAllParts **الكانسُ يجيب عن كلّ قسمٍ معلَّق** — `T6` (G1).
//
// **كان الكانسُ يقبل الطلبَ كلَّه** — والقسمُ الذي لم يُجَب يبقى «بانتظار» تحت طلبٍ مقبول.
func TestMultiStop_SweepsAnswerAllParts(t *testing.T) {
	s := newMSShop(t, "merchants")
	s.h.Setting("orders.auto_accept_min", "1")
	oid := s.order(t)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted", "prep_minutes": 7})
	// **وبعد المهلة بثوانٍ لا بعشر دقائق** — الكانسُ يأخذ أحدثَ خمسين طلباً تجاوزت مهلتَها
	// (`sweepAutoAccept`)، **وفي التشغيل الكامل تبقى عشراتُ الطلبات «بانتظار» من اختباراتٍ
	// قبله أعمارُها بين دقيقةٍ وعشر** فيخرج طلبُ الاختبار من الخمسين. (قِيس: ٥٥ طلباً عمرُها
	// خمسُ دقائق ⇒ «الكانسُ لم يقبل الطلب».) **فطلبُ الاختبار أحدثُ ما تجاوز المهلة.**
	if _, err := s.h.Pool.Exec(ctxBG(),
		`UPDATE orders SET created_at = now() - interval '61 seconds' WHERE id = $1::uuid`, oid); err != nil {
		t.Fatal(err)
	}
	s.h.Orders.SweepAutoAcceptOnce(ctxBG())
	if got := s.h.statusOf(oid); got == "pending" {
		t.Fatalf("**الكانسُ لم يقبل الطلب**")
	}
	if p := partOf(t, s.h, oid, s.mB.ID); p.Status != "accepted" || p.Role != "system" || p.Prep == nil {
		t.Errorf("**قسمُ الثاني بعد الكانس** %+v — يُنتظر accepted بيد النظام بمدّة افتراضه", p)
	}
	if p := partOf(t, s.h, oid, s.mA.ID); p.Role != "merchant" || p.Prep == nil || *p.Prep != 7 {
		t.Errorf("**الكانسُ بدّل جوابَ الأوّل** %+v", p)
	}
}

// TestMultiStop_ReadyPerPart **«جاهز» لكلّ قسم — والطلبُ جاهزٌ حين تجهز كلُّها** — `T7` (G-06).
//
// **كان «جاهز» الأوّلِ يقول للسائق «جاهز» والثاني يطبخ.**
func TestMultiStop_ReadyPerPart(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid := s.order(t)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})

	ready := func(label, tok string) {
		t.Helper()
		if r := s.h.POST("/api/v1/merchant/orders/"+oid+"/ready", tok, nil); r.Code != http.StatusOK {
			t.Fatalf("%s: %s", label, r)
		}
	}
	// **وقسمٌ لم يُقبل لا يجهز.**
	if r := s.h.POST("/api/v1/merchant/orders/"+oid+"/ready", s.tokB, nil); r.Code != http.StatusConflict || r.Err() != "not_readyable" {
		t.Errorf("«جاهز» على قسمٍ معلَّق: %s — يُنتظر ٤٠٩ not_readyable", r)
	}
	ready("جهوزُ الأوّل", s.tokA)
	if !partOf(t, s.h, oid, s.mA.ID).Ready {
		t.Errorf("قسمُ الأوّل لم يُوسَم جاهزاً")
	}
	if r, _ := orderFlags(t, s.h, oid); r {
		t.Fatalf("**الطلبُ «جاهز» والثاني لم يقبل بعد**")
	}
	s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})
	if r, _ := orderFlags(t, s.h, oid); r {
		t.Fatalf("**الطلبُ «جاهز» والثاني يطبخ**")
	}
	ready("جهوزُ الثاني", s.tokB)
	if r, _ := orderFlags(t, s.h, oid); !r {
		t.Errorf("جهز القسمان والطلبُ لم يُوسَم جاهزاً")
	}
	ready("إعادةُ جهوز الأوّل", s.tokA)
}

// msNotifier **بوتٌ يسجّل ما أُرسل ولمن.**
type msNotifier struct {
	mu    sync.Mutex
	ready bool
	sent  []msTicket
}

type msTicket struct{ phone, text string }

func (n *msNotifier) Ready() bool { return n.ready }
func (n *msNotifier) SendText(_ context.Context, phone, text string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, msTicket{phone, text})
	return nil
}
func (n *msNotifier) tickets() []msTicket {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]msTicket(nil), n.sent...)
}

// TestMultiStop_WhatsAppTicketOwnItemsOwnPhone **لكلّ متجرٍ تذكرتُه — ببنوده وإلى هاتفه** — `T8` (G-07).
//
// **كانت التذكرةُ تذهب إلى هاتف الأوّل وفيها أصنافُ الاثنين** — والثاني لا يعلم.
func TestMultiStop_WhatsAppTicketOwnItemsOwnPhone(t *testing.T) {
	s := newMSShop(t, "platform")
	oid := s.order(t)
	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/transition", s.admin.Token,
		map[string]any{"to": "accepted"}); r.Code >= 400 {
		t.Fatalf("قبولُ المكتب: %s", r)
	}
	preview := func(q string) map[string]any {
		t.Helper()
		r := s.h.GET("/api/v1/admin/orders/"+oid+"/message"+q, s.admin.Token)
		if r.Code != http.StatusOK {
			t.Fatalf("المعاينة: %s", r)
		}
		return r.JSON()
	}
	for _, c := range []struct {
		label     string
		q         string
		own, othr *Item
		m         *Merchant
	}{
		{"تذكرةُ الثاني", "?merchant_id=" + s.mB.ID, s.b, s.a, s.mB},
		// **واللوحةُ القديمةُ بلا متجر** ⇒ أوّلُ قسمٍ لم يُبلَّغ.
		{"بلا متجر", "", s.a, s.b, s.mA},
	} {
		v := preview(c.q)
		text, _ := v["text"].(string)
		if !strings.Contains(text, c.own.Name) || strings.Contains(text, c.othr.Name) {
			t.Errorf("**%s: بنودٌ ليست له** — %q", c.label, text)
		}
		if v["phone"] != c.m.Owner.Phone || v["merchant_id"] != c.m.ID {
			t.Errorf("%s: الهاتف %v والمتجر %v — يُنتظر %s و%s", c.label, v["phone"], v["merchant_id"],
				c.m.Owner.Phone, c.m.ID)
		}
	}
	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/whatsapp", s.admin.Token,
		map[string]any{"channel": "manual", "merchant_id": s.mB.ID}); r.Code >= 400 || r.JSON()["merchant_id"] != s.mB.ID {
		t.Fatalf("إرسالُ تذكرة الثاني: %s", r)
	}
	if !partOf(t, s.h, oid, s.mB.ID).Sent || partOf(t, s.h, oid, s.mA.ID).Sent {
		t.Errorf("**وُسم غيرُ القسم الذي أُرسل**")
	}
}

// TestMultiStop_StoreReportsPerStore **تقريرُ كلّ متجرٍ بقسمه** — `T19` (G-14).
//
// **كان الطلبُ كلُّه يُعدّ تحت الأوّل** — مبيعُه بنودُ الاثنين وعمولتُه عمولتُهما، والثاني لم يبع شيئاً.
func TestMultiStop_StoreReportsPerStore(t *testing.T) {
	s := newMSShop(t, "merchants")
	base := financialBaseline(t, s.h)
	oid := s.order(t)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
	s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})
	deliverOrder(t, s.h, oid, s.h.driverOf(oid))

	for _, c := range []struct {
		label     string
		m         *Merchant
		tok       string
		sales, cm int64
	}{{"المتجرُ الأوّل", s.mA, s.tokA, 1000, 100}, {"المتجرُ الثاني", s.mB, s.tokB, 500, 100}} {
		r := s.h.GET("/api/v1/merchant/stores/"+c.m.ID+"/reports", c.tok)
		if r.Code != http.StatusOK {
			t.Fatalf("%s: التقرير: %s", c.label, r)
		}
		sum, _ := r.JSON()["summary"].(map[string]any)
		if num(sum["orders"]) != 1 || num(sum["delivered"]) != 1 || num(sum["sold"]) != 1 ||
			num(sum["sales"]) != c.sales || num(sum["platform_commission"]) != c.cm {
			t.Errorf("**%s: الملخّص %v** — يُنتظر طلبٌ مسلَّمٌ بمبيع %v وعمولة %v", c.label, sum, c.sales, c.cm)
		}
		var days int64
		dl, _ := r.JSON()["days"].([]any)
		for _, d := range dl {
			days += num(d.(map[string]any)["sales"])
		}
		if days != c.sales {
			t.Errorf("%s: مجموعُ الأيّام %v — يُنتظر %v كالملخّص", c.label, days, c.sales)
		}
	}
	assertNewViolations(t, s.h, base)
}

// TestMultiStop_PrivacyContract **الزبونُ لا يعرف من أين تُشترى بضاعتُه** — `T21` (P-1).
//
// **وخبرُ قسمٍ قُبل يصله بلا اسم متجر** — **وسجلُّه بلا أحداث الأقسام.**
func TestMultiStop_PrivacyContract(t *testing.T) {
	s := newMSShop(t, "merchants")
	capC := s.h.Listen("user:"+s.cust.ID, "customer:"+s.cust.ID)
	oid := s.order(t)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})

	leaks := []string{s.mA.Name, s.mB.Name, s.mA.ID, s.mB.ID}
	clean := func(label, txt string) {
		t.Helper()
		for _, l := range leaks {
			if strings.Contains(txt, l) {
				t.Errorf("**%s يحمل هويّةَ متجر** (%q):\n%s", label, l, txt)
			}
		}
	}
	partNotice := false
	for _, ev := range capC.Drain(1500 * time.Millisecond) {
		raw, _ := json.Marshal(ev)
		clean("بثُّ الزبون", string(raw))
		if n, ok := ev["notification"].(map[string]any); ok && n["title"] == "قسم من طلبك انقبل" {
			partNotice = true
		}
	}
	if !partNotice {
		t.Errorf("**لم يصل الزبونَ خبرُ القسم المقبول** — والطلبُ ما زال ينتظر غيرَه")
	}
	s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})

	r := s.h.GET("/api/v1/my/orders/"+oid, s.cust.Token)
	if r.Code != http.StatusOK {
		t.Fatalf("طلبُ الزبون: %s", r)
	}
	clean("طلبُ الزبون وسجلُّه", string(r.Body))
	var bodies string
	if err := s.h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(string_agg(title || ' ' || body, ' | '), '') FROM notifications
		WHERE user_id = $1::uuid AND entity_id = $2`, s.cust.ID, oid).Scan(&bodies); err != nil {
		t.Fatal(err)
	}
	clean("إشعاراتُ الزبون", bodies)
	// **ولا حقلَ جديدٌ في الحمولات** — عددُ المحطّات داخليّ.
	for _, tok := range []string{s.tokA, s.admin.Token} {
		path := "/api/v1/merchant/orders/" + oid
		if tok == s.admin.Token {
			path = "/api/v1/admin/orders/" + oid
		}
		if v := s.h.GET(path, tok).JSON(); v["stops"] != nil {
			t.Errorf("**حقلٌ غيرُ مصنَّف** stops في %s", path)
		}
	}
}

// TestMultiStop_DispatchAfterLastUnsentPart **الإنزالُ بعد آخر تذكرة — واللوحةُ القديمةُ تمرّ** — `T31` (D5).
//
// **كانت أوّلُ تذكرةٍ تُنزل الطلب.** **واللوحةُ القديمةُ ترسل بلا متجر** — فتذهب تذكرتُها
// إلى أوّل قسمٍ لم يُبلَّغ، ثمّ إلى الذي بعده. **وإعادةُ الإرسال بعد الإنزال لا تفكّ سائقاً.**
// (**وإزالةُ آخر قسمٍ لم يُبلَّغ تُنزله أيضاً** — مع `RemovePart` في الدفعة التالية.)
func TestMultiStop_DispatchAfterLastUnsentPart(t *testing.T) {
	s := newMSShop(t, "platform")
	oid := s.order(t)
	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/transition", s.admin.Token,
		map[string]any{"to": "accepted"}); r.Code >= 400 {
		t.Fatalf("قبولُ المكتب: %s", r)
	}
	send := func(body map[string]any) Res {
		t.Helper()
		r := s.h.POST("/api/v1/admin/orders/"+oid+"/whatsapp", s.admin.Token, body)
		if r.Code >= 400 {
			t.Fatalf("الإرسال: %s", r)
		}
		return r
	}
	if r := send(map[string]any{"channel": "manual"}); r.JSON()["merchant_id"] != s.mA.ID {
		t.Errorf("التذكرةُ الأولى بلا متجر ذهبت إلى %v — يُنتظر الأوّل", r.JSON()["merchant_id"])
	}
	if got := s.h.statusOf(oid); got != "accepted" {
		t.Fatalf("**نزل بعد أوّل تذكرة**: %q", got)
	}
	if r := send(map[string]any{"channel": "manual"}); r.JSON()["merchant_id"] != s.mB.ID {
		t.Errorf("التذكرةُ الثانية بلا متجر ذهبت إلى %v — يُنتظر الثاني", r.JSON()["merchant_id"])
	}
	if got := s.h.statusOf(oid); got != "dispatching" {
		t.Fatalf("بعد آخر تذكرة: %q — يُنتظر dispatching", got)
	}
	drv := s.f.Driver(OnShift())
	// **ولا يُترك سائقٌ على ورديّةٍ وبيده طلب** — وإلّا أُسند إليه طلبُ اختبارٍ تالٍ «بنفس المسار».
	t.Cleanup(func() {
		s.h.POST("/api/v1/admin/orders/"+oid+"/transition", s.admin.Token,
			map[string]any{"to": "cancelled", "note": "تنظيف اختبار"})
		_, _ = s.h.Pool.Exec(ctxBG(), `UPDATE users SET on_shift = false WHERE id = $1::uuid`, drv.ID)
	})
	if r := s.h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil); r.Code >= 400 {
		t.Fatalf("قبولُ السائق: %s", r)
	}
	send(map[string]any{"channel": "manual", "merchant_id": s.mA.ID})
	if got := s.h.statusOf(oid); got != "assigned" {
		t.Errorf("**إعادةُ الإرسال فكّت السائق**: %q", got)
	}
}

// TestMultiStop_AutoTransferNeedsEveryPhone **التحويلُ التلقائيُّ يسأل هاتفَ كلّ متجرٍ قبل القبول** — `T34` (D8 · C12).
//
// **كان يسأل هاتفَ الأوّل وحدَه** — فيُقبل الطلبُ وتذهب تذكرةٌ واحدةٌ ببنود الاثنين.
func TestMultiStop_AutoTransferNeedsEveryPhone(t *testing.T) {
	s := newMSShop(t, "platform")
	n := &msNotifier{ready: true}
	s.h.Setting("orders.auto_transfer", "true")
	s.h.API.SetMerchantNotifier(n)
	t.Cleanup(func() { s.h.API.SetMerchantNotifier(nil) })

	// ١ · متجرٌ ثانٍ بلا هاتف ⇒ الطلبُ كلُّه بيد المكتب.
	mC := s.f.Merchant()
	c := s.h.NewItemPriced(mC, 400, 500)
	if _, err := s.h.Pool.Exec(ctxBG(),
		`UPDATE merchants SET phone = '', owner_user_id = NULL WHERE id = $1::uuid`, mC.ID); err != nil {
		t.Fatal(err)
	}
	r := s.h.POSTKey("/api/v1/orders", s.cust.Token, uniq("ms2at"), twoStoreBody(s.a, c))
	if r.Code >= 400 {
		t.Fatalf("الإنشاء: %s", r)
	}
	noPhone, _ := r.JSON()["id"].(string)
	time.Sleep(1500 * time.Millisecond)
	if got := s.h.statusOf(noPhone); got != "pending" {
		t.Errorf("**قُبل طلبٌ متجرُه الثاني بلا هاتف**: %q", got)
	}
	if got := len(n.tickets()); got != 0 {
		t.Errorf("**أُرسلت %d تذكرة** ومتجرٌ لا يُبلَّغ", got)
	}

	// ٢ · المتجران بهاتفيهما ⇒ تذكرةٌ لكلٍّ ببنوده، والإنزالُ بعد الثانية.
	oid := s.order(t)
	if got := waitStatusIs(t, s.h, oid, "dispatching", 4*time.Second); got != "dispatching" {
		t.Fatalf("بعد التحويل التلقائيّ: %q — يُنتظر dispatching", got)
	}
	got := n.tickets()
	if len(got) != 2 {
		t.Fatalf("%d تذكرة — يُنتظر ٢", len(got))
	}
	for _, tk := range got {
		switch tk.phone {
		case s.mA.Owner.Phone:
			if !strings.Contains(tk.text, s.a.Name) || strings.Contains(tk.text, s.b.Name) {
				t.Errorf("**تذكرةُ الأوّل** %q", tk.text)
			}
		case s.mB.Owner.Phone:
			if !strings.Contains(tk.text, s.b.Name) || strings.Contains(tk.text, s.a.Name) {
				t.Errorf("**تذكرةُ الثاني** %q", tk.text)
			}
		default:
			t.Errorf("تذكرةٌ إلى هاتفٍ غريب %q", tk.phone)
		}
	}
	for _, m := range []*Merchant{s.mA, s.mB} {
		if !partOf(t, s.h, oid, m.ID).Sent {
			t.Errorf("قسمُ متجرٍ لم يُوسَم مُبلَّغاً")
		}
	}
}

// TestMultiStop_StoreBReportsAndComplaintsTarget **المتجرُ الثاني يُبلّغ — والشكوى لا تقع على الأوّل** — `T40` (C7).
func TestMultiStop_StoreBReportsAndComplaintsTarget(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid := s.order(t)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
	s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})
	deliverOrder(t, s.h, oid, s.h.driverOf(oid))

	if r := s.h.POST("/api/v1/merchant/orders/"+oid+"/report", s.tokB,
		map[string]any{"reason": "driver_late_pickup", "note": "تأخّر ربع ساعة"}); r.Code >= 400 {
		t.Errorf("**المتجرُ الثاني لا يُبلّغ عن سائق طلبه**: %s", r)
	}
	// **ومن سجلّه يُبلَّغ** — الطلبُ في قائمة «ما يُبلَّغ عنه» للمتجر الثاني قبل بلاغه؟ (بلّغ للتوّ).
	if r := s.h.POST("/api/v1/my/orders/"+oid+"/complaint", s.cust.Token,
		map[string]any{"reason": "wrong_items", "note": "صنفٌ غلط"}); r.Code >= 400 {
		t.Fatalf("شكوى الزبون: %s", r)
	}
	var against *string
	if err := s.h.Pool.QueryRow(ctxBG(), `
		SELECT against_user_id::text FROM tickets
		WHERE order_id = $1::uuid AND created_by = $2::uuid`, oid, s.cust.ID).Scan(&against); err != nil {
		t.Fatal(err)
	}
	if against != nil {
		t.Errorf("**شكوى «أصناف غلط» على طلبٍ بعدّة محطّات وقعت على %s** — يُنتظر للمكتب (فارغ)", *against)
	}
	for _, c := range []struct {
		label string
		m     *Merchant
		tok   string
	}{{"الأوّل", s.mA, s.tokA}, {"الثاني", s.mB, s.tokB}} {
		r := s.h.GET("/api/v1/merchant/stores/"+c.m.ID+"/reports-against", c.tok)
		if r.Code != http.StatusOK {
			t.Fatalf("الشكاوى على المتجر %s: %s", c.label, r)
		}
		if strings.Contains(string(r.Body), "wrong_items") {
			t.Errorf("**شكوى بلا متجرٍ محدَّدٍ تظهر للمتجر %s**", c.label)
		}
	}
	// **ومتى حدّد المكتبُ المتجرَ الثانيَ ظهرت له وحدَه.**
	if _, err := s.h.Pool.Exec(ctxBG(), `
		UPDATE tickets SET against_user_id = $3::uuid
		WHERE order_id = $1::uuid AND created_by = $2::uuid`, oid, s.cust.ID, s.mB.Owner.ID); err != nil {
		t.Fatal(err)
	}
	if r := s.h.GET("/api/v1/merchant/stores/"+s.mB.ID+"/reports-against", s.tokB); !strings.Contains(string(r.Body), "wrong_items") {
		t.Errorf("**الشكوى على المتجر الثاني لا تظهر له**: %s", r)
	}
	if r := s.h.GET("/api/v1/merchant/stores/"+s.mA.ID+"/reports-against", s.tokA); strings.Contains(string(r.Body), "wrong_items") {
		t.Errorf("**شكوى الثاني تظهر للأوّل**")
	}
}

// TestMultiStop_OfficePushOnStoreRejection **متجرٌ رفض قسمَه ⇒ المكتبُ يُخبَر** — `T42` (C13).
func TestMultiStop_OfficePushOnStoreRejection(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid := s.order(t)
	s.mustAnswer(t, "رفضُ الثاني", s.tokB, oid, map[string]any{"to": "rejected", "note": "نفد الصنف"})
	var title, body string
	deadline := time.Now().Add(2 * time.Second)
	for {
		err := s.h.Pool.QueryRow(ctxBG(), `
			SELECT title, body FROM notifications
			WHERE user_id = $1::uuid AND entity_id = $2 AND title = 'متجر رفض قسمه من طلب'`,
			s.admin.ID, oid).Scan(&title, &body)
		if err == nil || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if title == "" {
		t.Fatalf("**لم يُخبَر المكتبُ برفض المتجر لقسمه**")
	}
	if !strings.Contains(body, s.mB.Name) || !strings.Contains(body, "نفد الصنف") {
		t.Errorf("نصُّ إشعار المكتب %q — يُنتظر اسمُ المتجر وسببُه", body)
	}
	// **والقسمُ المرفوضُ في سجلّ المتجر «مرفوض»** والطلبُ ما زال ينتظر الأوّل.
	r := s.h.GET("/api/v1/merchant/stores/"+s.mB.ID+"/orders?closed_only=true", s.tokB)
	if v := listed(r, oid); v == nil || v["status"] != "rejected" || num(v["subtotal"]) != 0 {
		t.Errorf("**القسمُ المرفوضُ في سجلّ الثاني** %v — يُنتظر rejected بمالٍ صفر", v)
	}
	if got := s.h.statusOf(oid); got != "pending" {
		t.Errorf("الطلبُ بعد رفض الثاني: %q — يُنتظر pending", got)
	}
}

// TestMultiStop_OrderEditTellsEachStoreItsOwnChanges **تعديلُ الطلب: كلُّ متجرٍ يُخبَر بما تغيّر عنده** — `G-24`.
//
// **كان كلُّ متجرٍ في الطلب يُخبَر بكلّ ما تغيّر** — أصنافِ غيره وأسعارِها.
func TestMultiStop_OrderEditTellsEachStoreItsOwnChanges(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid := s.order(t)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
	s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})
	var lineB string
	if err := s.h.Pool.QueryRow(ctxBG(), `
		SELECT id::text FROM order_items WHERE order_id = $1::uuid AND merchant_id = $2::uuid`,
		oid, s.mB.ID).Scan(&lineB); err != nil {
		t.Fatal(err)
	}
	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/edit", s.admin.Token, map[string]any{
		"reason": "الزبون زاد", "lines": []map[string]any{{"item_id": lineB, "qty": 2}},
	}); r.Code >= 400 {
		t.Fatalf("تعديلُ الطلب: %s", r)
	}
	edited := func(owner string) []string {
		t.Helper()
		rows, err := s.h.Pool.Query(ctxBG(), `
			SELECT body FROM notifications
			WHERE user_id = $1::uuid AND entity_id = $2 AND title = 'تعدّل طلب من متجرك'`, owner, oid)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var b string
			if err := rows.Scan(&b); err != nil {
				t.Fatal(err)
			}
			out = append(out, b)
		}
		return out
	}
	if got := edited(s.mA.Owner.ID); len(got) != 0 {
		t.Errorf("**المتجرُ الأوّل أُخبر بتعديل قسم الثاني**: %q", got)
	}
	got := edited(s.mB.Owner.ID)
	if len(got) != 1 || !strings.Contains(got[0], s.b.Name) || strings.Contains(got[0], s.a.Name) {
		t.Errorf("إشعارُ الثاني %q — يُنتظر واحدٌ بصنفه وحدَه", got)
	}
}
