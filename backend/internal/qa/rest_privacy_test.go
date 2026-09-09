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
	vs := CheckPayload(role, channel, p)
	t.Logf("  %-28s حقولٌ=%-3d خرقٌ=%d", label, len(p), len(vs))
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

	// ── الزبون ────────────────────────────────────────────────────
	detail := h.GET("/api/v1/my/orders/"+fx.OrderID, fx.Cust.Token)
	if detail.Code >= 400 {
		t.Fatalf("تفصيلُ الزبون: %s", detail)
	}
	probes = append(probes, probe{"customer/rest/detail", RoleCustomer, detail.JSON()})

	list := h.GET("/api/v1/my/orders", fx.Cust.Token)
	if list.Code >= 400 {
		t.Fatalf("قائمةُ الزبون: %s", list)
	}
	probes = append(probes, probe{"customer/rest/list", RoleCustomer, firstOrderOf(t, list, "orders")})

	// ── المتجر ────────────────────────────────────────────────────
	mDetail := h.GET("/api/v1/merchant/orders/"+fx.OrderID, fx.MerchToken)
	if mDetail.Code < 400 {
		probes = append(probes, probe{"merchant/rest/detail", RoleMerchant, mDetail.JSON()})
	} else {
		t.Logf("تفصيلُ المتجر: %s", mDetail)
	}
	mList := h.GET("/api/v1/merchant/stores/"+fx.MerchantID+"/orders?closed_only=true", fx.MerchToken)
	if mList.Code < 400 {
		probes = append(probes, probe{"merchant/rest/list", RoleMerchant, firstOrderOf(t, mList, "orders")})
	} else {
		t.Logf("قائمةُ المتجر: %s", mList)
	}

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
				t.Logf("  **هاتفُ السائق في %s** — %s", p.label, fx.DriverPhone)
			}
		case RoleMerchant:
			if deepFind(raw, fx.CustPhone) {
				nested++
				t.Logf("  **هاتفُ الزبون في %s** — %s", p.label, fx.CustPhone)
			}
		}
	}
	t.Logf("NESTED VALUE LEAKS = %d", nested)

	if total == 0 && nested == 0 {
		t.Log("D21/D23 REST PRIVACY = PASS — الأبوابُ تطابق العقد. احذفِ الوسم.")
		return
	}
	t.Logf("EXPECTED FAIL / BLOCKED BY D21+D23 — %d خرقاً في أبواب REST "+
		"و%d تسريباً في الأعشاش", total, nested)

	// ══════════════════════════════════════════════════════════════════
	// **وثلاثةٌ منها ليست تسريباً بل خلافُ عقدين**
	// ══════════════════════════════════════════════════════════════════
	//
	// **`subtotal` و`platform_commission` و`commission_percent` تصل
	// المتجرَ بقرارِ المالك** (٢٠٢٦-٠٨-٢٦: «شقد المبلغ المباع وشقد
	// نسبة العمولة للمنصّة — هيك لازم يكون بشفافية»)، **وتُعرض في
	// شاشتَي تطبيقه** (`OrdersScreen` · `HistoryScreen`).
	//
	// **وعقدُ `P-1` يمنعها عليه.** **فليست شيفرةً تخالف عقداً بل
	// عقدان يتخالفان** — **وحلُّها قرارُ مالكٍ لا تعديلُ مبرمج.**
	disputed := map[string]bool{
		"subtotal": true, "platform_commission": true, "commission_percent": true,
	}
	for _, p := range probes {
		if p.role != RoleMerchant || p.payload == nil {
			continue
		}
		for _, v := range CheckPayload(p.role, ChannelREST, p.payload) {
			if disputed[v.Field] {
				t.Logf("  PRODUCT TRUTH CONFLICT — %q في %s: العقدُ يمنعه "+
					"وتطبيقُ المتجر يعرضه", v.Field, p.label)
			}
		}
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
		for _, v := range CheckPayload(role, ChannelREST, rest[role]) {
			if liveBad[v.Field] {
				both++
				t.Logf("  [%s] %q محظورٌ ويخرج من القناتين", role, v.Field)
				continue
			}
			mismatch++
			t.Logf("  [%s] %q — REST=يخرج · REALTIME=لا يخرج · العقد=DENY",
				role, v.Field)
		}
		t.Logf("%s: بثٌّ=%d حقلاً (خرقٌ %d) · REST=%d حقلاً",
			role, len(live[role]), len(liveBad), len(rest[role]))
	}

	t.Logf("UNAUTHORIZED CROSS-CHANNEL DIFFERENCES = %d · محظورٌ في القناتين = %d",
		mismatch, both)
	if mismatch == 0 && both == 0 {
		t.Log("D23 CROSS-CHANNEL = PASS — المنعُ واحدٌ في القناتين. احذفِ الوسم.")
		return
	}
	t.Logf("EXPECTED FAIL / BLOCKED BY D23 — %d حقلاً محظوراً يخرج من `REST` "+
		"وحدَه و%d من القناتين", mismatch, both)
}
