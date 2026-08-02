package server

// تصفّحُ المنصة — **بالأصناف لا بالمتاجر.**
//
// # المسألة
//
// التصفّحُ كان «اختر متجراً ثمّ صنفاً»، **والزبونُ لا يفكّر هكذا**: يشتهي
// شاورما ولا يعرف من يصنع أفضلَها، ولا يريد أن يقارن بين عشرة مطاعم.
//
// **وأخطرُ منه أنه يكشف المصدر.** في مدينةٍ يعرف أهلُها بعضهم، **زبونٌ رأى
// اسمَ المطعم يتّصل به مباشرةً في المرّة القادمة** — يوفّر رسمَ التوصيل
// والمطعمُ يوفّر عمولتنا. **وكلُّ منصةِ توصيلٍ تموت من هذا الباب لا من غيره.**
//
// # ولماذا لا يُرسَل معرّفُ المتجر أصلاً
//
// **الإخفاءُ عند المصدر لا عند العرض.** من فتح أدوات المتصفّح قرأ الردَّ كما
// هو، **ومعرّفٌ في الردّ يُفتح به `/public/merchants/{id}` فيُقرأ الاسمُ
// كاملاً.** فلا يخرج من هنا إلّا ما يلزم للطلب: معرّفُ الصنف وسعرُه.

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// publicItem صنفٌ كما يراه الزبون — **بلا مصدره.**
type publicItem struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	Price         int64      `json:"price"`
	ImageThumbURL *string    `json:"image_thumb_url"`
	Available     bool       `json:"available"`
	SourceClosed  bool       `json:"source_closed"`
	SourceOpensAt *time.Time `json:"source_opens_at"`
	SectionID     string     `json:"section_id"`
	SectionName   string     `json:"section_name"`
}

// itemSelect ما يُقرأ لكلّ صنفٍ معروض.
//
// **وسعرُ البيع يُحسب في Go لا في SQL** — فالمعادلةُ في `pricing` وحدَها،
// **ولو كُتبت هنا لَافترقت عن حسبةِ الطلب** فيرى الزبونُ سعراً ويُحاسَب بغيره.
const itemSelect = `
	SELECT i.id, i.name, i.description, i.merchant_price, i.margin_override,
	       im.thumb_path, i.available, ps.id, ps.name, ps.margin_override,
	       ` + orders.OpenNowSQL + `, ` + orders.NextOpenSQL + `
	FROM menu_items i
	JOIN merchants m ON m.id = i.merchant_id
	JOIN platform_sections ps ON ps.id = i.platform_section_id
	LEFT JOIN media im ON im.id = i.image_media_id
	WHERE m.status = 'active' AND ps.active`

func (s *Server) scanItems(w http.ResponseWriter, r *http.Request, sql string, args ...any) {
	rule := pricing.RuleFrom(r.Context(), s.settings)
	rows, err := s.pg.Query(r.Context(), sql, args...)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	out := []publicItem{}
	for rows.Next() {
		var it publicItem
		var cost int64
		var itemMargin, sectionMargin *int64
		var open bool
		var opensAt *time.Time
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &cost, &itemMargin,
			&it.ImageThumbURL, &it.Available, &it.SectionID, &it.SectionName,
			&sectionMargin, &open, &opensAt); err != nil {
			s.respondErr(w, err)
			return
		}
		it.Price = rule.SalePrice(cost, itemMargin, sectionMargin)
		it.ImageThumbURL = media.URLForPtr(it.ImageThumbURL)
		it.SourceClosed = !open
		if !open {
			it.SourceOpensAt = opensAt
		}
		out = append(out, it)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": out})
}

