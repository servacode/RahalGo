package server

import (
	"os"
	"strings"
	"testing"
)

// TestQACounterpartKindsRegistered — الأنواعُ الجديدةُ مسجَّلةٌ في السماح،
// وتصنيفُها (بحاجةٍ لهويّة زبون QA أو لا) صحيحٌ فتُوجَّه في المكان الصائب.
func TestQACounterpartKindsRegistered(t *testing.T) {
	for _, k := range []string{"order_advance", "order_chat_send", "zone_close", "zone_reopen", "min_version", "driver_shift"} {
		if !qaSeedAllowlist[k] {
			t.Fatalf("%s must be in qaSeedAllowlist", k)
		}
	}
	// دوامُ المنطقة والحدُّ الأدنى للنسخة لا تلزمها هويّةُ زبون QA ⇒ state seeds.
	// وشاهدُ التوزيع (driver_shift) يدير هويّةَ سائق QA بنفسِه ⇒ state seed أيضاً.
	for _, k := range []string{"zone_close", "zone_reopen", "min_version", "driver_shift"} {
		if !qaStateSeed[k] {
			t.Fatalf("%s must be a state seed (no QA customer uid needed)", k)
		}
	}
	// سوقُ الطلب ورسالةُ المحادثة على طلبِ زبون QA ⇒ تلزمهما هويّتُه ⇒ ليست state.
	for _, k := range []string{"order_advance", "order_chat_send"} {
		if qaStateSeed[k] {
			t.Fatalf("%s must NOT be a state seed (needs QA customer uid for ownership)", k)
		}
	}
}

// TestQAOrderLadder — سُلّمُ الطلب صاعدٌ ومطابقٌ للحالات، والمجهولُ -1.
func TestQAOrderLadder(t *testing.T) {
	if qaLadderIndex("pending") != 0 {
		t.Fatal("pending must be index 0")
	}
	if qaLadderIndex("delivered") != len(qaOrderLadder)-1 {
		t.Fatal("delivered must be last")
	}
	if qaLadderIndex("nope") != -1 {
		t.Fatal("unknown status must be -1")
	}
	prev := -1
	for _, s := range []string{"dispatching", "assigned", "picked_up", "on_the_way", "at_dropoff", "delivered"} {
		i := qaLadderIndex(s)
		if i <= prev {
			t.Fatalf("ladder not strictly ascending at %s (idx %d, prev %d)", s, i, prev)
		}
		prev = i
	}
}

// TestQASeedFailsClosedInProduction — الحارسُ الحاكم: الإنتاجُ لا يُفعّل
// بذّارَ QA أبداً ولو حُقنت الرايةُ؛ التجهيزُ يلزمه العَلَمُ صراحةً.
func TestQASeedFailsClosedInProduction(t *testing.T) {
	os.Setenv("RAHALGO_STAGING", "1")
	defer os.Unsetenv("RAHALGO_STAGING")

	if newFaultServer(t, "production").qaStagingEnabled() {
		t.Fatal("production must fail closed even with RAHALGO_STAGING=1")
	}
	if !newFaultServer(t, "staging").qaStagingEnabled() {
		t.Fatal("staging + RAHALGO_STAGING=1 must enable")
	}
	os.Unsetenv("RAHALGO_STAGING")
	if newFaultServer(t, "staging").qaStagingEnabled() {
		t.Fatal("staging without the flag must be disabled")
	}
}

// TestQAVideoPhones — أرقامُ فيديو الشرح عشرةٌ في المدى الوهميّ، مسموحٌ لها
// إصدارُ الرمز، **وخطّافُ «لا إرسال» لا يُركَّب إلّا خلف `qaStagingEnabled`.**
func TestQAVideoPhones(t *testing.T) {
	if len(qaVideoPhones) != 10 {
		t.Fatalf("want 10 video phones, got %d", len(qaVideoPhones))
	}
	for _, p := range qaVideoPhones {
		if !strings.HasPrefix(p, "+963900555") || !qaOTPPhones[p] || !qaIsVideoPhone(p) {
			t.Fatalf("%s must be a fake-range OTP-allowed video phone", p)
		}
	}
	if qaIsVideoPhone("+963900555001") {
		t.Fatal("QA customer is not a video phone")
	}
	src, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)
	gate := strings.Index(s, "if s.qaStagingEnabled() {")
	hook := strings.Index(s, "s.identity.SetQANoSend(qaIsVideoPhone)")
	if gate < 0 || hook < gate || hook-gate > 300 || strings.Count(s, "SetQANoSend") != 1 {
		t.Fatal("SetQANoSend must be installed once, inside the qaStagingEnabled block")
	}
}
