package offers

// بابُ الإدارة — قراراتُ المالك ٢٠٢٦-١٠-٠٤ (قسمُ «العروض والخصومات»).
//
//	٤ · الإدارةُ لا تحمّل الخصمَ على المتجر — المتجرُ يعمل خصمَه من تطبيقه.
//	١ · خصمٌ على المنصّة فوق ٢٠٪ ينتظر موافقةَ الماليّة قبل أن يسري.
//	٥ · الإشعارُ خيارٌ في النافذة بعدد مستلميه، ولزبائن منطقة المتجر وحدَهم.
//	  · النهايةُ آخرُ اليوم بتوقيت دمشق — لا منتصفُ الليل بتوقيت غرينتش.

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// Damascus **توقيتُ الخدمة** — دمشق على +٣ طولَ السنة منذ ٢٠٢٢.
var Damascus = time.FixedZone("Asia/Damascus", 3*60*60)

// ErrBadDate **تاريخٌ لا يُقرأ** — والصيغةُ `YYYY-MM-DD`.
var ErrBadDate = httpx.NewError(http.StatusBadRequest, "bad_date", "errors.bad_date")

// EndOfDay **آخرُ لحظةٍ من يومٍ بتوقيت دمشق** — `YYYY-MM-DD` ⇒ ٢٣:٥٩:٥٩.
//
// **كان التاريخُ يُحفظ منتصفَ ليلِ غرينتش** = الثالثةَ فجراً في سوريا من
// اليوم نفسِه، **فيقف العرضُ أوّلَ اليوم لا آخرَه** — **واختيارُ اليوم نفسِه
// يُرفض لأنّه «مضى».**
func EndOfDay(date string) (time.Time, error) {
	d, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(date), Damascus)
	if err != nil {
		return time.Time{}, ErrBadDate
	}
	return d.Add(24*time.Hour - time.Second), nil
}

