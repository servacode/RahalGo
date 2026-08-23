package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// تحكّم المتجر بقائمته.
//
// كان يملك **الإتاحة وحدها** (يوقف صنفاً ويشغّله)، والإضافة والتعديل والحذف
// للإدارة حصراً — فصاحب مطعم يريد إضافة طبق جديد يتّصل بالمنصة وينتظر. والقائمة
// **بضاعته هو**: يعرف أصنافه وأسعارها ومتى تنفد، ولا أحد أقدر منه على تحديثها.
//
// والمنطق نفسه لا يُكرَّر: هذه المعالِجات تتحقق من الملكية ثم تنادي نفس دوال
// `catalog` التي تناديها الإدارة. الفرق **في الحارس لا في العملية**.

// ownsSection يتحقق أن القسم يتبع متجراً يملكه الفاعل.
func (s *Server) ownsSection(r *http.Request, sectionID string) bool {
	var owns bool
	err := s.pg.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM menu_sections ms
		              JOIN merchants m ON m.id = ms.merchant_id
		              WHERE ms.id = $1 AND m.owner_user_id = $2)`,
		sectionID, userIDFrom(r)).Scan(&owns)
	return err == nil && owns
}

// ownsItem يتحقق أن الصنف يتبع متجراً يملكه الفاعل.
func (s *Server) ownsItem(r *http.Request, itemID string) bool {
	var owns bool
	err := s.pg.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM menu_items mi
		              JOIN merchants m ON m.id = mi.merchant_id
		              WHERE mi.id = $1 AND m.owner_user_id = $2)`,
		itemID, userIDFrom(r)).Scan(&owns)
	return err == nil && owns
}

