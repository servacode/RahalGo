package qa

import "testing"

// ══════════════════════════════════════════════════════════════════════
// **`XG-31` — الالتزامُ واقعةٌ تُقرأ لا رقمٌ يُزاد**
// ══════════════════════════════════════════════════════════════════════
//
// **وكلُّ فحصٍ هنا يسأل السؤالَ الذي عجز النظامُ عنه**: **من أين جاء
// هذا الرقم؟**

// obligation صورةُ واقعةِ نشأةٍ كما تُقرأ للمراجعة.
type obligation struct {
	ID      string
	Amount  int64
	Settled int64
	Cause   string
	Order   string
	Closed  bool
	Created string
	By      string
}

func obligationsOf(t *testing.T, h *Harness, party, id string) []obligation {
	t.Helper()
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT id::text, amount, settled, cause,
		       COALESCE(order_id::text, ''), closed_at IS NOT NULL,
		       created_at::text, COALESCE(created_by::text, '')
		  FROM financial_obligations
		 WHERE party_kind = $1 AND party_id = $2::uuid
		 ORDER BY created_at, id`, party, id)
	if err != nil {
		t.Fatalf("قراءةُ الالتزامات: %v", err)
	}
	defer rows.Close()
	var out []obligation
	for rows.Next() {
		var o obligation
		if err := rows.Scan(&o.ID, &o.Amount, &o.Settled, &o.Cause,
			&o.Order, &o.Closed, &o.Created, &o.By); err != nil {
			t.Fatalf("مسحُ الالتزام: %v", err)
		}
		out = append(out, o)
	}
	return out
}

// settlement سطرُ تسويةٍ كما يُقرأ للمراجعة.
type settlement struct {
	Amount    int64
	Remaining int64
	TxID      int64
	Order     string
}

func settlementsOf(t *testing.T, h *Harness, obID string) []settlement {
	t.Helper()
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT amount, remaining, COALESCE(ledger_tx_id, 0),
		       COALESCE(order_id::text, '')
		  FROM obligation_settlements
		 WHERE obligation_id = $1::uuid ORDER BY created_at, id`, obID)
	if err != nil {
		t.Fatalf("قراءةُ التسويات: %v", err)
	}
	defer rows.Close()
	var out []settlement
	for rows.Next() {
		var r settlement
		if err := rows.Scan(&r.Amount, &r.Remaining, &r.TxID, &r.Order); err != nil {
			t.Fatalf("مسحُ التسوية: %v", err)
		}
		out = append(out, r)
	}
	return out
}

func merchantOf(t *testing.T, h *Harness, orderID string) (mID, ownerID string) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT m.id::text, m.owner_user_id::text
		  FROM orders o JOIN merchants m ON m.id = o.merchant_id
		 WHERE o.id = $1::uuid`, orderID).Scan(&mID, &ownerID); err != nil {
		t.Fatalf("قراءةُ المتجر: %v", err)
	}
	return
}

func merchantEarned(t *testing.T, h *Harness, orderID string) int64 {
	t.Helper()
	var v int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'merchant_earning' AND amount > 0`,
		orderID).Scan(&v)
	return v
}

func refundOrder(t *testing.T, h *Harness, orderID string) int {
	t.Helper()
	admin := h.NewUser("admin")
	return h.POST("/api/v1/admin/orders/"+orderID+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "XG-31"}).Code
}

