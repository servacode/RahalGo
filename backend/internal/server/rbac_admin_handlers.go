package server

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **إدارةُ الأدوار والقدرات من اللوحة** — `ADG-2` · `AQ-1`
// ══════════════════════════════════════════════════════════════════════
//
// # العقد
//
//	NO-CODE FOR OPERATIONS · CODE FOR NEW CAPABILITIES
//
// **فيُقرَأ المعجمُ ولا يُحرَّر** — **والأدمنُ يُسنِد الموجودَ ولا
// يخترع قدرة.**
//
// # وحارسُها `roles.manage`
//
// **وهو لـ`owner_super_admin` وحدَه في المصفوفة الكانونيّة** —
// **ولا يُمنَح لغيره إلّا بيد من يملكه.**
//
// # ولا تصعيدَ ذاتيّاً
//
// **من لا يملك `roles.manage` لا يبلغ هذه المسارات أصلاً** — والسياسةُ
// المركزيّةُ تحرسها. **ومن يملكه يملك كلَّ شيءٍ بالتعريف**، فلا معنى
// لمنعه من منح ما يملكه.
//
// **والحدُّ الحقيقيُّ أن تبقى `roles.manage` عزيزةً** — **ولا تُبذَر
// لدورٍ وظيفيّ.**

// handleListRoles قائمةُ الأدوار وعددُ قدرات كلٍّ.
func (s *Server) handleListRoles(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT r.code, r.name_key,
		       COALESCE(array_agg(rc.capability_code ORDER BY rc.capability_code)
		                FILTER (WHERE rc.capability_code IS NOT NULL), '{}'),
		       (SELECT count(*) FROM user_roles ur WHERE ur.role_code = r.code)
		  FROM roles r
		  LEFT JOIN role_capabilities rc ON rc.role_code = r.code
		 GROUP BY r.code, r.name_key
		 ORDER BY r.code`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type roleOut struct {
		Code         string   `json:"code"`
		NameKey      string   `json:"name_key"`
		Capabilities []string `json:"capabilities"`
		Members      int      `json:"members"`
	}
	out := []roleOut{}
	for rows.Next() {
		var o roleOut
		if err := rows.Scan(&o.Code, &o.NameKey, &o.Capabilities, &o.Members); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, o)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleRoleDetail قدراتُ دورٍ بعينه.
func (s *Server) handleRoleDetail(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	var nameKey string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT name_key FROM roles WHERE code = $1`, code).Scan(&nameKey); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	var caps []string
	if err := s.pg.QueryRow(r.Context(), `
		SELECT COALESCE(array_agg(capability_code ORDER BY capability_code), '{}')
		  FROM role_capabilities WHERE role_code = $1`, code).Scan(&caps); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"code": code, "name_key": nameKey, "capabilities": caps,
	})
}

