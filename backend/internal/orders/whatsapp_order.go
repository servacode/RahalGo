package orders

// ══════════════════════════════════════════════════════════════════════
// **طلباتُ الواتساب** — قرارُ المالك ٢٠٢٦-١١-٠٤ ثمّ ٢٠٢٦-١١-٠٥
// ══════════════════════════════════════════════════════════════════════
//
// «عندي طلبات خارجية على الواتس اب، بدي خيار باللوحة… نكتب تفاصيل الطلب، اسم المتجر، اسم
// المتسلم وموقعه على الخريطة، وينزل طلب نظامي ع التطبيق، كأنه طلب جاي من تطبيق».
//
// **والغايةُ أن يصير كلُّ طلبٍ طلباً في المنصّة** — فمشوارٌ بلا طلبٍ يظهر في مراقبة السائقين.
//
// # ولا نوعَ جديد — «لدي توصيلة» بعلامتها
//
// `kind='merchant_delivery'` بوضع «توصيل» (`send`) والمستلِمُ يدفع الأجرة (`recipient`) نقداً،
// **وعلامتُه `origin='whatsapp'`**. فكلُّ قارئٍ قائمٍ يعرف ما يفعل: تطبيقُ السائق (القديمُ
// كذلك)، والتسويةُ، وحرّاسُ الدفتر، والتوزيعُ بالقرب، والبابُ، والإثباتُ، وكودُ التسليم.
//
// # وقراراتُ المالك الخمسة (٢٠٢٦-١١-٠٥)
//
//	١ المتجر    من متاجرنا أو متجرٌ غيرُ تابعٍ يُكتب اسمُه ويُحدَّد على الخريطة
//	٢ المال     «مثل الطلب العادي»: السائقُ يدفع للمتجر ويقبض من المستلِم، والمسجَّلُ الأجرةُ
//	            وحدَها — حصّتُه لمحفظته وحصّةُ المنصّة كالعادة، ولا عمولةَ على متجرٍ غير تابع
//	٣ الأجرة    تُحسب كالتوصيلات (بالمسافة) ويعدّلها الموظّف
//	٤ من ينشئه  الأدمنُ والموظّفون (`orders.whatsapp`)
//	٥ الموافقة  بلا موافقة — ينزل للسائقين مباشرةً
//
// # والمتجرُ منسوخٌ لا مربوط
//
// **لا `merchant_id` على طلب الواتساب أبداً** — ربطُه يُشعل آثارَ المتجر كلَّها: «توصيلاتي»
// عنده، وإلغاؤه، وإشعاراتُه، وتحميلُه فشلَ المستلِم، وعرضُ توصيله المجّانيّ. **فاسمُه ورقمُه
// وعنوانُه يُنسخون على الطلب** (`pickup_*`)، **ونقطتُه في `pickup_override`**، و
// `pickup_merchant_id` أثرٌ وحدَه لمتجرٍ اختير من متاجرنا.

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// أصلُ الطلب — `orders.origin`.
const (
	// OriginApp **طلبٌ من تطبيقٍ أو موقع** — الافتراض.
	OriginApp = "app"
	// OriginWhatsApp **طلبٌ كتبه الموظّفُ من رسالة واتساب** (قرارُ المالك ٢٠٢٦-١١-٠٥).
	OriginWhatsApp = "whatsapp"
)

// MaxWhatsAppDetails **حدُّ «تفاصيل الطلب»** — قائمةُ مشترياتٍ لا وصفُ كيس، **فمئتا «لدي
// توصيلة» لا تكفيها.** وهو حدُّ ملاحظات الطلب في البوّابة (`maxOrderNotes`).
const MaxWhatsAppDetails = 500

// **أخطاءُ طلب الواتساب علاماتٌ لا ردود** — الردُّ في بابه (`server/whatsapp_order_handlers.go`):
// البابُ للّوحة وحدَها، **فرموزُه رموزُ لوحةٍ لا تبلغ هاتفاً.**
var (
	// ErrWAStoreRequired **لا متجر**: لا مختارٌ من متاجرنا ولا اسمٌ ونقطةٌ لمتجرٍ غير تابع.
	ErrWAStoreRequired = errors.New("wa_store_required")
	// ErrWAStoreConflict **متجران**: مختارٌ ومكتوبٌ معاً.
	ErrWAStoreConflict = errors.New("wa_store_conflict")
	// ErrWAStoreUnavailable **المتجرُ المختارُ غيرُ فعّالٍ أو بلا موقع.**
	ErrWAStoreUnavailable = errors.New("wa_store_unavailable")
	// ErrWAFeeBelowShare **أجرةٌ لا تزيد على حصّة المنصّة الثابتة** — السائقُ لا يأخذ شيئاً.
	ErrWAFeeBelowShare = errors.New("wa_fee_below_share")
	// ErrWANotWhatsApp **الإلغاءُ من هذا الباب لطلب الواتساب وحدَه.**
	ErrWANotWhatsApp = errors.New("wa_not_whatsapp")
	// ErrWACancelAfterPickup **السائقُ استلم** — الحلُّ من باب الزبون لا من الإلغاء.
	ErrWACancelAfterPickup = errors.New("wa_cancel_after_pickup")
	// ErrWAFeeRequired **طلبٌ بلا موقعٍ للمستلِم بلا أجرةٍ مكتوبة** — لا مسافةَ تُحسب منها
	// (ملحقُ المالك ٢٠٢٦-١١-٠٥، ب: «الأجرة يكتبها الموظّف»).
	ErrWAFeeRequired = errors.New("wa_fee_required")
	// ErrWAAddressClosed **تعديلُ العنوان بعد التسليم أو الإقفال** — الطلبُ انتهى.
	ErrWAAddressClosed = errors.New("wa_address_closed")
	// ErrWAAmanahSender **أمانةٌ بلا رقم صاحبها** (قرارُ المالك ٢٠٢٦-١١-١١) — منه يُطلب موقعُ الاستلام.
	ErrWAAmanahSender = errors.New("wa_amanah_sender_required")
	// ErrWAAmanahStore **أمانةٌ ومتجرٌ معاً** — الاستلامُ من شخصٍ لا من محلّ.
	ErrWAAmanahStore = errors.New("wa_amanah_store")
	// ErrWASamePhone **صاحبُ الأمانة والمستلِمُ رقمٌ واحد** — ردُّه بالموقع لا يُعرف لأيّهما.
	ErrWASamePhone = errors.New("wa_same_phone")
	// ErrWAPickupPending **أمانةٌ لم يصل موقعُ صاحبها** — لا تنزل للسائقين قبل أن تُعرف نقطةُ الاستلام.
	// **ويبلغ من أبواب الانتقال العامّة** (المكتبُ ينزل الطلبَ بيده) — فردُّه في `respondErr` نفسِه.
	ErrWAPickupPending = errors.New("wa_pickup_pending")
)

// WAAmanahSenderName **اسمُ صاحب الأمانة إن لم يُكتب** — والقيدُ يشترط اسماً للاستلام.
const WAAmanahSenderName = "صاحب الأمانة"

// WAFeeRangeError **أجرةٌ معدَّلةٌ خارجَ حدَّيها** — `delivery.custom_fee_min/max`، والحدّان
// يُقالان في الردّ.
type WAFeeRangeError struct{ Min, Max int64 }

