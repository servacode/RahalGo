package server

// ══════════════════════════════════════════════════════════════════════
// **«نزل تحديث جديد من رحّال غو»** (طلبُ المالك ٢٠٢٦-١٠-٠٨)
// ══════════════════════════════════════════════════════════════════════
//
// «لازم يصل إشعار في حال تحدّث التطبيق للمتاجر أو السائقين أو المندوب أو
// الزبون.» — **كانت شاشةُ التحديث تظهر عند الفتح وحدَه**، فمن لم يفتح التطبيقَ
// لم يعلم. **فيرنّ إشعارٌ في التطبيق نفسِه حين تُرفع نسخةٌ أحدث.**
//
// **يرنّ ولا يُحفظ** (`Transient`): صندوقُ الإشعارات للشخص لا للتطبيق، **فإعلانُ
// نسخة السائق لا يملأ صندوقَه في تطبيق الزبون.** و`Collapse` يجعل الأحدثَ يحلّ محلَّ
// ما قبله فلا يتراكم.

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// appUpdateRole **دورُ مستعملي كلّ تطبيق.**
var appUpdateRole = map[string]string{
	"customer": "customer",
	"driver":   "driver",
	"merchant": "merchant",
	"rep":      "sales",
}

func (s *Server) announceAppUpdate(ctx context.Context, appKey, version string) {
	role, ok := appUpdateRole[appKey]
	if !ok || s.notify == nil {
		return
	}
	body := "صارت نسخة جديدة جاهزة — افتح التطبيق واكبس «تحديث»."
	if version != "" {
		body = "النسخة " + version + " صارت جاهزة — افتح التطبيق واكبس «تحديث»."
	}
	in := notifications.Input{
		Kind:      notifications.KindAccount,
		Title:     "نزل تحديث جديد من رحّال غو 🎉",
		Body:      body,
		Apps:      []string{appKey},
		Transient: true,
		Collapse:  "app_update_" + appKey,
	}
	// **وبعد الردّ لا قبله** — آلافُ الزبائن لا يُبطئون رفعَ الملفّ.
	go s.notify.NotifyRoles(context.WithoutCancel(ctx), []string{role}, in)
	s.logger.Info("أُعلن تحديثُ التطبيق", "app", appKey, "version", version)
}
