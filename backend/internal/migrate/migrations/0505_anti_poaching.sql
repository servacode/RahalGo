-- ══════════════════════════════════════════════════════════════════════
-- **حمايةُ الزبون من سحبه برّا المنصّة** (قرارُ المالك ٢٠٢٦-١١-٠٥: «موافق على كلشي»)
-- ══════════════════════════════════════════════════════════════════════
--
-- سبعُ حماياتٍ حتّى لا يأخذ السائقُ الزبونَ برّا التطبيق. **اثنتان منها تحتاجان
-- مكاناً في القاعدة** — والباقي في الشيفرة وحدها.
--
-- (١) رسالةٌ انخفى منها رقمُ جوّالٍ أو رابطُ واتساب أو كلمتُه — تُوسَم كالشتيمة
--     (`flagged` و`flag_word` من ٠١٠٤)، **وتُعدّ لكلّ سائقٍ بصفحة المراقبة.**
--     والعمودُ بقيمةٍ افتراضيّةٍ ثابتة لا يعيد كتابةَ الجدول.
ALTER TABLE order_messages
    ADD COLUMN IF NOT EXISTS contact_leak boolean NOT NULL DEFAULT false;
CREATE INDEX IF NOT EXISTS order_messages_contact_leak_idx
    ON order_messages (driver_id, created_at DESC) WHERE contact_leak;

-- (٧) موافقةُ السائق على نسخة الشروط — دليلٌ بالنسخة وبصمةِ النصّ الذي رآه
--     والوقتِ ونسخةِ التطبيق. **وصفٌّ واحدٌ لكلّ نسخة**: الموافقةُ الثانية لا تُكرَّر.
CREATE TABLE IF NOT EXISTS terms_acceptances (
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        text NOT NULL,
    version     int  NOT NULL CHECK (version > 0),
    text_sha256 text NOT NULL,
    app_build   int,
    accepted_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role, version)
);
