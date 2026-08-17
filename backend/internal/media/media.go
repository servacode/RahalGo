// Package media نظام الوسائط المركزي (المرحلة 3): استقبال الصور ومعالجتها
// وتخزينها على القرص وتسجيلها في قاعدة البيانات.
//
// قواعد المعالجة الأمنية:
//   - النوع يُفحص من البايتات الأولى (magic bytes) لا من الامتداد أو الترويسة.
//   - الصورة تُفكَّك وتُعاد كتابتها بالكامل — أي حمولة مدسوسة أو بيانات EXIF تُمحى.
//   - أسماء الملفات uuid عشوائية داخل مجلدات yyyy/mm — لا اسم من العميل يصل للقرص.
package media

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // فك WebP ضمن image.Decode

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

const (
	// MaxUploadBytes **احتياطيٌّ لا حدّ** — والحدُّ الفعليُّ في
	// `media.max_upload_mb`. (نُقل إلى اللوحة 2026-08-09 بقرار المالك.)
	//
	// **ويبقى مصدّراً**: النداءاتُ قبل الربط تعمل به، **واختبارُ الوسائط لا
	// يحتاج مخزنَ إعدادات.**
	MaxUploadBytes = 5 << 20 // 5MB
	maxDim         = 1600    // البعد الأقصى للنسخة الكاملة

	// maxPixels **سقفُ مساحة الصورة قبل فكّ ترميزها.**
	//
	// **ثلاثةَ عشرَ مليوناً — فوق كاميرا الهاتف الشائعة بهامش.**
	// (قِيس ٢٠٢٦-٠٨-١١: حدُّ ١٢ مليوناً بالضبط **رفض ٤٠٣٢×٣٠٢٤** —
	//  وهي ١٢٫١٩ مليوناً، **أشيعُ مقاسِ صورةٍ سيرفعه الناس.**)
	//
	// **وفكُّها يحجز نحوَ ٥٢ ميغا**، ومع نسختَي التصغير يبقى المجموعُ في
	// حدود خطّةٍ بنصف غيغا — **قِيس تحت ٥١٢ ميغا فبلغ ١٣٦.**
	//
	// **وما فوقه لا يفيد**: الخرجُ ١٦٠٠ بكسلاً على أيّ حال.
	maxPixels   = 13_000_000
	thumbDim    = 400 // البعد الأقصى للمصغرة
	jpegQuality = 82

	// backgroundQuality **جودةُ الخلفيّات وحدَها.**
	//
	// **وهي تُرسم تحت حجابٍ وخلفَ زجاج** — فلا عينَ تفحص تفاصيلَها،
	// **وكلُّ نقطةٍ من الجودة ثمنُها بايتاتٌ ينتظرها زائرٌ على شبكةٍ ضعيفة.**
	backgroundQuality = 62

	// **مقاسُ اللمحة وجودتُها** — انظر `blurData`.
	blurDim     = 24
	blurQuality = 30
)

// isBackground **الخلفيّاتُ صنفٌ له قواعدُه** — لا شفافيّةَ تحتها ولا تُفحص
// تفاصيلُها، **ووزنُها وحدَه ما يُرى** لأنّها أوّلُ ما يُحمَّل وأكبرُه.
func isBackground(kind string) bool {
	return kind == "site_background" || kind == "auth_background"
}

// keepsAlpha **الشفافيّةُ معنًى في الشعارات وحدَها.**
//
// (طلبُ المالك ٢٠٢٦-٠٨-١٧: «أنا أرفع أيَّ نوعِ صورةٍ يجب أن يُعدَّل
//
//	برمجيّاً بحيث تُحلّ مشكلةُ بطء تحميل الصور».)
//
// **وقِيست لافتتُه فوُجدت أربعةً وثلاثين ألفَ... لا: أربعةَ ملايينَ
// وثلاثمئةِ ألفِ بايت** — لأنّها رُفعت PNG وبقيت PNG. **وPNG أسوأُ صيغةٍ
// للصور الطبيعيّة**، وكانت القاعدةُ تحوّل الخلفيّاتِ وحدَها.
//
// **والشعارُ يُوضع على ألوانٍ شتّى** — شريطٍ داكنٍ وورقةٍ بيضاءَ وقرصٍ
// أبيض: **فشفافيّتُه معنًى، وتسطيحُها يضع حولَه مربّعاً.**
//
// **وما عداه صورةٌ تملأ إطارَها** — لافتةٌ أو صنفٌ أو صورةُ حسابٍ أو
// إثباتُ تسليم: **لا شيءَ خلفَها يُرى.**
func keepsAlpha(kind string) bool {
	return kind == "platform_logo" || kind == "merchant_logo"
}

