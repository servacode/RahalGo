package server

// **نبضُ المتجر** — أرقامُ رأس ملفّه (٣٦٠) تساوي ما في القاعدة: طلباتُ اليوم والجارية،
// والأصنافُ غيرُ المتوفّرة، و«مفتوح الآن» بالدوام. وبلا دوامٍ مضبوطٍ: مفتوحٌ دائماً (كما يقرّر الإنشاء).

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

func TestMerchantPulse_NumbersMatchTheDatabase(t *testing.T) {
	srv := overviewServer(t)
	ctx := context.Background()
	orderID := seedMonitorOrder(t, srv, orders.StPending, 5, -1)

	var merchantID, menuSectionID, platformSectionID string
	if err := srv.pg.QueryRow(ctx, `SELECT merchant_id::text FROM orders WHERE id = $1`, orderID).Scan(&merchantID); err != nil {
		t.Fatal(err)
	}
	if err := srv.pg.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'قائمةُ النبض') RETURNING id::text`, merchantID).Scan(&menuSectionID); err != nil {
		t.Fatal(err)
	}
	if err := srv.pg.QueryRow(ctx, `
		INSERT INTO platform_sections (name, sort_order) VALUES ('قسمُ النبض', 901) RETURNING id::text`).Scan(&platformSectionID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = srv.pg.Exec(bg, `DELETE FROM menu_items WHERE merchant_id = $1`, merchantID)
		_, _ = srv.pg.Exec(bg, `DELETE FROM menu_sections WHERE id = $1`, menuSectionID)
		_, _ = srv.pg.Exec(bg, `DELETE FROM platform_sections WHERE id = $1`, platformSectionID)
	})
	for i, available := range []bool{true, true, false} {
		if _, err := srv.pg.Exec(ctx, `
			INSERT INTO menu_items (merchant_id, section_id, platform_section_id, name, price, merchant_price, available)
			VALUES ($1, $2, $3, $4, 1000, 1000, $5)`, merchantID, menuSectionID, platformSectionID, "صنف"+string(rune('أ'+i)), available); err != nil {
			t.Fatal(err)
		}
	}

	w := callAs(srv.handleAdminMerchantPulse, "GET", "/admin/merchants/x/pulse", "", merchantID, []string{"merchants.read"})
	var p merchantPulse
	if err := json.Unmarshal(dataOf(t, w), &p); err != nil {
		t.Fatalf("الردُّ لا يُقرأ: %v", err)
	}
	if p.OrdersToday != 1 || p.OrdersOpen != 1 {
		t.Fatalf("طلباتُ اليوم %d والجارية %d — المتوقّع 1 و1", p.OrdersToday, p.OrdersOpen)
	}
	if p.ItemsTotal != 3 || p.ItemsUnavailable != 1 {
		t.Fatalf("الأصناف %d وغيرُ المتوفّر %d — المتوقّع 3 و1", p.ItemsTotal, p.ItemsUnavailable)
	}
	if !p.OpenNow {
		t.Fatal("متجرٌ بلا ساعات دوامٍ مفتوحٌ دائماً")
	}
}

func TestMerchantPulse_UnknownStoreIs404(t *testing.T) {
	srv := overviewServer(t)
	w := callAs(srv.handleAdminMerchantPulse, "GET", "/x", "", "not-a-uuid", nil)
	if w.Code != 404 {
		t.Fatalf("معرّفٌ فاسدٌ يردّ %d لا 404", w.Code)
	}
}
