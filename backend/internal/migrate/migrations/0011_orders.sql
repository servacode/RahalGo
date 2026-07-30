-- المرحلة 2 — محرك الطلبات: الطلب، أصنافه (لقطة أسعار)، سجل الانتقالات، استرداد الأكواد

CREATE TABLE orders (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    number         bigint GENERATED ALWAYS AS IDENTITY (START WITH 1001) UNIQUE, -- رقم إنساني
    customer_id    uuid NOT NULL REFERENCES users (id),
    merchant_id    uuid NOT NULL REFERENCES merchants (id),
    driver_id      uuid REFERENCES users (id),

    status         text NOT NULL DEFAULT 'pending' CHECK (status IN
        ('pending','accepted','preparing','dispatching','assigned','at_pickup',
         'picked_up','on_the_way','at_dropoff','delivered',
         'rejected','cancelled','failed','refunded')),

    -- عنوان التسليم (دبوس + وصف — واقع الرقة)
    address_text   text NOT NULL,
    dropoff        geography(Point, 4326) NOT NULL,
    zone_id        uuid REFERENCES delivery_zones (id),

    -- المالية (لقطة ثابتة بالليرة السورية)
    payment_method text NOT NULL DEFAULT 'cash' CHECK (payment_method IN ('cash','wallet','mixed')),
    subtotal       bigint NOT NULL CHECK (subtotal >= 0),
    delivery_fee   bigint NOT NULL DEFAULT 0 CHECK (delivery_fee >= 0),
    discount       bigint NOT NULL DEFAULT 0 CHECK (discount >= 0),
    total          bigint NOT NULL CHECK (total >= 0),
    wallet_paid    bigint NOT NULL DEFAULT 0 CHECK (wallet_paid >= 0),
    cash_due       bigint NOT NULL DEFAULT 0 CHECK (cash_due >= 0),
    promo_code     citext,

    notes          text NOT NULL DEFAULT '',
    cancel_reason  text NOT NULL DEFAULT '',
    created_by     uuid REFERENCES users (id),      -- من أنشأه (موظف = طلب هاتفي)

    accepted_at    timestamptz,
    picked_up_at   timestamptz,
    delivered_at   timestamptz,
    closed_at      timestamptz,                     -- أي نهاية: تسليم/إلغاء/رفض/فشل
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX orders_status_idx    ON orders (status, created_at DESC);
CREATE INDEX orders_customer_idx  ON orders (customer_id, created_at DESC);
CREATE INDEX orders_merchant_idx  ON orders (merchant_id, created_at DESC);
CREATE INDEX orders_driver_idx    ON orders (driver_id, created_at DESC);

-- أصناف الطلب — لقطة كاملة (الاسم والسعر وقت الطلب، لا مراجع حية للأسعار)
CREATE TABLE order_items (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id     uuid NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    menu_item_id uuid REFERENCES menu_items (id) ON DELETE SET NULL,
    name         text   NOT NULL,
    unit_price   bigint NOT NULL,                  -- السعر + مجموع فروق الخيارات
    qty          int    NOT NULL CHECK (qty BETWEEN 1 AND 50),
    note         text   NOT NULL DEFAULT '',
    options      jsonb  NOT NULL DEFAULT '[]'      -- [{group, name, price_delta}]
);
CREATE INDEX order_items_order_idx ON order_items (order_id);

-- سجل انتقالات آلة الحالات — من فعل ماذا ومتى
CREATE TABLE order_events (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id    uuid NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    from_status text NOT NULL,
    to_status   text NOT NULL,
    actor_id    uuid REFERENCES users (id),
    note        text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX order_events_order_idx ON order_events (order_id, id);

-- استخدامات أكواد الخصم (لقاعدة "مرة لكل مستخدم")
CREATE TABLE promo_redemptions (
    promo_id   uuid NOT NULL REFERENCES promo_codes (id) ON DELETE CASCADE,
    order_id   uuid NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (promo_id, order_id)
);
CREATE INDEX promo_redemptions_user_idx ON promo_redemptions (promo_id, user_id);
