// أمر زراعة بيانات التطوير: يملأ قاعدة البيانات ببيانات تجريبية كاملة وصحيحة
// (حسابات بكل الأدوار وكلمات مرور معروفة، متاجر بقوائمها، مناطق، أكواد خصم)
// بشكل قابل للتكرار — تشغيله مرتين لا يكرر شيئاً.
//
//	make seed   أو   go run ./cmd/seed
//
// **ووضعان آخران**:
//   - `go run ./cmd/seed -store` يزرع **متجراً واحداً كاملاً وصاحبه**: مطعمٌ
//     بستّة أقسام وأربعين صنفاً وخياراتها وساعات عمله. بياناتٌ للتجربة الحقيقية
//     لا للفحص — والفرق بينهما الكثافة: متجرٌ بصنفين يُثبت أن الشيفرة تعمل
//     ولا يُظهر كيف تبدو المنصة لزبونٍ يتصفّح.
//   - `go run ./cmd/seed -drivers` يزرع **ثلاثة سائقين خارج الدوام**. والعدد
//     ليس اعتباطاً: الطابور مشترك، وسائقٌ واحد لا يُظهر التنافس عليه ولا سقفَ
//     الطلبات المتزامنة ولا الحاجةَ إلى الإسناد اليدوي.
//   - `go run ./cmd/seed -customer` يزرع **زبوناً بعنوانَين ورصيدِ محفظة**.
//     والرصيد **مُقيَّدٌ لا مكتوب**: الرصيد عمودٌ مشتقّ من دفتر، وكتابتُه بلا
//     قيدٍ يقابله تُنتج محفظةً بمالٍ لا مصدر له.
//   - `go run ./cmd/seed -staff` يزرع **طاقم المنصة وحده** —
//
// أدمن وعمليات ومالية، بلا متاجر ولا زبائن ولا أرصدة تجريبية. وهو ما يلزم
// بعد تنظيف القاعدة لبدايةٍ نظيفة: **قاعدةٌ بلا أدمن قاعدةٌ لا يُدخَل إليها**،
// وبقية الكيانات يصنعها صاحبها من اللوحة كما يصنعها في الإنتاج.
//
// ممنوع في الإنتاج: يرفض العمل إذا APP_ENV=production.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/config"
	"github.com/servacode/rahalgo/backend/internal/database"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/migrate"
)

// حسابات التطوير المعتمدة — الأرقام تُطبّع إلى ‎+963 تلقائياً عند الدخول.
var accounts = []struct {
	Phone, Name, Password, Role string
	InviteCode                  string
	WalletBalance               int64
}{
	{"+963999000001", "مدير المنصة", "RahalGo@2026", "admin", "", 0},
	{"+963955333444", "سارة العمليات", "Ops@2026", "ops", "", 0},
	{"+963955444555", "منى المالية", "Finance@2026", "finance", "", 0},
	{"+963977888999", "أحمد المندوب", "Rep@2026", "sales", "RH-DEMO1", 0},
	{"+963955111222", "محمد السائق", "Driver@2026", "driver", "", 0},
	{"+963966777888", "أبو خالد — صاحب قصر الشام", "Store@2026", "merchant", "", 0},
	{"+963966888999", "أم ليث — صاحبة بقالية الفرات", "Store@2026", "merchant", "", 0},
	{"+963933000111", "زبون تجريبي", "Customer@2026", "customer", "", 200000},
}

