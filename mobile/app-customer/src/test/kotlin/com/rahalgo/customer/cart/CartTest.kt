package com.rahalgo.customer.cart

/**
 * **طبقةُ الوحدة — منطقُ السلّة.**
 *
 * المعرّفات: `CART-*` · الوسم: `@unit @critical @release`
 *
 * # لماذا هي أوّلُ ما يُختبر في التطبيق بعد الصياغة
 *
 * (أولويّةُ المالك ٢٠٢٦-٠٨-١٩، البند الرابع: «Cart totals».)
 *
 * **السلّةُ هي آخرُ ما يراه الزبونُ قبل أن يدفع** — **وخطأٌ فيها لا
 * يُسقط شيئاً ويُغيّر المبلغ.** والمحرّكُ يُعيد الحسابَ عند الإرسال،
 * **لكنّ من رأى رقماً في الشاشة ثمّ خُصم غيرُه لا يثق ثانية.**
 *
 * # وتُختبر بلا جهاز
 *
 * **`save()` تخرج صامتةً إن لم يُركَّب القرص** (`store ?: return`) —
 * **فالمنطقُ في الذاكرة يعمل كاملاً بلا `Context`.** وهذا ليس التفافاً:
 * الحفظُ طبقةٌ تحت المنطق، **ولها اختبارُها حين تُبنى طبقةُ التخزين.**
 *
 * # وما لا تقيسه
 *
 * **لا تقيس الحفظَ على القرص ولا القراءةَ منه** — تلك تحتاج
 * `Context`. **ولا تقيس أجرةَ التوصيل ولا الحسم**: تُحسبان في المحرّك،
 * **ولهما `FIN-*` هناك.**
 */
import com.rahalgo.shared.model.Item
import com.rahalgo.shared.model.ModifierOption
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test

class CartTest {

    private fun item(id: String, price: Long) = Item(id = id, name = "صنف $id", price = price)

    private fun opt(id: String, delta: Long) =
        ModifierOption(id = id, name = "خيار $id", priceDelta = delta)

    /**
     * **مفتاحُ سطرٍ بلا خيارات.**
     *
     * **والسطرُ لم يعد يُعرف بمعرّف الصنف وحدَه** (٢٠٢٦-٠٨-٢٢): «برغر
     * كبير بجبنة» و«برغر صغير» صنفٌ واحدٌ ومعرّفٌ واحد، **فجمعُهما في
     * سطرٍ يُخفي أحدَ الطلبين.**
     */
    private fun key(id: String) = "$id|"

    /** **وكلُّ اختبارٍ يبدأ من سلّةٍ فارغة** — والحالُ ساكنةٌ يشترك فيها الجميع. */
    @Before
    fun empty() {
        Cart.clear()
    }

    /** CART-001 **الفارغةُ صفرٌ وصفر.** */
    @Test
    fun `السلّةُ الفارغةُ صفر`() {
        assertEquals(0, Cart.count)
        assertEquals(0L, Cart.subtotal)
    }

    /** CART-002 **والمجموعُ حاصلُ السعرِ في العدد.** */
    @Test
    fun `المجموعُ سعرٌ في عدد`() {
        Cart.add(item("a", 230), 2)
        assertEquals(2, Cart.count)
        assertEquals(460L, Cart.subtotal)
    }

    /** CART-003 **وأصنافٌ مختلفةٌ تُجمَع.** */
    @Test
    fun `الأصنافُ المختلفةُ تُجمَع`() {
        Cart.add(item("a", 230))
        Cart.add(item("b", 350))
        Cart.add(item("c", 180))
        assertEquals(3, Cart.count)
        assertEquals(760L, Cart.subtotal)
    }

    /**
     * CART-010 **والصنفُ نفسُه يرفع عدَّه ولا يُكرَّر سطرا.**
     *
     * **وسطران لصنفٍ واحدٍ يُقرآن خطأً في المتجر** — ويجعلان الحذفَ
     * يحذف نصفَ ما يظنّه صاحبُه.
     */
    @Test
    fun `الصنفُ المكرَّرُ يرفعُ العدَّ لا يُنشئُ سطرا`() {
        Cart.add(item("a", 230))
        Cart.add(item("a", 230))
        Cart.add(item("a", 230))
        assertEquals("تكرّر السطر", 1, Cart.lines.size)
        assertEquals(3, Cart.count)
        assertEquals(690L, Cart.subtotal)
    }

