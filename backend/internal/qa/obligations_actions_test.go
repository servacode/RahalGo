package qa

import (
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/obligations"
)

// ══════════════════════════════════════════════════════════════════════
// **قسمُ الديون — قراراتُ المالك ٢٠٢٦-١٠-٠٤**
// ══════════════════════════════════════════════════════════════════════
//
// دفعٌ نقديٌّ بالمكتب · شطبٌ بموافقة مدير المنصّة · الشحنُ يسدّ الدينَ أوّلاً ·
// تنبيهٌ بعد مهلة · الدينُ في تطبيق صاحبه · والملغى لا يُعرض مسدَّداً.

// legacyDebt **دينٌ قديمٌ على طرف** — بلا طلب، كما تنقله الهجرة `0130`.
func legacyDebt(t *testing.T, h *Harness, kind, id string, amount int64) string {
	t.Helper()
	if _, err := obligations.Create(ctxBG(), h.Pool, kind, id, amount,
		obligations.CauseLegacy, "", nil); err != nil {
		t.Fatalf("إنشاءُ الدين: %v", err)
	}
	var oid string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT id::text FROM financial_obligations
		 WHERE party_kind = $1 AND party_id = $2::uuid ORDER BY created_at DESC LIMIT 1`,
		kind, id).Scan(&oid); err != nil {
		t.Fatal(err)
	}
	return oid
}

func obligationState(t *testing.T, h *Harness, id string) (settled int64, closed bool, methods []string) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT settled, closed_at IS NOT NULL FROM financial_obligations WHERE id = $1::uuid`, id).
		Scan(&settled, &closed); err != nil {
		t.Fatal(err)
	}
	rows, err := h.Pool.Query(ctxBG(),
		`SELECT method FROM obligation_settlements WHERE obligation_id = $1::uuid ORDER BY id`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var m string
		_ = rows.Scan(&m)
		methods = append(methods, m)
	}
	return
}

func merchantDebtCol(t *testing.T, h *Harness, mID string) int64 {
	t.Helper()
	var v int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT debt FROM merchants WHERE id = $1::uuid`, mID).Scan(&v)
	return v
}

// ── ١ · دفعٌ نقديٌّ بالمكتب: اقتراحٌ ثمّ موافقةُ غيرِ المقترِح ──────────────

func TestOBLX_OfficeCashPaymentEntersCashboxAndTreasury(t *testing.T) {
	h := New(t)
	f := h.Factory()
	tid := treasury(t, h)
	m := f.Merchant()
	ob := legacyDebt(t, h, "merchant", m.ID, 9000)
	fin := h.NewUser("finance")
	admin := h.NewUser("admin")
	before := treasuryBalance(t, h, tid)

	// **ولا يتجاوز الباقي.**
	if r := h.POST("/api/v1/admin/obligations/"+ob+"/office-cash", fin.Token,
		map[string]any{"amount": 9001, "note": "دفع"}); r.Err() != "obligation_over_amount" {
		t.Errorf("فوق الباقي: %s", r)
	}
	r := h.POST("/api/v1/admin/obligations/"+ob+"/office-cash", fin.Token,
		map[string]any{"amount": 4000, "note": "دفع صاحب المتجر بالمكتب"})
	if r.Code != 201 {
		t.Fatalf("الاقتراح: %s", r)
	}
	reqID, _ := r.JSON()["request_id"].(string)
	// **واقتراحٌ ثانٍ على الدين نفسِه يُرفض ما دام الأوّلُ معلَّقاً.**
	if r2 := h.POST("/api/v1/admin/obligations/"+ob+"/office-cash", fin.Token,
		map[string]any{"amount": 1000, "note": "x"}); r2.Err() != "obligation_request_pending" {
		t.Errorf("اقتراحٌ مكرَّر: %s", r2)
	}
	// **ولا قيدَ قبل الموافقة.**
	if s, _, _ := obligationState(t, h, ob); s != 0 {
		t.Fatalf("**سُدّ الدينُ قبل الموافقة** (%d)", s)
	}
	// **وصاحبُ الاقتراح لا يوافق على نفسه.**
	if self := h.POST("/api/v1/admin/obligation-requests/"+reqID+"/approve", fin.Token,
		map[string]any{}); self.Err() != "self_approve" {
		t.Errorf("موافقةُ المقترِح: %s", self)
	}
	if ok := h.POST("/api/v1/admin/obligation-requests/"+reqID+"/approve", admin.Token,
		map[string]any{}); ok.Code != 200 {
		t.Fatalf("الموافقة: %s", ok)
	}
	settled, closed, methods := obligationState(t, h, ob)
	if settled != 4000 || closed || len(methods) != 1 || methods[0] != "office_cash" {
		t.Errorf("الدين بعد الدفع: مسدَّد %d مغلق %v طرق %v", settled, closed, methods)
	}
	var cash int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0) FROM office_cash_entries
		WHERE source = 'obligation_cash' AND ref = $1 AND direction = 'in'`, reqID).Scan(&cash)
	if cash != 4000 {
		t.Errorf("**النقدُ لم يدخل صندوقَ المكتب** (%d)", cash)
	}
	if after := treasuryBalance(t, h, tid); after-before != 4000 {
		t.Errorf("الخزينة: قبل %d بعد %d — والمتوقَّع +4000", before, after)
	}
	if d := merchantDebtCol(t, h, m.ID); d != 5000 {
		t.Errorf("صورةُ الدين %d والمتوقَّع 5000", d)
	}
}

