package eventmap

import "sort"

// Channel قناةُ التوصيل.
type Channel string

const (
	ChRealtime Channel = "REALTIME"
	ChPush     Channel = "PUSH"
	ChInApp    Channel = "IN_APP"
)

// Durability هل يُعاد إن سقط.
type Durability string

const (
	// BestEffort يُحاوَل مرّةً — **ولا إعادةَ ولا صفَّ انتظار.**
	BestEffort Durability = "BEST_EFFORT"
	// DurableRequired العقدُ يشترط أن يعلم المستلمُ — **فسقوطُه يحتاج مساراً.**
	DurableRequired Durability = "DURABLE_REQUIRED"
	// LogicalPart جزءٌ من نجاحِ العمليّة نفسِها.
	LogicalPart Durability = "PART_OF_LOGICAL_SUCCESS"
)

// Status حالُ العقد كما قِيس.
type Status string

const (
	Held         Status = "HELD"
	ExpectedFail Status = "EXPECTED_FAIL"
	Partial      Status = "PARTIAL"
	Unproven     Status = "UNPROVEN"
	DeferredP8   Status = "DEFERRED_TO_P8"
)

// Event عقدُ حدثٍ واحد.
type Event struct {
	ID string `json:"id"`
	// Flow تدفّقُ المنظومة.
	Flows []string `json:"flows"`
	// Title ما هو.
	Title string `json:"title"`
	// Entity نوعُ الكيان وأيلزم معرّفُه.
	Entity   string `json:"entity"`
	NeedsID  bool   `json:"entity_id_required"`
	DeepLink string `json:"deep_link_contract"`
	// Audience من يجب أن يستقبل.
	Audience []string `json:"must_receive"`
	// Forbidden من يجب ألّا يستقبل.
	Forbidden []string `json:"must_not_receive"`
	// Channels القنواتُ المطلوبة.
	Channels []Channel `json:"channels"`
	// Apps تطبيقاتُ المستلم.
	Apps []string `json:"apps,omitempty"`
	// Privacy عقدُ الخصوصيّة — **يشير إلى `P-1` ولا ينسخه.**
	Privacy string `json:"privacy_contract"`
	// Durability ما يقع عند السقوط.
	Durability Durability `json:"durability"`
	// Retry سياسةُ الإعادة كما قِيست.
	Retry string `json:"retry_policy"`
	// Tests الاختباراتُ التي تحرسه.
	Tests []string `json:"tests"`
	// Status الحالُ المقيسة.
	Status Status `json:"status"`
	// Evidence سطرُ الدليل — **ولا حالَ بلا دليل.**
	Evidence string `json:"evidence"`
	// Registers ما يمسّه من السجلّات.
	Registers []string `json:"registers,omitempty"`
}

