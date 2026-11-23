package qa

// ══════════════════════════════════════════════════════════════════════
// **«جوائز رحّال غو»** — السحبُ والرصيدُ وطلبُ الجوائز (قرارُ المالك ٢٠٢٦-١١-٢٣)
// ══════════════════════════════════════════════════════════════════════
//
// الإعداداتُ كعائلة `MPS`: عمولةُ ١٠٪، هامشٌ صفر، أجرةٌ ٥٠٠ وحصّةُ المنصّة ١٠٪ (الكابتنُ ٤٥٠).

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/offers"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/prizes"
)

// prizeStore **متجرُ الجوائز وصنفٌ فيه** — وما كان قبله يُطفأ (متجرٌ واحدٌ لا غير).
func prizeStore(t *testing.T, h *Harness, f *Factory, cost int64) (*Merchant, *Item) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET status = 'inactive' WHERE status = 'prize'`); err != nil {
		t.Fatal(err)
	}
	m := f.Merchant()
	item := h.NewItemFor(m, cost)
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET status = 'prize' WHERE id = $1::uuid`, m.ID); err != nil {
		t.Fatalf("متجرُ الجوائز: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `UPDATE merchants SET status = 'inactive' WHERE id = $1::uuid`, m.ID)
	})
	return m, item
}

// award **يمنح رصيدَ جوائز** بالدفتر نفسِه الذي يمنحه السحب.
func award(t *testing.T, h *Harness, userID string, amount int64) {
	t.Helper()
	tx, err := h.Pool.Begin(ctxBG())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctxBG()) }()
	if err := h.API.Prizes().AwardTx(ctxBG(), tx, userID, amount, prizes.MonthStart(time.Now())); err != nil {
		t.Fatalf("المنح: %v", err)
	}
	if err := tx.Commit(ctxBG()); err != nil {
		t.Fatal(err)
	}
}

func prizeBalance(t *testing.T, h *Harness, userID string) int64 {
	t.Helper()
	var b int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE((SELECT balance FROM prize_balances WHERE user_id = $1::uuid), 0)`, userID).Scan(&b); err != nil {
		t.Fatal(err)
	}
	return b
}

// assertPrizeLedger **مجموعُ القيود = الرصيد** — ولا رصيدَ سالب (قيدُ القاعدة).
func assertPrizeLedger(t *testing.T, h *Harness, userID string) {
	t.Helper()
	var sum int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(sum(amount), 0) FROM prize_entries WHERE user_id = $1::uuid`, userID).Scan(&sum); err != nil {
		t.Fatal(err)
	}
	if b := prizeBalance(t, h, userID); b != sum {
		t.Fatalf("**الدفترُ لا يساوي الرصيد**: القيود %d والرصيد %d", sum, b)
	}
}

func prizeOrderBody(it *Item, qty int) map[string]any { return orderBody(it, qty) }

