package orders

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// noticeSvc خدمةٌ بقاعدةٍ حقيقيّة — **`notifyCredits` تقرأ رقمَ الطلب
// لتضعه في النصّ، ومجمّعٌ فارغٌ يُسقط النداء.**
func noticeSvc(t *testing.T) (*Service, *noticeSpy) {
	t.Helper()
	spy := &noticeSpy{}
	return &Service{db: testdb.Pool(t), notify: spy}, spy
}

// ══════════════════════════════════════════════════════════════════════
// **من تحرّكت محفظتُه يعرف عمّاذا**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١١: «الرصيد يتغيّر وما حدا بيعرف ليش».)
//
// **ما يُحرَس هنا ثلاثةٌ:**
//
// **الأوّل** أنّ الإشعارَ يُرسَل أصلاً — **والسائقُ كان يوصّل ويقبض فلا
// يصله شيء**، فيرى الرقمَ يزيد ولا يعرف عمّاذا.
//
// **والثاني** أنّه يذهب إلى التطبيق الصحيح — **صاحبُ المتجر يحمل تطبيقين**،
// ومستحقُّ مبيعاتِه يخصّ تطبيقَ متجرِه لا تطبيقَ الزبون.
//
// **والثالث** أنّ المبلغَ في نصّ الإشعار — **إشعارٌ يقول «تحرّكت محفظتُك»
// بلا رقمٍ يُجبر صاحبَه أن يفتح التطبيق ليعرف**، وهذا نقضٌ لغرضه.

// noticeSpy ناقلٌ يسجّل كلَّ ما أُرسل.
type noticeSpy struct {
	mu   sync.Mutex
	sent []notifications.Input
}

func (n *noticeSpy) Notify(_ context.Context, in notifications.Input) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, in)
}
func (n *noticeSpy) NotifyRoles(context.Context, []string, notifications.Input) {}
func (n *noticeSpy) NotifyOps(context.Context, notifications.Input)             {}

// TestWalletNotice_SentPerCredit **كلُّ حركةٍ تُجمَع تصير إشعاراً.**
func TestWalletNotice_SentPerCredit(t *testing.T) {
	svc, spy := noticeSvc(t)

	var done settled
	done.credit("driver-1", 2500, t2.driverEarned, notifications.AppDriver)
	done.credit("merchant-1", 18000, t2.merchantEarned, notifications.AppMerchant)
	// **والصفرُ لا يُسجَّل** — إشعارٌ بصفرٍ ضجيجٌ يُعلّم صاحبَه ألّا يقرأ.
	done.credit("driver-1", 0, t2.compensated, notifications.AppDriver)
	done.credit("", 900, t2.compensated, notifications.AppDriver)

	svc.notifyCredits(context.Background(), "order-x", done.credits)

	if len(spy.sent) != 2 {
		t.Fatalf("أُرسل %d إشعاراً من ٢ — **فمن تحرّكت محفظتُه لا يعرف عمّاذا**", len(spy.sent))
	}
}

// TestWalletNotice_GoesToOwnApp **ويذهب إلى تطبيق صاحبه.**
func TestWalletNotice_GoesToOwnApp(t *testing.T) {
	svc, spy := noticeSvc(t)

	var done settled
	done.credit("merchant-1", 18000, t2.merchantEarned, notifications.AppMerchant)
	svc.notifyCredits(context.Background(), "order-x", done.credits)

	if len(spy.sent) != 1 {
		t.Fatalf("أُرسل %d إشعاراً من ١", len(spy.sent))
	}
	got := spy.sent[0]
	if len(got.Apps) != 1 || got.Apps[0] != notifications.AppMerchant {
		t.Fatalf("الجمهورُ %v لا [merchant] — **فيرنّ مستحقُّ المبيعات في تطبيق "+
			"الزبون عند صاحب المتجر**", got.Apps)
	}
	if got.Kind != notifications.KindWallet {
		t.Fatalf("النوعُ %q لا wallet — **فيُرشَّح خطأً في صفحة الإشعارات**", got.Kind)
	}
}

// TestWalletNotice_CarriesAmount **والمبلغُ في النصّ.**
//
// **وإشعارٌ يقول «تحرّكت محفظتُك» بلا رقمٍ يُجبر صاحبَه أن يفتح التطبيق**
// — وهو بالضبط ما جاء هذا الإشعارُ ليُغنيَ عنه.
func TestWalletNotice_CarriesAmount(t *testing.T) {
	svc, spy := noticeSvc(t)

	var done settled
	done.credit("driver-1", 2500, t2.driverEarned, notifications.AppDriver)
	svc.notifyCredits(context.Background(), "order-x", done.credits)

	body := spy.sent[0].Body
	if !strings.Contains(body, "2500") {
		t.Fatalf("النصُّ %q بلا مبلغ — **فيُفتح التطبيقُ ليُعرَف الرقم**", body)
	}
	if !strings.Contains(body, "+") {
		t.Fatalf("النصُّ %q بلا إشارة — **ولا يُعرف أدخل المالُ أم خرج**", body)
	}
}

// TestWalletNotice_NegativeShowsMinus **والخارجُ بإشارته.**
func TestWalletNotice_NegativeShowsMinus(t *testing.T) {
	svc, spy := noticeSvc(t)

	var done settled
	done.credit("merchant-1", -5000, t2.merchantEarned, notifications.AppMerchant)
	svc.notifyCredits(context.Background(), "order-x", done.credits)

	body := spy.sent[0].Body
	if !strings.Contains(body, "−5000") {
		t.Fatalf("النصُّ %q — **وخصمٌ يُعرض موجباً يجعل صاحبَه يظنّ أنّه قبض**", body)
	}
}

// t2 اختصارٌ لعناوين الإشعارات — الحزمةُ نفسُها.
var t2 = t
