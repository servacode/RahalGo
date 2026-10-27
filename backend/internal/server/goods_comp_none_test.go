package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestGoodsCompNone_LeavesQueueWithoutMoney **«ما في تعويض»** (قرارُ المالك ٢٠٢٦-١٠-٢٧):
// يُردّ والطلبُ له تعويضٌ معلَّق، ثمّ يُخرج الطلبَ من الطابور نهائياً بلا قرش،
// ولا تعويضَ ولا «ما في تعويض» ثانيةً بعده.
func TestGoodsCompNone_LeavesQueueWithoutMoney(t *testing.T) {
	f := newDriverFixture(t, 1)
	pinOldScaleMoney(t, f.pool, f.srv)
	treasury := compTreasury(t, f)
	ctx := context.Background()
	owner := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET owner_user_id = $2 WHERE id = $1`, f.merchantID, owner); err != nil {
		t.Fatal(err)
	}
	orderID := f.problemOrderAt(t, "at_dropoff", f.drivers[0])
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'failed', closed_at = now(),
		goods_settled_to = 'merchant' WHERE id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO order_items (order_id, name, unit_price, qty, merchant_id, merchant_price)
		VALUES ($1, 'صنف', 12000, 1, $2, 10000)`, orderID, f.merchantID); err != nil {
		t.Fatal(err)
	}
	fin := testdb.NewUser(t, f.pool, "finance")
	none := func(body string) (int, string) {
		w := f.call(f.srv.handleGoodsNoCompensation, http.MethodPost,
			"/admin/orders/"+orderID+"/goods/compensation/none", orderID, fin, []string{"finance"}, body)
		return w.Code, w.Body.String()
	}
	inQueue := func() bool {
		w := f.call(f.srv.handleGoodsCompensations, http.MethodGet,
			"/admin/losses/goods-compensations", "", fin, []string{"finance"}, "")
		var out struct {
			Data struct {
				Orders []goodsCompRow `json:"orders"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		for _, r := range out.Data.Orders {
			if r.OrderID == orderID {
				return true
			}
		}
		return false
	}
	if !inQueue() {
		t.Fatal("الطلبُ ليس في الطابور قبل الفعل")
	}

	// **طلبُ تعويضٍ معلَّق يُردّ أوّلاً** — فلا يبقى معلَّقٌ لطلبٍ خرج من الطابور.
	w := f.call(f.srv.handleGoodsCompensation, http.MethodPost,
		"/admin/orders/"+orderID+"/goods/compensation", orderID, fin, []string{"finance"}, `{"amount":8000}`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("الاقتراح %d: %s", w.Code, w.Body.String())
	}
	if code, body := none(`{}`); code != http.StatusConflict || !strings.Contains(body, "goods_comp_pending") {
		t.Fatalf("«ما في تعويض» مع تعويضٍ معلَّق ردّ %d: %s", code, body)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE driver_compensation_requests SET status = 'rejected'
		WHERE order_id = $1 AND kind = 'merchant_goods'`, orderID); err != nil {
		t.Fatal(err)
	}

	ownerBefore, treasuryBefore := f.bal(t, owner), f.bal(t, treasury)
	if code, body := none(`{"reason":"رجعت سليمة"}`); code != http.StatusOK {
		t.Fatalf("«ما في تعويض» ردّ %d: %s", code, body)
	}
	if f.bal(t, owner) != ownerBefore || f.bal(t, treasury) != treasuryBefore {
		t.Fatal("**«ما في تعويض» حرّك مالاً**")
	}
	var reason string
	if err := f.pool.QueryRow(ctx, `SELECT COALESCE(goods_comp_none_reason, '') FROM orders
		WHERE id = $1 AND goods_comp_none_at IS NOT NULL AND goods_comp_none_by = $2`, orderID, fin).
		Scan(&reason); err != nil || reason != "رجعت سليمة" {
		t.Fatalf("الوسمُ لم يُحفظ: %v · %q", err, reason)
	}
	if inQueue() {
		t.Fatal("**الطلبُ باقٍ في الطابور بعد «ما في تعويض»**")
	}

	// **نهائيّ**: لا مرّةً ثانية، ولا تعويضَ بعده.
	if code, body := none(`{}`); code != http.StatusConflict || !strings.Contains(body, "goods_comp_none") {
		t.Fatalf("«ما في تعويض» ثانيةً ردّ %d: %s", code, body)
	}
	w = f.call(f.srv.handleGoodsCompensation, http.MethodPost,
		"/admin/orders/"+orderID+"/goods/compensation", orderID, fin, []string{"finance"}, `{"amount":8000}`)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "goods_comp_none") {
		t.Fatalf("تعويضٌ بعد «ما في تعويض» ردّ %d: %s", w.Code, w.Body.String())
	}
}
