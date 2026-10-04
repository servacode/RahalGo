package server

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/rolesguard"
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
// **وهو لـ`admin` و`owner_super_admin` وحدَهما** (قرارُ المالك ٢٠٢٦-١٠-٠٤) —
// **ولا يمنحه إلّا حاملُ دور المالك الأعلى.**
//
// # ولا تصعيدَ ذاتيّاً
//
// **من لا يملك `roles.manage` لا يبلغ هذه المسارات أصلاً** — والسياسةُ
// المركزيّةُ تحرسها. **ومن يملكه يملك كلَّ شيءٍ بالتعريف**، فلا معنى
// لمنعه من منح ما يملكه.
//
// **والحدُّ الحقيقيُّ أن تبقى `roles.manage` عزيزةً** — **ولا تُبذَر
// لدورٍ وظيفيّ.**

// ══════════════════════════════════════════════════════════════════════
// **حرّاسُ قرار المالك ٢٠٢٦-١٠-٠٤ — قسمُ الأدوار والصلاحيّات**
// ══════════════════════════════════════════════════════════════════════
//
// **وكانت القدرةُ تُمنَح لأيّ دورٍ بلا سؤال** — فقيدُ محفظةٍ لدور «الزبون»
// كان يمرّ، و`roles.manage` لدور «الدعم»، ودورُ المالك الأعلى يُحرَّر. والآن:
//
//	لا قدرةَ إداريّةَ لأدوار الحسابات (زبون · سائق · متجر · مندوب)
//	ولا أحدَ يحرّر قدرات `owner_super_admin`
//	و`roles.manage` لـ`admin`/`owner_super_admin` وحدَهما، ويمنحها المالكُ وحدَه
//	ولا يُنزَع آخرُ `roles.manage` في المنصّة
//	ولا سطرَ في السجلّ لما لم يتغيّر · والمنحُ بسببٍ مكتوبٍ يُحفظ في السجلّ

var (
	errRoleProtected = httpx.NewError(http.StatusForbidden,
		"role_protected", "errors.role_protected")
	errRoleAccountType = httpx.NewError(http.StatusForbidden,
		"role_account_type", "errors.role_account_type")
	errRoleLegacy = httpx.NewError(http.StatusForbidden,
		"role_legacy", "errors.role_legacy")
	errRolesManageScope = httpx.NewError(http.StatusForbidden,
		"roles_manage_scope", "errors.roles_manage_scope")
	errRolesManageOwnerOnly = httpx.NewError(http.StatusForbidden,
		"roles_manage_owner_only", "errors.roles_manage_owner_only")
	errRoleNotDeletable = httpx.NewError(http.StatusForbidden,
		"role_not_deletable", "errors.role_not_deletable")
	errRoleNotEmpty = httpx.NewError(http.StatusConflict,
		"role_not_empty", "errors.role_not_empty")
)

// capBlockErr **سببُ المنع خطأً يُقرأ في اللوحة.**
func capBlockErr(b authz.CapBlock) error {
	switch b {
	case authz.CapBlockProtected:
		return errRoleProtected
	case authz.CapBlockAccountType:
		return errRoleAccountType
	case authz.CapBlockLegacy:
		return errRoleLegacy
	case authz.CapBlockRolesManageScope:
		return errRolesManageScope
	}
	return nil
}

// roleClassOrder **ترتيبُ الأقسام في الشاشة**: إدارةٌ عليا · موظّفون · مخصّصة ·
// أنواعُ الحسابات · قديمة.
var roleClassOrder = map[authz.RoleClass]int{
	authz.ClassProtected:   0,
	authz.ClassElevated:    1,
	authz.ClassStaff:       2,
	authz.ClassCustom:      3,
	authz.ClassAccountType: 4,
	authz.ClassLegacy:      5,
}

