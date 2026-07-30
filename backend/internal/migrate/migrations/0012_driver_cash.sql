-- الصندوق النقدي للسائق: نقد المنصة الذي بحوزته من الطلبات المسلَّمة
-- دفتر قيود دائم (كنمط المحفظة) + سقف نقدي ديناميكي يوقف الإسناد عند التجاوز

CREATE TABLE driver_cash_boxes (
    driver_id  uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    held       bigint NOT NULL DEFAULT 0 CHECK (held >= 0),   -- النقد بحوزته ل.س
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE driver_cash_entries (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    driver_id  uuid   NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    amount     bigint NOT NULL CHECK (amount <> 0),           -- موجب تحصيل، سالب تسليم
    kind       text   NOT NULL CHECK (kind IN
        ('order_collection',   -- تحصيل نقد طلب مسلَّم
         'settlement',         -- تسليم الصندوق للمالية
         'adjustment')),       -- تسوية إدارية
    ref        text NOT NULL DEFAULT '',                      -- معرف الطلب/التسوية
    note       text NOT NULL DEFAULT '',
    created_by uuid REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX driver_cash_entries_idx ON driver_cash_entries (driver_id, created_at DESC);

-- السقف النقدي الافتراضي (يُدار من شاشة الإعدادات)
INSERT INTO app_settings (key, value) VALUES ('drivers.cash_limit', '500000')
ON CONFLICT (key) DO NOTHING;
