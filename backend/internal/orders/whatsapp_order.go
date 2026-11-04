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
)

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
	// Lat/Lng **موقعُ المستلِم على الخريطة** — إلزاميّ («وموقعه على الخريطة»).
	Lat, Lng   *float64
	DriverNote string
	// Fee **أجرةٌ عدّلها الموظّف** — وnil: المحسوبة.
	Fee *int64
}

// WhatsAppQuote **أجرةُ طلب الواتساب قبل الإرسال** — بحرّاس الإنشاء ودالّته نفسِها.
type WhatsAppQuote struct {
	Fee       int64  `json:"fee"`
	FeeBasis  string `json:"fee_basis"`
	DistanceM int64  `json:"distance_m"`
	ZoneName  string `json:"zone_name"`
	// SharePercent/ShareFixed **حصّةُ المنصّة** — حصّةُ «لدي توصيلة» (`delivery.share_merchant`).
	SharePercent  int64 `json:"share_percent"`
	ShareFixed    int64 `json:"share_fixed"`
	DriverFee     int64 `json:"driver_fee"`
	PlatformShare int64 `json:"platform_share"`
	// FeeMin/FeeMax **حدّا تعديل الأجرة** — حدّا أجرة الطلب الخاصّ.
	FeeMin int64 `json:"fee_min"`
	FeeMax int64 `json:"fee_max"`
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
}

