package server

// ══════════════════════════════════════════════════════════════════════
// **موافقةُ السائق على شروط العمل** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٧)
// ══════════════════════════════════════════════════════════════════════
//
// **في الشروط بندٌ جديد**: «أخذ زبون أو رقمه للتعامل برّا المنصّة يعني إيقاف الحساب»
// (`settings.DriverTermsClause`). **والسائقُ يوافق عليه في تطبيقه قبل أن يفتح دوامه**
// — موافقةٌ مسجَّلةٌ بالنسخة وبصمةِ النصّ الذي رآه والوقتِ ونسخةِ التطبيق
// (`terms_acceptances`، الهجرة ٠٥٠٥).
//
// # والتطبيقاتُ القديمةُ لا تُقفل
//
// **القفلُ لمن يملك شاشةَ الموافقة وحدَه** — تطبيقُ سائقٍ بناؤه `driverTermsMinBuild`
// فما فوق (ترويستا `X-RahalGo-Client`/`X-RahalGo-Version`). **وما دونه يفتح دوامَه
// كما كان**، **وأوّلُ ما يُرفع التطبيقُ الجديد يتحدّث الجميعُ إجباراً** فيوافقون.
//
// **والخروجُ من الدوام لا يُقفل**، **وما بيده من طلباتٍ لا يتأثّر** — فتحُ الدوام وحدَه.

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

// driverTermsMinBuild **أوّلُ بناءٍ لتطبيق السائق فيه شاشةُ الموافقة** — يساوي رقمَ
// بناء النسخة التي فيها `TermsSheet` (خطوةُ الإصدار).
const driverTermsMinBuild = 22

var (
	errDriverTermsRequired = httpx.NewError(http.StatusConflict,
		"driver_terms_required", "errors.driver_terms_required")
	errTermsVersionChanged = httpx.NewError(http.StatusConflict,
		"terms_version_changed", "errors.terms_version_changed")
)

// driverTermsState **النسخةُ المطلوبةُ وأعلى نسخةٍ وافق عليها السائق.**
type driverTermsState struct {
	Version  int64
	Accepted int64
}

// Required **أعليه أن يوافق؟** — نسخةٌ مطلوبةٌ (لا صفر) لم يوافق عليها بعد.
func (st driverTermsState) Required() bool {
	return st.Version > 0 && st.Accepted < st.Version
}

// driverTermsStateOf **الحالُ من مصدرٍ واحد** — يقرؤه البابُ والدوامُ و`/driver/me`.
func (s *Server) driverTermsStateOf(r *http.Request, uid string) (driverTermsState, error) {
	st := driverTermsState{Version: s.settings.GetInt(r.Context(), "drivers.terms_version")}
	err := s.pg.QueryRow(r.Context(), `
		SELECT COALESCE(max(version), 0) FROM terms_acceptances
		WHERE user_id = $1 AND role = 'driver'`, uid).Scan(&st.Accepted)
	return st, err
}

// driverTermsBlock **أيُمنع هذا من فتح دوامه؟** — تطبيقُ سائقٍ فيه شاشةُ الموافقة،
// **ولم يوافق على النسخة المطلوبة.** **بلا ترويسةٍ أو ببناءٍ أقدم: لا قفل.**
func (s *Server) driverTermsBlock(r *http.Request, uid string) bool {
	kind, build := requestAppBuild(r)
	if kind != "driver" || build < driverTermsMinBuild {
		return false
	}
	st, err := s.driverTermsStateOf(r, uid)
	if err != nil {
		// **عطبُ القراءة لا يقفل دوامَ أحد** — يُسجَّل والدوامُ يُفتح.
		s.logger.Warn("شروطُ السائق: تعذّرت قراءةُ الموافقة", "user", uid, "error", err)
		return false
	}
	return st.Required()
}

// effectiveDriverTerms **النصُّ الذي يراه السائقُ ويوقّعه** — وفيه البندُ أبداً.
func (s *Server) effectiveDriverTerms(r *http.Request) string {
	return settings.EffectiveDriverTerms(s.settings.GetString(r.Context(), "page.driver_terms_text"))
}

// handleDriverTerms `GET /driver/terms` — النسخةُ وما وافق عليه والنصُّ والبند.
func (s *Server) handleDriverTerms(w http.ResponseWriter, r *http.Request) {
	st, err := s.driverTermsStateOf(r, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"version":          st.Version,
		"accepted_version": st.Accepted,
		"required":         st.Required(),
		"text":             s.effectiveDriverTerms(r),
		"clause":           settings.DriverTermsClause,
		// **وقوالبُ النصّ تُملأ في التطبيق** (`{name}` و`{phone}`) — كما في صفحة الشروط.
		"name":  s.settings.GetString(r.Context(), "platform.name"),
		"phone": s.settings.GetString(r.Context(), "platform.support_phone"),
	})
}

// handleDriverTermsAccept `POST /driver/terms/accept` — `{version}`.
//
// **والنسخةُ تُرسَل لا تُفترض**: من قرأ نسخةً ثمّ رُفعت وهو يقرأ لا يُسجَّل موافقاً على
// ما لم يرَ (`terms_version_changed`). **والموافقةُ الثانيةُ على النسخة نفسِها لا تُكرَّر.**
func (s *Server) handleDriverTermsAccept(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Version int64 `json:"version"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	uid := userIDFrom(r)
	st, err := s.driverTermsStateOf(r, uid)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if st.Version <= 0 {
		// **لا موافقةَ مطلوبة** — لا شيءَ يُسجَّل.
		httpx.JSON(w, http.StatusOK, map[string]any{"accepted_version": st.Accepted})
		return
	}
	if req.Version != st.Version {
		s.respondErr(w, errTermsVersionChanged)
		return
	}
	sum := sha256.Sum256([]byte(s.effectiveDriverTerms(r)))
	var build *int
	if kind, b := requestAppBuild(r); kind == "driver" && b > 0 {
		build = &b
	}
	tag, err := s.pg.Exec(r.Context(), `
		INSERT INTO terms_acceptances (user_id, role, version, text_sha256, app_build)
		VALUES ($1, 'driver', $2, $3, $4)
		ON CONFLICT (user_id, role, version) DO NOTHING`,
		uid, st.Version, hex.EncodeToString(sum[:]), build)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() > 0 {
		s.audit(r, "driver.terms_accepted", "user", uid, map[string]any{
			"version": st.Version, "text_sha256": hex.EncodeToString(sum[:]),
			"app_build": build, "clause": strings.TrimSpace(settings.DriverTermsClause),
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"accepted_version": st.Version})
}
