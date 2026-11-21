package notify

// «ارسل التطبيق» (قرارُ المالك ٢٠٢٦-١١-٢١): الأمرُ يُعرف بأخواته، ولا يُخلط بكلامٍ عاديّ؛
// ورسالةُ الملفّ تحمل اسمَه ونوعَه وسطرَه.

import (
	"testing"

	"go.mau.fi/whatsmeow"
)

func TestWAAPP01_SendAppCommand(t *testing.T) {
	yes := []string{"ارسل التطبيق", "أرسل التطبيق", "إرسل التطبيق", "ابعت التطبيق", "ابعث التطبيق", "بعت التطبيق",
		"ارسال التطبيق", "ابعت تطبيق", "  ارسل   التطبيق  ", "ارسل التطبيق.", "ابعت التطبيق؟"}
	no := []string{"", "ارسل", "التطبيق", "ارسل التطبيق للزبون", "كيف ارسل التطبيق", "ارسل الموقع",
		"نزّلت التطبيق", "ارسلي التطبيق لو سمحت"}
	for _, s := range yes {
		if !IsSendAppCommand(s) {
			t.Errorf("لازم يكون أمر: %q", s)
		}
	}
	for _, s := range no {
		if IsSendAppCommand(s) {
			t.Errorf("مو أمر: %q", s)
		}
	}
}

func TestWAAPP02_DocumentMessage(t *testing.T) {
	up := whatsmeow.UploadResponse{URL: "u", DirectPath: "/p", MediaKey: []byte{1}, FileEncSHA256: []byte{2},
		FileSHA256: []byte{3}, FileLength: 42}
	m := documentMessage(up, OutgoingFile{FileName: "رحال غو 1.15.0.apk", Mime: "application/vnd.android.package-archive",
		Caption: SendAppCaption})
	d := m.GetDocumentMessage()
	if d == nil {
		t.Fatal("لازم تكون رسالة ملف")
	}
	if d.GetFileName() != "رحال غو 1.15.0.apk" || d.GetMimetype() != "application/vnd.android.package-archive" ||
		d.GetCaption() != SendAppCaption || d.GetFileLength() != 42 || d.GetDirectPath() != "/p" || d.GetURL() != "u" {
		t.Fatalf("حقول الملف غلط: %+v", d)
	}
}
