package qa

// ══════════════════════════════════════════════════════════════════════
// **حديثُ الطلب — لكلّ طلبٍ حديثُه** (`CHAT`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// # ولا كيانَ محادثةٍ في القاعدة
//
// **والرسائلُ تُنسَب إلى الطلب مباشرةً** (`order_messages.order_id`) —
// **فلا محادثتان لطلبٍ ولا محادثةٌ لطلبين**: **والعزلُ من بنية
// الجدول لا من شرطٍ يُكتب.**
//
// **وهذا يقيس ما يقع فعلاً** — **لا ما يقوله الجدول**: **زبونٌ
// وطلبان وسائقان، ثمّ زبونٌ وطلبان وسائقٌ واحد.**

import (
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **أدواتُ الفحص**
// ══════════════════════════════════════════════════════════════════════

type chatFx struct {
	Cust *User
	DrvA *User
	DrvB *User
	OrdA string
	OrdB string
	Zone zoneFx
}

// assignDriver **يُسند الطلبَ لسائق** — **فكسچرٌ لا مسارُ منتَج.**
func assignDriver(t *testing.T, h *Harness, orderID, driverID string) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET driver_id = $2::uuid, status = 'assigned' WHERE id = $1::uuid`,
		orderID, driverID); err != nil {
		t.Fatalf("إسنادُ السائق: %v", err)
	}
}

// newChatFx **زبونٌ وطلبان** — **بسائقين أو بسائقٍ واحد.**
func newChatFx(t *testing.T, h *Harness, sameDriver bool) chatFx {
	t.Helper()
	z := zoneForDemand(t, h, "منطقةُ حديثِ الطلب")
	cust := h.Customer()
	drvA := h.NewUser("driver")
	drvB := drvA
	if !sameDriver {
		drvB = h.NewUser("driver")
	}
	a := placeOrder(t, h, cust, z, h.NewItem(1000))
	b := placeOrder(t, h, cust, z, h.NewItem(1000))
	assignDriver(t, h, a, drvA.ID)
	assignDriver(t, h, b, drvB.ID)
	return chatFx{Cust: cust, DrvA: drvA, DrvB: drvB, OrdA: a, OrdB: b, Zone: z}
}

func say(t *testing.T, h *Harness, tok, orderID, body string) Res {
	t.Helper()
	return h.POST("/api/v1/orders/"+orderID+"/messages", tok, map[string]any{"body": body})
}

// sayAged **يقول ثمّ يُزيح الزمنَ** — **وحدُّ المعدّل في المحرّك
// يمنع رسالتين متلاحقتين، وهو عقدٌ قائمٌ لا يُمَسّ.**
func sayAged(t *testing.T, h *Harness, tok, orderID, body string) {
	t.Helper()
	if r := say(t, h, tok, orderID, body); r.Code != http.StatusCreated {
		t.Fatalf("**الرسالةُ رُدّت**: %d / %s", r.Code, r.Err())
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE order_messages SET created_at = created_at - interval '1 minute'
		 WHERE order_id = $1::uuid`, orderID); err != nil {
		t.Fatalf("إزاحةُ الزمن: %v", err)
	}
}

