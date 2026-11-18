package qa

// ══════════════════════════════════════════════════════════════════════
// **سعرُ طلب الواتساب من الكارت ومن الكابتن — بالمسار الحيّ** (قرارُ المالك ٢٠٢٦-١١-١٨)
// ══════════════════════════════════════════════════════════════════════
//
// «تعديل السعر على طلبية الواتس اب ايضا من الكرت ومن تطبيق الكابتن» — واختار المالك: من الكارت
// ثمنُ الطلب وأجرةُ التوصيل، **والكابتن يطلب الأجرة والإدارة توافق.**
// والأرقامُ من `newWAFx`: الأجرةُ ٥٠٠ وحصّةُ المنصّة ١٠٪، والمستلِمُ يدفع نقداً.

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func waPriceOrder(t *testing.T, h *Harness, name string) (waFx, string, *User) {
	t.Helper()
	w := newWAFx(t, h, name)
	c := waEngineCreate(t, h, w.md.admin.ID, w.typed())
	drv := h.driverOf(c.ID)
	return w, c.ID, drv
}

func feeReqOf(t *testing.T, r Res) map[string]any {
	t.Helper()
	if r.Code != http.StatusOK {
		t.Fatalf("طلبُ الأجرة: %s", r)
	}
	v, _ := r.JSON()["request"].(map[string]any)
	return v
}

// TestWAPRICE01_DriverRequestsOfficeApproves **الكابتن يطلب، والمكتب يعاين ويوافق** — والأجرةُ الجديدة
// يقبضها الكابتن، **وأجرُه وحصّةُ المنصّة عليها عند التسليم**، والدفترُ بلا خرق.
func TestWAPRICE01_DriverRequestsOfficeApproves(t *testing.T) {
	h := New(t)
	base := mdaBaseline(t, h)
	w, id, drv := waPriceOrder(t, h, "منطقةُ WAPRICE-01")
	url := "/api/v1/driver/orders/" + id + "/fee-request"

	// **ولا يغيّر الكابتنُ شيئاً بطلبه** — الأجرةُ كما هي حتى الموافقة.
	req := feeReqOf(t, h.POST(url, drv.Token, map[string]any{"fee": 800, "reason": "المحل نقل لآخر الحي"}))
	if req["status"] != "pending" || feeNum(req, "old_fee") != 500 || feeNum(req, "fee") != 800 {
		t.Fatalf("الطلب: %v", req)
	}
	var fee, cash int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT delivery_fee, cash_due FROM orders WHERE id = $1`, id).Scan(&fee, &cash)
	if fee != 500 || cash != 500 {
		t.Fatalf("**الطلبُ وحدَه غيّر المال**: أجرة %d نقد %d", fee, cash)
	}
	var told int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications WHERE entity_id = $1
		AND title LIKE '%طلب تعديل أجرة%'`, id).Scan(&told)
	if told == 0 {
		t.Fatalf("**المكتب ما انخبر**")
	}

	// **والكارت يراه** — من معلومات المكتب.
	info, err := h.Orders.BoardInfoOf(ctxBG(), []string{id})
	if err != nil || info[id] == nil || info[id].FeeRequest == nil || info[id].FeeRequest.Fee != 800 {
		t.Fatalf("**الكارت ما شاف الطلب**: %v %+v", err, info[id])
	}

	// **وغريبٌ لا يطلب على طلبٍ ليس معه.**
	other := h.NewUser("driver")
	if r := h.POST(url, other.Token, map[string]any{"fee": 900, "reason": "x"}); r.Code != http.StatusNotFound {
		t.Fatalf("كابتنٌ غريب: %s", r)
	}

	approve := "/api/v1/admin/orders/" + id + "/fee-request/approve"
	reqID, _ := req["id"].(string)
	res := feeEditResult(t, h.POST(approve, w.md.admin.Token, map[string]any{"request_id": reqID, "dry_run": true}))
	if feeNum(res, "new_fee") != 800 || feeNum(res, "driver_fee") != 720 || feeNum(res, "cash_due") != 800 {
		t.Fatalf("المعاينة: %v", res)
	}
	if p := feeReqOf(t, h.GET("/api/v1/admin/orders/"+id+"/fee-request", w.md.admin.Token)); p == nil || p["status"] != "pending" {
		t.Fatalf("**المعاينةُ قرّرت الطلب**: %v", p)
	}
	feeEditResult(t, h.POST(approve, w.md.admin.Token, map[string]any{"request_id": reqID}))
	if p := feeReqOf(t, h.GET(url, drv.Token)); p["status"] != "approved" {
		t.Fatalf("الكابتن يقرأ: %v", p)
	}
	if r := h.POST(approve, w.md.admin.Token, map[string]any{"request_id": reqID}); r.Code != http.StatusConflict {
		t.Fatalf("**موافقةٌ مرّتين**: %s", r)
	}

	deliverOrder(t, h, id, drv)
	if got := walletOf(t, h, drv.ID); got != 720 {
		t.Fatalf("**أجرُ الكابتن %d والمنتظَر 720**", got)
	}
	if got := waCashOf(t, h, id); got != 800 {
		t.Fatalf("**النقد %d والمنتظَر 800**", got)
	}
	if led := waLedgerOf(t, h, id); led["platform_profit"] != 80 || led["driver_earning"] != 720 {
		t.Fatalf("الدفتر: %v", led)
	}
	mdaNoNewViolations(t, h, base)
}

