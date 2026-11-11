package server

import "time"

// أسبابُ «بدّهن متابعة» — ما يحتاج المتجرُ من مندوبه كلمةً.
const (
	// FollowUpNoItems لا صنفَ معروضاً في قائمته.
	FollowUpNoItems = "no_items"
	// FollowUpConditions شروطُ التفعيل ناقصة (دوام أو موقع أو صور أصناف).
	FollowUpConditions = "conditions"
	// FollowUpNoOrders فعّالٌ وله أصنافٌ ولم يُسلَّم له طلبٌ منذ أسبوع.
	FollowUpNoOrders = "no_orders"
)

// followUpIdleDays مهلةُ «ما إجاه طلب» بالأيام.
const followUpIdleDays = 7

// repFollowUpReasons أسبابُ متابعة متجرٍ واحد، وفارغةٌ إن لم يحتج.
//
// الشروطُ الناقصة بمنطق الشارات نفسِه: الهدفُ «pending» ثمّ دوامٌ وموقعٌ وصورُ أصناف.
// و«ما إجاه طلب» لا يُحكم به على متجرٍ فُتح قبل أقلّ من أسبوع، ولا على موقوفٍ، ولا على
// خالي الأصناف (له سببُه الأوّل).
func repFollowUpReasons(status, goalState string, hasHours, hasLocation bool,
	photoItems int, minItems int64, itemsCount int,
	lastDelivered *time.Time, joinedDay string, now time.Time) []string {

	out := []string{}
	if status == "suspended" {
		return out
	}
	if itemsCount == 0 {
		out = append(out, FollowUpNoItems)
	}
	if goalState == "pending" && (!hasHours || !hasLocation || int64(photoItems) < minItems) {
		out = append(out, FollowUpConditions)
	}
	if status == "active" && itemsCount > 0 {
		cut := now.AddDate(0, 0, -followUpIdleDays)
		young := false
		if d, err := time.Parse("2006-01-02", joinedDay); err == nil && d.After(cut) {
			young = true
		}
		if !young && (lastDelivered == nil || lastDelivered.Before(cut)) {
			out = append(out, FollowUpNoOrders)
		}
	}
	return out
}