// TestPRZ01_PrizeOrderMoney **طلبُ الجوائز: الرصيدُ أوّلاً، والتوصيلُ على المنصّة، والكابتنُ بأجره.**
//
// صنفٌ ١٠٠٠ ورصيدٌ ٦٠٠ ⇒ خصمُ جوائز ٦٠٠ · أجرةٌ صفر ومعفى ٥٠٠ · مجموعٌ ٤٠٠ نقداً، والكابتنُ ٤٥٠
// كاملاً. **والدفترُ متوازنٌ** (`FI-*`) بلا قيدٍ جديد — الخصمُ تتحمّله الخزينةُ كخصم الكود.
func TestPRZ01_PrizeOrderMoney(t *testing.T) {
	h := New(t)
	treasury(t, h)
	mpsSettings(h)
	mpsNoCampaign(t, h)
	base := financialBaseline(t, h)
	_, item := prizeStore(t, h, h.Factory(), 1000)
	cust := h.Customer()
	award(t, h, cust.ID, 600)

	// **والتسعيرةُ قبل الطلب تقول الشيءَ نفسَه.**
	q := h.POST("/api/v1/public/quote", cust.Token, map[string]any{
		"items": []map[string]any{{"menu_item_id": item.ID, "qty": 1}}, "lat": 35.9506, "lng": 39.0094,
	})
	if q.Code >= 400 {
		t.Fatalf("التسعيرة: %s", q)
	}
	if j := q.JSON(); j["prize_discount"] != float64(600) || j["delivery_fee"] != float64(0) ||
		j["total"] != float64(400) || j["prize_order"] != true {
		t.Fatalf("التسعيرة: %s — والمنتظَر خصمُ جوائز 600 · أجرة 0 · مجموع 400", q)
	}

	oid := mpsPlace(t, h, cust, item, "")
	got := mpsRead(t, h, oid)
	var prize int64
	var isPrize bool
	if err := h.Pool.QueryRow(ctxBG(), `SELECT prize_discount, prize_order FROM orders WHERE id = $1::uuid`, oid).
		Scan(&prize, &isPrize); err != nil {
		t.Fatal(err)
	}
	if !isPrize || prize != 600 || got.discount != 600 || got.delivery != 0 || got.waived != 500 ||
		got.driverFee != 450 || got.total != 400 || got.cashDue != 400 {
		t.Fatalf("الفاتورة: جوائز %v/%d · خصم %d · أجرة %d · معفى %d · كابتن %d · مجموع %d · نقد %d",
			isPrize, prize, got.discount, got.delivery, got.waived, got.driverFee, got.total, got.cashDue)
	}
	if b := prizeBalance(t, h, cust.ID); b != 0 {
		t.Fatalf("الرصيدُ بعد الطلب %d — والمنتظَر 0", b)
	}
	if v := h.GET("/api/v1/my/orders/"+oid, cust.Token); v.JSON()["prize_discount"] != float64(600) {
		t.Fatalf("**الطلبُ لا يقول «من رصيد الجوائز»**: %s", v)
	}

	deliverOrder(t, h, oid, h.driverOf(oid))
	got = mpsRead(t, h, oid)
	if got.driver != 450 || got.collected != 400 || got.merchant != 900 {
		t.Fatalf("بعد التسليم: كابتن %d · قبض %d · متجر %d — والمنتظَر 450 · 400 · 900",
			got.driver, got.collected, got.merchant)
	}
	assertPrizeLedger(t, h, cust.ID)
	assertNewViolations(t, h, base, "FI-06", "FI-10", "FI-04", "FI-03")
}

// TestPRZ02_Rejections **لا خلطَ ولا طلبَ بلا رصيدٍ ولا كود.**
func TestPRZ02_Rejections(t *testing.T) {
	h := New(t)
	mpsSettings(h)
	mpsNoCampaign(t, h)
	_, item := prizeStore(t, h, h.Factory(), 1000)
	other := h.NewItem(700)
	cust := h.Customer()

	if r := h.POSTKey("/api/v1/orders", cust.Token, uniq("p0"), prizeOrderBody(item, 1)); r.Err() != "prize_no_balance" {
		t.Fatalf("بلا رصيد: %s — والمنتظَر prize_no_balance", r)
	}
	award(t, h, cust.ID, 2000)
	mixed := orderBody(item, 1)
	mixed["items"] = []map[string]any{{"menu_item_id": item.ID, "qty": 1}, {"menu_item_id": other.ID, "qty": 1}}
	if r := h.POSTKey("/api/v1/orders", cust.Token, uniq("pm"), mixed); r.Err() != "prize_mixed" {
		t.Fatalf("مخلوط: %s — والمنتظَر prize_mixed", r)
	}
	withCode := orderBody(item, 1)
	withCode["promo_code"] = "ANY"
	if r := h.POSTKey("/api/v1/orders", cust.Token, uniq("pc"), withCode); r.Err() != "prize_no_promo" {
		t.Fatalf("بكود: %s — والمنتظَر prize_no_promo", r)
	}
	if b := prizeBalance(t, h, cust.ID); b != 2000 {
		t.Fatalf("**طلبٌ مرفوضٌ مسّ الرصيد**: %d", b)
	}
}

