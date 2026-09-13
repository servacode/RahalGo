package release

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Public **ما يُقال للناس عن تطبيقٍ — ولا حرفَ فيه من داخل الخادم.**
//
// **ولا مسارَ ملفٍّ ولا مجلَّدَ نشرٍ ولا اسمَ ملفٍّ مخزَّن**: **اسمُ
// الملفّ المخزَّن معرّفٌ عشوائيٌّ يكشف بنيةَ التخزين**، **والمسارُ
// العامُّ وحدَه يُعرَض.**
type Public struct {
	Key       string  `json:"key"`
	PackageID string  `json:"package_id"`
	Channel   Channel `json:"channel"`
	Status    Status  `json:"status"`
	// PlayURL رابطُ المتجر — فارغٌ يعني «لم يُضبط».
	PlayURL string `json:"play_url"`
	// DownloadURL مسارُ الأثر الرسميّ — **فارغٌ يعني «لا زرَّ تنزيل».**
	DownloadURL string `json:"download_url"`
	// Version اسمُ النسخة إن كُتب — **ولا يُخترَع.**
	Version string `json:"version"`
	// SizeBytes حجمُ الأثر — **من القرص لا من إعداد.**
	SizeBytes int64 `json:"size_bytes"`
	// SHA256 بصمةُ الأثر — **محسوبةٌ من الملفّ لا مكتوبةٌ في إعداد.**
	SHA256 string `json:"sha256"`
}

// Store **ما يلزم لحساب الحال** — إعداداتٌ ومجلَّدُ الآثار.
//
// **وواجهةٌ صغيرةٌ لا حزمةُ خادمٍ كاملة**: **فيُقاس هذا المنطقُ بلا
// خادمٍ ولا قاعدةٍ** — **ومنطقٌ لا يُقاس إلّا بالنظام كلِّه لا يُقاس.**
type Store interface {
	// GetString قيمةُ إعدادٍ نصّيّ — فارغةٌ حين لا يوجد.
	GetString(ctx context.Context, key string) string
	// ArtifactDir مجلَّدُ الآثار المرفوعة.
	ArtifactDir() string
}

// PublicPath مسارُ التنزيل العامُّ لتطبيقٍ — **بمفتاحه لا باسم ملفّه.**
func PublicPath(key string) string { return "/api/v1/public/app/" + key }

// Resolve **حالُ التطبيقات الأربعة كما تُقال للناس.**
//
// # والترتيبُ مقصود
//
// **المتجرُ يسبق** حين يكون قناةَ التطبيق (وهو نصُّ القاعدة القائمة في
// `appHref`: «من رفع تطبيقَه إلى المتجر فالمتجرُ أولى»).
//
// # وما يجعل الزرَّ يظهر
//
//	رابطُ متجرٍ مضبوطٌ                 ⇒ زرُّ متجر
//	أثرٌ موجودٌ على القرص وله بصمة      ⇒ زرُّ تنزيلٍ مباشر
//	ولا شيءَ منهما                     ⇒ «غيرُ متوفّرٍ بعد»
//
// **والوجودُ على القرص شرطٌ لا يُستغنى عنه**: **إعدادٌ يحمل اسمَ ملفٍّ
// حُذف من القرص كان سيُظهر زرّاً يعطي ٤٠٤.** **فيُسأل القرصُ نفسُه.**
//
// **والبصمةُ شرطٌ ثانٍ**: **أثرٌ بلا بصمةٍ لا يُوصَف** — **والمعروضُ
// يجب أن يصف الملفَّ الذي يُنزَّل.**
func Resolve(ctx context.Context, st Store) []Public {
	out := make([]Public, 0, len(Apps))
	for _, a := range Apps {
		out = append(out, ResolveOne(ctx, st, a))
	}
	return out
}

// ResolveOne حالُ تطبيقٍ واحد.
func ResolveOne(ctx context.Context, st Store, a App) Public {
	p := Public{Key: a.Key, PackageID: a.PackageID, Channel: a.Channel,
		Status: StatusUnavailable}

	p.PlayURL = playURL(ctx, st, a)
	p.Version = strings.TrimSpace(st.GetString(ctx, VersionKey(a.Key)))

	if size, sha, ok := artifact(ctx, st, a); ok {
		p.DownloadURL = PublicPath(a.Key)
		p.SizeBytes = size
		p.SHA256 = sha
	}

	switch {
	case p.PlayURL != "" && p.DownloadURL != "":
		p.Status = StatusPlayAndDirect
	case p.PlayURL != "":
		p.Status = StatusPlay
	case p.DownloadURL != "":
		p.Status = StatusDirect
	}
	return p
}

