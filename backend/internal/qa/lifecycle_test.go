package qa

// **الطبقةُ الخامسة — دورةُ حياة الطلب وآلةُ الحال.**
//
// المعرّفات: `LIFE-*` · الوسم: `@api @e2e @critical @release`
//
// (أولويّةُ المالك ٢٠٢٦-٠٨-١٩، البندان ١٢ و١٣: «نريد أن نستطيع آليّاً
//
//	تحريكَ الطلب عبر حالاته… ثمّ أنشئ Tests لكلّ Transition».)
//
// # وقِيست على الإنتاج أوّلاً ثمّ ثُبّتت هنا
//
// **مُشيت الدورةُ كاملةً بحسابَي زبونٍ وسائقٍ حقيقيّين** (٢٠٢٦-٠٨-١٩)
// حتّى `delivered`. **وما مُشي مرّةً يُنسى** — فيُثبَّت اختباراً يعمل
// على قاعدة الاختبار بلا حسابٍ حقيقيّ.
//
// # وما كشفته المشية
//
// **`delivered` تُردّ `delivery_proof_required`** — قاعدةُ عملٍ لم تكن
// في وثيقة. **ولا تُقرأ عطباً**: من سلّم بلا إثباتٍ لا شيءَ يشهد له.

import (
	"net/http"
	"testing"
)

// driverOf **يصنع سائقاً ويُسنده إلى الطلب** — ولا يمرّ بطابورِ
// الإسناد: **اختبارُ آلةِ الحال ليس اختبارَ خوارزميّةِ التوزيع.**
func (h *Harness) driverOf(orderID string) *User {
	h.T.Helper()
	d := h.NewUser("driver")
	_, err := h.Pool.Exec(h.T.Context(), `
		UPDATE orders SET driver_id = $2::uuid, status = 'assigned'
		WHERE id = $1::uuid`, orderID, d.ID)
	if err != nil {
		h.T.Fatalf("qa: تعذّر إسنادُ السائق: %v", err)
	}
	return d
}

func (h *Harness) statusOf(orderID string) string {
	h.T.Helper()
	var s string
	if err := h.Pool.QueryRow(h.T.Context(),
		`SELECT status FROM orders WHERE id = $1::uuid`, orderID).Scan(&s); err != nil {
		h.T.Fatalf("qa: تعذّرت قراءةُ الحال: %v", err)
	}
	return s
}

// TestLIFE_001_HappyPath **الدورةُ الكاملةُ تمرّ.**
func TestLIFE_001_HappyPath(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)

	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("LIFE-001 تعذّر إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	if got := h.statusOf(oid); got != "pending" {
		t.Errorf("LIFE-001 الحالُ الابتدائيّة %q — يُنتظر pending", got)
	}

	drv := h.driverOf(oid)
	// **وكلُّ مرحلةٍ تُقاس بعدها** — **وردٌّ ٢٠٠ بلا تغيّرِ حالٍ يُقرأ
	// نجاحا.**
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to})
		if got.Code >= 400 {
			t.Fatalf("LIFE-001 الانتقالُ إلى %s رُدّ: %s", to, got)
		}
		if now := h.statusOf(oid); now != to {
			t.Errorf("LIFE-001 رُدّ 200 والحالُ %q لا %q", now, to)
		}
	}

	// ══════════════════════════════════════════════════════════════════
	// **والتسليمُ يحتاج إثباتاً — قاعدةُ عملٍ كُشفت بالمشية**
	// ══════════════════════════════════════════════════════════════════
	//
	// **ومن سلّم بلا إثباتٍ لا شيءَ يشهد له** حين يقول الزبونُ «لم
	// يصلني». **فالتخطّي بابٌ له سببٌ يُكتب.**
	bare := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"})
	if bare.Code < 400 {
		t.Errorf("LIFE-002 سُلّم بلا إثباتٍ ولا تخطٍّ — **لا شاهدَ للسائق**: %s", bare)
	} else if bare.Err() != "delivery_proof_required" {
		t.Logf("LIFE-002 رُدّ برمزٍ آخر: %s", bare.Err())
	}

	skip := h.POST("/api/v1/driver/orders/"+oid+"/proof/skip", drv.Token,
		map[string]any{"reason": "اختبارٌ آليّ"})
	if skip.Code >= 400 {
		t.Fatalf("LIFE-001 تعذّر تخطّي الإثبات: %s", skip)
	}
	done := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"})
	if done.Code >= 400 {
		t.Fatalf("LIFE-001 التسليمُ رُدّ بعد التخطّي: %s", done)
	}
	if now := h.statusOf(oid); now != "delivered" {
		t.Errorf("LIFE-001 الحالُ النهائيّة %q — يُنتظر delivered", now)
	}

	// **والزبونُ يرى ما وقع** — **وحالٌ تتغيّر في القاعدة ولا تصل
	// شاشتَه هي العطبُ الذي يشتكيه.**
	seen := h.GET("/api/v1/my/orders/"+oid, cust.Token)
	if seen.Code != http.StatusOK {
		t.Fatalf("LIFE-003 الزبونُ لا يقرأ طلبَه: %s", seen)
	}
	if st, _ := seen.JSON()["status"].(string); st != "delivered" {
		t.Errorf("LIFE-003 الزبونُ يرى %q والقاعدةُ delivered", st)
	}
}