// ── ٢ · الشطب: الماليّةُ تقترح، ومديرُ المنصّة وحدَه يوافق ─────────────────

func TestOBLX_WriteoffNeedsPlatformOwner(t *testing.T) {
	h := New(t)
	tid := treasury(t, h)
	rep := h.NewUser("sales")
	ob := legacyDebt(t, h, "rep", rep.ID, 6000)
	fin := h.NewUser("finance")
	fin2 := h.NewUser("finance")
	admin := h.NewUser("admin")
	before := treasuryBalance(t, h, tid)

	if r := h.POST("/api/v1/admin/obligations/"+ob+"/write-off", fin.Token,
		map[string]any{"note": "  "}); r.Err() != "obligation_note_required" {
		t.Errorf("بلا سبب: %s", r)
	}
	r := h.POST("/api/v1/admin/obligations/"+ob+"/write-off", fin.Token,
		map[string]any{"note": "المندوب ترك الشغل وما في طريقة نحصّل"})
	if r.Code != 201 {
		t.Fatalf("الاقتراح: %s", r)
	}
	reqID, _ := r.JSON()["request_id"].(string)
	if no := h.POST("/api/v1/admin/obligation-requests/"+reqID+"/approve", fin2.Token,
		map[string]any{}); no.Err() != "writeoff_owner_only" {
		t.Errorf("**ماليٌّ آخرُ وافق على الشطب**: %s", no)
	}
	if ok := h.POST("/api/v1/admin/obligation-requests/"+reqID+"/approve", admin.Token,
		map[string]any{}); ok.Code != 200 {
		t.Fatalf("موافقةُ المدير: %s", ok)
	}
	settled, closed, methods := obligationState(t, h, ob)
	if settled != 6000 || !closed || len(methods) != 1 || methods[0] != "written_off" {
		t.Errorf("بعد الشطب: %d %v %v", settled, closed, methods)
	}
	// **والخزينةُ تحمّلت المالَ يومَ نشأ الدين** — فلا قيدَ ثانٍ يعدّ الخسارةَ مرّتين.
	if after := treasuryBalance(t, h, tid); after != before {
		t.Errorf("الشطبُ حرّك الخزينة: %d ⟵ %d", before, after)
	}
	list := h.GET("/api/v1/admin/obligations?party_id="+rep.ID, admin.Token).JSON()
	rows, _ := list["obligations"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["state"] != "written_off" {
		t.Errorf("الحالُ في القائمة: %v", rows)
	}
}

// ── ٣ · الشحنُ يسدّ الدينَ أوّلاً ويُبلَّغ صاحبُه ──────────────────────────

func TestOBLX_TopupPaysDebtFirst(t *testing.T) {
	h := New(t)
	f := h.Factory()
	tid := treasury(t, h)
	m := f.Merchant()
	ob := legacyDebt(t, h, "merchant", m.ID, 3000)
	fin := h.NewUser("finance")
	admin := h.NewUser("admin")
	before := treasuryBalance(t, h, tid)

	r := walletPropose(h, fin, m.Owner.ID, map[string]any{"amount": 5000, "kind": "topup", "note": "نقد"})
	reqID, _ := r.JSON()["request_id"].(string)
	if ok := h.POST("/api/v1/admin/wallet-requests/"+reqID+"/approve", admin.Token, map[string]any{}); ok.Code != 200 {
		t.Fatalf("الموافقة: %s", ok)
	}
	if b := walletBalance(t, h, m.Owner.ID); b != 2000 {
		t.Errorf("**الشحنُ لم يسدّ الدين أوّلاً**: الرصيد %d والمتوقَّع 2000", b)
	}
	settled, closed, methods := obligationState(t, h, ob)
	if settled != 3000 || !closed || len(methods) != 1 || methods[0] != "wallet_topup" {
		t.Errorf("الدين: %d %v %v", settled, closed, methods)
	}
	if d := merchantDebtCol(t, h, m.ID); d != 0 {
		t.Errorf("صورةُ الدين %d", d)
	}
	if after := treasuryBalance(t, h, tid); after-before != 3000 {
		t.Errorf("الخزينة: %d ⟵ %d", before, after)
	}
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
		WHERE user_id = $1::uuid AND title LIKE '%دينك%'`, m.Owner.ID).Scan(&n)
	if n == 0 {
		t.Error("**لم يُبلَّغ المتجرُ أنّ شحنتَه سدّت دينَه**")
	}
}

// ── ٤ · وصفُ الاقتطاع في المحفظة من سبب الدين الحقيقيّ ───────────────────

func TestOBLX_OffsetNoteNamesTheRealCause(t *testing.T) {
	h := New(t)
	treasury(t, h)
	_, item := repFixture(t, h)
	legacyDebt(t, h, "merchant", item.MerchantID, 100)
	oid := placeAndDeliver(t, h, item)
	_, owner := merchantOf(t, h, oid)
	var note string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT note FROM wallet_transactions
		WHERE user_id = $1::uuid AND ref = $2 AND kind = 'merchant_earning' AND amount < 0`,
		owner, oid).Scan(&note); err != nil {
		t.Fatalf("لا اقتطاع: %v", err)
	}
	if strings.Contains(note, "بضاعة") || !strings.Contains(note, "دين سابق") {
		t.Errorf("**الوصفُ لا يقول السببَ الحقيقيّ**: %q", note)
	}
}

