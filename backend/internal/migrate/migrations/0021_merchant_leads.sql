-- طلبات انضمام المتاجر عبر رابط/باركود المندوب (المندوب أداة نمو المنصة).
-- المتجر يملأ نموذجاً عبر رابط المندوب فيُلتقط منسوباً له، ثم يحوّله الأدمن لمتجر.
CREATE TABLE merchant_leads (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    store_name        text NOT NULL,
    owner_name        text NOT NULL DEFAULT '',
    phone             text NOT NULL,
    area              text NOT NULL DEFAULT '',
    note              text NOT NULL DEFAULT '',
    sales_rep_user_id uuid REFERENCES users (id),   -- المندوب صاحب الكود
    -- new: بانتظار المعالجة | converted: حُوّل لمتجر | rejected: مرفوض
    status            text NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'converted', 'rejected')),
    merchant_id       uuid REFERENCES merchants (id),  -- المتجر الناتج عند التحويل
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX merchant_leads_rep_idx ON merchant_leads (sales_rep_user_id);
CREATE INDEX merchant_leads_status_idx ON merchant_leads (status);
