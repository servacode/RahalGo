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

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/racemap"
)

func main() {
	write("../docs/testing/system/FINANCIAL_INVARIANTS.json", fininv.Snapshot())
	write("../docs/testing/system/CONCURRENCY_MATRIX.json", racemap.Snapshot())
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