// waStore **المتجرُ كما يُكتب على الطلب.**
type waStore struct {
	merchantID           string
	name, phone, address string
	lat, lng             float64
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
		if (needName && in.StoreName == "") || in.StoreLat == nil || in.StoreLng == nil {
			return waStore{}, ErrWAStoreRequired
		}
		if !ValidPoint(*in.StoreLat, *in.StoreLng) {
			return waStore{}, ErrBadPoint
		}
		return waStore{name: in.StoreName, phone: in.StorePhone, address: in.StoreAddress,
			lat: *in.StoreLat, lng: *in.StoreLng}, nil
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
	st.lat, st.lng = *lat, *lng
	// **ورقمُ المتجر يُعدَّل في النموذج** — الموظّفُ يعرف من يردّ.
	if in.StorePhone != "" {
		st.phone = in.StorePhone
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
}

// whatsAppPricing **حرّاسُ التغطية ثمّ الأجرة** — النقطتان كلتاهما داخلَ التغطية،
// **والمنطقةُ مفتوحةٌ عند المستلِم** (حرّاسُ «لدي توصيلة» نفسُها، ولا رابع).
func (s *Service) whatsAppPricing(ctx context.Context, q dbtx.Querier,
	storeLat, storeLng, lat, lng float64) (waPricing, error) {
	var p waPricing
	if !ValidPoint(storeLat, storeLng) || !ValidPoint(lat, lng) {
		return p, ErrBadPoint
	}
	// **والمتجرُ داخلَ التغطية** — لا يُرسَل سائقٌ إلى خارجها.
	if err := s.requirePlaceLaunched(ctx, q, storeLat, storeLng); err != nil {
		return p, err
	}
	if _, err := s.RequireServiceable(ctx, q, storeLat, storeLng); err != nil {
		return p, err
	}
	if err := s.requirePlaceLaunched(ctx, q, lat, lng); err != nil {
		return p, err
	}
	zone, err := s.RequireServiceable(ctx, q, lat, lng)
	if err != nil {
		return p, err
	}
	if err := s.requireZoneOpen(ctx, q, zone); err != nil {
		return p, err
	}
	p.zone = zone
	if p.fee, p.distanceM, p.basis, err = s.whatsAppFee(ctx, q, storeLat, storeLng, lat, lng); err != nil {
		return p, err
	}
	// **وحصّةُ المنصّة حصّةُ «لدي توصيلة»** (سؤالُ المالك ١، الافتراضُ الموصى به).
	p.pct, p.fixed = s.platformShare(ctx, q, ShareMerchant)
	return p, nil
}

// whatsAppFee **الأجرةُ كالتوصيلات** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٣) — قاعدةُ «لدي توصيلة»
// (`merchantFeeRule`) على المسافة المستقيمة **من المتجر إلى المستلِم**. **والأجرةُ الخاصّةُ
// إن ضُبطت تسبقها**، والمسافةُ تُقال معها.
func (s *Service) whatsAppFee(ctx context.Context, q dbtx.Querier,
	storeLat, storeLng, lat, lng float64) (fee int64, distanceM float64, basis string, err error) {
	if err = q.QueryRow(ctx, `
		SELECT ST_Distance(ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
		                   ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography)`,
		storeLat, storeLng, lat, lng).Scan(&distanceM); err != nil {
		return 0, 0, "", err
	}
	flat, rule := s.merchantFeeRule(ctx, q)
	if flat > 0 {
		return flat, distanceM, FeeBasisFlat, nil
	}
	return rule.At(distanceM), distanceM, FeeBasisDistance, nil
}

// whatsAppFeeBounds **حدّا تعديل الأجرة** — حدّا أجرة الطلب الخاصّ (`delivery.custom_fee_min/max`).
func (s *Service) whatsAppFeeBounds(ctx context.Context, q dbtx.Querier) (lo, hi int64) {
	if s.settings == nil {
		return 0, 0
	}
	st := s.settings.On(q)
	return st.GetInt(ctx, "delivery.custom_fee_min"), st.GetInt(ctx, "delivery.custom_fee_max")
}

// QuoteWhatsAppOrder **أجرةُ الطلب قبل الإرسال** — المتجرُ بمعرّفه أو بنقطته.
func (s *Service) QuoteWhatsAppOrder(ctx context.Context, storeID string,
	storeLat, storeLng *float64, lat, lng float64) (*WhatsAppQuote, error) {
	in := WhatsAppOrderInput{StoreID: strings.TrimSpace(storeID), StoreLat: storeLat, StoreLng: storeLng}
	st, err := s.resolveWhatsAppStore(ctx, s.db, in, false)
	if err != nil {
		return nil, err
	}
	p, err := s.whatsAppPricing(ctx, s.db, st.lat, st.lng, lat, lng)
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
	if in.Details == "" || in.RecipientName == "" || in.RecipientPhone == "" || in.AddressText == "" {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if len([]rune(in.Details)) > MaxWhatsAppDetails {
		in.Details = string([]rune(in.Details)[:MaxWhatsAppDetails])
	}
	// **وموقعُ المستلِم على الخريطة إلزاميّ** — «وموقعه على الخريطة».
	if in.Lat == nil || in.Lng == nil || !ValidPoint(*in.Lat, *in.Lng) {
		return nil, ErrBadPoint
	}
	// **والثمنُ موجبٌ أو لا شيء** — صفرٌ يُقرأ «اقبض صفراً».
	if in.GoodsAmount != nil && *in.GoodsAmount <= 0 {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if in.Currency == "" {
		in.Currency = "SYP"
	}
	if in.Currency != "SYP" && in.Currency != "USD" {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}

	store, err := s.resolveWhatsAppStore(ctx, tx, in, true)
	if err != nil {
		return nil, err
	}
	p, err := s.whatsAppPricing(ctx, tx, store.lat, store.lng, *in.Lat, *in.Lng)
	if err != nil {
		return nil, err
	}

	// ── الأجرة: المحسوبةُ أو ما عدّله الموظّف بحدّيه ─────────────────────
	fee := p.fee
	if in.Fee != nil && *in.Fee != p.fee {
		lo, hi := s.whatsAppFeeBounds(ctx, tx)
		if *in.Fee < lo || (hi > 0 && *in.Fee > hi) || *in.Fee < 0 {
			return nil, &WAFeeRangeError{Min: lo, Max: hi}
		}
		fee = *in.Fee
	}
	// **وأجرةٌ معدَّلةٌ لا تزيد على الحصّة الثابتة تُرسل سائقاً بلا أجر** — تُردّ ولا تُكتب.
	if fee != p.fee && p.fixed > 0 && p.fixed >= fee {
		return nil, ErrWAFeeBelowShare
	}
	driverFee := DriverFeeWithShare(fee, p.pct, p.fixed)

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
		StoreName: store.name, PickupMerchantID: store.merchantID}
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
		                    accepted_at, dispatched_at)
		VALUES ('merchant_delivery', 'whatsapp', $1, $2,
		        $3, NULLIF($4, ''), NULLIF($5, ''), ST_SetSRID(ST_MakePoint($7, $6), 4326)::geography,
		        $8, ST_SetSRID(ST_MakePoint($10, $9), 4326)::geography, $11,
		        $12, $13, $14, 'recipient',
		        'dispatching', 'cash', 0, $15, $16,
		        $15, $15, $17,
		        $18, $19, $20, $21,
		        $22, true, $23,
		        'send', $24, $25, 0,
		        now(), now())
		RETURNING id::text, number`,
		actor, pickupMerchant,
		store.name, store.phone, store.address, store.lat, store.lng,
		in.AddressText, *in.Lat, *in.Lng, nullIfEmpty(p.zone.ID),
		in.RecipientName, in.RecipientPhone, in.Details,
		fee, driverFee, in.DriverNote,
		snap.MerchantCommission, snap.RepCommissionPercent, snap.CommissionSource, snap.ActivationOrders,
		p.pct, p.fixed,
		in.GoodsAmount, in.Currency).Scan(&out.ID, &out.Number); err != nil {
		return nil, err
	}

	// **ولحظةُ الميلاد تُقيَّد** — والفاعلُ الموظّفُ الذي كتبه، **والحالُ الأولى الطابور.**
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
		VALUES ($1, '', 'dispatching', $2, $3)`, out.ID, actor, whatsAppBirthNote); err != nil {
		return nil, err
	}
	return out, nil
}

// whatsAppBirthNote **سطرُ الميلاد في سجلّ الطلب.**
const whatsAppBirthNote = "طلب واتساب"

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
	if err := s.OfferNext(ctx, id, nil); err != nil {
		s.logger.Error("طلب واتساب: تعذّر أوّلُ عرض", "order", id, "error", err)
	}
	return o, nil
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
