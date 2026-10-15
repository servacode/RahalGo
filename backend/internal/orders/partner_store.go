package orders

// ══════════════════════════════════════════════════════════════════════
// **مشترياتٌ من متجرٍ تابعٍ إلنا** — قرارُ المالك ٢٠٢٦-١٠-١٥
// ══════════════════════════════════════════════════════════════════════
//
// طلبُ المشتريات بلا متجر: السائقُ يشتري من حيث شاء، ويوثّق ثمنَ البضاعة (G)، ويدفعه من
// سلفة المكتب. **وبعضُ ما يُطلب يُشترى من متجرٍ بيننا وبينه اتّفاق** — فيربطه المكتبُ
// بالطلب، ويرسل له الطلبَ على واتساب، **ويأخذ عمولتَه:**
//
//	العمولة          C = floor(G × النسبة / 100)   — نسبةُ المتجر ملقوطةً لحظةَ الربط
//	يدفعه السائقُ    G − C                          — للمتجر عند الشراء
//	الخزينة          تطرح (G − C) لا G               — فيبقى لها C ونصيبُها من الأجرة
//	المندوب          نصيبُه من C                     — كما في الطلب العاديّ (`settleRep`)
//
// **والمشترياتُ بلا متجرٍ تابعٍ كما كانت** — لا عمولة.
//
// # ولماذا عمودٌ منفصلٌ لا `merchant_id`
//
// **مساراتٌ كثيرةٌ تفترض أنّ الطلبَ الخاصَّ بلا متجر** — بوّابةُ المتجر تعرضه، والتوزيعُ
// يرسو عليه، ورسالةُ المتجر تقرأ أصنافَه، و`settleMerchant` يقيّد مستحقَّه. **فربطُه في
// `merchant_id` يفتح كلَّ هذه الأبواب على طلبٍ لم تُبنَ له.**
//
// # ولماذا قبل توثيق الثمن وحدَه
//
// **بعد التوثيق دفع السائقُ للمتجر** — فربطٌ بعده يغيّر ما دُفع بعد أن دُفع.

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

var (
	// ErrPartnerNotPurchase **الربطُ لطلب المشتريات وحدَه** — العاديُّ له متجرُه، والأمانةُ بلا ثمن.
	ErrPartnerNotPurchase = httpx.NewError(http.StatusConflict,
		"partner_not_purchase", "errors.partner_not_purchase")
	// ErrPartnerTooLate **لا ربطَ ولا فكَّ بعد توثيق ثمن البضاعة** — السائقُ دفع للمتجر.
	ErrPartnerTooLate = httpx.NewError(http.StatusConflict,
		"partner_goods_documented", "errors.partner_goods_documented")
	// ErrPartnerInactive **المتجرُ غيرُ فعّال** — لا يُرسَل له طلب.
	ErrPartnerInactive = httpx.NewError(http.StatusConflict,
		"partner_merchant_inactive", "errors.partner_merchant_inactive")
)

// PartnerChange **ما وقع في الربط** — للتدقيق.
type PartnerChange struct {
	MerchantID        string `json:"merchant_id"`
	MerchantName      string `json:"merchant_name"`
	CommissionPercent int64  `json:"commission_percent"`
	// PreviousMerchantID **المتجرُ السابق** إن بُدّل أو فُكّ — وفراغٌ إن لم يكن.
	PreviousMerchantID string `json:"previous_merchant_id"`
}

