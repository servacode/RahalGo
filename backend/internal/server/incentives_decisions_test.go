package server

// ══════════════════════════════════════════════════════════════════════
// **قسمُ «الأهداف والمكافآت» — قراراتُ المالك ٢٠٢٦-١٠-٠٤**
// ══════════════════════════════════════════════════════════════════════
//
//	١ · المكافأةُ والعقوبةُ اليدويّة طلبٌ يوافق عليه شخصٌ ثانٍ
//	٢ · المتجرُ يُحسب في هدف المندوب الذي فتحه — للأبد، والنقلُ ينقل العمولةَ وحدَها
//	٣ · طلبٌ مسترجَعٌ أوصل السائقَ لمرحلة ⇒ تنبيهٌ والماليّةُ تقرّر
//	٤ · متجرٌ حُذف أو تجريبيٌّ انحسب للمندوب ⇒ تنبيهٌ والماليّةُ تقرّر
//	٥ · اختيارُ شهرٍ وتصدير
//	٦ · المكافأةُ اليدويّةُ «تقدير» دائماً — لا `for_target`

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/incentives"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

type incFixture struct {
	pool     *pgxpool.Pool
	srv      *Server
	proposer string
	approver string
	// treasury **تُبدَّل في الاختبار** — خزينةٌ معطوبةٌ تُسقط المكافأة.
	treasury string
}

