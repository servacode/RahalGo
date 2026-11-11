package orders

// ══════════════════════════════════════════════════════════════════════
// **حمولةُ الطلب بالسماح لا بالمنع** — `D20` · `D21` · `D23`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان يقع
//
// **`publishOrder` كان يبثّ الطلبَ كلَّه** — الكائنَ الداخليَّ نفسَه —
// **إلى غرفة المتجر وغرفة السائق**، **وإلى الزبون بعد محو ثلاثة حقول
// وحدَها.** **والتنقيةُ الحقيقيّةُ في `internal/server`** ولا تبلغها
// هذه الحزمة.
//
// **فقيس على المصدر الحاليّ**: **تسعةَ عشرَ حقلاً محظوراً تصل غرفةَ
// المتجر** · **واثنان وعشرون حقلاً يزيدها البثُّ على `REST`** ·
// **وستّةُ خروقٍ في حمولة الزبون** ومنها **هاتفُ السائق.**
//
// # ولماذا بابٌ واحدٌ لا بابان
//
// **البثُّ ليس قناةً مميّزة**: **من لا يجوز أن يقرأ حقلاً في `REST`
// لا يقرؤه في مقبسٍ حيّ.** **والفرقُ بين القناتين كان عرَضَ `D23`.**
//
// # وبالسماح لا بالمنع
//
// **قائمةُ منعٍ تشيخ**: **حقلٌ جديدٌ في `Order` يتسرّب من نفسه** —
// **وتعديلاتُ الإطلاق ستضيف حقولاً ماليّةً وفروعاً وتسويات.**
//
// **فالحمولةُ تُبنى من مسموحٍ مذكورٍ بالاسم** — **وما لم يُذكَر لا
// يخرج.**
//
// # والأحكامُ من العقد المعتمد
//
// **مصدرُها `OrderPrivacy`** — عقدُ الخصوصيّة المُقرّ (`P-1`):
// `PC-6` و`MD-4` و`RQ-7` وتنقيتا `REST` القائمتان. **ولا حكمَ اخترعتُه
// هنا.** **ونسخةُ العقد في الفحص تبقى مستقلّةً ليقيس بها** — **ومن
// بدّل واحدةً وحدَها رأى المصفوفةَ تسقط.**
//
// # ويسقط مغلقاً
//
// **إن تعذّر بناءُ الحمولة لم يُبثَّ شيء** — **ولا يُرسَل العريضُ
// بديلاً**: **بثٌّ ناقصٌ يُصلَح بتحديثٍ، وتسريبٌ لا يُسحَب.**

import (
	"encoding/json"

	"github.com/servacode/rahalgo/backend/internal/textguard"
)

// Audience الطرفُ الذي تُبنى له الحمولة.
type Audience string

const (
	AudienceCustomer Audience = "customer"
	AudienceMerchant Audience = "merchant"
	AudienceDriver   Audience = "driver"
	AudienceRep      Audience = "rep"
	AudienceOps      Audience = "ops"
	// AudienceMerchantDelivery **صاحبُ المتجر يقرأ توصيلتَه** («لدي توصيلة»).
	//
	// **حقولُ المتجر نفسُها وفوقها ما كتبه هو**: المستلِمُ ورقمُه وعنوانُه
	// ووصفُ الغرض ومن يدفع والأجرة — **بياناتُه لا بياناتُ غيره.** **ولا
	// هويّةَ سائق** — كما في كلّ طلبٍ يراه المتجر. (الخطوة ١٨.)
	AudienceMerchantDelivery Audience = "merchant_delivery"
)

