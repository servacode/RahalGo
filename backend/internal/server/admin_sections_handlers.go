package server

// أقسامُ المنصة — إدارتُها من اللوحة.
//
// **والأسماءُ تحمل الفرق.** `handleCreateSection` كانت موجودةً لأقسام قائمة
// المتجر، **ومعالِجان باسمٍ واحدٍ لمعنيين مختلفين يُنادى أحدُهما مكان الآخر**
// — فيُنشأ قسمُ منصةٍ حيث أُريد قسمُ قائمة، ولا يظهر الخطأ إلّا في الشاشة.
//
// **والقسمُ غيرُ التصنيف**: `categories` تصف **من نشتري منه** (مطاعم · بقالة)،
// و`platform_sections` تصف **ما نبيعه** (شاورما · بيتزا · خضار). **والزبونُ
// يرى الثاني وحدَه.**

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

type adminPlatformSection struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Icon           string `json:"icon"`
	SortOrder      int    `json:"sort_order"`
	Active         bool   `json:"active"`
	MarginOverride *int64 `json:"margin_override"`
	// Items عددُ الأصناف المربوطة — **لتُعرف قيمةُ القسم قبل حذفه.**
	Items int `json:"items"`
}

// handleListPlatformSections الأقسامُ كلُّها — الفعّالةُ والمُطفَأة.
func (s *Server) handleListPlatformSections(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT ps.id, ps.name, ps.icon, ps.sort_order, ps.active, ps.margin_override,
		       count(i.id)
		FROM platform_sections ps
		LEFT JOIN menu_items i ON i.platform_section_id = ps.id
		GROUP BY ps.id
		ORDER BY ps.sort_order, ps.name`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []adminPlatformSection{}
	for rows.Next() {
		var x adminPlatformSection
		if err := rows.Scan(&x.ID, &x.Name, &x.Icon, &x.SortOrder, &x.Active,
			&x.MarginOverride, &x.Items); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sections": out})
}

type platformSectionInput struct {
	Name      *string `json:"name"`
	Icon      *string `json:"icon"`
	SortOrder *int    `json:"sort_order"`
	Active    *bool   `json:"active"`
	// MarginOverride هامشُ القسم — **وسالبُ واحدٍ يمحوه**.
	//
	// `COALESCE` وحدَه لا يفرّق بين «لم يُرسَل» و«أُرسل فارغاً» — وكلاهما
	// `NULL`. **فمن أراد أن يعيد قسماً إلى الهامش العامّ لم يملك سبيلاً.**
	MarginOverride *int64 `json:"margin_override"`
}

func (s *Server) handleCreatePlatformSection(w http.ResponseWriter, r *http.Request) {
	req, err := decode[platformSectionInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Name == nil || strings.TrimSpace(*req.Name) == "" {
		s.respondErr(w, errValidation)
		return
	}
	icon := ""
	if req.Icon != nil {
		icon = *req.Icon
	}
	sort := 0
	if req.SortOrder != nil {
		sort = *req.SortOrder
	}
	var id string
	if err := s.pg.QueryRow(r.Context(), `
		INSERT INTO platform_sections (name, icon, sort_order, margin_override)
		VALUES ($1, $2, $3, $4) RETURNING id`,
		strings.TrimSpace(*req.Name), icon, sort, req.MarginOverride).Scan(&id); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "catalog.section_create", "section", id, map[string]any{"name": *req.Name})
	s.touch("catalog", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleUpdatePlatformSection(w http.ResponseWriter, r *http.Request) {
	req, err := decode[platformSectionInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	id := chi.URLParam(r, "id")
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE platform_sections SET
			name       = COALESCE($2, name),
			icon       = COALESCE($3, icon),
			sort_order = COALESCE($4, sort_order),
			active     = COALESCE($5, active),
			margin_override = CASE WHEN $6::bigint IS NULL THEN margin_override
			                       WHEN $6 < 0 THEN NULL
			                       ELSE $6 END
		WHERE id = $1`,
		id, req.Name, req.Icon, req.SortOrder, req.Active, req.MarginOverride)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	s.audit(r, "catalog.section_update", "section", id, nil)
	s.touch("catalog", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

// handleDeletePlatformSection يحذف قسماً — **وأصنافُه تبقى بلا قسم لا تُحذف معه.**
//
// **والحذفُ يُقطع لا يُدمّر**: `ON DELETE SET NULL` يترك الصنفَ في متجره
// قابلاً للطلب من صفحته، **ويُخرجه من التصفّح وحدَه.** ولو حُذف معه لَضاعت
// أسعارٌ وخياراتٌ بُنيت على مدى شهور **بضغطةٍ واحدةٍ لا تُردّ.**
func (s *Server) handleDeletePlatformSection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tag, err := s.pg.Exec(r.Context(), `DELETE FROM platform_sections WHERE id = $1`, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	s.audit(r, "catalog.section_delete", "section", id, nil)
	s.touch("catalog", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}
