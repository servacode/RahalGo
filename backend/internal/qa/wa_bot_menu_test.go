package qa

// ══════════════════════════════════════════════════════════════════════
// **قائمة بوت الواتساب** (قرارُ المالك ٢٠٢٦-١١-٢٧) — `server/wa_bot_menu.go`
// ══════════════════════════════════════════════════════════════════════
//
// القائمةُ تظهر أوّلَ رسالةٍ وبعد سكوتِ ساعةٍ (إعداد) و«0»/«قائمة»؛ لا للكباتن والمندوبين والمتاجر؛
// خارجَ الدوام رسالةُ موعد الفتح؛ طلبٌ موجَّهٌ يولد معلّقاً لا يبلغ الكباتن قبل الموافقة؛ المتابعةُ
// لصاحب الطلب وحدَه؛ التسجيلُ ككابتن بتحقّق نعم/لا؛ تذكيرٌ بعد ثلاث دقائق ثمّ إسقاطٌ صامت؛ والردُّ
// غيرُ المفهوم يُوضَّح مرّتين ثمّ «0 أو 3» ثمّ صمت.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/notify"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/platform"
)

const (
	wbMenuHead   = "أهلاً بك في رحّال غو 👋"
	wbUnclear    = "عذراً، لم أفهم ردّك."
	wbStuck      = "يمكنك كتابة 0 للعودة إلى القائمة، أو 3 للتحدّث مع الإدارة."
	wbOrderAsk   = "يرجى كتابة طلبك كاملاً في رسالة مكتوبة، ونرجو عدم إرسال رسائل صوتية.\n\nيرجى العلم أن الطلب الواحد يكون من محل واحد فقط. إذا كان طلبك من محلّين أو أكثر، يُحسب كل محل توصيلة منفصلة بأجرة خاصة."
	wbEditQ      = "هل لديك تعديل أو إضافة على الطلب؟ أجب بـ: نعم أو لا."
	wbEditAsk    = "يرجى كتابة التعديل أو الإضافة كاملةً."
	wbStoreQ     = "هل تريد الطلب من متجر أو محل معيّن؟ اكتب اسمه، أو اكتب: لا يوجد."
	wbLocAsk     = "يرجى إرسال موقعك (العادي أو المباشر)."
	wbVoice      = "عذراً، يرجى كتابة الطلب كتابةً."
	wbYesNo      = "يرجى الإجابة بـ: نعم أو لا."
	wbTrackAsk   = "يرجى كتابة رقم طلبك."
	wbNotFound   = "لم نجد طلباً بهذا الرقم مرتبطاً برقمك."
	wbSignupDone = "تم تسجيل جميع معلوماتك، وستتواصل معك الإدارة لاحقاً في الوقت المناسب."
	wbReminder   = "هل ما زلت معنا؟ يرجى إكمال الإجابة لنتابع طلبك."
)

// wbOpen **المنصّةُ مفتوحة والقائمةُ مشتعلة** — لا يتأثّر الفحصُ بجدولٍ خلّفه غيرُه.
func wbOpen(t *testing.T, h *Harness) {
	t.Helper()
	h.Setting(platform.SwitchKey, `"open"`)
	h.Setting("whatsapp.bot_menu_enabled", `true`)
	h.Setting("whatsapp.bot_menu_silence_min", `60`)
}

func wbSay(h *Harness, phone, text string) string {
	return h.API.HandleWABotMenu(ctxBG(), notify.MenuInbound{From: phone, Kind: notify.MenuText, Text: text}).Text
}

func wbIn(h *Harness, in notify.MenuInbound) notify.MenuReply {
	return h.API.HandleWABotMenu(ctxBG(), in)
}

