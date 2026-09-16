package server

// ══════════════════════════════════════════════════════════════════════
//  **أنماطُ حالِ التطبيق — راحةٌ فوق الرايات، لا سلطانٌ ثانٍ**
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا نمطٌ أصلاً
//
// **وأبوابُ الزبون أربعةٌ**: التسجيلُ والتصفّحُ والطلبُ والطلبُ الخاصّ.
// **والمالكُ لا يفكّر بأربعِ رايات**، **بل بثلاثِ حالات**: ما قبل
// الافتتاح · واستعراضٌ قبله · ومفتوحٌ للعمل.
//
// **ومن قلّب أربعَ رايات بيده نسي واحدةً** — **ففُتح التسجيلُ والسوقُ
// مقفلٌ**، أو **فُتح الطلبُ ولا متجرَ فيه.**
//
// # ولا محرّكَ سياسةٍ ثانٍ
//
// **والرايةُ في `app_settings` هي الحقُّ المخزَّن** — **وهذا يكتبها ولا
// يحلّ محلَّها.** **و`launch_gate` يبقى وحدَه من يمنع**، **ومن نادى
// باباً مغلقاً رُدّ وإن لم يمرّ بنمطٍ قطّ.**
//
// # والأربعُ تُكتب معاً أو لا تُكتب
//
// **وحالُ افتتاحٍ نصفُها مكتوبٌ أسوأُ من الحالَين**: **تسجيلٌ مفتوحٌ
// وتصفّحٌ مغلقٌ يعني حساباتٍ تُنشأ لسوقٍ لا يُرى.** **فمعاملةٌ واحدة.**
//
// # ولا تُمَسّ أبوابُ العمل
//
// **والسائقُ والمتجرُ والمندوب خارجَ النمط عمداً** — **وفتحُ السوق
// للزبائن لا يعني تشغيلَ أسطول**، **وقرارُ كلٍّ منها للمالك وحدَه.**

