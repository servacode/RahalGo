package orders_test

// ══════════════════════════════════════════════════════════════════════
// **قواعدُ لوحة الطلبات في المحرّك** — قراراتُ المالك ٢٠٢٦-١٠-٠٤ (قسمُ «الطلبات»)
// ══════════════════════════════════════════════════════════════════════
//
//	العدّاداتُ = القوائمُ المرشَّحة      البند ٣ (والمشكلة ٢٢)
//	الترتيبُ بالأولويّة                  البند ١ (والمشكلة ٧)
//	حرّاسُ الإسناد اليدويّ               البند ١٠ (والمشكلتان ١٢ و١٣)
//	لا حسمَ قبل «سلّمت البضاعة»          البند ١٤ (والمشكلة ١٤)
//	سقفُ تعويض المتجر ومرّةٌ واحدة        البند ١٣ (والمشكلة ٦)
//	القبولُ التلقائيُّ حين يفرغ المكتب    البند ٨
//
// **وكلُّ رقمٍ يُقرأ من القاعدة** — لا ممّا تُرجعه الدالّةُ عن نفسها.

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// boardOrder طلبٌ في حالٍ بعينها على متجر العُدّة — **بلا سائقٍ إلّا إن طُلب**،
// وعمرُه بالدقائق.
func (f *fixture) boardOrder(t *testing.T, status string, ageMin int, withDriver bool, extra string) string {
	t.Helper()
	ctx := context.Background()
	var id string
	drv := any(nil)
	if withDriver {
		drv = f.driver
	}
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, driver_fee, total, wallet_paid, cash_due,
			created_at, updated_at, accepted_at, dispatched_at,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
		VALUES ($1, $2, $3, $4, 'عنوان اختبار اللوحة',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 10000, 1000, 1000, 11000, 0, 11000,
			now() - make_interval(mins => $5), now() - make_interval(mins => $5),
			CASE WHEN $4 <> 'pending' THEN now() - make_interval(mins => $5) END,
			CASE WHEN $4 IN ('dispatching','preparing') THEN now() - make_interval(mins => $5) END,
			`+qaSnapSQLX()+`)
		RETURNING id`, f.customer, f.merchantID, drv, status, ageMin).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ طلب اللوحة (%s): %v", status, err)
	}
	if extra != "" {
		if _, err := f.pool.Exec(ctx, `UPDATE orders SET `+extra+` WHERE id = $1`, id); err != nil {
			t.Fatalf("تعذّر ضبطُ الطلب: %v", err)
		}
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id) })
	return id
}

// TestBoard_CountsMatchFilteredLists **كلُّ عدّادٍ = القائمةُ التي يفتحها** (البند ٣).
//
// **والمشكلة ٢٢ بعينها**: «البطاقة ٢ والقائمة ٥». فيُقاس لكلّ فلترٍ أنّ العدَّ
// وإجماليَّ القائمة المرشَّحة رقمٌ واحد، **وأنّ طلباتِ العُدّة تقع حيث يجب.**
func TestBoard_CountsMatchFilteredLists(t *testing.T) {
	f := setup(t, "pending", 10_000, 1_000, 0)
	ctx := context.Background()
	// طلبُ العُدّة نفسُه: بانتظار القبول منذ ساعة ⇒ «عالق» أيضاً.
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET driver_id = NULL,
		created_at = now() - interval '60 minutes' WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	notSent := f.boardOrder(t, "accepted", 30, false, "")
	noDriver := f.boardOrder(t, "dispatching", 30, false, "")
	freshQueue := f.boardOrder(t, "dispatching", 0, false, "")
	onWay := f.boardOrder(t, "on_the_way", 1, true, "")
	atDoor := f.boardOrder(t, "at_dropoff", 1, true, "")

	counts, err := f.svc.BoardCounts(ctx)
	if err != nil {
		t.Fatalf("BoardCounts: %v", err)
	}
	want := map[string][]string{
		orders.BoardAwaitingAccept: {f.orderID},
		orders.BoardAwaitingDriver: {noDriver, freshQueue},
		orders.BoardOnTheWay:       {onWay},
		orders.BoardAtDoor:         {atDoor},
		orders.BoardStuck:          {f.orderID, notSent, noDriver},
		orders.BoardNoDriver:       {noDriver},
	}
	for _, name := range orders.BoardFilters {
		page, err := f.svc.List(ctx, orders.ListFilter{OpenOnly: true, Board: name, PerPage: 100})
		if err != nil {
			t.Fatalf("List(%s): %v", name, err)
		}
		if page.Total != counts[name] {
			t.Errorf("**العدّادُ %q يقول %d والقائمةُ %d** — والرقمُ يجب أن يطابق ما يفتحه",
				name, counts[name], page.Total)
		}
		mine, err := f.svc.List(ctx, orders.ListFilter{OpenOnly: true, Board: name,
			MerchantID: f.merchantID, PerPage: 100})
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]bool{}
		for _, o := range mine.Orders {
			got[o.ID] = true
		}
		for _, id := range want[name] {
			if !got[id] {
				t.Errorf("الفلتر %q لم يُظهر طلباً يخصّه (%s)", name, id)
			}
		}
		if name == orders.BoardStuck && got[freshQueue] {
			t.Errorf("**طلبٌ نزل الطابورَ الآن عُدّ عالقاً**")
		}
	}
	// **وفلترٌ لا يُعرف يُردّ ولا يُتجاهَل.**
	if _, err := f.svc.List(ctx, orders.ListFilter{OpenOnly: true, Board: "nope"}); err == nil {
		t.Error("فلترٌ مجهولٌ مرّ — والقائمةُ كلُّها تُعرض كأنّها مرشَّحة")
	}
	// **والعالقُ في العدّاد هو عددُ الصندوق الأحمر** — شرطٌ واحد.
	alerts, err := f.svc.Alerts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != counts[orders.BoardStuck] {
		t.Errorf("الصندوقُ الأحمر %d والعدّادُ «عالق» %d", len(alerts), counts[orders.BoardStuck])
	}
	reasons := map[string]string{}
	for _, a := range alerts {
		reasons[a.OrderID] = a.Reason
	}
	if reasons[notSent] != orders.StuckNotSent {
		t.Errorf("المقبولُ الذي لم يُرسَل سببُه %q والمتوقّع not_sent (المشكلة ١٠)", reasons[notSent])
	}
}

// TestBoard_PriorityOrder **المُنذَرُ أوّلاً ثمّ المتأخّرُ ثمّ الأقدم** (البند ١).
//
// **والمشكلة ٧**: كان الأحدثُ أوّلاً، فينزل أخطرُ طلبٍ إلى الصفحة الثانية.
func TestBoard_PriorityOrder(t *testing.T) {
	f := setup(t, "on_the_way", 10_000, 1_000, 0)
	ctx := context.Background()
	// العُدّةُ نفسُها: في الطريق منذ نصف ساعة ⇒ «الباقي».
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET created_at = now() - interval '30 minutes'
		WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	fresh := f.boardOrder(t, "pending", 0, false, "")           // جديدٌ لم تمضِ مهلتُه ⇒ الباقي، الأحدث
	late := f.boardOrder(t, "on_the_way", 120, true, "")        // تجاوز عمرَه ⇒ متأخّر
	stuck := f.boardOrder(t, "pending", 20, false, "")          // بانتظار القبول بعد مهلته ⇒ أوّلاً
	stuckOlder := f.boardOrder(t, "dispatching", 40, false, "") // بلا سائق ⇒ أوّلاً وأقدم

	page, err := f.svc.List(ctx, orders.ListFilter{OpenOnly: true, Priority: true,
		MerchantID: f.merchantID, PerPage: 50})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, o := range page.Orders {
		got = append(got, o.ID)
	}
	want := []string{stuckOlder, stuck, late, f.orderID, fresh}
	if len(got) != len(want) {
		t.Fatalf("عددُ الطلبات %d والمتوقّع %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("**الترتيبُ ليس بالأولويّة** — الموضعُ %d: %s والمتوقّع %s", i, got[i], want[i])
		}
	}
	// **وبلا أولويّةٍ يبقى السجلُّ الأحدثَ أوّلاً** — لا يتبدّل ما لم يُطلب.
	plain, err := f.svc.List(ctx, orders.ListFilter{OpenOnly: true, MerchantID: f.merchantID, PerPage: 50})
	if err != nil {
		t.Fatal(err)
	}
	if plain.Orders[0].ID != fresh {
		t.Errorf("السجلُّ بلا أولويّةٍ لم يبدأ بالأحدث")
	}
}

// TestAssign_ServerGuards **المحرّكُ يردّ ما كانت الشاشةُ وحدَها تمنعه** (البند ١٠).
func TestAssign_ServerGuards(t *testing.T) {
	f := setup(t, "dispatching", 10_000, 1_000, 0)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET driver_id = NULL WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	ops := testdb.NewUser(t, f.pool, "operations")
	drv := testdb.NewUser(t, f.pool, "driver")

	// ── ١ · خارج الدوام ⇒ يُردّ.
	if _, err := f.svc.AssignDriver(ctx, ops, []string{"ops"}, f.orderID, drv, "إسنادٌ يدويّ"); !errors.Is(err, orders.ErrDriverOffShift) {
		t.Fatalf("**أُسند لسائقٍ خارج دوامه**: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE users SET on_shift = true WHERE id = $1`, drv); err != nil {
		t.Fatal(err)
	}
	// ── ٢ · تركه من قبل ⇒ لا يعود إليه أبداً.
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET excluded_drivers = ARRAY[$2::uuid] WHERE id = $1`,
		f.orderID, drv); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.AssignDriver(ctx, ops, []string{"ops"}, f.orderID, drv, "إسنادٌ يدويّ"); !errors.Is(err, orders.ErrDriverExcluded) {
		t.Fatalf("**عاد الطلبُ إلى من تركه**: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET excluded_drivers = '{}' WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	// ── ٣ · أخذه سائقٌ قبلُ ⇒ لا يُكتب فوقه.
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET driver_id = $2 WHERE id = $1`, f.orderID, f.driver); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.AssignDriver(ctx, ops, []string{"ops"}, f.orderID, drv, "إسنادٌ يدويّ"); !errors.Is(err, orders.ErrOrderTaken) {
		t.Fatalf("**كُتب فوق سائقٍ قبل الطلب**: %v", err)
	}
	var still string
	_ = f.pool.QueryRow(ctx, `SELECT driver_id::text FROM orders WHERE id = $1`, f.orderID).Scan(&still)
	if still != f.driver {
		t.Fatalf("تبدّل السائقُ القابل: %s", still)
	}
	// ── ٤ · وسليمٌ يُسند، والعرضُ الحيُّ لغيره يُمحى.
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET driver_id = NULL, offered_driver_id = $2,
		offer_expires_at = now() + interval '40 seconds' WHERE id = $1`, f.orderID, f.driver); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.AssignDriver(ctx, ops, []string{"ops"}, f.orderID, drv, "إسنادٌ يدويّ"); err != nil {
		t.Fatalf("الإسنادُ السليمُ رُدّ: %v", err)
	}
	var got string
	var offered *string
	_ = f.pool.QueryRow(ctx, `SELECT driver_id::text, offered_driver_id::text FROM orders WHERE id = $1`,
		f.orderID).Scan(&got, &offered)
	if got != drv {
		t.Fatalf("أُسند لغيره: %s", got)
	}
	if offered != nil {
		t.Errorf("**العرضُ الحيُّ لسائقٍ آخر بقي** بعد الإسناد: %s", *offered)
	}
}

// goodsTrip طلبٌ فشل عند الباب **ومشوارُ إرجاعه إلى المتجر قائم.**
func goodsTrip(t *testing.T) (f *fixture, owner, treasury string) {
	t.Helper()
	f = setup(t, "at_pickup", 100_000, 10_000, 0)
	owner, treasury = f.armTreasury(t)
	ctx := context.Background()
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff"} {
		if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, to, ""); err != nil {
			t.Fatalf("الانتقالُ إلى %s: %v", to, err)
		}
	}
	f.failAtDoor(t, orders.FaultCustomer, "customer_refused")
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET return_to = 'store', goods_handed_at = NULL
		WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	return f, owner, treasury
}

// TestGoods_OnlyAfterHandedAndToTripDestination **لا حسمَ قبل «سلّمت البضاعة»،
// والوجهةُ وجهةُ المشوار** (البند ١٤).
func TestGoods_OnlyAfterHandedAndToTripDestination(t *testing.T) {
	f, _, ops := goodsTrip(t)
	ctx := context.Background()
	if err := f.svc.SettleGoods(ctx, f.orderID, orders.GoodsToMerchant, ops, 0); !errors.Is(err, orders.ErrGoodsNotHanded) {
		t.Fatalf("**حُسمت بضاعةٌ ما زالت في صندوق السائق**: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET goods_handed_at = now() WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.SettleGoods(ctx, f.orderID, orders.GoodsToOffice, ops, 0); !errors.Is(err, orders.ErrGoodsWrongPlace) {
		t.Fatalf("**المشوارُ إلى المتجر والحسمُ «إلى المكتب» مرّ**: %v", err)
	}
	if err := f.svc.SettleGoods(ctx, f.orderID, orders.GoodsToMerchant, ops, 0); err != nil {
		t.Fatalf("الحسمُ الصحيحُ رُدّ: %v", err)
	}
}

// TestGoods_CompensationCappedAndOnce **تعويضُ المتجر بسقف سعر الشراء ومرّةً واحدة**
// — خطوةُ الماليّة بعد حسم العمليّات (البندان ١٢ و١٣).
func TestGoods_CompensationCappedAndOnce(t *testing.T) {
	f, owner, treasury := goodsTrip(t)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET goods_handed_at = now() WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	// **قبل الحسم لا تعويض** — لا يُعوَّض عن بضاعةٍ لم يُقَل أين هي.
	if err := f.svc.CompensateGoods(ctx, f.orderID, treasury, 5_000); !errors.Is(err, orders.ErrGoodsBadCompensation) {
		t.Fatalf("عُوّض قبل الحسم: %v", err)
	}
	// **والحسمُ مع مبلغٍ فوق السقف يُردّ كلُّه.**
	if err := f.svc.SettleGoods(ctx, f.orderID, orders.GoodsToMerchant, treasury, f.merchantCost+1); !errors.Is(err, orders.ErrGoodsCompensationCap) {
		t.Fatalf("**تعويضٌ فوق سعر الشراء مرّ مع الحسم**: %v", err)
	}
	if err := f.svc.SettleGoods(ctx, f.orderID, orders.GoodsToMerchant, treasury, 0); err != nil {
		t.Fatalf("الحسم: %v", err)
	}
	before := f.balance(t, owner)
	if err := f.svc.CompensateGoods(ctx, f.orderID, treasury, f.merchantCost+1); !errors.Is(err, orders.ErrGoodsCompensationCap) {
		t.Fatalf("**تعويضٌ فوق سعر الشراء مرّ** (٥٠٠٠٠٠ بدل ٥٠٠٠٠): %v", err)
	}
	if got := f.balance(t, owner); got != before {
		t.Fatalf("تحرّك رصيدُ المتجر مع الرفض: %d ← %d", before, got)
	}
	if err := f.svc.CompensateGoods(ctx, f.orderID, treasury, 7_500); err != nil {
		t.Fatalf("التعويضُ تحت السقف رُدّ: %v", err)
	}
	paid, expense := f.compensationOf(t, owner)
	if paid != 7_500 || expense != -7_500 {
		t.Fatalf("التعويض %d ونفقةُ الخزينة %d — والمتوقّع 7500 و-7500", paid, expense)
	}
	if err := f.svc.CompensateGoods(ctx, f.orderID, treasury, 1_000); !errors.Is(err, orders.ErrGoodsAlreadyCompensated) {
		t.Fatalf("**عُوّض مرّتين**: %v", err)
	}
}

// TestAutoAccept_WhenOfficeEmpty **القبولُ التلقائيُّ حين لا أحدَ في المكتب** (البند ٨).
//
// **والإشارةُ آخرُ ظهورٍ لمن يملك `orders.intervene`** — فيُقاس الوجهان: مكتبٌ
// فارغٌ ⇒ يُقبل بعد المهلة · وموظّفٌ حاضرٌ ⇒ ينتظر يدَه (والقبولُ الدائمُ مطفأ).
func TestAutoAccept_WhenOfficeEmpty(t *testing.T) {
	f := setup(t, "pending", 10_000, 1_000, 0)
	ctx := context.Background()
	f.setSetting(t, "orders.auto_accept_min", 0)
	f.setSetting(t, "orders.unattended_auto_accept_min", 10)
	f.setSetting(t, "orders.staff_presence_min", 5)
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET driver_id = NULL,
		created_at = now() - interval '15 minutes' WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	// **المكتبُ فارغ**: لا أحدَ ممّن يملك القبول ظهر في الدقائق الخمس.
	if _, err := f.pool.Exec(ctx, `UPDATE users SET last_seen_at = now() - interval '1 day'
		WHERE last_seen_at > now() - interval '1 hour'`); err != nil {
		t.Fatal(err)
	}
	// **وموظّفٌ حاضرٌ أوّلاً** — فلا يُقبل.
	staff := testdb.NewUser(t, f.pool, "operations")
	if _, err := f.pool.Exec(ctx, `UPDATE users SET last_seen_at = now() WHERE id = $1`, staff); err != nil {
		t.Fatal(err)
	}
	if !f.svc.StaffPresent(ctx) {
		t.Fatal("موظّفُ عمليّاتٍ ظهر الآن ولم يُعدّ حاضراً")
	}
	f.svc.SweepAutoAcceptOnce(ctx)
	if st := f.statusOf(t); st != "pending" {
		t.Fatalf("**قُبل الطلبُ والموظّفُ جالس** — %s", st)
	}
	// **ثمّ يغادر** — فيُقبل بعد المهلة.
	if _, err := f.pool.Exec(ctx, `UPDATE users SET last_seen_at = now() - interval '1 day' WHERE id = $1`, staff); err != nil {
		t.Fatal(err)
	}
	if f.svc.StaffPresent(ctx) {
		t.Fatal("المكتبُ فارغٌ وعُدّ حاضراً")
	}
	f.svc.SweepAutoAcceptOnce(ctx)
	if st := f.statusOf(t); st != "accepted" {
		t.Fatalf("**المكتبُ فارغٌ والطلبُ لم يُقبل بعد ١٥ دقيقة** — %s", st)
	}
}
