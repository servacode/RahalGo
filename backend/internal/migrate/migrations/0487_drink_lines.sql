-- ══════════════════════════════════════════════════════════════════════
-- **المشروبُ بعددِه لا بعدد الصنف** — قرارُ المالك ٢٠٢٦-١٠-٣٠
-- ══════════════════════════════════════════════════════════════════════
--
-- كان «٢ × شاورما + عيران» يُحسب عيرانين: فرقُ الخيار يُضرب بعدد السطر.
-- **فصار المشروبُ الذي أُرسل عددُه (`option_qty`) بنداً مستقلّاً** في
-- `order_items`: اسمُه وسعرُه وعددُه — **فكلُّ جمعٍ قائمٍ (`unit_price * qty`،
-- `merchant_price * qty`) يبقى صحيحاً بلا تعديل**، وكلُّ شاشةٍ قديمةٍ تعرضه سطراً.
--
--   drink_for        **الصنفُ الذي طُلب معه** — يُمحى بمحوه (صنفٌ غير متوفّر).
--   drink_option_id  **معرّفُ خيار المشروب** — لإعادة الطلب.

ALTER TABLE order_items
    ADD COLUMN IF NOT EXISTS drink_for uuid REFERENCES order_items(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS drink_option_id uuid;

CREATE INDEX IF NOT EXISTS order_items_drink_for_idx ON order_items (drink_for)
    WHERE drink_for IS NOT NULL;
