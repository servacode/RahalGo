package orders

// ══════════════════════════════════════════════════════════════════════
// **التوصيلُ المجانيّ — عرضٌ بفترة** (قرارُ المالك ٢٠٢٦-١٠-٢٣)
// ══════════════════════════════════════════════════════════════════════
//
// يُشغَّل من اللوحة ببدايةٍ ونهاية، ويختار ما يشمل: العاديّ والمشتريات
// والأمانة و«لدي توصيلة». **مرّةٌ واحدةٌ لكلّ زبونٍ بكلّ عرض** — ببصمة رقمه
// فلا يتكرّر بحسابٍ جديد — **ولكلّ متجرٍ** في «لدي توصيلة».
//
// # والمال
//
// **السائقُ يأخذ أجرتَه كاملةً** — يُحسب `driver_fee` من الأجرة قبل الإعفاء.
// **والمعفى يُكتب في `promo_delivery_waived`** كإعفاء كود «توصيل مجاني»،
// **والخزينةُ تتحمّله بالحساب الفرقيّ** (`creditTreasury`: المدفوعُ ناقصٌ
// والأطرافُ أخذت حقّها). **فلا قيدَ مخترَع.**
//
// # والإلغاء
//
// طلبٌ أُلغي أو رُفض **يُحرّر حقَّ صاحبه** — يُحذف صفُّه عند أوّل فحصٍ بعده.

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// أنواعُ ما يشمله العرض — كما تُكتب في `free_delivery_campaigns.kinds`.
const (
	FreeKindStandard = "standard"
	FreeKindPurchase = "purchase"
	FreeKindAmanah   = "amanah"
	FreeKindMerchant = "merchant"
)

// freeOffer **ما يُعفى منه الآن** — وعرضُه، ليُثبَّت بعد إدراج الطلب.
type freeOffer struct {
	campaignID string
	claimant   string
	waive      int64
}

// customerClaimant **بصمةُ رقم الزبون** — ومعرّفُه إن لم يكن له رقم.
func customerClaimant(ctx context.Context, q wallet.Querier, customerID string) string {
	if customerID == "" {
		return ""
	}
	var c string
	if err := q.QueryRow(ctx,
		`SELECT COALESCE(phone_hash_of($1::uuid), $1::text)`, customerID).Scan(&c); err != nil {
		return ""
	}
	return c
}

// merchantClaimant **المتجرُ صاحبُ «لدي توصيلة»**.
func merchantClaimant(merchantID string) string {
	if merchantID == "" {
		return ""
	}
	return "m:" + merchantID
}

// waiveOf **ما يُعفى من أجرةٍ بسقف العرض.**
func waiveOf(fee int64, maxFee *int64) int64 {
	if fee <= 0 {
		return 0
	}
	if maxFee != nil && *maxFee < fee {
		return *maxFee
	}
	return fee
}

// freeDeliveryOffer **أله توصيلةٌ مجانيّةٌ الآن؟** — عرضٌ فعّالٌ يشمل النوع، ولم
// ينلها صاحبُها بطلبٍ حيّ. **ويُقفل الثنائيَّ (العرض، صاحبه) حتّى نهاية المعاملة**
// — فطلبان متزامنان لا ينالانها معاً.
//
// `lock=false` للتسعيرة وحدَها (لا معاملةَ تُكتب بعدها).
func (s *Service) freeDeliveryOffer(ctx context.Context, q wallet.Querier,
	kind, claimant string, fee int64, lock bool) (freeOffer, error) {
	if fee <= 0 || claimant == "" {
		return freeOffer{}, nil
	}
	var id string
	var maxFee *int64
	err := q.QueryRow(ctx, `
		SELECT id::text, max_fee FROM free_delivery_campaigns
		 WHERE active AND now() >= starts_at AND now() < ends_at AND $1 = ANY(kinds)
		 ORDER BY created_at DESC LIMIT 1`, kind).Scan(&id, &maxFee)
	if errors.Is(err, pgx.ErrNoRows) {
		return freeOffer{}, nil
	}
	if err != nil {
		return freeOffer{}, err
	}
	if lock {
		if _, err := q.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1 || '|' || $2))`,
			id, claimant); err != nil {
			return freeOffer{}, err
		}
	}
	var taken bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM free_delivery_claims c JOIN orders o ON o.id = c.order_id
		                WHERE c.campaign_id = $1 AND c.claimant = $2
		                  AND o.status NOT IN ('cancelled', 'rejected'))`, id, claimant).Scan(&taken); err != nil {
		return freeOffer{}, err
	}
	if taken {
		return freeOffer{}, nil
	}
	return freeOffer{campaignID: id, claimant: claimant, waive: waiveOf(fee, maxFee)}, nil
}