type roleOut struct {
	Code         string   `json:"code"`
	NameKey      string   `json:"name_key"`
	Class        string   `json:"class"`
	Capabilities []string `json:"capabilities"`
	Members      int      `json:"members"`
	// Editable **أتُبدَّل قدراتُه من اللوحة؟** — لا للمحميّ ولا لأنواع الحسابات.
	Editable bool `json:"editable"`
	// Deletable **أيُحذَف؟** — مخصّصٌ أو قديمٌ بلا حامل.
	Deletable bool `json:"deletable"`
}

func newRoleOut(code, nameKey string, caps []string, members int) roleOut {
	cls := authz.ClassOf(code)
	return roleOut{
		Code: code, NameKey: nameKey, Class: string(cls), Capabilities: caps,
		Members:   members,
		Editable:  cls != authz.ClassProtected && cls != authz.ClassAccountType,
		Deletable: members == 0 && authz.RoleDeletable(code),
	}
}

// handleListRoles قائمةُ الأدوار وعددُ قدرات كلٍّ — مرتّبةً بأقسامها.
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

	out := []roleOut{}
	for rows.Next() {
		var code, nameKey string
		var caps []string
		var members int
		if err := rows.Scan(&code, &nameKey, &caps, &members); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, newRoleOut(code, nameKey, caps, members))
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	sort.SliceStable(out, func(i, j int) bool {
		ci := roleClassOrder[authz.RoleClass(out[i].Class)]
		cj := roleClassOrder[authz.RoleClass(out[j].Class)]
		if ci != cj {
			return ci < cj
		}
		return out[i].Code < out[j].Code
	})
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
	var members int
	if err := s.pg.QueryRow(r.Context(), `
		SELECT COALESCE((SELECT array_agg(capability_code ORDER BY capability_code)
		                   FROM role_capabilities WHERE role_code = $1), '{}'),
		       (SELECT count(*) FROM user_roles WHERE role_code = $1)`, code).
		Scan(&caps, &members); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, newRoleOut(code, nameKey, caps, members))
}

// handleRoleMembers **مَن يحمل الدور** — كان الرقمُ يُرى ولا يُنقَر (قرارُ المالك).
//
// **بلا هاتف**: الرقمُ قدرةٌ لحالها (`users.contact.read`)، والاسمُ يكفي ليُعرَف.
func (s *Server) handleRoleMembers(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	var exists bool
	if err := s.pg.QueryRow(r.Context(),
		`SELECT EXISTS (SELECT 1 FROM roles WHERE code = $1)`, code).Scan(&exists); err != nil {
		s.respondErr(w, err)
		return
	}
	if !exists {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT u.id::text, COALESCE(NULLIF(u.full_name, ''), ''), u.status, ur.granted_at
		  FROM user_roles ur JOIN users u ON u.id = ur.user_id
		 WHERE ur.role_code = $1
		 ORDER BY u.full_name, u.id
		 LIMIT 500`, code)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type memberOut struct {
		ID        string    `json:"id"`
		FullName  string    `json:"full_name"`
		Status    string    `json:"status"`
		GrantedAt time.Time `json:"granted_at"`
	}
	out := []memberOut{}
	for rows.Next() {
		var m memberOut
		if err := rows.Scan(&m.ID, &m.FullName, &m.Status, &m.GrantedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleRoleImpact **أثرُ المنح قبل وقوعه** — «رح يقدر ٣ موظفين…».
//
// **ويُعَدّ من سيكسب القدرةَ فعلاً**: حسابٌ فعّالٌ يحمل الدورَ ولا يملكها من
// دورٍ آخر. **ومن يملكها أصلاً لا يتغيّر عليه شيء.**
func (s *Server) handleRoleImpact(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	capability := r.URL.Query().Get("capability")
	if !authz.Known(authz.Capability(capability)) {
		s.respondErr(w, errValidation)
		return
	}
	var members, gain int
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*),
		       count(*) FILTER (WHERE NOT EXISTS (
		         SELECT 1 FROM user_roles o
		           JOIN role_capabilities rc ON rc.role_code = o.role_code
		          WHERE o.user_id = u.id AND rc.capability_code = $2))
		  FROM user_roles ur JOIN users u ON u.id = ur.user_id
		 WHERE ur.role_code = $1 AND u.status = 'active'`, code, capability).
		Scan(&members, &gain); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"role": code, "capability": capability,
		"active_members": members, "would_gain": gain,
		"risk":  string(authz.RiskOf(authz.Capability(capability))),
		"block": string(authz.CanGrantCapability(code, authz.Capability(capability))),
	})
}

