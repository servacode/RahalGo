package server

// **صفحةُ «التعويضات» الواحدة** (قرارُ المالك ٢٠٢٦-١٠-٠٤) — اختباراتُ القرارات
// الستّة وعيوبِ الفحص: الزرُّ بمعرّف طلب التعويض، والمقترحُ غيرُ الموافق،
// والسقف، والنسبةُ صفر، وتعويضُ الشكوى من الخزينة لصاحبها، وتعويضُ البضاعة
// عبر الطابور، والسببُ بالعربيّة، والسجلّ، والتأخّر.

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"testing"
	"unicode"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// compTreasury **خزينةٌ للاختبار** — القائمةُ إن وُجدت، وإلّا تُنشأ.
func compTreasury(t *testing.T, f *driverFixture) string {
	t.Helper()
	ctx := context.Background()
	var id string
	if err := f.pool.QueryRow(ctx,
		`SELECT user_id::text FROM wallets WHERE is_treasury LIMIT 1`).Scan(&id); err == nil {
		return id
	}
	id = testdb.NewUser(t, f.pool, "admin")
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO wallets (user_id, balance, is_treasury) VALUES ($1, 0, true)
		ON CONFLICT (user_id) DO UPDATE SET is_treasury = true`, id); err != nil {
		t.Fatalf("الخزينة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`UPDATE wallets SET is_treasury = false WHERE user_id = $1`, id)
	})
	return id
}

func (f *driverFixture) bal(t *testing.T, userID string) int64 {
	t.Helper()
	var b int64
	_ = f.pool.QueryRow(context.Background(),
		`SELECT COALESCE((SELECT balance FROM wallets WHERE user_id = $1), 0)`, userID).Scan(&b)
	return b
}

// pendingRow **صفٌّ معلَّقٌ مكتوبٌ باليد** — كما يكتبه المحرّكُ أو غرفةُ الطوارئ.
func (f *driverFixture) pendingRow(t *testing.T, orderID, driverID, fault, reason string,
	suggested int64, age string) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO driver_compensation_requests
		    (order_id, driver_id, fault, fail_reason, suggested_amount, created_at)
		VALUES ($1, $2, $3, $4, $5, now() - $6::interval) RETURNING id::text`,
		orderID, driverID, fault, reason, suggested, age).Scan(&id); err != nil {
		t.Fatalf("صفّ التعويض: %v", err)
	}
	return id
}

func (f *driverFixture) approveComp(id, actor string, roles []string, amount int64) int {
	w := f.call(f.srv.handleApproveCompensation, http.MethodPost,
		"/admin/compensations/"+id+"/approve", id, actor, roles,
		`{"amount":`+strconv.FormatInt(amount, 10)+`,"note":"موافقة الاختبار"}`)
	return w.Code
}

func (f *driverFixture) approveCompW(id, actor string, amount int64) (int, string) {
	w := f.call(f.srv.handleApproveCompensation, http.MethodPost,
		"/admin/compensations/"+id+"/approve", id, actor, []string{"finance"},
		`{"amount":`+strconv.FormatInt(amount, 10)+`,"note":"موافقة الاختبار"}`)
	return w.Code, w.Body.String()
}

