-- ══════════════════════════════════════════════════════════════════════
-- **شو بيندفع من المحفظة** — قرار المالك ٢٠٢٦-١١-٠٩
-- ══════════════════════════════════════════════════════════════════════
--
-- الزبون صار يختار: «الكل من المحفظة» أو «التوصيل من المحفظة والباقي كاش» أو كاش.
--
-- **و`payment_method` يبقى `cash | wallet` كما هو** — التطبيقات القديمة بتقرأ `== "wallet"`.
-- والعمود الجديد بيقول قدّيش من المحفظة:
--
--	none      كاش كلّه                      (payment_method = cash)
--	full      الكل من المحفظة               (payment_method = wallet)
--	delivery  التوصيل من المحفظة والباقي كاش (payment_method = wallet)
--
-- **والمشغّل بيخلّي أي كاتب قديم صحيح**: يلّي بيكتب `wallet` بلا نطاق بيصير `full`،
-- ويلّي بيكتب `cash` بيصير `none` — فالتطبيق القديم والإدخالات القديمة ما بتنكسر.
--
-- **وطلب النموذج ٢ (البضاعة برّا الحسابات، هجرة ٠٥٣٢) المحفظة فيه بتدفع التوصيل بس**
-- (قرار المالك ٢٠٢٦-١١-١٠): `wallet` بلا نطاق بيصير `delivery` مو `full`.
-- (رقم الهجرة ٠٥٣٤ لا ٠٥٢٦: الفرع انكتب قبل ٠٥٢٧–٠٥٣٣ ونُقل بعدها.)
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS wallet_scope text NOT NULL DEFAULT 'none';

UPDATE orders SET wallet_scope = 'full' WHERE payment_method = 'wallet' AND wallet_scope = 'none';
UPDATE orders SET wallet_scope = 'none' WHERE payment_method = 'cash' AND wallet_scope <> 'none';

CREATE OR REPLACE FUNCTION orders_wallet_scope_normalize() RETURNS trigger AS $$
BEGIN
    IF NEW.payment_method = 'cash' THEN
        NEW.wallet_scope := 'none';
    ELSIF NEW.payment_method = 'wallet' AND NEW.wallet_scope = 'none' THEN
        IF NEW.money_model = 2 THEN
            NEW.wallet_scope := 'delivery';
        ELSE
            NEW.wallet_scope := 'full';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS orders_wallet_scope_normalize ON orders;
CREATE TRIGGER orders_wallet_scope_normalize
    BEFORE INSERT OR UPDATE OF payment_method, wallet_scope ON orders
    FOR EACH ROW EXECUTE FUNCTION orders_wallet_scope_normalize();

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_wallet_scope_check;
ALTER TABLE orders ADD CONSTRAINT orders_wallet_scope_check
    CHECK (wallet_scope IN ('none', 'full', 'delivery')
           AND ((payment_method = 'wallet') = (wallet_scope IN ('full', 'delivery'))));
