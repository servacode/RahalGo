-- أوقات دوام المتاجر + الإغلاق الطارئ

-- day_of_week: 0=الأحد … 6=السبت
CREATE TABLE merchant_hours (
    merchant_id uuid NOT NULL REFERENCES merchants (id) ON DELETE CASCADE,
    day_of_week int  NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    closed      boolean NOT NULL DEFAULT false,
    open_time   time NOT NULL DEFAULT '09:00',
    close_time  time NOT NULL DEFAULT '23:00',
    PRIMARY KEY (merchant_id, day_of_week)
);

-- الإغلاق الطارئ يتجاوز جدول الدوام فوراً (يظهر المتجر مغلقاً مهما كان الوقت)
ALTER TABLE merchants ADD COLUMN emergency_closed boolean NOT NULL DEFAULT false;
