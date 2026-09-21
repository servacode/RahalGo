// أداةُ بيئة التجهيز — **وكلُّ أمرٍ فيها يمرّ بحارس الإنتاج أوّلاً.**
//
// (`P-0` البنود ٦ و١٧ و١٨ و٢٩ و٤٣.)
//
// # ولماذا أداةٌ واحدةٌ لا نصوصٌ متفرّقة
//
// **البند ١٧ يشترط ألّا تعتمد إعادةُ الضبط على قصاصات `SQL` يدويّة** —
// **ومن نسخ جملةَ `DELETE` من مستندٍ نسي `WHERE` مرّةً.**
//
// # ورموزُ الخروج
//
//	0  تمّ
//	1  فشلٌ في التنفيذ
//	2  خطأُ استعمال
//	3  **حارسُ الإنتاج رفض** — وهو الفرقُ الذي يُقرأ في خطّ النشر
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/envguard"
)

const (
	exitOK      = 0
	exitFail    = 1
	exitUsage   = 2
	exitGuarded = 3
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(exitUsage)
	}
	cmd := os.Args[1]

	// ══════════════════════════════════════════════════════════════
	// **ولا أمرَ قبل الحارس** (البند ٦)
	// ══════════════════════════════════════════════════════════════
	//
	// **حتّى `guard` نفسُه** — فهو تقريرُه.
	env := envguard.FromOS()
	res := envguard.Inspect(env)

	if cmd == "guard" {
		report(res)
		if !res.Safe {
			os.Exit(exitGuarded)
		}
		os.Exit(exitOK)
	}
	if err := envguard.MustBeSafe(env); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitGuarded)
	}

	var err error
	switch cmd {
	case "reset":
		err = reset()
	case "backup":
		err = backup()
	case "restore":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "restore يحتاج مسارَ نسخة")
			os.Exit(exitUsage)
		}
		err = restore(os.Args[2])
	case "verify":
		err = verify()
	case "flag":
		// **بعد الحارس** — قلبُ إعدادٍ مسموحٍ للحملة (الفئة B).
		err = flagCmd(os.Args[2:])
	default:
		usage()
		os.Exit(exitUsage)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ %v\n", err)
		os.Exit(exitFail)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `أداةُ بيئة التجهيز

	stagingctl guard      يفحص الحرّاسَ الخمسةَ ويقول حكمَ كلٍّ منها
	stagingctl reset      يمحو بياناتِ التجربة ويُبقي المخطَّط
	stagingctl backup     نسخةٌ كاملة
	stagingctl restore F  استعادةٌ من نسخة
	stagingctl verify     يعدّ الصفوفَ ويثبت سلامةَ المخطَّط
	stagingctl flag get K       يقرأ إعداداً مسموحاً
	stagingctl flag set K V     يقلب إعداداً مسموحاً ويطبع أمرَ الاستعادة

**ولا بابَ خلفيّ** — كلُّها ترفض الإنتاجَ بخمسة حرّاسٍ مستقلّة.
`)
}

func report(r envguard.Result) {
	fmt.Println("حرّاسُ الإنتاج:")
	for _, c := range r.Checks {
		mark := "✓"
		if !c.Passed {
			mark = "✗"
		}
		fmt.Printf("  %s %-16s %s\n", mark, c.Name, c.Detail)
	}
	if r.Safe {
		fmt.Println("\n✓ بيئةٌ آمنةٌ للأوامر الهدّامة")
		return
	}
	fmt.Println("\n✗ **رُفض** — ولا بابَ خلفيّ")
}

func pool() (*pgxpool.Pool, error) {
	return pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
}

