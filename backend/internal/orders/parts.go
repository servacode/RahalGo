package orders

// ══════════════════════════════════════════════════════════════════════
// **أقسامُ الطلب — صفٌّ لكلّ متجر** (قرارُ المالك ٢٠٢٦-١١-٠٥: «١ موافق»)
// ══════════════════════════════════════════════════════════════════════
//
// طلبٌ واحد · محطّةُ استلامٍ لكلّ متجر · سائقٌ واحد · أجرةٌ واحدة · كودُ تسليمٍ
// واحد · دردشةٌ واحدة · تقييمٌ واحد.
//
// # متى يُكتب القسم
//
// **عند الإنشاء وحدَه، وللطلب من متجرين فأكثر وحدَه** (`order_parts`، هجرة ٠٥٠٢):
//
//	متجرٌ واحد       ⇒ لا صفّ · `stops = 1` — **ومسارُ اليوم حرفاً**
//	متجران فأكثر     ⇒ صفٌّ لكلّ متجر · `stops = n` — **والأوّلُ `seq = 1`**
//
// **ولماذا لا صفَّ لكلّ طلب**: **لا يتبدّل شيءٌ لطلبات اليوم** (الإنتاجُ
// على متجرٍ واحد)، **ولا يُملأ تاريخ**، **والرجوعُ سقفٌ يُنزَل إلى الواحد.**
// **ولا يصير طلبٌ من متجرٍ واحدٍ طلبَ متجرين بعد إنشائه**: التعديلُ لا يُدخل
// متجراً جديداً (`order_edit.go`).
//
// # وما بعد هذه الدفعة
//
// **لا شيءَ يقرأ الأقسامَ بعد**: قبولُ كلّ متجرٍ لقسمه والاستلامُ لكلّ محطّةٍ
// في الدفعات التالية. **فالصفُّ يُكتب `pending` ويبقى** — والطلبُ يسير
// بمساره القائم.

import (
	"context"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// partOrder **ترتيبُ الأقسام** — المتجرُ الأوّلُ (`orders.merchant_id`) أوّلاً،
// ثمّ البقيّةُ بترتيب السلّة.
//
// **والأوّلُ هو ما صار متجرَ الطلب** — أوّلُ ما في السلّة، **أو ما أرسله
// المكتبُ من مصادرها** (`CreateTx`). **فـ`seq = 1` يطابق `orders.merchant_id`
// دائماً** — ولا يُقرأ «الأوّلُ» من موضعين بمعنيين.
func partOrder(primary string, sources []string) []string {
	out := make([]string, 0, len(sources))
	out = append(out, primary)
	for _, id := range sources {
		if !strings.EqualFold(id, primary) {
			out = append(out, id)
		}
	}
	return out
}

// insertPartsTx **يكتب صفَّ كلِّ متجرٍ في طلبٍ من متجرين فأكثر** — `pending`.
//
// **وفي معاملة الإنشاء نفسِها** — **فلا طلبَ من متجرين بلا أقسامه**، ولا
// أقسامَ بلا طلب.
func insertPartsTx(ctx context.Context, q dbtx.Querier, orderID, primary string, sources []string) error {
	_, err := q.Exec(ctx, `
		INSERT INTO order_parts (order_id, merchant_id, seq)
		SELECT $1::uuid, p.merchant_id, p.seq::smallint
		FROM unnest($2::uuid[]) WITH ORDINALITY AS p(merchant_id, seq)`,
		orderID, partOrder(primary, sources))
	return err
}
