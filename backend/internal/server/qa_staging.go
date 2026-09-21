package server

// ══════════════════════════════════════════════════════════════════════
// **بابُ جلسةٍ للاختبار الحيّ — على التجهيز وحدَه، ويسقط مغلقاً في الإنتاج**
// ══════════════════════════════════════════════════════════════════════
//
// (لتمكين الاختبار الآليّ الحيّ لتطبيق الزبون على التجهيز بلا OTP يدويّ —
//  قرارُ المالك: قدرةٌ QA دائمةٌ على التجهيز.)
//
// # لماذا وكيف يُؤمَّن
//
// **لا يُسجَّل مسارُه إلّا على التجهيز** (`APP_ENV=staging` و`RAHALGO_STAGING=1`)
// — فلا وجودَ له في الإنتاج أصلاً (`server.go`). **ويُتحقَّق ثانيةً وقتَ
// الطلب** دفاعاً في العمق: بيئةٌ غيرُ التجهيز ⇒ `404` كأنّه غيرُ موجود.
//
// **ولا يكشف سرّاً**: يُصدر جلسةَ زبونٍ عاديّةً كأيّ دخولٍ ناجح، لزبون QA
// ثابتٍ مبذور — **بلا كلمةِ إنتاج، ولا سرِّ توقيع، ولا بابِ أدمن، ولا رمز.**
// **والإصدارُ يُسجَّل** (سطرُ تحذيرٍ) فيبقى مسموعاً.

import (
	"errors"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
)

// qaStagingPhone **زبونُ QA الثابت** — رقمٌ محجوزٌ للاختبار على التجهيز.
//
// **ومختارٌ بعيداً عن أرقام البذور** (الأدمن `+963999…`، الطاقم `+963955…`،
// وأرقامُ اختبارِ الأطوار `+96390000000x`) — **فلا يُصدَر بالخطأ سِمةُ حسابٍ
// مُمتاز.** وحارسُ الدور أدناه يمنع ذلك على كلّ حال.
const qaStagingPhone = "+963900555001"

// qaStagingEnabled **أعلى التجهيز نحن؟** — الشرطان معاً، لا أحدُهما.
func (s *Server) qaStagingEnabled() bool {
	return s.cfg.Env == "staging" && os.Getenv("RAHALGO_STAGING") == "1"
}

// handleQAStagingSession يُصدر جلسةَ زبون QA — على التجهيز وحدَه.
func (s *Server) handleQAStagingSession(w http.ResponseWriter, r *http.Request) {
	// **دفاعٌ في العمق**: ولو سُجّل المسارُ خطأً في غير التجهيز، يسقط مغلقاً.
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}

	phone, ok := identity.NormalizePhone(qaStagingPhone)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}

	// **يُبحَث عنه أولاً** — **فلا يُعاد منحُ الدور على القائم** (يُدرج
	// `granted_by` فارغاً فيسقط)، ولا يُنشأ إلّا مرّةً.
	var uid string
	err := s.pg.QueryRow(r.Context(), `SELECT id::text FROM users WHERE phone = $1`, phone).Scan(&uid)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		user, cerr := s.identity.EnsureUserWithRole(r.Context(), "", qaStagingPhone, "customer", "زبون الاختبار QA", "", clientIP(r))
		if cerr != nil {
			s.respondErr(w, cerr)
			return
		}
		uid = user.ID
	case err != nil:
		s.respondErr(w, err)
		return
	}

	// ══════════════════════════════════════════════════════════════════
	// **حارسُ الدور — لا تُصدَر جلسةٌ إلّا لزبونٍ محض** (لا أدمن ولا طاقم)
	// ══════════════════════════════════════════════════════════════════
	//
	// **دفاعٌ لو اصطدم رقمُ QA بحسابٍ مُمتازٍ مبذور** — **فلا يُصنَع توكنُ
	// أدمنٍ من بابِ الاختبار.** أيُّ دورٍ غيرِ `customer` ⇒ رفضٌ مغلق.
	rows, rerr := s.pg.Query(r.Context(), `SELECT role_code FROM user_roles WHERE user_id = $1::uuid`, uid)
	if rerr != nil {
		s.respondErr(w, rerr)
		return
	}
	privileged := false
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			rows.Close()
			s.respondErr(w, err)
			return
		}
		if role != "customer" {
			privileged = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	if privileged {
		s.logger.Warn("QA staging session REFUSED — account carries a privileged role", "user", uid)
		s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_not_customer_only", "errors.forbidden"))
		return
	}

	// **جلسةٌ عاديّة** — عائلةُ الجلسة القائمةُ إن وُجدت وإلّا جديدة.
	sid := s.identity.ActiveSessionID(r.Context(), uid)
	if sid == "" {
		sid = uuid.NewString()
	}
	res, err := s.identity.IssueForUserID(r.Context(), uid, sid, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA staging session issued (staging-only)", "user", uid, "ip", clientIP(r))
	httpx.JSON(w, http.StatusOK, res)
}
