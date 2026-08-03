package server

// **«طلبات قادمة» تُنادى في اختبار — لأنّها لم تُنادَ قطّ.**
//
// # الحادثة
//
// كان `$1` مكتوباً في استعلام الطابور **ولا يُمرَّر إليه شيء**:
//
//	s.scanDriverOrders(w, r, `… AND o.offered_driver_id = $1 …`)   // بلا وسائط
//
// فيردّ `pgx` بخطأ عددِ الوسائط، **وتردّ الواجهةُ خمسمئة في كلّ نداء.**
// وشاشةُ السائق تبتلع الخطأ (`.catch(() => undefined)`) **فتبقى القائمةُ
// فارغةً بلا كلمة.**
//
// **ولم يرَ سائقٌ طلباً قطّ** — واضطُرّ المالكُ إلى الإسناد اليدويّ في كلّ
// مرّة، وظنّ أنّ الترتيبَ لا يعمل. **والترتيبُ كان يعمل، والطابورُ لا يُقرأ.**
//
// **ولم يُمسك في بناءٍ ولا `vet` ولا اختبار**: عددُ الوسائط يُفحص وقتَ التنفيذ،
// **ولا اختبارَ كان ينادي هذا المسار.**
//
// # ولذلك يُنادى هنا
//
// **لا يفحص هذا الاختبار منطقَ الترتيب** — يفحص أنّ المسار **يردّ**. وأرخصُ
// اختبارٍ يمنع أغلى خطأ: نداءٌ واحدٌ يقول «مئتان» بدل «خمسمئة».

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDriverQueue_Responds **يردّ ٢٠٠ ومصفوفةً — لا خمسمئة.**
func TestDriverQueue_Responds(t *testing.T) {
	f := newDriverFixture(t, 1)

	ctx := context.WithValue(context.Background(), ctxUserID, f.drivers[0])
	ctx = context.WithValue(ctx, ctxRoles, []string{"driver"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/driver/queue", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	f.srv.handleDriverQueue(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("«طلبات قادمة» ردّت %d لا 200 — والشاشةُ تبتلع الخطأ فتبدو فارغة\nالردّ: %s",
			rec.Code, rec.Body.String())
	}
	// **والردُّ مغلَّف** `{"data": [...]}` كسائر الواجهات.
	var out struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("الردُّ ليس على الشكل المتوقَّع: %v", err)
	}
	if out.Data == nil {
		t.Error("لا حقلَ `data` في الردّ — والشاشةُ تقرؤه فتجده فارغاً")
	}
}
