package orders_test

import (
	"context"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ══════════════════════════════════════════════════════════════════════
// **دائرةُ توصيلٍ واحدةٌ للعُدّة — تُرسَم ولا تُورَث**
// ══════════════════════════════════════════════════════════════════════
//
// (خطوةُ نظافة الاختبار ٢٠٢٦-٠٨-٢٠، بأمر المالك.)
//
// # ولماذا لزمت أصلاً
//
// **`Create` يكتب `zone_id` نصّاً**، **و`ZoneAt` على جدولٍ فارغٍ يردّ
// منطقةً صفريّةً بمعرّفٍ فارغ** — فيرفض بوستغرس `”` في عمود `uuid`
// بـ`22P02`.
//
// **وهذا عيبُ منتجٍ لا عيبُ اختبار** — سُجّل ولم يُمسّ هنا (`ZONE-000`
// في `docs/WORKLOG.md`): **تركيبٌ جديدٌ بلا دائرةٍ مرسومةٍ يردّ كلَّ طلب.**
//
// **والعُدّةُ كانت تنجو منه بدائرةٍ خلّفها عملٌ سابق** — فتُرسم صراحةً
// هنا: **الاختبارُ يفحص ما كُتب له، ولا يتّكئ على ما لا يعرف.**
//
// **والدبّوسُ `35.95, 39.00` هو الرقّة** — ودائرةُ خمسةٍ وعشرين
// كيلومتراً تغطّي كلَّ نقاط العُدّة.
func seedZone(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	// **ومرّةً واحدةً في العمليّة** — لا مع كلّ عُدّة.
	zoneOnce.Do(func() {
		_, zoneErr = pool.Exec(context.Background(), `
			INSERT INTO delivery_zones (name, center, radius_m, delivery_fee, min_order, active)
			SELECT 'دائرة العُدّة',
			       ST_SetSRID(ST_MakePoint(39.0094, 35.9506), 4326)::geography,
			       25000, 100, 0, true
			WHERE NOT EXISTS (SELECT 1 FROM delivery_zones WHERE name = 'دائرة العُدّة')`)
	})
	if zoneErr != nil {
		t.Fatalf("تعذّر رسمُ دائرة التوصيل: %v", zoneErr)
	}
}

var (
	zoneOnce sync.Once
	zoneErr  error
)
