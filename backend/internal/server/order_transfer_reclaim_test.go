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

// atPickupRipeSilence **إسنادٌ مباشر، السائقُ عند المتجر، ومهلةُ الصمت منقضية** — الحالُ التي
// يقع فيها خللُ PD-4 بكلّ مسارٍ يُعيد `at_pickup` إلى `assigned`. يردّ الطلبَ وسائقَه وموظّفَ
// عمليّات.
func atPickupRipeSilence(t *testing.T) (f *driverFixture, orderID, driverID, ops string) {
	t.Helper()
	f = newDriverFixture(t, 2)
	ctx := context.Background()
	armRotation(t, f, 60)
	f.armDirect(t, true)
	for _, d := range f.drivers {
		f.onShift(t, d, true)
	}
	ops = f.armOps(t)

	orderID = f.dispatchingOrder(t, 48_000, 10_000)
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
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, orderID)
	})
	return f, orderID, first, ops
}

// assertNotReclaimed **بعد كانس المهل: الطلبُ مع سائقه في `assigned`، ولا مهلةَ صمتٍ عليه.**
func assertNotReclaimed(t *testing.T, f *driverFixture, orderID, first string) {
	t.Helper()
	var expires *string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT offer_expires_at::text FROM orders WHERE id = $1`, orderID).Scan(&expires); err != nil {
		t.Fatal(err)
	}
	if expires != nil {
		t.Errorf("offer_expires_at=%s بعد العودة إلى assigned — يُنتظر NULL: مهلةُ الصمت انتهت بتحرّكه", *expires)
	}

	f.srv.orders.SweepExpiredOffers(context.Background())

	still, status := f.assignedTo(t, orderID)
	if still != first || status != "assigned" {
		t.Fatalf("driver=%q status=%q بعد الكانس — **نُزع الطلبُ من سائقٍ واقفٍ عند المتجر** (يُنتظر %s)",
			still, status, first)
	}
}

// TestTransferAtPickupNotReclaimed **T0d — إسنادٌ مباشر، السائقُ عند المتجر، تحويلٌ، ثمّ
// كانسُ المهل ⇒ الطلبُ يبقى معه.**
func TestTransferAtPickupNotReclaimed(t *testing.T) {
	f, orderID, first, ops := atPickupRipeSilence(t)
	ctx := context.Background()

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
	assertNotReclaimed(t, f, orderID, first)
}

// TestExternalStoreAtPickupNotReclaimed **شقيقُ T0d — «من متجرٍ غير تابع» والسائقُ عند المتجر.**
//
// القلبُ (`ConvertToExternalStore`) يُعيد `at_pickup` إلى `assigned` بقيدٍ مباشرٍ كالتحويل،
// **وكانسُ الصمت لا يفرّق بين الأنواع** — فيأخذ طلبَ المشتريات من سائقٍ واقفٍ عند المتجر ويُعيده
// إلى الطابور. ⇒ يبقى معه.
func TestExternalStoreAtPickupNotReclaimed(t *testing.T) {
	f, orderID, first, ops := atPickupRipeSilence(t)
	_, ch, err := f.srv.orders.ConvertToExternalStore(context.Background(), ops, orderID,
		"المتجرُ مغلق والزبونُ يريده من غيره", nil)
	if err != nil {
		t.Fatalf("القلب: %v", err)
	}
	if ch.FromStatus != "at_pickup" || ch.ToStatus != "assigned" {
		t.Fatalf("القلبُ من %q إلى %q — يُنتظر at_pickup ← assigned", ch.FromStatus, ch.ToStatus)
	}
	if d, st := f.assignedTo(t, orderID); d != first || st != "assigned" {
		t.Fatalf("بعد القلب driver=%q status=%q — يُنتظر معه في الطريق", d, st)
	}
	assertNotReclaimed(t, f, orderID, first)
}
