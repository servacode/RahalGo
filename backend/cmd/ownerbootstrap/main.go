// أداةُ تهيئةِ أوّلِ مالك — **مرّةً واحدةً في عمر المنصّة.**
//
// ══════════════════════════════════════════════════════════════════════
//
//	**لماذا وُجدت**
//
// ══════════════════════════════════════════════════════════════════════
//
// **سياسةُ ٢٠٢٦-٠٩-١٢ تجعل `owner_super_admin` و`admin` للمالك وحدَه**
// منحاً — **ولا يرقّي أدمنٌ نفسَه** (قرارُ المالك، بندُ و).
//
// **وقِيس أنّ الإنتاجَ فيه صفرُ مالكين.** **فالسياسةُ مُغلَقةٌ بقصد**:
// لا مالكَ يمنح، ولا بابَ في الـAPI يُنشئ أوّلَه. **ولو فُتح بابٌ
// «إن لم يوجد مالكٌ فليُنشئه أيُّ أدمن» لَعادت الثغرةُ بعينها.**
//
// **فالمفتاحُ على الخادم لا في الشبكة** — وهذا هو.
//
// ══════════════════════════════════════════════════════════════════════
//
//	**وحارسُها معكوسٌ عن `stagingctl`**
//
// ══════════════════════════════════════════════════════════════════════
//
// **`stagingctl` ترفض الإنتاجَ** (`envguard.MustBeSafe`) لأنّ أوامرَها
// هدّامة. **وهذه لا تعمل إلّا في الإنتاج** — فحارسُها معكوس:
//
//	١ · `APP_ENV` يجب أن يكون `production` — **ولا تعمل في غيره**
//	    (والتجهيزُ له بذرُه، ولا يحتاج هذه)
//	٢ · و`DATABASE_URL` يجب أن يشير إلى قاعدةِ إنتاجٍ معروفةٍ باسمها
//	    (`envguard.ProductionDBNames`)
//	٣ · وإذنٌ مكتوبٌ حرفاً بحرفٍ في البيئة — **فلا تُنادى بالخطأ**
//	٤ · ورقمُ الهاتف يُؤكَّد مرّتين — **والثانيةُ بعد رؤية الهويّة**
//
// ══════════════════════════════════════════════════════════════════════
//
//	**والمرّةُ الواحدةُ بالثابت لا بعلَم**
//
// ══════════════════════════════════════════════════════════════════════
//
// **ولا صفَّ «قد نُفِّذت» يُحفظ** — **وعلَمٌ يُمحى تُعاد به الأداة.**
// **والشرطُ نفسُه هو الضمان**: **صفرُ مالكين.** فبعد أوّلِ نجاحٍ صار
// العددُ واحداً، **فتردّ نفسَها إلى الأبد** ولو نُوديت ألفاً.
//
// **ويُقفل الصفوفُ ثمّ يُعَدّ داخلَ المعاملة** — **وأمران متزامنان
// يقرأ كلٌّ منهما «صفراً» فيكتبان معاً ⇒ مالكان.**
//
// الاستعمال:
//
//	ownerbootstrap -check -phone 09xxxxxxxx
//	  **قراءةٌ محضة** — يطبع الهويّةَ والعددَ ولا يكتب حرفاً
//
//	ownerbootstrap -phone 09xxxxxxxx -confirm-phone 09xxxxxxxx
//	  **يكتب** — وبإذن البيئة، وبتطابق الرقمين بعد التطبيع
//
// ورموزُ الخروج:
//
//	0  تمّ
//	1  فشلٌ في التنفيذ
//	2  خطأُ استعمال
//	3  **الحارسُ رفض** — وهو الفرقُ الذي يُقرأ في خطّ النشر
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/envguard"
	"github.com/servacode/rahalgo/backend/internal/identity"
)

const (
	exitOK      = 0
	exitFail    = 1
	exitUsage   = 2
	exitGuarded = 3
)

// authzEnv **الإذنُ المكتوبُ حرفاً بحرف.**
//
// **ولا قيمةَ «نعم» ولا «1»** — **جملةٌ لا تُكتب سهواً ولا تبقى في
// ملفِّ بيئةٍ بالخطأ.**
const (
	authzEnv  = "RAHALGO_OWNER_BOOTSTRAP"
	authzWord = "I-AUTHORIZE-FIRST-OWNER"
)

