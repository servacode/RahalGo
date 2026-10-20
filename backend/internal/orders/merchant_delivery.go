package orders

// ══════════════════════════════════════════════════════════════════════
// **«لدي توصيلة» — المتجرُ يطلب سائقاً لغرضٍ جاهزٍ عنده**
// ══════════════════════════════════════════════════════════════════════
//
// **زبونٌ اشترى من المتجر خارجَ رحّال غو** (اتّصالٌ أو واتساب)، والمتجرُ
// جهّز الغرضَ **ويطلب سائقاً وحدَه.** (قرارُ المالك ٢٠٢٦-٠٩-٢٩.)
//
// # خدمةُ توصيلٍ لا بيعٌ في سوق
//
// **لا أصنافَ ولا سلّةَ ولا عمولةَ مبيعات** — **ومالُ التجارةِ منفصلٌ عن
// مالِ التوصيل** (قرارُ المالك ٨). وما يُحسب هنا أجرةُ توصيلٍ وحدَها.
//
// # والمستلِمُ ليس مستخدماً
//
// **لا حسابَ يُنشأ له ولا دعوةَ ولا رسالةَ خارجيّة** (قرارُ المالك ٤).
// **و`customer_id` يبقى فارغاً** — ولا يُعبَّأ بصاحب المتجر: **من فعل جعل
// التوصيلةَ تظهر في «طلباتي» عنده بوصفه زبوناً، وخلطَ مالَ التجارة بمال
// التوصيل في كلّ تقرير.**
//
// # وما يُعاد ولا يُبنى
//
// **آلةُ الحالات** (`transitionsFor` — النوعُ الثالث) · **محرّكُ التوزيع
// بالقرب** (لها متجرٌ بموقعٍ حقيقيٍّ فتمرّ في حلقة القرب بلا تعديلِ سطر) ·
// **الراصد** · **شاشةُ الرحلة** · **إثباتُ التسليم** · **دفترُ المال.**

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// MaxParcelNote حدُّ وصفِ الغرض — **يقرؤه السائقُ ليعرف ما يحمل.**
//
// **ومئتان تكفي «كيس طعام ساخن، لا يُقلب»** — وما فوقها يصير رسالةً لا وصفاً،
// **ويُقرأ على شاشةِ هاتفٍ في الشارع.**
const MaxParcelNote = 200

// MerchantDeliveryInput ما يكتبه صاحبُ المتجر.
type MerchantDeliveryInput struct {
	RecipientName  string
	RecipientPhone string
	AddressText    string
	Lat, Lng       float64
	ParcelNote     string
	DriverNote     string
	// FeePayer **من يدفع أجرةَ التوصيل** — `merchant` · `merchant_cash` · `recipient`.
	FeePayer string
	// HasPoint **أحدّد نقطةَ التسليم؟** — وإلّا فموقعُ المتجر، والنقطةُ «غيرُ معروفة».
	HasPoint bool
	// Mode **«توصيل» أو «إحضار»** (قرارُ المالك ٢٠٢٦-١٠-١٥) — `send` (وفارغُه) أو `fetch`.
	// **وفي الإحضار العنوانُ والنقطةُ مكانُ الاستلام**، والوجهةُ المتجر.
	Mode string
	// CollectAmount **«بدك السائق يقبض مبلغ من المستلم؟»** — عرضٌ وحدَه، **لا يدخل الدفترَ
	// ولا الصندوق**: السائقُ يدفعه للمتجر من جيبه ويأخذه من المستلِم خارجَ النظام. وnil: لا شيء.
	CollectAmount *int64
	// Currency **عملةُ المبلغ** — «SYP» أو «USD» (قرارُ المالك ٢٠٢٦-١٠-٢٠)، والدولارُ بالسنتات
	// في `CollectAmount`. **وفارغٌ ليرة.**
	Currency string
}

// وضعا التوصيلة (قرارُ المالك ٢٠٢٦-١٠-١٥).
const (
	// DeliveryModeSend **«توصيل»** — من المتجر إلى المستلِم. الافتراض.
	DeliveryModeSend = "send"
	// DeliveryModeFetch **«إحضار»** — من العنوان إلى المتجر، والأجرةُ ذهاباً وإياباً.
	DeliveryModeFetch = "fetch"
)