func first8(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

func first19(s string) string {
	if len(s) > 19 {
		return s[:19]
	}
	return s
}

// ══════════════════════════════════════════════════════════════════════
// **١ · رصيدٌ كافٍ ⇒ لا التزامَ أصلاً**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يُقيَّد التزامٌ صفريّ** — **ودفترٌ يمتلئ بأصفارٍ لا يُقرأ.**
func TestOBL_MerchantSufficient_NoObligation(t *testing.T) {
	h := New(t)
	treasury(t, h)
	_, item := repFixture(t, h)
	oid := placeAndDeliver(t, h, item)
	mID, _ := merchantOf(t, h, oid)

	if code := refundOrder(t, h, oid); code >= 400 {
		t.Fatalf("الاسترداد: %d", code)
	}
	if got := obligationsOf(t, h, "merchant", mID); len(got) != 0 {
		t.Errorf("نشأ التزامٌ ورصيدُ المتجر كافٍ: %+v", got)
	}
	t.Log("رصيدٌ كافٍ ⇒ عُكس مباشرةً ولا التزام")
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · رصيدٌ عاجز ⇒ التزامٌ يقول من أين**
// ══════════════════════════════════════════════════════════════════════
func TestOBL_MerchantInsufficient_OriginTraceable(t *testing.T) {
	h := New(t)
	treasury(t, h)
	_, item := repFixture(t, h)
	oid := placeAndDeliver(t, h, item)
	mID, ownerID := merchantOf(t, h, oid)

	earned := merchantEarned(t, h, oid)
	if earned <= 0 {
		t.Skip("لم يُقيَّد مستحقٌّ للمتجر")
	}
	h.Factory().Credit(ownerID, -earned, "payout")

	if code := refundOrder(t, h, oid); code >= 400 {
		t.Fatalf("الاسترداد: %d", code)
	}
	got := obligationsOf(t, h, "merchant", mID)
	if len(got) != 1 {
		t.Fatalf("الالتزاماتُ %d والمتوقَّع واحد: %+v", len(got), got)
	}
	o := got[0]
	t.Logf("مبلغٌ %d · سببٌ %q · طلبٌ %s · وقتٌ %s · منشئٌ %s · مغلقٌ %v",
		o.Amount, o.Cause, first8(o.Order), first19(o.Created), first8(o.By), o.Closed)

	if o.Amount != earned {
		t.Errorf("المبلغُ %d والمستحقُّ كان %d", o.Amount, earned)
	}
	if o.Cause != "refund_merchant_earning" {
		t.Errorf("السببُ %q", o.Cause)
	}
	if o.Order != oid {
		t.Errorf("الطلبُ %q والمتوقَّع %q", o.Order, oid)
	}
	if o.By == "" {
		t.Error("**لا منشئ** — ومن أنشأ الالتزامَ جزءٌ من تفسيره")
	}
	if o.Closed {
		t.Error("أُغلق وهو لم يُسدَّد")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · عمولةٌ قائمة ⇒ عكسٌ مباشرٌ بلا التزام**
// ══════════════════════════════════════════════════════════════════════
func TestOBL_RepAvailable_NoObligation(t *testing.T) {
	h := New(t)
	treasury(t, h)
	rep, item := repFixture(t, h)
	oid := placeAndDeliver(t, h, item)

	if code := refundOrder(t, h, oid); code >= 400 {
		t.Fatalf("الاسترداد: %d", code)
	}
	if got := obligationsOf(t, h, "rep", rep.ID); len(got) != 0 {
		t.Errorf("نشأ التزامٌ والعمولةُ كانت في محفظته: %+v", got)
	}
	t.Log("عمولةٌ قائمة ⇒ عُكست مباشرةً ولا التزام")
}

// ══════════════════════════════════════════════════════════════════════
// **٤ · عمولةٌ مسحوبة ⇒ التزامٌ يقول من أين**
// ══════════════════════════════════════════════════════════════════════
func TestOBL_RepWithdrawn_OriginTraceable(t *testing.T) {
	h := New(t)
	treasury(t, h)
	rep, item := repFixture(t, h)
	oid := placeAndDeliver(t, h, item)

	s0 := snapRep(t, h, rep.ID, oid)
	if s0.OrderNet <= 0 {
		t.Skip("لم تُقيَّد عمولة")
	}
	h.Factory().Credit(rep.ID, -s0.Balance, "payout")

	if code := refundOrder(t, h, oid); code >= 400 {
		t.Fatalf("الاسترداد: %d", code)
	}
	got := obligationsOf(t, h, "rep", rep.ID)
	if len(got) != 1 {
		t.Fatalf("الالتزاماتُ %d: %+v", len(got), got)
	}
	o := got[0]
	t.Logf("مبلغٌ %d · سببٌ %q · طلبٌ %s · وقتٌ %s",
		o.Amount, o.Cause, first8(o.Order), first19(o.Created))
	if o.Amount != s0.OrderNet || o.Cause != "refund_rep_commission" || o.Order != oid {
		t.Errorf("الواقعةُ لا تفسّر نفسَها: %+v (العمولةُ كانت %d)", o, s0.OrderNet)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · عاجزان معاً ⇒ الزبونُ يستردّ والالتزامان في معاملةٍ واحدة**
// ══════════════════════════════════════════════════════════════════════
func TestOBL_BothInsufficient_AtomicOrigins(t *testing.T) {
	h := New(t)
	treasury(t, h)
	rep, item := repFixture(t, h)
	oid := placeAndDeliver(t, h, item)
	mID, ownerID := merchantOf(t, h, oid)

	s0 := snapRep(t, h, rep.ID, oid)
	earned := merchantEarned(t, h, oid)
	if s0.OrderNet <= 0 || earned <= 0 {
		t.Skip("السيناريو يشترط مستحقّاً وعمولة")
	}
	h.Factory().Credit(ownerID, -earned, "payout")
	h.Factory().Credit(rep.ID, -s0.Balance, "payout")

	if code := refundOrder(t, h, oid); code >= 400 {
		t.Fatalf("**الاستردادُ سقط وكلاهما عاجز**: %d", code)
	}
	m := obligationsOf(t, h, "merchant", mID)
	r := obligationsOf(t, h, "rep", rep.ID)
	if len(m) != 1 || len(r) != 1 {
		t.Fatalf("الالتزامان: متجرٌ %d ومندوبٌ %d", len(m), len(r))
	}
	// **ووقتاهما من معاملةٍ واحدة** — و`now()` ثابتةٌ داخلها.
	if m[0].Created != r[0].Created {
		t.Errorf("**نشأا في معاملتين**: %s ≠ %s", m[0].Created, r[0].Created)
	}
	var refunded int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'refund'`, oid).Scan(&refunded)
	t.Logf("المستردُّ %d · التزامُ متجرٍ %d · التزامُ مندوبٍ %d · وقتٌ واحد %s",
		refunded, m[0].Amount, r[0].Amount, first19(m[0].Created))
	if refunded <= 0 {
		t.Error("**الزبونُ لم يستردّ**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٦ · إعادةُ النداء لا تُنشئ ثانياً**
// ══════════════════════════════════════════════════════════════════════
func TestOBL_ReplayCreatesNoDuplicate(t *testing.T) {
	h := New(t)
	treasury(t, h)
	rep, item := repFixture(t, h)
	oid := placeAndDeliver(t, h, item)
	s0 := snapRep(t, h, rep.ID, oid)
	if s0.OrderNet <= 0 {
		t.Skip("لم تُقيَّد عمولة")
	}
	h.Factory().Credit(rep.ID, -s0.Balance, "payout")

	if code := refundOrder(t, h, oid); code >= 400 {
		t.Fatalf("الأوّل: %d", code)
	}
	one := obligationsOf(t, h, "rep", rep.ID)
	code2 := refundOrder(t, h, oid)
	two := obligationsOf(t, h, "rep", rep.ID)

	t.Logf("بعد الأوّل %d التزاماً · النداءُ الثاني %d · بعده %d", len(one), code2, len(two))
	if len(two) != len(one) {
		t.Errorf("**تضاعفت الوقائع**: %d ← %d", len(one), len(two))
	}
	if len(two) == 1 && len(one) == 1 && two[0].Amount != one[0].Amount {
		t.Errorf("**المبلغُ تبدّل**: %d ← %d", one[0].Amount, two[0].Amount)
	}
	// **والحارسُ في القاعدة لا في الشيفرة** — يُثبَت بمحاولةٍ مباشرة.
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO financial_obligations (party_kind, party_id, amount, cause, order_id)
		VALUES ('rep', $1::uuid, 1, 'refund_rep_commission', $2::uuid)`,
		rep.ID, oid); err == nil {
		t.Error("**القاعدةُ قبلت نشأةً ثانيةً للطلب نفسِه** — والفهرسُ لا يحرس")
	} else {
		t.Log("القاعدةُ ردّت النشأةَ الثانية — الحارسُ في المخطَّط لا في ترتيب الشيفرة")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٧+٨ · عمولةٌ قادمةٌ تسدّد — بسطرٍ يقول كم ومن أين وكم بقي**
// ══════════════════════════════════════════════════════════════════════
func TestOBL_FutureEarningsSettleWithEvidence(t *testing.T) {
	h := New(t)
	treasury(t, h)
	rep, item := repFixture(t, h)
	oid1 := placeAndDeliver(t, h, item)
	s0 := snapRep(t, h, rep.ID, oid1)
	if s0.OrderNet <= 0 {
		t.Skip("لم تُقيَّد عمولة")
	}
	h.Factory().Credit(rep.ID, -s0.Balance, "payout")
	if code := refundOrder(t, h, oid1); code >= 400 {
		t.Fatalf("الاسترداد: %d", code)
	}
	born := obligationsOf(t, h, "rep", rep.ID)
	if len(born) != 1 {
		t.Fatalf("الالتزاماتُ %d", len(born))
	}

	// **طلبٌ ثانٍ ⇒ عمولةٌ جديدةٌ تُقتطَع منها.**
	oid2 := placeAndDeliver(t, h, item)
	after := obligationsOf(t, h, "rep", rep.ID)
	set := settlementsOf(t, h, born[0].ID)
	if len(set) == 0 {
		t.Fatal("**لا سطرَ تسوية** — والاقتطاعُ بلا دليل")
	}
	s := set[0]
	t.Logf("سُدّد %d · بقي %d · من طلبٍ %s · قيدٌ #%d · والالتزامُ مغلقٌ %v",
		s.Amount, s.Remaining, first8(s.Order), s.TxID, after[0].Closed)

	if s.Order != oid2 {
		t.Errorf("**سطرُ التسوية لا يقول من أين جاء المال**: %q", s.Order)
	}
	if s.TxID == 0 {
		t.Error("**لا وصلَ بقيد الدفتر** — والسطران متجاوران بلا رابط")
	}
	if s.Amount+s.Remaining != born[0].Amount {
		t.Errorf("الحسابُ لا يقفل: سُدّد %d وبقي %d والأصلُ %d",
			s.Amount, s.Remaining, born[0].Amount)
	}
	// **والقيدُ المُوصَل هو قيدُ الاقتطاع نفسُه.**
	var note string
	var amt int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT note, amount FROM wallet_transactions WHERE id = $1`,
		s.TxID).Scan(&note, &amt)
	t.Logf("القيدُ الموصول: %q بمقدار %d", note, amt)
	if amt >= 0 {
		t.Errorf("القيدُ الموصولُ ليس اقتطاعاً: %d", amt)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٩+١٠ · التزامان وتسويةٌ ثمّ إغلاق — والتاريخُ يبقى**
// ══════════════════════════════════════════════════════════════════════
//
// **والأقدمُ أوّلاً** — سياسةٌ مُعلَنةٌ لا مُستنتَجة.
func TestOBL_MultipleObligationsFIFO(t *testing.T) {
	h := New(t)
	treasury(t, h)
	rep, item := repFixture(t, h)

	// **يُسلَّم طلبان أوّلاً فتتراكم العمولتان** — **ثمّ يُسحب كلُّ
	// شيءٍ ثمّ يُستردّان.**
	//
	// **ولا يصحّ تسليمٌ ثمّ استردادٌ بالتناوب**: **عمولةُ الثاني
	// تُقتطَع فوراً لالتزامِ الأوّل فلا يبقى ما يُعكَس**، ولا ينشأ
	// التزامٌ ثانٍ أصلاً.
	oids := []string{placeAndDeliver(t, h, item), placeAndDeliver(t, h, item)}
	s0 := snapRep(t, h, rep.ID, oids[0])
	if s0.OrderNet <= 0 {
		t.Skip("لم تُقيَّد عمولة")
	}
	h.Factory().Credit(rep.ID, -s0.Balance, "payout")
	for i, oid := range oids {
		if code := refundOrder(t, h, oid); code >= 400 {
			t.Fatalf("الاستردادُ %d: %d", i, code)
		}
	}
	born := obligationsOf(t, h, "rep", rep.ID)
	if len(born) != 2 {
		t.Fatalf("الالتزاماتُ %d والمتوقَّع اثنان", len(born))
	}
	t.Logf("التزامان: %d و%d ⇒ المجموع %d",
		born[0].Amount, born[1].Amount, born[0].Amount+born[1].Amount)

	// **عمولةٌ قادمةٌ تسدّ الأقدمَ أوّلاً.**
	_ = placeAndDeliver(t, h, item)
	now := obligationsOf(t, h, "rep", rep.ID)
	var open int64
	for _, o := range now {
		open += o.Amount - o.Settled
	}
	t.Logf("بعد التسوية: الأوّلُ مغلقٌ %v (%d/%d) · الثاني مغلقٌ %v (%d/%d) · الباقي %d",
		now[0].Closed, now[0].Settled, now[0].Amount,
		now[1].Closed, now[1].Settled, now[1].Amount, open)

	if !now[0].Closed {
		t.Error("**الأقدمُ لم يُسدَّد أوّلاً** — والسياسةُ المعلَنةُ الأقدمُ أوّلاً")
	}
	if now[1].Settled != 0 {
		t.Errorf("مُسّ الثاني قبل إغلاق الأوّل: سُدّد منه %d", now[1].Settled)
	}
	// **والتاريخُ باقٍ بعد الإغلاق.**
	if len(settlementsOf(t, h, now[0].ID)) == 0 {
		t.Error("**التزامٌ مغلقٌ بلا سطرِ تسوية**")
	}

	// **والصورةُ تطابق الوقائع.**
	var cache int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT commission_debt FROM users WHERE id = $1::uuid`, rep.ID).Scan(&cache)
	if cache != open {
		t.Errorf("**الصورةُ فارقت الوقائع**: عمودٌ %d ووقائعُ %d", cache, open)
	}
	t.Logf("الصورةُ %d = الوقائعُ %d", cache, open)
}