func (e *WAFeeRangeError) Error() string { return "wa_fee_out_of_range" }

// WhatsAppOrderInput **ما يكتبه الموظّفُ من رسالة الواتساب.**
//
// **والمتجرُ واحدٌ من اثنين**: `StoreID` (من متاجرنا) **أو** `StoreName` مع نقطته.
type WhatsAppOrderInput struct {
	StoreID string
	// StoreName/StorePhone/StoreAddress **متجرٌ غيرُ تابع** — ورقمُ المتجر المختار يُعدَّل هنا أيضاً.
	StoreName, StorePhone, StoreAddress string
	StoreLat, StoreLng                  *float64

	// Details **تفاصيلُ الطلب** — يقرؤها السائقُ قبل أن يقبل (`parcel_note`).
	Details string
	// GoodsAmount **ثمنُ البضاعة إن عُرف** — يدفعه السائقُ للمتجر ويقبضه من المستلِم،
	// **خارجَ الدفتر** (`collect_amount`). وnil: الحسابُ حسبَ الفاتورة.
	GoodsAmount *int64
	// Currency «SYP» أو «USD» (والدولارُ بالسنتات) — وفارغٌ ليرة.
	Currency string

	RecipientName, RecipientPhone, AddressText string
	// Lat/Lng **موقعُ المستلِم على الخريطة** — **أو لا شيء** (ملحقُ المالك ٢٠٢٦-١١-٠٥، ب: «الطلب
	// ما بيوقف على الموقع»): يُنشأ بالعنوان المكتوب، **والتغطيةُ والدوامُ من نقطة المتجر ومنطقته،
	// والأجرةُ يكتبها الموظّف** (`Fee` إلزاميّة). والموقعُ إن وصل بعدها كُتب في الطلب
	// (`ApplyWhatsAppShared`).
	Lat, Lng   *float64
	DriverNote string
	// Fee **أجرةٌ عدّلها الموظّف** — وnil: المحسوبة. **وبلا موقعٍ للمستلِم إلزاميّة.**
	Fee *int64

	// Amanah **«أمانة من شخص»** (قرارُ المالك ٢٠٢٦-١١-١١) — الاستلامُ من صاحب الأمانة لا من متجر:
	// اسمُه ورقمُه مكانَ المتجر، **ونقطتُه اختياريّة** (وبلاها يطلبها البوتُ منه). **ولا ثمنَ بضاعة.**
	Amanah               bool
	SenderName           string
	SenderPhone          string
	SenderLat, SenderLng *float64
}

// hasPoint **أمعه موقعُ المستلِم؟** — والنصفُ (خطُّ عرضٍ بلا طول) خطأٌ لا «بلا موقع».
func (in WhatsAppOrderInput) hasPoint() (bool, error) {
	switch {
	case in.Lat == nil && in.Lng == nil:
		return false, nil
	case in.Lat == nil || in.Lng == nil || !ValidPoint(*in.Lat, *in.Lng):
		return false, ErrBadPoint
	}
	return true, nil
}

// WhatsAppQuote **أجرةُ طلب الواتساب قبل الإرسال** — بحرّاس الإنشاء ودالّته نفسِها.
type WhatsAppQuote struct {
	Fee       int64  `json:"fee"`
	FeeBasis  string `json:"fee_basis"`
	DistanceM int64  `json:"distance_m"`
	ZoneName  string `json:"zone_name"`
	// SharePercent/ShareFixed **حصّةُ المنصّة** — حصّةُ «طلب واتساب» (`delivery.share_whatsapp`).
	SharePercent  int64 `json:"share_percent"`
	ShareFixed    int64 `json:"share_fixed"`
	DriverFee     int64 `json:"driver_fee"`
	PlatformShare int64 `json:"platform_share"`
	// FeeMin/FeeMax **حدّا تعديل الأجرة** — حدّا أجرة الطلب الخاصّ.
	FeeMin int64 `json:"fee_min"`
	FeeMax int64 `json:"fee_max"`
	// HasPoint **أحُسبت من موقع المستلِم؟** — وبلاه `Fee` أساسُ الأجرة اقتراحاً، **و`FeeRequired`
	// يقول للنموذج أن يكتبها الموظّف** (ملحقُ المالك ٢٠٢٦-١١-٠٥، ب).
	HasPoint    bool `json:"has_point"`
	FeeRequired bool `json:"fee_required"`
}

// WhatsAppCreated **ما وقع في الإنشاء** — للردّ وللتدقيق.
type WhatsAppCreated struct {
	ID          string
	Number      int64
	ComputedFee int64
	Fee         int64
	DriverFee   int64
	StoreName   string
	// PickupMerchantID **المتجرُ المختارُ من متاجرنا** — وفارغٌ لمتجرٍ غير تابع.
	PickupMerchantID string
	// HasPoint **أُنشئ بموقع المستلِم** — وبلاه بالعنوان المكتوب (ملحقُ المالك ٢٠٢٦-١١-٠٥، ب).
	HasPoint bool
	// Amanah/PickupKnown **أمانةٌ، وأعُرفت نقطةُ صاحبها؟** — وبلاها يولد «مقبولاً» لا في الطابور.
	Amanah      bool
	PickupKnown bool
}

// waStore **المتجرُ كما يُكتب على الطلب.**
type waStore struct {
	merchantID           string
	name, phone, address string
	lat, lng             float64
	// known **أللمتجر نقطةٌ؟** — متجرٌ غيرُ تابعٍ يُكتب اسمُه وحدَه (قرارُ المالك ٢٠٢٦-١١-٠٦: «بس اسم
	// المحل، بلا رقم ولا موقع») فلا نقطةَ له، **ومتجرُنا نقطتُه دبّوسُه.**
	known bool
}

// resolveWhatsAppStore **المتجرُ من متاجرنا (يُنسخ) أو المكتوبُ (كما هو).**
//
// **و`needName` في الإنشاء وحدَه** — عرضُ السعر يكفيه موقعُ المتجر المكتوب.
func (s *Service) resolveWhatsAppStore(ctx context.Context, q dbtx.Querier,
	in WhatsAppOrderInput, needName bool) (waStore, error) {
	typed := in.StoreName != "" || in.StoreLat != nil || in.StoreLng != nil
	if in.StoreID != "" && typed {
		return waStore{}, ErrWAStoreConflict
	}
	if in.StoreID == "" {
		if needName && in.StoreName == "" {
			return waStore{}, ErrWAStoreRequired
		}
		st := waStore{name: in.StoreName, phone: in.StorePhone, address: in.StoreAddress}
		// **والنقطةُ اختياريّةٌ للمتجر غير التابع** (قرارُ المالك ٢٠٢٦-١١-٠٦) — والنصفُ خطأ.
		switch {
		case in.StoreLat == nil && in.StoreLng == nil:
		case in.StoreLat == nil || in.StoreLng == nil || !ValidPoint(*in.StoreLat, *in.StoreLng):
			return waStore{}, ErrBadPoint
		default:
			st.lat, st.lng, st.known = *in.StoreLat, *in.StoreLng, true
		}
		return st, nil
	}
	if _, err := uuid.Parse(in.StoreID); err != nil {
		return waStore{}, ErrWAStoreUnavailable
	}
	var st waStore
	var status string
	var lat, lng *float64
	err := q.QueryRow(ctx, `
		SELECT m.id::text, m.name,
		       -- **رقمُ المتجر ثمّ رقمُ صاحبه** — زرُّ الاتّصال عند السائق.
		       COALESCE(NULLIF(m.phone::text, ''), NULLIF(u.phone::text, ''), ''),
		       m.address_text, m.status,
		       ST_Y(m.location::geometry), ST_X(m.location::geometry)
		  FROM merchants m LEFT JOIN users u ON u.id = m.owner_user_id
		 WHERE m.id = $1`, in.StoreID).
		Scan(&st.merchantID, &st.name, &st.phone, &st.address, &status, &lat, &lng)
	if errors.Is(err, pgx.ErrNoRows) {
		return waStore{}, ErrWAStoreUnavailable
	}
	if err != nil {
		return waStore{}, err
	}
	// **متجرٌ موقوفٌ أو بلا دبّوسٍ لا يُرسَل إليه سائق** — والمسكَّرُ الآن يُقبل (الموظّفُ يعرف).
	if status != "active" || lat == nil || lng == nil {
		return waStore{}, ErrWAStoreUnavailable
	}
	st.lat, st.lng, st.known = *lat, *lng, true
	// **ورقمُ المتجر يُعدَّل في النموذج** — الموظّفُ يعرف من يردّ.
	if in.StorePhone != "" {
		st.phone = in.StorePhone
	}
	return st, nil
}

