package orders_test

// **كودُ الخصم يُصرف مرّتين بطلبين متزامنين.**
//
// # الثغرة
//
// `validatePromo` تقرأ `used_count` وتسأل `EXISTS(promo_redemptions)` —
// **على `s.db` مباشرةً، قبل أن تبدأ المعاملة** (سطر ٣٠٥). ثمّ يُدرج القيدُ
// ويُزاد العدّادُ **داخل المعاملة** (سطر ٣٦٠).
//
// **فطلبان يصلان معاً يقرآن كلاهما «لم يُستعمل بعد»** ثمّ يُدرجان — والفهرسُ
// الفريدُ على `(promo_id, order_id)` **لا يمنعهما لأنّ رقمَي الطلبين
// مختلفان**، والفهرسُ على `(promo_id, user_id)` **ليس فريداً أصلاً.**
//
// # وأثرُها مالٌ يخرج
//
//	`once_per_user`  يُستعمل مرّتين — **والقاعدةُ تقول مرّةً**
//	`max_uses`       كودٌ بقيت له مرّةٌ يُصرف عشراً — **حملةٌ بألفٍ تكلّف عشرة**
//
// **ولا يحتاج مهارة**: زرّان يُضغطان معاً، أو شبكةٌ بطيئةٌ تُعيد الإرسال.

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// armItem صنفٌ صالحٌ للطلب — **والعُدّةُ الأصليّةُ تُنشئ طلباً بالسطر لا صنفاً.**
func (f *fixture) armItem(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	var sectionID, itemID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'قسمُ سباق')
		RETURNING id`, f.merchantID).Scan(&sectionID); err != nil {
		t.Fatalf("تعذّر إنشاءُ قسم: %v", err)
	}
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, section_id, platform_section_id, name,
			price, merchant_price, available, approved)
		-- **وقسمُ السوق إلزاميّ** (المهاجرة ٠١١٨): صنفٌ بلا قسمٍ لا يراه
		-- زبونٌ إطلاقاً، **والقاعدةُ ترفضه اليوم.**
		VALUES ($1, $2, (SELECT id FROM platform_sections ORDER BY sort_order LIMIT 1),
		        'صنفُ سباق', 30000, 25000, true, true)
		RETURNING id`, f.merchantID, sectionID).Scan(&itemID); err != nil {
		t.Fatalf("تعذّر إنشاءُ صنف: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `DELETE FROM menu_items WHERE id = $1`, itemID)
		_, _ = f.pool.Exec(c, `DELETE FROM menu_sections WHERE id = $1`, sectionID)
	})
	return itemID
}

// verifyWhatsApp يوثّق رقمَ الزبون — **شرطُ الطلب، وحارسٌ صحيحٌ لا يُلتفّ
// عليه في الإنتاج**: بلاه تُفتح مئةُ حسابٍ بأرقامٍ تُشترى.
func (f *fixture) verifyWhatsApp(t *testing.T) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), `
		UPDATE users SET whatsapp_phone = phone::text, whatsapp_verified_at = now()
		WHERE id = $1`, f.customer); err != nil {
		t.Fatalf("تعذّر التوثيق: %v", err)
	}
}

