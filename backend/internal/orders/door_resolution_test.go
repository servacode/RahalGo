package orders_test

// ══════════════════════════════════════════════════════════════════════
// **عند باب الزبون السائقُ يُبلّغ والإدارةُ تُنهي** (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// «ويبقى الطلبُ مع السائق إلى أن تُحلّ القصّة… وقتها الإدارةُ هي تُنهي الطلبَ
// من عندها، يصل أمرٌ للسائق — مثلاً العودة إلى المكتب.» **وكلُّ اختبارٍ هنا
// يسقط على الشيفرة القديمة**: كان السائقُ يُغلق الطلبَ عند الباب بضغطة.

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// failAtDoor **المكتبُ يُنهي عند الباب: عُد إلى المكتب بذنبٍ يكتبه.**
func (f *fixture) failAtDoor(t *testing.T, fault, reason string) {
	t.Helper()
	endAtDoor(t, f.svc, f.pool, f.orderID, fault, reason)
}

// endAtDoor **كـ`failAtDoor` لأيّ عُدّة** — المحرّكُ والقاعدةُ ومعرّفُ الطلب.
func endAtDoor(t *testing.T, svc *orders.Service, pool *pgxpool.Pool, orderID, fault, reason string) {
	t.Helper()
	ops := testdb.NewUser(t, pool, "operations")
	if _, err := svc.ResolveDoor(context.Background(), ops, []string{"ops"}, orderID,
		orders.DoorResolution{Action: orders.DoorReturnToOffice, Fault: fault,
			Reason: reason, Note: "قرارُ المكتب بعد الاتّصال بالزبون"}, nil); err != nil {
		t.Fatalf("إنهاءُ المكتب رُدّ: %v", err)
	}
}

// TestDoor_DriverCannotEndOrder **السائقُ لا يُنهي الطلبَ عند الباب — بأيّ سبب.**
func TestDoor_DriverCannotEndOrder(t *testing.T) {
	for _, reason := range []string{"customer_refused", "address_wrong", "customer_absent", "driver_late", ""} {
		t.Run("r="+reason, func(t *testing.T) {
			f := setup(t, "at_dropoff", 100_000, 10_000, 0)
			ctx := context.Background()
			f.armTreasury(t)
			_, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
				f.orderID, "failed", "لم يستلم", reason)
			if err == nil {
				t.Fatalf("أغلق السائقُ الطلبَ عند الباب بـ«%s» — **والإدارةُ هي التي تُنهي**", reason)
			}
			if st := f.statusOf(t); st != "at_dropoff" {
				t.Fatalf("الحالُ %q — **والطلبُ يبقى مع السائق إلى أن تُحلّ القصّة**", st)
			}
		})
	}
}

// TestDoor_GenericOpsTransitionRefused **ولا انتقالُ اللوحة العامّ** — بلا ذنبٍ يُكتب.
func TestDoor_GenericOpsTransitionRefused(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()
	ops := testdb.NewUser(t, f.pool, "operations")
	_, err := f.svc.Transition(ctx, ops, []string{"ops"}, f.orderID, "failed", "تعذّر")
	if !errors.Is(err, orders.ErrDoorNeedsOps) {
		t.Fatalf("انتقالُ اللوحة العامّ أنهى الطلبَ عند الباب (%v) — **بلا ذنبٍ لا يُعرف على من الخسارة**", err)
	}
	if st := f.statusOf(t); st != "at_dropoff" {
		t.Fatalf("الحالُ %q", st)
	}
}

// TestDoor_ReturnToOffice_FaultDecidesCompensation **الذنبُ يكتبه المكتب — ومنه
// التعويض:** الزبونُ والمتجرُ والمنصّةُ ⇒ طلبٌ معلَّق · السائقُ ⇒ لا شيء.
func TestDoor_ReturnToOffice_FaultDecidesCompensation(t *testing.T) {
	cases := []struct {
		fault   string
		pending bool
	}{
		{orders.FaultCustomer, true},
		{orders.FaultMerchant, true},
		{orders.FaultPlatform, true},
		{orders.FaultDriver, false},
	}
	for _, c := range cases {
		t.Run(c.fault, func(t *testing.T) {
			f := setup(t, "at_dropoff", 100_000, 10_000, 0)
			ctx := context.Background()
			f.armTreasury(t)
			before := f.balance(t, f.driver)
			f.failAtDoor(t, c.fault, "customer_refused")

			var status, fault, reason, instr string
			if err := f.pool.QueryRow(ctx, `
				SELECT status, COALESCE(fault,''), COALESCE(fail_reason,''), door_instruction
				FROM orders WHERE id = $1`, f.orderID).Scan(&status, &fault, &reason, &instr); err != nil {
				t.Fatal(err)
			}
			if status != "failed" || fault != c.fault || reason != "customer_refused" ||
				instr != orders.DoorReturnToOffice {
				t.Fatalf("(%s · %s · %s · %s) والمتوقّع (failed · %s · customer_refused · %s)",
					status, fault, reason, instr, c.fault, orders.DoorReturnToOffice)
			}
			if got := f.balance(t, f.driver) - before; got != 0 {
				t.Errorf("قُيّد للسائق %d — **والتعويضُ بعد موافقة العمليات**", got)
			}
			gotFault, suggested, found := f.pendingRequest(t)
			if found != c.pending {
				t.Fatalf("طلبُ تعويضٍ معلَّق = %v والمتوقّع %v", found, c.pending)
			}
			if found && (gotFault != c.fault || suggested != 5_000) {
				t.Errorf("الطلبُ المعلَّق (%s · %d) والمتوقّع (%s · 5000)", gotFault, suggested, c.fault)
			}

			// **والسائقُ يقرأ الأمرَ من مخرَج طلبه** — ولو ضاع الإشعار.
			out, err := f.svc.DriverOutcomeOf(ctx, f.orderID, f.driver)
			if err != nil {
				t.Fatalf("المخرَج: %v", err)
			}
			if out.Reason != orders.LossReturnToOffice || out.DoorInstruction != orders.DoorReturnToOffice {
				t.Errorf("المخرَجُ (%q · %q) — والسائقُ لا يعرف أن يعود إلى المكتب", out.Reason, out.DoorInstruction)
			}
		})
	}
}

