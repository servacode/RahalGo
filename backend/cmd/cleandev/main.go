// Command cleandev ينظّف قاعدة التطوير من بيانات التجربة ويُبقي الإعداد.
//
// (طلبُ المالك ٢٠٢٦-٠٨-٠٨: «لازم ننضّف البيانات مشان نختبر على نظافة».)
//
// # ما يبقى ولماذا
//
//	app_settings       إعداداتُ المالك — اسمُ المنصة والعمولات والحدود
//	media              الشعارُ والخلفيّاتُ المرفوعة
//	categories         تصنيفاتُ المتاجر
//	platform_sections  أقسامُ السوق (٢٦ قسماً بناها المالك)
//	delivery_zones     مناطقُ التوصيل ورسومُها
//	schema_migrations  سجلُّ الترحيلات — ولا يُمسّ
//
// **وما عداها بياناتُ تجربةٍ تُمسح**: حساباتٌ ومتاجرُ وطلباتٌ وحركاتُ مالٍ
// وشكاوى وإشعاراتٌ وسجلُّ تدقيق.
//
// # والترتيبُ يُكتشف لا يُكتب
//
// الجداولُ مترابطةٌ بمفاتيحَ أجنبيّة، **وترتيبُ الحذف المكتوبُ بيدٍ ينسى
// جدولاً فيسقط.** فتُحاول كلُّها في دورات: ما يُرفض يُؤجَّل للدورة التالية،
// **وتقف حين لا تتقدّم دورةٌ كاملة.**
//
// # وكلُّه في معاملةٍ واحدة
//
// **حذفٌ ينكسر في منتصفه أسوأُ من قاعدةٍ متّسخة**: يترك متاجرَ بلا أصحاب
// وطلباتٍ بلا زبائن. **فإمّا أن يتمّ كلُّه أو لا يقع منه شيء.**
package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var keep = map[string]bool{
	"app_settings":      true,
	"media":             true,
	"categories":        true,
	"platform_sections": true,
	"delivery_zones":    true,
	"schema_migrations": true,

	// **وجداولُ مرجعيّةٌ ليست بيانات** — نُسيت أوّلَ مرّة فمُسحت:
	//
	//	roles             سبعةُ أدوارٍ يقوم عليها التخويلُ كلُّه
	//	spatial_ref_sys   ثمانيةُ آلافِ نظامِ إحداثيّاتٍ تملكها PostGIS
	//
	// **ومسحُ الثانية كلّف أكثرَ من مسحها**: إعادتُها بإعادة تثبيت الامتداد
	// أسقطت أعمدةَ الإحداثيّات من الجداول (`CASCADE`)، **فلزم إرجاعُ القاعدة
	// كلِّها من نسخةٍ احتياطيّة.**
	"roles":           true,
	"spatial_ref_sys": true,

	// **والمدنُ بيانٌ مرجعيٌّ كالمناطق** — أُضيفت بعد هذا الأمر فنُسيت.
	//
	// **و`delivery_zones` مُبقاةٌ وتشير إليها** — فمحاولةُ حذف `cities`
	// تُرفض في كلّ دورة، **ويقف الأمرُ بـ«مفاتيحُ أجنبيّةٌ متبادلة»**
	// وهي ليست متبادلةً بل مُبقًى يشير إلى محذوف. (وقع ٢٠٢٦-٠٨-٢٢.)
	"cities": true,
}

// nullFirst أعمدةٌ في الجداول الباقية تشير إلى المحذوف — تُفكّ قبل الحذف.
var nullFirst = [][2]string{
	{"app_settings", "updated_by"},
	{"media", "created_by"},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "خطأ:", err)
		os.Exit(1)
	}
}

func run() error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return fmt.Errorf("DATABASE_URL غير مضبوط")
	}
	if strings.Contains(url, "/rahalgo_test") {
		return fmt.Errorf("هذه أداةُ قاعدة التطوير — وقاعدةُ الاختبار تُبنى من جديد")
	}
	// **وممنوعةٌ في الإنتاج قطعاً** — كالبذرة. **وأداةُ حذفٍ بلا هذا الشرط
	// تنتظر خطأً واحداً في متغيّرِ بيئة.**
	if os.Getenv("APP_ENV") == "production" {
		return fmt.Errorf("ممنوع في الإنتاج (APP_ENV=production)")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()

	tables, err := allTables(ctx, pool)
	if err != nil {
		return err
	}
	var targets []string
	for _, t := range tables {
		if !keep[t] {
			targets = append(targets, t)
		}
	}
	sort.Strings(targets)

	before, err := counts(ctx, pool, tables)
	if err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, nf := range nullFirst {
		if _, err := tx.Exec(ctx,
			fmt.Sprintf("UPDATE %s SET %s = NULL WHERE %s IS NOT NULL", nf[0], nf[1], nf[1])); err != nil {
			return fmt.Errorf("فكُّ %s.%s: %w", nf[0], nf[1], err)
		}
	}

	remaining := append([]string(nil), targets...)
	for pass := 1; len(remaining) > 0; pass++ {
		var stuck []string
		progress := false
		for _, t := range remaining {
			sp := fmt.Sprintf("p%d_%s", pass, strings.ReplaceAll(t, ".", "_"))
			if _, err := tx.Exec(ctx, "SAVEPOINT "+sp); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, "DELETE FROM "+t); err != nil {
				if _, e2 := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT "+sp); e2 != nil {
					return e2
				}
				stuck = append(stuck, t)
				continue
			}
			progress = true
		}
		if !progress {
			return fmt.Errorf("توقّف الحذفُ عند: %s — **مفاتيحُ أجنبيّةٌ متبادلة**", strings.Join(stuck, ", "))
		}
		remaining = stuck
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	after, err := counts(ctx, pool, tables)
	if err != nil {
		return err
	}
	fmt.Printf("%-24s %10s %10s\n", "الجدول", "قبل", "بعد")
	for _, t := range tables {
		if before[t] == 0 && after[t] == 0 {
			continue
		}
		mark := ""
		if keep[t] {
			mark = "  ← بقي"
		}
		fmt.Printf("%-24s %10d %10d%s\n", t, before[t], after[t], mark)
	}
	return nil
}

func allTables(ctx context.Context, p *pgxpool.Pool) ([]string, error) {
	rows, err := p.Query(ctx, `
		SELECT tablename FROM pg_tables
		WHERE schemaname = 'public' ORDER BY tablename`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func counts(ctx context.Context, p *pgxpool.Pool, tables []string) (map[string]int64, error) {
	out := map[string]int64{}
	for _, t := range tables {
		var n int64
		if err := p.QueryRow(ctx, "SELECT count(*) FROM "+t).Scan(&n); err != nil {
			if err == pgx.ErrNoRows {
				continue
			}
			return nil, err
		}
		out[t] = n
	}
	return out, nil
}
