package qa

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **`XG-33` · `R8` · `C-06` — عملٌ وعلامةُ تثبيتٍ في معاملةٍ واحدة**
// ══════════════════════════════════════════════════════════════════════
//
//	IDEMPOTENCY DURABLE COMMIT STATE MUST BE ATOMIC WITH BUSINESS COMMIT
//
// **وكلُّ فحصٍ هنا يسأل الدفترَ لا الشيفرة.**

const idemOrdersEndpoint = "POST /api/v1/orders"

// claimState حالُ مطالبةٍ كما تُقرأ للمراجعة.
type claimState struct {
	Found     bool
	Committed bool
	Owner     string
	Status    int
}

func claimStateOf(t *testing.T, h *Harness, uid, endpoint, key string) claimState {
	t.Helper()
	var st claimState
	var owner *string
	err := h.Pool.QueryRow(ctxBG(), `
		SELECT committed_at IS NOT NULL, owner_token::text, status_code
		  FROM idempotency_keys
		 WHERE user_id = $1::uuid AND endpoint = $2 AND key = $3`,
		uid, endpoint, key).Scan(&st.Committed, &owner, &st.Status)
	if err != nil {
		return st
	}
	st.Found = true
	if owner != nil {
		st.Owner = *owner
	}
	return st
}

func idemOrdersOf(t *testing.T, h *Harness, uid string) int {
	t.Helper()
	return countRows(t, h,
		`SELECT count(*) FROM orders WHERE customer_id = $1::uuid`, uid)
}

