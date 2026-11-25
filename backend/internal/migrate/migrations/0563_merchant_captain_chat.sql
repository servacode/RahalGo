-- ══════════════════════════════════════════════════════════════════════
-- **حديثُ «لدي توصيلة» بين المتجر وكابتنه** (قرارُ المالك ٢٠٢٦-١١-٢٦)
-- ══════════════════════════════════════════════════════════════════════
--
-- **التوصيلةُ لا زبونَ لها بحساب** — المستلِمُ اسمٌ ورقم (٠١٦٦). **فالطرفُ الثاني فيها
-- المتجرُ الذي طلبها**، ويُكتب دورُه `merchant`.
--
-- **والولايةُ كما هي** (٠١٧٢): رسالةُ المتجر تُنسب لكابتن التوصيلة ساعتَها — والمشغّلُ
-- يقرأ `orders.driver_id` لكلّ دورٍ غير `driver`، **فلا يحتاج تعديلاً.**
ALTER TABLE order_messages DROP CONSTRAINT IF EXISTS order_messages_sender_role_check;
ALTER TABLE order_messages ADD CONSTRAINT order_messages_sender_role_check
    CHECK (sender_role IN ('customer', 'driver', 'merchant'));
