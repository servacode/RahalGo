package fininv

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// KindContract عقدُ نوعِ قيدٍ واحد — **ولا نوعَ بلا عقد.**
//
// ```
// NEW LEDGER KIND WITHOUT FINANCIAL CONTRACT = FAIL
// ```
//
// **والحارسُ يقرأ القيدَ من القاعدة الحيّة** (`pg_get_constraintdef`) لا من
// نصٍّ في هجرة — **فمن أضاف نوعاً بهجرةٍ جديدةٍ سقط بناؤه حتّى يملأ الحقولَ
// الأربعة**: من يكتبه · ما دلالتُه الماليّة · أيُّ ثابتٍ يغطّيه · أيَشترط
// مرجعاً.
type KindContract struct {
	// Kind اسمُه في القاعدة.
	Kind string
	// Creators مواضعُ الكتابة — **مقيسةٌ لا مذكورة.**
	Creators []string
	// Path البابُ الذي يبلغه فعلٌ خارجيّ — فارغٌ إن كان داخليّاً محضاً.
	Path string
	// Semantics دلالتُه الماليّة في جملة.
	Semantics string
	// Sign الاتّجاهُ المسموح: "+" أو "-" أو "±".
	Sign string
	// RefRequired أيشترط مرجعاً غيرَ فارغ.
	RefRequired bool
	// RefTarget إلى أيّ جدولٍ يشير مرجعُه.
	RefTarget string
	// Invariants الثوابتُ التي تحرسه.
	Invariants []string
	// Reachable أيستطيع كودُ الإنتاجِ اليومَ أن يكتبه.
	Reachable bool
}