// staleClaim مطالبةٌ محجوزةٌ انتهت مهلتُها ولها مالكٌ — **قابلةٌ للاسترداد.**
func staleClaim(t *testing.T, h *Harness, uid, endpoint, key string) string {
	t.Helper()
	var owner string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO idempotency_keys
		       (user_id, endpoint, key, done, created_at, owner_token, lease_until)
		VALUES ($1::uuid, $2, $3, false, now() - interval '10 minutes',
		        gen_random_uuid(), now() - interval '5 minutes')
		RETURNING owner_token::text`, uid, endpoint, key).Scan(&owner); err != nil {
		t.Fatalf("مطالبةٌ شائخة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(), `
			DELETE FROM idempotency_keys
			 WHERE user_id = $1::uuid AND endpoint = $2 AND key = $3`, uid, endpoint, key)
	})
	return owner
}

// ══════════════════════════════════════════════════════════════════════
// **T1 · نداءان متزامنان بالمفتاح نفسِه ⇒ تنفيذٌ واحد**
// ══════════════════════════════════════════════════════════════════════
func TestIDEM_T1_ConcurrentDuplicateExecutesOnce(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)
	const key = "t1-concurrent"

	before := idemOrdersOf(t, h, cust.ID)
	body := orderBody(item, 1)
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "زبونٌ-أ", Do: func(ctx context.Context) any {
			return h.POSTKey("/api/v1/orders", cust.Token, key, body)
		}},
		Actor{Name: "زبونٌ-ب", Do: func(ctx context.Context) any {
			return h.POSTKey("/api/v1/orders", cust.Token, key, body)
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	after := idemOrdersOf(t, h, cust.ID)
	t.Logf("متزامنان بمفتاحٍ واحد: طلباتٌ %d ← %d · تداخلٌ مقيسٌ %d — %s",
		before, after, r.Probe.Max(), r)
	if after != before+1 {
		t.Errorf("**التنفيذُ لم يقع مرّةً واحدة**: %d ← %d", before, after)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T2 · مطالبةٌ يتيمةٌ قبل معاملة العمل ⇒ استردادٌ آمنٌ وتنفيذٌ واحد**
// ══════════════════════════════════════════════════════════════════════
//
// **وهي `C-06` بعينها** — كانت `409` إلى أن يُقلَّم بعد ٢٤ ساعة.
func TestIDEM_T2_OrphanBeforeTxIsReclaimed(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)
	const key = "t2-orphan"
	staleClaim(t, h, cust.ID, idemOrdersEndpoint, key)

	before := idemOrdersOf(t, h, cust.ID)
	got := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	after := idemOrdersOf(t, h, cust.ID)
	st := claimStateOf(t, h, cust.ID, idemOrdersEndpoint, key)
	t.Logf("يتيمةٌ شائخة ⇒ الردُّ %d · طلباتٌ %d ← %d · مثبَّتةٌ=%v",
		got.Code, before, after, st.Committed)

	if got.Code >= 400 {
		t.Errorf("**اليتيمةُ ما زالت تحبس** (%d) — `C-06`", got.Code)
	}
	if after != before+1 {
		t.Errorf("**التنفيذُ %d ← %d** والمتوقَّع واحد", before, after)
	}
	if !st.Committed {
		t.Error("**لم تُثبَّت العلامةُ بعد نجاح**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T3 · مالكٌ فقد مطالبتَه ⇒ صفرُ كتابات**
// ══════════════════════════════════════════════════════════════════════
//
// **والسياجُ هو الحارس** — **لا المهلة**: **من نام قبل أن يقفل الصفَّ
// ثمّ استيقظ يجد رمزَه لا يطابق.**
func TestIDEM_T3_StaleOwnerIsFenced(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)
	const key = "t3-fenced"

	// **مالكٌ قديمٌ برمزٍ قديم** — ثمّ يستردّها غيرُه.
	old := staleClaim(t, h, cust.ID, idemOrdersEndpoint, key)
	before := idemOrdersOf(t, h, cust.ID)
	if got := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1)); got.Code >= 400 {
		t.Fatalf("الاستردادُ الجديد: %s", got)
	}
	mid := idemOrdersOf(t, h, cust.ID)
	newOwner := claimStateOf(t, h, cust.ID, idemOrdersEndpoint, key)
	t.Logf("رمزٌ قديمٌ %s · رمزٌ جديدٌ %s · طلباتٌ %d ← %d · مثبَّتةٌ=%v",
		first8(old), first8(newOwner.Owner), before, mid, newOwner.Committed)

	if old == newOwner.Owner {
		t.Error("**الرمزُ لم يتبدّل بالاسترداد** — ولا سياجَ إذاً")
	}
	// **والمالكُ القديمُ يصحو** — يُحاكى بمحاولة كتابةٍ برمزه.
	tag, err := h.Pool.Exec(ctxBG(), `
		UPDATE idempotency_keys SET committed_at = now(), status_code = 999
		 WHERE user_id = $1::uuid AND endpoint = $2 AND key = $3
		   AND owner_token = $4::uuid AND committed_at IS NULL`,
		cust.ID, idemOrdersEndpoint, key, old)
	if err != nil {
		t.Fatalf("محاولةُ المالك القديم: %v", err)
	}
	after := idemOrdersOf(t, h, cust.ID)
	t.Logf("المالكُ القديمُ استيقظ: صفوفٌ مسّها %d · طلباتٌ %d",
		tag.RowsAffected(), after)
	if tag.RowsAffected() != 0 {
		t.Errorf("**المالكُ القديمُ كتب** (%d صفّاً) — **والسياجُ لا يعمل**",
			tag.RowsAffected())
	}
	if after != mid {
		t.Errorf("**تنفيذٌ ثانٍ**: %d ← %d", mid, after)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T4 · انتهاءُ المهلة ومعاملةُ العمل حيّة ⇒ لا سرقة**
// ══════════════════════════════════════════════════════════════════════
//
// **معاملةٌ تمسك الصفَّ بـ`FOR UPDATE`** ومهلتُها انتهت. **ومحاولةُ
// الاستردادِ تصطدم بالقفل فتنتظر**، ثمّ تقرأ الحقيقةَ التي ثبتت.
func TestIDEM_T4_ActiveClaimCannotBeStolen(t *testing.T) {
	h := New(t)
	const key = "t4-active"
	cust := h.Customer()
	staleClaim(t, h, cust.ID, idemOrdersEndpoint, key)

	ctx := ctxBG()
	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("معاملةٌ حيّة: %v", err)
	}
	var owner string
	if err := tx.QueryRow(ctx, `
		SELECT owner_token::text FROM idempotency_keys
		 WHERE user_id = $1::uuid AND endpoint = $2 AND key = $3 FOR UPDATE`,
		cust.ID, idemOrdersEndpoint, key).Scan(&owner); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("قفلُ الصفّ: %v", err)
	}

	// **مستردٌّ يحاول بينما القفلُ قائم.**
	// **ومحاولةُ السرقة بمهلة** — **حارسٌ يحبس صفّاً بلا حدٍّ يُعدي
	// الحزمةَ كلَّها**: وقعت مرّةً فسقط فحصٌ بعيدٌ بـ`500` بعد ثلاثين
	// ثانية. **ومهلةُ الطلب في الخادم ثلاثون، فخمسٌ هنا تكفي وتزيد.**
	steal, cancelSteal := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelSteal()

	blocked := make(chan int64, 1)
	go func() {
		tag, err := h.Pool.Exec(steal, `
			UPDATE idempotency_keys SET owner_token = gen_random_uuid()
			 WHERE user_id = $1::uuid AND endpoint = $2 AND key = $3
			   AND committed_at IS NULL AND owner_token IS NOT NULL
			   AND lease_until < now()`, cust.ID, idemOrdersEndpoint, key)
		if err != nil {
			blocked <- -1
			return
		}
		blocked <- tag.RowsAffected()
	}()

	select {
	case n := <-blocked:
		_ = tx.Rollback(ctx)
		t.Fatalf("**سُرقت المطالبةُ والمعاملةُ حيّة** — مسّ %d صفّاً", n)
	case <-time.After(400 * time.Millisecond):
		t.Log("المستردُّ محبوسٌ على القفل — **ولا سرقةَ من عاملٍ حيّ**")
	}

	// **ثمّ تُثبَّت العمليّةُ الحيّة** — فيجد المستردُّ حقيقتَها.
	if _, err := tx.Exec(ctx, `
		UPDATE idempotency_keys SET committed_at = now(), done = true, status_code = 201
		 WHERE user_id = $1::uuid AND endpoint = $2 AND key = $3`,
		cust.ID, idemOrdersEndpoint, key); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("تثبيتُ الحيّة: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("التثبيت: %v", err)
	}
	n := <-blocked
	t.Logf("بعد التثبيت: المستردُّ مسّ %d صفّاً", n)
	if n != 0 {
		t.Errorf("**المستردُّ سرق مطالبةً ثبتت** (%d)", n)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T5 · ارتدادُ العمل ⇒ صفرُ كتاباتٍ وإعادةٌ ممكنة**
// ══════════════════════════════════════════════════════════════════════
func TestIDEM_T5_BusinessRollbackLeavesNothing(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)
	const key = "t5-rollback"

	before := idemOrdersOf(t, h, cust.ID)
	fp := h.ArmAny("T5/order-insert", "orders", "INSERT")
	failed := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	fp.MustFire(t)
	mid := idemOrdersOf(t, h, cust.ID)
	st := claimStateOf(t, h, cust.ID, idemOrdersEndpoint, key)
	t.Logf("ارتدادٌ (%d): طلباتٌ %d ← %d · المطالبةُ موجودةٌ=%v",
		failed.Code, before, mid, st.Found)

	if mid != before {
		t.Errorf("**كتابةٌ بقيت بعد الارتداد**: %d ← %d", before, mid)
	}

	retry := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	after := idemOrdersOf(t, h, cust.ID)
	t.Logf("الإعادةُ (%d): طلباتٌ %d ← %d", retry.Code, mid, after)
	if retry.Code >= 400 {
		t.Errorf("**الإعادةُ محبوسة** (%d) — والعملُ لم يقع قطّ", retry.Code)
	}
	if after != mid+1 {
		t.Errorf("**الإعادةُ لم تنفّذ مرّةً واحدة**: %d ← %d", mid, after)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T6+T7 · ثبت العملُ ومات قبل الردّ ⇒ تُعاد النتيجةُ بلا تكرار**
// ══════════════════════════════════════════════════════════════════════
//
// **وهي الحالُ التي كانت تمنع أيَّ استرداد**: **الآن العلامةُ داخلَ
// المعاملة، فوجودُها يقين.**
func TestIDEM_T6_CommittedThenDeathReplaysWithoutDuplicate(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)
	const key = "t6-committed"

	first := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	if first.Code >= 400 {
		t.Fatalf("الطلبُ الأوّل: %s", first)
	}
	mid := idemOrdersOf(t, h, cust.ID)

	// **موتٌ بعد التثبيت وقبل الردّ**: المهلةُ تنتهي والعلامةُ باقية.
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE idempotency_keys SET lease_until = now() - interval '5 minutes'
		 WHERE user_id = $1::uuid AND endpoint = $2 AND key = $3`,
		cust.ID, idemOrdersEndpoint, key); err != nil {
		t.Fatalf("إشاخةُ المهلة: %v", err)
	}

	retry := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	after := idemOrdersOf(t, h, cust.ID)
	t.Logf("ثبت ثمّ مات: الأوّلُ %d · الإعادةُ %d · طلباتٌ %d ← %d",
		first.Code, retry.Code, mid, after)

	if after != mid {
		t.Errorf("**تنفيذٌ ثانٍ لعملٍ ثبت**: %d ← %d — "+
			"**ومهلةٌ انتهت لا تُبيح إعادةَ تنفيذ.**", mid, after)
	}
	if retry.Code != first.Code {
		t.Errorf("الردُّ المُعاد %d والأصلُ %d", retry.Code, first.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T8 · مستردّان متزامنان على يتيمة ⇒ تنفيذٌ واحد**
// ══════════════════════════════════════════════════════════════════════
func TestIDEM_T8_TwoReclaimersExecuteOnce(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)
	const key = "t8-two"
	staleClaim(t, h, cust.ID, idemOrdersEndpoint, key)

	before := idemOrdersOf(t, h, cust.ID)
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
	after := idemOrdersOf(t, h, cust.ID)
	t.Logf("مستردّان متزامنان: طلباتٌ %d ← %d · تداخلٌ مقيسٌ %d — %s",
		before, after, r.Probe.Max(), r)
	if after != before+1 {
		t.Errorf("**التنفيذُ ليس مرّةً واحدة**: %d ← %d", before, after)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T9 · مالكٌ قديمٌ لا يمحو مطالبةَ غيرِه**
// ══════════════════════════════════════════════════════════════════════
func TestIDEM_T9_StaleOwnerCannotDeleteNewerClaim(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	const key = "t9-delete"
	old := staleClaim(t, h, cust.ID, idemOrdersEndpoint, key)

	// **مالكٌ جديدٌ يستردّها.**
	var newOwner string
	if err := h.Pool.QueryRow(ctxBG(), `
		UPDATE idempotency_keys SET owner_token = gen_random_uuid(),
		       lease_until = now() + interval '60 seconds'
		 WHERE user_id = $1::uuid AND endpoint = $2 AND key = $3
		RETURNING owner_token::text`, cust.ID, idemOrdersEndpoint, key).Scan(&newOwner); err != nil {
		t.Fatalf("الاسترداد: %v", err)
	}

	// **والقديمُ يصل إلى تنظيف خطئه.**
	tag, err := h.Pool.Exec(ctxBG(), `
		DELETE FROM idempotency_keys
		 WHERE user_id = $1::uuid AND endpoint = $2 AND key = $3
		   AND committed_at IS NULL AND owner_token = $4::uuid`,
		cust.ID, idemOrdersEndpoint, key, old)
	if err != nil {
		t.Fatalf("حذفُ القديم: %v", err)
	}
	st := claimStateOf(t, h, cust.ID, idemOrdersEndpoint, key)
	t.Logf("القديمُ حذف %d صفّاً · المطالبةُ باقيةٌ=%v · مالكُها %s",
		tag.RowsAffected(), st.Found, first8(st.Owner))

	if tag.RowsAffected() != 0 {
		t.Errorf("**القديمُ محا مطالبةَ غيرِه** (%d)", tag.RowsAffected())
	}
	if !st.Found || st.Owner != newOwner {
		t.Errorf("**المطالبةُ تبدّلت**: باقيةٌ=%v · مالكٌ %q والمتوقَّع %q",
			st.Found, st.Owner, newOwner)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T10 · التقليمُ لا يمسّ مطالبةً حيّة**
// ══════════════════════════════════════════════════════════════════════
func TestIDEM_T10_CleanupSparesLiveClaim(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)
	const key = "t10-live"

	// **صفٌّ قديمُ الإنشاءِ ومهلتُه حيّة** — **عمليّةٌ طويلةٌ تعمل الآن.**
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO idempotency_keys
		       (user_id, endpoint, key, done, created_at, owner_token, lease_until)
		VALUES ($1::uuid, $2, $3, false, now() - interval '30 hours',
		        gen_random_uuid(), now() + interval '60 seconds')`,
		cust.ID, idemOrdersEndpoint, key); err != nil {
		t.Fatalf("مطالبةٌ حيّةٌ قديمة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(), `
			DELETE FROM idempotency_keys
			 WHERE user_id = $1::uuid AND endpoint = $2 AND key = $3`,
			cust.ID, idemOrdersEndpoint, key)
	})

	// **نداءٌ بمفتاحٍ آخرَ يُجري التقليم.**
	if got := h.POSTKey("/api/v1/orders", cust.Token, "t10-other",
		orderBody(item, 1)); got.Code >= 400 {
		t.Fatalf("النداءُ الآخر: %s", got)
	}
	st := claimStateOf(t, h, cust.ID, idemOrdersEndpoint, key)
	t.Logf("بعد تقليمٍ جرى: المطالبةُ الحيّةُ باقيةٌ=%v", st.Found)
	if !st.Found {
		t.Error("**التقليمُ محا مطالبةً حيّة** — **وعمليّةٌ تعمل فقدت " +
			"حمايتَها فيُنفَّذ عملُها مرّتين.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T11 · مفتاحٌ واحدٌ بجسمين — العقدُ كما هو**
// ══════════════════════════════════════════════════════════════════════
//
// **`XOB-11` خارجَ النطاق**: **المفتاحُ وحدَه يحكم ولا يُقارَن الجسم.**
// **ويُوثَّق لئلّا ينكسر صدفةً.**
func TestIDEM_T11_SameKeyDifferentPayloadContractUnchanged(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	one := h.NewItem(1000)
	two := h.NewItem(7000)
	const key = "t11-payload"

	first := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(one, 1))
	if first.Code >= 400 {
		t.Fatalf("الأوّل: %s", first)
	}
	mid := idemOrdersOf(t, h, cust.ID)
	second := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(two, 3))
	after := idemOrdersOf(t, h, cust.ID)

	t.Logf("مفتاحٌ واحدٌ بجسمين: الأوّلُ %d · الثاني %d · طلباتٌ %d ← %d",
		first.Code, second.Code, mid, after)
	if after != mid {
		t.Errorf("**جسمٌ مختلفٌ نفّذ عملاً ثانياً**: %d ← %d — "+
			"**والعقدُ الحاليّ: المفتاحُ وحدَه يحكم.**", mid, after)
	}
	t.Log("XOB-11 CONTRACT UNCHANGED — إعادةٌ صامتةٌ للنتيجة الأولى")
}

// ══════════════════════════════════════════════════════════════════════
// **حارسٌ بنيويّ: لا مسارَ محميٌّ خارجَ المنسّق**
// ══════════════════════════════════════════════════════════════════════
//
// **والاختبارُ يُثبت مساراً مساراً** — **والحارسُ يمنع السابعَ الذي
// يُكتب غداً.**
func TestIDEM_AllProtectedPathsUseCoordinator(t *testing.T) {
	root := backendRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "internal/server/server.go"))
	if err != nil {
		t.Fatalf("قراءةُ المسارات: %v", err)
	}

	// **أسماءُ المعالجات المحميّة تُستخرَج من التسجيل نفسِه** —
	// **ولا تُكتب قائمةً تشيخ.**
	var protected []string
	for _, line := range strings.Split(string(src), "\n") {
		i := strings.Index(line, "s.idempotent(s.")
		if i < 0 {
			continue
		}
		rest := line[i+len("s.idempotent(s."):]
		if j := strings.IndexAny(rest, ")"); j > 0 {
			protected = append(protected, rest[:j])
		}
	}
	sort.Strings(protected)
	t.Logf("المساراتُ المحميّة: %d — %v", len(protected), protected)
	if len(protected) == 0 {
		t.Fatal("**لم يُعثَر على مسارٍ محميّ** — **وحارسٌ أعمى أسوأُ من لا حارس**")
	}

	// **ويُقرأ الشجرُ النحويُّ**: أيُّ معالجٍ محميٍّ ينادي المنسّق؟
	uses := map[string]bool{}
	err = filepath.Walk(filepath.Join(root, "internal/server"),
		func(p string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() || !strings.HasSuffix(p, ".go") ||
				strings.HasSuffix(p, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			f, perr := parser.ParseFile(fset, p, nil, 0)
			if perr != nil {
				return nil
			}
			for _, d := range f.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok &&
						sel.Sel.Name == "WithIdempotentTx" {
						uses[fn.Name.Name] = true
					}
					return true
				})
			}
			return nil
		})
	if err != nil {
		t.Fatalf("مسحُ الشجر: %v", err)
	}

	for _, h := range protected {
		if !uses[h] {
			t.Errorf("**مسارٌ محميٌّ لا يستعمل `WithIdempotentTx`**: %s — "+
				"**فعلامةُ تثبيته خارجَ معاملته، وهي فجوةُ `XG-33`.**", h)
		}
	}
}
