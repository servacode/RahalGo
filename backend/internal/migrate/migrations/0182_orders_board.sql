-- ══════════════════════════════════════════════════════════════════════
--  لوحةُ الطلبات — قراراتُ المالك ٢٠٢٦-١٠-٠٤
-- ══════════════════════════════════════════════════════════════════════
--
-- # ١ · الإنذارُ يتجدّد حين يتبدّل سببُه
--
-- كان `alerted_at` يُكتب مرّةً للطلب كلِّه — **فطلبٌ أُنذر لأنّه «لم يُقبل»
-- ثمّ علق «بلا سائق» لا يُنذَر ثانيةً.** فيُحفظ معه السببُ الذي أُنذر به،
-- **والإنذارُ يقع مع كلّ سببٍ جديد.**
ALTER TABLE orders ADD COLUMN IF NOT EXISTS alerted_reason text;

-- # ٢ · موظّفُ العمليّات يرى الطوارئ ويستلمها (القرار ٧)
--
-- **قدرةٌ بعينها لا `support.manage` كلُّها** — تلك تذاكرُ ونزاعاتٌ وتقييمات،
-- **والعمليّاتُ تحتاج الطارئ وحدَه.** وتُمنح لكلّ دورٍ كان يبلغ الطوارئ
-- (`support.manage`) فلا يفقد أحدٌ ما كان يراه، **ثمّ للعمليّات.**
INSERT INTO role_capabilities (role_code, capability_code)
SELECT DISTINCT rc.role_code, 'emergencies.manage'
  FROM role_capabilities rc
 WHERE rc.capability_code = 'support.manage'
ON CONFLICT DO NOTHING;

INSERT INTO role_capabilities (role_code, capability_code)
SELECT r.code, 'emergencies.manage'
  FROM roles r
 WHERE r.code IN ('operations', 'owner_super_admin', 'admin')
ON CONFLICT DO NOTHING;

-- # ٣ · «إعادةُ حساب التسوية» للمالك والأدمن وحدَهما (القرار ١٥)
--
-- **قيدٌ ماليٌّ يُنشأ بيد** — وكان بقدرة `orders.intervene` فيملكه موظّفُ
-- العمليّات. **قدرةٌ بعينها تُمنح لصاحب المنصّة ومديرها لا غير.**
INSERT INTO role_capabilities (role_code, capability_code)
SELECT r.code, 'finance.recompute'
  FROM roles r
 WHERE r.code IN ('owner_super_admin', 'admin')
ON CONFLICT DO NOTHING;
