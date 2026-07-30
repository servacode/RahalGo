-- الفئات الديناميكية والمتاجر (PLAN.md: متعدد الفئات من اليوم الأول)

-- الفئات تُدار بالكامل من لوحة الأدمن — إضافة فئة جديدة لا تتطلب أي نشر
CREATE TABLE categories (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    icon       text NOT NULL DEFAULT '',            -- إيموجي أو اسم أيقونة
    sort_order int  NOT NULL DEFAULT 0,
    active     boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- الفئات الافتراضية للإطلاق (قابلة للتعديل والإضافة من اللوحة)
INSERT INTO categories (name, icon, sort_order) VALUES
    ('مطاعم',   '🍔', 1),
    ('بقالة',    '🛒', 2),
    ('صيدليات', '💊', 3),
    ('حلويات',  '🍰', 4),
    ('هدايا',    '🎁', 5);

CREATE TABLE merchants (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name          text NOT NULL,
    description   text NOT NULL DEFAULT '',
    category_id   uuid NOT NULL REFERENCES categories (id),
    phone         citext NOT NULL DEFAULT '',
    address_text  text NOT NULL DEFAULT '',          -- وصف العنوان (واقع الرقة)
    location      geography(Point, 4326),            -- دبوس الخريطة (يُضبط بشاشة المناطق)
    owner_user_id uuid REFERENCES users (id),        -- حساب صاحب المتجر (دور merchant)
    status        text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX merchants_category_idx ON merchants (category_id);
CREATE INDEX merchants_status_idx   ON merchants (status);
CREATE INDEX merchants_owner_idx    ON merchants (owner_user_id);
CREATE INDEX merchants_location_idx ON merchants USING GIST (location);
