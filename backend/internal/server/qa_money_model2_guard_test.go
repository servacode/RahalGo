package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **حرّاسُ جسر شهادة النموذج ٢** — تُقاس من المصدر فلا يتّسع الجسرُ لاحقاً
// ══════════════════════════════════════════════════════════════════════
//
// **يُثبَت**: (١) كلُّ مسارٍ جديدٍ داخلَ بوّابة التجهيز وحدَها ⇒ ٤٠٤ في الإنتاج. (٢) أفعالُ المكتب
// خلف حاقن الفاعل وحارس «طلبُ QA وحدَه». (٣) المعالِجاتُ الإنتاجيّةُ عينُها. (٤) قائمةُ الرايات
// فيها مفتاحٌ ماليٌّ واحدٌ لا غير. (٥) الوسائطُ والمعالِجاتُ تسقط مغلقةً خارجَ التجهيز وقتَ الطلب.

// m2BridgeRoutes **مساراتُ الجسر كما في `server.go`** — وكلُّ واحدٍ بحارسه.
var m2BridgeRoutes = []string{
	`r.Get("/qa/money", s.handleQAMoneyReadback)`,
	`r.With(s.qaOrderScope).Post("/qa/orders/{id}/transition", s.qaOrderTransition)`,
	`r.With(s.qaOrderScope).Post("/qa/orders/{id}/door-resolution", s.handleDoorResolution)`,
	`r.With(s.qaOrderScope).Post("/qa/orders/{id}/goods", s.handleGoods)`,
	`r.With(s.qaOrderScope).Post("/qa/orders/{id}/stuck-goods", s.handleStuckGoodsDecision)`,
	`r.With(s.qaOrderScope).Post("/qa/orders/{id}/edit", s.handleOrderEdit)`,
	`r.With(s.qaOrderScope).Post("/qa/orders/{id}/edit-request/approve", s.handleAdminApproveEditRequest)`,
	`r.With(s.qaOrderScope).Post("/qa/orders/{id}/edit-request/reject", s.handleAdminRejectEditRequest)`,
	`r.With(s.qaOrderScope).Post("/qa/orders/{id}/proof-exception", s.handleAuthorizeProofException)`,
	`r.With(s.qaOrderScope).Post("/qa/orders/{id}/partner", s.handleLinkPartner)`,
	`r.Post("/qa/whatsapp-orders", s.qaWhatsAppGuard(s.idempotent(s.handleCreateWhatsAppOrder)))`,
	`r.With(s.qaDriverScope).Post("/qa/drivers/{id}/settle", s.idempotent(s.handleDriverSettle))`,
}

func TestM2Bridge_RoutesInsideStagingGateBehindActor(t *testing.T) {
	src, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)
	gateOpen := strings.Index(s, "if s.qaStagingEnabled() {")
	gateEnd := strings.Index(s, "QA staging endpoints ENABLED")
	if gateOpen < 0 || gateEnd <= gateOpen {
		t.Fatal("**تعذّر تحديدُ حدود بوّابة التجهيز**")
	}
	// **ومجموعتُها تبدأ بحاقن الفاعل وقدراته** — فلا فعلَ مكتبٍ بلا فاعلٍ في التدقيق.
	group := strings.Index(s, "جسرُ شهادةِ النموذج ٢ الحيّة")
	if group < gateOpen || group > gateEnd {
		t.Fatal("**مجموعةُ النموذج ٢ خارجَ بوّابة التجهيز**")
	}
	block := s[group:gateEnd]
	for _, mw := range []string{"r.Use(s.qaAdminActor)", "r.Use(s.qaAdminAllCaps)", "r.Use(s.announceWrites)"} {
		if !strings.Contains(block, mw) {
			t.Errorf("**مجموعةُ النموذج ٢ بلا %s**", mw)
		}
	}
	for _, route := range m2BridgeRoutes {
		i := strings.Index(s, route)
		if i < 0 {
			t.Errorf("**مسارٌ غيرُ مسجَّل**: %s", route)
			continue
		}
		if i < gateOpen || i > gateEnd {
			t.Errorf("**مسارٌ خارجَ بوّابة التجهيز** (فيظهر في الإنتاج): %s", route)
		}
		if strings.Count(s, route) != 1 {
			t.Errorf("**مسارٌ مسجَّلٌ أكثرَ من مرّة**: %s", route)
		}
	}
	// **والانتقالُ يمرّ بالمعالِج الإنتاجيّ عينِه** — الغلافُ سطرٌ واحدٌ لا منطقَ فيه.
	if m2, err := os.ReadFile("qa_money_model2.go"); err != nil ||
		!strings.Contains(string(m2), "func (s *Server) qaOrderTransition(w http.ResponseWriter, r *http.Request) {\n\ts.handleOrderTransition(w, r)\n}") {
		t.Error("**qaOrderTransition must be a one-line call to handleOrderTransition**")
	}
	// **وكلُّ مسارٍ بمعرّف طلبٍ تحت `/qa/orders/` خلف حارس «طلبُ QA وحدَه».**
	for _, line := range strings.Split(block, "\n") {
		if strings.Contains(line, `"/qa/orders/{id}/`) && !strings.Contains(line, "r.With(s.qaOrderScope)") {
			t.Errorf("**فعلُ مكتبٍ على طلبٍ بلا qaOrderScope**: %s", strings.TrimSpace(line))
		}
	}
}

