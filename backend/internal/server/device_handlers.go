package server

import (
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
//  **تسجيلُ جهازٍ للدفع — نقطتان لا أكثر**
// ══════════════════════════════════════════════════════════════════════
//
// (خطّةُ تطبيق أندرويد، البند الثاني.)
//
// **والتسجيلُ في كلّ إقلاعٍ للتطبيق لا مرّةً واحدة**: رمزُ FCM يتبدّل عند
// إعادة التثبيت وعند مسح البيانات **وأحياناً من تلقاء نفسه.** ومن سجّله
// مرّةً ثمّ نسي **يفقد إشعاراتِه بلا أن يعلم** — والعطبُ صامتٌ تماماً:
// التطبيقُ يعمل، والصندوقُ يمتلئ، **ولا شيءَ يرنّ.**

const maxDeviceToken = 4096

// handleDeviceRegister يُسجّل رمزَ جهازٍ لصاحب الجلسة.
func (s *Server) handleDeviceRegister(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Token string `json:"token"`
		// android · ios — **والمجهولُ أندرويد** (انظر `push.normalizePlatform`).
		Platform string `json:"platform"`
		// **إصدارُ التطبيق** — يُقرأ حين يشكو صاحبُه، **ويُعرف من أيّ
		// نسخةٍ يشكو قبل أن يُسأل.**
		AppVersion string `json:"app_version"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	token := strings.TrimSpace(req.Token)
	// **ورمزٌ فارغٌ أو مهولٌ يُرفض هنا لا في القاعدة**: رموزُ FCM نحو ١٦٠
	// حرفاً، **وأربعةُ آلافٍ سقفٌ سخيٌّ يمنع أن يُملأ الجدولُ بنصٍّ حرّ.**
	if token == "" || len(token) > maxDeviceToken {
		s.respondErr(w, errValidation)
		return
	}
	if err := s.push.Register(r.Context(), userIDFrom(r), token, req.Platform, req.AppVersion); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"registered": true})
}

// handleDeviceUnregister يحذف رمزَ الجهاز — **عند الخروج من التطبيق.**
//
// **ولولاه لَبقي الهاتفُ يستقبل إشعاراتِ حسابٍ خرج منه** — وهو تسريبٌ
// حقيقيّ: هاتفٌ مشترَكٌ في البيت يُظهر أسماءَ زبائنِ سائقٍ خرج.
func (s *Server) handleDeviceUnregister(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Token string `json:"token"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.push.Unregister(r.Context(), userIDFrom(r), strings.TrimSpace(req.Token)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"removed": true})
}
