package identity

import (
	"context"
	"testing"
)

type countSender struct{ n int }

func (c *countSender) SendOTP(context.Context, string, string) error { c.n++; return nil }

// TestQANoSendSkipsOnlyHookedPhones — رقمُ التصوير لا يُرسَل له، وغيرُه يُرسَل،
// وبلا خطّافٍ (الإنتاج) يُرسَل للكلّ.
func TestQANoSendSkipsOnlyHookedPhones(t *testing.T) {
	cs := &countSender{}
	s := &Service{sender: cs}
	ctx := context.Background()
	_ = s.sendCode(ctx, "+963900555770", "1234")
	if cs.n != 1 {
		t.Fatalf("without hook every code must be sent, got %d", cs.n)
	}
	s.SetQANoSend(func(p string) bool { return p == "+963900555770" })
	_ = s.sendCode(ctx, "+963900555770", "1234")
	if cs.n != 1 {
		t.Fatal("hooked video phone must not be sent")
	}
	_ = s.sendCode(ctx, "+963911111111", "1234")
	if cs.n != 2 {
		t.Fatal("other phones must still be sent")
	}
}