func readChat(t *testing.T, h *Harness, tok, orderID string) []map[string]any {
	t.Helper()
	r := h.GET("/api/v1/orders/"+orderID+"/messages", tok)
	if r.Code != http.StatusOK {
		t.Fatalf("**قراءةُ الحديث رُدّت**: %d / %s", r.Code, r.Err())
	}
	raw, _ := r.JSON()["messages"].([]any)
	out := []map[string]any{}
	for _, x := range raw {
		if m, ok := x.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// threadUnread **غيرُ المقروء لطلبٍ بعينه من سجلّ المحادثات.**
func threadUnread(t *testing.T, h *Harness, tok, orderID string) int {
	t.Helper()
	r := h.GET("/api/v1/my/chats", tok)
	if r.Code != http.StatusOK {
		t.Fatalf("**سجلُّ المحادثات رُدّ**: %d", r.Code)
	}
	rows, _ := r.JSON()["threads"].([]any)
	for _, x := range rows {
		m, _ := x.(map[string]any)
		if m["order_id"] == orderID {
			n, _ := m["unread"].(float64)
			return int(n)
		}
	}
	return -1
}

// chatPushes **صفوفُ الدفع التي تخصّ حديثاً** — **نوعاً ووجهةً.**
func chatPushes(t *testing.T, h *Harness, userID string) []map[string]string {
	t.Helper()
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT title, entity, entity_id::text, push_pending::text
		  FROM notifications
		 WHERE user_id = $1::uuid AND kind = 'chat'
		 ORDER BY created_at`, userID)
	if err != nil {
		t.Fatalf("قراءةُ صفوف الدفع: %v", err)
	}
	defer rows.Close()
	out := []map[string]string{}
	for rows.Next() {
		var title, entity, id, pending string
		if err := rows.Scan(&title, &entity, &id, &pending); err != nil {
			t.Fatalf("مسحُ الصفّ: %v", err)
		}
		out = append(out, map[string]string{
			"title": title, "entity": entity, "entity_id": id, "pending": pending,
		})
	}
	return out
}

// ═════════════════ CHAT-01 · CHAT-03 · CHAT-06 ═════════════════

// TestCHAT01_TwoOrdersTwoConversations **ولكلّ طلبٍ حديثُه.**
func TestCHAT01_TwoOrdersTwoConversations(t *testing.T) {
	hh := New(t)
	fx := newChatFx(t, hh, false)

	sayAged(t, hh, fx.DrvA.Token, fx.OrdA, "أنا عند المتجر — أ")
	sayAged(t, hh, fx.DrvB.Token, fx.OrdB, "وصلت — ب")

	a := readChat(t, hh, fx.Cust.Token, fx.OrdA)
	b := readChat(t, hh, fx.Cust.Token, fx.OrdB)
	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("**تسرّبت الرسائلُ بين الطلبين**: أ=%d ب=%d", len(a), len(b))
	}
	// **CHAT-03 · ولا يظهر نصُّ أ في ب.**
	if a[0]["body"] == b[0]["body"] {
		t.Fatalf("**الحديثان واحد**")
	}
	// **CHAT-06 · وسائقُ ب ليس طرفاً في أ** — **ويُردّ كما يُردّ
	// الغائب**: **ولا يُقال «ليس لك» فتُعَدّ المعرّفات.**
	r := hh.GET("/api/v1/orders/"+fx.OrdA+"/messages", fx.DrvB.Token)
	if r.Code != http.StatusNotFound {
		t.Fatalf("**سائقٌ بلغ حديثَ طلبٍ ليس له**: %d", r.Code)
	}
}

// ═════════════════ CHAT-02 ═════════════════

// TestCHAT02_SameDriverTwoOrdersStaySeparate **والسائقُ نفسُه لطلبين.**
//
// **ولو كان المفتاحُ زوجَ الطرفين لاندمج الحديثان** — **وهو ما
// يمنعه مفتاحُ الطلب.**
func TestCHAT02_SameDriverTwoOrdersStaySeparate(t *testing.T) {
	hh := New(t)
	fx := newChatFx(t, hh, true)
	if fx.DrvA.ID != fx.DrvB.ID {
		t.Fatalf("**فكسچرٌ فاسد**: سائقان")
	}

	sayAged(t, hh, fx.DrvA.Token, fx.OrdA, "الطلب أ — أيُّ باب؟")
	sayAged(t, hh, fx.Cust.Token, fx.OrdB, "الطلب ب — الطابق الثالث")

	a := readChat(t, hh, fx.Cust.Token, fx.OrdA)
	b := readChat(t, hh, fx.Cust.Token, fx.OrdB)
	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("**اندمج الحديثان لأنّ الطرفين واحد**: أ=%d ب=%d", len(a), len(b))
	}
	if a[0]["body"] == b[0]["body"] {
		t.Fatalf("**الحديثان واحد**")
	}
}

// ═════════════════ CHAT-04 ═════════════════

// TestCHAT04_MarkReadIsPerOrder **وفتحُ أ لا يمسح غيرَ المقروء في ب.**
func TestCHAT04_MarkReadIsPerOrder(t *testing.T) {
	hh := New(t)
	fx := newChatFx(t, hh, false)

	for _, s := range []string{"أ-١", "أ-٢", "أ-٣"} {
		sayAged(t, hh, fx.DrvA.Token, fx.OrdA, s)
	}
	sayAged(t, hh, fx.DrvB.Token, fx.OrdB, "ب-١")

	if got := threadUnread(t, hh, fx.Cust.Token, fx.OrdA); got != 3 {
		t.Fatalf("**غيرُ المقروء في أ**: %d — **والمنتظَر ٣**", got)
	}
	if got := threadUnread(t, hh, fx.Cust.Token, fx.OrdB); got != 1 {
		t.Fatalf("**غيرُ المقروء في ب**: %d — **والمنتظَر ١**", got)
	}

	// **والقراءةُ هي الوسم** — **ولا نداءَ ثانٍ.**
	readChat(t, hh, fx.Cust.Token, fx.OrdA)
	if got := threadUnread(t, hh, fx.Cust.Token, fx.OrdA); got != 0 {
		t.Fatalf("**قُرئ أ ولم يُوسَم**: %d", got)
	}
	if got := threadUnread(t, hh, fx.Cust.Token, fx.OrdB); got != 1 {
		t.Fatalf("**فتحُ أ مسح غيرَ المقروء في ب**: %d", got)
	}
}

// ═════════════════ CHAT-05 ═════════════════

// TestCHAT05_StrangerGetsNotFound **ومن ليس طرفاً لا يعرف أنّ الطلبَ قائم.**
func TestCHAT05_StrangerGetsNotFound(t *testing.T) {
	hh := New(t)
	fx := newChatFx(t, hh, false)
	stranger := hh.Customer()

	if r := hh.GET("/api/v1/orders/"+fx.OrdA+"/messages", stranger.Token); r.Code != http.StatusNotFound {
		t.Fatalf("**غريبٌ قرأ حديثَ طلب**: %d", r.Code)
	}
	if r := say(t, hh, stranger.Token, fx.OrdA, "دخيل"); r.Code != http.StatusNotFound {
		t.Fatalf("**غريبٌ كتب في حديث طلب**: %d", r.Code)
	}
	// **ولا صفَّ كُتب.**
	var n int
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM order_messages WHERE order_id = $1::uuid`, fx.OrdA).Scan(&n)
	if n != 0 {
		t.Fatalf("**كُتبت رسالةٌ رغم المنع**: %d", n)
	}
}

// ═════════════════ CHAT-07 · CHAT-08 · CHAT-09 ═════════════════

// TestCHAT07_MessageProducesExactlyOnePush **ورسالةٌ واحدةٌ دفعٌ واحد.**
//
// **وقِيس قبل هذا الإصلاح: صفوفٌ=٠ ونيّةُ دفعٍ=٠ وأهدافُ نقلٍ=٠** —
// **فمن أغلق تطبيقَه لم يعلم أنّ السائق يسأله «أيّ طابق؟».**
func TestCHAT07_MessageProducesExactlyOnePush(t *testing.T) {
	hh := New(t)
	fx := newChatFx(t, hh, false)

	sayAged(t, hh, fx.DrvA.Token, fx.OrdA, "أيُّ طابق؟")

	// **CHAT-08 · والزبونُ وحدَه يُخطَر.**
	got := chatPushes(t, hh, fx.Cust.ID)
	if len(got) != 1 {
		t.Fatalf("**صفوفُ الدفع للزبون**: %d — **والمنتظَر واحد**", len(got))
	}
	if got[0]["entity"] != "order_chat" || got[0]["entity_id"] != fx.OrdA {
		t.Fatalf("**الوجهةُ لا تدلّ على حديث الطلب**: %v", got[0])
	}
	if got[0]["pending"] != "true" {
		t.Fatalf("**لا نيّةَ دفعٍ في الصفّ** — **فلا يلتقطه عاملُ النقل**: %v", got[0])
	}
	// **ورقمُ الطلب في العنوان** — **فتُفرَّق رسالتان من طلبين.**
	if !hasNumber(got[0]["title"]) {
		t.Fatalf("**العنوانُ بلا رقم طلب**: %q", got[0]["title"])
	}
	// **ولا يُخطَر السائقُ برسالة نفسِه.**
	if n := len(chatPushes(t, hh, fx.DrvA.ID)); n != 0 {
		t.Fatalf("**أُخطر المرسِلُ برسالته**: %d", n)
	}

	// **CHAT-09 · ورسالةُ الزبون تُخطِر سائقَ الطلب وحدَه.**
	sayAged(t, hh, fx.Cust.Token, fx.OrdA, "الطابق الثالث")
	if n := len(chatPushes(t, hh, fx.DrvA.ID)); n != 1 {
		t.Fatalf("**صفوفُ الدفع لسائق أ**: %d — **والمنتظَر واحد**", n)
	}
	if n := len(chatPushes(t, hh, fx.DrvB.ID)); n != 0 {
		t.Fatalf("**أُخطر سائقُ طلبٍ آخر**: %d", n)
	}

	// **ويُفرَّع هدفٌ لجهازه فعلاً** — **والصفُّ وحدَه لا يوصِل.**
	if _, err := hh.Pool.Exec(ctxBG(), `
		INSERT INTO device_tokens (user_id, token, platform, app, last_seen_at)
		VALUES ($1::uuid, $2, 'android', 'customer', now())
		ON CONFLICT (token) DO NOTHING`, fx.Cust.ID, "CHAT-TOKEN-"+fx.Cust.ID); err != nil {
		t.Fatalf("تسجيلُ الجهاز: %v", err)
	}
	sayAged(t, hh, fx.DrvA.Token, fx.OrdA, "وصلتُ الباب")
	fanned, _, _, _ := hh.API.DeliverPushOnce(ctxBG())
	if fanned < 1 {
		t.Fatalf("**لم يُفرَّع هدفٌ لجهاز الزبون**: %d", fanned)
	}
}

func hasNumber(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

// ═════════════════ CHAT-10 ═════════════════

// TestCHAT10_FormerDriverGetsNoNewPush **ومن سُحب منه الطلبُ لا يُخطَر.**
//
// **ويبقى له أن يقرأ ما كتب** — **عقدٌ قائمٌ لا يُمَسّ** — **لكنّه لا
// يتلقّى رسالةً جديدة.**
func TestCHAT10_FormerDriverGetsNoNewPush(t *testing.T) {
	hh := New(t)
	fx := newChatFx(t, hh, false)
	sayAged(t, hh, fx.DrvA.Token, fx.OrdA, "أنا في الطريق")

	before := len(chatPushes(t, hh, fx.DrvA.ID))

	// **ثمّ يُسحَب الطلبُ منه ويُسنَد لغيره.**
	assignDriver(t, hh, fx.OrdA, fx.DrvB.ID)
	sayAged(t, hh, fx.Cust.Token, fx.OrdA, "أين أنت؟")

	if got := len(chatPushes(t, hh, fx.DrvA.ID)); got != before {
		t.Fatalf("**أُخطر سائقٌ سُحب منه الطلب**: %d ← %d", before, got)
	}
	if got := len(chatPushes(t, hh, fx.DrvB.ID)); got != 1 {
		t.Fatalf("**لم يُخطَر السائقُ الحامل**: %d", got)
	}
	// **ويبقى الأوّلُ يقرأ ما كتب** — **حجّةً له.**
	if r := hh.GET("/api/v1/orders/"+fx.OrdA+"/messages", fx.DrvA.Token); r.Code != http.StatusOK {
		t.Fatalf("**مُنع من قراءة ما كتب**: %d", r.Code)
	}
	// **ولا يكتب.**
	if r := say(t, hh, fx.DrvA.Token, fx.OrdA, "ما زلتُ هنا"); r.Code == http.StatusCreated {
		t.Fatalf("**كتب سائقٌ سُحب منه الطلب**")
	}
}

// ═════════════════ CHAT-11 · CHAT-12 ═════════════════

// TestCHAT11_CHAT12_TerminalOrderReadableNotWritable **والمنتهي يُقرأ ولا يُكتب.**
func TestCHAT11_CHAT12_TerminalOrderReadableNotWritable(t *testing.T) {
	hh := New(t)
	fx := newChatFx(t, hh, false)
	sayAged(t, hh, fx.DrvA.Token, fx.OrdA, "سلّمتُ الطلب")

	if _, err := hh.Pool.Exec(ctxBG(), `
		UPDATE orders SET status = 'delivered', delivered_at = now() WHERE id = $1::uuid`,
		fx.OrdA); err != nil {
		t.Fatalf("إنهاءُ الطلب: %v", err)
	}

	// **CHAT-12 · ويبقى الحديثُ مقروءاً** — **حجّةً عند الخلاف.**
	if got := readChat(t, hh, fx.Cust.Token, fx.OrdA); len(got) != 1 {
		t.Fatalf("**ذهب حديثُ طلبٍ انتهى**: %d", len(got))
	}
	// **CHAT-11 · ولا يُكتب فيه.**
	if r := say(t, hh, fx.Cust.Token, fx.OrdA, "بعد التسليم"); r.Code == http.StatusCreated {
		t.Fatalf("**كُتب في حديث طلبٍ منتهٍ**")
	}
	// **ولا دفعَ لِما لم يُكتب.**
	if n := len(chatPushes(t, hh, fx.DrvA.ID)); n != 0 {
		t.Fatalf("**دُفع إشعارٌ لرسالةٍ مرفوضة**: %d", n)
	}
}

// ═════════════════ CHAT-13 ═════════════════

// TestCHAT13_RetryDoesNotDuplicatePush **وإعادةُ الإرسال لا تُضاعف الدفع.**
//
// **وحدُّ المعدّل في المحرّك يردّ الثانيةَ** (`ErrTooFast`) — **فلا
// رسالةٌ ثانيةٌ ولا صفُّ دفعٍ ثانٍ.** **ورسالتان مختلفتان حقّاً
// تُدفعان كلتاهما** — **ولا يُخلَط المنعُ بالكتم.**
func TestCHAT13_RetryDoesNotDuplicatePush(t *testing.T) {
	hh := New(t)
	fx := newChatFx(t, hh, false)

	first := say(t, hh, fx.DrvA.Token, fx.OrdA, "وصلت")
	if first.Code != http.StatusCreated {
		t.Fatalf("الأولى: %d / %s", first.Code, first.Err())
	}
	// **وإعادةٌ فوريّةٌ كما تقع على شبكةٍ بطيئة.**
	again := say(t, hh, fx.DrvA.Token, fx.OrdA, "وصلت")
	if again.Code == http.StatusCreated {
		t.Fatalf("**قُبلت إعادةٌ فوريّةٌ فصارت رسالتين**")
	}
	if n := len(chatPushes(t, hh, fx.Cust.ID)); n != 1 {
		t.Fatalf("**صفوفُ الدفع**: %d — **والمنتظَر واحد**", n)
	}

	// **ورسالةٌ ثانيةٌ حقيقيّةٌ تُدفَع** — **والسقفُ ليس كتماً.**
	if _, err := hh.Pool.Exec(ctxBG(), `
		UPDATE order_messages SET created_at = created_at - interval '1 minute'
		 WHERE order_id = $1::uuid`, fx.OrdA); err != nil {
		t.Fatalf("إزاحةُ الزمن: %v", err)
	}
	if r := say(t, hh, fx.DrvA.Token, fx.OrdA, "أنا عند الباب"); r.Code != http.StatusCreated {
		t.Fatalf("الثانية: %d / %s", r.Code, r.Err())
	}
	if n := len(chatPushes(t, hh, fx.Cust.ID)); n != 2 {
		t.Fatalf("**رسالتان مختلفتان ودفعٌ واحد**: %d", n)
	}
}

// ═════════════════ وصفُّ الحديث لا يُعرَض في الصندوق ═════════════════

// TestCHAT_PushRowIsNotAnInboxEntry **ويُدفَع ولا يُعرَض.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «رسائلُ المحادثة ما يصير تصل كإشعارات
//
//	أيضا».) **والحديثُ نفسُه هو السجلّ.**
func TestCHAT_PushRowIsNotAnInboxEntry(t *testing.T) {
	hh := New(t)
	fx := newChatFx(t, hh, false)
	sayAged(t, hh, fx.DrvA.Token, fx.OrdA, "خبرٌ في الحديث")

	r := hh.GET("/api/v1/me/notifications", fx.Cust.Token)
	if r.Code != http.StatusOK {
		t.Fatalf("**الصندوقُ رُدّ**: %d", r.Code)
	}
	items, _ := r.JSON()["items"].([]any)
	for _, x := range items {
		m, _ := x.(map[string]any)
		if m["kind"] == "chat" {
			t.Fatalf("**صفُّ حديثٍ ظهر في الصندوق**: %v", m)
		}
	}
	if n, _ := r.JSON()["unread"].(float64); n != 0 {
		t.Fatalf("**عُدّ صفُّ الحديث في شارة الجرس**: %v", n)
	}
}
