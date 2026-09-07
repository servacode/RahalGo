package qa

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **عطلُ ذاكرةٍ حقيقيٌّ — ولا يُغني عنه تمثيل** — `R16`
// ══════════════════════════════════════════════════════════════════════
//
// **`P-0` أثبت العيبَ بذاكرةٍ حقيقيّةٍ تُوقَف** — **فإغلاقُه يحتاج
// المثلَ.** **ومحاكاةُ خطأٍ في وحدةٍ لا تُثبت ما يفعله سائقُ الشبكة
// حين تختفي الخدمة.**
//
// **وعلى التجهيز وحدَه**: `rahalgo-staging-redis`. **ولا تُمسّ ذاكرةُ
// الإنتاج ولا حاويةُ مشروعٍ آخر.**

// ══════════════════════════════════════════════════════════════════════
// **R4…R10 · الذاكرةُ تسقط ثمّ تعود**
// ══════════════════════════════════════════════════════════════════════
func TestR16_STG_RedisOutageAuthorityHolds(t *testing.T) {
	stagingOnly(t)
	h := New(t)

	valid, _, _ := liveSession(t, h)
	revoked, revSid, _ := liveSession(t, h)

	// ── ١+٢+٣ · الذاكرةُ قائمةٌ والعقدُ يعمل ─────────────────────
	if res := h.GET(mePath, valid); res.Code != http.StatusOK {
		t.Fatalf("جلسةٌ حيّةٌ لا تعمل والذاكرةُ قائمة: %d", res.Code)
	}
	markRevokedInRedis(t, h, revSid)
	revokeAnySession(t, h, revSid)
	if res := h.GET(mePath, revoked); res.Code != http.StatusUnauthorized {
		t.Fatalf("مُبطَلةٌ تعمل والذاكرةُ قائمة: %d", res.Code)
	}
	t.Log("✓ ١–٣ الذاكرةُ قائمة: الحيّةُ تعمل والمُبطَلةُ تُرفَض")

	// ── ٤ · تُعزَل الذاكرة ───────────────────────────────────────
	dockerRedis(t, "stop")
	restored := false
	t.Cleanup(func() {
		if !restored {
			dockerRedis(t, "start")
		}
	})
	time.Sleep(2 * time.Second)

	// ── ٥ · R4/R6 · حيّةٌ والذاكرةُ ساقطة ⇒ القاعدةُ تُثبتها ─────
	r4 := h.GET(mePath, valid)
	t.Logf("R4/R6: حيّةٌ + ذاكرةٌ ساقطة ⇒ %d", r4.Code)
	if r4.Code != http.StatusOK {
		t.Errorf("**جلسةٌ حيّةٌ رُفضت لأنّ خبيئةً سقطت**: %d", r4.Code)
	}

	// ── ٦ · R5/R7 · مُبطَلةٌ والذاكرةُ ساقطة ⇒ رفض ───────────────
	r5 := h.GET(mePath, revoked)
	t.Logf("R5/R7: مُبطَلةٌ + ذاكرةٌ ساقطة ⇒ %d", r5.Code)
	if r5.Code == http.StatusOK {
		t.Errorf("**`R16` قائم**: جلسةٌ مُبطَلةٌ قُبلت لأنّ الذاكرةَ سقطت")
	}
	if r5.Code != http.StatusUnauthorized {
		t.Errorf("**رُفضت بردٍّ غيرِ ٤٠١**: %d", r5.Code)
	}

	// ── ٧+٨ · R15 · تُبطَل جلسةٌ **والذاكرةُ ساقطة** ─────────────
	//
	// **وهذه الحالُ التي لا يُصلحها الرجوعُ عند الخطأ وحدَه**:
	// **كتابةُ `Redis` تسقط صامتةً** — **فتعود الذاكرةُ نظيفةً من
	// إبطالٍ وقع فعلاً.**
	late, lateSid, lateUID := liveSession(t, h)
	if res := h.GET(mePath, late); res.Code != http.StatusOK {
		t.Fatalf("جلسةٌ ثالثةٌ لا تعمل: %d", res.Code)
	}
	revokeAnySession(t, h, lateSid)

	var lateLive int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM refresh_tokens
		 WHERE session_id = $1::uuid AND revoked_at IS NULL`, lateSid).
		Scan(&lateLive); err != nil {
		t.Fatalf("قراءةُ الحقيقة الموثوقة: %v", err)
	}
	t.Logf("٧+٨: أُبطلت والذاكرةُ ساقطة — صفوفٌ حيّةٌ في القاعدة=%d · حسابٌ=%s…",
		lateLive, lateUID[:8])
	if lateLive != 0 {
		t.Fatalf("**القاعدةُ لم تُسجّل الإبطال**: %d", lateLive)
	}

	// ── ٩ · تعود الذاكرة ────────────────────────────────────────
	dockerRedis(t, "start")
	restored = true
	time.Sleep(2 * time.Second)

	// ── ١٠ · ولا مفتاحَ لتلك الجلسة — كما هو متوقَّع ─────────────
	n, err := h.Redis().Exists(context.Background(), "sess:revoked:"+lateSid).Result()
	t.Logf("١٠: بعد العودة — مفتاحُ إبطالِ الجلسة الثالثة=%d (والمرتقَبُ صفر · خطأٌ=%v)",
		n, err)
	if n != 0 {
		t.Errorf("**التركيبةُ خطأ**: المفتاحُ موجودٌ فلا تُقاس الحالُ المقصودة")
	}

	// ── ١١ · R15 · ومع ذلك تبقى مرفوضةً — بالحقيقة الموثوقة ─────
	r15 := h.GET(mePath, late)
	t.Logf("R15: أُبطلت أثناء العطل · عادت الذاكرةُ بلا مفتاح ⇒ %d", r15.Code)
	if r15.Code == http.StatusOK {
		t.Errorf("**جلسةٌ مُبطَلةٌ بُعثت بعودة الذاكرة** — " +
			"**وغيابُ المفتاح قُرئ سلامةً.** (`R16`)")
	}

	// ── ١٢ · R9/R10 · والتعافي سليم ─────────────────────────────
	t.Logf("R9: الحيّةُ بعد العودة ⇒ %d", h.GET(mePath, valid).Code)
	if h.GET(mePath, valid).Code != http.StatusOK {
		t.Error("**جلسةٌ حيّةٌ لم تتعافَ بعد عودة الذاكرة**")
	}
	if h.GET(mePath, revoked).Code == http.StatusOK {
		t.Error("**مُبطَلةٌ عادت تعمل بعد عودة الذاكرة**")
	}
	t.Log("R10 ✓ المُبطَلةُ تبقى مرفوضةً بعد العودة")

	// ── ١٣ · وسلامةُ بيانات التجهيز ─────────────────────────────
	var users int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM users`).Scan(&users); err != nil {
		t.Fatalf("سلامةُ البيانات: %v", err)
	}
	t.Logf("١٣ ✓ سلامةُ البيانات: حساباتٌ=%d · والقاعدةُ تردّ", users)
}

