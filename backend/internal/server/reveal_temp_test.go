package server

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/config"
)

// TestRevealTemp_NeverInProduction **الكلمةُ المؤقّتةُ لا تظهر في الإنتاج أبداً** —
// وفي التجهيز تظهر حين لم تصل الرسالةُ وحدَه (قرارُ المالك ٢٠٢٦-١٠-٠٥).
func TestRevealTemp_NeverInProduction(t *testing.T) {
	for _, tc := range []struct {
		env  string
		sent bool
		want string
	}{
		{"production", false, ""},
		{"production", true, ""},
		{"staging", true, ""},
		{"staging", false, "Abc12345"},
		{"development", false, "Abc12345"},
	} {
		s := &Server{cfg: &config.Config{Env: tc.env}}
		if got := s.revealTemp(tc.sent, "Abc12345"); got != tc.want {
			t.Errorf("env=%s sent=%v ⇒ %q — والمتوقّع %q", tc.env, tc.sent, got, tc.want)
		}
	}
}