// TestPRZ03_CancelRefunds **الإلغاءُ قبل التسليم يردّ الرصيد — مرّةً واحدة.**
func TestPRZ03_CancelRefunds(t *testing.T) {
	h := New(t)
	treasury(t, h)
	mpsSettings(h)
	mpsNoCampaign(t, h)
	_, item := prizeStore(t, h, h.Factory(), 1000)
	cust := h.Customer()
	award(t, h, cust.ID, 1500)

	oid := mpsPlace(t, h, cust, item, "")
	if b := prizeBalance(t, h, cust.ID); b != 500 {
		t.Fatalf("بعد الطلب %d — والمنتظَر 500", b)
	}
	if r := h.POST("/api/v1/orders/"+oid+"/cancel", cust.Token, map[string]any{"reason": "غيّرتُ رأيي"}); r.Code >= 400 {
		t.Fatalf("الإلغاء: %s", r)
	}
	if b := prizeBalance(t, h, cust.ID); b != 1500 {
		t.Fatalf("**الإلغاءُ لم يردّ الرصيد**: %d — والمنتظَر 1500", b)
	}
	// **ونداءٌ ثانٍ لا يردّ مرّتين.**
	tx, _ := h.Pool.Begin(ctxBG())
	if n, err := h.API.Prizes().RefundOrderTx(ctxBG(), tx, oid, 0); err != nil || n != 0 {
		t.Fatalf("ردٌّ ثانٍ: %d %v", n, err)
	}
	_ = tx.Commit(ctxBG())
	assertPrizeLedger(t, h, cust.ID)
}

// TestPRZ04_ConcurrentSpend **لا صرفَ مرّتين بمعاملتين متزامنتين** — قفلُ صفّ الرصيد.
//
// رصيدٌ ١٠٠٠ وعشرُ معاملاتٍ تصرف كلٌّ منها ٣٠٠ معاً ⇒ ثلاثٌ تمرّ والرصيدُ ١٠٠.
func TestPRZ04_ConcurrentSpend(t *testing.T) {
	h := New(t)
	_, item := prizeStore(t, h, h.Factory(), 1000)
	cust := h.Customer()
	award(t, h, cust.ID, 1000)
	p := h.API.Prizes()

	// طلبٌ حقيقيٌّ واحدٌ يُنسب له الصرف (مفتاحٌ أجنبيّ).
	mpsSettings(h)
	mpsNoCampaign(t, h)
	var ok, refused int
	var mu sync.Mutex
	var wg sync.WaitGroup
	oid := mpsPlace(t, h, cust, item, "") // يصرف ١٠٠٠ كلَّه
	award(t, h, cust.ID, 1000)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			tx, err := h.Pool.Begin(ctx)
			if err != nil {
				return
			}
			defer func() { _ = tx.Rollback(ctx) }()
			if _, err := p.LockedBalanceTx(ctx, tx, cust.ID); err != nil {
				return
			}
			err = p.SpendTx(ctx, tx, cust.ID, oid, 300)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				refused++
				return
			}
			if tx.Commit(ctx) == nil {
				ok++
			}
		}()
	}
	wg.Wait()
	if ok != 3 || refused != 7 {
		t.Fatalf("مرّ %d ورُفض %d — والمنتظَر 3 و7", ok, refused)
	}
	if b := prizeBalance(t, h, cust.ID); b != 100 {
		t.Fatalf("الرصيد %d — والمنتظَر 100", b)
	}
	assertPrizeLedger(t, h, cust.ID)
}

