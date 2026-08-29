// **يولّد بصمةَ كلمة مرورٍ بأداة المشروع نفسِها.**
//
// **ولا تُولَّد بأداةٍ خارجيّة** — **وبصمةٌ بمعاملِ كلفةٍ مختلفٍ تُقبل
// في الدخول وتُبطئه**، أو تُرفض إن اختلفت الصيغة.
package main

import (
	"fmt"
	"os"

	"github.com/servacode/rahalgo/backend/internal/auth"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: hashgen <password>")
		os.Exit(2)
	}
	h, err := auth.HashPassword(os.Args[1])
	if err != nil {
		panic(err)
	}
	fmt.Println(h)
}
