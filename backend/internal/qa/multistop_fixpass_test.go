package qa

// ══════════════════════════════════════════════════════════════════════
// **جولةُ إصلاح المراجعة على الدفعة P2** — طلبٌ واحدٌ بعدّة محطّات
// ══════════════════════════════════════════════════════════════════════
//
// كلُّ اختبارٍ هنا كان ساقطاً قبل إصلاحه.

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/opsmap"
)

// notifiedWith **أوصل المستخدمَ إشعارٌ بهذا العنوان على هذا الطلب؟** — ينتظر قليلاً.
func notifiedWith(t *testing.T, h *Harness, userID, oid, title string, within time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(within)
	for {
		var n int
		if err := h.Pool.QueryRow(ctxBG(), `
			SELECT count(*) FROM notifications
			WHERE user_id = $1::uuid AND entity_id = $2 AND title = $3`, userID, oid, title).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TestMultiStop_CancelWindowFromFirstPartWhenAccepted **مهلةُ الإلغاء من أوّل قبولٍ — ولو صار الطلبُ «مقبولاً»** (G7).
//
// **كان آخرُ جوابٍ يكتب `accepted_at = now()`** — فيقرأ فحصُ المهلة وعدّادُها منه، **فتنفتح
// المهلةُ من جديد والمتجرُ الأوّلُ يطبخ منذ عشر دقائق.**
func TestMultiStop_CancelWindowFromFirstPartWhenAccepted(t *testing.T) {
	s := newMSShop(t, "merchants")
	s.h.Setting("orders.customer_cancel_window_sec", "120")
	oid := s.order(t)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
	// **قبل الأوّلُ منذ عشر دقائق، ثمّ قبل الثاني الآن فصار الطلبُ «مقبولاً»** — كما يقع
	// حين ينقلب الوضعُ إلى «المنصّة تدير» والطلبُ مفتوح، أو يتعثّر التحضيرُ التلقائيّ.
	if _, err := s.h.Pool.Exec(ctxBG(), `
		UPDATE order_parts SET answered_at = now() - interval '10 minutes'
		WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, s.mA.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.h.Pool.Exec(ctxBG(), `
		UPDATE order_parts SET status = 'accepted', answered_at = now(), answered_role = 'merchant'
		WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, s.mB.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.h.Pool.Exec(ctxBG(), `
		UPDATE orders SET status = 'accepted', accepted_at = now() WHERE id = $1::uuid`, oid); err != nil {
		t.Fatal(err)
	}

	r := s.h.GET("/api/v1/my/orders/"+oid, s.cust.Token)
	if r.Code != http.StatusOK {
		t.Fatalf("طلبُ الزبون: %s", r)
	}
	if left := num(r.JSON()["cancel_seconds_left"]); left != 0 {
		t.Errorf("**عدّادُ المهلة %d ثانية** — يُنتظر صفر: الأوّلُ قبل منذ عشر دقائق", left)
	}
	c := s.h.POST("/api/v1/orders/"+oid+"/cancel", s.cust.Token, map[string]any{"note": "بدا لي"})
	if c.Code < 400 {
		t.Errorf("**ألغى الزبونُ بعد انقضاء المهلة** — والمتجرُ الأوّلُ يطبخ منذ عشر دقائق: %s", c)
	}
	if got := s.h.statusOf(oid); got != "accepted" {
		t.Errorf("حالُ الطلب %q — يُنتظر accepted", got)
	}
}

// TestMultiStop_CustomerCancelTellsAcceptedStores **ألغى الزبونُ ⇒ من قبل قسمَه يُخبَر** (G7).
//
// **كان المتجرُ الذي قبل وبدأ يطبخ لا يعلم** إلّا بإشارة البثّ أو حين يعيد التحميل.
func TestMultiStop_CustomerCancelTellsAcceptedStores(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid := s.order(t)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
	if r := s.h.POST("/api/v1/orders/"+oid+"/cancel", s.cust.Token, map[string]any{"note": "بدا لي"}); r.Code >= 400 {
		t.Fatalf("إلغاءُ الزبون: %s", r)
	}
	const title = "الزبون ألغى طلب قبلته"
	if !notifiedWith(t, s.h, s.mA.Owner.ID, oid, title, 2*time.Second) {
		t.Errorf("**لم يُخبَر المتجرُ الذي قبل قسمَه بإلغاء الزبون**")
	}
	if notifiedWith(t, s.h, s.mB.Owner.ID, oid, title, 300*time.Millisecond) {
		t.Errorf("أُخبر متجرٌ لم يقبل قسمَه")
	}
}

// TestMultiStop_PartEventsKeepAssignedAndDoorClocks **حدثُ القسم لا يصفّر «مُسند منذ» ولا انتظارَ الباب** (D7).
//
// **أحداثُ الأقسام تُكتب بحال الطلب** (`to_status`) — **وكان قارئان يأخذان آخرَ حدثٍ بها**:
// خريطةُ المكتب (`assigned`) وبلاغُ المرحلة (`at_dropoff`). **وكانت تذكرةٌ تُعاد بعد الاستلام.**
func TestMultiStop_PartEventsKeepAssignedAndDoorClocks(t *testing.T) {
	s := newMSShop(t, "platform")
	oid := s.order(t)
	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/transition", s.admin.Token,
		map[string]any{"to": "accepted"}); r.Code >= 400 {
		t.Fatalf("قبولُ المكتب: %s", r)
	}
	send := func(mid string) Res {
		return s.h.POST("/api/v1/admin/orders/"+oid+"/whatsapp", s.admin.Token,
			map[string]any{"channel": "manual", "merchant_id": mid})
	}
	for _, m := range []*Merchant{s.mA, s.mB} {
		if r := send(m.ID); r.Code >= 400 {
			t.Fatalf("الإرسال: %s", r)
		}
	}
	drv := s.f.Driver(OnShift())
	t.Cleanup(func() {
		s.h.POST("/api/v1/admin/orders/"+oid+"/transition", s.admin.Token,
			map[string]any{"to": "cancelled", "note": "تنظيف اختبار"})
		_, _ = s.h.Pool.Exec(ctxBG(), `UPDATE users SET on_shift = false WHERE id = $1::uuid`, drv.ID)
	})
	if r := s.h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil); r.Code >= 400 {
		t.Fatalf("قبولُ السائق: %s", r)
	}
	if _, err := s.h.Pool.Exec(ctxBG(), `
		UPDATE order_events SET created_at = now() - interval '10 minutes'
		WHERE order_id = $1::uuid AND to_status = 'assigned'`, oid); err != nil {
		t.Fatal(err)
	}
	// **إعادةُ تذكرةٍ بعد الإسناد** — حدثُ قسمٍ بحال `assigned`.
	if r := send(s.mA.ID); r.Code >= 400 {
		t.Fatalf("إعادةُ الإرسال بعد الإسناد: %s", r)
	}
	list, err := opsmap.ActiveOrders(context.Background(), s.h.Pool, nil,
		opsmap.OrderFilter{DriverID: drv.ID}, false, 10)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, o := range list {
		if o.ID != oid {
			continue
		}
		found = true
		if o.AssignedAt == nil || time.Since(*o.AssignedAt) < 9*time.Minute {
			t.Errorf("**«مُسند منذ» صفّرته إعادةُ التذكرة**: %v — يُنتظر قبل عشر دقائق", o.AssignedAt)
		}
	}
	if !found {
		t.Fatalf("الطلبُ ليس على خريطة المكتب")
	}

	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		if r := s.h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to}); r.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s: %s", to, r)
		}
	}
	if _, err := s.h.Pool.Exec(ctxBG(), `
		UPDATE order_events SET created_at = now() - interval '7 minutes 30 seconds'
		WHERE order_id = $1::uuid AND to_status = 'at_dropoff' AND merchant_id IS NULL`, oid); err != nil {
		t.Fatal(err)
	}
	// **ولا تذكرةَ بعد الاستلام** — البضاعةُ مع السائق.
	if r := send(s.mA.ID); r.Code < 400 {
		t.Errorf("**أُعيدت تذكرةٌ والسائقُ عند باب الزبون**: %s", r)
	}
	// **وحدثُ قسمٍ بحال الباب** (جهوزٌ متأخّرٌ مثلاً) لا يصفّر انتظارَ السائق.
	if _, err := s.h.Pool.Exec(ctxBG(), `
		INSERT INTO order_events (order_id, from_status, to_status, note, merchant_id)
		VALUES ($1::uuid, 'at_dropoff', 'at_dropoff', 'حدثُ قسم', $2::uuid)`, oid, s.mB.ID); err != nil {
		t.Fatal(err)
	}
	if r := s.h.POST("/api/v1/driver/orders/"+oid+"/report", drv.Token,
		map[string]any{"code": "customer_absent"}); r.Code >= 400 {
		t.Fatalf("بلاغُ المرحلة: %s", r)
	}
	var body string
	deadline := time.Now().Add(2 * time.Second)
	for body == "" && time.Now().Before(deadline) {
		_ = s.h.Pool.QueryRow(ctxBG(), `
			SELECT body FROM notifications WHERE entity_id = $1 AND body LIKE '%عند الباب منذ%' LIMIT 1`,
			oid).Scan(&body)
		if body == "" {
			time.Sleep(50 * time.Millisecond)
		}
	}
	if !strings.Contains(body, "منذ 7 د") {
		t.Errorf("**انتظارُ الباب في بلاغ المرحلة** %q — يُنتظر «منذ 7 د»", body)
	}
}

