package server

// «ارسل رحال غو» (قرارُ المالك ٢٠٢٦-١١-٢١): البوتُ يبعت **نفسَ ملفّ تطبيق الزبون المرفوع**، باسمه العربيّ؛
// وبلا ملفٍّ مرفوع ما بيبعت شي.

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/release"
)

func TestWAAPP03_CustomerAppFile(t *testing.T) {
	f := newAppFixture(t)
	ctx := context.Background()
	key := release.ApkKey("customer")
	t.Cleanup(func() {
		_ = f.srv.settings.SetInternal(context.Background(), key, "")
		_ = f.srv.settings.SetInternal(context.Background(), release.VersionKey("customer"), "")
	})
	_ = f.srv.settings.SetInternal(ctx, key, "")

	if _, err := f.srv.CustomerAppFile(ctx); !errors.Is(err, errNoCustomerApp) {
		t.Fatalf("بلا ملف لازم يرجع errNoCustomerApp، رجع %v", err)
	}

	raw := []byte("PK\x03\x04 fake customer apk")
	if err := os.MkdirAll(filepath.Join(f.dir, appDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.dir, appDir, "c1.apk"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := f.srv.settings.SetInternal(ctx, key, "c1.apk"); err != nil {
		t.Fatal(err)
	}
	if err := f.srv.settings.SetInternal(ctx, release.VersionKey("customer"), "1.15.0"); err != nil {
		t.Fatal(err)
	}
	out, err := f.srv.CustomerAppFile(ctx)
	if err != nil {
		t.Fatalf("الملف موجود ورجع خطأ: %v", err)
	}
	if !bytes.Equal(out.Data, raw) || out.FileName != "رحال غو 1.15.0.apk" ||
		out.Mime != "application/vnd.android.package-archive" || out.Caption == "" {
		t.Fatalf("الملف غلط: name=%q mime=%q bytes=%d", out.FileName, out.Mime, len(out.Data))
	}
	// **والملفُّ ممسوحٌ من القرص ⇒ ما بيبعت شي**، لا ملفاً فاضياً.
	_ = os.Remove(filepath.Join(f.dir, appDir, "c1.apk"))
	if _, err := f.srv.CustomerAppFile(ctx); !errors.Is(err, errNoCustomerApp) {
		t.Fatalf("الملف ممسوح ورجع %v", err)
	}
}
