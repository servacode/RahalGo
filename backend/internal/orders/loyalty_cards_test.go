package orders_test

// ══════════════════════════════════════════════════════════════════════
// **بطاقة رحّال غو وكرت اليوم** — قرارُ المالك ٢٠٢٦-١١-٢٧ (`loyalty_cards.go`)
// ══════════════════════════════════════════════════════════════════════
//
// الختمُ بالتسليم وحدَه وبمفتاح الطلب، وتصحيحُ التسليم يُسقطه، والجائزةُ بعد امتلاء البطاقة
// **والكابتنُ بأجرته كاملةً والخزينةُ تتحمّل** (`FI-06`)، والسقف، وعودةُ الجائزة بالإلغاء،
// **وإعفاءٌ واحدٌ مع عرض التوصيل المجانيّ**، وكرتُ اليوم بحدّ يومه وإغلاق المنصّة.

import (
	"context"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// loyaltySetting **يضبط إعداداً للاختبار ويعيده بعده.**
func loyaltySetting(t *testing.T, f *fixture, key, value string) {
	t.Helper()
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES ($1, $2::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, key, value); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM app_settings WHERE key = $1`, key) })
}

// loyaltyFixture **زبونٌ بصنفٍ وأجرة ٥٠٠٠، والبطاقتان مطفأتان حتّى يُشغّلهما الاختبار.**
func loyaltyFixture(t *testing.T) (*fixture, string) {
	t.Helper()
	f := setup(t, "pending", 100_000, 0, 0)
	f.verifyWhatsApp(t)
	f.armTreasury(t)
	fundTestTreasury(t)
	itemID := f.armItem(t)
	setDeliveryFee(t, f, "5000")
	// **ولا عرضَ توصيلٍ مجانيٍّ يسبق** — عرضٌ تُرك شغّالاً يُطفأ.
	if _, err := f.pool.Exec(context.Background(), `UPDATE free_delivery_campaigns SET active = false`); err != nil {
		t.Fatal(err)
	}
	loyaltySetting(t, f, "stamp_card.enabled", "false")
	loyaltySetting(t, f, "daily_card.enabled", "false")
	return f, itemID
}

// deliverStd **يوصل الطلبَ إلى الباب ثمّ يسلّمه بيد السائق.**
func (f *fixture) deliverStd(t *testing.T, id string) {
	t.Helper()
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET driver_id = $2, status = 'at_dropoff' WHERE id = $1`, id, f.driver); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, id, "delivered", ""); err != nil {
		t.Fatalf("التسليم: %v", err)
	}
}

func (f *fixture) loyalty(t *testing.T) orders.StampCardView {
	t.Helper()
	v, err := f.svc.CustomerLoyalty(context.Background(), f.customer)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func (f *fixture) rewardCard(t *testing.T, orderID string) string {
	t.Helper()
	var card string
	_ = f.pool.QueryRow(context.Background(),
		`SELECT card FROM loyalty_rewards WHERE order_id = $1`, orderID).Scan(&card)
	return card
}

// TestLoyalty_StampOnDeliveryOnly **الختمُ بالتسليم وحدَه، مرّةً للطلب، ويسقط بتصحيح التسليم.**
func TestLoyalty_StampOnDeliveryOnly(t *testing.T) {
	f, itemID := loyaltyFixture(t)
	ctx := context.Background()
	loyaltySetting(t, f, "stamp_card.enabled", "true")
	loyaltySetting(t, f, "stamp_card.size", "3")

	o := f.createStd(t, itemID)
	if v := f.loyalty(t); v.Stamps != 0 || !v.Enabled || v.Size != 3 {
		t.Fatalf("قبل التسليم: %+v — والمنتظَر 0 أختام", v)
	}
	// **ملغًى لا يُختم.**
	c := f.createStd(t, itemID)
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'cancelled' WHERE id = $1`, c.ID); err != nil {
		t.Fatal(err)
	}
	f.deliverStd(t, o.ID)
	v := f.loyalty(t)
	if v.Stamps != 1 || len(v.Recent) != 1 || v.Recent[0].OrderID != o.ID {
		t.Fatalf("بعد التسليم: %+v — والمنتظَر ختمٌ واحدٌ لطلبه", v)
	}
	// **ومرّةً للطلب** — رجوعٌ للتسليم ثانيةً لا يختم مرّتين.
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'at_dropoff' WHERE id = $1`, o.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'delivered' WHERE id = $1`, o.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM loyalty_stamps WHERE order_id = $1`, o.ID).Scan(&n)
	if n != 1 || f.loyalty(t).Stamps != 1 {
		t.Fatalf("**خُتم الطلبُ %d مرّة**", n)
	}
	// **وتصحيحُ التسليم/الاسترجاعُ يُسقط الختم.**
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'refunded' WHERE id = $1`, o.ID); err != nil {
		t.Fatal(err)
	}
	if v := f.loyalty(t); v.Stamps != 0 {
		t.Fatalf("بعد الاسترجاع بقي %d ختم", v.Stamps)
	}
	// **والبطاقةُ المطفأةُ لا تختم.**
	loyaltySetting(t, f, "stamp_card.enabled", "false")
	o2 := f.createStd(t, itemID)
	f.deliverStd(t, o2.ID)
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM loyalty_stamps WHERE order_id = $1`, o2.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("**البطاقةُ مطفأةٌ وخُتم الطلب**")
	}
}

