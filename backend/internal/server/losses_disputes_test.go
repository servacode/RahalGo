package server

// ══════════════════════════════════════════════════════════════════════
// **الخسائر والنزاعات — قراراتُ المالك ٢٠٢٦-١٠-٠٤** (`/dashboard/losses`)
// ══════════════════════════════════════════════════════════════════════
//
//	LD-01  نزاعٌ يدويٌّ على متجرٍ يُفتح بمعرّف المتجر، ومعرّفُ صاحبه يُردّ بكلمة
//	LD-02  الماليّةُ ترى النزاعات وتحسم، والدعمُ يرى ويفتح ولا يحسم
//	LD-03  نزاعٌ محسومٌ لا يكبر — تعويضٌ جديدٌ يفتح نزاعاً جديداً
//	LD-04  الحسمُ اقتراحٌ وموافقةُ شخصٍ آخر، ولا مالَ قبل الموافقة
//	LD-05  رصيدٌ لا يكفي: يُخصم الموجودُ ويبقى الباقي مفتوحاً
//	LD-06  الطرفُ المختارُ يحمل الدورَ المختار
//	LD-07  الكرتان بالترشيح نفسِه · و«انسقط عنه قبل X مرّات»
//	LD-08  الخسائرُ: خرج · رجع · الصافي · ولنا عند الناس — وأيّامُ دمشق
//	LD-09  سببُ نزاع المتجر من طلب التعويض لا من فشل الطلب

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

type ldFixture struct {
	pool       *pgxpool.Pool
	srv        *Server
	proposer   string
	approver   string
	owner      string
	merchantID string
}

func newLDFixture(t *testing.T) *ldFixture {
	t.Helper()
	ctx := context.Background()
	pool := testdb.Pool(t)
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := realtime.NewHub(quiet)
	walletSvc := wallet.NewService(pool)
	// **قاعدةٌ فارغةٌ بلا إداريّ لا خزينةَ لها** — فيُنشأ إداريٌّ قبل ضمانها.
	testdb.NewUser(t, pool, "admin")
	if tid, err := wallet.EnsureTreasury(ctx, pool); err != nil || tid == "" {
		t.Fatalf("لا خزينة: %q %v", tid, err)
	}
	f := &ldFixture{
		pool: pool,
		srv: &Server{
			pg: pool, logger: quiet, hub: hub, wallet: walletSvc,
			orders:   orders.NewService(pool, nil, walletSvc, nil, nil, quiet),
			settings: settings.NewStore(pool),
			notify:   notifications.New(pool, hub, quiet),
		},
		proposer: testdb.NewUser(t, pool, "finance"),
		approver: testdb.NewUser(t, pool, "finance"),
		owner:    testdb.NewUser(t, pool, "merchant"),
	}
	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, owner_user_id)
		VALUES ('متجرُ نزاعات', $1, 10, $2) RETURNING id::text`, categoryID, f.owner).
		Scan(&f.merchantID); err != nil {
		t.Fatalf("تعذّر المتجر: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM disputes WHERE merchant_id = $1`, f.merchantID)
		_, _ = pool.Exec(c, `DELETE FROM merchants WHERE id = $1`, f.merchantID)
	})
	return f
}

func (f *ldFixture) call(actor, method, path string, params map[string]string, body string,
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

// order **طلبٌ فاشلٌ على متجر الفحص** — مرجعٌ لنزاعٍ مولودٍ من طلب.
func (f *ldFixture) order(t *testing.T, failReason string) string {
	t.Helper()
	ctx := context.Background()
	customer := testdb.NewUser(t, f.pool, "customer")
	var id string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, cash_due, fail_reason,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
		VALUES ($1, $2, 'failed', 'عنوانُ اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 10000, 2000, 12000, 12000, $3,
			`+qaSnapSQL()+`)
		RETURNING id::text`, customer, f.merchantID, failReason).Scan(&id); err != nil {
		t.Fatalf("تعذّر الطلب: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `DELETE FROM disputes WHERE order_id = $1`, id)
		_, _ = f.pool.Exec(c, `DELETE FROM driver_compensation_requests WHERE order_id = $1`, id)
		_, _ = f.pool.Exec(c, `DELETE FROM orders WHERE id = $1`, id)
	})
	return id
}

func (f *ldFixture) dispute(t *testing.T, amount int64) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO disputes (party_role, merchant_id, reason, amount)
		VALUES ('merchant', $1, 'بضاعةٌ ناقصة', $2) RETURNING id::text`, f.merchantID, amount).
		Scan(&id); err != nil {
		t.Fatalf("تعذّر النزاع: %v", err)
	}
	return id
}

