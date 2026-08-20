package notify

// مُرسِلُ الرسائل النصّية — بوّابةٌ عامّة تُضبَط بالبيئة.
//
// ## لماذا عامّة ولمزوّدٍ بعينه
//
// مزوّدو الرسائل في سوريا يتغيّرون، وكثيرٌ من البوّابات العالمية لا تصلها.
// **وربطُ المشروع بمزوّدٍ واحد في الشيفرة يعني تعديلَ شيفرةٍ ونشراً كلَّما
// تغيّر المزوّد** — وقد يتغيّر لأسبابٍ لا يملكها أحد.
//
// فالبوّابة هنا **قالبٌ يُملأ**: عنوانٌ وطريقةٌ وجسمٌ فيه `{phone}` و`{text}`،
// وترويسةُ استيثاقٍ اختيارية. ومن يبدّل مزوّده يبدّل ثلاثة متغيّرات بيئة.
//
// ## ولماذا لا في الإعدادات
//
// مفتاحُ المزوّد **سرّ**. وجدولُ `app_settings` تقرؤه نقطةُ `/admin/settings`
// المتاحة للأدمن والعمليات والمالية — **فوضعُه هناك يُسلّمه لثلاثة أدوار لا
// شأن لاثنين منها به**. والأسرارُ في البيئة كـ`JWT_SECRET`.
//
// ## والفشلُ يُقال ولا يُبتلَع
//
// بوّابةٌ غيرُ مضبوطة أو لا تستجيب تُرجع خطأً صريحاً. **ورسالةٌ ابتُلعت أسوأ
// من رسالةٍ لم تُرسَل**: الأولى يظنّ صاحبُها أنها وصلت، فينتظر المطعمُ طلباً
// لا يعلم به وينتظر الزبونُ طعاماً لا يُطبخ.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf16"
)

var ErrSMSNotConfigured = errors.New("notify: بوّابة الرسائل غير مضبوطة")

// SMSConfig ضبطُ البوّابة — كلُّه من البيئة.
type SMSConfig struct {
	// URL عنوان البوّابة. يقبل `{phone}` و`{text}` للمزوّدين الذين يأخذونهما
	// في المسار أو في سلسلة الاستعلام.
	URL string
	// Method الافتراضي POST.
	Method string
	// Body قالبُ الجسم — يقبل `{phone}` و`{text}` و`{text_hex}`.
	// فارغٌ يعني بلا جسم.
	Body string
	// ContentType الافتراضي application/json.
	ContentType string
	// AuthHeader ترويسةُ الاستيثاق كاملةً، مثل: `Authorization: Bearer xxx`.
	AuthHeader string
	// Sender اسمُ المُرسِل إن طلبه المزوّد — يُتاح في القالب كـ`{sender}`.
	Sender string
}

type SMSSender struct {
	cfg    SMSConfig
	client *http.Client
	logger *slog.Logger
}

func NewSMSSender(cfg SMSConfig, logger *slog.Logger) *SMSSender {
	if cfg.Method == "" {
		cfg.Method = http.MethodPost
	}
	if cfg.ContentType == "" {
		cfg.ContentType = "application/json"
	}
	return &SMSSender{
		cfg: cfg,
		// مهلةٌ قصيرة: بوّابةٌ بطيئة تُجمّد شاشةَ الموظّف وهو ينتظر ردّاً
		client: &http.Client{Timeout: 15 * time.Second},
		logger: logger,
	}
}

// Configured هل البوّابة جاهزة؟ يُسأل قبل عرض الزرّ.
func (s *SMSSender) Configured() bool { return s != nil && s.cfg.URL != "" }

// ══════════════════════════════════════════════════════════════════════
// **والعربيّةُ لا تمرّ نصّاً عند كثيرٍ من البوّابات**
// ══════════════════════════════════════════════════════════════════════
//
// (قِيس ٢٠٢٦-٠٨-٢٠ عند مسح المزوّدين الذين يخدمون سوريا.)
//
// **معيارُ الرسائل القصيرة يعرف أبجديّتين**: GSM-7 لِلاتينيّة،
// **وUCS-2 لكلّ ما عداها.** والعربيّةُ في الثانية، **وكثيرٌ من
// البوّابات تطلبها ستّةَ عشرَ نظاماً لا حروفاً** — `UTF-16BE` مكتوبةً
// بالستّةَ عشر.
//
// **ومن أرسل «رمز التحقق» نصّاً خامّاً إليها وصلت علاماتِ استفهام** —
// **ولا خطأَ ولا سجلّ**: البوّابةُ تردّ ٢٠٠ والرسالةُ تصل ممسوخة.
//
// **فالقالبُ يقبل الشكلين**: `{text}` لمن يقبل النصّ، **و`{text_hex}`
// لمن يطلب الترميز** — ومن بدّل مزوّدَه بدّل قالبَه ولا يُمسّ كود.
//
// **ولا يُخمَّن أيُّهما**: مكتوبٌ في وثيقة كلّ مزوّد.