// normalizeDeliveryMode **الفارغُ «توصيل»**، وما سوى الاثنين خطأ.
func normalizeDeliveryMode(m string) (string, bool) {
	switch strings.TrimSpace(m) {
	case "", DeliveryModeSend:
		return DeliveryModeSend, true
	case DeliveryModeFetch:
		return DeliveryModeFetch, true
	}
	return "", false
}

// CreateMerchantDelivery **يُنشئ التوصيلةَ «بانتظار موافقة المنصّة».**
//
// # وتمرّ بالمكتب كأيّ طلب (قرارُ المالك ٢٠٢٦-١٠-٠٥)
//
// **كانت تُولد في الطابور بلا «معلَّق»** — بحجّة أنّ المتجرَ مُنشئٌ لا قابل.
// **فصارت تُولد `pending`** وتظهر في لوحة الطلبات بالرنين نفسِه، **ولا تنزل
// إلى السائقين إلّا بعد قبول المكتب** (أو قبولٍ تلقائيٍّ بقواعده). **وللمكتب
// أن يرفضها** فيُخبَر المتجرُ بالسبب.
//
// **ولا مالَ يتحرّك عند الإنشاء**: أجرةُ «المتجر يدفع» تُخصم عند القبول لا
// قبله (`settleMerchantDelivery`) — **فالرفضُ لا يحتاج ردّاً لأنّ شيئاً لم يُؤخذ.**
//
// # والتغطيةُ تُفحص كما تُفحص لكلّ طلب
//
// (قرارُ المالك ٥: «لا يكفي وجود Driver لتجاوز service zone».)
//
// **وثلاثةُ حرّاسٍ قائمةٍ تُعاد**: أُطلِق المكانُ إداريّاً · النقطةُ داخلَ
// منطقةِ خدمة · والمنطقةُ مفتوحةٌ الآن. **ولا حارسَ رابعٌ يُخترَع.**
func (s *Service) CreateMerchantDelivery(ctx context.Context, merchantID, actorID string,
	in MerchantDeliveryInput) (*Order, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // **إرجاعٌ بعد إيداعٍ لا يضرّ**
	id, err := s.CreateMerchantDeliveryIn(ctx, tx, merchantID, actorID, in)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.AfterMerchantDelivery(ctx, id)
}

