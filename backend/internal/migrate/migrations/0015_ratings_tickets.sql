-- الجودة والدعم: التقييم المزدوج + نظام التذاكر والتعويضات (PLAN §6.7)

-- تقييم واحد لكل طلب مُسلَّم: نجوم المتجر (إلزامي) والسائق (إن وجد) + تعليق
CREATE TABLE order_ratings (
    order_id       uuid PRIMARY KEY REFERENCES orders (id) ON DELETE CASCADE,
    customer_id    uuid NOT NULL REFERENCES users (id),
    merchant_stars int  NOT NULL CHECK (merchant_stars BETWEEN 1 AND 5),
    driver_stars   int  CHECK (driver_stars BETWEEN 1 AND 5),
    comment        text NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tickets (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    number       bigint GENERATED ALWAYS AS IDENTITY (START WITH 501) UNIQUE,
    customer_id  uuid NOT NULL REFERENCES users (id),
    order_id     uuid REFERENCES orders (id),
    subject      text NOT NULL,
    status       text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'resolved')),
    compensation bigint NOT NULL DEFAULT 0 CHECK (compensation >= 0),  -- تعويض للمحفظة عند الحل
    resolution   text NOT NULL DEFAULT '',
    created_by   uuid REFERENCES users (id),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    resolved_at  timestamptz
);
CREATE INDEX tickets_status_idx ON tickets (status, created_at DESC);
CREATE INDEX tickets_customer_idx ON tickets (customer_id);

CREATE TABLE ticket_replies (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ticket_id  uuid NOT NULL REFERENCES tickets (id) ON DELETE CASCADE,
    author_id  uuid REFERENCES users (id),
    body       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ticket_replies_ticket_idx ON ticket_replies (ticket_id, id);
