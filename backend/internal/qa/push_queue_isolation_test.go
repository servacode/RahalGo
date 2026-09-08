package qa

// ══════════════════════════════════════════════════════════════════════
// **صفُّ الدفع لا يُورَث — واختبارٌ لا يجوع بعملِ من سبقه** — `XG-41C`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان يقع
//
// **`PF-09` تنجح وحدَها وتسقط في الحزمة** — **والعرَضُ «لم يُنادَ
// الناقلُ أصلاً»**، وصفوفُ النقل صفر.
//
// **والسببُ مقيسٌ لا مُستنتَج**: `fanOut` يأخذ **الخمسين الأقدمَ** من
// المعلَّق (`deliveryBatch`). **وفي الإنتاج نبضةٌ كلَّ ثلاثين ثانيةً
// تستهلك الصفَّ أوّلاً بأوّل** — **وفي الاختبار لا نبضةَ أصلاً**،
// `RunPushDeliveryWorker` لا يناديها إلّا `cmd/api`.
//
// **فيتراكم المعلَّق**: قيس في نهاية `internal/qa` على `ed8b43b5`
// **٣٨٦٥ إشعاراً معلَّقاً**. **وحين يتجاوز المتراكمُ خمسين لا تبلغ
// جولةُ الفحصِ إشعارَه أبداً.**
//
// **وقيس عند الحدّ**: `older_pending=86` · `fanned=50` · **صفوفٌ=0**.
//
// # وليس عطبَ منتَج
//
// **العقدُ «لا يضيع عملٌ» قائم**: الإشعارُ يبقى معلَّقاً حتّى تبلغه
// جولة. **والمنتَجُ يدير جولةً كلَّ ثلاثين ثانيةً فيستهلكه.**
// **والغائبُ في الاختبار هو المنفّذُ لا العقد.**
//
// # وهذا الفحصُ يقيس السلسلةَ كاملةً
//
// **يزرع ركاماً ⇒ يثبت الجوع ⇒ يبني مِسنَداً جديداً ⇒ يثبت أنّه لم
// يرثه.** **فمن أزال التبرّؤَ غداً سقط هنا، لا في `PF-09` بعد ساعة.**

import (
	"fmt"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/server"
)

// backlogSeed **ما يُزرع من ركامٍ متروك.**
//
// **وأكبرُ من `deliveryBatch` بأربعة أضعاف** — **فمن رفع الدفعةَ في
// المنتَج يرى هذا الفحصَ يسقط ويعيد ضبطَ الرقم**، **ولا يمرّ صامتاً.**
const backlogSeed = 200

func TestXG41C_InheritedPushQueueDoesNotStarveTheNextTest(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	// **ومصنعٌ واحدٌ لكلّ الحسابات** — **مصنعان يبدآن من العدّاد
	// نفسِه فيتصادم هاتفاهما.**
	f := h.Factory()

	// ── ١ ── ركامٌ متروكٌ كما يتركه اختبارٌ انتهى ─────────────────
	//
	// **وأقدمُ من إشعارِنا صراحةً** — **فالترتيبُ بالإنشاء.**
	victim := f.NewUserWith("customer")
	for i := 0; i < backlogSeed; i++ {
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO notifications (user_id, kind, title, push_pending, created_at)
			VALUES ($1::uuid, 'system', $2, true, now() - interval '1 hour')`,
			victim.ID, fmt.Sprintf("ركامٌ %d", i)); err != nil {
			t.Fatalf("زرعُ الركام: %v", err)
		}
	}

	var backlog int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM notifications WHERE push_pending`).Scan(&backlog); err != nil {
		t.Fatalf("عدُّ المعلَّق: %v", err)
	}
	if backlog < backlogSeed {
		t.Fatalf("**لم يُزرع الركام**: %d", backlog)
	}

	// ── ٢ ── والجوعُ يقع فعلاً — **وهذا هو الإثبات لا الادّعاء** ──
	u := f.NewUserWith("customer")
	if !addToken(t, h, u.ID, "xg41c-starved") {
		t.Skip("جدولُ رموز الدفع غيرُ متاح")
	}
	notifyUser(t, h, admin, u.ID, 1000)
	h.API.DeliverPushOnce(ctxBG())

	starved := deliveryRows(t, h, u.ID)
	t.Logf("BEFORE: ركامٌ=%d · صفوفُ نقلِ إشعارِنا=%d", backlog, starved)
	if starved != 0 {
		t.Fatalf("**لم يقع الجوعُ بركامٍ %d** — **فإمّا ارتفعت الدفعةُ في "+
			"المنتَج فوق هذا الزرع، وإمّا تبدّل الترتيب.** "+
			"**يُعاد ضبطُ `backlogSeed` بعد قراءة `deliveryBatch`.**", backlogSeed)
	}

	// ── ٣ ── ومِسنَدٌ جديدٌ لا يرث الصفَّ ──────────────────────────
	fake := NewFakePush()
	next := NewWith(t, server.WithPushTransport(fake))

	var inherited int
	if err := next.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM notifications WHERE push_pending`).Scan(&inherited); err != nil {
		t.Fatalf("عدُّ الموروث: %v", err)
	}
	if inherited != 0 {
		t.Errorf("**وُرِث صفُّ دفعٍ فيه %d** — **والاختبارُ التالي يجوع به**",
			inherited)
	}

	// ── ٤ ── وإشعارُه يبلغ الناقل ────────────────────────────────
	// **والحسابُ نفسُه يصلح**: **القاعدةُ واحدةٌ وسرُّ التوقيع واحد**
	// — **والمبدَّلُ هو المِسنَدُ لا المنصّة.**
	u2 := f.NewUserWith("customer")
	if !addToken(t, next, u2.ID, "xg41c-fed") {
		t.Skip("جدولُ رموز الدفع غيرُ متاح")
	}
	notifyUser(t, next, admin, u2.ID, 1000)
	settle(t, next, u2.ID, 1)
	next.API.DeliverPushOnce(ctxBG())

	fed := deliveryRows(t, next, u2.ID)
	t.Logf("AFTER: موروثٌ=%d · صفوفُ نقلِ إشعارِه=%d · نداءاتٌ=%d",
		inherited, fed, len(fake.Calls()))
	if fed == 0 {
		t.Error("**إشعارُ المِسنَدِ الجديدِ لم يُفرَّع** — **الجوعُ باقٍ**")
	}
}

// deliveryRows عددُ صفوفِ نقلِ إشعارات حساب.
func deliveryRows(t *testing.T, h *Harness, uid string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM notification_deliveries d
		  JOIN notifications n ON n.id = d.notification_id
		 WHERE n.user_id = $1::uuid`, uid).Scan(&n); err != nil {
		t.Fatalf("عدُّ صفوف النقل: %v", err)
	}
	return n
}
