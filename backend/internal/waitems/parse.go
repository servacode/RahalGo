package waitems

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// ══════════════════════════════════════════════════════════════════════
// **قراءة «اسم وسعر» من تعليق الصورة** — دالّة نقيّة بلا قاعدة
// ══════════════════════════════════════════════════════════════════════
//
// الاسم والسعر بأي ترتيب؛ أرقام عربيّة وغربيّة؛ «٢٥ ألف»؛ «$» أو «دولار» ⇒ دولار.
// **ولا تخمين**: رقمان محتملان بلا علامة ⇒ `Ambiguous` (يسأل البوت)، ولا رقم ⇒ يسأل.
// وأرقام الأحجام («٢٥٠ مل»، «٢ لتر») جزء من الاسم لا سعر.

// Parsed ناتج القراءة.
type Parsed struct {
	// Name اسم الصنف (قد يكون فارغاً).
	Name string
	// Price السعر بوحدة عملته (ليرة أو دولار) — صالح فقط إن HasPrice.
	Price    float64
	HasPrice bool
	// Ambiguous أكثر من رقم محتمل للسعر بلا علامة تحسم.
	Ambiguous bool
	// Currency "USD" أو "SYP" إن ذُكرت صراحةً، وإلا فارغ (عملة المتجر).
	Currency string
}

var arabicDigits = strings.NewReplacer(
	"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
	"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
	"٫", ".", "٬", ",", "،", ",",
)

// ToASCIIDigits يحوّل الأرقام العربيّة والفارسيّة وفواصلها إلى غربيّة.
func ToASCIIDigits(s string) string { return arabicDigits.Replace(s) }

var (
	reNum      = regexp.MustCompile(`^\d[\d.,]*$`)
	reNumUnit  = regexp.MustCompile(`^(\d[\d.,]*)(\D.*)$`)
	reThousand = regexp.MustCompile(`^\d{1,3}(,\d{3})+$`)
	reDotThou  = regexp.MustCompile(`^\d{1,3}(\.\d{3})+$`)
	reCurSyp   = regexp.MustCompile(`(^|[^\p{L}])ل\s*\.?\s*س\.?([^\p{L}]|$)`)
)

// كلمات مضاعِف الألف.
var thousandWords = map[string]bool{"ألف": true, "الف": true, "آلاف": true, "الاف": true, "k": true, "ك": true}

// كلمات تدلّ على الدولار / الليرة.
var usdWords = map[string]bool{"$": true, "دولار": true, "دولارات": true, "usd": true, "بالدولار": true, "بدولار": true, "dollar": true, "dollars": true}
var sypWords = map[string]bool{"ليرة": true, "ليره": true, "ل.س": true, "syp": true, "بالليرة": true, "ليرات": true}

// كلمات تسبق السعر.
var priceWords = map[string]bool{"سعر": true, "السعر": true, "بسعر": true, "سعرو": true, "سعره": true, "ب": true, "بـ": true, "price": true}

// وحدات تعني أن الرقم قبلها جزء من الاسم (حجم/وزن/عدد).
var unitWords = map[string]bool{
	"مل": true, "ملل": true, "ml": true, "لتر": true, "ليتر": true, "l": true, "ل": true, "لترات": true,
	"غ": true, "غم": true, "غرام": true, "جرام": true, "جم": true, "g": true, "gr": true, "gm": true,
	"كغ": true, "كغم": true, "كيلو": true, "كلغ": true, "kg": true, "كج": true,
	"سم": true, "cm": true, "مم": true, "mm": true, "م": true, "متر": true,
	"حبة": true, "حبات": true, "حبه": true, "قطعة": true, "قطع": true, "قطعه": true, "علبة": true, "علب": true, "علبه": true,
	"كيس": true, "اكياس": true, "أكياس": true, "عبوة": true, "عبوات": true, "كرتونة": true, "كرتون": true, "باكيت": true, "باكو": true,
	"وجبة": true, "وجبات": true, "شخص": true, "اشخاص": true, "أشخاص": true, "دزينة": true, "قنينة": true, "زجاجة": true,
	"انش": true, "إنش": true, "بوصة": true, "%": true, "٪": true, "رطل": true, "اوقية": true, "أوقية": true,
	"ورقة": true, "اوراق": true, "أوراق": true, "ساعة": true, "ساعات": true, "يوم": true, "ايام": true, "أيام": true,
	"gb": true, "tb": true, "mah": true, "w": true, "v": true,
}

type tok struct {
	raw      string
	num      float64
	isNum    bool
	numStr   string
	drop     bool // لا يدخل الاسم
	mult     float64
	cur      string // USD/SYP محاط بالرقم
	marked   bool
	unitNext bool
}

func cleanWord(s string) string {
	return strings.ToLower(strings.TrimFunc(s, func(r rune) bool {
		return unicode.IsPunct(r) && r != '$' && r != '%' && r != '٪'
	}))
}

