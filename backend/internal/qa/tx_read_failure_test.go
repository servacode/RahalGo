package qa

// ══════════════════════════════════════════════════════════════════════
// **قراءةٌ تسقط داخلَ الوحدة ⇒ ترتدّ الوحدةُ كلُّها** — `XG-46` · `XG-33`
// ══════════════════════════════════════════════════════════════════════
//
// # لماذا هذا الفحصُ بعينه
//
// **دورةُ ٣٩ نقلت ثلاثةَ عشرَ قارئاً إلى معاملة الوحدة.** **والمُثبَت
// حتّى الآن ارتدادٌ عند سقوط كتابة** (`IDEM_T5` تُسقط إدراجَ الطلب) —
// **لا عند سقوط قراءةٍ صارت على المعاملة.**
//
// **والفرقُ ليس شكليّاً**: **الكتابةُ تسقط قبل أن يقع شيء، والقراءةُ
// هنا تقع بعد أن كُتب الطلبُ وبنودُه ولقطتُه.** **فالمُقاسُ ارتدادٌ
// حقيقيٌّ لا امتناعٌ مبكّر.**
//
// # وكيف يُسقَط قارئٌ إسقاطاً حقيقيّاً
//
// **الحاقنُ المعتمَد محفّزاتُ صفوف** — **ولا محفّزَ قبل `SELECT` في
// بوستغرس.** **فيُخفى الجدولُ نفسُه** (`ALTER TABLE … RENAME`)
// فيسقط الاستعلامُ بخطأٍ حقيقيّ (`42P01`) لا بخطأِ تحقّقٍ مصنوع.
//
// **و`order_ratings` هو الجدولُ المختار**: **تقرؤه `ratingFor` بعد
// الكتابات كلِّها، ولا تكتب فيه وحدةُ الإنشاء** — **فإخفاؤه يُسقط
// قراءةً ولا يمسّ كتابة.**
//
// # وهو يقيس ابتلاعَ الخطأ أيضاً
//
// **`ratingFor` تبتلع خطأها** (`if err != nil { return nil }`). **وعلى
// المَسبَح كان ذلك بلا أثر؛ وعلى المعاملة يُفسدها** — **فيسقط
// التثبيتُ ويرتدّ كلُّ شيء.**
//
// **والاتّجاهُ آمن**: **لا كتابةَ جزئيّةَ ولا نجاحَ كاذب** — وهذا ما
// يُثبَت هنا لا ما يُدّعى.

import (
	"testing"
)

func TestXG46_ReadFailureInsideUnitRollsBackEverything(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)
	const key = "xg46-read-failure"

	count := func(q string) int64 {
		t.Helper()
		var n int64
		if err := h.Pool.QueryRow(ctxBG(), q, cust.ID).Scan(&n); err != nil {
			t.Fatalf("عدٌّ: %v", err)
		}
		return n
	}
	const (
		qOrders = `SELECT count(*) FROM orders WHERE customer_id = $1::uuid`
		qItems  = `SELECT count(*) FROM order_items i
		             JOIN orders o ON o.id = i.order_id WHERE o.customer_id = $1::uuid`
		qEvents = `SELECT count(*) FROM order_events e
		             JOIN orders o ON o.id = e.order_id WHERE o.customer_id = $1::uuid`
		qSnap = `SELECT count(*) FROM orders
		           WHERE customer_id = $1::uuid AND snap_commission_source IS NOT NULL`
		qMoney  = `SELECT count(*) FROM wallet_transactions WHERE user_id = $1::uuid`
		qNotify = `SELECT count(*) FROM notifications WHERE user_id = $1::uuid`
	)
	before := map[string]int64{
		"orders": count(qOrders), "items": count(qItems), "events": count(qEvents),
		"snap": count(qSnap), "money": count(qMoney), "notify": count(qNotify),
	}

	// ── إخفاءُ الجدول: القراءةُ تسقط بخطأٍ حقيقيّ ──────────────────
	hide := func(on bool) {
		t.Helper()
		sql := `ALTER TABLE order_ratings RENAME TO order_ratings_xg46_hidden`
		if !on {
			sql = `ALTER TABLE order_ratings_xg46_hidden RENAME TO order_ratings`
		}
		if _, err := h.Pool.Exec(ctxBG(), sql); err != nil {
			t.Fatalf("إخفاءُ الجدول (%v): %v", on, err)
		}
	}
	hide(true)
	restored := false
	t.Cleanup(func() {
		if !restored {
			_, _ = h.Pool.Exec(ctxBG(),
				`ALTER TABLE order_ratings_xg46_hidden RENAME TO order_ratings`)
		}
	})

	got := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	st := claimStateOf(t, h, cust.ID, idemOrdersEndpoint, key)
	t.Logf("XG-46 قراءةٌ ساقطة: الردُّ=%d · المطالبةُ موجودةٌ=%v مثبَّتةٌ=%v",
		got.Code, st.Found, st.Committed)

	if got.Code < 400 {
		t.Fatalf("**نجاحٌ كاذبٌ ومعاملةٌ فاسدة** — الردُّ %d: %s", got.Code, got.Body)
	}

	// ── ولا أثرَ يبقى ─────────────────────────────────────────────
	for name, q := range map[string]string{
		"orders": qOrders, "items": qItems, "events": qEvents,
		"snap": qSnap, "money": qMoney, "notify": qNotify,
	} {
		if now := count(q); now != before[name] {
			t.Errorf("**بقي أثرٌ بعد الارتداد** — %s: %d ← %d", name, before[name], now)
		}
	}
	if st.Committed {
		t.Error("**عُلّمت المطالبةُ مثبَّتةً وعملُها ارتدّ** — " +
			"**وذاك يحبس الإعادةَ إلى الأبد.** (`XG-33`)")
	}

	// ── ثمّ تُعاد فتنفّذ مرّةً واحدة ──────────────────────────────
	hide(false)
	restored = true
	retry := h.POSTKey("/api/v1/orders", cust.Token, key, orderBody(item, 1))
	t.Logf("XG-46 الإعادةُ: %d · طلباتٌ %d ← %d",
		retry.Code, before["orders"], count(qOrders))

	if retry.Code >= 400 {
		t.Errorf("**الإعادةُ محبوسةٌ بعد ارتداد** (%d): %s — "+
			"**والعملُ لم يقع قطّ**", retry.Code, retry.Body)
	}
	if now := count(qOrders); now != before["orders"]+1 {
		t.Errorf("**الإعادةُ لم تنفّذ مرّةً واحدة**: %d ← %d", before["orders"], now)
	}
}
