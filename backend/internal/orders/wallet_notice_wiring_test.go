package orders_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// ══════════════════════════════════════════════════════════════════════
// **التسليمُ يُخبر السائقَ بأجره فعلاً — لا في الشيفرة وحدَها**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١١: «الرصيد يتغيّر وما حدا بيعرف ليش».)
//
// # ولماذا هذا الاختبارُ بجانب الذي قبله
//
// **`wallet_notice_test.go` يفحص أنّ المُرسِل يُرسل** — يبني حركاتٍ بيده
// ثمّ يسأل: أخرجت إشعاراً؟ **وهو يمرّ حتّى لو لم يُنادَ من مسار التسليم
// أصلاً.**
//
// **وقِيس ذلك**: حُذف سطرُ تسجيل أجر السائق من `payDriver` **فمرّ كلُّ
// شيء** — الحارسُ كان يحرس الدالّة، والعطبُ في ألّا تُنادى.
//
// **فيُسلَّم طلبٌ حقيقيٌّ ويُسأل الناقلُ ماذا وصله.**
type wiringSpy struct {
	mu   sync.Mutex
	sent []notifications.Input
}

func (n *wiringSpy) Notify(_ context.Context, in notifications.Input) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, in)
}
func (n *wiringSpy) NotifyRoles(context.Context, []string, notifications.Input) {}
func (n *wiringSpy) NotifyOps(context.Context, notifications.Input)             {}

func (n *wiringSpy) walletOf(userID string) *notifications.Input {
	n.mu.Lock()
	defer n.mu.Unlock()
	for i := range n.sent {
		if n.sent[i].UserID == userID && n.sent[i].Kind == notifications.KindWallet {
			return &n.sent[i]
		}
	}
	return nil
}

func TestDelivery_TellsDriverAboutEarning(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	spy := &wiringSpy{}
	f.svc.SetNotifier(spy)
	ctx := context.Background()

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليم فشل: %v", err)
	}

	got := f.walletNotice(t, spy, f.driver)
	if !strings.Contains(got.Body, "10000") {
		t.Fatalf("نصُّ الإشعار %q بلا مبلغ الأجر — **فيُفتح التطبيقُ ليُعرَف الرقم**", got.Body)
	}
	if len(got.Apps) != 1 || got.Apps[0] != notifications.AppDriver {
		t.Fatalf("جمهورُ الإشعار %v لا [driver] — **فيرنّ أجرُ التوصيل في تطبيق "+
			"الزبون عند السائق**", got.Apps)
	}
}

// TestDelivery_TellsMerchantAboutEarning **والمتجرُ يُخبَر بمستحقّه.**
func TestDelivery_TellsMerchantAboutEarning(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	spy := &wiringSpy{}
	f.svc.SetNotifier(spy)
	ctx := context.Background()

	// **ومتجرُ الميدان بلا صاحبٍ افتراضاً** — ومستحقُّ متجرٍ بلا صاحبٍ لا
	// يُقيَّد أصلاً (`sh.ownerID == nil`). فيُعيَّن له واحد.
	owner := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(ctx,
		`UPDATE merchants SET owner_user_id = $2 WHERE id = $1`, f.merchantID, owner); err != nil {
		t.Fatalf("تعيينُ صاحب المتجر: %v", err)
	}

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليم فشل: %v", err)
	}

	got := f.walletNotice(t, spy, owner)
	if len(got.Apps) != 1 || got.Apps[0] != notifications.AppMerchant {
		t.Fatalf("جمهورُ إشعار المتجر %v لا [merchant]", got.Apps)
	}
}

// walletNotice إشعارُ محفظةٍ لصاحبٍ بعينه — أو سقوطٌ برسالةٍ تقول ما ضاع.
func (f *fixture) walletNotice(t *testing.T, spy *wiringSpy, userID string) *notifications.Input {
	t.Helper()
	got := spy.walletOf(userID)
	if got == nil {
		t.Fatal("سُلّم الطلبُ وقُيّد المالُ ولم يصل صاحبَه إشعار — " +
			"**فيرى الرقمَ في شريطه يزيد ولا يعرف عمّاذا**، ولا يعرف إلّا إن فتح المحفظة")
	}
	return got
}
