package qa

// ناقلُ دفعٍ بديلٌ يُسجّل — **`P-7` البندان ١٥ و٣٠.**
//
// **ولا يقع إلّا في اختبار**: يُمرَّر بـ`server.WithPushTransport`،
// **والإنتاجُ لا يمرّر خياراً.**

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/servacode/rahalgo/backend/internal/push"
)

// PushOutcome ما يفعله الناقلُ في النداء التالي.
type PushOutcome string

const (
	PushSuccess   PushOutcome = "SUCCESS"
	PushHTTP500   PushOutcome = "HTTP_500"
	PushTimeout   PushOutcome = "TIMEOUT"
	PushConnReset PushOutcome = "CONNECTION_FAILURE"
	// PushDead المنصّةُ ترفض الرمزَ نهائيّاً — فيُحذَف.
	PushDead PushOutcome = "DEAD_TOKEN"
)

// PushCall نداءٌ واحدٌ كما وقع — **الدليلُ الذي يُستهلَك لاحقاً** (البند ٣٠).
type PushCall struct {
	At      time.Time
	Tokens  []string
	Title   string
	Body    string
	Data    map[string]string
	Urgent  bool
	Outcome PushOutcome
	Err     error
	Dead    []string
}

// FakePush ناقلٌ مبرمَجٌ يُسجّل.
type FakePush struct {
	mu      sync.Mutex
	calls   []PushCall
	script  []PushOutcome
	def     PushOutcome
	platfrm string
}

// NewFakePush ناقلٌ افتراضُه النجاح.
func NewFakePush() *FakePush {
	return &FakePush{def: PushSuccess, platfrm: "android"}
}

// Script يبرمج نتائجَ النداءات بالترتيب — **وما بعدها الافتراضُ.**
func (f *FakePush) Script(out ...PushOutcome) *FakePush {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.script = append(f.script, out...)
	return f
}

// Platform المنصّة.
func (f *FakePush) Platform() string { return f.platfrm }

// Send ينفّذ ما بُرمج ويُسجّل.
func (f *FakePush) Send(ctx context.Context, tokens []string, msg push.Message) ([]string, error) {
	f.mu.Lock()
	out := f.def
	if len(f.script) > 0 {
		out, f.script = f.script[0], f.script[1:]
	}
	call := PushCall{
		At: time.Now(), Tokens: append([]string(nil), tokens...),
		Title: msg.Title, Body: msg.Body, Urgent: msg.Urgent,
		Data: cloneMap(msg.Data), Outcome: out,
	}
	f.mu.Unlock()

	var dead []string
	var err error
	switch out {
	case PushSuccess:
	case PushHTTP500:
		err = errors.New("fake fcm: 500 internal server error")
	case PushTimeout:
		err = context.DeadlineExceeded
	case PushConnReset:
		err = &net.OpError{Op: "read", Err: errors.New("connection reset by peer")}
	case PushDead:
		dead = append([]string(nil), tokens...)
	}

	f.mu.Lock()
	call.Err, call.Dead = err, dead
	f.calls = append(f.calls, call)
	f.mu.Unlock()
	return dead, err
}

// Calls ما وقع.
func (f *FakePush) Calls() []PushCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]PushCall(nil), f.calls...)
}

// Reset يمسح السجلَّ والبرنامج.
func (f *FakePush) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls, f.script = nil, nil
}

// WaitCalls ينتظر بلوغَ عددٍ من النداءات — **الدفعُ في خيطٍ منفصل.**
func (f *FakePush) WaitCalls(n int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if len(f.Calls()) >= n {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func cloneMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