// TestDoor_DeliverNow_OrderStaysWithDriver **«سلّم الآن» — الطلبُ يبقى عند الباب
// معه، والأمرُ مكتوبٌ يقرؤه، ثمّ يُسلّمه كالعادة.**
func TestDoor_DeliverNow_OrderStaysWithDriver(t *testing.T) {
	f := setup(t, "on_the_way", 100_000, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)
	// **الوصولُ بالانتقال الحقيقيّ** — ومنه حدثٌ يحمل السائقَ يقرؤه المخرَج.
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "at_dropoff", ""); err != nil {
		t.Fatalf("الوصول: %v", err)
	}
	ops := testdb.NewUser(t, f.pool, "operations")
	if _, err := f.svc.ResolveDoor(ctx, ops, []string{"ops"}, f.orderID,
		orders.DoorResolution{Action: orders.DoorDeliverNow, Note: "الزبونُ نازل"}, nil); err != nil {
		t.Fatalf("«سلّم الآن» رُدّ: %v", err)
	}
	if st := f.statusOf(t); st != "at_dropoff" {
		t.Fatalf("الحالُ %q — **والطلبُ يبقى معه ليسلّمه**", st)
	}
	out, err := f.svc.DriverOutcomeOf(ctx, f.orderID, f.driver)
	if err != nil {
		t.Fatalf("المخرَج: %v", err)
	}
	if out.DoorInstruction != orders.DoorDeliverNow || out.DoorNote != "الزبونُ نازل" || out.Reason != "" {
		t.Errorf("المخرَجُ (%q · %q · %q) — والمتوقّعُ أمرُ «سلّم الآن» والطلبُ بيده",
			out.DoorInstruction, out.DoorNote, out.Reason)
	}
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليمُ بعد الأمر رُدّ: %v", err)
	}
}

// TestDoor_Validation **أمرٌ مجهول · عودةٌ بلا ذنبٍ أو بلا كلمة · طلبٌ ليس عند الباب.**
func TestDoor_Validation(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()
	ops := testdb.NewUser(t, f.pool, "operations")
	try := func(in orders.DoorResolution) error {
		_, err := f.svc.ResolveDoor(ctx, ops, []string{"ops"}, f.orderID, in, nil)
		return err
	}
	if err := try(orders.DoorResolution{Action: "cancel"}); !errors.Is(err, orders.ErrDoorBadAction) {
		t.Errorf("أمرٌ مجهول: %v", err)
	}
	if err := try(orders.DoorResolution{Action: orders.DoorReturnToOffice, Note: "x"}); !errors.Is(err, orders.ErrDoorBadFault) {
		t.Errorf("عودةٌ بلا ذنب: %v", err)
	}
	if err := try(orders.DoorResolution{Action: orders.DoorReturnToOffice, Fault: "customer"}); !errors.Is(err, orders.ErrDoorNeedsNote) {
		t.Errorf("عودةٌ بلا كلمة: %v", err)
	}
	if err := try(orders.DoorResolution{Action: orders.DoorReturnToOffice, Fault: "customer",
		Reason: "merchant_closed", Note: "x"}); !errors.Is(err, orders.ErrFailReasonStage) {
		t.Errorf("سببٌ من غير الباب: %v", err)
	}
	if st := f.statusOf(t); st != "at_dropoff" {
		t.Fatalf("تبدّلت الحالُ إلى %q مع الرفض", st)
	}

	// **وقبل السائق لا أمر** — وقبل المتجر صار للمكتب «أكمل الطلب» (قرارُ المالك ٢٠٢٦-١٠-٠٣).
	g := setup(t, "dispatching", 100_000, 10_000, 0)
	if _, err := g.svc.ResolveDoor(ctx, ops, []string{"ops"}, g.orderID,
		orders.DoorResolution{Action: orders.DoorDeliverNow}, nil); !errors.Is(err, orders.ErrNotAtDoor) {
		t.Errorf("أمرٌ لطلبٍ بلا سائق: %v", err)
	}
	h := setup(t, "assigned", 100_000, 10_000, 0)
	if _, err := h.svc.ResolveDoor(ctx, ops, []string{"ops"}, h.orderID,
		orders.DoorResolution{Action: orders.DoorReturnToOffice, Fault: "customer", Note: "x"}, nil); !errors.Is(err, orders.ErrNotAtDoor) {
		t.Errorf("«عُد إلى المكتب» قبل المتجر — ولا بضاعةَ معه: %v", err)
	}
}