// ══════════════════════════════════════════════════════════════════════
// **إعادةُ الضبط — البند ١٧**
// ══════════════════════════════════════════════════════════════════════
//
// # ويُبقي المخطَّطَ والهجرات
//
// **`TRUNCATE` لا `DROP`** — **ومن أسقط المخطَّطَ أعاد الهجراتِ كلَّها
// في كلّ إعادةِ ضبط**، وذلك دقائقُ في كلّ دورةِ اختبار.
//
// **و`schema_migrations` لا تُمسّ** — **وإلّا أُعيد تطبيقُ ما طُبِّق.**
//
// # ولا تُعدَّد الجداولُ بيد
//
// **جدولٌ جديدٌ يُضاف في هجرةٍ ولا يُضاف هنا يبقى فيه صفٌّ بعد كلّ
// إعادةِ ضبط** — **فتُقرأ أسماءُ الجداول من القاعدة نفسِها.**
func reset() error {
	ctx := context.Background()
	db, err := pool()
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query(ctx, `
		SELECT tablename FROM pg_tables
		 WHERE schemaname = 'public'
		   AND tablename NOT IN ('schema_migrations', 'spatial_ref_sys')
		 ORDER BY tablename`)
	if err != nil {
		return err
	}
	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			rows.Close()
			return err
		}
		tables = append(tables, `"`+t+`"`)
	}
	rows.Close()
	if len(tables) == 0 {
		return fmt.Errorf("لا جدولَ واحدٌ — أهذه قاعدةٌ مهاجَرة؟")
	}

	// **و`CASCADE` لازمة** — المفاتيحُ الأجنبيّةُ تمنع الإفراغَ جدولاً جدولاً.
	// **و`RESTART IDENTITY` تُعيد العدّادات** — فأرقامُ الطلبات تبدأ من
	// جديدٍ في كلّ دورة، **وبيانتان لا تشتركان في رقم.**
	sql := "TRUNCATE TABLE " + strings.Join(tables, ", ") + " RESTART IDENTITY CASCADE"
	if _, err := db.Exec(ctx, sql); err != nil {
		return fmt.Errorf("الإفراغ: %w", err)
	}
	fmt.Printf("✓ أُفرغ %d جدولاً — والمخطَّطُ والهجراتُ كما هي\n", len(tables))
	return nil
}

// ══════════════════════════════════════════════════════════════════════
// **النسخُ والاستعادة — البند ٢٩**
// ══════════════════════════════════════════════════════════════════════

func dsnParts() (host, port, user, pass, name string) {
	// **ويُقرأ من `DATABASE_URL` وحدَه** — **ولا يُبنى من متغيّراتٍ
	// متفرّقةٍ قد يشير كلٌّ منها إلى مكان.**
	raw := os.Getenv("DATABASE_URL")
	rest := strings.TrimPrefix(raw, "postgres://")
	rest = strings.TrimPrefix(rest, "postgresql://")
	if i := strings.Index(rest, "@"); i >= 0 {
		cred := rest[:i]
		rest = rest[i+1:]
		if j := strings.Index(cred, ":"); j >= 0 {
			user, pass = cred[:j], cred[j+1:]
		} else {
			user = cred
		}
	}
	hostPort := rest
	if i := strings.Index(rest, "/"); i >= 0 {
		hostPort = rest[:i]
		name = rest[i+1:]
		if j := strings.Index(name, "?"); j >= 0 {
			name = name[:j]
		}
	}
	host = hostPort
	port = "5432"
	if i := strings.LastIndex(hostPort, ":"); i >= 0 {
		host, port = hostPort[:i], hostPort[i+1:]
	}
	return
}

// pgTool يبني نداءَ أداةِ بوستغرس — **داخلَ الحاوية أو خارجَها.**
//
// # ولماذا الوجهان
//
// **على الخادم قد تكون الأدواتُ مثبَّتةً بجانب دوكر** — **وعلى جهازٍ
// آخرَ لا تكون.** **وأداةٌ تشترط تثبيتَ عميلِ بوستغرس على كلّ مضيفٍ
// تُعطّل التدريبَ حيث يلزم أكثرَ ما يلزم.**
//
// **و`STAGING_PG_CONTAINER` مضبوطةٌ ⇒ يُنادى داخلَ الحاوية** —
// **وهناك يوجد `pg_dump` دائماً، وحجمُ النسخ مركَّبٌ عليها.**
//
// **والمضيفُ يصير `localhost` داخلَها** — **وعنوانُ المضيف من خارجٍ
// لا يُحلّ في الداخل.**
func pgTool(name string, args ...string) *exec.Cmd {
	if c := os.Getenv("STAGING_PG_CONTAINER"); c != "" {
		_, _, user, pass, _ := dsnParts()
		full := append([]string{"exec", "-e", "PGPASSWORD=" + pass, c, name}, args...)
		// **ويُبدَّل المضيفُ والمنفذُ إلى ما تراه الحاويةُ من داخلها.**
		for i := 0; i < len(full); i++ {
			if full[i] == "-h" && i+1 < len(full) {
				full[i+1] = "localhost"
			}
			if full[i] == "-p" && i+1 < len(full) {
				full[i+1] = "5432"
			}
		}
		_ = user
		return exec.Command("docker", full...)
	}
	return exec.Command(name, args...)
}