func main() {
	staffOnly := flag.Bool("staff", false, "زراعة طاقم المنصة وحده (أدمن/عمليات/مالية) بلا بيانات تجريبية")
	storeOnly := flag.Bool("store", false, "زراعة متجرٍ واحد كامل وصاحبه ومندوبه — ولا شيء غيره")
	driversOnly := flag.Bool("drivers", false, "زراعة ثلاثة سائقين خارج الدوام — ولا شيء غيرهم")
	ordersOnly := flag.Bool("orders", false, "زراعة طلباتٍ في كلّ الحالات — يحتاج حساباتٍ ومتجراً موجودَين")
	customerOnly := flag.Bool("customer", false, "زراعة زبونٍ بعنوانَين ورصيدِ محفظةٍ مُقيَّد — ولا شيء غيره")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.Env == "production" {
		log.Fatal("seed: ممنوع في الإنتاج (APP_ENV=production)")
	}

	ctx := context.Background()
	pg, err := database.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pg.Close()

	if n, err := migrate.Up(ctx, pg); err != nil {
		log.Fatal(err)
	} else if n > 0 {
		fmt.Printf("طُبّقت %d هجرة\n", n)
	}

	tx, err := pg.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// أوضاعٌ مركَّزة: كلٌّ يزرع ما يخصّه ولا يمرّ ببقية الزراعة
	if *storeOnly || *driversOnly || *customerOnly || *ordersOnly {
		if *storeOnly {
			seedStore(ctx, tx)
		}
		if *driversOnly {
			seedDrivers(ctx, tx)
		}
		if *customerOnly {
			seedCustomer(ctx, tx)
		}
		if *ordersOnly {
			seedOrders(ctx, tx)
		}
		if err := tx.Commit(ctx); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	// ---------- الحسابات والأدوار والمحافظ ----------
	ids := map[string]string{} // phone → user id
	for _, a := range accounts {
		// وضع الطاقم: الأدوار الثلاثة التي تُدير المنصة لا التي تستعملها
		if *staffOnly && a.Role != "admin" && a.Role != "ops" && a.Role != "finance" {
			continue
		}
		hash, err := auth.HashPassword(a.Password)
		if err != nil {
			log.Fatal(err)
		}
		var id string
		// موجود مسبقاً؟ نضمن الاسم وكلمة المرور والدور — لا ننشئ نسخة ثانية
		err = tx.QueryRow(ctx, `
			INSERT INTO users (phone, full_name, password_hash, invite_code)
			VALUES ($1, $2, $3, NULLIF($4, ''))
			ON CONFLICT (phone) DO UPDATE SET
				full_name     = CASE WHEN users.full_name = '' THEN EXCLUDED.full_name ELSE users.full_name END,
				password_hash = EXCLUDED.password_hash,
				invite_code   = COALESCE(users.invite_code, EXCLUDED.invite_code)
			RETURNING id`, a.Phone, a.Name, hash, a.InviteCode).Scan(&id)
		if err != nil {
			log.Fatalf("user %s: %v", a.Phone, err)
		}
		ids[a.Phone] = id
		// **وزبونُ التطوير موثَّقُ الواتساب.**
		//
		// الحارسُ يمنع الطلبَ من رقمٍ غير موثَّق — **وهو صحيح**: الرقمُ الوهميّ
		// يعني سائقاً يقف أمام بابٍ لا أحد فيه. **لكنّ زبوناً مزروعاً لا
		// يستطيع أن يطلب زبونٌ لا ينفع في تجربة**، فيُطفأ الحارسُ في كلّ
		// تجربةٍ — **وحارسٌ يُطفأ ليُجرَّب النظامُ حارسٌ لا يُجرَّب أبداً.**
		if _, err := tx.Exec(ctx, `
			UPDATE users SET whatsapp_phone = phone, whatsapp_verified_at = now()
			WHERE id = $1 AND whatsapp_verified_at IS NULL`, id); err != nil {
			log.Fatalf("whatsapp %s: %v", a.Phone, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_code) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, id, a.Role); err != nil {
			log.Fatalf("role %s: %v", a.Phone, err)
		}
		// دورُ الزبون للأدوار الميدانية — بالمفتاح المركزي لا بشرطٍ محلّي
		for _, extra := range fieldRoles(a.Role)[1:] {
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_roles (user_id, role_code) VALUES ($1, $2)
				ON CONFLICT DO NOTHING`, id, extra); err != nil {
				log.Fatalf("customer role %s: %v", a.Phone, err)
			}
		}
		if a.WalletBalance > 0 {
			var exists bool
			_ = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM wallets WHERE user_id = $1)`, id).Scan(&exists)
			if !exists {
				if _, err := tx.Exec(ctx,
					`INSERT INTO wallets (user_id, balance) VALUES ($1, $2)`, id, a.WalletBalance); err != nil {
					log.Fatal(err)
				}
				if _, err := tx.Exec(ctx, `
					INSERT INTO wallet_transactions (user_id, amount, kind, note, created_by)
					VALUES ($1, $2, 'topup', 'رصيد تجريبي — زراعة بيانات التطوير', $3)`,
					id, a.WalletBalance, ids["+963999000001"]); err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	// ---------- مناطق التغطية (دوائر على الرقة) ----------
	for _, z := range []struct {
		Name          string
		Lat, Lng      float64
		RadiusM       int
		Fee, MinOrder int64
	}{
		{"مركز المدينة", 35.9528, 39.0079, 4000, 10000, 20000},
		{"المشلب والدرعية", 35.9650, 39.0450, 3000, 15000, 25000},
	} {
		if _, err := tx.Exec(ctx, `
			INSERT INTO delivery_zones (name, center, radius_m, delivery_fee, min_order)
			SELECT $1, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, $4, $5, $6
			WHERE NOT EXISTS (SELECT 1 FROM delivery_zones WHERE name = $1)`,
			z.Name, z.Lat, z.Lng, z.RadiusM, z.Fee, z.MinOrder); err != nil {
			log.Fatal(err)
		}
	}

	// ---------- المتاجر بقوائمها ----------
	//
	// وضعُ الطاقم يتخطّاها ويتخطّى كود الخصم: **المناطق تبقى** لأن بلا منطقةٍ
	// واحدة لا يُقبل أيّ طلب — وهي بنيةُ عملٍ لا بيانات عرض. أمّا المتاجر
	// فيصنعها صاحبها من اللوحة كما يصنعها في الإنتاج.
	if !*staffOnly {
		seedDemo(ctx, tx, ids)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("✅ الزراعة اكتملت — الحسابات:")
	for _, a := range accounts {
		if *staffOnly && a.Role != "admin" && a.Role != "ops" && a.Role != "finance" {
			continue
		}
		fmt.Printf("  %-14s %-28s %-10s %s\n", a.Phone, a.Name, a.Role, a.Password)
	}
	os.Exit(0)
}

// fieldRoles الأدوار التي تُمنح لصاحب حسابٍ ميدانيّ.
//
// دورُ الزبون يُضاف **إن كان المفتاح المركزي مرفوعاً** (`identity.FieldRolesAreCustomers`)
// — وهو مُطفأٌ مؤقّتاً لأجل التجربة. والزراعة تمرّ بالمفتاح نفسه كي لا تُنشئ
// حساباتٍ تخالف ما يفعله الخادم.
func fieldRoles(role string) []string {
	if identity.FieldRolesAreCustomers {
		return []string{role, "customer"}
	}
	return []string{role}
}

// seedDemo البيانات التجريبية: متجران بقوائمهما وكود خصم ترحيبي.
func seedDemo(ctx context.Context, tx pgx.Tx, ids map[string]string) {
	seedMerchant(ctx, tx, merchantSeed{
		Name: "مطعم قصر الشام", Category: "مطاعم", Phone: "0223456789",
		Address: "شارع تل أبيض، مقابل الجامع الكبير", Desc: "مشاوي ووجبات شرقية",
		Owner: ids["+963966777888"], Rep: ids["+963977888999"],
		Lat: 35.9500, Lng: 39.0100, Commission: 10,
		Sections: []sectionSeed{
			{"مشاوي", []itemSeed{
				{"شيش طاووق", "مع البطاطا والثوم", 48000, []groupSeed{
					{"الحجم", 1, 1, []optSeed{{"عادي", 0}, {"دوبل", 20000}}},
					{"إضافات", 0, 3, []optSeed{{"جبنة", 5000}, {"بطاطا إضافية", 7000}, {"ثوم إضافي", 2000}}},
				}},
				{"كباب حلبي", "كيلو مشوي على الفحم", 95000, nil},
			}},
			{"مشروبات", []itemSeed{
				{"عصير برتقال", "طازج", 8000, nil},
				{"غازيات", "", 6000, nil},
			}},
		},
	})
	seedMerchant(ctx, tx, merchantSeed{
		Name: "بقالية الفرات", Category: "بقالة", Phone: "",
		Address: "حي المشلب", Desc: "مواد غذائية وتموينية",
		Owner: ids["+963966888999"], Rep: ids["+963977888999"],
		Lat: 35.9640, Lng: 39.0430, Commission: 5,
		Sections: []sectionSeed{
			{"أساسيات", []itemSeed{
				{"ربطة خبز", "", 4000, nil},
				{"حليب مبستر 1ل", "", 12000, nil},
				{"بيض (طبق 30)", "", 45000, nil},
			}},
		},
	})

	// ---------- كود خصم ترحيبي ----------
	if _, err := tx.Exec(ctx, `
		INSERT INTO promo_codes (code, kind, value, min_order, first_order_only, once_per_user, max_uses)
		VALUES ('WELCOME50', 'percent', 50, 30000, true, true, 100)
		ON CONFLICT (code) DO NOTHING`); err != nil {
		log.Fatal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("✅ الزراعة اكتملت — حسابات التطوير:")
	for _, a := range accounts {
		fmt.Printf("  %-14s %-28s %-10s %s\n", a.Phone, a.Name, a.Role, a.Password)
	}
	os.Exit(0)
}

type optSeed struct {
	Name  string
	Delta int64
}
type groupSeed struct {
	Name     string
	Min, Max int
	Options  []optSeed
}
type itemSeed struct {
	Name, Desc string
	Price      int64
	Groups     []groupSeed
}
type sectionSeed struct {
	Name  string
	Items []itemSeed
}
type merchantSeed struct {
	Name, Category, Phone, Address, Desc string
	Owner, Rep                           string
	Lat, Lng                             float64
	Commission                           int
	Sections                             []sectionSeed
}

func seedMerchant(ctx context.Context, tx pgx.Tx, m merchantSeed) {
	var id string
	err := tx.QueryRow(ctx, `SELECT id FROM merchants WHERE name = $1`, m.Name).Scan(&id)
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, `
			INSERT INTO merchants (name, description, category_id, phone, address_text,
			                       owner_user_id, sales_rep_user_id, commission_percent, location)
			SELECT $1, $2, c.id, $3, $4, $5, $6, $7,
			       ST_SetSRID(ST_MakePoint($9, $8), 4326)::geography
			FROM categories c WHERE c.name = $10
			RETURNING id`,
			m.Name, m.Desc, m.Phone, m.Address, m.Owner, m.Rep, m.Commission,
			m.Lat, m.Lng, m.Category).Scan(&id)
	}
	if err != nil {
		log.Fatalf("merchant %s: %v", m.Name, err)
	}

	// دوام كامل الأسبوع 09:00–23:00
	for d := 0; d <= 6; d++ {
		if _, err := tx.Exec(ctx, `
			INSERT INTO merchant_hours (merchant_id, day_of_week, closed, open_time, close_time)
			VALUES ($1, $2, false, '09:00', '23:00') ON CONFLICT DO NOTHING`, id, d); err != nil {
			log.Fatal(err)
		}
	}

	for si, sec := range m.Sections {
		var secID string
		err := tx.QueryRow(ctx,
			`SELECT id FROM menu_sections WHERE merchant_id = $1 AND name = $2`, id, sec.Name).Scan(&secID)
		if err == pgx.ErrNoRows {
			err = tx.QueryRow(ctx, `
				INSERT INTO menu_sections (merchant_id, name, sort_order)
				VALUES ($1, $2, $3) RETURNING id`, id, sec.Name, si+1).Scan(&secID)
		}
		if err != nil {
			log.Fatal(err)
		}
		for ii, it := range sec.Items {
			var itemID string
			err := tx.QueryRow(ctx,
				`SELECT id FROM menu_items WHERE merchant_id = $1 AND name = $2`, id, it.Name).Scan(&itemID)
			if err == pgx.ErrNoRows {
				err = tx.QueryRow(ctx, `
					INSERT INTO menu_items (merchant_id, section_id, name, description,
					                        merchant_price, price, sort_order)
					VALUES ($1, $2, $3, $4, $5, $5, $6) RETURNING id`,
					id, secID, it.Name, it.Desc, it.Price, ii+1).Scan(&itemID)
				if err != nil {
					log.Fatal(err)
				}
				for gi, g := range it.Groups {
					var groupID string
					if err := tx.QueryRow(ctx, `
						INSERT INTO modifier_groups (item_id, name, min_select, max_select, sort_order)
						VALUES ($1, $2, $3, $4, $5) RETURNING id`,
						itemID, g.Name, g.Min, g.Max, gi).Scan(&groupID); err != nil {
						log.Fatal(err)
					}
					for oi, o := range g.Options {
						if _, err := tx.Exec(ctx, `
							INSERT INTO modifier_options (group_id, name, price_delta, sort_order)
							VALUES ($1, $2, $3, $4)`, groupID, o.Name, o.Delta, oi); err != nil {
							log.Fatal(err)
						}
					}
				}
			} else if err != nil {
				log.Fatal(err)
			}
		}
	}
}
