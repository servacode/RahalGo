package server

import (
	"context"
	"net/http"

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
