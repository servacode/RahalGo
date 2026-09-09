// عقدُ الخصوصيّة — **حقيقةٌ واحدةٌ تحكم القنواتِ كلَّها.**
//
// (المرحلةُ `P-1` من منظومة الاختبار الدائمة — قرارُ المالك ٢٠٢٦-٠٩-٠٥.)
//
// # لماذا وُجد
//
// **`D20` و`D21` عيبان مجمَّدان سببُهما واحد**: **قائمتان للتنقية لا
// واحدة.** `redactForCustomer` تمحو أربعةَ حقول و`redactForMerchant` تمحو
// ستّةَ عشر — **وكلٌّ منهما مكتوبةٌ بيدٍ في ملفّها، فما قُرّر في إحداهما
// لا يعرفه الآخر.** **وهاتفُ السائق مُنع من المتجر ووصل الزبون** — **لا
// سهواً في الاثنتين، بل قراراً وقع في واحدةٍ ونُسي في الثانية.**
//
// **والبثُّ لا يعرف أيَّهما**: `publishOrder` في طبقة `orders` والتنقيةُ
// في طبقة `server` — **فيمرّ الطلبُ كاملاً إلى غرفة المتجر.**
//
// # فما يفعله هذا الملفّ
//
// **يجعل القرارَ بيانةً لا شيفرة** — **جدولاً واحداً تستهلكه الاختباراتُ
// كلُّها**: REST والبثُّ والدفع. **فما يُقرَّر مرّةً يسري في القنوات
// الثلاث، ولا تنحرف قائمةٌ عن أختها.**
//
// # وما لا يفعله
//
// **لا يمسّ شيفرةَ الإنتاج.** **هذا حارسٌ يقيس ولا يُصلح** — وإصلاحُ
// `D20` و`D21` و`D22` مرحلةٌ أخرى بإذنٍ آخر.
package qa

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// Visibility حكمُ حقلٍ لدورٍ بعينه.
//
// **ولا قيمةَ صفريّةٍ تعني «مسموح»** — `VisUnknown` هو الصفر، **وهو
// سقوطٌ لا إجازة.** (البند ١١ من طلب المالك: `UNKNOWN = FAIL`.)
type Visibility int

const (
	// VisUnknown **الحقلُ لم يُصنَّف** — والاختبارُ يسقط.
	VisUnknown Visibility = iota
	// VisAllowed **يجوز أن يصل هذا الدورَ.**
	VisAllowed
	// VisForbidden **لا يجوز أن يصل** — ولو لم تعرضه الواجهة.
	VisForbidden
	// VisScoped **يخضع لصلاحيّةٍ داخلَ الدور.**
	//
	// **لا يُستعمل في `P-1`** — **ووُجد لئلّا يُغلَق التصميمُ دون
	// `AQ-1`.** (البند ١٢: الأدمنُ ليس استثناءً شاملاً.) **والاختبارُ
	// يتخطّاه اليومَ ويسجّله.**
	VisScoped
)

func (v Visibility) String() string {
	switch v {
	case VisAllowed:
		return "ALLOWED"
	case VisForbidden:
		return "FORBIDDEN"
	case VisScoped:
		return "SCOPED"
	}
	return "UNKNOWN"
}

// أدوارُ العقد — **وأسماؤها أسماءُ التطبيقات لا أسماءُ الأدوار في القاعدة.**
const (
	RoleCustomer = "customer"
	RoleMerchant = "merchant"
	RoleDriver   = "driver"
	RoleRep      = "rep"
	RoleAdmin    = "admin"
)

// PrivacyRoles الأدوارُ التي يُقاس عليها العقد.
var PrivacyRoles = []string{RoleCustomer, RoleMerchant, RoleDriver, RoleRep, RoleAdmin}

// القنواتُ التي يسري عليها العقدُ نفسُه — **ولا قائمةَ لكلّ قناة.**
const (
	ChannelREST     = "rest"
	ChannelRealtime = "realtime"
	ChannelPush     = "push"
)

// PrivacyChannels القنواتُ الثلاث.
var PrivacyChannels = []string{ChannelREST, ChannelRealtime, ChannelPush}

// FieldRule حكمُ حقلٍ واحدٍ في الأدوار الخمسة، ومرجعُ العقد الذي أوجبه.
type FieldRule struct {
	// Vis **الحكمُ لكلّ دور** — **وغيابُ دورٍ يعني `VisUnknown` فسقوط.**
	Vis map[string]Visibility
	// Ref **مرجعُ العقد أو العيب** — يُطبع في رسالة السقوط فيُعرَف سببُه.
	Ref string
}

