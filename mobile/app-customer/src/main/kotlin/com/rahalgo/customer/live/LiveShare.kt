package com.rahalgo.customer.live

import android.content.Context
import android.content.Intent
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.shared.customer.MyOrder

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مشاركةُ الموقع المباشر مع الكابتن** (قرارُ المالك ٢٠٢٦-١٠-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **دبابيسُ الخريطة في الرقّة كثيراً ما تخطئ** — فيقف الكابتنُ بعيداً عن الزبون.
 * **فحين يصير الطلبُ «بالطريق» يُسأل الزبونُ مرّةً واحدةً لكلّ طلب**: «شارك موقعي»
 * أو «لا، العنوان صحيح». **ومن شارك أرسل تطبيقُه موقعَه كلَّ عشر ثوانٍ** بخدمةٍ
 * أماميّةٍ ظاهرةٍ في الإشعارات (`LiveShareService`) حتّى يصله الطلب أو يوقفها.
 *
 * # ولماذا بخدمةٍ ظاهرة
 *
 * **سياسةُ غوغل للموقع**: فعلٌ بدأه صاحبُه، **ويراه ما دام يعمل، ويوقفه بضغطة.**
 * **ولا `ACCESS_BACKGROUND_LOCATION`** — الخدمةُ تبدأ والتطبيقُ أمامه.
 *
 * # والقرارُ في دوالَّ صافيةٍ تُختبر
 *
 * **متى يُسأل ومتى يُترك** (`LiveSharePolicy`) — **فنافذةٌ تُفتح مرّتين لطلبٍ
 * واحدٍ تُقاس بلا جهاز.**
 */
object LiveSharePolicy {

    /** **متى يُسأل** — «بالطريق» وحدَها: الكابتنُ استلم وهو آتٍ إليه. */
    const val ASK_STATUS = "on_the_way"

    /**
     * **ما دام الطلبُ في واحدةٍ منها فالمشاركةُ قائمة** — كما يقبلها المحرّك
     * (`orders.LiveStatuses`). **وما سواها نهايةٌ تُوقفها.**
     */
    val SHARE_STATUSES = setOf("assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff")

    /** **كم طلباً يُذكر قرارُه** — والأقدمُ يُنسى، فلا تنمو الذاكرةُ بلا حدّ. */
    const val MAX_REMEMBERED = 50

    /** **أوّلُ طلبٍ يُسأل عنه الآن** — بالطريق ولم يُقرَّر فيه. و`null`: لا سؤال. */
    fun pending(open: List<MyOrder>, decided: List<String>): MyOrder? =
        open.firstOrNull { it.status == ASK_STATUS && it.id !in decided }

    /** **يُذكر قرارُه** — شارك أو رفض، فلا يُسأل عن الطلب نفسِه ثانيةً. */
    fun remember(decided: List<String>, orderId: String): List<String> =
        (decided - orderId + orderId).takeLast(MAX_REMEMBERED)

    /** **أتبقى المشاركةُ قائمة؟** — طلبُها جارٍ وفي طريقه. */
    fun keepSharing(sharingId: String, open: List<MyOrder>): Boolean =
        open.any { it.id == sharingId && it.status in SHARE_STATUSES }

    /** **تُحفظ سطراً واحداً بترتيبها** — `StringSet` لا ترتيبَ لها فلا يُعرف الأقدم. */
    fun encode(decided: List<String>): String = decided.joinToString(",")

    fun decode(raw: String?): List<String> =
        raw.orEmpty().split(',').map { it.trim() }.filter { it.isNotEmpty() }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالُ المشاركة في التطبيق كلِّه** — أيُّ طلبٍ يُشارَك موقعُه الآن
 * ══════════════════════════════════════════════════════════════════════
 *
 * **تكتبه الخدمةُ وتقرؤه الشاشات** — بطاقةُ الطلب تُظهر «وقّف المشاركة» منه.
 * **والعمليّةُ واحدة** فحالٌ في كائنٍ يكفي، **ولا تخزينَ دائم**: من قتل النظامُ
 * تطبيقَه انتهت مشاركتُه، **ولا تعود بلا ضغطةٍ منه.**
 */
object LiveShare {

    /** **الطلبُ الذي يُشارَك موقعُه** — وفارغٌ: لا مشاركة. */
    var activeOrderId by mutableStateOf<String?>(null)
        internal set

    /** **يبدأ المشاركةَ لطلب** — والإذنُ يُفحص قبله في الشاشة. */
    fun start(context: Context, orderId: String) {
        val intent = Intent(context, LiveShareService::class.java)
            .putExtra(LiveShareService.EXTRA_ORDER, orderId)
        runCatching { context.startForegroundService(intent) }
    }

    /** **«وقّف المشاركة»** — تُمحى النقطةُ في المحرّك فيعود الكابتنُ إلى الدبّوس. */
    fun stop(context: Context) {
        if (activeOrderId == null) return
        val intent = Intent(context, LiveShareService::class.java)
            .setAction(LiveShareService.ACTION_STOP)
        runCatching { context.startService(intent) }
    }

    /** **يُوقف بلا نداء** — للخروج من الحساب: الجلسةُ ذهبت فلا بابَ يُنادى. */
    fun stopLocal(context: Context) {
        activeOrderId = null
        runCatching { context.stopService(Intent(context, LiveShareService::class.java)) }
    }
}
