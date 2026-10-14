package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
)

// **طلبٌ قطعه الجهازُ ليس عطباً في الخادم** (٢٠٢٦-١٠-١٤): كان «context canceled»
// يُردّ ٥٠٠ فيُعدّ في «أخطاء الخادم» ويُكتب في السجلّ خطأً. **والآن ٤٩٩.**
func TestRespondErr_ClientClosedIsNotServerError(t *testing.T) {
	s := &Server{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	for _, err := range []error{
		context.Canceled,
		fmt.Errorf("identity: تعذّر التحقّق من الجلسة: %w", context.Canceled),
	} {
		w := httptest.NewRecorder()
		s.respondErr(w, err)
		if w.Code != 499 {
			t.Fatalf("%v ⇒ %d — والمنتظَر 499 لا خطأَ خادم", err, w.Code)
		}
	}
	w := httptest.NewRecorder()
	s.respondErr(w, fmt.Errorf("عطبٌ حقيقيّ"))
	if w.Code != 500 {
		t.Fatalf("العطبُ الحقيقيّ ⇒ %d — ويبقى 500", w.Code)
	}
}
