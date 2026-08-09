package orders_test

// **حديثُ الطلب الخاصّ يبدأ بنفسه — ولا يُكتب مرّتين.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «مجرّد ما السائق ياخذ الطلب… رسالةٌ تلقائيّةٌ
//  للسائق بالطلب المطلوب، وردٌّ للمستخدم».)
//
// # ولماذا يُحرَس
//
// **الإسنادُ يقع أكثرَ من مرّة**: يُسحب الطلبُ من سائقٍ فيعود إلى الطابور
// فيأخذه ثانٍ — **وسطرٌ يُكتب في كلّ إسنادٍ يملأ الحديثَ بنسخٍ من الطلب
// نفسِه.**
//
// **ونسختان من نصّ الطلب في حديثٍ واحدٍ تُقرآن طلبين** — فيشتري السائقُ
// مرّتين، أو يسأل «أيُّهما الصحيح؟» في طلبٍ يُقاس بالدقائق.

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

const chatRequest = "فروجة مشوية مع كولا عائليّ وربطة خبز"

// customAtQueue **طلبٌ خاصٌّ في الطابور، وسائقان.**
func customAtQueue(t *testing.T) (*pgxpool.Pool, *orders.Service, string, string, string, string) {
	t.Helper()
	pool := testdb.Pool(t)
	store := settings.NewStore(pool)
	svc := orders.NewService(pool, nil, wallet.NewService(pool),
		cashbox.NewService(pool, store), nil,
		slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	svc.SetSettings(store)
	customer := testdb.NewUser(t, pool, "customer")
	driverA := testdb.NewUser(t, pool, "driver")
	driverB := testdb.NewUser(t, pool, "driver")
	ctx := context.Background()

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders (kind, customer_id, status, address_text, dropoff,
			payment_method, custom_request, subtotal, delivery_fee, total, cash_due)
		VALUES ('custom', $1, 'dispatching', 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', $2, 0, 0, 0, 0)
		RETURNING id::text`, customer, chatRequest).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ طلبٍ خاصّ: %v", err)
	}
	return pool, svc, id, customer, driverA, driverB
}

// take **يأخذ السائقُ الطلبَ كما يأخذه من زرّه.**
func take(t *testing.T, pool *pgxpool.Pool, svc *orders.Service, orderID, driverID string) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`UPDATE orders SET driver_id = $2 WHERE id = $1`, orderID, driverID); err != nil {
		t.Fatalf("تعذّر وضعُ السائق: %v", err)
	}
	if _, err := svc.Transition(ctx, driverID, []string{"driver"},
		orderID, "assigned", ""); err != nil {
		t.Fatalf("تعذّر الإسناد: %v", err)
	}
}

type line struct {
	role string
	body string
}

func chatOf(t *testing.T, pool *pgxpool.Pool, orderID string) []line {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT sender_role, body FROM order_messages
		WHERE order_id = $1 ORDER BY created_at, body`, orderID)
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ الحديث: %v", err)
	}
	defer rows.Close()
	var out []line
	for rows.Next() {
		var l line
		if err := rows.Scan(&l.role, &l.body); err != nil {
			t.Fatalf("تعذّر مسحُ صفّ: %v", err)
		}
		out = append(out, l)
	}
	return out
}

// TestCustomChat_OpensItselfOnAssign **سطران: الطلبُ وجوابُه.**
func TestCustomChat_OpensItselfOnAssign(t *testing.T) {
	pool, svc, order, _, driverA, _ := customAtQueue(t)
	take(t, pool, svc, order, driverA)

	got := chatOf(t, pool, order)
	if len(got) != 2 {
		t.Fatalf("في الحديث %d سطراً والمنتظَر اثنان: %+v", len(got), got)
	}

	// **نصُّ الطلب منسوبٌ إلى صاحبه** — **وهو كلامُه حرفاً بحرف.**
	var sawRequest, sawGreeting bool
	for _, l := range got {
		if l.role == "customer" && l.body == chatRequest {
			sawRequest = true
		}
		if l.role == "driver" && strings.Contains(l.body, "استلمتُ طلبك") {
			sawGreeting = true
		}
	}
	if !sawRequest {
		t.Fatalf("لا نصَّ للطلب في الحديث — **والسائقُ يفتحه فلا يعرف ماذا يشتري**: %+v", got)
	}
	if !sawGreeting {
		t.Fatalf("لا جوابَ للزبون — **يبقى ينتظر خبراً لا يأتي**: %+v", got)
	}
}

// TestCustomChat_NotWrittenTwice **يُسحب الطلبُ ويُعاد فلا يُكرَّر نصُّه.**
func TestCustomChat_NotWrittenTwice(t *testing.T) {
	pool, svc, order, _, driverA, driverB := customAtQueue(t)
	ctx := context.Background()

	take(t, pool, svc, order, driverA)
	// **سُحب منه وعاد إلى الطابور** — ثمّ أخذه ثانٍ.
	if _, err := pool.Exec(ctx, `
		UPDATE orders SET driver_id = NULL, status = 'dispatching' WHERE id = $1`,
		order); err != nil {
		t.Fatalf("تعذّر سحبُ الطلب: %v", err)
	}
	take(t, pool, svc, order, driverB)

	got := chatOf(t, pool, order)
	var requests, greetings int
	for _, l := range got {
		if l.body == chatRequest {
			requests++
		}
		if strings.Contains(l.body, "استلمتُ طلبك") {
			greetings++
		}
	}
	if requests != 1 {
		t.Fatalf("نصُّ الطلب مكتوبٌ %d مرّةً — **ونسختان تُقرآن طلبين فيشتري مرّتين**", requests)
	}
	// **والثاني يُحيّي** — الزبونُ يعرف أنّ الطلبَ صار في يدٍ أخرى.
	if greetings != 2 {
		t.Fatalf("جوابُ السائق %d — والمنتظَر واحدٌ لكلّ سائقٍ أخذ الطلب", greetings)
	}
}
