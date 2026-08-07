package server

// **ملفُّ التطبيق: رفعٌ وتنزيلٌ وأولويّة.**
//
// (طلبُ المالك ٢٠٢٦-٠٨-٠٨: «حقلٌ لإدخال الرابط من غوغل بلاي أو رفع التطبيق
//  بشكلٍ مباشر… إذا كان الموجودُ رابطاً يذهب إلى غوغل بلاي، وإذا ملفّاً
//  ينزل بشكلٍ مباشر».)
//
// ما تحرسه:
//
//	الامتداد   `.apk` وحدَه
//	البصمة     أرشيفُ ZIP — ومن رفع صورةً باسم `.apk` يُردّ
//	الأولويّة  الرابطُ يسبق الملفَّ، والفارغُ يُخفي الزرَّ كلَّه
//	التنزيل    ملفٌّ باسمٍ مقروءٍ ونوعٍ صحيح
//	الحذف      يمسح الملفَّ ويُفرغ الإعداد

import (
	"bytes"
	"context"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

type appFixture struct {
	srv   *Server
	admin string
	dir   string
}

func newAppFixture(t *testing.T) *appFixture {
	t.Helper()
	pool := testdb.Pool(t)
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	dir := t.TempDir()
	mediaSvc, err := media.NewService(pool, dir)
	if err != nil {
		t.Fatalf("تعذّر تهيئةُ الوسائط: %v", err)
	}
	store := settings.NewStore(pool)
	// **ولا يُترك أثرُ اختبارٍ في الإعدادات** — المفتاحُ مشتركٌ مع بقيّة الحزمة.
	t.Cleanup(func() {
		_ = store.SetInternal(context.Background(), appFileSetting, "")
		_ = store.SetInternal(context.Background(), "platform.app_url", "")
	})
	return &appFixture{
		srv: &Server{
			pg:       pool,
			logger:   quiet,
			hub:      realtime.NewHub(quiet),
			settings: store,
			media:    mediaSvc,
		},
		admin: testdb.NewUser(t, pool, "admin"),
		dir:   dir,
	}
}

func (f *appFixture) upload(filename string, body []byte) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("file", filename)
	_, _ = part.Write(body)
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/admin/app-file", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	ctx := context.WithValue(req.Context(), ctxUserID, f.admin)
	ctx = context.WithValue(ctx, ctxRoles, []string{"admin"})
	w := httptest.NewRecorder()
	f.srv.handleUploadAppFile(w, req.WithContext(ctx))
	return w
}

func (f *appFixture) href() string {
	req := httptest.NewRequest(http.MethodGet, "/public/platform", nil)
	return f.srv.appHref(req)
}

// apk أصغرُ ما يبدأ ببصمة ZIP.
func apk(extra int) []byte {
	b := []byte{'P', 'K', 3, 4}
	return append(b, bytes.Repeat([]byte{0x41}, extra)...)
}

