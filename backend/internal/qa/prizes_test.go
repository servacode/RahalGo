package qa

// ══════════════════════════════════════════════════════════════════════
// **«جوائز رحّال غو»** — النقاطُ والإعلانُ والرصيدُ وطلبُ الجوائز (قرارا المالك ٢٠٢٦-١١-٢٣ و٢٠٢٦-١١-٢٥)
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

// award **يمنح رصيدَ جوائز** بالدفتر نفسِه الذي يمنحه الإعلان.
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

// prizeClean **إعلاناتُ ونقاطُ الفحص السابق تُمحى** — الشهرُ مفتاحٌ مشترك.
//
// **ومصادرُ النقاط في أشهر الفحص (٢٠٣١) تُنقل خارجَها** — فما زرعته جولةٌ سابقةٌ لا يدخل.
func prizeClean(t *testing.T, h *Harness) {
	t.Helper()
	for _, q := range []string{
		`DELETE FROM prize_winners`, `DELETE FROM prize_draws`,
		`DELETE FROM prize_checkins`, `DELETE FROM prize_follows`,
		`UPDATE referrals SET created_at = '2026-06-15' WHERE created_at >= '2031-01-01'`,
		`UPDATE orders SET delivered_at = delivered_at - interval '20 years' WHERE delivered_at >= '2031-01-01'`,
	} {
		if _, err := h.Pool.Exec(ctxBG(), q); err != nil {
			t.Fatal(err)
		}
	}
}

// prizeOn **المسابقةُ مفعّلةٌ بقيمها الافتراضيّة** — والإطلاقُ ٢٠٢٦-١٢-٠١.
func prizeOn(h *Harness) {
	h.Setting("prizes.enabled", "true")
	h.Setting("prizes.launch_on", `"2026-12-01"`)
	h.Setting("prizes.winners", "5")
	h.Setting("prizes.min_points", "80")
	h.Setting("prizes.points_checkin", "1")
	h.Setting("prizes.points_order", "10")
	h.Setting("prizes.points_invite_signup", "5")
	h.Setting("prizes.points_invite_first_order", "10")
	h.Setting("prizes.points_follow_facebook", "5")
	h.Setting("prizes.points_follow_instagram", "5")
}

// prizeCheckins **يزرع حضوراً بلحظاته** — كما يكتبه `CheckIn`.
func prizeCheckins(t *testing.T, h *Harness, u *User, at ...time.Time) {
	t.Helper()
	for _, a := range at {
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO prize_checkins (user_id, day, created_at) VALUES ($1::uuid, $2::date, $3)`,
			u.ID, prizes.DamascusDay(a), a); err != nil {
			t.Fatal(err)
		}
	}
}

// prizeDeliveredAt **طلبٌ حقيقيٌّ يُسلَّم ثمّ تُنقل لحظةُ تسليمه** إلى شهر الفحص.
func prizeDeliveredAt(t *testing.T, h *Harness, cust *User, item *Item, at time.Time) string {
	t.Helper()
	oid := mpsPlace(t, h, cust, item, "")
	deliverOrder(t, h, oid, h.driverOf(oid))
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET delivered_at = $2 WHERE id = $1::uuid`, oid, at); err != nil {
		t.Fatal(err)
	}
	return oid
}

func prizeMe(t *testing.T, h *Harness, u *User) map[string]any {
	t.Helper()
	r := h.GET("/api/v1/prizes/me", u.Token)
	if r.Code >= 400 {
		t.Fatalf("حالي: %s", r)
	}
	return r.JSON()
}