var (
	ErrBadImage = httpx.NewError(http.StatusBadRequest, "invalid_image", "errors.invalid_image")
	ErrTooLarge = httpx.NewError(http.StatusBadRequest, "image_too_large", "errors.image_too_large")
	// ErrImageTooBig **أبعادٌ فوق الحدّ** — غيرُ حدِّ البايتات.
	//
	// **ورسالةٌ خاصّةٌ بها لا رسالةُ الحجم**: من قُيل له «صورتك أكبر من
	// ٥ ميغا» وهي ثلاثةٌ **يضغطها أكثرَ فتُرفض ثانيةً** — والعلّةُ في
	// الأبعاد لا في الوزن.
	ErrImageTooBig = httpx.NewError(http.StatusBadRequest, "image_dimensions", "errors.image_dimensions")
)

// **وإثباتُ التسليم نوعٌ منها** — له الفحصُ والحدُّ نفسُهما.
// **و`platform_logo` شعارُ المنصة** — يُرفع من الإعدادات ويُعرض في كلّ
// شريطٍ علويٍّ وفوترٍ وفاتورة. (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا تنسَ إضافة هوية
// المنصة أيضاً — الاسم واللوغو».)
// **و`auth_background` خلفيّةُ شاشات الدخول** — تُرفع من الإعدادات وتُعرض
// خلف بطاقة الدخول والتسجيل والاستعادة في الخمسة.
// (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «خيار بالإعدادات أرفع الصورة وأغيّرها إيمت
//
//	ما بدّي».)
//
// **و`site_background` خلفيّةُ الموقع كلِّه** — تُرفع من الإعدادات وتُرسم على
// `body` فترثها البوّاباتُ الخمس.
//
// **ونوعٌ مستقلٌّ عن `auth_background` عمداً**: خلفيّةُ شاشة الدخول قد تكون
// صورةً هادئةً وخلفيّةُ الموقع أخرى، **ونوعٌ واحدٌ للاثنين يجعل حذفَ إحداهما
// يبحث في صور الأخرى.**
// **و`menu_section` قسمُ قائمةِ متجرٍ — غيرُ `platform_section` الذي للمنصة.**
// (هجرة ٠٠٨٣ · طلبُ المالك ٢٠٢٦-٠٨-٠٧.)
var validKinds = map[string]bool{"merchant_logo": true, "menu_item": true, "menu_section": true,
	"banner": true, "avatar": true, "delivery_proof": true, "platform_logo": true,
	"auth_background": true, "site_background": true}

type Media struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumb_url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Bytes    int64  `json:"bytes"`
	// Blur **لمحةٌ فوريّةٌ مضمَّنةٌ في الصفّ** — `data:` بنحو ستّمئة بايت.
	//
	// (طلبُ المالك ٢٠٢٦-٠٨-١٧: «بحيث لا يلاحظ المستخدمُ أنّ الصور
	//  تتحمّل مهما كان الإنترنت بطيئاً».)
	//
	// **تُرسل مع الورقة فتُرى في أوّل رسمة** — **ولو كانت ملفّاً
	// لَاحتاجت رحلةً ثانيةً**، وهي التي جاءت لتُلغي الانتظار.
	Blur string `json:"blur"`
	// Sizes **هل وُلّدت النسخُ الصغيرة؟** — الصفوفُ القديمةُ بلا نسخ،
	// **ومن طلب مقاساً لم يُولَّد يأخذ ٤٠٤ في وسط الصفحة.**
	Sizes bool `json:"sizes"`
}

// VariantWidths **مقاساتُ النسخ المولَّدة** — بترتيب تصاعديّ.
//
// **وأربعُمئةٍ وثمانون تكفي هاتفاً بكثافةٍ مضاعفة** (٢٤٠ نقطةً منطقيّة)،
// **وتسعُمئةٍ وستّون لوحيّاً ولابتوب** — **والأصلُ لِما فوقهما.**
var VariantWidths = []int{480, 960}

// VariantURL **مسارُ نسخةٍ بعرضٍ بعينه** — يُشتقّ ولا يُخزَّن.
//
// **وأعمدةٌ لكلّ مقاسٍ تعني هجرةً كلَّما أُضيف مقاس.**
func VariantURL(fullURL string, w int) string {
	dot := strings.LastIndex(fullURL, ".")
	if dot < 0 {
		return fullURL
	}
	return fullURL[:dot] + "_" + strconv.Itoa(w) + fullURL[dot:]
}

