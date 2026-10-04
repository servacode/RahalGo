package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := s.catalog.ListCategories(r.Context(), r.URL.Query().Get("active") == "1")
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, cats)
}

func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.CategoryInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	c, err := s.catalog.CreateCategory(r.Context(), userIDFrom(r), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, c)
}

func (s *Server) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.CategoryInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	c, err := s.catalog.UpdateCategory(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (s *Server) handleListMerchants(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	res, err := s.catalog.ListMerchants(r.Context(),
		q.Get("query"), q.Get("category_id"), q.Get("status"), q.Get("rep_id"), q.Get("owner_id"), page, perPage)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والنسبةُ العامّةُ معها** — «١٠٪ (عام)» لا «٠٪» حين لا نسبةَ خاصّة (قرارُ المالك ٢٠٢٦-١٠-٠٤).
	httpx.JSON(w, http.StatusOK, struct {
		*catalog.MerchantPage
		General int64 `json:"general_commission_percent"`
	}{res, s.settings.GetInt(r.Context(), "merchants.commission_percent")})
}

func (s *Server) handleCreateMerchant(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.MerchantInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والنصوصُ تمرّ بالحارس المركزيّ** — انظر `text_limits.go`.
	if err := s.guardMerchant(r, req); err != nil {
		s.respondErr(w, err)
		return
	}
	// ══════════════════════════════════════════════════════════════════
	// **ومنطقتُه تُفحص كما تُفحص في البابين الآخرين**
	// ══════════════════════════════════════════════════════════════════
	//
	// (سؤالُ المالك ٢٠٢٦-٠٨-٣٠: «عند إنشاء متجرٍ جديد يجب أن نختار
	//  المحافظة والمنطقة».)
	//
	// **وهو المسارُ الثالث** — نموذجُ الويب وتطبيقُ المندوب يرسلانها في
	// طلب الانضمام، **وهذا يُنشئ المتجرَ مباشرةً.** **وبابٌ من ثلاثةٍ
	// يُترك مفتوحاً يُبطل إغلاقَ الاثنين.**
	//
	// **ولا تُلزَم**: الأدمنُ يُنشئ متجراً لسببٍ عاجلٍ أحياناً — **وحقلٌ
	// إلزاميٌّ يوقفه عن عملٍ يعرف ما يفعل فيه.** والفراغُ يُرى في
	// اللوحة فيُصحَّح.
	if _, err := s.validDistrict(r, strDeref(req.DistrictID)); err != nil {
		s.respondErr(w, err)
		return
	}
	// ══════════════════════════════════════════════════════════════════
	// **ولا كلمةَ يكتبها الموظّفُ لصاحب المتجر** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
	// ══════════════════════════════════════════════════════════════════
	//
	// **صاحبٌ جديدٌ تُولَّد له كلمةٌ وتُرسَل** مع رابط تطبيق المتجر؛ **ومن له حسابٌ
	// قائمٌ** تصله «صار عندك متجر» بلا كلمةٍ جديدة.
	req.OwnerPassword = nil
	existed := false
	if req.OwnerPhone != nil {
		if ph, ok := identity.NormalizePhone(*req.OwnerPhone); ok {
			_ = s.pg.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM users WHERE phone = $1)`, ph).Scan(&existed)
		}
	}
	m, err := s.catalog.CreateMerchant(r.Context(), userIDFrom(r), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والإدارةُ تفتح متاجرَ برمز مندوبٍ أيضاً** — وهي تُحسب له.
	s.grantSalesTargetIfAny(r.Context(), m.ID)
	welcome := map[string]any{"sent": false, "existing_owner": existed}
	if m.OwnerUserID != nil {
		if existed {
			welcome["sent"] = s.notifyNewStoreOwner(r.Context(), userIDFrom(r), *m.OwnerUserID, m.Name, clientIP(r))
		} else {
			sent, exp, werr := s.issueWelcomeFor(r.Context(), userIDFrom(r), *m.OwnerUserID, clientIP(r), "merchant")
			welcome["sent"], welcome["expires_at"], welcome["ok"] = sent, exp, werr == nil
		}
	}
	httpx.JSON(w, http.StatusCreated, struct {
		*catalog.Merchant
		Welcome map[string]any `json:"welcome"`
	}{m, welcome})
}

func (s *Server) handleUpdateMerchant(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.MerchantInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والنصوصُ تمرّ بالحارس المركزيّ** — انظر `text_limits.go`.
	if err := s.guardMerchant(r, req); err != nil {
		s.respondErr(w, err)
		return
	}
	// **والتعديلُ يُفحص كذلك** — وهو بابُ إصلاحِ ما وُلد بلا منطقة.
	if _, err := s.validDistrict(r, strDeref(req.DistrictID)); err != nil {
		s.respondErr(w, err)
		return
	}
	m, err := s.catalog.UpdateMerchant(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, m)
}

// handleAdminGetMerchant متجرٌ بعينه — **لملفّه المفرد.**
//
// كان الوصولُ إليه بالبحث في القائمة باسمه، **ومتجران متشابها الاسم يُخلطان.**
// وكلُّ ما يخصّه — قائمتُه وساعاتُه ومخالفاتُه — كان في نوافذَ منبثقةٍ داخل
// جدول: **تُفتح واحدةً وتُغلق لتُفتح أخرى، ولا تُرى صورتُه مجتمعةً.**
func (s *Server) handleAdminGetMerchant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	mr, err := s.catalog.MerchantByID(r.Context(), id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, struct {
		*catalog.Merchant
		General int64 `json:"general_commission_percent"`
	}{mr, s.settings.GetInt(r.Context(), "merchants.commission_percent")})
}
