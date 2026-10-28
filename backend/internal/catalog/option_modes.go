package catalog

// ══════════════════════════════════════════════════════════════════════
// **طريقةُ سعر الخيارات** (قرارُ المالك ٢٠٢٦-١٠-٢٨) — هجرة 0482
// ══════════════════════════════════════════════════════════════════════
//
//   priced  «بسعر»       فرقُ سعرٍ لكلّ خيار (الحجم · العبوة · الإضافات).
//   none    «بدون سعر»   لا سعر (المقاس · اللون · النمرة · بدون).
//   weight  «حسب الوزن»  سعرُ الصنف لوزنٍ أساسيّ، ولكلّ خيارٍ غراماته.
//
// **والسعرُ تحسبه القاعدةُ لا هذا الملفّ** (`weight_option_delta` ومشغّلاتُها
// في الهجرة) — فيبقى صحيحاً من كلّ بابٍ يكتب سعرَ الصنف. وهنا ما يسبق الكتابة:
// تنظيفُ الطريقة، وأسماءُ الأوزان، **وحفظُ طريقة المجموعة لنسخ التطبيقات
// القديمة** التي لا ترسلها.

import (
	"context"
	"strconv"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

const (
	PriceModePriced = "priced"
	PriceModeNone   = "none"
	PriceModeWeight = "weight"

	// DefaultBaseGrams الوزنُ الذي يخصّه سعرُ الصنف إن لم يُكتب.
	DefaultBaseGrams = 100
	maxGrams         = 1_000_000
)

// validPriceMode **الطريقةُ المعروفة أو «بسعر»** — قيمةٌ غريبةٌ لا تُسقط الحفظ.
func validPriceMode(m string) string {
	switch m {
	case PriceModeNone, PriceModeWeight:
		return m
	}
	return PriceModePriced
}

// WeightName **اسمُ الوزن بالعربيّة الفصحى** — «250 غ»، «نصف كيلو»، «كيلو»، «1.5 كيلو».
func WeightName(g int) string {
	switch {
	case g == 500:
		return "نصف كيلو"
	case g == 1000:
		return "كيلو"
	case g > 1000 && g%500 == 0:
		return strconv.FormatFloat(float64(g)/1000, 'f', -1, 64) + " كيلو"
	}
	return strconv.Itoa(g) + " غ"
}

// GramsFromName **يقرأ الوزنَ من اسمٍ** — لخيارٍ أرسلته نسخةٌ لا تعرف الغرامات.
// ويُرجع صفراً إن لم يُفهم.
func GramsFromName(name string) int {
	s := strings.TrimSpace(toLatinDigits(name))
	switch s {
	case "ربع كيلو":
		return 250
	case "نص كيلو", "نصف كيلو":
		return 500
	case "كيلو", "كيلو غرام", "كغ":
		return 1000
	}
	num, unit := s, ""
	for _, u := range []string{"كيلو", "كغ", "غرام", "غ"} {
		if strings.HasSuffix(s, u) {
			num, unit = strings.TrimSpace(strings.TrimSuffix(s, u)), u
			break
		}
	}
	if unit == "" {
		return 0
	}
	v, err := strconv.ParseFloat(num, 64)
	if err != nil || v <= 0 {
		return 0
	}
	if unit == "كيلو" || unit == "كغ" {
		v *= 1000
	}
	if v > maxGrams {
		return 0
	}
	return int(v + 0.5)
}

func toLatinDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '٠' && r <= '٩':
			b.WriteRune('0' + (r - '٠'))
		case r == '٫':
			b.WriteRune('.')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// cleanGroupMode **يضبط طريقةَ المجموعة قبل كتابتها** — ويُرجع ما يُكتب في
// `price_mode` و`base_grams`. والغراماتُ تُكتب في كلّ خيارٍ وزنيّ، **واسمُه
// يُولَّد منها** فلا يكتب كلُّ تطبيقٍ الاسمَ بطريقته.
func cleanGroupMode(g *ModifierGroupInput) (string, *int) {
	mode := PriceModePriced
	if g.PriceMode != nil {
		mode = validPriceMode(*g.PriceMode)
	}
	if mode != PriceModeWeight {
		for i := range g.Options {
			g.Options[i].Grams = nil
			if mode == PriceModeNone {
				g.Options[i].PriceDelta = 0
				g.Options[i].priceDeltaUSDCents = nil
			}
		}
		return mode, nil
	}
	base := DefaultBaseGrams
	if g.BaseGrams != nil && *g.BaseGrams > 0 && *g.BaseGrams <= maxGrams {
		base = *g.BaseGrams
	}
	for i := range g.Options {
		o := &g.Options[i]
		if o.Grams == nil || *o.Grams <= 0 || *o.Grams > maxGrams {
			o.Grams = nil
			if n := GramsFromName(o.Name); n > 0 {
				o.Grams = &n
			}
		}
		if o.Grams != nil {
			o.Name = WeightName(*o.Grams)
			o.priceDeltaUSDCents = nil
		}
	}
	return mode, &base
}

// inheritModes **نسخةُ التطبيق القديمة لا ترسل الطريقة** — فتُحفظ طريقةُ
// المجموعة القائمة بالاسم نفسِه، ووزنُها الأساسيّ، وغراماتُ خياراتها بأسمائها.
// **ولولاه لَصار صنفٌ بالوزن «بسعر» كلّما عدّله تطبيقٌ قديم.**
func inheritModes(ctx context.Context, q dbtx.Querier, itemID string, groups []ModifierGroupInput) error {
	need := false
	for _, g := range groups {
		if g.PriceMode == nil {
			need = true
		}
	}
	if !need {
		return nil
	}
	type prior struct {
		mode  string
		base  *int
		grams map[string]int
	}
	byName := map[string]*prior{}
	rows, err := q.Query(ctx, `
		SELECT g.name, g.price_mode, g.base_grams, o.name, o.grams
		  FROM modifier_groups g
		  LEFT JOIN modifier_options o ON o.group_id = g.id
		 WHERE g.item_id = $1 AND NOT g.store_drinks`, itemID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var gname, mode string
		var base, grams *int
		var oname *string
		if err := rows.Scan(&gname, &mode, &base, &oname, &grams); err != nil {
			rows.Close()
			return err
		}
		p, ok := byName[gname]
		if !ok {
			p = &prior{mode: mode, base: base, grams: map[string]int{}}
			byName[gname] = p
		}
		if oname != nil && grams != nil {
			p.grams[*oname] = *grams
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for gi := range groups {
		g := &groups[gi]
		if g.PriceMode != nil {
			continue
		}
		p, ok := byName[g.Name]
		if !ok || p.mode == PriceModePriced {
			continue
		}
		mode := p.mode
		g.PriceMode = &mode
		if g.BaseGrams == nil {
			g.BaseGrams = p.base
		}
		for oi := range g.Options {
			o := &g.Options[oi]
			if o.Grams == nil {
				if n, ok := p.grams[o.Name]; ok {
					o.Grams = &n
				}
			}
		}
	}
	return nil
}
