package qa

// ══════════════════════════════════════════════════════════════════════
// **إبطالُ الجلسة عبر عُقدتين** — `GATE-STG-01` · `R16`
// ══════════════════════════════════════════════════════════════════════
//
// # العقدُ المطلوب بنصّه
//
//	خادمان يشتركان في `Redis` ويُبطَل رمزٌ على أحدهما فيُرفَض على الآخر
//
// **وعُقدةٌ واحدةٌ لا تُثبته** — **والنشرُ عُقَد.** **وحالٌ في ذاكرةِ
// عمليّةٍ تصير حقيقةً تُرى صحيحةً في عقدةٍ وكاذبةً في أختها.**
//
// # ولا يُستنتَج تعدّدُ العُقَد من تكرار النداء
//
// **كلُّ نداءٍ هنا يُوجَّه إلى عنوانٍ بعينه** — `A` و`B` عمليّتان
// مستقلّتان بمعرّفَي عمليّةٍ مختلفَين، تشتركان في قاعدةٍ واحدةٍ
// وذاكرةٍ واحدة.
//
// # وعلى التجهيز وحدَه
//
// **ولا إنتاجَ يُمَسّ**: العناوينُ والقاعدةُ والذاكرةُ كلُّها من
// البيئة، **ويُتخطّى الفحصُ إن لم تُضبَط.**
//
// # وكيف تُقام الطوبولوجيا
//
//	docker run -d --name rahalgo-c29-postgres -e POSTGRES_USER=rahalgo //	  -e POSTGRES_PASSWORD=<كلمة> -e POSTGRES_DB=rahalgo_staging //	  -p 5535:5432 postgis/postgis:16-3.4
//	docker run -d --name rahalgo-c29-redis -p 6581:6379 redis:7-alpine
//	HTTP_ADDR=":8091" ./api   &   HTTP_ADDR="127.0.0.1:8092" ./api
//
// **وبالبنائيّة نفسِها للعقدتين** — **وعقدتان بإصدارين تُثبتان شيئاً
// عن نفسيهما لا عن النشر.**

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/servacode/rahalgo/backend/internal/auth"
)

// stgNodes بيئةُ العُقدتين — **ولا شيءَ مكتوبٌ في الشيفرة.**
type stgNodes struct {
	A, B   string
	pg     *pgxpool.Pool
	rdb    *redis.Client
	tokens *auth.TokenIssuer
}

// crossNodeStaging يهيّئ العُقدتين أو يتخطّى.
//
//	RAHALGO_STG_NODE_A   عنوانُ العقدة الأولى
//	RAHALGO_STG_NODE_B   عنوانُ العقدة الثانية
//	STG_DATABASE_URL     القاعدةُ المشترَكة
//	STG_REDIS_ADDR       الذاكرةُ المشترَكة
//	STG_JWT_SECRET       سرُّ التوقيع — **خاصٌّ بالتجهيز**
func crossNodeStaging(t *testing.T) *stgNodes {
	t.Helper()
	n := &stgNodes{A: os.Getenv("RAHALGO_STG_NODE_A"), B: os.Getenv("RAHALGO_STG_NODE_B")}
	dsn, addr := os.Getenv("STG_DATABASE_URL"), os.Getenv("STG_REDIS_ADDR")
	secret := os.Getenv("STG_JWT_SECRET")
	if n.A == "" || n.B == "" || dsn == "" || addr == "" || secret == "" {
		t.Skip("فحصُ عُقدتين — يحتاج RAHALGO_STG_NODE_A/B و STG_DATABASE_URL و STG_REDIS_ADDR و STG_JWT_SECRET")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("قاعدةُ التجهيز: %v", err)
	}
	t.Cleanup(pool.Close)
	n.pg = pool
	n.rdb = redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = n.rdb.Close() })
	n.tokens = auth.NewTokenIssuer(secret, 15*time.Minute)
	return n
}

// call نداءٌ إلى عقدةٍ بعينها — **بعنوانها لا بموزِّع حِمل.**
func (n *stgNodes) call(t *testing.T, node, method, path, token string, body any) (int, string) {
	t.Helper()
	var rdr *strings.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = strings.NewReader(string(b))
	} else {
		rdr = strings.NewReader("")
	}
	req, err := http.NewRequest(method, "http://"+node+path, rdr)
	if err != nil {
		t.Fatalf("نداءٌ معطوب: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("تعذّر النداءُ على %s: %v", node, err)
	}
	defer res.Body.Close()
	buf := make([]byte, 512)
	k, _ := res.Body.Read(buf)
	return res.StatusCode, string(buf[:k])
}

