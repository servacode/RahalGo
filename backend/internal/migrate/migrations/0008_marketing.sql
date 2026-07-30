-- التسويق: أكواد الخصم والبانرات + ربط مندوب المبيعات بالمتجر

CREATE TABLE promo_codes (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code             citext UNIQUE NOT NULL,
    -- percent: نسبة مئوية | fixed: مبلغ ثابت | free_delivery: توصيل مجاني
    kind             text NOT NULL CHECK (kind IN ('percent', 'fixed', 'free_delivery')),
    value            bigint NOT NULL DEFAULT 0 CHECK (value >= 0),  -- نسبة أو مبلغ ل.س
    min_order        bigint NOT NULL DEFAULT 0,
    first_order_only boolean NOT NULL DEFAULT false,
    once_per_user    boolean NOT NULL DEFAULT true,
    max_uses         int,                                            -- NULL = بلا حد
    used_count       int NOT NULL DEFAULT 0,
    expires_at       timestamptz,                                    -- NULL = بلا انتهاء
    active           boolean NOT NULL DEFAULT true,
    created_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE banners (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title      text NOT NULL,
    image_url  text NOT NULL DEFAULT '',
    -- وجهة النقر: merchant:<id> أو category:<id> أو url:<link> أو فارغ
    target     text NOT NULL DEFAULT '',
    sort_order int  NOT NULL DEFAULT 0,
    active     boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- مندوب المبيعات الذي جلب المتجر (لحساب عمولاته لاحقاً)
ALTER TABLE merchants ADD COLUMN sales_rep_user_id uuid REFERENCES users (id);
CREATE INDEX merchants_sales_rep_idx ON merchants (sales_rep_user_id);
