package orders

// حسمُ بضاعةِ طلبٍ تعذّر تسليمُه — **وجهتان لا ثالثةَ لهما.**
//
// # أين وقف الأمر
//
// المتجرُ يقبض عند خروج البضاعة من يده لا عند وصولها (وهو قرارٌ سابق: بضاعتُه
// خرجت فحقُّه ثبت). فإن تعذّر التسليمُ **بقي المالُ عنده والبضاعةُ في صندوق
// السائق** — والسائقُ يعود بها إلى المكتب.
//
// **ثمّ ماذا؟** سؤالان لا يُجيبهما النظام:
//
//	أقبِلَ المتجرُ استردادَها؟   ←  فيُسترجع ثمنُها
//	أم لم يقبل؟                ←  فتبقى له، والخسارةُ على المنصة
//
// **ويجيبهما من استلمها في المكتب** — لا من حملها ولا من طبخها.
//
// # الأولى · رُدّت إلى المتجر
//
//	يُسترجع منه ما قُبض        —  أخذ بضاعتَه فلا يأخذ ثمنَها معها
//	ويُدفع له دعمٌ بنسبة       —  `merchants.return_support_percent`
//	والخزينةُ تستردّ ما دفعت    —  بالفرق، لا بقيدٍ ثانٍ
//
// # الثانية · إلى المكتب
//
// **لا قيدَ أصلاً.** المالُ عند المتجر منذ الاستلام، والخزينةُ خصمته منذ
// حينها. **والخسارةُ مقيَّدةٌ قبل أن يُضغط الزرّ** — وما يفعله الزرُّ أن يقول
// «انتهى أمرُ هذه البضاعة»، فلا تبقى معلّقةً في السجلّ.
//
// # والمندوبُ لا شيءَ له في الحالين
//
// عمولتُه على طلبٍ وصل. **ولا يُقيَّد إلّا عند التسليم** (`settleRep`) — فلا
// يحتاج هنا نفياً، هو منفيٌّ بالبناء.
//
// # والسائقُ أخذ تعويضَه لحظةَ الفشل
//
// تلقائياً وبلا يد (`compensateDriverOnFail`) — **قبل أن يُسأل أحدٌ عن
// البضاعة**، لأنّ مشوارَه وقع سواءٌ استُرِدّت أم لا.