// audienceAllow **ما يجوز لكلّ طرفٍ أن يقرأه** — بوسم `json` لا باسم
// الحقل في Go، **فهو ما يصل الطرفَ الآخرَ فعلا.**
var audienceAllow = map[Audience]map[string]bool{
	AudienceCustomer: {
		"custom_reserved_amount": true,
		"custom_goods_pending":   true,
		"custom_mode":            true,
		"amanah_pickup_lat":      true,
		"amanah_pickup_lng":      true,
		"holder_name":            true,
		"holder_phone":           true,
		"accepted_at":            true,
		"address_text":           true,
		"blocked_reason":         true,
		"cancel_reason":          true,
		"cash_due":               true,
		"closed_at":              true,
		"created_at":             true,
		"custom_agreed_at":       true,
		"custom_fee":             true,
		"custom_goods_amount":    true,
		// **وثمنُ البضاعة بالدولار يراه كما كُتب** (قرارُ المالك ٢٠٢٦-١١-٠٩).
		"custom_goods_usd":         true,
		"custom_request":           true,
		"custom_photos":            true,
		"quote_confirmed_at":       true,
		"quote_confirmed_total":    true,
		"quote_confirmed_version":  true,
		"quote_rejected_at":        true,
		"live_requested_at":        true,
		"dropoff_suggest_lat":      true,
		"dropoff_suggest_lng":      true,
		"quote_version":            true,
		"customer_id":              true,
		"customer_name":            true,
		"customer_phone":           true,
		"delivered_at":             true,
		"delivery_estimate_min":    true,
		"delivery_fee":             true,
		"free_waived":              true,
		"discount":                 true,
		"dispatched_at":            true,
		"driver_assigned":          true,
		"driver_name":              true,
		"fail_reason":              true,
		"id":                       true,
		"items":                    true,
		"items_count":              true,
		"items_preview":            true,
		"kind":                     true,
		"lat":                      true,
		"lng":                      true,
		"merchant_accepts_returns": true,
		"notes":                    true,
		"number":                   true,
		"payment_method":           true,
		"wallet_scope":             true,
		"picked_up_at":             true,
		"prep_minutes":             true,
		"promo_code":               true,
		"proof_taken_at":           true,
		"proof_url":                true,
		"rating":                   true,
		"ready_at":                 true,
		"returned_at":              true,
		"stage":                    true,
		"stage_at":                 true,
		"stages":                   true,
		"status":                   true,
		"subtotal":                 true,
		"to_door_eta_sec":          true,
		"to_store_eta_sec":         true,
		"total":                    true,
		"wallet_paid":              true,
		"zone_id":                  true,
		"zone_name":                true,
		// **أقسامُ طلبه بلا اسم متجرٍ ولا معرّفه** (`P-1`) — الطلبُ بعدّة محطّات وحدَه.
		"sections": true,
	},
	AudienceMerchant: {
		"accepted_at":              true,
		"cancel_reason":            true,
		"closed_at":                true,
		"commission_percent":       true,
		"commission_percent_exact": true,
		"created_at":               true,
		"delivered_at":             true,
		"delivery_estimate_min":    true,
		"dispatched_at":            true,
		"driver_assigned":          true,
		"fail_reason":              true,
		"id":                       true,
		"items":                    true,
		"items_count":              true,
		"items_preview":            true,
		"kind":                     true,
		"merchant_accepts_returns": true,
		"merchant_id":              true,
		"merchant_logo_thumb_url":  true,
		"merchant_name":            true,
		"merchant_net":             true,
		"notes":                    true,
		"number":                   true,
		"picked_up_at":             true,
		"platform_commission":      true,
		"prep_minutes":             true,
		"ready_at":                 true,
		"returned_at":              true,
		"sent_to_merchant_at":      true,
		"stage":                    true,
		"stage_at":                 true,
		"stages":                   true,
		"subtotal":                 true,
		"status":                   true,
		"to_door_eta_sec":          true,
		"to_store_eta_sec":         true,
		// **«قسمك من طلب فيه {n} محلات»** — العددُ لا الأسماء.
		"shared_stops": true,
	},
	AudienceDriver: {
		// **«طلب واتساب» على بطاقته** (قرارُ المالك ٢٠٢٦-١١-٠٥).
		"origin":                       true,
		"custom_fee_source":            true,
		"custom_goods_pending":         true,
		"custom_mode":                  true,
		"amanah_pickup_lat":            true,
		"amanah_pickup_lng":            true,
		"holder_name":                  true,
		"holder_phone":                 true,
		"custom_fee_snapshot":          true,
		"custom_driver_may_change_fee": true,
		"accepted_at":                  true,
		"address_text":                 true,
		"cancel_reason":                true,
		"cash_due":                     true,
		"closed_at":                    true,
		"created_at":                   true,
		"custom_agreed_at":             true,
		"custom_fee":                   true,
		"custom_goods_amount":          true,
		"custom_goods_usd":             true,
		"custom_request":               true,
		"custom_photos":                true,
		// **«لدي توصيلة»** — ما يحمله · ومن يقبض منه · وأتُتبَع النقطة.
		"parcel_note": true,
		// **وأين البضاعةُ بعد طارئ** — للسائق التالي (٢٠٢٦-١٠-٠٢).
		"pickup_note":   true,
		"fee_payer":     true,
		"dropoff_known": true,
		// **ووضعُ التوصيلة وما يقبضه من المستلِم للمتجر** (قرارُ المالك ٢٠٢٦-١٠-١٥).
		"delivery_mode":  true,
		"collect_amount": true,
		// **ومبلغُ الإحضار الحقيقيُّ وموافقةُ المتجر** (قرارُ المالك ٢٠٢٦-١٠-١٦).
		"fetch_amount":            true,
		"fetch_amount_version":    true,
		"fetch_amount_status":     true,
		"amount_currency":         true,
		"quote_confirmed_at":      true,
		"quote_confirmed_total":   true,
		"quote_confirmed_version": true,
		"quote_rejected_at":       true,
		"live_requested_at":       true,
		"entrance_lat":            true,
		"entrance_lng":            true,
		"live_request_count":      true,
		"quote_version":           true,
		"customer_id":             true,
		"customer_name":           true,
		"delivered_at":            true,
		"delivery_estimate_min":   true,
		"delivery_fee":            true,
		"free_waived":             true,
		// **والخصمُ في فاتورته** (قرارُ المالك ٢٠٢٦-١٠-٣٠: «السائق يشوف الفاتورة كاملة»).
		"discount":                 true,
		"dispatched_at":            true,
		"driver_assigned":          true,
		"driver_id":                true,
		"driver_name":              true,
		"driver_phone":             true,
		"driver_to_pickup_m":       true,
		"driver_to_drop_m":         true,
		"fail_reason":              true,
		"id":                       true,
		"items":                    true,
		"items_count":              true,
		"items_preview":            true,
		"kind":                     true,
		"lat":                      true,
		"leg_m":                    true,
		"lng":                      true,
		"merchant_accepts_returns": true,
		"merchant_id":              true,
		"merchant_logo_thumb_url":  true,
		"merchant_name":            true,
		"notes":                    true,
		"number":                   true,
		"offered_driver_name":      true,
		"payment_method":           true,
		"wallet_scope":             true,
		"picked_up_at":             true,
		"prep_minutes":             true,
		"proof_meters":             true,
		"proof_skip_reason":        true,
		"proof_taken_at":           true,
		"proof_url":                true,
		"ready_at":                 true,
		"returned_at":              true,
		"stage":                    true,
		"stage_at":                 true,
		"stages":                   true,
		"status":                   true,
		"subtotal":                 true,
		"to_door_eta_sec":          true,
		"to_store_eta_sec":         true,
		"total":                    true,
		"zone_id":                  true,
		"zone_name":                true,
		// **ومشوارُ إرجاع البضاعة** — مشوارُه هو (قرارُ المالك ٢٠٢٦-١٠-٠٣).
		"return_to":       true,
		"goods_handed_at": true,
		// **وما يدفعه للمتاجر عند الاستلام** (قرارُ المالك ٢٠٢٦-١٠-١٥).
		"pay_merchant":  true,
		"pay_merchants": true,
		// **«خد من المتجر X»** — فرقُ تعديلٍ بعد الاستلام يأخذه الكابتنُ من المتجر (النموذج ٢، B8).
		"take_from_stores": true,
		// **ومن أيّ متجرٍ تابعٍ يشتري** (قرارُ المالك ٢٠٢٦-١٠-١٥) — اسمُه لا نسبتُه.
		"partner_merchant_name": true,
		// **وموقعُ الزبون المباشر** (٢٠٢٦-١٠-١٦) — **غرفةُ السائق غرفةُ المُسنَد وحدَه**
		// (`publishOrder`)، **والطلبُ المبثوثُ لا يحمله أصلاً** (يُلحَق في الردّ وحدَه).
		"customer_live": true,
	},
	AudienceRep: {
		"accepted_at":           true,
		"cancel_reason":         true,
		"closed_at":             true,
		"created_at":            true,
		"delivered_at":          true,
		"delivery_estimate_min": true,
		"delivery_fee":          true,
		"free_waived":           true,
		"id":                    true,
		"items_count":           true,
		"kind":                  true,
		"number":                true,
		"picked_up_at":          true,
		"ready_at":              true,
		"returned_at":           true,
		"stage":                 true,
		"stage_at":              true,
		"stages":                true,
		"status":                true,
		"subtotal":              true,
		"total":                 true,
	},
	AudienceOps: {
		// **وعمولةُ الطلب المسلَّم كما تُعرض في سجلّ اللوحة** (٢٠٢٦-١٠-٣٠) — البثُّ لا يضيّقها.
		"commission_taken":     true,
		"commission_waived":    true,
		"driver_fee":           true,
		"parcel_note":          true,
		"pickup_note":          true,
		"fee_payer":            true,
		"dropoff_known":        true,
		"delivery_mode":        true,
		"collect_amount":       true,
		"fetch_amount":         true,
		"fetch_amount_version": true,
		"fetch_amount_status":  true,
		"amount_currency":      true,
		// **سعرُ الدولار الملقوطُ ومعادلُه** (٢٠٢٦-١١-٠٩) — للمكتب وحدَه.
		"goods_fx_rate":                true,
		"goods_usd_syp":                true,
		"custom_reserved_amount":       true,
		"custom_fee_source":            true,
		"custom_goods_pending":         true,
		"custom_mode":                  true,
		"amanah_pickup_lat":            true,
		"amanah_pickup_lng":            true,
		"holder_name":                  true,
		"holder_phone":                 true,
		"custom_fee_snapshot":          true,
		"custom_driver_may_change_fee": true,
		"accepted_at":                  true,
		"address_text":                 true,
		"blocked_reason":               true,
		"cancel_reason":                true,
		"cash_due":                     true,
		"closed_at":                    true,
		"commission_percent":           true,
		"commission_percent_exact":     true,
		"created_at":                   true,
		"custom_agreed_at":             true,
		"custom_fee":                   true,
		"custom_goods_amount":          true,
		"custom_goods_usd":             true,
		"custom_request":               true,
		"custom_photos":                true,
		"quote_confirmed_at":           true,
		"quote_confirmed_total":        true,
		"quote_confirmed_version":      true,
		"quote_rejected_at":            true,
		"live_requested_at":            true,
		"entrance_lat":                 true,
		"entrance_lng":                 true,
		"dropoff_suggest_lat":          true,
		"dropoff_suggest_lng":          true,
		"live_request_count":           true,
		"quote_version":                true,
		"customer_id":                  true,
		"customer_name":                true,
		"customer_phone":               true,
		"customer_phone_masked":        true,
		"delivered_at":                 true,
		"delivery_estimate_min":        true,
		"delivery_fee":                 true,
		"free_waived":                  true,
		"discount":                     true,
		"dispatched_at":                true,
		"driver_assigned":              true,
		"driver_id":                    true,
		"driver_name":                  true,
		"driver_phone":                 true,
		"driver_seen_at":               true,
		"driver_to_pickup_m":           true,
		"driver_to_drop_m":             true,
		"ended_by":                     true,
		"events":                       true,
		"fail_reason":                  true,
		"fault":                        true,
		"goods_settled_to":             true,
		"id":                           true,
		"items":                        true,
		"items_count":                  true,
		"items_preview":                true,
		"kind":                         true,
		"lat":                          true,
		"leg_m":                        true,
		"lng":                          true,
		"merchant_accepts_returns":     true,
		"merchant_id":                  true,
		"merchant_logo_thumb_url":      true,
		"merchant_name":                true,
		"merchant_phone":               true,
		// **وعنوانُ المتجر ونقطتُه** — قسمُ المتجر على بطاقة اللوحة (٢٠٢٦-١١-٠٣).
		"pickup_address":      true,
		"pickup_lat":          true,
		"pickup_lng":          true,
		"merchant_net":        true,
		"notes":               true,
		"number":              true,
		"offered_driver_name": true,
		"ops_stage_at":        true,
		"ops_stage_late":      true,
		"ops_stage_times":     true,
		"ops_stages":          true,
		"payment_method":      true,
		"wallet_scope":        true,
		"picked_up_at":        true,
		"platform_commission": true,
		"prep_minutes":        true,
		"promo_code":          true,
		"proof_meters":        true,
		"proof_mocked":        true,
		"proof_skip_reason":   true,
		"proof_taken_at":      true,
		"proof_url":           true,
		"rating":              true,
		"ready_at":            true,
		"returned_at":         true,
		"sent_to_merchant_at": true,
		"stage":               true,
		"stage_at":            true,
		"stages":              true,
		"status":              true,
		"subtotal":            true,
		// **ولوحةُ «الطلب مع السائق» للمكتب** — يُجيزها العقد (`privacy_contract.go`)،
		// واللوحةُ تعيد القراءةَ عند كلّ بثّ فتأتيها من ردّ القائمة.
		"door":             true,
		"to_door_eta_sec":  true,
		"to_store_eta_sec": true,
		"total":            true,
		"wallet_paid":      true,
		"zone_id":          true,
		"zone_name":        true,
		// **وأين البضاعةُ ومتى سلّمها السائق** — لبطاقة الطلب (قرارُ المالك ٢٠٢٦-١٠-٠٣).
		"return_to":       true,
		"goods_handed_at": true,
		// **وسطرُ لوحة العمل** (لمن عُرض الطلب · باقي المهلة · سببُ العلوق) —
		// يُجيزه العقد، وبثٌّ بلاه يعرض قديماً حتّى يُعاد الجلب.
		"board": true,
		// **والمسافةُ المقطوعة لكلّ سائق** (٢٠٢٦-١١-٠٥) — تُملأ في ردّ الإدارة (`FillDrives`).
		"drive": true,
		// **وما يدفعه السائقُ للمتاجر عند الاستلام** (٢٠٢٦-١٠-١٥).
		"pay_merchant":  true,
		"pay_merchants": true,
		// **«خد من المتجر X»** — فرقُ تعديلٍ بعد الاستلام يأخذه الكابتنُ من المتجر (النموذج ٢، B8).
		"take_from_stores": true,
		// **والمتجرُ التابعُ في المشتريات ونسبتُه وما دُفع له** (٢٠٢٦-١٠-١٥).
		"partner_merchant_id":              true,
		"partner_merchant_name":            true,
		"partner_commission_percent":       true,
		"partner_commission_percent_exact": true,
		"custom_store_paid":                true,
		// **و«من متجر غير تابع لنا»** — المتجرُ الأوّلُ ومتى ولماذا (٢٠٢٦-١٠-٢٩).
		"ext_store_merchant_name": true,
		"ext_store_at":            true,
		"ext_store_reason":        true,
		// **وموقعُ الزبون المباشر** (٢٠٢٦-١٠-١٦) — يُلحَق في ردّ الطلب الواحد وحدَه.
		"customer_live": true,
		// **وطلبُ الواتساب: أصلُه ومن كتبه ومتجرُ متاجرنا المنسوخ** (قرارُ المالك ٢٠٢٦-١١-٠٥).
		"origin":             true,
		"created_by_name":    true,
		"pickup_merchant_id": true,
		// **صندوقٌ لكلّ متجرٍ في الطلب بعدّة محطّات** (العقدُ §2).
		"parts": true,
		// **وبيد السائق بضاعةُ محطّة** (§5.3) — غرفةُ الطوارئ ونافذةُ التعويض.
		"goods_in_hand": true,
	},
}

