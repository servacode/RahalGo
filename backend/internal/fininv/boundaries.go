package fininv

// TxBoundary حدُّ معاملةٍ لعمليّةٍ ماليّةٍ واحدة — **مقيسٌ من الاستعمال.**
//
// **ولا يُخمَّن من الاسم** (البند ٢٣): `settle` يبدو معاملةً واحدةً وهو
// **لا يفتح واحدةً بل يرث معاملةَ مناديه**، **و`handleCreateExpense` يبدو
// عمليّةً واحدةً وهو كتابتان منفصلتان.**
//
// **والتصنيفُ ثلاثيّ:**
//
//	ATOMIC       يفتح معاملةً ويودعها — الكلُّ أو لا شيء
//	INHERITS_TX  يأخذ معاملةَ مناديه — ذرّيٌّ بقدر ذرّيّةِ من ناداه
//	NON_ATOMIC   كتابتان فأكثرُ بلا معاملة — **يقع نصفُه**
//
// **و`INHERITS_TX` هي `PARTIALLY_ATOMIC` في تقرير المالك**: ذرّيّتُها
// مشروطةٌ بغيرها، **فلا تُعَدّ ذرّيّةً بذاتها ولا غيرَ ذرّيّةٍ ظلماً.**
type TxBoundary struct {
	Op    string
	File  string
	Sig   string
	Class string
	Note  string
}

// TxBoundaries العمليّاتُ الماليّةُ وحدودُها — **وحارسٌ في `internal/qa`
// يعيد قياسَها من المصدر ويسقط إن فارقت.**
var TxBoundaries = []TxBoundary{
	{
		Op: "settle (التسوية)", File: "internal/orders/transitions.go",
		Sig: "func (s *Service) settle(", Class: "INHERITS_TX",
		Note: "كلُّ الأثر الماليّ للانتقال داخلَ معاملةِ المستدعي",
	},
	{
		Op: "settleMerchant", File: "internal/orders/transitions.go",
		Sig: "func (s *Service) settleMerchant(", Class: "INHERITS_TX",
		Note: "المستحقُّ واقتطاعُ الدَّين وتحديثُ العمولة معاً",
	},
	{
		Op: "settleRep", File: "internal/orders/transitions.go",
		Sig: "func (s *Service) settleRep(", Class: "INHERITS_TX",
	},
	{
		Op: "reverseCommissions", File: "internal/orders/transitions.go",
		Sig: "func (s *Service) reverseCommissions(", Class: "INHERITS_TX",
	},
	{
		Op: "payDriver", File: "internal/orders/transitions.go",
		Sig: "func (s *Service) payDriver(", Class: "INHERITS_TX",
	},
	{
		Op: "creditTreasury", File: "internal/orders/treasury.go",
		Sig: "func (s *Service) creditTreasury(", Class: "INHERITS_TX",
	},
	{
		Op: "cashbox.applyTx", File: "internal/cashbox/cashbox.go",
		Sig: "func (s *Service) applyTx(", Class: "INHERITS_TX",
		Note: "الرصيدُ والقيدُ معاً — **ولا يُكتب أحدُهما وحدَه**",
	},
	{
		Op: "cashbox.apply", File: "internal/cashbox/cashbox.go",
		Sig: "func (s *Service) apply(", Class: "ATOMIC",
	},
	{
		Op: "wallet.Apply", File: "internal/wallet/wallet.go",
		Sig: "func (s *Service) Apply(", Class: "ATOMIC",
	},
	{
		Op: "wallet.ApplyTx", File: "internal/wallet/wallet.go",
		Sig: "func (s *Service) ApplyTx(", Class: "INHERITS_TX",
	},
	{
		Op: "incentives.Grant", File: "internal/incentives/incentives.go",
		Sig: "func (s *Service) Grant(", Class: "ATOMIC",
		Note: "القيدُ والمرآةُ في الخزينة وصفُّ الحافز معاً",
	},
	{
		Op: "incentives.grantTarget", File: "internal/incentives/target.go",
		Sig: "func (s *Service) grantTarget(", Class: "ATOMIC",
		Note: "**ذرّيّةٌ في نفسِها ومنفصلةٌ عمّا استحقّها** — وهو D2",
	},
	{
		Op: "handleDecidePayout", File: "internal/server/payout_handlers.go",
		Sig: "func (s *Server) handleDecidePayout(", Class: "ATOMIC",
		Note: "قفلٌ FOR UPDATE وشرطُ pending — منعُ تكرارٍ ببناءِ المعاملة",
	},
	{
		Op: "handleCreateExpense", File: "internal/server/expenses_handlers.go",
		Sig: "func (s *Server) handleCreateExpense(", Class: "ATOMIC",
		Note: "**D5 · `PF-02` — أُصلح في دورةِ إصلاحٍ ٤**: صفُّ المصروف " +
			"وقيدُ الخزينة في معاملةٍ واحدة (`ApplyTx`). " +
			"**وكان مصروفٌ يبقى بلا خصمٍ فيقول تقريرُ الأرباح ربحاً لم يقع.**",
	},
	{
		Op: "handleVoidExpense", File: "internal/server/expenses_handlers.go",
		Sig: "func (s *Server) handleVoidExpense(", Class: "ATOMIC",
		Note: "**D5 · `PF-02` — أُصلح في دورةِ إصلاحٍ ٤**: وسمُ الإلغاء " +
			"وردُّ المال في معاملةٍ واحدة.",
	},
	{
		Op: "handleAdminWalletApply", File: "internal/server/admin_wallet_handlers.go",
		Sig: "func (s *Server) handleAdminWalletApply(", Class: "NON_ATOMIC",
		Note: "قيدٌ واحدٌ فقط — **ولا كتابةَ ثانيةً تُفقَد**",
	},
	{
		Op: "convertLead", File: "internal/server/leads_handlers.go",
		Sig: "func (s *Server) convertLead(", Class: "ATOMIC",
		Note: "**D2 · D25 · XG-18 · `PF-01` — أُصلح في دورةِ إصلاحٍ ٤**: " +
			"ستُّ الكتاباتِ في معاملةٍ واحدةٍ تعبر أربعَ طبقات " +
			"(`CreateMerchantTx` · `EnsureUserWithRoleTx` · " +
			"`GrantTargetIfReachedTx`). " +
			"**وكان سقوطُ التثبيت يترك متجراً ومكافأةً مدفوعةً ومرشَّحاً " +
			"`new`، والإعادةُ تُنشئ متجراً ثانياً.** " +
			"**والإشعاراتُ بعد التثبيت** — إشعارٌ خرج ثمّ ارتدّت المعاملةُ " +
			"كذبٌ لا يُسحَب.",
	},
}
