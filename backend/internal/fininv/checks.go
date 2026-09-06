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
		SQL: `
			SELECT u.full_name, w.balance, COALESCE(SUM(t.amount), 0)::bigint AS مجموع_القيود
			FROM wallets w
			JOIN users u ON u.id = w.user_id
			LEFT JOIN wallet_transactions t ON t.user_id = w.user_id
			GROUP BY u.full_name, w.balance
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
		Kinds: []string{"merchant_earning"},
		SQL: `
			SELECT o.number, o.total, o.status
			FROM orders o
			WHERE o.status = 'delivered' AND o.kind <> 'custom'
			  AND NOT EXISTS (
				SELECT 1 FROM wallet_transactions t
				WHERE t.ref = o.id::text AND t.kind = 'merchant_earning')`,
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
		Name:  "لا تعويضَ مكرَّرٌ لطلب",
		Why:   "طلبٌ عُوِّض أكثرَ من مرّة — وهو ما أُصلح ٢٠٢٦-٠٨-٠٨",
		Flows: []string{"F-16"},
		Kinds: []string{"compensation"},
		SQL: `
			SELECT t.ref, count(*)::bigint AS مرّات
			FROM wallet_transactions t
			WHERE t.kind = 'compensation' AND t.ref <> ''
			GROUP BY t.ref HAVING count(*) > 1`,
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
				SELECT sum(t.amount) AS net FROM wallet_transactions t
				WHERE t.ref = o.id::text
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
		Name:  "أجرُ السائقِ المقيَّد = أجرةُ الطلب",
		Why:   "أجرٌ يخالف الأجرةَ المحفوظةَ في الطلب — **والسائقُ قبض غيرَ ما اتُّفق.**",
		Flows: []string{"F-14"},
		Kinds: []string{"driver_earning"},
		// payDriver:1229 يقيّد أجرةَ الطلب — **وهي المحفوظةُ في العمود
		// لحظةَ الإنشاء، لا إعدادٌ يُقرأ حيّاً.** (وهذا ملقوطٌ فعلاً — FI-07.)
		SQL: `
			SELECT o.number, o.delivery_fee, sum(t.amount)::bigint AS المقيَّد
			FROM orders o
			JOIN wallet_transactions t
			  ON t.ref = o.id::text AND t.kind = 'driver_earning'
			WHERE o.kind <> 'custom'
			GROUP BY o.number, o.delivery_fee
			HAVING sum(t.amount) <> o.delivery_fee`,
	},

	// ═══════════════════════════════════════════════════════════════════
	// FI-10 — مطابقةُ نقدِ السائق
	// ═══════════════════════════════════════════════════════════════════

	{
		ID: "FI-10.a", Family: FI10, Status: ProvableNow, Ops: true,
		Name:  "محتجَزُ الصندوق = مجموعُ قيوده",
		Why:   "صندوقٌ لا يطابق قيودَه — فتسويةُ السائق تُبنى على رقمٍ خاطئ",
		Flows: []string{"F-27"},
		SQL: `
			SELECT u.full_name, b.held, COALESCE(SUM(e.amount), 0)::bigint AS مجموع_القيود
			FROM driver_cash_boxes b
			JOIN users u ON u.id = b.driver_id
			LEFT JOIN driver_cash_entries e ON e.driver_id = b.driver_id
			GROUP BY u.full_name, b.held
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
		Kinds:     []string{"platform_profit", "platform_expense", "operating_expense"},
		SQL: `
			SELECT t.id, t.kind, t.user_id::text, t.amount
			FROM wallet_transactions t
			JOIN wallets w ON w.user_id = t.user_id
			WHERE t.kind IN ('platform_profit','platform_expense','operating_expense')
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
	{
		ID: "FI-11.d", Family: FI11, Status: NotImplemented, Ops: false,
		Name: "طبقاتُ رصيدِ السحب — عقدُ شام كاش المستقبليّ",
		Why: "حالُ طلبِ السحب يقبل ثلاثاً فقط (pending · paid · rejected) — " +
			"**ولا AVAILABLE ولا RESERVED ولا PROCESSING ولا FAILED ولا " +
			"REVERSED.** **فالمعلَّقُ لا يُحجَز**: يبقى الرصيدُ صالحاً " +
			"للإنفاق حتّى القرار.",
		Flows:     []string{"F-24"},
		Registers: []string{"XG-12"},
		SQL:       `SELECT 1 WHERE false`,
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
