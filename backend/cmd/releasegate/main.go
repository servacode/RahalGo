// أمرُ بوّابة الإطلاق — **`P-10` البند ٤٤.**
//
// # رموزُ الخروج — موثَّقةٌ ومُميَّزة
//
//	0  READY FOR PRODUCTION = YES
//	1  NOT READY — موانعُ قائمة
//	2  GATE INTERNAL ERROR — دليلٌ ناقصٌ أو غيرُ مقروء
//
// **والفرقُ بين ١ و٢ جوهريّ**: **«غيرُ جاهز» حكمٌ صحيح، و«عطبُ بوّابة»
// لا حكمَ فيه** — **ومن خلطهما نشر على عمى.**
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/servacode/rahalgo/backend/internal/gate"
)

const (
	exitReady    = 0
	exitNotReady = 1
	exitError    = 2
)

func main() {
	// **ومخرَجُ التشغيل خارجَ ما يُتتبَّع** — `system/generated/`.
	//
	// # لماذا نُقل
	//
	// **كان يُكتب في `system/` المتتبَّع فيُوسّخ الشجرةَ بكلّ تشغيل**:
	// يحمل `run_id` عشوائيّاً و`generated_at` من الساعة، **فلا ينفع
	// «لا تكتب إن لم يتبدّل المحتوى»** — المحتوى يتبدّل حتماً.
	//
	// **ومرشَّحُ إصدارٍ نظيفٌ صار يبدو `dirty` بسبب أثرٍ ولّدته
	// البوّابةُ نفسُها** — **وهي تقيس النظافةَ وتنقضها في النَّفَس ذاته.**
	//
	// **وليس هذا حقيقةً بل تقريرَ تشغيل**: `TEST_TRUTH.json` والمصفوفاتُ
	// السبعُ هي مُدخلاتُ البوّابة **وتبقى متتبَّعةً بحارس شيخوختها**.
	// **ولا شيءَ يقرأ هذا الملفَّ أصلاً** (مُقاس).
	out := flag.String("out", "../docs/testing/system/generated/RELEASE_GATE.json", "مسارُ الحكم الآليّ")
	quiet := flag.Bool("json", false, "الحكمُ الآليُّ فقط بلا تقريرٍ بشريّ")
	flag.Parse()

	backend, err := os.Getwd()
	if err != nil {
		die(err)
	}
	root := filepath.Dir(backend)
	docs := filepath.Join(root, "docs")

	ev, err := gate.LoadEvidence(backend, docs)
	if err != nil {
		die(fmt.Errorf("دليلٌ ناقص: %w", err))
	}
	cand, err := gate.BuildCandidate(root)
	if err != nil {
		die(fmt.Errorf("هويّةُ المرشَّح: %w", err))
	}

	d := gate.Decide(cand, ev, gate.Waivers)

	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		die(err)
	}
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		die(err)
	}

	if !*quiet {
		fmt.Print(d.Human())
		fmt.Printf("\nكُتب %s\n", *out)
	}
	if d.Ready {
		os.Exit(exitReady)
	}
	os.Exit(exitNotReady)
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "GATE INTERNAL ERROR: %v\n", err)
	os.Exit(exitError)
}