// parseNumber يقرأ رقماً بفواصل (آلاف أو عشريّة). usd: النقطة عشريّة دائماً.
func parseNumber(s string, usd bool) (float64, bool) {
	s = strings.Trim(s, ".,")
	if s == "" {
		return 0, false
	}
	hasC, hasD := strings.Contains(s, ","), strings.Contains(s, ".")
	switch {
	case hasC && hasD:
		s = strings.ReplaceAll(s, ",", "")
	case hasC:
		if reThousand.MatchString(s) {
			s = strings.ReplaceAll(s, ",", "")
		} else if strings.Count(s, ",") == 1 {
			s = strings.Replace(s, ",", ".", 1)
		} else {
			return 0, false
		}
	case hasD:
		if !usd && reDotThou.MatchString(s) {
			s = strings.ReplaceAll(s, ".", "")
		} else if strings.Count(s, ".") > 1 {
			return 0, false
		}
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

// ParseCaption **يقرأ الاسم والسعر من تعليق** — انظر رأس الملف.
func ParseCaption(caption string) Parsed {
	s := ToASCIIDigits(caption)
	s = reCurSyp.ReplaceAllString(s, "${1} ليرة ${2}")
	s = strings.NewReplacer("$", " $ ", "‏", " ", "‎", " ", "×", " × ").Replace(s)
	low := strings.ToLower(s)
	usdMentioned := strings.Contains(low, "$") || strings.Contains(low, "دولار") || strings.Contains(low, "usd")

	words := strings.Fields(s)
	toks := make([]*tok, 0, len(words))
	for _, w := range words {
		t := &tok{raw: w}
		cw := cleanWord(w)
		// «ب25000» / «بـ25000»: بادئة السعر ملتصقة.
		for _, p := range []string{"بـ", "ب"} {
			if strings.HasPrefix(cw, p) && len(cw) > len(p) && cw[len(p)] >= '0' && cw[len(p)] <= '9' {
				cw = strings.TrimPrefix(cw, p)
				t.marked = true
				break
			}
		}
		trimmed := strings.TrimRight(cw, ".,:-–—=")
		if reNum.MatchString(trimmed) {
			if v, ok := parseNumber(trimmed, usdMentioned); ok {
				t.isNum, t.num, t.numStr = true, v, trimmed
			}
		} else if m := reNumUnit.FindStringSubmatch(cw); m != nil {
			// «25k» أو «250مل»
			unit := cleanWord(m[2])
			if v, ok := parseNumber(strings.TrimRight(m[1], ".,"), usdMentioned); ok {
				switch {
				case thousandWords[unit]:
					t.isNum, t.num, t.numStr, t.mult = true, v, m[1], 1000
				case unitWords[unit]:
					// جزء من الاسم — يبقى كما هو
				}
			}
		}
		toks = append(toks, t)
	}

	// علاقات الرقم بجيرانه.
	for i, t := range toks {
		if !t.isNum {
			continue
		}
		if i+1 < len(toks) {
			nw := cleanWord(toks[i+1].raw)
			switch {
			case t.mult == 0 && thousandWords[nw]:
				t.mult = 1000
				toks[i+1].drop = true
				if i+2 < len(toks) {
					t.setCur(cleanWord(toks[i+2].raw), toks[i+2])
				}
			case unitWords[nw] && !usdWords[nw] && !sypWords[nw]:
				t.unitNext = true
			default:
				t.setCur(nw, toks[i+1])
			}
		}
		if i > 0 && !toks[i-1].drop {
			pw := cleanWord(toks[i-1].raw)
			if priceWords[pw] {
				t.marked = true
				toks[i-1].drop = true
			}
			if usdWords[pw] || sypWords[pw] {
				// «$25» أو «دولار 25»
				t.setCur(pw, toks[i-1])
			}
		}
		if t.mult == 0 {
			t.mult = 1
		}
	}

	out := Parsed{}
	switch {
	case strings.Contains(low, "$") || strings.Contains(low, "دولار") || strings.Contains(low, "usd"):
		out.Currency = "USD"
	case strings.Contains(s, "ليرة") || strings.Contains(s, "ليره") || strings.Contains(low, "syp"):
		out.Currency = "SYP"
	}

	// المرشَّحون: أرقام ليست أحجاماً.
	var cands []*tok
	for _, t := range toks {
		if t.isNum && !t.unitNext {
			cands = append(cands, t)
		}
	}
	var chosen *tok
	switch {
	case len(cands) == 1:
		chosen = cands[0]
	case len(cands) > 1:
		var marked []*tok
		for _, t := range cands {
			if t.marked || t.cur != "" {
				marked = append(marked, t)
			}
		}
		if len(marked) == 1 {
			chosen = marked[0]
		} else {
			out.Ambiguous = true
		}
	}
	if chosen != nil {
		v := chosen.num * chosen.mult
		if v > 0 {
			out.Price, out.HasPrice = v, true
		}
		chosen.drop = true
		if chosen.cur != "" {
			out.Currency = chosen.cur
		}
	}
	// أسماء: كل ما لم يُستهلك. الأرقام المرشَّحة المرفوضة (الغامضة) تُسقَط من الاسم أيضاً.
	var parts []string
	for _, t := range toks {
		if t.drop {
			continue
		}
		if out.Ambiguous && t.isNum && !t.unitNext {
			continue
		}
		cw := cleanWord(t.raw)
		if usdWords[cw] || sypWords[cw] {
			continue
		}
		if t.raw == "×" && len(parts) == 0 {
			continue
		}
		parts = append(parts, t.raw)
	}
	out.Name = cleanName(strings.Join(parts, " "))
	return out
}

func (t *tok) setCur(w string, nb *tok) {
	switch {
	case usdWords[w]:
		t.cur = "USD"
		nb.drop = true
	case sypWords[w]:
		t.cur = "SYP"
		nb.drop = true
	}
}

// cleanName يشذّب الاسم: فواصل وعلامات على الأطراف.
func cleanName(s string) string {
	s = strings.TrimFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune("-–—:=،,.|/\\*_~\"'()[]", r)
	})
	return strings.Join(strings.Fields(s), " ")
}
