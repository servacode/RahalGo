// Package geo وسيط العنونة (Geocoding): تحويل إحداثيات ↔ وصف عنوان.
//
// لماذا وسيط في الخادم بدل نداء مباشر من المتصفح:
//   - سياسة استخدام المزوّد تتطلب معرّف تطبيق ومعدّلاً محدوداً — يُضبط هنا مرة.
//   - كاش مشترك: عشرة مندوبين في الحي نفسه لا يستهلكون عشرة نداءات.
//   - تبديل المزوّد لاحقاً (Nominatim ذاتي الاستضافة) تهيئةٌ لا تعديل واجهات.
//
// **حدّ الواقع**: تغطية العنونة في الرقة ضعيفة أصلاً — لذا قرار المشروع
// (PLAN §6.4) أن العنوان وصف نصّي + دبوس، والعنونة هنا **مساعدة لا مصدر حقيقة**.
// كل دوالها تفشل بهدوء: تعيد فراغاً ولا تكسر أي تدفّق.
package geo

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	cacheTTL    = 7 * 24 * time.Hour // العناوين لا تتغير كثيراً
	httpTimeout = 6 * time.Second
	maxResults  = 5
)

// Place نتيجة عنونة واحدة — وصف مختصر بالعربية وإحداثيات.
type Place struct {
	Label string  `json:"label"`
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
}

type Service struct {
	base   string
	client *http.Client
	rdb    *redis.Client
	logger *slog.Logger
}

func New(baseURL string, rdb *redis.Client, logger *slog.Logger) *Service {
	return &Service{
		base:   strings.TrimRight(baseURL, "/"),
		client: &http.Client{Timeout: httpTimeout},
		rdb:    rdb,
		logger: logger,
	}
}

// nominatimAddress الحقول التي تهمّنا من ردّ المزوّد.
type nominatimAddress struct {
	Road          string `json:"road"`
	Neighbourhood string `json:"neighbourhood"`
	Suburb        string `json:"suburb"`
	Quarter       string `json:"quarter"`
	City          string `json:"city"`
	Town          string `json:"town"`
	Village       string `json:"village"`
	State         string `json:"state"`
}

type nominatimResult struct {
	Lat         string           `json:"lat"`
	Lon         string           `json:"lon"`
	DisplayName string           `json:"display_name"`
	Name        string           `json:"name"`
	Address     nominatimAddress `json:"address"`
}

// label يبني وصفاً عربياً موجزاً: المعلم/الشارع ← الحي ← المدينة.
// نتجنّب display_name الكامل لأنه يذيّل كل شيء بـ"سوريا" ورموز بريدية لا تعني أحداً.
func (r nominatimResult) label() string {
	parts := []string{}
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		for _, p := range parts {
			if p == v {
				return
			}
		}
		parts = append(parts, v)
	}
	add(r.Name)
	add(r.Address.Road)
	add(firstNonEmpty(r.Address.Neighbourhood, r.Address.Quarter, r.Address.Suburb))
	add(firstNonEmpty(r.Address.City, r.Address.Town, r.Address.Village))
	if len(parts) == 0 {
		return strings.TrimSpace(r.DisplayName)
	}
	return strings.Join(parts, "، ")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// get ينفّذ نداءً مع كاش Redis — الفشل صامت (يعيد بايتات فارغة).
func (s *Service) get(ctx context.Context, cacheKey, path string, q url.Values) []byte {
	if v, err := s.rdb.Get(ctx, cacheKey).Bytes(); err == nil {
		return v
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.base+path+"?"+q.Encode(), nil)
	if err != nil {
		return nil
	}
	// سياسة Nominatim تُلزم بمعرّف تطبيق حقيقي
	req.Header.Set("User-Agent", "RahalGo/1.0 (delivery platform; Raqqa)")
	req.Header.Set("Accept-Language", "ar")

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Warn("geo: request failed", "error", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		s.logger.Warn("geo: bad status", "status", resp.StatusCode)
		return nil
	}
	body := make([]byte, 0, 4096)
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		body = append(body, buf[:n]...)
		if err != nil || len(body) > 1<<20 {
			break
		}
	}
	s.rdb.Set(ctx, cacheKey, body, cacheTTL)
	return body
}

// Reverse إحداثيات ← وصف عنوان. يعيد وصفاً فارغاً عند التعذّر.
func (s *Service) Reverse(ctx context.Context, lat, lng float64) Place {
	q := url.Values{}
	q.Set("format", "jsonv2")
	q.Set("lat", fmt.Sprintf("%.6f", lat))
	q.Set("lon", fmt.Sprintf("%.6f", lng))
	q.Set("zoom", "18")
	q.Set("accept-language", "ar")

	key := fmt.Sprintf("geo:rev:%.5f:%.5f", lat, lng)
	body := s.get(ctx, key, "/reverse", q)
	if len(body) == 0 {
		return Place{Lat: lat, Lng: lng}
	}
	var res nominatimResult
	if err := json.Unmarshal(body, &res); err != nil {
		return Place{Lat: lat, Lng: lng}
	}
	return Place{Label: res.label(), Lat: lat, Lng: lng}
}

// Search وصف عنوان ← مواقع مرشّحة (سوريا فقط). قائمة فارغة عند التعذّر.
func (s *Service) Search(ctx context.Context, query string) []Place {
	query = strings.TrimSpace(query)
	if len([]rune(query)) < 3 {
		return []Place{}
	}
	q := url.Values{}
	q.Set("format", "jsonv2")
	q.Set("q", query)
	q.Set("countrycodes", "sy")
	q.Set("limit", fmt.Sprint(maxResults))
	q.Set("addressdetails", "1")
	q.Set("accept-language", "ar")

	body := s.get(ctx, "geo:q:"+strings.ToLower(query), "/search", q)
	out := []Place{}
	if len(body) == 0 {
		return out
	}
	var results []nominatimResult
	if err := json.Unmarshal(body, &results); err != nil {
		return out
	}
	for _, r := range results {
		var lat, lng float64
		if _, err := fmt.Sscanf(r.Lat, "%f", &lat); err != nil {
			continue
		}
		if _, err := fmt.Sscanf(r.Lon, "%f", &lng); err != nil {
			continue
		}
		out = append(out, Place{Label: r.label(), Lat: lat, Lng: lng})
	}
	return out
}