// claim **يُثبّت التوصيلةَ المجانيّةَ على طلبها** — بعد إدراجه. وحقٌّ حرّره
// طلبٌ ملغى يُحذف صفُّه أوّلاً.
func (o freeOffer) claim(ctx context.Context, q wallet.Querier, orderID, kind string) error {
	if o.waive <= 0 {
		return nil
	}
	if _, err := q.Exec(ctx, `
		DELETE FROM free_delivery_claims c USING orders o
		 WHERE o.id = c.order_id AND c.campaign_id = $1 AND c.claimant = $2
		   AND o.status IN ('cancelled', 'rejected')`, o.campaignID, o.claimant); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `
		INSERT INTO free_delivery_claims (campaign_id, claimant, order_id, kind, waived)
		VALUES ($1, $2, $3, $4, $5)`, o.campaignID, o.claimant, orderID, kind, o.waive)
	return err
}

// freeDeliveryReapply **أجرةُ طلبٍ تبدّلت وله توصيلةٌ مجانيّة** — يُعاد الإعفاءُ على
// الأجرة الجديدة بسقف عرضه نفسِه، **ولو انتهى العرضُ بعد الطلب** (وعدٌ قُطع يومَه).
// يردّ صفراً إن لم يكن للطلب حقّ.
func (s *Service) freeDeliveryReapply(ctx context.Context, q wallet.Querier, orderID string, fee int64) (int64, error) {
	var maxFee *int64
	err := q.QueryRow(ctx, `
		SELECT f.max_fee FROM free_delivery_claims c
		  JOIN free_delivery_campaigns f ON f.id = c.campaign_id
		 WHERE c.order_id = $1`, orderID).Scan(&maxFee)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	w := waiveOf(fee, maxFee)
	if w <= 0 {
		_, err := q.Exec(ctx, `DELETE FROM free_delivery_claims WHERE order_id = $1`, orderID)
		return 0, err
	}
	_, err = q.Exec(ctx, `UPDATE free_delivery_claims SET waived = $2 WHERE order_id = $1`, orderID, w)
	return w, err
}

// freeDeliveryForQuote **إعفاءُ طلبٍ خاصٍّ عند تثبيت أجرته** — حقُّه القائمُ إن كان،
// وإلّا عرضٌ فعّالٌ يناله الآن.
func (s *Service) freeDeliveryForQuote(ctx context.Context, q wallet.Querier,
	orderID, customerID, kind string, fee int64) (int64, error) {
	var has bool
	if err := q.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM free_delivery_claims WHERE order_id = $1)`, orderID).Scan(&has); err != nil {
		return 0, err
	}
	if has {
		return s.freeDeliveryReapply(ctx, q, orderID, fee)
	}
	off, err := s.freeDeliveryOffer(ctx, q, kind, customerClaimant(ctx, q, customerID), fee, true)
	if err != nil {
		return 0, err
	}
	return off.waive, off.claim(ctx, q, orderID, kind)
}

// FreeDeliveryPreview **ما يُعفى منه الزبونُ لو طلب الآن** — للتسعيرة قبل الطلب.
// زبونٌ مجهولٌ يرى العرضَ إن كان فعّالاً.
func (s *Service) FreeDeliveryPreview(ctx context.Context, customerID, kind string, fee int64) int64 {
	claimant := customerClaimant(ctx, s.db, customerID)
	if claimant == "" {
		claimant = "anon"
	}
	off, err := s.freeDeliveryOffer(ctx, s.db, kind, claimant, fee, false)
	if err != nil {
		return 0
	}
	return off.waive
}
