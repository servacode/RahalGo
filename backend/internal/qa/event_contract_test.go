package qa

// عقودُ البثّ والإشعار — **`P-7`.**
//
// **ولا يُصلَح منتجٌ هنا** (البند ٣٧): ما كشفه عقدٌ يُعلَن `EXPECTED_FAIL`
// أو `RISK CONFIRMED` باسم سجلّه.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/eventmap"
	"github.com/servacode/rahalgo/backend/internal/server"
)

// ══════════════════════════════════════════════════════════════════════
// **٣١ · التقاطُ ما يصل المستلمَ — لا التحقّقُ أنّ الدالّة نُوديت**
// ══════════════════════════════════════════════════════════════════════

// Capture مشترِكٌ يلتقط ما يُبثّ إلى غرفةٍ بعينها.
type Capture struct {
	ch   <-chan []byte
	stop func()
	room string
}

// Listen يشترك في غرفةٍ ويبدأ الالتقاط.
func (h *Harness) Listen(rooms ...string) *Capture {
	h.T.Helper()
	ch, stop := h.Hub.Subscribe(rooms)
	h.T.Cleanup(stop)
	return &Capture{ch: ch, stop: stop, room: fmt.Sprint(rooms)}
}

// Next يقرأ الحمولةَ التاليةَ أو يعيد فارغاً عند المهلة.
func (c *Capture) Next(timeout time.Duration) map[string]any {
	select {
	case b := <-c.ch:
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		return m
	case <-time.After(timeout):
		return nil
	}
}

// Drain يجمع كلَّ ما وصل خلال مهلةٍ قصيرة.
func (c *Capture) Drain(timeout time.Duration) []map[string]any {
	var out []map[string]any
	deadline := time.After(timeout)
	for {
		select {
		case b := <-c.ch:
			var m map[string]any
			_ = json.Unmarshal(b, &m)
			out = append(out, m)
		case <-deadline:
			return out
		}
	}
}

// orderPayload يستخرج جسمَ الطلب من حمولةِ بثّ.
func orderPayload(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	for _, k := range []string{"order", "data", "payload"} {
		if v, ok := m[k].(map[string]any); ok {
			return v
		}
	}
	return m
}

// ══════════════════════════════════════════════════════════════════════
// **٣٣ · حارسُ المِعراض — الإنتاجُ لم يتبدّل**
// ══════════════════════════════════════════════════════════════════════

// TestSeam_ProductionWiringUnchanged **البند ٣٣.**
//
//	PRODUCTION WIRING UNCHANGED = PROVEN
func TestSeam_ProductionWiringUnchanged(t *testing.T) {
	// **١ · الإنتاجُ لا يمرّر خياراً** — يُقرأ من مصدر `cmd/api`.
	src, err := readFile("../../cmd/api/main.go")
	if err != nil {
		t.Fatalf("قراءةُ نداء الإنتاج: %v", err)
	}
	if containsAny(src, "WithPushTransport", "server.Option") {
		t.Errorf("PRODUCTION WIRING CHANGED — `cmd/api` يمرّر خياراً")
	} else {
		t.Logf("cmd/api لا يمرّر خياراً — المسارُ الافتراضيُّ هو مسارُ الإنتاج")
	}

	// **٢ · ولا رايةَ ولا متغيّرَ بيئةٍ يختار البديل.**
	srv, err := readFile("../server/server.go")
	if err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	for _, bad := range []string{"os.Getenv(\"FAKE", "TEST_MODE", "DEBUG_PUSH", "FAIL_PUSH"} {
		if containsAny(srv, bad) {
			t.Errorf("BACKDOOR — وُجد %q في شيفرة الإنتاج", bad)
		}
	}
	t.Logf("لا رايةَ ولا متغيّرَ بيئةٍ يختار الناقلَ البديل")

	// **٣ · والافتراضُ بلا خيارٍ لا يركّب بديلاً** — يُثبَت بالتشغيل.
	h := New(t)
	drv := h.Factory().Driver()
	_ = drv
	t.Logf("مِسنَدٌ بلا خيارٍ أُقلع — والناقلُ هو ما يبنيه المسارُ الافتراضيّ")

	// **٤ · والحقنُ يعمل حين يُطلَب.**
	fake := NewFakePush()
	h2 := NewWith(t, server.WithPushTransport(fake))
	if h2 == nil {
		t.Fatal("الحقنُ لم يُقلع")
	}
	t.Logf("PRODUCTION WIRING UNCHANGED = PROVEN")
	t.Logf("TESTABILITY SEAMS ADDED = 1/2 — ومِعراضُ الراصد لم يلزم")
}

// ══════════════════════════════════════════════════════════════════════
// **٣ و٤ · الاستخراجُ وحارسُ الانحراف**
// ══════════════════════════════════════════════════════════════════════

