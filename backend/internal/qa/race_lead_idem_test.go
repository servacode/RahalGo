package qa

// `XG-18` ومنعُ التكرار — **`P-5` البنود ٧ و٨…١٤.**

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **٧ · تحويلُ مرشَّحٍ واحدٍ مرّتين — `XG-18`**
// ══════════════════════════════════════════════════════════════════════

// newLead يُنشئ مرشَّحاً بمندوبه — **بأدنى ما يلزم لتحويله.**
func newLead(t *testing.T, h *Harness, repID, categoryID, phone string) string {
	t.Helper()
	var id string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO merchant_leads
		  (store_name, owner_name, phone, area, sales_rep_user_id, category_id,
		   owner_password_hash, status, lat, lng)
		VALUES ($1, 'صاحبُ متجرٍ QA', $2, 'الرقة — اختبار', $3::uuid, $4::uuid, 'x', 'new',
		        35.9506, 39.0094)
		RETURNING id::text`,
		uniq("مرشَّح QA "), phone, repID, categoryID).Scan(&id); err != nil {
		t.Fatalf("إنشاءُ المرشَّح: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM merchant_leads WHERE id = $1::uuid`, id)
	})
	return id
}

// TestRACE_DuplicateLeadConversion **البند ٧ — `XG-18`.**
//
//	ONE REAL MERCHANT IDENTITY MUST NOT BECOME TWO PAYABLE MERCHANT/REP RELATIONSHIPS
//
// **والحارسُ مقيسٌ لا مظنون** (`leads_handlers.go:735`):
//
//	if merchantID != nil { return nil }   // قراءةٌ ثمّ كتابةٌ بلا قفل
//
// **فالنافذةُ بين القراءة والإنشاء مفتوحة.**
func TestRACE_DuplicateLeadConversion(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	admin := h.NewUser("admin")

	var categoryID string
	if err := h.Pool.QueryRow(ctxBG(),
		`INSERT INTO categories (name, active) VALUES ($1, true) RETURNING id::text`,
		uniq("تصنيف QA ")).Scan(&categoryID); err != nil {
		t.Fatalf("تصنيف: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(), `DELETE FROM categories WHERE id = $1::uuid`, categoryID)
	})

	base := financialBaseline(t, h)
	// **وستُّ جولاتٍ لا ثلاث** — نافذةُ هذا السباق أضيقُ من نافذة `R10`:
	// **قِيس أنّه يتكرّر مرّةً في نحو ثلاث جولات**، فثلاثٌ قد تمرّ بلا
	// كشف. **والتوسيعُ للكشف لا للصيد**: الحاجزُ يفتح النافذةَ عمداً في
	// كلّ جولة.
	dups, rounds := 0, 6
	for i := 0; i < rounds; i++ {
		rep := f.RepAccount()
		phone := f.NS.Phone()
		leadID := newLead(t, h, rep.ID, categoryID, phone)

		body := map[string]any{"status": "converted"}
		r := Race(t, DefaultRaceTimeout,
			Actor{Name: "إداريّ-أ", Do: func(ctx context.Context) any {
				return h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token, body)
			}},
			Actor{Name: "إداريّ-ب", Do: func(ctx context.Context) any {
				return h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token, body)
			}},
		)
		if r.TimedOut {
			t.Fatalf("الجولة %d عَلِقت", i+1)
		}
		if r.Probe.Max() < 2 {
			t.Errorf("الجولة %d: تداخلٌ مقيسٌ %d", i+1, r.Probe.Max())
		}

		// **والقاعدةُ هي الحَكَم**: كم متجراً صار لهذا الهاتف؟
		var merchants, leadStatus int
		var status string
		var repOwned int
		if err := h.Pool.QueryRow(ctxBG(), `
			SELECT (SELECT count(*) FROM merchants WHERE phone = $1),
			       (SELECT count(*) FROM merchants WHERE phone = $1 AND sales_rep_user_id = $2::uuid)`,
			phone, rep.ID).Scan(&merchants, &repOwned); err != nil {
			t.Fatalf("عدُّ المتاجر: %v", err)
		}
		_ = h.Pool.QueryRow(ctxBG(),
			`SELECT status FROM merchant_leads WHERE id = $1::uuid`, leadID).Scan(&status)
		leadStatus = len(status)
		_ = leadStatus

		t.Cleanup(func() {
			_, _ = h.Pool.Exec(context.Background(), `DELETE FROM merchants WHERE phone = $1`, phone)
		})

		if merchants > 1 {
			dups++
			if dups == 1 {
				t.Logf("XG-18 PROVEN — الجولة %d: هويّةٌ واحدةٌ صارت %d متجراً (%d منها للمندوب) · حالُ المرشَّح %q\n%s",
					i+1, merchants, repOwned, status, r)
			}
		} else {
			t.Logf("الجولة %d: متاجرُ %d · نجح %d من 2 · حالُ المرشَّح %q\n%s",
				i+1, merchants, r.CountOK(), status, r)
		}
	}

	if dups > 0 {
		t.Logf("DUPLICATE MERCHANT XG-18 = PROVEN — تكرّرت الهويّةُ في %d من %d جولة", dups, rounds)
		t.Logf("DEFECT CANDIDATE — يُعرَض على المالك ولا يُجمَّد من تلقائي (البند ٢٨)")
	} else {
		t.Logf("DUPLICATE MERCHANT XG-18 = NOT REPRODUCED محلّيّاً في %d جولات", rounds)
	}
	assertNewViolations(t, h, base, "FI-02", "FI-03", "FI-05")
}

