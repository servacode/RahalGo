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

    /**
     * **قبل أن يقبله كابتن** — والمشاركةُ تبدأ فيها (قرارُ المالك ٢٠٢٦-١١-٢٣: «اذا فاتح
     * او مسكر يظل الموقع المباشر شغال»). **تبدأ والتطبيقُ أمامه لحظةَ التأكيد** فتبقى
     * تعمل بالخلفية بإذن «أثناء الاستخدام» — **ولا تُرسَل نقطةٌ حتّى يُسند الطلب.**
     */
    val EARLY_STATUSES = setOf("pending", "accepted", "preparing", "dispatching")

    /**
     * **ما دام الطلبُ في واحدةٍ منها فالمشاركةُ قائمة** — كما يقبلها المحرّك
     * (`orders.LiveStatuses`). **وما سواها نهايةٌ تُوقفها.**
     */
    val SHARE_STATUSES = setOf("assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff")

    /** **ما دامت المشاركةُ تعمل** — من التأكيد إلى الباب. */
    val RUN_STATUSES = EARLY_STATUSES + SHARE_STATUSES

    /** **الطلبُ العاديّ والخاصّ** — كما يقبلها المحرّك (`liveKinds`). */
    private fun liveKind(o: MyOrder) = o.kind.isEmpty() || o.kind == "standard" || o.kind == "custom"

    /**
     * **أيُّ طلبٍ تبدأ مشاركتُه تلقائيّاً الآن** — جارٍ، ولم يوقفها صاحبُه بيده، ولا مشاركةَ
     * قائمة. **ولا يُشترط أنّه لم يُقرَّر**: من قتل النظامُ تطبيقَه عادت مشاركتُه حين يفتحه.
     */
    fun autoStart(open: List<MyOrder>, stopped: List<String>, activeId: String?): MyOrder? =
        if (activeId != null) null
        else open.firstOrNull { it.status in RUN_STATUSES && liveKind(it) && it.id !in stopped }

    /** **كم طلباً يُذكر قرارُه** — والأقدمُ يُنسى، فلا تنمو الذاكرةُ بلا حدّ. */
    const val MAX_REMEMBERED = 50

    /** **أوّلُ طلبٍ يُسأل عنه الآن** — بالطريق ولم يُقرَّر فيه. و`null`: لا سؤال. */
    fun pending(open: List<MyOrder>, decided: List<String>): MyOrder? =
        // **يُسأل لحظةَ التأكيد** (٢٠٢٦-١١-٢٣) — وهو في التطبيق، **لا حين يصير بالطريق**
        // وقد أغلقه.
        open.firstOrNull { it.status in RUN_STATUSES && liveKind(it) && it.id !in decided }
            // **وطلبُ السائق يعيد السؤال** (قرارُ المالك ٢٠٢٦-١٠-٢١: «ممكن يرفض اول مرة
            // بالغلط») — كلُّ طلبٍ وقتُه مفتاحٌ جديد، **ومن يشارك الآن لا يُسأل.**
            ?: open.firstOrNull {
                it.liveRequestedAt != null && it.status in SHARE_STATUSES &&
                    requestKey(it) !in decided && LiveShare.activeOrderId != it.id
            }

    /** **مفتاحُ طلبِ السائق** — الطلبُ ووقتُ طلبه. */
    fun requestKey(o: MyOrder): String = o.id + "@" + o.liveRequestedAt.orEmpty()

    /** **يُذكر قرارُه** — شارك أو رفض، فلا يُسأل عن الطلب نفسِه ثانيةً. */
    fun remember(decided: List<String>, orderId: String): List<String> =
        (decided - orderId + orderId).takeLast(MAX_REMEMBERED)

    /** **ويُذكر الجوابُ على طلب السائق أيضاً** — فلا يُسأل عنه مرّتين. */
    fun remember(decided: List<String>, order: MyOrder): List<String> {
        var d = remember(decided, order.id)
        if (order.liveRequestedAt != null) d = remember(d, requestKey(order))
        return d
    }

    /** **أتبقى المشاركةُ قائمة؟** — طلبُها جارٍ وفي طريقه. */
    fun keepSharing(sharingId: String, open: List<MyOrder>): Boolean =
        open.any { it.id == sharingId && it.status in RUN_STATUSES }

    /** **أمبكّرٌ هو؟** — لعنوان النافذة: «السائق في الطريق» كذبٌ قبل أن يُسند. */
    fun early(o: MyOrder): Boolean = o.status in EARLY_STATUSES

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

    /**
     * **طلبُ الطرف الآخر الذي يُشارَك موقعي له** (قرارُ المالك ٢٠٢٦-١١-٢٨) — **منفصلٌ عن `activeOrderId`**
     * فلا يُخلط بطلبي أنا، **والخدمةُ الواحدةُ تحمل أحدَهما في كلّ لحظة.**
     */
    var activePartyId by mutableStateOf<String?>(null)
        internal set

    /** **يبدأ مشاركةَ الطرف الآخر** — بعد «نعم»، والإذنُ يُفحص قبله في الشاشة. */
    fun startParty(context: Context, orderId: String, amanah: Boolean) {
        val intent = Intent(context, LiveShareService::class.java)
            .putExtra(LiveShareService.EXTRA_ORDER, orderId)
            .putExtra(LiveShareService.EXTRA_PARTY, true)
            .putExtra(LiveShareService.EXTRA_PARTY_AMANAH, amanah)
        runCatching { context.startForegroundService(intent) }
    }

    /** **«إيقاف» بيده** — فلا تُستأنف تلقائيّاً لهذا الطلب، وتُمحى نقطتُه في المحرّك. */
    fun stopParty(context: Context) {
        if (activePartyId == null) return
        val intent = Intent(context, LiveShareService::class.java)
            .setAction(LiveShareService.ACTION_STOP)
        runCatching { context.startService(intent) }
    }

    /** **يبدأ المشاركةَ لطلب** — والإذنُ يُفحص قبله في الشاشة. */
    fun start(context: Context, orderId: String) {
        val intent = Intent(context, LiveShareService::class.java)
            .putExtra(LiveShareService.EXTRA_ORDER, orderId)
        runCatching { context.startForegroundService(intent) }
    }

    /**
     * **«وقّف المشاركة» بيده** — من بطاقة الطلب أو الإشعار، **فلا تعود تلقائيّاً لهذا الطلب.**
     */
    fun userStop(context: Context, orderId: String? = activeOrderId) {
        if (orderId != null) LiveStopped.add(context, orderId)
        stop(context)
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
        activePartyId = null
        runCatching { context.stopService(Intent(context, LiveShareService::class.java)) }
    }
}

