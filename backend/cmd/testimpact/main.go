// testimpact محرّكُ أثر التغيير — **`P-9`.**
//
//	go run ./cmd/testimpact                      شجرةُ العمل
//	go run ./cmd/testimpact -base HEAD~1         مدىً في git
//	go run ./cmd/testimpact -files a.go,b.go     ملفّاتٌ صريحة
//	go run ./cmd/testimpact -json                الصورةُ الآليّةُ فقط
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/impact"
)

func main() {
	base := flag.String("base", "", "التزامُ الأساس")
	head := flag.String("head", "", "التزامُ الرأس")
	files := flag.String("files", "", "ملفّاتٌ صريحةٌ مفصولةٌ بفاصلة")
	out := flag.String("out", "", "مسارُ الصورة الآليّة")
	jsonOnly := flag.Bool("json", false, "الصورةُ الآليّةُ إلى الخرج")
	flag.Parse()

	root, err := repoRoot()
	if err != nil {
		die(err)
	}
	eng, err := impact.Load(root)
	if err != nil {
		die(err)
	}

	var explicit []string
	if *files != "" {
		explicit = strings.Split(*files, ",")
	}
	src, changed, err := impact.ChangedFrom(root, *base, *head, explicit)
	if err != nil {
		die(err)
	}

	started := time.Now()
	res := eng.Analyze(src, *base, *head, changed)
	elapsed := time.Since(started)

	b, _ := json.MarshalIndent(map[string]any{
		"result": res, "elapsed_ms": elapsed.Milliseconds(),
	}, "", "  ")

	if *jsonOnly {
		fmt.Println(string(b))
		return
	}
	fmt.Print(res.Human())
	fmt.Printf("\nELAPSED = %s\n", elapsed.Round(time.Millisecond))

	p := *out
	if p == "" {
		p = filepath.Join(root, "docs/testing/system/CHANGE_IMPACT.json")
	}
	if err := os.WriteFile(p, append(b, '\n'), 0o644); err != nil {
		die(err)
	}
	fmt.Println("كُتب", p)
}

func repoRoot() (string, error) {
	d, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(d, "docs/testing/system/TEST_TRUTH.json")); err == nil {
			return d, nil
		}
		d = filepath.Dir(d)
	}
	return "", fmt.Errorf("لم أجد جذرَ المستودع")
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "testimpact:", err)
	os.Exit(2)
}
