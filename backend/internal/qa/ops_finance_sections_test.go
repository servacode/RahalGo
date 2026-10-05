package qa

import (
	"net/http"
	"sort"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **العمليّاتُ بلا عائقٍ في أقسامها التسعة · والماليّةُ في أقسامها الخمسة**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارا المالك ٢٠٢٦-١٠-٠٥ — هجرة `0422`.)
//
// # أ · العمليّات
//
// **بلاغُ المالك من التجهيز**: «موظّفُ العمليّات ما عنده صلاحيّة يحوّل الطلب
// للمتجر». **والجدولُ يقول إنّ التحويلَ بـ`orders.intervene` وهي له** — فالعطبُ
// في نداءٍ آخرَ يسبقه: زرُّ «حوّل للمتجر» يقرأ أوّلاً رسالةَ المنصّة إلى المتجر
// (`GET /orders/{id}/message`) **وكانت بـ`orders.communications.read`.** وصفحةُ
// قسم السوق تُضيف الصنفَ وتعدّله وتُطفئه ببابَي إدارة المتاجر (`merchants.manage`).
//
// # ب · الماليّة
//
// **ترى خمسةَ أقسامٍ لا غير**: الخزينة · الديون · الخسائر والنزاعات · الأهداف
// والمكافآت · العروض والخصومات. **ولا الطلبات ولا سجلّها ولا الحسابات** — ولا
// التقارير ولا سجلّ الأحداث ولا الإعدادات. **وكلُّ زرٍّ في أقسامها يعمل لها.**
//
// **والحكمُ على منع الجدول وحدَه** — `403` برمز `forbidden`. **و`400`/`404`/`409`
// نجاحُ تخويلٍ وردُّ منطق**، ومنعُ المنطق (`self_approve` مثلاً) ليس منعَ قسم.

// secCall نداءٌ تمشيه الصفحةُ — بفعله ومساره وجسمه.
type secCall struct {
	method, path string
	body         map[string]any
}

// do **ينادي ويُعيد الردّ.**
func (c secCall) do(hh *Harness, tok string) Res {
	switch c.method {
	case "GET":
		return hh.GET(c.path, tok)
	case "PATCH":
		return hh.PATCH(c.path, tok, c.body)
	default:
		return hh.Call(c.method, c.path, tok, c.body, nil)
	}
}

// capDenied **أمنعه جدولُ القدرات؟** — لا منعُ منطقٍ داخل المعالِج.
func capDenied(r Res) bool {
	return r.Code == http.StatusUnauthorized ||
		(r.Code == http.StatusForbidden && r.Err() == "forbidden")
}

// walkCalls **يمشي النداءاتِ بدورٍ ويحكم** — المسموحُ لا يُمنَع، والممنوعُ يُمنَع.
func walkCalls(t *testing.T, hh *Harness, role, tok string, allowed, denied []secCall) {
	t.Helper()
	for _, c := range allowed {
		if r := c.do(hh, tok); capDenied(r) {
			t.Errorf("**زرٌّ في أقسام `%s` يُردّ**: %s %s ← %s", role, c.method, c.path, r)
		}
	}
	for _, c := range denied {
		if r := c.do(hh, tok); !capDenied(r) {
			t.Errorf("**`%s` بلغ ما ليس له**: %s %s ← %d (أُريد ٤٠٣)", role, c.method, c.path, r.Code)
		}
	}
	t.Logf("✓ %s: مسموحٌ=%d · ممنوعٌ=%d", role, len(allowed), len(denied))
}

// placedOrder **طلبٌ حقيقيٌّ قبله المكتب** — بموظّف العمليّات نفسِه.
func placedOrder(t *testing.T, hh *Harness, staffTok string) (oid string, item *Item) {
	t.Helper()
	item = hh.NewItem(1000)
	cust := hh.NewUser("customer")
	made := hh.POSTKey("/api/v1/orders", cust.Token, uniq("opsfwd"), orderBody(item, 1))
	if made.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ = made.JSON()["id"].(string)
	if r := hh.POST("/api/v1/admin/orders/"+oid+"/transition", staffTok,
		map[string]any{"to": "accepted", "note": "قبولٌ من العمليّات"}); r.Code >= 400 {
		t.Fatalf("قبولُ الطلب بيد العمليّات: %s", r)
	}
	return oid, item
}

