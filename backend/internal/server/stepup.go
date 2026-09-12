package server

// ══════════════════════════════════════════════════════════════════════
// **تأكيدُ الفعل الحسّاس بكلمةِ صاحبه** — `ADG-3` · `AQ-1`
// ══════════════════════════════════════════════════════════════════════
//
// # العقدُ ثلاثةُ شروطٍ لا بدائل
//
//	جلسةٌ صالحة   `R16`   — تُقاس في `RequireAuth`
//	قدرةٌ قائمة   `ADG-2` — تُقاس في `enforceAdminPolicy`
//	إثباتُ تأكيدٍ لهذا الفعل بعينه — يُقاس هنا
//
// **وترتيبُ الوسائط هو ترتيبُ الشروط** — **فمن لا جلسةَ له لا يبلغ
// القدرة، ومن لا قدرةَ له لا يبلغ التأكيد.**
//
// # ولماذا وسيطٌ لا نداءٌ في كلّ معالِج
//
// **معالِجٌ نُسي فيه النداءُ يمضي بلا تأكيدٍ ولا يُكتشَف** —
// **والوسيطُ يقرأ المعجمَ فلا ينسى.** **واللوحةُ ليست حدَّ الأمن**:
// نداءٌ مباشرٌ بلا إثباتٍ يُردّ كما يُردّ من اللوحة.

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
)

// stepUpHeader ترويسةُ الإثبات — **معرّفٌ لا سرّ.**
//
// **والإثباتُ لا يُغني عن جلسةٍ ولا قدرة** — فلا خطرَ في معرّفٍ يُقرأ،
// **وحدُّه أنّه لصاحبه ولجلسته ولفعله.**
const stepUpHeader = "X-Step-Up"

// errStepUpRequired **يلزم تأكيدٌ بكلمة الحساب.**
var errStepUpRequired = httpx.NewError(http.StatusForbidden,
	"step_up_required", "errors.step_up_required")

// errStepUpInvalid **لا إثباتَ صالحٌ لهذا الفعل.**
//
// **ولا يُفصَّل السبب** — منتهٍ أم مستهلَكٌ أم لغيره.
var errStepUpInvalid = httpx.NewError(http.StatusForbidden,
	"step_up_invalid", "errors.step_up_invalid")

// sessionIDFrom عائلةُ الجلسة التي حملت الطلب — من السياق لا من الطلب.
func sessionIDFrom(r *http.Request) string {
	sid, _ := r.Context().Value(ctxSID).(string)
	return sid
}

