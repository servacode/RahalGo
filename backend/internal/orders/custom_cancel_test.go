package orders

// **إلغاءُ الطلب الخاصّ — لصاحبه إلى أن يُشترى، ثمّ لا.**
//
// (شهده المالك ٢٠٢٦-٠٨-١٠: «الإلغاءُ لم يُطبَّق على الطلبات الخاصّة بنفس
//  الأسلوب — لا يوجد إلغاءُ طلب».)
//
// # ولماذا حتّى الشراء لا أقلّ ولا أكثر
//
// **قبل ضغطة «اشتريتُ الطلب» لا مالَ خرج من جيب أحد** — والإلغاءُ يكلّف
// السائقَ وقتاً. **والمنعُ يكلّفه بضاعة**: من مُنع من الإلغاء لا يقبل الطلبَ
// عند الباب، **إنّما يرفضه بعد أن اشترى السائقُ بماله.** **فالمنعُ لا يحمي
// السائقَ، إنّما يؤخّر الرفضَ إلى ما بعد الخسارة.**
//
// **وبعد الشراء لا إلغاءَ لصاحبه** — العملياتُ وحدَها، **ومن ألغى بعده ترك
// بضاعةً في يدٍ دفعت ثمنَها.**
//
// # والحكمُ في الخارطة لا في الشاشة
//
// **الشاشةُ تعرض ما يقبله المحرّك** — **وشرطان يفترقان يجعلان زرّاً يُضغط
// فيُردّ**، أو باباً مفتوحاً في المحرّك لا يراه أحد.

import "testing"

// TestCustomCancel_OwnerUntilBought **بابُ الإلغاء يُفتح ويُقفل في موضعه.**
func TestCustomCancel_OwnerUntilBought(t *testing.T) {
	customer := []string{"customer"}

	for _, c := range []struct {
		name string
		from string
		want bool
	}{
		// **قبل موافقة الإدارة** — ولا أحدَ رآه بعد.
		{"معلَّق", StPending, true},
		// **وفي الطابور** — ولا سائقَ التزم.
		{"في الطابور", StDispatching, true},
		// **وبيد سائقٍ يتّفق** — ولم يشترِ بعد.
		{"أُسند لسائق", StAssigned, true},
		// **وبعد الشراء لا** — مالُ السائق خرج.
		{"اشترى السائق", StPickedUp, false},
		{"في الطريق", StOnTheWay, false},
		{"عند الباب", StAtDropoff, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := canTransition(KindCustom, c.from, StCancelled, customer)
			if got != c.want {
				t.Fatalf("من %q: الإلغاءُ لصاحبه = %v والمنتظَر %v", c.from, got, c.want)
			}
		})
	}
}

// TestCustomCancel_OpsAlwaysMay **والعملياتُ تُلغي في كلّ طورٍ حيّ.**
//
// **ومن اشترى ثمّ وقع له طارئٌ يحتاج من يُقفل طلبَه** — وإلّا بقي معلّقاً
// في مهامّه إلى الأبد.
func TestCustomCancel_OpsAlwaysMay(t *testing.T) {
	ops := []string{"ops"}
	for _, from := range []string{
		StPending, StDispatching, StAssigned, StPickedUp, StOnTheWay, StAtDropoff,
	} {
		if !canTransition(KindCustom, from, StCancelled, ops) {
			t.Fatalf("من %q: العملياتُ لا تستطيع الإلغاء — **وطلبٌ لا يُقفل يبقى في مهامّ سائقه**", from)
		}
	}
}

// TestCustomCancel_DriverMayNot **ولا يُلغيه السائق.**
//
// **له أن يفكّ إسنادَه فيعود إلى الطابور** — أمّا الإلغاءُ فقرارُ صاحب الطلب
// أو العمليات. **ومن ألغى طلباً لأنّه تعذّر عليه حرم غيرَه من أخذه.**
func TestCustomCancel_DriverMayNot(t *testing.T) {
	driver := []string{"driver"}
	for _, from := range []string{StDispatching, StAssigned, StPickedUp} {
		if canTransition(KindCustom, from, StCancelled, driver) {
			t.Fatalf("من %q: السائقُ يُلغي الطلبَ — **وهو ليس صاحبَه**", from)
		}
	}
}
