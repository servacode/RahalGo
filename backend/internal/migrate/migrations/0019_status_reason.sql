-- سبب الإيقاف/الحظر — موثق وظاهر (ملاحظة مراجعة قسم الحسابات)
ALTER TABLE users ADD COLUMN status_reason text NOT NULL DEFAULT '';
