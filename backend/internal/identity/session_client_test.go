package identity

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// ══════════════════════════════════════════════════════════════════════
// **جلسةُ التطبيق لا تقتل جلسةَ المتصفّح — وتقتل جلسةَ هاتفٍ آخر**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذا اختبارُ أثرٍ لا اختبارُ دالّة.** حارسُ `client_kind_test.go` يفحص
// تطبيعَ النصّ، **وهو يمرّ حتّى لو لم يُقرأ النوعُ في الإبطال أصلاً** —
// وذاك بالضبط شكلُ العطب: عمودٌ يُكتب ولا يُقرأ.
//
// **فيُقاس على قاعدةٍ حقيقيّة**: تُخزَّن جلستان بنوعين، ثمّ يُبطَل نوعٌ،
// **ويُسأل من بقي حيّاً.**
//
// # ما يُكسَر إن سقط
//
// **السائقُ يفتح التطبيقَ فيخرج من الويب، ويفتح الويبَ فيخرج من التطبيق**
// — حلقةٌ لا مخرجَ منها، وهي سببُ هذا التغيير كلِّه.
//
// # وما لا يبلغه هذا الاختبار
//
// **مرآةُ Redis** التي تُبطل توكناتِ الوصول القائمة (`revokeClientSessions`)
// — لا خادمَ Redis في بنية الاختبارات. **وهي السطرُ نفسُه الحرفيَّ الذي
// يستعمله `revokeAllSessions` منذ زمنٍ ويُغطّيه الوسيط.**

const testSessionTTL = time.Hour

func TestSessionClient_AppDoesNotKillWeb(t *testing.T) {
	pool := testdb.Pool(t)
	repo := NewRepo(pool)
	ctx := context.Background()
	uid := testdb.NewUser(t, pool, "driver")

	web, err := repo.StoreRefresh(ctx, uid, "hash-web", testSessionTTL, "ua-web", "1.1.1.1", "", ClientWeb)
	if err != nil {
		t.Fatalf("جلسةُ المتصفّح لم تُخزَّن: %v", err)
	}
	app, err := repo.StoreRefresh(ctx, uid, "hash-app", testSessionTTL, "ua-app", "2.2.2.2", "", "android-driver")
	if err != nil {
		t.Fatalf("جلسةُ التطبيق لم تُخزَّن: %v", err)
	}
	if web == app {
		t.Fatal("الجلستان عائلةٌ واحدة — **فإبطالُ إحداهما يقتل الأخرى حتماً**")
	}

	// **دخولٌ من التطبيق يُبطل جلساتِ التطبيق وحدَها** — وهو ما يفعله
	// `issueSession` عند دخولٍ جديد.
	if _, err := repo.RevokeClientTokens(ctx, uid, "android-driver"); err != nil {
		t.Fatalf("إبطالُ نوعٍ واحد: %v", err)
	}

	alive := repo.mustActive(ctx, t, uid)
	if !alive[web] {
		t.Fatal("دخولُ التطبيق أبطل جلسةَ المتصفّح — **وهي الحلقةُ المقفلة**: " +
			"السائقُ يفتح التطبيقَ فيخرج من الويب، ويفتح الويبَ فيخرج من التطبيق")
	}
	if alive[app] {
		t.Fatal("جلسةُ التطبيق نجت من إبطال نوعها — **فيبقى هاتفٌ ضائعٌ داخلاً**")
	}
}

func TestSessionClient_SecondPhoneKillsFirst(t *testing.T) {
	pool := testdb.Pool(t)
	repo := NewRepo(pool)
	ctx := context.Background()
	uid := testdb.NewUser(t, pool, "driver")

	first, err := repo.StoreRefresh(ctx, uid, "hash-p1", testSessionTTL, "هاتف-١", "2.2.2.2", "", "android-driver")
	if err != nil {
		t.Fatalf("الهاتفُ الأوّل: %v", err)
	}
	// **والثاني يُبطل ما سبق من نوعه** — القيدُ يُضيَّق ولا يُلغى.
	if _, err := repo.RevokeClientTokens(ctx, uid, "android-driver"); err != nil {
		t.Fatalf("إبطال: %v", err)
	}
	second, err := repo.StoreRefresh(ctx, uid, "hash-p2", testSessionTTL, "هاتف-٢", "3.3.3.3", "", "android-driver")
	if err != nil {
		t.Fatalf("الهاتفُ الثاني: %v", err)
	}

	alive := repo.mustActive(ctx, t, uid)
	if alive[first] {
		t.Fatal("هاتفان يعملان على حسابٍ واحد — **فيتشارك سائقان حساباً، " +
			"فيقبضان على اسمٍ واحدٍ ولا تعرف المنصّةُ من سلّم**")
	}
	if !alive[second] {
		t.Fatal("الهاتفُ الثاني أبطل نفسَه")
	}
}

