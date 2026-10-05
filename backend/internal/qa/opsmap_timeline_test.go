package qa

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **خطُّ زمن بطاقة الطلب — متى أُسند** (تحسيناتُ الخريطة ٢٠٢٦-١٠-٠٦)
// ══════════════════════════════════════════════════════════════════════
//
// **والإسنادُ لا عمودَ له في الطلب** — يُقرأ من سجلّ الحالات. **فطلبٌ لم
// يُسنَد لا وقتَ إسنادٍ له**، **والمُسنَدُ مرّتين يُقرأ آخرُ إسنادٍ له.**

type mapOrderTimes struct {
	ID         string     `json:"id"`
	AssignedAt *time.Time `json:"assigned_at"`
}

func orderTimesOf(t *testing.T, h *Harness, tok, id string) *mapOrderTimes {
	t.Helper()
	res := h.GET("/api/v1/admin/ops-map/orders", tok)
	if res.Code != http.StatusOK {
		t.Fatalf("طبقةُ الطلبات — %d · %s", res.Code, string(res.Body))
	}
	var env struct {
		Data struct {
			Orders []mapOrderTimes `json:"orders"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	for i := range env.Data.Orders {
		if env.Data.Orders[i].ID == id {
			return &env.Data.Orders[i]
		}
	}
	t.Fatalf("الطلبُ %s غائبٌ عن الطبقة", id)
	return nil
}

func TestOpsMap_OrderCarriesLastAssignedAt(t *testing.T) {
	h := New(t)
	item := h.NewItem(5000)
	cust := h.Customer()
	orderID := placeMapOrder(t, h, cust, item)
	tok := h.NewUser("admin").Token

	if got := orderTimesOf(t, h, tok, orderID); got.AssignedAt != nil {
		t.Fatalf("طلبٌ لم يُسنَد يحمل وقتَ إسناد %v", got.AssignedAt)
	}

	first := time.Now().Add(-20 * time.Minute).UTC().Truncate(time.Second)
	last := time.Now().Add(-5 * time.Minute).UTC().Truncate(time.Second)
	for _, at := range []time.Time{first, last} {
		if _, err := h.Pool.Exec(context.Background(),
			`INSERT INTO order_events (order_id, from_status, to_status, created_at)
			 VALUES ($1::uuid, 'dispatching', 'assigned', $2)`, orderID, at); err != nil {
			t.Fatal(err)
		}
	}
	got := orderTimesOf(t, h, tok, orderID)
	if got.AssignedAt == nil {
		t.Fatal("طلبٌ مُسنَدٌ بلا وقتِ إسناد — خطُّ الزمن يكذب")
	}
	if !got.AssignedAt.Equal(last) {
		t.Fatalf("وقتُ الإسناد %v والمنتظَرُ آخرُ إسنادٍ %v", got.AssignedAt, last)
	}
}
