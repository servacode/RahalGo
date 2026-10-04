// Package officecash **صندوقُ المكتب** — النقدُ الذي يدخل اليدَ فعلاً ويخرج منها.
//
// ══════════════════════════════════════════════════════════════════════
// **قرارُ المالك ٢٠٢٦-١٠-٠٤ — الخزينة**
// ══════════════════════════════════════════════════════════════════════
//
//	داخلٌ   نقدٌ يسلّمه السائق · شحنُ محفظةٍ نقداً · دفعةُ دَينٍ نقداً
//	خارجٌ   نقدٌ يُدفع لمتجر · سحبٌ يُصرف نقداً · سحبُ الأدمن من الدرج
//
// **وليس محفظةً ولا قيداً في الدفتر** — سجلُّ نقدٍ يقابله قيدُ الدفتر بالمرجع
// نفسِه، **ويُكتب في معاملة القيد نفسِها** فلا يقع أحدُهما دون الآخر.
//
// **ومصدرٌ واحدٌ يكتبه** (`Record`) — كلُّ قسمٍ ينادي هذه الدالّة ولا يكتب
// `INSERT` بيده، فلا يفترق سطرٌ عن أخيه في معناه.
package officecash

import (
	"context"
	"errors"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// Direction اتّجاهُ النقد.
type Direction string

const (
	// In نقدٌ دخل الدرج.
	In Direction = "in"
	// Out نقدٌ خرج منه.
	Out Direction = "out"
)

// مصادرُ السطر — **مغلقةٌ ومسمّاة**؛ مصدرٌ جديدٌ يُضاف هنا وفي معجم الشاشة.
const (
	SourceDriverSettle       = "driver_settle"       // داخلٌ: نقدٌ سلّمه سائق
	SourceWalletTopup        = "wallet_topup"        // داخلٌ: شحنُ محفظةٍ نقداً
	SourceMerchantCashPaid   = "merchant_cash_paid"  // خارجٌ: مستحقٌّ نقديٌّ دُفع لمتجر
	SourcePayoutPaid         = "payout_paid"         // خارجٌ: سحبٌ صُرف
	SourcePayoutReversed     = "payout_reversed"     // داخلٌ: سحبٌ مصروفٌ ارتدّ
	SourceTreasuryWithdrawal = "treasury_withdrawal" // خارجٌ: سحبُ الأدمن من الدرج
	SourceShortfallFound     = "shortfall_found"     // داخلٌ: نقصٌ وُجد بعد الإغلاق
	SourceObligationCash     = "obligation_cash"     // داخلٌ: دفعةُ دَينٍ نقداً في المكتب
	SourceExpenseCash        = "expense_cash"        // خارجٌ: مصروفٌ دُفع نقداً من الدرج
)

// Sources كلُّ المصادر المعروفة واتّجاهُها — يقرؤها الحارسُ والشاشة.
var Sources = map[string]Direction{
	SourceDriverSettle:       In,
	SourceWalletTopup:        In,
	SourceMerchantCashPaid:   Out,
	SourcePayoutPaid:         Out,
	SourcePayoutReversed:     In,
	SourceTreasuryWithdrawal: Out,
	SourceShortfallFound:     In,
	SourceObligationCash:     In,
	SourceExpenseCash:        Out,
}

// Entry سطرٌ واحدٌ في الصندوق.
type Entry struct {
	Direction Direction
	// Amount موجبٌ دائماً — والاتّجاهُ من `Direction`.
	Amount int64
	// Source من `Sources`.
	Source string
	// Ref **مرجعُ القيد المقابل** — معرّفُ الطلب أو السحب أو التسوية. **ومعه
	// المصدرُ مفتاحٌ فريد**: النداءُ الثاني بالمرجع نفسِه لا يكتب شيئاً.
	Ref string
	// UserID الطرفُ المقابل (السائق، صاحبُ المتجر…) — وفارغٌ إن لم يكن.
	UserID string
	// Actor من سجّل.
	Actor string
	Note  string
}

var (
	// ErrInvalid سطرٌ ناقص.
	ErrInvalid = errors.New("officecash: invalid entry")
)

// Record **يكتب سطرَ الصندوق داخل معاملة المستدعي.**
//
// **ويُنادى في المعاملة التي كتبت قيدَ الدفتر** — فإن سقط أحدُهما سقطا معاً.
// **ومكرَّرُه بالمصدر والمرجع نفسَيهما لا يكتب ثانيةً.**
func Record(ctx context.Context, q dbtx.Querier, e Entry) error {
	if e.Amount <= 0 || e.Ref == "" || (e.Direction != In && e.Direction != Out) {
		return ErrInvalid
	}
	if want, ok := Sources[e.Source]; !ok || want != e.Direction {
		return ErrInvalid
	}
	_, err := q.Exec(ctx, `
		INSERT INTO office_cash_entries (direction, amount, source, ref, user_id, recorded_by, note)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::uuid, NULLIF($6, '')::uuid, $7)
		ON CONFLICT (source, ref) DO NOTHING`,
		string(e.Direction), e.Amount, e.Source, e.Ref, e.UserID, e.Actor, e.Note)
	return err
}

// Book **رصيدُ الصندوق في الدفتر** — مجموعُ الداخل ناقصَ الخارج منذ أوّل سطر.
func Book(ctx context.Context, q dbtx.Querier) (int64, error) {
	var b int64
	err := q.QueryRow(ctx, `
		SELECT COALESCE(sum(CASE WHEN direction = 'in' THEN amount ELSE -amount END), 0)::bigint
		FROM office_cash_entries`).Scan(&b)
	return b, err
}
