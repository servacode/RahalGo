package qa

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/obs"
)

// ══════════════════════════════════════════════════════════════════════
// **لماذا رُفضت المصافحة — دورةُ ٧٠أ (فجوةُ ٦٩ج)**
// ══════════════════════════════════════════════════════════════════════
//
// **ثلاثةُ أسبابٍ مختلفةٍ كانت تردّ ٤٠١ واحدةً، ولا سطرَ يقول أيُّها
// وقع.** **فلمّا لُوحظت حلقةُ إعادةِ وصلٍ في الإنتاج لم يُعرف سببُها
// إلّا باستنتاجٍ من إيقاعها** — وذاك استدلالٌ لا قياس.
//
// # والفرعُ يُمشى لا يُحقَن
//
// **ومن نادى `obs.WSAuth` في الاختبار مباشرةً أثبت أنّ العدّادَ يعدّ،
// ولم يُثبت أنّ المحرّكَ ينادِيه.** **فكلُّ حالةٍ هنا مصافحةٌ حقيقيّةٌ
// عبر المسار الحيّ.**

// wsReasonCounts **يقرأ عدّادَ الأسباب من الباب المحميّ نفسِه.**
//
// **ومن قرأه من `obs` مباشرةً تخطّى نصفَ العقد**: **الباب هو ما
// سيقرؤه Sentinel.**
func wsReasonCounts(t *testing.T, hh *Harness, tok string) map[string]int64 {
	t.Helper()
	r := hh.GET(opsHealthPath, tok)
	if r.Code != http.StatusOK {
		t.Fatalf("**بابُ الرصد لا يُجيب** — %d: %s", r.Code, r.Body)
	}
	// **والردُّ ملفوفٌ بـ`data`** — عقدُ `httpx.JSON`.
	var env struct {
		Data struct {
			WS struct {
				Auth map[string]int64 `json:"auth"`
			} `json:"ws"`
		} `json:"data"`
	}
	if err := json.Unmarshal(r.Body, &env); err != nil {
		t.Fatalf("**الجسدُ ليس JSON**: %v", err)
	}
	if env.Data.WS.Auth == nil {
		t.Fatalf("**عدّادُ الأسباب غائبٌ من الردّ** — %s", r.Body)
	}
	return env.Data.WS.Auth
}

// TestOps70A_B1_WSReasonsAreDistinguished **كلُّ فرعٍ يُعدّ باسمه.**
func TestOps70A_B1_WSReasonsAreDistinguished(t *testing.T) {
	// **ولا توازيَ هنا** — `XG-41C`: **العدّادُ مشتركٌ في العمليّة
	// كلِّها، ومن وازى قرأ ما عدّه غيرُه.** (و`-p 1` يحمي بين الحزم
	// لا داخلَها.)
	hh := New(t)
	capRole(t, hh, "qa-ws-reason", authz.ObservabilityRead)
	_, obsTok := capUser(t, hh, "qa-ws-reason")

	type step struct {
		name   string
		reason string
		want   int
		call   func() Res
	}

	u := hh.NewUser("customer")

	steps := []step{
		{
			// **مشوَّهٌ** — نصٌّ ليس توكناً أصلاً.
			name:   "مشوَّه",
			reason: string(obs.WSInvalidToken),
			want:   http.StatusUnauthorized,
			call:   func() Res { return hh.GET("/api/v1/ws?token=not-a-jwt-at-all", "") },
		},
		{
			// **وغائبٌ يمرّ بالفرع نفسِه** — ولا فرعَ ثالثَ له.
			name:   "غائب",
			reason: string(obs.WSInvalidToken),
			want:   http.StatusUnauthorized,
			call:   func() Res { return hh.GET("/api/v1/ws", "") },
		},
		{
			// **ومنتهٍ** — **وهو وحدَه ما يشفيه تجديد.**
			name:   "منتهٍ",
			reason: string(obs.WSExpiredAccess),
			want:   http.StatusUnauthorized,
			call: func() Res {
				return hh.GET("/api/v1/ws?token="+hh.ExpiredToken(u.ID, "customer"), "")
			},
		},
	}

	for _, s := range steps {
		before := wsReasonCounts(t, hh, obsTok)[s.reason]
		got := s.call()
		if got.Code != s.want {
			t.Fatalf("%s: **الحالُ %d والمنتظَرُ %d** — %s", s.name, got.Code, s.want, got.Body)
		}
		after := wsReasonCounts(t, hh, obsTok)[s.reason]
		if after != before+1 {
			t.Errorf("%s: **العدّادُ %q لم يتقدّم** — %d ← %d",
				s.name, s.reason, before, after)
		}
		t.Logf("B1: %-8s ⇒ %d · %s = %d", s.name, got.Code, s.reason, after)
	}
}

