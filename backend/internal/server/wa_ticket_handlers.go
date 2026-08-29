package server

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
)

// ══════════════════════════════════════════════════════════════════════
// **تذكرةُ واتساب — يفتحها التطبيقُ ويرسلها صاحبُها بيده**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك 2026-08-25.)
//
// **والردُّ يحمل رابطاً جاهزاً** — فالتطبيقُ يفتحه ولا يبني نصّاً ولا
// يعرف رقمَ المنصّة. **ومن بنى الرسالةَ في أربعة تطبيقاتٍ اختلفت
// أربعَ مرّات**، ورقمٌ يُبدَّل في اللوحة يجب أن يعمل بلا بناءٍ جديد.
func (s *Server) handleWATicket(w http.ResponseWriter, r *http.Request) {
	in, err := decode[struct {
		Phone   string `json:"phone"`
		Purpose string `json:"purpose"` // verify | reset
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	purpose := identity.WAVerify
	if in.Purpose == "reset" {
		purpose = identity.WAReset
	}
	tag, err2 := s.identity.WATicket(r.Context(), in.Phone, purpose)
	if err2 != nil {
		s.respondErr(w, err2)
		return
	}

	num := strings.TrimSpace(s.settings.GetString(r.Context(), "platform.whatsapp"))
	num = strings.TrimPrefix(num, "+")
	// **ورقمٌ غيرُ مضبوطٍ يُقال صراحةً** — ورابطٌ إلى `wa.me/` فارغٍ
	// يفتح واتساب على لا شيء، **فيظنّ صاحبُه العطبَ في هاتفه.**
	if num == "" {
		s.respondErr(w, identity.ErrOTPSendFailed)
		return
	}

	body := "مرحبا رحال غو، أريد توثيق حسابي."
	if purpose == identity.WAReset {
		body = "مرحبا رحال غو، أريد استعادة كلمة المرور الخاصة بي."
	}
	// **والوسمُ في آخر السطر لا في وسطه** — فمن حذف كلاماً أو أضاف
	// بقي الوسمُ سليماً، **و`waTagRe` تلتقطه أينما وقع.**
	text := body + "\nرمز الطلب: " + tag

	httpx.JSON(w, http.StatusOK, map[string]string{
		"tag":    tag,
		"text":   text,
		"wa_url": "https://wa.me/" + num + "?text=" + url.QueryEscape(text),
	})
}
