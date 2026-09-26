package server

import "testing"

// TestFmtMoneyAr **المبلغُ يُعرَض بفواصل الآلاف ووحدةِ العملة** — ليُقرأ في
// إشعارٍ عابرٍ بلا خطأٍ في القدر.
func TestFmtMoneyAr(t *testing.T) {
	cases := map[int64]string{
		0:       "0 ل.س",
		100:     "100 ل.س",
		1000:    "1,000 ل.س",
		50000:   "50,000 ل.س",
		100000:  "100,000 ل.س",
		1234567: "1,234,567 ل.س",
	}
	for in, want := range cases {
		if got := fmtMoneyAr(in); got != want {
			t.Errorf("fmtMoneyAr(%d) = %q, want %q", in, got, want)
		}
	}
}
