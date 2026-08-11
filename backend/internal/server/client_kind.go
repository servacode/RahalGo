package server

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/identity"
)

// clientKindHeader **الترويسةُ التي يُصرّح بها التطبيقُ عن نفسِه.**
//
// **ولا تُشتقّ من `User-Agent`**: نصٌّ حرٌّ يكتبه كلُّ متصفّحٍ بطريقته
// ويتبدّل مع كلّ إصدار، **ومطابقتُه بأنماطٍ نصّيّةٍ تكسر بلا أن تُخبر.**
// **وترويسةٌ صريحةٌ تُقرأ وتُختبر.**
const clientKindHeader = "X-RahalGo-Client"

// clientKind يضع نوعَ العميل في سياق الطلب.
//
// **وقبل المصادقة عمداً**: أخطرُ نقطةٍ تحتاجه هي الدخولُ نفسُه — وهي
// نقطةٌ عامّةٌ بلا توكن، **فلو وُضع في `RequireAuth` لَما رآه من يدخل.**
//
// **وما لا يُصرّح متصفّح** — انظر `identity.ClientFrom`.
func clientKind(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(
			identity.WithClient(r.Context(), r.Header.Get(clientKindHeader))))
	})
}
