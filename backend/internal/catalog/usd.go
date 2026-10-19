package catalog

// ══════════════════════════════════════════════════════════════════════
//  **أسعارٌ بالدولار وسعرُ صرفٍ للمتجر** (طلبُ المالك ٢٠٢٦-١٠-١٢)
// ══════════════════════════════════════════════════════════════════════
//
// **كلُّ صنفٍ يختار عملتَه**: بالليرة يبقى كما هو، **وبالدولار يُحفظ سعرُه
// بالسنت ويُكتب مقابلُه بالليرة** = السنتُ × سعرِ الصرف ÷ ١٠٠، **لأقرب ٥ ليرات**.
// **ومتى بدّل المتجرُ سعرَ صرفه أُعيد حسابُ كلّ صنفٍ دولاريٍّ وإضافاته** في
// معاملةٍ واحدة — والليريُّ لا يُمسّ.
//
// **وحارسُ القفزة**: تبديلٌ يزيد أو ينقص أكثرَ من ١٠٪ يُردّ بمثالٍ من أصنافه
// حتّى يؤكّده (`confirm`) — **فـ١١٥٠٠ مكانَ ١١٥ لا يضاعف أسعارَه مئةَ مرّةٍ صامتاً.**

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

const (
	CurrencySYP = "SYP"
	CurrencyUSD = "USD"
	// rateJumpPct **ما فوقه يُطلب تأكيدُه.**
	rateJumpPct = 0.10
)

var (
	ErrExchangeRateRequired = httpx.NewError(http.StatusBadRequest,
		"exchange_rate_required", "errors.exchange_rate_required")
	ErrExchangeRateInvalid = httpx.NewError(http.StatusBadRequest,
		"exchange_rate_invalid", "errors.exchange_rate_invalid")
)

// RoundTo5 **لأقرب ٥ ليرات** — قرارُ المالك.
func RoundTo5(v float64) int64 {
	return int64(math.Round(v/5)) * 5
}

// USDToSYP السنتُ بسعر الصرف ⇒ ليراتٌ مدوَّرةٌ لأقرب ٥.
func USDToSYP(cents int64, rate float64) int64 {
	return RoundTo5(float64(cents) * rate / 100)
}

func dollarsToCents(d float64) int64 { return int64(math.Round(d * 100)) }

// merchantRate سعرُ صرف المتجر — وصفرٌ إن لم يُكتب.
func merchantRate(ctx context.Context, q dbtx.Querier, merchantID string) (float64, error) {
	var r *float64
	err := q.QueryRow(ctx, `SELECT exchange_rate::float8 FROM merchants WHERE id = $1::uuid`, merchantID).Scan(&r)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.ErrNotFound
	}
	if err != nil || r == nil {
		return 0, err
	}
	return *r, nil
}

// usdPlan **ما يُكتب في الأعمدة الدولاريّة** بعد حلّ المُدخَل.
type usdPlan struct {
	// set: الصنفُ دولاريٌّ بهذا السنت. clear: صار ليريّاً. وكلاهما false: بلا تغيير.
	set   bool
	clear bool
	cents int64
}

// resolveUSD **يحلّ الدولارَ إلى ليرة قبل الكتابة** — فيكتب الإنشاءُ والتعديلُ
// `price` و`price_delta` بالليرة كما كانا، **ويُرجع ما يُكتب في عمود السنت.**
//
// **وسعرٌ بالليرة بلا عملةٍ يجعل الصنفَ ليريّاً** — نسخةُ تطبيقٍ قديمةٌ تعدّل
// صنفاً دولاريّاً فترسل ليرةً: **آخرُ سعرٍ صريحٍ يغلب**، ولا يعيده سعرُ الصرف.
func resolveUSD(ctx context.Context, q dbtx.Querier, merchantID string, in *MenuItemInput) (usdPlan, error) {
	usd := in.PriceCurrency != nil && *in.PriceCurrency == CurrencyUSD
	anyOptUSD := false
	if in.Modifiers != nil {
		for _, g := range *in.Modifiers {
			for _, o := range g.Options {
				if o.PriceDeltaUSD != nil {
					anyOptUSD = true
				}
			}
		}
	}
	if !usd && !anyOptUSD {
		if in.Price != nil || (in.PriceCurrency != nil && *in.PriceCurrency == CurrencySYP) {
			return usdPlan{clear: true}, nil
		}
		return usdPlan{}, nil
	}
	rate, err := merchantRate(ctx, q, merchantID)
	if err != nil {
		return usdPlan{}, err
	}
	if rate <= 0 {
		return usdPlan{}, ErrExchangeRateRequired
	}
	plan := usdPlan{}
	if usd {
		if in.PriceUSD == nil || *in.PriceUSD < 0 {
			return usdPlan{}, ErrNameRequired
		}
		plan = usdPlan{set: true, cents: dollarsToCents(*in.PriceUSD)}
		syp := USDToSYP(plan.cents, rate)
		in.Price = &syp
	}
	if in.Modifiers != nil {
		groups := *in.Modifiers
		for gi := range groups {
			for oi := range groups[gi].Options {
				o := &groups[gi].Options[oi]
				if o.PriceDeltaUSD != nil {
					c := dollarsToCents(*o.PriceDeltaUSD)
					o.priceDeltaUSDCents = &c
					o.PriceDelta = USDToSYP(c, rate)
				}
			}
		}
	}
	return plan, nil
}

