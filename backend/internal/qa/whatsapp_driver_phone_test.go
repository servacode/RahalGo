package qa

// **ولا رقمَ لمستلِم طلب الواتساب عند السائق** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٧): «السائق يرجع
// لموضوع ما فيه ياخذ رقم أي زبون بشكل خارجي».

import (
	"testing"
	"time"
)

// TestWAORD021_DriverNeverGetsRecipientPhone **و · لا رقمَ لمستلِم طلب الواتساب عند السائق** — لا في
// الطابور ولا بعد الإسناد (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٧). **و«لدي توصيلة» كما هي.**
//
// **وعلى شيفرة المراحل ١–٥** كان الرقمُ يصل كلَّ سائقٍ يرى الطلب.
func TestWAORD021_DriverNeverGetsRecipientPhone(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-021")
	// **وضعُ «الطابور»** — يراه كلُّ قريبٍ مؤهَّل، لا من عُرض عليه وحدَه («بالترتيب»).
	h.Setting("drivers.assignment_mode", `"queue"`)
	nearby := w.md.f.Driver(OnShift(), LocationAt(w.storeLat, w.storeLng, time.Now()))
	c := waEngineCreate(t, h, w.md.admin.ID, w.typed())
	// **والطلبُ في طابور القريب فعلاً** — وإلّا فحصُ الرقم فيه لا يقيس شيئاً (مراجعةُ ٢٠٢٦-١١-٠٥).
	seen := false
	for _, o := range mdList(h.GET("/api/v1/driver/queue", nearby.Token)) {
		if m, _ := o.(map[string]any); m != nil && m["id"] == c.ID {
			seen = true
			if m["recipient_phone"] != nil && m["recipient_phone"] != "" {
				t.Fatalf("**رقمُ المستلِم وصل السائقَ في الطابور**: %v", m["recipient_phone"])
			}
		}
	}
	if !seen {
		t.Fatal("**طلبُ الواتساب لم يظهر في طابور السائق القريب** — فحصُ الطابور فارغ")
	}
	drv := h.driverOf(c.ID)
	got := driverOrderOf(t, h, drv.Token, c.ID)
	if got["recipient_phone"] != nil && got["recipient_phone"] != "" {
		t.Fatalf("**رقمُ المستلِم وصل السائقَ المُسنَد**: %v", got["recipient_phone"])
	}

	// **و«لدي توصيلة» يبقى رقمُها للسائق** (قرارُ المالك ٢٠٢٦-١٠-٠٣) — لا تتغيّر.
	md := w.md.create(t, h, "recipient", uniq("wa21"))
	mdDrv := h.NewUser("driver")
	assignDriver(t, h, md, mdDrv.ID)
	if m := driverOrderOf(t, h, mdDrv.Token, md); m["recipient_phone"] == nil || m["recipient_phone"] == "" {
		t.Fatalf("**«لدي توصيلة» فقدت رقمَ المستلِم**: %v", m)
	}
}