func backup() error {
	host, port, user, pass, name := dsnParts()
	dir := os.Getenv("STAGING_BACKUP_DIR")
	if dir == "" {
		dir = "./backups"
	}
	if c := os.Getenv("STAGING_PG_CONTAINER"); c != "" {
		if out, err := exec.Command("docker", "exec", c, "mkdir", "-p", dir).
			CombinedOutput(); err != nil {
			return fmt.Errorf("إنشاءُ مجلَّد النسخ داخلَ الحاوية: %w · %s", err, out)
		}
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	out := fmt.Sprintf("%s/%s-%s.dump", dir, name,
		time.Now().UTC().Format("20060102T150405Z"))

	// **وصيغةٌ مضغوطةٌ مخصَّصة** (`-Fc`) — **تُستعاد انتقائيّاً وتُفحَص
	// قبل الاستعادة**، **ونصٌّ خامٌّ لا يُفحَص إلّا بقراءته كلِّه.**
	cmd := pgTool("pg_dump", "-Fc", "-h", host, "-p", port,
		"-U", user, "-d", name, "-f", out)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+pass)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump: %w", err)
	}
	// **وحين تكتب الحاويةُ الملفَّ لا يراه المضيف** — فيُسأل عنه هناك.
	size, err := outSize(out)
	if err != nil {
		return err
	}
	fi := sizeOnly(size)
	// **ونسخةٌ فارغةٌ ليست نسخة** — **وملفٌّ بحجم صفرٍ ينجح فيه
	// `pg_dump` ولا يُستعاد منه شيء.**
	if fi < 1024 {
		return fmt.Errorf("نسخةٌ حجمُها %d بايت — **وهذه ليست نسخة**", fi)
	}
	fmt.Printf("✓ نسخةٌ %s · %d بايت\n", out, fi)
	return nil
}

func sizeOnly(n int64) int64 { return n }

// outSize حجمُ ملفٍّ — **حيثما كُتب.**
func outSize(path string) (int64, error) {
	if c := os.Getenv("STAGING_PG_CONTAINER"); c != "" {
		b, err := exec.Command("docker", "exec", c, "stat", "-c", "%s", path).Output()
		if err != nil {
			return 0, fmt.Errorf("قياسُ النسخة داخلَ الحاوية: %w", err)
		}
		var n int64
		if _, err := fmt.Sscan(strings.TrimSpace(string(b)), &n); err != nil {
			return 0, err
		}
		return n, nil
	}
	fi, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

func restore(file string) error {
	host, port, user, pass, name := dsnParts()
	if _, err := outSize(file); err != nil {
		return fmt.Errorf("لا نسخةَ في %s: %w", file, err)
	}
	// **و`--clean --if-exists` تُسقط ما في الهدف ثمّ تبني** —
	// **واستعادةٌ فوق قاعدةٍ فيها صفوفٌ تخلط القديمَ بالجديد.**
	cmd := pgTool("pg_restore", "--clean", "--if-exists", "--no-owner",
		"-h", host, "-p", port, "-U", user, "-d", name, file)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+pass)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_restore: %w", err)
	}
	fmt.Printf("✓ استُعيدت %s إلى %s\n", file, name)
	return nil
}

// verify يعدّ الصفوفَ ويثبت أنّ المخطَّطَ سليم.
//
// **ولا يكفي أن ينجح `pg_restore`** (البند ٢٩) — **قاعدةٌ استُعيدت
// فارغةً تنجح أيضاً.**
func verify() error {
	ctx := context.Background()
	db, err := pool()
	if err != nil {
		return err
	}
	defer db.Close()

	var tables, migrations int
	if err := db.QueryRow(ctx,
		`SELECT count(*) FROM pg_tables WHERE schemaname = 'public'`).Scan(&tables); err != nil {
		return err
	}
	if err := db.QueryRow(ctx,
		`SELECT count(*) FROM schema_migrations`).Scan(&migrations); err != nil {
		return err
	}

	// **والامتدادُ المكانيُّ يُسأل عنه** — **قاعدةٌ بلا `PostGIS` تقبل
	// الهجراتِ حتّى أوّلِ عمودٍ جغرافيّ.**
	var postgis bool
	if err := db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'postgis')`).
		Scan(&postgis); err != nil {
		return err
	}

	counts := map[string]int{}
	for _, t := range []string{"users", "merchants", "orders", "wallet_transactions",
		"delivery_zones", "coverage_requests", "branches"} {
		var n int
		if err := db.QueryRow(ctx, `SELECT count(*) FROM `+t).Scan(&n); err != nil {
			return fmt.Errorf("عدُّ %s: %w", t, err)
		}
		counts[t] = n
	}

	fmt.Printf("جداولُ %d · هجراتٌ %d · PostGIS %v\n", tables, migrations, postgis)
	for _, t := range []string{"users", "merchants", "orders", "wallet_transactions",
		"delivery_zones", "coverage_requests", "branches"} {
		fmt.Printf("  %-20s %d\n", t, counts[t])
	}
	if tables < 50 || migrations < 100 || !postgis {
		return fmt.Errorf("مخطَّطٌ ناقص — **والاستعادةُ لم تكتمل**")
	}
	fmt.Println("✓ المخطَّطُ سليم")
	return nil
}