// TestLoyalty_RewardCaptainPaidFull **الثالثةُ بعد بطاقةٍ من ٢ مجانيّة — والكابتنُ بأجرته والحفظُ سليم.**
func TestLoyalty_RewardCaptainPaidFull(t *testing.T) {
	f, itemID := loyaltyFixture(t)
	ctx := context.Background()
	loyaltySetting(t, f, "stamp_card.enabled", "true")
	loyaltySetting(t, f, "stamp_card.size", "2")

	base, err := fininv.Capture(ctx, f.pool, "FI-06")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		o := f.createStd(t, itemID)
		if d, _, w, _ := f.fees(t, o.ID); d != 5000 || w != 0 {
			t.Fatalf("الطلب %d قبل امتلاء البطاقة: يدفع %d · المعفى %d", i+1, d, w)
		}
		f.deliverStd(t, o.ID)
	}
	if v := f.loyalty(t); v.Stamps != 2 || !v.FreeReady {
		t.Fatalf("البطاقةُ ممتلئة: %+v", v)
	}
	if w, src := f.svc.FreeDeliveryPreviewSource(ctx, f.customer, orders.FreeKindStandard, 5000); w != 5000 || src != "stamp" {
		t.Fatalf("التسعيرة: %d / %q — والمنتظَر 5000 / stamp", w, src)
	}
	free := f.createStd(t, itemID)
	if d, drv, w, _ := f.fees(t, free.ID); d != 0 || drv != 5000 || w != 5000 {
		t.Fatalf("الجائزة: يدفع %d · السائقُ %d · المعفى %d — والمنتظَر 0 · 5000 · 5000", d, drv, w)
	}
	if c := f.rewardCard(t, free.ID); c != orders.CardStamp {
		t.Fatalf("جائزةُ الطلب %q — والمنتظَر stamp", c)
	}
	if v := f.loyalty(t); v.FreeReady || !v.RewardPending {
		t.Fatalf("الجائزةُ في الطريق: %+v", v)
	}
	before := f.balance(t, f.driver)
	f.deliverStd(t, free.ID)
	if got := f.balance(t, f.driver) - before; got != 5000 {
		t.Fatalf("أجرُ الكابتن %d — والمنتظَر 5000 كاملاً", got)
	}
	after, err := fininv.Run(ctx, f.pool, "FI-06")
	if err != nil {
		t.Fatal(err)
	}
	if fresh := base.New(after); len(fresh) > 0 {
		t.Fatalf("الحفظُ خُرق بجائزة البطاقة: %v", fresh)
	}
	// **وبعد التسليم تعود فارغة — وطلبُ الجائزة لا يُختم.**
	if v := f.loyalty(t); v.Stamps != 0 || v.FreeReady || v.RewardPending {
		t.Fatalf("بعد الجائزة: %+v — والمنتظَر بطاقةٌ فارغة", v)
	}
	next := f.createStd(t, itemID)
	if d, _, w, _ := f.fees(t, next.ID); d != 5000 || w != 0 {
		t.Fatalf("**جائزةٌ ثانيةٌ بلا أختام**: يدفع %d · المعفى %d", d, w)
	}
}

