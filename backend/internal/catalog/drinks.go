package catalog

// ══════════════════════════════════════════════════════════════════════
// **قائمةُ مشروبات المتجر** (قرارُ المالك ٢٠٢٦-١٠-١٩)
// ══════════════════════════════════════════════════════════════════════
//
// «قائمة المشروبات لازم يقدر يحطها المتجر مرة وحدة بعدين تنضاف تلقائي على كل
// الاصناف» — **وبالدولار إن شاء**، فيتبع سعرَ الصرف كالأصناف.
//
// # ولمَ تُنسخ ولا تُحقن
//
// **الطلبُ يتحقّق من الخيار ويسعّره من `modifier_options`** — فالمنسوخةُ خياراتٌ
// عاديّةٌ تمرّ بالطريق القائم، **ولا يُمسّ حسابُ سعر الطلب.** والحقنُ عند القراءة
// كان يحتاج بابين يتّفقان: عرضاً وتسعيراً.
//
// **والمنسوخةُ مُعلَّمة** (`store_drinks`): **لا يراها محرّرُ الصنف** (`GetMenu`)
// **ولا يمحوها حفظُه**، وتُستبدل كلُّها حين تتبدّل القائمة.

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// DrinksGroupName **اسمُ المجموعة كما يقرؤه الزبون.**
const DrinksGroupName = "مشروبات"

// maxDrinks **سقفُ القائمة** — قائمةُ مطعمٍ لا مستودع.
const maxDrinks = 40

// ErrTooManyDrinks **قائمةٌ أطولُ من السقف.**
var ErrTooManyDrinks = httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")

// ErrDrinksNotAllowed **قائمةُ المشروبات للمطاعم وحدَها** (قرارُ المالك ٢٠٢٦-١٠-٣٠).
var ErrDrinksNotAllowed = httpx.NewError(http.StatusBadRequest, "drinks_not_allowed", "errors.drinks_not_allowed")

// Drink **مشروبٌ في القائمة** — `price` بالليرة دائماً، و`price_usd` إن كان دولاريّاً.
//
// **وصورتُه اختياريّة** (٢٠٢٦-١٠-٣٠): تُرسل بمعرّفها (`image_media_id`) وتُقرأ
// برابطيها. **وغيابُ المعرّف أو فراغُه «بلا صورة».**
type Drink struct {
	Name          string   `json:"name"`
	Price         int64    `json:"price"`
	PriceUSD      *float64 `json:"price_usd,omitempty"`
	ImageMediaID  *string  `json:"image_media_id"`
	ImageURL      *string  `json:"image_url"`
	ImageThumbURL *string  `json:"image_thumb_url"`
}

