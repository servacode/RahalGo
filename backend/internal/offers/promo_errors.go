package offers

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// رسائلُ التحقّق لأكواد الخصم — يقرؤها `catalog` (قرارُ المالك ٢٠٢٦-١٠-٠٤).
var (
	// ErrPromoCode **كودٌ فارغٌ أو فيه فراغٌ أو رموز** — حروفٌ وأرقامٌ وشَرطة.
	ErrPromoCode = httpx.NewError(http.StatusBadRequest, "promo_code_invalid", "errors.promo_code_invalid")
	// ErrPromoKind **نوعٌ غيرُ مفهوم.**
	ErrPromoKind = httpx.NewError(http.StatusBadRequest, "promo_kind_invalid", "errors.promo_kind_invalid")
	// ErrPromoPercent **نسبةٌ خارج ١–٩٠.**
	ErrPromoPercent = httpx.NewError(http.StatusBadRequest, "promo_percent_range", "errors.promo_percent_range")
	// ErrPromoCap **كودُ نسبةٍ بلا سقفٍ بالليرة.**
	ErrPromoCap = httpx.NewError(http.StatusBadRequest, "promo_cap_required", "errors.promo_cap_required")
	// ErrPromoAmount **مبلغٌ ثابتٌ صفرٌ أو سالب.**
	ErrPromoAmount = httpx.NewError(http.StatusBadRequest, "promo_amount_invalid", "errors.promo_amount_invalid")
	// ErrPromoNegative **حدٌّ أدنى أو سقفُ استخداماتٍ سالب.**
	ErrPromoNegative = httpx.NewError(http.StatusBadRequest, "promo_negative", "errors.promo_negative")
	// ErrPromoExpired **تاريخُ انتهاءٍ مضى.**
	ErrPromoExpired = httpx.NewError(http.StatusBadRequest, "promo_expiry_past", "errors.promo_expiry_past")
)