// عقدُ الطلب — **الحقيقةُ الواحدة.**
//
// **والمفتاحُ وسمُ `json` لا اسمُ الحقل في Go** — **فهو ما يصل الطرفَ
// الآخرَ فعلاً**، وهو ما يُقاس.
//
// # كيف قُرِئت الأحكام
//
// **من العقود المعتمدة ومن الشيفرة القائمة معاً**:
//
//   - `redactForCustomer` (`server/customer_privacy.go:40`) — أربعةُ حقول
//   - `redactForMerchant` (`server/merchant_privacy.go:35`) — ستّةَ عشر
//   - `PC-6` وعقدُ خصوصيّة الزبون
//   - `MD-4`: «المتجرُ يسلّم لمن يأتي ولا شأنَ له بمن هو»
//   - `RQ-7`: «المندوبُ يرى استحقاقَه لا ربحَ المنصّة الداخليّ»
//
// **وما خالف الشيفرةُ فيه العقدَ يبقى العقدُ هو الحكم** — **فالاختبارُ
// يقيس الشيفرةَ بالعقد لا العكس.** (وهو موضعُ `D20` و`D21`.)
var OrderPrivacy = map[string]FieldRule{
	// ── الهويّةُ والحال — يراها الجميع ─────────────────────────────
	"id":     all(VisAllowed, "معرّفٌ فنّيّ"),
	"number": all(VisAllowed, "رقمُ الطلب — لغةُ التخاطب بين الأطراف"),
	"status": all(VisAllowed, "حالُ الطلب"),
	"kind":   all(VisAllowed, "عاديٌّ أم خاصّ"),
	"stage":  all(VisAllowed, "مرحلةُ العرض"),
	"stages": all(VisAllowed, "مراحلُ العرض"),
	"stage_at": all(VisAllowed,
		"وقتُ المرحلة"),
	"created_at": all(VisAllowed, "وقتُ الإنشاء"),

	// ── الزبون — **ممنوعٌ على المتجر بنصّ `redactForMerchant`** ────
	"customer_id":   {Ref: "merchant_privacy.go:36", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"customer_name": {Ref: "merchant_privacy.go:37 · والسائقُ يحتاجه للتسليم", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	// **وهاتفُ الزبون ممنوعٌ على السائق بالعقد** — `PC-6`.
	//
	// **ولا حارسَ له في الشيفرة**: لا `redactForDriver` إطلاقاً.
	// **فالمنعُ إن وقع فهو في الواجهة لا في الباب** — والحدُّ حدُّ
	// الحمولة الخارجة لا ما تعرضه الشاشة.
	"customer_phone": {Ref: "PC-6 · ولا حارسَ للسائق في الشيفرة", Vis: v(VisAllowed, VisForbidden, VisForbidden, VisForbidden, VisAllowed)},

	// ── العنوانُ والنقطة ───────────────────────────────────────────
	"address_text": {Ref: "merchant_privacy.go:39 · والسائقُ يوصِل إليه", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"lat":          {Ref: "merchant_privacy.go:40", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"lng":          {Ref: "merchant_privacy.go:40", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"zone_id":      {Ref: "merchant_privacy.go:41", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"zone_name":    {Ref: "merchant_privacy.go:41", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},

	// ── المتجر — **ممنوعٌ على الزبون بنصّ `redactForCustomer`** ────
	"merchant_id":              {Ref: "customer_privacy.go:41", Vis: v(VisForbidden, VisAllowed, VisAllowed, VisForbidden, VisAllowed)},
	"merchant_name":            {Ref: "customer_privacy.go:42", Vis: v(VisForbidden, VisAllowed, VisAllowed, VisForbidden, VisAllowed)},
	"merchant_logo_thumb_url":  {Ref: "customer_privacy.go:43", Vis: v(VisForbidden, VisAllowed, VisAllowed, VisForbidden, VisAllowed)},
	"merchant_accepts_returns": {Ref: "سياسةُ متجرٍ عامّة", Vis: v(VisAllowed, VisAllowed, VisAllowed, VisForbidden, VisAllowed)},
	"merchant_net":             {Ref: "مستحقُّ المتجر — لا يخصّ زبوناً ولا سائقاً ولا مندوباً", Vis: v(VisForbidden, VisAllowed, VisForbidden, VisForbidden, VisAllowed)},
	"prep_minutes":             {Ref: "مهلةُ التحضير", Vis: v(VisAllowed, VisAllowed, VisAllowed, VisForbidden, VisAllowed)},

	// ── السائق ────────────────────────────────────────────────────
	"driver_id":       {Ref: "merchant_privacy.go:44 · MD-4", Vis: v(VisForbidden, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"driver_name":     {Ref: "merchant_privacy.go:44 · MD-4 · والزبونُ يعرف من يطرق بابَه", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"driver_assigned": {Ref: "أُسنِد أم لا — حالٌ لا هويّة", Vis: v(VisAllowed, VisAllowed, VisAllowed, VisForbidden, VisAllowed)},
	// **`D21` هنا بعينه**: **يُمحى للمتجر صراحةً ولا يُمحى للزبون.**
	"driver_phone":        {Ref: "D21 · PC-6 · merchant_privacy.go:44", Vis: v(VisForbidden, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"offered_driver_name": {Ref: "customer_privacy.go:46 · merchant_privacy.go:45", Vis: v(VisForbidden, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},

	// ── المال — **وللمتجر منه تسويتُه هو لا سواها** ────────────────
	//
	// ══════════════════════════════════════════════════════════════
	// **تصحيحُ العقد** — قرارُ المالك ٢٠٢٦-٠٩-٠٩ (دورةُ ٤٦)
	// ══════════════════════════════════════════════════════════════
	//
	// **كان هذا الجدولُ يمنع المتجرَ من `subtotal` و
	// `platform_commission` و`commission_percent`** — **ويعرضها
	// تطبيقُه في شاشتَيه** («المجموع» · «خصم المنصة ١٠٪» · «المستحق
	// لك») **بقرارِ المالك ٢٠٢٦-٠٨-٢٦**: «سجلّ الطلبات ما فيه شقد
	// المبلغ المباع وشقد نسبة العمولة للمنصة — هيك لازم يكون
	// بشفافية».
	//
	// **فليست شيفرةٌ خالفت عقداً بل عقدان تخالفا** — **والمُصحَّحُ
	// هذا الجدولُ لا الشاشات.**
	//
	// # ومن أين جاء الخطأ
	//
	// **كُتبت هذه الصفوفُ من `redactForMerchant` وحدَها** — **وهي
	// تُصفّر `subtotal` فعلاً.** **ثمّ يُعيد `fillMerchantMoney`
	// ملأها بعد مئتَي سطر** — فمن قرأ التنقيةَ ولم يقرأ ما بعدها
	// **كتب في العقد غيرَ ما يخرج من الباب.**
	//
	// # وحدُّ الإذن — **تسويتُه هو لا اقتصادُ المنصّة**
	//
	// **والمأذونُ ثلاثةٌ بأعيانها**: **ما بِيع من عنده · ما اقتُطع
	// منه · بأيّ نسبة.** **ولا يُقاس عليها غيرُها** — أجرُ السائق
	// وعمولةُ المندوب وهامشُ المنصّة الداخليُّ وتسوياتُها **تبقى
	// ممنوعةً كما كانت** (`RQ-7`)، **ولا يُوسَّع العقدُ ليخضرَّ
	// فحص.**
	"payment_method": {Ref: "merchant_privacy.go:51", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	// **شفافيّةُ تسوية المتجر** — قرارُ المالك ٢٠٢٦-٠٨-٢٦ و٢٠٢٦-٠٩-٠٩.
	"subtotal":     {Ref: "شفافيّةُ تسوية المتجر (قرارُ المالك ٢٠٢٦-٠٩-٠٩) — ما بِيع من عنده · وRQ-7 يجيزه للمندوب", Vis: v(VisAllowed, VisAllowed, VisAllowed, VisAllowed, VisAllowed)},
	"delivery_fee": {Ref: "merchant_privacy.go:49 · وأجرُ السائق", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisAllowed, VisAllowed)},
	"discount":     {Ref: "merchant_privacy.go:49", Vis: v(VisAllowed, VisForbidden, VisForbidden, VisForbidden, VisAllowed)},
	"total":        {Ref: "merchant_privacy.go:49 · وRQ-7", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisAllowed, VisAllowed)},
	"wallet_paid":  {Ref: "merchant_privacy.go:50", Vis: v(VisAllowed, VisForbidden, VisForbidden, VisForbidden, VisAllowed)},
	"cash_due":     {Ref: "merchant_privacy.go:50 · والسائقُ يقبضه", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"promo_code":   {Ref: "merchant_privacy.go:52", Vis: v(VisAllowed, VisForbidden, VisForbidden, VisForbidden, VisAllowed)},
	// **`RQ-7`**: «المندوبُ يرى استحقاقَه لا ربحَ المنصّة الداخليّ».
	//
	// **والمتجرُ يرى اقتطاعَه هو** (قرارُ المالك ٢٠٢٦-٠٩-٠٩): **ما
	// خُصم من طلبه ونسبتُه** — **ولا يرى ما اقتُطع من غيره ولا ربحَ
	// المنصّة جملةً.** **والمندوبُ يبقى ممنوعاً** (`XG-16`).
	"platform_commission": {Ref: "شفافيّةُ تسوية المتجر (قرارُ المالك ٢٠٢٦-٠٩-٠٩) — ما اقتُطع من طلبه · وRQ-7/XG-16 يمنعانه عن المندوب", Vis: v(VisForbidden, VisAllowed, VisForbidden, VisForbidden, VisAllowed)},
	"commission_percent":  {Ref: "شفافيّةُ تسوية المتجر (قرارُ المالك ٢٠٢٦-٠٩-٠٩) — بأيّ نسبةٍ اقتُطع · وRQ-7 يمنعها عن المندوب", Vis: v(VisForbidden, VisAllowed, VisForbidden, VisForbidden, VisAllowed)},

	// ── الطلبُ الخاصّ ──────────────────────────────────────────────
	"custom_request":      {Ref: "نصُّ الطلب — لصاحبه ولمن ينفّذه", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"custom_agreed_at":    {Ref: "وقتُ الاتّفاق", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"custom_goods_amount": {Ref: "قيمةُ البضاعة", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"custom_fee":          {Ref: "أجرُ الخدمة", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},

	// ── الأصنافُ والتوقيتات ───────────────────────────────────────
	"items":                 {Ref: "الأصناف — والمتجرُ يحضّرها", Vis: v(VisAllowed, VisAllowed, VisAllowed, VisForbidden, VisAllowed)},
	"items_count":           all(VisAllowed, "عددُ الأصناف"),
	"items_preview":         {Ref: "معاينةٌ نصّيّة", Vis: v(VisAllowed, VisAllowed, VisAllowed, VisForbidden, VisAllowed)},
	"delivery_estimate_min": all(VisAllowed, "الزمنُ المتوقَّع"),
	"ready_at":              all(VisAllowed, "وقتُ الجهوزيّة"),
	"accepted_at":           all(VisAllowed, "وقتُ القبول"),
	"picked_up_at":          all(VisAllowed, "وقتُ الاستلام"),
	"delivered_at":          all(VisAllowed, "وقتُ التسليم"),
	"closed_at":             all(VisAllowed, "وقتُ الإغلاق"),
	"returned_at":           all(VisAllowed, "وقتُ الإرجاع"),
	"sent_to_merchant_at":   {Ref: "تشغيليٌّ للمكتب والمتجر", Vis: v(VisForbidden, VisAllowed, VisForbidden, VisForbidden, VisAllowed)},
	"dispatched_at":         {Ref: "تشغيليٌّ للإرسال", Vis: v(VisAllowed, VisAllowed, VisAllowed, VisForbidden, VisAllowed)},

	// ── إثباتُ التسليم — **صورةُ بابِ زبونٍ وعنوانِه** ──────────────
	"proof_url":      {Ref: "merchant_privacy.go:46 · D13 — صورةُ بابِ الزبون", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"proof_taken_at": {Ref: "merchant_privacy.go:46", Vis: v(VisAllowed, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"proof_meters":   {Ref: "merchant_privacy.go:47", Vis: v(VisForbidden, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"proof_skip_reason": {Ref: "merchant_privacy.go:47 — تشغيليٌّ داخليّ",
		Vis: v(VisForbidden, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"proof_mocked": {Ref: "موقعٌ مزيَّفٌ — حكمٌ داخليٌّ على السائق", Vis: v(VisForbidden, VisForbidden, VisForbidden, VisForbidden, VisAllowed)},

	// ── الإنهاءُ والمسؤوليّة — **أحكامٌ داخليّةٌ لا تُنشَر** ─────────
	"cancel_reason":  {Ref: "السببُ يُقال لمن يخصّه", Vis: v(VisAllowed, VisAllowed, VisAllowed, VisAllowed, VisAllowed)},
	"blocked_reason": {Ref: "سببُ الحظر — لصاحبه والإدارة", Vis: v(VisAllowed, VisForbidden, VisForbidden, VisForbidden, VisAllowed)},
	"ended_by":       {Ref: "من أنهى — حكمٌ تشغيليٌّ للإدارة", Vis: v(VisForbidden, VisForbidden, VisForbidden, VisForbidden, VisAllowed)},
	"fault":          {Ref: "نسبةُ الخطأ — حكمٌ على طرف", Vis: v(VisForbidden, VisForbidden, VisForbidden, VisForbidden, VisAllowed)},
	"fail_reason":    {Ref: "سببُ التعذّر", Vis: v(VisAllowed, VisAllowed, VisAllowed, VisForbidden, VisAllowed)},
	"goods_settled_to": {Ref: "لمن سُلّمت البضاعةُ — تسويةٌ داخليّة",
		Vis: v(VisForbidden, VisForbidden, VisForbidden, VisForbidden, VisAllowed)},

	// ── الملاحةُ والمسافات ────────────────────────────────────────
	"to_store_eta_sec":   {Ref: "زمنُ وصول السائق — للزبون والمتجر", Vis: v(VisAllowed, VisAllowed, VisAllowed, VisForbidden, VisAllowed)},
	"to_door_eta_sec":    {Ref: "زمنُ الوصول للباب", Vis: v(VisAllowed, VisAllowed, VisAllowed, VisForbidden, VisAllowed)},
	"leg_m":              {Ref: "طولُ المرحلة — تشغيليٌّ للسائق", Vis: v(VisForbidden, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},
	"driver_to_pickup_m": {Ref: "بُعدُ السائق — موضعُه ضمناً", Vis: v(VisForbidden, VisForbidden, VisAllowed, VisForbidden, VisAllowed)},

	// ── الملاحظاتُ والأحداثُ والتقييم ──────────────────────────────
	"notes":  {Ref: "تعليماتُ الزبون — والسائقُ ينفّذها", Vis: v(VisAllowed, VisAllowed, VisAllowed, VisForbidden, VisAllowed)},
	"events": {Ref: "الخطُّ الزمنيّ — يحمل فاعلين", Vis: v(VisForbidden, VisForbidden, VisForbidden, VisForbidden, VisAllowed)},
	"rating": {Ref: "تقييمُ الزبون", Vis: v(VisAllowed, VisForbidden, VisForbidden, VisForbidden, VisAllowed)},

	// ── مراحلُ المكتب — **داخليّةٌ للإدارة وحدَها** ─────────────────
	"ops_stages":      adminOnly("مراحلُ مكتبِ العمليّات"),
	"ops_stage_at":    adminOnly("وقتُ مرحلة المكتب"),
	"ops_stage_times": adminOnly("أزمنةُ مراحل المكتب"),
	"ops_stage_late":  adminOnly("تأخّرُ مرحلةِ مكتب"),
}

// v يبني حكماً بالترتيب: زبونٌ · متجرٌ · سائقٌ · مندوبٌ · أدمن.
func v(customer, merchant, driver, rep, admin Visibility) map[string]Visibility {
	return map[string]Visibility{
		RoleCustomer: customer,
		RoleMerchant: merchant,
		RoleDriver:   driver,
		RoleRep:      rep,
		RoleAdmin:    admin,
	}
}

// all حكمٌ واحدٌ للأدوار الخمسة.
func all(vis Visibility, ref string) FieldRule {
	return FieldRule{Ref: ref, Vis: v(vis, vis, vis, vis, vis)}
}

// adminOnly **للإدارة وحدَها** — وممنوعٌ على الأربعة.
func adminOnly(ref string) FieldRule {
	return FieldRule{Ref: ref, Vis: v(VisForbidden, VisForbidden, VisForbidden, VisForbidden, VisAllowed)}
}

// OrderFields **حقولُ الطلب كما تخرج فعلاً** — بالانعكاس لا بقائمةٍ مكتوبة.
//
// **وهذا لبُّ الحارس**: **من أضاف حقلاً إلى `orders.Order` ظهر هنا في
// الحال**، **وإن لم يُصنَّف سقط الاختبار.** (البند ٢ و١٠ من طلب المالك.)
//
// **والحقولُ المُهمَلةُ في التسلسل (`json:"-"`) لا تخرج فلا تُصنَّف.**
func OrderFields() []string {
	t := reflect.TypeOf(orders.Order{})
	out := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" || name == "-" {
			continue
		}
		out = append(out, name)
	}
	return out
}

// RestEnvelope **حقولُ الغلاف في `REST`** — ما ليس من `orders.Order`.
//
// **و`REST` تزيد على الحمولة ما يلزم شاشتَها**: مهلةُ الإلغاء ومسارُ
// الطلب بأوقاته. **وهي عرضٌ لا حقولُ كائن** — **فلا مكانَ لها في
// `OrderPrivacy`** (حارسُ الأشباح يرفض ما ليس في البنية).
//
// **ولا تُتخطّى بصمت**: **ما لم يُذكَر هنا يُقرأ مجهولاً فيسقط** —
// **وغلافٌ مفتوحٌ بابٌ خلفيٌّ للتسريب.**
//
// # وما دخل وما لم يدخل
//
//   - `cancel_seconds_left` — **مهلةُ زرّ الإلغاء**، رقمٌ من الخادم
//     لئلّا تخالفه الشاشةُ بعد أوّل تعديل
//   - `timeline` — **مسارُه بأوقاته** (قرارُ المالك ٢٠٢٦-٠٨-١٢):
//     **حالٌ ووقتُه ولا فاعلَ فيه** — **بخلاف `events` التي تحمل
//     `actor_id` فتبقى ممنوعة**
var RestEnvelope = map[string]map[string]bool{
	RoleCustomer: {"cancel_seconds_left": true, "timeline": true},
	RoleMerchant: {},
	RoleDriver:   {},
	RoleRep:      {},
	RoleAdmin:    {},
}

// Visible حكمُ حقلٍ لدور — **والمجهولُ سقوطٌ لا إجازة.**
func Visible(field, role string) Visibility {
	rule, ok := OrderPrivacy[field]
	if !ok {
		return VisUnknown
	}
	vis, ok := rule.Vis[role]
	if !ok {
		return VisUnknown
	}
	return vis
}

// Violation خرقٌ واحدٌ مُثبَت.
type Violation struct {
	Field   string
	Role    string
	Channel string
	Value   any
	Vis     Visibility
	Ref     string
}

func (x Violation) String() string {
	return fmt.Sprintf("[%s/%s] الحقلُ %q %s — وصلت قيمتُه %v  (%s)",
		x.Role, x.Channel, x.Field, x.Vis, x.Value, x.Ref)
}

// CheckPayload يقيس حمولةً خارجةً بالعقد ويردّ كلَّ خرق.
//
// **ولا يُفحَص إلّا ما خرج فعلاً**: **حقلٌ غائبٌ أو صفريٌّ ليس خرقاً** —
// **فالتنقيةُ في هذا المشروع تصفّر ولا تحذف** (`o.CustomerPhone = ""`)،
// **والقيمةُ الصفريّةُ لا تكشف شيئاً.**
//
// **والقناةُ تُمرَّر للرسالة لا للحكم** — **العقدُ واحدٌ عبرها كلِّها.**
// (البند ٨ من طلب المالك: لا قائمتان تنحرفان.)
func CheckPayload(role, channel string, payload map[string]any) []Violation {
	var out []Violation
	for field, val := range payload {
		vis := Visible(field, role)
		switch vis {
		case VisAllowed, VisScoped:
			continue
		case VisForbidden, VisUnknown:
			if isEmpty(val) {
				continue
			}
			out = append(out, Violation{
				Field: field, Role: role, Channel: channel,
				Value: val, Vis: vis, Ref: OrderPrivacy[field].Ref,
			})
		}
	}
	return out
}

// isEmpty **أوصلت القيمةُ شيئاً؟** — الصفرُ والفراغُ والعدمُ لا تُوصِل.
func isEmpty(val any) bool {
	if val == nil {
		return true
	}
	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.String:
		return rv.Len() == 0
	case reflect.Slice, reflect.Map, reflect.Array:
		return rv.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return rv.IsNil()
	case reflect.Float64, reflect.Float32:
		return rv.Float() == 0
	case reflect.Int, reflect.Int64, reflect.Int32:
		return rv.Int() == 0
	case reflect.Bool:
		return !rv.Bool()
	}
	return false
}
