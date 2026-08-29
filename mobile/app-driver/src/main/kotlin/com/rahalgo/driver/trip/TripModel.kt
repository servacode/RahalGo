package com.rahalgo.driver.trip

import com.rahalgo.navigation.TripMap
import com.rahalgo.navigation.MarkerIcons
import androidx.compose.animation.AnimatedVisibility
import com.rahalgo.ui.Countdown
import androidx.compose.foundation.layout.IntrinsicSize
import com.rahalgo.design.Rahal
import com.rahalgo.ui.etaText
import com.rahalgo.ui.minutesShort
import com.rahalgo.ui.dist
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.material3.Surface
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.gestures.detectVerticalDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.layout.width
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalContext
import com.rahalgo.driver.BuildConfig
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.ui.money
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.FailReasonItem
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.Tone
import com.rahalgo.ui.RahalTextButton
import org.maplibre.android.geometry.LatLng

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نموذجُ الرحلة** — حالُها وأفعالُها وجداولُها. **ولا رسمَ فيه.**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **أُخرج من `TripScreen.kt`** (٢٠٢٦-٠٨-٢٣، فحصُ التطبيق):
 * **ألفان ومئتا سطرٍ في ملفٍّ واحدٍ يصعب تعديلُه بلا كسر.**
 */

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خطوات الرحلة السبع — وستّة أحوال في المحرّك**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والفرق مقصود**: «قبلت الطلب» و«في الطريق إلى المتجر» **حالٌ واحد في
 * المحرّك** (`assigned`) — لكنّهما لحظتان مختلفتان عند السائق: **قَبِل،
 * ثمّ مشى.**
 *
 * # وخطأٌ وقع هنا
 *
 * **كُتب الشريطُ يقرأ خطوتَه من ترتيب القائمة والزرُّ يقرأ التالية منها**
 * — فظهر الشريطُ يقول «قبلت الطلب» والزرُّ يقول «استلمت البضاعة»:
 * **قفزةُ خطوةٍ كاملة.** (قيس على الجهاز ٢٠٢٦-٠٨-١٢.)
 *
 * **فصار كلٌّ منهما يُشتقّ من حال الطلب في المحرّك وحدَه** — لا من
 * الآخر.
 */
enum class TripStep(val label: Int) {
    ACCEPTED(R.string.trip_s_accepted),
    TO_PICKUP(R.string.trip_s_to_pickup),
    AT_PICKUP(R.string.trip_s_at_pickup),
    PICKED_UP(R.string.trip_s_picked_up),
    TO_CUSTOMER(R.string.trip_s_to_customer),
    AT_CUSTOMER(R.string.trip_s_at_customer),
    DELIVERED(R.string.trip_s_delivered),
    ;

    companion object {
        /**
         * **يقرأ الخطوة من حال الطلب.**
         *
         * **و`assigned` تُقرأ «في الطريق إلى المتجر»** لا «قبلت»: **القبول
         * لحظةٌ مضت**، وما يفعله الآن هو المشي.
         */
        fun of(status: String): TripStep = when (status) {
            "assigned" -> TO_PICKUP
            "at_pickup" -> AT_PICKUP
            "picked_up" -> PICKED_UP
            "on_the_way" -> TO_CUSTOMER
            "at_dropoff" -> AT_CUSTOMER
            "delivered" -> DELIVERED
            else -> ACCEPTED
        }
    }
}

/** الفعل التالي — **باسم الحال في المحرّك ونصِّ الزرّ.** */
data class NextAction(val status: String, val label: Int)

/**
 * **ما يفعله الآن** — يُشتقّ من حال الطلب لا من موضعه في الشريط.
 *
 * **والمحرّك يرفض أيّ قفزة** (`orders/statuses.go`) — فلو تأخّرت الشاشة
 * عن حاله الحقيقيّ ردّ خطأً ولم يقع شيء.
 */