// ViewFor حمولةُ الطلب كما يجوز لهذا الطرف أن يقرأها.
//
// **وفارغٌ يعني «لا تبثّ»** — لا «ابثث كلَّ شيء».
func init() {
	md := map[string]bool{}
	for k, v := range audienceAllow[AudienceMerchant] {
		md[k] = v
	}
	for _, k := range []string{"customer_name", "customer_phone", "address_text",
		"parcel_note", "fee_payer", "dropoff_known", "delivery_fee", "cash_due",
		"delivery_mode", "collect_amount",
		// **والمبلغُ الذي يوافق عليه** في «إحضار» (قرارُ المالك ٢٠٢٦-١٠-١٦).
		"fetch_amount", "fetch_amount_version", "fetch_amount_status", "amount_currency"} {
		md[k] = true
	}
	audienceAllow[AudienceMerchantDelivery] = md
}

func ViewFor(a Audience, o *Order) map[string]any {
	allow, ok := audienceAllow[a]
	if o == nil || !ok {
		return nil
	}
	raw, err := json.Marshal(o)
	if err != nil {
		return nil
	}
	var full map[string]any
	if err := json.Unmarshal(raw, &full); err != nil {
		return nil
	}
	out := make(map[string]any, len(allow))
	for k, v := range full {
		if allow[k] {
			out[k] = v
		}
	}
	// **والزبونُ يرى «كابتن رحال غو» لا اسمَ السائق** (قرارُ المالك مساءَ
	// ٢٠٢٦-١٠-٠٢) — **في الخادم لا في الشاشة**: اسمٌ يُرسَل ويُخفى في التطبيق
	// يُقرأ من الشبكة.
	if a == AudienceCustomer {
		if n, ok := out["driver_name"].(string); ok && n != "" {
			out["driver_name"] = CustomerDriverLabel
		}
		// **والتوصيلُ المجانيّ يُرى أجرةً مخفَّضة** (٢٠٢٦-١٠-٢٣) — نسختُه الحاليّة تعرض
		// `custom_fee` أجرةً، **فتُقال له ما يدفعه** لا ما يأخذه السائق.
		if _, ok := out["custom_fee"]; ok && o.CustomFee != nil && o.DeliveryFee < *o.CustomFee {
			out["custom_fee"] = o.DeliveryFee
		}
	}
	// **والسائقُ يرى أقربَ معلمٍ ملصقاً بالعنوان** (قرارُ المالك ٢٠٢٦-١٠-٢١) — في
	// الحقل الذي تعرضه نسختُه الحاليّة، **فلا تحديثَ للتطبيق**. وللزبون والمتجر
	// والمكتب عنوانُه كما كتبه.
	// **والسائقُ يرى أجرتَه كاملةً ولو كان التوصيلُ مجّانيّاً للزبون** (قرارُ المالك
	// ٢٠٢٦-١٠-٢٦) — `delivery_fee` ما دفعه الزبون، فكان يقرأ «الأجرة 0» وهو يقبضها كاملة.
	// والإعفاءُ يبقى في `free_waived` فتقول نسختُه الجديدة «مجانيّة للزبون».
	if a == AudienceDriver && o.FreeWaived > 0 {
		out["delivery_fee"] = o.DeliveryFee + o.FreeWaived
	}
	if a == AudienceDriver && o.NearLandmark != "" {
		if addr, ok := out["address_text"].(string); ok {
			if addr == "" {
				out["address_text"] = o.NearLandmark
			} else {
				out["address_text"] = addr + " · " + o.NearLandmark
			}
		}
	}
	// **والسائقُ يرى الاسمَ الأوّلَ للزبون وحده** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٥) —
	// **في الخادم لا في الشاشة**، فالحمولةُ نفسُها لا تحمل الاسمَ كاملاً. **و«لدي توصيلة»
	// لا تُمسّ**: الاسمُ فيها اسمُ المستلِم (وقد يكون محلّاً) ورقمُه عند السائق أصلاً.
	if a == AudienceDriver && o.Kind != KindMerchantDelivery {
		if _, ok := out["customer_name"]; ok {
			out["customer_name"] = textguard.FirstName(o.CustomerName)
		}
		maskDriverContacts(out)
	}
	return out
}