// resolveWhatsAppSender **صاحبُ الأمانة مكانَ المتجر** (قرارُ المالك ٢٠٢٦-١١-١١) — اسمُه ورقمُه
// يُنسخان على الطلب كما يُنسخ المتجرُ غيرُ التابع، **فيقرؤهما تطبيقُ السائق حيث اعتاد** («استلام من»
// وزرُّ الاتّصال). **ونقطتُه اختياريّة**: بلاها يطلبها البوتُ منه، والطلبُ لا ينزل حتّى تصل.
//
// **ولا متجرَ مع الأمانة**، ولا رقمَ واحدٌ للطرفين — ردُّه بالموقع لا يُعرف لأيّهما (`HandleWALocation`
// يملأ أحدثَ طلبٍ مفتوحٍ لرقم المرسل).
func resolveWhatsAppSender(in WhatsAppOrderInput) (waStore, error) {
	if in.StoreID != "" || in.StoreName != "" || in.StorePhone != "" || in.StoreAddress != "" ||
		in.StoreLat != nil || in.StoreLng != nil {
		return waStore{}, ErrWAAmanahStore
	}
	if in.SenderPhone == "" {
		return waStore{}, ErrWAAmanahSender
	}
	if in.SenderPhone == in.RecipientPhone {
		return waStore{}, ErrWASamePhone
	}
	st := waStore{name: in.SenderName, phone: in.SenderPhone}
	if st.name == "" {
		st.name = WAAmanahSenderName
	}
	switch {
	case in.SenderLat == nil && in.SenderLng == nil:
	case in.SenderLat == nil || in.SenderLng == nil || !ValidPoint(*in.SenderLat, *in.SenderLng):
		return waStore{}, ErrBadPoint
	default:
		st.lat, st.lng, st.known = *in.SenderLat, *in.SenderLng, true
	}
	return st, nil
}

// waPricing **أجرةُ النقطتين وحصّةُ المنصّة** — يقرؤها عرضُ السعر والإنشاءُ معاً.
type waPricing struct {
	zone       ZoneCharge
	fee        int64
	distanceM  float64
	basis      string
	pct, fixed int64
	// computed **أحُسبت الأجرةُ من المسافة؟** — بنقطتي المتجر والمستلِم معاً. **وبلاها `fee` أساسُ
	// القاعدة اقتراحاً لا أجرة**، والموظّفُ يكتبها.
	computed bool
	// anchorLat/anchorLng **نقطةٌ تُحكم منها التغطيةُ والمنطقة** — المستلِمُ ثمّ المتجرُ ثمّ مركزُ أوّل
	// منطقةٍ فعّالة (`waCityAnchor`). وهي نقطةُ التسليم «غيرُ المعروفة» حتّى يصل الموقع.
	anchorLat, anchorLng float64
}

// waCityAnchor **مركزُ أوّل منطقةٍ فعّالة** — لطلبٍ لا نقطةَ فيه أصلاً: متجرٌ غيرُ تابعٍ باسمه
// وحدَه (قرارُ المالك ٢٠٢٦-١١-٠٦) **وموقعُ المستلِم لم يصل بعد.** منه تُحكم التغطيةُ والدوام،
// **ولا يُوجَّه إليه سائق** (`dropoff_known = false`، ولا `pickup_override`). وبلا منطقةٍ فعّالة
// لا تغطية (`coverage_unavailable`).
func waCityAnchor(ctx context.Context, q dbtx.Querier) (lat, lng float64, err error) {
	err = q.QueryRow(ctx, `
		SELECT ST_Y(center::geometry), ST_X(center::geometry)
		  FROM delivery_zones
		 WHERE active AND center IS NOT NULL
		 ORDER BY sort_order, id
		 LIMIT 1`).Scan(&lat, &lng)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, ErrCoverageUnavailable
	}
	return lat, lng, err
}

// whatsAppPricing **حرّاسُ التغطية ثمّ الأجرة** — كلُّ نقطةٍ معروفةٍ داخلَ التغطية، **والمنطقةُ
// مفتوحةٌ** (حرّاسُ «لدي توصيلة» نفسُها، ولا رابع).
//
// # والنقاطُ المعروفة (قرارُ المالك ٢٠٢٦-١١-٠٦)
//
// **المنطقةُ منطقةُ المستلِم إن عُرف موقعُه، وإلّا منطقةُ المتجر، وإلّا منطقةُ مركز المدينة**
// (`waCityAnchor`). **والأجرةُ بالمسافة إن عُرفت النقطتان معاً**، وإلّا أساسُ القاعدة اقتراحاً
// (`no_point`) — يكتبها الموظّف.
func (s *Service) whatsAppPricing(ctx context.Context, q dbtx.Querier,
	store waStore, lat, lng float64, hasPoint bool) (waPricing, error) {
	p := waPricing{}
	if (store.known && !ValidPoint(store.lat, store.lng)) || (hasPoint && !ValidPoint(lat, lng)) {
		return p, ErrBadPoint
	}
	var zone ZoneCharge
	var err error
	// **والمتجرُ داخلَ التغطية** — لا يُرسَل سائقٌ إلى خارجها.
	if store.known {
		if err := s.requirePlaceLaunched(ctx, q, store.lat, store.lng); err != nil {
			return p, err
		}
		if zone, err = s.RequireServiceable(ctx, q, store.lat, store.lng); err != nil {
			return p, err
		}
		p.anchorLat, p.anchorLng = store.lat, store.lng
	}
	if hasPoint {
		if err := s.requirePlaceLaunched(ctx, q, lat, lng); err != nil {
			return p, err
		}
		if zone, err = s.RequireServiceable(ctx, q, lat, lng); err != nil {
			return p, err
		}
		p.anchorLat, p.anchorLng = lat, lng
	}
	if !store.known && !hasPoint {
		if p.anchorLat, p.anchorLng, err = waCityAnchor(ctx, q); err != nil {
			return p, err
		}
		if zone, err = s.RequireServiceable(ctx, q, p.anchorLat, p.anchorLng); err != nil {
			return p, err
		}
	}
	if err := s.requireZoneOpen(ctx, q, zone); err != nil {
		return p, err
	}
	p.zone = zone
	if store.known && hasPoint {
		if p.fee, p.distanceM, err = s.whatsAppFee(ctx, q, store.lat, store.lng, lat, lng); err != nil {
			return p, err
		}
		p.basis, p.computed = FeeBasisDistance, true
	} else {
		p.fee, p.basis = s.whatsAppFeeRule(ctx, q).Base, FeeBasisNoPoint
	}
	// **وحصّةُ المنصّة حصّةُ «طلب واتساب»** (قرارُ المالك ٢٠٢٦-١١-٠٥: «مثل الاجرة وحصة المنصة»).
	p.pct, p.fixed = s.platformShare(ctx, q, ShareWhatsApp)
	return p, nil
}

