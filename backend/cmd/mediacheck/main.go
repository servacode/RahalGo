// Command mediacheck **يقارن ما في القاعدة بما على القرص.**
//
// (كشفه جردُ لوحة الزبون 2026-08-09: سبعُ صورٍ مكسورةٍ في ثلاث صفحات —
//
//	والسببُ مجلّدا رفعٍ لا واحد.)
//
// # لماذا أداةٌ لا اختبار
//
// **الاختبارُ يفحص الشيفرة، وهذا يفحص القرص** — ولا يُعرف مجلّدُ الرفع إلّا
// من بيئة التشغيل. **واختبارٌ يخمّن مساراً يمرّ في جهازٍ ويسقط في آخر.**
//
// **وهو من عائلة `moneycheck`**: يُنادى قبل النشر وبعده، **ويردّ رمزَ خروجٍ
// غيرَ صفرٍ إن وجد خللاً** فيُوقف خطَّ النشر.
//
// # وما يُقاس
//
// **صفٌّ بلا ملفّ** — صورةٌ تُطلب فتردّ 404: **العطبُ الذي وقع.**
//
// **وملفٌّ بلا صفّ** — رفعٌ لم يكتمل أو صفٌّ حُذف: **لا يكسر شاشةً، لكنّه
// يملأ القرصَ بما لا يُقرأ.** فيُقال ولا يُسقط.
//
// # ولا يُحذف شيء
//
// **يقرأ ولا يكتب.** والحذفُ قرارُ مالكٍ لا قرارُ أداة — **وملفٌّ يبدو يتيماً
// اليومَ قد يكون صفُّه في هجرةٍ لم تُشغَّل بعد.**
package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("mediacheck: DATABASE_URL مطلوب")
		os.Exit(2)
	}
	dir := os.Getenv("UPLOADS_DIR")
	if dir == "" {
		dir = "./uploads"
	}
	abs, err := filepath.Abs(dir)
	if err == nil {
		dir = abs
	}

	ctx := context.Background()
	pg, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Println("mediacheck: تعذّر الاتّصال بالقاعدة:", err)
		os.Exit(2)
	}
	defer pg.Close()

	// ── ما في القاعدة ────────────────────────────────────────────────
	//
	// **والمصغَّرةُ صفٌّ في العمود نفسِه** — `thumb_path` ملفٌّ ثانٍ لكلّ
	// وسيط، **وغيابُه يكسر البطاقاتِ وإن سلمت الصورةُ الأصليّة.**
	rows, err := pg.Query(ctx, `
		SELECT path, COALESCE(thumb_path, '') FROM media WHERE path <> ''`)
	if err != nil {
		fmt.Println("mediacheck: تعذّرت القراءة:", err)
		os.Exit(2)
	}
	want := map[string]bool{}
	for rows.Next() {
		var p, t string
		if err := rows.Scan(&p, &t); err != nil {
			fmt.Println("mediacheck:", err)
			os.Exit(2)
		}
		want[filepath.ToSlash(p)] = true
		if t != "" {
			want[filepath.ToSlash(t)] = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		fmt.Println("mediacheck:", err)
		os.Exit(2)
	}

	// ── وما على القرص ────────────────────────────────────────────────
	have := map[string]bool{}
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil //nolint:nilerr // مجلّدٌ غيرُ مقروءٍ يُتخطّى ولا يُسقط الجرد
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return nil
		}
		have[filepath.ToSlash(rel)] = true
		return nil
	})

	missing := diff(want, have)
	orphan := diff(have, want)

	fmt.Printf("مجلّدُ الوسائط: %s\n", dir)
	fmt.Printf("في القاعدة: %d ملفّاً · على القرص: %d\n\n", len(want), len(have))

	if len(missing) > 0 {
		fmt.Printf("✗ %d ملفّاً في القاعدة ولا وجودَ له على القرص:\n", len(missing))
		fmt.Println("   الأثرُ: صورةٌ تُطلب فتردّ 404 — بطاقةٌ مكسورةٌ أو شعارٌ غائب.")
		for _, p := range head(missing, 12) {
			fmt.Println("   ·", p)
		}
		if len(missing) > 12 {
			fmt.Printf("   … و%d غيرُها\n", len(missing)-12)
		}
		fmt.Println()
	}

	if len(orphan) > 0 {
		fmt.Printf("• %d ملفّاً على القرص ولا صفَّ له في القاعدة — لا يكسر شيئاً، ويشغل مكاناً.\n\n", len(orphan))
	}

	if len(missing) > 0 {
		fmt.Println("**خللٌ في الوسائط.** وأوّلُ ما يُسأل عنه: أَمجلّدُ الرفع هو نفسُه الذي أقلع منه المحرّك؟")
		os.Exit(1)
	}
	fmt.Println("الوسائطُ متماسكة — كلُّ صفٍّ له ملفّه.")
}

func diff(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func head(xs []string, n int) []string {
	if len(xs) <= n {
		return xs
	}
	return xs[:n]
}
