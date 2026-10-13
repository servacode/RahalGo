-- ══════════════════════════════════════════════════════════════════════
-- **حصّةُ المنصّة من الأجرة لكلّ نوعٍ وحدَه — نسبةً أو مبلغاً** (قرارُ المالك ٢٠٢٦-١٠-١٣)
-- ══════════════════════════════════════════════════════════════════════
--
-- كانت `delivery.platform_percent` نسبةً واحدةً للأنواع كلِّها. **تُنقل قيمتُها كما هي إلى
-- الأربعة نسبةً** — فلا يتبدّل أجرُ سائقٍ يومَ النشر — **ثمّ يُمحى المفتاحُ القديم.**
ALTER TABLE orders ADD COLUMN IF NOT EXISTS snap_platform_delivery_fixed bigint NOT NULL DEFAULT 0;

INSERT INTO app_settings (key, value)
SELECT k, s.value
  FROM app_settings s
 CROSS JOIN unnest(ARRAY['delivery.share_standard', 'delivery.share_merchant',
                         'delivery.share_purchase', 'delivery.share_amanah']) AS k
 WHERE s.key = 'delivery.platform_percent'
ON CONFLICT (key) DO NOTHING;

DELETE FROM app_settings WHERE key = 'delivery.platform_percent';
