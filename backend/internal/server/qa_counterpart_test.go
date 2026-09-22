package server

import (
	"os"
	"testing"
)

// TestQACounterpartKindsRegistered — الأنواعُ الجديدةُ مسجَّلةٌ في السماح،
// وتصنيفُها (بحاجةٍ لهويّة زبون QA أو لا) صحيحٌ فتُوجَّه في المكان الصائب.
func TestQACounterpartKindsRegistered(t *testing.T) {
	for _, k := range []string{"order_advance", "order_chat_send", "zone_close", "zone_reopen", "min_version"} {
		if !qaSeedAllowlist[k] {
			t.Fatalf("%s must be in qaSeedAllowlist", k)
		}
	}
	// دوامُ المنطقة والحدُّ الأدنى للنسخة لا تلزمها هويّةُ زبون QA ⇒ state seeds.
	for _, k := range []string{"zone_close", "zone_reopen", "min_version"} {
		if !qaStateSeed[k] {
			t.Fatalf("%s must be a state seed (no QA uid needed)", k)
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
