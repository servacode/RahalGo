package server

import (
	"context"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
)

// ══════════════════════════════════════════════════════════════════════
// **رقمٌ له متجرٌ لا يُفتح عليه متجرٌ آخر** (قرارُ المالك ٢٠٢٦-١٠-١٣)
// ══════════════════════════════════════════════════════════════════════
//
// «عند فتح متكرر لمتجر يقبل الفتح بشكل عادي — لازم يعطي: الرقم مستخدم من
// قبل متجر آخر، لا يمكن فتح حساب بهذا الرقم، يرى تسجيل الدخول.» **ولا فرعَ
// ثانياً من اللوحة أيضاً** («نتركها لاحقاً»).
//
// **وكان تحذيراً للمكتب وحدَه** (`DuplicatePhone`) — والموافقةُ التلقائيّة
// تمرّ عليه فيُفتح المتجرُ ويُضاف إلى الحساب القائم.

// ErrPhoneHasStore **الرقمُ لمتجرٍ قائم أو لطلبِ فتحٍ ينتظر.**
var ErrPhoneHasStore = httpx.NewError(http.StatusConflict, "phone_has_store", "errors.phone_has_store")

// phoneHasStore **أللرقم متجرٌ قائم — رقمَ متجرٍ أو رقمَ صاحبه — أو طلبُ فتحٍ
// مفتوح؟** و`exceptLead` طلبٌ يُستثنى (المُحوَّلُ نفسُه).
func phoneHasStore(ctx context.Context, q dbtx.Querier, raw, exceptLead string) (bool, error) {
	phone, ok := identity.NormalizePhone(raw)
	if !ok {
		return false, nil
	}
	var taken bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM merchants m
		                LEFT JOIN users u ON u.id = m.owner_user_id
		                WHERE m.phone = $1 OR u.phone = $1)
		    OR EXISTS (SELECT 1 FROM merchant_leads l
		                WHERE l.phone = $1 AND l.status IN ('new', 'needs_info')
		                  AND l.id::text <> $2)`, phone, exceptLead).Scan(&taken)
	return taken, err
}