// TestMultiStop_DriverReportStopOnSingleStoreOrder **`stop` في طلب المتجر الواحد لا يُضيّع جهةَ البلاغ** (C7).
//
// **كانت `stop` تُسأل في الأقسام وحدَها** — وطلبُ المتجر الواحد بلا أقسام، فيبقى البلاغُ بلا جهة.
func TestMultiStop_DriverReportStopOnSingleStoreOrder(t *testing.T) {
	s := newMSShop(t, "merchants")
	r := s.h.POSTKey("/api/v1/orders", s.cust.Token, uniq("ms1stop"), orderBody(s.a, 1))
	if r.Code >= 400 {
		t.Fatalf("الإنشاء: %s", r)
	}
	oid, _ := r.JSON()["id"].(string)
	drv := s.h.driverOf(oid)
	deliverOrder(t, s.h, oid, drv)
	if r := s.h.POST("/api/v1/driver/orders/"+oid+"/report", drv.Token,
		map[string]any{"reason": "merchant_slow", "note": "تأخّر", "stop": s.mA.ID}); r.Code >= 400 {
		t.Fatalf("بلاغُ السائق: %s", r)
	}
	var against *string
	if err := s.h.Pool.QueryRow(ctxBG(), `
		SELECT against_user_id::text FROM tickets WHERE order_id = $1::uuid AND created_by = $2::uuid`,
		oid, drv.ID).Scan(&against); err != nil {
		t.Fatal(err)
	}
	if against == nil || *against != s.mA.Owner.ID {
		t.Errorf("**بلاغٌ «على المتجر» في طلب المتجر الواحد جهتُه %v** — يُنتظر صاحبُ المتجر", against)
	}
}