// TestPRZ05_RecomputeKeepsPrize **إعادةُ الحساب تُبقي حصّةَ الجوائز بقدر البضاعة وتردّ الباقي.**
//
// قطعتان ١٠٠٠ ورصيدٌ ١٥٠٠ ⇒ جوائز ١٥٠٠ ومجموعٌ ٥٠٠. **تُنقص قطعةٌ بتعديل الطلب** ⇒ بضاعةٌ ١٠٠٠،
// جوائز ١٠٠٠ ومجموعٌ صفر، ويعود ٥٠٠ للرصيد. **وكان الخصمُ يُعاد صفراً بلا كود** فتضيع الجوائز.
func TestPRZ05_RecomputeKeepsPrize(t *testing.T) {
	h := New(t)
	treasury(t, h)
	mpsSettings(h)
	mpsNoCampaign(t, h)
	_, item := prizeStore(t, h, h.Factory(), 1000)
	cust := h.Customer()
	award(t, h, cust.ID, 1500)

	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("pr"), prizeOrderBody(item, 2))
	if made.Code >= 400 {
		t.Fatalf("الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	var lineID string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM order_items WHERE order_id = $1::uuid`, oid).Scan(&lineID); err != nil {
		t.Fatal(err)
	}
	admin := h.NewUser("admin")
	if _, _, err := h.Orders.EditOrder(ctxBG(), admin.ID, oid, orders.EditRequest{
		Reason: "نقصت قطعة", Lines: []orders.EditLine{{ItemID: lineID, Qty: 1}},
	}, true, nil); err != nil {
		t.Fatalf("التعديل: %v", err)
	}
	var prize, discount, total, fee, waived int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT prize_discount, discount, total, delivery_fee, promo_delivery_waived FROM orders WHERE id = $1::uuid`, oid).
		Scan(&prize, &discount, &total, &fee, &waived); err != nil {
		t.Fatal(err)
	}
	if prize != 1000 || discount != 1000 || total != 0 || fee != 0 || waived != 500 {
		t.Fatalf("بعد التعديل: جوائز %d · خصم %d · مجموع %d · أجرة %d · معفى %d — والمنتظَر 1000 · 1000 · 0 · 0 · 500",
			prize, discount, total, fee, waived)
	}
	if b := prizeBalance(t, h, cust.ID); b != 500 {
		t.Fatalf("الرصيد %d — والمنتظَر 500 (الفرقُ عاد)", b)
	}
	assertPrizeLedger(t, h, cust.ID)
}

// prizeInvites **يزرع دعواتٍ بتاريخٍ** — كما يكتبها `referrals.Attach`.
func prizeInvites(t *testing.T, h *Harness, inviter *User, n int, at time.Time) {
	t.Helper()
	for i := 0; i < n; i++ {
		invitee := h.Customer()
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO referrals (invitee_id, inviter_id, rank, created_at)
			VALUES ($1::uuid, $2::uuid, $3, $4)`, invitee.ID, inviter.ID, i+1, at); err != nil {
			t.Fatal(err)
		}
	}
}

// prizeClean **سحوباتُ ورصيدُ الفحص السابق تُمحى** — الشهرُ مفتاحٌ مشترك.
func prizeClean(t *testing.T, h *Harness) {
	t.Helper()
	for _, q := range []string{`DELETE FROM prize_winners`, `DELETE FROM prize_draws`} {
		if _, err := h.Pool.Exec(ctxBG(), q); err != nil {
			t.Fatal(err)
		}
	}
}

