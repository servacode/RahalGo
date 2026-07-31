-- هدف المندوب الشهري (PLAN §7: «لوحة المندوب: متابعة العمولات **والأهداف**»).
-- عدد المتاجر التي يُفترض أن يجلبها المندوب في الشهر — إعداد ديناميكي يُدار من
-- لوحة الإعدادات لا رقم مدفون في الكود (GROUND-RULES §1.3).
INSERT INTO app_settings (key, value) VALUES ('sales.monthly_target', '5')
ON CONFLICT (key) DO NOTHING;
