-- طلب الانضمام يحمل بيانات كافية لتحويله إلى متجر فعلي عند موافقة الإدارة:
-- التصنيف، الموقع على الخريطة، وكلمة مرور صاحب المتجر (مُجزّأة) التي حدّدها
-- عند التسجيل عبر المندوب. لا حساب ولا متجر يُنشأ قبل الموافقة.
ALTER TABLE merchant_leads
    ADD COLUMN category_id         uuid REFERENCES categories (id),
    ADD COLUMN lat                 double precision,
    ADD COLUMN lng                 double precision,
    ADD COLUMN owner_password_hash text NOT NULL DEFAULT '';
