package server

// **مرحلةٌ فيها زرُّ تعذّرٍ فيها أسبابٌ تُختار.**
//
// # الحادثة
//
// شهد المالكُ في شاشة السائق (٢٠٢٦-٠٨-٠٤): «لا أسباب لتعذّر التسليم تظهر
// بالقائمة» — والنافذةُ تقول «لا أسبابَ معرَّفةٌ لهذه المرحلة»، **وزرُّ
// الإرسال مُطفأٌ لأنّ الرمزَ شرط**. فالطلبُ لا يُغلق ولا يُعرف لماذا.
//
// # ولا اختبارَ كان يمرّ من هنا
//
// `FailReasonsAt` كانت بلا اختبارٍ واحد، **والنقطةُ التي تناديها بلا اختبار**.
// ولها مدخلٌ نصّيٌّ حرّ (`?at=`) **يردّ قائمةً فارغةً على كلّ ما لا يعرفه** —
// لا خطأً. فحرفٌ يتغيّر في اسم الحالة، أو حالةٌ جديدةٌ يظهر فيها الزرّ،
// **يُنتجان الشاشةَ نفسَها بالضبط**: زرٌّ مُطفأٌ وجملةٌ تقول إنّ الإعدادات
// خالية.
//
// **والفراغُ ردٌّ صالحٌ نحوياً** — فلا يسقط شيء، ولا يظهر في أيّ سجلّ.
//
// # ولماذا هنا لا في حزمة الطلبات
//
// المسارُ الذي يسلكه السائقُ يمرّ بالمعالِج لا بالدالّة: **الغلافُ والمفتاحُ
// واسمُ المعامل** كلُّها بين القائمة والشاشة. واختبارُ الدالّةِ وحدَها يمرّ
// وإن سُمّي المعاملُ `stage` بينما ترسل الشاشةُ `at`.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// failReasonsAt ينادي النقطةَ كما تناديها شاشةُ السائق — بالمفتاح نفسِه.
func (f *driverFixture) failReasonsAt(t *testing.T, at string) []map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/driver/fail-reasons?at="+at, nil)
	ctx := context.WithValue(req.Context(), ctxRoles, []string{"driver"})
	w := httptest.NewRecorder()
	f.srv.handleFailReasons(w, req.WithContext(ctx))

	if w.Code != http.StatusOK {
		t.Fatalf("at=%s ردّ %d: %s", at, w.Code, w.Body.String())
	}
	var body struct {
		Data struct {
			Reasons []map[string]any `json:"reasons"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("at=%s ردٌّ لا يُقرأ: %v — %s", at, err, w.Body.String())
	}
	return body.Data.Reasons
}

// TestFailReasons_EveryFailableStageHasReasons **لا مرحلةَ يُعرض فيها الزرُّ
// وتُردّ فارغة.**
func TestFailReasons_EveryFailableStageHasReasons(t *testing.T) {
	f := newDriverFixture(t, 1)

	// **والمرحلتان مكتوبتان في شاشة السائق أيضاً** (`FAIL_AT` في
	// `portal/page.tsx`) — فمن أضاف ثالثةً هناك ونسي `failreasons.go`
	// أظهر زرّاً بلا أسباب، وهو ما يسقط عليه هذا الاختبار.
	for _, at := range []string{orders.StAtPickup, orders.StAtDropoff} {
		list := f.failReasonsAt(t, at)
		if len(list) == 0 {
			t.Fatalf("at=%s ردّت فارغةً — **والشاشةُ تعرض «لا أسبابَ معرَّفةٌ لهذه المرحلة» "+
				"وزرُّ الإرسال مُطفأ**، فيقف السائقُ عند الباب بطلبٍ لا يُغلق.\n"+
				"والرموزُ في `orders/failreasons.go` تحت `At: %s`.", at, at)
		}
		// **وكلُّ سببٍ يحمل ذنبَه** — الذنبُ هو من يقرّر التعويض، **وفراغُه
		// يُقرأ «لا ذنب» فلا يُعوَّض سائقٌ قاد المشوارَ كاملاً.**
		for _, r := range list {
			code, _ := r["code"].(string)
			if code == "" {
				t.Fatalf("at=%s سببٌ بلا رمز: %v — **والرمزُ هو ما يُرسَل**", at, r)
			}
			if fault, _ := r["fault"].(string); fault == "" {
				t.Fatalf("at=%s السببُ %q بلا ذنب — **والذنبُ يقرّر التعويض**", at, code)
			}
		}
	}
}

// TestFailReasons_UnknownStageIsEmptyNotError **ومرحلةٌ مجهولةٌ تُردّ فارغةً
// بلا خطأ** — وهو التوقيعُ الذي يجعل الخللَ صامتاً.
//
// يُثبَّت هنا عمداً: **الخادمُ لا يستطيع تمييزَ المجهول**، فالتمييزُ واجبُ
// الشاشة — ولذلك جُلبت الأسبابُ مع الصفحة وصار «لم تصل» غيرَ «وصلت فارغة».
func TestFailReasons_UnknownStageIsEmptyNotError(t *testing.T) {
	f := newDriverFixture(t, 1)

	if list := f.failReasonsAt(t, "on_the_way"); len(list) != 0 {
		t.Fatalf("حالةٌ لا يُعلَن فيها التعذّرُ ردّت %d سبباً", len(list))
	}
	if list := f.failReasonsAt(t, ""); len(list) != 0 {
		t.Fatalf("مفتاحٌ فارغٌ ردّ %d سبباً", len(list))
	}
}
