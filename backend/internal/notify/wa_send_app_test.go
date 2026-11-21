package notify

// «ارسل رحال غو» (قرارُ المالك ٢٠٢٦-١١-٢١): الأمرُ يُعرف بأخواته، ولا يُخلط بكلامٍ عاديّ؛
// ورسالةُ الملفّ تحمل اسمَه ونوعَه وسطرَه.

import (
	"testing"

	"go.mau.fi/whatsmeow"
)

func TestWAAPP01_SendAppCommand(t *testing.T) {
	yes := []string{"ارسل رحال غو", "أرسل رحال غو", "إرسل رحّال غو", "ابعت رحال غو", "ابعث رحالغو", "بعت رحّالغو",
		"ارسال رحال غو", "  ارسل   رحال   غو  ", "ارسل رحال غو.", "ابعت رحال غو؟"}
	// **«التطبيق» ما عاد أمر** (قراره «رحال غو بس»).
	no := []string{"", "ارسل", "رحال غو", "ارسل التطبيق", "ابعت تطبيق", "ارسل رحال غو للزبون", "كيف ارسل رحال غو",
		"ارسل رحال", "ارسل غو", "ارسل رحال غو غو", "نزّلت رحال غو", "ارسلي رحال غو"}
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