fun nextAction(status: String, custom: Boolean = false): NextAction? = when (status) {
    // ══════════════════════════════════════════════════════════════════
    // **ولا «وصلتُ إلى المتجر» في الطلب الخاصّ**
    // ══════════════════════════════════════════════════════════════════
    //
    // (تصحيحُ المالك ٢٠٢٦-٠٨-٠٩: «ما في شيء اسمه وصلتُ للمتجر» —
    //  وقرارُه ٢٠٢٦-٠٨-١٣: «نغلق الطلبَ الخاصَّ أوّلاً».)
    //
    // **لا متجرَ يقف عنده** — والمرحلةُ بعد الإسناد محادثةٌ واتّفاقٌ ثمّ
    // شراء. **وخارطةُ المحرّك تقولها صراحة**: الخاصُّ يمضي من «أُسند»
    // إلى «اشتريتُ» رأساً (`orders/statuses.go: customTransitions`).
    //
    // **وكان الزرُّ يرسل `at_pickup`** — وهو انتقالٌ لا تعرفه خارطةُ
    // الخاصّ. **فيضغطه السائقُ فيُردّ ولا يفهم لماذا**: الشاشةُ تعرض
    // خطوةً لا وجودَ لها في المحرّك.
    "assigned" ->
        if (custom) {
            NextAction("picked_up", R.string.step_bought)
        } else {
            NextAction("at_pickup", R.string.step_at_pickup)
        }
    "at_pickup" -> NextAction("picked_up", R.string.step_picked_up)
    "picked_up" -> NextAction("on_the_way", R.string.step_on_the_way)
    "on_the_way" -> NextAction("at_dropoff", R.string.step_at_dropoff)
    "at_dropoff" -> NextAction("delivered", R.string.step_delivered)
    else -> null
}

/** ما تعرضه الرحلة — **ولا تملكه هي.** */
data class TripState(
    val order: DriverOrder? = null,
    /** ما بقي من الطريق بالمتر — **وسالبٌ يعني لا يُعرف.** */
    val remainingM: Double = -1.0,
    // ══════════════════════════════════════════════════════════════════
    // **والمسارُ الحقيقيُّ يسبق المستقيم**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢.)
    //
    // **وفارغٌ يعني أنّ محرّك المسارات لم يردّ** — فيُرسم الخطُّ المستقيمُ
    // وتُقرأ المسافةُ الهوائيّة، **ولا تبقى الخريطةُ بلا خطّ.**
    /** نقاطُ خطّ الشوارع — **وفارغةٌ تعني المستقيم.** */
    val routeLine: List<LatLng> = emptyList(),
    /** طولُ الطريق بالشوارع — **وسالبٌ يعني لا يُعرف.** */
    val routeM: Double = -1.0,
    /** مدّتُه بالثواني كما يحسبها المحرّك بسرعات الشوارع. */
    val routeSec: Double = -1.0,
    // ══════════════════════════════════════════════════════════════════
    // **ومسارُ الملاحة — المرحلة ٣أ**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وحدةٌ واحدةٌ لا حقولٌ متفرّقة**: هندسةٌ وتراكميّةٌ ومناوراتٌ
    // معاً، **فلا لحظةَ تكون فيها جديدةٌ مع قديمة.**
    //
    // **وفارغٌ يعني «لا إرشاد»** — والخريطةُ ترسم `routeLine` كما
    // كانت. (انظر `NavRouteMapper`.)
    val navRoute: com.rahalgo.navigation.NavRoute? = null,
    /** سرعة السائق الوسطى من المحرّك — **وصفر يعني لا تُحسب مدّة.** */
    val avgSpeedKmh: Long = 0,
    /** **هل هو على بُعد خطوات من وجهته؟** — يُقترح ولا يُنفَّذ. */
    val nearDestination: Boolean = false,
    /** أتلزم صورة تسليم؟ — **يقرّره المحرّك** (`drivers.require_delivery_photo`). */
    val requirePhoto: Boolean = false,
    /** أنافذة الاتّفاق مفتوحة؟ — **للطلب الخاصّ وحدَه.** */
    val agreeOpen: Boolean = false,
    /** محطّاته كلّها — **وواحدةٌ منها هي المعروضة.** */
    val stops: List<Stop> = emptyList(),
    /** **أنافذة الطارئ مفتوحة؟** — تُفتح حين لا سببَ يُختار. */
    val emergencyOpen: Boolean = false,
    /** عرضٌ نزل وهو في رحلة — **وفارغ يعني لا عرض.** */
    val onRouteOffer: DriverOrder? = null,
    val failReasons: List<FailReasonItem>? = null,
    val step: TripStep = TripStep.ACCEPTED,
    val driver: LatLng? = null,
    val pickup: LatLng? = null,
    val dropoff: LatLng? = null,
    val busy: Boolean = false,
    val error: String = "",

    // ══════════════════════════════════════════════════════════════════
    // **خياراتُ المسار — إغلاقُ واجهة ٧، ٢٠٢٦-٠٨-٢١**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وفارغةٌ تعني: لا لوحةَ تُعرض** (البند ٥) — ولا مساحةَ تُؤخذ من
    // شاشةِ سائقٍ يقود.
    val routeChoices: com.rahalgo.navigation.RouteChoices? = null,
    /** **المعايَنُ** — بصريٌّ محضٌ لا ملاحة (البند ١٣). */
    val choicePreview: String? = null,
    /** **وقد بطل** — إشعارٌ قصيرٌ ثمّ يزول (البند ١٩). */
    val choiceStale: Boolean = false,
    /** **مسارٌ اعتُمد وينتظر التسليم** — البند ١٧. */
    val committedRoute: com.rahalgo.navigation.NavRoute? = null,
    /**
     * **هويّةُ المسار النافذ** — المرحلة ٨ب.
     *
     * **معرّفٌ محايدٌ من الخادم** — لا يُخترع في الجوّال.
     */
    val navRouteId: String? = null,

    /** **سببُ آخرِ تركيب** — واختيارُ السائق لا يُطلق جلباً (البند ٦). */
    val installReason: com.rahalgo.navigation.RouteInstallReason =
        com.rahalgo.navigation.RouteInstallReason.INITIAL,
)

