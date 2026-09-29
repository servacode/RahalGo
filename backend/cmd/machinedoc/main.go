// أمرُ توليد **عقدِ آلة الحالات** — يُنادى بعد كلّ تغييرٍ في قواعد الانتقال.
//
//	cd backend && go run ./cmd/machinedoc
//
// **ويكتب ملفّين**: `docs/ORDER-STATE-MACHINE.md` (ما بين فواصلها فقط —
// **فالسردُ بيدِ كاتبه**) و`docs/testing/system/ORDER_STATE_MACHINE.json`
// (كلُّه مولَّد).
//
// **ويحرسه `TestMachineDocIsCurrent`** — فمن بدّل حدّاً ولم يُنادِ الأمرَ
// **سقط بناؤه**، ولم تشِخ وثيقةٌ صامتةً.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/servacode/rahalgo/backend/internal/machinedoc"
)

func main() {
	doc := filepath.Join("..", "docs", "ORDER-STATE-MACHINE.md")
	js := filepath.Join("..", "docs", "testing", "system", "ORDER_STATE_MACHINE.json")
	if _, err := os.Stat(doc); err != nil {
		log.Fatalf("لم تُوجد الوثيقة في %s — نادِ الأمرَ من مجلّد backend", doc)
	}
	if err := machinedoc.Write(doc, js); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✔ ORDER-STATE-MACHINE.md و ORDER_STATE_MACHINE.json وُلِّدا من المحرّك")
}