func wbAge(t *testing.T, h *Harness, phone string, d time.Duration) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE wa_bot_chats SET last_activity_at = now() - make_interval(secs => $2)
		WHERE phone = $1`, phone, d.Seconds()); err != nil {
		t.Fatal(err)
	}
}

func wbState(t *testing.T, h *Harness, phone string) (state, step string) {
	t.Helper()
	_ = h.Pool.QueryRow(ctxBG(), `SELECT state, step FROM wa_bot_chats WHERE phone = $1`, phone).Scan(&state, &step)
	return
}

func wbMust(t *testing.T, got, wantPrefix, what string) {
	t.Helper()
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("%s: الردّ %q والمنتظَر يبدأ بـ %q", what, got, wantPrefix)
	}
}

// TestWABOT01_MenuTriggers **أوّلُ رسالة، وبعد السكوت، و«0»/«قائمة» — وردُّ الموظّف يُسكتها.**
func TestWABOT01_MenuTriggers(t *testing.T) {
	h := New(t)
	wbOpen(t, h)
	p := uniqPhone()

	wbMust(t, wbSay(h, p, "مرحبا"), wbMenuHead, "أوّلُ رسالة")
	// **«0» و«قائمة» والرقمُ الهنديّ** تعيدها.
	wbMust(t, wbSay(h, p, "0"), wbMenuHead, "«0»")
	wbMust(t, wbSay(h, p, "٠"), wbMenuHead, "«٠»")
	wbMust(t, wbSay(h, p, "قائمة"), wbMenuHead, "«قائمة»")
	// **قبل السكوت لا تعود** — والردُّ رقمُ خدمة.
	wbAge(t, h, p, 59*time.Minute)
	if r := wbSay(h, p, "3"); strings.HasPrefix(r, wbMenuHead) || !strings.Contains(r, "سيردّ عليك") {
		t.Fatalf("قبل الساعة: %q", r)
	}
	// **ومحادثةُ الإدارة صامتة** حتّى السكوت.
	if r := wbSay(h, p, "متى يصل الطلب؟"); r != "" {
		t.Fatalf("**البوت قاطع محادثة الإدارة**: %q", r)
	}
	// **بعد ساعة سكوت تعود** — حتّى بعد محادثة الإدارة.
	wbAge(t, h, p, 61*time.Minute)
	wbMust(t, wbSay(h, p, "مرحبا"), wbMenuHead, "بعد ساعة")

	// **ردُّ موظّفٍ من هاتف الرقم يُسكت القائمةَ المعروضة** — ثمّ تعود بعد السكوت.
	wbIn(h, notify.MenuInbound{From: p, Kind: notify.MenuText, Text: "أهلاً، تفضّل", Outgoing: true})
	if r := wbSay(h, p, "أريد سؤالاً"); r != "" {
		t.Fatalf("**البوت قاطع الموظّف**: %q", r)
	}
	wbAge(t, h, p, 61*time.Minute)
	wbMust(t, wbSay(h, p, "مرحبا"), wbMenuHead, "بعد ردّ الموظّف والسكوت")

	// **ومدّةُ السكوت إعداد** — عشرُ دقائق.
	h.Setting("whatsapp.bot_menu_silence_min", `10`)
	wbSay(h, p, "3")
	wbAge(t, h, p, 11*time.Minute)
	wbMust(t, wbSay(h, p, "مرحبا"), wbMenuHead, "بعد عشر دقائق بالإعداد")

	// **رسالةٌ عالجها مسارٌ آخر (رمزُ توثيق) نشاطٌ**: لا قائمةَ على التي تليها.
	q := uniqPhone()
	wbIn(h, notify.MenuInbound{From: q, Kind: notify.MenuText, Text: "ABCDEFG", Handled: true})
	if r := wbSay(h, q, "شكراً"); r != "" {
		t.Fatalf("**قائمةٌ بعد رمز التوثيق مباشرةً**: %q", r)
	}
}

// TestWABOT02_RoleExclusions **الكابتن والمندوب والمتجر وموظّفُه بلا قائمة — والمفتاحُ يطفئها.**
func TestWABOT02_RoleExclusions(t *testing.T) {
	h := New(t)
	wbOpen(t, h)
	for _, role := range []string{"driver", "sales", "merchant"} {
		u := h.NewUser(role)
		if r := wbSay(h, u.Phone, "مرحبا"); r != "" {
			t.Fatalf("**%s وصلته القائمة**: %q", role, r)
		}
	}
	owner := h.NewUser("merchant")
	staff := uniqPhone()
	if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO merchant_staff (owner_user_id, phone, name, password_hash)
		VALUES ($1::uuid, $2, 'موظّف', 'x')`, owner.ID, staff); err != nil {
		t.Fatal(err)
	}
	if r := wbSay(h, staff, "مرحبا"); r != "" {
		t.Fatalf("**موظّفُ المتجر وصلته القائمة**: %q", r)
	}
	// **والزبونُ تصله.**
	c := h.Customer()
	wbMust(t, wbSay(h, c.Phone, "مرحبا"), wbMenuHead, "الزبون")
	// **والمفتاحُ مطفأ ⇒ لا شيء.**
	h.Setting("whatsapp.bot_menu_enabled", `false`)
	if r := wbSay(h, uniqPhone(), "مرحبا"); r != "" {
		t.Fatalf("**المفتاح مطفأ والقائمة طلعت**: %q", r)
	}
}

