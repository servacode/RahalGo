package orders

import "testing"

func TestItemsCountAr(t *testing.T) {
	for n, want := range map[int]string{1: "صنف واحد", 2: "صنفان", 3: "3 أصناف", 10: "10 أصناف", 11: "11 صنفاً"} {
		if got := itemsCountAr(n); got != want {
			t.Fatalf("%d: %q want %q", n, got, want)
		}
	}
}
