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
 * **لوحُ الرحلة** — إلى أين، وكم بقي، وأين صار من مراحلها.
 * ══════════════════════════════════════════════════════════════════════
 *
 * **أُخرج من `TripScreen.kt`** (٢٠٢٦-٠٨-٢٣، فحصُ التطبيق):
 * **ألفان ومئتا سطرٍ في ملفٍّ واحدٍ يصعب تعديلُه بلا كسر.**
 */

/**
 * **لوحُ الرحلة** — إلى أين، وكم بقي، وأين صار من مراحلها.
 *
 * # ولماذا لوحٌ واحد
 *
 * **لوحان فوق خريطةٍ يقضمان ثلثَها** — وهي ما يقود عليه. **وأحدُهما
 * يختفي عند الوقوف فيترك فراغاً معلّقا** لا يُقرأ شيئا.
 *
 * # والطورُ يغيب والمراحلُ تبقى
 *
 * **من وقف عند الباب لا يقرأ «الطريق إلى فلان» ولا مسافةً ولا زمنا** —
 * وهو واقفٌ فيه. **لكنّه يبقى يريد أن يعرف أين صار من الرحلة.**
 */
@Composable
internal fun TripPanel(state: TripState) {
    val order = state.order ?: return
    val moving = state.step == TripStep.TO_PICKUP || state.step == TripStep.TO_CUSTOMER ||
        state.step == TripStep.PICKED_UP

    // ══════════════════════════════════════════════════════════════════
    // **وشريطُ المراحل يُسحب إلى الأعلى فيختفي**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «شريطُ الحالات لازم يختفي من أمام
    //  السائق — يعني يُسحب للأعلى أيضاً مثل الكرت السفليّ».)
    //
    // **والخريطةُ هي الشاشة**: لوحٌ فوقها وبطاقةٌ تحتها يقضمان ثلثيها،
    // **ومن يقود يريد أن يرى الشارعَ الذي أمامه.**
    //
    // **والمراحلُ يُنظر إليها مرّةً في الطور** — لا كلَّ ثانية.
    // **والوجهةُ والمسافةُ تبقيان**: هما ما يُقرأ في نظرةٍ خاطفة.
    //
    // **والحركةُ حركةُ البطاقة السفليّة نفسُها** — سحبٌ على اللوح
    // كلِّه ولمسةٌ على المقبض: **إيماءتان مختلفتان في شاشةٍ واحدةٍ
    // تُنسيان إحداهما.**
    //
    // **وتعود مع كلّ طلبٍ جديد** (`rememberSaveable(order.id)`) — من
    // بدأ طوراً جديداً يريد أن يرى أين صار.
    var stripOpen by rememberSaveable(order.id) { mutableStateOf(true) }

    Column(
        Modifier
            .padding(horizontal = 12.dp, vertical = 8.dp)
            .clip(Rahal.shape.lg)
            .background(Rahal.colors.panel)
            .pointerInput(Unit) {
                detectVerticalDragGestures { _, dy ->
                    if (dy < -6f) stripOpen = false
                    if (dy > 6f) stripOpen = true
                }
            }
            .padding(horizontal = 14.dp, vertical = 10.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        if (moving) {
            // ══════════════════════════════════════════════════════════
            // **والوجهةُ باسمها لا بصفتها**
            // ══════════════════════════════════════════════════════════
            //
            // (وهو ما قرّره المالكُ في بطاقة الطلب ٢٠٢٦-٠٨-١٢: «ما في
            //  داعي لكلمة المتجر» — واللوحُ أولى به.)
            //
            // **و«الطريق إلى الزبون» لا تقول لمن يحمل ثلاثة طلبات
            // أيَّها هذا** — و«ابوطيف» تقول.
            val name = if (state.step >= TripStep.PICKED_UP) {
                order.customerName.ifBlank { stringResource(R.string.detail_customer) }
            } else {
                order.merchantName.ifBlank { stringResource(R.string.nav_custom_order) }
            }
            // ══════════════════════════════════════════════════════════
            // **والخاصُّ قبل الشراء لا وجهةَ له**
            // ══════════════════════════════════════════════════════════
            //
            // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نغلق الطلبَ الخاصَّ أوّلاً».)
            //
            // **كان يُكتب «الطريق إلى طلب خاصّ»** — واسمُ المتجر فيه
            // هو الكلمةُ «طلب خاصّ» نفسُها. **فتُقرأ الجملةُ طريقاً إلى
            // مكانٍ اسمُه «طلب خاصّ».**
            //
            // **وما يفعله في هذا الطور محادثةٌ واتّفاقٌ ثمّ شراء** —
            // لا سيرٌ إلى موضع. **فيُقال له ذلك.**
            val custom = order.kind == "custom"
            Text(
                text = if (custom && state.step < TripStep.PICKED_UP) {
                    stringResource(R.string.trip_p_custom)
                } else {
                    stringResource(phaseLabel(state.step), name)
                },
                color = Color.White,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleMedium,
            )

            // **ورقمُ المحرّك يسبق الهوائيّ** — «٩٨٢ م ودقيقتان» كانت
            // تعني في الواقع «١٫١ كم وسبعَ دقائق». (قيس ٢٠٢٦-٠٨-١٢.)
            val meters = if (state.routeM >= 0) state.routeM else state.remainingM
            if (meters >= 0) {
                Spacer(Modifier.height(4.dp))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    // **والمدّة من المحرّك إن وُجدت** — محسوبةً بسرعات
                    // الشوارع نفسِها لا بسرعةٍ واحدةٍ في الإعدادات.
                    val minutes = if (state.routeSec >= 0) {
                        minutesShort((state.routeSec / 60).toLong().coerceAtLeast(1))
                    } else {
                        eta(meters, state.avgSpeedKmh).substringAfter("~")
                    }
                    if (minutes.isNotEmpty()) {
                        PanelChip(R.drawable.ic_time, minutes)
                        Spacer(Modifier.size(14.dp))
                    }
                    PanelChip(R.drawable.ic_pin, distanceText(meters))
                }
            }
            if (stripOpen) {
                Spacer(Modifier.height(10.dp))
                HorizontalDivider(color = Color.White.copy(alpha = 0.12f))
            }
        }

        androidx.compose.animation.AnimatedVisibility(visible = stripOpen) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Spacer(Modifier.height(10.dp))
                LegStrip(
                    status = order.status,
                    custom = order.kind == "custom",
                    // **وعلامةُ الطور أنّ الثمنَ وُثّق** — لا حالٌ ثانيةٌ
                    // في المحرّك: يبقى `assigned` قبل التوثيق وبعده.
                    agreed = order.customFee != null,
                )
            }
        }

        // **ومقبضٌ يُرى** — شريطٌ فاتحٌ يقول «هذا يُسحب»، **ولوحٌ يُسحب
        // ولا يقول** لا يعرف أحدٌ أنّه يُسحب. **ومن أخفاه لا يجد ما
        // يعيده به.**
        Spacer(Modifier.height(8.dp))
        Box(
            Modifier
                .clip(Rahal.shape.pill)
                .background(Color.White.copy(alpha = 0.30f))
                .size(width = 44.dp, height = 5.dp)
                .clickable { stripOpen = !stripOpen },
        )
    }
}

