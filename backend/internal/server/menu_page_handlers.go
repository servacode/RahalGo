package server

// ══════════════════════════════════════════════════════════════════════
// Paged store menus — owner decision 2026-11-07: «ما يصير يكون بطيء»
// ══════════════════════════════════════════════════════════════════════
//
// A mall of 5,311 items made the store's items tab a 3 MB answer that took 45 s on a weak link
// (staging, 2026-11-07); malls may reach 15,000. So the store owner and the customer now read a
// light summary (sections with counts) and then small keyset pages filtered on the server.
//
// The old full answers stay untouched for old app versions, the admin and the rep:
// GET /merchant/stores/{id}/menu and GET /public/stores/{id}.

import (
	"github.com/servacode/rahalgo/backend/internal/offers"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// handleMerchantMenuSummary GET /merchant/stores/{id}/menu/summary — own store only.
func (s *Server) handleMerchantMenuSummary(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	sum, err := s.catalog.GetMenuSummary(r.Context(), merchantID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	sum.WaItemsOn = sum.WaItemsStore && s.settings.GetBool(r.Context(), "merchants.wa_items_enabled")
	httpx.JSON(w, http.StatusOK, sum)
}

// handleMerchantMenuItems GET /merchant/stores/{id}/menu/items — one keyset page of the own
// store's items: ?section= &q= &filter=unavailable|available|no_photo &item= &after= &limit= (60, max 200).
//
// The platform sale price never reaches the store owner — `price` carries his own price, as
// in `handleMerchantMenu`.
func (s *Server) handleMerchantMenuItems(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	qs := r.URL.Query()
	limit, _ := strconv.Atoi(qs.Get("limit"))
	page, err := s.catalog.ListMenuItems(r.Context(), merchantID, catalog.MenuQuery{
		SectionID: qs.Get("section"),
		Q:         qs.Get("q"),
		Filter:    qs.Get("filter"),
		ItemID:    qs.Get("item"),
		After:     qs.Get("after"),
		Limit:     limit,
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for i := range page.Items {
		page.Items[i].Price = page.Items[i].MerchantPrice
	}
	httpx.JSON(w, http.StatusOK, page)
}

// storeSectionCount a market section of the store page with how many of its items show there.
type storeSectionCount struct {
	storeSectionHead
	Count int `json:"count"`
}

// handlePublicStoreSummary GET /public/stores/{id}/summary — the store header and its market
// sections with counts, under the same visibility as the store page (`itemSelect` and the
// visitor's city). The items come page by page from /public/stores/{id}/items.
func (s *Server) handlePublicStoreSummary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h, where, geo, err := s.publicStoreHead(r, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT ps.id, ps.name, ps.icon, sm.path, sm.thumb_path, count(*)
		FROM menu_items i
		JOIN merchants m ON m.id = i.merchant_id
		JOIN platform_sections ps ON ps.id = i.platform_section_id
		LEFT JOIN media sm ON sm.id = ps.image_media_id
		WHERE m.status = 'active' AND ps.active AND m.id = $1::uuid`+where+`
		GROUP BY ps.id, sm.path, sm.thumb_path
		ORDER BY ps.sort_order, ps.name, ps.id`, append([]any{id}, geo...)...)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []storeSectionCount{}
	total := 0
	for rows.Next() {
		var x storeSectionCount
		if err := rows.Scan(&x.ID, &x.Name, &x.Icon, &x.ImageURL, &x.ImageThumbURL, &x.Count); err != nil {
			s.respondErr(w, err)
			return
		}
		x.ImageURL = media.URLForPtr(x.ImageURL)
		x.ImageThumbURL = media.URLForPtr(x.ImageThumbURL)
		total += x.Count
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	// **وكم صنفاً عليه عرضٌ ساري** (قرارُ المالك ٢٠٢٦-١١-١٢: زرُّ «🔥 العروض» بصفحة المتجر) — بالشرط
	// نفسِه الذي يعرض به الصنفُ حسمَه (`offers.LiveCond`)، **وصفرُه لا زرّ.**
	offersCount := 0
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*) FROM menu_items i
		JOIN merchants m ON m.id = i.merchant_id
		JOIN platform_sections ps ON ps.id = i.platform_section_id
		WHERE m.status = 'active' AND ps.active AND m.id = $1::uuid`+where+storeOffersCond,
		append([]any{id}, geo...)...).Scan(&offersCount); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"store": h, "sections": out, "total": total, "offers": offersCount})
}

// publicStoreCursor the sort key of the last item of a public store page: section order, name
// and id, "not available", item order, name and id.
type publicStoreCursor struct {
	S  int    `json:"s"`
	N  string `json:"n"`
	P  string `json:"p"`
	U  bool   `json:"u"`
	O  int    `json:"o"`
	I  string `json:"i"`
	ID string `json:"id"`
}

// handlePublicStoreItems GET /public/stores/{id}/items — one keyset page of a store's items as
// the customer sees them (`itemSelect`: sale price, discount, dollars, has_options):
// ?section= &q= &after= &limit= (60, max 200). Order as the store page: market section, then
// the available first, then the item's order and name.
func (s *Server) handlePublicStoreItems(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, where, geo, err := s.publicStoreHead(r, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	qs := r.URL.Query()
	limit, _ := strconv.Atoi(qs.Get("limit"))
	limit = catalog.ClampLimit(limit)
	args := append([]any{id}, geo...)
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	cond := ` AND m.id = $1::uuid` + where
	if sec := qs.Get("section"); sec != "" {
		if !catalog.ValidUUID(sec) {
			s.respondErr(w, errValidation)
			return
		}
		cond += ` AND i.platform_section_id = ` + arg(sec) + `::uuid`
	}
	if q := catalog.NormalizeQuery(qs.Get("q")); q != "" {
		cond += ` AND i.name ILIKE ` + arg(catalog.LikePattern(q)) + ` ESCAPE '\'`
	}
	// **«🔥 العروض»: ما عليه حسمٌ ساريٌ وحدَه** (قرارُ المالك ٢٠٢٦-١١-١٢) — عبر الأقسام كلِّها.
	if qs.Get("offers") == "1" {
		cond += storeOffersCond
	}
	total := -1
	after := qs.Get("after")
	if after == "" {
		if err := s.pg.QueryRow(r.Context(), `
			SELECT count(*) FROM menu_items i
			JOIN merchants m ON m.id = i.merchant_id
			JOIN platform_sections ps ON ps.id = i.platform_section_id
			WHERE m.status = 'active' AND ps.active`+cond, args...).Scan(&total); err != nil {
			s.respondErr(w, err)
			return
		}
	} else {
		var c publicStoreCursor
		if !catalog.DecodeCursor(after, &c) || !catalog.ValidUUID(c.ID) || !catalog.ValidUUID(c.P) {
			s.respondErr(w, errValidation)
			return
		}
		cond += ` AND (ps.sort_order, ps.name, ps.id, NOT i.available, i.sort_order, i.name, i.id) > (` +
			arg(c.S) + `::int, ` + arg(c.N) + `::text, ` + arg(c.P) + `::uuid, ` + arg(c.U) + `::bool, ` +
			arg(c.O) + `::int, ` + arg(c.I) + `::text, ` + arg(c.ID) + `::uuid)`
	}
	items, err := s.queryItems(r, itemSelect+cond+`
		ORDER BY ps.sort_order, ps.name, ps.id, NOT i.available, i.sort_order, i.name, i.id
		LIMIT `+strconv.Itoa(limit+1), args...)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		last := items[limit-1]
		c := publicStoreCursor{N: last.SectionName, P: last.SectionID, U: !last.Available, I: last.Name, ID: last.ID}
		if err := s.pg.QueryRow(r.Context(), `
			SELECT ps.sort_order, i.sort_order, NOT i.available
			FROM menu_items i JOIN platform_sections ps ON ps.id = i.platform_section_id
			WHERE i.id = $1::uuid`, last.ID).Scan(&c.S, &c.O, &c.U); err != nil {
			s.respondErr(w, err)
			return
		}
		next = catalog.EncodeCursor(c)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "next_after": next, "total": total})
}

// storeOffersCond **صنفٌ عليه حسمٌ ساري** — بشرط السريان الواحد (`offers.LiveCond`).
const storeOffersCond = ` AND EXISTS (SELECT 1 FROM offers o WHERE o.menu_item_id = i.id AND o.kind = 'discount' AND ` +
	offers.LiveCond + `)`
