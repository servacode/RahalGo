package orders

import "testing"

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to string
		roles    []string
		want     bool
	}{
		{StPending, StAccepted, []string{"merchant"}, true},
		{StPending, StAccepted, []string{"ops"}, true},
		{StPending, StAccepted, []string{"customer"}, false},
		{StPending, StCancelled, []string{"customer"}, true},
		{StPending, StDelivered, []string{"admin"}, false}, // قفز غير معرف حتى للأدمن
		{StAccepted, StPreparing, []string{"merchant"}, true},
		{StPreparing, StDispatching, []string{"ops"}, true},
		{StPreparing, StDispatching, []string{"merchant"}, false},
		{StDispatching, StAssigned, []string{"driver"}, true},
		{StAtDropoff, StDelivered, []string{"driver"}, true},
		{StAtDropoff, StFailed, []string{"driver"}, true},
		{StDelivered, StRefunded, []string{"admin"}, true},
		{StDelivered, StRefunded, []string{"ops"}, false},
		{StDelivered, StPending, []string{"admin"}, false},
		{StCancelled, StAccepted, []string{"admin"}, false}, // لا عودة من نهائية
	}
	for _, c := range cases {
		if got := canTransition(c.from, c.to, c.roles); got != c.want {
			t.Errorf("canTransition(%s→%s, %v) = %v, want %v", c.from, c.to, c.roles, got, c.want)
		}
	}
}

func TestTerminal(t *testing.T) {
	for _, st := range []string{StDelivered, StRejected, StCancelled, StFailed, StRefunded} {
		if !terminal(st) {
			t.Errorf("terminal(%s) should be true", st)
		}
	}
	for _, st := range []string{StPending, StPreparing, StOnTheWay} {
		if terminal(st) {
			t.Errorf("terminal(%s) should be false", st)
		}
	}
}
