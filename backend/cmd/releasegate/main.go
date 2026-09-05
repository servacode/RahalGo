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
	out := flag.String("out", "../docs/testing/system/RELEASE_GATE.json", "مسارُ الحكم الآليّ")
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
