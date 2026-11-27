-- ══════════════════════════════════════════════════════════════════════
-- **بطاقة رحّال غو وكرت اليوم** (قرارُ المالك ٢٠٢٦-١١-٢٧)
-- ══════════════════════════════════════════════════════════════════════
--
-- **بطاقةُ رحّال غو**: كلُّ توصيلةٍ مسلَّمةٍ تختم دائرة (١٠ افتراضاً)، **والتي بعد امتلائها مجانيّة**.
-- **كرتُ اليوم**: ٥ توصيلاتٍ مسلَّمةٍ في يوم شغل المنصّة الواحد ⇒ **التالية في اليوم نفسِه مجانيّة**،
-- وتسقط إن لم تُستعمل قبل إغلاق المنصّة. **للزبون بطاقتُه وللمتجر بطاقتُه** («لدي توصيلة»).
--
-- # الختم
--
-- صفٌّ لكلّ طلبٍ يُسلَّم **بمفتاح الطلب** (فلا يُختم مرّتين)، يكتبه المشغّلُ أدناه في معاملة التسليم
-- نفسِها — **أيّاً كان بابُ التسليم.** **والختمُ يُحسب ما دام الطلبُ مسلَّماً**: تصحيحُ تسليمٍ أو
-- استرجاعٌ بعده يُسقطه من العدّ بلا حذف.
--
-- # الجائزة
--
-- **المالُ مسارُ عرض التوصيل المجانيّ نفسُه** (`promo_delivery_waived`، `orders/free_delivery.go`):
-- السائقُ يأخذ أجرتَه كاملةً والخزينةُ تتحمّل الفرق. `loyalty_rewards` يقول **أيُّ بطاقةٍ** أعطت
-- الإعفاءَ لأيّ طلب — **وجائزةٌ تُستهلك بتسليم طلبها، وتعود إن أُلغي أو رُفض أو فشل أو استُرجع.**
-- **وطلبُ الجائزة لا يُختم.**

CREATE TABLE IF NOT EXISTS loyalty_stamps (
    order_id   uuid PRIMARY KEY REFERENCES orders(id) ON DELETE CASCADE,
    -- `c:<customer_id>` للزبون · `m:<merchant_id>` للمتجر.
    owner      text NOT NULL,
    -- **أكانت بطاقةُ رحّال غو شغّالةً ساعةَ الختم؟** — كرتُ اليوم يعدّ كلَّ ختمٍ في يومه.
    for_stamp  boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS loyalty_stamps_owner_idx ON loyalty_stamps (owner, created_at DESC);

CREATE TABLE IF NOT EXISTS loyalty_rewards (
    order_id   uuid PRIMARY KEY REFERENCES orders(id) ON DELETE CASCADE,
    owner      text NOT NULL,
    card       text NOT NULL CHECK (card IN ('stamp', 'daily')),
    -- **حجمُ البطاقة ساعةَ الجائزة** — تغييرُ الإعداد بعدها لا يبدّل ما استُهلك.
    size       int NOT NULL CHECK (size > 0),
    -- **سقفُ المعفى ساعةَ الجائزة** (NULL بلا سقف) — ويُعاد الإعفاءُ به إن تبدّلت الأجرة.
    max_fee    bigint,
    waived     bigint NOT NULL CHECK (waived >= 0),
    -- **بدايةُ يوم الشغل** لكرت اليوم — وNULL لبطاقة رحّال غو.
    day_start  timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS loyalty_rewards_owner_idx ON loyalty_rewards (owner, card);

-- loyalty_setting_on **مفتاحٌ مضبوطٌ على `true`** — والغائبُ مطفأ (افتراضُ الكتالوج).
CREATE OR REPLACE FUNCTION loyalty_setting_on(k text) RETURNS boolean AS $$
    SELECT COALESCE((SELECT value = 'true'::jsonb FROM app_settings WHERE key = k), false)
$$ LANGUAGE sql STABLE;

-- orders_loyalty_stamp **يختم الطلبَ ساعةَ يصير مسلَّماً** — إن كانت إحدى البطاقتين شغّالة.
CREATE OR REPLACE FUNCTION orders_loyalty_stamp() RETURNS trigger AS $$
DECLARE
    v_owner     text;
    v_stamp_on  boolean;
    v_daily_on  boolean;
BEGIN
    IF NEW.kind = 'merchant_delivery' THEN
        -- **«لدي توصيلة» لمتجرها** — وطلبُ الواتساب ليس طلبَ متجر.
        IF NEW.merchant_id IS NOT NULL AND COALESCE(NEW.origin, 'app') <> 'whatsapp' THEN
            v_owner := 'm:' || NEW.merchant_id::text;
        END IF;
    ELSIF NEW.customer_id IS NOT NULL THEN
        v_owner := 'c:' || NEW.customer_id::text;
    END IF;
    IF v_owner IS NULL THEN
        RETURN NEW;
    END IF;
    -- **وطلبُ الجائزة لا يُختم.**
    IF EXISTS (SELECT 1 FROM loyalty_rewards WHERE order_id = NEW.id) THEN
        RETURN NEW;
    END IF;
    v_stamp_on := loyalty_setting_on('stamp_card.enabled');
    v_daily_on := loyalty_setting_on('daily_card.enabled');
    IF NOT (v_stamp_on OR v_daily_on) THEN
        RETURN NEW;
    END IF;
    INSERT INTO loyalty_stamps (order_id, owner, for_stamp)
    VALUES (NEW.id, v_owner, v_stamp_on)
    ON CONFLICT (order_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS orders_loyalty_stamp ON orders;
CREATE TRIGGER orders_loyalty_stamp
    AFTER UPDATE OF status ON orders
    FOR EACH ROW
    WHEN (NEW.status = 'delivered' AND OLD.status IS DISTINCT FROM 'delivered')
    EXECUTE FUNCTION orders_loyalty_stamp();
