-- ══════════════════════════════════════════════════════════════════════
-- **سحب طلب التعديل + علم «أكبر من المطلوب» لتحويل شام كاش** — قرار المالك ٢٠٢٦-١١-١٠
-- ══════════════════════════════════════════════════════════════════════
--
-- ١) الكابتن بيسحب طلب التعديل المعلّق بنفسه ⇒ حالة جديدة `withdrawn` (ما بتفتح معلّق ولا بتمنع التسليم).
-- ٢) السائق كتب مبلغ شام كاش أكبر من المبلغ المطلوب من الزبون ⇒ الخادم بيقبله **وبيعلّم الحركة**
--    (`exceeds_due`) واللوحة بتظلّلها. `due_at_entry` المطلوب من الزبون وقت الإدخال (ليرة)، للعرض.
ALTER TABLE order_edit_requests DROP CONSTRAINT IF EXISTS order_edit_requests_status_check;
ALTER TABLE order_edit_requests ADD CONSTRAINT order_edit_requests_status_check
    CHECK (status IN ('pending', 'approved', 'rejected', 'superseded', 'stale', 'closed', 'withdrawn'));

ALTER TABLE shamcash_movements ADD COLUMN IF NOT EXISTS exceeds_due boolean NOT NULL DEFAULT false;
ALTER TABLE shamcash_movements ADD COLUMN IF NOT EXISTS due_at_entry bigint;