// TestPRZ06_DrawOnceWeightedRanks **السحبُ مرّةً واحدةً للشهر، بالمراتب وجوائزها.**
//
// ديسمبر ٢٠٢٦: أ دعا ١٠ (فرصتان) · ب دعا ٥ (فرصة) · ج دعا ٤ (لا شيء) · د دعا ٩ نوفمبر (قبل الإطلاق).
// الساعةُ ١ يناير ١١:٥٩ ⇒ لا سحب. ١٢:٠٥ ⇒ رابحان بجائزتي المرتبتين ١٥٠٠ و١٠٠٠. **وسحبان
// متزامنان ثمّ ثالثٌ لا يسحبون مرّةً ثانية.**
func TestPRZ06_DrawOnceWeightedRanks(t *testing.T) {
	h := New(t)
	prizeClean(t, h)
	h.Setting("prizes.enabled", "true")
	h.Setting("prizes.launch_on", `"2026-12-01"`)
	h.Setting("prizes.invites_per_entry", "5")
	h.Setting("prizes.winners", "5")
	// **ودعواتُ الشهر من فحوصٍ أخرى لا تدخل** — تُنقل خارجَه.
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE referrals SET created_at = '2026-06-15' WHERE created_at >= '2026-11-01' AND created_at < '2027-02-01'`); err != nil {
		t.Fatal(err)
	}
	dec := time.Date(2026, 12, 15, 10, 0, 0, 0, offers.Damascus)
	a, b, c, d := h.Customer(), h.Customer(), h.Customer(), h.Customer()
	prizeInvites(t, h, a, 10, dec)
	prizeInvites(t, h, b, 5, dec)
	prizeInvites(t, h, c, 4, dec)
	prizeInvites(t, h, d, 9, time.Date(2026, 11, 20, 10, 0, 0, 0, offers.Damascus))

	p := h.API.Prizes()
	t.Cleanup(func() { p.SetClock(nil); prizeClean(t, h) })
	p.SetClock(func() time.Time { return time.Date(2027, 1, 1, 11, 59, 0, 0, offers.Damascus) })
	if w, err := p.DrawDue(ctxBG()); err != nil || len(w) != 0 {
		t.Fatalf("**سحبٌ قبل موعده**: %v %v", w, err)
	}
	p.SetClock(func() time.Time { return time.Date(2027, 1, 1, 12, 5, 0, 0, offers.Damascus) })

	var wg sync.WaitGroup
	results := make([]int, 3)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w, err := p.DrawDue(ctxBG())
			if err != nil {
				t.Errorf("السحب: %v", err)
			}
			results[i] = len(w)
		}(i)
	}
	wg.Wait()
	drew := 0
	for _, n := range results {
		if n > 0 {
			drew++
			if n != 2 {
				t.Fatalf("الرابحون %d — والمنتظَر 2 (أ و ب)", n)
			}
		}
	}
	if drew != 1 {
		t.Fatalf("**سُحب %d مرّة** — والمنتظَر مرّةً واحدة", drew)
	}
	if w, _ := p.DrawDue(ctxBG()); len(w) != 0 {
		t.Fatal("**سحبٌ ثانٍ للشهر نفسِه**")
	}

	rows, err := h.Pool.Query(ctxBG(), `
		SELECT rank, user_id::text, entries, amount FROM prize_winners WHERE month = '2026-12-01' ORDER BY rank`)
	if err != nil {
		t.Fatal(err)
	}
	won := map[string]int64{}
	for rows.Next() {
		var rank, entries int
		var uid string
		var amt int64
		if err := rows.Scan(&rank, &uid, &entries, &amt); err != nil {
			t.Fatal(err)
		}
		want := map[int]int64{1: 1500, 2: 1000}[rank]
		if amt != want {
			t.Errorf("المرتبة %d جائزتها %d — والمنتظَر %d", rank, amt, want)
		}
		won[uid] = amt
	}
	rows.Close()
	if _, ok := won[a.ID]; !ok {
		t.Error("أ لم يربح")
	}
	if _, ok := won[b.ID]; !ok {
		t.Error("ب لم يربح")
	}
	if _, ok := won[c.ID]; ok {
		t.Error("**ج ربح بلا فرصة**")
	}
	if _, ok := won[d.ID]; ok {
		t.Error("**د ربح بدعواتٍ قبل الإطلاق**")
	}
	for uid, amt := range won {
		if got := prizeBalance(t, h, uid); got != amt {
			t.Errorf("رصيدُ الرابح %d — والمنتظَر %d", got, amt)
		}
	}

	// **والقراءاتُ العامّة**: رابحو آخر سحبٍ بلا رقم.
	res := h.GET("/api/v1/prizes/winners", a.Token)
	if res.Code >= 400 || res.JSON()["month"] != "2026-12" || strings.Contains(string(res.Body), a.Phone) {
		t.Fatalf("الرابحون: %s", res)
	}
	admin := h.NewUser("admin")
	if r := h.GET("/api/v1/admin/prizes/draws", admin.Token); r.Code >= 400 || !strings.Contains(string(r.Body), a.ID) {
		t.Fatalf("سحوباتُ اللوحة: %s", r)
	}
	if r := h.GET("/api/v1/admin/prizes/entries?month=2026-12", admin.Token); r.Code >= 400 ||
		!strings.Contains(string(r.Body), b.ID) || strings.Contains(string(r.Body), c.ID) {
		t.Fatalf("مشتركو اللوحة: %s", r)
	}
	if r := h.GET("/api/v1/admin/prizes/draws", h.Customer().Token); r.Code != http.StatusForbidden {
		t.Fatalf("**زبونٌ يقرأ سحوباتِ اللوحة**: %d", r.Code)
	}
}

// TestPRZ07_RemindOnceThenExpire **«بينتهي بعد ٣ أيام» مرّةً، ثمّ ينتهي آخرَ الشهر.**
func TestPRZ07_RemindOnceThenExpire(t *testing.T) {
	h := New(t)
	p := h.API.Prizes()
	t.Cleanup(func() { p.SetClock(nil) })
	cust := h.Customer()
	p.SetClock(func() time.Time { return time.Date(2027, 1, 1, 12, 5, 0, 0, offers.Damascus) })
	award(t, h, cust.ID, 750)

	reminded := func() bool {
		var r bool
		_ = h.Pool.QueryRow(ctxBG(), `SELECT reminded_for IS NOT NULL FROM prize_balances WHERE user_id = $1::uuid`, cust.ID).Scan(&r)
		return r
	}
	p.SetClock(func() time.Time { return time.Date(2027, 1, 27, 23, 0, 0, 0, offers.Damascus) })
	if _, err := p.RemindDue(ctxBG()); err != nil || reminded() {
		t.Fatalf("**تذكيرٌ قبل الأيّام الثلاثة** (%v)", err)
	}
	p.SetClock(func() time.Time { return time.Date(2027, 1, 29, 9, 0, 0, 0, offers.Damascus) })
	if _, err := p.RemindDue(ctxBG()); err != nil || !reminded() {
		t.Fatalf("لم يُذكَّر (%v)", err)
	}
	var notes int
	count := func() int {
		_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications WHERE user_id = $1::uuid AND title = $2`,
			cust.ID, "رصيد جوائزك بينتهي بعد ٣ أيام").Scan(&notes)
		return notes
	}
	if count() != 1 {
		t.Fatalf("إشعاراتُ التذكير %d — والمنتظَر 1", notes)
	}
	_, _ = p.RemindDue(ctxBG())
	if count() != 1 {
		t.Fatalf("**ذُكِّر مرّتين**: %d", notes)
	}

	p.SetClock(func() time.Time { return time.Date(2027, 1, 31, 23, 59, 0, 0, offers.Damascus) })
	_, _ = p.ExpireDue(ctxBG())
	if b := prizeBalance(t, h, cust.ID); b != 750 {
		t.Fatalf("**انتهى قبل آخر الشهر**: %d", b)
	}
	p.SetClock(func() time.Time { return time.Date(2027, 2, 1, 0, 0, 1, 0, offers.Damascus) })
	if _, err := p.ExpireDue(ctxBG()); err != nil {
		t.Fatal(err)
	}
	if b := prizeBalance(t, h, cust.ID); b != 0 {
		t.Fatalf("الرصيدُ بعد الشهر %d — والمنتظَر 0", b)
	}
	assertPrizeLedger(t, h, cust.ID)
	if m := h.GET("/api/v1/prizes/me", cust.Token); m.Code >= 400 || m.JSON()["prize_balance"] != float64(0) {
		t.Fatalf("حالي: %s", m)
	}
}

