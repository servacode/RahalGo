package qa

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/androidmap"
)

// ══════════════════════════════════════════════════════════════════════
// **`D19` — إعادةُ الوصل برمزٍ منتهٍ**
// ══════════════════════════════════════════════════════════════════════
//
// # العطب
//
// **حلقةُ `LiveSocket` كانت تقرأ `session.accessToken()` عند كلّ
// محاولةٍ ولا تجدّد أبداً** — **فرمزٌ انتهى يُعاد به الوصلُ إلى
// الأبد**، وعائلةُ التجديد سليمةٌ طوالَ ذلك.
//
// # وقسمةُ الدليل
//
// **الشقُّ الأوّلُ هنا**: **المحرّكُ يقول متى ينتهي الرمز**
// (`access_expires_at`) — **وعليه يقوم قرارُ الجهاز كلُّه.** **ولو
// سقط هذا الحقلُ صمتاً لَعاد العطبُ ولا يسقط حارسٌ في أندرويد**:
// **يصير المنتهى مجهولاً، والمجهولُ لا يُجدَّد له.**
//
// **والشقُّ الثاني في `mobile/shared`** — **مقبسٌ حقيقيٌّ ومصافحةٌ
// حقيقيّة**، **ويُحرَس هنا أن يبقى موجوداً.**