// illegal **انتقالاتٌ ممنوعةٌ صراحة** — والقائمةُ من آلةِ الحال لا
// مُخمَّنة.
var illegal = []struct{ ID, From, To string }{
	{"LIFE-010", "delivered", "pending"},
	{"LIFE-011", "delivered", "cancelled"},
	{"LIFE-012", "cancelled", "accepted"},
	{"LIFE-013", "pending", "delivered"},
	{"LIFE-014", "delivered", "at_pickup"},
}

// TestLIFE_IllegalTransitions **وما لا يجوز يُردّ.**
//
// **وآلةُ حالٍ تقبل كلَّ شيءٍ ليست آلةَ حال** — ومن قفز من `pending`
// إلى `delivered` سلّم طلباً لم يُطبَخ.
func TestLIFE_IllegalTransitions(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("LIFE: تعذّر التجهيز: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)

	for _, c := range illegal {
		t.Run(c.ID, func(t *testing.T) {
			// **وتُزرع الحالُ مباشرةً** — بلوغُها بالطريق الشرعيّ
			// يجعل الاختبارَ يقيس الطريقَ لا المنع.
			if _, err := h.Pool.Exec(t.Context(),
				`UPDATE orders SET status = $2 WHERE id = $1::uuid`, oid, c.From); err != nil {
				t.Fatalf("تعذّرت زراعةُ الحال %q: %v", c.From, err)
			}
			got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
				map[string]any{"to": c.To})
			if got.Code < 400 {
				t.Errorf("%s %s ← %s قُبل — **آلةُ الحال مكسورة**: %s",
					c.ID, c.From, c.To, got)
			}
			if now := h.statusOf(oid); now != c.From {
				t.Errorf("%s رُدّ النداءُ والحالُ تغيّرت إلى %q", c.ID, now)
			}
		})
	}
}

// TestLIFE_020_ForeignDriverCannotMove **وسائقٌ غيرُ سائقِه لا يحرّكه.**
//
// **وهذا IDOR في آلة الحال**: من عرف معرّفَ طلبٍ حرّكه إلى `delivered`
// **فأُغلق طلبُ غيرِه وحُسبت له أجرتُه.**
func TestLIFE_020_ForeignDriverCannotMove(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("LIFE-020 تعذّر التجهيز: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	_ = h.driverOf(oid)
	intruder := h.NewUser("driver")

	got := h.POST("/api/v1/driver/orders/"+oid+"/transition", intruder.Token,
		map[string]any{"to": "picked_up"})
	if got.Code < 400 {
		t.Errorf("LIFE-020 سائقٌ أجنبيٌّ حرّك طلبَ غيرِه: %s", got)
	}
	if now := h.statusOf(oid); now != "assigned" {
		t.Errorf("LIFE-020 تغيّرت الحالُ إلى %q بيدِ سائقٍ أجنبيّ", now)
	}
}
