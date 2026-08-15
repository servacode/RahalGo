package orders_test

// **طلباتُ الحساب تُقرأ كاملةً — بصفحاتٍ لا بسقف.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٥: «اجعلها كاملةً وليس ٥٠ فقط… واجعل باجينيشن
//  أيضاً».)
//
// # ولماذا حارسٌ لهذا
//
// **تبويبُ «طلباته» كان يجلب `per_page=50` بلا صفحةٍ ثانية** — فمن طلب
// ستّين يُعرض له خمسون، **والعشرةُ الباقيةُ لا بابَ إليها.** ولا خطأَ ولا
// إنذار: **قائمةٌ تعمل وتَنقُص.**
//
// **والعنوانُ كان يعدّ الصفَّ المعروض** لا مجموعَ ما في القاعدة — فيقول
// «طلباته (٥٠)» **والنظرةُ العامّةُ فوقَه تقول ٦٠**، ولا أحدَ يعرف أيُّهما
// الصواب.
//
// **وما يُحرَس هنا عقدُ المحرّك** الذي تتّكئ عليه الشاشة: **`total` مجموعُ
// ما في القاعدة لا طولُ الصفحة**، والصفحاتُ لا تُسقط طلباً ولا تكرّره.
//
// # وأنّ الخاصَّ والملغيَّ فيها
//
// **مُرشِّحُ القائمة لا يذكر نوعاً ولا حالاً** — فالملغيُّ والمسلَّم
// والخاصُّ كلُّها تُقرأ. **وضمُّ المتجر يساريٌّ قصداً**: كان صلباً فاختفى
// أوّلُ طلبٍ خاصٍّ من كلّ قراءةٍ كأنّه لم يُكتب.

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// TestList_PagesThroughEveryOrder **خمسةٌ وعشرون تُقرأ كلُّها في صفحتين.**
func TestList_PagesThroughEveryOrder(t *testing.T) {
	pool := testdb.Pool(t)
	store := settings.NewStore(pool)
	svc := orders.NewService(pool, nil, wallet.NewService(pool),
		cashbox.NewService(pool, store), nil,
		slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	svc.SetSettings(store)
	ctx := context.Background()
	customer := testdb.NewUser(t, pool, "customer")

	// **ومتجرٌ للعاديّ** — القاعدةُ تفرضه (`orders_standard_has_merchant`)،
	// **وهو نفسُه ما لا يملكه الخاصّ**، فيُقاس الاثنان في قائمةٍ واحدة.
	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات في القاعدة: %v", err)
	}
	var merchantID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent)
		VALUES ('متجر فحص الصفحات', $1, 10) RETURNING id::text`,
		categoryID).Scan(&merchantID); err != nil {
		t.Fatalf("تعذّر إنشاءُ متجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, merchantID)
	})

	// **وحالاتُها مختلفة** — ملغيٌّ ومسلَّمٌ وجارٍ: **المُرشِّحُ لا يسأل عن
	// الحال**، ولو سأل لَنقصت القائمةُ بلا سبب.
	states := []string{"delivered", "cancelled", "dispatching", "rejected", "failed"}
	const n = 25
	customs := 0
	for i := range n {
		kind, req := "standard", ""
		mer := &merchantID
		// **وواحدٌ من كلّ خمسةٍ خاصّ** — بلا متجرٍ، وهو ما كان يسقط.
		if i%5 == 0 {
			kind, req, mer = "custom", fmt.Sprintf("طلبٌ خاصٌّ رقم %d", i), nil
			customs++
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO orders (kind, customer_id, merchant_id, status, address_text, dropoff,
				payment_method, custom_request, subtotal, delivery_fee, total, cash_due,
				created_at)
			VALUES ($1, $2, $3, $4, 'عنوان اختبار',
				ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
				'cash', $5, 0, 0, 0, 0, now() - ($6::int || ' minutes')::interval)`,
			kind, customer, mer, states[i%len(states)], req, i); err != nil {
			t.Fatalf("تعذّر إنشاءُ الطلب %d: %v", i, err)
		}
	}

	// ══════════════════════════════════════════════════════════════════
	// **والمجموعُ ما في القاعدة لا ما في الصفحة**
	// ══════════════════════════════════════════════════════════════════
	first, err := svc.List(ctx, orders.ListFilter{CustomerID: customer, Page: 1, PerPage: 20})
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ الصفحة الأولى: %v", err)
	}
	if first.Total != n {
		t.Fatalf("المجموعُ %d والمكتوبُ في القاعدة %d — **والعنوانُ يقرأ منه، فيقول رقماً تكذّبه النظرةُ العامّة**",
			first.Total, n)
	}
	if len(first.Orders) != 20 {
		t.Fatalf("الصفحةُ الأولى %d صفّاً لا 20", len(first.Orders))
	}

	// **والثانيةُ تحمل الباقي** — وهو ما لم يكن له بابٌ قبل اليوم.
	second, err := svc.List(ctx, orders.ListFilter{CustomerID: customer, Page: 2, PerPage: 20})
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ الصفحة الثانية: %v", err)
	}
	if len(second.Orders) != n-20 {
		t.Fatalf("الصفحةُ الثانية %d صفّاً لا %d", len(second.Orders), n-20)
	}

	// **ولا طلبَ يسقط ولا يتكرّر بين الصفحتين.**
	//
	// **والترتيبُ بالزمن** — وصفّان في اللحظة نفسِها يتأرجحان بين الصفحتين،
	// فيُقرأ أحدهما مرّتين والآخرُ لا يُقرأ. (ولهذا فُرّقت أزمنةُ الإنشاء.)
	seen := map[string]bool{}
	got, gotCustom := 0, 0
	for _, o := range append(append([]orders.Order{}, first.Orders...), second.Orders...) {
		if seen[o.ID] {
			t.Fatalf("الطلبُ %s قُرئ مرّتين — **صفحتان تتداخلان تُريان عملاً لم يقع**", o.ID)
		}
		seen[o.ID] = true
		got++
		if o.Kind == "custom" {
			gotCustom++
			if o.CustomRequest == "" {
				t.Fatalf("طلبٌ خاصٌّ بلا نصّ — **والشاشةُ تعرضه مكانَ اسم المتجر، فيبقى الفراغُ فراغا**")
			}
		}
	}
	if got != n {
		t.Fatalf("قُرئ %d من %d — **والنقصُ صامتٌ: قائمةٌ تعمل وتَنقُص**", got, n)
	}
	// **والخاصُّ فيها** — وضمٌّ صلبٌ للمتجر كان يمحوه من كلّ قراءة.
	if gotCustom != customs {
		t.Fatalf("الخاصُّ %d من %d — **ولا شيءَ يقول إنّه غاب**", gotCustom, customs)
	}
}
