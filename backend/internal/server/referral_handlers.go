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
)

// handleMyReferral رمزُ الدعوة وحالُها — لصاحب الحساب.
func (s *Server) handleMyReferral(w http.ResponseWriter, r *http.Request) {
	st, err := s.referrals.Standing(r.Context(), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والرابطُ يُبنى في الخادم لا في الشاشة.**
	//
	// **وعنوانُ الموقع يتغيّر** — من نشرٍ إلى نشر، ومن نطاقٍ إلى نطاق.
	// **ورابطٌ يُركَّب في متصفّحٍ يحمل عنوانَ الصفحة التي فُتحت منها**،
	// فمن فتح اللوحةَ على `localhost` أرسل دعوةً إلى `localhost`.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"code": st.Code,
		// **ووجهتُه صفحةُ دخول الزبون لا صفحةُ انضمام المتاجر** —
		// `/join` للمتاجر بكود المندوب، **وشيءٌ اسمُه «ref» في مكانين
		// يُخلط بينهما.**
		"link":        s.siteURL() + "/login?ref=" + st.Code,
		"invited":     st.Invited,
		"rewarded":    st.Rewarded,
		"earned":      st.Earned,
		"next_reward": st.NextReward,
	})
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