// handleListCapabilities **المعجمُ يُقرأ ولا يُحرَّر** — بمجموعته وخطره.
func (s *Server) handleListCapabilities(w http.ResponseWriter, r *http.Request) {
	type capOut struct {
		Code        string `json:"code"`
		Description string `json:"description"`
		Group       string `json:"group"`
		Risk        string `json:"risk"`
	}
	out := make([]capOut, 0, authz.Count())
	for _, c := range authz.All() {
		out = append(out, capOut{Code: string(c), Description: authz.Describe(c),
			Group: string(authz.GroupOf(c)), Risk: string(authz.RiskOf(c))})
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleGrantCapability منحُ قدرةٍ لدور.
//
// **والمجهولةُ تُرفَض** — **فلا يُكتب في القاعدة نصٌّ لا يعني شيئاً.**
// **والسببُ إلزاميّ** ويُحفظ في السجلّ (قرارُ المالك ٢٠٢٦-١٠-٠٤).
func (s *Server) handleGrantCapability(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	req, err := decode[struct {
		Capability string `json:"capability"`
		Reason     string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if !authz.Known(authz.Capability(req.Capability)) {
		s.respondErr(w, errValidation)
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		s.respondErr(w, errReasonRequired)
		return
	}
	changed, err := s.roleCapabilityTx(r, code, req.Capability, true, clip(reason, 500))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"granted": true, "changed": changed})
}

// handleRevokeCapability نزعُ قدرةٍ من دور — والسببُ في `?reason=` اختياريّ.
func (s *Server) handleRevokeCapability(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	cap := chi.URLParam(r, "cap")
	reason := clip(strings.TrimSpace(r.URL.Query().Get("reason")), 500)
	changed, err := s.roleCapabilityTx(r, code, cap, false, reason)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"revoked": true, "changed": changed})
}