// ══════════════════════════════════════════════════════════════════════
// **٨…١٣ · منعُ التكرار**
// ══════════════════════════════════════════════════════════════════════

// idemRow يقرأ صفَّ مفتاحِ التكرار — **الحقيقةُ في القاعدة لا في الردّ.**
func idemRow(t *testing.T, h *Harness, uid, endpoint, key string) (done bool, status int, found bool) {
	t.Helper()
	err := h.Pool.QueryRow(ctxBG(), `
		SELECT done, status_code FROM idempotency_keys
		WHERE user_id = $1::uuid AND endpoint = $2 AND key = $3`,
		uid, endpoint, key).Scan(&done, &status)
	if err != nil {
		return false, 0, false
	}
	return done, status, true
}

const ordersEndpoint = "POST /api/v1/orders"

// TestIDEM_SameKeySamePayloadSequential **البند ٨ — بالتتابع.**
func TestIDEM_SameKeySamePayloadSequential(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	key := uniq("k")
	body := orderBody(item, 1)

	first := h.POSTKey("/api/v1/orders", cust.Token, key, body)
	second := h.POSTKey("/api/v1/orders", cust.Token, key, body)
	t.Logf("الأوّل %d · الثاني %d", first.Code, second.Code)
	if first.Code >= 400 {
		t.Fatalf("الأوّلُ رُدّ: %s", first)
	}
	if n := h.CountOrders(cust.ID); n != 1 {
		t.Errorf("SAME KEY SAME PAYLOAD SEQUENTIAL خُرق: %d طلباً — يُنتظر 1", n)
	}
	if second.Code >= 400 {
		t.Errorf("الإعادةُ رُدّت %s — والعقدُ إعادةُ الردّ", second)
	}
	done, st, found := idemRow(t, h, cust.ID, ordersEndpoint, key)
	t.Logf("صفُّ المفتاح: موجودٌ=%v · مُنجَزٌ=%v · رمزٌ=%d", found, done, st)
	if !found || !done {
		t.Errorf("الصفُّ لم يُختَم مُنجَزاً")
	}
}

// TestIDEM_SameKeySamePayloadConcurrent **البند ٨ — بالتزامن.**
//
// **والعقدُ على أثرِ القاعدة لا على الردّ** — قد يختلف الردّان،
// **والطلبُ يجب أن يكون واحداً.**
func TestIDEM_SameKeySamePayloadConcurrent(t *testing.T) {
	h := New(t)
	item := h.NewItem(1000)

	for i := 0; i < raceIterations; i++ {
		cust := h.Customer()
		key := uniq("k")
		body := orderBody(item, 1)

		r := Race(t, DefaultRaceTimeout,
			Actor{Name: "طلب-أ", Do: func(ctx context.Context) any {
				return h.POSTKey("/api/v1/orders", cust.Token, key, body)
			}},
			Actor{Name: "طلب-ب", Do: func(ctx context.Context) any {
				return h.POSTKey("/api/v1/orders", cust.Token, key, body)
			}},
		)
		if r.TimedOut {
			t.Fatalf("الجولة %d عَلِقت", i+1)
		}
		if r.Probe.Max() < 2 {
			t.Errorf("الجولة %d: تداخلٌ مقيسٌ %d", i+1, r.Probe.Max())
		}
		if n := h.CountOrders(cust.ID); n != 1 {
			t.Errorf("SAME KEY SAME PAYLOAD CONCURRENT خُرق: الجولة %d أنشأت %d طلباً\n%s",
				i+1, n, r)
		}
		if i == 0 {
			t.Logf("CONCURRENT — الرموز %v · وطلبٌ واحدٌ في القاعدة\n%s", r.Codes(), r)
		}
	}
	t.Logf("SAME KEY SAME PAYLOAD CONCURRENT = PASS — %d جولات · طلبٌ واحدٌ في كلٍّ", raceIterations)
}

