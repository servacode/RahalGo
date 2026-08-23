package catalog

// **قسمُ السوق يُكتب عند الإنشاء كما يُكتب عند التعديل.**
//
// كان الحقلُ يُرسَل من نافذة الصنف ولا تذكره جملةُ الإدخال، **فيولد كلُّ صنفٍ
// خارج السوق**: لا يراه الزبونُ ولا يُعَدُّ في قسمه — **حتى يفتحه أحدٌ ويحفظه
// ثانيةً بلا تغيير**، فالتعديلُ وحدَه كان يكتبه.
//
// **ولا يكشفه فحصٌ بالعين**: النداءُ يردّ ٢٠١ ويعيد معرّفاً، والصنفُ يظهر في
// قائمة المتجر كاملاً. **الغائبُ وحدَه ظهورُه عند الزبون** — وهو ما لا يُنظر
// إليه بعد إضافةِ صنف.

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// menuFixture متجرٌ بقسمِ قائمةٍ وقسمِ سوق — أقلُّ ما يلزم لإنشاء صنف.
type menuFixture struct {
	pool       *pgxpool.Pool
	svc        *Service
	actorID    string
	merchantID string
	sectionID  string // قسمُ قائمة المتجر
	platformID string // قسمُ السوق
}

func newMenuFixture(t *testing.T) *menuFixture {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	f := &menuFixture{pool: pool, svc: NewService(pool, nil)}

	f.actorID = testdb.NewUser(t, pool, "admin")
	ownerID := testdb.NewUser(t, pool, "merchant")

	var catID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO categories (name, icon, sort_order) VALUES ('فئةُ اختبار', '', 900)
		RETURNING id`).Scan(&catID); err != nil {
		t.Fatalf("تعذّر إنشاء الفئة: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM categories WHERE id = $1`, catID) })

	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, owner_user_id) VALUES ('متجرُ اختبار', $1, $2)
		RETURNING id`, catID, ownerID).Scan(&f.merchantID); err != nil {
		t.Fatalf("تعذّر إنشاء المتجر: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, f.merchantID) })

	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'قسمُ المتجر')
		RETURNING id`, f.merchantID).Scan(&f.sectionID); err != nil {
		t.Fatalf("تعذّر إنشاء قسم القائمة: %v", err)
	}

	if err := pool.QueryRow(ctx, `
		INSERT INTO platform_sections (name, sort_order) VALUES ('قسمُ سوقٍ للاختبار', 900)
		RETURNING id`).Scan(&f.platformID); err != nil {
		t.Fatalf("تعذّر إنشاء قسم السوق: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM platform_sections WHERE id = $1`, f.platformID)
	})
	return f
}

func ptr[T any](v T) *T { return &v }

// TestCreateItem_KeepsPlatformSection **الصنفُ يولد في قسمه.**
func TestCreateItem_KeepsPlatformSection(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()

	id, err := f.svc.CreateItem(ctx, f.actorID, f.merchantID, MenuItemInput{
		SectionID:         ptr(f.sectionID),
		Name:              ptr("صنفٌ بقسمٍ مُختار"),
		Price:             ptr(int64(1000)),
		PlatformSectionID: ptr(f.platformID),
	}, "127.0.0.1")
	if err != nil {
		t.Fatalf("تعذّر إنشاء الصنف: %v", err)
	}

	var got *string
	if err := f.pool.QueryRow(ctx,
		`SELECT platform_section_id::text FROM menu_items WHERE id = $1`, id).Scan(&got); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if got == nil {
		t.Fatal("قسمُ السوق ضاع عند الإنشاء — الصنفُ وُلد خارج السوق")
	}
	if *got != f.platformID {
		t.Fatalf("قسمُ السوق %s والمُختار %s", *got, f.platformID)
	}
}