func newIncFixture(t *testing.T) *incFixture {
	t.Helper()
	pool := testdb.Pool(t)
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := realtime.NewHub(quiet)
	walletSvc := wallet.NewService(pool)
	st := settings.NewStore(pool)
	ord := orders.NewService(pool, nil, walletSvc, nil, nil, quiet)
	f := &incFixture{pool: pool}
	f.treasury = ord.TreasuryID(context.Background())
	// **وخزينةٌ قائمةٌ دائماً** — بلاها يُقيَّد الطرفُ الأوّلُ وحدَه فيصيح فحصُ
	// الدفتر (`FI-03.a`) على مالٍ خُلق في الاختبار لا في المنتَج.
	if f.treasury == "" {
		// **ولا تُمحى بعده** — فحصُ الدفتر يطلب خزينةً واحدةً قائمة (`FI-02.c`).
		if err := pool.QueryRow(context.Background(), `
			INSERT INTO users (phone, full_name) VALUES ('+963999000011', 'خزينة اختبار الحوافز')
			ON CONFLICT (phone) DO UPDATE SET full_name = excluded.full_name
			RETURNING id::text`).Scan(&f.treasury); err != nil {
			t.Fatalf("حسابُ الخزينة: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO wallets (user_id, balance, is_treasury) VALUES ($1, 0, true)
			ON CONFLICT (user_id) DO UPDATE SET is_treasury = true`, f.treasury); err != nil {
			t.Fatalf("الخزينة: %v", err)
		}
	}
	inc := incentives.New(pool, walletSvc, st, func(context.Context) string { return f.treasury })
	f.srv = &Server{
		pg: pool, logger: quiet, hub: hub, wallet: walletSvc, orders: ord,
		settings: st, notify: notifications.New(pool, hub, quiet), incentives: inc,
	}
	f.proposer = testdb.NewUser(t, pool, "finance")
	f.approver = testdb.NewUser(t, pool, "finance")
	return f
}

// kept **صاحبُ مالٍ لا يُمحى بعد الاختبار** — محوُ حسابه يمحو طرفَ قيده ويُبقي
// طرفَ الخزينة، فيصيح فحصُ الدفتر (`FI-03.a`) على ما صنعه الاختبار.
func (f *incFixture) kept(t *testing.T, role string) string {
	t.Helper()
	ctx := context.Background()
	var id string
	var err error
	for i := 0; i < 50; i++ {
		err = f.pool.QueryRow(ctx, `
			INSERT INTO users (phone, full_name)
			VALUES ('+9639' || lpad((nextval('test_phone_seq') % 100000000)::text, 8, '0'), $1)
			ON CONFLICT (phone) DO NOTHING
			RETURNING id::text`, "حوافز "+role).Scan(&id)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("حساب: %v", err)
	}
	if _, err := f.pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_code) VALUES ($1, $2) ON CONFLICT DO NOTHING`, id, role); err != nil {
		t.Fatalf("الدور: %v", err)
	}
	return id
}

func (f *incFixture) setTargets(t *testing.T, kv map[string]string) {
	t.Helper()
	ctx := context.Background()
	for k, v := range kv {
		if _, err := f.pool.Exec(ctx, `
			INSERT INTO app_settings (key, value) VALUES ($1, $2::jsonb)
			ON CONFLICT (key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
			t.Fatalf("ضبطُ %s: %v", k, err)
		}
	}
	t.Cleanup(func() {
		for k := range kv {
			_, _ = f.pool.Exec(context.Background(), `DELETE FROM app_settings WHERE key = $1`, k)
		}
	})
}

func (f *incFixture) call(actor, method, path string, params map[string]string, body string,
	h http.HandlerFunc) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	for k, v := range params {
		rc.URLParams.Add(k, v)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, actor)
	ctx = context.WithValue(ctx, ctxRoles, []string{"finance"})
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	return w
}

func (f *incFixture) balance(t *testing.T, uid string) int64 {
	t.Helper()
	var b int64
	if err := f.pool.QueryRow(context.Background(),
		`SELECT COALESCE((SELECT balance FROM wallets WHERE user_id = $1), 0)`, uid).Scan(&b); err != nil {
		t.Fatalf("الرصيد: %v", err)
	}
	return b
}

func (f *incFixture) newStore(t *testing.T, rep string) string {
	return f.newNamedStore(t, rep, "متجر حوافز")
}

func (f *incFixture) newNamedStore(t *testing.T, rep, name string) string {
	t.Helper()
	ctx := context.Background()
	owner := testdb.NewUser(t, f.pool, "merchant")
	var cat string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO categories (name, active) VALUES ('تصنيف حوافز ' || gen_random_uuid()::text, true)
		RETURNING id::text`).Scan(&cat); err != nil {
		t.Fatalf("تصنيف: %v", err)
	}
	var id string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, owner_user_id, sales_rep_user_id, category_id, status, location)
		VALUES ($3, $1, $2, $4::uuid, 'active',
		        ST_SetSRID(ST_MakePoint(39.0, 35.95), 4326)::geography)
		RETURNING id::text`, owner, rep, name, cat).Scan(&id); err != nil {
		t.Fatalf("إنشاءُ متجر: %v", err)
	}
	return id
}

func (f *incFixture) targetRows(t *testing.T, uid string) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM incentives WHERE user_id = $1 AND for_target`, uid).Scan(&n); err != nil {
		t.Fatalf("عدُّ مكافآت الهدف: %v", err)
	}
	return n
}

// TestINC_ManualGrantIsARequestApprovedByAnother **القرار ١ و٦.**
//
// **كان «اعتمد» يُخرج المالَ فوراً بيد شخصٍ واحد** — والآن طلبٌ معلَّقٌ لا يمسّ
// المحفظة، وصاحبُه لا يوافق عليه، والموافقةُ من غيره تقيّده «تقديراً» لا
// «عن الهدف» ولو أُرسل `for_target`.
func TestINC_ManualGrantIsARequestApprovedByAnother(t *testing.T) {
	f := newIncFixture(t)
	driver := f.kept(t, "driver")

	w := f.call(f.proposer, http.MethodPost, "/admin/users/"+driver+"/incentive",
		map[string]string{"id": driver},
		`{"kind":"reward","amount":7000,"reason":"أداءٌ ممتاز","for_target":true}`,
		f.srv.handleIncentiveGrant)
	if w.Code != http.StatusCreated {
		t.Fatalf("رمزُ الاقتراح %d — والمنتظَر 201 طلبٌ معلَّق: %s", w.Code, w.Body.String())
	}
	if b := f.balance(t, driver); b != 0 {
		t.Fatalf("تحرّك الرصيدُ %d قبل أيّ موافقة — **مالٌ بيد شخصٍ واحد.**", b)
	}
	var res struct {
		Data struct {
			RequestID string `json:"request_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Data.RequestID == "" {
		t.Fatalf("لا معرّفَ طلب: %s", w.Body.String())
	}

	self := f.call(f.proposer, http.MethodPost, "/admin/incentive-requests/x/approve",
		map[string]string{"id": res.Data.RequestID}, `{}`, f.srv.handleDecideIncentiveRequest(true))
	if self.Code != http.StatusForbidden {
		t.Fatalf("وافق صاحبُ الاقتراح على نفسه برمز %d", self.Code)
	}

	ok := f.call(f.approver, http.MethodPost, "/admin/incentive-requests/x/approve",
		map[string]string{"id": res.Data.RequestID}, `{}`, f.srv.handleDecideIncentiveRequest(true))
	if ok.Code != http.StatusOK {
		t.Fatalf("رُفضت موافقةُ شخصٍ ثانٍ برمز %d: %s", ok.Code, ok.Body.String())
	}
	if b := f.balance(t, driver); b != 7000 {
		t.Fatalf("الرصيدُ بعد الموافقة %d — والمنتظَر 7000", b)
	}
	if n := f.targetRows(t, driver); n != 0 {
		t.Fatalf("قُيّدت المكافأةُ اليدويّةُ «عن الهدف» (%d) — **وهي تقديرٌ دائماً.**", n)
	}
	var body string
	if err := f.pool.QueryRow(context.Background(), `
		SELECT body FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`,
		driver).Scan(&body); err != nil || !strings.Contains(body, "أداءٌ ممتاز") {
		t.Fatalf("لم يصل صاحبَ المكافأة إشعارٌ بسببها: %q (%v)", body, err)
	}
}

// TestINC_TransferredStoreStaysWithOpener **القرار ٢ — لا مكافأةَ مرّتين على متجرٍ واحد.**
//
// مندوبٌ فتح متجرين فبلغ المرحلة (٢) وقبضها، ثمّ نُقل المتجران لزميله ففتح
// الزميلُ متجراً واحداً. **كان عدّادُ الزميل ٣ فيقبض على متاجرَ قُبض عنها.**
func TestINC_TransferredStoreStaysWithOpener(t *testing.T) {
	f := newIncFixture(t)
	f.setTargets(t, map[string]string{"sales.monthly_target": "2", "sales.target_reward": "3000"})
	ctx := context.Background()
	rep1 := f.kept(t, "sales")
	rep2 := f.kept(t, "sales")

	a := f.newStore(t, rep1)
	b := f.newStore(t, rep1)
	if paid, err := f.srv.incentives.GrantTargetIfReachedTx(ctx, f.pool, rep1, "sales"); err != nil || paid != 3000 {
		t.Fatalf("مكافأةُ الفاتح: %d (%v) — والمنتظَر 3000", paid, err)
	}
	if _, err := f.pool.Exec(ctx,
		`UPDATE merchants SET sales_rep_user_id = $1 WHERE id = ANY($2::uuid[])`,
		rep2, []string{a, b}); err != nil {
		t.Fatalf("النقل: %v", err)
	}
	f.newStore(t, rep2)
	paid, err := f.srv.incentives.GrantTargetIfReachedTx(ctx, f.pool, rep2, "sales")
	if err != nil {
		t.Fatal(err)
	}
	if paid != 0 || f.targetRows(t, rep2) != 0 {
		t.Fatalf("قبض الزميلُ %d على متاجرَ فتحها غيرُه — **مكافأةٌ مرّتين.**", paid)
	}
}

// deliveredOrder **طلبٌ مسلَّمٌ الآن لسائق** — يكفي للعدّ.
func (f *incFixture) deliveredOrder(t *testing.T, driver string) string {
	t.Helper()
	customer := testdb.NewUser(t, f.pool, "customer")
	store := f.newStore(t, f.kept(t, "sales"))
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text,
			dropoff, payment_method, subtotal, delivery_fee, total, cash_due, delivered_at,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
		VALUES ($1, $2, $3, 'delivered', 'عنوان',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 10000, 3000, 13000, 13000, now(),
			`+qaSnapSQL()+`)
		RETURNING id::text`, customer, store, driver).Scan(&id); err != nil {
		t.Fatalf("طلب: %v", err)
	}
	// **ويُمحى بعده** — طلبٌ مسلَّمٌ بلا تسويةٍ يصيح عليه فحصُ الدفتر (`FI-04.a`).
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id)
	})
	return id
}