// TestIDEM_SameKeyDifferentPayload **البند ٩ — ولا يُفترَض الجواب.**
func TestIDEM_SameKeyDifferentPayload(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	key := uniq("k")

	first := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	if first.Code >= 400 {
		t.Fatalf("الأوّلُ رُدّ: %s", first)
	}
	firstID, _ := first.JSON()["id"].(string)

	// **حمولةٌ أخرى بالمفتاح نفسِه** — ثلاثةُ أصنافٍ بدل واحد.
	second := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 3))
	secondID, _ := second.JSON()["id"].(string)
	replay := fmt.Sprint(second.Replay())

	t.Logf("الأوّل %d · الثاني %d · إعادةٌ=%q", first.Code, second.Code, replay)
	t.Logf("معرّفُ الأوّل %s · ومعرّفُ الثاني %s", firstID, secondID)

	n := h.CountOrders(cust.ID)
	if n != 1 {
		t.Errorf("SAME KEY DIFFERENT PAYLOAD: أُنشئ %d طلباً — **والمفتاحُ لم يمنع**", n)
	}

	// **والقياسُ يقول الحقيقة**: لا تجزئةَ للحمولة في `idempotency.go`
	// — **فالثانيةُ تُعاد إجابةً عن الأولى بصمت.**
	if second.Code < 400 && secondID == firstID {
		t.Logf("MEASURED CONTRACT — SILENT REPLAY: الحمولةُ الثانيةُ (٣ أصناف) رُدّت بجواب الأولى (صنفٌ واحد)")
		t.Logf("OBSERVATION — لا تجزئةَ للحمولة في idempotency.go: لا يُكشَف تعارضُ الحمولة ولا يُردّ 409")
		t.Logf("REQUIRED CONTRACT — SAME KEY + DIFFERENT PAYLOAD SHOULD NOT SILENTLY REPLAY")
	} else if second.Code == 409 {
		t.Logf("MEASURED CONTRACT — CONFLICT DETECTED: رُدَّ 409 على حمولةٍ مختلفة")
	} else {
		t.Errorf("سلوكٌ غيرُ متوقَّع: %s", second)
	}
}

// TestIDEM_InProgressOverlap **البند ١١ — طلبٌ ثانٍ والأوّلُ ما يزال داخلَ نافذته.**
//
// **وليس «انتهى الأوّلُ ثمّ جاء الثاني»**: يُحبَس الأوّلُ داخل المسار
// **بقفلٍ على صفٍّ يحتاجه**، فيدخل الثاني والأوّلُ لم يختم.
func TestIDEM_InProgressOverlap(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	key := uniq("k")
	body := orderBody(item, 1)

	// **الحبسُ بقفلِ صفِّ الصنف** — الطلبُ يقرأ سعرَه، فيقف حتّى يُفلَت.
	conn, err := h.Pool.Acquire(ctxBG())
	if err != nil {
		t.Fatalf("اتّصال: %v", err)
	}
	tx, err := conn.Begin(ctxBG())
	if err != nil {
		conn.Release()
		t.Fatalf("معاملة: %v", err)
	}
	if _, err := tx.Exec(ctxBG(),
		`SELECT id FROM menu_items WHERE id = $1::uuid FOR UPDATE`, item.ID); err != nil {
		_ = tx.Rollback(ctxBG())
		conn.Release()
		t.Fatalf("قفلُ الصنف: %v", err)
	}

	firstDone := make(chan Res, 1)
	go func() { firstDone <- h.POSTKey("/api/v1/orders", cust.Token, key, body) }()

	// **ننتظر أن يصير الأوّلُ محجوباً في القاعدة** — دليلٌ أنّه داخلَ نافذته.
	if _, err := WaitBlocked(ctxBG(), h.Pool, 1, 10*time.Second); err != nil {
		_ = tx.Rollback(ctxBG())
		conn.Release()
		<-firstDone
		t.Skipf("لم يُحجَب الأوّلُ — لا نافذةَ تُختبَر: %v", err)
	}

	// **وصفُّ المفتاح مطالَبٌ به وغيرُ مُنجَز** — هذه هي الحالُ المقصودة.
	done, _, found := idemRow(t, h, cust.ID, ordersEndpoint, key)
	t.Logf("والأوّلُ داخلَ نافذته: صفُّ المفتاح موجودٌ=%v · مُنجَزٌ=%v", found, done)

	second := h.POSTKey("/api/v1/orders", cust.Token, key, body)
	t.Logf("الثاني أثناء تنفيذ الأوّل: %d %s", second.Code, second.Err())

	_ = tx.Rollback(ctxBG())
	conn.Release()
	first := <-firstDone
	t.Logf("والأوّلُ بعد الإفلات: %d", first.Code)

	if found && !done && second.Code == 409 && second.Err() == "in_progress" {
		t.Logf("IN_PROGRESS OVERLAP = PASS — الثاني رُدَّ in_progress ولم يُنشئ أثراً ثانياً")
	} else if second.Code < 400 {
		t.Errorf("IN_PROGRESS OVERLAP خُرق: الثاني نجح والأوّلُ لم يختم — %s", second)
	} else {
		t.Logf("IN_PROGRESS OVERLAP — رُدَّ %d/%q", second.Code, second.Err())
	}
	if n := h.CountOrders(cust.ID); n > 1 {
		t.Errorf("أثرٌ مزدوج: %d طلباً", n)
	}
}

