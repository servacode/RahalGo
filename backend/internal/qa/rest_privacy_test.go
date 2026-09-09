// خصوصيّةُ `REST` — **`D21` و`D23`.**
//
// # لماذا هنا
//
// **دورةُ ٤٤ أغلقت البثَّ** (`D20`): صار يبني حمولتَه بالسماح من عقد
// `P-1`. **وبقيت `REST` على المنع**: `redactForCustomer` تمحو أربعةَ
// حقولٍ، و`redactForMerchant` ستّةَ عشر — **وما لم تُذكَر في المحو
// يخرج.**
//
// # وما يُقاس
//
// **الأبوابُ كما يفتحها التطبيقُ لا كما تُقرأ في الشيفرة**: القائمةُ
// والتفصيلُ للزبون، والقائمةُ والتفصيلُ للمتجر، **والبثُّ نفسُه** —
// **فالمقارنةُ بين قناتين هي `D23`.**
//
// # وطلبٌ مملوءٌ لا عيّنة
//
// **حقلٌ فارغٌ ليس خرقاً** (`CheckPayload`) — **فطلبٌ نصفُه أصفارٌ
// يُخفي التسريبَ لا يُظهره.** **فتُملأ الأعمدةُ الحسّاسةُ كلُّها
// قبل القياس.**
package qa

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// restFixture طلبٌ واحدٌ مملوءٌ وأطرافُه.
type restFixture struct {
	OrderID     string
	Cust        *User
	MerchantID  string
	MerchToken  string
	Driver      *User
	DriverPhone string
	CustPhone   string
}

// populatedOrder **طلبٌ لا عمودَ حسّاسٌ فيه فارغ.**
//
// **ويُبنى بالمسار الحقيقيّ** — إنشاءٌ وقبولُ سائق — **ثمّ تُملأ
// الأعمدةُ التي لا يبلغها مسارٌ واحد** (إثباتُ التسليم · اللقطةُ
// الماليّة · حكمُ الخطأ · تسويةُ البضاعة).
func populatedOrder(t *testing.T, h *Harness) restFixture {
	t.Helper()
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.assignment_mode", `"queue"`)
	h.Setting("drivers.max_active_orders", "5")
	h.Setting("drivers.cash_limit", "9000000")

	item := h.NewItem(1000)
	cust := h.Customer()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 2))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	if oid == "" {
		t.Fatal("لم يُقرأ معرّفُ الطلب")
	}

	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET status = 'dispatching' WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	drv := f.Driver(OnShift())
	if got := h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil); got.Code >= 400 {
		t.Fatalf("قبولُ السائق: %s", got)
	}

	// **وما لا يبلغه مسارٌ واحد يُملأ مباشرةً** — **والقياسُ على
	// حقلٍ فارغٍ لا يقول شيئاً.**
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET
			notes = 'اطرق البابَ مرّتين',
			promo_code = 'QA10',
			prep_minutes = 20,
			ready_at = now(), accepted_at = now(), picked_up_at = now(),
			delivered_at = now(), closed_at = now(),
			platform_commission = 1500,
			snap_merchant_commission_percent = 10,
			snap_rep_commission_percent = 5,
			goods_settled_to = 'merchant',
			ended_by = 'driver',
			fault = 'driver',
			fail_reason = 'تعذّرَ الوصول',
			pod_skip_reason = 'لا شبكة',
			pod_taken_at = now(),
			pod_mocked = true,
			to_store_eta_sec = 300, to_door_eta_sec = 600,
			custom_request = 'طلبٌ خاصّ', custom_goods_amount = 5000,
			custom_fee = 500, custom_agreed_at = now(),
			dispatched_at = now(), alerted_at = now(), sent_to_merchant_at = now()
		WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("ملءُ الأعمدة: %v", err)
	}

	// **ونسبةُ المتجر تُضبَط** — **وبدونها يخرج `fillMerchantMoney`
	// صامتاً** (قراءةُ `commission_percent` تسقط على `NULL`)، **فلا
	// تُقاس شفافيّةُ المتجر أصلاً.**
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE merchants SET commission_percent = 10
		WHERE id = (SELECT merchant_id FROM orders WHERE id = $1::uuid)`,
		oid); err != nil {
		t.Fatalf("ضبطُ النسبة: %v", err)
	}

	var merchantID, ownerID, driverPhone, custPhone string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT o.merchant_id::text, m.owner_user_id::text, du.phone, cu.phone
		FROM orders o
		JOIN merchants m ON m.id = o.merchant_id
		JOIN users du ON du.id = o.driver_id
		JOIN users cu ON cu.id = o.customer_id
		WHERE o.id = $1::uuid`, oid).
		Scan(&merchantID, &ownerID, &driverPhone, &custPhone); err != nil {
		t.Fatalf("قراءةُ الأطراف: %v", err)
	}

	return restFixture{
		OrderID: oid, Cust: cust, MerchantID: merchantID,
		MerchToken:  h.TokenFor(ownerID, "merchant"),
		Driver:      drv,
		DriverPhone: driverPhone, CustPhone: custPhone,
	}
}