func (f *ldFixture) fund(t *testing.T, uid string, amount int64) {
	t.Helper()
	if _, err := f.srv.wallet.Apply(context.Background(), uid, amount, "topup", "ld-fund", "رصيدُ فحص", nil); err != nil {
		t.Fatalf("تعذّر الشحن: %v", err)
	}
}

func (f *ldFixture) propose(t *testing.T, disputeID, action string) string {
	t.Helper()
	w := f.call(f.proposer, http.MethodPost, "/x", map[string]string{"id": disputeID},
		`{"action":"`+action+`","note":"سمعنا المتجر"}`, f.srv.handleProposeDisputeResolution)
	if w.Code != http.StatusCreated {
		t.Fatalf("الاقتراحُ ردّ %d: %s", w.Code, w.Body.String())
	}
	var res struct {
		Data struct {
			ID string `json:"resolution_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	return res.Data.ID
}

// ledgerClean **فحوصُ الدفتر على أطراف الفحص** — رصيدٌ يطابق قيودَه، ولا سالبَ إلّا
// الخزينة، ولا قيدَ خزينةٍ لغيرها. **ويُقرأ ما يخصّ أطرافَنا وحدَها** — فالقاعدةُ
// مشتركةٌ مع فحوصٍ أخرى قد تترك أثرَها.
func (f *ldFixture) ledgerClean(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	var treasury string
	_ = f.pool.QueryRow(ctx, `SELECT user_id::text FROM wallets WHERE is_treasury LIMIT 1`).Scan(&treasury)
	vs, err := fininv.Run(ctx, f.pool, "FI-02.a", "FI-02.b", "FI-12.a")
	if err != nil {
		t.Fatalf("تعذّر فحصُ الدفتر: %v", err)
	}
	for _, v := range vs {
		for _, r := range v.Rows {
			line := fmt.Sprint(r...)
			if strings.Contains(line, f.owner) || (treasury != "" && strings.Contains(line, treasury)) {
				t.Errorf("**خرقُ دفتر**: %s — %s", v.Check.ID, line)
			}
		}
	}
}

func (f *ldFixture) row(t *testing.T, id string) (status string, amount, recovered int64) {
	t.Helper()
	if err := f.pool.QueryRow(context.Background(),
		`SELECT status, amount, recovered FROM disputes WHERE id = $1`, id).
		Scan(&status, &amount, &recovered); err != nil {
		t.Fatalf("تعذّرت قراءةُ النزاع: %v", err)
	}
	return
}

// LD-01 + LD-06 — **الطرفُ بدوره وبمعرّفه الصحيح.**
func TestLD_ManualDisputePartyByRole(t *testing.T) {
	f := newLDFixture(t)
	open := func(role, id string) *httptest.ResponseRecorder {
		return f.call(f.proposer, http.MethodPost, "/x", nil,
			`{"party_role":"`+role+`","party_id":"`+id+`","amount":1500,"reason":"صندوقٌ لم يُسلَّم"}`,
			f.srv.handleCreateDispute)
	}
	// **متجرٌ بمعرّف المتجر يُفتح.**
	if w := open("merchant", f.merchantID); w.Code != http.StatusCreated {
		t.Fatalf("نزاعٌ على متجرٍ بمعرّفه ردّ %d: %s", w.Code, w.Body.String())
	}
	// **ومعرّفُ صاحبه في خانة المتجر يُردّ بكلمةٍ لا بخطأ قاعدة.**
	if w := open("merchant", f.owner); w.Code != http.StatusBadRequest || errCode(t, w) != "dispute_party_mismatch" {
		t.Fatalf("معرّفُ صاحب المتجر ردّ %d: %s", w.Code, w.Body.String())
	}
	// **وسائقٌ لا يُسجَّل زبوناً.**
	driver := testdb.NewUser(t, f.pool, "driver")
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM disputes WHERE party_user_id = $1`, driver)
	})
	if w := open("customer", driver); w.Code != http.StatusBadRequest || errCode(t, w) != "dispute_party_mismatch" {
		t.Fatalf("سائقٌ سُجّل زبوناً: %d %s", w.Code, w.Body.String())
	}
	if w := open("driver", driver); w.Code != http.StatusCreated {
		t.Fatalf("سائقٌ بدوره ردّ %d: %s", w.Code, w.Body.String())
	}
	// **وقائمةُ الأطراف بالدور**: المتاجرُ بمعرّف المتجر.
	w := f.call(f.proposer, http.MethodGet, "/x?role=merchant&q=نزاعات", nil, "", f.srv.handleDisputeParties)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), f.merchantID) {
		t.Fatalf("قائمةُ المتاجر لا تحمل معرّفَ المتجر: %d %s", w.Code, w.Body.String())
	}
}

