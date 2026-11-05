package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/servacode/rahalgo/backend/internal/realtime"
)

// ══════════════════════════════════════════════════════════════════════
// **ما تكتبه اللوحةُ يُسمَع في الجيب — بلا إعادة تشغيل**
// ══════════════════════════════════════════════════════════════════════
//
// (شكوى المالك ٢٠٢٦-٠٨-١٨: «في مشكلةُ التحديث اللحظيّ بالتطبيق… أيُّ
//
//	تعديلٍ من لوحة الأدمن فوراً يُطبَّق حتّى ولو الزبونُ فاتحٌ التطبيق،
//	ما يلزم يحدّث أو يعيد تشغيل التطبيق».)
//
// # ما كان يقع
//
// **`hub.Publish` لم يكن يُنادى إلّا للطلبات والمحفظة والإشعارات
// وطابور السائقين.** فمن غيّر سعراً أو صنفاً أو بانراً أو أجرةَ توصيلٍ
// **بثَّ لا شيء** — والتطبيقُ المفتوحُ يعرض ما جلبه عند إقلاعه.
//
// **وشاشةٌ تُظهر سعراً قديماً لا تقول إنّه قديم**: يطلب بسعرٍ رآه
// ويُحاسَب بغيره.
//
// # ولماذا وسيطٌ لا نداءٌ في كلّ معالج
//
// **نقاطُ الكتابة في اللوحة والمتجر عشرات** — ومن بثَّ في كلٍّ منها
// بيده **نسي واحدةً**، وهي التي يُشتكى منها بعد شهر. **والوسيطُ لا
// يُنسى**: يمرّ به كلُّ ما دخل المجموعتين، وما يُضاف غداً يُبَثّ بلا
// أن يعلم كاتبُه.
//
// # ولا يُبَثّ إلّا بعد نجاح
//
// **القراءةُ لا تُبَثّ** — `GET` لم يغيّر شيئاً، وبثٌّ عندها يوقظ كلَّ
// هاتفٍ كلَّما فتح موظّفٌ شاشة. **والفشلُ لا يُبَثّ**: ٤٠٠ أو ٤٠٣ لم
// يكتبا، **وإشارةٌ بعدهما تجعل كلَّ التطبيقات تُعيد الجلبَ بلا سبب.**
//
// # وإشارةٌ لا حمولة
//
// انظر `realtime.TopicCatalog` — تقول «تغيّر شيء» ويُعيد كلُّ تطبيقٍ
// جلبَ ما يخصّه من نقطته، **فتبقى قواعدُ ما يُرى في الخادم.**
func (s *Server) announceWrites(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead ||
			r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		rec := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(rec, r)
		// **و`0` يعني أنّ المعالجَ كتب بلا ترويسة** — وهي ٢٠٠ عند Go.
		if quietWrite(r) {
			return
		}
		if code := rec.Status(); code == 0 || code < 400 {
			s.hub.Publish(realtime.TopicCatalog, map[string]any{"type": "catalog"})
		}
	})
}

// quietWrites **كتاباتٌ لا تمسّ ما يراه الزبون — فلا تُبَثّ.**
//
// «شيلو من القائمة» (قرارُ المالك ٢٠٢٦-١١-٠٥) قائمةٌ خاصّةٌ بالمتجر نفسِه: **بثُّها يوقظ كلَّ
// تطبيقٍ مفتوحٍ ليُعيد جلبَ الفهرس بلا شيءٍ تغيّر فيه.** وتُطابَق بنمط المسار بعد التوجيه.
var quietWrites = []string{
	"/merchant/stores/{id}/delivery-recipients/{phone}",
}

// quietWrite **أهذه الكتابةُ من الصامتة؟** — نمطُ المسار يُقرأ بعد أن وجّهه chi.
func quietWrite(r *http.Request) bool {
	rc := chi.RouteContext(r.Context())
	if rc == nil {
		return false
	}
	p := rc.RoutePattern()
	for _, q := range quietWrites {
		if strings.HasSuffix(p, q) {
			return true
		}
	}
	return false
}
