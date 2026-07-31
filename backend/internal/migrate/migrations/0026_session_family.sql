-- جلسة واحدة لكل حساب، لكن الجلسة تمتد على تطبيقات المنصة الأربعة.
-- session_id يجمع توكنات التجديد التي تخصّ نفس الدخول: التسليم بين اللوحات (SSO)
-- وتدوير التوكن يبقيان داخل نفس العائلة، بينما أي دخول جديد يبطل العائلة كلها.
ALTER TABLE refresh_tokens
    ADD COLUMN IF NOT EXISTS session_id uuid NOT NULL DEFAULT gen_random_uuid();

CREATE INDEX IF NOT EXISTS refresh_tokens_session_idx
    ON refresh_tokens (user_id, session_id) WHERE revoked_at IS NULL;
