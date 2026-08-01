package main

// متجرٌ واحد كامل — بياناتٌ تصلح للتجربة الحقيقية لا للفحص.
//
// الفرق بين الاثنين ليس في الاسم بل في الكثافة: متجرُ فحصٍ فيه صنفان يُثبت أن
// الشيفرة تعمل، ولا يُظهر كيف تبدو المنصة لزبونٍ يتصفّح. ستّة أقسام وأربعون
// صنفاً وخياراتُها هي ما يكشف ما لا يظهر في القليل — قائمةٌ تحتاج تمريراً،
// وأصنافٌ ذات خياراتٍ إلزامية، وأسعارٌ تتغيّر بالحجم، وصنفٌ نفد من المطبخ.
//
//	go run ./cmd/seed -store
//
// **ومعه مندوبُه.** فالمتجر في هذه المنصة لا يأتي من فراغ: مندوبٌ يزوره في
// السوق ويعطيه رابط دعوته، فيسجّل صاحبه، فتوافق الإدارة. ومتجرٌ بلا مندوب
// يُخفي نصف الدورة — لا عمولة إحالةٍ عليه، ولا عميلَ في لوحة مندوب، ولا عتبةَ
// تفعيلٍ تُختبر. فالزراعة تُنشئ الطرفين وتربطهما كما يربطهما الواقع، وتترك
// **أثر الرحلة**: طلبُ انضمامٍ مُحوَّل يشير إلى المتجر الذي وُلد منه.
//
// ولا يُنشئ غير هذين: لا زبون ولا سائق.

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/auth"
)

// storeOwner صاحب المطعم — حسابٌ بكلمة مرور يدخل به إلى بوابته.
var storeOwner = struct{ Phone, Name, Password string }{
	Phone:    "+963932556677",
	Name:     "أبو محمود الحاج علي",
	Password: "Matam@2026",
}

// storeRep المندوب الذي جلب المطعم.
//
// وكودُ دعوته **ثابتٌ في الزراعة لا مولَّد**: الكود المولَّد يتغيّر في كل
// قاعدة، فلا يصلح أن يُكتب في وثيقةٍ ولا أن يُجرَّب رابطُه مرّتين. وفي المسار
// الحقيقي يولّده `GrantRole` عشوائياً — وهذا مسار زراعةٍ لا مسارُ منصّة.
var storeRep = struct{ Phone, Name, Password, Code string }{
	Phone:    "+963944778899",
	Name:     "زياد العبدالله",
	Password: "Mandoub@2026",
	Code:     "RH-BAYT1",
}

// hoursSeed دوام يومٍ واحد. والأيام: 0 الأحد … 6 السبت.
type hoursSeed struct {
	Day         int
	Closed      bool
	Open, Close string
}

// storeSeed المطعم كاملاً.
type storeSeed struct {
	Name, Desc, Phone, Address, Category string
	Lat, Lng                             float64
	Commission                           int
	PrepMinutes                          int
	MinOrder                             int64
	Hours                                []hoursSeed
	Sections                             []storeSection
}

type storeSection struct {
	Name  string
	Items []storeItem
}

type storeItem struct {
	Name, Desc string
	Price      int64
	// Unavailable صنفٌ نفد اليوم — واقعُ كل مطبخ، وتجربةٌ لا تكتمل بدونه:
	// الزبون يجب أن يرى كيف يُعرض الصنف المنتهي، لا أن يفترض أن كلَّ شيء متاح.
	Unavailable bool
	Groups      []groupSeed
}