// maskDriverContacts **ما كتبه الزبونُ بيده يُقرأ للسائق بلا رقم تواصل** (قرارُ المالك
// ٢٠٢٦-١١-٠٥، البند ١ج): الملاحظاتُ ونصُّ الطلب الخاصّ والعنوانُ وملاحظةُ كلّ صنف —
// **فرقمٌ مُخفى في الحديث لا يصل من بطاقة الطلب.** **وبالقراءة وحدَها**: الأصلُ في
// القاعدة للإدارة. **ولا يُمسّ رقمُ صاحب الأمانة** (`holder_phone`): السائقُ يتّصل به
// بقرار المالك ٢٠٢٦-١٠-٣٠.
func maskDriverContacts(out map[string]any) {
	for _, k := range []string{"notes", "custom_request", "address_text"} {
		if v, ok := out[k].(string); ok && v != "" {
			out[k], _ = textguard.MaskContacts(v, "***")
		}
	}
	items, _ := out["items"].([]any)
	for _, x := range items {
		if it, ok := x.(map[string]any); ok {
			if n, ok := it["note"].(string); ok && n != "" {
				it["note"], _ = textguard.MaskContacts(n, "***")
			}
		}
	}
}

// DriverFinishedKeep **ما يبقى عند السائق من طلبٍ انتهى — بالسماح لا بالمنع.**
//
// (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٢: «سجلّ السائق بعد ما ينتهي الطلب ما بيرجع فيه اسم
// الزبون ولا عنوانه ولا نقاطه».) **قائمةُ المنع تشيخ**: حقلٌ جديدٌ يتسرّب من نفسه. **فما
// لم يُذكر هنا لا يخرج** — رقمُ الطلب والمتجرُ والمبالغُ والمسافةُ والتواريخ. **وقائمةٌ
// واحدةٌ لبابين**: سجلُّه (`server/driver_history.go`) وبثُّ غرفته (`publishOrder`).
var DriverFinishedKeep = map[string]bool{
	"id": true, "number": true, "status": true, "kind": true, "custom_mode": true,
	"merchant_name": true, "total": true, "subtotal": true, "delivery_fee": true,
	"free_waived": true, "discount": true, "cash_due": true, "payment_method": true, "wallet_scope": true,
	"custom_goods_amount": true, "custom_goods_usd": true, "custom_fee": true, "fail_reason": true,
	"created_at": true, "picked_up_at": true, "delivered_at": true, "closed_at": true,
	"returned_at": true, "merchant_accepts_returns": true, "items_count": true,
	// **المسافةُ** — من الاستلام إلى الباب، رقماً لا نقطة.
	"leg_m": true,
}