import (
	"context"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// launchPreset **حالُ أبواب الزبون الأربعة.**
type launchPreset struct {
	Signup, Browse, Orders, CustomOrders bool
}

// launchPresets **الثلاثةُ ولا رابعَ لها.**
//
// **والأسماءُ لاتينيّةٌ في العقد وعربيّةٌ في الشاشة** — **ومعرّفٌ
// يُترجَم عقدٌ يُكسَر بأوّل تصحيحٍ لغويّ.**
var launchPresets = map[string]launchPreset{
	// **ما قبل الافتتاح** — **ولا شيءَ للزبون إلّا الدخولُ بحسابٍ قائم.**
	"pre_launch": {Signup: false, Browse: false, Orders: false, CustomOrders: false},
	// **استعراضٌ قبل الافتتاح** — **يُسجَّل ويُتصفَّح ولا يُطلَب.**
	"browse_only": {Signup: true, Browse: true, Orders: false, CustomOrders: false},
	// **مفتوحٌ للعمل.**
	"open": {Signup: true, Browse: true, Orders: true, CustomOrders: true},
}

// values **المفاتيحُ الأربعةُ وقيمُها في هذا النمط.**
func (p launchPreset) values() map[string]bool {
	return map[string]bool{
		launchCustomerSignup:       p.Signup,
		launchCustomerBrowse:       p.Browse,
		launchCustomerOrders:       p.Orders,
		launchCustomerCustomOrders: p.CustomOrders,
	}
}

// currentLaunch **ما هو مكتوبٌ الآن** — **من المخزن لا من ذاكرة.**
func (s *Server) currentLaunch(ctx context.Context) map[string]bool {
	return map[string]bool{
		launchCustomerSignup:       s.launchOpen(ctx, launchCustomerSignup),
		launchCustomerBrowse:       s.launchOpen(ctx, launchCustomerBrowse),
		launchCustomerOrders:       s.launchOpen(ctx, launchCustomerOrders),
		launchCustomerCustomOrders: s.launchOpen(ctx, launchCustomerCustomOrders),
	}
}

// workLaunch **أبوابُ العمل — تُقرأ ولا يمسّها نمط.**
func (s *Server) workLaunch(ctx context.Context) map[string]bool {
	return map[string]bool{
		launchDriverWork:     s.launchOpen(ctx, launchDriverWork),
		launchMerchantOrders: s.launchOpen(ctx, launchMerchantOrders),
		launchRepAcquisition: s.launchOpen(ctx, launchRepAcquisition),
	}
}

// matchPreset **أيُّ نمطٍ يصف الحالَ القائم؟** — **وفارغٌ يعني مخصَّصاً.**
func matchPreset(cur map[string]bool) string {
	for name, p := range launchPresets {
		ok := true
		for k, want := range p.values() {
			if cur[k] != want {
				ok = false
				break
			}
		}
		if ok {
			return name
		}
	}
	return ""
}

// handleLaunchState **يقرأ الحالَ وما يُغيّره كلُّ نمط** — **ليُعرَض
// قبل التأكيد لا بعده.**
func (s *Server) handleLaunchState(w http.ResponseWriter, r *http.Request) {
	cur := s.currentLaunch(r.Context())
	presets := make(map[string]any, len(launchPresets))
	for name, p := range launchPresets {
		enable, disable := []string{}, []string{}
		for k, want := range p.values() {
			if cur[k] == want {
				continue
			}
			if want {
				enable = append(enable, k)
			} else {
				disable = append(disable, k)
			}
		}
		presets[name] = map[string]any{
			"flags":   p.values(),
			"enable":  enable,
			"disable": disable,
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"current": cur,
		"active":  matchPreset(cur),
		"notice":  s.settings.GetString(r.Context(), launchNotice),
		"presets": presets,
		// **وأبوابُ العمل تُعرَض ليعلم المالكُ أنّها لم تتبدّل.**
		"untouched": s.workLaunch(r.Context()),
	})
}

// handleApplyLaunchPreset **يكتب الأربعَ معاً** — **أو لا يكتب شيئاً.**
func (s *Server) handleApplyLaunchPreset(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Preset string `json:"preset"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	p, ok := launchPresets[req.Preset]
	if !ok {
		s.respondErr(w, errValidation)
		return
	}

	// **والقدرةُ قدرةُ تبديل هذه المفاتيح نفسِها** — **ولا تُخترَع
	// قدرةٌ جديدةٌ لبابٍ يكتب ما يكتبه `PUT /settings/{key}`.**
	if need := settingCapability(launchCustomerOrders); !s.hasCapability(r, need) {
		s.logger.Info("التخويل: مُنع نمطُ حال التطبيق",
			"capability", string(need), "preset", req.Preset,
			"user", userIDFrom(r), "roles", rolesFrom(r))
		s.respondErr(w, errForbiddenCap)
		return
	}

	actor := userIDFrom(r)
	before := s.currentLaunch(r.Context())
	after := p.values()
	// **ودعوى «لم أمسَّ أبوابَ العمل» بلا قياسٍ دعوى** — فتُقرأ قبلُ وبعدُ.
	workBefore := s.workLaunch(r.Context())

	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		for key, want := range after {
			if err := s.settings.SetTx(ctx, q, key, want, &actor); err != nil {
				return err
			}
		}
		// **والتدقيقُ في المعاملة نفسِها** — **ونمطٌ طُبِّق وسقط سجلُّه
		// لا يُعرَف من طبّقه.**
		return s.auditTx(ctx, q, r, "admin.launch_preset", "setting", req.Preset,
			map[string]any{"before": before, "after": after, "preset": req.Preset})
	}); err != nil {
		s.respondErr(w, err)
		return
	}

	s.touch("settings", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"preset":    req.Preset,
		"before":    before,
		"after":     s.currentLaunch(r.Context()),
		"untouched": workBefore,
	})
}
