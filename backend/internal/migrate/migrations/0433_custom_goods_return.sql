-- ══════════════════════════════════════════════════════════════════════
-- **ثمنُ بضاعة المشتريات يعود للمنصّة لا للسائق** (قرارُ المالك ٢٠٢٦-١٠-١٣)
-- ══════════════════════════════════════════════════════════════════════
--
-- «المصاري أصلاً المنصّة رح تعطيها للسائق يشتري المطلوب ويرجعهن… لهيك السائق
-- ما يشتري شي من جيبه». **فمحفظةُ السائق أجرتُه وحدَها**، وثمنُ البضاعة قيدٌ
-- للخزينة بنوعه — **رجعةُ مالٍ دفعه المكتب، لا ربح.**
DO $$
DECLARE
    def text;
BEGIN
    IF to_regproc('wallet_kinds_add') IS NOT NULL THEN
        PERFORM wallet_kinds_add('custom_goods_return');
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
    IF position('''custom_goods_return''' IN def) = 0 THEN
        def := replace(def, 'ARRAY[', 'ARRAY[''custom_goods_return''::text, ');
        EXECUTE 'ALTER TABLE wallet_transactions DROP CONSTRAINT wallet_transactions_kind_check';
        EXECUTE 'ALTER TABLE wallet_transactions ADD CONSTRAINT wallet_transactions_kind_check ' || def;
    END IF;
END $$;
