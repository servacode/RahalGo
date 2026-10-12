-- ══════════════════════════════════════════════════════════════════════
-- **تصحيحُ طلبات المشتريات القديمة: ثمنُ البضاعة يعود للمنصّة** (قرارُ المالك ٢٠٢٦-١٠-١٣)
-- ══════════════════════════════════════════════════════════════════════
--
-- قبل 0433 كان ثمنُ بضاعة المشتريات يُودَع محفظةَ السائق مع أجرته — على أنّه دفعه
-- من جيبه. **والمكتبُ هو الذي يعطيه ثمنَ البضاعة** (المالك: «المكتب دفع»)، فمحفظتُه
-- أجرتُه وحدَها. **قِيس في الإنتاج**: #1002 — بضاعة ١١٠٠ وأجرة ١٠٠، ودخل محفظتَه ١٢٠٠.
--
-- **فيُنقل ثمنُ البضاعة من محفظة السائق إلى الخزينة بقيدين بالمرجع نفسِه**:
-- `driver_earning` سالبٌ على السائق، و`custom_goods_return` موجبٌ للخزينة. **المجموعُ
-- لا يتغيّر** فيبقى صافي دفتر الطلب = نقدَ السائق منه (`FI-06.a`).
--
-- **ولا يُمسّ إلّا**: طلبُ مشترياتٍ سُلّم (أو استُرجع)، ببضاعةٍ موجبة، **ولم يُصحَّح
-- قبلُ** (لا `custom_goods_return` له)، **وأُودع السائقُ أكثرَ من أجرته** بمقدار البضاعة
-- على الأقلّ، **ورصيدُه يكفي** — فلا رصيدَ سالبٌ (`FI-02.b`). **وبلا خزينةٍ لا شيء.**
DO $$
DECLARE
    tid uuid;
    r record;
BEGIN
    SELECT user_id INTO tid FROM wallets WHERE is_treasury LIMIT 1;
    IF tid IS NULL THEN
        RETURN;
    END IF;
    FOR r IN
        SELECT o.id, o.number, o.driver_id, o.custom_goods_amount AS goods
          FROM orders o
          JOIN wallets w ON w.user_id = o.driver_id
         WHERE o.kind = 'custom' AND o.custom_mode = 'purchase'
           AND o.status IN ('delivered', 'refunded')
           AND COALESCE(o.custom_goods_amount, 0) > 0
           AND o.driver_id IS NOT NULL
           AND NOT EXISTS (SELECT 1 FROM wallet_transactions g
                            WHERE g.ref = o.id::text AND g.kind = 'custom_goods_return')
           AND (SELECT COALESCE(sum(t.amount), 0) FROM wallet_transactions t
                 WHERE t.ref = o.id::text AND t.kind = 'driver_earning'
                   AND t.user_id = o.driver_id) >= o.custom_goods_amount
           AND w.balance >= o.custom_goods_amount
         FOR UPDATE OF o
    LOOP
        UPDATE wallets SET balance = balance - r.goods, updated_at = now() WHERE user_id = r.driver_id;
        INSERT INTO wallet_transactions (user_id, amount, kind, ref, note)
        VALUES (r.driver_id, -r.goods, 'driver_earning', r.id::text,
                'تصحيح: ثمن بضاعة الطلب #' || r.number || ' يعود للمنصّة — المكتب دفعه');
        UPDATE wallets SET balance = balance + r.goods, updated_at = now() WHERE user_id = tid;
        INSERT INTO wallet_transactions (user_id, amount, kind, ref, note)
        VALUES (tid, r.goods, 'custom_goods_return', r.id::text,
                'تصحيح: رجعة ثمن بضاعة الطلب #' || r.number);
    END LOOP;
END $$;