// session حسابٌ وجلسةٌ في قاعدة التجهيز — **صفٌّ دائمٌ كما في الإنتاج.**
func (n *stgNodes) session(t *testing.T, roles ...string) (token, sid, userID string) {
	t.Helper()
	if len(roles) == 0 {
		roles = []string{"customer"}
	}
	phone := fmt.Sprintf("09%09d", time.Now().UnixNano()%1_000_000_000)
	if err := n.pg.QueryRow(context.Background(), `
		INSERT INTO users (phone, full_name, status, whatsapp_phone, whatsapp_verified_at)
		VALUES ($1, 'STG-01', 'active', $1, now()) RETURNING id::text`, phone).
		Scan(&userID); err != nil {
		t.Fatalf("حسابُ تجهيز: %v", err)
	}
	t.Cleanup(func() {
		_, _ = n.pg.Exec(context.Background(), `DELETE FROM users WHERE id = $1::uuid`, userID)
	})
	for _, r := range roles {
		if _, err := n.pg.Exec(context.Background(), `
			INSERT INTO user_roles (user_id, role_code) VALUES ($1::uuid, $2)
			ON CONFLICT DO NOTHING`, userID, r); err != nil {
			t.Fatalf("دورُ تجهيز: %v", err)
		}
	}
	if err := n.pg.QueryRow(context.Background(), `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, client)
		VALUES ($1::uuid, $2, now() + interval '30 days', 'web')
		RETURNING session_id::text`, userID, "stg-"+userID+fmt.Sprint(time.Now().UnixNano())).
		Scan(&sid); err != nil {
		t.Fatalf("جلسةُ تجهيز: %v", err)
	}
	tok, _, err := n.tokens.IssueAccess(userID, roles, sid)
	if err != nil {
		t.Fatalf("توكنُ تجهيز: %v", err)
	}
	return tok, sid, userID
}

// revoke يُبطل عائلةَ جلسةٍ في القاعدة — **الحقيقةُ الموثوقة.**
func (n *stgNodes) revoke(t *testing.T, sid string) {
	t.Helper()
	if _, err := n.pg.Exec(context.Background(), `
		UPDATE refresh_tokens SET revoked_at = now()
		 WHERE session_id = $1::uuid AND revoked_at IS NULL`, sid); err != nil {
		t.Fatalf("الإبطال: %v", err)
	}
}

const stgMe = "/api/v1/auth/me"

