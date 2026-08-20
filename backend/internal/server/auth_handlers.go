package server

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var errValidation = httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")

func decode[T any](r *http.Request) (*T, error) {
	var v T
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(&v); err != nil {
		return nil, errValidation
	}
	return &v, nil
}

// respondErr يحول أخطاء المجال إلى استجابة موحدة، ويخفي التفاصيل الداخلية.
func (s *Server) respondErr(w http.ResponseWriter, err error) {
	var appErr *httpx.AppError
	if errors.As(err, &appErr) {
		httpx.Error(w, appErr)
		return
	}
	// ══════════════════════════════════════════════════════════════════
	// **ومعرّفٌ لا شكلَ له «غير موجود» — لا «عطبٌ في الخادم»**
	// ══════════════════════════════════════════════════════════════════
	//
	// **`22P02` تعني أنّ النصَّ ليس معرّفاً أصلاً** — «not-a-uuid» أو `null`
	// أو بقيّةُ رابطٍ مقصوص. **وهذه حالُ الطالب لا حالُ المنصّة.**
	//
	// **وقِيس ٢٠٢٦-٠٨-١٠**: من ٩٧ مساراً فيه معرّف، **٣٧ ترد خمسَمئة** على
	// معرّفٍ فاسد. وثلاثةُ أضرارٍ لا واحد:
	//
	//   - **الشاشةُ تقول «خطأٌ داخليّ» ومكانُها «غير موجود»** — فيظنّ
	//     المستخدمُ أنّ المنصّة تعطّلت ويعيد ويعيد.
	//   - **وسجلُّ الأخطاء يمتلئ بما ليس خطأً** — فيُدفن العطبُ الحقيقيّ
	//     بين مئةِ سطرٍ من روابطَ مقصوصة.
	//   - **ومن أراد أن يعرف أيّ نقطةٍ تنهار يجرّب حرفاً واحداً** ويقرأ
	//     الجواب من رمز الحالة.
	//
	// **ويُعالَج في المركز لا في سبعةٍ وثلاثين معالِجاً** — حارسٌ في كلٍّ
	// منها يُنسى في الثامن والثلاثين، **وهذه بالضبط علّةُ تفرّقِها اليوم**:
	// بعضُها يحرس بـ`isUUID` وبعضُها لا.
	//
	// **ولا يُبتلع خطأٌ آخر**: الرمزُ محدَّدٌ بعينه، وما عداه يبقى خمسَمئة.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		httpx.Error(w, httpx.ErrNotFound)
		return
	}
	s.logger.Error("internal error", "error", err)
	httpx.Error(w, httpx.ErrInternal)
}

func (s *Server) handleOTPRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.RequestOTP(r.Context(), req.Phone, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": true})
}

