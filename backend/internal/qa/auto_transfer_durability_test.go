package qa

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **`PF-08` — تقدّمٌ لازمٌ في خيطٍ عابر**
// ══════════════════════════════════════════════════════════════════════
//
// # التسلسلُ الحاليُّ مقيسٌ من الشيفرة
//
//	تثبيتُ الطلب  →  AfterCommit → go autoTransfer(...)   ← حدُّ انهيارٍ ١
//	→ إعدادُ `orders.auto_transfer` · وحالُ الطلب `pending`
//	→ Transition ⇒ `accepted`                              ← تحوّلٌ دائمٌ ١ · حدّ ٢
//	→ واتساب SendText                                      ← تواصلٌ · حدّ ٣
//	→ UPDATE sent_to_merchant_at                           ← تحوّلٌ دائمٌ ٢ · حدّ ٤
//	→ AutoDispatch ⇒ `dispatching`                         ← تحوّلٌ دائمٌ ٣ · حدّ ٥
//
// # وما يتعافى اليوم
//
// **`sweepAutoAccept` يلتقط ما بقي `pending`** بعد `orders.auto_accept_min`
// — **فيغطّي الحدَّين ١ و٢ إن كان الإعدادُ موجباً.**
//
// **ولا شيءَ يلتقط ما بلغ `accepted` ولم يُنزَل**: **طلبٌ مقبولٌ بلا
// إنزالٍ يبقى كذلك أبداً** — **لا سائقَ يُعرَض عليه ولا أحدَ يعلم.**
//
// **وهذا هو `PF-08`**: **تقدّمٌ لازمٌ مصيرُه خيطٌ يموت بموت العمليّة.**

// stuckAccepted طلبٌ بلغ `accepted` ولم يُنزَل — **كما يتركه انهيارٌ
// بين الحدّ ٢ والحدّ ٥.**
func stuckAccepted(t *testing.T, h *Harness) string {
	t.Helper()
	treasury(t, h)
	h.Setting("orders.auto_transfer", "true")
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)

	// **ويُعاد إلى `accepted` بلا سائق** — **حالُ الانهيار بعينها.**
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET status = 'accepted', driver_id = NULL,
		                  updated_at = now() - interval '10 minutes'
		 WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("تهيئةُ حال الانهيار: %v", err)
	}
	return oid
}

