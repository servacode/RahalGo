package server

// تعويضُ الشكوى إلى طابور التعويضات الموحّد — قرارُ المالك ٢٠٢٦-١٠-٠٤.
//
// حلُّ التذكرة ينادي بابَ اقتراحٍ واحداً (`support.CompensationProposer`)،
// وافتراضُه كان طلبَ محفظة. وهنا يُبدَّل بالبابِ الواحدِ لكلّ تعويض
// (`orders.ProposeCompensationTx`، نوعُ «شكوى»)، فيظهر الطلبُ في صفحة التعويضات
// وفي صفحة الموافقات الموحّدة، ولا يتحرّك مالٌ قبل موافقة الماليّة.

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/support"
)

// complaintCompensationProposer غلافٌ رقيقٌ على `orders.ProposeCompensationTx`.
type complaintCompensationProposer struct{}

func (complaintCompensationProposer) ProposeCompensation(ctx context.Context, q dbtx.Querier,
	p support.CompensationProposal) (string, error) {
	return orders.ProposeCompensationTx(ctx, q, orders.CompensationProposal{
		Kind: orders.CompKindComplaint, TicketID: p.TicketID,
		BeneficiaryID: p.BeneficiaryID, Reason: orders.ReasonComplaint,
		Amount: p.Amount, Note: p.Note, ProposedBy: p.ProposedBy,
	})
}
