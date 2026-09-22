package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/config"
)

// TestQAFaultStore — تسليحٌ حتميّ: يُؤخذ بعددِ الإصابات ثمّ يُنزَع تلقائيّاً.
func TestQAFaultStore(t *testing.T) {
	st := &qaFaultStore{m: map[string]qaFault{}}
	if st.armed() {
		t.Fatal("default must be OFF (empty)")
	}
	st.arm("/p", qaFaultError5xx, 0, 2)
	if !st.armed() {
		t.Fatal("armed after arm")
	}
	if _, ok := st.take("/other"); ok {
		t.Fatal("unrelated path must not fault")
	}
	if _, ok := st.take("/p"); !ok {
		t.Fatal("hit 1 must fault")
	}
	if _, ok := st.take("/p"); !ok {
		t.Fatal("hit 2 must fault")
	}
	if _, ok := st.take("/p"); ok {
		t.Fatal("hit 3 must NOT fault (auto-disarm after count)")
	}
	if st.armed() {
		t.Fatal("must auto-disarm to empty")
	}
	// clear-all
	st.arm("/a", qaFaultLatency, 100, 1)
	st.arm("/b", qaFaultError5xx, 0, 1)
	st.clear("")
	if st.armed() {
		t.Fatal("clear-all must empty the store")
	}
}

func newFaultServer(t *testing.T, env string) *Server {
	t.Helper()
	return &Server{
		cfg:    &config.Config{Env: env},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// TestQAFaultFailClosedInProduction — الإنتاجُ لا يُفعّل الحاقنَ أبداً،
// ولو كان مسلَّحاً وحمل الطلبُ الترويسة: يمرّ بلا حقنٍ ولا استهلاك.
func TestQAFaultFailClosedInProduction(t *testing.T) {
	qaFaults.clear("")
	qaFaults.arm("/api/v1/orders", qaFaultError5xx, 0, 1)
	defer qaFaults.clear("")

	os.Setenv("RAHALGO_STAGING", "1") // even with the flag, production Env must fail closed
	defer os.Unsetenv("RAHALGO_STAGING")

	s := newFaultServer(t, "production")
	r := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
	r.Header.Set("X-QA-Fault", "1")
	w := httptest.NewRecorder()

	if s.qaMaybeFault(w, r) {
		t.Fatal("PRODUCTION must never inject a fault")
	}
	if _, ok := qaFaults.take("/api/v1/orders"); !ok {
		t.Fatal("fault must remain armed (production did not consume it)")
	}
}

// TestQAFaultStagingInjectsForQAHeader — على التجهيز، طلبٌ يحمل الترويسة
// على مسارٍ مسلَّحٍ يُحقَن ٥٠٣ ويُستهلَك.
func TestQAFaultStagingInjectsForQAHeader(t *testing.T) {
	qaFaults.clear("")
	defer qaFaults.clear("")
	os.Setenv("RAHALGO_STAGING", "1")
	defer os.Unsetenv("RAHALGO_STAGING")

	s := newFaultServer(t, "staging")
	qaFaults.arm("/api/v1/public/quote", qaFaultError5xx, 0, 1)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/public/quote", nil)
	r.Header.Set("X-QA-Fault", "1")
	w := httptest.NewRecorder()
	if !s.qaMaybeFault(w, r) {
		t.Fatal("staging + armed + QA header must inject")
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", w.Code)
	}
	if qaFaults.armed() {
		t.Fatal("one-shot fault must auto-disarm after the hit")
	}
}

// TestQAFaultNotScopedWithoutQA — على التجهيز، طلبٌ بلا ترويسةٍ ولا زبونِ QA
// لا يُحقَن (لا أثرَ على غير QA).
func TestQAFaultNotScopedWithoutQA(t *testing.T) {
	qaFaults.clear("")
	qaFaults.arm("/api/v1/public/quote", qaFaultError5xx, 0, 1)
	defer qaFaults.clear("")
	os.Setenv("RAHALGO_STAGING", "1")
	defer os.Unsetenv("RAHALGO_STAGING")

	s := newFaultServer(t, "staging")
	r := httptest.NewRequest(http.MethodPost, "/api/v1/public/quote", nil) // no header, no user
	w := httptest.NewRecorder()
	if s.qaMaybeFault(w, r) {
		t.Fatal("non-QA request must not be faulted")
	}
	if !qaFaults.armed() {
		t.Fatal("fault must remain armed (not consumed by non-QA)")
	}
}