func main() {
	var (
		check   = flag.Bool("check", false, "قراءةٌ محضة — لا كتابة")
		phone   = flag.String("phone", "", "هاتفُ الحساب القائم")
		confirm = flag.String("confirm-phone", "", "الهاتفُ ثانيةً — تأكيدٌ صريح")
	)
	flag.Parse()

	if strings.TrimSpace(*phone) == "" {
		fmt.Fprintln(os.Stderr, "استعمال: ownerbootstrap [-check] -phone 09xxxxxxxx [-confirm-phone 09xxxxxxxx]")
		os.Exit(exitUsage)
	}

	// ── ١ · التطبيعُ بدالّة المشروع لا بيدٍ ──────────────────────────
	//
	// **ولا يُكتب `+963` هنا** — **ومن طبّع بيده أنشأ حساباً ثانياً
	// لرقمٍ واحد.** (`package-identity-from-source` بعينه.)
	norm, ok := identity.NormalizePhone(*phone)
	if !ok {
		fmt.Fprintf(os.Stderr, "✗ رقمٌ غيرُ صالح: %q\n", *phone)
		os.Exit(exitUsage)
	}
	fmt.Printf("INPUT_PHONE=%s\nNORMALIZED_PHONE=%s\n", *phone, norm)

	env := envguard.FromOS()
	if env.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "✗ DATABASE_URL غيرُ مضبوط")
		os.Exit(exitUsage)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, env.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ تعذّر الاتّصال: %v\n", err)
		os.Exit(exitFail)
	}
	defer pool.Close()

	// ── ٢ · الهويّةُ والعددُ — قراءةٌ قبل كلّ شيء ────────────────────
	tgt, err := readTarget(ctx, pool, norm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ %v\n", err)
		os.Exit(exitFail)
	}
	owners, err := countOwners(ctx, pool)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ عدُّ المالكين: %v\n", err)
		os.Exit(exitFail)
	}

	fmt.Printf("DB_NAME=%s\nAPP_ENV=%s\n", dbNameOf(env.DatabaseURL), env.AppEnv)
	fmt.Printf("TARGET_ACCOUNT_COUNT=%d\n", tgt.matches)
	if tgt.matches == 1 {
		fmt.Printf("TARGET_USER_ID=%s\nTARGET_DISPLAY_NAME=%q\nTARGET_PHONE=%s\n"+
			"TARGET_STATUS=%s\nTARGET_ROLES=%s\nTARGET_CREATED_AT=%s\n",
			tgt.id, tgt.name, tgt.phone, tgt.status,
			strings.Join(tgt.roles, ","), tgt.created.UTC().Format(time.RFC3339))
	}
	fmt.Printf("OWNER_COUNT=%d\n", owners)

	if *check {
		// **وفحصٌ يقول ماذا سيقع لو كُتب** — بلا كتابة.
		fmt.Printf("WOULD_PROCEED=%v\n", tgt.matches == 1 && tgt.status == "active" && owners == 0)
		fmt.Printf("SECOND_RUN_WOULD_BE_BLOCKED=%v\n", owners >= 1)
		fmt.Println("CHECK_ONLY=YES — لم يُكتب حرف")
		os.Exit(exitOK)
	}

	// ── ٣ · الحارسُ المعكوس ─────────────────────────────────────────
	if strings.ToLower(strings.TrimSpace(env.AppEnv)) != "production" {
		fmt.Fprintf(os.Stderr, "✗ الحارس: APP_ENV=%q — **وهذه الأداةُ للإنتاج وحدَه**\n", env.AppEnv)
		os.Exit(exitGuarded)
	}
	if !isProductionDB(env.DatabaseURL) {
		fmt.Fprintf(os.Stderr, "✗ الحارس: القاعدةُ %q ليست قاعدةَ إنتاجٍ معروفة\n",
			dbNameOf(env.DatabaseURL))
		os.Exit(exitGuarded)
	}
	if os.Getenv(authzEnv) != authzWord {
		fmt.Fprintf(os.Stderr, "✗ الحارس: %s غيرُ مضبوطٍ بالإذن الحرفيّ\n", authzEnv)
		os.Exit(exitGuarded)
	}
	// **والتأكيدُ بعد رؤية الهويّة** — ويُقارَن مطبَّعاً لا نصّاً.
	cnorm, cok := identity.NormalizePhone(*confirm)
	if !cok || cnorm != norm {
		fmt.Fprintf(os.Stderr, "✗ الحارس: التأكيدُ لا يطابق — %q ضدّ %q\n", *confirm, *phone)
		os.Exit(exitGuarded)
	}
	if tgt.matches != 1 {
		fmt.Fprintf(os.Stderr, "✗ الحارس: الحساباتُ المطابقةُ %d ولا يجوز إلّا واحد\n", tgt.matches)
		os.Exit(exitGuarded)
	}
	if tgt.status != "active" {
		fmt.Fprintf(os.Stderr, "✗ الحارس: حالُ الحساب %q — ولا يُرقّى غيرُ الفعّال\n", tgt.status)
		os.Exit(exitGuarded)
	}
	if owners != 0 {
		fmt.Fprintf(os.Stderr, "✗ الحارس: المالكونَ %d — **والتهيئةُ لأوّلِ مالكٍ وحدَه**\n", owners)
		os.Exit(exitGuarded)
	}

	// ── ٤ · الكتابةُ وأثرُها في معاملةٍ واحدة ────────────────────────
	// **والكتابةُ في خدمة الهويّة لا هنا** — **ومن كتب صفّاً من
	// `cmd/` تجاوز حدَّ المعاملة وقيدَ التدقيق وحارسَ السباق.**
	if err := identity.BootstrapFirstOwner(ctx, pool, tgt.id, norm); err != nil {
		fmt.Fprintf(os.Stderr, "✗ %v\n", err)
		if errors.Is(err, identity.ErrOwnerAlreadyExists) {
			os.Exit(exitGuarded)
		}
		os.Exit(exitFail)
	}

	after, err := countOwners(ctx, pool)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ عدُّ المالكين بعدَ الكتابة: %v\n", err)
		os.Exit(exitFail)
	}
	fmt.Printf("BOOTSTRAP=DONE\nOWNER_COUNT_AFTER=%d\nSECOND_RUN_WOULD_BE_BLOCKED=%v\n",
		after, after >= 1)
	os.Exit(exitOK)
}

