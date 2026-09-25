package qa

import (
	"context"
	"strconv"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **`COD` — سقفُ النقد غيرِ المسدَّدِ بذمّة الزبون**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك: الزبونُ لا يُراكم نقداً غيرَ مقبوضٍ فوق حدٍّ يضبطه الأدمن.)
//
// # العقد
//
// **ما بذمّته من نقدٍ مفتوح** (`closed_at IS NULL` · `payment_method='cash'`)
// **زائدَ نقدِ الطلب الجديد** — إن عبر `customers.cod_limit` رُدَّ الطلبُ
// بـ`cod_limit_exceeded` (409). **حدٌّ على المجموع لا على الطلب الواحد** فلا
// يُلتفّ عليه بالتفريق. **والعاديُّ يحمل نقدَه في `cash_due`، والخاصُّ في
// `total`.** **وصفرُ السقفِ يُطفئ الحارسَ** (الإنتاجُ لا يتغيّر). **والمحفظةُ
// لا تُمَسّ** — الحدُّ على النقد وحدَه.
//
// **والقفلُ `customer-admit:` معقودٌ قبل القياس** — فطلبان متزامنان لا يعبران
// المتّسعَ معاً (`COD-07`). **والمردودُ لا يخلّف طلباً ولا قيداً** (`COD-14`).

// codSetup **زبونٌ ومنصّةٌ بلا حارسٍ سوى النقد** — واتساب مطفأٌ، وسقفُ المفتوح
// معطّل، فلا يشوّش على قياس سقف النقد.
func codSetup(t *testing.T, h *Harness) *User {
	t.Helper()
	h.Setting("customers.require_whatsapp", "false")
	h.Setting("orders.max_open_per_customer", "0")
	treasury(t, h)
	return h.Customer()
}

// codExposure **مجموعُ النقد المفتوحِ بذمّة الزبون** — عينُ ما يقيسه الحارس.
func codExposure(t *testing.T, h *Harness, customerID string) int64 {
	t.Helper()
	var v int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(SUM(CASE WHEN kind = 'custom' THEN total ELSE cash_due END), 0)
		FROM orders
		WHERE customer_id = $1::uuid
		  AND payment_method = 'cash'
		  AND closed_at IS NULL`, customerID).Scan(&v); err != nil {
		t.Fatalf("قياسُ التعرّض: %v", err)
	}
	return v
}

// codDue **نقدُ طلبٍ بعينه** — `total` للخاصّ و`cash_due` للعاديّ.
func codDue(t *testing.T, h *Harness, orderID string) int64 {
	t.Helper()
	var v int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT CASE WHEN kind = 'custom' THEN total ELSE cash_due END
		   FROM orders WHERE id = $1::uuid`, orderID).Scan(&v); err != nil {
		t.Fatalf("قراءةُ نقد الطلب: %v", err)
	}
	return v
}

// codErr **رمزُ الخطأ من الجسم** — لتمييز ردّ السقف عن سواه.
func codErr(r Res) string {
	e, _ := r.JSON()["error"].(map[string]any)
	c, _ := e["code"].(string)
	return c
}

// codCreate **إنشاءُ طلبٍ عاديٍّ نقديٍّ بمفتاحٍ جديد.**
func codCreate(t *testing.T, h *Harness, cust *User, item *Item) Res {
	t.Helper()
	return h.POSTKey("/api/v1/orders", cust.Token, uniq("cod"), orderBody(item, 1))
}