// LD-02 — **الماليّةُ ترى وتحسم، والدعمُ يرى ويفتح ولا يحسم.**
func TestLD_DisputePermissionsByDecision(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	need := func(method, path string) authz.Capability {
		c, ok := authz.LookupAdmin(method, path)
		if !ok {
			t.Fatalf("لا سياسةَ لـ %s %s", method, path)
		}
		return c
	}
	has := func(role string, c authz.Capability) bool {
		var ok bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM role_capabilities
			WHERE role_code = $1 AND capability_code = $2)`, role, string(c)).Scan(&ok); err != nil {
			t.Fatal(err)
		}
		return ok
	}
	see, open := need("GET", "/disputes"), need("POST", "/disputes")
	propose := need("POST", "/disputes/{id}/propose")
	approve := need("POST", "/dispute-resolutions/{id}/approve")
	for _, role := range []string{"finance", "customer_support"} {
		if !has(role, see) || !has(role, open) {
			t.Errorf("**%s لا يرى النزاعات أو لا يفتحها** (%s · %s)", role, see, open)
		}
	}
	if !has("finance", propose) || !has("finance", approve) {
		t.Errorf("**الماليّةُ لا تحسم** (%s · %s)", propose, approve)
	}
	if has("customer_support", propose) || has("customer_support", approve) {
		t.Errorf("**الدعمُ يحسم** — والقرارُ أنّه يرى ويفتح فقط")
	}
}

// LD-03 — **نزاعٌ محسومٌ لا يكبر: تعويضٌ جديدٌ يفتح نزاعاً جديداً.**
func TestLD_SettledDisputeNeverGrows(t *testing.T) {
	f := newLDFixture(t)
	ctx := context.Background()
	orderID := f.order(t, "merchant_closed")
	if err := f.srv.orders.OpenMerchantClaimTx(ctx, f.pool, orderID, 5_000); err != nil {
		t.Fatalf("تعذّرت المطالبة: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `
		UPDATE disputes SET status = 'settled', settlement = 'charged', settled_at = now()
		WHERE order_id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	if err := f.srv.orders.OpenMerchantClaimTx(ctx, f.pool, orderID, 5_000); err != nil {
		t.Fatalf("تعذّرت المطالبةُ الثانية: %v", err)
	}
	var settled, open int64
	if err := f.pool.QueryRow(ctx, `
		SELECT COALESCE(sum(amount) FILTER (WHERE status = 'settled'), 0),
		       COALESCE(sum(amount) FILTER (WHERE status = 'open'), 0)
		FROM disputes WHERE order_id = $1`, orderID).Scan(&settled, &open); err != nil {
		t.Fatal(err)
	}
	if settled != 5_000 || open != 5_000 {
		t.Fatalf("**المحسومُ %d والمفتوحُ %d** — والمتوقّعُ ٥٠٠٠ محسوماً ونزاعٌ جديدٌ مفتوحٌ بـ٥٠٠٠", settled, open)
	}
}