import (
	"context"
	"errors"
	"fmt"

	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// أخطاءُ الحسم — يقرؤها الموظّفُ في شاشته.
var (
	// ErrGoodsNotFailed لا حسمَ لبضاعةِ طلبٍ لم يفشل.
	ErrGoodsNotFailed = errors.New("الطلبُ ليس فاشلاً")
	// ErrGoodsAlreadySettled حُسمت مرّةً — **ومرّتان تسترجعان الثمنَ مرّتين.**
	ErrGoodsAlreadySettled = errors.New("بضاعةُ الطلب محسومةٌ سلفاً")
	// ErrGoodsLedgerMismatch الحسابُ لا يطابق الدفتر — **ولا يُسترجع بالتقدير.**
	ErrGoodsLedgerMismatch = errors.New("ما يُحسب لا يطابق ما قُيّد")
)

// وجهتا البضاعة.
const (
	GoodsToMerchant = "merchant"
	GoodsToOffice   = "platform"
)

// merchantShare نصيبُ متجرٍ واحدٍ من طلبٍ متعدّد المصادر.
type merchantShare struct {
	merchantID string
	ownerID    string
	earned     int64 // ما قُيّد له فعلاً عن هذا الطلب
}

// SettleGoods يحسم بضاعةَ طلبٍ فشل — **في معاملةٍ واحدة.**
//
// **ولا يُقبل نصفُه**: استرجاعٌ بلا تسويةِ خزينةٍ يجعل المنصةَ تبدو خاسرةً وقد
// استُرِدّ لها، **ودفترٌ نصفُه مكتوبٌ أسوأُ من دفترٍ لم يُكتب.**
func (s *Service) SettleGoods(ctx context.Context, orderID, to, actorID string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// **القفلُ داخل المعاملة**: ضغطتان متزامنتان تسترجعان الثمنَ مرّتين لولاه.
	var status string
	var settled *string
	if err := tx.QueryRow(ctx,
		`SELECT status, goods_settled_to FROM orders WHERE id = $1 FOR UPDATE`,
		orderID).Scan(&status, &settled); err != nil {
		return err
	}
	if status != StFailed {
		return ErrGoodsNotFailed
	}
	if settled != nil {
		return ErrGoodsAlreadySettled
	}

	if to == GoodsToMerchant {
		if err := s.clawBackGoods(ctx, tx, orderID, actorID); err != nil {
			return err
		}
		// **والخزينةُ تُعيد الحساب** — لا تُقيَّد بيدٍ ثانية.
		//
		// `creditTreasury` تقرأ ما قُيّد للأطراف من الدفتر وتضع الفرق، **فقيدٌ
		// سالبٌ للمتجر يرفع نصيبَها تلقائياً.** ولو كُتب لها قيدٌ مستقلٌّ لصارت
		// حسبتان بجانب بعضهما — **وحسبتان تفترقان يوماً.**
		if err := s.creditTreasury(ctx, tx, orderID, actorID); err != nil {
			return err
		}
	}

	returned := "NULL"
	if to == GoodsToMerchant {
		returned = "now()"
	}
	if _, err := tx.Exec(ctx,
		`UPDATE orders SET goods_settled_to = $2, returned_at = `+returned+
			` WHERE id = $1`, orderID, to); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	s.publishWalletsOf(ctx, orderID)
	s.pub.Publish("ops", map[string]any{"type": "order"})
	return nil
}

// clawBackGoods يسترجع ثمنَ ما رُدّ، ويدفع الدعمَ، ويقيّد ما عجزت عنه المحفظة.
func (s *Service) clawBackGoods(ctx context.Context, q wallet.Querier, orderID, actorID string) error {
	shares, err := s.goodsShares(ctx, q, orderID)
	if err != nil {
		return err
	}
	pct := int64(0)
	if s.settings != nil {
		pct = s.settings.GetInt(ctx, "merchants.return_support_percent")
	}

	for _, sh := range shares {
		if sh.earned <= 0 {
			continue
		}
		// **الدعمُ أوّلاً ثمّ الاسترجاع** — لا لترتيبٍ جماليّ: الدعمُ يرفع
		// رصيدَه، **فيقلّ ما يعجز عنه ويصغر الدَّين.** ولو عُكس لَقُيّد دَينٌ
		// أكبرُ ثمّ دُفع دعمٌ يبقى في محفظته بلا مقاصّة.
		if support := pct * sh.earned / 100; support > 0 {
			if _, err := s.wallet.ApplyTx(ctx, q, sh.ownerID, support,
				"compensation", orderID,
				"دعمُ المنصة عن بضاعةٍ رُدّت", &actorID); err != nil {
				return err
			}
			// **ويخرج من الخزينة في المعاملة نفسِها** — دعمٌ يُقيَّد للمتجر
			// وحدَه يجعل المنصةَ تظهر رابحةً وهي تدفع.
			if err := s.DebitTreasury(ctx, q, support, orderID,
				"دعمُ متجرٍ عن بضاعةٍ رُدّت", actorID); err != nil {
				return err
			}
		}

		// **ولا يُخصم إلّا ما تحتمله المحفظة**: قيدُ الصفر في القاعدة يرفض
		// السالبَ **كلَّه لا جزأه** — فيُقرأ الاسترجاعُ فشلاً ذريعاً، وتبقى
		// البضاعةُ عنده وثمنُها في جيبه.
		var balance int64
		if err := q.QueryRow(ctx,
			`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1`,
			sh.ownerID).Scan(&balance); err != nil {
			return err
		}
		take := sh.earned
		if balance < take {
			take = balance
		}
		if take > 0 {
			// **بالنوع نفسِه سالباً** — لا بنوعٍ جديد: الخزينةُ تجمع
			// `merchant_earning` لتعرف ما خرج للأطراف، **ونوعٌ لا تعرفه
			// يجعلها تحسب أنّ المالَ ما زال عنده.**
			if _, err := s.wallet.ApplyTx(ctx, q, sh.ownerID, -take,
				"merchant_earning", orderID,
				"استرجاعُ ثمنِ بضاعةٍ رُدّت إلى المتجر", &actorID); err != nil {
				return err
			}
		}
		if rest := sh.earned - take; rest > 0 {
			if _, err := q.Exec(ctx,
				`UPDATE merchants SET debt = debt + $2 WHERE id = $1`,
				sh.merchantID, rest); err != nil {
				return err
			}
		}
	}
	return nil
}

// goodsShares ما قُيّد لكلّ متجرٍ عن هذا الطلب — **من الدفتر لا من المعادلة.**
//
// # ولماذا من الدفتر
//
// المعادلةُ تقرأ العمولةَ **بقيمتها اليوم**، والقيدُ وقع بقيمتها **يومَها**.
// **فنسبةٌ تُغيَّر بين الاستلام والحسم تجعل المسترجَعَ غيرَ المدفوع** — يُسترجع
// أكثرُ ممّا أُعطي أو أقلّ، **ولا يظهر الفرقُ في أيّ خطأ.**
//
// **ويُطابَق بالمتجر**: الدفترُ يعرف صاحبَ المحفظة لا المتجر، **وصاحبٌ يملك
// متجرين لا يُقاصّ دَينُ أحدهما من الآخر.** فتُقرأ حصصُ الطلب من بنوده، ثمّ
// **تُوزن بما في الدفتر**: مجموعُهما يجب أن يتطابق، وإلّا رُفض الحسمُ كلُّه.
//
// **ورفضٌ صريحٌ خيرٌ من استرجاعٍ بالتقدير** — هذا مالُ الناس.
func (s *Service) goodsShares(ctx context.Context, q wallet.Querier, orderID string) ([]merchantShare, error) {
	// ما قُيّد فعلاً لكلّ صاحبِ محفظة.
	rows, err := q.Query(ctx, `
		SELECT user_id::text, sum(amount)
		FROM wallet_transactions
		WHERE ref = $1 AND kind = 'merchant_earning'
		GROUP BY user_id`, orderID)
	if err != nil {
		return nil, err
	}
	posted := map[string]int64{}
	var postedTotal int64
	for rows.Next() {
		var owner string
		var amount int64
		if err := rows.Scan(&owner, &amount); err != nil {
			rows.Close()
			return nil, err
		}
		posted[owner] = amount
		postedTotal += amount
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if postedTotal <= 0 {
		return nil, nil // لم يُقبض شيءٌ بعد — لا استرجاعَ له
	}

	// ومتاجرُ الطلب وأصحابُها — من بنوده، بلقطةِ البند لا بقائمة اليوم.
	mrows, err := q.Query(ctx, `
		SELECT COALESCE(oi.merchant_id, o.merchant_id)::text,
		       m.owner_user_id::text,
		       sum(oi.merchant_price * oi.qty)
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN merchants m ON m.id = COALESCE(oi.merchant_id, o.merchant_id)
		WHERE oi.order_id = $1 AND m.owner_user_id IS NOT NULL
		GROUP BY 1, 2`, orderID)
	if err != nil {
		return nil, err
	}
	type row struct {
		merchantID, ownerID string
		cost                int64
	}
	var list []row
	byOwner := map[string]int64{}
	for mrows.Next() {
		var r row
		if err := mrows.Scan(&r.merchantID, &r.ownerID, &r.cost); err != nil {
			mrows.Close()
			return nil, err
		}
		list = append(list, r)
		byOwner[r.ownerID] += r.cost
	}
	mrows.Close()
	if err := mrows.Err(); err != nil {
		return nil, err
	}
	// **وطلبٌ بلا بنودٍ يقع على متجره الأوّل** — طلباتُ ما قبل البنود.
	if len(list) == 0 {
		var merchantID, ownerID string
		if err := q.QueryRow(ctx, `
			SELECT m.id::text, m.owner_user_id::text
			FROM orders o JOIN merchants m ON m.id = o.merchant_id
			WHERE o.id = $1 AND m.owner_user_id IS NOT NULL`, orderID).
			Scan(&merchantID, &ownerID); err != nil {
			return nil, err
		}
		list = []row{{merchantID: merchantID, ownerID: ownerID, cost: 1}}
		byOwner[ownerID] = 1
	}

	// **وأصحابُ المحافظِ في الدفتر هم أصحابُ متاجر الطلب — لا زيادةَ ولا نقص.**
	if len(posted) != len(byOwner) {
		return nil, fmt.Errorf("%w: %d محفظةً في الدفتر و%d في بنود الطلب",
			ErrGoodsLedgerMismatch, len(posted), len(byOwner))
	}

	out := make([]merchantShare, 0, len(list))
	var given int64
	for i, r := range list {
		amount, ok := posted[r.ownerID]
		if !ok {
			return nil, fmt.Errorf("%w: متجرٌ في الطلب بلا قيدٍ في الدفتر", ErrGoodsLedgerMismatch)
		}
		// **وحصّةُ المتجر من قيدِ صاحبه بنسبة بضاعته** — وصاحبٌ بمتجرٍ واحد
		// يأخذ قيدَه كاملاً. **والأخيرُ يأخذ الباقي** فلا يضيع قرشٌ في القسمة.
		share := amount * r.cost / byOwner[r.ownerID]
		if i == len(list)-1 || r.ownerID != list[i+1].ownerID {
			share = amount - sharesOf(out, r.ownerID)
		}
		given += share
		out = append(out, merchantShare{merchantID: r.merchantID, ownerID: r.ownerID, earned: share})
	}
	if given != postedTotal {
		return nil, fmt.Errorf("%w: يُسترجع %d وقُيّد %d",
			ErrGoodsLedgerMismatch, given, postedTotal)
	}
	return out, nil
}

// sharesOf ما وُزّع حتى الآن على صاحبِ محفظةٍ بعينه.
func sharesOf(list []merchantShare, ownerID string) int64 {
	var sum int64
	for _, x := range list {
		if x.ownerID == ownerID {
			sum += x.earned
		}
	}
	return sum
}
