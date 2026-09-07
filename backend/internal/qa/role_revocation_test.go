package qa

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **سحبُ الدور لا يسري** — `R15`
// ══════════════════════════════════════════════════════════════════════
//
// # ما قِيس في المصدر
//
// **`RequireRoles` يقرأ `ctxRoles`** — **ومصدرُه `claims.Roles` من
// رمز الوصول** (`middleware.go:110`). **ولا قاعدةَ تُسأل ولا خبيئة.**
//
// **فالدورُ المسحوبُ يبقى نافذاً حتّى تنتهي مهلةُ الرمز** — ربعُ
// ساعة. **ومن سُحب منه دورُ الإدارة يبقى إدارةً فيها.**
//
// # والتوثيقُ غيرُ التخويل
//
// **الجلسةُ قد تبقى صالحةً** — **والصلاحيّةُ المسحوبةُ يجب أن تقف.**
// **ولا تُبطَل الجلساتُ لأنّ دوراً تبدّل** ما لم يقل العقدُ ذلك.

// staffToken حسابٌ بدورين وجلسةٌ دائمةٌ له — **ورمزٌ يحمل الدورين.**
func staffToken(t *testing.T, hh *Harness, roles ...string) (u *User, tok, sid string) {
	t.Helper()
	u = hh.NewUser(roles[0])
	for _, r := range roles[1:] {
		if _, err := hh.Pool.Exec(ctxBG(), `
			INSERT INTO user_roles (user_id, role_code) VALUES ($1::uuid, $2)
			ON CONFLICT DO NOTHING`, u.ID, r); err != nil {
			t.Fatalf("منحُ الدور %s: %v", r, err)
		}
	}
	if err := hh.Pool.QueryRow(ctxBG(), `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, client)
		VALUES ($1::uuid, $2, now() + interval '30 days', 'web')
		RETURNING session_id::text`,
		u.ID, "qa-"+uniq("h")).Scan(&sid); err != nil {
		t.Fatalf("صفُّ الجلسة: %v", err)
	}
	return u, hh.TokenWithSession(u.ID, sid, roles...), sid
}

// revokeRole سحبُ دورٍ عبر المسار الإداريّ الحقيقيّ.
func revokeRole(t *testing.T, hh *Harness, userID, role string) Res {
	t.Helper()
	admin := hh.NewUser("admin")
	return hh.Call("DELETE", "/api/v1/admin/users/"+userID+"/roles/"+role,
		admin.Token, nil, nil)
}

// grantRole منحُ دورٍ عبر المسار الإداريّ الحقيقيّ.
func grantRole(t *testing.T, hh *Harness, userID, role string) Res {
	t.Helper()
	admin := hh.NewUser("admin")
	return hh.POST("/api/v1/admin/users/"+userID+"/roles", admin.Token,
		map[string]any{"role": role, "reason": "R15"})
}

// adminPath **فعلٌ يشترط أحدَ أدوار المكتب** — قراءةٌ لا تُغيّر شيئاً.
//
// **والبوّابةُ `RequireRoles("admin","ops","finance")`** (`server.go:722`).
const adminPath = "/api/v1/admin/users?limit=1"

// ══════════════════════════════════════════════════════════════════════
// **والدورُ المستعمَلُ في التركيبات `ops` لا `admin`**
// ══════════════════════════════════════════════════════════════════════
//
// **قيس**: `NeedsPin` تشترط رمزاً ثانياً للأدمن (`admin_pin.go:85`) —
// **فدخولُ أدمنٍ يردّ `pin_required` بلا توكنات**، فلا رمزَ تجديدٍ
// يُقاس عليه.
//
// **و`ops` يفتح البابَ نفسَه** — فالعقدُ واحدٌ والتركيبةُ أبسط.
const privRole = "ops"

