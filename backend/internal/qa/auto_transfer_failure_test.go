package qa

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// ══════════════════════════════════════════════════════════════════════
// **أنماطُ فشل التحويل السبعة** — `R24` · `F-20`
// ══════════════════════════════════════════════════════════════════════
//
// # العقدُ الكانونيّ
//
//	فشلُ التحويل بلا أثرٍ يُقرأ — عمليّات · التدفّقُ: التحويل
//	نافذةُ الفشل المُثبَتة: نمطان من سبعة · ٥ من ٧ مرئيّة
//
// (`FINAL_STATIC_CLOSEOUT.md` سطرُ `R24` · والجردُ في
// `INDEPENDENT_REVIEW_RECONCILIATION_5.md` تحت `R5-9`.)
//
// # و«التحويل» هنا فعلٌ بعينه
//
// **إرسالُ الطلب إلى المتجر تلقائيّاً** (`autoTransfer`) — **لا نقلُ
// طلبٍ بين سائقين ولا حوالةٌ ماليّة.** **وهو ثلاثةُ أفعالٍ لا واحد**:
// **قبولٌ · وإبلاغُ المتجر · وإنزالٌ إلى السائقين.**
//
// # وما يجعل نمطاً «مرئيّاً»
//
// **أن تبقى في القاعدة حالٌ تقول ما جرى بعد انتهاء الطلب** —
// **والسجلُّ ليس حقيقةً تشغيليّة**: لا يُقرأ في لوحةٍ ولا يُستدرَك منه.
//
// **والأنماطُ الأربعةُ الأُوَل تُبقي الطلبَ `pending`** — وهي حالٌ
// يراها المكتبُ ويرصدها الراصد. **والسابعُ يُبقيه `accepted` مُبلَّغاً
// بلا سائق** — وله زرٌّ. **والخامسُ والسادسُ هما السؤال.**

// fakeNotifier مُبلِّغُ متاجرَ يُتحكَّم به — **جاهزيّةٌ ونتيجةٌ مقيستان.**
//
// **ولا يُحقَن إلّا في فحصٍ يقيس نمطاً** — والخادمُ يُعاد إلى ما كان.
type fakeNotifier struct {
	ready bool
	err   error
	calls int
}

func (f *fakeNotifier) Ready() bool { return f.ready }
func (f *fakeNotifier) SendText(context.Context, string, string) error {
	f.calls++
	return f.err
}

// transferProbe حالُ الطلب بعد أن تهدأ محاولةُ التحويل.
type transferProbe struct {
	Status   string
	SentAt   *time.Time
	Events   int
	Audits   int
	Assigned bool
}

