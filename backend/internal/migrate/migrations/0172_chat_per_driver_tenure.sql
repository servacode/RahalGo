-- ══════════════════════════════════════════════════════════════════════
-- **حديثٌ جديدٌ لكلّ سائق** (قرارُ المالك ٢٠٢٦-١٠-٠٢)
-- ══════════════════════════════════════════════════════════════════════
--
-- **كان الحديثُ للطلب وحدَه** (٠٠٨٧): السائقُ الثاني يقرأ كلَّ ما قيل للأوّل،
-- **والأوّلُ بعد أن ترك الطلبَ يقرأ ما يقوله الزبونُ للثاني** — ويَسِمه
-- مقروءاً فيرى الزبونُ «قُرئت» من رجلٍ لم يعد سائقَه.
--
-- **فكلُّ رسالةٍ تُنسب إلى ولاية سائقٍ بعينه** (`driver_id`): سائقُ الطلب
-- لحظةَ كتابتها. **ورسالةُ السائق ولايتُها هو**، ورسالةُ الزبون ولايتُها من
-- يحمل الطلبَ ساعتَها.
--
-- **والعمودُ يُملأ في القاعدة إن لم يُمرَّر** — كلُّ بابٍ يكتب في الحديث
-- (رسالةٌ وتحيّةٌ وسطرُ خطوةٍ واعتذار) **ولو نسي أحدُها لَصارت رسالتُه بلا
-- ولايةٍ فلا يراها أحد.**
ALTER TABLE order_messages
    ADD COLUMN IF NOT EXISTS driver_id uuid REFERENCES users(id) ON DELETE SET NULL;

-- **والقديمُ يُنسب بأبسط ما يصحّ**: رسالةُ السائق لكاتبها، **ورسالةُ الزبون
-- لآخر سائقٍ تكلّم قبلها** — ثمّ لحامل الطلب اليوم — ثمّ لأيّ سائقٍ تكلّم فيه.
UPDATE order_messages SET driver_id = sender_id
 WHERE sender_role = 'driver' AND driver_id IS NULL;

UPDATE order_messages m SET driver_id = COALESCE(
    (SELECT d.sender_id FROM order_messages d
      WHERE d.order_id = m.order_id AND d.sender_role = 'driver'
        AND d.created_at <= m.created_at
      ORDER BY d.created_at DESC LIMIT 1),
    (SELECT o.driver_id FROM orders o WHERE o.id = m.order_id),
    (SELECT d.sender_id FROM order_messages d
      WHERE d.order_id = m.order_id AND d.sender_role = 'driver'
      ORDER BY d.created_at LIMIT 1))
 WHERE m.sender_role = 'customer' AND m.driver_id IS NULL;

CREATE OR REPLACE FUNCTION order_messages_tenure() RETURNS trigger AS $$
BEGIN
    IF NEW.driver_id IS NULL THEN
        IF NEW.sender_role = 'driver' THEN
            NEW.driver_id := NEW.sender_id;
        ELSE
            SELECT o.driver_id INTO NEW.driver_id FROM orders o WHERE o.id = NEW.order_id;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS order_messages_tenure_trigger ON order_messages;
CREATE TRIGGER order_messages_tenure_trigger
    BEFORE INSERT ON order_messages
    FOR EACH ROW EXECUTE FUNCTION order_messages_tenure();

-- **وحديثُ الولاية يُقرأ مرتّباً** — وهو كلُّ ما يُستعلَم بعد اليوم.
CREATE INDEX IF NOT EXISTS order_messages_tenure_idx
    ON order_messages (order_id, driver_id, created_at);

-- **وسجلُّ السائق يسأل «أين كانت لي ولاية؟»** — لا «أين كتبتُ؟».
CREATE INDEX IF NOT EXISTS order_messages_driver_idx
    ON order_messages (driver_id, order_id);

-- ══════════════════════════════════════════════════════════════════════
-- **ومن حمل الطلبَ يُكتب في حدث الانتقال**
-- ══════════════════════════════════════════════════════════════════════
--
-- **والزبونُ يُخبَر «تغيّر سائقك» حين يُسند الطلبُ لثانٍ** — **ولا يُعرف
-- الثاني إلّا بمعرفة الأوّل.** و`actor_id` فاعلُ الحدث لا حاملُ الطلب: يُسند
-- المكتبُ فيكون الفاعلُ موظّفاً. **فيُكتب الحاملُ صريحاً.**
--
-- **والقديمُ يبقى فارغاً** — لا يُخمَّن من فاعلٍ قد لا يكون السائق.
ALTER TABLE order_events
    ADD COLUMN IF NOT EXISTS driver_id uuid REFERENCES users(id) ON DELETE SET NULL;
