package server

// **خطوةٌ ثبتت وضاع ردُّها** (٢٠٢٦-١٠-٠٢) — يُعيدها التطبيقُ بالمفتاح نفسِه فيُردّ
// عليه بالطلب كما هو، **لا بـ«انتقالٌ غيرُ جائز» على خطوةٍ وقعت.**

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func (f *driverFixture) transitionKeyed(orderID, driverID, body, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/driver/orders/"+orderID+"/transition", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", orderID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, driverID)
	ctx = context.WithValue(ctx, ctxRoles, []string{"driver"})
	w := httptest.NewRecorder()
	f.srv.handleDriverTransition(w, req.WithContext(ctx))
	return w
}

func TestDriverTransition_LostResponseRetryIsReplayed(t *testing.T) {
	f := newDriverFixture(t, 2)
	d, other := f.drivers[0], f.drivers[1]
	id := f.problemOrderAt(t, "at_pickup", d)

	if w := f.transitionKeyed(id, d, `{"to":"picked_up"}`, "k-1"); w.Code != http.StatusOK {
		t.Fatalf("الخطوةُ الأولى %d: %s", w.Code, w.Body.String())
	}
	// **والردُّ ضاع — فيُعاد بالمفتاح.**
	w := f.transitionKeyed(id, d, `{"to":"picked_up"}`, "k-1")
	if w.Code != http.StatusOK || w.Header().Get("Idempotent-Replay") != "true" {
		t.Fatalf("الإعادةُ %d (replay=%q): %s — **خطأٌ على خطوةٍ وقعت**",
			w.Code, w.Header().Get("Idempotent-Replay"), w.Body.String())
	}
	// **وبلا مفتاحٍ تبقى خطأً** — نسخةٌ قديمةٌ أو ضغطةٌ متعمَّدة.
	if w := f.transitionKeyed(id, d, `{"to":"picked_up"}`, ""); w.Code < 400 {
		t.Errorf("بلا مفتاح: %d — **الإعادةُ لمن طلبها**", w.Code)
	}
	// **ولا تُعاد خطوةُ غيره** — سائقٌ آخرُ لا يرث جوابَها.
	if w := f.transitionKeyed(id, other, `{"to":"picked_up"}`, "k-1"); w.Code < 400 {
		t.Errorf("سائقٌ غريبٌ أُجيب بإعادة: %d", w.Code)
	}
	// **وخطوةٌ لم تقع لا تُعاد** — «سلّمتُ» قبل الطريق.
	if w := f.transitionKeyed(id, d, `{"to":"delivered"}`, "k-2"); w.Code < 400 {
		t.Errorf("خطوةٌ لم تقع أُجيبت: %d", w.Code)
	}
}