// LD-04 — **اقتراحٌ ثمّ موافقةُ غيرِ المقترِح، ولا مالَ قبلها.**
func TestLD_ResolutionNeedsAnotherApprover(t *testing.T) {
	f := newLDFixture(t)
	f.fund(t, f.owner, 10_000)
	id := f.dispute(t, 4_000)
	res := f.propose(t, id, "charge")

	if st, _, rec := f.row(t, id); st != "open" || rec != 0 {
		t.Fatalf("**تحرّك مالٌ بالاقتراح وحدَه**: %s · %d", st, rec)
	}
	if bal, _ := f.srv.wallet.Balance(context.Background(), f.owner); bal != 10_000 {
		t.Fatalf("خُصم قبل الموافقة: %d", bal)
	}
	// **واقتراحٌ ثانٍ على نزاعٍ معلَّقٍ يُردّ.**
	if w := f.call(f.approver, http.MethodPost, "/x", map[string]string{"id": id},
		`{"action":"waive","note":"x"}`, f.srv.handleProposeDisputeResolution); w.Code != http.StatusConflict {
		t.Fatalf("اقتراحٌ ثانٍ ردّ %d", w.Code)
	}
	// **وصاحبُ الاقتراح لا يوافق عليه** — وفي المنصّة ماليٌّ آخر.
	if w := f.call(f.proposer, http.MethodPost, "/x", map[string]string{"id": res}, `{}`,
		f.srv.handleDecideDisputeResolution(true)); w.Code != http.StatusForbidden || errCode(t, w) != "self_approve" {
		t.Fatalf("وافق المقترِحُ على نفسه: %d %s", w.Code, w.Body.String())
	}
	if w := f.call(f.approver, http.MethodPost, "/x", map[string]string{"id": res}, `{}`,
		f.srv.handleDecideDisputeResolution(true)); w.Code != http.StatusOK {
		t.Fatalf("الموافقةُ ردّت %d: %s", w.Code, w.Body.String())
	}
	if st, _, rec := f.row(t, id); st != "settled" || rec != 4_000 {
		t.Fatalf("بعد الموافقة: %s · %d", st, rec)
	}
	if bal, _ := f.srv.wallet.Balance(context.Background(), f.owner); bal != 6_000 {
		t.Fatalf("رصيدُ صاحب المتجر %d والمتوقّع ٦٠٠٠", bal)
	}
	// **والإسقاطُ كذلك اقتراحٌ وموافقة** — ولا حركةَ مال.
	id2 := f.dispute(t, 2_500)
	res2 := f.propose(t, id2, "waive")
	if w := f.call(f.approver, http.MethodPost, "/x", map[string]string{"id": res2}, `{}`,
		f.srv.handleDecideDisputeResolution(true)); w.Code != http.StatusOK {
		t.Fatalf("موافقةُ الإسقاط ردّت %d: %s", w.Code, w.Body.String())
	}
	if st, _, _ := f.row(t, id2); st != "waived" {
		t.Fatalf("الإسقاطُ لم يُكتب: %s", st)
	}
	f.ledgerClean(t)
}