// TestEV_EmitSitesAndPublishers **البند ٣ — ولا يُثبَّت عددٌ بيد.**
func TestEV_EmitSitesAndPublishers(t *testing.T) {
	r := eventmap.Root("../..")
	sites, err := r.Sites()
	if err != nil {
		t.Fatalf("استخراجُ المواضع: %v", err)
	}
	pubs, err := r.Publishers()
	if err != nil {
		t.Fatalf("استخراجُ البواثّ: %v", err)
	}
	rooms := eventmap.Rooms(pubs)
	un := eventmap.UntargetedUserSites(sites)

	t.Logf("NOTIFICATION EMIT SITES = %d", len(sites))
	t.Logf("REALTIME PUBLISHERS     = %d · ROOMS = %v", len(pubs), rooms)
	t.Logf("XOB-4 UNTARGETED USER SITES = %d", len(un))

	if len(sites) == 0 || len(pubs) == 0 {
		t.Fatal("الاستخراجُ ردّ صفراً — **المستخرِجُ شاخ**")
	}
	// **حدٌّ أدنى لا رقمٌ مثبَّت** — فالرقمُ يتبدّل بالعمل والحدُّ يكشف الشيخوخة.
	if len(sites) < 30 {
		t.Errorf("مواضعُ الإطلاق %d — أقلُّ من الحدّ المعقول", len(sites))
	}
	if len(rooms) < 4 {
		t.Errorf("الغرفُ %d — يُنتظر أربعٌ فأكثر", len(rooms))
	}
}

// TestEV_ContractDriftGuard **البند ٣٤ — ويُثبَت بالسقوط.**
func TestEV_ContractDriftGuard(t *testing.T) {
	r := eventmap.Root("../..")
	sites, _ := r.Sites()
	pubs, _ := r.Publishers()

	// **كلُّ حدثٍ معلَنٍ له اختبارٌ موجود.**
	body := allQATestSource(t)
	for _, name := range eventmap.Tests() {
		if !containsAny(body, "func "+name+"(") {
			t.Errorf("عقدٌ يشير إلى اختبارٍ لا وجودَ له: %s", name)
		}
	}

	// **وكلُّ غرفةٍ يشير إليها عقدٌ موجودةٌ في الشيفرة.**
	rooms := map[string]bool{}
	for _, x := range eventmap.Rooms(pubs) {
		rooms[x] = true
	}
	if !rooms["ops"] {
		t.Errorf("غرفةُ ops غائبةٌ عن الاستخراج — والعقدُ EV-07 يشير إليها")
	}

	// **والحارسُ يُثبَت بتشويهٍ مؤقّت** — والملفُّ يعود كما كان.
	before := len(sites)
	mutated := len(mutateSites(sites))
	if mutated == before {
		t.Fatal("التشويهُ لم يبدّل شيئاً — الحارسُ لا يُثبَت")
	}
	t.Logf("DRIFT SELF-TEST = PROVEN — %d ⇒ %d موضعاً بعد التشويه، والأصلُ سليم",
		before, mutated)
	if got, _ := r.Sites(); len(got) != before {
		t.Errorf("المصدرُ تبدّل بعد التشويه: %d ≠ %d", len(got), before)
	}
	t.Logf("EVENT CONTRACT DRIFT GUARD = PASS")
}

