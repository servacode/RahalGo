-- ══════════════════════════════════════════════════════════════════════
-- **قسمُ الحسابات** — قراراتُ المالك ٢٠٢٦-١٠-٠٤ (ثلاثةٌ وعشرون قراراً)
-- ══════════════════════════════════════════════════════════════════════

-- ── ١ · حركةُ المحفظة اليدويّة طلبٌ لا قيد ─────────────────────────────
--
-- «من يقترح غيرُ من يوافق»: الموظّفُ يقترح من ملفّ الحساب، والماليّةُ توافق.
-- **ولا يُقيَّد شيءٌ قبل الموافقة**، وعند الموافقة قيدٌ بطرفين:
--   topup        نقدٌ دخل المكتب ⇒ محفظةُ الشخص + سطرٌ في صندوق المكتب
--   compensation ⇒ محفظةُ الشخص + خصمٌ من الخزينة
--   adjustment   إيداعٌ ⇒ خصمٌ من الخزينة · خصمٌ ⇒ إيداعٌ في الخزينة
CREATE TABLE IF NOT EXISTS wallet_requests (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind            text NOT NULL CHECK (kind IN ('topup', 'compensation', 'adjustment')),
    debit           boolean NOT NULL DEFAULT false,
    amount          bigint NOT NULL CHECK (amount > 0),
    note            text NOT NULL CHECK (length(btrim(note)) > 0),
    status          text NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'approved', 'rejected')),
    proposed_by     uuid NOT NULL REFERENCES users(id),
    decided_by      uuid REFERENCES users(id),
    decided_at      timestamptz,
    decision_note   text NOT NULL DEFAULT '',
    -- **موافقةُ الشخص نفسه تُعلَّم** — تقع حين لا موظّفَ ماليّاً غيرُه فقط.
    self_approved   boolean NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now(),
    CHECK (kind = 'adjustment' OR NOT debit)
);
CREATE INDEX IF NOT EXISTS wallet_requests_pending
    ON wallet_requests (created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS wallet_requests_user ON wallet_requests (user_id, created_at DESC);

-- **صندوقُ المكتب** — نقدٌ يدخل اليدَ فعلاً. (قرارُ الخزينة ٢٠٢٦-١٠-٠٤: «نقدٌ يشحن به زبونٌ
-- محفظتَه يُسجَّل داخلاً إلى الصندوق».) **ليس محفظةً ولا قيداً في الدفتر**: سجلُّ نقدٍ يقابله
-- قيدُ المحفظة بالمرجع نفسِه.
CREATE TABLE IF NOT EXISTS office_cash_entries (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    direction    text NOT NULL CHECK (direction IN ('in', 'out')),
    amount       bigint NOT NULL CHECK (amount > 0),
    source       text NOT NULL,
    ref          text NOT NULL DEFAULT '',
    user_id      uuid REFERENCES users(id) ON DELETE SET NULL,
    recorded_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    note         text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, ref)
);

-- ── ٥ · كلمةُ سرٍّ مؤقّتةٌ من النظام ───────────────────────────────────
ALTER TABLE users ADD COLUMN IF NOT EXISTS temp_password_expires_at timestamptz;
ALTER TABLE users ADD COLUMN IF NOT EXISTS welcome_sent_at timestamptz;

-- **ومن بدّلها بنفسه سقطت مهلتُها** — في كلّ مسارٍ يُنزل العلَم، بلا سطرٍ في كلّ واحد.
CREATE OR REPLACE FUNCTION users_clear_temp_expiry() RETURNS trigger AS $$
BEGIN
    IF NOT NEW.must_change_password THEN
        NEW.temp_password_expires_at := NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS users_clear_temp_expiry ON users;
CREATE TRIGGER users_clear_temp_expiry BEFORE UPDATE OF must_change_password ON users
    FOR EACH ROW EXECUTE FUNCTION users_clear_temp_expiry();

