package qa

import (
	"context"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **إنذارُ الراصد: وسمٌ ونيّةٌ معاً** — `PF-07` · `R22`
// ══════════════════════════════════════════════════════════════════════
//
// # التسلسلُ الذي كان
//
//	١ UPDATE orders SET alerted_at   ← الوسم
//	٢ RowsAffected == 0 ⇒ لا إعادة   ← الحارس
//	٣ NotifyOps                      ← الإشعار
//
// **فإن سقط الإشعارُ أو مات المنفّذُ بينهما صمت الإنذارُ إلى الأبد** —
// **والوسمُ يمنع إعادةَ المحاولة.**
//
// **و`alerted_at` يقول «أُنذر» وهو لا يعني إلّا «قرّرنا أن نُنذر».**
//
// # وما صار
//
// **الوسمُ والنيّةُ الدائمةُ في معاملةٍ واحدة** — **فإن سقطت النيّةُ
// سقط الوسمُ وأُعيدت المحاولةُ في الجولة التالية.**
//
// **ولا يضمن هذا وصولَ الدفعة إلى هاتف** — **تلك شبكةٌ وطرفٌ ثالث،
// وعقدُها `PF-09`.**

// alertableOrder طلبٌ مستحقٌّ للإنذار — قديمٌ وبلا سائق.
func alertableOrder(t *testing.T, h *Harness) string {
	t.Helper()
	treasury(t, h)
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET status = 'pending', driver_id = NULL, alerted_at = NULL,
		                  created_at = now() - interval '3 hours'
		 WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("إشاخةُ الطلب: %v", err)
	}
	return oid
}