// armPromo كودٌ بمرّةٍ واحدةٍ لكلّ زبون.
func (f *fixture) armPromo(t *testing.T, code string, oncePerUser bool, maxUses *int) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO promo_codes (code, kind, value, once_per_user, max_uses, active)
		VALUES ($1, 'fixed', 5000, $2, $3, true)
		RETURNING id`, code, oncePerUser, maxUses).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ الكود: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = f.pool.Exec(ctx, `DELETE FROM promo_redemptions WHERE promo_id = $1`, id)
		_, _ = f.pool.Exec(ctx, `DELETE FROM promo_codes WHERE id = $1`, id)
	})
	return id
}

// TestPromo_OncePerUserSurvivesConcurrency **مرّةٌ واحدةٌ تعني مرّةً واحدة.**
func TestPromo_OncePerUserSurvivesConcurrency(t *testing.T) {
	f := setup(t, "pending", 100_000, 10_000, 0)
	ctx := context.Background()
	f.verifyWhatsApp(t)
	itemID := f.armItem(t)
	promoID := f.armPromo(t, "RACE1", true, nil)

	in := orders.CreateInput{
		CustomerID:    f.customer,
		MerchantID:    f.merchantID,
		Items:         []orders.ItemInput{{MenuItemID: itemID, Qty: 1}},
		AddressText:   "سباقٌ على كود",
		Lat:           35.9528,
		Lng:           39.0079,
		PaymentMethod: "cash",
		PromoCode:     "RACE1",
	}

	// **طلبان يصلان معاً** — كزرٍّ يُضغط مرّتين أو شبكةٍ تُعيد الإرسال.
	var wg sync.WaitGroup
	made := make([]*orders.Order, 2)
	start := make(chan struct{})
	for i := range made {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			o, err := f.svc.Create(ctx, f.customer, []string{"customer"}, in, "127.0.0.1")
			if err != nil {
				t.Logf("طلب %d فشل: %v", i, err)
				return
			}
			made[i] = o
		}(i)
	}
	close(start)
	wg.Wait()

	for _, o := range made {
		if o != nil {
			t.Cleanup(func() {
				_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, o.ID)
			})
		}
	}

	// **ولا سباقَ إن لم يقع شيء**: لو فشل الطلبان لسببٍ آخر (منطقةٌ أو
	// إعداد) لَمرّ الاختبارُ وهو لم يفحص شيئاً.
	ok := 0
	for _, o := range made {
		if o != nil {
			ok++
		}
	}
	if ok == 0 {
		t.Fatal("لم ينجح طلبٌ واحد — **والاختبارُ لم يفحص السباقَ أصلاً**")
	}

	var redeemed int
	if err := f.pool.QueryRow(ctx,
		`SELECT count(*) FROM promo_redemptions WHERE promo_id = $1`, promoID).Scan(&redeemed); err != nil {
		t.Fatalf("تعذّر العدّ: %v", err)
	}
	t.Logf("نجح %d من %d طلب · صُرف الكودُ %d مرّة", ok, len(made), redeemed)
	if redeemed > 1 {
		t.Fatalf("صُرف الكودُ %d مرّاتٍ و«مرّةٌ لكلّ زبون» — **زرّان معاً يكسران القاعدة**", redeemed)
	}

	// **والعدّادُ يطابق ما وقع** — ولو زاد لَحُسبت حملةٌ أكثرَ ممّا صُرف.
	var used int
	_ = f.pool.QueryRow(ctx, `SELECT used_count FROM promo_codes WHERE id = $1`, promoID).Scan(&used)
	if used != redeemed {
		t.Errorf("العدّادُ %d والقيودُ %d — **والحسبةُ تخالف الدفتر**", used, redeemed)
	}
}

// TestPromo_MaxUsesSurvivesConcurrency **وسقفُ الاستعمال سقف.**
//
// **وهذه أثقلُ في المال**: كودُ حملةٍ سقفُه مئةٌ يُصرف مئتين، **والفرقُ
// يخرج من الخزينة.**
func TestPromo_MaxUsesSurvivesConcurrency(t *testing.T) {
	f := setup(t, "pending", 100_000, 10_000, 0)
	ctx := context.Background()
	one := 1
	f.verifyWhatsApp(t)
	itemID := f.armItem(t)
	promoID := f.armPromo(t, "RACE2", false, &one)

	in := orders.CreateInput{
		CustomerID:    f.customer,
		MerchantID:    f.merchantID,
		Items:         []orders.ItemInput{{MenuItemID: itemID, Qty: 1}},
		AddressText:   "سباقٌ على سقف",
		Lat:           35.9528,
		Lng:           39.0079,
		PaymentMethod: "cash",
		PromoCode:     "RACE2",
	}

	const racers = 4
	var wg sync.WaitGroup
	made := make([]*orders.Order, racers)
	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			// **وتباعدٌ مجهريٌّ يُقرّب السباقَ من الواقع** — لا يُلغيه.
			time.Sleep(time.Duration(i) * time.Millisecond)
			if o, err := f.svc.Create(ctx, f.customer, []string{"customer"}, in, "127.0.0.1"); err == nil {
				made[i] = o
			}
		}(i)
	}
	close(start)
	wg.Wait()

	for _, o := range made {
		if o != nil {
			t.Cleanup(func() {
				_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, o.ID)
			})
		}
	}

	ok := 0
	for _, o := range made {
		if o != nil {
			ok++
		}
	}
	if ok == 0 {
		t.Fatal("لم ينجح طلبٌ واحد — **والاختبارُ لم يفحص السباقَ أصلاً**")
	}

	var redeemed int
	_ = f.pool.QueryRow(ctx,
		`SELECT count(*) FROM promo_redemptions WHERE promo_id = $1`, promoID).Scan(&redeemed)
	t.Logf("نجح %d من %d طلب · صُرف الكودُ %d مرّة", ok, racers, redeemed)
	if redeemed > 1 {
		t.Fatalf("صُرف الكودُ %d مرّاتٍ وسقفُه واحدة — **والفرقُ يخرج من الخزينة**", redeemed)
	}
}
