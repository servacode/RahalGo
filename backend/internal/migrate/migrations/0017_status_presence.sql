-- ملاحظات مراجعة قسم الحسابات:
-- 1) حالة "إيقاف مؤقت" (suspended) للتحقيقات — بين النشط والحظر.
-- 2) آخر ظهور للحساب (تتبع حضور خفيف يُحدَّث مع النشاط الموثق).

ALTER TABLE users DROP CONSTRAINT users_status_check;
ALTER TABLE users ADD CONSTRAINT users_status_check
    CHECK (status IN ('active', 'suspended', 'blocked'));

ALTER TABLE users ADD COLUMN last_seen_at timestamptz;
