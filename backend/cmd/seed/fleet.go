package main

// ══════════════════════════════════════════════════════════════════════
// **أسطولُ التجربة — `go run ./cmd/seed -fleet`** (طلبُ المالك ٢٠٢٦-١٠-٠١)
// ══════════════════════════════════════════════════════════════════════
//
// (نصُّه: «سجّل حساب ٣ مناديب ثمّ أنشئ لكلّ مندوبٍ متجرين أو ٣ متاجر ثمّ
//  بكلّ متجرٍ أضف أصنافاً بحيث ١٠ أصنافٍ لكلّ متجر، وأيضاً المتاجر الموجودة
//  سابقاً، ثمّ أنشئ ٥ سائقين».)
//
// - ثلاثةُ مناديب، لكلٍّ ثلاثةُ متاجرَ بأصنافٍ عشرة، **ودوامٌ طوالَ اليوم**
//   فتُطلَب في أيّ ساعةِ تجربة.
// - **والمتاجرُ القائمةُ تُكمَّل إلى عشرة** — لا يُمسّ ما فيها.
// - خمسةُ سائقين خارجَ الدوام.
//
// **وكلمةُ المرور من البيئة** (`FLEET_PASSWORD`) **لا من الشيفرة** — المستودعُ
// عامّ. **ولا مالَ يُمسّ**: لا عمولةَ تُضبط (افتراضُ القاعدة)، ولا مرشَّحَ
// يُحوَّل (فلا مكافأةَ هدفٍ تُقيَّد)، ولا رصيد.
//
// **ويُعاد تشغيلُه بلا تكرار**: كلُّ صفٍّ يُطابَق بمفتاحه قبل أن يُدرج.

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/catalog"
)

type fleetItem struct {
	Name, Desc string
	Price      int64
	Platform   string
}

