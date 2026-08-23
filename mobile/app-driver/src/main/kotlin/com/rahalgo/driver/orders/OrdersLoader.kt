package com.rahalgo.driver.orders

import com.rahalgo.shared.model.DriverMe
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.OrderRoute

/**
 * ══════════════════════════════════════════════════════════════════════
 * **دورةُ التحديث — وترتيبُ نداءاتها يُختبر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (منظومةُ اختبار تطبيق السائق، الطبقةُ الأولى — بأمر المالك
 *  ٢٠٢٦-٠٨-٢٣.)
 *
 * # ولماذا أُخرجت من نموذج العرض
 *
 * **`OrdersViewModel` ينادي المحرّكَ مباشرةً** — فلا يُختبر بلا شبكةٍ
 * ولا جهاز. **ومنطقُ الترتيب فيه هو الذي انكسر اليوم.**
 *
 * # والعطبُ الذي بُنيت هذه لتمنعه
 *
 * **كان `loadRoute()` يُنادى قبل `orders()`** — وهو يسأل عن الطلب
 * الحاليّ (`openId ?: mine.first()`). **وعند أوّل فتحةٍ لا طلبات
 * بعد**، فيردّ فراغاً ويخرج بلا نداءٍ واحد.
 *
 * **فترسم الخريطةُ احتياطيَّها**: خطٌّ مستقيمٌ من فوق البيوت. **وقِيس
 * على الطلب ١٠٣٠ — المستقيمُ ١٫٨ كم والطريقُ ٢٫٩ كم.**
 *
 * **ولم يمسكه مترجمٌ ولا حارس** — أمسكته عينُ المالك في الشارع، مرّتين.
 *
 * # وما تحرسه الاختباراتُ عليها
 *
 * **١ · المسارُ يُطلب بعد الطلبات لا قبلها** — وهو العطبُ نفسُه.
 *
 * **٢ · ويُطلب للطلب المفتوح إن وُجد** — لا لأوّل ما في اليد:
 * **من فتح طلبَه الثانيَ فرأى مسارَ الأوّل قاد إلى غير وجهته.**
 *
 * **٣ · وسقوطُ نداءٍ لا يُسقط الدورة** — **وشاشةٌ تنهار لأنّ تقريراً
 * تعثّر تمنع السائقَ من العمل**، والمسارُ زينةٌ حول الطلب لا شرطٌ له.
 */

/** **ما تحتاجه الدورةُ من المحرّك** — ولا شيءَ غيرَه. */
interface OrdersFeed {
    suspend fun queue(): List<DriverOrder>
    suspend fun orders(): List<DriverOrder>
    suspend fun me(): DriverMe
    suspend fun route(orderId: String): OrderRoute
}

/** **حصيلةُ دورةٍ واحدة.** */
data class LoadOutcome(
    val offers: List<DriverOrder>,
    val mine: List<DriverOrder>,
    val me: DriverMe?,
    /** **مسارُ الطلب الذي في يده** — وفارغٌ يعني «لم يُتاح». */
    val route: OrderRoute?,
    /** **ما تعذّر** — وفارغٌ يعني أنّ الدورةَ تمّت. */
    val error: Throwable? = null,
)

/**
 * **يجري دورةً واحدة.**
 *
 * @param openId الطلبُ المفتوحُ على الشاشة، أو فارغٌ فيُؤخذ أوّلُ ما في يده.
 */
suspend fun loadOnce(feed: OrdersFeed, openId: String?): LoadOutcome {
    val offers: List<DriverOrder>
    val mine: List<DriverOrder>
    val me: DriverMe
    try {
        offers = feed.queue()
        mine = feed.orders()
        me = feed.me()
    } catch (e: Exception) {
        // **وسقوطُ القائمة يُسقط الدورة** — **وشاشةُ طلباتٍ فارغةٌ بلا
        // سببٍ أسوأُ من رسالةِ عطب**: يظنّ أنّه لا عمل.
        return LoadOutcome(emptyList(), emptyList(), null, null, e)
    }

    // ══════════════════════════════════════════════════════════════════
    // **والمسارُ بعدها لا قبلها**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وهذا هو السطرُ الذي كان في غير موضعه.**
    val id = openId ?: mine.firstOrNull()?.id
    val route = if (id.isNullOrEmpty()) {
        null
    } else {
        // **وفشلُه صامتٌ بالتصميم** — الشاشةُ ترسم مستقيمَها ولا تسقط.
        // **والمسارُ زينةٌ حول الطلب لا شرطٌ له.**
        runCatching { feed.route(id) }.getOrNull()?.takeIf { it.available }
    }
    return LoadOutcome(offers, mine, me, route)
}