// ── المطعم ────────────────────────────────────────────────────────────────
//
// الأسعار بالليرة السورية ومُقدَّرةٌ على سوق الرقة: مقبّلةٌ حول العشرة آلاف،
// وسندويشة حول الثلاثين، وطبق مشاوٍ بين الخمسين والمئة. والتفاوت بين الأقسام
// مقصود — قائمةٌ كلُّ أصنافها بسعرٍ واحد لا تختبر شيئاً.
var restaurant = storeSeed{
	Name:     "مطعم بيت الرقة",
	Category: "مطاعم",
	Desc: "مشاوٍ على الفحم ووجبات شرقية، من مطبخ الرقة. " +
		"نفتح يومياً ونحضّر الطلب عند وروده لا قبله.",
	Phone:       "0223344556",
	Address:     "شارع تل أبيض — مقابل الحديقة العامة، الرقة",
	Lat:         35.9520,
	Lng:         39.0095,
	Commission:  10,
	PrepMinutes: 25, // مطبخ فحمٍ لا مطبخ سريع
	MinOrder:    25000,

	// **يفتح ٨:٠٠ لا ١١:٠٠**: القائمة فيها فتّة ولبنة وحمص وشاي — مطعمٌ يقدّم
	// هذه يفتح مع الصباح. وكانت ١١:٠٠ تجعله مغلقاً في أكثر ساعات التجربة.
	//
	// **والجمعة تبقى مختلفة**: تفصيلٌ صغير يجعل ساعات العمل تُختبر فعلاً بدل
	// أن تكون سبعة أيامٍ متطابقة لا تُفرّق بين منطقٍ صحيح وخاطئ.
	//
	// ولا تعبر منتصف الليل: `openNowSQL` يقارن بـ`BETWEEN` بسيط، فساعاتٌ من
	// ٢٣:٠٠ إلى ٠٢:٠٠ تُقرأ مغلقةً دائماً (D-13).
	Hours: []hoursSeed{
		{Day: 0, Open: "08:00", Close: "23:59"},
		{Day: 1, Open: "08:00", Close: "23:59"},
		{Day: 2, Open: "08:00", Close: "23:59"},
		{Day: 3, Open: "08:00", Close: "23:59"},
		{Day: 4, Open: "08:00", Close: "23:59"},
		{Day: 5, Open: "11:00", Close: "23:59"}, // الجمعة بعد الصلاة
		{Day: 6, Open: "08:00", Close: "23:59"},
	},

	Sections: []storeSection{
		// ── ١. المقبلات ──────────────────────────────────────────────────
		{Name: "المقبلات والسلطات", Items: []storeItem{
			{Name: "حمص بالطحينة", Desc: "حمص مطحون بالطحينة وزيت الزيتون", Price: 12000},
			{Name: "متبل باذنجان", Desc: "باذنجان مشوي على الفحم مع اللبن والثوم", Price: 13000},
			{Name: "تبولة", Desc: "برغل ناعم وبقدونس وبندورة وليمون", Price: 15000},
			{Name: "فتوش", Desc: "خضار موسمية مع خبز محمّص ودبس رمان", Price: 15000},
			{Name: "لبنة بزيت الزيتون", Desc: "لبنة بلدية مع نعناع وزيتون", Price: 10000},
			{Name: "سلطة خضراء", Desc: "خس وخيار وبندورة بصلصة الليمون", Price: 10000},
			{Name: "مخللات وزيتون", Desc: "طبق مشكّل", Price: 6000},
		}},

		// ── ٢. المشاوي — القسم الذي تُختبر فيه الخيارات ──────────────────
		{Name: "المشاوي على الفحم", Items: []storeItem{
			{
				Name: "شيش طاووق", Desc: "صدر دجاج متبّل، مع الخبز والثوم والمخلل",
				Price: 55000,
				Groups: []groupSeed{
					// إلزاميّ: الحجم يغيّر السعر، ولا يصحّ طلبٌ بلا حجم
					{Name: "الحجم", Min: 1, Max: 1, Options: []optSeed{
						{Name: "نصف وجبة (٤ أسياخ)", Delta: 0},
						{Name: "وجبة كاملة (٨ أسياخ)", Delta: 30000},
					}},
					{Name: "درجة الاستواء", Min: 1, Max: 1, Options: []optSeed{
						{Name: "وسط", Delta: 0},
						{Name: "مستوي جيداً", Delta: 0},
					}},
					{Name: "الإضافات", Min: 0, Max: 4, Options: []optSeed{
						{Name: "بطاطا مقلية", Delta: 12000},
						{Name: "خبز إضافي", Delta: 3000},
						{Name: "ثومية زيادة", Delta: 2000},
						{Name: "سلطة صغيرة", Delta: 8000},
					}},
				},
			},
			{
				Name: "كباب حلبي", Desc: "لحم غنم مفروم مع الفلفل الحلبي",
				Price: 68000,
				Groups: []groupSeed{
					{Name: "الحجم", Min: 1, Max: 1, Options: []optSeed{
						{Name: "نصف وجبة (٣ أسياخ)", Delta: 0},
						{Name: "وجبة كاملة (٦ أسياخ)", Delta: 40000},
					}},
					{Name: "درجة الحرارة", Min: 1, Max: 1, Options: []optSeed{
						{Name: "عادي", Delta: 0},
						{Name: "حار", Delta: 0},
					}},
					{Name: "الإضافات", Min: 0, Max: 4, Options: []optSeed{
						{Name: "بطاطا مقلية", Delta: 12000},
						{Name: "خبز إضافي", Delta: 3000},
						{Name: "سلطة صغيرة", Delta: 8000},
					}},
				},
			},
			{
				Name: "ريش غنم", Desc: "ريش بلدية على الفحم — تُحضَّر عند الطلب",
				Price: 98000,
				Groups: []groupSeed{
					{Name: "الإضافات", Min: 0, Max: 3, Options: []optSeed{
						{Name: "بطاطا مقلية", Delta: 12000},
						{Name: "خبز إضافي", Delta: 3000},
						{Name: "سلطة صغيرة", Delta: 8000},
					}},
				},
			},
			{
				Name: "فروج مشوي", Desc: "فروج بلدي كامل متبّل بالثوم والليمون",
				Price: 62000,
				Groups: []groupSeed{
					{Name: "الحجم", Min: 1, Max: 1, Options: []optSeed{
						{Name: "نصف فروج", Delta: 0},
						{Name: "فروج كامل", Delta: 35000},
					}},
					{Name: "الإضافات", Min: 0, Max: 3, Options: []optSeed{
						{Name: "بطاطا مقلية", Delta: 12000},
						{Name: "ثومية زيادة", Delta: 2000},
					}},
				},
			},
			{
				Name: "مشاوي مشكّلة", Desc: "طاووق وكباب وريش — تكفي شخصين",
				Price: 145000,
				Groups: []groupSeed{
					{Name: "الإضافات", Min: 0, Max: 3, Options: []optSeed{
						{Name: "بطاطا مقلية", Delta: 12000},
						{Name: "خبز إضافي", Delta: 3000},
						{Name: "حمص", Delta: 12000},
					}},
				},
			},
			{
				Name: "كبد غنم مشوي", Desc: "كبد طازج مع الفلفل والبصل",
				Price: 58000, Unavailable: true, // نفد اليوم
			},
		}},

		// ── ٣. الوجبات والصواني ──────────────────────────────────────────
		{Name: "الوجبات والصواني", Items: []storeItem{
			{Name: "فتة حمص باللحمة", Desc: "خبز محمّص وحمص ولبن ولحمة مفرومة وصنوبر", Price: 48000},
			{Name: "كبة مقلية", Desc: "ستّ حبات كبة برغل محشوّة", Price: 42000},
			{Name: "مقلوبة دجاج", Desc: "أرز وباذنجان ودجاج مع اللبن", Price: 65000},
			{Name: "صينية دجاج بالخضار", Desc: "دجاج وخضار في الفرن — تكفي شخصين", Price: 88000},
			{Name: "شاورما صحن", Desc: "شاورما دجاج مع البطاطا والثومية", Price: 52000,
				Groups: []groupSeed{
					{Name: "النوع", Min: 1, Max: 1, Options: []optSeed{
						{Name: "دجاج", Delta: 0},
						{Name: "لحمة", Delta: 15000},
					}},
				},
			},
		}},

		// ── ٤. الساندويشات ───────────────────────────────────────────────
		{Name: "الساندويشات", Items: []storeItem{
			{
				Name: "ساندويش شاورما دجاج", Desc: "بخبز الصاج مع الثومية والمخلل",
				Price: 26000,
				Groups: []groupSeed{
					{Name: "الحجم", Min: 1, Max: 1, Options: []optSeed{
						{Name: "عادي", Delta: 0},
						{Name: "كبير", Delta: 9000},
					}},
					{Name: "الإضافات", Min: 0, Max: 3, Options: []optSeed{
						{Name: "جبنة", Delta: 5000},
						{Name: "بطاطا داخل الساندويش", Delta: 4000},
						{Name: "حار", Delta: 0},
					}},
				},
			},
			{
				Name: "ساندويش شاورما لحمة", Desc: "لحم غنم مع الطحينة والبقدونس",
				Price: 34000,
				Groups: []groupSeed{
					{Name: "الحجم", Min: 1, Max: 1, Options: []optSeed{
						{Name: "عادي", Delta: 0},
						{Name: "كبير", Delta: 11000},
					}},
					{Name: "الإضافات", Min: 0, Max: 2, Options: []optSeed{
						{Name: "جبنة", Delta: 5000},
						{Name: "حار", Delta: 0},
					}},
				},
			},
			{Name: "ساندويش شيش طاووق", Desc: "طاووق وثومية ومخلل", Price: 29000},
			{Name: "ساندويش كباب", Desc: "كباب حلبي مع البندورة والبقدونس", Price: 32000},
			{Name: "ساندويش فلافل", Desc: "فلافل مع الخضار والطحينة", Price: 15000},
		}},

		// ── ٥. المشروبات ─────────────────────────────────────────────────
		{Name: "المشروبات", Items: []storeItem{
			{Name: "عيران", Desc: "لبن مخفوق بالنعناع", Price: 7000},
			{Name: "عصير ليمون بالنعناع", Desc: "طازج", Price: 13000},
			{Name: "مشروب غازي", Desc: "عبوة ٣٣٠ مل", Price: 6000,
				Groups: []groupSeed{
					{Name: "النكهة", Min: 1, Max: 1, Options: []optSeed{
						{Name: "كولا", Delta: 0},
						{Name: "برتقال", Delta: 0},
						{Name: "ليمون", Delta: 0},
					}},
				},
			},
			{Name: "ماء", Desc: "عبوة ٦٠٠ مل", Price: 3000},
			{Name: "شاي", Desc: "كأس شاي أحمر", Price: 4000},
		}},

		// ── ٦. الحلويات ──────────────────────────────────────────────────
		{Name: "الحلويات", Items: []storeItem{
			{Name: "كنافة نابلسية", Desc: "بالجبنة والقطر — تُحضَّر عند الطلب", Price: 32000},
			{Name: "بقلاوة", Desc: "أربع قطع مشكّلة", Price: 26000},
			{Name: "مهلبية", Desc: "بالحليب والفستق", Price: 15000},
			{Name: "أرز بحليب", Desc: "بالقرفة", Price: 14000},
		}},
	},
}

