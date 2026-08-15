package catalog

// **لا متجرَ بلا موضعٍ على الأرض.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «نسوي دبّوس المتجر إلزامي مو اختياري عند فتح
//  الحساب».)
//
// **وكان العمودُ يقبل الفراغ والواجهةُ تعرض الخريطةَ ولا تُلزم بها** —
// فيُفتح متجرٌ ويستقبل طلباتٍ وهو بلا نقطة. **ووقع فعلاً**: طلبٌ حقيقيٌّ
// (#1003) لم تظهر مسافتُه للسائق، والمحرّكُ يردّ `-1` أي «لا يُعرف».
//
// **ولا يكشفه فحصٌ بالعين**: الإنشاءُ يردّ ٢٠١ والمتجرُ يظهر في القائمة
// كاملاً. **الغائبُ وحدَه موضعُه** — ولا يُنظر إليه إلّا يومَ يقف سائقٌ في
// الشارع لا يعرف أين يذهب.

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// merchantFixture أقلُّ ما يلزم لإنشاء متجر: فاعلٌ وفئة.
type merchantFixture struct {
	svc     *Service
	actorID string
	catID   string
}

func newMerchantFixture(t *testing.T) *merchantFixture {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	// **وخدمةُ الهويّة تلزم الآن** — **ولا متجرَ بلا صاحب**
	// (٢٠٢٦-٠٨-١٥)، وإنشاءُ حسابه يمرّ بها.
	//
	// **وتُبنى بمستودعها وحدَه**: `EnsureUserWithRole` لا تقرأ ذاكرةً
	// ولا تُصدر توكناً — **وحقنُ خدمةٍ كاملةٍ في اختبارٍ لا يحتاجها
	// يجعله يسقط ليومَ تتبدّل تلك الخدمة.**
	f := &merchantFixture{
		svc: NewService(pool, identity.NewService(identity.NewRepo(pool), nil, nil, nil, "", slog.Default())),
	}
	f.actorID = testdb.NewUser(t, pool, "admin")

	if err := pool.QueryRow(ctx, `
		INSERT INTO categories (name, icon, sort_order) VALUES ('فئةُ موضعٍ للاختبار', '', 901)
		RETURNING id`).Scan(&f.catID); err != nil {
		t.Fatalf("تعذّر إنشاء الفئة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE category_id = $1`, f.catID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM categories WHERE id = $1`, f.catID)
	})
	return f
}

// TestCreateMerchant_RejectsMissingPin **بلا دبّوسٍ لا يُفتح متجر.**
func TestCreateMerchant_RejectsMissingPin(t *testing.T) {
	f := newMerchantFixture(t)

	_, err := f.svc.CreateMerchant(context.Background(), f.actorID, MerchantInput{
		Name:       ptr("متجرٌ بلا موضع"),
		CategoryID: ptr(f.catID),
	}, "127.0.0.1")

	if !errors.Is(err, ErrLocationRequired) {
		t.Fatalf("**قُبل متجرٌ بلا دبّوس** — والسائقُ لا يعرف أين يذهب إليه، "+
			"ولا مسافةَ له في بطاقة الطلب. الجواب: %v", err)
	}
}

// TestCreateMerchant_RejectsNullIsland **والصفران ليسا فراغاً — هما نقطةٌ في البحر.**
//
// **ومن قبلهما ضمناً** فتح متجراً «موضعُه» على بعد ألفي كيلومترٍ من
// الرقّة، **ثمّ حسب المحرّكُ مسافةً حقيقيّةً إليه** فبدت البطاقةُ سليمة.
func TestCreateMerchant_RejectsNullIsland(t *testing.T) {
	f := newMerchantFixture(t)

	_, err := f.svc.CreateMerchant(context.Background(), f.actorID, MerchantInput{
		Name:       ptr("متجرٌ في جزيرة الصفر"),
		CategoryID: ptr(f.catID),
		Lat:        ptr(0.0),
		Lng:        ptr(0.0),
	}, "127.0.0.1")

	if !errors.Is(err, ErrLocationRequired) {
		t.Fatalf("**قُبلت النقطة 0,0** — وهي في خليج غينيا لا في الرقّة. الجواب: %v", err)
	}
}

// TestCreateMerchant_AcceptsPin **وبدبّوسٍ صحيحٍ يُفتح ويُكتب موضعُه.**
func TestCreateMerchant_AcceptsPin(t *testing.T) {
	f := newMerchantFixture(t)
	ctx := context.Background()

	// **ولا متجرَ بلا صاحب** — (قرارُ المالك ٢٠٢٦-٠٨-١٥).
	m, err := f.svc.CreateMerchant(ctx, f.actorID, MerchantInput{
		Name:       ptr("متجرٌ بموضعٍ في الرقّة"),
		CategoryID: ptr(f.catID),
		OwnerPhone: ptr("+963900111222"),
		Lat:        ptr(35.9594),
		Lng:        ptr(39.0079),
	}, "127.0.0.1")
	if err != nil {
		t.Fatalf("رُفض متجرٌ بدبّوسٍ صحيح: %v", err)
	}

	// **والموضعُ يُقرأ من القاعدة لا من الجواب** — الجوابُ قد يعيد ما
	// أُرسل، **والسؤالُ هل وصل العمود.**
	var lat, lng float64
	if err := testdb.Pool(t).QueryRow(ctx, `
		SELECT ST_Y(location::geometry), ST_X(location::geometry)
		FROM merchants WHERE id = $1`, m.ID).Scan(&lat, &lng); err != nil {
		t.Fatalf("تعذّرت قراءة الموضع: %v", err)
	}
	if lat < 35.95 || lat > 35.97 || lng < 39.0 || lng > 39.02 {
		t.Fatalf("**الموضعُ المحفوظ غير المرسَل**: %v, %v", lat, lng)
	}
}
