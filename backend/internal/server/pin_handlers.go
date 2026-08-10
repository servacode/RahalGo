package server

// **معالِجاتُ رمز الأدمن** — أبوابٌ رفيعةٌ فوق `identity`، ولا منطقَ فيها.
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «رمزُ دخولٍ ثانٍ من ٤ أرقام… ويستطيع تبديله من
//
//	لوحة التحكّم».)

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// handlePinVerify **الخطوةُ الثانية: رمزٌ فتُصدَر الجلسة.**
func (s *Server) handlePinVerify(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Challenge string `json:"challenge"`
		Pin       string `json:"pin"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.identity.VerifyPin(r.Context(), req.Challenge, req.Pin, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

// handlePinSetup **أوّلُ ضبطٍ للرمز — بالتحدّي نفسِه.**
//
// **ولا يُضبط فوق رمزٍ قائم** — الخدمةُ ترفض، **فمن سرق تحدّياً لا يُبدّل
// رمزَ من يملكه.**
func (s *Server) handlePinSetup(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Challenge string `json:"challenge"`
		Pin       string `json:"pin"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.identity.SetPinFirstTime(r.Context(), req.Challenge, req.Pin, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

// handlePinState **أله رمزٌ ومتى ضُبط؟** — تقرؤها شاشةُ «حسابي».
func (s *Server) handlePinState(w http.ResponseWriter, r *http.Request) {
	set, at, err := s.identity.HasPin(r.Context(), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	required := false
	for _, role := range rolesFrom(r) {
		if role == "admin" {
			required = true
		}
	}
	out := map[string]any{"required": required, "set": set}
	if at != nil {
		out["set_at"] = at.Format(time.RFC3339)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handlePinChange **تبديلُ الرمز من اللوحة — بالقديم.**
func (s *Server) handlePinChange(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Current string `json:"current"`
		Pin     string `json:"pin"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.ChangePin(r.Context(), userIDFrom(r), req.Current, req.Pin, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "auth.pin_changed", "user", userIDFrom(r), map[string]any{})
	httpx.JSON(w, http.StatusOK, map[string]any{"changed": true})
}

// handlePinResetRequest **بابُ النجاة — رمزٌ إلى واتسابه المُوثَّق.**
//
// **ومن نسي رمزَه بلا هذا الباب أُقفلت لوحتُه على نفسه** — ولا أحدَ فوقه
// يفتحها له، **فهو المالك.**
func (s *Server) handlePinResetRequest(w http.ResponseWriter, r *http.Request) {
	if err := s.identity.ResetPinRequest(r.Context(), userIDFrom(r), clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": true})
}

// handlePinResetConfirm **رمزُ واتسابٍ فرمزٌ جديد.**
func (s *Server) handlePinResetConfirm(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Code string `json:"code"`
		Pin  string `json:"pin"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.ResetPinConfirm(r.Context(), userIDFrom(r), req.Code, req.Pin, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "auth.pin_reset", "user", userIDFrom(r), map[string]any{})
	httpx.JSON(w, http.StatusOK, map[string]any{"reset": true})
}
