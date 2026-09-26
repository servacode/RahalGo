package server

// ══════════════════════════════════════════════════════════════════════
// **تنظيفُ كياناتِ اختبارِ المندوب من التجهيز** — Rep QA cleanup, staging-only
// ══════════════════════════════════════════════════════════════════════
//
// **حذفٌ خامٌّ لصفوفِ اختبارٍ محصورةٍ بالاسم** — كنظيرِه `merchant_second_clear`
// و`qaClearDense`: **هذه سياسةُ تنظيفِ QA المعتمدة** (حذفٌ مباشرٌ لكياناتِ
// التجهيز التي لا مسارَ إنتاجيَّ لحذفها). **وهو منفصلٌ عن الجسور** التي تمرّ
// بالمعالِجات الإنتاجيّة بلا SQL خام — لئلّا يختلط تنظيفُ العتاد بمنطق العمل.
//
// # ما يُحفَظ عمداً
//
// **لا يمسّ مستخدماً ولا محفظةً ولا `wallet_transactions` ولا `audit_log` ولا
// `incentives` ولا `payout_requests`** — التاريخُ الماليُّ والتدقيقُ يبقيان،
// وهويّاتُ QA الثابتةُ (المندوب/الأدمن) تبقى للاستعمال القادم. **يحذف فقط**
// المرشَّحاتِ والمتاجرَ والأصنافَ التي أنشأتها شهاداتُ المندوب.

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// qaRepCleanupNames **أسماءُ كياناتِ شهادةِ المندوب المسموحُ حذفُها** — قائمةٌ
// صارمةٌ لمرشَّحات/متاجرِ الاختبار وحدَها، **لا نمطَ عامّ ولا اسمٌ من الطلب.**
var qaRepCleanupNames = []string{
	"QA-Device-Store",
	"QA-Device-Store2",
	"QA-RewardDemo",
	"QA E2E Store",
}

// qaRepQACleanup **يحذف كياناتِ اختبارِ المندوب** — المرشَّحاتِ والمتاجرَ
// والأصنافَ (وتَشلَّح العروضُ وساعاتُ المتجر بالسلسلة) بأسماءٍ محصورة.
// **بمعاملةٍ واحدة**: خرقُ مفتاحٍ يُرجِعها كلَّها. — kind=rep_qa_cleanup
// (على التجهيز وحدَه، عبر بوّابة بذّار QA `handleQAStagingSeed`).
func (s *Server) qaRepQACleanup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	names := qaRepCleanupNames
	// ① الأصنافُ (يَشلَّح العرضُ بالسلسلة على menu_item) ثمّ الأقسام —
	//    كترتيب `merchant_second_clear` نفسِه.
	if _, err := tx.Exec(ctx,
		`DELETE FROM menu_items WHERE merchant_id IN (SELECT id FROM merchants WHERE name = ANY($1))`,
		names); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM menu_sections WHERE merchant_id IN (SELECT id FROM merchants WHERE name = ANY($1))`,
		names); err != nil {
		s.respondErr(w, err)
		return
	}
	// ② المرشَّحاتُ قبل المتاجر — المرشَّحُ يشير إلى متجره (`merchant_id`).
	leadTag, err := tx.Exec(ctx, `DELETE FROM merchant_leads WHERE store_name = ANY($1)`, names)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// ③ المتاجرُ (تَشلَّح ساعاتُها بالسلسلة) — بعد أن غاب من يشير إليها.
	merchTag, err := tx.Exec(ctx, `DELETE FROM merchants WHERE name = ANY($1)`, names)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA rep entities cleaned (staging-only)",
		"leads", leadTag.RowsAffected(), "merchants", merchTag.RowsAffected())
	httpx.JSON(w, http.StatusOK, map[string]any{
		"deleted_leads":     leadTag.RowsAffected(),
		"deleted_merchants": merchTag.RowsAffected(),
	})
}
