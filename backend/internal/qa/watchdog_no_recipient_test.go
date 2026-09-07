package qa

import "testing"

// ══════════════════════════════════════════════════════════════════════
// **W9 · ولا موظّفَ عمليّاتٍ يُنذَر** — `PF-07` · `R22`
// ══════════════════════════════════════════════════════════════════════
//
// # السؤال
//
// **`alerted_at` يقول «أُنذر».** فماذا يقول حين لا مستقبِلَ أصلاً؟
//
// **ووسمٌ بلا مُنذَرٍ كذبٌ** — **يُسكت الطلبَ إلى الأبد بينما لم يعلم
// به أحد.**
//
// # كيف عُزل الشرط
//
// **القاعدةُ المشتركةُ فيها موظّفو فحوصٍ أخرى** — **فالشرطُ لا يتحقّق
// فيها من تلقائه.**
//
// **وجُرّبت قاعدةٌ ثانيةٌ فأُلغيت**: **قيست فأسقطت ثلاثةَ فحوصٍ أخرى
// بانتظار قفل** (`TestEV_R23PushFailureIsLost` و`TestEV_R21…` و
// `TestRACE_FinancialTruth…`) — **صفرُ سقطاتٍ بدونها وثلاثٌ معها،
// مرّتين.** **وفحصٌ يُسقط ثمانمئةَ فحصٍ غيرِه ليس فحصاً.**
//
// **فصار الشرطُ يُصنَع في القاعدة نفسِها ويُستردّ**: **يُعطَّل موظّفو
// المكتب لحظةَ الجولة ثمّ يُعادون** — **ولا يُحذَف مستخدمٌ ولا تُمسّ
// محفظة**، **وفحوصُ الحزمة متتاليةٌ لا متوازية.**
//
// **والعكسُ ممنوع**: **إضافةُ موظّفٍ لتمرير الفحص تقيس عكسَ ما يُسأل
// عنه.**

// TestR22_W9_NoOpsRecipientLeavesNoMarker **صفرُ مستقبِلين ⇒ لا وسم.**
func TestR22_W9_NoOpsRecipientLeavesNoMarker(t *testing.T) {
	h := New(t)
	oid := alertableOrder(t, h)

	// ══════════════════════════════════════════════════════════════
	// **يُخلى المكتبُ لحظةً ثمّ يُعاد**
	// ══════════════════════════════════════════════════════════════
	//
	// **والاستردادُ في `Cleanup` لا في نهاية الجسد** — **فسقوطٌ في
	// المنتصف لا يترك الحزمةَ بلا مكتب.**
	//
	// **و`suspended` لا `deleted`**: **حالٌ قائمةٌ في المنتَج**،
	// **والاستعلامُ الذي يُنذِر يشترط `active`.**
	emptied, err := h.Pool.Exec(ctxBG(), `
		UPDATE users SET status = 'suspended'
		 WHERE status = 'active' AND id IN (
		   SELECT ur.user_id FROM user_roles ur WHERE ur.role_code = ANY($1))`,
		[]string{"admin", "ops"})
	if err != nil {
		t.Fatalf("إخلاءُ المكتب: %v", err)
	}
	t.Cleanup(func() {
		if _, err := h.Pool.Exec(ctxBG(), `
			UPDATE users SET status = 'active'
			 WHERE status = 'suspended' AND id IN (
			   SELECT ur.user_id FROM user_roles ur
			    WHERE ur.role_code = ANY($1))`,
			[]string{"admin", "ops"}); err != nil {
			t.Errorf("**تعذّرت إعادةُ المكتب** — **وما بعده يعمل بلا "+
				"موظّفين**: %v", err)
		}
	})
	t.Logf("عُطّل %d موظّفاً مؤقّتاً", emptied.RowsAffected())

	// **والقياسُ في لحظة الجولة لا قبل بناء التركيبة.**
	desk := deskSize(t, h)
	t.Logf("مستقبِلو المكتب عند الجولة = %d", desk)
	if desk != 0 {
		t.Fatalf("**التركيبةُ ليست بصفر مستقبِلين**: %d — "+
			"**والفحصُ لا معنى له.**", desk)
	}

	// ── الجولةُ الأولى ────────────────────────────────────────────
	h.Orders.EscalateAlertsOnce(ctxBG())
	marked, intents := alertState(t, h, oid)
	t.Logf("الجولةُ الأولى: موسومٌ=%v · نيّاتٌ=%d", marked, intents)

	if intents != 0 {
		t.Errorf("**نيّاتٌ %d ولا مستقبِلَ**", intents)
	}
	if marked {
		t.Errorf("**وُسم «أُنذر» ولا مستقبِلَ** — **فيصمت الطلبُ أبداً " +
			"ولم يعلم به أحد.** (`PF-07` · `R22`)")
	}

	// ── الجولةُ الثانية: يبقى مستحقّاً ─────────────────────────────
	h.Orders.EscalateAlertsOnce(ctxBG())
	marked2, intents2 := alertState(t, h, oid)
	t.Logf("الجولةُ الثانية: موسومٌ=%v · نيّاتٌ=%d", marked2, intents2)
	if marked2 {
		t.Errorf("**وُسم في الجولة الثانية ولا مستقبِلَ**")
	}

	// ══════════════════════════════════════════════════════════════
	// **ولا دورةَ محمومة**
	// ══════════════════════════════════════════════════════════════
	//
	// **الراصدُ دوريٌّ بفترةٍ ثابتة** — **لا يُعيد المحاولةَ في حلقةٍ
	// داخل الجولة الواحدة.** **فبقاءُ الطلب مستحقّاً يعني جولةً
	// واحدةً كلَّ دورة، لا حَمْيَ.**
	//
	// **والقياسُ هنا**: **جولتان متتاليتان لم تُنشئا شيئاً ولم
	// تسقطا** — **والعملُ محدودٌ بـ`LIMIT` في الاستعلام.**
	if t.Failed() {
		return
	}
	t.Log("**ولا مانعَ للتعافي**: الطلبُ باقٍ مستحقّاً — " +
		"**فمن عيّن موظّفَ عمليّاتٍ أُنذر عنه في الجولة التالية.**")

	// **والدليلُ على ذلك بالفعل**: يُعيَّن موظّفٌ **بعد** القياس، فيقع.
	_ = h.NewUser("ops")
	h.Orders.EscalateAlertsOnce(ctxBG())
	marked3, intents3 := alertState(t, h, oid)
	t.Logf("بعد تعيين موظّف: موسومٌ=%v · نيّاتٌ=%d", marked3, intents3)
	if !marked3 || intents3 == 0 {
		t.Errorf("**لم يُنذَر بعد تعيين موظّف**: موسومٌ=%v · نيّاتٌ=%d",
			marked3, intents3)
	}
}

// deskSize عددُ موظّفي المكتب الفاعلين — **الاستعلامُ نفسُه الذي يُنذِر.**
func deskSize(t *testing.T, h *Harness) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(DISTINCT u.id) FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role_code = ANY($1) AND u.status = 'active'`,
		[]string{"admin", "ops"}).Scan(&n); err != nil {
		t.Fatalf("عدُّ المكتب: %v", err)
	}
	return n
}