/**
 * **الطلباتُ التي أوقف صاحبُها مشاركتَها بيده** — فلا تبدأ تلقائيّاً ثانيةً (٢٠٢٦-١١-٢٣).
 *
 * **يكتبها زرُّ البطاقة وزرُّ الإشعار** (الخدمة) — ومحدودةٌ بـ`MAX_REMEMBERED` كالقرارات.
 */
object LiveStopped {
    private const val FILE = "live_share"
    private const val KEY = "user_stopped"

    fun load(context: Context): List<String> = runCatching {
        LiveSharePolicy.decode(
            context.applicationContext.getSharedPreferences(FILE, Context.MODE_PRIVATE).getString(KEY, ""),
        )
    }.getOrDefault(emptyList())

    /** **وإعادةُ تشغيل التفضيل تمحوها** — من أعاد «شارك موقعي» أراد عودتَها لطلبه الجاري. */
    fun clear(context: Context) {
        runCatching {
            context.applicationContext.getSharedPreferences(FILE, Context.MODE_PRIVATE)
                .edit().remove(KEY).apply()
        }
    }

    /** **يُنسى إيقافُه بيده** — من أعاد «نعم» لطلبِ طرفٍ آخر أراد مشاركتَه. */
    fun remove(context: Context, key: String) {
        runCatching {
            val next = load(context) - key
            context.applicationContext.getSharedPreferences(FILE, Context.MODE_PRIVATE)
                .edit().putString(KEY, LiveSharePolicy.encode(next)).apply()
        }
    }

    fun add(context: Context, orderId: String) {
        runCatching {
            val next = LiveSharePolicy.remember(load(context), orderId)
            context.applicationContext.getSharedPreferences(FILE, Context.MODE_PRIVATE)
                .edit().putString(KEY, LiveSharePolicy.encode(next)).apply()
        }
    }
}
