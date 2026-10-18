package server

// ══════════════════════════════════════════════════════════════════════
// **متاجرُ مميّزةٌ وصفحةُ متجر** — قرارُ المالك ٢٠٢٦-١٠-١٨
// ══════════════════════════════════════════════════════════════════════
//
// **كانت المتاجرُ لا بابَ لها عند الزبون** (٢٠٢٦-٠٨-٠٥: «يشتري من رحّال لا
// من مطعم فلان»)، **ثمّ ظهر اسمُ المتجر على الصنف** (٢٠٢٦-١٠-١٥). **واليومَ
// صار لها بابان**: شريطُ «متاجر مميّزة» في رأس السوق — يختارها الأدمنُ
// ويرتّبها ولا حدَّ لعددها — **وصفحةُ متجرٍ** بأصنافه مجموعةً تحت أقسام
// المنصّة.
//
// **والتصفّحُ يبقى بالأقسام والأصناف** — الصفحةُ تعرض الأصنافَ نفسَها بالشكل
// نفسِه (`itemSelect`)، **وشرطُ الظهور شرطُ القسم عينُه**: ما لا يظهر في
// قسمِه لا يظهر في متجره.
//
// **وسعرُ الشراء وهامشُه محجوبان كما في كلّ تصفّح.**

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// featuredStore **مربّعٌ في شريط «متاجر مميّزة».**
type featuredStore struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// LogoURL الشعارُ — **وفراغُه يُرسم حرفاً أوّلَ على مربّعٍ ملوّن.**
	LogoURL      *string `json:"logo_url"`
	LogoThumbURL *string `json:"logo_thumb_url"`
	// OpenNow **أيستقبل الآن** — بشرط الطلب نفسِه (`orders.OpenNowSQL`).
	OpenNow bool `json:"open_now"`
	// OpensAt **موعدُ فتحه القادم** — و`null` ما دام مفتوحاً أو بلا دوام.
	OpensAt *time.Time `json:"opens_at"`
}

// featuredStores **شريطُ «متاجر مميّزة»** — بترتيب الأدمن ثمّ الاسم.
//
// **وسوقُ المدينة وحدَه** (`cityWhere`) كالأقسام: **مربّعٌ لمتجرٍ لا يصله
// يُفتح على أصنافٍ لا تُطلب.** **والفعّالُ وحدَه** — الموقوفُ والمحظورُ لا
// يظهران.
//
// **وحدٌّ أعلى لا حدُّ عدد**: المالكُ لم يحدّ عددَها، **والحدُّ هنا
// حارسُ استعلامٍ لا قرارُ سوق.**
func (s *Server) featuredStores(r *http.Request) ([]featuredStore, error) {
	where, geo := s.cityWhere(r.Context(), scopeFrom(r), 1)
	rows, err := s.pg.Query(r.Context(), `
		SELECT m.id, m.name, lm.path, lm.thumb_path,
		       `+orders.OpenNowSQL+`, `+orders.NextOpenSQL+`
		FROM merchants m
		LEFT JOIN media lm ON lm.id = m.logo_media_id
		WHERE m.featured AND m.status = 'active'`+where+`
		ORDER BY m.featured_order, m.name, m.id
		LIMIT 200`, geo...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []featuredStore{}
	for rows.Next() {
		var f featuredStore
		if err := rows.Scan(&f.ID, &f.Name, &f.LogoURL, &f.LogoThumbURL, &f.OpenNow, &f.OpensAt); err != nil {
			return nil, err
		}
		f.LogoURL = media.URLForPtr(f.LogoURL)
		f.LogoThumbURL = media.URLForPtr(f.LogoThumbURL)
		out = append(out, f)
	}
	return out, rows.Err()
}

// storeSectionHead **قسمُ منصّةٍ كما يُعنوَن في صفحة المتجر.**
type storeSectionHead struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Icon          string  `json:"icon"`
	ImageURL      *string `json:"image_url"`
	ImageThumbURL *string `json:"image_thumb_url"`
}

// storeSection **قسمٌ وأصنافُ المتجر فيه.**
type storeSection struct {
	storeSectionHead
	Items []publicItem `json:"items"`
}

// storeHeader **رأسُ صفحة المتجر.**
type storeHeader struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	LogoURL      *string    `json:"logo_url"`
	LogoThumbURL *string    `json:"logo_thumb_url"`
	OpenNow      bool       `json:"open_now"`
	OpensAt      *time.Time `json:"opens_at"`
	// PrepMinutes **مدّةُ تحضيره المعتادة** — «يجهز خلال ٢٠ دقيقة».
	PrepMinutes int `json:"prep_minutes"` // Today **دوامُ اليوم** (ملاحظةُ المالك ٢٠٢٦-١٠-١٨: «تحط ساعات الدوام… كل يوم بيومو»)
	// — يُعرض بدل «مفتوح هلّق»: «مفتوح ٢٤ ساعة» أو «من … لـ …» أو «مغلق اليوم».
	Today todayHours `json:"today"`
}

// todayHours **دوامُ المتجر اليومَ بتوقيت دمشق.** وبلا صفوفِ دوامٍ مفتوحٌ دائماً
// (`OpenNowSQL`)، **فيُقال ٢٤ ساعة.**
type todayHours struct {
	Closed bool   `json:"closed"`
	Open24 bool   `json:"open_24h"`
	Open   string `json:"open"`
	Close  string `json:"close"`
}

