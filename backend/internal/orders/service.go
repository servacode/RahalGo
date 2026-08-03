// Package orders محرك الطلبات: الإنشاء بتسعير خادمي، آلة الحالات، والتسويات المالية.
package orders

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/pricing"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// Publisher واجهة البث الحي — ينشر المحرك تحديثات الطلبات عبرها.
type Publisher interface {
	Publish(topic string, event any)
}

type noopPublisher struct{}

func (noopPublisher) Publish(string, any) {}

// Notifier واجهة الإشعارات المركزية — يستعملها المحرك ليُعلم أطراف الطلب بما يقع.
// تُحقن بعد الإنشاء لأن خدمة الإشعارات تُبنى في الخادم.
type Notifier interface {
	Notify(ctx context.Context, in notifications.Input)
	NotifyRoles(ctx context.Context, roles []string, in notifications.Input)
	// NotifyOps مكتب المنصة كاملاً — يستعمله الراصد للتصعيد الباقي
	NotifyOps(ctx context.Context, in notifications.Input)
}

type Service struct {
	db       *pgxpool.Pool
	identity *identity.Service
	wallet   *wallet.Service
	cashbox  *cashbox.Service
	pub      Publisher
	notify   Notifier
	logger   *slog.Logger
	// settings قواعدُ العمل التي يملك المالك ضبطها من اللوحة.
	//
	// **اختيارية**: بلا حقنٍ يعمل المحرّك بسلوكه الافتراضي، فاختبارات التسويات
	// لا تحتاج مخزناً لتفحص حساباً. ومن يحتاجها يحقنها عند الإقلاع.
	settings *settings.Store
}

// SetNotifier يحقن خدمة الإشعارات بعد بناء الخادم (لا إشعارات قبلها).
func (s *Service) SetNotifier(n Notifier) { s.notify = n }

// SetSettings يحقن مخزن الإعدادات (يُنادى مرّة عند الإقلاع).
func (s *Service) SetSettings(st *settings.Store) { s.settings = st }

func NewService(db *pgxpool.Pool, identitySvc *identity.Service, walletSvc *wallet.Service,
	cashboxSvc *cashbox.Service, pub Publisher, logger *slog.Logger) *Service {
	if pub == nil {
		pub = noopPublisher{}
	}
	return &Service{db: db, identity: identitySvc, wallet: walletSvc, cashbox: cashboxSvc, pub: pub, logger: logger}
}

// publishOrder يبث ملخص الطلب لغرفة العمليات ولموضوع المتجر المعني.
func (s *Service) publishOrder(o *Order) {
	if o == nil {
		return
	}
	event := map[string]any{"type": "order", "order": o}
	s.pub.Publish("ops", event)
	s.pub.Publish("merchant:"+o.MerchantID, event)

	// **وحدثُ الزبون بلا مصدر.**
	//
	// حُجب اسمُ المتجر في التصفّح وفي الطلبات وفي التقييمات وفي الإشعارات —
	// **وبقي في البثّ الحيّ**: نسخةٌ كاملةٌ من الطلب تُرسَل إلى قناة الزبون
	// في كلّ انتقال. **ولا تظهر في شاشةٍ فتُنتبَه**، بل تُقرأ في أدوات
	// المتصفّح — وهي أهدأُ مواضع التسريب وأبقاها.
	//
	// **وحجبٌ في أربعة مواضعَ من خمسة ليس حجباً.** (وُجد في فحص البثّ نفسِه،
	// ٢٠٢٦-٠٨-٠٣.)
	cust := *o
	cust.MerchantID, cust.MerchantName, cust.MerchantLogoThumb = "", "", nil
	s.pub.Publish("customer:"+o.CustomerID, map[string]any{"type": "order", "order": &cust})

	// **السائقُ الذي يحمل الطلب يعلم بما يجري فيه.**
	//
	// كان يُستثنى من البثّ كلِّه: تُسند إليه العملياتُ طلباً فلا يعلم حتى
	// يُحدّث الصفحة، وتُلغيه فيمضي إلى عنوانٍ لا طلبَ فيه.
	if o.DriverID != nil && *o.DriverID != "" {
		s.pub.Publish("driver:"+*o.DriverID, event)
	}

	// **وإشارةٌ للطابور — بلا حمولة.**
	//
	// كان الطلبُ ينزل إلى الطابور ولا يعلم به أحد: تبقى شاشةُ السائق كما هي
	// حتى يُحدّثها بيده. **والطابورُ الذي لا يُرى حتى يُحدَّث ليس طابوراً حيّاً،
	// هو قائمةٌ يتذكّر أحدٌ أن ينظر إليها.**
	//
	// وتُرسَل عند الدخول وعند الخروج معاً: من أخذه واحدٌ يجب أن يختفي عن
	// شاشات الباقين، **وإلّا ضغطوا عليه فردَّهم «سبقك غيرُك»** — وهو ردٌّ صحيح
	// يُغني عنه عرضٌ صحيح.
	if queueAffecting(o.Status) {
		s.pub.Publish(topicDriverQueue, map[string]any{"type": "order"})
	}
}

