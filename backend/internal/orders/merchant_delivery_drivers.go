package orders

// ══════════════════════════════════════════════════════════════════════
// **لا توصيلةَ بلا سائقٍ بالدوام** — قرارُ المالك ٢٠٢٦-١٠-٠١
// ══════════════════════════════════════════════════════════════════════
//
// (نصُّه الأوّل: «لازم نحمي هي الخطوة إذا كانت المنصّة خارج أوقات العمل والسائقين
//  خارج أوقات العمل أيضاً».)
//
// **كانت التوصيلةُ تُرسَل والسائقون كلُّهم خارجَ الدوام** — فيبقى المتجرُ ينتظر
// سائقاً لا وجودَ له، **والغرضُ يبرد على الرفّ.**
//
// # ولا قُربَ ولا حداثة — بقرارٍ ثانٍ في اليوم نفسِه
//
// **بُني أوّلاً على «سائقٍ قريبٍ حديثِ الموقع»** (اختيارُه يومَها)، ثمّ قال:
// «مو ضروري السائقين يكونون قريبين، المهمّ يكون في سائقين بالدوام — حتّى لو
// كانوا مشغولين أو بعيدين شوي». **فالشرطُ: سائقٌ واحدٌ فاعلٌ بالدوام.** والتوزيعُ
// يتولّى البحثَ والتوسّعَ بعدها كما يتولّاه في كلّ طلب.

import (
	"context"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ErrNoDriversOnShift **لا سائقَ بالدوام الآن.**
//
// **و٥٠٣ لا ٤٠٠** — «ليس الآن» لا «طلبُك خطأ»، **كعائلة `platform_closed_now`.**
var ErrNoDriversOnShift = httpx.NewError(http.StatusServiceUnavailable,
	"no_drivers_on_shift", "errors.no_drivers_on_shift")

// requireDriverOnShift يردّ `ErrNoDriversOnShift` إن لم يكن سائقٌ واحدٌ بالدوام.
func (s *Service) requireDriverOnShift(ctx context.Context, q dbtx.Querier) error {
	var ok bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1 FROM users u
		    JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'driver'
		    WHERE u.on_shift AND u.status = 'active')`).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return ErrNoDriversOnShift
	}
	return nil
}
