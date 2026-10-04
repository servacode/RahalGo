package server

// **شاشةُ المراقب** (قراراتُ المالك ٢٠٢٦-١٠-٠٤ — قسمُ «مراقبة التشغيل»). وما يُحرَس:
//
//   - **العالقُ في الشاشة = فلترُ «عالق» في لوحة الطلبات** — وكلُّ عدّادِ مرحلةٍ = فلترُه.
//   - **البابان بقدرتين**: سيرُ الطلبات لـ`orders.read` (موظّفُ العمليّات)،
//     والتفاصيلُ التقنيّةُ لـ`observability.read` وحدَها — ولا رقمَ تقنيّاً في الأوّل.
//   - **الحكمُ يرى الإشعاراتِ الفاشلةَ وانتظارَ المسبح** — لا «سليم» لأنّ القاعدةَ أجابت.
//   - **إشعارُ التعطّل بعد المدّة لا قبلها** — مرّةً للانقطاع، ومرّةً حين يعود.
//   - **بطاقةُ «بلا سائق» في الرئيسيّة = فلترُ `no_driver`.**

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/obs"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// seedMonitorOrder **طلبٌ بحالةٍ وعمرٍ** — والعمرُ يحرّك شروطَ العلوق.
func seedMonitorOrder(t *testing.T, srv *Server, status string, ageMin int, dispatchedAgoMin int) string {
	t.Helper()
	pool, ctx := srv.pg, context.Background()
	customer := testdb.NewUser(t, pool, "customer")
	var categoryID, merchantID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent)
		VALUES ('متجرُ اختبار المراقبة', $1, 10) RETURNING id`, categoryID).Scan(&merchantID); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders, created_at, dispatched_at)
		VALUES ($1, $2, $3, 'عنوانُ اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 10000, 2000, 12000, 0, 12000,
			`+qaSnapSQL()+`, now() - make_interval(mins => $4),
			CASE WHEN $5::int >= 0 THEN now() - make_interval(mins => $5::int) END)
		RETURNING id`, customer, merchantID, status, ageMin, dispatchedAgoMin).Scan(&id); err != nil {
		t.Fatalf("طلبٌ %s: %v", status, err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM orders WHERE id = $1`, id)
		_, _ = pool.Exec(bg, `DELETE FROM merchants WHERE id = $1`, merchantID)
	})
	return id
}

func readMonitor(t *testing.T, srv *Server, caps []string) (opsMonitor, string) {
	t.Helper()
	w := callAs(srv.handleOpsMonitor, "GET", "/admin/ops/monitor", "", "", caps)
	raw := dataOf(t, w)
	var m opsMonitor
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("الشاشةُ لا تُقرأ: %v", err)
	}
	return m, string(raw)
}

// ── ١ · العالقُ = فلترُ اللوحة، وكلُّ مرحلةٍ = فلترُها ─────────────────

func TestOpsMonitor_StuckListEqualsBoardFilter(t *testing.T) {
	srv := overviewServer(t)
	l := srv.orders.StuckLimitsOf(context.Background())
	// عالقٌ بلا قبول · بلا سائقٍ بعد مهلته · وطلبٌ حديثٌ ليس عالقاً.
	seedMonitorOrder(t, srv, orders.StPending, l.AcceptMin+30, -1)
	seedMonitorOrder(t, srv, orders.StDispatching, 5, l.DriverMin+20)
	seedMonitorOrder(t, srv, orders.StPending, 0, -1)

	m, _ := readMonitor(t, srv, []string{string(authz.OrdersRead)})
	if len(m.Missing) != 0 {
		t.Fatalf("أقسامٌ لم تُقرأ: %v", m.Missing)
	}
	page := func(filter string) int64 {
		return totalOf(t, callAs(srv.handleListOrders, "GET", "/x?filter="+filter+"&per_page=1", "", "", nil))
	}
	stuck := page(orders.BoardStuck)
	if int64(len(m.Stuck)) != stuck {
		t.Fatalf("**جدولُ العالق %d واللوحةُ بفلتر «عالق» %d** — تعريفان", len(m.Stuck), stuck)
	}
	if stuck < 2 {
		t.Fatalf("ما زُرع عالقاً لم يُعدّ: %d", stuck)
	}
	for _, f := range orders.BoardFilters {
		got := must(t, "مرحلة "+f, m.Flow[f])
		if want := page(f); got != want {
			t.Errorf("عدّادُ %s: الشاشة %d واللوحة %d", f, got, want)
		}
	}
	// **ولا هاتفَ زبونٍ في جدول المراقب.**
	_, raw := readMonitor(t, srv, []string{string(authz.OrdersRead)})
	if strings.Contains(raw, "customer_phone") {
		t.Error("هاتفُ الزبون في ردّ شاشة المراقب")
	}
}

// ── ٢ · البابان بقدرتين ───────────────────────────────────────────────

func TestOpsMonitor_CapabilitySplit(t *testing.T) {
	if need, ok := authz.LookupAdmin("GET", "/ops/monitor"); !ok || need != authz.OrdersRead {
		t.Fatalf("سياسةُ شاشة المراقب %q (%v) — يجب `orders.read`", need, ok)
	}
	if need, ok := authz.LookupAdmin("GET", "/ops/health"); !ok || need != authz.ObservabilityRead {
		t.Fatalf("سياسةُ التفاصيل التقنيّة %q (%v) — يجب `observability.read`", need, ok)
	}
	if _, ok := authz.IsExempt("/ops/status"); !ok {
		t.Fatal("حالُ الخادم للشريط يجب أن يبلغه كلُّ موظّف")
	}
	if need, ok := authz.LookupAdmin("POST", "/orders/{}/alert-ack"); !ok || need != authz.OrdersIntervene {
		t.Fatalf("«أنا عليه» %q (%v) — يجب `orders.intervene`", need, ok)
	}

	srv := overviewServer(t)
	// **موظّفُ العمليّات يملك `orders.read` ولا يملك `observability.read`.**
	var hasRead, hasObs bool
	if err := srv.pg.QueryRow(context.Background(), `
		SELECT bool_or(capability_code = 'orders.read'), bool_or(capability_code = 'observability.read')
		FROM role_capabilities WHERE role_code = 'operations'`).Scan(&hasRead, &hasObs); err != nil {
		t.Fatal(err)
	}
	if !hasRead || hasObs {
		t.Fatalf("قدراتُ العمليّات: orders.read=%v observability.read=%v", hasRead, hasObs)
	}

	// **وبالوسيط نفسِه**: العمليّاتُ تبلغ الشاشةَ ولا تبلغ التفاصيل.
	opsCaps := []string{string(authz.OrdersRead), string(authz.OrdersIntervene)}
	through := func(path string, h http.HandlerFunc) int {
		req := httptest.NewRequest("GET", "/api/v1/admin"+path, nil)
		req = req.WithContext(context.WithValue(req.Context(), ctxCaps, opsCaps))
		w := httptest.NewRecorder()
		srv.enforceAdminPolicy(h).ServeHTTP(w, req)
		return w.Code
	}
	if c := through("/ops/monitor", srv.handleOpsMonitor); c != http.StatusOK {
		t.Fatalf("العمليّاتُ على شاشة المراقب: %d", c)
	}
	if c := through("/ops/health", srv.handleOpsHealth); c != http.StatusForbidden {
		t.Fatalf("**العمليّاتُ بلغت التفاصيلَ التقنيّة**: %d", c)
	}
	if c := through("/ops/status", srv.handleOpsStatus); c != http.StatusOK {
		t.Fatalf("العمليّاتُ على حال الخادم: %d", c)
	}

	// **ولا رقمَ تقنيّاً ولا سببَ في ردّ المراقب** — كلماتٌ وحدَها.
	_, raw := readMonitor(t, srv, opsCaps)
	for _, k := range []string{"pg_pool", "goroutines", "heap_mb", "ping_ms", "empty_acquire_count", "reason\":\"no_answer", "push_failed"} {
		if strings.Contains(raw, k) {
			t.Errorf("**حقلٌ تقنيٌّ في شاشة العمليّات**: %q", k)
		}
	}
	var m opsMonitor
	_ = json.Unmarshal([]byte(raw), &m)
	for _, p := range opsPartsOrder {
		if _, ok := m.System.Parts[p]; !ok {
			t.Errorf("كلمةُ %q غائبةٌ عن حال الخادم", p)
		}
	}
}

// ── ٣ · الحكمُ يرى ما وراء «القاعدةُ أجابت» ───────────────────────────

func TestJudgeOps_DegradedOnFailingPushAndPoolWaits(t *testing.T) {
	healthy := opsJudgeInput{DBUp: true, DBPingMs: 3, CacheWired: true, CacheUp: true,
		CachePingMs: 1, RealtimeOn: true, PoolMax: 10, PoolAcquired: 1, Goroutines: 50}
	if v := judgeOps(healthy); v.State != stOK {
		t.Fatalf("بيئةٌ سليمةٌ حُكم عليها %q: %+v", v.State, v.Parts)
	}

	in := healthy
	in.Delta = opsCounters{pushAttempted: 6, pushFailed: 6}
	v := judgeOps(in)
	if v.State == stOK || v.Parts[partPush].State != stDown || v.Parts[partPush].Reason != whyPushFail {
		t.Fatalf("**الإشعاراتُ كلُّها تفشل والحكمُ %q** — %+v", v.State, v.Parts[partPush])
	}

	in = healthy
	in.Delta = opsCounters{poolEmpty: 68}
	v = judgeOps(in)
	if v.State == stOK || v.Parts[partDB].State != stSlow || v.Parts[partDB].Reason != whyPoolWaits {
		t.Fatalf("**٦٨ انتظاراً على مسبحٍ فارغ والحكمُ %q** — %+v", v.State, v.Parts[partDB])
	}

	in = healthy
	long := int64(3600)
	in.LongestTxSec = &long
	if v := judgeOps(in); v.Parts[partDB].Reason != whyLongTx || v.State == stOK {
		t.Fatalf("معاملةٌ مفتوحةٌ ساعةً والحكمُ %q — %+v", v.State, v.Parts[partDB])
	}

	in = healthy
	in.DBUp = false
	if v := judgeOps(in); v.State != stDown {
		t.Fatalf("القاعدةُ واقفةٌ والحكمُ %q", v.State)
	}
}

// TestOpsHealth_FailingPushDegradesTheEndpoint **والبابُ نفسُه يقول «متدهور».**
func TestOpsHealth_FailingPushDegradesTheEndpoint(t *testing.T) {
	obs.Reset()
	t.Cleanup(obs.Reset)
	srv := overviewServer(t)
	obs.Push(obs.PushAttempted, 4)
	obs.Push(obs.PushFailed, 4)

	var h opsHealth
	if err := json.Unmarshal(dataOf(t, callAs(srv.handleOpsHealth, "GET", "/admin/ops/health", "", "",
		[]string{string(authz.ObservabilityRead)})), &h); err != nil {
		t.Fatal(err)
	}
	if h.Status != "degraded" || h.Verdict.Parts[partPush].State != stDown {
		t.Fatalf("**الإشعاراتُ تفشل والبابُ يقول %q** — %+v", h.Status, h.Verdict.Parts)
	}
	// **والذاكرةُ غيرُ موصولةٍ هنا تُقال «غير معروف» لا «سليم».**
	if h.Verdict.Parts[partCache].State != stUnknown {
		t.Errorf("ذاكرةٌ غيرُ موصولةٍ حُكم عليها %q", h.Verdict.Parts[partCache].State)
	}
	// **ونشاطُ القاعدة مقروءٌ هنا** — فلا `null`.
	if h.DB.Backends == nil || h.DB.LongestTxSeconds == nil {
		t.Errorf("نشاطُ القاعدة لم يُقرأ: %+v", h.DB)
	}
}

// ── ٤ · إشعارُ التعطّل بعد المدّة ────────────────────────────────────

func TestOutage_NotifiesOwnerAfterThreshold(t *testing.T) {
	srv := overviewServer(t)
	srv.notify = notifications.New(srv.pg, srv.hub, srv.logger)
	ctx := context.Background()
	admin := testdb.NewUser(t, srv.pg, "admin")
	ops := testdb.NewUser(t, srv.pg, "operations")
	t.Cleanup(func() {
		_, _ = srv.pg.Exec(context.Background(), `DELETE FROM notifications WHERE entity = 'system' AND user_id IN ($1, $2)`, admin, ops)
	})
	count := func(user string) int {
		t.Helper()
		var n int
		if err := srv.pg.QueryRow(ctx, `SELECT count(*) FROM notifications
			WHERE user_id = $1 AND entity = 'system'`, user).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	limit := srv.outageNotifyMin(ctx)
	if limit != 5 {
		t.Fatalf("المدّةُ الافتراضيّة %d لا ٥", limit)
	}
	bad := opsVerdict{State: stDown, Parts: map[string]opsPart{partDB: {State: stDown, Reason: whyNoAnswer}}}
	good := opsVerdict{State: stOK, Parts: map[string]opsPart{}}
	t0 := time.Now()

	if srv.outageStep(ctx, bad, t0) || srv.outageStep(ctx, bad, t0.Add(4*time.Minute)) {
		t.Fatal("**أُشعر قبل أن تمضي المدّة**")
	}
	if count(admin) != 0 {
		t.Fatalf("إشعارٌ قبل المدّة: %d", count(admin))
	}
	if !srv.outageStep(ctx, bad, t0.Add(time.Duration(limit)*time.Minute)) {
		t.Fatal("**مضت المدّةُ والقاعدةُ واقفة ولم يُشعَر أحد**")
	}
	if count(admin) != 1 {
		t.Fatalf("المديرُ وصله %d لا ١", count(admin))
	}
	if count(ops) != 0 {
		t.Errorf("موظّفُ العمليّات وصله إشعارُ التعطّل — للمالك والمدير وحدَهما")
	}
	var body string
	_ = srv.pg.QueryRow(ctx, `SELECT body FROM notifications WHERE user_id = $1 AND entity = 'system'`, admin).Scan(&body)
	if !strings.Contains(body, outagePartNames[partDB]) {
		t.Errorf("نصُّ الإشعار لا يسمّي الجزءَ الواقف: %q", body)
	}
	if srv.outageStep(ctx, bad, t0.Add(20*time.Minute)) || count(admin) != 1 {
		t.Fatalf("**أُعيد إشعارُ الانقطاع نفسِه**: %d", count(admin))
	}
	// **وعودةُ السلامة تُقال.**
	srv.outageStep(ctx, good, t0.Add(25*time.Minute))
	if count(admin) != 2 {
		t.Fatalf("لم يصل إشعارُ «رجع كل شي شغّال»: %d", count(admin))
	}
	// **وانقطاعٌ جديدٌ يبدأ عدَّه من جديد.**
	if srv.outageStep(ctx, bad, t0.Add(26*time.Minute)) {
		t.Fatal("انقطاعٌ جديدٌ أُشعر به فوراً — العدُّ لم يبدأ من جديد")
	}
}

// ── ٥ · بطاقةُ «بلا سائق» في الرئيسيّة = فلترُ `no_driver` ─────────────

func TestOverview_NoDriverCardEqualsFilter(t *testing.T) {
	srv := overviewServer(t)
	l := srv.orders.StuckLimitsOf(context.Background())
	// **في الطابور منذ قليل** — ليس «بلا سائق» بعد. **وآخرُ تجاوز مهلتَه.**
	seedMonitorOrder(t, srv, orders.StDispatching, 1, 0)
	seedMonitorOrder(t, srv, orders.StDispatching, 5, l.DriverMin+15)

	ov := readOverview(t, srv)
	got := must(t, "بلا سائق", ov.Awaiting.OrdersUnassigned)
	want := totalOf(t, callAs(srv.handleListOrders, "GET", "/x?filter="+orders.BoardNoDriver+"&per_page=1", "", "", nil))
	if got != want {
		t.Fatalf("**بطاقةُ «بلا سائق» %d والقائمةُ التي تفتحها %d**", got, want)
	}
	if want < 1 {
		t.Fatalf("ما تجاوز مهلتَه لم يُعدّ: %d", want)
	}
}
