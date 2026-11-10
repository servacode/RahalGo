package waitems

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

func (f *fx) storeOff(id string, off bool) {
	f.t.Helper()
	must(f.t, execOne(f, `UPDATE merchants SET wa_items_enabled = $2 WHERE id = $1::uuid`, id, !off))
}

func execOne(f *fx, q string, args ...any) error {
	_, err := f.pool.Exec(f.ctx, q, args...)
	return err
}

// المتجر مفعّل افتراضاً، والمفتاح العامّ مشغَّل: البوت يتولّى الصورة.
func TestSwitchDefaultOnUnchanged(t *testing.T) {
	f := newFx(t)
	f.svc.Enabled = func(context.Context) bool { return true }
	f.img(f.phone, "جبنة بيضاء 25000")
	f.flush()
	if f.count(f.merchant) != 1 {
		t.Fatal("الصنف لم يُضف والمفتاح مفعّل")
	}
}

// مطفأ عامّاً: لا يتولّى الصورة، ولا يلتقط نصّاً معلّقاً.
func TestSwitchGlobalOffIgnored(t *testing.T) {
	f := newFx(t)
	f.svc.Enabled = func(context.Context) bool { return false }
	if f.svc.Accepts(f.ctx, f.phone) {
		t.Fatal("Accepts يجب أن يكون false والمفتاح العامّ مطفأ")
	}
	if r := f.img(f.phone, "جبنة بيضاء 25000"); r != "" {
		t.Fatalf("ردّ: %q", r)
	}
	if got := f.flush(); len(got) != 0 {
		t.Fatalf("أُرسل شيء: %q", got)
	}
	if f.count(f.merchant) != 0 {
		t.Fatal("أُضيف صنف والمفتاح العامّ مطفأ")
	}
	if r, ok := f.svc.HandleText(f.ctx, f.phone, "فرع", ""); ok || r != "" {
		t.Fatalf("النصّ التُقط: %q %v", r, ok)
	}
}

// سؤال معلّق ثمّ إطفاء المتجر: الجواب لا يلتقطه البوت.
func TestSwitchStoreOffIgnored(t *testing.T) {
	f := newFx(t)
	f.img(f.phone, "جبنة بيضاء") // بلا سعر ⇒ سؤال معلّق
	f.flush()
	f.storeOff(f.merchant, true)
	if f.svc.Accepts(f.ctx, f.phone) {
		t.Fatal("Accepts يجب أن يكون false ومتجره مطفأ")
	}
	if r := f.img(f.phone, "لبنة 8000"); r != "" {
		t.Fatalf("ردّ: %q", r)
	}
	if got := f.flush(); len(got) != 0 {
		t.Fatalf("أُرسل شيء: %q", got)
	}
	if f.count(f.merchant) != 0 {
		t.Fatal("أُضيف صنف والمتجر مطفأ")
	}
	if r, ok := f.svc.HandleText(f.ctx, f.phone, "25000", ""); ok || r != "" {
		t.Fatalf("الجواب التُقط: %q %v", r, ok)
	}
	if f.count(f.merchant) != 0 {
		t.Fatal("الجواب أضاف صنفاً")
	}
}

// عدّة متاجر: يُعرض ما مفتاحه مفعّل فقط، وإن لم يبقَ شيء يمرّ.
func TestSwitchMixedStoresOnlyEnabledOffered(t *testing.T) {
	f := newFx(t)
	second := f.newStore(f.owner, "الفرع الثاني", "SYP", 0)
	f.declare(second, f.dairy)
	third := f.newStore(f.owner, "الفرع الثالث", "SYP", 0)
	f.declare(third, f.dairy)
	f.storeOff(second, true)

	f.img(f.phone, "جبنة بيضاء 25000")
	q := one(t, f.flush())
	has(t, q, "لأي فرع؟", "١ - متجر الاختبار", "٢ - الفرع الثالث")
	if contains(q, "الفرع الثاني") {
		t.Fatalf("عُرض متجر مطفأ: %s", q)
	}

	// تبقّى متجرٌ واحد مفعّل ⇒ بلا سؤال
	f2 := newFx(t)
	b := f2.newStore(f2.owner, "الفرع الثاني", "SYP", 0)
	f2.declare(b, f2.dairy)
	f2.storeOff(b, true)
	f2.img(f2.phone, "لبنة 8000")
	has(t, one(t, f2.flush()), "انضاف (لبنة)")
	if f2.count(f2.merchant) != 1 || f2.count(b) != 0 {
		t.Fatal("الصنف بالمتجر الخطأ")
	}

	// كلّها مطفأة ⇒ تمرّ
	f2.storeOff(f2.merchant, true)
	if f2.svc.Accepts(f2.ctx, f2.phone) {
		t.Fatal("Accepts يجب أن يكون false وكلّ المتاجر مطفأة")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// قرار المالك: المفتاح العامّ افتراضه مطفأ — الصورة تمرّ ولا يلتقطها البوت (والمتجر مفعّل افتراضاً).
func TestSwitchGlobalDefaultOffIgnores(t *testing.T) {
	f := newFx(t)
	f.svc.Enabled = func(ctx context.Context) bool {
		return settings.NewStore(f.pool).GetBool(ctx, "merchants.wa_items_enabled")
	}
	if f.svc.Accepts(f.ctx, f.phone) {
		t.Fatal("الافتراضي العامّ يجب أن يكون مطفأ")
	}
	if r := f.img(f.phone, "جبنة بيضاء 25000"); r != "" {
		t.Fatalf("ردّ: %q", r)
	}
	if f.count(f.merchant) != 0 || len(f.flush()) != 0 {
		t.Fatal("البوت عالج الصورة والمفتاح العامّ بافتراضه")
	}
	var on bool
	must(t, f.pool.QueryRow(f.ctx, `SELECT wa_items_enabled FROM merchants WHERE id = $1::uuid`, f.merchant).Scan(&on))
	if !on {
		t.Fatal("مفتاح المتجر يجب أن يكون مفعّلاً افتراضاً")
	}
	// المالك شغّل العامّ ⇒ يعمل لكل المتاجر
	must(t, execOne(f, `INSERT INTO app_settings (key, value) VALUES ('merchants.wa_items_enabled', 'true'::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = 'true'::jsonb`))
	if !f.svc.Accepts(f.ctx, f.phone) {
		t.Fatal("بعد التشغيل العامّ يجب أن يتولّى البوت")
	}
}
