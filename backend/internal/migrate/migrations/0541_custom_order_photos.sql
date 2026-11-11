-- ══════════════════════════════════════════════════════════════════════
-- صور الطلب الخاص — قرار المالك ٢٠٢٦-١١-١١
-- ══════════════════════════════════════════════════════════════════════
--
-- الزبون يرفق صورة قائمة مشترياته أو الغرض (١–٣ صور) بدل الكتابة وحدها.
-- جدول ربط لا عمود مصفوفة: كل صورة صفّ وسيط بمفتاح أجنبي فلا تتيتّم.
-- لا مال هنا ولا قيد ولا محفظة.

ALTER TABLE media DROP CONSTRAINT IF EXISTS media_kind_check;
ALTER TABLE media ADD CONSTRAINT media_kind_check CHECK (
  kind = ANY (ARRAY[
    'merchant_logo'::text,
    'menu_item'::text,
    'menu_section'::text,
    'banner'::text,
    'avatar'::text,
    'delivery_proof'::text,
    'platform_logo'::text,
    'auth_background'::text,
    'site_background'::text,
    'expense_receipt'::text,
    'odometer'::text,
    'custom_photo'::text
  ])
);

CREATE TABLE IF NOT EXISTS order_custom_photos (
    order_id   uuid NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    media_id   uuid NOT NULL REFERENCES media (id) ON DELETE CASCADE,
    position   smallint NOT NULL CHECK (position BETWEEN 0 AND 2),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (order_id, position),
    -- الصورة تخدم طلباً واحداً فقط
    UNIQUE (media_id)
);
