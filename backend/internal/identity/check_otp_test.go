package identity

import (
	"context"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// مستودعٌ على قاعدة الاختبار — **والهواتفُ تُفرَّق بالاختبار** كي لا يتصادم
// اثنان يعملان على القاعدة نفسِها.
func testRepo(t *testing.T) *Repo {
	t.Helper()
	return NewRepo(testdb.Pool(t))
}

// TestCheckOTPDoesNotConsume — **الفحصُ يقرأ ولا يُبطل.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا تظهر المعلومات إلّا بعد التحقّق من الرمز».)
//
// **وهذا هو الخطرُ الوحيدُ في الخطوة الجديدة**: لو استهلك الفحصُ الرمزَ لَصدّق
// الخادمُ عليه ثمّ رفضه بعد دقيقةٍ حين يضغط «إنشاء حساب» — **وصاحبُه يرى
// رمزاً صحيحاً يُرفض، فيظنّ المنصةَ معطوبة.**
//
// **وفحصٌ يمرّ في الحالين لا يحرس شيئاً** — فالاختبارُ يُثبت الثلاثة معاً:
// الفحصُ يصدّق · ثمّ يصدّق ثانيةً (فلم يُستهلك) · **ثمّ يستهلكه `ConsumeOTP`
// مرّةً واحدة.**
func TestCheckOTPDoesNotConsume(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	const phone, hash = "0999000111", "hash-of-code"
	if err := repo.CreateOTP(ctx, phone, hash, "signup", 10*time.Minute); err != nil {
		t.Fatalf("تعذّر إنشاء الرمز: %v", err)
	}

	// ١ · يصدّق
	ok, err := repo.CheckOTP(ctx, phone, hash, "signup")
	if err != nil || !ok {
		t.Fatalf("الفحصُ الأوّل ردّ (%v, %v) — والمنتظر (true, nil)", ok, err)
	}

	// ٢ · ويصدّق ثانيةً — **وهذا ما يُثبت أنّه لم يستهلك**
	ok, err = repo.CheckOTP(ctx, phone, hash, "signup")
	if err != nil || !ok {
		t.Fatalf("الفحصُ الثاني ردّ (%v, %v) — **فالفحصُ استهلك الرمز**", ok, err)
	}

	// ٣ · والاستهلاكُ يقع مرّةً واحدةً بعده
	ok, err = repo.ConsumeOTP(ctx, phone, hash, "signup")
	if err != nil || !ok {
		t.Fatalf("الاستهلاكُ بعد الفحص ردّ (%v, %v) — والمنتظر (true, nil)", ok, err)
	}
	if ok, _ = repo.ConsumeOTP(ctx, phone, hash, "signup"); ok {
		t.Fatal("استُهلك الرمزُ مرّتين")
	}
	// ٤ · **ولا يُفحص بعد الاستهلاك**
	if ok, _ = repo.CheckOTP(ctx, phone, hash, "signup"); ok {
		t.Fatal("**الفحصُ صدّق رمزاً مُستهلَكاً** — ونافذةُ إنشاءِ حسابٍ مفتوحة")
	}
}

// TestCheckOTPCountsFailures — **الفحصُ يعدّ الفاشلةَ كما يعدّها الاستهلاك.**
//
// **فحصٌ لا يعاقب التخمينَ يصير أداةَ تخمين**: رمزٌ من ستّة أرقامٍ يُكسر
// بمليونِ نداءٍ إن لم يُبطَل. **وبعد خمسٍ يُبطل الرمزُ حتّى للصحيح.**
func TestCheckOTPCountsFailures(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	const phone, good, bad = "0999000222", "right-hash", "wrong-hash"
	if err := repo.CreateOTP(ctx, phone, good, "signup", 10*time.Minute); err != nil {
		t.Fatalf("تعذّر إنشاء الرمز: %v", err)
	}
	for i := 0; i < 5; i++ {
		if ok, _ := repo.CheckOTP(ctx, phone, bad, "signup"); ok {
			t.Fatalf("صدّق الفحصُ رمزاً خاطئاً في المحاولة %d", i+1)
		}
	}
	// **والصحيحُ نفسُه يُرفض بعد الخامسة** — وإلّا لم تكن العقوبةُ عقوبة
	if ok, _ := repo.CheckOTP(ctx, phone, good, "signup"); ok {
		t.Fatal("**قُبل الرمزُ الصحيحُ بعد خمس محاولاتٍ فاشلة** — فالفحصُ لا يعاقب التخمين")
	}
}
