package server

// موضعُ السائق — **يُرسله هو، ولا يُخمَّن.**
//
// # ولماذا نبضةٌ لا حساب
//
// كان يمكن أن يُستنتج من آخر طلبٍ سلّمه. **ومن سلّم في الرميلة ثمّ عاد إلى
// وسط المدينة يُقرأ في الرميلة** — فيُحرَم ممّا هو تحت يده ويُعطى ما هو بعيد.
//
// # وحين يعمل وحدَه
//
// **موضعُ من ليس على الدوام لا يُسأل عنه أحد** — والتطبيقُ لا يُرسله. فإن وصل
// فلا ضير: العمودُ يُقرأ مع وقته، **ومن لم يُحدّث موضعَه منذ مدّةٍ لا يُقاس
// عليه.**

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// StaleLocationAfter بعدها تُعامَل النقطةُ كأنّها غيرُ موجودة.
//
// **ونقطةٌ بلا وقتٍ كذبةٌ تشيخ**: سائقٌ أغلق التطبيقَ قبل ساعةٍ يبقى موضعُه
// مكتوباً، **فيُحسب أقربَ الجميع وهو في بيته.** وربعُ ساعةٍ يكفي: من يعمل
// يُرسل كلَّ دقيقة، ومن انقطع ربعَ ساعةٍ لم يعد يُقاس عليه.
const StaleLocationAfter = 15 * time.Minute

// staleLocationMinutes الرقمُ نفسُه بالدقائق — **لاستعلامٍ لا يفهم `Duration`.**
//
// **ويُشتقّ لا يُكتب**: رقمان لمعنًى واحدٍ يفترقان يوماً.
var staleLocationMinutes = int(StaleLocationAfter / time.Minute)

// handleDriverLocation يسجّل آخرَ موضعٍ للسائق.
func (s *Server) handleDriverLocation(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **وصفرٌ صفرٌ ليس موضعاً** — هو ما يرسله جهازٌ لم يجد إشارة، **ونقطةٌ في
	// المحيط الأطلسيّ تجعل كلَّ مسافةٍ آلافَ الكيلومترات** فيسقط الترتيبُ
	// كلُّه بهدوء.
	if req.Lat == 0 && req.Lng == 0 {
		s.respondErr(w, errValidation)
		return
	}
	if req.Lat < -90 || req.Lat > 90 || req.Lng < -180 || req.Lng > 180 {
		s.respondErr(w, errValidation)
		return
	}
	if _, err := s.pg.Exec(r.Context(), `
		UPDATE users
		SET last_location = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography,
		    last_location_at = now()
		WHERE id = $1`, userIDFrom(r), req.Lng, req.Lat); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"saved": true})
}
