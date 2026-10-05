package server

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// handleUploadMedia يستقبل صورة multipart (حقل file + حقل kind)
// ويعيد سجل الوسائط بمساريه — يُربط لاحقاً بالكيان عبر معرفه.
func (s *Server) handleUploadMedia(w http.ResponseWriter, r *http.Request) {
	// الحد أعلى قليلاً من حد الصورة لاستيعاب غلاف multipart
	lim := s.media.MaxBytes(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, lim+64<<10)
	if err := r.ParseMultipartForm(lim); err != nil {
		s.respondErr(w, media.ErrTooLarge)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	defer file.Close()

	m, err := s.media.Save(r.Context(), userIDFrom(r), r.FormValue("kind"), file)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, m)
}

// merchantKinds ما يملك صاحبُ المتجر رفعَه — **ولا شيءَ سواه.**
//
// (طلبُ المالك ٢٠٢٦-٠٨-٠٧: «كلُّ منتجٍ يكون له صورة».)
//
// # ولماذا قائمةٌ بيضاءُ لا حراسةٌ بالدور وحدَها
//
// **نقطةُ الرفع تأخذ النوعَ من الطلب** — فمن ملك الرفعَ ملك كلَّ نوعٍ في
// `validKinds`: **شعارَ المنصة وخلفيّةَ الموقع وصورةَ إثبات تسليم.**
//
// **وحراسةُ الدور تقول «هذا تاجر» ولا تقول «ماذا يرفع».** فالقائمةُ هنا،
// **ومن أضاف نوعاً جديداً للمنصة لا يهبه للتجّار سهواً.**
var merchantKinds = map[string]bool{"menu_item": true, "menu_section": true, "merchant_logo": true}

// handleMerchantUploadMedia رفعُ صورةٍ من بوّابة المتجر.
//
// **والنوعُ يُفحص قبل أن يُمرَّر** — لا بعد أن تُكتب الصورةُ على القرص.
func (s *Server) handleMerchantUploadMedia(w http.ResponseWriter, r *http.Request) {
	lim := s.media.MaxBytes(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, lim+64<<10)
	if err := r.ParseMultipartForm(lim); err != nil {
		s.respondErr(w, media.ErrTooLarge)
		return
	}
	if !merchantKinds[r.FormValue("kind")] {
		s.respondErr(w, errForbidden)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	defer file.Close()

	m, err := s.media.Save(r.Context(), userIDFrom(r), r.FormValue("kind"), file)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, m)
}

// marketKinds **ما يرفعه موظّفُ السوق** — صورةُ صنفٍ أو قسم (قرارُ المالك ٢٠٢٦-١٠-٠٥).
//
// **وقسمُ السوق صورتُه من صنف `banner`** (`SectionsTab`) — **ورفعُ صورةٍ ليس نشرَ
// لافتة**: اللافتةُ تُنشأ ببابها (`/banners`) وهو بقدرة المحتوى.
var marketKinds = map[string]bool{"menu_item": true, "menu_section": true, "banner": true}

// handleMarketUploadMedia **رفعُ صورةٍ من صفحة السوق** — `POST /admin/market/media`.
//
// # ولماذا بابٌ ثانٍ لا توسيعُ `POST /media`
//
// **`POST /media` يأخذ النوعَ من الطلب** — شعارَ المنصّة وخلفيّةَ الموقع —
// **وهو بقدرة المحتوى.** **وجدولُ السياسة قدرةٌ واحدةٌ لكلّ مسار**: من أُعطي
// البابَ العامَّ لأجل صورةِ صنفٍ نال تبديلَ شعار المنصّة. **فبابٌ بقائمةٍ بيضاء
// بقدرة السوق** — كبوّابة المتجر (`handleMerchantUploadMedia`) وإيصالِ المصروف.
func (s *Server) handleMarketUploadMedia(w http.ResponseWriter, r *http.Request) {
	lim := s.media.MaxBytes(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, lim+64<<10)
	if err := r.ParseMultipartForm(lim); err != nil {
		s.respondErr(w, media.ErrTooLarge)
		return
	}
	if !marketKinds[r.FormValue("kind")] {
		s.respondErr(w, errForbidden)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	defer file.Close()

	m, err := s.media.Save(r.Context(), userIDFrom(r), r.FormValue("kind"), file)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, m)
}
