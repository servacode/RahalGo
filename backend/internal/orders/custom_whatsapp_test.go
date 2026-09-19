package orders_test

// ══════════════════════════════════════════════════════════════════════
//  D8 — **توثيقُ واتساب يشمل الطلبَ الخاصَّ كما يشمل العاديّ**
// ══════════════════════════════════════════════════════════════════════
//
// (بوّابةُ العوائق · `CUSTOMER-ACCEPTANCE-MASTER.md` §40.8.)
//
// # العيب
//
// **شرطُ توثيق واتساب يُفحَص في مسار الطلب العاديّ وحدَه** (`CreateTx` →
// `RequireWhatsApp` → `ErrWhatsAppRequired`). **والطلبُ الخاصُّ
// (`POST /orders/custom`) بابٌ مفتوح**: حين يُشغّل المالكُ الشرطَ، من لم يوثّق
// رقمَه يُنشئ طلباً خاصّاً ويلتفّ على الحارس. **وكامنٌ اليومَ** لأنّ السياسةَ
// مطفأةٌ (`customers.require_whatsapp`=false افتراضاً) — فإن شُغّلت ظهر الفرق.
//
// # العقدُ بعد الإصلاح
//
// **السياسةُ واحدةٌ لبابين** — `RequireWhatsApp` (المفتاحُ العامُّ
// `auth.require_whatsapp` يعلو مفتاحَ الدور `customers.require_whatsapp`)،
// والرسالةُ نفسُها (`ErrWhatsAppRequired`). **والمطفأُ يبقى مطفأً**: حين تكون
// السياسةُ off لا يُخترَع شرطٌ لم يُرِده المالك.

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// setWhatsAppVerified **يضبط حالةَ التوثيق لزبون العُدّة صراحةً** — فلا يتّكئ
// الاختبارُ على ما قد يفعله المِسنَد.
func setWhatsAppVerified(t *testing.T, f *fixture, verified bool) {
	t.Helper()
	q := `UPDATE users SET whatsapp_verified_at = now() WHERE id = $1`
	if !verified {
		q = `UPDATE users SET whatsapp_verified_at = NULL WHERE id = $1`
	}
	if _, err := f.pool.Exec(context.Background(), q, f.customer); err != nil {
		t.Fatalf("تعذّر ضبطُ التوثيق: %v", err)
	}
}

// requireWhatsApp **يُشغّل شرطَ التوثيق أو يُطفئه** — المفتاحان معاً، ويُرجَعان
// بالتنظيف (`setSetting`).
func requireWhatsApp(t *testing.T, f *fixture, master, role bool) {
	t.Helper()
	f.setSetting(t, "auth.require_whatsapp", master)
	f.setSetting(t, "customers.require_whatsapp", role)
}

// ── A + D · الشرطُ مشتغلٌ وزبونٌ غيرُ موثَّق ⇒ يُرفض ولا يُكتب ────────────
//
// **والمسارُ الخدميُّ هو النداءُ المباشر بعينه** — `POST /orders/custom` لا
// يزيد عليه. فرفضُ `CreateCustomTx` يُثبت أنّ الطلبَ المصوغَ بيده لا يلتفّ.
func TestCustomWhatsApp_RequiredUnverifiedRejected(t *testing.T) {
	ctx := context.Background()
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	requireWhatsApp(t, f, true, true)
	setWhatsAppVerified(t, f, false)

	before := customCount(ctx, f)
	wtx := walletTxCount(ctx, f)
	_, err := placeCustom(ctx, f, "wallet")
	if !errors.Is(err, orders.ErrWhatsAppRequired) {
		t.Fatalf("**طلبٌ خاصٌّ من حسابٍ غيرِ موثَّقٍ لم يُرفض** (%v) — "+
			"والبابُ الخاصُّ يلتفّ على شرطٍ يحميه العاديّ", err)
	}
	if after := customCount(ctx, f); after != before {
		t.Errorf("**أُنشئ طلبٌ خاصٌّ رغم الرفض**: %d ⇐ %d", before, after)
	}
	if w := walletTxCount(ctx, f); w != wtx {
		t.Errorf("**رفضُ توثيقٍ حرّك المحفظة**: %d ⇐ %d", wtx, w)
	}
}

// ── B · الشرطُ مشتغلٌ وزبونٌ موثَّقٌ ⇒ يُقبل ──────────────────────────────
//
// **وحارسٌ يرفض الجميعَ نصفُ حارس** — لا بدّ أن يمرّ الموثَّق.
func TestCustomWhatsApp_RequiredVerifiedAccepted(t *testing.T) {
	ctx := context.Background()
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	requireWhatsApp(t, f, true, true)
	setWhatsAppVerified(t, f, true)

	o, err := placeCustom(ctx, f, "wallet")
	if err != nil {
		t.Fatalf("**رُفض طلبٌ خاصٌّ من حسابٍ موثَّقٍ والشرطُ مشتغل** — %v", err)
	}
	if o == nil || o.ID == "" {
		t.Error("لم يُنشأ الطلبُ الخاصُّ المشروع")
	}
}

// ── C-role · الشرطُ مطفأٌ بمفتاح الدور ⇒ لا يُخترَع منعٌ ─────────────────
//
// **مفتاحُ الدور false (الافتراض)** — `RequireWhatsApp` تردّ false، فيمرّ غيرُ
// الموثَّق كما يمرّ في العاديّ. **الإصلاحُ لا يفرض ما لم يُرِده المالك.**
func TestCustomWhatsApp_RoleOffNotBlocked(t *testing.T) {
	ctx := context.Background()
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	requireWhatsApp(t, f, true, false) // العامُّ مشتغلٌ والدورُ مطفأ
	setWhatsAppVerified(t, f, false)

	o, err := placeCustom(ctx, f, "wallet")
	if err != nil {
		t.Fatalf("**مُنع طلبٌ خاصٌّ والشرطُ مطفأٌ بمفتاح الدور** — %v", err)
	}
	if o == nil || o.ID == "" {
		t.Error("لم يُنشأ الطلبُ رغم إطفاء الشرط")
	}
}

// ── C-master · المفتاحُ العامُّ يعلو مفتاحَ الدور ─────────────────────────
//
// **العامُّ off ولو كان الدورُ on** — السياسةُ off. **وهو سببُ كمون D8**:
// حين يُطفأ العامُّ لا فرقَ بين البابين، فلا يُخترَع منعٌ.
func TestCustomWhatsApp_MasterOffOverridesRole(t *testing.T) {
	ctx := context.Background()
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	requireWhatsApp(t, f, false, true) // العامُّ مطفأٌ والدورُ مشتغل
	setWhatsAppVerified(t, f, false)

	o, err := placeCustom(ctx, f, "wallet")
	if err != nil {
		t.Fatalf("**العامُّ مطفأٌ ومع ذلك مُنع الطلبُ الخاصّ** — %v", err)
	}
	if o == nil || o.ID == "" {
		t.Error("لم يُنشأ الطلبُ والعامُّ مطفأ")
	}
}
