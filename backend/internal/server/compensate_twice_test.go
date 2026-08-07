package server

// **تعويضُ السائق يُدفع مرّتين — بضغطةٍ مكرّرةٍ لا بسباق.**
//
// (فحصُ التغطية ٢٠٢٦-٠٨-٠٨، بقرار المالك: «نعم ابدأ بالخمسة».)
//
// # المرض
//
// `handleCompensateDriver` **لا يحمل مانعَ تكرارٍ من أيّ نوع**:
//
//	SELECT status, driver_id FROM orders WHERE id = $1   ← خارجَ المعاملة، بلا قفل
//	if status != "failed" { رفض }
//	BEGIN → ApplyTx(+amount) → DebitTreasury(-amount) → COMMIT
//
// **ولا شيءَ يُعلَّم**: الطلبُ يبقى `failed` بعد التعويض كما كان قبله. فمن
// نادى مرّتين — **ضغطةٌ مكرّرةٌ أو شبكةٌ أعادت الإرسال** — دُفع التعويضُ
// مرّتين وخُصمت الخزينةُ مرّتين.
//
// **وليس سباقاً**: نداءان متتاليان يكفيان. **والسباقُ يزيده سوءاً فقط.**
//
// # والمشروعُ يحرس نظيرَه
//
// `settleMerchant` تفحص `EXISTS(... ref = orderID AND kind = 'merchant_earning')`
// قبل أن تقيّد. **فالعادةُ موجودةٌ — وهذا المسلكُ خارجَها.**
//
// # ولا يظهر في سجلّ
//
// كلا النداءين يردّ ٢٠٠، وكلا القيدين صالح، **ودفترُ التدقيق يسجّل تعويضين
// كأنّهما قرارا إدارةٍ منفصلان.**

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