// ══════════════════════════════════════════════════════════════════════
// **`COD-01` · طلبٌ دون المتّسع — يمرّ**
// ══════════════════════════════════════════════════════════════════════
func TestCOD01_BelowLimitAllowed(t *testing.T) {
	h := New(t)
	cust := codSetup(t, h)
	item := h.NewItem(1000)
	h.Setting("customers.cod_limit", "100000000")

	r := codCreate(t, h, cust, item)
	t.Logf("COD-01: سقفٌ واسعٌ ⇒ %d · تعرّضٌ=%d", r.Code, codExposure(t, h, cust.ID))
	if r.Code >= 400 {
		t.Fatalf("**رُدّ طلبٌ نقديٌّ دون السقف**: %s", r)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`COD-02` · طلبٌ واحدٌ فوق السقف — يُردّ خادميّاً**
// ══════════════════════════════════════════════════════════════════════
func TestCOD02_SingleAboveLimitRejected(t *testing.T) {
	h := New(t)
	cust := codSetup(t, h)
	item := h.NewItem(1000)
	h.Setting("customers.cod_limit", "1") // أيُّ طلبٍ يعبر الواحد

	r := codCreate(t, h, cust, item)
	t.Logf("COD-02: السقفُ 1 ⇒ %d %s", r.Code, r)
	if r.Code != 409 {
		t.Fatalf("**طلبٌ فوق السقف لم يُردّ** (%d)", r.Code)
	}
	if code := codErr(r); code != "cod_limit_exceeded" {
		t.Errorf("**رُدّ برمزٍ غير رمز السقف**: %q", code)
	}
	if open := codExposure(t, h, cust.ID); open != 0 {
		t.Errorf("**المردودُ ترك تعرّضاً**: %d", open)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`COD-04` · تعرّضٌ قائمٌ + جديدٌ يعبر — يُردّ**
// ══════════════════════════════════════════════════════════════════════
func TestCOD04_ExistingPlusNewExceeds(t *testing.T) {
	h := New(t)
	cust := codSetup(t, h)
	item := h.NewItem(1000)
	h.Setting("customers.cod_limit", "100000000")

	first := codCreate(t, h, cust, item)
	if first.Code >= 400 {
		t.Fatalf("الأوّل رُدّ: %s", first)
	}
	oid, _ := first.JSON()["id"].(string)
	tAmt := codDue(t, h, oid)

	// **السقفُ يتّسع لواحدٍ لا لاثنين**: `tAmt <= limit < 2*tAmt`.
	h.Setting("customers.cod_limit", strconv.FormatInt(2*tAmt-1, 10))
	second := codCreate(t, h, cust, item)
	t.Logf("COD-04: نقدُ الطلب=%d · السقفُ=%d · الثاني ⇒ %d",
		tAmt, 2*tAmt-1, second.Code)
	if second.Code != 409 {
		t.Errorf("**تعرّضٌ قائمٌ + جديدٌ عبر السقفَ ولم يُردّ** (%d)", second.Code)
	}
	if code := codErr(second); second.Code == 409 && code != "cod_limit_exceeded" {
		t.Errorf("**رُدّ برمزٍ غير رمز السقف**: %q", code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`COD-05` · تعرّضٌ قائمٌ + جديدٌ يبقى ضمن السقف — يمرّ**
// ══════════════════════════════════════════════════════════════════════
func TestCOD05_ExistingPlusNewWithinLimit(t *testing.T) {
	h := New(t)
	cust := codSetup(t, h)
	item := h.NewItem(1000)
	h.Setting("customers.cod_limit", "100000000")

	first := codCreate(t, h, cust, item)
	if first.Code >= 400 {
		t.Fatalf("الأوّل رُدّ: %s", first)
	}
	oid, _ := first.JSON()["id"].(string)
	tAmt := codDue(t, h, oid)

	// **السقفُ يتّسع لاثنين بالضبط** (`2*tAmt`) — والحدُّ شاملٌ (`<=`).
	h.Setting("customers.cod_limit", strconv.FormatInt(2*tAmt, 10))
	second := codCreate(t, h, cust, item)
	t.Logf("COD-05: نقدُ الطلب=%d · السقفُ=%d · الثاني ⇒ %d",
		tAmt, 2*tAmt, second.Code)
	if second.Code >= 400 {
		t.Errorf("**رُدّ طلبٌ يبقى المجموعُ به ضمن السقف** (%d) — والحدُّ شامل", second.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`COD-06` · إغلاقُ طلبٍ يردّ المتّسع** — `closed_at IS NULL`
// ══════════════════════════════════════════════════════════════════════
func TestCOD06_TerminalReleasesCapacity(t *testing.T) {
	h := New(t)
	cust := codSetup(t, h)
	item := h.NewItem(1000)
	h.Setting("customers.cod_limit", "100000000")

	first := codCreate(t, h, cust, item)
	if first.Code >= 400 {
		t.Fatalf("الأوّل رُدّ: %s", first)
	}
	oid, _ := first.JSON()["id"].(string)
	tAmt := codDue(t, h, oid)

	h.Setting("customers.cod_limit", strconv.FormatInt(2*tAmt-1, 10))
	if again := codCreate(t, h, cust, item).Code; again != 409 {
		t.Fatalf("**الثاني مرّ والسقفُ يتّسع لواحد** (%d)", again)
	}

	before := codExposure(t, h, cust.ID)
	// **والإغلاقُ من المسار الحقيقيّ** — إلغاءُ الزبون.
	cancel := h.POST("/api/v1/orders/"+oid+"/cancel", cust.Token,
		map[string]any{"reason": "COD"})
	if cancel.Code >= 400 {
		t.Fatalf("الإلغاء: %s", cancel)
	}
	after := codExposure(t, h, cust.ID)
	next := codCreate(t, h, cust, item).Code
	t.Logf("COD-06: تعرّضٌ قبلَ=%d · بعده=%d · التالي ⇒ %d", before, after, next)

	if after >= before {
		t.Errorf("**الإغلاقُ لم يُنقص التعرّض**: %d ⇒ %d", before, after)
	}
	if next >= 400 {
		t.Errorf("**المتّسعُ لم يُردّ بعد الإغلاق** (%d)", next)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`COD-07` · التزامنُ لا يتجاوز السقف**
// ══════════════════════════════════════════════════════════════════════
//
// **طلبان متزامنان لواحدٍ سعتُه طلبٌ واحد** — بلا قفلٍ يقرآن صفراً فيمرّان
// معاً. **والقفلُ `customer-admit:` يُسلسلهما** فلا يعبر إلّا واحد.
func TestCOD07_ConcurrentCannotExceed(t *testing.T) {
	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	h.Setting("orders.max_open_per_customer", "0")
	treasury(t, h)
	f := h.Factory()

	// **يُقاس نقدُ طلبٍ بزبونٍ منفصلٍ** فلا يُضاف تعرّضاً لزبون السباق.
	probe := h.Customer()
	item := h.NewItem(1000)
	h.Setting("customers.cod_limit", "100000000")
	pr := codCreate(t, h, probe, item)
	if pr.Code >= 400 {
		t.Fatalf("طلبُ القياس رُدّ: %s", pr)
	}
	poid, _ := pr.JSON()["id"].(string)
	tAmt := codDue(t, h, poid)

	// **السقفُ = نقدُ طلبٍ واحد** — فاثنان لا يمرّان معاً.
	h.Setting("customers.cod_limit", strconv.FormatInt(tAmt, 10))

	over, rounds := 0, 60
	for i := 0; i < rounds; i++ {
		cust := f.NewUserWith("customer")
		k1, k2 := uniq("cod-c1"), uniq("cod-c2")
		r := Race(t, DefaultRaceTimeout,
			Actor{Name: "الأوّل", Do: func(context.Context) any {
				return h.POSTKey("/api/v1/orders", cust.Token, k1, orderBody(item, 1))
			}},
			Actor{Name: "الثاني", Do: func(context.Context) any {
				return h.POSTKey("/api/v1/orders", cust.Token, k2, orderBody(item, 1))
			}},
		)
		if r.TimedOut {
			t.Fatalf("السباقُ عَلِق — %s", r)
		}
		if exp := codExposure(t, h, cust.ID); exp > tAmt {
			over++
			if over == 1 {
				t.Logf("COD-07: تجاوزٌ في الجولة %d — تعرّضٌ=%d سقفٌ=%d · %s",
					i, exp, tAmt, r)
			}
		}
	}
	t.Logf("COD-07: تزامنٌ ×%d — تجاوزٌ في %d", rounds, over)
	if over != 0 {
		t.Errorf("**التزامنُ تجاوز السقف**: %d من %d — "+
			"**والقياسُ ثمّ الإدراجُ ليس ذرّيّاً.**", over, rounds)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`COD-08` · المحفظةُ تبقى مفتوحةً وإن سُدّ النقد**
// ══════════════════════════════════════════════════════════════════════
func TestCOD08_WalletAllowedWhenCashBlocked(t *testing.T) {
	h := New(t)
	cust := codSetup(t, h)
	item := h.NewItem(1000)
	h.Factory().Credit(cust.ID, 500_000, "topup")
	h.Setting("customers.cod_limit", "1") // النقدُ مسدود

	body := orderBody(item, 1)
	body["payment_method"] = "wallet"
	r := h.POSTKey("/api/v1/orders", cust.Token, uniq("cod8"), body)
	t.Logf("COD-08: السقفُ 1 · دفعٌ بالمحفظة ⇒ %d", r.Code)
	if r.Code >= 400 {
		t.Errorf("**مُنعت المحفظةُ والحدُّ على النقد وحدَه**: %s", r)
	}
	// **ولا يُحوَّل النقدُ صامتاً إلى محفظة**: الطلبُ النقديُّ يُردّ لا يُبدَّل — `COD-09`.
	cash := codCreate(t, h, cust, item)
	if cash.Code != 409 {
		t.Errorf("**طلبٌ نقديٌّ فوق السقف لم يُردّ** (%d) — لا تحويلَ صامت", cash.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`COD-10` · الطلبُ الخاصُّ عند التأكيد يخضع للسقف نفسِه**
// ══════════════════════════════════════════════════════════════════════
//
// **لا سعرَ قبل عرض السائق** — فالحارسُ عند التأكيد لا عند الإنشاء. **وتعرّضٌ
// قائمٌ من طلبٍ عاديٍّ + مبلغُ الخاصّ المتّفقُ عليه** إن عبر السقفَ رُدَّ التأكيد.
func TestCOD10_CustomConfirmOverLimitRejected(t *testing.T) {
	h := New(t)
	cust := codSetup(t, h)
	item := h.NewItem(1000)
	h.Setting("customers.cod_limit", "100000000")

	// تعرّضٌ قائمٌ من طلبٍ عاديٍّ نقديّ.
	base := codCreate(t, h, cust, item)
	if base.Code >= 400 {
		t.Fatalf("الطلبُ الأساس رُدّ: %s", base)
	}
	boid, _ := base.JSON()["id"].(string)
	tAmt := codDue(t, h, boid)

	// طلبٌ خاصٌّ يُنشأ ثمّ يُتّفق على مبلغه.
	cx := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("cod10"),
		customBody("أدويةٌ من الصيدليّة — خاصّ"))
	if cx.Code >= 400 {
		t.Fatalf("إنشاءُ الخاصّ رُدّ: %s", cx)
	}
	oid, _ := cx.JSON()["id"].(string)
	drv := h.driverOf(oid)
	var goods, fee int64 = 5000, 1000
	c := goods + fee
	agree := h.POST("/api/v1/driver/orders/"+oid+"/agree", drv.Token,
		map[string]any{"goods_amount": goods, "fee": fee})
	if agree.Code >= 400 {
		t.Fatalf("توثيقُ العرض رُدّ: %s", agree)
	}
	var ver int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT quote_version FROM orders WHERE id = $1::uuid`, oid).Scan(&ver); err != nil {
		t.Fatalf("قراءةُ نسخة العرض: %v", err)
	}

	// **السقفُ يتّسع للعاديّ لا للعاديّ والخاصّ معاً.**
	h.Setting("customers.cod_limit", strconv.FormatInt(tAmt+c-1, 10))
	confirm := h.POST("/api/v1/orders/"+oid+"/confirm-quote", cust.Token,
		map[string]any{"payment_method": "cash", "expected_total": c, "expected_quote_version": ver})
	t.Logf("COD-10: تعرّضٌ=%d · مبلغُ الخاصّ=%d · السقفُ=%d · التأكيدُ ⇒ %d",
		tAmt, c, tAmt+c-1, confirm.Code)
	if confirm.Code != 409 {
		t.Errorf("**تأكيدُ خاصٍّ نقديٍّ عبر السقفَ ولم يُردّ** (%d)", confirm.Code)
	}
	if code := codErr(confirm); confirm.Code == 409 && code != "cod_limit_exceeded" {
		t.Errorf("**رُدّ برمزٍ غير رمز السقف**: %q", code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`COD-11` · تأكيدُ الخاصِّ بالمحفظة لا يمسّه سقفُ النقد**
// ══════════════════════════════════════════════════════════════════════
//
// **وعقدُ Batch-2 كما هو**: الحجزُ والتأكيدُ والنسخة — لا يغيّرها الحارس.
func TestCOD11_CustomWalletConfirmUnaffected(t *testing.T) {
	h := New(t)
	cust := codSetup(t, h)
	h.Factory().Credit(cust.ID, 500_000, "topup")
	h.Setting("customers.cod_limit", "1") // النقدُ مسدودٌ تماماً

	cx := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("cod11"),
		customBody("طلبٌ خاصٌّ يُدفع من المحفظة"))
	if cx.Code >= 400 {
		t.Fatalf("إنشاءُ الخاصّ رُدّ: %s", cx)
	}
	oid, _ := cx.JSON()["id"].(string)
	drv := h.driverOf(oid)
	var goods, fee int64 = 5000, 1000
	c := goods + fee
	if a := h.POST("/api/v1/driver/orders/"+oid+"/agree", drv.Token,
		map[string]any{"goods_amount": goods, "fee": fee}); a.Code >= 400 {
		t.Fatalf("توثيقُ العرض رُدّ: %s", a)
	}
	var ver int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT quote_version FROM orders WHERE id = $1::uuid`, oid).Scan(&ver); err != nil {
		t.Fatalf("قراءةُ نسخة العرض: %v", err)
	}
	confirm := h.POST("/api/v1/orders/"+oid+"/confirm-quote", cust.Token,
		map[string]any{"payment_method": "wallet", "expected_total": c, "expected_quote_version": ver})
	t.Logf("COD-11: السقفُ 1 · تأكيدٌ بالمحفظة ⇒ %d", confirm.Code)
	if confirm.Code >= 400 {
		t.Errorf("**مُنع تأكيدُ الخاصِّ بالمحفظة وسقفُ النقد لا يمسّها**: %s", confirm)
	}
	var pm string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT payment_method FROM orders WHERE id = $1::uuid`, oid).Scan(&pm); err == nil && pm != "wallet" {
		t.Errorf("طريقةُ الدفع ليست محفظة: %q", pm)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`COD-12` · تغييرُ السقف من الأدمن يعمل حيّاً بلا نسخةٍ جديدة**
// ══════════════════════════════════════════════════════════════════════
func TestCOD12_SettingChangeTakesEffectLive(t *testing.T) {
	h := New(t)
	cust := codSetup(t, h)
	item := h.NewItem(1000)

	h.Setting("customers.cod_limit", "100000000")
	if r := codCreate(t, h, cust, item); r.Code >= 400 {
		t.Fatalf("**رُدّ طلبٌ والسقفُ واسع**: %s", r)
	}
	// **يُخفَض السقفُ حيّاً** — فيُردُّ التالي بلا إعادة إقلاعٍ ولا تطبيقٍ جديد.
	h.Setting("customers.cod_limit", "1")
	blocked := codCreate(t, h, cust, item)
	if blocked.Code != 409 {
		t.Errorf("**خفضُ السقف لم يسرِ حيّاً** (%d)", blocked.Code)
	}
	// **ثمّ يُرفَع فيمرّ ثانيةً** — أثرُ الإعداد فوريٌّ في الاتجاهين.
	h.Setting("customers.cod_limit", "100000000")
	if r := codCreate(t, h, cust, item); r.Code >= 400 {
		t.Errorf("**رفعُ السقف لم يسرِ حيّاً**: %s", r)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`COD-14` · المردودُ بالسقف لا يترك أثراً**
// ══════════════════════════════════════════════════════════════════════
//
// **لا طلبَ ولا قيدَ محفظةٍ ولا حدثَ** — ورفضُ قبولٍ ليس نصفَ إنشاء.
func TestCOD14_RejectedLeavesNothing(t *testing.T) {
	h := New(t)
	cust := codSetup(t, h)
	item := h.NewItem(1000)
	h.Setting("customers.cod_limit", "1")

	count := func(table string) int64 {
		t.Helper()
		var n int64
		if err := h.Pool.QueryRow(ctxBG(),
			`SELECT count(*) FROM `+table).Scan(&n); err != nil {
			t.Fatalf("عدُّ %s: %v", table, err)
		}
		return n
	}
	ordersBefore := h.CountOrders(cust.ID)
	moneyBefore := count("wallet_transactions")
	eventsBefore := count("order_events")

	r := codCreate(t, h, cust, item)
	if r.Code != 409 {
		t.Fatalf("**المردودُ لم يُردّ** (%d)", r.Code)
	}

	ordersAfter := h.CountOrders(cust.ID)
	moneyAfter := count("wallet_transactions")
	eventsAfter := count("order_events")
	t.Logf("COD-14: طلباتٌ %d⇒%d · مالٌ %d⇒%d · أحداثٌ %d⇒%d",
		ordersBefore, ordersAfter, moneyBefore, moneyAfter, eventsBefore, eventsAfter)

	if ordersAfter != ordersBefore {
		t.Errorf("**المردودُ أنشأ طلباً**: %d ⇒ %d", ordersBefore, ordersAfter)
	}
	if moneyAfter != moneyBefore {
		t.Errorf("**المردودُ حرّك مالاً**: %d ⇒ %d", moneyBefore, moneyAfter)
	}
	if eventsAfter != eventsBefore {
		t.Errorf("**المردودُ كتب حدثاً**: %d ⇒ %d", eventsBefore, eventsAfter)
	}
}
