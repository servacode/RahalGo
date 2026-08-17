package media

// **اللمحةُ تُولَد والنسخُ تصغر — ويُقاس البايت.**
//
// (طلبُ المالك ٢٠٢٦-٠٨-١٧: «نريد حلَّ مشكلة الصور بحيث لا يلاحظ المستخدمُ
//  أنّ الصور تتحمّل مهما كان الإنترنت بطيئاً».)
//
// # ولماذا يُوزن ولا يُعدّ
//
// **وجودُ ملفٍّ باسمٍ صحيحٍ لا يقول إنّه أخفّ** — من ولّد نسخةً بعرض ٤٨٠
// وجودةٍ ٩٥ كتب ملفّاً أثقلَ ممّا يحلّ. **والوزنُ هو الشكوى نفسُها.**
//
// # واللمحةُ لها سقفٌ صارم
//
// **تُرسل مع كلّ ورقةٍ تحمل الصورة** — **ولمحةٌ بألفَي بايتٍ في صفحةٍ فيها
// عشرون صورةً أربعون كيلوبايتاً قبل أن يُرى شيء.**

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestVariants_BlurIsTinyAndSizesShrink(t *testing.T) {
	pool := testdb.Pool(t)
	dir := t.TempDir()
	svc, err := NewService(pool, dir)
	if err != nil {
		t.Fatalf("تعذّرت الخدمة: %v", err)
	}
	actor := testdb.NewUser(t, pool, "admin")

	// **صورةٌ تشبه الصورَ الحقيقيّة** — انظر `noisyPNG` في فحص الخلفيّات.
	img := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	rnd := rand.New(rand.NewSource(11))
	for y := 0; y < 900; y++ {
		for x := 0; x < 1600; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(rnd.Intn(256)), G: uint8(rnd.Intn(256)),
				B: uint8(rnd.Intn(256)), A: 255})
		}
	}
	var raw bytes.Buffer
	if err := png.Encode(&raw, img); err != nil {
		t.Fatalf("تعذّر ترميزُ الفحص: %v", err)
	}

	m, err := svc.Save(context.Background(), actor, "banner", bytes.NewReader(raw.Bytes()))
	if err != nil {
		t.Fatalf("تعذّر الرفع: %v", err)
	}

	// ══════════════════════════════════════════════════════════════════
	// **اللمحةُ موجودةٌ وصغيرة**
	// ══════════════════════════════════════════════════════════════════
	if !strings.HasPrefix(m.Blur, "data:image/jpeg;base64,") {
		t.Fatalf("لا لمحةَ في الصفّ: %q", m.Blur)
	}
	if len(m.Blur) > 2000 {
		t.Fatalf("اللمحةُ %d بايتاً — **وهي تُرسل مع كلّ ورقةٍ تحمل الصورة**",
			len(m.Blur))
	}

	// ══════════════════════════════════════════════════════════════════
	// **والنسخُ مولَّدةٌ وأخفُّ من الأصل**
	// ══════════════════════════════════════════════════════════════════
	if !m.Sizes {
		t.Fatal("لم تُولَّد النسخُ الصغيرة — **والعلمُ يقول ذلك للشاشة**")
	}
	relFull := strings.TrimPrefix(m.URL, "/media/")
	stFull, err := os.Stat(filepath.Join(dir, filepath.FromSlash(relFull)))
	if err != nil {
		t.Fatalf("لا أصلَ على القرص: %v", err)
	}
	prev := stFull.Size()
	for i := len(VariantWidths) - 1; i >= 0; i-- {
		w := VariantWidths[i]
		rel := strings.TrimPrefix(VariantURL(m.URL, w), "/media/")
		st, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("نسخةُ %d مفقودة: %v", w, err)
		}
		// **وكلُّ نسخةٍ أخفُّ ممّا فوقها** — **وترتيبٌ منكوسٌ يعني أنّ
		// المتصفّحَ سيختار الأثقلَ ظانّاً أنّه الأخفّ.**
		if st.Size() >= prev {
			t.Fatalf("نسخةُ %d وزنُها %d وما فوقها %d — **ولا تُخفِّف**",
				w, st.Size(), prev)
		}
		prev = st.Size()
	}
	t.Logf("الأصلُ %d بايتاً · ٩٦٠ ثمّ ٤٨٠ أخفُّ منه · اللمحةُ %d حرفاً",
		stFull.Size(), len(m.Blur))
}
