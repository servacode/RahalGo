package server

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// TestInbox_FilteredByApp **صندوقُ كلِّ تطبيقٍ بما يخصّه** (فحصُ ٢٠٢٦-١٠-٢٩) — صاحبُ المتجر
// حسابٌ واحد، **وكان يرى في تطبيق متجره ترويجَ الزبائن.**
func TestInbox_FilteredByApp(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	notif := notifications.New(f.pool, f.srv.hub, f.srv.logger)
	uid := f.drivers[0]

	notif.Notify(ctx, notifications.Input{UserID: uid, Kind: notifications.KindAccount,
		Title: "ترويج للزبون", Apps: []string{notifications.AppCustomer}})
	notif.Notify(ctx, notifications.Input{UserID: uid, Kind: notifications.KindAccount,
		Title: "خبر للمتجر", Apps: []string{notifications.AppMerchant}})
	notif.Notify(ctx, notifications.Input{UserID: uid, Kind: notifications.KindAccount,
		Title: "خبر للجميع"})

	titles := func(app string) map[string]bool {
		items, _, err := notif.List(ctx, uid, 200, "", app)
		if err != nil {
			t.Fatalf("القراءة: %v", err)
		}
		out := map[string]bool{}
		for _, n := range items {
			out[n.Title] = true
		}
		return out
	}

	m := titles(notifications.AppMerchant)
	if m["ترويج للزبون"] || !m["خبر للمتجر"] || !m["خبر للجميع"] {
		t.Fatalf("**صندوقُ المتجر**: %v — ترويجُ الزبون لا يظهر فيه، وخبرُه والعامُّ يظهران", m)
	}
	c := titles(notifications.AppCustomer)
	if !c["ترويج للزبون"] || c["خبر للمتجر"] || !c["خبر للجميع"] {
		t.Fatalf("**صندوقُ الزبون**: %v", c)
	}
	all := titles("")
	if len(all) < 3 {
		t.Fatalf("**بلا تطبيقٍ يظهر الكلّ** كما كان: %v", all)
	}
	if got := broadcastApps([]string{"customer", "sales"}); len(got) != 2 ||
		got[0] != notifications.AppCustomer || got[1] != notifications.AppRep {
		t.Fatalf("**تطبيقاتُ الإعلان**: %v", got)
	}
}
