package server

// **ترويساتُ الأمان — طبقةٌ ثانيةٌ حين تسقط الأولى.**
//
// (كشفه فحصُ المشروع ٢٠٢٦-٠٨-٠٧، وأُصلح بقرار المالك: «أكمل».)
//
// # لماذا
//
// **رمزُ الجلسة في `localStorage`** — فأيُّ ثغرةِ نصٍّ في الواجهة تسرقه.
// ولم أجد في المشروع `dangerouslySetInnerHTML` ولا `eval`، **فالخطرُ اليومَ
// نظريّ** — والترويساتُ هي ما يبقى حين يُكتب سطرٌ جديدٌ غداً.
//
//	nosniff       المتصفّحُ لا يخمّن نوعَ الملفّ: صورةٌ مرفوعةٌ فيها HTML
//	              تُنفَّذ إن خمّن. **والوسائطُ عندنا مرفوعةٌ من الناس.**
//	DENY          لا تُوضع لوحةُ الإدارة في إطارٍ داخل موقعٍ آخر
//	              (`clickjacking`): يُعرض زرٌّ شفّافٌ فوق زرِّ الحذف.
//	no-referrer   لا يُسرَّب مسارٌ فيه رقمُ طلبٍ أو معرّفُ حساب إلى موقعٍ
//	              خارجيٍّ في `Referer`.
//	Permissions   لا كاميرا ولا ميكروفون — والموقعُ يحتاج الموضعَ وحدَه.
//
// # وHSTS ليست هنا
//
// **تُضاف عند النشر خلف TLS** — وترويسةٌ تفرض HTTPS على `localhost`
// **تكسر التطويرَ كلَّه** ولا تحمي شيئاً.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	want := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := securityHeaders(next)

	// **على كلّ ردٍّ** — لا على الناجح وحدَه.
	for _, path := range []string{"/healthz", "/api/v1/public/platform", "/media/x.jpg"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		for k, v := range want {
			if got := rec.Header().Get(k); got != v {
				t.Fatalf("%s: الترويسةُ %s = %q — والمنتظَر %q.\n"+
					"**ورمزُ الجلسة في `localStorage`**، فهذه الطبقةُ الثانية.",
					path, k, got, v)
			}
		}
		if rec.Header().Get("Permissions-Policy") == "" {
			t.Fatalf("%s: لا `Permissions-Policy` — **والكاميرا والميكروفون مفتوحان لكلّ إطار.**", path)
		}
	}

	// **ولا HSTS في التطوير** — تفرض HTTPS على `localhost` فتكسره.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Fatalf("HSTS مضبوطةٌ بلا TLS — **تكسر التطويرَ ولا تحمي شيئاً.**")
	}
}
