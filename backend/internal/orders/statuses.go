package orders

// آلة حالات الطلب (PLAN.md §6.1) — كل انتقال مسموح به لأدوار محددة فقط،
// وكل ما عداه مرفوض. الانتقالات النهائية تُغلق الطلب وتُطلق التسويات المالية.

const (
	StPending     = "pending"
	StAccepted    = "accepted"
	StPreparing   = "preparing"
	StDispatching = "dispatching"
	StAssigned    = "assigned"
	StAtPickup    = "at_pickup"
	StPickedUp    = "picked_up"
	StOnTheWay    = "on_the_way"
	StAtDropoff   = "at_dropoff"
	StDelivered   = "delivered"
	StRejected    = "rejected"
	StCancelled   = "cancelled"
	StFailed      = "failed"
	StRefunded    = "refunded"
)

// transition يعرّف انتقالاً مسموحاً: إلى أي حالة، ومن أي الأدوار.
type transition struct {
	To    string
	Roles []string // الأدوار المخولة (admin دائماً مخول ضمنياً)
}

var opsRoles = []string{"ops"}
var merchantOps = []string{"merchant", "ops"}
var driverOps = []string{"driver", "ops"}

// allowedTransitions خارطة الآلة الكاملة.
var allowedTransitions = map[string][]transition{
	StPending: {
		{StAccepted, merchantOps},
		{StRejected, merchantOps},
		{StCancelled, []string{"customer", "ops"}},
	},
	StAccepted: {
		{StPreparing, merchantOps},
		{StCancelled, opsRoles},
	},
	StPreparing: {
		{StDispatching, opsRoles}, // طلب سائق
		{StCancelled, opsRoles},
	},
	StDispatching: {
		{StAssigned, driverOps}, // قبول سائق أو إسناد يدوي
		{StCancelled, opsRoles},
	},
	StAssigned: {
		{StAtPickup, driverOps},
		{StDispatching, driverOps}, // فك الإسناد وإعادة الطلب
		{StCancelled, opsRoles},
	},
	StAtPickup: {
		{StPickedUp, driverOps},
		{StCancelled, opsRoles},
	},
	StPickedUp: {
		{StOnTheWay, driverOps},
	},
	StOnTheWay: {
		{StAtDropoff, driverOps},
	},
	StAtDropoff: {
		{StDelivered, driverOps},
		{StFailed, driverOps}, // زبون لا يرد / يرفض الاستلام
	},
	StDelivered: {
		{StRefunded, []string{}}, // أدمن فقط
	},
}

// terminal الحالات النهائية — تُغلق الطلب.
func terminal(status string) bool {
	switch status {
	case StDelivered, StRejected, StCancelled, StFailed, StRefunded:
		return true
	}
	return false
}

// refundable الحالات النهائية التي تعيد المدفوع من المحفظة تلقائياً.
func refundOnEnter(status string) bool {
	switch status {
	case StRejected, StCancelled, StFailed, StRefunded:
		return true
	}
	return false
}

// canTransition يتحقق من شرعية الانتقال لهذه الأدوار.
func canTransition(from, to string, roles []string) bool {
	for _, r := range roles {
		if r == "admin" {
			// الأدمن مخول بكل الانتقالات المعرفة في الخارطة
			for _, t := range allowedTransitions[from] {
				if t.To == to {
					return true
				}
			}
			return false
		}
	}
	for _, t := range allowedTransitions[from] {
		if t.To != to {
			continue
		}
		for _, allowed := range t.Roles {
			for _, r := range roles {
				if r == allowed {
					return true
				}
			}
		}
	}
	return false
}