data class TripActions(
    val step: (String) -> Unit,
    /** يفتح الكاميرا لصورة التسليم. */
    val capture: () -> Unit,
    val release: () -> Unit,
    val chat: () -> Unit,
    val askAgree: () -> Unit,
    val agree: (Long, Long) -> Unit,
    val dismissAgree: () -> Unit,
    val pickStop: (String) -> Unit,
    val takeOffer: (String) -> Unit,
    val dismissOffer: () -> Unit,
    val askFail: () -> Unit,
    val fail: (String) -> Unit,
    val dismissFail: () -> Unit,
    /** **مشكلةٌ عند السائق نفسِه** — يُكتب سببُها ويُعاد الطلبُ أو
     *  تُنبَّه العمليات، بحسب موضعه من الرحلة. */
    val problem: (String) -> Unit,
    /** **بلاغُ الطارئ** — العملياتُ تُنبَّه وموضعُه يُقرأ. */
    val emergency: () -> Unit,
    val dismissEmergency: () -> Unit,
    val navigate: () -> Unit,
    /** **يقلب الصوت** — (طلبُ المالك ٢٠٢٦-٠٨-٢٤). */
    val toggleVoice: () -> Unit = {},
    val toOrders: () -> Unit,

    // ══════════════════════════════════════════════════════════════════
    // **اختيارُ المسار — إغلاقُ واجهة ٧**
    // ══════════════════════════════════════════════════════════════════
    //
    // **ولا اختيارَ بلمسةٍ واحدة** (البند ١١): `previewRoute` تُعاين،
    // **و`confirmRoute` وحدَها تعتمد.**

    /** **يطلب البدائل** — عند حدثٍ لا في حلقة (البند ٤). */
    val loadAlternatives: (
        Long,
        com.rahalgo.navigation.RouteTarget,
        Double,
        Double,
        com.rahalgo.navigation.RouteInstallReason,
        Long,
        List<com.rahalgo.navigation.GeoPoint>,
    ) -> Unit = { _, _, _, _, _, _, _ -> },

    /** **يُعاين** — بصريٌّ محضٌ (البند ١٣)، و`null` يُلغي. */
    val previewRoute: (String?) -> Unit = {},

    /** **يعتمد** — بعد فحصٍ، ويردّ هل قُبل (البندان ١٧ و١٨). */
    val confirmRoute: (Long, com.rahalgo.navigation.RouteTarget, Double, Double, Double, Boolean) -> Boolean =
        { _, _, _, _, _, _ -> false },

    /** **بعد أن تسلّم الشاشةُ المسارَ المعتمد.** */
    val onRouteCommitted: () -> Unit = {},

    /** **مسحٌ صريح** — تبدّلُ وجهةٍ أو نجاحُ إعادةِ حساب (٢٢ و٢٣). */
    val clearChoices: () -> Unit = {},

    /** **يسأل الخادمَ عن ارتباط الأثر بالمسار** — المرحلة ٨ب. */
    val askCorrelation: (
        String,
        List<com.rahalgo.shared.model.CorrelationFix>,
        (com.rahalgo.navigation.RoadCorrelation) -> Unit,
    ) -> Unit = { _, _, _ -> },

    /** **تُخبر نموذجَ العرض بما رُكّب ولماذا** — البند ٧. */
    val onRouteInstalled: (Long, com.rahalgo.navigation.RouteTarget, com.rahalgo.navigation.RouteInstallReason) -> Unit =
        { _, _, _ -> },
)


