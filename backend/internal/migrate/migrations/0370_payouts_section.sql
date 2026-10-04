-- قسمُ طلبات السحب — قراراتُ المالك ٢٠٢٦-١٠-٠٤.
--
-- ١ · السحبُ المرتجعُ نوعُ قيدٍ خاصٌّ به: `payout_reversal` («إرجاع سحب»).
--     كان يُكتب `refund` — وذاك لاسترجاع الطلبات ويشترط طلباً قائماً، فكان
--     فحصُ الدفتر FI-01.d يعدّه قيداً بلا طلب، وكشفُ الحساب يقول «استرجاع».
-- ٢ · طريقةُ الصرف تُحفظ: `cash` (نقداً من صندوق المكتب) أو `transfer`.
--     النقدُ من المكتب يُكتب سطرَ خروجٍ في صندوق المكتب.
-- ٣ · عرضٌ بعقد صفحة الموافقات الموحّدة (`payout_approvals`).

-- ── ١ · النوعُ الجديد في قيد الأنواع ───────────────────────────────────
--
-- يُضاف إلى القيد الحيّ كما هو ولا تُكتب القائمةُ كاملةً — فأقسامٌ أخرى
-- تضيف أنواعها في الوقت نفسه، وقائمةٌ كاملةٌ هنا كانت ستمحو ما أضافوه.
DO $$
DECLARE
    def text;
BEGIN
    -- دالّةُ الخزينة (هجرة 0300) هي الطريقُ المعتمد حين توجد.
    IF to_regproc('wallet_kinds_add') IS NOT NULL THEN
        PERFORM wallet_kinds_add('payout_reversal');
        RETURN;
    END IF;
    SELECT pg_get_constraintdef(c.oid) INTO def
      FROM pg_constraint c
      JOIN pg_class t ON t.oid = c.conrelid
     WHERE t.relname = 'wallet_transactions'
       AND c.conname = 'wallet_transactions_kind_check';
    IF def IS NULL THEN
        RAISE EXCEPTION 'قيد أنواع المحفظة غير موجود';
    END IF;
    IF position('''payout_reversal''' IN def) = 0 THEN
        def := replace(def, 'ARRAY[', 'ARRAY[''payout_reversal''::text, ');
        EXECUTE 'ALTER TABLE wallet_transactions DROP CONSTRAINT wallet_transactions_kind_check';
        EXECUTE 'ALTER TABLE wallet_transactions ADD CONSTRAINT wallet_transactions_kind_check ' || def;
    END IF;
END $$;

-- ما كُتب سابقاً بنوع «استرجاع» لسحبٍ مرتجع يُصحَّح اسمُه — المبلغُ لا يتغيّر.
UPDATE wallet_transactions t
   SET kind = 'payout_reversal'
  FROM payout_requests p
 WHERE t.kind = 'refund'
   AND t.ref = p.id::text
   AND p.status = 'reversed';

-- ── ٢ · طريقةُ الصرف ──────────────────────────────────────────────────
ALTER TABLE payout_requests
    ADD COLUMN IF NOT EXISTS paid_via text NOT NULL DEFAULT ''
    CHECK (paid_via IN ('', 'cash', 'transfer'));

-- ── ٣ · عقدُ صفحة الموافقات الموحّدة ─────────────────────────────────
--
-- الطلبُ ينتظر قراراً وهو `pending`، وبدأ صرفُه أو صُرف فهو موافَقٌ عليه،
-- ورُفض أو فشل أو ارتدّ فهو مردود.
CREATE OR REPLACE VIEW payout_approvals AS
SELECT p.id,
       CASE
           WHEN p.status = 'pending' THEN 'pending'
           WHEN p.status IN ('processing', 'paid') THEN 'approved'
           ELSE 'rejected'
       END AS status,
       p.amount,
       p.note,
       p.user_id AS proposed_by,
       p.decided_by,
       p.created_at,
       p.decided_at
  FROM payout_requests p;
