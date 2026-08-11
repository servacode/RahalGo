package identity

import (
	"context"
	"testing"
	"time"

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
