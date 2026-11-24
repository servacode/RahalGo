-- ══════════════════════════════════════════════════════════════════════
-- **شحن المحفظة بشام كاش + «شام كاش عند الاستلام»** — قرار المالك ٢٠٢٦-١١-٢٥
-- ══════════════════════════════════════════════════════════════════════
--
-- ١) الزبون بيحوّل على حساب شام كاش تبع المنصة (بالليرة الجديدة بس)، وبيكتب المبلغ ورقم العملية
--    وصورة الإيصال (اختياريّة) ⇒ طلب شحن «معلّق». المالية بتشوف التحويل بحسابها وبتكبس «تم، اشحن»
--    ⇒ قيد `topup` بمحفظة الزبون + حركة «شحن زبون» داخلة بسجلّ شام كاش، بنفس المعاملة.
--    أو «مرفوض» بسبب. **ورقم العملية ما بيتكرّر** بين المعلّق والمقبول.
--
--    `verifier` مين أكّد التحويل: `manual` اليوم (موظّف شاف حسابه)، وبكرا ربط رسميّ (createBill/webhook)
--    بيكتب اسمه هون وبيوافق لحاله بنفس الدالّة — والزبون ما بيتغيّر عليه شي.
--
-- ٢) `orders.pay_shamcash` — الزبون اختار يدفع الباقي (`cash_due`) شام كاش عند الاستلام. نيّة وبس:
--    المال متل الكاش تماماً، والكابتن بيأكّد «الزبون دفع شام كاش» بالسجلّ الجانبيّ متل قبل.

-- ── ١ · طلبات الشحن ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS shamcash_topups (
    id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           uuid        NOT NULL REFERENCES users (id),
    currency          text        NOT NULL DEFAULT 'SYP' CHECK (currency = 'SYP'),
    amount            bigint      NOT NULL CHECK (amount > 0),
    -- **رقم العملية كما كتبه الزبون** بعد التنظيف (بلا فراغات، أحرف كبيرة).
    txn_ref           text        NOT NULL CHECK (length(txn_ref) BETWEEN 3 AND 64),
    receipt_media_id  uuid        REFERENCES media (id),
    status            text        NOT NULL DEFAULT 'pending'
                                  CHECK (status IN ('pending', 'approved', 'rejected')),
    reject_reason     text        NOT NULL DEFAULT '',
    verifier          text        NOT NULL DEFAULT 'manual',
    decided_by        uuid        REFERENCES users (id),
    decided_at        timestamptz,
    -- **أثر الموافقة**: قيد المحفظة وحركة السجلّ — وكلاهما أو ولا واحد.
    wallet_tx_id      bigint      REFERENCES wallet_transactions (id),
    movement_id       uuid,
    created_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT shamcash_topup_decided CHECK ((status = 'pending') = (decided_at IS NULL)),
    CONSTRAINT shamcash_topup_approved_effects CHECK (
        (status = 'approved') = (wallet_tx_id IS NOT NULL AND movement_id IS NOT NULL)),
    CONSTRAINT shamcash_topup_reject_reason CHECK (status <> 'rejected' OR reject_reason <> '')
);

-- **رقم عملية واحد لطلب واحد حيّ** — المرفوض بيرجع ينكتب (الزبون غلط بالمبلغ مثلاً).
CREATE UNIQUE INDEX IF NOT EXISTS shamcash_topups_txn_live
    ON shamcash_topups (txn_ref) WHERE status <> 'rejected';
CREATE INDEX IF NOT EXISTS shamcash_topups_by_status ON shamcash_topups (status, created_at DESC);
CREATE INDEX IF NOT EXISTS shamcash_topups_by_user ON shamcash_topups (user_id, created_at DESC);

-- ── ٢ · حركة «شحن زبون» بسجلّ شام كاش ────────────────────────────────
ALTER TABLE shamcash_movements ADD COLUMN IF NOT EXISTS topup_id uuid REFERENCES shamcash_topups (id);
ALTER TABLE shamcash_movements DROP CONSTRAINT IF EXISTS shamcash_movements_kind_check;
ALTER TABLE shamcash_movements ADD CONSTRAINT shamcash_movements_kind_check
    CHECK (kind IN ('driver_transfer', 'deposit', 'withdrawal', 'customer_topup'));
ALTER TABLE shamcash_movements DROP CONSTRAINT IF EXISTS shamcash_topup_shape;
ALTER TABLE shamcash_movements ADD CONSTRAINT shamcash_topup_shape
    CHECK ((kind = 'customer_topup') = (topup_id IS NOT NULL));
-- **وحركة الشحن ما بتنلغى** — قيدها بالمحفظة صار؛ متل تحويل السائق.
ALTER TABLE shamcash_movements DROP CONSTRAINT IF EXISTS shamcash_void_owner_only;
ALTER TABLE shamcash_movements ADD CONSTRAINT shamcash_void_owner_only
    CHECK (voided_at IS NULL OR kind IN ('deposit', 'withdrawal'));
CREATE UNIQUE INDEX IF NOT EXISTS shamcash_one_per_topup
    ON shamcash_movements (topup_id) WHERE kind = 'customer_topup';

ALTER TABLE shamcash_topups DROP CONSTRAINT IF EXISTS shamcash_topups_movement_fk;
ALTER TABLE shamcash_topups ADD CONSTRAINT shamcash_topups_movement_fk
    FOREIGN KEY (movement_id) REFERENCES shamcash_movements (id);

-- ── ٣ · صورتان جديدتان ────────────────────────────────────────────────
--   shamcash_receipt  إيصال التحويل (محميّ برابط موقَّع)
--   shamcash_qr       باركود حساب المنصة (عامّ لكل زبون)
ALTER TABLE media DROP CONSTRAINT IF EXISTS media_kind_check;
ALTER TABLE media ADD CONSTRAINT media_kind_check CHECK (
  kind = ANY (ARRAY[
    'merchant_logo'::text,
    'menu_item'::text,
    'menu_section'::text,
    'banner'::text,
    'avatar'::text,
    'delivery_proof'::text,
    'platform_logo'::text,
    'auth_background'::text,
    'site_background'::text,
    'expense_receipt'::text,
    'odometer'::text,
    'custom_photo'::text,
    'chat_photo'::text,
    'chat_audio'::text,
    'shamcash_receipt'::text,
    'shamcash_qr'::text
  ])
);

-- ── ٤ · «شام كاش عند الاستلام» ────────────────────────────────────────
ALTER TABLE orders ADD COLUMN IF NOT EXISTS pay_shamcash boolean NOT NULL DEFAULT false;