// DrinksAllowed **هل لهذا المتجر قائمةُ مشروبات** — تصنيفُه معلَّمٌ `has_drinks` (المطاعم).
func (s *Service) DrinksAllowed(ctx context.Context, merchantID string) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM merchants m JOIN categories c ON c.id = m.category_id
		                WHERE m.id = $1::uuid AND c.has_drinks)`, merchantID).Scan(&ok)
	return ok, err
}

// Drinks **قائمةُ المتجر كما هي** — **ولغير المطعم فارغة.**
func (s *Service) Drinks(ctx context.Context, merchantID string) ([]Drink, error) {
	rows, err := s.db.Query(ctx, `
		SELECT d.name, d.price, d.price_usd_cents, d.image_media_id::text, md.path, md.thumb_path
		  FROM merchant_drinks d
		  JOIN merchants m ON m.id = d.merchant_id
		  JOIN categories c ON c.id = m.category_id AND c.has_drinks
		  LEFT JOIN media md ON md.id = d.image_media_id
		 WHERE d.merchant_id = $1::uuid ORDER BY d.sort_order`, merchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Drink{}
	for rows.Next() {
		var d Drink
		var cents *int64
		if err := rows.Scan(&d.Name, &d.Price, &cents, &d.ImageMediaID, &d.ImageURL, &d.ImageThumbURL); err != nil {
			return nil, err
		}
		if cents != nil {
			v := float64(*cents) / 100
			d.PriceUSD = &v
		}
		d.ImageURL = media.URLForPtr(d.ImageURL)
		d.ImageThumbURL = media.URLForPtr(d.ImageThumbURL)
		out = append(out, d)
	}
	return out, rows.Err()
}

// SetDrinks **يستبدل القائمةَ وينسخها إلى كلّ الأصناف** — في معاملةٍ واحدة.
//
// **والدولاريُّ يُحسب بسعر صرف المتجر** — ومتجرٌ بلا سعرِ صرفٍ يُردّ
// (`exchange_rate_required`) كالصنف الدولاريّ. **وقائمةٌ فارغةٌ تمحو المجموعةَ من الكلّ.**
func (s *Service) SetDrinks(ctx context.Context, actorID, merchantID string, in []Drink, ip string) ([]Drink, error) {
	if len(in) > maxDrinks {
		return nil, ErrTooManyDrinks
	}
	if len(in) > 0 {
		ok, err := s.DrinksAllowed(ctx, merchantID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrDrinksNotAllowed
		}
	}
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
	clean := make([]Drink, 0, len(in))
	cents := make([]*int64, 0, len(in))
	for _, d := range in {
		d.Name = strings.TrimSpace(d.Name)
		if d.Name == "" {
			return nil, ErrNameRequired
		}
		// **والصورةُ وسيطٌ قائم** — ومعرّفٌ لا يوجد يُردّ تحقّقاً لا ٥٠٠ من المفتاح.
		if d.ImageMediaID != nil {
			if v := strings.TrimSpace(*d.ImageMediaID); v == "" {
				d.ImageMediaID = nil
			} else {
				var found bool
				if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM media WHERE id::text = $1)`, v).Scan(&found); err != nil {
					return nil, err
				}
				if !found {
					return nil, ErrNameRequired
				}
				d.ImageMediaID = &v
			}
		}
		var c *int64
		if d.PriceUSD != nil {
			if *d.PriceUSD < 0 {
				return nil, ErrNameRequired
			}
			if rate == 0 {
				if rate, err = merchantRate(ctx, tx, merchantID); err != nil {
					return nil, err
				}
				if rate <= 0 {
					return nil, ErrExchangeRateRequired
				}
			}
			v := dollarsToCents(*d.PriceUSD)
			c = &v
			d.Price = USDToSYP(v, rate)
		}
		if d.Price < 0 {
			return nil, ErrNameRequired
		}
		if max > 0 && d.Price > max {
			return nil, &httpx.AppError{Status: http.StatusBadRequest, Code: "item_price_too_high",
				MessageKey: "errors.item_price_too_high", Details: map[string]any{"max": strconv.FormatInt(max, 10)}}
		}
		clean = append(clean, d)
		cents = append(cents, c)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM merchant_drinks WHERE merchant_id = $1::uuid`, merchantID); err != nil {
		return nil, err
	}
	for i, d := range clean {
		if _, err := tx.Exec(ctx, `
			INSERT INTO merchant_drinks (merchant_id, name, price, price_usd_cents, sort_order, image_media_id)
			VALUES ($1::uuid, $2, $3, $4, $5, $6::uuid)`, merchantID, d.Name, d.Price, cents[i], i, d.ImageMediaID); err != nil {
			return nil, err
		}
	}
	if err := syncDrinks(ctx, tx, merchantID, ""); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "menu.drinks_update", "merchant", merchantID, ip)
	return s.Drinks(ctx, merchantID)
}

// syncDrinks **ينسخ القائمةَ إلى أصناف المتجر** — أو إلى صنفٍ واحدٍ إن سُمّي.
//
// **يمحو المنسوخةَ القديمة أوّلاً ثمّ يكتب** — فلا تتكرّر المجموعة، **ولا تُمسّ
// مجموعاتُ صاحب المتجر.** وقائمةٌ فارغةٌ تكتفي بالمحو.
//
// **ولا تُنسخ إلّا لمطعم** (قرارُ المالك ٢٠٢٦-١٠-٣٠): تصنيفُه `has_drinks`.
// **والصورةُ تُنسخ مع الخيار** — فيقرؤها الزبونُ من الصنف بلا بابٍ ثانٍ.
func syncDrinks(ctx context.Context, tx dbtx.Querier, merchantID, itemID string) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM modifier_groups g USING menu_items i
		 WHERE g.item_id = i.id AND g.store_drinks
		   AND i.merchant_id = $1::uuid AND ($2 = '' OR i.id::text = $2)`, merchantID, itemID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		WITH g AS (
			INSERT INTO modifier_groups (item_id, name, min_select, max_select, sort_order, store_drinks)
			SELECT i.id, $3, 0, 99, 1000, true FROM menu_items i
			 WHERE i.merchant_id = $1::uuid AND ($2 = '' OR i.id::text = $2)
			   AND NOT i.drinks_off
			   AND EXISTS (SELECT 1 FROM merchant_drinks d WHERE d.merchant_id = i.merchant_id)
			   AND EXISTS (SELECT 1 FROM merchants m JOIN categories c ON c.id = m.category_id
			                WHERE m.id = i.merchant_id AND c.has_drinks)
			RETURNING id
		)
		INSERT INTO modifier_options (group_id, name, price_delta, sort_order, price_delta_usd_cents, image_media_id)
		SELECT g.id, d.name, d.price, d.sort_order, d.price_usd_cents, d.image_media_id
		  FROM g CROSS JOIN merchant_drinks d
		 WHERE d.merchant_id = $1::uuid`, merchantID, itemID, DrinksGroupName); err != nil {
		return err
	}
	return nil
}