// TestWABOT03_OffHours **خارجَ الدوام: رسالةُ موعد الفتح مكانَ القائمة — مرّةً، و«0» تعيدها.**
func TestWABOT03_OffHours(t *testing.T) {
	h := New(t)
	h.Setting("whatsapp.bot_menu_enabled", `true`)
	h.Setting("whatsapp.bot_menu_silence_min", `60`)
	h.Setting(platform.SwitchKey, `""`)
	h.Setting(platform.EnforcedKey, `true`)
	t.Cleanup(func() { _, _ = h.Pool.Exec(context.Background(), `DELETE FROM platform_hours`) })
	tomorrow := (int(time.Now().In(platform.Location()).Weekday()) + 1) % 7
	setHours(t, h, platform.Window{Day: tomorrow, Start: 9 * 60, End: 10 * 60})

	p := uniqPhone()
	r := wbSay(h, p, "مرحبا")
	if !strings.HasPrefix(r, "أهلاً بك في رحّال غو 🌙") || !strings.Contains(r, "غداً من الساعة 9:00 صباحاً") {
		t.Fatalf("رسالة خارج الدوام: %q", r)
	}
	if r := wbSay(h, p, "1"); r != "" {
		t.Fatalf("**رسالة خارج الدوام تكرّرت**: %q", r)
	}
	if r := wbSay(h, p, "0"); !strings.Contains(r, "غداً") {
		t.Fatalf("«0» خارج الدوام: %q", r)
	}
}

// wbOrderFlow **يمشي طلباً كاملاً** — صوتٌ مرفوض، وتعديلان، ومتجرٌ «لا يوجد»، وموقع. ويردّ رقمَ الطلب.
func wbOrderFlow(t *testing.T, h *Harness, p string) int64 {
	t.Helper()
	wbMust(t, wbSay(h, p, "مرحبا"), wbMenuHead, "القائمة")
	if r := wbSay(h, p, "1️⃣"); r != wbOrderAsk {
		t.Fatalf("الخيار ١: %q", r)
	}
	if r := wbIn(h, notify.MenuInbound{From: p, Kind: notify.MenuMedia}).Text; !strings.HasPrefix(r, wbVoice) ||
		!strings.Contains(r, wbOrderAsk) {
		t.Fatalf("**الصوت**: %q", r)
	}
	if r := wbSay(h, p, "2 شاورما دبل"); r != wbEditQ {
		t.Fatalf("بعد النصّ: %q", r)
	}
	if r := wbSay(h, p, "إي"); r != wbEditAsk {
		t.Fatalf("نعم: %q", r)
	}
	if r := wbSay(h, p, "زيادة بطاطا"); r != wbEditQ {
		t.Fatalf("بعد التعديل: %q", r)
	}
	if r := wbSay(h, p, "ممكن"); !strings.HasPrefix(r, wbUnclear) || !strings.Contains(r, wbYesNo) {
		t.Fatalf("**غيرُ نعم/لا**: %q", r)
	}
	if r := wbSay(h, p, "نعم"); r != wbEditAsk {
		t.Fatalf("نعم ثانية: %q", r)
	}
	if r := wbSay(h, p, "وعلبة كولا"); r != wbEditQ {
		t.Fatalf("التعديل الثاني: %q", r)
	}
	if r := wbSay(h, p, "لأ"); r != wbStoreQ {
		t.Fatalf("لا: %q", r)
	}
	if r := wbSay(h, p, "لا يوجد"); r != wbLocAsk {
		t.Fatalf("المتجر: %q", r)
	}
	if r := wbSay(h, p, "حي الرميلة"); !strings.HasPrefix(r, wbUnclear) || !strings.Contains(r, wbLocAsk) {
		t.Fatalf("**نصٌّ مكان الموقع**: %q", r)
	}
	r := wbIn(h, notify.MenuInbound{From: p, Kind: notify.MenuLocation,
		Loc: notify.InboundLocation{Lat: 35.9506, Lng: 39.0094, Live: true}}).Text
	var n int64
	if _, err := fmt.Sscanf(r, "شكراً لك، تم تسجيل طلبك برقم #%d، وسنتواصل معك قريباً.", &n); err != nil || n <= 0 {
		t.Fatalf("ردُّ التسجيل: %q (%v)", r, err)
	}
	return n
}