// handleListCapabilities **المعجمُ يُقرأ ولا يُحرَّر.**
func (s *Server) handleListCapabilities(w http.ResponseWriter, r *http.Request) {
	type capOut struct {
		Code        string `json:"code"`
		Description string `json:"description"`
	}
	out := make([]capOut, 0, authz.Count())
	for _, c := range authz.All() {
		out = append(out, capOut{Code: string(c), Description: authz.Describe(c)})
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleGrantCapability منحُ قدرةٍ لدور.
//
// **والمجهولةُ تُرفَض** — **فلا يُكتب في القاعدة نصٌّ لا يعني شيئاً.**
func (s *Server) handleGrantCapability(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	req, err := decode[struct {
		Capability string `json:"capability"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if !authz.Known(authz.Capability(req.Capability)) {
		s.respondErr(w, errValidation)
		return
	}
	if err := s.roleCapabilityTx(r, code, req.Capability, true); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"granted": true})
}

// handleRevokeCapability نزعُ قدرةٍ من دور.
func (s *Server) handleRevokeCapability(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	cap := chi.URLParam(r, "cap")
	if err := s.roleCapabilityTx(r, code, cap, false); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"revoked": true})
}

// roleCapabilityTx **تبديلُ سياسةِ أمنٍ وأثرُه في معاملةٍ واحدة.**
//
// **وهو تبديلُ تخويلٍ لا تبديلُ محتوى** — **فيُقيَّد كما تُقيَّد
// الأفعالُ الحسّاسة** (`AQ-4`). **ولا يُقبَل «أفضلَ جهد».**
func (s *Server) roleCapabilityTx(r *http.Request, role, capability string, grant bool) error {
	action := "admin.role_capability_revoke"
	if grant {
		action = "admin.role_capability_grant"
	}
	return s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		var err error
		if grant {
			actor := userIDFrom(r)
			_, err = q.Exec(ctx, `
				INSERT INTO role_capabilities (role_code, capability_code, granted_by)
				VALUES ($1, $2, $3::uuid) ON CONFLICT DO NOTHING`,
				role, capability, nilIfEmptyStr(actor))
		} else {
			_, err = q.Exec(ctx, `
				DELETE FROM role_capabilities
				 WHERE role_code = $1 AND capability_code = $2`, role, capability)
		}
		if err != nil {
			return err
		}
		return s.auditTx(ctx, q, r, action, "role", role,
			map[string]any{"capability": capability})
	})
}

// nilIfEmptyStr **فارغٌ يصير عدماً** — والعمودُ `uuid` لا يقبل نصّاً خاوياً.
func nilIfEmptyStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ══════════════════════════════════════════════════════════════════════
// **إنشاءُ دورٍ جديد** — الفعلُ الذي كان ناقصاً (دورةُ ٧٠ب-و١)
// ══════════════════════════════════════════════════════════════════════
//
// # لماذا لم يكن موجوداً
//
// **وجدولُ السياسة يحجز `/roles` لأيّ فعلٍ منذ `ADG-2`** — **والمُوجِّهُ
// لم يسجّل إلّا القراءة.** **فبابٌ محجوزٌ لم يُفتَح.**
//
// **وأثرُه عمليٌّ لا نظريّ**: **قدرةٌ تُضاف في الشيفرة لا تجد دوراً
// ضيّقاً يحملها** — **فإمّا تُمنَح لدورٍ عريضٍ يملك خمساً وعشرين قدرةً
// أصلاً، وإمّا يُكتب صفٌّ بيدٍ في القاعدة.** **وكلاهما نقضٌ لعقد
// «لا تعديلَ يدويٌّ للتشغيل».**
//
// # والقدراتُ تُسنَد ولا تُخترَع
//
// **ودورٌ يُنشَأ اليومَ لا يملك شيئاً** — **وهو الافتراضُ نفسُه في
// `ADG-1`.** **ثمّ تُمنَح قدراتُه واحدةً واحدةً بفعلٍ مؤكَّدٍ مستقلّ**،
// فيُقرأ في السجلّ ما مُنح ومتى ولمن.
//
// # واسمُ الدور نصٌّ لا مفتاحُ ترجمة
//
// **والأدوارُ المبذورةُ تحمل مفاتيحَ** (`roles.admin`) **لأنّ لها
// ترجماتٍ في المعجم.** **ودورٌ يُنشئه الأدمنُ اليومَ لا ترجمةَ له**،
// **فيُحفَظ اسمُه كما كتبه** — **والواجهةُ تعرض الترجمةَ إن وُجدت
// وإلّا عرضت النصَّ.** **ومن فرض مفتاحاً على اسمٍ حرٍّ أظهر
// `roles.xyz` لإنسانٍ يقرأ.**
//
// # ورمزُ الدور ضيّقٌ عمداً
//
// **وهو مفتاحٌ أوّليٌّ يدخل في `user_roles` و`role_capabilities`
// وفي كلّ سجلّ** — **فحرفٌ غريبٌ فيه يسكن الجداولَ سنين.**

// errRoleExists **رمزُ الدور مأخوذ** — **ولا يُكتَب فوق دورٍ قائم**:
// **إنشاءٌ صامتٌ فوق موجودٍ يمنح قدراتِ غيرِه لمن أنشأه.**
var errRoleExists = httpx.NewError(http.StatusConflict,
	"role_exists", "errors.role_exists")

// roleCodeRe **حروفٌ صغيرةٌ وأرقامٌ وشرطةٌ سفليّة** — كما رموزُ الأدوار
// القائمة كلِّها.
var roleCodeRe = regexp.MustCompile(`^[a-z][a-z0-9_]{2,31}$`)

// handleCreateRole **دورٌ جديدٌ بلا قدرة.**
func (s *Server) handleCreateRole(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	if !roleCodeRe.MatchString(code) || name == "" {
		s.respondErr(w, errValidation)
		return
	}

	err = s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		tag, err := q.Exec(ctx, `
			INSERT INTO roles (code, name_key) VALUES ($1, $2)
			ON CONFLICT (code) DO NOTHING`, code, name)
		if err != nil {
			return err
		}
		// **ولا يُبتلَع التصادمُ صامتاً** — **ومن ظنّ أنّه أنشأ دوراً
		// وهو يُسنِد دورَ غيرِه منح ما لم ينوِ.**
		if tag.RowsAffected() == 0 {
			return errRoleExists
		}
		return s.auditTx(ctx, q, r, "admin.role_create", "role", code,
			map[string]any{"name": name})
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{
		"code": code, "name_key": name, "capabilities": []string{}, "members": 0,
	})
}
