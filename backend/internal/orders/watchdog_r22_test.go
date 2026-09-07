package orders

// `R22` و`XOB-5` و`XOB-6` — **`P-7` البنود ١٦ و١٧ و١٨.**
//
// **وهنا لا في `internal/qa`** — **لأنّ `escalate` غيرُ مصدَّرة**، وحزمةُ
// الاختبار نفسُها تبلغها. **فلا مِعراضَ يلزم** (البند ١٦: «لا تبنِ نظاماً
// فرعيّاً جديداً»).
//
// **وهذا يُغلق ما طُلب في `P-6` بمِعراضٍ** — **ولم يلزم.**

import (
	"context"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"io"
	"log/slog"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestEV_R22WatchdogMarkerSuppressesRetry **البندان ١٦ و١٨.**
//
// **والترتيبُ مقيسٌ** (`watchdog.go:135`):
//
//	UPDATE orders SET alerted_at = now() WHERE alerted_at IS NULL  ← الوسم
//	if RowsAffected == 0 { continue }                              ← الحارس
//	s.notify.NotifyOps(…)                                          ← الإشعار
//
// **فالوسمُ يسبق الإشعار.** والسؤال: **إن سقط الإشعارُ، أيبقى الإنذارُ
// مكتوماً إلى الأبد؟**
func TestEV_R22WatchdogMarkerSuppressesRetry(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()

	// **وطلبٌ يُصنَع مباشرةً** — `escalate` لا تحتاج غيرَ صفٍّ موسومٍ أو لا.
	oid := seedAlertableOrder(t, pool)

	// **مُشعِرٌ بديلٌ يُحصي** — **ولا يمسّ شيفرةَ إنتاج**: `escalate` تنادي
	// الواجهةَ `Notifier`، والبديلُ يحلّ محلَّها في هذا الاختبار وحدَه.
	spy := &countingNotifier{}
	svc := &Service{db: pool, logger: quietLogger(), notify: spy}

	alerts := []Alert{{OrderID: oid, Number: 1, Status: "pending",
		MerchantName: "متجرُ QA", Reason: "no_accept"}}

	// ── التشغيلُ الأوّل ────────────────────────────────────────────
	svc.escalate(ctx, alerts)
	first := spy.calls
	var marked bool
	_ = pool.QueryRow(ctx,
		`SELECT alerted_at IS NOT NULL FROM orders WHERE id = $1::uuid`, oid).Scan(&marked)
	t.Logf("التشغيلُ الأوّل: نداءاتُ الإشعار=%d · موسومٌ=%v", first, marked)

	if !marked {
		t.Fatalf("لم يُوسَم الطلبُ — لا نافذةَ تُختبَر")
	}
	if first == 0 {
		t.Fatalf("لم يُنادَ الإشعارُ — لا شيءَ يسقط")
	}

	// ── والإشعارُ سقط · فيُعاد التشغيل ─────────────────────────────
	svc.escalate(ctx, alerts)
	second := spy.calls - first
	t.Logf("التشغيلُ الثاني: نداءاتُ الإشعار=%d", second)

	if second == 0 {
		t.Logf("R22 WATCHDOG ALERT = RISK CONFIRMED")
		t.Logf("  الوسمُ كُتب في التشغيل الأوّل · والإشعارُ سقط")
		t.Logf("  **والحارسُ (RowsAffected == 0) يمنع الإعادةَ إلى الأبد**")
		t.Logf("  ولا مسارَ يُصلحه — والعمليّاتُ لا تعلم أنّ إنذاراً ضاع")
		t.Logf("XOB-6 MARKER ORDERING = نفسُ الدليل — **ولا يلزم عيبٌ مستقلّ**")
		t.Logf("  RECOMMENDATION — XOB-6 يُطوى دليلاً في R22 لا يُجمَّد وحدَه")
	} else {
		t.Errorf("أُعيد الإشعارُ %d مرّة — **وR22 يقول إنّ الوسمَ يمنع. يُراجَع.**", second)
	}
}

// TestEV_XOB5_WatchdogComparisonKey **البند ١٧.**
//
// **ومفتاحُ المقارنة مقيسٌ** (`watchdog.go:107`):
//
//	key += a.OrderID + a.Reason + "|"
//
// **فهو يحمل المعرّفَ والسببَ وحدَهما** — **ولا يحمل الحالَ ولا العمر.**
// والسؤال: **أيبتلع تبدّلاً يستحقّ إنذاراً؟**
func TestEV_XOB5_WatchdogComparisonKey(t *testing.T) {
	// **والمفتاحُ يُبنى كما يبنيه المحرّك** — لا كما أظنّه.
	build := func(as []Alert) string {
		key := ""
		for _, a := range as {
			key += a.OrderID + a.Reason + "|"
		}
		return key
	}

	base := []Alert{{OrderID: "o-1", Number: 7, Status: "pending",
		MerchantName: "متجر", Reason: "no_accept", Minutes: 5}}

	cases := []struct {
		name  string
		alter func([]Alert) []Alert
		same  bool
	}{
		{"العمرُ تضاعف", func(a []Alert) []Alert {
			b := append([]Alert(nil), a...)
			b[0].Minutes = 90
			return b
		}, true},
		{"الحالُ تبدّلت", func(a []Alert) []Alert {
			b := append([]Alert(nil), a...)
			b[0].Status = "preparing"
			return b
		}, true},
		{"السببُ تبدّل", func(a []Alert) []Alert {
			b := append([]Alert(nil), a...)
			b[0].Reason = "no_driver"
			return b
		}, false},
	}

	swallowed := 0
	for _, c := range cases {
		got := build(c.alter(base)) == build(base)
		t.Logf("%-16s ⇒ المفتاحُ نفسُه: %v (يُنتظر: %v)", c.name, got, c.same)
		if got != c.same {
			t.Errorf("%s: المفتاحُ تصرّف خلافَ المقيس", c.name)
		}
		if got && c.same {
			swallowed++
		}
	}
	if swallowed > 0 {
		t.Logf("XOB-5 WATCHDOG KEY = CONTRACT GAP CANDIDATE")
		t.Logf("  المفتاحُ يحمل المعرّفَ والسببَ وحدَهما — **فتبدّلُ العمرِ والحالِ يُبتلَع**")
		t.Logf("  **والأثرُ محدود**: البلعُ يمنع إعادةَ البثِّ إلى غرفة العمليّات،")
		t.Logf("  **ولا يمنع `escalate`** — فالوسمُ هو الذي يمنعها (R22)")
		t.Logf("  RECOMMENDATION — ملاحظةٌ لا تُجمَّد: لا ضررَ مُثبَتٌ على الإنذار نفسِه")
	}
}

// countingNotifier مُشعِرٌ يُحصي ولا يفعل — **للاختبار وحدَه.**
//
// **وهو بديلُ الإشعارِ الساقط**: `escalate` تنادي `NotifyOps` ولا تقرأ
// نتيجةً، **فسقوطُ الإشعار وعدمُ وقوعه سواءٌ عندها** — وهذا بعينُه ما
// يجعل `R22` ما هو.
type countingNotifier struct{ calls int }

func (c *countingNotifier) Notify(ctx context.Context, in notifications.Input) {}
func (c *countingNotifier) NotifyRoles(ctx context.Context, roles []string, in notifications.Input) {
}
func (c *countingNotifier) NotifyOps(ctx context.Context, in notifications.Input) { c.calls++ }

// seedAlertableOrder طلبٌ قابلٌ للإنذار — **بأدنى ما يلزم.**
func seedAlertableOrder(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	ctx := context.Background()
	cust := testdb.NewUser(t, pool, "customer")
	// **والطلبُ القياسيُّ يشترط متجراً** — `orders_standard_has_merchant`.
	var catID, mID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO categories (name, active) VALUES ($1, true) RETURNING id::text`,
		"تصنيف R22 "+cust[:8]).Scan(&catID); err != nil {
		t.Fatalf("تصنيف: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, status, location)
		VALUES ($1, $2::uuid, 'active',
		        ST_SetSRID(ST_MakePoint(39.0094, 35.9506), 4326)::geography)
		RETURNING id::text`, "متجر R22 "+cust[:8], catID).Scan(&mID); err != nil {
		t.Fatalf("متجر: %v", err)
	}
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, status, payment_method, subtotal,
		                    delivery_fee, discount, total, wallet_paid, cash_due,
		                    address_text, dropoff)
		VALUES ($1::uuid, $2::uuid, 'pending', 'cash', 1000, 0, 0, 1000, 0, 1000,
		        'الرقة — اختبار',
		        ST_SetSRID(ST_MakePoint(39.0094, 35.9506), 4326)::geography)
		RETURNING id::text`, cust, mID).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ طلبٍ للإنذار: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM orders WHERE id = $1::uuid`, id)
		_, _ = pool.Exec(c, `DELETE FROM merchants WHERE id = $1::uuid`, mID)
		_, _ = pool.Exec(c, `DELETE FROM categories WHERE id = $1::uuid`, catID)
	})
	return id
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// **ونسختا المعاملة** — `PF-07`: الواجهةُ توسّعت فتوسّع الجاسوس.
func (s *countingNotifier) NotifyOpsTx(ctx context.Context, q dbtx.Querier,
	in notifications.Input) (int, error) {
	s.NotifyOps(ctx, in)
	return 1, nil
}

func (s *countingNotifier) PublishToUsers(ctx context.Context, q dbtx.Querier,
	roles []string, in notifications.Input) {
}