// ══════════════════════════════════════════════════════════════════════
// **R8 · تسقط الذاكرةُ والقاعدةُ معاً ⇒ سقوطٌ آمنٌ لا توثيق**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا تُوقَف قاعدةُ التجهيز**: **حاويةٌ واحدةٌ تخدم الحزمةَ كلَّها**،
// **ووقفُها يُسقط ثمانمئةَ فحصٍ غيرِ معنيّ.**
//
// **فيُقاس الشرطُ في محلّه**: **`CheckSession` تُرجع خطأً حين تسقط
// القاعدة، والوسيطُ يردّ ٥٠٣** — **وهو مقروءٌ في `middleware.go`
// ومحروسٌ بنيويّاً** (`TestR16_NoFailOpenSessionCheck`).
//
// **ولا أدّعي قياساً لم يقع.**
func TestR16_R8_BothDownIsSafeFailure(t *testing.T) {
	h := New(t)
	_, sid, _ := liveSession(t, h)

	// **ومعرّفُ جلسةٍ لا صفَّ له** — أقربُ ما يُقاس بلا إسقاط قاعدة:
	// **القاعدةُ ردّت ولم تُثبت سلامةً** ⇒ رفض.
	bogus := h.TokenWithSession(h.NewUser("customer").ID,
		"00000000-0000-0000-0000-000000000000", "customer")
	res := h.GET(mePath, bogus)
	t.Logf("جلسةٌ لا تعرفها الحقيقةُ الموثوقة ⇒ %d (والمعرّفُ الحيُّ %s…)",
		res.Code, sid[:8])
	if res.Code == http.StatusOK {
		t.Errorf("**جلسةٌ بلا صفٍّ قُرئت سليمة** — "+
			"**والسلامةُ تُثبَت ولا تُفترَض.** %d", res.Code)
	}
}
