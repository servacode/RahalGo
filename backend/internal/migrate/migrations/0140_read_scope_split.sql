-- ══════════════════════════════════════════════════════════════════════
-- **«يقرأ» ليست صنفاً واحداً**
-- ══════════════════════════════════════════════════════════════════════
--
-- (`ADG-2` · `AQ-1` — دورةُ إصلاحٍ ٢٦ · جولةٌ ثانية، ٢٠٢٦-٠٩-٠٨.)
--
-- **أربعُ قدراتٍ انقسمت عن قدرتين** — **والانقسامُ مقيسٌ من المسارات:**
--
--   `users.export`                ← سحبُ الدليل ملفّاً
--   `finance.export`              ← الدفترُ وكشفُ الطلبات ملفّاً
--   `orders.communications.read`  ← محادثاتُ الطلب ورسائلُه ومحادثاتُ المرء
--   `users.sensitive.read`        ← دفترُ العناوين وأثرُ الحساب

-- ── الأدوارُ الكانونيّة ─────────────────────────────────────────────

-- **الدعمُ يقرأ الشكوى ويقرأ عنوانَ من اشتكى.**
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'customer_support', c FROM unnest(ARRAY[
    'orders.communications.read', 'users.sensitive.read'
]) AS c ON CONFLICT DO NOTHING;

-- **والثقةُ والسلامةُ تحقّقٌ** — **والتحقيقُ يقرأ الكلامَ والأثر.**
--
-- **ولا تصديرَ لها**: **دورٌ أمنيٌّ لا يُمنَح سحبَ القاعدة لأنّه أمنيّ.**
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'trust_safety', c FROM unnest(ARRAY[
    'orders.communications.read', 'users.sensitive.read'
]) AS c ON CONFLICT DO NOTHING;

-- **والماليّةُ تُخرج كشفَها الشهريّ** — **وهو عملُها المكتوب في
-- `orders_export.go` نفسِه**: «محاسبٌ يريد كشفاً شهرياً لا يجد ما يأخذه».
--
-- **ولا دليلَ حساباتٍ تُصدّره** — لا شأنَ لها به.
INSERT INTO role_capabilities (role_code, capability_code)
VALUES ('finance', 'finance.export') ON CONFLICT DO NOTHING;

-- **والمالكُ يملك المعجمَ كلَّه.**
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'owner_super_admin', c FROM unnest(ARRAY[
    'users.export', 'finance.export',
    'orders.communications.read', 'users.sensitive.read'
]) AS c ON CONFLICT DO NOTHING;

-- ── والأدوارُ القديمةُ تحتفظ بوصولها المقيس ─────────────────────────
--
-- **`admin` كان يبلغ كلَّ شيء** · **و`ops` و`finance` كانا يبلغان
-- المحادثاتِ والعناوينَ والأثرَ بلا حارس** (مجموعةُ الإدارة بلا
-- `RequireRoles` على هذه المسارات). **ووصولٌ قائمٌ لا يُنزَع في هجرةِ
-- إعادةِ تصنيف.**
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'admin', c FROM unnest(ARRAY[
    'users.export', 'finance.export',
    'orders.communications.read', 'users.sensitive.read'
]) AS c ON CONFLICT DO NOTHING;

INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'ops', c FROM unnest(ARRAY[
    'orders.communications.read', 'users.sensitive.read'
]) AS c ON CONFLICT DO NOTHING;

INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'finance', c FROM unnest(ARRAY[
    'orders.communications.read', 'users.sensitive.read'
]) AS c ON CONFLICT DO NOTHING;

-- ══════════════════════════════════════════════════════════════════════
-- **ونزعُ ما لا يفرضه عملٌ قائم**
-- ══════════════════════════════════════════════════════════════════════

-- **مراجعُ السائقين** — **و`GET /drivers` تُخرج الاسمَ والهاتفَ والحالَ
-- وعددَ الطلبات في صفّها.** **فلا يحتاج دليلَ الحسابات ليقرأ سائقاً.**
--
-- **ولو أُبقيت لَورث تصديرَ الدليل ودفترَ العناوين والأثرَ كلَّه.**
DELETE FROM role_capabilities
 WHERE role_code = 'driver_verification' AND capability_code = 'users.read';
