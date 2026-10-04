package server

// ══════════════════════════════════════════════════════════════════════
// **«السوق» في لوحة الإدارة — الأصنافُ لا المتاجر** (قراراتُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// «السوق عندي يحوي الأصناف الخاصّة بكلّ المتاجر، لأنّ المتاجرَ أصلاً لا تُعرض
// على الزبون». **فأحدَ عشرَ صنفاً باسم «شاورما دجاج صحن» ليست تكراراً** —
// أحدَ عشرَ صنفاً لأحدَ عشرَ متجراً، **ولا دمجَ للأصناف هنا أبداً.**
//
// # التبويباتُ وأبوابُها
//
//	الأقسام        ←  `/sections` (وترتيبُها بالسحب `PUT /sections/order`)
//	الأصناف        ←  `/market/items` + `/market/items/bulk`
//	المتاجر        ←  `/market/stores`
//	جودةُ البيانات ←  `/market/quality` + `/market/test-data/delete`
//
// # «المضافُ حديثاً»
//
// «لازم أعرف الأصناف المضافة حديثاً»: **الأحدثُ أوّلاً**، وعلامةُ «جديد» على
// ما أُضيف في آخر `marketNewHours` ساعة، **وعدّادٌ في القائمة الجانبيّة** يعدّ
// ما أُضيف بعد آخر فتحٍ للسوق — **لكلّ موظّفٍ وحدَه** — ويُصفَّر بفتحه.

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// marketNewHours **عمرُ علامة «جديد»** — ثلاثةُ أيّام.
//
// **ثابتةٌ لا تتبع آخرَ فتح**: علامةٌ تختفي بفتح الصفحة تختفي قبل أن تُقرأ
// (الصفحةُ نفسُها تُصفّر العدّاد). **والعدّادُ يتبع آخرَ فتحٍ، والعلامةُ عمرَ
// الصنف** — سؤالان مختلفان. **ومن لم يفتح السوقَ قطّ يُعدّ له ما في المدّة نفسِها.**
const marketNewHours = 72

// marketTestMarker **علامةُ البيانات التجريبيّة في الاسم** — نمطٌ في القاعدة.
//
// **يُقرأ ولا يُحذف به**: الحذفُ بقائمةٍ صريحةٍ أكّدها الموظّف (قرارُ المالك:
// «لا حذفَ آليّ»)، **والنمطُ يُعاد فحصُه عند الحذف** فلا يُحذف بالخطأ ما
// ليس تجريبيّاً ولو أُرسل معرّفُه.
const marketTestMarker = `(^|[^[:alnum:]])(qa|test|demo|e2e)([^[:alnum:]]|$)|اختبار|تجريب|تجربة`

// marketBulkMax **سقفُ الإجراء الجماعيّ** — صفحتان من الجدول.
const marketBulkMax = 500

type marketItem struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	MerchantPrice  int64     `json:"merchant_price"`
	Available      bool      `json:"available"`
	MerchantID     string    `json:"merchant_id"`
	MerchantName   string    `json:"merchant_name"`
	MerchantStatus string    `json:"merchant_status"`
	SectionID      string    `json:"section_id"`
	SectionName    string    `json:"section_name"`
	ThumbURL       *string   `json:"thumb_url"`
	CreatedAt      time.Time `json:"created_at"`
	// IsNew **أُضيف في آخر `marketNewHours` ساعة** — علامةُ «جديد».
	IsNew bool `json:"is_new"`
}

