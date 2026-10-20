-- ══════════════════════════════════════════════════════════════════════
-- **موظّفو المتجر — يدخلون برقمهم ويعملون باسم المتجر** (قرارُ المالك ٢٠٢٦-١٠-٢٠)
-- ══════════════════════════════════════════════════════════════════════
--
-- «الصباحي يفتح من حسابه عادي والمسائي كمان يفتح من حسابه… حسابنا مع المطعم مو مع
-- الموظف». **فالموظّفُ ليس حساباً**: رقمٌ وكلمةُ سرٍّ تفتحان جلسةً **باسم حساب صاحب
-- المتجر** — فيرى ويعمل ما يراه ويعمله، **ولا يُمسّ شيءٌ من ربط المتجر بصاحبه.**
--
--   merchant_staff          الموظّفون — رقمٌ فريدٌ وكلمةُ سرٍّ وحسابُ المتجر الذي يتبعه
--   merchant_staff_sessions أيُّ جلسةٍ لأيِّ موظّف — بها لا تُزيح جلستُه جلسةَ صاحبه
CREATE TABLE IF NOT EXISTS merchant_staff (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    phone         text NOT NULL UNIQUE,
    name          text NOT NULL DEFAULT '',
    password_hash text NOT NULL,
    created_by    uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_login_at timestamptz
);
CREATE INDEX IF NOT EXISTS merchant_staff_owner_idx ON merchant_staff (owner_user_id);

CREATE TABLE IF NOT EXISTS merchant_staff_sessions (
    session_id uuid PRIMARY KEY,
    staff_id   uuid NOT NULL REFERENCES merchant_staff (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS merchant_staff_sessions_staff_idx ON merchant_staff_sessions (staff_id);