// TestOpsSections_ForwardToStoreAndTransfer **بلاغُ المالك — العمليّاتُ تحوّل
// الطلبَ للمتجر ثمّ تبدّل المتجر.** كان يسقط عند نداء الرسالة بـ٤٠٣.
func TestOpsSections_ForwardToStoreAndTransfer(t *testing.T) {
	hh := New(t)
	hh.Setting("customers.require_whatsapp", "false")
	treasury(t, hh)
	staff := hh.NewUser("operations")
	oid, item := placedOrder(t, hh, staff.Token)
	const a = "/api/v1/admin/orders/"

	// ── ١ · «حوّل للمتجر» — رسالةُ المنصّة ثمّ وسمُ الإرسال ─────────────
	msg := hh.GET(a+oid+"/message", staff.Token)
	if msg.Code != http.StatusOK {
		t.Fatalf("**موظّفُ العمليّات لا يقرأ رسالةَ التحويل للمتجر** — %s\n"+
			"(بلاغُ المالك ٢٠٢٦-١٠-٠٥: «ما عنده صلاحيّة يحوّل الطلب للمتجر»)", msg)
	}
	if link, _ := msg.JSON()["wa_link"].(string); link == "" {
		t.Errorf("رسالةُ التحويل بلا رابط واتساب: %s", msg)
	}
	if r := hh.POST(a+oid+"/whatsapp", staff.Token,
		map[string]any{"channel": "whatsapp"}); r.Code >= 400 {
		t.Fatalf("**وسمُ «أُرسل للمتجر» رُدّ** — %s", r)
	}

	// ── ٢ · «تبديل المتجر» — مرشّحون ثمّ تحويلٌ إلى متجرٍ يبيع الصنفَ نفسَه ──
	other := hh.NewItem(1000)
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE menu_items SET name = $2 WHERE id = $1::uuid`, other.ID, item.Name); err != nil {
		t.Fatalf("تسميةُ صنف المتجر الثاني: %v", err)
	}
	if r := hh.GET(a+oid+"/transfer-candidates", staff.Token); r.Code >= 400 {
		t.Fatalf("**مرشّحو التحويل رُدّوا للعمليّات** — %s", r)
	}
	if r := hh.POST(a+oid+"/transfer", staff.Token, map[string]any{
		"merchant_id": other.MerchantID, "note": "المتجر اعتذر — تحويل",
	}); r.Code >= 400 {
		t.Fatalf("**تحويلُ الطلب لمتجرٍ آخر رُدّ للعمليّات** — %s", r)
	}
	var merchant string
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT merchant_id::text FROM orders WHERE id = $1::uuid`, oid).Scan(&merchant); err != nil {
		t.Fatalf("قراءةُ متجر الطلب: %v", err)
	}
	if merchant != other.MerchantID {
		t.Fatalf("رُدّ نجاحاً والطلبُ عند %s لا %s — **نجاحٌ لا يكتب**", merchant, other.MerchantID)
	}
}