    /** CART-011 **والكمّيّةُ تُضبط لا تُضاف.** */
    @Test
    fun `ضبطُ الكمّيّةِ يستبدلُ لا يزيد`() {
        Cart.add(item("a", 230), 2)
        Cart.setQty(key("a"), 5)
        assertEquals(5, Cart.count)
        assertEquals(1150L, Cart.subtotal)
    }

    /**
     * CART-012 **والصفرُ يعني «ارفعه» لا «كمّيّةٌ صفر».**
     *
     * **وسطرٌ بكمّيّةِ صفرٍ يُرسَل إلى المحرّك فيُردّ** — أو أسوأ:
     * يُقبل فيصل طلبٌ بلا صنف.
     */
    @Test
    fun `الكمّيّةُ صفرٌ ترفعُ الصنفَ من السلّة`() {
        Cart.add(item("a", 230))
        Cart.add(item("b", 350))
        Cart.setQty(key("a"), 0)
        assertEquals(1, Cart.lines.size)
        assertEquals(350L, Cart.subtotal)
    }

    /** CART-013 **والسالبُ كالصفر** — ولا كمّيّةَ سالبة. */
    @Test
    fun `الكمّيّةُ السالبةُ ترفعُ الصنف`() {
        Cart.add(item("a", 230))
        Cart.setQty(key("a"), -3)
        assertEquals(0, Cart.lines.size)
        assertEquals(0L, Cart.subtotal)
    }

    /**
     * CART-014 **وضبطُ كمّيّةِ صنفٍ ليس فيها لا يفعل شيئاً.**
     *
     * **ولا يُنشئ سطراً من عدم** — معرّفٌ قديمٌ من شاشةٍ لم تُنعش
     * **يجعل سلّةً تنبت فيها أصنافٌ بلا سعر.**
     */
    @Test
    fun `ضبطُ كمّيّةِ صنفٍ غائبٍ لا يُنشئُ سطرا`() {
        Cart.add(item("a", 230))
        Cart.setQty(key("لا-وجود-له"), 9)
        assertEquals(1, Cart.lines.size)
        assertEquals(230L, Cart.subtotal)
    }

    /** CART-020 **والإفراغُ يُفرغ.** */
    @Test
    fun `الإفراغُ يمحو كلَّ شيء`() {
        Cart.add(item("a", 230), 3)
        Cart.add(item("b", 350), 2)
        Cart.clear()
        assertEquals(0, Cart.count)
        assertEquals(0L, Cart.subtotal)
        assertEquals(0, Cart.lines.size)
    }

    /**
     * CART-021 **وصنفٌ مجّانيٌّ لا يُسقط الحساب.**
     *
     * **والهدايا والعروضُ بسعرِ صفرٍ حقيقيّة** — ومن افترض سعراً موجباً
     * دائماً قسم عليه يوماً.
     */
    @Test
    fun `سعرُ صفرٍ لا يكسرُ المجموع`() {
        Cart.add(item("مجّانيّ", 0), 4)
        Cart.add(item("a", 100), 1)
        assertEquals(5, Cart.count)
        assertEquals(100L, Cart.subtotal)
    }

    // ══════════════════════════════════════════════════════════════════
    // **الخيارات — الحجمُ والإضافات** (٢٠٢٦-٠٨-٢٢)
    // ══════════════════════════════════════════════════════════════════

    /**
     * CART-030 **وفرقُ الخيار يدخل المجموع.**
     *
     * **والمحرّكُ يضيفه إلى السعرين معاً** (`orders/service.go`) —
     * **فشاشةٌ لا تضيفه تعرض رقماً ويُخصم غيرُه.**
     */
    @Test
    fun `فرقُ الخيارِ يدخلُ المجموع`() {
        Cart.add(item("برغر", 300), 2, listOf(opt("كبير", 80), opt("جبنة", 60)))
        assertEquals(2, Cart.count)
        assertEquals(880L, Cart.subtotal)
    }

