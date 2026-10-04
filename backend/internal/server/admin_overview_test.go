package server

// **رئيسيّةُ مدير المنصّة — كلُّ رقمٍ يطابق صفحتَه.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤.) وما يُحرَس:
//
//   - **البابُ للمدير وحدَه** — من لا يملك `platform.overview` يُردّ ٤٠٣.
//   - **كلُّ بطاقةٍ في «بانتظار قرارك» تساوي ما تعرضه صفحتُها** حين تُفتح
//     بالرابط نفسِه — بنداء معالِج الصفحة لا بنسخ شرطه.
//   - **صافي المنصّة اليوم = صافي صفحة الأرباح ليوم اليوم.**
//   - **الرقمُ الذي لم يُقرأ `null` واسمُ قسمه في `missing`** — لا صفر.
//   - **«استلمتها» تُخرج الطارئَ من الشريط وتُبقيه مفتوحاً.**

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

func overviewServer(t *testing.T) *Server {
	t.Helper()
	pool := testdb.Pool(t)
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	store := settings.NewStore(pool)
	walletSvc := wallet.NewService(pool)
	cb := cashbox.NewService(pool, store)
	ordersSvc := orders.NewService(pool, nil, walletSvc, cb, nil, quiet)
	return &Server{
		pg: pool, logger: quiet, wallet: walletSvc, settings: store, cashbox: cb,
		orders:  ordersSvc,
		hub:     realtime.NewHub(quiet),
		support: support.NewService(pool, nil, walletSvc),
	}
}

// callAs ينادي معالِجاً بقدراتٍ ومعرّفِ مسار.
func callAs(h http.HandlerFunc, method, target, userID, id string, caps []string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	if id != "" {
		rc.URLParams.Add("id", id)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxCaps, caps)
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	return w
}

func dataOf(t *testing.T, w *httptest.ResponseRecorder) json.RawMessage {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("ردّ %d: %s", w.Code, w.Body.String())
	}
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("ردٌّ لا يُقرأ: %v", err)
	}
	return env.Data
}

// totalOf **العددُ الذي تعرضه الصفحة** — `total` أو `count`.
func totalOf(t *testing.T, w *httptest.ResponseRecorder) int64 {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(dataOf(t, w), &m); err != nil {
		t.Fatalf("ليس كائناً: %v", err)
	}
	for _, k := range []string{"total", "count"} {
		if v, ok := m[k].(float64); ok {
			return int64(v)
		}
	}
	t.Fatalf("لا عددَ في الردّ: %v", m)
	return 0
}

var managerCaps = []string{string(authz.PlatformOverview)}

func readOverview(t *testing.T, srv *Server) overview {
	t.Helper()
	var ov overview
	if err := json.Unmarshal(dataOf(t, callAs(srv.handleAdminOverview, "GET",
		"/admin/overview", "", "", managerCaps)), &ov); err != nil {
		t.Fatalf("الرئيسيّةُ لا تُقرأ: %v", err)
	}
	return ov
}

func must(t *testing.T, label string, n num) int64 {
	t.Helper()
	if n == nil {
		t.Fatalf("%s: «غير معروف» وكان يجب أن يُقرأ", label)
	}
	return *n
}

// ── ١ · البابُ للمدير وحدَه ────────────────────────────────────────────