// topicDriverQueue نسخةٌ محلّية من `realtime.TopicDriverQueue`.
//
// **حرفياً لا استيراداً**: هذه الحزمة تنشر عبر واجهة `Publisher` ولا تعرف من
// ينفّذها — واستيرادُ `realtime` هنا يربط محرّك الطلبات بتنفيذِ بثٍّ بعينه.
// (وهو ما تفعله `"ops"` و`"merchant:"` فوق أصلاً.)
//
// **والانحرافُ بينهما يُمسك باختبار** لا بالانتباه: `TestDriverQueueTopic`.
const topicDriverQueue = "drivers:queue"

// queueAffecting أيغيّر هذا الوضعُ ما يراه السائقون في الطابور؟
//
// **دخولٌ وخروج لا دخولٌ وحده**: `dispatching` تُدخله، والباقيةُ تُخرجه —
// أخذَه سائقٌ أو أُلغي أو رُفض أو فشل.
func queueAffecting(status string) bool {
	switch status {
	case StDispatching, StAssigned, StCancelled, StRejected, StFailed:
		return true
	}
	return false
}

// Create ينشئ طلباً كاملاً: تحقق المتجر، تسعير خادمي للأصناف والخيارات،
// منطقة التسليم ورسمها، كود الخصم، ثم الدفع (نقدي/محفظة/مختلط) — كله ذرّياً.
func (s *Service) Create(ctx context.Context, actorID string, actorRoles []string, in CreateInput, ip string) (*Order, error) {
	// **والمصدرُ يُستنتج من الأصناف لا يُرسَل.**
	//
	// الزبونُ لا يرى المتاجر ولا يعرف معرّفاتها — **يطلب أصنافاً ونحن نعرف من
	// أين نشتريها.** وطلبُ `merchant_id` منه يعني أن التطبيق يعرفه، **ومعرّفٌ
	// يعرفه التطبيقُ معرّفٌ يُقرأ من الشبكة** فيُفتح به اسمُ المتجر.
	//
	// **ويبقى مقبولاً إن أُرسل**: الطلبُ الهاتفيّ تكتبه العملياتُ وهي ترى
	// المتاجر، **وواجهةٌ تختفي فجأةً تُسقط شاشةً لم تُحدَّث بعد.**
	var sources *Sources
	if len(in.Items) > 0 {
		var err error
		if sources, err = s.SourcesOf(ctx, in.Items); err != nil {
			return nil, err
		}
		// **والسقفُ يُفحص هنا لا في المتصفّح.**
		//
		// السلّةُ لا تعرف المصادر — أخفيناها عنها عمداً — **فلا تملك أن
		// تمنع.** والخادمُ يعرف، **وهو الموضعُ الذي لا يُلتفّ عليه.**
		if len(sources.IDs) > s.maxSources(ctx) {
			return nil, ErrTooManySources
		}
		if in.MerchantID == "" {
			in.MerchantID = sources.IDs[0]
		}
	}
	if len(in.Items) == 0 || in.AddressText == "" || in.MerchantID == "" {
		return nil, ErrBadItems
	}
	// طريقتان لا ثلاث: نقداً عند الاستلام، أو من المحفظة كاملاً.
	//
	// أُلغي «المختلط» بقرار المالك: كان يدفع ما في المحفظة ويترك الباقي نقداً،
	// فيصير للطلب الواحد مصدرا دفعٍ ومسارا تسويةٍ ومسارا استرجاع — تعقيدٌ في
	// أخطر جزء من النظام مقابل راحةٍ لا يطلبها أحد.
	switch in.PaymentMethod {
	case "", "cash":
		in.PaymentMethod = "cash"
	case "wallet":
	default:
		return nil, ErrBadItems
	}

	// الزبون: معرف مباشر أو رقم هاتف (طلب هاتفي — يُنشأ الحساب إن لزم)
	customerID := in.CustomerID
	if customerID == "" {
		if in.CustomerPhone == "" {
			return nil, ErrBadItems
		}
		u, err := s.identity.EnsureUserWithRole(ctx, actorID, in.CustomerPhone, "customer", ip)
		if err != nil {
			return nil, err
		}
		customerID = u.ID
	}

	// **المتجرُ يستقبل الآن — دوامُه لا حالتُه وحدَها.**
	//
	// كان الفحصُ «فعّالٌ وغيرُ مغلقٍ طارئاً» ولا ينظر في الدوام أصلاً. **فمن
	// فتح الصفحةَ قبل الإغلاق بدقيقة، أو تركها مفتوحةً ساعةً، يطلب من متجرٍ
	// مغلق** — فيصل الطلبُ ولا أحدَ يحضّره، ويبقى معلّقاً حتى تنتبه العمليات.
	//
	// **والشاشةُ كانت تعرف وتُخفيه**: `open_now` محسوبةٌ في كل صفحةٍ منذ
	// البداية، **ولا يفحصها إلّا العرض.** وحارسٌ في الشاشة وحدَها ليس حارساً:
	// كلُّ من يعرف النقطةَ يتجاوزه، **وكلُّ صفحةٍ قديمةٍ تتجاوزه بلا قصد.**
	var openNow bool
	err := s.db.QueryRow(ctx,
		`SELECT `+OpenNowSQL+` FROM merchants m WHERE m.id = $1`,
		in.MerchantID).Scan(&openNow)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !openNow {
		return nil, ErrMerchantClosed
	}

	// التسعير الخادمي للأصناف والخيارات (لقطة ثابتة)
	items, subtotal, err := s.priceItems(ctx, in.Items)
	if err != nil {
		return nil, err
	}

	// **لا طلب قبل توثيق واتساب.**
	//
	// الرقمُ الوهميّ يعني سائقاً يقف أمام بابٍ لا أحد فيه، وطلباً نقدياً لا
	// يُقبض، ومتجراً حضّر بضاعةً لا تُستلَم. والخسارة تقع على ثلاثة أطراف لا
	// على من كتب الرقم.
	//
	// **والتحقق هنا لا في الواجهة وحدها**: الواجهة تُخفي الزرّ، والخادم يمنع
	// الفعل. ومن يستطيع أن ينادي النقطة مباشرةً لا يوقفه إخفاءُ زرّ.
	if s.settings != nil && s.settings.GetBool(ctx, "customers.require_whatsapp") {
		var verified bool
		if err := s.db.QueryRow(ctx,
			`SELECT whatsapp_verified_at IS NOT NULL FROM users WHERE id = $1`,
			customerID).Scan(&verified); err != nil {
			return nil, err
		}
		if !verified {
			return nil, ErrWhatsAppRequired
		}
	}

	// منطقة التسليم من الدبوس
	var zoneID, zoneName string
	var deliveryFee, minOrder int64
	err = s.db.QueryRow(ctx, `
		SELECT id, name, delivery_fee, min_order FROM delivery_zones
		WHERE active AND ST_DWithin(center, ST_SetSRID(ST_MakePoint($2,$1),4326)::geography, radius_m)
		ORDER BY ST_Distance(center, ST_SetSRID(ST_MakePoint($2,$1),4326)::geography)
		LIMIT 1`, in.Lat, in.Lng).Scan(&zoneID, &zoneName, &deliveryFee, &minOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOutOfZone
	}
	if err != nil {
		return nil, err
	}
	// **لا حدّ أدنى للطلب في هذه المنصة** (قرار المالك، ٢٠٢٦-٠٨-٠١).
	//
	// رسمُ التوصيل يُؤخذ كاملاً من الزبون مهما كانت قيمة طلبه، فالمنصة لا تخسر
	// على الطلب الصغير ولا شأن لها بقيمته. والمنصات التي تفرض حدّاً أدنى إنما
	// تفرضه لأنها تُموّل جزءاً من التوصيل — وهذه لا تفعل.
	//
	// وكان قبل الإلغاء يُفرض **الأعلى** بين حدّ المنطقة وحدّ المتجر بينما تعرض
	// السلّة حدّ المنطقة وحده: فطلبٌ يتجاوز ما رآه صاحبُه يرسب في ما لم يره،
	// ورسالةُ الرفض تنسبه إلى المنطقة وهي قد قبلته. **حدٌّ خفيّ أسوأ من حدٍّ عالٍ.**
	_ = minOrder

	// **ورسمُ المصدر الإضافيّ — مجّانيٌّ حين لا يكلّف، محسوبٌ حين يكلّف.**
	//
	// وقفةٌ زائدةٌ بدقيقتين لا تكلّف شيئاً يُذكر، **ورسمٌ يُؤخذ بلا تكلفةٍ رسمٌ
	// يُشعر الزبونَ أنه يُعاقَب على اختياره.**
	//
	// **ويُضاف قبل الخصم**: كودٌ يُصفّر التوصيلَ يُصفّره كلَّه — **وأن يبقى
	// جزءٌ منه بعد «توصيلٌ مجّانيّ» وعدٌ يُخلَف.**
	deliveryFee += s.extraSourceFee(ctx, sources)

	// كود الخصم
	var promoID *string
	var discount int64
	promoCode := strings.TrimSpace(strings.ToUpper(in.PromoCode))
	if promoCode != "" {
		promoID, discount, err = s.validatePromo(ctx, promoCode, customerID, subtotal, &deliveryFee)
		if err != nil {
			return nil, err
		}
	}

	total := subtotal - discount + deliveryFee
	if total < 0 {
		total = 0
	}

	// توزيع الدفع
	var walletPaid int64
	switch in.PaymentMethod {
	case "wallet":
		walletPaid = total
	}
	cashDue := total - walletPaid

	// الإنشاء الذرّي
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var orderID string
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, address_text, dropoff, zone_id,
			payment_method, subtotal, delivery_fee, discount, total, wallet_paid, cash_due,
			promo_code, notes, created_by)
		VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($5,$4),4326)::geography, $6,
			$7, $8, $9, $10, $11, $12, $13, NULLIF($14,''), $15, $16)
		RETURNING id`,
		customerID, in.MerchantID, in.AddressText, in.Lat, in.Lng, zoneID,
		in.PaymentMethod, subtotal, deliveryFee, discount, total, walletPaid, cashDue,
		promoCode, in.Notes, actorID).Scan(&orderID)
	if err != nil {
		return nil, err
	}

	for _, it := range items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO order_items (order_id, menu_item_id, name, unit_price,
			                         merchant_price, merchant_id, qty, note, options)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			orderID, it.MenuItemID, it.Name, it.UnitPrice, it.MerchantPrice,
			it.MerchantID, it.Qty, it.Note,
			marshalOptions(it.Options)); err != nil {
			return nil, err
		}
	}

	if promoID != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO promo_redemptions (promo_id, order_id, user_id) VALUES ($1, $2, $3)`,
			*promoID, orderID, customerID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE promo_codes SET used_count = used_count + 1 WHERE id = $1`, *promoID); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
		VALUES ($1, '', 'pending', $2, '')`, orderID, actorID); err != nil {
		return nil, err
	}

	// خصم المحفظة **داخل معاملة الإنشاء** لا بعدها.
	//
	// كان الخصم يقع بعد `Commit`: فإن فشل (رصيد تغيّر لحظياً) يُلغى الطلب بقيد
	// تعويضي. والنتيجة طلبٌ ملغى يبقى في سجل الزبون وفي عدّاد المتجر وفي مقياس
	// «الملغي» عند المندوب — عن طلبٍ **لم يوجد تجارياً قط**. وإن فشل التعويض
	// نفسه بقي الطلب معلّقاً بلا دفع.
	//
	// وهو نفس خلل R-02 من بابٍ آخر: مالٌ خارج المعاملة يحتاج تعويضاً بدل أن
	// يتراجع معها. والعلاج نفسه: `ApplyTx` على معاملة الإنشاء — يفشل الخصم
	// فيتراجع الطلب كلّه، ولا يبقى أثر لطلبٍ لم يُدفع.
	if walletPaid > 0 {
		if _, err := s.wallet.ApplyTx(ctx, tx, customerID, -walletPaid, "order_payment",
			// **بلا ملاحظة.**
			//
			// كانت `دفع طلب #1d448c90` — **ثمانيةُ أحرفٍ من معرّفٍ داخليّ** لا
			// يعرفها صاحبُ المحفظة ولا يجدها في شيء. ورقمُ الطلب لم يكن قد
			// وُلد بعد في هذه اللحظة (يُولّده الإدراج)، **فكُتب ما هو متاحٌ لا
			// ما هو مفيد.**
			//
			// والمرجعُ (`ref`) يحمل معرّفَ الطلب، وكشفُ الحساب يترجمه إلى رقمه
			// المقروء بضمّه إلى الجدول. **فالملاحظةُ هنا تكرارٌ لعنوان الحركة
			// بلفظٍ أسوأ** — وحذفُها يُظهر الرقمَ الصحيح مكانها.
			orderID, "", &actorID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	created, err := s.GetByID(ctx, orderID)
	if err == nil {
		s.publishOrder(created)
		s.notifyCreated(ctx, created)
	}
	return created, err
}

// priceItems يجلب الأسعار الحقيقية من القائمة ويتحقق من الخيارات وقيود المجموعات.
// priceItems يُسعّر الأصنافَ — **ولا يُقيّدها بمتجرٍ واحد.**
//
// كان يشترط `merchant_id = $2` **لأن الطلبَ كان من مصدرٍ واحد**. وبعد أن صار
// من مصدرين **صار الشرطُ يُسقط نصفَ السلّة صامتاً**: يُقرأ الصنفُ فلا يوجد،
// فيُردّ `ErrBadItems` — **ورسالةٌ تقول «صنفٌ غير صالح» عن صنفٍ صالحٍ تماماً.**
//
// **والحارسُ لم يسقط بل انتقل**: `SourcesOf` تفحص السقفَ قبل أن يُسعَّر شيء.
func (s *Service) priceItems(ctx context.Context, inputs []ItemInput) ([]OrderItem, int64, error) {
	items := make([]OrderItem, 0, len(inputs))
	var subtotal int64
	rule := pricing.RuleFrom(ctx, s.settings)

	for _, in := range inputs {
		if in.Qty < 1 || in.Qty > 50 {
			return nil, 0, ErrBadItems
		}
		// **سعرُ البيع يُحسب هنا لا يُقرأ.**
		//
		// `menu_items.price` قد يكون قديماً: **الهامشُ إعدادٌ يملك المالكُ
		// تغييرَه في أيّ لحظة**، ولو قُرئ العمودُ المخزَّن لَبِيع بسعر الأمس
		// حتى يُعاد حسابُ ألف صنف. **وحسبةٌ عند الطلب لا تتخلّف أبداً.**
		//
		// والتجاوزان يُقرآن مع الصنف في استعلامٍ واحد: **الصنفُ يرث تصنيفَه،
		// والتصنيفُ يرث العام.**
		var it OrderItem
		var available bool
		// **والطبقةُ الوسطى قسمُ المنصة لا تصنيفُ المتجر.**
		//
		// الشاورما تُسعَّر كشاورما **سواءٌ جاءت من مطعمٍ أو مشاوٍ أو
		// كافتيريا**. وتصنيفُ المتجر يصف بائعَه لا سلعتَه، **وهامشٌ يتبع
		// البائعَ يجعل الصنفَ الواحد بسعرين.**
		var itemMargin, sectionMargin *int64
		err := s.db.QueryRow(ctx, `
			SELECT mi.id, mi.name, mi.merchant_price, mi.available,
			       mi.margin_override, ps.margin_override, mi.merchant_id::text
			FROM menu_items mi
			LEFT JOIN platform_sections ps ON ps.id = mi.platform_section_id
			WHERE mi.id = $1`, in.MenuItemID).
			Scan(&it.MenuItemID, &it.Name, &it.MerchantPrice, &available,
				&itemMargin, &sectionMargin, &it.MerchantID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, ErrBadItems
		}
		if err != nil {
			return nil, 0, err
		}
		if !available {
			return nil, 0, ErrItemUnavailable
		}
		it.UnitPrice = rule.SalePrice(it.MerchantPrice, itemMargin, sectionMargin)
		it.Qty = in.Qty
		it.Note = in.Note
		it.Options = []OptionSnapshot{}

		// الخيارات: يجب أن تتبع مجموعات هذا الصنف وتحترم أدنى/أقصى اختيار
		type groupRule struct{ min, max, chosen int }
		rules := map[string]*groupRule{}
		gRows, err := s.db.Query(ctx,
			`SELECT id, min_select, max_select FROM modifier_groups WHERE item_id = $1`, in.MenuItemID)
		if err != nil {
			return nil, 0, err
		}
		for gRows.Next() {
			var id string
			var mn, mx int
			if err := gRows.Scan(&id, &mn, &mx); err != nil {
				gRows.Close()
				return nil, 0, err
			}
			rules[id] = &groupRule{min: mn, max: mx}
		}
		gRows.Close()

		for _, optID := range in.OptionIDs {
			var groupID, groupName, optName string
			var delta int64
			var optAvailable bool
			err := s.db.QueryRow(ctx, `
				SELECT g.id, g.name, o.name, o.price_delta, o.available
				FROM modifier_options o
				JOIN modifier_groups g ON g.id = o.group_id
				WHERE o.id = $1 AND g.item_id = $2`, optID, in.MenuItemID).
				Scan(&groupID, &groupName, &optName, &delta, &optAvailable)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, 0, ErrBadItems
			}
			if err != nil {
				return nil, 0, err
			}
			if !optAvailable {
				return nil, 0, ErrItemUnavailable
			}
			rules[groupID].chosen++
			// **الفارقُ يُضاف إلى السعرين معاً.**
			//
			// «جبنة إضافية +٢٠٠٠» ثمنٌ يقبضه المتجرُ كما يقبض أصلَ الصنف،
			// **فإضافتُه إلى سعر البيع وحدَه تجعله هامشاً لنا** — ونربح على
			// ما لم نضف إليه شيئاً، **ويُحرم المتجرُ ثمنَ ما صنعه.**
			it.MerchantPrice += delta
			it.UnitPrice += delta
			it.Options = append(it.Options, OptionSnapshot{ID: optID, Group: groupName, Name: optName, PriceDelta: delta})
		}
		for _, r := range rules {
			if r.chosen < r.min || r.chosen > r.max {
				return nil, 0, ErrBadItems
			}
		}

		subtotal += it.UnitPrice * int64(it.Qty)
		items = append(items, it)
	}
	return items, subtotal, nil
}

