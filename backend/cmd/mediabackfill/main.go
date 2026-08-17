// **يعالج الصورَ المرفوعةَ قبل أن تُولَّد اللمحاتُ والمقاسات.**
//
// (طلبُ المالك ٢٠٢٦-٠٨-١٧: «نريد حلَّ مشكلة الصور… أنا أرفع أيَّ نوعِ
//
//	صورةٍ يجب أن يُعدَّل برمجيّاً».)
//
// # ولماذا أداةٌ لا «ارفعها من جديد»
//
// **المعالجةُ تقع عند الرفع** — وصورُه السبعُ وخلفيّتُه رُفعت قبلها.
// **وطلبُ إعادة الرفع يحمّل صاحبَ المنصّة عملاً تفعله آلة**، ويضيع منه
// ترتيبُ اللافتات وأيُّها مفعّلة.
//
// # وما لا تفعله
//
// **لا تعيد ترميزَ الأصل** — ذاك يفقد جودةً في كلّ تشغيلة، **ومن شغّلها
// مرّتين أفسد صوره.** تقرأ الأصلَ وتكتب ما ينقص: اللمحةَ والنسخَ.
//
// **وتتخطّى ما اكتمل** — فتُشغَّل مرّةً أو عشراً بالنتيجة نفسِها.
package main

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/media"
)

func main() {
	ctx := context.Background()
	dsn := os.Getenv("DATABASE_URL")
	dir := os.Getenv("UPLOADS_DIR")
	if dir == "" {
		dir = "./uploads"
	}
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL مطلوب")
		os.Exit(2)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "تعذّر الاتّصال:", err)
		os.Exit(1)
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, `
		SELECT id, path FROM media
		WHERE blur = '' OR sizes = false
		ORDER BY created_at`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "تعذّرت القراءة:", err)
		os.Exit(1)
	}
	type job struct{ id, path string }
	var jobs []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.id, &j.path); err == nil {
			jobs = append(jobs, j)
		}
	}
	rows.Close()

	var done, missing, failed int
	for _, j := range jobs {
		full := filepath.Join(dir, filepath.FromSlash(j.path))
		f, err := os.Open(full)
		if err != nil {
			// **وملفٌّ ذهب من القرص وبقي صفُّه** — يُعدّ ولا يُوقف الباقي.
			missing++
			continue
		}
		src, _, err := image.Decode(f)
		_ = f.Close()
		if err != nil {
			failed++
			continue
		}
		blur, sizes, err := media.Derive(src, full)
		if err != nil {
			failed++
			continue
		}
		if _, err := pool.Exec(ctx,
			`UPDATE media SET blur = $2, sizes = $3 WHERE id = $1`,
			j.id, blur, sizes); err != nil {
			failed++
			continue
		}
		done++
	}
	fmt.Printf("عولجت %d · ملفّاتٌ مفقودة %d · تعذّرت %d · من أصل %d\n",
		done, missing, failed, len(jobs))
	if strings.TrimSpace(os.Getenv("STRICT")) != "" && failed > 0 {
		os.Exit(1)
	}
}