// ══════════════════════════════════════════════════════════════════════
// **A1+A2+A3 · الدورُ يُسحَب فيقف — وما بقي يعمل**
// ══════════════════════════════════════════════════════════════════════
func TestR15_A1A2A3_RevokedRoleStopsAuthorizing(t *testing.T) {
	hh := New(t)
	u, tok, _ := staffToken(t, hh, privRole, "customer")

	// ── A1 · قبل السحب ───────────────────────────────────────────
	before := hh.GET(adminPath, tok)
	t.Logf("A1: فعلٌ إداريٌّ قبل السحب ⇒ %d", before.Code)
	if before.Code != http.StatusOK {
		t.Fatalf("**التركيبةُ خطأ**: الفعلُ الإداريُّ لا يعمل أصلاً (%d)",
			before.Code)
	}

	// ── السحبُ عبر المسار الحقيقيّ ────────────────────────────────
	got := revokeRole(t, hh, u.ID, privRole)
	if got.Code >= 400 {
		t.Fatalf("سحبُ الدور: %s", got)
	}
	var stillHas bool
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT EXISTS (SELECT 1 FROM user_roles
		                WHERE user_id = $1::uuid AND role_code = $2)`,
		u.ID, privRole).Scan(&stillHas); err != nil {
		t.Fatalf("قراءةُ الأدوار: %v", err)
	}

	// ── A2 · بالرمز نفسِه ────────────────────────────────────────
	after := hh.GET(adminPath, tok)
	t.Logf("A2: الدورُ في القاعدة=%v · الفعلُ بالرمز نفسِه ⇒ %d",
		stillHas, after.Code)

	if stillHas {
		t.Fatal("**السحبُ لم يقع في القاعدة** — التركيبةُ خطأ")
	}
	if after.Code == http.StatusOK {
		t.Errorf("**دورٌ سُحب وما زال يُخوِّل** (%d) — "+
			"**والرمزُ سلطةٌ لصلاحيّةٍ لم تعد قائمة.** (`R15`)", after.Code)
	}

	// ── A3 · وما بقي من أدوارٍ يعمل ──────────────────────────────
	mine := hh.GET(mePath, tok)
	t.Logf("A3: فعلٌ عاديٌّ بالرمز نفسِه ⇒ %d", mine.Code)
	if mine.Code != http.StatusOK {
		t.Errorf("**سحبُ دورٍ أخرج الحسابَ كلَّه** (%d) — "+
			"**والتوثيقُ غيرُ التخويل.**", mine.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **A4 · أدوارٌ عدّة: يُسحَب واحدٌ ويبقى الآخر**
// ══════════════════════════════════════════════════════════════════════
func TestR15_A4_MultiRoleKeepsRemaining(t *testing.T) {
	hh := New(t)
	u, tok, _ := staffToken(t, hh, privRole, "finance")

	if got := revokeRole(t, hh, u.ID, privRole); got.Code >= 400 {
		t.Fatalf("سحبُ الدور: %s", got)
	}
	var left []string
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(array_agg(role_code ORDER BY role_code), '{}')
		  FROM user_roles WHERE user_id = $1::uuid`, u.ID).Scan(&left); err != nil {
		t.Fatalf("الأدوارُ الباقية: %v", err)
	}

	// **والبابُ يقبل `finance` أيضاً** — **فيبقى مفتوحاً بدورٍ لم
	// يُسحَب**، وذاك عينُ المقصود: **لا يُختزَل الحسابُ إلى «لا
	// صلاحيّةَ له».**
	shared := hh.GET(adminPath, tok)
	t.Logf("A4: الباقي=%v · البابُ المشترك ⇒ %d", left, shared.Code)

	if len(left) != 1 || left[0] != "finance" {
		t.Fatalf("**التركيبةُ خطأ**: الباقي %v", left)
	}
	if shared.Code != http.StatusOK {
		t.Errorf("**دورٌ لم يُسحَب فقد صلاحيّتَه** (%d) — "+
			"**وسحبُ واحدٍ لا يُسقط الباقين.**", shared.Code)
	}

	// **وحسابٌ فقد كلَّ أدوار المكتب يُمنَع** — الوجهُ الآخرُ للعقد.
	other, otherTok, _ := staffToken(t, hh, privRole)
	if got := revokeRole(t, hh, other.ID, privRole); got.Code >= 400 {
		t.Fatalf("سحبُ الدور الثاني: %s", got)
	}
	none := hh.GET(adminPath, otherTok)
	t.Logf("A4: ومن فقد كلَّ أدوار المكتب ⇒ %d", none.Code)
	if none.Code == http.StatusOK {
		t.Errorf("**فقد كلَّ أدواره وما زال يمرّ** (%d)", none.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **A5 · والتجديدُ لا يُعيد ما سُحب**
// ══════════════════════════════════════════════════════════════════════
func TestR15_A5_RefreshDoesNotRestore(t *testing.T) {
	hh := New(t)
	u := hh.NewUser(privRole)
	raw, sid := issuedRefresh(t, hh, u.ID, "web")
	tok := hh.TokenWithSession(u.ID, sid, privRole)
	if hh.GET(adminPath, tok).Code != http.StatusOK {
		t.Fatal("**التركيبةُ خطأ**: الفعلُ الإداريُّ لا يعمل أصلاً")
	}

	if got := revokeRole(t, hh, u.ID, privRole); got.Code >= 400 {
		t.Fatalf("سحبُ الدور: %s", got)
	}
	ref := hh.POST("/api/v1/auth/refresh", "", map[string]any{"refresh_token": raw})
	if ref.Code >= 400 {
		t.Fatalf("التجديدُ رُفض: %s", ref)
	}
	fresh := tokenField(ref, "access_token")
	got := hh.GET(adminPath, fresh)
	t.Logf("A5: الفعلُ الإداريُّ برمزٍ مجدَّد ⇒ %d", got.Code)
	if got.Code == http.StatusOK {
		t.Errorf("**التجديدُ أعاد دوراً مسحوباً** (%d)", got.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **A6 · ودخولٌ جديدٌ لا يُصدر ما سُحب**
// ══════════════════════════════════════════════════════════════════════
func TestR15_A6_NewLoginHasNoRevokedRole(t *testing.T) {
	hh := New(t)
	u := hh.NewUser(privRole)
	issuedRefresh(t, hh, u.ID, "web")

	if got := revokeRole(t, hh, u.ID, privRole); got.Code >= 400 {
		t.Fatalf("سحبُ الدور: %s", got)
	}
	var phone string
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT phone FROM users WHERE id = $1::uuid`, u.ID).Scan(&phone); err != nil {
		t.Fatalf("هاتفُ الحساب: %v", err)
	}
	in := hh.POST("/api/v1/auth/login", "", map[string]any{
		"phone": phone, "password": "Qa!Refresh-2026",
	})
	if in.Code >= 400 {
		t.Skipf("تعذّر الدخول: %s", in)
	}
	got := hh.GET(adminPath, tokenField(in, "access_token"))
	t.Logf("A6: الفعلُ الإداريُّ بعد دخولٍ جديد ⇒ %d", got.Code)
	if got.Code == http.StatusOK {
		t.Errorf("**دخولٌ جديدٌ أصدر دوراً مسحوباً** (%d)", got.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **A9 · ومصافحةُ بثٍّ جديدةٌ لا تُخوِّل بمسحوب**
// ══════════════════════════════════════════════════════════════════════
func TestR15_A9_NewWSHandshakeDropsRevokedTopics(t *testing.T) {
	hh := New(t)
	u, tok, _ := staffToken(t, hh, privRole, "customer")
	if got := revokeRole(t, hh, u.ID, privRole); got.Code >= 400 {
		t.Fatalf("سحبُ الدور: %s", got)
	}
	res := hh.GET("/api/v1/ws?token="+tok, "")
	t.Logf("A9: مصافحةٌ بعد السحب ⇒ %d", res.Code)
	if res.Code == http.StatusUnauthorized {
		t.Log("**رُفضت المصافحةُ كلُّها** — يُقرأ في العقد")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **A11 · ومنحُ دورٍ — العقدُ المقيس**
// ══════════════════════════════════════════════════════════════════════
func TestR15_A11_GrantSemanticsMeasured(t *testing.T) {
	hh := New(t)
	u, tok, _ := staffToken(t, hh, "customer")
	before := hh.GET(adminPath, tok)

	if got := grantRole(t, hh, u.ID, "admin"); got.Code >= 400 {
		t.Skipf("منحُ الدور: %s", got)
	}
	after := hh.GET(adminPath, tok)
	t.Logf("A11: قبل المنح ⇒ %d · بعده بالرمز نفسِه ⇒ %d",
		before.Code, after.Code)

	if before.Code == http.StatusOK {
		t.Fatal("**التركيبةُ خطأ**: الزبونُ يمرّ من بابِ الإدارة")
	}
	t.Logf("A11 دلالةُ المنح = %s",
		map[bool]string{true: "فوريّةٌ من القاعدة", false: "بعد تجديدٍ أو دخول"}[after.Code == http.StatusOK])
}

// ══════════════════════════════════════════════════════════════════════
// **A10 · والأفعالُ العاديّةُ لا تتأثّر**
// ══════════════════════════════════════════════════════════════════════
func TestR15_A10_OrdinaryActionsUnaffected(t *testing.T) {
	hh := New(t)
	u, tok, _ := staffToken(t, hh, "customer")
	_ = u
	res := hh.GET(mePath, tok)
	t.Logf("A10: فعلٌ عاديّ ⇒ %d", res.Code)
	if res.Code != http.StatusOK {
		t.Errorf("**فعلٌ عاديٌّ تأثّر** (%d)", res.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **C1 · سحبٌ يتزامن مع فعلٍ مخوَّل**
// ══════════════════════════════════════════════════════════════════════
//
// **ونقطةُ التسلسل**: **قرارُ التخويل يقرأ الحقيقةَ الموثوقة** —
// **فما ثبت سحبُه قبل القراءة يُرفَض، وما لم يثبت يمرّ.** **ولا
// ارتدادَ لعمليّةٍ ثبتت قبل السحب.**
func TestR15_C1_RevokeVsPrivilegedRequest(t *testing.T) {
	hh := New(t)
	u, tok, _ := staffToken(t, hh, privRole, "customer")
	admin := hh.NewUser("admin")

	var privileged Res
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "فعلٌ مخوَّل", Do: func(context.Context) any {
			privileged = hh.GET(adminPath, tok)
			return privileged
		}},
		Actor{Name: "سحبُ الدور", Do: func(context.Context) any {
			return hh.Call("DELETE", "/api/v1/admin/users/"+u.ID+"/roles/"+privRole,
				admin.Token, nil, nil)
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	after := hh.GET(adminPath, tok)
	t.Logf("C1: الفعلُ في السباق ⇒ %d · وبعد ثبوت السحب ⇒ %d — %s",
		privileged.Code, after.Code, r)

	// **والمقيسُ ما بعد التثبيت** — لا أيُّهما سبق في الساعة.
	if after.Code == http.StatusOK {
		t.Errorf("**الفعلُ المخوَّلُ يمرّ بعد ثبوت السحب** (%d)", after.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **C2 · سحبٌ يتزامن مع تجديد**
// ══════════════════════════════════════════════════════════════════════
func TestR15_C2_RevokeVsRefresh(t *testing.T) {
	hh := New(t)
	u := hh.NewUser(privRole)
	raw, _ := issuedRefresh(t, hh, u.ID, "web")
	admin := hh.NewUser("admin")

	var ref Res
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "تجديدُ الرمز", Do: func(context.Context) any {
			ref = hh.POST("/api/v1/auth/refresh", "",
				map[string]any{"refresh_token": raw})
			return ref
		}},
		Actor{Name: "سحبُ الدور", Do: func(context.Context) any {
			return hh.Call("DELETE", "/api/v1/admin/users/"+u.ID+"/roles/"+privRole,
				admin.Token, nil, nil)
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	t.Logf("C2: التجديدُ ردّ %d — %s", ref.Code, r)
	if ref.Code < 400 {
		got := hh.GET(adminPath, tokenField(ref, "access_token"))
		t.Logf("C2: الرمزُ المجدَّدُ على بابِ الإدارة ⇒ %d", got.Code)
		if got.Code == http.StatusOK {
			t.Errorf("**رمزٌ مجدَّدٌ احتفظ بدورٍ سُحب** (%d)", got.Code)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **C3 · سحبٌ يتزامن مع دخولٍ جديد**
// ══════════════════════════════════════════════════════════════════════
func TestR15_C3_RevokeVsLogin(t *testing.T) {
	hh := New(t)
	u := hh.NewUser(privRole)
	issuedRefresh(t, hh, u.ID, "web")
	admin := hh.NewUser("admin")
	var phone string
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT phone FROM users WHERE id = $1::uuid`, u.ID).Scan(&phone); err != nil {
		t.Fatalf("هاتفُ الحساب: %v", err)
	}

	var in Res
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "دخولٌ جديد", Do: func(context.Context) any {
			in = hh.Call("POST", "/api/v1/auth/login", "", map[string]any{
				"phone": phone, "password": "Qa!Refresh-2026",
			}, map[string]string{"X-RahalGo-Client": "android-customer"})
			return in
		}},
		Actor{Name: "سحبُ الدور", Do: func(context.Context) any {
			return hh.Call("DELETE", "/api/v1/admin/users/"+u.ID+"/roles/"+privRole,
				admin.Token, nil, nil)
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	t.Logf("C3: الدخولُ ردّ %d — %s", in.Code, r)
	if in.Code < 400 {
		got := hh.GET(adminPath, tokenField(in, "access_token"))
		t.Logf("C3: رمزُ الدخول على بابِ الإدارة ⇒ %d", got.Code)
		if got.Code == http.StatusOK {
			t.Errorf("**دخولٌ سبق السحبَ أصدر دوراً مسحوباً** (%d)", got.Code)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **A7+A8 · ولا خبيئةَ تُحيي صلاحيّةً سُحبت**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا خبيئةَ للأدوار أصلاً**: `Redis` تحمل مفتاحَ إبطالِ جلسةٍ
// (`sess:revoked:`) وخبيئةَ حالِ حساب (`ustatus:`) — **ولا أدوار.**
//
// **فالمقيسُ أنّ فقدَها أو تلوّثَها لا يُعيد ما سُحب.**
func TestR15_A7A8_CacheCannotResurrectPrivilege(t *testing.T) {
	hh := New(t)
	u, tok, sid := staffToken(t, hh, privRole, "customer")
	if hh.GET(adminPath, tok).Code != http.StatusOK {
		t.Fatal("**التركيبةُ خطأ**: البابُ مغلقٌ قبل السحب")
	}
	if got := revokeRole(t, hh, u.ID, privRole); got.Code >= 400 {
		t.Fatalf("سحبُ الدور: %s", got)
	}

	// ── A7 · خبيئةٌ باردة: يُمحى كلُّ ما يخصُّ الحساب ─────────────
	ctx := context.Background()
	hh.Redis().Del(ctx, "sess:revoked:"+sid, "ustatus:"+u.ID)
	cold := hh.GET(adminPath, tok)
	t.Logf("A7: خبيئةٌ ممحوّة ⇒ %d", cold.Code)
	if cold.Code == http.StatusOK {
		t.Errorf("**فقدُ الخبيئة أعاد صلاحيّةً سُحبت** (%d)", cold.Code)
	}

	// ── A8 · وخبيئةٌ ملوَّثةٌ تقول «فعّال» ─────────────────────────
	//
	// **وهي أسوأُ من غيابها**: **جوابٌ خاطئٌ يُصدَّق.**
	hh.Redis().Set(ctx, "ustatus:"+u.ID, "active", 30*time.Second)
	warm := hh.GET(adminPath, tok)
	t.Logf("A8: خبيئةٌ دافئةٌ تقول «فعّال» ⇒ %d", warm.Code)
	if warm.Code == http.StatusOK {
		t.Errorf("**خبيئةُ الحال صارت مصدرَ تخويل** (%d) — "+
			"**والأدوارُ من القاعدة وحدَها.**", warm.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **حارسٌ بنيويّ: التخويلُ لا يعود إلى ادّعاءات الرمز**
// ══════════════════════════════════════════════════════════════════════
func TestR15_AuthorizationReadsAuthoritativeRoles(t *testing.T) {
	root := r16Root(t)

	mw := mustRead(t, filepath.Join(root, "backend/internal/server/middleware.go"))
	if strings.Contains(mw, "ctxRoles, claims.Roles)") {
		t.Error("**`ctxRoles` عادت تُملأ من ادّعاءات الرمز** — " +
			"**فدورٌ سُحب يبقى نافذاً ربعَ ساعة.** (`R15`)")
	}
	if !strings.Contains(mw, "roles = dbRoles") {
		t.Error("**الوسيطُ لا يأخذ أدوارَ اللحظة من الحقيقة الموثوقة**")
	}

	ws := mustRead(t, filepath.Join(root, "backend/internal/server/ws.go"))
	if strings.Contains(ws, "slices.Contains(claims.Roles,") ||
		strings.Contains(ws, "slices.ContainsFunc(claims.Roles,") {
		t.Error("**مصافحةُ البثّ تُخوِّل بادّعاءات الرمز** (`R15`)")
	}

	repo := mustRead(t, filepath.Join(root, "backend/internal/identity/repo.go"))
	if !strings.Contains(repo, "FROM user_roles ur WHERE ur.user_id = fam.user_id") {
		t.Error("**الاستعلامُ الموثوقُ لا يحمل الأدوار** — " +
			"**فإمّا رجعت الادّعاءاتُ وإمّا أُضيف استعلامٌ ثانٍ.**")
	}
}