// playURL **رابطُ المتجر — ولا يُقبل إلّا رابطَ متجرٍ آمناً.**
//
// **وقيمةٌ في الإعدادات تكتبها يدٌ** — **ويدٌ تخطئ**: **`javascript:` أو
// `http:` أو عنوانٌ آخرُ يصير زرّاً في صفحةٍ عامّة.** **فيُشترَط
// `https://play.google.com/`** — **وهي القناةُ المُعلَنةُ وحدَها.**
func playURL(ctx context.Context, st Store, a App) string {
	raw := strings.TrimSpace(st.GetString(ctx, PlayURLKey(a.Key)))
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "https://play.google.com/") {
		return ""
	}
	return raw
}

// artifact **الأثرُ على القرص وبصمتُه** — أو لا شيء.
//
// **وتنزيلُ الزبون المباشرُ مقفَلٌ هنا لا في الشاشة**: **من أقفله في
// الشاشة وحدَها تركه مكشوفاً بمسارٍ مباشر.**
func artifact(ctx context.Context, st Store, a App) (int64, string, bool) {
	if a.Key == "customer" && !CustomerDirectAllowed {
		return 0, "", false
	}
	name := strings.TrimSpace(st.GetString(ctx, ApkKey(a.Key)))
	if name == "" {
		return 0, "", false
	}
	// **والاسمُ يُقصَّر إلى قاعدته** — **ولا يخرج من المجلَّد ولو حُشي
	// بمسارٍ في القاعدة.**
	path := filepath.Join(st.ArtifactDir(), filepath.Base(name))
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() || fi.Size() == 0 {
		return 0, "", false
	}
	sha, err := sumOf(path, fi)
	if err != nil || sha == "" {
		return 0, "", false
	}
	return fi.Size(), sha, true
}

// ══════════════════════════════════════════════════════════════════════
// **البصمةُ محسوبةٌ لا مكتوبة** (`DLC`، ٢٠٢٦-٠٩-١٣)
// ══════════════════════════════════════════════════════════════════════
//
// **وإعدادٌ يحمل بصمةً تكتبه يدٌ يكذب** — **ومن بدّل الملفَّ ونسي
// البصمةَ عرض وصفاً لملفٍّ آخر.** **فتُحسب من القرص.**
//
// **وتُخبَّأ بمفتاحٍ من (الاسم · الحجم · وقت التعديل)** — **فملفٌّ
// بُدّل يُقرأ من جديدٍ بلا أن يُسأل أحد**، **وملفٌّ لم يتبدّل لا
// يُقرأ مئةَ ميغابايتٍ في كلّ زيارة.**
var (
	sumMu    sync.Mutex
	sumCache = map[string]string{}
)

func sumOf(path string, fi os.FileInfo) (string, error) {
	ck := path + "|" + strconv.FormatInt(fi.Size(), 10) + "|" +
		strconv.FormatInt(fi.ModTime().UnixNano(), 10)
	sumMu.Lock()
	if v, ok := sumCache[ck]; ok {
		sumMu.Unlock()
		return v, nil
	}
	sumMu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	sum := hex.EncodeToString(h.Sum(nil))

	sumMu.Lock()
	// **وخبيئةٌ تنمو بلا حدٍّ تسرّب ذاكرة** — **وأربعةُ آثارٍ لا أكثر،
	// فحدٌّ صغيرٌ يكفي ويُفرَغ عند تجاوزه.**
	if len(sumCache) > 32 {
		sumCache = map[string]string{}
	}
	sumCache[ck] = sum
	sumMu.Unlock()
	return sum, nil
}

// FileName **اسمُ الملفّ الذي يجده من نزّله — يقول ما هو.**
//
// **والمخزَّنُ معرّفٌ عشوائيّ** — **ومن نزّله وجده باسمٍ لا يقول شيئاً**
// (وهي عِلّةُ `rahalgo.apk` القائمة: **أربعةُ تطبيقاتٍ باسمٍ واحد**).
//
// **وتُوسَم الهويّةُ في الاسم**: النسخةُ إن عُرفت، **وإلّا فأوّلُ بصمته**
// — **فملفّان مختلفان لا يتشابهان في مجلَّد التنزيل.**
func FileName(p Public) string {
	tag := strings.TrimSpace(p.Version)
	if tag == "" && len(p.SHA256) >= 12 {
		tag = p.SHA256[:12]
	}
	if tag == "" {
		return "rahalgo-" + p.Key + ".apk"
	}
	return "rahalgo-" + p.Key + "-" + safeTag(tag) + ".apk"
}

// safeTag **وسمٌ يصلح في اسم ملفٍّ** — حروفٌ وأرقامٌ ونقطةٌ وشُرطة.
//
// **والنسخةُ نصٌّ تكتبه يد** — **ويدٌ تكتب مسافةً أو خطّاً مائلاً**،
// **واسمُ ملفٍّ فيه خطٌّ مائلٌ في ترويسة `Content-Disposition` عطبٌ.**
func safeTag(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z',
			r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}
