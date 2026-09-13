package server

/*
**ملفُّ التطبيق — يُرفع ويُنزَّل، ولا يمرّ بخطّ الصور.**

(طلبُ المالك ٢٠٢٦-٠٨-٠٨: «حقلٌ لإدخال الرابط من غوغل بلاي أو رفع التطبيق
 بشكلٍ مباشر… إذا كان الموجودُ رابطاً يذهب إلى غوغل بلاي، وإذا ملفّاً ينزل
 بشكلٍ مباشر».)

# ولماذا مسارٌ خاصّ

خطُّ الوسائط يفكّ الصورةَ ويصغّرها ويصنع مصغَّرةً لها — **وملفُّ تطبيقٍ ليس
صورة.** وحدُّ الصور خمسةُ ميغابايت، **وأصغرُ تطبيقٍ يتجاوزه.**

# وما يُفحص

  - **الحجم**: من `app.max_file_mb` — مئةُ ميغابايت افتراضاً، أكبرُ من أيّ
    تطبيقِ توصيلٍ معقول **وأصغرُ من أن يملأ القرصَ برفعةٍ واحدة.**
  - **البصمة**: ملفُّ أندرويد أرشيفُ ZIP — أوّلُ أربعةِ بايتاتٍ `PK\x03\x04`.
    **ومن رفع صورةً باسم `.apk` يُردّ**، فلا يُعرض على الناس زرُّ تنزيلٍ
    يعطيهم ملفّاً لا يُثبَّت.
  - **الامتداد**: `.apk` وحدَه.

**ولا يُفحص التوقيع**: ذاك يحتاج أدواتِ أندرويد على الخادم — **والمنصّةُ
تخدم ما رفعه صاحبُها، لا ما رفعه غريب.**

# والاسمُ يُولَد ولا يُؤخذ

اسمُ الملفّ الذي يرسله المتصفّح **يأتي من جهاز الرافع**، وقد يحمل مساراً أو
محارفَ تخرج من المجلّد. **فيُولَّد اسمٌ جديدٌ ويُحفظ**، ويبقى الاسمُ الأصليُّ
لعرضه وحدَه.
*/

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/release"
)

const (
	// maxAppBytes **احتياطيٌّ لا حدّ** — والحدُّ الفعليُّ في `app.max_file_mb`.
	// (نُقل إلى اللوحة 2026-08-09 بقرار المالك.)
	maxAppBytes = 100 << 20

	// appFileSetting مفتاحُ الإعداد الذي يحمل اسمَ الملفّ المخزَّن.
	appFileSetting = "platform.app_file"

	// appDir مجلّدُ الملفّ داخلَ مجلّد الرفع.
	appDir = "app"
)

var (
	errAppTooLarge = httpx.NewError(http.StatusRequestEntityTooLarge,
		"app_too_large", "errors.app_too_large")
	errAppNotAndroid = httpx.NewError(http.StatusBadRequest,
		"app_not_android", "errors.app_not_android")
	errNoAppFile = httpx.NewError(http.StatusNotFound,
		"app_file_missing", "errors.app_file_missing")
)

// zipMagic بصمةُ أرشيف ZIP — وملفُّ أندرويد أرشيفٌ منه.
var zipMagic = []byte{'P', 'K', 3, 4}

// ══════════════════════════════════════════════════════════════════════
// **وملفٌّ لكلّ تطبيق — لا ملفٌّ واحدٌ للأربعة** (`DLC`، ٢٠٢٦-٠٩-١٣)
// ══════════════════════════════════════════════════════════════════════
//
// **وكان المفتاحُ مثبَّتاً `platform.app_file`** — **يومَ كان «التطبيق»
// واحداً.** **ورفعُ تطبيقِ السائق كان يمحو تطبيقَ الزبون.**
//
// **والمفتاحُ يأتي في الطلب الآن** — **وهذا مكتبُ كتابةٍ في الإعدادات،
// فمفتاحٌ حرٌّ يعني كتابةَ أيِّ إعدادٍ بملفّ.** **فيُحلّ من قائمةٍ
// مغلقة**: مفاتيحُ سجلِّ التوزيع وحدَها، **والقديمُ يبقى مقبولاً**
// لبابِ الموقع القائم.
func appFileTarget(r *http.Request) (string, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("key"))
	if raw == "" {
		// **والغيابُ يعني القديمَ** — **فلا ينكسر رافعٌ منشورٌ يعمل.**
		return appFileSetting, true
	}
	if raw == appFileSetting {
		return raw, true
	}
	for _, a := range release.Apps {
		if raw == release.ApkKey(a.Key) {
			return raw, true
		}
	}
	return "", false
}