func TestCompensateDriverOnlyOnce(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := realtime.NewHub(quiet)
	store := settings.NewStore(pool)
	walletSvc := wallet.NewService(pool)

	srv := &Server{
		pg:       pool,
		logger:   quiet,
		hub:      hub,
		wallet:   walletSvc,
		settings: store,
		notify:   notifications.New(pool, hub, quiet),
		cashbox:  cashbox.NewService(pool, store),
		orders:   orders.NewService(pool, nil, walletSvc, cashbox.NewService(pool, store), nil, quiet),
	}

	admin := testdb.NewUser(t, pool, "admin")
	driver := testdb.NewUser(t, pool, "driver")
	customer := testdb.NewUser(t, pool, "customer")

	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	var merchantID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent)
		VALUES ('متجرُ اختبار التعويض', $1, 10) RETURNING id`, categoryID).Scan(&merchantID); err != nil {
		t.Fatalf("تعذّر إنشاء متجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, merchantID)
	})

	var orderID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due)
		VALUES ($1, $2, $3, 'failed', 'عنوانُ اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 10000, 2000, 12000, 0, 12000)
		RETURNING id`, customer, merchantID, driver).Scan(&orderID); err != nil {
		t.Fatalf("تعذّر إنشاء طلبٍ فاشل: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, orderID)
	})

	const amount int64 = 3000
	call := func() int {
		body := `{"amount":` + itoa64(amount) + `,"note":"تعويضٌ عن طلبٍ فشل"}`
		req := httptest.NewRequest(http.MethodPost, "/admin/orders/"+orderID+"/compensate-driver",
			strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rc := chi.NewRouteContext()
		rc.URLParams.Add("id", orderID)
		c := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
		c = context.WithValue(c, ctxUserID, admin)
		c = context.WithValue(c, ctxRoles, []string{"admin", "finance"})
		w := httptest.NewRecorder()
		srv.handleCompensateDriver(w, req.WithContext(c))
		return w.Code
	}

	first := call()
	if first != http.StatusOK {
		t.Fatalf("رُفض التعويضُ الأوّل برمز %d", first)
	}
	second := call()

	var paid int
	var credited int64
	if err := pool.QueryRow(ctx, `
		SELECT count(*), COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE user_id = $1 AND kind = 'compensation' AND ref = $2`,
		driver, orderID).Scan(&paid, &credited); err != nil {
		t.Fatalf("تعذّر عدُّ القيود: %v", err)
	}

	if paid != 1 || credited != amount {
		t.Fatalf("**دُفع التعويضُ %d مرّةً بمجموع %d** — والمنتظَر مرّةً واحدةً و%d.\n"+
			"ردُّ النداء الثاني: %d\n"+
			"**وليس سباقاً**: نداءان متتاليان. **ولا شيءَ يُعلَّم في الطلب** — يبقى "+
			"`failed` بعد التعويض كما كان قبله.\n"+
			"**والعلاجُ ما تفعله `settleMerchant`**: فحصُ وجودِ قيدٍ بالمرجع نفسِه "+
			"داخلَ معاملةٍ قفلت صفَّ الطلب.",
			paid, credited, amount, second)
	}
	if second == http.StatusOK {
		t.Fatalf("النداءُ الثاني ردّ ٢٠٠ ولم يُقيَّد شيء — **ونجاحٌ صامتٌ يُقرأ تعويضاً ثانياً وقع.**")
	}

	// ══════════════════════════════════════════════════════════════════
	// **والمتزامنون كذلك — وهو ما يحرسه القفلُ لا الفحص.**
	//
	// **فحصٌ بلا قفلٍ يمرّ عليه اثنان معاً**: يقرآن «لا تعويضَ بعد» كلاهما
	// فيقيّدان. **ولا يظهر ذلك في نداءين متتاليين** — ولذلك جولةٌ ثانية.
	// ══════════════════════════════════════════════════════════════════
	var second2 string
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due)
		VALUES ($1, $2, $3, 'failed', 'عنوانُ اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 10000, 2000, 12000, 0, 12000)
		RETURNING id`, customer, merchantID, driver).Scan(&second2); err != nil {
		t.Fatalf("تعذّر إنشاء طلبٍ ثانٍ: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, second2)
	})

	// **وأربعةٌ لا ثمانية.**
	//
	// كلُّ نداءٍ يمسك اتّصالاً بمعاملةٍ تنتظر قفلَ صفّ الطلب — **وثمانيةٌ
	// تستنزف مجمّعَ الاتّصالات فيعلّق الاختبارُ عشرَ دقائق.** وهو استنزافُ
	// مجمّعٍ لا عطبُ محرّك، **لكنّ اختباراً يعلّق لا يُقرأ.**
	//
	// **وأربعةٌ تكفي**: بلا قفلٍ يمرّ اثنان معاً — وهو ما يُقاس.
	const n = 4
	var start, done sync.WaitGroup
	start.Add(1)
	for i := 0; i < n; i++ {
		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait()
			body := `{"amount":` + itoa64(amount) + `,"note":"تعويضٌ متزامن"}`
			req := httptest.NewRequest(http.MethodPost, "/admin/orders/"+second2+"/compensate-driver",
				strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rc := chi.NewRouteContext()
			rc.URLParams.Add("id", second2)
			c, cancel := context.WithTimeout(req.Context(), 25*time.Second)
			defer cancel()
			c = context.WithValue(c, chi.RouteCtxKey, rc)
			c = context.WithValue(c, ctxUserID, admin)
			c = context.WithValue(c, ctxRoles, []string{"admin", "finance"})
			srv.handleCompensateDriver(httptest.NewRecorder(), req.WithContext(c))
		}()
	}
	start.Done()
	done.Wait()

	var concurrent int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM wallet_transactions
		WHERE user_id = $1 AND kind = 'compensation' AND ref = $2`,
		driver, second2).Scan(&concurrent); err != nil {
		t.Fatalf("تعذّر عدُّ القيود المتزامنة: %v", err)
	}
	if concurrent != 1 {
		t.Fatalf("**ثمانيةُ نداءاتٍ متزامنةٍ دفعت %d تعويضاً** — والمنتظَر واحد. "+
			"**والقفلُ `FOR UPDATE` هو ما يحرس هذا** — لا الفحصُ وحدَه: فحصٌ بلا "+
			"قفلٍ يمرّ عليه اثنان معاً.", concurrent)
	}
}
