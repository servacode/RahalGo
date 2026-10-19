package auth

import "testing"

// TestVerifyPasswordTyped **كلمةٌ صحيحةٌ بأرقامٍ عربيّةٍ أو فراغٍ زائدٍ لا تُرفض** (٢٠٢٦-١٠-١٩).
func TestVerifyPasswordTyped(t *testing.T) {
	latin, err := HashPassword("abc12345")
	if err != nil {
		t.Fatal(err)
	}
	arabic, err := HashPassword("abc١٢٣٤٥")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, typed, hash string
		want              bool
	}{
		{"كما هي", "abc12345", latin, true},
		{"أرقامٌ عربيّةٌ لكلمةٍ لاتينيّة", "abc١٢٣٤٥", latin, true},
		{"أرقامٌ فارسيّة", "abc۱۲۳۴۵", latin, true},
		{"فراغٌ زائد", "abc12345 ", latin, true},
		{"أرقامٌ لاتينيّةٌ لكلمةٍ عربيّة", "abc12345", arabic, true},
		{"كلمةٌ أخرى", "abc12346", latin, false},
	}
	for _, c := range cases {
		got, err := VerifyPasswordTyped(c.typed, c.hash)
		if err != nil || got != c.want {
			t.Errorf("%s: %v (%v)", c.name, got, err)
		}
	}
}
