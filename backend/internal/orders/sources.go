package orders

// مصادرُ الطلب — **متجرٌ واحدٌ في ذهن الزبون، مطابخُ في دفترنا.**
//
// # القرار
//
//	سقفُ مصدرين          ·  في الإعدادات
//	شرطُ قربٍ بينهما      ·  التكلفةُ في المسافة لا في العدد
//	رسمُ مصدرٍ إضافيّ     ·  صفرٌ داخل نصف القطر، محسوبٌ خارجه
//	ولا سقفَ للأصناف     ·  عائلةٌ تطلب خمسةً من مطبخٍ واحد
//
// **والقيدُ الحقيقيُّ «قريب» لا «كم»**: مصدران متجاوران وقفةٌ زائدةٌ بدقيقتين،
// **ومصدران على طرفَي المدينة رحلتان.** والعددُ وحدَه لا يفرّق بينهما.

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// Sources مصادرُ أصنافٍ، مرتّبةً بالأوّلِ ظهوراً في السلّة.
//
// **والترتيبُ ليس تجميلاً**: الأوّلُ يصير `orders.merchant_id` — **أوّلَ محطّةٍ
// للسائق**. ومن اختاره الزبونُ أوّلاً هو الأقربُ إلى ما يريد، **وطعامُه أولى
// بأن يُحمل أخيراً**… لا: **بل أوّلاً يُستلم لأنه أكثرُ ما في الطلب غالباً**،
// والقربُ يجعل الفرقَ دقائق.
type Sources struct {
	// IDs معرّفاتُ المتاجر بلا تكرار.
	IDs []string
	// FarApart المسافةُ بينها تتجاوز نصفَ القطر القريب.
	FarApart bool
	// MaxMeters أبعدُ ما بين مصدرين.
	MaxMeters float64
}