// LD-05 — **رصيدٌ لا يكفي: يُخصم الموجودُ ويبقى الباقي مفتوحاً.**
func TestLD_PartialChargeLeavesRemainderOpen(t *testing.T) {
	f := newLDFixture(t)
	f.fund(t, f.owner, 3_000)
	id := f.dispute(t, 5_000)
	res := f.propose(t, id, "charge")
	w := f.call(f.approver, http.MethodPost, "/x", map[string]string{"id": res}, `{}`,
		f.srv.handleDecideDisputeResolution(true))
	if w.Code != http.StatusOK {
		t.Fatalf("الموافقةُ ردّت %d: %s — **والرصيدُ ٣٠٠٠ يُخصم لا يُردّ**", w.Code, w.Body.String())
	}
	if st, _, rec := f.row(t, id); st != "open" || rec != 3_000 {
		t.Fatalf("بعد الخصم الجزئيّ: %s · %d — والمتوقّع مفتوحٌ بمسترَدٍّ ٣٠٠٠", st, rec)
	}
	if bal, _ := f.srv.wallet.Balance(context.Background(), f.owner); bal != 0 {
		t.Fatalf("الرصيدُ %d بعد الخصم", bal)
	}
	// **ورصيدٌ صفرٌ لا شيءَ يُخصم منه** — كلمةٌ لا موافقةٌ فارغة.
	res2 := f.propose(t, id, "charge")
	if w := f.call(f.approver, http.MethodPost, "/x", map[string]string{"id": res2}, `{}`,
		f.srv.handleDecideDisputeResolution(true)); w.Code != http.StatusConflict ||
		errCode(t, w) != "dispute_nothing_to_charge" {
		t.Fatalf("رصيدٌ صفرٌ ردّ %d: %s", w.Code, w.Body.String())
	}
	// **ويعود المالُ إلى الخزينة بمرجع النزاع.**
	var back int64
	if err := f.pool.QueryRow(context.Background(), `
		SELECT COALESCE(sum(t.amount), 0) FROM wallet_transactions t
		JOIN wallets w ON w.user_id = t.user_id
		WHERE w.is_treasury AND t.kind = 'platform_profit' AND t.ref = $1`, id).Scan(&back); err != nil {
		t.Fatal(err)
	}
	if back != 3_000 {
		t.Fatalf("عاد إلى الخزينة %d والمتوقّع ٣٠٠٠", back)
	}
	f.ledgerClean(t)
}

// LD-07 — **الكرتان بالترشيح نفسِه · و«انسقط عنه قبل X مرّات».**
func TestLD_ListSummaryMatchesFilterAndWaivedBefore(t *testing.T) {
	f := newLDFixture(t)
	ctx := context.Background()
	f.dispute(t, 1_000)
	old := f.dispute(t, 700)
	if _, err := f.pool.Exec(ctx, `UPDATE disputes SET status = 'waived', settlement = 'waived' WHERE id = $1`, old); err != nil {
		t.Fatal(err)
	}
	var body struct {
		Data struct {
			OpenCount int   `json:"open_count"`
			Total     int64 `json:"total"`
			Disputes  []struct {
				PartyID      string `json:"party_id"`
				WaivedBefore int    `json:"waived_before"`
				OpenAmount   int64  `json:"open_amount"`
			} `json:"disputes"`
		} `json:"data"`
	}
	// **ترشيحُ «السائقون»** — العددُ والمبلغُ بالطرف نفسِه.
	var drivers struct {
		OpenCount, Total int64
	}
	if err := f.pool.QueryRow(ctx, `SELECT count(*), COALESCE(sum(amount - recovered), 0)
		FROM disputes WHERE status = 'open' AND party_role = 'driver'`).Scan(&drivers.OpenCount, &drivers.Total); err != nil {
		t.Fatal(err)
	}
	w := f.call(f.proposer, http.MethodGet, "/x?party=driver", nil, "", f.srv.handleListDisputes)
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || w.Code != http.StatusOK {
		t.Fatalf("القائمةُ ردّت %d: %s", w.Code, w.Body.String())
	}
	if int64(body.Data.OpenCount) != drivers.OpenCount || body.Data.Total != drivers.Total {
		t.Fatalf("**الكرتان مختلفا الترشيح**: عددٌ %d (والسائقون %d) · مبلغٌ %d (والسائقون %d)",
			body.Data.OpenCount, drivers.OpenCount, body.Data.Total, drivers.Total)
	}
	w = f.call(f.proposer, http.MethodGet, "/x?party=merchant&per_page=100", nil, "", f.srv.handleListDisputes)
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	found := false
	for _, d := range body.Data.Disputes {
		if d.PartyID == f.merchantID {
			found = true
			if d.WaivedBefore != 1 || d.OpenAmount != 1_000 {
				t.Fatalf("انسقط عنه قبل %d مرّة (المتوقّع ١) · المفتوح %d", d.WaivedBefore, d.OpenAmount)
			}
		}
	}
	if !found {
		t.Fatal("نزاعُ المتجر غائبٌ عن القائمة")
	}
}