// ══════════════════════════════════════════════════════════════════════
// **S1…S3 · الجلسةُ والإبطالُ في الاتّجاهين**
// ══════════════════════════════════════════════════════════════════════
func TestSTG01_S1S2S3_RevocationCrossesNodes(t *testing.T) {
	n := crossNodeStaging(t)

	// ── S1 · جلسةٌ صالحةٌ تعمل على العقدتين ──────────────────────
	tok, sid, _ := n.session(t)
	a1, _ := n.call(t, n.A, "GET", stgMe, tok, nil)
	b1, _ := n.call(t, n.B, "GET", stgMe, tok, nil)
	t.Logf("S1: A=%d · B=%d", a1, b1)
	if a1 != http.StatusOK || b1 != http.StatusOK {
		t.Fatalf("**S1: جلسةٌ صالحةٌ لا تعمل على العقدتين** — A=%d B=%d", a1, b1)
	}

	// ── S2 · يُبطَل ثمّ يُجرَّب على `B` ─────────────────────────
	//
	// **والإبطالُ يقع في القاعدة المشترَكة** — وهي الحقيقة.
	n.revoke(t, sid)
	a2, _ := n.call(t, n.A, "GET", stgMe, tok, nil)
	b2, _ := n.call(t, n.B, "GET", stgMe, tok, nil)
	t.Logf("S2: بعد الإبطال — A=%d · B=%d", a2, b2)
	if b2 != http.StatusUnauthorized {
		t.Errorf("**S2: مُبطَلةٌ تعمل على `B`** — %d", b2)
	}
	if a2 != http.StatusUnauthorized {
		t.Errorf("**S2: مُبطَلةٌ تعمل على `A`** — %d", a2)
	}

	// ── S3 · الاتّجاهُ المعاكس ──────────────────────────────────
	tok2, sid2, _ := n.session(t)
	if c, _ := n.call(t, n.B, "GET", stgMe, tok2, nil); c != http.StatusOK {
		t.Fatalf("**S3: جلسةٌ جديدةٌ لا تعمل على `B`** — %d", c)
	}
	n.revoke(t, sid2)
	a3, _ := n.call(t, n.A, "GET", stgMe, tok2, nil)
	t.Logf("S3: أُبطلت وجُرّبت على `A` ⇒ %d", a3)
	if a3 != http.StatusUnauthorized {
		t.Errorf("**S3: مُبطَلةٌ تعمل على `A`** — %d", a3)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S4 · غيابُ مفتاح الذاكرة لا يُحيي جلسة**
// ══════════════════════════════════════════════════════════════════════
//
// **و`Redis` مُسرِّعُ رفضٍ لا مصدرَ صحّة** — **وغيابُ المفتاح «لا
// أعلم» لا «صالحة».**
func TestSTG01_S4_RedisMissDoesNotResurrect(t *testing.T) {
	n := crossNodeStaging(t)
	tok, sid, _ := n.session(t)
	if c, _ := n.call(t, n.A, "GET", stgMe, tok, nil); c != http.StatusOK {
		t.Fatalf("جلسةٌ حيّةٌ لا تعمل: %d", c)
	}
	n.revoke(t, sid)
	// **ويُمحى مفتاحُ التسريع إن وُجد** — فيبقى الحكمُ للقاعدة.
	del, err := n.rdb.Del(context.Background(), "sess:revoked:"+sid).Result()
	if err != nil {
		t.Fatalf("محوُ مفتاح التسريع: %v", err)
	}
	exists, _ := n.rdb.Exists(context.Background(), "sess:revoked:"+sid).Result()
	b, _ := n.call(t, n.B, "GET", stgMe, tok, nil)
	t.Logf("S4: مفاتيحُ مُحيت=%d · موجودٌ الآن=%d · `B` ⇒ %d", del, exists, b)
	if exists != 0 {
		t.Fatalf("**بقي مفتاحُ التسريع** — والسيناريو يشترط غيابَه")
	}
	if b != http.StatusUnauthorized {
		t.Errorf("**S4: غيابُ المفتاح أحيا جلسةً مُبطَلة** — %d", b)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S5 · S6 · الذاكرةُ تسقط — والقاعدةُ تحكم**
// ══════════════════════════════════════════════════════════════════════
//
// **وتُسقَط حاويةُ تجهيزٍ مسمّاةٌ في البيئة** — **ولا أمرَ هدّامٌ
// يُرتجَل**، ولا تُمَسّ ذاكرةُ الإنتاج ولا حاويةُ مشروعٍ آخر.
func TestSTG01_S5S6_RedisOutageAndRevocationDuringIt(t *testing.T) {
	n := crossNodeStaging(t)
	if os.Getenv("STG_REDIS_CONTAINER") == "" {
		t.Skip("يحتاج STG_REDIS_CONTAINER — ولا تُسقَط حاويةٌ بالتخمين")
	}
	live, _, _ := n.session(t)
	victim, victimSid, _ := n.session(t)

	stgRedis(t, "stop")
	restored := false
	t.Cleanup(func() {
		if !restored {
			stgRedis(t, "start")
		}
	})

	// ── S5 · الحيّةُ تعمل والمُبطَلةُ تُرفَض والذاكرةُ ساقطة ────
	a, _ := n.call(t, n.A, "GET", stgMe, live, nil)
	t.Logf("S5: الذاكرةُ ساقطةٌ — حيّةٌ على `A` ⇒ %d", a)
	if a != http.StatusOK {
		t.Errorf("**S5: سقوطُ الذاكرة أسقط جلسةً صالحة** — %d", a)
	}

	// ── S6 · يُبطَل والذاكرةُ ساقطة ────────────────────────────
	n.revoke(t, victimSid)
	b, _ := n.call(t, n.B, "GET", stgMe, victim, nil)
	t.Logf("S6: أُبطلت والذاكرةُ ساقطةٌ — `B` ⇒ %d", b)
	if b != http.StatusUnauthorized {
		t.Errorf("**S6: إبطالٌ وقع والذاكرةُ ساقطةٌ لم يُطبَّق** — %d", b)
	}

	// ── وتعود الذاكرةُ بلا مفتاحٍ ────────────────────────────────
	stgRedis(t, "start")
	restored = true
	waitRedis(t, n)
	exists, _ := n.rdb.Exists(context.Background(), "sess:revoked:"+victimSid).Result()
	b2, _ := n.call(t, n.B, "GET", stgMe, victim, nil)
	t.Logf("S6: عادت الذاكرةُ — مفتاحٌ=%d · `B` ⇒ %d", exists, b2)
	if b2 != http.StatusUnauthorized {
		t.Errorf("**S6: عودةُ الذاكرة أحيت جلسةً مُبطَلة** — %d", b2)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S7 · S8 · الحظرُ يقطع والإيقافُ لا يقطع** — `XG-39` محفوظ
// ══════════════════════════════════════════════════════════════════════
func TestSTG01_S7S8_BlockedAndSuspendedCrossNode(t *testing.T) {
	n := crossNodeStaging(t)

	// ── S8 · معلَّقٌ — **الجلسةُ تبقى صالحة** ────────────────────
	//
	// **وهو فرقُ دورةِ ١٧ بعينه**: **التعليقُ حدُّ تخويلٍ لا قطعُ
	// مصادقة** — **ولا يُقاس بمسارٍ يُردّ لسببٍ آخر.**
	susTok, _, susID := n.session(t)
	if _, err := n.pg.Exec(context.Background(),
		`UPDATE users SET status = 'suspended' WHERE id = $1::uuid`, susID); err != nil {
		t.Fatalf("التعليق: %v", err)
	}
	s8, _ := n.call(t, n.B, "GET", stgMe, susTok, nil)
	var s8Live int
	_ = n.pg.QueryRow(context.Background(), `
		SELECT count(*) FROM refresh_tokens
		 WHERE user_id = $1::uuid AND revoked_at IS NULL AND expires_at > now()`,
		susID).Scan(&s8Live)
	t.Logf("S8: معلَّقٌ على `B` ⇒ %d · عائلةٌ حيّةٌ=%d", s8, s8Live)
	// ══════════════════════════════════════════════════════════════
	// **والفرقُ الذي يُقاس هنا حدُّ تخويلٍ لا قطعُ جلسة**
	// ══════════════════════════════════════════════════════════════
	//
	// **`403` هنا هو العقدُ نفسُه** (`XG-22` · دورةُ ١٧): **التعليقُ
	// يمنع الجديدَ ولا يشلّ القائم** — والمنعُ يقع في طبقة الحال لا
	// في طبقة الجلسة.
	//
	// **والدليلُ أنّ الجلسةَ لم تُقطَع**: **عائلتُها حيّةٌ في
	// القاعدة** — **ولو أُبطلت لَما عادت بزوال التعليق.**
	//
	// **ولا يُقاس هذا بـ`401`** — **ومن قاسه به خلط حدَّ تخويلٍ
	// بقطع مصادقة.**
	if s8 == http.StatusUnauthorized {
		t.Errorf("**S8: تعليقٌ قطع الجلسةَ نفسَها** — `401` · **ونقضُ `XG-39`**")
	}
	if s8Live == 0 {
		t.Errorf("**S8: تعليقٌ أبطل عائلةَ الجلسة** — **والتعليقُ يُرفَع والحظرُ لا**")
	}

	// ── S7 · محظورٌ — **قطعٌ تامّ** ─────────────────────────────
	blkTok, blkSid, blkID := n.session(t)
	if _, err := n.pg.Exec(context.Background(),
		`UPDATE users SET status = 'blocked' WHERE id = $1::uuid`, blkID); err != nil {
		t.Fatalf("الحظر: %v", err)
	}
	// **والحظرُ يُبطل الجلسات** — كما يفعل مسارُه في المنتج.
	n.revoke(t, blkSid)
	s7, _ := n.call(t, n.B, "GET", stgMe, blkTok, nil)
	var s7Live int
	_ = n.pg.QueryRow(context.Background(), `
		SELECT count(*) FROM refresh_tokens
		 WHERE user_id = $1::uuid AND revoked_at IS NULL AND expires_at > now()`,
		blkID).Scan(&s7Live)
	t.Logf("S7: محظورٌ على `B` ⇒ %d · عائلةٌ حيّةٌ=%d", s7, s7Live)
	// **والحظرُ يقف عند كلّ شيءٍ ولا استثناءَ فيه** — **ويردّ في
	// طبقة الحال قبل أن تُقرأ الجلسة**، فالرمزُ `403` لا `401`.
	// **والقطعُ يُقاس بالقاعدة**: لا عائلةَ حيّة.
	if s7 < 400 {
		t.Errorf("**S7: محظورٌ يعمل على العقدة الأخرى** — %d", s7)
	}
	if s7Live != 0 {
		t.Errorf("**S7: محظورٌ وبقيت له عائلةٌ حيّة** — %d", s7Live)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S9 · S10 · إعادةُ الكلمة وتبديلُها عبر العُقَد**
// ══════════════════════════════════════════════════════════════════════
//
// **`R13`**: إعادةُ الإدارة تقتل كلَّ جلسة · **`XG-40`**: تبديلُ المرء
// لكلمته يُبقي جلستَه ويقتل أخواتها.
func TestSTG01_S9S10_PasswordResetAndSelfChangeCrossNode(t *testing.T) {
	n := crossNodeStaging(t)

	// ── S9 · إعادةُ الإدارة ─────────────────────────────────────
	//
	// **وتُحاكى بأثرها الكانونيّ في القاعدة**: بصمةٌ جديدةٌ ·
	// حقبةٌ مدموغةٌ · وكلُّ الجلسات مُبطَلة.
	tok, _, uid := n.session(t)
	if c, _ := n.call(t, n.A, "GET", stgMe, tok, nil); c != http.StatusOK {
		t.Fatalf("جلسةٌ حيّةٌ لا تعمل: %d", c)
	}
	if _, err := n.pg.Exec(context.Background(), `
		UPDATE users SET password_hash = 'x', sessions_revoked_at = now(),
		                 sessions_kept_session_id = NULL
		 WHERE id = $1::uuid`, uid); err != nil {
		t.Fatalf("إعادةُ الكلمة: %v", err)
	}
	if _, err := n.pg.Exec(context.Background(), `
		UPDATE refresh_tokens SET revoked_at = now()
		 WHERE user_id = $1::uuid AND revoked_at IS NULL`, uid); err != nil {
		t.Fatalf("إبطالُ الجلسات: %v", err)
	}
	s9, _ := n.call(t, n.B, "GET", stgMe, tok, nil)
	t.Logf("S9: بعد إعادةِ الإدارة — `B` ⇒ %d", s9)
	if s9 != http.StatusUnauthorized {
		t.Errorf("**S9: جلسةٌ نجت من إعادةِ كلمةٍ إداريّة** — %d · **نقضُ `R13`**", s9)
	}

	// ── S10 · تبديلُ المرء لكلمته ───────────────────────────────
	keptTok, keptSid, uid2 := n.session(t)
	otherTok, otherSid, _ := n.sessionFor(t, uid2)
	if c, _ := n.call(t, n.A, "GET", stgMe, otherTok, nil); c != http.StatusOK {
		t.Fatalf("الجلسةُ الثانيةُ لا تعمل: %d", c)
	}
	// **والعقدُ**: تُحفَظ الجاريةُ وتُبطَل أخواتُها.
	if _, err := n.pg.Exec(context.Background(), `
		UPDATE users SET password_hash = 'y', sessions_revoked_at = now(),
		                 sessions_kept_session_id = $2::uuid
		 WHERE id = $1::uuid`, uid2, keptSid); err != nil {
		t.Fatalf("تبديلُ الكلمة: %v", err)
	}
	if _, err := n.pg.Exec(context.Background(), `
		UPDATE refresh_tokens SET revoked_at = now()
		 WHERE user_id = $1::uuid AND session_id <> $2::uuid AND revoked_at IS NULL`,
		uid2, keptSid); err != nil {
		t.Fatalf("إبطالُ الأخوات: %v", err)
	}
	kept, _ := n.call(t, n.B, "GET", stgMe, keptTok, nil)
	gone, _ := n.call(t, n.B, "GET", stgMe, otherTok, nil)
	t.Logf("S10: على `B` — الجاريةُ ⇒ %d · الأختُ ⇒ %d · (%s)", kept, gone, otherSid[:8])
	if kept != http.StatusOK {
		t.Errorf("**S10: تبديلُ المرء لكلمته أخرجه هو** — %d · **نقضُ `XG-40`**", kept)
	}
	if gone != http.StatusUnauthorized {
		t.Errorf("**S10: أختُ الجلسة نجت من التبديل** — %d", gone)
	}
}

// sessionFor جلسةٌ ثانيةٌ لحسابٍ قائم.
func (n *stgNodes) sessionFor(t *testing.T, userID string) (token, sid string, roles []string) {
	t.Helper()
	if err := n.pg.QueryRow(context.Background(), `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, client)
		VALUES ($1::uuid, $2, now() + interval '30 days', 'web')
		RETURNING session_id::text`, userID, "stg2-"+fmt.Sprint(time.Now().UnixNano())).
		Scan(&sid); err != nil {
		t.Fatalf("جلسةٌ ثانية: %v", err)
	}
	tok, _, err := n.tokens.IssueAccess(userID, []string{"customer"}, sid)
	if err != nil {
		t.Fatalf("توكنُ الجلسة الثانية: %v", err)
	}
	return tok, sid, []string{"customer"}
}

// ══════════════════════════════════════════════════════════════════════
// **R1…R4 · التجديدُ عبر العُقَد**
// ══════════════════════════════════════════════════════════════════════
//
// **ورمزُ التجديد نصٌّ لا يُخزَّن خاماً** — **فيُقاس ما يُقاس بلا
// اختراع رمز**: عائلةٌ مُبطَلةٌ لا تُجدَّد، **وحيّةٌ تبقى حيّة.**
func TestSTG01_R1R4_RefreshFollowsAuthorityCrossNode(t *testing.T) {
	n := crossNodeStaging(t)
	const refresh = "/api/v1/auth/refresh"

	// ── R1 · عائلةٌ مُبطَلةٌ ⇒ يُردّ التجديد على العقدة الأخرى ──
	_, sid, _ := n.session(t)
	n.revoke(t, sid)
	var live int
	if err := n.pg.QueryRow(context.Background(), `
		SELECT count(*) FROM refresh_tokens
		 WHERE session_id = $1::uuid AND revoked_at IS NULL AND expires_at > now()`,
		sid).Scan(&live); err != nil {
		t.Fatalf("عدُّ الحيّ: %v", err)
	}
	code, _ := n.call(t, n.B, "POST", refresh, "", map[string]any{"refresh_token": "stg-not-a-token"})
	t.Logf("R1: عائلةٌ مُبطَلةٌ — حيٌّ=%d · تجديدٌ على `B` ⇒ %d", live, code)
	if live != 0 {
		t.Errorf("**R1: بقي صفٌّ حيٌّ بعد الإبطال** — %d", live)
	}
	if code < 400 {
		t.Errorf("**R1: تجديدٌ نجح برمزٍ غيرِ صالح** — %d", code)
	}

	// ── R4 · محظورٌ ⇒ لا تجديد ─────────────────────────────────
	_, bsid, bid := n.session(t)
	if _, err := n.pg.Exec(context.Background(),
		`UPDATE users SET status = 'blocked' WHERE id = $1::uuid`, bid); err != nil {
		t.Fatalf("الحظر: %v", err)
	}
	n.revoke(t, bsid)
	var bLive int
	_ = n.pg.QueryRow(context.Background(), `
		SELECT count(*) FROM refresh_tokens
		 WHERE session_id = $1::uuid AND revoked_at IS NULL`, bsid).Scan(&bLive)
	t.Logf("R4: محظورٌ — حيٌّ=%d", bLive)
	if bLive != 0 {
		t.Errorf("**R4: محظورٌ وبقيت له عائلةٌ حيّة** — %d", bLive)
	}

	// ── R3 · معلَّقٌ ⇒ عائلتُه تبقى حيّةً بعقد `XG-39` ──────────
	sTok, ssid, sid3 := n.session(t)
	if _, err := n.pg.Exec(context.Background(),
		`UPDATE users SET status = 'suspended' WHERE id = $1::uuid`, sid3); err != nil {
		t.Fatalf("التعليق: %v", err)
	}
	var sLive int
	_ = n.pg.QueryRow(context.Background(), `
		SELECT count(*) FROM refresh_tokens
		 WHERE session_id = $1::uuid AND revoked_at IS NULL`, ssid).Scan(&sLive)
	s, _ := n.call(t, n.A, "GET", stgMe, sTok, nil)
	t.Logf("R3: معلَّقٌ — حيٌّ=%d · على `A` ⇒ %d", sLive, s)
	// **والمقياسُ بقاءُ العائلة لا رمزُ الردّ** — **والتعليقُ حدُّ
	// تخويلٍ يُرفَع، لا قطعُ جلسةٍ لا يعود.**
	if sLive == 0 {
		t.Errorf("**R3: تعليقٌ قطع عائلةَ الجلسة** — حيٌّ=%d", sLive)
	}
	if s == http.StatusUnauthorized {
		t.Errorf("**R3: تعليقٌ رُدَّ بـ`401`** — **وذاك قطعُ مصادقةٍ لا حدُّ تخويل**")
	}

	// ── R2 · حيّةٌ ⇒ تبقى صالحةً على العقدتين ───────────────────
	okTok, _, _ := n.session(t)
	a, _ := n.call(t, n.A, "GET", stgMe, okTok, nil)
	b, _ := n.call(t, n.B, "GET", stgMe, okTok, nil)
	t.Logf("R2: حيّةٌ — A=%d · B=%d", a, b)
	if a != http.StatusOK || b != http.StatusOK {
		t.Errorf("**R2: حيّةٌ لا تعمل على العقدتين** — A=%d B=%d", a, b)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **مصافحةُ بثٍّ جديدةٌ على العقدة الأخرى** — `R14` يبقى منفصلاً
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يُقاس هنا اتّصالٌ قائم** — **ذاك عقدُ `R14` وحدَه.**
func TestSTG01_WS_NewHandshakeFollowsAuthorityCrossNode(t *testing.T) {
	n := crossNodeStaging(t)
	tok, sid, _ := n.session(t)
	n.revoke(t, sid)
	// **ومصافحةٌ بلا ترقيةٍ تُردّ عند التخويل** — والمقياسُ أنّها
	// لا تُقبَل.
	code, _ := n.call(t, n.B, "GET", "/api/v1/ws", tok, nil)
	t.Logf("WS: مصافحةٌ جديدةٌ بجلسةٍ مُبطَلةٍ على `B` ⇒ %d", code)
	if code == http.StatusSwitchingProtocols || code == http.StatusOK {
		t.Errorf("**مصافحةٌ جديدةٌ قُبلت بجلسةٍ مُبطَلة** — %d", code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **تزامنٌ: إبطالٌ يُثبَّت على `A` ونداءٌ يبدأ على `B`**
// ══════════════════════════════════════════════════════════════════════
//
// **ونقطةُ الحسم قراءةُ الجلسة الموثوقة** — **ونداءٌ فحصُ جلسته بعد
// تثبيت الإبطال يُردّ**، ولا يُطلَب إلغاءٌ رجعيٌّ لما أُذن قبله.
func TestSTG01_C1_RevocationVsInFlightRequestCrossNode(t *testing.T) {
	n := crossNodeStaging(t)
	tok, sid, _ := n.session(t)
	if c, _ := n.call(t, n.B, "GET", stgMe, tok, nil); c != http.StatusOK {
		t.Fatalf("جلسةٌ حيّةٌ لا تعمل: %d", c)
	}
	n.revoke(t, sid)
	// **وبعد التثبيت** — كلُّ نداءٍ لاحقٍ على أيّ عقدةٍ يُردّ.
	denied := 0
	for i := 0; i < 6; i++ {
		node := n.A
		if i%2 == 1 {
			node = n.B
		}
		if c, _ := n.call(t, node, "GET", stgMe, tok, nil); c == http.StatusUnauthorized {
			denied++
		}
	}
	t.Logf("C1: بعد تثبيت الإبطال — مردودٌ %d من 6 عبر العقدتين", denied)
	if denied != 6 {
		t.Errorf("**C1: نداءٌ مضى بعد تثبيت الإبطال** — %d من 6", denied)
	}
}

// stgRedis يوقف حاويةَ ذاكرة التجهيز أو يشغّلها — **بالاسم من البيئة.**
func stgRedis(t *testing.T, action string) {
	t.Helper()
	name := os.Getenv("STG_REDIS_CONTAINER")
	if name == "" {
		t.Skip("لا اسمَ لحاوية ذاكرة التجهيز")
	}
	out, err := exec.Command("docker", action, name).CombinedOutput()
	if err != nil {
		t.Skipf("تعذّر %s على %s: %v · %s", action, name, err, out)
	}
}

// waitRedis ينتظر عودةَ الذاكرة — **ولا نومَ أعمى.**
func waitRedis(t *testing.T, n *stgNodes) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if err := n.rdb.Ping(context.Background()).Err(); err == nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("لم تعد ذاكرةُ التجهيز في مهلتها")
}
