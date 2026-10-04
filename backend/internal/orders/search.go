package orders

// ══════════════════════════════════════════════════════════════════════
// **بحثُ السجلّ يفهم ما يكتبه الموظّف** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (سجلُّ الطلبات، البند ٥)
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **الهاتفُ محفوظٌ `+963…` والموظّفُ يكتب `09…`** — فقِيس على التجهيز: الرقمُ
// نفسُه لقي ١٣ طلباً بصيغة `+963` وصفراً بصيغة `09…` أو `00963…`. **ورقمُ الطلب
// `١٤١٦` من لوحة مفاتيحٍ عربيّة أو `#1416` يردّ صفراً**، و`%` وحدَه يطابق
// السجلَّ كلَّه.
//
// # وما صار
//
//   - **الأرقامُ الهنديّةُ والفارسيّة تُحوَّل** إلى 0123456789.
//   - **و`#` والمسافاتُ والشرطاتُ والأقواسُ تُحذف** قبل الحكم.
//   - **و`09…` و`00963…` و`963…` تصير `963…`** — ويُطابَق جزءاً من الهاتف.
//   - **والنصُّ اسمُ زبون** — و`%` و`_` حرفان لا أداتا مطابقة.

import (
	"strconv"
	"strings"
)

// SearchTerms **ما يُطابَق به بعد التطبيع** — والفارغُ لا شرطَ له.
type SearchTerms struct {
	// Number **رقمُ الطلب** — وصفرٌ «ليس رقماً».
	Number int64
	// Phone **أرقامُ الهاتف بلا `+`** — `963…` إن بدأ محلّيّاً.
	Phone string
	// Name **اسمُ الزبون مهرَّباً لـ`LIKE`** — `%` و`_` و`\` حروفٌ لا أدوات.
	Name string
}

// foldDigits يحوّل الأرقامَ الهنديّةَ (٠–٩) والفارسيّةَ (۰–۹) إلى 0–9.
func foldDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '٠' && r <= '٩':
			b.WriteRune('0' + (r - '٠'))
		case r >= '۰' && r <= '۹':
			b.WriteRune('0' + (r - '۰'))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// EscapeLike يهرّب `\` و`%` و`_` — **فتُطابَق حروفاً لا أدوات** (مع `ESCAPE '\'`).
func EscapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// NormalizeSearch **يطبّع نصَّ البحث** — رقمُ طلبٍ أو هاتفٌ أو اسم.
func NormalizeSearch(q string) SearchTerms {
	q = strings.TrimSpace(foldDigits(q))
	if q == "" {
		return SearchTerms{}
	}
	compact := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '-', '#', '(', ')', '.', ' ':
			return -1
		}
		return r
	}, q)
	plus := strings.HasPrefix(compact, "+")
	digits := strings.TrimPrefix(compact, "+")
	if digits != "" && strings.Trim(digits, "0123456789") == "" {
		var t SearchTerms
		// **رقمُ الطلب قصيرٌ بلا `+`** — ولا يتجاوز تسعةَ أرقام.
		if !plus && len(digits) <= 9 {
			if n, err := strconv.ParseInt(digits, 10, 64); err == nil && n > 0 {
				t.Number = n
			}
		}
		p := digits
		switch {
		case strings.HasPrefix(p, "00963"):
			p = "963" + p[5:]
		case !plus && strings.HasPrefix(p, "0") && len(p) > 1:
			p = "963" + p[1:]
		}
		t.Phone = p
		return t
	}
	// **وما ليس رقماً اسمٌ** — بعد حذف `#` في أوّله وحدَه.
	return SearchTerms{Name: EscapeLike(strings.TrimPrefix(q, "#"))}
}
