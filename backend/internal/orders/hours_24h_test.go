package orders_test

import (
	"context"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// ══════════════════════════════════════════════════════════════════════
// **«مفتوح ٢٤ ساعة»** — ٠٠:٠٠ ← ٠٠:٠٠ (قرارُ المالك ٢٠٢٦-١٠-١٨)
// ══════════════════════════════════════════════════════════════════════
//
// **وقبل الإصلاح**: الاستقبالُ صحيحٌ ضمناً، **لكنّ `NextOpenSQL` يعيد «يفتح
// ٠٠:٠٠ غداً» لمتجرٍ مفتوحٍ الساعةَ** — صفُّ الغد في المستقبل.

// open24Merchant متجرٌ أيّامُه السبعةُ ٠٠:٠٠ ← ٠٠:٠٠.
func open24Merchant(t *testing.T, closedDay int) string {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	var categoryID, merchantID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, status)
		VALUES ('متجرُ الأربع والعشرين', $1, 10, 'active') RETURNING id`, categoryID).
		Scan(&merchantID); err != nil {
		t.Fatalf("تعذّر إنشاء متجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, merchantID)
	})
	for d := 0; d < 7; d++ {
		if _, err := pool.Exec(ctx, `
			INSERT INTO merchant_hours (merchant_id, day_of_week, closed, open_time, close_time)
			VALUES ($1, $2, $3, '00:00', '00:00')`, merchantID, d, d == closedDay); err != nil {
			t.Fatalf("تعذّر ضبط الدوام: %v", err)
		}
	}
	return merchantID
}

// TestOpen24h_NoOpensAtWhileOpen **مفتوحٌ الآن ولا «يفتح الساعة».**
func TestOpen24h_NoOpensAtWhileOpen(t *testing.T) {
	pool := testdb.Pool(t)
	id := open24Merchant(t, -1)
	var open bool
	var next *time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT `+orders.OpenNowSQL+`, `+orders.NextOpenSQL+` FROM merchants m WHERE m.id = $1`, id).
		Scan(&open, &next); err != nil {
		t.Fatalf("الفحص: %v", err)
	}
	if !open {
		t.Error("متجرٌ ٢٤ ساعة قُرئ مغلقاً")
	}
	if next != nil {
		t.Errorf("قيل «يفتح الساعة %v» لمتجرٍ مفتوحٍ ٢٤ ساعة", next)
	}
}

// TestOpen24h_AtEveryHour **الثالثةُ فجراً والظهرُ و٢٣:٥٩:٣٠ — مفتوحٌ ولا موعد.**
//
// **ويومُ العطلة بعده مغلقٌ من أوّل ثانية** — ٠٠:٠٠ ← ٠٠:٠٠ لا يمتدّ إلى الغد.
func TestOpen24h_AtEveryHour(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	// **الثلاثاء ٢٠٢٦-١٠-٢٠ مفتوحٌ ٢٤ ساعة، والأربعاءُ (٣) عطلة.**
	id := open24Merchant(t, 3)
	const at = `($2::timestamp AT TIME ZONE 'Asia/Damascus')`
	check := func(local string) (bool, *time.Time) {
		var open bool
		var next *time.Time
		if err := pool.QueryRow(ctx,
			`SELECT `+orders.OpenAtSQL(at)+`, `+orders.NextOpenAtSQL(at)+` FROM merchants m WHERE m.id = $1`,
			id, local).Scan(&open, &next); err != nil {
			t.Fatalf("الفحص %s: %v", local, err)
		}
		return open, next
	}
	for _, local := range []string{"2026-10-20 03:00:00", "2026-10-20 12:00:00", "2026-10-20 23:59:30", "2026-10-20 00:00:00"} {
		open, next := check(local)
		if !open {
			t.Errorf("%s: متجرٌ ٢٤ ساعة قُرئ مغلقاً", local)
		}
		if next != nil {
			t.Errorf("%s: قيل «يفتح الساعة %v» وهو مفتوح", local, next)
		}
	}
	// **والعطلةُ بعده مغلقةٌ — وموعدُها أوّلُ الخميس.**
	open, next := check("2026-10-21 00:30:00")
	if open {
		t.Error("يومُ العطلة بعد يومٍ ٢٤ ساعة قُرئ مفتوحاً — ٠٠:٠٠ امتدّ إلى الغد")
	}
	want := "2026-10-22 00:00"
	if next == nil || next.In(time.FixedZone("Damascus", 3*3600)).Format("2006-01-02 15:04") != want {
		t.Errorf("موعدُ العودة %v — والمنتظَر %s", next, want)
	}
}

// TestOpen24hHelper **المعنى واحدٌ لـ`HH:MM` و`HH:MM:SS`.**
func TestOpen24hHelper(t *testing.T) {
	if !orders.Open24h("00:00", "00:00") || !orders.Open24h("00:00:00", "00:00:00") {
		t.Error("٠٠:٠٠ ← ٠٠:٠٠ لم تُقرأ ٢٤ ساعة")
	}
	if orders.Open24h("09:00", "09:00") || orders.Open24h("00:00", "23:59") {
		t.Error("غيرُ ٠٠:٠٠ ← ٠٠:٠٠ قُرئ ٢٤ ساعة")
	}
}