// ══════════════════════════════════════════════════════════════════════
// **والمحرّكُ يقول المنتهى — في الدخول وفي التجديد**
// ══════════════════════════════════════════════════════════════════════
func TestD19_ServerDeclaresAccessExpiry(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)

	// **ودخولٌ واحدٌ يُقاس عليه الشقّان** — **ودخولٌ ثانٍ يُدوّر
	// العائلةَ فيُبطل رمزَ الأوّل**، فيُقاس تعثّرٌ من تركيب الفحص.
	raw, sid := issuedRefreshStrict(t, h, drv.ID, "driver")

	// ── الدخولُ يقولها ────────────────────────────────────────────
	in := h.POST("/api/v1/auth/refresh", "", map[string]any{"refresh_token": raw})
	if in.Code >= 400 {
		t.Fatalf("التجديدُ الأوّل: %s", in)
	}
	loginExp := tokenField(in, "access_expires_at")
	oldAccess := tokenField(in, "access_token")
	raw = tokenField(in, "refresh_token")
	t.Logf("D19: الدخولُ — access_expires_at=%q", loginExp)
	if loginExp == "" {
		t.Fatal("**الدخولُ لا يقول متى ينتهي الرمز** — " +
			"**فالجهازُ لا يعرف أنّه انتهى فلا يجدّد** (`D19`).")
	}
	at, err := time.Parse(time.RFC3339, loginExp)
	if err != nil {
		t.Fatalf("**منتهىً لا يُقرأ بـRFC3339**: %q — %v", loginExp, err)
	}
	if !at.After(time.Now()) {
		t.Errorf("**رمزٌ يُسلَّم منتهياً**: %s", at)
	}

	// ── والتجديدُ يقولها كذلك ─────────────────────────────────────
	//
	// **وهو الموضعُ الحرِج**: **الجهازُ يحفظ ما يردّه التجديد** —
	// **فتجديدٌ بلا منتهىً يمحو المعرفةَ بعد أوّل دورة.**
	got := h.POST("/api/v1/auth/refresh", "", map[string]any{"refresh_token": raw})
	if got.Code >= 400 {
		t.Fatalf("التجديدُ الثاني: %s", got)
	}
	newAccess := tokenField(got, "access_token")
	newExp := tokenField(got, "access_expires_at")
	t.Logf("D19: التجديدُ — access_expires_at=%q · رمزٌ جديد=%t",
		newExp, newAccess != oldAccess)
	if newExp == "" {
		t.Fatal("**التجديدُ لا يقول متى ينتهي الرمزُ الجديد** — " +
			"**فتُمحى المعرفةُ بعد دورةٍ واحدة** (`D19`).")
	}
	// **ولا يُشترَط اختلافُ نصّ رمز الوصول هنا** — **وقيس (دورةُ ٥٨)
	// أنّ تجديدين في الثانية نفسِها يردّان نصّاً واحداً بعينه**:
	// **`iat` و`exp` بدقّة الثانية، والتوقيعُ حتميّ** — **فالنصّان
	// يتطابقان وليس ذلك عطباً.**
	//
	// **والاختلافُ مقيسٌ حيث يعني شيئاً**: **في التعافي رمزٌ عمرُه
	// ربعُ ساعةٍ فلا يتطابق** — **ويُقاس على السلك في
	// `D19ReconnectTest.t3`.**
	//
	// **والمقيسُ هنا ما هو عقدُ المحرّك حقّاً**: **رمزُ التجديد
	// يُدوَّر**، **والرمزُ المُسلَّم يعمل.**
	if newAccess == "" {
		t.Fatal("**لا رمزَ وصولٍ في ردّ التجديد**")
	}
	if nr := tokenField(got, "refresh_token"); nr == "" || nr == raw {
		t.Errorf("**رمزُ التجديد لم يُدوَّر** — **وتدويرُه عقدُ `XG-40`**")
	}
	if res := h.GET(mePath, newAccess); res.Code != 200 {
		t.Errorf("**الرمزُ المُسلَّمُ من التجديد لا يعمل** (%d)", res.Code)
	}

	// ── والعائلةُ نفسُها — `XG-39` لم تُمَسّ ──────────────────────
	var newest string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT session_id::text FROM refresh_tokens
		 WHERE user_id = $1::uuid AND revoked_at IS NULL
		 ORDER BY created_at DESC LIMIT 1`, drv.ID).Scan(&newest); err != nil {
		t.Fatalf("أحدثُ عائلة: %v", err)
	}
	if newest != sid {
		t.Errorf("**التجديدُ أنشأ عائلةً ثانية**: %s ≠ %s", newest, sid)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ودليلُ الجهاز يبقى موجوداً**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذا حارسُ وجودٍ لا حارسُ سلوك** — **والسلوكُ يُقاس في
// `mobile/shared` بمقبسٍ حقيقيّ.** **ويُقال صراحةً لئلّا يُحسَب
// برهاناً على ما لا يبرهنه.**
//
// **ولولاه لَحُذفت فحوصُ أندرويد ولم يُنبَّه إليه أحد**: **حزمةُ Go
// خضراء، والسجلُّ يقول `D19` مُغلَق.**
func TestD19_ClientEvidenceIsRegistered(t *testing.T) {
	const testFile = "../../../mobile/shared/src/test/kotlin/com/rahalgo/shared/net/D19ReconnectTest.kt"
	const stressFile = "../../../mobile/shared/src/test/kotlin/com/rahalgo/shared/net/D19StressTest.kt"
	const authFile = "../../../mobile/shared/src/main/kotlin/com/rahalgo/shared/net/RealtimeAuth.kt"
	const socketFile = "../../../mobile/shared/src/main/kotlin/com/rahalgo/shared/net/LiveSocket.kt"

	must := func(path string) string {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("**دليلُ `D19` غائبٌ عن الشجرة**: %s — %v", path, err)
		}
		return string(b)
	}

	tests := must(testFile)
	for _, name := range []string{
		"t1_ordinaryTransportDropDoesNotRefresh",
		"t2_expiredAccessRefreshesOnceAndConnects",
		"t3_handshakeCarriesNewTokenNotStaleCapture",
		"t4t5_revokedAndBlockedDoNotHammerRefresh",
		"t6_authorityUnavailableKeepsCredentials",
		"t8_concurrentTriggersRefreshOnce",
		"t9_recoveryLeavesOneSocket",
		"t11_noManualActionNeeded",
	} {
		if !strings.Contains(tests, name) {
			t.Errorf("**فحصُ `D19` اختفى**: %s", name)
		}
	}
	stress := must(stressFile)
	for _, name := range []string{
		"expiredRecovery100", "normalReconnect100",
		"unavailable50AndRevoked50", "concurrentRefresh100",
	} {
		if !strings.Contains(stress, name) {
			t.Errorf("**تكرارُ `D19` اختفى**: %s", name)
		}
	}

	// ── ولا يُعاد قراءةُ المخزن مباشرةً في الحلقة ─────────────────
	//
	// **وهو نصُّ العطب**: `"?token=" + session.accessToken()`.
	socket := must(socketFile)
	if strings.Contains(socket, "session.accessToken()") {
		t.Error("**الحلقةُ عادت تقرأ المخزنَ مباشرةً** — " +
			"**وذاك `D19` بعينه.**")
	}
	if !strings.Contains(socket, "auth.tokenForConnect()") {
		t.Error("**الحلقةُ لا تطلب رمزاً من سلطته**")
	}

	// ── والتجديدُ مشروطٌ بالمنتهى لا بكلّ انقطاع ──────────────────
	auth := must(authFile)
	if !strings.Contains(auth, "if (expired())") {
		t.Error("**التجديدُ صار بلا شرطِ انتهاء** — " +
			"**فيُدوَّر رمزٌ عند كلّ انقطاعِ نقل.**")
	}

	// ── ومصفوفةُ أندرويد تسجّله مُشغَّلاً ─────────────────────────
	var automated, onDevice int
	for _, c := range androidmap.Cases {
		for _, r := range c.Registers {
			if r != "D19" {
				continue
			}
			if c.Automation == androidmap.Automated {
				automated++
				if c.Result != androidmap.Pass {
					t.Errorf("**حالةٌ آليّةٌ لـ`D19` ليست ناجحة**: %s ⇒ %s",
						c.ID, c.Result)
				}
			} else {
				onDevice++
			}
		}
	}
	t.Logf("D19: حالاتٌ آليّةٌ=%d · على جهازٍ=%d", automated, onDevice)
	if automated == 0 {
		t.Error("**لا حالةَ آليّةٌ مسجَّلةٌ لـ`D19` في مصفوفة أندرويد**")
	}
	if onDevice == 0 {
		t.Error("**حُذف قبولُ الجهاز لـ`D19`** — " +
			"**والمحلّيُّ لا يُغني عن جهازٍ حقيقيّ** (`BG-01`).")
	}
}