// RateSample **مثالٌ من أصناف المتجر** — يُعرض قبل تأكيد قفزةٍ في سعر الصرف.
type RateSample struct {
	Name     string `json:"name"`
	OldPrice int64  `json:"old_price"`
	NewPrice int64  `json:"new_price"`
}

// SetExchangeRate **يبدّل سعرَ صرف المتجر ويعيد حسابَ أصنافه الدولاريّة.**
//
// **ويُرجع عددَ الأصناف المحسوبة**، أو `exchange_rate_jump` بمثالٍ إن زاد
// التبديلُ على ١٠٪ ولم يُؤكَّد.
func (s *Service) SetExchangeRate(ctx context.Context, actorID, merchantID string, rate float64, confirm bool, ip string) (int64, error) {
	if rate <= 0 || rate > 1_000_000 || math.IsNaN(rate) {
		return 0, ErrExchangeRateInvalid
	}
	old, err := merchantRate(ctx, s.db, merchantID)
	if err != nil {
		return 0, err
	}
	if old > 0 && !confirm && math.Abs(rate-old)/old > rateJumpPct {
		samples := []RateSample{}
		rows, err := s.db.Query(ctx, `
			SELECT name, merchant_price, price_usd_cents FROM menu_items
			 WHERE merchant_id = $1::uuid AND price_usd_cents IS NOT NULL
			 ORDER BY sort_order, created_at LIMIT 3`, merchantID)
		if err == nil {
			for rows.Next() {
				var sm RateSample
				var cents int64
				if rows.Scan(&sm.Name, &sm.OldPrice, &cents) == nil {
					sm.NewPrice = USDToSYP(cents, rate)
					samples = append(samples, sm)
				}
			}
			rows.Close()
		}
		// **والتفصيلُ نصوصٌ لا أرقام** — التطبيقُ يقرأ `details` خريطةَ نصوص.
		lines := make([]string, 0, len(samples))
		for _, sm := range samples {
			lines = append(lines, fmt.Sprintf("%s: %d ← %d", sm.Name, sm.NewPrice, sm.OldPrice))
		}
		return 0, &httpx.AppError{Status: http.StatusConflict, Code: "exchange_rate_jump",
			MessageKey: "errors.exchange_rate_jump",
			Details: map[string]any{
				"old_rate": strconv.FormatFloat(old, 'f', -1, 64),
				"new_rate": strconv.FormatFloat(rate, 'f', -1, 64),
				"samples":  strings.Join(lines, "\n"),
			}}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE merchants SET exchange_rate = $2, updated_at = now() WHERE id = $1::uuid`,
		merchantID, rate); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE menu_items
		   SET merchant_price = (round(price_usd_cents * $2::numeric / 100 / 5) * 5)::bigint,
		       price          = (round(price_usd_cents * $2::numeric / 100 / 5) * 5)::bigint,
		       updated_at     = now()
		 WHERE merchant_id = $1::uuid AND price_usd_cents IS NOT NULL`, merchantID, rate)
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE modifier_options o
		   SET price_delta = (round(o.price_delta_usd_cents * $2::numeric / 100 / 5) * 5)::bigint
		  FROM modifier_groups g JOIN menu_items i ON i.id = g.item_id
		 WHERE o.group_id = g.id AND i.merchant_id = $1::uuid
		   AND o.price_delta_usd_cents IS NOT NULL`, merchantID, rate); err != nil {
		return 0, err
	}
	// **وقائمةُ المشروبات تتبع الصرفَ كذلك** (٢٠٢٦-١٠-١٩) — ومنسوخاتُها تبدّلت بالسطر أعلاه.
	if _, err := tx.Exec(ctx, `
		UPDATE merchant_drinks
		   SET price = (round(price_usd_cents * $2::numeric / 100 / 5) * 5)::bigint
		 WHERE merchant_id = $1::uuid AND price_usd_cents IS NOT NULL`, merchantID, rate); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	s.audit(ctx, actorID, "menu.exchange_rate", "merchant", merchantID, ip)
	return tag.RowsAffected(), nil
}