// handleMarketItems **أصنافُ كلّ المتاجر — الأحدثُ أوّلاً.**
//
// والترشيحُ في المحرّك لا في الشاشة (`q` · `section` · `merchant` · `state`
// · `no_image` · `new`) — **وترشيحٌ فوق صفحةٍ وعدٌ بترشيح.**
func (s *Server) handleMarketItems(w http.ResponseWriter, r *http.Request) {
	pg := pagingOf(r, 50)
	qs := r.URL.Query()
	q := strings.TrimSpace(qs.Get("q"))
	section := qs.Get("section")
	if section != "" && !isUUID(section) {
		section = ""
	}
	merchant := qs.Get("merchant")
	if merchant != "" && !isUUID(merchant) {
		merchant = ""
	}
	state := qs.Get("state")
	noImage := qs.Get("no_image") == "1"
	onlyNew := qs.Get("new") == "1"

	const scope = `
		FROM menu_items i
		JOIN merchants m ON m.id = i.merchant_id
		JOIN platform_sections ps ON ps.id = i.platform_section_id
		WHERE ($1 = '' OR i.name ILIKE '%'||$1||'%' OR m.name ILIKE '%'||$1||'%')
		  AND ($2 = '' OR i.platform_section_id = $2::uuid)
		  AND ($3 = '' OR i.merchant_id = $3::uuid)
		  AND ($4 = '' OR (CASE WHEN m.status <> 'active' THEN 'store_off'
		                        WHEN NOT i.available THEN 'out'
		                        ELSE 'live' END) = $4)
		  AND (NOT $5 OR i.image_media_id IS NULL)
		  AND (NOT $6 OR i.created_at > now() - make_interval(hours => $7))`
	args := []any{q, section, merchant, state, noImage, onlyNew, marketNewHours}

	var total int
	if err := s.pg.QueryRow(r.Context(), `SELECT count(*)`+scope, args...).
		Scan(&total); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT i.id::text, i.name, i.merchant_price, i.available,
		       m.id::text, m.name, m.status, ps.id::text, ps.name,
		       (SELECT im.thumb_path FROM media im WHERE im.id = i.image_media_id),
		       i.created_at,
		       i.created_at > now() - make_interval(hours => $7)`+scope+`
		ORDER BY i.created_at DESC, i.id
		LIMIT $8 OFFSET $9`, append(args, pg.PerPage, pg.Offset)...)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []marketItem{}
	for rows.Next() {
		var x marketItem
		if err := rows.Scan(&x.ID, &x.Name, &x.MerchantPrice, &x.Available,
			&x.MerchantID, &x.MerchantName, &x.MerchantStatus,
			&x.SectionID, &x.SectionName, &x.ThumbURL, &x.CreatedAt, &x.IsNew); err != nil {
			s.respondErr(w, err)
			return
		}
		x.ThumbURL = media.URLForPtr(x.ThumbURL)
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	body := paged("items", out, total, pg)
	body["new_hours"] = marketNewHours
	httpx.JSON(w, http.StatusOK, body)
}

// handleMarketItemsBulk **إجراءٌ على أصنافٍ مختارة** — متوفّر · غير متوفّر · نقلٌ إلى قسم.
//
// **ولا حذفَ جماعيّاً للأصناف** — صنفُ المتجر عملُه، **وحذفُ مئةٍ بضغطةٍ لا
// يُردّ.** (والتجريبيُّ وحدَه يُحذف — من «جودة البيانات» بقائمةٍ مؤكَّدة.)
func (s *Server) handleMarketItemsBulk(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		IDs       []string `json:"ids"`
		Action    string   `json:"action"`
		SectionID string   `json:"section_id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > marketBulkMax {
		s.respondErr(w, errValidation)
		return
	}
	for _, id := range req.IDs {
		if !isUUID(id) {
			s.respondErr(w, errValidation)
			return
		}
	}
	ctx := r.Context()
	var n int64
	switch req.Action {
	case "available", "unavailable":
		tag, err := s.pg.Exec(ctx, `
			UPDATE menu_items SET available = $2, updated_at = now()
			 WHERE id = ANY($1::uuid[])`, req.IDs, req.Action == "available")
		if err != nil {
			s.respondErr(w, err)
			return
		}
		n = tag.RowsAffected()
	case "move":
		if !isUUID(req.SectionID) {
			s.respondErr(w, errSectionMoveTarget)
			return
		}
		tx, err := s.pg.Begin(ctx)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		var ok bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM platform_sections WHERE id = $1)`,
			req.SectionID).Scan(&ok); err != nil {
			s.respondErr(w, err)
			return
		}
		if !ok {
			s.respondErr(w, errSectionMoveTarget)
			return
		}
		tag, err := tx.Exec(ctx, `
			UPDATE menu_items SET platform_section_id = $2, updated_at = now()
			 WHERE id = ANY($1::uuid[])`, req.IDs, req.SectionID)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		n = tag.RowsAffected()
		// **ومتجرٌ أعلن أقسامَه يُضاف إليها القسمُ الجديد** — وإلّا عرض نموذجُ
		// الصنف قسماً لا يجده في اختياره. **والمتجرُ الذي لم يُعلن (= الكلّ)
		// لا يُمسّ** — صفٌّ واحدٌ يحصره في قسمٍ واحد.
		if _, err := tx.Exec(ctx, `
			INSERT INTO store_sections (store_id, section_id)
			SELECT DISTINCT i.merchant_id, $2::uuid FROM menu_items i
			 WHERE i.id = ANY($1::uuid[])
			   AND EXISTS (SELECT 1 FROM store_sections x WHERE x.store_id = i.merchant_id)
			ON CONFLICT DO NOTHING`, req.IDs, req.SectionID); err != nil {
			s.respondErr(w, err)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			s.respondErr(w, err)
			return
		}
	default:
		s.respondErr(w, errValidation)
		return
	}
	s.audit(r, "market.items_bulk", "menu_item", "", map[string]any{
		"action": req.Action, "ids": req.IDs, "section_id": req.SectionID, "affected": n,
	})
	s.touch("menu", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"affected": n})
}

// marketSeenSQL **آخرُ فتحٍ للسوق لهذا الموظّف** — ومن لم يفتحه قطّ يُعدّ له
// ما أُضيف في مدّة علامة «جديد».
const marketSeenSQL = `COALESCE(
	(SELECT seen_at FROM admin_market_seen WHERE user_id = $1::uuid),
	now() - make_interval(hours => $2))`

// handleMarketNewCount **عدّادُ القائمة الجانبيّة** — ما أُضيف بعد آخر فتح.
func (s *Server) handleMarketNewCount(w http.ResponseWriter, r *http.Request) {
	var n int
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*) FROM menu_items WHERE created_at > `+marketSeenSQL,
		userIDFrom(r), marketNewHours).Scan(&n); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"count": n})
}