// SourcesOf يقرأ مصادرَ الأصناف ويقيسُ ما بينها.
//
// **ويُقرأ من `menu_items` لا من السلّة**: السلّةُ في المتصفّح **ولا تعرف
// المصادر أصلاً** — أخفيناها عنها عمداً. **والخادمُ يعرف.**
func (s *Service) SourcesOf(ctx context.Context, items []ItemInput) (*Sources, error) {
	if len(items) == 0 {
		return nil, ErrBadItems
	}
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.MenuItemID)
	}

	// **الترتيبُ بأوّلِ ظهورٍ في السلّة** — لا بترتيب القاعدة العشوائيّ.
	//
	// ولولاه لَتغيّرت المحطّةُ الأولى بين نداءين لنفس السلّة، **فيرى الزبونُ
	// رسماً ثمّ رسماً آخر بلا أن يغيّر شيئاً.**
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT ON (mi.merchant_id) mi.merchant_id::text,
		       array_position($1::uuid[], mi.id)
		FROM menu_items mi
		WHERE mi.id = ANY($1::uuid[])
		ORDER BY mi.merchant_id, array_position($1::uuid[], mi.id)`, ids)
	if err != nil {
		return nil, err
	}
	type row struct {
		id  string
		pos int
	}
	found := []row{}
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.id, &x.pos); err != nil {
			rows.Close()
			return nil, err
		}
		found = append(found, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, ErrBadItems
	}
	// ترتيبٌ بأوّل ظهور — القائمةُ قصيرةٌ دائماً (مصدران أو ثلاثة).
	for i := 1; i < len(found); i++ {
		for j := i; j > 0 && found[j].pos < found[j-1].pos; j-- {
			found[j], found[j-1] = found[j-1], found[j]
		}
	}

	out := &Sources{IDs: make([]string, 0, len(found))}
	for _, x := range found {
		out.IDs = append(out.IDs, x.id)
	}
	if len(out.IDs) < 2 {
		return out, nil
	}

	// **أبعدُ ما بين مصدرين** — لا متوسّطُها.
	//
	// ثلاثةُ مطابخَ اثنان منها متجاوران والثالثُ بعيد **رحلتان لا واحدة**،
	// **والمتوسّطُ يخفي البعيدَ خلف القريبين.**
	if err := s.db.QueryRow(ctx, `
		SELECT COALESCE(max(ST_Distance(a.location, b.location)), 0)
		FROM merchants a JOIN merchants b ON b.id > a.id
		WHERE a.id = ANY($1::uuid[]) AND b.id = ANY($1::uuid[])
		  AND a.location IS NOT NULL AND b.location IS NOT NULL`,
		out.IDs).Scan(&out.MaxMeters); err != nil {
		return nil, err
	}
	out.FarApart = out.MaxMeters > float64(s.sourceProximityM(ctx))
	return out, nil
}

// maxSources سقفُ المصادر — **في الإعدادات لا في الشيفرة** (قرار المالك).
func (s *Service) maxSources(ctx context.Context) int {
	return int(s.settingInt(ctx, "orders.max_sources"))
}

// sourceProximityM نصفُ قطر القرب بالأمتار.
func (s *Service) sourceProximityM(ctx context.Context) int64 {
	return s.settingInt(ctx, "orders.source_proximity_m")
}

// extraSourceFee رسمُ المصدر الإضافيّ — **ويُضاف مرّةً لكلّ مصدرٍ بعد الأوّل.**
//
// **وصفرٌ داخل نصف القطر**: وقفةٌ زائدةٌ بدقيقتين لا تكلّف شيئاً يُذكر،
// **ورسمٌ يُؤخذ بلا تكلفةٍ رسمٌ يُشعر الزبونَ أنه يُعاقَب على اختياره.**
func (s *Service) extraSourceFee(ctx context.Context, src *Sources) int64 {
	if src == nil || len(src.IDs) < 2 || !src.FarApart || s.settings == nil {
		return 0
	}
	return s.settings.GetInt(ctx, "orders.extra_source_fee") * int64(len(src.IDs)-1)
}

// QuoteResult تسعيرةُ سلّةٍ قبل الطلب.
type QuoteResult struct {
	Subtotal int64 `json:"subtotal"`
	// DeliveryFee الرسمُ كاملاً — أساسُ المنطقة زائدَ رسمِ المصادر.
	DeliveryFee int64 `json:"delivery_fee"`
	// BaseFee أساسُ المنطقة وحدَه — **ليُرى الفرقُ ويُفهَم سببُه.**
	//
	// رقمٌ واحدٌ يرتفع بلا تفسير يُقرأ زيادةً بلا سبب، **ورقمان يقولان
	// «هذا للمنطقة وهذا لأنك اخترت من مكانين» يُقرآن حساباً.**
	BaseFee int64 `json:"base_fee"`
	// SourcesFee ما زِيد لأجل المصادر الإضافية — وصفرٌ حين لا يكلّف.
	SourcesFee int64 `json:"sources_fee"`
	Total      int64 `json:"total"`
	// Sources عددُ المطابخ، وسقفُها، وهل تجاوزته.
	Sources    int  `json:"sources"`
	MaxSources int  `json:"max_sources"`
	TooMany    bool `json:"too_many_sources"`
	// FarApart المصادرُ متباعدة — **وهو سببُ الرسم لا عددُها.**
	FarApart bool `json:"far_apart"`
}

// Quote يحسب ما سيدفعه الزبون **قبل أن يدفع** — ولا يُنشئ شيئاً.
//
// **ويُعيد التجاوزَ علَماً لا خطأً**: السلّةُ تعرض الحدَّ وتُعطّل زرَّ الدفع،
// **ورسالةُ خطأٍ على كلّ ضغطةِ حرفٍ في السلّة تُتعب ولا تُفيد.** والردُّ
// الصريح يبقى عند الإنشاء.
func (s *Service) Quote(ctx context.Context, items []ItemInput, lat, lng float64) (*QuoteResult, error) {
	out := &QuoteResult{MaxSources: s.maxSources(ctx)}
	if len(items) == 0 {
		return out, nil
	}

	src, err := s.SourcesOf(ctx, items)
	if err != nil {
		return nil, err
	}
	out.Sources = len(src.IDs)
	out.FarApart = src.FarApart
	out.TooMany = out.Sources > out.MaxSources

	// **والتسعيرُ بالمعادلة نفسِها التي عند الإنشاء** — `priceItems`.
	//
	// **ولو حُسب هنا بحسبةٍ ثانية لَافترقتا يوماً**: يرى الزبونُ رقماً في
	// السلّة ويُحاسَب بغيره، **وهو أسوأُ ما يقع في شاشة دفع.**
	_, subtotal, err := s.priceItems(ctx, items)
	if err != nil {
		return nil, err
	}
	out.Subtotal = subtotal

	if z, err := s.DeliveryAt(ctx, lat, lng); err == nil {
		out.BaseFee = z.Fee
	} else {
		// **خارجَ التغطية ليس خطأً في التسعيرة** — الرسمُ يبقى صفراً ويُردّ
		// الطلبُ عند الإنشاء بـ`out_of_zone`. **وسلّةٌ تنهار لأن الدبوسَ لم
		// يُوضع بعد سلّةٌ لا تُستعمل.**
		out.BaseFee = 0
	}
	out.SourcesFee = s.extraSourceFee(ctx, src)
	out.DeliveryFee = out.BaseFee + out.SourcesFee
	out.Total = out.Subtotal + out.DeliveryFee
	return out, nil
}

// ZoneCharge منطقةُ التسليم ورسمُها — **مصدرُ الحقيقة الواحد.**
type ZoneCharge struct {
	ID          string
	Name        string
	DeliveryFee int64
	MinOrder    int64
}

// ZoneAt المنطقةُ التي يقع فيها هذا الدبوس — **وأقربُها مركزاً حين تتداخل.**
//
// # لماذا في موضعٍ واحد
//
// كان الاستعلامُ مكتوباً مرّتين: في إنشاء الطلب وفي التسعيرة. **وهما يتّفقان
// اليومَ ويفترقان يوماً** — يُضاف شرطٌ في أحدهما (منطقةٌ تُغلق ليلاً، رسمٌ
// يتغيّر بالمسافة) فتقول السلّةُ رقماً ويُحاسَب الزبونُ بغيره. **ورقمٌ يظهر
// عند الدفع غيرَ الذي رآه في السلّة يُفقد الثقةَ بالتسعيرة كلِّها.**
//
// (ملاحظةُ المالك ٢٠٢٦-٠٨-٠٤: «قيمُ التوصيل يجب أن تأتي من مكانٍ واحدٍ بكلّ
// المشروع — من غير المعقول أن يكون هناك أكثرُ من مكانٍ لأجرة التوصيل».)
//
// **والرسمُ الكاملُ ليس هذا وحدَه**: يُضاف إليه رسمُ المصدر الإضافيّ
// (`extraSourceFee`) في الموضعين. **وهذه قاعدةُ المنطقة، تلك قاعدةُ التعدّد.**
func (s *Service) ZoneAt(ctx context.Context, lat, lng float64) (ZoneCharge, error) {
	var z ZoneCharge
	err := s.db.QueryRow(ctx, `
		SELECT id::text, name, delivery_fee, min_order FROM delivery_zones
		WHERE active AND ST_DWithin(center, ST_SetSRID(ST_MakePoint($2,$1),4326)::geography, radius_m)
		ORDER BY ST_Distance(center, ST_SetSRID(ST_MakePoint($2,$1),4326)::geography)
		LIMIT 1`, lat, lng).Scan(&z.ID, &z.Name, &z.DeliveryFee, &z.MinOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return z, ErrOutOfZone
	}
	return z, err
}

// DeliveryCharge أجرةُ التوصيل ومنطقتُها — **مصدرُ الحقيقة الواحد.**
type DeliveryCharge struct {
	ZoneCharge
	// Fee الأجرةُ النافذة — **رقمٌ مقطوعٌ من الإعدادات، لا من عمود المنطقة.**
	Fee int64
}

// DeliveryAt أجرةُ التوصيل لهذا الدبّوس — **والمنطقةُ تغطيةٌ لا تسعير.**
//
// # لماذا في موضعٍ واحد
//
// الأجرةُ تُحسب مرّتين: في التسعيرة (قبل الطلب) وعند الإنشاء (لحظتَه).
// **ولو حُسبت بحسبتين لَقالت السلّةُ رقماً ويُحاسَب الزبونُ بغيره** — وهو
// أسوأُ ما يقع في شاشة دفع.
//
// (ملاحظةُ المالك ٢٠٢٦-٠٨-٠٤: «قيمُ التوصيل يجب أن تأتي من مكانٍ واحدٍ بكلّ
// المشروع».)
//
// # والدوائرُ تقول «إلى أين» لا «بكم»
//
// **حدُّ التغطية غيرُ الأجرة.** خارجَ الدوائر يُرفض الطلبُ بـ`out_of_zone`،
// **وداخلَها الأجرةُ واحدةٌ للجميع** — رقمٌ مقطوعٌ في الإعدادات.
//
// **وعمودُ `delivery_zones.delivery_fee` لم يعد يُقرأ**: بقي في القاعدة ولا
// يُعرض في الشاشة، **فلا حقلٌ يَعِد بأثرٍ لا يقع.**
func (s *Service) DeliveryAt(ctx context.Context, lat, lng float64) (DeliveryCharge, error) {
	var out DeliveryCharge
	z, err := s.ZoneAt(ctx, lat, lng)
	if err != nil {
		return out, err
	}
	out.ZoneCharge = z
	out.Fee = pricing.DeliveryFee(ctx, s.settings)
	return out, nil
}

// OrderMarginSQL هامشُ طلبٍ من لقطات بنوده — **معادلةٌ واحدةٌ لمن يقرؤها.**
//
// # لماذا تُصدَّر
//
// تحسبها التسويةُ لتقيّد نصيبَ المندوب، **وتحسبها شاشتُه لتعرضه.** ولو كُتبت
// مرّتين لَافترقتا يوماً — **فيرى المندوبُ رقماً ويُقيَّد له غيرُه**، وهي
// عائلةُ الخلل التي طاردناها في التوصيل والمخالفات وأقسام السوق.
//
// # ومن اللقطتين لا من `menu_items` اليوم
//
// **سعرُ البيع وسعرُ الشراء محفوظان في البند لحظةَ الطلب.** ولو قُرئا من
// القائمة اليومَ **لَتغيّر هامشُ طلبٍ مضى** كلَّما غُيّر سعرٌ أو هامش —
// فيُعاد حسابُ عمولةِ مندوبٍ قُبضت.
//
// `orderExpr` تعبيرٌ يعطي معرّفَ الطلب في السياق المحيط.
func OrderMarginSQL(orderExpr string) string {
	return `COALESCE((SELECT sum((oi.unit_price - oi.merchant_price) * oi.qty)
	                  FROM order_items oi WHERE oi.order_id = ` + orderExpr + `), 0)`
}
