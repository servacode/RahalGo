package media

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **وسائطُ عامّةٌ ووسائطُ شخصيّة** — `D13`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **`http.FileServer(http.Dir(...))` بلا حارس**: **يسرد المجلَّدَ** حين
// لا `index.html`، **ويخدم كلَّ ملفٍّ لأيّ أحدٍ بلا توكن.** (مقيسٌ:
// `/media/` ⇒ `200` وصفحةُ سرد · وإثباتُ تسليمٍ ⇒ `200`.)
//
// # وأصنافُ الوسائط تسعةٌ في القاعدة
//
// **سبعةٌ عامّةٌ بطبيعتها** — شعارُ المتجر والصنفُ والقسمُ واللافتةُ
// وشعارُ المنصّة وخلفيّتا الدخول والموقع. **وخلفيّةُ الدخول تُعرَض قبل
// أن يملك أحدٌ توكناً أصلاً** — **فحجبُها يقفل بابَ الدخول.**
//
// **واثنان شخصيّان**: `avatar` و`delivery_proof`. **وإثباتُ التسليم
// صورةُ بابِ بيتٍ ومعه إحداثيّاتُه ووقتُه.**
//
// # ولماذا التوقيعُ لا الترويسة
//
// **المصادقةُ ترويسةُ `Bearer` وحدَها** (`middleware.go:28`) — **ووسمُ
// `<img>` لا يحمل ترويسة.** **فمن حجب الوسائطَ بالتوكن كسر عرضَ الصور
// في الويب وأندرويد معاً** — وذلك انقطاعٌ لا حماية.
//
// **فالرابطُ الموقَّعُ محدودُ الأجل**: **يُصدره الخادمُ مع الردّ
// المُصرَّح به أصلاً** — من رأى الطلبَ رأى إثباتَه. **والسرُّ في
// الخادم لا في العميل.**

// protectedKinds **الصنفان الشخصيّان** — ولا ثالثَ لهما اليوم.
var protectedKinds = map[string]bool{
	"avatar":         true,
	"delivery_proof": true,
}

// signTTL عمرُ الرابط الموقَّع.
//
// **ستُّ ساعاتٍ**: **الروابطُ تُستهلَك عند الرسم لا بعده** — وصفحةٌ
// تبقى مفتوحةً أطولَ من ذلك تُحدِّث بياناتها فتُوقَّع من جديد.
//
// **وقِصَرٌ مفرطٌ يكسر صفحةً مفتوحة، وطولٌ مفرطٌ يجعل الرابطَ المسروقَ
// مفتاحاً دائماً.**
const signTTL = 6 * time.Hour

var (
	signKeyMu sync.RWMutex
	signKey   []byte
)

// SetSigningKey يضبط سرَّ توقيع الوسائط — **مرّةً عند الإقلاع.**
//
// **وبلا سرٍّ لا يُوقَّع شيءٌ ولا يُقبَل توقيع** — **فالمحميُّ يُحجَب
// كلُّه.** **وسقوطٌ مغلقٌ لا مفتوح.**
func SetSigningKey(secret string) {
	signKeyMu.Lock()
	defer signKeyMu.Unlock()
	if secret == "" {
		signKey = nil
		return
	}
	sum := sha256.Sum256([]byte("rahalgo-media-v1:" + secret))
	signKey = sum[:]
}

func signingKey() []byte {
	signKeyMu.RLock()
	defer signKeyMu.RUnlock()
	return signKey
}

// sign بصمةُ مسارٍ إلى وقتٍ.
func sign(relPath string, exp int64) string {
	k := signingKey()
	if len(k) == 0 {
		return ""
	}
	mac := hmac.New(sha256.New, k)
	mac.Write([]byte(relPath))
	mac.Write([]byte{0})
	mac.Write([]byte(strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

// SignedURL رابطٌ موقَّعٌ محدودُ الأجل لوسيطٍ شخصيّ.
//
// **ويُنادى حيث يُصدَر الرابطُ في ردٍّ مُصرَّحٍ به** — **والتصريحُ وقع
// هناك، فلا يُعاد هنا.**
func SignedURL(relPath string) string {
	if relPath == "" {
		return ""
	}
	exp := time.Now().Add(signTTL).Unix()
	sig := sign(relPath, exp)
	if sig == "" {
		return URLFor(relPath)
	}
	return URLFor(relPath) + "?exp=" + strconv.FormatInt(exp, 10) + "&sig=" + sig
}

// SignedURLPtr كـ`SignedURL` للمؤشّرات — بنمط `URLForPtr`.
func SignedURLPtr(relPath *string) *string {
	if relPath == nil || *relPath == "" {
		return nil
	}
	u := SignedURL(*relPath)
	return &u
}

// verify يفحص توقيعَ طلبٍ.
func verify(relPath string, q url.Values) bool {
	k := signingKey()
	if len(k) == 0 {
		return false
	}
	exp, err := strconv.ParseInt(q.Get("exp"), 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	// **ومقارنةٌ بزمنٍ ثابت** — **ومقارنةُ نصٍّ عاديّةٌ تُسرِّب البصمةَ
	// حرفاً حرفاً لمن قاس الزمن.**
	return hmac.Equal([]byte(q.Get("sig")), []byte(sign(relPath, exp)))
}

// variantSuffix لواحقُ النسخ — `_t` للمصغَّرة و`_480` لعرضٍ معلوم.
var variantSuffix = regexp.MustCompile(`_(t|\d+)$`)

// basePathOf يردّ النسخةَ إلى أصلها.
//
// **والنسخُ لا صفوفَ لها في القاعدة** — **ومصغَّرةُ إثباتٍ إثباتٌ**،
// فمن حرس الأصلَ وترك مصغَّرتَه لم يحرس شيئاً.
func basePathOf(rel string) string {
	dir, file := path.Split(rel)
	ext := path.Ext(file)
	stem := strings.TrimSuffix(file, ext)
	return dir + variantSuffix.ReplaceAllString(stem, "") + ext
}

// IsProtected أالمسارُ لوسيطٍ شخصيّ؟
//
// **ويُسأل الجدولُ لا المسار**: **المسارُ `YYYY/MM/uuid` لا صنفَ فيه**،
// **ولا يُنقَل ملفٌّ قائمٌ لأجل الحراسة** — **ونقلُ ملفّاتٍ تشير إليها
// صفوفٌ بابُ ضياع.**
//
// **والجوابُ يُحفَظ**: **المسارُ وصنفُه لا يتبدّلان بعد الرفع.**
func (s *Service) IsProtected(ctx context.Context, rel string) bool {
	base := basePathOf(rel)
	if v, ok := s.protCache.Load(base); ok {
		return v.(bool)
	}
	var kind string
	err := s.db.QueryRow(ctx, `
		SELECT kind FROM media WHERE path = $1 OR thumb_path = $1 LIMIT 1`,
		base).Scan(&kind)
	if err != nil {
		// ══════════════════════════════════════════════════════════
		// **ومجهولُ النسب يُحجَب**
		// ══════════════════════════════════════════════════════════
		//
		// **ملفٌّ لا صفَّ له لا يُعرَف أعامٌّ هو أم شخصيّ** — **ومن
		// خدمه بحجّة الجهل خدم كلَّ ما تسرّب إلى المجلَّد.**
		//
		// **ولا يُحفَظ هذا الجواب**: **الصفُّ قد يُكتب بعد لحظة**
		// (الملفُّ يُكتب قبل صفِّه في `Save`).
		return true
	}
	prot := protectedKinds[kind]
	s.protCache.Store(base, prot)
	return prot
}
