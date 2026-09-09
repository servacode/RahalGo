// الطلبُ الخاصُّ يصل صاحبَه — **`D22`.**
//
// # ما يقع
//
// **`custom.go` لا تنادي `publishOrder` مرّةً واحدة**: الإنشاءُ يبثّ
// `touch("order","ops")` والتوثيقُ يبثّ `Publish("ops", …)` —
// **والعملياتُ تعلم وصاحبُ الطلب لا يعلم.**
//
// **والطلبُ العاديُّ يصل صاحبَه** لأنّه يمرّ بـ`publishOrder`.
// **فالفرقُ ليس في العقد بل في مسارٍ لم يُوصَل.**
//
// # وما يُقاس
//
// **الغرفةُ التي يسمعها التطبيقُ فعلاً** — `customer:<id>` (انظر
// `ws.go`)، **لا اسماً مخترَعاً.**
//
// **ومرّةً واحدة**: **المقبسُ الواحدُ يشترك في `customer:<id>`
// و`user:<id>` معاً** — **فمن بثّ فيهما أرسل حدثاً واحداً مرّتين
// إلى شاشةٍ واحدة.**
package qa

import (
	"testing"
	"time"
)

// newCustomOrder **ينشئ طلباً خاصّاً بالمسار الحقيقيّ** — **ويُثبت
// أنّ النداءَ نجح قبل أن يُفسَّر صمتُ المقبس.**
//
// **وصمتٌ سببُه نداءٌ مرفوض ليس دليلاً على شيء** — **وقد وقع في
// دورةِ ٤٤ فحصٌ يقرأ رفضاً ٤٠٩ ويسمّيه «لم يصل بثّ».**
func newCustomOrder(t *testing.T, h *Harness, owner *User) string {
	t.Helper()
	res := h.POST("/api/v1/orders/custom", owner.Token, map[string]any{
		"request": "كيلو لحمٍ من ملحمة الحيّ", "address_text": "الرقة — شارع الاختبار",
		"lat": 35.9506, "lng": 39.0094,
	})
	if res.Code != 201 {
		t.Fatalf("**إنشاءُ الطلب الخاصّ لم ينجح فلا يُقاس صمتٌ بعده**: %s", res)
	}
	id, _ := res.JSON()["id"].(string)
	if id == "" {
		t.Fatalf("**لم يُقرأ معرّفُ الطلب** — %s", res)
	}
	return id
}

// ══════════════════════════════════════════════════════════════════════
// **١ و٢ و٣ و٤ · يصل صاحبَه مرّةً · وتبقى العملياتُ · ولا يصل غيرَه**
// ══════════════════════════════════════════════════════════════════════

