-- ٠٠٥٧ · أقسامُ المنصة — **الموقعُ يعرض الأصنافَ لا المتاجر**
--
-- # المسألة
--
-- التصفّحُ كان: **اختر متجراً ثمّ صنفاً.** والزبونُ لا يفكّر هكذا — **يشتهي
-- شاورما ولا يعرف من يصنع أفضلَها**، ولا يريد أن يقارن بين عشرة مطاعم.
--
-- **وأخطرُ منه أنه يكشف المصدر.** في مدينةٍ يعرف أهلُها بعضهم، **زبونٌ رأى اسمَ
-- المطعم يتّصل به مباشرةً في المرّة القادمة** — يوفّر رسمَ التوصيل والمطعمُ
-- يوفّر عمولتنا. **وكلُّ منصةِ توصيلٍ تموت من هذا الباب لا من غيره.**
--
-- # والقسمُ غيرُ التصنيف
--
--	`categories`         ←  تصنيفُ المتجر: مطاعم · بقالة · صيدليات
--	`platform_sections`  ←  قسمُ المنصة: شاورما · بيتزا · مشاوي · خضار
--
-- **الأوّلُ يصف من نشتري منه، والثاني يصف ما نبيعه.** والزبونُ يرى الثاني
-- وحدَه.
CREATE TABLE IF NOT EXISTS platform_sections (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    icon       text NOT NULL DEFAULT '',
    sort_order int  NOT NULL DEFAULT 0,
    active     boolean NOT NULL DEFAULT true,
    -- margin_override هامشُ القسم — **يرثه كلُّ صنفٍ فيه ما لم يُخصَّ بهامش.**
    --
    -- **وهنا موضعُه لا في تصنيف المتجر**: الشاورما تُسعَّر كشاورما، **سواءٌ
    -- جاءت من مطعمٍ أو من مشاوٍ أو من كافتيريا.** وتصنيفُ المتجر يصف بائعَه
    -- لا سلعتَه، **وهامشٌ يتبع البائعَ يجعل الصنفَ الواحد بسعرين.**
    margin_override bigint,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS platform_sections_active
    ON platform_sections (sort_order, created_at) WHERE active;

-- **الصنفُ يُربط بقسم المنصة — والمتجرُ يبقى مالكَه في الدفتر.**
--
-- «كلُّ صنفٍ مربوطٌ بمتجره في القاعدة — **الإخفاءُ في الشاشة لا في الدفتر**»
-- (قرار المالك)، فيبقى القيدُ واضحاً والحسابُ صحيحاً.
--
-- **وفراغُه مشروع**: صنفٌ لم يُصنَّف بعد. **ولا يُعرض في التصفّح** — ويبقى
-- قابلاً للطلب من صفحة متجره، فلا ينقطع ما كان يعمل.
ALTER TABLE menu_items ADD COLUMN IF NOT EXISTS platform_section_id uuid
    REFERENCES platform_sections(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS menu_items_by_platform_section
    ON menu_items (platform_section_id) WHERE platform_section_id IS NOT NULL;

-- **وهامشُ تصنيف المتجر ينتقل إلى القسم.**
--
-- وُضع قبل ساعةٍ في `0056` حين كان `categories` التصنيفَ الوحيد. **وعمودٌ
-- يبقى بلا قارئ كلفةُ كتابةٍ بلا فائدة** — وأسوأ: يُملأ يوماً فلا يُقرأ،
-- فيُظنّ الهامشُ موضوعاً وهو لا يعمل.
ALTER TABLE categories DROP COLUMN IF EXISTS margin_override;

-- بذورُ الأقسام — **قابلةٌ للتعديل والحذف من اللوحة.**
--
-- **ولا تُترك فارغةً بانتظار المالك**: موقعٌ يُفتح بلا أقسامٍ موقعٌ فارغ،
-- **ومن رآه فارغاً مرّةً لا يعود ليرى إن امتلأ.**
INSERT INTO platform_sections (name, icon, sort_order) VALUES
    ('شاورما',        'food',    1),
    ('بيتزا وفطائر',  'food',    2),
    ('مشاوي',         'food',    3),
    ('برغر',          'food',    4),
    ('وجبات شعبية',   'food',    5),
    ('حلويات',        'sweets',  6),
    ('مشروبات',       'drinks',  7),
    ('خضار وفواكه',   'grocery', 8),
    ('بقالة',         'grocery', 9)
ON CONFLICT DO NOTHING;