// CreateMerchantDeliveryIn **يكتب التوصيلةَ في معاملة من يناديه** — ولا يُثبّت.
//
// **لتُثبَّت مع علامة منع التكرار في معاملةٍ واحدة** (`WithIdempotentTx`):
// توصيلةٌ أُنشئت وضاع ردُّها فأُعيدت **لا تُنشأ ثانيةً ولا تُخصم مرّتين.**
// **والتحقّقُ من المدخلات يُعاد هنا** — فمن ناداها مباشرةً لا يتخطّاه.
func (s *Service) CreateMerchantDeliveryIn(ctx context.Context, tx dbtx.Querier,
	merchantID, actorID string, in MerchantDeliveryInput) (string, error) {
	in.RecipientName = strings.TrimSpace(in.RecipientName)
	in.RecipientPhone = strings.TrimSpace(in.RecipientPhone)
	in.AddressText = strings.TrimSpace(in.AddressText)
	in.ParcelNote = strings.TrimSpace(in.ParcelNote)
	if in.RecipientName == "" || in.RecipientPhone == "" || in.AddressText == "" {
		return "", httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if !validFeePayer(in.FeePayer) {
		return "", httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	mode, ok := normalizeDeliveryMode(in.Mode)
	if !ok {
		return "", httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	// **والمبلغُ موجبٌ أو لا شيء** — صفرٌ يُقرأ «اقبض صفراً» فيسأل السائقُ عنه.
	if in.CollectAmount != nil && *in.CollectAmount <= 0 {
		return "", httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if in.Currency == "" {
		in.Currency = "SYP"
	}
	if in.Currency != "SYP" && in.Currency != "USD" {
		return "", httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if len([]rune(in.ParcelNote)) > MaxParcelNote {
		in.ParcelNote = string([]rune(in.ParcelNote)[:MaxParcelNote])
	}
	// **ونقطةُ التسليم اختياريّة** (نصُّ المالك: «يمكن لا يملك عنوانَ المستلِم
	// على الخريطة») — **فالأجرةُ والتغطيةُ من موقع المتجر**، والنقطةُ تُعلَّم
	// «غيرَ معروفة» فلا يُوجَّه السائقُ إليها.
	dropoffKnown := in.HasPoint
	if !in.HasPoint {
		lat, lng, err := s.merchantPoint(ctx, tx, merchantID)
		if err != nil {
			return "", err
		}
		in.Lat, in.Lng = lat, lng
	}
	if !ValidPoint(in.Lat, in.Lng) {
		return "", ErrBadPoint
	}

	// ── التغطية ───────────────────────────────────────────────────────
	if err := s.requirePlaceLaunched(ctx, tx, in.Lat, in.Lng); err != nil {
		return "", err
	}
	zone, err := s.RequireServiceable(ctx, tx, in.Lat, in.Lng)
	if err != nil {
		return "", err
	}
	if err := s.requireZoneOpen(ctx, tx, zone); err != nil {
		return "", err
	}
	// **ولا توصيلةَ بلا سائقٍ بالدوام** — انظر `merchant_delivery_drivers.go`.
	if err := s.requireDriverOnShift(ctx, tx); err != nil {
		return "", err
	}

	// ── المال ─────────────────────────────────────────────────────────
	//
	// **أجرةُ المنطقةِ هي الأجرة** — وهي مصدرُ الرقم في كلّ المشروع
	// (ملاحظةُ المالك ٢٠٢٦-٠٨-٠٤: «قيمُ التوصيل تأتي من مكانٍ واحد»).
	//
	// **ونصيبُ المنصّةِ يُلقَط لا يُقرأ وقتَ التسوية** (قرارُ المالك ٦):
	// رفعُ النسبة غداً لا يمسّ توصيلةَ اليوم.
	//
	// **والبقيّةُ للسائق** — و`creditTreasury` تُخرج الفرقَ إلى الخزينة من
	// نفسِها (مدفوعٌ − مستردٌّ − ما وصل الأطراف). **فلا قيدَ مخترَع.**
	//
	// **ثمّ صارت الأجرةُ العامّةَ نفسَها** (قرارُ المالك ٢٠٢٦-١٠-٠٤، الإعدادات
	// البند ٢): كانت تُقرأ من عمود المنطقة **ولوحةُ المناطق تكتبه صفراً في كلّ
	// تعديل** — فتعديلُ اسم منطقةٍ جعل كلَّ توصيلةٍ فيها مجّانيّةً وأجرَ السائق
	// صفراً بلا خطأ. **والأجرةُ من `delivery.fee` كالطلب العاديّ** (`DeliveryAt`).
	//
	// **ولها أجرتُها الخاصّة إن ضُبطت** (قرارُ المالك ٢٠٢٦-١٠-٠٥) — **ومنذ ٢٠٢٦-١٠-١٦
	// مفاتيحُ أساسها ومسافتها هي لا مفاتيحُ العاديّ.** انظر `merchantDeliveryFee`.
	fee, _, _, err := s.merchantDeliveryFee(ctx, tx, merchantID, in.Lat, in.Lng, in.HasPoint, mode)
	if err != nil {
		return "", err
	}
	pct, fixed := s.platformShare(ctx, tx, ShareMerchant)
	driverFee := DriverFeeWithShare(fee, pct, fixed)

	snap, err := s.snapshotNow(ctx, tx)
	if err != nil {
		return "", err
	}

	// **وقدرةُ المتجر تُفحص الآن ولا يُؤخذ شيء** — والخصمُ عند قبول المكتب.
	//
	// **ومن لا يقدر يُقال له الآن لا بعد ساعة**: توصيلةٌ تنتظر المكتبَ ثمّ
	// تُردّ لأنّ المحفظةَ فارغةٌ تُضيّع وقتَ الاثنين. **والقاعدةُ قاعدةُ الخصم
	// نفسُها** (المتاحُ = الرصيدُ − المحجوز) — ويُعاد فحصُها بقفلٍ عند القبول.
	//
	// **ولا دينَ أبداً** (قرارُ المالك ٢٠٢٦-١٠-١٥): «المتجر يدفع» بمحفظةٍ تكفي وإلّا
	// يختار النقد — **لا سقفَ ولا دينٌ مفتوح.**
	if in.FeePayer == FeePayerMerchant {
		can, err := s.merchantCanPay(ctx, tx, merchantID, fee)
		if err != nil {
			return "", err
		}
		if !can {
			return "", ErrDeliveryCreditExhausted
		}
	}

	// **ومن يدفع يحدّد أين يقع المال**: المتجرُ ⇐ يُخصم من محفظته عند القبول؛
	// المستلِمُ ⇐ نقدٌ بيد السائق عند التسليم.
	// **والنقدُ بيد السائق** — من المستلِم عند التسليم، أو من المتجر عند
	// الاستلام («أنا نقداً»). **ولا خصمَ ولا دينَ في الحالين.**
	var cashDue int64
	if in.FeePayer == FeePayerRecipient || in.FeePayer == FeePayerMerchantCash {
		cashDue = fee
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (kind, merchant_id, address_text, dropoff,
		                    recipient_name, recipient_phone, parcel_note, fee_payer,
		                    status, payment_method, subtotal, delivery_fee, driver_fee,
		                    total, cash_due, notes,
		                    snap_merchant_commission_percent, snap_rep_commission_percent,
		                    snap_commission_source, snap_activation_orders,
		                    snap_platform_delivery_percent, dropoff_known, snap_platform_delivery_fixed,
		                    delivery_mode, collect_amount, amount_currency)
		VALUES ('merchant_delivery', $1, $2,
		        ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography,
		        $5, $6, $7, $8,
		        'pending', 'cash', 0, $9, $10,
		        $9, $11, $12,
		        $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		RETURNING id::text`,
		merchantID, in.AddressText, in.Lat, in.Lng,
		in.RecipientName, in.RecipientPhone, nullIfEmpty(in.ParcelNote), in.FeePayer,
		fee, driverFee, cashDue, strings.TrimSpace(in.DriverNote), // **و`notes` NOT NULL** — فراغٌ لا NULL
		snap.MerchantCommission, snap.RepCommissionPercent,
		snap.CommissionSource, snap.ActivationOrders, pct, dropoffKnown, fixed,
		mode, in.CollectAmount, in.Currency).Scan(&id)
	if err != nil {
		return "", err
	}

	// ── دفعُ المتجر — عند قبول المكتب لا هنا ─────────────────────────
	//
	// (قرارُ المالك ٢٠٢٦-١٠-٠٥.) **كان يُخصم هنا** — والتوصيلةُ تنزل الطابورَ
	// فوراً. **وصار للمكتب أن يرفضها**، فخصمٌ عند الإنشاء يعني ردّاً عند كلّ
	// رفض: **قيدان لا قيدٌ، وبابٌ لخطأٍ في كلّ رفض.** فيُخصم في معاملة القبول
	// (`settleMerchantDelivery`) بقفل المحفظة نفسِه، **وإن لم تكفِ رُدّ القبول.**

	// **ولحظةُ الميلاد تُقيَّد** — **وسجلُّ حالاتٍ يبدأ من أوّل انتقالٍ لا من
	// الإنشاء يُقرأ طلباً بلا نشأة** (وهي علّةُ الخاصِّ التي أُصلحت).
	// **والفاعلُ صاحبُ المتجر** — هو من أنشأها.
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
		VALUES ($1, '', 'pending', $2, 'لدي توصيلة')`, id, actorID); err != nil {
		return "", err
	}

	return id, nil
}

// AfterMerchantDelivery **ما لا يقع إلّا بعد التثبيت** — القراءةُ والبثُّ وإخطارُ المكتب.
//
// **ولا عرضَ على سائق** — التوصيلةُ «بانتظار موافقة المنصّة» (قرارُ المالك
// ٢٠٢٦-١٠-٠٥)، **والعرضُ الأوّلُ يقع حين تنزل الطابور** (`transitionTx`).
func (s *Service) AfterMerchantDelivery(ctx context.Context, id string) (*Order, error) {
	o, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.publishOrder(o)
	s.notifyMerchantDeliveryCreated(ctx, o)
	return o, nil
}

// nullIfEmpty **الفراغُ يُكتب NULL لا نصّاً فارغاً.**
//
// **وعمودٌ فيه سلسلةٌ فارغةٌ يُقرأ «كُتب ثمّ مُحي»**، **وNULL تُقرأ «لم
// يُكتب»** — والفرقُ يُسأل عنه يوماً.
func nullIfEmpty(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// أطرافُ دفعِ الأجرة.
const (
	// FeePayerMerchant **المتجرُ يدفع** — من محفظته وحدَها، ولا دين (قرارُ المالك ٢٠٢٦-١٠-١٥).
	FeePayerMerchant = "merchant"
	// FeePayerRecipient **المستلِمُ يدفع** — نقداً بيد السائق عند التسليم.
	FeePayerRecipient = "recipient"
	// FeePayerMerchantCash **المتجرُ يدفع نقداً** بيد السائق عند الاستلام —
	// (نصُّ المالك ٢٠٢٦-١٠-٠١: «لازم في أنا نقدي»). لا محفظةَ ولا دين.
	FeePayerMerchantCash = "merchant_cash"
)

// ══════════════════════════════════════════════════════════════════════
// **حصّةُ المنصّة من الأجرة — لكلّ نوعٍ وحدَه، نسبةً أو مبلغاً** (قرارُ المالك ٢٠٢٦-١٠-١٣)
// ══════════════════════════════════════════════════════════════════════
//
// «إذا ٠ معناها كلها للسائق». **وتُلقَط على الطلب لحظةَ إنشائه** (`snap_platform_delivery_percent`
// و`snap_platform_delivery_fixed`) **فلا تمسّ طلباً قائماً.**

// أنواعُ الحصّة — بأسماء مفاتيحها.
const (
	ShareStandard = "standard"
	ShareMerchant = "merchant"
	SharePurchase = "purchase"
	ShareAmanah   = "amanah"
)

// SettingShare **مفتاحُ قيمة الحصّة لنوع** — و`SettingShare(t)+"_mode"` طريقتُها.
func SettingShare(t string) string { return "delivery.share_" + t }

// platformDeliveryPercentMax **لا أجرَ للسائق عند المئة** — فالحدُّ تسعون.
const platformDeliveryPercentMax = 90

// platformShare **حصّةُ المنصّة النافذةُ الآن لنوع** — نسبةٌ أو مبلغ (والآخرُ صفر).
func (s *Service) platformShare(ctx context.Context, q dbtx.Querier, t string) (pct, fixed int64) {
	if s.settings == nil {
		return 0, 0
	}
	st := s.settings.On(q)
	v := st.GetInt(ctx, SettingShare(t))
	if v <= 0 {
		return 0, 0
	}
	if st.GetString(ctx, SettingShare(t)+"_mode") == "fixed" {
		return 0, v
	}
	if v > platformDeliveryPercentMax {
		v = platformDeliveryPercentMax
	}
	return v, 0
}

// DriverFeeAfterShare **أجرُ السائق بعد حصّةٍ نسبيّة** — والكسرُ للسائق.
func DriverFeeAfterShare(fee, pct int64) int64 {
	if fee <= 0 || pct <= 0 {
		return fee
	}
	return fee - fee*pct/100
}

// DriverFeeWithShare **أجرُ السائق بعد حصّة المنصّة** — نسبةً أو مبلغاً ثابتاً، **ولا تتجاوز
// الحصّةُ الأجرةَ** (مبلغٌ أكبرُ من الأجرة يأخذها كلَّها ولا يُسالبها).
func DriverFeeWithShare(fee, pct, fixed int64) int64 {
	if fee <= 0 {
		return fee
	}
	if fixed > 0 {
		if fixed >= fee {
			return 0
		}
		return fee - fixed
	}
	return DriverFeeAfterShare(fee, pct)
}

// IsMerchantDelivery **أتوصيلةُ متجرٍ هي؟** — يُقرأ من الصفّ لا يُخمَّن.
func (s *Service) IsMerchantDelivery(ctx context.Context, orderID string) bool {
	var kind string
	if err := s.db.QueryRow(ctx,
		`SELECT kind FROM orders WHERE id = $1`, orderID).Scan(&kind); err != nil {
		return false
	}
	return kind == KindMerchantDelivery
}

// ══════════════════════════════════════════════════════════════════════
// **ما يقرؤه صاحبُ المتجر** — عرضُ السعر وقائمةُ توصيلاته (الخطوة ١٨)
// ══════════════════════════════════════════════════════════════════════

// MerchantDeliveryQuote **ما يلزم المتجرَ قبل أن يضغط «اطلب سائقاً».**
//
// **والأجرةُ تُقال قبل الإرسال لا بعده** — ومن عرف أنّها ٥٠٠٠ ومحفظتُه ٣٠٠٠
// اختار النقدَ ولم يُردّ.
type MerchantDeliveryQuote struct {
	Fee      int64  `json:"fee"`
	ZoneName string `json:"zone_name"`
	// FeeBasis **من أين جاءت الأجرة** (قرارُ المالك ٢٠٢٦-١٠-١٥) — `distance` (الأساسُ
	// ومسافةُ المتجر ← المستلِم) · `no_point` (بلا نقطةٍ: الأساسُ وحدَه) · `flat` (أجرةُ
	// «لدي توصيلة» الخاصّة).
	FeeBasis string `json:"fee_basis"`
	// DistanceM **المسافةُ التي سُعّرت** — خطٌّ مستقيمٌ بالمتر، ومضاعفةٌ في الإحضار. وصفرٌ
	// بلا نقطة.
	DistanceM int64 `json:"distance_m"`
	// Mode **الوضعُ الذي سُعّر** — `send` أو `fetch`.
	Mode          string `json:"mode"`
	WalletBalance int64  `json:"wallet_balance"`
	// AvailableBalance **المتاحُ للدفع** — الرصيدُ ناقصَ المحجوز (سحبٌ معلَّق).
	AvailableBalance int64 `json:"available_balance"`
	// MerchantCanPay **أيكفي المتاحُ في محفظته الأجرة؟** — وإلّا يُعطَّل خيارُ «من محفظتي».
	// **ويُحسب بقاعدة `chargeMerchantDelivery` نفسِها** لا بتقديرٍ ثانٍ. **ولا دين.**
	MerchantCanPay bool `json:"merchant_can_pay"`
}

// QuoteMerchantDelivery **أجرةُ النقطة ومقدرةُ المتجر** — بحرّاس الإنشاء نفسِها.
func (s *Service) QuoteMerchantDelivery(ctx context.Context, merchantID string,
	lat, lng float64, hasPoint bool, mode string) (*MerchantDeliveryQuote, error) {
	mode, ok := normalizeDeliveryMode(mode)
	if !ok {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if !hasPoint {
		var err error
		if lat, lng, err = s.merchantPoint(ctx, s.db, merchantID); err != nil {
			return nil, err
		}
	}
	if !ValidPoint(lat, lng) {
		return nil, ErrBadPoint
	}
	if err := s.requirePlaceLaunched(ctx, s.db, lat, lng); err != nil {
		return nil, err
	}
	zone, err := s.RequireServiceable(ctx, s.db, lat, lng)
	if err != nil {
		return nil, err
	}
	if err := s.requireZoneOpen(ctx, s.db, zone); err != nil {
		return nil, err
	}
	// **ويُقال عند فتح الشاشة** — لا بعد ملء النموذج كلِّه.
	if err := s.requireDriverOnShift(ctx, s.db); err != nil {
		return nil, err
	}
	// **والرقمُ بدالّة الإنشاء نفسِها** — فلا يُعرض رقمٌ ويُخصم غيرُه.
	fee, dist, basis, err := s.merchantDeliveryFee(ctx, s.db, merchantID, lat, lng, hasPoint, mode)
	if err != nil {
		return nil, err
	}
	q := &MerchantDeliveryQuote{Fee: fee, ZoneName: zone.Name, FeeBasis: basis,
		DistanceM: int64(dist), Mode: mode}
	if q.WalletBalance, q.AvailableBalance, err = merchantWallet(ctx, s.db, merchantID, false); err != nil {
		return nil, err
	}
	q.MerchantCanPay = merchantCanCover(q.AvailableBalance, q.Fee)
	return q, nil
}

// ListMerchantDeliveries **توصيلاتُ المتجر، الأحدثُ أوّلاً** — جاريةً ومنتهية.
//
// **وفي الوضعين** («المنصّة تدير» و«المتاجر تدير»): **هو أنشأها ويتابعها** —
// لا كطلبات السوق التي تُحجب جاريتُها في وضع المنصّة.
func (s *Service) ListMerchantDeliveries(ctx context.Context, merchantID string, limit, page int) ([]Order, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if page < 1 {
		page = 1
	}
	rows, err := s.db.Query(ctx, orderSelect+`
		WHERE o.merchant_id = $1 AND o.kind = 'merchant_delivery'
		ORDER BY o.created_at DESC LIMIT $2 OFFSET $3`, merchantID, limit, (page-1)*limit)
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

func validFeePayer(p string) bool {
	return p == FeePayerMerchant || p == FeePayerMerchantCash || p == FeePayerRecipient
}

// merchantPoint **موقعُ المتجر** — مصدرُ الأجرة والتغطية حين لا نقطةَ للمستلِم.
func (s *Service) merchantPoint(ctx context.Context, q dbtx.Querier, merchantID string) (float64, float64, error) {
	var lat, lng *float64
	if err := q.QueryRow(ctx, `
		SELECT ST_Y(location::geometry), ST_X(location::geometry)
		  FROM merchants WHERE id = $1`, merchantID).Scan(&lat, &lng); err != nil || lat == nil || lng == nil {
		// **متجرٌ بلا دبّوسٍ لا يُسعَّر له** — يُقال له أن يحدّد موقعه.
		return 0, 0, ErrBadPoint
	}
	return *lat, *lng, nil
}

// SettingMerchantDeliveryFee **أجرةُ «لدي توصيلة»** (قرارُ المالك ٢٠٢٦-١٠-٠٥).
const SettingMerchantDeliveryFee = "delivery.merchant_fee"

// أساسُ أجرة التوصيلة — يُقال في عرض السعر.
const (
	FeeBasisDistance = "distance"
	FeeBasisNoPoint  = "no_point"
	FeeBasisFlat     = "flat"
)

// ══════════════════════════════════════════════════════════════════════
// **أجرةُ التوصيلة: أساسٌ ثابتٌ + كيلومتراتُ المتجر ← المستلِم** (قرارُ المالك ٢٠٢٦-١٠-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// **الإنشاءُ وعرضُ السعر من هنا وحدَه.**
//
// **كانت المسافةُ من الدبّوس إلى مركز المنطقة** — فتوصيلةٌ من متجرٍ في طرف المنطقة إلى
// طرفها الآخر تُسعَّر كأنّها جارتُه. **وصارت بين موقع المتجر ونقطة المستلِم**، خطّاً
// مستقيماً بـ`ST_Distance` كما يقيس العاديُّ (`ZoneAt`) — **ومفاتيحُها هي منذ ٢٠٢٦-١٠-١٦**:
// `delivery.merchant_base_fee` أساساً و`delivery.merchant_per_km` لكلّ كيلومترٍ (مقرَّبٍ لأعلى)
// و`delivery.merchant_max_fee` سقفاً، **ومفتاحُ `delivery.merchant_by_distance` يحكمها** —
// والحسبةُ حسبةُ العاديّ نفسُها (`pricing.FeeRule.At`). **وكانت تقرأ مفاتيحَ العاديّ**، فمن رفع
// أجرةَ الطلب العاديّ رفعها معه.
//
//   - **والإحضارُ ذهابٌ وإياب**: المتجر ← العنوان ← المتجر، فالمسافةُ مضاعفة.
//   - **وبلا نقطةٍ الأساسُ وحدَه** — والعرضُ يقوله (`no_point`).
//   - **و`delivery.merchant_fee` فوق الصفر أجرةٌ ثابتةٌ تسبق كلَّ هذا** (قرارُ المالك
//     ٢٠٢٦-١٠-٠٥) — **وصفرُه «لا أجرةَ خاصّة» لا «مجّاناً».**
func (s *Service) merchantDeliveryFee(ctx context.Context, q dbtx.Querier, merchantID string,
	lat, lng float64, hasPoint bool, mode string) (fee int64, distanceM float64, basis string, err error) {
	st := s.settings.On(q)
	if own := st.GetInt(ctx, SettingMerchantDeliveryFee); own > 0 {
		return own, 0, FeeBasisFlat, nil
	}
	// **ومفاتيحُها هي لا مفاتيحُ العاديّ** (قرارُ المالك ٢٠٢٦-١٠-١٦) — `pricing.MerchantDeliveryFeeKeys`.
	rule := pricing.RuleOf(ctx, st, pricing.MerchantDeliveryFeeKeys)
	if !hasPoint {
		return rule.Base, 0, FeeBasisNoPoint, nil
	}
	if err := q.QueryRow(ctx, `
		SELECT COALESCE(ST_Distance(location, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography), 0)
		  FROM merchants WHERE id = $1`, merchantID, lat, lng).Scan(&distanceM); err != nil {
		return 0, 0, "", err
	}
	if mode == DeliveryModeFetch {
		distanceM *= 2
	}
	return rule.At(distanceM), distanceM, FeeBasisDistance, nil
}

// SettingMerchantDebtOpen **مفتاحٌ متروكٌ لا يُقرأ** (قرارُ المالك ٢٠٢٦-١٠-١٥: لا دينَ على
// أجرة توصيلة). **كان «دينُ لدي توصيلة بلا سقف» مشتعلاً افتراضاً** — فيُقيَّد على المتجر
// دينٌ لا حدَّ له ولا يُخبَر. **ويبقى في الفهرس والقاعدة** كي لا يُكسر حفظٌ قديمٌ له، ولا
// يقرؤه المحرّكُ بعد اليوم؛ وكذلك `merchants.delivery_credit_limit`.
const SettingMerchantDebtOpen = "delivery.merchant_debt_open"

// merchantCanCover **أيكفي المتاحُ الأجرة؟** — ولا دين.
//
// **قاعدةُ `chargeMerchantDelivery` نفسُها** — تُقرأ في عرض السعر وفي الإنشاء
// بلا قفل، **ويُعاد حسابُها بقفلٍ عند القبول** حيث يقع الخصم.
func merchantCanCover(available, fee int64) bool {
	return fee <= 0 || available >= fee
}

// merchantWallet **رصيدُ محفظة صاحب المتجر ومتاحُه** — المتاحُ الرصيدُ ناقصَ المحجوز
// (سحبٌ معلَّق). **و`lock` يقفل الصفَّ** حيث يقع الخصم. ولا محفظةَ بعد ⇒ صفران.
func merchantWallet(ctx context.Context, q dbtx.Querier, merchantID string, lock bool) (balance, available int64, err error) {
	sql := `
		SELECT w.balance, w.balance - w.reserved
		  FROM merchants m JOIN wallets w ON w.user_id = m.owner_user_id
		 WHERE m.id = $1`
	if lock {
		sql += ` FOR UPDATE OF w`
	}
	err = q.QueryRow(ctx, sql, merchantID).Scan(&balance, &available)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, nil
	}
	return balance, available, err
}

// merchantCanPay **قدرةُ المتجر الآن** — قراءةٌ بلا قفلٍ ولا خصم.
func (s *Service) merchantCanPay(ctx context.Context, q dbtx.Querier, merchantID string, fee int64) (bool, error) {
	_, available, err := merchantWallet(ctx, q, merchantID, false)
	if err != nil {
		return false, err
	}
	return merchantCanCover(available, fee), nil
}
