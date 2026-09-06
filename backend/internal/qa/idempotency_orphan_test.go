package qa

import (
	"context"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **`C-06` · `R8` — مطالبةٌ يتيمةٌ تحبس صاحبَها**
// ══════════════════════════════════════════════════════════════════════
//
// # العقد
//
//	ORPHANED IDEMPOTENCY CLAIM MUST NOT BLOCK THE OWNER INDEFINITELY
//
// # آلةُ الحال كما هي (مقيسة)
//
// **صفٌّ واحدٌ بمفتاح `(user_id, endpoint, key)`** فيه `done` و
// `status_code` و`response` و`created_at`.
//
//  1. **الحجز**: `INSERT … ON CONFLICT DO NOTHING` — **حُجز إن أُدخل.**
//  2. **لم يُحجَز** ⇒ إن كان `done` أُعيد الردُّ المحفوظ، **وإلّا
//     `409 in_progress`.**
//  3. **ردٌّ ≥ 400** ⇒ **يُحذَف الصفّ** فيُعاد الطلبُ بحرّيّة.
//  4. **ردٌّ ناجح** ⇒ `done = true` ويُحفَظ الجسم.
//
// # وأين تنكسر
//
// **بين ١ و٤ فجوة**: **موتُ المنفّذ يترك `done=false` إلى الأبد** —
// **ولا مسارَ يُطلقه إلّا التقليمُ بعد أربعٍ وعشرين ساعة.**
// **فصاحبُ المفتاح يقرأ `409` يوماً كاملاً** على عمليّةٍ لم تقع.
//
// # وكيف يُحاكى الموتُ حتميّاً
//
// **قتلُ عمليّةٍ حقيقيّةً غيرُ عمليٍّ في حزمةِ اختبار** — **والحدُّ
// المقصودُ هو حالُ القاعدة الدائمة لا كيفيّةُ بلوغها.** فتُكتب حالُ
// اليتم مباشرةً: **مطالبةٌ محجوزةٌ غيرُ منتهية** — **وهي بعينها ما
// يتركه الموت.** (البند ٨.)

// orphanClaim يكتب مطالبةً يتيمةً كما يتركها موتُ المنفّذ.
func orphanClaim(t *testing.T, h *Harness, uid, endpoint, key string, ageMinutes int) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO idempotency_keys (user_id, endpoint, key, done, created_at)
		VALUES ($1::uuid, $2, $3, false, now() - make_interval(mins => $4))`,
		uid, endpoint, key, ageMinutes); err != nil {
		t.Fatalf("كتابةُ المطالبة اليتيمة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(), `
			DELETE FROM idempotency_keys
			 WHERE user_id = $1::uuid AND endpoint = $2 AND key = $3`,
			uid, endpoint, key)
	})
}

func claimRow(t *testing.T, h *Harness, uid, endpoint, key string) (done bool, found bool) {
	t.Helper()
	err := h.Pool.QueryRow(ctxBG(), `
		SELECT done FROM idempotency_keys
		 WHERE user_id = $1::uuid AND endpoint = $2 AND key = $3`,
		uid, endpoint, key).Scan(&done)
	if err != nil {
		return false, false
	}
	return done, true
}

// ══════════════════════════════════════════════════════════════════════
// **ب · موتٌ قبل تثبيت العمل ⇒ الإعادةُ تستردّ وتنفّذ مرّةً واحدة**
// ══════════════════════════════════════════════════════════════════════
// **وهذا يوثّق العطبَ القائم** — **ينجح ما دام قائماً ويسقط يومَ
// يُصلَح**، فيُقرأ سقوطُه أمراً بتحديث السجلّ. (وهو مذهبُ `TestFAIL_*`
// في هذه الحزمة.)
func TestFAIL_C06_OrphanBeforeCommitBlocksOwner(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)

	const key = "c06-orphan-1"
	const endpoint = "POST /api/v1/orders"
	orphanClaim(t, h, cust.ID, endpoint, key, 30)

	before := countRows(t, h,
		`SELECT count(*) FROM orders WHERE customer_id = $1::uuid`, cust.ID)
	got := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	after := countRows(t, h,
		`SELECT count(*) FROM orders WHERE customer_id = $1::uuid`, cust.ID)
	done, found := claimRow(t, h, cust.ID, endpoint, key)

	t.Logf("مطالبةٌ يتيمةٌ عمرُها ٣٠ دقيقة ⇒ الردُّ %d · طلباتٌ %d ← %d · "+
		"الصفُّ موجودٌ=%v منتهٍ=%v", got.Code, before, after, found, done)

	switch {
	case got.Code == 409 && after == before:
		t.Logf("C-06 CONFIRMED — **مطالبةٌ يتيمةٌ تحبس صاحبَها**: " +
			"`409 in_progress` ولا تنفيذ، **ولا مسارَ يُطلقها إلّا " +
			"التقليمُ بعد ٢٤ ساعة.** والعقدُ المخروق: ORPHANED " +
			"IDEMPOTENCY CLAIM MUST NOT BLOCK THE OWNER INDEFINITELY")
	case got.Code < 400 && after == before+1 && done:
		t.Errorf("**`C-06` أُصلحت** — اليتيمةُ صارت تُستردّ وتُنفَّذ " +
			"مرّةً واحدة. **يُحدَّث السجلُّ (`R8`) والمصفوفة.**")
	default:
		t.Errorf("**حالٌ غيرُ موصوفة**: الردُّ %d · طلباتٌ %d ← %d · منتهٍ %v",
			got.Code, before, after, done)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ج · العملُ ثُبِّت والمنفّذُ مات قبل حفظ النتيجة ⇒ لا تنفيذَ ثانياً**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذه أخطرُ من سابقتها**: **استردادٌ أعمى هنا يُنشئ طلباً ثانياً
// بمالٍ ثانٍ.** **ولا يجوز إصلاحُ الحال (ب) بخلق تكرارٍ في (ج).**
func TestIDEM_CommittedBeforeResultDoesNotDuplicate(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)

	const key = "c06-orphan-2"
	const endpoint = "POST /api/v1/orders"

	// **عملٌ وقع فعلاً** — طلبٌ قائمٌ بهذا المفتاح.
	first := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	if first.Code >= 400 {
		t.Fatalf("الطلبُ الأوّل: %s", first)
	}
	afterFirst := countRows(t, h,
		`SELECT count(*) FROM orders WHERE customer_id = $1::uuid`, cust.ID)

	// **ثمّ يُمحى أثرُ النتيجة ويُشاخ الصفّ** — **كما لو مات المنفّذُ
	// بعد التثبيت وقبل الحفظ.**
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE idempotency_keys
		   SET done = false, status_code = 0, response = NULL,
		       created_at = now() - interval '30 minutes'
		 WHERE user_id = $1::uuid AND endpoint = $2 AND key = $3`,
		cust.ID, endpoint, key); err != nil {
		t.Fatalf("محاكاةُ الموت بعد التثبيت: %v", err)
	}

	retry := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	after := countRows(t, h,
		`SELECT count(*) FROM orders WHERE customer_id = $1::uuid`, cust.ID)
	t.Logf("عملٌ ثُبِّت ثمّ ماتت النتيجة ⇒ الإعادةُ %d · طلباتٌ %d ← %d",
		retry.Code, afterFirst, after)

	if after != afterFirst {
		t.Errorf("**التنفيذُ تكرّر**: طلباتٌ %d ← %d — "+
			"**واستردادٌ أعمى يخلق مالاً ثانياً.**", afterFirst, after)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **و · مستردّان متزامنان على مطالبةٍ يتيمة ⇒ تنفيذٌ واحد**
// ══════════════════════════════════════════════════════════════════════
func TestFAIL_C06_TwoReclaimersExecuteNothing(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)

	const key = "c06-orphan-3"
	const endpoint = "POST /api/v1/orders"
	orphanClaim(t, h, cust.ID, endpoint, key, 30)

	before := countRows(t, h,
		`SELECT count(*) FROM orders WHERE customer_id = $1::uuid`, cust.ID)
	body := orderBody(item, 1)
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "مستردٌّ-أ", Do: func(ctx context.Context) any {
			return h.POSTKey("/api/v1/orders", cust.Token, key, body)
		}},
		Actor{Name: "مستردٌّ-ب", Do: func(ctx context.Context) any {
			return h.POSTKey("/api/v1/orders", cust.Token, key, body)
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	after := countRows(t, h,
		`SELECT count(*) FROM orders WHERE customer_id = $1::uuid`, cust.ID)
	t.Logf("مستردّان متزامنان: طلباتٌ %d ← %d · تداخلٌ مقيسٌ %d — %s",
		before, after, r.Probe.Max(), r)

	// **ولا تكرارَ اليوم** — **لأنّ لا استردادَ أصلاً.**
	if after > before+1 {
		t.Errorf("**تنفيذان من مطالبةٍ واحدة**: %d ← %d — "+
			"**وهذا أخطرُ من الحبس**", before, after)
	}
	if after == before {
		t.Logf("C-06 CONFIRMED — **مستردّان ولا تنفيذ**: اليتيمةُ تحبس " +
			"الاثنين. **والسلامةُ هنا أثرُ الشلل لا أثرُ حراسة.**")
	} else {
		t.Errorf("**`C-06` أُصلحت** — استُردّت ونُفّذت مرّةً واحدة. " +
			"**يُحدَّث السجلُّ.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **د · ردٌّ ضاع ⇒ إعادةُ النتيجة نفسِها بلا تنفيذٍ ثانٍ**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذه تعمل اليوم** — تُحرَس لئلّا ينكسر ما يعمل.
func TestIDEM_LostResponseReplays(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)

	const key = "c06-replay-1"
	first := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	if first.Code >= 400 {
		t.Fatalf("الطلبُ الأوّل: %s", first)
	}
	mid := countRows(t, h,
		`SELECT count(*) FROM orders WHERE customer_id = $1::uuid`, cust.ID)

	again := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	after := countRows(t, h,
		`SELECT count(*) FROM orders WHERE customer_id = $1::uuid`, cust.ID)
	t.Logf("ردٌّ ضاع: الأوّلُ %d · الإعادةُ %d · طلباتٌ %d ← %d",
		first.Code, again.Code, mid, after)

	if after != mid {
		t.Errorf("**الإعادةُ نفّذت ثانيةً**: %d ← %d", mid, after)
	}
	if again.Code != first.Code {
		t.Errorf("الردُّ المُعاد %d والأصلُ %d — **وليس هو نفسَه**",
			again.Code, first.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ز · التقليمُ لا يمسّ مطالبةً حيّة**
// ══════════════════════════════════════════════════════════════════════
func TestIDEM_CleanupSparesLiveClaim(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)

	const live = "c06-live-1"
	const endpoint = "POST /api/v1/orders"
	orphanClaim(t, h, cust.ID, endpoint, live, 0) // **حجزٌ جديدٌ حيّ**

	// **نداءٌ آخرُ بمفتاحٍ مختلف** — يُجري التقليمَ في `claimIdempotency`.
	other := h.POSTKey("/api/v1/orders", cust.Token, "c06-live-other",
		orderBody(item, 1))
	if other.Code >= 400 {
		t.Fatalf("النداءُ الآخر: %s", other)
	}
	_, found := claimRow(t, h, cust.ID, endpoint, live)
	t.Logf("بعد تقليمٍ جرى مع نداءٍ آخر: المطالبةُ الحيّةُ موجودةٌ=%v", found)
	if !found {
		t.Error("**التقليمُ محا مطالبةً حيّة** — **وعمليّةٌ تعمل الآن " +
			"فقدت حمايتَها فيُنفَّذ عملُها مرّتين.**")
	}
}
