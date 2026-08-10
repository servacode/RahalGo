package server

// **نطاقاتُ المتصفّح — ولا يُفتح البابُ بالنسيان.**
//
// # ولماذا حارسٌ لهذا
//
// **قائمةٌ فارغةٌ تعني «بلا قيد» في مكتبة CORS لا «بلا سماح».** وقِيس
// (٢٠٢٦-٠٨-١٠) على صورةٍ تعمل: أُقلع المحرّكُ في الإنتاج بلا `WEB_ORIGINS`
// **فردّ `Access-Control-Allow-Origin: *` لكلّ نطاقٍ سأل** — أيُّ موقعٍ في
// الشبكة ينادي المحرّكَ بجلسة زائره.
//
// **ونسيانُه وارد**: قيمتُه لا تُعرف قبل أوّل نشرةٍ للويب، **فيُرفع المحرّكُ
// أوّلاً ويُملأ لاحقاً** — وبينهما بابٌ مشرَع.
//
// **والتطويرُ يبقى مفتوحاً للمنافذ المحلّيّة** — ولو ضاق لَما عمل جهازُ أحد.

import (
	"log/slog"
	"os"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/config"
)

func srvWith(env string, origins []string) *Server {
	return &Server{
		cfg:    &config.Config{Env: env, WebOrigins: origins},
		logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1})),
	}
}

func TestCORS_ProductionWithoutOriginsIsClosedNotOpen(t *testing.T) {
	got := srvWith("production", nil).allowedOrigins()
	if len(got) == 0 {
		t.Fatalf("قائمةٌ فارغةٌ في الإنتاج — **والمكتبةُ تقرؤها `*` فتفتح البابَ لكلّ موقع**")
	}
	for _, o := range got {
		if o == "*" {
			t.Fatalf("نجمةٌ في الإنتاج — **أيُّ موقعٍ ينادي المحرّكَ بجلسة زائره**")
		}
	}
}

func TestCORS_ProductionUsesConfiguredOrigins(t *testing.T) {
	want := []string{"https://rahalgo.com", "https://www.rahalgo.com"}
	got := srvWith("production", want).allowedOrigins()
	if len(got) != len(want) {
		t.Fatalf("النطاقاتُ المضبوطةُ لم تُقرأ: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("نطاقٌ تبدّل: %q بدل %q", got[i], want[i])
		}
	}
}

func TestCORS_DevelopmentStaysLocal(t *testing.T) {
	got := srvWith("development", nil).allowedOrigins()
	if len(got) == 0 {
		t.Fatalf("التطويرُ بلا نطاقات — **فلا يعمل جهازُ أحد**")
	}
	for _, o := range got {
		if o != "http://localhost:*" && o != "http://127.0.0.1:*" {
			t.Fatalf("نطاقٌ غيرُ محلّيٍّ في التطوير: %q", o)
		}
	}
}
