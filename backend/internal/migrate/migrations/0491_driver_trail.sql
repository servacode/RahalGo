-- ══════════════════════════════════════════════════════════════════════
-- **أثر السائق اليومي — لمنع الاحتيال** (قرار المالك ٢٠٢٦-١١-٠٣)
-- ══════════════════════════════════════════════════════════════════════
--
-- «بدنا نعرف كل سائق كم كيلو مشي باليوم، بطلب وبلا طلب، وهل بيتحرك بلا طلب،
-- مثلا بيوصّل طلب من ورا ضهري. هاد حقي لأحمي حالي.»
--
-- `driver_track` بيتقلّم لساعتين وبيخدم التوزيع بس — **ما بيتغيّر**. هاد جدول
-- تاريخ تاني: نقطة بوقت التقاطها للسائق وهو على الدوام (وقتها بس التطبيق
-- بيبعت)، **مخفَّفة** (نقطة كل ٢٠ ثانية أو كل ٢٥ متر) وبتتقلّم حسب
-- `drivers.trail_retention_days` (٣٠ يوم افتراضيا). انظر `drivermonitor/`.
CREATE TABLE IF NOT EXISTS driver_trail (
    driver_id   uuid             NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    recorded_at timestamptz      NOT NULL,
    lat         double precision NOT NULL,
    lng         double precision NOT NULL,
    accuracy_m  real,
    speed_mps   real,
    mocked      boolean          NOT NULL DEFAULT false,
    PRIMARY KEY (driver_id, recorded_at)
);
-- (driver_id, recorded_at) يغطّيه المفتاح الأساسي. وهاد للتقليم الشامل.
CREATE INDEX IF NOT EXISTS driver_trail_recorded_at_idx ON driver_trail (recorded_at);

-- القدرة الجديدة: `drivers.monitor` — للمالك ومدير المنصة بس.
INSERT INTO role_capabilities (role_code, capability_code)
SELECT r.code, 'drivers.monitor'
  FROM roles r
 WHERE r.code IN ('owner_super_admin', 'admin')
ON CONFLICT DO NOTHING;
