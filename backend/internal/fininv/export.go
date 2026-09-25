package fininv

import "sort"

// Export صورةُ المحرّك القابلةُ للقراءة آليّاً — **لا Markdown مصدراً وحيداً.**
//
// **و`P-10` سيحتاجها**: محرّكُ أثر التغيير يسأل «أيُّ ثابتٍ ماليٍّ يمسّه
// هذا الملفّ؟» — **وسؤالٌ كهذا لا يُجاب من نصٍّ عربيٍّ منسَّق.**
type Export struct {
	Families   []FamilyOut `json:"families"`
	Checks     []CheckOut  `json:"checks"`
	Kinds      []KindOut   `json:"kinds"`
	Counts     Counts      `json:"counts"`
	SettingIDs []string    `json:"financial_settings"`
}

type FamilyOut struct {
	ID     string   `json:"id"`
	Checks []string `json:"checks"`
}

type CheckOut struct {
	ID        string   `json:"id"`
	Family    string   `json:"family"`
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Ops       bool     `json:"ops"`
	Flows     []string `json:"flows,omitempty"`
	Registers []string `json:"registers,omitempty"`
	Kinds     []string `json:"kinds,omitempty"`
}

type KindOut struct {
	Kind        string   `json:"kind"`
	Semantics   string   `json:"semantics"`
	Sign        string   `json:"sign"`
	RefRequired bool     `json:"ref_required"`
	RefTarget   string   `json:"ref_target,omitempty"`
	Path        string   `json:"path,omitempty"`
	Creators    []string `json:"creators"`
	Invariants  []string `json:"invariants"`
	Reachable   bool     `json:"reachable"`
}

type Counts struct {
	Families       int `json:"families"`
	Checks         int `json:"checks"`
	ProvableNow    int `json:"provable_now"`
	NotImplemented int `json:"not_implemented"`
	DeferredP5     int `json:"deferred_p5"`
	DeferredP6     int `json:"deferred_p6"`
	Kinds          int `json:"kinds"`
}

// FinancialSettings مفاتيحُ الإعدادات التي تدخل حساباً ماليّاً.
//
// **ولا يُنسَخ مفتاحٌ من ذاكرة** (البند ٩): كلُّ مفتاحٍ هنا مقروءٌ من موضع
// قراءتِه في `internal/pricing` و`internal/cashbox` و`internal/server`،
// **وحارسٌ في الاختبار يطابقها بمعجم الإعدادات فيسقط إن اختفى مفتاح.**
var FinancialSettings = []string{
	"merchants.commission_percent",          // pricing.go:153 — عمولةُ المنصّة
	"sales.commission_percent",              // pricing.go:164 — عمولةُ المندوب
	"sales.commission_source",               // pricing.go — مصدرُ احتساب عمولته
	"sales.activation_orders",               // merchantActivated — عتبةُ التفعيل
	"pricing.margin_fixed",                  // pricing.go:69  — هامشُ التسعير
	"delivery.fee",                          // pricing.go:195 — أجرةُ التوصيل
	"delivery.per_km",                       // pricing.go:229
	"delivery.max_fee",                      // pricing.go:236
	"delivery.by_distance",                  // pricing.go:226
	"delivery.custom_fee_source",            // orders/custom.go — من يحدّد أجرةَ المخصَّص
	"delivery.custom_fee",                   // orders/custom.go — أجرةُ المخصَّص حين تحدّدها المنصة
	"delivery.custom_driver_may_change_fee", // orders/custom.go — هل يغيّرها السائق
	"drivers.cash_limit",                    // cashbox.go:52  — سقفُ النقد
	"customers.cod_limit",                   // orders/service.go — سقفُ نقد الزبون غيرِ المسدَّد
	"payouts.min_amount",                    // payout_handlers.go:120
	"customers.signup_bonus",                // referrals.go:443
	"referral.reward_1",
	"referral.reward_2",
	"referral.reward_3",
	"referral.reward_rest",
	"referral.reward_on",
	"drivers.target_reward",
	"drivers.reward_2",
	"drivers.reward_3",
	"sales.target_reward",
	"sales.reward_2",
	"sales.reward_3",
	"customers.cash_ban_days",
	"customers.cash_ban_failures",
	"orders.extra_source_fee",
	"orders.route_margin_pct",
}

// Snapshot يبني الصورة.
func Snapshot() Export {
	e := Export{SettingIDs: append([]string(nil), FinancialSettings...)}
	byFam := map[Family][]string{}
	for _, c := range All {
		byFam[c.Family] = append(byFam[c.Family], c.ID)
		e.Checks = append(e.Checks, CheckOut{
			ID: c.ID, Family: string(c.Family), Name: c.Name,
			Status: string(c.Status), Ops: c.Ops,
			Flows: c.Flows, Registers: c.Registers, Kinds: c.Kinds,
		})
		switch c.Status {
		case ProvableNow:
			e.Counts.ProvableNow++
		case NotImplemented:
			e.Counts.NotImplemented++
		case DeferredP5:
			e.Counts.DeferredP5++
		case DeferredP6:
			e.Counts.DeferredP6++
		}
	}
	for _, f := range Families() {
		e.Families = append(e.Families, FamilyOut{ID: string(f), Checks: byFam[f]})
	}
	for _, k := range ContractedKinds() {
		c := Kinds[k]
		e.Kinds = append(e.Kinds, KindOut{
			Kind: c.Kind, Semantics: c.Semantics, Sign: c.Sign,
			RefRequired: c.RefRequired, RefTarget: c.RefTarget, Path: c.Path,
			Creators: c.Creators, Invariants: c.Invariants, Reachable: c.Reachable,
		})
	}
	sort.Slice(e.Checks, func(i, j int) bool { return e.Checks[i].ID < e.Checks[j].ID })
	e.Counts.Families = len(e.Families)
	e.Counts.Checks = len(e.Checks)
	e.Counts.Kinds = len(e.Kinds)
	return e
}
