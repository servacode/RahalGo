-- المرحلة 2 — اللبنة الأولى: المحفظة (قرار 17)
-- دفتر قيود دائم: كل حركة سطر لا يُعدَّل ولا يُحذف، والرصيد محصلة مصانة ذرّياً.

CREATE TABLE wallets (
    user_id    uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    balance    bigint NOT NULL DEFAULT 0 CHECK (balance >= 0),   -- ل.س — لا سالب أبداً
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE wallet_transactions (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    uuid   NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    amount     bigint NOT NULL CHECK (amount <> 0),               -- موجب قيد، سالب خصم
    kind       text   NOT NULL CHECK (kind IN
        ('topup',          -- شحن (نقطة بيع / تسليم نقد للعمليات)
         'order_payment',  -- دفع طلب من المحفظة
         'refund',         -- استرجاع قيمة طلب
         'compensation',   -- تعويض شكوى
         'commission',     -- عمولة مندوب
         'payout',         -- سحب/تصفية رصيد
         'adjustment')),   -- تسوية إدارية
    ref        text NOT NULL DEFAULT '',                          -- مرجع (رقم طلب...)
    note       text NOT NULL DEFAULT '',
    created_by uuid REFERENCES users (id),                        -- من نفّذ الحركة
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX wallet_tx_user_idx ON wallet_transactions (user_id, created_at DESC);
