package main

// الزبون — آخر أطراف الدورة.
//
//	go run ./cmd/seed -customer
//
// **ورصيدُ محفظته يُقيَّد لا يُكتب.**
//
// الرصيد في هذه المنصة **عمودٌ مشتقّ من دفتر**: قاعدةُ المشروع أن
// `wallets.balance` يساوي مجموع `wallet_transactions` دائماً، وقد فُحصت هذه
// المطابقة على كل محفظةٍ في القاعدة بعد كل تنظيف. فكتابةُ رصيدٍ ابتدائيّ بلا
// قيدٍ يقابله تُنتج **محفظةً بمالٍ لا مصدر له** — وهو أوّل ما يُكسر في تدقيقٍ
// حقيقيّ، وأسوأ منه أنه يمرّ صامتاً.
//
// فالزراعة تفعل ما يفعله المسار الحقيقي: **إيداعٌ من الإدارة** (`topup`)
// منسوبٌ إلى حساب الأدمن، كأنّ الزبون سلّم نقداً في المكتب فقُيّد له.
//
// **وعنوانان في منطقتَين مختلفتين** — لا واحد: رسمُ التوصيل يختلف بينهما
// (١٠٬٠٠٠ لمركز المدينة و١٥٬٠٠٠ للمشلب)، وعنوانٌ واحد لا يُظهر أن الرسم يتبع
// الموقع لا المتجر.

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/auth"
)

var customer = struct {
	Phone, Name, Password string
	Balance               int64
}{
	Phone:    "+963935667788",
	Name:     "سليمان الخطيب",
	Password: "Zaboon@2026",
	// يكفي لطلبٍ كبير من مطعم بيت الرقة (مشاوي مشكّلة ١٤٥٬٠٠٠ + توصيل)
	// ويبقى منه بقيّة — فيُختبر الدفع من المحفظة ثم النقص لا الصفر.
	Balance: 250000,
}

// عنوانان في منطقتَي التغطية — والإحداثيات داخل نصف قطر كلٍّ منهما.
var customerAddresses = []struct {
	Label, Text string
	Lat, Lng    float64
	Default     bool
}{
	{
		Label: "البيت",
		Text:  "شارع تل أبيض — خلف الحديقة العامة، بناء الورد، الطابق الثاني",
		Lat:   35.9530, Lng: 39.0085, // مركز المدينة — رسم ١٠٬٠٠٠
		Default: true,
	},
	{
		Label: "المحل",
		Text:  "حي المشلب — مقابل فرن الأمل، محل قطع السيارات",
		Lat:   35.9648, Lng: 39.0445, // المشلب والدرعية — رسم ١٥٬٠٠٠
		Default: false,
	},
}

func seedCustomer(ctx context.Context, tx pgx.Tx) {
	hash, err := auth.HashPassword(customer.Password)
	if err != nil {
		log.Fatal(err)
	}

	var id string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (phone, full_name, password_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (phone) DO UPDATE SET
			full_name = EXCLUDED.full_name, password_hash = EXCLUDED.password_hash
		RETURNING id`, customer.Phone, customer.Name, hash).Scan(&id); err != nil {
		log.Fatalf("customer: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_code) VALUES ($1, 'customer')
		ON CONFLICT DO NOTHING`, id); err != nil {
		log.Fatal(err)
	}

	// العناوين — والافتراضيّ محروسٌ بفهرسٍ فريد جزئيّ، فالإدراج مشروط بالغياب
	for _, a := range customerAddresses {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_addresses (user_id, label, address_text, location, is_default)
			SELECT $1, $2, $3, ST_SetSRID(ST_MakePoint($5, $4), 4326)::geography, $6
			WHERE NOT EXISTS (
				SELECT 1 FROM user_addresses WHERE user_id = $1 AND label = $2)`,
			id, a.Label, a.Text, a.Lat, a.Lng, a.Default); err != nil {
			log.Fatalf("address %s: %v", a.Label, err)
		}
	}

	// المحفظة: **قيدٌ أوّلاً ثم رصيدٌ يطابقه** — لا رصيدٌ بلا مصدر.
	var walletExists bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM wallets WHERE user_id = $1)`, id).Scan(&walletExists); err != nil {
		log.Fatal(err)
	}
	if !walletExists {
		// المُودِع هو الأدمن إن وُجد — وإلّا فارغٌ (زراعةُ الزبون وحده)
		var adminID *string
		_ = tx.QueryRow(ctx, `
			SELECT u.id FROM users u JOIN user_roles r ON r.user_id = u.id
			WHERE r.role_code = 'admin' AND u.password_hash IS NOT NULL
			ORDER BY u.created_at LIMIT 1`).Scan(&adminID)

		if _, err := tx.Exec(ctx, `
			INSERT INTO wallet_transactions (user_id, amount, kind, note, created_by)
			VALUES ($1, $2, 'topup', 'إيداع نقديّ في مكتب المنصة', $3)`,
			id, customer.Balance, adminID); err != nil {
			log.Fatalf("topup: %v", err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO wallets (user_id, balance) VALUES ($1, $2)`,
			id, customer.Balance); err != nil {
			log.Fatalf("wallet: %v", err)
		}
	}

	// تحقّقٌ في المكان نفسه: الرصيد يساوي مجموع قيوده، وإلّا فالمعاملة تسقط
	// كلُّها. **وزراعةٌ تكسر ثابتاً محاسبياً أسوأ من زراعةٍ لا تعمل** — الأولى
	// تُنتج قاعدةً تبدو سليمة، والثانية تصرخ فوراً.
	var bal, sum int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE((SELECT balance FROM wallets WHERE user_id = $1), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions WHERE user_id = $1), 0)`,
		id).Scan(&bal, &sum); err != nil {
		log.Fatal(err)
	}
	if bal != sum {
		log.Fatalf("زراعة الزبون تكسر الدفتر: الرصيد %d ومجموع القيود %d", bal, sum)
	}

	fmt.Printf("✅ زُرع الزبون: %s\n", customer.Name)
	fmt.Printf("   الدخول: %s / %s\n", customer.Phone, customer.Password)
	fmt.Printf("   الرصيد: %d — **بقيدِ إيداعٍ يقابله**، لا رقماً مكتوباً\n", bal)
	fmt.Println("   عناوينه:")
	for _, a := range customerAddresses {
		mark := ""
		if a.Default {
			mark = "  (الافتراضي)"
		}
		fmt.Printf("     · %-6s %s%s\n", a.Label, a.Text, mark)
	}
	fmt.Println()
	fmt.Println("   والعنوانان في منطقتَين مختلفتين: رسم التوصيل ١٠٬٠٠٠ للبيت")
	fmt.Println("   و١٥٬٠٠٠ للمحل — فيُرى أن الرسم يتبع الموقع لا المتجر.")
}
