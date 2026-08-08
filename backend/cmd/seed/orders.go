package main

/*
**بذرةُ الطلبات — شاشةٌ لا تُفحص فارغة.**

(طلبُ المالك ٢٠٢٦-٠٨-٠٨: «لازم ننضّف البيانات مشان نختبر على نظافة»،
 وبياناتٌ تغطّي كلَّ الحالات.)

# لماذا

البذرةُ كانت تعطي حساباتٍ ومتاجرَ وأصنافاً — **وصفرَ طلبات.** فتُفتح لوحةُ
الطلبات فارغةً، وشاشةُ السائق فارغةً، وكشفُ المحفظة فارغاً — **ولا يُعرف
أعطبٌ هو أم لا شيءَ ليُعرض.**

# وما تصنعه

طلبٌ في كلِّ حالةٍ يمرّ بها الطلبُ فعلاً:

	pending      وصل ولم يقبله المتجرُ بعد   ← يُرى في طابور المتجر
	preparing    قُبل ويُحضَّر                ← عدّادُ التحضير يعمل
	dispatching  جاهزٌ ينتظر سائقاً           ← يُرى في طابور السائقين
	on_the_way   بيد السائق                  ← المسارُ يتحرّك في شاشة الزبون
	delivered    وصل ودُفع                   ← دفترٌ وتقييمٌ ومحفظة
	cancelled    ألغاه الزبون                ← يُرى في التاريخ
	failed       فشل التسليم                 ← بابُ التعويض والبضاعة
	rejected     رفضه المتجر                 ← مخالفةٌ محتملة

**والمالُ يتبع الحالة**: المسلَّمُ يقيّد مستحقَّ المتجر وعمولةَ المنصة،
والمدفوعُ نقداً يزيد صندوقَ السائق. **وطلبٌ مسلَّمٌ بلا قيدٍ يجعل كشفَ
الحساب يكذب.**

# ولا تُنادى أفعالُ المحرّك

**تُكتب الصفوفُ مباشرةً**: مسارُ المحرّك يتطلّب توقيتاً وسائقاً على الدوام
وقبولاً وردّاً — **وبذرةٌ تنتظر ذلك تصير اختباراً لا بذرة.**
*/

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

type orderSeed struct {
	Status   string
	Pay      string // cash | wallet
	AgoHours int
	Items    []orderItemSeed
	Note     string
	Reason   string // سببُ الإلغاء أو الرفض
}

type orderItemSeed struct {
	Name  string
	Price int64
	Qty   int
}

var orderSeeds = []orderSeed{
	{Status: "pending", Pay: "cash", AgoHours: 0, Note: "بلا بصل من فضلك",
		Items: []orderItemSeed{{"شاورما دجاج", 22000, 2}, {"بطاطا مقلية", 8000, 1}}},

	{Status: "preparing", Pay: "cash", AgoHours: 0,
		Items: []orderItemSeed{{"كباب حلبي", 68000, 1}, {"فتوش", 15000, 1}}},

	{Status: "dispatching", Pay: "wallet", AgoHours: 1,
		Items: []orderItemSeed{{"شيش طاووق", 55000, 1}}},

	{Status: "on_the_way", Pay: "cash", AgoHours: 1,
		Items: []orderItemSeed{{"بيتزا خضار", 45000, 1}, {"عصير برتقال", 9000, 2}}},

	{Status: "delivered", Pay: "cash", AgoHours: 26,
		Items: []orderItemSeed{{"حمص بالطحينة", 12000, 2}, {"متبل باذنجان", 13000, 1}}},

	{Status: "delivered", Pay: "wallet", AgoHours: 50,
		Items: []orderItemSeed{{"مشاوي مشكّلة", 90000, 1}}},

	{Status: "delivered", Pay: "cash", AgoHours: 74,
		Items: []orderItemSeed{{"تبولة", 15000, 1}, {"عرايس لحمة", 30000, 1}}},

	{Status: "cancelled", Pay: "cash", AgoHours: 30, Reason: "تأخّر التحضير",
		Items: []orderItemSeed{{"صحن مقبّلات", 25000, 1}}},

	{Status: "failed", Pay: "cash", AgoHours: 8, Reason: "الزبون لم يردّ على الهاتف",
		Items: []orderItemSeed{{"وجبة برغر", 35000, 2}}},

	{Status: "rejected", Pay: "cash", AgoHours: 12, Reason: "الصنف غير متوفّر",
		Items: []orderItemSeed{{"سمك مقلي", 80000, 1}}},
}

