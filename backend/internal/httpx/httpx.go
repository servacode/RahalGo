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
	ErrorWith(w, appErr, nil)
}

// ErrorWith **خطأٌ يحمل معه حقولاً في أعلى الجسم.**
//
// # ولماذا لا `JSON`
//
// **و`JSON` تغلّف ما تُعطى في `{"data":…}`** — **وهي صيغةُ النجاح.**
// **فمن كتب خطأً بها دفنه طبقةً**: العملاءُ يقرؤون `error` في الأعلى
// فلا يجدونه، **فيصير الخطأُ المُفصَّلُ خطأً «داخليّاً» مجهولاً.**
//
// **ووقع هذا في الإنتاج ٢٠٢٦-٠٩-١٢**: تحدّي التأكيد رُدّ ٤٠٣ بجسمٍ
// صحيحٍ **مغلَّفٍ في `data`** — **فلم تُفتح نافذةُ كلمةِ المرور قطُّ**،
// وقرأ المالكُ «حدث خطأ غير متوقع». **وكلُّ فعلٍ حسّاسٍ في اللوحة كان
// مكسوراً بذلك.**
//
// # والحقولُ الزائدةُ لا تزحم `error`
//
// **ومفتاحُ `error` محجوز** — فلا يُستبدَل بحقلٍ زائدٍ سهواً.
func ErrorWith(w http.ResponseWriter, appErr *AppError, extra map[string]any) {
	payload := make(map[string]any, len(extra)+1)
	for k, v := range extra {
		if k != "error" {
			payload[k] = v
		}
	}
	payload["error"] = appErr
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(appErr.Status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("httpx: encode error response", "error", err)
	}
}
