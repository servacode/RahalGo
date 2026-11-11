-- قوائم شكاوى المتجر (my-reports و reports-against) كانت تمسح جدول tickets كلَّه:
-- لا فهرس على created_by ولا على order_id إلّا الجزئيّ للمفتوح.
-- قيس على 300 ألف شكوى: my-reports 83 مل.ث بمسح كامل، و0.5 مل.ث بالفهرس.
CREATE INDEX IF NOT EXISTS tickets_created_by_idx ON tickets (created_by, created_at DESC);
CREATE INDEX IF NOT EXISTS tickets_order_idx ON tickets (order_id) WHERE order_id IS NOT NULL;