// armTransfer يهيّئ وضعَ المنصّة ويشغّل التحويلَ التلقائيّ.
func armTransfer(t *testing.T, hh *Harness, n *fakeNotifier) *Merchant {
	t.Helper()
	hh.Setting("platform.orders_mode", `"platform"`)
	hh.Setting("orders.auto_transfer", "true")
	hh.API.SetMerchantNotifier(n)
	t.Cleanup(func() { hh.API.SetMerchantNotifier(nil) })
	// **والوضعُ يُقرأ من القاعدة عند كلّ نداء** — **ومفتاحٌ عامٌّ
	// تكتبه فحوصٌ أخرى**، فيُثبَت أنّه استقرّ قبل أن يُنشأ طلب.
	// **وإلّا قِيس نمطٌ غيرُ الذي أُريد** بلا أن يُعلَم.
	var mode string
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT value::text FROM app_settings
		  WHERE key = 'platform.orders_mode'`).Scan(&mode); err != nil {
		t.Fatalf("قراءةُ وضع الطلبات: %v", err)
	}
	if mode != `"platform"` {
		t.Fatalf("**وضعُ الطلبات %s لا `platform`** — والنمطُ لا يُقاس في غير وضعه", mode)
	}
	return hh.Factory().Merchant()
}

// placeTransferOrder ينشئ طلباً ويعيد معرّفَه.
func placeTransferOrder(t *testing.T, hh *Harness, m *Merchant) string {
	t.Helper()
	item := hh.NewItemFor(m, 1500)
	made := hh.POSTKey("/api/v1/orders", hh.Customer().Token, uniq("r24"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	id, _ := made.JSON()["id"].(string)
	return id
}

// settleTransfer ينتظر حتّى **تنتهي** محاولةُ التحويل ثمّ يقرأ الحال.
//
// ══════════════════════════════════════════════════════════════════════
// **و«قُبل» ليست نهايةَ المحاولة** — `XG-44`
// ══════════════════════════════════════════════════════════════════════
//
// **كان الشرطُ `status != 'pending'`** — **وهو يتحقّق عند أوّلِ الأفعال
// الثلاثة لا عند آخرها.** **والمحاولةُ خيطٌ مستقلٌّ** يُطلَق بـ
// `go s.autoTransfer(context.WithoutCancel(...))`، **وترتيبُه في
// `auto_transfer.go` مقيس**:
//
//	١ Transition ⇒ accepted     ← **هنا كان الفحصُ يقرأ**
//	٢ SendText                  ← عدّادُ النداء
//	٣ sent_to_merchant_at       ← الوسم
//	٤ AutoDispatch
//
// **فيُقرأ منتصفُ عمليّةٍ ويُحكَم عليه**: **`نداءات=0`** إن قُرئ بين ١
// و٢ (وهو `F5`)، **و`مُرسَلٌ=false` ونداءٌ واحد** إن قُرئ بين ٢ و٣
// (وهو `F7`). **وكلاهما عرَضُ قراءةٍ مبكّرةٍ لا عطبُ منتَج.**
//
// **وقيس منفرداً بلا حزمةٍ ولا سلسلة**: عشرون تشغيلاً لكلٍّ ⇒
// **`F5` أربعُ سقطات · `F7` ثلاث.** **فلا تلوّثَ ولا سابقة.**
//
// # فيُنتظَر آخرُ الأفعال لا أوّلُها
//
// **والنهايةُ حالٌ في القاعدة لا مهلةُ ساعة**: **وسمٌ** أو **أثرُ فشلٍ
// منسوبٌ إلى الطلب** أو **سائقٌ أُسنِد.** **وما بقي `pending` فلم
// تبدأ محاولتُه أو رُفض** — وتلك حالُ الأنماط الأربعة الأُوَل، **وهي
// تُقرأ بعد المهلة كما كانت.**
//
// **ولا نومَ أُضيف ولا مهلةٌ رُفعت** — **بُدّل الشرطُ وحدَه.**
func settleTransfer(t *testing.T, hh *Harness, oid string) transferProbe {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var p transferProbe
	for {
		var failed int
		if err := hh.Pool.QueryRow(ctxBG(), `
			SELECT o.status, o.sent_to_merchant_at,
			       (SELECT count(*) FROM order_events e WHERE e.order_id = o.id),
			       (SELECT count(*) FROM audit_log a WHERE a.entity = 'order'
			                                          AND a.entity_id = o.id::text),
			       o.driver_id IS NOT NULL,
			       (SELECT count(*) FROM order_events e
			         WHERE e.order_id = o.id AND e.note LIKE $2 || '%')
			  FROM orders o WHERE o.id = $1::uuid`, oid, orders.TransferFailureNote).
			Scan(&p.Status, &p.SentAt, &p.Events, &p.Audits, &p.Assigned, &failed); err != nil {
			t.Fatalf("قراءةُ الطلب: %v", err)
		}
		if transferAttemptEnded(p, failed) || time.Now().After(deadline) {
			return p
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// transferAttemptEnded **هل بلغت المحاولةُ آخرَها؟**
//
// **و`accepted` وحدَها لا تكفي** — **هي أوّلُ الثلاثة لا آخرُها.**
func transferAttemptEnded(p transferProbe, failed int) bool {
	if p.Status == "pending" {
		return false
	}
	return p.SentAt != nil || failed > 0 || p.Assigned
}

// ══════════════════════════════════════════════════════════════════════
// **F1…F4 · أربعةٌ تُبقي الطلبَ بيد المكتب**
// ══════════════════════════════════════════════════════════════════════
//
// **والمرئيّةُ هنا ليست رسالةً** — **هي أنّ الطلبَ لم يُقبَل**: يبقى
// في طابور المكتب، ويرصده الراصدُ بإنذار `no_accept`.

// TestR24_F1_SettingReadFailureLeavesPending **قراءةُ الإعداد تفشل.**
//
// **والافتراضُ `false`** — فلا تحويلَ أصلاً، والطلبُ `pending`.
func TestR24_F1_SettingReadFailureLeavesPending(t *testing.T) {
	hh := New(t)
	n := &fakeNotifier{ready: true}
	m := armTransfer(t, hh, n)
	// **ومفتاحٌ مطفأٌ يُحاكي قراءةً ترتدّ إلى الافتراض** — والنتيجةُ
	// المقيسةُ واحدة: **لا تحويل.**
	hh.Setting("orders.auto_transfer", "false")

	oid := placeTransferOrder(t, hh, m)
	p := settleTransfer(t, hh, oid)
	t.Logf("F1: الحالُ=%q · مُرسَلٌ=%v · أحداثٌ=%d · نداءاتُ الإبلاغ=%d",
		p.Status, p.SentAt != nil, p.Events, n.calls)

	if p.Status != "pending" {
		t.Errorf("**F1: الطلبُ خرج من يد المكتب** — %q", p.Status)
	}
	if n.calls != 0 {
		t.Errorf("**F1: أُبلغ المتجرُ بلا تحويل** — %d", n.calls)
	}
}

// TestR24_F2F4_TransitionRefusedLeavesPending **قراءةُ الحالة أو
// القبولُ يفشل.**
//
// **وطلبٌ ليس `pending` لا يُحوَّل** — وهو نفسُ الحدّ الذي يحرس `F2`
// و`F4`: **ما لم يُقبَل يبقى في الطابور.**
func TestR24_F2F4_TransitionRefusedLeavesPending(t *testing.T) {
	hh := New(t)
	n := &fakeNotifier{ready: true}
	m := armTransfer(t, hh, n)

	// **والقبولُ وحدَه يُمنَع** — **بحاقنٍ يصيب صفَّ الانتقال إلى
	// `accepted` لا صفَّ الإنشاء**، وإلّا سقط إنشاءُ الطلب نفسُه
	// فلم يقع السيناريو أصلاً.
	fp := hh.Arm("R24/F4-transition", "order_events", "INSERT", 1,
		"to_status", "accepted")
	defer fp.Disarm()

	oid := placeTransferOrder(t, hh, m)
	p := settleTransfer(t, hh, oid)
	t.Logf("F2/F4: الحالُ=%q · مُرسَلٌ=%v · نداءاتُ الإبلاغ=%d",
		p.Status, p.SentAt != nil, n.calls)

	if p.Status != "pending" {
		t.Errorf("**F2/F4: قبولٌ وقع رغم سقوط كتابته** — %q", p.Status)
	}
	if n.calls != 0 {
		t.Errorf("**F2/F4: أُبلغ المتجرُ ولم يُقبَل الطلب** — %d", n.calls)
	}
}

// TestR24_F3_BotNotReadyLeavesPending **البوتُ غيرُ جاهزٍ — بالقصد.**
//
// **والقناةُ تُفحص قبل القبول لا بعده** (`auto_transfer.go`).
func TestR24_F3_BotNotReadyLeavesPending(t *testing.T) {
	hh := New(t)
	n := &fakeNotifier{ready: false}
	m := armTransfer(t, hh, n)

	oid := placeTransferOrder(t, hh, m)
	p := settleTransfer(t, hh, oid)
	t.Logf("F3: الحالُ=%q · نداءاتُ الإبلاغ=%d", p.Status, n.calls)

	if p.Status != "pending" {
		t.Errorf("**F3: قُبل الطلبُ وقناتُه ساقطة** — %q", p.Status)
	}
	if n.calls != 0 {
		t.Errorf("**F3: نُودي مُبلِّغٌ غيرُ جاهز** — %d", n.calls)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **F5 · الإبلاغُ يفشل بعد قبولٍ وقع** — النمطُ الأوّلُ الصامت
// ══════════════════════════════════════════════════════════════════════
//
// **والحالُ الباقيةُ صحيحةُ الشكل خاطئةُ المعنى**: `accepted` ·
// `sent_to_merchant_at = NULL` · **ولا سائقَ استُدعي.**
//
// **والزبونُ يقرأ «قيد التحضير» ولا أحدَ خلف الباب.**
func TestR24_F5_SendFailureAfterAccept(t *testing.T) {
	hh := New(t)
	n := &fakeNotifier{ready: true, err: errors.New("R24: قناةٌ ساقطة")}
	m := armTransfer(t, hh, n)

	oid := placeTransferOrder(t, hh, m)
	p := settleTransfer(t, hh, oid)
	t.Logf("F5: الحالُ=%q · مُرسَلٌ=%v · أحداثٌ=%d · تدقيقٌ=%d · "+
		"سائقٌ=%v · نداءاتُ الإبلاغ=%d",
		p.Status, p.SentAt != nil, p.Events, p.Audits, p.Assigned, n.calls)

	if n.calls == 0 {
		t.Fatal("**F5: لم يُنادَ المُبلِّغُ أصلاً** — والنمطُ لم يُعَد إنتاجُه")
	}
	assertTransferFailureIsReadable(t, hh, oid, p, "F5")
}

// ══════════════════════════════════════════════════════════════════════
// **F6 · لا هاتفَ للمتجر** — النمطُ الثاني الصامت
// ══════════════════════════════════════════════════════════════════════
func TestR24_F6_NoMerchantPhoneAfterAccept(t *testing.T) {
	hh := New(t)
	n := &fakeNotifier{ready: true}
	m := armTransfer(t, hh, n)
	// **ورقمُ واتساب يُقرأ من صاحب المتجر ثمّ من المتجر** —
	// `loadOrderMessage`. **فيُمحى المصدران معاً.**
	if _, err := hh.Pool.Exec(ctxBG(), `
		UPDATE merchants SET phone = '', owner_user_id = NULL
		 WHERE id = $1::uuid`, m.ID); err != nil {
		t.Fatalf("محوُ الهاتف: %v", err)
	}

	oid := placeTransferOrder(t, hh, m)
	p := settleTransfer(t, hh, oid)
	t.Logf("F6: الحالُ=%q · مُرسَلٌ=%v · أحداثٌ=%d · تدقيقٌ=%d · "+
		"سائقٌ=%v · نداءاتُ الإبلاغ=%d",
		p.Status, p.SentAt != nil, p.Events, p.Audits, p.Assigned, n.calls)

	if n.calls != 0 {
		t.Errorf("**F6: نُودي المُبلِّغُ بلا هاتف** — %d", n.calls)
	}
	// **ومتجرٌ بلا رقمٍ معلومٌ قبل القبول** — **فلا يُقبَل طلبُه ثمّ
	// يُكتشَف أنّه لا يُبلَّغ.** (`R24` — مصالحةُ دورةِ ٢٧.)
	if p.Status != "pending" {
		t.Errorf("**F6: قُبل طلبُ متجرٍ لا عنوانَ له** — %q · "+
			"**والعنوانُ نصفُ القناة، ويُفحص قبل القبول.**", p.Status)
	}
	assertTransferFailureIsReadable(t, hh, oid, p, "F6")
}

// ══════════════════════════════════════════════════════════════════════
// **F7 · الإنزالُ يفشل بعد إبلاغٍ وقع**
// ══════════════════════════════════════════════════════════════════════
//
// **والمتجرُ علم** — **والحالُ `accepted` مُبلَّغٌ بلا سائق**، ولها
// زرُّ «طلب سائق» في اللوحة. **فهي مرئيّةٌ بعقدها.**
func TestR24_F7_DispatchFailureIsVisible(t *testing.T) {
	hh := New(t)
	n := &fakeNotifier{ready: true}
	m := armTransfer(t, hh, n)
	// **ولا سائقَ في النظام** — فالإنزالُ لا يجد من يستدعي.
	oid := placeTransferOrder(t, hh, m)
	p := settleTransfer(t, hh, oid)
	var mode string
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT value::text FROM app_settings WHERE key = 'platform.orders_mode'`).Scan(&mode)
	t.Logf("F7: الحالُ=%q · مُرسَلٌ=%v · سائقٌ=%v · نداءاتُ الإبلاغ=%d · الوضعُ=%s",
		p.Status, p.SentAt != nil, p.Assigned, n.calls, mode)

	if p.Status == "pending" {
		t.Fatalf("**F7: لم يقع تحويلٌ أصلاً** — %q", p.Status)
	}
	// **والوسمُ هو الفارق**: **مُبلَّغٌ بلا سائقٍ حالٌ يعرفها المكتب.**
	if p.SentAt == nil {
		t.Errorf("**F7: أُبلغ المتجرُ ولم يُوسَم** — والوسمُ هو ما يفرّق")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **والأثرُ المقروء — ما يجب أن يبقى بعد انتهاء الطلب**
// ══════════════════════════════════════════════════════════════════════
//
// **والسجلُّ ليس أثراً**: لا يُقرأ في لوحةٍ ولا يُستدرَك منه ولا
// يُنسَب إلى طلب. **والمطلوبُ حالٌ في القاعدة تقول ما جرى.**
func assertTransferFailureIsReadable(t *testing.T, hh *Harness,
	oid string, p transferProbe, mode string) {
	t.Helper()

	if p.Status != "accepted" {
		t.Logf("%s: الحالُ %q — لم يقع النمطُ الصامت", mode, p.Status)
		return
	}
	if p.SentAt != nil {
		t.Errorf("**%s: وُسم الطلبُ مُرسَلاً ولم يُرسَل** — "+
			"**وذاك أسوأُ من غياب الأثر: نجاحٌ كاذب.**", mode)
	}

	// **أثرٌ دائمٌ ينسب الفشلَ إلى طلبه** — حدثٌ أو صفٌّ يُقرأ.
	var failEvents int
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM order_events
		 WHERE order_id = $1::uuid AND note LIKE $2 || '%'`,
		oid, orders.TransferFailureNote).Scan(&failEvents); err != nil {
		t.Fatalf("قراءةُ الأحداث: %v", err)
	}
	t.Logf("%s: أثرُ الفشل — أحداثٌ تصفه=%d · آثارُ تدقيقٍ=%d",
		mode, failEvents, p.Audits)

	if failEvents == 0 && p.Audits == 0 {
		t.Errorf("**%s: قُبل الطلبُ ولم يُبلَّغ المتجرُ ولا أثرَ يُقرأ** — "+
			"**لا حدثَ ولا تدقيق**، **والسجلُّ ليس حقيقةً تشغيليّة.** "+
			"(`R24`)", mode)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **والأثرُ يُقرأ في اللوحة ويبقى بعد النداء** — `R24`
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا نداءُ اللوحة لا قراءةُ جدول
//
// **صفٌّ في القاعدة ليس أثراً مقروءاً حتّى يصل إلى من يقرأ** —
// **ومن يقرأ هو المكتبُ في شاشة الطلب.**
//
// # وأمّا «بعد إعادة التشغيل»
//
// **الأثرُ صفٌّ مُقيَّدٌ في `order_events`** — **يبقى ما بقيت
// القاعدة**، ولا حالَ في ذاكرةِ عمليّةٍ تحمله. **فالإعادةُ تُقاس
// بنداءٍ جديدٍ بعد انتهاء الأوّل** — وهو ما يقع هنا.
func TestR24_F5_TraceIsAdminReadableAfterTheRequest(t *testing.T) {
	hh := New(t)
	n := &fakeNotifier{ready: true, err: errors.New("R24: قناةٌ ساقطة")}
	m := armTransfer(t, hh, n)

	oid := placeTransferOrder(t, hh, m)
	p := settleTransfer(t, hh, oid)
	if p.Status != "accepted" || n.calls == 0 {
		t.Fatalf("لم يقع النمطُ الخامس: الحالُ=%q · نداءاتٌ=%d", p.Status, n.calls)
	}

	// **ونداءٌ إداريٌّ جديدٌ بعد أن انتهى نداءُ الإنشاء.**
	admin := hh.NewUser("admin")
	got := hh.GET("/api/v1/admin/orders/"+oid, admin.Token)
	if got.Code != 200 {
		t.Fatalf("قراءةُ الطلب في اللوحة: %s", got)
	}
	body := string(got.Body)
	t.Logf("F5: اللوحةُ ⇒ %d · أفيها الأثرُ؟ %v", got.Code,
		strings.Contains(body, orders.TransferFailureNote))

	if !strings.Contains(body, orders.TransferFailureNote) {
		t.Errorf("**الأثرُ لا يُقرأ في اللوحة** — **وصفٌّ لا يصل من يقرؤه " +
			"ليس أثراً مقروءاً.** (`R24`)")
	}
	// **ولا يُوسَم مُرسَلاً ما لم يُرسَل** — ولا نجاحَ كاذب.
	if p.SentAt != nil {
		t.Error("**وُسم مُرسَلاً ولم يُرسَل**")
	}
}
