package server

import "testing"

// TestLeadAddress_StreetCityDistrict — «شارع تل أبيض - الرقة - مركز الرقة» (قرارُ
// المالك ٢٠٢٦-١٠-١٢): من الخاصّ إلى العامّ، **ولا تتكرّر المدينة.**
func TestLeadAddress_StreetCityDistrict(t *testing.T) {
	for _, c := range []struct{ area, want string }{
		{"شارع تل أبيض", "شارع تل أبيض - الرقة - مركز الرقة"},
		{"الرقة — شارع تل أبيض", "شارع تل أبيض - الرقة - مركز الرقة"},
		{"الرقة - شارع تل أبيض، جنب الجامع", "شارع تل أبيض، جنب الجامع - الرقة - مركز الرقة"},
		{"", "الرقة - مركز الرقة"},
	} {
		if got := leadAddress(c.area, "الرقة", "مركز الرقة"); got != c.want {
			t.Errorf("%q ⇒ %q، والمنتظَر %q", c.area, got, c.want)
		}
	}
}