// TestSessionClient_AdminLogoutAllStillGlobal **و«إنهاءُ الجلسات» يبقى شاملاً.**
//
// **وهذا أخطرُ ما ينكسر بالسهو**: أداةُ الإدارة لقطع وصولِ حسابٍ موقوف،
// **ولو صارت تُبطل نوعاً واحداً لَبقي الموقوفُ يعمل من هاتفه** والإدارةُ
// تراه مقطوعاً.
func TestSessionClient_AdminLogoutAllStillGlobal(t *testing.T) {
	pool := testdb.Pool(t)
	repo := NewRepo(pool)
	ctx := context.Background()
	uid := testdb.NewUser(t, pool, "driver")

	if _, err := repo.StoreRefresh(ctx, uid, "g-web", testSessionTTL, "ua-web", "1.1.1.1", "", ClientWeb); err != nil {
		t.Fatalf("جلسةُ المتصفّح: %v", err)
	}
	if _, err := repo.StoreRefresh(ctx, uid, "g-app", testSessionTTL, "ua-app", "2.2.2.2", "", "android-driver"); err != nil {
		t.Fatalf("جلسةُ التطبيق: %v", err)
	}

	n, err := repo.RevokeAllTokens(ctx, uid)
	if err != nil {
		t.Fatalf("الإبطالُ الشامل: %v", err)
	}
	if n != 2 {
		t.Fatalf("«إنهاءُ الجلسات» أبطل %d من ٢ — **فيبقى الموقوفُ يعمل من هاتفه "+
			"والإدارةُ تراه مقطوعاً**", n)
	}
	if len(repo.mustActive(ctx, t, uid)) != 0 {
		t.Fatal("بقيت جلسةٌ حيّةٌ بعد الإبطال الشامل")
	}
}

