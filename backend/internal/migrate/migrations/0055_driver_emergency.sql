-- ٠٠٥٥ · الطارئ — والبضاعةُ حيث وقع لا حيث بدأت
--
-- # المسألة
--
-- قرارُ المالك: «**ربما ياخذ السائق الطلب ثم يتحرك فجاءة يصبح معه حادث… زر
-- اسمه طارئ**».
--
-- **ولم يكن له بابٌ ألبتّة.** سائقٌ وقع له حادثٌ وهو حاملٌ الطعام لا يملك في
-- شاشته إلّا «تعذّر التسليم» — **فيُقفل طلبٌ كان يمكن أن يصل**، ويُحسب على
-- الزبون أو عليه ذنبٌ لم يقع، **ويبقى هو مشغولاً بشاشةٍ وهو في حالٍ لا
-- تحتمل الشاشات.**
--
-- # ولماذا الموقع
--
-- **الطعامُ ليس عند المتجر، هو عنده — حيث وقف.** فالسائقُ البديلُ يذهب إلى
-- موضع الحادث لا إلى المطعم، **وإلّا استلم طلباً ثانياً من مطبخٍ حضّر واحداً**
-- فتُدفع البضاعةُ مرّتين.
--
-- **ونقطةٌ تُكتب في الطلب لا في نصّ الإنذار**: ما يُكتب في نصٍّ يُقرأ بالعين
-- ويُنقل بالهاتف، **وما يُكتب في حقلٍ يظهر في شاشة البديل خريطةً يفتحها.**
CREATE TABLE IF NOT EXISTS driver_emergencies (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id   uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id    uuid REFERENCES orders(id) ON DELETE SET NULL,
    -- at موضعُ السائق لحظةَ الضغط — **ومنه يُستأنف الطلب.**
    at          geography(Point, 4326),
    note        text NOT NULL DEFAULT '',
    status      text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    resolved_at timestamptz,
    resolved_by uuid REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS driver_emergencies_open
    ON driver_emergencies (created_at DESC) WHERE status = 'open';

-- **نقطةُ الاستلام البديلة** — تُقرأ بدل موقع المتجر حين تُملأ.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS pickup_override geography(Point, 4326);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS pickup_override_note text NOT NULL DEFAULT '';
