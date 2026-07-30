-- ملاحظات داخلية تراكمية على الحساب (يراها الموظفون فقط — ملاحظة مراجعة)
ALTER TABLE users ADD COLUMN admin_notes text NOT NULL DEFAULT '';
