package server

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// officeCashPayout **سطرٌ في صندوق المكتب لسحبٍ صُرف نقداً من المكتب** —
// قرارُ المالك ٢٠٢٦-١٠-٠٤ (قسمُ طلبات السحب، البند ٦).
//
// **الموضعُ الوحيدُ الذي يكتب فيه قسمُ السحب في صندوق المكتب.** صندوقُ المكتب
// ملكُ قسم الخزينة، وقد نشر `officecash.Record` في فرعه (لم يُدمَج بعد)؛ وعند
// الدمج يُستبدَل هذا الجسمُ بندائها وحدَه، بالمصدرَين نفسَيهما:
//
//	out · payout_paid      صرفٌ نقداً من الدرج
//	in  · payout_reversed  سحبٌ نقديٌّ رجع إلى الدرج
//
// **والحوالةُ لا تكتب هنا** — المالُ لم يخرج من الدرج.
// والفهرسُ الفريدُ (source, ref) يمنع سطرين لصرفٍ واحد.
func (s *Server) officeCashPayout(ctx context.Context, q dbtx.Querier, direction string,
	amount int64, source, ref, userID, actor, note string) error {
	_, err := q.Exec(ctx, `
		INSERT INTO office_cash_entries (direction, amount, source, ref, user_id, recorded_by, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (source, ref) DO NOTHING`,
		direction, amount, source, ref, userID, actor, note)
	return err
}