// seedStore يزرع المطعم وصاحبه — وقابلٌ للتكرار: تشغيله مرّتين لا يُنشئ نسختين.
func seedStore(ctx context.Context, tx pgx.Tx) {
	hash, err := auth.HashPassword(storeOwner.Password)
	if err != nil {
		log.Fatal(err)
	}

	// المندوب أولاً — فالمتجر يُنسب إليه عند إنشائه لا بعده.
	repHash, err := auth.HashPassword(storeRep.Password)
	if err != nil {
		log.Fatal(err)
	}
	var repID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (phone, full_name, password_hash, invite_code)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (phone) DO UPDATE SET
			full_name = EXCLUDED.full_name, password_hash = EXCLUDED.password_hash,
			invite_code = COALESCE(users.invite_code, EXCLUDED.invite_code)
		RETURNING id`,
		storeRep.Phone, storeRep.Name, repHash, storeRep.Code).Scan(&repID); err != nil {
		log.Fatalf("rep: %v", err)
	}
	// المندوب زبونٌ أيضاً — يتسوّق من المنصة كما يسوّق لها (نفس منطق GrantRole)
	for _, role := range []string{"sales", "customer"} {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_code) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, repID, role); err != nil {
			log.Fatal(err)
		}
	}

	// صاحب المتجر. **وكلمة مرورٍ نهائية لا مؤقّتة**: الحسابات التي يُنشئها
	// الأدمن من اللوحة تُلزم صاحبها بالتغيير لأن طرفاً ثالثاً يعرف كلمتها —
	// وهذا حسابُ زراعةٍ للتجربة، لا يُسلَّم لأحد.
	var ownerID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (phone, full_name, password_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (phone) DO UPDATE SET
			full_name = EXCLUDED.full_name, password_hash = EXCLUDED.password_hash
		RETURNING id`, storeOwner.Phone, storeOwner.Name, hash).Scan(&ownerID); err != nil {
		log.Fatalf("owner: %v", err)
	}
	// دورا المتجر والزبون معاً — صاحب المتجر يتسوّق أيضاً (نفس منطق GrantRole)
	for _, role := range []string{"merchant", "customer"} {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_code) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, ownerID, role); err != nil {
			log.Fatal(err)
		}
	}

	m := restaurant
	var mid string
	err = tx.QueryRow(ctx, `SELECT id FROM merchants WHERE name = $1`, m.Name).Scan(&mid)
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, `
			INSERT INTO merchants (name, description, category_id, phone, address_text,
			                       owner_user_id, sales_rep_user_id, commission_percent,
			                       default_prep_minutes, min_order, location)
			SELECT $1, $2, c.id, $3, $4, $5, $6, $7, $8, $9,
			       ST_SetSRID(ST_MakePoint($11, $10), 4326)::geography
			FROM categories c WHERE c.name = $12
			RETURNING id`,
			m.Name, m.Desc, m.Phone, m.Address, ownerID, repID, m.Commission,
			m.PrepMinutes, m.MinOrder, m.Lat, m.Lng, m.Category).Scan(&mid)
	} else if err == nil {
		// متجرٌ قائم من تشغيلٍ سابق: تُصحَّح نسبتُه **إن كانت فارغة فقط**.
		//
		// وشرطُ `IS NULL` ليس احتياطاً زائداً: نقلُ متجرٍ من مندوبٍ إلى آخر
		// ينقل دخلاً من إنسانٍ إلى إنسان، وهو قرارٌ لا يتّخذه سكربتُ زراعة.
		// (وهذا نفسه ما سُدّ في R-59 حين كان الرابط يُعيد نسبة متجرٍ قائم.)
		if _, err := tx.Exec(ctx, `
			UPDATE merchants SET sales_rep_user_id = $2
			WHERE id = $1 AND sales_rep_user_id IS NULL`, mid, repID); err != nil {
			log.Fatalf("attribute: %v", err)
		}
	}
	if err != nil {
		log.Fatalf("merchant: %v", err)
	}

	// أثرُ الرحلة: طلبُ انضمامٍ مُحوَّل يشير إلى المتجر الذي وُلد منه. وبدونه
	// يظهر المتجر في «عملائي» عند المندوب بلا قصّةٍ تسبقه — كأنّه هبط.
	if _, err := tx.Exec(ctx, `
		INSERT INTO merchant_leads (store_name, owner_name, phone, area, note,
		                            sales_rep_user_id, status, merchant_id,
		                            category_id, lat, lng, owner_password_hash)
		SELECT $1, $2, $3, $4, $5, $6, 'converted', $7, c.id, $8, $9, $10
		FROM categories c WHERE c.name = $11
		  AND NOT EXISTS (SELECT 1 FROM merchant_leads WHERE merchant_id = $7)`,
		m.Name, storeOwner.Name, storeOwner.Phone, "وسط المدينة",
		"زيارة ميدانية — وافق صاحب المطعم في نفس اليوم",
		repID, mid, m.Lat, m.Lng, hash, m.Category); err != nil {
		log.Fatalf("lead: %v", err)
	}

	for _, h := range m.Hours {
		if _, err := tx.Exec(ctx, `
			INSERT INTO merchant_hours (merchant_id, day_of_week, closed, open_time, close_time)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (merchant_id, day_of_week) DO UPDATE SET
				closed = EXCLUDED.closed, open_time = EXCLUDED.open_time,
				close_time = EXCLUDED.close_time`,
			mid, h.Day, h.Closed, h.Open, h.Close); err != nil {
			log.Fatalf("hours: %v", err)
		}
	}

	items, groups, opts := 0, 0, 0
	for si, sec := range m.Sections {
		var secID string
		err := tx.QueryRow(ctx,
			`SELECT id FROM menu_sections WHERE merchant_id = $1 AND name = $2`,
			mid, sec.Name).Scan(&secID)
		if err == pgx.ErrNoRows {
			err = tx.QueryRow(ctx, `
				INSERT INTO menu_sections (merchant_id, name, sort_order)
				VALUES ($1, $2, $3) RETURNING id`, mid, sec.Name, si+1).Scan(&secID)
		}
		if err != nil {
			log.Fatalf("section %s: %v", sec.Name, err)
		}

		for ii, it := range sec.Items {
			var itemID string
			err := tx.QueryRow(ctx,
				`SELECT id FROM menu_items WHERE merchant_id = $1 AND name = $2`,
				mid, it.Name).Scan(&itemID)
			if err != pgx.ErrNoRows {
				if err != nil {
					log.Fatalf("item %s: %v", it.Name, err)
				}
				continue // موجودٌ سلفاً — لا نُكرّر خياراته
			}
			if err := tx.QueryRow(ctx, `
				INSERT INTO menu_items (merchant_id, section_id, name, description,
				                        price, available, sort_order)
				VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
				mid, secID, it.Name, it.Desc, it.Price, !it.Unavailable, ii+1).Scan(&itemID); err != nil {
				log.Fatalf("item %s: %v", it.Name, err)
			}
			items++

			for gi, g := range it.Groups {
				var gid string
				if err := tx.QueryRow(ctx, `
					INSERT INTO modifier_groups (item_id, name, min_select, max_select, sort_order)
					VALUES ($1, $2, $3, $4, $5) RETURNING id`,
					itemID, g.Name, g.Min, g.Max, gi).Scan(&gid); err != nil {
					log.Fatalf("group %s: %v", g.Name, err)
				}
				groups++
				for oi, o := range g.Options {
					if _, err := tx.Exec(ctx, `
						INSERT INTO modifier_options (group_id, name, price_delta, sort_order)
						VALUES ($1, $2, $3, $4)`, gid, o.Name, o.Delta, oi); err != nil {
						log.Fatalf("option %s: %v", o.Name, err)
					}
					opts++
				}
			}
		}
	}

	var repOf string
	_ = tx.QueryRow(ctx, `
		SELECT COALESCE(u.full_name, '—') FROM merchants mm
		LEFT JOIN users u ON u.id = mm.sales_rep_user_id WHERE mm.id = $1`, mid).Scan(&repOf)

	fmt.Printf("✅ زُرع المتجر: %s\n", m.Name)
	fmt.Printf("   %d قسماً · %d صنفاً · %d مجموعة خيارات · %d خياراً\n",
		len(m.Sections), items, groups, opts)
	fmt.Printf("   التحضير %d دقيقة · الحدّ الأدنى %d · العمولة %d%%\n",
		m.PrepMinutes, m.MinOrder, m.Commission)
	fmt.Println()
	fmt.Printf("   صاحب المتجر : %s / %s   (%s)\n",
		storeOwner.Phone, storeOwner.Password, storeOwner.Name)
	fmt.Printf("   المندوب     : %s / %s   (%s · كوده %s)\n",
		storeRep.Phone, storeRep.Password, storeRep.Name, storeRep.Code)
	fmt.Printf("   والمتجر منسوبٌ إلى: %s\n", repOf)
}
