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

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/obligations"
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
	// **ولها أجرتُها الخاصّة إن ضُبطت** (قرارُ المالك ٢٠٢٦-١٠-٠٥) — انظر
	// `merchantDeliveryFee`.
	fee := s.merchantDeliveryFee(ctx, tx, zone.DistanceM)
	pct := s.platformDeliveryPercent(ctx, tx)
	driverFee := DriverFeeAfterShare(fee, pct)

	snap, err := s.snapshotNow(ctx, tx)
	if err != nil {
		return "", err
	}

	// **وقدرةُ المتجر تُفحص الآن ولا يُؤخذ شيء** — والخصمُ عند قبول المكتب.
	//
	// **ومن لا يقدر يُقال له الآن لا بعد ساعة**: توصيلةٌ تنتظر المكتبَ ثمّ
	// تُردّ لأنّ المحفظةَ فارغةٌ تُضيّع وقتَ الاثنين. **والقاعدةُ قاعدةُ الخصم
	// نفسُها** (`merchantCanCover`) — ويُعاد فحصُها بقفلٍ عند القبول.
	if in.FeePayer == FeePayerMerchant {
		can, err := s.merchantCanPay(ctx, tx, merchantID, fee)
		if err != nil {
			return "", err
		}
		if !can {
			return "", ErrDeliveryCreditExhausted
		}
	}

	// **ومن يدفع يحدّد أين يقع المال**: المتجرُ ⇐ يُخصم أو يُقيَّد ديناً
	// مضبوطاً؛ المستلِمُ ⇐ نقدٌ بيد السائق عند التسليم.
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
		                    snap_platform_delivery_percent, dropoff_known)
		VALUES ('merchant_delivery', $1, $2,
		        ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography,
		        $5, $6, $7, $8,
		        'pending', 'cash', 0, $9, $10,
		        $9, $11, $12,
		        $13, $14, $15, $16, $17, $18)
		RETURNING id::text`,
		merchantID, in.AddressText, in.Lat, in.Lng,
		in.RecipientName, in.RecipientPhone, nullIfEmpty(in.ParcelNote), in.FeePayer,
		fee, driverFee, cashDue, strings.TrimSpace(in.DriverNote), // **و`notes` NOT NULL** — فراغٌ لا NULL
		snap.MerchantCommissionPercent, snap.RepCommissionPercent,
		snap.CommissionSource, snap.ActivationOrders, pct, dropoffKnown).Scan(&id)
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
	// FeePayerMerchant **المتجرُ يدفع** — من محفظته أو ديناً مضبوطاً بسقف.
	FeePayerMerchant = "merchant"
	// FeePayerRecipient **المستلِمُ يدفع** — نقداً بيد السائق عند التسليم.
	FeePayerRecipient = "recipient"
	// FeePayerMerchantCash **المتجرُ يدفع نقداً** بيد السائق عند الاستلام —
	// (نصُّ المالك ٢٠٢٦-١٠-٠١: «لازم في أنا نقدي»). لا محفظةَ ولا دين.
	FeePayerMerchantCash = "merchant_cash"
)

// SettingPlatformDeliveryPercent مفتاحُ حصّة المنصّة من أجرة التوصيل — **لكلّ
// أنواع الطلبات** (قرارُ المالك ٢٠٢٦-١٠-٠٤، الإعدادات البند ١). **مكانٌ واحدٌ
// لا نصٌّ يتكرّر**: مفتاحٌ يُكتب بيده في موضعين يفترق أحدُهما بخطأٍ مطبعيّ،
// **فيُقرأ صفراً بصمت** (`GetInt` تردّ الافتراضَ لمفتاحٍ مجهول).
const SettingPlatformDeliveryPercent = "delivery.platform_percent"

// platformDeliveryPercentMax **لا أجرَ للسائق عند المئة** — فالحدُّ تسعون.
const platformDeliveryPercentMax = 90

// platformDeliveryPercent **الحصّةُ النافذةُ الآن** — تُلقَط على الطلب لحظةَ
// إنشائه (`snap_platform_delivery_percent`) فلا تمسّ طلباً قائماً.
func (s *Service) platformDeliveryPercent(ctx context.Context, q dbtx.Querier) int64 {
	if s.settings == nil {
		return 0
	}
	pct := s.settings.On(q).GetInt(ctx, SettingPlatformDeliveryPercent)
	if pct < 0 {
		return 0
	}
	if pct > platformDeliveryPercentMax {
		return platformDeliveryPercentMax
	}
	return pct
}

// DriverFeeAfterShare **أجرُ السائق بعد حصّة المنصّة** — الحسبةُ الواحدةُ للأنواع
// الثلاثة. **والكسرُ للسائق**: الحصّةُ تُقرَّب إلى أسفل.
func DriverFeeAfterShare(fee, pct int64) int64 {
	if fee <= 0 || pct <= 0 {
		return fee
	}
	return fee - fee*pct/100
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
// وسقفُه صفرٌ اختار «المستلِمُ يدفع» ولم يُردّ.
type MerchantDeliveryQuote struct {
	Fee           int64  `json:"fee"`
	ZoneName      string `json:"zone_name"`
	WalletBalance int64  `json:"wallet_balance"`
	CreditLimit   int64  `json:"credit_limit"`
	CreditOwed    int64  `json:"credit_owed"`
	// MerchantCanPay **أيكفي المتجرَ أن يدفع هو؟** — محفظةً أو ديناً تحت سقفه.
	// **ويُحسب بقاعدة `chargeMerchantDelivery` نفسِها** لا بتقديرٍ ثانٍ.
	MerchantCanPay bool `json:"merchant_can_pay"`
}

// QuoteMerchantDelivery **أجرةُ النقطة ومقدرةُ المتجر** — بحرّاس الإنشاء نفسِها.
func (s *Service) QuoteMerchantDelivery(ctx context.Context, merchantID string,
	lat, lng float64, hasPoint bool) (*MerchantDeliveryQuote, error) {
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
	q := &MerchantDeliveryQuote{Fee: s.merchantDeliveryFee(ctx, s.db, zone.DistanceM), ZoneName: zone.Name}
	if err := s.db.QueryRow(ctx, `
		SELECT COALESCE((SELECT balance FROM wallets w WHERE w.user_id = m.owner_user_id), 0),
		       m.delivery_credit_limit
		  FROM merchants m WHERE m.id = $1`, merchantID).
		Scan(&q.WalletBalance, &q.CreditLimit); err != nil {
		return nil, err
	}
	_, owed, err := s.MerchantDeliveryCredit(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	q.CreditOwed = owed
	q.MerchantCanPay = s.merchantDebtOpen(ctx, s.db) ||
		merchantCanCover(q.WalletBalance, q.CreditOwed, q.CreditLimit, q.Fee)
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

// merchantDeliveryFee **أجرةُ التوصيلة** — الإنشاءُ وعرضُ السعر من هنا وحدَه.
//
// **مفتاحُها إن كان فوق الصفر**، وإلّا فالقاعدةُ العامّة (`delivery.fee`
// ومعها التسعيرُ بالمسافة إن اشتعل). **وصفرٌ «لا أجرةَ خاصّة» لا «مجّاناً»**:
// افتراضُه صفرٌ فلا يتبدّل شيءٌ يومَ يُنشَر.
func (s *Service) merchantDeliveryFee(ctx context.Context, q dbtx.Querier, distanceM float64) int64 {
	st := s.settings.On(q)
	if own := st.GetInt(ctx, SettingMerchantDeliveryFee); own > 0 {
		return own
	}
	return pricing.DeliveryFeeAt(ctx, st, distanceM)
}

// SettingMerchantDebtOpen **دينُ «لدي توصيلة» بلا سقف** (قرارُ المالك ٢٠٢٦-١٠-٠٥:
// «نقبل الرقم السالب وبعدها المتجر يسوّي الحساب… صاحب المتجر ما بيختفي، ونحنا ما
// رح نتركه يسحب على كيفه»). **مشتعلٌ افتراضاً**: ما لم تكفِ المحفظةُ قُيّد ديناً
// على المتجر بلا حدّ، **ويُتابَع في «الديون»**. ومطفأً يعود سقفُ كلّ متجرٍ حاكماً.
const SettingMerchantDebtOpen = "delivery.merchant_debt_open"

// merchantDebtOpen **أيُقيَّد الدينُ بلا سقف؟**
func (s *Service) merchantDebtOpen(ctx context.Context, q dbtx.Querier) bool {
	return s.settings != nil && s.settings.On(q).GetBool(ctx, SettingMerchantDebtOpen)
}

// merchantCanCover **أيقدر المتجرُ أن يدفع؟** — محفظةً تكفي، أو ديناً تحت سقفه.
//
// **قاعدةُ `chargeMerchantDelivery` نفسُها** — تُقرأ في عرض السعر وفي الإنشاء
// بلا قفل، **ويُعاد حسابُها بقفلٍ عند القبول** حيث يقع الخصم.
func merchantCanCover(balance, owed, limit, fee int64) bool {
	return fee <= 0 || balance >= fee || owed+fee <= limit
}

// merchantCanPay **قدرةُ المتجر الآن** — قراءةٌ بلا قفلٍ ولا خصم.
func (s *Service) merchantCanPay(ctx context.Context, q dbtx.Querier, merchantID string, fee int64) (bool, error) {
	var balance, limit int64
	if err := q.QueryRow(ctx, `
		SELECT COALESCE((SELECT balance FROM wallets w WHERE w.user_id = m.owner_user_id), 0),
		       m.delivery_credit_limit
		  FROM merchants m WHERE m.id = $1`, merchantID).Scan(&balance, &limit); err != nil {
		return false, err
	}
	if s.merchantDebtOpen(ctx, q) {
		return true, nil
	}
	owed, err := obligations.Balance(ctx, q, obligations.PartyMerchant, merchantID)
	if err != nil {
		return false, err
	}
	return merchantCanCover(balance, owed, limit, fee), nil
}
