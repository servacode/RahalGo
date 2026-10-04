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
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/textguard"
)

// varied **نصٌّ عربيٌّ من `n` حرفاً بلا تكرارٍ يُطوى** — فالحارسُ يقصّ الحرفَ
// المكرَّرَ أكثرَ من عشر، **ونصٌّ من حرفٍ واحدٍ مكرَّرٍ لا يقيس الحدّ.**
func varied(n int) string {
	return string([]rune(strings.Repeat("عبد", n/3+1))[:n])
}

// TestTextLimits_CountsLettersNotBytes **الحدُّ بالحروف** — والعربيُّ بايتان.
func TestTextLimits_CountsLettersNotBytes(t *testing.T) {
	var s Server
	arabic := varied(maxAddressText)
	if _, err := s.guardText(context.Background(), tf("address_text", &arabic, maxAddressText, textguard.Address)); err != nil {
		t.Fatalf("عنوانٌ من %d حرفاً عربيّاً رُفض — **عُدّ بالبايت لا بالحرف**: %v", maxAddressText, err)
	}
	over := varied(maxAddressText + 1)
	if _, err := s.guardText(context.Background(), tf("address_text", &over, maxAddressText, textguard.Address)); err == nil {
		t.Fatal("حرفٌ فوق الحدّ قُبل")
	}
}

// TestAddressLine_FitsOrderLimit **العنوانُ المحفوظُ بأطول أجزائه يُقبل في الطلب.**
//
// **والسطرُ يُركَّب من الأجزاء ثمّ يُرسَل `address_text`** — فإن زاد على حدّ
// الطلب رُفض طلبُ زبونٍ لم يكتب إلّا ما قُبل منه.
func TestAddressLine_FitsOrderLimit(t *testing.T) {
	line := addressLine(varied(maxAddressPart), varied(maxAddressPart), "1234567890123456789")
	var s Server
	if _, err := s.guardText(context.Background(), tf("address_text", &line, maxAddressText, textguard.Address)); err != nil {
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

	long := func(n int) string { return varied(n + 1) }
	js := func(v any) string { b, _ := json.Marshal(v); return string(b) }

	cases := []struct {
		name string
		h    http.HandlerFunc
		id   string
		body string
	}{
		{"عنوانُ الطلب", f.srv.handleCustomerCreateOrder, "", js(map[string]any{
			"address_text": varied(20000), "lat": 35.95, "lng": 39.0,
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

// TestOffensiveText_RejectOrMask **الاسمُ والعنوانُ والملاحظاتُ ترفض — والحديثُ
// والشكوى تُخفي وتُنبّه الإدارة.** (قرارُ المالك ٢٠٢٦-١٠-٠٣.)
func TestOffensiveText_RejectOrMask(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	f.srv.comms = comms.New(f.pool)
	f.srv.support = support.NewService(f.pool, f.srv.identity, f.srv.wallet)
	f.srv.notify = notifications.New(f.pool, f.srv.hub, f.srv.logger)
	ctx := context.Background()
	ops := testdb.NewUser(t, f.pool, "ops")
	if _, err := f.pool.Exec(ctx, `INSERT INTO user_roles (user_id, role_code) VALUES ($1::uuid, 'ops') ON CONFLICT DO NOTHING`, ops); err != nil {
		t.Fatal(err)
	}
	orderID := f.problemOrderAt(t, "on_the_way", d)
	var customer string
	if err := f.pool.QueryRow(ctx, `SELECT customer_id::text FROM orders WHERE id = $1`,
		orderID).Scan(&customer); err != nil {
		t.Fatal(err)
	}
	js := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	call := func(h http.HandlerFunc, id, body string) (int, string) {
		w := f.call(h, http.MethodPost, "/x", id, customer, []string{"customer"}, body)
		return w.Code, w.Body.String()
	}

	// ── يُرفض ──
	for name, c := range map[string]struct {
		h    http.HandlerFunc
		body string
		code string
	}{
		"عنوانٌ محفوظٌ بشتيمة":   {f.srv.handleCreateAddress, js(map[string]any{"area_building": "حي ك ل ب", "lat": 35.95, "lng": 39.0}), "text_offensive"},
		"عنوانٌ محفوظٌ بإيموجي":  {f.srv.handleCreateAddress, js(map[string]any{"area_building": "بيتي 🏠", "lat": 35.95, "lng": 39.0}), "text_bad_chars"},
		"نصُّ طلبٍ خاصٍّ بشتيمة": {f.srv.handleCreateCustomOrder, js(map[string]any{"request": "هات خبز يا 7مار", "address_text": "عنوان", "lat": 35.95, "lng": 39.0}), "text_offensive"},
	} {
		if code, body := call(c.h, "", c.body); code != http.StatusBadRequest || !strings.Contains(body, `"`+c.code+`"`) {
			t.Errorf("%s: %d %s — والمتوقّع ٤٠٠ %s", name, code, body, c.code)
		}
	}

	// ── يُخفى ويصل ويُنبَّه به ──
	code, body := call(f.srv.handleSendOrderMessage, orderID, js(map[string]any{"body": "تأخرت كتير يا حمار 😡"}))
	if code != http.StatusCreated {
		t.Fatalf("الحديثُ رُفض: %d %s", code, body)
	}
	var stored, word string
	var flagged bool
	if err := f.pool.QueryRow(ctx, `SELECT body, flagged, COALESCE(flag_word, '') FROM order_messages
		WHERE order_id = $1 AND NOT auto ORDER BY created_at DESC LIMIT 1`, orderID).Scan(&stored, &flagged, &word); err != nil {
		t.Fatal(err)
	}
	if stored != "تأخرت كتير يا *** 😡" || !flagged || word != "حمار" {
		t.Errorf("الحديثُ لم يُخفَ ويُوسَم: %q flagged=%v word=%q", stored, flagged, word)
	}

	// **والشكوى على طلبٍ انتهى.**
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'delivered', delivered_at = now(), closed_at = now() WHERE id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	code, body = call(f.srv.handleOpenComplaint, orderID, js(map[string]any{"reason": "late", "note": "السائق كلب"}))
	if code >= 400 {
		t.Fatalf("الشكوى رُفضت: %d %s", code, body)
	}
	var reply string
	if err := f.pool.QueryRow(ctx, `SELECT COALESCE(body, '') FROM ticket_replies tr JOIN tickets t ON t.id = tr.ticket_id
		WHERE t.order_id = $1 ORDER BY tr.created_at LIMIT 1`, orderID).Scan(&reply); err != nil {
		t.Fatal(err)
	}
	if reply != "السائق ***" {
		t.Errorf("الشتيمةُ لم تُخفَ في الشكوى: %q", reply)
	}

	var alerts int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND title = $2`,
		ops, notifTitles.offensiveText).Scan(&alerts); err != nil {
		t.Fatal(err)
	}
	if alerts != 2 {
		t.Errorf("الإدارةُ نُبّهت %d مرّة — والمتوقّع مرّتان (الحديثُ والشكوى)", alerts)
	}
}
