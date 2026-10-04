package server

import "time"

// نصوصُ قسم «النقد والصندوق» — عربيّةٌ مكتوبةٌ هنا لا في أوامر الصدفة.

var cashExportHead = []string{
	"السائق", "الهاتف", "بذمته", "نقد طلبات مفتوحة", "المجموع",
	"السقف", "أقدم مبلغ باق (دمشق)", "آخر استلام (دمشق)", "الدوام",
}

const (
	cashExportOnShift  = "على الدوام"
	cashExportOffShift = "خارج الدوام"
)

// damascusLoc توقيتُ دمشق — ومن غاب جدولُ المناطق عنده لا يفشل التصدير لأجله.
func damascusLoc() *time.Location {
	if loc, err := time.LoadLocation("Asia/Damascus"); err == nil {
		return loc
	}
	return time.FixedZone("Asia/Damascus", 3*60*60)
}

// cashSettledBody **نصُّ إشعار السائق بعد استلام نقده** — بفواصل الآلاف والعملة.
//
// كان «50000 — والباقي بذمّتك 106650»: أرقامٌ خامٌ في إشعارٍ عابر، ومن قرأها
// بلمحةٍ أخطأ في قدرها.
func cashSettledBody(amount, held int64, note string) string {
	body := "استُلم منك " + fmtMoneyAr(amount) + " — والباقي بذمّتك " + fmtMoneyAr(held)
	if note != "" {
		body += " · " + note
	}
	return body
}