/** **أيقونةٌ ورقم** — على الأرض الداكنة. */
@Composable
internal fun PanelChip(icon: Int, text: String) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(
            painter = painterResource(icon),
            contentDescription = null,
            tint = Rahal.colors.accent,
            modifier = Modifier.size(15.dp),
        )
        Spacer(Modifier.size(5.dp))
        Text(
            text = text,
            color = Color.White,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}

/**
 * **شريطُ المراحل الأربع** — دوائرُ موصولةٌ بخطّ.
 *
 * **والخطُّ بينها ليس زينة**: هو ما يقول إنّها **رحلةٌ واحدةٌ تمشي**، لا
 * أربعُ حالاتٍ متجاورة. **وما مضى منه ملوّنٌ وما بقي باهت** — فيُقرأ
 * التقدّمُ بالعين قبل أن تُقرأ الكلمات.
 *
 * **والحاليّةُ وحدَها كبيرةٌ برتقاليّة**: من لوّن الكلَّ جعل صاحبَه
 * يبحث عن موضعه بين أربعةٍ متشابهة.
 */
@Composable
internal fun LegStrip(status: String, custom: Boolean = false, agreed: Boolean = false) {
    // ══════════════════════════════════════════════════════════════════
    // **ومراحلُ الخاصّ غيرُ مراحل العاديّ**
    // ══════════════════════════════════════════════════════════════════
    //
    // (شكوى المالك ٢٠٢٦-٠٨-١٣ بلقطةٍ من جهازه: «برأيي الخطواتُ هنا مو
    //  مزبوطة — بالطلب الخاصّ».)
    //
    // **كان يُعرض شريطُ العاديّ**: أوّلُ مرحلةٍ فيه «استلمت الطلب» —
    // **فيقرأ صاحبُ طلبٍ خاصٍّ أنّه في طور الاستلام** وهو لم يتّفق
    // على شيءٍ بعد. **والزرُّ تحته يقول «اشتريتُ الطلب».**
    //
    // **والمحرّكُ يعرف مراحلَ الخاصّ وحدَها** (`OpsCustomStages`):
    // توثيقٌ ثمّ شراءٌ ثمّ طريقٌ ثمّ وصولٌ ثمّ تسليم. **فتُقرأ منه لا
    // تُخترع هنا.**
    val legs = if (custom) CUSTOM_LEGS else LEGS
    val icons = if (custom) CUSTOM_LEG_ICONS else LEG_ICONS
    val at = if (custom) customLegOf(status, agreed) else legOf(status)
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.Top) {
        for (i in legs.indices) {
            if (i > 0) {
                // **والخطُّ في مستوى الدوائر** — لا تحتها ولا فوقها.
                Box(
                    Modifier
                        .weight(1f)
                        .padding(top = 15.dp)
                        .height(2.dp)
                        .background(
                            if (i <= at) Rahal.colors.brand else Color.White.copy(alpha = 0.18f),
                        ),
                )
            }
            LegDot(legs[i], icons[i], i, at)
        }
    }
}