// Events العقودُ المعلَنة.
//
// **ولا يُخترَع حدثٌ لا وجودَ له** (البند ٢) — كلُّ ما هنا له موضعُ إطلاقٍ
// أو بثٍّ في الشيفرة.
var Events = []Event{
	{
		ID: "EV-01", Title: "طلبٌ جديدٌ للمتجر",
		Flows: []string{"F-01"}, Entity: "order", NeedsID: true,
		DeepLink:   "/orders/{id} في تطبيق المتجر",
		Audience:   []string{"merchant-owner"},
		Forbidden:  []string{"customer", "driver", "rep"},
		Channels:   []Channel{ChRealtime, ChPush, ChInApp},
		Apps:       []string{"merchant"},
		Privacy:    "OrderPrivacy[merchant] — P-1",
		Durability: DurableRequired, Retry: "لا إعادةَ مقيسة",
		Tests:     []string{"TestEV_MerchantRealtimePrivacy", "TestEV_OrderAudience"},
		Status:    ExpectedFail,
		Evidence:  "71 حقلاً وصل غرفةَ المتجر · 19 منها محظورٌ بعقد P-1",
		Registers: []string{"D20"},
	},
	{
		ID: "EV-02", Title: "تحديثُ حالِ الطلب للزبون",
		Flows: []string{"F-04", "F-08", "F-12", "F-13"}, Entity: "order", NeedsID: true,
		DeepLink:   "/orders/{id} في تطبيق الزبون",
		Audience:   []string{"order-customer"},
		Forbidden:  []string{"other-customer", "merchant-of-another-order"},
		Channels:   []Channel{ChRealtime, ChPush, ChInApp},
		Apps:       []string{"customer"},
		Privacy:    "OrderPrivacy[customer] — P-1 · ولا هاتفَ سائق",
		Durability: DurableRequired, Retry: "لا إعادةَ مقيسة",
		Tests:     []string{"TestEV_CustomerDriverAssignment", "TestEV_OrderAudience"},
		Status:    ExpectedFail,
		Evidence:  "driver_phone وصل الزبونَ لحظيّاً وREST — 9 و10 خروق",
		Registers: []string{"D21", "D23"},
	},
	{
		ID: "EV-03", Title: "الطلبُ الخاصُّ لصاحبه",
		Flows: []string{"F-02"}, Entity: "order", NeedsID: true,
		DeepLink:   "/orders/{id}",
		Audience:   []string{"order-customer"},
		Forbidden:  []string{"other-customer"},
		Channels:   []Channel{ChRealtime},
		Apps:       []string{"customer"},
		Privacy:    "OrderPrivacy[customer] — P-1",
		Durability: DurableRequired, Retry: "—",
		Tests:     []string{"TestEV_CustomOrderOwnerRealtime"},
		Status:    ExpectedFail,
		Evidence:  "العاديُّ وصل صاحبَه (true) والخاصُّ لا (false)",
		Registers: []string{"D22"},
	},
	{
		ID: "EV-04", Title: "قبولٌ تلقائيٌّ يُعلَم به المتجر",
		Flows: []string{"F-05"}, Entity: "order", NeedsID: true,
		DeepLink:   "/orders/{id}",
		Audience:   []string{"merchant-owner"},
		Forbidden:  []string{"customer"},
		Channels:   []Channel{ChInApp, ChPush},
		Apps:       []string{"merchant"},
		Privacy:    "OrderPrivacy[merchant] — P-1",
		Durability: DurableRequired, Retry: "—",
		Tests:  []string{"TestEV_AutoAcceptMerchantAwareness"},
		Status: Partial, Evidence: "لم يقع القبولُ التلقائيُّ — لا بابَ يُنادي sweepAutoAccept · SEAM",
		Registers: []string{"MD-5"},
	},
	{
		ID: "EV-05", Title: "إسنادُ سائقٍ يُعلَم به المتجر",
		Flows: []string{"F-08", "F-19"}, Entity: "order", NeedsID: true,
		DeepLink:   "/orders/{id}",
		Audience:   []string{"merchant-owner"},
		Forbidden:  []string{"—"},
		Channels:   []Channel{ChRealtime, ChInApp},
		Apps:       []string{"merchant"},
		Privacy:    "**ولا هويّةَ سائقٍ للمتجر** — OrderPrivacy[merchant]",
		Durability: DurableRequired, Retry: "—",
		Tests:  []string{"TestEV_MerchantDriverAssignment"},
		Status: Partial, Evidence: "المتجرُ أُعلِم بحمولةٍ واحدة · و3 حقولِ هويّةِ سائقٍ تسرّبت",
		Registers: []string{"MD-2", "D20"},
	},
	{
		ID: "EV-06", Title: "حركةُ محفظةٍ يُعلَم بها صاحبُها",
		Flows: []string{"F-14", "F-25"}, Entity: "wallet", NeedsID: false,
		DeepLink:   "/wallet",
		Audience:   []string{"wallet-owner"},
		Forbidden:  []string{"any-other-user"},
		Channels:   []Channel{ChInApp, ChPush},
		Privacy:    "لا حقولَ طلبٍ في مغلّف الإشعار",
		Durability: DurableRequired, Retry: "لا إعادةَ مقيسة",
		Tests:  []string{"TestEV_WalletAudience"},
		Status: Held, Evidence: "صاحبُ المحفظة 0⇒1 وغيرُه 0⇒0",
	},
	{
		ID: "EV-07", Title: "إنذارُ الراصد للعمليّات",
		Flows: []string{"F-09"}, Entity: "order", NeedsID: true,
		DeepLink:   "/dashboard/orders",
		Audience:   []string{"ops"},
		Forbidden:  []string{"customer", "driver", "merchant"},
		Channels:   []Channel{ChRealtime, ChInApp},
		Privacy:    "داخليٌّ — لا قيدَ خصوصيّةِ زبون",
		Durability: DurableRequired,
		Retry:      "**لا إعادة** — الوسمُ يسبق الإشعارَ ويمنعها",
		Tests:      []string{"TestEV_R22WatchdogMarkerSuppressesRetry"},
		Status:     ExpectedFail,
		Evidence:   "التشغيلُ الأوّل أشعر مرّةً · والثاني صفراً — الوسمُ يمنع",
		Registers:  []string{"R22", "XOB-6"},
	},
	{
		ID: "EV-08", Title: "دفعٌ إلى جهاز",
		Flows: []string{"F-07", "F-14"}, Entity: "*", NeedsID: true,
		DeepLink:   "Data[entity] و Data[entity_id]",
		Audience:   []string{"token-owner"},
		Forbidden:  []string{"other-users-tokens"},
		Channels:   []Channel{ChPush},
		Privacy:    "لا حقلَ طلبٍ في المغلّف — P-1 CheckPayload(push)",
		Durability: BestEffort,
		Retry:      "**مرّةٌ واحدةٌ بلا إعادة** — يُقاس",
		Tests:      []string{"TestEV_R23PushFailureIsLost", "TestEV_PushTokenTargeting"},
		Status:     ExpectedFail,
		Evidence:   "ثلاثةُ إخفاقاتٍ · ثلاثةُ نداءاتٍ · ولا نداءَ رابع",
		Registers:  []string{"R23", "D12"},
	},
	{
		ID: "EV-09", Title: "تحويلٌ تلقائيٌّ يُعلَم به المتجر",
		Flows: []string{"F-18"}, Entity: "order", NeedsID: true,
		DeepLink:   "/orders/{id}",
		Audience:   []string{"merchant-owner"},
		Forbidden:  []string{"—"},
		Channels:   []Channel{ChInApp, ChRealtime},
		Apps:       []string{"merchant"},
		Privacy:    "OrderPrivacy[merchant] — P-1",
		Durability: DurableRequired,
		Retry:      "**خيطٌ لا يُنتظَر** — XOB-7",
		Tests: []string{"TestEV_R21AutoTransferAwareness",
			"TestR21_T1_SelfManageAutoTransferInformsMerchant",
			"TestR21_T2_PlatformModeChecksChannelBeforeAccepting"},
		Status: Held,
		Evidence: "**والقياسُ القديمُ كان يضبط `orders.auto_transfer_amount`** " +
			"— **ومفتاحُ التشغيل `orders.auto_transfer` منطقيٌّ بلا عتبة** " +
			"(«وذهبت العتبتان»). **فلم يقع تحويلٌ قطُّ، والطلبُ يبقى " +
			"`pending` فيُقرأ `PARTIAL`** — **وذاك سببُ بقائه «يُقاس في " +
			"التشغيل».** " +
			"**وبالمفتاح الصحيح**: وضعُ المتاجر ⇒ الحالُ `dispatching` · " +
			"**إشعارٌ دائمٌ للمتجر 0⇒1** · وأربعُ رسائلِ بثّ. " +
			"**ووضعُ المنصّة ⇒ القناةُ تُفحص قبل القبول**: البوتُ غيرُ " +
			"جاهزٍ فيبقى `pending` — **ولا يُقبَل طلبٌ لا يعلم به متجرُه.**",
		Registers: []string{"R21", "XOB-7"},
	},
	{
		ID: "EV-10", Title: "طلبان متزامنان لا يختلطان",
		Flows: []string{"F-01"}, Entity: "order", NeedsID: true,
		DeepLink:   "لكلٍّ معرّفُه",
		Audience:   []string{"order-customer"},
		Forbidden:  []string{"cross-order-mix"},
		Channels:   []Channel{ChRealtime, ChInApp},
		Apps:       []string{"customer"},
		Privacy:    "OrderPrivacy[customer]",
		Durability: DurableRequired, Retry: "—",
		Tests:  []string{"TestEV_MultiOrderRouting"},
		Status: Held, Evidence: "طلبان بمعرّفين متمايزين ولا اختلاط",
	},
	{
		ID: "EV-11", Title: "غرفةُ الدورِ لا يدخلها غيرُ أهلها",
		Flows: []string{"F-34"}, Entity: "*", NeedsID: false,
		DeepLink:   "—",
		Audience:   []string{"room-owner"},
		Forbidden:  []string{"other-roles", "other-users"},
		Channels:   []Channel{ChRealtime},
		Privacy:    "التخويلُ قبل الاشتراك",
		Durability: LogicalPart, Retry: "—",
		Tests:  []string{"TestEV_RealtimeAuthorization"},
		Status: Held, Evidence: "زبونٌ آخرُ لم يستقبل · ومتجرٌ آخرُ لم يستقبل",
		// ══════════════════════════════════════════════════════════
		// **و`R14` نُزعت من هنا** — دورةُ ٢٣
		// ══════════════════════════════════════════════════════════
		//
		// **هذا العقدُ يقيس «التخويلُ قبل الاشتراك»** — من يدخل
		// الغرفةَ أصلاً.
		//
		// **و`R14` تقول «اتّصالُ البثّ لا يُراجَع بعد المصافحة»** —
		// **وصلةٌ قائمةٌ يُبطَل رمزُها فتبقى مفتوحة.**
		//
		// **وسؤالان مختلفان** — **ودليلٌ يُنسَب إلى سجلٍّ لا يقيسه
		// يُغلقه بالكلام.** **فتبقى `R14` بدليلها وحدَه.**
		Registers: nil,
	},
}

