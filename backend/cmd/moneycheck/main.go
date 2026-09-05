// moneycheck يفحص دفترَ المال على قاعدةٍ حيّة.
//
// # ما تبدّل في P-4
//
// **كانت الفحوصُ الثلاثةَ عشرَ مكتوبةً في هذا الملفّ.** فصارت في
// `internal/fininv` — **وهذا الأمرُ قشرةٌ تناديها**، **واختباراتُ
// `internal/qa` تنادي الحزمةَ نفسَها.**
//
// **ولا استعلامَ غُيّر في النقل** — نُقلت بنصّها ومُنحت معرّفاتٍ (`FI-02.a`)
// ونُسبت إلى عوائلها.
//
// **ولماذا؟** لأنّ فحصاً في `cmd/` لا يستطيع اختبارٌ أن يناديَه، **فكانت
// الاختباراتُ تكتب استعلاماتِها بيدها** — **وسؤالان يفترقان في الصياغة
// يفترقان في الجواب.**
//
//	go run ./cmd/moneycheck                 كلُّ فحوصِ التشغيل
//	go run ./cmd/moneycheck FI-06 FI-10     عائلتان
//	go run ./cmd/moneycheck -list           ماذا يفحص ولا يشغّل
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/fininv"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && (args[0] == "-list" || args[0] == "--list") {
		list()
		return
	}

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL غير مضبوط")
		os.Exit(2)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "اتّصال:", err)
		os.Exit(2)
	}
	defer pool.Close()

	// **وأنواعُ القيد تُفحَص قبل الأرقام** — فنوعٌ بلا عقدٍ يعني أنّ
	// ما بعده مقيسٌ بمسطرةٍ ناقصة.
	bad := checkKinds(ctx, pool)

	selected := fininv.Select(args...)
	var run []fininv.Check
	for _, c := range selected {
		if len(args) > 0 || c.Ops {
			run = append(run, c)
		}
	}

	for _, c := range run {
		if c.Status != fininv.ProvableNow {
			fmt.Printf("… %-10s %s — %s\n", c.ID, c.Name, c.Status)
			continue
		}
		vs, err := fininv.Run(ctx, pool, c.ID)
		if err != nil {
			fmt.Printf("✗ %-10s %s — تعذّر الفحص: %v\n", c.ID, c.Name, err)
			bad++
			continue
		}
		if len(vs) == 0 {
			fmt.Printf("✓ %-10s %s\n", c.ID, c.Name)
			continue
		}
		bad++
		v := vs[0]
		fmt.Printf("✗ %-10s %s — %d صفّاً\n", c.ID, c.Name, len(v.Rows))
		fmt.Printf("   %s\n", strip(c.Why))
		fmt.Printf("   %s\n", strings.Join(v.Cols, " | "))
		for i, row := range v.Rows {
			if i >= 10 {
				fmt.Printf("   ... و%d غيرُها\n", len(v.Rows)-10)
				break
			}
			cells := make([]string, len(row))
			for j, cell := range row {
				cells[j] = fmt.Sprint(cell)
			}
			fmt.Printf("   %s\n", strings.Join(cells, " | "))
		}
	}

	fmt.Printf("\n%d فحصاً · %d خرقاً\n", len(run), bad)
	if bad > 0 {
		os.Exit(1)
	}
}

// checkKinds حارسُ أنواع القيد — **من القاعدة الحيّة لا من هجرة.**
func checkKinds(ctx context.Context, pool *pgxpool.Pool) int {
	schema, err := fininv.SchemaKinds(ctx, pool)
	if err != nil {
		fmt.Printf("✗ %-10s أنواعُ القيد — %v\n", "KINDS", err)
		return 1
	}
	missing, stale := fininv.KindDrift(schema)
	if len(missing) == 0 && len(stale) == 0 {
		fmt.Printf("✓ %-10s أنواعُ القيد — %d نوعاً، لكلٍّ عقد\n", "KINDS", len(schema))
		return 0
	}
	if len(missing) > 0 {
		fmt.Printf("✗ %-10s أنواعٌ بلا عقدٍ ماليّ: %s\n", "KINDS", strings.Join(missing, ", "))
		fmt.Println("   NEW LEDGER KIND WITHOUT FINANCIAL CONTRACT = FAIL")
		fmt.Println("   يُملأ في internal/fininv/kinds.go: من يكتبه · دلالتُه · ثوابتُه · أيشترط مرجعاً")
	}
	if len(stale) > 0 {
		fmt.Printf("✗ %-10s عقودٌ لأنواعٍ لم تعد في القاعدة: %s\n", "KINDS", strings.Join(stale, ", "))
	}
	return 1
}

func list() {
	e := fininv.Snapshot()
	for _, f := range e.Families {
		fmt.Printf("%s\n", f.ID)
		for _, id := range f.Checks {
			for _, c := range e.Checks {
				if c.ID == id {
					mark := " "
					if c.Ops {
						mark = "•"
					}
					fmt.Printf("  %s %-10s %-20s %s\n", mark, c.ID, c.Status, c.Name)
				}
			}
		}
	}
	fmt.Printf("\n%d عائلةً · %d فحصاً · %d يُثبَت الآن · %d نوعَ قيدٍ متعاقَداً عليه\n",
		e.Counts.Families, e.Counts.Checks, e.Counts.ProvableNow, e.Counts.Kinds)
	fmt.Println("(• يُشغَّل على قاعدةِ تشغيل)")
}

// strip يُزيل نجومَ التوكيد — فالطرفيّةُ لا تعرضها.
func strip(s string) string { return strings.ReplaceAll(s, "**", "") }
