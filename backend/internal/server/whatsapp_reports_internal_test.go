package server

// **طلبُ الواتساب في التقارير والملفّ** (قرارُ المالك ٢٠٢٦-١١-٠٥) — نوعُه مقبولٌ في فلتر التقارير
// بأصله، و«توصيلة متجر» بلاه، وعمودُ النوع في الملفّ يقول «طلب واتساب». **بلا قاعدة.**

import (
	"strings"
	"testing"
)

func TestWhatsAppReportKind(t *testing.T) {
	if !reportKinds["whatsapp"] {
		t.Fatal("**التقاريرُ ترفض نوعَ «طلب واتساب»**")
	}
	if !strings.Contains(reportScopeSQL, "o.origin = 'whatsapp'") ||
		!strings.Contains(reportScopeSQL, "o.origin = 'app'") {
		t.Fatalf("**شرطُ التقارير لا يفصل الواتساب عن توصيلات المتاجر**: %s", reportScopeSQL)
	}
}

func TestWhatsAppExportKindLabel(t *testing.T) {
	if got := exportKind("whatsapp"); got != "طلب واتساب" {
		t.Fatalf("**نوعُ طلب الواتساب في الملفّ**: %q", got)
	}
}