func (s *Server) handleOTPVerify(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.identity.VerifyOTP(r.Context(), req.Phone, req.Code, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handlePasswordLogin(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.identity.LoginPassword(r.Context(), req.Phone, req.Password, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		RefreshToken string `json:"refresh_token"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.identity.Refresh(r.Context(), req.RefreshToken, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		RefreshToken string `json:"refresh_token"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.Logout(r.Context(), req.RefreshToken, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"logged_out": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := s.identity.Me(r.Context(), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (s *Server) handleSetPassword(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Password        string `json:"password"`
		CurrentPassword string `json:"current_password"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.SetPassword(r.Context(), userIDFrom(r), req.Password, req.CurrentPassword, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

/*
**عنوانُ العميل بلا منفذ.**

(كشفه قياسٌ حيٌّ ٢٠٢٦-٠٨-٠٧: أُرسل أربعةٌ وثلاثون رمزاً من مضيفٍ واحدٍ

	والحدُّ ثلاثون — **ولم يُمنع أحدُها.**)

كانت تردّ `r.RemoteAddr` كما هو: `127.0.0.1:54321`. **والمنفذُ عابرٌ يتبدّل
مع كلّ اتّصال**، فمفتاحُ الحدّ يتبدّل معه ولا يتراكم عدٌّ أبداً.

**ولا يخصّ الرموزَ وحدَها**: `loginMaxPerIP` مبنيٌّ على هذه الدالّة —
**فحدُّ محاولات الدخول للعنوان كان مكسوراً منذ كُتب.**

**ولم يره اختبارُ وحدةٍ قطّ**: الاختباراتُ تمرّر عنواناً نظيفاً نصّاً فتُصدّق
ما لا يقع. **وحدَه النداءُ الحيُّ كشفه.**

**ولم يظهر في الإنتاج** لأنّ `middleware.RealIP` تعيد كتابة `RemoteAddr` من
`X-Forwarded-For` حين توجد الترويسة — أي خلف وسيطٍ عكسيّ. **فالحدُّ يعمل خلف
الوسيط ويسقط في النداء المباشر.**

**والقسمةُ على النقطتين لا تكفي**: عنوانُ IPv6 فيه نقطتان كثيرة —
`[2001:db8::1]:44300`. و`net.SplitHostPort` تعرف الفرق.
*/
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// لا منفذَ فيه — وهو ما تكتبه `RealIP` خلف الوسيط.
		return r.RemoteAddr
	}
	return host
}

// handleMyLogins آخر دخولات الحساب — للمستخدم نفسه (شفافية أمان "هل كان هذا أنت؟").
func (s *Server) handleMyLogins(w http.ResponseWriter, r *http.Request) {
	// **وصفحةٌ محدودةٌ بعدٍّ** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).
	//
	// **وهذه الشاشةُ تسأل «هل كان هذا أنت؟»** — **ومن سُرق حسابُه يبحث عن
	// دخولٍ غريبٍ قد يكون قبل عشرين محاولة.** وعشرون صامتةٌ تُخفيه عنه
	// **في الشاشة التي بُنيت ليجده فيها.**
	pg := pagingOf(r, 20)
	const loginFilter = `
		FROM audit_log
		WHERE actor_user_id = $1
		  AND action IN ('auth.otp_login', 'auth.password_login', 'auth.password_failed')`
	var count int
	if err := s.pg.QueryRow(r.Context(), `SELECT count(*)`+loginFilter,
		userIDFrom(r)).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT action, COALESCE(ip, ''), created_at`+loginFilter+`
		ORDER BY id DESC LIMIT $2 OFFSET $3`, userIDFrom(r), pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type login struct {
		Action    string    `json:"action"`
		IP        string    `json:"ip"`
		CreatedAt time.Time `json:"created_at"`
	}
	out := []login{}
	for rows.Next() {
		var l login
		if err := rows.Scan(&l.Action, &l.IP, &l.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, l)
	}
	httpx.JSON(w, http.StatusOK, paged("logins", out, count, pg))
}

// --- استعادة كلمة المرور: رمز على الهاتف ثم كلمة مرور جديدة ---

func (s *Server) handleResetRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.RequestPasswordReset(r.Context(), req.Phone, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": true})
}

func (s *Server) handleResetConfirm(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone    string `json:"phone"`
		Code     string `json:"code"`
		Password string `json:"password"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.identity.ConfirmPasswordReset(r.Context(), req.Phone, req.Code, req.Password, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

// --- إنشاء حساب زبون: رمز تأكيد ثم اسم وكلمة مرور. الزبون فقط، لا دور آخر. ---

func (s *Server) handleSignupRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.RequestSignup(r.Context(), req.Phone, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": true})
}

// handleSignupVerify يتحقّق من الرمز قبل عرض نموذج البيانات.
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا تظهر المعلومات إلّا بعد التحقّق من الرمز».)
//
// **ولا يستهلك الرمز**: صاحبُه سيضغط «إنشاء حساب» بعد دقيقةٍ بالرمز نفسِه.
func (s *Server) handleSignupVerify(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.VerifySignupCode(r.Context(), req.Phone, req.Code); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"verified": true})
}

// handleResetVerify يتحقّق من رمز الاستعادة قبل عرض نموذج الكلمة الجديدة.
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٦.) **ولا يستهلك الرمز** — يُستهلك عند التأكيد.
func (s *Server) handleResetVerify(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.VerifyResetCode(r.Context(), req.Phone, req.Code); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"verified": true})
}

func (s *Server) handleSignupConfirm(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone    string `json:"phone"`
		Code     string `json:"code"`
		FullName string `json:"full_name"`
		Password string `json:"password"`
		// Ref رمزُ من دعاه — **اختياريّ**، ومن سجّل بلا دعوةٍ حسابُه كامل.
		Ref string `json:"ref"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.identity.ConfirmSignup(r.Context(), req.Phone, req.Code,
		strings.TrimSpace(req.FullName), req.Password, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// **والنسبُ بعد إنشاء الحساب لا قبله** — وحسابٌ لم يُنشأ لا يُنسب لأحد.
	//
	// **ورمزٌ خاطئٌ لا يُسقط تسجيلاً**: من كتب حرفاً زائداً في الرابط يفتح
	// حسابَه ويُحرَم المكافأةَ وحدَها، **ولا يُردّ على بابٍ قطعه كلَّه.**
	if req.Ref != "" && res != nil && res.User.ID != "" {
		if err := s.referrals.Attach(r.Context(), res.User.ID, req.Ref); err != nil {
			s.logger.Warn("الدعوة: تعذّر النسب", "code", req.Ref, "error", err)
		}
	}

	if res != nil && res.User.ID != "" {
		// ══════════════════════════════════════════════════════════════
		// **ورمزُ التسجيل وصل واتساب — فالحسابُ موثَّق**
		// ══════════════════════════════════════════════════════════════
		//
		// (قرارُ المالك ٢٠٢٦-٠٨-١٨: «بما أنّ الزبون سجّل ووصله رمزٌ على
		//  الواتس لإكمال عمليّة التسجيل، يُعتبر أنّه وثّق حسابَه — لا
		//  يوجد داعٍ لإعادة توثيق الحساب».)
		//
		// **والمحرّكُ كان يفرّق بين الرقمين عمداً**: رقمُ الدخول هويّة،
		// ورقمُ واتساب قناةُ تواصل — **وقد يفترقان** (خطٌّ أرضيّ، شريحةٌ
		// بلا واتساب، رقمُ عملٍ آخر).
		//
		// **وقرارُ المالك ألغى الافتراق** (٢٠٢٦-٠٨-١٤: «رقمٌ واحدٌ
		// للحساب وعليه واتساب، ولا رقمَ واتساب منفصل») — **فالتفريقُ
		// صار يطلب من الزبون أن يُثبت الرقمَ نفسَه مرّتين.**
		//
		// **ومزوّدُ الرمز هو الشرط**: من أُرسل رمزُه رسالةً نصّيّةً يوماً
		// لم يُثبت واتسابَه، **فلا يُوسَم موثَّقاً بما لم يقع.**
		if s.cfg != nil && s.cfg.OTPProvider == "whatsapp" {
			if err := s.identity.MarkWhatsAppFromSignup(
				r.Context(), res.User.ID, req.Phone, clientIP(r),
			); err != nil {
				// **ولا يُسقط تسجيلاً** — حسابٌ تمّ ووسمٌ تأخّر
				// يُصلَح بتوثيقٍ يدويٍّ من «حسابي».
				s.logger.Warn("التوثيق: تعذّر وسمُ واتساب عند التسجيل",
					"user", res.User.ID, "error", err)
			}
		}

		// **والهديّةُ بعد الحساب لا قبله** — انظر `GrantSignupBonus`.
		s.referrals.GrantSignupBonus(r.Context(), res.User.ID, res.User.ID)
	}
	httpx.JSON(w, http.StatusOK, res)
}

// handleHandoff ينشئ رمز تسليم لمرّة واحدة (SSO): يفتح المستخدم تطبيقاً آخر مسجّلاً
// بلا كلمة مرور. الرمز قصير العمر (60ث) ويُستهلك مرّة واحدة.
func (s *Server) handleHandoff(w http.ResponseWriter, r *http.Request) {
	code, _, err := auth.NewOpaqueToken()
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// نمرّر عائلة جلسة المصدر مع الرمز: الانتقال بين لوحاتنا امتداد لنفس الجلسة،
	// فلا تُبطل اللوحة الجديدة جلسة اللوحة التي جاء منها.
	uid := userIDFrom(r)
	payload := uid + "|" + s.identity.ActiveSessionID(r.Context(), uid)
	if err := s.rdb.Set(r.Context(), "sso:"+code, payload, 60*time.Second).Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"code": code})
}

// handleSSO يستبدل رمز التسليم بجلسة (تسجيل دخول بلا كلمة مرور عبر رمز موثوق).
func (s *Server) handleSSO(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Code string `json:"code"`
	}](r)
	if err != nil || req.Code == "" {
		s.respondErr(w, errValidation)
		return
	}
	payload, err := s.rdb.GetDel(r.Context(), "sso:"+req.Code).Result() // استهلاك لمرّة واحدة
	if err != nil || payload == "" {
		s.respondErr(w, errUnauthorized)
		return
	}
	uid, sessionID, _ := strings.Cut(payload, "|")
	if uid == "" {
		s.respondErr(w, errUnauthorized)
		return
	}
	res, err := s.identity.IssueForUserID(r.Context(), uid, sessionID, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

// handlePhoneChangeRequest يرسل رمزاً للرقم الجديد لتأكيد تغيير رقم الحساب.
func (s *Server) handlePhoneChangeRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.RequestPhoneChange(r.Context(), userIDFrom(r), req.Phone, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": true})
}

// handlePhoneChangeConfirm يتحقق من الرمز ويحدّث رقم الحساب.
func (s *Server) handlePhoneChangeConfirm(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.ConfirmPhoneChange(r.Context(), userIDFrom(r), req.Phone, req.Code, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

// --- توثيق رقم واتساب: رمز عبر واتساب يثبت أن الرقم حيّ ويملكه صاحب الحساب ---

func (s *Server) handleWhatsAppVerifyRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.RequestWhatsAppVerify(r.Context(), userIDFrom(r), req.Phone, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": true})
}

func (s *Server) handleWhatsAppVerifyConfirm(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.ConfirmWhatsAppVerify(r.Context(), userIDFrom(r), req.Phone, req.Code, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}

	// **وهنا يكتمل التسجيل** — لا عند فتح الحساب.
	//
	// **والواتسابُ هو الحارس**: بلاه تُفتح مئةُ حسابٍ في ساعةٍ بأرقامٍ تُشترى،
	// **فتُدفع مئةُ مكافأةٍ على مئةٍ لا وجودَ لها.** والتوثيقُ يجعل لكلّ حسابٍ
	// رقماً يملكه إنسان.
	//
	// **وتخرج صامتةً إن كان الوضعُ «عند أوّل طلب»** — أو إن لم يُدعَ أصلاً.
	// (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «عند التسجيل وتوثيق واتساب... وهيك تصير ثقة».)
	s.referrals.SettleOnSignup(r.Context(), userIDFrom(r), userIDFrom(r))

	// ══════════════════════════════════════════════════════════════════
	// **ولا رسالةَ «وُثّق رقمك»**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-٢٠ — حُذفت.)
	//
	// **كانت رسالةً ثانيةً بعد الرمز** تقول ما تقوله الشاشةُ نفسُها:
	// «وُثّق رقمك». **وشاشةٌ أمام عينه لا تحتاج هاتفاً يؤكّدها.**
	//
	// # وثمنُها كان أغلى من نفعها
	//
	// **رسالةٌ ثانيةٌ إلى رقمٍ جديدٍ لم يُراسَل من قبل** — **وهذا بعينه
	// نمطُ الإرسال الذي يُحظَر عليه بوتُ واتساب.** فتضاعف الخطرَ على
	// بابِ الدخول كلِّه، **مقابل جملةٍ يعرفها صاحبُها.**
	//
	// **ولم تكن تُرسَل أصلاً**: قناتُها بوّابةُ رسائلَ لم تُضبط قطّ —
	// **فحُذف ما لم يعمل يوماً ولم يفتقده أحد.**

	httpx.JSON(w, http.StatusOK, map[string]any{"verified": true})
}

// --- حذف الحساب نهائياً: رمز تأكيد على هاتف صاحبه ثم تجريد وإقفال ---

func (s *Server) handleDeleteAccountRequest(w http.ResponseWriter, r *http.Request) {
	if err := s.identity.RequestAccountDeletion(r.Context(), userIDFrom(r), clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": true})
}

func (s *Server) handleDeleteAccountConfirm(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Code string `json:"code"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.ConfirmAccountDeletion(r.Context(), userIDFrom(r), req.Code, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}
