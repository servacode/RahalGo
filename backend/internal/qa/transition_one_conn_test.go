// وحدةُ الانتقال تكفيها وصلةٌ واحدة — **`XG-48`.**
//
// # ما كان يقع
//
// **`transitionTx` تفتح معاملةً ثمّ تقرأ من المَسبَح**: نمطُ الإدارة
// (`platform.orders_mode`) · ومهلةُ تدارُك الزبون · ونسبةُ عمولة
// المتجر · ونسبةُ تعويض السائق.
//
// **فالنداءُ يمسك وصلةً ويطلب ثانيةً وهو ممسكٌ بالأولى.**
//
// # ولماذا عطبُ منتَجٍ لا بطءَ جهاز
//
// **سقفُ المَسبَح في الإنتاج عشرون** — **فعشرون انتقالاً متزامناً
// تمسك العشرين، ثمّ يطلب كلٌّ منها ثانية.** **ولا تُفكّ إلّا بانتهاء
// واحدةٍ لا تستطيع أن تنتهي.**
//
// # وهو غيرُ `XG-46`
//
// **`XG-46` أُغلقت على نطاقها المُثبَت**: وحدةُ الإنشاء (`CreateTx`).
// **وهذه وحدةُ الانتقال** — **بابٌ آخرُ وشيفرةٌ أخرى**، **وأصلحته
// الآلةُ نفسُها** (`s.on(tx)`) **لا هويّتُه.**
//
// # والفروعُ تُمشى كلُّها
//
// **والقراءةُ قد تكون في فرعٍ شرطيّ** — **فإعدادٌ افتراضيٌّ يُخفي
// موضعاً**: نمطُ الإدارة فرعان، والتسويةُ فرعٌ لا يمرّ به القبول.
package qa

import (
	"testing"
	"time"
)

// transitionOnOneConn **ينفّذ نداءً على مَسبَحٍ سقفُه واحد** ويردّ
// الردَّ أو يسقط بالجمود.
//
// **ولا مهلةٌ تُرفَع ولا نومٌ يُضاف** — **المقيسُ أن يعود الردّ.**
func transitionOnOneConn(t *testing.T, h *Harness, label string, call func() int) int {
	t.Helper()
	done := make(chan int, 1)
	start := time.Now()
	go func() { done <- call() }()

	select {
	case code := <-done:
		t.Logf("  %-34s ⇒ %d بعد %s", label, code,
			time.Since(start).Round(time.Millisecond))
		return code
	case <-time.After(8 * time.Second):
		st := h.Pool.Stat()
		t.Fatalf("**جمد %q على وصلةٍ واحدة** — **يمسك المعاملةَ ويطلب "+
			"وصلةً ثانية.** (محجوزٌ=%d خاملٌ=%d سقفٌ=%d) (`XG-48`)",
			label, st.AcquiredConns(), st.IdleConns(), st.MaxConns())
		return 0
	}
}