// handleUploadAppFile يستقبل ملفَّ التطبيق من الإدارة ويحفظه.
func (s *Server) handleUploadAppFile(w http.ResponseWriter, r *http.Request) {
	target, ok := appFileTarget(r)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	lim := s.settings.GetInt(r.Context(), "app.max_file_mb") << 20
	if lim <= 0 {
		lim = maxAppBytes
	}
	r.Body = http.MaxBytesReader(w, r.Body, lim+64<<10)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		s.respondErr(w, errAppTooLarge)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	defer func() { _ = file.Close() }()

	if !strings.EqualFold(filepath.Ext(header.Filename), ".apk") {
		s.respondErr(w, errAppNotAndroid)
		return
	}

	raw, err := io.ReadAll(io.LimitReader(file, lim+1))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if int64(len(raw)) > lim {
		s.respondErr(w, errAppTooLarge)
		return
	}
	if !bytes.HasPrefix(raw, zipMagic) {
		s.respondErr(w, errAppNotAndroid)
		return
	}

	dir := filepath.Join(s.media.Dir(), appDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		s.respondErr(w, err)
		return
	}
	name := uuid.NewString() + ".apk"
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
		s.respondErr(w, err)
		return
	}

	// **والقديمُ يُحذف بعد نجاح الجديد** — لا قبله: رفعٌ يفشل بعد الحذف
	// **يترك المنصّةَ بلا تطبيقٍ وبلا رسالةٍ تقول لماذا.**
	old := s.settings.GetString(r.Context(), target)
	if err := s.settings.SetInternal(r.Context(), target, name); err != nil {
		_ = os.Remove(filepath.Join(dir, name))
		s.respondErr(w, err)
		return
	}
	if old != "" && old != name {
		_ = os.Remove(filepath.Join(dir, filepath.Base(old)))
	}

	s.audit(r, "platform.app_upload", "settings", target, map[string]any{
		"file": header.Filename, "bytes": len(raw),
	})
	httpx.JSON(w, http.StatusCreated, map[string]any{
		"name":  header.Filename,
		"bytes": len(raw),
	})
}

// handleDeleteAppFile يمسح الملفَّ المخزَّن ويُفرغ الإعداد.
func (s *Server) handleDeleteAppFile(w http.ResponseWriter, r *http.Request) {
	target, ok := appFileTarget(r)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	name := s.settings.GetString(r.Context(), target)
	if name == "" {
		httpx.JSON(w, http.StatusOK, map[string]any{"removed": false})
		return
	}
	if err := s.settings.SetInternal(r.Context(), target, ""); err != nil {
		s.respondErr(w, err)
		return
	}
	_ = os.Remove(filepath.Join(s.media.Dir(), appDir, filepath.Base(name)))
	s.audit(r, "platform.app_delete", "settings", target, nil)
	httpx.JSON(w, http.StatusOK, map[string]any{"removed": true})
}

