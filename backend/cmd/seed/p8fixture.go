package main

// ======================================================================
//  **شاهدُ العزل — متجرٌ ثانٍ لمندوبٍ ثانٍ** (`P-8`)
// ======================================================================
//
// # ولماذا لزم متجرٌ ثانٍ
//
// **وكان في التجهيز متجرٌ واحدٌ منسوبٌ إلى مندوبٍ واحد** — **فبندُ
// «متجرٌ غيرُ منسوبٍ يُمنَع» لا يُثبَت**: **لا ثانيَ لتُمنَع عنه.**
// **ومنعٌ لا شاهدَ له منعٌ مُدَّعىً.**
//
// # ولماذا مندوبٌ ثانٍ لا متجرٌ بلا مندوب
//
// **ومتجرٌ بلا مندوبٍ يُثبت الأضعف**: **مُنع لأنّه لا مندوبَ له.**
// **والمطلوبُ أن يُمنع مندوبٌ عن متجرِ غيرِه** — **وذاك لا يظهر إلّا
// بمندوبَين.**
//
// # ولا SQL مرتجلة
//
// **ويمرّ هذا بمصنع `seedMerchant` نفسِه** الذي بنى المتجرَ الأوّل —
// **فالشاهدُ من طينة المشهود له.** **ومتجرٌ يُركَّب بيدٍ أخرى قد ينقصه
// دوامٌ أو موقعٌ أو صنفٌ فيمرّ الاختبارُ فارغاً.**

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/auth"
)

// **وحساباتُ الشاهد** — **أرقامٌ اصطناعيّةٌ في مدى التجارب.**
var (
	p8RepTwo = struct{ Phone, Name, Password string }{
		"+963900700021", "مندوبُ العزل الثاني", "P8Isolation@2026"}
	p8OwnerTwo = struct{ Phone, Name, Password string }{
		"+963900700022", "صاحبُ المتجر الثاني", "P8Isolation@2026"}
)

// seedP8Isolation **يزرع مندوباً ثانياً ومتجرَه** — وقابلٌ للتكرار.
func seedP8Isolation(ctx context.Context, tx pgx.Tx) {
	repID := p8User(ctx, tx, p8RepTwo.Phone, p8RepTwo.Name, p8RepTwo.Password, "sales")
	ownerID := p8User(ctx, tx, p8OwnerTwo.Phone, p8OwnerTwo.Name, p8OwnerTwo.Password, "merchant")

	// **والمتجرُ كاملٌ لا هيكل**: **دوامٌ وموقعٌ وأقسامٌ وأصناف** —
	// **فمن مُنع عنه مُنع عن متجرٍ يعمل، لا عن صفٍّ في جدول.**
	seedMerchant(ctx, tx, merchantSeed{
		Name: "مخبز الفرات", Category: "بقالة", Phone: "0223456799",
		Address: "حي الرميلة", Desc: "مخبزٌ ومعجّنات — شاهدُ عزلٍ للقبول",
		Owner: ownerID, Rep: repID,
		Lat: 35.9560, Lng: 39.0180, Commission: 10,
		Sections: []sectionSeed{
			{Name: "معجّنات", Platform: "وجبات شعبية", Items: []itemSeed{
				{"فطيرة جبنة", "عجينٌ طازجٌ وجبنةٌ بلديّة", 6000, nil},
				{"مناقيش زعتر", "زعترٌ وزيتُ زيتون", 5000, nil},
			}},
			{Name: "خبز", Platform: "بقالة", Items: []itemSeed{
				{"ربطة خبز", "ربطةٌ كاملة", 3000, nil},
			}},
		},
	})

	log.Printf("شاهدُ العزل: مندوبٌ ثانٍ %s · صاحبٌ %s · مخبز الفرات", repID, ownerID)
}

// p8User **حسابٌ بدورٍ واحدٍ** — ولا يُنشئ نسخةً ثانية.
func p8User(ctx context.Context, tx pgx.Tx, phone, name, password, role string) string {
	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatal(err)
	}
	var id string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (phone, full_name, password_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (phone) DO UPDATE SET
			full_name     = EXCLUDED.full_name,
			password_hash = EXCLUDED.password_hash
		RETURNING id`, phone, name, hash).Scan(&id); err != nil {
		log.Fatalf("p8 user %s: %v", phone, err)
	}
	for _, r := range fieldRoles(role) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_code) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, id, r); err != nil {
			log.Fatal(err)
		}
	}
	// **وحارسُ الواتساب يمنع الطلبَ من رقمٍ غير موثَّق** — **وشاهدٌ لا
	// يستطيع أن يطلب شاهدٌ نصفُه.**
	if _, err := tx.Exec(ctx, `
		UPDATE users SET whatsapp_phone = phone, whatsapp_verified_at = now()
		WHERE id = $1 AND whatsapp_verified_at IS NULL`, id); err != nil {
		log.Fatal(err)
	}
	return id
}
