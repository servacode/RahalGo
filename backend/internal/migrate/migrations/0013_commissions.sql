-- العمولات: نسبة المنصة لكل متجر + لقطة عمولة المنصة على الطلب

-- نسبة عمولة المنصة على المتجر (قابلة للتهيئة لكل متجر — PLAN §6.3)
ALTER TABLE merchants ADD COLUMN commission_percent int NOT NULL DEFAULT 10
    CHECK (commission_percent BETWEEN 0 AND 100);

-- لقطة عمولة المنصة المحسوبة عند التسليم (للتقارير والتسويات)
ALTER TABLE orders ADD COLUMN platform_commission bigint NOT NULL DEFAULT 0
    CHECK (platform_commission >= 0);
