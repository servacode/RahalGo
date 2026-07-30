-- قوائم المتاجر: أقسام ← أصناف ← مجموعات مُعدِّلات ← خيارات
-- الأسعار bigint بالليرة السورية (لا كسور عملياً)

CREATE TABLE menu_sections (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id uuid NOT NULL REFERENCES merchants (id) ON DELETE CASCADE,
    name        text NOT NULL,
    sort_order  int  NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX menu_sections_merchant_idx ON menu_sections (merchant_id, sort_order);

CREATE TABLE menu_items (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id uuid NOT NULL REFERENCES merchants (id) ON DELETE CASCADE,
    section_id  uuid NOT NULL REFERENCES menu_sections (id) ON DELETE CASCADE,
    name        text NOT NULL,
    description text NOT NULL DEFAULT '',
    price       bigint NOT NULL CHECK (price >= 0),      -- ل.س
    image_url   text NOT NULL DEFAULT '',
    available   boolean NOT NULL DEFAULT true,            -- إيقاف "نفد" المؤقت
    sort_order  int NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX menu_items_section_idx ON menu_items (section_id, sort_order);
CREATE INDEX menu_items_merchant_idx ON menu_items (merchant_id);

-- مجموعة مُعدِّلات لصنف: "الحجم" (اختيار واحد إلزامي)، "إضافات" (0..n)
CREATE TABLE modifier_groups (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id    uuid NOT NULL REFERENCES menu_items (id) ON DELETE CASCADE,
    name       text NOT NULL,
    min_select int  NOT NULL DEFAULT 0 CHECK (min_select >= 0),
    max_select int  NOT NULL DEFAULT 1 CHECK (max_select >= 1),
    sort_order int  NOT NULL DEFAULT 0,
    CHECK (min_select <= max_select)
);
CREATE INDEX modifier_groups_item_idx ON modifier_groups (item_id, sort_order);

CREATE TABLE modifier_options (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id    uuid NOT NULL REFERENCES modifier_groups (id) ON DELETE CASCADE,
    name        text NOT NULL,
    price_delta bigint NOT NULL DEFAULT 0,                -- فرق السعر (+/-) ل.س
    available   boolean NOT NULL DEFAULT true,
    sort_order  int NOT NULL DEFAULT 0
);
CREATE INDEX modifier_options_group_idx ON modifier_options (group_id, sort_order);
