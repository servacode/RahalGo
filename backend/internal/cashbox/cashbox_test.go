package cashbox_test

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// صندوق السائق يحرس نقد المنصة بحوزته: التحصيل عند التسليم، والتسليم للمالية،
// والسقف الذي يوقف إسناد طلبات نقدية جديدة.

func newService(t *testing.T) (*cashbox.Service, context.Context, string) {
	t.Helper()
	pool := testdb.Pool(t)
	driver := testdb.NewUser(t, pool, "driver")
	return cashbox.NewService(pool, settings.NewStore(pool)), context.Background(), driver
}

func TestCollectThenSettle(t *testing.T) {
	svc, ctx, driver := newService(t)

	if err := svc.Collect(ctx, driver, 60_000, "order-1", nil); err != nil {
		t.Fatalf("التحصيل فشل: %v", err)
	}
	held, err := svc.Held(ctx, driver)
	if err != nil {
		t.Fatalf("تعذّرت قراءة الصندوق: %v", err)
	}
	if held != 60_000 {
		t.Fatalf("بحوزته %d، والمتوقع 60000", held)
	}

	// تسليم جزئي للمالية
	held, err = svc.Settle(ctx, driver, 20_000, "تسليم جزئي", driver)
	if err != nil {
		t.Fatalf("التسوية فشلت: %v", err)
	}
	if held != 40_000 {
		t.Fatalf("بعد التسوية %d، والمتوقع 40000", held)
	}

	st, err := svc.StatementFor(ctx, driver, 10)
	if err != nil {
		t.Fatalf("تعذّر كشف الصندوق: %v", err)
	}
	if len(st.Entries) != 2 {
		t.Fatalf("عدد القيود %d، والمتوقع 2", len(st.Entries))
	}
}

func TestSettle_RejectsMoreThanHeld(t *testing.T) {
	svc, ctx, driver := newService(t)

	if err := svc.Collect(ctx, driver, 10_000, "order-1", nil); err != nil {
		t.Fatalf("التحصيل فشل: %v", err)
	}
	// تسليم أكثر مما بحوزته: لا صندوق سالب
	if _, err := svc.Settle(ctx, driver, 10_001, "", driver); !errors.Is(err, cashbox.ErrOverSettle) {
		t.Fatalf("التسوية الزائدة لم تُرفض: %v", err)
	}
	held, _ := svc.Held(ctx, driver)
	if held != 10_000 {
		t.Fatalf("الصندوق تأثّر بتسوية مرفوضة: %d", held)
	}
}

func TestOverLimit_BlocksAtCeiling(t *testing.T) {
	svc, ctx, driver := newService(t)

	over, err := svc.OverLimit(ctx, driver)
	if err != nil {
		t.Fatalf("تعذّر فحص السقف: %v", err)
	}
	if over {
		t.Fatal("سائق بصندوق فارغ اعتُبر متجاوزاً للسقف")
	}

	// نبلغ السقف تماماً — عندها يجب أن يتوقف إسناد الطلبات النقدية
	limit := svc.Limit(ctx)
	if err := svc.Collect(ctx, driver, limit, "order-big", nil); err != nil {
		t.Fatalf("التحصيل فشل: %v", err)
	}
	over, err = svc.OverLimit(ctx, driver)
	if err != nil {
		t.Fatalf("تعذّر فحص السقف: %v", err)
	}
	if !over {
		t.Fatalf("بلغ السقف (%d) ولم يُعتبر متجاوزاً", limit)
	}
}

func TestCollect_IgnoresNonPositive(t *testing.T) {
	svc, ctx, driver := newService(t)

	// طلب مدفوع بالكامل من المحفظة: لا نقد يُحصَّل ولا قيد يُكتب
	if err := svc.Collect(ctx, driver, 0, "order-wallet", nil); err != nil {
		t.Fatalf("تحصيل صفر يجب أن يمرّ بلا أثر: %v", err)
	}
	st, _ := svc.StatementFor(ctx, driver, 10)
	if len(st.Entries) != 0 {
		t.Fatalf("قيد كُتب لتحصيل صفر: %d", len(st.Entries))
	}
}
