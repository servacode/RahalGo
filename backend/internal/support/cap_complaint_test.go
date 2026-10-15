package support_test

// **سقفُ اقتراح الدعم هو سقفُ موافقة الشكوى** (قرارُ المالك ٢٠٢٦-١٠-١٥) —
// `compensations.cap_complaint` لا `finance.manual_wallet_max`.

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

func TestCMP1015_ComplaintCapIsCapComplaint(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	st := settings.NewStore(pool)
	if err := st.SetInternal(ctx, "compensations.cap_complaint", 3000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM app_settings WHERE key = 'compensations.cap_complaint'`)
	})
	svc := support.NewService(pool, nil, wallet.NewService(pool))
	svc.SetSettings(st)
	agent := testdb.NewUser(t, pool, "admin")
	customer := testdb.NewUser(t, pool, "customer")
	newTicket := func() string {
		return cmpTicket(t, ctx, `
			INSERT INTO tickets (customer_id, subject, reason, created_by, opened_by_customer)
			VALUES ($1, 'late', 'late', $1, true) RETURNING id`, customer)
	}

	_, err := svc.Resolve(ctx, agent, newTicket(), "فوق السقف", 4000, "127.0.0.1")
	var he *httpx.AppError
	if !errors.As(err, &he) || he.Code != "compensation_over_cap" {
		t.Fatalf("**الدعمُ اقترح فوق سقف الشكوى** (سقفُ المحفظة اليدويّة حكم): %v", err)
	}
	if _, err := svc.Resolve(ctx, agent, newTicket(), "عند السقف", 3000, "127.0.0.1"); err != nil {
		t.Fatalf("عند السقف رُدّ: %v", err)
	}
	if got := svc.CompensationCap(ctx); got != 3000 {
		t.Fatalf("السقفُ المعروض %d", got)
	}
}
