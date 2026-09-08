package wallet

// ══════════════════════════════════════════════════════════════════════
// **الحجزُ — مالٌ في المحفظة لا يُنفَق** — `XG-12` · `AQ-3`
// ══════════════════════════════════════════════════════════════════════
//
// # والحجزُ ليس خصماً
//
// **لم ينتقل قرشٌ حين وقع** — **والمنصّةُ ما زالت مدينةً بالمبلغ
// لصاحبه.** **فلا يُختلَق قيدٌ في الدفتر ليمثّله**: **الدفترُ حركةُ
// مالٍ، والحجزُ منعُ حركة.**
//
// **وحقيقتُه طلبُ السحب نفسُه** — `wallets.reserved` صورةٌ محفوظةٌ
// للقراءة السريعة يحرسها ثابتٌ دائم، **كما `merchants.debt` صورةٌ
// لـ`financial_obligations`.**
//
// # والمتاحُ يُشتقّ ولا يُخزَّن
//
//	AVAILABLE = POSTED − RESERVED
//
// **ورقمٌ ثالثٌ يُكتب يشيخ حين يُنسى تحديثُه** — **وثلاثةُ أرقامٍ
// لحقيقتين مصدرُ تناقض.**

import (
	"context"
	"errors"
)

// ErrReservedTooMuch **حجزٌ يتجاوز المتاح.**
var ErrReservedTooMuch = errors.New("wallet: المحجوزُ يتجاوز المتاح")

// Layers طبقاتُ المحفظة كما تُقرأ.
type Layers struct {
	// Balance **المُقيَّد** — ما في المحفظة فعلاً.
	Balance int64 `json:"balance"`
	// Reserved **المحجوزُ لعمليّةٍ جارية** — منه لا فوقه.
	Reserved int64 `json:"reserved"`
	// Available **المتاحُ للإنفاق** — مشتقٌّ لا مخزَّن.
	Available int64 `json:"available"`
}

// LayersOf يقرأ الطبقاتِ الثلاث.
//
// **ولا يُسمّى المجموعُ متاحاً** — **ومن عرض الرصيدَ الكلّيَّ باسم
// المتاح كذب على صاحبه.**
func (s *Service) LayersOf(ctx context.Context, userID string) (Layers, error) {
	return s.LayersTx(ctx, s.db, userID)
}

// LayersTx كسابقتها على مُنفِّذٍ معطى.
func (s *Service) LayersTx(ctx context.Context, q Querier, userID string) (Layers, error) {
	var l Layers
	// **وحسابٌ بلا محفظةٍ صفرٌ لا خطأ** — كما في `Balance`.
	err := q.QueryRow(ctx, `
		SELECT COALESCE((SELECT balance  FROM wallets WHERE user_id = $1), 0),
		       COALESCE((SELECT reserved FROM wallets WHERE user_id = $1), 0)`,
		userID).Scan(&l.Balance, &l.Reserved)
	if err != nil {
		return Layers{}, err
	}
	l.Available = l.Balance - l.Reserved
	return l, nil
}

// AvailableTx **ما يجوز إنفاقُه الآن.**
//
// **وكلُّ سؤالٍ «أيقدر أن ينفق كذا؟» يُجاب بهذه لا بالرصيد** —
// **والرصيدُ يشمل ما حُجز لغير هذا الإنفاق.**
func (s *Service) AvailableTx(ctx context.Context, q Querier, userID string) (int64, error) {
	l, err := s.LayersTx(ctx, q, userID)
	if err != nil {
		return 0, err
	}
	return l.Available, nil
}

// Available كسابقتها على البِركة.
func (s *Service) Available(ctx context.Context, userID string) (int64, error) {
	return s.AvailableTx(ctx, s.db, userID)
}

// ReserveTx يحجز مبلغاً — **ولا يمسّ الرصيدَ ولا الدفتر.**
//
// # ولماذا صفٌّ يُقفل
//
// **طلبان متزامنان يقرآن المتاحَ نفسَه فيحجزان ضعفَه** — **والقفلُ
// على صفّ المحفظة يجعل الثانيَ يقرأ بعد أن كُتب الأوّل.**
//
// **والقيدُ في الجدول حارسٌ ثانٍ** — فلو أفلت قفلٌ لَما أفلت القيد.
func (s *Service) ReserveTx(ctx context.Context, q Querier, userID string, amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO wallets (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
		return err
	}
	var reserved int64
	err := q.QueryRow(ctx, `
		UPDATE wallets SET reserved = reserved + $2, updated_at = now()
		 WHERE user_id = $1
		RETURNING reserved`, userID, amount).Scan(&reserved)
	if isCheckViolation(err) {
		return ErrReservedTooMuch
	}
	return err
}

// ReleaseTx يفكّ حجزاً — **ولا يمسّ الرصيدَ ولا الدفتر.**
//
// **ويُفكّ مرّةً واحدة**: المنادي يقع داخلَ معاملةِ انتقالِ حالِ
// الطلب، **والحالةُ تُقرأ بقفلٍ قبله** — فلا يُفكّ حجزُ طلبٍ أُغلق.
func (s *Service) ReleaseTx(ctx context.Context, q Querier, userID string, amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	var reserved int64
	err := q.QueryRow(ctx, `
		UPDATE wallets SET reserved = reserved - $2, updated_at = now()
		 WHERE user_id = $1
		RETURNING reserved`, userID, amount).Scan(&reserved)
	if isCheckViolation(err) {
		// **فكٌّ يتجاوز المحجوز** — **وذاك فكٌّ ثانٍ لحجزٍ واحد.**
		return ErrReservedTooMuch
	}
	return err
}

// SettleReservedTx **الخصمُ وفكُّ الحجز فعلٌ واحد.**
//
// **ولا يقعان مفترقَين**: **خصمٌ بلا فكٍّ يُبقي مالاً محجوزاً لا
// وجودَ له**، **وفكٌّ بلا خصمٍ يُطلق مالاً خرج.**
//
// **والترتيبُ يُفكّ ثمّ يُخصَم** — **فالقيدُ `reserved <= balance`
// يرفض الخصمَ قبل الفكّ.**
func (s *Service) SettleReservedTx(ctx context.Context, q Querier, userID string,
	amount int64, kind, ref, note string, actorID *string) (int64, error) {
	if err := s.ReleaseTx(ctx, q, userID, amount); err != nil {
		return 0, err
	}
	return s.ApplyTx(ctx, q, userID, -amount, kind, ref, note, actorID)
}
