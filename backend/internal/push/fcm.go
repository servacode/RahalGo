package push

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ══════════════════════════════════════════════════════════════════════
// **ناقلُ FCM — بلا حزمةٍ جديدةٍ في المشروع**
// ══════════════════════════════════════════════════════════════════════
//
// **الطريقُ المعتاد `firebase.google.com/go`** — وهي تجرّ عشراتِ الحزم
// (عميلَ جوجل السحابيَّ كلَّه وgRPC وما يتبعه). **والمشروعُ اليومَ إحدى
// عشرةَ تبعيّةً مباشرة**، وهذا اختيارٌ لا صدفة.
//
// **وما نحتاجه منها شيئان اثنان:**
//
// **الأوّل** رمزُ وصولٍ من حساب الخدمة — **وهو `JWT` موقَّعٌ بمفتاحٍ
// خاصٍّ يُبادَل برمز**، والمشروعُ يوقّع `JWT` منذ اليوم الأوّل
// (`golang-jwt/jwt/v5`).
//
// **والثاني** نداءُ `POST` واحدٌ بـ`JSON` — و`net/http` تكفيه.
//
// **فالحزمةُ كلُّها لأجل مئةِ سطر.** ولو احتجنا يوماً ما فيها أكثرَ من
// هذا، **تُضاف حينها وقد عرفنا لماذا.**

const (
	googleTokenURL = "https://oauth2.googleapis.com/token"
	fcmScope       = "https://www.googleapis.com/auth/firebase.messaging"
	// **مهلةُ الرمز ساعة، ويُجدَّد قبلها بخمس دقائق** — **ورمزٌ ينتهي بين
	// الفحص والنداء يُسقط الإشعار.**
	tokenSkew = 5 * time.Minute
)

// serviceAccount **ما يلزمنا من ملفّ حساب الخدمة** — لا كلَّ حقوله.
type serviceAccount struct {
	ProjectID  string `json:"project_id"`
	ClientMail string `json:"client_email"`
	PrivateKey string `json:"private_key"`
	TokenURI   string `json:"token_uri"`
}

// FCM ناقلُ أندرويد.
type FCM struct {
	sa     serviceAccount
	key    *rsa.PrivateKey
	http   *http.Client
	logger *slog.Logger

	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

// NewFCM يبني الناقلَ من ملفّ حساب الخدمة — **أو لا شيءَ إن لم يُضبط.**
//
// **ولا يسقط الإقلاعُ لغياب المفتاح**: المنصّةُ تعمل بلا دفعٍ اليوم،
// **وجهازُ المطوّر لا مفتاحَ فيه.** أمّا **مفتاحٌ موجودٌ ومعطوبٌ فيُصرّح
// به** — سكوتٌ عليه يعني منصّةً تظنّ أنّها تُرسل وهي لا تفعل.
func NewFCM(logger *slog.Logger) (*FCM, error) {
	raw, err := credentialsJSON()
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	var sa serviceAccount
	if err := json.Unmarshal(raw, &sa); err != nil {
		return nil, fmt.Errorf("الدفع: ملفُّ حساب الخدمة ليس JSON صالحاً: %w", err)
	}
	if sa.ProjectID == "" || sa.ClientMail == "" || sa.PrivateKey == "" {
		return nil, errors.New("الدفع: ملفُّ حساب الخدمة ناقصٌ — يلزم project_id وclient_email وprivate_key")
	}
	key, err := parseRSAKey(sa.PrivateKey)
	if err != nil {
		return nil, err
	}
	if sa.TokenURI == "" {
		sa.TokenURI = googleTokenURL
	}
	logger.Info("الدفع: FCM مهيّأ", "project", sa.ProjectID)
	return &FCM{
		sa:     sa,
		key:    key,
		http:   &http.Client{Timeout: 15 * time.Second},
		logger: logger,
	}, nil
}

// credentialsJSON **من ملفٍّ أو من متغيّرٍ مباشرةً.**
//
// **والاثنان لأنّ بيئتَي التشغيل مختلفتان**: على الجهاز ملفٌّ بجانب
// المشروع، **وعلى الاستضافة لا قرصَ دائماً** — فيُلصَق المحتوى في متغيّر.
func credentialsJSON() ([]byte, error) {
	if inline := strings.TrimSpace(os.Getenv("FCM_CREDENTIALS_JSON")); inline != "" {
		return []byte(inline), nil
	}
	path := strings.TrimSpace(os.Getenv("FCM_CREDENTIALS_FILE"))
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("الدفع: تعذّرت قراءةُ %s: %w", path, err)
	}
	return raw, nil
}

