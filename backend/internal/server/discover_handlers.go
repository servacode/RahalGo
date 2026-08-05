package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// البحث والمفضّلة — كيف يجد الزبون ما يريد.

// handlePublicSearch يبحث في المتاجر **وفي أصنافها**.
//
// البحث في الأصناف لا في أسماء المتاجر وحدها: من يشتهي «شاورما» لا يعرف اسم
// المطعم الذي يصنعها. وهذا ما يحوّل البحث من دليلٍ إلى أداة.
//
// عام بلا حساب — التصفّح كله كذلك.
func (s *Server) handlePublicSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) < 2 {
		httpx.JSON(w, http.StatusOK, []any{})
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT m.id, m.name, c.icon, lm.thumb_path, m.emergency_closed,
		       -- سبب الظهور: اسمُ المتجر أم صنفٌ فيه؟ يُعرض للزبون كي يفهم النتيجة
		       COALESCE((SELECT string_agg(x.name, '، ')
		                 FROM (SELECT i.name FROM menu_items i
		                       WHERE i.merchant_id = m.id AND i.available AND i.approved
		                         AND i.name ILIKE '%'||$1||'%' LIMIT 3) x), '')
		FROM merchants m
		JOIN categories c ON c.id = m.category_id
		LEFT JOIN media lm ON lm.id = m.logo_media_id
		WHERE m.status = 'active'
		  AND (m.name ILIKE '%'||$1||'%'
		       OR EXISTS (SELECT 1 FROM menu_items i
		                  WHERE i.merchant_id = m.id AND i.available AND i.approved
		                    AND i.name ILIKE '%'||$1||'%'))
		ORDER BY (m.name ILIKE '%'||$1||'%') DESC, m.name
		LIMIT 30`, q)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type hit struct {
		ID           string  `json:"id"`
		Name         string  `json:"name"`
		CategoryIcon string  `json:"category_icon"`
		LogoThumbURL *string `json:"logo_thumb_url"`
		Closed       bool    `json:"emergency_closed"`
		MatchedItems string  `json:"matched_items"`
	}
	out := []hit{}
	for rows.Next() {
		var h hit
		if err := rows.Scan(&h.ID, &h.Name, &h.CategoryIcon, &h.LogoThumbURL,
			&h.Closed, &h.MatchedItems); err != nil {
			s.respondErr(w, err)
			return
		}
		h.LogoThumbURL = media.URLForPtr(h.LogoThumbURL)
		out = append(out, h)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleMyFavorites أصنافُه المفضّلة — **صحونٌ لا متاجر.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «المفضّلة لازم تكون ع صنف وليس ع متجر».)
//
// # ولماذا يُعاد معها متجرُها وسعرُها
//
// **بطاقةٌ باسم الصنف وحدَه لا تكفي**: «شاورما دجاج» في ثلاثة مطاعم، **ومن
// رأى اسماً بلا مطعمٍ لا يعرف أيَّها حفظ.**
//
// **والسعرُ يُقرأ الآن لا يومَ الحفظ**: من حفظ صحناً بألفين ورآه بألفين وقد
// صار بثلاثة **يكتشفها في السلّة** — وهي أسوأُ لحظةٍ لاكتشافها.
//
// # وما لا يُعرض
//
// **الصنفُ الذي رفعه المتجرُ من قائمته يسقط من الجدول نفسِه** (`ON DELETE
// CASCADE`). **والذي أوقفه مؤقّتاً يبقى ويُقال عنه** — لأنّه يعود.
func (s *Server) handleMyFavorites(w http.ResponseWriter, r *http.Request) {
	// **والاستعلامُ هو استعلامُ التصفّح نفسُه** (`favoritesSelect`) — بعمودٍ
	// واحدٍ يفرّقهما: من صاحبُها.
	//
	// **وكان مرتجَلاً هنا** فقرأ عمودَ `price` الخام بدل سعر البيع **ونسي
	// الخصمَ السارّي** — فقالت المفضّلةُ للصنف الواحد رقماً والقسمُ غيرَه.
	//
	// **والردُّ بشكل `publicItem`** لأنّ الشاشةَ تعرضه ببطاقة التصفّح نفسِها:
	// **يُضغط فيُفتح في نافذةٍ لا في صفحة**، كالقسم والعروض تماماً. (قرارُ
	// المالك ٢٠٢٦-٠٨-٠٥: «اتّفقنا نافذةٌ منبثقةٌ نفسَ نظام الصفحة الرئيسيّة».)
	s.scanItems(w, r, favoritesSelect, userIDFrom(r))
}

// handleToggleFavorite يضيف صنفاً للمفضّلة أو يزيله — عمليةٌ واحدة لا اثنتان.
//
// الزرّ في الواجهة زرٌّ واحد يُضغط فينقلب، فمنطقُه في الخادم يجب أن يطابقه:
// نقطتان (إضافة/حذف) تفتحان باب اختلافٍ بين ما تظنّه الشاشة وما في القاعدة.
func (s *Server) handleToggleFavorite(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	itemID := chi.URLParam(r, "id")
	var added bool
	if err := s.pg.QueryRow(r.Context(), `
		WITH removed AS (
			DELETE FROM user_favorites WHERE user_id = $1 AND menu_item_id = $2 RETURNING 1
		), added AS (
			INSERT INTO user_favorites (user_id, menu_item_id)
			SELECT $1, $2 WHERE NOT EXISTS (SELECT 1 FROM removed) RETURNING 1
		)
		SELECT EXISTS (SELECT 1 FROM added)`, uid, itemID).Scan(&added); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"favorite": added})
}