// groupStoreSections **يجمع أصنافَ المتجر تحت أقسامها** — بترتيب الأقسام
// كما وصلت، **والأصنافُ بترتيبها داخل كلّ قسم.**
//
// **وقسمٌ بلا صنفٍ لا يُعنوَن** — عنوانٌ فوق فراغٍ يُقرأ عطباً. **وصنفٌ قسمُه
// غائبٌ عن القائمة لا يُعرض** — لا عنوانَ يحمله.
//
// **دالّةٌ خالصةٌ بلا قاعدة** — فتُختبر بلا قاعدة.
func groupStoreSections(heads []storeSectionHead, items []publicItem) []storeSection {
	by := map[string][]publicItem{}
	for _, it := range items {
		by[it.SectionID] = append(by[it.SectionID], it)
	}
	out := []storeSection{}
	for _, h := range heads {
		got := by[h.ID]
		if len(got) == 0 {
			continue
		}
		out = append(out, storeSection{storeSectionHead: h, Items: got})
	}
	return out
}

// handlePublicStore **صفحةُ متجر** — رأسُه وأصنافُه مجموعةً بأقسام المنصّة.
//
// **والصنفُ بشكل القسم حرفاً** (`publicItem`) — فتُعيد الشاشاتُ استعمالَ
// بطاقتها ونافذتها. **وشرطُ الظهور شرطُ القسم** (`itemSelect` ومدينةُ
// الزبون) — **وصنفٌ مخفيٌّ هناك لا يتسرّب من هنا.**
//
// **وما لا وجودَ له أو غيرُ فعّالٍ يُردّ ٤٠٤** — كالقسم (BUG-005): **«فارغٌ»
// و«لا وجودَ له» جوابان لمعنيين.** **ومتجرٌ خارجَ سوقِ مدينة الزبون كذلك**:
// لا يُطلب منه، **فلا يُفتح له باب.**
func (s *Server) handlePublicStore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	where, geo := s.cityWhere(r.Context(), scopeFrom(r), 2)
	var h storeHeader
	err := s.pg.QueryRow(r.Context(), `
		SELECT m.id, m.name, lm.path, lm.thumb_path,
		       `+orders.OpenNowSQL+`, `+orders.NextOpenSQL+`,
		       m.default_prep_minutes
		FROM merchants m
		LEFT JOIN media lm ON lm.id = m.logo_media_id
		WHERE m.id = $1::uuid AND m.status = 'active'`+where,
		append([]any{id}, geo...)...).
		Scan(&h.ID, &h.Name, &h.LogoURL, &h.LogoThumbURL, &h.OpenNow, &h.OpensAt, &h.PrepMinutes)
	if err != nil {
		// **ولا صفّ ⇒ ٤٠٤** — ولا يُفرَّق بين «لا وجود» و«موقوف»: **الفرقُ
		// يقول لزائرٍ أنّ المعرّفَ لمتجرٍ حقيقيّ.**
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	{
		var closed bool
		var open, close string
		err := s.pg.QueryRow(r.Context(), `
			SELECT closed, to_char(open_time, 'HH24:MI'), to_char(close_time, 'HH24:MI')
			FROM merchant_hours
			WHERE merchant_id = $1::uuid
			  AND day_of_week = EXTRACT(dow FROM (now() AT TIME ZONE 'Asia/Damascus'))::int`, id).
			Scan(&closed, &open, &close)
		switch {
		case err != nil:
			// **لا صفوفَ دوامٍ أصلاً ⇒ مفتوحٌ دائماً** (`OpenNowSQL`)، **وله دوامٌ بلا صفِّ اليوم ⇒ مغلقٌ اليوم.**
			var any bool
			_ = s.pg.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM merchant_hours WHERE merchant_id = $1::uuid)`, id).Scan(&any)
			h.Today = todayHours{Open24: !any, Closed: any}
		case closed:
			h.Today = todayHours{Closed: true}
		case open == close:
			h.Today = todayHours{Open24: true}
		default:
			h.Today = todayHours{Open: open, Close: close}
		}
	}
	h.LogoURL = media.URLForPtr(h.LogoURL)
	h.LogoThumbURL = media.URLForPtr(h.LogoThumbURL)

	// **والأصنافُ بأعمدة القسم وشرطِه** — والمتاحُ أوّلاً داخل كلّ قسم.
	items, err := s.queryItems(r, itemSelect+`
		  AND m.id = $1::uuid`+where+`
		ORDER BY ps.sort_order, ps.name, ps.id,
		         (i.available AND `+orders.OpenNowSQL+`) DESC, i.sort_order, i.name
		LIMIT 500`, append([]any{id}, geo...)...)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// **وأقسامُ ما ظهر وحدَها** — بصورها ورموزها لعناوين الصفحة وشريطها.
	ids := []string{}
	seen := map[string]bool{}
	for _, it := range items {
		if !seen[it.SectionID] {
			seen[it.SectionID] = true
			ids = append(ids, it.SectionID)
		}
	}
	heads := []storeSectionHead{}
	if len(ids) > 0 {
		rows, err := s.pg.Query(r.Context(), `
			SELECT ps.id, ps.name, ps.icon, sm.path, sm.thumb_path
			FROM platform_sections ps
			LEFT JOIN media sm ON sm.id = ps.image_media_id
			WHERE ps.id = ANY($1::uuid[]) AND ps.active
			ORDER BY ps.sort_order, ps.name, ps.id`, ids)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		for rows.Next() {
			var x storeSectionHead
			if err := rows.Scan(&x.ID, &x.Name, &x.Icon, &x.ImageURL, &x.ImageThumbURL); err != nil {
				rows.Close()
				s.respondErr(w, err)
				return
			}
			x.ImageURL = media.URLForPtr(x.ImageURL)
			x.ImageThumbURL = media.URLForPtr(x.ImageThumbURL)
			heads = append(heads, x)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			s.respondErr(w, err)
			return
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"store":    h,
		"sections": groupStoreSections(heads, items),
	})
}
