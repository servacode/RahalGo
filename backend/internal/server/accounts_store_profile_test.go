package server

// **ملفُّ المتجر لا يُفتح** (بلاغُ المالك ٢٠٢٦-١٠-٠٤).
//
// قِيس على التجهيز بمتصفّحٍ حقيقيّ: الصفحةُ تسقط كلُّها بـ
// `Cannot read properties of null (reading 'length')`. **والسببُ كشفُ المستحقّات
// النقديّة**: متجرٌ بلا مستحقٍّ يُردّ له `"settlements": null`، **وتبويبُ النظرة
// العامّة يقرأ طولَه قبل أن يرسم شيئاً.** فالعقدُ هنا: القائمةُ الفارغةُ مصفوفة.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestMerchantCashSettlements_EmptyIsArrayNotNull(t *testing.T) {
	f := newDriverFixture(t, 0)
	req := httptest.NewRequest(http.MethodGet, "/admin/merchants/"+f.merchantID+"/cash-settlements", nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", f.merchantID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	w := httptest.NewRecorder()
	f.srv.handleMerchantCashSettlements(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("الكشفُ ردّ %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, `"settlements":null`) {
		t.Fatalf("متجرٌ بلا مستحقّات رُدّ له null — **وهو ما يُسقط ملفَّ المتجر كلَّه**: %s", body)
	}
	if !strings.Contains(body, `"settlements":[]`) {
		t.Fatalf("القائمةُ الفارغةُ ليست مصفوفة: %s", body)
	}
}
