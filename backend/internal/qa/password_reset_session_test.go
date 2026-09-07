package qa

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **كلمةٌ أُعيدت وجلسةٌ باقية** — `R13`
// ══════════════════════════════════════════════════════════════════════
//
// # العقد
//
// **إعادةُ الإدارة لكلمةِ حسابٍ فعلُ استرداد**: **جهازٌ ضاع · حسابٌ
// اختُرق · موظّفٌ غادر.** **فإن بقيت جلستُه القائمةُ تعمل لم يُسترَدّ
// شيء** — **وصاحبُ الوصول القديمُ يبقى داخلاً، ولا يعلم به من أعاد
// الكلمة.**
//
// # وما قِيس في المصدر
//
// **`handleAdminResetPassword` يكتب البصمةَ ويرفع `must_change_password`
// ويسجّل تدقيقاً** — **ولا يُبطل جلسةً ولا رمزَ تجديد.**
//
// **ورمزُ التجديد يبقى يدور إلى الأبد** — **فالمهلةُ لا تُنقذ.**

// resetPassword إعادةٌ إداريّةٌ عبر المسار الحقيقيّ.
func resetPassword(t *testing.T, h *Harness, userID, pw string) Res {
	t.Helper()
	admin := h.NewUser("admin")
	got := h.POST("/api/v1/admin/users/"+userID+"/password", admin.Token,
		map[string]any{"password": pw})
	if got.Code >= 400 {
		t.Fatalf("إعادةُ الكلمة: %s", got)
	}
	return got
}

