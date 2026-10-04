-- ══════════════════════════════════════════════════════════════════════
-- قسم «العروض والخصومات» — قرارات المالك ٢٠٢٦-١٠-٠٤
-- ══════════════════════════════════════════════════════════════════════
--
-- ١ · المحتوى يعمل كوداً أو خصماً على حساب المنصة بحرية حتى ٢٠٪ و٥٠ استخدام،
--     وفوق ذلك يحتاج موافقة المالية (جدول promo_approvals على عقد الموافقات).
-- ٢ · نسبة الكود حدها ٩٠٪، وسقف بالليرة إجباري لكود النسبة (max_discount).
-- ٣ · «مرة لكل مستخدم» و«أول طلب» بالرقم لا بالحساب (بصمة الرقم كهدية التسجيل).
-- ٦ · كلفة التوصيل المجاني سطر لحاله: orders.promo_delivery_waived.
--     وخصم الصنف يُكتب في البند: order_items.offer_cut و offer_borne_by.

-- ── أكواد الخصم ──────────────────────────────────────────────────────
ALTER TABLE promo_codes ADD COLUMN IF NOT EXISTS max_discount bigint;
ALTER TABLE promo_codes DROP CONSTRAINT IF EXISTS promo_codes_max_discount_positive;
ALTER TABLE promo_codes ADD CONSTRAINT promo_codes_max_discount_positive
    CHECK (max_discount IS NULL OR max_discount > 0);
ALTER TABLE promo_codes ADD COLUMN IF NOT EXISTS created_by uuid REFERENCES users(id);
-- approval_state: ok = يعمل بلا موافقة أو وافقت المالية · pending = بانتظار المالية
-- rejected = رفضته المالية (لا يُفعَّل).
ALTER TABLE promo_codes ADD COLUMN IF NOT EXISTS approval_state text NOT NULL DEFAULT 'ok';
ALTER TABLE promo_codes DROP CONSTRAINT IF EXISTS promo_codes_approval_state_ck;
ALTER TABLE promo_codes ADD CONSTRAINT promo_codes_approval_state_ck
    CHECK (approval_state IN ('ok', 'pending', 'rejected'));
-- نسبة الكود لا تتجاوز ٩٠ — والقديم لا يُفحص (NOT VALID)، والجديد والمعدَّل يُفحص.
ALTER TABLE promo_codes DROP CONSTRAINT IF EXISTS promo_codes_percent_max;
ALTER TABLE promo_codes ADD CONSTRAINT promo_codes_percent_max
    CHECK (kind <> 'percent' OR value <= 90) NOT VALID;

-- ── خصومات الأصناف ────────────────────────────────────────────────────
ALTER TABLE offers ADD COLUMN IF NOT EXISTS approval_state text NOT NULL DEFAULT 'ok';
ALTER TABLE offers DROP CONSTRAINT IF EXISTS offers_approval_state_ck;
ALTER TABLE offers ADD CONSTRAINT offers_approval_state_ck
    CHECK (approval_state IN ('ok', 'pending', 'rejected'));

-- ── موافقات المالية — على عقد الموافقات الموحد ─────────────────────────
CREATE TABLE IF NOT EXISTS promo_approvals (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    status        text NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending', 'approved', 'rejected')),
    -- amount أقصى كلفة متوقعة بالليرة (صفر = بلا سقف معروف، والنص يقول ذلك)
    amount        bigint NOT NULL DEFAULT 0 CHECK (amount >= 0),
    note          text NOT NULL DEFAULT '',
    target_kind   text NOT NULL CHECK (target_kind IN ('promo', 'offer')),
    target_id     uuid NOT NULL,
    proposed_by   uuid NOT NULL REFERENCES users(id),
    decided_by    uuid REFERENCES users(id),
    decision_note text NOT NULL DEFAULT '',
    self_approved boolean NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL DEFAULT now(),
    decided_at    timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS promo_approvals_one_pending
    ON promo_approvals (target_kind, target_id) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS promo_approvals_status_idx
    ON promo_approvals (status, created_at DESC);

-- ── بصمة الرقم — دالة واحدة يقرؤها كل من يربط بالرقم ───────────────────
CREATE OR REPLACE FUNCTION phone_hash_of(uid uuid) RETURNS text
LANGUAGE sql STABLE AS $$
    SELECT encode(sha256(((SELECT value FROM app_secrets WHERE key = 'phone_pepper')
                          || u.phone)::bytea), 'hex')
    FROM users u WHERE u.id = uid AND u.phone NOT LIKE 'deleted-%'
$$;

ALTER TABLE promo_redemptions ADD COLUMN IF NOT EXISTS phone_hash text;
UPDATE promo_redemptions SET phone_hash = phone_hash_of(user_id) WHERE phone_hash IS NULL;
CREATE INDEX IF NOT EXISTS promo_redemptions_phone_idx
    ON promo_redemptions (promo_id, phone_hash) WHERE phone_hash IS NOT NULL;

-- «أول طلب» بالرقم: من حذف حسابه وعنده طلبات تُحجز بصمته بنوع promo_first_order.
ALTER TABLE phone_claims DROP CONSTRAINT IF EXISTS phone_claims_kind_check;
ALTER TABLE phone_claims ADD CONSTRAINT phone_claims_kind_check
    CHECK (kind IN ('signup_bonus', 'referral', 'promo_first_order'));

-- ── كلفة التوصيل المجاني — سطر لحاله ─────────────────────────────────
ALTER TABLE orders ADD COLUMN IF NOT EXISTS promo_delivery_waived bigint NOT NULL DEFAULT 0;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_promo_delivery_waived_ck;
ALTER TABLE orders ADD CONSTRAINT orders_promo_delivery_waived_ck
    CHECK (promo_delivery_waived >= 0);
-- الطلبات القديمة بكود توصيل مجاني: ما أُعفي منه الزبون = أجر السائق − ما دفعه.
UPDATE orders o
   SET promo_delivery_waived = greatest(o.driver_fee - o.delivery_fee, 0)
  FROM promo_codes p
 WHERE p.code = o.promo_code AND p.kind = 'free_delivery'
   AND o.promo_delivery_waived = 0;

-- ── خصم الصنف في البند ────────────────────────────────────────────────
-- offer_cut مقدار الخصم للوحدة الواحدة بالليرة، و offer_borne_by من يتحمله.
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS offer_cut bigint NOT NULL DEFAULT 0;
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_offer_cut_ck;
ALTER TABLE order_items ADD CONSTRAINT order_items_offer_cut_ck CHECK (offer_cut >= 0);
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS offer_borne_by text;
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_offer_borne_ck;
ALTER TABLE order_items ADD CONSTRAINT order_items_offer_borne_ck
    CHECK (offer_borne_by IS NULL OR offer_borne_by IN ('platform', 'merchant'));
