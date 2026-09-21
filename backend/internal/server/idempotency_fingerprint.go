package server

// ══════════════════════════════════════════════════════════════════════
// **بصمةُ الجسم — مفتاحٌ واحدٌ لجسمين مختلفين لا يمرّ**
// ══════════════════════════════════════════════════════════════════════
//
// (`CAF-02` · `13-029` — قرارُ المالك ٢٠٢٦-٠٩-٢١.)
//
// # لماذا بصمةٌ لا مقارنةٌ بايتيّة
//
// **جسمان متساويان بالمعنى قد يختلفان بالبايت**: ترتيبُ الأصناف في
// السلّة، ترتيبُ خيارات الصنف، مسافةٌ، `36` مقابل `36.0`. **ومقارنةُ
// النصّ الخام ترى هذه فروقاً فتردّ `409` على إعادةٍ صادقة.**
//
// **فتُبنى صورةٌ قانونيّةٌ للحقول ذات المعنى وحدَها** ثمّ تُبصَم: **ترتيبُ
// الأصناف يُوحَّد بالفرز، والخيارات تُفرَز، والعابرُ يُطرَح** (المفتاحُ
// نفسُه، والطوابعُ، ومعرّفُ الزبون الذي يُشتقّ من الرمز لا من الجسم).
//
// # ولماذا الحقولُ الدلاليّةُ وحدَها
//
// **جسمُ الطلب قد يحمل حقولاً لا تغيّر الطلب** (معرّفُ زبونٍ يتجاهله
// الخادم، هاتفٌ يُمحى). **فتُقرأ الحقولُ المقصودةُ فقط** — **وما لم
// يُقرأ لا يدخل البصمة**، فلا يُرفَض طلبٌ لاختلافٍ لا أثرَ له.

import (
	"crypto/sha256"
	"encoding/json"
	"sort"
)

// نقاطُ الإنشاء المبصومة — مساراتُها الكاملةُ كما في `endpoint`
// (`r.Method + " " + r.URL.Path`).
const (
	epNormalOrder = "POST /api/v1/orders"
	epCustomOrder = "POST /api/v1/orders/custom"
)

// wantsFingerprint هل تُبصَم هذه النقطة؟
//
// **وما لا يُبصَم لا يُقرأ جسمُه أصلاً** — فالحرسُ محصورٌ في بابَي الإنشاء،
// **ولا يمسّ سائرَ الأبواب المحميّة** (المحفظة، السحب، التسوية، المكافأة):
// مفاتيحُها تحرسها، **وحرسُ المحتوى لم يُطلَب لها في هذه الدورة.**
func wantsFingerprint(endpoint string) bool {
	return endpoint == epNormalOrder || endpoint == epCustomOrder
}

// fingerprintForEndpoint يحسب بصمةَ الجسم الدلاليّة حسب النقطة.
//
// **يُرجع `nil` لجسمٍ لا يُبصَم** (نقطةٌ غيرُ مسجّلة) — فيُعامَل كصفٍّ بلا
// بصمة. **وجسمٌ فاسدٌ يُبصَم على صورته الفارغة** — **ولا يُحقَّق هنا**:
// التحقّقُ عملُ المعالج، والبصمةُ لا تحكم على صحّة الجسم بل على تطابقه.
func fingerprintForEndpoint(endpoint string, body []byte) []byte {
	switch endpoint {
	case epNormalOrder:
		return fingerprintNormalOrder(body)
	case epCustomOrder:
		return fingerprintCustomOrder(body)
	default:
		return nil
	}
}

// fpItem صنفٌ في السلّة — بحقوله ذات المعنى وحدَها.
type fpItem struct {
	MenuItemID string   `json:"menu_item_id"`
	Qty        int      `json:"qty"`
	Note       string   `json:"note"`
	OptionIDs  []string `json:"option_ids"`
}

// fpNormalOrder الصورةُ القانونيّةُ للطلب العاديّ.
//
// **دون `customer_id`/`customer_phone`**: الخادمُ يشتقّهما من الرمز
// ويطرح ما أرسله العميل، **فلا يدخلان البصمة.**
type fpNormalOrder struct {
	MerchantID    string   `json:"merchant_id"`
	Items         []fpItem `json:"items"`
	AddressText   string   `json:"address_text"`
	Lat           float64  `json:"lat"`
	Lng           float64  `json:"lng"`
	PaymentMethod string   `json:"payment_method"`
	PromoCode     string   `json:"promo_code"`
	Notes         string   `json:"notes"`
}

func fingerprintNormalOrder(body []byte) []byte {
	var in fpNormalOrder
	// **بصمةٌ لا تحقّق**: جسمٌ فاسدٌ يُبصَم كما فُكّ (صورةٌ فارغة) ويرفضه
	// المعالجُ بعدَها بـ`validation`.
	_ = json.Unmarshal(body, &in)

	// **خياراتُ الصنف مجموعة** — ترتيبُها لا يعني شيئاً، فتُفرَز.
	for i := range in.Items {
		sort.Strings(in.Items[i].OptionIDs)
	}
	// **والأصنافُ مرتّبةٌ بمفتاحٍ قانونيّ** — **فإعادةُ ترتيب السلّة ليست
	// طلباً آخر**، لكنّ زيادةَ صنفٍ أو كمّيّةٍ أو خيارٍ تغيّر المفتاح.
	// **والتكرارُ يبقى** (صنفان متطابقان سطران، لا سطرٌ واحد): الفرزُ لا
	// يحذف — فالكمّيّةُ الحقيقيّةُ محفوظة.
	sort.SliceStable(in.Items, func(a, b int) bool {
		return fpItemKey(in.Items[a]) < fpItemKey(in.Items[b])
	})

	return canonHash(in)
}

// fpCustomOrder الصورةُ القانونيّةُ للطلب المخصَّص.
type fpCustomOrder struct {
	Request       string  `json:"request"`
	AddressText   string  `json:"address_text"`
	Lat           float64 `json:"lat"`
	Lng           float64 `json:"lng"`
	PaymentMethod string  `json:"payment_method"`
	Notes         string  `json:"notes"`
}

func fingerprintCustomOrder(body []byte) []byte {
	var in fpCustomOrder
	_ = json.Unmarshal(body, &in)
	return canonHash(in)
}

// fpItemKey مفتاحُ فرزٍ قانونيٌّ للصنف — يلتقط كلَّ حقوله.
//
// **صورتُه الـ`JSON` بعد فرز خياراته** — كاملةٌ وحتميّة، فصنفان يختلفان
// في أيّ حقلٍ يختلفان في المفتاح.
func fpItemKey(it fpItem) string {
	b, _ := json.Marshal(it)
	return string(b)
}

// canonHash يُرقِّم صورةً قانونيّةً بـ`SHA-256`.
//
// **و`json.Marshal` على `struct` حتميّة**: ترتيبُ الحقول ثابتٌ بترتيب
// التعريف، **فبصمةُ الصورة نفسِها هي هي في كلّ مرّة.**
func canonHash(v any) []byte {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return sum[:]
}
