package orders_test

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// TestCustomChat_FirstLineMasksContacts — **نصُّ الطلب الخاصّ يُكتب أوّلَ سطرٍ في الحديث
// تلقائيّاً** (`openCustomChat`) — **وكان يمرّ بلا حارس**: «هات خبز ورقمي 0991234567»
// تصل السائقَ كما هي. فيُخفى الرقمُ ويُوسَم السطرُ للإدارة (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ١).
func TestCustomChat_FirstLineMasksContacts(t *testing.T) {
	pool := testdb.Pool(t)
	store := settings.NewStore(pool)
	svc := orders.NewService(pool, nil, wallet.NewService(pool),
		cashbox.NewService(pool, store), nil,
		slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	svc.SetSettings(store)
	customer := testdb.NewUser(t, pool, "customer")
	driver := testdb.NewUser(t, pool, "driver")
	ctx := context.Background()

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders (kind, customer_id, status, address_text, dropoff,
			payment_method, custom_request, subtotal, delivery_fee, total, cash_due,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
		VALUES ('custom', $1, 'dispatching', 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', $2, 0, 0, 0, 0,
			`+qaSnapSQLX()+`)
		RETURNING id::text`, customer, "هات ربطة خبز ورقمي 0991234567").Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ طلبٍ خاصّ: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE orders SET driver_id = $2 WHERE id = $1`, id, driver); err != nil {
		t.Fatal(err)
	}
	svc.OpenCustomChatForTest(ctx, id, customer, driver)

	var body, word string
	var flagged, leak bool
	if err := pool.QueryRow(ctx, `
		SELECT body, COALESCE(flag_word, ''), flagged, contact_leak FROM order_messages
		 WHERE order_id = $1 AND sender_role = 'customer' ORDER BY created_at LIMIT 1`, id).
		Scan(&body, &word, &flagged, &leak); err != nil {
		t.Fatalf("أوّلُ سطر: %v", err)
	}
	if body != "هات ربطة خبز ورقمي ***" {
		t.Errorf("**نصُّ الطلب وصل الحديثَ برقمه**: %q", body)
	}
	if !flagged || !leak || !strings.Contains(word, "0991234567") {
		t.Errorf("**السطرُ لم يُوسَم**: flagged=%v leak=%v word=%q", flagged, leak, word)
	}
	// **والطلبُ نفسُه يبقى كما كتبه الزبون** — الأصلُ للإدارة.
	var request string
	if err := pool.QueryRow(ctx, `SELECT custom_request FROM orders WHERE id = $1`, id).Scan(&request); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(request, "0991234567") {
		t.Errorf("**نصُّ الطلب تغيّر في الصفّ**: %q", request)
	}
}