// alertState وسمُ الطلب وعددُ نيّاتِ إنذاره.
func alertState(t *testing.T, h *Harness, oid string) (marked bool, intents int) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT (SELECT alerted_at IS NOT NULL FROM orders WHERE id = $1::uuid),
		       (SELECT count(*) FROM notifications
		         WHERE entity = 'order' AND entity_id = $1::text
		           AND title IN ('طلبٌ لم يقبله متجره', 'طلبٌ بلا سائق',
		                         'طلبٌ تأخّر عن موعده'))`,
		oid).Scan(&marked, &intents); err != nil {
		t.Fatalf("قراءةُ حال الإنذار: %v", err)
	}
	return
}

// ══════════════════════════════════════════════════════════════════════
// **W1 · إنذارٌ مستحقٌّ ⇒ وسمٌ ونيّةٌ معاً**
// ══════════════════════════════════════════════════════════════════════
func TestR22_W1_AlertMarksAndCreatesIntent(t *testing.T) {
	h := New(t)
	_ = h.NewUser("ops") // **ولا وسمَ بلا مكتبٍ يُنذَر**
	oid := alertableOrder(t, h)

	h.Orders.EscalateAlertsOnce(ctxBG())
	marked, intents := alertState(t, h, oid)
	t.Logf("إنذارٌ مستحقّ: موسومٌ=%v · نيّاتٌ=%d", marked, intents)

	if !marked || intents == 0 {
		t.Errorf("**الإنذارُ لم يقع كاملاً**: موسومٌ=%v · نيّاتٌ=%d", marked, intents)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **W2+W3+W4 · سقوطُ النيّة ⇒ لا وسمَ والجولةُ التاليةُ تُعيد**
// ══════════════════════════════════════════════════════════════════════
//
// **وهو `PF-07` بعينه**: كان الوسمُ يبقى فيصمت الإنذارُ أبداً.
func TestR22_W2_IntentFailureLeavesNoMarker(t *testing.T) {
	h := New(t)
	_ = h.NewUser("ops")
	oid := alertableOrder(t, h)

	fp := h.ArmAny("R22/notify-write", "notifications", "INSERT")
	h.Orders.EscalateAlertsOnce(ctxBG())
	fp.MustFire(t)

	marked, intents := alertState(t, h, oid)
	t.Logf("سقوطُ النيّة: موسومٌ=%v · نيّاتٌ=%d", marked, intents)
	if marked {
		t.Errorf("**وُسم والإنذارُ لم يقع** — **والوسمُ يمنع الإعادةَ " +
			"فيصمت أبداً.** (`PF-07` · `R22`)")
	}
	if intents != 0 {
		t.Errorf("**نيّةٌ بقيت وقد سقط إدخالُها** (%d)", intents)
	}

	// **والجولةُ التالية تُعيد المحاولة.**
	h.Orders.EscalateAlertsOnce(ctxBG())
	marked2, intents2 := alertState(t, h, oid)
	t.Logf("الجولةُ التالية: موسومٌ=%v · نيّاتٌ=%d", marked2, intents2)
	if !marked2 || intents2 == 0 {
		t.Errorf("**لم تُعَد المحاولة**: موسومٌ=%v · نيّاتٌ=%d", marked2, intents2)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **W5 · نجاحٌ ثمّ جولةٌ ثانية ⇒ لا نيّةَ مكرَّرة**
// ══════════════════════════════════════════════════════════════════════
func TestR22_W5_SecondSweepDoesNotDuplicate(t *testing.T) {
	h := New(t)
	_ = h.NewUser("ops")
	oid := alertableOrder(t, h)

	h.Orders.EscalateAlertsOnce(ctxBG())
	_, first := alertState(t, h, oid)
	h.Orders.EscalateAlertsOnce(ctxBG())
	_, second := alertState(t, h, oid)
	t.Logf("جولتان: نيّاتٌ %d ← %d", first, second)

	if second != first {
		t.Errorf("**النيّةُ تكرّرت**: %d ← %d", first, second)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **W6 · راصدان معاً ⇒ نيّةٌ واحدة**
// ══════════════════════════════════════════════════════════════════════
func TestR22_W6_TwoWatchdogsAlertOnce(t *testing.T) {
	h := New(t)
	_ = h.NewUser("ops")
	oid := alertableOrder(t, h)

	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "راصدٌ-أ", Do: func(ctx context.Context) any {
			h.Orders.EscalateAlertsOnce(context.Background())
			return nil
		}},
		Actor{Name: "راصدٌ-ب", Do: func(ctx context.Context) any {
			h.Orders.EscalateAlertsOnce(context.Background())
			return nil
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	marked, intents := alertState(t, h, oid)
	t.Logf("راصدان: موسومٌ=%v · نيّاتٌ=%d — %s", marked, intents, r)

	// **والنيّاتُ نيّةٌ لكلّ موظّفِ عمليّات** — **لا نيّةٌ لكلّ راصد.**
	// **فمقياسُ التكرار أن تزيد على عدد المستقبِلين.**
	var desk int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(DISTINCT u.id) FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role_code = ANY($1) AND u.status = 'active'`,
		[]string{"admin", "ops"}).Scan(&desk); err != nil {
		t.Fatalf("عدُّ المكتب: %v", err)
	}
	t.Logf("مستقبِلو المكتب=%d", desk)

	if !marked {
		t.Error("**لم يقع إنذارٌ أصلاً**")
	}
	if desk > 0 && intents > desk {
		t.Errorf("**نيّاتٌ %d ومستقبِلون %d** — **راصدان أنشآ إنذارين**",
			intents, desk)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **وW9 لم يُكتَب** — ولا أدّعي فحصاً لا يُعزَل
// ══════════════════════════════════════════════════════════════════════
//
// **«ولا مكتبَ يُنذَر»** يحتاج قاعدةً بلا حسابِ عمليّاتٍ واحد —
// **وقاعدةُ الحزمة مشتركةٌ فيها حساباتُ فحوصٍ أخرى.**
//
// **والشرطُ مكتوبٌ في الشيفرة ومقروء** (`n == 0 ⇒ لا وسم`)، **ولم
// يُقَس** — فلا يُعَدّ مُثبَتاً.
