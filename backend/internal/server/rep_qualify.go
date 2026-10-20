package server

import (
	"context"
	"time"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// ══════════════════════════════════════════════════════════════════════
// **متجرُ المندوب يُحسب حين يجهز لا حين يُسجَّل** (قرارُ المالك ٢٠٢٦-١٠-١٣)
// ══════════════════════════════════════════════════════════════════════
//
// «المندوب مو مجرد ما يسجل المتجر ينحسبله — لازم يحقق الشروط: أولاً يضبط
// ساعات الدوام، ثانياً يحدد موقعه على الخريطة، ثالثاً يضيف عدد أصناف معين
// نحدده من اللوحة.» **والأصنافُ بصورها** («اي طبعا الها صورة»).
//
// **وبشهر اكتمال الشروط يُعدّ** (`rep_qualified_at`)، **ومن اكتمل يبقى
// محسوباً** ولو نقص بعدها، **والقديمُ كما حُسب** (`0437`).
//
// **والفحصُ دوريٌّ لا في كلّ بابٍ يكتب**: الساعاتُ والموقعُ والأصنافُ تُكتب
// من تطبيق المتجر والمندوب واللوحة — **ومن نادى الفحصَ من بابٍ ونسي آخر ترك
// متجراً جاهزاً لا يُحسب.** والمنتظرُ قليلٌ دائماً فمسحُه رخيص.

// repQualifyCond **شروطُ الاحتساب الثلاثة** — `minItems` معلَمةُ أقلِّ الأصناف.
//
// **والساعاتُ يومٌ مفتوحٌ واحدٌ على الأقلّ** — جدولٌ كلُّه مغلقٌ ليس دواماً.
func repQualifyCond(minItems string) string {
	return `m.rep_qualified_at IS NULL AND m.sales_rep_user_id IS NOT NULL AND m.status = 'active'
	AND EXISTS (SELECT 1 FROM merchant_hours h WHERE h.merchant_id = m.id AND NOT h.closed)
	AND m.location IS NOT NULL
	AND (SELECT count(*) FROM menu_items i
	      WHERE i.merchant_id = m.id AND i.available AND i.approved
	        AND i.image_media_id IS NOT NULL) >= ` + minItems
}

// qualifyRepMerchantTx **يُثبت اكتمالَ متجرٍ إن اكتمل** ويصرف ما بلغه المندوبُ
// بالمعاملة نفسِها. ويردّ: أاكتمل الآن، وكم صُرف.
func (s *Server) qualifyRepMerchantTx(ctx context.Context, q dbtx.Querier,
	merchantID string) (bool, int64, error) {
	// **ومتجران للمندوب نفسِه يكتملان معاً لا يفوّتان مرحلتَه**: كلٌّ يَعُدّ
	// بمعاملته فلا يرى الآخرَ — فيقف كلاهما دون العتبة. **فالمندوبُ الواحدُ
	// يُحسب له بالدور**، وقراءةُ ما بعد القفل ترى ما ثُبّت قبله.
	if _, err := q.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('rep-goal:' || sales_rep_user_id::text))
		FROM merchants WHERE id = $1 AND sales_rep_user_id IS NOT NULL`, merchantID); err != nil {
		return false, 0, err
	}
	tag, err := q.Exec(ctx, `UPDATE merchants m SET rep_qualified_at = now()
		WHERE m.id = $1 AND `+repQualifyCond("$2"),
		merchantID, s.settings.GetInt(ctx, "sales.min_items"))
	if err != nil || tag.RowsAffected() == 0 {
		return false, 0, err
	}
	paid, err := s.grantSalesTargetTx(ctx, q, merchantID)
	return true, paid, err
}

// qualifyRepMerchant **الفحصُ بمعاملته — والإشعارُ بعد التثبيت** (`PF-01`).
func (s *Server) qualifyRepMerchant(ctx context.Context, merchantID string) bool {
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return false
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ok, paid, err := s.qualifyRepMerchantTx(ctx, tx, merchantID)
	if err != nil || !ok {
		if err != nil {
			s.logger.Warn("rep qualify failed", "merchant", merchantID, "err", err)
		}
		return false
	}
	if err := tx.Commit(ctx); err != nil {
		return false
	}
	s.notifyRepMerchantCounted(ctx, merchantID)
	s.notifySalesTargetPaid(ctx, merchantID, paid)
	return true
}

// notifyRepMerchantCounted **يُبشَّر المندوبُ بمتجرٍ دخل هدفَه.**
func (s *Server) notifyRepMerchantCounted(ctx context.Context, merchantID string) {
	rep := s.salesOpenerOf(ctx, s.pg, merchantID)
	if rep == "" {
		return
	}
	var name string
	_ = s.pg.QueryRow(ctx, `SELECT name FROM merchants WHERE id = $1`, merchantID).Scan(&name)
	s.notify.Notify(ctx, notifications.Input{
		UserID: rep, Kind: notifications.KindLead,
		Title: notifTitles.repMerchantCounted, Body: name,
		Entity: "merchant", EntityID: merchantID, Href: "/portal/merchants",
		Apps: []string{notifications.AppRep},
	})
	s.touchUser(rep, "merchants")
}

// SweepRepQualify **يحسب كلَّ متجرٍ اكتملت شروطُه** — ويردّ كم حُسب.
func (s *Server) SweepRepQualify(ctx context.Context) int {
	rows, err := s.pg.Query(ctx, `SELECT m.id::text FROM merchants m
		WHERE `+repQualifyCond("$1")+` LIMIT 200`,
		s.settings.GetInt(ctx, "sales.min_items"))
	if err != nil {
		s.logger.Warn("rep qualify sweep", "err", err)
		return 0
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	n := 0
	for _, id := range ids {
		if s.qualifyRepMerchant(ctx, id) {
			n++
		}
	}
	return n
}

// RunRepQualify **نبضةٌ دوريّة** — دقيقتان تكفيان: المندوبُ لا ينتظر ثانية.
func (s *Server) RunRepQualify(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SweepRepQualify(ctx)
		}
	}
}