// TestCreateItem_RequiresPlatformSection **ولا صنفَ خارجَ السوق.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «الأدمنُ هو من يزرع الأقسام، والمتجرُ يجد
//
//	أقساماً جاهزة… وكلُّ الأصناف ستذهب إلى السوق بلوحة الأدمن مباشرة».)
//
// # وكان هذا الاختبارُ يحرس عكسَه
//
// **كان يقول: «ومن لم يختر لا يُختار له»** — والفراغُ يبقى فراغاً. وكان
// صحيحاً يومَ كان للمتجر أقسامُه الخاصّة: **الصنفُ يُعرض في قائمة متجره
// بقسمه المحلّيّ، وقسمُ السوق زيادةٌ للتصفّح.**
//
// **وبعد أن ذهبت طبقةُ أقسام المتجر** (هجرة ٠٠٨٤) صار قسمُ السوق **هو**
// انتماءَ الصنف — **فصنفٌ بلا قسمٍ لا يظهر في قائمة صاحبه ولا في السوق**:
// يُكتب في الجدول ولا يراه أحد.
//
// **فانقلب الحارسُ مع القاعدة**: يرفض ما كان يقبل.
func TestCreateItem_RequiresPlatformSection(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()

	for _, tc := range []struct {
		name string
		in   *string
	}{
		{"غيرُ مُرسَل", nil},
		{"مُرسَلٌ فارغاً", ptr("")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.svc.CreateItem(ctx, f.actorID, f.merchantID, MenuItemInput{
				Name:              ptr("صنفٌ بلا قسم — " + tc.name),
				Price:             ptr(int64(1000)),
				PlatformSectionID: tc.in,
			}, "127.0.0.1")
			if !errors.Is(err, ErrSectionRequired) {
				t.Fatalf("قُبل صنفٌ بلا قسمِ سوق (الخطأ: %v) — وهو لا يُعرض لأحد", err)
			}
		})
	}
}

// TestUpdateItem_RequiresPlatformSection **والبابُ الثاني كالأوّل.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٢٢: «إضافة الصنف القسم إلزامي».)
//
// **كان `CreateItem` وحدَه يحرسه** — فمن أنشأ صنفاً بقسمٍ ثمّ عدّله بلا
// قسمٍ أفلت، **وصار صنفاً لا يراه زبون**: المتجرُ يظنّه معروضاً ويدفع
// عمولةً على ما لا يُباع، **ولا رسالةَ تقول له.**
//
// **وحارسٌ على بابٍ من بابين ليس حارساً** — وهذا يحرس الثاني.
func TestUpdateItem_RequiresPlatformSection(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()

	// **صنفٌ سليمٌ أوّلاً** — فالمطلوبُ اختبارُ التعديل لا الإنشاء.
	id, err := f.svc.CreateItem(ctx, f.actorID, f.merchantID, MenuItemInput{
		SectionID:         ptr(f.sectionID),
		Name:              ptr("صنفٌ يُعدَّل"),
		Price:             ptr(int64(1000)),
		PlatformSectionID: ptr(f.platformID),
	}, "127.0.0.1")
	if err != nil {
		t.Fatalf("تعذّر إنشاءُ الصنف: %v", err)
	}

	// **والنزعُ الصريحُ يُردّ** — فراغٌ مُرسَلٌ يعني «ارفع القسم».
	t.Run("مُرسَلٌ فارغاً يُردّ", func(t *testing.T) {
		err := f.svc.UpdateItem(ctx, f.actorID, id, MenuItemInput{
			Name:              ptr("اسمٌ جديد"),
			PlatformSectionID: ptr(""),
		}, "127.0.0.1")
		if !errors.Is(err, ErrSectionRequired) {
			t.Fatalf("مرّ تعديلٌ ينزع القسمَ (الخطأ: %v) — والصنفُ يختفي من السوق صامتاً", err)
		}
	})

	// ══════════════════════════════════════════════════════════════════
	// **و«لم يُرسَل» يمرّ ولا يمسّ القسم**
	// ══════════════════════════════════════════════════════════════════
	//
	// (شكوى المالك ٢٠٢٦-٠٨-٢٢: «أريد تعديل الصورة يقول اختر القسم».)
	//
	// **نافذةُ التعديل في اللوحة لا ترسل القسمَ** لأنّها لا تبدّله —
	// **فحارسٌ يرفض غيابَه يمنع تبديلَ صورةٍ أو اسمٍ لكلّ صنفٍ في
	// المنصّة**، ولا يقول للمالك إلّا «اختر القسم» في نافذةٍ لا قسمَ
	// فيها.
	t.Run("غيرُ مُرسَلٍ يمرّ ويُبقي القسم", func(t *testing.T) {
		if err := f.svc.UpdateItem(ctx, f.actorID, id, MenuItemInput{
			Name: ptr("اسمٌ أحدث"),
		}, "127.0.0.1"); err != nil {
			t.Fatalf("رُدَّ تعديلٌ لا يمسّ القسمَ: %v", err)
		}
		var got *string
		if err := f.pool.QueryRow(ctx,
			`SELECT platform_section_id::text FROM menu_items WHERE id = $1`, id).
			Scan(&got); err != nil {
			t.Fatalf("تعذّرت القراءة: %v", err)
		}
		if got == nil || *got != f.platformID {
			t.Fatalf("تبدّل القسمُ بتعديلٍ لا يذكره: %v", got)
		}
	})
}
