package geolink

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// **أشكالُ الروابط كما تصل من «مشاركة» واتساب وغوغل مابس.**
func TestParse_Shapes(t *testing.T) {
	for _, tc := range []struct {
		in       string
		lat, lng float64
	}{
		{"https://maps.google.com/maps?q=35.9522%2C39.0120&z=17&hl=ar", 35.9522, 39.0120},
		{"https://maps.google.com/?q=35.95,39.01", 35.95, 39.01},
		{"https://www.google.com/maps/search/?api=1&query=35.9522,39.012", 35.9522, 39.012},
		{"https://www.google.com/maps/place/%D8%A7%D9%84%D8%B1%D9%82%D8%A9/@35.9500,39.0000,15z/data=!3m1!4b1!4m6!3m5!1s0x0:0x0!8m2!3d35.9522!4d39.0120", 35.9522, 39.0120},
		{"https://www.google.com/maps/@35.9511,39.0133,17z", 35.9511, 39.0133},
		{"https://www.google.com/maps/search/35.9522,+39.0120?entry=tts", 35.9522, 39.0120},
		{"https://www.google.com/maps/dir//35.9522,39.0120", 35.9522, 39.0120},
		{"https://maps.google.com/maps?q=loc:35.95,39.01", 35.95, 39.01},
	} {
		lat, lng, err := Parse(tc.in)
		if err != nil || lat != tc.lat || lng != tc.lng {
			t.Errorf("%s ⇒ %v,%v %v — المطلوب %v,%v", tc.in, lat, lng, err, tc.lat, tc.lng)
		}
	}
	for _, bad := range []string{
		"https://www.google.com/maps/place/Raqqa",
		"https://maps.google.com/?q=0,0",
		"https://maps.google.com/?q=95,39",
		"not a link",
	} {
		if _, _, err := Parse(bad); err == nil {
			t.Errorf("%q قُرئ موقعاً وليس فيه موقع", bad)
		}
	}
}

// **المشاركةُ تحمل نصّاً قبل الرابط** — يُلتقط الرابطُ منه.
func TestFindURL_InSharedText(t *testing.T) {
	got := FindURL("موقعي: https://maps.app.goo.gl/AbCd123 تعال لهون.")
	if got != "https://maps.app.goo.gl/AbCd123" {
		t.Fatalf("%q", got)
	}
}

// **ولا يُفتح إلّا غوغل مابس** — رابطٌ غريبٌ أو عنوانٌ داخليٌّ لا يُطرق.
func TestResolve_RejectsForeignHosts(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer srv.Close()
	for _, in := range []string{srv.URL + "/maps?q=35.9,39.0", "http://169.254.169.254/latest", "https://evil.com/maps?q=35.9,39.0", "https://goo.gl/notmaps"} {
		if _, _, err := Resolve(context.Background(), srv.Client(), in); err == nil {
			t.Errorf("%q قُبل", in)
		}
	}
	if hit {
		t.Fatal("**طُرق مضيفٌ ليس غوغل مابس**")
	}
}

// **والمختصرُ يُتبَع — وكلُّ قفزةٍ تُفحص**: تحويلٌ إلى مضيفٍ آخر يُوقف.
func TestResolve_FollowsAndChecksEveryHop(t *testing.T) {
	var target string
	rt := roundTrip(func(r *http.Request) *http.Response {
		h := http.Header{}
		h.Set("Location", target)
		return &http.Response{StatusCode: http.StatusFound, Header: h, Body: http.NoBody, Request: r}
	})
	c := &http.Client{Transport: rt}

	target = "https://consent.google.com/ml?continue=" + url.QueryEscape("https://www.google.com/maps/place/x/data=!3d35.9522!4d39.0120")
	lat, lng, err := Resolve(context.Background(), c, "https://maps.app.goo.gl/AbCd123")
	if err != nil || lat != 35.9522 || lng != 39.0120 {
		t.Fatalf("مختصرٌ عبر صفحة الموافقة: %v,%v %v", lat, lng, err)
	}

	target = "http://127.0.0.1:8080/maps?q=35.9,39.0"
	if _, _, err := Resolve(context.Background(), c, "https://maps.app.goo.gl/AbCd123"); err == nil {
		t.Fatal("**تحويلٌ إلى عنوانٍ داخليٍّ قُبل**")
	}
}

type roundTrip func(*http.Request) *http.Response

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r), nil }