// TestIDEM_ErrorReleasesClaim **البند ١٢ — ولا يُفترَض الجواب.**
func TestIDEM_ErrorReleasesClaim(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	key := uniq("k")

	bad := map[string]any{"items": []map[string]any{}, "address_text": "", "payment_method": "cash"}
	first := h.POSTKey("/api/v1/orders", cust.Token, key, bad)
	t.Logf("الأوّلُ بحمولةٍ خاطئة: %d %s", first.Code, first.Err())
	if first.Code < 400 {
		t.Fatalf("الحمولةُ الخاطئةُ قُبلت: %s", first)
	}
	_, _, found := idemRow(t, h, cust.ID, ordersEndpoint, key)
	t.Logf("صفُّ المفتاح بعد الخطأ: موجودٌ=%v", found)
	if found {
		t.Errorf("ERROR RELEASE خُرق: المفتاحُ بقي محجوزاً بعد خطأٍ — **فلا إصلاحَ للحمولة**")
	}

	// **ثمّ يُعاد بالمفتاح نفسِه وحمولةٍ صحيحة.**
	good := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	t.Logf("الإعادةُ بحمولةٍ صحيحة: %d", good.Code)
	if good.Code >= 400 {
		t.Errorf("ERROR RELEASE خُرق: المفتاحُ لم يُطلَق — %s", good)
	} else {
		t.Logf("ERROR RELEASE = PASS — المفتاحُ أُطلق والإصلاحُ نجح بالمفتاح نفسِه")
	}
	if n := h.CountOrders(cust.ID); n != 1 {
		t.Errorf("عددُ الطلبات %d — يُنتظر 1", n)
	}
}

