package catalog

// ══════════════════════════════════════════════════════════════════════
// Paged menu for the store owner (owner decision 2026-11-07: «ما يصير يكون بطيء»)
// ══════════════════════════════════════════════════════════════════════
//
// Measured on staging 2026-11-07: a mall with 5,311 items made GET .../menu a 3 MB answer
// (~300 KB gzip). On a weak link the items tab took 45 s once and failed another time, and
// malls may reach 15,000 items. So the screens now read a light summary (sections with their
// counts) and then small keyset pages of items, filtered by section, search and availability
// on the server.
//
// The full menu (`GetMenu`) stays as it was: old app versions, the admin and the rep keep it.
// Both read an item with the same columns and the same pricing (`menuItemCols`, `scanMenuItem`).

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// Page size bounds for the paged menu readers.
const (
	MenuPageDefault = 60
	MenuPageMax     = 200
)

// MenuSummarySection a market section that holds items of this store, with its counts.
type MenuSummarySection struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	ImageURL      *string `json:"image_url"`
	ImageThumbURL *string `json:"image_thumb_url"`
	SortOrder     int     `json:"sort_order"`
	Count         int     `json:"count"`
	Unavailable   int     `json:"unavailable"`
}

// MenuSummary what the items screen needs before the first page: the section chips with
// their counts, the "unavailable" count and the quick-photo count.
type MenuSummary struct {
	// Total the items that appear in the menu (those placed in a market section, as `GetMenu`).
	Total       int `json:"total"`
	Unavailable int `json:"unavailable"`
	// NoPhoto every item of the store without a photo — the same count as GET .../menu/no-photo.
	NoPhoto  int                  `json:"no_photo"`
	Sections []MenuSummarySection `json:"sections"`
	// WaItemsStore the store's own WhatsApp add-items flag (0531). Not sent as is: the handler
	// combines it with the global switch into WaItemsOn.
	WaItemsStore bool `json:"-"`
	// WaItemsOn true only when the global switch (merchants.wa_items_enabled) AND the store's
	// own flag are both on — the items screen shows the "add items by WhatsApp" hint from it
	// (owner 2026-11-11: the hint must not show while the bot is off).
	WaItemsOn bool `json:"wa_items_on"`
}

