package catalog

// ══════════════════════════════════════════════════════════════════════
// **مزامنةُ الأسعار والكميّات من برنامج الأمين** (قرارُ المالك ٢٠٢٦-١٠-١٩)
// ══════════════════════════════════════════════════════════════════════
//
// المولاتُ تحاسب بالأمين — **والسعرُ والكميّةُ هناك أصلاً.** فتصل قائمتُه هنا من
// بابين: **ملفٌّ يرفعه المحاسب** (`pricesync_file.go`) **أو ساعٍ على كمبيوتر المول**
// يبعثها برمزٍ خاصّ. **والقلبُ واحدٌ للبابين** — مطابقةٌ وتحديثٌ وتقرير.
//
// # المطابقة
//
//  1. **برقم المادّة** (`external_code`) — المضمون.
//  2. **ثمّ بالباركود.**
//  3. **ثمّ بالاسم إن طابق صنفاً واحداً تماماً** — **ويُحفظ رقمُ المادّة عليه**،
//     فالمرّةُ التالية بالرقم لا بالاسم.
//
// # والتحديث
//
//   - **السعرُ سعرُ البيع في الأمين** = سعرُ المتجر عندنا، وعمولةُ المنصّة عليه كأيّ صنف.
//     **وبالدولار إن كانت أسعارُ المول دولاراً** — فيتبع سعرَ الصرف كالأصناف.
//   - **والكميّةُ تحكم التوفّر** (قرارُ المالك): صفرٌ ⇒ «غير متوفر»، وفوقه ⇒ «متوفر».
//   - **ولا يُنشأ صنفٌ جديدٌ هنا** — مادّةٌ لا مقابلَ لها تُذكر في التقرير. **وإنشاؤها
//     ينتظر ملفَّ مولٍ حقيقيّ** يُرى فيه كيف تُقسَم موادُّه، فتوضع في أقسام السوق.
//
// **و«جرّب» لا يكتب شيئاً** — يقول ما سيتغيّر قبل أن يتغيّر.

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"unicode"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// SyncRow **مادّةٌ من الأمين** — وكلُّ حقلٍ فارغٍ لا يُمسّ به شيء.
type SyncRow struct {
	Code    string   `json:"code"`
	Barcode string   `json:"barcode,omitempty"`
	Name    string   `json:"name,omitempty"`
	Price   *float64 `json:"price,omitempty"`
	Qty     *float64 `json:"qty,omitempty"`
}

// SyncOptions **كيف تُقرأ القائمة.**
type SyncOptions struct {
	// Currency **عملةُ أسعار المول** — `SYP` أو `USD`.
	Currency string
	// DryRun **«جرّب»** — يقول ولا يكتب.
	DryRun bool
	// Source **من أين جاءت** — `file` · `agent` · `api`.
	Source string
	// ActorID **من رفعها** — وفارغٌ للساعي.
	ActorID string
}

// SyncIssue **مادّةٌ لم تُطبَّق** ولماذا.
type SyncIssue struct {
	Row    int    `json:"row"`
	Code   string `json:"code,omitempty"`
	Name   string `json:"name,omitempty"`
	Reason string `json:"reason"`
}

// SyncReport **ما وقع — أو ما سيقع في «جرّب».**
type SyncReport struct {
	DryRun          bool        `json:"dry_run"`
	Rows            int         `json:"rows"`
	Matched         int         `json:"matched"`
	PriceChanged    int         `json:"price_changed"`
	MadeAvailable   int         `json:"made_available"`
	MadeUnavailable int         `json:"made_unavailable"`
	LinkedByName    int         `json:"linked_by_name"`
	Unchanged       int         `json:"unchanged"`
	Unmatched       int         `json:"unmatched"`
	Skipped         []SyncIssue `json:"skipped"`
	// UnmatchedSample **أوّلُ ما لم يُطابَق** — لا كلُّه: قائمةُ مولٍ آلافُ مادّة.
	UnmatchedSample []SyncIssue `json:"unmatched_sample"`
}

// maxSyncRows **سقفُ القائمة الواحدة** — مولٌ كبيرٌ عشرةُ آلاف، والسقفُ فوقه بكثير.
const maxSyncRows = 50000

// maxSample **كم مادّةً تُذكر بالاسم** فيما لم يُطابَق أو تُرك.
const maxSample = 50

// ErrSyncEmpty **قائمةٌ فارغةٌ أو أطولُ من السقف.**
var ErrSyncEmpty = httpx.NewError(400, "price_sync_empty", "errors.price_sync_empty")

type syncItem struct {
	id        string
	name      string
	code      string
	barcode   string
	price     int64
	usdCents  *int64
	available bool
}

// normName **الاسمُ للمقارنة** — بلا فراغاتٍ زائدة، وبلا فروق الهمزات والتاء والياء.
func normName(s string) string {
	r := strings.NewReplacer("أ", "ا", "إ", "ا", "آ", "ا", "ة", "ه", "ى", "ي", "ـ", "")
	s = r.Replace(strings.ToLower(strings.TrimSpace(s)))
	return strings.Join(strings.FieldsFunc(s, unicode.IsSpace), " ")
}

