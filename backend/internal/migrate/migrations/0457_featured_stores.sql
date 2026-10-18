-- ══════════════════════════════════════════════════════════════════════
-- **متاجرُ مميّزة — شريطٌ في السوق وصفحةُ متجر** (قرارُ المالك ٢٠٢٦-١٠-١٨)
-- ══════════════════════════════════════════════════════════════════════
--
-- الأدمنُ يختار متاجرَ تظهر مربّعاتٍ بشعارها في أعلى صفحة التسوّق،
-- **والضغطةُ تفتح صفحةَ المتجر** بأصنافه مجموعةً تحت أقسام المنصّة.
--
--   featured        يظهر في الشريط أم لا — **ولا حدَّ لعددها**
--   featured_order  ترتيبُه في الشريط — الأصغرُ أوّلاً، ثمّ الاسم
--
-- **والشريطُ يُقرأ بالفهرس الجزئيّ** — المميّزةُ قليلةٌ من مئات.
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS featured boolean NOT NULL DEFAULT false;
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS featured_order integer NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS merchants_featured_idx
    ON merchants (featured_order, name) WHERE featured;
