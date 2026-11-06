package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestNoPhotoItems_OwnOnlyOrderPaging: lists only the own store's items without a photo,
// ordered by market section then name, keyset paged, leaving the list as photos are saved.
func TestNoPhotoItems_OwnOnlyOrderPaging(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	owner := testdb.NewUser(t, f.pool, "merchant")
	other := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET owner_user_id = $2 WHERE id = $1`, f.merchantID, owner); err != nil {
		t.Fatal(err)
	}
	var secA, secB, mediaID string
	if err := f.pool.QueryRow(ctx, `INSERT INTO platform_sections (name, sort_order) VALUES ('aa nophoto sec', 952) RETURNING id`).Scan(&secA); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `INSERT INTO platform_sections (name, sort_order) VALUES ('bb nophoto sec', 953) RETURNING id`).Scan(&secB); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `INSERT INTO media (kind, path, thumb_path, width, height, bytes, created_by)
		VALUES ('menu_item', 'x.jpg', 'x_t.jpg', 10, 10, 100, $1) RETURNING id`, owner).Scan(&mediaID); err != nil {
		t.Fatal(err)
	}
	add := func(merchant, sec, name string, media *string) string {
		var id string
		if err := f.pool.QueryRow(ctx, `
			INSERT INTO menu_items (merchant_id, platform_section_id, name, price, merchant_price, available, image_media_id)
			VALUES ($1, $2, $3, 100, 100, false, $4::uuid) RETURNING id`, merchant, sec, name, media).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	b2 := add(f.merchantID, secB, "b-two", nil)
	a2 := add(f.merchantID, secA, "a-two", nil)
	a1 := add(f.merchantID, secA, "a-one", nil)
	add(f.merchantID, secA, "has photo", &mediaID)
	// another store's item without photo must never show
	var foreign string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, phone, address_text)
		SELECT 'foreign nophoto', category_id, phone, address_text FROM merchants WHERE id = $1 RETURNING id`, f.merchantID).Scan(&foreign); err != nil {
		t.Skipf("cannot clone store: %v", err)
	}
	add(foreign, secA, "foreign item", nil)
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `DELETE FROM menu_items WHERE merchant_id IN ($1, $2)`, f.merchantID, foreign)
		_, _ = f.pool.Exec(c, `DELETE FROM merchants WHERE id = $1`, foreign)
		_, _ = f.pool.Exec(c, `DELETE FROM platform_sections WHERE id IN ($1, $2)`, secA, secB)
	})

	type page struct {
		Data struct {
			Items []struct {
				ID, Name string
			} `json:"items"`
			Total     int    `json:"total"`
			NextAfter string `json:"next_after"`
		} `json:"data"`
	}
	get := func(user, query string) (int, page) {
		w := f.call(f.srv.handleMerchantNoPhotoItems, http.MethodGet, "/merchant/stores/"+f.merchantID+"/menu/no-photo"+query,
			f.merchantID, user, []string{"merchant"}, "")
		var p page
		_ = json.Unmarshal(w.Body.Bytes(), &p)
		return w.Code, p
	}
	if code, _ := get(other, ""); code != http.StatusForbidden {
		t.Fatalf("other owner: %d", code)
	}
	code, p := get(owner, "")
	if code != http.StatusOK || p.Data.Total != 3 || len(p.Data.Items) != 3 {
		t.Fatalf("all: %d total=%d items=%d", code, p.Data.Total, len(p.Data.Items))
	}
	if p.Data.Items[0].ID != a1 || p.Data.Items[1].ID != a2 || p.Data.Items[2].ID != b2 {
		t.Fatalf("order: %+v", p.Data.Items)
	}
	// paging by cursor
	_, p1 := get(owner, "?limit=2")
	if len(p1.Data.Items) != 2 || p1.Data.NextAfter != a2 {
		t.Fatalf("page1: %+v", p1.Data)
	}
	_, p2 := get(owner, "?limit=2&after="+p1.Data.NextAfter)
	if len(p2.Data.Items) != 1 || p2.Data.Items[0].ID != b2 || p2.Data.NextAfter != "" {
		t.Fatalf("page2: %+v", p2.Data)
	}
	// saving a photo removes the item from the list
	if _, err := f.pool.Exec(ctx, `UPDATE menu_items SET image_media_id = $2 WHERE id = $1`, a1, mediaID); err != nil {
		t.Fatal(err)
	}
	_, p3 := get(owner, "")
	if p3.Data.Total != 2 || len(p3.Data.Items) != 2 || p3.Data.Items[0].ID != a2 {
		t.Fatalf("after photo: %+v", p3.Data)
	}
}