// أصنافٌ عشرةٌ لكلّ نوع — **وقسمُ السوق باسمه** كما في القاعدة.
var fleetMenus = map[string][]fleetItem{
	"مطاعم": {
		{"شاورما دجاج صحن", "مع البطاطا والثومية", 45000, "شاورما"},
		{"ساندويش شاورما", "بخبز الصاج", 22000, "شاورما"},
		{"شيش طاووق", "وجبة مع الخبز والمخلل", 55000, "مشاوي"},
		{"كباب حلبي", "نصف كيلو", 70000, "مشاوي"},
		{"برغر لحم", "مع الجبنة والبطاطا", 40000, "برغر"},
		{"بيتزا خضار", "حجم وسط", 48000, "بيتزا وفطائر"},
		{"فروج بروستد", "نصف فروج", 52000, "فروج وبروستد"},
		{"حمص بالطحينة", "صحن", 12000, "مقبّلات"},
		{"فتوش", "سلطة موسمية", 14000, "مقبّلات"},
		{"عيران", "كأس", 6000, "مشروبات باردة"},
	},
	"بقالة": {
		{"رز مصري", "كيس ٢ كغ", 28000, "مواد غذائيّة"},
		{"سكر", "كيس ١ كغ", 14000, "مواد غذائيّة"},
		{"زيت دوار الشمس", "عبوة ١ لتر", 26000, "مواد غذائيّة"},
		{"شاي", "علبة ٤٠٠ غ", 32000, "مواد غذائيّة"},
		{"لبن", "علبة ١ كغ", 12000, "ألبان وأجبان"},
		{"جبنة بيضاء", "نصف كيلو", 30000, "ألبان وأجبان"},
		{"بيض", "طبق ٣٠ بيضة", 45000, "دواجن"},
		{"مياه معدنية", "عبوة ١.٥ لتر", 4000, "مياه"},
		{"شيبس", "كيس كبير", 7000, "سناكات وتسالي"},
		{"مسحوق غسيل", "٣ كغ", 55000, "منظّفات"},
	},
	"حلويات": {
		{"كنافة نابلسية", "صحن", 30000, "حلويّات"},
		{"بقلاوة مشكّلة", "نصف كيلو", 60000, "حلويّات"},
		{"معمول بالتمر", "علبة", 35000, "حلويّات"},
		{"هريسة", "قطعتان", 12000, "حلويّات"},
		{"مهلبية", "كأس", 10000, "حلويّات"},
		{"أرز بحليب", "كأس", 10000, "حلويّات"},
		{"كاتو شوكولا", "قالب صغير", 85000, "حلويّات"},
		{"بوظة عربية", "كأس كبير", 15000, "حلويّات"},
		{"قهوة عربية", "فنجان", 5000, "قهوة ومشروبات ساخنة"},
		{"عصير برتقال", "كأس طازج", 12000, "مشروبات باردة"},
	},
	"صيدليات": {
		{"باراسيتامول", "علبة ٢٠ حبّة", 8000, "صيدليّة"},
		{"فيتامين سي", "علبة فوّار", 25000, "صيدليّة"},
		{"شاش طبي", "لفافة", 5000, "صيدليّة"},
		{"معقّم يدين", "عبوة ٢٥٠ مل", 12000, "عناية شخصيّة"},
		{"معجون أسنان", "أنبوب", 14000, "عناية شخصيّة"},
		{"شامبو", "عبوة ٤٠٠ مل", 30000, "عناية شخصيّة"},
		{"حفاضات أطفال", "علبة مقاس ٤", 95000, "أطفال"},
		{"حليب أطفال", "علبة ٤٠٠ غ", 120000, "أطفال"},
		{"كريم مرطّب", "عبوة", 35000, "تجميل وعطور"},
		{"مقياس حرارة", "رقمي", 40000, "صيدليّة"},
	},
	"هدايا": {
		{"باقة ورد أحمر", "١٢ وردة", 150000, "ورد وهدايا"},
		{"باقة ورد مشكّلة", "متوسطة", 110000, "ورد وهدايا"},
		{"علبة شوكولا", "هدية", 90000, "ورد وهدايا"},
		{"دبدوب", "حجم وسط", 70000, "ورد وهدايا"},
		{"بطاقة معايدة", "مع ظرف", 8000, "قرطاسيّة"},
		{"دفتر ملاحظات", "غلاف جلد", 25000, "قرطاسيّة"},
		{"قلم حبر", "علبة هدية", 30000, "قرطاسيّة"},
		{"عطر", "٥٠ مل", 180000, "تجميل وعطور"},
		{"ساعة جدارية", "خشب", 95000, "أدوات منزليّة"},
		{"شمعة معطّرة", "زجاجية", 35000, "أدوات منزليّة"},
	},
}

type fleetStore struct {
	Name, Category, Address string
	Lat, Lng                float64
}

type fleetRep struct {
	Phone, Name, Code string
	Stores            []fleetStore
}

// المواقعُ حولَ وسط الرقّة — **داخلَ التغطية.**
var fleetReps = []fleetRep{
	{"+963900600101", "مندوب تجربة 1 — سامر", "RH-TST01", []fleetStore{
		{"مطعم الفرات", "مطاعم", "شارع تل أبيض", 35.9530, 39.0120},
		{"بقالية النور", "بقالة", "حي الثكنة", 35.9480, 39.0060},
		{"حلويات الشام", "حلويات", "شارع القوتلي", 35.9555, 39.0170},
	}},
	{"+963900600102", "مندوب تجربة 2 — رامي", "RH-TST02", []fleetStore{
		{"مطعم الريف", "مطاعم", "حي المشلب", 35.9440, 39.0220},
		{"صيدلية الشفاء", "صيدليات", "دوار النعيم", 35.9510, 39.0010},
		{"بقالية الخير", "بقالة", "حي الرميلة", 35.9600, 39.0080},
	}},
	{"+963900600103", "مندوب تجربة 3 — خالد", "RH-TST03", []fleetStore{
		{"زهور الربيع", "هدايا", "شارع المنصور", 35.9470, 39.0150},
		{"مطعم السنابل", "مطاعم", "حي الدرعية", 35.9580, 39.0020},
		{"حلويات الأمل", "حلويات", "حي البدو", 35.9420, 39.0090},
	}},
}