// TestPRZ06_PointsCountAndCancel **النقاطُ من مصادرها، وإلغاءُ التسليم يُسقطها.**
//
// شباط ٢٠٣١ (حسمُه ٢٧ ١٢:٠٠): أ يسجّل حضوره (١، والثاني باليوم نفسِه لا يضيف) ويتابع فيسبوك
// (٥، والثانية لا تضيف) ويُسلَّم له طلب (١٠) ويدعو ب (٥) ويُسلَّم لب أوّلُ طلب (١٠ لأ و١٠ لب)
// ⇒ أ = ٣١. **وحضورٌ بعد الحسم للشهر التالي**، **وإلغاءُ تسليم طلب أ ثمّ طلب ب ⇒ ٢١ ثمّ ١١.**
// **وردُّ الزبون لا يحمل اسمَ غيره ولا معرّفَه.**
func TestPRZ06_PointsCountAndCancel(t *testing.T) {
	h := New(t)
	treasury(t, h)
	mpsSettings(h)
	mpsNoCampaign(t, h)
	prizeClean(t, h)
	prizeOn(h)
	p := h.API.Prizes()
	t.Cleanup(func() { p.SetClock(nil); prizeClean(t, h) })

	item := h.NewItem(1000)
	a, b := h.Customer(), h.Customer()
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO referrals (invitee_id, inviter_id, rank, created_at) VALUES ($1::uuid, $2::uuid, 1, $3)`,
		b.ID, a.ID, time.Date(2031, 2, 6, 10, 0, 0, 0, offers.Damascus)); err != nil {
		t.Fatal(err)
	}
	oa := prizeDeliveredAt(t, h, a, item, time.Date(2031, 2, 10, 10, 0, 0, 0, offers.Damascus))
	ob := prizeDeliveredAt(t, h, b, item, time.Date(2031, 2, 12, 10, 0, 0, 0, offers.Damascus))

	p.SetClock(func() time.Time { return time.Date(2031, 2, 5, 9, 0, 0, 0, offers.Damascus) })
	for i := 0; i < 2; i++ {
		if r := h.POST("/api/v1/prizes/checkin", a.Token, map[string]any{}); r.Code >= 400 || r.JSON()["checked_in_today"] != true {
			t.Fatalf("الحضور: %s", r)
		}
		if r := h.POST("/api/v1/prizes/follow", a.Token, map[string]any{"network": "facebook"}); r.Code >= 400 {
			t.Fatalf("المتابعة: %s", r)
		}
	}
	if r := h.POST("/api/v1/prizes/follow", a.Token, map[string]any{"network": "tiktok"}); r.Code < 400 {
		t.Fatalf("**شبكةٌ مجهولةٌ قُبلت**: %s", r)
	}

	p.SetClock(func() time.Time { return time.Date(2031, 2, 20, 9, 0, 0, 0, offers.Damascus) })
	me := prizeMe(t, h, a)
	if me["month"] != "2031-02" || me["points"] != float64(31) || me["to_qualify"] != float64(49) ||
		me["qualified"] != false || me["checked_in_today"] != false || me["followed_facebook"] != true ||
		me["followed_instagram"] != false || me["rank"] == float64(0) {
		t.Fatalf("حالُ أ: %v — والمنتظَر ٣١ نقطة وباقي ٤٩", me)
	}
	raw := string(h.GET("/api/v1/prizes/me", a.Token).Body)
	if strings.Contains(raw, b.ID) || (b.Name != "" && strings.Contains(raw, b.Name)) {
		t.Fatalf("**ردُّ الزبون يحمل غيرَه**: %s", raw)
	}
	if mb := prizeMe(t, h, b); mb["points"] != float64(10) {
		t.Fatalf("نقاطُ ب %v — والمنتظَر ١٠ (طلبُه)", mb["points"])
	}

	// **وحضورٌ بعد الحسم (٢٧ شباط ١٢:٠٠) لآذار.**
	p.SetClock(func() time.Time { return time.Date(2031, 2, 27, 13, 0, 0, 0, offers.Damascus) })
	if r := h.POST("/api/v1/prizes/checkin", a.Token, map[string]any{}); r.Code >= 400 ||
		r.JSON()["month"] != "2031-03" || r.JSON()["points"] != float64(1) {
		t.Fatalf("حضورٌ بعد الحسم: %s — والمنتظَر شهر ٢٠٣١-٠٣ ونقطة", r)
	}

	admin := h.NewUser("admin")
	board := func() map[string]float64 {
		r := h.GET("/api/v1/admin/prizes/leaderboard?month=2031-02", admin.Token)
		if r.Code >= 400 {
			t.Fatalf("ترتيبُ اللوحة: %s", r)
		}
		out := map[string]float64{}
		list, _ := r.JSON()["standings"].([]any)
		for _, x := range list {
			row, _ := x.(map[string]any)
			id, _ := row["user_id"].(string)
			out[id], _ = row["points"].(float64)
		}
		return out
	}
	if got := board(); got[a.ID] != 31 || got[b.ID] != 10 {
		t.Fatalf("اللوحة: أ %v ب %v — والمنتظَر ٣١ و١٠", got[a.ID], got[b.ID])
	}
	if r := h.GET("/api/v1/admin/prizes/leaderboard", a.Token); r.Code != http.StatusForbidden {
		t.Fatalf("**زبونٌ يقرأ الترتيب**: %d", r.Code)
	}

	if r := undeliver(h, admin, oa, "سُلّم خطأً"); r.Code >= 400 {
		t.Fatalf("إلغاءُ تسليم أ: %s", r)
	}
	if got := board(); got[a.ID] != 21 {
		t.Fatalf("أ بعد إلغاء طلبه %v — والمنتظَر ٢١", got[a.ID])
	}
	if r := undeliver(h, admin, ob, "سُلّم خطأً"); r.Code >= 400 {
		t.Fatalf("إلغاءُ تسليم ب: %s", r)
	}
	if got := board(); got[a.ID] != 11 || got[b.ID] != 0 {
		t.Fatalf("بعد إلغاء طلب ب: أ %v ب %v — والمنتظَر ١١ و٠ (أوّلُ طلبٍ للمدعوّ سقط)", got[a.ID], got[b.ID])
	}
}

// TestPRZ09_AnnounceOnceTieThresholdExclusion **الإعلانُ مرّةً واحدةً آخرَ الشهر.**
//
// شباط ٢٠٣١ والحضورُ بعشرين نقطة: س رابحُ كانون الثاني (٢٠٠، مستثنى) · ب ١٠٠ بلغها ٨ شباط ·
// أ ١٠٠ بلغها ١٠ شباط · ج ٨٠ تماماً · د ٦٠ · هـ ٨٠ لكنّ آخرَ حضوره ٢٧ شباط ١٣:٠٠ بعد الحسم
// (فله ٦٠ في شباط). ⇒ ب ١٥٠٠ · أ ١٢٥٠ · ج ١٠٠٠، ولا أحدَ غيرهم. **و٢٨ شباط ١١:٥٩ لا إعلان،
// و١٢:٠٥ ثلاثةُ إعلاناتٍ متزامنةٍ يقع منها واحد.**
func TestPRZ09_AnnounceOnceTieThresholdExclusion(t *testing.T) {
	h := New(t)
	prizeClean(t, h)
	prizeOn(h)
	h.Setting("prizes.points_checkin", "20")
	p := h.API.Prizes()
	t.Cleanup(func() { p.SetClock(nil); prizeClean(t, h) })

	d := func(day, hour int) time.Time { return time.Date(2031, 2, day, hour, 0, 0, 0, offers.Damascus) }
	s, a, b, c, dd, e := h.Customer(), h.Customer(), h.Customer(), h.Customer(), h.Customer(), h.Customer()
	prizeCheckins(t, h, s, d(1, 9), d(2, 9), d(3, 9), d(4, 9), d(5, 9), d(6, 9), d(7, 9), d(8, 9), d(9, 9), d(10, 9))
	prizeCheckins(t, h, b, d(4, 9), d(5, 9), d(6, 9), d(7, 9), d(8, 9))
	prizeCheckins(t, h, a, d(6, 9), d(7, 9), d(8, 9), d(9, 9), d(10, 9))
	prizeCheckins(t, h, c, d(1, 9), d(2, 9), d(3, 9), d(4, 9))
	prizeCheckins(t, h, dd, d(1, 9), d(2, 9), d(3, 9))
	prizeCheckins(t, h, e, d(1, 9), d(2, 9), d(3, 9), d(27, 13))
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO prize_draws (month, drawn_at) VALUES ('2031-01-01', '2031-01-31 09:00Z');
		`); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO prize_winners (month, rank, user_id, points, amount) VALUES ('2031-01-01', 1, $1::uuid, 300, 1500)`,
		s.ID); err != nil {
		t.Fatal(err)
	}

	p.SetClock(func() time.Time { return d(28, 11).Add(59 * time.Minute) })
	if w, err := p.AnnounceDue(ctxBG()); err != nil || len(w) != 0 {
		t.Fatalf("**إعلانٌ قبل موعده**: %v %v", w, err)
	}
	p.SetClock(func() time.Time { return d(28, 12).Add(5 * time.Minute) })
	var wg sync.WaitGroup
	results := make([]int, 3)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w, err := p.AnnounceDue(ctxBG())
			if err != nil {
				t.Errorf("الإعلان: %v", err)
			}
			results[i] = len(w)
		}(i)
	}
	wg.Wait()
	done := 0
	for _, n := range results {
		if n > 0 {
			done++
			if n != 3 {
				t.Fatalf("الرابحون %d — والمنتظَر 3 (ب أ ج)", n)
			}
		}
	}
	if done != 1 {
		t.Fatalf("**أُعلن %d مرّة** — والمنتظَر مرّةً واحدة", done)
	}
	if w, _ := p.AnnounceDue(ctxBG()); len(w) != 0 {
		t.Fatal("**إعلانٌ ثانٍ للشهر نفسِه**")
	}

	want := map[int]struct {
		user   string
		amount int64
	}{1: {b.ID, 1500}, 2: {a.ID, 1250}, 3: {c.ID, 1000}}
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT rank, user_id::text, points, amount FROM prize_winners WHERE month = '2031-02-01' ORDER BY rank`)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for rows.Next() {
		var rank, pts int
		var uid string
		var amt int64
		if err := rows.Scan(&rank, &uid, &pts, &amt); err != nil {
			t.Fatal(err)
		}
		n++
		if w := want[rank]; w.user != uid || w.amount != amt {
			t.Errorf("المرتبة %d: %s بـ %d — والمنتظَر %s بـ %d", rank, uid, amt, w.user, w.amount)
		}
	}
	rows.Close()
	if n != 3 {
		t.Fatalf("رابحو شباط %d — والمنتظَر 3", n)
	}
	for _, w := range want {
		if got := prizeBalance(t, h, w.user); got != w.amount {
			t.Errorf("رصيدُ الرابح %d — والمنتظَر %d", got, w.amount)
		}
		assertPrizeLedger(t, h, w.user)
	}
	var exp time.Time
	if err := h.Pool.QueryRow(ctxBG(), `SELECT expires_at FROM prize_balances WHERE user_id = $1::uuid`, b.ID).Scan(&exp); err != nil ||
		!exp.Equal(time.Date(2031, 3, 31, 23, 59, 59, 0, offers.Damascus)) {
		t.Fatalf("انتهاءُ الجائزة %v (%v) — والمنتظَر آخرَ آذار", exp, err)
	}

	// **وحالُ الرابح: نتيجتُه هو ومستثنًى من آذار** — ولا أسماءَ غيره.
	me := prizeMe(t, h, a)
	if me["last_month"] != "2031-02" || me["last_rank"] != float64(2) || me["last_amount"] != float64(1250) ||
		me["excluded"] != true || me["month"] != "2031-03" {
		t.Fatalf("حالُ أ بعد الإعلان: %v", me)
	}
	if raw := string(h.GET("/api/v1/prizes/me", a.Token).Body); strings.Contains(raw, b.ID) {
		t.Fatalf("**ردُّ الزبون يحمل الرابحَ الآخر**: %s", raw)
	}
	// **وحضورُ هـ بعد حسم شباط في آذار.**
	admin := h.NewUser("admin")
	if r := h.GET("/api/v1/admin/prizes/leaderboard?month=2031-03", admin.Token); r.Code >= 400 ||
		!strings.Contains(string(r.Body), e.ID) {
		t.Fatalf("ترتيبُ آذار: %s", r)
	}
	if r := h.GET("/api/v1/admin/prizes/draws", admin.Token); r.Code >= 400 || !strings.Contains(string(r.Body), b.ID) {
		t.Fatalf("نتائجُ اللوحة: %s", r)
	}
	if r := h.GET("/api/v1/admin/prizes/draws", h.Customer().Token); r.Code != http.StatusForbidden {
		t.Fatalf("**زبونٌ يقرأ نتائجَ اللوحة**: %d", r.Code)
	}
}

