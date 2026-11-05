package orders

import (
	"errors"
	"testing"
)

// TestWAPointOutsideOnlyWhenKnown **«موقع مستلم طلب واتساب برّا التغطية» بيقينٍ لا بظنّ** — فحصُ
// تغطيةٍ سقط (عطلُ القاعدة) ليس «خارجها»: **لا يُخبَر المكتبُ بإنذارٍ كاذب** (مراجعةُ ٢٠٢٦-١١-٠٥).
//
// **وعلى الشيفرة السابقة** (d419e42e) كان الإنذارُ يُبعث كلّما `covered=false` — والسقوطُ يردّها false.
func TestWAPointOutsideOnlyWhenKnown(t *testing.T) {
	cases := []struct {
		name    string
		covered bool
		err     error
		want    bool
	}{
		{"داخلَ التغطية", true, nil, false},
		{"خارجَها يقيناً", false, nil, true},
		{"الفحصُ سقط", false, errors.New("conn reset"), false},
	}
	for _, c := range cases {
		if got := waPointOutside(c.covered, c.err); got != c.want {
			t.Errorf("**%s**: %v — والمنتظَر %v", c.name, got, c.want)
		}
	}
}