type Service struct {
	db  *pgxpool.Pool
	dir string // مجلد التخزين الجذري
	// setting يقرأ إعداداً عددياً من اللوحة، أو يعيد الاحتياطي.
	//
	// **دالّة لا مخزن** — كما في حزمة الهوية: لو حُقن `*settings.Store`
	// لاعتمدت حزمةُ الوسائط على حزمة الإعدادات وهي أدنى منها في الترتيب.
	setting func(ctx context.Context, key string, fallback int64) int64
}

func NewService(db *pgxpool.Pool, dir string) (*Service, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("media: create uploads dir: %w", err)
	}
	return &Service{db: db, dir: dir}, nil
}

// SetSettingReader يربط الخدمة بإعدادات اللوحة (تُنادى مرّة عند الإقلاع).
func (s *Service) SetSettingReader(f func(ctx context.Context, key string, fallback int64) int64) {
	s.setting = f
}

// MaxBytes سقفُ الرفع بالبايت — من اللوحة أو الاحتياطيّ.
//
// **ويُنادى في الخادم أيضاً**: `MaxBytesReader` يقطع الجسدَ قبل قراءته،
// **وسقفان مختلفان يجعلان الرفضَ يقع بلا رسالةٍ مفهومة** — يُقطع الاتّصال
// بدل أن يُقال «الصورة كبيرة».
func (s *Service) MaxBytes(ctx context.Context) int64 {
	if s.setting == nil {
		return MaxUploadBytes
	}
	return s.setting(ctx, "media.max_upload_mb", MaxUploadBytes>>20) << 20
}

// Dir يعيد مجلد التخزين الجذري (لخدمة الملفات الساكنة).
func (s *Service) Dir() string { return s.dir }

// URLFor يحوّل مساراً نسبياً مخزناً إلى مسار عام يخدمه الخادم.
func URLFor(path string) string { return "/media/" + path }

// URLForPtr مثل URLFor لكن لمسار قد يكون NULL (كيان بلا صورة).
func URLForPtr(path *string) *string {
	if path == nil || *path == "" {
		return nil
	}
	u := URLFor(*path)
	return &u
}

