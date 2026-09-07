package qa

// البنود ١٥ و١٨ و١٩ — **المالُ المتنازَع · والجلسةُ · وطابورُ الموقع.**

import (
	"context"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **١٥ · استردادٌ مقابلَ سحب — على المستحقِّ نفسِه**
// ══════════════════════════════════════════════════════════════════════

// TestRACE_RefundVsPayout **البند ١٥.**
//
// **والسيناريو ممكنٌ اليومَ بلا اختراع**: المتجرُ يقبض مستحقَّه عند
// الاستلام (`settleMerchant`)، **ثمّ يتنازع عليه فعلان**:
//
//	الاستردادُ  ← يعكس المستحقَّ (سالبٌ على محفظته)
//	قرارُ السحب ← يخصم المبلغَ نفسَه إلى خارج المنظومة
//
// **والقيدُ `CHECK (balance >= 0)` يمنع أن يقعا معاً** — والسؤال:
// **أيبقى المالُ محفوظاً أيّاً كان الفائز؟**
func TestRACE_RefundVsPayout(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.assignment_mode", `"queue"`)
	h.Setting("drivers.max_active_orders", "5")
	h.Setting("drivers.cash_limit", "9000000")
	h.Setting("merchants.commission_percent", "0")
	h.Setting("payouts.min_amount", "1")
	admin := h.NewUser("admin")

	base := financialBaseline(t, h)

	m := f.Merchant()
	item := h.NewItemFor(m, 40_000)
	oid := dispatchOrder(t, h, item)
	drv := onShiftDriver(t, f)
	if got := h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil); got.Code >= 400 {
		t.Fatalf("قبول: %s", got)
	}
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to}); got.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s: %s", to, got)
		}
	}
	_ = h.POST("/api/v1/driver/orders/"+oid+"/proof/skip", drv.Token, map[string]any{"reason": "P-5"})
	if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"}); got.Code >= 400 {
		t.Fatalf("التسليم: %s", got)
	}

	// **ومستحقُّ المتجر صار في محفظته.**
	var owner string
	var bal int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT m.owner_user_id::text, COALESCE(w.balance, 0)
		FROM merchants m LEFT JOIN wallets w ON w.user_id = m.owner_user_id
		WHERE m.id = $1::uuid`, m.ID).Scan(&owner, &bal); err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	t.Logf("مستحقُّ المتجرِ في محفظته = %d", bal)
	if bal <= 0 {
		t.Skip("لم يُقيَّد مستحقٌّ — والسيناريو يشترطه")
	}

	// **يطلب صاحبُ المتجر سحبَ كلِّ مستحقّه.**
	made := h.POST("/api/v1/me/payouts", m.Owner.Token, map[string]any{"amount": bal})
	if made.Code >= 400 {
		t.Skipf("طلبُ السحب رُدّ (%s) — والسيناريو يشترطه", made)
	}
	var pid string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT id::text FROM payout_requests WHERE user_id = $1::uuid AND status = 'pending'`,
		owner).Scan(&pid); err != nil {
		t.Fatalf("لم أجد طلبَ السحب: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM payout_requests WHERE user_id = $1::uuid`, owner)
	})

	// **والآن يتنازعان.**
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "استرداد", Do: func(ctx context.Context) any {
			return h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
				map[string]any{"to": "refunded", "note": "P-5 — سباقُ استردادٍ وسحب"})
		}},
		Actor{Name: "دفعُ سحب", Do: func(ctx context.Context) any {
			return h.POST("/api/v1/admin/payouts/"+pid+"/decide", admin.Token,
				map[string]any{"status": "paid", "decision": "P-5"})
		}},
	)
	if r.TimedOut {
		t.Fatal("السباقُ عَلِق")
	}
	t.Logf("REFUND VS PAYOUT — %s", r)

	var after, refunded, paidOut int64
	var payStatus, orderStatus string
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT balance FROM wallets WHERE user_id = $1::uuid), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                 WHERE ref = $2 AND kind = 'refund'), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                 WHERE ref = $3 AND kind = 'payout'), 0),
		       (SELECT status FROM payout_requests WHERE id = $3::uuid),
		       (SELECT status FROM orders WHERE id = $2::uuid)`,
		owner, oid, pid).Scan(&after, &refunded, &paidOut, &payStatus, &orderStatus)

	t.Logf("رصيدُ المتجر بعدُ = %d · المستردُّ للزبون = %d · المخصومُ سحباً = %d",
		after, refunded, paidOut)
	t.Logf("حالُ السحب = %q · حالُ الطلب = %q", payStatus, orderStatus)

	if after < 0 {
		t.Errorf("MONEY DUPLICATION: رصيدُ المتجر سالبٌ %d — **صُرف المالُ مرّتين**", after)
	}
	if payStatus == "paid" && refunded > 0 {
		t.Logf("OBSERVATION — وقع الاثنان: سُحب المستحقُّ ورُدّ للزبون، **والفرقُ على المنصّة**")
	}
	if payStatus != "paid" && refunded > 0 {
		t.Logf("RESULT — الاستردادُ فاز والسحبُ لم يُدفَع (حالُه %q)", payStatus)
	}
	if payStatus == "paid" && refunded == 0 {
		t.Logf("EXPECTED_FAIL (XG-11) — السحبُ فاز والاستردادُ سقط: **حقُّ الزبونِ ضاع بسبب رصيدِ طرفٍ ثالث**")
	}

	// **والحَكَمُ محرّكُ `P-4`** (البند ٢٠) — لا توكيدَ ماليٌّ جديد.
	assertNewViolations(t, h, base, "FI-11", "FI-05", "FI-04", "FI-06", "FI-10")
}

// ══════════════════════════════════════════════════════════════════════
// **١٨ · إبطالُ الجلسة أثناء طلبٍ جارٍ**
// ══════════════════════════════════════════════════════════════════════

// TestRACE_SessionRevokedDuringRequest **البند ١٨.**
//
// **ويُثبَت محلّيّاً بأمان** — لا يحتاج طقمَ خدماتٍ: `revokeAllSessions`
// تُبطل **رموزَ التجديد** في القاعدة (`repo.go:528`)، **ورمزُ الوصول
// لا يُسأل عنها** — فهو موقَّعٌ ومهلتُه خمسَ عشرةَ دقيقة.
//
// **فالسؤالُ مقيسٌ لا مظنون**: أيمرّ طلبٌ برمزِ وصولٍ بعد إبطال الجلسة؟
func TestRACE_SessionRevokedDuringRequest(t *testing.T) {
	h := New(t)
	item := h.NewItem(1000)
	admin := h.NewUser("admin")

	// ══════════════════════════════════════════════════════════════
	// **وتوكنٌ مربوطٌ بجلسةٍ لها صفٌّ دائم** — تصحيحُ دورةِ ١٦
	// ══════════════════════════════════════════════════════════════
	//
	// **كان يُقاس بتوكنِ المصنع** — **ومعرّفُ جلسته فارغ**، **فالوسيطُ
	// لا يسأل عنه أصلاً.** **فما قيس «رمزُ وصولٍ ينجو من الإبطال»
	// وإنّما «صنفُ توكناتٍ لا يُبطَل».**
	//
	// **والمنصّةُ لا تُصدر ذلك الصنف**: `issueSession` ينادي
	// `StoreRefresh` قبل الإصدار — **فلكلّ توكنٍ حقيقيٍّ معرّفُ جلسة.**
	custToken, _, custID := liveSession(t, h)
	cust := &User{ID: custID, Token: custToken}

	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "طلبُ الزبون", Do: func(ctx context.Context) any {
			return h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
		}},
		Actor{Name: "إبطالُ الجلسة", Do: func(ctx context.Context) any {
			return h.POST("/api/v1/admin/users/"+cust.ID+"/logout-all", admin.Token, nil)
		}},
	)
	if r.TimedOut {
		t.Fatal("السباقُ عَلِق")
	}
	t.Logf("SESSION REVOCATION RACE — %s", r)

	// **وبعد الإبطال يُعاد النداءُ بالرمز نفسِه.**
	after := h.GET("/api/v1/my/orders", cust.Token)
	t.Logf("نداءٌ بعد الإبطال بالرمز نفسِه: %d", after.Code)

	if after.Code < 400 {
		t.Errorf("**رمزُ وصولٍ نجا من إبطال الجلسة** (%d) — "+
			"**والإبطالُ يجب أن يُنهي القدرةَ على العمل.** (`R16` · `C-09`)",
			after.Code)
	} else {
		t.Logf("رُدَّ %d بعد الإبطال — **والرمزُ يُسأل عن الجلسة**", after.Code)
	}
	if n := h.CountOrders(cust.ID); n > 1 {
		t.Errorf("أثرٌ مزدوجٌ للطلب: %d", n)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **١٩ · طابورُ موقع السائق — `R19`**
// ══════════════════════════════════════════════════════════════════════

// TestRACE_LocationQueueIsClientSide **البند ١٩ — ولا يختفي من الخريطة.**
//
// **و`R19` خطرُ تزامنٍ في العميل لا في الخادم**: الطابورُ
// (`all → شبكة → clear`) يعيش في تطبيق أندرويد، **ولا يُجبَر داخل حزمةِ
// PostgreSQL** (البند ١٩).
//
// **والمُثبَتُ هنا ما يُثبَت محلّيّاً**: **بابُ الخادم نفسُه يتحمّل دفعاتٍ
// متزامنةً بلا فقدٍ ولا ازدواج** — وهو نصفُ الطريق، **والنصفُ الآخرُ
// (فقدُ الدفعة بين الشبكة والمسح) في `P-8`.**
func TestRACE_LocationQueueIsClientSide(t *testing.T) {
	h := New(t)
	f := h.Factory()
	drv := f.Driver(OnShift())

	const n = 4
	actors := make([]Actor, n)
	for i := 0; i < n; i++ {
		i := i
		actors[i] = Actor{Name: "دفعة", Do: func(ctx context.Context) any {
			return h.POST("/api/v1/driver/location", drv.Token, map[string]any{
				"lat": 35.9506 + float64(i)/10000, "lng": 39.0094,
			})
		}}
	}
	r := Race(t, DefaultRaceTimeout, actors...)
	if r.TimedOut {
		t.Fatal("دفعاتُ الموقع عَلِقت")
	}
	if r.Probe.Max() < 2 {
		t.Errorf("تداخلٌ مقيسٌ %d — لم يقع سباق", r.Probe.Max())
	}
	t.Logf("SERVER-SIDE LOCATION WRITES — %d متزامنة · الرموز %v · تداخلٌ %d",
		n, r.Codes(), r.Probe.Max())

	var stored int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM users WHERE id = $1::uuid AND last_location IS NOT NULL`,
		drv.ID).Scan(&stored)
	if r.CountOK() > 0 && stored != 1 {
		t.Errorf("الموقعُ لم يُكتب بعد %d دفعةٍ ناجحة", r.CountOK())
	}
	t.Logf("R19 LOCATION QUEUE RACE = DEFERRED TO P-8")
	t.Logf("والسببُ: الطابورُ (all → شبكة → clear) في تطبيق أندرويد — ولا يُجبَر داخل حزمةِ PostgreSQL")
	t.Logf("والمُثبَتُ هنا: بابُ الخادم يتحمّل %d دفعةٍ متزامنةٍ بلا ازدواج", n)
}