// TestPRZ10_NotStartedRejects **قبل الإطلاق لا حضورَ ولا متابعة** — فلا تضيع متابعةُ العمر.
func TestPRZ10_NotStartedRejects(t *testing.T) {
	h := New(t)
	prizeOn(h)
	p := h.API.Prizes()
	t.Cleanup(func() { p.SetClock(nil) })
	p.SetClock(func() time.Time { return time.Date(2026, 11, 30, 10, 0, 0, 0, offers.Damascus) })
	cust := h.Customer()
	if r := h.POST("/api/v1/prizes/checkin", cust.Token, map[string]any{}); r.Err() != "prizes_not_started" {
		t.Fatalf("حضورٌ قبل الإطلاق: %s", r)
	}
	if r := h.POST("/api/v1/prizes/follow", cust.Token, map[string]any{"network": "instagram"}); r.Err() != "prizes_not_started" {
		t.Fatalf("متابعةٌ قبل الإطلاق: %s", r)
	}
	if me := prizeMe(t, h, cust); me["started"] != false || me["month"] != "2026-12" {
		t.Fatalf("حالي قبل الإطلاق: %v", me)
	}
}

// TestPRZ07_RemindOnceThenExpire **«بينتهي بعد ٣ أيام» مرّةً، ثمّ ينتهي آخرَ الشهر الذي يلي المنح.**
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
	p.SetClock(func() time.Time { return time.Date(2027, 2, 24, 23, 0, 0, 0, offers.Damascus) })
	if _, err := p.RemindDue(ctxBG()); err != nil || reminded() {
		t.Fatalf("**تذكيرٌ قبل الأيّام الثلاثة** (%v)", err)
	}
	p.SetClock(func() time.Time { return time.Date(2027, 2, 26, 9, 0, 0, 0, offers.Damascus) })
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

	p.SetClock(func() time.Time { return time.Date(2027, 2, 28, 23, 59, 0, 0, offers.Damascus) })
	_, _ = p.ExpireDue(ctxBG())
	if b := prizeBalance(t, h, cust.ID); b != 750 {
		t.Fatalf("**انتهى قبل آخر الشهر**: %d", b)
	}
	p.SetClock(func() time.Time { return time.Date(2027, 3, 1, 0, 0, 1, 0, offers.Damascus) })
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

