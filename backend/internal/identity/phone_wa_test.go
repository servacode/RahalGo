package identity

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestChangingPhoneDropsWhatsAppProof **تبديلُ الرقم يُسقط الإثبات.**
//
// ══════════════════════════════════════════════════════════════════════
// **العطبُ الذي يحرسه**
// ══════════════════════════════════════════════════════════════════════
//
// **كان تبديلُ الرقم لا يمسّ التوثيق** — فيبقى الرقمُ القديم ممهوراً
// بختم «موثَّق» وصاحبُه يدخل برقمٍ آخر. **وهما رقمان مختلفان في حسابٍ
// واحد**، وهو ما قال المالكُ عنه «هيك تخرب الدنيا».
//
// **وأخطرُ ما فيه أنّه صامت**: رمزُ استعادة رمز الأدمن يُقرأ من
// `whatsapp_phone` — **فيُرسَل إلى الرقم القديم.** ومن بدّل رقمَه لأنّه
// فقده، **يُرسَل رمزُ فتح لوحته إلى من يحمله الآن.**
func TestChangingPhoneDropsWhatsAppProof(t *testing.T) {
	pool := testdb.Pool(t)
	repo := NewRepo(pool)
	ctx := context.Background()
	id := testdb.NewUser(t, pool, "customer")

	const old = "+963900000001"
	const fresh = "+963900000002"

	if err := repo.SetPhone(ctx, id, old); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetWhatsApp(ctx, id, old); err != nil {
		t.Fatal(err)
	}
	if got, err := repo.VerifiedWhatsApp(ctx, id); err != nil || got != old {
		t.Fatalf("التوثيقُ لم يُثبَّت أصلا: %q %v", got, err)
	}

	// **يبدّل رقمَه** — فيسقط الإثبات.
	if err := repo.SetPhone(ctx, id, fresh); err != nil {
		t.Fatal(err)
	}
	got, err := repo.VerifiedWhatsApp(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("بقي رقمٌ موثَّقٌ بعد تبديل الرقم: %q — "+
			"ورمزُ استعادة اللوحة يُرسَل إليه", got)
	}
}

// TestVerifyingThenMakingItTheLoginKeepsTheProof **وما أُثبت لا يُهدَر.**
//
// **من وثّق رقماً ثمّ جعله رقمَ دخوله** لم يتبدّل عنده شيء — **وإسقاطُ
// إثباتٍ قائمٍ يُتعب بلا سبب**، ويجعل القاعدةَ تُقرأ عقوبةً لا حراسة.
func TestVerifyingThenMakingItTheLoginKeepsTheProof(t *testing.T) {
	pool := testdb.Pool(t)
	repo := NewRepo(pool)
	ctx := context.Background()
	id := testdb.NewUser(t, pool, "customer")

	const login = "+963900000003"
	const proven = "+963900000004"

	if err := repo.SetPhone(ctx, id, login); err != nil {
		t.Fatal(err)
	}
	// **وثّق رقماً آخرَ** ثمّ جعله رقمَ دخوله.
	if err := repo.SetWhatsApp(ctx, id, proven); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetPhone(ctx, id, proven); err != nil {
		t.Fatal(err)
	}
	got, err := repo.VerifiedWhatsApp(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got != proven {
		t.Fatalf("أُسقط إثباتٌ قائمٌ على الرقم نفسِه: %q", got)
	}
}
