package orders

import "testing"

func line(id, note string, price int64, qty int, opts ...string) OrderItem {
	it := OrderItem{MenuItemID: &id, Name: "برغر", UnitPrice: price, Qty: qty, Note: note}
	for _, o := range opts {
		it.Options = append(it.Options, OptionSnapshot{ID: o})
	}
	return it
}

// TestMergeSameFolds **سطران لصنفٍ واحدٍ يُدمجان.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «مكتوب برغر برغر، وهي لازم برغر ×٢».)
func TestMergeSameFolds(t *testing.T) {
	got := mergeSame([]OrderItem{
		line("a", "", 200, 1),
		line("a", "", 200, 1),
	})
	if len(got) != 1 {
		t.Fatalf("لم يُدمج السطران: %d", len(got))
	}
	if got[0].Qty != 2 {
		t.Fatalf("العددُ لم يُجمع: %d", got[0].Qty)
	}
}

// TestMergeKeepsDifferences **«برغر بلا بصل» غيرُ «برغر».**
//
// **ومن دمجهما أسقط طلبَ صاحبه** — فيصله ما لم يطلب.
func TestMergeKeepsDifferences(t *testing.T) {
	cases := [][]OrderItem{
		{line("a", "", 200, 1), line("a", "بلا بصل", 200, 1)},    // ملاحظةٌ تفرّق
		{line("a", "", 200, 1), line("b", "", 200, 1)},           // صنفان
		{line("a", "", 200, 1), line("a", "", 250, 1)},           // سعرٌ يفرّق
		{line("a", "", 200, 1, "x"), line("a", "", 200, 1, "y")}, // إضافةٌ تفرّق
		{line("a", "", 200, 1), line("a", "", 200, 1, "x")},      // بإضافةٍ وبلا
	}
	for i, in := range cases {
		if got := mergeSame(in); len(got) != 2 {
			t.Errorf("الحالة %d: دُمج ما لا يُدمج → %d", i, len(got))
		}
	}
}

// TestMergeSumsThenKeepsOrder **والترتيبُ يبقى** — أوّلُ ما أُضيف أوّل.
func TestMergeSameSumsThree(t *testing.T) {
	got := mergeSame([]OrderItem{
		line("a", "", 200, 1),
		line("b", "", 300, 2),
		line("a", "", 200, 3),
	})
	if len(got) != 2 || got[0].Qty != 4 || got[1].Qty != 2 {
		t.Fatalf("جمعٌ أو ترتيبٌ خاطئ: %+v", got)
	}
}
