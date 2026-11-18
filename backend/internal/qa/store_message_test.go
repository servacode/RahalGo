package qa

// **«رسالة للمتاجر»** (قرارُ المالك ٢٠٢٦-١١-١٨): واتساب + إشعارُ التطبيق لكلّ المتاجر الفعّالة، يدويّاً
// أو مجدوَلاً — **بفاصلٍ يحمي رقمَ البوت، ولا رسالتين لمتجرٍ باليوم.**
//
// **والبوتُ هنا مزيّفٌ والفاصلُ مُحقَن** — لا نومَ في الفحوص ولا واتساب حقيقيّ.

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/server"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

// smBot بوتٌ مزيّفٌ يسجّل ما أُرسل.
type smBot struct {
	mu    sync.Mutex
	ready bool
	err   error
	// onSend يُنادى بعد كلّ إرسال (لإسقاط الجاهزيّة في منتصف التشغيلة).
	onSend func(b *smBot)
	sent   []string // الأرقام بالترتيب
	texts  []string
}

func (b *smBot) Ready() bool { b.mu.Lock(); defer b.mu.Unlock(); return b.ready }
func (b *smBot) SendText(_ context.Context, phone, text string) error {
	b.mu.Lock()
	b.sent = append(b.sent, phone)
	b.texts = append(b.texts, text)
	err, hook := b.err, b.onSend
	b.mu.Unlock()
	if hook != nil {
		hook(b)
	}
	return err
}
func (b *smBot) setReady(v bool) { b.mu.Lock(); b.ready = v; b.mu.Unlock() }
func (b *smBot) count(phone string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := 0
	for _, p := range b.sent {
		if p == phone {
			n++
		}
	}
	return n
}
func (b *smBot) total() int { b.mu.Lock(); defer b.mu.Unlock(); return len(b.sent) }

func smDamascus(year, month, day, hour int) time.Time {
	return time.Date(year, time.Month(month), day, hour, 5, 0, 0, time.FixedZone("Damascus", 3*3600))
}

// smArm يجهّز الخادمَ: بوتٌ مزيّف، ساعةٌ ثابتة، فاصلٌ صفريّ، وتنظيفُ السجلّ قبل وبعد.
func smArm(t *testing.T, h *Harness, at time.Time) *smBot {
	t.Helper()
	bot := &smBot{ready: true}
	h.API.SetMerchantNotifier(bot)
	h.API.SetStoreMessageClockForTest(func() time.Time { return at })
	h.API.SetStoreMessageGapForTest(func() time.Duration { return 0 })
	clean := func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM store_message_runs`) }
	clean()
	t.Cleanup(func() {
		h.API.SetMerchantNotifier(nil)
		h.API.SetStoreMessageClockForTest(nil)
		h.API.SetStoreMessageGapForTest(nil)
		clean()
	})
	return bot
}

// smDrain يرسل كلَّ ما ينتظر (كما يفعل العاملُ بلا فاصل).
func smDrain(h *Harness) {
	for i := 0; i < 5000 && h.API.StoreMessageStepForTest(ctxBG()); i++ {
	}
}

func smRun(t *testing.T, h *Harness, id string) (status, reason string, target, sent, failed int) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT status, stop_reason, target_count, sent_count, failed_count
		FROM store_message_runs WHERE id = $1::uuid`, id).Scan(&status, &reason, &target, &sent, &failed); err != nil {
		t.Fatal(err)
	}
	return
}