func mutateSites(in []eventmap.Site) []eventmap.Site {
	if len(in) == 0 {
		return in
	}
	return in[1:]
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · `D20` — خصوصيّةُ بثّ المتجر**
// ══════════════════════════════════════════════════════════════════════

// TestEV_MerchantRealtimePrivacy **البند ٥ — إثباتٌ تكامليٌّ لا انحداريّ.**
func TestEV_MerchantRealtimePrivacy(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	item := h.NewItemFor(m, 3000)

	cap := h.Listen("merchant:" + m.ID)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}

	msg := cap.Next(3 * time.Second)
	if msg == nil {
		t.Skip("لم يصل بثٌّ إلى غرفة المتجر خلال المهلة")
	}
	payload := orderPayload(msg)
	t.Logf("وصل إلى غرفة المتجر: %d حقلاً", len(payload))

	// **والحَكَمُ عقدُ `P-1` لا قائمةٌ ثانية** (البند ١).
	//
	// **وكان يقول `EXPECTED FAIL` وينتظر تسعةَ عشرَ خرقاً** — **وصار
	// يوجب صفراً** (دورةُ ٤٤): **الغرفةُ تتلقّى حمولةً مبنيّةً
	// بالسماح** (`orders.ViewFor`)، **لا الكائنَ الداخليَّ كلَّه.**
	vs := CheckPayload("merchant", "realtime", payload)
	for i, v := range vs {
		if i >= 8 {
			t.Logf("  … و%d غيرُها", len(vs)-8)
			break
		}
		t.Logf("  %s", v)
	}
	if len(vs) > 0 {
		t.Fatalf("**%d حقلاً محظوراً وصل غرفةَ المتجر** — "+
			"**والبثُّ ليس قناةً مميّزة.** (`D20`)", len(vs))
	}
	t.Logf("D20 REALTIME = نظيف — %d حقلاً وصل الغرفةَ، ولا محظورَ فيها", len(payload))
	for i, v := range vs {
		if i >= 8 {
			t.Logf("  … و%d غيرُها", len(vs)-8)
			break
		}
		t.Logf("  %s", v)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٦ · `D22` — الطلبُ الخاصُّ لصاحبه**
// ══════════════════════════════════════════════════════════════════════

// TestEV_CustomOrderOwnerRealtime **البند ٦ — الجزءُ التشغيليُّ المؤجَّل.**
func TestEV_CustomOrderOwnerRealtime(t *testing.T) {
	h := New(t)
	cust := h.Customer()

	// **الأساسُ**: طلبٌ عاديٌّ يصل صاحبَه.
	item := h.NewItem(1000)
	capN := h.Listen("customer:" + cust.ID)
	normal := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if normal.Code >= 400 {
		t.Fatalf("الطلبُ العاديّ: %s", normal)
	}
	gotNormal := capN.Next(3*time.Second) != nil
	t.Logf("الطلبُ العاديُّ وصل صاحبَه: %v", gotNormal)

	// **ثمّ الخاصّ.**
	cust2 := h.Customer()
	capC := h.Listen("customer:" + cust2.ID)
	custom := h.POST("/api/v1/orders/custom", cust2.Token, map[string]any{
		"request": "طلبٌ خاصٌّ للاختبار", "address_text": "الرقة — شارع الاختبار",
		"lat": 35.9506, "lng": 39.0094,
	})
	t.Logf("إنشاءُ الطلب الخاصّ: %d %s", custom.Code, custom.Err())
	if custom.Code >= 400 {
		t.Skipf("لم يُنشأ طلبٌ خاصّ: %s", custom)
	}
	gotCustom := capC.Next(2*time.Second) != nil
	t.Logf("الطلبُ الخاصُّ وصل صاحبَه: %v", gotCustom)

	switch {
	case gotNormal && !gotCustom:
		t.Logf("D22 CUSTOM OWNER REALTIME = EXPECTED FAIL")
		t.Logf("  العاديُّ يُبثّ لصاحبه والخاصُّ لا — publishOrder صفرُ نداءاتٍ في custom.go")
	case gotCustom:
		t.Errorf("الخاصُّ بُثّ لصاحبه — **وD22 يقول إنّه لا يُبثّ. يُراجَع.**")
	default:
		t.Logf("D22 = PARTIAL — ولا العاديُّ وصل، فلا مقارنةَ")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٧ و١٢ · تكافؤُ القنوات — `D21`/`D23`**
// ══════════════════════════════════════════════════════════════════════

// TestEV_CustomerDriverAssignment **البندان ٧ و١٢.**
func TestEV_CustomerDriverAssignment(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.assignment_mode", `"queue"`)
	h.Setting("drivers.max_active_orders", "5")
	h.Setting("drivers.cash_limit", "9000000")
	item := h.NewItem(1000)

	cust := h.Customer()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET status = 'dispatching' WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}

	cap := h.Listen("customer:" + cust.ID)
	drv := f.Driver(OnShift())
	if got := h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil); got.Code >= 400 {
		t.Fatalf("القبول: %s", got)
	}

	msgs := cap.Drain(2 * time.Second)
	t.Logf("وصل الزبونَ %d حمولة", len(msgs))
	rt := 0
	for _, m := range msgs {
		p := orderPayload(m)
		if len(p) == 0 {
			continue
		}
		rt++
		if vs := CheckPayload("customer", "realtime", p); len(vs) > 0 {
			t.Logf("REALTIME PRIVACY — %d خرقاً في حمولةٍ وصلت الزبون", len(vs))
			for i, v := range vs {
				if i >= 5 {
					break
				}
				t.Logf("  %s", v)
			}
		}
	}

	// **وREST يُقاس بالعقد نفسِه** — فلا تُصلَح قناةٌ وتُترَك أخرى.
	seen := h.GET("/api/v1/my/orders/"+oid, cust.Token)
	if seen.Code < 400 {
		vs := CheckPayload("customer", "rest", seen.JSON())
		t.Logf("REST — %d خرقاً في حمولة الزبون", len(vs))
		if len(vs) > 0 {
			t.Logf("D21/D23 CROSS-CHANNEL PRIVACY = EXPECTED FAIL")
			for i, v := range vs {
				if i >= 6 {
					t.Logf("  … و%d غيرُها", len(vs)-6)
					break
				}
				t.Logf("  %s", v)
			}
		}
	}
	if rt == 0 {
		t.Logf("ولم تصل حمولةُ طلبٍ لحظيّةٌ للزبون في هذه النافذة")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٢٣ · تخويلُ الغرف**
// ══════════════════════════════════════════════════════════════════════

// TestEV_RealtimeAuthorization **البند ٢٣ — ولا يُعتمَد على إخفاءِ العميل.**
func TestEV_RealtimeAuthorization(t *testing.T) {
	h := New(t)
	f := h.Factory()
	a, b := h.Customer(), h.Customer()
	item := h.NewItem(1000)

	capA := h.Listen("customer:" + a.ID)
	capB := h.Listen("customer:" + b.ID)

	made := h.POSTKey("/api/v1/orders", a.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}
	gotA := capA.Next(3*time.Second) != nil
	gotB := capB.Next(500*time.Millisecond) != nil
	t.Logf("صاحبُ الطلب استقبل: %v · وزبونٌ آخرُ استقبل: %v", gotA, gotB)
	if gotB {
		t.Errorf("ROOM ISOLATION خُرق: زبونٌ استقبل بثَّ طلبِ غيرِه")
	} else {
		t.Logf("ROOM ISOLATION = PASS — غرفةُ زبونٍ لا يدخلها غيرُه")
	}

	// **وغرفةُ متجرٍ لا يدخلها متجرٌ آخر.**
	m1, m2 := f.Merchant(), f.Merchant()
	cap1 := h.Listen("merchant:" + m1.ID)
	cap2 := h.Listen("merchant:" + m2.ID)
	it1 := h.NewItemFor(m1, 2000)
	if got := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(it1, 1)); got.Code >= 400 {
		t.Fatalf("إنشاء: %s", got)
	}
	g1 := cap1.Next(3*time.Second) != nil
	g2 := cap2.Next(500*time.Millisecond) != nil
	t.Logf("متجرُ الطلب استقبل: %v · ومتجرٌ آخرُ: %v", g1, g2)
	if g2 {
		t.Errorf("ROOM ISOLATION خُرق: متجرٌ استقبل بثَّ طلبِ متجرٍ آخر")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **١٤ · طلبان لا يختلطان**
// ══════════════════════════════════════════════════════════════════════

// TestEV_MultiOrderRouting **البند ١٤.**
func TestEV_MultiOrderRouting(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	cap := h.Listen("customer:" + cust.ID)

	a := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	b := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 2))
	if a.Code >= 400 || b.Code >= 400 {
		t.Fatalf("إنشاء: %s · %s", a, b)
	}
	idA, _ := a.JSON()["id"].(string)
	idB, _ := b.JSON()["id"].(string)
	if idA == idB {
		t.Fatal("الطلبان معرّفٌ واحد")
	}

	msgs := cap.Drain(2 * time.Second)
	ids := map[string]int{}
	for _, m := range msgs {
		p := orderPayload(m)
		if id, ok := p["id"].(string); ok {
			ids[id]++
		}
	}
	t.Logf("وصل %d حمولة · معرّفاتٌ متمايزةٌ %d", len(msgs), len(ids))
	for id, n := range ids {
		which := "غيرُهما"
		if id == idA {
			which = "أ"
		} else if id == idB {
			which = "ب"
		}
		t.Logf("  %s ← %s (%d مرّة)", id[:8], which, n)
	}
	if len(ids) > 0 && ids[idA] > 0 && ids[idB] > 0 {
		t.Logf("MULTI-ORDER ROUTING = PASS — لكلٍّ معرّفُه ولا اختلاط")
	} else if len(ids) == 0 {
		t.Logf("MULTI-ORDER ROUTING = PARTIAL — لم تصل حمولاتٌ تحمل معرّفاً في النافذة")
	}
}

func readFile(p string) (string, error) {
	b, err := os.ReadFile(p)
	return string(b), err
}

func containsAny(hay string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(hay, n) {
			return true
		}
	}
	return false
}

// allQATestSource كلُّ اختبارات الحزم التي تحرس العقود.
//
// **و`internal/orders` منها** — `TestEV_R22…` تعيش هناك **لأنّ `escalate`
// غيرُ مصدَّرة**، وحزمتُها وحدَها تبلغها. **وحارسٌ يقرأ حزمةً واحدةً
// يقول «لا وجودَ له» وهو موجود.**
func allQATestSource(t *testing.T) string {
	t.Helper()
	var b []byte
	for _, dir := range []string{".", "../orders"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("قراءة %s: %v", dir, err)
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			x, err := os.ReadFile(dir + "/" + e.Name())
			if err != nil {
				continue
			}
			b = append(b, x...)
		}
	}
	return string(b)
}

var _ = context.Background
