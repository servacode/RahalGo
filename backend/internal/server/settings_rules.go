package server

// قواعدُ حفظ الإعدادات — قراراتُ المالك ٢٠٢٦-١٠-٠٤ (الإعدادات).

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

// settingConflictErr **قيمةٌ تتعارض مع مفتاحٍ مرتبط** — ويُسمّى الطرفُ الآخر
// كي تقول اللوحةُ أيَّ بطاقةٍ تُراجَع.
func settingConflictErr(with string) *httpx.AppError {
	e := httpx.NewError(http.StatusBadRequest, "setting_conflict", "errors.setting_conflict")
	e.Details = map[string]any{"with": with}
	return e
}

// settingPlaceholderErr **قالبٌ ينقصه نصٌّ إلزاميّ** — `{code}` في قالب الرمز.
func settingPlaceholderErr(placeholder string) *httpx.AppError {
	e := httpx.NewError(http.StatusBadRequest, "setting_placeholder_missing",
		"errors.setting_placeholder_missing")
	e.Details = map[string]any{"placeholder": placeholder}
	return e
}

// currentSettingValue **القيمةُ النافذةُ الآن** — المحفوظةُ أو الافتراض.
//
// **والمفتاحُ غيرُ المحفوظ يُقارَن بافتراضه** — فمن «حفظ» الافتراضَ نفسَه لم
// يغيّر شيئاً أيضاً.
func (s *Server) currentSettingValue(ctx context.Context, key string) (any, bool) {
	raw, err := s.settings.GetRaw(ctx, key)
	if err != nil || len(raw) == 0 {
		d, ok := settings.Lookup(key)
		if !ok {
			return nil, false
		}
		raw, err = json.Marshal(d.Default)
		if err != nil {
			return nil, false
		}
	}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return nil, false
	}
	return v, true
}

// sameSettingValue **أهما القيمةُ نفسُها؟** — `10` و`10.0` سواء.
func sameSettingValue(cur, next any) bool {
	if n, ok := next.(int64); ok {
		f, ok := cur.(float64)
		return ok && f == float64(n)
	}
	return reflect.DeepEqual(cur, next)
}
