package server

// ══════════════════════════════════════════════════════════════════════
// **من يدفع — قراراتُ المالك ٢٠٢٦-١٠-١٥**
// ══════════════════════════════════════════════════════════════════════
//
//	١ · «حظرُ الحساب» بذنب الزبون عند الباب يحظره فعلاً، بقدرة صفحة الحسابات
//	٢ · صافي صفحة الخسائر = سطورُ الخسارة في صفحة الأرباح − المستردّ
//	٣ · سطرُ محفظة الطرف في مطالبة النزاع لا يحمل الملاحظةَ الداخليّة

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// doorCall **بابُ الإنهاء بقدراتٍ صريحة** — والعُدّةُ العامّةُ لا تضع قدرات.
func (f *driverFixture) doorCall(orderID, actor string, caps []string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/admin/orders/"+orderID+"/door-resolution",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", orderID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, actor)
	ctx = context.WithValue(ctx, ctxRoles, []string{"ops"})
	ctx = context.WithValue(ctx, ctxCaps, caps)
	w := httptest.NewRecorder()
	f.srv.handleDoorResolution(w, req.WithContext(ctx))
	return w
}

// TestDoor_CustomerBlockBlocksAccount **«حظرُ الحساب» يحظر الحسابَ في معاملة الإنهاء** —
// بالحظر القائم نفسِه وتدقيقِه، **ولا يحظر من لا يملك قدرةَ صفحة الحسابات.**
func TestDoor_CustomerBlockBlocksAccount(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	ops := testdb.NewUser(t, f.pool, "operations")
	body := `{"action":"return_to_office","fault":"customer","reason":"customer_refused",` +
		`"note":"رفض الاستلام مرّتين","customer_action":"block"}`

	// **بلا القدرة يُردّ قبل أن يُنهى شيء.**
	order1 := f.problemOrderAt(t, "at_dropoff", f.drivers[0])
	w := f.doorCall(order1, ops, []string{string(authz.OrdersIntervene)}, body)
	if w.Code != http.StatusForbidden {
		t.Fatalf("حُظر الحسابُ بلا قدرة الحسابات: %d %s", w.Code, w.Body.String())
	}
	var st string
	if err := f.pool.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, order1).Scan(&st); err != nil {
		t.Fatal(err)
	}
	if st != "at_dropoff" {
		t.Fatalf("انتهى الطلبُ (%s) والطلبُ مردود", st)
	}

	w = f.doorCall(order1, ops, []string{string(authz.OrdersIntervene), string(authz.UsersStatusManage)}, body)
	if w.Code != http.StatusOK {
		t.Fatalf("الإنهاءُ ردّ %d: %s", w.Code, w.Body.String())
	}
	var customer, status, reason, action string
	if err := f.pool.QueryRow(ctx, `
		SELECT u.id::text, u.status, COALESCE(u.status_reason, ''), COALESCE(o.customer_action, '')
		  FROM orders o JOIN users u ON u.id = o.customer_id WHERE o.id = $1`, order1).
		Scan(&customer, &status, &reason, &action); err != nil {
		t.Fatal(err)
	}
	if status != "blocked" || action != "block" || reason == "" {
		t.Fatalf("الحساب (%s · سبب %q) والطلب (%s) — **والمكتبُ اختار حظرَ الحساب**", status, reason, action)
	}
	var audited int
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_log
		 WHERE action = 'admin.user_update' AND entity = 'user' AND entity_id = $1
		   AND details->>'source' = 'door_resolution' AND details->>'order_id' = $2`,
		customer, order1).Scan(&audited); err != nil {
		t.Fatal(err)
	}
	if audited != 1 {
		t.Fatalf("أثرُ الحظر في التدقيق %d — والمتوقّع سطرٌ واحد", audited)
	}
}

// TestLosses_NetEqualsProfitLossLines **صافي صفحة الخسائر = سطورُ الخسارة في الأرباح −
// المستردّ** — بطلبٍ فاشلٍ وتعويضٍ ونقصِ صندوقٍ وتسويةٍ يدويّةٍ إيداعاً وخصماً ودينٍ مشطوب.
func TestLosses_NetEqualsProfitLossLines(t *testing.T) {
	f := newProfitsFx(t, "فحص الخسائر الواحدة")
	ctx := context.Background()
	const dayS = "2001-08-23"
	day := damascusAt(t, dayS+" 12:00")
	boss := testdb.NewUser(t, f.srv.pg, "admin")

	failed := f.order(t, "failed", day)
	f.post(t, f.treasury, -3000, "platform_profit", failed, day) // خسارةُ طلبٍ فاشل
	f.post(t, f.treasury, -400, "platform_expense", failed, day) // تعويضُ سائق

	// **نقصُ الصندوق** — إغلاقٌ ناقصٌ مئة، ونقصٌ موافَقٌ عليه بقيده.
	var closeID, shortID string
	if err := f.srv.pg.QueryRow(ctx, `
		INSERT INTO office_cash_closes (day, opening, expected, counted, difference, amount,
		                                status, proposed_by)
		VALUES ($1::date, 0, 1000, 900, -100, 100, 'approved', $2) RETURNING id::text`,
		dayS, boss).Scan(&closeID); err != nil {
		t.Fatalf("الإغلاق: %v", err)
	}
	if err := f.srv.pg.QueryRow(ctx, `
		INSERT INTO office_cash_shortfalls (close_id, amount, status, proposed_by, eligible_at)
		VALUES ($1, 100, 'approved', $2, now()) RETURNING id::text`, closeID, boss).Scan(&shortID); err != nil {
		t.Fatalf("النقص: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.srv.pg.Exec(c, `DELETE FROM office_cash_shortfalls WHERE id = $1`, shortID)
		_, _ = f.srv.pg.Exec(c, `DELETE FROM office_cash_closes WHERE id = $1`, closeID)
	})
	f.post(t, f.treasury, -100, "platform_expense", shortID, day)

	credit := f.wr(t, "adjustment", false, 5000) // إيداعٌ يدويٌّ في محفظة ⇒ خسارة
	f.post(t, f.treasury, -5000, "platform_expense", credit, day)
	debit := f.wr(t, "adjustment", true, 700) // خصمٌ يدويٌّ من محفظة ⇒ مستردّ
	f.post(t, f.treasury, 700, "platform_profit", debit, day)

	// **دينٌ مشطوبٌ في اليوم نفسِه** — بلا قيدٍ في الدفتر.
	rep := testdb.NewUser(t, f.srv.pg, "sales")
	var obl string
	if err := f.srv.pg.QueryRow(ctx, `
		INSERT INTO financial_obligations (party_kind, party_id, amount, cause)
		VALUES ('rep', $1, 1200, 'legacy_opening') RETURNING id::text`, rep).Scan(&obl); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.srv.pg.Exec(context.Background(), `DELETE FROM financial_obligations WHERE id = $1`, obl)
	})
	if _, err := f.srv.pg.Exec(ctx, `
		INSERT INTO obligation_requests (obligation_id, kind, amount, note, status, proposed_by,
		                                 decided_by, decided_at)
		VALUES ($1, 'write_off', 1200, 'لن يُسدَّد', 'approved', $2, $2, $3)`, obl, boss, day); err != nil {
		t.Fatal(err)
	}

	p := readPlatformProfits(t, f.srv, dayS)
	want := map[string]int64{
		"lost_failed": -3000, "compensations": -400, "shortfalls": -100,
		"adjustment_credits": -5000, "recovered": 700,
	}
	for k, v := range want {
		if got := p.line(k); got != v {
			t.Errorf("سطرُ الأرباح %s = %d والمتوقّع %d", k, got, v)
		}
	}
	if !p.CheckOK || p.LinesSum != p.Net {
		t.Fatalf("مجموعُ البنود %d ≠ الصافي %d", p.LinesSum, p.Net)
	}
	var profitLoss int64
	for _, k := range []string{"lost_failed", "lost_cancelled", "lost_refunded", "compensations",
		"shortfalls", "adjustment_credits"} {
		profitLoss -= p.line(k)
	}

	w := callAs(f.srv.handlePlatformLosses, "GET", "/x?from="+dayS+"&to="+dayS, "", "", financeCaps)
	var l struct {
		Total     int64 `json:"total"`
		Recovered int64 `json:"recovered"`
		Net       int64 `json:"net"`
		Losses    []struct {
			Kind   string `json:"kind"`
			Amount int64  `json:"amount"`
		} `json:"losses"`
	}
	if err := json.Unmarshal(dataOf(t, w), &l); err != nil {
		t.Fatal(err)
	}
	if l.Total != profitLoss || l.Recovered != p.line("recovered") ||
		l.Net != profitLoss-p.line("recovered") {
		t.Fatalf("**صفحتان برقمين**: الخسائر (مجموع %d · مستردّ %d · صافٍ %d) والأرباح (خسائر %d · مستردّ %d)",
			l.Total, l.Recovered, l.Net, profitLoss, p.line("recovered"))
	}
	// **والمشطوبُ لا يُعدّ ثانيةً** — خسارتُه قُيّدت يومَ نشأ دينُه.
	if l.Net != 7800 {
		t.Fatalf("الصافي %d والمتوقّع 7800 (3000+400+100+5000−700)", l.Net)
	}
	if p.line("written_off") != 0 {
		t.Fatalf("المشطوبُ سطرٌ في الصافي (%d) — يُعدّ مرّتين", p.line("written_off"))
	}
	kinds := map[string]int64{}
	for _, x := range l.Losses {
		kinds[x.Kind] += x.Amount
	}
	for k, v := range map[string]int64{"lost_failed": 3000, "compensation": 400, "shortfall": 100,
		"adjustment_credit": 5000} {
		if kinds[k] != v {
			t.Errorf("صفُّ %s في الجدول = %d والمتوقّع %d", k, kinds[k], v)
		}
	}
}

// TestDispute_ChargeLineHidesInternalNote **سطرُ محفظة الطرف لا يحمل كلمةَ المكتب.**
func TestDispute_ChargeLineHidesInternalNote(t *testing.T) {
	f := newLDFixture(t)
	f.fund(t, f.owner, 3_000)
	id := f.dispute(t, 1_000)
	res := f.propose(t, id, "charge") // ملاحظةُ المقترِح: «سمعنا المتجر»
	w := f.call(f.approver, http.MethodPost, "/x", map[string]string{"id": res}, `{}`,
		f.srv.handleDecideDisputeResolution(true))
	if w.Code != http.StatusOK {
		t.Fatalf("الموافقةُ ردّت %d: %s", w.Code, w.Body.String())
	}
	var note string
	if err := f.pool.QueryRow(context.Background(), `
		SELECT note FROM wallet_transactions
		 WHERE user_id = $1 AND ref = $2 AND kind = 'adjustment'`, f.owner, id).Scan(&note); err != nil {
		t.Fatalf("سطرُ المحفظة: %v", err)
	}
	if strings.Contains(note, "سمعنا") || note != "مطالبة من المنصة" {
		t.Fatalf("سطرُ محفظة المتجر %q — **والملاحظةُ الداخليّةُ لا تصل الطرف**", note)
	}
}
