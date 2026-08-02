// أمرُ توليد **مصدر الحقيقة الواحد** — يُنادى بعد كلّ تغييرٍ في قواعد المحرّك.
//
// **ولا يُنشئ الوثيقة**: يملأ ما بين الفواصل فيها. **ووثيقةٌ تُكتب كلُّها آلياً
// لا تحمل قرار مالكٍ ولا سبباً** — إنّما جداول.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/servacode/rahalgo/backend/internal/truthdoc"
)

func main() {
	// **يُنادى من جذر `backend`** — والمسارُ في الحزمة نسبيٌّ لموضع الاختبار.
	path := filepath.Join("..", "docs", "TRUTH.md")
	if _, err := os.Stat(path); err != nil {
		log.Fatalf("لم تُوجد الوثيقة في %s — نادِ الأمرَ من مجلّد backend", path)
	}
	if err := truthdoc.Write(path); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✔ TRUTH.md وُلِّدت من المحرّك")
}
