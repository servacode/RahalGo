package qa

import (
	"context"
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **كلمةٌ بُدّلت تُخرج من عرفها — إلّا صاحبَها** — `XG-40`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **`SetPassword` تتحقّق من الحاليّة وتكتب الجديدةَ وتمضي** — **ولا
// تُبطل جلسةً واحدة.** **فمن شكّ أنّ أحداً يعرف كلمتَه فبدّلها لم
// يُخرجه**: جلسةُ المتطفّل تعمل ورمزُ تجديده يدور.
//
// # وعقدُ المالك (٢٠٢٦-٠٩-٠٧)
//
//	تبقى **العائلةُ التي نفّذت التغيير** · وتُقطَع البواقي
//
// **ولا تُقطَع كلُّها** — **وإلّا أخرج نفسَه من الصفحة التي يقف
// عليها.**
//
// **والإعادةُ الإداريّةُ والحظرُ والإخراجُ الشاملُ تقطع كلَّ شيء
// ولا استثناء** — **وأقوى الإبطالين يغلب.**

const setPwPath = "/api/v1/auth/password"

// twoSessions حسابٌ بعائلتَي جلسةٍ حقيقيّتَين — واحدةٌ برمز تجديدٍ حيّ.
//
// **والأولى «هذا الجهاز»** — من مسار الدخول، فلها رمزُ تجديدٍ يُقبَل.
func twoSessions(t *testing.T, h *Harness) (u *User, mine, other string, mineRaw, mineSid, otherSid string) {
	t.Helper()
	u = h.NewUser("customer")
	mineRaw, mineSid = issuedRefresh(t, h, u.ID, "web")
	mine = h.TokenWithSession(u.ID, mineSid, "customer")
	other, otherSid = deliverySession(t, h, u.ID, "android", "customer")
	return
}

// changePassword تبديلُ الكلمة من جلسةٍ بعينها — عبر المسار الحقيقيّ.
func changePassword(t *testing.T, h *Harness, token, cur, next string) Res {
	t.Helper()
	return h.POST(setPwPath, token, map[string]any{
		"current_password": cur, "password": next,
	})
}

// ══════════════════════════════════════════════════════════════════════
// **X1+X2+X4+X5 · العقدُ الأساس**
// ══════════════════════════════════════════════════════════════════════
func TestXG40_Contract_KeepMineRevokeOthers(t *testing.T) {
	hh := New(t)
	u, mine, other, mineRaw, mineSid, otherSid := twoSessions(t, hh)

	// **وكلمةُ الدخول التي وضعها `issuedRefresh`.**
	got := changePassword(t, hh, mine, "Qa!Refresh-2026", "Qa!Changed-2026")
	if got.Code >= 400 {
		t.Fatalf("تبديلُ الكلمة: %s", got)
	}

	// ── X2 · جلستي تبقى ──────────────────────────────────────────
	keep := hh.GET(mePath, mine)
	t.Logf("X2: جلستي بعد التبديل — صفوفٌ حيّةٌ=%d ⇒ %d",
		liveRows(t, hh, mineSid), keep.Code)
	if keep.Code != http.StatusOK {
		t.Errorf("**أخرج نفسَه بتبديل كلمته** (%d) — **وعكسُ ما طُلب.**",
			keep.Code)
	}

	// ── X4 · وغيرُها يُقطَع ───────────────────────────────────────
	gone := hh.GET(mePath, other)
	t.Logf("X4: الجهازُ الآخر — صفوفٌ حيّةٌ=%d ⇒ %d",
		liveRows(t, hh, otherSid), gone.Code)
	if gone.Code == http.StatusOK {
		t.Errorf("**جلسةُ جهازٍ آخرَ نجت من تبديل الكلمة** (%d) — "+
			"**فمن بدّلها لم يُخرج أحداً.** (`XG-40`)", gone.Code)
	}
	if liveRows(t, hh, otherSid) != 0 {
		t.Errorf("**بقيت صفوفٌ حيّةٌ للعائلة الأخرى**: %d",
			liveRows(t, hh, otherSid))
	}

	// ── X3 · وتجديدي يُبقي العائلةَ نفسَها ────────────────────────
	ref := hh.POST("/api/v1/auth/refresh", "",
		map[string]any{"refresh_token": mineRaw})
	t.Logf("X3: تجديدُ جلستي ⇒ %d", ref.Code)
	if ref.Code >= 400 {
		t.Fatalf("**تجديدُ جلستي رُفض بعد تبديل كلمتي** (%s)", ref)
	}
	var families int
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT count(DISTINCT session_id) FROM refresh_tokens
		 WHERE user_id = $1::uuid AND revoked_at IS NULL
		   AND expires_at > now()`, u.ID).Scan(&families); err != nil {
		t.Fatalf("عائلاتُ الجلسة: %v", err)
	}
	t.Logf("X3: عائلاتٌ حيّةٌ بعد التجديد = %d (والمرتقَبُ ١)", families)
	if families != 1 {
		t.Errorf("**التجديدُ لم يُبقِ العائلةَ نفسَها**: %d", families)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **X5 · ورمزُ تجديدِ الآخرين يموت**
// ══════════════════════════════════════════════════════════════════════
func TestXG40_X5_OtherRefreshTokenDenied(t *testing.T) {
	hh := New(t)
	u := hh.NewUser("customer")

	// **عائلتان لكلٍّ رمزُ تجديدٍ حقيقيّ** — دخولان من نوعين.
	firstRaw, firstSid := issuedRefresh(t, hh, u.ID, "web")
	secondRaw, secondSid := loginAgain(t, hh, u.ID, "android-customer")
	mine := hh.TokenWithSession(u.ID, firstSid, "customer")

	if got := changePassword(t, hh, mine, "Qa!Refresh-2026", "Qa!Changed-2026"); got.Code >= 400 {
		t.Fatalf("تبديلُ الكلمة: %s", got)
	}

	mineRef := hh.POST("/api/v1/auth/refresh", "", map[string]any{"refresh_token": firstRaw})
	otherRef := hh.POST("/api/v1/auth/refresh", "", map[string]any{"refresh_token": secondRaw})
	t.Logf("X5: تجديدي ⇒ %d · تجديدُ الآخر ⇒ %d (عائلتُه %s…)",
		mineRef.Code, otherRef.Code, secondSid[:8])

	if mineRef.Code >= 400 {
		t.Errorf("**تجديدي رُفض** (%d)", mineRef.Code)
	}
	if otherRef.Code < 400 {
		t.Errorf("**رمزُ تجديدِ جهازٍ آخرَ ما زال يدور** (%d)", otherRef.Code)
	}
}

// loginAgain **دخولٌ ثانٍ من نوعِ عميلٍ آخر** — فلا يُبطل الأوّل.
//
// **والنوعُ ترويسةٌ لا حقلٌ في الجسد** (`X-RahalGo-Client`)، **وصيغتُه
// `منصّة-تطبيق` مغلقةٌ** (`normalizeClient`) — **وما لا يُصرّح متصفّح.**
//
// **وقيس**: دخولان بلا ترويسةٍ يقعان كلاهما `web`، **فيُبطل الثاني
// الأوّلَ بقاعدة الجلسة الواحدة لكلّ نوع** — **وذاك سلوكٌ صحيحٌ لا
// عيب**، لكنّه يُفسد تركيبةَ «عائلتان حيّتان».
func loginAgain(t *testing.T, hh *Harness, userID, kind string) (raw, sid string) {
	t.Helper()
	var phone string
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT phone FROM users WHERE id = $1::uuid`, userID).Scan(&phone); err != nil {
		t.Fatalf("هاتفُ الحساب: %v", err)
	}
	in := hh.Call("POST", "/api/v1/auth/login", "", map[string]any{
		"phone": phone, "password": "Qa!Refresh-2026",
	}, map[string]string{"X-RahalGo-Client": kind})
	if in.Code >= 400 {
		t.Skipf("تعذّر الدخولُ الثاني: %s", in)
	}
	raw = tokenField(in, "refresh_token")
	if raw == "" {
		t.Skipf("لا رمزَ تجديدٍ في الدخول الثاني: %s", in)
	}
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT session_id::text FROM refresh_tokens
		 WHERE user_id = $1::uuid AND revoked_at IS NULL AND expires_at > now()
		 ORDER BY created_at DESC LIMIT 1`, userID).Scan(&sid); err != nil {
		t.Fatalf("معرّفُ الجلسة الثانية: %v", err)
	}
	return raw, sid
}

// ══════════════════════════════════════════════════════════════════════
// **X6 · وفقدُ الخبيئة لا يُحيي ما قُطع**
// ══════════════════════════════════════════════════════════════════════
func TestXG40_X6_CacheLossDoesNotResurrect(t *testing.T) {
	hh := New(t)
	_, mine, other, _, _, otherSid := twoSessions(t, hh)

	if got := changePassword(t, hh, mine, "Qa!Refresh-2026", "Qa!Changed-2026"); got.Code >= 400 {
		t.Fatalf("تبديلُ الكلمة: %s", got)
	}
	if err := hh.Redis().Del(context.Background(),
		"sess:revoked:"+otherSid).Err(); err != nil {
		t.Fatalf("محوُ المفتاح: %v", err)
	}
	n, _ := hh.Redis().Exists(context.Background(), "sess:revoked:"+otherSid).Result()
	res := hh.GET(mePath, other)
	t.Logf("X6: المفتاحُ بعد المحو=%d · الردّ ⇒ %d", n, res.Code)
	if res.Code == http.StatusOK {
		t.Errorf("**الإبطالُ عاش في الخبيئة وحدَها** (%d) — "+
			"**وعقدُ `R16` يقول القاعدةُ هي الحقيقة.**", res.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **X7 · وكلمةٌ حاليّةٌ خاطئةٌ لا تُغيّر شيئاً ولا تقطع**
// ══════════════════════════════════════════════════════════════════════
func TestXG40_X7_WrongCurrentPasswordChangesNothing(t *testing.T) {
	hh := New(t)
	_, mine, other, _, mineSid, otherSid := twoSessions(t, hh)

	got := changePassword(t, hh, mine, "Qa!WRONG-0000", "Qa!Changed-2026")
	t.Logf("X7: كلمةٌ حاليّةٌ خاطئة ⇒ %d · صفوفٌ حيّةٌ: لي=%d · للآخر=%d",
		got.Code, liveRows(t, hh, mineSid), liveRows(t, hh, otherSid))

	if got.Code < 400 {
		t.Fatalf("**قُبل تبديلٌ بكلمةٍ حاليّةٍ خاطئة** (%d)", got.Code)
	}
	if liveRows(t, hh, otherSid) == 0 || hh.GET(mePath, other).Code != http.StatusOK {
		t.Errorf("**محاولةٌ فاشلةٌ قطعت جلسات** — **ومن أخطأ كلمتَه " +
			"لا يُخرَج أحدٌ بسببه.**")
	}
	if hh.GET(mePath, mine).Code != http.StatusOK {
		t.Errorf("**محاولةٌ فاشلةٌ قطعت جلستي**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **X8 · تجديدٌ من عائلةٍ أخرى يتزامن مع التبديل — ولا يُفلت**
// ══════════════════════════════════════════════════════════════════════
func TestXG40_X8_ConcurrentOtherRefreshCannotEscape(t *testing.T) {
	hh := New(t)
	u := hh.NewUser("customer")
	firstRaw, firstSid := issuedRefresh(t, hh, u.ID, "web")
	secondRaw, secondSid := loginAgain(t, hh, u.ID, "android-customer")
	mine := hh.TokenWithSession(u.ID, firstSid, "customer")

	var refreshed Res
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "تجديدُ الآخر", Do: func(context.Context) any {
			refreshed = hh.POST("/api/v1/auth/refresh", "",
				map[string]any{"refresh_token": secondRaw})
			return refreshed
		}},
		Actor{Name: "تبديلُ الكلمة", Do: func(context.Context) any {
			return changePassword(t, hh, mine, "Qa!Refresh-2026", "Qa!Changed-2026")
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	live := liveRows(t, hh, secondSid)
	t.Logf("X8: تجديدُ الآخر ردّ %d · صفوفُه الحيّةُ=%d — %s",
		refreshed.Code, live, r)
	_ = firstRaw

	if live != 0 {
		t.Errorf("**عائلةٌ أخرى بقيت حيّةً بعد التبديل**: %d — "+
			"**ورمزٌ يُفلت خرقٌ أمنيّ.**", live)
	}
	if refreshed.Code < 400 {
		after := hh.GET(mePath, tokenField(refreshed, "access_token"))
		t.Logf("X8: الرمزُ الذي خرج ⇒ %d", after.Code)
		if after.Code == http.StatusOK {
			t.Errorf("**رمزُ وصولٍ أفلت من تبديل الكلمة** (%d)", after.Code)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **X9 · دخولٌ بالكلمة القديمة يتزامن — ولا جلسةَ تنجو**
// ══════════════════════════════════════════════════════════════════════
//
// **والحارسُ بصمةُ الكلمة في جملة الإدراج** — **فتغييرٌ يقع بين
// التحقّق والإصدار يُنتج جلسةً بكلمةٍ ماتت.**
func TestXG40_X9_ConcurrentOldPasswordLoginCannotSurvive(t *testing.T) {
	hh := New(t)
	u := hh.NewUser("customer")
	_, mineSid := issuedRefresh(t, hh, u.ID, "web")
	mine := hh.TokenWithSession(u.ID, mineSid, "customer")

	var phone string
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT phone FROM users WHERE id = $1::uuid`, u.ID).Scan(&phone); err != nil {
		t.Fatalf("هاتفُ الحساب: %v", err)
	}

	var login Res
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "دخولٌ بالقديمة", Do: func(context.Context) any {
			// **ونوعٌ آخر** — **وإلّا أبطل جلستي بقاعدة الجلسة
			// الواحدة لكلّ نوع، فقِيس شيءٌ غيرُ المقصود.**
			login = hh.Call("POST", "/api/v1/auth/login", "", map[string]any{
				"phone": phone, "password": "Qa!Refresh-2026",
			}, map[string]string{"X-RahalGo-Client": "android-customer"})
			return login
		}},
		Actor{Name: "تبديلُ الكلمة", Do: func(context.Context) any {
			return changePassword(t, hh, mine, "Qa!Refresh-2026", "Qa!Changed-2026")
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	t.Logf("X9: الدخولُ بالقديمة ردّ %d — %s", login.Code, r)

	if login.Code < 400 {
		tok := tokenField(login, "access_token")
		after := hh.GET(mePath, tok)
		t.Logf("X9: الجلسةُ التي خرجت من الدخول ⇒ %d", after.Code)
		if after.Code == http.StatusOK {
			t.Errorf("**جلسةٌ نجت من دخولٍ بكلمةٍ بُدّلت** (%d) — "+
				"**فالتبديلُ لم يُغلق البابَ خلفه.** (`XG-40`)", after.Code)
		}
	}

	// **وجلستي باقيةٌ في كلّ حال.**
	if hh.GET(mePath, mine).Code != http.StatusOK {
		t.Errorf("**أُخرجتُ من جلستي في السباق**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **X10 · وأقوى الإبطالين يغلب**
// ══════════════════════════════════════════════════════════════════════
//
// **فإعادةٌ إداريّةٌ أو حظرٌ بعد تبديلٍ ذاتيٍّ يقطعان المستثناةَ أيضاً.**
func TestXG40_X10_StrongerRevocationWins(t *testing.T) {
	hh := New(t)
	u, mine, _, mineRaw, mineSid, _ := twoSessions(t, hh)
	if got := changePassword(t, hh, mine, "Qa!Refresh-2026", "Qa!Changed-2026"); got.Code >= 400 {
		t.Fatalf("تبديلُ الكلمة: %s", got)
	}
	if hh.GET(mePath, mine).Code != http.StatusOK {
		t.Fatal("**جلستي قُطعت** — التركيبةُ خطأ")
	}

	// ── إعادةٌ إداريّة ────────────────────────────────────────────
	resetPassword(t, hh, u.ID, "Qa!AdminReset-2026")
	live := liveRows(t, hh, mineSid)
	access := hh.GET(mePath, mine)
	ref := hh.POST("/api/v1/auth/refresh", "", map[string]any{"refresh_token": mineRaw})
	t.Logf("X10: بعد الإعادة الإداريّة — صفوفٌ حيّةٌ=%d · الوصولُ ⇒ %d · التجديدُ ⇒ %d",
		live, access.Code, ref.Code)

	if live != 0 || access.Code == http.StatusOK || ref.Code < 400 {
		t.Errorf("**العائلةُ المستثناةُ نجت من إعادةٍ إداريّة** — "+
			"**وأقوى الإبطالين يجب أن يغلب.** (حيّةٌ=%d · وصولٌ=%d · تجديدٌ=%d)",
			live, access.Code, ref.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **X10b · والحظرُ كذلك يبطل الاستثناء**
// ══════════════════════════════════════════════════════════════════════
func TestXG40_X10b_BlockBeatsKeptSession(t *testing.T) {
	hh := New(t)
	u, mine, _, mineRaw, mineSid, _ := twoSessions(t, hh)
	if got := changePassword(t, hh, mine, "Qa!Refresh-2026", "Qa!Changed-2026"); got.Code >= 400 {
		t.Fatalf("تبديلُ الكلمة: %s", got)
	}

	suspend(t, hh, u.ID, "blocked")
	live := liveRows(t, hh, mineSid)
	ref := hh.POST("/api/v1/auth/refresh", "", map[string]any{"refresh_token": mineRaw})
	t.Logf("X10b: بعد الحظر — صفوفٌ حيّةٌ=%d · التجديدُ ⇒ %d · الوصولُ ⇒ %d",
		live, ref.Code, hh.GET(mePath, mine).Code)

	if live != 0 {
		t.Errorf("**المستثناةُ نجت من الحظر**: %d", live)
	}
	if ref.Code < 400 {
		t.Errorf("**تجديدُ محظورٍ نجح** (%d)", ref.Code)
	}
}
