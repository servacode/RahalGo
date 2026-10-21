package qa

import "testing"

// afLogin **دخولٌ حقيقيٌّ من تطبيق الزبون** — يردّ رمزَي الوصول والتحديث.
func afLogin(t *testing.T, h *Harness) (userID, access, refresh string) {
	t.Helper()
	phone := aprUser(t, h, "customer")
	got := h.Call("POST", "/api/v1/auth/login", "", map[string]any{
		"phone": phone, "password": aprPassword,
	}, map[string]string{"X-RahalGo-Client": "android-customer", "X-RahalGo-Version": "29"})
	if got.Code != 200 {
		t.Fatalf("الدخول: %s", got)
	}
	tok, _ := got.JSON()["tokens"].(map[string]any)
	user, _ := got.JSON()["user"].(map[string]any)
	access, _ = tok["access_token"].(string)
	refresh, _ = tok["refresh_token"].(string)
	userID, _ = user["id"].(string)
	if access == "" || refresh == "" || userID == "" {
		t.Fatalf("رموزُ الدخول ناقصة: %s", got)
	}
	return
}

func afRows(t *testing.T, h *Harness, userID, kind string) (n int, client, version, detail string) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*)::int, COALESCE(max(client), ''), COALESCE(max(app_version), ''), COALESCE(max(detail), '')
		  FROM auth_failures WHERE user_id = $1::uuid AND kind = $2`, userID, kind).
		Scan(&n, &client, &version, &detail); err != nil {
		t.Fatal(err)
	}
	return
}

// TestAuthFailures_RefreshAndRequestAreRecorded **«انتهت جلستك» يُسجَّل بسببه**
// (قرارُ المالك ٢٠٢٦-١٠-٢١): تجديدٌ فشل ونداءٌ رُفض لجلسةٍ انتهت — كلاهما صفٌّ
// بالحساب والتطبيق ونسخته والسبب.
func TestAuthFailures_RefreshAndRequestAreRecorded(t *testing.T) {
	h := New(t)
	uid, access, refresh := afLogin(t, h)
	hdr := map[string]string{"X-RahalGo-Client": "android-customer", "X-RahalGo-Version": "29"}
	if out := h.Call("POST", "/api/v1/auth/logout", "", map[string]any{"refresh_token": refresh}, hdr); out.Code != 200 {
		t.Fatalf("الخروج: %s", out)
	}

	if got := h.Call("POST", "/api/v1/auth/refresh", "", map[string]any{"refresh_token": refresh}, hdr); got.Code != 401 {
		t.Fatalf("تجديدٌ بعد الخروج يجب أن يُرفض: %s", got)
	}
	n, client, version, detail := afRows(t, h, uid, "refresh")
	if n != 1 || client != "android-customer" || version != "29" || detail != "logout" {
		t.Fatalf("صفُّ التجديد: عدد=%d تطبيق=%q نسخة=%q سبب=%q", n, client, version, detail)
	}

	if got := h.Call("GET", "/api/v1/auth/me", access, nil, hdr); got.Code != 401 {
		t.Fatalf("نداءٌ بجلسةٍ منتهية يجب أن يُرفض: %s", got)
	}
	if n, _, _, _ := afRows(t, h, uid, "request"); n != 1 {
		t.Fatalf("صفُّ النداء المرفوض = %d — والمتوقَّع ١", n)
	}
}

// TestAuthFailures_MidRotationIsNotLogout **منتصفُ التدوير لا يُخرج أحداً**: رمزٌ
// دُوِّر قبل لحظةٍ ولم يُكتب خلفُه بعد — الجلسةُ حيّة. **وبعد ثلاثين ثانيةً، أو إن
// أنهاها سببٌ آخر، تنتهي.**
func TestAuthFailures_MidRotationIsNotLogout(t *testing.T) {
	h := New(t)
	uid, access, _ := afLogin(t, h)
	set := func(sql string) {
		if _, err := h.Pool.Exec(ctxBG(), sql, uid); err != nil {
			t.Fatal(err)
		}
	}
	set(`UPDATE refresh_tokens SET revoked_at = now(), revoked_reason = 'rotated'
	      WHERE user_id = $1::uuid AND revoked_at IS NULL`)
	if got := h.GET("/api/v1/auth/me", access); got.Code != 200 {
		t.Fatalf("**جلسةٌ في منتصف التدوير رُفضت** — وهذا ما يُخرج الحساب: %s", got)
	}
	set(`UPDATE refresh_tokens SET revoked_at = now() - interval '1 minute' WHERE user_id = $1::uuid`)
	if got := h.GET("/api/v1/auth/me", access); got.Code != 401 {
		t.Fatalf("تدويرٌ قديمٌ بلا خلفٍ يجب أن يُنهي الجلسة: %s", got)
	}
	set(`UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1::uuid`)
	// **صفٌّ واحدٌ في كلّ جلسة** — للحساب جلستان (الإنشاءُ والدخول).
	set(`UPDATE refresh_tokens SET revoked_reason = 'logout'
	      WHERE id IN (SELECT DISTINCT ON (session_id) id FROM refresh_tokens
	                    WHERE user_id = $1::uuid ORDER BY session_id, id)`)
	if got := h.GET("/api/v1/auth/me", access); got.Code != 401 {
		t.Fatalf("جلسةٌ أنهاها خروجٌ لا تُعدّ في منتصف التدوير: %s", got)
	}
}
