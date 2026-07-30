package identity

import (
	"regexp"
	"strings"
)

var e164Re = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

// NormalizePhone يوحّد صيغة الرقم إلى E.164 مع دعم الصيغ السورية الشائعة:
// 0932123456 → +963932123456 ، 00963... → +963... ، 963... → +963...
func NormalizePhone(raw string) (string, bool) {
	p := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' || r == '+' {
			return r
		}
		return -1
	}, raw)

	switch {
	case strings.HasPrefix(p, "00"):
		p = "+" + p[2:]
	case strings.HasPrefix(p, "0") && len(p) == 10:
		p = "+963" + p[1:]
	case strings.HasPrefix(p, "963"):
		p = "+" + p
	}

	if !e164Re.MatchString(p) {
		return "", false
	}
	return p, true
}