// TestPRZ08_PrizeStoreHiddenEverywhere **متجرُ الجوائز غائبٌ عن كلّ ما يراه الزبون.**
//
// متجران بأصنافٍ تحمل الكلمةَ نفسَها: نشطٌ (شاهدٌ أنّ الباب يجد) ومتجرُ الجوائز. **كلُّ قائمةٍ
// وبحثٍ تجد الأوّلَ ولا تذكر الثاني** — **والاستثناءُ الوحيدُ نافذةُ الصنف بمعرّفه.**
func TestPRZ08_PrizeStoreHiddenEverywhere(t *testing.T) {
	h := New(t)
	mpsSettings(h)
	f := h.Factory()
	pm, pitem := prizeStore(t, h, f, 1000)
	am := f.Merchant()
	aitem := h.NewItemFor(am, 1000)
	// **وفي مدينةٍ واحدة** — فالتصفّحُ والبحثُ بسوق المدينة (`CITY-030`).
	city := h.cityAt("مدينة الجوائز", 35.9506, 39.0094, 25000, nil)
	h.placeMerchant(pitem, city, 35.9506, 39.0094)
	h.placeMerchant(aitem, city, 35.9506, 39.0094)
	token := strings.ToUpper(uniq("PRZ"))
	for _, it := range []*Item{pitem, aitem} {
		if _, err := h.Pool.Exec(ctxBG(), `UPDATE menu_items SET name = $2 WHERE id = $1::uuid`, it.ID, token+" "+it.ID[:4]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET name = $2, featured = true WHERE id = ANY($1::uuid[])`,
		[]string{pm.ID, am.ID}, token+" store"); err != nil {
		t.Fatal(err)
	}
	// **وعرضٌ على صنف الجوائز** — لا يظهر في العروض.
	admin := h.NewUser("admin")
	var offerID string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO offers (kind, title, menu_item_id, discount_amount, borne_by, starts_at, ends_at, active, created_by)
		VALUES ('discount', 'عرض', $1::uuid, 100, 'platform', now() - interval '1 hour', now() + interval '1 day', true, $2::uuid)
		RETURNING id::text`, pitem.ID, admin.ID).Scan(&offerID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM offers WHERE id = $1::uuid`, offerID) })
	cust := h.Customer()
	if r := h.POST("/api/v1/my/favorites/"+pitem.ID, cust.Token, nil); r.Code >= 500 {
		t.Logf("المفضّلة: %s", r)
	}

	q := url.QueryEscape(token)
	geo := "lat=35.9506&lng=39.0094"
	paths := []string{
		"/api/v1/public/search/items?q=" + q + "&" + geo,
		"/api/v1/public/search?q=" + q + "&" + geo,
		"/api/v1/public/suggest?q=" + q + "&" + geo,
		"/api/v1/public/items?" + geo,
		"/api/v1/public/sections?" + geo,
		"/api/v1/public/sections/" + pitem.SectionID + "/items?" + geo,
		"/api/v1/public/stores/" + pm.ID,
		"/api/v1/public/stores/" + pm.ID + "/items",
		"/api/v1/public/stores/" + pm.ID + "/summary",
		"/api/v1/public/map/stores",
		"/api/v1/public/offers",
		"/api/v1/public/home?" + geo,
		"/api/v1/my/favorites",
	}
	for _, p := range paths {
		r := h.GET(p, cust.Token)
		body := string(r.Body)
		if strings.Contains(body, pitem.ID) || strings.Contains(body, pm.ID+`"`) && !strings.HasPrefix(p, "/api/v1/public/stores/"+pm.ID) {
			t.Errorf("**متجرُ الجوائز ظهر** في %s: %d", p, r.Code)
		}
		if strings.HasPrefix(p, "/api/v1/public/stores/"+pm.ID) && r.Code < 400 && strings.Contains(body, pitem.ID) {
			t.Errorf("**صفحةُ متجر الجوائز تُفتح** %s: %s", p, r)
		}
	}
	// **والشاهد**: البحثُ يجد الصنفَ النشط — فالغيابُ أعلاه غيابٌ لا عطب.
	if r := h.GET(paths[0], cust.Token); !strings.Contains(string(r.Body), aitem.ID) {
		t.Fatalf("الشاهدُ لم يُوجد — البحثُ لا يعمل: %s", r)
	}
	// **والاستثناء**: نافذةُ الصنف بمعرّفه تفتح صنفَ الجوائز متاحاً.
	if r := h.GET("/api/v1/public/items/"+pitem.ID, cust.Token); r.Code != http.StatusOK || func() bool { it, _ := r.JSON()["item"].(map[string]any); return it["available"] != true }() {
		t.Fatalf("نافذةُ صنف الجوائز: %s", r)
	}
	if r := h.GET("/api/v1/prizes/items", cust.Token); !strings.Contains(string(r.Body), pitem.ID) ||
		strings.Contains(string(r.Body), aitem.ID) {
		t.Fatalf("أصنافُ الجوائز: %s", r)
	}
	if r := h.GET("/api/v1/prizes/me", cust.Token); r.JSON()["store_id"] != pm.ID {
		t.Fatalf("حالي لا يقول متجرَ الجوائز: %s", r)
	}
}
