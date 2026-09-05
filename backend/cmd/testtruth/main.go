// مولِّدُ حقيقة الاختبار — **يُشغَّل ويُكتب، ولا يُحرَّر مخرَجُه بيد.**
//
//	go run ./cmd/testtruth          # يكتب
//	go run ./cmd/testtruth -check   # يقارن ولا يكتب — وهو ما يفعله الحارس
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/servacode/rahalgo/backend/internal/testtruth"
)

func main() {
	check := flag.Bool("check", false, "قارِنْ ولا تكتب — يسقط عند الانحراف")
	flag.Parse()

	backend, docs, out, err := paths()
	if err != nil {
		fail(err)
	}

	t, err := testtruth.Build(backend, docs)
	if err != nil {
		fail(err)
	}

	jsonBytes, err := t.JSON()
	if err != nil {
		fail(err)
	}
	report := []byte(t.Report())

	jsonPath := filepath.Join(out, "TEST_TRUTH.json")
	mdPath := filepath.Join(out, "TEST_TRUTH.md")

	if *check {
		drift := 0
		for _, f := range []struct {
			path string
			want []byte
		}{{jsonPath, jsonBytes}, {mdPath, report}} {
			got, rerr := os.ReadFile(f.path)
			if rerr != nil {
				fmt.Fprintf(os.Stderr, "✗ %s غيرُ موجود — شغّلِ المولِّد.\n", filepath.Base(f.path))
				drift++
				continue
			}
			if !bytes.Equal(normalize(got), normalize(f.want)) {
				fmt.Fprintf(os.Stderr, "✗ %s شاخ — أعِد التوليد.\n", filepath.Base(f.path))
				drift++
			}
		}
		// **المراجعُ الشائخةُ سقوطٌ لا تقرير.**
		if len(t.Stale) > 0 {
			fmt.Fprintf(os.Stderr, "✗ %d مرجعاً شائخاً:\n", len(t.Stale))
			for _, s := range t.Stale {
				fmt.Fprintf(os.Stderr, "    %s\n", s)
			}
			drift++
		}
		if drift > 0 {
			os.Exit(1)
		}
		fmt.Printf("✓ حقيقةُ الاختبار حاضرة — %d تدفّقاً · %d عيباً · %d خطراً · %d فجوةً · %d إعداداً · %d اختباراً.\n",
			len(t.Flows), len(t.Defects), len(t.Risks), len(t.Gaps), len(t.Settings), len(t.Tests))
		return
	}

	if err := os.MkdirAll(out, 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(jsonPath, jsonBytes, 0o644); err != nil {
		fail(err)
	}
	if err := os.WriteFile(mdPath, report, 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("كُتبت حقيقةُ الاختبار — %d اختباراً · %d تدفّقاً · %d عيباً.\n",
		len(t.Tests), len(t.Flows), len(t.Defects))
	if len(t.Stale) > 0 {
		fmt.Printf("⚠️ %d مرجعاً شائخاً — انظر التقرير.\n", len(t.Stale))
	}
}

// normalize **نهاياتُ الأسطر لا تُعدّ انحرافاً** — ويندوز يبدّلها.
func normalize(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}

// paths **جذورُ المشروع** — تُشتقّ من مجلَّد التشغيل صعوداً.
func paths() (backend, docs, out string, err error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", "", "", err
	}
	dir := wd
	for i := 0; i < 6; i++ {
		if _, e := os.Stat(filepath.Join(dir, "go.mod")); e == nil {
			backend = dir
			break
		}
		dir = filepath.Dir(dir)
	}
	if backend == "" {
		return "", "", "", fmt.Errorf("لم أجد go.mod صعوداً من %s", wd)
	}
	root := filepath.Dir(backend)
	docs = filepath.Join(root, "docs")
	if _, e := os.Stat(docs); e != nil {
		return "", "", "", fmt.Errorf("لم أجد docs/ عند %s", docs)
	}
	out = filepath.Join(docs, "testing", "system")
	return backend, docs, out, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "✗", err)
	os.Exit(1)
}
