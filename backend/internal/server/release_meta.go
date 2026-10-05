package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/release"
)

// ══════════════════════════════════════════════════════════════════════
// **التحديثُ تلقائيٌّ من الملفّ المرفوع** (طلبُ المالك ٢٠٢٦-١٠-٠٦)
// ══════════════════════════════════════════════════════════════════════
//
// «رفعتُ التطبيقات الجديدة وحدّدتُ الإصدارات وما طلب منّي جوّالي التحديث»
// — **و«بدون ما ظلّ أبدّل إصدارات وأرقام».**
//
// # ما كان
//
// **رقمُ الإجبار (`app.min_version.<app>`) يُرفع بيدٍ في لوحٍ**، **واسمُ
// النسخة نصٌّ يُكتب بيدٍ في لوحٍ آخر** — **والملفُّ المرفوعُ لا يقول
// المحرّكُ ما فيه.** فرُفع الملفُّ ونُسي الرقم، **أو كُتب في النسخة اسمُ
// التطبيق.**
//
// # ما صار
//
//	الرفع        يُقرأ بيانُ الحزمة: المعرّفُ والرقمُ والاسم
//	             معرّفٌ لا يطابق الخانة ⇒ يُردّ ولا يُكتب شيء
//	             وإلّا ⇒ release.<app>.version_code و.version من الملفّ
//	الإجبار      الحدُّ = أكبرُ (app.min_version.<app>) و(رقمُ الملفّ إن
//	             كان release.<app>.auto_force مشغّلاً — وهو الافتراض)
//	ملفٌّ قديم   رُفع قبل هذا ولا رقمَ له ⇒ يُقرأ مرّةً عند أوّل حاجة

var (
	errAppWrongPackage = httpx.NewError(http.StatusBadRequest,
		"app_wrong_package", "errors.app_wrong_package")
	errAppManifest = httpx.NewError(http.StatusBadRequest,
		"app_manifest_unreadable", "errors.app_manifest_unreadable")
	errSettingReadOnly = httpx.NewError(http.StatusBadRequest,
		"setting_read_only", "errors.setting_read_only")
)

// releaseAppOfKey **التطبيقُ صاحبُ مفتاح الملفّ** — `release.driver.apk` ⇒ السائق.
func releaseAppOfKey(target string) (release.App, bool) {
	for _, a := range release.Apps {
		if target == release.ApkKey(a.Key) {
			return a, true
		}
	}
	return release.App{}, false
}

// checkAPKIdentity **يقرأ بيانَ الحزمة ويطابقه بخانتها** — أو يردّ الخطأَ
// المسمّى.
func (s *Server) checkAPKIdentity(app release.App, raw []byte) (release.Manifest, *httpx.AppError) {
	m, err := release.ReadAPKManifest(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return release.Manifest{}, errAppManifest
	}
	if want := release.ExpectedPackage(app, s.qaStagingEnabled()); m.Package != want {
		e := httpx.NewError(errAppWrongPackage.Status, errAppWrongPackage.Code,
			errAppWrongPackage.MessageKey)
		e.Details = map[string]any{"package": m.Package, "expected": want}
		return m, e
	}
	if m.VersionCode <= 0 {
		return m, errAppManifest
	}
	return m, nil
}

// storeReleaseMeta **يكتب رقمَ النسخة واسمَها كما في الملفّ.**
func (s *Server) storeReleaseMeta(ctx context.Context, app release.App, m release.Manifest) error {
	if err := s.settings.SetInternal(ctx, release.VersionCodeKey(app.Key), m.VersionCode); err != nil {
		return err
	}
	return s.settings.SetInternal(ctx, release.VersionKey(app.Key), m.VersionName)
}

// clearReleaseMeta **ملفٌّ حُذف لا يبقى رقمُه حدّاً** — **وإلّا حُبس الناسُ
// على شاشة تحديثٍ إلى نسخةٍ لا تُنزَّل.**
func (s *Server) clearReleaseMeta(ctx context.Context, app release.App) {
	_ = s.settings.SetInternal(ctx, release.VersionCodeKey(app.Key), 0)
	_ = s.settings.SetInternal(ctx, release.VersionKey(app.Key), "")
}

