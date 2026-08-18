package orders_test

// **جدولُ مناطقَ فارغٌ يُوصَّل إليه بأجرة اللوحة — لا يُردّ.**
//
// (شكوى المالك ٢٠٢٦-٠٨-١٨: «التوصيلُ لا يجلب سعرَ التوصيل من لوحة
//
//	الإدارة، تأكّد منه».)
//
// # ما قِيس على الخادم الحيّ
//
// `delivery.fee = 110` مضبوطةٌ في اللوحة، **و`delivery_zones` صفرُ
// صفوف.** فيردّ `ZoneAt` «خارجَ التغطية»، **وتضع التسعيرةُ الرسمَ
// صفراً بلا خطأ** — فيرى الزبونُ «التوصيل ٠».
//
// # وأخطرُ سطرٍ هنا الحالُ الثانية
//
// **من رسم دائرةً واحدةً يجب أن يعود الحدُّ يعمل.** ولو فُتحت التغطيةُ
// دائماً **لَصار كلُّ دبّوسٍ في سوريا داخلَ النطاق** — ويقبل الطلبَ من
// مدينةٍ لا سائقَ فيها.

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

func TestZoneAt_EmptyTableCoversEverywhere(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL غير مضبوط")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("تعذّر الاتّصال: %v", err)
	}
	// ══════════════════════════════════════════════════════════════════
	// **والإغلاقُ يُسجَّل أوّلاً ليقع أخيرا**
	// ══════════════════════════════════════════════════════════════════
	//
	// **و`defer db.Close()` يقع قبل `t.Cleanup`** — الأولى تُنفَّذ عند
	// خروج الدالّة، والثانية بعد خروجها. **فيُنظَّف على مسبحٍ مغلقٍ
	// فلا يقع شيء** — والأخطاءُ مبتلعةٌ فلا يصرخ أحد.
	//
	// **ووقع فعلاً ٢٠٢٦-٠٨-١٨**: بقيت دائرةٌ نافذةٌ بعيدةٌ في قاعدة
	// الاختبار، **فسقط أربعةُ اختباراتٍ بريئةٍ بـ`out_of_zone`.**
	//
	// **و`t.Cleanup` تُنفَّذ بعكس تسجيلها** — فيُسجَّل الإغلاقُ أوّلاً
	// ليقع آخرا.
	t.Cleanup(db.Close)

	// **وتُطفأ المناطقُ ولا تُحذف** — الطلباتُ تشير إليها بمفتاحٍ
	// أجنبيّ، **والحذفُ يسقط بـ`23503` فيُتخطّى الاختبار**: أخضرُ لا
	// يقيس شيئا. **والمقيسُ هو «لا منطقةَ نافذة» وهو ما تقرؤه الشيفرة.**
	//
	// ══════════════════════════════════════════════════════════════════
	// **ويُعاد ما أُطفئ — وإلّا سقط جيرانُه**
	// ══════════════════════════════════════════════════════════════════
	//
	// **القاعدةُ مشتركةٌ بين اختبارات الحزمة كلِّها**، وهي تُشغَّل على
	// التوالي. **فمن أطفأ مناطقَها وترك دائرةً بعيدةً نافذةً أسقط كلَّ
	// من ينشئ طلباً بعده بـ`out_of_zone`.**
	//
	// **ووقع فعلاً ٢٠٢٦-٠٨-١٨**: أربعةُ اختباراتِ خصمٍ وسباقٍ سقطت،
	// **والعطبُ في اختباري لا في شيفرتها** — وهو أخبثُ ما يقع: يُبحث
	// في البريء.
	var saved []string
	rows, err := db.Query(ctx, `SELECT id::text FROM delivery_zones WHERE active`)
	if err != nil {
		t.Skipf("لا قاعدةَ مهيّأة: %v", err)
	}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			saved = append(saved, id)
		}
	}
	rows.Close()

	t.Cleanup(func() {
		_, _ = db.Exec(ctx, `DELETE FROM delivery_zones WHERE name = 'دمشق-فحص'`)
		if len(saved) > 0 {
			_, _ = db.Exec(ctx,
				`UPDATE delivery_zones SET active = true WHERE id::text = ANY($1)`, saved)
		}
	})

	if _, err := db.Exec(ctx, `UPDATE delivery_zones SET active = false`); err != nil {
		t.Skipf("تعذّر الإطفاء: %v", err)
	}

	svc := orders.NewService(db, nil, nil, nil, nil, nil)

	// **الرقّة** — نقطةٌ حقيقيّةٌ لا صفران.
	const lat, lng = 35.9506, 39.0094

	z, err := svc.ZoneAt(ctx, lat, lng)
	if err != nil {
		t.Fatalf("بجدولٍ فارغٍ رُدَّ الدبّوس: %v — والزبونُ يرى «خارج النطاق»", err)
	}
	if z.ID != "" {
		t.Errorf("بجدولٍ فارغٍ رُدَّت منطقةٌ باسم %q", z.ID)
	}

	// **ودائرةٌ واحدةٌ تُعيد الحدَّ إلى العمل** — بعيدةٌ عن النقطة.
	if _, err := db.Exec(ctx, `
		INSERT INTO delivery_zones (name, center, radius_m, active, delivery_fee, min_order)
		VALUES ('دمشق-فحص', ST_SetSRID(ST_MakePoint(36.2765, 33.5138),4326)::geography,
		        3000, true, 0, 0)`); err != nil {
		t.Fatalf("تعذّر إدراجُ منطقة: %v", err)
	}
	if _, err := svc.ZoneAt(ctx, lat, lng); !errors.Is(err, orders.ErrOutOfZone) {
		t.Fatalf("بدائرةٍ بعيدةٍ يجب أن يُردَّ الدبّوس — رُدَّ %v", err)
	}
}
