-- ══════════════════════════════════════════════════════════════════════
-- **لقطةُ اقتصادِ الطلب — ولا يبدّلها إعدادٌ لاحق** — `XQ-2`
-- ══════════════════════════════════════════════════════════════════════
--
-- (دورةُ إصلاحٍ ٣٢، ٢٠٢٦-٠٩-٠٨ · بقرارِ مالكٍ معتمَدٍ سابق.)
--
-- # العقد
--
--	EXISTING ORDER USES ITS FINANCIAL SNAPSHOT
--
-- **والنسبُ تُثبَّت لحظةَ نشوء الالتزام الماليّ** — **ولا يبدّل إعدادٌ
-- لاحقٌ اقتصادَ طلبٍ قائم.**
--
-- # واثنتان ملتقَطتان من قبل
--
-- **سعرُ البيع وسعرُ الشراء في `order_items`** · **وأجرةُ التوصيل في
-- `orders.delivery_fee`** — **فاللقطةُ ليست بدعةً هنا، هي إتمامُ ما
-- بُدئ.** **وأربعٌ بقيت تُقرأ حيّةً لحظةَ التسليم:**
--
--	snap_merchant_commission_percent   نسبةُ عمولة المنصّة   `XG-25`
--	snap_rep_commission_percent        نسبةُ المندوب          `XG-26`
--	snap_commission_source             مصدرُ احتسابها         `XG-27`
--	snap_activation_orders             عتبةُ التفعيل          `XG-28`
--
-- **والأثرُ المقيس**: **طلبٌ أُنشئ بوضعٍ وسُلّم بآخر ⇒ مئتان بدل مئة.**
--
-- # ولماذا نسبةُ العمولة لا مقدارُها
--
-- **`orders.platform_commission` مقدارٌ يُحسب لحظةَ الاستلام ويُخزَّن**
-- — **وهو نتيجةٌ لا مقدّمة.** **والنسبةُ هي ما يشيخ**، فتُلتقَط.
--
-- # وما لا يُلتقَط
--
-- **لقطةُ اقتصادٍ لا مستودعُ إعدادات** — **ولا تدخلها إعداداتُ واجهةٍ
-- ولا إشعاراتٍ ولا تشغيلٍ ولا محتوى ولا أمن.**

ALTER TABLE orders
    ADD COLUMN snap_merchant_commission_percent int,
    ADD COLUMN snap_rep_commission_percent      int,
    ADD COLUMN snap_commission_source           text,
    ADD COLUMN snap_activation_orders           int;

-- **والوضعُ من معجمه لا نصّاً حرّاً** — فلا يُكتب في اللقطة ما لا يُقرأ.
ALTER TABLE orders
    ADD CONSTRAINT orders_snap_commission_source_check
    CHECK (snap_commission_source IS NULL
        OR snap_commission_source IN ('platform_commission', 'pricing_margin', 'both'));

-- **واللقطةُ كلٌّ أو لا شيء** — **ولا نصفَ اقتصاد.**
ALTER TABLE orders
    ADD CONSTRAINT orders_economics_snapshot_check
    CHECK (
        (snap_merchant_commission_percent IS NULL AND snap_rep_commission_percent IS NULL
         AND snap_commission_source IS NULL AND snap_activation_orders IS NULL)
     OR (snap_merchant_commission_percent IS NOT NULL AND snap_rep_commission_percent IS NOT NULL
         AND snap_commission_source IS NOT NULL AND snap_activation_orders IS NOT NULL));

-- ══════════════════════════════════════════════════════════════════════
-- **والطلباتُ القديمةُ لا يُختلَق لها اقتصاد**
-- ══════════════════════════════════════════════════════════════════════
--
-- # ولماذا لا تُملأ بإعدادات اليوم
--
-- **نسبةُ يومِ الإنشاء ليست مخزَّنةً في شيء** — **فملؤها من إعدادات
-- اليوم اختراعُ ماضٍ لا استرجاعُه**، **وهو بعينه ما يمنعه `XQ-2`.**
--
-- # والمنتهيةُ لا تحتاج لقطة
--
-- **اقتصادُها وقع وقُيّد في الدفتر** — **والاسترداد يعكس ما قُيّد فعلاً
-- لا ما يُعاد حسابُه** (`XG-10`). **فتبقى لقطتُها فارغةً بلا ضرر.**
--
-- # والقائمةُ تُرفَع بأسمائها
--
-- **طلبٌ لم يُسلَّم بعدُ ولا لقطةَ له سيُسوّى بأيّ اقتصاد؟** — **ولا
-- جواب.** **فيقف الترحيلُ ويُسمّيها**، ويُحسَم بيدٍ قبل التشغيل.
--
-- **وترحيلٌ يُسوّي رقماً ليمرّ ليس ترحيلاً** — درسُ `XG-12`.
DO $$
DECLARE
    pending_count int;
    sample text;
BEGIN
    SELECT count(*), string_agg(id::text, ', ' ORDER BY id)
      INTO pending_count, sample
      FROM (SELECT id FROM orders
             WHERE closed_at IS NULL
               AND status NOT IN ('delivered', 'cancelled', 'rejected', 'failed', 'refunded')
             LIMIT 20) x;

    IF pending_count > 0 THEN
        RAISE EXCEPTION
            'XQ-2: % طلباً قائماً بلا لقطةِ اقتصاد — تُحسَم قبل تشغيل اللقطة: %',
            pending_count, sample;
    END IF;
END $$;