func (s *Server) handleMerchantCreateSection(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	req, err := decode[catalog.SectionInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	sec, err := s.catalog.CreateSection(r.Context(), userIDFrom(r), merchantID, *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, sec)
}

// handleMerchantUpdateSection تعديلُ اسم القسم أو صورته — **بعد فحص الملكيّة.**
func (s *Server) handleMerchantUpdateSection(w http.ResponseWriter, r *http.Request) {
	sectionID := chi.URLParam(r, "sectionID")
	if !s.ownsSection(r, sectionID) {
		s.respondErr(w, errForbidden)
		return
	}
	req, err := decode[catalog.SectionInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	sec, err := s.catalog.UpdateSection(r.Context(), userIDFrom(r), sectionID, *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, sec)
}

func (s *Server) handleMerchantDeleteSection(w http.ResponseWriter, r *http.Request) {
	sectionID := chi.URLParam(r, "sectionID")
	if !s.ownsSection(r, sectionID) {
		s.respondErr(w, errForbidden)
		return
	}
	if err := s.catalog.DeleteSection(r.Context(), userIDFrom(r), sectionID, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (s *Server) handleMerchantCreateItem(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	req, err := decode[catalog.MenuItemInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// القسم يجب أن يكون من متجره هو — وإلا دسّ صنفاً في قائمة غيره
	if req.SectionID != nil && *req.SectionID != "" && !s.ownsSection(r, *req.SectionID) {
		s.respondErr(w, errForbidden)
		return
	}
	id, err := s.catalog.CreateItem(r.Context(), userIDFrom(r), merchantID, *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **وصنفٌ جديدٌ يُراجَع كلَّه** — لا شيءَ منه رآه أحدٌ بعد.
	pending := s.menuNeedsApproval(r)
	if pending {
		s.holdForReview(r, id)
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id, "pending_review": pending})
}

func (s *Server) handleMerchantUpdateItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")
	if !s.ownsItem(r, itemID) {
		s.respondErr(w, errForbidden)
		return
	}
	req, err := decode[catalog.MenuItemInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.SectionID != nil && *req.SectionID != "" && !s.ownsSection(r, *req.SectionID) {
		s.respondErr(w, errForbidden)
		return
	}
	if err := s.catalog.UpdateItem(r.Context(), userIDFrom(r), itemID, *req, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	// **والإتاحةُ وحدَها لا تُعلّق الصنف** — «نفد» قرارُ مطبخٍ في لحظته،
	// **ومراجعتُه تجعل المتجرَ يبيع ما نفد حتى نستيقظ.**
	pending := s.menuNeedsApproval(r) && touchesContent(*req)
	if pending {
		s.holdForReview(r, itemID)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true, "pending_review": pending})
}

func (s *Server) handleMerchantDeleteItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")
	if !s.ownsItem(r, itemID) {
		s.respondErr(w, errForbidden)
		return
	}
	if err := s.catalog.DeleteItem(r.Context(), userIDFrom(r), itemID, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// handleMerchantPlatformSections أقسامُ السوق كما يراها المتجر — **ليختار**.
//
// # لماذا نقطةٌ للمتجر
//
// **السوقُ يجمع بضاعةَ المتاجر، والمتجرُ يعرف ما يبيع.** وشاشةُ الصنف كانت
// تقرأ أقسامَ السوق من نقطة الأدمن، **فيُردُّ صاحبُ المتجر ٤٠٣ فيُخفى الحقلُ
// كلُّه** — فكلُّ ما يضيفه يبقى غيرَ مصنَّفٍ حتى يدخل الأدمنُ ويصنّفه بيده،
// **ولا شيءَ يقول له إنّ هناك أصنافاً تنتظر.**
//
// **والاختيارُ لا يُنشر بذاته**: مراجعةُ القائمة تبقى الحارس — الصنفُ الجديد
// يُعلَّق حتى يُقَرّ، **فالمتجرُ يقترح موضعَه ولا يفرضه.**
//
// # ولا يُعرض إلّا الفعّال
//
// قسمٌ مُطفأٌ لا يظهر في السوق، **وعرضُه للاختيار يَعِد بما لا يقع**: يضع
// المتجرُ صنفَه فيه ثمّ لا يجده معروضاً ولا يعرف لماذا.
func (s *Server) handleMerchantPlatformSections(w http.ResponseWriter, r *http.Request) {
	// ══════════════════════════════════════════════════════════════════
	// **ووجهُ القسم معه — لا اسمُه وحدَه**
	// ══════════════════════════════════════════════════════════════════
	//
	// (بُني تطبيقُ المتجر ٢٠٢٦-٠٨-٢٣، وشاشةُ أصنافه شبكةُ أقسامٍ بالصور
	//  كشبكةِ الويب.)
	//
	// **وكان الردُّ اسماً ومعرّفاً** — كفى نافذةَ اختيارٍ في محرّر الصنف،
	// **ولا يكفي شبكةً تُتصفَّح**: أخذت الشاشةُ صورةَ أوّلِ صنفٍ في القسم
	// **فظهرت حروفاً**، لأنّ أصنافَ المتجر بلا صورٍ بعد.
	//
	// **وأقسامُ السوق مصوَّرةٌ كلُّها** — فالصورةُ موجودةٌ ولا تُرسَل.
	rows, err := s.pg.Query(r.Context(), `
		SELECT ps.id::text, ps.name, im.path, im.thumb_path
		FROM platform_sections ps
		LEFT JOIN media im ON im.id = ps.image_media_id
		WHERE ps.active ORDER BY ps.sort_order, ps.name`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type section struct {
		ID            string  `json:"id"`
		Name          string  `json:"name"`
		ImageURL      *string `json:"image_url"`
		ImageThumbURL *string `json:"image_thumb_url"`
	}
	out := []section{}
	for rows.Next() {
		var x section
		if err := rows.Scan(&x.ID, &x.Name, &x.ImageURL, &x.ImageThumbURL); err != nil {
			s.respondErr(w, err)
			return
		}
		// **والمسارُ يُحوَّل عنواناً هنا** — **ومسارٌ خامٌّ يطلبه الجهازُ
		// من نفسه فتنكسر الصورة** (وقع مثلُه في شبكة الويب ٢٠٢٦-٠٨-٠٨).
		x.ImageURL = media.URLForPtr(x.ImageURL)
		x.ImageThumbURL = media.URLForPtr(x.ImageThumbURL)
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sections": out})
}
