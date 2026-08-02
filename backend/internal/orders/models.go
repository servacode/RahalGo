package orders

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	ErrMerchantClosed = httpx.NewError(http.StatusConflict, "merchant_closed", "errors.merchant_closed")
	// ErrMultiSource أصنافٌ من مصدرين — **حتى يُبنى الطلبُ متعدّدُ المصادر.**
	ErrMultiSource        = httpx.NewError(http.StatusConflict, "multi_source_order", "errors.multi_source_order")
	ErrItemUnavailable    = httpx.NewError(http.StatusConflict, "item_unavailable", "errors.item_unavailable")
	ErrBadItems           = httpx.NewError(http.StatusBadRequest, "invalid_items", "errors.validation")
	ErrOutOfZone          = httpx.NewError(http.StatusBadRequest, "out_of_zone", "errors.out_of_zone")
	ErrBelowMinOrder      = httpx.NewError(http.StatusBadRequest, "below_min_order", "errors.below_min_order")
	ErrWhatsAppRequired   = httpx.NewError(http.StatusForbidden, "whatsapp_required", "errors.whatsapp_required")
	ErrInvalidPromo       = httpx.NewError(http.StatusBadRequest, "invalid_promo", "errors.invalid_promo")
	ErrBadTransition      = httpx.NewError(http.StatusConflict, "invalid_transition", "errors.invalid_transition")
	ErrNeedsDriver        = httpx.NewError(http.StatusConflict, "driver_required", "errors.driver_required")
	ErrCancelWindowPassed = httpx.NewError(http.StatusConflict, "cancel_window_passed", "errors.cancel_window_passed")
)

type OptionSnapshot struct {
	// معرّف الخيار — يُحفظ لتصحّ **إعادة الطلب** بخياراته كما كان.
	// كانت اللقطة تحفظ الاسم والفرق فقط، فتعذّر إعادةُ صنفٍ له خيارات إلزامية:
	// لا سبيل لاستنتاج المعرّف من الاسم، وقد يتغيّر الاسم أو يتكرّر.
	// (طلباتٌ قديمة بلا معرّف تبقى صالحةً للعرض وتُستثنى من إعادة الطلب.)
	ID         string `json:"id,omitempty"`
	Group      string `json:"group"`
	Name       string `json:"name"`
	PriceDelta int64  `json:"price_delta"`
}

type OrderItem struct {
	ID         string  `json:"id"`
	MenuItemID *string `json:"menu_item_id"`
	Name       string  `json:"name"`
	UnitPrice  int64   `json:"unit_price"`
	// MerchantPrice سعرُ الشراء لحظةَ الطلب — **لقطةٌ لا قراءةٌ لاحقة**.
	//
	// يرفع المتجرُ سعرَه غداً **فتُعاد قراءةُ طلبات الأمس بتكلفةٍ لم تقع**،
	// فيبدو هامشُنا أصغرَ أو أكبرَ ممّا كان. **وتقريرُ ربحٍ يقرأ أسعارَ اليوم
	// على طلبات الأمس تقريرٌ يكذب بلا أن يخطئ أحد.**
	//
	// **ولا يُرسل إلى الزبون** — انظر `order_breakdown.go`.
	MerchantPrice int64            `json:"-"`
	Qty           int              `json:"qty"`
	Note          string           `json:"note"`
	Options       []OptionSnapshot `json:"options"`
}