// ══════════════════════════════════════════════════════════════════════
// **والملفُّ المرفوعُ قبل اليوم يُقرأ مرّةً** — عند أوّل حاجةٍ إليه.
// ══════════════════════════════════════════════════════════════════════
//
// **والمحاولةُ مرّةٌ لكلّ ملفٍّ في عمر العمليّة**: ملفٌّ لا يُقرأ (أو حزمتُه
// غريبة) لا يُعاد فتحُه مع كلّ نداءٍ من كلّ هاتف.
var releaseBackfillTried sync.Map

func (s *Server) backfillReleaseMeta(ctx context.Context, app release.App, name string) int64 {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || s.media == nil {
		return 0
	}
	if _, done := releaseBackfillTried.LoadOrStore(app.Key+"|"+name, true); done {
		return 0
	}
	raw, err := os.ReadFile(filepath.Join(s.media.Dir(), appDir, name))
	if err != nil {
		return 0
	}
	m, aerr := s.checkAPKIdentity(app, raw)
	if aerr != nil {
		s.logger.Warn("ملفُّ التطبيق المرفوعُ لا يُقرأ رقمُه أو حزمتُه غريبة",
			"app", app.Key, "file", name, "package", m.Package, "code", aerr.Code)
		return 0
	}
	if err := s.storeReleaseMeta(ctx, app, m); err != nil {
		return 0
	}
	s.logger.Info("قُرئ رقمُ نسخة الملفّ المرفوع", "app", app.Key,
		"version_code", m.VersionCode, "version", m.VersionName)
	return m.VersionCode
}

// backfillAllReleases **يقرأ ما لم يُقرأ من الملفّات الأربعة** — عند فتح لوح
// الإعدادات أو سجلّ التوزيع.
func (s *Server) backfillAllReleases(ctx context.Context) {
	for _, a := range release.Apps {
		if s.settings.GetInt(ctx, release.VersionCodeKey(a.Key)) > 0 {
			continue
		}
		if name := s.settings.GetString(ctx, release.ApkKey(a.Key)); name != "" {
			s.backfillReleaseMeta(ctx, a, name)
		}
	}
}

// effectiveMinVersion **أدنى نسخةٍ يُسمَح لها** — أكبرُ الحدّين:
//
//	app.min_version.<app>                      حدٌّ يدويٌّ إضافيّ
//	release.<app>.version_code (إن auto_force) رقمُ الملفّ المرفوع
//
// **ونداءٌ واحدٌ للقاعدة لا أربعة** — يقع مع كلّ نداءٍ من كلّ هاتف.
func (s *Server) effectiveMinVersion(ctx context.Context, appKey string) int64 {
	app, ok := release.Find(appKey)
	if !ok || s.settings == nil {
		return 0
	}
	keys := []string{"app.min_version." + appKey, release.AutoForceKey(appKey),
		release.VersionCodeKey(appKey), release.ApkKey(appKey)}
	vals := map[string]json.RawMessage{}
	if s.pg != nil {
		rows, err := s.pg.Query(ctx,
			`SELECT key, value FROM app_settings WHERE key = ANY($1)`, keys)
		if err == nil {
			for rows.Next() {
				var k string
				var v []byte
				if rows.Scan(&k, &v) == nil {
					vals[k] = v
				}
			}
			rows.Close()
		}
	}
	num := func(k string) int64 {
		var f float64
		if json.Unmarshal(vals[k], &f) == nil {
			return int64(f)
		}
		return 0
	}
	manual := num(keys[0])
	auto := true // الافتراضُ في الفهرس
	if raw, ok := vals[keys[1]]; ok {
		var b bool
		if json.Unmarshal(raw, &b) == nil {
			auto = b
		}
	}
	if !auto {
		return manual
	}
	// **ولا رقمَ بلا ملفّ**: رقمٌ بقي بعد حذف الملفّ يحبس الناسَ على شاشة
	// تحديثٍ إلى ما لا يُنزَّل.
	var name string
	_ = json.Unmarshal(vals[keys[3]], &name)
	if strings.TrimSpace(name) == "" {
		return manual
	}
	code := num(keys[2])
	if code <= 0 {
		code = s.backfillReleaseMeta(ctx, app, name)
	}
	if code > manual {
		return code
	}
	return manual
}