func smDeliveries(t *testing.T, h *Harness, merchantID string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM store_message_deliveries WHERE merchant_id = $1::uuid`, merchantID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// SM01 — التشغيلةُ اليدويّةُ تصل المتاجرَ المؤهَّلةَ وحدَها، بالنصّ الافتراضيّ، وإشعارُ التطبيق معها.
func TestSM01_ManualRunReachesEligibleStoresOnly(t *testing.T) {
	h := New(t)
	bot := smArm(t, h, smDamascus(2033, 5, 4, 11))
	f := h.Factory()
	ok := f.Merchant()
	suspended := f.Merchant(MerchantSuspended())
	ownerOff := f.Merchant()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET status = 'suspended' WHERE id = $1::uuid`, ownerOff.Owner.ID); err != nil {
		t.Fatal(err)
	}
	// **خطٌّ أرضيٌّ لا يصلح لواتساب** — يُتخطّى بصمت.
	landline := f.Merchant()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET phone = '0221234567' WHERE id = $1::uuid`, landline.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET phone = '+963211234567', whatsapp_phone = NULL, whatsapp_verified_at = NULL WHERE id = $1::uuid`, landline.Owner.ID); err != nil {
		t.Skipf("لا يُقبل هاتفٌ أرضيٌّ للمستخدم في هذه القاعدة: %v", err)
	}

	runID, n, err := h.API.StoreMessageRunForTest(ctxBG(), "manual", "")
	if err != nil || n < 1 || runID == "" {
		t.Fatalf("**يجب أن تُحجز تسليماتٌ**: n=%d err=%v", n, err)
	}
	smDrain(h)

	if bot.count(ok.Owner.Phone) != 1 {
		t.Fatalf("**المتجرُ المؤهَّل لم يصله مرّةً واحدة**: %d", bot.count(ok.Owner.Phone))
	}
	for name, m := range map[string]*Merchant{"موقوف": suspended, "صاحبُه موقوف": ownerOff, "خطٌّ أرضيّ": landline} {
		if smDeliveries(t, h, m.ID) != 0 || bot.count(m.Owner.Phone) != 0 {
			t.Errorf("**أُرسل لمتجرٍ غيرِ مؤهَّل (%s)**", name)
		}
	}
	// **والنصُّ هو الافتراضيُّ حرفاً بحرف.**
	if got := bot.texts[0]; got != settings.StoreMessageDefaultText || !strings.HasPrefix(got, "مرحباً شريكنا العزيز") {
		t.Fatalf("النصُّ ليس الافتراضيّ: %q", got)
	}
	// **وإشعارُ تطبيق المتجر لصاحب المتجر المؤهَّل وحدَه.**
	if in := obInbox(t, h, ok.Owner.ID); len(in) != 1 || in[0] != "تحديث أسعار أصنافك" {
		t.Fatalf("إشعارُ التطبيق: %v", in)
	}
	if in := obInbox(t, h, suspended.Owner.ID); len(in) != 0 {
		t.Fatalf("إشعارٌ لمتجرٍ موقوف: %v", in)
	}
	if st, _, target, sent, failed := smRun(t, h, runID); st != "done" || target != n || sent != n || failed != 0 {
		t.Fatalf("حالُ التشغيلة: %s target=%d sent=%d failed=%d (n=%d)", st, target, sent, failed, n)
	}
}

// SM02 — لا رسالتين لمتجرٍ في يومٍ واحد: ضغطتان، أو آليٌّ فوق يدويّ — والغدُ يصل.
func TestSM02_NoDuplicateSameDamascusDay(t *testing.T) {
	h := New(t)
	now := smDamascus(2033, 5, 5, 11)
	bot := smArm(t, h, now)
	m := h.Factory().Merchant()

	if _, n, err := h.API.StoreMessageRunForTest(ctxBG(), "manual", ""); err != nil || n < 1 {
		t.Fatalf("الأولى: n=%d err=%v", n, err)
	}
	smDrain(h)
	// **الضغطةُ الثانيةُ: لا حجزَ ولا تشغيلةٌ فارغةٌ في السجلّ.**
	runs := 0
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM store_message_runs`).Scan(&runs)
	if id, n, err := h.API.StoreMessageRunForTest(ctxBG(), "manual", ""); err != nil || n != 0 || id != "" {
		t.Fatalf("**الثانيةُ حجزت**: id=%q n=%d err=%v", id, n, err)
	}
	var runs2 int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM store_message_runs`).Scan(&runs2)
	if runs2 != runs {
		t.Fatalf("تشغيلةٌ فارغةٌ دخلت السجلّ: %d → %d", runs, runs2)
	}
	smDrain(h)
	if bot.count(m.Owner.Phone) != 1 || smDeliveries(t, h, m.ID) != 1 {
		t.Fatalf("**وصله أكثرُ من مرّة**: رسائل=%d صفوف=%d", bot.count(m.Owner.Phone), smDeliveries(t, h, m.ID))
	}
	// **والحارسُ في القاعدة لا في الشيفرة**: إدراجٌ مباشرٌ ثانٍ لنفس (متجر، يوم) يُردّ.
	var runID string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM store_message_runs LIMIT 1`).Scan(&runID)
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO store_message_deliveries (run_id, merchant_id, phone, day)
		VALUES ($1::uuid, $2::uuid, 'x', '2033-05-05')`, runID, m.ID); err == nil {
		t.Fatal("**القاعدةُ قبلت تسليماً ثانياً لنفس المتجر في نفس اليوم**")
	}
	// **ويومُ دمشق لا يومُ UTC**: ١١ مساءً بدمشق هو نفسُ اليوم، وبعد منتصف الليل يومٌ جديد.
	h.API.SetStoreMessageClockForTest(func() time.Time { return smDamascus(2033, 5, 5, 23) })
	if _, n, _ := h.API.StoreMessageRunForTest(ctxBG(), "manual", ""); n != 0 {
		t.Fatalf("١١ مساءً بدمشق نفسُ اليوم: n=%d", n)
	}
	h.API.SetStoreMessageClockForTest(func() time.Time { return smDamascus(2033, 5, 6, 0) })
	if _, n, _ := h.API.StoreMessageRunForTest(ctxBG(), "manual", ""); n < 1 {
		t.Fatalf("بعد منتصف الليل بدمشق يومٌ جديد: n=%d", n)
	}
	smDrain(h)
	if bot.count(m.Owner.Phone) != 2 {
		t.Fatalf("اليومُ التالي لم يصل: %d", bot.count(m.Owner.Phone))
	}
}