// TestLoyalty_CapAndCancelReturns **السقفُ يُعفي جزءاً، والإلغاءُ يعيد الجائزة.**
func TestLoyalty_CapAndCancelReturns(t *testing.T) {
	f, itemID := loyaltyFixture(t)
	ctx := context.Background()
	loyaltySetting(t, f, "stamp_card.enabled", "true")
	loyaltySetting(t, f, "stamp_card.size", "2")
	loyaltySetting(t, f, "stamp_card.max_free_fee", "2000")
	for i := 0; i < 2; i++ {
		o := f.createStd(t, itemID)
		f.deliverStd(t, o.ID)
	}
	if v := f.loyalty(t); v.FreeCap != 2000 {
		t.Fatalf("السقف %d — والمنتظَر 2000", v.FreeCap)
	}
	o := f.createStd(t, itemID)
	if d, drv, w, _ := f.fees(t, o.ID); d != 3000 || w != 2000 || drv != 5000 {
		t.Fatalf("بالسقف: يدفع %d · المعفى %d · السائق %d — والمنتظَر 3000 · 2000 · 5000", d, w, drv)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'cancelled' WHERE id = $1`, o.ID); err != nil {
		t.Fatal(err)
	}
	if v := f.loyalty(t); !v.FreeReady {
		t.Fatalf("**الإلغاءُ لم يُعد الجائزة**: %+v", v)
	}
	again := f.createStd(t, itemID)
	if d, _, w, _ := f.fees(t, again.ID); d != 3000 || w != 2000 {
		t.Fatalf("بعد الإلغاء: يدفع %d · المعفى %d", d, w)
	}
}

// TestLoyalty_CampaignNoDoubleWaive **العرضُ والبطاقة معاً: إعفاءٌ واحد، الأكبر.**
func TestLoyalty_CampaignNoDoubleWaive(t *testing.T) {
	f, itemID := loyaltyFixture(t)
	ctx := context.Background()
	loyaltySetting(t, f, "stamp_card.enabled", "true")
	loyaltySetting(t, f, "stamp_card.size", "2")
	for i := 0; i < 2; i++ {
		o := f.createStd(t, itemID)
		f.deliverStd(t, o.ID)
	}
	// **عرضٌ بسقف ١٠٠٠ والبطاقةُ كاملة ⇒ البطاقة، والعرضُ لا يُستهلك.**
	small := int64(1000)
	armFreeCampaign(t, []string{orders.FreeKindStandard}, &small)
	o := f.createStd(t, itemID)
	if d, _, w, _ := f.fees(t, o.ID); d != 0 || w != 5000 {
		t.Fatalf("الأكبر: يدفع %d · المعفى %d — والمنتظَر 0 · 5000 (لا 6000)", d, w)
	}
	var claims int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM free_delivery_claims WHERE order_id = $1`, o.ID).Scan(&claims)
	if claims != 0 || f.rewardCard(t, o.ID) != orders.CardStamp {
		t.Fatalf("**استُهلك العرضُ مع البطاقة** (%d)", claims)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'cancelled' WHERE id = $1`, o.ID); err != nil {
		t.Fatal(err)
	}
	// **وعرضٌ كاملٌ يساوي البطاقة ⇒ العرض، وتبقى جائزةُ البطاقة لصاحبها.**
	armFreeCampaign(t, []string{orders.FreeKindStandard}, nil)
	o2 := f.createStd(t, itemID)
	if d, _, w, _ := f.fees(t, o2.ID); d != 0 || w != 5000 {
		t.Fatalf("التساوي: يدفع %d · المعفى %d", d, w)
	}
	if c := f.rewardCard(t, o2.ID); c != "" {
		t.Fatalf("**استُهلكت جائزةُ البطاقة (%s) والعرضُ هو المطبَّق**", c)
	}
	if v := f.loyalty(t); !v.FreeReady {
		t.Fatalf("جائزةُ البطاقة لم تبقَ: %+v", v)
	}
}

// fakeDayHours **آلةُ أوقاتٍ تقول إنّ المنصّة أُغلقت عند `close`** — لكرت اليوم.
type fakeDayHours struct{ close time.Time }

func (fakeDayHours) ZoneOpen(context.Context, dbtx.Querier, string) (bool, *time.Time, error) {
	return true, nil, nil
}
func (fakeDayHours) ZoneNextCloseAt(context.Context, dbtx.Querier, string) (*time.Time, error) {
	return nil, nil
}
func (h fakeDayHours) WorkDayClose(_ context.Context, _ dbtx.Querier, _, _ time.Time) (time.Time, bool, error) {
	return h.close, true, nil
}

// TestLoyalty_DailyCard **كرتُ اليوم: ٢ باليوم ⇒ التاليةُ مجانيّة، مستقلٌّ عن بطاقة رحّال غو،
// ويبدأ اليومُ الجديدُ فارغاً، وتسقط الجائزةُ بإغلاق المنصّة.**
func TestLoyalty_DailyCard(t *testing.T) {
	f, itemID := loyaltyFixture(t)
	ctx := context.Background()
	loyaltySetting(t, f, "stamp_card.enabled", "true")
	loyaltySetting(t, f, "stamp_card.size", "10")
	loyaltySetting(t, f, "daily_card.enabled", "true")
	loyaltySetting(t, f, "daily_card.size", "2")

	var stamped []string
	for i := 0; i < 2; i++ {
		o := f.createStd(t, itemID)
		f.deliverStd(t, o.ID)
		stamped = append(stamped, o.ID)
	}
	v := f.loyalty(t)
	if !v.Daily.Open || v.Daily.Stamps != 2 || !v.Daily.FreeReady || v.Stamps != 2 || v.FreeReady {
		t.Fatalf("بعد توصيلتين: %+v — والمنتظَر كرتُ اليوم ممتلئٌ وبطاقةُ رحّال غو ٢ من ١٠", v)
	}
	if !v.Daily.RenewsAt.After(v.Daily.DayStart) {
		t.Fatalf("موعدُ التجدّد %v قبل بداية اليوم %v", v.Daily.RenewsAt, v.Daily.DayStart)
	}

	// **وتسقط بالإغلاق** — منصّةٌ أُغلقت قبل دقيقة.
	f.svc.SetZoneHours(fakeDayHours{close: time.Now().Add(-time.Minute)})
	if w, _ := f.svc.FreeDeliveryPreviewSource(ctx, f.customer, orders.FreeKindStandard, 5000); w != 0 {
		t.Fatalf("**جائزةُ يومٍ أُغلق ما زالت**: %d", w)
	}
	if v := f.loyalty(t); v.Daily.Open || v.Daily.FreeReady {
		t.Fatalf("بعد الإغلاق: %+v", v.Daily)
	}
	f.svc.SetZoneHours(fakeDayHours{close: time.Now().Add(time.Hour)})

	// **والجائزةُ في الدوام: كرتُ اليوم، والكابتنُ بأجرته.**
	base, err := fininv.Capture(ctx, f.pool, "FI-06")
	if err != nil {
		t.Fatal(err)
	}
	free := f.createStd(t, itemID)
	if d, drv, w, _ := f.fees(t, free.ID); d != 0 || drv != 5000 || w != 5000 || f.rewardCard(t, free.ID) != orders.CardDaily {
		t.Fatalf("جائزةُ اليوم: يدفع %d · السائق %d · المعفى %d · %q", d, drv, w, f.rewardCard(t, free.ID))
	}
	before := f.balance(t, f.driver)
	f.deliverStd(t, free.ID)
	if got := f.balance(t, f.driver) - before; got != 5000 {
		t.Fatalf("أجرُ الكابتن %d — والمنتظَر 5000", got)
	}
	after, err := fininv.Run(ctx, f.pool, "FI-06")
	if err != nil {
		t.Fatal(err)
	}
	if fresh := base.New(after); len(fresh) > 0 {
		t.Fatalf("الحفظُ خُرق بكرت اليوم: %v", fresh)
	}
	if v := f.loyalty(t); v.Daily.Stamps != 0 || v.Daily.FreeReady || v.Stamps != 2 {
		t.Fatalf("بعد جائزة اليوم: %+v — والمنتظَر كرتُ اليوم فارغٌ وبطاقةُ رحّال غو كما هي", v)
	}

	// **ويومٌ جديدٌ يبدأ فارغاً** — أختامُ أمس لا تُعدّ اليوم (وتبقى لبطاقة رحّال غو).
	for i := 0; i < 2; i++ {
		o := f.createStd(t, itemID)
		f.deliverStd(t, o.ID)
		stamped = append(stamped, o.ID)
	}
	if v := f.loyalty(t); !v.Daily.FreeReady {
		t.Fatalf("كرتُ اليوم لم يمتلئ ثانيةً: %+v", v.Daily)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE loyalty_stamps SET created_at = created_at - interval '1 day'
		WHERE order_id = ANY($1::uuid[])`, stamped); err != nil {
		t.Fatal(err)
	}
	if v := f.loyalty(t); v.Daily.Stamps != 0 || v.Daily.FreeReady || v.Stamps != 4 {
		t.Fatalf("يومٌ جديد: %+v — والمنتظَر كرتُ اليوم فارغٌ وبطاقةُ رحّال غو ٤", v)
	}
}