// fill **يملأ القالب** — و`esc` تهرّب النصَّ بحسب موضعه.
//
// **والنصُّ الخامُّ يُمرَّر لا المهرَّب**: `{text_hex}` تُحسب منه هو،
// **وحسابُها من نصٍّ هُرِّب لِلJSON يُدخل شرطةً مائلةً في الترميز**
// فتصل الرسالةُ ممسوخة. **وهو خطأٌ لا يظهر في العربيّة** — لا محرفَ
// فيها يُهرَّب — **فيبقى نائماً حتّى يكتب المالكُ علامةَ اقتباسٍ في
// قالبه.**
func (s *SMSSender) fill(tpl, phone, text string, esc func(string) string) string {
	r := strings.NewReplacer(
		"{phone}", phone,
		"{text}", esc(text),
		"{text_hex}", utf16BEHex(text),
		"{sender}", s.cfg.Sender,
	)
	return r.Replace(tpl)
}

// utf16BEHex **النصُّ بترميز `UTF-16BE` مكتوباً بالستّةَ عشر.**
//
// **والحرفُ خارجَ المستوى الأساسيّ يصير زوجاً بديلاً** — والوجهُ
// المبتسمُ أربعُ خاناتٍ لا اثنتان، **و`rune` واحدةٌ تُكتب أربعةَ
// بايتات.** ولا عربيّةَ خارجَ المستوى الأساسيّ، **لكنّ قالبَ المالك
// قد يحمل رمزاً.**
func utf16BEHex(s string) string {
	var b strings.Builder
	for _, u := range utf16.Encode([]rune(s)) {
		fmt.Fprintf(&b, "%04X", u)
	}
	return b.String()
}

// SendText يُرسل رسالةً نصّية.
func (s *SMSSender) SendText(ctx context.Context, phone, text string) error {
	if !s.Configured() {
		return ErrSMSNotConfigured
	}

	var body io.Reader
	if s.cfg.Body != "" {
		body = bytes.NewBufferString(s.fill(s.cfg.Body, phone, text, jsonEscape))
	}
	req, err := http.NewRequestWithContext(ctx, s.cfg.Method,
		s.fill(s.cfg.URL, phone, text, urlEscape), body)
	if err != nil {
		return fmt.Errorf("notify: بناء طلب الرسالة: %w", err)
	}
	if s.cfg.Body != "" {
		req.Header.Set("Content-Type", s.cfg.ContentType)
	}
	if k, v, ok := strings.Cut(s.cfg.AuthHeader, ":"); ok {
		req.Header.Set(strings.TrimSpace(k), strings.TrimSpace(v))
	}

	res, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("notify: بوّابة الرسائل لا تستجيب: %w", err)
	}
	defer res.Body.Close()
	// أوّلُ ٥١٢ بايت من الردّ في السجلّ عند الفشل: رسالةُ المزوّد هي ما يشرح
	// السبب، وبلا قراءتها يبقى «فشل الإرسال» بلا جواب.
	if res.StatusCode < 200 || res.StatusCode > 299 {
		snippet, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("notify: بوّابة الرسائل ردّت %d: %s",
			res.StatusCode, strings.TrimSpace(string(snippet)))
	}
	return nil
}

// jsonEscape يهرّب النصّ ليصلح داخل سلسلة JSON — الرسالة عربيةٌ وفيها أسطر.
func jsonEscape(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\n", `\n`,
		"\r", `\r`,
		"\t", `\t`,
	)
	return r.Replace(s)
}

// urlEscape للمزوّدين الذين يأخذون النصّ في سلسلة الاستعلام.
func urlEscape(s string) string {
	var b strings.Builder
	for _, r := range []byte(s) {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == '~' {
			b.WriteByte(r)
			continue
		}
		fmt.Fprintf(&b, "%%%02X", r)
	}
	return b.String()
}