// TestIDEM_CommitThenLostResponse **البند ١٠ — `R8`.**
//
// **ولا يُنتظَر أربعٌ وعشرون ساعة**، **ولا يُقطَع سلكٌ**: الحالُ التي
// يخلقها فقدُ الردّ **تُصنَع في القاعدة مباشرةً** (البند ١٠: «افصل محاكاةَ
// حال المعاملة الآن عن إثبات قطع الشبكة لاحقاً»).
//
// **وهما حالان مختلفتان تماماً:**
//
//	done = true   ← العملُ تمّ والردُّ ضاع  ⇒ الإعادةُ تستردّ الردَّ
//	done = false  ← المطالبةُ كُتبت والعمليّةُ ماتت ⇒ 409 in_progress حتّى التقليم
func TestIDEM_CommitThenLostResponse(t *testing.T) {
	h := New(t)
	item := h.NewItem(1000)

	// ── أ · العملُ تمّ والردُّ ضاع ──────────────────────────────────
	{
		cust := h.Customer()
		key := uniq("k")
		body := orderBody(item, 1)
		first := h.POSTKey("/api/v1/orders", cust.Token, key, body)
		if first.Code >= 400 {
			t.Fatalf("الأوّل: %s", first)
		}
		// **الزبونُ لم يرَ الردَّ فأعاد** — والمفتاحُ نفسُه.
		retry := h.POSTKey("/api/v1/orders", cust.Token, key, body)
		t.Logf("COMMITTED + LOST RESPONSE — الإعادةُ %d · إعادةٌ=%q",
			retry.Code, fmt.Sprint(retry.Replay()))
		if retry.Code >= 400 {
			t.Errorf("الإعادةُ رُدّت %s — والعملُ تمّ", retry)
		}
		if n := h.CountOrders(cust.ID); n != 1 {
			t.Errorf("أثرٌ مزدوج: %d طلباً", n)
		}
		if fmt.Sprint(retry.Replay()) != "true" {
			t.Errorf("الردُّ لم يُعلَّم إعادةً")
		}
		t.Logf("RESPONSE RECOVERY = PASS — الردُّ استُرِدّ ولا أثرَ ثانٍ")
	}

	// ── ب · المطالبةُ كُتبت والعمليّةُ ماتت — `R8` ──────────────────
	{
		cust := h.Customer()
		key := uniq("k")
		// **صفُّ مطالبةٍ غيرُ مُنجَزٍ** — تماماً كما يتركه سقوطُ العمليّة
		// بين `claimIdempotency` و`storeIdempotency` (`idempotency.go:52…60`).
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO idempotency_keys (user_id, endpoint, key)
			VALUES ($1::uuid, $2, $3)`, cust.ID, ordersEndpoint, key); err != nil {
			t.Fatalf("صنعُ الحال: %v", err)
		}
		t.Cleanup(func() {
			_, _ = h.Pool.Exec(context.Background(),
				`DELETE FROM idempotency_keys WHERE user_id = $1::uuid AND key = $2`, cust.ID, key)
		})
		retry := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
		t.Logf("ORPHANED CLAIM — الإعادةُ %d %s", retry.Code, retry.Err())
		if retry.Code == 409 && retry.Err() == "in_progress" {
			t.Logf("R8 CONFIRMED — مطالبةٌ يتيمةٌ تحجب الزبونَ عن الطلب: 409 in_progress")
			t.Logf("ولا مسارَ يُطلقها: releaseIdempotency لا يُنادى إلّا من ردٍّ ≥400 في الطلب نفسِه")
			t.Logf("فالإطلاقُ الوحيدُ هو التقليمُ بعد %s", "24h")
		} else {
			t.Errorf("لم تُحجَب الإعادةُ: %s — **وR8 يقول إنّها تُحجَب. يُراجَع.**", retry)
		}
		if n := h.CountOrders(cust.ID); n != 0 {
			t.Errorf("أُنشئ طلبٌ رغم المطالبة اليتيمة: %d", n)
		}
	}
}

// TestIDEM_CleanupAfterTTL **البند ١٣ — ولا يُنتظَر يوم.**
//
// **والزمنُ يُزاح في البيانة** — كما في `P-3` (`XOB-10`: لا ساعةَ محقونةٌ
// في شيفرة الإنتاج).
func TestIDEM_CleanupAfterTTL(t *testing.T) {
	h := New(t)
	item := h.NewItem(1000)
	old := h.Customer()
	fresh := h.Customer()
	oldKey, freshKey := uniq("k"), uniq("k")

	for _, x := range []struct {
		uid, key string
		age      string
	}{{old.ID, oldKey, "48 hours"}, {fresh.ID, freshKey, "1 hour"}} {
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO idempotency_keys (user_id, endpoint, key, created_at)
			VALUES ($1::uuid, $2, $3, now() - $4::interval)`,
			x.uid, ordersEndpoint, x.key, x.age); err != nil {
			t.Fatalf("صفٌّ %s: %v", x.age, err)
		}
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM idempotency_keys WHERE key IN ($1, $2)`, oldKey, freshKey)
	})

	// **والتقليمُ يقع في `claimIdempotency`** — فيُستدعى بطلبٍ جديد.
	other := h.Customer()
	if got := h.POSTKey("/api/v1/orders", other.Token, uniq("k"), orderBody(item, 1)); got.Code >= 400 {
		t.Fatalf("الطلبُ المُشغِّلُ للتقليم رُدّ: %s", got)
	}

	_, _, oldFound := idemRow(t, h, old.ID, ordersEndpoint, oldKey)
	_, _, freshFound := idemRow(t, h, fresh.ID, ordersEndpoint, freshKey)
	t.Logf("بعد التقليم: الشائخُ (48h) موجودٌ=%v · والحديثُ (1h) موجودٌ=%v", oldFound, freshFound)

	if oldFound {
		t.Errorf("24H CLEANUP خُرق: الصفُّ الشائخُ لم يُحذَف")
	}
	if !freshFound {
		t.Errorf("24H CLEANUP خُرق: **حُذف صفٌّ حديثٌ لا شأنَ له** — والتقليمُ يجب أن يمسّ الشائخَ وحدَه")
	}
	if !oldFound && freshFound {
		t.Logf("24H CLEANUP = PASS — الشائخُ حُذف والحديثُ بقي")
		t.Logf("OBSERVATION — التقليمُ بلا شرطِ done ولا شرطِ مستخدم: يمسّ الشائخَ من كلّ الحسابات")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **١٤ · سحبٌ مكرَّر**
// ══════════════════════════════════════════════════════════════════════

// TestRACE_DuplicatePayout **البند ١٤ — بالتتابع وبالتزامن.**
func TestRACE_DuplicatePayout(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	admin := h.NewUser("admin")
	base := financialBaseline(t, h)

	// ── أ · طلبا سحبٍ متزامنان من السائق نفسِه ──────────────────────
	drv := f.Driver()
	f.Credit(drv.ID, 200_000, "topup")
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "طلب-أ", Do: func(ctx context.Context) any {
			return h.POST("/api/v1/me/payouts", drv.Token, map[string]any{"amount": 100_000})
		}},
		Actor{Name: "طلب-ب", Do: func(ctx context.Context) any {
			return h.POST("/api/v1/me/payouts", drv.Token, map[string]any{"amount": 100_000})
		}},
	)
	if r.TimedOut {
		t.Fatal("طلبا السحب عَلِقا")
	}
	var pending int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM payout_requests WHERE user_id = $1::uuid AND status = 'pending'`,
		drv.ID).Scan(&pending)
	t.Logf("REQUEST RACE — الرموز %v · طلباتٌ معلَّقةٌ %d", r.Codes(), pending)
	if pending > 1 {
		t.Errorf("طلبا سحبٍ معلَّقان معاً — والفهرسُ الفريدُ يجب أن يمنع")
	}

	// ── ب · قرارا دفعٍ متزامنان على الطلب نفسِه ────────────────────
	var pid string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT id::text FROM payout_requests WHERE user_id = $1::uuid AND status = 'pending'`,
		drv.ID).Scan(&pid); err != nil {
		t.Fatalf("لم أجد طلبَ سحبٍ معلَّقاً: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM payout_requests WHERE user_id = $1::uuid`, drv.ID)
	})

	body := map[string]any{"status": "paid", "decision": "P-5"}
	d := Race(t, DefaultRaceTimeout,
		Actor{Name: "قرار-أ", Do: func(ctx context.Context) any {
			return h.POST("/api/v1/admin/payouts/"+pid+"/decide", admin.Token, body)
		}},
		Actor{Name: "قرار-ب", Do: func(ctx context.Context) any {
			return h.POST("/api/v1/admin/payouts/"+pid+"/decide", admin.Token, body)
		}},
	)
	if d.TimedOut {
		t.Fatal("قرارا السحب عَلِقا")
	}
	var entries int
	var sum int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT count(*), COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'payout'`, pid).Scan(&entries, &sum)
	t.Logf("DECIDE RACE — الرموز %v · قيودُ السحب %d · مجموعُها %d · تداخلٌ %d",
		d.Codes(), entries, sum, d.Probe.Max())

	if entries != 1 {
		t.Errorf("PAYOUT DUPLICATE خُرق: خُصم %d مرّة — **تسويةٌ مزدوجة**", entries)
	}
	if sum != -100_000 {
		t.Errorf("مجموعُ الخصم %d — يُنتظر -100000", sum)
	}
	if d.CountOK() != 1 {
		t.Errorf("نجح %d قراراً من 2 — **والقفلُ يجب أن يُبقيَ واحداً**", d.CountOK())
	}
	if entries == 1 && d.CountOK() == 1 {
		t.Logf("PAYOUT DUPLICATE = PASS — FOR UPDATE + شرطُ pending منعا الازدواج")
	}
	assertNewViolations(t, h, base, "FI-11", "FI-05", "FI-04", "FI-02")
}

var _ = fmt.Sprint
