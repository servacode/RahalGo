// apidoc **يُولّد عقدَ الـAPI من الشيفرة** — انظر `internal/apidoc`.
//
// يُشغَّل من مجلّد `backend`:  go run ./cmd/apidoc
package main

import (
	"fmt"
	"os"

	"github.com/servacode/rahalgo/backend/internal/apidoc"
)

func main() {
	// **الجذرُ فوق `backend`** — العقدُ يقرؤه الويبُ والتطبيقُ أيضاً،
	// **فموضعُه المستودعُ لا حزمةُ المحرّك.**
	if err := apidoc.Write("..", "internal/server"); err != nil {
		fmt.Fprintln(os.Stderr, "تعذّر توليدُ العقد:", err)
		os.Exit(1)
	}
	c, _ := apidoc.Build("internal/server")
	known := 0
	for _, e := range c.Endpoints {
		for _, f := range e.Response {
			if f.Type != "unknown" {
				known++
				break
			}
		}
	}
	fmt.Printf("عقدُ الـAPI: %d نقطةً · %d منها بحقول ردٍّ معروفة → %s\n",
		len(c.Endpoints), known, apidoc.Path)
}
