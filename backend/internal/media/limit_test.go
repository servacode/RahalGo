package media

// **سقفُ الرفع من اللوحة — بالميغابايت لا بالبايت.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «نطبّقها كلَّها».)
//
// **والمالكُ يكتب «٥» لا «5242880»** — فالتحويلُ في الشيفرة، **وخطؤه يجعل
// السقفَ خمسةَ بايتاتٍ فتُرفض كلُّ صورة**، أو يجعله خمسةَ تيرابايت فلا
// يُرفض شيء. **وكلاهما لا يسقط بناءً ولا يظهر في سجلّ.**

import (
	"context"
	"testing"
)

func TestMaxBytesConvertsMegabytes(t *testing.T) {
	ctx := context.Background()

	// **بلا ربطٍ باللوحة** — الاحتياطيُّ هو ما يعمل.
	if got := (&Service{}).MaxBytes(ctx); got != MaxUploadBytes {
		t.Errorf("السقفُ بلا لوحة = %d، والاحتياطيُّ %d", got, MaxUploadBytes)
	}

	s := &Service{}
	s.SetSettingReader(func(_ context.Context, key string, fallback int64) int64 {
		if key == "media.max_upload_mb" {
			return 12
		}
		return fallback
	})
	if got, want := s.MaxBytes(ctx), int64(12<<20); got != want {
		t.Errorf("السقفُ = %d، والمنتظَر %d — أميغاٌ قُرئت بايتاً؟", got, want)
	}
}
