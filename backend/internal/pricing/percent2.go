package pricing

// ══════════════════════════════════════════════════════════════════════
// **نسبةٌ بمنزلتين عشريّتين — بلا عددٍ عشريٍّ عائم** (قرارُ المالك ٢٠٢٦-١٠-١٨)
// ══════════════════════════════════════════════════════════════════════
//
// عمولةُ المنصّة كانت عدداً صحيحاً (١٠٪). **وطلب المالكُ ١٫٥٪** — فصارت
// النسبةُ تُحفظ بجزءٍ من مئةٍ من الواحد بالمئة (`150` = ١٫٥٪).
//
// # ولماذا عددٌ صحيحٌ لا `float64`
//
// **`0.1 + 0.2` لا يساوي `0.3` في الآلة** — ونسبةٌ عائمةٌ تُضرب في ثمنٍ
// تخرج بكسرٍ يُقرَّب مرّةً لأعلى ومرّةً لأسفل. **ودفترٌ يفترق عن الشاشة بليرةٍ
// يُسأل عنه.** فالنسبةُ عددٌ صحيحٌ والقسمةُ صحيحةٌ واتّجاهُ التقريب واحد.
//
// # والنسبُ الصحيحةُ كما كانت تماماً
//
//	قبلُ:  ثمن × ١٠ ÷ ١٠٠
//	الآن:  ثمن × ١٠٠٠ ÷ ١٠٠٠٠
//
// **والقسمتان تبتران نحو الصفر** — فكلُّ متجرٍ على نسبةٍ صحيحةٍ يقبض اليومَ
// ما كان يقبضه أمس ليرةً بليرة.

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// Percent2 **نسبةٌ مئويّةٌ بمنزلتين** — مخزّنةً بأجزاء المئة (`150` = ١٫٥٪).
//
// **وتُقرأ من عمودٍ `numeric(5,2)` وتُكتب إليه مباشرةً** (`ScanNumeric` و
// `NumericValue`) — فلا تمرّ بعددٍ عائمٍ في الطريق.
type Percent2 int64

// ErrBadPercent2 **نسبةٌ لا تُقرأ أو بأكثرَ من منزلتين.**
var ErrBadPercent2 = errors.New("pricing: نسبةٌ غيرُ صالحة — منزلتان عشريّتان على الأكثر")

// Pct نسبةٌ صحيحةٌ بالمئة — `Pct(10)` = ١٠٪.
func Pct(p int64) Percent2 { return Percent2(p * 100) }

// Hundredths أجزاءُ المئة — `150` لـ١٫٥٪.
func (p Percent2) Hundredths() int64 { return int64(p) }

// RoundedPercent **النسبةُ عدداً صحيحاً للعملاء القدامى** — تقريبٌ نصفيٌّ لأعلى.
//
// **ولا يُحسب بها مال** — هي لعرض تطبيقٍ قديمٍ يقرأ `commission_percent`
// عدداً صحيحاً (`Int` في كوتلن) فيسقط فكُّه على `1.5`.
func (p Percent2) RoundedPercent() int64 {
	if p < 0 {
		return -Percent2(-p).RoundedPercent()
	}
	return (int64(p) + 50) / 100
}

// String **بلا أصفارٍ زائدة**: `10` و`1.5` و`2.25`.
func (p Percent2) String() string {
	v := int64(p)
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	whole, frac := v/100, v%100
	switch {
	case frac == 0:
		return fmt.Sprintf("%s%d", sign, whole)
	case frac%10 == 0:
		return fmt.Sprintf("%s%d.%d", sign, whole, frac/10)
	default:
		return fmt.Sprintf("%s%d.%02d", sign, whole, frac)
	}
}

// MarshalJSON **رقمٌ لا نصّ** — `1.5` كما يكتبه المالك.
func (p Percent2) MarshalJSON() ([]byte, error) { return []byte(p.String()), nil }

// UnmarshalJSON يقبل رقماً (`1.5`) أو نصّاً (`"1.5"`) — **ويردّ ما زاد على منزلتين.**
func (p *Percent2) UnmarshalJSON(b []byte) error {
	s := string(bytes.TrimSpace(b))
	if s == "null" {
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	v, err := ParsePercent2(s)
	if err != nil {
		return err
	}
	*p = v
	return nil
}

// ParsePercent2 **يقرأ النصَّ حرفاً حرفاً** — `1.5` و`2.25` و`10` و`-1`.
//
// **ولا يمرّ بـ`ParseFloat`**: `1.555` يُردّ لا يُقرَّب إلى `1.56`، **ومن كتب
// ثلاثَ منازلَ أراد شيئاً لا نحفظه.**
func ParsePercent2(s string) (Percent2, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrBadPercent2
	}
	neg := false
	if s[0] == '-' || s[0] == '+' {
		neg = s[0] == '-'
		s = s[1:]
	}
	whole, frac, hasDot := strings.Cut(s, ".")
	if whole == "" {
		whole = "0"
	}
	if hasDot && frac == "" {
		return 0, ErrBadPercent2
	}
	// **والأصفارُ في الذيل لا تُعدّ منزلة** — `1.50` و`1.500` كـ`1.5`.
	frac = strings.TrimRight(frac, "0")
	if len(frac) > 2 {
		return 0, ErrBadPercent2
	}
	for _, r := range whole + frac {
		if r < '0' || r > '9' {
			return 0, ErrBadPercent2
		}
	}
	if len(whole) > 12 {
		return 0, ErrBadPercent2
	}
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, ErrBadPercent2
	}
	for len(frac) < 2 {
		frac += "0"
	}
	f, _ := strconv.ParseInt(frac, 10, 64)
	v := w*100 + f
	if neg {
		v = -v
	}
	return Percent2(v), nil
}

// NumericValue **يُكتب في عمود `numeric` بلا عائم** — `150` بأسٍّ `-2` = `1.50`.
func (p Percent2) NumericValue() (pgtype.Numeric, error) {
	return pgtype.Numeric{Int: big.NewInt(int64(p)), Exp: -2, Valid: true}, nil
}

// ScanNumeric **يُقرأ من عمود `numeric` بلا عائم** — ويُردّ ما زاد على منزلتين.
func (p *Percent2) ScanNumeric(n pgtype.Numeric) error {
	if !n.Valid {
		return errors.New("pricing: نسبةٌ فارغة تُقرأ في غير مؤشّر")
	}
	if n.NaN || n.InfinityModifier != pgtype.Finite {
		return ErrBadPercent2
	}
	v := new(big.Int).Set(n.Int)
	shift := int64(n.Exp) + 2
	ten := big.NewInt(10)
	if shift >= 0 {
		v.Mul(v, new(big.Int).Exp(ten, big.NewInt(shift), nil))
	} else {
		rem := new(big.Int)
		v.QuoRem(v, new(big.Int).Exp(ten, big.NewInt(-shift), nil), rem)
		if rem.Sign() != 0 {
			return ErrBadPercent2
		}
	}
	if !v.IsInt64() {
		return ErrBadPercent2
	}
	*p = Percent2(v.Int64())
	return nil
}

// Percent2Ptr مؤشّرٌ إلى نسبة — لحقول الردّ.
func Percent2Ptr(p Percent2) *Percent2 { return &p }

// RoundedPtr **النسبةُ الصحيحةُ لمؤشّرٍ قد يكون فارغاً** — `null` يبقى `null`.
func RoundedPtr(p *Percent2) *int64 {
	if p == nil {
		return nil
	}
	v := p.RoundedPercent()
	return &v
}