// LinkPartner **يربط متجراً تابعاً بطلب مشتريات** — `merchantID` فارغٌ يفكّ الربط.
//
// **والتدقيقُ في المعاملة نفسِها** (`hook`) — ربطٌ بلا أثرٍ لا يقع، وأثرٌ بلا ربطٍ لا يُكتب.
func (s *Service) LinkPartner(ctx context.Context, actorID, orderID, merchantID string,
	hook func(context.Context, dbtx.Querier, PartnerChange) error) (*Order, PartnerChange, error) {
	var ch PartnerChange
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, ch, httpx.ErrNotFound
	}
	if merchantID != "" {
		if _, err := uuid.Parse(merchantID); err != nil {
			return nil, ch, httpx.ErrNotFound
		}
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, ch, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var kind, mode, status, prev string
	var goods *int64
	var pending, pickedUp bool
	err = tx.QueryRow(ctx, `
		SELECT kind, custom_mode, status, COALESCE(partner_merchant_id::text, ''),
		       custom_goods_amount, custom_goods_pending, picked_up_at IS NOT NULL
		FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&kind, &mode, &status, &prev, &goods, &pending, &pickedUp)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ch, httpx.ErrNotFound
	}
	if err != nil {
		return nil, ch, err
	}
	if kind != KindCustom || mode != CustomModePurchase {
		return nil, ch, ErrPartnerNotPurchase
	}
	// **والثمنُ موثَّقٌ حين يُكتب ولا ينتظر** — صفرُ الانتظار (`custom_goods_pending`)
	// يُكتب مع الأجرة وليس ثمناً. **وما بعد الاستلام أو النهاية مغلقٌ كذلك.**
	if (goods != nil && !pending) || pickedUp || terminal(status) {
		return nil, ch, ErrPartnerTooLate
	}
	ch.PreviousMerchantID = prev

	if merchantID == "" {
		if _, err := tx.Exec(ctx, `
			UPDATE orders SET partner_merchant_id = NULL, partner_commission_percent = NULL,
			       partner_linked_at = NULL, updated_at = now() WHERE id = $1`, orderID); err != nil {
			return nil, ch, err
		}
	} else {
		var mstatus string
		var override *int64
		err := tx.QueryRow(ctx, `
			SELECT name, status, commission_percent::bigint FROM merchants
			 WHERE id = $1`, merchantID).
			Scan(&ch.MerchantName, &mstatus, &override)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ch, httpx.ErrNotFound
		}
		if err != nil {
			return nil, ch, err
		}
		if mstatus != "active" {
			return nil, ch, ErrPartnerInactive
		}
		// **نسبةُ المتجر إن خُصّ بها، وإلّا العامّةُ** — ملقوطةً الآن، **فتعديلٌ لاحقٌ في
		// المتجر أو في الإعداد العامّ لا يبدّل ما اتُّفق عليه في هذا الطلب.**
		ch.MerchantID = merchantID
		ch.CommissionPercent = pricing.MerchantCommission(ctx, s.settings, override).Value
		if _, err := tx.Exec(ctx, `
			UPDATE orders SET partner_merchant_id = $2, partner_commission_percent = $3,
			       partner_linked_at = now(), updated_at = now() WHERE id = $1`,
			orderID, merchantID, ch.CommissionPercent); err != nil {
			return nil, ch, err
		}
	}
	if hook != nil {
		if err := hook(ctx, tx, ch); err != nil {
			return nil, ch, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, ch, err
	}
	o, err := s.GetByID(ctx, orderID)
	if err != nil {
		return nil, ch, err
	}
	s.publishOrder(o)
	return o, ch, nil
}

// customStoreCut **ما يدفعه السائقُ للمتجر في طلب المشتريات، وعمولةُ المنصّة منه.**
//
//	بلا متجرٍ تابع  ←  الثمنُ كاملاً، ولا عمولة
//	مع متجرٍ تابع   ←  الثمنُ ناقصَ العمولة، والعمولةُ بالنسبة الملقوطة
//
// **ومصدرٌ واحدٌ للتسوية وللسائق** (`payMerchantsFor`) — **فلا يفترق رقمُ الشاشة عن رقم
// الدفتر.** والأمانةُ وغيرُ الخاصّ صفران.
func customStoreCut(ctx context.Context, q dbtx.Querier, orderID string) (storePaid, commission int64, err error) {
	var goods int64
	var pct *int64
	if err := q.QueryRow(ctx, `
		SELECT CASE WHEN kind = 'custom' AND custom_mode = 'purchase'
		            THEN COALESCE(custom_goods_amount, 0) ELSE 0 END,
		       CASE WHEN partner_merchant_id IS NOT NULL THEN partner_commission_percent END
		FROM orders WHERE id = $1`, orderID).Scan(&goods, &pct); err != nil {
		return 0, 0, err
	}
	if pct != nil {
		commission = pricing.Amount{Mode: "percent", Value: *pct}.Of(goods)
	}
	return goods - commission, commission, nil
}
