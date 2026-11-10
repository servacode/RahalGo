package orders

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// نموذجُ المال على الطلب (`orders.money_model`، هجرة ٠٥٣٢ — خطّةُ المال ٢٠٢٦-١١-١٠، الدفعة B1).
const (
	// MoneyModelLegacy **القديم**: البضاعةُ تمرّ بالدفتر (احتباسٌ · محفظةُ متجر · سلفةُ الدرج).
	MoneyModelLegacy = 1
	// MoneyModelGoodsOutside **الجديد**: البضاعةُ خارجَ كلّ حساب — المنصّةُ تلمس الأجرةَ والعمولةَ وحدَهما.
	MoneyModelGoodsOutside = 2
)

// newOrderMoneyModel **نموذجُ مالِ طلبٍ يُنشأ الآن** — يُختَم مع الإنشاء ولا يتغيّر بعده.
//
// **المفتاحُ `finance.goods_outside` مطفأٌ افتراضاً** ⇒ ١ كما كان كلُّ طلب. **ولا شيءَ في هذه
// الدفعة يقرأ ٢** — الختمُ وحدَه، لتُبنى الدفعاتُ التالية عليه وتُشغَّل معاً.
//
// **ويُقرأ المفتاحُ بمنفّذ الإنشاء نفسِه** (`q`) لا بالمَسبَح: الإنشاءُ داخلَ معاملةٍ تمسك اتّصالاً،
// **وقراءةٌ من المَسبَح تطلب ثانياً** — فيسقط الإنشاءُ بـ٥٠٣ تحت مَسبَحٍ ضيّق (`TestXG48_…`).
func (s *Service) newOrderMoneyModel(ctx context.Context, q dbtx.Querier) int16 {
	if s.settings != nil && s.settings.On(q).GetBool(ctx, "finance.goods_outside") {
		return MoneyModelGoodsOutside
	}
	return MoneyModelLegacy
}