// TestWABOT04_OrderFlowPendingNotDispatched **الطلبُ الموجَّه يولد معلّقاً بلا أجرة، ولا يبلغ الكباتن قبل الموافقة.**
func TestWABOT04_OrderFlowPendingNotDispatched(t *testing.T) {
	h := New(t)
	wbOpen(t, h)
	p := uniqPhone()
	n := wbOrderFlow(t, h, p)

	var id, status, note, pickup, rphone string
	var bot bool
	var fee int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT id::text, status, wa_bot, parcel_note, pickup_name, recipient_phone,
		delivery_fee FROM orders WHERE number = $1`, n).Scan(&id, &status, &bot, &note, &pickup, &rphone, &fee); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || !bot || fee != 0 || rphone != p || pickup != orders.WABotNoStore {
		t.Fatalf("**الطلب**: status=%s bot=%v fee=%d phone=%s pickup=%s", status, bot, fee, rphone, pickup)
	}
	for _, want := range []string{"2 شاورما دبل", "زيادة بطاطا", "وعلبة كولا"} {
		if !strings.Contains(note, want) {
			t.Fatalf("**النصّ ناقص %q**: %q", want, note)
		}
	}
	// **القبولُ التلقائيُّ والنزولُ بلا واتساب يتخطّيانه.**
	h.Setting("orders.auto_accept_min", `1`)
	h.Setting("orders.no_whatsapp_dispatch_min", `1`)
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET created_at = now() - interval '5 minutes' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	h.Orders.SweepAutoAcceptOnce(ctxBG())
	h.Orders.SweepNoWhatsAppDispatchOnce(ctxBG())
	_ = h.Pool.QueryRow(ctxBG(), `SELECT status FROM orders WHERE id = $1`, id).Scan(&status)
	if status != "pending" {
		t.Fatalf("**طلبُ البوت قُبل وحدَه**: %s", status)
	}
	// **وزرُّ القبول العامّ يُردّ بلا أجرة.**
	if _, err := h.Orders.Transition(ctxBG(), "", []string{"ops"}, id, orders.StAccepted, ""); err == nil {
		t.Fatal("**قُبل طلبُ البوت بلا أجرة**")
	}
	var offers int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM order_offers WHERE order_id = $1`, id).Scan(&offers)
	if offers != 0 {
		t.Fatalf("**عُرض على كابتن قبل الموافقة**: %d", offers)
	}

	// **في قائمة طلبات الواتساب بعلامته** — ثمّ الموافقةُ بأجرة.
	admin := h.NewUser("admin")
	list := h.GET(waURL, admin.Token)
	if list.Code != http.StatusOK || !strings.Contains(string(list.Body), `"wa_bot":true`) {
		t.Fatalf("القائمة: %d %s", list.Code, list.Body)
	}
	if r := h.POST(waURL+"/"+id+"/approve", admin.Token, map[string]any{"fee": 0}); r.Code == http.StatusOK {
		t.Fatal("**وافق بأجرة صفر**")
	}
	r := h.POST(waURL+"/"+id+"/approve", admin.Token, map[string]any{"fee": 300, "store_name": "فلافل الأمير"})
	if r.Code != http.StatusOK {
		t.Fatalf("الموافقة: %d %s", r.Code, r.Body)
	}
	_ = h.Pool.QueryRow(ctxBG(), `SELECT status, delivery_fee, pickup_name FROM orders WHERE id = $1`, id).
		Scan(&status, &fee, &pickup)
	if status == "pending" || fee != 300 || pickup != "فلافل الأمير" {
		t.Fatalf("**بعد الموافقة**: %s %d %s", status, fee, pickup)
	}
	if r := h.POST(waURL+"/"+id+"/approve", admin.Token, map[string]any{"fee": 300}); r.Code != http.StatusConflict ||
		r.Err() != "wa_bot_not_pending" {
		t.Fatalf("موافقةٌ ثانية: %d %s", r.Code, r.Err())
	}

	// **والرفضُ لطلبٍ آخر بسبب.**
	n2 := wbOrderFlow(t, h, uniqPhone())
	var id2 string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM orders WHERE number = $1`, n2).Scan(&id2)
	if r := h.POST(waURL+"/"+id2+"/reject", admin.Token, map[string]any{"reason": ""}); r.Code == http.StatusOK {
		t.Fatal("**رفضٌ بلا سبب**")
	}
	if r := h.POST(waURL+"/"+id2+"/reject", admin.Token, map[string]any{"reason": "خارج التغطية"}); r.Code != http.StatusOK {
		t.Fatalf("الرفض: %d %s", r.Code, r.Body)
	}
	_ = h.Pool.QueryRow(ctxBG(), `SELECT status FROM orders WHERE id = $1`, id2).Scan(&status)
	if status != "rejected" {
		t.Fatalf("بعد الرفض: %s", status)
	}
}

// TestWABOT05_TrackOwnership **المتابعةُ لصاحب الطلب وحدَه.**
func TestWABOT05_TrackOwnership(t *testing.T) {
	h := New(t)
	wbOpen(t, h)
	owner := uniqPhone()
	n := wbOrderFlow(t, h, owner)

	other := uniqPhone()
	wbSay(h, other, "مرحبا")
	if r := wbSay(h, other, "2"); r != wbTrackAsk {
		t.Fatalf("الخيار ٢: %q", r)
	}
	if r := wbSay(h, other, "رقمي"); !strings.HasPrefix(r, wbUnclear) || !strings.Contains(r, wbTrackAsk) {
		t.Fatalf("**رقمٌ غيرُ عدديّ**: %q", r)
	}
	if r := wbSay(h, other, fmt.Sprint(n)); r != wbNotFound {
		t.Fatalf("**كُشف طلبُ غيره**: %q", r)
	}

	wbSay(h, owner, "0")
	wbSay(h, owner, "٢")
	if r := wbSay(h, owner, "#"+fmt.Sprint(n)); !strings.Contains(r, "قيد المراجعة لدى الإدارة") ||
		!strings.Contains(r, fmt.Sprint(n)) {
		t.Fatalf("متابعةُ صاحبه: %q", r)
	}
}

// TestWABOT06_CaptainSignup **سؤالٌ سؤال، ونعم/لا تُتحقّق، والطلبُ في اللوحة.**
func TestWABOT06_CaptainSignup(t *testing.T) {
	h := New(t)
	wbOpen(t, h)
	p := uniqPhone()
	wbSay(h, p, "مرحبا")
	steps := []struct{ say, want string }{
		{"6", "يرجى كتابة اسمك الكامل."},
		{"علي أحمد", "كم عمرك؟"},
		{"كبير", wbUnclear},
		{"٢٥", "ما المنطقة التي تسكن فيها؟"},
		{"حي الدرعية", "هل لديك دراجة كهربائية؟ (نعم/لا)"},
		{"ربما", wbUnclear},
		{"نعم", "هل لديك دراجة نارية؟ (نعم/لا)"},
		{"لا", "هل لديك رخصة قيادة؟ (نعم/لا)"},
		{"ايوه", "هل لديك جوال أندرويد؟ (نعم/لا)"},
		{"yes", "هل عملت في التوصيل من قبل؟"},
		{"نعم، سنتين", "ما الساعات التي تستطيع العمل فيها؟"},
		{"من 9 صباحاً حتى 5 مساءً", wbSignupDone},
	}
	for _, s := range steps {
		if r := wbSay(h, p, s.say); !strings.HasPrefix(r, s.want) {
			t.Fatalf("«%s»: الردّ %q والمنتظَر %q", s.say, r, s.want)
		}
	}
	var id, name, age string
	var ebike, moto, lic, android bool
	if err := h.Pool.QueryRow(ctxBG(), `SELECT id::text, full_name, age, has_ebike, has_motorbike, has_license, has_android
		FROM captain_applications WHERE phone = $1`, p).Scan(&id, &name, &age, &ebike, &moto, &lic, &android); err != nil {
		t.Fatal(err)
	}
	if name != "علي أحمد" || age != "25" || !ebike || moto || !lic || !android {
		t.Fatalf("**الأجوبة**: %s %s %v %v %v %v", name, age, ebike, moto, lic, android)
	}
	admin := h.NewUser("admin")
	if r := h.GET("/api/v1/admin/captain-applications?status=new", admin.Token); r.Code != http.StatusOK ||
		!strings.Contains(string(r.Body), id) {
		t.Fatalf("قائمةُ اللوحة: %d %s", r.Code, r.Body)
	}
	if r := h.POST("/api/v1/admin/captain-applications/"+id+"/status", admin.Token,
		map[string]any{"status": "contacted"}); r.Code != http.StatusOK {
		t.Fatalf("تبديلُ الحال: %d %s", r.Code, r.Body)
	}
	// **ولا يراها زبون.**
	if r := h.GET("/api/v1/admin/captain-applications", h.Customer().Token); r.Code == http.StatusOK {
		t.Fatal("**زبونٌ قرأ طلبات التسجيل**")
	}
}

// wbBot **بوتٌ يسجّل ما يُرسَل** — للتذكير.
type wbBot struct {
	mu   sync.Mutex
	sent []string
}

func (b *wbBot) SendText(_ context.Context, phone, text string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sent = append(b.sent, phone+"|"+text)
	return nil
}
func (b *wbBot) Ready() bool { return true }
func (b *wbBot) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.sent)
}

// TestWABOT07_ReminderThenSilentDrop **تذكيرٌ مرّةً بعد ثلاث دقائق، ثمّ إسقاطٌ صامت.**
func TestWABOT07_ReminderThenSilentDrop(t *testing.T) {
	h := New(t)
	wbOpen(t, h)
	bot := &wbBot{}
	h.API.SetMerchantNotifier(bot)
	t.Cleanup(func() { h.API.SetMerchantNotifier(nil) })

	p := uniqPhone()
	wbSay(h, p, "مرحبا")
	wbSay(h, p, "1")
	wbSay(h, p, "طلبي")
	// **قبل الثلاث لا تذكير.**
	h.API.SweepWABotMenuOnce(ctxBG())
	if bot.count() != 0 {
		t.Fatalf("**تذكيرٌ قبل وقته**: %v", bot.sent)
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE wa_bot_chats SET step_at = now() - interval '4 minutes' WHERE phone = $1`, p); err != nil {
		t.Fatal(err)
	}
	if n := h.API.SweepWABotMenuOnce(ctxBG()); n != 1 || bot.count() != 1 || bot.sent[0] != p+"|"+wbReminder {
		t.Fatalf("التذكير: %d %v", n, bot.sent)
	}
	h.API.SweepWABotMenuOnce(ctxBG())
	if bot.count() != 1 {
		t.Fatalf("**تذكيرٌ ثانٍ**: %v", bot.sent)
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE wa_bot_chats SET reminded_at = now() - interval '4 minutes' WHERE phone = $1`, p); err != nil {
		t.Fatal(err)
	}
	h.API.SweepWABotMenuOnce(ctxBG())
	if st, _ := wbState(t, h, p); st != "idle" {
		t.Fatalf("**لم يُسقَط**: %s", st)
	}
	if bot.count() != 1 {
		t.Fatalf("**الإسقاطُ أرسل شيئاً**: %v", bot.sent)
	}
	// **وبعدها صمتٌ حتّى السكوت أو «0».**
	if r := wbSay(h, p, "لا"); r != "" {
		t.Fatalf("**ردٌّ بعد الإسقاط**: %q", r)
	}
	wbMust(t, wbSay(h, p, "0"), wbMenuHead, "«0» بعد الإسقاط")

	// **والجوابُ بعد التذكير يُكمل ويصفّره.**
	wbSay(h, p, "1")
	_, _ = h.Pool.Exec(ctxBG(), `UPDATE wa_bot_chats SET step_at = now() - interval '4 minutes' WHERE phone = $1`, p)
	h.API.SweepWABotMenuOnce(ctxBG())
	if r := wbSay(h, p, "طلبي الثاني"); r != wbEditQ {
		t.Fatalf("الجواب بعد التذكير: %q", r)
	}
	var reminded bool
	_ = h.Pool.QueryRow(ctxBG(), `SELECT reminded_at IS NOT NULL FROM wa_bot_chats WHERE phone = $1`, p).Scan(&reminded)
	if reminded {
		t.Fatal("**التذكير لم يُصفَّر بعد الجواب**")
	}
}

// TestWABOT08_UnclearNoLoop **غيرُ المفهوم: توضيحٌ مرّتين، ثمّ «0 أو 3»، ثمّ صمت — والجوابُ الصحيح يصفّره.**
func TestWABOT08_UnclearNoLoop(t *testing.T) {
	h := New(t)
	wbOpen(t, h)
	p := uniqPhone()
	wbSay(h, p, "مرحبا")
	for i := 0; i < 2; i++ {
		if r := wbSay(h, p, "سلام"); !strings.HasPrefix(r, wbUnclear) ||
			!strings.Contains(r, "يرجى إرسال رقم الخدمة من 1 إلى 7، أو 0 لعرض القائمة.") {
			t.Fatalf("التوضيح %d: %q", i+1, r)
		}
	}
	if r := wbSay(h, p, "؟"); r != wbStuck {
		t.Fatalf("الثالثة: %q", r)
	}
	if r := wbSay(h, p, "مرحبا؟"); r != "" {
		t.Fatalf("**البوت يدور**: %q", r)
	}
	if r := wbSay(h, p, "9"); r != "" {
		t.Fatalf("**رقمٌ خارج القائمة بعد الصمت**: %q", r)
	}
	if r := wbSay(h, p, "2"); r != wbTrackAsk {
		t.Fatalf("الجواب الصحيح بعد الصمت: %q", r)
	}
	if r := wbSay(h, p, "x"); !strings.HasPrefix(r, wbUnclear) {
		t.Fatalf("**العدّاد لم يُصفَّر**: %q", r)
	}
}

// TestWABOT09_ResetAndApp **استعادةُ كلمة المرور بالمسار الحقيقيّ، والتطبيقُ بلا ملف.**
func TestWABOT09_ResetAndApp(t *testing.T) {
	h := New(t)
	wbOpen(t, h)
	c := h.Customer()
	wbSay(h, c.Phone, "مرحبا")
	if r := wbSay(h, c.Phone, "4"); r != "أرسلنا إليك رمز استعادة كلمة المرور، أدخله في التطبيق." {
		t.Fatalf("صاحبُ حساب: %q", r)
	}
	var codes int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM otp_codes WHERE phone = $1 AND purpose = 'reset'`, c.Phone).Scan(&codes)
	if codes == 0 {
		t.Fatal("**لم يُولَّد رمزُ استعادة**")
	}
	p := uniqPhone()
	wbSay(h, p, "مرحبا")
	if r := wbSay(h, p, "4"); r != "هذا الرقم غير مسجّل لدينا. يمكنك إنشاء حساب من تطبيق رحّال غو." {
		t.Fatalf("بلا حساب: %q", r)
	}
	h.Setting("release.customer.apk", `""`)
	h.Setting("release.customer.play_url", `""`)
	q := uniqPhone()
	wbSay(h, q, "مرحبا")
	if r := wbIn(h, notify.MenuInbound{From: q, Kind: notify.MenuText, Text: "5"}); r.SendApp ||
		!strings.Contains(r.Text, "غير متاح") {
		t.Fatalf("التطبيق بلا ملف: %+v", r)
	}
}
