package server

/*
**الإنذارُ يبلغ كلَّ دور — ويصل صاحبَه.**

(شكوى المالك ٢٠٢٦-٠٨-٠٩: «وجّه إنذار للسائق… بس ما وصل الإنذار» · «اجعله لكلّ
 الأدوار حتّى الزبون — ممكن يكون تعامل بسوء مع السائق، بالنهاية كمان السائق
 بشر ويقدّم خدمة».)

# ما كان

**جدولُ الإنذارات للمتاجر وحدَها.** فحين وُجّه إنذارٌ لسائقٍ **لم يُسجَّل شيء**
— كانت ملاحظةً نصّيّةً في سجلّ المتابعة **تصل الزبونَ لا السائق.**

**وإنذارٌ لا يُسجَّل لا يُعدّ ولا يُحتجّ به يومَ الحظر** — ولا يعرف صاحبُه
أنّه أُنذر.

# وجدولٌ واحدٌ لا أربعة

**أربعةُ جداولَ لأربعة أدوارٍ أربعُ نسخٍ من قاعدةٍ واحدة**: يُضاف حقلٌ في
إحداها ويُنسى في الثلاث. **والإنذارُ إنذارٌ مهما كان دورُ صاحبه.**

# ويُنسب إلى حسابٍ لا إلى كيان

**المتجرُ كيانٌ وصاحبُه شخص** — ومن أُنذر هو من يقرأ، **والحسابُ هو ما يُحظَر
لا اللافتة.**
*/