// TestWAPRICE02_RejectStaleAndRules **الرفضُ بسببٍ يصل الكابتن، والأجرةُ المتغيّرة تحت الطلب تجعله قديماً**،
// والطلبُ العاديّ (غيرُ الواتساب) لا يُطلب عليه.
func TestWAPRICE02_RejectStaleAndRules(t *testing.T) {
	h := New(t)
	w, id, drv := waPriceOrder(t, h, "منطقةُ WAPRICE-02")
	url := "/api/v1/driver/orders/" + id + "/fee-request"
	adm := w.md.admin.Token

	if r := h.POST(url, drv.Token, map[string]any{"fee": 500, "reason": "نفسها"}); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("نفس الأجرة: %s", r)
	}
	if r := h.POST(url, drv.Token, map[string]any{"fee": 700, "reason": " "}); r.Code != http.StatusBadRequest {
		t.Fatalf("بلا سبب: %s", r)
	}

	first := feeReqOf(t, h.POST(url, drv.Token, map[string]any{"fee": 700, "reason": "زحمة"}))
	second := feeReqOf(t, h.POST(url, drv.Token, map[string]any{"fee": 750, "reason": "زحمة كتير"}))
	var pending int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM order_fee_requests WHERE order_id = $1 AND status = 'pending'`, id).Scan(&pending)
	if pending != 1 {
		t.Fatalf("**طلبان معلّقان**: %d", pending)
	}
	reject := "/api/v1/admin/orders/" + id + "/fee-request/reject"
	if r := h.POST(reject, adm, map[string]any{"request_id": first["id"], "reason": "لا"}); r.Code != http.StatusConflict {
		t.Fatalf("**رفضُ طلبٍ حلّ محلَّه غيرُه**: %s", r)
	}
	if r := h.POST(reject, adm, map[string]any{"request_id": second["id"], "reason": "الأجرة مناسبة"}); r.Code != http.StatusOK {
		t.Fatalf("الرفض: %s", r)
	}
	if p := feeReqOf(t, h.GET(url, drv.Token)); p["status"] != "rejected" || p["reject_reason"] != "الأجرة مناسبة" {
		t.Fatalf("الكابتن يقرأ الرفض: %v", p)
	}
	var told int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications WHERE entity_id = $1 AND user_id = $2::uuid
		AND body LIKE '%الأجرة مناسبة%'`, id, drv.ID).Scan(&told)
	if told != 1 {
		t.Fatalf("**الكابتن ما انخبر بالرفض**: %d", told)
	}

	// **قديم**: المكتب عدّل الأجرة بعد الطلب.
	third := feeReqOf(t, h.POST(url, drv.Token, map[string]any{"fee": 650, "reason": "مشوار أبعد"}))
	feeEditResult(t, feeEditCall(h, adm, id, 600, false))
	r := h.POST("/api/v1/admin/orders/"+id+"/fee-request/approve", adm, map[string]any{"request_id": third["id"]})
	if r.Code != http.StatusConflict || !strings.Contains(fmt.Sprint(r), "fee_request_stale") {
		t.Fatalf("**القديم وافق**: %s", r)
	}
	if p := feeReqOf(t, h.GET(url, drv.Token)); p["status"] != "stale" {
		t.Fatalf("الحال: %v", p)
	}
	var fee int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT delivery_fee FROM orders WHERE id = $1`, id).Scan(&fee)
	if fee != 600 {
		t.Fatalf("الأجرة %d والمنتظَر 600 (تعديل المكتب)", fee)
	}

}

// TestWAPRICE04_NotForPlainDelivery **و«لدي توصيلة» العاديّة (مو واتساب) لا** — هلق للواتساب وحده متل ما طلب المالك.
func TestWAPRICE04_NotForPlainDelivery(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ WAPRICE-02b")
	fundWallet(t, h, m.admin, m.fx.M.Owner.ID, 2000)
	mid, mdrv := mdmCreate(t, h, m, "merchant", "", "waprice02")
	if r := h.POST("/api/v1/driver/orders/"+mid+"/fee-request", mdrv.Token, map[string]any{"fee": 900, "reason": "x"}); r.Code != http.StatusConflict {
		t.Fatalf("طلب غير واتساب: %s", r)
	}
}

// TestWAPRICE03_OfficeEditsGoods **المكتب يكتب ثمن الطلب من الكارت ويصحّحه** — قبل وصول الكابتن للمحل كمان،
// بسببٍ إجباريّ، والمبلغُ يُقبض من المستلِم، والسجلّ يقول «المكتب».
func TestWAPRICE03_OfficeEditsGoods(t *testing.T) {
	h := New(t)
	w, id, drv := waPriceOrder(t, h, "منطقةُ WAPRICE-03")
	url := "/api/v1/admin/whatsapp-orders/" + id + "/goods"
	adm := w.md.admin.Token

	if r := h.POST(url, adm, map[string]any{"amount": 3000}); r.Code != http.StatusBadRequest {
		t.Fatalf("بلا سبب: %s", r)
	}
	// **وغيرُ صاحب القدرة لا** — كابتن.
	if r := h.POST(url, drv.Token, map[string]any{"amount": 3000, "reason": "x"}); r.Code < 400 {
		t.Fatalf("**الكابتن كتب من باب المكتب**: %s", r)
	}
	if r := h.POST(url, adm, map[string]any{"amount": 3000, "reason": "الزبون أكّد بالواتساب"}); r.Code != http.StatusOK {
		t.Fatalf("الكتابة: %s", r)
	}
	var goods, collect *int64
	var status, note string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT fetch_amount, collect_amount, COALESCE(fetch_amount_status, '') FROM orders WHERE id = $1`, id).
		Scan(&goods, &collect, &status)
	if goods == nil || *goods != 3000 || collect == nil || *collect != 3000 || status != "approved" {
		t.Fatalf("الثمن: %v %v %s", goods, collect, status)
	}
	if r := h.POST(url, adm, map[string]any{"amount": 3500, "reason": "صنف زيادة"}); r.Code != http.StatusOK {
		t.Fatalf("التصحيح: %s", r)
	}
	_ = h.Pool.QueryRow(ctxBG(), `SELECT note FROM order_events WHERE order_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1`, id).Scan(&note)
	if note == "" || !strings.Contains(note, "المكتب عدّل ثمن الطلب") || !strings.Contains(note, "صنف زيادة") {
		t.Fatalf("السجلّ: %q", note)
	}

	// **والكابتن بيضل يقدر يصحّح** من المحل — بابُه كما هو.
	waToDoor(t, h, id, drv)
	if r := h.POST("/api/v1/driver/orders/"+id+"/fetch-amount", drv.Token, map[string]any{"amount": 3400}); r.Code != http.StatusOK {
		t.Fatalf("تصحيح الكابتن: %s", r)
	}
	h.authorizeProofExempt(id)
	if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": "delivered"}); r.Code >= 400 {
		t.Fatalf("التسليم: %s", r)
	}
	if r := h.POST(url, adm, map[string]any{"amount": 1, "reason": "بعد التسليم"}); r.Code != http.StatusConflict {
		t.Fatalf("**بعد التسليم انكتب**: %s", r)
	}
}
