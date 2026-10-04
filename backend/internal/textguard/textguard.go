// Package textguard **حارسُ النصوص — كلُّ ما يكتبه إنسانٌ يمرّ من هنا.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٣: «لازم يكون عندنا فلترة لحقول النصوص — مسبّات،
//
//	شتائم، كلام بذيء، أكواد اختراق، رموز غريبة — بكلّ حقول الإدخال».)
//
// # ما يفعله بكلّ نصّ
//
//  1. **ينظّف**: يُسقط المحارفَ الخفيّةَ التي تقلب الاتّجاه أو تخبّئ حروفاً،
//     ووسومَ الصفحات (`<script>` وأخواتها)، ويطوي الفراغات، **ويقصّ التكرارَ
//     العبثيّ** (حرفٌ واحدٌ أكثرَ من عشر مرّاتٍ يصير عشراً).
//  2. **يحدّ الطول** بالحروف لا بالبايتات — `text_too_long`.
//  3. **يمنع الإيموجي في الاسم والعنوان** — `text_bad_chars`.
//  4. **يمسك الشتيمة**: في الاسم والعنوان والملاحظات **يرفض** (`text_offensive`)،
//     وفي الحديث والشكاوى **يُخفيها «***»** ويقولها لمن نادى ليُنبّه الإدارة.
//
// # ولماذا مكانٌ واحد
//
// **كان لكلّ بابٍ حدُّه وتنظيفُه** — والحديثُ وحدَه يُنقّى من المحارف الخفيّة
// (`comms/clean.go`)، **والاسمُ والعنوانُ يُخزَّنان كما وصلا.** **وقاعدةٌ في
// موضعٍ واحدٍ تُطبَّق حيث نُوديت وتُنسى حيث لم تُنادَ** — فالحارسُ واحدٌ،
// **والبابُ يناديه بسطرٍ واحد.**
package textguard

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// Policy **ما يُفعل بالحقل** — نوعُه يقرّر ما يُرفض وما يُخفى.
type Policy int

const (
	// Name اسمُ شخصٍ أو متجرٍ أو صنف — سطرٌ واحد، **لا إيموجي، والشتيمةُ تُرفض.**
	Name Policy = iota
	// Address عنوانٌ أو جزؤه — كالاسم.
	Address
	// Notes ملاحظةٌ أو وصفٌ أو نصُّ طلب — أسطر، **والإيموجي مقبول، والشتيمةُ تُرفض.**
	Notes
	// Chat رسالةُ حديث — **الشتيمةُ تُخفى «***» ولا تُرفض**: الحديثُ يصل.
	Chat
	// Complaint شكوى أو ردٌّ عليها — كالحديث: **شكوى الغاضب تصل.**
	Complaint
)

func (p Policy) singleLine() bool { return p == Name || p == Address }
func (p Policy) noEmoji() bool    { return p == Name || p == Address }
func (p Policy) masks() bool      { return p == Chat || p == Complaint }
func (p Policy) rejectsBad() bool { return !p.masks() }
func (p Policy) String() string {
	return [...]string{"name", "address", "notes", "chat", "complaint"}[p]
}
func (p Policy) valid() bool       { return p >= Name && p <= Complaint }
func (p Policy) maskToken() string { return "***" }

// Field **حقلٌ واحد** — اسمُه (يعود في تفاصيل الخطأ)، وقيمتُه تُنظَّف في موضعها.
type Field struct {
	Name   string
	Value  *string
	Max    int
	Policy Policy
	// TooLong **خطأُ الطول إن كان للباب خطؤه القائم** — `item_text_too_long`
	// في القائمة. **وفارغُه `text_too_long`.**
	TooLong *httpx.AppError
}

// MaxRepeat **أكثرُ ما يتكرّر حرفٌ واحدٌ متتالياً** — وما زاد يُقصّ إليه.
const MaxRepeat = 10

var (
	// ErrTooLong **أطولُ من حدّه.**
	ErrTooLong = httpx.NewError(http.StatusBadRequest, "text_too_long", "errors.text_too_long")
	// ErrOffensive **لفظٌ مسيء** في حقلٍ يُرفض فيه.
	ErrOffensive = httpx.NewError(http.StatusBadRequest, "text_offensive", "errors.text_offensive")
	// ErrBadChars **رموزٌ لا تُقبل** — إيموجي في اسمٍ أو عنوان.
	ErrBadChars = httpx.NewError(http.StatusBadRequest, "text_bad_chars", "errors.text_bad_chars")
)

// withField **الخطأُ ومعه اسمُ الحقل** — ليعرف التطبيقُ أيَّها. **والتفاصيلُ
// نصوصٌ لا أرقام**: نموذجُ الخطأ في التطبيق `Map<String, String>`.
func withField(base *httpx.AppError, field string, extra map[string]any) error {
	e := *base
	e.Details = map[string]any{"field": field}
	for k, v := range extra {
		e.Details[k] = v
	}
	return &e
}

// Guard **الحارس** — وقائمةُ الألفاظ تُقرأ من الإعدادات عند كلّ نداء.
//
// **و`nil` حارسٌ صالح** — يعمل بالقائمة المدمجة. **فخادمٌ في اختبارٍ لم
// يُركَّب له حارسٌ لا ينهار بمؤشّرٍ فارغ.**
type Guard struct {
	read func(ctx context.Context) string
}

// New حارسٌ يقرأ قائمتَه من `read` — وفارغُها يعني القائمةَ المدمجة.
func New(read func(ctx context.Context) string) *Guard { return &Guard{read: read} }