var fleetDrivers = []struct{ Phone, Name string }{
	{"+963900600301", "سائق تجربة 1 — أحمد"},
	{"+963900600302", "سائق تجربة 2 — محمود"},
	{"+963900600303", "سائق تجربة 3 — يوسف"},
	{"+963900600304", "سائق تجربة 4 — علي"},
	{"+963900600305", "سائق تجربة 5 — حسن"},
}

// fleetUser يُدرج حساباً أو يجده — ويمنحه دورَه وزبونَه (`fieldRoles`).
func fleetUser(ctx context.Context, tx pgx.Tx, phone, name, hash, code, role string) string {
	var id string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (phone, full_name, password_hash, invite_code)
		VALUES ($1, $2, $3, NULLIF($4, ''))
		ON CONFLICT (phone) DO UPDATE SET
			full_name = EXCLUDED.full_name, password_hash = EXCLUDED.password_hash,
			invite_code = COALESCE(users.invite_code, EXCLUDED.invite_code)
		RETURNING id`, phone, name, hash, code).Scan(&id); err != nil {
		log.Fatalf("user %s: %v", phone, err)
	}
	for _, r := range fieldRoles(role) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_code) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, id, r); err != nil {
			log.Fatalf("role %s: %v", phone, err)
		}
	}
	// **والرقمُ نفسُه رقمُ واتساب موثَّق** — حسابٌ واحدٌ برقمٍ واحد.
	if _, err := tx.Exec(ctx, `
		UPDATE users SET whatsapp_phone = phone, whatsapp_verified_at = now()
		WHERE id = $1 AND whatsapp_verified_at IS NULL`, id); err != nil {
		log.Fatalf("whatsapp %s: %v", phone, err)
	}
	return id
}

// fleetFill **يُكمل قائمةَ متجرٍ إلى عشرة** — لا يُكرّر اسماً ولا يمسّ القائم.
func fleetFill(ctx context.Context, tx pgx.Tx, merchantID, category string) int {
	menu, ok := fleetMenus[category]
	if !ok {
		menu = fleetMenus["مطاعم"]
	}
	var have int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM menu_items WHERE merchant_id = $1`, merchantID).Scan(&have); err != nil {
		log.Fatal(err)
	}
	added := 0
	for i, it := range menu {
		if have+added >= 10 {
			break
		}
		var exists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM menu_items WHERE merchant_id = $1 AND name = $2)`,
			merchantID, it.Name).Scan(&exists); err != nil {
			log.Fatal(err)
		}
		if exists {
			continue
		}
		var plat string
		if err := tx.QueryRow(ctx, `
			SELECT id FROM platform_sections WHERE name = $1 AND active
			ORDER BY sort_order, id LIMIT 1`, it.Platform).Scan(&plat); err != nil {
			// **وقاعدةٌ بأقسامٍ أقلّ** (نظيفة) — أوّلُ قسمٍ فاعل، ويُقال.
			if err := tx.QueryRow(ctx, `
				SELECT id FROM platform_sections WHERE active
				ORDER BY sort_order, id LIMIT 1`).Scan(&plat); err != nil {
				log.Fatalf("لا قسمَ سوقٍ فاعل: %v", err)
			}
			fmt.Printf("   ⚠ قسمُ «%s» غيرُ موجود — وُضع «%s» في أوّل قسمٍ فاعل\n", it.Platform, it.Name)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO menu_items (merchant_id, platform_section_id, name, description,
			                        merchant_price, price, available, sort_order)
			VALUES ($1, $2, $3, $4, $5, $5, true, $6)`,
			merchantID, plat, it.Name, it.Desc, it.Price, i+1); err != nil {
			log.Fatalf("item %s: %v", it.Name, err)
		}
		added++
	}
	return added
}