-- ── ٦ · تغييرُ الرقم فوق حدِّ الرصيد ينتظر شخصاً ثانياً ─────────────────
CREATE TABLE IF NOT EXISTS phone_change_requests (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    old_phone     text NOT NULL,
    new_phone     text NOT NULL,
    balance       bigint NOT NULL DEFAULT 0,
    status        text NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending', 'approved', 'rejected')),
    proposed_by   uuid NOT NULL REFERENCES users(id),
    decided_by    uuid REFERENCES users(id),
    decided_at    timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS phone_change_one_pending
    ON phone_change_requests (user_id) WHERE status = 'pending';

-- ── ١٢ · الملاحظاتُ الداخليّة سجلٌّ لا يُمحى ────────────────────────────
CREATE TABLE IF NOT EXISTS user_notes (
    id          bigserial PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    author_id   uuid REFERENCES users(id) ON DELETE SET NULL,
    body        text NOT NULL CHECK (length(btrim(body)) > 0),
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS user_notes_user ON user_notes (user_id, id DESC);
-- **والنصُّ القديمُ أوّلُ سطرٍ في السجلّ** — بلا كاتبٍ معروف.
INSERT INTO user_notes (user_id, body, created_at)
SELECT u.id, u.admin_notes, u.updated_at FROM users u
 WHERE btrim(u.admin_notes) <> ''
   AND NOT EXISTS (SELECT 1 FROM user_notes n WHERE n.user_id = u.id);
CREATE OR REPLACE FUNCTION user_notes_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'user_notes append-only';
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS user_notes_append_only ON user_notes;
CREATE TRIGGER user_notes_append_only BEFORE UPDATE ON user_notes
    FOR EACH ROW EXECUTE FUNCTION user_notes_append_only();

-- ── ١٣ · رفعُ منع النقد عن زبونٍ بسببٍ مكتوب ─────────────────────────────
--
-- **والمنعُ يُحسب من الفشل بعد آخر رفع** — فلا يُمحى ماضٍ ولا يُكتب في الطلبات.
ALTER TABLE users ADD COLUMN IF NOT EXISTS cash_ban_lifted_at timestamptz;
CREATE TABLE IF NOT EXISTS cash_ban_lifts (
    id          bigserial PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lifted_by   uuid REFERENCES users(id) ON DELETE SET NULL,
    reason      text NOT NULL CHECK (length(btrim(reason)) > 0),
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- ── ١٥ · معلوماتُ المركبة (اختياريّة) ────────────────────────────────
ALTER TABLE users ADD COLUMN IF NOT EXISTS vehicle_type text NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS vehicle_plate text NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS vehicle_color text NOT NULL DEFAULT '';

-- ── ١٦ · سقفُ نقدٍ خاصٌّ بسائق ───────────────────────────────────────
ALTER TABLE users ADD COLUMN IF NOT EXISTS cash_limit_override bigint
    CHECK (cash_limit_override IS NULL OR cash_limit_override >= 0);

-- ── ١٧ · قفلُ ما بعد الحادث ───────────────────────────────────────────
ALTER TABLE users ADD COLUMN IF NOT EXISTS accident_lock_at timestamptz;
ALTER TABLE users ADD COLUMN IF NOT EXISTS accident_cleared_at timestamptz;
ALTER TABLE users ADD COLUMN IF NOT EXISTS accident_cleared_by uuid REFERENCES users(id) ON DELETE SET NULL;

-- ── ١٩ · عمولةُ المندوب الموقوف محجوزة ────────────────────────────────
CREATE TABLE IF NOT EXISTS rep_held_commissions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    rep_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id     uuid NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
    amount       bigint NOT NULL CHECK (amount > 0),
    status       text NOT NULL DEFAULT 'held'
                 CHECK (status IN ('held', 'released', 'forfeited')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    decided_at   timestamptz,
    decided_by   uuid REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS rep_held_open ON rep_held_commissions (rep_id) WHERE status = 'held';

-- ── ٢٣ · صاحبُ متجرٍ لم يبقَ له متجرٌ يفقد دورَه ─────────────────────────
--
-- **في القاعدة لا في مسارٍ واحد**: نقلُ الملكيّة وحذفُ المتجر وأيُّ بابٍ يُفتح غداً.
CREATE OR REPLACE FUNCTION merchants_drop_orphan_owner_role() RETURNS trigger AS $$
DECLARE
    old_owner uuid;
BEGIN
    old_owner := OLD.owner_user_id;
    IF old_owner IS NULL THEN
        RETURN NULL;
    END IF;
    IF TG_OP = 'UPDATE' AND NEW.owner_user_id IS NOT DISTINCT FROM old_owner THEN
        RETURN NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM merchants WHERE owner_user_id = old_owner) THEN
        DELETE FROM user_roles WHERE user_id = old_owner AND role_code = 'merchant';
        INSERT INTO audit_log (actor_user_id, action, entity, entity_id, details)
        VALUES (NULL, 'system.merchant_role_dropped', 'user', old_owner::text,
                jsonb_build_object('merchant_id', OLD.id));
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS merchants_drop_orphan_owner_role ON merchants;
CREATE TRIGGER merchants_drop_orphan_owner_role
    AFTER UPDATE OF owner_user_id OR DELETE ON merchants
    FOR EACH ROW EXECUTE FUNCTION merchants_drop_orphan_owner_role();

-- ── ١٣ · رفعُ منع النقد لمديرِ المنصّة وحدَه ─────────────────────────────
INSERT INTO role_capabilities (role_code, capability_code)
SELECT r, 'users.cashban.lift' FROM unnest(ARRAY['admin', 'owner_super_admin']) AS r
 WHERE EXISTS (SELECT 1 FROM roles WHERE code = r)
ON CONFLICT DO NOTHING;
