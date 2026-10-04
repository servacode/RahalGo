package orders_test

// ══════════════════════════════════════════════════════════════════════
// **السائقُ يُخبَر حين يُؤخذ منه طلبُه** (٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// **قِيس حيّاً**: إلغاءٌ أو إعادةٌ إلى الطابور والطلبُ بيد سائق — **فلا
// إشعارَ له ولا سطرَ في صندوقه**، والتطبيقُ يقفز إلى «الطلبات» صامتاً.

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// inbox **مُبلِّغٌ يحفظ ما قيل لكلّ حساب** — العنوانَ والنصَّ والتطبيق.
type inbox struct {
	mu  sync.Mutex
	got map[string][]notifications.Input
}

func (b *inbox) Notify(_ context.Context, in notifications.Input) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.got == nil {
		b.got = map[string][]notifications.Input{}
	}
	b.got[in.UserID] = append(b.got[in.UserID], in)
}
func (b *inbox) NotifyRoles(context.Context, []string, notifications.Input) {}
func (b *inbox) NotifyOps(context.Context, notifications.Input)             {}
func (b *inbox) NotifyOpsTx(context.Context, dbtx.Querier, notifications.Input) (int, error) {
	return 1, nil
}
func (b *inbox) PublishToUsers(context.Context, dbtx.Querier, []string, notifications.Input) {}

func (b *inbox) of(user string) []notifications.Input {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]notifications.Input(nil), b.got[user]...)
}

func lostNotice(t *testing.T, b *inbox, driver string) notifications.Input {
	t.Helper()
	for _, in := range b.of(driver) {
		if in.Kind == notifications.KindOrder && in.Title == "طلبٌ لم يعد معك" {
			return in
		}
	}
	t.Fatalf("لم يُخبَر السائقُ بأنّ طلبَه أُخذ منه — وصله: %+v", b.of(driver))
	return notifications.Input{}
}

func TestDriverLost_OpsRequeueTellsTheDriver(t *testing.T) {
	f := setup(t, "assigned", 100_000, 10_000, 0)
	ctx := context.Background()
	b := &inbox{}
	f.svc.SetNotifier(b)
	ops := testdb.NewUser(t, f.pool, "operations")

	if _, err := f.svc.Transition(ctx, ops, []string{"ops"}, f.orderID, "dispatching", "سائقٌ أقرب"); err != nil {
		t.Fatalf("تعذّرت الإعادة: %v", err)
	}
	in := lostNotice(t, b, f.driver)
	if !strings.Contains(in.Body, "أعادته الإدارة إلى الطابور") || !strings.HasPrefix(in.Body, "#") {
		t.Errorf("النصّ %q — **رقمُ الطلب وسببُه**", in.Body)
	}
	if len(in.Apps) != 1 || in.Apps[0] != notifications.AppDriver || in.Silent || in.Transient {
		t.Errorf("الإشعار %+v — **تطبيقُ السائق، يرنّ ويُحفَظ**", in)
	}

	out, err := f.svc.DriverOutcomeOf(ctx, f.orderID, f.driver)
	if err != nil {
		t.Fatal(err)
	}
	if out.Reason != orders.LossRequeuedOps || out.Message == "" {
		t.Errorf("المآل %+v — **التطبيقُ يقرأ منه لماذا اختفى الطلب**", out)
	}
}

func TestDriverLost_AdminCancelTellsTheDriver(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	b := &inbox{}
	f.svc.SetNotifier(b)
	admin := testdb.NewUser(t, f.pool, "admin")

	if _, err := f.svc.Transition(ctx, admin, []string{"admin"}, f.orderID, "cancelled", "طلب الزبون بالهاتف"); err != nil {
		t.Fatalf("تعذّر الإلغاء: %v", err)
	}
	if in := lostNotice(t, b, f.driver); !strings.Contains(in.Body, "ألغت الإدارة الطلب") {
		t.Errorf("النصّ %q", in.Body)
	}
	out, err := f.svc.DriverOutcomeOf(ctx, f.orderID, f.driver)
	if err != nil {
		t.Fatal(err)
	}
	if out.Reason != orders.LossCancelledOps || out.Status != "cancelled" {
		t.Errorf("المآل %+v", out)
	}
}