// DriverFinished **أانتهى الطلبُ في حقّ سائقه؟** — أُغلق (أو حالُه نهاية)، **إلّا مشوارَ
// إرجاعٍ لم تُسلَّم بضاعتُه**: الطلبُ في قائمته والطريقُ إلى المكتب أو المتجر.
func DriverFinished(o *Order) bool {
	if o == nil {
		return false
	}
	switch {
	case o.ClosedAt != nil:
	case o.Status == StDelivered, o.Status == StCancelled, o.Status == StRejected,
		o.Status == StFailed, o.Status == StRefunded:
	default:
		return false
	}
	openReturn := o.Status == StFailed && o.ReturnTo != nil && *o.ReturnTo != "" && o.GoodsHandedAt == nil
	return !openReturn
}

// DriverFinishedView **حمولةُ السائق لطلبٍ انتهى** — قائمتُه (`ViewFor`) ثمّ
// `DriverFinishedKeep` فوقها. **وفارغٌ إن تعذّر البناء** (يسقط مغلقاً كأصله).
func DriverFinishedView(o *Order) map[string]any {
	full := ViewFor(AudienceDriver, o)
	if full == nil {
		return nil
	}
	out := make(map[string]any, len(DriverFinishedKeep))
	for k, v := range full {
		if DriverFinishedKeep[k] {
			out[k] = v
		}
	}
	return out
}

// CustomerDriverLabel **ما يراه الزبونُ مكانَ اسم السائق — في كلّ موضع.**
//
// (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢: «ما بدّنا ينعرف اسم السائق، وإذا قدّم شكوى
// فرقمُ الطلب يكفي ونحن نعرف السائق منه».) **والسائقُ يرى اسمَ الزبون** ليتأكّد
// منه عند التسليم — فالحجبُ في اتّجاهٍ واحد.
const CustomerDriverLabel = "كابتن رحال غو"
