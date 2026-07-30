-- نظام الهوية: المستخدمون، الأدوار (RBAC)، رموز التحقق، الجلسات، سجل التدقيق

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    phone         citext UNIQUE NOT NULL,          -- بصيغة E.164 (+963...)
    full_name     text   NOT NULL DEFAULT '',
    password_hash text,                            -- NULL = لم يضبط كلمة مرور بعد
    status        text   NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'blocked')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- الأدوار السبعة المثبتة في PLAN.md §3 — جدول مرجعي ثابت
CREATE TABLE roles (
    code        text PRIMARY KEY,
    name_key    text NOT NULL                      -- مفتاح ترجمة مركزي
);

INSERT INTO roles (code, name_key) VALUES
    ('customer', 'roles.customer'),
    ('driver',   'roles.driver'),
    ('merchant', 'roles.merchant'),
    ('sales',    'roles.sales'),
    ('ops',      'roles.ops'),
    ('finance',  'roles.finance'),
    ('admin',    'roles.admin');

CREATE TABLE user_roles (
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_code  text NOT NULL REFERENCES roles (code),
    granted_at timestamptz NOT NULL DEFAULT now(),
    granted_by uuid REFERENCES users (id),
    PRIMARY KEY (user_id, role_code)
);

-- رموز التحقق (OTP) — تُخزَّن مجزأة، صالحة لدقائق، بمحاولات محدودة
CREATE TABLE otp_codes (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    phone       citext NOT NULL,
    code_hash   text   NOT NULL,
    purpose     text   NOT NULL DEFAULT 'login',
    attempts    int    NOT NULL DEFAULT 0,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX otp_codes_phone_idx ON otp_codes (phone, created_at DESC);

-- جلسات التحديث (Refresh Tokens) — مجزأة وقابلة للإبطال، تُدوَّر عند كل استخدام
CREATE TABLE refresh_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    user_agent text NOT NULL DEFAULT '',
    ip         text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX refresh_tokens_user_idx ON refresh_tokens (user_id);

-- سجل التدقيق — كل عملية حساسة تُسجَّل (GROUND-RULES §5)
CREATE TABLE audit_log (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_user_id uuid REFERENCES users (id),
    action        text  NOT NULL,
    entity        text  NOT NULL DEFAULT '',
    entity_id     text  NOT NULL DEFAULT '',
    details       jsonb NOT NULL DEFAULT '{}',
    ip            text  NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_actor_idx ON audit_log (actor_user_id, created_at DESC);
CREATE INDEX audit_log_entity_idx ON audit_log (entity, entity_id, created_at DESC);
