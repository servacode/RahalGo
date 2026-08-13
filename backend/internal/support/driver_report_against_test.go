package support

import (
	"os"
	"regexp"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **بلاغُ السائق يقول على من هو — وإلّا قُرئ عليه هو**
// ══════════════════════════════════════════════════════════════════════
//
// (شكوى المالك ٢٠٢٦-٠٨-١٣: «أنا كسائقٍ قمتُ بالإبلاغ على زبونٍ ومتجر،
//
//	ولكنّ البلاغَ تمّ فهمُه على أنّه ضدّي».)
//
// # ما وقع
//
// **`DriverReport` كانت تُدرج التذكرةَ بلا `against_user_id`.** وشاشةُ
// السمعة تعرض ما جهتُه فارغةٌ **للجميع** — عمداً، كي لا يضيع تاريخُ
// صفوفٍ سبقت ذلك العمود.
//
// **فصار بلاغُ السائق على المتجر يُعرض له هو في «الشكاوى عليك».**
//
// **ومن رفع بلاغاً فوجده شكوى عليه لا يرفع ثانياً** — وهو أسوأُ ما يقع
// بباب شكوى: **يُسكِت من فتحه ليتكلّم.**
//
// # ولماذا فحصان لا واحد
//
// **الأوّلُ يقيس القائمة**: كلُّ سببٍ يعرف جهتَه — وسببٌ يُضاف بلا جهةٍ
// يعود بالعطب نفسِه صامتاً.
//
// **والثاني يقيس الإدراج**: قائمةٌ صحيحةٌ لا تنفع إن لم يقرأها من
// يُدرج. **وهو ما وقع بالضبط**: الجهةُ مكتوبةٌ في `DriverReportReasons`
// منذ اليوم الأوّل، **ولم تكن تُقرأ.**

// TestEveryDriverReasonDeclaresItsTarget **كلُّ سببٍ يقول على من هو.**
func TestEveryDriverReasonDeclaresItsTarget(t *testing.T) {
	if len(DriverReportReasons) < 5 {
		t.Fatalf("الأسبابُ %d فقط — أحُذفت القائمة؟", len(DriverReportReasons))
	}
	for _, r := range DriverReportReasons {
		switch r.Against {
		case AgainstMerchant, AgainstCustomer:
			// **وجهةٌ صريحة** — تُنسب التذكرةُ إليها.
		case "":
			// **ولا جهةَ إلّا لـ«سببٍ آخر»** — وما عداه سكوتٌ يُقرأ
			// «على المنصّة»، **فلا يُنذَر أحدٌ ولا يُخبَر.**
			if r.Code != "other" {
				t.Errorf("السببُ %q بلا جهة — يُقرأ على المنصّة صامتاً", r.Code)
			}
		default:
			t.Errorf("السببُ %q جهتُه %q — وليست متجراً ولا زبونا", r.Code, r.Against)
		}
	}
}

// TestDriverReportWritesItsTarget **والإدراجُ يكتب الجهةَ فعلاً.**
//
// **يُقرأ الإدراجُ من الشيفرة نفسِها** — لا من ذاكرةِ من كتبه: **العمودُ
// كان غائباً عن `INSERT` سنةً كاملةً والقائمةُ صحيحة.**
func TestDriverReportWritesItsTarget(t *testing.T) {
	src, err := os.ReadFile("driver_report.go")
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	insert := regexp.MustCompile(`(?s)INSERT INTO tickets.*?RETURNING id`).Find(src)
	if insert == nil {
		t.Fatal("لم أجد إدراجَ التذكرة — تبدّل شكلُه والحارسُ صار أعمى")
	}
	if !regexp.MustCompile(`against_user_id`).Match(insert) {
		t.Error("إدراجُ بلاغ السائق بلا against_user_id — يُقرأ البلاغُ على صاحبه")
	}
	if !regexp.MustCompile(`againstDriverReport\(`).Match(src) {
		t.Error("الجهةُ لا تُشتقّ من السبب — قائمةٌ صحيحةٌ لا يقرأها أحد")
	}
}
