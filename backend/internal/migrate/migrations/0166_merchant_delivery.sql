-- ══════════════════════════════════════════════════════════════════════
-- **«لدي توصيلة» — توصيلةُ المتجر** — قراراتُ المالك ٢٠٢٦-٠٩-٢٩
-- ══════════════════════════════════════════════════════════════════════
--
-- **خدمةُ توصيلٍ لا بيعٌ في سوق.** زبونٌ اشترى من المتجر **خارجَ رحّال غو**
-- (اتّصالٌ أو واتساب)، والمتجرُ جهّز الغرضَ **ويطلب سائقاً وحدَه.**
--
-- **فلا أصنافَ ولا سلّةَ ولا عمولةَ مبيعات** — **ومالُ التجارةِ منفصلٌ عن مالِ
-- التوصيل** (قرارُ المالك ٨). وما يُحسب هنا أجرةُ توصيلٍ وحدَها.
--
-- # نوعٌ ثالثٌ لا نظامٌ موازٍ
--
-- **للطلبِ `kind` أصلاً** (`standard`/`custom`)، **والتوصيلةُ ثالثُها.**
-- وتُعاد كما هي: آلةُ الحالات، ومحرّكُ التوزيع بالقرب، والراصد، وشاشةُ
-- الرحلة، وإثباتُ التسليم، ودفترُ المال. **ولا جدولَ موازياً ولا محرّك.**
--
-- **والاسمُ `merchant_delivery`** بقرار المالك صراحةً (٢٠٢٦-٠٩-٢٩) — **وكان
-- العقدُ المكتوبُ ٢٠٢٦-٠٩-٢١ يسمّيه `external`، فنُسخ الاسمُ ويبقى تاريخُه
-- في سجلّ القرارات.**
--
-- # وأوّلُه يختلف وطريقُه واحد
--
-- **المتجرُ هو المُنشئ لا القابل** — فلا `pending` ولا `accepted` ولا
-- `preparing`: **يُولد الطلبُ في الطابور مباشرةً.** وما بعده كالعاديّ حرفاً:
-- إسنادٌ ⇐ وصلَ المتجرَ ⇐ استلم ⇐ في الطريق ⇐ وصلَ المستلمَ ⇐ سُلّم.
--
-- # والمالُ ببدائيّاتٍ قائمةٍ لا بمسارٍ جديد
--
-- **`delivery_fee` ما يدفعه الدافع، و`driver_fee` ما يأخذه السائق** (هجرة
-- 0164)، **والفرقُ يبلغ الخزينةَ من نفسِه** عبر الحساب الفرقيِّ في
-- `creditTreasury` (بقيّةٌ = مدفوعٌ − مستردٌّ − ما وصل الأطراف). **فنسبةُ
-- المنصّة تُنفَّذ بلا قيدٍ مخترَع** — وهي ثنائيّةُ القيد بنيويّاً.
--
-- **والنسبةُ تُلقَط على الطلب** لا تُقرأ من الإعداد وقتَ التسوية: **تغييرُها
-- غداً لا يمسّ طلبَ اليوم** (قرارُ المالك ٦).

-- ── ١ · النوعُ الثالث ─────────────────────────────────────────────────
--
-- **ويُبدَّل القيدُ لا يُزاد**: قيدان على عمودٍ واحدٍ يمرّ أحدُهما ويسقط
-- الآخرُ **فيُقرأ الرفضُ لغزاً.**
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_kind_check;
ALTER TABLE orders ADD CONSTRAINT orders_kind_check
    CHECK (kind = ANY (ARRAY['standard', 'custom', 'merchant_delivery']));

-- ── ٢ · المستلِمُ — ولا حسابَ يُنشأ له ────────────────────────────────
--
-- **المستلِمُ ليس مستخدمَ رحّال غو ولا يُشترط أن يكون** (قرارُ المالك ٤):
-- **لا تسجيلَ ولا دعوةَ ولا رسالةَ خارجيّة.** والمتجرُ يكتب ما يلزم للتوصيل
-- وحدَه.
--
-- **و`customer_id` يبقى فارغاً** — **ولا يُعبَّأ بصاحب المتجر**: من عبّأه به
-- جعل التوصيلةَ تظهر في «طلباتي» عند صاحب المتجر بوصفه زبوناً، **وخلَطَ
-- مالَ التجارة بمال التوصيل في كلّ تقرير.**
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS recipient_name  text,
    ADD COLUMN IF NOT EXISTS recipient_phone text,
    -- **وصفُ الغرضِ اختياريٌّ ومحدود** — يقرؤه السائقُ ليعرف حجمَ ما يحمل.
    ADD COLUMN IF NOT EXISTS parcel_note     text,
    -- **ومن يدفع أجرةَ التوصيل** — متجرٌ أو مستلِم (قرارُ المالك ٦).
    ADD COLUMN IF NOT EXISTS fee_payer       text,
    -- **ولقطةُ نصيبِ المنصّة** — بالمئة، تُثبَّت لحظةَ الإنشاء.
    ADD COLUMN IF NOT EXISTS snap_platform_delivery_percent int NOT NULL DEFAULT 0;

