package server

// **نصوصُ الزبون لها حدود — والزائدُ يُرفض بـ٤٠٠ ولا يُخزَّن.**
//
// (فحصُ القبول ٢٠٢٦-١٠-٠٣: `POST /orders` قبل عنواناً من عشرين ألف حرف
//  **وخزّنه**.)

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/comms"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/support"
)

// TestCheckTextLimits_CountsLettersNotBytes **الحدُّ بالحروف** — والعربيُّ بايتان.
func TestCheckTextLimits_CountsLettersNotBytes(t *testing.T) {
	arabic := strings.Repeat("ع", maxAddressText)
	if err := checkTextLimits(textField{"address_text", arabic, maxAddressText}); err != nil {
		t.Fatalf("عنوانٌ من %d حرفاً عربيّاً رُفض — **عُدّ بالبايت لا بالحرف**", maxAddressText)
	}
	if err := checkTextLimits(textField{"address_text", arabic + "ع", maxAddressText}); err == nil {
		t.Fatal("حرفٌ فوق الحدّ قُبل")
	}
}

// TestAddressLine_FitsOrderLimit **العنوانُ المحفوظُ بأطول أجزائه يُقبل في الطلب.**
//
// **والسطرُ يُركَّب من الأجزاء ثمّ يُرسَل `address_text`** — فإن زاد على حدّ
// الطلب رُفض طلبُ زبونٍ لم يكتب إلّا ما قُبل منه.
func TestAddressLine_FitsOrderLimit(t *testing.T) {
	line := addressLine(strings.Repeat("ع", maxAddressPart), strings.Repeat("ع", maxAddressPart),
		strings.Repeat("9", maxAddressFloor))
	if err := checkTextLimits(textField{"address_text", line, maxAddressText}); err != nil {
		t.Fatalf("أطولُ عنوانٍ محفوظٍ (%d حرفاً) يُرفض في الطلب", len([]rune(line)))
	}
}

// TestCustomerText_TooLongIsRejected **كلُّ بابٍ يكتب فيه الزبونُ نصّاً حرّاً.**
func TestCustomerText_TooLongIsRejected(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	f.srv.comms = comms.New(f.pool)
	f.srv.support = support.NewService(f.pool, f.srv.identity, f.srv.wallet)
	f.srv.notify = notifications.New(f.pool, f.srv.hub, f.srv.logger)
	ctx := context.Background()

	// **طلبٌ بيد سائقٍ** — للحديث، **وصاحبُه زبونٌ حقيقيّ.**
	orderID := f.problemOrderAt(t, "on_the_way", d)
	var customer string
	if err := f.pool.QueryRow(ctx, `SELECT customer_id::text FROM orders WHERE id = $1`,
		orderID).Scan(&customer); err != nil {
		t.Fatal(err)
	}
	// **وتذكرةٌ مفتوحةٌ له** — للردّ عليها.
	var ticketID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO tickets (customer_id, created_by, opened_by_customer, subject, status)
		VALUES ($1, $1, true, 'موضوع', 'open') RETURNING id::text`, customer).Scan(&ticketID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM tickets WHERE id = $1`, ticketID) })

	long := func(n int) string { return strings.Repeat("ع", n+1) }
	js := func(v any) string { b, _ := json.Marshal(v); return string(b) }

	cases := []struct {
		name string
		h    http.HandlerFunc
		id   string
		body string
	}{
		{"عنوانُ الطلب", f.srv.handleCustomerCreateOrder, "", js(map[string]any{
			"address_text": strings.Repeat("A", 20000), "lat": 35.95, "lng": 39.0,
			"payment_method": "cash", "items": []any{}})},
		{"ملاحظاتُ الطلب", f.srv.handleCustomerCreateOrder, "", js(map[string]any{
			"address_text": "عنوان", "notes": long(maxOrderNotes), "lat": 35.95, "lng": 39.0,
			"payment_method": "cash", "items": []any{}})},
		{"ملاحظةُ صنف", f.srv.handleCustomerCreateOrder, "", js(map[string]any{
			"address_text": "عنوان", "lat": 35.95, "lng": 39.0, "payment_method": "cash",
			"items": []any{map[string]any{"menu_item_id": "x", "qty": 1, "note": long(maxItemNote)}}})},
		{"نصُّ الطلب الخاصّ", f.srv.handleCreateCustomOrder, "", js(map[string]any{
			"request": long(maxCustomRequest), "address_text": "عنوان", "lat": 35.95, "lng": 39.0})},
		{"عنوانُ الطلب الخاصّ", f.srv.handleCreateCustomOrder, "", js(map[string]any{
			"request": "خبز", "address_text": long(maxAddressText), "lat": 35.95, "lng": 39.0})},
		{"ملاحظاتُ الطلب الخاصّ", f.srv.handleCreateCustomOrder, "", js(map[string]any{
			"request": "خبز", "address_text": "عنوان", "notes": long(maxOrderNotes), "lat": 35.95, "lng": 39.0})},
		{"منطقةُ العنوان المحفوظ", f.srv.handleCreateAddress, "", js(map[string]any{
			"area_building": long(maxAddressPart), "lat": 35.95, "lng": 39.0})},
		{"شارعُ العنوان المحفوظ", f.srv.handleCreateAddress, "", js(map[string]any{
			"area_building": "المنطقة", "street": long(maxAddressPart), "lat": 35.95, "lng": 39.0})},
		{"طابقُ العنوان المحفوظ", f.srv.handleCreateAddress, "", js(map[string]any{
			"area_building": "المنطقة", "floor": long(maxAddressFloor), "lat": 35.95, "lng": 39.0})},
		{"تعديلُ العنوان المحفوظ", f.srv.handleUpdateAddress, "00000000-0000-0000-0000-000000000000",
			js(map[string]any{"area_building": long(maxAddressPart)})},
		{"رسالةُ الحديث", f.srv.handleSendOrderMessage, orderID, js(map[string]any{
			"body": long(maxChatBody)})},
		{"تفصيلُ الشكوى", f.srv.handleOpenComplaint, orderID, js(map[string]any{
			"reason": "late", "note": long(maxComplaintNote)})},
		{"ردٌّ على تذكرة", f.srv.handleMyTicketReply, ticketID, js(map[string]any{
			"body": long(maxTicketReply)})},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := f.call(c.h, http.MethodPost, "/x", c.id, customer, []string{"customer"}, c.body)
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `"text_too_long"`) {
				t.Fatalf("ردّ %d %s — **والمتوقّع ٤٠٠ text_too_long**", w.Code, w.Body.String())
			}
		})
	}

	// **ولا أثرَ لما رُفض** — لا طلبَ بالعنوان الطويل، ولا رسالةَ ولا عنوان.
	var n int
	if err := f.pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM orders WHERE customer_id = $1 AND length(address_text) > $2)
		     + (SELECT count(*) FROM user_addresses WHERE user_id = $1)
		     + (SELECT count(*) FROM order_messages WHERE order_id = $3)`,
		customer, maxAddressText, orderID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("خُزّن %d صفّاً من نصوصٍ مرفوضة", n)
	}
}