    /**
     * CART-031 **والصنفُ نفسُه بخيارين مختلفين سطران.**
     *
     * **ولو جُمعا لَاختفى أحدُ الطلبين** — ويصنع المتجرُ متطابقين.
     */
    @Test
    fun `الخياراتُ المختلفةُ تفصلُ السطرين`() {
        Cart.add(item("برغر", 300), 1, listOf(opt("كبير", 80)))
        Cart.add(item("برغر", 300), 1, listOf(opt("صغير", 0)))
        assertEquals(2, Cart.lines.size)
        assertEquals(680L, Cart.subtotal)
    }

    /** CART-032 **والخياراتُ نفسُها ترفع العدَّ ولا تُكرّر سطرا.** */
    @Test
    fun `الخياراتُ نفسُها ترفعُ العد`() {
        Cart.add(item("برغر", 300), 1, listOf(opt("كبير", 80)))
        Cart.add(item("برغر", 300), 2, listOf(opt("كبير", 80)))
        assertEquals(1, Cart.lines.size)
        assertEquals(3, Cart.count)
    }

    /**
     * CART-033 **وترتيبُ الاختيار لا يصنع سطراً ثانيا.**
     *
     * **«جبنة ثمّ ثوم» و«ثوم ثمّ جبنة» اختيارٌ واحد** — **ولو لم
     * يُرتَّب المفتاحُ لَصارا سطرين، ويقرأ الزبونُ سلّتَه مكرّرة.**
     */
    @Test
    fun `ترتيبُ الاختيارِ لا يفصلُ السطر`() {
        Cart.add(item("برغر", 300), 1, listOf(opt("جبنة", 60), opt("ثوم", 40)))
        Cart.add(item("برغر", 300), 1, listOf(opt("ثوم", 40), opt("جبنة", 60)))
        assertEquals(1, Cart.lines.size)
        assertEquals(2, Cart.count)
    }

    /**
     * CART-034 **والحمولةُ تحمل معرّفاتِ الخيارات.**
     *
     * **والمحرّكُ يردّ الطلبَ كلَّه** إن نقص اختيارٌ من مجموعةٍ
     * إلزاميّة — **وكان التطبيقُ لا يرسلها إطلاقاً.**
     */
    @Test
    fun `الحمولةُ تحملُ معرّفاتِ الخيارات`() {
        Cart.add(item("برغر", 300), 2, listOf(opt("كبير", 80), opt("جبنة", 60)))
        val payload = Cart.lines.single().toPayload()
        assertEquals("برغر", payload.menuItemId)
        assertEquals(2, payload.qty)
        assertEquals(listOf("كبير", "جبنة"), payload.optionIds)
    }

    /**
     * CART-035 **وحذفُ سطرٍ لا يمسّ أخاه.**
     *
     * **ولو كان الحذفُ بمعرّف الصنف لَحذفهما معاً** — ويفقد الزبونُ
     * ما لم يطلب حذفَه.
     */
    @Test
    fun `حذفُ سطرٍ لا يمسُّ أخاه`() {
        Cart.add(item("برغر", 300), 1, listOf(opt("كبير", 80)))
        Cart.add(item("برغر", 300), 1, listOf(opt("صغير", 0)))
        val first = Cart.lines.first().key
        Cart.setQty(first, 0)
        assertEquals(1, Cart.lines.size)
        assertEquals(300L, Cart.subtotal)
    }

    /**
     * CART-022 **والمجموعُ الكبيرُ لا يفيض.**
     *
     * **و`Int` تفيض عند ٢٫١ مليار** — **والليرةُ السوريّةُ تبلغها في
     * طلبٍ واحد.** فالمجموعُ `Long`، **وهذا ما يحرسه.**
     */
    @Test
    fun `المبالغُ الكبيرةُ لا تفيض`() {
        Cart.add(item("غالٍ", 500_000_000L), 9)
        assertEquals(4_500_000_000L, Cart.subtotal)
    }
}
