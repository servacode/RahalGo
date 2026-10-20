package server

import (
	"context"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestDeleteMerchant **متجرٌ بلا طلبٍ يُحذف نهائيّاً ويُحرَّر رقمُ صاحبه** (٢٠٢٦-١٠-٢٠) —
// ومتجرٌ له طلبٌ يُردّ ولا يُمسّ.
func TestDeleteMerchant(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	admin := testdb.NewUser(t, f.pool, "admin")
	owner := testdb.NewUser(t, f.pool, "merchant")
	var cat, store string
	if err := f.pool.QueryRow(ctx, `SELECT category_id::text FROM merchants WHERE id = $1`, f.merchantID).Scan(&cat); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `INSERT INTO merchants (name, owner_user_id, status, category_id)
		VALUES ('متجر مكرّر', $1, 'active', $2) RETURNING id`, owner, cat).Scan(&store); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO merchant_hours (merchant_id, weekday, open_time, close_time, closed)
		VALUES ($1, 1, '09:00', '22:00', false)`, store); err != nil {
		t.Logf("ساعاتٌ للفحص لم تُدرج (شكلُ الجدول): %v", err)
	}
	del := func(id string) int {
		return f.call(f.srv.handleDeleteMerchant, http.MethodDelete, "/admin/merchants/"+id,
			id, admin, []string{"admin"}, "").Code
	}

	if code := del(store); code != http.StatusOK {
		t.Fatalf("متجرٌ بلا طلبٍ لم يُحذف: %d", code)
	}
	var left int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM merchants WHERE id = $1`, store).Scan(&left)
	if left != 0 {
		t.Fatal("المتجرُ باقٍ بعد الحذف")
	}
	var isMerchant bool
	_ = f.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role_code = 'merchant')`,
		owner).Scan(&isMerchant)
	if isMerchant {
		t.Fatal("صاحبُ المتجر المحذوف بقي بصفة متجر — فلا يصير موظّفاً عند غيره")
	}

	// **ومتجرٌ له طلبٌ يُردّ.**
	f.dispatchingOrder(t, 20_000, 5_000)
	if code := del(f.merchantID); code != http.StatusConflict {
		t.Fatalf("متجرٌ له طلبٌ حُذف أو ردّ غيرَ التعارض: %d", code)
	}
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM merchants WHERE id = $1`, f.merchantID).Scan(&left)
	if left != 1 {
		t.Fatal("متجرٌ له طلبٌ مُسّ")
	}
}