// Save يعالج صورة مرفوعة ويخزنها: فحص النوع الحقيقي، إعادة ترميز كاملة،
// تصغير للحد الأقصى + نسخة مصغرة، ثم تسجيل في قاعدة البيانات.
func (s *Service) Save(ctx context.Context, actorID, kind string, r io.Reader) (*Media, error) {
	if !validKinds[kind] {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}

	limit := s.MaxBytes(ctx)
	raw, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, ErrTooLarge
	}
	if !looksLikeImage(raw) {
		return nil, ErrBadImage
	}

	// ══════════════════════════════════════════════════════════════════
	// **والأبعادُ تُقرأ قبل فكّ الترميز — وإلّا قُتلت الحاوية**
	// ══════════════════════════════════════════════════════════════════
	//
	// (شهده المالك ٢٠٢٦-٠٨-١١ على النسخة المرفوعة: رفعُ خلفيّة الموقع يردّ
	//  «حدث خطأٌ غير متوقّع».)
	//
	// **حدُّ البايتات لا يحدّ الذاكرة**: صورةٌ مضغوطةٌ بأربعة ميغا تُفكّ إلى
	// `عرض × ارتفاع × ٤` بايت. **و٦٠٠٠×٨٠٠٠ = ١٩٢ ميغا للنسخة الواحدة**،
	// ومعها نسخةُ التصغير والمصغَّرة.
	//
	// **وقِيس فعلاً**: بلغت الحاويةُ **٦٣٣ ميغا** — وخطّةُ الاستضافة
	// **٥١٢**. فتُقتل الحاويةُ في منتصف الطلب، **وينقطع الاتصالُ بلا ردّ**
	// فتقرأ الشاشةُ «خطأٌ غير متوقّع» ولا شيءَ في سجلّ يقول لماذا.
	//
	// **و`DecodeConfig` تقرأ الترويسةَ وحدَها** — بايتاتٍ معدودةً بلا حجز.
	//
	// **والحدُّ بالمساحة لا بالضلع**: صورةٌ ٢٠٠٠٠×١٠٠ ضلعُها كبيرٌ ومساحتُها
	// صغيرة، **والذاكرةُ تتبع المساحة.**
	//
	// **والخرجُ ١٦٠٠ بكسلاً على أيّ حال** — فما فوق الحدّ لا يزيد جودةً،
	// **إنّما يزيد خطرَ سقوط الخدمة.**
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, ErrBadImage
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxPixels {
		return nil, ErrImageTooBig
	}

	src, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, ErrBadImage
	}

	// PNG يبقى PNG للحفاظ على الشفافية (شعارات)، وكل ما عداه يُعاد ترميزه JPEG.
	//
	// ══════════════════════════════════════════════════════════════════
	// **إلّا الخلفيّات — فلا شفافيّةَ تحتها أصلاً**
	// ══════════════════════════════════════════════════════════════════
	//
	// (شكوى المالك ٢٠٢٦-٠٨-١٦: «يجب أن يتمّ تحميلها بسرعة رغم ضعف
	//  الإنترنت — يعني ما تتأخّر بالتحميل».)
	//
	// **وخلفيّتُه كانت PNG بمليونٍ ومئةِ ألفِ بايت** — ١٦٠٠×٩٠٠، **قِيس
	// تحميلُها من اتّصالٍ جيّدٍ فبلغ ثلاثَ عشرةَ ثانية.** والصورةُ نفسُها
	// JPEG لا تبلغ عُشرَ ذلك.
	//
	// **والسببُ أنّ القاعدةَ فوقها كُتبت للشعارات**: شعارٌ بخلفيّةٍ شفّافةٍ
	// يجب أن يبقى PNG، **وصورةٌ فوتوغرافيّةٌ تُرفع PNG تبقى PNG** — وهي
	// أسوأُ صيغةٍ للصور الطبيعيّة على الإطلاق.
	//
	// **والخلفيّةُ تملأ الشاشةَ ولا شيءَ خلفَها** — فالشفافيّةُ فيها لا
	// معنى لها، **وثمنُها ثوانٍ ينتظرها كلُّ زائر.**
	ext, encode := ".jpg", encodeJPEG
	if format == "png" && keepsAlpha(kind) {
		ext, encode = ".png", encodePNG
	}
	// **وجودةٌ أخفضُ للخلفيّة وحدَها.**
	//
	// **وهي تُرسم تحت حجابٍ معتم** (`platform.background_dim`) **وخلفَ
	// بطاقاتٍ زجاجيّة** — فما يُفقد من التفاصيل لا تراه عينٌ أصلاً،
	// **والمكسبُ نحوُ الثلث من الحجم.**
	if isBackground(kind) {
		encode = encodeJPEGAt(backgroundQuality)
	}

	full := downscale(src, maxDim)
	thumb := downscale(src, thumbDim)
	hasSizes := false

	now := time.Now().UTC()
	base := filepath.Join(now.Format("2006"), now.Format("01"))
	if err := os.MkdirAll(filepath.Join(s.dir, base), 0o755); err != nil {
		return nil, err
	}
	id := uuid.NewString()
	relPath := filepath.ToSlash(filepath.Join(base, id+ext))
	relThumb := filepath.ToSlash(filepath.Join(base, id+"_t"+ext))

	size, err := writeImage(filepath.Join(s.dir, relPath), full, encode)
	if err != nil {
		return nil, err
	}
	if _, err := writeImage(filepath.Join(s.dir, relThumb), thumb, encode); err != nil {
		_ = os.Remove(filepath.Join(s.dir, relPath))
		return nil, err
	}

	// ══════════════════════════════════════════════════════════════════
	// **ونسختان أصغرُ يختار المتصفّحُ بينهما**
	// ══════════════════════════════════════════════════════════════════
	//
	// (طلبُ المالك ٢٠٢٦-٠٨-١٧.)
	//
	// **صورةٌ بعرض ١٦٠٠ تُرسل إلى هاتفٍ عرضُه ٣٩٠** — **فيُحمَّل أربعةُ
	// أضعافِ ما يُرى**، وهو أكثرُ ما يبطئ الصفحةَ على شبكةٍ ضعيفة.
	//
	// **ولا تُكبَّر صورةٌ صغيرة**: من رفع عرضاً أقلَّ من المقاس لا يُولَّد
	// له — **ونسخةٌ مكبَّرةٌ أثقلُ من أصلها وأسوأُ منه.**
	for _, w := range VariantWidths {
		if full.Bounds().Dx() <= w {
			continue
		}
		vp := filepath.Join(s.dir, filepath.FromSlash(
			VariantURL(relPath, w)))
		if _, err := writeImage(vp, downscale(full, w), encode); err != nil {
			// **وفشلُ نسخةٍ لا يُسقط الرفع** — الأصلُ موجودٌ ويُعرض،
			// **والعلمُ يبقى كاذباً لو رُفع**، فيُطفأ أدناه.
			hasSizes = false
			break
		}
		hasSizes = true
	}

	// **واللمحةُ من الصورة نفسِها** — أربعةٌ وعشرون بكسلاً بجودةٍ منخفضة.
	blur := blurData(src)

	b := full.Bounds()
	m := &Media{Kind: kind, Width: b.Dx(), Height: b.Dy(), Bytes: size,
		URL: URLFor(relPath), ThumbURL: URLFor(relThumb), Blur: blur, Sizes: hasSizes}
	err = s.db.QueryRow(ctx, `
		INSERT INTO media (kind, path, thumb_path, width, height, bytes, created_by,
		                   blur, sizes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		kind, relPath, relThumb, m.Width, m.Height, size, actorID, blur, hasSizes).Scan(&m.ID)
	if err != nil {
		_ = os.Remove(filepath.Join(s.dir, relPath))
		_ = os.Remove(filepath.Join(s.dir, relThumb))
		return nil, err
	}
	return m, nil
}

// looksLikeImage يتحقق من التواقيع الثنائية المدعومة: JPEG وPNG وWebP.
func looksLikeImage(b []byte) bool {
	switch {
	case len(b) > 3 && bytes.HasPrefix(b, []byte{0xFF, 0xD8, 0xFF}):
		return true // JPEG
	case len(b) > 8 && bytes.HasPrefix(b, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return true // PNG
	case len(b) > 12 && bytes.HasPrefix(b, []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return true // WebP
	}
	return false
}

// blurData **لمحةٌ صغيرةٌ جدّاً تُضمَّن نصّاً.**
//
// (طلبُ المالك ٢٠٢٦-٠٨-١٧: «لا يلاحظ المستخدمُ أنّ الصور تتحمّل».)
//
// **وأربعةٌ وعشرون بكسلاً بجودةِ ثلاثين** — نحوُ أربعمئةِ بايتٍ بعد
// الترميز. **وأكبرُ منها يثقل كلَّ ورقةٍ تحملها**، وهي تُرسل مع كلّ صفحة.
//
// **والمتصفّحُ يمدّها فتصير ضبابيّةً** — لا تُقرأ تفصيلاً، **إنّما تقول
// «هنا صورةٌ وهذه ألوانُها» فلا يُرى صندوقٌ فارغ.**
//
// **وفشلُها يُرجع فراغاً ولا يُسقط الرفع** — **وصورةٌ بلا لمحةٍ تعمل،
// ورفعٌ يسقط لأجل لمحةٍ لا.**
func blurData(src image.Image) string {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, downscale(src, blurDim),
		&jpeg.Options{Quality: blurQuality}); err != nil {
		return ""
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// downscale يصغّر الصورة بحيث لا يتجاوز أطول أبعادها max — لا تكبير أبداً.
func downscale(src image.Image, max int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= max && h <= max {
		return src
	}
	if w > h {
		h = h * max / w
		w = max
	} else {
		w = w * max / h
		h = max
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	return dst
}

func encodeJPEG(w io.Writer, img image.Image) error {
	return jpeg.Encode(w, img, &jpeg.Options{Quality: jpegQuality})
}

// encodeJPEGAt **جودةٌ مختارةٌ لصنفٍ بعينه** — انظر `backgroundQuality`.
func encodeJPEGAt(q int) func(io.Writer, image.Image) error {
	return func(w io.Writer, img image.Image) error {
		return jpeg.Encode(w, img, &jpeg.Options{Quality: q})
	}
}

func encodePNG(w io.Writer, img image.Image) error {
	return (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(w, img)
}

func writeImage(path string, img image.Image, encode func(io.Writer, image.Image) error) (int64, error) {
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	if err := encode(f, img); err != nil {
		f.Close()
		_ = os.Remove(path)
		return 0, err
	}
	if err := f.Close(); err != nil {
		return 0, err
	}
	st, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return st.Size(), nil
}

// FileServer يخدم الملفات المخزنة بترويسات تخزين مؤقت طويلة —
// الأسماء uuid فريدة فلا يتغير محتوى مسار أبداً.
func (s *Service) FileServer() http.Handler {
	fs := http.FileServer(http.Dir(s.dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "..") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fs.ServeHTTP(w, r)
	})
}
