package server

import (
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// نقاط العنونة — لأي مستخدم مسجّل (المندوب يسجّل متجراً، الزبون يضبط عنوانه).
// محدودة المعدّل: المزوّد خارجي وسياسته تُلزمنا، ولا نريد أن يُغرقه تطبيقنا.

const geoMaxPerHour = 120

// geoAllowed حدّ معدّل لكل مستخدم (وعنوانه) — يمنع الاستنزاف بلا إزعاج الاستعمال العادي.
func (s *Server) geoAllowed(r *http.Request) bool {
	key := "geo:rl:" + userIDFrom(r)
	if key == "geo:rl:" {
		ip := clientIP(r)
		if host, _, err := net.SplitHostPort(ip); err == nil {
			ip = host
		}
		key = "geo:rl:ip:" + ip
	}
	n, err := s.rdb.Incr(r.Context(), key).Result()
	if err != nil {
		return true // عطل الكاش لا يعطّل الميزة
	}
	if n == 1 {
		s.rdb.Expire(r.Context(), key, time.Hour)
	}
	return n <= geoMaxPerHour
}

// handleGeoReverse إحداثيات ← وصف عنوان مقترح (المستخدم يبقى حرّاً في تعديله).
func (s *Server) handleGeoReverse(w http.ResponseWriter, r *http.Request) {
	if !s.geoAllowed(r) {
		s.respondErr(w, httpx.NewError(http.StatusTooManyRequests, "rate_limited", "errors.rate_limited"))
		return
	}
	lat, err1 := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, err2 := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if err1 != nil || err2 != nil {
		s.respondErr(w, errValidation)
		return
	}
	httpx.JSON(w, http.StatusOK, s.geo.Reverse(r.Context(), lat, lng))
}

// handleGeoSearch وصف عنوان ← مواقع مرشّحة. قائمة فارغة نتيجة مشروعة لا خطأ:
// تغطية العنونة في الرقة ضعيفة، والواجهة تتعامل مع الفراغ بأن تُبقي الدبوس يدوياً.
func (s *Server) handleGeoSearch(w http.ResponseWriter, r *http.Request) {
	if !s.geoAllowed(r) {
		s.respondErr(w, httpx.NewError(http.StatusTooManyRequests, "rate_limited", "errors.rate_limited"))
		return
	}
	httpx.JSON(w, http.StatusOK, s.geo.Search(r.Context(), r.URL.Query().Get("q")))
}