// Kinds العقودُ — **أربعةَ عشرَ نوعاً، بعددِ ما يسمح به قيدُ القاعدة.**
var Kinds = map[string]KindContract{
	"topup": {
		Kind: "topup", Sign: "+", RefRequired: false,
		Creators:   []string{"internal/server/admin_wallet_handlers.go:64"},
		Path:       "POST /admin/users/{id}/wallet",
		Semantics:  "شحنٌ يدويٌّ لمحفظةٍ من الإدارة — **مالٌ يدخل المنظومةَ من خارجها.**",
		Invariants: []string{"FI-02.a"},
		Reachable:  true,
	},
	"order_payment": {
		Kind: "order_payment", Sign: "-", RefRequired: true, RefTarget: "orders",
		Creators:   []string{"internal/orders/service.go:521", "internal/orders/custom.go:278"},
		Path:       "POST /api/v1/orders",
		Semantics:  "خصمُ ثمنِ الطلب من محفظة الزبون لحظةَ الإنشاء.",
		Invariants: []string{"FI-01.d", "FI-04.b", "FI-06.a"},
		Reachable:  true,
	},
	"refund": {
		Kind: "refund", Sign: "+", RefRequired: true, RefTarget: "orders",
		Creators:   []string{"internal/orders/transitions.go:659", "internal/orders/transitions.go:739"},
		Path:       "انتقالُ حالةٍ إلى نهايةٍ غيرِ التسليم",
		Semantics:  "ردُّ ما دفعه الزبون — **من المحفظة قبل التسليم، وكاملُ المبلغ بعده.**",
		Invariants: []string{"FI-01.d", "FI-05.g", "FI-06.a"},
		Reachable:  true,
	},
	"compensation": {
		Kind: "compensation", Sign: "+", RefRequired: false, RefTarget: "orders|tickets",
		Creators: []string{
			"internal/orders/transitions.go:539", "internal/orders/goods.go:148",
			"internal/server/failure_aftermath.go:135", "internal/server/failure_aftermath.go:290",
			"internal/support/support.go:287",
		},
		Path:       "تعذّرُ التسليم تلقائيّاً · وحلُّ تذكرةٍ بتعويض",
		Semantics:  "تعويضُ طرفٍ عن ضررٍ لا ذنبَ له فيه — **يخرج من الخزينة.**",
		Invariants: []string{"FI-05.a", "FI-05.b", "FI-06.a"},
		Reachable:  true,
	},
	"commission": {
		Kind: "commission", Sign: "+", RefRequired: true, RefTarget: "orders",
		Creators:   []string{"internal/orders/transitions.go:988"},
		Path:       "تسويةُ التسليم",
		Semantics:  "نصيبُ المندوبِ من هامشِ طلبٍ سُلّم لمتجرٍ فعّله.",
		Invariants: []string{"FI-01.d", "FI-05.e", "FI-06.a", "FI-09.a"},
		Reachable:  true,
	},
	"merchant_earning": {
		Kind: "merchant_earning", Sign: "±", RefRequired: true, RefTarget: "orders",
		Creators: []string{
			"internal/orders/transitions.go:877", "internal/orders/transitions.go:923",
			"internal/orders/transitions.go:1089", "internal/orders/goods.go:178",
			"internal/server/driver_return.go:98",
		},
		Path:      "الاستلامُ من المتجر · والاستردادُ · وردُّ البضاعة",
		Semantics: "مستحقُّ المتجرِ عن بضاعةٍ خرجت من يده — **والسالبُ عكسٌ أو اقتطاعُ دَين.**",
		// **والاتّجاهان مقصودان**: القيدُ نفسُه يحمل الاستحقاقَ وعكسَه
		// **بالمرجع نفسِه**، **وهذا ما يجعل الاستردادَ مانعاً لنفسِه
		// من التكرار**: مجموعُ الصفوف يصير صفراً فلا يُعكَس ثانيةً.
		Invariants: []string{"FI-01.d", "FI-04.a", "FI-05.c", "FI-06.b"},
		Reachable:  true,
	},
	"driver_earning": {
		Kind: "driver_earning", Sign: "+", RefRequired: true, RefTarget: "orders",
		Creators:   []string{"internal/orders/transitions.go:1236", "internal/orders/custom.go:282"},
		Path:       "تسويةُ التسليم",
		Semantics:  "أجرُ توصيلِ طلبٍ سُلّم — **وهو أجرةُ الطلبِ المحفوظةُ لا إعدادٌ حيّ.**",
		Invariants: []string{"FI-01.d", "FI-05.f", "FI-06.c"},
		Reachable:  true,
	},
	"payout": {
		Kind: "payout", Sign: "-", RefRequired: false, RefTarget: "payout_requests",
		Creators:   []string{"internal/server/payout_handlers.go:290", "internal/server/admin_wallet_handlers.go:64"},
		Path:       "PATCH /admin/payouts/{id} · وPOST /admin/users/{id}/wallet",
		Semantics:  "خروجُ مالٍ من محفظةٍ إلى صاحبها خارجَ المنظومة.",
		Invariants: []string{"FI-01.e", "FI-04.c", "FI-05.d", "FI-11.a", "FI-11.b", "FI-11.c"},
		Reachable:  true,
	},
	"adjustment": {
		Kind: "adjustment", Sign: "±", RefRequired: false,
		Creators:   []string{"internal/server/admin_wallet_handlers.go:64", "internal/server/disputes.go:280"},
		Path:       "POST /admin/users/{id}/wallet · ومطالبةُ المنصّة في نزاع",
		Semantics:  "تسويةٌ يدويّةٌ بقرارِ إداريّ — **بلا سقفٍ ولا مصادقةٍ ثانية.**",
		Invariants: []string{"FI-02.a", "FI-02.b"},
		Reachable:  true,
	},
	"platform_profit": {
		Kind: "platform_profit", Sign: "±", RefRequired: false, RefTarget: "orders|disputes",
		Creators:   []string{"internal/orders/treasury.go:166", "internal/orders/treasury.go:215"},
		Path:       "كلُّ انتقالٍ ماليٍّ للطلب — creditTreasury",
		Semantics:  "ما بقي للمنصّة بعد كلّ الأنصبة — **يُعاد حسابُه لا يُضاف تراكماً.**",
		Invariants: []string{"FI-06.a", "FI-12.a"},
		Reachable:  true,
	},
	"platform_expense": {
		Kind: "platform_expense", Sign: "-", RefRequired: false, RefTarget: "orders",
		Creators:   []string{"internal/orders/treasury.go:184"},
		Path:       "تعويضُ سائقٍ · ودعمُ متجرٍ عن بضاعةٍ رُدّت",
		Semantics:  "مالٌ خرج من الخزينة خارجَ تسويةِ الطلب.",
		Invariants: []string{"FI-06.a", "FI-12.a"},
		Reachable:  true,
	},
	"operating_expense": {
		Kind: "operating_expense", Sign: "±", RefRequired: true, RefTarget: "expenses",
		Creators:   []string{"internal/server/expenses_handlers.go:281", "internal/server/expenses_handlers.go:313"},
		Path:       "POST /admin/expenses · وDELETE /admin/expenses/{id}",
		Semantics:  "مصروفُ تشغيلٍ من الخزينة — **والموجبُ إلغاؤه لا غير.**",
		Invariants: []string{"FI-01.f", "FI-04.d", "FI-04.e", "FI-12.a", "FI-12.b"},
		Reachable:  true,
	},
	"reward": {
		Kind: "reward", Sign: "±", RefRequired: false, RefTarget: "orders|users|—",
		Creators: []string{
			"internal/incentives/incentives.go:277", "internal/incentives/target.go:237",
			"internal/referrals/referrals.go:314", "internal/referrals/referrals.go:443",
		},
		Path:      "POST /admin/users/{id}/incentive · وبلوغُ هدفٍ · ودعوةٌ ومنحةُ حسابٍ جديد",
		Semantics: "مكافأةٌ تخرج من الخزينة إلى مستحقّها — **والسالبُ وجهُها في الخزينة.**",
		// **ومرجعُه يختلف باختلاف مصدره**: رقمُ الطلب في الدعوة، ومعرّفُ
		// الزبون في منحة الحساب، **وفارغٌ في الحافز وهدفِ المرحلة** —
		// **فلا يُشترط.** (ملاحظةٌ لا حكم: نوعٌ واحدٌ بثلاث دلالاتِ مرجع.)
		Invariants: []string{"FI-03.a", "FI-05.h"},
		Reachable:  true,
	},
	"penalty": {
		Kind: "penalty", Sign: "±", RefRequired: false,
		Creators:   []string{"internal/incentives/incentives.go:277", "internal/incentives/incentives.go:286"},
		Path:       "POST /admin/users/{id}/incentive",
		Semantics:  "عقوبةٌ تُخصم من مخالفٍ وتدخل الخزينة — **وتُرفض إن لم يكفِ رصيدُه.**",
		Invariants: []string{"FI-03.a", "FI-02.b"},
		Reachable:  true,
	},
}

