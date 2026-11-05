package orders

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// GoodsLeftStore **أخرجت البضاعةُ من المتجر؟** — خلل PD-8 (قرارُ المالك ٢٠٢٦-١١-٠٦ «أ»).
//
// **ويُعرف بالأثر لا بالحال**: طلبٌ أُرسل مع سائقٍ آخرَ بعد طارئ عاد إلى «بانتظار سائق»
// (`picked_up → dispatching`) **والبضاعةُ مع الأوّل ومستحقُّ متجره مقيَّدٌ منذ الاستلام.**
// فالحالُ وحدَها كانت تفتح له «التحويلَ لمتجرٍ آخر» و«غير متوفّر» من جديد:
//
//   - **تحويلٌ بعدها** ⇒ الاستلامُ الثاني يكتب تسويةً للمتجر الجديد وتبقى تسويةُ الأوّل —
//     **بضاعةٌ واحدةٌ تُدفع لمتجرين.**
//   - **وحذفُ صنفٍ بعدها** ⇒ يتغيّر مستحقُّ المتجر في الطلب ولا يتغيّر في صفّ تسويته،
//     **فيتعطّل الاستلامُ الثاني بـXS-1** — وتصحيحُ ما بعد الاستلام بابُه «تعديل الطلب».
//
// **ثلاثُ علاماتٍ وأيُّها يكفي**: وقتُ الاستلام (ولو كان المستحقُّ صفراً فلا صفَّ)،
// وصفُّ تسوية، وقيدُ مستحقٍّ في المحفظة (التراثُ قبل صفوف التسوية).
func GoodsLeftStore(ctx context.Context, q dbtx.Querier, orderID string) (bool, error) {
	var left bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM orders WHERE id = $1::uuid AND picked_up_at IS NOT NULL)
		    OR EXISTS(SELECT 1 FROM merchant_settlements WHERE order_id = $1::uuid)
		    OR EXISTS(SELECT 1 FROM wallet_transactions
		               WHERE ref = $1::text AND kind IN ('merchant_earning', 'merchant_cash_accrued'))`,
		orderID).Scan(&left)
	return left, err
}