func (g *Guard) filter(ctx context.Context) *Filter {
	if g == nil || g.read == nil {
		return defaultFilter
	}
	raw := strings.TrimSpace(g.read(ctx))
	if raw == "" {
		return defaultFilter
	}
	return NewFilter(ParseList(raw))
}

// Apply **يمرّ على الحقول بالترتيب** — ينظّف كلَّ قيمةٍ في موضعها، ويردّ
// أوّلَ خطأ، **أو الألفاظَ التي أُخفيت** في حقول الحديث والشكوى.
func (g *Guard) Apply(ctx context.Context, fields ...Field) (masked []string, err error) {
	f := g.filter(ctx)
	for _, fd := range fields {
		if fd.Value == nil || !fd.Policy.valid() {
			continue
		}
		// **والطولُ يُقاس قبل طيّ التكرار** — وإلّا صار اسمٌ من مئةِ حرفٍ مكرَّرٍ
		// عشرةً فقُبل، **والحدُّ يقول ما كُتب لا ما بقي بعد التنظيف.**
		if fd.Max > 0 && utf8.RuneCountInString(strings.TrimSpace(*fd.Value)) > fd.Max {
			base := ErrTooLong
			if fd.TooLong != nil {
				base = fd.TooLong
			}
			return nil, withField(base, fd.Name, map[string]any{"max": strconv.Itoa(fd.Max)})
		}
		v := Clean(*fd.Value, !fd.Policy.singleLine())
		if fd.Policy.noEmoji() && HasEmoji(v) {
			return nil, withField(ErrBadChars, fd.Name, nil)
		}
		if fd.Policy.rejectsBad() {
			if w := f.Find(v); w != "" {
				return nil, withField(ErrOffensive, fd.Name, nil)
			}
		} else {
			out, words := f.Mask(v, fd.Policy.maskToken())
			v = out
			masked = append(masked, words...)
		}
		*fd.Value = v
	}
	return masked, nil
}

// ══════════════════════════════════════════════════════════════════════
// **التنظيف**
// ══════════════════════════════════════════════════════════════════════

// tagRe **وسمُ صفحة** — `<script>` و`<img onerror=…>` و`</b>`. **ولا يُترك
// نصفُه**: ما بين القوسين يُسقط كلُّه.
var tagRe = regexp.MustCompile(`<[^<>]{0,500}>`)

// Clean **ينظّف النصّ** — يُسقط الخفيَّ والوسومَ ويطوي الفراغَ والتكرار.
//
// `multiline` **يُبقي السطرَ الجديد** (ملاحظةٌ وحديث) — **والاسمُ سطرٌ واحد.**
func Clean(s string, multiline bool) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n':
			if multiline {
				b.WriteRune('\n')
			} else {
				b.WriteRune(' ')
			}
		case r == '\r':
			// يُسقط — `\r\n` سطرٌ واحد.
		case r == '\t':
			b.WriteRune(' ')
		case isInvisible(r):
			// يُسقط
		case r == utf8.RuneError:
			// بايتاتٌ فاسدة — تُسقط.
		default:
			b.WriteRune(r)
		}
	}
	out := tagRe.ReplaceAllString(b.String(), "")
	out = collapseRepeats(out, MaxRepeat)
	// **والفراغاتُ تُطوى**: مسافاتٌ متتاليةٌ واحدة، وأكثرُ من سطرين فارغين سطران.
	lines := strings.Split(out, "\n")
	for i, l := range lines {
		lines[i] = strings.Join(strings.Fields(l), " ")
	}
	out = strings.Join(lines, "\n")
	for strings.Contains(out, "\n\n\n") {
		out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(out)
}

// isInvisible **محرفٌ لا يُرى ويغيّر ما يُرى** — أو محرفُ تحكّمٍ لا مكان له في نصّ.
//
// **وتُكتب بأرقامها لا بأشكالها**: محرفٌ لا يُرى مكتوبٌ في الشيفرة **لا يراه
// من يقرؤها.**
func isInvisible(r rune) bool {
	switch {
	// **عرضٌ صفريٌّ ووصلٌ وفصلٌ وعلاماتُ الاتّجاه** — تقطّع الكلمة فتمرّ من كلّ فحص.
	case r >= 0x200B && r <= 0x200F:
		return true
	// **قلبُ الاتّجاه وتجاوزُه وتضمينُه** — يُكتب سطرٌ ويُقرأ آخر.
	case r >= 0x202A && r <= 0x202E, r >= 0x2066 && r <= 0x2069:
		return true
	case r == 0xFEFF, r == 0x2060, r == 0x180E, r == 0x00AD:
		return true
	}
	return unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Cf, r)
}

// collapseRepeats **حرفٌ واحدٌ متتالٍ أكثرَ من `max` يُقصّ إليه.**
func collapseRepeats(s string, max int) string {
	var b strings.Builder
	b.Grow(len(s))
	var last rune = -1
	n := 0
	for _, r := range s {
		if r == last {
			n++
		} else {
			last, n = r, 1
		}
		if n <= max {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// HasEmoji **أفي النصّ إيموجي أو رمزٌ مصوَّر؟**
func HasEmoji(s string) bool {
	for _, r := range s {
		switch {
		case r >= 0x1F000 && r <= 0x1FAFF, // الوجوهُ والرموزُ والأعلام
			r >= 0x2600 && r <= 0x27BF, // رموزٌ متفرّقةٌ وزخارف
			r >= 0x2B00 && r <= 0x2BFF, // أسهمٌ ونجوم
			r == 0xFE0F, r == 0x20E3:   // محدِّدُ الصورة ومفتاحُ الرقم
			return true
		case unicode.Is(unicode.So, r):
			return true
		}
	}
	return false
}
