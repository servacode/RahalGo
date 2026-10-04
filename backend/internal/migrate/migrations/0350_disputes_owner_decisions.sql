-- ══════════════════════════════════════════════════════════════════════
-- **الخسائر والنزاعات — قرارات المالك 2026-10-04**
-- ══════════════════════════════════════════════════════════════════════
--
-- # 1 · من يرى النزاعات ومن يحسمها (القرار 1)
--
-- كان عرضُ النزاعات وفتحُها بقدرة `support.manage`، والماليّةُ لا تملكها
-- (نُزعت منها عمداً في 0141) — **فالماليّةُ تحسم ولا ترى ما تحسمه.**
-- **قدرةٌ بعينها**: `disputes.manage` = عرضُ النزاعات وفتحُ نزاعٍ يدويّ.
-- تُمنح لكلّ دورٍ كان يبلغ النزاعات (`support.manage`) فلا يفقد أحدٌ ما كان
-- يراه، **ثمّ للماليّة والأدمن والمالك الأعلى.** والحسمُ يبقى `finance.manage`.
INSERT INTO role_capabilities (role_code, capability_code)
SELECT DISTINCT rc.role_code, 'disputes.manage'
  FROM role_capabilities rc
 WHERE rc.capability_code = 'support.manage'
ON CONFLICT DO NOTHING;

INSERT INTO role_capabilities (role_code, capability_code)
SELECT r.code, 'disputes.manage'
  FROM roles r
 WHERE r.code IN ('finance', 'admin', 'owner_super_admin')
ON CONFLICT DO NOTHING;

-- # 2 · المسترَدُّ يُحفظ — والخصمُ الجزئيّ ممكن (القرار 3)
--
-- **رصيدٌ لا يكفي يُخصم منه الموجود ويبقى الباقي مفتوحاً.** فالنزاعُ يحفظ
-- ما استُرِدّ منه، **والمفتوحُ = المبلغ − المسترَدّ.**
ALTER TABLE disputes ADD COLUMN IF NOT EXISTS recovered bigint NOT NULL DEFAULT 0;
-- ما خُصم كاملاً قبل اليوم مسترَدٌّ كلُّه.
UPDATE disputes SET recovered = amount
 WHERE status = 'settled' AND settlement = 'charged' AND recovered = 0;
ALTER TABLE disputes DROP CONSTRAINT IF EXISTS disputes_recovered_range;
ALTER TABLE disputes ADD CONSTRAINT disputes_recovered_range
    CHECK (recovered >= 0 AND recovered <= amount);

-- # 3 · نزاعٌ محسومٌ لا يكبر (المشكلة 4)
--
-- كان الفهرسُ الفريدُ على (الطلب · الطرف) كلَّ عمره — **فتعويضٌ ثانٍ بعد
-- الحسم يُضاف إلى النزاع المحسوم ويبقى «محسوماً» فلا يُطالَب به أبداً.**
-- **والآن الفريدُ على المفتوح وحدَه**: تعويضٌ جديدٌ بعد الحسم يفتح نزاعاً جديداً.
DROP INDEX IF EXISTS disputes_order_party_idx;
CREATE UNIQUE INDEX IF NOT EXISTS disputes_order_party_open_idx
    ON disputes (order_id, party_role)
    WHERE order_id IS NOT NULL AND status = 'open';

-- # 4 · الحسمُ اقتراحٌ وموافقةٌ من شخصٍ آخر (القرار 2)
--
-- **عقدُ الموافقات الموحَّد**: id · status · amount · note · proposed_by ·
-- decided_by · created_at · decided_at.
--   action    charge (اخصم) · waive (أسقط)
--   amount    المفتوحُ يومَ الاقتراح
--   charged   ما خُصم فعلاً عند الموافقة — قد يقلّ عن المبلغ إن لم يكفِ الرصيد
CREATE TABLE IF NOT EXISTS dispute_resolutions (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    dispute_id    uuid NOT NULL REFERENCES disputes(id) ON DELETE CASCADE,
    action        text NOT NULL CHECK (action IN ('charge', 'waive')),
    status        text NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'approved', 'rejected')),
    amount        bigint NOT NULL CHECK (amount > 0),
    charged       bigint,
    note          text NOT NULL,
    proposed_by   uuid NOT NULL REFERENCES users(id),
    decided_by    uuid REFERENCES users(id),
    decision_note text NOT NULL DEFAULT '',
    self_approved boolean NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL DEFAULT now(),
    decided_at    timestamptz
);

-- **اقتراحٌ معلَّقٌ واحدٌ لكلّ نزاع** — ضغطتان لا تصنعان اقتراحين.
CREATE UNIQUE INDEX IF NOT EXISTS dispute_resolutions_one_pending
    ON dispute_resolutions (dispute_id) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS dispute_resolutions_status_idx
    ON dispute_resolutions (status, created_at DESC);