// TestSessionClient_DefaultIsWeb **والصفوفُ القديمةُ متصفّحٌ** — الافتراضُ
// في الهجرة `0099`. **ولو كانت فارغةً لَما أبطلها دخولُ ويبٍ جديد**، فبقيت
// جلساتٌ قديمةٌ حيّةً إلى الأبد.
func TestSessionClient_DefaultIsWeb(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	uid := testdb.NewUser(t, pool, "customer")

	// كتابةٌ لا تذكر العمودَ — كما كُتبت كلُّ الصفوف قبل الهجرة.
	if _, err := pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, 'legacy', now() + interval '1 hour')`, uid); err != nil {
		t.Fatalf("صفٌّ قديم: %v", err)
	}
	var client string
	if err := pool.QueryRow(ctx,
		`SELECT client FROM refresh_tokens WHERE token_hash = 'legacy'`).Scan(&client); err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	if client != ClientWeb {
		t.Fatalf("الصفُّ القديم نوعُه %q لا %q — **فلا يُبطله دخولُ ويبٍ جديد "+
			"وتبقى جلساتٌ قديمةٌ حيّة**", client, ClientWeb)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **الإزاحةُ تقطع وجهةَ الجهاز القديم وتُثبِت السبب — بلا مساسٍ بالمتصفّح**
// ══════════════════════════════════════════════════════════════════════
//
// **اختبارُ أثرٍ على قاعدةٍ حقيقيّة** (Obs 3): دخولٌ جديدٌ من نوعِ العميل نفسِه
// يجب أن (١) يُبطل العائلةَ القديمة، (٢) يقطع وجهةَ دفعها فلا يستقبل جهازٌ
// مُخرَجٌ إشعاراً خاصّاً، (٣) يُثبِت السببَ `superseded` دائماً في القاعدة،
// (٤) **ولا يمسّ جلسةَ المتصفّح ولا وجهتَها** — القيدُ لكلّ نوعٍ لا للحساب.
func TestSessionClient_SupersedeCutsDeviceTokenAndStampsReason(t *testing.T) {
	pool := testdb.Pool(t)
	repo := NewRepo(pool)
	ctx := context.Background()
	uid := testdb.NewUser(t, pool, "customer")

	appSID, err := repo.StoreRefresh(ctx, uid, "hash-old-app", testSessionTTL, "هاتف قديم", "2.2.2.2", "", "android-customer")
	if err != nil {
		t.Fatalf("جلسةُ التطبيق: %v", err)
	}
	webSID, err := repo.StoreRefresh(ctx, uid, "hash-web", testSessionTTL, "متصفّح", "1.1.1.1", "", ClientWeb)
	if err != nil {
		t.Fatalf("جلسةُ المتصفّح: %v", err)
	}
	insertDeviceToken(ctx, t, pool, "tok-old-app", uid, appSID)
	insertDeviceToken(ctx, t, pool, "tok-web", uid, webSID)

	sids, err := repo.RevokeClientSessionsAtomic(ctx, uid, "android-customer")
	if err != nil {
		t.Fatalf("الإبطالُ الذرّي: %v", err)
	}
	if len(sids) != 1 || sids[0] != appSID {
		t.Fatalf("العائلاتُ المُبطَلة %v — يُنتظَر [%s] وحدَها", sids, appSID)
	}

	// (٢) **وجهةُ الجهاز القديم تُقطَع** — وإلّا استقبل جهازٌ مُخرَجٌ إشعاراً خاصّاً.
	if deviceTokenExists(ctx, t, pool, "tok-old-app") {
		t.Fatal("رمزُ دفعِ الجهاز القديم بقي بعد الإزاحة — **انكشافُ خصوصيّة**")
	}
	// (٤) **ووجهةُ المتصفّح تبقى** — القيدُ لكلّ نوعِ عميل، لا للحساب كلِّه.
	if !deviceTokenExists(ctx, t, pool, "tok-web") {
		t.Fatal("إزاحةُ التطبيق قطعت وجهةَ المتصفّح — **والقيدُ لكلّ نوعٍ لا للحساب**")
	}
	// (٣) **والسببُ يُثبَت دائماً** — فيعرفه الجهازُ القديمُ ولو عاد بعد زوال Redis.
	if got, _ := repo.RevokedReasonOfSession(ctx, appSID); got != ReasonSuperseded {
		t.Fatalf("سببُ العائلة %q لا %q", got, ReasonSuperseded)
	}
	if got, _ := repo.RevokedReasonOfToken(ctx, "hash-old-app"); got != ReasonSuperseded {
		t.Fatalf("سببُ التوكن %q لا %q — **فالجهازُ العائدُ متأخّراً لا يعرف أنّه «جهازٌ آخر»**", got, ReasonSuperseded)
	}
	// (١)+(٤) **والمتصفّحُ حيٌّ** بعد إزاحة التطبيق.
	if !repo.mustActive(ctx, t, uid)[webSID] {
		t.Fatal("جلسةُ المتصفّح ماتت مع إزاحة التطبيق")
	}
}

// TestSessionClient_GenericRevokeStaysDistinct **والإبطالُ العامُّ يبقى مميَّزاً**
// (Obs 3): خروجٌ · حظرٌ · حذفٌ · إعادةُ كلمة لا تُوسَم `superseded` — فلا يُعرَض
// «من جهازٍ آخر» في غير موضعه.
func TestSessionClient_GenericRevokeStaysDistinct(t *testing.T) {
	pool := testdb.Pool(t)
	repo := NewRepo(pool)
	ctx := context.Background()
	uid := testdb.NewUser(t, pool, "customer")

	sid, err := repo.StoreRefresh(ctx, uid, "hash-generic", testSessionTTL, "هاتف", "2.2.2.2", "", "android-customer")
	if err != nil {
		t.Fatalf("جلسة: %v", err)
	}
	if _, err := repo.RevokeAllTokens(ctx, uid); err != nil {
		t.Fatalf("الإبطالُ الشامل: %v", err)
	}
	if got, _ := repo.RevokedReasonOfSession(ctx, sid); got == ReasonSuperseded {
		t.Fatal("إبطالٌ شاملٌ وُسم `superseded` — **فيُعرَض «من جهازٍ آخر» على خروجٍ عاديّ**")
	}
}

// insertDeviceToken وجهةُ دفعٍ مربوطةٌ بعائلةِ جلسة — لاختبار قطعِها عند الإزاحة.
func insertDeviceToken(ctx context.Context, t *testing.T, pool *pgxpool.Pool, token, userID, sid string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO device_tokens (token, user_id, session_id, platform, app, app_version)
		VALUES ($1, $2, $3::uuid, 'android', 'customer', '1.0')`, token, userID, sid); err != nil {
		t.Fatalf("إدراجُ وجهةِ الدفع %q: %v", token, err)
	}
}

// deviceTokenExists **أباقٍ صفُّ الوجهة؟** — أساسُ فحصِ قطعِ الوجهة.
func deviceTokenExists(ctx context.Context, t *testing.T, pool *pgxpool.Pool, token string) bool {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM device_tokens WHERE token = $1`, token).Scan(&n); err != nil {
		t.Fatalf("عدُّ وجهةِ الدفع %q: %v", token, err)
	}
	return n > 0
}

// mustActive عائلاتُ الجلسات الحيّة — مجموعةً لتُسأل بالاسم.
func (r *Repo) mustActive(ctx context.Context, t *testing.T, userID string) map[string]bool {
	t.Helper()
	sids, err := r.ActiveSessionIDs(ctx, userID)
	if err != nil {
		t.Fatalf("قراءةُ الجلسات الحيّة: %v", err)
	}
	out := map[string]bool{}
	for _, sid := range sids {
		out[sid] = true
	}
	return out
}
