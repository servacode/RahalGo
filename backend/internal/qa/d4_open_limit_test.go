package qa

import (
	"context"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **`D4` — سقفُ المفتوح كان تحت شرط واتساب**
// ══════════════════════════════════════════════════════════════════════
//
// **`checkOpenLimit` كانت داخلَ `if RequireWhatsApp`** — **فإطفاءُ
// التوثيق يُطفئ السقفَ معه.** **وقيس: السقفُ اثنان وواتساب مطفأ ⇒
// خمسةُ طلباتٍ مرّت كلُّها.**
//
// **وهما حارسان لا حارس**: **التوثيقُ شرطٌ اختياريٌّ يُشغّله المالك**،
// **والسقفُ ثابتُ قبولٍ لا يُساوم عليه إعداد.**
//
// **وعطبٌ ثانٍ في الجذر نفسِه**: **العدُّ ثمّ الإدراج ليس ذرّيّاً** —
// **السقفُ واحدٌ وطلبان متزامنان ⇒ مفتوحان.**

// d4Setup **زبونٌ وصنفٌ وسقفٌ وحالُ واتساب.**
func d4Setup(t *testing.T, h *Harness, whatsapp, cap string) (*User, *Item) {
	t.Helper()
	h.Setting("customers.require_whatsapp", whatsapp)
	h.Setting("orders.max_open_per_customer", cap)
	treasury(t, h)
	return h.Customer(), h.NewItem(1000)
}

// d4Open **المفتوحُ وحدَه** — `closed_at IS NULL`.
//
// **و`CountOrders` تعدّ كلَّ ما أنشأه** — **فلا تصلح لقياس خانةٍ
// رُدَّت بالإغلاق.**
func d4Open(t *testing.T, h *Harness, customerID string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM orders
		 WHERE customer_id = $1::uuid AND closed_at IS NULL`, customerID).Scan(&n); err != nil {
		t.Fatalf("عدُّ المفتوح: %v", err)
	}
	return n
}

// d4Create **إنشاءُ طلبٍ عاديٍّ بمفتاحٍ جديد** — طلبٌ متمايزٌ لا إعادة.
func d4Create(t *testing.T, h *Harness, cust *User, item *Item) Res {
	t.Helper()
	return h.POSTKey("/api/v1/orders", cust.Token, uniq("d4"), orderBody(item, 1))
}

// ══════════════════════════════════════════════════════════════════════
// **`D4-T1` · واتساب مطفأ — والسقفُ يعمل**
// ══════════════════════════════════════════════════════════════════════
func TestD4_CapHoldsWhenWhatsAppIsOff(t *testing.T) {
	h := New(t)
	cust, item := d4Setup(t, h, "false", "2")

	codes := []int{}
	for i := 0; i < 5; i++ {
		codes = append(codes, d4Create(t, h, cust, item).Code)
	}
	open := d4Open(t, h, cust.ID)
	t.Logf("D4-T1: واتساب مطفأ · السقفُ 2 · خمسُ محاولات ⇒ %v · مفتوحٌ=%d",
		codes, open)

	if codes[0] >= 400 || codes[1] >= 400 {
		t.Fatalf("**رُدَّ ما دون السقف**: %v", codes)
	}
	for i := 2; i < 5; i++ {
		if codes[i] != 409 {
			t.Errorf("**المحاولةُ %d مرّت والسقفُ ممتلئ** (%d) — "+
				"**وإطفاءُ واتساب يُطفئ السقف**: `D4`.", i+1, codes[i])
		}
	}
	if open != 2 {
		t.Errorf("**المفتوحُ %d والسقفُ 2**", open)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D4-T2`+`T3` · وواتساب مشتغلٌ لا يُبدّل السقف**
// ══════════════════════════════════════════════════════════════════════
func TestD4_CapIndependentOfWhatsAppSetting(t *testing.T) {
	h := New(t)
	cust, item := d4Setup(t, h, "true", "2")
	// **ومِسنَدُ الفحص يُوثّق الزبونَ افتراضاً** — **فيُنزَع توثيقُه
	// ليُقاس عقدُ واتساب على حاله الحقيقيّة.**
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE users SET whatsapp_verified_at = NULL WHERE id = $1::uuid`,
		cust.ID); err != nil {
		t.Fatalf("نزعُ التوثيق: %v", err)
	}

	// ── `T3` · غيرُ موثَّقٍ يُردّ بعقد واتساب لا بعقد السقف ───────
	first := d4Create(t, h, cust, item)
	t.Logf("D4-T3: واتساب مشتغل · غيرُ موثَّق ⇒ %d %s", first.Code, first)
	if first.Code < 400 {
		t.Fatalf("**غيرُ موثَّقٍ مرّ** (%d)", first.Code)
	}
	if code, _ := first.JSON()["error"].(map[string]any)["code"].(string); code != "whatsapp_required" {
		t.Errorf("**رُدّ بغير عقد واتساب**: %q", code)
	}

	// ── `T2` · وموثَّقٌ يخضع للسقف نفسِه ─────────────────────────
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE users SET whatsapp_verified_at = now() WHERE id = $1::uuid`,
		cust.ID); err != nil {
		t.Fatalf("توثيقُ الحساب: %v", err)
	}
	codes := []int{}
	for i := 0; i < 3; i++ {
		codes = append(codes, d4Create(t, h, cust, item).Code)
	}
	t.Logf("D4-T2: واتساب مشتغل · موثَّق · السقفُ 2 ⇒ %v", codes)
	if codes[0] >= 400 || codes[1] >= 400 {
		t.Fatalf("**رُدَّ ما دون السقف**: %v", codes)
	}
	if codes[2] != 409 {
		t.Errorf("**الثالثُ مرّ والسقفُ 2** (%d)", codes[2])
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D4-T5` · وحدُّ السقف بعينه**
// ══════════════════════════════════════════════════════════════════════
func TestD4_BoundaryAtLimit(t *testing.T) {
	h := New(t)
	cust, item := d4Setup(t, h, "false", "2")

	one := d4Create(t, h, cust, item).Code
	two := d4Create(t, h, cust, item).Code
	three := d4Create(t, h, cust, item).Code
	t.Logf("D4-T5: مفتوحٌ 0⇒%d · 1⇒%d · 2⇒%d", one, two, three)
	if one >= 400 || two >= 400 {
		t.Errorf("**رُدَّ ما دون السقف**: %d ثمّ %d", one, two)
	}
	if three != 409 {
		t.Errorf("**عند السقف مرّ** (%d)", three)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D4-T4` · وإغلاقُ طلبٍ يردّ خانتَه**
// ══════════════════════════════════════════════════════════════════════
//
// **والسقفُ على المفتوح لا على ما مضى.**
func TestD4_ClosingAnOrderReleasesCapacity(t *testing.T) {
	h := New(t)
	cust, item := d4Setup(t, h, "false", "1")

	made := d4Create(t, h, cust, item)
	if made.Code >= 400 {
		t.Fatalf("الأوّل: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	if again := d4Create(t, h, cust, item).Code; again != 409 {
		t.Fatalf("**الثاني مرّ والسقفُ 1** (%d)", again)
	}

	before := d4Open(t, h, cust.ID)
	// **والإغلاقُ من المسار الحقيقيّ** — إلغاءُ الزبون.
	cancel := h.POST("/api/v1/orders/"+oid+"/cancel", cust.Token,
		map[string]any{"reason": "D4"})
	if cancel.Code >= 400 {
		t.Fatalf("الإلغاء: %s", cancel)
	}
	after := d4Open(t, h, cust.ID)
	next := d4Create(t, h, cust, item).Code
	t.Logf("D4-T4: قبلَ الإغلاق=%d · بعده=%d · التالي ⇒ %d", before, after, next)

	if after >= before {
		t.Errorf("**الإغلاقُ لم يُنقص المفتوح**: %d ⇒ %d", before, after)
	}
	if next >= 400 {
		t.Errorf("**الخانةُ لم تُردّ بعد الإغلاق** (%d)", next)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D4-T6` · وخفضُ السقف لا يُلغي ما بيده**
// ══════════════════════════════════════════════════════════════════════
func TestD4_LoweringCapKeepsExistingOrders(t *testing.T) {
	h := New(t)
	cust, item := d4Setup(t, h, "false", "3")

	for i := 0; i < 3; i++ {
		if c := d4Create(t, h, cust, item).Code; c >= 400 {
			t.Fatalf("الطلبُ %d: %d", i+1, c)
		}
	}
	h.Setting("orders.max_open_per_customer", "1")
	open := d4Open(t, h, cust.ID)
	next := d4Create(t, h, cust, item).Code
	t.Logf("D4-T6: خُفض السقفُ إلى 1 · المفتوحُ=%d · التالي ⇒ %d", open, next)

	if open != 3 {
		t.Errorf("**خفضُ السقف ألغى طلباتٍ قائمة**: %d", open)
	}
	if next != 409 {
		t.Errorf("**مرّ طلبٌ جديدٌ فوق السقف الجديد** (%d)", next)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D4-T7` · وإعادةُ المفتاح نفسِه تسترجع الطلبَ لا تُردّ بالسقف**
// ══════════════════════════════════════════════════════════════════════
//
// **وعقدُ الردّ الضائع** (`F-03`): **من ضاع ردُّه يُعيد النداءَ
// بالمفتاح نفسِه فيجد طلبَه** — **ولا يُقال له «سقفُك ممتلئ» بسببِ
// طلبٍ هو نفسُه.**
func TestD4_IdempotentRetryRecoversSameOrder(t *testing.T) {
	h := New(t)
	cust, item := d4Setup(t, h, "false", "1")

	key := uniq("d4-idem")
	first := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	if first.Code >= 400 {
		t.Fatalf("الأوّل: %s", first)
	}
	firstID, _ := first.JSON()["id"].(string)

	retry := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	retryID, _ := retry.JSON()["id"].(string)
	open := d4Open(t, h, cust.ID)
	t.Logf("D4-T7: الإعادةُ ⇒ %d · المعرّفُ نفسُه=%t · مفتوحٌ=%d",
		retry.Code, firstID != "" && firstID == retryID, open)

	if retry.Code == 409 {
		t.Errorf("**الإعادةُ رُدَّت بالسقف** — **وطلبُه هو ما ملأه.**")
	}
	if retry.Code >= 400 {
		t.Errorf("**الإعادةُ رُدَّت** (%d)", retry.Code)
	}
	if retryID != firstID {
		t.Errorf("**الإعادةُ لم تسترجع الطلبَ نفسَه**: %q ≠ %q", retryID, firstID)
	}
	if open != 1 {
		t.Errorf("**الإعادةُ أنشأت طلباً ثانياً**: %d", open)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D4-T8` · ومفاتيحُ متمايزةٌ طلباتٌ متمايزةٌ يحكمها السقف**
// ══════════════════════════════════════════════════════════════════════
func TestD4_DistinctKeysAreCapped(t *testing.T) {
	h := New(t)
	cust, item := d4Setup(t, h, "false", "1")

	first := d4Create(t, h, cust, item).Code
	second := d4Create(t, h, cust, item).Code
	open := d4Open(t, h, cust.ID)
	t.Logf("D4-T8: مفتاحان متمايزان ⇒ %d ثمّ %d · مفتوحٌ=%d", first, second, open)
	if first >= 400 {
		t.Fatalf("الأوّل رُدّ: %d", first)
	}
	if second != 409 {
		t.Errorf("**مفتاحٌ متمايزٌ التفّ على السقف** (%d)", second)
	}
	if open != 1 {
		t.Errorf("**المفتوحُ %d والسقفُ 1**", open)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D4-T9` · والمردودُ لا يترك أثراً**
// ══════════════════════════════════════════════════════════════════════
//
// **لا طلبَ ولا لقطةَ مالٍ ولا قيدَ ولا حدثَ بثٍّ** — **ورفضُ قبولٍ
// ليس نصفَ إنشاء.**
func TestD4_RejectedCreateLeavesNothing(t *testing.T) {
	h := New(t)
	cust, item := d4Setup(t, h, "false", "1")

	if c := d4Create(t, h, cust, item).Code; c >= 400 {
		t.Fatalf("الأوّل: %d", c)
	}
	ordersBefore := h.CountOrders(cust.ID)
	count := func(table string) int64 {
		t.Helper()
		var n int64
		if err := h.Pool.QueryRow(ctxBG(),
			`SELECT count(*) FROM `+table).Scan(&n); err != nil {
			t.Fatalf("عدُّ %s: %v", table, err)
		}
		return n
	}
	moneyBefore := count("wallet_transactions")
	eventsBefore := count("order_events")

	res := d4Create(t, h, cust, item)
	if res.Code != 409 {
		t.Fatalf("**المردودُ لم يُردّ** (%d)", res.Code)
	}

	ordersAfter := h.CountOrders(cust.ID)
	moneyAfter := count("wallet_transactions")
	eventsAfter := count("order_events")
	t.Logf("D4-T9: طلباتٌ %d⇒%d · مالٌ %d⇒%d · أحداثٌ %d⇒%d",
		ordersBefore, ordersAfter, moneyBefore, moneyAfter, eventsBefore, eventsAfter)

	if ordersAfter != ordersBefore {
		t.Errorf("**المردودُ أنشأ طلباً**: %d ⇒ %d", ordersBefore, ordersAfter)
	}
	if moneyAfter != moneyBefore {
		t.Errorf("**المردودُ حرّك مالاً**: %d ⇒ %d", moneyBefore, moneyAfter)
	}
	if eventsAfter != eventsBefore {
		t.Errorf("**المردودُ كتب حدثَ طلب**: %d ⇒ %d", eventsBefore, eventsAfter)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D4` · التزامنُ لا يتجاوز السقف**
// ══════════════════════════════════════════════════════════════════════
//
// **وقيس قبل الإصلاح: السقفُ واحدٌ ⇒ مفتوحان.**
func TestD4_ConcurrentCreatesCannotExceedCap(t *testing.T) {
	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	h.Setting("orders.max_open_per_customer", "1")
	treasury(t, h)
	f := h.Factory()

	over, rounds := 0, 100
	for i := 0; i < rounds; i++ {
		cust := f.NewUserWith("customer")
		item := h.NewItem(1000)
		k1, k2 := uniq("d4-c1"), uniq("d4-c2")
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
		if open := d4Open(t, h, cust.ID); open > 1 {
			over++
			if over == 1 {
				t.Logf("D4: تجاوزٌ في الجولة %d — مفتوحٌ=%d · %s", i, open, r)
			}
		}
	}
	t.Logf("D4: تزامنٌ ×%d — تجاوزٌ في %d", rounds, over)
	if over != 0 {
		t.Errorf("**التزامنُ تجاوز السقف**: %d من %d — "+
			"**والعدُّ ثمّ الإدراجُ ليس ذرّيّاً.**", over, rounds)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D4-T10` · وزبونان لا يتسلسلان**
// ══════════════════════════════════════════════════════════════════════
//
// **والقفلُ على الزبون وحدَه** — **ومن قفل الإنشاءَ كلَّه جعل المنصّةَ
// طابوراً واحداً.**
func TestD4_DifferentCustomersAreNotSerialized(t *testing.T) {
	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	h.Setting("orders.max_open_per_customer", "1")
	treasury(t, h)
	f := h.Factory()

	a := f.NewUserWith("customer")
	b := f.NewUserWith("customer")
	item := h.NewItem(1000)
	ka, kb := uniq("d4-a"), uniq("d4-b")

	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "زبونٌ أ", Do: func(context.Context) any {
			return h.POSTKey("/api/v1/orders", a.Token, ka, orderBody(item, 1))
		}},
		Actor{Name: "زبونٌ ب", Do: func(context.Context) any {
			return h.POSTKey("/api/v1/orders", b.Token, kb, orderBody(item, 1))
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	openA, openB := d4Open(t, h, a.ID), d4Open(t, h, b.ID)
	t.Logf("D4-T10: أ=%d · ب=%d — %s", openA, openB, r)
	if openA != 1 || openB != 1 {
		t.Errorf("**زبونٌ حُرم بسبب زبونٍ آخر**: أ=%d · ب=%d", openA, openB)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D4` · والطلبُ الخاصُّ يشترك في السقف نفسِه**
// ══════════════════════════════════════════════════════════════════════
//
// **وهو حدُّ النطاق**: **`D6` و`D8` و`D9` عقودٌ أخرى في مسار الخاصّ
// ولا تُجرّ إلى هنا** — **وهذا سقفُ المفتوح بعينه**، **قاعدةٌ واحدةٌ
// لبابين** (`checkOpenLimit`).
//
// **ولا يُترك بابٌ يلتفّ به عليه** — **ولا يُخترَع له عيبٌ ثانٍ
// لاحقاً بالقاعدة نفسِها.**
func TestD4_CustomOrdersShareTheSameCap(t *testing.T) {
	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	h.Setting("orders.max_open_per_customer", "1")
	treasury(t, h)
	cust := h.Customer()

	first := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("d4-x1"),
		customBody("طلبٌ خاصٌّ أوّل"))
	second := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("d4-x2"),
		customBody("طلبٌ خاصٌّ ثانٍ"))
	open := d4Open(t, h, cust.ID)
	t.Logf("D4/خاصّ: %d ثمّ %d · مفتوحٌ=%d", first.Code, second.Code, open)

	if first.Code >= 400 {
		t.Fatalf("الأوّل رُدّ: %s", first)
	}
	if second.Code != 409 {
		t.Errorf("**الخاصُّ التفّ على السقف** (%d)", second.Code)
	}
	if open != 1 {
		t.Errorf("**المفتوحُ %d والسقفُ 1**", open)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **والتزامنُ في بابِ الخاصّ لا يتجاوزه كذلك**
// ══════════════════════════════════════════════════════════════════════
func TestD4_ConcurrentCustomCreatesCannotExceedCap(t *testing.T) {
	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	h.Setting("orders.max_open_per_customer", "1")
	treasury(t, h)
	f := h.Factory()

	over, rounds := 0, 30
	for i := 0; i < rounds; i++ {
		cust := f.NewUserWith("customer")
		k1, k2 := uniq("d4-xc1"), uniq("d4-xc2")
		r := Race(t, DefaultRaceTimeout,
			Actor{Name: "خاصٌّ أوّل", Do: func(context.Context) any {
				return h.POSTKey("/api/v1/orders/custom", cust.Token, k1,
					customBody("خاصٌّ متزامنٌ أوّل"))
			}},
			Actor{Name: "خاصٌّ ثانٍ", Do: func(context.Context) any {
				return h.POSTKey("/api/v1/orders/custom", cust.Token, k2,
					customBody("خاصٌّ متزامنٌ ثانٍ"))
			}},
		)
		if r.TimedOut {
			t.Fatalf("السباقُ عَلِق — %s", r)
		}
		if open := d4Open(t, h, cust.ID); open > 1 {
			over++
			if over == 1 {
				t.Logf("D4/خاصّ: تجاوزٌ في الجولة %d — %s", i, r)
			}
		}
	}
	t.Logf("D4/خاصّ: تزامنٌ ×%d — تجاوزٌ في %d", rounds, over)
	if over != 0 {
		t.Errorf("**بابُ الخاصّ يتجاوز السقفَ بالتزامن**: %d من %d", over, rounds)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **تكرارُ `D4`** — **مرّةٌ تُصادِف، والمئةُ تحكم**
// ══════════════════════════════════════════════════════════════════════

func TestD4_StressSequentialCap(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	item := h.NewItem(1000)

	for _, wa := range []struct {
		flag   string
		rounds int
		verify bool
	}{{"false", 100, false}, {"true", 50, true}} {
		h.Setting("customers.require_whatsapp", wa.flag)
		h.Setting("orders.max_open_per_customer", "2")
		bypass, falseReject := 0, 0
		for i := 0; i < wa.rounds; i++ {
			cust := f.NewUserWith("customer")
			if wa.verify {
				if _, err := h.Pool.Exec(ctxBG(),
					`UPDATE users SET whatsapp_verified_at = now() WHERE id = $1::uuid`,
					cust.ID); err != nil {
					t.Fatalf("التوثيق: %v", err)
				}
			}
			c1 := h.POSTKey("/api/v1/orders", cust.Token, uniq("s1"), orderBody(item, 1)).Code
			c2 := h.POSTKey("/api/v1/orders", cust.Token, uniq("s2"), orderBody(item, 1)).Code
			c3 := h.POSTKey("/api/v1/orders", cust.Token, uniq("s3"), orderBody(item, 1)).Code
			if c1 >= 400 || c2 >= 400 {
				falseReject++
			}
			if c3 != 409 {
				bypass++
			}
		}
		t.Logf("D4/تكرار: واتساب=%s ×%d — التفافٌ=%d · ردٌّ كاذبٌ=%d",
			wa.flag, wa.rounds, bypass, falseReject)
		if bypass != 0 {
			t.Errorf("**التفافٌ على السقف** (واتساب=%s): %d", wa.flag, bypass)
		}
		if falseReject != 0 {
			t.Errorf("**ردٌّ كاذبٌ دون السقف** (واتساب=%s): %d", wa.flag, falseReject)
		}
	}
}

func TestD4_StressReleaseAndIdempotency(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	h.Setting("customers.require_whatsapp", "false")
	h.Setting("orders.max_open_per_customer", "1")
	item := h.NewItem(1000)

	notReleased, badRetry, duplicated, keyBypass := 0, 0, 0, 0
	for i := 0; i < 100; i++ {
		cust := f.NewUserWith("customer")

		// ── الإعادةُ بالمفتاح نفسِه ───────────────────────────────
		key := uniq("d4-sk")
		a := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
		b := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
		if b.Code == 409 || b.Code >= 400 {
			badRetry++
		}
		ida, _ := a.JSON()["id"].(string)
		idb, _ := b.JSON()["id"].(string)
		if ida == "" || ida != idb {
			duplicated++
		}
		// ── ومفتاحٌ متمايزٌ يُردّ بالسقف ──────────────────────────
		if c := h.POSTKey("/api/v1/orders", cust.Token, uniq("d4-sd"),
			orderBody(item, 1)).Code; c != 409 {
			keyBypass++
		}
		// ── والإغلاقُ يردّ الخانة ─────────────────────────────────
		if i < 50 {
			if cancel := h.POST("/api/v1/orders/"+ida+"/cancel", cust.Token,
				map[string]any{"reason": "D4"}); cancel.Code >= 400 {
				t.Fatalf("الإلغاء: %s", cancel)
			}
			if c := h.POSTKey("/api/v1/orders", cust.Token, uniq("d4-sr"),
				orderBody(item, 1)).Code; c >= 400 {
				notReleased++
			}
		}
	}
	t.Logf("D4/تكرار: إعادةٌ×100 ردٌّ كاذبٌ=%d · ازدواجٌ=%d · "+
		"مفتاحٌ متمايزٌ التفّ=%d · خانةٌ لم تُردّ=%d",
		badRetry, duplicated, keyBypass, notReleased)
	if badRetry != 0 {
		t.Errorf("**إعادةٌ رُدَّت بالسقف**: %d", badRetry)
	}
	if duplicated != 0 {
		t.Errorf("**إعادةٌ أنشأت طلباً ثانياً**: %d", duplicated)
	}
	if keyBypass != 0 {
		t.Errorf("**مفتاحٌ متمايزٌ التفّ على السقف**: %d", keyBypass)
	}
	if notReleased != 0 {
		t.Errorf("**خانةٌ لم تُردّ بعد الإغلاق**: %d", notReleased)
	}
}
