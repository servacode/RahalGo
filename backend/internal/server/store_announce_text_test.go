package server

import (
	"strings"
	"testing"
)

func TestStoreAnnounceTextIsolatesLatinName(t *testing.T) {
	title, body := storeAnnounceText("QA-Money-Model2")
	want := "\u2068QA-Money-Model2\u2069"
	for _, s := range []string{title, body} {
		if !strings.Contains(s, want) {
			t.Fatalf("الاسم اللاتيني غير معزول كاملاً: %q", s)
		}
	}
	t1, _ := storeAnnounceText("دجاجتي")
	if strings.Contains(t1, "\u2068") {
		t.Fatalf("اسم عربي لا يُعزل: %q", t1)
	}
}
