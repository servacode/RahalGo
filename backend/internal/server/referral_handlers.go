package server

// نقاطُ الدعوة — **رمزٌ يُنسخ ورابطٌ يُرسَل.**
//
// # ولماذا رابطٌ لا رمزٌ فقط
//
// **رمزٌ يُملى بالهاتف يُكتب خطأً**، ورابطٌ يُلصق في واتساب يُضغط. **والفرقُ
// بينهما هو الفرقُ بين دعوةٍ تصل ودعوةٍ تضيع.**

import (
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/referrals"
)

// handleMyReferral رمزُ الدعوة وحالُها — لصاحب الحساب.
func (s *Server) handleMyReferral(w http.ResponseWriter, r *http.Request) {
	st, err := s.referrals.Standing(r.Context(), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والحالُ تُرسَل كما هي، والرابطُ يُضاف فوقها.**
	//
	// **وكانت الحقولُ تُنسخ حقلاً حقلاً** — فأُضيفت الدرجاتُ والشرطُ إلى
	// `Standing` **ولم يصلا الشاشةَ أبداً، ولا خطأَ يُقال**: الخادمُ يبني،
	// والواجهةُ تقرأ `undefined`، **والجدولُ لا يظهر ولا أحدَ يعرف لماذا.**
	//
	// **والتضمينُ يجعل الحقلَ الجديدَ يُشحن وحدَه** — وهي عائلةُ «قاعدةٌ
	// مكتوبةٌ مرّتين تفترق بلا صوت» نفسُها.
	//
	// **والرابطُ يُبنى في الخادم لا في الشاشة**: عنوانُ الموقع يتغيّر من نشرٍ
	// إلى نشر، **ورابطٌ يُركَّب في متصفّحٍ يحمل عنوانَ الصفحة التي فُتحت
	// منها** — فمن فتح اللوحةَ على `localhost` أرسل دعوةً إلى `localhost`.
	//
	// **ووجهتُه صفحةُ إنشاء الحساب لا صفحةُ انضمام المتاجر** — `/join`
	// للمتاجر بكود المندوب، **وشيءٌ اسمُه «ref» في مكانين يُخلط بينهما.**
	//
	// **وكانت `/login`** (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا يوجد رابطُ تسجيل»):
	// **من دُعي ليُنشئ حساباً كان يقع على شاشةٍ تطلب كلمةَ مرورٍ لا يملكها**،
	// وعليه أن يجد «إنشاء حساب» بنفسه. **ودعوةٌ تقود إلى الباب الخطأ دعوةٌ
	// ضائعة.**
	httpx.JSON(w, http.StatusOK, struct {
		*referrals.Standing
		Link string `json:"link"`
	}{st, s.siteURL() + "/signup?ref=" + st.Code})
}

// siteURL عنوانُ موقع الزبائن — **من البيئة لا من رأس الطلب.**
//
// **ورأسُ `Origin` يقول من أين فُتحت الشاشةُ لا أين يعيش الموقع**: من فتح
// اللوحةَ على `localhost` أرسل دعوةً إلى `localhost`، **ومن فتحها من خلف
// وسيطٍ أرسل عنوانَ الوسيط.**
func (s *Server) siteURL() string {
	if v := os.Getenv("PUBLIC_SITE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	// **وافتراضُ التطوير موقعُ الزبون** — لا الخادم: الرابطُ يُفتح في متصفّح.
	if s.cfg.Env == "development" {
		return "http://localhost:3003"
	}
	return "https://rahalgo.com"
}

// inviteClaimMaxPerHour **حدُّ الحجز للعنوان** — صفحةٌ عامّةٌ بلا حساب.
const inviteClaimMaxPerHour = 20

// ErrInviteClaimRateLimited حجوزٌ كثيرةٌ من عنوانٍ واحد.
var ErrInviteClaimRateLimited = httpx.NewError(http.StatusTooManyRequests,
	"rate_limited", "errors.rate_limited")

// handleInviteClaim **يحجز الدعوةَ برقم الصديق** — `POST /public/invite/claim`.
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٥: «بدون نسخ رمز».) صفحةُ الدعوة ترسل الرمزَ من
// رابطها والرقمَ الذي كتبه الصديق، **وعند تسجيله بلا `ref` يُنسب بالحجز**
// (`handleSignupConfirm`). انظر `referrals/claims.go`.
//
// **وعامٌّ بلا حساب** — فالحارسُ حدُّ معدّلٍ بالعنوان، **ويُعدّ ولو رُفض**:
// وإلّا صار تخمينُ الرموز مجّانيّاً.
func (s *Server) handleInviteClaim(w http.ResponseWriter, r *http.Request) {
	if !s.inviteClaimAllowed(r) {
		s.respondErr(w, ErrInviteClaimRateLimited)
		return
	}
	req, err := decode[struct {
		Code  string `json:"code"`
		Phone string `json:"phone"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	phone, ok := identity.NormalizePhone(req.Phone)
	if !ok {
		s.respondErr(w, identity.ErrInvalidPhone)
		return
	}
	switch err := s.referrals.ClaimInvite(r.Context(), req.Code, phone); {
	case errors.Is(err, referrals.ErrPhoneRegistered):
		s.respondErr(w, identity.ErrPhoneTaken)
		return
	case err != nil:
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// inviteClaimAllowed حدُّ معدّلٍ بالعنوان — كحارس العنونة (`geoAllowed`).
func (s *Server) inviteClaimAllowed(r *http.Request) bool {
	ip := clientIP(r)
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	key := "invite:claim:ip:" + ip
	n, err := s.rdb.Incr(r.Context(), key).Result()
	if err != nil {
		return true // عطلُ الكاش لا يعطّل الصفحة
	}
	if n == 1 {
		s.rdb.Expire(r.Context(), key, time.Hour)
	}
	return n <= inviteClaimMaxPerHour
}
