package obligations

// ══════════════════════════════════════════════════════════════════════
// **الدينُ أوّلاً — من كلّ مالٍ يدخل** (قرارُ المالك ٢٠٢٦-١٠-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// **كانت المكافأةُ والتعويضُ يُقيَّدان كاملَين ومن عليه دينٌ يبقى عليه** —
// فيسحب المالَ ويبقى الدينُ ينتظر أجراً قد لا يأتي. **وصار كلُّ مالٍ يدخل
// يسدّ الدينَ أوّلاً** (الأقدمُ أوّلاً)، **ويبقى له الباقي.**
//
// # والقيدُ كقيد الشحن
//
// **المكافأةُ تُقيَّد كاملةً** فيرى في كشفه ما استحقّه، **ثمّ سطرٌ يقتطع**
// (`adjustment` سالباً) **ويعود إلى الخزينة** (`platform_profit`) — كما يسدّ
// الشحنُ الدينَ (`payDebtFromTopup`). **ولا يُقتطَع بنوع المكافأة**: مجموعُ
// `reward` يبقى ما صرفته المنصّةُ فعلاً (`FI-03.a`)، والمستردُّ يُقرأ مستردّاً.
//
// **ومرجعُ القيدين معرّفُ المصدر لا رقمُ طلب** — قيدُ خزينةٍ بمرجع طلبٍ يُمحى
// عند إعادة حساب نصيب المنصّة (`creditTreasury` تجمع `platform_profit` بمرجعه).

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// CreditPay **كيف يُسدّ الدينُ من مالٍ دخل.**
type CreditPay struct {
	Wallet *wallet.Service
	// TreasuryID الخزينة — وفراغُها يقتطع ولا يقيّد مقابلاً (كالشحن بلا خزينة).
	TreasuryID string
	// Method طريقةُ التسوية — `MethodEarning` لِما يُخبَر به، و`MethodTopup` للشحن.
	Method string
	// From **من أين جاء المال** بكلمة الكشف: «مكافأة» · «تعويض» · «الشحن».
	From string
}

// PayFromCredit **يسدّ ديونَ صاحب المال من مالٍ دخل محفظتَه الآن** — داخل
// معاملة القيد.
//
// **والأطرافُ كلُّها**: دينُه سائقاً ومندوباً، **وديونُ متاجره** إن كان مالكاً.
// ويُقتطَع الأقلُّ من الدين والمالِ الداخل والمتاح. يُرجع ما اقتُطع.
func PayFromCredit(ctx context.Context, q Querier, userID string, credit int64,
	ref string, actorID *string, opt CreditPay) (int64, error) {
	if credit <= 0 || userID == "" || opt.Wallet == nil {
		return 0, nil
	}
	type party struct{ kind, id string }
	var parties []party
	rows, err := q.Query(ctx, `
		SELECT party_kind, party_id::text FROM financial_obligations
		 WHERE closed_at IS NULL
		   AND ((party_kind IN ('driver', 'rep') AND party_id = $1::uuid)
		     OR (party_kind = 'merchant' AND party_id IN (
		            SELECT id FROM merchants WHERE owner_user_id = $1::uuid)))
		 GROUP BY party_kind, party_id
		 ORDER BY min(created_at)`, userID)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var p party
		if err := rows.Scan(&p.kind, &p.id); err != nil {
			rows.Close()
			return 0, err
		}
		parties = append(parties, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	method := opt.Method
	if method == "" {
		method = MethodEarning
	}
	var paid int64
	budget := credit
	for _, p := range parties {
		if budget <= 0 {
			break
		}
		debt, err := Balance(ctx, q, p.kind, p.id)
		if err != nil {
			return 0, err
		}
		avail, err := opt.Wallet.AvailableTx(ctx, q, userID)
		if err != nil {
			return 0, err
		}
		take := min(debt, budget, avail)
		if take <= 0 {
			continue
		}
		note, err := OffsetNote(ctx, q, p.kind, p.id, take)
		if err != nil {
			return 0, err
		}
		if opt.From != "" {
			note = strings.Replace(note, "اقتطاع دين", "سداد دين من "+opt.From, 1)
		}
		_, txID, err := opt.Wallet.ApplyTxID(ctx, q, userID, -take, "adjustment", ref, note, actorID)
		if err != nil {
			return 0, err
		}
		if opt.TreasuryID != "" {
			if _, txID, err = opt.Wallet.ApplyTxID(ctx, q, opt.TreasuryID, take, "platform_profit",
				ref, note, actorID); err != nil {
				return 0, err
			}
		}
		applied, err := SettleBy(ctx, q, p.kind, p.id, take, "", txID, actorID, method, ref)
		if err != nil {
			return 0, err
		}
		if applied != take {
			return 0, errors.New("سدادُ الدين من مالٍ داخلٍ لم يطابق")
		}
		paid += take
		budget -= take
	}
	return paid, nil
}

// PartyUser **صاحبُ محفظة الطرف وتطبيقُه** — مالكُ المتجر في تطبيق المتجر،
// والمندوبُ والسائقُ نفساهما في تطبيقيهما.
func PartyUser(ctx context.Context, q Querier, kind, id string) (userID, app string, err error) {
	switch kind {
	case PartyRep:
		return id, notifications.AppRep, nil
	case PartyDriver:
		return id, notifications.AppDriver, nil
	}
	var owner string
	err = q.QueryRow(ctx,
		`SELECT COALESCE(owner_user_id::text, '') FROM merchants WHERE id = $1`, id).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil
	}
	return owner, notifications.AppMerchant, err
}

// noticeCreated «صار عليك دين X بسبب …، وبينقطع من أرباحك الجاية».
func noticeCreated(ctx context.Context, q Querier, kind, id string, amount int64, cause string) error {
	return notice(ctx, q, kind, id, "صار عليك دين",
		"صار عليك دين "+Money(amount)+" بسبب "+CauseWords(cause)+"، وبينقطع من أرباحك الجاية.")
}

// noticeDeducted «انقطع من دينك X، وبقي عليك Y».
func noticeDeducted(ctx context.Context, q Querier, kind, id string, taken int64) error {
	rest, err := Balance(ctx, q, kind, id)
	if err != nil {
		return err
	}
	body := "انقطع من دينك " + Money(taken) + "، وبقي عليك " + Money(rest) + "."
	if rest <= 0 {
		body = "انقطع من دينك " + Money(taken) + "، وانسدّ دينك كله."
	}
	return notice(ctx, q, kind, id, "انقطع من دينك", body)
}

func notice(ctx context.Context, q Querier, kind, id, title, body string) error {
	user, app, err := PartyUser(ctx, q, kind, id)
	if err != nil || user == "" {
		return err
	}
	return notifications.NotifyUserTx(ctx, q, notifications.Input{
		UserID: user, Kind: notifications.KindWallet, Title: title, Body: body,
		Entity: "wallet", Href: "/portal/wallet", Apps: []string{app},
	})
}

// Money **مبلغٌ بفواصل الآلاف ووحدةِ العملة** — «100000» ⟵ «100,000 ل.س».
func Money(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var b []byte
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			b = append(b, ',')
		}
		b = append(b, s[i])
	}
	out := string(b)
	if neg {
		out = "-" + out
	}
	return out + " ل.س"
}