// LD-08 — **خرج · رجع · الصافي · ولنا عند الناس — وأيّامُ دمشق.**
func TestLD_LossesNetAndDamascusDay(t *testing.T) {
	f := newLDFixture(t)
	ctx := context.Background()
	orderID := f.order(t, "merchant_closed")
	// **خسارةٌ في الواحدة فجراً بتوقيت دمشق** (٢٢:٠٠ غرينتش من اليوم السابق).
	if err := f.srv.orders.DebitTreasury(ctx, f.pool, 4_000, orderID, "تعويضُ فحص", f.proposer); err != nil {
		t.Fatalf("تعذّر القيد: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `
		UPDATE wallet_transactions SET created_at = '2031-03-09 22:00:00+00'
		WHERE ref = $1 AND kind = 'platform_expense'`, orderID); err != nil {
		t.Fatal(err)
	}
	var body struct {
		Data struct {
			Total     int64 `json:"total"`
			Recovered int64 `json:"recovered"`
			Net       int64 `json:"net"`
			Losses    []struct {
				OrderID *string `json:"order_id"`
			} `json:"losses"`
		} `json:"data"`
	}
	w := f.call(f.proposer, http.MethodGet, "/x?from=2031-03-10&to=2031-03-10", nil, "", f.srv.handlePlatformLosses)
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || w.Code != http.StatusOK {
		t.Fatalf("الخسائرُ ردّت %d: %s", w.Code, w.Body.String())
	}
	if body.Data.Total != 4_000 {
		t.Fatalf("**خسارةُ الواحدة فجراً بتوقيت دمشق حُسبت لغير يومها**: مجموعُ ١٠ آذار = %d", body.Data.Total)
	}
	if body.Data.Net != body.Data.Total-body.Data.Recovered {
		t.Fatalf("الصافي %d ≠ %d − %d", body.Data.Net, body.Data.Total, body.Data.Recovered)
	}
	if len(body.Data.Losses) != 1 || body.Data.Losses[0].OrderID == nil || *body.Data.Losses[0].OrderID != orderID {
		t.Fatalf("سطرُ الخسارة بلا رابطِ طلبه: %s", w.Body.String())
	}
}

// LD-09 — **سببُ نزاع المتجر من طلب التعويض الذي ذنبُه على المتجر.**
func TestLD_MerchantClaimReasonFromCompensationRequest(t *testing.T) {
	f := newLDFixture(t)
	ctx := context.Background()
	// **الطلبُ فشل أخيراً لأنّ الزبون غاب** — لكنّ التعويضَ كان عن إغلاق المتجر.
	orderID := f.order(t, "customer_absent")
	driver := testdb.NewUser(t, f.pool, "driver")
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO driver_compensation_requests (order_id, driver_id, fault, fail_reason, suggested_amount)
		VALUES ($1, $2, 'merchant', 'merchant_closed', 2500)`, orderID, driver); err != nil {
		t.Fatalf("تعذّر طلبُ التعويض: %v", err)
	}
	if err := f.srv.orders.OpenMerchantClaimTx(ctx, f.pool, orderID, 2_500); err != nil {
		t.Fatalf("تعذّرت المطالبة: %v", err)
	}
	var reason string
	if err := f.pool.QueryRow(ctx, `SELECT reason FROM disputes WHERE order_id = $1`, orderID).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	if reason != "merchant_closed" {
		t.Fatalf("**سببُ النزاع %q — والمتجرُ يُطالَب بذنبِ غيره**", reason)
	}
}
