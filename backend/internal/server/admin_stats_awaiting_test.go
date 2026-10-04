package server

// **«بانتظار قرارك» — كلُّ بطاقةٍ تعدّ ما ينتظر فعلاً.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٣: أعلى رئيسيّة اللوحة بطاقاتٌ تُفتح كلٌّ على
// صفحتها مرشَّحة.)
//
// # وما يُحرَس
//
// **كلُّ رقمٍ يزيد حين يقع ما يعدّه** — تعويضٌ معلَّق · بلاغُ سائقٍ في
// الطريق · طارئٌ مفتوح · طلبٌ في الطابور بعد مهلة الإسناد · طلبُ انضمام ·
// صنفٌ ينتظر المراجعة · سائقٌ بلغ سقفَ النقد.
//
// **وبلاغٌ أجاب عنه المكتبُ بأمرٍ لا ينتظر أحداً** — فينقص العدُّ.
//
// **والقاعدةُ مشتركةٌ مع غيره** — فيُقاس الفرقُ قبلُ وبعدُ لا الرقمُ المطلق.

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

func TestAdminStatsAwaitingDecision(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	store := settings.NewStore(pool)
	walletSvc := wallet.NewService(pool)
	srv := &Server{
		pg:       pool,
		logger:   quiet,
		wallet:   walletSvc,
		settings: store,
		cashbox:  cashbox.NewService(pool, store),
		orders:   orders.NewService(pool, nil, walletSvc, cashbox.NewService(pool, store), nil, quiet),
	}

	read := func() map[string]int64 {
		t.Helper()
		w := httptest.NewRecorder()
		srv.handleAdminStats(w, httptest.NewRequest(http.MethodGet, "/admin/stats", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("الأرقامُ ردّت %d: %s", w.Code, w.Body.String())
		}
		var env struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatalf("ردٌّ لا يُقرأ: %v", err)
		}
		out := map[string]int64{}
		for k, v := range env.Data {
			if f, ok := v.(float64); ok {
				out[k] = int64(f)
			}
		}
		return out
	}

	before := read()
	for _, k := range []string{"compensations_pending", "reports_waiting", "emergencies_open",
		"orders_unassigned", "leads_new", "menu_pending", "drivers_over_cash",
		"payouts_pending", "tickets_open"} {
		if _, ok := before[k]; !ok {
			t.Fatalf("الحقلُ %q غائبٌ عن الردّ — **وبطاقتُه في الرئيسيّة تقرأ صفراً كاذباً**", k)
		}
	}

	driver := testdb.NewUser(t, pool, "driver")
	customer := testdb.NewUser(t, pool, "customer")

	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	var merchantID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent)
		VALUES ('متجرُ اختبار بانتظار قرارك', $1, 10) RETURNING id`, categoryID).Scan(&merchantID); err != nil {
		t.Fatalf("تعذّر إنشاء متجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM menu_items WHERE merchant_id = $1`, merchantID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM menu_sections WHERE merchant_id = $1`, merchantID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, merchantID)
	})

	newOrder := func(status string, withDriver bool, extra string) string {
		t.Helper()
		var drv any
		if withDriver {
			drv = driver
		}
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
				payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due,
				snap_merchant_commission_percent, snap_rep_commission_percent,
				snap_commission_source, snap_activation_orders)
			VALUES ($1, $2, $3, $4, 'عنوانُ اختبار',
				ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
				'cash', 10000, 2000, 12000, 0, 12000,
				`+qaSnapSQL()+`)
			RETURNING id`, customer, merchantID, drv, status).Scan(&id); err != nil {
			t.Fatalf("تعذّر إنشاءُ طلبٍ %s: %v", status, err)
		}
		if extra != "" {
			if _, err := pool.Exec(ctx, `UPDATE orders SET `+extra+` WHERE id = $1`, id); err != nil {
				t.Fatalf("تعذّر ضبطُ الطلب: %v", err)
			}
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM driver_emergencies WHERE order_id = $1`, id)
			_, _ = pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id)
		})
		return id
	}

	// ── تعويضٌ معلَّق ─────────────────────────────────────────────
	failed := newOrder("failed", true, "")
	if _, err := pool.Exec(ctx, `
		INSERT INTO driver_compensation_requests (order_id, driver_id, fault, suggested_amount)
		VALUES ($1, $2, 'customer', 1000)`, failed, driver); err != nil {
		t.Fatalf("تعذّر طلبُ التعويض: %v", err)
	}

	// ── بلاغُ سائقٍ في الطريق ─────────────────────────────────────
	code := ""
	for _, r := range orders.StageReports {
		if r.At == orders.StOnTheWay {
			code = r.Code
			break
		}
	}
	if code == "" {
		t.Fatalf("لا بلاغَ في الطريق في القائمة — **والبطاقةُ لا تعدّ شيئاً**")
	}
	onWay := newOrder(orders.StOnTheWay, true, "")
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_log (actor_user_id, action, entity, entity_id, details)
		VALUES ($1, 'driver.stage_report', 'order', $2,
		        jsonb_build_object('code', $3::text, 'status', $4::text, 'note', ''))`,
		driver, onWay, code, orders.StOnTheWay); err != nil {
		t.Fatalf("تعذّر البلاغ: %v", err)
	}

	// ── طارئٌ مفتوح ───────────────────────────────────────────────
	if _, err := pool.Exec(ctx, `
		INSERT INTO driver_emergencies (driver_id, order_id, note) VALUES ($1, $2, 'اختبار')`,
		driver, failed); err != nil {
		t.Fatalf("تعذّر الطارئ: %v", err)
	}

	// ── في الطابور بلا سائقٍ منذ ساعة ─────────────────────────────
	newOrder("dispatching", false, "dispatched_at = now() - interval '1 hour'")

	// ── طلبُ انضمام ───────────────────────────────────────────────
	var leadID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchant_leads (store_name, phone) VALUES ('متجرٌ يطلب الانضمام', '+963900000000')
		RETURNING id`).Scan(&leadID); err != nil {
		t.Fatalf("تعذّر طلبُ الانضمام: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchant_leads WHERE id = $1`, leadID)
	})

	// ── صنفٌ ينتظر المراجعة ──────────────────────────────────────
	var sectionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name, sort_order) VALUES ($1, 'قسمُ مراجعة', 1)
		RETURNING id`, merchantID).Scan(&sectionID); err != nil {
		t.Fatalf("تعذّر القسم: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO menu_items (merchant_id, section_id, platform_section_id, name,
		                        merchant_price, price, available, approved)
		VALUES ($1, $2, (SELECT id FROM platform_sections ORDER BY sort_order LIMIT 1),
		        'صنفٌ ينتظر', 20000, 20000, true, false)`, merchantID, sectionID); err != nil {
		t.Fatalf("تعذّر الصنف: %v", err)
	}

	// ── سائقٌ بلغ السقف ───────────────────────────────────────────
	if _, err := pool.Exec(ctx, `
		INSERT INTO driver_cash_entries (driver_id, amount, kind, note)
		VALUES ($1, $2, 'adjustment', 'اختبار السقف')`,
		driver, srv.cashbox.Limit(ctx)); err != nil {
		t.Fatalf("تعذّر قيدُ النقد: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM driver_cash_entries WHERE driver_id = $1`, driver)
	})

	after := read()
	for _, k := range []string{"compensations_pending", "reports_waiting", "emergencies_open",
		"orders_unassigned", "leads_new", "menu_pending", "drivers_over_cash"} {
		if after[k] < before[k]+1 {
			t.Errorf("%s: قبلُ %d وبعدُ %d — **وما وقع لم يُعدّ، فالبطاقةُ تقول «لا شيءَ ينتظرك»**",
				k, before[k], after[k])
		}
	}

	// ── والبطاقةُ تفتح الطلباتِ على ما عدّته ───────────────────────
	page, err := srv.orders.List(ctx, orders.ListFilter{CustomerID: customer, AwaitingOffice: true})
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ المُرشَّح: %v", err)
	}
	if page.Total != 1 || len(page.Orders) != 1 || page.Orders[0].ID != onWay {
		t.Fatalf("مُرشِّحُ «بلاغٌ ينتظر» ردّ %d طلباً — **والمطلوبُ طلبُ البلاغ وحدَه**", page.Total)
	}

	// ── وأمرُ المكتب بعد البلاغ قرارٌ وقع ─────────────────────────
	if _, err := pool.Exec(ctx, `
		UPDATE orders SET door_instruction = 'deliver_now', door_instruction_at = now() + interval '1 second'
		WHERE id = $1`, onWay); err != nil {
		t.Fatalf("تعذّر أمرُ المكتب: %v", err)
	}
	decided := read()
	if decided["reports_waiting"] != after["reports_waiting"]-1 {
		t.Errorf("بعد أمر المكتب: %d وكان %d — **وبلاغٌ أُجيب عنه ما زال يُعدّ منتظراً**",
			decided["reports_waiting"], after["reports_waiting"])
	}
}
