package fininv

// All كلُّ الثوابت — **مرتَّبةً بعوائلها.**
//
// # من أين جاءت الصيغ
//
// **لم تُخترَع صيغةٌ واحدة.** كلُّ علاقةٍ هنا مقروءةٌ من موضعها:
//
//	مستحقُّ المتجر   internal/orders/transitions.go:761  settleMerchant
//	أجرُ السائق      internal/orders/transitions.go:1229 payDriver
//	عمولةُ المندوب   internal/orders/transitions.go:936  settleRep
//	نصيبُ المنصّة    internal/orders/treasury.go:120     creditTreasury
//	العكسُ           internal/orders/transitions.go:1006 reverseCommissions
//	نقدُ السائق      internal/cashbox/cashbox.go:125     applyTx
//	السحب            internal/server/payout_handlers.go:183
//	المصروف          internal/server/expenses_handlers.go:224
//
// **وثلاثةَ عشرَ فحصاً منها كانت في `cmd/moneycheck`** — نُقلت بنصّها
// ومُنحت معرّفاتٍ ونُسبت إلى عوائلها. **ولم يُغيَّر استعلامٌ منها.**
var All = []Check{
	// ═══════════════════════════════════════════════════════════════════
	// FI-01 — سلامةُ مرجعِ الدفتر
	// ═══════════════════════════════════════════════════════════════════

	{
		ID: "FI-01.a", Family: FI01, Status: ProvableNow, Ops: true,
		Name: "لا قيدَ بصفر",
		Why:  "قيدٌ بمبلغِ صفرٍ — ضجيجٌ في الكشف بلا معنى",
		SQL:  `SELECT id, user_id, kind FROM wallet_transactions WHERE amount = 0`,
	},
	{
		ID: "FI-01.b", Family: FI01, Status: ProvableNow, Ops: true,
		Name: "كلُّ قيدٍ له محفظةٌ قائمة",
		Why: "قيدٌ لصاحبٍ بلا محفظةٍ لا يظهر في كشفٍ ولا في رصيد — " +
			"**مالٌ تحرّك ولا أحدَ يراه.**",
		SQL: `
			SELECT t.id, t.user_id::text, t.kind, t.amount
			FROM wallet_transactions t
			LEFT JOIN wallets w ON w.user_id = t.user_id
			WHERE w.user_id IS NULL`,
	},
	{
		ID: "FI-01.c", Family: FI01, Status: ProvableNow, Ops: true,
		Name:  "كلُّ نوعِ قيدٍ له عقد",
		Why:   "نوعٌ في القاعدة بلا عقدٍ في المنظومة — **مالٌ يتحرّك بدلالةٍ لا يعرفها أحد.**",
		Kinds: []string{"*"},
		SQL: `
			SELECT DISTINCT t.kind
			FROM wallet_transactions t
			WHERE t.kind <> ALL (ARRAY[` + contractedKindsSQL + `])`,
	},
	{
		ID: "FI-01.d", Family: FI01, Status: ProvableNow, Ops: true,
		Name: "قيدُ الطلبِ يشير إلى طلبٍ قائم",
		Why: "قيدٌ مرجعُه فارغٌ أو يشير إلى طلبٍ لا وجودَ له — " +
			"**فلا يُعرَف من أين جاء المال ولا كيف يُعكَس.**",
		Flows: []string{"F-14", "F-15"},
		Kinds: []string{"order_payment", "refund", "merchant_earning", "driver_earning", "commission"},
		SQL: `
			SELECT t.id, t.kind, t.ref, t.amount
			FROM wallet_transactions t
			WHERE t.kind IN ('order_payment','refund','merchant_earning','driver_earning','commission')
			  AND (t.ref = '' OR NOT EXISTS (
					SELECT 1 FROM orders o WHERE o.id::text = t.ref))`,
	},
	{
		ID: "FI-01.e", Family: FI01, Status: ProvableNow, Ops: true,
		Name:  "قيدُ السحبِ يشير إلى طلبِ سحبٍ قائم",
		Why:   "خصمُ سحبٍ بلا طلبٍ يقابله — **مالٌ خرج ولا ورقةَ تسنده.**",
		Flows: []string{"F-24"},
		Kinds: []string{"payout"},
		// **والمرجعُ الفارغُ مسموح** — الإدارةُ تسوّي بيدها بلا طلبٍ
		// (admin_wallet_handlers.go:64 تمرّر مرجعاً فارغاً)، **وذاك بابٌ
		// آخرُ له تدقيقُه.** ونُسأل هنا عمّا ادّعى مرجعاً فكذب.
		SQL: `
			SELECT t.id, t.ref, t.amount
			FROM wallet_transactions t
			WHERE t.kind = 'payout' AND t.ref <> ''
			  AND NOT EXISTS (SELECT 1 FROM payout_requests p WHERE p.id::text = t.ref)`,
	},
	{
		ID: "FI-01.g", Family: FI01, Status: ProvableNow, Ops: true,
		Name: "قيدُ إرجاعِ السحب يشير إلى سحبٍ مرتجعٍ بقيمته",
		Why: "مالٌ رجع إلى محفظةٍ باسم «إرجاع سحب» ولا سحبَ ارتدّ بهذا المبلغ — " +
			"**مالٌ دخل بلا ورقةٍ تسنده.**",
		Flows: []string{"F-24"},
		Kinds: []string{"payout_reversal"},
		SQL: `
			SELECT t.id, t.ref, t.amount
			FROM wallet_transactions t
			WHERE t.kind = 'payout_reversal'
			  AND NOT EXISTS (SELECT 1 FROM payout_requests p
			                   WHERE p.id::text = t.ref AND p.status = 'reversed'
			                     AND p.user_id = t.user_id AND p.amount = t.amount)`,
	},
	{
		ID: "FI-01.f", Family: FI01, Status: ProvableNow, Ops: true,
		Name:      "قيدُ المصروفِ يشير إلى مصروفٍ قائم",
		Why:       "خصمُ خزينةٍ بلا مصروفٍ يسنده — **خسارةٌ بلا سبب.**",
		Flows:     []string{"F-26"},
		Registers: []string{"D5"},
		Kinds:     []string{"operating_expense"},
		SQL: `
			SELECT t.id, t.ref, t.amount
			FROM wallet_transactions t
			WHERE t.kind = 'operating_expense'
			  AND NOT EXISTS (SELECT 1 FROM expenses e WHERE e.id::text = t.ref)`,
	},

	// ═══════════════════════════════════════════════════════════════════
	// FI-02 — مطابقةُ رصيدِ المحفظة
	// ═══════════════════════════════════════════════════════════════════

	{
		ID: "FI-02.a", Family: FI02, Status: ProvableNow, Ops: true,
		Name: "رصيدُ المحفظة = مجموعُ حركاتها",
		Why:  "رصيدٌ لا يطابق قيودَه — فكشفُ الحساب يقول غيرَ ما في الجيب",
		// **والصيغةُ مقيسةٌ لا مفترَضة** (wallet.go:251): لا رصيدَ افتتاحيٌّ
		// ولا مُرحَّل — الرصيدُ يبدأ صفراً ويُزاد بكلّ قيد، **فمجموعُ
		// القيود هو الرصيدُ نفسُه.**
		// **والطرفُ هو المحفظة، ومفتاحُها `user_id`** — `XG-43`:
		// **الاسمُ المعروضُ كان مفتاحَ التجميع**، فمتشابها الاسمِ يُدمجان
		// **فيُنذَر على سليمين، وأسوأُ منه أن يستر رصيدٌ بلا قيدٍ واحدٍ
		// خلفَ متشابهٍ سليم.** **والاسمُ يبقى وصفاً في الجواب لا حكماً.**
		SQL: `
			SELECT w.user_id::text, u.full_name, w.balance,
			       COALESCE(SUM(t.amount), 0)::bigint AS مجموع_القيود
			FROM wallets w
			JOIN users u ON u.id = w.user_id
			LEFT JOIN wallet_transactions t ON t.user_id = w.user_id
			GROUP BY w.user_id, u.full_name, w.balance
			HAVING w.balance <> COALESCE(SUM(t.amount), 0)`,
	},
	{
		ID: "FI-02.b", Family: FI02, Status: ProvableNow, Ops: true,
		Name: "لا رصيدَ سالبٌ إلّا الخزينة",
		Why:  "محفظةٌ بالسالب — ومالٌ صُرف ولم يكن موجوداً",
		SQL: `
			SELECT u.full_name, w.balance
			FROM wallets w JOIN users u ON u.id = w.user_id
			WHERE w.balance < 0 AND NOT w.is_treasury`,
	},
	{
		ID: "FI-02.c", Family: FI02, Status: ProvableNow, Ops: true,
		Name: "للمنصّة خزينةٌ واحدة",
		Why:  "بلا خزينةٍ يُتجاهَل مصروفُ المنصة بصمت — فتقريرُ الخسائر يبقى فارغاً أبداً",
		SQL: `
			SELECT 'عددُ الخزائن' AS الحالة, count(*)::bigint AS العدد
			FROM wallets WHERE is_treasury
			HAVING count(*) <> 1`,
	},

	// ═══════════════════════════════════════════════════════════════════
	// FI-03 — لا خلقَ للمال
	// ═══════════════════════════════════════════════════════════════════

	{
		ID: "FI-03.a", Family: FI03, Status: ProvableNow, Ops: true,
		Name: "المكافأةُ والعقوبةُ تخرجان من الخزينةِ وتعودان إليها",
		Why: "مكافأةٌ أُودعت ولم تُخصَم من خزينةٍ — **مالٌ خُلق من عدم.** " +
			"وعقوبةٌ حُصّلت ولم تدخلها — **مالٌ اختفى.**",
		Flows:     []string{"F-21", "F-23"},
		Kinds:     []string{"reward", "penalty"},
		Registers: []string{"XOB-9"},
		// **المرجعُ فارغٌ في هذين النوعين** (incentives.go:277 وtarget.go:237)،
		// **فلا يُطابَق قيدٌ بقيد** — والمجموعُ الكلّيُّ هو ما يُسأل عنه.
		// **وتُشترط خزينةٌ**: بلا خزينةٍ تُتخطّى المرآةُ بصمتٍ وليس ذلك خرقاً
		// هنا — **بل هو FI-02.c بعينه، ولا يُنذَر مرّتين على شيءٍ واحد.**
		SQL: `
			SELECT 'مكافآتٌ وعقوباتٌ غيرُ متوازنة' AS الحالة,
			       COALESCE(sum(t.amount), 0)::bigint AS الفرق
			FROM wallet_transactions t
			WHERE t.kind IN ('reward','penalty')
			  AND EXISTS (SELECT 1 FROM wallets WHERE is_treasury)
			HAVING COALESCE(sum(t.amount), 0) <> 0`,
	},
	{
		ID: "FI-03.b", Family: FI03, Status: ProvableNow, Ops: true,
		Name:  "مجاميعُ الطلب متّسقة",
		Why:   "طلبٌ مجموعُه لا يساوي أجزاءَه — فالفاتورةُ تكذب",
		Flows: []string{"F-01"},
		SQL: `
			SELECT number, subtotal, delivery_fee, discount, total
			FROM orders
			WHERE total <> subtotal + delivery_fee - discount`,
	},
	{
		ID: "FI-03.c", Family: FI03, Status: ProvableNow, Ops: true,
		Name:  "الدفعُ يغطّي المجموع",
		Why:   "طلبٌ مدفوعُه لا يساوي مجموعَه — فرقٌ ضائعٌ لا يعرف أحدٌ أين ذهب",
		Flows: []string{"F-01", "F-14"},
		SQL: `
			SELECT number, total, wallet_paid, cash_due
			FROM orders
			WHERE status = 'delivered' AND kind <> 'custom'
			  AND wallet_paid + cash_due <> total`,
	},

	// ═══════════════════════════════════════════════════════════════════
	// FI-04 — لا ضياعَ للمال
	// ═══════════════════════════════════════════════════════════════════

	{
		ID: "FI-04.a", Family: FI04, Status: ProvableNow, Ops: true,
		Name:  "كلُّ طلبٍ مسلَّمٍ له مستحقُّ متجر",
		Why:   "طلبٌ سُلّم ولم يُقيَّد مستحقُّ متجره — فالمتجرُ لم يُدفع له",
		Flows: []string{"F-12", "F-14"},
		Kinds: []string{"merchant_earning", "merchant_cash_accrued"},
		// **والمتجرُ المُسوّى نقداً مستحقُّه قيدُ احتباسٍ لا قيدُ محفظة**
		// (`merchant_cash_accrued`، تسويةُ ٢٠٢٦-٠٩-٢٧) — كان الفحصُ لا يعرفه، فكلُّ
		// طلبٍ نقديِّ التسوية سُلّم يُقرأ خرقاً. (كشفه قسمُ النقد ٢٠٢٦-١٠-٠٤.)
		SQL: `
			SELECT o.number, o.total, o.status
			FROM orders o
			WHERE o.status = 'delivered' AND o.kind <> 'custom'
			  AND NOT EXISTS (
				SELECT 1 FROM wallet_transactions t
				WHERE t.ref = o.id::text
				  AND t.kind IN ('merchant_earning', 'merchant_cash_accrued'))`,
	},
	{
		ID: "FI-04.b", Family: FI04, Status: ProvableNow, Ops: true,
		Name:  "كلُّ طلبٍ من المحفظة له خصمٌ مقابل",
		Why:   "طلبٌ دُفع من المحفظة ولم يُخصم — فالزبونُ أخذ بلا مقابل",
		Flows: []string{"F-01", "F-14"},
		Kinds: []string{"order_payment"},
		SQL: `
			SELECT o.number, o.wallet_paid
			FROM orders o
			WHERE o.wallet_paid > 0 AND o.status = 'delivered'
			  AND NOT EXISTS (
				SELECT 1 FROM wallet_transactions t
				WHERE t.ref = o.id::text AND t.kind = 'order_payment')`,
	},
	{
		ID: "FI-04.c", Family: FI04, Status: ProvableNow, Ops: true,
		Name:  "كلُّ سحبٍ مدفوعٍ له خصم",
		Why:   "سحبٌ حالتُه «مدفوع» ولا خصمَ له — فالمالُ خرج من الورق لا من المحفظة",
		Flows: []string{"F-24"},
		Kinds: []string{"payout"},
		SQL: `
			SELECT p.amount, u.full_name
			FROM payout_requests p JOIN users u ON u.id = p.user_id
			WHERE p.status = 'paid'
			  AND NOT EXISTS (
				SELECT 1 FROM wallet_transactions t
				WHERE t.ref = p.id::text AND t.kind = 'payout')`,
	},
	{
		ID: "FI-04.d", Family: FI04, Status: ProvableNow, Ops: true,
		Name: "كلُّ مصروفٍ قائمٍ له خصمُ خزينة",
		Why: "مصروفٌ سُجّل ولم يُخصَم من الخزينة — **فتقريرُ الأرباح يقول ربحاً " +
			"لم يقع.** وهذا وجهُ D5 الذي يُثبَت بلا حقنِ عطل.",
		Flows:     []string{"F-26"},
		Registers: []string{"D5"},
		Kinds:     []string{"operating_expense"},
		SQL: `
			SELECT e.id::text, e.amount, e.spent_at
			FROM expenses e
			WHERE e.voided_at IS NULL
			  AND EXISTS (SELECT 1 FROM wallets WHERE is_treasury)
			  AND NOT EXISTS (
				SELECT 1 FROM wallet_transactions t
				WHERE t.ref = e.id::text AND t.kind = 'operating_expense' AND t.amount < 0)`,
	},
	{
		ID: "FI-04.e", Family: FI04, Status: ProvableNow, Ops: true,
		Name: "المصروفُ المُلغى يتوازن إلى صفر",
		Why: "مصروفٌ أُلغي وبقي خصمُه — **خسارةٌ دائمةٌ لعمليّةٍ رُجع عنها.** " +
			"(handleVoidExpense يقيّد الردَّ بالنوع نفسِه والمرجع نفسِه.)",
		Flows:     []string{"F-26"},
		Registers: []string{"D5"},
		Kinds:     []string{"operating_expense"},
		SQL: `
			SELECT e.id::text, e.amount, COALESCE(sum(t.amount), 0)::bigint AS صافي_القيود
			FROM expenses e
			LEFT JOIN wallet_transactions t
			       ON t.ref = e.id::text AND t.kind = 'operating_expense'
			WHERE e.voided_at IS NOT NULL
			  AND EXISTS (SELECT 1 FROM wallets WHERE is_treasury)
			GROUP BY e.id, e.amount
			HAVING COALESCE(sum(t.amount), 0) <> 0`,
	},

	// ═══════════════════════════════════════════════════════════════════
	// FI-05 — أثرٌ ماليٌّ مرّةً واحدةً بالضبط
	// ═══════════════════════════════════════════════════════════════════

	{
		ID: "FI-05.a", Family: FI05, Status: ProvableNow, Ops: true,
		Name: "لا تعويضَ مكرَّرٌ لمستحقٍّ عن طلب",
		// **والوحدةُ (طلب، مستحقّ) لا (طلب) وحدَه**: طلبٌ حُظر عند المتجر يعود
		// إلى `accepted` ويُعاد توزيعُه (`merchant_blocked.go`)، **فسائقٌ ثانٍ
		// يقود مشوارَه ويُحظَر يستحقُّ تعويضَه** — سطران بمرجعٍ واحدٍ لسائقَين
		// حقٌّ لا خطأ. **والخطأُ أن يُعوَّض السائقُ نفسُه مرّتين** عن الطلب
		// نفسِه (يمنعه حارسُ الباب اليدويّ `driver_already_compensated` — وهو
		// بابُ التعويض الوحيدُ منذ ٢٠٢٦-١٠-٠٢)، وهو ما يمسكه هذا الثابت.
		Why:   "المستحقُّ عُوِّض عن الطلب نفسِه أكثرَ من مرّة — تكرارٌ ماليّ",
		Flows: []string{"F-16"},
		Kinds: []string{"compensation"},
		SQL: `
			SELECT t.ref, t.user_id, count(*)::bigint AS مرّات
			FROM wallet_transactions t
			WHERE t.kind = 'compensation' AND t.ref <> ''
			GROUP BY t.ref, t.user_id HAVING count(*) > 1`,
	},
	{
		ID: "FI-05.b", Family: FI05, Status: ProvableNow, Ops: true,
		Name:  "لا تعويضَ شكوى مكرَّر",
		Why:   "شكوى عُوِّضت أكثرَ من مرّة — وهو ما أُصلح ٢٠٢٦-٠٨-٠٧",
		Flows: []string{"F-31"},
		Kinds: []string{"compensation"},
		SQL: `
			SELECT tk.number, count(*)::bigint AS مرّات
			FROM tickets tk
			JOIN wallet_transactions t ON t.ref = tk.id::text AND t.kind = 'compensation'
			GROUP BY tk.number HAVING count(*) > 1`,
	},
	{
		ID: "FI-05.c", Family: FI05, Status: ProvableNow, Ops: true,
		Name:  "لا مستحقَّ متجرٍ مكرَّر",
		Why:   "طلبٌ قُيّد مستحقُّه مرّتين — فالمتجرُ قُبض له ضِعف",
		Flows: []string{"F-12"},
		Kinds: []string{"merchant_earning"},
		// **والسالبُ لا يُعَدّ** — العكسُ واقتطاعُ الدَّين يكتبان بالنوع نفسِه
		// والمرجع نفسِه (transitions.go:1089 و:923)، **والسؤالُ عن استحقاقٍ
		// قُيّد مرّتين لا عن قيدٍ ثانٍ يُنقص.**
		SQL: `
			SELECT t.ref, t.user_id::text, count(*)::bigint AS مرّات
			FROM wallet_transactions t
			WHERE t.kind = 'merchant_earning' AND t.amount > 0
			GROUP BY t.ref, t.user_id HAVING count(*) > 1`,
	},
	{
		ID: "FI-05.d", Family: FI05, Status: ProvableNow, Ops: true,
		Name:  "لا خصمَ سحبٍ مكرَّر",
		Why:   "طلبُ سحبٍ خُصم مرّتين — **والسائقُ دفع ثمنَ سحبٍ واحدٍ مرّتين.**",
		Flows: []string{"F-24"},
		Kinds: []string{"payout"},
		SQL: `
			SELECT t.ref, count(*)::bigint AS مرّات
			FROM wallet_transactions t
			WHERE t.kind = 'payout' AND t.ref <> ''
			GROUP BY t.ref HAVING count(*) > 1`,
	},
	{
		ID: "FI-05.k", Family: FI05, Status: ProvableNow, Ops: true,
		Name:  "لا إرجاعَ سحبٍ مكرَّر",
		Why:   "سحبٌ ارتدّ فرجع مالُه مرّتين — **وصاحبُه قبض ضعفَ ما خرج منه.**",
		Flows: []string{"F-24"},
		Kinds: []string{"payout_reversal"},
		SQL: `
			SELECT t.ref, count(*)::bigint AS مرّات
			FROM wallet_transactions t
			WHERE t.kind = 'payout_reversal'
			GROUP BY t.ref HAVING count(*) > 1`,
	},
	{
		ID: "FI-05.e", Family: FI05, Status: ProvableNow, Ops: true,
		Name:  "لا عمولةَ مندوبٍ مكرَّرةٌ لطلب",
		Why:   "طلبٌ دفع عمولتَه مرّتين — **والمنصّةُ خسرت ضعفَ ما قرّرت.**",
		Flows: []string{"F-23"},
		Kinds: []string{"commission"},
		SQL: `
			SELECT t.ref, count(*)::bigint AS مرّات
			FROM wallet_transactions t
			WHERE t.kind = 'commission' AND t.ref <> '' AND t.amount > 0
			GROUP BY t.ref HAVING count(*) > 1`,
	},
	{
		ID: "FI-05.f", Family: FI05, Status: ProvableNow, Ops: true,
		Name:  "لا أجرَ سائقٍ مكرَّرٌ لطلب",
		Why:   "طلبٌ دُفع أجرُه مرّتين — **والخزينةُ خسرت أجرةً بلا توصيل.**",
		Flows: []string{"F-14"},
		Kinds: []string{"driver_earning"},
		SQL: `
			SELECT t.ref, count(*)::bigint AS مرّات
			FROM wallet_transactions t
			WHERE t.kind = 'driver_earning' AND t.ref <> '' AND t.amount > 0
			GROUP BY t.ref HAVING count(*) > 1`,
	},
	{
		ID: "FI-05.g", Family: FI05, Status: ProvableNow, Ops: true,
		Name:  "لا استردادَ مكرَّرٌ لطلب",
		Why:   "طلبٌ رُدَّ ثمنُه مرّتين — **والزبونُ قبض ضعفَ ما دفع.**",
		Flows: []string{"F-15"},
		Kinds: []string{"refund"},
		SQL: `
			SELECT t.ref, count(*)::bigint AS مرّات
			FROM wallet_transactions t
			WHERE t.kind = 'refund' AND t.ref <> ''
			GROUP BY t.ref HAVING count(*) > 1`,
	},
	{
		ID: "FI-05.h", Family: FI05, Status: ProvableNow, Ops: true,
		Name: "لا مكافأةَ هدفٍ مكرَّرةٌ في مرحلةٍ واحدة",
		Why: "مكافأةُ مرحلةٍ صُرفت مرّتين — **والفهرسُ الفريد " +
			"(0089_target_reward.sql:35) يمنعها، وهذا يثبت أنّه يعمل.**",
		Flows: []string{"F-21"},
		Kinds: []string{"reward"},
		SQL: `
			SELECT user_id::text, period, count(*)::bigint AS مرّات
			FROM incentives
			WHERE for_target AND period IS NOT NULL
			GROUP BY user_id, period HAVING count(*) > 1`,
	},
	{
		ID: "FI-05.i", Family: FI05, Status: ProvableNow, Ops: true,
		Name:  "لا تحصيلَ نقدٍ مكرَّرٌ لطلب",
		Why:   "طلبٌ حُصّل نقدُه مرّتين — **وصندوقُ السائق يحمل ضِعفَ ما قبض.**",
		Flows: []string{"F-14", "F-27"},
		SQL: `
			SELECT e.ref, count(*)::bigint AS مرّات
			FROM driver_cash_entries e
			WHERE e.kind = 'order_collection' AND e.ref <> ''
			GROUP BY e.ref HAVING count(*) > 1`,
	},

	// ═══════════════════════════════════════════════════════════════════
	// FI-06 — حفظُ اقتصاد الطلب
	// ═══════════════════════════════════════════════════════════════════

	{
		ID: "FI-06.a", Family: FI06, Status: ProvableNow, Ops: true,
		Name: "صافي دفترِ الطلب = نقدُ السائق منه",
		Why: "**قانونُ الحفظ الأكبر.** كلُّ ما دخل الدفترَ باسم هذا الطلب " +
			"وخرج منه يجب أن يساوي بالضبط النقدَ الذي بيد السائق عنه — " +
			"**فما زاد مالٌ خُلق، وما نقص مالٌ ضاع.**",
		Flows: []string{"F-01", "F-12", "F-14", "F-15", "F-16"},
		// # الاشتقاق — **ولا صيغةَ مخترَعة**
		//
		// قيودُ طلبٍ مسلَّم: الزبونُ سالبُ المدفوع من المحفظة · المتجرُ
		// مستحقُّه · السائقُ أجرتُه · المندوبُ عمولتُه · والخزينةُ
		// المدفوعُ ناقص المسترجَع ناقص أنصبةِ الأطراف (treasury.go:146).
		// فالمجموع:
		//
		//	−wp + الأطراف + (wp + نقد − 0 − الأطراف) = النقد
		//
		// **والاستردادُ لا يكسرها**: creditTreasury تُعيد الحسابَ بعد
		// العكس فتقرأ ما بقي مقيَّداً، والمجموعُ يبقى النقدَ نفسَه.
		// **والطلبُ الخاصُّ**: خصمٌ ثمّ إيداعٌ بالمقدار نفسِه ⇒ صفرٌ،
		// ونقدُه صفر.
		//
		// **وتُشترط خزينة**: بلا خزينةٍ لا يُقيَّد نصيبُ المنصّة أصلاً
		// (treasuryOn تردّ فارغاً) — فالسؤالُ لا معنى له، **وهو FI-02.c.**
		SQL: `
			SELECT o.number, o.status,
			       COALESCE(l.net, 0)::bigint  AS صافي_الدفتر,
			       COALESCE(c.cash, 0)::bigint AS نقدُ_الصندوق
			FROM orders o
			LEFT JOIN LATERAL (
				-- **وقيدُ الدفع النقديّ تسويةُ التزامٍ لا توزيعُ طلب** — ينقل
				-- نقداً من الاحتباس إلى المتجر خارجَ معادلة «التوزيع = نقدُ
				-- السائق». يحرسه FI-14.f (المدفوعُ = المستحقُّ) وFI-14.a
				-- (الاحتباس = القائم)، فيُستثنى هنا كي لا يُقرأ خرقاً.
				SELECT sum(t.amount) AS net FROM wallet_transactions t
				WHERE t.ref = o.id::text AND t.kind <> 'merchant_cash_paid'
			) l ON true
			LEFT JOIN LATERAL (
				SELECT sum(e.amount) AS cash FROM driver_cash_entries e
				WHERE e.ref = o.id::text AND e.kind = 'order_collection'
			) c ON true
			WHERE EXISTS (SELECT 1 FROM wallets WHERE is_treasury)
			  AND EXISTS (SELECT 1 FROM wallet_transactions p
			              WHERE p.ref = o.id::text AND p.kind = 'platform_profit')
			  AND COALESCE(l.net, 0) <> COALESCE(c.cash, 0)`,
	},
	{
		ID: "FI-06.d", Family: FI06, Status: ProvableNow, Ops: true,
		Name: "الطلبُ قبل المحاسبة يحمل خصمَه وحدَه",
		Why: "**قرينةُ FI-06.a للمرحلة السابقة.** طلبٌ لم تُحاسِبه الخزينةُ " +
			"بعدُ يجب ألّا يحمل في الدفتر إلّا خصمَ محفظةِ الزبون — " +
			"**فأيُّ قيدٍ آخرَ هناك مالٌ تحرّك قبل أن يستحقّ أحدٌ شيئاً.**",
		Flows: []string{"F-01", "F-07"},
		// # لماذا مرحلتان لا ثابتٌ واحد
		//
		// **الخزينةُ لا تُحاسِب قبل الاستلام** (`settle` تناديها من
		// `StPickedUp` فصاعداً)، **فالمالُ المدفوعُ مسبقاً محبوسٌ ولم
		// يُنسَب بعد.** وصيغةُ `net = cash` تعطي عندئذٍ `−wallet_paid ≠ 0`
		// **وهو الصوابُ لا خرق.**
		//
		// **فقُسمت الحياةُ بموضع المحاسبة**: ما قبلَها له ثابتُه، وما بعدها
		// له ثابتُه. **وهذا أقوى من ثابتٍ واحدٍ يستثني نصفَ الطلبات.**
		SQL: `
			SELECT o.number, o.status, o.wallet_paid,
			       COALESCE(l.net, 0)::bigint AS صافي_الدفتر
			FROM orders o
			LEFT JOIN LATERAL (
				SELECT sum(t.amount) AS net FROM wallet_transactions t
				WHERE t.ref = o.id::text
			) l ON true
			WHERE NOT EXISTS (SELECT 1 FROM wallet_transactions p
			                  WHERE p.ref = o.id::text AND p.kind = 'platform_profit')
			  AND COALESCE(l.net, 0) <> -o.wallet_paid`,
	},
	{
		ID: "FI-06.b", Family: FI06, Status: ProvableNow, Ops: true,
		Name: "عمولةُ المنصّة = المقيَّدُ على الطلب",
		Why: "العمودُ platform_commission تقرؤه التقاريرُ ويحسب منه settleRep " +
			"— **ورقمٌ فيه لا يطابق ما قُيّد يجعل عمولةَ المندوب تُبنى على وهم.**",
		Flows: []string{"F-14", "F-23"},
		// **الصيغةُ من settleMerchant:822**: العمولةُ مجموعُ نسبةِ كلّ متجرٍ
		// من كلفته، **ثمّ تُكتب في العمود.** والمستحقُّ المقيَّدُ = الكلفةُ
		// ناقص العمولة، فالعمولةُ = الكلفةُ ناقص المستحقِّ الموجبِ المقيَّد.
		SQL: `
			SELECT o.number, o.platform_commission,
			       (COALESCE(k.cost, 0) - COALESCE(m.due, 0))::bigint AS المحسوبة
			FROM orders o
			LEFT JOIN LATERAL (
				SELECT sum(oi.merchant_price * oi.qty) AS cost
				FROM order_items oi WHERE oi.order_id = o.id
			) k ON true
			LEFT JOIN LATERAL (
				SELECT sum(t.amount) AS due FROM wallet_transactions t
				WHERE t.ref = o.id::text AND t.kind = 'merchant_earning' AND t.amount > 0
			) m ON true
			WHERE o.status = 'delivered' AND o.kind <> 'custom'
			  AND COALESCE(m.due, 0) > 0
			  AND o.platform_commission <> COALESCE(k.cost, 0) - COALESCE(m.due, 0)`,
	},
	{
		ID: "FI-06.c", Family: FI06, Status: ProvableNow, Ops: true,
		Name:  "أجرُ السائقِ المقيَّد = أجرُ السائقِ المعتمَد",
		Why:   "أجرٌ يخالف الأجرَ المحفوظَ في الطلب — **والسائقُ قبض غيرَ ما اتُّفق.**",
		Flows: []string{"F-14"},
		Kinds: []string{"driver_earning"},
		// payDriver يقيّد `driver_fee` — **أجرَ السائقِ المعتمَدَ المحفوظَ في
		// العمود لحظةَ الإنشاء، لا `delivery_fee`** (ما يدفعه الزبون؛ قد يصفّره
		// عرضُ «توصيلٍ مجّانيّ»). **فالطلبُ المجّانيُّ سائقُه يقبض أجرَه كاملاً
		// والخزينةُ تموّله** (٢٠٢٦-٠٩-٢٧) — والمقارنةُ إذن مع `driver_fee`.
		SQL: `
			SELECT o.number, o.driver_fee, sum(t.amount)::bigint AS المقيَّد
			FROM orders o
			JOIN wallet_transactions t
			  ON t.ref = o.id::text AND t.kind = 'driver_earning'
			WHERE o.kind <> 'custom'
			GROUP BY o.number, o.driver_fee
			HAVING sum(t.amount) <> o.driver_fee`,
	},

	// ═══════════════════════════════════════════════════════════════════
	// FI-10 — مطابقةُ نقدِ السائق
	// ═══════════════════════════════════════════════════════════════════

	{
		ID: "FI-10.a", Family: FI10, Status: ProvableNow, Ops: true,
		Name:  "محتجَزُ الصندوق = مجموعُ قيوده",
		Why:   "صندوقٌ لا يطابق قيودَه — فتسويةُ السائق تُبنى على رقمٍ خاطئ",
		Flows: []string{"F-27"},
		// **والطرفُ هو الصندوق، ومفتاحُه `driver_id`** — `XG-43`،
		// **كسابقه في `FI-02.a`.**
		SQL: `
			SELECT b.driver_id::text, u.full_name, b.held,
			       COALESCE(SUM(e.amount), 0)::bigint AS مجموع_القيود
			FROM driver_cash_boxes b
			JOIN users u ON u.id = b.driver_id
			LEFT JOIN driver_cash_entries e ON e.driver_id = b.driver_id
			GROUP BY b.driver_id, u.full_name, b.held
			HAVING b.held <> COALESCE(SUM(e.amount), 0)`,
	},
	{
		ID: "FI-10.b", Family: FI10, Status: ProvableNow, Ops: true,
		Name: "تحصيلُ النقدِ = نقدُ الطلب المستحقّ",
		Why: "قيدُ تحصيلٍ يخالف نقدَ الطلب — **والسائقُ يدين بغيرِ ما قبض.** " +
			"**ولا يُكتفى بالمحتجَز** (البند ١٠): الرصيدُ يُخفي قيدين خاطئين " +
			"يلغي أحدهما الآخر.",
		Flows: []string{"F-14", "F-27"},
		SQL: `
			SELECT o.number, o.cash_due, sum(e.amount)::bigint AS المحصَّل
			FROM orders o
			JOIN driver_cash_entries e
			  ON e.ref = o.id::text AND e.kind = 'order_collection'
			GROUP BY o.number, o.cash_due
			HAVING sum(e.amount) <> o.cash_due`,
	},
	{
		ID: "FI-10.c", Family: FI10, Status: ProvableNow, Ops: true,
		Name: "لا محتجَزَ سالبٌ في صندوق",
		Why: "صندوقٌ بالسالب — **تسويةٌ فاقت المحصَّل، والمنصّةُ تدين للسائق " +
			"من حيث تحسبه مديناً.**",
		Flows: []string{"F-27"},
		SQL:   `SELECT driver_id::text, held FROM driver_cash_boxes WHERE held < 0`,
	},

	// ═══════════════════════════════════════════════════════════════════
	// FI-11 — حفظُ السحب
	// ═══════════════════════════════════════════════════════════════════

	{
		ID: "FI-11.a", Family: FI11, Status: ProvableNow, Ops: true,
		Name: "السحبُ المرفوضُ لا يُخصَم",
		Why: "طلبُ سحبٍ رُفض وخُصم — **مالٌ أُخذ مقابلَ لا شيء.** " +
			"(handleDecidePayout:244 يقيّد الخصمَ للمدفوع وحدَه.)",
		Flows: []string{"F-24"},
		Kinds: []string{"payout"},
		SQL: `
			SELECT p.id::text, p.amount, p.status
			FROM payout_requests p
			WHERE p.status = 'rejected'
			  AND EXISTS (SELECT 1 FROM wallet_transactions t
			              WHERE t.ref = p.id::text AND t.kind = 'payout')`,
	},
	{
		ID: "FI-11.b", Family: FI11, Status: ProvableNow, Ops: true,
		Name:      "السحبُ المعلَّقُ لا يُخصَم",
		Why:       "خصمٌ قبل القرار — **مالٌ حُجز بلا حجزٍ معلَن** (XG-12: لا طبقاتِ رصيد).",
		Flows:     []string{"F-24"},
		Kinds:     []string{"payout"},
		Registers: []string{"XG-12"},
		SQL: `
			SELECT p.id::text, p.amount
			FROM payout_requests p
			WHERE p.status = 'pending'
			  AND EXISTS (SELECT 1 FROM wallet_transactions t
			              WHERE t.ref = p.id::text AND t.kind = 'payout')`,
	},
	{
		ID: "FI-11.c", Family: FI11, Status: ProvableNow, Ops: true,
		Name:  "مقدارُ الخصمِ = مقدارُ السحب",
		Why:   "سحبٌ خُصم بغيرِ مقداره — **فرقٌ بين الورقة والجيب.**",
		Flows: []string{"F-24"},
		Kinds: []string{"payout"},
		SQL: `
			SELECT p.id::text, p.amount, sum(t.amount)::bigint AS المخصوم
			FROM payout_requests p
			JOIN wallet_transactions t ON t.ref = p.id::text AND t.kind = 'payout'
			WHERE p.status = 'paid'
			GROUP BY p.id, p.amount
			HAVING sum(t.amount) <> -p.amount`,
	},

	// ═══════════════════════════════════════════════════════════════════
	// FI-12 — الخزينةُ والمصروف
	// ═══════════════════════════════════════════════════════════════════

	{
		ID: "FI-12.a", Family: FI12, Status: ProvableNow, Ops: true,
		Name: "لا قيدَ خزينةٍ لغيرِ الخزينة",
		Why: "أنواعُ المنصّة الثلاثةُ تُكتب للخزينة وحدَها (treasury.go " +
			"وexpenses_handlers.go) — **وقيدٌ منها في محفظةِ شخصٍ يعني " +
			"مسارَ كتابةٍ لا يعرفه أحد.**",
		Flows:     []string{"F-26"},
		Registers: []string{"XOB-9"},
		Kinds:     []string{"platform_profit", "platform_expense", "operating_expense", "treasury_withdrawal"},
		SQL: `
			SELECT t.id, t.kind, t.user_id::text, t.amount
			FROM wallet_transactions t
			JOIN wallets w ON w.user_id = t.user_id
			WHERE t.kind IN ('platform_profit','platform_expense','operating_expense','treasury_withdrawal')
			  AND NOT w.is_treasury`,
	},
	{
		ID: "FI-12.b", Family: FI12, Status: ProvableNow, Ops: true,
		Name: "مصروفُ التشغيلِ خصمٌ لا إيداع",
		Why: "قيدُ مصروفٍ موجبٌ لا يقابل إلغاءَ مصروف — " +
			"**إيداعٌ في الخزينة باسم مصروف.**",
		Flows:     []string{"F-26"},
		Registers: []string{"D5"},
		Kinds:     []string{"operating_expense"},
		SQL: `
			SELECT t.id, t.ref, t.amount
			FROM wallet_transactions t
			WHERE t.kind = 'operating_expense' AND t.amount > 0
			  AND NOT EXISTS (
				SELECT 1 FROM expenses e
				WHERE e.id::text = t.ref AND e.voided_at IS NOT NULL)`,
	},

	// **سحبُ الأدمن من رصيد الخزينة** (قرارُ المالك ٢٠٢٦-١٠-٠٤): خصمٌ من الخزينة
	// وحدَها، بمقدار ورقته، **وورقةٌ بلا قيدٍ أو قيدٌ بلا ورقة خرق.**
	{
		ID: "FI-12.d", Family: FI12, Status: ProvableNow, Ops: true,
		Name: "سحبُ الأدمن خصمٌ من الخزينة بمقدار ورقته",
		Why: "قيدُ سحبٍ في غير الخزينة أو موجبٌ أو بلا ورقةٍ أو بغير مقدارها — " +
			"**مالٌ خرج باسم مدير المنصّة ولا يطابق ما وقّعه.**",
		Kinds: []string{"treasury_withdrawal"},
		SQL: `
			SELECT t.id, t.user_id::text, t.amount, t.ref
			FROM wallet_transactions t
			JOIN wallets w ON w.user_id = t.user_id
			LEFT JOIN treasury_withdrawals tw ON tw.id::text = t.ref
			WHERE t.kind = 'treasury_withdrawal'
			  AND (NOT w.is_treasury OR t.amount >= 0 OR tw.id IS NULL OR tw.amount <> -t.amount)`,
	},
	{
		ID: "FI-12.e", Family: FI12, Status: ProvableNow, Ops: true,
		Name:  "كلُّ ورقةِ سحبٍ من الخزينة لها قيدٌ واحد",
		Why:   "ورقةُ سحبٍ بلا قيدٍ أو بقيدين — **الكشفُ يقول غيرَ ما وقع.**",
		Kinds: []string{"treasury_withdrawal"},
		SQL: `
			SELECT tw.id::text, tw.amount, count(t.id)::int AS القيود
			FROM treasury_withdrawals tw
			LEFT JOIN wallet_transactions t
			       ON t.kind = 'treasury_withdrawal' AND t.ref = tw.id::text
			GROUP BY tw.id, tw.amount
			HAVING count(t.id) <> 1`,
	},

	// ═══════════════════════════════════════════════════════════════════
	// FI-15 — صندوقُ المكتب (قرارُ المالك ٢٠٢٦-١٠-٠٤)
	// ═══════════════════════════════════════════════════════════════════
	//
	// **كلُّ نقدٍ دخل الدرجَ أو خرج منه له سطرٌ بمرجع قيده** — وإلّا لم يطابق
	// الإغلاقُ اليوميُّ ما في اليد. **والقديمُ قبل الهجرة `0300` لا يُحاسَب**:
	// لم يكن للصندوق وجودٌ يومَها.
	{
		ID: "FI-15.a", Family: FI15, Status: ProvableNow, Ops: true,
		Name: "نقدٌ سلّمه سائقٌ دخل صندوقَ المكتب",
		Why:  "تسليمٌ من صندوق السائق بلا سطرٍ داخلٍ بمقداره — **مالٌ خرج من يده ولم يصل الدرج.**",
		SQL: `
			SELECT e.id, e.driver_id::text, -e.amount AS المبلغ
			FROM driver_cash_entries e
			WHERE e.kind = 'settlement'
			  AND e.created_at >= ` + officeCashSince + `
			  AND NOT EXISTS (SELECT 1 FROM office_cash_entries c
			                  WHERE c.source = 'driver_settle' AND c.direction = 'in'
			                    AND c.ref = e.id::text AND c.amount = -e.amount)`,
	},
	{
		ID: "FI-15.b", Family: FI15, Status: ProvableNow, Ops: true,
		Name:  "مستحقٌّ نقديٌّ دُفع لمتجرٍ خرج من الصندوق",
		Why:   "تأكيدُ دفعٍ نقديٍّ لمتجرٍ بلا سطرٍ خارج — **الدرجُ يقول أكثرَ ممّا فيه.**",
		Kinds: []string{"merchant_cash_paid"},
		SQL: `
			SELECT ms.id::text, ms.paid_at
			FROM merchant_settlements ms
			WHERE ms.state = 'cash_paid'
			  AND ms.paid_at >= ` + officeCashSince + `
			  AND NOT EXISTS (SELECT 1 FROM office_cash_entries c
			                  WHERE c.source = 'merchant_cash_paid' AND c.direction = 'out'
			                    AND c.ref = ms.id::text)`,
	},
	{
		ID: "FI-15.c", Family: FI15, Status: ProvableNow, Ops: true,
		Name:  "سحبٌ صُرف نقداً خرج من الصندوق بمقداره",
		Why:   "سحبٌ مدفوعٌ بلا سطرٍ خارجٍ بمقداره — **الإغلاقُ اليوميُّ لا يطابق.**",
		Flows: []string{"F-24"},
		Kinds: []string{"payout"},
		SQL: `
			SELECT t.ref, -t.amount AS المبلغ
			FROM wallet_transactions t
			WHERE t.kind = 'payout' AND t.ref <> ''
			  AND t.created_at >= ` + officeCashSince + `
			  -- **والحوالةُ لا سطرَ لها في الدرج** — النقدُ وحدَه (paid_via).
			  AND EXISTS (SELECT 1 FROM payout_requests p
			              WHERE p.id::text = t.ref AND p.paid_via = 'cash')
			  AND NOT EXISTS (SELECT 1 FROM office_cash_entries c
			                  WHERE c.source = 'payout_paid' AND c.direction = 'out'
			                    AND c.ref = t.ref AND c.amount = -t.amount)`,
	},
	{
		ID: "FI-15.d", Family: FI15, Status: ProvableNow, Ops: true,
		Name:  "سحبُ الأدمن من الدرج خرج من الصندوق",
		Why:   "سحبٌ من الخزينة نقداً بلا سطرٍ خارج — **أو سطرٌ لسحبٍ لم يكن من الدرج.**",
		Kinds: []string{"treasury_withdrawal"},
		SQL: `
			SELECT tw.id::text, tw.amount, tw.from_cashbox
			FROM treasury_withdrawals tw
			LEFT JOIN office_cash_entries c
			       ON c.source = 'treasury_withdrawal' AND c.ref = tw.id::text
			WHERE (tw.from_cashbox AND (c.id IS NULL OR c.direction <> 'out' OR c.amount <> tw.amount))
			   OR (NOT tw.from_cashbox AND c.id IS NOT NULL)`,
	},
	{
		ID: "FI-15.e", Family: FI15, Status: ProvableNow, Ops: true,
		Name: "مصدرُ كلّ سطرٍ في الصندوق معروفٌ باتّجاهه",
		Why:  "سطرٌ بمصدرٍ لا يعرفه أحدٌ أو باتّجاهٍ يخالف معناه — **نقدٌ تحرّك بلا تفسير.**",
		SQL: `
			SELECT c.id::text, c.source, c.direction, c.amount
			FROM office_cash_entries c
			WHERE (c.source, c.direction) NOT IN (` + officeCashSourcesSQL + `)`,
	},
	{
		ID: "FI-15.f", Family: FI15, Status: ProvableNow, Ops: true,
		Name: "نقصُ الصندوق: الخسارةُ بقيدها والموجودُ بسطره",
		Why: "نقصٌ اعتُمد خسارةً بلا قيدٍ في الخزينة، أو وُجد بلا سطرٍ داخل، " +
			"أو قيدُ خسارةٍ لنقصٍ لم يُعتمَد — **الخسائرُ تقول غيرَ ما قُرّر.**",
		Kinds: []string{"platform_expense"},
		SQL: `
			SELECT s.id::text, s.status, s.amount
			FROM office_cash_shortfalls s
			WHERE (s.status = 'approved' AND NOT EXISTS (
			          SELECT 1 FROM wallet_transactions t
			           WHERE t.kind = 'platform_expense' AND t.ref = s.id::text AND t.amount = -s.amount))
			   OR (s.status = 'resolved' AND NOT EXISTS (
			          SELECT 1 FROM office_cash_entries c
			           WHERE c.source = 'shortfall_found' AND c.ref = s.id::text AND c.amount = s.amount))
			   OR (s.status IN ('pending', 'rejected') AND EXISTS (
			          SELECT 1 FROM wallet_transactions t
			           WHERE t.kind = 'platform_expense' AND t.ref = s.id::text))`,
	},
	{
		ID: "FI-15.g", Family: FI15, Status: ProvableNow, Ops: true,
		Name:  "شحنٌ نقديٌّ مُوافَقٌ عليه دخل الصندوق",
		Why:   "شحنُ محفظةٍ نقداً بلا سطرٍ داخل — **مالٌ في محفظةٍ لم يصل الدرج.**",
		Kinds: []string{"topup"},
		SQL: `
			SELECT wr.id::text, wr.amount
			FROM wallet_requests wr
			WHERE wr.kind = 'topup' AND wr.status = 'approved'
			  AND NOT EXISTS (SELECT 1 FROM office_cash_entries c
			                  WHERE c.source = 'wallet_topup' AND c.direction = 'in'
			                    AND c.ref = wr.id::text AND c.amount = wr.amount)`,
	},

	// ═══════════════════════════════════════════════════════════════════
	// FI-14 — تسويةُ المتجر نقداً: الاحتباسُ والهُويّةُ والعكس
	// ═══════════════════════════════════════════════════════════════════

	{
		ID: "FI-14.a", Family: FI14, Status: ProvableNow, Ops: true,
		Name: "رصيدُ الاحتباس = المستحقُّ النقديُّ القائم",
		Why: "**محفظةُ الاحتباس ليست مالاً حرّاً** — رصيدُها التزامٌ نقديٌّ لم " +
			"يُدفَع بعد. **فإن خالف مجموعَ (`amount − reversed`) للمستحقّات " +
			"القائمة فإمّا قيدٌ بلا صفٍّ أو صفٌّ بلا قيد** — والالتزامُ ضاع أو خُلق.",
		Kinds: []string{"merchant_cash_accrued", "merchant_cash_paid"},
		SQL: `
			SELECT w.user_id, w.balance,
			       COALESCE((SELECT sum(amount - reversed_amount)
			                 FROM merchant_settlements
			                 WHERE method = 'cash' AND state = 'cash_due'), 0) AS outstanding
			FROM wallets w
			WHERE w.is_cash_holding
			  AND w.balance <> COALESCE((SELECT sum(amount - reversed_amount)
			                             FROM merchant_settlements
			                             WHERE method = 'cash' AND state = 'cash_due'), 0)`,
	},
	{
		ID: "FI-14.b", Family: FI14, Status: ProvableNow, Ops: true,
		Name: "المحفظةُ والنقدُ لا يجتمعان لمصدرٍ واحد",
		Why: "**عقدُ XOR**: مستحقُّ (طلب، متجر) يُسوّى محفظةً أو نقداً لا كليهما. " +
			"**فتسويةٌ نقديّةٌ لها قيدُ مستحقٍّ محفظيّ، أو محفظيّةٌ لها قيدُ احتباسٍ " +
			"أو دفعٍ نقديّ، تعني مصدراً قُيّد بطريقتين.**",
		SQL: `
			SELECT id, method, state
			FROM merchant_settlements
			WHERE (method = 'cash'
			         AND (accrued_tx_id IS NULL OR earning_tx_id IS NOT NULL))
			   OR (method = 'wallet'
			         AND (earning_tx_id IS NULL
			              OR accrued_tx_id IS NOT NULL
			              OR paid_tx_id IS NOT NULL))`,
	},
	{
		ID: "FI-14.c", Family: FI14, Status: ProvableNow, Ops: true,
		Name: "المعكوسُ المخزَّن = مجموعُ سطورِ العكس",
		Why: "**`reversed_amount` صورةٌ محفوظةٌ لمجموعِ سطورِ العكس** (ردٌّ أو " +
			"اقتطاعُ دَين). **فافتراقُهما يعني معكوساً لا سطرَ له، أو سطراً لم " +
			"يُحدِّث القائم** — فيُدفَع ما استُرِدّ.",
		SQL: `
			SELECT ms.id
			FROM merchant_settlements ms
			WHERE ms.reversed_amount <> COALESCE(
			        (SELECT sum(r.amount) FROM merchant_settlement_reversals r
			          WHERE r.settlement_id = ms.id), 0)`,
	},
	{
		ID: "FI-14.d", Family: FI14, Status: ProvableNow, Ops: true,
		Name: "قيودُ الدفتر تطابق صفَّ التسوية النقديّة",
		Why: "**الصفُّ والدفترُ يقولان الشيءَ نفسَه**: قيدُ الاحتباس (+) = " +
			"`amount`، ومجموعُ قيودِ العكس (−) = `reversed_amount`. **وافتراقُهما " +
			"رقمٌ في الصفّ لا يقابله مالٌ في الدفتر** — وهو أصلُ كلِّ تسريب.",
		Kinds: []string{"merchant_cash_accrued"},
		SQL: `
			SELECT ms.id
			FROM merchant_settlements ms
			WHERE ms.method = 'cash'
			  AND (
			    ms.amount <> COALESCE((SELECT wt.amount FROM wallet_transactions wt
			                            WHERE wt.id = ms.accrued_tx_id), 0)
			 OR ms.reversed_amount <> COALESCE(
			        (SELECT -sum(wt.amount)
			           FROM merchant_settlement_reversals r
			           JOIN wallet_transactions wt ON wt.id = r.tx_id
			          WHERE r.settlement_id = ms.id), 0)
			  )`,
	},
	{
		ID: "FI-14.e", Family: FI14, Status: ProvableNow, Ops: true,
		Name: "الاحتباسُ لا يُسحَب ولا يُنفَق ولا يهبط تحت الصفر",
		Why: "**حسابٌ نظاميٌّ لا محفظةُ إنسان**: ليس خزينةً فلا يهبط تحت الصفر، " +
			"ولا يُحجَز منه، **ولا يُطلَب منه سحبٌ** — فطلبُ سحبٍ منه مسارٌ يعامله " +
			"كرصيدٍ حرّ.",
		SQL: `
			SELECT w.user_id
			FROM wallets w
			WHERE w.is_cash_holding AND (w.balance < 0 OR w.reserved <> 0)
			UNION ALL
			SELECT pr.user_id
			FROM payout_requests pr
			JOIN wallets w ON w.user_id = pr.user_id
			WHERE w.is_cash_holding`,
	},
	{
		ID: "FI-14.f", Family: FI14, Status: ProvableNow, Ops: true,
		Name: "النقدُ المدفوعُ = مجموعُ المستحقّات المسوّاة",
		Why: "**كلُّ دينارٍ خرج من الاحتباس دفعاً يقابله مستحقٌّ صار `cash_paid`** " +
			"بمقداره. **فافتراقُهما دفعٌ بلا تسويةٍ أو تسويةٌ بلا دفع** — نقدٌ خرج " +
			"لا يُعرَف لمن، أو مستحقٌّ يُدفَع مرّتين.",
		Kinds: []string{"merchant_cash_paid"},
		SQL: `
			SELECT paid.total AS paid_out, due.total AS settled_due
			FROM (SELECT COALESCE(-sum(amount), 0) AS total
			        FROM wallet_transactions WHERE kind = 'merchant_cash_paid') paid,
			     (SELECT COALESCE(sum(amount - reversed_amount), 0) AS total
			        FROM merchant_settlements
			       WHERE method = 'cash' AND state = 'cash_paid') due
			WHERE paid.total <> due.total`,
	},

	// ═══════════════════════════════════════════════════════════════════
	// **ما لا يُثبَت اليوم** — ويُعلَن ولا يُخفى
	// ═══════════════════════════════════════════════════════════════════

	{
		ID: "FI-09.a", Family: FI09, Status: ProvableNow, Ops: true,
		Name: "عمولةُ المندوبِ تُعكَس عند الاسترداد أو تصير التزاماً",
		Why: "**المنصّةُ ردّت للزبون ثمنَه** — **فلا يبقى لأحدٍ نصيبٌ منه.** " +
			"وعمولةُ المندوب تُعكَس من محفظته، **وما عجز عنه رصيدُه يصير " +
			"التزاماً يُقتطَع من عمولةٍ قادمة** (`RQ-5` · قرارُ المالك " +
			"٢٠٢٦-٠٩-٠٥). **وعمولةٌ تبقى بلا عكسٍ ولا التزامٍ مالٌ خُلق.**",
		Flows:     []string{"F-15", "F-23"},
		Registers: []string{"XG-10"},
		Kinds:     []string{"commission"},
		// ══════════════════════════════════════════════════════════
		// **والالتزامُ رصيدٌ جارٍ لا سطرٌ لكلّ طلب**
		// ══════════════════════════════════════════════════════════
		//
		// **فلا يُسأل الطلبُ وحدَه** — **يُسأل المندوب**: مجموعُ ما بقي
		// له من عمولاتٍ على طلباتٍ استُرِدّت **يجب ألّا يتجاوز التزامَه
		// القائم.** **وتجاوزُه يعني عمولةً لم تُعكَس ولم تُدَّن.**
		//
		// **ولا يُشترَط التساوي**: **التزامٌ اقتُطع من عمولةٍ قادمة
		// ينقص** — والباقي يبقى مغطّىً بما عُكس.
		SQL: `
			WITH بالمندوب AS (
				SELECT t.user_id,
				       sum(t.amount)::bigint AS الباقي
				FROM orders o
				JOIN wallet_transactions t
				  ON t.ref = o.id::text AND t.kind = 'commission'
				WHERE o.status = 'refunded'
				GROUP BY t.user_id
			)
			SELECT u.full_name, r.الباقي, u.commission_debt
			FROM بالمندوب r
			JOIN users u ON u.id = r.user_id
			WHERE r.الباقي > COALESCE(u.commission_debt, 0)`,
	},
	{
		ID: "FI-12.c", Family: FI12, Status: DeferredP6, Ops: false,
		Name: "المصروفُ وقيدُه يقعان معاً أو لا يقعان",
		Why: "handleCreateExpense:262 يُدخل صفَّ المصروف ثمّ يقيّد الخزينةَ " +
			"**في عمليّتين منفصلتين بلا معاملة** — **فسقوطُ الثانية يترك " +
			"مصروفاً بلا خصم.** والحالُ الناتجةُ يمسكها FI-04.d؛ " +
			"**وإثباتُ وقوعِها يحتاج إسقاطَ الخطوة الثانية عمداً.**",
		Flows:     []string{"F-26"},
		Registers: []string{"D5"},
		SQL:       `SELECT 1 WHERE false`,
	},
	{
		ID: "FI-05.j", Family: FI05, Status: DeferredP5, Ops: false,
		Name: "قرارا سحبٍ متزامنان لا يخصمان مرّتين",
		Why: "handleDecidePayout:221 يقفل بـFOR UPDATE ويشترط pending — " +
			"**والحالُ الناتجةُ يمسكها FI-05.d، وإثباتُ القفلِ نفسِه يحتاج " +
			"خيطين حقيقيّين.**",
		Flows:     []string{"F-24"},
		Registers: []string{"R10"},
		SQL:       `SELECT 1 WHERE false`,
	},
	// ══════════════════════════════════════════════════════════════
	// **طبقاتُ الرصيد** — `XG-12` · `AQ-3` (دورةُ إصلاحٍ ٣٠)
	// ══════════════════════════════════════════════════════════════
	//
	// **والمتاحُ مشتقٌّ لا مخزَّن** — فلا ثابتَ يحرس `available` بذاته،
	// **وإنّما يُحرَس طرفاه.**
	//
	// # ونطاقان لا نطاق
	//
	// **`AQ-3` عقدُ محافظِ الناس** — مندوبٌ وسائقٌ ومتجر: **رصيدٌ غيرُ
	// سالبٍ ومحجوزٌ منه ومتاحٌ غيرُ سالب.**
	//
	// **والخزينةُ حسابٌ محاسبيٌّ للمنصّة لا محفظةُ إنسان** — **تدفع
	// قبل أن تقبض بالقصد** (`0048_treasury.sql`)، **فتُستثنى من
	// «لا سالب» منذ يومها.**
	//
	// **ولا يُضعَّف عقدُ الناس لأنّ الخزينة تخالفه** — **بل يُفصَل
	// النطاقان ويُحرَس كلٌّ بما يخصّه**: `FI-11.d` لمحافظ الناس،
	// **و`FI-11.i` تُثبت أنّ الخزينة خارجَ عقد الحجز أصلاً.**
	{
		ID: "FI-11.d", Family: FI11, Status: ProvableNow, Ops: true,
		Name: "محافظُ الناس: المحجوزُ لا يتجاوز المُقيَّد ولا ينزل عن صفر",
		Why: "**المتاحُ = المُقيَّد − المحجوز** — **وخرقُ هذا يجعل المتاحَ " +
			"سالباً**، فيُنفَق مالٌ محجوزٌ أو يُحجَز مالٌ لا وجودَ له.",
		Flows:     []string{"F-24"},
		Registers: []string{"XG-12"},
		// **والخزينةُ مستثناةٌ من `balance >= 0` بعقدها** — فتُستثنى
		// هنا كما استُثنيت في القيد. **ولا حجزَ لها أصلاً**، ويحرسه
		// `FI-11.e`.
		SQL: `SELECT user_id::text, balance, reserved
		        FROM wallets
		       WHERE reserved < 0 OR (reserved > balance AND NOT is_treasury)`,
	},
	{
		ID: "FI-11.e", Family: FI11, Status: ProvableNow, Ops: true,
		Name: "محافظُ الناس: المحجوزُ يُصالِح طلباتِ السحب وحجوزَ الطلبات المخصَّصة",
		Why: "**`wallets.reserved` صورةٌ محفوظةٌ لا حقيقةٌ ثانية** — " +
			"**والحقيقةُ طلباتُ السحب `pending`/`processing` وحجوزُ الطلبات " +
			"المخصَّصةِ الحيّةِ من المحفظة** (Batch 2a). **ورقمٌ لا يُنسَب إلى " +
			"مصدرٍ قائمٍ مالٌ مجمَّدٌ بلا سبب.**",
		Flows:     []string{"F-24"},
		Registers: []string{"XG-12"},
		// **مصدران للحجز يُجمعان**: طلباتُ السحب المعلَّقة، وحجوزُ الطلبات
		// المخصَّصة (`custom_reserved_amount`). **وكلُّ حجزٍ منسوبٌ لطلبه**،
		// فيُصالَح المجموعُ صفّاً بصفّ.
		SQL: `SELECT w.user_id::text, w.reserved, COALESCE(a.total, 0) + COALESCE(c.total, 0)
		        FROM wallets w
		        LEFT JOIN (SELECT user_id, sum(amount) AS total
		                     FROM payout_requests
		                    WHERE status IN ('pending','processing')
		                    GROUP BY user_id) a ON a.user_id = w.user_id
		        LEFT JOIN (SELECT customer_id AS user_id, sum(custom_reserved_amount) AS total
		                     FROM orders
		                    WHERE kind = 'custom' AND custom_reserved_amount > 0
		                    GROUP BY customer_id) c ON c.user_id = w.user_id
		       WHERE NOT w.is_treasury
		         AND w.reserved <> COALESCE(a.total, 0) + COALESCE(c.total, 0)`,
	},
	{
		ID: "FI-11.f", Family: FI11, Status: ProvableNow, Ops: true,
		Name: "طلبٌ أُغلق لا يبقى له حجز",
		Why: "**`paid` خُصمت و`rejected`/`failed` لم يقع فيها صرف** — " +
			"**وكلُّها تفكّ الحجز.** **وحجزٌ يبقى لطلبٍ أُغلق يُجمّد مالَ " +
			"صاحبه إلى الأبد.**",
		Flows:     []string{"F-24"},
		Registers: []string{"XG-12"},
		SQL: `SELECT p.id::text, p.status, p.amount
		        FROM payout_requests p
		       WHERE p.status IN ('paid','rejected','failed','reversed')
		         AND EXISTS (SELECT 1 FROM wallets w
		                      WHERE w.user_id = p.user_id
		                        AND w.reserved > COALESCE((
		                              SELECT sum(a.amount) FROM payout_requests a
		                               WHERE a.user_id = p.user_id
		                                 AND a.status IN ('pending','processing')), 0))`,
	},
	{
		ID: "FI-11.g", Family: FI11, Status: ProvableNow, Ops: true,
		Name: "سحبٌ مدفوعٌ لا يُخصَم مرّتين",
		Why: "**قرارٌ يُكرَّر أو سباقُ موافقتين** — **وقيدُ `payout` " +
			"لطلبٍ واحدٍ يقع مرّةً.** (يحرسه القفلُ ومنعُ التكرار، " +
			"**والثابتُ يقيس النتيجة لا الآليّة.**)",
		Flows:     []string{"F-24"},
		Registers: []string{"XG-12", "R10"},
		SQL: `SELECT p.id::text, count(t.id)
		        FROM payout_requests p
		        JOIN wallet_transactions t
		          ON t.ref = p.id::text AND t.kind = 'payout'
		       WHERE p.status IN ('paid','reversed')
		       GROUP BY p.id
		      HAVING count(t.id) <> 1`,
	},
	{
		ID: "FI-11.h", Family: FI11, Status: ProvableNow, Ops: true,
		Name: "سحبٌ ارتدّ له قيدُ «إرجاع سحب» يُعيد المال",
		Why: "**والارتدادُ لا يمحو الخصمَ الأوّل** — **دفترٌ يُمحى منه " +
			"سطرٌ لا يُراجَع.** **فيُقيَّد ردٌّ بقيمته** بنوعه `payout_reversal` " +
			"(قرارُ المالك ٢٠٢٦-١٠-٠٤؛ كان `refund`).",
		Flows:     []string{"F-24"},
		Kinds:     []string{"payout_reversal"},
		Registers: []string{"XG-12"},
		SQL: `SELECT p.id::text, p.amount
		        FROM payout_requests p
		       WHERE p.status = 'reversed'
		         AND NOT EXISTS (SELECT 1 FROM wallet_transactions t
		                          WHERE t.ref = p.id::text AND t.kind = 'payout_reversal'
		                            AND t.amount = p.amount)`,
	},
	// ══════════════════════════════════════════════════════════════
	// **حجزُ الطلب المخصَّص — مملوكٌ للطلب، مصالَحٌ، لا يبقى لطلبٍ ميّت** — Batch 2a
	// ══════════════════════════════════════════════════════════════
	//
	// **`custom_reserved_amount` حجزٌ منسوبٌ لطلبه** — يدخل مصالحةَ `FI-11.e`
	// أعلاه، **وهذه الثوابتُ تحرس شكلَه**: غيرُ سالبٍ، للمخصَّص وحدَه، لا يبقى
	// لطلبٍ انتهى، **وموجبُه يقابل طلبَ محفظةٍ حيّاً أكّده صاحبُه بنسخته الحاليّة.**
	{
		ID: "FI-11.j", Family: FI11, Status: ProvableNow, Ops: true,
		Name: "حجزُ الطلب المخصَّص لا يكون سالباً",
		Why: "**فكٌّ يتجاوز الحجزَ يُنزله تحت الصفر** — **ومالٌ محجوزٌ سالبٌ " +
			"لا معنى له.** (يحرسه القيدُ `orders_custom_reserved_nonneg`، " +
			"**والثابتُ يقيس النتيجة.**)",
		Flows:     []string{"F-24"},
		Registers: []string{"XG-12"},
		SQL:       `SELECT id::text, custom_reserved_amount FROM orders WHERE custom_reserved_amount < 0`,
	},
	{
		ID: "FI-11.k", Family: FI11, Status: ProvableNow, Ops: true,
		Name: "طلبٌ غيرُ مخصَّصٍ لا يحمل حجزاً مخصَّصاً",
		Why: "**الحجزُ المخصَّصُ بابُه المخصَّصُ وحدَه** — **ورقمٌ فيه على طلبٍ " +
			"عاديٍّ يُجمَّد مالُ صاحبه بلا مصدر.** (يحرسه القيدُ " +
			"`orders_custom_reserved_only_custom`.)",
		Flows:     []string{"F-24"},
		Registers: []string{"XG-12"},
		SQL:       `SELECT id::text, kind, custom_reserved_amount FROM orders WHERE kind <> 'custom' AND custom_reserved_amount <> 0`,
	},
	{
		ID: "FI-11.l", Family: FI11, Status: ProvableNow, Ops: true,
		Name: "طلبٌ مخصَّصٌ انتهى لا يبقى له حجز",
		Why: "**التسليمُ يُسوّي الحجزَ فيُصفّره، والإلغاءُ/التعذّرُ/الرفضُ " +
			"يفكّه** — **وحجزٌ يبقى لطلبٍ انتهى يُجمّد مالَ صاحبه إلى الأبد.**",
		Flows:     []string{"F-24"},
		Registers: []string{"XG-12"},
		SQL: `SELECT id::text, status, custom_reserved_amount
		        FROM orders
		       WHERE kind = 'custom' AND custom_reserved_amount <> 0
		         AND status IN ('delivered','cancelled','rejected','failed')`,
	},
	{
		ID: "FI-11.m", Family: FI11, Status: ProvableNow, Ops: true,
		Name: "حجزٌ موجبٌ يقابل طلبَ محفظةٍ حيّاً مؤكَّداً بنسخته الحاليّة",
		Why: "**المالُ لا يُحجَز إلّا لطلبِ محفظةٍ حيٍّ أكّده صاحبُه**، " +
			"**والمحجوزُ = ما أكّده بالضبط** (`quote_confirmed_total`) " +
			"**بالنسخة الحاليّة** — **وحجزٌ لا يطابق تأكيداً حيّاً مالٌ مجمَّدٌ " +
			"بلا عقد.**",
		Flows:     []string{"F-24"},
		Registers: []string{"XG-12"},
		SQL: `SELECT id::text, status, payment_method, custom_reserved_amount,
		             quote_confirmed_total, quote_confirmed_version, quote_version
		        FROM orders
		       WHERE kind = 'custom' AND custom_reserved_amount > 0
		         AND ( payment_method <> 'wallet'
		            OR quote_confirmed_at IS NULL
		            OR custom_paid_at IS NOT NULL
		            OR quote_confirmed_version IS DISTINCT FROM quote_version
		            OR quote_confirmed_total IS DISTINCT FROM custom_reserved_amount
		            OR status IN ('delivered','cancelled','rejected','failed') )`,
	},
	// ══════════════════════════════════════════════════════════════
	// **لقطةُ اقتصادِ الطلب** — `XQ-2` · `XG-25`…`XG-28` (دورةُ ٣٢)
	// ══════════════════════════════════════════════════════════════
	{
		ID: "FI-07.b", Family: FI07, Status: ProvableNow, Ops: true,
		Name: "كلُّ طلبٍ قائمٍ له لقطةُ اقتصادٍ كاملة",
		Why: "**طلبٌ يُسوّى بلا لقطةٍ يُسوّى بإعدادات اليوم** — " +
			"**واقتصادٌ يُخترَع بعد شهرٍ ليس اقتصادَ الطلب** (`XQ-2`). " +
			"**والمنتهيةُ لا تحتاج لقطة**: اقتصادُها وقع وقُيّد في الدفتر.",
		Flows:     []string{"F-01", "F-02", "F-14"},
		Registers: []string{"XG-25", "XG-26", "XG-27", "XG-28"},
		SQL: `SELECT id::text, status
		        FROM orders
		       WHERE closed_at IS NULL
		         AND status NOT IN ('delivered','cancelled','rejected','failed','refunded')
		         AND (snap_merchant_commission_percent IS NULL
		           OR snap_rep_commission_percent IS NULL
		           OR snap_commission_source IS NULL
		           OR snap_activation_orders IS NULL)`,
	},
	{
		ID: "FI-07.c", Family: FI07, Status: ProvableNow, Ops: true,
		Name: "ولا نصفَ لقطة",
		Why: "**اللقطةُ كلٌّ أو لا شيء** — **ونصفُها يجعل نصفَ الاقتصاد " +
			"من عقدٍ ونصفَه من آخر.** (يحرسه قيدُ الجدول، **والثابتُ " +
			"يقيس النتيجةَ لا الآليّة.**)",
		Flows:     []string{"F-01", "F-02"},
		Registers: []string{"XG-25", "XG-26", "XG-27", "XG-28"},
		SQL: `SELECT id::text
		        FROM orders
		       WHERE num_nulls(snap_merchant_commission_percent, snap_rep_commission_percent,
		                       snap_commission_source, snap_activation_orders) NOT IN (0, 4)`,
	},
	{
		ID: "FI-11.i", Family: FI11, Status: ProvableNow, Ops: true,
		Name: "الخزينةُ خارجَ عقد الحجز — ولا محجوزَ لها",
		Why: "**الخزينةُ تُستثنى من «لا سالب»** — **ودلالةُ المتاح على " +
			"حسابٍ سالبٍ غيرُ معرَّفة.** **فالأمانُ أن تكون خارجَ العقد " +
			"لا أن يُخمَّن لها معنى.** " +
			"**وهو مُثبَتٌ بالبناء**: طلبُ السحب لأدوار الميدان وحدَها " +
			"(`sales` · `driver` · `merchant`)، **وعلَمُ الخزينة على " +
			"محفظة أدمن** — **ودورُ مكتبٍ لا يجتمع بدورِ ميدان** " +
			"(`ADG-2`). **وهذا الثابتُ يقيس النتيجةَ لا الآليّة.**",
		Flows:     []string{"F-24"},
		Registers: []string{"XG-12"},
		SQL: `SELECT user_id::text, balance, reserved
		        FROM wallets WHERE is_treasury AND reserved <> 0`,
	},
	{
		ID: "FI-07.a", Family: FI07, Status: NotImplemented, Ops: false,
		Name: "اقتصادُ الطلبِ لا يشيخ بتبديل الإعداد",
		Why: "settleRep وsettleMerchant تقرآن الإعدادَ **لحظةَ التسوية** " +
			"لا لحظةَ الإنشاء — **فطلبٌ أُنشئ بنسبةٍ يُسوّى بأخرى.** (XQ-2.)",
		Flows:     []string{"F-14", "F-23", "F-33"},
		Registers: []string{"XG-13"},
		SQL:       `SELECT 1 WHERE false`,
	},
	{
		ID: "FI-08.a", Family: FI08, Status: NotImplemented, Ops: false,
		Name: "حقُّ الزبونِ في الاسترداد غيرُ مشروطٍ برصيدِ غيره",
		Why: "**عقدٌ مقرَّرٌ لا شيفرةٌ قائمة.** اليومَ لا يُعكَس قيدُ المندوب " +
			"أصلاً (XG-10) **فلا يسقط الاستردادُ بسببه** — والنجاحُ عَرَضٌ " +
			"لا التزام. **ويوم يُنفَّذ العكسُ يصير XG-11 حيّاً.**",
		Flows:     []string{"F-15"},
		Registers: []string{"XG-10", "XG-11"},
		SQL:       `SELECT 1 WHERE false`,
	},
	{
		ID: "FI-04.f", Family: FI04, Status: NotImplemented, Ops: false,
		Name: "الدَّينُ المستحقُّ يُسجَّل حين يتعذّر العكس",
		Why: "**لا جدولَ دَينٍ لمندوب** — عمودُ الدَّين للمتاجر وحدَها. " +
			"**فحين لا يُعكَس قيدٌ لا يبقى أثرٌ للمطلوب.** (AQ-3.)",
		Flows:     []string{"F-15", "F-23"},
		Registers: []string{"XG-10", "XG-11"},
		SQL:       `SELECT 1 WHERE false`,
	},
	{
		ID: "FI-03.d", Family: FI03, Status: DeferredP6, Ops: false,
		Name: "مكافأةُ الهدفِ لا تسبق تثبيتَ التحويل",
		Why: "convertLead:811 ينادي grantSalesTargetIfAny **قبل** أن يُثبّت " +
			"حالَ المرشَّح في :813 — **والمكافأةُ تُودَع في معاملتها الخاصّة " +
			"وتُودَع فوراً.** فسقوطُ التثبيت يترك مكافأةً عن تحويلٍ لم يقع. " +
			"**والترتيبُ يُثبَت الآن؛ ووقوعُ الحالِ يحتاج إسقاطَ خطوة.**",
		Flows:     []string{"F-21"},
		Registers: []string{"D2"},
		SQL:       `SELECT 1 WHERE false`,
	},

	// ═══════════════════════════════════════════════════════════════════
	// FI-13 — **تتبّعُ الالتزامات** (`XG-31`)
	// ═══════════════════════════════════════════════════════════════════
	//
	// **الالتزامُ واقعةٌ لا رقم.** **والعمودان صورةٌ محفوظة** — فإن
	// فارقا الوقائعَ فأحدُهما يكذب، **ولا يُعرَف أيُّهما.**

	{
		ID: "FI-13.a", Family: FI13, Status: ProvableNow, Ops: true,
		Name: "دَينُ المتجرِ يطابق وقائعَه",
		Why: "**صورةٌ فارقت أصلَها** — ورقمٌ لا تفسّره وقائعُه لا يُراجَع " +
			"ولا يُنازَع فيه (XG-31).",
		Flows:     []string{"F-14", "F-17"},
		Registers: []string{"XG-31"},
		SQL: `
			SELECT m.id::text, m.name, m.debt AS الصورة,
			       COALESCE(o.الوقائع, 0) AS الوقائع
			FROM merchants m
			LEFT JOIN (SELECT party_id, sum(amount - settled)::bigint AS الوقائع
			             FROM financial_obligations
			            WHERE party_kind = 'merchant' AND closed_at IS NULL
			            GROUP BY party_id) o ON o.party_id = m.id
			WHERE m.debt <> COALESCE(o.الوقائع, 0)`,
	},
	{
		ID: "FI-13.b", Family: FI13, Status: ProvableNow, Ops: true,
		Name:      "التزامُ المندوبِ يطابق وقائعَه",
		Why:       "**كسابقه للمندوب** — والعمودُ `users.commission_debt` صورةٌ لا حقيقة.",
		Flows:     []string{"F-14"},
		Kinds:     []string{"commission"},
		Registers: []string{"XG-31", "XG-10"},
		SQL: `
			SELECT u.id::text, u.full_name, u.commission_debt AS الصورة,
			       COALESCE(o.الوقائع, 0) AS الوقائع
			FROM users u
			LEFT JOIN (SELECT party_id, sum(amount - settled)::bigint AS الوقائع
			             FROM financial_obligations
			            WHERE party_kind = 'rep' AND closed_at IS NULL
			            GROUP BY party_id) o ON o.party_id = u.id
			WHERE u.commission_debt <> COALESCE(o.الوقائع, 0)`,
	},
	{
		ID: "FI-13.c", Family: FI13, Status: ProvableNow, Ops: true,
		Name: "المسدَّدُ يساوي مجموعَ تسوياته",
		Why: "**عمودُ `settled` صورةٌ ثانية** — وإن فارق سطورَه فالتاريخُ " +
			"ناقصٌ أو مضاعَف.",
		Registers: []string{"XG-31"},
		SQL: `
			SELECT o.id::text, o.cause, o.settled AS الصورة,
			       COALESCE(s.المجموع, 0) AS السطور
			FROM financial_obligations o
			LEFT JOIN (SELECT obligation_id, sum(amount)::bigint AS المجموع
			             FROM obligation_settlements GROUP BY obligation_id) s
			       ON s.obligation_id = o.id
			WHERE o.settled <> COALESCE(s.المجموع, 0)`,
	},
	{
		ID: "FI-13.d", Family: FI13, Status: ProvableNow, Ops: true,
		Name: "كلُّ التزامٍ يُفسَّر بأصله",
		Why: "**التزامٌ بلا طلبٍ ولا سببِ تركة** — **وهو الرقمُ الذي لا " +
			"يُقال من أين** الذي وُجدت `XG-31` لأجله.",
		Registers: []string{"XG-31"},
		SQL: `
			SELECT id::text, party_kind, amount, cause
			FROM financial_obligations
			WHERE order_id IS NULL AND cause <> 'legacy_opening'`,
	},
}
