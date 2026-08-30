package server

// ══════════════════════════════════════════════════════════════════════
// **المحافظاتُ ومناطقُها — تُقرأ في النموذج وتُدار من اللوحة**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-٣٠: «يجب أن يكون بلوحة الأدمن خيارٌ لإضافة
//  المحافظات والمناطق وتعديلها وإيقافها وتفعيلها».)
//
// # ولا تُضاف محافظةٌ بهجرةٍ جديدة
//
// **وهي حجّةُ المدن نفسُها**: من أراد منطقةً غداً يفتح اللوحةَ ويكتبها.
// **ومنصّةٌ تحتاج نشرَ خادمٍ لتفتح منطقةً لا تتوسّع.**
//
// # والقراءةُ عامّةٌ بلا حساب
//
// **نموذجُ تسجيل المتجر في الويب يُملأ قبل أن يسجّل أحد** — فلو طُلب
// توكنٌ لقراءة المحافظات **لَوقف من جاء يفتح متجرَه عند أوّل حقل.**
//
// # والاسمُ `divisions` لا `geo`
//
// **و`geo_handlers.go` مأخوذ** — لترميز العناوين عكسيّاً والبحثِ فيها،
// **وهو شيءٌ آخر**: ذاك يسأل مزوّداً خارجيّاً عن نقطةٍ على الأرض،
// **وهذا يقرأ تقسيمَ الدولة من قاعدتنا.**

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	errDivNeedsName = httpx.NewError(http.StatusBadRequest,
		"division_needs_name", "errors.division_needs_name")
	errDivNeedsGovernorate = httpx.NewError(http.StatusBadRequest,
		"division_needs_governorate", "errors.division_needs_governorate")
	// **وما تحته شيءٌ يُطفأ ولا يُحذف** — انظر `handleDeleteGovernorate`.
	errDivInUse = httpx.NewError(http.StatusBadRequest,
		"division_in_use", "errors.division_in_use")
	errDivBadBody = httpx.NewError(http.StatusBadRequest,
		"bad_json", "errors.validation")
)

// governorate محافظةٌ كما تُقرأ وتُكتب.
type governorate struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Active    bool   `json:"active"`
	SortOrder int    `json:"sort_order"`
	// Districts **كم منطقةً تحتها** — تُقرأ في اللوحة وحدَها.
	//
	// **ومن يطفئ محافظةً يجب أن يعرف كم منطقةً يُخفي معها.**
	Districts *int `json:"districts,omitempty"`
}

// district منطقةٌ إداريّةٌ تحت محافظتها.
//
// **وهي غيرُ منطقة التوصيل** (`delivery_zones`): تلك مضلَّعٌ نرسمه
// ونسعّره، **وهذه تقسيمُ الدولة لا نخترعه.** **وخلطُهما يُضيع
// الاثنين.**
type district struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	GovID string `json:"governorate_id"`
	// GovName **اسمُ محافظتها** — تُعرض في اللوحة بلا نداءٍ ثانٍ.
	GovName   string `json:"governorate_name,omitempty"`
	Active    bool   `json:"active"`
	SortOrder int    `json:"sort_order"`
	// Cities **كم مدينةً نُسبت إليها** — تُقرأ قبل الحذف.
	Cities *int `json:"cities,omitempty"`
}

// ══════════════════════════════════════════════════════════════════════
// **القراءة**
// ══════════════════════════════════════════════════════════════════════

// handlePublicGovernorates **المحافظاتُ الفعّالةُ للنموذج.**
func (s *Server) handlePublicGovernorates(w http.ResponseWriter, r *http.Request) {
	s.listGovernorates(w, r, true)
}

// handleAdminGovernorates **كلُّها للوحة — والمطفأةُ معها.**
//
// **ومن أطفأ محافظةً أخفاها عن المسجّلين الجدد ولم يمحُ من نُسب إليها.**
func (s *Server) handleAdminGovernorates(w http.ResponseWriter, r *http.Request) {
	s.listGovernorates(w, r, false)
}

func (s *Server) listGovernorates(w http.ResponseWriter, r *http.Request, liveOnly bool) {
	where := ""
	if liveOnly {
		where = ` WHERE g.active`
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT g.id::text, g.name, g.active, g.sort_order,
		       (SELECT count(*) FROM districts d WHERE d.governorate_id = g.id)::int
		  FROM governorates g`+where+`
		 ORDER BY g.sort_order, g.name`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []governorate{}
	for rows.Next() {
		var g governorate
		var n int
		if err := rows.Scan(&g.ID, &g.Name, &g.Active, &g.SortOrder, &n); err != nil {
			s.respondErr(w, err)
			return
		}
		if !liveOnly {
			g.Districts = &n
		}
		out = append(out, g)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"governorates": out})
}

// handlePublicDistricts **مناطقُ محافظةٍ بعينها** — القائمةُ الثانيةُ في
// النموذج، تُجلب حين تُختار الأولى.
//
// **ولا تُجلب المناطقُ كلُّها دفعةً**: ثلاثٌ وستّون اليومَ وقد تصير
// مئتين، **وقائمةٌ منسدلةٌ بمئتي سطرٍ لا يُبحث فيها بالإصبع.**
//
// **ومحافظةٌ مطفأةٌ لا تُظهر مناطقَها** — وإلّا سُجّل متجرٌ في محافظةٍ
// أُغلقت عمداً.
func (s *Server) handlePublicDistricts(w http.ResponseWriter, r *http.Request) {
	gov := r.URL.Query().Get("governorate_id")
	if !isUUID(gov) {
		s.respondErr(w, errDivNeedsGovernorate)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT d.id::text, d.name, d.governorate_id::text, d.active, d.sort_order
		  FROM districts d
		  JOIN governorates g ON g.id = d.governorate_id
		 WHERE d.governorate_id = $1::uuid AND d.active AND g.active
		 ORDER BY d.sort_order, d.name`, gov)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []district{}
	for rows.Next() {
		var d district
		if err := rows.Scan(&d.ID, &d.Name, &d.GovID, &d.Active, &d.SortOrder); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, d)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"districts": out})
}

