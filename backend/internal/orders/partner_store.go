package orders

// ══════════════════════════════════════════════════════════════════════
// **مشترياتٌ من متجرٍ تابعٍ إلنا** — قرارُ المالك ٢٠٢٦-١٠-١٥
// ══════════════════════════════════════════════════════════════════════
//
// طلبُ المشتريات بلا متجر: السائقُ يشتري من حيث شاء، ويوثّق ثمنَ البضاعة (G)، ويدفعه من
// سلفة المكتب. **وبعضُ ما يُطلب يُشترى من متجرٍ بيننا وبينه اتّفاق** — فيربطه المكتبُ
// بالطلب، ويرسل له الطلبَ على واتساب، **ويأخذ عمولتَه:**
//
//	العمولة          C = round(G × النسبة / 100) لأقرب ليرة   — نسبةُ المتجر ملقوطةً لحظةَ الربط
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
	MerchantID   string `json:"merchant_id"`
	MerchantName string `json:"merchant_name"`
	// Commission **النسبةُ الملقوطةُ بمنزلتين** (١٫٥٪) — قرارُ المالك ٢٠٢٦-١٠-١٨.
	Commission pricing.Percent2 `json:"commission_percent_exact"`
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
	// **والربطُ قبل توثيق الثمن وحدَه** — الثمنُ يُكتب من متجرٍ بعينه. **والفكُّ حتّى التسليم**
	// (قرارُ المالك ٢٠٢٦-١٠-٢٥: «خيار الإلغاء يظل موجود بالكرت لآخر لحظة»): زبونٌ غيّر متجرَه
	// إلى غير تابعٍ بعد الشراء ⇒ تسقط العمولة، **والعمولةُ تُحسب عند التسليم من الربط القائم.**
	linkTooLate := (goods != nil && !pending) || pickedUp
	if terminal(status) || (merchantID != "" && linkTooLate) {
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
		// **ولا `::bigint`** — كان يقرّب ١٫٥ إلى ٢ (٢٠٢٦-١٠-١٨).
		var override *pricing.Percent2
		err := tx.QueryRow(ctx, `
			SELECT name, status, commission_percent FROM merchants
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
		ch.Commission = pricing.MerchantCommission(ctx, s.settings, override).Percent()
		if _, err := tx.Exec(ctx, `
			UPDATE orders SET partner_merchant_id = $2, partner_commission_percent = $3,
			       partner_linked_at = now(), updated_at = now() WHERE id = $1`,
			orderID, merchantID, ch.Commission); err != nil {
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
	var pct *pricing.Percent2
	if err := q.QueryRow(ctx, `
		SELECT CASE WHEN kind = 'custom' AND custom_mode = 'purchase'
		            THEN COALESCE(custom_goods_amount, 0) ELSE 0 END,
		       CASE WHEN partner_merchant_id IS NOT NULL THEN partner_commission_percent END
		FROM orders WHERE id = $1`, orderID).Scan(&goods, &pct); err != nil {
		return 0, 0, err
	}
	if pct != nil {
		commission = pricing.Amount{Mode: pricing.ModePercent2, Value: int64(*pct)}.Of(goods)
	}
	return goods - commission, commission, nil
}

// RefreshPartnerCommission **عمولةُ المتجر التابع تتبع نسبتَه الحاليّة حتّى التسليم** (قرارُ المالك
// ٢٠٢٦-١٠-٢٥: «لازم ياخذ العمولة مباشر») — كانت تُلقَط لحظةَ الربط فلا يرى الطلبُ تعديلاً بعده.
// تُنادى بعد تعديل عمولة متجرٍ (`merchantID`) أو العمولةِ العامّة (`merchantID` فارغ ⇒ كلُّ متجرٍ
// بلا نسبةٍ خاصّة). **والطلبُ المسلَّمُ أو المنتهي لا يُمسّ** — عمولتُه قُيّدت.
func (s *Service) RefreshPartnerCommission(ctx context.Context, merchantID string) (int64, error) {
	where := `m.commission_percent IS NULL`
	args := []any{}
	if merchantID != "" {
		where = `m.id = $1`
		args = append(args, merchantID)
	}
	rows, err := s.db.Query(ctx, `SELECT m.id::text, m.commission_percent FROM merchants m WHERE `+where, args...)
	if err != nil {
		return 0, err
	}
	type mc struct {
		id  string
		pct *pricing.Percent2
	}
	var list []mc
	for rows.Next() {
		var x mc
		if err := rows.Scan(&x.id, &x.pct); err != nil {
			rows.Close()
			return 0, err
		}
		list = append(list, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	var n int64
	for _, x := range list {
		pct := pricing.MerchantCommission(ctx, s.settings, x.pct).Percent()
		tag, err := s.db.Exec(ctx, `
			UPDATE orders SET partner_commission_percent = $2, updated_at = now()
			 WHERE partner_merchant_id = $1 AND custom_store_paid IS NULL
			   AND status NOT IN ('delivered', 'cancelled', 'rejected', 'failed', 'refunded')
			   AND partner_commission_percent IS DISTINCT FROM $2`, x.id, pct)
		if err != nil {
			return n, err
		}
		n += tag.RowsAffected()
	}
	return n, nil
}
