package orders_test

// **ما يُقفل الطلبَ يعود في ردّه.**
//
// (شهده المالك ٢٠٢٦-٠٨-١٠: «مجرّد ما يتمّ تسليم الطلب ينتقل إلى طلبات
//  سابقة» — وكان يبقى في «جارية» إلى الأبد.)
//
// # ولماذا حقلٌ واحدٌ يستحقّ حارساً
//
// **`closed_at` يُكتب في القاعدة عند كلّ نهاية** — تسليماً وإلغاءً ورفضاً
// وتعذّراً — **ولم يكن في `Order` أصلاً**، فلا في الاستعلام ولا في المسح
// ولا في JSON.
//
// **وحقلٌ غائبٌ من JSON يُقرأ في الواجهة `undefined`** — و`!undefined`
// صوابٌ، **فشرطُ «جارٍ» يصدق على كلّ طلبٍ أبداً** والتبويبان الآخران فارغان
// دائماً.
//
// **ولا خطأ ولا سطرٌ في سجلّ**: شرطٌ يعمل ويُجيب بالخطأ — **وهي أسوأُ
// الأعطاب**، لأنّ من ينظر يرى شاشةً تعمل.

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// TestClosedAt_ComesBackInTheOrder **يُقفل فيُقرأ إقفالُه — وفي JSON أيضاً.**
//
// **والفحصُ على JSON لا على الحقل**: الواجهةُ تقرأ ما يُرسَل، **وحقلٌ في
// البنية بلا وسمٍ صحيحٍ لا يصل إليها.**
func TestClosedAt_ComesBackInTheOrder(t *testing.T) {
	pool := testdb.Pool(t)
	store := settings.NewStore(pool)
	svc := orders.NewService(pool, nil, wallet.NewService(pool),
		cashbox.NewService(pool, store), nil,
		slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	svc.SetSettings(store)
	ctx := context.Background()
	customer := testdb.NewUser(t, pool, "customer")

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders (kind, customer_id, status, address_text, dropoff,
			payment_method, custom_request, subtotal, delivery_fee, total, cash_due)
		VALUES ('custom', $1, 'dispatching', 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 'طلبٌ للفحص', 0, 0, 0, 0)
		RETURNING id::text`, customer).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ طلب: %v", err)
	}

	// **وقبل النهاية فارغ** — وإلّا لَقُرئ كلُّ طلبٍ منتهياً.
	before, err := svc.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ الطلب: %v", err)
	}
	if before.ClosedAt != nil {
		t.Fatalf("طلبٌ جارٍ ومعه وقتُ إقفال — **فيُقرأ منتهياً وهو في الطريق**")
	}

	// **ثمّ يُقفل كما يُقفله المحرّك** — العمودُ نفسُه الذي يكتبه
	// `transitions.go` عند التسليم والإلغاء والرفض والتعذّر.
	//
	// **والعطبُ كان في القراءة لا في الكتابة**: القاعدةُ تحمل الوقتَ في كلّ
	// طلبٍ منتهٍ — **والردُّ لا يحمله.** فهذا ما يُحرَس.
	if _, err := pool.Exec(ctx, `
		UPDATE orders SET status = 'delivered', delivered_at = now(), closed_at = now()
		WHERE id = $1`, id); err != nil {
		t.Fatalf("تعذّر إقفالُ الطلب: %v", err)
	}

	after, err := svc.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ الطلب بعد التسليم: %v", err)
	}
	if after.ClosedAt == nil {
		t.Fatalf("سُلّم الطلبُ ولا وقتَ إقفالٍ في ردّه — **فيبقى في «جارية» إلى الأبد**")
	}

	// **وفي JSON** — وهو ما تقرؤه الشاشة.
	raw, err := json.Marshal(after)
	if err != nil {
		t.Fatalf("تعذّر ترميزُ الطلب: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("تعذّر فكُّ الترميز: %v", err)
	}
	if v, ok := payload["closed_at"]; !ok || v == nil {
		t.Fatalf("لا `closed_at` في حمولة الطلب — **والواجهةُ تقرأ الغائبَ «لم يُقفل»**")
	}
}
