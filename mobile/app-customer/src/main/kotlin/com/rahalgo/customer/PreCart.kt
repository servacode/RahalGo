package com.rahalgo.customer

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.model.Availability
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أيُقبَل طلبٌ إلى عنواني؟ — قبل أن أملأ سلّة** (`PC`، ٢٠٢٦-٠٩-١٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # المسألة
 *
 * **وكان أوّلُ خبرٍ يبلغه أنّ عنوانَه خارجَ النطاق يأتيه في السلّة** —
 * **بعد أن اختار وأضاف وقرأ الأسعار.** **ومن مشى الطريقَ كلَّه ليُردّ
 * في آخره يقرأ الردَّ عقوبةً لا خبرا.**
 *
 * **وزرٌّ يبدو صالحاً ثمّ يُردّ أسوأُ من زرٍّ مُعطَّلٍ بسببٍ مكتوب.**
 *
 * # والحالُ للعنوان لا للشاشة
 *
 * **وشاشةُ السوق تقرؤها وبطاقةُ الصنف تقرؤها والسلّة** — **وثلاثُ
 * نسخٍ تفترق فتقول إحداها «يُوصَّل» والأخرى «لا» في اللحظة نفسِها.**
 *
 * # وتُنسى حين يُبدَّل العنوان
 *
 * **وحالُ عنوانٍ ليست حالَ آخر** — **ومن بدّل إلى عنوانٍ خارجَ النطاق
 * ورأى الزرَّ صالحاً قرأ كذباً.** **فالمفتاحُ هو النقطةُ نفسُها**:
 * **ما لم تُقرأ لهذه النقطة فليست لها.**
 */
object Orderable {

    /** **آخرُ ما قاله الخادمُ عن هذه النقطة** — و`null` قبل أوّل قراءة. */
    var state: Availability? by mutableStateOf(null)
        private set

    /** **النقطةُ التي تصفها الحال** — وفارغٌ يعني لا حالَ لأحد. */
    var forPoint: String by mutableStateOf("")
        private set

    private var readAt: Long = 0L

    /** **مفتاحُ النقطة** — **والإحداثيّتان تصفان عنواناً بعينه.** */
    fun pointKey(lat: Double?, lng: Double?): String =
        if (lat == null || lng == null) "" else "$lat,$lng"

    /**
     * **أنحتاج سؤالَ الخادم؟**
     *
     * **والقياسُ بساعة التشغيل لا بساعة الحائط** — **من بدّل ساعةَ
     * جهازه قفز العمرُ قفزةً** (`SR`/`CC-18`).
     */
    fun stale(point: String, nowElapsed: Long, maxAgeMs: Long = FRESH_MS): Boolean =
        point.isNotEmpty() && (point != forPoint || nowElapsed - readAt >= maxAgeMs)

    fun put(av: Availability, point: String, nowElapsed: Long) {
        state = av
        forPoint = point
        readAt = nowElapsed
    }

    /** **يُنسى ما كان** — عند تبديل العنوان أو الخروج. */
    fun invalidate() {
        state = null
        forPoint = ""
        readAt = 0L
    }

    /**
     * **الحالُ لهذه النقطة وحدَها** — **و`null` لغيرها.**
     *
     * **ولا تُقرأ حالُ عنوانٍ على عنوان** — **وهو العطبُ بعينه.**
     */
    fun of(point: String): Availability? =
        if (point.isNotEmpty() && point == forPoint) state else null

    /** **عمرُ الحال المقبول** — دقيقتان، كحال المنصّة. */
    const val FRESH_MS: Long = 2 * 60 * 1000

    /** **تُصفَّر في الفحوص وحدَها.** */
    fun resetForTest() = invalidate()
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ما يفعله زرُّ «أضِف»** — **قرارٌ واحدٌ تقرؤه كلُّ شاشة**
 * ══════════════════════════════════════════════════════════════════════
 */
enum class AddAction {
    /** **يُضاف** — العنوانُ مخدومٌ أو لا نعلم بعد. */
    ADD,

    /** **يُسأل عن عنوانه أوّلاً** (`PC-02`) — **ولا يُخترَع عنوان.** */
    NEED_ADDRESS,

    /** **يُمنَع ويُقال السبب** (`PC-03`…`PC-08`). */
    BLOCKED,
}

/**
 * addAction **ماذا يفعل الزرُّ الآن؟**
 *
 * # ولا يُمنَع أحدٌ بلا علم
 *
 * **وقبل أوّل قراءةٍ يُضاف** — **ومنعٌ بلا علمٍ أسوأُ من ردٍّ بعلم**:
 * **نداءٌ لم يصل بعدُ يُغلق السوقَ على صاحبه والمنصّةُ مفتوحة.**
 * **والمحرّكُ يردّ عند الإنشاء على كلّ حال** — **وهذه تكفيه مؤونةَ أن
 * يملأ سلّةً ليُردّ في آخرها.**
 *
 * # ولا موقعُ الجهاز عنوانَ توصيل
 *
 * **ومن لا عنوانَ له يُسأل** — **ولا تُؤخذ نقطتُه الحاليّةُ حقيقةً**:
 * **من تسوّق من عمله يُوصَّل إلى بيته.**
 */
fun addAction(hasAddress: Boolean, av: Availability?): AddAction = when {
    !hasAddress -> AddAction.NEED_ADDRESS
    av == null -> AddAction.ADD
    !av.available -> AddAction.BLOCKED
    else -> AddAction.ADD
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **من يسأل الخادمَ عن العنوان** (`PC`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ولا يُسأل في كلّ رسمة** — **والرسمُ يتكرّر بلا حصرٍ في `Compose`**:
 * **نداءٌ في كلّ إعادةِ تركيبٍ يُغرق الخادمَ ويستنزف حزمةَ صاحبه**
 * (`SR-12`/`CC-26`).
 *
 * **فيُسأل حين تتبدّل النقطةُ أو تشيخ الحال** — **والحارسُ في
 * `Orderable.stale` لا في الشاشة.**
 */
class PreCartViewModel(app: android.app.Application) :
    androidx.lifecycle.AndroidViewModel(app) {

    private val api = com.rahalgo.shared.customer.CustomerApi(com.rahalgo.ui.AppCore.get().api)

    /** **نداءٌ واحدٌ في الجوّ** — **ولا اثنان لنقطةٍ واحدة.** */
    private var inFlight: String = ""

    /** **جارٍ تسجيلُ النيّة.** */
    var demandBusy by mutableStateOf(false)
        private set

    /** **سُجّلت** — فيُبدَّل الزرُّ بنصِّ ما بعد التسجيل. */
    var demandDone by mutableStateOf(false)
        private set

    /**
     * sendDemand **يسجّل نيّةَ التوسّع التي يقرّرها الخادم** (الدفعةُ
     * الرابعة).
     *
     * **ومتى يُعرَض الزرُّ تقرّره `ServiceReason`** — **ولو قرّرته
     * الشاشةُ لَعرضته يوماً لسببٍ زمنيّ.**
     */
    fun sendDemand(reason: String, address: com.rahalgo.shared.model.Address) {
        if (demandBusy || demandDone) return
        val kind = com.rahalgo.ui.ServiceReason.ctaKind(reason)
        if (kind.isEmpty()) return
        demandBusy = true
        viewModelScope.launch {
            runCatching { api.demand(address.lat, address.lng, kind, address.text) }
                .onSuccess { demandDone = true }
            demandBusy = false
        }
    }

    /**
     * refresh **يسأل إن وجب.**
     *
     * **وسقوطُ النداء لا يمنع أحداً** — **والمحرّكُ يردّ عند الإنشاء
     * على كلّ حال**، **ومنعٌ بلا علمٍ أسوأُ من ردٍّ بعلم.**
     */
    fun refresh(point: String, address: com.rahalgo.shared.model.Address?) {
        if (point.isEmpty() || address == null) {
            // **ولا عنوانَ لا حال** — **وحالُ عنوانٍ سابقٍ على لا عنوانَ
            // كذب.**
            Orderable.invalidate()
            return
        }
        val now = android.os.SystemClock.elapsedRealtime()
        if (!Orderable.stale(point, now)) return
        if (inFlight == point) return
        inFlight = point
        viewModelScope.launch {
            runCatching { api.availability(address.lat, address.lng) }
                .onSuccess {
                    Orderable.put(it, point, android.os.SystemClock.elapsedRealtime())
                }
            inFlight = ""
        }
    }
}


/**
 * ══════════════════════════════════════════════════════════════════════
 * **لافتةُ المنع — السببُ ثمّ ما يفعله** (`PC`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والنصُّ من `ServiceReason` لا من هنا** — **وسببٌ يُصاغ في شاشتين
 * يفترق فيهما يوماً.**
 *
 * **وزرُّ النيّة بالسياسة المركزيّة** — **ومن مُنع لسببٍ زمنيٍّ لا
 * يُدعى إلى طلب منطقته** (`ctaKind` تردّ فراغاً فلا يُرسَم شيء).
 */
@androidx.compose.runtime.Composable
fun ServiceBlockNotice(
    av: Availability,
    address: com.rahalgo.shared.model.Address?,
    vm: PreCartViewModel,
) {
    val ctx = androidx.compose.ui.platform.LocalContext.current
    com.rahalgo.ui.Note(
        com.rahalgo.ui.ServiceReason.text(
            ctx,
            reason = av.reason,
            message = av.message,
            placeName = av.placeName,
            nextAvailableAt = av.nextAvailableAt,
        ),
        com.rahalgo.design.Rahal.colors.danger,
    )
    val cta = com.rahalgo.ui.ServiceReason.ctaKind(av.reason)
    if (cta.isEmpty() || address == null) return
    androidx.compose.foundation.layout.Spacer(
        androidx.compose.ui.Modifier.height(8.dp),
    )
    if (vm.demandDone) {
        com.rahalgo.ui.Note(
            com.rahalgo.ui.ServiceReason.ctaDoneText(ctx, av.reason),
            com.rahalgo.design.Rahal.colors.brand,
        )
    } else {
        com.rahalgo.ui.RahalButton(
            onClick = { vm.sendDemand(av.reason, address) },
            enabled = !vm.demandBusy,
            modifier = androidx.compose.ui.Modifier.fillMaxWidth(),
        ) {
            androidx.compose.material3.Text(
                com.rahalgo.ui.ServiceReason.ctaText(ctx, av.reason, av.placeName),
            )
        }
    }
}
