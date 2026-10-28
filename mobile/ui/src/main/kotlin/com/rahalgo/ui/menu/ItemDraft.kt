package com.rahalgo.ui.menu

import com.rahalgo.shared.merchant.DEFAULT_BASE_GRAMS
import com.rahalgo.shared.merchant.ModifierGroup
import com.rahalgo.shared.merchant.PRICE_MODE_NONE
import com.rahalgo.shared.merchant.PRICE_MODE_WEIGHT
import com.rahalgo.shared.merchant.weightName

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رسمُ الصنف — واحدٌ للمتجر وللمندوب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٣٠: «الأصناف بالمندوب والمتجر لازم تكون
 *  مركزيّة، فحاليّاً الإضافة عند المندوب مختلفة عن المتجر وهي لنفس
 *  الغرض».)
 *
 * # وكان رسمين متطابقين بأسماءَ مختلفة
 *
 *     المتجر   itemId · modifiers · imageMediaId
 *     المندوب  itemID · groups    · imageMediaID
 *
 * **تسعةُ حقولٍ هي هي** — والاختلافُ في حرفٍ كبيرٍ وحرفٍ صغير.
 *
 * # وثمنُ الرسمين يُدفع في كلّ إصلاح
 *
 * **وقع مرّتين في يومٍ واحد** (٢٠٢٦-٠٨-٢٩):
 *
 * - **صورةُ الصنف لا تظهر بعد رفعها** — أُصلحت في الاثنين لأنّي تذكّرت.
 * - **زرُّ «أضف خياراً» بلا إطار** — أُصلح في المتجر، **وبقي في
 *   المندوب حتّى قاله المالك.**
 *
 * **والثاني هو الشاهد**: ما يُنسى لا يُكتشف ببناءٍ ولا باختبار،
 * **إنّما بعينِ صاحب المنصّة.**
 */
data class ItemDraft(
    /** **فارغٌ يعني صنفاً جديداً** — وغيرُ الفارغ تعديلُ قائم. */
    val itemId: String = "",
    val name: String = "",
    val description: String = "",
    /** **نصٌّ لا رقم** — الحقلُ يُكتب فيه، والتحويلُ عند الحفظ. */
    val price: String = "",
    /** **قسمُ السوق** — بلاه لا يظهر الصنفُ للزبون. */
    val platformSectionId: String = "",
    /** **معرّفُ الصورة المرفوعة** — وفراغٌ صريحٌ يعني «أزِلها». */
    val imageMediaId: String? = null,
    /** **عنوانُ المصغّرة للعرض** — يأتي مع الرفع أو مع الصنف. */
    val imageThumb: String? = null,
    /** **الإضافاتُ والخيارات** — مجموعاتٌ لكلٍّ خياراتُها. */
    val modifiers: List<ModifierGroup> = emptyList(),
    /**
     * **عملةُ الصنف** «SYP» أو «USD» (طلبُ المالك ٢٠٢٦-١٠-١٢) — زرٌّ ذكيٌّ جنب
     * السعر. **وبالدولار يُكتب `priceUsd`** والليرةُ يحسبها الخادمُ من سعر صرف
     * المتجر، **وتتبعه متى تبدّل.** والإضافاتُ بعملة صنفها.
     */
    val currency: String = CURRENCY_SYP,
    /** **السعرُ بالدولار نصّاً** — «1.5» كما يُكتب. */
    val priceUsd: String = "",
    /**
     * **«أضف مشروباتي»** (ملاحظاتُ المالك ٢٠٢٦-١٠-٢٩) — قائمةُ مشروبات المتجر تُنسخ
     * إلى هذا الصنف أم لا. **وافتراضُه نعم** كما كان قبل المفتاح.
     */
    val storeDrinks: Boolean = true,
) {
    val isNew: Boolean get() = itemId.isEmpty()
    val isUsd: Boolean get() = currency == CURRENCY_USD
}

const val CURRENCY_SYP = "SYP"
const val CURRENCY_USD = "USD"

/** **دولارٌ نصّاً ⇒ رقم** — ويقبل الفاصلةَ العربيّة «٫» والإنكليزيّة. */
fun parseUsd(text: String): Double? =
    text.trim().replace('٫', '.').replace(',', '.').toDoubleOrNull()?.takeIf { it >= 0 }

/** **ما يُكتب في الحقل** — بلا أصفارٍ زائدة: 1.5 لا 1.50، و3 لا 3.0. */
fun usdText(v: Double?): String {
    if (v == null) return ""
    val cents = Math.round(v * 100)
    return if (cents % 100 == 0L) (cents / 100).toString()
    else java.math.BigDecimal(cents).movePointLeft(2).stripTrailingZeros().toPlainString()
}

/** **لأقرب ٥ ليرات** — كما يحسبها الخادم، **للعرض تحت الحقل وحدَه.** */
fun usdToSyp(usd: Double, rate: Double): Long = Math.round(usd * 100 * rate / 100 / 5) * 5

/** **الإضافاتُ تُحمَّل بنصّها الدولاريّ** — فيُحرَّر كما حُفظ. */
fun ModifierGroup.withUsdText(): ModifierGroup =
    copy(options = options.map { it.copy(usdText = usdText(it.priceDeltaUsd)) })

/**
 * **الإضافاتُ للإرسال بعملة صنفها** — دولاريٌّ يرسل `price_delta_usd`،
 * **وليريٌّ يمحوه** فلا يُحسب فرقُه من سعر الصرف بعد أن صار بالليرة.
 */