func TestOverview_ManagerOnly(t *testing.T) {
	if need, ok := authz.LookupAdmin("GET", "/overview"); !ok || need != authz.PlatformOverview {
		t.Fatalf("سياسةُ المسار %q (%v) — **والمالُ كلُّه فيه**", need, ok)
	}
	srv := overviewServer(t)
	// **موظّفُ العمليّات يملك التحليلات ولا يملك الرئيسيّة.**
	w := callAs(srv.handleAdminOverview, "GET", "/admin/overview", "", "",
		[]string{"analytics.read", "orders.read", "finance.read"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("غيرُ المدير ردّ %d لا ٤٠٣", w.Code)
	}
	if w := callAs(srv.handleAdminOverview, "GET", "/admin/overview", "", "", managerCaps); w.Code != http.StatusOK {
		t.Fatalf("المديرُ ردّ %d: %s", w.Code, w.Body.String())
	}
	// **والقدرةُ ممنوحةٌ في القاعدة للأدمن والمالك الأعلى وحدَهما.**
	rows, err := srv.pg.Query(context.Background(),
		`SELECT role_code FROM role_capabilities WHERE capability_code = 'platform.overview' ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var r string
		_ = rows.Scan(&r)
		got = append(got, r)
	}
	if strings.Join(got, ",") != "admin,owner_super_admin" {
		t.Fatalf("حاملو القدرة: %v", got)
	}
}

// ── ٢ · كلُّ رقمٍ يطابق صفحتَه ─────────────────────────────────────────

func TestOverview_CountsMatchTheirPages(t *testing.T) {
	srv := overviewServer(t)
	pool, ctx := srv.pg, context.Background()

	driver := testdb.NewUser(t, pool, "driver")
	customer := testdb.NewUser(t, pool, "customer")
	var categoryID, merchantID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent)
		VALUES ('متجرُ اختبار الرئيسيّة', $1, 10) RETURNING id`, categoryID).Scan(&merchantID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, merchantID)
	})
	newOrder := func(status string, withDriver bool) string {
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
			t.Fatalf("طلبٌ %s: %v", status, err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM driver_compensation_requests WHERE order_id = $1`, id)
			_, _ = pool.Exec(context.Background(), `DELETE FROM driver_emergencies WHERE order_id = $1`, id)
			_, _ = pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id)
		})
		return id
	}
	newOrder(orders.StDispatching, false)
	newOrder(orders.StPreparing, false)
	newOrder(orders.StAssigned, true)
	newOrder(orders.StAtPickup, true)
	onWay := newOrder(orders.StOnTheWay, true)
	failed := newOrder(orders.StFailed, true)
	if _, err := pool.Exec(ctx, `
		INSERT INTO driver_compensation_requests (order_id, driver_id, fault, suggested_amount)
		VALUES ($1, $2, 'customer', 1000)`, failed, driver); err != nil {
		t.Fatal(err)
	}
	var emergencyID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO driver_emergencies (driver_id, order_id, note) VALUES ($1, $2, 'اختبار')
		RETURNING id`, driver, onWay).Scan(&emergencyID); err != nil {
		t.Fatal(err)
	}
	var leadID, payoutID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchant_leads (store_name, phone) VALUES ('متجرٌ يطلب', '+963900000001')
		RETURNING id`).Scan(&leadID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO payout_requests (user_id, amount) VALUES ($1, 5000) RETURNING id`, driver).
		Scan(&payoutID); err != nil {
		t.Fatal(err)
	}
	// **شكوى متأخّرةٌ وأخرى حديثة.**
	var lateTicket, freshTicket string
	for _, x := range []struct {
		dst *string
		age string
	}{{&lateTicket, "5 hours"}, {&freshTicket, "5 minutes"}} {
		if err := pool.QueryRow(ctx, `
			INSERT INTO tickets (customer_id, subject, created_at)
			VALUES ($1, 'شكوى اختبار', now() - $2::interval) RETURNING id`, customer, x.age).Scan(x.dst); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO driver_cash_entries (driver_id, amount, kind, note)
		VALUES ($1, $2, 'adjustment', 'اختبار السقف')`, driver, srv.cashbox.Limit(ctx)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM merchant_leads WHERE id = $1`, leadID)
		_, _ = pool.Exec(bg, `DELETE FROM payout_requests WHERE id = $1`, payoutID)
		_, _ = pool.Exec(bg, `DELETE FROM tickets WHERE id IN ($1, $2)`, lateTicket, freshTicket)
		_, _ = pool.Exec(bg, `DELETE FROM driver_cash_entries WHERE driver_id = $1`, driver)
	})

	ov := readOverview(t, srv)
	if len(ov.Missing) != 0 {
		t.Fatalf("أقسامٌ لم تُقرأ: %v", ov.Missing)
	}
	a := ov.Awaiting
	page := func(h http.HandlerFunc, target string) int64 {
		return totalOf(t, callAs(h, "GET", target, "", "", nil))
	}
	eq := func(label string, got num, want int64) {
		t.Helper()
		if must(t, label, got) != want {
			t.Errorf("%s: الرئيسيّة %d والصفحة %d", label, *got, want)
		}
	}
	eq("بلاغات", a.ReportsWaiting, page(srv.handleListOrders, "/x?awaiting=1&open=1&per_page=1"))
	eq("بلا سائق", a.OrdersUnassigned, page(srv.handleListOrders, "/x?status=dispatching&open=1&per_page=1"))
	eq("طوارئ", a.EmergenciesOpen, page(srv.handleOpenEmergencies, "/x"))
	eq("تعويضات", a.CompensationsPending, page(srv.handlePendingCompensations, "/x"))
	eq("سحوبات", a.PayoutsPending, page(srv.handleAdminPayouts, "/x?status=pending"))
	eq("شكاوى", a.TicketsOpen, page(srv.handleListTickets, "/x?status=unresolved"))
	eq("متأخّرة", a.TicketsLate, page(srv.handleListTickets, "/x?late=1"))
	eq("انضمام", a.LeadsNew, page(srv.handleAdminLeads, "/x?status=new"))
	for _, st := range orders.LiveStages {
		eq("مرحلة "+st, ov.Live.Stages[st], page(srv.handleListOrders, "/x?stage="+st+"&open=1&per_page=1"))
	}

	// **والنقدُ من صفحة النقد** — مجموعُه ومن بلغ السقف.
	var cash struct {
		Holders []cashHolder `json:"holders"`
		Total   int64        `json:"total"`
		Limit   int64        `json:"limit"`
	}
	if err := json.Unmarshal(dataOf(t, callAs(srv.handleCashOutstanding, "GET", "/x", "", "", nil)), &cash); err != nil {
		t.Fatal(err)
	}
	var over int64
	for _, h := range cash.Holders {
		if h.Held >= cash.Limit {
			over++
		}
	}
	eq("فوق السقف", a.DriversOverCash, over)
	eq("نقد مع السائقين", ov.Money.CashHeld, cash.Total)

	// **والعالقُ هو تنبيهاتُ شاشة الطلبات.**
	var alerts []any
	if err := json.Unmarshal(dataOf(t, callAs(srv.handleOrderAlerts, "GET", "/x", "", "", nil)), &alerts); err != nil {
		t.Fatal(err)
	}
	eq("عالق", ov.Live.OrdersStuck, int64(len(alerts)))

	// **والمعدودُ فعلاً** — لا تطابقُ صفرين.
	if *a.TicketsLate < 1 || *a.TicketsOpen < 2 || *a.EmergenciesOpen < 1 || *a.DriversOverCash < 1 {
		t.Fatalf("ما زُرع لم يُعدّ: %+v", a)
	}
}