// SyncPrices **يطابق القائمةَ بأصناف المتجر ويحدّثها** — أو يقول ما سيفعل.
func (s *Service) SyncPrices(ctx context.Context, merchantID string, rows []SyncRow, opt SyncOptions) (*SyncReport, error) {
	if len(rows) == 0 || len(rows) > maxSyncRows {
		return nil, ErrSyncEmpty
	}
	usd := opt.Currency == CurrencyUSD
	var max int64
	if s.settings != nil {
		max = s.settings.GetInt(ctx, "merchants.max_item_price")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var rate float64
	if usd {
		if rate, err = merchantRate(ctx, tx, merchantID); err != nil {
			return nil, err
		}
		if rate <= 0 {
			return nil, ErrExchangeRateRequired
		}
	}

	dbRows, err := tx.Query(ctx, `
		SELECT id::text, name, COALESCE(external_code, ''), COALESCE(barcode, ''),
		       merchant_price, price_usd_cents, available
		  FROM menu_items WHERE merchant_id = $1::uuid`, merchantID)
	if err != nil {
		return nil, err
	}
	items := []syncItem{}
	for dbRows.Next() {
		var it syncItem
		if err := dbRows.Scan(&it.id, &it.name, &it.code, &it.barcode, &it.price, &it.usdCents, &it.available); err != nil {
			dbRows.Close()
			return nil, err
		}
		items = append(items, it)
	}
	dbRows.Close()
	if err := dbRows.Err(); err != nil {
		return nil, err
	}
	byCode, byBarcode, byName := map[string]int{}, map[string]int{}, map[string][]int{}
	for i, it := range items {
		if it.code != "" {
			byCode[it.code] = i
		}
		if it.barcode != "" {
			byBarcode[it.barcode] = i
		}
		n := normName(it.name)
		byName[n] = append(byName[n], i)
	}

	rep := &SyncReport{DryRun: opt.DryRun, Rows: len(rows), Skipped: []SyncIssue{}, UnmatchedSample: []SyncIssue{}}
	skip := func(i int, r SyncRow, why string) {
		if len(rep.Skipped) < maxSample {
			rep.Skipped = append(rep.Skipped, SyncIssue{Row: i + 1, Code: r.Code, Name: r.Name, Reason: why})
		}
	}
	touched := map[int]bool{}
	for i, r := range rows {
		r.Code, r.Barcode = strings.TrimSpace(r.Code), strings.TrimSpace(r.Barcode)
		idx, byNameHit := -1, false
		if j, ok := byCode[r.Code]; ok && r.Code != "" {
			idx = j
		} else if j, ok := byBarcode[r.Barcode]; ok && r.Barcode != "" {
			idx = j
		} else if hits := byName[normName(r.Name)]; r.Name != "" && len(hits) == 1 {
			idx, byNameHit = hits[0], true
		}
		if idx < 0 {
			rep.Unmatched++
			if len(rep.UnmatchedSample) < maxSample {
				rep.UnmatchedSample = append(rep.UnmatchedSample, SyncIssue{Row: i + 1, Code: r.Code, Name: r.Name, Reason: "unmatched"})
			}
			continue
		}
		if touched[idx] {
			skip(i, r, "duplicate")
			continue
		}
		touched[idx] = true
		rep.Matched++
		it := &items[idx]

		var newPrice int64 = it.price
		var newCents *int64 = it.usdCents
		priceSet := false
		if r.Price != nil {
			p := *r.Price
			if p <= 0 || math.IsNaN(p) || math.IsInf(p, 0) {
				skip(i, r, "bad_price")
				continue
			}
			if usd {
				c := dollarsToCents(p)
				newCents, newPrice = &c, USDToSYP(c, rate)
			} else {
				newCents, newPrice = nil, int64(math.Round(p))
			}
			if max > 0 && newPrice > max {
				skip(i, r, "price_too_high")
				continue
			}
			priceSet = true
		}
		newAvail := it.available
		if r.Qty != nil {
			newAvail = *r.Qty > 0
		}
		link := byNameHit || (it.code == "" && r.Code != "") || (it.barcode == "" && r.Barcode != "")
		if byNameHit {
			rep.LinkedByName++
		}

		priceChanged := priceSet && (newPrice != it.price || !sameCents(newCents, it.usdCents))
		availChanged := newAvail != it.available
		if priceChanged {
			rep.PriceChanged++
		}
		if availChanged && newAvail {
			rep.MadeAvailable++
		}
		if availChanged && !newAvail {
			rep.MadeUnavailable++
		}
		if !priceChanged && !availChanged && !link {
			rep.Unchanged++
			continue
		}
		if opt.DryRun {
			continue
		}
		code, barcode := it.code, it.barcode
		if code == "" {
			code = r.Code
		}
		if barcode == "" {
			barcode = r.Barcode
		}
		if _, err := tx.Exec(ctx, `
			UPDATE menu_items SET
				merchant_price  = $2,
				price           = $2,
				price_usd_cents = $3,
				available       = $4,
				external_code   = NULLIF($5, ''),
				barcode         = NULLIF($6, ''),
				updated_at      = now()
			 WHERE id = $1::uuid`,
			it.id, newPrice, newCents, newAvail, code, barcode); err != nil {
			return nil, err
		}
	}

	if opt.DryRun {
		return rep, nil
	}
	raw, _ := json.Marshal(rep)
	if _, err := tx.Exec(ctx, `
		INSERT INTO price_sync_runs (merchant_id, source, report, created_by)
		VALUES ($1::uuid, $2, $3, NULLIF($4, '')::uuid)`,
		merchantID, opt.Source, raw, opt.ActorID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	s.audit(ctx, opt.ActorID, "menu.price_sync", "merchant", merchantID, "")
	return rep, nil
}

func sameCents(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
