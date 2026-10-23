-- «التوصيل المجاني» — قرارُ المالك ٢٠٢٦-١٠-٢٣.
--
-- عرضٌ بفترةٍ (بداية ونهاية) يُشغَّل من اللوحة، ويختار ما يشمل: الطلبُ العاديّ،
-- والمشتريات، والأمانة، و«لدي توصيلة» للمتاجر. **مرّةٌ واحدةٌ لكلّ زبونٍ بكلّ عرض**
-- (برقمه لا بحسابه)، ولكلّ متجرٍ في «لدي توصيلة». **والسائقُ يأخذ أجرتَه كاملةً**:
-- المعفى يُكتب في `orders.promo_delivery_waived`، والخزينةُ تتحمّله بالحساب الفرقيّ.

CREATE TABLE free_delivery_campaigns (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    active     boolean     NOT NULL DEFAULT false,
    starts_at  timestamptz NOT NULL,
    ends_at    timestamptz NOT NULL,
    -- standard · purchase · amanah · merchant
    kinds      text[]      NOT NULL DEFAULT '{}',
    -- أعلى ما يُعفى منه — NULL = الأجرةُ كلُّها
    max_fee    bigint CHECK (max_fee IS NULL OR max_fee > 0),
    created_by uuid REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at),
    CHECK (kinds <@ ARRAY['standard', 'purchase', 'amanah', 'merchant']::text[])
);

-- **من نال التوصيلةَ المجانيّة** — الزبونُ ببصمة رقمه، والمتجرُ بـ`m:<id>`.
-- وطلبٌ أُلغي يُحرّر حقَّه (يُحذف صفُّه عند أوّل فحصٍ بعده).
CREATE TABLE free_delivery_claims (
    campaign_id uuid   NOT NULL REFERENCES free_delivery_campaigns (id) ON DELETE CASCADE,
    claimant    text   NOT NULL,
    order_id    uuid   NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    kind        text   NOT NULL,
    waived      bigint NOT NULL CHECK (waived > 0),
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (campaign_id, claimant)
);
CREATE UNIQUE INDEX free_delivery_claims_order ON free_delivery_claims (order_id);
