package com.rahalgo.ui.menu

import com.rahalgo.shared.merchant.ModifierGroup

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
    g.copy(options = g.options.map { o ->
        if (isUsd) o.copy(priceDeltaUsd = parseUsd(o.usdText) ?: 0.0)
        else o.copy(priceDeltaUsd = null)
    })
}
