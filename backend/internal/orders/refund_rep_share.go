package orders

// ══════════════════════════════════════════════════════════════════════
// **استرجاعٌ بعد التسليم: المندوبُ يبقى بعمولته — ومن يتحمّلها؟** (قرارُ المالك ٢٠٢٦-١٠-١٦)
// ══════════════════════════════════════════════════════════════════════
//
// «المندوبُ ما إلو ذنب» — **فلا تُعكَس عمولتُه** (`reverseCommissions`)، لكلّ أنواع الطلبات.
// **والمكتبُ يختار مع الاسترجاع من يحمل نصيبَه:**
//
//	المنصّة  `platform` (الافتراض)  لا شيء — الخزينةُ تتحمّلها بالفرق (`creditTreasury`)
//	المتجر   `merchant`             دينٌ على متجر الطلب بمقدار العمولة
//	السائق   `driver`               دينٌ على سائق الطلب بمقدارها
//
// **والدينُ التزامٌ بسبب `refund_rep_share`** — يُقتطع من أوّل مستحقٍّ قادمٍ كسائر الديون،
// **وحين يُسدَّد يعود إلى الخزينة** (أبوابُ التسوية القائمة). **ولا يُخلق مالٌ بإنشائه.**

import (
	"context"
	"errors"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/obligations"
)

// من يتحمّل نصيبَ المندوب عند الاسترجاع.
const (
	RepSharePlatform = "platform"
	RepShareMerchant = "merchant"
	RepShareDriver   = "driver"
)

// ErrRepShareNoParty **الطرفُ المختارُ غائبٌ عن الطلب** — لا متجرَ له أو لا سائق.
var ErrRepShareNoParty = errors.New("الطرفُ المختارُ لنصيب المندوب غائبٌ عن الطلب")

// IsRepShare **اختيارٌ معروف** — والفارغُ افتراضُه المنصّة.
func IsRepShare(v string) bool {
	switch v {
	case "", RepSharePlatform, RepShareMerchant, RepShareDriver:
		return true
	}
	return false
}

// ChargeRefundRepShareTx **يُحمّل نصيبَ المندوب لمن اختاره المكتب** — في معاملة الاسترجاع
// وبعد تسويتها. ويُرجع مبلغَ الدين (صفرٌ: المنصّةُ تتحمّل، أو لا عمولةَ على الطلب).
//
// **والنصيبُ من الدفتر لا من نسبة**: مجموعُ `commission` بمرجع الطلب — **ما قبضه المندوبُ فعلاً.**
// **والمتجرُ متجرُ الطلب** (`merchant_id`، أو المتجرُ التابعُ في المشتريات) **والسائقُ سائقُه.**
func ChargeRefundRepShareTx(ctx context.Context, q dbtx.Querier, orderID, party, actorID string) (int64, error) {
	if party == "" || party == RepSharePlatform {
		return 0, nil
	}
	var share int64
	var merchantID, driverID *string
	if err := q.QueryRow(ctx, `
		SELECT COALESCE((SELECT sum(t.amount) FROM wallet_transactions t
		                  WHERE t.ref = o.id::text AND t.kind = 'commission'), 0)::bigint,
		       COALESCE(o.merchant_id, o.partner_merchant_id)::text, o.driver_id::text
		  FROM orders o WHERE o.id = $1`, orderID).Scan(&share, &merchantID, &driverID); err != nil {
		return 0, err
	}
	if share <= 0 {
		return 0, nil
	}
	kind, id := obligations.PartyMerchant, merchantID
	if party == RepShareDriver {
		kind, id = obligations.PartyDriver, driverID
	}
	if id == nil || *id == "" {
		return 0, ErrRepShareNoParty
	}
	var actor *string
	if actorID != "" {
		actor = &actorID
	}
	if _, err := obligations.CreateRef(ctx, q, kind, *id, share,
		obligations.CauseRefundRepShare, orderID, orderID, actor); err != nil {
		return 0, err
	}
	return share, nil
}
