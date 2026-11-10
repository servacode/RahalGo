-- مفتاح «أضف أصناف بصورة واتساب» لكل متجر (قرار المالك 2026-11-10).
-- الافتراضي تشغيل فلا يتغيّر شي عند النشر. إطفاؤه يخرج المتجر من بوت الأصناف كلياً.
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS wa_items_enabled boolean NOT NULL DEFAULT true;
