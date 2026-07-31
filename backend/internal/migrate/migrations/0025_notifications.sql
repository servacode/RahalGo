-- الإشعارات: صندوق وارد دائم لكل مستخدم. الركيزة الأساسية أن يبقى الجميع على
-- اطلاع بأي حركة جديدة دون تحديث الصفحة — والبث الحي وحده لا يكفي: ما يقع
-- والمستخدم غير متصل يجب أن يجده عند عودته.
CREATE TABLE notifications (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind        text NOT NULL,                    -- order | ticket | wallet | rating | lead | account …
    title       text NOT NULL,
    body        text NOT NULL DEFAULT '',
    entity      text NOT NULL DEFAULT '',         -- نوع الكيان (order/ticket/…)
    entity_id   text NOT NULL DEFAULT '',         -- معرّفه للانتقال إليه
    href        text NOT NULL DEFAULT '',         -- وجهة النقر في الواجهة
    read_at     timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- الاستعلام الأشيع: إشعارات مستخدم مرتبة زمنياً + عدّ غير المقروء
CREATE INDEX notifications_user_idx ON notifications (user_id, created_at DESC);
CREATE INDEX notifications_unread_idx ON notifications (user_id) WHERE read_at IS NULL;
