-- ══════════════════════════════════════════════════════════════════════
-- **الفروعُ والمناطقُ التشغيليّة — والجغرافيا ليست فرعاً**
-- ══════════════════════════════════════════════════════════════════════
--
-- (خريطةُ العمليات — `MAP-4`. البنود ١٩ و٢٠ و٢١ و٢٢ و٤١.)
--
-- # الهرمُ المعتمد
--
--     محافظة ← مدينة ← فرعُ المدينة الرئيسيّ ← مناطقُ تشغيليّة
--                            └── فروعٌ فرعيّةٌ اختياريّة
--
-- # ولماذا لا تصير «المنطقةُ الإداريّة» فرعاً تلقائيّاً (البند ١٩)
--
-- **`Geography ≠ Branch`.** **المنطقةُ الإداريّةُ تقسيمُ الدولة، ثابتٌ
-- لا نخترعه** (`districts` — الهجرة ٠١٢٢). **والفرعُ قرارُ عملٍ نتّخذه**:
-- مكتبٌ ومديرٌ وسائقون.
--
-- **ومن جعل كلَّ منطقةٍ فرعاً افتتح ثلاثةً وستّين فرعاً في يومٍ واحد** —
-- **وكلُّها فارغة.**
--
-- # وثلاثةُ جداولَ لا واحد
--
-- **والمنطقةُ التشغيليّةُ ليست منطقةَ تغطية**: تلك مضلَّعٌ نرسمه ونسعّره
-- (`delivery_zones`)، **وهذه تقسيمُ عملٍ داخليٌّ** — «حيُّ المشلب»
-- يُسنَد إلى فرعٍ ويُتابَع أداؤه. **وقد ترتبط بمنطقة تغطيةٍ وقد لا
-- ترتبط**، فالربطُ اختياريّ.
--
-- # ولا نظامَ موارد بشريّة (البند ٢١)
--
-- **ولا حقلَ «مدير»** — **ونظامُ الموظّفين اليومَ أدوارٌ في `user_roles`
-- لا انتماءَ لفرع.** **ومن أضاف عموداً يشير إلى موظّفٍ بنى نصفَ نظامٍ
-- لا يُقرأ.**

-- ══════════════════════════════════════════════════════════════════════
-- **الفروع**
-- ══════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS branches (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL CHECK (btrim(name) <> ''),
    -- **`primary` فرعُ المدينة، و`sub` فرعٌ تحته.**
    type       text NOT NULL DEFAULT 'primary',
    city_id    uuid NOT NULL REFERENCES cities(id) ON DELETE RESTRICT,
    -- **والأبُ فرعٌ لا مدينة** — `RESTRICT` فلا يُمحى أبٌ تحته أبناء.
    parent_id  uuid REFERENCES branches(id) ON DELETE RESTRICT,
    -- **و`active` لا حذف** — فرعٌ أُغلق يبقى مقروءاً فيما نُسب إليه.
    status     text NOT NULL DEFAULT 'active',
    -- **والموقعُ يقبل الفراغ**: فرعٌ يُسجَّل قبل أن يُستأجر مكتبُه.
    location   geography(Point, 4326),
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'branches_type_ck') THEN
        ALTER TABLE branches ADD CONSTRAINT branches_type_ck
            CHECK (type IN ('primary', 'sub'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'branches_status_ck') THEN
        ALTER TABLE branches ADD CONSTRAINT branches_status_ck
            CHECK (status IN ('active', 'inactive'));
    END IF;
    -- **والهرمُ يُحرَس في القاعدة لا في المعالج** (البند ٤١):
    -- **فرعُ مدينةٍ لا أبَ له، وفرعٌ فرعيٌّ لا بدَّ له من أب.**
    -- **ومن حرس بالمعالج وحدَه ترك بابَ الحقن مفتوحاً.**
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'branches_parent_ck') THEN
        ALTER TABLE branches ADD CONSTRAINT branches_parent_ck
            CHECK ((type = 'primary' AND parent_id IS NULL)
                OR (type = 'sub' AND parent_id IS NOT NULL));
    END IF;
END $$;

-- **ولكلّ مدينةٍ فرعٌ رئيسيٌّ واحد** (البند ٢٠) — **والتعدّدُ بالفروع
-- الفرعيّة.** **وقيدٌ جزئيٌّ لا كلّيّ**: الفروعُ الفرعيّةُ تتعدّد بلا حدّ.
CREATE UNIQUE INDEX IF NOT EXISTS branches_one_primary_per_city
    ON branches (city_id) WHERE type = 'primary';

CREATE INDEX IF NOT EXISTS branches_city_idx ON branches (city_id);
CREATE INDEX IF NOT EXISTS branches_parent_idx ON branches (parent_id)
    WHERE parent_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS branches_location_idx ON branches USING gist (location)
    WHERE location IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS branches_city_name_key ON branches (city_id, name);

-- ══════════════════════════════════════════════════════════════════════
-- **المناطقُ التشغيليّة**
-- ══════════════════════════════════════════════════════════════════════
--
-- **و`ON DELETE SET NULL` للفرع**: فرعٌ يُحذف لا يمحو المنطقةَ نفسَها —
-- **تبقى بلا فرعٍ حتّى تُسنَد.** **وحذفُ الفرع ممنوعٌ أصلاً ما دام
-- تحته شيء** (`RESTRICT` أعلاه)، **وهذا احتياطٌ ثانٍ لا بديل.**
CREATE TABLE IF NOT EXISTS operational_areas (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL CHECK (btrim(name) <> ''),
    city_id    uuid NOT NULL REFERENCES cities(id) ON DELETE RESTRICT,
    branch_id  uuid REFERENCES branches(id) ON DELETE SET NULL,
    -- **والربطُ بمنطقة تغطيةٍ اختياريّ** — **وهي شكلٌ وهذه تنظيم.**
    zone_id    uuid REFERENCES delivery_zones(id) ON DELETE SET NULL,
    active     boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS operational_areas_city_name_key
    ON operational_areas (city_id, name);
CREATE INDEX IF NOT EXISTS operational_areas_branch_idx
    ON operational_areas (branch_id) WHERE branch_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS operational_areas_zone_idx
    ON operational_areas (zone_id) WHERE zone_id IS NOT NULL;