// TestTrip_OnTheWay_OpsDecides **وفي الطريق الإدارةُ تقرّر كذلك** (قرارُ المالك
// ٢٠٢٦-١٠-٠٣): الزبونُ طلب الإلغاءَ من الدردشة والسائقُ ماشٍ — «أكمل» يبقي الطلبَ
// معه بأمرٍ مكتوب، و«عُد إلى المكتب» يُنهيه بذنبٍ وسببٍ من بلاغ الطريق.
func TestTrip_OnTheWay_OpsDecides(t *testing.T) {
	t.Run("continue", func(t *testing.T) {
		f := setup(t, "on_the_way", 100_000, 10_000, 0)
		ctx := context.Background()
		ops := testdb.NewUser(t, f.pool, "operations")
		if _, err := f.svc.ResolveDoor(ctx, ops, []string{"ops"}, f.orderID,
			orders.DoorResolution{Action: orders.DoorDeliverNow, Note: "الزبونُ تراجع"}, nil); err != nil {
			t.Fatalf("«أكمل» في الطريق رُدّ: %v", err)
		}
		if st := f.statusOf(t); st != "on_the_way" {
			t.Fatalf("الحالُ %q — **والطلبُ يبقى معه في الطريق**", st)
		}
		var instr string
		if err := f.pool.QueryRow(ctx, `SELECT door_instruction FROM orders WHERE id = $1`,
			f.orderID).Scan(&instr); err != nil {
			t.Fatal(err)
		}
		if instr != orders.DoorDeliverNow {
			t.Fatalf("الأمرُ %q — **والسائقُ يقرأ «أكمل» من طلبه**", instr)
		}
	})
	t.Run("return", func(t *testing.T) {
		f := setup(t, "on_the_way", 100_000, 10_000, 0)
		ctx := context.Background()
		f.armTreasury(t)
		endAtDoor(t, f.svc, f.pool, f.orderID, orders.FaultCustomer, "customer_cancelled_by_phone")
		var status, reason string
		if err := f.pool.QueryRow(ctx, `SELECT status, COALESCE(fail_reason,'') FROM orders WHERE id = $1`,
			f.orderID).Scan(&status, &reason); err != nil {
			t.Fatal(err)
		}
		if status != "failed" || reason != "customer_cancelled_by_phone" {
			t.Fatalf("(%s · %s) والمتوقّعُ (failed · customer_cancelled_by_phone)", status, reason)
		}
		out, err := f.svc.DriverOutcomeOf(ctx, f.orderID, f.driver)
		if err != nil {
			t.Fatalf("المخرَج: %v", err)
		}
		if out.Reason != orders.LossReturnToOffice {
			t.Errorf("المخرَجُ %q — **والبضاعةُ معه، فأمرُه العودةُ إلى المكتب**", out.Reason)
		}
	})
}

// TestMerchantDelivery_FailAtDoor_PlatformPays **فشلُ «لدي توصيلة» عند الزبون —
// السائقُ يُعوَّض من المنصّة** (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢، البند ٨).
func TestMerchantDelivery_FailAtDoor_PlatformPays(t *testing.T) {
	for _, c := range []struct {
		fault   string
		pending bool
	}{{orders.FaultCustomer, true}, {orders.FaultMerchant, true}, {orders.FaultDriver, false}} {
		t.Run(c.fault, func(t *testing.T) {
			f := setup(t, "at_dropoff", 0, 10_000, 0)
			ctx := context.Background()
			f.armTreasury(t)
			if _, err := f.pool.Exec(ctx, `
				UPDATE orders SET kind = 'merchant_delivery', recipient_name = 'مستلم',
				       recipient_phone = '0999000000', fee_payer = 'merchant'
				WHERE id = $1`, f.orderID); err != nil {
				t.Fatalf("تعذّر جعلُه توصيلة: %v", err)
			}
			f.failAtDoor(t, c.fault, "customer_absent")
			gotFault, suggested, found := f.pendingRequest(t)
			if found != c.pending {
				t.Fatalf("طلبُ تعويضٍ معلَّق = %v والمتوقّع %v — **«المنصّة تدفع طبعاً»**", found, c.pending)
			}
			if found && (gotFault != orders.FaultPlatform || suggested != 5_000) {
				t.Errorf("الطلبُ المعلَّق (%s · %d) والمتوقّع (platform · 5000) — **فلا تُفتح مطالبةٌ على المتجر**",
					gotFault, suggested)
			}
		})
	}
}