// handleDownloadApp يخدم الملفَّ لمن ضغط زرَّ التحميل — بلا توثيق.
//
// **واسمُ التنزيل ثابتٌ مقروء**: المخزَّنُ معرّفٌ عشوائيّ، **ومن نزّله
// وجده في مجلّده باسمٍ لا يقول ما هو.**
func (s *Server) handleDownloadApp(w http.ResponseWriter, r *http.Request) {
	name := s.settings.GetString(r.Context(), appFileSetting)
	if name == "" {
		s.respondErr(w, errNoAppFile)
		return
	}
	path := filepath.Join(s.media.Dir(), appDir, filepath.Base(name))
	f, err := os.Open(path)
	if err != nil {
		s.respondErr(w, errNoAppFile)
		return
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		s.respondErr(w, errNoAppFile)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.android.package-archive")
	w.Header().Set("Content-Disposition", `attachment; filename="rahalgo.apk"`)
	http.ServeContent(w, r, "rahalgo.apk", st.ModTime(), f)
}

// ══════════════════════════════════════════════════════════════════════
// **مركزُ التنزيل — بابان عامّان** (`DLC`، ٢٠٢٦-٠٩-١٣)
// ══════════════════════════════════════════════════════════════════════

// handleReleases **حالُ التطبيقات الأربعة — بلا توثيق.**
//
// **ولا حسابَ يُشترَط لتنزيل تطبيق** (قرارُ المالك): **من يريد أن يصير
// زبوناً لا حسابَ له بعد.**
//
// **وما يُرسَل محسوبٌ في `internal/release`** — **موضعٌ واحدٌ تقرؤه
// أربعُ صفحاتٍ**، **وقاعدةُ أولويّةٍ تُكتب في كلٍّ منها تفترق في
// الرابعة.**
func (s *Server) handleReleases(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"apps": release.Resolve(r.Context(), s.releaseStore()),
	})
}

// handleDownloadAppByKey **يخدم أثرَ تطبيقٍ بمفتاحه.**
//
// # وحدُّ الأمان في المفتاح لا في المسار
//
// **ولا اسمَ ملفٍّ يأتي من الطلب أبداً**: **المفتاحُ يُحلّ من قائمةٍ
// مغلقةٍ من أربعة**، **والاسمُ يُقرأ من الإعداد ويُقصَّر إلى قاعدته.**
// **فلا انفلاتَ من المجلَّد ولو كُتب في الإعداد مسارٌ.**
//
// **والحالُ المحسوبةُ هي البوّابة**: **من لا أثرَ له لا يُخدَم** —
// **وتنزيلُ الزبون المباشرُ مقفَلٌ في الحساب نفسِه، فلا يُفتَح من هنا.**
func (s *Server) handleDownloadAppByKey(w http.ResponseWriter, r *http.Request) {
	app, ok := release.Find(chi.URLParam(r, "key"))
	if !ok {
		s.respondErr(w, errNoAppFile)
		return
	}
	pub := release.ResolveOne(r.Context(), s.releaseStore(), app)
	if pub.DownloadURL == "" {
		s.respondErr(w, errNoAppFile)
		return
	}
	name := s.settings.GetString(r.Context(), release.ApkKey(app.Key))
	path := filepath.Join(s.media.Dir(), appDir, filepath.Base(name))
	f, err := os.Open(path)
	if err != nil {
		s.respondErr(w, errNoAppFile)
		return
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		s.respondErr(w, errNoAppFile)
		return
	}
	// **واسمُ التنزيل يقول ما هو ويحمل هويّتَه** — **وأربعةُ تطبيقاتٍ
	// باسم `rahalgo.apk` واحدٍ تختلط في مجلَّد التنزيل.**
	out := release.FileName(pub)
	w.Header().Set("Content-Type", "application/vnd.android.package-archive")
	w.Header().Set("Content-Disposition", `attachment; filename="`+out+`"`)
	// **وأثرٌ موسومٌ بهويّته لا يتبدّل** — فيُخزَّن طويلاً.
	w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
	http.ServeContent(w, r, out, st.ModTime(), f)
}

// releaseStore **جسرٌ إلى حزمة السجلّ** — إعداداتٌ ومجلَّدُ آثار.
func (s *Server) releaseStore() release.Store { return releaseStore{s} }

type releaseStore struct{ s *Server }

func (r releaseStore) GetString(ctx context.Context, key string) string {
	return r.s.settings.GetString(ctx, key)
}
func (r releaseStore) ArtifactDir() string {
	return filepath.Join(r.s.media.Dir(), appDir)
}
