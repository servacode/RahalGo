-- الإضافات الأساسية للمنصة:
-- postgis: الاستعلامات الجغرافية (المناطق، أقرب سائق، المسافات)
-- citext:  نصوص غير حساسة لحالة الأحرف (أسماء مستخدمين، بريد)
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS citext;