func TestAppFileUploadDownloadAndPriority(t *testing.T) {
	f := newAppFixture(t)

	t.Run("لا زرَّ حين لا رابطَ ولا ملفّ", func(t *testing.T) {
		if h := f.href(); h != "" {
			t.Fatalf("الوجهةُ %q ولا رابطَ ولا ملفّ — **وزرُّ تحميلٍ لا ينزّل شيئاً يُقرأ عطباً.**", h)
		}
	})

	t.Run("الامتدادُ apk وحدَه", func(t *testing.T) {
		for _, name := range []string{"app.zip", "app.exe", "app", "app.apk.txt"} {
			if code := f.upload(name, apk(64)).Code; code < 400 {
				t.Fatalf("قُبل %q برمز %d", name, code)
			}
		}
	})

	t.Run("والبصمةُ أرشيف", func(t *testing.T) {
		// صورةٌ بامتداد apk — تُردّ.
		png := append([]byte{0x89, 'P', 'N', 'G'}, bytes.Repeat([]byte{0}, 64)...)
		if code := f.upload("app.apk", png).Code; code < 400 {
			t.Fatalf("قُبل ملفٌّ ليس أرشيفاً برمز %d — **ومن نزّله لا يستطيع تثبيته.**", code)
		}
	})

	t.Run("الرفعُ يحفظ ويضبط الإعداد", func(t *testing.T) {
		if code := f.upload("rahalgo-1.0.apk", apk(512)).Code; code != http.StatusCreated {
			t.Fatalf("رُفض الرفعُ الصحيحُ برمز %d", code)
		}
		name := f.srv.settings.GetString(context.Background(), appFileSetting)
		if name == "" {
			t.Fatal("رُفع الملفُّ ولم يُضبط الإعداد")
		}
		if _, err := os.Stat(filepath.Join(f.dir, appDir, name)); err != nil {
			t.Fatalf("الإعدادُ يشير إلى ملفٍّ غيرِ موجود: %v", err)
		}
		// **والاسمُ المخزَّنُ مولَّدٌ لا مأخوذٌ من الرافع.**
		if name == "rahalgo-1.0.apk" {
			t.Fatal("حُفظ باسم الرافع — **واسمٌ من جهازٍ غريبٍ قد يحمل مساراً.**")
		}
		if h := f.href(); h != "/api/v1/public/app" {
			t.Fatalf("الوجهةُ %q بعد الرفع — والمنتظَر مسارَ التنزيل", h)
		}
	})

	t.Run("والتنزيلُ ملفٌّ باسمٍ مقروء", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/public/app", nil)
		w := httptest.NewRecorder()
		f.srv.handleDownloadApp(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("التنزيلُ ردّ %d", w.Code)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/vnd.android.package-archive" {
			t.Fatalf("النوعُ %q", ct)
		}
		if cd := w.Header().Get("Content-Disposition"); cd == "" || !bytes.Contains([]byte(cd), []byte("rahalgo.apk")) {
			t.Fatalf("الترويسةُ %q — **والمخزَّنُ معرّفٌ عشوائيّ، فمن نزّله يجده باسمٍ لا يقول ما هو.**", cd)
		}
		if w.Body.Len() != 516 {
			t.Fatalf("الحجمُ %d — والمنتظَر 516", w.Body.Len())
		}
	})

	// **والرابطُ يسبق**: من رفع تطبيقَه إلى المتجر فالمتجرُ أولى.
	t.Run("والرابطُ يسبق الملفَّ", func(t *testing.T) {
		const link = "https://play.google.com/store/apps/details?id=com.rahalgo"
		if err := f.srv.settings.SetInternal(context.Background(), "platform.app_url", link); err != nil {
			t.Fatalf("تعذّر ضبطُ الرابط: %v", err)
		}
		if h := f.href(); h != link {
			t.Fatalf("الوجهةُ %q والرابطُ مضبوط — **والرابطُ يسبق.**", h)
		}
	})

	t.Run("والحذفُ يمسح الملفَّ", func(t *testing.T) {
		name := f.srv.settings.GetString(context.Background(), appFileSetting)
		req := httptest.NewRequest(http.MethodDelete, "/admin/app-file", nil)
		ctx := context.WithValue(req.Context(), ctxUserID, f.admin)
		ctx = context.WithValue(ctx, ctxRoles, []string{"admin"})
		w := httptest.NewRecorder()
		f.srv.handleDeleteAppFile(w, req.WithContext(ctx))
		if w.Code != http.StatusOK {
			t.Fatalf("الحذفُ ردّ %d", w.Code)
		}
		if _, err := os.Stat(filepath.Join(f.dir, appDir, name)); err == nil {
			t.Fatal("الإعدادُ فُرِّغ والملفُّ باقٍ على القرص")
		}
		if f.srv.settings.GetString(context.Background(), appFileSetting) != "" {
			t.Fatal("الملفُّ مُسح والإعدادُ ما زال يشير إليه")
		}
	})
}
