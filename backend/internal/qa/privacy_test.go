// اختباراتُ عقد الخصوصيّة — **المرحلةُ `P-1`.**
//
// **وتقيس ولا تُصلح.** `D20` و`D21` و`D22` مجمَّدةٌ ولم تُمسّ — **والاختبارُ
// يمثّل الحقيقةَ كما هي**، فيسقط باسم العيب ولا يُخفَّف العقدُ ليخضرّ.
//
// # الوسمُ والدلالة
//
// **`EXPECTED FAIL / BLOCKED BY D20` ليس نجاحاً** — **وهو عيبٌ مانعٌ في
// بوّابة الإصدار بحسب شدّته** (`TQ-1`). **ويُطبَع بـ`t.Log` ولا يُسقط
// البناء**، **فحين يُصلَح العيبُ ينجح الادّعاءُ ويُحذف الوسم.**
package qa

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// ══════════════════════════════════════════════════════════════════════
// **١ · حارسُ الاكتمال — أهمُّ ما في المرحلة**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ١٠ من طلب المالك: `NEW ORDER FIELD WITHOUT PRIVACY DECISION =
// TEST FAILURE`.)
//
// **الحقولُ تُقرأ بالانعكاس من `orders.Order` نفسِه** — **فلا قائمةَ
// مكتوبةٌ تشيخ بصمت.**