// TestCOMP_ApproveByRequestID_PaysTheShownDriver **الزرُّ بمعرّف طلب التعويض لا برقم الطلب.**
//
// طلبٌ تعذّر عند المتجر مع سائقَين متتاليَين ⇒ طلبان معلَّقان. **كان «موافقة» على صفّ
// الثاني يدفع للأوّل** (الأقدم) والنافذةُ تعرض اسمَ الثاني.
func TestCOMP_ApproveByRequestID_PaysTheShownDriver(t *testing.T) {
	f := newDriverFixture(t, 2)
	compTreasury(t, f)
	a, b := f.drivers[0], f.drivers[1]
	orderID := f.problemOrderAt(t, "at_pickup", b)
	f.pendingRow(t, orderID, a, "merchant", "merchant_closed", 2500, "1 hour")
	second := f.pendingRow(t, orderID, b, "merchant", "merchant_closed", 2500, "0")
	fin := testdb.NewUser(t, f.pool, "finance")

	// **البابُ القديمُ بلا معرّفٍ لا يختار الأقدمَ بعد اليوم.**
	w := f.call(f.srv.handleCompensateDriver, http.MethodPost,
		"/admin/orders/"+orderID+"/compensate-driver", orderID, fin, []string{"finance"},
		`{"amount":2500,"note":"موافقة"}`)
	if w.Code != http.StatusConflict || errCode(t, w) != "compensation_ambiguous" {
		t.Fatalf("**البابُ القديمُ دفع لأحد السائقَين بلا تسمية** — %d: %s (رصيدُ الأوّل %d)",
			w.Code, w.Body.String(), f.bal(t, a))
	}
	if code, body := f.approveCompW(second, fin, 2500); code != http.StatusOK {
		t.Fatalf("الموافقةُ بالمعرّف ردّت %d: %s", code, body)
	}
	if f.bal(t, b) != 2500 || f.bal(t, a) != 0 {
		t.Fatalf("دُفع للسائق الخطأ: الأوّل %d · الثاني %d", f.bal(t, a), f.bal(t, b))
	}
	// **والسائقُ لا يرى ملاحظةَ المكتب** — سطرُ المحفظة بلا ملاحظة.
	var note string
	_ = f.pool.QueryRow(context.Background(), `SELECT note FROM wallet_transactions
		WHERE user_id = $1 AND kind = 'compensation'`, b).Scan(&note)
	if note != "" {
		t.Fatalf("ملاحظةُ الموظّف في كشف السائق: %q", note)
	}
}

