package server

import (
	"context"
	"net/http"
	"strings"
)

// countOpen **يزيد عدّادَ اليوم بواحد — بلا هويّةٍ ولا انتظار.**
//
// (طلبُ المالك 2026-08-25: «أضف بلوحة التحكم خيار لمعرفة عدد الزوار».)
//
// # ولا يُبطئ نداءَ الزبون
//
// **والزيادةُ كتابةٌ في قاعدة، ونداءُ الصفحة الرئيسة أكثرُ النداءات
// وقوعا.** فلو انتظرها الردُّ أضاف مللي ثانية إلى كلّ فتحة.
//
// **فتقع في خيطٍ مستقلٍّ بسياقٍ مستقلّ** — وسياقُ الطلب يموت مع الردّ،
// **ومن ورّثه للخيط كتب في سياقٍ ملغى فضاع العدّ صامتا.**
//
// # وعطبُ العدّاد لا يُسقط صفحة
//
// **ولا يُرجَع خطأ**: من عجز عن العدّ عجز عن رقمٍ في لوحة، **ومن أسقط
// الصفحة لأجل ذلك أسقط المتجر كلَّه لأجل إحصاء.**
func (s *Server) countOpen(r *http.Request) {
	client := strings.TrimSpace(r.Header.Get(clientKindHeader))
	if client == "" {
		client = "unknown"
	}
	if len(client) > 40 {
		client = client[:40]
	}
	go func() {
		// **بتوقيت دمشق** — انظر الترحيل 0119.
		_, _ = s.pg.Exec(context.Background(), `
			INSERT INTO app_opens_daily (day, client, opens)
			VALUES (
			  (now() AT TIME ZONE 'Asia/Damascus')::date,
			  $1, 1
			)
			ON CONFLICT (day, client)
			DO UPDATE SET opens = app_opens_daily.opens + 1
		`, client)
	}()
}
