// Package httpx يوحّد صيغة كل استجابات الـAPI.
//
// النجاح:  {"data": ...}
// الخطأ:   {"error": {"code": "...", "message_key": "...", "details": {...}}}
//
// message_key مفتاح ترجمة مركزي (GROUND-RULES §1.1) تعرضه الواجهات
// بلغة المستخدم — الخادم لا يرسل نصوصاً جاهزة أبداً.
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type AppError struct {
	Status     int            `json:"-"`
	Code       string         `json:"code"`
	MessageKey string         `json:"message_key"`
	Details    map[string]any `json:"details,omitempty"`
}

func (e *AppError) Error() string { return e.Code }

func NewError(status int, code, messageKey string) *AppError {
	return &AppError{Status: status, Code: code, MessageKey: messageKey}
}

// أخطاء عامة جاهزة.
var (
	ErrNotFound = NewError(http.StatusNotFound, "not_found", "errors.not_found")
	ErrInternal = NewError(http.StatusInternalServerError, "internal", "errors.internal")
)

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]any{"data": data}); err != nil {
		slog.Error("httpx: encode response", "error", err)
	}
}

func Error(w http.ResponseWriter, appErr *AppError) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(appErr.Status)
	if err := json.NewEncoder(w).Encode(map[string]any{"error": appErr}); err != nil {
		slog.Error("httpx: encode error response", "error", err)
	}
}
