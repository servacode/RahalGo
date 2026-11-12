-- ══════════════════════════════════════════════════════════════════════
-- صور ورسائل صوتية بحديث الطلب — قرار المالك ٢٠٢٦-١١-١٢
-- ══════════════════════════════════════════════════════════════════════
--
-- الزبون والسائق يرسلان صورة (كاميرا أو معرض) أو رسالة صوتية (حتى ٦٠ ثانية).
-- الملف صفّ وسيط محميّ (رابط موقَّع) يراه طرفا الحديث والإدارة وحدهم.
-- و`body` يبقى إلزاميّاً: نصّ بديل «📷 صورة» / «🎤 رسالة صوتية» للتطبيق القديم.
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
    'custom_photo'::text,
    'chat_photo'::text,
    'chat_audio'::text
  ])
);

ALTER TABLE order_messages
  ADD COLUMN IF NOT EXISTS attachment_media_id uuid REFERENCES media (id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS attachment_kind text
    CHECK (attachment_kind IS NULL OR attachment_kind IN ('photo', 'audio')),
  ADD COLUMN IF NOT EXISTS attachment_duration_ms integer
    CHECK (attachment_duration_ms IS NULL OR attachment_duration_ms BETWEEN 0 AND 61000);

-- سقف المرفقات لكل مرسِل بالطلب يُعدّ من هنا.
CREATE INDEX IF NOT EXISTS order_messages_attachments_idx
  ON order_messages (order_id, sender_id) WHERE attachment_kind IS NOT NULL;
