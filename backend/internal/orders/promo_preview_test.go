package orders

// **المعاينةُ تقول ما يقوله الطلب — بحرفه.**
//
// (كشفَ الحاجةَ إليها فحصٌ يدويٌّ ٢٠٢٦-٠٨-٠٨: السلّةُ تحمل حقلَ كودٍ ولا
//
//	تقول عنه شيئاً — لا خصماً ولا رفضاً — حتّى يُضغط «تأكيد الطلب».)
//
// # ما يحرسه هذا الاختبار
//
// **ليس أنّ المعاينةَ تعمل** — بل **أنّها لا تفترق عن الطلب.** فالخطرُ هنا
// ليس عطباً في شاشةٍ، **بل قاعدتان تقولان الشيءَ نفسَه بكلمتين**: تَعِد
// الشاشةُ بخصمٍ يرفضه الخادمُ عند الطلب، **وهو أسوأُ من ألّا تَعِد بشيء.**
//
// **ولذلك تُنادي `PreviewPromo` دالّةَ الطلب نفسَها** (`validatePromo`) في
// معاملةٍ تُلغى. وهذا الاختبارُ يُثبت الأمرين:
//
//   - **أنّ الجوابَ واحد** — لكلّ قاعدةٍ من قواعد الكود السبع.
//   - **وأنّ المعاينةَ لا تترك أثراً**: لا عدّادَ يزيد ولا قيدَ يُكتب.
//     فمن جرّب عشرةَ أكوادٍ لم يستهلك واحداً، **ومن عاين كوداً «مرّةً لكلّ
//     مستخدم» بقي له.**

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// promoFixture كودٌ وزبونٌ — أقلُّ ما يلزم.
func promoFixture(t *testing.T, kind string, value, minOrder int64, active bool) (*Service, string, string, string) {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	svc := &Service{db: pool}

	customer := testdb.NewUser(t, pool, "customer")

	code := "PRV" + customer[:6]
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO promo_codes (code, kind, value, min_order, active)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		code, kind, value, minOrder, active).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاء الكود: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM promo_codes WHERE id = $1`, id)
	})
	return svc, code, customer, id
}

// TestPreviewPromo_MatchesOrder **ما تَعِد به المعاينةُ يقع عند الطلب.**
func TestPreviewPromo_MatchesOrder(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name     string
		kind     string
		value    int64
		minOrder int64
		active   bool
		subtotal int64
		fee      int64
		valid    bool
		discount int64
		feeAfter int64
	}{
		{"نسبةٌ عشرة", "percent", 10, 0, true, 30000, 10000, true, 3000, 10000},
		{"مبلغٌ ثابت", "fixed", 5000, 0, true, 30000, 10000, true, 5000, 10000},
		{"ثابتٌ يتجاوز الفاتورة يُقصّ", "fixed", 90000, 0, true, 30000, 10000, true, 30000, 10000},
		{"توصيلٌ مجّانيّ", "free_delivery", 0, 0, true, 30000, 10000, true, 0, 0},
		{"مطفأٌ يُردّ", "percent", 10, 0, false, 30000, 10000, false, 0, 10000},
		{"دون الحدّ الأدنى يُردّ", "percent", 10, 50000, true, 30000, 10000, false, 0, 10000},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, code, customer, _ := promoFixture(t, c.kind, c.value, c.minOrder, c.active)

			got, err := svc.PreviewPromo(ctx, code, customer, c.subtotal, c.fee)
			if err != nil {
				t.Fatalf("تعذّرت المعاينة: %v", err)
			}
			if got.Valid != c.valid {
				t.Fatalf("الصلاحيّة %v والمنتظَر %v", got.Valid, c.valid)
			}
			if got.Discount != c.discount {
				t.Fatalf("الخصمُ %d والمنتظَر %d", got.Discount, c.discount)
			}
			if got.DeliveryFee != c.feeAfter {
				t.Fatalf("التوصيلُ %d والمنتظَر %d", got.DeliveryFee, c.feeAfter)
			}

			// **والطلبُ يقول ما قالته المعاينة** — بالدالّة التي يستعملها.
			tx, err := svc.db.Begin(ctx)
			if err != nil {
				t.Fatalf("تعذّرت المعاملة: %v", err)
			}
			defer func() { _ = tx.Rollback(ctx) }()
			fee := c.fee
			_, discount, vErr := svc.validatePromo(ctx, tx, code, customer, c.subtotal, &fee)
			if (vErr == nil) != c.valid {
				t.Fatalf("**الطلبُ خالف المعاينة**: خطؤه %v والمعاينةُ قالت صالح=%v", vErr, c.valid)
			}
			if c.valid && (discount != got.Discount || fee != got.DeliveryFee) {
				t.Fatalf("**رقمان لمعنًى واحد**: المعاينةُ %d/%d والطلبُ %d/%d",
					got.Discount, got.DeliveryFee, discount, fee)
			}
		})
	}
}

// TestPreviewPromo_LeavesNoTrace **ومن عاين لم يستهلك.**
func TestPreviewPromo_LeavesNoTrace(t *testing.T) {
	ctx := context.Background()
	svc, code, customer, id := promoFixture(t, "percent", 10, 0, true)

	for i := 0; i < 5; i++ {
		if _, err := svc.PreviewPromo(ctx, code, customer, 30000, 10000); err != nil {
			t.Fatalf("تعذّرت المعاينة %d: %v", i, err)
		}
	}

	var used int
	if err := svc.db.QueryRow(ctx,
		`SELECT used_count FROM promo_codes WHERE id = $1`, id).Scan(&used); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if used != 0 {
		t.Fatalf("**خمسُ معايناتٍ استهلكت %d** — والمعاينةُ قراءةٌ لا استعمال", used)
	}

	var redemptions int
	if err := svc.db.QueryRow(ctx,
		`SELECT count(*) FROM promo_redemptions WHERE promo_id = $1`, id).Scan(&redemptions); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if redemptions != 0 {
		t.Fatalf("**كُتب %d قيدَ استعمال** بلا طلب", redemptions)
	}
}

// TestPreviewPromo_UnknownCodeIsAnswerNotError **وكودٌ لا يوجد جوابٌ لا عطب.**
func TestPreviewPromo_UnknownCodeIsAnswerNotError(t *testing.T) {
	ctx := context.Background()
	svc, _, customer, _ := promoFixture(t, "percent", 10, 0, true)

	got, err := svc.PreviewPromo(ctx, "LA-YUJAD-ABADAN", customer, 30000, 10000)
	if err != nil {
		t.Fatalf("**رُدَّ بخطأٍ لا بجواب** — والشاشةُ تعرض «حدث خطأ» بدل «الكود غير صالح»: %v", err)
	}
	if got.Valid {
		t.Fatal("كودٌ لا يوجد عُدَّ صالحاً")
	}
	if got.DeliveryFee != 10000 {
		t.Fatalf("التوصيلُ تبدّل بكودٍ مرفوض: %d", got.DeliveryFee)
	}
}
