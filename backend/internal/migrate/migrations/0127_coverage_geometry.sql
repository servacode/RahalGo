-- ══════════════════════════════════════════════════════════════════════
-- **التغطيةُ تصير شكلاً — والدوائرُ تبقى كما هي**
-- ══════════════════════════════════════════════════════════════════════
--
-- (خريطةُ العمليات — `MAP-3`. البنود ١٢ و١٣ و١٤ و١٥ و٤١.)
--
-- # الحالُ الذي قِيس قبل هذه الهجرة
--
-- **الهجرةُ ٠٠٠٧ بدأت بمضلَّعٍ، ثمّ حذفته ٠٠٠٩ عمداً** ووضعت مركزاً
-- ونصفَ قطر: «نموذجُ الدوائر بدل رسم المضلعات». **وكان قراراً صحيحاً
-- يومَه** — مدينةٌ واحدةٌ ومركزُها، ورسمُ مضلَّعٍ عبءٌ بلا عائد.
--
-- **وقد بطل سببُه**: التوسّعُ إلى محافظاتٍ ومناطق، **والدائرةُ تكذب على
-- الأرض** — نهرٌ يقطع المدينة، وحيٌّ خارجَ الدائرة نصلُه وآخرُ داخلَها
-- لا نصله.
--
-- # ولا تُحوَّل الدوائرُ قسراً (البند ١٤)
--
-- **`LEGACY_RADIUS` تبقى نوعاً أوّلَ الدرجة، لا حالاً انتقاليّة.**
-- **ومنطقةٌ دائريّةٌ تعمل اليومَ تبقى تعمل بحرفها** — ولا صفَّ يُمسّ،
-- ولا سلوكَ يتبدّل لمن لم يرسم شيئاً.
--
-- **والعمودُ الجديد يقبل الفراغَ** — فالهجرةُ لا تُسقط صفّاً ولا تكتب
-- في واحد.
--
-- # وعقدُ الخدمة لا يتبدّل بغفلة (البند ١٥)
--
--     Delivery Address controls serviceability
--
-- **وهو باقٍ حرفاً.** ما يتبدّل **كيف** تُقاس التغطيةُ لا **من**
-- يقرّرها: `ST_DWithin` للدائرة **و`ST_Covers` للمضلَّع**، **وكلُّ
-- منطقةٍ تُقاس بشكلها هي.**

-- ══════════════════════════════════════════════════════════════════════
-- **١ · شكلُ المنطقة**
-- ══════════════════════════════════════════════════════════════════════
--
-- **و`radius` افتراضاً** — فكلُّ صفٍّ قائمٍ يصير `LEGACY_RADIUS` بلا
-- كتابةٍ ولا تخمين.
ALTER TABLE delivery_zones
    ADD COLUMN IF NOT EXISTS shape text NOT NULL DEFAULT 'radius';

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'delivery_zones_shape_ck') THEN
        ALTER TABLE delivery_zones
            ADD CONSTRAINT delivery_zones_shape_ck
            CHECK (shape IN ('radius', 'polygon'));
    END IF;
END $$;

-- ══════════════════════════════════════════════════════════════════════
-- **٢ · المساحة**
-- ══════════════════════════════════════════════════════════════════════
--
-- **و`MultiPolygon` لا `Polygon`** (البند ١٢): مدينةٌ يقطعها نهرٌ
-- منطقتان في صفٍّ واحد، **ومن اختار `Polygon` عاد يهاجر حين تُطلَب
-- ضفّتان.** **و`MultiPolygon` يقبل الواحدةَ أيضاً**، فلا خسارة.
ALTER TABLE delivery_zones
    ADD COLUMN IF NOT EXISTS area geography(MultiPolygon, 4326);

-- **ومضلَّعٌ بلا مساحةٍ لا يُقبَل** — والقيدُ يمنع صفّاً يَعِد بما لا يملك.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'delivery_zones_area_ck') THEN
        ALTER TABLE delivery_zones
            ADD CONSTRAINT delivery_zones_area_ck
            CHECK (shape <> 'polygon' OR area IS NOT NULL);
    END IF;
END $$;

-- **وفهرسٌ مكانيّ** (البند ٤١) — **وبلاه يُمسح كلُّ مضلَّعٍ لكلّ عنوان.**
CREATE INDEX IF NOT EXISTS delivery_zones_area_idx
    ON delivery_zones USING gist (area)
    WHERE area IS NOT NULL;

-- **وفهرسٌ للشكل** — الاستعلامُ يفرّق بينهما في كلّ نداء.
CREATE INDEX IF NOT EXISTS delivery_zones_shape_idx
    ON delivery_zones (shape) WHERE active;

-- ══════════════════════════════════════════════════════════════════════
-- **٣ · طلباتُ التغطية** (البند ١٦)
-- ══════════════════════════════════════════════════════════════════════
--
-- **زرُّ «اطلب تغطية منطقتي» يحتاج مكاناً يكتب فيه** — **ومن ضغطه اليومَ
-- لا يُكتب طلبُه في مكان.**
--
-- # ولماذا نقطةٌ لا عنوانٌ نصّيٌّ وحدَه
--
-- **«حيّ المشلب» ليست موضعاً على الأرض** — **والكثافةُ لا تُرسَم بنصّ.**
-- فتُحفَظ النقطةُ ويبقى النصُّ معها لمن يقرأ.
--
-- # و`user_id` يقبل الفراغ
--
-- **ومن طلب التغطيةَ قبل أن يسجّل حسابَه طلبٌ حقيقيّ** — **ورفضُه لأنّه
-- بلا حساب يمحو أصدقَ إشارةِ طلبٍ عندنا**: من يريد الخدمةَ ولا يملكها.
--
-- **و`ON DELETE SET NULL`**: حسابٌ يُحذف لا يمحو إشارةَ طلبٍ من منطقةٍ
-- كاملة.
CREATE TABLE IF NOT EXISTS coverage_requests (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid REFERENCES users(id) ON DELETE SET NULL,
    at          geography(Point, 4326) NOT NULL,
    address_text text NOT NULL DEFAULT '',
    city_id     uuid REFERENCES cities(id),
    district_id uuid REFERENCES districts(id),
    -- **والحالاتُ خمسٌ** — و`new` هي المدخل.
    status      text NOT NULL DEFAULT 'new',
    -- **والمصدرُ يقول من أين جاء** — تطبيقُ الزبون أو الموقع أو المكتب.
    source      text NOT NULL DEFAULT 'customer_app',
    -- **وملاحظةُ المكتب داخليّةٌ** — لا يراها من طلب.
    note        text NOT NULL DEFAULT '',
    decided_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    decided_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'coverage_requests_status_ck') THEN
        ALTER TABLE coverage_requests
            ADD CONSTRAINT coverage_requests_status_ck
            CHECK (status IN ('new', 'reviewing', 'planned', 'covered', 'rejected'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS coverage_requests_at_idx
    ON coverage_requests USING gist (at);
-- **والسؤالُ الغالبُ «ما الجديدُ اليوم؟»** — فيُفهرَس به.
CREATE INDEX IF NOT EXISTS coverage_requests_status_idx
    ON coverage_requests (status, created_at DESC);
CREATE INDEX IF NOT EXISTS coverage_requests_city_idx
    ON coverage_requests (city_id) WHERE city_id IS NOT NULL;