fun ItemDraft.modifiersForSave(): List<ModifierGroup> = modifiers.map { g ->
    when (g.priceMode) {
        // **بالوزن: الغراماتُ وحدَها** — والاسمُ منها والسعرُ يحسبه الخادم (٢٠٢٦-١٠-٢٨).
        PRICE_MODE_WEIGHT -> g.copy(
            name = g.name.trim(),
            baseGrams = g.baseGrams ?: DEFAULT_BASE_GRAMS,
            options = g.options.filter { (it.grams ?: 0) > 0 }
                .map { o -> o.copy(name = weightName(o.grams!!), priceDelta = 0, priceDeltaUsd = null) },
        )
        // **بدون سعر: لا فرقَ يُرسَل.**
        PRICE_MODE_NONE -> g.copy(
            name = g.name.trim(),
            options = g.options.filter { it.name.isNotBlank() }
                .map { o -> o.copy(name = o.name.trim(), priceDelta = 0, priceDeltaUsd = null) },
        )
        // **والخيارُ بلا اسمٍ يُسقط** (تدقيقُ ٢٠٢٦-١٠-٢٨) — سطرٌ فارغٌ نسيه لا يُرسَل.
        else -> g.copy(name = g.name.trim(), options = g.options.filter { it.name.isNotBlank() }.map { o ->
            if (isUsd) o.copy(name = o.name.trim(), priceDeltaUsd = parseUsd(o.usdText) ?: 0.0)
            else o.copy(name = o.name.trim(), priceDeltaUsd = null)
        })
    }
}

/**
 * **سعرُ الصنف بالليرة للمعاينة** — من الحقل أو من الدولار بسعر الصرف. و`null`
 * إن لم يُكتب بعد.
 */
fun ItemDraft.itemPriceSyp(rate: Double?): Long? =
    if (isUsd) parseUsd(priceUsd)?.let { u -> rate?.takeIf { it > 0 }?.let { usdToSyp(u, it) } }
    else price.toLongOrNull()

/**
 * **سببُ رفض الإضافات قبل الإرسال** — و`null` لا سبب. (تدقيقُ ٢٠٢٦-١٠-٢٨:
 * كان المحرّكُ يردّ «validation» عامّةً لا يفهمها صاحبُ المتجر.)
 *
 * - مجموعةٌ بلا اسم.
 * - مجموعةٌ إلزاميّةٌ بلا خيارٍ متوفّر — **فلا يقدر الزبونُ يطلب الصنفَ أصلاً.**
 */
fun ItemDraft.optionsError(): String? {
    val app = com.rahalgo.ui.AppCore.get().app
    val groups = modifiersForSave()
    if (groups.any { it.name.isBlank() }) return app.getString(com.rahalgo.ui.R.string.mod_v_group_name)
    groups.firstOrNull { g -> g.minSelect > 0 && g.options.none { it.available } }?.let {
        return app.getString(com.rahalgo.ui.R.string.mod_v_required_empty, it.name)
    }
    return null
}

/**
 * **خياراتٌ بالليرة ⇒ بعملة الصنف** — مجموعةٌ جاهزةٌ تُضاف لصنفٍ دولاريّ تُحسب
 * بالدولار من سعر الصرف. **وبلا سعر صرفٍ يبقى الحقلُ فارغاً ويُقال** (الثاني
 * `true`) — **ولا يُصفَّر سعرٌ بصمت.**
 */
fun ModifierGroup.forItemCurrency(usd: Boolean, rate: Double?): Pair<ModifierGroup, Boolean> {
    if (!usd || priceMode == PRICE_MODE_NONE || priceMode == PRICE_MODE_WEIGHT) return this to false
    var lost = false
    val opts = options.map { o ->
        when {
            o.priceDelta == 0L -> o.copy(usdText = "")
            rate != null && rate > 0 -> o.copy(usdText = usdText(o.priceDelta / rate))
            else -> { lost = true; o.copy(usdText = "") }
        }
    }
    return copy(options = opts) to lost
}

/**
 * **تبديلُ عملة الصنف يحوّل فروقَ خياراته** (تدقيقُ ٢٠٢٦-١٠-٢٨) — كانت تصير
 * صفراً بصمت. **والثاني `true`: فرقٌ لم يُحوَّل لغياب سعر الصرف.**
 */
fun ItemDraft.withCurrency(to: String, rate: Double?): Pair<ItemDraft, Boolean> {
    if (to == currency) return this to false
    val ok = rate != null && rate > 0
    var lost = false
    val groups = modifiers.map { g ->
        // **ولا فرقَ يُحوَّل في «بدون سعر» و«حسب الوزن»** — الخادمُ يحسبهما.
        if (g.priceMode == PRICE_MODE_NONE || g.priceMode == PRICE_MODE_WEIGHT) return@map g
        g.copy(options = g.options.map { o ->
            if (to == CURRENCY_USD) {
                when {
                    o.priceDelta == 0L -> o.copy(usdText = "")
                    ok -> o.copy(usdText = usdText(o.priceDelta / rate!!))
                    else -> { lost = true; o.copy(usdText = "") }
                }
            } else {
                val u = parseUsd(o.usdText)
                when {
                    u == null || u == 0.0 -> o.copy(priceDelta = 0L)
                    ok -> o.copy(priceDelta = usdToSyp(u, rate!!))
                    // **وفرقُه بالليرة من المحرّك يبقى** إن وُجد — وإلّا يُقال.
                    o.priceDelta > 0 -> o
                    else -> { lost = true; o }
                }
            }
        })
    }
    return copy(currency = to, modifiers = groups) to lost
}