// ── ٥ · الملغى لا يُعرض مسدَّداً ─────────────────────────────────────────

func TestOBLX_VoidedDebtIsNotPaid(t *testing.T) {
	h := New(t)
	treasury(t, h)
	_, item := repFixture(t, h)
	oid := placeAndDeliver(t, h, item)
	if _, err := obligations.Create(ctxBG(), h.Pool, "merchant", item.MerchantID, 5000,
		obligations.CauseDeliveryFee, oid, nil); err != nil {
		t.Fatal(err)
	}
	if err := obligations.VoidForOrder(ctxBG(), h.Pool, "merchant", item.MerchantID, oid,
		obligations.CauseDeliveryFee, nil); err != nil {
		t.Fatal(err)
	}
	admin := h.NewUser("admin")
	list := h.GET("/api/v1/admin/obligations?party_id="+item.MerchantID+"&cause=merchant_delivery_fee",
		admin.Token).JSON()
	rows, _ := list["obligations"].([]any)
	if len(rows) != 1 {
		t.Fatalf("الصفوف: %v", list)
	}
	row := rows[0].(map[string]any)
	if row["state"] != "voided" {
		t.Errorf("**الملغى يُعرض بحال %v**", row["state"])
	}
	st, _ := row["settlements"].([]any)
	if len(st) != 1 || st[0].(map[string]any)["method"] != "voided" {
		t.Errorf("سطرُ التسوية: %v", st)
	}
	if row["order_id"] != oid {
		t.Errorf("رابطُ الطلب: %v", row["order_id"])
	}
}

// ── ٦ · البحثُ والمجاميعُ بالنوع والتنبيهُ بعد المهلة ─────────────────────

func TestOBLX_SearchTotalsAndOverdueAlert(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	rep := h.NewUser("sales")
	ob := legacyDebt(t, h, "merchant", m.ID, 7000)
	legacyDebt(t, h, "rep", rep.ID, 2000)
	h.Setting("finance.obligation_alert_days", "10")
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE financial_obligations
		SET created_at = now() - interval '11 days' WHERE id = $1::uuid`, ob); err != nil {
		t.Fatal(err)
	}
	admin := h.NewUser("admin")

	got := h.GET("/api/v1/admin/obligations?q="+m.Name, admin.Token).JSON()
	rows, _ := got["obligations"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["party_name"] != m.Name {
		t.Errorf("البحثُ بالاسم: %v", rows)
	}
	sum, _ := got["summary"].(map[string]any)
	if sum["merchants_total"].(float64) < 7000 || sum["reps_total"].(float64) < 2000 {
		t.Errorf("المجاميعُ بالنوع: %v", sum)
	}
	over := h.GET("/api/v1/admin/obligations?overdue=1", admin.Token).JSON()
	if orows, _ := over["obligations"].([]any); len(orows) != 1 {
		t.Errorf("المتأخّرُ: %d", len(orows))
	}
	ov := h.GET("/api/v1/admin/overview", admin.Token).JSON()
	aw, _ := ov["awaiting"].(map[string]any)
	if n, _ := aw["obligations_overdue"].(float64); n < 1 {
		t.Errorf("**الرئيسيّةُ لا تنبّه على دينٍ تجاوز المهلة**: %v", aw["obligations_overdue"])
	}
	if bad := h.GET("/api/v1/admin/obligations?state=nope", admin.Token); bad.Code != 400 {
		t.Errorf("حالٌ مجهول: %s", bad)
	}
}

// ── ٧ · الدينُ في تطبيق صاحبه ──────────────────────────────────────────

func TestOBLX_WalletShowsDebtToOwner(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	legacyDebt(t, h, "merchant", m.ID, 1500)
	got := h.GET("/api/v1/my/wallet", m.Owner.Token).JSON()
	if got["debt_total"] != float64(1500) {
		t.Errorf("**الدينُ غائبٌ عن محفظة صاحبه**: %v", got["debt_total"])
	}
	debts, _ := got["debts"].([]any)
	if len(debts) != 1 || debts[0].(map[string]any)["cause"] != "legacy_opening" {
		t.Errorf("السطر: %v", debts)
	}
	// **ومن لا دينَ عليه لا يرى حقلاً.**
	c := h.Customer()
	if plain := h.GET("/api/v1/my/wallet", c.Token).JSON(); plain["debts"] != nil {
		t.Errorf("زبونٌ بلا دينٍ رأى: %v", plain["debts"])
	}
}