// ══════════════════════════════════════════════════════════════════════
// **T1+T2 · رمزُ الوصول ورمزُ التجديد يموتان بالإعادة**
// ══════════════════════════════════════════════════════════════════════
func TestR13_T1T2_ResetKillsAccessAndRefresh(t *testing.T) {
	h := New(t)
	u := h.NewUser("customer")
	raw, sid := issuedRefresh(t, h, u.ID, "web")
	tok := h.TokenWithSession(u.ID, sid, "customer")

	if res := h.GET(mePath, tok); res.Code != http.StatusOK {
		t.Fatalf("الجلسةُ لا تعمل قبل الإعادة: %d", res.Code)
	}

	resetPassword(t, h, u.ID, "Qa!Reset-2026")

	live := liveRows(t, h, sid)
	access := h.GET(mePath, tok)
	refresh := h.POST("/api/v1/auth/refresh", "", map[string]any{"refresh_token": raw})
	t.Logf("T1/T2: بعد الإعادة — صفوفٌ حيّةٌ=%d · الوصولُ ⇒ %d · التجديدُ ⇒ %d",
		live, access.Code, refresh.Code)

	if live != 0 {
		t.Errorf("**الإعادةُ لم تُبطل الجلسة**: صفوفٌ حيّةٌ=%d — "+
			"**فالحسابُ لم يُسترَدّ.** (`R13`)", live)
	}
	if access.Code == http.StatusOK {
		t.Errorf("**رمزُ وصولٍ قديمٌ يعمل بعد إعادة الكلمة** (%d) — "+
			"**وصاحبُ الوصول القديمُ ما زال داخلاً.** (`R13`)", access.Code)
	}
	if refresh.Code < 400 {
		t.Errorf("**رمزُ تجديدٍ قديمٌ يدور بعد إعادة الكلمة** (%d) — "+
			"**فالمهلةُ لا تُنقذ، والوصولُ دائم.** (`R13`)", refresh.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T3 · وكلُّ الأجهزة لا جهازٌ واحد**
// ══════════════════════════════════════════════════════════════════════
func TestR13_T3_ResetKillsEveryDevice(t *testing.T) {
	h := New(t)
	u := h.NewUser("driver")
	tokA, sidA := deliverySession(t, h, u.ID, "driver", "driver")
	tokB, sidB := deliverySession(t, h, u.ID, "web", "driver")

	resetPassword(t, h, u.ID, "Qa!Reset-2026")
	t.Logf("T3: أ صفوفٌ حيّةٌ=%d ⇒ %d · ب صفوفٌ حيّةٌ=%d ⇒ %d",
		liveRows(t, h, sidA), h.GET(mePath, tokA).Code,
		liveRows(t, h, sidB), h.GET(mePath, tokB).Code)

	for name, tok := range map[string]string{"أ": tokA, "ب": tokB} {
		if h.GET(mePath, tok).Code == http.StatusOK {
			t.Errorf("**الجهازُ %s ما زال داخلاً بعد الإعادة**", name)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T4 · ولا تمتدُّ الإعادةُ إلى حسابٍ آخر**
// ══════════════════════════════════════════════════════════════════════
func TestR13_T4_ResetDoesNotTouchOtherAccounts(t *testing.T) {
	h := New(t)
	victim := h.NewUser("customer")
	bystander := h.NewUser("customer")
	_, vSid := deliverySession(t, h, victim.ID, "web", "customer")
	bTok, bSid := deliverySession(t, h, bystander.ID, "web", "customer")

	resetPassword(t, h, victim.ID, "Qa!Reset-2026")
	t.Logf("T4: الهدفُ صفوفٌ حيّةٌ=%d · الآخرُ صفوفٌ حيّةٌ=%d ⇒ %d",
		liveRows(t, h, vSid), liveRows(t, h, bSid), h.GET(mePath, bTok).Code)

	if liveRows(t, h, bSid) == 0 || h.GET(mePath, bTok).Code != http.StatusOK {
		t.Errorf("**الإعادةُ أخرجت حساباً آخر** — **وعقوبةٌ على غير فاعل.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T5 · والدخولُ بالكلمة الجديدة يعمل**
// ══════════════════════════════════════════════════════════════════════
//
// **فالإبطالُ استردادٌ لا إغلاقُ حساب.**
func TestR13_T5_NewPasswordStillLogsIn(t *testing.T) {
	h := New(t)
	u := h.NewUser("customer")
	const pw = "Qa!Reset-2026"
	resetPassword(t, h, u.ID, pw)

	var phone string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT phone FROM users WHERE id = $1::uuid`, u.ID).Scan(&phone); err != nil {
		t.Fatalf("هاتفُ الحساب: %v", err)
	}
	in := h.POST("/api/v1/auth/login", "", map[string]any{
		"phone": phone, "password": pw, "client": "web",
	})
	t.Logf("T5: الدخولُ بالكلمة الجديدة ⇒ %d", in.Code)
	if in.Code >= 400 {
		t.Errorf("**لا دخولَ بالكلمة الجديدة** (%s) — **فالإبطالُ صار "+
			"إغلاقاً.**", in)
	}
	if tokenField(in, "access_token") == "" {
		t.Error("**لا رمزَ وصولٍ بعد الدخول الجديد**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T6 · إعادةٌ تتزامن مع تجديد — ولا رمزَ يُفلت بعدها**
// ══════════════════════════════════════════════════════════════════════
func TestR13_T6_ResetVsRefreshRace(t *testing.T) {
	h := New(t)
	u := h.NewUser("customer")
	raw, sid := issuedRefresh(t, h, u.ID, "web")
	admin := h.NewUser("admin")

	var refreshed Res
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "تجديدُ الرمز", Do: func(context.Context) any {
			refreshed = h.POST("/api/v1/auth/refresh", "",
				map[string]any{"refresh_token": raw})
			return refreshed
		}},
		Actor{Name: "إعادةُ الكلمة", Do: func(context.Context) any {
			return h.POST("/api/v1/admin/users/"+u.ID+"/password", admin.Token,
				map[string]any{"password": "Qa!Reset-2026"})
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	live := liveRows(t, h, sid)
	t.Logf("T6: التجديدُ ردّ %d · صفوفٌ حيّةٌ بعد الإعادة=%d — %s",
		refreshed.Code, live, r)

	if live != 0 {
		t.Errorf("**بقيت صفوفٌ حيّةٌ بعد إعادة الكلمة**: %d — "+
			"**ورمزٌ يُفلت من الاسترداد خرقٌ أمنيّ.**", live)
	}
	if refreshed.Code < 400 {
		tok := tokenField(refreshed, "access_token")
		after := h.GET(mePath, tok)
		t.Logf("T6: الرمزُ الذي خرج من السباق ⇒ %d", after.Code)
		if after.Code == http.StatusOK {
			t.Errorf("**رمزُ وصولٍ أفلت من إعادة الكلمة وما زال يعمل** (%d)",
				after.Code)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T7 · والإبطالُ في الحقيقة الموثوقة لا في الخبيئة وحدَها**
// ══════════════════════════════════════════════════════════════════════
//
// **وعقدُ `R16` يسبق**: **مفتاحُ `Redis` تسريعٌ** — **فيُمحى ويبقى
// الرفضُ قائماً بالقاعدة.**
func TestR13_T7_RevocationSurvivesCacheLoss(t *testing.T) {
	h := New(t)
	u := h.NewUser("customer")
	_, sid := issuedRefresh(t, h, u.ID, "web")
	tok := h.TokenWithSession(u.ID, sid, "customer")

	resetPassword(t, h, u.ID, "Qa!Reset-2026")

	// **تُمحى الخبيئةُ كأنّها سقطت وعادت نظيفة.**
	if err := h.Redis().Del(context.Background(),
		"sess:revoked:"+sid).Err(); err != nil {
		t.Fatalf("محوُ المفتاح: %v", err)
	}
	n, _ := h.Redis().Exists(context.Background(), "sess:revoked:"+sid).Result()
	res := h.GET(mePath, tok)
	t.Logf("T7: المفتاحُ بعد المحو=%d · الردّ ⇒ %d", n, res.Code)

	if res.Code == http.StatusOK {
		t.Errorf("**الإبطالُ عاش في الخبيئة وحدَها** (%d) — "+
			"**وعقدُ `R16` يقول القاعدةُ هي الحقيقة.**", res.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **حارسٌ بنيويّ: الإعادةُ لا تعود جملةً في معالِج**
// ══════════════════════════════════════════════════════════════════════
//
// **والعيبُ لم يكن في قيمةٍ بل في موضع**: **المعالِجُ كتب البصمةَ
// بـSQL خامٍّ وتجاوز خدمةَ الهويّة كلَّها** — **فلا حدَّ معاملةٍ ولا
// إبطال.**
//
// **فيُحرَس الموضعُ لا النتيجة** — **ومن أعادها غداً لم يخالف نوعاً.**
func TestR13_ResetGoesThroughIdentityService(t *testing.T) {
	root := r16Root(t)

	h := mustRead(t, filepath.Join(root,
		"backend/internal/server/admin_users_handlers.go"))
	start := strings.Index(h, "func (s *Server) handleAdminResetPassword")
	if start < 0 {
		t.Fatal("**اختفى معالِجُ إعادة الكلمة** — يُعاد القياس")
	}
	end := strings.Index(h[start:], "\n}\n")
	body := h[start : start+end]

	if !strings.Contains(body, "s.identity.AdminResetPassword(") {
		t.Error("**المعالِجُ لا ينادي خدمةَ الهويّة** — " +
			"**فبصمةٌ تُكتب بلا إبطالِ جلسة.** (`R13`)")
	}
	if strings.Contains(body, "UPDATE users") {
		t.Error("**المعالِجُ يكتب `users` بنفسه** — " +
			"**والاستردادُ فعلُ خدمةٍ بحدّ معاملة.** (`R13`)")
	}

	// ── وثلاثةُ أفعالٍ في معاملةٍ واحدة ───────────────────────────
	svc := mustRead(t, filepath.Join(root, "backend/internal/identity/admin.go"))
	s2 := strings.Index(svc, "func (s *Service) AdminResetPassword")
	if s2 < 0 {
		t.Fatal("**لا خدمةَ لإعادة الكلمة**")
	}
	fn := svc[s2 : s2+strings.Index(svc[s2:], "\n}\n")]
	for _, want := range []string{
		"Begin(ctx)",            // معاملةٌ واحدة
		"sessions_revoked_at",   // الحِقبة
		"UPDATE refresh_tokens", // الإبطال
	} {
		if !strings.Contains(fn, want) {
			t.Errorf("**الإعادةُ فقدت %q** — **واستردادٌ ناقصٌ يُطمئن "+
				"من طلبه وهو لم يسترجع شيئاً.**", want)
		}
	}

	// ── والحِقبةُ تُقرأ عند إصدار جلسةٍ مجدَّدة ────────────────────
	repo := mustRead(t, filepath.Join(root, "backend/internal/identity/repo.go"))
	if !strings.Contains(repo, "u.sessions_revoked_at") {
		t.Error("**`StoreRefresh` لا يقرأ الحِقبة** — " +
			"**فتجديدٌ سبق الإبطالَ يُدرج بعده ويُفلت.** (`R13` · T6)")
	}
}
