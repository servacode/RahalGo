-- ══════════════════════════════════════════════════════════════════════
--  سجلُّ الطلبات — قراراتُ المالك ٢٠٢٦-١٠-٠٤
-- ══════════════════════════════════════════════════════════════════════
--
-- # ١ · هاتفُ الزبون وموقعُ بابه وصورةُ التسليم لثلاثةٍ وحدَهم (البند ٣)
--
-- **كانت القائمةُ تردّ الطلبَ كاملاً لكلّ من يقرأ الطلبات** — الماليّةُ ومراقبُ
-- المنصّة والثقةُ والأمان. **فقدرةٌ بعينها** تُمنح للمالك والأدمن والعمليّات
-- (وإرثِها `ops`) وخدمةِ الزبائن، **والباقي يرى الهاتفَ مخفيّاً جزئيّاً** بلا
-- إحداثيّاتٍ ولا صورة.
INSERT INTO role_capabilities (role_code, capability_code)
SELECT r.code, 'orders.customer_details.read'
  FROM roles r
 WHERE r.code IN ('owner_super_admin', 'admin', 'ops', 'operations', 'customer_support')
ON CONFLICT DO NOTHING;

-- # ٢ · فهارسُ السجلّ (المشكلة ١٢)
--
-- **السجلُّ المنتهيةُ بالأحدث** — `closed_at IS NOT NULL ORDER BY created_at DESC`.
-- **ومدى التاريخ وحدَه** على `created_at`. وكانت المقارناتُ بتحويلٍ لنصّ تُعطّل
-- الفهارسَ القائمة — أُزيلت في `orders.ListWhere`.
CREATE INDEX IF NOT EXISTS orders_closed_created_idx ON orders (closed_at, created_at DESC);
CREATE INDEX IF NOT EXISTS orders_created_idx ON orders (created_at DESC);