// SM03 — «تجربة»: متجرٌ واحدٌ بعينه وحدَه، ولا تحجز اليومَ عن الإرسال الحقيقيّ.
func TestSM03_TestSendHitsExactlyOneStore(t *testing.T) {
	h := New(t)
	bot := smArm(t, h, smDamascus(2033, 5, 7, 12))
	f := h.Factory()
	target := f.Merchant()
	other := f.Merchant()
	landline := f.Merchant()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET phone = '0221234567' WHERE id = $1::uuid`, landline.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET phone = '+963211234567', whatsapp_phone = NULL, whatsapp_verified_at = NULL WHERE id = $1::uuid`, landline.Owner.ID); err != nil {
		t.Skipf("هاتفٌ أرضيّ غيرُ مقبول: %v", err)
	}

	if _, _, err := h.API.StoreMessageRunForTest(ctxBG(), "test", landline.ID); err == nil {
		t.Fatal("**تجربةٌ لمتجرٍ بلا رقمٍ صالحٍ يجب أن تُردّ**")
	}
	_, n, err := h.API.StoreMessageRunForTest(ctxBG(), "test", target.ID)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	smDrain(h)
	if bot.total() != 1 || bot.count(target.Owner.Phone) != 1 || bot.count(other.Owner.Phone) != 0 {
		t.Fatalf("**التجربةُ يجب أن تصيب متجراً واحداً فقط**: المجموعُ=%d", bot.total())
	}
	// **والتجربةُ لا تَسُدّ الإرسالَ الحقيقيَّ لنفس المتجر اليوم.**
	if _, n, err := h.API.StoreMessageRunForTest(ctxBG(), "manual", ""); err != nil || n < 1 {
		t.Fatalf("الإرسالُ الحقيقيّ بعد التجربة: n=%d err=%v", n, err)
	}
	smDrain(h)
	if bot.count(target.Owner.Phone) != 2 || bot.count(other.Owner.Phone) != 1 {
		t.Fatalf("بعد التجربة: هدف=%d آخر=%d", bot.count(target.Owner.Phone), bot.count(other.Owner.Phone))
	}
}

