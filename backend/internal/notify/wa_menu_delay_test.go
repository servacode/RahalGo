package notify

import (
	"testing"
	"time"
)

// TestBotTypingDelay_Bounds **«يكتب…» بين ثانيةٍ وثلاث** (قرارُ المالك ٢٠٢٦-١١-٢٧).
func TestBotTypingDelay_Bounds(t *testing.T) {
	if d := botTypingDelay(0, 0); d != time.Second {
		t.Fatalf("empty text, no jitter = %v, want 1s", d)
	}
	if d := botTypingDelay(5000, 1); d != 3*time.Second {
		t.Fatalf("long text = %v, want capped at 3s", d)
	}
	short, long := botTypingDelay(20, 0.5), botTypingDelay(200, 0.5)
	if !(short < long) || short < time.Second || long > 3*time.Second {
		t.Fatalf("delays not growing within bounds: short=%v long=%v", short, long)
	}
}