func parseRSAKey(pemKey string) (*rsa.PrivateKey, error) {
	// **الأسطرُ الجديدةُ تصل `\n` حرفيّةً حين تُلصق في متغيّر بيئة** —
	// **ومفتاحٌ بسطرٍ واحدٍ لا يُفكّ**، والرسالةُ تقول «PEM غيرُ صالح»
	// فيبحث القارئُ في المفتاح وهو سليم.
	pemKey = strings.ReplaceAll(pemKey, `\n`, "\n")
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, errors.New("الدفع: المفتاحُ الخاصُّ ليس PEM صالحاً")
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			return rk, nil
		}
		return nil, errors.New("الدفع: المفتاحُ ليس RSA")
	}
	// **وحساباتُ الخدمة القديمةُ بصيغة PKCS#1** — تُقبل ولا تُرفض.
	k, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("الدفع: تعذّر فكُّ المفتاح الخاصّ: %w", err)
	}
	return k, nil
}

func (f *FCM) Platform() string { return PlatformAndroid }

// accessToken رمزُ وصولٍ صالح — **يُجدَّد عند الحاجة وحدَها.**
//
// **ورمزٌ يُطلب مع كلّ إشعارٍ نداءٌ شبكيٌّ زائدٌ يُضاعف زمنَ الوصول**،
// ويستهلك حصّةَ جوجل بلا سبب. **وهو صالحٌ ساعةً.**
func (f *FCM) accessToken(ctx context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.token != "" && time.Now().Before(f.tokenExp.Add(-tokenSkew)) {
		return f.token, nil
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   f.sa.ClientMail,
		"scope": fcmScope,
		"aud":   f.sa.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(f.key)
	if err != nil {
		return "", fmt.Errorf("الدفع: تعذّر توقيعُ طلب الرمز: %w", err)
	}

	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {signed},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.sa.TokenURI,
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := f.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("الدفع: تعذّر بلوغُ خادم الرموز: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<16))
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("الدفع: خادمُ الرموز ردّ %d: %s", res.StatusCode, truncate(body))
	}
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.AccessToken == "" {
		return "", errors.New("الدفع: ردُّ خادم الرموز بلا رمزِ وصول")
	}
	f.token = out.AccessToken
	f.tokenExp = now.Add(time.Duration(out.ExpiresIn) * time.Second)
	return f.token, nil
}

// Send يرسل الرسالةَ إلى كلّ رمز — **ويعيد ما رفضته جوجل نهائيّاً.**
//
// ══════════════════════════════════════════════════════════════════════
// **ولماذا نداءٌ لكلّ رمزٍ لا نداءٌ واحدٌ للجميع**
// ══════════════════════════════════════════════════════════════════════
//
// **`HTTP v1` لا تقبل قائمةَ رموزٍ في نداءٍ واحد** — هذا رحل مع `legacy`
// التي أُغلقت. **والبديلُ المتاح `sendEach`**، وهو ما نفعله.
//
// **والعددُ صغيرٌ عمليّاً**: حسابٌ له جهازٌ أو جهازان، **لا مئة.**
func (f *FCM) Send(ctx context.Context, tokens []string, msg Message) ([]string, error) {
	if len(tokens) == 0 {
		return nil, nil
	}
	token, err := f.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", f.sa.ProjectID)

	var dead []string
	var firstErr error
	for _, dev := range tokens {
		gone, err := f.sendOne(ctx, endpoint, token, dev, msg)
		if gone {
			dead = append(dead, dev)
			continue
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return dead, firstErr
}

func (f *FCM) sendOne(ctx context.Context, endpoint, auth, device string, msg Message) (gone bool, err error) {
	priority := "normal"
	if msg.Urgent {
		priority = "high"
	}
	payload := map[string]any{
		"message": map[string]any{
			"token": device,
			// ══════════════════════════════════════════════════════
			// **بياناتٌ خالصةٌ — ولا حقلَ `notification`**
			// ══════════════════════════════════════════════════════
			//
			// (قِيس على جهاز المالك ٢٠٢٦-٠٨-٢٣: الإشعارُ يصل **بلا
			//  صوتٍ ولا اهتزاز** على قناة فايربيس الاحتياطيّة.)
			//
			// **وكان يُرسَل الحقلان معاً** بنيّةٍ صحيحة: «النظامُ يعرض
			// الإشعارَ ولو كان التطبيقُ مقتولاً». **وثمنُها أنّ
			// الرسالةَ لا تصل التطبيقَ وهو في الخلفيّة**: يرسمها
			// النظامُ بنفسه، **فلا يملك التطبيقُ قناةً ولا صوتاً ولا
			// وجهةَ فتح.**
			//
			// **ورسالةُ البيانات تُسلَّم إلى `onMessageReceived` في
			// الأحوال الثلاثة** — مقدّمةً وخلفيّةً ومقتولاً — **فيرسم
			// التطبيقُ إشعارَه بقناته وصوته، ويفتح على الطلب الصحيح.**
			//
			// **والعنوانُ والنصُّ ينتقلان إلى الحمولة** — ولا يضيعان.
			"data": withText(msg),
			"android": map[string]any{
				"priority": priority,
				// **ومهلةُ الحياة قصيرة**: طلبٌ عمرُه ساعةٌ لا معنى لتسليمه
				// بعد يوم — **وإشعارٌ متأخّرٌ عن حدثٍ انتهى يُربك ولا يُفيد.**
				"ttl": "3600s",
				// **ولا `notification` هنا أيضاً** — وهي تعمل مع
				// الحقل الأعلى وحدَه. **والقناةُ يختارها التطبيقُ
				// الآن** (`ui/PushChannels.kt`)، **وأسماؤها عقدٌ
				// بين الطرفين**: `rahalgo_urgent` و`rahalgo_default`.
				//
				// **ووُجد أنّهما كانا يفترقان** (٢٠٢٦-٠٨-٢٣): المحرّكُ
				// يرسل `rahalgo_urgent` والتطبيقُ ينشئ `rahalgo_order`
				// — **فيرتدّ أندرويد إلى قناةٍ صامتة.**
			},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+auth)
	req.Header.Set("Content-Type", "application/json")

	res, err := f.http.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<16))
	switch {
	case res.StatusCode < 300:
		return false, nil
	// ══════════════════════════════════════════════════════════════════
	// **٤٠٤ موتُ رمز — و٤٠٣ ليست**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قِيس ٢٠٢٦-٠٨-١٤ على جهازٍ حقيقيّ: **جهازٌ واحدٌ مسجَّل، وصفرٌ
	//  قُبل** — والرمزُ يُحذف بعد كلّ تسجيل، فلا يرنّ شيءٌ أبدا.)
	//
	// **كانتا سواءً هنا** — والفرقُ بينهما كلُّ شيء:
	//
	//	٤٠٤  الرمزُ لا يعرفه المشروع    ← جهازٌ حُذف منه التطبيق · يُحذف
	//	٤٠٣  البابُ مغلقٌ في المشروع    ← واجهةٌ لم تُفعَّل أو حسابُ خدمةٍ
	//	                                   بلا صلاحيّة · **يُصلَح لا يُحذف**
	//
	// **وحذفُ الرمز على ٤٠٣ يُخفي العطبَ ويضاعفه**: يُمحى رمزُ كلّ سائقٍ
	// في المنصّة واحداً بعد واحد، **ثمّ لا يبقى ما يُرسَل إليه** — فيبدو
	// أنّ لا أجهزةَ أصلاً، **والسببُ سطرٌ في إعدادات غوغل.**
	case res.StatusCode == http.StatusNotFound:
		return true, nil
	// **و٤٠٠ قد تكون رمزاً مشوَّهاً** — تُقرأ من نصّ الردّ.
	case res.StatusCode == http.StatusBadRequest && bytes.Contains(raw, []byte("INVALID_ARGUMENT")):
		return true, nil
	default:
		return false, fmt.Errorf("FCM ردّ %d: %s", res.StatusCode, truncate(raw))
	}
}

// channelFor **قناةُ الإشعار في أندرويد** — والاسمُ عقدٌ مع التطبيق.
//
// **ولو اختلف حرفٌ لَظهر الإشعارُ بلا صوتٍ ولا اهتزاز** بلا أيّ خطأ:
// أندرويد يُنشئ قناةً افتراضيّةً صامتةً لِما لا يعرف.
// withText **يضمّ العنوانَ والنصَّ إلى الحمولة.**
//
// **ولا حقلَ `notification` بعد اليوم** — فلو ضاع العنوانُ لَوصل
// إشعارٌ فارغ. **والتطبيقُ يقرؤهما من `title` و`body` في البيانات.**
//
// **والقيمُ نصوصٌ كلُّها** — FCM ترفض حمولةً فيها رقمٌ أو قيمةُ صدق.
func withText(msg Message) map[string]string {
	out := make(map[string]string, len(msg.Data)+3)
	for k, v := range msg.Data {
		out[k] = v
	}
	out["title"] = msg.Title
	out["body"] = msg.Body
	// **والعجلةُ تُقال للتطبيق** — فيختار قناتَه بنفسه، **ولا يستنتجها
	// من نوع الخبر** فيفترق حكمُه عن حكم المحرّك.
	if msg.Urgent {
		out["urgent"] = "1"
	}
	return out
}

func channelFor(urgent bool) string {
	if urgent {
		return "rahalgo_urgent"
	}
	return "rahalgo_default"
}

// truncate **يقصّ ردَّ الخطأ** — **وردٌّ بطول صفحةٍ في سطر سجلٍّ يُخفي ما بعده.**
func truncate(b []byte) string {
	const max = 300
	if len(b) > max {
		return string(b[:max]) + "…"
	}
	return string(b)
}