type Event struct {
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	ActorID    *string   `json:"actor_id"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
}

type Order struct {
	ID            string `json:"id"`
	Number        int64  `json:"number"`
	CustomerID    string `json:"customer_id"`
	CustomerPhone string `json:"customer_phone"`
	CustomerName  string `json:"customer_name"`
	MerchantID    string `json:"merchant_id"`
	MerchantName  string `json:"merchant_name"`
	// حلقة المطبخ: كم دقيقة قال المتجر، ومتى أعلن الجاهزية فعلاً
	PrepMinutes *int       `json:"prep_minutes"`
	ReadyAt     *time.Time `json:"ready_at"`
	AcceptedAt  *time.Time `json:"accepted_at"`
	DeliveredAt *time.Time `json:"delivered_at"`
	// شعار المتجر وملخّص الأصناف — لبطاقة الطلب في القوائم
	MerchantLogoThumb *string `json:"merchant_logo_thumb_url"`
	ItemsCount        int     `json:"items_count"`
	ItemsPreview      string  `json:"items_preview"`
	DriverID          *string `json:"driver_id"`
	DriverPhone       *string `json:"driver_phone"`
	DriverName        *string `json:"driver_name"`
	// DriverFee أجرُ السائق عن هذا الطلب.
	//
	// **قبل التسليم تقديرٌ وبعده واقع**: يُحسب من رسم التوصيل وإعدادات الحصّة
	// حتى يُقيَّد فعلاً في الدفتر. وغرفةُ العمليات تحتاج الرقم قبل التسليم لا
	// بعده — فهي تقرّر به الإسناد.
	DriverFee     int64   `json:"driver_fee"`
	Status        string  `json:"status"`
	AddressText   string  `json:"address_text"`
	Lat           float64 `json:"lat"`
	Lng           float64 `json:"lng"`
	ZoneID        *string `json:"zone_id"`
	ZoneName      *string `json:"zone_name"`
	PaymentMethod string  `json:"payment_method"`
	Subtotal      int64   `json:"subtotal"`
	DeliveryFee   int64   `json:"delivery_fee"`
	Discount      int64   `json:"discount"`
	Total         int64   `json:"total"`
	WalletPaid    int64   `json:"wallet_paid"`
	CashDue       int64   `json:"cash_due"`
	PromoCode     *string `json:"promo_code"`
	Notes         string  `json:"notes"`
	CancelReason  string  `json:"cancel_reason"`
	// SentToMerchantAt متى حُوِّل الطلب إلى المتجر — لا «متى وصله».
	//
	// **كان يُكتب ولا يُقرأ**: تكتبه نقطةُ الإبلاغ في القاعدة ولا يعود في
	// الردّ، فوسمُ «حُوِّل» في اللوحة لا يظهر أبداً — **ويُحوَّل الطلبُ مرّتين
	// فيُطبخ مرّتين.** حقلٌ يُكتب ولا يُقرأ ليس حقلاً، هو نيّة.
	SentToMerchantAt *time.Time `json:"sent_to_merchant_at"`
	// DispatchedAt متى نزل إلى طابور السائقين — **ومنه تُقاس مهلةُ ظهور
	// زرّ الإسناد اليدويّ**، فلا تعتمد الشاشةُ على `updated_at` الذي يتغيّر
	// مع كل مسّ.
	DispatchedAt *time.Time `json:"dispatched_at"`
	// EndedBy الدورُ الذي أنهى الطلب: customer · merchant · ops · driver.
	//
	// **كان يُكتب ولا يُقرأ**: ترى العملياتُ «ملغي» ولا تعرف من ألغاه —
	// **وثلاثةُ أخبارٍ يُخفيها لفظٌ واحد**، وأحدُها يستوجب اتّصالاً بالمتجر
	// والآخر لا يستوجب شيئاً.
	EndedBy string `json:"ended_by"`
	// Fault من تسبّب في الفشل — **غيرُ `EndedBy`**: ذاك من ضغط وهذا من تسبّب.
	Fault string `json:"fault"`
	// FailReason رمزُ سبب التعذّر المُصنَّف
	FailReason string `json:"fail_reason"`
	// ReturnedAt متى أُعيدت البضاعةُ إلى متجرها
	ReturnedAt *time.Time `json:"returned_at"`
	// GoodsSettledTo مصيرُ بضاعة طلبٍ فشل: merchant استردّها · platform
	// تحمّلتها المنصةُ ودفعت للمتجر · فارغٌ يعني **لم يُحسم بعد**.
	GoodsSettledTo *string     `json:"goods_settled_to"`
	Items          []OrderItem `json:"items,omitempty"`
	Events         []Event     `json:"events,omitempty"`
	Rating         *Rating     `json:"rating,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
}

type OrderPage struct {
	Orders  []Order `json:"orders"`
	Total   int     `json:"total"`
	Page    int     `json:"page"`
	PerPage int     `json:"per_page"`
}

// CreateInput مدخلات إنشاء الطلب — الأسعار تُحسب في الخادم حصراً، لا تُقبل من العميل.
type CreateInput struct {
	CustomerPhone string      `json:"customer_phone"` // للطلب الهاتفي بالنيابة
	CustomerID    string      `json:"customer_id"`    // أو معرف مباشر
	MerchantID    string      `json:"merchant_id"`
	Items         []ItemInput `json:"items"`
	AddressText   string      `json:"address_text"`
	Lat           float64     `json:"lat"`
	Lng           float64     `json:"lng"`
	PaymentMethod string      `json:"payment_method"` // cash | wallet
	PromoCode     string      `json:"promo_code"`
	Notes         string      `json:"notes"`
}

type ItemInput struct {
	MenuItemID string   `json:"menu_item_id"`
	Qty        int      `json:"qty"`
	Note       string   `json:"note"`
	OptionIDs  []string `json:"option_ids"`
}

func marshalOptions(opts []OptionSnapshot) []byte {
	b, _ := json.Marshal(opts)
	return b
}