// TestM2Bridge_SettingAllowlistOneMoneyKey **مفتاحٌ ماليٌّ واحدٌ في قائمة الرايات** — والقائمةُ محصورة.
func TestM2Bridge_SettingAllowlistOneMoneyKey(t *testing.T) {
	want := map[string]bool{
		"launch.customer_signup": true, "launch.customer_browse": true, "launch.customer_orders": true,
		"launch.customer_custom_orders": true, "launch.merchant_orders": true, "hours.platform_enforced": true,
		"finance.goods_outside": true,
	}
	if len(qaFlagAllowlist) != len(want) {
		t.Fatalf("qaFlagAllowlist size = %d, want %d — the flag allowlist must stay bounded", len(qaFlagAllowlist), len(want))
	}
	money := 0
	for k := range qaFlagAllowlist {
		if !want[k] {
			t.Errorf("unexpected flag in allowlist: %s", k)
		}
		if strings.HasPrefix(k, "finance.") || strings.Contains(k, "commission") || strings.Contains(k, "fee") {
			money++
		}
	}
	if money != 1 {
		t.Errorf("money keys in the flag allowlist = %d, want exactly 1 (finance.goods_outside, owner 2026-11-10)", money)
	}
	if !qaSeedAllowlist["m2_store_setup"] || !qaStateSeed["m2_store_setup"] {
		t.Error("m2_store_setup must be an allowlisted state seed")
	}
}

// TestM2Bridge_FailClosedOutsideStaging **الوسائطُ والمعالِجاتُ الجديدةُ تسقط ٤٠٤ خارجَ التجهيز** —
// ولو سُجّلت خطأً، ولو حُقن العَلَم في الإنتاج. **والمعالِجُ الداخليّ لا يُنادى.**
func TestM2Bridge_FailClosedOutsideStaging(t *testing.T) {
	t.Setenv("RAHALGO_STAGING", "1")
	for _, env := range []string{"production", "test"} {
		s := newFaultServer(t, env)
		called := false
		inner := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
		for name, h := range map[string]http.Handler{
			"qaOrderScope":     s.qaOrderScope(inner),
			"qaDriverScope":    s.qaDriverScope(inner),
			"qaAdminAllCaps":   s.qaAdminAllCaps(inner),
			"qaAdminActor":     s.qaAdminActor(inner),
			"qaWhatsAppGuard":  s.qaWhatsAppGuard(inner),
			"qaMoneyReadback":  http.HandlerFunc(s.handleQAMoneyReadback),
			"qaStagingSetting": http.HandlerFunc(s.handleQAStagingSetting),
			"qaStagingSeed":    http.HandlerFunc(s.handleQAStagingSeed),
		} {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/qa/x?order_id=00000000-0000-0000-0000-000000000000",
				strings.NewReader(`{"key":"finance.goods_outside","value":true,"kind":"m2_store_setup"}`)))
			if w.Code != http.StatusNotFound {
				t.Errorf("%s on %s: status %d, want 404", name, env, w.Code)
			}
		}
		if called {
			t.Errorf("%s: the inner handler ran outside staging", env)
		}
	}
}

// TestM2Bridge_StdLadder **سُلّمُ العاديّ صاعدٌ ومطابق** — والتحضيرُ بين القبول والطابور.
func TestM2Bridge_StdLadder(t *testing.T) {
	prev := -1
	for _, st := range []string{"pending", "accepted", "dispatching", "assigned", "at_pickup", "picked_up",
		"on_the_way", "at_dropoff", "delivered"} {
		i := qaStdIndex(st)
		if i <= prev {
			t.Fatalf("ladder not strictly ascending at %s", st)
		}
		prev = i
	}
	if qaStdIndex("preparing") != qaStdIndex("accepted") || qaStdIndex("failed") != -1 {
		t.Fatal("preparing sits with accepted; terminal off-ladder states are -1")
	}
	if !qaRowMentions([]any{int64(1), [16]byte{0x12, 0x34}}, "12340000-0000-0000-0000-000000000000") {
		t.Fatal("a raw uuid cell must match its text form")
	}
}