// ── ٣ · صافي المنصّة = صفحةُ الأرباح ────────────────────────────────────

func TestOverview_ProfitIsTheProfitsPage(t *testing.T) {
	srv := overviewServer(t)
	ov := readOverview(t, srv)
	var p struct {
		Net    int64 `json:"net"`
		Sales  int64 `json:"sales"`
		Losses int64 `json:"losses"`
	}
	w := callAs(srv.handleProfits, "GET", "/x?tab=platform&from="+ov.Today+"&to="+ov.Today, "", "", nil)
	if err := json.Unmarshal(dataOf(t, w), &p); err != nil {
		t.Fatal(err)
	}
	if must(t, "صافي", ov.Money.Net) != p.Net || *ov.Money.Sales != p.Sales || *ov.Money.Losses != p.Losses {
		t.Fatalf("الرئيسيّة (%d·%d·%d) والأرباح (%d·%d·%d)",
			*ov.Money.Net, *ov.Money.Sales, *ov.Money.Losses, p.Net, p.Sales, p.Losses)
	}
	// **ويومُ دمشق لا يومُ غرينتش.**
	want, _, _ := overviewDays(time.Now())
	if ov.Today != want.Format("2006-01-02") {
		t.Fatalf("اليومُ %s لا %s", ov.Today, want.Format("2006-01-02"))
	}
}

// ── ٤ · الرقمُ الغائبُ «غير معروف» لا صفر ──────────────────────────────

func TestOverview_MissingIsUnknownNotZero(t *testing.T) {
	srv := overviewServer(t)
	srv.support = nil // **قسمُ الشكاوى لا يُقرأ**
	ov := readOverview(t, srv)
	if ov.Awaiting.TicketsOpen != nil || ov.Awaiting.TicketsLate != nil {
		t.Fatalf("الشكاوى قُرئت أرقاماً وهي لم تُقرأ")
	}
	found := false
	for _, m := range ov.Missing {
		if m == "tickets" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing لا يذكر الشكاوى: %v", ov.Missing)
	}
	// **والردُّ نفسُه يحمل null لا 0.**
	raw := dataOf(t, callAs(srv.handleAdminOverview, "GET", "/x", "", "", managerCaps))
	if !strings.Contains(string(raw), `"tickets_open":null`) {
		t.Fatalf("الحقلُ لم يُرسَل null")
	}
}

// ── ٥ · «استلمتها» ─────────────────────────────────────────────────────

func TestOverview_AckLeavesBannerButKeepsEmergencyOpen(t *testing.T) {
	srv := overviewServer(t)
	pool, ctx := srv.pg, context.Background()
	driver := testdb.NewUser(t, pool, "driver")
	staff := testdb.NewUser(t, pool, "customer_support")
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO driver_emergencies (driver_id, note) VALUES ($1, 'اختبار الشريط') RETURNING id`,
		driver).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM driver_emergencies WHERE id = $1`, id)
	})
	inBanner := func() bool {
		b, err := srv.emergencyBannerData(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range b.Drivers {
			if d.ID == id {
				return true
			}
		}
		return false
	}
	if !inBanner() {
		t.Fatalf("الطارئُ الجديدُ ليس في الشريط")
	}
	if w := callAs(srv.handleAckEmergency, "POST", "/x", staff, id, nil); w.Code != http.StatusOK {
		t.Fatalf("استلمتها ردّت %d: %s", w.Code, w.Body.String())
	}
	if inBanner() {
		t.Fatalf("الطارئُ المستلَمُ ما زال في الشريط")
	}
	var status string
	_ = pool.QueryRow(ctx, `SELECT status FROM driver_emergencies WHERE id = $1`, id).Scan(&status)
	if status != "open" {
		t.Fatalf("الاستلامُ أغلق الطارئ (%s)", status)
	}
	if w := callAs(srv.handleAckEmergency, "POST", "/x", staff, id, nil); w.Code != http.StatusNotFound {
		t.Fatalf("الاستلامُ الثاني ردّ %d لا ٤٠٤", w.Code)
	}
	if need, ok := authz.LookupAdmin("POST", "/emergencies/{id}/ack"); !ok || need != authz.SupportManage {
		t.Fatalf("سياسةُ الاستلام %q", need)
	}
}