// TestCOMP_ProposerCannotApproveOwn **لا يوافق أحدٌ على ما اقترحه.**
func TestCOMP_ProposerCannotApproveOwn(t *testing.T) {
	f := newDriverFixture(t, 1)
	treasury := compTreasury(t, f)
	d := f.drivers[0]
	orderID := f.problemOrderAt(t, "at_dropoff", d)
	proposer := testdb.NewUser(t, f.pool, "finance")
	other := testdb.NewUser(t, f.pool, "finance")

	w := f.call(f.srv.handleProposeCompensation, http.MethodPost, "/admin/compensations", "",
		proposer, []string{"finance"},
		`{"order_id":"`+orderID+`","user_id":"`+d+`","amount":3000,"fault":"platform","note":"خطأ المنصة"}`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("الاقتراحُ ردّ %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Proposed string `json:"proposed"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp.Data.Proposed
	if f.bal(t, d) != 0 {
		t.Fatal("**الاقتراحُ دفع** — والمالُ عند الموافقة وحدَها")
	}
	if code := f.approveComp(id, proposer, []string{"finance"}, 3000); code != http.StatusForbidden {
		t.Fatalf("**المقترحُ وافق على نفسه** — %d", code)
	}
	before := f.bal(t, treasury)
	if code, body := f.approveCompW(id, other, 3000); code != http.StatusOK {
		t.Fatalf("موافقةُ غيره ردّت %d: %s", code, body)
	}
	if f.bal(t, d) != 3000 || f.bal(t, treasury) != before-3000 {
		t.Fatalf("السائق %d · الخزينة %d ← %d — والمتوقّع خصمُ ٣٠٠٠ منها", f.bal(t, d), before, f.bal(t, treasury))
	}
}

// TestCOMP_AboveCapOnlyOwner **فوق سقف النوع مديرُ المنصّة وحدَه يوافق.**
func TestCOMP_AboveCapOnlyOwner(t *testing.T) {
	f := newDriverFixture(t, 1)
	compTreasury(t, f)
	f.srv.orders.SetSettings(settings.NewStore(f.pool))
	f.setSetting(t, "compensations.cap_driver", 4000)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM settings WHERE key = 'compensations.cap_driver'`)
	})
	d := f.drivers[0]
	orderID := f.problemOrderAt(t, "at_dropoff", d)
	id := f.pendingRow(t, orderID, d, "customer", "customer_refused", 2500, "0")
	fin := testdb.NewUser(t, f.pool, "finance")
	w := f.call(f.srv.handleApproveCompensation, http.MethodPost,
		"/admin/compensations/"+id+"/approve", id, fin, []string{"finance"},
		`{"amount":5000,"note":"فوق السقف"}`)
	if w.Code != http.StatusForbidden || errCode(t, w) != "compensation_above_cap" {
		t.Fatalf("**الماليّةُ وافقت فوق السقف** — %d: %s", w.Code, w.Body.String())
	}
	owner := testdb.NewUser(t, f.pool, "owner_super_admin")
	if code, body := f.approveCompW(id, owner, 5000); code != http.StatusOK {
		t.Fatalf("المالكُ فوق السقف ردّ %d: %s", code, body)
	}
}

// TestCOMP_ZeroPercentStillCreatesRequest **النسبةُ صفرٌ تكتب طلباً بمقترَحٍ صفر.**
func TestCOMP_ZeroPercentStillCreatesRequest(t *testing.T) {
	f := newDriverFixture(t, 1)
	f.armCompensation(t)
	f.setSetting(t, "drivers.failed_compensation_percent", 0)
	t.Cleanup(func() { f.setSetting(t, "drivers.failed_compensation_percent", 50) })
	d := f.drivers[0]
	orderID := f.problemOrderAt(t, "at_dropoff", d)
	if w := f.endAtDoor(t, orderID, "customer", "customer_refused"); w.Code != http.StatusOK {
		t.Fatalf("الإنهاءُ ردّ %d: %s", w.Code, w.Body.String())
	}
	var suggested int64 = -1
	_ = f.pool.QueryRow(context.Background(), `SELECT suggested_amount FROM driver_compensation_requests
		WHERE order_id = $1 AND driver_id = $2`, orderID, d).Scan(&suggested)
	if suggested != 0 {
		t.Fatalf("**النسبةُ صفرٌ أسكتت الطلب** — المقترَح %d (‎-1 = لا طلب)", suggested)
	}
	if def, _ := settings.Lookup("drivers.failed_compensation_percent"); def.Default != 50 {
		t.Fatalf("الافتراضيُّ في الفهرس %v — والقاعدةُ ٥٠", def.Default)
	}
}

// TestCOMP_ComplaintPaysOwnerFromTreasury **تعويضُ الشكوى لصاحبها ومن الخزينة.**
func TestCOMP_ComplaintPaysOwnerFromTreasury(t *testing.T) {
	f := newDriverFixture(t, 1)
	treasury := compTreasury(t, f)
	d := f.drivers[0] // **صاحبُ الشكوى سائقٌ لا زبون.**
	ctx := context.Background()
	var ticketID string
	if err := f.pool.QueryRow(ctx, `INSERT INTO tickets (customer_id, subject, status)
		VALUES ($1, 'شكوى سائق', 'resolved') RETURNING id::text`, d).Scan(&ticketID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM tickets WHERE id = $1`, ticketID) })
	support := testdb.NewUser(t, f.pool, "customer_support")
	id, err := orders.ProposeCompensationTx(ctx, f.pool, orders.CompensationProposal{
		Kind: orders.CompKindComplaint, TicketID: ticketID, BeneficiaryID: d,
		Amount: 4000, Note: "تأخير متكرر", ProposedBy: support,
	})
	if err != nil {
		t.Fatalf("الاقتراح: %v", err)
	}
	if _, err := orders.ProposeCompensationTx(ctx, f.pool, orders.CompensationProposal{
		Kind: orders.CompKindComplaint, TicketID: ticketID, BeneficiaryID: d, Amount: 1, ProposedBy: support,
	}); err != orders.ErrCompensationDuplicate {
		t.Fatalf("اقتراحٌ ثانٍ للشكوى نفسِها: %v", err)
	}
	fin := testdb.NewUser(t, f.pool, "finance")
	before := f.bal(t, treasury)
	if code, body := f.approveCompW(id, fin, 4000); code != http.StatusOK {
		t.Fatalf("الموافقةُ ردّت %d: %s", code, body)
	}
	if f.bal(t, d) != 4000 || f.bal(t, treasury) != before-4000 {
		t.Fatalf("صاحبُ الشكوى %d · الخزينة %d ← %d — **مالٌ خُلق أو ذهب لغير صاحبه**",
			f.bal(t, d), before, f.bal(t, treasury))
	}
	var comp int64
	_ = f.pool.QueryRow(ctx, `SELECT compensation FROM tickets WHERE id = $1`, ticketID).Scan(&comp)
	if comp != 4000 {
		t.Fatalf("تعويضُ التذكرة %d", comp)
	}
}

