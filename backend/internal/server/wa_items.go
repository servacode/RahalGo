package server

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/waitems"
)

// NewWAItems **خدمةُ «صورة + اسم وسعر» على الواتساب** (قرارُ المالك 2026-11-09) — تُربط مع البوت عند
// الإقلاع في `cmd/api/main.go`. تستعمل خدمات الخادم نفسَها (الكتالوج والوسائط والإعدادات) وبثَّ
// اللوحات، و`send` لإرسال ملخّص الدفعة.
func (s *Server) NewWAItems(send func(ctx context.Context, phone, text string) error) *waitems.Service {
	w := waitems.New(s.pg, s.catalog, s.media, nil)
	w.Send = send
	w.SiteURL = func() string {
		if b := s.siteBase(); b != "" {
			return b
		}
		return s.siteURL()
	}
	w.Enabled = func(ctx context.Context) bool { return s.settings.GetBool(ctx, "merchants.wa_items_enabled") }
	w.NameMax = s.itemNameMax
	w.OnChange = func() { s.touch("menu", "ops") }
	return w
}
