package qa

import (
	"context"
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **الإيقافُ حالُ عملٍ لا حالُ أمن** — `XG-39`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **`AdminUpdateUser` تُبطل كلَّ التوكنات عند أيّ حالٍ غيرِ `active`.**
//
// **ولم يكن لذلك أثرٌ يُرى**: تكتب القاعدةَ ولا تكتب مفتاحَ `Redis`،
// **والوسيطُ يسأل `Redis` وحدَها** — **فبابُ دورةِ ١١ يعمل بالمصادفة.**
//
// **ثمّ صارت القاعدةُ هي الحقيقة** (`R16`) **فظهر التعارض**: الموقوفُ
// يُرَدُّ بـ٤٠١ قبل أن يبلغ استثناءَه، **والطلبُ الحيُّ يبقى معلَّقاً
// بمن لا يقدر.**
//
// # وعقدُ المالك
//
//	suspended  توثيقٌ يبقى · وتخويلٌ ضيّقٌ على الطلب الحيّ وحدَه
//	blocked    إبطالٌ شاملٌ فوريّ · ولا استثناء
//
// **والتوثيقُ غيرُ التخويل** — **فليست الجلسةُ الباقيةُ إذناً.**

// deliverySession جلسةٌ دائمةٌ لفاعلٍ بعينه بدورٍ محدَّد.
func deliverySession(t *testing.T, h *Harness, userID, client, role string) (token, sid string) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, client)
		VALUES ($1::uuid, $2, now() + interval '30 days', $3)
		RETURNING session_id::text`,
		userID, "qa-"+uniq("h"), client).Scan(&sid); err != nil {
		t.Fatalf("صفُّ الجلسة: %v", err)
	}
	return h.TokenWithSession(userID, sid, role), sid
}

// liveRows صفوفُ عائلةِ جلسةٍ الحيّة.
func liveRows(t *testing.T, h *Harness, sid string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM refresh_tokens
		 WHERE session_id = $1::uuid AND revoked_at IS NULL
		   AND expires_at > now()`, sid).Scan(&n); err != nil {
		t.Fatalf("صفوفُ الجلسة: %v", err)
	}
	return n
}