import (
	"github.com/servacode/rahalgo/backend/internal/support"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

type warningItem struct {
	ID        string    `json:"id"`
	Reason    string    `json:"reason"`
	Note      string    `json:"note"`
	Role      string    `json:"role_code"`
	OrderNum  *int64    `json:"order_number"`
	CreatedAt time.Time `json:"created_at"`
}

const warningItemSelect = `
	SELECT w.id::text, w.reason, w.note, w.role_code, o.number, w.created_at
	FROM warnings w
	LEFT JOIN orders o ON o.id = w.order_id`

func (s *Server) scanWarningItems(w http.ResponseWriter, r *http.Request, q string, args ...any) {
	rows, err := s.pg.Query(r.Context(), q, args...)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []warningItem{}
	for rows.Next() {
		var x warningItem
		if err := rows.Scan(&x.ID, &x.Reason, &x.Note, &x.Role, &x.OrderNum, &x.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"warnings": out})
}

// handleMyWarnings **إنذاراتي — يراها صاحبُها.**
//
// **ومن أُنذر ولا يعلم لا يُصلح شيئاً**: يُحظَر يوماً فيُفاجأ، **ويظنّ الظلمَ
// حيث كان خبر.**
func (s *Server) handleMyWarnings(w http.ResponseWriter, r *http.Request) {
	s.scanWarningItems(w, r, warningItemSelect+`
		WHERE w.user_id = $1 ORDER BY w.created_at DESC LIMIT 50`, userIDFrom(r))
}

// handleAdminUserWarnings **ما على هذا الحساب** — كما تراه العمليات.
func (s *Server) handleAdminUserWarnings(w http.ResponseWriter, r *http.Request) {
	// ══════════════════════════════════════════════════════════════════
	// **وإنذاراتُ متجره معها — بشارةٍ تفرّقها**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٦: «ابدأ بملفّ صاحب المتجر».)
	//
	// **جدولان لا واحد**: `warnings` على الحساب و`merchant_warnings` على
	// المتجر — **ولكلٍّ أسبابُه ومنطقُه.** وكان الملفُّ يعرض الأوّلَ وحدَه.
	//
	// **ومتجرٌ يُحظر بعد أربع مخالفات** — **فمن راجع صاحبَه ليقرّر قرأ
	// صفراً** وهو على ثلاثٍ من أربع.
	//
	// **والدورُ يقول أيَّهما هو** (`role_code`): حسابٌ أم متجر — **فلا
	// يُخلط إنذارُ إنسانٍ بإنذارِ كيان.**
	s.scanWarningItems(w, r, warningItemSelect+`
		WHERE w.user_id = $1
		UNION ALL
		SELECT mw.id::text, mw.reason, mw.note, 'store', o.number, mw.created_at
		FROM merchant_warnings mw
		JOIN merchants m ON m.id = mw.merchant_id
		LEFT JOIN orders o ON o.id = mw.order_id
		WHERE m.owner_user_id = $1
		ORDER BY created_at DESC LIMIT 100`, chi.URLParam(r, "id"))
}

// issueWarning **القيدُ نفسُه — بابان يناديانه ولا يفترقان.**
//
// **العنوانان اثنان بقصد**: `‎/users/{id}/warnings` يخاطب حساباً،
// و`‎/merchants/{id}/warnings` يخاطب متجراً — **وشاشةُ المتاجر تعرف متجرَها
// لا صاحبَه.**
//
// **والتنفيذُ واحد**: لو كُتب مرّتين لَافترقا — **يُضاف إشعارٌ في أحدهما
// ويُنسى في الآخر**، فيُنذَر سائقٌ فيعلم ويُنذَر متجرٌ فلا يعلم.
func (s *Server) issueWarning(w http.ResponseWriter, r *http.Request,
	userID, reason, note string, orderID, ticketID *string) {
	// **ودورُه وقتَ الإنذار يُثبَّت** — لا يُشتقّ عند القراءة.
	//
	// **فمن كان سائقاً ثمّ صار مندوباً** تُقرأ إنذاراتُه القديمةُ باسم دوره
	// الجديد، **ويبدو المندوبُ سيّئَ السجلّ في عملٍ لم يعمله.**
	var role string
	if err := s.pg.QueryRow(r.Context(), `
		SELECT COALESCE((SELECT role_code FROM user_roles WHERE user_id = $1
		                 ORDER BY granted_at LIMIT 1), 'customer')`, userID).
		Scan(&role); err != nil {
		s.respondErr(w, err)
		return
	}

	// ══════════════════════════════════════════════════════════════════
	// **والسببُ من القائمة لا نصٌّ حرّ**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٥.)
	//
	// **والحكمُ هنا لا في الشاشة** — من نادى الواجهةَ البرمجيّة
	// مباشرةً تجاوز قائمةَ الاختيار.
	//
	// **والأسبابُ تتبع الدور**: «رفضُ الاستلام» لا يُنذَر به سائق.
	if !support.ValidWarnReason(reason, role) {
		s.respondErr(w, errValidation)
		return
	}

	var id string
	if err := s.pg.QueryRow(r.Context(), `
		INSERT INTO warnings (user_id, role_code, reason, note, order_id, ticket_id, issued_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id::text`,
		userID, role, reason, clip(note, 500), orderID, ticketID,
		userIDFrom(r)).Scan(&id); err != nil {
		s.respondErr(w, err)
		return
	}

	// **ويصل صاحبَه — وهو ما لم يكن.**
	//
	// **إنذارٌ لا يبلغ من أُنذر ليس إنذاراً**: هو سطرٌ في دفترٍ يُقرأ يومَ
	// الحظر، **ولا فرصةَ لصاحبه أن يُصلح.** (وهي شكوى المالك بعينها.)
	// ══════════════════════════════════════════════════════════════════
	// **والإشعارُ يقول السببَ لا العنوانَ وحدَه**
	// ══════════════════════════════════════════════════════════════════
	//
	// (شكوى المالك ٢٠٢٦-٠٨-١٥: «وصل الإنذارُ للزبون بس لم يعرف
	//  سببَه».)
	//
	// **وكان الجسدُ «الملاحظة» وحدَها وهي اختياريّة** — فمن أنذر
	// بسببٍ بلا ملاحظةٍ أرسل **عنواناً بلا جسد**: «إنذارٌ على حسابك»
	// وكفى. **وإنذارٌ لا يُعرف سببُه لا يُصحَّح**، إنّما يُخيف.
	//
	// **والاسمُ عربيٌّ لا رمز**: الإشعارُ يُخزَّن نصّاً ويُقرأ كما
	// خُزّن، **فترجمتُه في الشاشة تأتي بعد فوات الأوان.**
	body := support.WarnReasonAr(reason)
	if n := strings.TrimSpace(note); n != "" {
		body += " — " + n
	}
	// **ونوعُه `account` لا `order`** — لا طلبَ فيه، **ومن صنّفه
	// طلباً خلطه بأخبارِ طلباته فضاع بينها.**
	//
	// **وبلا وجهة — لأنّه لا صفحةَ تعرض إنذاراتِ حساب.**
	//
	// **وكانت الوجهةُ `/portal/complaints` للجميع** — بوّابةَ متجرٍ
	// يصلها الزبون فيقف على بابٍ ليس له. **ثمّ جعلتُها `/portal/warnings`
	// فكانت أسوأ**: لا مسارَ بهذا الاسم عند سائقٍ ولا مندوب، **فتُفتح
	// أربعمئةٌ وأربعة.** وصفحةُ الإنذارات موجودةٌ للمتجر وحدَه
	// (`/store/reviews`) **وهي تقرأ إنذاراتِ المتجر لا إنذاراتِ صاحبه.**
	//
	// **وخبرٌ بلا رابطٍ أصدقُ من رابطٍ يكذب** — والنصُّ يقول كلَّ شيء.
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: userID, Kind: notifications.KindAccount,
		Title: notifTitles.warningOnYou, Body: clip(body, 200),
		Entity: "user", EntityID: userID,
	})
	s.audit(r, "ops.warning_issued", "user", userID, map[string]any{"reason": reason})
	s.touch("user", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

// handleIssueUserWarning **إنذارٌ على حسابٍ — أيَّ دورٍ كان.**
func (s *Server) handleIssueUserWarning(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Reason   string  `json:"reason"`
		Note     string  `json:"note"`
		OrderID  *string `json:"order_id"`
		TicketID *string `json:"ticket_id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والسببُ إلزاميّ**: إنذارٌ بلا سببٍ لا يُصحَّح ولا يُحتجّ به.
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		s.respondErr(w, errValidation)
		return
	}
	s.issueWarning(w, r, chi.URLParam(r, "id"), reason, req.Note, req.OrderID, req.TicketID)
}

// handleIssueMerchantWarning **إنذارٌ على متجرٍ — يُقيَّد على صاحبه.**
//
// **والمتجرُ كيانٌ وصاحبُه شخص**: الحسابُ هو ما يُحظَر ويُشعَر، **لا اللافتة.**
func (s *Server) handleIssueMerchantWarning(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Reason string `json:"reason"`
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		s.respondErr(w, errValidation)
		return
	}
	var owner *string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT owner_user_id::text FROM merchants WHERE id = $1`,
		chi.URLParam(r, "id")).Scan(&owner); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	// **ومتجرٌ بلا صاحبٍ لا يُنذَر** — لا حسابَ يقرأ ولا حسابَ يُحظَر.
	if owner == nil {
		s.respondErr(w, errValidation)
		return
	}
	s.issueWarning(w, r, *owner, reason, req.Note, nil, nil)
}

// handleWarnReasons **أسبابُ الإنذار لهذا الحساب — بدوره.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٥: «يجب أن يكون هناك أسبابٌ جاهزةٌ للإنذار».)
//
// **وتتبع الدورَ لا تُسرَد كلُّها**: «رفضُ الاستلام» لا يُنذَر به سائق،
// **ومن رأى سبباً لا يخصّ من أمامه اختار أقربَه** فكُتب سببٌ لا يصف ما وقع.
func (s *Server) handleWarnReasons(w http.ResponseWriter, r *http.Request) {
	var role string
	if err := s.pg.QueryRow(r.Context(), `
		SELECT COALESCE((SELECT role_code FROM user_roles WHERE user_id = $1
		                 ORDER BY granted_at LIMIT 1), 'customer')`,
		chi.URLParam(r, "id")).Scan(&role); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"reasons": support.WarnReasonsFor(role)})
}
