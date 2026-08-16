package media

// **الخلفيّةُ لا تُخزَّن PNG — والوزنُ يُقاس لا يُدَّعى.**
//
// (شكوى المالك ٢٠٢٦-٠٨-١٦: «يجب أن يتمّ تحميلها بسرعة رغم ضعف الإنترنت».)
//
// # ما كان
//
// **خلفيّةُ موقعه كانت PNG بمليونٍ ومئةِ ألفِ بايت** — ١٦٠٠×٩٠٠. **قِيس
// تحميلُها من اتّصالٍ جيّدٍ فبلغ ثلاثَ عشرةَ ثانية**، وعلى شبكةٍ ضعيفةٍ
// أضعافُها.
//
// **والسببُ قاعدةٌ كُتبت للشعارات**: «PNG يبقى PNG للحفاظ على الشفافيّة».
// **وهي صحيحةٌ لشعارٍ وقاتلةٌ لصورةٍ فوتوغرافيّة** — وPNG أسوأُ صيغةٍ
// للصور الطبيعيّة.
//
// # ولماذا يُوزن ولا يُقرأ الامتداد
//
// **امتدادُ `.jpg` لا يقول إنّ الملفَّ صغُر**: من رفع الجودةَ إلى ٩٥ غداً
// يبقى الامتدادُ jpg **ويعود الوزنُ إلى مئات الكيلوبايتات.** **والوزنُ هو
// الشكوى نفسُها**، فهو ما يُفحص.

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

// noisyPNG **صورةٌ تشبه الصورَ الحقيقيّة** — **وPNG لمساحةٍ لونيّةٍ واحدةٍ
// يضغطها إلى بايتاتٍ معدودة**، فيمرّ الفحصُ على شيءٍ لا يقع في التشغيل.
func noisyPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	rnd := rand.New(rand.NewSource(7)) // ثابتٌ — فحصٌ لا يتأرجح بين تشغيلتين
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(rnd.Intn(256)), G: uint8(rnd.Intn(256)),
				B: uint8(rnd.Intn(256)), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("تعذّر ترميزُ الفحص: %v", err)
	}
	return buf.Bytes()
}

func TestBackgroundPNG_IsStoredAsJPEGAndShrinks(t *testing.T) {
	pool := testdb.Pool(t)
	dir := t.TempDir()
	svc, err := NewService(pool, dir)
	if err != nil {
		t.Fatalf("تعذّرت الخدمة: %v", err)
	}
	actor := testdb.NewUser(t, pool, "admin")
	ctx := context.Background()

	raw := noisyPNG(t, 1600, 900)

	// **والشعارُ يبقى PNG** — شفافيّتُه معنًى، **وقاعدةٌ تحوّل الكلَّ تكسر
	// شعارَ المنصّة على أيّ خلفيّةٍ يوضع عليها.**
	logo, err := svc.Save(ctx, actor, "platform_logo", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("تعذّر رفعُ الشعار: %v", err)
	}
	if !strings.HasSuffix(logo.URL, ".png") {
		t.Fatalf("الشعارُ خرج %q — **وشفافيّةُ الشعار معنًى يُحفظ**", logo.URL)
	}

	bg, err := svc.Save(ctx, actor, "site_background", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("تعذّر رفعُ الخلفيّة: %v", err)
	}
	if !strings.HasSuffix(bg.URL, ".jpg") {
		t.Fatalf("الخلفيّةُ خُزّنت %q — **ولا شفافيّةَ تحت خلفيّةٍ تملأ الشاشة**",
			bg.URL)
	}

	// ══════════════════════════════════════════════════════════════════
	// **والوزنُ هو الشكوى — فهو ما يُفحص**
	// ══════════════════════════════════════════════════════════════════
	rel := strings.TrimPrefix(bg.URL, "/media/")
	st, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("لا ملفَّ على القرص: %v", err)
	}
	// **والنسبةُ لا الرقمُ المطلق**: حجمُ الخرج يتبع محتوى الصورة،
	// **وفحصٌ يشترط رقماً بعينه يسقط يومَ تُبدَّل صورةُ الفحص.**
	if st.Size() >= int64(len(raw))/2 {
		t.Fatalf("الخلفيّةُ %d بايتاً من أصل %d — **ولم تصغر النصفَ حتّى**",
			st.Size(), len(raw))
	}
	t.Logf("PNG %d بايتاً → JPEG %d بايتاً (%.0f%%)",
		len(raw), st.Size(), 100*float64(st.Size())/float64(len(raw)))
}
