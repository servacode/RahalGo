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
)

// Audience الطرفُ الذي تُبنى له الحمولة.
type Audience string

const (
	AudienceCustomer Audience = "customer"
	AudienceMerchant Audience = "merchant"
	AudienceDriver   Audience = "driver"
	AudienceRep      Audience = "rep"
	AudienceOps      Audience = "ops"
)

// audienceAllow **ما يجوز لكلّ طرفٍ أن يقرأه** — بوسم `json` لا باسم
// الحقل في Go، **فهو ما يصل الطرفَ الآخرَ فعلا.**
var audienceAllow = map[Audience]map[string]bool{
	AudienceCustomer: {
		"accepted_at":              true,
		"address_text":             true,
		"blocked_reason":           true,
		"cancel_reason":            true,
		"cash_due":                 true,
		"closed_at":                true,
		"created_at":               true,
		"custom_agreed_at":         true,
		"custom_fee":               true,
		"custom_goods_amount":      true,
		"custom_request":           true,
		"customer_id":              true,
		"customer_name":            true,
		"customer_phone":           true,
		"delivered_at":             true,
		"delivery_estimate_min":    true,
		"delivery_fee":             true,
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
	},
	AudienceMerchant: {
		"accepted_at":              true,
		"cancel_reason":            true,
		"closed_at":                true,
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
		"prep_minutes":             true,
		"ready_at":                 true,
		"returned_at":              true,
		"sent_to_merchant_at":      true,
		"stage":                    true,
		"stage_at":                 true,
		"stages":                   true,
		"status":                   true,
		"to_door_eta_sec":          true,
		"to_store_eta_sec":         true,
	},
	AudienceDriver: {
		"accepted_at":              true,
		"address_text":             true,
		"cancel_reason":            true,
		"cash_due":                 true,
		"closed_at":                true,
		"created_at":               true,
		"custom_agreed_at":         true,
		"custom_fee":               true,
		"custom_goods_amount":      true,
		"custom_request":           true,
		"customer_id":              true,
		"customer_name":            true,
		"delivered_at":             true,
		"delivery_estimate_min":    true,
		"delivery_fee":             true,
		"dispatched_at":            true,
		"driver_assigned":          true,
		"driver_id":                true,
		"driver_name":              true,
		"driver_phone":             true,
		"driver_to_pickup_m":       true,
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
	},
	AudienceRep: {
		"accepted_at":           true,
		"cancel_reason":         true,
		"closed_at":             true,
		"created_at":            true,
		"delivered_at":          true,
		"delivery_estimate_min": true,
		"delivery_fee":          true,
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
		"accepted_at":              true,
		"address_text":             true,
		"blocked_reason":           true,
		"cancel_reason":            true,
		"cash_due":                 true,
		"closed_at":                true,
		"commission_percent":       true,
		"created_at":               true,
		"custom_agreed_at":         true,
		"custom_fee":               true,
		"custom_goods_amount":      true,
		"custom_request":           true,
		"customer_id":              true,
		"customer_name":            true,
		"customer_phone":           true,
		"delivered_at":             true,
		"delivery_estimate_min":    true,
		"delivery_fee":             true,
		"discount":                 true,
		"dispatched_at":            true,
		"driver_assigned":          true,
		"driver_id":                true,
		"driver_name":              true,
		"driver_phone":             true,
		"driver_to_pickup_m":       true,
		"ended_by":                 true,
		"events":                   true,
		"fail_reason":              true,
		"fault":                    true,
		"goods_settled_to":         true,
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
		"merchant_net":             true,
		"notes":                    true,
		"number":                   true,
		"offered_driver_name":      true,
		"ops_stage_at":             true,
		"ops_stage_late":           true,
		"ops_stage_times":          true,
		"ops_stages":               true,
		"payment_method":           true,
		"picked_up_at":             true,
		"platform_commission":      true,
		"prep_minutes":             true,
		"promo_code":               true,
		"proof_meters":             true,
		"proof_mocked":             true,
		"proof_skip_reason":        true,
		"proof_taken_at":           true,
		"proof_url":                true,
		"rating":                   true,
		"ready_at":                 true,
		"returned_at":              true,
		"sent_to_merchant_at":      true,
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
	},
}

// ViewFor حمولةُ الطلب كما يجوز لهذا الطرف أن يقرأها.
//
// **وفارغٌ يعني «لا تبثّ»** — لا «ابثث كلَّ شيء».
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
	return out
}