/**
 * ══════════════════════════════════════════════════════════════════════
 * **إلى أيّ حالٍ يُنقل الوصولُ التلقائيّ — وقرارٌ يُختبر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٣: «المقصودُ تقليلُ الأزرار والضغط عليها»؛
 *  ثمّ فحصُ التطبيق كشف أنّ هذا القرارَ كان داخلَ الشاشة لا يُختبر.)
 *
 * # ولماذا خرج من الشاشة
 *
 * **هذا القرارُ يفجّر انتقالاً لا رجعةَ فيه**: جدولُ المحرّك
 * (`orders/statuses.go`) لا يعرف `at_pickup → assigned` ولا
 * `at_dropoff → on_the_way`. **فخطأٌ هنا يُسجَّل في دفتر الطلب ولا
 * يُردّ إلّا بيد المكتب.**
 *
 * **ومنطقٌ بهذا الأثر داخلَ `@Composable` لا يُختبر** — يحتاج شاشةً
 * حيّةً وجهازاً. **فأُخرج دالّةً صافية.**
 *
 * # وثلاثُ قواعدَ لا واحدة
 *
 * **١ · لا وصولَ للمتجر في الطلب الخاصّ** — (تصحيحُ المالك
 * ٢٠٢٦-٠٨-٠٩: «ما في شيء اسمه وصلتُ للمتجر»)، **والمحرّكُ يرفضه
 * أصلاً**: جدولُه للطلب الخاصّ يقفز من `assigned` إلى `picked_up`.
 * **فنداءٌ هنا يُردّ بخطأٍ يراه السائقُ ولا يفهمه.**
 *
 * **٢ · ولا وصولَ للزبون في الخاصّ كذلك** — والقاعدةُ واحدةٌ لا
 * قاعدتان: **الخاصُّ خارجَ هذا الباب كلِّه.**
 *
 * **٣ · والحالُ يُطابَق تماماً لا يُقارَب** — `assigned` وحدَها
 * و`on_the_way` وحدَها. **وما بينهما (`at_pickup` · `picked_up`)
 * حالٌ بلغها بيده، ولا يُعاد تسجيلُها.**
 */
fun autoArrivalTarget(status: String, custom: Boolean): String? {
    if (custom) return null
    return when (status) {
        "assigned" -> "at_pickup"
        "on_the_way" -> "at_dropoff"
        else -> null
    }
}
