package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// **وأخطاءُ المدن تُعرَّف مرّةً** — والرسالةُ في المعجم لا في الشيفرة:
// **نصٌّ عربيٌّ في `NewError` لا يُترجَم ولا يُصحَّح من مكانٍ واحد.**
var (
	errCityNeedsName = httpx.NewError(http.StatusBadRequest,
		"city_needs_name", "errors.city_needs_name")
	errCityBadPoint = httpx.NewError(http.StatusBadRequest,
		"city_bad_point", "errors.city_bad_point")
	errCityBadRadius = httpx.NewError(http.StatusBadRequest,
		"city_bad_radius", "errors.city_bad_radius")
	errCityBadReach = httpx.NewError(http.StatusBadRequest,
		"city_bad_reach", "errors.city_bad_reach")
	// **ومدينةٌ فيها متاجرُ تُطفأ ولا تُحذف** — انظر `handleDeleteCity`.
	errCityHasMerchants = httpx.NewError(http.StatusBadRequest,
		"city_has_merchants", "errors.city_has_merchants")
	errCityBadBody = httpx.NewError(http.StatusBadRequest,
		"bad_json", "errors.validation")
)

// ══════════════════════════════════════════════════════════════════════
// **المدنُ — تُقرأ للتصفّح وتُدار من اللوحة**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا تُضاف مدينةٌ بهجرةٍ جديدة**: من أراد دمشقَ غداً يفتح اللوحةَ
// ويكتبها. **ومنصّةٌ تحتاج نشرَ خادمٍ لتفتح مدينةً لا تتوسّع.**

// city مدينةٌ كما تُقرأ وتُكتب.
type city struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	// RadiusM **نصفُ قطرِ المدينة** — من وقع داخله فهو من أهلها.
	RadiusM int `json:"radius_m"`
	// MaxDeliveryM **مدى التوصيل فيها** — فارغٌ يعني «خذ افتراضَ المنصّة».
	MaxDeliveryM *int `json:"max_delivery_m"`
	Active       bool `json:"active"`
	SortOrder    int  `json:"sort_order"`
	// Merchants **كم متجراً فيها** — تُقرأ في اللوحة وحدَها.
	Merchants *int `json:"merchants,omitempty"`
}

const citySelect = `
	SELECT c.id::text, c.name,
	       ST_Y(c.center::geometry), ST_X(c.center::geometry),
	       c.radius_m, c.max_delivery_m, c.active, c.sort_order`

// handlePublicCities **مدنُ المنصّة الفعّالة** — يقرؤها التطبيقُ ليعرض
// «أنت تتسوّق في…» ويسمح بالتبديل.
//
// **وعامّةٌ بلا حساب**: الضيفُ يختار مدينتَه قبل أن يسجّل.
func (s *Server) handlePublicCities(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), citySelect+`
		FROM cities c WHERE c.active
		ORDER BY c.sort_order, c.name`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []city{}
	for rows.Next() {
		var c city
		if err := rows.Scan(&c.ID, &c.Name, &c.Lat, &c.Lng,
			&c.RadiusM, &c.MaxDeliveryM, &c.Active, &c.SortOrder); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, c)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"cities": out})
}

// handleAdminCities **كلُّ المدن للوحة** — المطفأةُ منها أيضاً.
//
// **ومعها عددُ متاجرها**: من يطفئ مدينةً يجب أن يعرف كم متجراً يُخفي.
func (s *Server) handleAdminCities(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), citySelect+`,
	       (SELECT count(*) FROM merchants m
	         WHERE m.city_id = c.id AND m.status = 'active')::int
		FROM cities c ORDER BY c.sort_order, c.name`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []city{}
	for rows.Next() {
		var c city
		var n int
		if err := rows.Scan(&c.ID, &c.Name, &c.Lat, &c.Lng,
			&c.RadiusM, &c.MaxDeliveryM, &c.Active, &c.SortOrder, &n); err != nil {
			s.respondErr(w, err)
			return
		}
		c.Merchants = &n
		out = append(out, c)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"cities": out})
}

// cityInput ما تُرسله اللوحة.
type cityInput struct {
	Name         string  `json:"name"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	RadiusM      int     `json:"radius_m"`
	MaxDeliveryM *int    `json:"max_delivery_m"`
	Active       *bool   `json:"active"`
	SortOrder    int     `json:"sort_order"`
}

// validate **التحقّقُ هنا لا في الشاشة** — والقاعدةُ تحرس ما تحرسه،
// **لكنّ رسالةَ القاعدةِ لا تُقرأ بالعربيّة.**
func (in *cityInput) validate() error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return errCityNeedsName
	}
	if in.Lat < -90 || in.Lat > 90 || in.Lng < -180 || in.Lng > 180 {
		return errCityBadPoint
	}
	// **ونصفُ قطرٍ صفرٌ يخفي المدينةَ عن نفسها** — ولا أحدَ يقصده.
	if in.RadiusM <= 0 {
		return errCityBadRadius
	}
	if in.MaxDeliveryM != nil && *in.MaxDeliveryM <= 0 {
		// **والفراغُ هو «بلا حدّ» لا الصفر** — ليطابق العمودَ في القاعدة.
		return errCityBadReach
	}
	return nil
}

// handleCreateCity **مدينةٌ جديدة** — للأدمن وحدَه.
func (s *Server) handleCreateCity(w http.ResponseWriter, r *http.Request) {
	var in cityInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		s.respondErr(w, errCityBadBody)
		return
	}
	if err := in.validate(); err != nil {
		s.respondErr(w, err)
		return
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	var id string
	err := s.pg.QueryRow(r.Context(), `
		INSERT INTO cities (name, center, radius_m, max_delivery_m, active, sort_order)
		VALUES ($1, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, $4, $5, $6, $7)
		RETURNING id::text`,
		in.Name, in.Lat, in.Lng, in.RadiusM, in.MaxDeliveryM, active, in.SortOrder).Scan(&id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
}

// handleUpdateCity **تعديلُ مدينة.**
func (s *Server) handleUpdateCity(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in cityInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		s.respondErr(w, errCityBadBody)
		return
	}
	if err := in.validate(); err != nil {
		s.respondErr(w, err)
		return
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE cities
		   SET name = $2,
		       center = ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography,
		       radius_m = $5, max_delivery_m = $6, active = $7, sort_order = $8
		 WHERE id = $1::uuid`,
		id, in.Name, in.Lat, in.Lng, in.RadiusM, in.MaxDeliveryM, active, in.SortOrder)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleDeleteCity **حذفُ مدينةٍ لا متاجرَ فيها.**
//
// **ومدينةٌ فيها متاجرُ لا تُحذف بل تُطفأ**: الحذفُ يُفرغ `city_id`
// فتصير متاجرُها بلا مدينةٍ **فتُخفى عن الجميع بلا أن يعلم أحد.**
func (s *Server) handleDeleteCity(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var n int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*)::int FROM merchants WHERE city_id = $1::uuid`, id).Scan(&n); err != nil {
		s.respondErr(w, err)
		return
	}
	if n > 0 {
		s.respondErr(w, errCityHasMerchants)
		return
	}
	tag, err := s.pg.Exec(r.Context(), `DELETE FROM cities WHERE id = $1::uuid`, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
