-- ══════════════════════════════════════════════════════════════════════
-- **الأهدافُ والمكافآت — قراراتُ المالك 2026-10-04**
-- ══════════════════════════════════════════════════════════════════════
--
--   1 · المكافأةُ والعقوبةُ اليدويّة طلبٌ يوافق عليه شخصٌ ثانٍ (`incentive_requests`)
--   2 · المتجرُ يُحسب في هدف المندوب الذي فتحه للأبد (`merchants.opened_by_rep_id`)
--   3 · طلبٌ مسترجَعٌ أوصل السائقَ لمرحلة ⇒ تنبيهٌ والماليّةُ تقرّر
--   4 · متجرٌ حُذف أو تجريبيٌّ انحسب للمندوب ⇒ تنبيهٌ والماليّةُ تقرّر
--      (`incentive_alert_decisions` — وعددُ المرحلة محفوظٌ على صفّ المكافأة)
--   6 · المكافأةُ اليدويّةُ «تقدير» دائماً — لا `for_target`
--   ومكافأةُ هدفٍ تعثّرت تُسجَّل وتُعاد (`incentive_grant_failures`).

-- ── 2 · من فتح المتجر ────────────────────────────────────────────────
--
-- **كان العدُّ على المندوب الحاليّ** — فالمتجرُ المنقولُ يدخل عدّادَ الجديد
-- والقديمُ قبض عليه: مكافأةٌ مرّتين. **والآن عمودٌ يُكتب عند الإنشاء ولا
-- يتبدّل**، والنقلُ يحرّك `sales_rep_user_id` (العمولةَ القادمة) وحدَه.
ALTER TABLE merchants
    ADD COLUMN IF NOT EXISTS opened_by_rep_id uuid REFERENCES users (id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS merchants_opened_by_rep_idx
    ON merchants (opened_by_rep_id, created_at);

-- **والماضي من سجلّ النقل**: أوّلُ نقلٍ مسجَّلٍ يقول من كان صاحبَه قبله؛
-- ومن لم يُنقل قطُّ فصاحبُه الحاليُّ هو من فتحه.
UPDATE merchants m
   SET opened_by_rep_id = CASE
         WHEN t.from_rep IS NULL THEN m.sales_rep_user_id
         WHEN t.from_rep ~ '^[0-9a-f-]{36}$'
              AND EXISTS (SELECT 1 FROM users u WHERE u.id = t.from_rep::uuid)
           THEN t.from_rep::uuid
         ELSE NULL
       END
  FROM merchants m2
  LEFT JOIN LATERAL (
        SELECT COALESCE(a.details->>'from_rep', '') AS from_rep
          FROM audit_log a
         WHERE a.action = 'admin.merchant_rep_transfer'
           AND a.entity_id = m2.id::text
         ORDER BY a.created_at ASC
         LIMIT 1) t ON true
 WHERE m2.id = m.id AND m.opened_by_rep_id IS NULL;

CREATE OR REPLACE FUNCTION merchants_opened_by_rep_fixed() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        NEW.opened_by_rep_id := COALESCE(NEW.opened_by_rep_id, NEW.sales_rep_user_id);
    ELSIF NEW.opened_by_rep_id IS NOT NULL
          AND NEW.opened_by_rep_id IS DISTINCT FROM OLD.opened_by_rep_id THEN
        -- **لا يتبدّل بعد الإنشاء** — إلّا أن يُفرَّغ حين يُحذف الحساب.
        NEW.opened_by_rep_id := OLD.opened_by_rep_id;
    END IF;
    RETURN NEW;
END $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS merchants_opened_by_rep_fixed ON merchants;
CREATE TRIGGER merchants_opened_by_rep_fixed
    BEFORE INSERT OR UPDATE OF opened_by_rep_id ON merchants
    FOR EACH ROW EXECUTE FUNCTION merchants_opened_by_rep_fixed();

-- ── 3 و4 · ما بُنيت عليه مكافأةُ المرحلة ───────────────────────────────
--
-- **عددُ المرحلة يوم الصرف ودورُ صاحبها** — فإن نزل العدُّ بعدها تحتها
-- (طلبٌ مسترجَع · متجرٌ حُذف أو تجريبيّ) ظهر التنبيه، **ولو تبدّل الإعدادُ بعدها.**
ALTER TABLE incentives
    ADD COLUMN IF NOT EXISTS target_count integer,
    ADD COLUMN IF NOT EXISTS target_role text;

UPDATE incentives
   SET target_count = NULLIF(substring(reason from '— ([0-9]+)'), '')::integer,
       target_role  = CASE WHEN reason LIKE '%عميلاً%' THEN 'sales' ELSE 'driver' END
 WHERE for_target AND period IS NOT NULL AND target_count IS NULL;

-- ── 6 · اليدويُّ تقديرٌ دائماً ─────────────────────────────────────────
--
-- **كانت الشاشةُ تعلّم اليدويّةَ «عن الهدف»** لمن بلغ كلَّ المراحل، فتنتفخ
-- تقاريرُ «كم صرفنا على الأهداف». **ومكافأةُ الهدف وحدَها لها شهرٌ ومرحلة.**
UPDATE incentives SET for_target = false WHERE for_target AND period IS NULL;
ALTER TABLE incentives DROP CONSTRAINT IF EXISTS incentives_target_has_period;
ALTER TABLE incentives
    ADD CONSTRAINT incentives_target_has_period CHECK (NOT for_target OR period IS NOT NULL);

-- ── 1 · الطلبُ والموافقة ──────────────────────────────────────────────
--
-- **عقدُ الموافقات الموحّد**: id · status · amount · note · proposed_by ·
-- decided_by · created_at · decided_at. **ولا مالَ قبل الموافقة.**
CREATE TABLE IF NOT EXISTS incentive_requests (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind           text NOT NULL CHECK (kind IN ('reward', 'penalty')),
    amount         bigint NOT NULL CHECK (amount > 0),
    note           text NOT NULL CHECK (length(btrim(note)) > 0),
    status         text NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending', 'approved', 'rejected')),
    proposed_by    uuid NOT NULL REFERENCES users (id),
    decided_by     uuid REFERENCES users (id),
    decided_at     timestamptz,
    decision_note  text NOT NULL DEFAULT '',
    self_approved  boolean NOT NULL DEFAULT false,
    -- **ما قُيّد عند الموافقة** — صفُّ الحافز الذي خرج منه المال.
    incentive_id   uuid REFERENCES incentives (id) ON DELETE SET NULL,
    -- **ومن أين جاء**: يدويٌّ من الصفحة، أو استرجاعٌ قرّرته الماليّةُ على تنبيه.
    source         text NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'alert')),
    alert_incentive_id uuid REFERENCES incentives (id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS incentive_requests_pending
    ON incentive_requests (created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS incentive_requests_user
    ON incentive_requests (user_id, created_at DESC);

-- ── 3 و4 · قرارُ الماليّة على التنبيه ──────────────────────────────────
--
-- **«تبقى»** أو **«تُسترجَع»** — والاسترجاعُ طلبُ عقوبةٍ يمرّ بالموافقة نفسِها.
CREATE TABLE IF NOT EXISTS incentive_alert_decisions (
    incentive_id  uuid PRIMARY KEY REFERENCES incentives (id) ON DELETE CASCADE,
    decision      text NOT NULL CHECK (decision IN ('keep', 'clawback')),
    note          text NOT NULL DEFAULT '',
    request_id    uuid REFERENCES incentive_requests (id) ON DELETE SET NULL,
    decided_by    uuid NOT NULL REFERENCES users (id),
    decided_at    timestamptz NOT NULL DEFAULT now()
);

-- ── مكافأةُ هدفٍ تعثّرت ──────────────────────────────────────────────
--
-- **كان خطؤها يُبلَع** — ومن وقف عند الهدف بالضبط لا يقبضها أبداً. **والآن
-- تُسجَّل وتُعاد** دوريّاً ومن زرٍّ في الصفحة، لشهرها لا للشهر الجاري.
CREATE TABLE IF NOT EXISTS incentive_grant_failures (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role         text NOT NULL CHECK (role IN ('driver', 'sales')),
    month        text NOT NULL CHECK (month ~ '^[0-9]{4}-[0-9]{2}$'),
    last_error   text NOT NULL DEFAULT '',
    attempts     integer NOT NULL DEFAULT 1,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    resolved_at  timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS incentive_grant_failures_open
    ON incentive_grant_failures (user_id, role, month) WHERE resolved_at IS NULL;