func TestOrderFieldsAllClassified(t *testing.T) {
	fields := OrderFields()
	if len(fields) == 0 {
		t.Fatal("لم يُقرأ حقلٌ واحدٌ من orders.Order — الانعكاسُ مكسور")
	}

	var missing []string
	for _, f := range fields {
		if _, ok := OrderPrivacy[f]; !ok {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		t.Errorf("حقولٌ في orders.Order بلا حكمِ خصوصيّة (%d):\n  %v\n\n"+
			"**من أضاف حقلاً إلى الطلب يقول من يراه** — أضِفه إلى OrderPrivacy.",
			len(missing), missing)
	}
	t.Logf("حقولُ الطلب المُصنَّفة: %d/%d", len(fields)-len(missing), len(fields))
}

// **وعقدٌ يصف حقلاً لا وجودَ له يُنظَّف** — وإلّا تراكم الميّتُ فيه.
func TestPrivacyContractHasNoGhosts(t *testing.T) {
	live := map[string]bool{}
	for _, f := range OrderFields() {
		live[f] = true
	}
	var ghosts []string
	for f := range OrderPrivacy {
		if !live[f] {
			ghosts = append(ghosts, f)
		}
	}
	if len(ghosts) > 0 {
		t.Errorf("العقدُ يصنّف حقولاً لا وجودَ لها في orders.Order: %v", ghosts)
	}
}

// **ولكلّ حقلٍ حكمٌ في الأدوار الخمسة** — **ولا دورَ يسقط سهواً.**
func TestEveryFieldCoversEveryRole(t *testing.T) {
	for _, f := range OrderFields() {
		rule, ok := OrderPrivacy[f]
		if !ok {
			continue // يُبلَّغ عنه في TestOrderFieldsAllClassified
		}
		for _, role := range PrivacyRoles {
			if vis, ok := rule.Vis[role]; !ok || vis == VisUnknown {
				t.Errorf("الحقلُ %q بلا حكمٍ للدور %q", f, role)
			}
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · المجهولُ سقوطٌ لا إجازة**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ١١: `UNKNOWN = FAIL`.)

func TestUnknownFieldFailsClosed(t *testing.T) {
	if got := Visible("لا_وجود_له", RoleCustomer); got != VisUnknown {
		t.Fatalf("حقلٌ مجهولٌ ردّ %v — والمنتظَرُ UNKNOWN", got)
	}
	if got := Visible("driver_phone", "دورٌ_لا_وجود_له"); got != VisUnknown {
		t.Fatalf("دورٌ مجهولٌ ردّ %v — والمنتظَرُ UNKNOWN", got)
	}
	// **والمجهولُ يُعدّ خرقاً في القياس لا يُتخطّى.**
	bad := CheckPayload(RoleCustomer, ChannelREST, map[string]any{"حقل_طارئ": "قيمة"})
	if len(bad) != 1 {
		t.Fatalf("حقلٌ مجهولٌ لم يُعدّ خرقاً — العقدُ يُجيز ما لا يعرف")
	}
	if bad[0].Vis != VisUnknown {
		t.Fatalf("الخرقُ سُجّل %v لا UNKNOWN", bad[0].Vis)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · حارسُ الحقل الممنوع — الاختبارُ الذاتيّ**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ١٧: `FORBIDDEN FIELD GUARD = PROVEN`.)
//
// **ولا تُثبَت أداةٌ بأنّها موجودة** — **تُثبَت بأن تُمرَّر إليها الحالةُ
// التي وُضعت لها فتمسكها.**

func TestForbiddenFieldGuardCatchesLeak(t *testing.T) {
	// **حمولةٌ فيها هاتفُ سائقٍ إلى زبون** — **وهي `D21` بعينها.**
	leak := map[string]any{
		"id":           "ord-1",
		"number":       int64(1001),
		"status":       "on_the_way",
		"driver_name":  "سائق",
		"driver_phone": "0900000000", // ← ممنوع
	}
	bad := CheckPayload(RoleCustomer, ChannelREST, leak)
	if len(bad) != 1 {
		t.Fatalf("الحارسُ لم يمسك تسريبَ هاتف السائق — أُمسك %d خرقاً", len(bad))
	}
	if bad[0].Field != "driver_phone" {
		t.Fatalf("أُمسك الحقلُ %q لا driver_phone", bad[0].Field)
	}
	t.Logf("FORBIDDEN FIELD GUARD = PROVEN — %s", bad[0])

	// **والقيمةُ الفارغةُ ليست خرقاً** — **التنقيةُ هنا تصفّر ولا تحذف.**
	clean := map[string]any{"id": "ord-1", "driver_phone": ""}
	if bad := CheckPayload(RoleCustomer, ChannelREST, clean); len(bad) != 0 {
		t.Fatalf("حقلٌ مصفَّرٌ عُدَّ خرقاً: %v", bad)
	}

	// **والمسموحُ يمرّ** — **وحارسٌ يمسك كلَّ شيءٍ لا يمسك شيئاً.**
	ok := map[string]any{"id": "ord-1", "driver_name": "سائق", "total": int64(5000)}
	if bad := CheckPayload(RoleCustomer, ChannelREST, ok); len(bad) != 0 {
		t.Fatalf("حقولٌ مسموحةٌ عُدَّت خرقاً: %v", bad)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٤ · `D21` — هاتفُ السائق يصل الزبون**
// ══════════════════════════════════════════════════════════════════════
//
// **يقيس التنقيةَ القائمةَ نفسَها** — لا نسخةً منها.

func TestD21_CustomerRedactionAgainstContract(t *testing.T) {
	o := fullOrder()

	// **والمُقاسُ هو المرشَّحُ الذي يخرج منه الردُّ فعلاً.**
	//
	// **وكانت هنا نسخةٌ من `redactForCustomer`** — أربعةُ أسطرٍ تمحو
	// ما يدلّ على المصدر، **ثمّ يُسلسَل الباقي.** **ورُفعت الدالّةُ في
	// دورةِ ٤٦** (`orders.ViewFor` صارت الحكم)، **ومحاكاةٌ تصف شيفرةً
	// زالت تخضرّ على عدم.**
	view := orders.ViewFor(orders.AudienceCustomer, &o)
	if view == nil {
		t.Fatal("تعذّر بناءُ حمولة الزبون — **والسقوطُ مغلقٌ فلا يُبثُّ خام**")
	}

	bad := CheckPayload(RoleCustomer, ChannelREST, view)
	for _, x := range bad {
		t.Errorf("  %s", x)
	}
	if len(bad) > 0 {
		t.Errorf("**%d حقلاً محظوراً في حمولة الزبون.** (`D21`)", len(bad))
	}

	// **وما يدلّ على المصدر لا يخرج** — وهو حكمُ العقد نفسِه.
	for _, k := range []string{"merchant_id", "merchant_name",
		"merchant_logo_thumb_url", "offered_driver_name", "driver_phone"} {
		if _, ok := view[k]; ok {
			t.Errorf("**%q وصل الزبونَ** — %v", k, view[k])
		}
	}
	t.Logf("D21 CUSTOMER = مغلق — %d حقلاً وصل الزبون، ولا محظورَ فيها", len(view))
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · `D20` — البثُّ يتجاوز تنقيةَ المتجر**
// ══════════════════════════════════════════════════════════════════════
//
// **ويُقاس الفرقُ بين قناتين على العقد نفسِه** — **وهو ما يجعل العيبَ
// مرئيّاً**: الحمولةُ نفسُها تمرّ في بابٍ وتُمنع في آخر.

func TestD20_MerchantRealtimeVsREST(t *testing.T) {
	// **REST**: ما تفعله `redactForMerchant` — ستّةَ عشرَ حقلاً.
	//
	// **ونُسخت هنا لأنّ الدالّةَ غيرُ مُصدَّرةٍ من حزمة `server`.**
	rest := fullOrder()
	rest.CustomerID, rest.CustomerName, rest.CustomerPhone = "", "", ""
	rest.AddressText = ""
	rest.Lat, rest.Lng = 0, 0
	rest.ZoneID, rest.ZoneName = nil, nil
	rest.DriverID, rest.DriverPhone, rest.DriverName = nil, nil, nil
	rest.OfferedDriverName = nil
	rest.ProofURL, rest.ProofTakenAt = nil, nil
	rest.ProofMeters, rest.ProofSkipReason = 0, ""
	rest.Subtotal, rest.DeliveryFee, rest.Discount, rest.Total = 0, 0, 0, 0
	rest.WalletPaid, rest.CashDue = 0, 0
	rest.PaymentMethod = ""
	rest.PromoCode = nil

	restBad := CheckPayload(RoleMerchant, ChannelREST, toMap(t, rest))

	// **والبثُّ يُقاس من بابه هو لا من محاكاة** — `orders.ViewFor`.
	//
	// **وكان هذا السطرُ يُسلسِل الكائنَ كما هو** لأنّ `publishOrder`
	// كانت تبثّه كما هو. **فلمّا صارت تبني بالسماح تبدّل المقياس** —
	// **ومحاكاةٌ تصف شيفرةً زالت تخضرّ على عدم.**
	live := fullOrder()
	liveBad := CheckPayload(RoleMerchant, ChannelRealtime,
		orders.ViewFor(orders.AudienceMerchant, &live))

	t.Logf("REST     : %d خرقاً", len(restBad))
	t.Logf("REALTIME : %d خرقاً", len(liveBad))

	// **والحكمُ اتّجاهان**: **لا يزيد البثُّ على `REST`** — وهو `D20` —
	// **ولا يخرق البثُّ العقدَ أصلاً**، **فمقارنةٌ بقناةٍ مخروقةٍ
	// تُجيز الخرقَ الموروث.**
	for _, x := range liveBad {
		t.Errorf("  %s", x)
	}
	if len(liveBad) > 0 {
		t.Errorf("**%d حقلاً محظوراً وصل غرفةَ المتجر** — **والبثُّ ليس قناةً "+
			"مميّزة.** (`D20`)", len(liveBad))
	}
	if len(liveBad) > len(restBad) {
		t.Errorf("**البثُّ يسرّب %d حقلاً زيادةً على REST** — **والفرقُ بين "+
			"القناتين هو `D23` نفسُه.**", len(liveBad)-len(restBad))
	}
	t.Logf("D20 MERCHANT = مغلق — والباقي في `REST` وحدَه (%d خرقاً، `D23`)",
		len(restBad))
}

// **وحمولةُ الزبون في البثّ تُقاس بالعقد لا بمحوٍ بعد البناء.**
//
// **وكان `publishOrder` يمحو ثلاثةَ حقولٍ ويبثّ الباقي** — **و`REST`
// تمحو أربعة**: **فـ`offered_driver_name` يُمنع في بابٍ ويمرّ في آخر.**
func TestD20_CustomerRealtimeVsREST(t *testing.T) {
	live := fullOrder()
	view := orders.ViewFor(orders.AudienceCustomer, &live)

	bad := CheckPayload(RoleCustomer, ChannelRealtime, view)
	for _, x := range bad {
		t.Errorf("  %s", x)
	}
	if len(bad) > 0 {
		t.Errorf("**%d حقلاً محظوراً وصل الزبونَ في البثّ.** (`D20`/`D21`)", len(bad))
	}
	if _, ok := view["offered_driver_name"]; ok {
		t.Errorf("**`offered_driver_name` يمرّ في البثّ ويُمنع في REST** — " +
			"**وهذا هو الانحرافُ بعينه.**")
	}
	t.Logf("D20/D21 CUSTOMER REALTIME = مغلق — %d حقلاً وصل الزبون", len(view))
}

// ══════════════════════════════════════════════════════════════════════
// **٦ · `D22` — عقدُ القناة لا تنقيةُ الحقول**
// ══════════════════════════════════════════════════════════════════════
//
// **`D22` ليس تسريبَ حقلٍ بل غيابَ نشرة** — **فما يُبنى هنا هو موضعُ
// العقد**، **وإثباتُه التشغيليُّ يحتاج مشتركَ بثٍّ حيّاً وهو من `P-7`.**
// (البند ٦ من طلب المالك: لا تُوسَّع `P-1` بلا داعٍ.)

// ChannelRequirement **ما يجب أن يصل صاحبَه في القناة الحيّة.**
type ChannelRequirement struct {
	Event string
	Rooms []string
	Ref   string
}

// OrderChannelContract **غرفُ البثّ الواجبةُ لكلّ حدث.**
var OrderChannelContract = []ChannelRequirement{
	{Event: "order.created", Rooms: []string{"ops", "merchant", "customer"},
		Ref: "orders/service.go:118-134 — publishOrder"},
	{Event: "custom_order.created", Rooms: []string{"ops", "customer"},
		Ref: "D22 · XQ-4 — والقائمُ يبثّ ops وحدَها (custom.go:213)"},
	{Event: "order.transition", Rooms: []string{"ops", "merchant", "customer", "driver"},
		Ref: "orders/service.go — publishOrder"},
}

func TestD22_CustomOrderOwnerChannelContract(t *testing.T) {
	var req *ChannelRequirement
	for i := range OrderChannelContract {
		if OrderChannelContract[i].Event == "custom_order.created" {
			req = &OrderChannelContract[i]
		}
	}
	if req == nil {
		t.Fatal("عقدُ القناة لا يصف إنشاءَ الطلب الخاصّ")
	}
	var hasCustomer bool
	for _, r := range req.Rooms {
		if r == "customer" {
			hasCustomer = true
		}
	}
	if !hasCustomer {
		t.Fatal("العقدُ لا يوجب غرفةَ الزبون — وهو ما يقوله XQ-4")
	}
	t.Logf("EXPECTED FAIL / BLOCKED BY D22 — العقدُ يوجب غرفةَ الزبون عند إنشاء الطلب الخاصّ، "+
		"و`custom.go:213` يبثّ ops وحدَها. %s", req.Ref)
	t.Log("الإثباتُ التشغيليُّ (مشتركُ بثٍّ حيّ) مؤجَّلٌ إلى P-7 — انظر خطّةَ التنفيذ.")
}

// ══════════════════════════════════════════════════════════════════════
// **٧ · قناةُ الدفع — نطاقٌ يُثبَت لا يُفترَض**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ٩: لا يُفترَض أنّ الدفعَ آمنٌ لأنّه «مجرّدُ إشعار».)

func TestPushPayloadScope(t *testing.T) {
	// **ما يحمله الدفعُ فعلاً** — `notifications.go:265-274`:
	// عنوانٌ ونصٌّ و`kind` و`entity` و`entity_id`.
	//
	// **ولا حقلَ من حقول الطلب الخمسة والسبعين** — **فالنطاقُ ضيّقٌ
	// بطبيعته، ويُثبَت لا يُفترَض.**
	push := map[string]any{
		"kind":      "order",
		"entity":    "order",
		"entity_id": "ord-1",
	}

	// **ولا حقلَ من حقول الطلب في المغلّف** — **وهذا ما يُثبَت.**
	fields := map[string]bool{}
	for _, f := range OrderFields() {
		fields[f] = true
	}
	var carried []string
	for k := range push {
		if fields[k] {
			carried = append(carried, k)
		}
	}

	// ⚠️ **وتصادمُ أسماءٍ كشفه القياس.**
	//
	// **`kind` اسمٌ في المغلّف واسمٌ في الطلب معاً**: في المغلّف نوعُ
	// الإشعار (`order`/`wallet`)، وفي الطلب نوعُه (`normal`/`custom`).
	// **فالاسمُ واحدٌ والمعنى اثنان.**
	//
	// **ولا ضررَ اليومَ**: قيمتُه في المغلّف ليست قيمةَ الطلب، **والحقلُ
	// مسموحٌ للأدوار الخمسة على كلّ حال.** **ويُسجَّل لئلّا يُقاس يوماً
	// بحكمِ غيرِه.**
	if len(carried) != 1 || carried[0] != "kind" {
		t.Fatalf("مغلّفُ الإشعار يحمل حقولَ طلبٍ غيرَ متوقَّعة: %v", carried)
	}

	for _, role := range PrivacyRoles {
		for _, x := range CheckPayload(role, ChannelPush, push) {
			// **المجهولُ هنا مغلّفٌ لا حقلُ طلب** — **والعقدُ يحكم حقولَ
			// `Order` وحدَها.**
			if x.Vis != VisUnknown {
				t.Errorf("خرقٌ في مغلّف الإشعار لدور %s: %s", role, x)
			}
		}
	}
	t.Log("PUSH PRIVACY CONTRACT = NOT APPLICABLE (اليوم) — " +
		"الدفعُ يحمل مغلّفاً لا حقولَ طلب. ويصير منطبقاً إن حمل حقلاً من Order.")
	t.Log("ملاحظة: الاسمُ `kind` مشتركٌ بين المغلّف والطلب — معنيان باسمٍ واحد.")
}

// ══════════════════════════════════════════════════════════════════════
// **أدواتٌ**
// ══════════════════════════════════════════════════════════════════════

// fullOrder **طلبٌ كلُّ حقلٍ فيه غيرُ صفريّ** — **فما مرّ مرّ عن قصد.**
//
// **وكان قائمةً مكتوبةً بيد** — **فحقلٌ يُضاف إلى `orders.Order` يبقى
// صفريّاً فيها**، **والصفرُ لا يُعَدّ خرقاً** (`CheckPayload`): **فيمرّ
// الجديدُ لأنّه كان فارغاً لا لأنّه مُنِع.**
//
// **فصار يُملأ بالانعكاس** — انظر `fillNonZero`.
func fullOrder() orders.Order {
	var o orders.Order
	fillNonZero(reflect.ValueOf(&o).Elem(), 0)
	return o
}

// toMap **الحمولةُ كما تخرج فعلاً** — عبر التسلسل نفسِه لا عبر الانعكاس.
func toMap(t *testing.T, o orders.Order) map[string]any {
	t.Helper()
	raw, err := json.Marshal(o)
	if err != nil {
		t.Fatalf("تعذّر تسلسلُ الطلب: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("تعذّرت قراءةُ الحمولة: %v", err)
	}
	return m
}
