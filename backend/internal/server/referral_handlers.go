package server

// نقاطُ الدعوة — **رمزٌ يُنسخ ورابطٌ يُرسَل.**
//
// # ولماذا رابطٌ لا رمزٌ فقط
//
// **رمزٌ يُملى بالهاتف يُكتب خطأً**، ورابطٌ يُلصق في واتساب يُضغط. **والفرقُ
// بينهما هو الفرقُ بين دعوةٍ تصل ودعوةٍ تضيع.**

import (
	"net/http"
	"os"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/httpx"
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
	// **ووجهتُه صفحةُ دخول الزبون لا صفحةُ انضمام المتاجر** — `/join`
	// للمتاجر بكود المندوب، **وشيءٌ اسمُه «ref» في مكانين يُخلط بينهما.**
	httpx.JSON(w, http.StatusOK, struct {
		*referrals.Standing
		Link string `json:"link"`
	}{st, s.siteURL() + "/login?ref=" + st.Code})
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
