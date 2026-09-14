package orders

// ══════════════════════════════════════════════════════════════════════
// **ما تبدّل منذ أن رآه — باسمه لا بـ«حدث خطأ»** (`CC`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// # المسألة
//
// **والمحرّكُ يعرف أيَّ صنفٍ نفد وأيَّ سعرٍ تبدّل** — **ثمّ يردّ رمزاً
// واحداً لا يقول أيَّها.** **فتقرأ الشاشةُ «هذا الصنف غير متاح» ولا
// تعرف أيَّ سطرٍ تُعلّم**، **فتقول «حدث خطأ» أو تمحو السلّةَ كلَّها.**
//
// **ومن مُحيت سلّتُه لأنّ صنفاً واحداً نفد خسر عشرةَ اختيارات.**
//
// # ولا محرّكَ تسعيرٍ ثانٍ
//
// **والحسبةُ هي `priceItems` عينُها** — **تُنادى لكلّ سطرٍ على حدةٍ
// فيُترجَم خطؤها إلى تبدّلٍ باسمه.**
//
// **ولو كُتبت حسبةٌ ثانيةٌ للتشخيص لَافترقت عن حسبة الإنشاء يوماً** —
// **فتقول الشاشةُ «لا تبدّل» ويردّ الإنشاءُ** — **وهو أسوأُ ما يقع.**
//
// # ولا يُقرَّر شيءٌ هنا
//
// **هذه تُخبِر والمنعُ عند الإنشاء** — **وسطرٌ يُعلَّم لا يُحذَف**:
// **صاحبُ السلّة يقرّر ما يفعل بما تبدّل.**

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// أنواعُ التبدّل — **ولا يُخترَع نوعٌ لسلوكٍ لا يقع.**
const (
	// ChangeItemRemoved **صنفٌ لم يعد في القائمة.**
	ChangeItemRemoved = "product_removed"
	// ChangeItemUnavailable **صنفٌ موقوفٌ أو نفد.**
	ChangeItemUnavailable = "product_unavailable"
	// ChangeQtyInvalid **عددٌ لم يعد مقبولاً.**
	ChangeQtyInvalid = "quantity_invalid"
	// ChangeItemPrice **تبدّل سعرُ الصنف عمّا عُرض.**
	ChangeItemPrice = "product_price_changed"
	// ChangeDeliveryFee **تبدّلت أجورُ التوصيل.**
	ChangeDeliveryFee = "delivery_fee_changed"
	// ChangePromo **تبدّل الخصمُ أو سقط.**
	ChangePromo = "promo_or_discount_changed"
)

// Change **تبدّلٌ واحدٌ باسمه.**
//
// **ويُسمّى الصنفُ ليُعلَّم سطرُه** — **ومن قال «تبدّل شيءٌ» ولم يقل
// أيُّه ترك صاحبَه يقرأ سلّتَه سطراً سطرا.**
type Change struct {
	Type string `json:"type"`
	// MenuItemID و Name **للتبدّلات التي تخصّ صنفاً.**
	MenuItemID string `json:"menu_item_id,omitempty"`
	Name       string `json:"name,omitempty"`
	Qty        int    `json:"qty,omitempty"`
	// OldValue ما كان معروضاً عند صاحب السلّة · NewValue ما يقوله
	// المحرّكُ الآن. **والاثنان يُعرضان معاً**: **«تبدّل من كذا إلى
	// كذا» تُفهَم، و«تبدّل السعر» لا تُفهَم.**
	OldValue int64 `json:"old_value"`
	NewValue int64 `json:"new_value"`
}

// Expected **ما كان معروضاً على الشاشة حين راجعه.**
//
// **ويُرسله العميلُ ليُقارَن** — **ولا يُصدَّق منه حكمٌ**: **يُقارَن
// به فقط، والحقيقةُ من المحرّك.**
type Expected struct {
	// Lines سعرُ الوحدة المعروضُ لكلّ صنف.
	Lines map[string]int64 `json:"lines"`
	// DeliveryFee و Discount **بالسالب يعني «لم يُرسَل»** — **وصفرٌ
	// قيمةٌ صحيحةٌ لا غياب.**
	DeliveryFee int64 `json:"delivery_fee"`
	Discount    int64 `json:"discount"`
	Has         bool  `json:"has"`
}