// CreatePlatform **عرضٌ من الإدارة** — على المنصّة دائماً، وبموافقةٍ فوق الحدّ.
//
// يردّ العرضَ و`pending` إن أُنشئ نازلاً بانتظار الماليّة. **وطلبُ الموافقة
// يُكتب في المعاملة نفسِها** — لا عرضَ ينتظر بلا طلب، ولا طلبَ بلا عرض.
func (s *Service) CreatePlatform(ctx context.Context, actorID string, in Input,
	marginOf PriceFn) (*Offer, bool, error) {
	if in.BorneBy != nil && *in.BorneBy == ByMerchant {
		return nil, false, ErrAdminChargesStore
	}
	platform := ByPlatform
	in.BorneBy = &platform
	if in.EndsOn != "" {
		end, err := EndOfDay(in.EndsOn)
		if err != nil {
			return nil, false, err
		}
		in.EndsAt = &end
	}
	if in.MenuItemID == nil || strings.TrimSpace(*in.MenuItemID) == "" {
		return nil, false, ErrBadDiscount
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var cost int64
	var itemMargin, sectionMargin *int64
	if err := tx.QueryRow(ctx, `
		SELECT mi.price, mi.margin_override, ps.margin_override
		FROM menu_items mi
		LEFT JOIN platform_sections ps ON ps.id = mi.platform_section_id
		WHERE mi.id = $1::uuid`,
		*in.MenuItemID).Scan(&cost, &itemMargin, &sectionMargin); err != nil {
		return nil, false, ErrBadDiscount
	}
	pending := NeedsApproval(marginOf(cost, itemMargin, sectionMargin), in.DiscountPercent, in.DiscountAmount)
	if pending {
		off := false
		in.Active = &off
	}
	o, err := s.CreateIn(ctx, tx, actorID, in, marginOf)
	if err != nil {
		return nil, false, err
	}
	if pending {
		if _, err := tx.Exec(ctx,
			`UPDATE offers SET approval_state = 'pending' WHERE id = $1::uuid`, o.ID); err != nil {
			return nil, false, err
		}
		// **والكلفةُ القصوى مجهولةٌ لخصم صنفٍ** — لا سقفَ استعمال، فيُكتب صفرٌ
		// والنصُّ يقول الخصمَ للقطعة.
		cut := Cut(o.PriceBefore, in.DiscountPercent, in.DiscountAmount)
		if _, err := tx.Exec(ctx, `
			INSERT INTO promo_approvals (target_kind, target_id, amount, note, proposed_by)
			VALUES ('offer', $1::uuid, 0, $2, $3::uuid)`,
			o.ID, offerApprovalNote(o, cut), actorID); err != nil {
			return nil, false, err
		}
		o.ApprovalState = ApprovalPending
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, err
	}
	return o, pending, nil
}

// offerApprovalNote **ما يقرؤه الموافق** — الصنفُ والخصمُ على القطعة.
func offerApprovalNote(o *Offer, cut int64) string {
	return "خصم على صنف: " + o.ItemName + " — " + o.MerchantName +
		" · الخصم على القطعة " + strconv.FormatInt(cut, 10) + " ل.س"
}

// Decide **قرارُ الماليّة على عرضٍ ينتظر** — في معاملة المنادي.
//
// الموافقةُ تُسريه (وتُنزل أيَّ عرضٍ قائمٍ على الصنف كما يفعل الإنشاء)،
// والرفضُ يُبقيه نازلاً ويمنع تفعيلَه.
func (s *Service) Decide(ctx context.Context, q dbtx.Querier, offerID string, approve bool) error {
	if !approve {
		_, err := q.Exec(ctx, `
			UPDATE offers SET approval_state = 'rejected', active = false, updated_at = now()
			WHERE id = $1::uuid`, offerID)
		return err
	}
	var itemID *string
	var endsAt *time.Time
	err := q.QueryRow(ctx,
		`SELECT menu_item_id::text, ends_at FROM offers WHERE id = $1::uuid FOR UPDATE`, offerID).
		Scan(&itemID, &endsAt)
	if err != nil {
		return err
	}
	active := endsAt == nil || endsAt.After(time.Now())
	if itemID != nil && active {
		if _, err := q.Exec(ctx, `
			UPDATE offers SET active = false, updated_at = now()
			WHERE menu_item_id = $1::uuid AND kind = 'discount' AND active AND id <> $2::uuid`,
			*itemID, offerID); err != nil {
			return err
		}
	}
	_, err = q.Exec(ctx, `
		UPDATE offers SET approval_state = 'ok', active = $2, updated_at = now()
		WHERE id = $1::uuid`, offerID, active)
	return err
}

// ══════════════════════════════════════════════════════════════════════
//
//	**جمهورُ إشعار العرض — زبائنُ منطقة المتجر وحدَهم** (البند ٥)
//
// ══════════════════════════════════════════════════════════════════════
//
// **زبونٌ «في منطقة المتجر»**: حسابُه زبونٌ فعّالٌ بلا دورٍ آخر (كما في
// `NotifyShoppers`)، **وله عنوانٌ محفوظٌ داخل منطقةِ توصيلٍ تغطّي موقعَ
// المتجر، أو طلبٌ سابقٌ في تلك المنطقة.** ومتجرٌ بلا موقعٍ أو خارج كلّ
// منطقةٍ جمهورُه صفر — **ولا يُبلَّغ أحدٌ عن عرضٍ لا يصله.**
const audienceSQL = `
	WITH z AS (
		SELECT dz.id, dz.shape, dz.center, dz.radius_m, dz.area
		FROM delivery_zones dz
		JOIN menu_items mi ON mi.id = $1::uuid
		JOIN merchants m ON m.id = mi.merchant_id
		WHERE dz.active AND m.location IS NOT NULL AND (
		        (dz.shape = 'radius' AND ST_DWithin(dz.center, m.location, dz.radius_m))
		     OR (dz.shape = 'polygon' AND dz.area IS NOT NULL AND ST_Covers(dz.area, m.location)))
	)
	SELECT u.id::text FROM users u
	JOIN user_roles c ON c.user_id = u.id AND c.role_code = 'customer'
	WHERE u.status = 'active'
	  AND NOT EXISTS (SELECT 1 FROM user_roles o
	                   WHERE o.user_id = u.id AND o.role_code <> 'customer')
	  AND (EXISTS (SELECT 1 FROM user_addresses a, z
	                WHERE a.user_id = u.id AND (
	                      (z.shape = 'radius' AND ST_DWithin(z.center, a.location, z.radius_m))
	                   OR (z.shape = 'polygon' AND z.area IS NOT NULL AND ST_Covers(z.area, a.location))))
	    OR EXISTS (SELECT 1 FROM orders ord
	                WHERE ord.customer_id = u.id AND ord.zone_id IN (SELECT id FROM z)))`

// Audience **معرّفاتُ من يصله إشعارُ عرضٍ على هذا الصنف.**
func (s *Service) Audience(ctx context.Context, menuItemID string) ([]string, error) {
	rows, err := s.db.Query(ctx, audienceSQL, menuItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// AudienceCount **عددُهم** — يُعرض في النافذة قبل «انشر».
func (s *Service) AudienceCount(ctx context.Context, menuItemID string) (int, error) {
	var n int
	err := s.db.QueryRow(ctx, `SELECT count(*) FROM (`+audienceSQL+`) a`, menuItemID).Scan(&n)
	return n, err
}