// Counts إحصاءٌ مولَّد.
type Counts struct {
	Events, Held, ExpectedFail, Partial, Unproven, DeferredP8 int
	Realtime, Push, InApp                                     int
}

// Snapshot صورةٌ آليّة.
func Snapshot(sites []Site, pubs []Publisher) map[string]any {
	c := Counts{Events: len(Events)}
	for _, e := range Events {
		switch e.Status {
		case Held:
			c.Held++
		case ExpectedFail:
			c.ExpectedFail++
		case Partial:
			c.Partial++
		case Unproven:
			c.Unproven++
		case DeferredP8:
			c.DeferredP8++
		}
		for _, ch := range e.Channels {
			switch ch {
			case ChRealtime:
				c.Realtime++
			case ChPush:
				c.Push++
			case ChInApp:
				c.InApp++
			}
		}
	}
	ev := append([]Event(nil), Events...)
	sort.Slice(ev, func(i, j int) bool { return ev[i].ID < ev[j].ID })
	return map[string]any{
		"events": ev, "counts": c,
		"emit_sites":      sites,
		"realtime_pubs":   pubs,
		"rooms":           Rooms(pubs),
		"untargeted_user": UntargetedUserSites(sites),
		"derived": map[string]int{
			"emit_sites":            len(sites),
			"realtime_publishers":   len(pubs),
			"rooms":                 len(Rooms(pubs)),
			"untargeted_user_sites": len(UntargetedUserSites(sites)),
		},
	}
}

// Tests كلُّ الاختبارات المذكورة.
func Tests() []string {
	seen := map[string]bool{}
	var out []string
	for _, e := range Events {
		for _, n := range e.Tests {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	sort.Strings(out)
	return out
}
