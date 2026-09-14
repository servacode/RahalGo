package server

// ══════════════════════════════════════════════════════════════════════
// **بوّابةُ الاستقبال — أيُقبَل طلبٌ جديدٌ في هذه اللحظة** (`PH`)
// ══════════════════════════════════════════════════════════════════════
//
// # وثلاثُ طبقاتٍ مرتَّبةٌ لا واحدة
//
//	١ · `requireLaunch`   أفُتح البابُ للناس أصلاً؟   ⇒ `launch_closed`
//	٢ · إيقافٌ مؤقّت       أهو موقوفٌ الآن؟            ⇒ `temporarily_unavailable`
//	٣ · جدولُ الدوام       أنحن داخلَ الدوام؟          ⇒ `platform_closed_now`
//
// **والأولى تبقى حيث هي ولا تُستبدَل** — **وبابٌ لم يُفتح بعدُ ليس بابَ
// مغلقٍ مؤقّتاً**: **ومن خلط بينهما قال للناس «نعود الساعةَ الرابعة» عن
// بابٍ لا موعدَ لفتحه.**
//
// # وموضعُها قبل قراءةِ الجسم
//
// **وتُنادى حيث تُنادى `requireLaunch` تماماً** — **قبل قراءةِ الجسم
// وقبل مفتاح التفرّد وقبل المعاملة.** **فالمردودُ لا يُخلّف صفَّ طلبٍ
// ولا لقطةً ماليّةً ولا إشعاراً ولا التزاماً ولا يستهلك مفتاحاً.**
//
// **ولو نُودي داخلَ المعاملة لَكان الأثرُ إرجاعاً لا منعاً** — **وإرجاعٌ
// يترك وراءه ما يقع خارجَ المعاملة**: الإشعارُ والدفعُ يقعان بعد
// التثبيت، **لكنّ مفتاحَ التفرّد يُستهلَك، فيُردّ الزبونُ ولا يستطيع
// إعادةَ المحاولة بالمفتاح نفسِه.**
//
// # ولمَ ٥٠٣ لا ٤٠٩
//
// **والخدمةُ غيرُ متاحةٍ مؤقّتاً** — وهو ما يعنيه الحالان. **وهو رمزُ
// `launch_closed` نفسُه**، **فالثلاثةُ عائلةٌ واحدةٌ يفرّقها الرمزُ
// النصّيُّ لا الرقم.**

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/platform"
)

// ErrTemporarilyUnavailable **إيقافٌ تشغيليٌّ مؤقّت** — صيانةٌ أو طارئ.
var ErrTemporarilyUnavailable = httpx.NewError(http.StatusServiceUnavailable,
	"temporarily_unavailable", "errors.temporarily_unavailable")

// ErrPlatformClosedNow **خارجَ دوام المنصّة.**
var ErrPlatformClosedNow = httpx.NewError(http.StatusServiceUnavailable,
	"platform_closed_now", "errors.platform_closed_now")

// orderingError **يبني الخطأَ وفيه ما تحتاجه الشاشةُ لتقول متى نعود.**
//
// **ونصُّ المالك في `notice`** — **وهو المفتاحُ عينُه الذي يقرؤه
// التطبيقُ في `launch_closed` منذ ٢٠٢٦-٠٩-١٣**، **ولا مفتاحَ ثانٍ
// يُخترَع لمعنًى واحد.**
//
// **وموعدُ العودة بـRFC 3339** — **والشاشةُ تنسّقه بمنطقتها ولا تحكم
// به**: **الأهليّةُ قُضيت في الخادم قبل أن يُكتب هذا الردّ.**
func orderingError(st platform.State) error {
	base := ErrPlatformClosedNow
	if st.Reason == platform.ReasonTemporarilyUnavailable {
		base = ErrTemporarilyUnavailable
	}
	e := *base
	details := map[string]any{}
	if st.Message != "" {
		details["notice"] = st.Message
	}
	if st.NextAvailableAt != nil {
		details["next_available_at"] = st.NextAvailableAt.Format(time.RFC3339)
	}
	if len(details) > 0 {
		e.Details = details
	}
	return &e
}

// requireOrdering **يردّ النداءَ إن كان الاستقبالُ مغلقاً الآن.**
//
// **وعطبُ القراءة لا يفتح ولا يغلق** — **يُمضي.** **والطبقةُ الأولى
// (`requireLaunch`) تُغلق عند العطب أصلاً**، **فالفشلُ المغلقُ محفوظٌ
// فوقَ هذه.** **ولو أغلقت هذه أيضاً عند كلّ تعثّرِ قراءةٍ لَأوقف
// انقطاعُ ثانيةٍ استقبالَ المنصّة كلِّها وهي في دوامها.**
func (s *Server) requireOrdering(w http.ResponseWriter, r *http.Request) bool {
	st, err := s.platform.State(r.Context(), s.pg)
	if err != nil {
		s.logger.Error("تعذّر قراءةُ حال الاستقبال", "err", err)
		return true
	}
	if st.OrderingAvailable {
		return true
	}
	s.respondErr(w, orderingError(st))
	return false
}
