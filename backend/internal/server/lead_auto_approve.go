package server

// ══════════════════════════════════════════════════════════════════════
// **قبولُ طلبات الانضمام تلقائياً** — قرارُ المالك ٢٠٢٦-١٠-٠٨
// ══════════════════════════════════════════════════════════════════════
//
// **مفتاحُ `leads.auto_approve` (افتراضُه لا)**: مشغَّلاً يُحوَّل كلُّ طلبٍ
// يرفعه مندوبٌ إلى متجرٍ فوراً — **بالمسار نفسِه الذي يضغطه المكتب**
// (`convertLead`): الحسابُ والكلمةُ المؤقّتةُ والمتجرُ والهدفُ في معاملةٍ
// واحدة، ثمّ رسالةُ الدخول وإشعارُ المندوب.
//
// # والفاعلُ حسابُ النظام
//
// **لا يدٌ بشريّةٌ وافقت** — فالفاعلُ حسابُ النظام (`+000000000001`، يبذره
// الترحيل). **ويُكتب سطرُ `ops.lead_auto_approved` بفاعله ذاك** وبعلامة
// `auto` — فمن سأل «مين وافق؟» قرأ «النظام» لا اسمَ موظّف.
//
// **وأفضلُ جهد**: يقع بعد تثبيت الطلب، **وسقوطُه يُبقي الطلبَ «جديداً»**
// ينتظر يدَ المكتب كما كان — **ولا يُسقط نداءَ المندوب.**

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// systemActorID **معرّفُ حساب النظام** — فارغٌ إن لم يوجد.
func (s *Server) systemActorID(ctx context.Context) string {
	var id string
	if err := s.pg.QueryRow(ctx,
		`SELECT id::text FROM users WHERE phone = $1`, systemUserPhone).Scan(&id); err != nil {
		return ""
	}
	return id
}

// autoApproveLead **يحوّل الطلبَ إن كان القبولُ التلقائيّ مشغَّلاً.**
func (s *Server) autoApproveLead(ctx context.Context, leadID, storeName, repID, ip string) {
	if !s.settings.GetBool(ctx, "leads.auto_approve") {
		return
	}
	actor := s.systemActorID(ctx)
	if actor == "" {
		s.logger.Warn("القبولُ التلقائيّ: لا حسابَ نظام — يبقى الطلبُ بيد المكتب", "lead", leadID)
		return
	}
	welcome, err := s.convertLead(ctx, actor, leadID, ip)
	if err != nil {
		s.logger.Warn("القبولُ التلقائيّ: تعذّر التحويل — يبقى الطلبُ بيد المكتب",
			"lead", leadID, "error", err)
		return
	}
	meta := map[string]any{"name": storeName, "auto": true, "sales_rep_id": repID}
	if welcome != nil {
		meta["welcome_sent"] = welcome["sent"]
	}
	s.auditCtx(ctx, actor, ip, "ops.lead_auto_approved", "lead", leadID, meta)
	// **والمكتبُ يرى ما وقع** — لا يُفاجأ بمتجرٍ لم يمرّ به.
	s.notify.NotifyOps(ctx, notifications.Input{
		Kind: notifications.KindLead, Title: m.leadAutoOps,
		Body: storeName, Entity: "lead", EntityID: leadID, Href: "/dashboard/leads",
	})
	s.touch("lead", "ops", "sales:"+repID)
}