// contractedKindsSQL قائمةُ الأنواع المتعاقَدِ عليها نصّاً لاستعمالها في SQL.
//
// **وتُولَّد من الخريطة لا تُكتب بيدٍ ثانية** — فمن أضاف عقداً لم ينسَ
// تحديثَ الاستعلام.
var contractedKindsSQL = func() string {
	var ks []string
	for k := range Kinds {
		ks = append(ks, "'"+k+"'")
	}
	sort.Strings(ks)
	return strings.Join(ks, ",")
}()

// ContractedKinds الأنواعُ المتعاقَدُ عليها مرتَّبةً.
func ContractedKinds() []string {
	var ks []string
	for k := range Kinds {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// SchemaKinds يقرأ الأنواعَ التي تسمح بها القاعدةُ **الحيّة**.
//
// **ولا يُقرأ نصُّ هجرةٍ** — الهجراتُ تُسقط القيدَ وتعيد بناءه سبعَ مرّات،
// **ومن قرأ واحدةً منها قرأ تاريخاً لا حاضراً.** (**وهذا بعينه ما وقع في
// مستخرِج `P-2`**: كان يبحث عن `kind IN (` والهجرتان الأخيرتان تكتبان
// `kind = ANY (ARRAY[` — **فقرأ قيداً شائخاً ينقصه نوع.**)
func SchemaKinds(ctx context.Context, q Querier) ([]string, error) {
	var def string
	err := q.QueryRow(ctx, `
		SELECT pg_get_constraintdef(c.oid)
		FROM pg_constraint c
		JOIN pg_class t ON t.oid = c.conrelid
		WHERE t.relname = 'wallet_transactions'
		  AND c.conname = 'wallet_transactions_kind_check'`).Scan(&def)
	if err != nil {
		return nil, fmt.Errorf("قيدُ أنواع المحفظة غيرُ مقروء: %w", err)
	}
	var out []string
	for _, part := range strings.Split(def, "'") {
		if part != "" && !strings.ContainsAny(part, "(),: =[]") {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("قيدُ الأنواع لم يُفهَم: %s", def)
	}
	sort.Strings(out)
	return out, nil
}

// KindDrift الفرقُ بين ما تسمح به القاعدةُ وما تعاقدت عليه المنظومة.
//
// يعيد الناقصَ عقداً والزائدَ عقداً — **وأيُّهما غيرُ فارغٍ يعني سقوطاً.**
func KindDrift(schema []string) (missingContract, staleContract []string) {
	have := map[string]bool{}
	for _, k := range schema {
		have[k] = true
	}
	for _, k := range schema {
		if _, ok := Kinds[k]; !ok {
			missingContract = append(missingContract, k)
		}
	}
	for k := range Kinds {
		if !have[k] {
			staleContract = append(staleContract, k)
		}
	}
	sort.Strings(missingContract)
	sort.Strings(staleContract)
	return
}