// TestOpsSections_EveryButtonWorks **كلُّ زرٍّ في الأقسام التسعة يُفتح للعمليّات** —
// وما هو للماليّة أو للمدير يُردّ.
func TestOpsSections_EveryButtonWorks(t *testing.T) {
	hh := New(t)
	hh.Setting("customers.require_whatsapp", "false")
	treasury(t, hh)
	staff := hh.NewUser("operations")
	oid, item := placedOrder(t, hh, staff.Token)
	drv := hh.Factory().Driver(OnShift())
	const a = "/api/v1/admin"
	o := a + "/orders/" + oid
	none := "00000000-0000-0000-0000-000000000001"

	allowed := []secCall{
		// ── الطلبات وسجلُّها ──
		{"GET", a + "/orders?limit=1", nil}, {"GET", a + "/orders/board", nil},
		{"GET", a + "/orders/alerts", nil}, {"GET", o, nil},
		{"GET", o + "/message", nil}, {"POST", o + "/whatsapp", map[string]any{"channel": "whatsapp"}},
		{"GET", o + "/assign-candidates", nil}, {"GET", o + "/transfer-candidates", nil},
		{"POST", o + "/assign", map[string]any{"driver_id": drv.ID, "note": "إسناد"}},
		{"POST", o + "/seen", map[string]any{}}, {"POST", o + "/alert-ack", map[string]any{}},
		{"POST", o + "/proof-exception", map[string]any{"reason": "الكاميرا معطّلة"}},
		{"POST", o + "/goods", map[string]any{"to": "platform"}},
		{"POST", o + "/door-resolution", map[string]any{}},
		{"POST", o + "/transition", map[string]any{"to": "cancelled", "note": "إلغاء"}},
		{"GET", a + "/drivers", nil}, {"GET", a + "/merchants", nil},
		{"GET", a + "/merchants/" + item.MerchantID + "/menu", nil}, {"GET", a + "/settings", nil},
		// ── السوق — والصنفُ يُضاف ويُعدَّل ويُطفأ من بابه ──
		{"GET", a + "/sections", nil}, {"GET", a + "/sections/" + item.SectionID + "/items", nil},
		{"GET", a + "/categories", nil}, {"GET", a + "/market/items", nil},
		{"GET", a + "/market/stores", nil}, {"GET", a + "/market/quality", nil},
		{"POST", a + "/market/seen", map[string]any{}},
		{"POST", a + "/market/items/bulk", map[string]any{}},
		{"POST", a + "/market/stores/" + item.MerchantID + "/items", map[string]any{}},
		// ── الشكاوى والتقييمات ──
		{"GET", a + "/tickets", nil}, {"POST", a + "/tickets", map[string]any{}},
		{"GET", a + "/tickets/" + none, nil}, {"POST", a + "/tickets/" + none + "/replies", map[string]any{"body": "رد"}},
		{"POST", a + "/tickets/" + none + "/resolve", map[string]any{"resolution": "حُلّت", "compensation": 1000}},
		{"GET", a + "/ratings", nil},
		// ── الطوارئ ──
		{"GET", a + "/emergencies", nil}, {"GET", a + "/emergencies/count", nil},
		{"GET", a + "/emergencies/banner", nil}, {"GET", a + "/emergencies/map", nil},
		{"POST", a + "/emergencies/" + none + "/ack", map[string]any{}},
		// ── طلبات الانضمام ──
		{"GET", a + "/leads", nil}, {"GET", a + "/governorates", nil},
		{"POST", a + "/leads/" + none + "/status", map[string]any{"status": "approved"}},
		// ── خريطة العمليّات وطلبات التوسّع ──
		{"GET", a + "/ops-map/meta", nil}, {"GET", a + "/ops-map/orders", nil},
		{"GET", a + "/ops-map/drivers", nil}, {"GET", a + "/ops-map/merchants", nil},
		{"GET", a + "/ops-map/expansion", nil}, {"GET", a + "/ops-map/search?q=x", nil},
		// ── مراقبة التشغيل ──
		{"GET", a + "/ops/monitor", nil}, {"GET", a + "/ops/status", nil},
	}
	denied := []secCall{
		// **المالُ للماليّة** — والتعويضُ يُقترَح من الشكوى وتوافق عليه الماليّة.
		{"GET", o + "/breakdown", nil}, {"POST", o + "/recompute", map[string]any{}},
		{"POST", o + "/compensate-driver", map[string]any{"amount": 100, "reason": "x"}},
		{"POST", o + "/goods/compensation", map[string]any{"amount": 100}},
		{"POST", a + "/compensations/" + none + "/approve", map[string]any{}},
		{"POST", a + "/wallet-requests/" + none + "/approve", map[string]any{}},
		// **وكلامُ الناس لدعم العملاء** — ولا الحساباتُ ولا التقارير.
		{"GET", o + "/chat", nil}, {"GET", a + "/users?limit=1", nil},
		{"GET", a + "/users/" + staff.ID, nil}, {"GET", a + "/reports", nil},
		// **والقائمةُ بباب إدارة المتاجر تبقى مغلقة** — السوقُ من بابه.
		{"PATCH", a + "/menu/items/" + item.ID, map[string]any{"available": false}},
		{"POST", a + "/merchants/" + item.MerchantID + "/menu/items", map[string]any{}},
		// **والتغطيةُ وإبلاغُ المنتظرين لمن يملك الإعدادات العامّة.**
		{"POST", a + "/ops-map/expansion/notify", map[string]any{}},
		// **والعروضُ للماليّة والتسويق.**
		{"GET", a + "/promos", nil}, {"GET", a + "/offers", nil},
	}
	walkCalls(t, hh, "operations", staff.Token, allowed, denied)

	// **وإطفاءُ صنفٍ من قسم السوق يكتب** — لا تخويلٌ يمرّ ومنطقٌ يسقط.
	if r := hh.PATCH(a+"/market/items/"+item.ID, staff.Token,
		map[string]any{"available": false}); r.Code != http.StatusOK {
		t.Fatalf("**إطفاءُ صنفٍ من السوق رُدّ للعمليّات** — %s", r)
	}
	var avail bool
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT available FROM menu_items WHERE id = $1::uuid`, item.ID).Scan(&avail); err != nil {
		t.Fatalf("قراءةُ الصنف: %v", err)
	}
	if avail {
		t.Fatal("رُدّ نجاحاً والصنفُ متاح — **نجاحٌ لا يكتب**")
	}
}

// financeOwnerSections **أقسامُ الماليّة كما سمّاها المالك** — بأبوابها في اللوحة.
var financeOwnerSections = []string{
	"/dashboard/treasury",    // الخزينة
	"/dashboard/obligations", // الديون
	"/dashboard/losses",      // الخسائر والنزاعات
	"/dashboard/incentives",  // الأهداف والمكافآت
	"/dashboard/promos",      // العروض والخصومات
}

// TestFinSections_NavEqualsOwnerList **ما تراه الماليّةُ في القائمة = قائمةُ المالك.**
func TestFinSections_NavEqualsOwnerList(t *testing.T) {
	hh := New(t)
	caps := map[string]bool{}
	for _, c := range roleCaps(t, hh, "finance") {
		caps[c] = true
	}
	var got []string
	for _, e := range readDashboardNav(t) {
		for _, c := range e.caps {
			if caps[c] {
				got = append(got, e.href)
				break
			}
		}
	}
	want := append([]string(nil), financeOwnerSections...)
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("**أقسامُ `finance` فارقت قرارَ المالك**\n   الظاهر  = %s\n   المُقرَّر = %s",
			strings.Join(got, " · "), strings.Join(want, " · "))
	}
	t.Logf("✓ finance ترى %d أقسام — كما قرّر المالك", len(got))
}

// TestFinSections_EveryButtonWorks **كلُّ زرٍّ في الأقسام الخمسة يُفتح للماليّة** —
// وصفحاتُ الخزينة (النقد · السحوبات · الأرباح · المصروفات · التعويضات) منها.
// **وما خرج عنها يُردّ**، **والخطِرُ يبقى للمالك.**
func TestFinSections_EveryButtonWorks(t *testing.T) {
	hh := New(t)
	treasury(t, hh)
	_, tok := roleUser(t, hh, "finance")
	f := hh.Factory()
	m := f.Merchant()
	drv := f.Driver(OnShift())
	oid, _ := activeOrderFor(t, hh)
	const a = "/api/v1/admin"
	none := "00000000-0000-0000-0000-000000000001"

	allowed := []secCall{
		// ── الخزينة — وتبويباتُها ──
		{"GET", a + "/treasury/overview", nil}, {"GET", a + "/treasury/statement", nil},
		{"GET", a + "/treasury/statement/export", nil}, {"GET", a + "/treasury/withdrawals", nil},
		{"GET", a + "/treasury/health", nil}, {"GET", a + "/cashbox", nil},
		{"GET", a + "/cashbox/closes", nil}, {"POST", a + "/cashbox/closes", map[string]any{}},
		{"POST", a + "/cashbox/closes/" + none + "/approve", map[string]any{}},
		{"POST", a + "/cashbox/shortfalls/" + none + "/resolve", map[string]any{}},
		{"GET", a + "/approvals", nil}, {"GET", a + "/treasury-candidates", nil},
		// ── صفحاتُ المال التي تُفتح من الخزينة ──
		{"GET", a + "/cash/outstanding", nil}, {"GET", a + "/cash/outstanding/export", nil},
		{"GET", a + "/cash/merchant-dues", nil},
		{"POST", a + "/drivers/" + drv.ID + "/settle", map[string]any{"amount": 0}},
		{"GET", a + "/merchants/" + m.ID + "/cash-settlements", nil},
		{"POST", a + "/merchant-cash-settlements/" + none + "/pay", map[string]any{}},
		{"GET", a + "/payouts", nil}, {"POST", a + "/payouts/" + none + "/decide", map[string]any{}},
		{"GET", a + "/profits", nil},
		{"GET", a + "/expenses", nil}, {"GET", a + "/expenses/categories", nil},
		{"GET", a + "/expenses/export", nil}, {"POST", a + "/expenses", map[string]any{}},
		{"GET", a + "/expense-requests", nil},
		{"POST", a + "/expense-requests/" + none + "/approve", map[string]any{}},
		{"GET", a + "/compensations", nil}, {"GET", a + "/compensations/pending", nil},
		{"POST", a + "/compensations/" + none + "/approve", map[string]any{}},
		{"GET", a + "/wallet-requests", nil},
		{"POST", a + "/wallet-requests/" + none + "/approve", map[string]any{}},
		{"POST", a + "/orders/" + oid + "/compensate-driver", map[string]any{"amount": 0}},
		// ── الديون ──
		{"GET", a + "/obligations", nil}, {"GET", a + "/obligations/export", nil},
		{"POST", a + "/obligations/" + none + "/office-cash", map[string]any{}},
		{"POST", a + "/obligations/" + none + "/write-off", map[string]any{}},
		{"GET", a + "/obligation-requests", nil},
		{"POST", a + "/obligation-requests/" + none + "/approve", map[string]any{}},
		// ── الخسائر والنزاعات ──
		{"GET", a + "/reports/losses", nil}, {"GET", a + "/disputes", nil},
		{"GET", a + "/disputes/parties", nil}, {"POST", a + "/disputes", map[string]any{}},
		{"POST", a + "/disputes/" + none + "/propose", map[string]any{}},
		{"GET", a + "/dispute-resolutions", nil},
		{"POST", a + "/dispute-resolutions/" + none + "/approve", map[string]any{}},
		// ── الأهداف والمكافآت ──
		{"GET", a + "/incentives/driver", nil}, {"GET", a + "/incentive-requests", nil},
		{"POST", a + "/incentive-requests/" + none + "/approve", map[string]any{}},
		{"POST", a + "/users/" + drv.ID + "/incentive", map[string]any{}},
		// ── العروض والخصومات — بقدرتها الجديدة ──
		{"GET", a + "/promos", nil}, {"GET", a + "/promos/summary", nil},
		{"POST", a + "/promos", map[string]any{}}, {"PATCH", a + "/promos/" + none, map[string]any{}},
		{"GET", a + "/offers", nil}, {"GET", a + "/offers/audience", nil},
		{"POST", a + "/offers", map[string]any{}},
		{"POST", a + "/offers/" + none + "/active", map[string]any{"active": true}},
		{"GET", a + "/referrals", nil}, {"GET", a + "/promo-approvals", nil},
		{"POST", a + "/promo-approvals/" + none + "/approve", map[string]any{}},
		// **وقراءةُ الإعدادات تبقى** — صفحاتُ المال تقرأ سقوفها.
		{"GET", a + "/settings", nil},
	}
	denied := []secCall{
		// **لا الطلبات ولا سجلّها ولا الحسابات** (قرارُ المالك ٢٠٢٦-١٠-٠٥).
		{"GET", a + "/orders?limit=1", nil}, {"GET", a + "/orders/board", nil},
		{"GET", a + "/orders/" + oid, nil}, {"GET", a + "/ops/monitor", nil},
		{"GET", a + "/ops-map/meta", nil}, {"GET", a + "/ops-map/expansion", nil},
		{"GET", a + "/users?limit=1", nil}, {"GET", a + "/users/" + drv.ID, nil},
		// **ولا التقارير ولا سجلّ الأحداث ولا الإعدادات.**
		{"GET", a + "/reports", nil}, {"GET", a + "/stats", nil}, {"GET", a + "/audit", nil},
		{"GET", a + "/overview", nil},
		{"PUT", a + "/settings/finance.manual_wallet_max", map[string]any{"value": 1}},
		// **ولا اللافتاتُ ولا الحملاتُ ولا البثُّ ولا السوقُ ولا صورُ المنصّة.**
		{"GET", a + "/banners", nil}, {"GET", a + "/campaigns", nil},
		{"GET", a + "/broadcast/count", nil}, {"GET", a + "/sections", nil},
		{"GET", a + "/market/items", nil}, {"POST", a + "/media", map[string]any{}},
		// **والخطِرُ يبقى للمالك**: سحبُ الخزينة وقبولُ عجز الصندوق وإعادةُ الحساب
		// والأدوار.
		{"POST", a + "/treasury/withdrawals", map[string]any{}},
		{"POST", a + "/cashbox/shortfalls/" + none + "/approve", map[string]any{}},
		{"POST", a + "/orders/" + oid + "/recompute", map[string]any{}},
		{"GET", a + "/roles", nil},
		// **ولا يدَ تشغيليّة.**
		{"POST", a + "/orders/" + oid + "/assign", map[string]any{"driver_id": drv.ID}},
	}
	walkCalls(t, hh, "finance", tok, allowed, denied)
}
