package incentives

// ══════════════════════════════════════════════════════════════════════
// **مفتاحُ مكافآت السائقين الواحد** — قرارُ المالك ٢٠٢٦-١١-١٨
// ══════════════════════════════════════════════════════════════════════
//
// `drivers.rewards_enabled` مطفأٌ ⇒ **لا مكافأةَ لسائقٍ من أيّ باب**:
//
//	الهدفُ الشهريّ   ⇐ مراحلُه تُقرأ بمكافأةٍ صفر (`levelsFrom`) — فلا تُصرف ولا تُحفظ
//	                   لموقوف، والجولةُ لا تسأل عن أحد، وشاشتُه لا تَعِده بمبلغ
//	الطلبُ اليدويّ   ⇐ يُردّ عند الاقتراح وعند الموافقة (`ErrDriverRewardsOff`)
//
// **والعقوباتُ لا تتأثّر**، **والمندوبُ خارجَه** — له مفاتيحُه.

import (
	"context"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

// DriverRewardsKey **مفتاحُ مكافآت السائقين.**
const DriverRewardsKey = "drivers.rewards_enabled"

// ErrDriverRewardsOff **مكافآتُ السائقين موقّفة** — طلبُ مكافأةٍ لسائقٍ والمفتاحُ مطفأ.
var ErrDriverRewardsOff = httpx.NewError(http.StatusConflict,
	"driver_rewards_off", "errors.driver_rewards_off")

// boolSettings **ما يقرأ مفتاحاً منطقيّاً** — المخزنُ الحقيقيّ. وغيرُه (اختباراتٌ بأرقامٍ وحدَها)
// يُقرأ «مشغَّلاً» كافتراض المفتاح.
type boolSettings interface {
	GetBool(ctx context.Context, key string) bool
}

// rewardsOn **أمشغَّلةٌ مكافآتُ هذا الدور؟** — السائقُ بمفتاحه، وغيرُه دائماً.
func rewardsOn(ctx context.Context, st Settings, role string) bool {
	if role != "driver" {
		return true
	}
	if b, ok := st.(boolSettings); ok && b != nil {
		return b.GetBool(ctx, DriverRewardsKey)
	}
	return true
}

// DriverRewardsOn **أمشغَّلةٌ مكافآتُ السائقين؟** — للشاشات.
func (s *Service) DriverRewardsOn(ctx context.Context) bool {
	return rewardsOn(ctx, s.settings, "driver")
}

// rewardsOnTx **بالمعاملة الممسوكة** — كما تُقرأ المراحل (`levelsOn`).
func (s *Service) rewardsOnTx(ctx context.Context, q dbtx.Querier, role string) bool {
	if st, ok := s.settings.(*settings.Store); ok && st != nil {
		return rewardsOn(ctx, st.On(q), role)
	}
	return rewardsOn(ctx, s.settings, role)
}