type incPage struct {
	Data struct {
		Month     string `json:"month"`
		Standings []struct {
			UserID   string `json:"user_id"`
			Done     int    `json:"done"`
			Level    int    `json:"level"`
			Levels   int    `json:"levels"`
			AutoPaid int64  `json:"auto_paid"`
			Balance  int64  `json:"balance"`
		} `json:"standings"`
		Summary struct {
			ReachedPerLevel []int `json:"reached_per_level"`
			AutoPaid        int64 `json:"auto_paid"`
		} `json:"summary"`
		Alerts []struct {
			Kind        string   `json:"kind"`
			IncentiveID string   `json:"incentive_id"`
			FailureID   string   `json:"failure_id"`
			UserID      string   `json:"user_id"`
			Current     int      `json:"current"`
			Refunded    int      `json:"refunded"`
			FakeStores  []string `json:"fake_stores"`
		} `json:"alerts"`
	} `json:"data"`
}

func (f *incFixture) page(t *testing.T, role, month string) incPage {
	t.Helper()
	w := f.call(f.proposer, http.MethodGet, "/admin/incentives/"+role+"?month="+month,
		map[string]string{"role": role}, ``, f.srv.handleIncentiveStandings)
	if w.Code != http.StatusOK {
		t.Fatalf("الصفحة %d: %s", w.Code, w.Body.String())
	}
	var p incPage
	_ = json.Unmarshal(w.Body.Bytes(), &p)
	return p
}