// SM04 — البوتُ غيرُ جاهز: التشغيلةُ تتوقّف ولا يُرسل شيء، وتنبيهٌ للإدارة، ويُعاد بعد رجوعه.
func TestSM04_BotNotReadyStopsRun(t *testing.T) {
	h := New(t)
	bot := smArm(t, h, smDamascus(2033, 5, 8, 12))
	f := h.Factory()
	ops := f.NewUserWith("operations")
	m := f.Merchant()

	runID, n, err := h.API.StoreMessageRunForTest(ctxBG(), "manual", "")
	if err != nil || n < 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	bot.setReady(false)
	smDrain(h)
	if bot.total() != 0 {
		t.Fatalf("**أُرسل والبوتُ غيرُ جاهز**: %d", bot.total())
	}
	st, reason, _, sent, _ := smRun(t, h, runID)
	if st != "stopped" || reason != "bot_not_ready" || sent != 0 {
		t.Fatalf("حالُ التشغيلة: %s/%s sent=%d", st, reason, sent)
	}
	// **وما لم يُرسل يُحرَّر** — فيُعاد بعد رجوع البوت، لا يبقى محجوزاً كأنّه وصل.
	if smDeliveries(t, h, m.ID) != 0 {
		t.Fatal("**بقي تسليمٌ معلَّقٌ بعد التوقّف**")
	}
	if in := obInbox(t, h, ops.ID); len(in) == 0 || !strings.Contains(in[len(in)-1], "توقّفت رسالة المتاجر") {
		t.Fatalf("**لا تنبيهَ للإدارة**: %v", in)
	}
	bot.setReady(true)
	if _, n, err := h.API.StoreMessageRunForTest(ctxBG(), "manual", ""); err != nil || n < 1 {
		t.Fatalf("إعادةُ المحاولة بعد رجوع البوت: n=%d err=%v", n, err)
	}
	smDrain(h)
	if bot.count(m.Owner.Phone) != 1 {
		t.Fatalf("لم يصل بعد رجوع البوت: %d", bot.count(m.Owner.Phone))
	}
}

// SM05 — البوتُ يسقط في منتصف التشغيلة: ما أُرسل يبقى، والباقي يتوقّف، ولا إرسالَ على قناةٍ مقطوعة.
func TestSM05_BotDropsMidRun(t *testing.T) {
	h := New(t)
	bot := smArm(t, h, smDamascus(2033, 5, 9, 12))
	f := h.Factory()
	f.Merchant()
	f.Merchant()
	f.Merchant()
	bot.onSend = func(b *smBot) { b.setReady(false) }
	bot.err = context.DeadlineExceeded

	runID, n, err := h.API.StoreMessageRunForTest(ctxBG(), "manual", "")
	if err != nil || n < 3 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	smDrain(h)
	if bot.total() != 1 {
		t.Fatalf("**يجب أن تقف بعد أوّل فشلٍ والبوتُ مقطوع**: أُرسل %d", bot.total())
	}
	st, reason, _, sent, failed := smRun(t, h, runID)
	if st != "stopped" || reason != "bot_not_ready" || sent != 0 || failed != 1 {
		t.Fatalf("حالُ التشغيلة: %s/%s sent=%d failed=%d", st, reason, sent, failed)
	}
}

// SM06 — فشلُ إرسالٍ واحدٍ (والبوتُ جاهز) لا يوقف الباقي ولا يُعاد.
func TestSM06_SingleFailureDoesNotStopRun(t *testing.T) {
	h := New(t)
	bot := smArm(t, h, smDamascus(2033, 5, 10, 12))
	f := h.Factory()
	a, b := f.Merchant(), f.Merchant()
	bot.err = context.DeadlineExceeded
	runID, n, err := h.API.StoreMessageRunForTest(ctxBG(), "manual", "")
	if err != nil || n < 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	smDrain(h)
	if bot.count(a.Owner.Phone) != 1 || bot.count(b.Owner.Phone) != 1 {
		t.Fatalf("**كلُّ متجرٍ يُحاوَل مرّةً واحدة**: %d/%d", bot.count(a.Owner.Phone), bot.count(b.Owner.Phone))
	}
	if st, _, _, sent, failed := smRun(t, h, runID); st != "done" || sent != 0 || failed != n {
		t.Fatalf("%s sent=%d failed=%d", st, sent, failed)
	}
}