// handleMarketSeen **فُتح السوق — فيُصفَّر عدّادُ هذا الموظّف وحدَه.**
func (s *Server) handleMarketSeen(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	if !isUUID(uid) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if _, err := s.pg.Exec(r.Context(), `
		INSERT INTO admin_market_seen (user_id, seen_at) VALUES ($1, now())
		ON CONFLICT (user_id) DO UPDATE SET seen_at = now()`, uid); err != nil {
		s.respondErr(w, err)
		return
	}
	// **وعدّادُه في القائمة يُعاد قراءتُه** — له وحدَه لا لكلّ الموظّفين.
	s.touchUser(uid, "market_seen")
	httpx.JSON(w, http.StatusOK, map[string]any{"seen": true})
}

type marketStore struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Status          string  `json:"status"`
	EmergencyClosed bool    `json:"emergency_closed"`
	LogoURL         *string `json:"logo_url"`
	Items           int     `json:"items"`
	LiveItems       int     `json:"live_items"`
	NewItems        int     `json:"new_items"`
	NoImageItems    int     `json:"no_image_items"`
	HasHours        bool    `json:"has_hours"`
}

// handleMarketStores **المتاجرُ بصفحةٍ موحّدة** — تُدار في اللوحة ولا تُعرض
// على الزبون. **واسمُها «المتاجر» لا «المورّدون»** (تصحيحُ المالك).
func (s *Server) handleMarketStores(w http.ResponseWriter, r *http.Request) {
	pg := pagingOf(r, 50)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	var total int
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*) FROM merchants m WHERE ($1 = '' OR m.name ILIKE '%'||$1||'%')`,
		q).Scan(&total); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT m.id::text, m.name, m.status, m.emergency_closed, lm.thumb_path,
		       count(i.id),
		       count(i.id) FILTER (WHERE i.available),
		       count(i.id) FILTER (WHERE i.created_at > now() - make_interval(hours => $2)),
		       count(i.id) FILTER (WHERE i.image_media_id IS NULL),
		       EXISTS (SELECT 1 FROM merchant_hours h WHERE h.merchant_id = m.id)
		FROM merchants m
		LEFT JOIN media lm ON lm.id = m.logo_media_id
		LEFT JOIN menu_items i ON i.merchant_id = m.id
		WHERE ($1 = '' OR m.name ILIKE '%'||$1||'%')
		GROUP BY m.id, lm.thumb_path
		ORDER BY m.name, m.id
		LIMIT $3 OFFSET $4`, q, marketNewHours, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []marketStore{}
	for rows.Next() {
		var x marketStore
		if err := rows.Scan(&x.ID, &x.Name, &x.Status, &x.EmergencyClosed, &x.LogoURL,
			&x.Items, &x.LiveItems, &x.NewItems, &x.NoImageItems, &x.HasHours); err != nil {
			s.respondErr(w, err)
			return
		}
		x.LogoURL = media.URLForPtr(x.LogoURL)
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, paged("stores", out, total, pg))
}

type qualityItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	MerchantID   string    `json:"merchant_id"`
	MerchantName string    `json:"merchant_name"`
	SectionName  string    `json:"section_name"`
	CreatedAt    time.Time `json:"created_at"`
}

type qualityStore struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Items  int    `json:"items"`
	Orders int    `json:"orders"`
}

// qualityListMax **سقفُ ما يُعرض من كلّ قائمة** — والعددُ الكاملُ يُرسَل معه.
const qualityListMax = 200

// handleMarketQuality **جودةُ البيانات** — أصنافٌ بلا صورة · متاجرُ بلا دوام ·
// البياناتُ التجريبيّة.
//
// **يُقرأ ولا يُصلَح وحدَه** — كلُّ قائمةٍ تقول ما ينقص، **والإصلاحُ قرارُ موظّف.**
func (s *Server) handleMarketQuality(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	readItems := func(where string, args ...any) ([]qualityItem, int, error) {
		var n int
		if err := s.pg.QueryRow(ctx, `
			SELECT count(*) FROM menu_items i JOIN merchants m ON m.id = i.merchant_id
			WHERE `+where, args...).Scan(&n); err != nil {
			return nil, 0, err
		}
		rows, err := s.pg.Query(ctx, `
			SELECT i.id::text, i.name, m.id::text, m.name, ps.name, i.created_at
			FROM menu_items i
			JOIN merchants m ON m.id = i.merchant_id
			JOIN platform_sections ps ON ps.id = i.platform_section_id
			WHERE `+where+`
			ORDER BY i.created_at DESC, i.id
			LIMIT `+strconv.Itoa(qualityListMax), args...)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		out := []qualityItem{}
		for rows.Next() {
			var x qualityItem
			if err := rows.Scan(&x.ID, &x.Name, &x.MerchantID, &x.MerchantName,
				&x.SectionName, &x.CreatedAt); err != nil {
				return nil, 0, err
			}
			out = append(out, x)
		}
		return out, n, rows.Err()
	}
	readStores := func(where string, args ...any) ([]qualityStore, error) {
		rows, err := s.pg.Query(ctx, `
			SELECT m.id::text, m.name, m.status,
			       (SELECT count(*) FROM menu_items i WHERE i.merchant_id = m.id),
			       (SELECT count(*) FROM orders o WHERE o.merchant_id = m.id)
			FROM merchants m WHERE `+where+`
			ORDER BY m.name, m.id`, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := []qualityStore{}
		for rows.Next() {
			var x qualityStore
			if err := rows.Scan(&x.ID, &x.Name, &x.Status, &x.Items, &x.Orders); err != nil {
				return nil, err
			}
			out = append(out, x)
		}
		return out, rows.Err()
	}

	noImage, noImageN, err := readItems(`i.image_media_id IS NULL`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	noHours, err := readStores(
		`NOT EXISTS (SELECT 1 FROM merchant_hours h WHERE h.merchant_id = m.id)`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	testStores, err := readStores(`m.name ~* $1`, marketTestMarker)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والصنفُ التجريبيُّ في متجرٍ حقيقيّ** — أمّا أصنافُ المتجر التجريبيّ
	// فتُعدّ معه ولا تُكرَّر هنا.
	testItems, testItemsN, err := readItems(`i.name ~* $1 AND NOT (m.name ~* $1)`,
		marketTestMarker)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"items_without_image":       noImage,
		"items_without_image_count": noImageN,
		"stores_without_hours":      noHours,
		"test_stores":               testStores,
		"test_items":                testItems,
		"test_items_count":          testItemsN,
		"limit":                     qualityListMax,
	})
}

// errNotTestData **ما أُرسل للحذف ليس تجريبيّاً** — فلا يُحذف شيءٌ منه.
var errNotTestData = httpx.NewError(http.StatusConflict,
	"not_test_data", "errors.not_test_data")

// errTestStoreHasOrders **متجرٌ تجريبيٌّ عليه طلبات** — والطلبُ أثرُ مال.
var errTestStoreHasOrders = httpx.NewError(http.StatusConflict,
	"test_store_has_orders", "errors.test_store_has_orders")

// handleMarketTestDataDelete **حذفُ البيانات التجريبيّة — بقائمةٍ صريحة.**
//
// (قرارُ المالك: «البياناتُ التجريبيّةُ وحذفُها» — **ولا حذفَ إلّا بتأكيدٍ
// يسرد ما سيُحذف، ولا حذفَ آليّاً أبداً.**)
//
// **والقائمةُ يرسلها من رآها وأكّدها**، **والمحرّكُ يعيد فحصَ كلِّ معرّف**:
//
//	صنف   ←  اسمُه أو اسمُ متجره يحمل علامةَ التجريبيّ
//	متجر  ←  اسمُه يحمل العلامة **ولا طلبَ عليه قطّ** — والطلبُ أثرُ مالٍ في
//	          الدفتر لا يُمسّ
//
// **وكلُّها أو لا شيء** — معاملةٌ واحدة: معرّفٌ واحدٌ لا يطابق يردّ الطلبَ كلَّه.
func (s *Server) handleMarketTestDataDelete(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		ItemIDs  []string `json:"item_ids"`
		StoreIDs []string `json:"store_ids"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if len(req.ItemIDs)+len(req.StoreIDs) == 0 ||
		len(req.ItemIDs) > marketBulkMax || len(req.StoreIDs) > marketBulkMax {
		s.respondErr(w, errValidation)
		return
	}
	for _, id := range append(append([]string{}, req.ItemIDs...), req.StoreIDs...) {
		if !isUUID(id) {
			s.respondErr(w, errValidation)
			return
		}
	}
	if req.ItemIDs == nil {
		req.ItemIDs = []string{}
	}
	if req.StoreIDs == nil {
		req.StoreIDs = []string{}
	}
	ctx := r.Context()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// ── ١ · كلُّ معرّفٍ يُطابق العلامة ─────────────────────────────
	var badItems, badStores, ordered int
	if err := tx.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM unnest($1::uuid[]) AS x(id)
		    WHERE NOT EXISTS (
		      SELECT 1 FROM menu_items i JOIN merchants m ON m.id = i.merchant_id
		       WHERE i.id = x.id AND (i.name ~* $3 OR m.name ~* $3))),
		  (SELECT count(*) FROM unnest($2::uuid[]) AS x(id)
		    WHERE NOT EXISTS (SELECT 1 FROM merchants m WHERE m.id = x.id AND m.name ~* $3)),
		  (SELECT count(*) FROM orders o WHERE o.merchant_id = ANY($2::uuid[]))`,
		req.ItemIDs, req.StoreIDs, marketTestMarker).
		Scan(&badItems, &badStores, &ordered); err != nil {
		s.respondErr(w, err)
		return
	}
	if badItems > 0 || badStores > 0 {
		s.respondErr(w, errNotTestData)
		return
	}
	if ordered > 0 {
		s.respondErr(w, errTestStoreHasOrders)
		return
	}

	// ── ٢ · الأصنافُ ثمّ المتاجر ────────────────────────────────────
	itemsTag, err := tx.Exec(ctx, `
		DELETE FROM menu_items
		 WHERE id = ANY($1::uuid[]) OR merchant_id = ANY($2::uuid[])`,
		req.ItemIDs, req.StoreIDs)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	var storesDeleted int64
	if len(req.StoreIDs) > 0 {
		if _, err := tx.Exec(ctx,
			`DELETE FROM menu_sections WHERE merchant_id = ANY($1::uuid[])`,
			req.StoreIDs); err != nil {
			s.respondErr(w, err)
			return
		}
		// **والمرشَّحُ يبقى سجلّاً** — يُفكّ عن متجره ولا يُحذف.
		if _, err := tx.Exec(ctx,
			`UPDATE merchant_leads SET merchant_id = NULL WHERE merchant_id = ANY($1::uuid[])`,
			req.StoreIDs); err != nil {
			s.respondErr(w, err)
			return
		}
		tag, err := tx.Exec(ctx,
			`DELETE FROM merchants WHERE id = ANY($1::uuid[])`, req.StoreIDs)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		storesDeleted = tag.RowsAffected()
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "market.test_data_delete", "menu_item", "", map[string]any{
		"item_ids": req.ItemIDs, "store_ids": req.StoreIDs,
		"items_deleted": itemsTag.RowsAffected(), "stores_deleted": storesDeleted,
	})
	s.touch("menu", "ops")
	s.touch("catalog", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"items_deleted": itemsTag.RowsAffected(), "stores_deleted": storesDeleted,
	})
}
