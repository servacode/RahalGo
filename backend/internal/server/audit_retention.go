package server

import (
	"context"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **حفظُ سجلّ الأحداث — الدخولُ تسعون يوماً وما سواه للأبد**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك الثاني ٢٠٢٦-١٠-٠٤.)
//
// كان ٦٨٪ من سطور السجلّ دخولاً وجلسات، والجدولُ بلا سياسة حفظ.
//
// **والحذفُ لا يُكتب هنا** — بابُه دالّةٌ واحدةٌ في القاعدة
// (`audit_prune_sessions`، الهجرة ٠٢٢٠) **ترفع رايةً يقبلها حارسُ الجدول
// لأفعال الدخول وحدَها**. فلو أخطأ هذا العاملُ يوماً في شرطه لم يحذف
// سطرَ مال: الحارسُ يرفضه.

// auditRetentionKey مفتاحُ مدّة الحفظ بالأيّام.
const auditRetentionKey = "security.audit_session_retention_days"

// RunAuditRetention يُنظّف سطورَ الدخول القديمة — مرّةً عند الإقلاع ثمّ
// كلَّ دورة.
func (s *Server) RunAuditRetention(ctx context.Context, interval time.Duration) {
	_, _ = s.PruneAuditSessions(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.PruneAuditSessions(ctx)
		}
	}
}

// PruneAuditSessions جولةٌ واحدة — **ويُنادى من الفحص مباشرةً.**
func (s *Server) PruneAuditSessions(ctx context.Context) (int64, error) {
	days := s.settings.GetInt(ctx, auditRetentionKey)
	if days < 30 {
		days = 30
	}
	var n int64
	if err := s.pg.QueryRow(ctx, `SELECT audit_prune_sessions($1)`, int(days)).Scan(&n); err != nil {
		s.logger.Error("audit: تعذّر تنظيفُ سطور الدخول", "error", err)
		return 0, err
	}
	if n > 0 {
		s.logger.Info("audit: نُظّفت سطورُ دخولٍ قديمة", "rows", n, "days", days)
	}
	return n, nil
}