// SM07 — المجدوِل: يطلق في ساعته ويومه فقط، ومرّةً واحدةً باليوم، ومطفأً لا يطلق.
func TestSM07_SchedulerFiresOnlyAtConfiguredHourAndDay(t *testing.T) {
	h := New(t)
	smArm(t, h, smDamascus(2033, 5, 11, 10))
	h.Factory().Merchant()
	runs := func() int {
		var n int
		_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM store_message_runs WHERE trigger = 'auto'`).Scan(&n)
		return n
	}
	wed := smDamascus(2033, 5, 11, 10) // الأربعاء
	if wed.Weekday() != time.Wednesday {
		t.Fatalf("افتراضُ الاختبار: %v", wed.Weekday())
	}

	// مطفأ (الافتراضيّ) ⇒ لا شيء.
	if h.API.StoreMessageSchedTickForTest(ctxBG(), wed) || runs() != 0 {
		t.Fatal("**أطلق والجدولةُ مطفأة**")
	}
	h.Setting("store_message.auto_enabled", "true")
	h.Setting("store_message.auto_frequency", `"weekly"`)
	h.Setting("store_message.auto_weekday", "3")
	h.Setting("store_message.auto_hour", "10")

	if h.API.StoreMessageSchedTickForTest(ctxBG(), smDamascus(2033, 5, 11, 9)) || runs() != 0 {
		t.Fatal("**أطلق قبل ساعته**")
	}
	if h.API.StoreMessageSchedTickForTest(ctxBG(), smDamascus(2033, 5, 12, 10)) || runs() != 0 {
		t.Fatal("**أطلق في غير يوم الأسبوع**")
	}
	if !h.API.StoreMessageSchedTickForTest(ctxBG(), wed) || runs() != 1 {
		t.Fatal("**لم يُطلق في ساعته ويومه**")
	}
	// **ونبضاتُ بقيّة الساعة لا تُعيد.**
	if h.API.StoreMessageSchedTickForTest(ctxBG(), wed.Add(30*time.Minute)) || runs() != 1 {
		t.Fatal("**أُطلق مرّتين في اليوم**")
	}
	// **والأسبوعُ التالي يُطلق.**
	if !h.API.StoreMessageSchedTickForTest(ctxBG(), wed.AddDate(0, 0, 7)) || runs() != 2 {
		t.Fatal("لم يُطلق الأسبوعَ التالي")
	}
	// **ويوميٌّ: أيَّ يومٍ بساعته.**
	h.Setting("store_message.auto_frequency", `"daily"`)
	if !h.API.StoreMessageSchedTickForTest(ctxBG(), smDamascus(2033, 5, 13, 10)) || runs() != 3 {
		t.Fatal("اليوميُّ لم يُطلق")
	}
}

// SM08 — الفاصلُ العشوائيُّ بين ٢٠ و٤٥ ثانية، وقابلٌ للحقن.
func TestSM08_GapIsRandom20To45AndInjectable(t *testing.T) {
	h := New(t)
	seen := map[time.Duration]bool{}
	for i := 0; i < 300; i++ {
		d := server.StoreMessageGapRandomForTest()
		if d < 20*time.Second || d > 45*time.Second {
			t.Fatalf("الفاصلُ خارجَ ٢٠–٤٥ ثانية: %v", d)
		}
		seen[d] = true
	}
	if len(seen) < 20 {
		t.Fatalf("الفاصلُ ليس عشوائيّاً: %d قيمةً مختلفة", len(seen))
	}
	h.API.SetStoreMessageGapForTest(func() time.Duration { return 7 * time.Second })
	t.Cleanup(func() { h.API.SetStoreMessageGapForTest(nil) })
	if h.API.StoreMessageGapForTest() != 7*time.Second {
		t.Fatal("الفاصلُ لا يقبل الحقن")
	}
}

// SM09 — إعادةُ التشغيل: ما كان قيد الإرسال يُوسم فاشلاً ولا يُعاد، وما ينتظر يكمل.
func TestSM09_RestartMarksSendingFailedAndKeepsPending(t *testing.T) {
	h := New(t)
	bot := smArm(t, h, smDamascus(2033, 5, 14, 12))
	f := h.Factory()
	a, b := f.Merchant(), f.Merchant()
	runID, n, err := h.API.StoreMessageRunForTest(ctxBG(), "manual", "")
	if err != nil || n < 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	// كأنّ الخادم سقط وسط إرسالِ متجر a.
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE store_message_deliveries SET status = 'sending' WHERE merchant_id = $1::uuid`, a.ID); err != nil {
		t.Fatal(err)
	}
	h.API.StoreMessageRecoverForTest(ctxBG())
	smDrain(h)
	if bot.count(a.Owner.Phone) != 0 {
		t.Fatal("**أُعيد إرسالُ ما انقطع وسطَه** — قد يكون وصل")
	}
	if bot.count(b.Owner.Phone) != 1 {
		t.Fatal("ما كان ينتظر لم يكمل بعد الإقلاع")
	}
	if st, _, _, sent, failed := smRun(t, h, runID); st != "done" || sent != n-1 || failed != 1 {
		t.Fatalf("%s sent=%d failed=%d", st, sent, failed)
	}
}