@Composable
internal fun LegDot(label: Int, icon: Int, index: Int, at: Int) {
    val done = index < at
    val here = index == at
    val ground = when {
        here -> Rahal.colors.accent
        done -> Rahal.colors.brand
        else -> Color.White.copy(alpha = 0.12f)
    }
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        modifier = Modifier.width(76.dp),
    ) {
        Box(
            Modifier.size(32.dp).clip(CircleShape).background(ground),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(if (done) R.drawable.ic_check_circle else icon),
                contentDescription = null,
                tint = if (here || done) Color.White else Color.White.copy(alpha = 0.45f),
                modifier = Modifier.size(17.dp),
            )
        }
        Spacer(Modifier.height(4.dp))
        Text(
            text = stringResource(label),
            color = when {
                here -> Color.White
                done -> Color.White.copy(alpha = 0.7f)
                else -> Color.White.copy(alpha = 0.4f)
            },
            fontWeight = if (here) FontWeight.Bold else FontWeight.Normal,
            style = MaterialTheme.typography.labelSmall,
            textAlign = TextAlign.Center,
            maxLines = 2,
        )
    }
}

private val LEGS = listOf(
    R.string.leg_picked,
    R.string.ord_st_onway,
    R.string.leg_arrived,
    R.string.ord_st_delivered,
)

private val LEG_ICONS = listOf(
    R.drawable.ic_store,
    R.drawable.ic_moto,
    R.drawable.ic_pin,
    R.drawable.ic_check_circle,
)

/**
 * **مراحلُ الطلب الخاصّ** — كما يعرفها المحرّك (`OpsCustomStages`).
 *
 * **وخمسٌ لا أربع**: بين الإسناد والطريق طوران لا طور — **يتّفق ثمّ
 * يشتري**، وكلاهما فعلٌ يقع في وقتٍ ويُسأل عنه في الشكوى.
 */
private val CUSTOM_LEGS = listOf(
    R.string.leg_agree,
    R.string.leg_buy,
    R.string.ord_st_onway,
    R.string.leg_arrived,
    R.string.ord_st_delivered,
)

private val CUSTOM_LEG_ICONS = listOf(
    R.drawable.ic_chat,
    R.drawable.ic_cash,
    R.drawable.ic_moto,
    R.drawable.ic_pin,
    R.drawable.ic_check_circle,
)

/**
 * **أيُّ مرحلةٍ من مراحل الخاصّ هو فيها.**
 *
 * **والحالُ لا يفرّق بين التوثيق والشراء** — يبقى `assigned` فيهما،
 * **والفارقُ أنّ الثمنَ وُثّق.** (`custom_fee` غيرُ فارغ.)
 */
internal fun customLegOf(status: String, agreed: Boolean): Int = when (status) {
    "picked_up", "on_the_way" -> 2
    "at_dropoff" -> 3
    "delivered" -> 4
    else -> if (agreed) 1 else 0
}

/**
 * **أيُّ مرحلةٍ هو فيها الآن.**
 *
 * **وما قبل الاستلام كلُّه المرحلةُ الأولى**: `assigned` و`at_pickup`
 * طريقُه إلى المتجر — **وهي عندنا مرحلةٌ واحدة** لأنّ الزبون لا يفرّق
 * بينهما، **والسائقُ يقرأ ما عليه في الزرّ لا في الشريط.**
 */
internal fun legOf(status: String): Int = when (status) {
    "picked_up", "on_the_way" -> 1
    "at_dropoff" -> 2
    "delivered" -> 3
    else -> 0
}

/**
 * **اسمُ الطور** — والرحلةُ طوران لا سبعة.
 *
 * **وما قبل الاستلام كلُّه «إلى المتجر»**، وما بعده كلُّه «إلى الزبون»
 * — **والوقوفُ عند أحدهما طورٌ ثالثٌ قصير** يُقال لأنّ الفعل التالي
 * يختلف.
 */
internal fun phaseLabel(step: TripStep): Int = when (step) {
    TripStep.AT_PICKUP -> R.string.trip_p_at_store
    TripStep.PICKED_UP, TripStep.TO_CUSTOMER -> R.string.trip_p_to_customer
    TripStep.AT_CUSTOMER -> R.string.trip_p_at_customer
    TripStep.DELIVERED -> R.string.trip_p_delivered
    else -> R.string.trip_p_to_store
}

/** المسافة بالمتر أو بالكيلومتر — **لا «1400 م».** */
internal fun distanceText(meters: Double): String = dist(meters)

/**
 * **الوقت المتوقّع** — من المسافة وسرعة السائق.
 *
 * **وفارغ إن لم تُضبط السرعة**: **رقمٌ مبنيٌّ على صفر يقول «الآن»**،
 * ووعدٌ كاذبٌ للزبون أسوأ من لا وعد.
 *
 * **ودقيقة على الأقلّ** — «٠ دقيقة» لا تُقال لمن لم يصل بعد.
 */
internal fun eta(meters: Double, avgSpeedKmh: Long): String =
    etaText(meters, avgSpeedKmh)
