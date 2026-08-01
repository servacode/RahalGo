package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// TestExceptPathsSpaenLivedRequests المهلةُ تقطع الطلبَ الطويل — إلّا المُعفى.
//
// **هذا اختبارُ ما لم يشتكِ.** كانت `middleware.Timeout` تلفّ `/api/v1/ws`
// فتقتل قناةَ البثّ بعد ثلاثين ثانية، **ولا يظهر ذلك في شيء**: الشاشةُ تبقى
// معروضةً بآخر ما جلبت، والجمودُ يبدو هدوءاً. فلا خطأ يُرصد ولا اختبارٌ يحمرّ
// — والعطبُ يُكتشف بمستخدمٍ يقول «لا يظهر إلّا عند التحديث».
//
// وهنا يُرصد صراحةً: معالجٌ ينتظر انتهاء سياقه. **مقطوعٌ حيث تسري المهلة،
// حيٌّ حيث تُعفى.**
func TestExceptPathsSparesLivedRequests(t *testing.T) {
	// معالجٌ يعيش ما دام سياقُه حيّاً — كقناة البثّ تماماً.
	long := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			// **ينصرف بلا كتابة** — كما يفعل معالجُ البثّ: الاتصالُ مخطوفٌ
			// فلا ترويسةَ تُكتب. وعندها يكتب الوسيطُ 504 في مؤجَّله.
			return
		case <-time.After(300 * time.Millisecond):
		}
		w.WriteHeader(http.StatusOK)
	})
	h := exceptPaths(middleware.Timeout(80*time.Millisecond), "/api/v1/ws")(long)

	cases := []struct {
		path string
		want int
		why  string
	}{
		{"/api/v1/ws", http.StatusOK, "قناةُ البثّ مُعفاة — تعيش"},
		{"/api/v1/admin/orders", http.StatusGatewayTimeout, "الطلبُ العاديّ تقطعه المهلة"},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.path, nil))
		if rec.Code != c.want {
			t.Errorf("%s: الرمز %d والمتوقّع %d — %s", c.path, rec.Code, c.want, c.why)
		}
	}
}