// validatePromo يتحقق من كل قواعد الكود ويعيد الخصم (وقد يصفّر رسم التوصيل).
func (s *Service) validatePromo(ctx context.Context, code, customerID string, subtotal int64, deliveryFee *int64) (*string, int64, error) {
	var id, kind string
	var value, minOrder int64
	var firstOnly, oncePerUser, active bool
	var maxUses *int
	var usedCount int
	var expiresAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT id, kind, value, min_order, first_order_only, once_per_user,
		       max_uses, used_count, expires_at, active
		FROM promo_codes WHERE code = $1`, code).
		Scan(&id, &kind, &value, &minOrder, &firstOnly, &oncePerUser,
			&maxUses, &usedCount, &expiresAt, &active)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, ErrInvalidPromo
	}
	if err != nil {
		return nil, 0, err
	}

	switch {
	case !active,
		expiresAt != nil && time.Now().After(*expiresAt),
		maxUses != nil && usedCount >= *maxUses,
		subtotal < minOrder:
		return nil, 0, ErrInvalidPromo
	}

	if oncePerUser {
		var used bool
		if err := s.db.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM promo_redemptions WHERE promo_id = $1 AND user_id = $2)`,
			id, customerID).Scan(&used); err != nil {
			return nil, 0, err
		}
		if used {
			return nil, 0, ErrInvalidPromo
		}
	}
	if firstOnly {
		var hasOrders bool
		if err := s.db.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM orders WHERE customer_id = $1
				AND status NOT IN ('cancelled','rejected','failed'))`,
			customerID).Scan(&hasOrders); err != nil {
			return nil, 0, err
		}
		if hasOrders {
			return nil, 0, ErrInvalidPromo
		}
	}

	var discount int64
	switch kind {
	case "percent":
		discount = subtotal * value / 100
	case "fixed":
		discount = min64(value, subtotal)
	case "free_delivery":
		*deliveryFee = 0
	}
	return &id, discount, nil
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// sourceOf مصدرُ الأصناف — **ويلزم أن يكون واحداً.**
//
// # لماذا واحدٌ اليوم
//
// الطلبُ متعدّدُ المصادر قرارٌ مُتَّفقٌ عليه (سقفُ اثنين وشرطُ قرب)، **لكنّه
// يغيّر التوصيلَ والتسوية معاً**: مسارُ سائقٍ إلى بابين، ورسمٌ إضافيّ،
// ومستحقّان لمتجرين. **وفتحُه قبل أن يُبنى ذلك كلُّه يُنتج طلباتٍ لا يعرف
// المحرّكُ كيف يسوّيها.**
//
// **فيُردّ صراحةً لا يُقبل صامتاً**: من طلب من مصدرين يُقال له، **ولا يُترك
// طلبٌ نصفُه في مطبخٍ ونصفُه في آخر بلا من يجمعهما.**
func (s *Service) sourceOf(ctx context.Context, items []ItemInput) (string, error) {
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.MenuItemID)
	}
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT merchant_id::text FROM menu_items WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	found := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		found = append(found, id)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		// **صنفٌ لا وجودَ له** — والخطأُ يُقال باسمه لا بـ«لا مصدر».
		return "", ErrBadItems
	}
	return "", ErrMultiSource
}
