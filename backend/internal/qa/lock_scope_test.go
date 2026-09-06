package qa

import (
	"context"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **`XG-34` — نطاقُ القفل وتسرّبُ المعاملات**
// ══════════════════════════════════════════════════════════════════════
//
// **رُصد تعثّرٌ متقطّعٌ تحت حمل الحزمة الكاملة** — ٦٢ و٣٢ و٣٠ ثانية،
// **ومهلةُ الطلب ثلاثون.** **والظنُّ كان أنّ منسّقَ منع التكرار يُطيل
// إمساكَ المعاملات فيتنازع.**
//
// **وهذان الحارسان يقطعان الظنّ**: **مفتاحان مختلفان لا يتسلسلان**،
// **ولا معاملةَ تبقى مفتوحةً بعد الفحص.**

// ══════════════════════════════════════════════════════════════════════
// **١ · مفتاحان مختلفان لا يتسلسل أحدُهما خلف الآخر**
// ══════════════════════════════════════════════════════════════════════
//
// **والقفلُ على صفٍّ واحدٍ بمفتاح `(user_id, endpoint, key)`** — **فمن
// اختلف مفتاحُه اختلف صفُّه.** **ولو تسلسلا لكان القفلُ على الجدول لا
// على الصفّ، وذلك عطبٌ يخنق المنصّةَ تحت حمل.**
func TestXG34_DifferentKeysDoNotSerialize(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)

	before := idemOrdersOf(t, h, cust.ID)
	body := orderBody(item, 1)
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "مفتاحٌ-أ", Do: func(ctx context.Context) any {
			return h.POSTKey("/api/v1/orders", cust.Token, "xg34-a", body)
		}},
		Actor{Name: "مفتاحٌ-ب", Do: func(ctx context.Context) any {
			return h.POSTKey("/api/v1/orders", cust.Token, "xg34-b", body)
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	after := idemOrdersOf(t, h, cust.ID)
	t.Logf("مفتاحان مختلفان: طلباتٌ %d ← %d · نجح %d من 2 — %s",
		before, after, r.CountOK(), r)

	// **والتسلسلُ يظهر في العدّ لا في الزمن**: **نداءان كلاهما نجح
	// وأنتج طلباً يعني أنّهما لم يحبس أحدُهما الآخرَ إلى الفشل.**
	if after != before+2 {
		t.Errorf("**مفتاحان مختلفان ولم يُنتجا طلبين**: %d ← %d — "+
			"**فالقفلُ أوسعُ من صفِّه.**", before, after)
	}
	if r.CountOK() != 2 {
		t.Errorf("نجح %d من 2 — **ومفتاحان مختلفان لا يتنازعان**", r.CountOK())
	}
	// **وتداخلٌ مقيسٌ يُثبت أنّهما تزاحما فعلاً** — ولولاه لكان الفحصُ
	// نداءين متتاليين لا سباقاً.
	if r.Probe.Max() < 2 {
		t.Errorf("تداخلٌ مقيسٌ %d — **والسيناريو يشترط تزامناً**", r.Probe.Max())
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · لا معاملةَ تبقى مفتوحةً ولا اتّصالَ محجوزاً**
// ══════════════════════════════════════════════════════════════════════
//
// **ومعاملةٌ تُترَك مفتوحةً تمسك اتّصالاً وقفلاً معاً** — **فيتعثّر
// فحصٌ بعيدٌ بلا سببٍ ظاهر**، وهو عرَضُ `XG-34` بعينه.
//
// **ويُقاس بعد سلسلةٍ من النداءات المحميّة** — **لا على قاعدةٍ ساكنة**:
// **قياسٌ على السكون يمرّ أبداً ولا يرى شيئاً.**
func TestXG34_NoTransactionIsLeftOpen(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)

	for i := 0; i < 5; i++ {
		if got := h.POSTKey("/api/v1/orders", cust.Token,
			uniq("xg34-leak"), orderBody(item, 1)); got.Code >= 400 {
			t.Fatalf("الطلبُ %d: %s", i+1, got)
		}
	}

	var openTx, blocked int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT (SELECT count(*) FROM pg_stat_activity
		         WHERE datname = current_database()
		           AND state = 'idle in transaction'),
		       (SELECT count(*) FROM pg_stat_activity
		         WHERE datname = current_database()
		           AND cardinality(pg_blocking_pids(pid)) > 0)`).
		Scan(&openTx, &blocked); err != nil {
		t.Skipf("تعذّرت قراءةُ حال الخادم: %v", err)
	}
	t.Logf("بعد خمسةِ نداءاتٍ محميّة: معاملاتٌ مفتوحةٌ=%d · جلساتٌ محجوبةٌ=%d",
		openTx, blocked)

	if openTx > 0 {
		t.Errorf("**%d معاملةً بقيت مفتوحة** — **تمسك اتّصالاً وقفلاً "+
			"فيتعثّر فحصٌ بعيدٌ بلا سببٍ ظاهر.**", openTx)
	}
	if blocked > 0 {
		t.Errorf("**%d جلسةً محجوبة** بعد نداءاتٍ متتالية", blocked)
	}
}
