package server

// **التحويلُ التلقائيُّ — مفتاحٌ واحدٌ وحارسٌ فوقه.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «الطلبات يوجد خيارٌ تلقائيٌّ وخيارٌ يدويّ».)
//
// # ولماذا يُحرَس المُطفأ
//
// **المفتاحُ مُطفأٌ افتراضاً** — ومنصّةٌ تُبلّغ المتاجرَ من تلقاء نفسها قبل
// أن يُهيَّأ البوتُ **تُرسل إلى الفراغ**: يُوسَم الطلبُ مُحوَّلاً ولا أحدَ
// يعلم به.
//
// **وشرطٌ يُعكس بسطرٍ يُضاف بعد شهر** — والعطبُ لا يظهر في شاشةٍ، إنّما في
// سائقٍ يقف أمام مطبخٍ لم يسمع بالطلب.
//
// # ولماذا لا يُختبر المسارُ كلُّه هنا
//
// **`autoTransfer` تنادي القاعدةَ والمُرسِل** — والمُختبَرُ هو **القرار**:
// أيمضي أم يقف؟ **وقرارٌ يُتّخذ قبل أيّ نداءٍ يُقاس وحدَه.**

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestAutoTransfer_OffByDefault **مُطفأٌ ما لم يُشعله المالك.**
//
// **ويُقاس من الفهرس لا من القاعدة**: `GetBool` تردّ الصفَّ المخزَّن إن وُجد،
// **فصفٌّ كتبه اختبارٌ آخرُ يُخفي الافتراضَ ويجعل الحارسَ يمرّ دائماً.**
// (وقع فعلاً: زُرع الافتراضُ `true` فلم يسقط هذا الاختبار.)
func TestAutoTransfer_OffByDefault(t *testing.T) {
	def, ok := settings.Lookup("orders.auto_transfer")
	if !ok {
		t.Fatalf("المفتاحُ ليس في الفهرس")
	}
	if def.Default != false {
		t.Fatalf("افتراضُ التحويل التلقائيّ %v — **فتُبلَّغ المتاجرُ قبل أن يجهز البوت**", def.Default)
	}
}

// TestAutoTransfer_ToggleIsReadable **ويُشعل ويُطفأ من الإعدادات.**
//
// **والمفتاحُ يُقرأ بـ`GetBool`** — وهي التي يناديها `autoTransfer`.
// **ومفتاحٌ يُكتب بنوعٍ ويُقرأ بآخرَ يُقرأ مُطفأً أبداً** — وقد وقع في هذه
// المنصّة من قبل (`GetInt` على قيمةٍ منطقيّة).
func TestAutoTransfer_ToggleIsReadable(t *testing.T) {
	pool := testdb.Pool(t)
	store := settings.NewStore(pool)
	ctx := context.Background()

	if err := store.Set(ctx, "orders.auto_transfer", true, nil); err != nil {
		t.Fatalf("تعذّر إشعالُ المفتاح: %v", err)
	}
	if !store.GetBool(ctx, "orders.auto_transfer") {
		t.Fatalf("أُشعل المفتاحُ ويُقرأ مُطفأً — **فلا يُحوَّل طلبٌ أبداً ولا خطأَ يقول لماذا**")
	}

	if err := store.Set(ctx, "orders.auto_transfer", false, nil); err != nil {
		t.Fatalf("تعذّر إطفاءُ المفتاح: %v", err)
	}
	if store.GetBool(ctx, "orders.auto_transfer") {
		t.Fatalf("أُطفئ المفتاحُ ويُقرأ مُشتعلاً — **فيُحوَّل ما أراد المالكُ أن يبقى بيده**")
	}
}

// TestAutoTransfer_OldKeysAreGone **ولا بقيّةَ للعتبتين.**
//
// **ومفتاحٌ يبقى في الفهرس بعد أن مات يُعرض في شاشة الإعدادات** — فيضبطه
// المالكُ ولا يفعل شيئاً. **وهي عائلةُ «الحدّ الأدنى للطلب» بعينها.**
func TestAutoTransfer_OldKeysAreGone(t *testing.T) {
	for _, k := range []string{
		"orders.auto_transfer_min_total",
		"orders.auto_transfer_min_items",
	} {
		if _, ok := settings.Lookup(k); ok {
			t.Fatalf("المفتاحُ %q ما زال في الفهرس — **يُضبط ولا يُقرأ**", k)
		}
	}
	if _, ok := settings.Lookup("orders.auto_transfer"); !ok {
		t.Fatalf("المفتاحُ الجديدُ ليس في الفهرس — **فلا يظهر في شاشة الإعدادات أصلاً**")
	}
}
