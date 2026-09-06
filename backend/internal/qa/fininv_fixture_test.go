package qa

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/migrate"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// ══════════════════════════════════════════════════════════════════════
// **ثوابتُ المال على أساسٍ صحيحٍ معلوم** — `FI-02.c`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان يقع
//
// **`FI-02.c` («للمنصّة خزينةٌ واحدة») كانت تُخرَق في كلّ دورة**،
// وتُوصَف مرّةً بعد مرّةٍ بأنّها «أثرُ مسحِ الاختبارات» **بلا إثبات.**
//
// **والقياسُ حسمها**: `admins=0 · treasuries=0 · users=0 · wallet_tx=0`
// — **قاعدةٌ مُهاجَرةٌ لم تُشغَّل قطّ.**
//
// **والخزينةُ لا تُنشئها هجرة**: `cmd/api/main.go` ينادي
// `wallet.EnsureTreasury` عند الإقلاع، **وهي تُرجع فارغاً حين لا
// إداريَّ بعد** فيُكتب تحذيرٌ ويمضي. **فقاعدةٌ بلا إداريٍّ بلا خزينةٍ
// حالٌ صحيحةٌ لبرنامجٍ لم يُنشَر** — **وليست خرقاً ماليّاً.**
//
// # فالعلّةُ في الأساس لا في المنتَج ولا في العقد
//
// **كنّا نُجري تدقيقاً ماليّاً على قاعدةٍ خالية.** **ومن دقّق دفتراً
// فارغاً وجد «لا خزينة» فظنّها فقداناً.**
//
// # وهذا الفحصُ يقطع الشكّ
//
// **قاعدةٌ خاصّةٌ به وحدَه**: تُنشأ وتُهاجَر وتُبنى إلى الحال التي
// يبلغها المنتَجُ عند الإقلاع، **ثمّ تُشغَّل الثوابتُ كلُّها.**
//
// **ولا تُشارك قاعدةَ الحزمة**: **تلك مقبرةُ سيناريوهات** — فحوصُ
// `P-4` تحقن فيها فساداً عمداً لتُثبت أنّ الحارسَ يراه. **ونظافةُ
// الأساس لا تُقاس في مقبرة.**
//
// **ولم تُعطَّل `FI-02.c` ولم تُستثنَ** — **بل صار لها أساسٌ تُسأل عنه.**

// fixtureDB ينشئ قاعدةً مستقلّةً مهاجَرةً ويُرجع مَسبَحَها.
func fixtureDB(t *testing.T, name string) *pgxpool.Pool {
	t.Helper()
	src := os.Getenv("TEST_DATABASE_URL")
	if src == "" {
		t.Skip("TEST_DATABASE_URL غير مضبوط")
	}
	ctx := context.Background()

	admin, err := pgxpool.New(ctx, src)
	if err != nil {
		t.Skipf("تعذّر الاتّصال: %v", err)
	}
	defer admin.Close()

	// **ولا تُنشأ قاعدةٌ باسمٍ لم يُوثَق به.**
	if !regexp.MustCompile(`^[a-z0-9_]{1,40}$`).MatchString(name) {
		t.Fatalf("اسمُ قاعدةٍ غيرُ صالح: %q", name)
	}
	if _, err := admin.Exec(ctx, `DROP DATABASE IF EXISTS `+name); err != nil {
		t.Skipf("تعذّر تنظيفُ القاعدة: %v", err)
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		t.Skipf("تعذّر إنشاءُ القاعدة: %v", err)
	}

	pool, err := pgxpool.New(ctx, swapDBName(src, name))
	if err != nil {
		t.Fatalf("مَسبَحُ الأساس: %v", err)
	}
	if _, err := migrate.Up(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("هجراتُ الأساس: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		c, err := pgxpool.New(context.Background(), src)
		if err != nil {
			return
		}
		defer c.Close()
		_, _ = c.Exec(context.Background(), `DROP DATABASE IF EXISTS `+name)
	})
	return pool
}

// swapDBName يبدّل اسمَ القاعدة في وصلةٍ نصّيّة.
func swapDBName(dsn, name string) string {
	i := strings.LastIndex(dsn, "/")
	if i < 0 {
		return dsn
	}
	rest := ""
	if j := strings.Index(dsn[i:], "?"); j >= 0 {
		rest = dsn[i+j:]
	}
	return dsn[:i+1] + name + rest
}

// ══════════════════════════════════════════════════════════════════════
// **الأساسُ الصحيحُ نظيفٌ — كلُّ الثوابت**
// ══════════════════════════════════════════════════════════════════════
func TestFIN_InvariantsCleanOnValidFixture(t *testing.T) {
	pool := fixtureDB(t, "rahalgo_fininv_fixture")
	ctx := context.Background()

	// ── ١ ── قاعدةٌ مُهاجَرةٌ بلا إداريّ: **لا خزينة** ─────────────
	//
	// **وهي حالُ قاعدةِ الاختبار التي كنّا ندقّقها.**
	var treasuries int
	_ = pool.QueryRow(ctx,
		`SELECT count(*) FROM wallets WHERE is_treasury`).Scan(&treasuries)
	before, err := fininv.Run(ctx, pool, "FI-02.c")
	if err != nil {
		t.Fatalf("تشغيلُ الثابت: %v", err)
	}
	t.Logf("مُهاجَرةٌ بلا إقلاع: خزائنُ=%d · خروقُ FI-02.c=%d",
		treasuries, len(before))
	if len(before) == 0 {
		t.Error("**قاعدةٌ خاليةٌ لم تُخرَق فيها `FI-02.c`** — " +
			"**فالتشخيصُ الذي بُني عليه هذا الفحصُ باطل.**")
	}

	// ── ٢ ── ثمّ ما يفعله الإقلاعُ بالضبط ─────────────────────────
	//
	// **إداريٌّ ثمّ `EnsureTreasury`** — **شيفرةُ المنتَج نفسُها**،
	// `cmd/api/main.go:198`. **ولا يُكتب صفُّ خزينةٍ بيدٍ هنا**:
	// **من زرع الأساسَ بيده أثبت زرعَه لا أثبت المنتَج.**
	var adminID string
	if err := pool.QueryRow(ctx, `
		WITH u AS (
		  INSERT INTO users (phone, full_name, status)
		  VALUES ('0900000001', 'إداريُّ الأساس', 'active') RETURNING id)
		INSERT INTO user_roles (user_id, role_code)
		SELECT id, 'admin' FROM u RETURNING user_id::text`).Scan(&adminID); err != nil {
		t.Fatalf("إداريُّ الأساس: %v", err)
	}
	tid, err := wallet.EnsureTreasury(ctx, pool)
	if err != nil {
		t.Fatalf("EnsureTreasury: %v", err)
	}
	if tid == "" {
		t.Fatal("**لم تُنشَأ خزينةٌ وللمنصّة إداريّ** — وهذا عطبُ منتَج")
	}

	// ── ٣ ── والثوابتُ كلُّها على أساسٍ صحيح ──────────────────────
	all, err := fininv.Run(ctx, pool)
	if err != nil {
		t.Fatalf("تشغيلُ الثوابت: %v", err)
	}
	total := len(fininv.Select())
	t.Logf("أساسٌ صحيح: خزينةٌ %s · %d ثابتاً · %d خرقاً",
		first8(tid), total, len(all))
	for _, v := range all {
		t.Errorf("**خرقٌ على أساسٍ صحيح** — %s", v)
	}
}
