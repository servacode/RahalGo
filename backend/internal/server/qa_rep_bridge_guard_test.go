package server

import (
	"os"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **حرّاسُ جسورِ شهادةِ المندوب** — تُقاس من المصدر فلا يتّسع نطاقُها لاحقاً
// ══════════════════════════════════════════════════════════════════════
//
// **يُثبَت**: (١) كلُّ جسرٍ مسجَّلٌ داخلَ بوّابة `qaStagingEnabled` وحدَها ⇒ ٤٠٤
// في الإنتاج. (٢) أفعالُ الأدمن تمرّ بالمعالِجات الإنتاجيّة عينِها — لا منطقَ
// عملٍ بديل. (٣) لا SQL خامٌّ على جداول العمل في الجسر. (٤) لا توكنَ أدمنٍ
// يُصدَر — الحقنُ في السياق فقط. (٥) دفاعٌ في العمق ورفضُ الدور المُمتاز.

// TestRepBridge_GateUnit **البوّابةُ نفسُها**: التجهيزُ بالعَلَم يُفعّل، والإنتاج
// يسقط مغلقاً ولو حُقن العَلَم.
func TestRepBridge_GateUnit(t *testing.T) {
	os.Setenv("RAHALGO_STAGING", "1")
	defer os.Unsetenv("RAHALGO_STAGING")
	if newFaultServer(t, "production").qaStagingEnabled() {
		t.Fatal("production must fail closed even with RAHALGO_STAGING=1")
	}
	if !newFaultServer(t, "staging").qaStagingEnabled() {
		t.Fatal("staging + RAHALGO_STAGING=1 must enable the rep bridges")
	}
	os.Unsetenv("RAHALGO_STAGING")
	if newFaultServer(t, "staging").qaStagingEnabled() {
		t.Fatal("staging without the flag must be disabled")
	}
}

// repBridgeRoutes **مساراتُ الجسور الخمسة** — كما تظهر في `server.go`.
var repBridgeRoutes = []string{
	`r.Post("/qa/rep-session", s.handleQARepSession)`,
	`r.Post("/qa/leads/{id}/status", s.handleAdminLeadStatus)`,
	`r.Post("/qa/payouts/{id}/decide", s.handleDecidePayout)`,
	`r.Patch("/qa/users/{id}", s.handleAdminUpdateUser)`,
	`r.Patch("/qa/merchants/{id}", s.handleUpdateMerchant)`,
}

// TestRepBridge_RegisteredInsideStagingGateOnly **كلُّ مسارٍ داخلَ `if
// s.qaStagingEnabled()` وحدَه** — بين فتحِ البوّابة وسطرِ التمكين الذي يختمها.
func TestRepBridge_RegisteredInsideStagingGateOnly(t *testing.T) {
	src, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatalf("read server.go: %v", err)
	}
	s := string(src)
	gateOpen := strings.Index(s, "if s.qaStagingEnabled() {")
	gateEnd := strings.Index(s, "QA staging endpoints ENABLED") // آخرُ سطرٍ في البوّابة
	if gateOpen < 0 || gateEnd < 0 || gateEnd <= gateOpen {
		t.Fatalf("**تعذّر تحديدُ حدود بوّابة qaStagingEnabled** (open=%d end=%d)", gateOpen, gateEnd)
	}
	for _, route := range repBridgeRoutes {
		i := strings.Index(s, route)
		if i < 0 {
			t.Errorf("**مسارٌ غيرُ مسجَّل**: %s", route)
			continue
		}
		if i < gateOpen || i > gateEnd {
			t.Errorf("**مسارٌ خارجَ بوّابة التجهيز** (فيظهر في الإنتاج): %s", route)
		}
	}
	// **وأفعالُ الأدمن خلف حاقنِ الفاعل** — لا تُشغَّل بلا `qaAdminActor`.
	if !strings.Contains(s, "r.Use(s.qaAdminActor)") {
		t.Error("**مجموعةُ أفعال الأدمن بلا qaAdminActor** — لا فاعلَ حقيقيٌّ في التدقيق")
	}
}

// TestRepBridge_ReusesRealHandlersNoFakeLogic **أفعالُ الأدمن تستدعي المعالِجات
// الإنتاجيّة عينَها** — لا نسخةَ منطقٍ بديلة (شرطُ المالك الصريح).
func TestRepBridge_ReusesRealHandlersNoFakeLogic(t *testing.T) {
	// المعالِجاتُ الحقيقيّةُ الأربعةُ يستعملها الجسرُ نصّاً.
	real := []string{
		"s.handleAdminLeadStatus", // تحويلُ المرشَّح (convertLead)
		"s.handleDecidePayout",    // قرارُ السحب
		"s.handleAdminUpdateUser", // إيقاف/تفعيل
		"s.handleUpdateMerchant",  // نقلُ المتجر
	}
	src, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatalf("read server.go: %v", err)
	}
	for _, h := range real {
		if !strings.Contains(string(src), "/qa/") { // sanity
			break
		}
		if !strings.Contains(string(src), h) {
			t.Errorf("**الجسرُ لا يعيد استعمالَ المعالِج الإنتاجيّ %s**", h)
		}
	}
}