func orderStatus(t *testing.T, h *Harness, oid string) string {
	t.Helper()
	var st string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT status FROM orders WHERE id = $1::uuid`, oid).Scan(&st); err != nil {
		t.Fatalf("قراءةُ الحال: %v", err)
	}
	return st
}

// ══════════════════════════════════════════════════════════════════════
// **F1+F4 · انهيارٌ قبل الإنزال ⇒ يتعافى**
// ══════════════════════════════════════════════════════════════════════
func TestPF08_F1_StuckAcceptedIsRecovered(t *testing.T) {
	h := New(t)
	oid := stuckAccepted(t, h)
	before := orderStatus(t, h, oid)

	picked, progressed := h.API.SweepAutoTransfer(ctxBG())
	after := orderStatus(t, h, oid)
	t.Logf("طلبٌ عالقٌ في `accepted`: %q ← %q · التُقط=%d · تقدّم=%d",
		before, after, picked, progressed)

	if after == "accepted" {
		t.Errorf("**بقي عالقاً بعد الكنس** — **ولا مصدرَ دائمٌ لتقدّمه.** "+
			"(`PF-08`) · الحالُ %q", after)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **F6 · كانسان معاً ⇒ تقدّمٌ واحد**
// ══════════════════════════════════════════════════════════════════════
func TestPF08_F6_TwoWorkersProgressOnce(t *testing.T) {
	h := New(t)
	oid := stuckAccepted(t, h)

	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "كانسٌ-أ", Do: func(ctx context.Context) any {
			h.API.SweepAutoTransfer(ctxBG())
			return nil
		}},
		Actor{Name: "كانسٌ-ب", Do: func(ctx context.Context) any {
			h.API.SweepAutoTransfer(ctxBG())
			return nil
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	// **ولا يُبتلَع خطأُ القياس** — **قياسٌ يسقط صامتاً يقرأ صفراً
	// فيُقرأ الصفرُ نجاحاً وهو عمى.**
	var events int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM order_events
		 WHERE order_id = $1::uuid AND to_status = 'dispatching'`,
		oid).Scan(&events); err != nil {
		t.Fatalf("عدُّ أحداث الإنزال: %v", err)
	}
	t.Logf("كانسان: حالٌ=%q · أحداثُ إنزالٍ=%d — %s",
		orderStatus(t, h, oid), events, r)

	if events != 1 {
		t.Errorf("**أحداثُ الإنزال %d والمتوقَّع واحد** — "+
			"**وكانسان يقتسمان العملَ ولا يكرّرانه.**", events)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **F10 · والإعدادُ مُطفأً ⇒ لا تقدّمَ يُخترَع**
// ══════════════════════════════════════════════════════════════════════
func TestPF08_F10_DisabledMeansNoProgression(t *testing.T) {
	h := New(t)
	oid := stuckAccepted(t, h)
	h.Setting("orders.auto_transfer", "false")

	_, _ = h.API.SweepAutoTransfer(ctxBG())
	after := orderStatus(t, h, oid)
	t.Logf("والإعدادُ مُطفأ: الحالُ %q", after)
	if after != "accepted" {
		t.Errorf("**تقدّمَ والإعدادُ مُطفأ**: %q", after)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **F11+F12 · ما تقدّم أو انتهى لا يُمَسّ**
// ══════════════════════════════════════════════════════════════════════
func TestPF08_F11_ProgressedAndTerminalAreNoOp(t *testing.T) {
	h := New(t)
	for _, st := range []string{"dispatching", "delivered", "cancelled"} {
		oid := stuckAccepted(t, h)
		if _, err := h.Pool.Exec(ctxBG(),
			`UPDATE orders SET status = $2 WHERE id = $1::uuid`, oid, st); err != nil {
			t.Fatalf("ضبطُ الحال %s: %v", st, err)
		}
		_, _ = h.API.SweepAutoTransfer(ctxBG())
		got := orderStatus(t, h, oid)
		t.Logf("%-12s ⇒ %q", st, got)
		if got != st {
			t.Errorf("**مُسّت حالٌ لا تُمَسّ**: %q ← %q", st, got)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **F13 · إعادةُ تشغيلٍ ⇒ العملُ في القاعدة لا في الذاكرة**
// ══════════════════════════════════════════════════════════════════════
//
// **ومِسنَدٌ ثانٍ على القاعدة نفسِها هو «الخادمُ بعد الإقلاع»** —
// **عمليّةٌ لم تعرف الطلبَ قطّ ولا خيطَ لها فيه.** **فإن التقطته
// فالمصدرُ دائمٌ حقّاً.**
func TestPF08_F13_RecoveryNeedsNoMemory(t *testing.T) {
	h := New(t)
	oid := stuckAccepted(t, h)

	fresh := New(t) // **خادمٌ آخرُ — كأنّه أُقلع للتوّ**
	picked, progressed := fresh.API.SweepAutoTransfer(ctxBG())
	after := orderStatus(t, h, oid)
	t.Logf("خادمٌ آخرُ يكنس: التُقط=%d · تقدّم=%d · الحالُ %q",
		picked, progressed, after)

	if after == "accepted" {
		t.Errorf("**لم يتعافَ بخادمٍ آخر** — **فالمصدرُ في الذاكرة لا في "+
			"القاعدة.** الحالُ %q", after)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **F7 · وواتسابُ تواصلٌ لا حقيقة**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا مُبلِّغَ في المِسنَد أصلاً** (`merchantReady() == false`) —
// **فهذه هي حالُ تعذّر الإبلاغ بعينها.** **والطلبُ يبقى صحيحاً
// والتقدّمُ يبقى ممكناً** — وهو ما تُثبته `F1` بجميعها.
func TestPF08_F7_CommunicationFailureKeepsOrderRecoverable(t *testing.T) {
	h := New(t)
	oid := stuckAccepted(t, h)

	var total, refunds int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT (SELECT count(*) FROM orders WHERE id = $1::uuid),
		       (SELECT count(*) FROM wallet_transactions
		         WHERE ref = $1::text AND kind = 'refund')`,
		oid).Scan(&total, &refunds); err != nil {
		t.Fatalf("قراءةُ الطلب: %v", err)
	}
	_, progressed := h.API.SweepAutoTransfer(ctxBG())
	t.Logf("بلا مُبلِّغ: الطلبُ قائمٌ=%d · استرداداتٌ=%d · تقدّم=%d",
		total, refunds, progressed)

	if total != 1 {
		t.Errorf("**الطلبُ اختفى** بتعذّر الإبلاغ")
	}
	if refunds != 0 {
		t.Errorf("**ارتدّ مالٌ** بتعذّر الإبلاغ — %d قيداً", refunds)
	}
	if progressed != 1 {
		t.Errorf("**لم يتقدّم** رغم أنّ الإبلاغَ تواصلٌ لا شرطُ تقدّم")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **حارسٌ بنيويّ: لا تقدّمَ لازمٌ مصيرُه خيطٌ وحدَه**
// ══════════════════════════════════════════════════════════════════════
//
// **والخيطُ السريعُ يبقى تعجيلاً** — **والحارسُ يمنع أن يعود مصدرَ
// الحقيقة الوحيد.**
func TestPF08_FastPathHasDurableCounterpart(t *testing.T) {
	root := backendRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "internal/server/customer_handlers.go"))
	if err != nil {
		t.Fatalf("قراءةُ المعالج: %v", err)
	}
	if !strings.Contains(string(src), "go s.autoTransfer(") {
		t.Log("لا خيطَ سريعاً — والمصدرُ الدائمُ وحدَه")
		return
	}
	sweep, err := os.ReadFile(filepath.Join(root, "internal/server/auto_transfer_sweep.go"))
	if err != nil {
		t.Fatalf("**خيطٌ سريعٌ بلا كانسٍ دائم** — وهو `PF-08` بعينه: %v", err)
	}
	for _, must := range []string{"SweepAutoTransfer", "RunAutoTransferSweeper",
		"FOR UPDATE SKIP LOCKED"} {
		if !strings.Contains(string(sweep), must) {
			t.Errorf("**الكانسُ الدائمُ ينقصه %q**", must)
		}
	}
	main, err := os.ReadFile(filepath.Join(root, "cmd/api/main.go"))
	if err != nil || !strings.Contains(string(main), "RunAutoTransferSweeper") {
		t.Error("**الكانسُ لا يُشغَّل عند الإقلاع** — **ومصدرٌ دائمٌ لا " +
			"يعمل ليس مصدراً.**")
	}
}