func seedFleet(ctx context.Context, tx pgx.Tx) {
	pw := os.Getenv("FLEET_PASSWORD")
	if len(pw) < 8 {
		log.Fatal("fleet: FLEET_PASSWORD مطلوبة (٨ أحرفٍ فأكثر) — لا تُكتب في الشيفرة")
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("── المناديب ومتاجرهم ──")
	for ri, r := range fleetReps {
		repID := fleetUser(ctx, tx, r.Phone, r.Name, hash, r.Code, "sales")
		fmt.Printf("مندوب: %s  %s  (كود %s)\n", r.Phone, r.Name, r.Code)
		for si, st := range r.Stores {
			ownerPhone := fmt.Sprintf("+9639006002%d%d", ri+1, si+1)
			ownerID := fleetUser(ctx, tx, ownerPhone, "صاحب "+st.Name, hash, "", "merchant")
			var mid string
			err := tx.QueryRow(ctx, `SELECT id FROM merchants WHERE owner_user_id = $1 AND name = $2`,
				ownerID, st.Name).Scan(&mid)
			if err == pgx.ErrNoRows {
				err = tx.QueryRow(ctx, `
					INSERT INTO merchants (name, description, category_id, phone, address_text,
					                       owner_user_id, sales_rep_user_id, location, city_id)
					SELECT $1, $2, c.id, $3, $4, $5, $6,
					       ST_SetSRID(ST_MakePoint($8, $7), 4326)::geography,
					       `+catalog.CityOfPointSQL(7, 8)+`
					FROM categories c WHERE c.name = $9
					RETURNING id`,
					st.Name, "متجر تجربة", ownerPhone, st.Address+"، الرقة",
					ownerID, repID, st.Lat, st.Lng, st.Category).Scan(&mid)
			}
			if err != nil {
				log.Fatalf("merchant %s: %v", st.Name, err)
			}
			// **دوامٌ طوالَ اليوم** — فيُطلَب منه في أيّ ساعةِ تجربة.
			for d := 0; d < 7; d++ {
				if _, err := tx.Exec(ctx, `
					INSERT INTO merchant_hours (merchant_id, day_of_week, closed, open_time, close_time)
					VALUES ($1, $2, false, '00:00', '23:59')
					ON CONFLICT (merchant_id, day_of_week) DO NOTHING`, mid, d); err != nil {
					log.Fatalf("hours: %v", err)
				}
			}
			n := fleetFill(ctx, tx, mid, st.Category)
			fmt.Printf("   متجر: %-16s %-8s صاحبه %s  (+%d صنفاً)\n", st.Name, st.Category, ownerPhone, n)
		}
	}

	fmt.Println("── المتاجر القائمة تُكمَّل إلى عشرة ──")
	rows, err := tx.Query(ctx, `
		SELECT m.id, m.name, c.name FROM merchants m JOIN categories c ON c.id = m.category_id
		WHERE m.owner_user_id NOT IN (SELECT id FROM users WHERE phone LIKE '+9639006002%')
		ORDER BY m.created_at`)
	if err != nil {
		log.Fatal(err)
	}
	type ex struct{ id, name, cat string }
	var olds []ex
	for rows.Next() {
		var e ex
		if err := rows.Scan(&e.id, &e.name, &e.cat); err != nil {
			log.Fatal(err)
		}
		olds = append(olds, e)
	}
	rows.Close()
	for _, e := range olds {
		fmt.Printf("   %-16s +%d صنفاً\n", e.name, fleetFill(ctx, tx, e.id, e.cat))
	}

	fmt.Println("── السائقون (خارجَ الدوام) ──")
	for _, d := range fleetDrivers {
		fleetUser(ctx, tx, d.Phone, d.Name, hash, "", "driver")
		fmt.Printf("   %s  %s\n", d.Phone, d.Name)
	}
}
