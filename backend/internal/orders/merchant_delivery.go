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
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/httpx"
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
	// FeePayer **من يدفع أجرةَ التوصيل** — `merchant` أو `recipient`.
	FeePayer string
}

// CreateMerchantDelivery **يُنشئ التوصيلةَ في الطابور مباشرةً.**
//
// # ولا «معلَّق» لها
//
// **المتجرُ هو المُنشئ لا القابل** — والغرضُ جاهزٌ عنده قبل أن يضغط،
// **وخطوةُ قبولٍ يطلبها من نفسِه عبثٌ يُبطئ سائقاً.**
//
// # والتغطيةُ تُفحص كما تُفحص لكلّ طلب
//
// (قرارُ المالك ٥: «لا يكفي وجود Driver لتجاوز service zone».)
//
// **وثلاثةُ حرّاسٍ قائمةٍ تُعاد**: أُطلِق المكانُ إداريّاً · النقطةُ داخلَ
// منطقةِ خدمة · والمنطقةُ مفتوحةٌ الآن. **ولا حارسَ رابعٌ يُخترَع.**
func (s *Service) CreateMerchantDelivery(ctx context.Context, merchantID, actorID string,
	in MerchantDeliveryInput) (*Order, error) {
	in.RecipientName = strings.TrimSpace(in.RecipientName)
	in.RecipientPhone = strings.TrimSpace(in.RecipientPhone)
	in.AddressText = strings.TrimSpace(in.AddressText)
	in.ParcelNote = strings.TrimSpace(in.ParcelNote)

	// **والفراغُ يُردّ قبل أن يُفتح اتّصالٌ بالقاعدة** — نداءٌ يمضي إلى
	// المعاملة ثمّ يسقط على قيدٍ يُضيّع قفلاً ويُربك السجلّ.
	if in.RecipientName == "" || in.RecipientPhone == "" || in.AddressText == "" {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if in.FeePayer != FeePayerMerchant && in.FeePayer != FeePayerRecipient {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if len([]rune(in.ParcelNote)) > MaxParcelNote {
		in.ParcelNote = string([]rune(in.ParcelNote)[:MaxParcelNote])
	}
	if !ValidPoint(in.Lat, in.Lng) {
		return nil, ErrBadPoint
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // **إرجاعٌ بعد إيداعٍ لا يضرّ**

	// ── التغطية ───────────────────────────────────────────────────────
	if err := s.requirePlaceLaunched(ctx, tx, in.Lat, in.Lng); err != nil {
		return nil, err
	}
	zone, err := s.RequireServiceable(ctx, tx, in.Lat, in.Lng)
	if err != nil {
		return nil, err
	}
	if err := s.requireZoneOpen(ctx, tx, zone); err != nil {
		return nil, err
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
	fee := zone.DeliveryFee
	pct := s.settingInt(ctx, SettingMerchantDeliveryPlatformPercent)
	if pct < 0 {
		pct = 0
	}
	if pct > 90 {
		pct = 90
	}
	platformShare := fee * pct / 100
	driverFee := fee - platformShare

	snap, err := s.snapshotNow(ctx, tx)
	if err != nil {
		return nil, err
	}

	// **ومن يدفع يحدّد أين يقع المال**: المتجرُ ⇐ يُخصم أو يُقيَّد ديناً
	// مضبوطاً؛ المستلِمُ ⇐ نقدٌ بيد السائق عند التسليم.
	var cashDue int64
	if in.FeePayer == FeePayerRecipient {
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
		                    snap_platform_delivery_percent, dispatched_at)
		VALUES ('merchant_delivery', $1, $2,
		        ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography,
		        $5, $6, $7, $8,
		        'dispatching', 'cash', 0, $9, $10,
		        $9, $11, $12,
		        $13, $14, $15, $16, $17, now())
		RETURNING id::text`,
		merchantID, in.AddressText, in.Lat, in.Lng,
		in.RecipientName, in.RecipientPhone, nullIfEmpty(in.ParcelNote), in.FeePayer,
		fee, driverFee, cashDue, nullIfEmpty(in.DriverNote),
		snap.MerchantCommissionPercent, snap.RepCommissionPercent,
		snap.CommissionSource, snap.ActivationOrders, pct).Scan(&id)
	if err != nil {
		return nil, err
	}

	// ── دفعُ المتجر — في المعاملة نفسِها ──────────────────────────────
	//
	// **ولا توصيلةَ بلا دفعِها**: لو أُودع الصفُّ ثمّ سقط الخصمُ لخرج سائقٌ
	// لتوصيلةٍ لم يُدفع أجرُها، **ولو سقط الدينُ على السقف لبقي طلبٌ في
	// الطابور يُعرض على السائقين وهو مرفوض.**
	//
	// **والمستلِمُ دافعاً لا شيءَ هنا** — نقدُه يُقبَض عند التسليم.
	if in.FeePayer == FeePayerMerchant {
		if err := s.chargeMerchantDelivery(ctx, tx, id, merchantID, actorID, fee); err != nil {
			return nil, err
		}
	}

	// **ولحظةُ الميلاد تُقيَّد** — **وسجلُّ حالاتٍ يبدأ من أوّل انتقالٍ لا من
	// الإنشاء يُقرأ طلباً بلا نشأة** (وهي علّةُ الخاصِّ التي أُصلحت).
	// **والفاعلُ صاحبُ المتجر** — هو من أنشأها.
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
		VALUES ($1, '', 'dispatching', $2, 'لدي توصيلة')`, id, actorID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	o, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.publishOrder(o)
	// **وأوّلُ عرضٍ بعد الإيداع لا داخلَه** — **عرضٌ خرج ثمّ ارتدّت المعاملةُ
	// يُوقظ سائقاً لطلبٍ لا وجودَ له.**
	if err := s.OfferNext(ctx, id, nil); err != nil {
		s.logger.Error("التوصيلة: تعذّر أوّلُ عرض", "order", id, "error", err)
	}
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
	// FeePayerMerchant **المتجرُ يدفع** — من محفظته أو ديناً مضبوطاً بسقف.
	FeePayerMerchant = "merchant"
	// FeePayerRecipient **المستلِمُ يدفع** — نقداً بيد السائق عند التسليم.
	FeePayerRecipient = "recipient"
)

// SettingMerchantDeliveryPlatformPercent مفتاحُ نصيب المنصّة — **مكانٌ واحدٌ
// لا نصٌّ يتكرّر**: مفتاحٌ يُكتب بيده في موضعين يفترق أحدُهما بخطأٍ مطبعيّ،
// **فيُقرأ صفراً بصمت** (`GetInt` تردّ الافتراضَ لمفتاحٍ مجهول).
const SettingMerchantDeliveryPlatformPercent = "delivery.merchant_delivery_platform_percent"

// IsMerchantDelivery **أتوصيلةُ متجرٍ هي؟** — يُقرأ من الصفّ لا يُخمَّن.
func (s *Service) IsMerchantDelivery(ctx context.Context, orderID string) bool {
	var kind string
	if err := s.db.QueryRow(ctx,
		`SELECT kind FROM orders WHERE id = $1`, orderID).Scan(&kind); err != nil {
		return false
	}
	return kind == KindMerchantDelivery
}