// whatsAppFeeRule **قاعدةُ أجرة «طلب واتساب»** — مفاتيحُه هو (`pricing.WhatsAppFeeKeys`، قرارُ المالك
// ٢٠٢٦-١١-٠٥)، **لا مفاتيحُ «لدي توصيلة» ولا أجرتُها الثابتة.** وبلا مخزنٍ صفر.
func (s *Service) whatsAppFeeRule(ctx context.Context, q dbtx.Querier) pricing.FeeRule {
	if s.settings == nil {
		return pricing.FeeRule{}
	}
	return pricing.RuleOf(ctx, s.settings.On(q), pricing.WhatsAppFeeKeys)
}

// whatsAppFee **الأجرةُ كالتوصيلات** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٣) — قاعدةُ «طلب واتساب»
// (`whatsAppFeeRule`) على المسافة المستقيمة **من المتجر إلى المستلِم**، والمسافةُ تُقال معها.
func (s *Service) whatsAppFee(ctx context.Context, q dbtx.Querier,
	storeLat, storeLng, lat, lng float64) (fee int64, distanceM float64, err error) {
	if err = q.QueryRow(ctx, `
		SELECT ST_Distance(ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
		                   ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography)`,
		storeLat, storeLng, lat, lng).Scan(&distanceM); err != nil {
		return 0, 0, err
	}
	return s.whatsAppFeeRule(ctx, q).At(distanceM), distanceM, nil
}

// whatsAppFeeBounds **حدّا تعديل الأجرة** — حدّا أجرة الطلب الخاصّ (`delivery.custom_fee_min/max`).
func (s *Service) whatsAppFeeBounds(ctx context.Context, q dbtx.Querier) (lo, hi int64) {
	if s.settings == nil {
		return 0, 0
	}
	st := s.settings.On(q)
	return st.GetInt(ctx, "delivery.custom_fee_min"), st.GetInt(ctx, "delivery.custom_fee_max")
}

// QuoteWhatsAppOrder **أجرةُ الطلب قبل الإرسال** — المتجرُ بمعرّفه أو بنقطته، **والمستلِمُ بنقطته
// أو بلاها** (`lat/lng` nil معاً: الأساسُ اقتراحاً والأجرةُ يكتبها الموظّف).
func (s *Service) QuoteWhatsAppOrder(ctx context.Context, storeID string,
	storeLat, storeLng, lat, lng *float64) (*WhatsAppQuote, error) {
	in := WhatsAppOrderInput{StoreID: strings.TrimSpace(storeID), StoreLat: storeLat, StoreLng: storeLng,
		Lat: lat, Lng: lng}
	hasPoint, err := in.hasPoint()
	if err != nil {
		return nil, err
	}
	st, err := s.resolveWhatsAppStore(ctx, s.db, in, false)
	if err != nil {
		return nil, err
	}
	var dLat, dLng float64
	if hasPoint {
		dLat, dLng = *lat, *lng
	}
	p, err := s.whatsAppPricing(ctx, s.db, st, dLat, dLng, hasPoint)
	if err != nil {
		return nil, err
	}
	driverFee := DriverFeeWithShare(p.fee, p.pct, p.fixed)
	lo, hi := s.whatsAppFeeBounds(ctx, s.db)
	return &WhatsAppQuote{
		Fee: p.fee, FeeBasis: p.basis, DistanceM: int64(p.distanceM), ZoneName: p.zone.Name,
		SharePercent: p.pct, ShareFixed: p.fixed,
		DriverFee: driverFee, PlatformShare: p.fee - driverFee,
		FeeMin: lo, FeeMax: hi,
		HasPoint: hasPoint, FeeRequired: !p.computed,
	}, nil
}