// TestCOMP_GoodsCompensationGoesThroughQueue **تعويضُ البضاعة اقتراحٌ يمرّ بالطابور.**
func TestCOMP_GoodsCompensationGoesThroughQueue(t *testing.T) {
	f := newDriverFixture(t, 1)
	compTreasury(t, f)
	ctx := context.Background()
	owner := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET owner_user_id = $2 WHERE id = $1`, f.merchantID, owner); err != nil {
		t.Fatal(err)
	}
	orderID := f.problemOrderAt(t, "at_dropoff", f.drivers[0])
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'failed', closed_at = now(),
		goods_settled_to = 'merchant' WHERE id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO order_items (order_id, name, unit_price, qty, merchant_id, merchant_price)
		VALUES ($1, 'صنف', 12000, 1, $2, 10000)`, orderID, f.merchantID); err != nil {
		t.Fatal(err)
	}
	fin := testdb.NewUser(t, f.pool, "finance")
	w := f.call(f.srv.handleGoodsCompensation, http.MethodPost,
		"/admin/orders/"+orderID+"/goods/compensation", orderID, fin, []string{"finance"}, `{"amount":20000}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("فوق سعر الشراء ردّ %d: %s", w.Code, w.Body.String())
	}
	w = f.call(f.srv.handleGoodsCompensation, http.MethodPost,
		"/admin/orders/"+orderID+"/goods/compensation", orderID, fin, []string{"finance"}, `{"amount":8000}`)
	if w.Code != http.StatusAccepted || f.bal(t, owner) != 0 {
		t.Fatalf("**تعويضُ البضاعة دُفع بضغطة كاتبه** — %d · رصيدُ المتجر %d: %s", w.Code, f.bal(t, owner), w.Body.String())
	}
	var id string
	_ = f.pool.QueryRow(ctx, `SELECT id::text FROM driver_compensation_requests
		WHERE order_id = $1 AND kind = 'merchant_goods'`, orderID).Scan(&id)
	if code := f.approveComp(id, fin, []string{"finance"}, 8000); code != http.StatusForbidden {
		t.Fatalf("كاتبُ المبلغ وافق على نفسه — %d", code)
	}
	other := testdb.NewUser(t, f.pool, "finance")
	if code, body := f.approveCompW(id, other, 8000); code != http.StatusOK || f.bal(t, owner) != 8000 {
		t.Fatalf("الموافقة %d · رصيدُ المتجر %d: %s", code, f.bal(t, owner), body)
	}
}

// TestCOMP_ReasonsArabicEverywhere **لا رمزَ آلةٍ في الصفحة ولا في الإشعار.**
func TestCOMP_ReasonsArabicEverywhere(t *testing.T) {
	codes := []string{"", "unknown_code", "emergency_accident", orders.ReasonComplaint, orders.ReasonGoodsReturned}
	for _, l := range [][]orders.FailReason{orders.StageReports, orders.ReleaseReasons} {
		for _, r := range l {
			codes = append(codes, r.Code)
		}
	}
	for _, c := range codes {
		body := orders.CompensationAlertBody(1349, c, 5000)
		for _, ch := range body {
			if ch < unicode.MaxASCII && unicode.IsLetter(ch) {
				t.Fatalf("الإشعارُ فيه رمزٌ إنكليزيّ للسبب %q: %s", c, body)
			}
		}
	}
}

// TestCOMP_HistoryTabsSummaryMismatch **السجلُّ بتبويباته وبطاقاته وتحذيرُ التطابق.**
func TestCOMP_HistoryTabsSummaryMismatch(t *testing.T) {
	f := newDriverFixture(t, 2)
	compTreasury(t, f)
	f.srv.orders.SetSettings(settings.NewStore(f.pool))
	o1 := f.problemOrderAt(t, "at_pickup", f.drivers[0])
	o2 := f.problemOrderAt(t, "at_pickup", f.drivers[1])
	// **«المتجرُ مغلق» وذنبُه «الزبون»** — كما على التجهيز (#1344).
	mis := f.pendingRow(t, o1, f.drivers[0], "customer", "merchant_closed", 2500, "30 hours")
	rej := f.pendingRow(t, o2, f.drivers[1], "merchant", "merchant_closed", 2500, "0")
	fin := testdb.NewUser(t, f.pool, "finance")
	w := f.call(f.srv.handleRejectCompensationByID, http.MethodPost, "/admin/compensations/"+rej+"/reject",
		rej, fin, []string{"finance"}, `{"note":"بيانات تجربة"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("الرفضُ ردّ %d: %s", w.Code, w.Body.String())
	}
	list := func(q string) (rows []orders.CompensationRequest, sum orders.CompensationSummary) {
		w := f.call(f.srv.handleListCompensations, http.MethodGet, "/admin/compensations?"+q, "",
			fin, []string{"finance"}, "")
		var out struct {
			Data struct {
				Compensations []orders.CompensationRequest `json:"compensations"`
				Summary       orders.CompensationSummary   `json:"summary"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || w.Code != http.StatusOK {
			t.Fatalf("القائمة %d: %s", w.Code, w.Body.String())
		}
		return out.Data.Compensations, out.Data.Summary
	}
	find := func(rows []orders.CompensationRequest, id string) *orders.CompensationRequest {
		for i := range rows {
			if rows[i].ID == id {
				return &rows[i]
			}
		}
		return nil
	}
	rows, sum := list("status=rejected&person=" + f.drivers[1])
	if r := find(rows, rej); r == nil || r.DecisionNote != "بيانات تجربة" {
		t.Fatalf("**المرفوضُ لا يظهر في سجلّه**: %+v", rows)
	}
	if sum.RejectedMonthCount < 1 || sum.PendingCount < 1 || sum.PendingSum < 2500 {
		t.Fatalf("البطاقات: %+v", sum)
	}
	rows, _ = list("status=pending&fault=customer&kind=driver")
	r := find(rows, mis)
	if r == nil || !r.FaultMismatch || r.ExpectedFault != "merchant" || r.ReasonLabel != "المتجر مغلق" || !r.Overdue {
		t.Fatalf("**الذنبُ المخالفُ للسبب لا يُحذَّر منه أو التأخّرُ لا يحمرّ**: %+v", r)
	}
	if w := f.call(f.srv.handleListCompensations, http.MethodGet, "/admin/compensations?status=x", "",
		fin, []string{"finance"}, ""); w.Code != http.StatusBadRequest {
		t.Fatalf("حالٌ مجهولةٌ قُبلت: %d", w.Code)
	}
}

// TestCOMP_OverdueAlertsOwnerOnce **المعلَّقُ فوق المهلة يُنبَّه عنه المالكُ مرّةً واحدة.**
func TestCOMP_OverdueAlertsOwnerOnce(t *testing.T) {
	f := newDriverFixture(t, 1)
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	n := notifications.New(f.pool, f.srv.hub, quiet)
	f.srv.notify = n
	f.srv.orders.SetNotifier(n)
	f.srv.orders.SetSettings(settings.NewStore(f.pool))
	owner := testdb.NewUser(t, f.pool, "owner_super_admin")
	o := f.problemOrderAt(t, "at_pickup", f.drivers[0])
	id := f.pendingRow(t, o, f.drivers[0], "merchant", "merchant_closed", 2500, "25 hours")
	ctx := context.Background()
	if _, err := f.srv.orders.SweepOverdueCompensations(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := f.srv.orders.SweepOverdueCompensations(ctx); err != nil {
		t.Fatal(err)
	}
	var alerted bool
	_ = f.pool.QueryRow(ctx, `SELECT overdue_alerted_at IS NOT NULL FROM driver_compensation_requests WHERE id = $1`, id).Scan(&alerted)
	var got int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1
		AND href = '/dashboard/compensations'`, owner).Scan(&got)
	if !alerted || got < 1 {
		t.Fatalf("**تأخّر ٢٥ ساعةً ولم يُنبَّه المالك** — معلَّم %v · إشعارات %d", alerted, got)
	}
	if got != 1 {
		t.Fatalf("نُبّه مرّتين")
	}
}
