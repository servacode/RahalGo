package catalog

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestCreateMerchant_DerivesCity **ومتجرٌ بلا مدينةٍ متجرٌ لا يراه أحد.**
//
// (شكوى المالك ٢٠٢٦-٠٨-٢٢: «الأصنافُ لم تظهر بالتطبيق أبداً».)
//
// # **العطبُ الذي أصلحته هذه**
//
// **`INSERT INTO merchants` لم يكن يذكر العمودَ إطلاقاً** — فكلُّ متجرٍ
// يُولد بلا مدينة. **وترشيحُ السوق يشترط تساويَ مدينةِ المتجر ومدينةِ
// الزبون** (`city_filter.go`)، فلا يظهر لأحد.
//
// **والأثرُ صامتٌ تماماً**: الأقسامُ تُعرض والعدّادُ صفر، ولا خطأَ ولا
// سجلّ. **وقِيس على الإنتاج**: ٣٧٦ صنفاً في القاعدة، **وصفرٌ لمن يرسل
// موقعَه** — والمالكُ رأى سوقاً فارغاً وظنّ التطبيقَ عطبان.
//
// **ولا تُسأل المدينةُ من صاحب المتجر**: وضع نقطتَه على الخريطة،
// **والمدينةُ تُعرف منها** — وسؤالُه بعدها سؤالٌ عمّا قال.
func TestCreateMerchant_DerivesCity(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	svc := NewService(pool, identity.NewService(identity.NewRepo(pool), nil, nil, nil, "test-secret", slog.New(slog.NewTextHandler(io.Discard, nil))))

	actor := testdb.NewUser(t, pool, "admin")

	var catID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO categories (name) VALUES ('مطاعم') RETURNING id`).Scan(&catID); err != nil {
		t.Fatalf("تعذّر التصنيف: %v", err)
	}
	// **مدينةٌ حقيقيّةُ الشكل** — مركزٌ ونصفُ قطر، كما في الإنتاج.
	//
	// **وموضعُها بعيدٌ عن كلّ مدينةٍ مزروعةٍ في قاعدة الفحص** — فلو وُضعت
	// على الرقّة لالتقط الاشتقاقُ رقّةً أخرى موجودةً من قبل، **فينجح
	// الاختبارُ لسببٍ غيرِ الذي يفحصه.**
	var cityID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO cities (name, center, radius_m, active)
		VALUES ('مدينةُ الفحص', ST_SetSRID(ST_MakePoint(60.0, 20.0), 4326)::geography, 5000, true)
		RETURNING id`).Scan(&cityID); err != nil {
		t.Fatalf("تعذّرت المدينة: %v", err)
	}

	lat, lng := 20.004, 60.004 // داخلَها بنحو نصف كيلومتر
	m, err := svc.CreateMerchant(ctx, actor, MerchantInput{
		Name:          ptr("متجرُ اختبارِ المدينة"),
		CategoryID:    ptr(catID),
		OwnerPhone:    ptr("+963900111222"),
		OwnerName:     ptr("صاحبُ المتجر"),
		OwnerPassword: ptr("Test@2026"),
		Lat:           &lat,
		Lng:           &lng,
	}, "127.0.0.1")
	if err != nil {
		t.Fatalf("تعذّر إنشاءُ المتجر: %v", err)
	}

	var got *string
	if err := pool.QueryRow(ctx,
		`SELECT city_id::text FROM merchants WHERE id = $1`, m.ID).Scan(&got); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if got == nil {
		t.Fatal("المتجرُ وُلد بلا مدينة — ولن يراه زبونٌ أبداً، ولا رسالةَ تقول ذلك")
	}
	if *got != cityID {
		t.Fatalf("نُسب إلى مدينةٍ أخرى: %s بدل %s", *got, cityID)
	}
}
