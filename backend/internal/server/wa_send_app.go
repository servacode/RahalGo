package server

// **«ارسل رحال غو»** (قرارُ المالك ٢٠٢٦-١١-٢١) — مصدرُ الملفّ للبوت (notify/wa_send_app.go):
// **نفسُ ملفّ تطبيق الزبون اللي بيتنزّل من الموقع** (`release.customer.apk`)، وباسمه العربيّ نفسِه.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/notify"
	"github.com/servacode/rahalgo/backend/internal/release"
)

var errNoCustomerApp = errors.New("no customer apk uploaded")

// CustomerAppFile **ملفُّ تطبيق الزبون المرفوع** — وما في ملف ⇒ `errNoCustomerApp`.
func (s *Server) CustomerAppFile(ctx context.Context) (notify.OutgoingFile, error) {
	app, ok := release.Find("customer")
	if !ok {
		return notify.OutgoingFile{}, errNoCustomerApp
	}
	name := strings.TrimSpace(s.settings.GetString(ctx, release.ApkKey(app.Key)))
	if name == "" {
		return notify.OutgoingFile{}, errNoCustomerApp
	}
	raw, err := os.ReadFile(filepath.Join(s.media.Dir(), appDir, filepath.Base(name)))
	if err != nil || len(raw) == 0 {
		return notify.OutgoingFile{}, errNoCustomerApp
	}
	pub := release.ResolveOne(ctx, s.releaseStore(), app)
	return notify.OutgoingFile{
		Data:     raw,
		FileName: release.DisplayFileName(pub),
		Mime:     "application/vnd.android.package-archive",
		Caption:  notify.SendAppCaption,
	}, nil
}