// handleAdminDistricts **مناطقُ اللوحة** — المطفأةُ معها، وبعددِ مدنها.
//
// **وتُصفّى بمحافظةٍ إن طُلبت** — وإلّا رُدّت كلُّها مرتّبةً بمحافظاتها.
func (s *Server) handleAdminDistricts(w http.ResponseWriter, r *http.Request) {
	gov := r.URL.Query().Get("governorate_id")
	args := []any{}
	where := ""
	if gov != "" {
		if !isUUID(gov) {
			s.respondErr(w, errDivNeedsGovernorate)
			return
		}
		where = ` WHERE d.governorate_id = $1::uuid`
		args = append(args, gov)
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT d.id::text, d.name, d.governorate_id::text, g.name,
		       d.active, d.sort_order,
		       (SELECT count(*) FROM cities c WHERE c.district_id = d.id)::int
		  FROM districts d
		  JOIN governorates g ON g.id = d.governorate_id`+where+`
		 ORDER BY g.sort_order, g.name, d.sort_order, d.name`, args...)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []district{}
	for rows.Next() {
		var d district
		var n int
		if err := rows.Scan(&d.ID, &d.Name, &d.GovID, &d.GovName,
			&d.Active, &d.SortOrder, &n); err != nil {
			s.respondErr(w, err)
			return
		}
		d.Cities = &n
		out = append(out, d)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"districts": out})
}

// ══════════════════════════════════════════════════════════════════════
// **والمنطقةُ تُفحص قبل أن تُقيَّد**
// ══════════════════════════════════════════════════════════════════════
//
// **والمفتاحُ الأجنبيُّ يحرس الوجود، وهذا يحرس الحياة**: منطقةٌ أُطفئت
// أو محافظتُها أُطفئت **موجودةٌ في الجدول وغيرُ مقبولة** — ومن أرسل
// معرّفَها من نموذجٍ قديمٍ في جهازه يُقيَّد في محافظةٍ أُغلقت عمداً.
//
// **ويردّ فارغاً حين يُرسَل فارغاً** — الإلزامُ قرارُ من يناديها، لا
// قرارُ هذه الدالّة: **بابُ المندوب يلزمه وبابُ الويب في طريقه إليه.**
func (s *Server) validDistrict(r *http.Request, id string) (*string, error) {
	if strings.TrimSpace(id) == "" {
		return nil, nil
	}
	if !isUUID(id) {
		return nil, errDivNeedsGovernorate
	}
	var ok bool
	if err := s.pg.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM districts d
		              JOIN governorates g ON g.id = d.governorate_id
		              WHERE d.id = $1::uuid AND d.active AND g.active)`, id).Scan(&ok); err != nil {
		return nil, err
	}
	if !ok {
		return nil, errDivNeedsGovernorate
	}
	return &id, nil
}

// strDeref **مؤشّرُ نصٍّ إلى نصّ** — والفارغُ فراغ.
//
// **ونماذجُ الإدارة تستعمل المؤشّراتِ لتفرّق «لم يُرسَل» عن «أُرسل
// فارغاً»** — و`validDistrict` تأخذ نصّاً: **الفراغُ عندها يعني «لا
// منطقةَ» في الحالين، وهو الصواب.**
func strDeref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// districtLabelByLead **«منطقة، محافظة» لطلبِ انضمام.**
//
// **ويُقرأ من المعرّف لا يُنسَخ في صفّ**: اسمٌ منسوخٌ يشيخ حين يُعدَّل في
// مصدره، **فيبقى في العناوين اسمٌ بدّله المالكُ من سنة.**
//
// **وفراغٌ عند أيّ تعثّر** — عنوانٌ ناقصٌ خيرٌ من موافقةٍ تسقط.
func (s *Server) districtLabelByLead(ctx context.Context, leadID string) string {
	var label string
	if err := s.pg.QueryRow(ctx, `
		SELECT d.name || '، ' || g.name
		  FROM merchant_leads l
		  JOIN districts d ON d.id = l.district_id
		  JOIN governorates g ON g.id = d.governorate_id
		 WHERE l.id = $1`, leadID).Scan(&label); err != nil {
		return ""
	}
	return label
}