// TestPRZ_StorePicker **«خلّيه متجر الجوائز» من اللوحة** (٢٠٢٦-١١-٢٤) — يُختار متجرٌ فيختفي من السوق،
// **والثاني يُردّ بـ`prize_store_taken`**، والإرجاعُ يطفئه. والزبونُ لا يختار (والماليّةُ تختار: العروضُ قسمُها).
func TestPRZ_StorePicker(t *testing.T) {
	h := New(t)
	f := h.Factory()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET status = 'inactive' WHERE status = 'prize'`); err != nil {
		t.Fatal(err)
	}
	a, b := f.Merchant(), f.Merchant()
	h.NewItemFor(a, 1000)
	admin := h.NewUser("admin")
	const url = "/api/v1/admin/prizes/store"
	if r := h.GET(url, admin.Token); r.Code != http.StatusOK || r.JSON()["store"] != nil {
		t.Fatalf("قبل الاختيار: %s", r)
	}
	if r := h.Call("PUT", url, h.NewUser("customer").Token, map[string]any{"merchant_id": a.ID}, nil); r.Code != http.StatusForbidden {
		t.Fatalf("**زبونٌ اختار**: %d", r.Code)
	}
	r := h.Call("PUT", url, admin.Token, map[string]any{"merchant_id": a.ID}, nil)
	st, _ := r.JSON()["store"].(map[string]any)
	if r.Code != http.StatusOK || st == nil || st["id"] != a.ID || st["items"] != float64(1) {
		t.Fatalf("الاختيار: %s", r)
	}
	if r := h.Call("PUT", url, admin.Token, map[string]any{"merchant_id": b.ID}, nil); r.Code != http.StatusConflict ||
		!strings.Contains(string(r.Body), "prize_store_taken") {
		t.Fatalf("**متجرٌ ثانٍ قُبل**: %s", r)
	}
	if r := h.Call("PUT", url, admin.Token, map[string]any{"merchant_id": ""}, nil); r.Code != http.StatusOK || r.JSON()["store"] != nil {
		t.Fatalf("الإرجاع: %s", r)
	}
	var status string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT status FROM merchants WHERE id = $1::uuid`, a.ID).Scan(&status)
	if status != "inactive" {
		t.Fatalf("**بعد الإرجاع**: %s", status)
	}
}