// CreateWhatsAppOrderIn **يكتب طلبَ الواتساب في معاملة من يناديه** — ولا يُثبّت.
//
// **ليُثبَّت مع علامة منع التكرار والتدقيق في معاملةٍ واحدة** (`WithIdempotentTx`) — طلبٌ
// أُنشئ وضاع ردُّه فأُعيد لا يُنشأ ثانيةً، **ولا طلبٌ بلا أثرِ من كتبه.**
//
// # ويولد في الطابور (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٥: «بلا موافقة»)
//
// `dispatching` بقبولٍ ونزولٍ في لحظته — **فالموظّفُ الذي كتبه هو المكتب.** والعرضُ الأوّلُ
// بعد التثبيت (`AfterWhatsAppOrder`).
//
// # والمالُ عند التسليم وحدَه
//
// `subtotal 0` · `delivery_fee = total = cash_due = F` · `driver_fee = F − حصّة المنصّة` —
// **وثمنُ البضاعة في `collect_amount` عرضٌ لا قيد.** ولا عرضَ توصيلٍ مجّانيّ ولا كود.
func (s *Service) CreateWhatsAppOrderIn(ctx context.Context, tx dbtx.Querier,
	actorID string, in WhatsAppOrderInput) (*WhatsAppCreated, error) {
	in.StoreID = strings.TrimSpace(in.StoreID)
	in.StoreName = strings.TrimSpace(in.StoreName)
	in.StorePhone = strings.TrimSpace(in.StorePhone)
	in.StoreAddress = strings.TrimSpace(in.StoreAddress)
	in.Details = strings.TrimSpace(in.Details)
	in.RecipientName = strings.TrimSpace(in.RecipientName)
	in.RecipientPhone = strings.TrimSpace(in.RecipientPhone)
	in.AddressText = strings.TrimSpace(in.AddressText)
	in.DriverNote = strings.TrimSpace(in.DriverNote)
	in.SenderName = strings.TrimSpace(in.SenderName)
	in.SenderPhone = strings.TrimSpace(in.SenderPhone)
	// **رقمُ الزبون وحدَه إلزاميّ** (قرارُ المالك ٢٠٢٦-١١-٠٦) — لا اسمَ له («زبون»)، **والعنوانُ
	// والموقعُ يطلبهما البوتُ منه** ويملآن الطلبَ حين يصلان. والتفاصيلُ ملاحظةٌ اختياريّة.
	if in.RecipientPhone == "" {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if in.RecipientName == "" {
		in.RecipientName = WhatsAppRecipientName
	}
	if len([]rune(in.Details)) > MaxWhatsAppDetails {
		in.Details = string([]rune(in.Details)[:MaxWhatsAppDetails])
	}
	// **وموقعُ المستلِم على الخريطة أو لا شيء** (ملحقُ المالك ٢٠٢٦-١١-٠٥، ب) — والطلبُ لا
	// ينتظره: بلاه العنوانُ المكتوبُ يكفي **والأجرةُ يكتبها الموظّف.**
	hasPoint, err := in.hasPoint()
	if err != nil {
		return nil, err
	}
	// **والثمنُ صفرٌ = أمانةٌ بلا ثمن** (قرارُ المالك ٢٠٢٦-١١-٠٨) — كأنّه لم يُكتب. **والسالبُ يُردّ.**
	if in.GoodsAmount != nil && *in.GoodsAmount < 0 {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if in.GoodsAmount != nil && *in.GoodsAmount == 0 {
		in.GoodsAmount = nil
	}
	if in.Currency == "" {
		in.Currency = "SYP"
	}
	if in.Currency != "SYP" && in.Currency != "USD" {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}

	// **والأمانةُ لا ثمنَ بضاعةٍ فيها** (قرارُ المالك ٢٠٢٦-١١-١١) — والصفرُ صار nil أعلاه.
	if in.Amanah && in.GoodsAmount != nil {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	var store waStore
	if in.Amanah {
		store, err = resolveWhatsAppSender(in)
	} else {
		store, err = s.resolveWhatsAppStore(ctx, tx, in, true)
	}
	if err != nil {
		return nil, err
	}
	var rLat, rLng float64
	if hasPoint {
		rLat, rLng = *in.Lat, *in.Lng
	}
	p, err := s.whatsAppPricing(ctx, tx, store, rLat, rLng, hasPoint)
	if err != nil {
		return nil, err
	}
	// **ولا أجرةَ تُحسب بلا النقطتين** — يكتبها الموظّف (ملحقُ المالك ٢٠٢٦-١١-٠٥، ب).
	if !p.computed && in.Fee == nil {
		return nil, ErrWAFeeRequired
	}
	// **ونقطةُ التسليم بلا موقعٍ «غيرُ معروفة»** — نقطةُ المتجر أو مركزُ المدينة (`anchor`)، سابقةُ
	// «لدي توصيلة» بلا نقطة: لا يُوجَّه السائقُ إليها، **ويقرأ العنوانَ** حتّى يصل الموقع.
	dropLat, dropLng := p.anchorLat, p.anchorLng
	// **ونقطةُ الاستلام نقطةُ المتجر إن عُرفت** — ومتجرٌ باسمه وحدَه بلا نقطة: يعرفه السائق.
	var pickLat, pickLng *float64
	if store.known {
		pickLat, pickLng = &store.lat, &store.lng
	}

	// ── الأجرة: المحسوبةُ أو ما عدّله الموظّف بحدّيه ─────────────────────
	//
	// **وبلا موقعٍ لا محسوبةَ أصلاً** — فالمكتوبةُ تمرّ بالحدّين دائماً، **ولا تكون صفراً**: سائقٌ
	// يُرسَل بلا أجر.
	fee := p.fee
	typed := in.Fee != nil && (*in.Fee != p.fee || !p.computed)
	if typed {
		lo, hi := s.whatsAppFeeBounds(ctx, tx)
		if *in.Fee < lo || (hi > 0 && *in.Fee > hi) || *in.Fee < 0 || (!p.computed && *in.Fee == 0) {
			return nil, &WAFeeRangeError{Min: lo, Max: hi}
		}
		fee = *in.Fee
	}
	// **وأجرةٌ معدَّلةٌ لا تزيد على الحصّة الثابتة تُرسل سائقاً بلا أجر** — تُردّ ولا تُكتب.
	if typed && p.fixed > 0 && p.fixed >= fee {
		return nil, ErrWAFeeBelowShare
	}
	driverFee := DriverFeeWithShare(fee, p.pct, p.fixed)

	// ══════════════════════════════════════════════════════════════════
	// **وأمانةٌ بلا نقطة صاحبها تولد «مقبولةً» لا في الطابور** (قرارُ المالك ٢٠٢٦-١١-١١)
	// ══════════════════════════════════════════════════════════════════
	//
	// **لا يراها سائقٌ ولا تُعرض** — العرضُ والإسنادُ من الطابور وحدَه. **وتنزل لحظةَ وصول موقعه**
	// (`ApplyWhatsAppPickup`)، **والآلةُ تحرس النزولَ قبله** (`ErrWAPickupPending`). **وثمنُ الأمانة
	// صفرٌ موثَّقٌ من الإنشاء** — فلا يُسأل السائقُ عنه بمكان الاستلام (تطبيقُه كما هو).
	status := StDispatching
	if in.Amanah && !store.known {
		status = StAccepted
	}

	snap, err := s.snapshotNow(ctx, tx)
	if err != nil {
		return nil, err
	}

	var actor, pickupMerchant *string
	if actorID != "" {
		actor = &actorID
	}
	if store.merchantID != "" {
		pickupMerchant = &store.merchantID
	}
	out := &WhatsAppCreated{ComputedFee: p.fee, Fee: fee, DriverFee: driverFee,
		StoreName: store.name, PickupMerchantID: store.merchantID, HasPoint: hasPoint,
		Amanah: in.Amanah, PickupKnown: store.known}
	if err := tx.QueryRow(ctx, `
		INSERT INTO orders (kind, origin, created_by, pickup_merchant_id,
		                    pickup_name, pickup_phone, pickup_address, pickup_override,
		                    address_text, dropoff, zone_id,
		                    recipient_name, recipient_phone, parcel_note, fee_payer,
		                    status, payment_method, subtotal, delivery_fee, driver_fee,
		                    total, cash_due, notes,
		                    snap_merchant_commission_percent, snap_rep_commission_percent,
		                    snap_commission_source, snap_activation_orders,
		                    snap_platform_delivery_percent, dropoff_known, snap_platform_delivery_fixed,
		                    delivery_mode, collect_amount, amount_currency, promo_delivery_waived,
		                    accepted_at, dispatched_at, money_model,
		                    wa_amanah, fetch_amount, fetch_amount_version, fetch_amount_status,
		                    fetch_amount_at, fetch_amount_decided_at)
		VALUES ('merchant_delivery', 'whatsapp', $1, $2,
		        $3, NULLIF($4, ''), NULLIF($5, ''),
	        CASE WHEN $6::float8 IS NULL OR $7::float8 IS NULL THEN NULL
	             ELSE ST_SetSRID(ST_MakePoint($7, $6), 4326)::geography END,
		        $8, ST_SetSRID(ST_MakePoint($10, $9), 4326)::geography, $11,
		        $12, $13, $14, 'recipient',
		        $28, 'cash', 0, $15, $16,
		        $15, $15, $17,
		        $18, $19, $20, $21,
		        $22, $26, $23,
		        'send', $24, $25, 0,
		        now(), CASE WHEN $28::text = 'dispatching' THEN now() END, $27,
		        $29, CASE WHEN $29 THEN 0 END, CASE WHEN $29 THEN 1 ELSE 0 END,
		        CASE WHEN $29 THEN 'approved' END, CASE WHEN $29 THEN now() END, CASE WHEN $29 THEN now() END)
		RETURNING id::text, number`,
		actor, pickupMerchant,
		store.name, store.phone, store.address, pickLat, pickLng,
		in.AddressText, dropLat, dropLng, nullIfEmpty(p.zone.ID),
		in.RecipientName, in.RecipientPhone, in.Details,
		fee, driverFee, in.DriverNote,
		snap.MerchantCommission, snap.RepCommissionPercent, snap.CommissionSource, snap.ActivationOrders,
		p.pct, p.fixed,
		in.GoodsAmount, in.Currency, hasPoint, s.newOrderMoneyModel(ctx, tx),
		status, in.Amanah).Scan(&out.ID, &out.Number); err != nil {
		return nil, err
	}
	// **وثمنٌ بالدولار كتبه الموظّف يُلقط سعرُه الآن** (قرارُ المالك ٢٠٢٦-١١-٠٩) — كما يُلقط حين يكتبه
	// السائق: عمولتُه عند التسليم بسعر يوم كتابته.
	if in.GoodsAmount != nil && in.Currency == CurrencyUSD {
		if r, err := s.goodsRateFor(ctx, tx, out.ID); err != nil {
			return nil, err
		} else if r > 0 {
			if _, err := tx.Exec(ctx, `UPDATE orders SET goods_fx_rate = $2 WHERE id = $1`, out.ID, r); err != nil {
				return nil, err
			}
		}
	}

	// **ولحظةُ الميلاد تُقيَّد** — والفاعلُ الموظّفُ الذي كتبه، **والحالُ الأولى الطابور** (أو «مقبول»
	// لأمانةٍ تنتظر موقعَ صاحبها).
	birthNote := whatsAppBirthNote
	if in.Amanah {
		birthNote = whatsAppAmanahBirthNote
		if !store.known {
			birthNote = whatsAppAmanahWaitNote
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
		VALUES ($1, '', $4, $2, $3)`, out.ID, actor, birthNote, status); err != nil {
		return nil, err
	}
	return out, nil
}

// WhatsAppRecipientName **اسمُ المستلِم في طلب الواتساب** — لا يُكتب له اسم (قرارُ المالك ٢٠٢٦-١١-٠٦:
// «اسم المستلم زبون»).
const WhatsAppRecipientName = "زبون"

// whatsAppBirthNote **سطرُ الميلاد في سجلّ الطلب.**
const whatsAppBirthNote = "طلب واتساب"

// whatsAppAmanahBirthNote/whatsAppAmanahWaitNote **سطرا ميلاد الأمانة** — بنقطة صاحبها أو بانتظارها.
const (
	whatsAppAmanahBirthNote = "أمانة واتساب"
	whatsAppAmanahWaitNote  = "أمانة واتساب — بانتظار موقع صاحب الأمانة"
)

// AfterWhatsAppOrder **ما لا يقع إلّا بعد التثبيت** — البثُّ والعرضُ الأوّل على السائقين.
//
// **ولا رنينَ للمكتب**: هو من كتبه (والرنينُ للمعلَّق غيرِ المرئيّ وحدَه). **والسائقون
// يُعرض عليهم كأيّ طلبٍ نزل الطابور** — بالدور (`OfferNext`) أو للقريبين (`publishOrder`).
func (s *Service) AfterWhatsAppOrder(ctx context.Context, id string) (*Order, error) {
	o, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.publishOrder(o)
	// **وأمانةٌ تنتظر موقعَ صاحبها لا تُعرض** — «مقبولةٌ» حتّى يصل (`ApplyWhatsAppPickup`).
	if o.Status != StDispatching {
		return o, nil
	}
	if err := s.OfferNext(ctx, id, nil); err != nil {
		s.logger.Error("طلب واتساب: تعذّر أوّلُ عرض", "order", id, "error", err)
	}
	return o, nil
}

// WhatsAppStore **متجرٌ من متاجرنا في بحث نموذج الواتساب** — الفعّالُ ذو الموقع وحدَه.
type WhatsAppStore struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	AddressText string  `json:"address_text"`
	Phone       string  `json:"phone"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	// OpenNow **مفتوحٌ الآن بدوامه** (`OpenNowSQL`) — والمسكَّرُ يُقال ولا يُمنع.
	OpenNow bool `json:"open_now"`
}

// WhatsAppStores **بحثٌ باسم المتجر أو عنوانه** — في متاجرنا الفعّالة ذات الموقع، **والاسمُ الذي
// يبدأ بالكلمة أوّلاً.** وحدٌّ أعلى `limit`.
func (s *Service) WhatsAppStores(ctx context.Context, q string, limit int) ([]WhatsAppStore, error) {
	q = strings.TrimSpace(q)
	out := []WhatsAppStore{}
	if len([]rune(q)) < 2 {
		return out, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	p := EscapeLike(q)
	rows, err := s.db.Query(ctx, `
		SELECT m.id::text, m.name, m.address_text,
		       COALESCE(NULLIF(m.phone::text, ''), NULLIF(u.phone::text, ''), ''),
		       ST_Y(m.location::geometry), ST_X(m.location::geometry),
		       `+OpenNowSQL+`
		  FROM merchants m LEFT JOIN users u ON u.id = m.owner_user_id
		 WHERE m.status = 'active' AND m.location IS NOT NULL
		   AND (m.name ILIKE '%' || $1 || '%' ESCAPE '\' OR m.address_text ILIKE '%' || $1 || '%' ESCAPE '\')
		 ORDER BY (m.name ILIKE $1 || '%' ESCAPE '\') DESC, m.name
		 LIMIT $2`, p, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var st WhatsAppStore
		if err := rows.Scan(&st.ID, &st.Name, &st.AddressText, &st.Phone, &st.Lat, &st.Lng,
			&st.OpenNow); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// WhatsAppOrders **آخرُ طلبات الواتساب** — **الجاريةُ أوّلاً** ثمّ المنتهية، والأحدثُ أوّلاً في
// كلٍّ، للّوحة: طلبٌ مفتوحٌ أقدمُ من آخر المنتهية يبقى في القائمة فيُصحَّح عنوانُه منها.
// **و`phone` يحصرها في رقم مستلِمٍ** (والجاريةُ وحدَها مع `openOnly`): «في طلب واتساب مفتوح
// لهالرقم» — كي لا يُرسَل سائقان إلى بابٍ واحد.
func (s *Service) WhatsAppOrders(ctx context.Context, phone string, openOnly bool, limit int) ([]Order, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	where := ` WHERE o.origin = 'whatsapp'`
	args := []any{limit}
	if phone != "" {
		args = append(args, phone)
		where += ` AND o.recipient_phone = $2`
	}
	if openOnly {
		where += ` AND o.closed_at IS NULL`
	}
	rows, err := s.db.Query(ctx, orderSelect+where+
		` ORDER BY (o.closed_at IS NULL) DESC, o.created_at DESC LIMIT $1`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

// CancelWhatsAppOrder **يُلغي موظّفُ الواتساب طلبَه قبل الاستلام** — بسببٍ إلزاميّ.
//
// **ولمن يملك `orders.whatsapp` وحدَها** — من يملك التدخّلَ يُلغي من بابه العامّ. **والآلةُ
// تحكم كما تحكم هناك** (فاعلُ عمليّات: الطابورُ وأُسند وعند المتجر)، **والتدقيقُ في
// المعاملة نفسِها** (`hook`).
func (s *Service) CancelWhatsAppOrder(ctx context.Context, actorID, orderID, reason string,
	hook func(context.Context, dbtx.Querier) error) (*Order, error) {
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, httpx.ErrNotFound
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	var origin, status string
	err := s.db.QueryRow(ctx, `SELECT origin, status FROM orders WHERE id = $1`, orderID).
		Scan(&origin, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if origin != OriginWhatsApp {
		return nil, ErrWANotWhatsApp
	}
	if pastPickup[status] || status == StDelivered {
		return nil, ErrWACancelAfterPickup
	}
	return s.TransitionAudited(ctx, actorID, []string{"ops"}, orderID, StCancelled, reason, hook)
}

// ══════════════════════════════════════════════════════════════════════
// **موقعُ المستلِم يصل بعد الإنشاء، والعنوانُ يُصحَّح من اللوحة** (ملحقُ المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// «من لوحة التحكم مثل توصيلة متجر وامانة نحط الرقم صاحب الطلب ويشارك موقعه ويجي لعندنا».
// **والطلبُ لا ينتظر الموقع** — يُنشأ بالعنوان المكتوب، وطلبُ الموقع مربوطٌ به. **فإن وصل
// (بالرابط أو بالردّ على الواتساب) كُتب نقطةَ تسليمٍ وبُثّ للسائق في لحظته**، ولا تُمسّ الأجرة.
// **وإن لم يصل** فالسائقُ يعتمد العنوانَ المكتوب، **والمكتبُ يصحّحه** (`UpdateWhatsAppAddress`).
//
// **ولا رقمَ للمستلِم يبلغ السائقَ في الحالين** (البند ٧) — لا في الإشعار ولا في الحمولة.

// waLocationArrivedTitle/waAddressChangedTitle **ما يقرؤه السائقُ في إشعاره.**
const (
	waLocationArrivedTitle = "وصل موقع المستلم — صار على الخريطة"
	waAddressChangedTitle  = "تغيّر عنوان المستلم"
	waLocationOutsideTitle = "موقع مستلم طلب واتساب برّا التغطية"
)

// ApplyWhatsAppShared **مستلِمُ طلب الواتساب شارك موقعَه بعد إنشاء الطلب** — يُكتب نقطةَ تسليمٍ
// معروفة (`dropoff`، `dropoff_known`) ما لم يُسلَّم الطلبُ أو يُقفَل، **ويُبثّ للسائق المُسنَد في
// لحظته** (البثُّ وإشعارٌ يوقظ الهاتف). **والأجرةُ لا تتبدّل** — كُتبت عند الإنشاء.
//
// **ونقطةٌ خارجَ التغطية تُكتب أيضاً ويُخبَر المكتب** — المستلِمُ أعلمُ بمكانه، والسائقُ في
// الطريق؛ والمكتبُ يقرّر. **ومنطقةُ الطلب منطقةُ النقطة الواصلة إن كانت داخلَ التغطية** — طلبٌ
// أُنشئ بلا موقعٍ كان على منطقة المتجر، **كما يُنشأ بالموقع على منطقة المستلِم.** ويردّ أكُتبت.
func (s *Service) ApplyWhatsAppShared(ctx context.Context, orderID string, lat, lng float64) (bool, error) {
	return s.applyWhatsAppShared(ctx, orderID, lat, lng, false)
}

// FillWhatsAppShared **يكتب الموقعَ الواصلَ في طلبٍ بلا نقطةٍ معروفةٍ وحدَه** — لا فوق نقطةٍ
// معروفة. **لما يُكتب بعد وقته**: موقعٌ وصل قبل الإنشاء خارجَ التغطية (يُنشأ الطلبُ بلا نقطةٍ ثمّ
// يُكتب)، **وجولةُ الإصلاح** لموقعٍ وصل وسقطت كتابتُه. **فكاتبان معاً لا يُخبران السائقَ مرّتين**:
// الثاني يجد النقطةَ معروفةً فلا يكتب.
func (s *Service) FillWhatsAppShared(ctx context.Context, orderID string, lat, lng float64) (bool, error) {
	return s.applyWhatsAppShared(ctx, orderID, lat, lng, true)
}

// waPointOutside **أخارجَ التغطية يقيناً؟** — فحصٌ سقط ليس «خارجها»: **لا يُخبَر المكتبُ بظنّ.**
func waPointOutside(covered bool, err error) bool { return err == nil && !covered }

func (s *Service) applyWhatsAppShared(ctx context.Context, orderID string, lat, lng float64,
	onlyUnknown bool) (bool, error) {
	if !ValidPoint(lat, lng) {
		return false, ErrBadPoint
	}
	// **والتغطيةُ تُفحص ولا تمنع** — نقطةٌ خارجها تُكتب والمكتبُ يُخبَر. **وداخلَها منطقتُها منطقةُ
	// الطلب**؛ وفحصٌ سقط يُبقي المنطقةَ كما هي.
	covered, cerr := s.CoverableAt(ctx, s.db, lat, lng)
	if cerr != nil {
		s.logger.Warn("طلب واتساب: فحصُ تغطية الموقع الواصل سقط", "order", orderID, "error", cerr)
	}
	var zoneID *string
	if cerr == nil && covered {
		if z, err := s.ZoneAt(ctx, s.db, lat, lng); err == nil && z.ID != "" {
			zoneID = &z.ID
		}
	}
	var number int64
	var driverID *string
	err := s.db.QueryRow(ctx, `
		UPDATE orders
		   SET dropoff = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography,
		       dropoff_known = true, zone_id = COALESCE($4::uuid, zone_id), updated_at = now()
		 WHERE id = $1 AND origin = 'whatsapp'
		   AND delivered_at IS NULL AND closed_at IS NULL
		   AND (NOT $5 OR NOT dropoff_known)
		RETURNING number, driver_id::text`, orderID, lat, lng, zoneID, onlyUnknown).Scan(&number, &driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	s.afterWhatsAppRecipientChange(ctx, orderID, driverID, number, waLocationArrivedTitle, "")

	if waPointOutside(covered, cerr) && s.notify != nil {
		s.notify.NotifyOps(ctx, notifications.Input{
			Kind: notifications.KindOrder, Title: waLocationOutsideTitle,
			Body:   "#" + itoa(number) + " — الموقع انكتب بالطلب ووصل للسائق، شوفوا إذا بيقدر يوصل",
			Entity: "order", EntityID: orderID, Href: "/dashboard/orders",
		})
	}
	return true, nil
}

// ══════════════════════════════════════════════════════════════════════
// **موقعُ صاحب الأمانة يصل بعد الإنشاء** (قرارُ المالك ٢٠٢٦-١١-١١)
// ══════════════════════════════════════════════════════════════════════
//
// «فنروح ع موقع الامانة نستلمها ونرجع نوصلها لصاحبها». **الأمانةُ بلا نقطة صاحبها «مقبولةٌ» لا
// يراها سائق** — فإن وصل موقعُه (ردُّه على البوت أو الرابط) كُتب نقطةَ استلامٍ (`pickup_override`)
// **ونزل الطلبُ للطابور وعُرض في لحظته.** والأجرةُ لا تتبدّل — كُتبت عند الإنشاء.
//
// **ونقطةٌ خارجَ التغطية تُكتب ولا ينزل الطلب** — يُخبَر المكتبُ ويقرّر (ينزله بيده أو يلغيه).

// waPickupArrivedTitle/waPickupOutsideTitle **ما يقرؤه السائقُ والمكتب.**
const (
	waPickupArrivedTitle = "وصل موقع صاحب الأمانة — صار على الخريطة"
	waPickupOutsideTitle = "موقع صاحب أمانة واتساب برّا التغطية"
	// waAmanahDispatchNote **سطرُ السجلّ لنزول الأمانة بعد وصول الموقع.**
	waAmanahDispatchNote = "وصل موقع صاحب الأمانة — نزل للسائقين"
)

// ApplyWhatsAppPickup **صاحبُ الأمانة شارك موقعَه** — يُكتب نقطةَ استلامٍ ما لم يُستلَم الطلبُ أو
// يُقفَل، **وينزل للسائقين إن كان ينتظره.** ويردّ أكُتب.
func (s *Service) ApplyWhatsAppPickup(ctx context.Context, orderID string, lat, lng float64) (bool, error) {
	return s.applyWhatsAppPickup(ctx, orderID, lat, lng, false)
}

// FillWhatsAppPickup **كـ`ApplyWhatsAppPickup` في أمانةٍ بلا نقطة استلامٍ وحدَها** — لجولة الإصلاح:
// موقعٌ وصل وسقطت كتابتُه. **وكاتبان معاً لا يُنزلان الطلبَ مرّتين**: الثاني يجد النقطةَ مكتوبة.
func (s *Service) FillWhatsAppPickup(ctx context.Context, orderID string, lat, lng float64) (bool, error) {
	return s.applyWhatsAppPickup(ctx, orderID, lat, lng, true)
}

func (s *Service) applyWhatsAppPickup(ctx context.Context, orderID string, lat, lng float64,
	onlyUnknown bool) (bool, error) {
	if !ValidPoint(lat, lng) {
		return false, ErrBadPoint
	}
	covered, cerr := s.CoverableAt(ctx, s.db, lat, lng)
	if cerr != nil {
		s.logger.Warn("أمانة واتساب: فحصُ تغطية موقع صاحبها سقط", "order", orderID, "error", cerr)
	}
	var number int64
	var driverID *string
	var status string
	err := s.db.QueryRow(ctx, `
		UPDATE orders
		   SET pickup_override = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, updated_at = now()
		 WHERE id = $1 AND origin = 'whatsapp' AND wa_amanah
		   AND picked_up_at IS NULL AND closed_at IS NULL
		   AND (NOT $4 OR pickup_override IS NULL)
		RETURNING number, driver_id::text, status`, orderID, lat, lng, onlyUnknown).
		Scan(&number, &driverID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	outside := waPointOutside(covered, cerr)
	if outside && s.notify != nil {
		s.notify.NotifyOps(ctx, notifications.Input{
			Kind: notifications.KindOrder, Title: waPickupOutsideTitle,
			Body:   "#" + itoa(number) + " — الموقع انكتب بالطلب وما نزل للسائقين، شوفوا إذا بنقدر نستلم",
			Entity: "order", EntityID: orderID, Href: "/dashboard/orders",
		})
	}
	// **ينتظر الموقعَ وداخلَ التغطية (أو فحصُها سقط) ⇒ ينزل ويُعرض** — بالآلة نفسِها باسم المكتب
	// (والآلةُ تبثّ وتعرض أوّلَ عرضٍ بنفسها، `afterTransition`)، **وتعثّرُه لا يُبطل نقطةً كُتبت**:
	// المكتبُ يراه «مقبولاً» بنقطةٍ وينزله بيده.
	if status == StAccepted && !outside {
		if _, err := s.Transition(ctx, "", []string{"ops"}, orderID, StDispatching, waAmanahDispatchNote); err != nil {
			s.logger.Warn("أمانة واتساب: النزولُ بعد الموقع تعثّر — تنتظر المكتب", "order", orderID, "error", err)
			s.afterWhatsAppRecipientChange(ctx, orderID, driverID, number, waPickupArrivedTitle, "")
		}
		return true, nil
	}
	s.afterWhatsAppRecipientChange(ctx, orderID, driverID, number, waPickupArrivedTitle, "")
	return true, nil
}

// UpdateWhatsAppAddress **المكتبُ يصحّح العنوانَ المكتوب** (ملحقُ المالك ٢٠٢٦-١١-٠٥، د: «نحن رح
// نعبيه يدوي») — قبل التسليم وحدَه، **والتدقيقُ في المعاملة نفسِها** (`hook`)، **ويُبثّ للسائق
// المُسنَد في لحظته** بلا رقم المستلِم. ويردّ العنوانَ القديم مع الطلب.
func (s *Service) UpdateWhatsAppAddress(ctx context.Context, orderID, address string,
	hook func(context.Context, dbtx.Querier, string) error) (*Order, error) {
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, httpx.ErrNotFound
	}
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var origin, old string
	var number int64
	var driverID *string
	var ended bool
	if err := tx.QueryRow(ctx, `
		SELECT origin, address_text, number, driver_id::text,
		       delivered_at IS NOT NULL OR closed_at IS NOT NULL
		  FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&origin, &old, &number, &driverID, &ended); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		return nil, err
	}
	if origin != OriginWhatsApp {
		return nil, ErrWANotWhatsApp
	}
	if ended {
		return nil, ErrWAAddressClosed
	}
	if _, err := tx.Exec(ctx,
		`UPDATE orders SET address_text = $2, updated_at = now() WHERE id = $1`, orderID, address); err != nil {
		return nil, err
	}
	if hook != nil {
		if err := hook(ctx, tx, old); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	s.afterWhatsAppRecipientChange(ctx, orderID, driverID, number, waAddressChangedTitle, address)
	return s.GetByID(ctx, orderID)
}

// afterWhatsAppRecipientChange **البثُّ للمكتب والسائق، وإشعارٌ يوقظ هاتفَ السائق المُسنَد** —
// فيقرأ النقطةَ أو العنوانَ الجديد بلا تحديث. **ولا رقمَ للمستلِم في شيءٍ منها.**
func (s *Service) afterWhatsAppRecipientChange(ctx context.Context, orderID string, driverID *string,
	number int64, title, body string) {
	if o, err := s.GetByID(ctx, orderID); err == nil {
		s.publishOrder(o)
	}
	if driverID == nil || *driverID == "" || s.notify == nil {
		return
	}
	text := "#" + itoa(number)
	if body != "" {
		text += " — " + body
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: *driverID, Kind: notifications.KindOrder,
		Title: title, Body: text,
		Entity: "order", EntityID: orderID, Href: "/portal",
		Apps: []string{notifications.AppDriver},
	})
	s.pub.Publish("driver:"+*driverID, map[string]any{"type": "order"})
}

// WhatsAppArrivedText **رسالةُ الوصول لزبون طلب الواتساب** (قرارُ المالك ٢٠٢٦-١١-٠٨) — نصُّه حرفاً.
const WhatsAppArrivedText = "سائق رحّال غو وصل إلى موقعك، الرجاء الخروج لاستلام طلبك"

// WhatsAppArrivedNotice **رقمُ زبون طلب الواتساب لرسالة الوصول** — أو `false` لغيره.
func (s *Service) WhatsAppArrivedNotice(ctx context.Context, orderID string) (phone string, ok bool, err error) {
	var origin string
	err = s.db.QueryRow(ctx, `SELECT origin, COALESCE(recipient_phone, '') FROM orders WHERE id = $1`, orderID).
		Scan(&origin, &phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return phone, origin == OriginWhatsApp && phone != "", nil
}