// TestDriverLost_HisOwnReleaseIsSilent **ولا يُخبَر بفعله هو.**
func TestDriverLost_HisOwnReleaseIsSilent(t *testing.T) {
	f := setup(t, "assigned", 100_000, 10_000, 0)
	ctx := context.Background()
	b := &inbox{}
	f.svc.SetNotifier(b)

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "dispatching", "تعطّلت"); err != nil {
		t.Fatalf("تعذّرت الإعادة: %v", err)
	}
	for _, in := range b.of(f.driver) {
		if in.Title == "طلبٌ لم يعد معك" {
			t.Fatalf("أُخبر بما فعله للتوّ: %+v", in)
		}
	}
	out, err := f.svc.DriverOutcomeOf(ctx, f.orderID, f.driver)
	if err != nil {
		t.Fatal(err)
	}
	if out.Reason != "" {
		t.Errorf("المآل %+v — **إعادتُه لا خبرَ فيها**", out)
	}
}

// TestDriverLost_MerchantBlockedOutcome **قرارُ المكتب بتبديل المتجر يُقال للسائق** — برنّة،
// **فهو لم يُنهِ شيئاً بيده** (٢٠٢٦-١٠-٠٣: السائقُ يُبلّغ وينتظر، والمكتبُ يقرّر).
func TestDriverLost_MerchantBlockedOutcome(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)
	b := &inbox{}
	f.svc.SetNotifier(b)

	if _, err := officeStoreBlock(t, f.svc, f.pool, f.orderID, "merchant_closed"); err != nil {
		t.Fatalf("تعذّر البلاغ: %v", err)
	}
	rang := false
	for _, in := range b.of(f.driver) {
		if in.Title == "طلبٌ لم يعد معك" {
			rang = true
		}
	}
	if !rang {
		t.Fatal("قرّر المكتبُ ولم يعلم السائقُ الواقفُ عند المتجر أنّ الطلبَ عاد")
	}
	out, err := f.svc.DriverOutcomeOf(ctx, f.orderID, f.driver)
	if err != nil {
		t.Fatal(err)
	}
	if out.Reason != orders.LossMerchantBlocked {
		t.Errorf("المآل %+v — **عاد إلى الإدارة لتبديل المتجر**", out)
	}
}

// TestDriverLost_StrangerGetsNotFound **ولا يُجاب عن طلبٍ لم يكن بيده.**
func TestDriverLost_StrangerGetsNotFound(t *testing.T) {
	f := setup(t, "assigned", 100_000, 10_000, 0)
	other := testdb.NewUser(t, f.pool, "driver")
	if _, err := f.svc.DriverOutcomeOf(context.Background(), f.orderID, other); err == nil {
		t.Fatal("أُجيب سائقٌ غريبٌ عن طلبٍ لم يكن بيده")
	}
}

func TestDriverLossCode_Table(t *testing.T) {
	cases := []struct {
		to, endedBy string
		byHim, sys  bool
		want        string
	}{
		{"cancelled", "customer", false, false, orders.LossCancelledCustomer},
		{"cancelled", "merchant", false, false, orders.LossCancelledMerchant},
		{"cancelled", "admin", false, false, orders.LossCancelledOps},
		{"dispatching", "", false, true, orders.LossRequeuedSystem},
		{"dispatching", "", false, false, orders.LossRequeuedOps},
		{"dispatching", "", true, false, ""},
		{"failed", "ops", false, false, orders.LossFailedOps},
		{"failed", "driver", true, false, ""},
		{"accepted", "", true, false, orders.LossMerchantBlocked},
		{"delivered", "driver", true, false, ""},
	}
	for _, c := range cases {
		if got := orders.DriverLossCode(c.to, c.endedBy, c.byHim, c.sys); got != c.want {
			t.Errorf("%+v → %q", c, got)
		}
		if c.want != "" && orders.LossText(c.want) == "" {
			t.Errorf("رمزٌ بلا جملة: %s", c.want)
		}
	}
}
