// Package geolink **يقرأ رابطَ موقعٍ من غوغل مابس ويردّه نقطة** (طلبُ المالك ٢٠٢٦-١٠-٠٨).
//
// «الكلّ يعطي مشاركة، ما حدا يبعث رابط مباشر» — الزبونُ يشارك موقعَه بواتساب،
// **والمتجرُ ينسخ الرابطَ ويلصقه في «لدي توصيلة»** بدل أن يبحث عن نقطةٍ في
// خريطةٍ لا يعرفها. **والسائقُ يرى نقطةً على خريطتنا لا رابطاً.**
//
// # الأشكالُ المقروءة
//
//   - `maps.google.com/maps?q=35.95,39.01` — مشاركةُ موقعٍ من واتساب.
//   - `google.com/maps/place/…/@35.95,39.01,17z/data=…!3d35.95!4d39.01`
//   - `maps.app.goo.gl/xxxx` و`goo.gl/maps/xxxx` — مختصرٌ يُتبَع إلى الكامل.
//
// # والأمان (SSRF)
//
// **لا يُفتح إلّا رابطُ غوغل مابس** — وكلُّ قفزةِ تحويلٍ تُفحص من جديد، **فمختصرٌ
// يحوّل إلى عنوانٍ داخليٍّ لا يُتبَع.** ولا يُقرأ جسمُ الصفحة: الإحداثيّاتُ في الرابط.
package geolink

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ErrNoLocation **رابطٌ لا موقعَ فيه** — أو ليس رابطَ خرائط أصلاً.
var ErrNoLocation = errors.New("geolink: no location in link")

var (
	urlInText  = regexp.MustCompile(`https?://[^\s<>"']+`)
	googleHost = regexp.MustCompile(`^(www\.|maps\.)?google\.[a-z]{2,3}(\.[a-z]{2})?$`)
	// `!3d<lat>!4d<lng>` — أدقُّ ما في رابط المكان (موضعُ الدبّوس لا وسطُ الشاشة).
	pinData = regexp.MustCompile(`!3d(-?\d{1,2}\.\d+)!4d(-?\d{1,3}\.\d+)`)
	// `/@<lat>,<lng>,<zoom>z` — وسطُ الخريطة.
	atCenter = regexp.MustCompile(`/@(-?\d{1,2}\.\d+),(-?\d{1,3}\.\d+)`)
	// `<lat>,<lng>` في معاملٍ أو مسار — بفاصلةٍ وفراغٍ أو `+` اختياريّ.
	pair     = regexp.MustCompile(`^\s*(-?\d{1,2}(?:\.\d+)?)\s*,\s*\+?\s*(-?\d{1,3}(?:\.\d+)?)\s*$`)
	pathPair = regexp.MustCompile(`/(?:place|search|dir/?)/(-?\d{1,2}\.\d+),\+?\s*(-?\d{1,3}\.\d+)`)
)

// FindURL **أوّلُ رابطٍ في نصٍّ ملصوق** — المشاركةُ قد تحمل اسمَ المكان قبل الرابط.
func FindURL(text string) string {
	return strings.TrimRight(urlInText.FindString(text), ".,;)")
}

// allowed **أرابطُ خرائط غوغل؟** — المختصرُ أو الكامل.
func allowed(u *url.URL) bool {
	if u.Scheme != "https" && u.Scheme != "http" {
		return false
	}
	h := strings.ToLower(u.Hostname())
	switch {
	case h == "maps.app.goo.gl":
		return true
	case h == "goo.gl":
		return strings.HasPrefix(u.Path, "/maps")
	case h == "maps.google.com" || strings.HasPrefix(h, "maps.google."):
		return googleHost.MatchString(h)
	case googleHost.MatchString(h):
		return strings.HasPrefix(u.Path, "/maps") || u.Path == "/" || u.Path == ""
	}
	return false
}

// Parse **يقرأ النقطةَ من رابطٍ كامل** — بلا شبكة.
func Parse(raw string) (lat, lng float64, err error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return 0, 0, ErrNoLocation
	}
	full := u.String()
	if dec, e := url.PathUnescape(full); e == nil {
		full = dec
	}
	if m := pinData.FindStringSubmatch(full); m != nil {
		return valid(m[1], m[2])
	}
	q := u.Query()
	for _, k := range []string{"q", "query", "ll", "destination", "daddr", "center"} {
		if m := pair.FindStringSubmatch(strings.TrimPrefix(q.Get(k), "loc:")); m != nil {
			return valid(m[1], m[2])
		}
	}
	if m := pathPair.FindStringSubmatch(full); m != nil {
		return valid(m[1], m[2])
	}
	if m := atCenter.FindStringSubmatch(full); m != nil {
		return valid(m[1], m[2])
	}
	return 0, 0, ErrNoLocation
}

func valid(a, b string) (float64, float64, error) {
	lat, e1 := strconv.ParseFloat(a, 64)
	lng, e2 := strconv.ParseFloat(b, 64)
	if e1 != nil || e2 != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 || (lat == 0 && lng == 0) {
		return 0, 0, ErrNoLocation
	}
	return lat, lng, nil
}

// Resolve **يقرأ نصّاً ملصوقاً ويردّ نقطة** — ويتبع المختصرَ قفزةً قفزة.
//
// **وكلُّ قفزةٍ تُفحص**: مضيفُ غوغل مابس وحدَه. **وصفحةُ الموافقة** (`consent.google.*`،
// تظهر لخادمٍ في أوروبا) **لا تُفتح** — الرابطُ الحقيقيُّ في معاملها `continue`.
func Resolve(ctx context.Context, client *http.Client, text string) (float64, float64, error) {
	raw := FindURL(text)
	if raw == "" {
		return 0, 0, ErrNoLocation
	}
	if client == nil {
		client = &http.Client{Timeout: 6 * time.Second}
	}
	noFollow := *client
	noFollow.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	for hop := 0; hop < 6; hop++ {
		u, err := url.Parse(raw)
		if err != nil {
			return 0, 0, ErrNoLocation
		}
		if strings.HasPrefix(strings.ToLower(u.Hostname()), "consent.google.") {
			if c := u.Query().Get("continue"); c != "" {
				raw = c
				continue
			}
			return 0, 0, ErrNoLocation
		}
		if !allowed(u) {
			return 0, 0, ErrNoLocation
		}
		if lat, lng, err := Parse(raw); err == nil {
			return lat, lng, nil
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
		if err != nil {
			return 0, 0, ErrNoLocation
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 13) RahalGo")
		res, err := noFollow.Do(req)
		if err != nil {
			return 0, 0, err
		}
		_ = res.Body.Close()
		loc := res.Header.Get("Location")
		if loc == "" {
			return 0, 0, ErrNoLocation
		}
		next, err := u.Parse(loc)
		if err != nil {
			return 0, 0, ErrNoLocation
		}
		raw = next.String()
	}
	return 0, 0, ErrNoLocation
}