// handlePublicSections أقسامُ المنصة وعددُ ما فيها.
//
// **والعددُ لما هو متاحٌ الآن لا لكلّ ما سُجّل.** قسمٌ يقول «١٢ صنفاً» ثمّ
// يُفتح على ثلاثةٍ **يجعل الزبونَ يشكّ في كلّ رقمٍ بعده** — وقد نام تسعةٌ منها
// مع مصادرها.
func (s *Server) handlePublicSections(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT ps.id, ps.name, ps.icon,
		       count(i.id) FILTER (WHERE i.available AND `+orders.OpenNowSQL+`)
		FROM platform_sections ps
		LEFT JOIN menu_items i ON i.platform_section_id = ps.id
		LEFT JOIN merchants m ON m.id = i.merchant_id AND m.status = 'active'
		WHERE ps.active
		GROUP BY ps.id, ps.name, ps.icon, ps.sort_order
		ORDER BY ps.sort_order, ps.name`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type section struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Icon  string `json:"icon"`
		Count int    `json:"count"`
	}
	out := []section{}
	for rows.Next() {
		var x section
		if err := rows.Scan(&x.ID, &x.Name, &x.Icon, &x.Count); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sections": out})
}

// handlePublicSectionItems أصنافُ قسمٍ — **من كلّ المصادر مختلطةً.**
//
// **والمتاحُ أوّلاً لا الأرخصُ أوّلاً.** ترتيبٌ بالسعر يجعل الزبونَ يقارن
// **وهو ما لا نريده**: الأصنافُ عندنا لا عند متاجر، والفرقُ في السعر فرقُ
// صنفٍ لا فرقُ بائع. **والنائمُ يُعرض آخراً ولا يُخفى** — من رآه عرف أنّه
// موجودٌ وعاد له.
func (s *Server) handlePublicSectionItems(w http.ResponseWriter, r *http.Request) {
	s.scanItems(w, r, itemSelect+`
		  AND ps.id = $1
		ORDER BY (i.available AND `+orders.OpenNowSQL+`) DESC, i.sort_order, i.name
		LIMIT 200`, chi.URLParam(r, "id"))
}

// handlePublicItem صنفٌ واحدٌ بتفصيله — **صفحةُ الصنف.**
func (s *Server) handlePublicItem(w http.ResponseWriter, r *http.Request) {
	rule := pricing.RuleFrom(r.Context(), s.settings)
	var it publicItem
	var cost int64
	var itemMargin, sectionMargin *int64
	var open bool
	var opensAt *time.Time
	if err := s.pg.QueryRow(r.Context(), itemSelect+` AND i.id = $1`,
		chi.URLParam(r, "id")).
		Scan(&it.ID, &it.Name, &it.Description, &cost, &itemMargin,
			&it.ImageThumbURL, &it.Available, &it.SectionID, &it.SectionName,
			&sectionMargin, &open, &opensAt); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	it.Price = rule.SalePrice(cost, itemMargin, sectionMargin)
	it.ImageThumbURL = media.URLForPtr(it.ImageThumbURL)
	it.SourceClosed = !open
	if !open {
		it.SourceOpensAt = opensAt
	}

	// **والخياراتُ تُقرأ معه** — «حجم» و«إضافات»، وهي جزءٌ من سعره.
	groups, err := s.catalog.ItemModifiers(r.Context(), it.ID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"item": it, "modifiers": groups})
}

// handleSearchItems بحثٌ بالأصناف — **لا بالمتاجر.**
//
// **ومن يشتهي صنفاً لا يعرف اسم المتجر الذي يصنعه.** والبحثُ القديم كان يعيد
// متاجر ويقول «طابق في: شاورما، فروج» — **فيُقرأ اسمُ المتجر أوّلاً وهو ما
// نخفيه**، ويُطلب من الزبون خطوةٌ زائدة: يفتح المتجرَ ثمّ يبحث فيه ثانيةً.
func (s *Server) handleSearchItems(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) < 2 {
		httpx.JSON(w, http.StatusOK, map[string]any{"items": []publicItem{}})
		return
	}
	s.scanItems(w, r, itemSelect+`
		  AND (i.name ILIKE '%'||$1||'%' OR ps.name ILIKE '%'||$1||'%')
		ORDER BY (i.name ILIKE $1||'%') DESC,
		         (i.available AND `+orders.OpenNowSQL+`) DESC, i.name
		LIMIT 60`, q)
}
