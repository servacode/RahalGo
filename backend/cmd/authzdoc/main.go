// أمرُ توليد **عقدِ التخويل** — يُنادى بعد كلّ تغييرٍ في القدرات أو السياسة
// أو الأفعال الحسّاسة.
//
//	cd backend && go run ./cmd/authzdoc
//
// **ويكتب ملفّين**: `docs/SECURITY-CAPABILITY-CONTRACT.md` (ما بين فواصلها)
// و`docs/testing/system/AUTHZ_CONTRACT.json` (كلُّه مولَّد).
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/servacode/rahalgo/backend/internal/authzdoc"
)

func main() {
	doc := filepath.Join("..", "docs", "SECURITY-CAPABILITY-CONTRACT.md")
	js := filepath.Join("..", "docs", "testing", "system", "AUTHZ_CONTRACT.json")
	if _, err := os.Stat(doc); err != nil {
		log.Fatalf("لم تُوجد الوثيقة في %s — نادِ الأمرَ من مجلّد backend", doc)
	}
	if err := authzdoc.Write(doc, js); err != nil {
		log.Fatal(err)
	}
	// **وصلاحيّاتُ الواجهة من الجدول نفسِه** (قرارُ المالك ٢٠٢٦-١٠-٠٤).
	web := filepath.Join("..", "web", "apps", "rahalgo", "src", "lib", "adminPolicy.gen.ts")
	if err := authzdoc.WriteWebPolicy(web); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✔ SECURITY-CAPABILITY-CONTRACT.md و AUTHZ_CONTRACT.json وُلِّدا")
}