// TestRepMoneySeed_RegisteredAndBounded **بذّارُ مالِ المندوب مسجَّلٌ حالةً،
// وقائمةُ مفاتيحه محصورةٌ بالثمانية** — لا يتّسع إلى إعدادٍ آخر (كنسخةِ النسخة
// أو حدِّ النسخة).
func TestRepMoneySeed_RegisteredAndBounded(t *testing.T) {
	if !qaSeedAllowlist["rep_money_set"] {
		t.Fatal("rep_money_set must be in qaSeedAllowlist")
	}
	if !qaStateSeed["rep_money_set"] {
		t.Fatal("rep_money_set must be a state seed (no QA customer uid needed)")
	}
	want := map[string]bool{
		"sales.commission_percent":     true,
		"merchants.commission_percent": true,
		"pricing.margin_fixed":         true,
		"sales.commission_source":      true,
		"sales.activation_orders":      true,
		"sales.monthly_target":         true,
		"sales.target_reward":          true,
		"payouts.min_amount":           true,
	}
	if len(qaRepMoneyKeys) != len(want) {
		t.Fatalf("qaRepMoneyKeys size = %d, want %d — the money key allowlist must stay bounded", len(qaRepMoneyKeys), len(want))
	}
	for k := range want {
		if !qaRepMoneyKeys[k] {
			t.Errorf("qaRepMoneyKeys missing expected key %q", k)
		}
	}
	for k := range qaRepMoneyKeys {
		if !want[k] {
			t.Errorf("qaRepMoneyKeys carries UNEXPECTED key %q — the money setter must not reach beyond rep money", k)
		}
	}
	// **ولا مفتاحَ خطيرٍ يتسلّل**: حدُّ النسخة ليس مالاً، ولا يُضبَط من هنا.
	for _, forbidden := range []string{"app.min_version.rep", "app.min_version.customer"} {
		if qaRepMoneyKeys[forbidden] {
			t.Errorf("qaRepMoneyKeys must NOT contain %q — min_version is not a money key", forbidden)
		}
	}
}

// TestRepBridge_NoRawBusinessLogicNoAdminToken **الجسرُ لا يصنع حالةً خاماً ولا
// يُصدر توكنَ أدمن.**
func TestRepBridge_NoRawBusinessLogicNoAdminToken(t *testing.T) {
	src, err := os.ReadFile("qa_rep_bridge.go")
	if err != nil {
		t.Fatalf("read qa_rep_bridge.go: %v", err)
	}
	s := string(src)

	// (١) لا كتابةَ عملٍ خامّة — الإدراجُ/التحديثُ/الحذفُ على جداول العمل ممنوع.
	//     القراءةُ (SELECT) لهويّةٍ مسموحة.
	for _, sqlWrite := range []string{"INSERT INTO", "DELETE FROM", "UPDATE merchants", "UPDATE wallets", "UPDATE payout_requests", "UPDATE user_roles"} {
		if strings.Contains(s, sqlWrite) {
			t.Errorf("**SQL كتابةٍ خامٌّ في الجسر (%q)** — يجب أن يمرّ بالمعالِج الإنتاجيّ", sqlWrite)
		}
	}

	// (٢) دفاعٌ في العمق: البوّابةُ في جلسة المندوب وفي حاقنِ الأدمن كليهما.
	if n := strings.Count(s, "if !s.qaStagingEnabled() {"); n < 2 {
		t.Errorf("**حارسُ qaStagingEnabled ناقصٌ**: وجد %d من 2 (جلسة المندوب + حاقن الأدمن)", n)
	}

	// (٣) حاقنُ الأدمن يحقن فاعلاً حقيقيّاً في السياق — لا توكن.
	if !strings.Contains(s, "context.WithValue(r.Context(), ctxUserID, adminID)") {
		t.Error("**qaAdminActor لا يحقن أدمنَ QA فاعلاً في السياق**")
	}

	// (٤) لا يُصدَر توكنٌ إلّا في جلسة المندوب — والإصدارُ الوحيدُ في المعالِج
	//     `handleQARepSession` (لا في حاقن الأدمن).
	if c := strings.Count(s, "IssueForUserID"); c != 1 {
		t.Errorf("**إصدارُ التوكن يجب أن يكون مرّةً واحدةً (جلسة المندوب) — وجد %d**", c)
	}
	repSessionIdx := strings.Index(s, "func (s *Server) handleQARepSession")
	adminActorIdx := strings.Index(s, "func (s *Server) qaAdminActor")
	issueIdx := strings.Index(s, "IssueForUserID")
	if repSessionIdx < 0 || adminActorIdx < 0 || issueIdx < 0 {
		t.Fatal("**تعذّر تحديدُ مواضع الدوال**")
	}
	// **الإصدارُ في جلسة المندوب لا في حاقن الأدمن** — بين رأسِ رأسِ الأولى ورأسِ الثانية.
	if !(issueIdx > repSessionIdx && issueIdx < adminActorIdx) {
		t.Error("**إصدارُ التوكن خارجَ handleQARepSession** — لا يُصدَر توكنُ أدمنٍ من باب الاختبار")
	}

	// (٥) رفضُ الدور المُمتاز في جلسة المندوب.
	if !strings.Contains(s, "qaCarriesPrivileged(ctx, uid)") {
		t.Error("**جلسةُ المندوب بلا حارسِ الدور المُمتاز**")
	}
}
