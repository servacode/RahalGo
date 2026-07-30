package identity

import "testing"

func TestNormalizePhone(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"0932123456", "+963932123456", true},
		{"+963932123456", "+963932123456", true},
		{"00963932123456", "+963932123456", true},
		{"963932123456", "+963932123456", true},
		{"0932 123 456", "+963932123456", true},
		{"0932-123-456", "+963932123456", true},
		{"+905321234567", "+905321234567", true}, // رقم تركي صالح
		{"12345", "", false},
		{"abc", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := NormalizePhone(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("NormalizePhone(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}