// roleCapabilityTx **تبديلُ سياسةِ أمنٍ وأثرُه في معاملةٍ واحدة.**
//
// **وهو تبديلُ تخويلٍ لا تبديلُ محتوى** — **فيُقيَّد كما تُقيَّد
// الأفعالُ الحسّاسة** (`AQ-4`). **ولا يُقبَل «أفضلَ جهد».**
//
// **والحرّاسُ كلُّها داخلَ المعاملة قبل الكتابة**، **ولا سطرَ في السجلّ إن لم
// يتغيّر شيء** — منحُ قدرةٍ موجودةٍ أو نزعُ غائبةٍ كان يُكتب «مُنح/نُزع».
func (s *Server) roleCapabilityTx(r *http.Request, role, capability string, grant bool,
	reason string) (bool, error) {
	action := "admin.role_capability_revoke"
	if grant {
		action = "admin.role_capability_grant"
	}
	capCode := authz.Capability(capability)
	changed := false
	err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		// **والدورُ يُقفَل** — منحٌ وحذفٌ متزامنان لا يتقاطعان.
		var locked string
		if err := q.QueryRow(ctx,
			`SELECT code FROM roles WHERE code = $1 FOR UPDATE`, role).
			Scan(&locked); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.ErrNotFound
			}
			return err
		}
		actor := userIDFrom(r)
		if grant {
			if err := capBlockErr(authz.CanGrantCapability(role, capCode)); err != nil {
				return err
			}
			// **و`roles.manage` يمنحها المالكُ وحدَه** — ولو كان الدورُ `admin`.
			if authz.GrantCapabilityNeedsOwner(capCode) {
				owner, err := actorHoldsRoleTx(ctx, q, actor, authz.RoleOwnerSuperAdmin)
				if err != nil {
					return err
				}
				if !owner {
					return errRolesManageOwnerOnly
				}
			}
		} else {
			if err := capBlockErr(authz.CanRevokeCapability(role)); err != nil {
				return err
			}
			if capCode == authz.RolesManage {
				if err := rolesguard.GuardRolesManageRemains(ctx, q,
					rolesguard.RolesManageLoss{Role: role}); err != nil {
					return err
				}
			}
		}

		var affected int64
		if grant {
			tag, err := q.Exec(ctx, `
				INSERT INTO role_capabilities (role_code, capability_code, granted_by)
				VALUES ($1, $2, $3::uuid) ON CONFLICT DO NOTHING`,
				role, capability, nilIfEmptyStr(actor))
			if err != nil {
				return err
			}
			affected = tag.RowsAffected()
		} else {
			tag, err := q.Exec(ctx, `
				DELETE FROM role_capabilities
				 WHERE role_code = $1 AND capability_code = $2`, role, capability)
			if err != nil {
				return err
			}
			affected = tag.RowsAffected()
		}
		if affected == 0 {
			return nil
		}
		changed = true
		var members int
		if err := q.QueryRow(ctx,
			`SELECT count(*) FROM user_roles WHERE role_code = $1`, role).Scan(&members); err != nil {
			return err
		}
		details := map[string]any{"capability": capability, "members": members}
		if reason != "" {
			details["reason"] = reason
		}
		if risk := authz.RiskOf(capCode); risk != authz.RiskNone {
			details["risk"] = string(risk)
		}
		return s.auditTx(ctx, q, r, action, "role", role, details)
	})
	return changed, err
}

// actorHoldsRoleTx **أيحمل الفاعلُ هذا الدور؟** — داخلَ المعاملة.
func actorHoldsRoleTx(ctx context.Context, q dbtx.Querier, actor, role string) (bool, error) {
	if actor == "" {
		return false, nil
	}
	var has bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM user_roles
		                WHERE user_id = $1::uuid AND role_code = $2)`, actor, role).Scan(&has)
	return has, err
}

// handleDeleteRole **حذفُ دورٍ خالٍ** (قرارُ المالك ٢٠٢٦-١٠-٠٤).
//
// **المخصَّصُ والقديمُ وحدَهما، وبلا حاملٍ واحد** — والكانونيُّ تقرؤه الشيفرة.
// **وقدراتُه تُحذف معه** (`ON DELETE CASCADE`) ويُكتب ما كان فيه في السجلّ.
func (s *Server) handleDeleteRole(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if !authz.RoleDeletable(code) {
		s.respondErr(w, errRoleNotDeletable)
		return
	}
	err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		var nameKey string
		if err := q.QueryRow(ctx,
			`SELECT name_key FROM roles WHERE code = $1 FOR UPDATE`, code).Scan(&nameKey); err != nil {
			return httpx.ErrNotFound
		}
		var members int
		var caps []string
		if err := q.QueryRow(ctx, `
			SELECT (SELECT count(*) FROM user_roles WHERE role_code = $1),
			       COALESCE((SELECT array_agg(capability_code ORDER BY capability_code)
			                   FROM role_capabilities WHERE role_code = $1), '{}')`, code).
			Scan(&members, &caps); err != nil {
			return err
		}
		if members > 0 {
			return errRoleNotEmpty
		}
		if _, err := q.Exec(ctx, `DELETE FROM roles WHERE code = $1`, code); err != nil {
			return err
		}
		return s.auditTx(ctx, q, r, "admin.role_delete", "role", code,
			map[string]any{"name": nameKey, "capabilities": caps})
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
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