// dispatchedOrder طلبٌ في الطابور جاهزٌ لسائق.
func dispatchedOrder(t *testing.T, h *Harness, cust *User, cost int64) string {
	t.Helper()
	item := h.NewItem(cost)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("xg48"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("تجهيزُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET status = 'dispatching' WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	return oid
}

// ══════════════════════════════════════════════════════════════════════
// **١ · وفرعا نمط الإدارة كلاهما**
// ══════════════════════════════════════════════════════════════════════
//
// **و`MerchantsSelfManage` تُقرأ في كلّ انتقال** — **وفرعٌ واحدٌ لا
// يُثبت أنّ القراءةَ من المعاملة.**

func TestXG48_TransitionBothModesNeedOneConnection(t *testing.T) {
	for _, mode := range []string{"merchants", "platform"} {
		t.Run(mode, func(t *testing.T) {
			h := oneConnHarness(t, 1)
			f := h.Factory()
			treasury(t, h)
			h.Setting("platform.orders_mode", `"`+mode+`"`)
			h.Setting("drivers.cash_limit", "9000000")
			h.Setting("drivers.assignment_mode", `"queue"`)

			cust := h.Customer()
			oid := dispatchedOrder(t, h, cust, 1000)
			drv := f.Driver(OnShift())

			code := transitionOnOneConn(t, h, "انتزاعُ السائق ["+mode+"]", func() int {
				return h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil).Code
			})
			if code >= 400 {
				t.Errorf("**رُدَّ الانتزاعُ بـ%d على وصلةٍ واحدة** [%s]", code, mode)
			}
		})
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · ومهلةُ تدارُك الزبون** — فرعٌ لا يمرّ به الانتزاع
// ══════════════════════════════════════════════════════════════════════
//
// **و`cancelWindowSec` تُقرأ في إلغاء الزبون وحدَه** — **فإعدادٌ
// افتراضيٌّ يُخفي موضعاً حتّى يُمشى فرعُه.**

func TestXG48_CustomerCancelNeedsOneConnection(t *testing.T) {
	h := oneConnHarness(t, 1)
	treasury(t, h)
	h.Setting("orders.customer_cancel_window_sec", "600")

	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("xg48c"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("تجهيزُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	// **والمهلةُ تُقاس من القبول** — فيُقبَل الطلبُ أوّلاً.
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET status = 'accepted', accepted_at = now()
		WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}

	code := transitionOnOneConn(t, h, "إلغاءُ الزبون", func() int {
		return h.POST("/api/v1/orders/"+oid+"/cancel", cust.Token,
			map[string]any{"note": "بدا لي"}).Code
	})
	if code >= 400 {
		t.Errorf("**رُدَّ الإلغاءُ بـ%d على وصلةٍ واحدة**", code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · والتسويةُ** — عمولةُ المتجر وتعويضُ السائق
// ══════════════════════════════════════════════════════════════════════
//
// **وفرعُ التسوية لا يمرّ به القبولُ ولا الإلغاء** — **وفيه قراءتان
// كانتا من المَسبَح**: نسبةُ عمولة المتجر، ونسبةُ تعويض السائق.

func TestXG48_DeliverySettlementNeedsOneConnection(t *testing.T) {
	h := oneConnHarness(t, 1)
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.cash_limit", "9000000")
	h.Setting("drivers.assignment_mode", `"queue"`)

	cust := h.Customer()
	oid := dispatchedOrder(t, h, cust, 5000)
	drv := f.Driver(OnShift())
	if got := h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil); got.Code >= 400 {
		t.Fatalf("القبول: %s", got)
	}
	// **و`delivered` تأتي من `at_dropoff` لا من `on_the_way`** —
	// **ومن قاس على الحال الخطأ قرأ رفضاً ٤٠٩ وظنّ الفرعَ مقيساً.**
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		if _, err := h.Pool.Exec(ctxBG(),
			`UPDATE orders SET status = $2 WHERE id = $1::uuid`, oid, to); err != nil {
			t.Fatalf("تهيئةُ %s: %v", to, err)
		}
	}

	// **والتسليمُ يشترط إثباتاً** (`delivery_proof_required`) —
	// **ويُتخطّى بسببٍ مسمّى، وهو بابٌ قائمٌ في المنتَج.**
	if got := h.POST("/api/v1/driver/orders/"+oid+"/proof/skip", drv.Token,
		map[string]any{"reason": "لا شبكة"}); got.Code >= 400 {
		t.Fatalf("تخطّي الإثبات: %s", got)
	}

	var errText string
	code := transitionOnOneConn(t, h, "التسليمُ والتسوية", func() int {
		res := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": "delivered"})
		errText = res.Err()
		return res.Code
	})
	// **ورفضٌ يعني أنّ فرعَ التسوية لم يُمشَ** — **والفحصُ الذي لا
	// يبلغ فرعَه يمرّ دائماً.**
	if code >= 400 {
		t.Fatalf("**لم يقع التسليمُ فلا تُقاس تسويتُه**: %d %q", code, errText)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٤ · والتكرارُ مئةً** — **جولةٌ واحدةٌ لا تُثبت وصلةً**
// ══════════════════════════════════════════════════════════════════════

func TestXG48_TransitionUnderRepetitionOnOneConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("تكرارٌ — لا يُشغَّل في الوضع القصير")
	}
	h := oneConnHarness(t, 1)
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.cash_limit", "9000000")
	h.Setting("drivers.max_active_orders", "500")
	h.Setting("drivers.assignment_mode", `"queue"`)
	cust := h.Customer()
	drv := f.Driver(OnShift())

	const rounds = 100
	stalls, refused := 0, 0
	for i := 0; i < rounds; i++ {
		oid := dispatchedOrder(t, h, cust, 1000)
		done := make(chan int, 1)
		go func() {
			done <- h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil).Code
		}()
		select {
		case code := <-done:
			if code >= 400 {
				refused++
			}
		case <-time.After(8 * time.Second):
			stalls++
			st := h.Pool.Stat()
			t.Fatalf("**جمدت الجولةُ %d على وصلةٍ واحدة** — "+
				"(محجوزٌ=%d سقفٌ=%d) (`XG-48`)",
				i+1, st.AcquiredConns(), st.MaxConns())
		}
	}
	t.Logf("XG-48 ×%d على وصلةٍ واحدة — جمودٌ %d · مردودٌ %d",
		rounds, stalls, refused)
	if stalls > 0 || refused > 0 {
		t.Errorf("**جمودٌ %d · مردودٌ %d من %d.** (`XG-48`)", stalls, refused, rounds)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · وتعويضُ السائق عند التعذّر** — فرعٌ لا يمرّ به التسليمُ الناجح
// ══════════════════════════════════════════════════════════════════════
//
// **و`compensateDriverOnFail` تقرأ `drivers.failed_compensation_percent`**
// — **ولا تُنادى إلّا عند تعذّرٍ ذنبُه على الزبون أو المتجر.**
//
// **وعمولةُ المتجر ليست موضعاً**: `orderPct` تردّ مؤشّراً غيرَ عدمٍ
// دائماً (اللقطةُ أو التجاوز)، **فقراءةُ الإعداد في `MerchantCommission`
// لا تُبلَغ من هذا المسار أصلاً.**

func TestXG48_FailedDeliveryCompensationNeedsOneConnection(t *testing.T) {
	h := oneConnHarness(t, 1)
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.cash_limit", "9000000")
	h.Setting("drivers.assignment_mode", `"queue"`)
	h.Setting("drivers.failed_compensation_percent", "50")

	cust := h.Customer()
	oid := dispatchedOrder(t, h, cust, 5000)
	drv := f.Driver(OnShift())
	if got := h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil); got.Code >= 400 {
		t.Fatalf("القبول: %s", got)
	}
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		if _, err := h.Pool.Exec(ctxBG(),
			`UPDATE orders SET status = $2 WHERE id = $1::uuid`, oid, to); err != nil {
			t.Fatalf("تهيئةُ %s: %v", to, err)
		}
	}
	// **وأجرةُ التوصيل شرطُ التعويض** — **ومِسنَدُ الاختبار ينشئ
	// طلباً بأجرةٍ صفر**، فتخرج الدالّةُ قبل قراءة الإعداد
	// **ويمرّ الفحصُ وهو لا يبلغ موضعَه.**
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET delivery_fee = 3000 WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("أجرةُ التوصيل: %v", err)
	}

	var errText string
	code := transitionOnOneConn(t, h, "التعذّرُ والتعويض", func() int {
		res := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": "failed", "reason": "customer_absent",
				"note": "لا يردّ"})
		errText = res.Err()
		return res.Code
	})
	if code >= 400 {
		t.Fatalf("**لم يقع التعذّرُ فلا يُقاس تعويضُه**: %d %q", code, errText)
	}

	// **ولا يُصدَّق أنّ الفرعَ مُشي** — **يُقاس أثرُه.**
	var fault string
	var fee int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(fault,''), delivery_fee FROM orders WHERE id = $1::uuid`,
		oid).Scan(&fault, &fee)
	t.Logf("  ذنبُ التعذّر = %q · أجرةُ التوصيل = %d", fault, fee)
	// **وتعويضٌ بلا أجرةِ توصيلٍ لا يُحسَب** —
	// تخرج قبل قراءة الإعداد. **ففحصٌ بأجرةٍ صفرٍ لا يبلغ الموضع.**
	if fee <= 0 {
		t.Fatalf("**أجرةُ التوصيل صفرٌ** — **وفرعُ التعويض لا يُبلَغ**، فلا يُقاس شيء")
	}
	if fault != "customer" && fault != "merchant" {
		t.Errorf("**الذنبُ %q — وفرعُ التعويض لا يُبلَغ إلّا بذنبِ زبونٍ "+
			"أو متجر**، فلم يُقَس شيء", fault)
	}
}
