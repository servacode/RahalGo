package qa

// ══════════════════════════════════════════════════════════════════════
// **من خلّف عملاً في صفّ الدفع نظّفه** — `XG-41C`
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
// جولةُ الفحصِ إشعارَه أبداً.** **وقيس عند الحدّ**: `older_pending=86`
// · `fanned=50` · **صفوفٌ=0**.
//
// # وليس عطبَ منتَج
//
// **العقدُ «لا يضيع عملٌ» قائم**: الإشعارُ يبقى معلَّقاً حتّى تبلغه
// جولة. **والغائبُ في الاختبار هو المنفّذُ لا العقد.**
//
// # وهذا الملفّ يقيس عقدَ الملكيّة كلَّه
//
//	١ الجوعُ يقع فعلاً بركامٍ موروث
//	٢ والمنتِجُ ينظّف ما خلّفه في خاتمته
//	٣ ومن بعده يجد صفَّه خالياً وإشعارُه يبلغ الناقل
//	٤ **وزرعُ الفحصِ الجاري لا يُمَسّ** ولو بُني مِسنَدٌ ثانٍ
//	٥ والحصريّةُ التي يقوم عليها هذا كلُّه محروسة

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/server"
)

// backlogSeed **ما يُزرع من ركامٍ متروك.**
//
// **وأكبرُ من `deliveryBatch` بأربعة أضعاف** — **فمن رفع الدفعةَ في
// المنتَج يرى هذا الفحصَ يسقط ويعيد ضبطَ الرقم**، **ولا يمرّ صامتاً.**
const backlogSeed = 200

