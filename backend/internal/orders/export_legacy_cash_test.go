package orders

import "context"

// AccrueLegacyCashDueForTest **مستحقٌّ نقديٌّ على حاله قبل قرار ٢٠٢٦-١٠-١٥** — يُنشئ صفَّ
// التسوية `cash_due` بلا دفعِ السائق، كطلبٍ استُلم قبل النشر. **فيبقى بابُ الأدمن
// (`MarkCashSettlementPaid`) والعكسُ قبل الدفع مختبَرَين** — وهما قائمان لما بقي من القديم.
func (s *Service) AccrueLegacyCashDueForTest(ctx context.Context, orderID, actorID string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := s.settleMerchant(ctx, tx, orderID, actorID, nil, &settled{}); err != nil {
		return err
	}
	if err := s.creditTreasury(ctx, tx, orderID, actorID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