// ══════════════════════════════════════════════════════════════════════
// **S1+S3+S4 · الإيقافُ لا يُبطل · والاستمرارُ ضيّقٌ كما هو**
// ══════════════════════════════════════════════════════════════════════
func TestXG39_S1S3S4_SuspensionKeepsSessionAndNarrowScope(t *testing.T) {
	h := New(t)
	oid, drv := activeOrderFor(t, h)
	tok, sid := deliverySession(t, h, drv.ID, "driver", "driver")

	suspend(t, h, drv.ID, "suspended")

	// ── S1 · الجلسةُ لم تُبطَل ────────────────────────────────────
	live := liveRows(t, h, sid)
	t.Logf("S1: بعد الإيقاف — صفوفٌ حيّةٌ للجلسة = %d", live)
	if live == 0 {
		t.Fatalf("**الإيقافُ العاديُّ أبطل الجلسة** — **وذاك `XG-39`.**")
	}

	// ── S3 · وفعلُ الاستمرار يمرّ ─────────────────────────────────
	ok := h.POST("/api/v1/driver/orders/"+oid+"/transition", tok,
		map[string]any{"to": "at_pickup"})
	t.Logf("S3: انتقالُ الطلب الحيّ ⇒ %d", ok.Code)
	if ok.Code >= 400 {
		t.Errorf("**الموقوفُ لا يستطيع إتمامَ طلبِه** (%d) — "+
			"**والطلبُ يبقى معلَّقاً بمن لا يقدر.**", ok.Code)
	}

	// ── S4 · ولا يمتدُّ الإذنُ إلى غيره ───────────────────────────
	other := h.GET("/api/v1/driver/queue", tok)
	t.Logf("S4: طابورُ عملٍ جديد ⇒ %d", other.Code)
	if other.Code < 400 {
		t.Errorf("**جلسةُ الموقوف صارت إذناً عامّاً** (%d) — "+
			"**والتوثيقُ غيرُ التخويل.**", other.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S5 · انتهاءُ رمزِ الوصول لا يقتل الاستمرار**
// ══════════════════════════════════════════════════════════════════════
//
// **ومهلتُه خمسَ عشرةَ دقيقة** — **ورحلةٌ قد تطول.**
//
// **والتجديدُ يُبقي الجلسةَ نفسَها** — **ولا يُنشئ عائلةً ثانية.**
func TestXG39_S5_SuspendedRefreshPreservesSameSession(t *testing.T) {
	h := New(t)
	oid, drv := activeOrderFor(t, h)

	// **ورمزُ تجديدٍ حقيقيٌّ يُصدره المحرّك** — **ولا يُصطنَع**:
	// **العقدُ أن يُقبَل هذا الرمزُ بعينه.**
	raw, _ := issuedRefresh(t, h, drv.ID, "driver")
	suspend(t, h, drv.ID, "suspended")

	got := h.POST("/api/v1/auth/refresh", "", map[string]any{"refresh_token": raw})
	t.Logf("S5: تجديدُ جلسةٍ قائمةٍ لموقوف ⇒ %d", got.Code)
	if got.Code >= 400 {
		t.Fatalf("**التجديدُ رُفض** (%s) — **فاستثناءُ دورةِ ١١ يموت "+
			"بعد ربع ساعةٍ ولو لم يُبطَل شيء.**", got)
	}

	fresh := tokenField(got, "access_token")
	if fresh == "" {
		t.Fatal("**لا رمزَ وصولٍ في ردّ التجديد**")
	}

	// ── والجلسةُ نفسُها لا عائلةٌ ثانية ───────────────────────────
	var families int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(DISTINCT session_id) FROM refresh_tokens
		 WHERE user_id = $1::uuid`, drv.ID).Scan(&families); err != nil {
		t.Fatalf("عائلاتُ الجلسة: %v", err)
	}
	t.Logf("S5: عائلاتُ جلسةٍ للحساب = %d (والمرتقَبُ ١)", families)
	if families != 1 {
		t.Errorf("**التجديدُ أنشأ عائلةً ثانية**: %d", families)
	}

	// ── والاستمرارُ ما زال ممكناً بالرمز الجديد ───────────────────
	cont := h.POST("/api/v1/driver/orders/"+oid+"/transition", fresh,
		map[string]any{"to": "at_pickup"})
	t.Logf("S5: الاستمرارُ بالرمز المجدَّد ⇒ %d", cont.Code)
	if cont.Code >= 400 {
		t.Errorf("**الاستمرارُ سقط بعد التجديد** (%d)", cont.Code)
	}
}

// issuedRefresh **رمزُ تجديدٍ من المحرّك** — عبر مسار الدخول بكلمةٍ
// يضعها المسارُ الإداريُّ نفسُه.
//
// **ولا يُصطنَع الرمزُ**: `RevokeRefresh` تبحث عن بصمته، **فمصنوعٌ
// لا يُقبَل، ولا يُقاس به عقدُ التجديد.**
func issuedRefresh(t *testing.T, h *Harness, userID, client string) (raw, sid string) {
	t.Helper()
	admin := h.NewUser("admin")
	const pw = "Qa!Refresh-2026"
	set := h.POST("/api/v1/admin/users/"+userID+"/password", admin.Token,
		map[string]any{"password": pw})
	if set.Code >= 400 {
		t.Skipf("لا مسارَ لوضع كلمةٍ بهذا الشكل: %s", set)
	}
	var phone string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT phone FROM users WHERE id = $1::uuid`, userID).Scan(&phone); err != nil {
		t.Fatalf("هاتفُ الحساب: %v", err)
	}
	in := h.POST("/api/v1/auth/login", "", map[string]any{
		"phone": phone, "password": pw, "client": client,
	})
	if in.Code >= 400 {
		t.Skipf("تعذّر الدخول: %s", in)
	}
	// **والتوكناتُ متداخلةٌ تحت `tokens`** — لا في جذر الردّ.
	raw = tokenField(in, "refresh_token")
	if raw == "" {
		t.Skipf("لا رمزَ تجديدٍ في ردّ الدخول: %s", in)
	}
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT session_id::text FROM refresh_tokens
		 WHERE user_id = $1::uuid AND revoked_at IS NULL
		 ORDER BY created_at DESC LIMIT 1`, userID).Scan(&sid); err != nil {
		t.Fatalf("معرّفُ الجلسة: %v", err)
	}
	return raw, sid
}

// ══════════════════════════════════════════════════════════════════════
// **S6+S7 · ولا دخولَ جديدٌ ولا جهازٌ جديدٌ لموقوف**
// ══════════════════════════════════════════════════════════════════════
func TestXG39_S6S7_SuspendedCannotOpenNewSession(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)
	issuedRefresh(t, h, drv.ID, "driver")

	suspend(t, h, drv.ID, "suspended")

	var phone string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT phone FROM users WHERE id = $1::uuid`, drv.ID).Scan(&phone); err != nil {
		t.Fatalf("هاتفُ الحساب: %v", err)
	}
	got := h.POST("/api/v1/auth/login", "", map[string]any{
		"phone": phone, "password": "Qa!Refresh-2026", "client": "driver",
	})
	t.Logf("S6/S7: دخولٌ جديدٌ لموقوف ⇒ %d", got.Code)
	if got.Code < 400 {
		t.Errorf("**موقوفٌ فتح جلسةً جديدة** (%d) — "+
			"**والإيقافُ يمنع نشاطاً جديداً.**", got.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S8 · وجلسةٌ أُبطلت لا يُنجيها الإيقاف**
// ══════════════════════════════════════════════════════════════════════
func TestXG39_S8_RevokedSuspendedSessionStaysDenied(t *testing.T) {
	h := New(t)
	oid, drv := activeOrderFor(t, h)
	tok, sid := deliverySession(t, h, drv.ID, "driver", "driver")
	suspend(t, h, drv.ID, "suspended")

	revokeAnySession(t, h, sid)
	got := h.POST("/api/v1/driver/orders/"+oid+"/transition", tok,
		map[string]any{"to": "at_pickup"})
	t.Logf("S8: موقوفٌ بجلسةٍ مُبطَلة ⇒ %d", got.Code)
	if got.Code != http.StatusUnauthorized {
		t.Errorf("**جلسةٌ مُبطَلةٌ عملت باستثناء الإيقاف** (%d) — "+
			"**وعقدُ `R16` يسبق.**", got.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S9+S10 · والحظرُ يقطع كلَّ شيء**
// ══════════════════════════════════════════════════════════════════════
func TestXG39_S9S10_BlockRevokesEverything(t *testing.T) {
	h := New(t)
	oid, drv := activeOrderFor(t, h)
	tok, sid := deliverySession(t, h, drv.ID, "driver", "driver")

	suspend(t, h, drv.ID, "suspended")
	if liveRows(t, h, sid) == 0 {
		t.Fatal("**الإيقافُ أبطل الجلسة** — التركيبةُ خطأ")
	}

	suspend(t, h, drv.ID, "blocked")
	live := liveRows(t, h, sid)
	got := h.POST("/api/v1/driver/orders/"+oid+"/transition", tok,
		map[string]any{"to": "at_pickup"})
	t.Logf("S9/S10: بعد الحظر — صفوفٌ حيّةٌ=%d · الانتقالُ ⇒ %d", live, got.Code)

	if live != 0 {
		t.Errorf("**الحظرُ لم يُبطل الجلسات**: %d", live)
	}
	if got.Code < 400 {
		t.Errorf("**محظورٌ ما زال يعمل** (%d) — **ولا استثناءَ للحظر.**",
			got.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S11+S12 · والعودةُ إلى `active` لا تبعث ما أُبطل**
// ══════════════════════════════════════════════════════════════════════
func TestXG39_S11S12_ReactivationDoesNotResurrect(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)
	blockedTok, blockedSid := deliverySession(t, h, drv.ID, "driver", "driver")

	suspend(t, h, drv.ID, "blocked")
	suspend(t, h, drv.ID, "active")

	live := liveRows(t, h, blockedSid)
	res := h.GET(mePath, blockedTok)
	t.Logf("S11: حُظر ثمّ أُعيد — صفوفٌ حيّةٌ=%d · `me` ⇒ %d", live, res.Code)
	if res.Code == http.StatusOK {
		t.Errorf("**جلسةٌ أُبطلت بالحظر بُعثت بالتفعيل** (%d)", res.Code)
	}

	// ── S12 · وجلسةُ الموقوف تعود طبيعيّةً ────────────────────────
	tok2, sid2 := deliverySession(t, h, drv.ID, "driver", "driver")
	suspend(t, h, drv.ID, "suspended")
	suspend(t, h, drv.ID, "active")
	t.Logf("S12: أُوقف ثمّ أُعيد — صفوفٌ حيّةٌ=%d · `me` ⇒ %d",
		liveRows(t, h, sid2), h.GET(mePath, tok2).Code)
	if h.GET(mePath, tok2).Code != http.StatusOK {
		t.Errorf("**جلسةٌ سليمةٌ لم تعد طبيعيّةً بعد رفع الإيقاف**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **C1 · إيقافٌ يتزامن مع انتقالِ طلب**
// ══════════════════════════════════════════════════════════════════════
func TestXG39_C1_SuspendVsTransition(t *testing.T) {
	h := New(t)
	oid, drv := activeOrderFor(t, h)
	tok, sid := deliverySession(t, h, drv.ID, "driver", "driver")
	admin := h.NewUser("admin")

	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "انتقالُ الطلب", Do: func(context.Context) any {
			return h.POST("/api/v1/driver/orders/"+oid+"/transition", tok,
				map[string]any{"to": "at_pickup"})
		}},
		Actor{Name: "إيقافُ السائق", Do: func(context.Context) any {
			return h.PATCH("/api/v1/admin/users/"+drv.ID, admin.Token,
				map[string]any{"status": "suspended", "status_reason": "XG-39"})
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}

	var status string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT status FROM orders WHERE id = $1::uuid`, oid).Scan(&status); err != nil {
		t.Fatalf("حالُ الطلب: %v", err)
	}
	t.Logf("C1: حالُ الطلب=%q · صفوفٌ حيّةٌ=%d — %s", status, liveRows(t, h, sid), r)

	if status != "dispatching" && status != "at_pickup" && status != "assigned" {
		t.Errorf("**حالُ الطلب مكسورة**: %q", status)
	}
	if liveRows(t, h, sid) == 0 {
		t.Errorf("**الإيقافُ المتزامنُ أبطل الجلسة**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **C2 · حظرٌ يتزامن مع تجديد — ولا رمزَ يُفلت بعده**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذا حرِجٌ أمنيّاً**: **متى ثبت الحظرُ لم يجز أن يخرج بعده رمزُ
// وصولٍ صالح.**
//
// **والحكمُ بترتيب القاعدة لا بالتوقيت**: **يُقرأ ما ثبت ثمّ يُقاس
// أثرُه** — **لا «أيُّهما سبق في الساعة».**
func TestXG39_C2_BlockVsRefresh(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)
	raw, sid := issuedRefresh(t, h, drv.ID, "driver")
	admin := h.NewUser("admin")

	var refreshed Res
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "تجديدُ الرمز", Do: func(context.Context) any {
			refreshed = h.POST("/api/v1/auth/refresh", "",
				map[string]any{"refresh_token": raw})
			return refreshed
		}},
		Actor{Name: "حظرُ الحساب", Do: func(context.Context) any {
			return h.PATCH("/api/v1/admin/users/"+drv.ID, admin.Token,
				map[string]any{"status": "blocked", "status_reason": "XG-39"})
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}

	live := liveRows(t, h, sid)
	t.Logf("C2: التجديدُ ردّ %d · صفوفٌ حيّةٌ بعد الحظر=%d — %s",
		refreshed.Code, live, r)

	// **والحكمُ على ما ثبت**: **الحظرُ نافذٌ ⇒ لا رمزَ يعمل بعده.**
	if live != 0 {
		t.Errorf("**الحظرُ ثبت وبقيت صفوفٌ حيّة**: %d — "+
			"**ورمزٌ يُفلت بعد الحظر خرقٌ أمنيّ.**", live)
	}
	if refreshed.Code < 400 {
		tok := tokenField(refreshed, "access_token")
		after := h.GET(mePath, tok)
		t.Logf("C2: الرمزُ الذي خرج من السباق ⇒ %d", after.Code)
		if after.Code == http.StatusOK {
			t.Errorf("**رمزُ وصولٍ أفلت من الحظر وما زال يعمل** — "+
				"**والحقيقةُ الموثوقةُ تقول محظور.** (%d)", after.Code)
		}
	}
}

// tokenField يقرأ توكناً من `tokens` في الردّ.
func tokenField(res Res, name string) string {
	toks, _ := res.JSON()["tokens"].(map[string]any)
	v, _ := toks[name].(string)
	return v
}

// ══════════════════════════════════════════════════════════════════════
// **S2 · موقوفٌ بلا طلبٍ حيّ — جلسةٌ قائمةٌ وتخويلٌ مغلق**
// ══════════════════════════════════════════════════════════════════════
//
// **والفرقُ بينه وبين `S1` أنّ لا استثناءَ له أصلاً** — **فيُقاس أنّ
// بقاءَ الجلسة ليس إذناً.**
func TestXG39_S2_SuspendedWithoutOrderKeepsSessionButNoActivity(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	tok, sid := deliverySession(t, h, drv.ID, "driver", "driver")

	suspend(t, h, drv.ID, "suspended")
	live := liveRows(t, h, sid)
	res := h.GET("/api/v1/driver/queue", tok)
	t.Logf("S2: صفوفٌ حيّةٌ=%d · طابورُ العمل ⇒ %d", live, res.Code)

	if live == 0 {
		t.Errorf("**الإيقافُ العاديُّ أبطل جلسةً** — **حتّى بلا طلبٍ حيّ.**")
	}
	if res.Code < 400 {
		t.Errorf("**موقوفٌ بلا طلبٍ يعمل** (%d) — **والإيقافُ يمنع "+
			"نشاطاً جديداً.**", res.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **والبثُّ يتبع النموذجَ نفسَه**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا تُغلَق `R14`**: **وصلةٌ قائمةٌ عقدٌ آخرُ لم يُقَس هنا.**
func TestXG39_WS_HandshakeFollowsStatusModel(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	tok, sid := deliverySession(t, h, drv.ID, "driver", "driver")

	suspend(t, h, drv.ID, "suspended")
	sus := h.GET("/api/v1/ws?token="+tok, "")
	t.Logf("مصافحةٌ بجلسةِ موقوفٍ سليمة ⇒ %d", sus.Code)

	suspend(t, h, drv.ID, "blocked")
	t.Logf("بعد الحظر — صفوفٌ حيّةٌ=%d", liveRows(t, h, sid))
	blk := h.GET("/api/v1/ws?token="+tok, "")
	t.Logf("مصافحةٌ بجلسةٍ أُبطلت بالحظر ⇒ %d", blk.Code)

	if blk.Code == http.StatusOK || blk.Code == http.StatusSwitchingProtocols {
		t.Errorf("**قناةُ بثٍّ فُتحت لمحظورٍ أُبطلت جلستُه**: %d", blk.Code)
	}
}
