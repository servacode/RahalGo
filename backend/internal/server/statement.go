package server

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// مدى كشف الحساب من متغيّرات الاستعلام: `from` و`to` بصيغة YYYY-MM-DD.
//
// **بتوقيت دمشق لا UTC**: من يطلب كشف تموز يقصد تموزَ عنده. حساب الحدود بـUTC
// يزحزح اليوم ساعتين فتقع حركةُ أول الشهر في الشهر السابق — خطأٌ في كشفٍ مالي
// لا في عرضٍ تجميلي. و`to` **شامل ليومه**: من كتب 31/7 يريد يوم 31 كاملاً،
// فنجعل الحدّ الأعلى بداية اليوم التالي.
func statementRange(r *http.Request) wallet.StatementRange {
	loc, err := time.LoadLocation("Asia/Damascus")
	if err != nil {
		loc = time.UTC // بلا قاعدة بيانات مناطق زمنية: لا نفشل الطلب لأجل الإزاحة
	}
	q := r.URL.Query()
	var rng wallet.StatementRange
	if t, err := time.ParseInLocation("2006-01-02", q.Get("from"), loc); err == nil {
		rng.From = &t
	}
	if t, err := time.ParseInLocation("2006-01-02", q.Get("to"), loc); err == nil {
		end := t.AddDate(0, 0, 1)
		rng.To = &end
	}
	return rng
}