// TestMultiStop_WaivedCommissionInStoreMoney **عمولةٌ مُحيت لا تظهر للمتجر في طلبٍ بعدّة محطّات** (G-14).
//
// **كان فرعُ الأقسام يحسب `الكلفة − الحقّ` ولا يطرح المحوّ** — فيرى المتجرُ عمولةً لم تُدفع.
func TestMultiStop_WaivedCommissionInStoreMoney(t *testing.T) {
	s := newMSShop(t, "merchants")
	base := financialBaseline(t, s.h)
	for _, m := range []*Merchant{s.mA, s.mB} {
		if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE merchants SET settlement_method = 'cash' WHERE id = $1::uuid`,
			m.ID); err != nil {
			t.Fatal(err)
		}
	}
	oid := s.order(t)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
	s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})
	deliverOrder(t, s.h, oid, s.h.driverOf(oid))
	fin := s.h.NewUser("finance")
	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/waive-commission", fin.Token,
		map[string]any{"reason": "المتجر رفض يخصم العمولة"}); r.Code != http.StatusOK {
		t.Fatalf("المحو: %s", r)
	}
	for _, c := range []struct {
		label string
		m     *Merchant
		tok   string
		sales int64
	}{{"الأوّل", s.mA, s.tokA, 1000}, {"الثاني", s.mB, s.tokB, 500}} {
		r := s.h.GET("/api/v1/merchant/stores/"+c.m.ID+"/reports", c.tok)
		if r.Code != http.StatusOK {
			t.Fatalf("التقرير: %s", r)
		}
		sum, _ := r.JSON()["summary"].(map[string]any)
		if num(sum["platform_commission"]) != 0 || num(sum["sales"]) != c.sales {
			t.Errorf("**تقريرُ المتجر %s بعد المحو** %v — يُنتظر عمولةٌ صفر ومبيعٌ %d", c.label, sum, c.sales)
		}
		v := listed(s.h.GET("/api/v1/merchant/stores/"+c.m.ID+"/orders?with_active=true", c.tok), oid)
		if v == nil || num(v["platform_commission"]) != 0 || num(v["merchant_net"]) != c.sales {
			t.Errorf("**سجلُّ المتجر %s بعد المحو** %v — يُنتظر عمولةٌ صفر وصافٍ %d", c.label, v, c.sales)
		}
	}
	assertNewViolations(t, s.h, base)
}

// TestMultiStop_OwnerOfTwoBranchesSeesBothMoney **مالكُ فرعين في الطلب يرى مالَ الفرعين مع بنودهما** (§3.3).
//
// **كانت البنودُ للفرعين والمجموعُ والصافي والعمولةُ لفرعٍ واحد.**
func TestMultiStop_OwnerOfTwoBranchesSeesBothMoney(t *testing.T) {
	s := newMSShop(t, "merchants")
	mA2 := s.f.Merchant(OwnedBy(s.mA.Owner), Commission(20))
	a2 := s.h.NewItemPriced(mA2, 500, 600)
	r := s.h.POSTKey("/api/v1/orders", s.cust.Token, uniq("ms2own"), twoStoreBody(s.a, a2))
	if r.Code >= 400 {
		t.Fatalf("الإنشاء: %s", r)
	}
	oid, _ := r.JSON()["id"].(string)
	g := s.h.GET("/api/v1/merchant/orders/"+oid, s.tokA)
	if g.Code != http.StatusOK {
		t.Fatalf("فتحُ الطلب: %s", g)
	}
	v := g.JSON()
	items, _ := v["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("%d صنفاً — يُنتظر بندا الفرعين", len(items))
	}
	if num(v["subtotal"]) != 1500 || num(v["platform_commission"]) != 200 || num(v["merchant_net"]) != 1300 {
		t.Errorf("**مالُ مالك الفرعين** المجموع %v العمولة %v الصافي %v — يُنتظر 1500 و200 و1300",
			v["subtotal"], v["platform_commission"], v["merchant_net"])
	}
}

// TestMultiStop_StoreStatusFilterMatchesProjection **فلترُ الحال في سجلّ المتجر يجد ما يُعرض** (§3.3).
//
// **في وضع المتاجر يُعرض القسمُ المقبولُ «قيد التحضير»** — وكان الفلترُ يبحث عنه بـ`accepted`.
func TestMultiStop_StoreStatusFilterMatchesProjection(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid := s.order(t)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
	path := "/api/v1/merchant/stores/" + s.mA.ID + "/orders?with_active=true&status="
	v := listed(s.h.GET(path+"preparing", s.tokA), oid)
	if v == nil {
		t.Errorf("**فلترُ «قيد التحضير» أضاع الطلب** — وهو يُعرض بها")
	} else if v["status"] != "preparing" {
		t.Errorf("حالُه في القائمة %v", v["status"])
	}
	if w := listed(s.h.GET(path+"accepted", s.tokA), oid); w != nil {
		t.Errorf("**فلترُ «مقبول» ردّ صفّاً مكتوبٌ عليه %v**", w["status"])
	}
}

// TestMultiStop_AutoTransferAnswersPartsAsSystem **التحويلُ التلقائيُّ يجيب الأقسامَ باسم النظام** (G1).
//
// **كان الفاعلُ الزبونَ الذي أنشأ الطلب بدور المكتب** — فيُنسب «قبل X قسمه» إليه.
func TestMultiStop_AutoTransferAnswersPartsAsSystem(t *testing.T) {
	s := newMSShop(t, "platform")
	n := &msNotifier{ready: true}
	s.h.Setting("orders.auto_transfer", "true")
	s.h.API.SetMerchantNotifier(n)
	t.Cleanup(func() { s.h.API.SetMerchantNotifier(nil) })
	oid := s.order(t)
	if got := waitStatusIs(t, s.h, oid, "dispatching", 4*time.Second); got != "dispatching" {
		t.Fatalf("بعد التحويل التلقائيّ: %q — يُنتظر dispatching", got)
	}
	var bad int
	if err := s.h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM order_parts
		WHERE order_id = $1::uuid AND (answered_role <> 'system' OR answered_by IS NOT NULL)`, oid).Scan(&bad); err != nil {
		t.Fatal(err)
	}
	if bad != 0 {
		t.Errorf("**%d قسماً أُجيب باسم الزبون** — يُنتظر النظام", bad)
	}
	var byCustomer int
	if err := s.h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM order_events
		WHERE order_id = $1::uuid AND actor_id = $2::uuid AND to_status <> 'pending'`, oid, s.cust.ID).Scan(&byCustomer); err != nil {
		t.Fatal(err)
	}
	if byCustomer != 0 {
		t.Errorf("**%d حدثاً بعد الإنشاء منسوبٌ إلى الزبون**", byCustomer)
	}
}

// TestMultiStop_LastRejectAndSweeperUnderLock **رفضُ الأخير والكانسُ يتزاحمان على القفل** — `T32` (D6).
//
// **بنودُ المتجر الثاني خرجت من الطلب** فيقدر الكانسُ أن يقبل. **ويُمسَك قفلُ الطلب حتّى
// ينتظر الاثنان معاً** ثمّ يُطلق — أيُّهما سبق: **«مقبولٌ» ومعه قسمٌ مقبول**، أو «مرفوضٌ» ولا
// قسمَ مقبولاً ولا حدثَ قبول.
func TestMultiStop_LastRejectAndSweeperUnderLock(t *testing.T) {
	for _, round := range []string{"أ", "ب"} {
		t.Run(round, func(t *testing.T) {
			s := newMSShop(t, "merchants")
			s.h.Setting("orders.auto_accept_min", "1")
			oid := s.order(t)
			s.mustAnswer(t, "رفضُ الثاني", s.tokB, oid, map[string]any{"to": "rejected", "note": "نفد الصنف"})
			var line string
			if err := s.h.Pool.QueryRow(ctxBG(), `
			SELECT id::text FROM order_items WHERE order_id = $1::uuid AND merchant_id = $2::uuid`,
				oid, s.mB.ID).Scan(&line); err != nil {
				t.Fatal(err)
			}
			if r := s.h.POST("/api/v1/admin/orders/"+oid+"/items/"+line+"/unavailable", s.admin.Token,
				map[string]any{}); r.Code >= 400 {
				t.Fatalf("بنودُ الثاني تخرج: %s", r)
			}
			if got := s.h.statusOf(oid); got != "pending" {
				t.Fatalf("قبل السباق: %q — يُنتظر pending", got)
			}
			if _, err := s.h.Pool.Exec(ctxBG(),
				`UPDATE orders SET created_at = now() - interval '10 minutes' WHERE id = $1::uuid`, oid); err != nil {
				t.Fatal(err)
			}

			tx, err := s.h.Pool.Begin(ctxBG())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctxBG(), `SELECT 1 FROM orders WHERE id = $1::uuid FOR UPDATE`, oid); err != nil {
				t.Fatal(err)
			}
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				s.answer(s.tokA, oid, map[string]any{"to": "rejected", "note": "مغلق للصيانة"})
			}()
			go func() {
				defer wg.Done()
				s.h.Orders.SweepAutoAcceptOnce(context.Background())
			}()
			time.Sleep(400 * time.Millisecond)
			_ = tx.Rollback(ctxBG())
			wg.Wait()

			var status string
			var accParts, accEvents int
			if err := s.h.Pool.QueryRow(ctxBG(), `
			SELECT o.status,
			       (SELECT count(*) FROM order_parts p WHERE p.order_id = o.id AND p.status = 'accepted'),
			       (SELECT count(*) FROM order_events e WHERE e.order_id = o.id
			           AND e.to_status = 'accepted' AND e.merchant_id IS NULL)
			FROM orders o WHERE o.id = $1::uuid`, oid).Scan(&status, &accParts, &accEvents); err != nil {
				t.Fatal(err)
			}
			switch status {
			case "accepted", "preparing", "dispatching":
				if accParts == 0 {
					t.Errorf("جولة %s: **الطلبُ %s ولا قسمَ مقبول**", round, status)
				}
			case "rejected":
				if accParts != 0 || accEvents != 0 {
					t.Errorf("جولة %s: **رُفض الطلبُ ومعه %d قسماً مقبولاً و%d حدثَ قبول**", round, accParts, accEvents)
				}
			default:
				t.Errorf("جولة %s: حالٌ غيرُ منتظرة %q", round, status)
			}
		})
	}
}