-- ── ٣ · وما يلزم التوصيلةَ يُفرَض عليها وحدَها ────────────────────────
--
-- **والشرطُ يُكتب `kind <> '…' OR …`** — فلا يمسّ العاديَّ ولا الخاصّ.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_merchant_delivery_shape;
ALTER TABLE orders ADD CONSTRAINT orders_merchant_delivery_shape CHECK (
    kind <> 'merchant_delivery' OR (
        -- **الالتقاطُ متجرُ صاحبِه** — ولا توصيلةَ بلا متجرٍ يُقصد.
        merchant_id IS NOT NULL
        -- **والمستلِمُ يُعرَف باسمه ورقمه** — والسائقُ يحتاجهما ليسلّم.
        AND recipient_name IS NOT NULL AND length(btrim(recipient_name)) > 0
        AND recipient_phone IS NOT NULL AND length(btrim(recipient_phone)) > 0
        -- **وعنوانٌ مكتوبٌ يُقرأ** — والدبّوسُ وحدَه يضيع في زقاق.
        AND address_text IS NOT NULL AND length(btrim(address_text)) > 0
        AND fee_payer IN ('merchant', 'recipient')
    )
);

-- **ودافعُ الأجرةِ لا معنى له في غيرها** — **وحقلٌ يُملأ حيث لا يُقرأ يُقرأ
-- يوماً.**
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_fee_payer_only_delivery;
ALTER TABLE orders ADD CONSTRAINT orders_fee_payer_only_delivery
    CHECK (kind = 'merchant_delivery' OR fee_payer IS NULL);

-- **وبياناتُ المستلِمِ كذلك** — لا تُكتب في طلبٍ له زبونٌ بحساب.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_recipient_only_delivery;
ALTER TABLE orders ADD CONSTRAINT orders_recipient_only_delivery CHECK (
    kind = 'merchant_delivery'
    OR (recipient_name IS NULL AND recipient_phone IS NULL AND parcel_note IS NULL)
);

-- ── ٤ · سقفُ ائتمانِ المتجر — **صفرٌ افتراضاً** ───────────────────────
--
-- (قرارُ المالك ٢، ٢٠٢٦-٠٩-٢٩: «لا يُسمح للمتجر بتكوين دين للمنصة
--  افتراضياً. الإدارة وحدها تستطيع منح/تعديل credit limit».)
--
-- **والائتمانُ محسوبٌ في `financial_obligations` القائم** (`Create`/`Settle`/
-- `Balance`، وخلفَه `merchants.debt`) — **وهو كيف «يدين» المتجرُ، لا محفظةٌ
-- سالبةٌ بلا حدّ.**
--
-- **وصفرٌ يعني: ادفع من محفظتك أو لا توصيلة** — ولا دينَ يتراكم بصمت.
ALTER TABLE merchants
    ADD COLUMN IF NOT EXISTS delivery_credit_limit bigint NOT NULL DEFAULT 0;

ALTER TABLE merchants DROP CONSTRAINT IF EXISTS merchants_delivery_credit_limit_nonneg;
ALTER TABLE merchants ADD CONSTRAINT merchants_delivery_credit_limit_nonneg
    CHECK (delivery_credit_limit >= 0);

-- ── ٥ · فهرسٌ لقوائم المتجر ───────────────────────────────────────────
--
-- **وشاشةُ المتجر تسأل «توصيلاتي»** — فهرسٌ على (المتجر، النوع، الإنشاء)
-- يمنع مسحَ جدول الطلبات كلِّه في كلّ فتحة.
CREATE INDEX IF NOT EXISTS idx_orders_merchant_delivery
    ON orders (merchant_id, created_at DESC)
    WHERE kind = 'merchant_delivery';

-- ── التراجع ───────────────────────────────────────────────────────────
--
-- **ولا تُحذف الأعمدةُ في تراجع** — صفٌّ واحدٌ من نوعِ التوصيلة يمنع إعادةَ
-- القيد القديم، **فالتراجعُ يبدأ بإغلاق البابِ ثمّ إغلاق الصفوف القائمة**:
--
--   1. يُمنع الإنشاءُ من الطبقة (الباب)، ولا تُلمس القاعدة.
--   2. تُغلَق التوصيلاتُ الحيّةُ بمسارها المدعوم (إلغاءٌ أو تسليم).
--   3. عندها وحدَها: DROP CONSTRAINT ثمّ إعادتُه بنوعين.
--
-- **وحذفُ عمودٍ فيه بياناتُ مستلِمٍ حذفٌ لسجلِّ تسليمٍ وقع** — لا يُفعل.