// requireStepUp يمنع فعلاً حسّاساً بلا إثباتٍ لهذا النداء بعينه.
func (s *Server) requireStepUp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pattern := adminPattern(r.URL.Path)
		act, ok := authz.LookupSensitive(r.Method, pattern)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		// **والجسمُ يُقرأ ويُعاد** — المعالِجُ بعدَنا يقرؤه كما لو
		// لم يُمَسّ.
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			s.respondErr(w, errValidation)
			return
		}
		_ = r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(body))

		if !s.sensitiveNow(r.URL.Path, act, body) {
			next.ServeHTTP(w, r)
			return
		}

		id := strings.TrimSpace(r.Header.Get(stepUpHeader))
		if id == "" {
			// **والردُّ يقول ما يُراد تأكيدُه** — **فتعرض اللوحةُ
			// نصَّ الفعل والهدفَ قبل أن تسأل الكلمة**، **ولا
			// تخترعه من مسارٍ تقرؤه بنفسها.**
			s.respondStepUpRequired(w, act, r.URL.Path)
			return
		}
		// **والفاعلُ وجلستُه من سياق الخادم لا من الطلب** — **ولا
		// يُصدَّق زبونٌ يقول من هو.**
		if err := s.identity.ClaimStepUp(r.Context(), userIDFrom(r), sessionIDFrom(r),
			identity.StepUpScope(r.Method, r.URL.Path, authz.Material(act, body))); err != nil {
			s.logger.Info("التأكيد: إثباتٌ غيرُ صالح",
				"action", act.Action, "user", userIDFrom(r))
			httpx.Error(w, errStepUpInvalid)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// respondStepUpRequired **يطلب التأكيدَ ويصف الفعلَ المطلوب.**
//
// **ولا تُرسَل حمولةُ الطلب في الردّ** — الفعلُ والهدفُ لا غير.
// **والظرفُ ظرفُ خطأٍ لا ظرفُ نجاح** — **و`httpx.JSON` تغلّف في
// `data`، فيصير التحدّي مدفوناً طبقةً ولا يراه العميل.** (وقع في
// الإنتاج ٢٠٢٦-٠٩-١٢: ٤٠٣ بجسمٍ صحيحٍ مغلَّفٍ ⇒ لم تُفتح النافذةُ
// وقُرئ «حدث خطأ غير متوقع».)
//
// **والرمزُ والمفتاحُ من `errStepUpRequired` لا حروفاً تُعاد** —
// **وكانا مكتوبين مرّتين، فشاخت إحداهما.**
func (s *Server) respondStepUpRequired(w http.ResponseWriter, act authz.Sensitive, path string) {
	httpx.ErrorWith(w, errStepUpRequired, map[string]any{
		"step_up": map[string]any{
			"action":      act.Action,
			"target_type": act.TargetType,
			"target_id":   authz.Target(act, path),
		},
	})
}

// sensitiveNow **أهذا النداءُ بعينه حسّاس؟**
//
// **والاسمُ لا يكفي في فعلين**: **تبديلُ إعدادٍ** يُصنَّف بمفتاحه،
// **وتبديلُ حالِ حسابٍ** يُصنَّف بحاله. **والمصنِّفان قائمان** —
// `criticalSettingKey` من دورةِ ٢١، **وفرقُ الإيقاف عن الحظر من
// دورةِ ١٧** — **ولا يُكتب أحدُهما ثانيةً.**
func (s *Server) sensitiveNow(path string, act authz.Sensitive, body []byte) bool {
	switch act.Conditional {
	case "":
		return true
	case authz.CondSettingSensitivity:
		return criticalSettingKey(lastSegment(path))
	case authz.CondStatusIsStrong:
		var in struct {
			Status string `json:"status"`
		}
		_ = json.Unmarshal(body, &in)
		// **والإيقافُ العاديُّ يمضي** — **وهو عقدُ دورةِ ١٧**:
		// معلَّقٌ يُتمّ طلبَه، ومحظورٌ ينقطع.
		return in.Status == "blocked" || in.Status == "deleted"
	}
	return true
}

// lastSegment آخرُ مقطعٍ في المسار — مفتاحُ الإعداد.
func lastSegment(p string) string {
	i := strings.LastIndex(strings.TrimRight(p, "/"), "/")
	if i < 0 {
		return ""
	}
	return strings.TrimRight(p, "/")[i+1:]
}

// ══════════════════════════════════════════════════════════════════════
// **بابُ الإصدار**
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا يُرسِل الزبونُ النداءَ الذي ينويه
//
// **الإثباتُ مربوطٌ بما نوى** — **فلا بدّ أن يقوله.** **وقولُه لا
// يُصدَّق تخويلاً**: القدرةُ تُقاس من الحقيقة الموثوقة، **وكلُّ ما
// يفعله قولُه أن يضيّق إثباتَه على نفسه.**
//
// # ولا كلمةَ تُخزَّن ولا تُدقَّق
//
// **تُتحقَّق وتُنسى** — ولا تدخل سجلّاً ولا رسالةَ خطأ.
func (s *Server) handleStepUp(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Method   string          `json:"method"`
		Path     string          `json:"path"`
		Body     json.RawMessage `json:"body"`
		Password string          `json:"password"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	path := strings.TrimSpace(req.Path)
	if method == "" || !strings.HasPrefix(path, "/api/v1/admin/") || req.Password == "" {
		s.respondErr(w, errValidation)
		return
	}

	pattern := adminPattern(path)
	act, ok := authz.LookupSensitive(method, pattern)
	if !ok {
		// **ولا يُصدَر إثباتٌ لفعلٍ لا يحتاجه** — **وإلّا صار البابُ
		// آلةَ تخمينِ كلمات.**
		s.respondErr(w, errValidation)
		return
	}

	// **والقدرةُ تُقاس هنا أيضاً** — **فلا يُخمَّن على كلمةِ حسابٍ
	// من طريقِ فعلٍ لا يملكه أصلاً.**
	//
	// **والمستثنى يُحسَم كما يُحسَم في معالِجه** — `/settings/{key}`
	// قدرتُها تتبع المفتاح، **ولا يُكتب التصنيفُ ثانيةً.**
	need, hasPolicy := authz.LookupAdmin(method, pattern)
	if !hasPolicy {
		if _, ex := authz.IsExempt(pattern); ex && act.Action == "admin.setting_update" {
			need, hasPolicy = settingCapability(lastSegment(path)), true
		}
	}
	if !hasPolicy || !s.hasCapability(r, need) {
		httpx.Error(w, errForbiddenCap)
		return
	}

	body := []byte(req.Body)
	if len(body) == 0 {
		body = []byte("{}")
	}
	if !s.sensitiveNow(path, act, body) {
		// **وفعلٌ لا يحتاج تأكيداً لا يُصدَر له إثبات.**
		s.respondErr(w, errValidation)
		return
	}

	grant, err := s.identity.IssueStepUp(r.Context(), userIDFrom(r), sessionIDFrom(r),
		act.Action, act.TargetType, authz.Target(act, path),
		identity.StepUpScope(method, path, authz.Material(act, body)), req.Password)
	if err != nil {
		s.logger.Info("التأكيد: تعذّر الإصدار",
			"action", act.Action, "user", userIDFrom(r))
		// **وكلمةٌ خاطئةٌ لا تُفرَّق عن حسابٍ بلا كلمة** في الردّ.
		httpx.Error(w, errStepUpInvalid)
		return
	}
	httpx.JSON(w, http.StatusOK, grant)
}
