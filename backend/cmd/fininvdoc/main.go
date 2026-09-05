// fininvdoc يكتب الصورَ الآليّةَ لمنظومة الاختبار.
//
// **ولا Markdown مصدراً وحيداً** (`P-4` البند ٢٨ · `P-5` البند ٣٣):
// **`P-9` يسأل «أيُّ ثابتٍ أو سباقٍ يمسّه هذا الملفّ؟» و`P-10` يسأل «أكلُّ
// حرِجٍ له اختبارٌ شُغّل؟»** — وسؤالان كهذان لا يُجابان من نصٍّ منسَّق.
//
//	go run ./cmd/fininvdoc
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/servacode/rahalgo/backend/internal/androidmap"
	"github.com/servacode/rahalgo/backend/internal/eventmap"
	"github.com/servacode/rahalgo/backend/internal/failmap"
	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/racemap"
)

func main() {
	write("../docs/testing/system/FINANCIAL_INVARIANTS.json", fininv.Snapshot())
	write("../docs/testing/system/CONCURRENCY_MATRIX.json", racemap.Snapshot())
	write("../docs/testing/system/FAILURE_INJECTION_MATRIX.json", failmap.Snapshot())

	// **وعقودُ الأحداث تُخرَج مع ما استُخرج من الشيفرة** — فلا رقمَ يُثبَّت بيد.
	r := eventmap.Root(".")
	sites, err := r.Sites()
	if err != nil {
		fmt.Fprintln(os.Stderr, "مواضعُ الإطلاق:", err)
		os.Exit(1)
	}
	pubs, err := r.Publishers()
	if err != nil {
		fmt.Fprintln(os.Stderr, "البواثّ:", err)
		os.Exit(1)
	}
	write("../docs/testing/system/EVENT_CONTRACT_MATRIX.json", eventmap.Snapshot(sites, pubs))

	snap := androidmap.Snapshot()
	write("../docs/testing/system/ANDROID_DEVICE_MATRIX.json", map[string]any{
		"devices": snap["devices"], "counts": snap["counts"],
	})
	write("../docs/testing/system/ANDROID_TEST_MATRIX.json", map[string]any{
		"cases": snap["cases"], "counts": snap["counts"],
	})
}

func write(path string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, path, err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, path, err)
		os.Exit(1)
	}
	fmt.Println("كُتب", path)
}