// TestOps70A_B2_ExpiredIsNotConfusedWithInvalid **الشاهدُ السالب.**
//
// **ولو صُنّف المنتهي «مشوَّهاً» لَمرّ B1 كاملاً** — **العدّادان
// يتقدّمان، وكلُّ واحدٍ منهما يتقدّم.** **فالمقياسُ أن يتقدّم واحدٌ
// ويثبت الآخر.**
func TestOps70A_B2_ExpiredIsNotConfusedWithInvalid(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa-ws-sep", authz.ObservabilityRead)
	_, obsTok := capUser(t, hh, "qa-ws-sep")
	u := hh.NewUser("customer")

	before := wsReasonCounts(t, hh, obsTok)
	if r := hh.GET("/api/v1/ws?token="+hh.ExpiredToken(u.ID, "customer"), ""); r.Code != http.StatusUnauthorized {
		t.Fatalf("**المنتهي يجب ٤٠١** — %d", r.Code)
	}
	after := wsReasonCounts(t, hh, obsTok)

	if after[string(obs.WSExpiredAccess)] != before[string(obs.WSExpiredAccess)]+1 {
		t.Errorf("**عدّادُ المنتهي لم يتقدّم**")
	}
	if after[string(obs.WSInvalidToken)] != before[string(obs.WSInvalidToken)] {
		t.Errorf("**وعدّادُ المشوَّه تقدّم معه** — %d ← %d: **الفرعان مخلوطان**",
			before[string(obs.WSInvalidToken)], after[string(obs.WSInvalidToken)])
	}
	t.Logf("B2: المنتهي وحدَه تقدّم · المشوَّهُ ثابتٌ عند %d",
		after[string(obs.WSInvalidToken)])
}

// TestOps70A_B3_RevokedSessionHasItsOwnReason **المُبطَلةُ سببٌ ثالث.**
//
// **وتردّ ٤٠١ كالمنتهي** — **وهي عقدُ `R16`** — **ولا يُميَّز بينهما
// بالرمز.** **وهذا هو بيتُ القصيد: العملُ المطلوبُ مختلفٌ تماماً.**
func TestOps70A_B3_RevokedSessionHasItsOwnReason(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa-ws-revoked", authz.ObservabilityRead)
	_, obsTok := capUser(t, hh, "qa-ws-revoked")

	u, tok := capUser(t, hh)
	// **ورمزٌ صالحٌ تماماً وجلستُه تُبطَل تحته.**
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1::uuid`,
		u.ID); err != nil {
		t.Fatalf("إبطالُ العائلة: %v", err)
	}

	before := wsReasonCounts(t, hh, obsTok)
	r := hh.GET("/api/v1/ws?token="+tok, "")
	if r.Code != http.StatusUnauthorized {
		t.Fatalf("**المُبطَلةُ تُردّ ٤٠١** — %d: %s", r.Code, r.Body)
	}
	after := wsReasonCounts(t, hh, obsTok)

	if after[string(obs.WSRevokedSession)] != before[string(obs.WSRevokedSession)]+1 {
		t.Fatalf("**عدّادُ المُبطَلة لم يتقدّم** — %d ← %d",
			before[string(obs.WSRevokedSession)], after[string(obs.WSRevokedSession)])
	}
	if after[string(obs.WSExpiredAccess)] != before[string(obs.WSExpiredAccess)] {
		t.Errorf("**وخُلطت بالمنتهي** — والرمزُ لم ينتهِ أصلاً")
	}
	t.Logf("B3: ٤٠١ واحدةٌ وسببان مفترقان · مُبطَلة = %d",
		after[string(obs.WSRevokedSession)])
}

// TestOps70A_B4_BlockedAccountIsForbiddenNotUnauthorized **المحظورُ ٤٠٣.**
//
// **وهو عقدُ `D14`** — **ويُعدّ على حدة**: **حلقةُ إعادةِ وصلٍ سببُها
// حظرٌ غيرُ حلقةٍ سببُها رمزٌ منتهٍ.**
func TestOps70A_B4_BlockedAccountIsForbiddenNotUnauthorized(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa-ws-blocked", authz.ObservabilityRead)
	_, obsTok := capUser(t, hh, "qa-ws-blocked")

	u, tok := capUser(t, hh)
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE users SET status = 'blocked' WHERE id = $1::uuid`, u.ID); err != nil {
		t.Fatalf("حظرُ الحساب: %v", err)
	}

	before := wsReasonCounts(t, hh, obsTok)
	r := hh.GET("/api/v1/ws?token="+tok, "")
	if r.Code != http.StatusForbidden {
		t.Fatalf("**المحظورُ يُردّ ٤٠٣ لا %d** — %s", r.Code, r.Body)
	}
	after := wsReasonCounts(t, hh, obsTok)
	if after[string(obs.WSForbidden)] != before[string(obs.WSForbidden)]+1 {
		t.Errorf("**عدّادُ المحظور لم يتقدّم** — %d ← %d",
			before[string(obs.WSForbidden)], after[string(obs.WSForbidden)])
	}
	t.Logf("B4: ٤٠٣ · محظور = %d", after[string(obs.WSForbidden)])
}