// ══════════════════════════════════════════════════════════════════════
// **الكتابة**
// ══════════════════════════════════════════════════════════════════════

type divisionInput struct {
	Name      string `json:"name"`
	GovID     string `json:"governorate_id"`
	Active    *bool  `json:"active"`
	SortOrder int    `json:"sort_order"`
}

// readDivision **يقرأ الجسمَ ويقلّم الاسم.**
//
// **والتحقّقُ هنا لا في الشاشة** — القاعدةُ تحرس ما تحرسه، **لكنّ
// رسالتَها لا تُقرأ بالعربيّة.**
func (s *Server) readDivision(w http.ResponseWriter, r *http.Request) (*divisionInput, bool) {
	var in divisionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		s.respondErr(w, errDivBadBody)
		return nil, false
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		s.respondErr(w, errDivNeedsName)
		return nil, false
	}
	return &in, true
}

// live **والافتراضُ فعّال** — من أضاف محافظةً أرادها تعمل.
func (in *divisionInput) live() bool { return in.Active == nil || *in.Active }

func (s *Server) handleCreateGovernorate(w http.ResponseWriter, r *http.Request) {
	in, ok := s.readDivision(w, r)
	if !ok {
		return
	}
	var id string
	if err := s.pg.QueryRow(r.Context(), `
		INSERT INTO governorates (name, active, sort_order)
		VALUES ($1, $2, $3) RETURNING id::text`,
		in.Name, in.live(), in.SortOrder).Scan(&id); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
}

func (s *Server) handleUpdateGovernorate(w http.ResponseWriter, r *http.Request) {
	in, ok := s.readDivision(w, r)
	if !ok {
		return
	}
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE governorates SET name = $2, active = $3, sort_order = $4
		 WHERE id = $1::uuid`,
		chi.URLParam(r, "id"), in.Name, in.live(), in.SortOrder)
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

// handleDeleteGovernorate **محافظةٌ فارغةٌ تُحذف — وما تحته مناطقُ يُطفأ.**
//
// **والقاعدةُ تمنعه بـ`RESTRICT`** — **لكنّ خطأَ بوستغرس يصل الشاشةَ
// إنجليزيّاً**، فيُفحص هنا ليُقال بالعربيّة **ولماذا**.
func (s *Server) handleDeleteGovernorate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var n int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*)::int FROM districts WHERE governorate_id = $1::uuid`, id).Scan(&n); err != nil {
		s.respondErr(w, err)
		return
	}
	if n > 0 {
		s.respondErr(w, errDivInUse)
		return
	}
	tag, err := s.pg.Exec(r.Context(), `DELETE FROM governorates WHERE id = $1::uuid`, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (s *Server) handleCreateDistrict(w http.ResponseWriter, r *http.Request) {
	in, ok := s.readDivision(w, r)
	if !ok {
		return
	}
	if !isUUID(in.GovID) {
		s.respondErr(w, errDivNeedsGovernorate)
		return
	}
	var id string
	if err := s.pg.QueryRow(r.Context(), `
		INSERT INTO districts (governorate_id, name, active, sort_order)
		VALUES ($1::uuid, $2, $3, $4) RETURNING id::text`,
		in.GovID, in.Name, in.live(), in.SortOrder).Scan(&id); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
}

// handleUpdateDistrict **تعديلُ منطقة — ونقلُها بين محافظتين إن لزم.**
//
// **والمحافظةُ لا تُبدَّل إلّا حين تُرسَل** — **ومنطقةٌ لا تُنقل تُرسل
// حقلاً فارغاً**، فلا يضيع انتماؤها بحذفِ حقلٍ من نموذج.
func (s *Server) handleUpdateDistrict(w http.ResponseWriter, r *http.Request) {
	in, ok := s.readDivision(w, r)
	if !ok {
		return
	}
	var gov *string
	if in.GovID != "" {
		if !isUUID(in.GovID) {
			s.respondErr(w, errDivNeedsGovernorate)
			return
		}
		gov = &in.GovID
	}
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE districts
		   SET name = $2, active = $3, sort_order = $4,
		       governorate_id = COALESCE($5::uuid, governorate_id)
		 WHERE id = $1::uuid`,
		chi.URLParam(r, "id"), in.Name, in.live(), in.SortOrder, gov)
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

// handleDeleteDistrict **منطقةٌ لا مدينةَ فيها تُحذف.**
//
// **وإلّا أُفرغت `city_id` من مدنها** فصارت بلا انتماءٍ **بلا أن يعلم
// أحد** — وهي علّةُ حذف المدينة نفسُها.
func (s *Server) handleDeleteDistrict(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var n int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*)::int FROM cities WHERE district_id = $1::uuid`, id).Scan(&n); err != nil {
		s.respondErr(w, err)
		return
	}
	if n > 0 {
		s.respondErr(w, errDivInUse)
		return
	}
	tag, err := s.pg.Exec(r.Context(), `DELETE FROM districts WHERE id = $1::uuid`, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}
