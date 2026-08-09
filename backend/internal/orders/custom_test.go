package orders

// **الطلبُ الخاصّ لا يحرّك مالاً في المنصّة.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «السعر والأجرة لن تدخل بالحسابات، لأنّها خدمة
//
//	للسائق فقط».)
//
// # وهذا وعدٌ يجب أن يُحرَس
//
// **السائقُ يدفع من جيبه ويستردّ عند التسليم** — ولا عمولةَ ولا مستحقَّ متجرٍ
// ولا أجرَ توصيلٍ ولا قيدَ صندوق.
//
// **ووعدٌ ماليٌّ لا يحرسه شيءٌ يُنقَض بسطرٍ يُضاف بعد شهر**: يُوصَل الطلبُ
// الخاصُّ بمسار التسوية «ليُحسب أجرُ السائق كالبقيّة»، **فيصير للمنصّة دخلٌ من
// خدمةٍ قالت إنّها لا تأخذ منها شيئاً** — ولا يظهر ذلك في شاشةٍ ولا في خطأ.
//
// # ولا قاعدةَ تُفتح
//
// **`settle` هي بابُ المال الوحيد** في انتقالات الطلب — ويُفحص أنّها تخرج
// صفراً للخاصّ. **ودالّةٌ تُنادي القاعدةَ لا يبلغها اختبار**، فيُفحص القرارُ
// وحدَه: **أيمرّ الخاصُّ من الباب أم لا.**

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// errWouldSpend **قيدٌ حاول أن يقع** — أيُّ نداءٍ للقاعدة في `settle` يعني
// أنّ المال تحرّك.
var errWouldSpend = errors.New("الطلبُ الخاصُّ حرّك مالاً")

// spyQuerier **منفّذٌ يشي بمن يناديه** — ولا ينفّذ شيئاً.
//
// **ولا مخزنَ ولا قاعدة**: يُختبَر القرارُ لا الاستعلام.
type spyQuerier struct{ touched bool }

func (q *spyQuerier) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	q.touched = true
	return pgconn.CommandTag{}, errWouldSpend
}

func (q *spyQuerier) QueryRow(context.Context, string, ...any) pgx.Row {
	q.touched = true
	return errRow{}
}

func (q *spyQuerier) Query(context.Context, string, ...any) (pgx.Rows, error) {
	q.touched = true
	return nil, errWouldSpend
}

// errRow صفٌّ يردّ خطأً — فلا يمضي المسارُ على قراءةٍ وهميّة.
type errRow struct{}

func (errRow) Scan(...any) error { return errWouldSpend }

func TestCustomOrderMovesNoMoney(t *testing.T) {
	s := &Service{}
	for _, to := range []string{StPickedUp, StDelivered, StCancelled, StFailed} {
		t.Run(to, func(t *testing.T) {
			q := &spyQuerier{}
			var out settled
			err := s.settle(context.Background(), q, settlement{
				orderID: "o1", from: StOnTheWay, to: to,
				actorID: "a", customerID: "c",
				// **وأرقامٌ غيرُ صفريّةٍ عمداً** — لو مرّ من الباب لتحرّكت.
				//
				// **وصفرٌ في الحقول يجعل الاختبارَ يمرّ بلا أن يفحص شيئاً**:
				// لا شيءَ يتحرّك أصلاً، **فيُقرأ نجاحاً وهو صمت.**
				walletPaid: 5000, cashDue: 8000, deliveryFee: 3000,
				custom: true,
			}, &out)
			if err != nil {
				t.Fatalf("الطلبُ الخاصُّ يجب أن يخرج بلا أثر: %v", err)
			}
			if q.touched {
				t.Error("**حرّك مالاً** — والمنصّةُ توثّق ولا تحاسب في الطلب الخاصّ")
			}
		})
	}
}

// TestStandardOrderStillSettles **والعاديُّ يمرّ كما كان.**
//
// **وحارسٌ يمنع الخاصَّ ولا يتحقّق من العاديّ نصفُ حارس**: من عطّل التسويةَ
// كلَّها يمرّ عنده، **فتصير المنصّةُ بلا دخلٍ ولا يُنذر أحد.**
func TestStandardOrderStillSettles(t *testing.T) {
	s := &Service{}
	q := &spyQuerier{}
	var out settled
	_ = s.settle(context.Background(), q, settlement{
		orderID: "o1", from: StOnTheWay, to: StCancelled,
		actorID: "a", customerID: "c", walletPaid: 5000,
		custom: false,
	}, &out)
	if !q.touched {
		t.Error("الطلبُ العاديُّ لم يمسّ المال — أعُطّلت التسويةُ كلُّها؟")
	}
}

// TestCustomGoesStraightToTheQueue **موافقةٌ واحدةٌ تُنزله الطابور.**
//
// (تصحيحُ المالك ٢٠٢٦-٠٨-٠٩: «لا يوجد تحويل للمتجر، هذا خطأ — لأنّه بالأساس
//
//	الطلبُ ليس من متجر».)
//
// **«مقبول» تعني قَبِله المتجر و«تحضير» تعني يطبخه** — ولا متجرَ هنا.
// **فكانت الإدارةُ تضغط مرّتين**: تقبل نيابةً عن لا أحد، ثمّ تحوّل. **وحالةٌ
// بلا معنًى تُقرأ خبراً كاذباً**: من رأى «قيد التحضير» ظنّ أنّ مطبخاً يعمل.
func TestCustomGoesStraightToTheQueue(t *testing.T) {
	ops := []string{"ops"}

	if !canTransition(KindCustom, StPending, StDispatching, ops) {
		t.Error("**الخاصُّ لا ينزل الطابورَ بموافقةٍ واحدة** — وهو نصُّ قرار المالك")
	}
	if canTransition(KindCustom, StPending, StAccepted, ops) {
		t.Error("**الخاصُّ يقبل حالةَ «مقبول»** — ولا متجرَ يقبله")
	}
	if canTransition(KindCustom, StAccepted, StPreparing, ops) {
		t.Error("**الخاصُّ يقبل «تحضير»** — ولا مطبخَ يحضّره")
	}

	// **والعاديُّ لم يتبدّل** — حارسٌ يفتح للخاصّ ويكسر العاديَّ نصفُ حارس.
	if canTransition(KindStandard, StPending, StDispatching, ops) {
		t.Error("**العاديُّ صار ينزل الطابورَ بلا قبولٍ من متجره**")
	}
	if !canTransition(KindStandard, StPending, StAccepted, ops) {
		t.Error("**العاديُّ لم يعد يُقبَل** — أعُطّلت خارطتُه؟")
	}

	// **ومراحلُ الطريق واحدةٌ في النوعين** — الفرقُ في أوّله لا في طريقه.
	if !canTransition(KindCustom, StAtDropoff, StDelivered, []string{"driver"}) {
		t.Error("**الخاصُّ لا يُسلَّم** — ومراحلُ الطريق مشتركة")
	}
}
