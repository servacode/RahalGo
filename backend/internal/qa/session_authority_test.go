package qa

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **غيابُ خبرٍ ليس خبراً بالسلامة** — `R16`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **`SessionRevoked` تردّ `bool`**: `err == nil && n > 0`. **فخطأُ
// `Redis` يُقرأ «ليست مُبطَلة»** — **وجلسةٌ أُبطلت تعود تعمل بسقوط
// خبيئة.**
//
// # وما هو أخطرُ — ولا يُصلحه الرجوعُ إلى القاعدة عند الخطأ وحدَه
//
// **الإبطالُ يكتب القاعدةَ أوّلاً ثمّ `Redis`**، **وكتابةُ `Redis`
// تسقط صامتةً أثناء العطل.** **فتعود `Redis` وفيها غيابٌ لجلسةٍ
// أُبطلت فعلاً** — **والغيابُ ليس برهانَ سلامة.**
//
// # العقدُ المقيس
//
//	إصابةٌ في Redis   ⇒ رفضٌ فوريّ
//	غيابٌ أو خطأٌ     ⇒ **غيرُ حاسم** ⇒ القاعدةُ تحكم
//	صفٌّ حيٌّ         ⇒ VALID
//	لا صفَّ حيّاً      ⇒ REVOKED
//	القاعدةُ ساقطة    ⇒ **٥٠٣** لا ٤٠١ ولا ٢٠٠

const mePath = "/api/v1/auth/me"