// GetMenuSummary the light head of the paged menu. Sections are in market order
// (sort_order, name, id) — the same order the paged items follow.
func (s *Service) GetMenuSummary(ctx context.Context, merchantID string) (*MenuSummary, error) {
	m, err := s.merchantByID(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	out := &MenuSummary{Sections: []MenuSummarySection{}, WaItemsStore: m.WaItemsEnabled}
	rows, err := s.db.Query(ctx, `
		SELECT ps.id, ps.name, sm.path, sm.thumb_path, ps.sort_order,
		       count(*), count(*) FILTER (WHERE NOT i.available)
		FROM menu_items i
		JOIN platform_sections ps ON ps.id = i.platform_section_id
		LEFT JOIN media sm ON sm.id = ps.image_media_id
		WHERE i.merchant_id = $1
		GROUP BY ps.id, sm.path, sm.thumb_path
		ORDER BY ps.sort_order, ps.name, ps.id`, merchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var x MenuSummarySection
		if err := rows.Scan(&x.ID, &x.Name, &x.ImageURL, &x.ImageThumbURL, &x.SortOrder, &x.Count, &x.Unavailable); err != nil {
			return nil, err
		}
		x.ImageURL = media.URLForPtr(x.ImageURL)
		x.ImageThumbURL = media.URLForPtr(x.ImageThumbURL)
		out.Total += x.Count
		out.Unavailable += x.Unavailable
		out.Sections = append(out.Sections, x)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM menu_items WHERE merchant_id = $1 AND image_media_id IS NULL`,
		merchantID).Scan(&out.NoPhoto); err != nil {
		return nil, err
	}
	return out, nil
}

// MenuQuery the filters of one page. All are optional.
type MenuQuery struct {
	// SectionID one market section only.
	SectionID string
	// Q a search on the item name (contains, case-insensitive) or its exact Ameen code / barcode.
	Q string
	// Filter "unavailable", "available" (the offer item picker) or "no_photo"; empty is every item.
	Filter string
	// ItemID one item only — for the editor and after a duplicate.
	ItemID string
	// After the opaque cursor returned as `next_after` by the previous page.
	After string
	Limit int
}

// MenuPage one keyset page of items (with their modifiers).
type MenuPage struct {
	Items []MenuItem `json:"items"`
	// NextAfter the cursor of the next page; empty when this page is the last.
	NextAfter string `json:"next_after"`
	// Total the items matching the filters — counted on the first page only (no cursor),
	// and -1 on later pages: the screen already has it.
	Total int `json:"total"`
}

// menuCursor the sort key of the last item shown: section order, section name, section id,
// item name, item id. Carrying the key (not just the id) keeps paging correct when that item
// is deleted or moved between two pages.
type menuCursor struct {
	S  int    `json:"s"`
	N  string `json:"n"`
	P  string `json:"p"`
	I  string `json:"i"`
	ID string `json:"id"`
}

// EncodeCursor an opaque page cursor (base64url of the JSON sort key).
func EncodeCursor(v any) string {
	b, _ := json.Marshal(v)
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeCursor reads a cursor made by EncodeCursor; false when malformed.
func DecodeCursor(s string, v any) bool {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return false
	}
	return json.Unmarshal(b, v) == nil
}

// ValidUUID reports whether s has the textual form of a UUID (so it can be cast safely).
func ValidUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
				return false
			}
		}
	}
	return true
}

// ErrBadMenuQuery a malformed cursor or filter.
var ErrBadMenuQuery = ErrNameRequired

// LikePattern a contains-pattern for ILIKE with % _ and \ escaped.
func LikePattern(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

// NormalizeQuery trims the search text and caps it at 60 characters (as the search fields do).
func NormalizeQuery(q string) string {
	q = strings.TrimSpace(q)
	if r := []rune(q); len(r) > 60 {
		q = string(r[:60])
	}
	return q
}

// ClampLimit the page size within [1, MenuPageMax], defaulting to MenuPageDefault.
func ClampLimit(n int) int {
	if n <= 0 {
		return MenuPageDefault
	}
	if n > MenuPageMax {
		return MenuPageMax
	}
	return n
}

// ListMenuItems one page of the store's items in the order of the summary: market section
// (sort_order, name, id), then item name, then id. Items outside a market section are left
// out, as in `GetMenu`. Prices are those of `GetMenu` (sale price in `price`); the merchant
// handler swaps in the merchant price as it does for the full menu.
func (s *Service) ListMenuItems(ctx context.Context, merchantID string, q MenuQuery) (*MenuPage, error) {
	if _, err := s.merchantByID(ctx, merchantID); err != nil {
		return nil, err
	}
	limit := ClampLimit(q.Limit)
	args := []any{merchantID}
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	where := ` WHERE i.merchant_id = $1`
	if q.SectionID != "" {
		if !ValidUUID(q.SectionID) {
			return nil, ErrBadMenuQuery
		}
		where += ` AND i.platform_section_id = ` + arg(q.SectionID) + `::uuid`
	}
	if q.ItemID != "" {
		if !ValidUUID(q.ItemID) {
			return nil, ErrBadMenuQuery
		}
		where += ` AND i.id = ` + arg(q.ItemID) + `::uuid`
	}
	switch q.Filter {
	case "":
	case "unavailable":
		where += ` AND NOT i.available`
	case "available":
		where += ` AND i.available`
	case "no_photo":
		where += ` AND i.image_media_id IS NULL`
	default:
		return nil, ErrBadMenuQuery
	}
	if text := NormalizeQuery(q.Q); text != "" {
		p := arg(LikePattern(text))
		e := arg(text)
		where += ` AND (i.name ILIKE ` + p + ` ESCAPE '\' OR i.external_code = ` + e + ` OR i.barcode = ` + e + `)`
	}
	const from = `
		FROM menu_items i
		JOIN platform_sections ps ON ps.id = i.platform_section_id
		LEFT JOIN media im ON im.id = i.image_media_id`

	out := &MenuPage{Items: []MenuItem{}, Total: -1}
	if q.After == "" {
		if err := s.db.QueryRow(ctx, `SELECT count(*)`+from+where, args...).Scan(&out.Total); err != nil {
			return nil, err
		}
	}
	pageWhere := where
	if q.After != "" {
		var c menuCursor
		if !DecodeCursor(q.After, &c) || !ValidUUID(c.ID) || !ValidUUID(c.P) {
			return nil, ErrBadMenuQuery
		}
		pageWhere += ` AND (ps.sort_order, ps.name, ps.id, i.name, i.id) > (` +
			arg(c.S) + `::int, ` + arg(c.N) + `::text, ` + arg(c.P) + `::uuid, ` + arg(c.I) + `::text, ` + arg(c.ID) + `::uuid)`
	}
	rule := pricing.RuleFrom(ctx, s.settings)
	var open bool
	var opensAt *time.Time
	_ = s.db.QueryRow(ctx, `
		SELECT `+orders.OpenNowSQL+`, `+orders.NextOpenSQL+`
		FROM merchants m WHERE m.id = $1`, merchantID).Scan(&open, &opensAt)

	rows, err := s.db.Query(ctx, `
		SELECT `+menuItemCols+`, ps.sort_order`+from+pageWhere+`
		ORDER BY ps.sort_order, ps.name, ps.id, i.name, i.id
		LIMIT `+strconv.Itoa(limit+1), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lastSort int
	sorts := []int{}
	for rows.Next() {
		it, err := scanMenuItemWith(rows, rule, open, opensAt, &lastSort)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, it)
		sorts = append(sorts, lastSort)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if len(out.Items) > limit {
		out.Items = out.Items[:limit]
		last := out.Items[limit-1]
		ps := ""
		if last.PlatformSectionID != nil {
			ps = *last.PlatformSectionID
		}
		out.NextAfter = EncodeCursor(menuCursor{S: sorts[limit-1], N: last.PlatformSectionName, P: ps, I: last.Name, ID: last.ID})
	}
	if err := s.attachModifiers(ctx, out.Items); err != nil {
		return nil, err
	}
	return out, nil
}

// scanMenuItemWith reads `menuItemCols` followed by one extra column (the section sort order).
func scanMenuItemWith(rows interface {
	Scan(dest ...any) error
}, rule pricing.MarginRule, open bool, opensAt *time.Time, sortOrder *int) (MenuItem, error) {
	var it MenuItem
	var sectionMargin, usdCents *int64
	if err := rows.Scan(&it.ID, &it.SectionID, &it.Name, &it.Description,
		&it.MerchantPrice, &it.MarginOverride, &sectionMargin,
		&it.PlatformSectionID, &it.PlatformSectionName,
		&it.ImageURL, &it.ImageThumbURL, &it.Available, &it.SortOrder, &usdCents,
		&it.StoreDrinks, &it.ExternalCode, &it.Barcode, &it.CreatedVia, sortOrder); err != nil {
		return it, err
	}
	it.ImageURL = media.URLForPtr(it.ImageURL)
	it.ImageThumbURL = media.URLForPtr(it.ImageThumbURL)
	dressMenuItem(&it, rule, open, opensAt, sectionMargin, usdCents)
	return it, nil
}

// attachModifiers loads the modifier groups and options of the given items — the same rows
// and order `GetMenu` reads (copied store drinks excluded: they are managed from their list).
func (s *Service) attachModifiers(ctx context.Context, items []MenuItem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, len(items))
	at := map[string]int{}
	for i := range items {
		ids[i] = items[i].ID
		at[items[i].ID] = i
	}
	type loc struct{ item, group int }
	groupAt := map[string]loc{}
	rows, err := s.db.Query(ctx, `
		SELECT g.id, g.item_id, g.name, g.min_select, g.max_select, g.sort_order, g.price_mode, g.base_grams
		FROM modifier_groups g
		WHERE g.item_id = ANY($1::uuid[]) AND NOT g.store_drinks
		ORDER BY g.sort_order`, ids)
	if err != nil {
		return err
	}
	for rows.Next() {
		var g ModifierGroup
		var itemID string
		if err := rows.Scan(&g.ID, &itemID, &g.Name, &g.MinSelect, &g.MaxSelect, &g.SortOrder, &g.PriceMode, &g.BaseGrams); err != nil {
			rows.Close()
			return err
		}
		g.Options = []ModifierOption{}
		i, ok := at[itemID]
		if !ok {
			continue
		}
		groupAt[g.ID] = loc{i, len(items[i].Modifiers)}
		items[i].Modifiers = append(items[i].Modifiers, g)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(groupAt) == 0 {
		return nil
	}
	gids := make([]string, 0, len(groupAt))
	for id := range groupAt {
		gids = append(gids, id)
	}
	rows, err = s.db.Query(ctx, `
		SELECT o.id, o.group_id, o.name, o.price_delta, o.available, o.sort_order, o.price_delta_usd_cents, o.grams
		FROM modifier_options o
		WHERE o.group_id = ANY($1::uuid[])
		ORDER BY o.sort_order`, gids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var o ModifierOption
		var groupID string
		var optUSD *int64
		if err := rows.Scan(&o.ID, &groupID, &o.Name, &o.PriceDelta, &o.Available, &o.SortOrder, &optUSD, &o.Grams); err != nil {
			return err
		}
		if optUSD != nil {
			d := float64(*optUSD) / 100
			o.PriceDeltaUSD = &d
		}
		l, ok := groupAt[groupID]
		if !ok {
			continue
		}
		g := &items[l.item].Modifiers[l.group]
		g.Options = append(g.Options, o)
	}
	return rows.Err()
}