// seedOrders يزرع طلباً في كلّ حالة، بمالِه وأحداثِه.
func seedOrders(ctx context.Context, tx pgx.Tx) {
	var customer, driver, merchant, owner string
	if err := tx.QueryRow(ctx, `
		SELECT u.id FROM users u
		JOIN user_roles r ON r.user_id = u.id AND r.role_code = 'customer'
		ORDER BY u.created_at LIMIT 1`).Scan(&customer); err != nil {
		log.Fatalf("لا زبون في القاعدة — شغّل البذرة الأساسية أوّلاً: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT u.id FROM users u
		JOIN user_roles r ON r.user_id = u.id AND r.role_code = 'driver'
		ORDER BY u.created_at LIMIT 1`).Scan(&driver); err != nil {
		log.Fatalf("لا سائق: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT id, owner_user_id FROM merchants
		WHERE owner_user_id IS NOT NULL ORDER BY created_at LIMIT 1`).Scan(&merchant, &owner); err != nil {
		log.Fatalf("لا متجر بصاحب: %v", err)
	}

	// **والعنوانُ من عنوان الزبون إن وُجد** — فيقع داخلَ منطقةِ توصيلٍ حقيقيّة.
	var lat, lng float64
	var addr string
	if err := tx.QueryRow(ctx, `
		SELECT ST_Y(location::geometry), ST_X(location::geometry), label
		FROM user_addresses WHERE user_id = $1 ORDER BY is_default DESC LIMIT 1`,
		customer).Scan(&lat, &lng, &addr); err != nil {
		lat, lng, addr = 35.9528, 39.0079, "حي المشلب — الرقة"
	}

	const deliveryFee int64 = 10000

	// **ويُشحن الزبونُ قبل أن يدفع** — طلبٌ من المحفظة على رصيدٍ صفرٍ يرفضه
	// قيدُ القاعدة، **وهو محقّ**: لا يُخلق مالٌ ليُنفَق.
	credit(ctx, tx, customer, 300000, "topup", "", "شحنُ محفظةٍ للتجربة",
		time.Now().Add(-96*time.Hour))

	made := 0

	for _, o := range orderSeeds {
		var subtotal int64
		for _, it := range o.Items {
			subtotal += it.Price * int64(it.Qty)
		}
		total := subtotal + deliveryFee
		var walletPaid, cashDue int64
		if o.Pay == "wallet" {
			walletPaid = total
		} else {
			cashDue = total
		}

		when := time.Now().Add(-time.Duration(o.AgoHours) * time.Hour)
		var drv *string
		if hasDriver(o.Status) {
			drv = &driver
		}

		var id string
		if err := tx.QueryRow(ctx, `
			INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
				payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due,
				notes, cancel_reason, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5,
				ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography,
				$8, $9, $10, $11, $12, $13, $14, $15, $16, $16)
			RETURNING id`,
			customer, merchant, drv, o.Status, addr, lng, lat,
			o.Pay, subtotal, deliveryFee, total, walletPaid, cashDue,
			o.Note, o.Reason, when).Scan(&id); err != nil {
			log.Fatalf("طلب %s: %v", o.Status, err)
		}

		for _, it := range o.Items {
			if _, err := tx.Exec(ctx, `
				INSERT INTO order_items (order_id, name, unit_price, merchant_price, qty)
				VALUES ($1, $2, $3, $3, $4)`, id, it.Name, it.Price, it.Qty); err != nil {
				log.Fatalf("صنف %s: %v", it.Name, err)
			}
		}

		// **والمالُ يتبع الحالة** — وكشفٌ بلا قيدٍ يكذب.
		if o.Status == "delivered" {
			commission := subtotal / 10 // ١٠٪ عمولةُ المنصة
			earning := subtotal - commission
			credit(ctx, tx, owner, earning, "merchant_earning", id,
				fmt.Sprintf("مستحقُّ طلبٍ مسلَّم"), when)
			if o.Pay == "wallet" {
				credit(ctx, tx, customer, -walletPaid, "order_payment", id, "دفعُ طلبٍ من المحفظة", when)
			} else {
				// نقدٌ بيد السائق — يدخل صندوقَه.
				if _, err := tx.Exec(ctx, `
					INSERT INTO driver_cash_boxes (driver_id) VALUES ($1)
					ON CONFLICT (driver_id) DO NOTHING`, driver); err != nil {
					log.Fatalf("صندوق: %v", err)
				}
				if _, err := tx.Exec(ctx, `
					UPDATE driver_cash_boxes SET held = held + $2 WHERE driver_id = $1`,
					driver, cashDue); err != nil {
					log.Fatalf("صندوق: %v", err)
				}
				if _, err := tx.Exec(ctx, `
					INSERT INTO driver_cash_entries (driver_id, amount, kind, ref, note, created_at)
					VALUES ($1, $2, 'order_collection', $3, 'تحصيلُ طلبٍ نقداً', $4)`,
					driver, cashDue, id, when); err != nil {
					log.Fatalf("قيد صندوق: %v", err)
				}
			}
		}

		made++
	}

	seedAftermath(ctx, tx, customer, owner)
	fmt.Printf("   طُلبت %d طلبات — من «وصل ولم يُقبل» إلى «فشل التسليم»\n", made)
}

// seedAftermath ما يتبع الطلبات: تقييماتٌ وشكاوى وطلبُ سحب.
//
// **وشاشةٌ فارغةٌ لا تُفحص**: صفحةُ الشكاوى بلا شكوى، وصفحةُ السحوبات بلا
// طلب، وسمعةُ السائق بلا نجمة — **كلُّها تُقرأ «تعمل» وهي لم تُجرَّب.**
func seedAftermath(ctx context.Context, tx pgx.Tx, customer, owner string) {
	rows, err := tx.Query(ctx, `
		SELECT id FROM orders WHERE status = 'delivered' ORDER BY created_at`)
	if err != nil {
		log.Fatalf("طلبات مسلَّمة: %v", err)
	}
	var delivered []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			log.Fatalf("قراءة: %v", err)
		}
		delivered = append(delivered, id)
	}
	rows.Close()

	// ── تقييماتٌ على المسلَّم ──
	stars := []struct {
		platform, driver int
		comment          string
	}{
		{5, 5, "أكل ممتاز والتوصيل سريع"},
		{4, 5, "طيّب بس تأخّر شويّ"},
		{3, 4, "عادي"},
	}
	for i, id := range delivered {
		if i >= len(stars) {
			break
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO order_ratings (order_id, customer_id, platform_stars, driver_stars, comment)
			VALUES ($1, $2, $3, $4, $5) ON CONFLICT (order_id) DO NOTHING`,
			id, customer, stars[i].platform, stars[i].driver, stars[i].comment); err != nil {
			log.Fatalf("تقييم: %v", err)
		}
	}

	// ── شكويان: واحدةٌ مفتوحةٌ وواحدةٌ محلولةٌ بتعويض ──
	if len(delivered) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO tickets (customer_id, order_id, subject, reason, status, opened_by_customer)
			VALUES ($1, $2, 'شكوى على طلب', 'late', 'open', true)`, customer, delivered[0]); err != nil {
			log.Fatalf("شكوى مفتوحة: %v", err)
		}
	}
	if len(delivered) > 1 {
		var t string
		if err := tx.QueryRow(ctx, `
			INSERT INTO tickets (customer_id, order_id, subject, reason, status,
				opened_by_customer, resolution, compensation, resolved_at)
			VALUES ($1, $2, 'شكوى على طلب', 'missing_items', 'resolved', true,
				'عُوِّض الفرقُ في المحفظة', 5000, now())
			RETURNING id`, customer, delivered[1]).Scan(&t); err != nil {
			log.Fatalf("شكوى محلولة: %v", err)
		}
		credit(ctx, tx, customer, 5000, "compensation", t, "تعويضُ شكوى", time.Now())
	}

	// ── طلبُ سحبٍ معلَّقٌ من صاحب المتجر ──
	// **وتُعاد البذرةُ بلا انفجار**: الفهرسُ الفريدُ يمنع طلبين معلّقين —
	// **وهو محقّ** — فلا يُنشأ ثانٍ إن وُجد أوّل.
	if _, err := tx.Exec(ctx, `
		INSERT INTO payout_requests (user_id, amount, note)
		SELECT $1, 100000, 'تحويل إلى الحساب'
		WHERE NOT EXISTS (
			SELECT 1 FROM payout_requests WHERE user_id = $1 AND status = 'pending')`,
		owner); err != nil {
		log.Fatalf("طلب سحب: %v", err)
	}

	fmt.Println("   وتقييماتٌ وشكويان وطلبُ سحبٍ معلَّق")
}

func hasDriver(status string) bool {
	switch status {
	case "assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff", "delivered", "failed":
		return true
	}
	return false
}

func credit(ctx context.Context, tx pgx.Tx, user string, amount int64, kind, ref, note string, when time.Time) {
	if _, err := tx.Exec(ctx, `
		INSERT INTO wallets (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, user); err != nil {
		log.Fatalf("محفظة: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE wallets SET balance = balance + $2, updated_at = now() WHERE user_id = $1`,
		user, amount); err != nil {
		log.Fatalf("رصيد: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO wallet_transactions (user_id, amount, kind, ref, note, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`, user, amount, kind, ref, note, when); err != nil {
		log.Fatalf("قيد محفظة: %v", err)
	}
}
