package server

// ══════════════════════════════════════════════════════════════════════
// **التحويلُ والسائقُ عند المتجر لا يُنزع منه الطلب** — خلل PD-4 (٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// في الإسناد المباشر يُكتب `offer_expires_at` مهلةَ صمتٍ للسائق (`assignDirectly`)، **ووصولُه
// إلى المتجر لا يمحوها** — يخرج بها من `assigned` فلا يراه كانسُ الصمت. **والتحويلُ عند
// `at_pickup` يُعيده إلى `assigned`** (إلى المتجر الجديد) والمهلةُ القديمةُ منقضية، **فيقرؤه
// الكانسُ إسناداً صامتاً وينزعه من سائقٍ واقفٍ عند المتجر.**

import (
	"context"
	"net/http"
	"testing"
)

// TestTransferAtPickupNotReclaimed **T0d — إسنادٌ مباشر، السائقُ عند المتجر، تحويلٌ، ثمّ
// كانسُ المهل ⇒ الطلبُ يبقى معه.**
func TestTransferAtPickupNotReclaimed(t *testing.T) {
	f := newDriverFixture(t, 2)
	ctx := context.Background()
	armRotation(t, f, 60)
	f.armDirect(t, true)
	for _, d := range f.drivers {
		f.onShift(t, d, true)
	}
	ops := f.armOps(t)

	orderID := f.dispatchingOrder(t, 48_000, 10_000)
	if err := f.srv.orders.OfferNext(ctx, orderID, nil); err != nil {
		t.Fatalf("الإسناد فشل: %v", err)
	}
	first, status := f.assignedTo(t, orderID)
	if first == "" || status != "assigned" {
		t.Fatalf("driver=%q status=%q — يُنتظر إسنادٌ مباشر", first, status)
	}
	if _, err := f.srv.orders.Transition(ctx, first, []string{"driver"},
		orderID, "at_pickup", ""); err != nil {
		t.Fatalf("الوصولُ إلى المتجر: %v", err)
	}
	// **وانتظر عند المتجر أطولَ من مهلة الصمت** — تُنضَج بالقاعدة لا بالانتظار.
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET offer_expires_at = now() - interval '1 second' WHERE id = $1`,
		orderID); err != nil {
		t.Fatalf("تعذّر إنضاجُ المهلة: %v", err)
	}

	var other string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent)
		SELECT 'المتجر البديل', category_id, 10 FROM merchants WHERE id = $1
		RETURNING id`, f.merchantID).Scan(&other); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, orderID)
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, other)
	})
	w := f.call(f.srv.handleTransferOrder, http.MethodPost,
		"/admin/orders/"+orderID+"/transfer", orderID, ops, []string{"ops"},
		`{"merchant_id":"`+other+`","note":"المتجرُ مغلق"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("التحويلُ ردّ %d: %s", w.Code, w.Body.String())
	}
	if d, st := f.assignedTo(t, orderID); d != first || st != "assigned" {
		t.Fatalf("بعد التحويل driver=%q status=%q — يُنتظر معه في الطريق للمتجر الجديد", d, st)
	}
	var expires *string
	if err := f.pool.QueryRow(ctx,
		`SELECT offer_expires_at::text FROM orders WHERE id = $1`, orderID).Scan(&expires); err != nil {
		t.Fatal(err)
	}
	if expires != nil {
		t.Errorf("offer_expires_at=%s بعد التحويل — يُنتظر NULL: مهلةُ الصمت انتهت بتحرّكه", *expires)
	}

	f.srv.orders.SweepExpiredOffers(ctx)

	still, status := f.assignedTo(t, orderID)
	if still != first || status != "assigned" {
		t.Fatalf("driver=%q status=%q بعد الكانس — **نُزع الطلبُ من سائقٍ واقفٍ عند المتجر** (يُنتظر %s)",
			still, status, first)
	}
}