// firstOrderOf **أوّلُ طلبٍ في ردّ قائمة** — والقائمةُ مغلّفٌ لا طلب.
func firstOrderOf(t *testing.T, res Res, key string) map[string]any {
	t.Helper()
	raw, ok := res.JSON()[key].([]any)
	if !ok || len(raw) == 0 {
		return nil
	}
	m, _ := raw[0].(map[string]any)
	return m
}

// deepFind **أتظهر هذه القيمةُ في أيّ موضعٍ من الحمولة؟**
//
// **والأعشاشُ تُقرأ كما تُقرأ القمّة**: **حمولةٌ نظيفةٌ في سطحها
// وتحمل هاتفاً في `events[0].actor_phone` مسرِّبةٌ تماماً.**
func deepFind(v any, needle string) bool {
	if needle == "" {
		return false
	}
	switch x := v.(type) {
	case string:
		return strings.Contains(x, needle)
	case []any:
		for _, e := range x {
			if deepFind(e, needle) {
				return true
			}
		}
	case map[string]any:
		for _, e := range x {
			if deepFind(e, needle) {
				return true
			}
		}
	}
	return false
}

// report يطبع خروقَ حمولةٍ ويردّ عددَها.
func report(t *testing.T, role, channel, label string, p map[string]any) int {
	t.Helper()
	if p == nil {
		t.Logf("  %s — لا حمولة", label)
		return 0
	}
	// **وحقولُ الغلاف تُنحّى بأسمائها لا بالتخطّي** — `RestEnvelope`.
	body := make(map[string]any, len(p))
	env := RestEnvelope[role]
	for k, v := range p {
		if channel == ChannelREST && env[k] {
			continue
		}
		body[k] = v
	}
	vs := CheckPayload(role, channel, body)
	t.Logf("  %-28s حقولٌ=%-3d (غلافٌ %d) خرقٌ=%d",
		label, len(p), len(p)-len(body), len(vs))
	for i, v := range vs {
		if i >= 25 {
			t.Logf("      … و%d غيرُها", len(vs)-25)
			break
		}
		t.Logf("      %s", v)
	}
	return len(vs)
}

// ══════════════════════════════════════════════════════════════════════
// **مصفوفةُ الطرف × القناة × الحقل**
// ══════════════════════════════════════════════════════════════════════

