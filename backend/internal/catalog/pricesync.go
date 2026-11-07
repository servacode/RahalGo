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
//   - **ولا يُنشأ صنفٌ جديدٌ ما لم يُطلب صراحةً** (`CreateMissing`، ٢٠٢٦-١١-٠٤): مادّةٌ لا
//     مقابلَ لها تُذكر في التقرير، **وبالخيار تصير صنفاً مخفيّاً** في قسم انتظار
//     (`pricesync_create.go`).
//
// **و«جرّب» لا يكتب شيئاً** — يقول ما سيتغيّر قبل أن يتغيّر.

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// SyncRow **مادّةٌ من الأمين** — وكلُّ حقلٍ فارغٍ لا يُمسّ به شيء.
type SyncRow struct {
	Code    string   `json:"code"`
	Barcode string   `json:"barcode,omitempty"`
	Name    string   `json:"name,omitempty"`
	Price   *float64 `json:"price,omitempty"`
	Qty     *float64 `json:"qty,omitempty"`
	// Group **مجموعة المادّة بالأمين** (٢٠٢٦-١١-٠٧) — اختياريّة؛ بتحدّد قسم الصنف الجديد
	// إذا صاحب المتجر اختار قسم لهالمجموعة (`pricesync_groups.go`).
	Group string `json:"group,omitempty"`
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
	// CreateMissing **«أنشئ المواد غير الموجودة أصنافاً مخفيّة»** (٢٠٢٦-١١-٠٤) — مُطفأٌ
	// افتراضاً. انظر `pricesync_create.go`.
	CreateMissing bool
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
	DryRun          bool `json:"dry_run"`
	Rows            int  `json:"rows"`
	Matched         int  `json:"matched"`
	PriceChanged    int  `json:"price_changed"`
	MadeAvailable   int  `json:"made_available"`
	MadeUnavailable int  `json:"made_unavailable"`
	LinkedByName    int  `json:"linked_by_name"`
	Unchanged       int  `json:"unchanged"`
	Unmatched       int  `json:"unmatched"`
	// Created **أصنافٌ جديدةٌ مخفيّة** (أو ما سيُنشأ في «جرّب»)، وCreatedZeroPrice منها
	// ما سعرُه صفرٌ أو غائب. CreateSkipped **مادّةٌ بلا رقمٍ أو بلا اسم** فلم تُنشأ.
	// ZeroPrice **كلُّ صفٍّ سعرُه صفر** مطابَقاً كان أو جديداً.
	Created          int `json:"created"`
	CreatedZeroPrice int `json:"created_zero_price"`
	CreateSkipped    int `json:"create_skipped"`
	// وين راحت الأصناف الجديدة (٢٠٢٦-١١-٠٧): لقسم مجموعتها · لقسم من اسمها · للانتظار.
	CreatedByGroup int `json:"created_by_group"`
	CreatedByName  int `json:"created_by_name"`
	CreatedPending int `json:"created_pending"`
	ZeroPrice      int `json:"zero_price"`
	// CreatedSample **أوّلُ ما أُنشئ** للعرض.
	CreatedSample []SyncIssue `json:"created_sample"`
	Skipped       []SyncIssue `json:"skipped"`
	// UnmatchedSample **أوّلُ ما لم يُطابَق** — لا كلُّه: قائمةُ مولٍ آلافُ مادّة.
	UnmatchedSample []SyncIssue `json:"unmatched_sample"`
	// LinkRejected **رقمٌ أو باركودٌ لم يُحفظ** (٢٠٢٦-١١-٠٥) — فيه فراغٌ أو أطولُ من الحدّ
	// (`validLink`)، **وصفُّه طُبّق سعرُه وكميّتُه** فلا يُذكر في Skipped («لم تُطبَّق»).
	// وLinkRejectedSample أوّلُه بسببه: `bad_code` · `bad_barcode`.
	LinkRejected       int         `json:"link_rejected"`
	LinkRejectedSample []SyncIssue `json:"link_rejected_sample"`
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
	group     string
	source    string
	section   string
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

	// **والأقدمُ أوّلاً** — انظر «والأقدمُ يملك» تحت.
	dbRows, err := tx.Query(ctx, `
		SELECT id::text, name, COALESCE(external_code, ''), COALESCE(barcode, ''),
		       merchant_price, price_usd_cents, available,
		       COALESCE(external_group, ''), section_source, COALESCE(platform_section_id::text, '')
		  FROM menu_items WHERE merchant_id = $1::uuid
		 ORDER BY created_at, id`, merchantID)
	if err != nil {
		return nil, err
	}
	items := []syncItem{}
	for dbRows.Next() {
		var it syncItem
		if err := dbRows.Scan(&it.id, &it.name, &it.code, &it.barcode, &it.price, &it.usdCents, &it.available,
			&it.group, &it.source, &it.section); err != nil {
			dbRows.Close()
			return nil, err
		}
		items = append(items, it)
	}
	dbRows.Close()
	if err := dbRows.Err(); err != nil {
		return nil, err
	}
	// **والمطابقةُ بأرقامٍ لاتينيّةٍ في الطرفين** (`linkText`، ٢٠٢٦-١١-٠٥) — رقمٌ حُفظ «٠١٧٠٦٤»
	// يطابق «017064» في الملفّ، **وملفٌّ بأرقامٍ هنديّةٍ يطابق المحفوظَ لاتينيّاً.**
	//
	// **والأقدمُ يملك الرقمَ والباركود إن تشاركهما صنفان** (مراجعةُ ما قبل الإصدار ٢٠٢٦-١١-٠٥).
	// صنفُ المتجر حُفظ رقمُه «٠١٧٠٦٤» بيده، **و«أنشئ غير الموجود» صنعت قبل التحويل نسخةً
	// مخفيّةً برقم «017064»** — رقمٌ واحدٌ بعد `linkText`. **وكان الاستعلامُ بلا ترتيبٍ والأخيرُ
	// يكتب فوق الأوّل**، فيربح أيُّهما اتّفق: والأرجحُ النسخةُ الأحدث، فتتحدّث هي لا صنفُ المتجر.
	byCode, byBarcode, byName := map[string]int{}, map[string]int{}, map[string][]int{}
	for i, it := range items {
		if c := linkText(it.code); c != "" {
			if _, taken := byCode[c]; !taken {
				byCode[c] = i
			}
		}
		if b := linkText(it.barcode); b != "" {
			if _, taken := byBarcode[b]; !taken {
				byBarcode[b] = i
			}
		}
		n := normName(it.name)
		byName[n] = append(byName[n], i)
	}

	rep := &SyncReport{DryRun: opt.DryRun, Rows: len(rows), Skipped: []SyncIssue{}, UnmatchedSample: []SyncIssue{},
		CreatedSample: []SyncIssue{}, LinkRejectedSample: []SyncIssue{}}
	var toCreate []newItem
	createdCodes := map[string]bool{}
	skip := func(i int, r SyncRow, why string) {
		if len(rep.Skipped) < maxSample {
			rep.Skipped = append(rep.Skipped, SyncIssue{Row: i + 1, Code: r.Code, Name: r.Name, Reason: why})
		}
	}
	rejectLink := func(i int, r SyncRow, why string) {
		rep.LinkRejected++
		if len(rep.LinkRejectedSample) < maxSample {
			rep.LinkRejectedSample = append(rep.LinkRejectedSample, SyncIssue{Row: i + 1, Code: r.Code, Name: r.Name, Reason: why})
		}
	}
	touched := map[int]bool{}
	// **مجموعات الأمين** (٢٠٢٦-١١-٠٧): كلّ مجموعة شافها الملف بتنحفظ، والصنف المطابَق
	// بتنكتب عليه مجموعته إذا تغيّرت.
	seenGroups := map[string]string{}
	var groupIDs, groupTexts []string
	for i, r := range rows {
		r.Code, r.Barcode = linkText(r.Code), linkText(r.Barcode)
		r.Group = strings.TrimSpace(r.Group)
		if utf8.RuneCountInString(r.Group) > maxGroupKey {
			r.Group = string([]rune(r.Group)[:maxGroupKey])
		}
		if k := groupKey(r.Group); k != "" {
			seenGroups[k] = r.Group
		}
		idx, byNameHit := -1, false
		if j, ok := byCode[r.Code]; ok && r.Code != "" {
			idx = j
		} else if j, ok := byBarcode[r.Barcode]; ok && r.Barcode != "" {
			idx = j
		} else if hits := byName[normName(CleanItemName(r.Name))]; r.Name != "" && len(hits) == 1 {
			idx, byNameHit = hits[0], true
		} else if hits := byName[normName(r.Name)]; r.Name != "" && len(hits) == 1 {
			idx, byNameHit = hits[0], true
		}
		if idx < 0 && opt.CreateMissing {
			name := CleanItemName(r.Name)
			switch {
			case r.Code == "" || name == "":
				rep.CreateSkipped++
				skip(i, r, "no_code_or_name")
			// **ولا صنفَ برقمٍ لا يقبله النموذج** (`validLink`، ٢٠٢٦-١١-٠٥) — وإلّا وُلد صنفٌ
			// لا يُحفظ تعديلُه: عمودُ الاسم يُختار رقماً فتصير الأسماءُ أرقاماً بفراغاتها.
			case !validLink(r.Code):
				skip(i, r, "bad_code")
			case createdCodes[r.Code]:
				skip(i, r, "duplicate")
			default:
				n := newItem{code: r.Code, name: name, group: r.Group}
				if r.Price != nil && *r.Price > 0 && !math.IsNaN(*r.Price) && !math.IsInf(*r.Price, 0) {
					if usd {
						c := dollarsToCents(*r.Price)
						if c > 0 {
							n.cents, n.price = &c, USDToSYP(c, rate)
						}
					} else {
						n.price = int64(math.Round(*r.Price))
					}
				}
				if max > 0 && n.price > max {
					skip(i, r, "price_too_high")
					continue
				}
				if n.price == 0 {
					rep.ZeroPrice++
					rep.CreatedZeroPrice++
				}
				createdCodes[r.Code] = true
				toCreate = append(toCreate, n)
				if len(rep.CreatedSample) < maxSample {
					rep.CreatedSample = append(rep.CreatedSample, SyncIssue{Row: i + 1, Code: r.Code, Name: name, Reason: "created"})
				}
			}
			continue
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
		if r.Group != "" && r.Group != it.group {
			groupIDs, groupTexts = append(groupIDs, it.id), append(groupTexts, r.Group)
		}

		var newPrice int64 = it.price
		var newCents *int64 = it.usdCents
		priceSet := false
		if r.Price != nil {
			p := *r.Price
			if p <= 0 || math.IsNaN(p) || math.IsInf(p, 0) {
				if p == 0 {
					rep.ZeroPrice++
				}
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
		// **ورقمٌ أو باركودٌ لا يقبله النموذج لا يُكتب — والصفُّ يُطبَّق** (`validLink`، ٢٠٢٦-١١-٠٥).
		// فراغٌ في وسطه (باركودان بخانة) أو أطولُ من الحدّ **كان يُكتب فيُردّ بعدها كلُّ حفظٍ
		// للصنف.** فيُترك هو وحدَه ويُذكر (`LinkRejected`)، **والسعرُ والكميّةُ يُطبَّقان كما كانا**:
		// تركُ الصفّ كلِّه كان يُبقي صنفاً خلص بالأمين «متوفراً» بسعرٍ قديم في كلّ مزامنة.
		// **وما للصنف رقمُه لا يُكتب فوقه أصلاً** — فلا يُفحص.
		orig := r
		rejected := false
		if it.code == "" && !validLink(r.Code) {
			rejectLink(i, orig, "bad_code")
			r.Code = ""
			rejected = true
		}
		if it.barcode == "" && !validLink(r.Barcode) {
			rejectLink(i, orig, "bad_barcode")
			r.Barcode = ""
			rejected = true
		}
		// **ومطابقٌ بالاسم رُفض رابطُه ولم يُحفظ له شيءٌ لا يُعدّ «رُبط بالاسم»** — فالمرّةُ القادمةُ
		// بالاسم أيضاً، لا بالرقم كما تقول الصفحة.
		link := (byNameHit && !rejected) || (it.code == "" && r.Code != "") || (it.barcode == "" && r.Barcode != "")
		if byNameHit && link {
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

	rep.Created = len(toCreate)
	if err := placeNewItems(ctx, tx, merchantID, toCreate, opt.DryRun, rep); err != nil {
		return nil, err
	}
	if opt.DryRun {
		return rep, nil
	}
	if err := upsertGroups(ctx, tx, merchantID, seenGroups); err != nil {
		return nil, err
	}
	if len(groupIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE menu_items i SET external_group = t.g
			  FROM unnest($1::text[], $2::text[]) AS t(id, g)
			 WHERE i.id = t.id::uuid`, groupIDs, groupTexts); err != nil {
			return nil, err
		}
	}
	// **وصنف قديم لسّا بالانتظار ومجموعته إلها قسم بينتقل** — بس اللي مو يدويّ.
	if gm, err := loadGroupMap(ctx, tx, merchantID); err != nil {
		return nil, err
	} else if len(gm) > 0 {
		newGroup := make(map[string]string, len(groupIDs))
		for j, id := range groupIDs {
			newGroup[id] = groupTexts[j]
		}
		var ids, secs []string
		for idx := range touched {
			it := items[idx]
			g := it.group
			if ng, ok := newGroup[it.id]; ok {
				g = ng
			}
			if sec := gm[groupKey(g)]; g != "" && sec != "" && it.source == SourcePending && it.section != sec {
				ids, secs = append(ids, it.id), append(secs, sec)
			}
		}
		if _, err := moveItems(ctx, tx, merchantID, ids, secs, SourceGroup); err != nil {
			return nil, err
		}
	}
	var placed []string
	for _, n := range toCreate {
		if n.source != SourcePending {
			placed = append(placed, n.section)
		}
	}
	if err := addStoreSections(ctx, tx, merchantID, placed); err != nil {
		return nil, err
	}
	made, err := insertNewItems(ctx, tx, merchantID, toCreate)
	if err != nil {
		return nil, err
	}
	rep.Created = made
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