// TestINC_StagesMonthAndSuspended **المرحلة «٢ من ٣» · شهرٌ ماضٍ · الموقوفُ لا يظهر.**
func TestINC_StagesMonthAndSuspended(t *testing.T) {
	f := newIncFixture(t)
	f.setTargets(t, map[string]string{
		"sales.monthly_target": "1", "sales.target_reward": "1000",
		"sales.target_2": "2", "sales.reward_2": "2000",
		"sales.target_3": "5", "sales.reward_3": "3000",
	})
	ctx := context.Background()
	rep := f.kept(t, "sales")
	gone := f.kept(t, "sales")
	f.newStore(t, rep)
	f.newStore(t, rep)
	if paid, err := f.srv.incentives.GrantTargetIfReachedTx(ctx, f.pool, rep, "sales"); err != nil || paid != 3000 {
		t.Fatalf("مكافأةُ مرحلتين: %d (%v)", paid, err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE users SET status = 'suspended' WHERE id = $1`, gone); err != nil {
		t.Fatal(err)
	}

	p := f.page(t, "sales", "")
	var found bool
	for _, s := range p.Data.Standings {
		if s.UserID == gone {
			t.Fatalf("الموقوفُ ظاهرٌ في القائمة")
		}
		if s.UserID == rep {
			found = true
			if s.Level != 2 || s.Levels != 3 || s.Done != 2 || s.AutoPaid != 3000 {
				t.Fatalf("صفُّ المندوب: %+v — والمنتظَر المرحلة ٢ من ٣ وآليٌّ 3000", s)
			}
		}
	}
	if !found {
		t.Fatalf("المندوبُ غائبٌ عن القائمة")
	}
	if len(p.Data.Summary.ReachedPerLevel) != 3 || p.Data.Summary.ReachedPerLevel[1] < 1 {
		t.Fatalf("كروتُ المراحل: %v", p.Data.Summary.ReachedPerLevel)
	}
	// **والشهرُ الماضي يُقرأ** ولا يحمل متاجرَ هذا الشهر.
	past := f.page(t, "sales", "2025-01")
	for _, s := range past.Data.Standings {
		if s.UserID == rep && (s.Done != 0 || s.AutoPaid != 0) {
			t.Fatalf("الشهرُ الماضي يحمل أرقامَ هذا الشهر: %+v", s)
		}
	}
	bad := f.call(f.proposer, http.MethodGet, "/x?month=2099-01", map[string]string{"role": "sales"},
		``, f.srv.handleIncentiveStandings)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("شهرٌ في المستقبل قُبل برمز %d", bad.Code)
	}
	// **والموقوفُ لا يُكافأ آليّاً ولا يُقترح له.**
	f.newStore(t, gone)
	if paid, _ := f.srv.incentives.GrantTargetIfReachedTx(ctx, f.pool, gone, "sales"); paid != 0 {
		t.Fatalf("كوفئ الموقوفُ بـ%d", paid)
	}
	w := f.call(f.proposer, http.MethodPost, "/x", map[string]string{"id": gone},
		`{"kind":"reward","amount":100,"reason":"س"}`, f.srv.handleIncentiveGrant)
	if w.Code != http.StatusConflict {
		t.Fatalf("اقتراحٌ لموقوفٍ برمز %d", w.Code)
	}
}

// TestINC_RefundAndFakeStoreRaiseAlerts **القراران ٣ و٤ — تنبيهٌ والماليّةُ تقرّر.**
func TestINC_RefundAndFakeStoreRaiseAlerts(t *testing.T) {
	f := newIncFixture(t)
	f.setTargets(t, map[string]string{
		"drivers.monthly_target": "2", "drivers.target_reward": "4000",
		"sales.monthly_target": "2", "sales.target_reward": "1500",
	})
	ctx := context.Background()

	driver := f.kept(t, "driver")
	o1 := f.deliveredOrder(t, driver)
	f.deliveredOrder(t, driver)
	if paid := f.srv.incentives.GrantTargetIfReached(ctx, driver, "driver"); paid != 4000 {
		t.Fatalf("مكافأةُ السائق %d", paid)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'refunded' WHERE id = $1`, o1); err != nil {
		t.Fatal(err)
	}
	var incID string
	for _, a := range f.page(t, "driver", "").Data.Alerts {
		if a.UserID == driver && a.Kind == "count_dropped" {
			incID = a.IncentiveID
			if a.Current != 1 || a.Refunded != 1 {
				t.Fatalf("التنبيه: %+v", a)
			}
		}
	}
	if incID == "" {
		t.Fatalf("طلبٌ مسترجَعٌ أنزل السائقَ تحت مرحلته ولا تنبيه")
	}
	// **«تُسترجَع» تفتح طلبَ عقوبةٍ يوافق عليه شخصٌ ثانٍ** — ولا تمسّ المال.
	before := f.balance(t, driver)
	w := f.call(f.proposer, http.MethodPost, "/x", map[string]string{"id": incID},
		`{"decision":"clawback","note":"طلب مسترجع"}`, f.srv.handleDecideIncentiveAlert)
	if w.Code != http.StatusOK {
		t.Fatalf("القرار %d: %s", w.Code, w.Body.String())
	}
	if f.balance(t, driver) != before {
		t.Fatalf("الاسترجاعُ خصم قبل الموافقة")
	}
	var pending int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM incentive_requests
		WHERE user_id = $1 AND kind = 'penalty' AND status = 'pending' AND source = 'alert'`, driver).Scan(&pending)
	if pending != 1 {
		t.Fatalf("طلباتُ الاسترجاع المعلّقة %d", pending)
	}
	for _, a := range f.page(t, "driver", "").Data.Alerts {
		if a.IncentiveID == incID {
			t.Fatalf("التنبيهُ باقٍ بعد القرار")
		}
	}

	rep := f.kept(t, "sales")
	f.newStore(t, rep)
	f.newNamedStore(t, rep, "متجر تجربة")
	if paid, err := f.srv.incentives.GrantTargetIfReachedTx(ctx, f.pool, rep, "sales"); err != nil || paid != 1500 {
		t.Fatalf("مكافأةُ المندوب %d (%v)", paid, err)
	}
	var fake bool
	for _, a := range f.page(t, "sales", "").Data.Alerts {
		if a.UserID == rep && a.Kind == "count_dropped" && len(a.FakeStores) == 1 {
			fake = true
		}
	}
	if !fake {
		t.Fatalf("متجرٌ تجريبيٌّ انحسب للمندوب ولا تنبيه")
	}
}

// TestINC_FailedStoreBonusIsRecordedAndRetried **مكافأةٌ تعثّرت تُسجَّل وتُعاد.**
func TestINC_FailedStoreBonusIsRecordedAndRetried(t *testing.T) {
	f := newIncFixture(t)
	f.setTargets(t, map[string]string{"sales.monthly_target": "1", "sales.target_reward": "2500"})
	ctx := context.Background()
	rep := f.kept(t, "sales")
	good := f.treasury
	// **خزينةٌ معطوبة** — معرّفٌ لا حسابَ له فيسقط القيد.
	f.treasury = "00000000-0000-0000-0000-000000000001"
	m := f.newStore(t, rep)
	f.srv.grantSalesTargetIfAny(ctx, m)
	if b := f.balance(t, rep); b != 0 {
		t.Fatalf("دخل مالٌ رغم العثرة: %d", b)
	}
	var failID string
	for _, a := range f.page(t, "sales", "").Data.Alerts {
		if a.UserID == rep && a.Kind == "grant_failed" {
			failID = a.FailureID
		}
	}
	if failID == "" {
		t.Fatalf("العثرةُ بُلعت — لا تنبيه")
	}
	f.treasury = good
	w := f.call(f.proposer, http.MethodPost, "/x", map[string]string{"id": failID}, ``,
		f.srv.handleRetryIncentiveFailure)
	if w.Code != http.StatusOK {
		t.Fatalf("الإعادة %d: %s", w.Code, w.Body.String())
	}
	if b := f.balance(t, rep); b != 2500 {
		t.Fatalf("الرصيدُ بعد الإعادة %d — والمنتظَر 2500", b)
	}
}
