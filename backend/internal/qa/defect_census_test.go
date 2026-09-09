// جردُ العيوب — **دورةُ ٤٨، وقياسٌ لا تصنيفٌ من ذاكرة.**
//
// # لماذا
//
// **السجلُّ يقول `EXPECTED_FAIL` عن ستّة**، **وحالُه مشتقٌّ من قاعدة**:
// «له اختبارٌ ولا نصَّ إصلاحٍ» — **لا من قياسٍ لسلوكٍ اليوم.**
//
// **وفحصٌ مربوطٌ بعيبٍ قد يقيس شيئاً آخر**: `D12` مربوطٌ بفحص استهداف
// الرموز، و`D14` بفحص وسيطٍ في `HTTP` — **وكلاهما يمرّ ولا يقول شيئاً
// عن عيبه.**
//
// **فهذه الفحوصُ تقيس ما لم يُقَس** — **ولا تُصلح شيئاً.**
package qa

import (
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **`D7` — أيمنع السقفُ نقداً يفوقه؟**
// ══════════════════════════════════════════════════════════════════════
//
// **العيبُ الأصليّ**: «السقفُ يقيس المحصَّل لا المكشوف» — **الشرطُ
// `held >= limit` لا يجمع نقدَ الطلب الداخل**، فسائقٌ محتجَزُه صفرٌ
// يقبل طلباً نقدُه ضعفُ السقف.
//
// **ويُقاس الرمزُ لا الرقمُ وحدَه**: **رفضٌ سببُه سقفُ الطلبات
// المفتوحة ليس رفضاً بسبب النقد** — **ومن قرأ `409` وحدَه ظنّ العيبَ
// مُصلَحاً.**

func TestCENSUS_D7_CashCeilingCountsIncomingOrder(t *testing.T) {
	h := New(t)
	f := h.Factory()
	h.Setting("drivers.cash_limit", "100000")

	drv := f.Driver(OnShift())
	admin := h.NewUser("admin")
	item := h.NewItem(90_000)

	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 3))
	if made.Code >= 400 {
		t.Fatalf("تجهيزُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)

	var due int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT cash_due FROM orders WHERE id = $1::uuid`, oid).Scan(&due); err != nil {
		t.Fatalf("قراءةُ النقد: %v", err)
	}
	if due <= 100_000 {
		t.Fatalf("**نقدُ الطلب %d لا يفوق السقف** — ولا يُقاس العيبُ على طلبٍ دونه", due)
	}

	held, err := f.HeldOf(drv.ID)
	if err != nil {
		t.Fatalf("قراءةُ المحتجَز: %v", err)
	}
	// **والإسنادُ اليدويُّ لا يقع على `pending`** — **يشترط
	// `preparing` أو `dispatching`** (`transitions.go`). **ومن قاس
	// على `pending` قرأ رفضاً سببُه الحالُ لا النقد**، **فظنّ
	// السقفَ يعمل.**
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET status = 'dispatching' WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("تهيئةُ الحال: %v", err)
	}
	got := h.POST("/api/v1/admin/orders/"+oid+"/assign", admin.Token,
		map[string]any{"driver_id": drv.ID})

	t.Logf("D7 — محتجَزٌ=%d · سقفٌ=100000 · نقدُ الطلب=%d ⇒ %d %s",
		held, due, got.Code, got.Err())
	switch {
	case got.Code < 400:
		t.Logf("D7 CURRENT REPRODUCTION = YES — **قُبل طلبٌ نقدُه %d "+
			"لسائقٍ محتجَزُه %d والسقفُ 100000**", due, held)
	default:
		t.Logf("D7 CURRENT REPRODUCTION = NO — **رُدَّ بـ%d** · الرمزُ %q",
			got.Code, got.Err())
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D14` — أتُقبَل مصافحةُ البثّ لموقوف؟**
// ══════════════════════════════════════════════════════════════════════
//
// **العيبُ الأصليّ**: «`handleWS` لا يفحص `ActiveStatus`».
//
// **والفحصُ المربوطُ به يقيس وسيطَ `HTTP`** (`middleware.go:40`) —
// **لا قناةَ البثّ.** **فمرورُه لا يقول شيئاً عن العيب.**
//
// **ويُقاس البابُ نفسُه**: `GET /api/v1/ws` بتوكن موقوف.

func TestCENSUS_D14_SuspendedCannotOpenSocket(t *testing.T) {
	h := New(t)
	u := h.Customer()

	// **والحالُ سليمٌ أوّلاً** — **ومقارنةٌ بلا أساسٍ لا تقول شيئاً.**
	before := h.GET("/api/v1/ws?token="+u.Token, "")
	t.Logf("قبل الإيقاف: مصافحةُ البثّ ⇒ %d", before.Code)

	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE users SET status = 'suspended' WHERE id = $1::uuid`, u.ID); err != nil {
		t.Fatalf("الإيقاف: %v", err)
	}

	// **والوسيطُ يردّ الموقوفَ في `HTTP`** — يُقاس ليُفرَّق البابان.
	rest := h.GET("/api/v1/my/orders", u.Token)
	after := h.GET("/api/v1/ws?token="+u.Token, "")
	t.Logf("بعد الإيقاف: REST ⇒ %d · مصافحةُ البثّ ⇒ %d %s",
		rest.Code, after.Code, after.Err())

	switch {
	case after.Code == before.Code && rest.Code >= 400:
		t.Logf("D14 CURRENT REPRODUCTION = YES — **`REST` يردّ الموقوفَ " +
			"والبثُّ يقبله**")
	case after.Code >= 400:
		t.Logf("D14 CURRENT REPRODUCTION = NO — **البثُّ ردّه بـ%d**", after.Code)
	default:
		t.Logf("D14 = غيرُ حاسم — REST=%d · WS=%d", rest.Code, after.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D6` و`D8` و`D9` — الطلبُ الخاصُّ وحرّاسُ الإنشاء**
// ══════════════════════════════════════════════════════════════════════
//
// **ثلاثةُ عيوبٍ في مسارٍ واحد** — **وتُقاس معاً لأنّ المسارَ واحد**:
//
//	`D6` — لا ينادي `cashBlocked` (المدينُ يفتح خاصّةً)
//	`D8` — لا يطلب توثيقَ واتساب
//	`D9` — بلا حدثِ `''→pending` في `order_events`

func TestCENSUS_D6_D9_CustomOrderCreationGuards(t *testing.T) {
	h := New(t)
	cust := h.Customer()

	// ── `D9` ── حدثُ الإنشاء ──────────────────────────────────────
	res := h.POST("/api/v1/orders/custom", cust.Token, map[string]any{
		"request": "قياسُ الجرد", "address_text": "الرقة",
		"lat": 35.9506, "lng": 39.0094,
	})
	if res.Code != 201 {
		t.Fatalf("إنشاءُ الخاصّ: %s", res)
	}
	oid, _ := res.JSON()["id"].(string)

	var events int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM order_events WHERE order_id = $1::uuid`, oid).
		Scan(&events); err != nil {
		t.Fatalf("قراءةُ الأحداث: %v", err)
	}
	t.Logf("D9 — أحداثُ الطلب الخاصّ بعد الإنشاء = %d", events)
	if events == 0 {
		t.Logf("D9 CURRENT REPRODUCTION = YES — **لا حدثَ إنشاءٍ في السجلّ**")
	} else {
		t.Logf("D9 CURRENT REPRODUCTION = NO — **كُتب %d حدثاً**", events)
	}

	// ── `D8` ── توثيقُ واتساب ─────────────────────────────────────
	//
	// **ويُقاس بالمقارنة**: يُفعَّل الشرطُ ثمّ يُنشأ العاديُّ والخاصّ.
	h.Setting("customers.require_whatsapp", "true")
	other := h.Customer()
	// **والمِسنَدُ قد يوثّق واتسابَ من يصنعه** — **فيُنزَع التوثيقُ
	// صراحةً**، وإلّا مرّ العاديُّ ولم يُقَس فرقٌ.
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE users SET whatsapp_verified_at = NULL WHERE id = $1::uuid`,
		other.ID); err != nil {
		t.Fatalf("نزعُ التوثيق: %v", err)
	}
	item := h.NewItem(1000)
	normal := h.POSTKey("/api/v1/orders", other.Token, uniq("k"), orderBody(item, 1))
	custom := h.POST("/api/v1/orders/custom", other.Token, map[string]any{
		"request": "قياسُ واتساب", "address_text": "الرقة",
		"lat": 35.9506, "lng": 39.0094,
	})
	t.Logf("D8 — واتساب مطلوب: العاديُّ ⇒ %d %s · الخاصُّ ⇒ %d %s",
		normal.Code, normal.Err(), custom.Code, custom.Err())
	if normal.Code >= 400 && custom.Code == 201 {
		t.Logf("D8 CURRENT REPRODUCTION = YES — **العاديُّ يُمنع والخاصُّ يمرّ**")
	} else {
		t.Logf("D8 CURRENT REPRODUCTION = NO/غيرُ حاسم")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D1` — فكُّ الإسناد وحدثُه**
// ══════════════════════════════════════════════════════════════════════
//
// **العيبُ الأصليّ**: «فكُّ الإسناد بلا حدثٍ في `order_events`».

func TestCENSUS_D1_ReleaseWritesEvent(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.assignment_mode", `"queue"`)
	h.Setting("drivers.cash_limit", "9000000")

	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET status = 'dispatching' WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	drv := f.Driver(OnShift())
	if got := h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil); got.Code >= 400 {
		t.Fatalf("القبول: %s", got)
	}

	var before int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM order_events WHERE order_id = $1::uuid`, oid).Scan(&before)

	rel := h.POST("/api/v1/driver/orders/"+oid+"/release", drv.Token, nil)
	if rel.Code >= 400 {
		t.Fatalf("**فكُّ الإسناد لم يقع فلا يُقاس ما بعده**: %s", rel)
	}

	var after int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM order_events WHERE order_id = $1::uuid`, oid).Scan(&after)
	t.Logf("D1 — أحداثٌ قبل الفكّ=%d · بعده=%d", before, after)
	if after == before {
		t.Logf("D1 CURRENT REPRODUCTION = YES — **فُكّ الإسنادُ ولا حدثَ له**")
	} else {
		t.Logf("D1 CURRENT REPRODUCTION = NO — **كُتب %d حدثاً**", after-before)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D4` — سقفُ المفتوح داخلَ بوّابة واتساب**
// ══════════════════════════════════════════════════════════════════════
//
// **الشرطُ متداخلٌ في الشيفرة** (`service.go`): `checkOpenLimit` تقع
// **داخلَ** `if RequireWhatsApp(...)`. **فمن أطفأ توثيقَ واتساب أطفأ
// سقفَ المفتوح معه** — **وحارسان في شرطٍ واحدٍ يسقطان معاً.**
//
// **والخاصُّ يناديها بلا شرط** (`custom.go:94,153) — **فالقاعدةُ نفسُها
// تُطبَّق في بابٍ وتُترَك في آخر.**

func TestCENSUS_D4_OpenLimitNestedInWhatsAppGate(t *testing.T) {
	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	h.Setting("orders.max_open_per_customer", "2")

	cust := h.Customer()
	item := h.NewItem(1000)

	codes := []int{}
	for i := 0; i < 5; i++ {
		res := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
		codes = append(codes, res.Code)
	}
	open := h.CountOrders(cust.ID)
	t.Logf("D4 — واتساب مطفأ · السقفُ 2 · خمسُ محاولات ⇒ %v · طلباتٌ=%d",
		codes, open)

	blocked := 0
	for _, c := range codes {
		if c >= 400 {
			blocked++
		}
	}
	if blocked == 0 {
		t.Logf("D4 CURRENT REPRODUCTION = YES — **خمسةٌ مرّت والسقفُ 2** — " +
			"**السقفُ معطَّلٌ بإطفاء واتساب**")
	} else {
		t.Logf("D4 CURRENT REPRODUCTION = NO — **رُدَّ %d منها**", blocked)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D6` — الطلبُ الخاصُّ لا ينادي `cashBlocked`**
// ══════════════════════════════════════════════════════════════════════
//
// **ومن مُنع من النقد لإخفاقاتٍ بخطئه يفتح خاصّةً نقديّةً بلا حارس.**

func TestCENSUS_D6_CustomOrderSkipsCashBan(t *testing.T) {
	h := New(t)
	f := h.Factory()
	h.Setting("customers.cash_ban_days", "30")
	h.Setting("customers.cash_ban_failures", "1")

	cust := h.Customer()
	item := h.NewItem(1000)

	// **ويُصنَع إخفاقٌ بخطأ الزبون** — حدثٌ في السجلّ لا حالٌ في الطلب.
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET status = 'failed', fault = 'customer' WHERE id = $1::uuid`,
		oid); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id)
		VALUES ($1::uuid, 'assigned', 'failed', $2::uuid)`, oid, cust.ID); err != nil {
		t.Fatalf("حدثُ الإخفاق: %v", err)
	}
	_ = f

	// **والعاديُّ النقديُّ يُمنع** — أساسُ المقارنة.
	normal := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	custom := h.POST("/api/v1/orders/custom", cust.Token, map[string]any{
		"request": "قياسُ حظر النقد", "address_text": "الرقة",
		"lat": 35.9506, "lng": 39.0094, "payment_method": "cash",
	})
	t.Logf("D6 — محظورُ النقد: العاديُّ ⇒ %d %s · الخاصُّ ⇒ %d %s",
		normal.Code, normal.Err(), custom.Code, custom.Err())

	switch {
	case normal.Code >= 400 && custom.Code == 201:
		t.Logf("D6 CURRENT REPRODUCTION = YES — **العاديُّ مُنع والخاصُّ مرّ**")
	case normal.Code >= 400 && custom.Code >= 400:
		t.Logf("D6 CURRENT REPRODUCTION = NO — **البابان يمنعان**")
	default:
		t.Logf("D6 = غيرُ حاسم — **العاديُّ لم يُمنع أصلاً** (%d): "+
			"**ولا يُقاس فرقٌ بلا أساس**", normal.Code)
	}
}