type target struct {
	matches int
	id      string
	name    string
	phone   string
	status  string
	roles   []string
	created time.Time
}

// readTarget **الهويّةُ وأدوارُها** — ولا بصمةَ كلمةٍ ولا رمزَ لوحةٍ ولا جلسة.
func readTarget(ctx context.Context, pool *pgxpool.Pool, norm string) (target, error) {
	var t target
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM users WHERE phone = $1`, norm).Scan(&t.matches); err != nil {
		return t, fmt.Errorf("عدُّ الحسابات: %w", err)
	}
	if t.matches != 1 {
		return t, nil
	}
	if err := pool.QueryRow(ctx, `
		SELECT u.id::text, u.full_name, u.phone::text, u.status, u.created_at,
		       COALESCE(array_agg(ur.role_code ORDER BY ur.role_code)
		                FILTER (WHERE ur.role_code IS NOT NULL), '{}')
		  FROM users u LEFT JOIN user_roles ur ON ur.user_id = u.id
		 WHERE u.phone = $1
		 GROUP BY u.id, u.full_name, u.phone, u.status, u.created_at`,
		norm).Scan(&t.id, &t.name, &t.phone, &t.status, &t.created, &t.roles); err != nil {
		return t, fmt.Errorf("قراءةُ الحساب: %w", err)
	}
	return t, nil
}

func countOwners(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var n int
	err := pool.QueryRow(ctx,
		`SELECT count(*) FROM user_roles WHERE role_code = $1`,
		authz.RoleOwnerSuperAdmin).Scan(&n)
	return n, err
}

func dbNameOf(raw string) string {
	if i := strings.LastIndex(raw, "/"); i >= 0 {
		rest := raw[i+1:]
		if j := strings.IndexAny(rest, "?"); j >= 0 {
			return rest[:j]
		}
		return rest
	}
	return ""
}

func isProductionDB(raw string) bool {
	name := dbNameOf(raw)
	for _, p := range envguard.ProductionDBNames {
		if name == p {
			return true
		}
	}
	return false
}