// liveSession **جلسةٌ لها صفٌّ دائمٌ كما يكتبه المحرّك.**
//
// # ولماذا لا يُنادى الدخول
//
// **مصنعُ الفحص لا يضع كلمةَ مرور** — **ومسارُ الدخول يقيس التوثيقَ
// لا الجلسة.**
//
// **والصفُّ هو العقد**: `issueSession` ينادي `StoreRefresh` فيُنشئ
// صفّاً بمعرّفِ جلسة، **ثمّ يُصدر توكناً يحمله.** **فيُصنَع الصفُّ
// بالشكل نفسِه ويُصدَر توكنٌ بمعرّفه** — **والوسيطُ لا يعرف من
// كتبه.**
func liveSession(t *testing.T, h *Harness, roles ...string) (token, sid, userID string) {
	t.Helper()
	if len(roles) == 0 {
		roles = []string{"customer"}
	}
	// **وحسابُ المِسنَد لا المصنع**: **`h.Factory()` يُنشئ مصنعاً
	// جديداً بعدّادٍ من الصفر** — **فنداءان في فحصٍ واحدٍ يولّدان
	// الهاتفَ نفسَه.** (قِيس: `users_phone_key`.)
	u := h.NewUser(roles[0])
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, client)
		VALUES ($1::uuid, $2, now() + interval '30 days', 'web')
		RETURNING session_id::text`,
		u.ID, "qa-"+uniq("h")).Scan(&sid); err != nil {
		t.Fatalf("صفُّ الجلسة: %v", err)
	}
	return h.TokenWithSession(u.ID, sid, roles...), sid, u.ID
}

// revokeInDB يُبطل عائلةَ الجلسة **في الحقيقة الموثوقة وحدَها**.
//
// **ولا يُكتب مفتاحُ `Redis`** — **وهو تمثيلُ «أُبطلت والذاكرةُ ساقطة».**
func revokeInDB(t *testing.T, h *Harness, sid string) {
	t.Helper()
	tag, err := h.Pool.Exec(ctxBG(), `
		UPDATE refresh_tokens SET revoked_at = now()
		 WHERE session_id = $1::uuid AND revoked_at IS NULL`, sid)
	if err != nil {
		t.Fatalf("إبطالٌ في القاعدة: %v", err)
	}
	if tag.RowsAffected() == 0 {
		t.Fatal("**لا صفَّ أُبطل** — التركيبةُ خطأ")
	}
}

// markRevokedInRedis يكتب مُسرِّعَ الرفض كما يكتبه المحرّك.
func markRevokedInRedis(t *testing.T, h *Harness, sid string) {
	t.Helper()
	if err := h.Redis().Set(context.Background(),
		"sess:revoked:"+sid, "1", time.Hour).Err(); err != nil {
		t.Fatalf("كتابةُ الإبطال في الذاكرة: %v", err)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **R1 · جلسةٌ حيّةٌ والذاكرةُ لا تعرفها ⇒ القاعدةُ تُثبتها**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذه الحالُ الغالبة**: **مفتاحُ الإبطال لا يُكتب إلّا عند إبطال** —
// **فكلُّ طلبٍ سليمٍ غيابٌ في الذاكرة.**
func TestR16_R1_ValidSessionRedisMissDBValidates(t *testing.T) {
	h := New(t)
	tok, sid, _ := liveSession(t, h)

	var n int64
	n, _ = h.Redis().Exists(context.Background(), "sess:revoked:"+sid).Result()
	res := h.GET(mePath, tok)
	t.Logf("R1: مفتاحُ الذاكرة=%d · الردّ=%d", n, res.Code)

	if n != 0 {
		t.Fatalf("**التركيبةُ خطأ**: مفتاحُ الإبطال موجودٌ لجلسةٍ حيّة")
	}
	if res.Code != http.StatusOK {
		t.Errorf("**جلسةٌ حيّةٌ رُفضت وغيابُ الذاكرة هو السبب**: %d", res.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **R2 · إصابةٌ في الذاكرة ⇒ رفضٌ فوريّ**
// ══════════════════════════════════════════════════════════════════════
func TestR16_R2_RedisHitDenies(t *testing.T) {
	h := New(t)
	tok, sid, _ := liveSession(t, h)
	markRevokedInRedis(t, h, sid)

	res := h.GET(mePath, tok)
	t.Logf("R2: الردّ=%d", res.Code)
	if res.Code != http.StatusUnauthorized {
		t.Errorf("**إصابةٌ في مُسرِّع الرفض ولم تُرفَض**: %d", res.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **R3 · مُبطَلةٌ في القاعدة والذاكرةُ لا تعرفها ⇒ القاعدةُ تُمسكها**
// ══════════════════════════════════════════════════════════════════════
//
// **وهي الحالُ التي لا يُصلحها «الرجوعُ عند الخطأ» وحدَه**: **الذاكرةُ
// سليمةٌ وتردّ صفراً**، **والجلسةُ مُبطَلةٌ فعلاً.** (تقع حين تسقط
// كتابةُ `Redis` أثناء عطل، ثمّ تعود.)
func TestR16_R3_RedisHealthyButKeyMissingDBCatches(t *testing.T) {
	h := New(t)
	tok, sid, _ := liveSession(t, h)
	revokeInDB(t, h, sid)

	n, _ := h.Redis().Exists(context.Background(), "sess:revoked:"+sid).Result()
	res := h.GET(mePath, tok)
	t.Logf("R3: مفتاحُ الذاكرة=%d (غائبٌ عمداً) · الردّ=%d", n, res.Code)

	if n != 0 {
		t.Fatal("**التركيبةُ خطأ**: المفتاحُ موجودٌ فلا تُقاس الحالُ المقصودة")
	}
	if res.Code != http.StatusUnauthorized {
		t.Errorf("**جلسةٌ مُبطَلةٌ في الحقيقة الموثوقة مرّت** — "+
			"**وغيابُ المفتاح قُرئ سلامةً.** (`R16`) الردّ=%d", res.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **R11 · مسارٌ عامٌّ لا يمسّه شيءٌ من هذا**
// ══════════════════════════════════════════════════════════════════════
func TestR16_R11_PublicRouteUnaffected(t *testing.T) {
	h := New(t)
	res := h.GET("/healthz", "")
	t.Logf("R11: `/healthz` بلا توكن = %d", res.Code)
	if res.Code >= 400 {
		t.Errorf("**مسارٌ عامٌّ صار يشترط توثيقاً**: %d", res.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **R12+R13 · استثناءُ الإيقاف لا يتجاوز الإبطال**
// ══════════════════════════════════════════════════════════════════════
//
// **دورةُ ١١ فتحت باباً ضيّقاً**: موقوفٌ عاديٌّ يُتمّ طلباً حيّاً.
// **وسؤالُ هذه**: **أيصير البابُ التفافاً على الإبطال؟**
//
// # وقياسٌ كشف تعارضاً بين عقدين — يُعرَض ولا يُحسَم هنا
//
// **`AdminUpdateUser` تُبطل كلَّ التوكنات عند أيّ حالٍ غيرِ `active`**
// (`admin.go:159` — `RevokeAllTokens`) — **في القاعدة وحدَها، ولا
// تكتب مفتاحَ `Redis`.**
//
// **فقبل هذا الإصلاح لم يكن لذلك أثر**: الوسيطُ يسأل `Redis` وحدَها
// **فلا يرى الإبطال** — **وبابُ دورةِ ١١ يعمل بالمصادفة.**
//
// **وبعده صارت القاعدةُ هي الحقيقة** — **فالإيقافُ نفسُه يُبطل
// الجلسة، والموقوفُ يُرفَض بـ٤٠١ قبل أن يبلغ استثناءَه.**
//
// **وعقدُ دورةِ ١١ يقول**: «الإيقافُ العاديُّ يمنع نشاطاً جديداً
// **ولا يترك طلباً حيّاً معلَّقاً**»، **و`blocked` بابٌ آخر.**
//
// **فالحسمُ للمالك**: أيقتصر إبطالُ التوكنات على `blocked`؟
// **ولا أُضعّف الإبطالَ من تلقائي.** (`XG-39`)
func TestR16_R12R13_SuspensionExceptionDoesNotBypassRevocation(t *testing.T) {
	h := New(t)
	oid, drv := activeOrderFor(t, h)

	var sid string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, client)
		VALUES ($1::uuid, $2, now() + interval '30 days', 'driver')
		RETURNING session_id::text`,
		drv.ID, "qa-"+uniq("h")).Scan(&sid); err != nil {
		t.Fatalf("صفُّ الجلسة: %v", err)
	}
	tok := h.TokenWithSession(drv.ID, sid, "driver")
	path := "/api/v1/driver/orders/" + oid + "/transition"

	// ── ولا إبطالَ بعدُ: البابُ الضيّقُ مفتوح ────────────────────
	suspend(t, h, drv.ID, "suspended")

	var live int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM refresh_tokens
		 WHERE session_id = $1::uuid AND revoked_at IS NULL
		   AND expires_at > now()`, sid).Scan(&live); err != nil {
		t.Fatalf("صفوفُ الجلسة: %v", err)
	}
	got := h.POST(path, tok, map[string]any{"to": "at_pickup"})
	t.Logf("R12: بعد الإيقاف — صفوفٌ حيّةٌ=%d · الانتقالُ ⇒ %d", live, got.Code)

	if live == 0 {
		t.Logf("**تعارضٌ مقيس** (`XG-39`): **الإيقافُ نفسُه أبطل الجلسة** " +
			"(`AdminUpdateUser` → `RevokeAllTokens`) — **فبابُ دورةِ ١١ " +
			"لا يُبلَغ.** **وكان يعمل لأنّ الوسيطَ يسأل `Redis` وحدَها " +
			"ولا أحدَ يكتب المفتاحَ في هذا المسار.**")
	}

	// ── R13 · وجلسةٌ مُبطَلةٌ تبقى مرفوضةً مهما كان لصاحبها طلب ──
	revokeAnySession(t, h, sid)
	after := h.POST(path, tok, map[string]any{"to": "picked_up"})
	t.Logf("R13: جلسةٌ مُبطَلةٌ وصاحبُها يحمل طلباً حيّاً ⇒ %d", after.Code)

	if after.Code == http.StatusOK {
		t.Errorf("**استثناءُ الإيقاف صار التفافاً على الإبطال** — " +
			"**ورمزٌ مُبطَلٌ يعمل لأنّ صاحبَه يحمل طلباً حيّاً.** (`R16`)")
	}
}

// revokeAnySession يُبطل ما بقي حيّاً — **ولا يشترط أن يجد شيئاً.**
func revokeAnySession(t *testing.T, h *Harness, sid string) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE refresh_tokens SET revoked_at = now()
		 WHERE session_id = $1::uuid AND revoked_at IS NULL`, sid); err != nil {
		t.Fatalf("إبطالٌ في القاعدة: %v", err)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **R14 · مصافحةُ بثٍّ جديدةٌ تتبع العقدَ نفسَه**
// ══════════════════════════════════════════════════════════════════════
func TestR16_R14_NewWSHandshakeFollowsAuthority(t *testing.T) {
	h := New(t)
	tok, sid, _ := liveSession(t, h)

	// **مُبطَلةٌ في القاعدة والذاكرةُ لا تعرفها** — الحالُ الأصعب.
	revokeInDB(t, h, sid)

	res := h.GET("/api/v1/ws?token="+tok, "")
	t.Logf("R14: مصافحةٌ بجلسةٍ مُبطَلةٍ والمفتاحُ غائب ⇒ %d", res.Code)
	if res.Code == http.StatusOK || res.Code == http.StatusSwitchingProtocols {
		t.Errorf("**قناةُ بثٍّ فُتحت لجلسةٍ مُبطَلة**: %d", res.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **حارسٌ بنيويٌّ: لا خطأَ يُقرأ سلامةً**
// ══════════════════════════════════════════════════════════════════════
//
// **والعيبُ لم يكن في القيمة بل في الشكل**: **`bool` لا مكانَ فيه
// لـ«لا أعرف»** — **فمن كتب `err == nil && n > 0` لم يخالف نوعاً.**
//
// **فيُمنَع الشكلُ لا الحالة.**
func TestR16_NoFailOpenSessionCheck(t *testing.T) {
	root := r16Root(t)

	// ── ١ · الدالّةُ القديمةُ لا تعود ─────────────────────────────
	svc := mustRead(t, filepath.Join(root,
		"backend/internal/identity/service.go"))
	if regexp.MustCompile(`func \(s \*Service\) SessionRevoked\(`).MatchString(svc) {
		t.Error("**عادت `SessionRevoked` التي تردّ `bool`** — " +
			"**والشكلُ نفسُه هو ما أوقع `R16`.**")
	}

	// ── ٢ · ولا يُبتلَع خطأُ التحقّق ──────────────────────────────
	//
	// **النمطُ الممنوع**: نداءُ `CheckSession` ثمّ تجاهلُ خطئه
	// بـ`_`. **فمن كتبها أعاد السقوطَ المفتوح بحرفٍ واحد.**
	banned := regexp.MustCompile(`(?:state|st)?,\s*_\s*(?::)?=\s*[\w.]*CheckSession\(`)
	for _, rel := range []string{
		"backend/internal/server/middleware.go",
		"backend/internal/server/ws.go",
	} {
		src := mustRead(t, filepath.Join(root, rel))
		if banned.MatchString(src) {
			t.Errorf("**`%s` يبتلع خطأَ التحقّق** — "+
				"**و«لا أعرف» تصير «سليمة».** (`R16`)", rel)
		}
		if !strings.Contains(src, "errAuthUnavailable") {
			t.Errorf("**`%s` لا يميّز تعذّرَ التحقّق** — "+
				"**و٤٠١ على جلسةٍ سليمةٍ كذبٌ يُخرج صاحبَها.**", rel)
		}
	}

	// ── ٣ · والحقيقةُ الموثوقةُ تُسأل فعلاً ───────────────────────
	chk := mustRead(t, filepath.Join(root,
		"backend/internal/identity/session_check.go"))
	if !strings.Contains(chk, "sessionStateFromDB") {
		t.Error("**لا رجوعَ إلى الحقيقة الموثوقة**")
	}
	if !regexp.MustCompile(`err == nil && n > 0`).MatchString(chk) {
		t.Log("**تنبيه**: تبدّل شكلُ مُسرِّع الرفض — يُعاد القياس")
	}
	// **والإصابةُ وحدَها حاسمة** — والغيابُ يمضي إلى القاعدة.
	if strings.Contains(chk, "return SessionValid, nil\n\t}\n\n\treturn s.sessionStateFromDB") {
		t.Error("**غيابُ المفتاح يُقرأ سلامةً** — **وذاك `R16` عائداً.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **وكلُّ توكنٍ يُصدره المحرّكُ يحمل معرّفَ جلسةٍ له صفٌّ دائم**
// ══════════════════════════════════════════════════════════════════════
//
// **وإلّا صار «لا صفَّ ⇒ رفضٌ» إغلاقاً على صنفٍ مشروع.**
func TestR16_IssuedTokensAlwaysCarrySession(t *testing.T) {
	h := New(t)
	_, sid, _ := liveSession(t, h)

	var live, total int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FILTER (WHERE revoked_at IS NULL AND expires_at > now()),
		       count(*) FROM refresh_tokens WHERE session_id = $1::uuid`,
		sid).Scan(&live, &total); err != nil {
		t.Fatalf("صفوفُ الجلسة: %v", err)
	}
	t.Logf("توكنُ دخولٍ: معرّفٌ=%s… · صفوفٌ حيّة=%d من %d", sid[:8], live, total)
	if live == 0 {
		t.Errorf("**توكنٌ صدر بلا صفٍّ حيٍّ يُثبته** — "+
			"**فالعقدُ الجديدُ يُغلق على صنفٍ مشروع.** (حيٌّ=%d · كلٌّ=%d)",
			live, total)
	}

	// **والمصدرُ يشهد**: `StoreRefresh` قبل `IssueAccess`.
	src := mustRead(t, filepath.Join(r16Root(t),
		"backend/internal/identity/service.go"))
	store := strings.Index(src, "s.repo.StoreRefresh(ctx")
	issue := strings.Index(src, "s.tokens.IssueAccess(user.ID")
	if store < 0 || issue < 0 || store > issue {
		t.Error("**ترتيبُ الإصدار تبدّل** — **يُعاد قياسُ صنفِ التوكنات " +
			"قبل الاعتماد على «لكلّ توكنٍ صفٌّ».**")
	}
}

func r16Root(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(wd)))
}

func mustRead(t *testing.T, p string) string {
	t.Helper()
	s, err := readFile(p)
	if err != nil {
		t.Fatalf("قراءةُ %s: %v", p, err)
	}
	return s
}
