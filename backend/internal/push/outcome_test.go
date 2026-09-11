package push

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/obs"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// ══════════════════════════════════════════════════════════════════════
// **دلالةُ عدّاد الدفع — تُثبَت بناقلٍ مُصطنَعٍ لا بإرسالٍ حقيقيّ**
// ══════════════════════════════════════════════════════════════════════
//
// (دورةُ ٧٠أ-س١ · `Phase 14`.)
//
// **وناقلُ التجهيز مُطفأ** — **صفرُ متغيّرِ `FCM` في حاويته** — **فلا
// يمرّ العدّادُ هناك أصلاً**: `SendToUser` تعود قبله حين لا ناقل.
// **وتشغيلُ ناقلٍ حقيقيٍّ على التجهيز يعني إشعاراً قد يصل إنساناً**،
// وذاك ممنوع.
//
// **فتُثبَت الدلالةُ هنا**: **ناقلٌ يردّ ما يُملى عليه** — فيُعرف ما
// يعنيه كلُّ رقمٍ بالضبط.

// errTransport ناقلٌ يفشل دائماً — **وفشلُ المزوّد غيرُ رفضِ رمز.**
type errTransport struct{}

func (errTransport) Platform() string { return PlatformAndroid }
func (errTransport) Send(context.Context, []string, Message) ([]string, error) {
	return nil, errors.New("provider unreachable")
}

// TestPushOutcome_SemanticsAreExact **كلُّ عدّادٍ يعني ما اسمُه.**
func TestPushOutcome_SemanticsAreExact(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()

	// ── ١ · حسابٌ بلا جهاز ⇒ `no_device` وحدَه ───────────────────
	obs.Reset()
	bare := testdb.NewUser(t, pool, "driver")
	New(pool, quietLogger(), &fakeTransport{}).
		SendToUser(ctx, bare, Message{Title: "لا جهاز"})
	s := obs.Take().Push
	if s[string(obs.PushNoDevice)] != 1 {
		t.Errorf("**حسابٌ بلا جهازٍ يجب `no_device`=١** — %v", s)
	}
	if s[string(obs.PushAttempted)] != 0 {
		t.Errorf("**ولا محاولةَ تُعدّ حين لا جهاز** — %v", s)
	}

	// ── ٢ · جهازان أحدُهما مرفوضٌ نهائيّاً ───────────────────────
	obs.Reset()
	uid := testdb.NewUser(t, pool, "driver")
	svc := New(pool, quietLogger(), &fakeTransport{dead: []string{"out-dead"}})
	if err := svc.Register(ctx, uid, "out-dead", PlatformAndroid, "driver", "1.0"); err != nil {
		t.Fatalf("تسجيل: %v", err)
	}
	if err := svc.Register(ctx, uid, "out-live", PlatformAndroid, "driver", "1.0"); err != nil {
		t.Fatalf("تسجيل: %v", err)
	}
	svc.SendToUser(ctx, uid, Message{Title: "طلبٌ جديد"})
	s = obs.Take().Push
	if s[string(obs.PushAttempted)] != 2 {
		t.Errorf("**رمزان عُرضا على الناقل ⇒ `attempted`=٢** — %v", s)
	}
	if s[string(obs.PushDeadToken)] != 1 {
		t.Errorf("**وواحدٌ رُفض نهائيّاً ⇒ `dead_token`=١** — %v", s)
	}
	if s[string(obs.PushSent)] != 1 {
		t.Errorf("**والمقبولُ ما لم يُردّ رمزُه ⇒ `sent`=١** — %v", s)
	}
	if s[string(obs.PushFailed)] != 0 {
		t.Errorf("**ورفضُ رمزٍ ليس فشلَ إرسال** — %v", s)
	}

	// ── ٣ · مزوّدٌ لا يُبلَغ ⇒ `failed` لا `dead_token` ──────────
	obs.Reset()
	uid2 := testdb.NewUser(t, pool, "driver")
	svc2 := New(pool, quietLogger(), errTransport{})
	if err := svc2.Register(ctx, uid2, "out-err", PlatformAndroid, "driver", "1.0"); err != nil {
		t.Fatalf("تسجيل: %v", err)
	}
	svc2.SendToUser(ctx, uid2, Message{Title: "مزوّدٌ ساقط"})
	s = obs.Take().Push
	if s[string(obs.PushFailed)] != 1 {
		t.Errorf("**تعذُّرُ بلوغ المزوّد ⇒ `failed`=١** — %v", s)
	}
	if s[string(obs.PushSent)] != 0 {
		t.Errorf("**ولا يُحسَب مقبولاً ما لم يُقبَل** — %v", s)
	}
	if s[string(obs.PushDeadToken)] != 0 {
		t.Errorf("**ولا رمزَ يُعدّ ميّتاً لأنّ الشبكةَ سقطت** — %v", s)
	}

	// **وحدٌّ يُقال**: **`sent` = «قبله المزوّد» لا «رنَّ في جيب»**
	// — **وغوغل لا تقول أكثر.**
	t.Logf("PUSH SEMANTICS: no_device · attempted · sent · dead_token · failed — كلٌّ بمعناه")
}