func TestD22_CustomOrderReachesItsOwner(t *testing.T) {
	h := New(t)
	owner := h.Customer()
	other := h.Customer()

	capOwner := h.Listen("customer:" + owner.ID)
	capOther := h.Listen("customer:" + other.ID)
	capOps := h.Listen("ops")

	oid := newCustomOrder(t, h, owner)
	t.Logf("الطلبُ الخاصّ = %s · صاحبُه = %s", oid, owner.ID)

	ownerMsgs := capOwner.Drain(2 * time.Second)
	opsMsgs := capOps.Drain(500 * time.Millisecond)
	otherMsgs := capOther.Drain(500 * time.Millisecond)

	t.Logf("OWNER EVENTS = %d · OPS EVENTS = %d · NON-OWNER EVENTS = %d",
		len(ownerMsgs), len(opsMsgs), len(otherMsgs))

	// ── ١ ── يصل صاحبَه ───────────────────────────────────────────
	if len(ownerMsgs) == 0 {
		t.Errorf("**لم يصل صاحبَ الطلب شيء** — **والعاديُّ يصله**: " +
			"`custom.go` لا تنادي `publishOrder`. (`D22`)")
	}

	// ── ٣ ── ومرّةً واحدة ─────────────────────────────────────────
	//
	// **والمقبسُ يشترك في `customer:` و`user:` معاً** — **فبثٌّ
	// فيهما حدثٌ واحدٌ يصل شاشةً واحدةً مرّتين.**
	if len(ownerMsgs) > 1 {
		t.Errorf("**وصل صاحبَه %d حمولاتٍ لحدثٍ واحد** — "+
			"**والشاشةُ تُعيد الجلبَ بعددها.**", len(ownerMsgs))
	}

	// ── ٢ ── وتبقى العملياتُ كما كانت ─────────────────────────────
	if len(opsMsgs) == 0 {
		t.Errorf("**انقطع بثُّ العمليات** — **ولا يُحلّ توجيهُ صاحبِ " +
			"الطلب بإسكات المكتب.**")
	}

	// ── ٤ ── ولا يصل غيرَه ────────────────────────────────────────
	if len(otherMsgs) > 0 {
		t.Errorf("**وصل زبوناً آخرَ %d حمولة** — **وغرفةُ المرء له وحدَه.**",
			len(otherMsgs))
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · وحمولتُه حمولةُ زبون** — **لا الكائنَ الداخليّ**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يُصلَح `D22` بتسريب**: **الوصولُ الصحيحُ حمولةٌ آمنةٌ تصل** —
// **لا حمولةٌ عريضةٌ ولا صمت.**

func TestD22_OwnerPayloadObeysCustomerPrivacy(t *testing.T) {
	h := New(t)
	owner := h.Customer()
	f := h.Factory()

	capOwner := h.Listen("customer:" + owner.ID)
	oid := newCustomOrder(t, h, owner)
	_ = capOwner.Drain(2 * time.Second)

	// **ويُملأ بما يُفترَض أن يُمنع** — **وحقلٌ فارغٌ لا يُثبت منعاً.**
	drv := f.Driver(OnShift())
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET driver_id = $2::uuid, status = 'assigned',
			fault = 'driver', ended_by = 'driver', goods_settled_to = 'merchant',
			platform_commission = 900, pod_mocked = true,
			pod_skip_reason = 'لا شبكة', sent_to_merchant_at = now()
		WHERE id = $1::uuid`, oid, drv.ID); err != nil {
		t.Fatalf("ملءُ الأعمدة: %v", err)
	}

	// **ثمّ حركةٌ حقيقيّةٌ تُوقظ البثّ** — توثيقُ ما اتُّفق عليه.
	agreed := h.POST("/api/v1/driver/orders/"+oid+"/agree", drv.Token,
		map[string]any{"goods_amount": 5000, "fee": 1000})
	if agreed.Code >= 400 {
		t.Fatalf("**التوثيقُ لم يقع فلا يُقاس ما بعده**: %s", agreed)
	}

	msgs := capOwner.Drain(2 * time.Second)
	t.Logf("وصل صاحبَ الطلب %d حمولة بعد التوثيق", len(msgs))
	if len(msgs) == 0 {
		t.Fatalf("**لم يصل صاحبَ الطلب شيءٌ بعد توثيق ما يدفعه** — " +
			"**واتّفاقٌ لا يراه صاحبُه ليس اتّفاقاً.** (`D22`)")
	}
	if len(msgs) > 1 {
		t.Errorf("**وصلته %d حمولاتٍ لحركةٍ واحدة.**", len(msgs))
	}

	bad := 0
	for _, m := range msgs {
		p := orderPayload(m)
		if len(p) == 0 {
			t.Logf("  حمولةٌ بلا طلب — إشارةُ تحديثٍ صامتة")
			continue
		}
		vs := CheckPayload(RoleCustomer, ChannelRealtime, p)
		for _, v := range vs {
			t.Errorf("  %s", v)
		}
		bad += len(vs)
		t.Logf("  حمولةُ الزبون: %d حقلاً · خرقٌ %d", len(p), len(vs))
	}
	if bad > 0 {
		t.Errorf("**%d حقلاً محظوراً وصل صاحبَ الطلب الخاصّ** — "+
			"**والخاصُّ ليس بابَ استثناء.** (`D22`/`D20`)", bad)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٨ · وغرفُ الخاصّ كغرف العاديّ** — **ولا يُنقَص أحد**
// ══════════════════════════════════════════════════════════════════════

func TestD22_CustomOrderRoomSetIsComplete(t *testing.T) {
	h := New(t)
	owner := h.Customer()

	capOwner := h.Listen("customer:" + owner.ID)
	capOps := h.Listen("ops")
	capQueue := h.Listen("drivers:queue")

	oid := newCustomOrder(t, h, owner)

	got := map[string]int{
		"customer": len(capOwner.Drain(2 * time.Second)),
		"ops":      len(capOps.Drain(500 * time.Millisecond)),
		"queue":    len(capQueue.Drain(500 * time.Millisecond)),
	}
	t.Logf("الطلبُ %s — الغرفُ: %v", oid, got)

	// **والخاصُّ ينتظر موافقةَ المكتب فلا ينزل الطابورَ عند إنشائه** —
	// **وهذا عقدُه لا نقصٌ فيه.** (`custom.go`: يُنشأ `pending`.)
	for _, room := range []string{"customer", "ops"} {
		if got[room] == 0 {
			t.Errorf("**غرفةُ %q لم يصلها شيء.** (`D22`)", room)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **والتكرارُ يكشف ما لا تكشفه مرّة**
// ══════════════════════════════════════════════════════════════════════
//
// **حدثٌ يصل مرّةً في القياس الأوّل قد يصل مرّتين في المئة** —
// **وخيطُ بثٍّ يسبق خيطاً لا يُرى في نداءٍ واحد.**

func TestD22_OwnerDeliveryUnderRepetition(t *testing.T) {
	if testing.Short() {
		t.Skip("تكرارٌ — لا يُشغَّل في الوضع القصير")
	}
	h := New(t)
	owner := h.Customer()
	other := h.Customer()

	// **وسقفُ المفتوح يُرفَع لهذا القياس وحدَه** — **ومئةُ طلبٍ
	// لزبونٍ واحدٍ تصطدم به في الرابع.** **والسقفُ حكمٌ قائمٌ لا
	// يُلغى**: يُضبط في مِسنَد هذا الفحص ويعود بعده.
	h.Setting("orders.max_open_per_customer", "500")

	capOwner := h.Listen("customer:" + owner.ID)
	capOther := h.Listen("customer:" + other.ID)
	capOps := h.Listen("ops")

	const rounds = 100
	misses, dups, leaks, bad := 0, 0, 0, 0
	opsMisses := 0

	for i := 0; i < rounds; i++ {
		_ = newCustomOrder(t, h, owner)

		msgs := capOwner.Drain(1500 * time.Millisecond)
		switch {
		case len(msgs) == 0:
			misses++
		case len(msgs) > 1:
			dups++
		}
		for _, m := range msgs {
			if p := orderPayload(m); len(p) > 0 {
				bad += len(CheckPayload(RoleCustomer, ChannelRealtime, p))
			}
		}

		// **والمكتبُ يُقاس في النصف الأوّل** — خمسون تكفي، **والباقي
		// للتوصيل والتفرّد.**
		if i < 50 {
			if len(capOps.Drain(300*time.Millisecond)) == 0 {
				opsMisses++
			}
		}
		if i < 50 {
			leaks += len(capOther.Drain(200 * time.Millisecond))
		}
	}

	t.Logf("OWNER ×%d — لم يصل %d · مكرَّرٌ %d · خرقٌ %d", rounds, misses, dups, bad)
	t.Logf("OPS ×50 — لم يصل %d · NON-OWNER ×50 — تسرّب %d", opsMisses, leaks)
	if misses > 0 || dups > 0 || bad > 0 || opsMisses > 0 || leaks > 0 {
		t.Errorf("**التكرار: %d لم يصل · %d مكرَّر · %d خرقاً · %d مكتبٌ لم يصله · "+
			"%d تسريباً لغير صاحبه.** (`D22`)", misses, dups, bad, opsMisses, leaks)
	}
}