// pendingCount ما في صفّ الدفع الآن.
func pendingCount(t *testing.T, h *Harness) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM notifications WHERE push_pending`).Scan(&n); err != nil {
		t.Fatalf("عدُّ المعلَّق: %v", err)
	}
	return n
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

func TestXG41C_PushQueueIsOwnedByItsProducer(t *testing.T) {
	base := New(t)

	// **وصاحبُ الركام يعيش بعد المنتِج** — **وإلّا محا حذفُه إشعاراتِه
	// بالتتالي فبدا الصفُّ نظيفاً بلا تنظيف**، **وحارسٌ يمرّ بلا حارس
	// أسوأُ من لا حارس.** **فيُنشأ هنا ويُحذف في خاتمة هذا الفحص.**
	var victim string
	if err := base.Pool.QueryRow(ctxBG(), `
		INSERT INTO users (phone, full_name, status)
		VALUES ($1, 'صاحبُ الركام', 'active') RETURNING id::text`,
		"+963900414100").Scan(&victim); err != nil {
		t.Fatalf("صاحبُ الركام: %v", err)
	}
	t.Cleanup(func() {
		_, _ = base.Pool.Exec(ctxBG(), `DELETE FROM users WHERE id::text = $1`, victim)
	})

	// ── ١ و٢ ── منتِجٌ يجوع بركامه ثمّ ينظّفه في خاتمته ──────────
	t.Run("الجوعُ_يقع_ثمّ_يُنظَّف_ما_خُلِّف", func(t *testing.T) {
		h := New(t)
		admin := h.NewUser("admin")
		// **ومصنعٌ واحدٌ لكلّ الحسابات** — **مصنعان يبدآن من العدّاد
		// نفسِه فيتصادم هاتفاهما.**
		f := h.Factory()

		// **ركامٌ متروكٌ كما يتركه فحصٌ انتهى** — **وأقدمُ صراحةً**،
		// فالترتيبُ بالإنشاء.
		for i := 0; i < backlogSeed; i++ {
			if _, err := h.Pool.Exec(ctxBG(), `
				INSERT INTO notifications (user_id, kind, title, push_pending, created_at)
				VALUES ($1::uuid, 'system', $2, true, now() - interval '1 hour')`,
				victim, fmt.Sprintf("ركامٌ %d", i)); err != nil {
				t.Fatalf("زرعُ الركام: %v", err)
			}
		}
		if got := pendingCount(t, h); got < backlogSeed {
			t.Fatalf("**لم يُزرع الركام**: %d", got)
		}

		// **والجوعُ يقع فعلاً — وهذا إثباتُ النمط لا ادّعاؤه.**
		u := f.NewUserWith("customer")
		if !addToken(t, h, u.ID, "xg41c-starved") {
			t.Skip("جدولُ رموز الدفع غيرُ متاح")
		}
		notifyUser(t, h, admin, u.ID, 1000)
		h.API.DeliverPushOnce(ctxBG())

		starved := deliveryRows(t, h, u.ID)
		t.Logf("BEFORE: ركامٌ=%d · صفوفُ نقلِ إشعارِنا=%d", backlogSeed, starved)
		if starved != 0 {
			t.Fatalf("**لم يقع الجوعُ بركامٍ %d** — **فإمّا ارتفعت الدفعةُ في "+
				"المنتَج فوق هذا الزرع، وإمّا تبدّل الترتيب.** "+
				"**يُعاد ضبطُ `backlogSeed` بعد قراءة `deliveryBatch`.**", backlogSeed)
		}

		// ── ٤ ── **وزرعُ الفحصِ الجاري لا يُمَسّ** ───────────────
		//
		// **مِسنَدٌ ثانٍ داخلَ الفحص نفسِه لا يُطفئ ما زرعه صاحبُه** —
		// **فالتنظيفُ في الخاتمة لا في المُفتَتَح.**
		second := New(t)
		if got := pendingCount(t, second); got == 0 {
			t.Error("**مِسنَدٌ ثانٍ محا زرعَ الفحص الجاري** — " +
				"**والملكيّةُ للمنتِج ما دام حيّاً**")
		} else {
			t.Logf("OWNERSHIP: بعد بناء مِسنَدٍ ثانٍ بقي %d معلَّقاً — **لم يُمَسّ زرعُنا**", got)
		}
	})

	// ── ٣ ── ومن بعده يجد صفَّه خالياً ───────────────────────────
	//
	// **والفحصُ السابقُ انتهت خاتمتُه قبل هذا السطر.**
	fake := NewFakePush()
	next := NewWith(t, server.WithPushTransport(fake))
	inherited := pendingCount(t, next)
	if inherited != 0 {
		t.Errorf("**وُرِث صفُّ دفعٍ فيه %d** — **والذي يليه يجوع به**", inherited)
	}

	admin := next.NewUser("admin")
	nf := next.Factory()
	u := nf.NewUserWith("customer")
	if !addToken(t, next, u.ID, "xg41c-fed") {
		t.Skip("جدولُ رموز الدفع غيرُ متاح")
	}
	notifyUser(t, next, admin, u.ID, 1000)
	settle(t, next, u.ID, 1)
	next.API.DeliverPushOnce(ctxBG())

	fed := deliveryRows(t, next, u.ID)
	t.Logf("AFTER: موروثٌ=%d · صفوفُ نقلِ إشعارِه=%d · نداءاتٌ=%d",
		inherited, fed, len(fake.Calls()))
	if fed == 0 {
		t.Error("**إشعارُ المِسنَدِ الجديدِ لم يُفرَّع** — **الجوعُ باقٍ**")
	}

	// ── ٦ ── **و«مطفأٌ بلا نقل» حالٌ يصنعها المنتَجُ نفسُه** ──────
	//
	// **وهذا ما يجعل التنظيفَ صادقاً لا تزويراً**: `fanOutOne` تطفئ
	// العلامةَ في معاملتها **ولو لم يكن للحساب جهازٌ واحد** —
	// **فالصفُّ المطفأُ بلا أهدافٍ عقدٌ قائمٌ في الإنتاج**، لا شكلٌ
	// مستحيلٌ اخترعه الاختبار.
	bare := nf.NewUserWith("customer")
	notifyUser(t, next, admin, bare.ID, 1000)
	next.API.DeliverPushOnce(ctxBG())

	// **ويُنتظَر أن تهدأ الجولة**: **النبضةُ الموقظةُ خيطٌ مستقلّ** —
	// **وجولتُنا تتخطّى صفّاً هي ممسكةٌ به (`SKIP LOCKED`)**، فتُقرأ
	// العلامةُ في منتصف عملٍ لم يُودَع. **وقياسُ منتصفِ عمليّةٍ سباقٌ
	// لا عقد.**
	stillPending := true
	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := next.Pool.QueryRow(ctxBG(), `
			SELECT push_pending FROM notifications
			 WHERE user_id = $1::uuid ORDER BY created_at DESC LIMIT 1`,
			bare.ID).Scan(&stillPending); err != nil {
			t.Fatalf("قراءةُ العلامة: %v", err)
		}
		if !stillPending || time.Now().After(deadline) {
			break
		}
		next.API.DeliverPushOnce(ctxBG())
		time.Sleep(20 * time.Millisecond)
	}
	bareRows := deliveryRows(t, next, bare.ID)
	t.Logf("VALID STATE: حسابٌ بلا جهاز ⇒ معلَّقٌ=%v · صفوفُ نقلٍ=%d",
		stillPending, bareRows)
	if stillPending || bareRows != 0 {
		t.Errorf("**المنتَجُ لا يصنع «مطفأً بلا نقل»**: معلَّقٌ=%v · صفوفٌ=%d — "+
			"**فيُراجَع معنى الإطفاء في التنظيف**", stillPending, bareRows)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **وحصريّةُ المِسنَد شرطُ صحّةِ التنظيف — فتُحرَس** — `XG-41C`
// ══════════════════════════════════════════════════════════════════════
//
// # لماذا
//
// **تنظيفُ الخاتمة يلمس صفّاً عامّاً.** **وهو آمنٌ لأنّه لا فحصَ آخرَ
// حيٌّ لحظتَه** — **ولو صار فحصان يتداخلان لمحا أحدُهما عملَ الآخر.**
//
// **فلا يُتَّكَل على أنّ فحوصَنا اليومَ لا تتداخل** — **يُحرَس الشرط.**
//
//	داخلَ الحزمة  · **لا `t.Parallel`** — وهي وحدَها ما يُداخِل فحصين
//	بين الحزم     · **قفلُ جلسةٍ في `testdb`** — ثنائيّةٌ تنتظر أختَها
func TestXG41C_HarnessExclusivityIsEnforced(t *testing.T) {
	// **ويُقرأ المصدرُ بعد طيّ المسافات** — **فلا يُسقطه تنسيق.**
	parallel := regexp.MustCompile(`\bt\s*\.\s*Parallel\s*\(`)
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("تعذّرت قراءةُ مصادر الحزمة: %v", err)
	}
	var offenders []string
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("قراءةُ %s: %v", f, err)
		}
		if parallel.Match(b) {
			offenders = append(offenders, f)
		}
	}
	if len(offenders) > 0 {
		t.Errorf("**فحوصٌ متوازيةٌ في حزمةٍ مِسنَدُها يملك صفّاً عامّاً**: %s\n"+
			"  **فإمّا تُعزَل الملكيّةُ لكلّ فحصٍ، وإمّا لا توازيَ هنا.** (`XG-41C`)",
			strings.Join(offenders, " · "))
	}

	// **وبين الحزم**: **قفلُ جلسةٍ لا قفلُ معاملة** — **فيعيش عمرَ
	// الثنائيّة، ولا تقتسم ثنائيّتان القاعدة.**
	lock, err := os.ReadFile(filepath.Join("..", "testdb", "reset.go"))
	if err != nil {
		t.Fatalf("قراءةُ حارس القاعدة: %v", err)
	}
	if !strings.Contains(string(lock), "pg_advisory_lock") {
		t.Error("**سقط قفلُ الجلسة من `testdb`** — " +
			"**وثنائيّتان على قاعدةٍ واحدةٍ تمحوان عملَ بعضهما**")
	}
	t.Logf("EXCLUSIVITY: %d ملفّاً فُحص · صفرُ توازٍ · قفلُ جلسةٍ قائم", len(files))
}