func TestD21_RestOrderPrivacyMatrix(t *testing.T) {
	h := New(t)
	fx := populatedOrder(t, h)

	type probe struct {
		label   string
		role    string
		payload map[string]any
	}
	var probes []probe
	add := func(label, role string, p map[string]any) {
		probes = append(probes, probe{label, role, p})
	}

	// ── الزبون — **الأبوابُ الخمسةُ كلُّها** ──────────────────────
	//
	// **ولا يُقاس بابٌ ويُترك أخوه**: **ثلاثةٌ منها كانت تُسلسِل
	// الكائنَ الداخليَّ بلا تنقيةٍ إطلاقاً** (دورةُ ٤٥).
	detail := h.GET("/api/v1/my/orders/"+fx.OrderID, fx.Cust.Token)
	if detail.Code >= 400 {
		t.Fatalf("تفصيلُ الزبون: %s", detail)
	}
	add("customer/rest/detail", RoleCustomer, detail.JSON())

	list := h.GET("/api/v1/my/orders", fx.Cust.Token)
	if list.Code >= 400 {
		t.Fatalf("قائمةُ الزبون: %s", list)
	}
	add("customer/rest/list", RoleCustomer, firstOrderOf(t, list, "orders"))

	// **وردُّ الإنشاء** — طلبٌ ثانٍ لهذا الزبون.
	item2 := h.NewItem(700)
	made := h.POSTKey("/api/v1/orders", fx.Cust.Token, uniq("k"), orderBody(item2, 1))
	if made.Code >= 400 {
		t.Fatalf("ردُّ الإنشاء: %s", made)
	}
	add("customer/rest/create", RoleCustomer, made.JSON())

	// **وردُّ الإلغاء.**
	if id, _ := made.JSON()["id"].(string); id != "" {
		cancelled := h.POST("/api/v1/orders/"+id+"/cancel", fx.Cust.Token,
			map[string]any{"note": "بدا لي"})
		if cancelled.Code < 400 {
			add("customer/rest/cancel", RoleCustomer, cancelled.JSON())
		} else {
			t.Fatalf("ردُّ الإلغاء: %s", cancelled)
		}
	}

	// **وردُّ إنشاء الخاصّ.**
	cust2 := h.Customer()
	custom := h.POST("/api/v1/orders/custom", cust2.Token, map[string]any{
		"request": "طلبٌ خاصٌّ للقياس", "address_text": "الرقة — شارع الاختبار",
		"lat": 35.9506, "lng": 39.0094,
	})
	if custom.Code < 400 {
		add("customer/rest/custom-create", RoleCustomer, custom.JSON())
	} else {
		t.Fatalf("ردُّ الطلب الخاصّ: %s", custom)
	}

	// ── المتجر ────────────────────────────────────────────────────
	mDetail := h.GET("/api/v1/merchant/orders/"+fx.OrderID, fx.MerchToken)
	if mDetail.Code >= 400 {
		t.Fatalf("تفصيلُ المتجر: %s", mDetail)
	}
	add("merchant/rest/detail", RoleMerchant, mDetail.JSON())

	mList := h.GET("/api/v1/merchant/stores/"+fx.MerchantID+"/orders?closed_only=true",
		fx.MerchToken)
	if mList.Code >= 400 {
		t.Fatalf("قائمةُ المتجر: %s", mList)
	}
	merchantList := firstOrderOf(t, mList, "orders")
	add("merchant/rest/list", RoleMerchant, merchantList)

	// **وردُّ الانتقال** — طلبٌ جديدٌ ينتظر قبولَ متجره.
	//
	// **ولا يقبل المتجرُ إلّا في وضع «المتاجر تدير»** — وإلّا ردّت
	// آلةُ الحالات `invalid_transition`.
	//
	// **والتحويلُ التلقائيُّ يُطفأ** — **وإلّا سبقنا خيطُه إلى الطلب
	// فحرّكه عن `pending`**، **فيسقط الفحصُ في الحزمة الكاملة وينجح
	// وحدَه**: **سباقٌ لا عيبٌ في المنتَج، ولا يُداوى بإعادةِ محاولة.**
	h.Setting("platform.orders_mode", `"merchants"`)
	h.Setting("orders.auto_transfer", "false")
	h.Setting("orders.auto_accept_min", "0")

	fresh := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"),
		orderBody(h.NewItemFor(&Merchant{ID: fx.MerchantID}, 900), 1))
	if fresh.Code >= 400 {
		t.Fatalf("تجهيزُ طلبِ الانتقال: %s", fresh)
	}
	fid, _ := fresh.JSON()["id"].(string)
	moved := h.POST("/api/v1/merchant/orders/"+fid+"/transition", fx.MerchToken,
		map[string]any{"to": "accepted"})
	if moved.Code >= 400 {
		var st string
		_ = h.Pool.QueryRow(ctxBG(),
			`SELECT status FROM orders WHERE id = $1::uuid`, fid).Scan(&st)
		t.Fatalf("**ردُّ الانتقال لم يقع فلا يُقاس**: %s (حالُ الطلب %q)", moved, st)
	}
	add("merchant/rest/transition", RoleMerchant, moved.JSON())

	total := 0
	t.Log("── خروقُ العقد في أبواب REST ──")
	for _, p := range probes {
		total += report(t, p.role, ChannelREST, p.label, p.payload)
	}

	// ── وأعشاشُها ─────────────────────────────────────────────────
	//
	// **والقيمةُ نفسُها تُطلَب في كلّ موضع** — لا اسمُ الحقل.
	nested := 0
	for _, p := range probes {
		if p.payload == nil {
			continue
		}
		var raw any
		b, _ := json.Marshal(p.payload)
		_ = json.Unmarshal(b, &raw)
		switch p.role {
		case RoleCustomer:
			if deepFind(raw, fx.DriverPhone) {
				nested++
				t.Errorf("**هاتفُ السائق في %s** — %s", p.label, fx.DriverPhone)
			}
		case RoleMerchant:
			if deepFind(raw, fx.CustPhone) {
				nested++
				t.Errorf("**هاتفُ الزبون في %s** — %s", p.label, fx.CustPhone)
			}
		}
	}
	t.Logf("NESTED VALUE LEAKS = %d", nested)

	if total > 0 || nested > 0 {
		t.Errorf("**%d خرقاً في أبواب REST و%d تسريباً في الأعشاش.** (`D21`/`D23`)",
			total, nested)
	}

	// ══════════════════════════════════════════════════════════════════
	// **والاتّجاهُ الثاني — ما أُجيز لا يُحجَب**
	// ══════════════════════════════════════════════════════════════════
	//
	// **وحمولةٌ فارغةٌ تُخضِرّ كلَّ فحصِ تسريب** — **فيُقاس النقصانُ
	// كما يُقاس الفيض.**
	need := map[string][]string{
		RoleCustomer: {"id", "number", "status", "total", "items", "address_text",
			"created_at", "payment_method", "cash_due"},
		RoleMerchant: {"id", "number", "status", "items", "notes", "created_at"},
	}
	for _, p := range probes {
		if p.payload == nil {
			continue
		}
		for _, k := range need[p.role] {
			// **والخاصُّ بلا أصناف** — **يطلبه الزبونُ بلفظه**،
			// **فاشتراطُ `items` عليه اشتراطُ ما لا يكون.**
			if k == "items" && p.payload["kind"] == "custom" {
				continue
			}
			if _, ok := p.payload[k]; !ok {
				t.Errorf("**%s بلا %q** — **وحمولةٌ ناقصةٌ تُعمي شاشةً "+
					"ولا تسقط من نفسها.**", p.label, k)
			}
		}
	}

	// ══════════════════════════════════════════════════════════════════
	// **وشفافيّةُ تسوية المتجر** — قرارُ المالك ٢٠٢٦-٠٩-٠٩
	// ══════════════════════════════════════════════════════════════════
	//
	// **ثلاثةٌ تُعرض في شاشتَي تطبيقه**: «المجموع» و«خصم المنصة ١٠٪»
	// و«المستحق لك». **ومن نزعها أعمى الشاشتين** — **والعقدُ يوجبها
	// لا يجيزها فحسب.**
	//
	// **وتُقاس قيمةً لا وجوداً**: **صفرٌ يُقرأ «لا عمولة» فيُفاجأ
	// صاحبُه عند التسوية.**
	if merchantList == nil {
		t.Fatal("**لا طلبَ في قائمة المتجر** — ولا تُقاس شفافيّةُ تسويةٍ على فراغ")
	}
	for _, k := range []string{"subtotal", "platform_commission",
		"commission_percent", "merchant_net"} {
		v, ok := merchantList[k]
		if !ok || isEmpty(v) {
			t.Errorf("**%q غائبٌ أو صفرٌ في قائمة المتجر** — "+
				"**وشفافيّةُ تسويته قرارُ مالكٍ لا تفصيلَ عرض.** (%v)", k, v)
			continue
		}
		t.Logf("  شفافيّةُ التسوية: %s = %v", k, v)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **تكافؤُ القنوات** — `D23`
// ══════════════════════════════════════════════════════════════════════
//
// **ولا تُطلَب حمولتان متطابقتان**: **الغلافُ يختلف والترقيمُ يختلف.**
// **وإنّما يُطلَب أن يكون المنعُ واحداً** — **فحقلٌ مُنع في قناةٍ
// وخرج في أختها منعٌ لم يقع.**

func TestD23_CrossChannelPrivacyParity(t *testing.T) {
	h := New(t)
	fx := populatedOrder(t, h)

	// **والبثُّ يُلتقَط من غرفته لا يُحاكى** — **ومحاكاةٌ تقيس ما
	// أظنُّه لا ما يخرج.**
	capC := h.Listen("customer:" + fx.Cust.ID)
	capM := h.Listen("merchant:" + fx.MerchantID)

	// **ويُوقَظ بانتقالٍ حقيقيّ** — التقاطُ البضاعة.
	if got := h.POST("/api/v1/driver/orders/"+fx.OrderID+"/transition",
		fx.Driver.Token, map[string]any{"to": "at_pickup"}); got.Code >= 400 {
		t.Fatalf("**الانتقالُ لم يقع فلا بثَّ يُقاس**: %s", got)
	}

	live := map[string]map[string]any{
		RoleCustomer: orderPayload(capC.Next(3 * time.Second)),
		RoleMerchant: orderPayload(capM.Next(3 * time.Second)),
	}
	rest := map[string]map[string]any{
		RoleCustomer: h.GET("/api/v1/my/orders/"+fx.OrderID, fx.Cust.Token).JSON(),
		RoleMerchant: h.GET("/api/v1/merchant/orders/"+fx.OrderID, fx.MerchToken).JSON(),
	}

	mismatch, both := 0, 0
	for _, role := range []string{RoleCustomer, RoleMerchant} {
		// **وغيابُ البثّ ليس نجاحاً** — **فحصٌ لا يقيس شيئاً يمرّ
		// دائماً**، **وهو أسوأُ من فحصٍ ساقط.**
		if live[role] == nil {
			t.Fatalf("**لم يصل بثٌّ إلى غرفة %q في المهلة** — "+
				"**ولا مقارنةَ بين قناتين إحداهما صامتة.**", role)
		}
		liveBad := map[string]bool{}
		for _, v := range CheckPayload(role, ChannelRealtime, live[role]) {
			liveBad[v.Field] = true
		}
		// **وحقولُ الغلاف تُنحّى بأسمائها** — `RestEnvelope`.
		body := make(map[string]any, len(rest[role]))
		for k, val := range rest[role] {
			if !RestEnvelope[role][k] {
				body[k] = val
			}
		}
		for _, v := range CheckPayload(role, ChannelREST, body) {
			if liveBad[v.Field] {
				both++
				t.Logf("  [%s] %q محظورٌ ويخرج من القناتين", role, v.Field)
				continue
			}
			mismatch++
			t.Logf("  [%s] %q — REST=يخرج · REALTIME=لا يخرج · العقد=DENY",
				role, v.Field)
		}
		t.Logf("%s: بثٌّ=%d حقلاً (خرقٌ %d) · REST=%d حقلاً (غلافٌ %d)",
			role, len(live[role]), len(liveBad), len(rest[role]),
			len(rest[role])-len(body))
	}

	t.Logf("UNAUTHORIZED CROSS-CHANNEL DIFFERENCES = %d · محظورٌ في القناتين = %d",
		mismatch, both)
	if mismatch > 0 || both > 0 {
		t.Errorf("**%d حقلاً محظوراً يخرج من `REST` وحدَه و%d من القناتين** — "+
			"**والمنعُ إمّا يقع في القناتين أو لم يقع.** (`D23`)", mismatch, both)
	}

	// ══════════════════════════════════════════════════════════════════
	// **ولا يُصلَح انحرافٌ بانحرافٍ مقلوب**
	// ══════════════════════════════════════════════════════════════════
	//
	// **وشفافيّةُ تسوية المتجر أُجيزت للقناتين معاً** (قرارُ المالك
	// ٢٠٢٦-٠٩-٠٩): **فمن أعطاها في `REST` ومنعها في البثّ صنع
	// `D23` جديداً بيده.**
	for _, k := range []string{"subtotal", "platform_commission", "commission_percent"} {
		if !audienceAllows(RoleMerchant, k) {
			t.Errorf("**%q لم يعُد مأذوناً للمتجر** — والقرارُ يقول خلافَه", k)
		}
	}
}

// audienceAllows **أيجيز العقدُ هذا الحقلَ لهذا الطرف؟**
func audienceAllows(role, field string) bool {
	return Visible(field, role) == VisAllowed
}

// ══════════════════════════════════════════════════════════════════════
// **ويسقط مغلقاً** — **لا كائنَ خامّاً بديلاً**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذا ما يجعل السقوطَ آمناً**: **`orderView` تردّ عطباً حين يتعذّر
// بناءُ الحمولة**، **ولا تُسلسِل `orders.Order` كما هو.**
//
// **ويُقاس بالبنية لا بالنيّة**: **حزمةُ `server` لا تُسلسِل طلباً
// خامّاً في بابٍ للزبون أو المتجر** — **ومن أعاد سطراً كهذا غداً
// سقط هنا.**

func TestD21_NoRawOrderSerializationRemains(t *testing.T) {
	root := filepath.Join("..", "server")
	ents, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("قراءةُ حزمة الخادم: %v", err)
	}

	// **وأبوابُ الإدارة والسائق خارجَ هذا العقد** — **لها أطرافُها
	// وعقودُها**، ولا يُقاس بها هذا الحارس.
	audience := map[string]bool{
		"customer_handlers.go":     true,
		"custom_order_handlers.go": true,
		"merchant_handlers.go":     true,
	}
	raw := regexp.MustCompile(`httpx\.JSON\([^,]+,[^,]+,\s*(o|updated|order)\)|Payload:\s*(o|updated|order),`)

	for _, e := range ents {
		if !audience[e.Name()] {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, e.Name()))
		if err != nil {
			t.Fatalf("قراءةُ %s: %v", e.Name(), err)
		}
		for _, m := range raw.FindAllString(string(b), -1) {
			t.Errorf("**%s يُسلسِل طلباً خامّاً**: `%s` — "+
				"**والحمولةُ تُبنى بـ`orderView` لا تُرسَل كما هي.** (`D21`)",
				e.Name(), strings.TrimSpace(m))
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **والتكرارُ يكشف ما لا تكشفه مرّة**
// ══════════════════════════════════════════════════════════════════════
//
// **حمولةٌ تُبنى من خريطةٍ ترتيبُها غيرُ مضمون** — **ومرشَّحٌ يعتمد
// على ترتيبٍ يمرّ مرّةً ويسقط في العاشرة.** **ولا يُقاس بابٌ مرّةً
// ويُقال «آمن».**

func TestD21_RestPrivacyUnderRepetition(t *testing.T) {
	if testing.Short() {
		t.Skip("تكرارٌ — لا يُشغَّل في الوضع القصير")
	}
	h := New(t)
	fx := populatedOrder(t, h)

	type route struct {
		label string
		role  string
		times int
		call  func() map[string]any
	}
	routes := []route{
		{"customer/detail", RoleCustomer, 50, func() map[string]any {
			return h.GET("/api/v1/my/orders/"+fx.OrderID, fx.Cust.Token).JSON()
		}},
		{"customer/list", RoleCustomer, 50, func() map[string]any {
			return firstOrderOf(t, h.GET("/api/v1/my/orders", fx.Cust.Token), "orders")
		}},
		{"merchant/detail", RoleMerchant, 50, func() map[string]any {
			return h.GET("/api/v1/merchant/orders/"+fx.OrderID, fx.MerchToken).JSON()
		}},
		{"merchant/list", RoleMerchant, 50, func() map[string]any {
			return firstOrderOf(t, h.GET("/api/v1/merchant/stores/"+
				fx.MerchantID+"/orders?closed_only=true", fx.MerchToken), "orders")
		}},
	}

	for _, r := range routes {
		bad, empty := 0, 0
		for i := 0; i < r.times; i++ {
			p := r.call()
			if len(p) == 0 {
				empty++
				continue
			}
			body := make(map[string]any, len(p))
			for k, v := range p {
				if !RestEnvelope[r.role][k] {
					body[k] = v
				}
			}
			bad += len(CheckPayload(r.role, ChannelREST, body))
		}
		if bad > 0 || empty > 0 {
			t.Errorf("**%s ×%d — %d خرقاً و%d حمولةً فارغة.**",
				r.label, r.times, bad, empty)
		}
		t.Logf("  %-18s ×%d — نظيف", r.label, r.times)
	}

	// **وأبوابُ الكتابة تُعاد كذلك** — **ولكلٍّ مفتاحُه**، فمنعُ
	// التكرار يردّ المحفوظَ لا الجديد.
	writes := 0
	for i := 0; i < 30; i++ {
		item := h.NewItemFor(&Merchant{ID: fx.MerchantID}, 500)
		made := h.POSTKey("/api/v1/orders", fx.Cust.Token, uniq("k"), orderBody(item, 1))
		if made.Code >= 400 {
			t.Fatalf("الإنشاءُ %d: %s", i, made)
		}
		writes += len(CheckPayload(RoleCustomer, ChannelREST, made.JSON()))
		id, _ := made.JSON()["id"].(string)
		got := h.POST("/api/v1/orders/"+id+"/cancel", fx.Cust.Token,
			map[string]any{"note": "قياس"})
		if got.Code >= 400 {
			t.Fatalf("الإلغاءُ %d: %s", i, got)
		}
		writes += len(CheckPayload(RoleCustomer, ChannelREST, got.JSON()))
	}
	if writes > 0 {
		t.Errorf("**%d خرقاً في أبواب الكتابة ×30.**", writes)
	}
	t.Logf("  %-18s ×30 — نظيف", "customer/create+cancel")
}
