package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

// ---------- أكواد الخصم ----------

func (s *Server) handleListPromos(w http.ResponseWriter, r *http.Request) {
	promos, err := s.catalog.ListPromos(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, promos)
}

func (s *Server) handleCreatePromo(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.PromoInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	p, err := s.catalog.CreatePromo(r.Context(), userIDFrom(r), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

func (s *Server) handleUpdatePromo(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.PromoInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	p, err := s.catalog.UpdatePromo(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

// ---------- البانرات ----------

func (s *Server) handleListBanners(w http.ResponseWriter, r *http.Request) {
	banners, err := s.catalog.ListBanners(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, banners)
}

func (s *Server) handleCreateBanner(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.BannerInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	b, err := s.catalog.CreateBanner(r.Context(), userIDFrom(r), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, b)
}

func (s *Server) handleUpdateBanner(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.BannerInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	b, err := s.catalog.UpdateBanner(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, b)
}

func (s *Server) handleDeleteBanner(w http.ResponseWriter, r *http.Request) {
	if err := s.catalog.DeleteBanner(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// ---------- الإعدادات الديناميكية ----------

// handleListSettings يعيد كل إعداد **مع تعريفه**: نوعه ومداه وخياراته ووحدته.
//
// كانت تعيد المفتاح والقيمة الخام فحسب، فتُضطر اللوحة إلى عرض `drivers.share_value`
// اسماً لاتينياً وقيمتَه في مربّع نصٍّ حرّ — ويُطلب من صاحب المنصة أن يكتب JSON
// صحيحاً في حقلٍ يحكم رواتب سائقيه.
//
// والتعريف يأتي من الخادم لا من الواجهة: **المدى الذي يحرسه الخادم هو المدى
// الذي يجب أن يعرضه الحقل**. ولو كُتب في الواجهة لانحرف عنه يوماً، فيرى المالك
// حقلاً يقبل ما يرفضه الحفظ.
func (s *Server) handleListSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT s.key, s.value, s.updated_at, u.full_name
		FROM app_settings s
		LEFT JOIN users u ON u.id = s.updated_by`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type stored struct {
		value     json.RawMessage
		updatedAt time.Time
		updatedBy *string
	}
	current := map[string]stored{}
	for rows.Next() {
		var k string
		var st stored
		if err := rows.Scan(&k, &st.value, &st.updatedAt, &st.updatedBy); err != nil {
			s.respondErr(w, err)
			return
		}
		current[k] = st
	}

	type setting struct {
		settings.Def
		Value     json.RawMessage `json:"value"`
		UpdatedAt *time.Time      `json:"updated_at"`
		UpdatedBy *string         `json:"updated_by"`
	}
	// الترتيب ترتيبُ الكتالوج لا ترتيب القاعدة: المفاتيح مجموعةٌ بالموضوع،
	// وترتيبُها الأبجدي يبعثر «مهلة القبول» عن «مهلة التوصيل».
	out := make([]setting, 0, len(settings.Catalog))
	for _, d := range settings.Catalog {
		item := setting{Def: d}
		if st, ok := current[d.Key]; ok {
			item.Value = st.value
			item.UpdatedAt = &st.updatedAt
			item.UpdatedBy = st.updatedBy
		} else {
			// مفتاحٌ في الكتالوج ولم يُبذر بعد: يُعرض بافتراضيه لا يُخفى —
			// النظام يعمل به فعلاً، فإخفاؤه يُخفي سلوكاً قائماً.
			raw, _ := json.Marshal(d.Default)
			item.Value = raw
		}
		out = append(out, item)
	}
	// **وترتيبُ الأقسام يُرسَل معها.**
	//
	// كانت اللوحةُ تشتقّه من المفاتيح: أوّلُ ظهورٍ للمجموعة هو موضعُها.
	// **فقسمٌ بلا مفاتيحَ لا يظهر** — ولا يُبنى قسمٌ يُملأ على مراحل.
	//
	// **ولا يُكتب في الواجهة**: ترتيبان يصفان الشيءَ نفسَه يفترقان — وقد
	// طاردنا هذه العائلةَ اليومَ في التوصيل والمخالفات وأقسام السوق.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"settings": out,
		"groups":   settings.Groups,
	})
}

func (s *Server) handleSetSetting(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Value json.RawMessage `json:"value"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	var v any
	if err := json.Unmarshal(req.Value, &v); err != nil {
		s.respondErr(w, errValidation)
		return
	}

	key := chi.URLParam(r, "key")
	actor := userIDFrom(r)
	// القيمة القديمة تُقرأ **قبل** الكتابة: سجلٌّ يقول «صار ٧٠» بلا «كان ٥٠»
	// يُثبت الفعل ولا يُظهر أثره — ومن يراجع الدفتر يريد الفرق لا النتيجة.
	before, _ := s.settings.GetRaw(r.Context(), key)

	// التحقق كلُّه في الكتالوج: المفتاح المجهول مرفوض، والقيمة خارج المدى
	// مرفوضة، والنوع الخاطئ مرفوض. وكان هنا شرطان لمفتاحين من اثنين وعشرين.
	if err := s.settings.Set(r.Context(), key, v, &actor); err != nil {
		var unknown settings.ErrUnknownKey
		var invalid settings.ErrInvalidValue
		if errors.As(err, &unknown) || errors.As(err, &invalid) {
			s.respondErr(w, errValidation)
			return
		}
		s.respondErr(w, err)
		return
	}

	s.audit(r, "admin.setting_update", "setting", key, map[string]any{
		"before": json.RawMessage(before),
		"after":  json.RawMessage(req.Value),
	})
	// الإعدادات تُقرأ لحظياً في كل مكان — واللوحات المفتوحة الآن تعرض القديم
	s.touch("settings", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}