// CartChanges **ما تبدّل بين ما رآه وما يقوله المحرّكُ الآن.**
//
// **وفارغةٌ تعني: لا شيءَ تبدّل** — **ولا تعني أنّ الطلبَ مقبول**:
// **الإتاحةُ سؤالٌ آخرُ يجيبه `AvailabilityAt`.**
func (s *Service) CartChanges(
	ctx context.Context, q dbtx.Querier, items []ItemInput, exp Expected,
) []Change {
	out := []Change{}
	if !exp.Has {
		return out
	}
	for _, in := range items {
		// **وسطرٌ سطرٌ** — **فخطأُ الأوّل لا يُخفي تبدّلَ الثاني.**
		// (`CA-06`: **تبدّلان يُعرضان معاً لا الأوّلُ وحدَه.**)
		priced, _, err := s.priceItems(ctx, q, []ItemInput{in})
		switch {
		case errors.Is(err, ErrItemGone):
			out = append(out, Change{
				Type:       ChangeItemRemoved,
				MenuItemID: in.MenuItemID,
				Name:       s.itemName(ctx, q, in.MenuItemID),
				Qty:        in.Qty,
			})
			continue
		case errors.Is(err, ErrItemUnavailable):
			out = append(out, Change{
				Type:       ChangeItemUnavailable,
				MenuItemID: in.MenuItemID,
				Name:       s.itemName(ctx, q, in.MenuItemID),
				Qty:        in.Qty,
			})
			continue
		case errors.Is(err, ErrBadQty):
			out = append(out, Change{
				Type:       ChangeQtyInvalid,
				MenuItemID: in.MenuItemID,
				Name:       s.itemName(ctx, q, in.MenuItemID),
				Qty:        in.Qty,
			})
			continue
		case err != nil:
			// **وعطبٌ لا يُترجَم لا يُخترَع له نوع** — **ولا يُقال
			// للزبون إنّ صنفَه نفد والعلّةُ في قاعدتنا.**
			continue
		}
		if len(priced) == 0 {
			continue
		}
		was, ok := exp.Lines[in.MenuItemID]
		if !ok {
			continue
		}
		// **وسعرُ الوحدة بخياراته** — **كما تعرضه الشاشةُ وكما
		// يُقيَّد في الطلب.**
		now := priced[0].UnitPrice
		if was != now {
			out = append(out, Change{
				Type:       ChangeItemPrice,
				MenuItemID: in.MenuItemID,
				Name:       priced[0].Name,
				Qty:        in.Qty,
				OldValue:   was,
				NewValue:   now,
			})
		}
	}
	return out
}

// FeeChange **أتبدّلت أجورُ التوصيل عمّا عُرض؟**
//
// **وتُقاس على التسعيرة المحسوبة للتوّ** — **لا على استعلامٍ ثانٍ.**
func FeeChange(exp Expected, fee int64) []Change {
	if !exp.Has || exp.DeliveryFee < 0 || exp.DeliveryFee == fee {
		return nil
	}
	return []Change{{
		Type:     ChangeDeliveryFee,
		OldValue: exp.DeliveryFee,
		NewValue: fee,
	}}
}

// PromoChange **أسقط الخصمُ أو تبدّل؟**
//
// **ووعدٌ بخصمٍ سقط أسوأُ من لا خصم** — **ومن قرأ مجموعاً ثمّ دُفع
// غيرُه ظنّ أنّه خُدع.**
func PromoChange(exp Expected, discount int64) []Change {
	if !exp.Has || exp.Discount < 0 || exp.Discount == discount {
		return nil
	}
	return []Change{{
		Type:     ChangePromo,
		OldValue: exp.Discount,
		NewValue: discount,
	}}
}

// itemName **اسمُ الصنف ليُعلَّم سطرُه** — وفارغٌ لما لم يعد موجوداً.
func (s *Service) itemName(ctx context.Context, q dbtx.Querier, id string) string {
	var name string
	err := q.QueryRow(ctx, `SELECT name FROM menu_items WHERE id = $1`, id).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) || err != nil {
		return ""
	}
	return name
}
