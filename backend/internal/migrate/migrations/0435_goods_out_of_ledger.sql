-- ══════════════════════════════════════════════════════════════════════
-- **ثمنُ بضاعة المشتريات لا يدخل الحسابات** (قرارُ المالك ٢٠٢٦-١٠-١٣)
-- ══════════════════════════════════════════════════════════════════════
--
-- «المبلغ ١١٠٠ مو ربح للمنصّة… هو مبلغ مستردّ فقط، لهيك لازم ما يفوت بالحسابات». **قِيس في
-- الإنتاج**: قيدُ رجعةٍ +١١٠٠ للخزينة (#1002) بلا قيدِ الدفع المقابل (دُفع نقداً من الصندوق)،
-- **فظهرت الخزينةُ −٩٦٠ بدل −٢٠٦٠ وأُخفيت خسارةُ المكافآت.**
--
-- **فيُعكَس كلُّ `custom_goods_return` بقيدٍ سالبٍ بالمرجع نفسِه** — لا يُحذف سطر — فيصير
-- مجموعُه صفراً. والسائقُ لا يُمسّ (أجرتُه وحدَها منذ 0434).
DO $$
DECLARE
    r record;
BEGIN
    FOR r IN
        SELECT t.user_id, t.ref, sum(t.amount) AS total
          FROM wallet_transactions t
          JOIN wallets w ON w.user_id = t.user_id AND w.is_treasury
         WHERE t.kind = 'custom_goods_return'
         GROUP BY t.user_id, t.ref
        HAVING sum(t.amount) <> 0
    LOOP
        UPDATE wallets SET balance = balance - r.total, updated_at = now() WHERE user_id = r.user_id;
        INSERT INTO wallet_transactions (user_id, amount, kind, ref, note)
        VALUES (r.user_id, -r.total, 'custom_goods_return', r.ref,
                'تصحيح: ثمن البضاعة مستردٌّ لا ربح — لا يدخل الحسابات');
    END LOOP;
END $$;