// TestOps70A_B5_SuccessIsCountedAfterUpgrade **والمقامُ يُعدّ كذلك.**
//
// **ونسبةُ رفضٍ بلا مقامٍ رقمٌ لا يُقرأ** — **مئةُ رفضٍ من مئةٍ غيرُ
// مئةٍ من مئة ألف.**
//
// **ولا يُعدّ إلّا بعد الترقية**: **من عدّه عند اجتياز التخويل عدّ
// مصافحةً لم تتمّ.** **ومِسندُ `httptest` لا يُرقّي**، فالمنتظَرُ
// هنا **ألّا يتقدّم** مع اجتياز التخويل — وهو ما يُثبت موضعَ العدّ.
func TestOps70A_B5_SuccessIsCountedAfterUpgrade(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa-ws-ok", authz.ObservabilityRead)
	_, obsTok := capUser(t, hh, "qa-ws-ok")

	_, tok := capUser(t, hh)
	before := wsReasonCounts(t, hh, obsTok)
	r := hh.GET("/api/v1/ws?token="+tok, "")
	after := wsReasonCounts(t, hh, obsTok)

	// **ورمزٌ سليمٌ لا يُعدّ رفضاً مهما كان مآلُ الترقية.**
	for _, bad := range []obs.WSReason{
		obs.WSExpiredAccess, obs.WSInvalidToken,
		obs.WSRevokedSession, obs.WSForbidden,
	} {
		if after[string(bad)] != before[string(bad)] {
			t.Errorf("**رمزٌ سليمٌ عُدّ رفضاً** — %q تقدّم (ردُّ المصافحة %d)",
				bad, r.Code)
		}
	}
	t.Logf("B5: ردُّ المصافحة %d · لا عدّادَ رفضٍ تقدّم · النجاحُ = %d",
		r.Code, after[string(obs.WSSuccess)])
}

// TestOps70A_B6_RealUpgradeCountsSuccess **مصافحةٌ كاملةٌ تُعدّ نجاحاً.**
//
// **وB5 يُثبت أنّ العدّادَ بعد الترقية** — **وهذا يُثبت أنّه يعدّ
// أصلاً.** **وبلا هذا يبقى `ws_auth_success` حقلاً لا يتحرّك أبداً،
// ونسبةُ الرفض بلا مقام.**
//
// **ومقبسٌ حقيقيٌّ على خادمٍ حقيقيّ** — `httptest.NewServer` لا
// `NewRecorder`، **فالترقيةُ تقع فعلاً.**
func TestOps70A_B6_RealUpgradeCountsSuccess(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa-ws-real", authz.ObservabilityRead)
	_, obsTok := capUser(t, hh, "qa-ws-real")

	_, tok := capUser(t, hh)
	before := wsReasonCounts(t, hh, obsTok)[string(obs.WSSuccess)]

	url := "ws" + strings.TrimPrefix(hh.Srv.URL, "http") + "/api/v1/ws?token=" + tok
	ctx, cancel := context.WithTimeout(ctxBG(), 10*time.Second)
	defer cancel()
	conn, resp, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		t.Fatalf("**المصافحةُ لم تتمّ** — %d: %v", code, err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("**المنتظَرُ ١٠١** — %d", resp.StatusCode)
	}
	defer conn.CloseNow()

	after := wsReasonCounts(t, hh, obsTok)[string(obs.WSSuccess)]
	if after != before+1 {
		t.Fatalf("**عدّادُ النجاح لم يتقدّم** — %d ← %d", before, after)
	}
	t.Logf("B6: ١٠١ حقيقيّة · النجاحُ = %d", after)
}
